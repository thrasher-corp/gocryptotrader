package binance

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceWebsocketAPIURL = "wss://ws-api.binance.com:443/ws-api/v3"
	// spotWebsocketAPI is the message filter of the spot WebSocket API connection
	spotWebsocketAPI = "Spot-Websocket-API"
	// wsAPIResponseMaxLimit is the least time the WebSocket API connection waits for a response. Binance gives a request
	// 10 seconds before answering -1007, execution status unknown, so a shorter wait reports an order Binance may still
	// place as failed, and a caller that retries can place it twice
	wsAPIResponseMaxLimit = 11 * time.Second
)

var (
	errWebsocketAPIRequestFailed         = errors.New("websocket API request failed")
	errOrderListIDRequired               = errors.New("order list ID or client order list ID required")
	errStopPriceOrTrailingDeltaRequired  = errors.New("stop price or trailing delta required")
	errListenTokenRequired               = errors.New("listen token required")
	errUserDataEventWithoutSubscription  = errors.New("user data event without a subscription ID")
	errUnknownUserDataStreamSubscription = errors.New("unknown user data stream subscription")
)

// wsAPIConnection keeps the WebSocket API connection for the requests sent on it. Looking the connection up through the
// websocket manager waits on the manager's lock, which connecting and flushing hold across every dial, so each order
// request would wait while any connection connects
type wsAPIConnection struct {
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
func (w *wsAPIConnection) set(conn websocket.Connection, responseLimit time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.conn, w.requests, w.retired, w.responseLimit = conn, new(sync.WaitGroup), false, responseLimit
}

// get returns the connection kept for requests, or nil once it has been retired
func (w *wsAPIConnection) get() websocket.Connection {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.retired {
		return nil
	}
	return w.conn
}

// kept returns the connection kept for requests, retired or not
func (w *wsAPIConnection) kept() websocket.Connection {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.conn
}

// acquire returns the connection kept for requests, counting a request in flight on it until release is called, or a
// nil connection once it has been retired
func (w *wsAPIConnection) acquire() (conn websocket.Connection, release func()) {
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
func (w *wsAPIConnection) retire(conn websocket.Connection) (wait func()) {
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

// userDataStreams maps the WebSocket API connection's User Data Stream subscriptions to the asset of the account they
// deliver events for, since events only carry their subscription's ID
type userDataStreams struct {
	mu sync.Mutex
	// pending maps the request IDs of subscription requests in flight to the asset of their account, so the response
	// handler records each subscription before any of its events can be processed. A sent request stays pending until
	// its response or the next connection, since Binance subscribes whatever became of the caller
	pending map[string]asset.Item
	// awaiting holds the pending requests whose senders still wait for their responses. A request that timed out or was
	// cancelled stays pending, for a late response to be recorded, but no longer holds back a retry
	awaiting      map[string]asset.Item
	subscriptions map[uint64]asset.Item
}

// expect registers a subscription request before it is sent
func (u *userDataStreams) expect(requestID string, a asset.Item) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.pending == nil {
		u.pending = make(map[string]asset.Item)
		u.awaiting = make(map[string]asset.Item)
	}
	u.pending[requestID], u.awaiting[requestID] = a, a
}

// returned records that a subscription request's sender stopped waiting for its response
func (u *userDataStreams) returned(requestID string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.awaiting, requestID)
}

// recordResponse records the subscription a successful response to a pending subscription request created
func (u *userDataStreams) recordResponse(requestID string, respRaw []byte) {
	u.mu.Lock()
	defer u.mu.Unlock()
	a, ok := u.pending[requestID]
	if !ok {
		return
	}
	delete(u.pending, requestID)
	delete(u.awaiting, requestID)
	id, err := parseSubscriptionID(respRaw, "result", "subscriptionId")
	if err != nil {
		return
	}
	if u.subscriptions == nil {
		u.subscriptions = make(map[uint64]asset.Item)
	}
	u.subscriptions[id] = a
}

// parseSubscriptionID returns the subscription ID at path
func parseSubscriptionID(data []byte, path ...string) (uint64, error) {
	raw, _, _, err := jsonparser.Get(data, path...)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(raw), 10, 64)
}

// asset returns the asset of a subscription's account
func (u *userDataStreams) asset(subscriptionID uint64) (asset.Item, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	a, ok := u.subscriptions[subscriptionID]
	return a, ok
}

// has reports whether an account's stream is subscribed, or has a subscription request whose sender awaits its
// response. A request whose sender gave up is not counted, so that a retry subscribes the account instead: should the
// given up request's response arrive after a retry found the account without a subscription, the account has two
// subscriptions until the connection ends, and its events arrive twice
func (u *userDataStreams) has(a asset.Item) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, subscribed := range u.subscriptions {
		if subscribed == a {
			return true
		}
	}
	for _, requested := range u.awaiting {
		if requested == a {
			return true
		}
	}
	return false
}

// remove removes an ended subscription
func (u *userDataStreams) remove(subscriptionID uint64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.subscriptions, subscriptionID)
}

// reset forgets every subscription and subscription request, since they end with their connection
func (u *userDataStreams) reset() {
	u.mu.Lock()
	defer u.mu.Unlock()
	clear(u.pending)
	clear(u.awaiting)
	clear(u.subscriptions)
}

// wsParams holds a WebSocket API request's parameters as the JSON types the documentation gives them: strings
// (including decimals), integers, booleans and string arrays. A SIGNED request's signature covers the same text
type wsParams map[string]any

func (p wsParams) setString(key, value string) {
	if value != "" {
		p[key] = value
	}
}

func (p wsParams) setUint(key string, value uint64) {
	if value != 0 {
		p[key] = value
	}
}

// setFloat sends a decimal as a string in plain notation, since Binance rejects exponents
func (p wsParams) setFloat(key string, value float64) {
	if value != 0 {
		p[key] = strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func (p wsParams) setTime(key string, value time.Time) {
	if !value.IsZero() {
		p[key] = value.UnixMilli()
	}
}

func (p wsParams) setBool(key string, value bool) {
	if value {
		p[key] = true
	}
}

// signaturePayload returns the parameters as name=value pairs sorted by name and joined by '&', which is what a SIGNED
// request signs; values are not percent-encoded
func (p wsParams) signaturePayload() (string, error) {
	var sb strings.Builder
	for i, key := range slices.Sorted(maps.Keys(p)) {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(key)
		sb.WriteByte('=')
		switch v := p[key].(type) {
		case string:
			sb.WriteString(v)
		case uint64:
			sb.WriteString(strconv.FormatUint(v, 10))
		case int64:
			sb.WriteString(strconv.FormatInt(v, 10))
		case bool:
			sb.WriteString(strconv.FormatBool(v))
		default:
			raw, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("%w %q: %w", errWebsocketAPIRequestFailed, key, err)
			}
			sb.Write(raw)
		}
	}
	return sb.String(), nil
}

// setSymbols sets symbol for one pair and symbols for several, the two forms the market data methods accept
func (e *Exchange) setSymbols(params wsParams, pairs currency.Pairs) error {
	symbols := make([]string, len(pairs))
	for i := range pairs {
		if pairs[i].IsEmpty() {
			return currency.ErrCurrencyPairEmpty
		}
		var err error
		if symbols[i], err = e.FormatSymbol(pairs[i], asset.Spot); err != nil {
			return err
		}
	}
	switch len(symbols) {
	case 0:
	case 1:
		params["symbol"] = symbols[0]
	default:
		params["symbols"] = symbols
	}
	return nil
}

// setSymbol sets a required symbol
func (e *Exchange) setSymbol(params wsParams, pair currency.Pair) error {
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return err
	}
	params["symbol"] = symbol
	return nil
}

// spotWebsocketAPIConnectionSetup returns the setup of the spot WebSocket API connection, which also delivers the spot
// and margin accounts' User Data Streams. Binance limits WebSocket API requests by the weight pools they share with
// REST, not by a message rate, so the connection has no rate limit of its own and a SIGNED request that has already
// waited for its pool is sent at once
func (e *Exchange) spotWebsocketAPIConnectionSetup(exch *config.Exchange) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                      binanceWebsocketAPIURL,
		Connector:                e.WebsocketConnectAPI,
		Authenticate:             e.WebsocketAuthenticateAPI,
		Handler:                  e.wsHandleSpotAPIData,
		MessageFilter:            spotWebsocketAPI,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         wsAPIResponseLimit(exch),
		SubscriptionsNotRequired: true,
		Optional:                 true,
	}
}

// wsAPIResponseLimit returns how long the WebSocket API connection waits for a response
func wsAPIResponseLimit(exch *config.Exchange) time.Duration {
	return max(exch.WebsocketResponseMaxLimit, wsAPIResponseMaxLimit)
}

// WebsocketConnectAPI connects to the spot WebSocket API and keeps the connection for the requests sent on it
func (e *Exchange) WebsocketConnectAPI(ctx context.Context, conn websocket.Connection) error {
	e.userData.reset()
	e.wsAPI.set(nil, 0)
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
	e.wsAPI.set(conn, wsAPIResponseLimit(e.Config))
	return nil
}

// WebsocketAuthenticateAPI subscribes the WebSocket API connection to the User Data Streams of the enabled spot and
// margin accounts. Failures are reported instead of returned, and tried again in the background: an authentication
// error would take every connection down, while this connection still serves requests without user data
func (e *Exchange) WebsocketAuthenticateAPI(ctx context.Context, conn websocket.Connection) error {
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		if e.CurrencyPairs.IsAssetEnabled(a) != nil {
			continue
		}
		if err := e.subscribeAccountUserDataStream(ctx, conn, a); err != nil {
			e.reportUserDataStreamError(ctx, a, err)
			e.retryAccountUserDataStream(ctx, conn, a, false)
		}
	}
	return nil
}

// subscribeAccountUserDataStream subscribes the connection to the spot or margin account's User Data Stream
func (e *Exchange) subscribeAccountUserDataStream(ctx context.Context, conn websocket.Connection, a asset.Item) error {
	if a == asset.Margin {
		return e.subscribeMarginUserDataStream(ctx, conn)
	}
	_, err := e.subscribeUserDataStream(ctx, conn)
	return err
}

// retryAccountUserDataStream subscribes the WebSocket API connection to the spot or margin account's User Data Stream
// in the background, and again after each failure, while the connection is kept for requests and the account is enabled
// with usable credentials, and until a subscription, or a request for one awaiting its response, is recorded
func (e *Exchange) retryAccountUserDataStream(ctx context.Context, conn websocket.Connection, a asset.Item, immediately bool) {
	e.retryUserDataSubscription(ctx, conn, a, immediately, func(ctx context.Context) (bool, error) {
		enabled := e.CurrencyPairs.IsAssetEnabled(a) == nil
		if e.wsAPI.get() != conn || !enabled || !e.AreCredentialsValid(ctx) || e.userData.has(a) {
			return true, nil
		}
		err := e.subscribeAccountUserDataStream(ctx, conn, a)
		return err == nil, err
	})
}

func (e *Exchange) reportUserDataStreamError(ctx context.Context, a asset.Item, err error) {
	err = fmt.Errorf("%s %s user data stream unavailable: %w", e.Name, a, err)
	if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
		log.Errorf(log.WebsocketMgr, "%s: %s", errSend, err)
	}
}

// IsAPIStreamConnected reports whether the WebSocket API connection is established and taking requests
func (e *Exchange) IsAPIStreamConnected() bool {
	_, release, err := e.acquireWsAPIConnection()
	if err != nil {
		return false
	}
	release()
	return true
}

// wsHandleSpotAPIData routes WebSocket API responses to their requests, and User Data Stream and connection events to
// their handlers
func (e *Exchange) wsHandleSpotAPIData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if id, err := jsonparser.GetString(respRaw, "id"); err == nil {
		e.userData.recordResponse(id, respRaw)
		return conn.RequireMatchWithData(id, respRaw)
	}
	event, _, _, err := jsonparser.Get(respRaw, "event")
	if err != nil {
		return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
	}
	eventType, err := jsonparser.GetString(event, "e")
	if err != nil {
		return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
	}
	if eventType == "serverShutdown" {
		// The WebSocket API connection is optional, so the connection monitor replaces it alone, leaving the streams up
		return e.handleServerShutdown(event, conn, func() error {
			if !isConnected(conn) {
				return nil
			}
			return conn.Shutdown()
		})
	}
	subscriptionID, err := parseSubscriptionID(respRaw, "subscriptionId")
	if err != nil {
		return fmt.Errorf("%w: %s", errUserDataEventWithoutSubscription, respRaw)
	}
	a, ok := e.userData.asset(subscriptionID)
	if !ok {
		return fmt.Errorf("%w %d: %s", errUnknownUserDataStreamSubscription, subscriptionID, respRaw)
	}
	switch eventType {
	case "outboundAccountPosition":
		return e.processAccountPosition(ctx, a, event)
	case "balanceUpdate":
		var resp WebsocketBalanceUpdate
		if err := json.Unmarshal(event, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &resp)
	case "executionReport":
		return e.processExecutionReport(ctx, a, event)
	case "listStatus":
		return e.processListStatus(ctx, a, event)
	case "externalLockUpdate":
		var resp WebsocketExternalLockUpdate
		if err := json.Unmarshal(event, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &resp)
	case "eventStreamTerminated":
		var resp WebsocketEventStreamTerminated
		if err := json.Unmarshal(event, &resp); err != nil {
			return err
		}
		e.userData.remove(subscriptionID)
		log.Warnf(log.WebsocketMgr, "%s %s user data stream ended at %s, resubscribing", e.Name, a, resp.EventTime.Time())
		// A subscription request waits for its response, which this reader has to deliver, so it is made in the
		// background, and made again should it fail
		e.retryAccountUserDataStream(ctx, conn, a, true)
		return nil
	default:
		return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
	}
}

// processAccountPosition stores the balances of a spot outboundAccountPosition event and relays them, and relays a
// margin event as it is. A margin event carries no borrowing, so after a borrow its free balance would count the loan
// as the account's own, and REST's margin snapshot gives no time to order the two by, so margin balances come from REST
// alone. A spot event is ordered against REST snapshots by account update time: one predating the stored snapshot is
// left out, as the snapshot holds its balances and maybe later ones
func (e *Exchange) processAccountPosition(ctx context.Context, a asset.Item, event []byte) error {
	var resp WebsocketAccountPosition
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	if a != asset.Spot {
		return e.Websocket.DataHandler.Send(ctx, &resp)
	}
	updated := resp.LastAccountUpdateTime.Time()
	subAccount := accounts.NewSubAccount(a, "")
	if err := e.withBalanceStore(func() error {
		if updated.Before(e.balanceSnapshotTime(a)) {
			return nil
		}
		codes := make([]currency.Code, len(resp.Balances))
		for i := range resp.Balances {
			codes[i] = resp.Balances[i].Asset
		}
		e.recordBalanceEvent(a, updated, codes...)
		for i := range resp.Balances {
			free := resp.Balances[i].Free.Float64()
			locked := resp.Balances[i].Locked.Float64()
			balance, err := e.Accounts.UpdateBalance(ctx, "", a, resp.Balances[i].Asset, func(b *accounts.Balance) {
				b.Total = free + locked
				b.Free = free
				b.Hold = locked
			})
			if err != nil {
				return err
			}
			subAccount.Balances.Set(resp.Balances[i].Asset, balance)
		}
		return nil
	}); err != nil {
		return err
	}
	if len(subAccount.Balances) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAccount})
}

// keepStreamedSpotBalances puts the stored balance in place of each currency of a spot REST poll's snapshot that the
// user data stream reported at or after asOf, the account update time the snapshot shows: the snapshot predates that
// event, or shares its millisecond and may predate it, while the stream's later events in that millisecond carry the
// latest balance. A currency the snapshot leaves out keeps its stored balance in the same case rather than being
// removed. The user data stream is the configured account's, so it reports whether the snapshot is that account's and
// carries a time to order it by, leaving any other as it is. The caller holds the balance store lock until the snapshot
// is stored
func (e *Exchange) keepStreamedSpotBalances(ctx context.Context, subAccount *accounts.SubAccount, asOf time.Time) bool {
	if asOf.IsZero() {
		return false
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return false
	}
	if streamed := e.GetDefaultCredentials(); streamed == nil || *creds != *streamed {
		return false
	}
	reported := make(map[*currency.Item]currency.Code, len(subAccount.Balances))
	for c := range subAccount.Balances {
		reported[c.Item] = c
	}
	for item, eventTime := range e.balanceEventTimes(asset.Spot) {
		if eventTime.Before(asOf) {
			continue
		}
		c, ok := reported[item]
		if !ok {
			c = item.Currency()
		}
		if stored, err := e.Accounts.GetBalance("", creds, asset.Spot, c); err == nil {
			subAccount.Balances.Set(c, stored)
		}
	}
	return true
}

// processExecutionReport relays an executionReport event as an order update
func (e *Exchange) processExecutionReport(ctx context.Context, a asset.Item, event []byte) error {
	var resp WebsocketExecutionReport
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(resp.Symbol, a, false)
	if err != nil {
		return err
	}
	orderID := strconv.FormatUint(resp.OrderID, 10)
	status, err := stringToOrderStatus(resp.CurrentOrderStatus)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	oType, err := stringToSpotOrderType(resp.OrderType)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	tif, err := order.StringToTimeInForce(resp.TimeInForce)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	clientOrderID := resp.ClientOrderID
	if status == order.Cancelled {
		// A cancellation reports the cancel request's ID in c and the order's own ID in C
		clientOrderID = resp.OriginalClientOrderID
	}
	executed := resp.CumulativeFilledQuantity.Float64()
	quantity := resp.OrderQuantity.Float64()
	var remaining float64
	if quantity > executed {
		remaining = remainingAmount(resp.OrderQuantity, resp.CumulativeFilledQuantity)
	}
	detail := &order.Detail{
		Price:                resp.OrderPrice.Float64(),
		Amount:               quantity,
		TriggerPrice:         resp.StopPrice.Float64(),
		AverageExecutedPrice: averagePrice(resp.CumulativeQuoteAssetTransactedQuantity, resp.CumulativeFilledQuantity),
		QuoteAmount:          resp.QuoteOrderQuantity.Float64(),
		ExecutedAmount:       executed,
		RemainingAmount:      remaining,
		Cost:                 resp.CumulativeQuoteAssetTransactedQuantity.Float64(),
		CostAsset:            pair.Quote,
		FeeAsset:             resp.CommissionAsset,
		Exchange:             e.Name,
		OrderID:              orderID,
		ClientOrderID:        clientOrderID,
		Type:                 oType,
		Side:                 side,
		Status:               status,
		AssetType:            a,
		Date:                 resp.OrderCreationTime.Time(),
		LastUpdated:          resp.TransactionTime.Time(),
		Pair:                 pair,
		TimeInForce:          spotTimeInForce(oType, tif),
	}
	if resp.CurrentExecutionType == "TRADE" {
		// The commission is the trade's alone, so it is reported with the trade rather than as the order's fee
		detail.Trades = []order.TradeHistory{{
			Price:     resp.LastExecutedPrice.Float64(),
			Amount:    resp.LastExecutedQuantity.Float64(),
			Fee:       resp.CommissionAmount.Float64(),
			FeeAsset:  resp.CommissionAsset.String(),
			Exchange:  e.Name,
			TID:       strconv.FormatInt(resp.TradeID, 10),
			Type:      oType,
			Side:      side,
			Timestamp: resp.TransactionTime.Time(),
			IsMaker:   resp.IsMaker,
			Total:     resp.LastQuoteAssetTransactedQuantity.Float64(),
		}}
	}
	return e.Websocket.DataHandler.Send(ctx, detail)
}

// processListStatus relays a listStatus event, and an OCO order list as the order GCT tracks it: SubmitOrder returns an
// OCO order list by its list ID, which only this event updates, since the legs' executionReport events carry their
// own order IDs
func (e *Exchange) processListStatus(ctx context.Context, a asset.Item, event []byte) error {
	var resp WebsocketListStatus
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, &resp); err != nil {
		return err
	}
	if resp.ContingencyType != ocoContingencyType {
		return nil
	}
	pair, err := e.MatchSymbolWithAvailablePairs(resp.Symbol, a, false)
	if err != nil {
		return err
	}
	detail, err := e.orderListDetail(a, resp.OrderListID, resp.ListClientOrderID, resp.ListOrderStatus, resp.TransactionTime.Time(), pair)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: detail.OrderID, Err: err})
	}
	return e.Websocket.DataHandler.Send(ctx, &detail)
}

// acquireWsAPIConnection returns the WebSocket API connection while the websocket is connected, counting a request in
// flight on it until release is called, so that a serverShutdown event closes the connection only once the request
// returns. The connection kept is the one last connected; connecting the WebSocket API again, as the websocket
// reconnects or as the connection monitor replaces it alone, keeps the new one
func (e *Exchange) acquireWsAPIConnection() (conn websocket.Connection, release func(), err error) {
	if !e.Websocket.IsConnected() {
		return nil, nil, websocket.ErrNotConnected
	}
	conn, release = e.wsAPI.acquire()
	if conn == nil {
		return nil, nil, fmt.Errorf("%w: %s", websocket.ErrNotConnected, spotWebsocketAPI)
	}
	// The websocket counts as connected until its shutdown has closed every connection, and a connection whose reads
	// failed or that was closed stays down until a new one replaces it, so orders go over REST rather than to a closed
	// connection
	if !isConnected(conn) {
		release()
		return nil, nil, fmt.Errorf("%w: %s", websocket.ErrNotConnected, spotWebsocketAPI)
	}
	return conn, release, nil
}

// isConnected reports whether a connection is up, taking one that cannot tell as up
func isConnected(conn websocket.Connection) bool {
	c, ok := conn.(interface{ IsConnected() bool })
	return !ok || c.IsConnected()
}

// SendWebsocketAPIRequest sends a WebSocket API request and decodes its result into result, which may be nil
func (e *Exchange) SendWebsocketAPIRequest(ctx context.Context, epl request.EndpointLimit, method string, params map[string]any, result any) error {
	conn, release, err := e.acquireWsAPIConnection()
	if err != nil {
		return err
	}
	defer release()
	return e.sendWsAPIRequest(ctx, conn, epl, &WebsocketAPIRequest{ID: e.MessageID(), Method: method, Params: params}, result)
}

// SendSignedWebsocketAPIRequest signs a WebSocket API request with the API key and secret, sends it and decodes its
// result into result, which may be nil
func (e *Exchange) SendSignedWebsocketAPIRequest(ctx context.Context, epl request.EndpointLimit, method string, params map[string]any, result any) error {
	conn, release, err := e.acquireWsAPIConnection()
	if err != nil {
		return err
	}
	defer release()
	if params == nil {
		params = make(map[string]any)
	}
	if err := e.rateLimitAndSignWsAPIParams(ctx, epl, params); err != nil {
		return err
	}
	return e.sendWsAPIRequest(ctx, conn, wsConnectionMessageRate, &WebsocketAPIRequest{ID: e.MessageID(), Method: method, Params: params}, result)
}

// rateLimitAndSignWsAPIParams waits for a SIGNED request's rate limit, then signs it. Binance rejects a timestamp older
// than recvWindow when the request arrives, which a wait after signing could exceed when requests queue for a pool.
// The request is then sent on wsConnectionMessageRate, which finds no limiter on the WebSocket API connection, so the
// pool is charged once
func (e *Exchange) rateLimitAndSignWsAPIParams(ctx context.Context, epl request.EndpointLimit, params wsParams) error {
	if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
		return err
	}
	return e.signWsAPIParams(ctx, params)
}

// signWsAPIParams adds the API key, timestamp and HMAC-SHA256 signature a SIGNED request requires. A signature from an
// earlier attempt is left out of the new one, since Binance verifies the signature over every other parameter
func (e *Exchange) signWsAPIParams(ctx context.Context, params wsParams) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	delete(params, "signature")
	params["apiKey"] = creds.Key
	params["timestamp"] = time.Now().UnixMilli()
	payload, err := params.signaturePayload()
	if err != nil {
		return err
	}
	signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
	if err != nil {
		return err
	}
	params["signature"] = hex.EncodeToString(signature)
	return nil
}

func (e *Exchange) sendWsAPIRequest(ctx context.Context, conn websocket.Connection, epl request.EndpointLimit, req *WebsocketAPIRequest, result any) error {
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

// sendUserDataStreamSubscription sends a User Data Stream subscription request on the endpoint limit epl, recording the
// subscription against the asset of its account
func (e *Exchange) sendUserDataStreamSubscription(ctx context.Context, conn websocket.Connection, a asset.Item, epl request.EndpointLimit, method string, params wsParams, result any) error {
	// A request cancelled before it is sent creates no subscription to wait for
	if err := ctx.Err(); err != nil {
		return err
	}
	req := &WebsocketAPIRequest{ID: e.MessageID(), Method: method, Params: params}
	e.userData.expect(req.ID, a)
	defer e.userData.returned(req.ID)
	return e.sendWsAPIRequest(ctx, conn, epl, req, result)
}

// GetWebsocketAveragePrice returns the current average price of a symbol (Current average price)
func (e *Exchange) GetWebsocketAveragePrice(ctx context.Context, symbol currency.Pair) (*WebsocketAveragePriceResponse, error) {
	params := wsParams{}
	if err := e.setSymbol(params, symbol); err != nil {
		return nil, err
	}
	var resp *WebsocketAveragePriceResponse
	return resp, e.SendWebsocketAPIRequest(ctx, getCurrentAveragePriceRate, "avgPrice", params, &resp)
}

// GetWebsocketOrderbook returns the order book of a symbol (Order book); limit defaults to 100 levels when zero, and
// symbolStatus filters by trading status (TRADING, HALT or BREAK) when set
func (e *Exchange) GetWebsocketOrderbook(ctx context.Context, symbol currency.Pair, limit uint64, symbolStatus string) (*OrderBookResponse, error) {
	params := wsParams{}
	if err := e.setSymbol(params, symbol); err != nil {
		return nil, err
	}
	params.setUint("limit", limit)
	params.setString("symbolStatus", symbolStatus)
	var resp *OrderBookResponse
	return resp, e.SendWebsocketAPIRequest(ctx, spotOrderbookLimit(limit), "depth", params, &resp)
}

// GetWebsocketCandlestick returns klines of a symbol (Klines)
func (e *Exchange) GetWebsocketCandlestick(ctx context.Context, arg *WebsocketKlinesRequest) ([]CandleStickResponse, error) {
	return e.getWsKlines(ctx, "klines", arg)
}

func (e *Exchange) getWsKlines(ctx context.Context, method string, arg *WebsocketKlinesRequest) ([]CandleStickResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	interval, err := intervalToString(arg.Interval)
	if err != nil {
		return nil, err
	}
	params["interval"] = interval
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params.setTime("startTime", arg.StartTime)
	params.setTime("endTime", arg.EndTime)
	params.setString("timeZone", arg.TimeZone)
	params.setUint("limit", arg.Limit)
	var resp []CandleStickResponse
	return resp, e.SendWebsocketAPIRequest(ctx, getKlineRate, method, params, &resp)
}

// GetWebsocketRollingWindowPriceChanges returns price change statistics over a custom rolling window (Rolling window
// price change statistics)
func (e *Exchange) GetWebsocketRollingWindowPriceChanges(ctx context.Context, arg *WebsocketRollingWindowTickerRequest) ([]PriceChangeStatsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if len(arg.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	params := wsParams{}
	if err := e.setSymbols(params, arg.Symbols); err != nil {
		return nil, err
	}
	if arg.WindowSize != 0 {
		windowSize, err := windowSizeToString(arg.WindowSize)
		if err != nil {
			return nil, err
		}
		params["windowSize"] = windowSize
	}
	params.setString("type", arg.Type)
	params.setString("symbolStatus", arg.SymbolStatus)
	var resp objectOrArray[PriceChangeStatsResponse]
	return resp, e.SendWebsocketAPIRequest(ctx, spotTickerSymbolsLimit(len(arg.Symbols)), "ticker", params, &resp)
}

// windowSizeToString formats a rolling window as a whole number of minutes, hours or days, the units the ticker
// method accepts
func windowSizeToString(windowSize time.Duration) (string, error) {
	const day = 24 * time.Hour
	switch {
	case windowSize >= day && windowSize <= 7*day && windowSize%day == 0:
		return strconv.FormatInt(int64(windowSize/day), 10) + "d", nil
	case windowSize >= time.Hour && windowSize < day && windowSize%time.Hour == 0:
		return strconv.FormatInt(int64(windowSize/time.Hour), 10) + "h", nil
	case windowSize >= time.Minute && windowSize < time.Hour && windowSize%time.Minute == 0:
		return strconv.FormatInt(int64(windowSize/time.Minute), 10) + "m", nil
	}
	return "", fmt.Errorf("%w: %s", errInvalidWindowSize, windowSize)
}

// GetWebsocket24HourPriceChanges returns 24 hour rolling window price change statistics (24hr ticker price change
// statistics); every symbol's when arg.Symbols is empty
func (e *Exchange) GetWebsocket24HourPriceChanges(ctx context.Context, arg *WebsocketTickerRequest) ([]PriceChangeStatsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := wsParams{}
	if err := e.setSymbols(params, arg.Symbols); err != nil {
		return nil, err
	}
	params.setString("type", arg.Type)
	params.setString("symbolStatus", arg.SymbolStatus)
	var resp objectOrArray[PriceChangeStatsResponse]
	return resp, e.SendWebsocketAPIRequest(ctx, spotTicker24HourLimit(len(arg.Symbols)), "ticker.24hr", params, &resp)
}

// GetWebsocketSymbolOrderbookTicker returns the best bid and ask of symbols (Symbol order book ticker); every symbol's
// when symbols is empty. symbolStatus filters by trading status (TRADING, HALT or BREAK) when set
func (e *Exchange) GetWebsocketSymbolOrderbookTicker(ctx context.Context, symbols currency.Pairs, symbolStatus string) ([]WebsocketBookTickerResponse, error) {
	params := wsParams{}
	if err := e.setSymbols(params, symbols); err != nil {
		return nil, err
	}
	params.setString("symbolStatus", symbolStatus)
	epl := spotOrderbookTickerAllRate
	if len(symbols) == 1 {
		epl = spotBookTickerRate
	}
	var resp objectOrArray[WebsocketBookTickerResponse]
	return resp, e.SendWebsocketAPIRequest(ctx, epl, "ticker.book", params, &resp)
}

// GetWebsocketSymbolPriceTicker returns the latest price of symbols (Symbol price ticker); every symbol's when symbols
// is empty. symbolStatus filters by trading status (TRADING, HALT or BREAK) when set
func (e *Exchange) GetWebsocketSymbolPriceTicker(ctx context.Context, symbols currency.Pairs, symbolStatus string) ([]WebsocketPriceTickerResponse, error) {
	params := wsParams{}
	if err := e.setSymbols(params, symbols); err != nil {
		return nil, err
	}
	params.setString("symbolStatus", symbolStatus)
	epl := spotSymbolPriceAllRate
	if len(symbols) == 1 {
		epl = spotSymbolPriceRate
	}
	var resp objectOrArray[WebsocketPriceTickerResponse]
	return resp, e.SendWebsocketAPIRequest(ctx, epl, "ticker.price", params, &resp)
}

// GetWebsocketTradingDayTickers returns price change statistics for the current trading day (Trading Day Ticker)
func (e *Exchange) GetWebsocketTradingDayTickers(ctx context.Context, arg *WebsocketTradingDayTickerRequest) ([]PriceChangeStatsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if len(arg.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	params := wsParams{}
	if err := e.setSymbols(params, arg.Symbols); err != nil {
		return nil, err
	}
	params.setString("timeZone", arg.TimeZone)
	params.setString("type", arg.Type)
	params.setString("symbolStatus", arg.SymbolStatus)
	var resp objectOrArray[PriceChangeStatsResponse]
	return resp, e.SendWebsocketAPIRequest(ctx, spotTickerSymbolsLimit(len(arg.Symbols)), "ticker.tradingDay", params, &resp)
}

// GetWebsocketAggregatedTrades returns aggregate trades of a symbol (Aggregate trades)
func (e *Exchange) GetWebsocketAggregatedTrades(ctx context.Context, arg *WebsocketAggregatedTradesRequest) ([]AggregatedTradeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params.setUint("fromId", arg.FromID)
	params.setTime("startTime", arg.StartTime)
	params.setTime("endTime", arg.EndTime)
	params.setUint("limit", arg.Limit)
	var resp []AggregatedTradeResponse
	return resp, e.SendWebsocketAPIRequest(ctx, aggTradesRate, "trades.aggregate", params, &resp)
}

// GetWebsocketMostRecentTrades returns the most recent trades of a symbol (Recent trades); limit defaults to 500 when
// zero
func (e *Exchange) GetWebsocketMostRecentTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]RecentTradeResponse, error) {
	params := wsParams{}
	if err := e.setSymbol(params, symbol); err != nil {
		return nil, err
	}
	params.setUint("limit", limit)
	var resp []RecentTradeResponse
	return resp, e.SendWebsocketAPIRequest(ctx, getRecentTradesListRate, "trades.recent", params, &resp)
}

// GetWebsocketOptimisedCandlestick returns klines of a symbol modified for the presentation of candlestick charts (UI
// Klines)
func (e *Exchange) GetWebsocketOptimisedCandlestick(ctx context.Context, arg *WebsocketKlinesRequest) ([]CandleStickResponse, error) {
	return e.getWsKlines(ctx, "uiKlines", arg)
}

// WebsocketLogOutOfSession forgets the API key a session.logon authenticated the session with (Log out of the session)
func (e *Exchange) WebsocketLogOutOfSession(ctx context.Context) (*WebsocketSessionStatusResponse, error) {
	var resp *WebsocketSessionStatusResponse
	return resp, e.SendWebsocketAPIRequest(ctx, wsSessionRate, "session.logout", nil, &resp)
}

// GetWebsocketSessionStatus returns the WebSocket API session's status, including the API key authenticating it if any
// (Query session status)
func (e *Exchange) GetWebsocketSessionStatus(ctx context.Context) (*WebsocketSessionStatusResponse, error) {
	var resp *WebsocketSessionStatusResponse
	return resp, e.SendWebsocketAPIRequest(ctx, wsSessionRate, "session.status", nil, &resp)
}

// WebsocketCancelOpenOrders cancels every open order of a symbol, including order lists (Cancel open orders)
func (e *Exchange) WebsocketCancelOpenOrders(ctx context.Context, symbol currency.Pair) ([]WebsocketCancelOrderResponse, error) {
	params := wsParams{}
	if err := e.setSymbol(params, symbol); err != nil {
		return nil, err
	}
	var resp []WebsocketCancelOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotCancelRate, "openOrders.cancelAll", params, &resp)
}

// WebsocketCancelExistingOrder cancels an order (Cancel order); cancelling an order list's leg cancels the whole list
func (e *Exchange) WebsocketCancelExistingOrder(ctx context.Context, arg *WebsocketCancelOrderRequest) (*WebsocketCancelOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 && arg.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params.setUint("orderId", arg.OrderID)
	params.setString("origClientOrderId", arg.OriginalClientOrderID)
	params.setString("newClientOrderId", arg.NewClientOrderID)
	params.setString("cancelRestrictions", arg.CancelRestrictions)
	var resp *WebsocketCancelOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotCancelRate, "order.cancel", params, &resp)
}

// WebsocketCancelAndReplaceTradeOrder cancels an order and places a new order in its place (Cancel and replace order)
func (e *Exchange) WebsocketCancelAndReplaceTradeOrder(ctx context.Context, arg *WebsocketCancelReplaceOrderRequest) (*WebsocketCancelReplaceOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CancelReplaceMode == "" {
		return nil, errCancelReplaceModeRequired
	}
	if arg.CancelOrderID == 0 && arg.CancelOriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params["cancelReplaceMode"] = arg.CancelReplaceMode
	params.setUint("cancelOrderId", arg.CancelOrderID)
	params.setString("cancelOrigClientOrderId", arg.CancelOriginalClientOrderID)
	params.setString("cancelNewClientOrderId", arg.CancelNewClientOrderID)
	params["side"] = arg.Side
	params["type"] = arg.Type
	params.setString("timeInForce", arg.TimeInForce)
	params.setFloat("price", arg.Price)
	params.setFloat("quantity", arg.Quantity)
	params.setFloat("quoteOrderQty", arg.QuoteOrderQuantity)
	params.setString("newClientOrderId", arg.NewClientOrderID)
	params.setString("newOrderRespType", arg.NewOrderResponseType)
	params.setFloat("stopPrice", arg.StopPrice)
	params.setUint("trailingDelta", arg.TrailingDelta)
	params.setFloat("icebergQty", arg.IcebergQuantity)
	params.setUint("strategyId", arg.StrategyID)
	params.setUint("strategyType", arg.StrategyType)
	params.setString("selfTradePreventionMode", arg.SelfTradePreventionMode)
	params.setString("cancelRestrictions", arg.CancelRestrictions)
	params.setString("orderRateLimitExceededMode", arg.OrderRateLimitExceededMode)
	params.setString("pegPriceType", arg.PegPriceType)
	params.setUint("pegOffsetValue", arg.PegOffsetValue)
	params.setString("pegOffsetType", arg.PegOffsetType)
	var resp *WebsocketCancelReplaceOrderResponse
	err := e.SendSignedWebsocketAPIRequest(ctx, spotOrderRate, "order.cancelReplace", params, &resp)
	return cancelReplaceOutcome(resp, err)
}

// WebsocketCancelOCOOrder cancels an order list (Cancel Order list)
func (e *Exchange) WebsocketCancelOCOOrder(ctx context.Context, arg *WebsocketCancelOrderListRequest) (*WebsocketCancelOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderListID == 0 && arg.ListClientOrderID == "" {
		return nil, errOrderListIDRequired
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params.setUint("orderListId", arg.OrderListID)
	params.setString("listClientOrderId", arg.ListClientOrderID)
	params.setString("newClientOrderId", arg.NewClientOrderID)
	var resp *WebsocketCancelOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotCancelRate, "orderList.cancel", params, &resp)
}

// WebsocketPlaceOCOOrder places a one-cancels-the-other order list: a LIMIT_MAKER order and a STOP_LOSS or
// STOP_LOSS_LIMIT order, where the activation of one cancels the other (Place new OCO - Deprecated). Binance deprecated
// the method in favour of orderList.place.oco but still serves it
func (e *Exchange) WebsocketPlaceOCOOrder(ctx context.Context, arg *WebsocketPlaceOCOOrderRequest) (*WebsocketOrderListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.StopPrice <= 0 && arg.TrailingDelta == 0 {
		return nil, errStopPriceOrTrailingDeltaRequired
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params["side"] = arg.Side
	params.setFloat("price", arg.Price)
	params.setFloat("quantity", arg.Quantity)
	params.setString("listClientOrderId", arg.ListClientOrderID)
	params.setString("limitClientOrderId", arg.LimitClientOrderID)
	params.setFloat("limitIcebergQty", arg.LimitIcebergQuantity)
	params.setUint("limitStrategyId", arg.LimitStrategyID)
	params.setUint("limitStrategyType", arg.LimitStrategyType)
	params.setFloat("stopPrice", arg.StopPrice)
	params.setUint("trailingDelta", arg.TrailingDelta)
	params.setString("stopClientOrderId", arg.StopClientOrderID)
	params.setFloat("stopLimitPrice", arg.StopLimitPrice)
	params.setString("stopLimitTimeInForce", arg.StopLimitTimeInForce)
	params.setFloat("stopIcebergQty", arg.StopIcebergQuantity)
	params.setUint("stopStrategyId", arg.StopStrategyID)
	params.setUint("stopStrategyType", arg.StopStrategyType)
	params.setString("newOrderRespType", arg.NewOrderResponseType)
	params.setString("selfTradePreventionMode", arg.SelfTradePreventionMode)
	var resp *WebsocketOrderListResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotOCOOrderRate, "orderList.place", params, &resp)
}

// WebsocketPlaceOCOOrderList places a one-cancels-the-other order list of an above and a below order, where the
// activation of one order cancels the other (Place new Order list - OCO). One order must be LIMIT_MAKER, TAKE_PROFIT or
// TAKE_PROFIT_LIMIT and the other STOP_LOSS or STOP_LOSS_LIMIT, and the list counts as two orders against the unfilled
// order count
func (e *Exchange) WebsocketPlaceOCOOrderList(ctx context.Context, arg *OCOOrderListRequest) (*WebsocketOrderListResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Above.Type == "" {
		return nil, fmt.Errorf("%w: aboveType is required", order.ErrTypeIsInvalid)
	}
	if arg.Below.Type == "" {
		return nil, fmt.Errorf("%w: belowType is required", order.ErrTypeIsInvalid)
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params["side"] = arg.Side
	params.setFloat("quantity", arg.Quantity)
	params.setString("listClientOrderId", arg.ListClientOrderID)
	setWsOCOLegParams(params, "above", &arg.Above)
	setWsOCOLegParams(params, "below", &arg.Below)
	params.setString("newOrderRespType", arg.NewOrderResponseType)
	params.setString("selfTradePreventionMode", arg.SelfTradePreventionMode)
	var resp *WebsocketOrderListResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotOCOOrderRate, "orderList.place.oco", params, &resp)
}

// setWsOCOLegParams sets the parameters of an order list's above or below order, which carry the order's position as a
// prefix
func setWsOCOLegParams(params wsParams, leg string, arg *OCOOrderListLeg) {
	params[leg+"Type"] = arg.Type
	params.setString(leg+"ClientOrderId", arg.ClientOrderID)
	params.setUint(leg+"IcebergQty", arg.IcebergQuantity)
	params.setFloat(leg+"Price", arg.Price)
	params.setFloat(leg+"StopPrice", arg.StopPrice)
	params.setUint(leg+"TrailingDelta", arg.TrailingDelta)
	params.setString(leg+"TimeInForce", arg.TimeInForce)
	params.setUint(leg+"StrategyId", arg.StrategyID)
	params.setUint(leg+"StrategyType", arg.StrategyType)
	params.setString(leg+"PegPriceType", arg.PegPriceType)
	params.setString(leg+"PegOffsetType", arg.PegOffsetType)
	params.setUint(leg+"PegOffsetValue", arg.PegOffsetValue)
}

// orderParams validates an order.place or order.test request and returns its parameters
func (e *Exchange) orderParams(arg *WebsocketPlaceOrderRequest) (wsParams, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params["side"] = arg.Side
	params["type"] = arg.Type
	params.setString("timeInForce", arg.TimeInForce)
	params.setFloat("price", arg.Price)
	params.setFloat("quantity", arg.Quantity)
	params.setFloat("quoteOrderQty", arg.QuoteOrderQuantity)
	params.setString("newClientOrderId", arg.NewClientOrderID)
	params.setString("newOrderRespType", arg.NewOrderResponseType)
	params.setFloat("stopPrice", arg.StopPrice)
	params.setUint("trailingDelta", arg.TrailingDelta)
	params.setFloat("icebergQty", arg.IcebergQuantity)
	params.setUint("strategyId", arg.StrategyID)
	params.setUint("strategyType", arg.StrategyType)
	params.setString("selfTradePreventionMode", arg.SelfTradePreventionMode)
	params.setString("pegPriceType", arg.PegPriceType)
	params.setUint("pegOffsetValue", arg.PegOffsetValue)
	params.setString("pegOffsetType", arg.PegOffsetType)
	return params, nil
}

// WebsocketPlaceNewOrder places an order (Place new order)
func (e *Exchange) WebsocketPlaceNewOrder(ctx context.Context, arg *WebsocketPlaceOrderRequest) (*WebsocketOrderResponse, error) {
	params, err := e.orderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *WebsocketOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotOrderRate, "order.place", params, &resp)
}

// WebsocketTestNewOrder validates an order and the request's signature without sending the order to the matching engine
// (Test new order); the order's commission rates are returned when computeCommissionRates is set, and nil otherwise
func (e *Exchange) WebsocketTestNewOrder(ctx context.Context, arg *WebsocketPlaceOrderRequest, computeCommissionRates bool) (*WebsocketOrderCommissionRatesResponse, error) {
	params, err := e.orderParams(arg)
	if err != nil {
		return nil, err
	}
	if !computeCommissionRates {
		return nil, e.SendSignedWebsocketAPIRequest(ctx, spotDefaultRate, "order.test", params, nil)
	}
	params["computeCommissionRates"] = true
	var resp *WebsocketOrderCommissionRatesResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, testNewOrderWithCommissionRate, "order.test", params, &resp)
}

// wsSOROrderParams validates a sor.order.place or sor.order.test request and returns its parameters
func (e *Exchange) wsSOROrderParams(arg *WebsocketSOROrderRequest) (wsParams, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params["side"] = arg.Side
	params["type"] = arg.Type
	params.setString("timeInForce", arg.TimeInForce)
	params.setFloat("price", arg.Price)
	params.setFloat("quantity", arg.Quantity)
	params.setString("newClientOrderId", arg.NewClientOrderID)
	params.setString("newOrderRespType", arg.NewOrderResponseType)
	params.setFloat("icebergQty", arg.IcebergQuantity)
	params.setUint("strategyId", arg.StrategyID)
	params.setUint("strategyType", arg.StrategyType)
	params.setString("selfTradePreventionMode", arg.SelfTradePreventionMode)
	return params, nil
}

// WebsocketPlaceNewSOROrder places an order using smart order routing (Place new order using SOR)
func (e *Exchange) WebsocketPlaceNewSOROrder(ctx context.Context, arg *WebsocketSOROrderRequest) ([]WebsocketSOROrderResponse, error) {
	params, err := e.wsSOROrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp []WebsocketSOROrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotOrderRate, "sor.order.place", params, &resp)
}

// WebsocketTestNewOrderUsingSOR validates an order using smart order routing and the request's signature without
// sending the order to the matching engine (Test new order using SOR); the order's commission rates are returned when
// computeCommissionRates is set, and nil otherwise
func (e *Exchange) WebsocketTestNewOrderUsingSOR(ctx context.Context, arg *WebsocketSOROrderRequest, computeCommissionRates bool) (*WebsocketSOROrderCommissionRatesResponse, error) {
	params, err := e.wsSOROrderParams(arg)
	if err != nil {
		return nil, err
	}
	if !computeCommissionRates {
		return nil, e.SendSignedWebsocketAPIRequest(ctx, spotDefaultRate, "sor.order.test", params, nil)
	}
	params["computeCommissionRates"] = true
	var resp *WebsocketSOROrderCommissionRatesResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, testNewOrderWithCommissionRate, "sor.order.test", params, &resp)
}

// WebsocketAccountCommissionRates returns the account's commission rates for a symbol (Account Commission Rates)
func (e *Exchange) WebsocketAccountCommissionRates(ctx context.Context, symbol currency.Pair) (*WebsocketAccountCommissionResponse, error) {
	params := wsParams{}
	if err := e.setSymbol(params, symbol); err != nil {
		return nil, err
	}
	var resp *WebsocketAccountCommissionResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, getCommissionRate, "account.commission", params, &resp)
}

// WebsocketQueryAccountOrderRateLimits returns the account's unfilled order counts against the order rate limits
// (Unfilled Order Count)
func (e *Exchange) WebsocketQueryAccountOrderRateLimits(ctx context.Context) ([]WebsocketRateLimitResponse, error) {
	var resp []WebsocketRateLimitResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, currentOrderCountUsageRate, "account.rateLimits.orders", nil, &resp)
}

// GetWebsocketAccountInfo returns the account's information and balances (Account information); omitZeroBalances leaves
// out the zero balances
func (e *Exchange) GetWebsocketAccountInfo(ctx context.Context, omitZeroBalances bool) (*AccountResponse, error) {
	params := wsParams{}
	params.setBool("omitZeroBalances", omitZeroBalances)
	var resp *AccountResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotAccountInformationRate, "account.status", params, &resp)
}

// WebsocketQueryAccountOCOOrderHistory returns the account's order lists, filtered by time range or starting from an
// order list ID (Account order list history)
func (e *Exchange) WebsocketQueryAccountOCOOrderHistory(ctx context.Context, arg *WebsocketOrderListHistoryRequest) ([]OCOOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := wsParams{}
	params.setUint("fromId", arg.FromID)
	params.setTime("startTime", arg.StartTime)
	params.setTime("endTime", arg.EndTime)
	params.setUint("limit", arg.Limit)
	var resp []OCOOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, getAllOCOOrdersRate, "allOrderLists", params, &resp)
}

// WebsocketQueryAccountOrderHistory returns the account's orders of a symbol, filtered by time range or starting from
// an order ID (Account order history)
func (e *Exchange) WebsocketQueryAccountOrderHistory(ctx context.Context, arg *WebsocketOrderHistoryRequest) ([]TradeOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params.setUint("orderId", arg.OrderID)
	params.setTime("startTime", arg.StartTime)
	params.setTime("endTime", arg.EndTime)
	params.setUint("limit", arg.Limit)
	var resp []TradeOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotAllOrdersRate, "allOrders", params, &resp)
}

// WebsocketAccountAllocation returns the account's allocations from SOR order placement (Account allocations)
func (e *Exchange) WebsocketAccountAllocation(ctx context.Context, arg *WebsocketAllocationsRequest) ([]WebsocketAllocationResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params.setTime("startTime", arg.StartTime)
	params.setTime("endTime", arg.EndTime)
	params.setUint("fromAllocationId", arg.FromAllocationID)
	params.setUint("limit", arg.Limit)
	params.setUint("orderId", arg.OrderID)
	var resp []WebsocketAllocationResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, getAllocationsRate, "myAllocations", params, &resp)
}

// WebsocketAccountPreventedMatches returns the account's orders expired by self-trade prevention (Account prevented
// matches)
func (e *Exchange) WebsocketAccountPreventedMatches(ctx context.Context, arg *WebsocketPreventedMatchesRequest) ([]WebsocketPreventedMatchResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PreventedMatchID == 0 && arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	params.setUint("preventedMatchId", arg.PreventedMatchID)
	params.setUint("orderId", arg.OrderID)
	params.setUint("fromPreventedMatchId", arg.FromPreventedMatchID)
	params.setUint("limit", arg.Limit)
	epl := preventedMatchesRate
	if arg.PreventedMatchID == 0 {
		epl = preventedMatchesByOrderIDRate
	}
	var resp []WebsocketPreventedMatchResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, epl, "myPreventedMatches", params, &resp)
}

// WebsocketAccountTradeHistory returns the account's trades of a symbol (Account trade history)
func (e *Exchange) WebsocketAccountTradeHistory(ctx context.Context, arg *WebsocketAccountTradesRequest) ([]WebsocketAccountTradeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := wsParams{}
	if err := e.setSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params.setUint("orderId", arg.OrderID)
	params.setTime("startTime", arg.StartTime)
	params.setTime("endTime", arg.EndTime)
	params.setUint("fromId", arg.FromID)
	params.setUint("limit", arg.Limit)
	var resp []WebsocketAccountTradeResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotAccountTradeListLimit(arg.OrderID), "myTrades", params, &resp)
}

// WebsocketCurrentOpenOCOOrders returns the account's open order lists (Current open Order lists)
func (e *Exchange) WebsocketCurrentOpenOCOOrders(ctx context.Context) ([]OCOOrderResponse, error) {
	var resp []OCOOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, getOpenOCOListRate, "openOrderLists.status", nil, &resp)
}

// WebsocketCurrentOpenOrders returns the account's open orders of a symbol, or of every symbol when symbol is empty
// (Current open orders)
func (e *Exchange) WebsocketCurrentOpenOrders(ctx context.Context, symbol currency.Pair) ([]TradeOrderResponse, error) {
	return e.wsCurrentOpenOrders(ctx, symbol, spotOpenOrdersAllRate)
}

// wsCurrentOpenOrders lists open orders over the WebSocket API, drawing on allSymbols for a listing of every symbol's,
// so the listing a cancellation of every symbol's orders makes can draw on the cancel pool
func (e *Exchange) wsCurrentOpenOrders(ctx context.Context, symbol currency.Pair, allSymbols request.EndpointLimit) ([]TradeOrderResponse, error) {
	params := wsParams{}
	epl := allSymbols
	if !symbol.IsEmpty() {
		if err := e.setSymbol(params, symbol); err != nil {
			return nil, err
		}
		epl = spotOpenOrdersSpecificRate
	}
	var resp []TradeOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, epl, "openOrders.status", params, &resp)
}

// WebsocketQueryOCOOrder returns an order list by its order list ID or client order list ID (Query Order list)
func (e *Exchange) WebsocketQueryOCOOrder(ctx context.Context, orderListID uint64, listClientOrderID string) (*OCOOrderResponse, error) {
	if orderListID == 0 && listClientOrderID == "" {
		return nil, errOrderListIDRequired
	}
	params := wsParams{}
	params.setUint("orderListId", orderListID)
	// The method takes the client order list ID as origClientOrderId
	params.setString("origClientOrderId", listClientOrderID)
	var resp *OCOOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, getOCOListRate, "orderList.status", params, &resp)
}

// WebsocketQueryOrder returns an order by its order ID or client order ID (Query order)
func (e *Exchange) WebsocketQueryOrder(ctx context.Context, symbol currency.Pair, orderID uint64, originalClientOrderID string) (*TradeOrderResponse, error) {
	if orderID == 0 && originalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := wsParams{}
	if err := e.setSymbol(params, symbol); err != nil {
		return nil, err
	}
	params.setUint("orderId", orderID)
	params.setString("origClientOrderId", originalClientOrderID)
	var resp *TradeOrderResponse
	return resp, e.SendSignedWebsocketAPIRequest(ctx, spotOrderQueryRate, "order.status", params, &resp)
}

// WebsocketSubscribeUserDataStream subscribes the WebSocket API connection to the spot account's User Data Stream
// (Subscribe to User Data Stream through signature subscription). It needs no session logon, so it works with every API
// key type
func (e *Exchange) WebsocketSubscribeUserDataStream(ctx context.Context) (*WebsocketUserDataStreamSubscriptionResponse, error) {
	conn, release, err := e.acquireWsAPIConnection()
	if err != nil {
		return nil, err
	}
	defer release()
	return e.subscribeUserDataStream(ctx, conn)
}

func (e *Exchange) subscribeUserDataStream(ctx context.Context, conn websocket.Connection) (*WebsocketUserDataStreamSubscriptionResponse, error) {
	params := wsParams{}
	if err := e.rateLimitAndSignWsAPIParams(ctx, userDataStreamSubscribeRate, params); err != nil {
		return nil, err
	}
	var resp *WebsocketUserDataStreamSubscriptionResponse
	return resp, e.sendUserDataStreamSubscription(ctx, conn, asset.Spot, wsConnectionMessageRate, "userDataStream.subscribe.signature", params, &resp)
}

// WebsocketSubscribeUserDataStreamWithListenToken subscribes the WebSocket API connection to the margin account User
// Data Stream a listen token from CreateMarginListenToken gives access to (userDataStream.subscribe.listenToken). The
// subscription ends when the token expires unless it is renewed with a new token
func (e *Exchange) WebsocketSubscribeUserDataStreamWithListenToken(ctx context.Context, listenToken string) (*WebsocketListenTokenSubscriptionResponse, error) {
	conn, release, err := e.acquireWsAPIConnection()
	if err != nil {
		return nil, err
	}
	defer release()
	return e.subscribeListenToken(ctx, conn, listenToken)
}

func (e *Exchange) subscribeListenToken(ctx context.Context, conn websocket.Connection, listenToken string) (*WebsocketListenTokenSubscriptionResponse, error) {
	if listenToken == "" {
		return nil, errListenTokenRequired
	}
	var resp *WebsocketListenTokenSubscriptionResponse
	return resp, e.sendUserDataStreamSubscription(ctx, conn, asset.Margin, userDataStreamSubscribeRate, "userDataStream.subscribe.listenToken", wsParams{"listenToken": listenToken}, &resp)
}

// subscribeMarginUserDataStream subscribes the connection to the cross margin account's User Data Stream with a new
// listen token. The token's default validity of 24 hours matches the connection's lifetime, and a subscription that
// ends first is renewed when its eventStreamTerminated event arrives
func (e *Exchange) subscribeMarginUserDataStream(ctx context.Context, conn websocket.Connection) error {
	token, err := e.CreateMarginListenToken(ctx, &MarginListenTokenRequest{})
	if err != nil {
		return err
	}
	_, err = e.subscribeListenToken(ctx, conn, token.Token)
	return err
}
