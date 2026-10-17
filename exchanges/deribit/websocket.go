package deribit

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	// websocketHeartbeatInterval has Deribit send a heartbeat every 15 seconds, so a stale connection is noticed well
	// within the traffic timeout
	websocketHeartbeatInterval = 15 * time.Second

	wsMethodSubscription = "subscription"
	wsMethodHeartbeat    = "heartbeat"
	// wsHeartbeatTestRequest is the heartbeat type that Deribit closes the connection for unless a public/test request
	// answers it
	wsHeartbeatTestRequest = "test_request"

	// Channel intervals; raw is only available to an authenticated connection, and agg2 aggregates about a second
	intervalRaw       = "raw"
	intervalHundredMS = "100ms"
	intervalAgg2      = "agg2"

	// Values a kind or currency channel parameter takes to select every kind or currency
	channelAny = "any"

	// Book channel notification types
	bookSnapshot = "snapshot"
	bookChange   = "change"
	bookDelete   = "delete"
)

// Deribit's subscription channels, which a subscription names, other than through a standard channel, to subscribe to
const (
	ChannelPlatformState              = "platform_state"
	ChannelPlatformPublicMethodsState = "platform_state.public_methods_state"
	ChannelAnnouncements              = "announcements"
	ChannelBook                       = "book"
	ChannelTicker                     = "ticker"
	ChannelIncrementalTicker          = "incremental_ticker"
	ChannelPerpetual                  = "perpetual"
	ChannelQuote                      = "quote"
	ChannelTrades                     = "trades"
	ChannelBlockTradeConfirmations    = "block_trade_confirmations"
	ChannelPriceIndex                 = "deribit_price_index"
	ChannelPriceRanking               = "deribit_price_ranking"
	ChannelPriceStatistics            = "deribit_price_statistics"
	ChannelVolatilityIndex            = "deribit_volatility_index"
	ChannelMarkPriceOptions           = "markprice.options"
	ChannelEstimatedExpirationPrice   = "estimated_expiration_price"
	ChannelChartTrades                = "chart.trades"
	ChannelInstrumentCreation         = "instrument.creation"
	ChannelInstrumentState            = "instrument.state"
	ChannelUserMMPTrigger             = "user.mmp_trigger"
	ChannelUserPortfolio              = "user.portfolio"
	ChannelUserTrades                 = "user.trades"
	ChannelUserComboTrades            = "user.combo_trades"
	ChannelUserOrders                 = "user.orders"
	ChannelUserChanges                = "user.changes"
	ChannelUserAccessLog              = "user.access_log"
	ChannelUserLock                   = "user.lock"
	ChannelUserLiquidation            = "user.liquidation"
	ChannelUserIsolatedLiquidation    = "user.isolated.liquidation"
	ChannelUserLSP                    = "user.lsp"
	ChannelBlockRFQMaker              = "block_rfq.maker"
	ChannelBlockRFQTaker              = "block_rfq.taker"
	ChannelBlockRFQMakerQuotes        = "block_rfq.maker.quotes"
	ChannelBlockRFQTrades             = "block_rfq.trades"
)

var (
	errNoTickerSnapshot            = errors.New("incremental ticker change received before its snapshot")
	errUnsupportedInstrumentFormat = errors.New("unsupported instrument format")
	errChannelParameter            = errors.New("invalid channel parameter")
	errChannelNeedsAuthentication  = errors.New("channel needs an authenticated subscription")
	errChannelNeedsInstrument      = errors.New("channel needs an asset and pairs")
	errOrderbookOutOfSequence      = errors.New("orderbook change out of sequence")
	errMalformedNotification       = errors.New("malformed notification")
)

// websocketChannel describes how a Deribit channel is qualified
type websocketChannel struct {
	// instrument channels may be qualified per pair by its instrument name, for a subscription with an asset
	instrument bool
	// kindCurrency channels may be qualified by the kind and currency parameters, each any when unset, for a
	// subscription without an asset
	kindCurrency bool
	// interval channels end with the subscription's interval: 100ms, raw (one millisecond) or agg2 (one second)
	interval bool
	// resolution channels end with the subscription's interval as a candle resolution
	resolution bool
	// indexName channels are qualified by the index_name parameter, such as btc_usd
	indexName bool
	// currency channels are qualified by the currency parameter, any when unset; optionalCurrency channels only when it
	// is set
	currency         bool
	optionalCurrency bool
	// private channels need an authenticated subscription
	private bool
}

// websocketChannels lists every Deribit channel a subscription may name
var websocketChannels = map[string]websocketChannel{
	ChannelPlatformState:              {},
	ChannelPlatformPublicMethodsState: {},
	ChannelAnnouncements:              {},
	ChannelBook:                       {instrument: true, interval: true},
	ChannelTicker:                     {instrument: true, interval: true},
	ChannelIncrementalTicker:          {instrument: true},
	ChannelPerpetual:                  {instrument: true, interval: true},
	ChannelQuote:                      {instrument: true},
	ChannelTrades:                     {instrument: true, kindCurrency: true, interval: true},
	ChannelBlockTradeConfirmations:    {optionalCurrency: true, private: true},
	ChannelPriceIndex:                 {indexName: true},
	ChannelPriceRanking:               {indexName: true},
	ChannelPriceStatistics:            {indexName: true},
	ChannelVolatilityIndex:            {indexName: true},
	ChannelMarkPriceOptions:           {indexName: true},
	ChannelEstimatedExpirationPrice:   {indexName: true},
	ChannelChartTrades:                {instrument: true, resolution: true},
	ChannelInstrumentCreation:         {kindCurrency: true},
	ChannelInstrumentState:            {kindCurrency: true},
	ChannelUserMMPTrigger:             {indexName: true, private: true},
	ChannelUserPortfolio:              {currency: true, private: true},
	ChannelUserTrades:                 {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserComboTrades:            {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserOrders:                 {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserChanges:                {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelUserAccessLog:              {private: true},
	ChannelUserLock:                   {private: true},
	ChannelUserLiquidation:            {private: true},
	ChannelUserIsolatedLiquidation:    {private: true},
	ChannelUserLSP:                    {private: true},
	ChannelBlockRFQMaker:              {currency: true, private: true},
	ChannelBlockRFQTaker:              {currency: true, private: true},
	ChannelBlockRFQMakerQuotes:        {currency: true, private: true},
	ChannelBlockRFQTrades:             {currency: true},
}

// websocketStandardChannels maps standard channels to the Deribit channels that serve them
var websocketStandardChannels = map[string]string{
	subscription.TickerChannel:    ChannelTicker,
	subscription.OrderbookChannel: ChannelBook,
	subscription.AllTradesChannel: ChannelTrades,
	subscription.CandlesChannel:   ChannelChartTrades,
	subscription.MyOrdersChannel:  ChannelUserOrders,
	subscription.MyTradesChannel:  ChannelUserTrades,
	subscription.MyAccountChannel: ChannelUserPortfolio,
}

// websocketChannelsByLength lists the channel names longest first, so a notification's channel is matched to the most
// specific name it starts with
var websocketChannelsByLength = func() []string {
	names := slices.Collect(maps.Keys(websocketChannels))
	slices.SortFunc(names, func(a, b string) int { return cmp.Or(len(b)-len(a), strings.Compare(a, b)) })
	return names
}()

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneDay},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}, // Raw is available for authenticated users
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel, Interval: kline.HundredMilliseconds},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Interval: kline.HundredMilliseconds, Authenticated: true},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyTradesChannel, Interval: kline.HundredMilliseconds, Authenticated: true},
}

// WebsocketAPI calls Deribit's methods over the websocket connection rather than REST; a method with a REST form takes
// that form's name and arguments
type WebsocketAPI struct {
	e *Exchange
}

// WebsocketAPI returns the methods called over the websocket connection
func (e *Exchange) WebsocketAPI() WebsocketAPI {
	return WebsocketAPI{e: e}
}

// WebsocketConnect connects the websocket, asks Deribit for heartbeats and authenticates the connection when
// credentials are set
func (e *Exchange) WebsocketConnect() error {
	ctx := context.TODO()
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	if err := e.Websocket.Conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(ctx)
	if err := e.WebsocketAPI().SetHeartbeat(ctx, websocketHeartbeatInterval); err != nil {
		log.Errorf(log.WebsocketMgr, "%s websocket heartbeat could not be set: %s", e.Name, err)
	}
	if e.Websocket.CanUseAuthenticatedEndpoints() {
		if err := e.wsLogin(ctx); err != nil {
			log.Errorf(log.WebsocketMgr, "%s websocket authentication failed: %s", e.Name, err)
			e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	return nil
}

// wsLogin authenticates the websocket connection with a client signature, so the secret is not sent
func (e *Exchange) wsLogin(ctx context.Context) error {
	if !e.IsWebsocketAuthenticationSupported() {
		return exchange.ErrAuthenticationSupportNotEnabled
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	arg, err := NewClientSignatureRequest(creds, time.Now(), e.Requester.GetNonce(nonce.UnixNano).String(), "")
	if err != nil {
		return err
	}
	if _, err := e.WebsocketAPI().Authenticate(ctx, arg); err != nil {
		e.Websocket.SetCanUseAuthenticatedEndpoints(false)
		return err
	}
	return nil
}

// SendWebsocketRequest sends a JSON-RPC request over the websocket and decodes its result; a private method needs the
// connection to be authenticated
func (e *Exchange) SendWebsocketRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	private := isPrivateMethod(method)
	if private && !e.Websocket.CanUseAuthenticatedEndpoints() {
		return fmt.Errorf("%w: %s", errWebsocketNotAuthenticated, method)
	}
	if !e.websocketDialled() {
		return fmt.Errorf("%w: %s", websocket.ErrNotConnected, method)
	}
	if private {
		var err error
		if params, err = authorisedParams(ctx, params); err != nil {
			return err
		}
	}
	return retryRequest(ctx, func() error {
		if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
			return err
		}
		req := &JSONRPCRequest{JSONRPC: jsonRPCVersion, ID: e.MessageSequence(), Method: method, Params: params}
		raw, err := e.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
		if err != nil {
			return err
		}
		resp := &JSONRPCResponse{Result: result}
		var challenged privateResult
		if private {
			challenged.result = result
			resp.Result = &challenged
		}
		if err := json.Unmarshal(raw, resp); err != nil {
			return err
		}
		if resp.Error != nil {
			return resp.Error
		}
		if challenged.challenge != nil {
			return challenged.challenge
		}
		return nil
	})
}

// websocketDialled reports whether the websocket connection is dialled; the manager only reports itself connected once
// WebsocketConnect returns, while WebsocketConnect sends its own requests
func (e *Exchange) websocketDialled() bool {
	conn, ok := e.Websocket.Conn.(interface{ IsConnected() bool })
	return ok && conn.IsConnected()
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

// wsHandleData passes a response to the request awaiting it, answers a heartbeat test request and processes a
// subscription notification
func (e *Exchange) wsHandleData(ctx context.Context, respRaw []byte) error {
	var envelope wsEnvelope
	if err := json.Unmarshal(respRaw, &envelope); err != nil {
		return fmt.Errorf("%s could not parse websocket data: %w: %s", e.Name, err, respRaw)
	}
	if envelope.ID != 0 {
		return e.Websocket.Match.RequireMatchWithData(envelope.ID, respRaw)
	}
	switch envelope.Method {
	case wsMethodHeartbeat:
		if envelope.Params.Type == wsHeartbeatTestRequest {
			go e.wsAnswerHeartbeat(ctx)
		}
		return nil
	case wsMethodSubscription:
		return e.wsHandleNotification(ctx, envelope.Params.Channel, respRaw)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// wsAnswerHeartbeat answers a heartbeat test request with a public/test request, without which Deribit closes the
// connection
func (e *Exchange) wsAnswerHeartbeat(ctx context.Context) {
	if _, err := e.WebsocketAPI().TestConnection(ctx, ""); err != nil {
		log.Errorf(log.WebsocketMgr, "%s websocket heartbeat test request could not be answered: %s", e.Name, err)
	}
}

// wsHandleNotification passes a subscription notification to its channel's handler
func (e *Exchange) wsHandleNotification(ctx context.Context, channel string, respRaw []byte) error {
	switch notificationChannelName(channel) {
	case ChannelBook:
		return e.wsHandleOrderbook(ctx, channel, respRaw)
	case ChannelIncrementalTicker:
		return e.wsHandleIncrementalTicker(ctx, channel, respRaw)
	case ChannelChartTrades:
		return e.wsHandleCandle(ctx, channel, respRaw)
	case ChannelTicker:
		return e.wsHandleTicker(ctx, respRaw)
	case ChannelTrades:
		return e.wsHandleTrades(ctx, respRaw)
	case ChannelUserOrders:
		return e.wsHandleOrders(ctx, channel, respRaw)
	case ChannelUserTrades, ChannelUserComboTrades:
		return e.wsHandleUserTrades(ctx, respRaw)
	case ChannelPerpetual:
		return wsRelay[WebsocketPerpetualInterest](ctx, e, respRaw)
	case ChannelQuote:
		return wsRelay[WebsocketQuote](ctx, e, respRaw)
	case ChannelPriceIndex:
		return wsRelay[WebsocketPriceIndex](ctx, e, respRaw)
	case ChannelPriceRanking:
		return wsRelay[[]WebsocketPriceRanking](ctx, e, respRaw)
	case ChannelPriceStatistics:
		return wsRelay[WebsocketPriceStatistics](ctx, e, respRaw)
	case ChannelVolatilityIndex:
		return wsRelay[WebsocketVolatilityIndex](ctx, e, respRaw)
	case ChannelMarkPriceOptions:
		return wsRelay[[]WebsocketMarkPriceOption](ctx, e, respRaw)
	case ChannelEstimatedExpirationPrice:
		return wsRelay[WebsocketEstimatedExpirationPrice](ctx, e, respRaw)
	case ChannelInstrumentState:
		return wsRelay[WebsocketInstrumentState](ctx, e, respRaw)
	case ChannelPlatformState:
		return wsRelay[WebsocketPlatformState](ctx, e, respRaw)
	case ChannelPlatformPublicMethodsState:
		return wsRelay[WebsocketPublicMethodsState](ctx, e, respRaw)
	case ChannelUserLock:
		return wsRelay[WebsocketUserLock](ctx, e, respRaw)
	case ChannelUserLiquidation, ChannelUserIsolatedLiquidation:
		return wsRelay[WebsocketUserLiquidation](ctx, e, respRaw)
	case ChannelUserMMPTrigger:
		return wsRelay[WebsocketMMPTrigger](ctx, e, respRaw)
	case ChannelAnnouncements:
		return wsRelay[WebsocketAnnouncement](ctx, e, respRaw)
	case ChannelInstrumentCreation:
		return wsRelay[WebsocketInstrumentCreation](ctx, e, respRaw)
	case ChannelUserPortfolio:
		return wsRelay[AccountSummary](ctx, e, respRaw)
	case ChannelUserChanges:
		return wsRelay[WebsocketUserChanges](ctx, e, respRaw)
	case ChannelUserAccessLog:
		return wsRelay[AccessLogEntry](ctx, e, respRaw)
	case ChannelUserLSP:
		return wsRelay[WebsocketLSPEvent](ctx, e, respRaw)
	case ChannelBlockTradeConfirmations:
		return wsRelay[PendingBlockTrade](ctx, e, respRaw)
	case ChannelBlockRFQMaker, ChannelBlockRFQTaker:
		return wsRelay[BlockRFQ](ctx, e, respRaw)
	case ChannelBlockRFQMakerQuotes:
		return wsRelay[WebsocketBlockRFQQuotes](ctx, e, respRaw)
	case ChannelBlockRFQTrades:
		return wsRelay[BlockRFQTrade](ctx, e, respRaw)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// notificationChannelName returns the Deribit channel a notification's qualified channel belongs to, or an empty string
// for an unknown channel
func notificationChannelName(channel string) string {
	for _, name := range websocketChannelsByLength {
		if channel == name || strings.HasPrefix(channel, name+".") {
			return name
		}
	}
	return ""
}

// decodeNotification decodes a subscription notification's data into the value data points to
func decodeNotification(respRaw []byte, data any) error {
	notification := wsNotification{Params: wsNotificationParams{Data: data}}
	if err := json.Unmarshal(respRaw, &notification); err != nil {
		return fmt.Errorf("%w: %w", errMalformedNotification, err)
	}
	return nil
}

// wsRelay decodes a notification's data into its exported type and relays it
func wsRelay[T any](ctx context.Context, e *Exchange, respRaw []byte) error {
	data := new(T)
	if err := decodeNotification(respRaw, data); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, data)
}

// generateSubscriptions returns the configured subscriptions with their channels qualified
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	subs, err := e.Features.Subscriptions.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(subs, e.isUnstreamedSubscription), nil
}

// isUnstreamedSubscription reports whether a subscription is to the trades or candles of spot Deribit routes to Coinbase
// Exchange, which Deribit refuses
func (e *Exchange) isUnstreamedSubscription(sub *subscription.Subscription) bool {
	if sub.Asset != asset.Spot || len(sub.Pairs) != 1 {
		return false
	}
	if name, err := websocketChannelName(sub); err != nil || (name != ChannelTrades && name != ChannelChartTrades) {
		return false
	}
	return e.isCoinbaseRoutedSpot(formatPairString(asset.Spot, sub.Pairs[0]))
}

// GetSubscriptionTemplate returns the channel qualification template, which qualifies an instrument channel per pair
// for a subscription with an asset and any other channel once
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":       websocketChannelName,
		"perInstrument":     func(name string) bool { return websocketChannels[name].instrument },
		"qualifyInstrument": qualifyInstrumentChannel,
		"qualify":           qualifyChannel,
	}).Parse(websocketSubscriptionTemplate)
}

// websocketSubscriptionTemplate qualifies a subscription per pair for instrument channels and once otherwise
const websocketSubscriptionTemplate = `
{{- $name := channelName $.S }}
{{- if and $.S.Asset (perInstrument $name) }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- range $pair := $pairs }}
			{{- qualifyInstrument $.S $asset $pair }}
			{{- $.PairSeparator }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- else }}
	{{- qualify $.S }}
{{- end }}
`

// websocketChannelName returns the Deribit channel a subscription refers to: a standard channel's Deribit channel, or
// the channel it names
func websocketChannelName(sub *subscription.Subscription) (string, error) {
	if sub == nil {
		return "", common.ErrNilPointer
	}
	if name, ok := websocketStandardChannels[sub.Channel]; ok {
		return name, nil
	}
	if _, ok := websocketChannels[sub.Channel]; ok {
		return sub.Channel, nil
	}
	return "", fmt.Errorf("%w: %s", subscription.ErrNotSupported, sub.Channel)
}

// qualifyInstrumentChannel returns a subscription's channel for one of its pairs
func qualifyInstrumentChannel(sub *subscription.Subscription, a asset.Item, pair currency.Pair) (string, error) {
	name, channel, err := validateChannelSubscription(sub)
	if err != nil {
		return "", err
	}
	qualified := name + "." + formatPairString(a, pair)
	if name == ChannelBook && sub.Levels != 0 {
		group, err := bookGroup(sub.Params)
		if err != nil {
			return "", err
		}
		switch sub.Levels {
		case 1, 10, 20:
		default:
			return "", fmt.Errorf("%w: %d, as a grouped book sends 1, 10 or 20 levels", subscription.ErrInvalidLevel, sub.Levels)
		}
		qualified += "." + group + "." + strconv.Itoa(sub.Levels)
	}
	return appendChannelInterval(qualified, sub, &channel)
}

// qualifyChannel returns a subscription's channel when it is not qualified per pair
func qualifyChannel(sub *subscription.Subscription) (string, error) {
	name, channel, err := validateChannelSubscription(sub)
	if err != nil {
		return "", err
	}
	qualified := name
	switch {
	case channel.kindCurrency:
		kind, err := channelParameter(sub.Params, "kind", channelAny)
		if err != nil {
			return "", err
		}
		ccy, err := channelCurrency(sub.Params, true)
		if err != nil {
			return "", err
		}
		qualified += "." + kind + "." + ccy
	case channel.instrument:
		return "", fmt.Errorf("%w: %s", errChannelNeedsInstrument, name)
	case channel.indexName:
		index, err := channelParameter(sub.Params, "index_name", "")
		if err != nil {
			return "", err
		}
		qualified += "." + index
	case channel.currency:
		ccy, err := channelCurrency(sub.Params, true)
		if err != nil {
			return "", err
		}
		qualified += "." + ccy
	case channel.optionalCurrency:
		ccy, err := channelCurrency(sub.Params, false)
		if err != nil {
			return "", err
		}
		if ccy != "" {
			qualified += "." + ccy
		}
	}
	return appendChannelInterval(qualified, sub, &channel)
}

// validateChannelSubscription returns the Deribit channel a subscription refers to, checking a private channel is
// subscribed authenticated
func validateChannelSubscription(sub *subscription.Subscription) (string, websocketChannel, error) {
	name, err := websocketChannelName(sub)
	if err != nil {
		return "", websocketChannel{}, err
	}
	channel := websocketChannels[name]
	if channel.private && !sub.Authenticated {
		return "", websocketChannel{}, fmt.Errorf("%w: %s", errChannelNeedsAuthentication, name)
	}
	return name, channel, nil
}

// appendChannelInterval appends the subscription's interval to a channel that takes one; Deribit rejects a whole
// subscribe request holding a raw channel from an unauthenticated connection, so a raw channel must be subscribed
// authenticated
func appendChannelInterval(qualified string, sub *subscription.Subscription, channel *websocketChannel) (string, error) {
	switch {
	case channel.resolution:
		resolution, err := chartResolution(sub.Interval)
		if err != nil {
			return "", err
		}
		return qualified + "." + resolution, nil
	case channel.interval:
		var interval string
		switch sub.Interval {
		case kline.HundredMilliseconds:
			interval = intervalHundredMS
		case kline.ThousandMilliseconds:
			interval = intervalAgg2
		case kline.Raw:
			if !sub.Authenticated {
				return "", fmt.Errorf("%w: %s raw interval", errChannelNeedsAuthentication, qualified)
			}
			interval = intervalRaw
		default:
			return "", fmt.Errorf("%w: %s, as Deribit aggregates over 100ms, 1s or raw", kline.ErrUnsupportedInterval, sub.Interval)
		}
		return qualified + "." + interval, nil
	}
	return qualified, nil
}

// chartResolution returns the chart.trades resolution of an interval
func chartResolution(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin, kline.ThreeMin, kline.FiveMin, kline.TenMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.ThreeHour, kline.SixHour, kline.TwelveHour:
		return strconv.FormatInt(int64(interval.Duration()/time.Minute), 10), nil
	case kline.OneDay:
		return "1D", nil
	default:
		return "", fmt.Errorf("%w: %s, as chart.trades serves 1m to 12h and 1D", kline.ErrUnsupportedInterval, interval)
	}
}

// bookGroup returns a grouped book's price grouping from the group parameter, none when unset; a number, as
// configuration supplies it, is formatted as Deribit names it
func bookGroup(params map[string]any) (string, error) {
	var group string
	switch value := params["group"].(type) {
	case nil:
		return "none", nil
	case string:
		group = value
	case float64:
		group = strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		group = strconv.Itoa(value)
	default:
		return "", fmt.Errorf("%w: group %v", errChannelParameter, value)
	}
	switch group {
	case "none", "1", "2", "5", "10", "25", "100", "250":
		return group, nil
	default:
		return "", fmt.Errorf("%w: group %q", errChannelParameter, group)
	}
}

// channelParameter returns a string channel parameter, or the default when it is unset; a parameter without a default
// is required
func channelParameter(params map[string]any, name, defaultValue string) (string, error) {
	switch value := params[name].(type) {
	case nil:
		if defaultValue == "" {
			return "", fmt.Errorf("%w: %s is required", errChannelParameter, name)
		}
		return defaultValue, nil
	case string:
		if value == "" || strings.Contains(value, ".") {
			return "", fmt.Errorf("%w: %s %q", errChannelParameter, name, value)
		}
		return value, nil
	default:
		return "", fmt.Errorf("%w: %s %v", errChannelParameter, name, value)
	}
}

// channelCurrency returns the currency parameter, upper case as Deribit names currencies, or any; when required an
// unset currency is any, otherwise empty
func channelCurrency(params map[string]any, required bool) (string, error) {
	var ccy string
	switch value := params["currency"].(type) {
	case nil:
		if required {
			return channelAny, nil
		}
		return "", nil
	case string:
		ccy = value
	case currency.Code:
		ccy = value.String()
	default:
		return "", fmt.Errorf("%w: currency %v", errChannelParameter, value)
	}
	if strings.EqualFold(ccy, channelAny) {
		return channelAny, nil
	}
	if ccy == "" || strings.Contains(ccy, ".") {
		return "", fmt.Errorf("%w: currency %q", errChannelParameter, ccy)
	}
	return strings.ToUpper(ccy), nil
}

// Subscribe subscribes the websocket to channels, through private/subscribe for private channels
func (e *Exchange) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	errs := e.handleSubscription(ctx, true, false, subs.Public())
	return common.AppendError(errs, e.handleSubscription(ctx, true, true, subs.Private()))
}

// Unsubscribe unsubscribes the websocket from channels, through private/unsubscribe for private channels
func (e *Exchange) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	errs := e.handleSubscription(ctx, false, false, subs.Public())
	return common.AppendError(errs, e.handleSubscription(ctx, false, true, subs.Private()))
}

// handleSubscription subscribes to or unsubscribes from channels, then stores or removes each channel Deribit
// acknowledged and reports the rest
func (e *Exchange) handleSubscription(ctx context.Context, subscribe, private bool, subs subscription.List) error {
	subs, err := subs.ExpandTemplates(e)
	if err != nil || len(subs) == 0 {
		return err
	}
	channels := subs.QualifiedChannels()
	var acknowledged []string
	switch {
	case subscribe && private:
		acknowledged, err = e.WebsocketAPI().SubscribePrivate(ctx, &PrivateSubscribeRequest{Channels: channels})
	case subscribe:
		acknowledged, err = e.WebsocketAPI().Subscribe(ctx, channels)
	case private:
		acknowledged, err = e.WebsocketAPI().UnsubscribePrivate(ctx, channels)
	default:
		acknowledged, err = e.WebsocketAPI().Unsubscribe(ctx, channels)
	}
	if err != nil {
		return err
	}
	var errs error
	for _, s := range subs {
		if !slices.Contains(acknowledged, s.QualifiedChannel) {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", errUnacknowledgedSubscription, s.QualifiedChannel))
			continue
		}
		if subscribe {
			errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(e.Websocket.Conn, s))
		} else {
			errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(e.Websocket.Conn, s))
		}
	}
	return errs
}
