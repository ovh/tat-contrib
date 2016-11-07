package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat-contrib/al2tat/utils"
	"github.com/ovh/tat/models"
)

// Alert struct
type Alert struct {
	Alert       string         `json:"alert"`
	NbAlert     int64          `json:"nbAlert"`
	Service     string         `json:"service"`
	Summary     string         `json:"summary"`
	IDReference string         `json:"idReference"`
	Action      string         `json:"action"`
	Labels      []models.Label `json:"labels,omitempty"`
}

// PostToTatEngine an alarm
func (alarm *Alert) PostToTatEngine(tatUsername, tatPassword, topic string) (models.Message, error) {

	alarm.fixAlertReceived()
	alarm.computeReplay(topic, tatUsername, tatPassword)
	alarm.computeLabel()

	text := fmt.Sprintf("#%s #Nb:%d #%s %s", alarm.Alert, alarm.NbAlert, alarm.Service, alarm.Summary)
	m := messageJSON{Text: text, Action: alarm.Action, IDReference: alarm.IDReference, Labels: alarm.Labels}

	jsonStr, err := json.Marshal(m)
	if err != nil {
		return models.Message{}, err
	}
	b, err := utils.PostWant("/message"+topic, jsonStr, tatUsername, tatPassword)
	if err != nil {
		return models.Message{}, err
	}
	var created messageJSONOut
	json.Unmarshal(b, &created)

	return created.Message, nil
}

func (alarm *Alert) fixAlertReceived() {

	if alarm.Alert == "" {
		alarm.Alert = "AL"
	}

	alert := alarm.Alert + strconv.FormatInt(alarm.NbAlert, 10)

	r, _ := regexp.Compile("^(AL|UP)([0-9]*)$")
	if r.MatchString(alert) {
		matches := r.FindStringSubmatch(alert)
		if len(matches[1]) > 0 {
			alarm.Alert = matches[1]
		}
		if len(matches[2]) > 0 {
			nb, err := strconv.Atoi(matches[2])
			if err != nil {
				log.Errorf("Error while converting NbAlert to int %s", err)
				alarm.NbAlert = 1
			} else {
				alarm.NbAlert = int64(nb)
			}
		}
	}

	if alarm.NbAlert <= 0 {
		alarm.NbAlert = 1
	}
}

func (alarm *Alert) computeLabel() {
	if alarm.Alert == "AL" && alarm.IDReference == "" {
		alarm.Labels = append(alarm.Labels, models.Label{Text: "open", Color: red})
		log.Debugf("New AL root msg, add label open")
	} else if alarm.Alert == "UP" && alarm.IDReference == "" {
		alarm.Labels = append(alarm.Labels, models.Label{Text: "done", Color: green})
		log.Debugf("New UP root msg, add label done")
	}
}

func (alarm *Alert) computeReplay(topic, tatUsername, tatPassword string) {
	summaryToSearch := alarm.getSummaryFixed()
	duration := time.Minute * 65
	lastHour := strconv.FormatInt(time.Now().Add(-duration).Unix(), 10)
	now := strconv.FormatInt(time.Now().Unix(), 10)
	alarms, err := messagesList(tatUsername, tatPassword, topic, summaryToSearch, alarm.Service, "", lastHour, now)

	if err != nil {
		log.Errorf("Error while fetching alarm for computeReplay :%s", err.Error())
		return
	}

	if len(alarms.Messages) == 0 {
		log.Debugf("No previous Al found for replay")
		return
	}

	rootMsg := alarms.Messages[0]

	// alarms are sorted, recents at first
	// if last al with same summary is UP, return
	if utils.ArrayContains(rootMsg.Tags, "UP") {
		log.Debugf("UP found, no replay")
		return
	}

	isOpen := false
	labelDoing := false
	labelOpen := false
	log.Debugf("RootMsg : %+v", rootMsg)
	for _, l := range rootMsg.Labels {
		if l.Text == "doing" || l.Text == "open" {
			isOpen = true
		}
		if l.Text == "open" {
			labelOpen = true
		}
		if l.Text == "doing" {
			labelDoing = true
		}
	}

	if alarm.Alert == "AL" && !isOpen {
		log.Debugf("No replay, previous msg is not open or doing")
		return
	}

	alarm.IDReference = rootMsg.ID
	alarm.Action = "reply"
	alarm.Summary = fmt.Sprintf("#replay %s", alarm.Summary)

	if alarm.Alert == "UP" {
		log.Debugf("It's a UP msg, set thread to done")
		alarm.addLabel(topic, tatUsername, tatPassword, rootMsg.ID, "done")
		if labelOpen {
			removeLabel(topic, tatUsername, tatPassword, rootMsg.ID, "open")
		}
		if labelDoing {
			removeLabel(topic, tatUsername, tatPassword, rootMsg.ID, "doing")
		}
	}

	log.Debugf("Reply computed, attached to %s", alarm.IDReference)
	purgeReplies(rootMsg, tatUsername, tatPassword)
}

// Replies type to sort them
type Replies []models.Message

func (slice Replies) Len() int {
	return len(slice)
}

func (slice Replies) Less(i, j int) bool {
	return slice[i].DateUpdate < slice[j].DateUpdate
}

func (slice Replies) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// AlertPurge apply purge compute on a topic
func AlertPurge(skip, limit int, tatUsername, tatPassword, topic string) error {
	var messagesRoot messagesJSON
	msgs, err := utils.GetWantBody(fmt.Sprintf("/messages%s?skip=%d&limit=%d&onlyMsgRoot=true", topic, skip, limit), tatUsername, tatPassword)
	if err != nil {
		log.Errorf("Error with getting list root err:%s", err.Error())
		return err
	}

	if err := json.Unmarshal(msgs, &messagesRoot); err != nil {
		log.Errorf("Error with Unmarshal list root err:%s", err.Error())
		return err
	}

	for _, m := range messagesRoot.Messages {
		var messages messagesJSON
		msgs, err := utils.GetWantBody(fmt.Sprintf("/messages%s?skip=%d&limit=%d&treeView=onetree&idMessage=%s", topic, 0, 1, m.ID), tatUsername, tatPassword)
		if err != nil {
			log.Errorf("Error with getting list replies for rootMsg:%s err:%s", m.ID, err.Error())
			continue
		}
		err = json.Unmarshal(msgs, &messages)
		if err != nil {
			log.Errorf("Error with Unmarshal list replies for rootMsg:%s err:%s", m.ID, err.Error())
			continue
		}
		purgeReplies(messages.Messages[0], tatUsername, tatPassword)
	}
	return nil
}

// purgeReplies purges 30 older replies of an AL
func purgeReplies(rootMsg models.Message, tatUsername, tatPassword string) {
	var replies Replies
	replies = append(replies, rootMsg.Replies...)
	sort.Sort(sort.Reverse(replies))

	if len(replies) > 29 {
		for _, r := range replies[30:] {
			_, err := utils.DeleteWant("/message/"+r.ID, nil, tatUsername, tatPassword)
			if err != nil {
				log.Errorf("purgeReplies : error while delete msgId: %s, err:%s", r.ID, err.Error())
			}
		}
	}
}

func (alarm *Alert) addLabel(topic, tatUsername, tatPassword, idMessage, label string) {
	color := ""
	switch label {
	case "done":
		color = green
	case "open":
		color = red
	}
	l := models.Label{Text: label, Color: color}
	writeLabel(topic, tatUsername, tatPassword, idMessage, l)
}

// remove date at the end of summary
// if alerts comes from tatsamon, get only sailabove registry, account and service
func (alarm *Alert) getSummaryFixed() string {
	if strings.Contains(alarm.Summary, "#tatmon") {
		//#tatmon #sailabove:sailabove.io/v1 #account:yourAccount #service:kibana-es #host:kibana-es-1 #state:stopped err:Get http://kibana
		r := regexp.MustCompile("^(.*)(#tatmon)\\s(#sailabove:.*)\\s(#account:.*)\\s(#service:[a-zA-Z0-9\\-\\.]+)\\s(.*)$")
		if r.MatchString(alarm.Summary) {
			s := r.FindStringSubmatch(alarm.Summary)
			t := s[3] + " " + s[4] + " " + s[5]
			return t
		}
	}

	r := regexp.MustCompile("^(.*)(\\(.*..:..:..\\))$")
	if r.MatchString(alarm.Summary) {
		s := r.FindStringSubmatch(alarm.Summary)[1]
		log.Debugf("Computed summary : %s", s)
		return s
	}

	log.Debugf("Computed summary : %s", alarm.Summary)
	return alarm.Summary
}
