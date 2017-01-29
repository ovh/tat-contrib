package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-xmpp"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var (
	tatbot *botClient
)

const resource = "tat"

type topicConf struct {
	topic            string
	conference       string
	typeHook         string
	isOnlyFilterHook bool
}

var topicConfs []topicConf
var topicConfsFilterHook []topicConf

func (bot *botClient) born() {

	bot.creation = time.Now().UTC()

	topicConfs = []topicConf{}
	topicConfsFilterHook = []topicConf{}
	rand.Seed(time.Now().Unix())

	bot.chats = make(chan xmpp.Chat)
	go bot.sendToXMPP()

	bot.helloWorld()

	go bot.receive()

	for {
		sendInitialPresence(bot.XMPPClient)
		time.Sleep(10 * time.Second)
		bot.sendPresencesOnConfs(true)
		time.Sleep(20 * time.Second)
	}
}

func (bot *botClient) helloWorld() {
	for _, a := range bot.admins {
		log.Infof("helloWorld >> sending hello world to %s", a)

		bot.chats <- xmpp.Chat{
			Remote: a,
			Type:   "chat",
			Text:   fmt.Sprintf("Hi, I'm Tat2XMPP, what a good day to be alive"),
		}
	}

}

const status = `
Tat2XMPP Status

Started: {{.started}} since {{.since}}
Admin: {{.admin}}

XMPP:
- sent: {{.nbXMPPSent}}, errors: {{.nbXMPPErrors}}, errors after retry: {{.nbXMPPErrorsAfterRetry}}
- renew: {{.nbRenew}}

----
Bot:
- answers: {{.nbXMPPAnswers}}
- aliases: {{.aliases}}
- count on tat: {{.nbRequestsCountTat}}, errors: {{.nbRequestsCountTatErrors}}
- get on tat: {{.nbRequestsGetTat}}, errors: {{.nbRequestsGetTatErrors}}
- aliases used: {{.nbRequestsWithAlias}}, errors: {{.nbRequestsWithAliasErrors}}

----
Tat:
- sent: {{.nbTatSent}}, errors: {{.nbTatErrors}}
- conf on topic parameter: {{.nbTopicConfsFilterHook}}
- conf with filterHook:
- confs:
{{.stopicConfs}}

`

func (bot *botClient) getStatus() string {

	stopicConfs := ""
	for _, t := range topicConfs {
		stopicConfs += fmt.Sprintf("%s -> %s type:%s isOnlyFilterHook:%t\n", t.topic, t.conference, t.typeHook, t.isOnlyFilterHook)
	}

	data := map[string]string{
		"started": fmt.Sprintf("%s", tatbot.creation),
		"since":   fmt.Sprintf("%s", time.Now().Sub(tatbot.creation)),
		"admin":   viper.GetString("admin_tat2xmpp"),
		//-- xmpp
		"nbXMPPSent":             fmt.Sprintf("%d", bot.nbXMPPSent),
		"nbXMPPErrors":           fmt.Sprintf("%d", bot.nbXMPPErrors),
		"nbXMPPErrorsAfterRetry": fmt.Sprintf("%d", bot.nbXMPPErrorsAfterRetry),
		"nbRenew":                fmt.Sprintf("%d", bot.nbRenew),
		//-- bot
		"nbXMPPAnswers":             fmt.Sprintf("%d", bot.nbXMPPAnswers),
		"aliases":                   fmt.Sprintf("%d", len(bot.aliases)),
		"nbRequestsCountTat":        fmt.Sprintf("%d", bot.nbRequestsCountTat),
		"nbRequestsCountTatErrors":  fmt.Sprintf("%d", bot.nbRequestsCountTatErrors),
		"nbRequestsGetTat":          fmt.Sprintf("%d", bot.nbRequestsGetTat),
		"nbRequestsGetTatErrors":    fmt.Sprintf("%d", bot.nbRequestsGetTatErrors),
		"nbRequestsWithAlias":       fmt.Sprintf("%d", bot.nbRequestsWithAlias),
		"nbRequestsWithAliasErrors": fmt.Sprintf("%d", bot.nbRequestsWithAliasErrors),
		//-- tat
		"nbTatSent":              fmt.Sprintf("%d", bot.nbTatSent),
		"nbTatErrors":            fmt.Sprintf("%d", bot.nbTatErrors),
		"nbTopicConfs":           fmt.Sprintf("%d", bot.nbTopicConfs),
		"nbTopicConfsFilterHook": fmt.Sprintf("%d", len(topicConfsFilterHook)),
		"stopicConfs":            stopicConfs,
	}

	t, errp := template.New("status").Parse(status)
	if errp != nil {
		log.Errorf("getStatus> Error:%s", errp.Error())
		return "Error while prepare status:" + errp.Error()
	}

	var buffer bytes.Buffer
	if err := t.Execute(&buffer, data); err != nil {
		log.Errorf("getStatus> Error:%s", errp.Error())
		return "Error while prepare status (execute):" + err.Error()
	}

	return buffer.String()
}

func (bot *botClient) renewXMPP() {
	bot.nbRenew++
	bot.sendPresencesOnConfs(false)
}

func (bot *botClient) sendPresencesOnConfs(refreshAlias bool) error {
	topicsJSON, err := bot.TatClient.TopicList(&tat.TopicCriteria{})
	if err != nil {
		return err
	}

	topicConfsNew := []topicConf{}
	newAliases := []tat.Message{}
	for _, t := range topicsJSON.Topics {
		for _, p := range t.Parameters {
			if strings.HasPrefix(p.Key, tat.HookTypeXMPP) {
				if strings.Contains(p.Value, "@conference.") {
					topicConfsNew = append(topicConfsNew, topicConf{
						topic:      t.Topic,
						conference: p.Value,
						typeHook:   p.Key,
					})
				}
			}
		}

		if refreshAlias {
			newAliases = append(newAliases, bot.getAlias(t.Topic)...)
		}

	}

	bot.nbTopicConfs = len(topicConfsNew)
	topicConfs = topicConfsNew
	topicConfs = append(topicConfs, topicConfsFilterHook...)

	for _, t := range topicConfs {
		sd := strings.Split(t.conference, ";")
		destination := strings.TrimSpace(sd[0])
		bot.XMPPClient.JoinMUCNoHistory(strings.TrimSpace(destination), resource)
	}

	if refreshAlias {
		bot.aliases = newAliases
	}

	return nil
}

func (bot *botClient) getAlias(topic string) []tat.Message {
	msgs, err := bot.TatClient.MessageList(topic, &tat.MessageCriteria{AndTag: "tatbot,alias", NotLabel: "off"})
	if err != nil {
		log.Errorf("getAlias >> error while requesting tat:%s on topic %s", err, topic)
		return nil
	}

	return msgs.Messages
}

func (bot *botClient) sendToXMPP() {
	for {
		bot.XMPPClient.Send(<-bot.chats)
		time.Sleep(time.Duration(viper.GetInt("xmpp_delay")) * time.Second)
	}
}

func (bot *botClient) receive() {
	for {
		chat, err := bot.XMPPClient.Recv()
		if err != nil {
			if !strings.Contains(err.Error(), "EOF") {
				log.Errorf("receive >> err: %s", err)
			}
		}
		isError := false
		switch v := chat.(type) {
		case xmpp.Chat:
			if v.Remote != "" {
				if v.Type == "error" {

					isError = true
					log.Errorf("receive> msg error from xmpp :%+v\n", v)

					if !strings.HasSuffix(v.Text, " [tat2xmppRetry]") {
						bot.nbXMPPErrors++
						go tatbot.sendRetry(v)
					} else {
						bot.nbXMPPErrorsAfterRetry++
					}
				} else {
					log.Debugf("receive> msg from xmpp :%+v\n", v)
				}
			}

			if !isError {
				bot.receiveMsg(v)
			}

			/* Code for presence case xmpp.Presence:
			fmt.Printf("Receive pres from jabb :%s\n", v)
			fmt.Println(v.From, v.Show)
			*/
		}
	}
}

func (bot *botClient) sendRetry(v xmpp.Chat) {
	time.Sleep(60 * time.Second)
	bot.chats <- xmpp.Chat{
		Remote: v.Remote,
		Type:   getTypeChat(v.Remote),
		Text:   v.Text + " [tat2xmppRetry]",
	}
}

func getTypeChat(s string) string {
	if strings.Contains(s, "@conference.") {
		return typeGroupChat
	}
	return typeChat
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

	if strings.HasPrefix(chat.Text, "tat, ") || strings.HasPrefix(chat.Text, "/tat ") {
		log.Infof("receiveMsg for tat bot >> %s from remote:%s stamp:%s", chat.Text, chat.Remote, chat.Stamp)
		bot.answer(chat)
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
					if _, err := bot.TatClient.MessageAdd(tat.MessageJSON{Text: text, Topic: t.topic}); err != nil {
						log.Errorf("Error while send message on tat:%s", err)
						bot.nbTatErrors++
					} else {
						bot.nbTatSent++
					}
					time.Sleep(1 * time.Second)
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

	ctx.JSON(http.StatusCreated, fmt.Sprintf("Message received"))

	go hookProcess(hook)
}

func hookProcess(hook tat.HookJSON) {
	sd := strings.Split(hook.Hook.Destination, ";")
	destination := strings.TrimSpace(sd[0])
	from := ""
	topic := ""

	log.Debugf("hookJSON> Hook received destination:%s compute: %s", hook.Hook.Destination, destination)

	if len(sd) > 1 {
		for _, arg := range sd {
			if strings.HasPrefix(arg, "from:") && len(arg) > len("from:") {
				from = fmt.Sprintf(" from %s", arg[5:])
			} else if arg == "topic=true" {
				topic = fmt.Sprintf(" on topic %s", hook.HookMessage.MessageJSONOut.Message.Topic)
			}
		}
	}

	typeXMPP := getTypeChat(destination)
	if typeXMPP == typeGroupChat {
		presenceToSend := true
		for _, c := range topicConfs {
			if strings.HasPrefix(c.conference, destination) {
				presenceToSend = false
			}
		}

		if presenceToSend {
			log.Debugf("hookJSON> presenceToSend Add t:%s c:%s t:%s", hook.HookMessage.MessageJSONOut.Message.Topic, destination, hook.Hook.Type)
			topicConfsFilterHook = append(topicConfsFilterHook, topicConf{
				topic:            hook.HookMessage.MessageJSONOut.Message.Topic,
				conference:       destination,
				typeHook:         hook.Hook.Type,
				isOnlyFilterHook: true,
			})

			tatbot.renewXMPP()
			time.Sleep(30 * time.Second)
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

	tatbot.chats <- xmpp.Chat{
		Remote: destination,
		Type:   typeXMPP,
		Text:   text,
	}
	tatbot.nbXMPPSent++
}

func getHeader(ctx *gin.Context, headerName string) string {
	for k, v := range ctx.Request.Header {
		if strings.ToLower(k) == strings.ToLower(headerName) {
			return v[0]
		}
	}
	return ""
}
