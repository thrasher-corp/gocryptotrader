package hyperliquid

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	websocketPingInterval           = 25 * time.Second
	websocketMessageInterval        = 30 * time.Millisecond
	maximumWebsocketSubscriptions   = 998 // Reserve two of Hyperliquid's 1000 per-IP subscriptions for account feeds.
	wsMethodSubscribe               = "subscribe"
	wsMethodUnsubscribe             = "unsubscribe"
	wsChannelActiveAssetContext     = "activeAssetCtx"
	wsChannelActiveSpotAssetContext = "activeSpotAssetCtx"
	wsChannelOrderbook              = "l2Book"
	wsChannelTrades                 = "trades"
	wsChannelCandle                 = "candle"
	wsChannelOrderUpdates           = "orderUpdates"
	wsChannelUserFills              = "userFills"
	wsChannelSubscriptionResponse   = "subscriptionResponse"
	wsChannelPong                   = "pong"
	wsChannelError                  = "error"
	websocketConnectionEstablished  = "Websocket connection established."
)

var (
	defaultSubscriptions = subscription.List{
		{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
	}
	errWebsocketAssetMismatch = errors.New("websocket market asset mismatch")
	errWebsocketServer        = errors.New("websocket server error")
	errWebsocketSubscription  = errors.New("websocket subscription acknowledgement error")
)

var websocketSubscriptionNames = map[string]string{
	subscription.TickerChannel:    wsChannelActiveAssetContext,
	subscription.OrderbookChannel: wsChannelOrderbook,
	subscription.AllTradesChannel: wsChannelTrades,
	subscription.CandlesChannel:   wsChannelCandle,
	subscription.MyOrdersChannel:  wsChannelOrderUpdates,
	subscription.MyTradesChannel:  wsChannelUserFills,
}

type websocketRequest struct {
	Method       string                `json:"method"`
	Subscription websocketSubscription `json:"subscription"`
}

type websocketSubscription struct {
	Type            string `json:"type"`
	Coin            string `json:"coin,omitempty"`
	Interval        string `json:"interval,omitempty"`
	User            string `json:"user,omitempty"`
	AggregateByTime bool   `json:"aggregateByTime,omitempty"`
}

type websocketSubscriptionResponse struct {
	Method       string                `json:"method"`
	Subscription websocketSubscription `json:"subscription"`
}

type websocketPendingKey struct {
	authenticated bool
	subscription  websocketSubscription
}

type websocketPendingOperation struct {
	method        string
	connection    websocket.Connection
	subscription  *subscription.Subscription
	previousState subscription.State
	done          chan error
}

type websocketEnvelope struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

type websocketAssetContext struct {
	Coin    string          `json:"coin"`
	Context json.RawMessage `json:"ctx"`
}

type websocketUserFills struct {
	IsSnapshot bool            `json:"isSnapshot"`
	User       string          `json:"user"`
	Fills      []websocketFill `json:"fills"`
}

type websocketFill struct {
	Coin          string       `json:"coin"`
	Price         types.Number `json:"px"`
	Size          types.Number `json:"sz"`
	Side          string       `json:"side"`
	Time          types.Time   `json:"time"`
	Hash          string       `json:"hash"`
	OrderID       uint64       `json:"oid"`
	TradeID       uint64       `json:"tid"`
	ClientOrderID *string      `json:"cloid"`
}

// WsConnect connects the public stream and, when configured, an address-scoped account stream.
func (e *Exchange) WsConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	if err := e.connectWebsocket(ctx, e.Websocket.Conn); err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.websocketReadLoop(ctx, e.Websocket.Conn, false)

	if !e.IsWebsocketAuthenticationSupported() {
		return nil
	}
	if _, err := e.getWatchAddress(ctx); err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Warnf(log.WebsocketMgr, "%s address-scoped websocket disabled: %s", e.Name, err)
		return nil
	}
	if err := e.connectWebsocket(ctx, e.Websocket.AuthConn); err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		log.Warnf(log.WebsocketMgr, "%s address-scoped websocket connection failed: %s", e.Name, err)
		return nil
	}
	// Hyperliquid does not authenticate websocket subscriptions. In this adapter,
	// "authenticated" means only that an address is configured for account-scoped feeds.
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	e.Websocket.Wg.Add(1)
	go e.websocketReadLoop(ctx, e.Websocket.AuthConn, true)
	return nil
}

func (e *Exchange) connectWebsocket(ctx context.Context, connection websocket.Connection) error {
	if connection == nil {
		return common.ErrNilPointer
	}
	if err := connection.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	connection.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"method":"ping"}`),
		Delay:       websocketPingInterval,
	})
	return nil
}

func (e *Exchange) websocketReadLoop(ctx context.Context, connection websocket.Connection, authenticated bool) {
	defer e.Websocket.Wg.Done()
	for {
		message := connection.ReadMessage()
		if message.Raw == nil {
			_ = e.failWebsocketPending(authenticated, websocket.ErrNotConnected)
			return
		}
		if err := e.websocketHandleDataForConnection(ctx, message.Raw, authenticated); err != nil {
			if sendErr := e.Websocket.DataHandler.Send(ctx, err); sendErr != nil {
				log.Errorf(log.WebsocketMgr, "%s websocket data handler: %s; source error: %s", e.Name, sendErr, err)
			}
		}
	}
}

func (e *Exchange) websocketHandleData(ctx context.Context, raw []byte) error {
	return e.websocketHandleDataForConnection(ctx, raw, false)
}

func (e *Exchange) websocketHandleDataForConnection(ctx context.Context, raw []byte, authenticated bool) error {
	if string(raw) == websocketConnectionEstablished {
		return nil
	}
	var envelope websocketEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	switch envelope.Channel {
	case wsChannelSubscriptionResponse:
		var response websocketSubscriptionResponse
		if err := json.Unmarshal(envelope.Data, &response); err != nil {
			return fmt.Errorf("%w: %w", errWebsocketSubscription, err)
		}
		if (response.Method != wsMethodSubscribe && response.Method != wsMethodUnsubscribe) ||
			strings.TrimSpace(response.Subscription.Type) == "" {
			return fmt.Errorf("%w: invalid response method %q or subscription", errWebsocketSubscription, response.Method)
		}
		key := websocketPendingKey{authenticated: authenticated, subscription: response.Subscription}
		e.websocketPendingMu.Lock()
		pending := e.websocketPending[key]
		if pending == nil {
			e.websocketPendingMu.Unlock()
			return nil
		}
		if pending.method != response.Method {
			e.websocketPendingMu.Unlock()
			return fmt.Errorf("%w: expected %s, got %s", errWebsocketSubscription, pending.method, response.Method)
		}
		delete(e.websocketPending, key)
		e.websocketPendingMu.Unlock()

		var err error
		switch response.Method {
		case wsMethodSubscribe:
			err = pending.subscription.SetState(subscription.SubscribedState)
		case wsMethodUnsubscribe:
			err = e.Websocket.RemoveSubscriptions(pending.connection, pending.subscription)
		}
		if err != nil {
			err = common.AppendError(err, e.rollbackWebsocketPending(pending))
		}
		pending.done <- err
		return err
	case wsChannelPong:
		return nil
	case wsChannelError:
		var message string
		if err := json.Unmarshal(envelope.Data, &message); err != nil {
			message = strings.TrimSpace(string(envelope.Data))
		}
		if message == "" || message == "null" {
			message = "unspecified error"
		}
		return e.failWebsocketPending(authenticated, fmt.Errorf("%w: %s", errWebsocketServer, message))
	case wsChannelActiveAssetContext:
		return e.websocketHandleTicker(ctx, envelope.Data, asset.PerpetualContract)
	case wsChannelActiveSpotAssetContext:
		return e.websocketHandleTicker(ctx, envelope.Data, asset.Spot)
	case wsChannelOrderbook:
		return e.websocketHandleOrderbook(ctx, envelope.Data)
	case wsChannelTrades:
		return e.websocketHandleTrades(ctx, envelope.Data)
	case wsChannelCandle:
		return e.websocketHandleCandle(ctx, envelope.Data)
	case wsChannelOrderUpdates:
		return e.websocketHandleOrderUpdates(ctx, envelope.Data)
	case wsChannelUserFills:
		return e.websocketHandleUserFills(ctx, envelope.Data)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(raw),
		})
	}
}

func (e *Exchange) rollbackWebsocketPending(pending *websocketPendingOperation) error {
	if e == nil || e.Websocket == nil || pending == nil || pending.connection == nil || pending.subscription == nil {
		return common.ErrNilPointer
	}
	switch pending.method {
	case wsMethodSubscribe:
		return e.Websocket.RemoveSubscriptions(pending.connection, pending.subscription)
	case wsMethodUnsubscribe:
		if pending.subscription.State() == pending.previousState {
			return nil
		}
		return pending.subscription.SetState(pending.previousState)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, pending.method)
	}
}

func (e *Exchange) failWebsocketPending(authenticated bool, cause error) error {
	if cause == nil {
		return common.ErrNilPointer
	}
	e.websocketPendingMu.Lock()
	pending := make([]*websocketPendingOperation, 0, len(e.websocketPending))
	for key, operation := range e.websocketPending {
		if key.authenticated == authenticated {
			delete(e.websocketPending, key)
			pending = append(pending, operation)
		}
	}
	e.websocketPendingMu.Unlock()

	result := cause
	for i := range pending {
		rollbackErr := e.rollbackWebsocketPending(pending[i])
		pending[i].done <- common.AppendError(cause, rollbackErr)
		result = common.AppendError(result, rollbackErr)
	}
	return result
}

func (e *Exchange) abortWebsocketPending(
	key *websocketPendingKey,
	pending *websocketPendingOperation,
	cause error,
) error {
	if e == nil || key == nil || pending == nil || pending.done == nil || cause == nil {
		return common.ErrNilPointer
	}
	e.websocketPendingMu.Lock()
	if e.websocketPending[*key] != pending {
		e.websocketPendingMu.Unlock()
		return <-pending.done
	}
	delete(e.websocketPending, *key)
	e.websocketPendingMu.Unlock()
	return common.AppendError(cause, e.rollbackWebsocketPending(pending))
}

func (e *Exchange) websocketHandleTicker(ctx context.Context, raw []byte, expectedAsset asset.Item) error {
	var update websocketAssetContext
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	mapping, a, err := e.lookupPairMappingByCoin(update.Coin)
	if err != nil {
		return err
	}
	if a != expectedAsset {
		return fmt.Errorf("%w: expected %s, got %s for %s", errWebsocketAssetMismatch, expectedAsset, a, update.Coin)
	}
	price := &ticker.Price{
		ExchangeName: e.Name,
		Pair:         mapping.pair,
		AssetType:    a,
		LastUpdated:  time.Now().UTC(),
	}
	switch a {
	case asset.Spot:
		var market SpotAssetContext
		if err := json.Unmarshal(update.Context, &market); err != nil {
			return err
		}
		price.Last = market.MidPrice.Float64()
		if price.Last == 0 {
			price.Last = market.MarkPrice.Float64()
		}
		price.Open = market.PreviousDayPrice.Float64()
		price.Volume = market.DayBaseVolume.Float64()
		price.QuoteVolume = market.DayNotionalVolume.Float64()
		price.MarkPrice = market.MarkPrice.Float64()
	case asset.PerpetualContract:
		var market PerpetualAssetContext
		if err := json.Unmarshal(update.Context, &market); err != nil {
			return err
		}
		price.Last = market.MidPrice.Float64()
		if price.Last == 0 {
			price.Last = market.MarkPrice.Float64()
		}
		price.Open = market.PreviousDayPrice.Float64()
		price.Volume = market.DayBaseVolume.Float64()
		price.QuoteVolume = market.DayNotionalVolume.Float64()
		price.OpenInterest = market.OpenInterest.Float64()
		price.MarkPrice = market.MarkPrice.Float64()
		price.IndexPrice = market.OraclePrice.Float64()
	}
	return e.Websocket.DataHandler.Send(ctx, price)
}

func (e *Exchange) websocketHandleOrderbook(_ context.Context, raw []byte) error {
	var update L2Book
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	if len(update.Levels) != 2 {
		return fmt.Errorf("%w: expected 2 sides, got %d", errInvalidBookLevelCount, len(update.Levels))
	}
	mapping, a, err := e.lookupPairMappingByCoin(update.Coin)
	if err != nil {
		return err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              mapping.pair,
		Asset:             a,
		LastUpdated:       update.Time.Time().UTC(),
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              make(orderbook.Levels, len(update.Levels[0])),
		Asks:              make(orderbook.Levels, len(update.Levels[1])),
	}
	for i := range update.Levels[0] {
		book.Bids[i] = orderbook.Level{Price: update.Levels[0][i].Price.Float64(), Amount: update.Levels[0][i].Size.Float64()}
	}
	for i := range update.Levels[1] {
		book.Asks[i] = orderbook.Level{Price: update.Levels[1][i].Price.Float64(), Amount: update.Levels[1][i].Size.Float64()}
	}
	return e.Websocket.Orderbook.LoadSnapshot(book)
}

func (e *Exchange) websocketHandleTrades(_ context.Context, raw []byte) error {
	var updates []RecentTrade
	if err := json.Unmarshal(raw, &updates); err != nil {
		return err
	}
	trades := make([]trade.Data, len(updates))
	for i := range updates {
		mapping, a, err := e.lookupPairMappingByCoin(updates[i].Coin)
		if err != nil {
			return err
		}
		var side order.Side
		switch updates[i].Side {
		case "A":
			side = order.Sell
		case "B":
			side = order.Buy
		default:
			return fmt.Errorf("%w: %q", order.ErrSideIsInvalid, updates[i].Side)
		}
		trades[i] = trade.Data{
			TID:          strconv.FormatUint(updates[i].TradeID, 10),
			Exchange:     e.Name,
			CurrencyPair: mapping.pair,
			AssetType:    a,
			Side:         side,
			Price:        updates[i].Price.Float64(),
			Amount:       updates[i].Size.Float64(),
			Timestamp:    updates[i].Time.Time().UTC(),
		}
	}
	return e.Websocket.Trade.Update(e.IsSaveTradeDataEnabled(), trades...)
}

func (e *Exchange) websocketHandleCandle(ctx context.Context, raw []byte) error {
	var update Candle
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	mapping, a, err := e.lookupPairMappingByCoin(update.Symbol)
	if err != nil {
		return err
	}
	interval, err := parseWebsocketInterval(update.Interval)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Exchange: e.Name,
		Asset:    a,
		Pair:     mapping.pair,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   update.OpenTime.Time().UTC(),
			Open:   update.Open.Float64(),
			High:   update.High.Float64(),
			Low:    update.Low.Float64(),
			Close:  update.Close.Float64(),
			Volume: update.Volume.Float64(),
		}},
	})
}

func (e *Exchange) websocketHandleOrderUpdates(ctx context.Context, raw []byte) error {
	var updates []HistoricalOrder
	if err := json.Unmarshal(raw, &updates); err != nil {
		return err
	}
	orders := make([]order.Detail, 0, len(updates))
	var errs error
	for i := range updates {
		mapping, a, err := e.lookupPairMappingByCoin(updates[i].Order.Coin)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("order update %d: %w", i, err))
			continue
		}
		converted, err := e.convertOrderFromMapping(&updates[i].Order, updates[i].Status, updates[i].StatusTimestamp.Time(), &mapping, a)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("order update %d: %w", i, err))
			continue
		}
		orders = append(orders, converted)
	}
	if len(orders) > 0 || (len(updates) == 0 && errs == nil) {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, orders))
	}
	return errs
}

func (e *Exchange) websocketHandleUserFills(_ context.Context, raw []byte) error {
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	var update websocketUserFills
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}
	if update.IsSnapshot {
		return nil
	}
	fills := make([]fill.Data, 0, len(update.Fills))
	var errs error
	for i := range update.Fills {
		mapping, a, err := e.lookupPairMappingByCoin(update.Fills[i].Coin)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("user fill %d: %w", i, err))
			continue
		}
		var side order.Side
		switch update.Fills[i].Side {
		case "A":
			side = order.Sell
		case "B":
			side = order.Buy
		default:
			errs = common.AppendError(errs, fmt.Errorf("user fill %d: %w: %q", i, order.ErrSideIsInvalid, update.Fills[i].Side))
			continue
		}
		clientOrderID := ""
		if update.Fills[i].ClientOrderID != nil {
			clientOrderID = *update.Fills[i].ClientOrderID
		}
		fills = append(fills, fill.Data{
			ID:            strconv.FormatUint(update.Fills[i].TradeID, 10),
			Timestamp:     update.Fills[i].Time.Time().UTC(),
			Exchange:      e.Name,
			AssetType:     a,
			CurrencyPair:  mapping.pair,
			Side:          side,
			OrderID:       strconv.FormatUint(update.Fills[i].OrderID, 10),
			ClientOrderID: clientOrderID,
			TradeID:       strconv.FormatUint(update.Fills[i].TradeID, 10),
			Price:         update.Fills[i].Price.Float64(),
			Amount:        update.Fills[i].Size.Float64(),
		})
	}
	if len(fills) > 0 || (len(update.Fills) == 0 && errs == nil) {
		errs = common.AppendError(errs, e.Websocket.Fills.Update(fills...))
	}
	return errs
}

func parseWebsocketInterval(interval string) (kline.Interval, error) {
	for _, candidate := range []kline.Interval{
		kline.OneMin,
		kline.ThreeMin,
		kline.FiveMin,
		kline.FifteenMin,
		kline.ThirtyMin,
		kline.OneHour,
		kline.TwoHour,
		kline.FourHour,
		kline.EightHour,
		kline.TwelveHour,
		kline.OneDay,
		kline.ThreeDay,
		kline.OneWeek,
		kline.OneMonth,
	} {
		formatted, _ := formatInterval(candidate)
		if formatted == interval {
			return candidate, nil
		}
	}
	return 0, fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

func websocketChannelName(sub *subscription.Subscription) (string, error) {
	if sub == nil {
		return "", common.ErrNilPointer
	}
	name, ok := websocketSubscriptionNames[sub.Channel]
	if !ok {
		return "", subscription.ErrNotSupported
	}
	return name, nil
}

// GetSubscriptionTemplate returns the channel qualification template.
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName": websocketChannelName,
		"interval": func(value kline.Interval) string {
			formatted, _ := formatInterval(value)
			return formatted
		},
	}).Parse(websocketSubscriptionTemplate)
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// Subscribe sends Hyperliquid subscription requests.
func (e *Exchange) Subscribe(subscriptions subscription.List) error {
	ctx := context.TODO()
	for _, sub := range subscriptions {
		if sub == nil {
			return common.ErrNilPointer
		}
	}
	expanded, errs := subscriptions.ExpandTemplates(e)
	for _, sub := range expanded {
		errs = common.AppendError(errs, e.manageWebsocketSubscription(ctx, wsMethodSubscribe, sub))
	}
	return errs
}

// Unsubscribe sends Hyperliquid unsubscription requests.
func (e *Exchange) Unsubscribe(subscriptions subscription.List) error {
	ctx := context.TODO()
	for _, sub := range subscriptions {
		if sub == nil {
			return common.ErrNilPointer
		}
	}
	expanded, errs := subscriptions.ExpandTemplates(e)
	for _, key := range expanded {
		sub := e.Websocket.GetSubscription(key)
		if sub == nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", subscription.ErrNotFound, key))
			continue
		}
		errs = common.AppendError(errs, e.manageWebsocketSubscription(ctx, wsMethodUnsubscribe, sub))
	}
	return errs
}

func (e *Exchange) manageWebsocketSubscription(ctx context.Context, method string, sub *subscription.Subscription) error {
	if sub == nil {
		return common.ErrNilPointer
	}
	if method != wsMethodSubscribe && method != wsMethodUnsubscribe {
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, method)
	}
	payload, err := e.websocketSubscriptionPayload(ctx, sub)
	if err != nil {
		return err
	}
	connection := e.Websocket.Conn
	if sub.Authenticated {
		connection = e.Websocket.AuthConn
	}
	if connection == nil {
		return common.ErrNilPointer
	}
	previousState := sub.State()
	if method == wsMethodSubscribe {
		if err := e.Websocket.AddSubscriptions(connection, sub); err != nil {
			return err
		}
	} else if err := sub.SetState(subscription.UnsubscribingState); err != nil {
		return err
	}
	key := websocketPendingKey{authenticated: sub.Authenticated, subscription: payload}
	pending := &websocketPendingOperation{
		method:        method,
		connection:    connection,
		subscription:  sub,
		previousState: previousState,
		done:          make(chan error, 1),
	}
	e.websocketPendingMu.Lock()
	if e.websocketPending == nil {
		e.websocketPending = make(map[websocketPendingKey]*websocketPendingOperation)
	}
	if e.websocketPending[key] != nil {
		e.websocketPendingMu.Unlock()
		return common.AppendError(
			fmt.Errorf("%w: operation already pending for %s", errWebsocketSubscription, sub),
			e.rollbackWebsocketPending(pending))
	}
	e.websocketPending[key] = pending
	e.websocketPendingMu.Unlock()

	if err := connection.SendJSONMessage(ctx, request.Unset, websocketRequest{Method: method, Subscription: payload}); err != nil {
		return e.abortWebsocketPending(&key, pending, err)
	}
	timeout := e.WebsocketResponseMaxLimit
	if timeout <= 0 {
		timeout = exchange.DefaultWebsocketResponseMaxLimit
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-pending.done:
		return err
	case <-ctx.Done():
		return e.abortWebsocketPending(&key, pending, ctx.Err())
	case <-timer.C:
		return e.abortWebsocketPending(
			&key,
			pending,
			fmt.Errorf("%w: %s", websocket.ErrSignatureTimeout, sub))
	}
}

func (e *Exchange) websocketSubscriptionPayload(ctx context.Context, sub *subscription.Subscription) (websocketSubscription, error) {
	if sub == nil {
		return websocketSubscription{}, common.ErrNilPointer
	}
	name, ok := websocketSubscriptionNames[sub.Channel]
	if !ok {
		return websocketSubscription{}, subscription.ErrNotSupported
	}
	payload := websocketSubscription{Type: name}
	if sub.Authenticated {
		if sub.Channel != subscription.MyOrdersChannel && sub.Channel != subscription.MyTradesChannel {
			return websocketSubscription{}, subscription.ErrNotSupported
		}
		address, err := e.getWatchAddress(ctx)
		if err != nil {
			return websocketSubscription{}, err
		}
		payload.User = address
		return payload, nil
	}
	if sub.Channel == subscription.MyOrdersChannel || sub.Channel == subscription.MyTradesChannel {
		return websocketSubscription{}, subscription.ErrNotSupported
	}
	if len(sub.Pairs) != 1 {
		return websocketSubscription{}, subscription.ErrNotSinglePair
	}
	mapping, err := e.getPairMapping(ctx, sub.Pairs[0], sub.Asset)
	if err != nil {
		return websocketSubscription{}, err
	}
	payload.Coin = mapping.coin
	if sub.Channel == subscription.CandlesChannel {
		payload.Interval, err = formatInterval(sub.Interval)
		if err != nil {
			return websocketSubscription{}, err
		}
	}
	return payload, nil
}

const websocketSubscriptionTemplate = `
{{- if $.S.Asset }}
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- range $pair := $pairs }}
			{{- channelName $.S -}} : {{- $asset -}} : {{- $pair -}}
			{{- if eq $.S.Channel "candles" -}} : {{- interval $.S.Interval -}}{{- end -}}
			{{ $.PairSeparator }}
		{{- end }}
		{{ $.AssetSeparator }}
	{{- end }}
{{- else -}}
	{{ channelName $.S }}
{{- end }}
`
