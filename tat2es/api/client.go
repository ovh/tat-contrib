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

// SEPARATOR is the value separator for viper parameters
const SEPARATOR = ","

type esConn struct {
	*elastigo.Conn
	pause  int
	index  string
	prefix string
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

	for i, host := range strings.Split(viper.GetString("host_es"), SEPARATOR) {
		c := elastigo.NewConn()
		c.Domain = host
		c.Protocol = getStringValue("protocol_es", i)
		c.Port = getStringValue("port_es", i)
		c.Username = getStringValue("user_es", i)
		c.Password = getStringValue("password_es", i)
		esConns = append(esConns, esConn{
			Conn:   c,
			pause:  getIntValue("pause_es", i),
			index:  getStringValue("force_index_es", i),
			prefix: getStringValue("prefix_index_es", i),
		})
	}
	return esConns, nil
}

func getStringValue(paramName string, index int) string {
	array := strings.Split(viper.GetString(paramName), ",")
	if len(array) == 0 {
		return ""
	}
	if index >= len(array) {
		log.Fatalf("missing value for %s", paramName)
	}
	return array[index]
}

func getIntValue(paramName string, index int) int {
	array := strings.Split(viper.GetString(paramName), ",")
	if len(array) == 0 {
		return 0
	}
	if index >= len(array) {
		log.Fatalf("missing value for %s", paramName)
	}
	val, err := strconv.Atoi(array[index])
	if err != nil {
		log.Fatalf("invalid number %s for param %s", array[index], paramName)
	}
	return val
}
