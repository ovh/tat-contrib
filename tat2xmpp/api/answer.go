package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-xmpp"
	"github.com/spf13/viper"

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
	question := strings.TrimSpace(text[5:]) // remove '/tat ' or 'tat, '
	if question == "help" {
		return help()
	} else if question == "tat2xmpp status" {
		if isAdmin(remote) {
			return getStatus()
		}
		return "forbidden for you " + remote
	} else if strings.HasPrefix(question, "aliases") {
		return getAliases(remote, question)
	} else if strings.HasPrefix(question, "GET ") || strings.HasPrefix(question, "COUNT ") {
		return bot.requestTat(question, remote)
	} else if strings.HasPrefix(question, "!") {
		return bot.execAlias(question, remote)
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

func (bot *botClient) execAlias(question, remote string) string {
	if len(aliases) == 0 {
		return fmt.Sprintf("no alias configured")
	}

	isadm := isAdmin(remote)
	for _, alias := range aliases {
		if !canViewAlias(isadm, alias, remote) {
			continue
		}
		for _, tag := range alias.Tags {
			if strings.HasPrefix(tag, "alias:") {
				for _, cmd := range strings.Split(strings.TrimPrefix(tag, "alias:"), ",") {
					if strings.HasPrefix(question, "!"+cmd) {
						return bot.execAliasRequest(alias, remote, strings.TrimSpace(strings.TrimPrefix(question, "!"+cmd)))
					}
				}
			}
		}
	}
	return fmt.Sprintf("Invalid Alias %s, please check aliases with command /tat aliases", question)
}

func (bot *botClient) execAliasRequest(msg tat.Message, remote, args string) string {

	values := strings.Split(args, " ")
	va := make([]interface{}, len(values))
	for i, v := range values {
		va[i] = v
	}
	format := ""
	for _, tag := range msg.Tags {
		if strings.HasPrefix(tag, "format:") {
			format = tag
			break
		}
	}
	for _, tag := range msg.Tags {
		if strings.HasPrefix(tag, "get:") {
			nbRequestsWithAlias++
			if args != "" {
				return bot.requestTat(fmt.Sprintf("GET "+tag[4:]+" "+format, va...), remote)
			}
			return bot.requestTat(fmt.Sprintf("GET "+tag[4:]+" "+format), remote)
		} else if strings.HasPrefix(tag, "count:") {
			nbRequestsWithAlias++
			if args != "" {
				return bot.requestTat(fmt.Sprintf("COUNT "+tag[6:]+" "+format, va...), remote)
			}
			return bot.requestTat(fmt.Sprintf("COUNT "+tag[6:]+" "+format), remote)
		}
	}
	nbRequestsWithAliasErrors++
	return "Invalid alias: " + msg.Text
}

func getAliases(remote, question string) string {

	filter := strings.TrimSpace(strings.TrimPrefix(question, "aliases"))
	if filter == "" {
		filter = "common"
	}

	isadm := isAdmin(remote)
	out := ""
	for _, alias := range aliases {
		// for private topics, if not admin, check author of message
		if !canViewAlias(isadm, alias, remote) {
			continue
		}

		if filter != "all" {
			found := false
			for _, t := range alias.Labels {
				if t.Text == filter {
					found = true
				}
			}
			if !found {
				continue
			}
		}

		t := strings.Replace(strings.TrimSpace(alias.Text), "#tatbot ", "", 1)
		t = strings.Replace(t, "#alias ", "", 1)
		t = strings.Replace(t, "#get:", "/tat GET ", 1)
		t = strings.Replace(t, "#count:", "/tat COUNT ", 1)
		out += fmt.Sprintf("%s \nAuthor: %s in topic %s\n------\n", t, alias.Author.Username, alias.Topic)
	}
	if out == "" {
		return "no alias configured"
	}
	return " aliases:\n------\n" + out
}

func canViewAlias(isAdm bool, msg tat.Message, remote string) bool {
	if isAdm {
		return true
	}
	if strings.HasPrefix(msg.Topic, "/Private/") && strings.HasPrefix(remote, msg.Author.Username+"@") {
		return true
	}
	return true
}

func help() string {
	out := `
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
urls,tags,dateCreation,dateUpdate,username,fullname,nbReplies,tatwebuiURL

User tat.system.jabber have to be RO on tat topic for requesting tat.

Get aliases : "/tat aliases", same as "/tat aliases common"
Get aliases with a specific tag : "/tat aliases atag"

Execute an alias : "/tat !myAlias arg1 arg2"

If you add a tat message, with label "common" and text:
"#tatbot #alias #alias:alert #get:/Internal/Alerts?tag=%s&label=%s #format:dateUpdate,text"
you can execute it over XMPP as : "/tat !alert CD open"

For a count request:
"#tatbot #alias #alias:alert.count #count:/Internal/Alerts?tag=%s&label=%s"
you can execute it over XMPP as : "/tat !alert.count CD open"

`

	return out + viper.GetString("more_help")
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
	defaultLimit := 5

	in = strings.TrimSpace(in)

	help := "Invalid request " + in + ". See /tat help"
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
			return "Invalid format for request " + in + ", see /tat help"
		}
		format = strings.TrimPrefix(format, "format:")
	}

	var values url.Values
	if strings.Contains(topic, "?") {
		tuple2 := strings.Split(topic, "?")
		if len(tuple2) != 2 {
			return "Invalid request. Request have to contains ?, see /tat help"
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
		nbRequestsCountTatErrors++
		return "Error while requesting tat (count)"
	}
	nbRequestsCountTat++

	msgs := fmt.Sprintf("%d message%s matching request %s", out.Count, plurial(out.Count), in)
	if strings.HasPrefix(in, "COUNT ") || out.Count == 0 {
		if viper.GetString("url_tatwebui") != "" {
			path := fmt.Sprintf("%s%s?%s", viper.GetString("url_tatwebui"), topic, criteria.GetURL())
			msgs += "\nSee on tatwebui: " + path
		}
		return msgs
	}

	criteria.Limit = defaultLimit
	outmsg, errc := bot.TatClient.MessageList(topic, criteria)
	if errc != nil {
		log.Warnf("Error requesting tat (list) for %s :%s", remote, errc)
		nbRequestsGetTatErrors++
		return "Error while requesting tat: %s" + errc.Error()
	}
	nbRequestsGetTat++

	if len(outmsg.Messages) == 0 {
		return msgs + " but 0 message after requesting details... strange..."
	}

	if len(outmsg.Messages) > defaultLimit {
		msgs += fmt.Sprintf(" but show only %d here", defaultLimit)
	}

	msgs += " :\n"
	for _, m := range outmsg.Messages {
		f, err := m.Format(format, viper.GetString("url_tatwebui"))
		if err != nil {
			return fmt.Sprintf("Invalid format "+format+", see /tat help", format)
		}
		msgs += f + "\n"
	}

	if viper.GetString("url_tatwebui") != "" {
		path := fmt.Sprintf("%s%s?%s", viper.GetString("url_tatwebui"), topic, criteria.GetURL())
		msgs += "See on tatwebui: " + path
	}

	return msgs
}

func plurial(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
