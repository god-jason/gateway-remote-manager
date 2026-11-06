package grpv1

import "encoding/json"

// Codec 消息编解码器。生产环境应替换为 Protobuf Codec。
type Codec interface {
	Encode(env *Envelope) ([]byte, error)
	Decode(data []byte) (*Envelope, error)
}

// JSONCodec JSON 调试编码实现（协议允许开发阶段使用 JSON）
type JSONCodec struct{}

// envelopeWire 与 Envelope 对应的线格式，payload 以原始 JSON 内联
type envelopeWire struct {
	Version   uint32          `json:"version"`
	Kind      MessageKind     `json:"kind"`
	Type      string          `json:"type"`
	MessageID string          `json:"message_id,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	DeviceID  string          `json:"device_id,omitempty"`
	Sequence  uint64          `json:"sequence,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
	Flags     uint32          `json:"flags,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	TraceID   string          `json:"trace_id,omitempty"`
}

// Encode 序列化
func (JSONCodec) Encode(env *Envelope) ([]byte, error) {
	w := envelopeWire{
		Version:   env.Version,
		Kind:      env.Kind,
		Type:      env.Type,
		MessageID: env.MessageID,
		RequestID: env.RequestID,
		SessionID: env.SessionID,
		DeviceID:  env.DeviceID,
		Sequence:  env.Sequence,
		Timestamp: env.Timestamp,
		Flags:     env.Flags,
		TraceID:   env.TraceID,
	}
	if len(env.Payload) > 0 {
		w.Payload = json.RawMessage(env.Payload)
	}
	return json.Marshal(w)
}

// Decode 反序列化
func (JSONCodec) Decode(data []byte) (*Envelope, error) {
	var w envelopeWire
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	return &Envelope{
		Version:   w.Version,
		Kind:      w.Kind,
		Type:      w.Type,
		MessageID: w.MessageID,
		RequestID: w.RequestID,
		SessionID: w.SessionID,
		DeviceID:  w.DeviceID,
		Sequence:  w.Sequence,
		Timestamp: w.Timestamp,
		Flags:     w.Flags,
		Payload:   []byte(w.Payload),
		TraceID:   w.TraceID,
	}, nil
}

// MarshalPayload 将业务对象编码为 payload 字节
func MarshalPayload(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalPayload 解码 payload 字节到对象
func UnmarshalPayload(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// NewEventEnvelope 便捷构造 EVENT 消息
func NewEventEnvelope(typ, deviceID string, payload interface{}) (*Envelope, error) {
	env := NewEnvelope(KindEvent, typ, deviceID)
	if payload != nil {
		b, err := MarshalPayload(payload)
		if err != nil {
			return nil, err
		}
		env.Payload = b
	}
	return env, nil
}