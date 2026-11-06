package grpv1

// 文件管理消息类型（请求/响应，走控制通道）
const (
	TypeFsDirList   = "filesystem.directory.list"
	TypeFsDirCreate = "filesystem.directory.create"
	TypeFsFileInfo  = "filesystem.file.info"
	TypeFsFileRead  = "filesystem.file.read"
	TypeFsFileWrite = "filesystem.file.write"
	TypeFsFileDelete = "filesystem.file.delete"
	TypeFsFileRename = "filesystem.file.rename"
	TypeFsFileCopy   = "filesystem.file.copy"
	TypeFsFileMove   = "filesystem.file.move"
)

// FileEntry 文件条目
type FileEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path,omitempty"`
	IsDir      bool   `json:"is_dir"`
	Size       int64  `json:"size"`
	Mode       uint32 `json:"mode,omitempty"`
	Perm       string `json:"perm,omitempty"`
	ModifiedAt int64  `json:"modified_at,omitempty"`
}

// PathRequest 路径请求
type PathRequest struct {
	Path string `json:"path"`
}

// RenameRequest 重命名/移动请求
type RenameRequest struct {
	Path    string `json:"path"`
	NewPath string `json:"new_path"`
}

// CopyRequest 复制请求
type CopyRequest struct {
	Path string `json:"path"`
	Dest string `json:"dest"`
}

// WriteRequest 写文件请求
type WriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FsResult 文件操作结果（list 含 entries，read 含 content，info 含 entry）
type FsResult struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Path      string      `json:"path,omitempty"`
	Entries   []FileEntry `json:"entries,omitempty"`
	Entry     *FileEntry  `json:"entry,omitempty"`
	Content   string      `json:"content,omitempty"`
	Size      int64       `json:"size,omitempty"`
	Truncated bool        `json:"truncated,omitempty"`
}