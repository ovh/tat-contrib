package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// GetHeader returns header value from request
func GetHeader(ctx *gin.Context, headerName string) string {
	h := strings.ToLower(headerName)
	hd := strings.ToLower(strings.Replace(headerName, "_", "-", -1))
	for k, v := range ctx.Request.Header {
		if strings.ToLower(k) == h {
			return v[0]
		} else if strings.ToLower(k) == hd {
			return v[0]
		}
	}
	return ""
}
