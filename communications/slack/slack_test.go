package slack

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	if err := s.Connect(); err == nil {
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
	if v := s.BuildURL("lol123"); v != "https://slack.com/api/rtm.start?token=lol123" {
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
	require.Error(t, err, "GetIDByName must return an error for non-existent name")
	assert.Empty(t, id, "GetIDByName should return an empty string for non-existent name")
	s.Details.Groups = append(s.Details.Groups, group{
		Name: "this is a group",
		ID:   "210314",
	})
	id, err = s.GetIDByName("this is a group")

	require.NoError(t, err, "GetIDByName must not return an error for existing name")
	assert.Equal(t, "210314", id, "GetIDByName should return the correct ID for existing name")
}

func TestGetGroupIDByName(t *testing.T) {
	t.Parallel()
	var s Slack
	id, err := s.GetGroupIDByName("batman")
	require.Error(t, err, "GetGroupIDByName must return an error for non-existent group name")
	assert.Empty(t, id, "GetGroupIDByName should return an empty string for non-existent group name")

	s.Details.Groups = append(s.Details.Groups, group{
		Name: "another group",
		ID:   "11223344",
	})
	id, err = s.GetGroupIDByName("another group")
	require.NoError(t, err, "GetGroupIDByName must not return an error for existing group name")
	assert.Equal(t, "11223344", id, "GetGroupIDByName should return the correct ID for existing group name")
}

func TestGetChannelIDByName(t *testing.T) {
	t.Parallel()
	var s Slack
	id, err := s.GetChannelIDByName("1337")
	require.Error(t, err, "GetChannelIDByName must return an error for non-existent channel name")
	assert.Empty(t, id, "GetChannelIDByName should return an empty string for non-existent channel name")

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
	require.NoError(t, err, "GetChannelIDByName must not return an error for existing channel name")
	assert.Equal(t, "2048", id, "GetChannelIDByName should return the correct ID for existing channel name")
}

func TestGetUsersInGroup(t *testing.T) {
	t.Parallel()
	var s Slack
	assert.Empty(t, s.GetUsersInGroup("supergroup"), "GetUsersInGroup should return an empty slice")

	s.Details.Groups = append(s.Details.Groups, group{
		Name:    "three guys",
		ID:      "3",
		Members: []string{"Guy one", "Guy two", "Guy three"},
	})

	assert.Len(t, s.GetUsersInGroup("three guys"), 3)
}

func TestNewConnection(t *testing.T) {
	t.Parallel()
	var s Slack
	if err := s.NewConnection(); err == nil {
		t.Error("slack NewConnection() error")
	}
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	var s Slack
	if err := s.WebsocketConnect(); err == nil {
		t.Error("slack WebsocketConnect() error")
	}
}

func TestHandlePresenceChange(t *testing.T) {
	t.Parallel()
	var s Slack
	var presChange PresenceChange
	presChange.User = "1337"
	presChange.Presence = "Present"

	err := s.handlePresenceChange([]byte(`{"malformedjson}`))
	if err == nil {
		t.Error("slack handlePresenceChange(), unmarshalled malformed json")
	}

	data, err := json.Marshal(presChange)
	if err != nil {
		t.Fatal(err)
	}
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
	resp, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	err = s.handleMessageResponse(resp, data)
	if err != nil {
		t.Error("slack HandleMessage(), Sent message through nil websocket")
	}

	msg.Text = "!notacommand"
	resp, err = json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

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
	assert.Error(t, err, "handleReconnectResponse should error on malformed JSON")

	testURL := struct {
		URL string `json:"url"`
	}{
		URL: "https://www.thrasher.io",
	}

	data, err := json.Marshal(testURL)
	require.NoError(t, err, "Marshal must not error")

	err = s.handleReconnectResponse(data)
	require.NoError(t, err, "handleReconnectResponse must not error")
	assert.Equal(t, "https://www.thrasher.io", s.ReconnectURL)
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
