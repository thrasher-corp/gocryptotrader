package slack

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	verificationToken = ""
)

var s Slack

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig(config.ConfigTestFile)

	s.Setup(cfg.GetCommunicationsConfig())
}

func TestConnect(t *testing.T) {
	err := s.Connect()
	if err == nil {
		t.Error("test failed - slack Connect() error")
	}
}

func TestPushEvent(t *testing.T) {
	err := s.PushEvent(base.Event{})
	if err == nil {
		t.Error("test failed - slack PushEvent() error")
	}
}

func TestBuildURL(t *testing.T) {
	v := s.BuildURL("lol123")
	if v != "https://slack.com/api/rtm.start?token=lol123" {
		t.Error("test failed - slack BuildURL() error")
	}
}

func TestGetChannelsString(t *testing.T) {
	chans := s.GetChannelsString()
	if len(chans) != 0 {
		t.Error("test failed - slack GetChannelsString() error")
	}
}

func TestGetUsernameByID(t *testing.T) {
	username := s.GetUsernameByID("1337")
	if len(username) != 0 {
		t.Error("test failed - slack GetUsernameByID() error")
	}
}

func TestGetIDByName(t *testing.T) {
	id, err := s.GetIDByName("batman")
	if err == nil {
		t.Error("test failed - slack GetIDByName() error")
	}
	if len(id) != 0 {
		t.Error("test failed - slack GetIDByName() error")
	}
}

func TestGetChannelIDByName(t *testing.T) {
	id, err := s.GetChannelIDByName("1337")
	if err == nil {
		t.Error("test failed - slack GetChannelIDByName() error")
	}
	if len(id) != 0 {
		t.Error("test failed - slack GetChannelIDByName() error")
	}
}

func TestGetUsersInGroup(t *testing.T) {
	username := s.GetUsersInGroup("supergroup")
	if len(username) != 0 {
		t.Error("test failed - slack GetUsersInGroup() error")
	}
}

func TestNewConnection(t *testing.T) {
	err := s.NewConnection()
	if err == nil {
		t.Error("test failed - slack NewConnection() error")
	}
}
