package api

import (
	"github.com/gin-gonic/gin"

	"gateway-manager/pkg/curd"
)

// RequirePermission 校验权限的中间件
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasPermission(c, perm) {
			c.Next()
			return
		}
		curd.ErrorWithStatus(c, 403, 403, "无权限: "+perm)
		c.Abort()
	}
}

// RequireAny 满足任意一个权限即可
func RequireAny(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		owned := c.GetStringSlice("permissions")
		for _, p := range perms {
			if ownedContains(owned, p) {
				c.Next()
				return
			}
		}
		curd.ErrorWithStatus(c, 403, 403, "无权限")
		c.Abort()
	}
}

func ownedContains(owned []string, want string) bool {
	for _, p := range owned {
		if p == "*" || p == want {
			return true
		}
	}
	return false
}