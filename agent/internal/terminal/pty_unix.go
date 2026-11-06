//go:build !windows

package terminal

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// Terminal 包装 PTY 主端与子进程
type Terminal struct {
	File *os.File
	cmd  *exec.Cmd
}

// Start 创建 PTY 并启动 shell，目标 shell 不可用时回退到 /bin/sh
func Start(shell string, cols, rows uint16, wd string) (*Terminal, error) {
	if shell == "" {
		shell = "/bin/sh"
	}
	t, err := startShell(shell, cols, rows, wd)
	if err != nil && shell != "/bin/sh" {
		t, err = startShell("/bin/sh", cols, rows, wd)
	}
	return t, err
}

func startShell(shell string, cols, rows uint16, wd string) (*Terminal, error) {
	c := exec.Command(shell)
	if wd != "" {
		c.Dir = wd
	}
	c.Env = append(os.Environ(), "TERM=xterm-256color")

	f, err := pty.Start(c)
	if err != nil {
		return nil, err
	}
	if err := pty.Setsize(f, &pty.Winsize{Rows: rows, Cols: cols}); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &Terminal{File: f, cmd: c}, nil
}

// Resize 调整 PTY 窗口
func (t *Terminal) Resize(rows, cols uint16) error {
	return pty.Setsize(t.File, &pty.Winsize{Rows: rows, Cols: cols})
}

// Close 终止子进程并关闭 PTY
func (t *Terminal) Close() {
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
		_ = t.cmd.Wait()
	}
	if t.File != nil {
		_ = t.File.Close()
	}
}