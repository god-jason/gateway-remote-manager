//go:build windows

package terminal

import (
	"errors"
	"os"
)

// ErrUnsupported Windows 平台不支持 PTY（Agent 目标平台为 Linux 网关）
var ErrUnsupported = errors.New("远程终端暂不支持当前平台（目标平台为 Linux 网关）")

// Terminal 空实现，保证跨平台编译
type Terminal struct {
	File *os.File
}

// Start 返回 unsupported 错误
func Start(shell string, cols, rows uint16, wd string) (*Terminal, error) {
	return nil, ErrUnsupported
}

// Resize 返回 unsupported 错误
func (t *Terminal) Resize(rows, cols uint16) error { return ErrUnsupported }

// Close 空实现
func (t *Terminal) Close() {}