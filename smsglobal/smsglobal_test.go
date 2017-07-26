package smsglobal

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

func TestGetEnabledSMSContacts(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test Failed. GetEnabledSMSContacts: Function return is incorrect with, %s.",
			err,
		)
	}
	numberOfContacts := GetEnabledSMSContacts(cfg.SMS)
	if numberOfContacts != len(cfg.SMS.Contacts) {
		t.Errorf(
			"Test Failed. GetEnabledSMSContacts: Function return is incorrect with, %d.",
			numberOfContacts,
		)
	}
}

func TestSMSSendToAll(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test Failed. SMSSendToAll: \nFunction return is incorrect with, %s.",
			err,
		)
	}
	SMSSendToAll("SMSGLOBAL Test - SMSSENDTOALL", *cfg)
}

func TestSMSGetNumberByName(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test Failed. SMSGetNumberByName: Function return is incorrect with, %s.",
			err,
		)
	}
	number := SMSGetNumberByName("StyleGherkin", cfg.SMS)
	if number == "" {
		t.Error("Test Failed. SMSNotify Error: No number, name not found.")
	}
	number = SMSGetNumberByName("testy", cfg.SMS)
	if number == "" {
		t.Error("Test Failed. SMSNotify Error: No number, name not found.")
	}
}

func TestSMSNotify(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Errorf(
			"Test Failed. SMSNotify: \nFunction return is incorrect with, %s.",
			err,
		)
	}
	// err2 := SMSNotify("+61312112718", "teststring", *cfg)
	// if err2 != nil {
	// 	t.Error("Test Failed. SMSNotify: \nError: ", err2)
	// }
}
