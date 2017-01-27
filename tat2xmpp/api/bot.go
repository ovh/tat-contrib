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
	tatbot                 *botClient
	nbXMPPErrors           int
	nbXMPPErrorsAfterRetry int
	nbXMPPSent             int
	nbTatErrors            int
	nbTatSent              int
	nbXMPPAnswers          int
	nbRenew                int
	nbTopicConfs           int
	chats                  chan xmpp.Chat
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
		bot.sendPresencesOnConfs()
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
	return fmt.Sprintf("tat2xmpp-status>> started:%s nbXMPPErrors:%d nbXMPPErrorsAfterRetry:%d nbXMPPSent:%d nbXMPPAnswers:%d nbTatErrors:%d nbTatSent:%d nbTopicConfs:%d nbTopicConfsFilterHook:%d renew:%d",
		tatbot.creation, nbXMPPErrors, nbXMPPErrorsAfterRetry, nbXMPPSent, nbXMPPAnswers, nbTatErrors, nbTatSent, nbTopicConfs, len(topicConfsFilterHook), nbRenew)
}

func status() {
	log.Infof(getStatus())
	time.Sleep(10 * time.Minute)
}

func (bot *botClient) renewXMPP() {
	nbRenew++
	bot.sendPresencesOnConfs()
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
				if strings.Contains(p.Value, "@conference.") {
					topicConfsNew = append(topicConfsNew, topicConf{
						topic:      t.Topic,
						conference: p.Value,
						typeHook:   p.Key,
					})
				}
			}
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

	return nil
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
		return "groupchat"
	}
	return "chat"
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

	if strings.HasPrefix(chat.Text, "tat, ") {
		log.Infof("receiveMsg for tat bot >> %s from remote:%s stamp:%s", chat.Text, chat.Remote, chat.Stamp)
		answer(chat)
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

func answer(chat xmpp.Chat) {

	typeXMPP := getTypeChat(chat.Remote)
	remote := chat.Remote
	to := chat.Remote
	if typeXMPP == "groupchat" {
		if strings.Contains(chat.Remote, "/") {
			t := strings.Split(chat.Remote, "/")
			remote = t[0]
			to = t[1]
		}
	} else {
		to = strings.Split(chat.Remote, "@")[0]
	}

	chats <- xmpp.Chat{
		Remote: remote,
		Type:   typeXMPP,
		Text:   prepareAnswer(chat.Text, to),
	}
	nbXMPPAnswers++
}

func prepareAnswer(question, remote string) string {
	if question == "tat, give me tat2xmpp status" {
		return getStatus()
	} else if question == "tat, ping" {
		return "pong"
	} else if strings.HasPrefix(question, "tat, hi") {
		return "Hi " + remote
	} else if strings.HasPrefix(question, "tat, yes or no?") {
		if rand.Int()%2 == 0 {
			return "yes"
		}
		return "no"
	}
	return random()
}

func random() string {
	answers := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes definitely",
		"You may rely on it",
		"As I see it yes",
		"Most likely",
		"Outlook good",
		"Yes",
		"Signs point to yes",
		"Reply hazy try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
		"Nooooo",
	}
	return answers[rand.Intn(len(answers))]
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
	if typeXMPP == "groupchat" {
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
