package smtpservice

import (
	"errors"
	"fmt"
	"net/smtp"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
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
func (s *SMTPservice) Setup(cfg *config.CommunicationsConfig) {
	s.Name = cfg.SMTPConfig.Name
	s.Enabled = cfg.SMTPConfig.Enabled
	s.Verbose = cfg.SMTPConfig.Verbose
	s.Host = cfg.SMTPConfig.Host
	s.Port = cfg.SMTPConfig.Port
	s.AccountName = cfg.SMTPConfig.AccountName
	s.AccountPassword = cfg.SMTPConfig.AccountPassword
	s.RecipientList = cfg.SMTPConfig.RecipientList
}

// Connect connects to service
func (s *SMTPservice) Connect() error {
	s.Connected = true
	return nil
}

// PushEvent sends an event to supplied recipient list via SMTP
func (s *SMTPservice) PushEvent(base.Event) error {
	return common.ErrNotYetImplemented
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
