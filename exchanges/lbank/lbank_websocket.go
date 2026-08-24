package lbank

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	lbankWsSubscribe   = "subscribe"
	lbankWsUnsubscribe = "unsubscribe"
	lbankWsTicker      = "tick"
	lbankWsTrades      = "trade"
	lbankWsOrderbook   = "depth"
	lbankWsKbar        = "kbar"
	lbankWsOrderUpdate = "orderUpdate"
	lbankWsAssetUpdate = "assetUpdate"
	lbankWsPing        = "ping"
	lbankWsPong        = "pong"
	lbankWsAction      = "action"
	lbankWsKline1Min   = "1min"
	lbankWsKline5Min   = "5min"
	lbankWsKline15Min  = "15min"
	lbankWsKline30Min  = "30min"
	lbankWsKline1Hr    = "1hr"
	lbankWsKline4Hr    = "4hr"
	lbankWsKlineDay    = "day"
	lbankWsKlineWeek   = "week"
	lbankWsKlineMonth  = "month"
)

var klineIntervals = map[kline.Interval]string{
	kline.OneMin:     lbankWsKline1Min,
	kline.FiveMin:    lbankWsKline5Min,
	kline.FifteenMin: lbankWsKline15Min,
	kline.ThirtyMin:  lbankWsKline30Min,
	kline.OneHour:    lbankWsKline1Hr,
	kline.FourHour:   lbankWsKline4Hr,
	kline.OneDay:     lbankWsKlineDay,
	kline.OneWeek:    lbankWsKlineWeek,
	kline.OneMonth:   lbankWsKlineMonth,
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 100},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    lbankWsTicker,
	subscription.AllTradesChannel: lbankWsTrades,
	subscription.OrderbookChannel: lbankWsOrderbook,
	subscription.CandlesChannel:   lbankWsKbar,
	subscription.MyOrdersChannel:  lbankWsOrderUpdate,
	subscription.MyAccountChannel: lbankWsAssetUpdate,
}

var defaultSubscriptionTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"channelName": func(s *subscription.Subscription) string {
		return subscriptionNames[s.Channel]
	},
}).Parse(`
{{- if $.S.Asset -}}
{{ range $asset, $pairs := $.AssetPairs }}
{{- range $p := $pairs -}}
{{- channelName $.S }}_{{ $p.Lower.String }}
{{- $.PairSeparator }}
{{- end -}}
{{ $.AssetSeparator }}
{{- end -}}
{{- else -}}
{{- channelName $.S }}
{{- end }}
`))

// WsConnect connects to the LBank websocket
func (e *Exchange) WsConnect() error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	ctx := context.TODO()
	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(ctx, &dialer, http.Header{}, nil)
	if err != nil {
		return err
	}
	if e.IsWebsocketAuthenticationSupported() {
		if e.privateKey == nil {
			if err := e.loadPrivKey(ctx); err != nil {
				e.Websocket.SetCanUseAuthenticatedEndpoints(false)
				log.Errorf(log.ExchangeSys, "%s failed to load private key for websocket auth: %v\n", e.Name, err)
			}
		}
		if e.privateKey != nil {
			key, err := e.GetWebsocketSubscribeKey(ctx)
			if err != nil {
				e.Websocket.SetCanUseAuthenticatedEndpoints(false)
				log.Errorf(log.ExchangeSys, "%s websocket auth failed: %v\n", e.Name, err)
			} else {
				e.ws.mu.Lock()
				e.ws.subscribeKey = key
				e.ws.mu.Unlock()
				e.Websocket.SetCanUseAuthenticatedEndpoints(true)
				e.Websocket.Wg.Add(1)
				go e.wsRefreshSubscribeKey(ctx)
			}
		}
	}
	if e.Verbose {
		log.Debugf(log.ExchangeSys, "%s Connected to Websocket.\n", e.Name)
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)
	return nil
}

// wsReadData receives and passes on websocket messages for processing
func (e *Exchange) wsReadData(ctx context.Context) {
	defer e.Websocket.Wg.Done()
	for {
		resp := e.Websocket.Conn.ReadMessage()
		if resp.Raw == nil {
			return
		}
		if err := e.wsHandleData(ctx, resp.Raw); err != nil {
			if errSend := e.Websocket.DataHandler.Send(ctx, err); errSend != nil {
				log.Errorf(log.WebsocketMgr, "%s %s: %s %s", e.Name, e.Websocket.Conn.GetURL(), errSend, err)
			}
		}
	}
}

// wsHandleData handles incoming websocket messages
func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	var base websocketResponse
	if err := json.Unmarshal(respRaw, &base); err != nil {
		return err
	}

	// Handle ping challenge before type check
	var ping websocketPingResponse
	if err := json.Unmarshal(respRaw, &ping); err == nil && ping.Action == lbankWsPing {
		return e.Websocket.Conn.SendJSONMessage(ctx, 0, map[string]string{
			lbankWsAction: lbankWsPong,
			"pong":        ping.Ping,
		})
	}

	if base.Type == "" {
		if base.Message != "" {
			return fmt.Errorf("lbank websocket error: %s", base.Message)
		}
		return nil
	}
	switch base.Type {
	case lbankWsTicker:
		return e.wsHandleTicker(ctx, respRaw)
	case lbankWsTrades:
		return e.wsHandleTrades(ctx, respRaw)
	case lbankWsOrderbook:
		return e.wsHandleOrderbook(respRaw)
	case lbankWsKbar:
		return e.wsHandleKbar(ctx, respRaw)
	case lbankWsOrderUpdate:
		return e.wsHandleOrderUpdate(ctx, respRaw)
	case lbankWsAssetUpdate:
		return e.wsHandleAssetUpdate(ctx, respRaw)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// wsHandleTicker handles ticker websocket messages
func (e *Exchange) wsHandleTicker(ctx context.Context, respRaw []byte) error {
	var resp websocketTickResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
		ExchangeName: e.Name,
		Pair:         resp.Pair, // ← direct use
		AssetType:    asset.Spot,
		High:         resp.Tick.High.Float64(),
		Low:          resp.Tick.Low.Float64(),
		Last:         resp.Tick.Latest.Float64(),
		Volume:       resp.Tick.Vol.Float64(),
	})
}

// wsHandleTrades handles trade websocket messages
func (e *Exchange) wsHandleTrades(ctx context.Context, respRaw []byte) error {
	tradeFeed := e.IsTradeFeedEnabled()
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !tradeFeed && !saveTradeData {
		return nil
	}

	var resp websocketTradeResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}

	side, err := order.StringToOrderSide(resp.Trade.Direction)
	if err != nil {
		return err
	}
	tradeData := trade.Data{
		Exchange:     e.Name,
		AssetType:    asset.Spot,
		CurrencyPair: resp.Pair,
		Price:        resp.Trade.Price.Float64(),
		Amount:       resp.Trade.Volume.Float64(),
		Timestamp:    resp.Trade.TS.Time(),
		Side:         side,
	}
	if tradeFeed {
		if err := e.Websocket.DataHandler.Send(ctx, tradeData); err != nil {
			return err
		}
	}
	if saveTradeData {
		return trade.AddTradesToBuffer(tradeData)
	}
	return nil
}

// wsHandleOrderbook handles orderbook websocket messages
func (e *Exchange) wsHandleOrderbook(respRaw []byte) error {
	var resp websocketDepthResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}

	return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:          e.Name,
		Pair:              resp.Pair,
		Asset:             asset.Spot,
		ValidateOrderbook: e.ValidateOrderbook,
		Asks:              resp.Depth.Asks.Levels(),
		Bids:              resp.Depth.Bids.Levels(),
		LastUpdated:       time.Now(),
	})
}

// wsHandleKbar handles kline websocket messages
func (e *Exchange) wsHandleKbar(ctx context.Context, respRaw []byte) error {
	var resp websocketKbarResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}

	interval, err := klineIntervalFromString(resp.Kbar.Slot)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     resp.Pair,
		Asset:    asset.Spot,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   resp.Kbar.Timestamp,
			Open:   resp.Kbar.Open.Float64(),
			High:   resp.Kbar.High.Float64(),
			Low:    resp.Kbar.Low.Float64(),
			Close:  resp.Kbar.Close.Float64(),
			Volume: resp.Kbar.Volume.Float64(),
		}},
	})
}

// lbankOrderStatusToOrderStatus converts LBank integer status to order.Status
func lbankOrderStatusToOrderStatus(status int64) (order.Status, error) {
	switch status {
	case -1:
		return order.Cancelled, nil
	case 0:
		return order.New, nil
	case 1:
		return order.PartiallyFilled, nil
	case 2:
		return order.Filled, nil
	case 3:
		return order.PartiallyCancelled, nil
	case 4:
		return order.PendingCancel, nil
	default:
		return order.UnknownStatus, fmt.Errorf("lbank: unknown order status %d", status)
	}
}

// wsHandleOrderUpdate handles order update websocket messages
func (e *Exchange) wsHandleOrderUpdate(ctx context.Context, respRaw []byte) error {
	var resp websocketOrderUpdateResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}

	status, err := lbankOrderStatusToOrderStatus(resp.OrderUpdate.OrderStatus)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
		Exchange:    e.Name,
		AssetType:   asset.Spot,
		Pair:        resp.Pair,
		Price:       resp.OrderUpdate.Price.Float64(),
		Amount:      resp.OrderUpdate.Amount.Float64(),
		OrderID:     resp.OrderUpdate.UUID,
		Status:      status,
		LastUpdated: resp.OrderUpdate.UpdateTime.Time(),
	})
}

// wsHandleAssetUpdate handles asset update websocket messages
func (e *Exchange) wsHandleAssetUpdate(ctx context.Context, respRaw []byte) error {
	var resp websocketAssetUpdateResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, accounts.Change{
		AssetType: asset.Spot,
		Balance: accounts.Balance{
			Currency: currency.NewCode(resp.Data.AssetCode),
			Total:    resp.Data.Asset.Float64(),
			Free:     resp.Data.Free.Float64(),
			Hold:     resp.Data.Freeze.Float64(),
		},
	})
}

// klineIntervalFromString converts an LBank interval string to a kline.Interval
func klineIntervalFromString(s string) (kline.Interval, error) {
	switch s {
	case lbankWsKline1Min:
		return kline.OneMin, nil
	case lbankWsKline5Min:
		return kline.FiveMin, nil
	case lbankWsKline15Min:
		return kline.FifteenMin, nil
	case lbankWsKline30Min:
		return kline.ThirtyMin, nil
	case lbankWsKline1Hr:
		return kline.OneHour, nil
	case lbankWsKline4Hr:
		return kline.FourHour, nil
	case lbankWsKlineDay:
		return kline.OneDay, nil
	case lbankWsKlineWeek:
		return kline.OneWeek, nil
	case lbankWsKlineMonth:
		return kline.OneMonth, nil
	default:
		return 0, fmt.Errorf("lbank: unsupported kline interval string %s", s)
	}
}

// generateSubscriptions generates default subscriptions
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns the subscription template for LBank
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return defaultSubscriptionTemplate, nil
}

// wsRefreshSubscribeKey refreshes the subscribe key every 50 minutes
func (e *Exchange) wsRefreshSubscribeKey(ctx context.Context) {
	defer e.Websocket.Wg.Done()
	refreshTicker := time.NewTicker(50 * time.Minute)
	defer refreshTicker.Stop()
	for {
		select {
		case <-refreshTicker.C:
			e.ws.mu.RLock()
			key := e.ws.subscribeKey
			e.ws.mu.RUnlock()

			if err := e.RefreshWebsocketSubscribeKey(ctx, key); err != nil {
				log.Warnf(log.ExchangeSys, "%s failed to refresh websocket subscribe key, attempting to get new one: %v\n", e.Name, err)
				newKey, err := e.GetWebsocketSubscribeKey(ctx)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s failed to get new websocket subscribe key: %v\n", e.Name, err)
					continue
				}
				e.ws.mu.Lock()
				e.ws.subscribeKey = newKey
				e.ws.mu.Unlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

// manageSubs handles both subscribe and unsubscribe
func (e *Exchange) manageSubs(ctx context.Context, subs subscription.List, action string) error {
	var errs error
subscriptionLoop:
	for _, s := range subs {
		chName, ok := subscriptionNames[s.Channel]
		if !ok {
			errs = common.AppendError(errs, fmt.Errorf("lbank: unsupported channel %s", s.Channel))
			continue
		}

		e.ws.mu.RLock()
		subscribeKey := e.ws.subscribeKey
		e.ws.mu.RUnlock()

		var req map[string]any
		switch s.Channel {
		case subscription.MyOrdersChannel:
			req = map[string]any{
				lbankWsAction:  action,
				"subscribe":    lbankWsOrderUpdate,
				"subscribeKey": subscribeKey,
				"pair":         "all",
			}
		case subscription.MyAccountChannel:
			req = map[string]any{
				lbankWsAction:  action,
				"subscribe":    lbankWsAssetUpdate,
				"subscribeKey": subscribeKey,
			}
		}

		if req != nil {
			if err := e.Websocket.Conn.SendJSONMessage(ctx, 0, req); err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			if action == lbankWsSubscribe {
				errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s))
			} else {
				errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s))
			}
			continue
		}

		for _, p := range s.Pairs {
			var req map[string]any
			switch s.Channel {
			case subscription.OrderbookChannel:
				req = map[string]any{
					lbankWsAction: action,
					"subscribe":   chName,
					"depth":       strconv.Itoa(s.Levels),
					"pair":        p.Lower().String(),
				}
			case subscription.CandlesChannel:
				intervalStr, ok := klineIntervals[s.Interval]
				if !ok {
					errs = common.AppendError(errs, fmt.Errorf("lbank: unsupported kline interval %v", s.Interval))
					continue subscriptionLoop
				}
				req = map[string]any{
					lbankWsAction: action,
					"subscribe":   chName,
					"kbar":        intervalStr,
					"pair":        p.Lower().String(),
				}
			default:
				req = map[string]any{
					lbankWsAction: action,
					"subscribe":   chName,
					"pair":        p.Lower().String(),
				}
			}
			if err := e.Websocket.Conn.SendJSONMessage(ctx, 0, req); err != nil {
				errs = common.AppendError(errs, err)
				continue subscriptionLoop
			}
		}
		if action == lbankWsSubscribe {
			errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s))
		} else {
			errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s))
		}
	}
	return errs
}

// Subscribe subscribes to a list of websocket channels
func (e *Exchange) Subscribe(subs subscription.List) error {
	return e.manageSubs(context.TODO(), subs, lbankWsSubscribe)
}

// Unsubscribe unsubscribes from a list of websocket channels
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	return e.manageSubs(context.TODO(), subs, lbankWsUnsubscribe)
}
