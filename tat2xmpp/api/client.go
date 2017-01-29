package main

import (
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-xmpp"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

const headerXRemoteUser = "X-Remote-User"

type botClient struct {
	creation                  time.Time
	XMPPClient                *xmpp.Client
	TatClient                 *tat.Client
	nbXMPPErrors              int
	nbXMPPErrorsAfterRetry    int
	nbXMPPSent                int
	nbTatErrors               int
	nbTatSent                 int
	nbXMPPAnswers             int
	nbRenew                   int
	nbTopicConfs              int
	nbRequestsWithAlias       int
	nbRequestsWithAliasErrors int
	nbRequestsCountTat        int
	nbRequestsGetTat          int
	nbRequestsCountTatErrors  int
	nbRequestsGetTatErrors    int
	chats                     chan xmpp.Chat
	aliases                   []tat.Message
	admins                    []string
}

func getBotClient(username, password string) (*botClient, error) {

	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url_tat_engine"),
		Username: username,
		Password: password,
		Referer:  "tat2xmpp.v." + VERSION,
	})

	if err != nil {
		log.Errorf("Error while create new Tat Client:%s", err)
	}

	tat.DebugLogFunc = log.Debugf

	xClient, err := getNewXMPPClient()
	if err != nil {
		log.Errorf("getClient >> error with getNewXMPPClient err:%s", err)
		return nil, err
	}

	admins = strings.Split(viper.GetString("admin_tat2xmpp"), ",")
	log.Infof("admin configured:%+v", admins)

	instance := &botClient{
		TatClient:  tc,
		XMPPClient: xClient,
		admins:     admins,
	}

	return instance, nil
}

func readConfigFile() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
		viper.ReadInConfig() // Find and read the config file
	}
}
