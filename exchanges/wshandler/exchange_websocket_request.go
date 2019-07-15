package wshandler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
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
	RateLimit                    float64
	timeout                      time.Duration
	ExchangeName                 string
	URL                          string
	ProxyURL                     string
	Wg                           sync.WaitGroup
	WebsocketConnection          *websocket.Conn
	Shutdown                     chan struct{}
	// These are the requests and responses
	IDResponses map[int64][]byte
}

// AddResponseWithID adds data to IDResponses with locks and a nil check
func (w *WebsocketConnection) AddResponseWithID(id int64, data []byte) {
	w.Lock()
	defer w.Unlock()
	if w.IDResponses == nil {
		w.IDResponses = make(map[int64][]byte)
	}
	w.IDResponses[id] = data
}

// Dial will handle all your life's problems
func (w *WebsocketConnection) Dial(dialer *websocket.Dialer) error {
	if w.ProxyURL != "" {
		proxy, err := url.Parse(w.ProxyURL)
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	var conStatus *http.Response
	w.WebsocketConnection, conStatus, err = dialer.Dial(w.URL, http.Header{})
	if err != nil {
		if conStatus != nil {
			return fmt.Errorf("%v %v %v Error: %v", w.URL, conStatus, conStatus.StatusCode, err)

		} else {
			return fmt.Errorf("%v Error: %v", w.URL, err)
		}
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
	if w.RateLimit > 0 {
		time.Sleep(time.Duration(w.RateLimit) * time.Millisecond)
	}
	return w.WebsocketConnection.WriteMessage(websocket.TextMessage, json)
}

// SendMessageReturnResponse will send a WS message to the connection
// It will then run a goroutine to await a JSON response
// If there is no response it will return an error
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
	defer func() {
		delete(w.IDResponses, id)
	}()
	return w.IDResponses[id], nil
}

// WaitForResult will keep checking w.IDResponses for a response ID
// If the timer expires, it will return without
func (w *WebsocketConnection) WaitForResult(id int64, wg *sync.WaitGroup) {
	defer wg.Done()
	timer := time.NewTimer(7 * time.Second)
	for {
		select {
		case <-timer.C:
			return
		default:
			w.Lock()
			for k := range w.IDResponses {
				if k == id {
					w.Unlock()
					return
				}
			}
			w.Unlock()
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// ReadMessage reads messages, can handle text and binary
func (w *WebsocketConnection) ReadMessage() (WebsocketResponse, error) {
	mType, resp, err := w.WebsocketConnection.ReadMessage()
	if err != nil {
		return WebsocketResponse{}, err
	}
	var standardMessage []byte
	switch mType {
	case websocket.TextMessage:
		standardMessage = resp
	case websocket.BinaryMessage:
		// Detect GZIP
		if resp[0] == 31 && resp[1] == 139 {
			b := bytes.NewReader(resp)
			gReader, err := gzip.NewReader(b)
			if err != nil {
				return WebsocketResponse{}, err
			}
			standardMessage, err = ioutil.ReadAll(gReader)
			if err != nil {
				return WebsocketResponse{}, err
			}
			err = gReader.Close()
			if err != nil {
				return WebsocketResponse{}, err
			}
		} else {
			reader := flate.NewReader(bytes.NewReader(resp))
			standardMessage, err = ioutil.ReadAll(reader)
			reader.Close()
			if err != nil {
				return WebsocketResponse{}, err
			}
		}
	}
	if w.Verbose {
		log.Debugf("%v Websocket message received: %v",
			w.ExchangeName,
			string(standardMessage))
	}
	return WebsocketResponse{Raw: standardMessage}, nil
}

// GenerateMessageID Creates a messageID to checkout
func (w *WebsocketConnection) GenerateMessageID(useNano bool) int64 {
	if useNano {
		return time.Now().UnixNano()
	}
	return time.Now().Unix()
}
