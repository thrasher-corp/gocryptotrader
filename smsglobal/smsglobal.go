package smsglobal

import (
	"errors"
	"flag"
	"log"
	"net/url"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	smsGlobalAPIURL = "http://www.smsglobal.com/http-api.php"
	// ErrSMSContactNotFound is a general error code for "SMS Contact not found."
	ErrSMSContactNotFound = "SMS Contact not found."
	errSMSNotSent         = "SMS message not sent."
)

// GetEnabledSMSContacts returns how many SMS contacts are enabled in the
// contacts list.
func GetEnabledSMSContacts(smsCfg config.SMSGlobalConfig) int {
	counter := 0
	for _, contact := range smsCfg.Contacts {
		if contact.Enabled {
			counter++
		}
	}
	return counter
}

// SMSSendToAll sends a message to all enabled contacts in cfg
func SMSSendToAll(message string, cfg config.Config) {
	for _, contact := range cfg.SMS.Contacts {
		if contact.Enabled && len(contact.Number) == 10 {
			err := SMSNotify(contact.Number, message, cfg)
			if err != nil {
				log.Printf("Unable to send SMS to %s.\n", contact.Name)
			}
		}
	}
}

// SMSGetNumberByName returns contact number by supplied name
func SMSGetNumberByName(name string, smsCfg config.SMSGlobalConfig) string {
	for _, contact := range smsCfg.Contacts {
		if common.StringToUpper(contact.Name) == common.StringToUpper(name) {
			return contact.Number
		}
	}
	return ErrSMSContactNotFound
}

// SMSNotify sends a message to an individual contact
func SMSNotify(to, message string, cfg config.Config) error {
	if flag.Lookup("test.v") != nil {
		return nil
	}

	values := url.Values{}
	values.Set("action", "sendsms")
	values.Set("user", cfg.SMS.Username)
	values.Set("password", cfg.SMS.Password)
	values.Set("from", cfg.Name)
	values.Set("to", to)
	values.Set("text", message)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest(
		"POST", smsGlobalAPIURL, headers, strings.NewReader(values.Encode()),
	)

	if err != nil {
		return err
	}

	if !common.StringContains(resp, "OK: 0; Sent queued message") {
		return errors.New(errSMSNotSent)
	}
	return nil
}
