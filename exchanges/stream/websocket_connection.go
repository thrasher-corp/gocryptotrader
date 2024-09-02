package stream

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// WebsocketConnection contains all the data needed to send a message to a WS
// connection
type WebsocketConnection struct {
	// Gorilla websocket does not allow more than one goroutine to utilise writes methods
	writeControl     sync.Mutex
	connected        int32
	connection       *websocket.Conn
	rateLimit        *request.RateLimiterWithWeight
	_URL             string
	parent           *Websocket
	responseMaxLimit time.Duration
	reporter         Reporter
	// bespokeGenerateMessageID is a function that returns a unique message ID
	// defined externally. This is used for exchanges that require a unique
	// message ID for each message sent.
	bespokeGenerateMessageID func(highPrecision bool) int64
}

// Dial sets proxy urls and then connects to the websocket
func (w *WebsocketConnection) Dial(dialer *websocket.Dialer, headers http.Header) error {
	if w.parent.proxyAddr != "" {
		proxy, err := url.Parse(w.parent.proxyAddr)
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	var conStatus *http.Response

	w.connection, conStatus, err = dialer.Dial(w._URL, headers)
	if err != nil {
		if conStatus != nil {
			return fmt.Errorf("%s websocket connection: %v %v %v Error: %w", w.parent.exchangeName, w._URL, conStatus, conStatus.StatusCode, err)
		}
		return fmt.Errorf("%s websocket connection: %v Error: %w", w.parent.exchangeName, w._URL, err)
	}
	defer conStatus.Body.Close()

	if w.parent.verbose {
		log.Infof(log.WebsocketMgr, "%v Websocket connected to %s\n", w.parent.exchangeName, w._URL)
	}
	select {
	case w.parent.TrafficAlert <- struct{}{}:
	default:
	}
	w.setConnectedStatus(true)
	return nil
}

// SendJSONMessage sends a JSON encoded message over the connection
func (w *WebsocketConnection) SendJSONMessage(ctx context.Context, data interface{}) error {
	return w.writeToConn(ctx, func() error {
		if w.parent.verbose {
			if msg, err := json.Marshal(data); err == nil { // WriteJSON will error for us anyway
				log.Debugf(log.WebsocketMgr, "%s websocket connection: sending message: %s\n", w.parent.exchangeName, msg)
			}
		}
		return w.connection.WriteJSON(data)
	})
}

// SendRawMessage sends a message over the connection without JSON encoding it
func (w *WebsocketConnection) SendRawMessage(ctx context.Context, messageType int, message []byte) error {
	return w.writeToConn(ctx, func() error {
		if w.parent.verbose {
			log.Debugf(log.WebsocketMgr, "%v websocket connection: sending message [%s]\n", w.parent.exchangeName, message)
		}
		return w.connection.WriteMessage(messageType, message)
	})
}

func (w *WebsocketConnection) writeToConn(ctx context.Context, writeConn func() error) error {
	if !w.IsConnected() {
		return fmt.Errorf("%v websocket connection: cannot send message to a disconnected websocket", w.parent.exchangeName)
	}
	if w.rateLimit != nil {
		err := request.RateLimit(ctx, w.rateLimit)
		if err != nil {
			return fmt.Errorf("%s websocket connection: rate limit error: %w", w.parent.exchangeName, err)
		}
	}
	// This lock acts as a rolling gate to prevent WriteMessage panics. Acquire after rate limit check.
	w.writeControl.Lock()
	defer w.writeControl.Unlock()
	// NOTE: Secondary check to ensure the connection is still active after
	// semacquire and potential rate limit.
	if !w.IsConnected() {
		return fmt.Errorf("%v websocket connection: cannot send message to a disconnected websocket", w.parent.exchangeName)
	}
	return writeConn()
}

// SetupPingHandler will automatically send ping or pong messages based on
// WebsocketPingHandler configuration
func (w *WebsocketConnection) SetupPingHandler(handler PingHandler) {
	if handler.UseGorillaHandler {
		w.connection.SetPingHandler(func(msg string) error {
			err := w.connection.WriteControl(handler.MessageType, []byte(msg), time.Now().Add(handler.Delay))
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Timeout() {
				return nil
			}
			return err
		})
		return
	}
	w.parent.Wg.Add(1)
	go func() {
		defer w.parent.Wg.Done()
		ticker := time.NewTicker(handler.Delay)
		defer ticker.Stop()
		for {
			select {
			case <-w.parent.ShutdownC:
				return
			case <-ticker.C:
				err := w.SendRawMessage(context.TODO(), handler.MessageType, handler.Message)
				if err != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket connection: ping handler failed to send message [%s]: %v", w.parent.exchangeName, handler.Message, err)
					return
				}
			}
		}
	}()
}

// setConnectedStatus sets connection status if changed it will return true.
// TODO: Swap out these atomic switches and opt for sync.RWMutex.
func (w *WebsocketConnection) setConnectedStatus(b bool) bool {
	if b {
		return atomic.SwapInt32(&w.connected, 1) == 0
	}
	return atomic.SwapInt32(&w.connected, 0) == 1
}

// IsConnected exposes websocket connection status
func (w *WebsocketConnection) IsConnected() bool {
	return atomic.LoadInt32(&w.connected) == 1
}

// ReadMessage reads messages, can handle text, gzip and binary
func (w *WebsocketConnection) ReadMessage() Response {
	mType, resp, err := w.connection.ReadMessage()
	if err != nil {
		if IsDisconnectionError(err) {
			if w.setConnectedStatus(false) {
				// NOTE: When w.setConnectedStatus() returns true the underlying
				// state was changed and this infers that the connection was
				// externally closed and an error is reported else Shutdown()
				// method on WebsocketConnection type has been called and can
				// be skipped.
				select {
				case w.parent.ReadMessageErrors <- fmt.Errorf("%s %s: %w", w.parent.exchangeName, w._URL, err):
				default:
					// bypass if there is no receiver, as this stops it returning
					// when shutdown is called.
					log.Warnf(log.WebsocketMgr,
						"%s failed to relay error: %v",
						w.parent.exchangeName,
						err)
				}
			}
		}
		return Response{}
	}

	select {
	case w.parent.TrafficAlert <- struct{}{}:
	default: // Non-Blocking write ensures 1 buffered signal per trafficCheckInterval to avoid flooding
	}

	var standardMessage []byte
	switch mType {
	case websocket.TextMessage:
		standardMessage = resp
	case websocket.BinaryMessage:
		standardMessage, err = w.parseBinaryResponse(resp)
		if err != nil {
			log.Errorf(log.WebsocketMgr,
				"%v websocket connection: parseBinaryResponse error: %v",
				w.parent.exchangeName,
				err)
			return Response{}
		}
	}
	if w.parent.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v websocket connection: message received: %v",
			w.parent.exchangeName,
			string(standardMessage))
	}
	return Response{Raw: standardMessage, Type: mType}
}

// parseBinaryResponse parses a websocket binary response into a usable byte array
func (w *WebsocketConnection) parseBinaryResponse(resp []byte) ([]byte, error) {
	var reader io.ReadCloser
	var err error
	if len(resp) >= 2 && resp[0] == 31 && resp[1] == 139 { // Detect GZIP
		reader, err = gzip.NewReader(bytes.NewReader(resp))
		if err != nil {
			return nil, err
		}
	} else {
		reader = flate.NewReader(bytes.NewReader(resp))
	}
	standardMessage, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return standardMessage, reader.Close()
}

// GenerateMessageID generates a message ID for the individual connection.
// If a bespoke function is set (by using SetupNewConnection) it will use that,
// otherwise it will use the defaultGenerateMessageID function.
func (w *WebsocketConnection) GenerateMessageID(highPrec bool) int64 {
	if w.bespokeGenerateMessageID != nil {
		return w.bespokeGenerateMessageID(highPrec)
	}
	return w.defaultGenerateMessageID(highPrec)
}

// defaultGenerateMessageID generates the default message ID
func (w *WebsocketConnection) defaultGenerateMessageID(highPrec bool) int64 {
	var minValue int64 = 1e8
	var maxValue int64 = 2e8
	if highPrec {
		maxValue = 2e12
		minValue = 1e12
	}
	// utilization of hard coded positive numbers and default crypto/rand
	// io.reader will panic on error instead of returning
	randomNumber, err := rand.Int(rand.Reader, big.NewInt(maxValue-minValue+1))
	if err != nil {
		panic(err)
	}
	return randomNumber.Int64() + minValue
}

// Shutdown shuts down and closes specific connection
func (w *WebsocketConnection) Shutdown() error {
	if w == nil || w.connection == nil {
		return nil
	}
	w.setConnectedStatus(false)
	return w.connection.UnderlyingConn().Close()
}

// SetURL sets connection URL
func (w *WebsocketConnection) SetURL(url string) {
	w._URL = url
}

// GetURL returns the connection URL
func (w *WebsocketConnection) GetURL() string {
	return w._URL
}

// SendMessageReturnResponse will send a WS message to the connection and wait for response
func (w *WebsocketConnection) SendMessageReturnResponse(ctx context.Context, signature, request any) ([]byte, error) {
	resps, err := w.SendMessageReturnResponses(ctx, signature, request, 1)
	if err != nil {
		return nil, err
	}
	return resps[0], nil
}

// SendMessageReturnResponses will send a WS message to the connection and wait for N responses
// An error of ErrSignatureTimeout can be ignored if individual responses are being otherwise tracked
func (w *WebsocketConnection) SendMessageReturnResponses(ctx context.Context, signature, request any, expected int) ([][]byte, error) {
	outbound, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling json for %s: %w", signature, err)
	}

	ch, err := w.parent.Match.Set(signature, expected)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	err = w.SendRawMessage(ctx, websocket.TextMessage, outbound)
	if err != nil {
		return nil, err
	}

	timeout := time.NewTimer(w.responseMaxLimit * time.Duration(expected))

	resps := make([][]byte, 0, expected)
	for err == nil && len(resps) < expected {
		select {
		case resp := <-ch:
			resps = append(resps, resp)
		case <-timeout.C:
			w.parent.Match.RemoveSignature(signature)
			err = fmt.Errorf("%s %w %v", w.parent.exchangeName, ErrSignatureTimeout, signature)
		case <-ctx.Done():
			w.parent.Match.RemoveSignature(signature)
			err = ctx.Err()
		}
	}

	timeout.Stop()

	if err == nil && w.reporter != nil {
		w.reporter.Latency(w.parent.exchangeName, outbound, time.Since(start))
	}

	return resps, err
}
