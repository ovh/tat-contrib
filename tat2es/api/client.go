package main

import (
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/mattbaird/elastigo/lib"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var instance *tat.Client

type esConn struct {
	*elastigo.Conn
	pause int
	index string
}

// getClient initializes client on tat engine
func getClient() *tat.Client {
	if instance != nil {
		return instance
	}

	tc, err := tat.NewClient(tat.Options{
		URL:      viper.GetString("url_tat_engine"),
		Username: viper.GetString("username_tat_engine"),
		Password: viper.GetString("password_tat_engine"),
		Referer:  "tat2es.v." + Version,
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

// getClientsES initializes ES clients
func getClientsES() ([]esConn, error) {
	var esConns []esConn

	protocols := strings.Split(viper.GetString("protocol_es"), ",")
	hosts := strings.Split(viper.GetString("host_es"), ",")
	ports := strings.Split(viper.GetString("port_es"), ",")
	users := strings.Split(viper.GetString("user_es"), ",")
	passwords := strings.Split(viper.GetString("password_es"), ",")
	indices := strings.Split(viper.GetString("force_index_es"), ",")
	pauses := strings.Split(viper.GetString("pause_es"), ",")

	for i, host := range hosts {
		c := elastigo.NewConn()
		c.Domain = host
		c.Protocol = getStringValue(protocols, i)
		c.Port = getStringValue(ports, i)
		c.Username = getStringValue(users, i)
		c.Password = getStringValue(passwords, i)
		pause, err := getIntValue(pauses, i)
		if err != nil {
			return nil, err
		}
		esConns = append(esConns, esConn{Conn: c, pause: pause, index: getStringValue(indices, i)})
	}
	return esConns, nil
}

func getStringValue(array []string, index int) string {
	if len(array) == 0 {
		return ""
	}
	if index >= len(array) {
		return array[len(array)-1]
	}
	return array[index]
}

func getIntValue(array []string, index int) (int, error) {
	if len(array) == 0 {
		return 0, nil
	}
	if index >= len(array) {
		return strconv.Atoi(array[len(array)-1])
	}
	return strconv.Atoi(array[index])
}
