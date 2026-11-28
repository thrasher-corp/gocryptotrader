package binanceus

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errUserDataEventWithoutSubscription  = errors.New("user data event without a subscription ID")
	errUnknownUserDataStreamSubscription = errors.New("unknown user data stream subscription")
)

// A failed User Data Stream subscription is tried again after userDataRetryDelay, then after twice the last wait each
// time, up to userDataRetryMaxDelay
const (
	userDataRetryDelay    = 5 * time.Second
	userDataRetryMaxDelay = 5 * time.Minute
)

// userDataStreams tracks the WebSocket API connection's User Data Stream subscriptions by the credentials of the account
// each delivers events for, since events carry only their subscription's ID
type userDataStreams struct {
	mu sync.Mutex
	// pending maps the request IDs of subscription requests in flight to the credentials they were signed with, so the
	// response handler records each subscription before any of its events can be processed. A sent request stays pending
	// until its response or the next connection, since Binance.US subscribes whatever became of the caller
	pending map[string]*accounts.Credentials
	// awaiting holds the pending requests whose senders still wait for their responses. A request that timed out or was
	// cancelled stays pending, for a late response to be recorded, but no longer holds back a retry
	awaiting      map[string]*accounts.Credentials
	subscriptions map[uint64]*accounts.Credentials
	// unsubscribed records that a caller unsubscribed the configured account's stream, so that neither a retry nor a new
	// connection subscribes it again until a caller does
	unsubscribed bool
}

// expect registers a subscription request signed with creds before it is sent
func (u *userDataStreams) expect(requestID string, creds *accounts.Credentials) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.pending == nil {
		u.pending = make(map[string]*accounts.Credentials)
		u.awaiting = make(map[string]*accounts.Credentials)
	}
	u.pending[requestID], u.awaiting[requestID] = creds, creds
}

// returned records that a subscription request's sender stopped waiting for its response
func (u *userDataStreams) returned(requestID string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.awaiting, requestID)
}

// recordResponse records the subscription a successful response to a pending subscription request created, and returns
// the credentials a pending request was signed with, or nil for a response to another request
func (u *userDataStreams) recordResponse(requestID string, respRaw []byte) *accounts.Credentials {
	u.mu.Lock()
	defer u.mu.Unlock()
	creds, ok := u.pending[requestID]
	if !ok {
		return nil
	}
	delete(u.pending, requestID)
	delete(u.awaiting, requestID)
	id, err := parseSubscriptionID(respRaw, "result", "subscriptionId")
	if err != nil {
		return creds
	}
	if u.subscriptions == nil {
		u.subscriptions = make(map[uint64]*accounts.Credentials)
	}
	u.subscriptions[id] = creds
	return creds
}

// parseSubscriptionID returns the subscription ID at path
func parseSubscriptionID(data []byte, path ...string) (uint64, error) {
	raw, _, _, err := jsonparser.Get(data, path...)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(raw), 10, 64)
}

// credentials returns the credentials of a subscription's account
func (u *userDataStreams) credentials(subscriptionID uint64) (*accounts.Credentials, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	creds, ok := u.subscriptions[subscriptionID]
	return creds, ok
}

// has reports whether the account of creds has a subscription, or a subscription request whose sender awaits its
// response. A request whose sender gave up is not counted, so that a retry subscribes the account instead: should the
// given up request's response arrive after a retry found the account without a subscription, Binance.US refuses the
// retry's, as it holds one subscription per account on a connection
func (u *userDataStreams) has(creds *accounts.Credentials) bool {
	if creds == nil {
		return false
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, subscribed := range u.subscriptions {
		if *subscribed == *creds {
			return true
		}
	}
	for _, requested := range u.awaiting {
		if *requested == *creds {
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

// setWanted records whether the configured account's stream is wanted
func (u *userDataStreams) setWanted(wanted bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.unsubscribed = !wanted
}

// isWanted reports whether the configured account's stream is wanted: it is unless a caller unsubscribed it
func (u *userDataStreams) isWanted() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return !u.unsubscribed
}

// userDataRetries tracks the connection a User Data Stream subscription is being tried again on, so that one retry runs
// at a time and a retry on a replaced connection cannot end its successor's
type userDataRetries struct {
	mu      sync.Mutex
	running websocket.Connection
	// delay is the first wait before a retry, userDataRetryDelay when zero
	delay time.Duration
}

// WebsocketAuthenticateAPI subscribes the WebSocket API connection to the configured account's User Data Stream, unless
// a caller unsubscribed it. A failure is reported instead of returned and tried again in the background: an
// authentication error would take every connection down, while this connection still serves requests without user data
func (e *Exchange) WebsocketAuthenticateAPI(ctx context.Context, conn websocket.Connection) error {
	if !e.userData.isWanted() {
		return nil
	}
	if err := e.subscribeWantedUserDataStream(ctx, conn); err != nil {
		e.reportUserDataStreamError(ctx, err)
		e.retryUserDataStream(ctx, conn, false)
	}
	return nil
}

// userDataStreamNotActiveCode is the code Binance.US answers an unsubscription of a subscription that has ended with
const userDataStreamNotActiveCode = -2036

// subscribeWantedUserDataStream subscribes conn to the configured account's User Data Stream, then ends the new
// subscription should a caller have unsubscribed the stream while the request was under way: the caller's
// unsubscription can reach Binance.US before the subscription does and leave it running. A subscription the caller's
// unsubscription already ended is answered as not active, which is the outcome wanted
func (e *Exchange) subscribeWantedUserDataStream(ctx context.Context, conn websocket.Connection) error {
	resp, err := e.subscribeUserDataStream(ctx, conn)
	if err != nil || e.userData.isWanted() {
		return err
	}
	req := &WebsocketAPIRequest{
		ID:     e.MessageID(),
		Method: "userDataStream.unsubscribe",
		Params: map[string]string{"subscriptionId": strconv.FormatUint(resp.SubscriptionID, 10)},
	}
	err = e.sendOnWebsocketAPIConnection(ctx, conn, userDataStreamSubscriptionRate, req, nil)
	if apiErr := (*APIError)(nil); errors.As(err, &apiErr) && apiErr.Code == userDataStreamNotActiveCode {
		return nil
	}
	return err
}

// subscribeUserDataStream subscribes conn to the User Data Stream of the account that owns ctx's API key
func (e *Exchange) subscribeUserDataStream(ctx context.Context, conn websocket.Connection) (*WebsocketUserDataStreamSubscriptionResponse, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	params := make(map[string]string)
	if err := e.rateLimitAndSignWebsocketAPIParams(ctx, userDataStreamSubscriptionRate, params); err != nil {
		return nil, err
	}
	// A request cancelled before it is sent creates no subscription to wait for
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	req := &WebsocketAPIRequest{ID: e.MessageID(), Method: "userDataStream.subscribe.signature", Params: params}
	e.userData.expect(req.ID, creds)
	defer e.userData.returned(req.ID)
	var resp *WebsocketUserDataStreamSubscriptionResponse
	return resp, e.sendOnWebsocketAPIConnection(ctx, conn, wsConnectionMessageRate, req, &resp)
}

// retryUserDataStream subscribes the configured account's User Data Stream on conn in the background, at once or after
// the first retry delay, and again after a longer wait each time it fails, while conn is kept for requests, the stream
// is wanted with usable credentials and authenticated websocket use is on, until a subscription, or a request for one
// awaiting its response, is recorded, conn goes down, or the websocket shuts down. It must be called while the websocket's wait group counts a
// running routine, such as a connection's reader
func (e *Exchange) retryUserDataStream(ctx context.Context, conn websocket.Connection, immediately bool) {
	r := &e.userDataRetry
	r.mu.Lock()
	if r.running == conn {
		r.mu.Unlock()
		return
	}
	r.running = conn
	delay := cmp.Or(r.delay, userDataRetryDelay)
	r.mu.Unlock()
	var wait time.Duration
	if !immediately {
		wait = delay
	}
	shutdown := e.Websocket.ShutdownC
	e.Websocket.Wg.Go(func() {
		defer func() {
			r.mu.Lock()
			if r.running == conn {
				r.running = nil
			}
			r.mu.Unlock()
		}()
		// An attempt under way when the websocket shuts down is cancelled rather than holding the shutdown up
		retryCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			select {
			case <-shutdown:
				cancel()
			case <-retryCtx.Done():
			}
		}()
		timer := time.NewTimer(wait)
		defer timer.Stop()
		for {
			select {
			case <-retryCtx.Done():
				return
			case <-timer.C:
			}
			if !conn.IsConnected() || e.wsAPI.get() != conn || !e.userData.isWanted() || !e.Websocket.CanUseAuthenticatedEndpoints() || !e.AreCredentialsValid(retryCtx) || e.userData.has(e.GetDefaultCredentials()) {
				return
			}
			err := e.subscribeWantedUserDataStream(retryCtx, conn)
			if err == nil {
				return
			}
			e.reportUserDataStreamError(retryCtx, err)
			wait = nextUserDataRetryWait(wait, delay)
			timer.Reset(wait)
		}
	})
}

// nextUserDataRetryWait returns the wait before the next attempt at a User Data Stream subscription after a failed one
// that followed a wait of wait: the first retry delay after an immediate attempt, and twice the last wait after that,
// up to userDataRetryMaxDelay
func nextUserDataRetryWait(wait, delay time.Duration) time.Duration {
	if wait == 0 {
		return delay
	}
	return min(wait*2, userDataRetryMaxDelay)
}

// reportUserDataStreamError relays a failed User Data Stream subscription to the data handler, or logs it when the data
// handler cannot take it
func (e *Exchange) reportUserDataStreamError(ctx context.Context, err error) {
	err = fmt.Errorf("%s user data stream unavailable: %w", e.Name, err)
	if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
		log.Errorf(log.WebsocketMgr, "%s: %s", errSend, err)
	}
}

// isConfiguredAccount reports whether creds are the configured credentials, whose account's stream the WebSocket API
// connection subscribes when it connects
func (e *Exchange) isConfiguredAccount(creds *accounts.Credentials) bool {
	configured := e.GetDefaultCredentials()
	return creds != nil && configured != nil && *creds == *configured
}

// SubscribeUserDataStream subscribes the WebSocket API connection to the User Data Stream of the account that owns the
// API key: the configured account's, or that of credentials ctx carries, whose events are then processed as that
// account's. Binance.US holds one subscription per account on a connection, until the connection closes. Subscribing
// the configured account's stream has it subscribed on each new connection again, should a caller have unsubscribed it
func (w WebsocketAPI) SubscribeUserDataStream(ctx context.Context) (*WebsocketUserDataStreamSubscriptionResponse, error) {
	conn, release, err := w.e.acquireWebsocketAPIConnection()
	if err != nil {
		return nil, err
	}
	defer release()
	creds, err := w.e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if w.e.isConfiguredAccount(creds) {
		w.e.userData.setWanted(true)
	}
	return w.e.subscribeUserDataStream(ctx, conn)
}

// UnsubscribeUserDataStream ends one of the WebSocket API connection's User Data Stream subscriptions. Ending the
// configured account's leaves its stream unsubscribed on new connections too, until SubscribeUserDataStream subscribes
// it again
func (w WebsocketAPI) UnsubscribeUserDataStream(ctx context.Context, subscriptionID uint64) error {
	// Binance.US sends eventStreamTerminated as it ends the subscription, perhaps before its response, so the stream is
	// marked as unwanted first, or the event would have it subscribed again
	if creds, ok := w.e.userData.credentials(subscriptionID); ok && w.e.isConfiguredAccount(creds) {
		w.e.userData.setWanted(false)
	}
	params := url.Values{}
	params.Set("subscriptionId", strconv.FormatUint(subscriptionID, 10))
	return w.e.SendWebsocketAPIRequest(ctx, userDataStreamSubscriptionRate, "userDataStream.unsubscribe", params, nil)
}

// UnsubscribeAllUserDataStreams ends every User Data Stream subscription on the WebSocket API connection, leaving the
// configured account's stream unsubscribed on new connections too, until SubscribeUserDataStream subscribes it again
func (w WebsocketAPI) UnsubscribeAllUserDataStreams(ctx context.Context) error {
	w.e.userData.setWanted(false)
	return w.e.SendWebsocketAPIRequest(ctx, userDataStreamSubscriptionRate, "userDataStream.unsubscribe", nil, nil)
}

// GetUserDataStreamSubscriptions returns the WebSocket API connection's active User Data Stream subscriptions
func (w WebsocketAPI) GetUserDataStreamSubscriptions(ctx context.Context) ([]WebsocketUserDataStreamSubscriptionResponse, error) {
	var resp []WebsocketUserDataStreamSubscriptionResponse
	return resp, w.e.SendWebsocketAPIRequest(ctx, userDataStreamSubscriptionRate, "session.subscriptions", nil, &resp)
}

// CreateUserDataStream starts a user data stream as Exchange.CreateUserDataStream does, over the WebSocket API; its
// events arrive on the stream server's connection to the listen key, not on the WebSocket API connection
//
// Deprecated: Binance.US is retiring listen keys, targeting Q4 2026 for a date it has yet to confirm. Use
// SubscribeUserDataStream instead
func (w WebsocketAPI) CreateUserDataStream(ctx context.Context) (*ListenKeyResponse, error) {
	return w.e.createUserDataStream(ctx, true)
}

// ExtendUserDataStream keeps a user data stream alive as Exchange.ExtendUserDataStream does, over the WebSocket API
//
// Deprecated: Binance.US is retiring listen keys, targeting Q4 2026 for a date it has yet to confirm. Use
// SubscribeUserDataStream instead
func (w WebsocketAPI) ExtendUserDataStream(ctx context.Context, listenKey string) error {
	return w.e.extendUserDataStream(ctx, listenKey, true)
}

// CloseUserDataStream closes a user data stream as Exchange.CloseUserDataStream does, over the WebSocket API
//
// Deprecated: Binance.US is retiring listen keys, targeting Q4 2026 for a date it has yet to confirm. Use
// SubscribeUserDataStream instead
func (w WebsocketAPI) CloseUserDataStream(ctx context.Context, listenKey string) error {
	return w.e.closeUserDataStream(ctx, listenKey, true)
}

// handleUserDataEvent processes a User Data Stream event as the event of its subscription's account
func (e *Exchange) handleUserDataEvent(ctx context.Context, conn websocket.Connection, respRaw, event []byte, eventType string) error {
	subscriptionID, err := parseSubscriptionID(respRaw, "subscriptionId")
	if err != nil {
		return fmt.Errorf("%w: %s", errUserDataEventWithoutSubscription, respRaw)
	}
	creds, ok := e.userData.credentials(subscriptionID)
	if !ok {
		return fmt.Errorf("%w %d: %s", errUnknownUserDataStreamSubscription, subscriptionID, respRaw)
	}
	ctx = accounts.DeployCredentialsToContext(ctx, creds)
	switch eventType {
	case "outboundAccountPosition":
		return e.processAccountPosition(ctx, creds, event)
	case "balanceUpdate":
		var resp WebsocketBalanceUpdate
		if err := json.Unmarshal(event, &resp); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &resp)
	case "executionReport":
		return e.processExecutionReport(ctx, event)
	case "listStatus":
		return e.processListStatus(ctx, event)
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
		if e.isConfiguredAccount(creds) && e.userData.isWanted() {
			log.Warnf(log.WebsocketMgr, "%s user data stream ended at %s, resubscribing", e.Name, resp.EventTime.Time())
			// The subscription request waits for its response, which this reader delivers, so it is made in the
			// background
			e.retryUserDataStream(ctx, conn, true)
		}
		return e.Websocket.DataHandler.Send(ctx, &resp)
	default:
		return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
	}
}

// balanceOrdering orders the spot balances User Data Stream events store against those REST polls store, by the account
// update time both carry, for the events and polls of the configured account, as each begins or is stored
type balanceOrdering struct {
	// storeMu keeps a balance event from being stored while a REST snapshot is accepted, so a snapshot is judged against
	// every event stored before it and no event is stored between its checks and its store
	storeMu sync.Mutex
	mu      sync.Mutex
	// byAccount holds the times of each account ordered, so that each account is ordered by its own updates alone,
	// whichever account SetCredentials or a Setup configures meanwhile
	byAccount map[accounts.Credentials]*balanceTimes
}

// balanceTimes holds the account update times an account's balances are ordered by
type balanceTimes struct {
	// eventTimes holds, by currency, the account update time of the latest event to report the currency's balance,
	// which a REST snapshot must not predate
	eventTimes map[*currency.Item]time.Time
	// snapshotTime is the account update time of the latest REST snapshot stored, which an event must not predate
	snapshotTime time.Time
}

// timesOf returns the times recorded for account, creating an empty set on first use. Callers sharing b hold b.mu while
// they look up and use the times
func (b *balanceOrdering) timesOf(account *accounts.Credentials) *balanceTimes {
	var key accounts.Credentials
	if account != nil {
		key = *account
	}
	if b.byAccount == nil {
		b.byAccount = make(map[accounts.Credentials]*balanceTimes)
	}
	times, ok := b.byAccount[key]
	if !ok {
		times = new(balanceTimes)
		b.byAccount[key] = times
	}
	return times
}

// withBalanceStore runs f holding the balance store lock, which events hold to store their balances and polls hold to
// accept a REST snapshot; REST requests are made outside it
func (e *Exchange) withBalanceStore(f func() error) error {
	e.balances.storeMu.Lock()
	defer e.balances.storeMu.Unlock()
	return f()
}

// recordBalanceEvent records the account update time of an event of account that reported the balances of codes
func (e *Exchange) recordBalanceEvent(account *accounts.Credentials, updated time.Time, codes ...currency.Code) {
	b := &e.balances
	b.mu.Lock()
	defer b.mu.Unlock()
	times := b.timesOf(account)
	if times.eventTimes == nil {
		times.eventTimes = make(map[*currency.Item]time.Time)
	}
	for _, c := range codes {
		if updated.After(times.eventTimes[c.Item]) {
			times.eventTimes[c.Item] = updated
		}
	}
}

// recordBalanceSnapshot records the account update time of a REST snapshot stored for account
func (e *Exchange) recordBalanceSnapshot(account *accounts.Credentials, asOf time.Time) {
	b := &e.balances
	b.mu.Lock()
	defer b.mu.Unlock()
	if times := b.timesOf(account); asOf.After(times.snapshotTime) {
		times.snapshotTime = asOf
	}
}

// balanceSnapshotTime returns the account update time of the latest REST snapshot stored for account
func (e *Exchange) balanceSnapshotTime(account *accounts.Credentials) time.Time {
	b := &e.balances
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.timesOf(account).snapshotTime
}

// balanceEventTimes returns the account update time of account's latest event to report each currency
func (e *Exchange) balanceEventTimes(account *accounts.Credentials) map[*currency.Item]time.Time {
	b := &e.balances
	b.mu.Lock()
	defer b.mu.Unlock()
	return maps.Clone(b.timesOf(account).eventTimes)
}

// keepStreamedSpotBalances puts the stored balance in place of each currency of a REST snapshot of account that its User
// Data Stream reported at or after asOf, the account update time the snapshot shows: the snapshot predates that event,
// or shares its millisecond and may predate it, while the stream's later events in that millisecond carry the latest
// balance. A currency the snapshot leaves out keeps its stored balance in the same case rather than being removed. A
// snapshot without a time is left as it is. The caller holds the balance store lock until the snapshot is stored
func (e *Exchange) keepStreamedSpotBalances(account *accounts.Credentials, subAccount *accounts.SubAccount, asOf time.Time) {
	if asOf.IsZero() {
		return
	}
	reported := make(map[*currency.Item]currency.Code, len(subAccount.Balances))
	for c := range subAccount.Balances {
		reported[c.Item] = c
	}
	for item, eventTime := range e.balanceEventTimes(account) {
		if eventTime.Before(asOf) {
			continue
		}
		c, ok := reported[item]
		if !ok {
			c = item.Currency()
		}
		if stored, err := e.Accounts.GetBalance("", account, asset.Spot, c); err == nil {
			subAccount.Balances.Set(c, stored)
		}
	}
}

// processAccountPosition stores the balances of an outboundAccountPosition event of creds' account and relays them. An
// event of the configured account, as it arrives or is stored, is ordered against that account's REST snapshots by
// account update time: one predating the stored snapshot is left out, as the snapshot holds its balances and maybe later
// ones
func (e *Exchange) processAccountPosition(ctx context.Context, creds *accounts.Credentials, event []byte) error {
	var resp WebsocketAccountPosition
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	updated := resp.LastAccountUpdateTime.Time()
	ordered := e.isConfiguredAccount(creds)
	subAccount := accounts.NewSubAccount(asset.Spot, "")
	if err := e.withBalanceStore(func() error {
		ordered = ordered || e.isConfiguredAccount(creds)
		if ordered {
			if updated.Before(e.balanceSnapshotTime(creds)) {
				return nil
			}
			codes := make([]currency.Code, len(resp.Balances))
			for i := range resp.Balances {
				codes[i] = resp.Balances[i].Asset
			}
			e.recordBalanceEvent(creds, updated, codes...)
		}
		for i := range resp.Balances {
			free := resp.Balances[i].Free.Float64()
			locked := resp.Balances[i].Locked.Float64()
			balance, err := e.Accounts.UpdateBalance(ctx, "", asset.Spot, resp.Balances[i].Asset, func(b *accounts.Balance) {
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

// processExecutionReport relays an executionReport event as an order update
func (e *Exchange) processExecutionReport(ctx context.Context, event []byte) error {
	var resp WebsocketExecutionReport
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(resp.Symbol, asset.Spot, false)
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
	side, err := order.StringToOrderSide(string(resp.Side))
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: orderID, Err: err})
	}
	tif, err := order.StringToTimeInForce(string(resp.TimeInForce))
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
		remaining = remainingAmount(resp.OrderQuantity, resp.CumulativeFilledQuantity, resp.PreventedQuantity, status)
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
		AssetType:            asset.Spot,
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
// OCO order list by its order list ID, tagged as an order list's, which only this event updates, since the legs'
// executionReport events carry their own order IDs
func (e *Exchange) processListStatus(ctx context.Context, event []byte) error {
	var resp WebsocketListStatus
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, &resp); err != nil {
		return err
	}
	if resp.ContingencyType != oneCancelsTheOtherContingencyType {
		return nil
	}
	pair, err := e.MatchSymbolWithAvailablePairs(resp.Symbol, asset.Spot, false)
	if err != nil {
		return err
	}
	detail, err := e.orderListDetail(resp.OrderListID, resp.ListClientOrderID, resp.ListOrderStatus, resp.TransactionTime.Time(), pair)
	if err != nil {
		return e.Websocket.DataHandler.Send(ctx, order.ClassificationError{Exchange: e.Name, OrderID: detail.OrderID, Err: err})
	}
	return e.Websocket.DataHandler.Send(ctx, &detail)
}
