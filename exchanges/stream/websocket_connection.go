package stream

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// errConnectionFault is a connection fault error which alerts the system that a connection cycle needs to take place.
	errConnectionFault         = errors.New("connection fault")
	errWebsocketIsDisconnected = errors.New("websocket connection is disconnected")
	errRateLimitNotFound       = errors.New("rate limit definition not found")
)

// Dial sets proxy urls and then connects to the websocket
func (w *WebsocketConnection) Dial(dialer *websocket.Dialer, headers http.Header) error {
	return w.DialContext(context.Background(), dialer, headers)
}

// DialContext sets proxy urls and then connects to the websocket
func (w *WebsocketConnection) DialContext(ctx context.Context, dialer *websocket.Dialer, headers http.Header) error {
	if w.ProxyURL != "" {
		proxy, err := url.Parse(w.ProxyURL)
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	var conStatus *http.Response
	w.Connection, conStatus, err = dialer.DialContext(ctx, w.URL, headers)
	if err != nil {
		if conStatus != nil {
			_ = conStatus.Body.Close()
			return fmt.Errorf("%s websocket connection: %v %v %v Error: %w", w.ExchangeName, w.URL, conStatus, conStatus.StatusCode, err)
		}
		return fmt.Errorf("%s websocket connection: %v Error: %w", w.ExchangeName, w.URL, err)
	}
	_ = conStatus.Body.Close()

	if w.Verbose {
		log.Infof(log.WebsocketMgr, "%v Websocket connected to %s\n", w.ExchangeName, w.URL)
	}
	select {
	case w.Traffic <- struct{}{}:
	default:
	}
	w.setConnectedStatus(true)
	return nil
}

// SendJSONMessage sends a JSON encoded message over the connection
func (w *WebsocketConnection) SendJSONMessage(ctx context.Context, epl request.EndpointLimit, data any) error {
	return w.writeToConn(ctx, epl, func() error {
		if request.IsVerbose(ctx, w.Verbose) {
			if msg, err := json.Marshal(data); err == nil { // WriteJSON will error for us anyway
				log.Debugf(log.WebsocketMgr, "%v %v: Sending message: %v", w.ExchangeName, removeURLQueryString(w.URL), string(msg))
			}
		}
		return w.Connection.WriteJSON(data)
	})
}

// SendRawMessage sends a message over the connection without JSON encoding it
func (w *WebsocketConnection) SendRawMessage(ctx context.Context, epl request.EndpointLimit, messageType int, message []byte) error {
	return w.writeToConn(ctx, epl, func() error {
		if request.IsVerbose(ctx, w.Verbose) {
			log.Debugf(log.WebsocketMgr, "%v %v: Sending message: %v", w.ExchangeName, removeURLQueryString(w.URL), string(message))
		}
		return w.Connection.WriteMessage(messageType, message)
	})
}

func (w *WebsocketConnection) writeToConn(ctx context.Context, epl request.EndpointLimit, writeConn func() error) error {
	if !w.IsConnected() {
		return fmt.Errorf("%v websocket connection: cannot send message %w", w.ExchangeName, errWebsocketIsDisconnected)
	}

	var rl *request.RateLimiterWithWeight
	if w.RateLimitDefinitions != nil {
		var ok bool
		if rl, ok = w.RateLimitDefinitions[epl]; !ok && w.RateLimit == nil {
			// Return an error if no specific connection rate limit is found for the endpoint but a global rate limit is
			// set. This ensures the system attempts to apply rate limiting, prioritizing endpoint-specific limits
			// if they are defined.
			return fmt.Errorf("%s websocket connection: %w for %v", w.ExchangeName, errRateLimitNotFound, epl)
		}
	}

	if rl == nil {
		// If a global rate limit definition is not found, use the connection rate limit as a fallback.
		rl = w.RateLimit
	}

	if rl != nil {
		if err := request.RateLimit(ctx, rl); err != nil {
			return fmt.Errorf("%s websocket connection: rate limit error: %w", w.ExchangeName, err)
		}
	}
	// This lock acts as a rolling gate to prevent WriteMessage panics. Acquire after rate limit check.
	w.writeControl.Lock()
	defer w.writeControl.Unlock()
	return writeConn()
}

// SetupPingHandler will automatically send ping or pong messages based on
// WebsocketPingHandler configuration
func (w *WebsocketConnection) SetupPingHandler(epl request.EndpointLimit, handler PingHandler) {
	if handler.UseGorillaHandler {
		w.Connection.SetPingHandler(func(msg string) error {
			err := w.Connection.WriteControl(handler.MessageType, []byte(msg), time.Now().Add(handler.Delay))
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Timeout() {
				return nil
			}
			return err
		})
		return
	}
	w.Wg.Add(1)
	go func() {
		defer w.Wg.Done()
		ticker := time.NewTicker(handler.Delay)
		for {
			select {
			case <-w.shutdown:
				ticker.Stop()
				return
			case <-ticker.C:
				err := w.SendRawMessage(context.TODO(), epl, handler.MessageType, handler.Message)
				if err != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket connection: ping handler failed to send message [%s]: %v", w.ExchangeName, handler.Message, err)
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
	mType, resp, err := w.Connection.ReadMessage()
	if err != nil {
		// If any error occurs, a Response{Raw: nil, Type: 0} is returned, causing the
		// reader routine to exit. This leaves the connection without an active reader,
		// leading to potential buffer issue from the ongoing websocket writes.
		// Such errors are passed to `w.readMessageErrors` when the connection is active.
		// The `connectionMonitor` handles these errors by flushing the buffer, reconnecting,
		// and resubscribing to the websocket to restore the connection.
		if w.setConnectedStatus(false) {
			// NOTE: When w.setConnectedStatus() returns true the underlying
			// state was changed and this infers that the connection was
			// externally closed and an error is reported else Shutdown()
			// method on WebsocketConnection type has been called and can
			// be skipped.
			select {
			case w.readMessageErrors <- fmt.Errorf("%w: %w", err, errConnectionFault):
			default:
				// bypass if there is no receiver, as this stops it returning
				// when shutdown is called.
				log.Warnf(log.WebsocketMgr, "%s failed to relay error: %v", w.ExchangeName, err)
			}
		}
		return Response{}
	}

	select {
	case w.Traffic <- struct{}{}:
	default: // Non-Blocking write ensures 1 buffered signal per trafficCheckInterval to avoid flooding
	}

	var standardMessage []byte
	switch mType {
	case websocket.TextMessage:
		standardMessage = resp
	case websocket.BinaryMessage:
		standardMessage, err = w.parseBinaryResponse(resp)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%v %v: Parse binary response error: %v", w.ExchangeName, removeURLQueryString(w.URL), err)
			return Response{Raw: []byte(``)} // Non-nil response to avoid the reader returning on this case.
		}
	}
	if w.Verbose {
		log.Debugf(log.WebsocketMgr, "%v %v: Message received: %v", w.ExchangeName, removeURLQueryString(w.URL), string(standardMessage))
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
	if w == nil || w.Connection == nil {
		return nil
	}
	w.setConnectedStatus(false)
	w.writeControl.Lock()
	defer w.writeControl.Unlock()
	return w.Connection.NetConn().Close()
}

// SetURL sets connection URL
func (w *WebsocketConnection) SetURL(url string) {
	w.URL = url
}

// SetProxy sets connection proxy
func (w *WebsocketConnection) SetProxy(proxy string) {
	w.ProxyURL = proxy
}

// GetURL returns the connection URL
func (w *WebsocketConnection) GetURL() string {
	return w.URL
}

// SendMessageReturnResponse will send a WS message to the connection and wait for response
func (w *WebsocketConnection) SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature, request any) ([]byte, error) {
	resps, err := w.SendMessageReturnResponses(ctx, epl, signature, request, 1)
	if err != nil {
		return nil, err
	}
	return resps[0], nil
}

// SendMessageReturnResponses will send a WS message to the connection and wait for N responses
// An error of ErrSignatureTimeout can be ignored if individual responses are being otherwise tracked
func (w *WebsocketConnection) SendMessageReturnResponses(ctx context.Context, epl request.EndpointLimit, signature, payload any, expected int) ([][]byte, error) {
	return w.SendMessageReturnResponsesWithInspector(ctx, epl, signature, payload, expected, nil)
}

// SendMessageReturnResponsesWithInspector will send a WS message to the connection and wait for N responses
// An error of ErrSignatureTimeout can be ignored if individual responses are being otherwise tracked
func (w *WebsocketConnection) SendMessageReturnResponsesWithInspector(ctx context.Context, epl request.EndpointLimit, signature, payload any, expected int, messageInspector Inspector) ([][]byte, error) {
	outbound, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling json for %s: %w", signature, err)
	}

	ch, err := w.Match.Set(signature, expected)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	err = w.SendRawMessage(ctx, epl, websocket.TextMessage, outbound)
	if err != nil {
		return nil, err
	}

	resps, err := w.waitForResponses(ctx, signature, ch, expected, messageInspector)
	if err != nil {
		return nil, err
	}

	if w.Reporter != nil {
		w.Reporter.Latency(w.ExchangeName, outbound, time.Since(start))
	}

	return resps, err
}

// waitForResponses waits for N responses from a channel
func (w *WebsocketConnection) waitForResponses(ctx context.Context, signature any, ch <-chan []byte, expected int, messageInspector Inspector) ([][]byte, error) {
	timeout := time.NewTimer(w.ResponseMaxLimit * time.Duration(expected))
	defer timeout.Stop()

	resps := make([][]byte, 0, expected)
inspection:
	for range expected {
		select {
		case resp := <-ch:
			resps = append(resps, resp)
			// Checks recently received message to determine if this is in fact the final message in a sequence of messages.
			if messageInspector != nil && messageInspector.IsFinal(resp) {
				w.Match.RemoveSignature(signature)
				break inspection
			}
		case <-timeout.C:
			w.Match.RemoveSignature(signature)
			return nil, fmt.Errorf("%s %w %v", w.ExchangeName, ErrSignatureTimeout, signature)
		case <-ctx.Done():
			w.Match.RemoveSignature(signature)
			return nil, ctx.Err()
		}
	}

	// Only check context verbosity. If the exchange is verbose, it will log the responses in the ReadMessage() call.
	if request.IsVerbose(ctx, false) {
		for i := range resps {
			log.Debugf(log.WebsocketMgr, "%v %v: Received response [%d/%d]: %v", w.ExchangeName, removeURLQueryString(w.URL), i+1, len(resps), string(resps[i]))
		}
	}

	return resps, nil
}

func removeURLQueryString(url string) string {
	if index := strings.Index(url, "?"); index != -1 {
		return url[:index]
	}
	return url
}
