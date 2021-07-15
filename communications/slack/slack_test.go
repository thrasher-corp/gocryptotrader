package slack

import (
	"encoding/json"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
)

type group struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

func TestSetup(t *testing.T) {
	t.Parallel()
	var s Slack
	cfg := &config.Config{Communications: base.CommunicationsConfig{}}
	commsCfg := cfg.GetCommunicationsConfig()
	s.Setup(&commsCfg)
}

func TestConnect(t *testing.T) {
	t.Parallel()
	var s Slack
	err := s.Connect()
	if err == nil {
		t.Error("slack Connect() error cannot be nil")
	}
}

func TestPushEvent(t *testing.T) {
	t.Parallel()
	var s Slack
	err := s.PushEvent(base.Event{})
	if err == nil {
		t.Error("slack PushEvent() error cannot be nil")
	}
}

func TestBuildURL(t *testing.T) {
	t.Parallel()
	var s Slack
	v := s.BuildURL("lol123")
	if v != "https://slack.com/api/rtm.start?token=lol123" {
		t.Error("slack BuildURL() error")
	}
}

func TestGetChannelsString(t *testing.T) {
	t.Parallel()
	var s Slack
	s.Details.Channels = append(s.Details.Channels, struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		NameNormalized string   `json:"name_normalized"`
		PreviousNames  []string `json:"previous_names"`
	}{
		NameNormalized: "General",
	})

	chans := s.GetChannelsString()
	testpassed := false
	for i := range chans {
		if chans[i] == "General" {
			testpassed = true
		}
	}
	if !testpassed {
		t.Error("slack GetChannelsString() error")
	}
}

func TestGetUsernameByID(t *testing.T) {
	t.Parallel()
	var s Slack
	username := s.GetUsernameByID("1337")
	if username != "" {
		t.Error("slack GetUsernameByID() error")
	}

	s.Details.Users = append(s.Details.Users, struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		TeamID string `json:"team_id"`
	}{
		ID:   "1337",
		Name: "cranktakular",
	})

	username = s.GetUsernameByID("1337")
	if username != "cranktakular" {
		t.Error("slack GetUsernameByID() error")
	}
}

func TestGetIDByName(t *testing.T) {
	t.Parallel()
	var s Slack
	id, err := s.GetIDByName("batman")
	if err == nil || id != "" {
		t.Error("slack GetIDByName() error")
	}

	s.Details.Groups = append(s.Details.Groups, group{
		Name: "this is a group",
		ID:   "210314",
	})
	id, err = s.GetIDByName("this is a group")
	if err != nil || id != "210314" {
		t.Errorf("slack GetIDByName() Expected '210314' Actual '%s' Error: %s",
			id, err)
	}
}

func TestGetGroupIDByName(t *testing.T) {
	t.Parallel()
	var s Slack
	id, err := s.GetGroupIDByName("batman")
	if err == nil || id != "" {
		t.Error("slack GetGroupIDByName() error")
	}

	s.Details.Groups = append(s.Details.Groups, group{
		Name: "another group",
		ID:   "11223344",
	})
	id, err = s.GetGroupIDByName("another group")
	if err != nil || id != "11223344" {
		t.Errorf("slack GetGroupIDByName() Expected '11223344' Actual '%s' Error: %s",
			id, err)
	}
}

func TestGetChannelIDByName(t *testing.T) {
	t.Parallel()
	var s Slack
	id, err := s.GetChannelIDByName("1337")
	if err == nil || id != "" {
		t.Error("slack GetChannelIDByName() error")
	}

	s.Details.Channels = append(s.Details.Channels, struct {
		ID             string   `json:"id"`
		Name           string   `json:"name"`
		NameNormalized string   `json:"name_normalized"`
		PreviousNames  []string `json:"previous_names"`
	}{
		ID:   "2048",
		Name: "Slack Test",
	})

	id, err = s.GetChannelIDByName("Slack Test")
	if err != nil || id != "2048" {
		t.Errorf("slack GetChannelIDByName() Expected '2048' Actual '%s' Error: %s",
			id, err)
	}
}

func TestGetUsersInGroup(t *testing.T) {
	t.Parallel()
	var s Slack
	username := s.GetUsersInGroup("supergroup")
	if len(username) != 0 {
		t.Error("slack GetUsersInGroup() error")
	}

	s.Details.Groups = append(s.Details.Groups, group{
		Name:    "three guys",
		ID:      "3",
		Members: []string{"Guy one", "Guy two", "Guy three"},
	})

	username = s.GetUsersInGroup("three guys")
	if len(username) != 3 {
		t.Errorf("slack GetUsersInGroup() Expected '3' Actual '%s'",
			username)
	}
}

func TestNewConnection(t *testing.T) {
	t.Parallel()
	var s Slack
	err := s.NewConnection()
	if err == nil {
		t.Error("slack NewConnection() error")
	}
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	var s Slack
	err := s.WebsocketConnect()
	if err == nil {
		t.Error("slack WebsocketConnect() error")
	}
}

func TestHandlePresenceChange(t *testing.T) {
	t.Parallel()
	var s Slack
	var pres PresenceChange
	pres.User = "1337"
	pres.Presence = "Present"

	err := s.handlePresenceChange([]byte(`{"malformedjson}`))
	if err == nil {
		t.Error("slack handlePresenceChange(), unmarshalled malformed json")
	}

	data, _ := json.Marshal(pres)
	err = s.handlePresenceChange(data)
	if err != nil {
		t.Errorf("slack handlePresenceChange() Error: %s", err)
	}
}

func TestHandleMessageResponse(t *testing.T) {
	t.Parallel()
	var s Slack
	var data WebsocketResponse
	data.ReplyTo = 1

	err := s.handleMessageResponse(nil, data)
	if err.Error() != "reply to is != 0" {
		t.Errorf("slack handleMessageResponse(), Incorrect Error: %s",
			err)
	}

	data.ReplyTo = 0

	err = s.handleMessageResponse([]byte(`{"malformedjson}`), data)
	if err == nil {
		t.Error("slack handleMessageResponse(), unmarshalled malformed json")
	}

	var msg Message
	msg.User = "1337"
	msg.Text = "Hello World!"
	resp, _ := json.Marshal(msg)

	err = s.handleMessageResponse(resp, data)
	if err != nil {
		t.Error("slack HandleMessage(), Sent message through nil websocket")
	}

	msg.Text = "!notacommand"
	resp, _ = json.Marshal(msg)

	err = s.handleMessageResponse(resp, data)
	if err == nil {
		t.Errorf("slack handleMessageResponse() Expected error")
	}
}

func TestHandleErrorResponse(t *testing.T) {
	t.Parallel()
	var s Slack
	var data WebsocketResponse
	err := s.handleErrorResponse(data)
	if err == nil {
		t.Error("slack handleErrorResponse() Ignored strange input")
	}

	data.Error.Msg = "Socket URL has expired"
	err = s.handleErrorResponse(data)
	if err == nil {
		t.Error("slack handleErrorResponse() Didn't error on nil websocket")
	}
}

func TestHandleHelloResponse(t *testing.T) {
	t.Parallel()
	var s Slack
	s.handleHelloResponse()
}

func TestHandleReconnectResponse(t *testing.T) {
	t.Parallel()
	var s Slack
	err := s.handleReconnectResponse([]byte(`{"malformedjson}`))
	if err == nil {
		t.Error("slack handleReconnectResponse(), unmarshalled malformed json")
	}

	var testURL struct {
		URL string `json:"url"`
	}
	testURL.URL = "https://www.thrasher.io"

	data, _ := json.Marshal(testURL)

	err = s.handleReconnectResponse(data)
	if err != nil || s.ReconnectURL != "https://www.thrasher.io" {
		t.Errorf("slack handleReconnectResponse() Expected 'https://www.thrasher.io' Actual '%s' Error: %s",
			s.ReconnectURL, err)
	}
}

func TestWebsocketSend(t *testing.T) {
	t.Parallel()
	var s Slack
	err := s.WebsocketSend("test", "Hello World!")
	if err == nil {
		t.Error("slack WebsocketSend(), Sent message through nil websocket")
	}
}

func TestHandleMessage(t *testing.T) {
	t.Parallel()
	var s Slack
	msg := &Message{}
	err := s.HandleMessage(msg)
	if err == nil {
		t.Error("slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdStatus
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdHelp
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("slack HandleMessage(), Sent message through nil websocket")
	}
}
