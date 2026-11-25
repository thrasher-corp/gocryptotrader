package lbank

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// subscribeKeyLifetime is how long a subscribe key lasts from its creation or last renewal
	subscribeKeyLifetime = time.Hour
	// subscribeKeyRefreshInterval renews a subscribe key well within its lifetime
	subscribeKeyRefreshInterval = 30 * time.Minute
	// subscribeKeyRetryInterval spaces the attempts to renew or replace a subscribe key after one fails, as waiting a full
	// refresh interval would let a single passing outage expire the key
	subscribeKeyRetryInterval = time.Minute
	// subscribeKeyCloseTimeout bounds the request that closes a subscribe key once its connection ends
	subscribeKeyCloseTimeout = 5 * time.Second
)

// subscribeKeyTimings holds how often a subscribe key is renewed, how soon a failed renewal is retried and how long a key
// lasts without renewal
type subscribeKeyTimings struct {
	refresh  time.Duration
	retry    time.Duration
	lifetime time.Duration
}

var defaultSubscribeKeyTimings = subscribeKeyTimings{
	refresh:  subscribeKeyRefreshInterval,
	retry:    subscribeKeyRetryInterval,
	lifetime: subscribeKeyLifetime,
}

// nextRenewal returns how long to wait before renewing a subscribe key again: the refresh interval after a renewal, and
// the retry interval, if sooner, after a failure
func (t subscribeKeyTimings) nextRenewal(err error) time.Duration {
	if err != nil {
		return min(t.refresh, t.retry)
	}
	return t.refresh
}

// CreateSubscribeKey creates a subscribe key, which the account websocket channels take and which lasts an hour unless
// refreshed
func (e *Exchange) CreateSubscribeKey(ctx context.Context) (string, error) {
	var key string
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/subscribe/get_key.do", nil, &key); err != nil {
		return "", err
	}
	if key == "" {
		return "", common.ErrNoResponse
	}
	return key, nil
}

// RefreshSubscribeKey extends a subscribe key so it lasts an hour from now
func (e *Exchange) RefreshSubscribeKey(ctx context.Context, subscribeKey string) error {
	return e.subscribeKeyRequest(ctx, "/v2/subscribe/refresh_key.do", subscribeKey)
}

// CloseSubscribeKey closes a subscribe key, ending the account data its subscriptions stream
func (e *Exchange) CloseSubscribeKey(ctx context.Context, subscribeKey string) error {
	return e.subscribeKeyRequest(ctx, "/v2/subscribe/destroy_key.do", subscribeKey)
}

func (e *Exchange) subscribeKeyRequest(ctx context.Context, path, subscribeKey string) error {
	if subscribeKey == "" {
		return errSubscribeKeyRequired
	}
	params := url.Values{}
	params.Set("subscribeKey", subscribeKey)
	return e.SendAuthHTTPRequest(ctx, defaultEPL, path, params, nil)
}

// subscribeKeyStore holds the subscribe key of each account connection, and the key each account channel on it was
// last subscribed with
type subscribeKeyStore struct {
	mu   sync.Mutex
	keys map[websocket.Connection]string
	sent map[websocket.Connection]map[string]string
	// account serialises creating and replacing keys with the account subscriptions, so a connection gets one key and
	// renewer, and a replacement key subscribes again only the channels still subscribed
	account sync.Mutex
}

func (s *subscribeKeyStore) set(conn websocket.Connection, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys == nil {
		s.keys = make(map[websocket.Connection]string)
	}
	s.keys[conn] = key
}

func (s *subscribeKeyStore) get(conn websocket.Connection) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keys[conn]
}

// remove forgets a connection's subscribe key and the channels subscribed with it, unless a reconnect has given the
// connection another
func (s *subscribeKeyStore) remove(conn websocket.Connection, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys[conn] == key {
		delete(s.keys, conn)
		delete(s.sent, conn)
	}
}

// markSent records the key an account channel on a connection was subscribed with, while the connection still holds
// that key, so a subscription finishing as its connection shuts down records nothing
func (s *subscribeKeyStore) markSent(conn websocket.Connection, channel, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys[conn] != key {
		return
	}
	if s.sent == nil {
		s.sent = make(map[websocket.Connection]map[string]string)
	}
	if s.sent[conn] == nil {
		s.sent[conn] = make(map[string]string)
	}
	s.sent[conn][channel] = key
}

// forgetSent forgets an account channel a connection has unsubscribed from
func (s *subscribeKeyStore) forgetSent(conn websocket.Connection, channel string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sent[conn], channel)
}

// sentWith returns the key an account channel on a connection was last subscribed with, and whether it is subscribed
func (s *subscribeKeyStore) sentWith(conn websocket.Connection, channel string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.sent[conn][channel]
	return key, ok
}

// connectionGone reports whether a connection has been shut down; one that can't tell is taken as live
func connectionGone(conn websocket.Connection) bool {
	c, ok := conn.(interface{ IsConnected() bool })
	return ok && !c.IsConnected()
}

// startSubscribeKey creates a connection's subscribe key and starts the renewer that keeps it alive until the websocket
// shuts down or the connection ends. Its caller holds the account lock, and the manager's lock, which reading ShutdownC
// and adding to Wg need
func (e *Exchange) startSubscribeKey(ctx context.Context, conn websocket.Connection, timings subscribeKeyTimings) error {
	shutdown := e.Websocket.ShutdownC
	key, err := e.CreateSubscribeKey(ctx)
	if err != nil {
		return fmt.Errorf("creating subscribe key: %w", err)
	}
	if connectionGone(conn) {
		// The connection was shut down while its key was created, leaving no renewer to close the key
		e.closeSubscribeKey(ctx, key)
		return fmt.Errorf("creating subscribe key: %w", websocket.ErrNotConnected)
	}
	e.subscribeKeys.set(conn, key)
	e.Websocket.Wg.Add(1)
	// The renewer outlives the request that created the key, such as a flush, so it renews on a context of its own
	go e.keepSubscribeKeyAlive(context.WithoutCancel(ctx), conn, key, shutdown, timings)
	return nil
}

// subscribeKeyRenewal is the state of a connection's subscribe key renewer
type subscribeKeyRenewal struct {
	key     string
	renewed time.Time
}

// keepSubscribeKeyAlive renews a connection's subscribe key on a timer until the websocket shuts down or the connection
// ends, then closes it. A connection lost on its own is noticed at the next renewal, so its key is closed up to one
// interval later
func (e *Exchange) keepSubscribeKeyAlive(ctx context.Context, conn websocket.Connection, key string, shutdown <-chan struct{}, timings subscribeKeyTimings) {
	defer e.Websocket.Wg.Done()
	r := &subscribeKeyRenewal{key: key, renewed: time.Now()}
	defer func() {
		e.subscribeKeys.remove(conn, r.key)
		e.closeSubscribeKey(ctx, r.key)
	}()
	timer := time.NewTimer(timings.refresh)
	defer timer.Stop()
	for {
		select {
		case <-shutdown:
			return
		case <-timer.C:
			// The manager can close one connection without shutting the websocket down, such as a connection it rolls
			// back, so a key whose connection has gone is closed rather than renewed
			if connectionGone(conn) {
				return
			}
			err := e.renewSubscribeKey(ctx, conn, r, timings.lifetime)
			if err != nil {
				if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
					log.Errorf(log.WebsocketMgr, "%s: %v: %v", e.Name, errSend, err)
				}
			}
			timer.Reset(timings.nextRenewal(err))
		}
	}
}

// renewSubscribeKey refreshes a connection's subscribe key, or replaces it once its refreshes have failed for as long as
// a key lasts, then subscribes with the current key each account channel the connection carries under an earlier one
func (e *Exchange) renewSubscribeKey(ctx context.Context, conn websocket.Connection, r *subscribeKeyRenewal, lifetime time.Duration) error {
	var errs error
	var replacement string
	if time.Since(r.renewed) < lifetime {
		if err := e.RefreshSubscribeKey(ctx, r.key); err != nil {
			errs = fmt.Errorf("refreshing subscribe key: %w", err)
		} else {
			r.renewed = time.Now()
		}
	} else {
		var err error
		if replacement, err = e.CreateSubscribeKey(ctx); err != nil {
			return fmt.Errorf("replacing expired subscribe key: %w", err)
		}
	}
	// Only installing a replacement and subscribing with it wait for the account subscriptions, so a slow request holds
	// up none of them
	e.subscribeKeys.account.Lock()
	defer e.subscribeKeys.account.Unlock()
	if replacement != "" {
		e.subscribeKeys.set(conn, replacement)
		r.key, r.renewed = replacement, time.Now()
	}
	return common.AppendError(errs, e.resubscribeAccountChannels(ctx, conn))
}

// subscribedAccountSubscriptions returns the account subscriptions the websocket holds as subscribed
func (e *Exchange) subscribedAccountSubscriptions() subscription.List {
	return slices.DeleteFunc(e.Websocket.GetSubscriptions(), func(s *subscription.Subscription) bool {
		return !isAccountSubscription(s) || s.State() != subscription.SubscribedState
	})
}

// resubscribeAccountChannels subscribes with the connection's current subscribe key each account channel still
// subscribed that the connection carries under an earlier key. Its caller holds the account lock, so no subscription
// change can overtake it
func (e *Exchange) resubscribeAccountChannels(ctx context.Context, conn websocket.Connection) error {
	var errs error
	for _, s := range e.subscribedAccountSubscriptions() {
		if key, ok := e.subscribeKeys.sentWith(conn, subscriptionNames[s.Channel]); !ok || key == e.subscribeKeys.get(conn) {
			continue
		}
		req, err := e.subscriptionRequest(conn, "subscribe", s)
		if err == nil {
			err = conn.SendJSONMessage(ctx, request.Unset, req)
		}
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("resubscribing %s with a new subscribe key: %w", s.QualifiedChannel, err))
			continue
		}
		e.subscribeKeys.markSent(conn, req.Subscribe, req.SubscribeKey)
	}
	return errs
}

// closeSubscribeKey closes a subscribe key on a detached context, since the connection's is usually done by then
func (e *Exchange) closeSubscribeKey(ctx context.Context, key string) {
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), subscribeKeyCloseTimeout)
	defer cancel()
	if err := e.CloseSubscribeKey(closeCtx, key); err != nil {
		log.Warnf(log.WebsocketMgr, "%s: closing subscribe key: %v", e.Name, err)
	}
}

// wsProcessOrderUpdate relays an order's update, and the trade it reports when the order traded
func (e *Exchange) wsProcessOrderUpdate(ctx context.Context, msg *websocketMessage) error {
	u := msg.OrderUpdate
	if u == nil {
		return fmt.Errorf("%w: order update", common.ErrNoResponse)
	}
	pair, err := e.matchSpotPair(u.Symbol)
	if err != nil {
		return err
	}
	side, orderType, tif, err := parseOrderType(u.Type)
	if err != nil {
		return err
	}
	if !strings.Contains(u.Type, "_") {
		// LBank documents the pushed type as the order's side, yet pushes a market buy as buy_market, as its order queries
		// type it, so only a suffixed type sets the order's type; a bare side leaves the stored order's type alone
		orderType, tif = order.UnknownType, order.UnknownTIF
	}
	status, err := orderStatus(u.OrderStatus)
	if err != nil {
		return err
	}
	detail := &order.Detail{
		Exchange:             e.Name,
		OrderID:              u.OrderID,
		ClientOrderID:        u.CustomerID,
		Pair:                 pair,
		AssetType:            asset.Spot,
		Side:                 side,
		Type:                 orderType,
		TimeInForce:          tif,
		Status:               status,
		Price:                u.OrderPrice.Float64(),
		ExecutedAmount:       u.AccumulatedAmount.Float64(),
		AverageExecutedPrice: u.AveragePrice.Float64(),
		Cost:                 u.AveragePrice.Float64() * u.AccumulatedAmount.Float64(),
		LastUpdated:          u.UpdateTime.Time(),
	}
	switch {
	case side == order.Buy && orderType == order.Market:
		// A market buy orders the quote amount it spends, leaving its base amount, and so its remaining amount, unknown
		detail.QuoteAmount = u.OrderAmount.Float64()
	case side == order.Sell:
		detail.Amount = u.OrderAmount.Float64()
		detail.RemainingAmount = u.RemainAmount.Float64()
	default:
		// LBank pushes a buy's remaining quote value, so its remaining amount is taken from its amount and fill
		detail.Amount = u.OrderAmount.Float64()
		detail.RemainingAmount = max(detail.Amount-detail.ExecutedAmount, 0)
	}
	if err := e.Websocket.DataHandler.Send(ctx, detail); err != nil {
		return err
	}
	if u.TransactionID == "" || u.Amount.Float64() == 0 {
		return nil
	}
	err = e.Websocket.Fills.Update(fill.Data{
		ID:            u.TransactionID,
		TradeID:       u.TransactionID,
		Exchange:      e.Name,
		AssetType:     asset.Spot,
		CurrencyPair:  pair,
		Side:          side,
		OrderID:       u.OrderID,
		ClientOrderID: u.CustomerID,
		Price:         u.Price.Float64(),
		Amount:        u.Amount.Float64(),
		Timestamp:     u.UpdateTime.Time(),
	})
	if errors.Is(err, fill.ErrFeedDisabled) {
		return nil
	}
	return err
}

// wsProcessAssetUpdate stores a changed spot balance and relays it
func (e *Exchange) wsProcessAssetUpdate(ctx context.Context, msg *websocketMessage) error {
	u := msg.AssetUpdate
	if u == nil {
		return fmt.Errorf("%w: asset update", common.ErrNoResponse)
	}
	balance, err := e.Accounts.UpdateBalance(ctx, "", asset.Spot, u.AssetCode, func(b *accounts.Balance) {
		b.Total = u.Total.Float64()
		b.Free = u.Free.Float64()
		b.Hold = u.Frozen.Float64()
	})
	if err != nil {
		return err
	}
	subAccount := accounts.NewSubAccount(asset.Spot, "")
	subAccount.Balances.Set(u.AssetCode, balance)
	return e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAccount})
}
