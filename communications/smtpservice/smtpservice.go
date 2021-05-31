package smtpservice

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	mime    = "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msgSMTP = "To: %s\r\nSubject: %s\r\n%s\r\n%s"
)

// SMTPservice uses the net/smtp package to send emails to a recipient list
type SMTPservice struct {
	base.Base
	Host            string
	Port            string
	AccountName     string
	AccountPassword string
	From            string
	RecipientList   string
}

// Setup takes in a SMTP configuration and sets SMTP server details and
// recipient list
func (s *SMTPservice) Setup(cfg *base.CommunicationsConfig) {
	s.Name = cfg.SMTPConfig.Name
	s.Enabled = cfg.SMTPConfig.Enabled
	s.Verbose = cfg.SMTPConfig.Verbose
	s.Host = cfg.SMTPConfig.Host
	s.Port = cfg.SMTPConfig.Port
	s.AccountName = cfg.SMTPConfig.AccountName
	s.AccountPassword = cfg.SMTPConfig.AccountPassword
	s.From = cfg.SMTPConfig.From
	s.RecipientList = cfg.SMTPConfig.RecipientList
	log.Debugf(log.CommunicationMgr, "SMTP: Setup - From: %v. To: %s. Server: %s.\n", s.From, s.RecipientList, s.Host)
}

// IsConnected returns whether or not the connection is connected
func (s *SMTPservice) IsConnected() bool {
	return s.Connected
}

// Connect connects to service
func (s *SMTPservice) Connect() error {
	s.Connected = true
	return nil
}

// PushEvent sends an event to supplied recipient list via SMTP
func (s *SMTPservice) PushEvent(e base.Event) error {
	return s.Send(e.Type, e.Message)
}

// Send sends an email template to the recipient list via your SMTP host when
// an internal event is triggered by GoCryptoTrader
func (s *SMTPservice) Send(subject, msg string) error {
	if subject == "" || msg == "" {
		return errors.New("STMPservice Send() please add subject and alert")
	}
	if s.Host == "" ||
		s.Port == "" ||
		s.AccountName == "" ||
		s.AccountPassword == "" ||
		s.From == "" {
		return errors.New("STMPservice Send() cannot send with unset service properties")
	}

	log.Debugf(log.CommunicationMgr, "SMTP: Sending email to %v. Subject: %s Message: %s [From: %s]\n", s.RecipientList,
		subject, msg, s.From)
	messageToSend := fmt.Sprintf(
		msgSMTP,
		s.RecipientList,
		subject,
		mime,
		msg)

	return smtp.SendMail(
		s.Host+":"+s.Port,
		smtp.PlainAuth("", s.AccountName, s.AccountPassword, s.Host),
		s.From,
		strings.Split(s.RecipientList, ","),
		[]byte(messageToSend))
}
