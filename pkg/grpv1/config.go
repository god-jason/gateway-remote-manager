package grpv1

// 配置管理消息类型（请求/响应，走控制通道，采用期望/实际状态模型）
const (
	TypeConfigList   = "config.list"
	TypeConfigGet    = "config.get"
	TypeConfigSet    = "config.set"
	TypeConfigDelete = "config.delete"
)

// ConfigObject 配置对象
type ConfigObject struct {
	ConfigKey  string `json:"config_key"`
	ConfigType string `json:"config_type,omitempty"`
	Content    string `json:"content,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
	Version    uint64 `json:"version,omitempty"`
	ModifiedAt int64  `json:"modified_at,omitempty"`
}

// ConfigKeyRequest 配置 key 请求
type ConfigKeyRequest struct {
	ConfigKey string `json:"config_key"`
}

// ConfigSetRequest 下发期望配置
type ConfigSetRequest struct {
	ConfigKey string `json:"config_key"`
	Content   string `json:"content"`
	Checksum  string `json:"checksum,omitempty"` // 校验，可选
}

// ConfigListResult 配置列表结果
type ConfigListResult struct {
	Success bool           `json:"success"`
	Message string         `json:"message,omitempty"`
	Configs []ConfigObject `json:"configs,omitempty"`
}

// ConfigResult 配置操作结果
type ConfigResult struct {
	Success bool          `json:"success"`
	Message string        `json:"message,omitempty"`
	Config  *ConfigObject `json:"config,omitempty"`
}