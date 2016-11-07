package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/yesnault/go-imap/imap"
)

var imapLogMask = imap.LogConn | imap.LogState | imap.LogCmd
var imapSafeLogMask = imap.LogConn | imap.LogState

const errorSleepPeriod = 60 * time.Second

func (instance *singleton) checkMail(box string) error {
	if instance.doingCheck {
		log.Infof("Already checking mail...")
		return nil
	}

	defer func() {
		instance.doingCheck = false
	}()

	instance.doingCheck = true

	if instance.imapClient == nil {
		log.Debugf("Connecting ...")
		if err := instance.connect(); err != nil {
			log.Errorf("%s Will now sleep for %s", err, errorSleepPeriod)
			instance.disconnect()
			time.Sleep(errorSleepPeriod)
			return fmt.Errorf("Error (sleep and disconnected 1):%s", err.Error())
		}
		log.Debugf("Connected!")
	}

	count, err := instance.queryCount(box)
	if err != nil {
		log.Errorf("%s Will now sleep for %s", err, errorSleepPeriod)
		instance.disconnect()
		time.Sleep(errorSleepPeriod)
		return fmt.Errorf("Error (sleep and disconnected 2):%s", err.Error())
	}

	log.Debugf("count messages:%d", count)

	if count == 0 {
		log.Debugf("No message to fetch")
		return nil
	}

	messages, err := instance.fetch(box, count)
	if err != nil {
		log.Errorf("%s Will now sleep for %s", err, errorSleepPeriod)
		instance.disconnect()
		time.Sleep(errorSleepPeriod)
		return fmt.Errorf("Error (sleep and disconnected 3):%s", err.Error())
	}

	log.Debugf("call StackInQueue")
	for _, msg := range messages {
		instance.StackInQueue(msg)
	}
	log.Debugf("call AnalyseQueue")
	instance.analyseQueue(box)
	log.Debugf("End AnalyseQueue")

	return nil
}

func (instance *singleton) connect() error {
	var c *imap.Client
	var err error

	c, err = imap.DialTLS(viper.GetString("imap_host")+":993", nil)
	if err != nil {
		log.Errorf("Unable to dial: %s", err)
		return err
	}

	instance.imapClient = c

	if c.Caps["STARTTLS"] {
		log.Debugf("STARTTLS")
		_, err = check(instance.imapClient.StartTLS(nil))
		if err != nil {
			fmt.Printf("Unable to start TLS: %s\n", err)
			return err
		}
	}

	c.SetLogMask(imapSafeLogMask)
	_, err = check(instance.imapClient.Login(viper.GetString("imap_username"), viper.GetString("imap_password")))
	c.SetLogMask(imapLogMask)
	if err != nil {
		log.Errorf("Unable to login: %s", err)
		return err
	}

	return nil
}

func (instance *singleton) fetch(box string, nb uint32) ([]imap.Response, error) {

	log.Debugf("call Select")
	_, err := instance.imapClient.Select(box, true)
	if err != nil {
		log.Errorf("Error with select %s", err.Error())
		return []imap.Response{}, err
	}

	seqset, _ := imap.NewSeqSet("1:*")

	cmd, err := instance.imapClient.Fetch(seqset, "ENVELOPE", "RFC822.HEADER", "RFC822.TEXT", "UID")
	if err != nil {
		log.Errorf("Error with fetch:%s", err)
		return []imap.Response{}, err
	}

	// Process responses while the command is running
	log.Debugf("Most recent messages:")

	messages := []imap.Response{}
	for cmd.InProgress() {
		// Wait for the next response (no timeout)
		instance.imapClient.Recv(-1)

		// Process command data
		for _, rsp := range cmd.Data {
			messages = append(messages, *rsp)
		}
		cmd.Data = nil
		instance.imapClient.Data = nil
	}
	log.Debugf("Nb messages fetch:%d", len(messages))
	return messages, nil
}

func (instance *singleton) disconnect() {
	if instance.imapClient != nil {
		instance.imapClient.Close(false)
	}
	instance.imapClient = nil
}

func (instance *singleton) queryCount(box string) (uint32, error) {
	//cmd, err := check(instance.imapClient.Status(m.config.Label))
	cmd, err := check(instance.imapClient.Status(box))
	if err != nil {
		return 0, err
	}

	var count uint32
	for _, result := range cmd.Data {
		mailboxStatus := result.MailboxStatus()
		if mailboxStatus != nil {
			count += mailboxStatus.Messages
		}
	}

	return count, nil
}

func check(cmd *imap.Command, err error) (*imap.Command, error) {
	if err != nil {
		log.Errorf("IMAP ERROR: %s", err)
		return nil, err
	}

	_, err = cmd.Result(imap.OK)
	if err != nil {
		log.Errorf("COMMAND ERROR: %s", err)
		return nil, err
	}

	return cmd, err
}
