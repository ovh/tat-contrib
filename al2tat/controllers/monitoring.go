package controllers

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat-contrib/al2tat/models"
	"github.com/ovh/tat-contrib/al2tat/utils"
)

// MonitoringController contains all methods about monitoring
type MonitoringController struct{}

// CreateSync creates a new monitoring, synchronous
func (m *MonitoringController) CreateSync(ctx *gin.Context) {
	m.create(ctx, true)
}

// CreateAsync creates a new monitoring, async
func (m *MonitoringController) CreateAsync(ctx *gin.Context) {
	m.create(ctx, false)
}

// Create creates a new monitoring
func (*MonitoringController) create(ctx *gin.Context, sync bool) {
	var monitoring models.Monitoring
	ctx.BindJSON(&monitoring)

	log.Infof("Call Post to tat Engine for %+v", monitoring)

	topic := utils.GetHeader(ctx, utils.TatTopicHeader)
	if topic == "" {
		ctx.JSON(http.StatusBadRequest, "Invalid Topic")
		return
	}

	monitoring.TatUsername = utils.GetHeader(ctx, utils.TatUsernameHeader)
	monitoring.TatPassword = utils.GetHeader(ctx, utils.TatPasswordHeader)
	monitoring.Topic = topic

	if sync {
		msg, err := monitoring.PostToTatEngine()
		if err != nil {
			ctx.JSON(http.StatusCreated, gin.H{"error": err.Error()})
		} else {
			ctx.JSON(http.StatusCreated, gin.H{"message": msg})
		}
	} else {
		go monitoring.PostToTatEngine()
		ctx.JSON(http.StatusCreated, "Request received, send async to tat-engine")
	}

}
