package smsglobal

import (
	"errors"
	"log"
	"net/url"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	SMSGLOBAL_API_URL     = "http://www.smsglobal.com/http-api.php"
	ErrSMSContactNotFound = "SMS Contact not found."
	ErrSMSNotSent         = "SMS message not sent."
)

func GetEnabledSMSContacts(smsCfg config.SMSGlobalConfig) int {
	counter := 0
	for _, contact := range smsCfg.Contacts {
		if contact.Enabled {
			counter++
		}
	}
	return counter
}

func SMSSendToAll(message string, cfg config.Config) { // return error here
	for _, contact := range cfg.SMS.Contacts {
		if contact.Enabled {
			err := SMSNotify(contact.Number, message, cfg)
			if err != nil {
				log.Printf("Unable to send SMS to %s.\n", contact.Name)
			}
		}
	}
}

func SMSGetNumberByName(name string, smsCfg config.SMSGlobalConfig) string {
	for _, contact := range smsCfg.Contacts {
		if contact.Name == name {
			return contact.Number
		}
	}
	return ErrSMSContactNotFound
}

func SMSNotify(to, message string, cfg config.Config) error {
	values := url.Values{}
	values.Set("action", "sendsms")
	values.Set("user", cfg.SMS.Username)
	values.Set("password", cfg.SMS.Password)
	values.Set("from", cfg.Name)
	values.Set("to", to)
	values.Set("text", message)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", SMSGLOBAL_API_URL, headers, strings.NewReader(values.Encode()))

	if err != nil {
		return err
	}

	if !common.StringContains(resp, "OK: 0; Sent queued message") {
		return errors.New(ErrSMSNotSent)
	}
	return nil
}
