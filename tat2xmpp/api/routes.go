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
	router.GET("/mon/status", func(ctx *gin.Context) {

		s := http.StatusOK
		ctx.JSON(s, gin.H{
			"started":                tatbot.creation,
			"nbXMPPErrors":           tatbot.nbXMPPErrors,
			"nbXMPPErrorsAfterRetry": tatbot.nbXMPPErrorsAfterRetry,
			"nbXMPPSent":             tatbot.nbXMPPSent,
			"nbXMPPAnswers":          tatbot.nbXMPPAnswers,
			"nbTatErrors":            tatbot.nbTatErrors,
			"nbTatSent":              tatbot.nbTatSent,
			"nbTopicConfs":           tatbot.nbTopicConfs,
			"nbTopicConfsFilterHook": len(topicConfsFilterHook),
			"nbRequestsCountTat":     tatbot.nbRequestsCountTat,
			"nbRequestsGetTat":       tatbot.nbRequestsGetTat,
			"nbRenew":                tatbot.nbRenew,
		})
	})
}
