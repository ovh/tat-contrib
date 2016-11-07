package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat-contrib/al2tat/utils"
	"github.com/ovh/tat/models"
)

// Monitoring struct
type Monitoring struct {
	Status      string          `json:"status"` // UP, AL
	Service     string          `json:"service"`
	Item        string          `json:"item"`
	Summary     string          `json:"summary"`
	Labels      []models.Label  `json:"labels"`
	TatUsername string          `json:"-"`
	TatPassword string          `json:"-"`
	Topic       string          `json:"-"`
	TatItem     *models.Message `json:"-"`
}

// PostToTatEngine an monitoring
func (monitoring *Monitoring) PostToTatEngine() (*models.Message, error) {

	// check if item exists
	err := monitoring.computeItem()
	if err != nil {
		return &models.Message{}, err
	}

	if monitoring.TatItem.ID == "" {
		log.Errorf("Invalid Created Item")
		return &models.Message{}, fmt.Errorf("Invalid Created Item")
	}

	// update label to item (root msg)
	monitoring.computeItemLabels()

	// add reply to item (root msg)
	monitoring.computeReplies()

	monitoring.purgeReplies()

	return monitoring.TatItem, nil
}

func (monitoring *Monitoring) computeItem() error {

	var items messagesJSON
	msgs, err := utils.GetWantBody(fmt.Sprintf("/messages%s?skip=0&limit=1&andTag=%s&treeView=onetree", monitoring.Topic, "monitoring,item:"+monitoring.Item), monitoring.TatUsername, monitoring.TatPassword)
	if err != nil {
		return err
	}
	err = json.Unmarshal(msgs, &items)
	if err != nil {
		log.Errorf("Error while fetching items from Tat: %s", err.Error())
		return err
	}

	if len(items.Messages) == 0 {
		return monitoring.createItem()
	}

	// Take first
	monitoring.TatItem = &items.Messages[0]
	return nil
}

func (monitoring *Monitoring) createItem() error {

	text := fmt.Sprintf("#monitoring #%s #item:%s", monitoring.Service, monitoring.Item)
	m := messageJSON{Text: text, Labels: monitoring.Labels}

	jsonStr, err := json.Marshal(m)
	if err != nil {
		return err
	}
	b, err := utils.PostWant("/message"+monitoring.Topic, jsonStr, monitoring.TatUsername, monitoring.TatPassword)
	if err != nil {
		return err
	}

	var created messageJSONOut
	json.Unmarshal(b, &created)
	monitoring.TatItem = &created.Message
	return nil
}

func (monitoring *Monitoring) computeReplies() error {

	m := messageJSON{
		Text:   fmt.Sprintf("#monitoring #item:%s %s", monitoring.Item, monitoring.Summary),
		Action: "reply", IDReference: monitoring.TatItem.ID,
	}

	jsonStr, err := json.Marshal(m)
	if err != nil {
		return err
	}
	b, err := utils.PostWant("/message"+monitoring.Topic, jsonStr, monitoring.TatUsername, monitoring.TatPassword)
	if err != nil {
		return err
	}
	var created messageJSONOut
	return json.Unmarshal(b, &created)
}

// purgeReplies keeps only 30 older replies if replies > 2 days.
func (monitoring *Monitoring) purgeReplies() {

	var replies Replies
	replies = append(replies, monitoring.TatItem.Replies...)
	sort.Sort(sort.Reverse(replies))

	lastTwoDays := float64(time.Now().AddDate(0, 0, -2).UnixNano()) / 1000000000

	if len(replies) > 29 {
		for _, r := range replies[30:] {
			if r.DateUpdate < lastTwoDays {
				_, err := utils.DeleteWant("/message/"+r.ID, nil, monitoring.TatUsername, monitoring.TatPassword)
				if err != nil {
					log.Errorf("purgeReplies: error while delete msgId: %s, err:%s", r.ID, err.Error())
				}
			}
		}
	}
}

func (monitoring *Monitoring) computeItemLabels() {
	containsUP := false
	containsAL := false

	for _, l := range monitoring.TatItem.Labels {
		if l.Text == "UP" {
			containsUP = true
		}
		if l.Text == "AL" {
			containsAL = true
		}
	}

	if monitoring.Status == "AL" || monitoring.Status == "" {
		log.Debugf("Add label AL on %s", monitoring.TatItem.ID)
		l := models.Label{Text: "AL", Color: red}
		if containsUP {
			removeLabel(monitoring.Topic, monitoring.TatUsername, monitoring.TatPassword, monitoring.TatItem.ID, "UP")
		}
		if !containsAL {
			writeLabel(monitoring.Topic, monitoring.TatUsername, monitoring.TatPassword, monitoring.TatItem.ID, l)
		}
	} else if monitoring.Status == "UP" {
		log.Debugf("Add label UP on %s", monitoring.TatItem.ID)
		l := models.Label{Text: "UP", Color: green}
		if containsAL {
			removeLabel(monitoring.Topic, monitoring.TatUsername, monitoring.TatPassword, monitoring.TatItem.ID, "AL")
		}
		if !containsUP {
			writeLabel(monitoring.Topic, monitoring.TatUsername, monitoring.TatPassword, monitoring.TatItem.ID, l)
		}
	}
}
