package exchange

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// IWebsocketConnection an interface for testing
type IWebsocketConnection interface {
	Setup(verbose, supportsMessageIDCorrelation, supportsMessageSequence bool, rateLimit float64, exchangeName string, connection *websocket.Conn) error
	SendMessage(data interface{}) error
	VerifyResponseID(responseID uint64) bool
	GenerateMessageID()
	Dial() error
}

// WebsocketConnection contains all the datas needed to send a message to a WS
type WebsocketConnection struct {
	verbose                      bool
	supportsMessageIDCorrelation bool
	supportsRetry                bool
	retryLimit                   int64
	timeout                      time.Duration
	MessageSequence              int64
	RateLimit                    float64
	ExchangeName                 string
	url                          string
	proxyURL                     string
	sync.Mutex
	Wg                        sync.WaitGroup
	Message                   interface{}
	WebsocketConnection       *websocket.Conn
	pendingMessageResponseIDs []WebsocketMessageIDTimeoutMonitor
	ResponseIDTrackChannel    chan int64
	Shutdown                  chan struct{}
}

// WebsocketMessageIDTimeoutMonitor l o l
type WebsocketMessageIDTimeoutMonitor struct {
	ExpectedResponseID int64
	RetryCount         int64
	Timeout            *time.Timer
}

// Dial will handle all your life's problems
func (w *WebsocketConnection) Dial() error {
	var dialer websocket.Dialer
	if w.proxyURL != "" {
		proxy, err := url.Parse(w.proxyURL)
		if err != nil {
			return err
		}

		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	var conStatus *http.Response
	w.WebsocketConnection, _, err = dialer.Dial(w.url, http.Header{})
	if err != nil {
		return fmt.Errorf("%v %v %v Error: %v", w.url, conStatus, conStatus.StatusCode, err)
	}
	return nil
}

// Setup functions aren't necessary, but use this to ensure everything will be done correctly with validation
func (w *WebsocketConnection) Setup(verbose, supportsMessageIDCorrelation bool, rateLimit float64, exchangeName string) error {
	if exchangeName == "" {
		return errors.New("Exchange name not set")
	}
	w.supportsMessageIDCorrelation = supportsMessageIDCorrelation
	w.verbose = verbose
	w.RateLimit = rateLimit
	w.ExchangeName = exchangeName
	return nil
}

// SendMessage the one true message request. Sends message to WS
func (w *WebsocketConnection) SendMessage(data interface{}) error {
	w.Lock()
	defer w.Unlock()
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	if w.verbose {
		log.Debugf("%v sending message to websocket %v", w.ExchangeName, string(json))
	}
	return w.WebsocketConnection.WriteMessage(websocket.TextMessage, json)
}

// ReadMessage reads messages
func (w *WebsocketConnection) ReadMessage() (WebsocketResponse, error) {
	_, resp, err := w.WebsocketConnection.ReadMessage()
	if err != nil {
		return WebsocketResponse{}, err
	}
	//p.Websocket.TrafficAlert <- struct{}{}
	return WebsocketResponse{Raw: resp}, nil
}

// ResponseChannelReader will loop and wait for ids to verify if the WS has replied
func (w *WebsocketConnection) ResponseChannelReader() {
	w.Wg.Add(1)
	defer w.Wg.Done()
	for {
		select {
		case <-w.Shutdown:
			return
		case resp := <-w.ResponseIDTrackChannel:
			w.VerifyResponseID(resp)
		default:
			w.verifyTimeouts()

		}
	}
}

func (w *WebsocketConnection) verifyTimeouts() {
	for i := 0; i < len(w.pendingMessageResponseIDs); i++ {

	}
}

// VerifyResponseID will check if the passed responseID matches any pending messages
func (w *WebsocketConnection) VerifyResponseID(responseID int64) bool {
	w.Lock()
	defer w.Unlock()
	for i := 0; i < len(w.pendingMessageResponseIDs); i++ {
		if responseID == w.pendingMessageResponseIDs[i].ExpectedResponseID {
			w.removePendingMessageResponse(i)
			return true
		}
	}
	return false
}

func (w *WebsocketConnection) removePendingMessageResponse(i int) {
	w.pendingMessageResponseIDs[i].Timeout.Stop()
	w.pendingMessageResponseIDs = append(w.pendingMessageResponseIDs[:i], w.pendingMessageResponseIDs[i+1:]...)
}

// GenerateMessageID Creates a messageID to checkout
func (w *WebsocketConnection) GenerateMessageID() int64 {
	w.Lock()
	defer w.Unlock()
	pendingMessageID := WebsocketMessageIDTimeoutMonitor{
		ExpectedResponseID: time.Now().Unix(),
		RetryCount:         w.retryLimit,
		Timeout:            time.NewTimer(w.timeout),
	}
	go w.monitorTimeout(pendingMessageID.Timeout)
	w.pendingMessageResponseIDs = append(w.pendingMessageResponseIDs, pendingMessageID)
	return pendingMessageID.ExpectedResponseID
}

func (w *WebsocketConnection) monitorTimeout(timeout *time.Timer) {
	select {
	case <-timeout.C:
		// RELEASE THE HOUNDS
		return
	}
}
