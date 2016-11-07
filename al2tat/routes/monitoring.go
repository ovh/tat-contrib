package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat-contrib/al2tat/controllers"
)

// InitRoutesMonitoring initialized routes for Monitoring Controller
func InitRoutesMonitoring(router *gin.Engine) {
	monitoringCtrl := &controllers.MonitoringController{}
	g := router.Group("/")
	g.POST("/monitoring", monitoringCtrl.CreateAsync)
	g.POST("/monitoring/sync", monitoringCtrl.CreateSync)
}
