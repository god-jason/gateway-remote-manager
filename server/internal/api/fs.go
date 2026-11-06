package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"gateway-manager/pkg/curd"
	"gateway-manager/pkg/grpv1"
	"gateway-manager/server/internal/device"
)

// registerFsRoutes 注册文件管理路由（JWT 之后）
func registerFsRoutes(r *gin.RouterGroup) {
	r.POST("/device/:deviceId/fs/list", fsList)
	r.POST("/device/:deviceId/fs/mkdir", fsMkdir)
	r.POST("/device/:deviceId/fs/delete", fsDelete)
	r.POST("/device/:deviceId/fs/rename", fsRename)
	r.POST("/device/:deviceId/fs/copy", fsCopy)
	r.POST("/device/:deviceId/fs/info", fsInfo)
	r.POST("/device/:deviceId/fs/read", fsRead)
	r.POST("/device/:deviceId/fs/write", fsWrite)
}

// fsCall 向 Agent 发起文件管理请求并解出结果
func fsCall(c *gin.Context, typ string, payload interface{}, out *grpv1.FsResult) bool {
	deviceID := c.Param("deviceId")
	conn, ok := device.Default.Connection(deviceID)
	if !ok {
		curd.ErrorWithStatus(c, 404, 404, "设备不在线")
		return false
	}
	env := grpv1.NewEnvelope(grpv1.KindRequest, typ, deviceID)
	env.Payload, _ = grpv1.MarshalPayload(payload)

	resp, err := conn.Request(c.Request.Context(), env, 15*time.Second)
	if err != nil {
		curd.ErrorWithStatus(c, 500, 500, err.Error())
		return false
	}
	if err := grpv1.UnmarshalPayload(resp.Payload, out); err != nil {
		curd.ErrorWithStatus(c, 500, 500, "解析响应失败")
		return false
	}
	if !out.Success {
		curd.ErrorWithStatus(c, 400, 400, out.Message)
		return false
	}
	return true
}

func fsList(c *gin.Context) {
	var req grpv1.PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsDirList, &req, &out) {
		curd.Success(c, out)
	}
}

func fsMkdir(c *gin.Context) {
	var req grpv1.PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsDirCreate, &req, &out) {
		curd.Success(c, nil)
	}
}

func fsDelete(c *gin.Context) {
	var req grpv1.PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsFileDelete, &req, &out) {
		curd.Success(c, nil)
	}
}

func fsRename(c *gin.Context) {
	var req grpv1.RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsFileRename, &req, &out) {
		curd.Success(c, nil)
	}
}

func fsCopy(c *gin.Context) {
	var req grpv1.CopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsFileCopy, &req, &out) {
		curd.Success(c, nil)
	}
}

func fsInfo(c *gin.Context) {
	var req grpv1.PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsFileInfo, &req, &out) {
		curd.Success(c, out)
	}
}

func fsRead(c *gin.Context) {
	var req grpv1.PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsFileRead, &req, &out) {
		curd.Success(c, out)
	}
}

func fsWrite(c *gin.Context) {
	var req grpv1.WriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		curd.Error(c, 400, "无效参数")
		return
	}
	var out grpv1.FsResult
	if fsCall(c, grpv1.TypeFsFileWrite, &req, &out) {
		curd.Success(c, nil)
	}
}