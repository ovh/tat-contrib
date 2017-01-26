package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

func initRoutes(router *gin.Engine) {
	router.POST("/hook", hookJSON)
	router.GET("/version", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"version": VERSION})
	})
	router.GET("/mon/status", func(ctx *gin.Context) {
		log.Infof("started:%s errors:%d ok:%d", tatbot.creation, nbErrors, nbOK)
		ctx.JSON(http.StatusOK, gin.H{
			"started":  tatbot.creation,
			"nbErrors": nbErrors,
			"nbOK":     nbOK,
		})
	})
}
