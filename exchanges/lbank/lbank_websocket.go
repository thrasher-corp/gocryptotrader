package lbank

import (
	"context"
	"fmt"
	"net/http"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
)

var klineIntervals = map[kline.Interval]string{
	kline.OneMin:     "1min",
	kline.FiveMin:    "5min",
	kline.FifteenMin: "15min",
	kline.ThirtyMin:  "30min",
	kline.OneHour:    "1hr",
	kline.FourHour:   "4hr",
	kline.OneDay:     "day",
	kline.OneWeek:    "week",
	kline.OneMonth:   "month",
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
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
{{- range $asset, $pairs := $.AssetPairs -}}
	{{- range $p := $pairs -}}
		{{- channelName $.S }}_{{ $p.Lower.String }}
		{{- $.PairSeparator }}
	{{- end -}}
	{{- $.AssetSeparator }}
{{- end -}}
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
		key, err := e.GetWebsocketSubscribeKey(ctx)
		if err != nil {
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
			log.Errorf(log.ExchangeSys, "%s websocket auth failed: %v\n", e.Name, err)
		} else {
			e.wsSubscribeKey = key
			e.Websocket.SetCanUseAuthenticatedEndpoints(true)
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
		return e.wsHandleTrades(respRaw)
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
	p, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &ticker.Price{
		ExchangeName: e.Name,
		Pair:         p,
		AssetType:    asset.Spot,
		High:         resp.Tick.High.Float64(),
		Low:          resp.Tick.Low.Float64(),
		Last:         resp.Tick.Latest.Float64(),
		Volume:       resp.Tick.Vol.Float64(),
	})
}

// wsHandleTrades handles trade websocket messages
func (e *Exchange) wsHandleTrades(respRaw []byte) error {
	if !e.IsSaveTradeDataEnabled() && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp websocketTradeResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	p, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Trade.Direction)
	if err != nil {
		return err
	}
	return trade.AddTradesToBuffer(trade.Data{
		Exchange:     e.Name,
		AssetType:    asset.Spot,
		CurrencyPair: p,
		Price:        resp.Trade.Price.Float64(),
		Amount:       resp.Trade.Volume.Float64(),
		Timestamp:    resp.Trade.TS.Time(),
		Side:         side,
	})
}

// wsHandleOrderbook handles orderbook websocket messages
func (e *Exchange) wsHandleOrderbook(respRaw []byte) error {
	var resp websocketDepthResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	p, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
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
	p, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	interval, err := klineIntervalFromString(resp.Kbar.Slot)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Pair:     p,
		Asset:    asset.Spot,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   resp.Kbar.Timestamp.Time(),
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
	p, err := currency.NewPairFromString(resp.Pair)
	if err != nil {
		return err
	}
	status, err := lbankOrderStatusToOrderStatus(resp.OrderUpdate.OrderStatus)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &order.Detail{
		Exchange:    e.Name,
		AssetType:   asset.Spot,
		Pair:        p,
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
	return e.Websocket.DataHandler.Send(ctx, resp.Data)
}

// klineIntervalFromString converts an LBank interval string to a kline.Interval
func klineIntervalFromString(s string) (kline.Interval, error) {
	for interval, str := range klineIntervals {
		if str == s {
			return interval, nil
		}
	}
	return 0, fmt.Errorf("lbank: unsupported kline interval string %s", s)
}

// generateSubscriptions generates default subscriptions
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns the subscription template for LBank
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return defaultSubscriptionTemplate, nil
}

// manageSubs handles both subscribe and unsubscribe
func (e *Exchange) manageSubs(ctx context.Context, subs subscription.List, action string) error {
	var errs error
	for _, s := range subs {
		chName, ok := subscriptionNames[s.Channel]
		if !ok {
			errs = common.AppendError(errs, fmt.Errorf("lbank: unsupported channel %s", s.Channel))
			continue
		}

		// orderUpdate and assetUpdate subscribe once without pairs
		switch s.Channel {
		case subscription.MyOrdersChannel:
			req := map[string]any{
				"action":       action,
				"subscribe":    lbankWsOrderUpdate,
				"subscribeKey": e.wsSubscribeKey,
				"pair":         "all",
			}
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
		case subscription.MyAccountChannel:
			req := map[string]any{
				"action":       action,
				"subscribe":    lbankWsAssetUpdate,
				"subscribeKey": e.wsSubscribeKey,
			}
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
					"action":    action,
					"subscribe": chName,
					"depth":     "100",
					"pair":      p.Lower().String(),
				}
			case subscription.CandlesChannel:
				intervalStr, ok := klineIntervals[s.Interval]
				if !ok {
					errs = common.AppendError(errs, fmt.Errorf("lbank: unsupported kline interval %v", s.Interval))
					continue
				}
				req = map[string]any{
					"action":    action,
					"subscribe": chName,
					"kbar":      intervalStr,
					"pair":      p.Lower().String(),
				}
			default:
				req = map[string]any{
					"action":    action,
					"subscribe": chName,
					"pair":      p.Lower().String(),
				}
			}
			if err := e.Websocket.Conn.SendJSONMessage(ctx, 0, req); err != nil {
				errs = common.AppendError(errs, err)
				continue
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
