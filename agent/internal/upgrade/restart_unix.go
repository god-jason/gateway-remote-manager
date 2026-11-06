//go:build !windows

package upgrade

import (
	"os"
	"syscall"

	"gateway-manager/pkg/logger"
)

// Restart 用替换后的新二进制重启当前进程（exec 替换进程镜像）
func Restart() {
	exe, err := os.Executable()
	if err != nil {
		logger.Warn("获取可执行文件失败", "error", err)
		return
	}
	logger.Info("Agent 正在重启以应用新版本", "path", exe)
	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		logger.Warn("重启失败", "error", err)
	}
}