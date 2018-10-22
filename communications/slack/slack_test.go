package slack

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	verificationToken = ""
)

var s Slack

type group struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	IsGroup        bool     `json:"is_group"`
	Created        int64    `json:"created"`
	Creator        string   `json:"creator"`
	IsArchived     bool     `json:"is_archived"`
	NameNormalised string   `json:"name_normalised"`
	IsMPIM         bool     `json:"is_mpim"`
	HasPins        bool     `json:"has_pins"`
	IsOpen         bool     `json:"is_open"`
	LastRead       string   `json:"last_read"`
	Members        []string `json:"members"`
	Topic          struct {
		Value   string `json:"value"`
		Creator string `json:"creator"`
		LastSet int64  `json:"last_set"`
	} `json:"topic"`
	Purpose struct {
		Value   string `json:"value"`
		Creator string `json:"creator"`
		LastSet int64  `json:"last_set"`
	} `json:"purpose"`
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig(config.ConfigTestFile)

	s.Setup(cfg.GetCommunicationsConfig())
	s.Verbose = true
}

func TestConnect(t *testing.T) {
	err := s.Connect()
	if err == nil {
		t.Error("test failed - slack Connect() error")
	}
}

func TestPushEvent(t *testing.T) {
	t.Parallel()
	err := s.PushEvent(base.Event{})
	if err == nil {
		t.Error("test failed - slack PushEvent() error")
	}
}

func TestBuildURL(t *testing.T) {
	t.Parallel()
	v := s.BuildURL("lol123")
	if v != "https://slack.com/api/rtm.start?token=lol123" {
		t.Error("test failed - slack BuildURL() error")
	}
}

func TestGetChannelsString(t *testing.T) {

	s.Details.Channels = append(s.Details.Channels, struct {
		Created        int      `json:"created"`
		Creator        string   `json:"creator"`
		HasPins        bool     `json:"has_pins"`
		ID             string   `json:"id"`
		IsArchived     bool     `json:"is_archived"`
		IsChannel      bool     `json:"is_channel"`
		IsGeneral      bool     `json:"is_general"`
		IsMember       bool     `json:"is_member"`
		IsOrgShared    bool     `json:"is_org_shared"`
		IsShared       bool     `json:"is_shared"`
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
	if testpassed == false {
		t.Error("test failed - slack GetChannelsString() error")
	}
}

func TestGetUsernameByID(t *testing.T) {
	username := s.GetUsernameByID("1337")
	if len(username) != 0 {
		t.Error("test failed - slack GetUsernameByID() error")
	}

	s.Details.Users = append(s.Details.Users, struct {
		Deleted  bool   `json:"deleted"`
		ID       string `json:"id"`
		IsBot    bool   `json:"is_bot"`
		Name     string `json:"name"`
		Presence string `json:"presence"`
		Profile  struct {
			AvatarHash         string      `json:"avatar_hash"`
			Email              string      `json:"email"`
			Fields             interface{} `json:"fields"`
			FirstName          string      `json:"first_name"`
			Image192           string      `json:"image_192"`
			Image24            string      `json:"image_24"`
			Image32            string      `json:"image_32"`
			Image48            string      `json:"image_48"`
			Image512           string      `json:"image_512"`
			Image72            string      `json:"image_72"`
			LastName           string      `json:"last_name"`
			RealName           string      `json:"real_name"`
			RealNameNormalized string      `json:"real_name_normalized"`
		} `json:"profile"`
		TeamID  string `json:"team_id"`
		Updated int    `json:"updated"`
	}{
		ID:   "1337",
		Name: "cranktakular",
	})

	username = s.GetUsernameByID("1337")
	if username != "cranktakular" {
		t.Error("test failed - slack GetUsernameByID() error")
	}

}

func TestGetIDByName(t *testing.T) {
	id, err := s.GetIDByName("batman")
	if err == nil || len(id) != 0 {
		t.Error("test failed - slack GetIDByName() error")
	}

	s.Details.Groups = append(s.Details.Groups, group{
		Name: "this is a group",
		ID:   "210314",
	})
	id, err = s.GetIDByName("this is a group")
	if err != nil || id != "210314" {
		t.Errorf("test failed - slack GetIDByName() Expected '210314' Actual '%s' Error: %s",
			id, err)
	}
}

func TestGetGroupIDByName(t *testing.T) {
	id, err := s.GetGroupIDByName("batman")
	if err == nil || len(id) != 0 {
		t.Error("test failed - slack GetGroupIDByName() error")
	}

	s.Details.Groups = append(s.Details.Groups, group{
		Name: "another group",
		ID:   "11223344",
	})
	id, err = s.GetGroupIDByName("another group")
	if err != nil || id != "11223344" {
		t.Errorf("test failed - slack GetGroupIDByName() Expected '11223344' Actual '%s' Error: %s",
			id, err)
	}

}

func TestGetChannelIDByName(t *testing.T) {
	id, err := s.GetChannelIDByName("1337")
	if err == nil || len(id) != 0 {
		t.Error("test failed - slack GetChannelIDByName() error")
	}

	s.Details.Channels = append(s.Details.Channels, struct {
		Created        int      `json:"created"`
		Creator        string   `json:"creator"`
		HasPins        bool     `json:"has_pins"`
		ID             string   `json:"id"`
		IsArchived     bool     `json:"is_archived"`
		IsChannel      bool     `json:"is_channel"`
		IsGeneral      bool     `json:"is_general"`
		IsMember       bool     `json:"is_member"`
		IsOrgShared    bool     `json:"is_org_shared"`
		IsShared       bool     `json:"is_shared"`
		Name           string   `json:"name"`
		NameNormalized string   `json:"name_normalized"`
		PreviousNames  []string `json:"previous_names"`
	}{
		ID:   "2048",
		Name: "Slack Test",
	})

	id, err = s.GetChannelIDByName("Slack Test")
	if err != nil || id != "2048" {
		t.Errorf("test failed - slack GetChannelIDByName() Expected '2048' Actual '%s' Error: %s",
			id, err)
	}
}

func TestGetUsersInGroup(t *testing.T) {
	username := s.GetUsersInGroup("supergroup")
	if len(username) != 0 {
		t.Error("test failed - slack GetUsersInGroup() error")
	}

	s.Details.Groups = append(s.Details.Groups, group{
		Name:    "three guys",
		ID:      "3",
		Members: []string{"Guy one", "Guy two", "Guy three"},
	})

	username = s.GetUsersInGroup("three guys")
	if len(username) != 3 {
		t.Errorf("test failed - slack GetUsersInGroup() Expected '3' Actual '%s'",
			username)
	}
}

func TestNewConnection(t *testing.T) {
	err := s.NewConnection()
	if err == nil {
		t.Error("test failed - slack NewConnection() error")
	}
}

func TestWebsocketConnect(t *testing.T) {
	err := s.WebsocketConnect()
	if err == nil {
		t.Error("test failed - slack WebsocketConnect() error")
	}
}

func TestHandlePresenceChange(t *testing.T) {
	var pres PresenceChange
	pres.User = "1337"
	pres.Presence = "Present"

	err := s.handlePresenceChange([]byte(`{"malformedjson}`))
	if err == nil {
		t.Error("test failed - slack handlePresenceChange(), unmarshalled malformed json")
	}

	data, _ := common.JSONEncode(pres)
	err = s.handlePresenceChange(data)
	if err != nil {
		t.Errorf("test failed - slack handlePresenceChange() Error: %s", err)
	}
}

func TestHandleMessageResponse(t *testing.T) {
	var data WebsocketResponse
	data.ReplyTo = 1

	err := s.handleMessageResponse(nil, data)
	if err.Error() != "ReplyTo != 0" {
		t.Errorf("test failed - slack handleMessageResponse(), Incorrect Error: %s",
			err)
	}

	data.ReplyTo = 0

	err = s.handleMessageResponse([]byte(`{"malformedjson}`), data)
	if err == nil {
		t.Error("test failed - slack handleMessageResponse(), unmarshalled malformed json")
	}

	var msg Message
	msg.User = "1337"
	msg.Text = "Hello World!"
	resp, _ := common.JSONEncode(msg)

	err = s.handleMessageResponse(resp, data)
	if err != nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}

	msg.Text = "!notacommand"
	resp, _ = common.JSONEncode(msg)

	err = s.handleMessageResponse(resp, data)
	if err == nil {
		t.Errorf("test failed - slack handleMessageResponse() Error: %s", err)
	}
}

func TestHandleErrorResponse(t *testing.T) {
	var data WebsocketResponse
	err := s.handleErrorResponse(data)
	if err == nil {
		t.Error("test failed - slack handleErrorResponse() Ignored strange input")
	}

	data.Error.Msg = "Socket URL has expired"
	err = s.handleErrorResponse(data)
	if err == nil {
		t.Error("test failed - slack handleErrorResponse() Didn't error on nil websocket")
	}
}

func TestHandleHelloResponse(t *testing.T) {
	var data WebsocketResponse
	s.handleHelloResponse(data)
}

func TestHandleReconnectResponse(t *testing.T) {

	err := s.handleReconnectResponse([]byte(`{"malformedjson}`))
	if err == nil {
		t.Error("test failed - slack handleReconnectResponse(), unmarshalled malformed json")
	}

	var testURL struct {
		URL string `json:"url"`
	}
	testURL.URL = "https://www.thrasher.io"

	data, _ := common.JSONEncode(testURL)

	err = s.handleReconnectResponse(data)
	if err != nil || s.ReconnectURL != "https://www.thrasher.io" {
		t.Errorf("test failed - slack handleReconnectResponse() Expected 'https://www.thrasher.io' Actual '%s' Error: %s",
			s.ReconnectURL, err)
	}
}

func TestWebsocketSend(t *testing.T) {
	err := s.WebsocketSend("test", "Hello World!")
	if err == nil {
		t.Error("test failed - slack WebsocketSend(), Sent message through nil websocket")
	}
}

func TestHandleMessage(t *testing.T) {
	var msg Message

	err := s.HandleMessage(msg)
	if err == nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdStatus
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdHelp
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdTicker
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdOrderbook
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdSettings
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}
	msg.Text = cmdPortfolio
	err = s.HandleMessage(msg)
	if err == nil {
		t.Error("test failed - slack HandleMessage(), Sent message through nil websocket")
	}
}
