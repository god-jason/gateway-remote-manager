package grpv1

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// SignChallenge 使用设备密钥对挑战随机数计算 HMAC-SHA256，返回十六进制字符串。
// 第一阶段使用对称共享密钥（device_key）做挑战应答，生产环境应替换为
// ed25519 签名或 mTLS 设备证书（见设计文档第 18 节）。
func SignChallenge(key, nonce string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(nonce))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyChallenge 校验签名是否匹配（常数时间比较）
func VerifyChallenge(sigHex, key, nonce string) bool {
	if sigHex == "" || key == "" {
		return false
	}
	expected := SignChallenge(key, nonce)
	return subtle.ConstantTimeCompare([]byte(sigHex), []byte(expected)) == 1
}