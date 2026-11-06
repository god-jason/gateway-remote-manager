// Package device 设备持久化与连接状态管理。
package device

import (
	"time"

	"xorm.io/builder"

	"gateway-manager/pkg/db"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/server/internal/types"
)

// FindByDeviceID 按设备编号查询
func FindByDeviceID(deviceID string) (*types.Device, bool, error) {
	var d types.Device
	found, err := db.FindOne("device", builder.Eq{"device_id": deviceID}, &d)
	if err != nil || !found {
		return nil, found, err
	}
	return &d, true, nil
}

// FindCredential 查询设备密钥
func FindCredential(deviceID string) (string, bool, error) {
	var c types.DeviceCredential
	found, err := db.FindOne("device_credential", builder.Eq{"device_id": deviceID}, &c)
	return c.Key, found, err
}

// CreateDevice 创建设备
func CreateDevice(d *types.Device) error {
	_, err := db.InsertOne("device", d)
	return err
}

// CreateCredential 创建凭据
func CreateCredential(deviceID, key string) error {
	c := &types.DeviceCredential{ID: db.NewID(), DeviceID: deviceID, Key: key, Created: time.Now()}
	_, err := db.InsertOne("device_credential", c)
	return err
}

// UpdateFields 按 device_id 更新字段
func UpdateFields(deviceID string, fields map[string]interface{}) error {
	_, err := db.UpdateOne("device", builder.Eq{"device_id": deviceID}, fields)
	return err
}

// EnsureDevice 确保设备已注册（自动注册或校验），并补录缺失凭据
func EnsureDevice(hello grpv1.Hello, allowAutoRegister bool) error {
	d, found, err := FindByDeviceID(hello.DeviceID)
	if err != nil {
		return err
	}
	if !found {
		if !allowAutoRegister {
			return grpv1.NewError(grpv1.ErrUnauthorized, "设备未注册且服务器未开启自动注册")
		}
		dev := &types.Device{
			ID:           db.NewID(),
			DeviceID:     hello.DeviceID,
			Name:         hello.DeviceID,
			AgentVersion: hello.AgentVersion,
			OS:           hello.OS,
			Architecture: hello.Arch,
			Status:       "online",
			Created:      time.Now(),
		}
		if err := CreateDevice(dev); err != nil {
			return err
		}
		if hello.DeviceKey != "" {
			return CreateCredential(hello.DeviceID, hello.DeviceKey)
		}
		return nil
	}
	if d.Status == "disabled" {
		return grpv1.NewError(grpv1.ErrDeviceDisabled, "设备已被禁用")
	}
	// 服务端无凭据时按本地提供的 key 补录
	if _, foundKey, _ := FindCredential(hello.DeviceID); !foundKey && hello.DeviceKey != "" {
		_ = CreateCredential(hello.DeviceID, hello.DeviceKey)
	}
	return nil
}

// MarkOnline 标记设备上线
func MarkOnline(deviceID, agentVersion, osName, arch string) error {
	return UpdateFields(deviceID, map[string]interface{}{
		"status":        "online",
		"agent_version": agentVersion,
		"os":            osName,
		"architecture":  arch,
		"last_seen_at":  time.Now(),
	})
}

// MarkOffline 标记设备离线
func MarkOffline(deviceID string) error {
	return UpdateFields(deviceID, map[string]interface{}{
		"status":       "offline",
		"last_seen_at": time.Now(),
	})
}

// TouchSeen 更新最后在线时间
func TouchSeen(deviceID string) error {
	return UpdateFields(deviceID, map[string]interface{}{"last_seen_at": time.Now()})
}