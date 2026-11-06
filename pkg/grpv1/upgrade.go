package grpv1

// 升级消息类型
const (
	TypeUpgradeStart    = "upgrade.start"
	TypeUpgradeProgress = "upgrade.progress"
	TypeUpgradeResult   = "upgrade.result"
)

// UpgradeStart 下发升级任务
type UpgradeStart struct {
	UpgradeID string `json:"upgrade_id"`
	URL       string `json:"url"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256"`
}

// UpgradeProgress 升级进度
type UpgradeProgress struct {
	UpgradeID string `json:"upgrade_id"`
	Progress  uint32 `json:"progress"`           // 0-100
	Stage     string `json:"stage,omitempty"`    // downloading / verifying / backing_up / replacing / done
}

// UpgradeResult 升级结果
type UpgradeResult struct {
	UpgradeID string `json:"upgrade_id"`
	Status    string `json:"status"` // success / failed
	Version   string `json:"version,omitempty"`
	Message   string `json:"message,omitempty"`
}