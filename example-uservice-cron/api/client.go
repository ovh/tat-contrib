package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var instance *tat.Client

// getClient initializes client on tat engine
func getClient() *tat.Client {
	if instance != nil {
		return instance
	}

	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url_tat_engine"),
		Username: viper.GetString("username_tat_engine"),
		Password: viper.GetString("password_tat_engine"),
		Referer:  "tatexampled.v." + Version,
	})

	if err != nil {
		log.Errorf("Error while create new Tat Client:%s", err)
	}

	if viper.GetBool("production") {
		tat.DebugLogFunc = log.Warnf
	} else {
		tat.DebugLogFunc = log.Debugf
	}

	instance = tc
	return instance
}
