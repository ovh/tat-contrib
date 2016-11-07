package main

import (
	log "github.com/Sirupsen/logrus"
	cron "gopkg.in/robfig/cron.v2"
)

var mail2tatCron *cron.Cron

func startCron() {
	log.Infof("Starting cron")
	mail2tatCron.Start()
}

func initCron() {
	mail2tatCron.AddFunc("0 * * * * *", func() {
		instance.do()
	})
}

// initAndStartCron inits and starts cron...
func initAndStartCron() {
	log.Infof("Init and start Cron")
	mail2tatCron = cron.New()
	initCron()
	startCron()
}
