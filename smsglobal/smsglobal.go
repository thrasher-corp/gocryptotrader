package smsglobal

import (
	"errors"
	"flag"
	"net/url"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	smsGlobalAPIURL = "https://www.smsglobal.com/http-api.php"
	// ErrSMSContactNotFound is a general error code for "SMS Contact not found."
	ErrSMSContactNotFound = "SMS Contact not found."
	errSMSNotSent         = "SMS message not sent."
)

// vars for the SMS global package
var (
	SMSGlobal *Base
)

// Contact struct stores information related to a SMSGlobal contact
type Contact struct {
	Name    string `json:"name"`
	Number  string `json:"number"`
	Enabled bool   `json:"enabled"`
}

// Base struct stores information related to the SMSGlobal package
type Base struct {
	Contacts []Contact `json:"contacts"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	SendFrom string    `json:"send_from"`
}

// New initialises the SMSGlobal var
func New(username, password, sendFrom string, contacts []Contact) *Base {
	if username == "" || password == "" || sendFrom == "" || len(contacts) == 0 {
		return nil
	}

	var goodContacts []Contact
	for x := range contacts {
		if contacts[x].Name != "" || contacts[x].Number != "" {
			goodContacts = append(goodContacts, contacts[x])
		}
	}

	SMSGlobal = &Base{
		Contacts: goodContacts,
		Username: username,
		Password: password,
		SendFrom: sendFrom,
	}
	return SMSGlobal
}

// GetEnabledContacts returns how many SMS contacts are enabled in the
// contact list
func (s *Base) GetEnabledContacts() int {
	counter := 0
	for x := range s.Contacts {
		if s.Contacts[x].Enabled {
			counter++
		}
	}
	return counter
}

// GetContactByNumber returns a contact with supplied number
func (s *Base) GetContactByNumber(number string) (Contact, error) {
	for x := range s.Contacts {
		if s.Contacts[x].Number == number {
			return s.Contacts[x], nil
		}
	}
	return Contact{}, errors.New(ErrSMSContactNotFound)
}

// GetContactByName returns a contact with supplied name
func (s *Base) GetContactByName(name string) (Contact, error) {
	for x := range s.Contacts {
		if common.StringToLower(s.Contacts[x].Name) == common.StringToLower(name) {
			return s.Contacts[x], nil
		}
	}
	return Contact{}, errors.New(ErrSMSContactNotFound)
}

// AddContact checks to see if a contact exists and adds them if it doesn't
func (s *Base) AddContact(contact Contact) {
	if contact.Name == "" || contact.Number == "" {
		return
	}

	if s.ContactExists(contact) {
		return
	}

	s.Contacts = append(s.Contacts, contact)
}

// ContactExists checks to see if a contact exists
func (s *Base) ContactExists(contact Contact) bool {
	for x := range s.Contacts {
		if s.Contacts[x].Number == contact.Number && common.StringToLower(s.Contacts[x].Name) == common.StringToLower(contact.Name) {
			return true
		}
	}
	return false
}

// RemoveContact removes a contact if it exists
func (s *Base) RemoveContact(contact Contact) {
	if !s.ContactExists(contact) {
		return
	}

	for x := range s.Contacts {
		if s.Contacts[x].Name == contact.Name && s.Contacts[x].Number == contact.Number {
			s.Contacts = append(s.Contacts[:x], s.Contacts[x+1:]...)
			return
		}
	}
}

// SendMessageToAll sends a message to all enabled contacts in cfg
func (s *Base) SendMessageToAll(message string) {
	for x := range s.Contacts {
		if s.Contacts[x].Enabled {
			s.SendMessage(s.Contacts[x].Name, message)
		}
	}
}

// SendMessage sends a message to an individual contact
func (s *Base) SendMessage(to, message string) error {
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
