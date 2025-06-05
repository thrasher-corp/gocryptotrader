package websocket

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
	"sync"
	"sync/atomic"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// errConnectionFault is a connection fault error which alerts the system that a connection cycle needs to take place.
	errConnectionFault         = errors.New("connection fault")
	errWebsocketIsDisconnected = errors.New("websocket connection is disconnected")
	errRateLimitNotFound       = errors.New("rate limit definition not found")
)

// Connection defines the interface for websocket connections
type Connection interface {
	Dial(context.Context, *gws.Dialer, http.Header) error
	ReadMessage() Response
	SetupPingHandler(request.EndpointLimit, PingHandler)
	// GenerateMessageID generates a message ID for the individual connection. If a bespoke function is set
	// (by using SetupNewConnection) it will use that, otherwise it will use the defaultGenerateMessageID function
	// defined in websocket_connection.go.
	GenerateMessageID(highPrecision bool) int64
	// SendMessageReturnResponse will send a WS message to the connection and wait for response
	SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature, request any) ([]byte, error)
	// SendMessageReturnResponses will send a WS message to the connection and wait for N responses
	SendMessageReturnResponses(ctx context.Context, epl request.EndpointLimit, signature, request any, expected int) ([][]byte, error)
	// SendMessageReturnResponsesWithInspector will send a WS message to the connection and wait for N responses with message inspection
	SendMessageReturnResponsesWithInspector(ctx context.Context, epl request.EndpointLimit, signature, request any, expected int, messageInspector Inspector) ([][]byte, error)
	// SendRawMessage sends a message over the connection without JSON encoding it
	SendRawMessage(ctx context.Context, epl request.EndpointLimit, messageType int, message []byte) error
	// SendJSONMessage sends a JSON encoded message over the connection
	SendJSONMessage(ctx context.Context, epl request.EndpointLimit, payload any) error
	SetURL(string)
	SetProxy(string)
	GetURL() string
	Shutdown() error
}

// ConnectionSetup defines variables for an individual stream connection
type ConnectionSetup struct {
	ResponseCheckTimeout    time.Duration
	ResponseMaxLimit        time.Duration
	RateLimit               *request.RateLimiterWithWeight
	Authenticated           bool
	ConnectionLevelReporter Reporter

	// URL defines the websocket server URL to connect to
	URL string
	// Connector is the function that will be called to connect to the
	// exchange's websocket server. This will be called once when the stream
	// service is started. Any bespoke connection logic should be handled here.
	Connector func(ctx context.Context, conn Connection) error
	// GenerateSubscriptions is a function that will be called to generate a
	// list of subscriptions to be made to the exchange's websocket server.
	GenerateSubscriptions func() (subscription.List, error)
	// Subscriber is a function that will be called to send subscription
	// messages based on the exchange's websocket server requirements to
	// subscribe to specific channels.
	Subscriber func(ctx context.Context, conn Connection, sub subscription.List) error
	// Unsubscriber is a function that will be called to send unsubscription
	// messages based on the exchange's websocket server requirements to
	// unsubscribe from specific channels. NOTE: IF THE FEATURE IS ENABLED.
	Unsubscriber func(ctx context.Context, conn Connection, unsub subscription.List) error
	// Handler defines the function that will be called when a message is
	// received from the exchange's websocket server. This function should
	// handle the incoming message and pass it to the appropriate data handler.
	Handler func(ctx context.Context, incoming []byte) error
	// BespokeGenerateMessageID is a function that returns a unique message ID.
	// This is useful for when an exchange connection requires a unique or
	// structured message ID for each message sent.
	BespokeGenerateMessageID func(highPrecision bool) int64
	Authenticate             func(ctx context.Context, conn Connection) error
	// MessageFilter defines the criteria used to match messages to a specific connection.
	// The filter enables precise routing and handling of messages for distinct connection contexts.
	MessageFilter any
}

// Inspector is used to verify messages via SendMessageReturnResponsesWithInspection
// It inspects the []bytes websocket message and returns true if the message is the final message in a sequence of expected messages
type Inspector interface {
	IsFinal([]byte) bool
}

// Response defines generalised data from the websocket connection
type Response struct {
	Type int
	Raw  []byte
}

// connection contains all the data needed to send a message to a websocket connection
type connection struct {
	Verbose                  bool
	connected                int32
	writeControl             sync.Mutex                     // Gorilla websocket does not allow more than one goroutine to utilise write methods
	RateLimit                *request.RateLimiterWithWeight // RateLimit is a rate limiter for the connection itself
	RateLimitDefinitions     request.RateLimitDefinitions   // RateLimitDefinitions contains the rate limiters shared between WebSocket and REST connections
	Reporter                 Reporter
	ExchangeName             string
	URL                      string
	ProxyURL                 string
	Wg                       *sync.WaitGroup
	Connection               *gws.Conn
	shutdown                 chan struct{}
	Match                    *Match
	ResponseMaxLimit         time.Duration
	Traffic                  chan struct{}
	readMessageErrors        chan error
	bespokeGenerateMessageID func(highPrecision bool) int64
}

// Dial sets proxy urls and then connects to the websocket
func (c *connection) Dial(ctx context.Context, dialer *gws.Dialer, headers http.Header) error {
	if c.ProxyURL != "" {
		proxy, err := url.Parse(c.ProxyURL)
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}

	var err error
	var conStatus *http.Response
	c.Connection, conStatus, err = dialer.DialContext(ctx, c.URL, headers)
	if err != nil {
		if conStatus != nil {
			_ = conStatus.Body.Close()
			return fmt.Errorf("%s websocket connection: %v %v %v Error: %w", c.ExchangeName, c.URL, conStatus, conStatus.StatusCode, err)
		}
		return fmt.Errorf("%s websocket connection: %v Error: %w", c.ExchangeName, c.URL, err)
	}
	_ = conStatus.Body.Close()

	if c.Verbose {
		log.Infof(log.WebsocketMgr, "%v Websocket connected to %s\n", c.ExchangeName, c.URL)
	}
	select {
	case c.Traffic <- struct{}{}:
	default:
	}
	c.setConnectedStatus(true)
	return nil
}

// SendJSONMessage sends a JSON encoded message over the connection
func (c *connection) SendJSONMessage(ctx context.Context, epl request.EndpointLimit, data any) error {
	return c.writeToConn(ctx, epl, func() error {
		if request.IsVerbose(ctx, c.Verbose) {
			if msg, err := json.Marshal(data); err == nil { // WriteJSON will error for us anyway
				log.Debugf(log.WebsocketMgr, "%v %v: Sending message: %v", c.ExchangeName, removeURLQueryString(c.URL), string(msg))
			}
		}
		return c.Connection.WriteJSON(data)
	})
}

// SendRawMessage sends a message over the connection without JSON encoding it
func (c *connection) SendRawMessage(ctx context.Context, epl request.EndpointLimit, messageType int, message []byte) error {
	return c.writeToConn(ctx, epl, func() error {
		if request.IsVerbose(ctx, c.Verbose) {
			log.Debugf(log.WebsocketMgr, "%v %v: Sending message: %v", c.ExchangeName, removeURLQueryString(c.URL), string(message))
		}
		return c.Connection.WriteMessage(messageType, message)
	})
}

func (c *connection) writeToConn(ctx context.Context, epl request.EndpointLimit, writeConn func() error) error {
	if !c.IsConnected() {
		return fmt.Errorf("%v websocket connection: cannot send message %w", c.ExchangeName, errWebsocketIsDisconnected)
	}

	var rl *request.RateLimiterWithWeight
	if c.RateLimitDefinitions != nil {
		var ok bool
		if rl, ok = c.RateLimitDefinitions[epl]; !ok && c.RateLimit == nil {
			// Return an error if no specific connection rate limit is found for the endpoint but a global rate limit is
			// set. This ensures the system attempts to apply rate limiting, prioritizing endpoint-specific limits
			// if they are defined.
			return fmt.Errorf("%s websocket connection: %w for %v", c.ExchangeName, errRateLimitNotFound, epl)
		}
	}

	if rl == nil {
		// If a global rate limit definition is not found, use the connection rate limit as a fallback.
		rl = c.RateLimit
	}

	if rl != nil {
		if err := request.RateLimit(ctx, rl); err != nil {
			return fmt.Errorf("%s websocket connection: rate limit error: %w", c.ExchangeName, err)
		}
	}
	// This lock acts as a rolling gate to prevent WriteMessage panics. Acquire after rate limit check.
	c.writeControl.Lock()
	defer c.writeControl.Unlock()
	return writeConn()
}

// SetupPingHandler will automatically send ping or pong messages based on
// WebsocketPingHandler configuration
func (c *connection) SetupPingHandler(epl request.EndpointLimit, handler PingHandler) {
	if handler.UseGorillaHandler {
		c.Connection.SetPingHandler(func(msg string) error {
			err := c.Connection.WriteControl(handler.MessageType, []byte(msg), time.Now().Add(handler.Delay))
			if err == gws.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Timeout() {
				return nil
			}
			return err
		})
		return
	}
	c.Wg.Add(1)
	go func() {
		defer c.Wg.Done()
		ticker := time.NewTicker(handler.Delay)
		for {
			select {
			case <-c.shutdown:
				ticker.Stop()
				return
			case <-ticker.C:
				err := c.SendRawMessage(context.TODO(), epl, handler.MessageType, handler.Message)
				if err != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket connection: ping handler failed to send message [%s]: %v", c.ExchangeName, handler.Message, err)
					return
				}
			}
		}
	}()
}

// setConnectedStatus sets connection status if changed it will return true.
// TODO: Swap out these atomic switches and opt for sync.RWMutex.
func (c *connection) setConnectedStatus(b bool) bool {
	if b {
		return atomic.SwapInt32(&c.connected, 1) == 0
	}
	return atomic.SwapInt32(&c.connected, 0) == 1
}

// IsConnected exposes websocket connection status
func (c *connection) IsConnected() bool {
	return atomic.LoadInt32(&c.connected) == 1
}

// ReadMessage reads messages, can handle text, gzip and binary
func (c *connection) ReadMessage() Response {
	mType, resp, err := c.Connection.ReadMessage()
	if err != nil {
		// If any error occurs, a Response{Raw: nil, Type: 0} is returned, causing the
		// reader routine to exit. This leaves the connection without an active reader,
		// leading to potential buffer issue from the ongoing websocket writes.
		// Such errors are passed to `c.readMessageErrors` when the connection is active.
		// The `connectionMonitor` handles these errors by flushing the buffer, reconnecting,
		// and resubscribing to the websocket to restore the connection.
		if c.setConnectedStatus(false) {
			// NOTE: When c.setConnectedStatus() returns true the underlying
			// state was changed and this infers that the connection was
			// externally closed and an error is reported else Shutdown()
			// method on WebsocketConnection type has been called and can
			// be skipped.
			select {
			case c.readMessageErrors <- fmt.Errorf("%w: %w", err, errConnectionFault):
			default:
				// bypass if there is no receiver, as this stops it returning
				// when shutdown is called.
				log.Warnf(log.WebsocketMgr, "%s failed to relay error: %v", c.ExchangeName, err)
			}
		}
		return Response{}
	}

	select {
	case c.Traffic <- struct{}{}:
	default: // Non-Blocking write ensures 1 buffered signal per trafficCheckInterval to avoid flooding
	}

	var standardMessage []byte
	switch mType {
	case gws.TextMessage:
		standardMessage = resp
	case gws.BinaryMessage:
		standardMessage, err = c.parseBinaryResponse(resp)
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%v %v: Parse binary response error: %v", c.ExchangeName, removeURLQueryString(c.URL), err)
			return Response{Raw: []byte(``)} // Non-nil response to avoid the reader returning on this case.
		}
	}
	if c.Verbose {
		log.Debugf(log.WebsocketMgr, "%v %v: Message received: %v", c.ExchangeName, removeURLQueryString(c.URL), string(standardMessage))
	}
	return Response{Raw: standardMessage, Type: mType}
}

// parseBinaryResponse parses a websocket binary response into a usable byte array
func (c *connection) parseBinaryResponse(resp []byte) ([]byte, error) {
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
func (c *connection) GenerateMessageID(highPrec bool) int64 {
	if c.bespokeGenerateMessageID != nil {
		return c.bespokeGenerateMessageID(highPrec)
	}
	return c.defaultGenerateMessageID(highPrec)
}

// defaultGenerateMessageID generates the default message ID
func (c *connection) defaultGenerateMessageID(highPrec bool) int64 {
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
func (c *connection) Shutdown() error {
	if c == nil || c.Connection == nil {
		return nil
	}
	c.setConnectedStatus(false)
	c.writeControl.Lock()
	defer c.writeControl.Unlock()
	return c.Connection.NetConn().Close()
}

// SetURL sets connection URL
func (c *connection) SetURL(url string) {
	c.URL = url
}

// SetProxy sets connection proxy
func (c *connection) SetProxy(proxy string) {
	c.ProxyURL = proxy
}

// GetURL returns the connection URL
func (c *connection) GetURL() string {
	return c.URL
}

// SendMessageReturnResponse will send a WS message to the connection and wait for response
func (c *connection) SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature, payload any) ([]byte, error) {
	resps, err := c.SendMessageReturnResponses(ctx, epl, signature, payload, 1)
	if err != nil {
		return nil, err
	}
	return resps[0], nil
}

// SendMessageReturnResponses will send a WS message to the connection and wait for N responses
// An error of ErrSignatureTimeout can be ignored if individual responses are being otherwise tracked
func (c *connection) SendMessageReturnResponses(ctx context.Context, epl request.EndpointLimit, signature, payload any, expected int) ([][]byte, error) {
	return c.SendMessageReturnResponsesWithInspector(ctx, epl, signature, payload, expected, nil)
}

// SendMessageReturnResponsesWithInspector will send a WS message to the connection and wait for N responses
// An error of ErrSignatureTimeout can be ignored if individual responses are being otherwise tracked
func (c *connection) SendMessageReturnResponsesWithInspector(ctx context.Context, epl request.EndpointLimit, signature, payload any, expected int, messageInspector Inspector) ([][]byte, error) {
	outbound, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling json for %s: %w", signature, err)
	}

	ch, err := c.Match.Set(signature, expected)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	err = c.SendRawMessage(ctx, epl, gws.TextMessage, outbound)
	if err != nil {
		return nil, err
	}

	resps, err := c.waitForResponses(ctx, signature, ch, expected, messageInspector)
	if err != nil {
		return nil, err
	}

	if c.Reporter != nil {
		c.Reporter.Latency(c.ExchangeName, outbound, time.Since(start))
	}

	return resps, err
}

// waitForResponses waits for N responses from a channel
func (c *connection) waitForResponses(ctx context.Context, signature any, ch <-chan []byte, expected int, messageInspector Inspector) ([][]byte, error) {
	timeout := time.NewTimer(c.ResponseMaxLimit * time.Duration(expected))
	defer timeout.Stop()

	resps := make([][]byte, 0, expected)
inspection:
	for range expected {
		select {
		case resp := <-ch:
			resps = append(resps, resp)
			// Checks recently received message to determine if this is in fact the final message in a sequence of messages.
			if messageInspector != nil && messageInspector.IsFinal(resp) {
				c.Match.RemoveSignature(signature)
				break inspection
			}
		case <-timeout.C:
			c.Match.RemoveSignature(signature)
			return nil, fmt.Errorf("%s %w %v", c.ExchangeName, ErrSignatureTimeout, signature)
		case <-ctx.Done():
			c.Match.RemoveSignature(signature)
			return nil, ctx.Err()
		}
	}

	// Only check context verbosity. If the exchange is verbose, it will log the responses in the ReadMessage() call.
	if request.IsVerbose(ctx, false) {
		for i := range resps {
			log.Debugf(log.WebsocketMgr, "%v %v: Received response [%d/%d]: %v", c.ExchangeName, removeURLQueryString(c.URL), i+1, len(resps), string(resps[i]))
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
