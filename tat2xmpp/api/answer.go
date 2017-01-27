package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

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
		Text:   bot.prepareAnswer(chat.Text, to, chat.Remote),
	}
	nbXMPPAnswers++
}

func (bot *botClient) prepareAnswer(text, short, remote string) string {
	question := text[5:] // remove '/tat ' or 'tat, '
	if question == "tat2xmpp status" {
		if isAdmin(remote) {
			return getStatus()
		}
		return short + ": forbidden for you " + remote
	} else if strings.HasPrefix(question, "GET ") || strings.HasPrefix(question, "COUNT ") {
		return bot.requestTat(question, remote)
	} else if question == "ping" {
		return short + ": pong"
	} else if strings.HasPrefix(question, "hi") {
		return short + ": Hi!"
	} else if strings.HasPrefix(question, "yes or no?") {
		if rand.Int()%2 == 0 {
			return short + ": yes"
		}
		return short + ": no"
	}
	return random()
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
	help := "invalid request prefix. Use COUNT or GET. Example COUNT /YourTopic?tag=foo"
	if !strings.HasPrefix(in, "COUNT ") && !strings.HasPrefix(in, "GET ") {
		return help
	}

	tuple := strings.Split(in, " ")
	if len(tuple) != 2 {
		return help
	}

	topic := tuple[1]
	var values url.Values
	if strings.Contains(topic, "?") {
		tuple2 := strings.Split(in, "?")
		if len(tuple2) != 2 {
			return "invalid request. Request have to contains ?, example COUNT"
		}
		topic = tuple2[0]
		var errv error
		values, errv = url.ParseQuery(tuple2[1])
		if errv != nil {
			log.Warnf("Invalid Query for %s :%s", remote, errv)
			return "Invalid Query"
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
	if strings.HasPrefix(in, "COUNT ") {
		return msgs
	}

	criteria.Limit = 5
	outmsg, errc := bot.TatClient.MessageList(topic, criteria)
	if errc != nil {
		log.Warnf("Error requesting tat (list) for %s :%s", remote, errc)
		return "Error while requesting tat: %s" + errc.Error()
	}

	if len(outmsg.Messages) == 0 {
		return "0 message after requesting details... strange..."
	}

	msgs += ":\n"
	for _, m := range outmsg.Messages {
		labels := ""
		for _, l := range m.Labels {
			labels += l.Text + " "
		}
		msgs += fmt.Sprintf("%s %s %s %s \n",
			m.Author.Username,
			fmt.Sprintf("%s", time.Unix(int64(m.DateUpdate), 0).Format(time.Stamp)),
			m.Text,
			labels,
		)
	}

	return msgs

}

func plurial(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
