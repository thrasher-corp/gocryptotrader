package wshandler

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
	sync.Mutex
	Verbose                      bool
	supportsMessageIDCorrelation bool
	supportsRetry                bool
	retryLimit                   int64
	MessageSequence              int64
	RateLimit                    float64
	timeout                      time.Duration
	ExchangeName                 string
	Url                          string
	ProxyURL                     string
	Wg                           sync.WaitGroup
	WebsocketConnection          *websocket.Conn
	pendingMessageResponseIDs    []WebsocketIDRequest
	// These are the requests and responses
	Shutdown    chan struct{}
	IDResponses map[int64][]byte
}

// WebsocketIDRequest l o l
type WebsocketIDRequest struct {
	MessageID  int64
	RetryCount int64
	Timeout    *time.Timer
	Message    interface{}
}

// Dial will handle all your life's problems
func (w *WebsocketConnection) Dial() error {
	var dialer websocket.Dialer
	if w.ProxyURL != "" {
		proxy, err := url.Parse(w.ProxyURL)
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	var conStatus *http.Response
	w.WebsocketConnection, _, err = dialer.Dial(w.Url, http.Header{})
	if err != nil {
		return fmt.Errorf("%v %v %v Error: %v", w.Url, conStatus, conStatus.StatusCode, err)
	}
	return nil
}

// Setup functions aren't necessary, but use this to ensure everything will be done correctly with validation
func (w *WebsocketConnection) Setup(verbose, supportsMessageIDCorrelation bool, rateLimit float64, exchangeName string) error {
	if exchangeName == "" {
		return errors.New("Exchange name not set")
	}
	w.supportsMessageIDCorrelation = supportsMessageIDCorrelation
	w.Verbose = verbose
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
	if w.Verbose {
		log.Debugf("%v sending message to websocket %v", w.ExchangeName, string(json))
	}
	return w.WebsocketConnection.WriteMessage(websocket.TextMessage, json)
}

func (w *WebsocketConnection) SendMessageReturnResponse(id int64, request interface{}) ([]byte, error) {
	err := w.SendMessage(request)
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go w.WaitForResult(id, &wg)
	wg.Wait()
	if _, ok := w.IDResponses[id]; !ok {
		return nil, fmt.Errorf("Timeout waiting for response with ID %v", id)
	}
	return w.IDResponses[id], nil
}

func (w *WebsocketConnection) WaitForResult(id int64, wg *sync.WaitGroup) {
	defer wg.Done()
	timer := time.NewTimer(3 * time.Second)
	for {
		select {
		case <-timer.C:
			return
		default:
			w.Lock()
			for k := range w.IDResponses {
				if k == id {
					// Remove entry too
					w.Unlock()
					return
				}
			}
			w.Unlock()
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// ReadMessage reads messages
func (w *WebsocketConnection) ReadMessage() (WebsocketResponse, error) {
	_, resp, err := w.WebsocketConnection.ReadMessage()
	if err != nil {
		return WebsocketResponse{}, err
	}
	return WebsocketResponse{Raw: resp}, nil
}

// GenerateMessageID Creates a messageID to checkout
func (w *WebsocketConnection) GenerateMessageID() int64 {
	w.Lock()
	defer w.Unlock()
	pendingMessageID := WebsocketIDRequest{
		MessageID:  time.Now().Unix(),
		RetryCount: w.retryLimit,
		Timeout:    time.NewTimer(w.timeout),
	}
	//go w.monitorTimeout(&pendingMessageID)
	w.pendingMessageResponseIDs = append(w.pendingMessageResponseIDs, pendingMessageID)
	return pendingMessageID.MessageID
}
