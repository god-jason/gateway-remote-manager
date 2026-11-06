package grpv1

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

// StreamAction Stream 帧动作
type StreamAction uint8

const (
	StreamUnspecified StreamAction = iota
	StreamOpen
	StreamOpenAck
	StreamData
	StreamWindowUpdate
	StreamHalfClose
	StreamClose
	StreamReset
)

// 消息类型：所有流帧统一使用 type="stream"，由 StreamFrame.stream_id 路由
const TypeStream = "stream"

const (
	streamControlID  uint32 = 0 // 保留给控制通道
	defaultMaxWindow uint32 = 256 * 1024
	openTimeout             = 10 * time.Second
)

// StreamFrame 逻辑流帧
type StreamFrame struct {
	StreamID    uint32       `json:"stream_id"`
	Action      StreamAction `json:"action"`
	Offset      uint64       `json:"offset,omitempty"`
	Data        []byte       `json:"data,omitempty"`
	WindowSize  uint32       `json:"window_size,omitempty"`
	ContentType string       `json:"content_type,omitempty"`
}

// StreamOptions 打开流的选项
type StreamOptions struct {
	ContentType string
	MaxWindow   uint32
}

// Stream 逻辑流接口（多路复用的数据通道）
type Stream interface {
	ID() uint32
	Read(ctx context.Context, data []byte) (int, error)
	Write(ctx context.Context, data []byte) (int, error)
	Close() error
	Reset(err error) error
}

// Mux 连接级多路复用器：在单条长连接上承载多个独立 Stream。
// 提供基于窗口的流量控制（WINDOW_UPDATE）、半关闭与 Reset。
type Mux struct {
	deviceID string
	send     func(ctx context.Context, env *Envelope) error
	maxWin   uint32

	mu      sync.Mutex
	streams map[uint32]*muxStream
	nextID  uint32
}

// NewMux 创建多路复用器
func NewMux(deviceID string, maxWindow uint32, send func(ctx context.Context, env *Envelope) error) *Mux {
	if maxWindow == 0 {
		maxWindow = defaultMaxWindow
	}
	return &Mux{
		deviceID: deviceID,
		send:     send,
		maxWin:   maxWindow,
		streams:  make(map[uint32]*muxStream),
		nextID:   1,
	}
}

func (m *Mux) alloc() uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	for {
		id := m.nextID
		m.nextID++
		if id != streamControlID {
			if _, ok := m.streams[id]; !ok {
				return id
			}
		}
	}
}

func (m *Mux) get(id uint32) *muxStream {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.streams[id]
}

// Get 按 ID 查询流
func (m *Mux) Get(id uint32) *muxStream { return m.get(id) }

// SetDeviceID 设置设备 ID（建立连接后、发送流帧前调用）
func (m *Mux) SetDeviceID(id string) {
	m.mu.Lock()
	m.deviceID = id
	m.mu.Unlock()
}

func (m *Mux) remove(id uint32) {
	m.mu.Lock()
	delete(m.streams, id)
	m.mu.Unlock()
}

func (m *Mux) newStream(id uint32, opts StreamOptions) *muxStream {
	s := &muxStream{
		mux: m,
		id:  id,
		ct:  opts.ContentType,
	}
	s.wCond = sync.NewCond(&s.wMu)
	s.rCond = sync.NewCond(&s.rMu)
	s.opened = make(chan struct{})
	return s
}

// sendFrame 将流帧封装为 STREAM 消息发送
func (m *Mux) sendFrame(ctx context.Context, frame *StreamFrame) error {
	m.mu.Lock()
	deviceID := m.deviceID
	m.mu.Unlock()
	env := NewEnvelope(KindStream, TypeStream, deviceID)
	env.Payload, _ = MarshalPayload(frame)
	return m.send(ctx, env)
}

// OpenStream 主动发起一个流（发起方）
func (m *Mux) OpenStream(ctx context.Context, opts StreamOptions) (*muxStream, error) {
	id := m.alloc()
	s := m.newStream(id, opts)
	m.mu.Lock()
	m.streams[id] = s
	m.mu.Unlock()

	if err := m.sendFrame(ctx, &StreamFrame{StreamID: id, Action: StreamOpen, ContentType: opts.ContentType}); err != nil {
		m.remove(id)
		return nil, err
	}

	select {
	case <-s.opened:
		return s, nil
	case <-ctx.Done():
		m.remove(id)
		return nil, ctx.Err()
	case <-time.After(openTimeout):
		m.remove(id)
		return nil, errors.New("grp: stream open timeout")
	}
}

// HandleFrame 处理到达的流帧（由传输层分发 STREAM 消息时调用）
func (m *Mux) HandleFrame(ctx context.Context, frame *StreamFrame) {
	switch frame.Action {
	case StreamOpen:
		s := m.newStream(frame.StreamID, StreamOptions{ContentType: frame.ContentType, MaxWindow: m.maxWin})
		m.mu.Lock()
		m.streams[frame.StreamID] = s
		m.mu.Unlock()
		_ = m.sendFrame(ctx, &StreamFrame{StreamID: frame.StreamID, Action: StreamOpenAck, WindowSize: m.maxWin, ContentType: frame.ContentType})
	case StreamOpenAck:
		if s := m.get(frame.StreamID); s != nil {
			s.applyOpen(frame.WindowSize)
		}
	case StreamData:
		if s := m.get(frame.StreamID); s != nil {
			s.onData(frame.Data)
		}
	case StreamWindowUpdate:
		if s := m.get(frame.StreamID); s != nil {
			s.onWindowUpdate(frame.WindowSize)
		}
	case StreamHalfClose:
		if s := m.get(frame.StreamID); s != nil {
			s.onRemoteClose()
		}
	case StreamClose:
		if s := m.get(frame.StreamID); s != nil {
			s.onRemoteClose()
		}
		m.remove(frame.StreamID)
	case StreamReset:
		if s := m.get(frame.StreamID); s != nil {
			s.onReset(errors.New("grp: stream reset"))
		}
		m.remove(frame.StreamID)
	}
}

// muxStream 具体实现
type muxStream struct {
	mux *Mux
	id  uint32
	ct  string

	wMu         sync.Mutex
	wCond       *sync.Cond
	sendWindow  int64
	localClosed bool
	localErr    error

	rMu          sync.Mutex
	rCond        *sync.Cond
	buf          bytes.Buffer
	remoteClosed bool
	remoteErr    error

	opened     chan struct{}
	openedOnce sync.Once
}

// ID 流 ID
func (s *muxStream) ID() uint32 { return s.id }

// applyOpen 发起方收到 OPEN_ACK：获得发送额度，并回授对端发送额度
func (s *muxStream) applyOpen(window uint32) {
	s.wMu.Lock()
	s.sendWindow = int64(window)
	s.wMu.Unlock()
	s.wCond.Broadcast()
	_ = s.mux.sendFrame(context.Background(), &StreamFrame{StreamID: s.id, Action: StreamWindowUpdate, WindowSize: s.mux.maxWin})
	s.openedOnce.Do(func() { close(s.opened) })
}

// Write 发送数据（受对端窗口约束，实现背压）
func (s *muxStream) Write(ctx context.Context, data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		s.wMu.Lock()
		for s.sendWindow <= 0 {
			if s.localErr != nil {
				err := s.localErr
				s.wMu.Unlock()
				return total, err
			}
			if s.localClosed {
				s.wMu.Unlock()
				return total, io.ErrClosedPipe
			}
			if ctx != nil && ctx.Err() != nil {
				s.wMu.Unlock()
				return total, ctx.Err()
			}
			s.wCond.Wait()
		}
		n := int64(len(data))
		if n > s.sendWindow {
			n = s.sendWindow
		}
		s.sendWindow -= n
		chunk := data[:int(n)]
		s.wMu.Unlock()

		if err := s.mux.sendFrame(ctx, &StreamFrame{StreamID: s.id, Action: StreamData, Data: chunk}); err != nil {
			_ = s.Reset(err)
			return total, err
		}
		total += int(n)
		data = data[int(n):]
	}
	return total, nil
}

// Read 读取数据，远端关闭后返回 io.EOF
func (s *muxStream) Read(ctx context.Context, data []byte) (int, error) {
	s.rMu.Lock()
	defer s.rMu.Unlock()
	for s.buf.Len() == 0 && !s.remoteClosed && s.remoteErr == nil {
		if ctx != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		s.rCond.Wait()
	}
	if s.remoteErr != nil {
		return 0, s.remoteErr
	}
	if s.buf.Len() == 0 && s.remoteClosed {
		return 0, io.EOF
	}
	n, _ := s.buf.Read(data)
	if n > 0 {
		_ = s.mux.sendFrame(context.Background(), &StreamFrame{StreamID: s.id, Action: StreamWindowUpdate, WindowSize: uint32(n)})
	}
	return n, nil
}

// Close 关闭流（双向关闭）
func (s *muxStream) Close() error {
	s.wMu.Lock()
	if s.localClosed {
		s.wMu.Unlock()
		return nil
	}
	s.localClosed = true
	s.wCond.Broadcast()
	s.wMu.Unlock()

	_ = s.mux.sendFrame(context.Background(), &StreamFrame{StreamID: s.id, Action: StreamClose})
	return nil
}

// Reset 异常重置流
func (s *muxStream) Reset(err error) error {
	s.wMu.Lock()
	if s.localErr == nil {
		s.localErr = err
	}
	s.localClosed = true
	s.wCond.Broadcast()
	s.wMu.Unlock()

	_ = s.mux.sendFrame(context.Background(), &StreamFrame{StreamID: s.id, Action: StreamReset})
	return nil
}

// ===== 由 Mux 回调（读循环 goroutine 中执行） =====

func (s *muxStream) onData(data []byte) {
	s.rMu.Lock()
	s.buf.Write(data)
	s.rCond.Broadcast()
	s.rMu.Unlock()
}

func (s *muxStream) onWindowUpdate(window uint32) {
	s.wMu.Lock()
	s.sendWindow += int64(window)
	s.wCond.Broadcast()
	s.wMu.Unlock()
}

func (s *muxStream) onRemoteClose() {
	s.rMu.Lock()
	s.remoteClosed = true
	s.rCond.Broadcast()
	s.rMu.Unlock()
}

func (s *muxStream) onReset(err error) {
	s.rMu.Lock()
	s.remoteErr = err
	s.remoteClosed = true
	s.rCond.Broadcast()
	s.rMu.Unlock()

	s.wMu.Lock()
	if s.localErr == nil {
		s.localErr = err
	}
	s.localClosed = true
	s.wCond.Broadcast()
	s.wMu.Unlock()
}