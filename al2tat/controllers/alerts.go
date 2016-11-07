package controllers

import (
	"errors"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tat-contrib/al2tat/models"
	"github.com/ovh/tat-contrib/al2tat/utils"
)

// AlertsController contains all methods about alarms manipulation
type AlertsController struct{}

// CreateSync creates a new alarm, synchrone
func (a *AlertsController) CreateSync(ctx *gin.Context) {
	a.create(ctx, true)
}

// CreateAsync creates a new alarm, asynchrone
func (a *AlertsController) CreateAsync(ctx *gin.Context) {
	a.create(ctx, false)
}

func (*AlertsController) create(ctx *gin.Context, sync bool) {
	var alarm models.Alert
	ctx.BindJSON(&alarm)

	log.Infof("Call Post to tat Engine for %+v", alarm)
	topic := utils.GetHeader(ctx, utils.TatTopicHeader)
	if topic == "" {
		topic = "/Internal/Alerts"
	}

	if sync {
		msg, err := alarm.PostToTatEngine(utils.GetHeader(ctx, utils.TatUsernameHeader), utils.GetHeader(ctx, utils.TatPasswordHeader), topic)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		} else {
			ctx.JSON(http.StatusCreated, gin.H{"message": msg})
		}
	} else {
		go alarm.PostToTatEngine(utils.GetHeader(ctx, utils.TatUsernameHeader), utils.GetHeader(ctx, utils.TatPasswordHeader), topic)
		ctx.JSON(http.StatusCreated, "Request received, send async to tat-engine")
	}
}

// getParamInt returns the value of a parameter in Url.
// Example : http://host:port/:paramName
func getParamInt(ctx *gin.Context, paramName string) (int, error) {
	value, found := ctx.Params.Get(paramName)
	if !found {
		s := paramName + " in url does not exist"
		ctx.JSON(http.StatusBadRequest, gin.H{"error": s})
		return -1, errors.New(s)
	}
	valInt, err := strconv.Atoi(value)
	return valInt, err
}

// Purge purges AL on a topic
func (*AlertsController) Purge(ctx *gin.Context) {

	topic := utils.GetHeader(ctx, utils.TatTopicHeader)
	log.Infof("Call Alert Purge for %+v", topic)

	if topic == "" {
		ctx.JSON(http.StatusBadRequest, "Invalid Topic")
		return
	}

	skip, err := getParamInt(ctx, "skip")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skip in query"})
		return
	}

	limit, err := getParamInt(ctx, "limit")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit in query"})
		return
	}

	models.AlertPurge(skip, limit, utils.GetHeader(ctx, utils.TatUsernameHeader), utils.GetHeader(ctx, utils.TatPasswordHeader), topic)
	ctx.JSON(http.StatusCreated, "Purge on "+topic+" done")
}
