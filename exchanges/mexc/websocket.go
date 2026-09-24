package mexc

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"google.golang.org/protobuf/proto"
)

const (
	spotWebsocketURL = "wss://wbs-api.mexc.com/ws"

	channelBookTiker        = "public.aggre.bookTicker.v3.api.pb"
	channelMiniTickerV3     = "public.miniTicker.v3.api.pb"
	channelMiniTickersV3    = "public.miniTickers.v3.api.pb"
	channelAggreDealsV3     = "public.aggre.deals.v3.api.pb"
	channelKlineV3          = "public.kline.v3.api.pb"
	channelLimitDepthV3     = "public.limit.depth.v3.api.pb"
	channelBookTickerBatch  = "public.bookTicker.batch.v3.api.pb"
	channelAccountV3        = "private.account.v3.api.pb"
	channelPrivateDealsV3   = "private.deals.v3.api.pb"
	channelPrivateOrdersAPI = "private.orders.v3.api.pb"

	// miniTickerTimezone is a mandatory suffix of the spot miniTicker and miniTickers channels: MEXC
	// rejects the subscription without it ("Not Subscribed successfully! ... Reason: Blocked!" — measured
	// live).
	// It only shifts the rate fields we do not consume; price/high/low/volume are timezone-agnostic.
	miniTickerTimezone = "UTC+8"
	// wsPongMessage is the msg field of the spot ping acknowledgement
	wsPongMessage = "PONG"
)

// WsConnect initiates a websocket connection
func (e *Exchange) WsConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var listenKey string
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		var err error
		if listenKey, err = e.GenerateListenKey(ctx); err != nil {
			return err
		}
		conn.SetURL(conn.GetURL() + "?listenKey=" + listenKey)
	}
	if err := conn.Dial(ctx, &gws.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
	}, http.Header{}, nil); err != nil {
		if listenKey != "" {
			// No renewer owns the key until the connection is up, so release it here: the monitor retries a
			// failed connect every few seconds and would otherwise use up the account's listen keys.
			e.releaseListenKey(ctx, listenKey)
		}
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"method": "PING"}`),
		Delay:       time.Second * 20,
	})
	if listenKey != "" {
		// The stream closes 60 minutes after creation unless a keepalive is sent; renew this
		// connection's own key on a timer, but only once the connection is up so a failed dial does
		// not leak the renewer.
		go e.keepListenKeyAlive(ctx, conn, listenKey)
	}
	return nil
}

// listenKeyKeepAliveInterval renews the user data stream well within its 60-minute expiry. A variable
// rather than a constant so tests can shorten it.
var listenKeyKeepAliveInterval = 30 * time.Minute

// listenKeyCloseTimeout bounds the request that releases a listen key once its renewer stops
const listenKeyCloseTimeout = 5 * time.Second

// keepListenKeyAlive renews one connection's user data stream on a timer for as long as the connection
// lives. Each authenticated connection mints its own listen key, so the renewer is handed that key and
// renews it: once subscriptions span more than one connection, a single shared slot would hold only
// the last key minted and leave every other stream to expire after an hour, silently stopping the
// private updates on it. The stream closes 60 minutes after creation unless a keepalive PUT is sent;
// the PING handler keeps the socket open but does not touch the key. When the renewer stops it closes
// the key, since the venue caps the listen keys an account may hold and every reconnect mints a new
// one. The renewer stops when the manager shuts down or its context is cancelled; a connection lost on
// its own is only noticed at the next renewal tick, so its key is released up to one interval later.
func (e *Exchange) keepListenKeyAlive(ctx context.Context, conn websocket.Connection, listenKey string) {
	e.Websocket.Wg.Add(1)
	defer e.Websocket.Wg.Done()
	defer e.releaseListenKey(ctx, listenKey)
	renew := time.NewTicker(listenKeyKeepAliveInterval)
	defer renew.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.Websocket.ShutdownC:
			return
		case <-renew.C:
			// The manager can close one connection without closing ShutdownC: it rolls back the
			// connections already made when a later one fails. Stop renewing a key whose socket is gone,
			// so a rolled-back connection does not go on burning one of the account's listen keys.
			if c, ok := conn.(interface{ IsConnected() bool }); ok && !c.IsConnected() {
				return
			}
			if err := e.ExtendListenKey(ctx, listenKey); err != nil {
				_ = e.Websocket.DataHandler.Send(ctx, err)
			}
		}
	}
}

// releaseListenKey closes a connection's listen key. The connection's context is usually done by then,
// so the request runs on a detached one.
func (e *Exchange) releaseListenKey(ctx context.Context, listenKey string) {
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), listenKeyCloseTimeout)
	defer cancel()
	if err := e.CloseListenKey(closeCtx, listenKey); err != nil {
		log.Warnf(log.WebsocketMgr, "%s: closing listen key: %v", e.Name, err)
	}
}

// Subscribe subscribes to a channel
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, channelsToSubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, "SUBSCRIPTION", channelsToSubscribe)
}

// Unsubscribe unsubscribes to a channel
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, channelsToSubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, "UNSUBSCRIPTION", channelsToSubscribe)
}

func assetTypeToString(assetType asset.Item) string {
	if assetType != asset.Spot {
		return ""
	}
	return strings.ToLower(assetType.String())
}

func channelName(s *subscription.Subscription) string {
	if s.Asset == asset.Spot {
		switch s.Channel {
		case subscription.TickerChannel:
			return channelBookTiker
		case subscription.OrderbookChannel:
			return channelLimitDepthV3
		case subscription.AllTradesChannel:
			return channelAggreDealsV3
		case subscription.CandlesChannel:
			return channelKlineV3
		case subscription.MyTradesChannel:
			return channelPrivateDealsV3
		case subscription.MyOrdersChannel:
			return channelPrivateOrdersAPI
		case subscription.MyAccountChannel:
			return channelAccountV3
		}
	}
	return s.Channel
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 5},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.FifteenMin},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
	// bookTicker (above) carries only the best bid/offer; miniTicker carries last/high/low/volume.
	// Both feed the same ticker.Price: without the second one the websocket ticker would suppress
	// the REST ticker sync and freeze last/high/low/volume at their last polled value.
	{Enabled: true, Asset: asset.Spot, Channel: channelMiniTickerV3},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds},

	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	return e.Features.Subscriptions.ExpandTemplates(e)
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").
		Funcs(template.FuncMap{
			"channelName":       channelName,
			"channelSuffix":     channelSuffix,
			"assetTypeToString": assetTypeToString,
			"wsIntervalString":  wsIntervalString,
			"isSymbolChannel":   isSymbolChannel,
			"formatPair":        e.FormatExchangeCurrency,
		}).
		Parse(subTplText)
}

func wsIntervalString(s *subscription.Subscription) string {
	intervalString, err := intervalToString(s.Interval, true)
	if err != nil {
		return ""
	}
	return intervalString
}

// wsChannelName returns the channel name of a qualified push channel. MEXC qualifies a channel as
// "spot@<name>[@<extra>...]": a public channel carries an interval and/or a symbol after the name,
// a private channel carries nothing. The name must be read from the decoded frame — splitting the
// raw protobuf bytes on "@" returns the name glued to the binary body whenever nothing follows it,
// which matched no case and made every private channel unroutable.
func wsChannelName(qualifiedChannel string) string {
	parts := strings.Split(qualifiedChannel, "@")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// wsUnhandled reports a frame the handler has no mapping for
func (e *Exchange) wsUnhandled(ctx context.Context, respRaw []byte) error {
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
		Message: string(respRaw) + websocket.UnhandledMessage,
	})
}

// isSymbolChannel reports whether a channel is subscribed per symbol. The private channels and the
// all-symbols miniTickers channel carry no symbol.
func isSymbolChannel(channel string) bool {
	return !slices.Contains([]string{channelAccountV3, channelPrivateDealsV3, channelPrivateOrdersAPI, channelMiniTickersV3}, channel)
}

// channelSuffix returns the trailing element a channel requires after the symbol, or after the channel
// name when it carries no symbol, if any
func channelSuffix(channel string) string {
	if channel == channelMiniTickerV3 || channel == channelMiniTickersV3 {
		return "@" + miniTickerTimezone
	}
	return ""
}

// subscriptionAccepted reports whether MEXC actually accepted the subscription. MEXC answers a
// rejected subscription with code 0 and an error text in msg (measured live:
// `code=0 msg="Not Subscribed successfully! [<channel>]. Reason： Blocked!"`), so the code alone
// cannot distinguish success from failure and a rejected channel would be registered as live.
// An accepted request echoes the qualified channel back verbatim.
func subscriptionAccepted(method, qualifiedChannel, msg string) bool {
	if method != "SUBSCRIPTION" {
		return true
	}
	return msg == qualifiedChannel
}

func (e *Exchange) handleSubscription(ctx context.Context, conn websocket.Connection, method string, subs subscription.List) error {
	var confirmed, rejected subscription.List
	for s := range subs {
		id := e.MessageSequence()
		data, err := conn.SendMessageReturnResponse(ctx, request.UnAuth, id, &WsSubscriptionPayload{
			ID:     id,
			Method: method,
			Params: []string{subs[s].QualifiedChannel},
		})
		if err != nil {
			return err
		}
		var resp *WsSubscriptionResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		} else if resp.Code != 0 || !subscriptionAccepted(method, subs[s].QualifiedChannel, resp.Message) {
			rejected = append(rejected, subs[s])
		} else {
			confirmed = append(confirmed, subs[s])
		}
	}
	if method == "UNSUBSCRIPTION" {
		// A confirmed unsubscription removes the channel from the active set; a rejected one is left
		// registered because it is still live. The previous code added every confirmed channel via
		// AddSuccessfulSubscriptions, so an accepted unsubscribe (MEXC replies code 0 echoing the
		// channel — measured live) re-registered the channel it had just cancelled.
		return e.Websocket.RemoveSubscriptions(conn, confirmed...)
	}
	// A rejected subscription was never stored, so there is nothing to remove: register the confirmed
	// ones and name the rejected ones in the error.
	err := e.Websocket.AddSuccessfulSubscriptions(conn, confirmed...)
	if len(rejected) > 0 {
		err = common.AppendError(err, fmt.Errorf("%w: %s", websocket.ErrSubscriptionFailure, rejected))
	}
	return err
}

// wsUpdateSpotTicker merges a partial spot ticker update into the cached ticker and publishes it.
// MEXC splits the spot ticker over two channels — bookTicker carries the best bid/offer only and
// miniTicker carries last/high/low/volume — so each update must be applied on top of the current
// ticker instead of replacing it, otherwise every channel would blank the other one's fields.
func (e *Exchange) wsUpdateSpotTicker(ctx context.Context, cp currency.Pair, updated time.Time, apply func(*ticker.Price)) error {
	// bookTicker and miniTicker for one pair can land on different connections once subscriptions span
	// more than one, so serialise the read-merge-write of the cached ticker or concurrent updates race
	// and one is lost.
	e.wsTickerMu.Lock()
	defer e.wsTickerMu.Unlock()
	tick, err := e.GetCachedTicker(cp, asset.Spot)
	if err != nil {
		tick = &ticker.Price{Pair: cp, ExchangeName: e.Name, AssetType: asset.Spot}
	}
	apply(tick)
	tick.LastUpdated = updated
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// wsUpdateSpotMiniTicker applies one miniTicker record to the cached spot ticker. The per-symbol
// miniTicker channel and the all-symbols miniTickers channel push the same record, so both go through
// here.
func (e *Exchange) wsUpdateSpotMiniTicker(ctx context.Context, cp currency.Pair, updated time.Time, body *mexc_proto_types.PublicMiniTickerV3Api) error {
	last, err := parseOptionalFloat(body.Price)
	if err != nil {
		return err
	}
	high, err := parseOptionalFloat(body.High)
	if err != nil {
		return err
	}
	low, err := parseOptionalFloat(body.Low)
	if err != nil {
		return err
	}
	// Measured against GET /api/v3/ticker/24hr for KASUSDT: miniTicker `quantity` is the base
	// asset volume and `volume` is the quote volume — the opposite of the REST field naming.
	baseVolume, err := parseOptionalFloat(body.Quantity)
	if err != nil {
		return err
	}
	quoteVolume, err := parseOptionalFloat(body.Volume)
	if err != nil {
		return err
	}
	return e.wsUpdateSpotTicker(ctx, cp, updated, func(t *ticker.Price) {
		setIfNonZero(&t.Last, last)
		setIfNonZero(&t.High, high)
		setIfNonZero(&t.Low, low)
		// Volume is legitimately zero on an idle symbol, so presence in the frame decides rather
		// than the value: setIfNonZero cannot tell an explicit "0" from an omitted field.
		if body.Quantity != "" {
			t.BaseVolume = baseVolume
		}
		if body.Volume != "" {
			t.QuoteVolume = quoteVolume
		}
	})
}

// parseOptionalFloat parses a numeric field which the exchange may omit entirely
func parseOptionalFloat(v string) (float64, error) {
	if v == "" {
		return 0, nil
	}
	return strconv.ParseFloat(v, 64)
}

// setIfNonZero keeps the previously known value when an update omits the field
func setIfNonZero(dst *float64, v float64) {
	if v != 0 {
		*dst = v
	}
}

// wsSendTime returns the exchange send time of a push frame, falling back to local time when absent
func wsSendTime(w *mexc_proto_types.PushDataV3ApiWrapper) time.Time {
	if st := w.GetSendTime(); st != 0 {
		return time.UnixMilli(st)
	}
	return time.Now()
}

// isSpotPongMessage reports whether the payload is the acknowledgement of {"method":"PING"}.
// It arrives as {"id":0,"code":0,"msg":"PONG"}: id 0 matches no pending request, so without
// this check it would be reported as an unhandled message.
func isSpotPongMessage(respRaw []byte) bool {
	msg, err := jsonparser.GetString(respRaw, "msg")
	return err == nil && strings.EqualFold(msg, wsPongMessage)
}

// privateOrderNumbers holds the decoded numeric fields of a private order push
type privateOrderNumbers struct {
	price, avgPrice            float64
	quantity, remainQuantity   float64
	amount, cumulativeQuantity float64
	cumulativeAmount           float64
	triggerPrice               float64
}

// parse decodes the numeric fields of a private order push, which the exchange sends as strings and
// omits when they do not apply
func (n *privateOrderNumbers) parse(body *mexc_proto_types.PrivateOrdersV3Api) error {
	for _, f := range []struct {
		name string
		raw  string
		dst  *float64
	}{
		{"price", body.Price, &n.price},
		{"avgPrice", body.AvgPrice, &n.avgPrice},
		{"quantity", body.Quantity, &n.quantity},
		{"remainQuantity", body.RemainQuantity, &n.remainQuantity},
		{"amount", body.Amount, &n.amount},
		{"cumulativeQuantity", body.CumulativeQuantity, &n.cumulativeQuantity},
		{"cumulativeAmount", body.CumulativeAmount, &n.cumulativeAmount},
		// The stop trigger price is only present on stop orders; the REST order carries it as
		// stopPrice, so a stop order pushed over the socket reports the same trigger the REST side does.
		{"triggerPrice", body.GetTriggerPrice(), &n.triggerPrice},
	} {
		v, err := parseOptionalFloat(f.raw)
		if err != nil {
			return fmt.Errorf("private order field %s: %w", f.name, err)
		}
		*f.dst = v
	}
	return nil
}

// WsHandleData will read websocket raw data and pass to appropriate handler
func (e *Exchange) WsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if strings.HasPrefix(string(respRaw), "{") {
		if isSpotPongMessage(respRaw) {
			return nil
		}
		if id, err := jsonparser.GetInt(respRaw, "id"); err == nil {
			if !conn.IncomingWithData(id, respRaw) {
				return e.wsUnhandled(ctx, respRaw)
			}
		}
		// Ignore json messages which doesn't have an ID.
		return nil
	}
	result := &mexc_proto_types.PushDataV3ApiWrapper{}
	if err := proto.Unmarshal(respRaw, result); err != nil {
		return err
	}
	switch wsChannelName(result.GetChannel()) {
	case channelBookTiker:
		body := result.GetPublicAggreBookTicker()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		ask := orderbook.Level{}
		var err error
		ask.Price, err = strconv.ParseFloat(body.AskPrice, 64)
		if err != nil {
			return err
		}
		ask.Amount, err = strconv.ParseFloat(body.AskQuantity, 64)
		if err != nil {
			return err
		}
		bid := orderbook.Level{}
		bid.Price, err = strconv.ParseFloat(body.BidPrice, 64)
		if err != nil {
			return err
		}
		bid.Amount, err = strconv.ParseFloat(body.BidQuantity, 64)
		if err != nil {
			return err
		}
		cp, err := e.MatchSymbolWithAvailablePairs(result.GetSymbol(), asset.Spot, false)
		if err != nil {
			return err
		}
		// The best bid/offer is published as a ticker fact only. It deliberately does not write to
		// the orderbook: the depth channel owns that book, and a one-level update against a
		// multi-level snapshot would insert a level the exchange never sent. Other exchanges in this
		// repository keep the same separation by never enabling a BBO channel alongside a depth one.
		return e.wsUpdateSpotTicker(ctx, cp, wsSendTime(result), func(t *ticker.Price) {
			t.Bid, t.BidSize = bid.Price, bid.Amount
			t.Ask, t.AskSize = ask.Price, ask.Amount
		})
	case channelMiniTickerV3:
		body := result.GetPublicMiniTicker()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		cp, err := e.MatchSymbolWithAvailablePairs(body.Symbol, asset.Spot, false)
		if err != nil {
			return err
		}
		return e.wsUpdateSpotMiniTicker(ctx, cp, wsSendTime(result), body)
	case channelMiniTickersV3:
		body := result.GetPublicMiniTickers()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		updated := wsSendTime(result)
		for _, item := range body.Items {
			// The push covers every symbol whose price moved. Only enabled pairs are published: the
			// engine's sync manager rejects a ticker for any pair it does not track.
			cp, enabled, err := e.MatchSymbolCheckEnabled(item.Symbol, asset.Spot, false)
			if err != nil || !enabled {
				continue
			}
			if err := e.wsUpdateSpotMiniTicker(ctx, cp, updated, item); err != nil {
				return err
			}
		}
		return nil
	case channelAggreDealsV3:
		// Read both trade settings per frame so a feed switched on after setup takes effect straight
		// away; skip the work entirely when neither wants the trades. The private deals channel is
		// separate and carries the account's own fills.
		saveTradeData := e.IsSaveTradeDataEnabled()
		tradeFeed := e.IsTradeFeedEnabled()
		if !saveTradeData && !tradeFeed {
			return nil
		}
		cp, err := e.MatchSymbolWithAvailablePairs(result.GetSymbol(), asset.Spot, false)
		if err != nil {
			return err
		}
		body := result.GetPublicAggreDeals()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		tradesDetail := make([]trade.Data, len(body.Deals))
		for t := range body.Deals {
			price, err := strconv.ParseFloat(body.Deals[t].Price, 64)
			if err != nil {
				return err
			}
			amount, err := strconv.ParseFloat(body.Deals[t].Quantity, 64)
			if err != nil {
				return err
			}
			tradesDetail[t] = trade.Data{
				Exchange:     e.Name,
				CurrencyPair: cp,
				AssetType:    asset.Spot,
				Price:        price,
				Amount:       amount,
				Timestamp:    time.UnixMilli(body.Deals[t].Time),
				Side: func() order.Side {
					if body.Deals[t].TradeType == 1 {
						return order.Buy
					}
					return order.Sell
				}(),
			}
		}
		if tradeFeed {
			if err := e.Websocket.DataHandler.Send(ctx, tradesDetail); err != nil {
				return err
			}
		}
		if saveTradeData {
			// AddTradesToBuffer writes to the trades it is given, so hand it its own copy rather than
			// the slice already passed to the data handler.
			return trade.AddTradesToBuffer(slices.Clone(tradesDetail)...)
		}
		return nil
	case channelKlineV3:
		body := result.GetPublicSpotKline()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		cp, err := e.MatchSymbolWithAvailablePairs(result.GetSymbol(), asset.Spot, false)
		if err != nil {
			return err
		}
		interval, err := IntervalFromString(body.Interval)
		if err != nil {
			return err
		}
		klineData := kline.Candle{}
		// MEXC sends windowStart/windowEnd in whole seconds (measured live on BTCUSDT Min1:
		// windowStart=1788890580 => 2026-09-08 18:03:00Z). Reading windowEnd as milliseconds put
		// every candle near the Unix epoch. The candle is stamped with its open (windowStart), matching
		// this repository's kline convention that Candle.Time is the interval start.
		klineData.Time = time.Unix(body.WindowStart, 0)
		// `volume` is the base-asset volume; `amount` is the quote turnover (measured live for one
		// BTCUSDT candle: volume=0.11909876 base vs amount=9356.31 quote). Candle.Volume is base terms.
		if klineData.Volume, err = strconv.ParseFloat(body.Volume, 64); err != nil {
			return err
		}
		klineData.Low, err = strconv.ParseFloat(body.LowestPrice, 64)
		if err != nil {
			return err
		}
		klineData.High, err = strconv.ParseFloat(body.HighestPrice, 64)
		if err != nil {
			return err
		}
		klineData.Open, err = strconv.ParseFloat(body.OpeningPrice, 64)
		if err != nil {
			return err
		}
		klineData.Close, err = strconv.ParseFloat(body.ClosingPrice, 64)
		if err != nil {
			return err
		}
		// MEXC pushes a window repeatedly while it is open and never sends a closing frame, so every
		// candle relayed here is still forming. The routine manager matches kline.Item by value.
		klineData.ValidationIssues = kline.PartialCandle
		return e.Websocket.DataHandler.Send(ctx, kline.Item{
			Pair:     cp,
			Exchange: e.Name,
			Asset:    asset.Spot,
			Interval: interval,
			Candles:  []kline.Candle{klineData},
		})
	case channelLimitDepthV3:
		cp, err := e.MatchSymbolWithAvailablePairs(result.GetSymbol(), asset.Spot, false)
		if err != nil {
			return err
		}
		body := result.GetPublicLimitDepths()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		asks := make(orderbook.Levels, len(body.Asks))
		for a := range body.Asks {
			asks[a].Price, err = strconv.ParseFloat(body.Asks[a].Price, 64)
			if err != nil {
				return err
			}
			asks[a].Amount, err = strconv.ParseFloat(body.Asks[a].Quantity, 64)
			if err != nil {
				return err
			}
		}
		bids := make(orderbook.Levels, len(body.Bids))
		for b := range body.Bids {
			bids[b].Price, err = strconv.ParseFloat(body.Bids[b].Price, 64)
			if err != nil {
				return err
			}
			bids[b].Amount, err = strconv.ParseFloat(body.Bids[b].Quantity, 64)
			if err != nil {
				return err
			}
		}
		return e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Exchange:    e.Name,
			Asset:       asset.Spot,
			Bids:        bids,
			Asks:        asks,
			Pair:        cp,
			LastUpdated: wsSendTime(result),
		})
	case channelBookTickerBatch:
		cp, err := e.MatchSymbolWithAvailablePairs(result.GetSymbol(), asset.Spot, false)
		if err != nil {
			return err
		}
		body := result.GetPublicBookTickerBatch()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		// Merge each item onto the cached ticker through the shared helper rather than sending fresh
		// ticker.Price values: the batch frame carries only the best bid/offer, so replacing the ticker
		// would blank the last/high/low/volume fields the miniTicker channel maintains.
		for a := range body.Items {
			bid, err := strconv.ParseFloat(body.Items[a].BidPrice, 64)
			if err != nil {
				return err
			}
			ask, err := strconv.ParseFloat(body.Items[a].AskPrice, 64)
			if err != nil {
				return err
			}
			bidSize, err := strconv.ParseFloat(body.Items[a].BidQuantity, 64)
			if err != nil {
				return err
			}
			askSize, err := strconv.ParseFloat(body.Items[a].AskQuantity, 64)
			if err != nil {
				return err
			}
			if err := e.wsUpdateSpotTicker(ctx, cp, wsSendTime(result), func(t *ticker.Price) {
				t.Bid, t.BidSize = bid, bidSize
				t.Ask, t.AskSize = ask, askSize
			}); err != nil {
				return err
			}
		}
		return nil
	case channelAccountV3:
		body := result.GetPrivateAccount()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		balanceAmount, err := strconv.ParseFloat(body.BalanceAmount, 64)
		if err != nil {
			return err
		}
		frozenAmount, err := strconv.ParseFloat(body.FrozenAmount, 64)
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, accounts.Change{
			AssetType: asset.Spot,
			Balance: accounts.Balance{
				Currency: currency.NewCode(body.VcoinName),
				// balanceAmount is available and frozenAmount is frozen; total is their sum. This
				// matches UpdateAccountBalances over REST (Total = free + locked).
				Total: balanceAmount + frozenAmount,
				Hold:  frozenAmount,
				Free:  balanceAmount,
			},
		})
	case channelPrivateDealsV3:
		cp, err := e.MatchSymbolWithAvailablePairs(result.GetSymbol(), asset.Spot, false)
		if err != nil {
			return err
		}
		body := result.GetPrivateDeals()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		price, err := strconv.ParseFloat(body.Price, 64)
		if err != nil {
			return err
		}
		// The fill's base size is quantity; amount is the quote value (price*quantity). fill.Data.Amount
		// is base terms, so it must come from quantity. The trade id is tradeId - orderId identifies the
		// order, not the individual fill, and would collide across a partially filled order's fills.
		quantity, err := strconv.ParseFloat(body.Quantity, 64)
		if err != nil {
			return err
		}
		// The account's own fills are published as fill.Data, as bybit and gateio do, so that they are not
		// mistaken for the market's trades and keep the order they belong to.
		side := order.Sell
		if body.TradeType == 1 {
			side = order.Buy
		}
		return e.Websocket.DataHandler.Send(ctx, []fill.Data{{
			ID:            body.TradeId,
			TradeID:       body.TradeId,
			Timestamp:     time.UnixMilli(body.Time),
			Exchange:      e.Name,
			AssetType:     asset.Spot,
			CurrencyPair:  cp,
			Side:          side,
			OrderID:       body.OrderId,
			ClientOrderID: body.ClientOrderId,
			Price:         price,
			Amount:        quantity,
		}})
	case channelPrivateOrdersAPI:
		var oType order.Type
		var tif order.TimeInForce
		body := result.GetPrivateOrders()
		if body == nil {
			return e.wsUnhandled(ctx, respRaw)
		}
		switch body.OrderType {
		case 1:
			tif = order.GoodTillCancel
			oType = order.Limit
		case 2:
			tif = order.PostOnly
			oType = order.Limit
		case 3:
			tif = order.ImmediateOrCancel
			oType = order.Limit
		case 4:
			oType = order.Limit
			tif = order.FillOrKill
		case 5:
			oType = order.Market
		case 100:
			// Documented only as "Stop loss/take profit". The docs do not state whether the trigger
			// executes at market or at a limit, so StopMarket is an assumption, not a documented mapping.
			oType = order.StopMarket
		default:
			// An unrecognised order type must not sink the frame: report it as unknown, log the raw
			// value for diagnosis and still publish the order.
			oType = order.UnknownType
			log.Warnf(log.ExchangeSys, "%s: unhandled private order type %d", e.Name, body.OrderType)
		}
		var oStatus order.Status
		switch body.Status {
		case 1:
			oStatus = order.New
		case 2:
			oStatus = order.Filled
		case 3:
			oStatus = order.PartiallyFilled
		case 4:
			oStatus = order.Cancelled
		case 5:
			oStatus = order.PartiallyCancelled
		default:
			oStatus = order.UnknownStatus
			log.Warnf(log.ExchangeSys, "%s: unhandled private order status %d", e.Name, body.Status)
		}
		cp, err := e.MatchSymbolWithAvailablePairs(result.GetSymbol(), asset.Spot, false)
		if err != nil {
			return err
		}
		// MEXC reports two parallel sets of figures: quantity/remainQuantity/cumulativeQuantity are
		// base terms and amount/remainAmount/cumulativeAmount are quote terms. order.Detail.Amount,
		// ExecutedAmount and RemainingAmount are base terms, so they must come from the quantity
		// fields; mixing the two reported the same order in different units depending on the source,
		// the same defect GetOrderInfo carried on the REST side.
		var nums privateOrderNumbers
		if err := nums.parse(body); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &order.Detail{
			Exchange:             e.Name,
			Price:                nums.price,
			Amount:               nums.quantity,
			AverageExecutedPrice: nums.avgPrice,
			TriggerPrice:         nums.triggerPrice,
			QuoteAmount:          nums.amount,
			// cumulativeAmount is the quote actually spent; without it a filled order reports a zero cost.
			Cost:            nums.cumulativeAmount,
			ExecutedAmount:  nums.cumulativeQuantity,
			RemainingAmount: nums.remainQuantity,
			OrderID:         body.Id,
			// clientId on the order push is the caller's own client order id; ClientID is the
			// account-level identifier and does not belong here.
			ClientOrderID: body.ClientId,
			Type:          oType,
			Side: func() order.Side {
				if body.TradeType == 1 {
					return order.Buy
				}
				return order.Sell
			}(),
			Status:      oStatus,
			AssetType:   asset.Spot,
			Date:        time.UnixMilli(body.CreateTime),
			LastUpdated: wsSendTime(result),
			Pair:        cp,
			TimeInForce: tif,
		})
	default:
		return e.wsUnhandled(ctx, respRaw)
	}
}

const subTplText = `
{{- with $name := channelName $.S }}
		{{- if isSymbolChannel $name -}}
			{{- range $asset, $pairs := $.AssetPairs }}
				{{- if (gt $.S.Interval 0) }}
					{{- range $p := $pairs -}}
						{{- if  (eq $name "public.kline.v3.api.pb") -}}
							{{- assetTypeToString $asset }}@{{- $name -}}@{{- formatPair $p $asset }}@{{- wsIntervalString $.S}}
						{{- else }}
							{{- assetTypeToString $asset }}@{{- $name -}}@{{- wsIntervalString $.S}}@{{- formatPair $p $asset }}
						{{- end }}
						{{- $.PairSeparator }}
					{{- end }}
				{{- else if (gt $.S.Levels 0) }}
					{{- range $p := $pairs -}}
						{{- assetTypeToString $asset }}@{{- $name -}}@{{- formatPair $p $asset }}@{{ $.S.Levels }}
						{{- $.PairSeparator }}
					{{- end }}
				{{- else }}
					{{- range $p := $pairs -}}
						{{- assetTypeToString $asset }}@{{- $name -}}@{{- formatPair $p $asset }}{{- channelSuffix $name }}
						{{- $.PairSeparator }}
					{{- end }}
				{{- end }}
				{{- $.AssetSeparator }}
			{{- end }}
	{{- else }}
		{{- assetTypeToString $.S.Asset}}@{{- $name -}}{{- channelSuffix $name }}
	{{- end }}
{{- end }}
`
