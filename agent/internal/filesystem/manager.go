// Package filesystem Agent 端文件传输：在 Stream 上收发文件，含 SHA256 校验与原子替换。
package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"gateway-manager/pkg/grpv1"
	"gateway-manager/pkg/logger"
)

const chunkSize = 64 * 1024

// SendResult 发送传输结果事件的回调
type SendResult func(result grpv1.FileTransferResult)

// Upload 接收文件：Stream -> 临时文件 -> SHA256 校验 -> 原子替换
func Upload(ctx context.Context, stream grpv1.Stream, open grpv1.FileTransferOpen, send SendResult) {
	res := grpv1.FileTransferResult{TransferID: open.TransferID}
	defer func() {
		if !res.Success {
			logger.Warn("文件上传失败", "transfer", open.TransferID, "path", open.Path, "error", res.Message)
		}
		send(res)
	}()

	target := filepath.Clean(open.Path)
	if !open.Overwrite {
		if _, err := os.Stat(target); err == nil {
			res.Message = "目标文件已存在"
			return
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		res.Message = err.Error()
		return
	}

	tmp := target + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		res.Message = err.Error()
		return
	}
	defer func() {
		if !res.Success {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()

	h := sha256.New()
	buf := make([]byte, chunkSize)
	written := int64(0)
	for {
		n, err := stream.Read(ctx, buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				res.Message = werr.Error()
				return
			}
			h.Write(buf[:n])
			written += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			res.Message = err.Error()
			return
		}
	}
	if err := f.Close(); err != nil {
		res.Message = err.Error()
		return
	}

	sum := hex.EncodeToString(h.Sum(nil))
	if open.SHA256 != "" && open.SHA256 != sum {
		res.Message = "SHA256 校验失败"
		return
	}
	if err := os.Rename(tmp, target); err != nil {
		res.Message = err.Error()
		return
	}

	res.Success = true
	res.SHA256 = sum
	res.FileSize = written
	logger.Info("文件上传完成", "transfer", open.TransferID, "path", open.Path, "size", written)
}

// Download 发送文件：文件 -> Stream
func Download(ctx context.Context, stream grpv1.Stream, path string, send SendResult) {
	res := grpv1.FileTransferResult{}
	defer send(res)

	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		res.Message = err.Error()
		return
	}
	defer f.Close()

	buf := make([]byte, chunkSize)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			if _, werr := stream.Write(ctx, buf[:n]); werr != nil {
				res.Message = werr.Error()
				return
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			res.Message = err.Error()
			return
		}
	}
	// 通知接收方数据结束
	_ = stream.Close()
	res.Success = true
	logger.Info("文件下载完成", "path", path)
}