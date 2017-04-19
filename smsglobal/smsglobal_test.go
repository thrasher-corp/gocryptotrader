package smsglobal

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

func TestGetEnabledSMSContacts(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.dat")
	if err != nil {
		t.Errorf("Test Failed. GetEnabledSMSContacts: \nFunction return is incorrect with, %s.", err)
	}

	numberOfContacts := GetEnabledSMSContacts(cfg.SMS)
	if numberOfContacts != len(cfg.SMS.Contacts) {
		t.Errorf("Test Failed. GetEnabledSMSContacts: \nFunction return is incorrect with, %d.", numberOfContacts)
	}
}

func TestSMSSendToAll(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.dat")
	if err != nil {
		t.Errorf("Test Failed. SMSSendToAll: \nFunction return is incorrect with, %s.", err)
	}

	SMSSendToAll("SMSGLOBAL Test - SMSSENDTOALL", *cfg) //+60sec reply issue without account details
}

func TestSMSGetNumberByName(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.dat")
	if err != nil {
		t.Errorf("Test Failed. SMSGetNumberByName: \nFunction return is incorrect with, %s.", err)
	}
	number := SMSGetNumberByName("StyleGherkin", cfg.SMS)
	if number == "" {
		t.Error("Test Failed. SMSNotify: \nError: No number, name not found.")
	}
}

func TestSMSNotify(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../testdata/configtest.dat")
	if err != nil {
		t.Errorf("Test Failed. SMSNotify: \nFunction return is incorrect with, %s.", err)
	}

	err2 := SMSNotify(cfg.SMS.Contacts[0].Number, "SMSGLOBAL Test - SMS SEND TO SINGLE", *cfg)
	if err2 != nil {
		t.Error("Test Failed. SMSNotify: \nError: ", err2)
	}
}
