package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Version is version of Al2Tat.
var Version string

// SystemController contains all methods about version
type SystemController struct{}

//GetVersion returns version of tat
func (*SystemController) GetVersion(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"version": Version})
}
