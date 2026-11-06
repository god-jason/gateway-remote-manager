package types

import "time"

// AgentRelease Agent 版本仓库条目
type AgentRelease struct {
	ID        string    `xorm:"'id' pk varchar(24)" json:"id"`
	Version   string    `xorm:"version varchar(64)" json:"version"`
	OS        string    `xorm:"os varchar(32)" json:"os"`
	Arch      string    `xorm:"arch varchar(32)" json:"arch"`
	FileName  string    `xorm:"file_name varchar(256)" json:"file_name"`
	FilePath  string    `xorm:"file_path varchar(512)" json:"file_path"`
	Size      int64     `xorm:"size" json:"size"`
	SHA256    string    `xorm:"sha256 varchar(64)" json:"sha256"`
	Notes     string    `xorm:"notes varchar(1024)" json:"notes"`
	Published bool      `xorm:"published" json:"published"`
	Created   time.Time `xorm:"created" json:"created"`
}

// UpgradeTask 升级任务
type UpgradeTask struct {
	ID         string    `xorm:"'id' pk varchar(24)" json:"id"`
	UpgradeID  string    `xorm:"upgrade_id varchar(64)" json:"upgrade_id"`
	DeviceID   string    `xorm:"device_id varchar(64)" json:"device_id"`
	ReleaseID  string    `xorm:"release_id varchar(24)" json:"release_id"`
	Version    string    `xorm:"version varchar(64)" json:"version"`
	Status     string    `xorm:"status varchar(32)" json:"status"` // pending/dispatched/downloading/verifying/backing_up/replacing/done/success/failed/error
	Message    string    `xorm:"message varchar(512)" json:"message"`
	Created    time.Time `xorm:"created" json:"created"`
	FinishedAt time.Time `xorm:"finished_at" json:"finished_at"`
}