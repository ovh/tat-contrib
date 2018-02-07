package main

import (
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
	"github.com/yesnault/go-imap/imap"
)

type singleton struct {
	doingCheck     bool
	imapClient     *imap.Client
	tatClient      *tat.Client
	allowedDomains []string
}

var instance *singleton
var once sync.Once

// initInstance initializes TatClient
func initInstance() {
	once.Do(func() {
		allowedDomains := []string{}
		if viper.GetString("allowed_domains") != "" {
			allowedDomains = strings.Split(viper.GetString("allowed_domains"), ",")
		}
		instance = &singleton{tatClient: getTatClient(), allowedDomains: allowedDomains}
	})
}

func (instance *singleton) do() error {
	return instance.checkMail("INBOX")
}

func getTatClient() *tat.Client {
	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url_tat_engine"),
		Username: viper.GetString("username_tat_engine"),
		Password: viper.GetString("password_tat_engine"),
		Referer:  Version,
	})

	if err != nil {
		log.Fatalf("Error while create new Tat Client: %s", err)
	}

	tat.DebugLogFunc = log.Debugf
	return tc
}
