package db

import (
	"xorm.io/builder"
)

// Count 统计表记录数
func Count(collection string, cond builder.Cond) (int64, error) {
	sess := Engine.Table(collection)
	if cond != nil {
		sess = sess.Where(cond)
	}
	return sess.Count()
}

// FindOne 查询单条记录，返回是否存在
func FindOne(collection string, cond builder.Cond, bean interface{}) (bool, error) {
	sess := Engine.Table(collection)
	if cond != nil {
		sess = sess.Where(cond)
	}
	return sess.Get(bean)
}

// Find 查询多条记录到切片指针
func Find(collection string, cond builder.Cond, beans interface{}, limit, offset int) error {
	sess := Engine.Table(collection)
	if cond != nil {
		sess = sess.Where(cond)
	}
	if limit > 0 {
		sess = sess.Limit(limit, offset)
	}
	return sess.Find(beans)
}

// PagedFind 分页查询到切片指针
func PagedFind(collection string, cond builder.Cond, page, pageSize int, orderBy string, orderAsc bool, beans interface{}) error {
	sess := Engine.Table(collection)
	if cond != nil {
		sess = sess.Where(cond)
	}
	dir := " desc"
	if orderAsc {
		dir = " asc"
	}
	sess = sess.OrderBy(orderBy + dir)
	if pageSize > 0 {
		sess = sess.Limit(pageSize, (page-1)*pageSize)
	}
	return sess.Find(beans)
}

// InsertOne 插入一条记录
func InsertOne(collection string, bean interface{}) (int64, error) {
	return Engine.Table(collection).Insert(bean)
}

// UpdateOne 按条件更新字段
func UpdateOne(collection string, cond builder.Cond, fields map[string]interface{}) (int64, error) {
	sess := Engine.Table(collection)
	if cond != nil {
		sess = sess.Where(cond)
	}
	return sess.Update(fields)
}

// DeleteOne 按条件删除记录
func DeleteOne(collection string, cond builder.Cond) (int64, error) {
	sess := Engine.Table(collection)
	if cond != nil {
		sess = sess.Where(cond)
	}
	return sess.Delete()
}