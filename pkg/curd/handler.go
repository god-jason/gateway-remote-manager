package curd

import (
	"reflect"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"xorm.io/builder"

	"gateway-manager/pkg/db"
)

// Handler 泛型 CRUD 处理器
type Handler[T any] struct {
	Collection   string            // 表名
	UpdateFields []string          // 可更新字段白名单（空表示不限制）
	Keywords     []string          // 关键词模糊搜索字段
	Filter       map[string]string // 默认过滤：gin context key -> 字段名
	IsInt64ID    bool              // 主键是否为 int64 自增
}

// Search 分页搜索
func (h *Handler[T]) Search(c *gin.Context) {
	var params ListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		Error(c, 400, "无效的请求参数")
		return
	}
	params.Normalize()

	conds := []builder.Cond{}
	if pc := params.BuildCond(); pc != nil {
		conds = append(conds, pc)
	}
	if params.HasKeyword() && len(h.Keywords) > 0 {
		kws := make([]builder.Cond, 0, len(h.Keywords))
		for _, kw := range h.Keywords {
			kws = append(kws, builder.Like{kw, "%" + params.Keyword + "%"})
		}
		conds = append(conds, builder.Or(kws...))
	}
	conds = append(conds, h.defaultFilterCond(c)...)

	var cond builder.Cond
	if len(conds) > 1 {
		cond = builder.And(conds...)
	} else if len(conds) == 1 {
		cond = conds[0]
	}

	total, err := db.Count(h.Collection, cond)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}

	orderAsc := params.Order == "asc"
	result := make([]T, 0, params.PageSize)
	if err := db.PagedFind(h.Collection, cond, params.Page, params.PageSize, params.OrderBy, orderAsc, &result); err != nil {
		Error(c, 500, err.Error())
		return
	}

	SuccessTotal(c, result, total)
}

// Get 获取单条
func (h *Handler[T]) Get(c *gin.Context) {
	idVal, ok := h.parseID(c.Param("id"))
	if !ok {
		Error(c, 400, "无效的ID")
		return
	}
	conds := []builder.Cond{builder.Eq{"id": idVal}}
	conds = append(conds, h.defaultFilterCond(c)...)

	result := new(T)
	found, err := db.FindOne(h.Collection, builder.And(conds...), result)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	if !found || isZero(result) {
		Error(c, 404, "数据不存在")
		return
	}
	Success(c, result)
}

// Create 创建
func (h *Handler[T]) Create(c *gin.Context) {
	result := new(T)
	if err := c.ShouldBindJSON(result); err != nil {
		Error(c, 400, "无效的请求参数")
		return
	}
	setID(result)
	setCreated(result)
	h.setDefaultFilterValues(result, c)

	if _, err := db.InsertOne(h.Collection, result); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, result)
}

// Update 更新
func (h *Handler[T]) Update(c *gin.Context) {
	idVal, ok := h.parseID(c.Param("id"))
	if !ok {
		Error(c, 400, "无效的ID")
		return
	}
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		Error(c, 400, "无效的请求参数")
		return
	}

	updateSet := map[string]interface{}{}
	for k, v := range updateData {
		if k == "id" || k == "created" || k == "created_at" {
			continue
		}
		if !h.updateAllowed(k) {
			continue
		}
		updateSet[k] = v
	}
	if len(updateSet) == 0 {
		Error(c, 400, "没有可更新的字段")
		return
	}

	conds := []builder.Cond{builder.Eq{"id": idVal}}
	conds = append(conds, h.defaultFilterCond(c)...)
	affected, err := db.UpdateOne(h.Collection, builder.And(conds...), updateSet)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	if affected == 0 {
		Error(c, 404, "数据不存在")
		return
	}
	Success(c, nil)
}

// Delete 删除
func (h *Handler[T]) Delete(c *gin.Context) {
	idVal, ok := h.parseID(c.Param("id"))
	if !ok {
		Error(c, 400, "无效的ID")
		return
	}
	conds := []builder.Cond{builder.Eq{"id": idVal}}
	conds = append(conds, h.defaultFilterCond(c)...)
	affected, err := db.DeleteOne(h.Collection, builder.And(conds...))
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	if affected == 0 {
		Error(c, 404, "数据不存在")
		return
	}
	Success(c, nil)
}

// ============ 辅助 ============

func (h *Handler[T]) updateAllowed(k string) bool {
	if len(h.UpdateFields) == 0 {
		return true
	}
	for _, f := range h.UpdateFields {
		if f == k {
			return true
		}
	}
	return false
}

func (h *Handler[T]) defaultFilterCond(c *gin.Context) []builder.Cond {
	var conds []builder.Cond
	for fieldName, ctxKey := range h.Filter {
		if val, ok := c.Get(ctxKey); ok {
			conds = append(conds, builder.Eq{fieldName: val})
		}
	}
	return conds
}

func (h *Handler[T]) setDefaultFilterValues(obj interface{}, c *gin.Context) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	for fieldName, ctxKey := range h.Filter {
		if ctxVal, ok := c.Get(ctxKey); ok {
			field := val.FieldByNameFunc(func(name string) bool {
				return tagName(obj, name) == fieldName
			})
			if field.IsValid() && field.CanSet() {
				setFieldValue(field, ctxVal)
			}
		}
	}
}

func tagName(obj interface{}, fieldName string) string {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	f, ok := val.Type().FieldByName(fieldName)
	if !ok {
		return fieldName
	}
	return f.Tag.Get("json")
}

func (h *Handler[T]) parseID(raw string) (interface{}, bool) {
	if h.IsInt64ID {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, false
		}
		return n, true
	}
	return raw, true
}

func isZero(obj interface{}) bool {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val.IsZero()
}

func setID(obj interface{}) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	idField := val.FieldByName("ID")
	if idField.IsValid() && idField.Kind() == reflect.String && idField.String() == "" {
		idField.SetString(db.NewID())
	}
}

func setCreated(obj interface{}) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	createdField := val.FieldByName("Created")
	if createdField.IsValid() && createdField.Type() == reflect.TypeOf(time.Time{}) {
		createdField.Set(reflect.ValueOf(time.Now()))
	}
}

func setFieldValue(fieldVal reflect.Value, val interface{}) {
	switch fieldVal.Type().Kind() {
	case reflect.String:
		if s, ok := val.(string); ok {
			fieldVal.SetString(s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := val.(type) {
		case int:
			fieldVal.SetInt(int64(v))
		case int64:
			fieldVal.SetInt(v)
		case string:
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				fieldVal.SetInt(n)
			}
		}
	case reflect.Bool:
		if b, ok := val.(bool); ok {
			fieldVal.SetBool(b)
		}
	}
}