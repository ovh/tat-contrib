package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-xmpp"
	"github.com/ovh/tat"
	"github.com/spf13/viper"
)

var (
	tatbot                    *botClient
	nbXMPPErrors              int
	nbXMPPErrorsAfterRetry    int
	nbXMPPSent                int
	nbTatErrors               int
	nbTatSent                 int
	nbXMPPAnswers             int
	nbRenew                   int
	nbTopicConfs              int
	nbRequestsWithAlias       int
	nbRequestsWithAliasErrors int
	nbRequestsCountTat        int
	nbRequestsGetTat          int
	nbRequestsCountTatErrors  int
	nbRequestsGetTatErrors    int
	chats                     chan xmpp.Chat
	aliases                   []tat.Message
)

const resource = "tat"

type topicConf struct {
	topic      string
	conference string
	typeHook   string
}

var topicConfs []topicConf
var topicConfsFilterHook []topicConf

func (bot *botClient) born() {

	bot.creation = time.Now().UTC()

	topicConfs = []topicConf{}
	topicConfsFilterHook = []topicConf{}
	rand.Seed(time.Now().Unix())

	chats = make(chan xmpp.Chat)
	go bot.sendToXMPP()

	bot.helloWorld()

	go bot.receive()
	go status()

	for {
		sendInitialPresence(bot.XMPPClient)
		time.Sleep(10 * time.Second)
		bot.sendPresencesOnConfs(true)
		time.Sleep(20 * time.Second)
	}
}

func (bot *botClient) helloWorld() {
	if viper.GetString("xmpp_hello_world") == "" {
		log.Infof("helloWorld >> param xmpp-hello-world is empty")
		return
	}

	log.Infof("helloWorld >> sending hello world to %s", viper.GetString("xmpp_hello_world"))

	chats <- xmpp.Chat{
		Remote: viper.GetString("xmpp_hello_world"),
		Type:   "chat",
		Text:   fmt.Sprintf("Hi, I'm tat2xmpp, what a good day to be alive"),
	}
}

func getStatus() string {
	return fmt.Sprintf("tat2xmpp-status>> started:%s\n nbXMPPErrors:%d\n nbXMPPErrorsAfterRetry:%d\n nbXMPPSent:%d\n nbXMPPAnswers:%d\n nbTatErrors:%d\n nbTatSent:%d\n nbTopicConfs:%d\n nbTopicConfsFilterHook:%d\n renew:%d\n nbRequestsCountTat:%d\n nbRequestsGetTat:%d nbRequestsCountTatErrors:%d\n nbRequestsGetTatErrors:%d nbRequestsWithAlias:%d nbRequestsWithAlias:%d nbRequestsWithAliasErrors:%d nbAliases:%d\n",
		tatbot.creation, nbXMPPErrors, nbXMPPErrorsAfterRetry, nbXMPPSent, nbXMPPAnswers, nbTatErrors, nbTatSent, nbTopicConfs, len(topicConfsFilterHook), nbRenew, nbRequestsCountTat, nbRequestsGetTat, nbRequestsCountTatErrors, nbRequestsGetTatErrors, nbRequestsWithAlias, nbRequestsWithAlias, nbRequestsWithAliasErrors, len(aliases))
}

func status() {
	log.Infof(getStatus())
	time.Sleep(10 * time.Minute)
}

func (bot *botClient) renewXMPP() {
	nbRenew++
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

	nbTopicConfs = len(topicConfsNew)
	topicConfs = topicConfsNew
	topicConfs = append(topicConfs, topicConfsFilterHook...)

	for _, t := range topicConfs {
		sd := strings.Split(t.conference, ";")
		destination := strings.TrimSpace(sd[0])
		bot.XMPPClient.JoinMUCNoHistory(strings.TrimSpace(destination), resource)
	}

	if refreshAlias {
		aliases = newAliases
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
		tatbot.XMPPClient.Send(<-chats)
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
						nbXMPPErrors++
						go tatbot.sendRetry(v)
					} else {
						nbXMPPErrorsAfterRetry++
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
	chats <- xmpp.Chat{
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
						nbTatErrors++
					} else {
						nbTatSent++
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
				topic:      hook.HookMessage.MessageJSONOut.Message.Topic,
				conference: destination,
				typeHook:   hook.Hook.Type,
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

	chats <- xmpp.Chat{
		Remote: destination,
		Type:   typeXMPP,
		Text:   text,
	}
	nbXMPPSent++
}

func getHeader(ctx *gin.Context, headerName string) string {
	for k, v := range ctx.Request.Header {
		if strings.ToLower(k) == strings.ToLower(headerName) {
			return v[0]
		}
	}
	return ""
}
