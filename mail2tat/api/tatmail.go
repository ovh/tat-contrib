package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/tat"
	"github.com/yesnault/go-imap/imap"
)

var topicsCache []string

const (
	grey = "#f3f3f3"
)

// TatMail ...
type TatMail struct {
	From       string
	Subject    string
	References string
	Date       time.Time
	Body       string
	MessageID  string
	TatRef     string
	UID        uint32
	//imapMsg    *imap.Message
	imapMsg  imap.Response
	Analyzed bool
}

// Queue ...
var Queue []*TatMail

// StackInQueue stacks msg in queue
func (instance *singleton) StackInQueue(msg imap.Response) {
	tm := extract(msg)

	for _, domain := range instance.allowedDomains {
		if !strings.HasSuffix(tm.From, domain) && !strings.HasSuffix(tm.From, domain) {
			log.Debugf("Mail is not from a valid domain")
			if e := tm.move(instance.imapClient, "Skipped"); e != nil {
				log.Errorf("Error while move mail to Skipped:%s", e.Error())
			}
			return
		}
	}

	tm.computeTatRef()
	tm.imapMsg = msg
	Queue = append(Queue, tm)
}

// AnalyseQueue analyses msg in queue
func (instance *singleton) analyseQueue(box string) {
	for _, tm := range Queue {
		if tm.References == "" {
			if err := instance.analyseTM(box, tm); err != nil {
				log.Errorf("err:%s", err)
			}
		}
	}

	for _, tm := range Queue {
		if err := instance.analyseTM(box, tm); err != nil {
			log.Errorf("err:%s", err)
		}
	}
	// reset queue
	Queue = []*TatMail{}
}

func (instance *singleton) analyseTM(box string, tm *TatMail) error {
	if tm.Analyzed {
		return nil
	}

	err := tm.post(instance)
	if err == nil {
		if e := tm.move(instance.imapClient, box+"Done"); e != nil {
			return fmt.Errorf("Error while move mail to %sDone:%s", box, e.Error())
		}
		tm.Analyzed = true
		return nil
	}
	return fmt.Errorf("Error while posting to tat:%s", err.Error())
}

func (tm *TatMail) getTatText() string {
	return fmt.Sprintf("#from:%s %s", tm.From, tm.Body)
}

func (tm *TatMail) computeTatRef() {
	if strings.Contains(tm.Subject, ",") {
		s := strings.Split(tm.Subject, ",")[1]
		s = strings.TrimSpace(s)
		tm.TatRef = strings.Replace(s, " ", "_", -1)
	}
	log.Infof("tm.TatRef is %s", tm.TatRef)
}

// extract qsdf <foo@ff.fr>
func extractWord(word string) string {
	first := strings.Index(word, "<")
	second := strings.Index(word, ">")
	if first >= 0 && second > first && second <= len(word) {
		return word[first+1 : second]
	}
	return word
}

func (tm *TatMail) getTatLabels() []tat.Label {
	if tm.TatRef == "" {
		return []tat.Label{}
	}
	labels := []tat.Label{{
		Text:  fmt.Sprintf("Ref:%s", tm.TatRef),
		Color: grey,
	}}
	return labels
}

func (tm *TatMail) post(instance *singleton) error {

	topic, err := tm.getTopic(instance.tatClient)
	if err != nil {
		return err
	}

	messages := []tat.Message{}
	if tm.TatRef != "" {
		crit := &tat.MessageCriteria{
			Limit:       1,
			OnlyMsgRoot: "true",
			Label:       "Ref:" + fmt.Sprintf("%s", tm.TatRef),
		}

		messagesJSON, err := instance.tatClient.MessageList(topic, crit)
		if err != nil {
			return err
		}
		messages = messagesJSON.Messages
	}

	msg := tat.MessageJSON{
		Text:         tm.getTatText(),
		Topic:        topic,
		Labels:       tm.getTatLabels(),
		DateCreation: float64(tm.Date.Unix()),
	}

	if len(messages) != 0 {
		msgRoot := messages[0]
		msg.IDReference = msgRoot.ID
	}
	if _, err := instance.tatClient.MessageAdd(msg); err != nil {
		return err
	}
	return nil
}

func (tm *TatMail) getTopic(tatClient *tat.Client) (string, error) {
	s := strings.TrimSpace(tm.Subject)

	if strings.Contains(tm.Subject, ",") {
		s = strings.Split(tm.Subject, ",")[0]
		s = strings.TrimSpace(s)
	}

	if !strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("Invalid topic name:%s", s)
	}

	if tat.ArrayContains(topicsCache, s) {
		return s, nil
	}

	// check topic on tat
	if TopicExists(tatClient, s) {
		topicsCache = append(topicsCache, s)
	}
	return s, nil
}

// TopicExists returns true if topic exists
func TopicExists(tatClient *tat.Client, topicName string) bool {
	//"curl -XGET https://<tatHostname>:<tatPort>/topic/topicName"
	log.Debugf("Search topicName %s on tat", topicName)

	if _, err := tatClient.TopicOne(topicName); err != nil {
		log.Debugf("Error while getting topic from tat:%s", err.Error())
		return false
	}
	return true
}

func (tm *TatMail) move(c *imap.Client, mbox string) error {
	seq, _ := imap.NewSeqSet("")
	seq.AddNum(tm.UID)

	if _, err := c.UIDMove(seq, mbox); err != nil {
		return fmt.Errorf("Error while move msg to %s, err:%s", mbox, err.Error())
	}
	return nil
}

func decodeHeader(msg *mail.Message, headerName string) string {
	dec := new(mime.WordDecoder)
	s, err := dec.DecodeHeader(msg.Header.Get(headerName))
	if err != nil {
		log.Errorf("Error while decode header %s:%s", headerName, msg.Header.Get(headerName))
		return msg.Header.Get(headerName)
	}
	return s
}

func hash(in string) string {
	h2 := md5.New()
	io.WriteString(h2, in)
	return fmt.Sprintf("%x", h2.Sum(nil))
}

func extract(rsp imap.Response) *TatMail {
	tm := &TatMail{}
	var params map[string]string
	var err error

	header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
	uid := imap.AsNumber((rsp.MessageInfo().Attrs["UID"]))
	body := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.TEXT"])
	if mmsg, _ := mail.ReadMessage(bytes.NewReader(header)); mmsg != nil {
		tm.Subject = decodeHeader(mmsg, "Subject")
		log.Debugf("|-- subject computed %s", tm.Subject)
		tm.From = extractWord(decodeHeader(mmsg, "From"))
		log.Debugf("|-- from %s", tm.From)

		//log.Warnf("Content-Type:%s", mmsg.Header.Get("Content-Type"))
		_, params, err = mime.ParseMediaType(mmsg.Header.Get("Content-Type"))
		if err != nil {
			log.Errorf("Error while read Content-Type:%s", err)
		}

		tm.References = hash(extractWord(mmsg.Header.Get("References")))
		tm.MessageID = hash(extractWord(mmsg.Header.Get("Message-ID")))

		//date := "Fri, 1 Jul 2016 14:31:48 +0000"
		date := strings.Trim(mmsg.Header.Get("Date"), " ")
		t, errt := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", date)
		if errt == nil {
			tm.Date = t
		} else {
			t2, errt2 := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700 (CEST)", date)
			if errt2 == nil {
				tm.Date = t2
			} else {
				log.Errorf("Error while converting date:%s", errt.Error())
				tm.Date = time.Now()
			}
		}
		tm.UID = uid
	}

	r := quotedprintable.NewReader(bytes.NewReader(body))
	bodya, err := ioutil.ReadAll(r)
	if err == nil {
		log.Infof("Decode quotedprintable OK")
		tm.Body = string(bodya)
		return tm
	} else if len(params) > 0 {
		log.Debugf("Error while read body:%s", err.Error())
		//r := bytes.NewReader(b.Bytes())
		r := bytes.NewReader(body)
		mr := multipart.NewReader(r, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				log.Debugf("errA:%s", err)
				continue
			}
			if err != nil {
				log.Debugf("--------> errB:%s", err)
				break
			}
			slurp, err := ioutil.ReadAll(p)
			if err != nil {
				log.Debugf("errC:%s", err)
				continue
			}
			log.Infof("Decode slurp OK")
			tm.Body = string(slurp)
			break
		}
		log.Debugf("stepD")
	}

	if tm.Body == "" {
		log.Debugf("EmptyBody, take body")
		tm.Body = string(bodya)
	} else {
		log.Debugf("stepF, tm.Body is ok")
	}
	return tm
}
