package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var instance *tat.Client

func getClient() *tat.Client {
	if instance != nil {
		return instance
	}

	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url_tat_engine"),
		Username: viper.GetString("username_tat_engine"),
		Password: viper.GetString("password_tat_engine"),
		Referer:  "tatdashing.v." + Version,
	})

	if err != nil {
		log.Errorf("Error while create new Tat Client:%s", err)
	}

	tat.DebugLogFunc = log.Debugf
	instance = tc
	return instance
}
