package models

import (
	"encoding/json"
	"fmt"
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat-contrib/al2tat/utils"
	"github.com/ovh/tat/models"
)

type messageJSON struct {
	Text        string         `json:"text"`
	IDReference string         `json:"idReference,omitempty"`
	Action      string         `json:"action,omitempty"`
	Option      string         `json:"option,omitempty"`
	Labels      []models.Label `json:"labels,omitempty"`
}

type messagesJSON struct {
	Messages  []models.Message `json:"messages"`
	IsTopicRw bool             `json:"isTopicRw"`
}

type messageJSONOut struct {
	Message models.Message `json:"message"`
	Info    string         `json:"info"`
}

func writeLabel(topic, tatUsername, tatPassword, idMessage string, label models.Label) {
	m := messageJSON{Text: label.Text, Action: "label", IDReference: idMessage, Option: label.Color}
	jsonStr, err := json.Marshal(m)
	if err != nil {
		log.Errorf("Error while addLabelStruct")
	}
	utils.PutWant("/message"+topic, jsonStr, tatUsername, tatPassword)
	log.Debugf("Add label to %s done", idMessage)
}

func removeLabel(topic, tatUsername, tatPassword, idMessage, labelToRemove string) {
	m := messageJSON{Text: labelToRemove, Action: "unlabel", IDReference: idMessage}
	jsonStr, err := json.Marshal(m)
	if err != nil {
		log.Errorf("Error while setLabelToDone")
	}
	utils.PutWant("/message"+topic, jsonStr, tatUsername, tatPassword)
	log.Debugf("Remove label %s from %s", labelToRemove, idMessage)
}

func messagesList(tatUsername, tatPassword, criteriaTopic, criteriaText, criteriaTag, criteriaLabel string, criteriaDateMinUpdate, criteriaDateMaxUpdate string) (messagesJSON, error) {
	c := ""
	if criteriaText != "" {
		c = c + "&text=" + url.QueryEscape(criteriaText)
	}
	if criteriaLabel != "" {
		c = c + "&label=" + criteriaLabel
	}
	if criteriaTag != "" {
		c = c + "&tag=" + criteriaTag
	}
	if criteriaDateMinUpdate != "" {
		c = c + "&dateMinUpdate=" + criteriaDateMinUpdate
	}
	if criteriaDateMaxUpdate != "" {
		c = c + "&dateMaxUpdate=" + criteriaDateMaxUpdate
	}
	var messages messagesJSON
	msgs, err := utils.GetWantBody(fmt.Sprintf("/messages%s?skip=%d&limit=%d%s&treeView=onetree", criteriaTopic, 0, 1, c), tatUsername, tatPassword)
	if err != nil {
		return messages, err
	}
	err = json.Unmarshal(msgs, &messages)
	return messages, err
}
