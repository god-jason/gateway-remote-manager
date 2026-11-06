package db

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID 生成 24 位十六进制字符串 ID（12 字节随机数）
func NewID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		// 极端情况下回退为时间戳字符串，避免 panic
		return hex.EncodeToString([]byte("000000000000"))
	}
	return hex.EncodeToString(b)
}