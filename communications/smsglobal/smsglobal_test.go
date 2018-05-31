package smsglobal

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
)

var s SMSGlobal

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	s.Setup(cfg.GetCommunicationsConfig())
}

func TestConnect(t *testing.T) {
	err := s.Connect()
	if err != nil {
		t.Error("test failed - SMSGlobal Connect() error")
	}
}

func TestPushEvent(t *testing.T) {
	err := s.PushEvent(base.Event{})
	if err == nil {
		t.Error("test failed - SMSGlobal PushEvent() error")
	}
}

func TestGetEnabledContacts(t *testing.T) {
	v := s.GetEnabledContacts()
	if v != 2 {
		t.Error("test failed - SMSGlobal GetEnabledContacts() error")
	}
}

func TestGetContactByNumber(t *testing.T) {
	_, err := s.GetContactByNumber("1234")
	if err != nil {
		t.Error("test failed - SMSGlobal GetContactByNumber() error", err)
	}
	_, err = s.GetContactByNumber("basketball")
	if err == nil {
		t.Error("test failed - SMSGlobal GetContactByNumber() error")
	}
}

func TestGetContactByName(t *testing.T) {
	_, err := s.GetContactByName("bob")
	if err != nil {
		t.Error("test failed - SMSGlobal GetContactByName() error", err)
	}
	_, err = s.GetContactByName("blah")
	if err == nil {
		t.Error("test failed - SMSGlobal GetContactByName() error")
	}
}

func TestAddContact(t *testing.T) {
	err := s.AddContact(Contact{Name: "bra", Number: "2876", Enabled: true})
	if err != nil {
		t.Error("test failed - SMSGlobal AddContact() error", err)
	}
	err = s.AddContact(Contact{Name: "bob", Number: "1234", Enabled: true})
	if err == nil {
		t.Error("test failed - SMSGlobal AddContact() error")
	}
	err = s.AddContact(Contact{Name: "", Number: "", Enabled: true})
	if err == nil {
		t.Error("test failed - SMSGlobal AddContact() error")
	}
}

func TestRemoveContact(t *testing.T) {
	err := s.RemoveContact(Contact{Name: "bob", Number: "1234", Enabled: true})
	if err != nil {
		t.Error("test failed - SMSGlobal RemoveContact() error", err)
	}
	err = s.RemoveContact(Contact{Name: "frieda", Number: "243453", Enabled: true})
	if err == nil {
		t.Error("test failed - SMSGlobal RemoveContact() error", err)
	}
}

func TestSendMessageToAll(t *testing.T) {
	err := s.SendMessageToAll("Hello,World!")
	if err != nil {
		t.Error("test failed - SMSGlobal SendMessageToAll() error", err)
	}
}

func TestSendMessage(t *testing.T) {
	err := s.SendMessage("1337", "Hello!")
	if err != nil {
		t.Error("test failed - SMSGlobal SendMessage() error", err)
	}
}
