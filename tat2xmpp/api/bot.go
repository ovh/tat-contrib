package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-xmpp"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var tatbot *botClient

const resource = "tat"

type topicConf struct {
	topic      string
	conference string
	typeHook   string
}

var topicConfs []topicConf

func (bot *botClient) born() {

	bot.creation = time.Now().UTC()

	topicConfs = []topicConf{}

	go bot.receive()

	for {
		bot.sendPresencesOnConfs()
		time.Sleep(30 * time.Second)
	}
}

func (bot *botClient) sendPresencesOnConfs() error {
	topicsJSON, err := bot.TatClient.TopicList(&tat.TopicCriteria{})
	if err != nil {
		return err
	}

	topicConfsNew := []topicConf{}
	for _, t := range topicsJSON.Topics {
		for _, p := range t.Parameters {
			if strings.HasPrefix(p.Key, tat.HookTypeXMPP) {
				topicConfsNew = append(topicConfsNew, topicConf{
					topic:      t.Topic,
					conference: p.Value,
					typeHook:   p.Key,
				})
				if strings.Contains(p.Value, "@conference.") {
					sd := strings.Split(p.Value, ";")
					destination := strings.TrimSpace(sd[0])
					conf := fmt.Sprintf("%s/%s", strings.TrimSpace(destination), resource)
					bot.sendPresence(conf)
				}
			}
		}
	}
	topicConfs = topicConfsNew
	return nil
}

func (bot *botClient) receive() {
	for {
		chat, err := bot.XMPPClient.Recv()
		if err != nil {
			log.Errorf("receive >> err: %s", err)
		}
		switch v := chat.(type) {
		case xmpp.Chat:
			fmt.Printf("Receive msg from xmpp :%+v\n", v)
			bot.receiveMsg(v)

			/* Code for presence case xmpp.Presence:
			fmt.Printf("Receive pres from jabb :%s\n", v)
			fmt.Println(v.From, v.Show)
			*/
		}
	}
}

func (bot *botClient) receiveMsg(chat xmpp.Chat) {
	log.Debugf("receiveMsg >> enter remote:%s text:%s", chat.Remote, chat.Text)
	/*
		chat.Stamp.Unix() contains... something wrong.
		if chat.Stamp.Unix() < bot.creation.Unix() {
			log.Debugf("receiveMsg >> exit, bot is starting... chat ts:%s, bot.creation:%s", chat.Stamp, bot.creation)
			return
		}*/
	if time.Now().Add(-10*time.Second).Unix() < bot.creation.Unix() {
		log.Debugf("receiveMsg >> exit, bot is starting... ")
		return
	}
	for _, t := range topicConfs {
		if t.typeHook != tat.HookTypeXMPPOut {
			log.Debugf("receiveMsg >> Check %s ", t.conference)
			if strings.Contains(chat.Remote, t.conference) {
				log.Debugf("Send message on tat topic %s , msg: %s", t.topic, chat.Text)
				username := strings.Replace(chat.Remote, t.conference+"/", "", 1)
				// if jid send msg on tat, do not resend on tat
				if username != resource && username != viper.GetString("xmpp_bot_jid") && strings.Trim(chat.Text, " ") != "" {
					text := fmt.Sprintf("#from:%s %s", username, chat.Text)
					bot.TatClient.MessageAdd(tat.MessageJSON{Text: text, Topic: t.topic})
				}
			}
		}
	}
}

// hookJSON is handler for Tat Webhook HookJSON
func hookJSON(ctx *gin.Context) {
	var hook tat.HookJSON
	ctx.BindJSON(&hook)

	if hook.HookMessage == nil || hook.HookMessage.MessageJSONOut == nil || hook.Hook.Destination == "" {
		log.Errorf("Invalid HookJSON received %+v", hook)
		ctx.JSON(http.StatusBadRequest, "Invalid HookJSON received")
		return
	}

	key := getHeader(ctx, tat.HookTat2XMPPHeaderKey)
	if key == "" || key != viper.GetString("hook_key") {
		ctx.JSON(http.StatusBadRequest, "Invalid key received")
		return
	}

	sd := strings.Split(hook.Hook.Destination, ";")
	destination := strings.TrimSpace(sd[0])
	from := ""
	topic := ""
	if len(sd) > 1 {
		for _, arg := range sd {
			if strings.HasPrefix(arg, "from:") && len(arg) > len("from:") {
				from = fmt.Sprintf(" from %s", arg[5:])
			} else if arg == "topic=true" {
				topic = fmt.Sprintf(" on topic %s", hook.HookMessage.MessageJSONOut.Message.Topic)
			}
		}
	}

	typeXMPP := "chat"
	if strings.Contains(destination, "@conference.") {
		typeXMPP = "groupchat"

		presenceToSend := true
		for _, c := range topicConfs {
			if strings.HasPrefix(c.conference, destination) {
				presenceToSend = false
			}
		}
		if presenceToSend {
			conf := fmt.Sprintf("%s/%s", destination, resource)
			tatbot.sendPresence(conf)
			time.Sleep(10 * time.Second)
		}
	}

	action := ""
	if hook.HookMessage.Action != tat.MessageActionCreate {
		action = fmt.Sprintf("[%s]", hook.HookMessage.Action)
	}

	by := ""
	if hook.Username != "" {
		by = fmt.Sprintf(" - hook by %s", hook.Username)
	}

	topicFilter := ""
	if hook.HookMessage.MessageJSONOut.Message.Topic != "" {
		topic = fmt.Sprintf(" on %s", hook.HookMessage.MessageJSONOut.Message.Topic)
	}
	text := fmt.Sprintf("[%s]%s %s%s%s%s%s",
		hook.HookMessage.MessageJSONOut.Message.Author.Username,
		action,
		hook.HookMessage.MessageJSONOut.Message.Text,
		by,
		from,
		topic,
		topicFilter,
	)

	labels := []string{}
	for _, l := range hook.HookMessage.MessageJSONOut.Message.Labels {
		labels = append(labels, l.Text)
	}
	if len(labels) > 0 {
		labelsTxt := strings.Join(labels, ", ")
		text = fmt.Sprintf("%s (%s)", text, labelsTxt)
	}

	tatbot.XMPPClient.Send(xmpp.Chat{
		Remote: destination,
		Type:   typeXMPP,
		Text:   text,
	})

	ctx.JSON(http.StatusCreated, "XMPP Message sent")
}

func getHeader(ctx *gin.Context, headerName string) string {
	for k, v := range ctx.Request.Header {
		if strings.ToLower(k) == strings.ToLower(headerName) {
			return v[0]
		}
	}
	return ""
}
