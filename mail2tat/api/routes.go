package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

// initRoutes initialized routes
func initRoutes(router *gin.Engine) {
	router.GET("/do", func(ctx *gin.Context) {
		if err := instance.do(); err != nil {
			log.Errorf("Err:%s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"result": "error, please check logs"})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"result": "success"})
	})
	router.GET("/version", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"version": Version})
	})
}
