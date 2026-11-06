package grpv1

import (
	"context"
	"time"
)

// Connection 抽象一条设备长连接（服务端视角）。
type Connection interface {
	ID() string
	DeviceID() string
	Send(ctx context.Context, env *Envelope) error
	Request(ctx context.Context, env *Envelope, timeout time.Duration) (*Envelope, error)
	OpenStream(ctx context.Context, opts StreamOptions) (Stream, error)
	GetStream(id uint32) Stream
	// SubscribeEvent 订阅到达的事件，返回取消订阅函数
	SubscribeEvent(handler func(env *Envelope)) func()
	Close() error
}