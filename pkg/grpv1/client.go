package grpv1

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"gateway-manager/pkg/logger"
)

// ConnectionState 连接状态
type ConnectionState string

const (
	StateDisconnected ConnectionState = "disconnected"
	StateConnecting   ConnectionState = "connecting"
	StateConnected    ConnectionState = "connected"
)

// Handler 请求处理器（REQUEST / EVENT 均分发，返回值作为 RESPONSE 回发）
type Handler func(ctx context.Context, env *Envelope) (*Envelope, error)

// EventHandler 事件处理器
type EventHandler func(env *Envelope)

// Client Agent 侧通信接口
type Client interface {
	Connect(ctx context.Context) error
	Close() error
	State() ConnectionState
	Send(ctx context.Context, env *Envelope) error
	Request(ctx context.Context, env *Envelope, timeout time.Duration) (*Envelope, error)
	OpenStream(ctx context.Context, opts StreamOptions) (Stream, error)
	GetStream(id uint32) Stream
	RegisterHandler(messageType string, handler Handler)
	OnEvent(handler EventHandler)
	OnDisconnect(handler func())
	DeviceID() string
	SetDeviceID(id string)
}

// WebsocketClient 基于 WebSocket 的 GRP 客户端实现
type WebsocketClient struct {
	url      string
	header   http.Header
	dialer   *websocket.Dialer
	codec    Codec
	deviceID string
	mux      *Mux

	mu    sync.Mutex
	conn  *websocket.Conn
	state ConnectionState

	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan *Envelope

	handlerMu sync.RWMutex
	handlers  map[string]Handler

	eventMu            sync.Mutex
	eventHandlers      []EventHandler
	disconnectHandlers []func()

	wg   sync.WaitGroup
	done chan struct{}
}

// NewWebsocketClient 创建客户端
func NewWebsocketClient(url string, header http.Header) *WebsocketClient {
	if header == nil {
		header = http.Header{}
	}
	c := &WebsocketClient{
		url:      url,
		header:   header,
		dialer:   websocket.DefaultDialer,
		codec:    JSONCodec{},
		state:    StateDisconnected,
		pending:  make(map[string]chan *Envelope),
		handlers: make(map[string]Handler),
	}
	c.mux = NewMux("", defaultMaxWindow, c.Send)
	return c
}

// Connect 建立连接并启动读循环
func (c *WebsocketClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	c.state = StateConnecting
	c.mu.Unlock()

	conn, _, err := c.dialer.DialContext(ctx, c.url, c.header)
	if err != nil {
		c.mu.Lock()
		c.state = StateDisconnected
		c.mu.Unlock()
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.state = StateConnected
	c.done = make(chan struct{})
	c.mu.Unlock()

	c.wg.Add(1)
	go c.readLoop()
	return nil
}

// Close 关闭连接并等待读循环退出
func (c *WebsocketClient) Close() error {
	c.mu.Lock()
	conn := c.conn
	c.state = StateDisconnected
	if c.done != nil {
		close(c.done)
		c.done = nil
	}
	c.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	c.wg.Wait()
	return nil
}

// State 当前状态
func (c *WebsocketClient) State() ConnectionState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// DeviceID 设备 ID
func (c *WebsocketClient) DeviceID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.deviceID
}

// SetDeviceID 设置设备 ID
func (c *WebsocketClient) SetDeviceID(id string) {
	c.mu.Lock()
	c.deviceID = id
	c.mu.Unlock()
	c.mux.SetDeviceID(id)
}

// Send 发送消息
func (c *WebsocketClient) Send(ctx context.Context, env *Envelope) error {
	data, err := c.codec.Encode(env)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("connection not established")
	}
	if env.DeviceID == "" {
		env.DeviceID = c.DeviceID()
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// Request 发送请求并等待响应
func (c *WebsocketClient) Request(ctx context.Context, env *Envelope, timeout time.Duration) (*Envelope, error) {
	if env.MessageID == "" {
		env.MessageID = NewMessageID()
	}
	env.Kind = KindRequest
	env.RequestID = env.MessageID

	ch := make(chan *Envelope, 1)
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

// OpenStream 主动发起流
func (c *WebsocketClient) OpenStream(ctx context.Context, opts StreamOptions) (Stream, error) {
	return c.mux.OpenStream(ctx, opts)
}

// GetStream 按 ID 获取流（未找到返回 nil 接口）
func (c *WebsocketClient) GetStream(id uint32) Stream {
	s := c.mux.Get(id)
	if s == nil {
		return nil
	}
	return s
}

// RegisterHandler 注册类型处理器
func (c *WebsocketClient) RegisterHandler(messageType string, handler Handler) {
	c.handlerMu.Lock()
	c.handlers[messageType] = handler
	c.handlerMu.Unlock()
}

// OnEvent 注册事件监听
func (c *WebsocketClient) OnEvent(handler EventHandler) {
	c.eventMu.Lock()
	c.eventHandlers = append(c.eventHandlers, handler)
	c.eventMu.Unlock()
}

// OnDisconnect 注册断线回调
func (c *WebsocketClient) OnDisconnect(handler func()) {
	c.eventMu.Lock()
	c.disconnectHandlers = append(c.disconnectHandlers, handler)
	c.eventMu.Unlock()
}

func (c *WebsocketClient) readLoop() {
	defer c.wg.Done()
	defer c.notifyDisconnect()

	for {
		_, data, err := c.safeRead()
		if err != nil {
			break
		}
		env, err := c.codec.Decode(data)
		if err != nil {
			logger.Warn("grp 解码失败", "error", err)
			continue
		}
		c.dispatch(env)
	}
}

func (c *WebsocketClient) safeRead() (messageType int, data []byte, err error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return 0, nil, fmt.Errorf("connection closed")
	}
	return conn.ReadMessage()
}

func (c *WebsocketClient) dispatch(env *Envelope) {
	switch env.Kind {
	case KindResponse:
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
	case KindStream:
		var frame StreamFrame
		if err := UnmarshalPayload(env.Payload, &frame); err == nil {
			c.mux.HandleFrame(context.Background(), &frame)
		}
	case KindRequest:
		h := c.lookupHandler(env.Type)
		if h == nil {
			return
		}
		resp, err := h(context.Background(), env)
		if resp != nil {
			resp.Kind = KindResponse
			resp.RequestID = env.MessageID
			if resp.MessageID == "" {
				resp.MessageID = NewMessageID()
			}
			resp.Type = env.Type
			_ = c.Send(context.Background(), resp)
		} else if err != nil {
			logger.Debug("grp handler error", "type", env.Type, "error", err)
		}
	default:
		c.fireEvent(env)
		if h := c.lookupHandler(env.Type); h != nil {
			_, _ = h(context.Background(), env)
		}
	}
}

func (c *WebsocketClient) lookupHandler(typ string) Handler {
	c.handlerMu.RLock()
	defer c.handlerMu.RUnlock()
	return c.handlers[typ]
}

func (c *WebsocketClient) fireEvent(env *Envelope) {
	c.eventMu.Lock()
	handlers := make([]EventHandler, len(c.eventHandlers))
	copy(handlers, c.eventHandlers)
	c.eventMu.Unlock()
	for _, h := range handlers {
		h(env)
	}
}

func (c *WebsocketClient) notifyDisconnect() {
	c.mu.Lock()
	c.conn = nil
	c.state = StateDisconnected
	c.mu.Unlock()

	c.eventMu.Lock()
	handlers := make([]func(), len(c.disconnectHandlers))
	copy(handlers, c.disconnectHandlers)
	c.eventMu.Unlock()
	for _, h := range handlers {
		h()
	}

	c.pendingMu.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendingMu.Unlock()
}