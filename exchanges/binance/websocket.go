package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443/stream"
	pingDelay                  = time.Minute * 9

	wsSubscribeMethod   = "SUBSCRIBE"
	wsUnsubscribeMethod = "UNSUBSCRIBE"

	// serverShutdownStream is the combined stream name of the serverShutdown event
	serverShutdownStream = "!serverShutdown"
)

var (
	errUnhandledMessage              = errors.New("unhandled websocket message")
	errUnsupportedChannel            = errors.New("unsupported channel")
	errUnsupportedLevels             = errors.New("unsupported order book levels")
	errUnsupportedFrequency          = errors.New("unsupported order book update frequency")
	errUnknownOrderStatus            = errors.New("unknown order status")
	errOrderbookSubscriptionConflict = errors.New("order book subscription conflicts with another for the same books")
)

// Depths, in levels a side, of the REST snapshots diff depth streams are synchronised with. Update IDs keep a book exact
// whatever the snapshot's depth, and a level beyond it is learned when it changes, so the depth trades how much of the
// book is known at once against request weight. USDⓈ-M and COIN-M take the 1000 levels their procedures ask for.
// Spot takes 1000 rather than its procedure's 5000, which costs 250 weight instead of 50. Options books hold a few
// dozen levels a side (47 at most across the busiest symbols in a live sample), so they take 50, the deepest snapshot
// costing 1 weight, rather than 1000 at 20 weight out of the 400 a minute their orders share
const (
	spotOrderbookSnapshotLimit    = 1000
	optionsOrderbookSnapshotLimit = 50
)

// spotStreamConnectionSetup returns the setup of the spot market data stream connection
func (e *Exchange) spotStreamConnectionSetup(exch *config.Exchange) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   binanceDefaultWebsocketURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		Handler:               e.wsHandleData,
		GenerateSubscriptions: e.generateSubscriptions,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		// Binance accepts 5 messages a second from a connection, pings and pongs included
		RateLimit:     request.NewWeightedRateLimitByDuration(250 * time.Millisecond),
		MessageFilter: asset.Spot,
	}
}

// WsConnect connects to the spot market data streams
func (e *Exchange) WsConnect(ctx context.Context, conn websocket.Connection) error {
	dialer := gws.Dialer{
		HandshakeTimeout: e.Config.HTTPTimeout,
		Proxy:            http.ProxyFromEnvironment,
	}
	if err := conn.Dial(ctx, &dialer, http.Header{}, nil); err != nil {
		return fmt.Errorf("%s unable to connect to the market data streams: %w", e.Name, err)
	}
	conn.SetupPingHandler(wsConnectionMessageRate, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})
	return nil
}

// wsHandleData routes the market data stream connection's subscription responses to their requests and its stream
// payloads to their handlers
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if id, err := jsonparser.GetString(respRaw, "id"); err == nil {
		return conn.RequireMatchWithData(id, respRaw)
	}
	stream, err := jsonparser.GetString(respRaw, "stream")
	if err != nil {
		return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
	}
	data, _, _, err := jsonparser.Get(respRaw, "data")
	if err != nil {
		return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
	}
	if stream == serverShutdownStream {
		return e.handleServerShutdown(data)
	}
	symbol, channel, _ := strings.Cut(stream, "@")
	switch {
	case channel == "trade":
		return e.processTrade(data)
	case channel == "ticker":
		return e.processTicker(ctx, data)
	case strings.HasPrefix(channel, "kline_"):
		return e.processKline(ctx, data)
	}
	if levels, ok := strings.CutPrefix(channel, "depth"); ok {
		// Diff depth streams are depth and depth@<speed>; partial depth streams name their levels
		if levels == "" || levels[0] == '@' {
			return e.processDepthUpdate(ctx, data)
		}
		return e.processPartialDepth(ctx, symbol, data)
	}
	return fmt.Errorf("%w: %s", errUnhandledMessage, respRaw)
}

// handleServerShutdown reconnects when a serverShutdown event warns that the server is about to close the connection
func (e *Exchange) handleServerShutdown(event []byte) error {
	var resp WsServerShutdown
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	log.Warnf(log.WebsocketMgr, "%s server shutting down at %s, reconnecting", e.Name, resp.EventTime.Time())
	// Shutting down waits for this connection's reader, so it cannot run on it; the connection monitor reconnects
	go func() {
		if err := e.Websocket.Shutdown(); err != nil {
			log.Errorf(log.WebsocketMgr, "%s unable to shut down for reconnection: %s", e.Name, err)
		}
	}()
	return nil
}

// processTrade relays a Trade Streams payload as a trade
func (e *Exchange) processTrade(data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var resp TradeStream
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.MatchSymbolCheckEnabled(resp.Symbol, asset.Spot, false)
	if err != nil || !enabled {
		return err
	}
	side := order.Buy
	if resp.IsBuyerMaker {
		// The seller took the buyer's order
		side = order.Sell
	}
	return e.Websocket.Trade.Update(saveTradeData, trade.Data{
		Exchange:     e.Name,
		CurrencyPair: pair,
		AssetType:    asset.Spot,
		Side:         side,
		TID:          strconv.FormatUint(resp.TradeID, 10),
		Price:        resp.Price.Float64(),
		Amount:       resp.Quantity.Float64(),
		Timestamp:    resp.TradeTime.Time(),
	})
}

// processTicker stores an Individual Symbol Ticker Streams payload and relays it
func (e *Exchange) processTicker(ctx context.Context, data []byte) error {
	var resp TickerStream
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.MatchSymbolCheckEnabled(resp.Symbol, asset.Spot, false)
	if err != nil || !enabled {
		return err
	}
	tick := &ticker.Price{
		Last:                       resp.LastPrice.Float64(),
		LastSize:                   resp.LastQuantity.Float64(),
		VolumeWeightedAveragePrice: resp.WeightedAveragePrice.Float64(),
		High:                       resp.HighPrice.Float64(),
		Low:                        resp.LowPrice.Float64(),
		Bid:                        resp.BestBidPrice.Float64(),
		BidSize:                    resp.BestBidQuantity.Float64(),
		Ask:                        resp.BestAskPrice.Float64(),
		AskSize:                    resp.BestAskQuantity.Float64(),
		BaseVolume:                 resp.TotalTradedBaseAssetVolume.Float64(),
		QuoteVolume:                resp.TotalTradedQuoteAssetVolume.Float64(),
		Open:                       resp.OpenPrice.Float64(),
		PercentChange24Hour:        resp.PriceChangePercent.Float64(),
		// Close is the previous close, as the REST ticker reports it
		Close:        resp.PreviousClosePrice.Float64(),
		Pair:         pair,
		ExchangeName: e.Name,
		AssetType:    asset.Spot,
		LastUpdated:  resp.EventTime.Time(),
	}
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// processKline relays a Kline/Candlestick Streams payload as a candle
func (e *Exchange) processKline(ctx context.Context, data []byte) error {
	var resp KlineStream
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.MatchSymbolCheckEnabled(resp.Symbol, asset.Spot, false)
	if err != nil || !enabled {
		return err
	}
	interval, err := formatToInterval(resp.Kline.Interval)
	if err != nil {
		return err
	}
	var validationIssues string
	if !resp.Kline.IsClosed {
		validationIssues = kline.PartialCandle
	}
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Pair:     pair,
		Asset:    asset.Spot,
		Exchange: e.Name,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:             resp.Kline.StartTime.Time(),
			Open:             resp.Kline.OpenPrice.Float64(),
			Close:            resp.Kline.ClosePrice.Float64(),
			High:             resp.Kline.HighPrice.Float64(),
			Low:              resp.Kline.LowPrice.Float64(),
			Volume:           resp.Kline.BaseAssetVolume.Float64(),
			ValidationIssues: validationIssues,
		}},
	})
}

// processDepthUpdate passes a Diff. Depth Stream payload of an enabled pair to the order book synchronisation
func (e *Exchange) processDepthUpdate(ctx context.Context, data []byte) error {
	var resp DiffDepthStream
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	pair, enabled, err := e.MatchSymbolCheckEnabled(resp.Symbol, asset.Spot, false)
	if err != nil || !enabled {
		return err
	}
	return e.processOrderbookUpdate(ctx, resp.FirstUpdateID, &orderbook.Update{
		UpdateID:   resp.FinalUpdateID,
		UpdateTime: resp.EventTime.Time(),
		Asset:      asset.Spot,
		Bids:       resp.Bids.Levels(),
		Asks:       resp.Asks.Levels(),
		Pair:       pair,
		// An event without level changes still moves the book's update ID on, or the next event would look like a gap
		AllowEmpty: true,
	})
}

// processOrderbookUpdate passes a diff depth event to the order book synchronisation, which serves every asset. The
// synchronisation applies an event to a synchronised book only when its first update ID follows the book's last one,
// and otherwise invalidates the book and synchronises it again from a new snapshot. Binance's procedures are applied
// first: an event the book has already moved past is dropped, so a late or repeated event cannot invalidate it, and a
// spot event overlapping the book is applied from the book's next update, as the spot procedure allows; derivatives
// events must instead chain to the previous one by pu
func (e *Exchange) processOrderbookUpdate(ctx context.Context, firstUpdateID int64, update *orderbook.Update) error {
	if e.orderbookSync == nil {
		return errOrderbookSyncNotSetUp
	}
	if last, err := e.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset); err == nil {
		if update.UpdateID <= last {
			return nil
		}
		if update.Asset == asset.Spot && firstUpdateID <= last {
			// The event's levels are absolute, so the updates the book already holds apply again unchanged
			firstUpdateID = last + 1
		}
	}
	return e.orderbookSync.ProcessOrderbookUpdate(ctx, firstUpdateID, update)
}

// orderbookSnapshotLimit returns the depth of the REST snapshot an asset's diff depth streams are synchronised with
func orderbookSnapshotLimit(a asset.Item) uint64 {
	switch a {
	case asset.Spot:
		return spotOrderbookSnapshotLimit
	case asset.Options:
		return optionsOrderbookSnapshotLimit
	default:
		return derivativesOrderbookSnapshotLimit
	}
}

// fetchOrderbookSnapshot returns the REST order book snapshot an asset's diff depth streams are synchronised with
func (e *Exchange) fetchOrderbookSnapshot(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if a != asset.Spot {
		return e.fetchDerivativesOrderbookDepth(ctx, p, a, orderbookSnapshotLimit(a))
	}
	resp, err := e.GetOrderBook(ctx, &OrderBookRequest{Symbol: p, Limit: orderbookSnapshotLimit(a)})
	if err != nil {
		return nil, err
	}
	return &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              resp.Bids.Levels(),
		Asks:              resp.Asks.Levels(),
		LastUpdateID:      resp.LastUpdateID,
		// The spot snapshot carries no time
		LastUpdated: time.Now(),
	}, nil
}

// checkPendingUpdate decides whether a diff depth event buffered while a snapshot was fetched applies to that snapshot,
// by the procedure of the event's product
func checkPendingUpdate(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error) {
	if update.Asset == asset.Spot {
		return checkSpotPendingUpdate(lastUpdateID, firstUpdateID, update)
	}
	return checkDerivativesPendingUpdate(lastUpdateID, firstUpdateID, update)
}

// checkSpotPendingUpdate applies the spot local order book procedure to diff events buffered while a snapshot was
// fetched. Events whose final update ID u is at or before the snapshot's lastUpdateId are dropped, and the first event
// applied must contain lastUpdateId+1: U <= lastUpdateId+1 <= u. An event starting later means updates between the
// snapshot and the buffered events were missed, so the book needs a new snapshot
func checkSpotPendingUpdate(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error) {
	if update.UpdateID <= lastUpdateID {
		return true, nil
	}
	if firstUpdateID > lastUpdateID+1 {
		return false, fmt.Errorf("%w: snapshot update ID %d precedes first update ID %d of %s %s", orderbookmanager.ErrOrderbookSnapshotOutdated, lastUpdateID, firstUpdateID, update.Asset, update.Pair)
	}
	return false, nil
}

// processPartialDepth loads a Partial Book Depth Streams payload, a complete top of the book, as a snapshot; the
// payload carries no symbol, so it comes from the stream name
func (e *Exchange) processPartialDepth(ctx context.Context, symbol string, data []byte) error {
	pair, enabled, err := e.MatchSymbolCheckEnabled(symbol, asset.Spot, false)
	if err != nil || !enabled {
		return err
	}
	var resp WsPartialDepth
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
		Pair:              pair,
		Exchange:          e.Name,
		Asset:             asset.Spot,
		ValidateOrderbook: e.ValidateOrderbook,
		LastUpdateID:      resp.LastUpdateID,
		// The payload carries no time
		LastUpdated: time.Now(),
		Bids:        orderbook.Levels(resp.Bids),
		Asks:        orderbook.Levels(resp.Asks),
	})
}

// stringToOrderStatus converts a spot order status
func stringToOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "PENDING_NEW":
		return order.Pending, nil
	case "PARTIALLY_FILLED":
		return order.PartiallyFilled, nil
	case "FILLED":
		return order.Filled, nil
	case "CANCELED":
		return order.Cancelled, nil
	case "PENDING_CANCEL":
		return order.PendingCancel, nil
	case "REJECTED":
		return order.Rejected, nil
	case "EXPIRED", "EXPIRED_IN_MATCH":
		// EXPIRED_IN_MATCH is an expiry by self-trade prevention, which leaves the order as closed as any expiry
		return order.Expired, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownOrderStatus, status)
	}
}

// generateSubscriptions returns the spot stream subscriptions of the enabled spot pairs. A subscription that cannot be
// expanded, or an order book subscription left out for conflicting with another, is reported as partial generation,
// so it does not stop the other connections
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	if !e.GetAssetTypes(true).Contains(asset.Spot) {
		return subscription.List{}, nil
	}
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, s := range e.Features.Subscriptions {
		switch s.Asset {
		case asset.Empty:
			// Configurations from before subscriptions had assets mean spot
			s = s.Clone()
			s.Asset = asset.Spot
		case asset.Spot:
		default:
			// The other assets' streams have their own connections
			continue
		}
		subs = append(subs, s)
	}
	expanded, err := subs.ExpandTemplates(e)
	expanded, conflicts := oneOrderbookSubscription(expanded)
	if err = common.AppendError(err, conflicts); err != nil {
		return expanded, fmt.Errorf("%w: %w", websocket.ErrSubscriptionPartial, err)
	}
	return expanded, nil
}

// oneOrderbookSubscription keeps one order book subscription for each pair. A pair has one stored book, which a second
// depth stream would keep overwriting: partial depth replaces the book with its top levels and breaks the update ID
// chain diff depth needs, so every diff event after it forces a new REST snapshot. A pair's first diff depth
// subscription is kept, as it maintains the whole book, or its first partial depth one when there is none; the others
// are returned as errors
func oneOrderbookSubscription(subs subscription.List) (subscription.List, error) {
	chosen := make(map[key.PairAsset]*subscription.Subscription)
	for _, s := range subs {
		if s.Channel != subscription.OrderbookChannel {
			continue
		}
		for _, p := range s.Pairs {
			k := key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: s.Asset}
			if current, ok := chosen[k]; !ok || current.Levels != 0 && s.Levels == 0 {
				chosen[k] = s
			}
		}
	}
	kept := make(subscription.List, 0, len(subs))
	var errs error
	for _, s := range subs {
		if s.Channel == subscription.OrderbookChannel && slices.ContainsFunc(s.Pairs, func(p currency.Pair) bool {
			return chosen[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: s.Asset}] != s
		}) {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s levels %d interval %s", errOrderbookSubscriptionConflict, s.Pairs, s.Levels, s.Interval))
			continue
		}
		kept = append(kept, s)
	}
	return kept, errs
}

// subscriptionTemplate is the stream name template, parsed once
var subscriptionTemplate = sync.OnceValues(func() (*template.Template, error) {
	return template.New("subscriptions.tmpl").
		Funcs(template.FuncMap{
			"channel": channelName,
			"fmt":     currency.EMPTYFORMAT.Format,
		}).
		Parse(subTplText)
})

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return subscriptionTemplate()
}

// channelName returns a subscription's stream name without its symbol
func channelName(s *subscription.Subscription) (string, error) {
	switch s.Channel {
	case subscription.TickerChannel:
		return "ticker", nil
	case subscription.AllTradesChannel:
		return "trade", nil
	case subscription.CandlesChannel:
		interval, err := intervalToString(s.Interval)
		if err != nil {
			return "", err
		}
		return "kline_" + interval, nil
	case subscription.OrderbookChannel:
		name := "depth"
		switch s.Levels {
		case 0:
		case 5, 10, 20:
			name += strconv.Itoa(s.Levels)
		default:
			return "", fmt.Errorf("%w: %d", errUnsupportedLevels, s.Levels)
		}
		switch s.Interval {
		case 0, kline.ThousandMilliseconds:
			// Streams update every second unless 100ms is requested
		case kline.HundredMilliseconds:
			name += "@100ms"
		default:
			return "", fmt.Errorf("%w: %s", errUnsupportedFrequency, s.Interval)
		}
		return name, nil
	}
	return "", fmt.Errorf("%w: %q", errUnsupportedChannel, s.Channel)
}

// Subscribe subscribes to a set of channels
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, channels subscription.List) error {
	return e.ParallelChanOp(ctx, channels, func(ctx context.Context, l subscription.List) error {
		return e.manageSubs(ctx, conn, wsSubscribeMethod, l)
	}, 50)
}

// Unsubscribe unsubscribes from a set of channels
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, channels subscription.List) error {
	return e.ParallelChanOp(ctx, channels, func(ctx context.Context, l subscription.List) error {
		return e.manageSubs(ctx, conn, wsUnsubscribeMethod, l)
	}, 50)
}

// manageSubs subscribes or unsubscribes from a list of subscriptions
func (e *Exchange) manageSubs(ctx context.Context, conn websocket.Connection, op string, subs subscription.List) error {
	if op == wsSubscribeMethod {
		if err := e.Websocket.AddSubscriptions(conn, subs...); err != nil { // Note: AddSubscription will set state to subscribing
			return err
		}
	} else {
		if err := subs.SetStates(subscription.UnsubscribingState); err != nil {
			return err
		}
	}

	req := WsPayload{
		ID:     e.MessageID(),
		Method: op,
		Params: subs.QualifiedChannels(),
	}
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsConnectionMessageRate, req.ID, req)
	if err == nil {
		if v, d, _, rErr := jsonparser.Get(respRaw, "result"); rErr != nil {
			err = rErr
		} else if d != jsonparser.Null { // null is the only expected and acceptable response
			err = fmt.Errorf("%w: %w: %s", websocket.ErrSubscriptionFailure, common.ErrUnknownError, v)
		}
	}

	if err != nil {
		err = fmt.Errorf("%w; Channels: %s", err, strings.Join(subs.QualifiedChannels(), ", "))
		if op == wsSubscribeMethod {
			if err2 := e.Websocket.RemoveSubscriptions(conn, subs...); err2 != nil {
				err = common.AppendError(err, err2)
			}
		}
	} else {
		if op == wsSubscribeMethod {
			err = common.AppendError(err, subs.SetStates(subscription.SubscribedState))
		} else {
			err = e.Websocket.RemoveSubscriptions(conn, subs...)
		}
	}

	return err
}

var klineIntervalList = []*struct {
	Interval kline.Interval
	String   string
}{
	{String: "1s", Interval: kline.ThousandMilliseconds},
	{String: "1m", Interval: kline.OneMin},
	{String: "3m", Interval: kline.ThreeMin},
	{String: "5m", Interval: kline.FiveMin},
	{String: "15m", Interval: kline.FifteenMin},
	{String: "30m", Interval: kline.ThirtyMin},
	{String: "1h", Interval: kline.OneHour},
	{String: "2h", Interval: kline.TwoHour},
	{String: "4h", Interval: kline.FourHour},
	{String: "6h", Interval: kline.SixHour},
	{String: "8h", Interval: kline.EightHour},
	{String: "12h", Interval: kline.TwelveHour},
	{String: "1d", Interval: kline.OneDay},
	{String: "3d", Interval: kline.ThreeDay},
	{String: "1w", Interval: kline.OneWeek},
	{String: "1M", Interval: kline.OneMonth},
}

// formatToInterval returns the interval of a kline interval string
func formatToInterval(intervalString string) (kline.Interval, error) {
	for _, interval := range klineIntervalList {
		if interval.String == intervalString {
			return interval.Interval, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", kline.ErrInvalidInterval, intervalString)
}

// intervalToString returns the kline interval string of an interval
func intervalToString(interval kline.Interval) (string, error) {
	for _, i := range klineIntervalList {
		if i.Interval == interval {
			return i.String, nil
		}
	}
	return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
}

// subTplText builds the stream names of a subscription's pairs; symbols are lower case
const subTplText = `
{{- range $pair := index $.AssetPairs $.S.Asset -}}
	{{- fmt $pair }}@{{ channel $.S }}
	{{- $.PairSeparator }}
{{- end -}}
`
