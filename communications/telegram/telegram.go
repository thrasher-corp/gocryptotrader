// Package telegram is used to connect to a cloud-based mobile and desktop
// messaging app using the bot API defined in
// https://core.telegram.org/bots/api#recent-changes
package telegram

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
	log "github.com/thrasher-/gocryptotrader/logger"
)

const (
	apiURL = "https://api.telegram.org/bot%s/%s"

	methodGetMe       = "getMe"
	methodGetUpdates  = "getUpdates"
	methodSendMessage = "sendMessage"

	cmdStart    = "/start"
	cmdStatus   = "/status"
	cmdHelp     = "/help"
	cmdSettings = "/settings"

	cmdHelpReply = `GoCryptoTrader TelegramBot, thank you for using this service!
	Current commands are:
	/start  		- Will authenticate your ID
	/status 		- Displays the status of the bot
	/help 			- Displays current command list
	/settings 	- Displays current bot settings`

	talkRoot = "GoCryptoTrader bot"
)

// Telegram is the overarching type across this package
type Telegram struct {
	base.Base
	Token             string
	Offset            int64
	AuthorisedClients []int64
}

// IsConnected returns whether or not the connection is connected
func (t *Telegram) IsConnected() bool { return t.Connected }

// Setup takes in a Telegram configuration and sets verification token
func (t *Telegram) Setup(cfg *config.CommunicationsConfig) {
	t.Name = cfg.TelegramConfig.Name
	t.Enabled = cfg.TelegramConfig.Enabled
	t.Token = cfg.TelegramConfig.VerificationToken
	t.Verbose = cfg.TelegramConfig.Verbose
}

// Connect starts an initial connection
func (t *Telegram) Connect() error {
	if err := t.TestConnection(); err != nil {
		return err
	}
	t.Connected = true
	go t.PollerStart()
	return nil
}

// PushEvent sends an event to a supplied recipient list via telegram
func (t *Telegram) PushEvent(event base.Event) error {
	for i := range t.AuthorisedClients {
		err := t.SendMessage(fmt.Sprintf("Type: %s Message: %s",
			event.Type, event.Message), t.AuthorisedClients[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// PollerStart starts the long polling sequence
func (t *Telegram) PollerStart() {
	t.InitialConnect()

	for {
		resp, err := t.GetUpdates()
		if err != nil {
			log.Errorln(log.CommunicationMgr, err)
		}

		for i := range resp.Result {
			if resp.Result[i].UpdateID > t.Offset {
				if string(resp.Result[i].Message.Text[0]) == "/" {
					err = t.HandleMessages(resp.Result[i].Message.Text, resp.Result[i].Message.From.ID)
					if err != nil {
						log.Errorln(log.CommunicationMgr, err)
					}
				}
				t.Offset = resp.Result[i].UpdateID
			}
		}
	}
}

// InitialConnect sets offset, and sends a welcome greeting to any associated
// IDs
func (t *Telegram) InitialConnect() {
	resp, err := t.GetUpdates()
	if err != nil {
		log.Errorln(log.CommunicationMgr, err)
		os.Exit(1)
	}

	if !resp.Ok {
		log.Errorln(log.CommunicationMgr, resp.Description)
		os.Exit(1)
	}

	warmWelcomeList := make(map[string]int64)
	for i := range resp.Result {
		if resp.Result[i].Message.From.ID != 0 {
			warmWelcomeList[resp.Result[i].Message.From.UserName] = resp.Result[i].Message.From.ID
		}
	}

	for userName, ID := range warmWelcomeList {
		err = t.SendMessage(fmt.Sprintf("GoCryptoTrader bot has connected: Hello, %s!", userName), ID)
		if err != nil {
			log.Errorln(log.CommunicationMgr, err)
			os.Exit(1)
		}
	}
	if len(resp.Result) == 0 {
		return
	}
	t.Offset = resp.Result[len(resp.Result)-1].UpdateID
}

// HandleMessages handles incoming message from the long polling routine
func (t *Telegram) HandleMessages(text string, chatID int64) error {
	switch {
	case strings.Contains(text, cmdHelp):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, cmdHelpReply), chatID)

	case strings.Contains(text, cmdStart):
		return t.SendMessage(fmt.Sprintf("%s: START COMMANDS HERE", talkRoot), chatID)

	case strings.Contains(text, cmdStatus):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, t.GetStatus()), chatID)

	default:
		return t.SendMessage(fmt.Sprintf("command %s not recognized", text), chatID)
	}
}

// GetUpdates gets new updates via a long poll connection
func (t *Telegram) GetUpdates() (GetUpdateResponse, error) {
	var newUpdates GetUpdateResponse
	path := fmt.Sprintf(apiURL, t.Token, methodGetUpdates)
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

	json, err := common.JSONEncode(&messageToSend)
	if err != nil {
		return err
	}

	resp := Message{}
	err = t.SendHTTPRequest(path, json, &resp)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return errors.New(resp.Description)
	}
	return nil
}

// SendHTTPRequest sends an authenticated HTTP request
func (t *Telegram) SendHTTPRequest(path string, json []byte, result interface{}) error {
	headers := make(map[string]string)
	headers["content-type"] = "application/json"

	resp, err := common.SendHTTPRequest(http.MethodPost, path, headers, bytes.NewBuffer(json))
	if err != nil {
		return err
	}

	return common.JSONDecode([]byte(resp), result)
}
