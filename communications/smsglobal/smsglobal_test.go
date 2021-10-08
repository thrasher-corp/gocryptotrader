package smsglobal

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	cfg := &config.Config{Communications: base.CommunicationsConfig{}}
	commsCfg := cfg.GetCommunicationsConfig()
	s.Setup(&commsCfg)
}

func TestConnect(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	if err := s.Connect(); err != nil {
		t.Error(err)
	}
}

func TestPushEvent(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	err := s.PushEvent(base.Event{})
	if err != nil {
		t.Error("SMSGlobal PushEvent() error", err)
	}
}

func TestGetEnabledContacts(t *testing.T) {
	t.Parallel()
	s := SMSGlobal{
		Contacts: []Contact{
			{
				Name:    "test123",
				Enabled: true,
			},
		},
	}
	if v := s.GetEnabledContacts(); v != 1 {
		t.Error("expected one enabled contact")
	}
}

func TestGetContactByNumber(t *testing.T) {
	t.Parallel()
	s := SMSGlobal{
		Contacts: []Contact{
			{
				Name:    "test123",
				Enabled: true,
				Number:  "1337",
			},
		},
	}
	_, err := s.GetContactByNumber("1337")
	if err != nil {
		t.Error("SMSGlobal GetContactByNumber() error", err)
	}
	_, err = s.GetContactByNumber("basketball")
	if err == nil {
		t.Error("SMSGlobal GetContactByNumber() error")
	}
}

func TestGetContactByName(t *testing.T) {
	t.Parallel()
	s := SMSGlobal{
		Contacts: []Contact{
			{
				Name:    "test123",
				Enabled: true,
			},
		},
	}
	_, err := s.GetContactByName("test123")
	if err != nil {
		t.Error("SMSGlobal GetContactByName() error", err)
	}
	_, err = s.GetContactByName("blah")
	if err == nil {
		t.Error("SMSGlobal GetContactByName() error")
	}
}

func TestAddContact(t *testing.T) {
	t.Parallel()
	s := SMSGlobal{
		Contacts: []Contact{},
	}
	err := s.AddContact(Contact{Name: "bra", Number: "2876", Enabled: true})
	if err != nil {
		t.Error("SMSGlobal AddContact() error", err)
	}
	err = s.AddContact(Contact{Name: "bra", Number: "2876", Enabled: true})
	if err == nil {
		t.Error("SMSGlobal AddContact() error")
	}
	err = s.AddContact(Contact{Name: "", Number: "", Enabled: true})
	if err == nil {
		t.Error("SMSGlobal AddContact() error")
	}
	if len(s.Contacts) == 0 {
		t.Error("failed to add contacts")
	}
}

func TestRemoveContact(t *testing.T) {
	t.Parallel()
	s := SMSGlobal{
		Contacts: []Contact{
			{
				Name:    "test123",
				Enabled: true,
				Number:  "1337",
			},
		},
	}
	err := s.RemoveContact(Contact{Name: "test123", Number: "1337", Enabled: true})
	if err != nil {
		t.Error("SMSGlobal RemoveContact() error", err)
	}
	err = s.RemoveContact(Contact{Name: "frieda", Number: "243453", Enabled: true})
	if err == nil {
		t.Error("SMSGlobal RemoveContact() Expected error")
	}
}

func TestSendMessageToAll(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	err := s.SendMessageToAll("Hello,World!")
	if err != nil {
		t.Error("SMSGlobal SendMessageToAll() error", err)
	}
}

func TestSendMessage(t *testing.T) {
	t.Parallel()
	var s SMSGlobal
	err := s.SendMessage("1337", "Hello!")
	if err != nil {
		t.Error("SMSGlobal SendMessage() error", err)
	}
}
