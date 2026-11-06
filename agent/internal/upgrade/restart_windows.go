//go:build windows

package upgrade

import "gateway-manager/pkg/logger"

// Restart Windows 平台不支持 exec 重启，仅提示
func Restart() {
	logger.Info("Agent 已升级，请手动重启")
}