// Package grpv1 实现 GRP (Gateway Remote Protocol) 的消息模型、编码与客户端 SDK。
// .proto 文件是协议的权威定义（见 proto/grp/v1），当前 Go 实现使用 JSON 编码
// （协议允许 JSON 调试编码），Codec 接口预留了 Protobuf 替换点。
package grpv1

import (
	"github.com/google/uuid"
)

// 协议版本
const ProtocolVersion uint32 = 1

// MessageKind 消息类型
type MessageKind uint8

const (
	KindUnspecified MessageKind = iota
	KindRequest
	KindResponse
	KindEvent
	KindStream
	KindControl
)

func (k MessageKind) String() string {
	switch k {
	case KindRequest:
		return "REQUEST"
	case KindResponse:
		return "RESPONSE"
	case KindEvent:
		return "EVENT"
	case KindStream:
		return "STREAM"
	case KindControl:
		return "CONTROL"
	default:
		return "UNSPECIFIED"
	}
}

// 消息名称常量（模块.资源.动作）
const (
	TypeError = "error"

	// 连接握手
	TypeHandshakeHello      = "handshake.hello"
	TypeHandshakeChallenge  = "handshake.challenge"
	TypeHandshakeAuth       = "handshake.auth"
	TypeHandshakeAuthResult = "handshake.auth_result"

	// 设备能力
	TypeDeviceCapabilities = "device.capabilities"

	// 心跳
	TypeHeartbeat    = "heartbeat"
	TypeHeartbeatAck = "heartbeat_ack"

	// 系统状态
	TypeSystemStatus = "system.status"
)

// 消息标志
const (
	FlagNone    uint32 = 0
	FlagNoReply uint32 = 1 << 0 // 无需响应
)

// Envelope 统一消息包装
type Envelope struct {
	Version   uint32      `json:"version"`
	Kind      MessageKind `json:"kind"`
	Type      string      `json:"type"`
	MessageID string      `json:"message_id,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	DeviceID  string      `json:"device_id,omitempty"`
	Sequence  uint64      `json:"sequence,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
	Flags     uint32      `json:"flags,omitempty"`
	Payload   []byte      `json:"-"`
	TraceID   string      `json:"trace_id,omitempty"`
}

// NewEnvelope 创建消息
func NewEnvelope(kind MessageKind, typ, deviceID string) *Envelope {
	return &Envelope{
		Version:   ProtocolVersion,
		Kind:      kind,
		Type:      typ,
		DeviceID:  deviceID,
		MessageID: uuid.NewString(),
		Timestamp: nowMillis(),
	}
}

// NewMessageID 生成消息唯一 ID
func NewMessageID() string { return uuid.NewString() }

func nowMillis() int64 {
	return currentTime().UnixMilli()
}