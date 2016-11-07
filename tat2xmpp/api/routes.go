package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func initRoutes(router *gin.Engine) {
	router.POST("/hook", hookJSON)
	router.GET("/version", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"version": VERSION})
	})
}
