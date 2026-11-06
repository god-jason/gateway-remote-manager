package types

import "time"

// Config 设备期望配置（配置中心模式，服务端维护期望状态）
type Config struct {
	ID         string    `xorm:"'id' pk varchar(24)" json:"id"`
	DeviceID   string    `xorm:"device_id varchar(64)" json:"device_id"`
	ConfigKey  string    `xorm:"config_key varchar(256)" json:"config_key"`
	ConfigType string    `xorm:"config_type varchar(32)" json:"config_type"`
	Content    string    `xorm:"content text" json:"content"`
	Checksum   string    `xorm:"checksum varchar(64)" json:"checksum"`
	Version    uint64    `xorm:"version" json:"version"`
	Status     string    `xorm:"status varchar(32)" json:"status"` // pending / applied / failed
	Updated    time.Time `xorm:"updated" json:"updated"`
	Created    time.Time `xorm:"created" json:"created"`
}