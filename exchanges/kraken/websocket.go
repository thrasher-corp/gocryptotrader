package kraken

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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
)

const (
	wsPublicURL  = "wss://ws.kraken.com/v2"
	wsPrivateURL = "wss://ws-auth.kraken.com/v2"
	wsLevel3URL  = "wss://ws-l3.kraken.com/v2"

	// wsPublicConnection, wsPrivateConnection and wsLevel3Connection are the message filters of the public connection,
	// the authenticated connection and the level 3 order book connection, which takes a token to subscribe
	wsPublicConnection  = "public"
	wsPrivateConnection = "private"
	wsLevel3Connection  = "level3"

	// wsPingDelay keeps a connection active, as Kraken closes connections after about a minute without traffic
	wsPingDelay = 27 * time.Second

	wsMethodSubscribe   = "subscribe"
	wsMethodUnsubscribe = "unsubscribe"
	wsMethodPong        = "pong"

	wsChannelStatus     = "status"
	wsChannelHeartbeat  = "heartbeat"
	wsChannelTicker     = "ticker"
	wsChannelBook       = "book"
	wsChannelOHLC       = "ohlc"
	wsChannelTrade      = "trade"
	wsChannelInstrument = "instrument"
	wsChannelExecutions = "executions"
	wsChannelBalances   = "balances"
	wsChannelLevel3     = "level3"

	wsTypeSnapshot = "snapshot"
)

// wsBookDepths maps the order book depths the book channel takes to the depths sent
var wsBookDepths = map[int]uint64{10: 10, 25: 25, 100: 100, 500: 500, 1000: 1000}

// wsLevel3Depths maps the order book depths the level3 channel takes to the depths sent
var wsLevel3Depths = map[int]uint64{10: 10, 100: 100, 1000: 1000}

// channelNames maps standard channel names to websocket v2 channels. My orders and my trades share the executions
// channel, which reports both, and all orders are the level 3 order book's
var channelNames = map[string]string{
	subscription.TickerChannel:    wsChannelTicker,
	subscription.OrderbookChannel: wsChannelBook,
	subscription.CandlesChannel:   wsChannelOHLC,
	subscription.AllTradesChannel: wsChannelTrade,
	subscription.AllOrdersChannel: wsChannelLevel3,
	subscription.MyOrdersChannel:  wsChannelExecutions,
	subscription.MyTradesChannel:  wsChannelExecutions,
	subscription.MyWalletChannel:  wsChannelBalances,
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 1000},
	{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.MyTradesChannel, Authenticated: true},
}

// connectionSetups returns the setups of the spot connections: the public connection, which streams market data, the
// private connection, which streams account data and takes trading requests, and the level 3 order book connection;
// and of the futures connection, which streams futures market and account data
func (e *Exchange) connectionSetups(exch *config.Exchange) ([]*websocket.ConnectionSetup, error) {
	setups := make([]*websocket.ConnectionSetup, 0, 4)
	for _, c := range []struct {
		filter                 string
		url                    exchange.URL
		connect                func(context.Context, websocket.Connection) error
		generate               func() (subscription.List, error)
		subscribe, unsubscribe func(context.Context, websocket.Connection, subscription.List) error
		handle                 func(context.Context, websocket.Connection, []byte) error
	}{
		{wsPublicConnection, exchange.WebsocketSpot, e.wsConnect, e.generatePublicSubscriptions, e.subscribeForConnection, e.unsubscribeForConnection, e.wsHandleData},
		{wsPrivateConnection, exchange.WebsocketSpotSupplementary, e.wsConnect, e.generatePrivateSubscriptions, e.subscribeForConnection, e.unsubscribeForConnection, e.wsHandleData},
		{wsLevel3Connection, exchange.WebsocketPrivate, e.wsConnect, e.generateLevel3Subscriptions, e.subscribeForConnection, e.unsubscribeForConnection, e.wsHandleData},
		{wsFuturesConnection, exchange.WebsocketFutures, e.wsFuturesConnect, e.generateFuturesSubscriptions, e.futuresSubscribe, e.futuresUnsubscribe, e.wsFuturesHandleData},
	} {
		u, err := e.API.Endpoints.GetURL(c.url)
		if err != nil {
			return nil, err
		}
		setups = append(setups, &websocket.ConnectionSetup{
			URL:                   u,
			MessageFilter:         c.filter,
			Connector:             c.connect,
			GenerateSubscriptions: c.generate,
			Subscriber:            c.subscribe,
			Unsubscriber:          c.unsubscribe,
			Handler:               c.handle,
			RateLimit:             request.NewWeightedRateLimitByDuration(50 * time.Millisecond),
			ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		})
	}
	return setups, nil
}

// wsConnect dials a connection and keeps it alive. The asset names are seeded first when they have not been, since
// websocket v2 names XBT and XDG pairs by their display names
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.assetNames.seeded() {
		if err := e.SeedAssets(ctx); err != nil {
			log.Warnf(log.WebsocketMgr, "%s websocket cannot translate pairs whose display names differ until assets are seeded: %s", e.Name, err)
		}
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Message:     []byte(`{"method":"ping"}`),
		Delay:       wsPingDelay,
		MessageType: gws.TextMessage,
	})
	return nil
}

// GetSubscriptionTemplate returns a subscription channel template, which makes one subscription for each pair as Kraken
// acknowledges each pair of a request on its own, except for a futures feed without products
func (e *Exchange) GetSubscriptionTemplate(*subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{"channelName": channelName, "perPair": perPair}).Parse(subTplText)
}

func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// ValidateSubscriptions rejects order book depths, candle intervals and futures feeds Kraken does not stream, and a
// pair subscribed to at two order book depths or to both its level 2 and level 3 order books, whose updates would be
// applied to one book
func (e *Exchange) ValidateSubscriptions(l subscription.List) error {
	type book struct {
		channel string
		levels  int
	}
	books := make(map[key.PairAsset]book)
	for _, s := range l {
		if s.Asset == asset.Futures {
			if err := validateFuturesSubscription(s); err != nil {
				return err
			}
			continue
		}
		switch s.QualifiedChannel {
		case wsChannelBook, wsChannelLevel3:
			depths, valid := wsBookDepths, "10, 25, 100, 500 or 1000"
			if s.QualifiedChannel == wsChannelLevel3 {
				depths, valid = wsLevel3Depths, "10, 100 or 1000"
			}
			if _, ok := depths[s.Levels]; !ok {
				return fmt.Errorf("%w: %s depth %d is not %s", errInvalidDepth, s, s.Levels, valid)
			}
			for _, p := range s.Pairs {
				k := key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: s.Asset}
				if b, ok := books[k]; ok && b != (book{s.QualifiedChannel, s.Levels}) {
					return fmt.Errorf("%w for %s order books %s %d and %s %d", subscription.ErrExclusiveSubscription, p, b.channel, b.levels, s.QualifiedChannel, s.Levels)
				}
				books[k] = book{s.QualifiedChannel, s.Levels}
			}
		case wsChannelOHLC:
			if _, err := intervalFromMinutes(uint64(time.Duration(s.Interval).Minutes())); err != nil {
				return fmt.Errorf("%s: %w", s, err)
			}
		}
	}
	return nil
}

// validateFuturesSubscription rejects a futures subscription to a feed Kraken does not stream, or to an order book at a
// depth, as the book feed streams whole books
func validateFuturesSubscription(s *subscription.Subscription) error {
	if !slices.Contains(futuresProductFeeds, s.QualifiedChannel) && !slices.Contains(futuresAccountFeeds, s.QualifiedChannel) && s.QualifiedChannel != wsFuturesFeedHeartbeat {
		return fmt.Errorf("%w: %s", errFuturesUnsupportedFeed, s)
	}
	if s.QualifiedChannel == wsFuturesFeedBook && s.Levels != 0 {
		return fmt.Errorf("%w: %s depth %d", errFuturesOrderbookLevels, s, s.Levels)
	}
	return nil
}

// generatePublicSubscriptions returns the public connection's subscriptions
func (e *Exchange) generatePublicSubscriptions() (subscription.List, error) {
	subs, err := e.generateSubscriptions()
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(subs.Public(), func(s *subscription.Subscription) bool {
		return s.Asset == asset.Futures || s.QualifiedChannel == wsChannelLevel3
	}), nil
}

// generateLevel3Subscriptions returns the level 3 order book connection's subscriptions. The level3 channel takes a
// token, so without valid credentials it returns none, and the connection is skipped
func (e *Exchange) generateLevel3Subscriptions() (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() || !e.AreCredentialsValid(context.TODO()) {
		return subscription.List{}, nil
	}
	subs, err := e.generateSubscriptions()
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(subs, func(s *subscription.Subscription) bool { return s.QualifiedChannel != wsChannelLevel3 }), nil
}

// generatePrivateSubscriptions returns the private connection's subscriptions. Without valid credentials it returns
// none, so the private connection is skipped rather than failing every connection. My orders and my trades share the
// executions channel, so they make one subscription, taking the trades snapshot only for my trades
func (e *Exchange) generatePrivateSubscriptions() (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() || !e.AreCredentialsValid(context.TODO()) {
		return subscription.List{}, nil
	}
	subs, err := e.generateSubscriptions()
	if err != nil {
		return nil, err
	}
	var executions *subscription.Subscription
	snapOrders, snapTrades := false, false
	private := subscription.List{}
	for _, s := range subs.Private() {
		if s.Asset == asset.Futures || s.QualifiedChannel == wsChannelLevel3 {
			continue
		}
		if s.QualifiedChannel != wsChannelExecutions {
			private = append(private, s)
			continue
		}
		switch s.Channel {
		case subscription.MyOrdersChannel:
			snapOrders = true
		case subscription.MyTradesChannel:
			snapTrades = true
		}
		if executions == nil || s.Channel == subscription.MyOrdersChannel {
			executions = s
		}
	}
	if executions != nil {
		// A subscription whose channel is already qualified is not copied by template expansion, so it is cloned
		// rather than changed
		executions = executions.Clone()
		executions.Params = map[string]any{"snap_orders": snapOrders, "snap_trades": snapTrades}
		private = append(private, executions)
	}
	return private, nil
}

// subscribeForConnection subscribes a connection to subscriptions
func (e *Exchange) subscribeForConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageSubs(ctx, conn, wsMethodSubscribe, subs)
}

// unsubscribeForConnection unsubscribes a connection from subscriptions
func (e *Exchange) unsubscribeForConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageSubs(ctx, conn, wsMethodUnsubscribe, subs)
}

// manageSubs subscribes or unsubscribes, sending one request for the subscriptions that differ only by pair. Kraken
// answers each symbol of a request on its own, so each subscription succeeds or fails alone
func (e *Exchange) manageSubs(ctx context.Context, conn websocket.Connection, method string, subs subscription.List) error {
	var errs error
	for _, group := range groupSubscriptions(subs) {
		errs = common.AppendError(errs, e.manageSubGroup(ctx, conn, method, group))
	}
	return errs
}

// groupSubscriptions groups subscriptions that can share a request, keeping their order
func groupSubscriptions(subs subscription.List) []subscription.List {
	var groups []subscription.List
	for _, s := range subs {
		found := false
		for i := range groups {
			g := groups[i][0]
			if g.QualifiedChannel == s.QualifiedChannel && g.Levels == s.Levels && g.Interval == s.Interval && len(s.Pairs) != 0 && len(g.Pairs) != 0 {
				groups[i] = append(groups[i], s)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, subscription.List{s})
		}
	}
	return groups
}

// manageSubGroup sends one subscribe or unsubscribe request for subscriptions that differ only by pair, and records
// each subscription the replies acknowledge
func (e *Exchange) manageSubGroup(ctx context.Context, conn websocket.Connection, method string, group subscription.List) error {
	params := &wsSubscriptionParams{Channel: group[0].QualifiedChannel}
	bySymbol := make(map[string]*subscription.Subscription, len(group))
	for _, s := range group {
		for _, p := range s.Pairs {
			symbol := e.displaySymbol(p)
			params.Symbols = append(params.Symbols, symbol)
			bySymbol[symbol] = s
		}
	}
	s := group[0]
	switch params.Channel {
	case wsChannelBook, wsChannelLevel3:
		params.Depth = wsBookDepths[s.Levels]
		if params.Channel == wsChannelLevel3 {
			params.Depth = wsLevel3Depths[s.Levels]
		}
		if method == wsMethodSubscribe {
			// A snapshot can arrive before the subscriber reads its acknowledgement, so the depth it is truncated to
			// is recorded first
			for _, sub := range group {
				for _, p := range sub.Pairs {
					e.setBookDepth(p, sub.Levels)
				}
			}
		}
	case wsChannelOHLC:
		params.Interval = uint64(time.Duration(s.Interval).Minutes())
	case wsChannelExecutions:
		if method == wsMethodSubscribe {
			if v, ok := s.Params["snap_orders"].(bool); ok {
				params.SnapOrders = &v
			}
			if v, ok := s.Params["snap_trades"].(bool); ok {
				params.SnapTrades = &v
			}
		}
	}
	// The level3 channel takes a token as private subscriptions do
	if s.Authenticated || params.Channel == wsChannelLevel3 {
		token, err := e.websocketToken(ctx, method == wsMethodSubscribe)
		if err != nil {
			return e.failSubGroup(method, group, err)
		}
		params.Token = token
	}

	expected := len(params.Symbols)
	if expected == 0 {
		expected = 1
	}
	req := &wsRequest{Method: method, Params: params, RequestID: e.MessageSequence()}
	resps, err := conn.SendMessageReturnResponses(ctx, request.Unset, req.RequestID, req, expected)
	if err != nil {
		return e.failSubGroup(method, group, err)
	}

	pending := make(map[*subscription.Subscription]error, len(group))
	for _, sub := range group {
		pending[sub] = fmt.Errorf("%w: %s", errSubscriptionResponseMissing, sub)
	}
	for _, raw := range resps {
		var resp wsResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return e.failSubGroup(method, group, err)
		}
		var result wsSubscriptionResult
		if len(resp.Result) != 0 {
			if err := json.Unmarshal(resp.Result, &result); err != nil {
				return e.failSubGroup(method, group, err)
			}
		}
		symbol := result.Symbol
		if symbol == "" {
			symbol = resp.Symbol
		}
		sub := s
		if len(bySymbol) != 0 {
			var ok bool
			if sub, ok = bySymbol[symbol]; !ok {
				return common.AppendError(fmt.Errorf("%w: %s", errUnexpectedSubscriptionReply, raw), e.failSubGroup(method, group, nil))
			}
		}
		if !resp.Success {
			pending[sub] = fmt.Errorf("%s %s: %w", method, sub, &APIError{Errors: []string{resp.Error}})
			continue
		}
		// Kraken warns of deprecated fields it still sends, such as the ohlc channel's timestamp, on every subscription
		if e.Verbose {
			for _, warning := range result.Warnings {
				log.Debugf(log.WebsocketMgr, "%s %s %s: %s", e.Name, method, sub, warning)
			}
		}
		pending[sub] = nil
	}

	var errs error
	for _, sub := range group {
		if err := pending[sub]; err != nil {
			errs = common.AppendError(errs, e.failSubGroup(method, subscription.List{sub}, err))
			continue
		}
		if method == wsMethodSubscribe {
			errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(conn, sub))
			continue
		}
		e.clearBookDepths(sub)
		errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(conn, sub))
	}
	return errs
}

// failSubGroup returns err for subscriptions a request failed for, forgetting the order book depths a failed
// subscription recorded
func (e *Exchange) failSubGroup(method string, group subscription.List, err error) error {
	if method == wsMethodSubscribe {
		for _, s := range group {
			e.clearBookDepths(s)
		}
	}
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s %s: %w", method, group, err)
}

// setBookDepth records the depth a pair's order book is subscribed with
func (e *Exchange) setBookDepth(p currency.Pair, depth int) {
	e.bookDepthsMu.Lock()
	defer e.bookDepthsMu.Unlock()
	if e.bookDepths == nil {
		e.bookDepths = make(map[key.PairAsset]int)
	}
	e.bookDepths[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: asset.Spot}] = depth
}

// clearBookDepths forgets the order book depths of an order book subscription's pairs, and the level 3 order books of a
// level 3 subscription's
func (e *Exchange) clearBookDepths(s *subscription.Subscription) {
	if s.QualifiedChannel != wsChannelBook && s.QualifiedChannel != wsChannelLevel3 {
		return
	}
	e.bookDepthsMu.Lock()
	for _, p := range s.Pairs {
		delete(e.bookDepths, key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: asset.Spot})
	}
	e.bookDepthsMu.Unlock()
	if s.QualifiedChannel != wsChannelLevel3 {
		return
	}
	e.level3BooksMu.Lock()
	defer e.level3BooksMu.Unlock()
	for _, p := range s.Pairs {
		delete(e.level3Books, key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: asset.Spot})
	}
}

// bookDepth returns the depth a pair's order book is subscribed with
func (e *Exchange) bookDepth(p currency.Pair) (int, bool) {
	e.bookDepthsMu.Lock()
	defer e.bookDepthsMu.Unlock()
	depth, ok := e.bookDepths[key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: asset.Spot}]
	return depth, ok
}

// websocketToken returns the token private subscriptions and requests carry. refresh fetches a new one, which a
// subscription made on a new connection needs, as a token must be used within 15 minutes of its creation; a token
// stays valid while a connection keeps a private subscription made with it
func (e *Exchange) websocketToken(ctx context.Context, refresh bool) (string, error) {
	if !refresh {
		e.wsTokenMu.RLock()
		token := e.wsToken
		e.wsTokenMu.RUnlock()
		if token != "" {
			return token, nil
		}
	}
	resp, err := e.GetWebsocketToken(ctx)
	if err != nil {
		return "", err
	}
	if resp.Token == "" {
		return "", errWebsocketTokenEmpty
	}
	e.wsTokenMu.Lock()
	e.wsToken = resp.Token
	e.wsTokenMu.Unlock()
	return resp.Token, nil
}

// wsHandleData routes a message from either connection: replies to requests by their request ID, and channel messages
// by their channel
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if method, err := jsonparser.GetUnsafeString(respRaw, "method"); err == nil {
		if reqID, err := jsonparser.GetInt(respRaw, "req_id"); err == nil && conn.IncomingWithData(reqID, respRaw) {
			return nil
		}
		if method == wsMethodPong {
			return nil
		}
		return fmt.Errorf("%w: %s", websocket.ErrSignatureNotMatched, respRaw)
	}
	var msg wsChannelMessage
	if err := json.Unmarshal(respRaw, &msg); err != nil {
		return err
	}
	switch msg.Channel {
	case wsChannelHeartbeat:
		return nil
	case wsChannelStatus:
		return e.wsProcessStatus(ctx, &msg)
	case wsChannelTicker:
		return e.wsProcessTicker(ctx, &msg)
	case wsChannelBook:
		return e.wsProcessBook(ctx, conn, &msg)
	case wsChannelLevel3:
		return e.wsProcessLevel3(ctx, conn, &msg)
	case wsChannelOHLC:
		return e.wsProcessCandles(ctx, &msg)
	case wsChannelTrade:
		return e.wsProcessTrades(ctx, &msg)
	case wsChannelInstrument:
		return e.wsProcessInstruments(ctx, &msg)
	case wsChannelExecutions:
		return e.wsProcessExecutions(ctx, &msg)
	case wsChannelBalances:
		return e.wsProcessBalances(ctx, &msg)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: fmt.Sprintf("%s: %s", websocket.UnhandledMessage, respRaw),
		})
	}
}

// wsProcessStatus relays the trading engine status, returning an error when the engine is not online
func (e *Exchange) wsProcessStatus(ctx context.Context, msg *wsChannelMessage) error {
	var statuses []WsStatus
	if err := json.Unmarshal(msg.Data, &statuses); err != nil {
		return err
	}
	for i := range statuses {
		if err := e.Websocket.DataHandler.Send(ctx, &statuses[i]); err != nil {
			return err
		}
		if statuses[i].System != "online" {
			return fmt.Errorf("%w: %s", errSystemNotOnline, statuses[i].System)
		}
	}
	return nil
}

// wsProcessTicker stores a message's tickers and relays those stored, reporting the others without holding them back
func (e *Exchange) wsProcessTicker(ctx context.Context, msg *wsChannelMessage) error {
	var tickers []WsTicker
	if err := json.Unmarshal(msg.Data, &tickers); err != nil {
		return err
	}
	var errs error
	prices := make([]ticker.Price, 0, len(tickers))
	for i := range tickers {
		pair, err := e.pairFromDisplaySymbol(tickers[i].Symbol)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		prices = append(prices, ticker.Price{
			ExchangeName:               e.Name,
			AssetType:                  asset.Spot,
			Pair:                       pair,
			Last:                       tickers[i].Last,
			Close:                      tickers[i].Last,
			High:                       tickers[i].High,
			Low:                        tickers[i].Low,
			Bid:                        tickers[i].Bid,
			BidSize:                    tickers[i].BidQuantity,
			Ask:                        tickers[i].Ask,
			AskSize:                    tickers[i].AskQuantity,
			BaseVolume:                 tickers[i].Volume,
			VolumeWeightedAveragePrice: tickers[i].VolumeWeightedAveragePrice,
			PercentChange24Hour:        tickers[i].ChangePercentage,
			LastUpdated:                tickers[i].Timestamp,
		})
	}
	processed, err := ticker.ProcessBatch(prices)
	errs = common.AppendError(errs, err)
	if len(processed) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, processed))
}

// wsProcessBook loads order book snapshots and applies updates, checking each against its checksum. A book that fails
// its checksum is resubscribed for a new snapshot
func (e *Exchange) wsProcessBook(ctx context.Context, conn websocket.Connection, msg *wsChannelMessage) error {
	var books []WsBook
	if err := json.Unmarshal(msg.Data, &books); err != nil {
		return err
	}
	for i := range books {
		pair, err := e.pairFromDisplaySymbol(books[i].Symbol)
		if err != nil {
			return err
		}
		if msg.Type == wsTypeSnapshot {
			err = e.wsLoadBookSnapshot(ctx, pair, &books[i])
		} else {
			err = e.wsUpdateBook(ctx, pair, &books[i])
		}
		if errors.Is(err, errChecksumMismatch) {
			e.resubscribeSpotBook(conn, subscription.OrderbookChannel, pair)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Exchange) wsLoadBookSnapshot(ctx context.Context, pair currency.Pair, book *WsBook) error {
	depth, _ := e.bookDepth(pair)
	b := &orderbook.Book{
		Exchange:               e.Name,
		Pair:                   pair,
		Asset:                  asset.Spot,
		Bids:                   wsBookLevels(book.Bids),
		Asks:                   wsBookLevels(book.Asks),
		LastUpdated:            book.Timestamp,
		ValidateOrderbook:      e.ValidateOrderbook,
		MaxDepth:               depth,
		ChecksumStringRequired: true,
	}
	if err := e.Websocket.Orderbook.LoadSnapshot(ctx, b); err != nil {
		return err
	}
	if checksum := bookChecksum(b); checksum != book.Checksum {
		return common.AppendError(fmt.Errorf("%s %w: snapshot expected %d, got %d", pair, errChecksumMismatch, book.Checksum, checksum), e.Websocket.Orderbook.InvalidateOrderbook(pair, asset.Spot))
	}
	return nil
}

func (e *Exchange) wsUpdateBook(ctx context.Context, pair currency.Pair, book *WsBook) error {
	var mismatch bool
	err := e.Websocket.Orderbook.Update(ctx, &orderbook.Update{
		Pair:             pair,
		Asset:            asset.Spot,
		Bids:             wsBookLevels(book.Bids),
		Asks:             wsBookLevels(book.Asks),
		UpdateTime:       book.Timestamp,
		ExpectedChecksum: book.Checksum,
		GenerateChecksum: func(b *orderbook.Book) uint32 {
			checksum := bookChecksum(b)
			mismatch = checksum != book.Checksum
			return checksum
		},
	})
	if err != nil && mismatch {
		return fmt.Errorf("%w: %w", errChecksumMismatch, err)
	}
	return err
}

// resubscribeSpotBook resubscribes a pair's spot order book, of the order book or all orders channel, at the depth it
// is subscribed with
func (e *Exchange) resubscribeSpotBook(conn websocket.Connection, channel string, pair currency.Pair) {
	depth, ok := e.bookDepth(pair)
	if !ok {
		return
	}
	e.resubscribeOrderbook(conn, &subscription.Subscription{Channel: channel, Asset: asset.Spot, Pairs: currency.Pairs{pair}, Levels: depth})
}

// resubscribeOrderbook resubscribes the order book subscription matching s for a new snapshot. It resubscribes in the
// background, since the reply is read by the connection's reader, which is running this
func (e *Exchange) resubscribeOrderbook(conn websocket.Connection, s *subscription.Subscription) {
	sub := e.Websocket.GetSubscription(subscription.ExactKey{Subscription: s})
	if sub == nil {
		return
	}
	go func() {
		if err := e.Websocket.ResubscribeToChannel(context.Background(), conn, sub); err != nil && !errors.Is(err, subscription.ErrInStateAlready) {
			log.Errorf(log.WebsocketMgr, "%s resubscribing to the %s %s order book: %s", e.Name, s.Pairs, s.Asset, err)
		}
	}()
}

// wsProcessLevel3 loads level 3 order book snapshots and applies updates, checking each against its checksum, and
// relays them with their order IDs, which the stored book cannot hold. A book that fails its checksum, or whose update
// names an order it does not hold, is resubscribed for a new snapshot
func (e *Exchange) wsProcessLevel3(ctx context.Context, conn websocket.Connection, msg *wsChannelMessage) error {
	var books []WsLevel3Book
	if err := json.Unmarshal(msg.Data, &books); err != nil {
		return err
	}
	for i := range books {
		pair, err := e.pairFromDisplaySymbol(books[i].Symbol)
		if err != nil {
			return err
		}
		if err := e.wsLevel3Book(ctx, pair, msg.Type == wsTypeSnapshot, &books[i]); err != nil {
			if errors.Is(err, errChecksumMismatch) || errors.Is(err, errLevel3OutOfSync) {
				e.resubscribeSpotBook(conn, subscription.AllOrdersChannel, pair)
			}
			return err
		}
		if err := e.Websocket.DataHandler.Send(ctx, &books[i]); err != nil {
			return err
		}
	}
	return nil
}

// wsLevel3Book applies a level 3 snapshot or update to a pair's book, then loads the whole book, as a book whose levels
// are orders cannot be updated by price
func (e *Exchange) wsLevel3Book(ctx context.Context, pair currency.Pair, snapshot bool, book *WsLevel3Book) error {
	depth, _ := e.bookDepth(pair)
	k := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: asset.Spot}
	e.level3BooksMu.Lock()
	b, ok := e.level3Books[k]
	var err error
	switch {
	case snapshot:
		if e.level3Books == nil {
			e.level3Books = make(map[key.PairAsset]*level3Book)
		}
		b = &level3Book{bids: slices.Clone(book.Bids), asks: slices.Clone(book.Asks)}
		e.level3Books[k] = b
	case !ok:
		err = fmt.Errorf("%s %w: update before a snapshot", pair, errLevel3OutOfSync)
	default:
		err = b.apply(book, depth)
	}
	if err == nil {
		if checksum := b.checksum(); checksum != book.Checksum {
			err = fmt.Errorf("%s %w: expected %d, got %d", pair, errChecksumMismatch, book.Checksum, checksum)
		}
	}
	var bids, asks orderbook.Levels
	if err == nil {
		bids, asks = level3Levels(b.bids), level3Levels(b.asks)
	} else {
		delete(e.level3Books, k)
	}
	e.level3BooksMu.Unlock()
	if err != nil {
		// A book that has not loaded yet has nothing to invalidate
		if invalidErr := e.Websocket.Orderbook.InvalidateOrderbook(pair, asset.Spot); !errors.Is(invalidErr, orderbook.ErrDepthNotFound) {
			err = common.AppendError(err, invalidErr)
		}
		return err
	}
	lastUpdated := book.Timestamp
	if lastUpdated.IsZero() {
		// Kraken documents a timestamp on every message, but its examples carry none, and an order's timestamp is when
		// it was placed, not when it changed
		lastUpdated = time.Now()
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
		Exchange:               e.Name,
		Pair:                   pair,
		Asset:                  asset.Spot,
		Bids:                   bids,
		Asks:                   asks,
		LastUpdated:            lastUpdated,
		PriceDuplication:       true,
		ValidateOrderbook:      e.ValidateOrderbook,
		ChecksumStringRequired: true,
	})
}

// level3Book is a level 3 order book: the orders of each side in the order they queue, from the best price
type level3Book struct {
	bids, asks []WsLevel3Order
}

// apply applies an update's events to each side, then truncates each to depth price levels, as Kraken sends no event
// for the orders of a level leaving the subscribed depth
func (b *level3Book) apply(update *WsLevel3Book, depth int) error {
	var err error
	if b.bids, err = applyLevel3Events(b.bids, update.Bids, func(price, other float64) bool { return price > other }); err != nil {
		return err
	}
	if b.asks, err = applyLevel3Events(b.asks, update.Asks, func(price, other float64) bool { return price < other }); err != nil {
		return err
	}
	b.bids, b.asks = truncateLevel3(b.bids, depth), truncateLevel3(b.asks, depth)
	return nil
}

// applyLevel3Events applies a side's events to its orders. An added order queues behind the orders at its price, and a
// modified order keeps its place unless its price changes
func applyLevel3Events(orders, events []WsLevel3Order, better func(price, other float64) bool) ([]WsLevel3Order, error) {
	for i := range events {
		o := events[i]
		o.Event = ""
		j := slices.IndexFunc(orders, func(held WsLevel3Order) bool { return held.OrderID == o.OrderID })
		switch events[i].Event {
		case "add":
			if j != -1 {
				return nil, fmt.Errorf("%w: order %s added twice", errLevel3OutOfSync, o.OrderID)
			}
			orders = insertLevel3Order(orders, &o, better)
		case "modify":
			if j == -1 {
				return nil, fmt.Errorf("%w: order %s modified but not held", errLevel3OutOfSync, o.OrderID)
			}
			if orders[j].LimitPrice.Float64() == o.LimitPrice.Float64() {
				orders[j] = o
				continue
			}
			orders = insertLevel3Order(slices.Delete(orders, j, j+1), &o, better)
		case "delete":
			if j == -1 {
				return nil, fmt.Errorf("%w: order %s deleted but not held", errLevel3OutOfSync, o.OrderID)
			}
			orders = slices.Delete(orders, j, j+1)
		default:
			return nil, fmt.Errorf("%w: unknown event %q", errLevel3OutOfSync, events[i].Event)
		}
	}
	return orders, nil
}

// insertLevel3Order inserts an order behind the orders at its price and those at better prices
func insertLevel3Order(orders []WsLevel3Order, o *WsLevel3Order, better func(price, other float64) bool) []WsLevel3Order {
	price := o.LimitPrice.Float64()
	i := slices.IndexFunc(orders, func(held WsLevel3Order) bool { return better(price, held.LimitPrice.Float64()) })
	if i == -1 {
		return append(orders, *o)
	}
	return slices.Insert(orders, i, *o)
}

// truncateLevel3 keeps the orders of a side's first depth price levels, keeping every order when depth is zero
func truncateLevel3(orders []WsLevel3Order, depth int) []WsLevel3Order {
	if depth <= 0 {
		return orders
	}
	levels := 0
	for i := range orders {
		if i == 0 || orders[i].LimitPrice.Float64() != orders[i-1].LimitPrice.Float64() {
			levels++
			if levels > depth {
				return orders[:i]
			}
		}
	}
	return orders
}

// checksum returns the CRC32 of the orders of the book's top 10 ask price levels from the lowest and top 10 bid price
// levels from the highest, each order written as its price and quantity without their decimal points and leading zeros
func (b *level3Book) checksum() uint32 {
	var s strings.Builder
	for _, orders := range [][]WsLevel3Order{truncateLevel3(b.asks, 10), truncateLevel3(b.bids, 10)} {
		for i := range orders {
			s.WriteString(checksumDigits(orders[i].LimitPrice.String()))
			s.WriteString(checksumDigits(orders[i].Quantity.String()))
		}
	}
	return crc32.ChecksumIEEE([]byte(s.String()))
}

// level3Levels converts a side's orders to order book levels, one for each order, keeping the digits Kraken sent
func level3Levels(orders []WsLevel3Order) orderbook.Levels {
	l := make(orderbook.Levels, len(orders))
	for i := range orders {
		l[i] = orderbook.Level{
			Price:     orders[i].LimitPrice.Float64(),
			StrPrice:  orders[i].LimitPrice.String(),
			Amount:    orders[i].Quantity.Float64(),
			StrAmount: orders[i].Quantity.String(),
		}
	}
	return l
}

// wsBookLevels converts order book levels, keeping the digits Kraken sent for the checksum
func wsBookLevels(levels []WsBookLevel) orderbook.Levels {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		l[i] = orderbook.Level{
			Price:     levels[i].Price.Float64(),
			StrPrice:  levels[i].Price.String(),
			Amount:    levels[i].Quantity.Float64(),
			StrAmount: levels[i].Quantity.String(),
		}
	}
	return l
}

// bookChecksum returns the CRC32 of a book's top 10 asks from the lowest and top 10 bids from the highest, each level
// written as its price and quantity without their decimal points and leading zeros
func bookChecksum(b *orderbook.Book) uint32 {
	var s strings.Builder
	for i := 0; i < 10 && i < len(b.Asks); i++ {
		s.WriteString(checksumDigits(b.Asks[i].StrPrice))
		s.WriteString(checksumDigits(b.Asks[i].StrAmount))
	}
	for i := 0; i < 10 && i < len(b.Bids); i++ {
		s.WriteString(checksumDigits(b.Bids[i].StrPrice))
		s.WriteString(checksumDigits(b.Bids[i].StrAmount))
	}
	return crc32.ChecksumIEEE([]byte(s.String()))
}

// checksumDigits removes a number's decimal point and leading zeros
func checksumDigits(n string) string {
	return strings.TrimLeft(strings.Replace(n, ".", "", 1), "0")
}

// wsProcessCandles relays candles, marking those whose interval had not ended when the message was sent as partial
func (e *Exchange) wsProcessCandles(ctx context.Context, msg *wsChannelMessage) error {
	var candles []WsCandle
	if err := json.Unmarshal(msg.Data, &candles); err != nil {
		return err
	}
	for i := range candles {
		pair, err := e.pairFromDisplaySymbol(candles[i].Symbol)
		if err != nil {
			return err
		}
		interval, err := intervalFromMinutes(candles[i].IntervalMinutes)
		if err != nil {
			return err
		}
		candle := kline.Candle{
			Time:   candles[i].IntervalBegin,
			Open:   candles[i].Open,
			High:   candles[i].High,
			Low:    candles[i].Low,
			Close:  candles[i].Close,
			Volume: candles[i].Volume,
		}
		if candles[i].IntervalBegin.Add(interval.Duration()).After(msg.Timestamp) {
			candle.ValidationIssues = kline.PartialCandle
		}
		if err := e.Websocket.DataHandler.Send(ctx, kline.Item{
			Exchange: e.Name,
			Pair:     pair,
			Asset:    asset.Spot,
			Interval: interval,
			Candles:  []kline.Candle{candle},
		}); err != nil {
			return err
		}
	}
	return nil
}

// wsProcessTrades relays and saves public trades, as the trade feed and trade saving settings ask
func (e *Exchange) wsProcessTrades(_ context.Context, msg *wsChannelMessage) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var wsTrades []WsTrade
	if err := json.Unmarshal(msg.Data, &wsTrades); err != nil {
		return err
	}
	trades := make([]trade.Data, len(wsTrades))
	for i := range wsTrades {
		pair, err := e.pairFromDisplaySymbol(wsTrades[i].Symbol)
		if err != nil {
			return err
		}
		side, err := order.StringToOrderSide(wsTrades[i].Side)
		if err != nil {
			return err
		}
		trades[i] = trade.Data{
			Exchange:     e.Name,
			CurrencyPair: pair,
			AssetType:    asset.Spot,
			TID:          formatTradeID(wsTrades[i].TradeID),
			Side:         side,
			Price:        wsTrades[i].Price,
			Amount:       wsTrades[i].Quantity,
			Timestamp:    wsTrades[i].Timestamp,
		}
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

// wsProcessInstruments relays the instrument channel's reference data
func (e *Exchange) wsProcessInstruments(ctx context.Context, msg *wsChannelMessage) error {
	var instruments WsInstruments
	if err := json.Unmarshal(msg.Data, &instruments); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &instruments)
}

// wsProcessExecutions relays order updates, with a trade event's fill as the order's trade, and the fills themselves
// when the fills feed is enabled
func (e *Exchange) wsProcessExecutions(ctx context.Context, msg *wsChannelMessage) error {
	var executions []WsExecution
	if err := json.Unmarshal(msg.Data, &executions); err != nil {
		return err
	}
	var fills []fill.Data
	for i := range executions {
		d, err := e.executionOrderDetail(&executions[i])
		if err != nil {
			return err
		}
		if err := e.Websocket.DataHandler.Send(ctx, d); err != nil {
			return err
		}
		if executions[i].ExecutionType == "trade" && e.IsFillsFeedEnabled() {
			fills = append(fills, fill.Data{
				ID:            executions[i].ExecutionID,
				Timestamp:     executions[i].Timestamp,
				Exchange:      e.Name,
				AssetType:     asset.Spot,
				CurrencyPair:  d.Pair,
				Side:          d.Side,
				OrderID:       executions[i].OrderID,
				ClientOrderID: executions[i].ClientOrderID,
				TradeID:       formatTradeID(executions[i].TradeID),
				Price:         executions[i].LastPrice,
				Amount:        executions[i].LastQuantity,
			})
		}
	}
	if len(fills) == 0 {
		return nil
	}
	return e.Websocket.Fills.Update(fills...)
}

// executionOrderDetail converts an execution report to an order update. Kraken sends only the changed fields on most
// events, so unset fields are left zero for the order store to keep what it holds
func (e *Exchange) executionOrderDetail(x *WsExecution) (*order.Detail, error) {
	d := &order.Detail{
		Exchange:             e.Name,
		AssetType:            asset.Spot,
		OrderID:              x.OrderID,
		ClientOrderID:        x.ClientOrderID,
		Amount:               x.OrderQuantity,
		ExecutedAmount:       x.CumulativeQuantity,
		AverageExecutedPrice: x.AveragePrice,
		Cost:                 x.CumulativeCost,
		LastUpdated:          x.Timestamp,
		ReduceOnly:           x.ReduceOnly,
		TriggerPrice:         x.Triggers.Price,
	}
	if x.Symbol != "" {
		pair, err := e.pairFromDisplaySymbol(x.Symbol)
		if err != nil {
			return nil, err
		}
		d.Pair = pair
	}
	if x.Side != "" {
		side, err := order.StringToOrderSide(x.Side)
		if err != nil {
			return nil, err
		}
		d.Side = side
	}
	if x.OrderType != "" {
		orderType, err := orderTypeFromString(x.OrderType)
		if err != nil {
			return nil, err
		}
		d.Type = orderType
	}
	if x.OrderStatus != "" {
		status, err := orderStatusFromString(x.OrderStatus)
		if err != nil {
			return nil, err
		}
		d.Status = status
	}
	if x.TimeInForce != "" {
		tif, err := timeInForceFromString(x.TimeInForce)
		if err != nil {
			return nil, err
		}
		d.TimeInForce = tif
	}
	if x.PostOnly {
		d.TimeInForce |= order.PostOnly
	}
	if x.LimitPrice != 0 {
		d.Price = x.LimitPrice
	}
	if x.ExecutionType == "pending_new" {
		d.Date = x.Timestamp
	}
	if x.ExecutionType == "trade" {
		// Fees are the trade event's, which Kraken charges in the quote currency, so they are the trade's rather than
		// the order's
		t := order.TradeHistory{
			TID:       x.ExecutionID,
			Price:     x.LastPrice,
			Amount:    x.LastQuantity,
			Exchange:  e.Name,
			Type:      d.Type,
			Side:      d.Side,
			IsMaker:   x.LiquidityIndicator == "m",
			Timestamp: x.Timestamp,
			Total:     x.Cost,
		}
		for i := range x.Fees {
			t.Fee += x.Fees[i].Quantity
			t.FeeAsset = x.Fees[i].Asset
			if alt, ok := e.assetNames.alternativeName(t.FeeAsset); ok {
				t.FeeAsset = alt
			}
		}
		d.Trades = []order.TradeHistory{t}
	}
	return d, nil
}

// wsProcessBalances stores and relays a snapshot's spot wallet balances. Updates are ledger transactions whose balance
// spans every wallet, so they are relayed without changing the stored spot balances
func (e *Exchange) wsProcessBalances(ctx context.Context, msg *wsChannelMessage) error {
	if msg.Type != wsTypeSnapshot {
		var entries []WsLedgerEntry
		if err := json.Unmarshal(msg.Data, &entries); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, entries)
	}
	var balances []WsBalance
	if err := json.Unmarshal(msg.Data, &balances); err != nil {
		return err
	}
	subAcct := accounts.NewSubAccount(asset.Spot, "")
	for i := range balances {
		var spot float64
		for _, w := range balances[i].Wallets {
			if w.Type == "spot" {
				spot += w.Balance
			}
		}
		name := balances[i].Asset
		if alt, ok := e.assetNames.alternativeName(name); ok {
			name = alt
		}
		code := currency.NewCode(name)
		balance, err := e.Accounts.UpdateBalance(ctx, "", asset.Spot, code, func(b *accounts.Balance) {
			// Websocket balances carry no amount on hold, so the stored hold is kept and the free amount follows from it
			b.Total = spot
			b.Free = max(b.Total-b.Hold, 0)
			b.AvailableWithoutBorrow = b.Free
		})
		if err != nil {
			return err
		}
		subAcct.Balances.Set(code, balance)
	}
	return e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAcct})
}

/*
One subscription for each pair, because:
  - Kraken acknowledges each pair of a request on its own
  - resubscribing a book needs subscriptions that match its acknowledgements
  - FlushChannels and GetChannelDiff would otherwise resubscribe existing subscriptions

A futures feed without products is subscribed to once.
*/
const subTplText = `
{{- if $.S.Asset -}}
	{{ range $asset, $pairs := $.AssetPairs }}
		{{- if perPair $.S $asset -}}
			{{- range $p := $pairs  -}}
				{{- channelName $.S $asset }}
				{{- $.PairSeparator }}
			{{- end -}}
		{{- else -}}
			{{- channelName $.S $asset }}
		{{- end -}}
		{{ $.AssetSeparator }}
	{{- end -}}
{{- else -}}
	{{- channelName $.S $.S.Asset }}
{{- end }}
`

// intervalFromMinutes returns the candle interval Kraken names by its minutes
func intervalFromMinutes(minutes uint64) (kline.Interval, error) {
	for _, i := range spotIntervals {
		if i.Duration().Minutes() == float64(minutes) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%w: %d minutes", kline.ErrUnsupportedInterval, minutes)
}

// channelName converts a standard channel name to an asset's channel: a websocket v2 channel, or a futures feed
func channelName(s *subscription.Subscription, a asset.Item) string {
	names := channelNames
	if a == asset.Futures {
		names = futuresChannelNames
	}
	if n, ok := names[s.Channel]; ok {
		return n
	}
	return s.Channel
}

// perPair reports whether a subscription is made for each pair: any but a futures feed without products
func perPair(s *subscription.Subscription, a asset.Item) bool {
	return a != asset.Futures || slices.Contains(futuresProductFeeds, channelName(s, a))
}

// formatTradeID formats a numeric trade ID
func formatTradeID(id uint64) string {
	return strconv.FormatUint(id, 10)
}
