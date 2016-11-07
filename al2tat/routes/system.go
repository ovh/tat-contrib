package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat-contrib/al2tat/controllers"
)

// InitRoutesSystem initialized routes for System Controller
func InitRoutesSystem(router *gin.Engine) {
	systemCtrl := &controllers.SystemController{}
	router.GET("/version", systemCtrl.GetVersion)
}
