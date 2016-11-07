package main

import (
	log "github.com/Sirupsen/logrus"
	cron "gopkg.in/robfig/cron.v2"
)

var crontab *cron.Cron

func startCron() {
	log.Infof("Starting cron")
	crontab.Start()
}

func stopCron() {
	log.Infof("Stopping cron")
	crontab.Stop()
	log.Infof("Cron stopped")
}

// initAndStartCron inits and starts cron...
func initAndStartCron() {
	log.Infof("Bootstrap Cron")
	crontab = cron.New()
	initCron()
	startCron()
}

func initCron() {
	// each two hours
	crontab.AddFunc("0 */2 * * * *", func() {
		do()
	})
}
