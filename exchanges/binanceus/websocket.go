package binanceus

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
	websocketStreamsURL = "wss://stream.binance.us:9443/stream"
	pingDelay           = time.Minute * 9

	wsSubscribeMethod   = "SUBSCRIBE"
	wsUnsubscribeMethod = "UNSUBSCRIBE"

	// serverShutdownEvent is the event type of the serverShutdown event, and serverShutdownStream the combined stream
	// name it would arrive under
	serverShutdownEvent  = "serverShutdown"
	serverShutdownStream = "!serverShutdown"

	// subscriptionBatchSize is the most streams one SUBSCRIBE or UNSUBSCRIBE carries; the server closes a connection
	// sent a control message longer than 4096 bytes
	subscriptionBatchSize = 50
	// maxStreamsPerConnection is the most streams one connection carries
	maxStreamsPerConnection = 1024
)

// spotOrderbookSnapshotLimit is the depth, in levels a side, of the REST snapshots diff depth streams are synchronised
// with. Update IDs keep a book exact whatever the snapshot's depth, and a level beyond it is learned when it changes, so
// 1000 levels at weight 50 are taken rather than the procedure's 5000 at weight 250
const spotOrderbookSnapshotLimit = 1000

var (
	errUnhandledMessage              = errors.New("unhandled websocket message")
	errUnsupportedChannel            = errors.New("unsupported channel")
	errUnsupportedLevels             = errors.New("unsupported order book levels")
	errUnsupportedFrequency          = errors.New("unsupported order book update frequency")
	errUnknownOrderStatus            = errors.New("unknown order status")
	errOrderbookSubscriptionConflict = errors.New("order book subscription conflicts with another for the same books")
	errOrderbookSyncNotSetUp         = errors.New("order book synchronisation not set up")
	errOrderbookUnsubscribed         = errors.New("order book stream unsubscribed")
	errOrderbookStreamStopped        = errors.New("order book stream stopped while its snapshot was fetched")
)

// defaultSubscriptions are the stream subscriptions, each expanded to the enabled pairs
var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
}

// websocketManagerSetup returns the setup of the websocket manager. The books the streams load are invalidated when the
// websocket shuts down, as UpdateOrderbook's REST snapshots defer to them while they are updated
func (e *Exchange) websocketManagerSetup(exch *config.Exchange) *websocket.ManagerSetup {
	return &websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
		// WebSocket API requests draw on the same weight pools as their REST equivalents, while stream control messages
		// fall back to the stream connection's own message rate limit
		RateLimitDefinitions:                   e.Requester.GetRateLimiterDefinitions(),
		MaxWebsocketSubscriptionsPerConnection: maxStreamsPerConnection,
		InvalidateOrderbooksOnShutdown:         true,
	}
}

// streamConnectionSetup returns the setup of the market data stream connection
func (e *Exchange) streamConnectionSetup(exch *config.Exchange) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   websocketStreamsURL,
		Connector:             e.WebsocketConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		Handler:               e.wsHandleData,
		GenerateSubscriptions: e.generateSubscriptions,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		// Binance.US accepts 5 messages a second from a connection, pings and pongs included
		RateLimit:     request.NewWeightedRateLimitByDuration(250 * time.Millisecond),
		MessageFilter: asset.Spot,
	}
}

// newOrderbookSync returns the order book synchronisation of the diff depth streams, fetching snapshots with fetch after
// delay and giving up on one after deadline. Binance.US's local order book procedure loads the snapshot, then checks the
// next event to arrive against it, so a book that changes rarely loads from its first event without waiting for another
func (e *Exchange) newOrderbookSync(fetch func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error), delay, deadline time.Duration) *orderbookmanager.UpdateManager {
	return orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		FetchDelay:          delay,
		FetchDeadline:       deadline,
		FetchOrderbook:      fetch,
		CheckPendingUpdate:  checkPendingUpdate,
		Orderbook:           &e.Websocket.Orderbook,
		LoadSnapshotsAtOnce: true,
	})
}

// WebsocketConnect connects to the market data streams
func (e *Exchange) WebsocketConnect(ctx context.Context, conn websocket.Connection) error {
	e.orderbookStreamGeneration.Add(1)
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
	if eventType, err := jsonparser.GetString(respRaw, "e"); err == nil && eventType == serverShutdownEvent {
		return e.handleStreamsServerShutdown(conn, respRaw)
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
		return e.handleStreamsServerShutdown(conn, data)
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

// handleStreamsServerShutdown reconnects the websocket when the stream server warns it is about to shut down. Shutting
// the websocket down closes the WebSocket API connection too
func (e *Exchange) handleStreamsServerShutdown(conn websocket.Connection, event []byte) error {
	return e.handleServerShutdown(event, e.wsAPI.kept(), e.shutdownWebsocketFor(conn))
}

// shutdownWebsocketFor returns the closing a serverShutdown event on conn runs: shutting the websocket down, for the
// connection monitor to connect it again. The monitor connects again only a websocket that is down, or one whose reader
// reports a lost connection, which closing conn alone leaves its reader nothing to report. A websocket shut down
// meanwhile has closed conn, and is left alone in case it has connected again
func (e *Exchange) shutdownWebsocketFor(conn websocket.Connection) func() error {
	return func() error {
		if !conn.IsConnected() {
			return nil
		}
		return e.Websocket.Shutdown()
	}
}

// handleServerShutdown runs closeConnection when a serverShutdown event warns that the server is about to close the
// connection, for the connection monitor to connect again. The server still answers the requests it has, so apiConn,
// the WebSocket API connection the closing takes down, takes no more from the event on, and closeConnection runs once
// those in flight on it have returned or their response limit has passed
func (e *Exchange) handleServerShutdown(event []byte, apiConn websocket.Connection, closeConnection func() error) error {
	var resp WebsocketServerShutdown
	if err := json.Unmarshal(event, &resp); err != nil {
		return err
	}
	log.Warnf(log.WebsocketMgr, "%s server shutting down at %s, reconnecting", e.Name, resp.EventTime.Time())
	wait := e.wsAPI.retire(apiConn)
	// Closing waits for a write in progress and shutting the websocket down for this connection's reader, so neither can
	// run on the reader
	go func() {
		wait()
		if err := closeConnection(); err != nil {
			log.Errorf(log.WebsocketMgr, "%s unable to close the connection for reconnection: %s", e.Name, err)
		}
	}()
	return nil
}

// processTrade relays a Trade Data Stream payload as a trade
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

// processTicker stores an Individual Symbol 24h Change Stream payload and relays it
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

// processKline relays a Candlestick Data Stream payload as a candle
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

// processDepthUpdate passes an Order Book Depth Diff Stream payload of an enabled pair to the order book
// synchronisation
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

// processOrderbookUpdate passes a diff depth event to the order book synchronisation, which applies it to a synchronised
// book only when its first update ID follows the book's last one, queues it while a snapshot is fetched, and otherwise
// invalidates the book and synchronises it again from a new snapshot. Binance.US's procedure ignores an event the book
// has moved past and applies one overlapping it, so an event at or before the last update ID of the book or of the
// events already passed on is dropped, and one overlapping them is passed on from the next update ID: its levels are
// absolute, so the updates already held apply again unchanged. Events queued during a snapshot are then contiguous, as
// the synchronisation requires
func (e *Exchange) processOrderbookUpdate(ctx context.Context, firstUpdateID int64, update *orderbook.Update) error {
	if e.orderbookSync == nil {
		return errOrderbookSyncNotSetUp
	}
	k := key.PairAsset{Base: update.Pair.Base.Item, Quote: update.Pair.Quote.Item, Asset: update.Asset}
	last := e.depthUpdateIDs.last(k)
	if book, err := e.Websocket.Orderbook.LastUpdateID(update.Pair, update.Asset); err == nil {
		last = max(last, book)
	}
	if last > 0 {
		if update.UpdateID <= last {
			return nil
		}
		if firstUpdateID <= last {
			firstUpdateID = last + 1
		}
	}
	err := e.orderbookSync.ProcessOrderbookUpdate(ctx, firstUpdateID, update)
	e.depthUpdateIDs.record(k, update.UpdateID)
	return err
}

// depthUpdateIDs tracks, by pair, the last update ID of the diff depth events passed to the order book synchronisation
type depthUpdateIDs struct {
	mu  sync.Mutex
	ids map[key.PairAsset]int64
}

// last returns the last update ID of a pair's events passed on, or 0 before any
func (d *depthUpdateIDs) last(k key.PairAsset) int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ids[k]
}

// record records the last update ID of a pair's event passed on
func (d *depthUpdateIDs) record(k key.PairAsset, updateID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ids == nil {
		d.ids = make(map[key.PairAsset]int64)
	}
	d.ids[k] = max(d.ids[k], updateID)
}

// fetchOrderbookSnapshot returns the REST order book snapshot the diff depth streams are synchronised with. A snapshot
// fetched while the streams connected again or an order book stream was unsubscribed is discarded, as is one for a pair
// no order book stream maintains, so that the book a stopped stream left is not loaded again
func (e *Exchange) fetchOrderbookSnapshot(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	generation := e.orderbookStreamGeneration.Load()
	resp, err := e.GetOrderBook(ctx, &OrderBookRequest{Symbol: p, Limit: spotOrderbookSnapshotLimit})
	if err != nil {
		return nil, err
	}
	if e.orderbookStreamGeneration.Load() != generation || !e.orderbookStreamed(p, a) {
		return nil, fmt.Errorf("%w: %s %s", errOrderbookStreamStopped, a, p)
	}
	return &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              resp.Bids.Levels(),
		Asks:              resp.Asks.Levels(),
		LastUpdateID:      resp.LastUpdateID,
		// The snapshot carries no time
		LastUpdated: time.Now(),
	}, nil
}

// checkPendingUpdate applies the local order book procedure to diff events buffered while a snapshot was fetched.
// Events whose final update ID u is at or before the snapshot's lastUpdateId are dropped, and the first event applied
// must contain lastUpdateId+1: U <= lastUpdateId+1 <= u. An event starting later means updates between the snapshot and
// the buffered events were missed, so the book needs a new snapshot
func checkPendingUpdate(lastUpdateID, firstUpdateID int64, update *orderbook.Update) (skip bool, err error) {
	if update.UpdateID <= lastUpdateID {
		return true, nil
	}
	if firstUpdateID > lastUpdateID+1 {
		return false, fmt.Errorf("%w: snapshot update ID %d precedes first update ID %d of %s %s", orderbookmanager.ErrOrderbookSnapshotOutdated, lastUpdateID, firstUpdateID, update.Asset, update.Pair)
	}
	return false, nil
}

// processPartialDepth loads a Partial Order Book Depth Stream payload, a complete top of the book, as a snapshot; the
// payload carries no symbol, so it comes from the stream name
func (e *Exchange) processPartialDepth(ctx context.Context, symbol string, data []byte) error {
	pair, enabled, err := e.MatchSymbolCheckEnabled(symbol, asset.Spot, false)
	if err != nil || !enabled {
		return err
	}
	var resp WebsocketPartialDepth
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

// stringToOrderStatus converts an order status
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

// generateSubscriptions returns the stream subscriptions of the enabled pairs. A subscription that cannot be expanded,
// or an order book subscription left out for conflicting with another, is reported as partial generation
func (e *Exchange) generateSubscriptions(_ context.Context) (subscription.List, error) {
	if !e.GetAssetTypes(true).Contains(asset.Spot) {
		return subscription.List{}, nil
	}
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, s := range e.Features.Subscriptions {
		switch s.Asset {
		case asset.Empty, asset.All:
			// Configurations from before subscriptions had assets mean spot, the only asset
			s = s.Clone()
			s.Asset = asset.Spot
		case asset.Spot:
		default:
			continue
		}
		subs = append(subs, s)
	}
	expanded, err := subs.ExpandTemplates(e)
	expanded, conflicts := oneOrderbookSubscription(distinctSubscriptions(expanded))
	if err = common.AppendError(err, conflicts); err != nil {
		return expanded, fmt.Errorf("%w: %w", websocket.ErrSubscriptionPartial, err)
	}
	return expanded, nil
}

// distinctSubscriptions keeps the first of the subscriptions to each stream, as an all-assets subscription and a spot
// one can expand to the same one, which the subscription store refuses to hold twice. Pairs are compared as the store
// compares them, whatever their order, case or delimiter
func distinctSubscriptions(subs subscription.List) subscription.List {
	type stream struct {
		channel, pairs string
		asset          asset.Item
		levels         int
		interval       kline.Interval
	}
	seen := make(common.Seen[stream], len(subs))
	kept := make(subscription.List, 0, len(subs))
	for _, s := range subs {
		pairs := make([]string, len(s.Pairs))
		for i, p := range s.Pairs {
			pairs[i] = p.Base.Upper().String() + "/" + p.Quote.Upper().String()
		}
		slices.Sort(pairs)
		k := stream{channel: s.Channel, pairs: strings.Join(pairs, ","), asset: s.Asset, levels: s.Levels, interval: s.Interval}
		if !seen.CheckAndRecord(k) {
			kept = append(kept, s)
		}
	}
	return kept
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
	}, subscriptionBatchSize)
}

// Unsubscribe unsubscribes from a set of channels
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, channels subscription.List) error {
	return e.ParallelChanOp(ctx, channels, func(ctx context.Context, l subscription.List) error {
		return e.manageSubs(ctx, conn, wsUnsubscribeMethod, l)
	}, subscriptionBatchSize)
}

// manageSubs subscribes or unsubscribes from a list of subscriptions. The books of order book streams are invalidated
// once their unsubscription is answered, goes unanswered or has its wait for the answer ended by ctx, as Binance.US may
// then have stopped them, while an unsubscription Binance.US refused, or that was not sent, leaves them subscribed
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

	req := WebsocketPayloadRequest{
		ID:     e.MessageID(),
		Method: op,
		Params: subs.QualifiedChannels(),
	}
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsConnectionMessageRate, req.ID, req)
	// The connection returns ctx's own error only once it has sent the request, and wraps one ending the wait to send
	unanswered := errors.Is(err, websocket.ErrSignatureTimeout) || err != nil && err == ctx.Err()
	if err == nil {
		if v, d, _, rErr := jsonparser.Get(respRaw, "result"); rErr != nil {
			err = rErr
		} else if d != jsonparser.Null { // null is the only expected and acceptable response
			err = fmt.Errorf("%w: %w: %s", websocket.ErrSubscriptionFailure, common.ErrUnknownError, v)
		}
	}
	if op == wsUnsubscribeMethod && (err == nil || unanswered) {
		e.invalidateUnsubscribedOrderbooks(subs)
	}

	if err != nil {
		err = fmt.Errorf("%w; Channels: %s", err, strings.Join(subs.QualifiedChannels(), ", "))
		switch {
		case op == wsSubscribeMethod:
			if err2 := e.Websocket.RemoveSubscriptions(conn, subs...); err2 != nil {
				err = common.AppendError(err, err2)
			}
		case !unanswered:
			// Binance.US refused the unsubscription, or it was not sent, so the streams carry on
			err = common.AppendError(err, subs.SetStates(subscription.SubscribedState))
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

// orderbookStreamed reports whether an order book stream, subscribed or subscribing, maintains pair's book; a stream's
// first events can arrive before its subscription is answered. Once none does, a book the websocket loads, from a frame
// or snapshot fetch still in flight when its stream stopped, is left to REST snapshots
func (e *Exchange) orderbookStreamed(p currency.Pair, a asset.Item) bool {
	for _, s := range e.Websocket.GetSubscriptions() {
		if s.Channel != subscription.OrderbookChannel || s.Asset != a || !s.Pairs.Contains(p, true) {
			continue
		}
		if state := s.State(); state == subscription.SubscribedState || state == subscription.SubscribingState {
			return true
		}
	}
	return false
}

// invalidateUnsubscribedOrderbooks invalidates the books the order book streams among subs loaded, which nothing updates
// once they are unsubscribed, so REST snapshots replace them rather than defer to them, and counts the unsubscriptions in
// orderbookStreamGeneration, so a snapshot fetch in flight for them is discarded; a book a REST snapshot replaced already
// is left as it is
func (e *Exchange) invalidateUnsubscribedOrderbooks(subs subscription.List) {
	for _, s := range subs {
		if s.Channel != subscription.OrderbookChannel {
			continue
		}
		e.orderbookStreamGeneration.Add(1)
		for _, p := range s.Pairs {
			if depth, err := orderbook.GetDepth(e.Name, p, s.Asset); err == nil {
				depth.InvalidateWebsocketSnapshot(errOrderbookUnsubscribed)
			}
		}
	}
}

// subTplText builds the stream names of a subscription's pairs; symbols are lower case
const subTplText = `
{{- range $pair := index $.AssetPairs $.S.Asset -}}
	{{- fmt $pair }}@{{ channel $.S }}
	{{- $.PairSeparator }}
{{- end -}}
`
