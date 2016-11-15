package main

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
)

type indexableData struct {
	data  []tat.Message
	index string
}

var postESChan chan *indexableData

// runner is a scheduled runner for each topic:index
type runner struct {
	topicPath string
	index     string
	timestamp int
}

//Run is called by cron
func (r *runner) Run() {
	var t = time.Now().Unix()
	work(r.topicPath, r.index, r.timestamp)
	r.timestamp = int(t)
}

func do() {
	//A chan to feed ES
	postESChan = make(chan *indexableData, 10)

	go postES()
	scheduler := cron.New()
	t := viper.GetString("topics_indexes")
	for _, arg := range strings.Split(t, ",") {
		tuple := strings.Split(arg, ":")
		if len(tuple) == 2 {
			log.Debugf("Add schedule %s for topic %s and es-index %s", viper.GetString("cron_schedule"), tuple[0], tuple[1])
			scheduler.AddJob(viper.GetString("cron_schedule"), &runner{tuple[0], tuple[1], viper.GetInt("timestamp")})
		} else {
			log.Errorf("Invalid values for --topics-indexes %s, %s", arg, tuple)
		}
	}
	scheduler.Start()
	var forever = make(chan bool, 1)
	<-forever
}

func work(topic string, index string, timestamp int) {

	countJSON, err := getClient().MessageCount(topic, &tat.MessageCriteria{DateMinUpdate: fmt.Sprintf("%d", timestamp)})
	if err != nil {
		log.Errorf("work> Error while getting messages on topic %s, err:%s", topic, err.Error())
		return
	}

	skip := 0
	log.Debugf("work> Total messages on topic %s : %d", topic, countJSON.Count)

	for {
		if skip > countJSON.Count {
			log.Debugf("work> Skip skip(%d) > countJSON.Count (%d) on topic %s", skip, countJSON.Count, topic)
			break
		}

		msgs, err := getClient().MessageList(topic, &tat.MessageCriteria{
			Skip:          skip,
			Limit:         viper.GetInt("messages_limit"),
			DateMinUpdate: fmt.Sprintf("%d", timestamp),
		})
		if err != nil {
			log.Errorf("Error while requesting TAT, err:%s", err)
		}

		postESChan <- &indexableData{msgs.Messages, index}

		time.Sleep(10 * time.Second)
		skip += viper.GetInt("messages_limit") - 1
	}

}

func postES() {
	log.Debugf("postES enter")
	for recvData := range postESChan {
		log.Debugf("postES -> recvData for index ", recvData.index)

		indexES := recvData.index
		for _, m := range recvData.data {
			tg := make(map[string]string)
			for _, v := range m.Tags {
				tuple := strings.Split(v, ":")
				if len(tuple) == 2 {
					tg[tuple[0]] = tuple[1]
				}
			}

			dataES := map[string]interface{}{
				"ID":           m.ID,
				"Text":         m.Text,
				"Topic":        m.Topic,
				"InReplyOfID":  m.InReplyOfID,
				"NbLikes":      m.NbLikes,
				"Labels":       m.Labels,
				"Likers":       m.Likers,
				"UserMentions": m.UserMentions,
				"Urls":         m.Urls,
				"Tags":         m.Tags,
				"TagValues":    tg,
				"Author":       m.Author,
				"DateCreation": tat.DateFromFloat(m.DateCreation),
				"DateUpdate":   tat.DateFromFloat(m.DateUpdate),
				"Delta":        m.DateUpdate - m.DateCreation,
			}

			for _, label := range m.Labels {
				if label.Text == "Waiting" ||
					label.Text == "Building" ||
					label.Text == "Success" ||
					label.Text == "Failed" {

				}
				dataES["Status"] = label.Text
				break
			}

			log.Debugf("push %s to ES", dataES["ID"].(string))
			_, err := esConn.IndexWithParameters(indexES, "tatmessage", dataES["ID"].(string), "", 0, "", "", tat.DateFromFloat(m.DateCreation).Format(time.RFC3339), 0, "", "", false, nil, dataES)
			time.Sleep(100 * time.Millisecond)
			if err != nil {
				log.Errorf("cannot index message %s in %s :%s", dataES["ID"].(string), indexES, err)
			}
		}
	}
}
