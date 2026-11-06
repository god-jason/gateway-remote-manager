package curd

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"xorm.io/builder"
)

// Response 统一响应结构
type Response struct {
	Code  int         `json:"code"`
	Msg   string      `json:"message"`
	Data  interface{} `json:"data,omitempty"`
	Total int64       `json:"total,omitempty"`
}

// FilterCondition 过滤条件
type FilterCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte, like, in
	Value    interface{} `json:"value"`
}

// ListParams 列表查询参数
type ListParams struct {
	Page     int               `form:"page" json:"page"`
	PageSize int               `form:"page_size" json:"page_size"`
	Keyword  string            `form:"keyword" json:"keyword"`
	Filter   []FilterCondition `form:"filter" json:"filter"`
	OrderBy  string            `form:"order_by" json:"order_by"`
	Order    string            `form:"order" json:"order"`
}

// Normalize 标准化列表参数
func (l *ListParams) Normalize() {
	if l.Page <= 0 {
		l.Page = 1
	}
	if l.PageSize <= 0 {
		l.PageSize = 20
	}
	if l.PageSize > 100 {
		l.PageSize = 100
	}
	if l.OrderBy == "" {
		l.OrderBy = "id"
	}
	if l.Order == "" {
		l.Order = "desc"
	}
	if l.Order != "asc" && l.Order != "desc" {
		l.Order = "desc"
	}
}

// HasKeyword 是否有关键字搜索
func (l *ListParams) HasKeyword() bool { return l.Keyword != "" }

// BuildCond 依据过滤条件构建 xorm builder.Cond，无过滤返回 nil
func (l *ListParams) BuildCond() builder.Cond {
	var conds []builder.Cond
	for _, f := range l.Filter {
		if c := condFromOp(f.Operator, f.Field, f.Value); c != nil {
			conds = append(conds, c)
		}
	}
	if len(conds) == 0 {
		return nil
	}
	if len(conds) == 1 {
		return conds[0]
	}
	return builder.And(conds...)
}

func condFromOp(op, field string, val interface{}) builder.Cond {
	switch op {
	case "eq", "":
		return builder.Eq{field: val}
	case "ne":
		return builder.Neq{field: val}
	case "gt":
		return builder.Gt{field: val}
	case "lt":
		return builder.Lt{field: val}
	case "gte":
		return builder.Gte{field: val}
	case "lte":
		return builder.Lte{field: val}
	case "like":
		s, _ := val.(string)
		return builder.Like{field, "%" + s + "%"}
	case "in":
		arr, ok := val.([]interface{})
		if !ok {
			return nil
		}
		return builder.In(field, arr...)
	}
	return nil
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "success", Data: data})
}

// SuccessTotal 成功响应（带总数）
func SuccessTotal(c *gin.Context, data interface{}, total int64) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "success", Data: data, Total: total})
}

// Error 错误响应（HTTP 200 + 业务码）
func Error(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{Code: code, Msg: msg})
}

// ErrorWithStatus 带 HTTP 状态码的错误响应
func ErrorWithStatus(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, Response{Code: code, Msg: msg})
}

// Unauthorized 401
func Unauthorized(c *gin.Context, msg string) {
	ErrorWithStatus(c, http.StatusUnauthorized, 401, msg)
}

// ParseJSON 解析 JSON 请求体
func ParseJSON(c *gin.Context, obj interface{}) error {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(obj)
}