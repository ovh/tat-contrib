package main

import (
	log "github.com/Sirupsen/logrus"
	elastigo "github.com/mattbaird/elastigo/lib"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var instance *tat.Client
var esConn *elastigo.Conn

// getClient initializes client on tat engine
func getClient() *tat.Client {
	if instance != nil {
		return instance
	}

	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url_tat_engine"),
		Username: viper.GetString("username_tat_engine"),
		Password: viper.GetString("password_tat_engine"),
		Referer:  "tat2es.v." + VERSION,
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

// getClient initializes client on tat engine
func getClientES() *elastigo.Conn {
	if esConn != nil {
		return esConn
	}
	esConn = elastigo.NewConn()
	esConn.Domain = viper.GetString("host_es")
	esConn.Port = viper.GetString("port_es")
	esConn.Username = viper.GetString("user_es")
	esConn.Password = viper.GetString("password_es")

	return esConn
}
