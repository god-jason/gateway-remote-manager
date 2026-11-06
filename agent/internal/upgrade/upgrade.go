// Package upgrade Agent 端自升级：下载、校验、备份、替换与重启。
package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"

	"gateway-manager/pkg/grpv1"
	"gateway-manager/pkg/logger"
)

var running atomic.Bool

// OnProgress 进度回调
type OnProgress func(grpv1.UpgradeProgress)

// OnResult 结果回调
type OnResult func(grpv1.UpgradeResult)

// Execute 下载并替换当前 Agent 可执行文件
func Execute(ctx context.Context, start grpv1.UpgradeStart, onProgress OnProgress, onResult OnResult) {
	if !running.CompareAndSwap(false, true) {
		onResult(grpv1.UpgradeResult{UpgradeID: start.UpgradeID, Status: "failed", Message: "已有升级任务进行中"})
		return
	}
	defer running.Store(false)

	res := grpv1.UpgradeResult{UpgradeID: start.UpgradeID, Version: start.Version}
	selfPath, err := os.Executable()
	if err != nil {
		res.Status = "failed"
		res.Message = err.Error()
		onResult(res)
		return
	}

	// 下载 + 校验
	onProgress(grpv1.UpgradeProgress{UpgradeID: start.UpgradeID, Progress: 0, Stage: "downloading"})
	tmp := selfPath + ".new"
	if err := download(ctx, start.URL, tmp, start.SHA256, func(p uint32) {
		onProgress(grpv1.UpgradeProgress{UpgradeID: start.UpgradeID, Progress: p, Stage: "downloading"})
	}); err != nil {
		_ = os.Remove(tmp)
		res.Status = "failed"
		res.Message = err.Error()
		onResult(res)
		return
	}

	// 备份当前程序
	onProgress(grpv1.UpgradeProgress{UpgradeID: start.UpgradeID, Progress: 90, Stage: "backing_up"})
	if _, err := os.Stat(selfPath); err == nil {
		if err := copyFile(selfPath, selfPath+".bak."+start.Version); err != nil {
			logger.Warn("备份当前程序失败", "error", err)
		}
	}

	// 替换
	onProgress(grpv1.UpgradeProgress{UpgradeID: start.UpgradeID, Progress: 95, Stage: "replacing"})
	_ = os.Chmod(tmp, 0755)
	if err := os.Rename(tmp, selfPath); err != nil {
		res.Status = "failed"
		res.Message = err.Error()
		onResult(res)
		return
	}

	onProgress(grpv1.UpgradeProgress{UpgradeID: start.UpgradeID, Progress: 100, Stage: "done"})
	res.Status = "success"
	onResult(res)
}

func download(ctx context.Context, url, dst, expectSHA string, onProgress func(uint32)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	total := resp.ContentLength
	buf := make([]byte, 64*1024)
	var written int64
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, _ = f.Write(buf[:n])
			_, _ = h.Write(buf[:n])
			written += int64(n)
			if total > 0 {
				onProgress(uint32(written * 100 / total))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}

	if expectSHA != "" {
		sum := hex.EncodeToString(h.Sum(nil))
		if sum != expectSHA {
			return fmt.Errorf("sha256 校验失败")
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}