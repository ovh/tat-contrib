package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-xmpp"

	"github.com/ovh/tat"
)

func (bot *botClient) answer(chat xmpp.Chat) {

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
		Text:   to + ": " + bot.prepareAnswer(chat.Text, to, chat.Remote),
	}
	nbXMPPAnswers++
}

func (bot *botClient) prepareAnswer(text, short, remote string) string {
	question := text[5:] // remove '/tat ' or 'tat, '
	if question == "help" {
		return help()
	} else if question == "tat2xmpp status" {
		if isAdmin(remote) {
			return getStatus()
		}
		return "forbidden for you " + remote
	} else if strings.HasPrefix(question, "GET ") || strings.HasPrefix(question, "COUNT ") {
		return bot.requestTat(question, remote)
	} else if question == "ping" {
		return "pong"
	} else if strings.HasPrefix(question, "hi") {
		return "Hi!"
	} else if strings.HasPrefix(question, "yes or no?") {
		if rand.Int()%2 == 0 {
			return "yes"
		}
		return "no"
	}
	return random()
}

func help() string {
	return `
Begin conversation with "tat," or "/tat"

Simple request: "tat, ping"

Request tat:
 "/tat COUNT /Internal/Alerts?tag=NETWORK,label=open"
 "/tat GET /Internal/Alerts?tag=PUBCLOUD-serv,PUBCLOUD-host&label=open"

Request tat and format output:
 "/tat COUNT /Internal/Alerts?tag=NETWORK,label=open format:dateUpdate,username,text"

Default format:dateUpdate,username,text,labels

You can use:
id,text,topic,inReplyOfID,inReplyOfIDRoot,nbLikes,labels,
votersUP,votersDown,nbVotesUP,nbVotesDown,userMentions,
urls,tags,dateCreation,dateUpdate,username,fullname,nbReplies
`
}

func isAdmin(r string) bool {
	for _, a := range admins {
		if strings.HasPrefix(r, a) {
			return true
		}
	}
	return false
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

func (bot *botClient) requestTat(in, remote string) string {
	help := "Invalid request. Use COUNT or GET. Example COUNT /YourTopic?tag=foo, see /tat help"
	if !strings.HasPrefix(in, "COUNT ") && !strings.HasPrefix(in, "GET ") {
		return help
	}

	tuple := strings.Split(in, " ")
	if len(tuple) != 2 && len(tuple) != 3 {
		return help
	}

	topic := tuple[1]
	format := ""
	if len(tuple) == 3 {
		format = tuple[2]
		if !strings.HasPrefix(format, "format:") {
			return "Invalid format, see /tat help"
		}
	}

	var values url.Values
	if strings.Contains(topic, "?") {
		tuple2 := strings.Split(topic, "?")
		if len(tuple2) != 2 {
			return "Invalid request. Request have to contains ?, example COUNT, see /tat help"
		}
		topic = tuple2[0]
		var errv error
		values, errv = url.ParseQuery(tuple2[1])
		if errv != nil {
			log.Warnf("Invalid Query for %s :%s", remote, errv)
			return "Invalid Query, see /tat help"
		}
	}

	criteria, errb := tat.GetMessageCriteriaFromURLValues(values)
	if errb != nil {
		return "Invalid Query (values)"
	}

	out, errc := bot.TatClient.MessageCount(topic, criteria)
	if errc != nil {
		log.Warnf("Error requesting tat (count) for %s :%s", remote, errc)
		return "Error while requesting tat (count)"
	}

	msgs := fmt.Sprintf("%d message%s matching", out.Count, plurial(out.Count))
	if strings.HasPrefix(in, "COUNT ") || out.Count == 0 {
		return msgs
	}

	criteria.Limit = 5
	outmsg, errc := bot.TatClient.MessageList(topic, criteria)
	if errc != nil {
		log.Warnf("Error requesting tat (list) for %s :%s", remote, errc)
		return "Error while requesting tat: %s" + errc.Error()
	}

	if len(outmsg.Messages) == 0 {
		return msgs + " but 0 message after requesting details... strange..."
	}

	msgs += ":\n"
	for _, m := range outmsg.Messages {
		f, err := m.Format(format)
		if err != nil {
			return "Invalid format, see /tat help"
		}
		msgs += f + "\n"
	}

	return msgs

}

func plurial(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
