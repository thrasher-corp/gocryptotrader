package smtpservice

import (
	"errors"
	"fmt"
	"net/smtp"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
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
	RecipientList   string
}

// Setup takes in a SMTP configuration and sets SMTP server details and
// recipient list
func (s *SMTPservice) Setup(config config.CommunicationsConfig) {
	s.Name = config.SMTPConfig.Name
	s.Enabled = config.SMTPConfig.Enabled
	s.Verbose = config.SMTPConfig.Verbose
	s.Host = config.SMTPConfig.Host
	s.Port = config.SMTPConfig.Port
	s.AccountName = config.SMTPConfig.AccountName
	s.AccountPassword = config.SMTPConfig.AccountPassword
	s.RecipientList = config.SMTPConfig.RecipientList
}

// Connect connects to service
func (s *SMTPservice) Connect() error {
	s.Connected = true
	return nil
}

// PushEvent sends an event to supplied recipient list via SMTP
func (s *SMTPservice) PushEvent(base.Event) error {
	return errors.New("not yet implemented")
}

// Send sends an email template to the recipient list via your SMTP host when
// an internal event is triggered by GoCryptoTrader
func (s *SMTPservice) Send(subject, alert string) error {
	if subject == "" || alert == "" {
		return errors.New("STMPservice Send() please add subject and alert")
	}

	list := common.SplitStrings(s.RecipientList, ",")

	for i := range list {
		messageToSend := fmt.Sprintf(
			msgSMTP,
			list[i],
			subject,
			mime,
			alert)

		err := smtp.SendMail(
			s.Host+":"+s.Port,
			smtp.PlainAuth("", s.AccountName, s.AccountPassword, s.Host),
			s.AccountName,
			[]string{list[i]},
			[]byte(messageToSend))
		if err != nil {
			return err
		}
	}
	return nil
}
