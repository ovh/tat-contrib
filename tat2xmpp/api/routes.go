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

		s := http.StatusOK
		if nbTatErrors > 10 {
			log.Errorf("nbXMPPErrors more than 10")
			s = http.StatusInternalServerError
		}
		ctx.JSON(s, gin.H{
			"started":                tatbot.creation,
			"nbXMPPErrors":           nbXMPPErrors,
			"nbXMPPErrorsAfterRetry": nbXMPPErrorsAfterRetry,
			"nbXMPPSent":             nbXMPPSent,
			"nbXMPPAnswers":          nbXMPPAnswers,
			"nbTatErrors":            nbTatErrors,
			"nbTatSent":              nbTatSent,
			"nbRenew":                nbRenew,
		})
	})
}
