package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/server/internal/device"
)

const transferChunkSize = 64 * 1024

// registerFileRoutes 注册文件传输路由（JWT 之后）
func registerFileRoutes(r *gin.RouterGroup) {
	r.POST("/device/:deviceId/file/upload", UploadFile)
	r.GET("/device/:deviceId/file/download", DownloadFile)
}

// UploadFile 上传文件到网关
func UploadFile(c *gin.Context) {
	deviceID := c.Param("deviceId")
	path := c.Query("path")
	if path == "" {
		curd.Error(c, 400, "缺少 path 参数")
		return
	}
	overwrite := c.Query("overwrite") == "true" || c.Query("overwrite") == "1"

	conn, ok := device.Default.Connection(deviceID)
	if !ok {
		curd.Error(c, 404, "设备不在线")
		return
	}

	// 服务端先落盘并计算 SHA256（保证传输完整性可校验）
	tmp, err := os.CreateTemp("", "gw-upload-*")
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	defer os.Remove(tmp.Name())

	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, h), c.Request.Body)
	if err != nil {
		_ = tmp.Close()
		curd.Error(c, 500, err.Error())
		return
	}
	if err := tmp.Close(); err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	sum := hex.EncodeToString(h.Sum(nil))

	ctx := c.Request.Context()
	stream, err := conn.OpenStream(ctx, grpv1.StreamOptions{ContentType: "file"})
	if err != nil {
		curd.Error(c, 500, "打开流失败: "+err.Error())
		return
	}
	defer stream.Close()

	transferID := uuid.NewString()
	openEnv := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeFileOpen, deviceID)
	openEnv.Payload, _ = grpv1.MarshalPayload(grpv1.FileTransferOpen{
		TransferID: transferID,
		StreamID:   stream.ID(),
		Path:       path,
		Direction:  grpv1.DirectionUpload,
		FileSize:   size,
		SHA256:     sum,
		Overwrite:  overwrite,
	})

	resp, err := conn.Request(ctx, openEnv, 10*time.Second)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	var ack grpv1.FileTransferResult
	_ = grpv1.UnmarshalPayload(resp.Payload, &ack)
	if !ack.Success {
		curd.Error(c, 400, ack.Message)
		return
	}

	// 订阅传输结果事件
	resultCh := make(chan grpv1.FileTransferResult, 1)
	unsub := conn.SubscribeEvent(func(env *grpv1.Envelope) {
		if env.Type != grpv1.TypeFileResult {
			return
		}
		var r grpv1.FileTransferResult
		if grpv1.UnmarshalPayload(env.Payload, &r) == nil && r.TransferID == transferID {
			select {
			case resultCh <- r:
			default:
			}
		}
	})
	defer unsub()

	// 流式发送文件数据
	src, err := os.Open(tmp.Name())
	if err != nil {
		_ = stream.Reset(err)
		curd.Error(c, 500, err.Error())
		return
	}
	defer src.Close()

	buf := make([]byte, transferChunkSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := stream.Write(ctx, buf[:n]); werr != nil {
				_ = stream.Reset(werr)
				curd.Error(c, 500, werr.Error())
				return
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = stream.Reset(err)
			curd.Error(c, 500, err.Error())
			return
		}
	}
	_ = stream.Close()

	select {
	case r := <-resultCh:
		if !r.Success {
			curd.Error(c, 400, r.Message)
			return
		}
		curd.Success(c, gin.H{"transfer_id": transferID, "sha256": r.SHA256, "size": r.FileSize})
	case <-ctx.Done():
	case <-time.After(15 * time.Minute):
		curd.Error(c, 500, "等待传输结果超时")
	}
}

// DownloadFile 从网关下载文件
func DownloadFile(c *gin.Context) {
	deviceID := c.Param("deviceId")
	path := c.Query("path")
	if path == "" {
		curd.Error(c, 400, "缺少 path 参数")
		return
	}

	conn, ok := device.Default.Connection(deviceID)
	if !ok {
		curd.ErrorWithStatus(c, http.StatusNotFound, 404, "设备不在线")
		return
	}

	ctx := c.Request.Context()
	stream, err := conn.OpenStream(ctx, grpv1.StreamOptions{ContentType: "file"})
	if err != nil {
		curd.ErrorWithStatus(c, http.StatusInternalServerError, 500, "打开流失败: "+err.Error())
		return
	}
	defer stream.Close()

	transferID := uuid.NewString()
	openEnv := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeFileOpen, deviceID)
	openEnv.Payload, _ = grpv1.MarshalPayload(grpv1.FileTransferOpen{
		TransferID: transferID,
		StreamID:   stream.ID(),
		Path:       path,
		Direction:  grpv1.DirectionDownload,
	})

	resp, err := conn.Request(ctx, openEnv, 10*time.Second)
	if err != nil {
		curd.ErrorWithStatus(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	var ack grpv1.FileTransferResult
	_ = grpv1.UnmarshalPayload(resp.Payload, &ack)
	if !ack.Success {
		curd.ErrorWithStatus(c, http.StatusNotFound, 404, ack.Message)
		return
	}

	filename := filepath.Base(path)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/octet-stream")
	if ack.FileSize > 0 {
		c.Header("Content-Length", strconv.FormatInt(ack.FileSize, 10))
	}
	c.Status(http.StatusOK)

	flusher, _ := c.Writer.(http.Flusher)
	buf := make([]byte, transferChunkSize)
	for {
		n, err := stream.Read(ctx, buf)
		if n > 0 {
			if _, werr := c.Writer.Write(buf[:n]); werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}