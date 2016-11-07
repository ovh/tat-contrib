package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat-contrib/al2tat/controllers"
)

// InitRoutesAlerts initialized routes for Alerts Controller
func InitRoutesAlerts(router *gin.Engine) {

	AlertsCtrl := &controllers.AlertsController{}
	g := router.Group("/")
	g.POST("/alert", AlertsCtrl.CreateAsync)
	g.POST("/alert/sync", AlertsCtrl.CreateSync)
	g.POST("/alarm", AlertsCtrl.CreateAsync)
	g.POST("/alarm/sync", AlertsCtrl.CreateSync)
	g.PUT("/purge/:skip/:limit", AlertsCtrl.Purge)
}
