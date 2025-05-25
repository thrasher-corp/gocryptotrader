// Package slack is used to connect to the slack network. Slack is a
// code-centric collaboration hub that allows users to connect via an app and
// share different types of data
package slack

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// const declares main slack url and commands that will be supported on client
// side
const (
	SlackURL = "https://slack.com/api/rtm.start"

	cmdStatus = "!status"
	cmdHelp   = "!help"

	getHelp = `GoCryptoTrader SlackBot, thank you for using this service!
	Current commands are:
	!status 		- Displays current working status of bot
	!help 			- Displays help text`
)

// Slack starts a websocket connection and uses https://api.slack.com/rtm real
// time messaging
type Slack struct {
	base.Base

	TargetChannel     string
	VerificationToken string

	TargetChannelID string
	Details         Response
	ReconnectURL    string
	WebsocketConn   *gws.Conn
	Connected       bool
	Shutdown        bool
	mu              sync.Mutex
}

// IsConnected returns whether or not the connection is connected
func (s *Slack) IsConnected() bool {
	return s.Connected
}

// Setup takes in a slack configuration, sets bots target channel and
// sets verification token to access workspace
func (s *Slack) Setup(cfg *base.CommunicationsConfig) {
	s.Name = cfg.SlackConfig.Name
	s.Enabled = cfg.SlackConfig.Enabled
	s.Verbose = cfg.SlackConfig.Verbose
	s.TargetChannel = cfg.SlackConfig.TargetChannel
	s.VerificationToken = cfg.SlackConfig.VerificationToken
}

// Connect connects to the service
func (s *Slack) Connect() error {
	if err := s.NewConnection(); err != nil {
		return err
	}

	s.Connected = true
	return nil
}

// PushEvent pushes an event to either a slack channel or specific client
func (s *Slack) PushEvent(event base.Event) error {
	if s.Connected {
		return s.WebsocketSend("message",
			fmt.Sprintf("event: %s %s", event.Type, event.Message))
	}
	return errors.New("slack not connected")
}

// BuildURL returns an appended token string with the SlackURL
func (s *Slack) BuildURL(token string) string {
	return fmt.Sprintf("%s?token=%s", SlackURL, token)
}

// GetChannelsString returns a list of all channels on the slack workspace
func (s *Slack) GetChannelsString() []string {
	channels := make([]string, len(s.Details.Channels))
	for i := range s.Details.Channels {
		channels[i] = s.Details.Channels[i].NameNormalized
	}
	return channels
}

// GetUsernameByID returns a users name by ID
func (s *Slack) GetUsernameByID(id string) string {
	for i := range s.Details.Users {
		if s.Details.Users[i].ID == id {
			return s.Details.Users[i].Name
		}
	}
	return ""
}

// GetIDByName returns either a group ID or Channel ID
func (s *Slack) GetIDByName(userName string) (string, error) {
	id, err := s.GetGroupIDByName(userName)
	if err != nil {
		return s.GetChannelIDByName(userName)
	}
	return id, err
}

// GetGroupIDByName returns a groupID by group name
func (s *Slack) GetGroupIDByName(group string) (string, error) {
	for i := range s.Details.Groups {
		if s.Details.Groups[i].Name == group {
			return s.Details.Groups[i].ID, nil
		}
	}
	return "", errors.New("channel not found")
}

// GetChannelIDByName returns a channel ID by its corresponding name
func (s *Slack) GetChannelIDByName(channel string) (string, error) {
	for i := range s.Details.Channels {
		if s.Details.Channels[i].Name == channel {
			return s.Details.Channels[i].ID, nil
		}
	}
	return "", errors.New("channel not found")
}

// GetUsersInGroup returns a list of users currently in a group
func (s *Slack) GetUsersInGroup(group string) []string {
	for i := range s.Details.Groups {
		if s.Details.Groups[i].Name == group {
			return s.Details.Groups[i].Members
		}
	}
	return nil
}

// NewConnection connects the bot to a slack workgroup using a verification
// token and a channel
func (s *Slack) NewConnection() error {
	if !s.Connected {
		contents, err := common.SendHTTPRequest(context.TODO(),
			http.MethodGet,
			s.BuildURL(s.VerificationToken),
			nil,
			nil,
			s.Verbose)
		if err != nil {
			return err
		}

		err = json.Unmarshal(contents, &s.Details)
		if err != nil {
			return err
		}

		if !s.Details.Ok {
			return errors.New(s.Details.Error)
		}

		if s.Verbose {
			log.Debugf(log.CommunicationMgr, "Slack: %s [%s] connected to %s [%s] \nWebsocket URL: %s.\n",
				s.Details.Self.Name,
				s.Details.Self.ID,
				s.Details.Team.Domain,
				s.Details.Team.ID,
				s.Details.URL)
			log.Debugf(log.CommunicationMgr, "Slack: Public channels: %s\n", s.GetChannelsString())
		}

		s.TargetChannelID, err = s.GetIDByName(s.TargetChannel)
		if err != nil {
			return err
		}

		log.Debugf(log.CommunicationMgr, "Slack: Target channel ID: %v [#%v]\n", s.TargetChannelID,
			s.TargetChannel)
		return s.WebsocketConnect()
	}
	return errors.New("newConnection() Already Connected")
}

// WebsocketConnect creates a websocket dialer amd initiates a websocket
// connection
func (s *Slack) WebsocketConnect() error {
	var dialer gws.Dialer
	var err error

	websocketURL := s.Details.URL
	if s.ReconnectURL != "" {
		websocketURL = s.ReconnectURL
	}

	var resp *http.Response
	s.WebsocketConn, resp, err = dialer.Dial(websocketURL, http.Header{})
	if err != nil {
		return err
	}
	resp.Body.Close()

	go s.WebsocketReader()
	return nil
}

// WebsocketReader reads incoming events from the websocket connection
func (s *Slack) WebsocketReader() {
	for {
		_, resp, err := s.WebsocketConn.ReadMessage()
		if err != nil {
			log.Errorln(log.CommunicationMgr, err)
		}

		var data WebsocketResponse
		err = json.Unmarshal(resp, &data)
		if err != nil {
			log.Errorln(log.CommunicationMgr, err)
			continue
		}

		switch data.Type {
		case "error":
			err = s.handleErrorResponse(data)
			if err != nil {
				continue
			}
		case "hello":
			s.handleHelloResponse()
		case "reconnect_url":
			err = s.handleReconnectResponse(resp)
			if err != nil {
				continue
			}
		case "presence_change":
			err = s.handlePresenceChange(resp)
			if err != nil {
				continue
			}
		case "message":
			err = s.handleMessageResponse(resp, data)
			if err != nil {
				continue
			}
		case "pong":
			if s.Verbose {
				log.Debugln(log.CommunicationMgr, "Slack: Pong received from server")
			}
		default:
			log.Debugln(log.CommunicationMgr, string(resp))
		}
	}
}

func (s *Slack) handlePresenceChange(resp []byte) error {
	var p PresenceChange
	err := json.Unmarshal(resp, &p)
	if err != nil {
		return err
	}
	if s.Verbose {
		log.Debugf(log.CommunicationMgr, "Slack: Presence change. User %s [%s] changed status to %s\n",
			s.GetUsernameByID(p.User),
			p.User, p.Presence)
	}
	return nil
}

func (s *Slack) handleMessageResponse(resp []byte, data WebsocketResponse) error {
	if data.ReplyTo != 0 {
		return errors.New("reply to is != 0")
	}
	var msg Message
	err := json.Unmarshal(resp, &msg)
	if err != nil {
		return err
	}
	if s.Verbose {
		log.Debugf(log.CommunicationMgr, "Slack: Message received by %s [%s] with text: %s\n",
			s.GetUsernameByID(msg.User),
			msg.User, msg.Text)
	}
	if string(msg.Text[0]) == "!" {
		return s.HandleMessage(&msg)
	}
	return nil
}

func (s *Slack) handleErrorResponse(data WebsocketResponse) error {
	if data.Error.Msg == "Socket URL has expired" {
		if s.Verbose {
			log.Debugln(log.CommunicationMgr, "Slack websocket URL has expired.. Reconnecting")
		}

		if s.WebsocketConn == nil {
			return errors.New("websocket connection is nil")
		}

		if err := s.WebsocketConn.Close(); err != nil {
			log.Errorln(log.CommunicationMgr, err)
		}

		s.ReconnectURL = ""
		s.Connected = false
		return s.NewConnection()
	}
	return fmt.Errorf("unknown error %q", data.Error.Msg)
}

func (s *Slack) handleHelloResponse() {
	if s.Verbose {
		log.Debugln(log.CommunicationMgr, "Slack: Websocket connected successfully.")
	}
	s.Connected = true
	go s.WebsocketKeepAlive()
}

func (s *Slack) handleReconnectResponse(resp []byte) error {
	type reconnectResponse struct {
		URL string `json:"url"`
	}
	var recURL reconnectResponse
	err := json.Unmarshal(resp, &recURL)
	if err != nil {
		return err
	}
	s.ReconnectURL = recURL.URL
	if s.Verbose {
		log.Debugf(log.CommunicationMgr, "Slack: Reconnect URL set to %s\n", s.ReconnectURL)
	}
	return nil
}

// WebsocketKeepAlive sends a ping every 5 minutes to keep connection alive
func (s *Slack) WebsocketKeepAlive() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		if err := s.WebsocketSend("ping", ""); err != nil {
			log.Errorf(log.CommunicationMgr, "Slack: WebsocketKeepAlive() error %s\n", err)
		}
	}
}

// WebsocketSend sends a message via the websocket connection
func (s *Slack) WebsocketSend(eventType, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	newMessage := SendMessage{
		ID:      time.Now().Unix(),
		Type:    eventType,
		Channel: s.TargetChannelID,
		Text:    text,
	}
	data, err := json.Marshal(newMessage)
	if err != nil {
		return err
	}

	if s.Verbose {
		log.Debugf(log.CommunicationMgr, "Slack: Sending websocket message: %s\n", string(data))
	}

	if s.WebsocketConn == nil {
		return errors.New("websocket not connected")
	}
	return s.WebsocketConn.WriteMessage(gws.TextMessage, data)
}

// HandleMessage handles incoming messages and/or commands from slack
func (s *Slack) HandleMessage(msg *Message) error {
	if msg == nil {
		return errors.New("slack msg is nil")
	}

	msg.Text = strings.ToLower(msg.Text)
	switch {
	case strings.Contains(msg.Text, cmdStatus):
		return s.WebsocketSend("message", s.GetStatus())

	case strings.Contains(msg.Text, cmdHelp):
		return s.WebsocketSend("message", getHelp)

	default:
		return s.WebsocketSend("message", "GoCryptoTrader SlackBot - Command Unknown!")
	}
}
