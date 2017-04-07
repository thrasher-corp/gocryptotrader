package test

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/smsglobal"
)

func TestGetEnabledSMSContacts(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig()
	if err != nil {
		t.Errorf("Test Failed. GetEnabledSMSContacts: \nFunction return is incorrect with, %s.", err)
	}

	numberOfContacts := smsglobal.GetEnabledSMSContacts(cfg.SMS)
	if numberOfContacts != len(cfg.SMS.Contacts) {
		t.Errorf("Test Failed. GetEnabledSMSContacts: \nFunction return is incorrect with, %d.", numberOfContacts)
	}
}

func TestSMSSendToAll(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig()
	if err != nil {
		t.Errorf("Test Failed. SMSSendToAll: \nFunction return is incorrect with, %s.", err)
	}

	smsglobal.SMSSendToAll("Test", *cfg) //+60sec reply issue without account details
}

func TestSMSGetNumberByName(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig()
	if err != nil {
		t.Errorf("Test Failed. SMSGetNumberByName: \nFunction return is incorrect with, %s.", err)
	}
	number := smsglobal.SMSGetNumberByName("POLYESTERGIRL", cfg.SMS)
	if number == "" {
		t.Error("Test Failed. SMSNotify: \nError: No number, name not found.")
	}
}

func TestSMSNotify(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig()
	if err != nil {
		t.Errorf("Test Failed. SMSNotify: \nFunction return is incorrect with, %s.", err)
	}

	err2 := smsglobal.SMSNotify(cfg.SMS.Contacts[0].Number, "Test", *cfg)
	if err2 != nil {
		t.Error("Test Failed. SMSNotify: \nError: ", err2)
	}
}
