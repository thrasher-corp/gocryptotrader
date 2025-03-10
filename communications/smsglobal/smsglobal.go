// Package smsglobal allows bulk messaging to a desired recipient list
package smsglobal

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	smsGlobalAPIURL = "https://www.smsglobal.com/http-api.php"
)

var (
	errSMSNotSent      = errors.New("SMSGlobal message not sent")
	errContactNotFound = errors.New("SMSGlobal error contact not found")
)

// SMSGlobal is the overarching type across this package
type SMSGlobal struct {
	base.Base
	Contacts []Contact
	Username string
	Password string
	SendFrom string
}

// Setup takes in a SMSGlobal configuration, sets username, password and
// recipient list
func (s *SMSGlobal) Setup(cfg *base.CommunicationsConfig) {
	s.Name = cfg.SMSGlobalConfig.Name
	s.Enabled = cfg.SMSGlobalConfig.Enabled
	s.Verbose = cfg.SMSGlobalConfig.Verbose
	s.Username = cfg.SMSGlobalConfig.Username
	s.Password = cfg.SMSGlobalConfig.Password
	s.SendFrom = cfg.SMSGlobalConfig.From

	contacts := make([]Contact, len(cfg.SMSGlobalConfig.Contacts))
	for x := range cfg.SMSGlobalConfig.Contacts {
		contacts[x] = Contact{
			Name:    cfg.SMSGlobalConfig.Contacts[x].Name,
			Number:  cfg.SMSGlobalConfig.Contacts[x].Number,
			Enabled: cfg.SMSGlobalConfig.Contacts[x].Enabled,
		}
		log.Debugf(log.CommunicationMgr, "SMSGlobal: SMS Contact: %s. Number: %s. Enabled: %v\n",
			cfg.SMSGlobalConfig.Contacts[x].Name,
			cfg.SMSGlobalConfig.Contacts[x].Number,
			cfg.SMSGlobalConfig.Contacts[x].Enabled)
	}
	s.Contacts = contacts
}

// IsConnected returns whether or not the connection is connected
func (s *SMSGlobal) IsConnected() bool {
	return s.Connected
}

// Connect connects to the service
func (s *SMSGlobal) Connect() error {
	s.Connected = true
	return nil
}

// PushEvent pushes an event to a contact list via SMS
func (s *SMSGlobal) PushEvent(event base.Event) error {
	return s.SendMessageToAll(event.Message)
}

// GetEnabledContacts returns how many SMS contacts are enabled in the
// contact list
func (s *SMSGlobal) GetEnabledContacts() int {
	counter := 0
	for x := range s.Contacts {
		if s.Contacts[x].Enabled {
			counter++
		}
	}
	return counter
}

// GetContactByNumber returns a contact with supplied number
func (s *SMSGlobal) GetContactByNumber(number string) (Contact, error) {
	for x := range s.Contacts {
		if s.Contacts[x].Number == number {
			return s.Contacts[x], nil
		}
	}
	return Contact{}, errContactNotFound
}

// GetContactByName returns a contact with supplied name
func (s *SMSGlobal) GetContactByName(name string) (Contact, error) {
	for x := range s.Contacts {
		if strings.EqualFold(s.Contacts[x].Name, name) {
			return s.Contacts[x], nil
		}
	}
	return Contact{}, errContactNotFound
}

// AddContact checks to see if a contact exists and adds them if it doesn't
func (s *SMSGlobal) AddContact(contact Contact) error {
	if contact.Name == "" || contact.Number == "" {
		return errors.New("SMSGlobal AddContact() error - nothing to add")
	}

	if s.ContactExists(contact) {
		return errors.New("SMSGlobal AddContact() error - contact already exists")
	}

	s.Contacts = append(s.Contacts, contact)
	return nil
}

// ContactExists checks to see if a contact exists
func (s *SMSGlobal) ContactExists(contact Contact) bool {
	for x := range s.Contacts {
		if s.Contacts[x].Number == contact.Number && strings.EqualFold(s.Contacts[x].Name, contact.Name) {
			return true
		}
	}
	return false
}

// RemoveContact removes a contact if it exists
func (s *SMSGlobal) RemoveContact(contact Contact) error {
	if !s.ContactExists(contact) {
		return errors.New("SMSGlobal RemoveContact() error - contact does not exist")
	}

	for x := range s.Contacts {
		if s.Contacts[x].Name == contact.Name && s.Contacts[x].Number == contact.Number {
			s.Contacts = slices.Delete(s.Contacts, x, x+1)
			return nil
		}
	}
	return errors.New("SMSGlobal RemoveContact() error - contact already removed")
}

// SendMessageToAll sends a message to all enabled contacts in cfg
func (s *SMSGlobal) SendMessageToAll(message string) error {
	for x := range s.Contacts {
		if s.Contacts[x].Enabled {
			if s.Verbose {
				log.Debugf(log.CommunicationMgr, "SMSGlobal: Sending SMS to %s. Number: %s. Message: %s [From: %s]\n",
					s.Contacts[x].Name, s.Contacts[x].Number, message, s.SendFrom)
			}
			err := s.SendMessage(s.Contacts[x].Number, message)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SendMessage sends a message to an individual contact
func (s *SMSGlobal) SendMessage(to, message string) error {
	if flag.Lookup("test.v") != nil {
		return nil
	}

	values := url.Values{}
	values.Set("action", "sendsms")
	values.Set("user", s.Username)
	values.Set("password", s.Password)
	values.Set("from", s.SendFrom)
	values.Set("to", to)
	values.Set("text", message)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest(context.TODO(),
		http.MethodPost,
		smsGlobalAPIURL,
		headers,
		strings.NewReader(values.Encode()),
		s.Verbose)
	if err != nil {
		return err
	}

	if !strings.Contains(string(resp), "OK: 0; Sent queued message") {
		return errSMSNotSent
	}
	return nil
}
