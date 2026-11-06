package grpv1

// Hello 连接握手
type Hello struct {
	DeviceID        string   `json:"device_id"`
	DeviceKey       string   `json:"device_key,omitempty"` // 仅首次注册时携带
	AgentVersion    string   `json:"agent_version"`
	ProtocolVersion string   `json:"protocol_version"`
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	Nonce           string   `json:"nonce"`
	SupportedCodecs []string `json:"supported_codecs,omitempty"`
}

// Challenge 认证挑战
type Challenge struct {
	ChallengeID      string   `json:"challenge_id"`
	Nonce            string   `json:"nonce"`
	ExpiresAt        int64    `json:"expires_at"`
	SupportedVersion []string `json:"supported_versions,omitempty"`
}

// Authenticate 认证响应
type Authenticate struct {
	DeviceID          string `json:"device_id"`
	ChallengeID       string `json:"challenge_id"`
	Signature         string `json:"signature"` // hex(HMAC-SHA256(device_key, challenge.nonce))
	CredentialVersion string `json:"credential_version,omitempty"`
}

// AuthenticateResult 认证结果
type AuthenticateResult struct {
	Success           bool   `json:"success"`
	ConnectionID      string `json:"connection_id,omitempty"`
	ServerNodeID      string `json:"server_node_id,omitempty"`
	HeartbeatInterval uint32 `json:"heartbeat_interval,omitempty"`
	HeartbeatTimeout  uint32 `json:"heartbeat_timeout,omitempty"`
	MaxStreams        uint32 `json:"max_streams,omitempty"`
	MaxFrameSize      uint32 `json:"max_frame_size,omitempty"`
	Message           string `json:"message,omitempty"`
}

// Capabilities 设备能力
type Capabilities struct {
	Features            []string `json:"features,omitempty"`
	Commands            []string `json:"commands,omitempty"`
	MaxTerminalSessions uint32   `json:"max_terminal_sessions,omitempty"`
	MaxFileTransfers    uint32   `json:"max_file_transfers,omitempty"`
	MaxFileSize         uint64   `json:"max_file_size,omitempty"`
}

// Heartbeat 心跳
type Heartbeat struct {
	AgentUptime    uint64 `json:"agent_uptime"`
	Sequence       uint64 `json:"sequence"`
	ActiveSessions uint32 `json:"active_sessions"`
	ActiveTasks    uint32 `json:"active_tasks"`
	PendingEvents  uint32 `json:"pending_events"`
	AgentState     string `json:"agent_state"`
}

// HeartbeatAck 心跳响应
type HeartbeatAck struct {
	ServerTime            int64  `json:"server_time"`
	NextHeartbeatInterval uint32 `json:"next_heartbeat_interval"`
	SyncRequired          bool   `json:"sync_required"`
}

// SystemStatus 系统状态
type SystemStatus struct {
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryTotal   uint64  `json:"memory_total"`
	MemoryUsed    uint64  `json:"memory_used"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskUsed      uint64  `json:"disk_used"`
	Temperature   float64 `json:"temperature"`
	NetworkRxByte uint64  `json:"network_rx_bytes"`
	NetworkTxByte uint64  `json:"network_tx_bytes"`
	Uptime        uint64  `json:"uptime"`
	Hostname      string  `json:"hostname"`
}

// ErrorMessage 错误消息负载
type ErrorMessage struct {
	Code      uint32 `json:"code"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
	Retryable bool   `json:"retryable"`
	RequestID string `json:"request_id,omitempty"`
}