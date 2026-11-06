// Package connection 处理 Agent 长连接：握手认证、心跳、状态上报与流多路复用分发。
package connection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gateway-manager/pkg/grpv1"
	"gateway-manager/server/internal/device"
)

// Conn 设备连接，实现 grpv1.Connection
type Conn struct {
	ws       *websocket.Conn
	id       string
	deviceID string
	codec    grpv1.Codec
	writeMu  sync.Mutex
	mux      *grpv1.Mux

	pendingMu sync.Mutex
	pending   map[string]chan *grpv1.Envelope

	eventMu   sync.Mutex
	eventNext int
	events    map[int]func(*grpv1.Envelope)

	onDisconnect func()
}

func newConn(ws *websocket.Conn, id string) *Conn {
	c := &Conn{
		ws:      ws,
		id:      id,
		codec:   grpv1.JSONCodec{},
		pending: make(map[string]chan *grpv1.Envelope),
		events:  make(map[int]func(*grpv1.Envelope)),
	}
	c.mux = grpv1.NewMux("", 0, c.Send)
	return c
}

// ID 连接 ID
func (c *Conn) ID() string { return c.id }

// DeviceID 设备 ID
func (c *Conn) DeviceID() string { return c.deviceID }

// SetDeviceID 设置设备 ID
func (c *Conn) SetDeviceID(id string) {
	c.deviceID = id
	c.mux.SetDeviceID(id)
}

// Send 发送消息
func (c *Conn) Send(ctx context.Context, env *grpv1.Envelope) error {
	data, err := c.codec.Encode(env)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if env.DeviceID == "" {
		env.DeviceID = c.deviceID
	}
	return c.ws.WriteMessage(websocket.TextMessage, data)
}

// Request 发送请求并等待响应
func (c *Conn) Request(ctx context.Context, env *grpv1.Envelope, timeout time.Duration) (*grpv1.Envelope, error) {
	if env.MessageID == "" {
		env.MessageID = grpv1.NewMessageID()
	}
	env.Kind = grpv1.KindRequest
	env.RequestID = env.MessageID

	ch := make(chan *grpv1.Envelope, 1)
	c.pendingMu.Lock()
	c.pending[env.RequestID] = ch
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, env.RequestID)
		c.pendingMu.Unlock()
	}()

	if err := c.Send(ctx, env); err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-ch:
		return resp, nil
	case <-timer.C:
		return nil, fmt.Errorf("request timeout: %s", env.Type)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// OpenStream 主动打开流
func (c *Conn) OpenStream(ctx context.Context, opts grpv1.StreamOptions) (grpv1.Stream, error) {
	return c.mux.OpenStream(ctx, opts)
}

// GetStream 按 ID 获取流（未找到返回 nil 接口）
func (c *Conn) GetStream(id uint32) grpv1.Stream {
	s := c.mux.Get(id)
	if s == nil {
		return nil
	}
	return s
}

// SubscribeEvent 订阅到达事件，返回取消订阅函数
func (c *Conn) SubscribeEvent(handler func(env *grpv1.Envelope)) func() {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	c.eventNext++
	id := c.eventNext
	c.events[id] = handler
	return func() {
		c.eventMu.Lock()
		delete(c.events, id)
		c.eventMu.Unlock()
	}
}

func (c *Conn) fireEvent(env *grpv1.Envelope) {
	c.eventMu.Lock()
	handlers := make([]func(*grpv1.Envelope), 0, len(c.events))
	for _, h := range c.events {
		handlers = append(handlers, h)
	}
	c.eventMu.Unlock()
	for _, h := range handlers {
		h(env)
	}
}

// Close 关闭连接
func (c *Conn) Close() error { return c.ws.Close() }

// readEnvelope 读取并解码一条消息
func (c *Conn) readEnvelope() (*grpv1.Envelope, error) {
	_, data, err := c.ws.ReadMessage()
	if err != nil {
		return nil, err
	}
	return c.codec.Decode(data)
}

// loop 读循环 + 分发
func (c *Conn) loop(ctx context.Context) {
	defer func() {
		if c.onDisconnect != nil {
			c.onDisconnect()
		}
	}()
	for {
		env, err := c.readEnvelope()
		if err != nil {
			break
		}
		c.dispatch(ctx, env)
	}
}

func (c *Conn) dispatch(ctx context.Context, env *grpv1.Envelope) {
	switch env.Kind {
	case grpv1.KindResponse:
		c.pendingMu.Lock()
		ch := c.pending[env.RequestID]
		delete(c.pending, env.RequestID)
		c.pendingMu.Unlock()
		if ch != nil {
			select {
			case ch <- env:
			default:
			}
		}
	case grpv1.KindStream:
		var frame grpv1.StreamFrame
		if err := grpv1.UnmarshalPayload(env.Payload, &frame); err == nil {
			c.mux.HandleFrame(ctx, &frame)
		}
	case grpv1.KindRequest:
		c.handleRequest(ctx, env)
	default:
		c.handleEvent(env)
	}
}

func (c *Conn) handleRequest(ctx context.Context, env *grpv1.Envelope) {
	switch env.Type {
	case grpv1.TypeHeartbeat:
		device.Default.Touch(c.deviceID)
		_ = device.TouchSeen(c.deviceID)
		ack := grpv1.HeartbeatAck{ServerTime: time.Now().UnixMilli(), NextHeartbeatInterval: 30}
		resp := grpv1.NewEnvelope(grpv1.KindResponse, grpv1.TypeHeartbeatAck, c.deviceID)
		resp.RequestID = env.MessageID
		resp.Payload, _ = grpv1.MarshalPayload(ack)
		_ = c.Send(ctx, resp)
	}
}

func (c *Conn) handleEvent(env *grpv1.Envelope) {
	switch env.Type {
	case grpv1.TypeSystemStatus:
		var st grpv1.SystemStatus
		if err := grpv1.UnmarshalPayload(env.Payload, &st); err == nil {
			device.Default.UpdateSystem(c.deviceID, &st)
		}
		device.Default.Touch(c.deviceID)
	case grpv1.TypeDeviceCapabilities:
		// 能力信息暂存内存，后续会话功能使用
	}
	c.fireEvent(env)
}