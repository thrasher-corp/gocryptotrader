package binanceus

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	websocketAPIURL = "wss://ws-api.binance.us:443/ws-api/v3"
	// websocketAPIMessageFilter is the message filter of the WebSocket API connection
	websocketAPIMessageFilter = "websocket-api"
	// websocketAPIResponseMaxLimit is the least time the WebSocket API connection waits for a response. A request left
	// unanswered may still be executed, so a short wait reports an order Binance.US may still place as failed, and a
	// caller that retries can place it twice; when its backend gives up first, Binance.US answers -1007, execution status
	// unknown
	websocketAPIResponseMaxLimit = 11 * time.Second
)

var errWebsocketAPIRequestFailed = errors.New("websocket API request failed")

// WebsocketAPI calls Binance.US's methods over the WebSocket API connection rather than REST; a method with a REST form
// takes that form's name and arguments. The connection is set up when authenticated websocket support is enabled
type WebsocketAPI struct {
	e *Exchange
}

// WebsocketAPI returns the methods called over the WebSocket API connection
func (e *Exchange) WebsocketAPI() WebsocketAPI {
	return WebsocketAPI{e: e}
}

// websocketAPIConnection keeps the WebSocket API connection for the requests sent on it. Looking the connection up
// through the websocket manager waits on the manager's lock, which connecting and flushing hold across every dial, so
// each order request would wait while any connection connects
type websocketAPIConnection struct {
	mu   sync.RWMutex
	conn websocket.Connection
	// requests counts the requests sent on conn that have not returned, and retired stops conn taking more once a
	// serverShutdown event warns that its server, or the stream server, is about to disconnect. The server still answers
	// what it has, so the connection is closed once those requests return, or responseLimit has passed, rather than
	// failing orders it has placed
	requests      *sync.WaitGroup
	retired       bool
	responseLimit time.Duration
}

// set keeps conn for requests, which wait up to responseLimit for their responses
func (w *websocketAPIConnection) set(conn websocket.Connection, responseLimit time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.conn, w.requests, w.retired, w.responseLimit = conn, new(sync.WaitGroup), false, responseLimit
}

// get returns the connection kept for requests, or nil once it has been retired
func (w *websocketAPIConnection) get() websocket.Connection {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.retired {
		return nil
	}
	return w.conn
}

// kept returns the connection kept for requests, retired or not
func (w *websocketAPIConnection) kept() websocket.Connection {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.conn
}

// acquire returns the connection kept for requests, counting a request in flight on it until release is called, or a
// nil connection once it has been retired
func (w *websocketAPIConnection) acquire() (conn websocket.Connection, release func()) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.conn == nil || w.retired {
		return nil, nil
	}
	w.requests.Add(1)
	return w.conn, w.requests.Done
}

// retire stops conn taking requests, when it is the connection kept for them, and returns a function that waits for the
// requests in flight on it to return, or for their response limit. A second notice for the same connection waits for
// the same requests
func (w *websocketAPIConnection) retire(conn websocket.Connection) (wait func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if conn == nil || w.conn != conn {
		return func() {}
	}
	w.retired = true
	requests, limit := w.requests, w.responseLimit
	return func() {
		returned := make(chan struct{})
		go func() {
			requests.Wait()
			close(returned)
		}()
		timer := time.NewTimer(limit)
		defer timer.Stop()
		select {
		case <-returned:
		case <-timer.C:
		}
	}
}

// websocketAPIConnectionSetup returns the setup of the WebSocket API connection, which also delivers the User Data
// Stream. Binance.US limits WebSocket API requests by the weight pools they share with REST, not by a message rate, so
// the connection has no rate limit of its own and a SIGNED request that has already waited for its pool is sent at once
func (e *Exchange) websocketAPIConnectionSetup(exch *config.Exchange) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                      websocketAPIURL,
		Connector:                e.WebsocketConnectAPI,
		Authenticate:             e.WebsocketAuthenticateAPI,
		Handler:                  e.wsHandleAPIData,
		MessageFilter:            websocketAPIMessageFilter,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         websocketAPIResponseLimit(exch),
		SubscriptionsNotRequired: true,
	}
}

// websocketAPIResponseLimit returns how long the WebSocket API connection waits for a response
func websocketAPIResponseLimit(exch *config.Exchange) time.Duration {
	return max(exch.WebsocketResponseMaxLimit, websocketAPIResponseMaxLimit)
}

// WebsocketConnectAPI connects to the WebSocket API and keeps the connection for the requests sent on it
func (e *Exchange) WebsocketConnectAPI(ctx context.Context, conn websocket.Connection) error {
	e.userData.reset()
	e.wsAPI.set(nil, 0)
	if err := e.Requester.InitiateRateLimit(ctx, websocketAPIConnectRate); err != nil {
		return err
	}
	dialer := gws.Dialer{
		HandshakeTimeout: e.Config.HTTPTimeout,
		Proxy:            http.ProxyFromEnvironment,
	}
	if err := conn.Dial(ctx, &dialer, http.Header{}, nil); err != nil {
		return fmt.Errorf("%s unable to connect to the WebSocket API: %w", e.Name, err)
	}
	conn.SetupPingHandler(wsConnectionMessageRate, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})
	e.wsAPI.set(conn, websocketAPIResponseLimit(e.Config))
	return nil
}

// wsHandleAPIData routes WebSocket API responses to their requests, and User Data Stream and connection events to their
// handlers
func (e *Exchange) wsHandleAPIData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if id, err := jsonparser.GetString(respRaw, "id"); err == nil {
		if creds := e.userData.recordResponse(id, respRaw); e.isConfiguredAccount(creds) && isRejectedKey(respRaw) {
			e.stopAuthenticatedWebsocket()
		}
		return conn.RequireMatchWithData(id, respRaw)
	}
	event := respRaw
	if wrapped, _, _, err := jsonparser.Get(respRaw, "event"); err == nil {
		event = wrapped
	}
	eventType, err := jsonparser.GetString(event, "e")
	if err != nil {
		return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
	}
	if eventType == serverShutdownEvent {
		// Binance.US documents the event as a JSON text frame without its shape, so it is taken whether or not it comes
		// wrapped as an event
		return e.handleServerShutdown(event, conn, e.shutdownWebsocketFor(conn))
	}
	return e.handleUserDataEvent(ctx, conn, respRaw, event, eventType)
}

// acquireWebsocketAPIConnection returns the WebSocket API connection while the websocket is connected, counting a
// request in flight on it until release is called, so that a serverShutdown event closes the connection only once the
// request returns. The connection kept is the one last connected
func (e *Exchange) acquireWebsocketAPIConnection() (conn websocket.Connection, release func(), err error) {
	if !e.Websocket.IsConnected() {
		return nil, nil, websocket.ErrNotConnected
	}
	conn, release = e.wsAPI.acquire()
	if conn == nil {
		return nil, nil, fmt.Errorf("%w: %s", websocket.ErrNotConnected, websocketAPIMessageFilter)
	}
	// The websocket counts as connected until its shutdown has closed every connection, and a connection whose reads
	// failed or that was closed stays down until a new one replaces it
	if !isConnected(conn) {
		release()
		return nil, nil, fmt.Errorf("%w: %s", websocket.ErrNotConnected, websocketAPIMessageFilter)
	}
	return conn, release, nil
}

// isConnected reports whether a connection is up, taking one that cannot tell as up
func isConnected(conn websocket.Connection) bool {
	c, ok := conn.(interface{ IsConnected() bool })
	return !ok || c.IsConnected()
}

// IsConnected reports whether the WebSocket API connection is established and taking requests
func (w WebsocketAPI) IsConnected() bool {
	_, release, err := w.e.acquireWebsocketAPIConnection()
	if err != nil {
		return false
	}
	release()
	return true
}

// SendWebsocketAPIRequest sends a WebSocket API request drawing on the endpoint limit epl and decodes its result into
// result, which may be nil. Parameters go as text, as REST sends them: Binance.US checks each parameter's text, so it
// takes a number or a boolean sent as a string, and a symbols list as its JSON text
func (e *Exchange) SendWebsocketAPIRequest(ctx context.Context, epl request.EndpointLimit, method string, params url.Values, result any) error {
	conn, release, err := e.acquireWebsocketAPIConnection()
	if err != nil {
		return err
	}
	defer release()
	return e.sendOnWebsocketAPIConnection(ctx, conn, epl, &WebsocketAPIRequest{ID: e.MessageID(), Method: method, Params: websocketAPIParams(params)}, result)
}

// SendAPIKeyWebsocketAPIRequest sends a WebSocket API request carrying the API key without a signature, which
// USER_STREAM methods require, and decodes its result into result, which may be nil
func (e *Exchange) SendAPIKeyWebsocketAPIRequest(ctx context.Context, epl request.EndpointLimit, method string, params url.Values, result any) error {
	conn, release, err := e.acquireWebsocketAPIConnection()
	if err != nil {
		return err
	}
	defer release()
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	req := &WebsocketAPIRequest{ID: e.MessageID(), Method: method, Params: websocketAPIParams(params)}
	req.Params["apiKey"] = creds.Key
	err = e.sendOnWebsocketAPIConnection(ctx, conn, epl, req, result)
	e.noteRejectedKey(creds, err)
	return err
}

// SendSignedWebsocketAPIRequest signs a WebSocket API request with the API key and secret, sends it drawing on the
// endpoint limit epl and decodes its result into result, which may be nil
func (e *Exchange) SendSignedWebsocketAPIRequest(ctx context.Context, epl request.EndpointLimit, method string, params url.Values, result any) error {
	conn, release, err := e.acquireWebsocketAPIConnection()
	if err != nil {
		return err
	}
	defer release()
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	req := &WebsocketAPIRequest{ID: e.MessageID(), Method: method, Params: websocketAPIParams(params)}
	if err := e.rateLimitAndSignWebsocketAPIParams(ctx, epl, req.Params); err != nil {
		return err
	}
	err = e.sendOnWebsocketAPIConnection(ctx, conn, wsConnectionMessageRate, req, result)
	e.noteRejectedKey(creds, err)
	return err
}

// websocketAPIParams returns a request's parameters as the WebSocket API's params object
func websocketAPIParams(values url.Values) map[string]string {
	params := make(map[string]string, len(values)+3)
	for k := range values {
		params[k] = values.Get(k)
	}
	return params
}

// rateLimitAndSignWebsocketAPIParams waits for a SIGNED request's rate limit, then signs it. Binance.US rejects a
// timestamp older than recvWindow when the request arrives, which a wait after signing could exceed when requests queue
// for a pool. The request is then sent on wsConnectionMessageRate, which finds no limiter on the WebSocket API
// connection, so the pool is charged once
func (e *Exchange) rateLimitAndSignWebsocketAPIParams(ctx context.Context, epl request.EndpointLimit, params map[string]string) error {
	if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
		return err
	}
	return e.signWebsocketAPIParams(ctx, params)
}

// signWebsocketAPIParams adds the API key, timestamp and HMAC-SHA256 signature a SIGNED request requires, and the
// default recvWindow unless the caller set one. A signature from an earlier attempt is left out of the new one, since
// Binance.US verifies the signature over every other parameter
func (e *Exchange) signWebsocketAPIParams(ctx context.Context, params map[string]string) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	delete(params, "signature")
	params["apiKey"] = creds.Key
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	if _, ok := params["recvWindow"]; !ok {
		params["recvWindow"] = strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10)
	}
	signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(signaturePayload(params)), []byte(creds.Secret))
	if err != nil {
		return err
	}
	params["signature"] = hex.EncodeToString(signature)
	return nil
}

// signaturePayload returns the parameters as name=value pairs sorted by name and joined by '&', which is what a SIGNED
// WebSocket API request signs; unlike REST's, the values are not percent-encoded
func signaturePayload(params map[string]string) string {
	var sb strings.Builder
	for i, key := range slices.Sorted(maps.Keys(params)) {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(params[key])
	}
	return sb.String()
}

// sendOnWebsocketAPIConnection sends a request on conn and decodes its result into result, which may be nil
func (e *Exchange) sendOnWebsocketAPIConnection(ctx context.Context, conn websocket.Connection, epl request.EndpointLimit, req *WebsocketAPIRequest, result any) error {
	// The connection writes a request whatever its context, so a request cancelled while it waited, such as an order,
	// must not be sent
	if err := ctx.Err(); err != nil {
		return err
	}
	respRaw, err := conn.SendMessageReturnResponse(ctx, epl, req.ID, req)
	if err != nil {
		return err
	}
	var resp *WebsocketAPIResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%w: %s: %w", errWebsocketAPIRequestFailed, req.Method, common.ErrNoResponse)
	}
	if resp.Error != nil {
		return fmt.Errorf("%w: %s status %d: %w", errWebsocketAPIRequestFailed, req.Method, resp.Status, &APIError{Code: resp.Error.Code, Message: resp.Error.Message, Data: resp.Error.Data})
	}
	if resp.Status != http.StatusOK {
		return fmt.Errorf("%w: %s status %d", errWebsocketAPIRequestFailed, req.Method, resp.Status)
	}
	if result == nil {
		return nil
	}
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return common.ErrNoResponse
	}
	return json.Unmarshal(resp.Result, result)
}

// rejectedKeyCode is Binance.US's REJECTED_MBX_KEY error, "Invalid API-key, IP, or permissions for action", after which
// it closes a WebSocket API connection
const rejectedKeyCode = -2015

// isRejectedKey reports whether a WebSocket API response refuses its request's API key
func isRejectedKey(respRaw []byte) bool {
	code, err := jsonparser.GetInt(respRaw, "error", "code")
	return err == nil && code == rejectedKeyCode
}

// stopAuthenticatedWebsocket turns authenticated websocket use off once the WebSocket API refused the configured
// account's User Data Stream subscription, as exchanges do when a websocket login fails: Binance.US closes the
// connection after a refusal, and connecting again would subscribe, and be refused, for as long as the key is refused.
// The connection's reader runs it before it reads that close
func (e *Exchange) stopAuthenticatedWebsocket() {
	e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	log.Errorf(log.WebsocketMgr, "%s WebSocket API refused the configured API key's User Data Stream subscription; authenticated websocket use is turned off", e.Name)
}

// noteRejectedKey has the wrapper send its requests over REST once the WebSocket API refused one with the configured
// account's key, as Binance.US closes the connection after each refusal, which reconnects the websocket
func (e *Exchange) noteRejectedKey(creds *accounts.Credentials, err error) {
	if apiErr := (*APIError)(nil); errors.As(err, &apiErr) && apiErr.Code == rejectedKeyCode && e.isConfiguredAccount(creds) && e.websocketAPIKeyRejected.CompareAndSwap(false, true) {
		log.Warnf(log.WebsocketMgr, "%s WebSocket API refused the configured API key; the wrapper sends its requests over REST", e.Name)
	}
}
