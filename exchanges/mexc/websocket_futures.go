package mexc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	futuresWebsocketURL = "wss://contract.mexc.com/edge"

	// futuresPublicConnection and futuresPrivateConnection filter the futures connections
	futuresPublicConnection  = "futuresPublic"
	futuresPrivateConnection = "futuresPrivate"

	// futuresWebsocketReplySignature matches a futures request to its reply. The venue's replies carry no request
	// ID, so one request is in flight at a time and its reply is the next one on the connection
	futuresWebsocketReplySignature = "futuresReply"

	futuresWebsocketPing         = `{"method":"ping"}`
	futuresWebsocketPingInterval = 15 * time.Second

	// FuturesPositionsChannel is the private futures positions channel, named for the venue's filter key
	FuturesPositionsChannel = "position"

	futuresChannelTicker       = "ticker"
	futuresChannelDeal         = "deal"
	futuresChannelDepthFull    = "depth.full"
	futuresChannelKline        = "kline"
	futuresChannelOrder        = "personal.order"
	futuresChannelOrderDeal    = "personal.order.deal"
	futuresChannelAsset        = "personal.asset"
	futuresChannelPosition     = "personal.position"
	futuresChannelPong         = "pong"
	futuresChannelReplyPrefix  = "rs."
	futuresChannelError        = "rs.error"
	futuresChannelPushPrefix   = "push."
	futuresReplySuccess        = "success"
	futuresDefaultDepthLevels  = 20
	futuresMethodLogin         = "login"
	futuresSubscribePrefix     = "sub."
	futuresUnsubscribePrefix   = "unsub."
	futuresQualifiedSeparator  = "@"
	futuresPrivateChannelStart = "personal."
)

// futuresPrivateChannels maps the private futures subscription channels to the venue's push channels
var futuresPrivateChannels = map[string]string{
	subscription.MyOrdersChannel:  futuresChannelOrder,
	subscription.MyTradesChannel:  futuresChannelOrderDeal,
	subscription.MyAccountChannel: futuresChannelAsset,
	FuturesPositionsChannel:       futuresChannelPosition,
}

// WebsocketFuturesConnect dials a futures websocket connection
func (e *Exchange) WebsocketFuturesConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	// The pushes count volumes in contracts and are restated with each contract's size, so the sizes are read
	// before the pushes start; the handler reads them itself should this fail
	e.futuresContractsMu.RLock()
	known := len(e.futuresContracts) > 0
	e.futuresContractsMu.RUnlock()
	if !known {
		if _, err := e.GetFuturesContracts(ctx, currency.EMPTYPAIR); err != nil {
			log.Warnf(log.WebsocketMgr, "%s: futures contract details unavailable: %v", e.Name, err)
		}
	}
	if err := conn.Dial(ctx, &gws.Dialer{
		EnableCompression: true,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
	}, http.Header{}, nil); err != nil {
		return err
	}
	// The venue closes a connection which has not pinged within a minute
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(futuresWebsocketPing),
		Delay:       futuresWebsocketPingInterval,
	})
	return nil
}

// websocketFuturesConnectPrivate dials the connection carrying the private futures pushes. Its login is sent by the
// subscriber, so that a refused login fails the private subscriptions rather than every connection
func (e *Exchange) websocketFuturesConnectPrivate(ctx context.Context, conn websocket.Connection) error {
	return e.WebsocketFuturesConnect(ctx, conn)
}

// futuresWebsocketRequest sends a request over a futures connection and checks its reply
func (e *Exchange) futuresWebsocketRequest(ctx context.Context, conn websocket.Connection, req *WebsocketFuturesRequest) error {
	e.futuresWebsocketRequestMu.Lock()
	defer e.futuresWebsocketRequestMu.Unlock()
	data, err := conn.SendMessageReturnResponse(ctx, request.Unset, futuresWebsocketReplySignature, req)
	if err != nil {
		if errors.Is(err, websocket.ErrSignatureTimeout) || errors.Is(err, context.DeadlineExceeded) {
			awaitLateFuturesReply(ctx, conn)
		}
		return err
	}
	var reply *WebsocketFuturesResponse
	if err := json.Unmarshal(data, &reply); err != nil {
		return err
	}
	if reply == nil {
		return common.ErrNoResponse
	}
	var message string
	if err := json.Unmarshal(reply.Data, &message); err != nil {
		message = string(reply.Data)
	}
	switch {
	case reply.Channel == futuresChannelReplyPrefix+req.Method && message == futuresReplySuccess:
		return nil
	case reply.Channel == futuresChannelError:
		return fmt.Errorf("%w: %s: %s", errFuturesWebsocketRejected, req.Method, message)
	}
	return fmt.Errorf("%w: %s answered by %s: %s", errFuturesWebsocketRejected, req.Method, reply.Channel, message)
}

// awaitLateFuturesReply waits, as long again, for the reply to a request which went unanswered in time. The venue's
// replies name no request, so a late one would otherwise answer the next request sent
func awaitLateFuturesReply(ctx context.Context, conn websocket.Connection) {
	late, err := conn.MatchReturnResponses(context.WithoutCancel(ctx), futuresWebsocketReplySignature, 1)
	if err != nil {
		return
	}
	<-late
}

// futuresChannelRequest returns the venue's method and parameters of a public futures subscription
func futuresChannelRequest(s *subscription.Subscription) (string, *WebsocketFuturesChannelParams, error) {
	if len(s.Pairs) != 1 {
		return "", nil, fmt.Errorf("%w: %s needs one pair, has %d", subscription.ErrNotSinglePair, s.Channel, len(s.Pairs))
	}
	params := &WebsocketFuturesChannelParams{Symbol: s.Pairs[0].Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}).String()}
	switch s.Channel {
	case subscription.TickerChannel:
		return futuresChannelTicker, params, nil
	case subscription.AllTradesChannel:
		return futuresChannelDeal, params, nil
	case subscription.OrderbookChannel:
		params.Limit = uint64(max(s.Levels, 0))
		if params.Limit == 0 {
			params.Limit = futuresDefaultDepthLevels
		}
		return futuresChannelDepthFull, params, nil
	case subscription.CandlesChannel:
		interval, err := futuresIntervalString(s.Interval)
		if err != nil {
			return "", nil, err
		}
		params.Interval = interval
		return futuresChannelKline, params, nil
	}
	return "", nil, fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel)
}

// WebsocketFuturesSubscribe subscribes to public futures channels, one request at a time. A refused subscription is
// returned as an error and the others are still sent
func (e *Exchange) WebsocketFuturesSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	var errs error
	for _, s := range subs {
		method, params, err := futuresChannelRequest(s)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if err := e.futuresWebsocketRequest(ctx, conn, &WebsocketFuturesRequest{Method: futuresSubscribePrefix + method, Param: params}); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s: %w", websocket.ErrSubscriptionFailure, s.QualifiedChannel, err))
			continue
		}
		errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(conn, s))
	}
	return errs
}

// WebsocketFuturesUnsubscribe unsubscribes from public futures channels. The venue does not answer an
// unsubscription, so it is sent with the parameters the subscription was, which the venue needs to match them
func (e *Exchange) WebsocketFuturesUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	var errs error
	for _, s := range subs {
		method, params, err := futuresChannelRequest(s)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if err := conn.SendJSONMessage(ctx, request.Unset, &WebsocketFuturesRequest{Method: futuresUnsubscribePrefix + method, Param: params}); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("unsubscribing %s: %w", s.QualifiedChannel, err))
			continue
		}
		errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(conn, s))
	}
	return errs
}

// websocketFuturesLogin logs a futures connection in, once per connection
func (e *Exchange) websocketFuturesLogin(ctx context.Context, conn websocket.Connection) error {
	e.futuresLoginMu.Lock()
	defer e.futuresLoginMu.Unlock()
	if e.futuresLoginConn == conn {
		return nil
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	requestTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(creds.Key+requestTime), []byte(creds.Secret))
	if err != nil {
		return err
	}
	if err := e.futuresWebsocketRequest(ctx, conn, &WebsocketFuturesRequest{
		Method: futuresMethodLogin,
		Param: &WebsocketFuturesLoginParams{
			APIKey:      creds.Key,
			RequestTime: requestTime,
			Signature:   hex.EncodeToString(mac),
		},
	}); err != nil {
		return err
	}
	e.futuresLoginConn = conn
	return nil
}

// websocketFuturesSubscribePrivate subscribes to private futures channels. After a login the venue pushes every
// private channel, so the subscriptions are recorded and the pushes of channels without one are dropped
func (e *Exchange) websocketFuturesSubscribePrivate(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	if err := e.websocketFuturesLogin(ctx, conn); err != nil {
		return fmt.Errorf("%w: %w", websocket.ErrSubscriptionFailure, err)
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// websocketFuturesUnsubscribePrivate unsubscribes from private futures channels, whose pushes are then dropped
func (e *Exchange) websocketFuturesUnsubscribePrivate(_ context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.Websocket.RemoveSubscriptions(conn, subs...)
}

// isFuturesPrivateChannel reports whether an expanded subscription is to a private futures channel
func isFuturesPrivateChannel(s *subscription.Subscription) bool {
	return strings.HasPrefix(s.QualifiedChannel, futuresPrivateChannelStart)
}

// generateFuturesPublicSubscriptions returns the configured subscriptions to the public futures channels
func (e *Exchange) generateFuturesPublicSubscriptions() (subscription.List, error) {
	subs, err := e.generateSubscriptions()
	return slices.DeleteFunc(subs, func(s *subscription.Subscription) bool {
		return s.Asset != asset.Futures || isFuturesPrivateChannel(s)
	}), err
}

// generateFuturesPrivateSubscriptions returns the configured subscriptions to the private futures channels, none
// unless the authenticated websocket can be used
func (e *Exchange) generateFuturesPrivateSubscriptions() (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}
	subs, err := e.generateSubscriptions()
	return slices.DeleteFunc(subs, func(s *subscription.Subscription) bool {
		return s.Asset != asset.Futures || !isFuturesPrivateChannel(s)
	}), err
}

// futuresQualifiedChannel returns a futures subscription's qualified channel for a pair, the venue's channel and
// its parameters; it is empty for a channel the venue does not serve, which fails the subscription's expansion
func futuresQualifiedChannel(s *subscription.Subscription, pair currency.Pair) string {
	if private, ok := futuresPrivateChannels[s.Channel]; ok {
		return private
	}
	sub := s.Clone()
	sub.Pairs = currency.Pairs{pair}
	method, params, err := futuresChannelRequest(sub)
	if err != nil {
		return ""
	}
	qualified := method + futuresQualifiedSeparator + params.Symbol
	switch {
	case params.Interval != "":
		qualified += futuresQualifiedSeparator + params.Interval
	case params.Limit != 0:
		qualified += futuresQualifiedSeparator + strconv.FormatUint(params.Limit, 10)
	}
	return qualified
}

// isFuturesPairChannel reports whether a futures subscription is made per pair
func isFuturesPairChannel(s *subscription.Subscription) bool {
	_, private := futuresPrivateChannels[s.Channel]
	return !private
}

// futuresSubscriptionTemplate expands futures subscriptions; the private channels carry no pair
func (e *Exchange) futuresSubscriptionTemplate() (*template.Template, error) {
	return template.New("futures.tmpl").
		Funcs(template.FuncMap{
			"qualifiedChannel": futuresQualifiedChannel,
			"isPairChannel":    isFuturesPairChannel,
			"formatPair":       e.FormatExchangeCurrency,
		}).
		Parse(futuresSubTplText)
}

const futuresSubTplText = `
{{- range $asset, $pairs := $.AssetPairs }}
	{{- if isPairChannel $.S }}
		{{- range $p := $pairs }}
			{{- qualifiedChannel $.S $p }}
			{{- $.PairSeparator }}
		{{- end }}
	{{- else }}
		{{- with $p := index $pairs 0 }}{{ qualifiedChannel $.S $p }}{{ end }}
	{{- end }}
	{{- $.AssetSeparator }}
{{- end }}
`

// futuresPrivateChannelSubscribed reports whether a private futures channel is subscribed
func (e *Exchange) futuresPrivateChannelSubscribed(channel string) bool {
	return e.Websocket.GetSubscription(subscription.IgnoringPairsKey{Subscription: &subscription.Subscription{Channel: channel, Asset: asset.Futures}}) != nil
}

// WebsocketFuturesHandleData routes a frame from a futures connection
func (e *Exchange) WebsocketFuturesHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var resp *WebsocketFuturesResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	switch {
	case resp == nil:
		return e.wsUnhandled(ctx, respRaw)
	case resp.Channel == futuresChannelPong:
		return nil
	case strings.HasPrefix(resp.Channel, futuresChannelReplyPrefix):
		if conn.IncomingWithData(futuresWebsocketReplySignature, respRaw) {
			return nil
		}
		if resp.Channel == futuresChannelError {
			// No request is waiting, so the refusal is reported rather than lost
			return fmt.Errorf("%w: %s", errFuturesWebsocketRejected, resp.Data)
		}
		// A reply arriving after its request timed out is already reported by the request
		return nil
	case !strings.HasPrefix(resp.Channel, futuresChannelPushPrefix):
		return e.wsUnhandled(ctx, respRaw)
	}
	switch strings.TrimPrefix(resp.Channel, futuresChannelPushPrefix) {
	case futuresChannelTicker:
		return e.processFuturesTicker(ctx, resp.Data)
	case futuresChannelDeal:
		return e.processFuturesTrades(ctx, resp.Symbol, resp.Data)
	case futuresChannelDepthFull:
		return e.processFuturesOrderbook(ctx, resp.Symbol, resp.Timestamp.Time(), resp.Data)
	case futuresChannelKline:
		return e.processFuturesCandle(ctx, resp.Data)
	case futuresChannelOrder:
		if !e.futuresPrivateChannelSubscribed(subscription.MyOrdersChannel) {
			return nil
		}
		return e.processFuturesOrder(ctx, resp.Data)
	case futuresChannelOrderDeal:
		if !e.futuresPrivateChannelSubscribed(subscription.MyTradesChannel) {
			return nil
		}
		return e.processFuturesFill(ctx, resp.Data)
	case futuresChannelAsset:
		if !e.futuresPrivateChannelSubscribed(subscription.MyAccountChannel) {
			return nil
		}
		return e.processFuturesAsset(ctx, resp.Data)
	case futuresChannelPosition:
		if !e.futuresPrivateChannelSubscribed(FuturesPositionsChannel) {
			return nil
		}
		return e.processFuturesPosition(ctx, resp.Data)
	}
	// The venue pushes every private channel after a login, and those without a mapping are not subscribable
	if strings.HasPrefix(resp.Channel, futuresChannelPushPrefix+futuresPrivateChannelStart) {
		return nil
	}
	return e.wsUnhandled(ctx, respRaw)
}

// processFuturesTicker stores a contract's pushed ticker and relays it
func (e *Exchange) processFuturesTicker(ctx context.Context, data []byte) error {
	var t WebsocketFuturesTicker
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(t.Symbol, asset.Futures, true)
	if err != nil {
		return err
	}
	contract, err := e.futuresContract(ctx, t.Symbol)
	if err != nil {
		return err
	}
	baseVolume, quoteVolume := futuresVolumes(contract, t.Volume24Hour, t.Amount24Hour)
	tick := &ticker.Price{
		Pair:         pair,
		ExchangeName: e.Name,
		AssetType:    asset.Futures,
		Last:         t.LastPrice,
		High:         t.High24Hour,
		Low:          t.Low24Hour,
		Bid:          t.BestBidPrice,
		Ask:          t.BestAskPrice,
		BaseVolume:   baseVolume,
		QuoteVolume:  quoteVolume,
		OpenInterest: t.HoldVolume * contract.ContractSize,
		MarkPrice:    t.FairPrice,
		IndexPrice:   t.IndexPrice,
		LastUpdated:  t.Timestamp.Time(),
	}
	if err := ticker.ProcessTicker(tick); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, tick)
}

// processFuturesTrades relays a contract's pushed trades. The venue merges trades by default and pushes them as a
// list, or one at a time when merging is turned off; volumes are in contracts
func (e *Exchange) processFuturesTrades(ctx context.Context, symbol string, data []byte) error {
	saveTradeData := e.IsSaveTradeDataEnabled()
	tradeFeed := e.IsTradeFeedEnabled()
	if !saveTradeData && !tradeFeed {
		return nil
	}
	pair, err := e.MatchSymbolWithAvailablePairs(symbol, asset.Futures, true)
	if err != nil {
		return err
	}
	var deals oneOrMany[*FuturesTradeResponse]
	if err := json.Unmarshal(data, &deals); err != nil {
		return err
	}
	trades := make([]trade.Data, 0, len(deals))
	for _, d := range deals {
		if d == nil {
			continue
		}
		trades = append(trades, trade.Data{
			TID:          d.TradeID.String(),
			Exchange:     e.Name,
			CurrencyPair: pair,
			AssetType:    asset.Futures,
			Side:         futuresTradeSide(d.Side),
			Price:        d.Price,
			Amount:       d.Volume,
			Timestamp:    d.Time.Time(),
		})
	}
	if len(trades) == 0 {
		return nil
	}
	if tradeFeed {
		if err := e.Websocket.DataHandler.Send(ctx, trades); err != nil {
			return err
		}
	}
	if saveTradeData {
		// AddTradesToBuffer writes to the trades it is given, so it gets its own copy
		return trade.AddTradesToBuffer(slices.Clone(trades)...)
	}
	return nil
}

// processFuturesOrderbook loads a contract's pushed order book, a full snapshot of its levels in contracts
func (e *Exchange) processFuturesOrderbook(ctx context.Context, symbol string, pushed time.Time, data []byte) error {
	pair, err := e.MatchSymbolWithAvailablePairs(symbol, asset.Futures, true)
	if err != nil {
		return err
	}
	var depth FuturesDepthSnapshotResponse
	if err := json.Unmarshal(data, &depth); err != nil {
		return err
	}
	lastUpdated := depth.MatchingEngineTime.Time()
	if lastUpdated.IsZero() {
		lastUpdated = pushed
	}
	return e.Websocket.Orderbook.LoadSnapshot(ctx, &orderbook.Book{
		Exchange:          e.Name,
		Asset:             asset.Futures,
		Pair:              pair,
		Bids:              depth.Bids.Levels(),
		Asks:              depth.Asks.Levels(),
		LastUpdated:       lastUpdated,
		LastPushed:        pushed,
		LastUpdateID:      int64(min(depth.Version, math.MaxInt64)),
		ValidateOrderbook: e.ValidateOrderbook,
	})
}

// processFuturesCandle relays a contract's forming candle; its volume is restated from contracts in the base currency
func (e *Exchange) processFuturesCandle(ctx context.Context, data []byte) error {
	var c WebsocketFuturesCandle
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(c.Symbol, asset.Futures, true)
	if err != nil {
		return err
	}
	interval, err := futuresIntervalFromString(c.Interval)
	if err != nil {
		return err
	}
	contract, err := e.futuresContract(ctx, c.Symbol)
	if err != nil {
		return err
	}
	baseVolume, _ := futuresVolumes(contract, c.Volume, c.Amount)
	return e.Websocket.DataHandler.Send(ctx, kline.Item{
		Pair:     pair,
		Exchange: e.Name,
		Asset:    asset.Futures,
		Interval: interval,
		Candles: []kline.Candle{{
			Time:   c.StartTime.Time(),
			Open:   c.Open,
			High:   c.High,
			Low:    c.Low,
			Close:  c.Close,
			Volume: baseVolume,
			// The venue pushes a window repeatedly while it is open and sends no closing frame
			ValidationIssues: kline.PartialCandle,
		}},
	})
}

// processFuturesOrder relays a pushed futures order, whose fees are the order's so far
func (e *Exchange) processFuturesOrder(ctx context.Context, data []byte) error {
	var o FuturesOrderResponse
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	detail, err := e.futuresOrderDetail(ctx, &o, currency.EMPTYPAIR)
	if err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &detail)
}

// processFuturesFill relays a pushed fill of one of the account's futures orders; its amount is in contracts
func (e *Exchange) processFuturesFill(ctx context.Context, data []byte) error {
	var d FuturesOrderDealResponse
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	pair, err := e.futuresSymbolPair(d.Symbol)
	if err != nil {
		return err
	}
	side, _ := orderSideFromFuturesSide(d.Side)
	return e.Websocket.DataHandler.Send(ctx, []fill.Data{{
		ID:            d.ID.String(),
		TradeID:       d.ID.String(),
		Timestamp:     d.Timestamp.Time(),
		Exchange:      e.Name,
		AssetType:     asset.Futures,
		CurrencyPair:  pair,
		Side:          side,
		OrderID:       d.OrderID.String(),
		ClientOrderID: d.ExternalOrderID,
		Price:         d.Price,
		Amount:        d.Volume,
	}})
}

// processFuturesAsset stores a pushed futures account balance and relays it
func (e *Exchange) processFuturesAsset(ctx context.Context, data []byte) error {
	var a WebsocketFuturesAsset
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if a.Currency.IsEmpty() {
		return fmt.Errorf("%w: futures balance push", currency.ErrCurrencyCodeEmpty)
	}
	pushed := futuresBalance(a.AvailableBalance, a.FrozenBalance, a.PositionMargin)
	balance, err := e.Accounts.UpdateBalance(ctx, "", asset.Futures, a.Currency, func(b *accounts.Balance) {
		b.Total, b.Hold, b.Free = pushed.Total, pushed.Hold, pushed.Free
	})
	if err != nil {
		return err
	}
	subAccount := accounts.NewSubAccount(asset.Futures, "")
	subAccount.Balances.Set(a.Currency, balance)
	return e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAccount})
}

// processFuturesPosition relays a pushed futures position; sizes are in contracts
func (e *Exchange) processFuturesPosition(ctx context.Context, data []byte) error {
	var p FuturesPositionResponse
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	pair, err := e.futuresSymbolPair(p.Symbol)
	if err != nil {
		return err
	}
	status := order.Open
	var closeDate time.Time
	if p.State == FuturesPositionStateClosed {
		status, closeDate = order.Closed, p.UpdateTime.Time()
	}
	direction := futuresPositionSide(p.PositionType)
	position := futures.Position{
		Exchange:                  e.Name,
		Asset:                     asset.Futures,
		Pair:                      pair,
		Underlying:                pair.Base,
		Leverage:                  decimal.NewFromInt(int64(min(p.Leverage, math.MaxInt64))),
		PositionMargin:            decimal.MustFromFloat(p.InitialMargin),
		InitialMarginRequirement:  decimal.MustFromFloat(p.InitialMargin),
		EstimatedLiquidationPrice: decimal.MustFromFloat(p.LiquidatePrice),
		UpdateID:                  int64(min(p.Version, math.MaxInt64)),
		RealisedPNL:               decimal.MustFromFloat(p.RealisedPNL),
		UnrealisedPNL:             decimal.MustFromFloat(p.unrealisedPNL()),
		Status:                    status,
		OpeningDate:               p.CreateTime.Time(),
		OpeningPrice:              decimal.MustFromFloat(p.OpenAveragePrice),
		OpeningDirection:          direction,
		LatestSize:                decimal.MustFromFloat(p.HoldVolume),
		LatestDirection:           direction,
		LastUpdated:               p.UpdateTime.Time(),
		CloseDate:                 closeDate,
	}
	if contract, err := e.futuresContract(ctx, p.Symbol); err == nil {
		position.CollateralCurrency = contract.SettleCoin
	}
	return e.Websocket.DataHandler.Send(ctx, []futures.Position{position})
}
