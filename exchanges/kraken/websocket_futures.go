package kraken

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	wsFuturesURL = "wss://futures.kraken.com/ws/v1"

	// wsFuturesConnection is the message filter of the futures connection, which carries market and account feeds
	wsFuturesConnection = "futures"

	// wsFuturesRequestSignature matches replies to requests. Kraken answers in the order requested without a request
	// ID, so requests are sent one at a time
	wsFuturesRequestSignature = "futures request"

	wsFuturesEventInfo         = "info"
	wsFuturesEventChallenge    = "challenge"
	wsFuturesEventSubscribe    = "subscribe"
	wsFuturesEventSubscribed   = "subscribed"
	wsFuturesEventUnsubscribe  = "unsubscribe"
	wsFuturesEventUnsubscribed = "unsubscribed"
	wsFuturesEventAlert        = "alert"

	// wsFuturesAlreadySubscribed and wsFuturesNotSubscribed alert a request whose subscription is already as asked
	wsFuturesAlreadySubscribed = "Already subscribed to feed, re-requesting"
	wsFuturesNotSubscribed     = "Not subscribed to feed"

	wsFuturesFeedHeartbeat          = "heartbeat"
	wsFuturesFeedTicker             = "ticker"
	wsFuturesFeedBook               = "book"
	wsFuturesFeedBookSnapshot       = "book_snapshot"
	wsFuturesFeedTrade              = "trade"
	wsFuturesFeedTradeSnapshot      = "trade_snapshot"
	wsFuturesFeedOpenOrders         = "open_orders"
	wsFuturesFeedOpenOrdersSnapshot = "open_orders_snapshot"
	wsFuturesFeedFills              = "fills"
	wsFuturesFeedFillsSnapshot      = "fills_snapshot"
	wsFuturesFeedBalances           = "balances"
	wsFuturesFeedBalancesSnapshot   = "balances_snapshot"
	wsFuturesFeedOpenPositions      = "open_positions"
	wsFuturesFeedNotifications      = "notifications_auth"
)

// futuresChannelNames maps standard channel names to futures feeds. A feed without a standard channel, such as
// open_positions, is subscribed to by its name
var futuresChannelNames = map[string]string{
	subscription.TickerChannel:    wsFuturesFeedTicker,
	subscription.OrderbookChannel: wsFuturesFeedBook,
	subscription.AllTradesChannel: wsFuturesFeedTrade,
	subscription.MyOrdersChannel:  wsFuturesFeedOpenOrders,
	subscription.MyTradesChannel:  wsFuturesFeedFills,
	subscription.MyWalletChannel:  wsFuturesFeedBalances,
	subscription.HeartbeatChannel: wsFuturesFeedHeartbeat,
}

// futuresProductFeeds are the market feeds, which are subscribed to for each product. heartbeat takes no product
var futuresProductFeeds = []string{wsFuturesFeedTicker, wsFuturesFeedBook, wsFuturesFeedTrade}

// futuresAccountFeeds are the account's feeds, which take a signed challenge and no products
var futuresAccountFeeds = []string{wsFuturesFeedOpenOrders, wsFuturesFeedFills, wsFuturesFeedBalances, wsFuturesFeedOpenPositions, wsFuturesFeedNotifications}

// futuresChallenge is a challenge Kraken issued on a connection, with its signature
type futuresChallenge struct {
	conn     websocket.Connection
	original string
	signed   string
}

// wsFuturesConnect dials the futures connection and pings it, as Kraken closes a connection not pinged within a minute
func (e *Exchange) wsFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.assetNames.seeded() {
		if err := e.SeedAssets(ctx); err != nil {
			log.Warnf(log.WebsocketMgr, "%s futures websocket cannot translate fee currencies until assets are seeded: %s", e.Name, err)
		}
	}
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.PingMessage,
		Delay:       wsPingDelay,
	})
	return nil
}

// generateFuturesSubscriptions returns the futures connection's subscriptions. Account feeds are left out without
// valid credentials. A feed without products is subscribed to once, so its subscription is cloned without the pairs
// templating gave it
func (e *Exchange) generateFuturesSubscriptions() (subscription.List, error) {
	subs, err := e.generateSubscriptions()
	if err != nil {
		return nil, err
	}
	authenticated := e.Websocket.CanUseAuthenticatedEndpoints() && e.AreCredentialsValid(context.TODO())
	futures := subscription.List{}
	for _, s := range subs {
		switch {
		case s.Asset != asset.Futures:
		case slices.Contains(futuresProductFeeds, s.QualifiedChannel):
			futures = append(futures, s)
		case slices.Contains(futuresAccountFeeds, s.QualifiedChannel) && !authenticated:
		default:
			s = s.Clone()
			s.Pairs = nil
			futures = append(futures, s)
		}
	}
	return futures, nil
}

// futuresSubscribe subscribes the futures connection to subscriptions
func (e *Exchange) futuresSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageFuturesSubs(ctx, conn, wsFuturesEventSubscribe, subs)
}

// futuresUnsubscribe unsubscribes the futures connection from subscriptions
func (e *Exchange) futuresUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageFuturesSubs(ctx, conn, wsFuturesEventUnsubscribe, subs)
}

// manageFuturesSubs subscribes or unsubscribes, sending one request for the subscriptions that differ only by product
func (e *Exchange) manageFuturesSubs(ctx context.Context, conn websocket.Connection, event string, subs subscription.List) error {
	var errs error
	for _, group := range groupSubscriptions(subs) {
		errs = common.AppendError(errs, e.manageFuturesSubGroup(ctx, conn, event, group))
	}
	return errs
}

// manageFuturesSubGroup sends one subscribe or unsubscribe request for subscriptions that differ only by product, and
// records each subscription its reply acknowledges. Kraken replies once for each product, or once for a feed without
// products, in the order requested, naming the product only in an acknowledgement
func (e *Exchange) manageFuturesSubGroup(ctx context.Context, conn websocket.Connection, event string, group subscription.List) error {
	req := &wsFuturesRequest{Event: event, Feed: group[0].QualifiedChannel}
	var answered []*subscription.Subscription
	for _, s := range group {
		for _, p := range s.Pairs {
			id, err := e.futuresMarketSymbol(p)
			if err != nil {
				return fmt.Errorf("%s %s: %w", event, s, err)
			}
			req.ProductIDs = append(req.ProductIDs, id)
			answered = append(answered, s)
		}
	}
	if len(answered) == 0 {
		answered = group[:1]
	}
	if slices.Contains(futuresAccountFeeds, req.Feed) {
		if err := e.signFuturesRequest(ctx, conn, req); err != nil {
			return fmt.Errorf("%s %s: %w", event, group, err)
		}
	}
	resps, err := e.futuresRequest(ctx, conn, req, len(answered))
	if err != nil {
		return fmt.Errorf("%s %s: %w", event, group, err)
	}

	outcomes := make(map[*subscription.Subscription]error, len(group))
	for _, s := range group {
		outcomes[s] = fmt.Errorf("%w: %s", errSubscriptionResponseMissing, s)
	}
	acknowledged, alreadyAsAsked := wsFuturesEventSubscribed, wsFuturesAlreadySubscribed
	if event == wsFuturesEventUnsubscribe {
		acknowledged, alreadyAsAsked = wsFuturesEventUnsubscribed, wsFuturesNotSubscribed
	}
	for i, raw := range resps {
		var reply wsFuturesEvent
		if err := json.Unmarshal(raw, &reply); err != nil {
			return fmt.Errorf("%s %s: %w", event, group, err)
		}
		if i >= len(answered) || len(reply.ProductIDs) > 1 || len(reply.ProductIDs) == 1 && (i >= len(req.ProductIDs) || reply.ProductIDs[0] != req.ProductIDs[i]) {
			return fmt.Errorf("%s %s: %w: %s", event, group, errFuturesUnexpectedReply, raw)
		}
		s := answered[i]
		switch {
		case reply.Event == acknowledged, reply.Event == wsFuturesEventAlert && reply.Message == alreadyAsAsked:
			outcomes[s] = nil
		default:
			outcomes[s] = fmt.Errorf("%s %s: %w: %s %s", event, s, errFuturesSubscriptionFailed, reply.Event, reply.Message)
		}
	}

	var errs error
	for _, s := range group {
		if err := outcomes[s]; err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if event == wsFuturesEventSubscribe {
			errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(conn, s))
			continue
		}
		errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(conn, s))
	}
	return errs
}

// signFuturesRequest adds the API key and the connection's signed challenge to a request for an account feed
func (e *Exchange) signFuturesRequest(ctx context.Context, conn websocket.Connection, req *wsFuturesRequest) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.futuresChallengeMu.Lock()
	defer e.futuresChallengeMu.Unlock()
	if e.futuresChallenge.conn != conn || e.futuresChallenge.original == "" {
		resps, err := e.futuresRequest(ctx, conn, &wsFuturesRequest{Event: wsFuturesEventChallenge, APIKey: creds.Key}, 1)
		if err != nil {
			return err
		}
		var reply wsFuturesEvent
		if err := json.Unmarshal(resps[0], &reply); err != nil {
			return err
		}
		if reply.Event != wsFuturesEventChallenge {
			return fmt.Errorf("%w: %s %s", errFuturesSubscriptionFailed, reply.Event, reply.Message)
		}
		if reply.Message == "" {
			return errFuturesChallengeEmpty
		}
		signed, err := futuresChallengeSignature(creds.Secret, reply.Message)
		if err != nil {
			return err
		}
		e.futuresChallenge = futuresChallenge{conn: conn, original: reply.Message, signed: signed}
	}
	req.APIKey, req.OriginalChallenge, req.SignedChallenge = creds.Key, e.futuresChallenge.original, e.futuresChallenge.signed
	return nil
}

// futuresRequest sends a request on the futures connection and returns its replies
func (e *Exchange) futuresRequest(ctx context.Context, conn websocket.Connection, req *wsFuturesRequest, expected int) ([][]byte, error) {
	e.futuresRequestMu.Lock()
	defer e.futuresRequestMu.Unlock()
	return conn.SendMessageReturnResponses(ctx, request.Unset, wsFuturesRequestSignature, req, expected)
}

// wsFuturesHandleData routes a futures message: events to the request they answer, and feed messages by their feed
func (e *Exchange) wsFuturesHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if event, err := jsonparser.GetUnsafeString(respRaw, "event"); err == nil {
		if event == wsFuturesEventInfo || conn.IncomingWithData(wsFuturesRequestSignature, respRaw) {
			return nil
		}
		return fmt.Errorf("%w: %s", errFuturesUnexpectedReply, respRaw)
	}
	feed, err := jsonparser.GetUnsafeString(respRaw, "feed")
	if err != nil {
		return fmt.Errorf("%w: %s", websocket.ErrSignatureNotMatched, respRaw)
	}
	switch feed {
	case wsFuturesFeedHeartbeat:
		return nil
	case wsFuturesFeedTicker:
		return e.wsFuturesProcessTicker(ctx, respRaw)
	case wsFuturesFeedBookSnapshot:
		return e.wsFuturesLoadBookSnapshot(ctx, respRaw)
	case wsFuturesFeedBook:
		return e.wsFuturesUpdateBook(ctx, conn, respRaw)
	case wsFuturesFeedTradeSnapshot, wsFuturesFeedTrade:
		return e.wsFuturesProcessTrades(feed, respRaw)
	case wsFuturesFeedOpenOrdersSnapshot, wsFuturesFeedOpenOrders:
		return e.wsFuturesProcessOpenOrders(ctx, feed, respRaw)
	case wsFuturesFeedFillsSnapshot, wsFuturesFeedFills:
		return e.wsFuturesProcessFills(ctx, respRaw)
	case wsFuturesFeedBalancesSnapshot, wsFuturesFeedBalances:
		return e.wsFuturesProcessBalances(ctx, respRaw)
	case wsFuturesFeedOpenPositions:
		var positions WsFuturesPositions
		if err := json.Unmarshal(respRaw, &positions); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &positions)
	case wsFuturesFeedNotifications:
		var notifications WsFuturesNotifications
		if err := json.Unmarshal(respRaw, &notifications); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &notifications)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: fmt.Sprintf("%s: %s", websocket.UnhandledMessage, respRaw),
		})
	}
}

// futuresPair returns the available futures pair a product ID names
func (e *Exchange) futuresPair(productID string) (currency.Pair, error) {
	return e.MatchSymbolWithAvailablePairs(productID, asset.Futures, true)
}

// wsFuturesProcessTicker stores and relays a futures ticker
func (e *Exchange) wsFuturesProcessTicker(ctx context.Context, respRaw []byte) error {
	var t WsFuturesTicker
	if err := json.Unmarshal(respRaw, &t); err != nil {
		return err
	}
	pair, err := e.futuresPair(t.ProductID)
	if err != nil {
		return err
	}
	baseVolume, quoteVolume := futuresTickerVolumes(t.ProductID, t.Volume, t.VolumeQuote)
	tick := &ticker.Price{
		Last:                t.Last,
		Bid:                 t.Bid,
		BidSize:             t.BidSize,
		Ask:                 t.Ask,
		AskSize:             t.AskSize,
		BaseVolume:          baseVolume,
		QuoteVolume:         quoteVolume,
		Open:                t.Open24Hour,
		High:                t.High24Hour,
		Low:                 t.Low24Hour,
		PercentChange24Hour: t.Change,
		OpenInterest:        t.OpenInterest,
		MarkPrice:           t.MarkPrice,
		IndexPrice:          t.Index,
		Pair:                pair,
		ExchangeName:        e.Name,
		AssetType:           asset.Futures,
		LastUpdated:         t.Time.Time(),
	}
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// wsFuturesLoadBookSnapshot loads a futures order book, which updates follow from its sequence number
func (e *Exchange) wsFuturesLoadBookSnapshot(ctx context.Context, respRaw []byte) error {
	var snapshot WsFuturesBookSnapshot
	if err := json.Unmarshal(respRaw, &snapshot); err != nil {
		return err
	}
	pair, err := e.futuresPair(snapshot.ProductID)
	if err != nil {
		return err
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             asset.Futures,
		Bids:              futuresBookLevels(snapshot.Bids),
		Asks:              futuresBookLevels(snapshot.Asks),
		LastUpdated:       snapshot.Timestamp.Time(),
		LastUpdateID:      int64(snapshot.Sequence), //nolint:gosec // Kraken's sequence numbers are far below the int64 limit
		ValidateOrderbook: e.ValidateOrderbook,
	})
}

// wsFuturesUpdateBook applies an update to a futures order book. An update out of sequence, or one the book rejects,
// leaves the book invalid until resubscribing brings a new snapshot, so updates until then are dropped
func (e *Exchange) wsFuturesUpdateBook(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var u WsFuturesBookUpdate
	if err := json.Unmarshal(respRaw, &u); err != nil {
		return err
	}
	pair, err := e.futuresPair(u.ProductID)
	if err != nil {
		return err
	}
	last, err := e.Websocket.Orderbook.LastUpdateID(pair, asset.Futures)
	if err != nil {
		if errors.Is(err, orderbook.ErrDepthNotFound) {
			return err
		}
		return nil
	}
	sequence := int64(u.Sequence) //nolint:gosec // Kraken's sequence numbers are far below the int64 limit
	if sequence != last+1 {
		err := fmt.Errorf("%s %w: expected %d, got %d", pair, errFuturesSequenceGap, last+1, sequence)
		e.resubscribeOrderbook(conn, &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{pair}})
		return common.AppendError(err, e.Websocket.Orderbook.InvalidateOrderbook(pair, asset.Futures))
	}
	update := &orderbook.Update{
		Pair:       pair,
		Asset:      asset.Futures,
		UpdateID:   sequence,
		UpdateTime: u.Timestamp.Time(),
	}
	level := orderbook.Levels{{Price: u.Price, Amount: u.Quantity}}
	if u.Side == "buy" {
		update.Bids = level
	} else {
		update.Asks = level
	}
	if err := e.Websocket.Orderbook.Update(ctx, update); err != nil {
		// An update is applied before the book is relayed, so a book that could not be relayed is still valid
		if errors.Is(err, orderbook.ErrOrderbookInvalid) {
			e.resubscribeOrderbook(conn, &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{pair}})
		}
		return err
	}
	return nil
}

// futuresBookLevels converts futures order book levels
func futuresBookLevels(levels []WsFuturesBookLevel) orderbook.Levels {
	l := make(orderbook.Levels, len(levels))
	for i := range levels {
		l[i] = orderbook.Level{Price: levels[i].Price, Amount: levels[i].Quantity}
	}
	return l
}

// wsFuturesProcessTrades relays and saves futures trades, as the trade feed and trade saving settings ask
func (e *Exchange) wsFuturesProcessTrades(feed string, respRaw []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	if !saveTradeData && !e.IsTradeFeedEnabled() {
		return nil
	}
	var wsTrades []WsFuturesTrade
	if feed == wsFuturesFeedTradeSnapshot {
		var snapshot WsFuturesTradeSnapshot
		if err := json.Unmarshal(respRaw, &snapshot); err != nil {
			return err
		}
		wsTrades = snapshot.Trades
	} else {
		var t WsFuturesTrade
		if err := json.Unmarshal(respRaw, &t); err != nil {
			return err
		}
		wsTrades = []WsFuturesTrade{t}
	}
	trades := make([]trade.Data, len(wsTrades))
	for i := range wsTrades {
		pair, err := e.futuresPair(wsTrades[i].ProductID)
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
			AssetType:    asset.Futures,
			TID:          wsTrades[i].UID,
			Side:         side,
			Price:        wsTrades[i].Price,
			Amount:       wsTrades[i].Quantity,
			Timestamp:    wsTrades[i].Time.Time(),
		}
	}
	return e.Websocket.Trade.Update(saveTradeData, trades...)
}

// wsFuturesProcessOpenOrders relays the account's open futures orders and the changes to them
func (e *Exchange) wsFuturesProcessOpenOrders(ctx context.Context, feed string, respRaw []byte) error {
	if feed == wsFuturesFeedOpenOrdersSnapshot {
		var snapshot WsFuturesOpenOrdersSnapshot
		if err := json.Unmarshal(respRaw, &snapshot); err != nil {
			return err
		}
		for i := range snapshot.Orders {
			d, err := e.futuresWsOrderDetail(&snapshot.Orders[i], futuresOrderUpdateStatus("", false, snapshot.Orders[i].Filled))
			if err != nil {
				return err
			}
			if err := e.Websocket.DataHandler.Send(ctx, d); err != nil {
				return err
			}
		}
		return nil
	}
	var u WsFuturesOpenOrderUpdate
	if err := json.Unmarshal(respRaw, &u); err != nil {
		return err
	}
	if u.Order == nil {
		// Kraken names only the order when it is cancelled
		return e.Websocket.DataHandler.Send(ctx, &order.Detail{
			Exchange:  e.Name,
			AssetType: asset.Futures,
			OrderID:   u.OrderID,
			Status:    futuresOrderUpdateStatus(u.Reason, u.IsCancel, 0),
		})
	}
	d, err := e.futuresWsOrderDetail(u.Order, futuresOrderUpdateStatus(u.Reason, u.IsCancel, u.Order.Filled))
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, d)
}

// futuresWsOrderDetail converts an open futures order
func (e *Exchange) futuresWsOrderDetail(o *WsFuturesOpenOrder, status order.Status) (*order.Detail, error) {
	pair, err := e.futuresPair(o.Instrument)
	if err != nil {
		return nil, err
	}
	orderType, tif, err := futuresOrderTypeFromString(o.Type)
	if err != nil {
		return nil, err
	}
	// A stop or take profit order with a limit price is a limit order once it triggers
	if (orderType == order.Stop || orderType == order.TakeProfit) && o.LimitPrice > 0 {
		orderType |= order.Limit
	}
	side := order.Buy
	if o.Direction == 1 {
		side = order.Sell
	}
	return &order.Detail{
		Exchange:        e.Name,
		AssetType:       asset.Futures,
		OrderID:         o.OrderID,
		ClientOrderID:   o.ClientOrderID,
		Pair:            pair,
		Side:            side,
		Type:            orderType,
		TimeInForce:     tif,
		Status:          status,
		ReduceOnly:      o.ReduceOnly,
		Price:           o.LimitPrice,
		TriggerPrice:    o.StopPrice,
		Amount:          o.Quantity,
		ExecutedAmount:  o.Filled,
		RemainingAmount: o.Quantity - o.Filled,
		Date:            o.Time.Time(),
		LastUpdated:     o.LastUpdateTime.Time(),
	}, nil
}

// futuresOrderUpdateStatus returns the status of a futures order, from why an update changed it. An order removed from
// the open orders was filled, cancelled, rejected, expired with its contract, or, for a stop, triggered
func futuresOrderUpdateStatus(reason string, isCancel bool, filled float64) order.Status {
	if !isCancel {
		if filled > 0 {
			return order.PartiallyFilled
		}
		return order.Open
	}
	switch reason {
	case "full_fill":
		return order.Filled
	case "stop_order_triggered":
		return order.Closed
	case "contract_expired":
		return order.Expired
	case "post_order_failed_because_it_would_filled", "ioc_order_failed_because_it_would_not_be_executed", "would_execute_self",
		"would_not_reduce_position", "order_for_edit_not_found":
		return order.Rejected
	}
	if filled > 0 {
		return order.PartiallyFilledCancelled
	}
	return order.Cancelled
}

// wsFuturesProcessFills relays the account's futures fills: each as its order's trade, with its fee, and as a fill when
// the fills feed is enabled
func (e *Exchange) wsFuturesProcessFills(ctx context.Context, respRaw []byte) error {
	var wsFills WsFuturesFills
	if err := json.Unmarshal(respRaw, &wsFills); err != nil {
		return err
	}
	var fills []fill.Data
	for i := range wsFills.Fills {
		f := &wsFills.Fills[i]
		pair, err := e.futuresPair(f.Instrument)
		if err != nil {
			return err
		}
		side := order.Sell
		if f.Buy {
			side = order.Buy
		}
		orderType, _, err := futuresOrderTypeFromString(f.OrderType)
		if err != nil {
			return err
		}
		if err := e.Websocket.DataHandler.Send(ctx, &order.Detail{
			Exchange:      e.Name,
			AssetType:     asset.Futures,
			OrderID:       f.OrderID,
			ClientOrderID: f.ClientOrderID,
			Pair:          pair,
			Side:          side,
			Type:          orderType,
			LastUpdated:   f.Time.Time(),
			Trades: []order.TradeHistory{{
				TID:       f.FillID,
				Price:     f.Price,
				Amount:    f.Quantity,
				Fee:       f.FeePaid,
				FeeAsset:  e.assetCode(f.FeeCurrency).String(),
				Exchange:  e.Name,
				Type:      orderType,
				Side:      side,
				Timestamp: f.Time.Time(),
				IsMaker:   f.FillType == "maker",
			}},
		}); err != nil {
			return err
		}
		if e.IsFillsFeedEnabled() {
			fills = append(fills, fill.Data{
				ID:            f.FillID,
				Timestamp:     f.Time.Time(),
				Exchange:      e.Name,
				AssetType:     asset.Futures,
				CurrencyPair:  pair,
				Side:          side,
				OrderID:       f.OrderID,
				ClientOrderID: f.ClientOrderID,
				TradeID:       f.FillID,
				Price:         f.Price,
				Amount:        f.Quantity,
			})
		}
	}
	if len(fills) == 0 {
		return nil
	}
	return e.Websocket.Fills.Update(fills...)
}

// wsFuturesProcessBalances stores the cash and multi-collateral wallet balances a balances message holds, and relays
// them with the message, whose single-collateral margin accounts are relayed only, as Kraken names them differently
// over REST
func (e *Exchange) wsFuturesProcessBalances(ctx context.Context, respRaw []byte) error {
	var b WsFuturesBalances
	if err := json.Unmarshal(respRaw, &b); err != nil {
		return err
	}
	var subAccts accounts.SubAccounts
	if len(b.Holding) != 0 {
		cash := accounts.NewSubAccount(asset.Futures, "cash")
		for code, amount := range b.Holding {
			c := currency.NewCode(code).Upper()
			balance, err := e.Accounts.UpdateBalance(ctx, cash.ID, asset.Futures, c, func(bal *accounts.Balance) {
				bal.Total, bal.Hold, bal.Free = amount, 0, amount
			})
			if err != nil {
				return err
			}
			cash.Balances.Set(c, balance)
		}
		subAccts = append(subAccts, cash)
	}
	if b.FlexFutures != nil && len(b.FlexFutures.Currencies) != 0 {
		flex := accounts.NewSubAccount(asset.Futures, "flex")
		for code, fc := range b.FlexFutures.Currencies {
			c := currency.NewCode(code).Upper()
			balance, err := e.Accounts.UpdateBalance(ctx, flex.ID, asset.Futures, c, func(bal *accounts.Balance) {
				free := min(max(fc.Available, 0), fc.Quantity)
				bal.Total, bal.Hold, bal.Free = fc.Quantity, fc.Quantity-free, free
			})
			if err != nil {
				return err
			}
			flex.Balances.Set(c, balance)
		}
		subAccts = append(subAccts, flex)
	}
	if len(subAccts) != 0 {
		if err := e.Websocket.DataHandler.Send(ctx, subAccts); err != nil {
			return err
		}
	}
	return e.Websocket.DataHandler.Send(ctx, &b)
}
