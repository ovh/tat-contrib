package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

var sslInsecureSkipVerify bool
var url, username, password string

// Tat_username header
var TatUsernameHeader = "Tat_username"

// Tat_password header
var TatPasswordHeader = "Tat_password"

// Tat_topic header
var TatTopicHeader = "Tat_topic"

func initRequest(req *http.Request, tatUsername, tatPassword string) {
	req.Header.Set(TatUsernameHeader, tatUsername)
	req.Header.Set(TatPasswordHeader, tatPassword)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")
}

// GetWantBody GET on tat engine, checks if http code is equals to 200 and returns body
func GetWantBody(path, tatUsername, tatPassword string) ([]byte, error) {
	return reqWant("GET", http.StatusOK, path, nil, tatUsername, tatPassword, 1)
}

// PutWant updates a message on tat engine, checks if http code is equals to 201
func PutWant(path string, jsonStr []byte, tatUsername, tatPassword string) error {
	_, err := reqWant("PUT", http.StatusCreated, path, jsonStr, tatUsername, tatPassword, 1)
	return err
}

// PostWant post a message to tat engine, checks if http code is equals to 201 and returns body
func PostWant(path string, jsonStr []byte, tatUsername, tatPassword string) ([]byte, error) {
	return reqWant("POST", http.StatusCreated, path, jsonStr, tatUsername, tatPassword, 1)
}

// DeleteWant deletes a message on tat engine, checks if http code is equals to 200 and returns body
func DeleteWant(path string, jsonStr []byte, tatUsername, tatPassword string) ([]byte, error) {
	return reqWant("DELETE", http.StatusOK, path, jsonStr, tatUsername, tatPassword, 1)
}

func isHTTPS() bool {
	if strings.HasPrefix(viper.GetString("url_tat_engine"), "https") {
		return true
	}
	return false
}

func getHTTPClient() *http.Client {
	var tr *http.Transport
	if isHTTPS() {
		tlsConfig := getTLSConfig()
		tr = &http.Transport{TLSClientConfig: tlsConfig}
	} else {
		tr = &http.Transport{}
	}

	return &http.Client{Transport: tr}
}

func getTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: sslInsecureSkipVerify,
	}
}

func reqWant(method string, wantCode int, path string, jsonStr []byte, tatUsername, tatPassword string, ntry int) ([]byte, error) {

	if viper.GetString("url_tat_engine") == "" {
		fmt.Println("Invalid Configuration : invalid URL. See al2tat --help")
		os.Exit(1)
	}

	requestPath := viper.GetString("url_tat_engine") + path
	var req *http.Request
	var err error
	if jsonStr != nil {
		req, err = http.NewRequest(method, requestPath, bytes.NewReader(jsonStr))
	} else {
		req, err = http.NewRequest(method, requestPath, nil)
	}

	if err != nil {
		e := fmt.Sprintf("Error with http.NewRequest %s", err.Error())
		log.Errorf(e)
		return []byte{}, fmt.Errorf(e)
	}

	initRequest(req, tatUsername, tatPassword)
	resp, err := getHTTPClient().Do(req)

	if err != nil {
		log.Errorf("Error with getHTTPClient %s", err.Error())

		// 5 tentatives max
		if ntry > 5 {
			log.Errorf("Error with getHTTPClient, try %d KO, return", ntry)
			return []byte{}, fmt.Errorf("Error with getHTTPClient %s", err.Error())
		}
		log.Errorf("Error with getHTTPClient %s, it's try %d, new try", err.Error(), ntry)
		time.Sleep(3 * time.Second)
		return reqWant(method, wantCode, path, jsonStr, tatUsername, tatPassword, ntry+1)
	}

	defer resp.Body.Close()

	if resp.StatusCode != wantCode {
		log.Error(fmt.Sprintf("Response Status:%s", resp.Status))
		log.Error(fmt.Sprintf("Request path :%s", requestPath))
		log.Error(fmt.Sprintf("Request :%s", string(jsonStr)))
		log.Error(fmt.Sprintf("Response Headers:%s", resp.Header))
		body, _ := ioutil.ReadAll(resp.Body)
		log.Error(fmt.Sprintf("Response Body:%s", string(body)))
		return []byte{}, fmt.Errorf("Response code %d with Body:%s", resp.StatusCode, string(body))
	}
	log.Debugf("%s %s", method, requestPath)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error with ioutil.ReadAll %s", err.Error())
		return []byte{}, fmt.Errorf("Error with ioutil.ReadAll %s", err.Error())
	}
	return body, nil
}
