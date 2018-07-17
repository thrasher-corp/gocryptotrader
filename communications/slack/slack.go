// Package slack is used to connect to the slack network. Slack is a
// code-centric collaboration hub that allows users to connect via an app and
// share different types of data
package slack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
)

// const declares main slack url and commands that will be supported on client
// side
const (
	SlackURL = "https://slack.com/api/rtm.start"

	cmdStatus    = "!status"
	cmdHelp      = "!help"
	cmdSettings  = "!settings"
	cmdTicker    = "!ticker"
	cmdPortfolio = "!portfolio"
	cmdOrderbook = "!orderbook"

	getHelp = `GoCryptoTrader SlackBot, thank you for using this service!
	Current commands are:
	!status 		- Displays current working status of bot
	!help 			- Displays help text
	!settings		- Displays current settings
	!ticker			- Displays recent ANX ticker
	!portfolio	- Displays portfolio data
	!orderbook	- Displays current ANX orderbook`
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
	WebsocketConn   *websocket.Conn
	Connected       bool
	Shutdown        bool
	sync.Mutex
}

// Setup takes in a slack configuration, sets bots target channel and
// sets verification token to access workspace
func (s *Slack) Setup(config config.CommunicationsConfig) {
	s.Name = config.SlackConfig.Name
	s.Enabled = config.SlackConfig.Enabled
	s.Verbose = config.SlackConfig.Verbose
	s.TargetChannel = config.SlackConfig.TargetChannel
	s.VerificationToken = config.SlackConfig.VerificationToken
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
func (s *Slack) PushEvent(base.Event) error {
	return errors.New("not yet implemented")
}

// BuildURL returns an appended token string with the SlackURL
func (s *Slack) BuildURL(token string) string {
	return fmt.Sprintf("%s?token=%s", SlackURL, token)
}

// GetChannelsString returns a list of all channels on the slack workspace
func (s *Slack) GetChannelsString() []string {
	var channels []string
	for i := range s.Details.Channels {
		channels = append(channels, s.Details.Channels[i].NameNormalized)
	}
	return channels
}

// GetUsernameByID returns a users name by ID
func (s *Slack) GetUsernameByID(ID string) string {
	for i := range s.Details.Users {
		if s.Details.Users[i].ID == ID {
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
	return "", errors.New("Channel not found")
}

// GetChannelIDByName returns a channel ID by its corresponding name
func (s *Slack) GetChannelIDByName(channel string) (string, error) {
	for i := range s.Details.Channels {
		if s.Details.Channels[i].Name == channel {
			return s.Details.Channels[i].ID, nil
		}
	}
	return "", errors.New("Channel not found")
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
	err := s.SendHTTPGetRequestSlack(s.BuildURL(s.VerificationToken), true, &s.Details)
	if err != nil {
		return err
	}

	if !s.Details.Ok {
		return errors.New(s.Details.Error)
	}

	if s.Verbose {
		log.Printf("%s [%s] connected to %s [%s] \nWebsocket URL: %s.\n",
			s.Details.Self.Name,
			s.Details.Self.ID,
			s.Details.Team.Domain,
			s.Details.Team.ID,
			s.Details.URL)
		log.Printf("Slack channels: %s", s.GetChannelsString())
	}

	s.TargetChannelID, err = s.GetIDByName(s.TargetChannel)
	if err != nil {
		return err
	}
	return s.WebsocketConnect()
}

// WebsocketConnect creates a websocket dialer amd initiates a websocket
// connection
func (s *Slack) WebsocketConnect() error {
	var Dialer websocket.Dialer
	var err error

	websocketURL := s.Details.URL
	if s.ReconnectURL != "" {
		websocketURL = s.ReconnectURL
	}

	s.WebsocketConn, _, err = Dialer.Dial(websocketURL, http.Header{})
	if err != nil {
		return err
	}

	go s.WebsocketReader()
	return nil
}

// WebsocketReader reads incoming events from the websocket connection
func (s *Slack) WebsocketReader() {
	for {
		_, resp, err := s.WebsocketConn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		var data WebsocketResponse

		err = common.JSONDecode(resp, &data)
		if err != nil {
			log.Println(err)
			continue
		}

		switch data.Type {

		case "error":
			s.handleErrorResponse(data)

		case "hello":
			s.handleHelloResponse(data)

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
				log.Println("Pong received from server")
			}
		default:
			log.Println(string(resp))
		}
	}
}

func (s *Slack) handlePresenceChange(resp []byte) error {
	var pres PresenceChange
	err := common.JSONDecode(resp, &pres)
	if err != nil {
		return err
	}
	if s.Verbose {
		log.Printf("Presence change. User %s [%s] changed status to %s\n",
			s.GetUsernameByID(pres.User),
			pres.User, pres.Presence)
	}
	return nil
}

func (s *Slack) handleMessageResponse(resp []byte, data WebsocketResponse) error {
	if data.ReplyTo != 0 {
		return fmt.Errorf("ReplyTo != 0")
	}
	var msg Message
	err := common.JSONDecode(resp, &msg)
	if err != nil {
		return err
	}
	if s.Verbose {
		log.Printf("Msg received by %s [%s] with text: %s\n",
			s.GetUsernameByID(msg.User),
			msg.User, msg.Text)
	}
	if string(msg.Text[0]) == "!" {
		s.HandleMessage(msg)
	}
	return nil
}
func (s *Slack) handleErrorResponse(data WebsocketResponse) {
	if data.Error.Msg == "Socket URL has expired" {
		if s.Verbose {
			log.Println("Slack websocket URL has expired.. Reconnecting")
		}
		if err := s.WebsocketConn.Close(); err != nil {
			log.Println(err)
		}
		s.ReconnectURL = ""
		s.Connected = false
		if err := s.NewConnection(); err != nil {
			log.Fatal(err)
		}
		return
	}
}

func (s *Slack) handleHelloResponse(data WebsocketResponse) {
	if s.Verbose {
		log.Println("Websocket connected successfully.")
	}
	s.Connected = true
	go s.WebsocketKeepAlive()
}

func (s *Slack) handleReconnectResponse(resp []byte) error {
	type reconnectResponse struct {
		URL string `json:"url"`
	}
	var recURL reconnectResponse
	err := common.JSONDecode(resp, &recURL)
	if err != nil {
		return err
	}
	s.ReconnectURL = recURL.URL
	if s.Verbose {
		log.Printf("Reconnect URL set to %s\n", s.ReconnectURL)
	}
	return nil
}

// WebsocketKeepAlive sends a ping every 5 minutes to keep connection alive
func (s *Slack) WebsocketKeepAlive() {
	ticker := time.NewTicker(5 * time.Minute)

	for {
		<-ticker.C
		if err := s.WebsocketSend("ping", ""); err != nil {
			log.Println("slack WebsocketKeepAlive() error", err)
		}
	}
}

// WebsocketSend sends a message via the websocket connection
func (s *Slack) WebsocketSend(eventType, text string) error {
	s.Lock()
	defer s.Unlock()
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
	return s.WebsocketConn.WriteMessage(websocket.TextMessage, data)
}

// HandleMessage handles incoming messages and/or commands from slack
func (s *Slack) HandleMessage(msg Message) {
	switch {
	case common.StringContains(msg.Text, cmdStatus):
		s.WebsocketSend("message", s.GetStatus())

	case common.StringContains(msg.Text, cmdHelp):
		s.WebsocketSend("message", getHelp)

	case common.StringContains(msg.Text, cmdTicker):
		s.WebsocketSend("message", s.GetTicker("ANX"))

	case common.StringContains(msg.Text, cmdOrderbook):
		s.WebsocketSend("message", s.GetOrderbook("ANX"))

	case common.StringContains(msg.Text, cmdSettings):
		s.WebsocketSend("message", s.GetSettings())

	case common.StringContains(msg.Text, cmdPortfolio):
		s.WebsocketSend("message", s.GetPortfolio())

	default:
		s.WebsocketSend("message", "GoCryptoTrader SlackBot - Command Unknown!")
	}
}

// SendHTTPGetRequestSlack sends a HTTP request
func (s *Slack) SendHTTPGetRequestSlack(url string, jsonDecode bool, result interface{}) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	httpCode := res.StatusCode
	if httpCode != 200 && httpCode != 400 {
		log.Printf("HTTP status code: %d\n", httpCode)
		return errors.New("status code was not 200")
	}

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if jsonDecode {
		return common.JSONDecode(contents, &result)
	}

	result = string(contents)
	return nil
}
