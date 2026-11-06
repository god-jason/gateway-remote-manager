// Package identity 管理 Agent 设备身份（device_id + device_key）。
// 身份持久化到本地文件，首次运行时自动生成。
package identity

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gateway-manager/pkg/grpv1"
)

// Identity 设备身份
type Identity struct {
	DeviceID   string `json:"device_id"`
	DeviceKey  string `json:"device_key"`
	Registered bool   `json:"registered"` // 服务器是否已确认注册
}

// Load 从文件加载身份，不存在则生成新身份（不落盘，由调用方决定保存时机）
func Load(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return nil, fmt.Errorf("解析身份文件失败: %w", err)
		}
		if id.DeviceID != "" && id.DeviceKey != "" {
			return &id, nil
		}
	}
	return Generate(), nil
}

// Generate 生成新身份
func Generate() *Identity {
	id := &Identity{
		DeviceID:  "gw-" + randomHex(6),
		DeviceKey: randomHex(32),
	}
	return id
}

// Save 持久化身份
func (i *Identity) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Sign 使用设备密钥对挑战随机数签名（HMAC-SHA256），返回十六进制
func (i *Identity) Sign(nonce string) string {
	return grpv1.SignChallenge(i.DeviceKey, nonce)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// 极端情况回退固定值（避免阻塞启动）
		for i := range b {
			b[i] = byte(i)
		}
	}
	return hex.EncodeToString(b)
}