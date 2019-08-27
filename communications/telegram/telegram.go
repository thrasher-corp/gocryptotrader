// Package telegram is used to connect to a cloud-based mobile and desktop
// messaging app using the bot API defined in
// https://core.telegram.org/bots/api#recent-changes
package telegram

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const (
	apiURL = "https://api.telegram.org/bot%s/%s"

	methodGetMe       = "getMe"
	methodGetUpdates  = "getUpdates"
	methodSendMessage = "sendMessage"

	cmdStart     = "/start"
	cmdStatus    = "/status"
	cmdHelp      = "/help"
	cmdSettings  = "/settings"
	cmdTicker    = "/ticker"
	cmdPortfolio = "/portfolio"
	cmdOrders    = "/orderbooks"

	cmdHelpReply = `GoCryptoTrader TelegramBot, thank you for using this service!
	Current commands are:
	/start  		- Will authenticate your ID
	/status 		- Displays the status of the bot
	/help 			- Displays current command list
	/settings 	- Displays current bot settings
	/ticker 		- Displays current ANX ticker data
	/portfolio	- Displays your current portfolio
	/orderbooks - Displays current orderbooks for ANX`

	talkRoot = "GoCryptoTrader bot"
)

var (
	// ErrWaiter is the default timer to wait if an err occurs
	// before retrying after successfully connecting
	ErrWaiter = time.Second * 30
)

// Telegram is the overarching type across this package
type Telegram struct {
	base.Base
	initConnected     bool
	Token             string
	Offset            int64
	AuthorisedClients []int64
}

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

	log.Debugln("Telegram: Connected successfully!")
	t.Connected = true
	go t.PollerStart()
	return nil
}

// PushEvent sends an event to a supplied recipient list via telegram
func (t *Telegram) PushEvent(event base.Event) error {
	msg := fmt.Sprintf("Type: %s Details: %s GainOrLoss: %s",
		event.Type, event.TradeDetails, event.GainLoss)
	for i := range t.AuthorisedClients {
		err := t.SendMessage(msg, t.AuthorisedClients[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// PollerStart starts the long polling sequence
func (t *Telegram) PollerStart() {
	errWait := func(err error) {
		log.Error(err)
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
				if string(resp.Result[i].Message.Text[0]) == "/" {
					err = t.HandleMessages(resp.Result[i].Message.Text, resp.Result[i].Message.From.ID)
					if err != nil {
						log.Errorf("Telegram: Unable to HandleMessages. Error: %s\n", err)
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

	warmWelcomeList := make(map[string]int64)
	for i := range resp.Result {
		if resp.Result[i].Message.From.ID != 0 {
			warmWelcomeList[resp.Result[i].Message.From.UserName] = resp.Result[i].Message.From.ID
		}
	}

	for userName, ID := range warmWelcomeList {
		err = t.SendMessage(fmt.Sprintf("GoCryptoTrader bot has connected: Hello, %s!", userName), ID)
		if err != nil {
			log.Errorf("Telegram: Unable to send welcome message. Error: %s\n", err)
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
		log.Debugf("Telegram: Received message: %s\n", text)
	}

	switch {
	case common.StringContains(text, cmdHelp):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, cmdHelpReply), chatID)

	case common.StringContains(text, cmdStart):
		return t.SendMessage(fmt.Sprintf("%s: START COMMANDS HERE", talkRoot), chatID)

	case common.StringContains(text, cmdOrders):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, t.GetOrderbook("ANX")), chatID)

	case common.StringContains(text, cmdStatus):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, t.GetStatus()), chatID)

	case common.StringContains(text, cmdTicker):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, t.GetTicker("ANX")), chatID)

	case common.StringContains(text, cmdSettings):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, t.GetSettings()), chatID)

	case common.StringContains(text, cmdPortfolio):
		return t.SendMessage(fmt.Sprintf("%s: %s", talkRoot, t.GetPortfolio()), chatID)

	default:
		return t.SendMessage(fmt.Sprintf("Command %s not recognized", text), chatID)
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

	if t.Verbose {
		log.Debugf("Telegram: Sent '%s'\n", text)
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
