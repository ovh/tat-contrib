package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
)

func run() {
	for {
		log.Debugf("RUN Dashing")
		do()
		time.Sleep(30 * time.Second)
	}
}

func do() {

	topics, err := getClient().TopicList(&tat.TopicCriteria{Skip: 0, Limit: 1000})
	if err != nil {
		log.Errorf("Error with topic list %s", err.Error())
		return
	}

	for _, topic := range topics.Topics {
		log.Debugf("Work on topic %s", topic.Topic)
		doTopic(topic)
		log.Debugf("End Work on topic %s", topic.Topic)
	}
}

func doTopic(topic tat.Topic) {
	messages, err := getClient().MessageList(topic.Topic, &tat.MessageCriteria{Skip: 0, Limit: 100, AndTag: "TatDashing", TreeView: tat.TreeViewOneTree})
	if err != nil {
		log.Errorf("Error with messages list on topic %s err:%s", topic.Topic, err.Error())
		return
	}

	log.Debugf("Topic %s %d messages", topic.Topic, len(messages.Messages))
	for _, msg := range messages.Messages {
		doMessage(msg)
	}
}

type sortedKeys []string

func (s sortedKeys) Len() int {
	return len(s)
}
func (s sortedKeys) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortedKeys) Less(i, j int) bool {
	ni := strings.Index(s[i], ":")
	nj := strings.Index(s[j], ":")

	if ni > 0 && nj > 0 {
		ki, erra := strconv.Atoi(s[i][0:ni])
		kj, errb := strconv.Atoi(s[j][0:nj])
		if erra != nil || errb != nil {
			return false
		}
		return ki < kj
	}
	return false
}

func doMessage(msg tat.Message) {
	toAdd := false

	labels := []tat.Label{}
	colorTag := ""
	bgColorTag := ""
	var valueItem float64
	colorToCompute := false
	for _, reply := range msg.Replies {
		found := false
		values := make(map[string]string)
		if tat.ArrayContains(reply.Tags, "TatDashing") {
			keys := []string{}
			for _, tag := range reply.Tags {
				if strings.HasPrefix(tag, "label:") {

					if strings.HasPrefix(tag, "label:color:") {
						colorTag = tag
						continue
					} else if strings.HasPrefix(tag, "label:bg-color:") {
						bgColorTag = tag
						continue
					}

					tuple := strings.Split(tag, ":") // #label:0:widget-data-series
					if len(tuple) != 3 {
						continue
					}
					index := tuple[1]
					label := tuple[2]
					var value string

					found = true
					value = getValue(reply, label)
					if n, err := strconv.ParseFloat(value, 64); err == nil {
						valueItem = n
						colorToCompute = true
					}
					values[label+":"+index] = value
					keys = append(keys, label+":"+index)
				}
			}

			if found {
				value := ""
				label := ""

				sort.Sort(sortedKeys(keys))
				for _, k := range keys {
					tuple := strings.Split(k, ":")
					label = tuple[0]
					value = fmt.Sprintf("%s", values[k])
				}
				toAdd = true
				labels = append(labels, tat.Label{Text: fmt.Sprintf("%s:%s", label, value)})
			}
		}
	}

	if !toAdd {
		return
	}

	if colorToCompute {
		if colorTag != "" {
			if c, err := computeColor(strings.Replace(colorTag, "label:color:", "", -1), valueItem); err == nil {
				labels = append(labels, tat.Label{Text: "color", Color: c})
			}
		}
		if bgColorTag != "" {
			if c, err := computeColor(strings.Replace(bgColorTag, "label:bg-color:", "", -1), valueItem); err == nil {
				labels = append(labels, tat.Label{Text: "bg-color", Color: c})
			}
		}
	}

	for _, l := range msg.Labels {

		if l.Text == "color" && colorTag != "" {
			continue
		} else if l.Text == "bg-color" && bgColorTag != "" {
			continue
		}

		tuple := strings.Split(l.Text, ":") // value:10
		if len(tuple) != 2 {
			labels = append(labels, l)
			continue
		}

		toAddE := true
		for _, label := range labels {
			if strings.HasPrefix(label.Text, tuple[0]+":") {
				toAddE = false
				break
			}
		}
		if toAddE {
			labels = append(labels, l)
		}
	}

	if _, err := getClient().MessageRelabel(msg.Topic, msg.ID, labels, nil); err != nil {
		log.Warnf("Error while MessageRelabel topic:%s msg:%s err:%s", msg.Topic, msg.ID, err.Error())
	}
}

func computeColor(t string, value float64) (string, error) {

	// #label:color:#eeeee:1:2;#fa6800:2:4;
	// green 93c47d
	tuple := strings.Split(t, ",")
	lastColor := ""
	for _, v := range tuple {
		trip := strings.Split(v, ":")
		lastColor = "#" + trip[0]
		if len(trip) != 3 {
			continue
		}
		if minValue, errMin := strconv.ParseFloat(trip[1], 64); errMin == nil {
			if maxValue, errMax := strconv.ParseFloat(trip[2], 64); errMax == nil {
				if value >= minValue && value <= maxValue {
					return lastColor, nil
				}
			}
		}
	}

	if lastColor != "" {
		return lastColor, nil
	}
	return "", fmt.Errorf("computeColor - no color")
}

func getValue(reply tat.Message, label string) string {
	log.Debugf("getValue for label %s", label)
	sort.Strings(reply.Tags)
	out := ""
	for _, tag := range reply.Tags {
		if strings.HasPrefix(tag, fmt.Sprintf("valuelabel:")) {
			tuple := strings.Split(tag, ":") // #valuelabel:0:label:/Internal/topic?
			if len(tuple) < 4 {
				log.Debugf("getValue for label valuelabel:, but invalid format")
				continue
			}
			out = fmt.Sprintf("%s %s", out, getValueLabelOnTat(strings.Join(tuple[3:], ":"), tuple[2:3][0]))
		} else if strings.HasPrefix(tag, fmt.Sprintf("value:")) {
			tuple := strings.Split(tag, ":") // #value:0:/Internal/topic?
			if len(tuple) < 3 {
				log.Debugf("getValue for label value:, but invalid format")
				continue
			}
			out = fmt.Sprintf("%s %s", out, getValueOnTat(strings.Join(tuple[2:], ":")))
		}
	}
	return strings.TrimSpace(out)
}

func getValueLabelOnTat(path, label string) string {

	tuple := strings.Split(path, "?")
	if len(tuple) != 2 {
		return ""
	}
	topic := tuple[0]

	values, err := url.ParseQuery(tuple[1])
	if err != nil {
		log.Warnf("Invalid query:%s", path)
		return "error query"
	}

	criteria, errb := tat.GetMessageCriteriaFromURLValues(values)
	if errb != nil {
		return ""
	}

	criteria.Limit = 2
	log.Debugf("criteria:%v", criteria)
	n, errc := getClient().MessageList(topic, criteria)
	if errc != nil {
		return "error List"
	}

	if len(n.Messages) != 1 {
		return "error != 1"
	}

	nlabel := 0
	var vlabel string
	for _, cur := range n.Messages[0].Labels {
		if strings.HasPrefix(cur.Text, label+":") {
			nlabel++
			vlabel = cur.Text
		}
	}

	if nlabel != 1 {
		return "error nb label " + label
	}
	return strings.Split(vlabel, ":")[1]
}

func getValueOnTat(path string) string {

	tuple := strings.Split(path, "?")
	if len(tuple) != 2 {
		return ""
	}
	topic := tuple[0]

	values, err := url.ParseQuery(tuple[1])
	if err != nil {
		log.Warnf("Invalid query:%s", path)
		return "error"
	}

	criteria, errb := tat.GetMessageCriteriaFromURLValues(values)
	if errb != nil {
		return ""
	}

	log.Debugf("criteria:%v", criteria)
	n, errc := getClient().MessageCount(topic, criteria)
	if errc != nil {
		return "error"
	}

	return strconv.Itoa(n.Count)
}
