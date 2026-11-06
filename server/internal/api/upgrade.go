package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"xorm.io/builder"

	"gateway-manager/pkg/config"
	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/db"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/server/internal/device"
	"gateway-manager/server/internal/types"
)

// RegisterUpgradePublic 注册公开路由（Agent 下载升级包，无需认证）
func RegisterUpgradePublic(r gin.IRoutes) {
	r.GET("/upgrade/release/:id/file", releaseFileDownload)
}

// registerUpgradeRoutes 注册升级管理路由（JWT 之后）
func registerUpgradeRoutes(r *gin.RouterGroup) {
	r.POST("/upgrade/release/upload", releaseUpload)
	r.POST("/upgrade/release/search", releaseSearch)
	r.GET("/upgrade/release/:id", releaseGet)
	r.DELETE("/upgrade/release/:id", releaseDelete)
	r.PUT("/upgrade/release/:id/publish", releasePublish)
	r.PUT("/upgrade/release/:id/disable", releaseDisable)

	r.POST("/upgrade/task/search", upgradeTaskSearch)
	r.POST("/device/:deviceId/upgrade", upgradeDevice)
}

// ===== 版本仓库 =====

func releaseUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		curd.Error(c, 400, "缺少文件")
		return
	}
	defer file.Close()

	version := c.PostForm("version")
	osName := c.PostForm("os")
	arch := c.PostForm("arch")
	notes := c.PostForm("notes")
	if version == "" || osName == "" || arch == "" {
		curd.Error(c, 400, "缺少 version/os/arch")
		return
	}

	dir := config.GetString("upgrade", "dir")
	if err := os.MkdirAll(dir, 0755); err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	id := db.NewID()
	dst := filepath.Join(dir, id+"-"+filepath.Base(header.Filename))
	out, err := os.Create(dst)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(out, h), file)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(dst)
		curd.Error(c, 500, err.Error())
		return
	}

	rel := types.AgentRelease{
		ID:        id,
		Version:   version,
		OS:        osName,
		Arch:      arch,
		FileName:  header.Filename,
		FilePath:  dst,
		Size:      size,
		SHA256:    hex.EncodeToString(h.Sum(nil)),
		Notes:     notes,
		Published: true,
		Created:   time.Now(),
	}
	if _, err := db.InsertOne("agent_release", &rel); err != nil {
		_ = os.Remove(dst)
		curd.Error(c, 500, err.Error())
		return
	}
	curd.Success(c, rel)
}

func releaseSearch(c *gin.Context) {
	var list []types.AgentRelease
	if err := db.Find("agent_release", nil, &list, 0, 0); err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Created.After(list[j].Created) })
	curd.Success(c, list)
}

func releaseGet(c *gin.Context) {
	var rel types.AgentRelease
	found, err := db.FindOne("agent_release", builder.Eq{"id": c.Param("id")}, &rel)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	if !found {
		curd.Error(c, 404, "版本不存在")
		return
	}
	curd.Success(c, rel)
}

func releaseDelete(c *gin.Context) {
	var rel types.AgentRelease
	found, _ := db.FindOne("agent_release", builder.Eq{"id": c.Param("id")}, &rel)
	if found && rel.FilePath != "" {
		_ = os.Remove(rel.FilePath)
	}
	_, _ = db.DeleteOne("agent_release", builder.Eq{"id": c.Param("id")})
	curd.Success(c, nil)
}

func releasePublish(c *gin.Context) {
	_, _ = db.UpdateOne("agent_release", builder.Eq{"id": c.Param("id")}, map[string]interface{}{"published": true})
	curd.Success(c, nil)
}

func releaseDisable(c *gin.Context) {
	_, _ = db.UpdateOne("agent_release", builder.Eq{"id": c.Param("id")}, map[string]interface{}{"published": false})
	curd.Success(c, nil)
}

// 公开：Agent 下载升级包
func releaseFileDownload(c *gin.Context) {
	var rel types.AgentRelease
	found, _ := db.FindOne("agent_release", builder.Eq{"id": c.Param("id")}, &rel)
	if !found || rel.FilePath == "" {
		c.JSON(404, gin.H{"code": 404, "message": "版本不存在"})
		return
	}
	c.FileAttachment(rel.FilePath, rel.FileName)
}

// ===== 升级任务 =====

func upgradeDevice(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var req struct {
		ReleaseID string `json:"release_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ReleaseID == "" {
		curd.Error(c, 400, "缺少 release_id")
		return
	}

	var rel types.AgentRelease
	found, err := db.FindOne("agent_release", builder.And(builder.Eq{"id": req.ReleaseID}, builder.Eq{"published": true}), &rel)
	if err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	if !found {
		curd.Error(c, 404, "版本不存在或未发布")
		return
	}

	task := types.UpgradeTask{
		ID:        db.NewID(),
		UpgradeID: uuid.NewString(),
		DeviceID:  deviceID,
		ReleaseID: rel.ID,
		Version:   rel.Version,
		Status:    "pending",
		Created:   time.Now(),
	}
	if _, err := db.InsertOne("upgrade_task", &task); err != nil {
		curd.Error(c, 500, err.Error())
		return
	}

	conn, ok := device.Default.Connection(deviceID)
	if !ok {
		curd.Success(c, task)
		return
	}

	url := upgradeBaseURL(c) + "/api/upgrade/release/" + rel.ID + "/file"
	env := grpv1.NewEnvelope(grpv1.KindRequest, grpv1.TypeUpgradeStart, deviceID)
	env.Payload, _ = grpv1.MarshalPayload(grpv1.UpgradeStart{
		UpgradeID: task.UpgradeID, URL: url, Version: rel.Version, SHA256: rel.SHA256,
	})
	resp, err := conn.Request(context.Background(), env, 5*time.Second)
	if err != nil {
		updateUpgradeStatus(deviceID, task.UpgradeID, "error", err.Error())
	} else {
		var ack grpv1.UpgradeResult
		_ = grpv1.UnmarshalPayload(resp.Payload, &ack)
		if ack.Status != "success" {
			updateUpgradeStatus(deviceID, task.UpgradeID, "error", ack.Message)
		} else {
			updateUpgradeStatus(deviceID, task.UpgradeID, "dispatched", "")
			// 订阅进度与结果
			_ = conn.SubscribeEvent(func(ev *grpv1.Envelope) {
				switch ev.Type {
				case grpv1.TypeUpgradeProgress:
					var p grpv1.UpgradeProgress
					if grpv1.UnmarshalPayload(ev.Payload, &p) == nil && p.UpgradeID == task.UpgradeID {
						updateUpgradeStatus(deviceID, task.UpgradeID, p.Stage, "")
					}
				case grpv1.TypeUpgradeResult:
					var r grpv1.UpgradeResult
					if grpv1.UnmarshalPayload(ev.Payload, &r) == nil && r.UpgradeID == task.UpgradeID {
						updateUpgradeStatus(deviceID, task.UpgradeID, r.Status, r.Message)
					}
				}
			})
		}
	}

	task.Status = "dispatched"
	curd.Success(c, task)
}

func upgradeTaskSearch(c *gin.Context) {
	var list []types.UpgradeTask
	if err := db.Find("upgrade_task", nil, &list, 0, 0); err != nil {
		curd.Error(c, 500, err.Error())
		return
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Created.After(list[j].Created) })
	curd.Success(c, list)
}

func updateUpgradeStatus(deviceID, upgradeID, status, message string) {
	fields := map[string]interface{}{"status": status}
	if message != "" {
		fields["message"] = message
	}
	if status == "success" || status == "failed" || status == "error" {
		fields["finished_at"] = time.Now()
	}
	_, _ = db.UpdateOne("upgrade_task", builder.And(builder.Eq{"device_id": deviceID}, builder.Eq{"upgrade_id": upgradeID}), fields)
}

func upgradeBaseURL(c *gin.Context) string {
	base := strings.TrimRight(config.GetString("upgrade", "base_url"), "/")
	if base != "" {
		return base
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}