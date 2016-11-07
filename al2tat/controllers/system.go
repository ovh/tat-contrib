package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// VERSION is version of Al2Tat.
const VERSION = "0.23.0"

// SystemController contains all methods about version
type SystemController struct{}

//GetVersion returns version of tat
func (*SystemController) GetVersion(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"version": VERSION})
}
