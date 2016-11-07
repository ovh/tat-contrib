package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
)

func do() {

	/* ********* */
	// here your code :-)
	// below is just an example
	/* ********* */

	topics, err := getClient().TopicList(&tat.TopicCriteria{Skip: 0, Limit: 100})
	if err != nil {
		log.Errorf("Error with topic list %s", err.Error())
		return
	}

	for _, topic := range topics.Topics {
		log.Debugf("Work on topic %s", topic.Topic)

		msgs, err := getClient().MessageList(topic.Topic, &tat.MessageCriteria{Tag: "exampleSearchOfTag"})
		if err != nil {
			log.Errorf("Error while getting messages on topic %s, err:%s", topic.Topic, err.Error())
			continue
		}
		for _, m := range msgs.Messages {
			log.Debugf("message:%s", m.Text)
		}
	}
}
