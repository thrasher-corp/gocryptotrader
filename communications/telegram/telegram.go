// Package telegram is used to connect to a cloud-based mobile and desktop
// messaging app using the bot API defined in
// https://core.telegram.org/bots/api#recent-changes
package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	apiURL = "https://api.telegram.org/bot%s/%s"

	methodGetMe       = "getMe"
	methodGetUpdates  = "getUpdates"
	methodSendMessage = "sendMessage"

	cmdStart  = "/start"
	cmdStatus = "/status"
	cmdHelp   = "/help"

	cmdHelpReply = `GoCryptoTrader TelegramBot, thank you for using this service!
	Current commands are:
	/start  		- Will authenticate your ID
	/status 		- Displays the status of the bot
	/help 			- Displays current command list`

	talkRoot = "GoCryptoTrader bot"
)

var (
	// ErrWaiter is the default timer to wait if an err occurs
	// before retrying after successfully connecting
	ErrWaiter = time.Second * 30

	// ErrNotConnected is the error message returned if Telegram is not connected
	ErrNotConnected = errors.New("Telegram not connected")
)

// Telegram is the overarching type across this package
type Telegram struct {
	base.Base
	initConnected     bool
	Token             string
	Offset            int64
	AuthorisedClients map[string]int64
}

// IsConnected returns whether or not the connection is connected
func (t *Telegram) IsConnected() bool { return t.Connected }

// Setup takes in a Telegram configuration and sets verification token
func (t *Telegram) Setup(cfg *base.CommunicationsConfig) {
	t.Name = cfg.TelegramConfig.Name
	t.Enabled = cfg.TelegramConfig.Enabled
	t.Token = cfg.TelegramConfig.VerificationToken
	t.Verbose = cfg.TelegramConfig.Verbose
	t.AuthorisedClients = cfg.TelegramConfig.AuthorisedClients
}

// Connect starts an initial connection
func (t *Telegram) Connect() error {
	if err := t.TestConnection(); err != nil {
		return err
	}

	log.Debugln(log.CommunicationMgr, "Telegram: Connected successfully!")
	t.Connected = true
	go t.PollerStart()
	return nil
}

// PushEvent sends an event to a supplied recipient list via telegram
func (t *Telegram) PushEvent(event base.Event) error {
	if !t.Connected {
		return ErrNotConnected
	}

	msg := fmt.Sprintf("Type: %s Message: %s",
		event.Type, event.Message)

	var errs error
	for user, ID := range t.AuthorisedClients {
		if ID == 0 {
			log.Warnf(log.CommunicationMgr, "Telegram: Unable to send message to %s as their ID isn't set. A user must issue any supported command to begin a session.\n", user)
			continue
		}
		if err := t.SendMessage(msg, ID); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// PollerStart starts the long polling sequence
func (t *Telegram) PollerStart() {
	errWait := func(err error) {
		log.Errorln(log.CommunicationMgr, err)
		time.Sleep(ErrWaiter)
	}

	for {
		if !t.initConnected {
			err := t.InitialConnect()
			if err != nil {
				errWait(err)
				continue
			}
			t.initConnected = true
		}

		resp, err := t.GetUpdates()
		if err != nil {
			errWait(err)
			continue
		}

		for i := range resp.Result {
			if resp.Result[i].UpdateID > t.Offset {
				username := resp.Result[i].Message.From.UserName
				if id, ok := t.AuthorisedClients[username]; ok && resp.Result[i].Message.Text[0] == '/' {
					if id == 0 {
						t.AuthorisedClients[username] = resp.Result[i].Message.From.ID
					}
					err = t.HandleMessages(resp.Result[i].Message.Text, resp.Result[i].Message.From.ID)
					if err != nil {
						log.Errorf(log.CommunicationMgr, "Telegram: Unable to HandleMessages. Error: %s\n", err)
						continue
					}
				}
				t.Offset = resp.Result[i].UpdateID
			}
		}
	}
}

// InitialConnect sets offset, and sends a welcome greeting to any associated
// IDs
func (t *Telegram) InitialConnect() error {
	resp, err := t.GetUpdates()
	if err != nil {
		return err
	}

	if !resp.Ok {
		return errors.New(resp.Description)
	}

	knownBadUsers := make(map[string]bool) // Used to prevent multiple warnings for the same unauthorised user
	for i := range resp.Result {
		if resp.Result[i].Message.From.UserName != "" && resp.Result[i].Message.From.ID != 0 {
			username := resp.Result[i].Message.From.UserName
			if _, ok := t.AuthorisedClients[username]; !ok {
				if !knownBadUsers[username] {
					log.Warnf(log.CommunicationMgr, "Telegram: Received message from unauthorised user: %s\n", username)
					knownBadUsers[username] = true
				}
				continue
			}
			t.AuthorisedClients[username] = resp.Result[i].Message.From.ID
		}
	}

	for userName, ID := range t.AuthorisedClients {
		if ID == 0 {
			continue
		}
		err = t.SendMessage(fmt.Sprintf("GoCryptoTrader bot has connected: Hello, %s!", userName), ID)
		if err != nil {
			log.Errorf(log.CommunicationMgr, "Telegram: Unable to send welcome message. Error: %s\n", err)
			continue
		}
	}

	if len(resp.Result) == 0 {
		return nil
	}

	t.Offset = resp.Result[len(resp.Result)-1].UpdateID
	return nil
}

// HandleMessages handles incoming message from the long polling routine
func (t *Telegram) HandleMessages(text string, chatID int64) error {
	if t.Verbose {
		log.Debugf(log.CommunicationMgr, "Telegram: Received message: %s\n", text)
	}

	switch {
	case strings.Contains(text, cmdHelp):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, cmdHelpReply), chatID)

	case strings.Contains(text, cmdStart):
		return t.SendMessage(talkRoot+": START COMMANDS HERE", chatID)

	case strings.Contains(text, cmdStatus):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, t.GetStatus()), chatID)

	default:
		return t.SendMessage(fmt.Sprintf("Command %s not recognized", text), chatID)
	}
}

// GetUpdates gets new updates via a long poll connection
func (t *Telegram) GetUpdates() (GetUpdateResponse, error) {
	var newUpdates GetUpdateResponse
	vals := url.Values{}
	if t.Offset != 0 {
		vals.Set("offset", strconv.FormatInt(t.Offset+1, 10))
	}
	path := common.EncodeURLValues(fmt.Sprintf(apiURL, t.Token, methodGetUpdates), vals)
	return newUpdates, t.SendHTTPRequest(path, nil, &newUpdates)
}

// TestConnection tests bot's supplied authentication token
func (t *Telegram) TestConnection() error {
	var isConnected User
	path := fmt.Sprintf(apiURL, t.Token, methodGetMe)

	err := t.SendHTTPRequest(path, nil, &isConnected)
	if err != nil {
		return err
	}

	if !isConnected.Ok {
		return errors.New(isConnected.Description)
	}
	return nil
}

// SendMessage sends a message to a user by their chatID
func (t *Telegram) SendMessage(text string, chatID int64) error {
	path := fmt.Sprintf(apiURL, t.Token, methodSendMessage)

	messageToSend := struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}{
		chatID,
		text,
	}

	jsonData, err := json.Marshal(&messageToSend)
	if err != nil {
		return err
	}

	resp := Message{}
	err = t.SendHTTPRequest(path, jsonData, &resp)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return errors.New(resp.Description)
	}

	if t.Verbose {
		log.Debugf(log.CommunicationMgr, "Telegram: Sent %q\n", text)
	}
	return nil
}

// SendHTTPRequest sends an authenticated HTTP request
func (t *Telegram) SendHTTPRequest(path string, data []byte, result any) error {
	headers := make(map[string]string)
	headers["content-type"] = "application/json"

	resp, err := common.SendHTTPRequest(context.TODO(),
		http.MethodPost,
		path,
		headers,
		bytes.NewBuffer(data),
		t.Verbose)
	if err != nil {
		return err
	}

	return json.Unmarshal(resp, result)
}
