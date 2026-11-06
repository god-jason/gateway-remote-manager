package grpv1

// 文件传输消息类型
const (
	TypeFileOpen   = "filesystem.transfer.open"
	TypeFileResult = "filesystem.transfer.result"
)

// 传输方向
const (
	DirectionUpload   = "upload"   // Server -> Agent：写入网关文件
	DirectionDownload = "download" // Agent -> Server：读取网关文件
)

// FileTransferOpen 打开文件传输会话（携带关联的 Stream ID）
type FileTransferOpen struct {
	TransferID string `json:"transfer_id"`
	SessionID  string `json:"session_id,omitempty"`
	StreamID   uint32 `json:"stream_id"`
	Path       string `json:"path"`
	Direction  string `json:"direction"` // upload / download
	FileSize   int64  `json:"file_size,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	ChunkSize  uint32 `json:"chunk_size,omitempty"`
	Overwrite  bool   `json:"overwrite"`
}

// FileTransferResult 传输结果（open 的应答 + 完成/失败事件共用）
type FileTransferResult struct {
	Success    bool   `json:"success"`
	TransferID string `json:"transfer_id,omitempty"`
	FileSize   int64  `json:"file_size,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	Message    string `json:"message,omitempty"`
}