package lbank

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// subscribeKeyRefreshInterval renews a subscribe key well within the hour it lasts from its creation or last renewal
	subscribeKeyRefreshInterval = 30 * time.Minute
	// subscribeKeyRetryInterval spaces the attempts to renew a subscribe key after one fails, as waiting a full refresh
	// interval would let a single passing outage expire the key
	subscribeKeyRetryInterval = time.Minute
	// subscribeKeyCloseTimeout bounds the request that closes a subscribe key once its connection ends
	subscribeKeyCloseTimeout = 5 * time.Second
)

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

// subscribeKeyStore holds the subscribe key of each account connection
type subscribeKeyStore struct {
	mu   sync.Mutex
	keys map[websocket.Connection]string
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

// remove forgets a connection's subscribe key, unless a reconnect has given the connection another
func (s *subscribeKeyStore) remove(conn websocket.Connection, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keys[conn] == key {
		delete(s.keys, conn)
	}
}

// wsConnectAccount dials the account connection with a subscribe key of its own, which a renewer keeps alive for as
// long as the connection lives. A key that cannot be created fails only this connection, which the manager retries
// on the next connect or flush, leaving the market data connections up
func (e *Exchange) wsConnectAccount(ctx context.Context, conn websocket.Connection) error {
	key, err := e.CreateSubscribeKey(ctx)
	if err != nil {
		return fmt.Errorf("creating subscribe key: %w", err)
	}
	if err := e.wsConnect(ctx, conn); err != nil {
		// No renewer owns the key until the connection is up, so it is closed here: the monitor retries a failed connect
		// every few seconds and would otherwise leave a key behind each time
		e.closeSubscribeKey(ctx, key)
		return err
	}
	e.subscribeKeys.set(conn, key)
	e.Websocket.Wg.Add(1)
	go e.keepSubscribeKeyAlive(ctx, conn, key, subscribeKeyRefreshInterval)
	return nil
}

// keepSubscribeKeyAlive refreshes a connection's subscribe key on a timer until the connection ends, then closes it.
// A connection lost on its own is noticed at the next refresh, so its key is closed up to one interval later
func (e *Exchange) keepSubscribeKeyAlive(ctx context.Context, conn websocket.Connection, key string, interval time.Duration) {
	defer e.Websocket.Wg.Done()
	defer e.subscribeKeys.remove(conn, key)
	defer e.closeSubscribeKey(ctx, key)
	refresh := time.NewTicker(interval)
	defer refresh.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.Websocket.ShutdownC:
			return
		case <-refresh.C:
			// The manager can close one connection without closing ShutdownC, such as a connection it rolls back, so a
			// key whose connection has gone is closed rather than refreshed
			if c, ok := conn.(interface{ IsConnected() bool }); ok && !c.IsConnected() {
				return
			}
			err := e.RefreshSubscribeKey(ctx, key)
			if err != nil {
				if errSend := e.Websocket.DataHandler.Send(ctx, fmt.Errorf("refreshing subscribe key: %w", err)); errSend != nil {
					log.Errorf(log.WebsocketMgr, "%s: %v: %v", e.Name, errSend, err)
				}
			}
			refresh.Reset(refreshDelay(err, interval))
		}
	}
}

// refreshDelay returns how long to wait before renewing a subscribe key again: the refresh interval after a renewal,
// and sooner after a failure
func refreshDelay(err error, interval time.Duration) time.Duration {
	if err != nil {
		return min(interval, subscribeKeyRetryInterval)
	}
	return interval
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
	side, err := order.StringToOrderSide(u.Type)
	if err != nil {
		return err
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
		Status:               status,
		Price:                u.OrderPrice.Float64(),
		Amount:               u.OrderAmount.Float64(),
		ExecutedAmount:       u.AccumulatedAmount.Float64(),
		AverageExecutedPrice: u.AveragePrice.Float64(),
		LastUpdated:          u.UpdateTime.Time(),
	}
	// A buy order's remaining amount is a quote value, which the order's base remaining amount can't be taken from
	if side == order.Sell {
		detail.RemainingAmount = u.RemainAmount.Float64()
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
