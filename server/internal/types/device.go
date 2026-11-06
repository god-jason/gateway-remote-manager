// Package types 服务端数据模型。
package types

import "time"

// Device 网关设备
type Device struct {
	ID           string    `xorm:"'id' pk varchar(24)" json:"id"`
	DeviceID     string    `xorm:"device_id varchar(64) unique" json:"device_id"`
	Name         string    `xorm:"name varchar(128)" json:"name"`
	ProductID    string    `xorm:"product_id varchar(64)" json:"product_id"`
	HardwareID   string    `xorm:"hardware_id varchar(128)" json:"hardware_id"`
	AgentVersion string    `xorm:"agent_version varchar(32)" json:"agent_version"`
	OS           string    `xorm:"os varchar(64)" json:"os"`
	Architecture string    `xorm:"architecture varchar(32)" json:"architecture"`
	Status       string    `xorm:"status varchar(32)" json:"status"` // online / offline / disabled
	LastSeenAt   time.Time `xorm:"last_seen_at" json:"last_seen_at"`
	Created      time.Time `xorm:"created" json:"created"`
}

// DeviceCredential 设备凭据（密钥生产环境应加密存储，并支持轮换/吊销）
type DeviceCredential struct {
	ID       string    `xorm:"'id' pk varchar(24)" json:"id"`
	DeviceID string    `xorm:"device_id varchar(64) unique" json:"device_id"`
	Key      string    `xorm:"key varchar(256)" json:"-"`
	Created  time.Time `xorm:"created" json:"created"`
}