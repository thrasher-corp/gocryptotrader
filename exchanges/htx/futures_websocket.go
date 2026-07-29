package htx

import (
	"context"
	"fmt"
	"net/http"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	wsFuturesURL      = "wss://api.hbdm.com/ws"
	wsCoinMarginedURL = "wss://api.hbdm.com/swap-ws"
	wsUSDTMarginedURL = "wss://api.hbdm.com/linear-swap-ws"
)

var defaultFuturesSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: wsFundingRateChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsFundingRateChannel},
}

// wsConnect establishes an HTX websocket connection for the asset selected by its connection setup.
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	return conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil)
}

// generateSubscriptionsForAsset isolates configured subscriptions for one websocket connection.
func (e *Exchange) generateSubscriptionsForAsset(a asset.Item, private bool) (subscription.List, error) {
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, sub := range e.Features.Subscriptions {
		if sub.Asset == a && sub.Authenticated == private {
			cloned := sub.Clone()
			if a == asset.Futures {
				pairs, err := e.GetEnabledPairs(a)
				if err != nil {
					return nil, err
				}
				e.futureContractCodesMutex.RLock()
				for _, pair := range pairs {
					quote := pair.Quote.String()
					for expiryCode, expiryDate := range e.futureContractCodes {
						if quote == expiryDate.String() {
							quote = expiryCode
							break
						}
					}
					cloned.Pairs = append(cloned.Pairs, currency.NewPairWithDelimiter(pair.Base.String(), quote, "_"))
				}
				e.futureContractCodesMutex.RUnlock()
			}
			subs = append(subs, cloned)
		}
	}
	return subs.ExpandTemplates(e)
}

// getWebsocketConnection resolves legacy subscription calls onto the asset-specific connection.
func (e *Exchange) getWebsocketConnection(sub *subscription.Subscription) (websocket.Connection, error) {
	if sub.Authenticated {
		if e.Websocket.AuthConn != nil {
			return e.Websocket.AuthConn, nil
		}
		return e.Websocket.GetConnection(exchange.WebsocketPrivate)
	}
	if e.Websocket.Conn != nil {
		return e.Websocket.Conn, nil
	}
	switch sub.Asset {
	case asset.Spot:
		return e.Websocket.GetConnection(exchange.WebsocketSpot)
	case asset.Futures:
		return e.Websocket.GetConnection(exchange.WebsocketFutures)
	case asset.CoinMarginedFutures:
		return e.Websocket.GetConnection(exchange.WebsocketCoinMargined)
	case asset.USDTMarginedFutures:
		return e.Websocket.GetConnection(exchange.WebsocketUSDTMargined)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, sub.Asset)
	}
}

// subscribeConnection subscribes through the connection selected by the websocket manager.
func (e *Exchange) subscribeConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, batch subscription.List) error {
		return e.manageSubs(ctx, conn, wsSubOp, batch)
	}, 1))
}

// unsubscribeConnection unsubscribes through the connection selected by the websocket manager.
func (e *Exchange) unsubscribeConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, batch subscription.List) error {
		return e.manageSubs(ctx, conn, wsUnsubOp, batch)
	}, 1))
}

// wsAuthenticateConnection authenticates the dedicated spot private connection.
func (e *Exchange) wsAuthenticateConnection(ctx context.Context, conn websocket.Connection) error {
	if err := e.wsLogin(ctx, conn); err != nil {
		return err
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return nil
}

// wsHandleFundingRateMsg forwards derivative funding-rate updates to the websocket data handler.
func (e *Exchange) wsHandleFundingRateMsg(ctx context.Context, s *subscription.Subscription, raw []byte) error {
	if s.Asset != asset.CoinMarginedFutures && s.Asset != asset.USDTMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, s.Asset)
	}
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var response *WsFundingRate
	if err := json.Unmarshal(raw, &response); err != nil {
		return err
	}
	response.Asset = s.Asset
	response.Pair = s.Pairs[0]
	return e.Websocket.DataHandler.Send(ctx, response)
}
