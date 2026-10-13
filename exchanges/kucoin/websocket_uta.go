package kucoin

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// UTA websocket hosts
const (
	kucoinUTASpotPublicWebsocketURL    = "wss://x-push-spot.kucoin.com"
	kucoinUTAFuturesPublicWebsocketURL = "wss://x-push-futures.kucoin.com"
	kucoinUTAPrivateWebsocketURL       = "wss://wsapi-push.kucoin.com"
	kucoinUTATradeWebsocketURL         = "wss://wsapi.kucoin.com/v2/private"
)

// Message filters of the UTA websocket connections
const (
	utaSpotPublicConnection    = "uta-spot-public"
	utaFuturesPublicConnection = "uta-futures-public"
	utaPrivateConnection       = "uta-private"
	utaTradeConnection         = "uta-trade"
)

// UTA websocket channels, as named in an exchange's subscription configuration. The uta. prefix keeps them apart from
// the Classic channels sharing that configuration; the rest of the name is KuCoin's channel name
const (
	UTATickerChannel             = "uta.ticker"
	UTAKlineChannel              = "uta.kline"
	UTATradeChannel              = "uta.trade"
	UTAOrderbookChannel          = "uta.obu"
	UTAMarkPriceChannel          = "uta.mark-price"
	UTAFundingFeeChannel         = "uta.funding-fee"
	UTAAllFundingFeeRatesChannel = "uta.funding-fee-all-symbols"
	UTACallAuctionInfoChannel    = "uta.callAuctionInfo"
	UTAAllOrdersChannel          = "uta.orderAll"
	UTAOrderChannel              = "uta.order"
	UTAExecutionLiteChannel      = "uta.execution.lite"
	UTAExecutionChannel          = "uta.execution"
	UTABalanceChannel            = "uta.balance"
	UTAAllPositionsChannel       = "uta.positionAll"
	UTAPositionChannel           = "uta.position"
	UTALiquidationWarningChannel = "uta.lw"
	UTALeverageChannel           = "uta.leverage"
)

const (
	utaSubscribeAction    = "SUBSCRIBE"
	utaUnsubscribeAction  = "UNSUBSCRIBE"
	utaUnifiedAccountType = "UNIFIED"
	// utaPingInterval is the interval every UTA welcome frame recommends
	utaPingInterval = 18 * time.Second
)

var utaPingMessage = []byte(`{"id":"ping","op":"ping"}`)

// UTA websocket errors
var (
	errUTAAuthenticationFailed = errors.New("UTA websocket authentication failed")
	errUTARequestRejected      = errors.New("UTA websocket request rejected")
	errUTAUnknownTopic         = errors.New("unknown UTA websocket topic")
	errUTAUnexpectedMessage    = errors.New("unexpected UTA websocket message")
)

// utaChannel describes how a UTA channel is subscribed to
type utaChannel struct {
	name      string // KuCoin's channel name
	private   bool   // served by the private push connection
	perSymbol bool   // subscribed to one symbol at a time
	spot      bool   // served for spot and margin symbols
	futures   bool   // served for futures symbols
	tradeType bool   // a public channel whose subscribe message names the market
}

var utaChannels = map[string]utaChannel{
	UTATickerChannel:             {name: "ticker", perSymbol: true, spot: true, futures: true, tradeType: true},
	UTAKlineChannel:              {name: "kline", perSymbol: true, spot: true, futures: true, tradeType: true},
	UTATradeChannel:              {name: "trade", perSymbol: true, spot: true, futures: true, tradeType: true},
	UTAOrderbookChannel:          {name: "obu", perSymbol: true, spot: true, futures: true, tradeType: true},
	UTAMarkPriceChannel:          {name: "mark-price", perSymbol: true, futures: true},
	UTAFundingFeeChannel:         {name: "funding-fee", perSymbol: true, futures: true},
	UTAAllFundingFeeRatesChannel: {name: "funding-fee-all-symbols", futures: true},
	UTACallAuctionInfoChannel:    {name: "callAuctionInfo", perSymbol: true, spot: true},
	UTAAllOrdersChannel:          {name: "orderAll", private: true},
	UTAOrderChannel:              {name: "order", private: true, perSymbol: true, spot: true, futures: true},
	UTAExecutionLiteChannel:      {name: "execution.lite", private: true},
	UTAExecutionChannel:          {name: "execution", private: true},
	UTABalanceChannel:            {name: "balance", private: true},
	UTAAllPositionsChannel:       {name: "positionAll", private: true},
	UTAPositionChannel:           {name: "position", private: true, perSymbol: true, futures: true},
	UTALiquidationWarningChannel: {name: "lw", private: true},
	UTALeverageChannel:           {name: "leverage", private: true},
}

var utaKlineIntervals = map[kline.Interval]string{
	kline.OneMin: "1min", kline.ThreeMin: "3min", kline.FiveMin: "5min", kline.FifteenMin: "15min", kline.ThirtyMin: "30min",
	kline.OneHour: "1hour", kline.TwoHour: "2hour", kline.FourHour: "4hour", kline.SixHour: "6hour", kline.EightHour: "8hour",
	kline.TwelveHour: "12hour", kline.OneDay: "1day", kline.OneWeek: "1week", kline.OneMonth: "1month",
}

// newUTASpotPublicConnection returns the setup of the UTA spot and margin public push connection
func (e *Exchange) newUTASpotPublicConnection(exch *config.Exchange, wsURL string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   wsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		ConnectionRateLimiter: newUTAPublicConnectionLimiter,
		Connector:             e.UTAWebsocketConnect,
		GenerateSubscriptions: func() (subscription.List, error) { return e.generateUTASubscriptions(utaSpotPublicConnection) },
		Subscriber:            e.UTASubscribe,
		Unsubscriber:          e.UTAUnsubscribe,
		Handler:               e.wsHandleUTAData,
		MessageFilter:         utaSpotPublicConnection,
	}
}

// newUTAFuturesPublicConnection returns the setup of the UTA futures public push connection
func (e *Exchange) newUTAFuturesPublicConnection(exch *config.Exchange, wsURL string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   wsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		ConnectionRateLimiter: newUTAPublicConnectionLimiter,
		Connector:             e.UTAWebsocketConnect,
		GenerateSubscriptions: func() (subscription.List, error) { return e.generateUTASubscriptions(utaFuturesPublicConnection) },
		Subscriber:            e.UTASubscribe,
		Unsubscriber:          e.UTAUnsubscribe,
		Handler:               e.wsHandleUTAData,
		MessageFilter:         utaFuturesPublicConnection,
	}
}

// newUTAPrivateConnection returns the setup of the UTA private push connection, which signs in before subscribing
func (e *Exchange) newUTAPrivateConnection(exch *config.Exchange, wsURL string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                   wsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		ConnectionRateLimiter: newUTAPrivateConnectionLimiter,
		Connector:             e.UTAWebsocketConnect,
		Authenticate:          e.AuthenticateUTAPrivateWebsocket,
		GenerateSubscriptions: func() (subscription.List, error) { return e.generateUTASubscriptions(utaPrivateConnection) },
		Subscriber:            e.UTASubscribe,
		Unsubscriber:          e.UTAUnsubscribe,
		Handler:               e.wsHandleUTAData,
		MessageFilter:         utaPrivateConnection,
	}
}

// newUTATradeConnection returns the setup of the UTA trade connection, which carries order operations and no
// subscriptions. Order operations share the UTA trading rate limit with REST orders rather than a connection limit
func (e *Exchange) newUTATradeConnection(exch *config.Exchange, wsURL string) *websocket.ConnectionSetup {
	return &websocket.ConnectionSetup{
		URL:                      wsURL,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         exch.WebsocketResponseMaxLimit,
		Connector:                e.UTAWebsocketConnect,
		Authenticate:             e.AuthenticateUTATradeWebsocket,
		Handler:                  e.wsHandleUTATradeData,
		MessageFilter:            utaTradeConnection,
		SubscriptionsNotRequired: true,
	}
}

// newUTAPublicConnectionLimiter limits a UTA public connection to the 300 messages per 10 seconds KuCoin accepts from
// it, counting subscriptions, unsubscriptions and pings
func newUTAPublicConnectionLimiter() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(10*time.Second, 300, 1)
}

// newUTAPrivateConnectionLimiter limits a UTA private push connection to the 100 messages per 10 seconds KuCoin accepts
// from it
func newUTAPrivateConnectionLimiter() *request.RateLimiterWithWeight {
	return request.NewRateLimitWithWeight(10*time.Second, 100, 1)
}

// UTAWebsocketConnect dials a UTA websocket connection and starts its keep-alive pings
func (e *Exchange) UTAWebsocketConnect(ctx context.Context, conn websocket.Connection) error {
	dialer := gws.Dialer{HandshakeTimeout: e.Config.HTTPTimeout, Proxy: http.ProxyFromEnvironment}
	if err := conn.Dial(ctx, &dialer, nil, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Delay:       utaPingInterval,
		Message:     utaPingMessage,
		MessageType: gws.TextMessage,
	})
	return nil
}

// utaAuthRequest returns a signed authentication message for a UTA private push or trade connection
func (e *Exchange) utaAuthRequest(ctx context.Context) (*UTAWebsocketAuthRequest, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	timestamp := time.Now().UnixMilli()
	return &UTAWebsocketAuthRequest{
		Operation:  "auth",
		APIKey:     creds.Key,
		Timestamp:  timestamp,
		Signature:  hmacSign(strconv.FormatInt(timestamp, 10)+"POST/api/websocket/users/verify", creds.Secret),
		Passphrase: hmacSign(creds.ClientID, creds.Secret),
	}, nil
}

// AuthenticateUTAPrivateWebsocket signs in a UTA private push connection. The docs list no ID for the authentication
// message, but KuCoin echoes one when sent, which lets the reply be matched
func (e *Exchange) AuthenticateUTAPrivateWebsocket(ctx context.Context, conn websocket.Connection) error {
	req, err := e.utaAuthRequest(ctx)
	if err != nil {
		return err
	}
	req.ID = e.utaMessageID()
	reply, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return err
	}
	var resp UTAWebsocketAckResponse
	if err := json.Unmarshal(reply, &resp); err != nil {
		return err
	}
	if !resp.Result.Bool() {
		return fmt.Errorf("%w: %s", errUTAAuthenticationFailed, cmp.Or(resp.Message, resp.Reason))
	}
	return nil
}

// hasUTASubscriptions reports whether the configuration enables any UTA channel
func (e *Exchange) hasUTASubscriptions() bool {
	return slices.ContainsFunc(e.Features.Subscriptions, isUTASubscription)
}

// isUTASubscription reports whether a subscription names a UTA channel, which the UTA connections serve
func isUTASubscription(s *subscription.Subscription) bool {
	_, ok := utaChannels[s.Channel]
	return ok
}

// utaTopic identifies the KuCoin topic a UTA subscription receives on its connection
type utaTopic struct {
	qualifiedChannel string
	interval         kline.Interval
}

// generateUTASubscriptions expands the configured UTA channels the given connection serves. Nothing is generated unless
// the configuration lists UTA channels, so the manager leaves UTA connections down by default. Each KuCoin topic is
// subscribed to once: spot and margin share the spot market's topics, so unsubscribing one would end the other, and
// the orderbook store holds one book per pair and asset, so the first orderbook depth configured for a symbol is kept
func (e *Exchange) generateUTASubscriptions(connection string) (subscription.List, error) {
	var subs subscription.List
	topics := make(map[utaTopic]struct{})
	for _, s := range e.Features.Subscriptions {
		ch, ok := utaChannels[s.Channel]
		if !ok || (ch.private && !e.Websocket.CanUseAuthenticatedEndpoints()) {
			continue
		}
		assets, err := e.utaSubscriptionAssets(s, ch)
		if err != nil {
			if utaConnectionFor(ch, s.Asset) != connection {
				continue
			}
			return nil, err
		}
		for _, a := range assets {
			if utaConnectionFor(ch, a) != connection {
				continue
			}
			expanded, err := e.expandUTASubscription(s, ch, a)
			if err != nil {
				return nil, err
			}
			for _, sub := range expanded {
				topic := utaTopic{qualifiedChannel: sub.QualifiedChannel, interval: sub.Interval}
				if _, ok := topics[topic]; ok {
					continue
				}
				topics[topic] = struct{}{}
				subs = append(subs, sub)
			}
		}
	}
	return subs, nil
}

// utaSubscriptionAssets returns the assets a configured UTA channel stands for. An account channel stands for its
// configured asset, which does not matter to it; asset.All stands for each asset the channel serves, and an empty asset
// for the market of a channel serving one market. As with the Classic channels, disabled assets stand for nothing, as
// does an empty asset for a channel serving both markets. An asset the channel does not serve is an error
func (e *Exchange) utaSubscriptionAssets(s *subscription.Subscription, ch utaChannel) (asset.Items, error) {
	if ch.private && !ch.perSymbol {
		return asset.Items{s.Asset}, nil
	}
	served := ch.assets()
	var assets asset.Items
	switch s.Asset {
	case asset.All:
		assets = served
	case asset.Empty:
		if ch.spot != ch.futures {
			assets = served[:1]
		}
	default:
		if !slices.Contains(served, s.Asset) {
			return nil, fmt.Errorf("%w: %s for asset %q", subscription.ErrNotSupported, s.Channel, s.Asset)
		}
		assets = asset.Items{s.Asset}
	}
	return slices.DeleteFunc(assets, func(a asset.Item) bool { return e.CurrencyPairs.IsAssetEnabled(a) != nil }), nil
}

// assets returns the assets whose symbols a UTA channel serves: spot and margin on the spot market, and futures
func (ch utaChannel) assets() asset.Items {
	var assets asset.Items
	if ch.spot {
		assets = append(assets, asset.Spot, asset.Margin)
	}
	if ch.futures {
		assets = append(assets, asset.Futures)
	}
	return assets
}

// utaConnectionFor returns the message filter of the connection serving a UTA channel for an asset; a public channel
// without an asset is served on the market it serves alone, or the spot market
func utaConnectionFor(ch utaChannel, a asset.Item) string {
	switch {
	case ch.private:
		return utaPrivateConnection
	case a == asset.Futures, a == asset.Empty && ch.futures && !ch.spot:
		return utaFuturesPublicConnection
	default:
		return utaSpotPublicConnection
	}
}

// expandUTASubscription returns the subscriptions a configured UTA channel stands for on an asset: one per symbol for
// symbol channels, using the enabled pairs of the asset when the configuration names no pairs
func (e *Exchange) expandUTASubscription(s *subscription.Subscription, ch utaChannel, a asset.Item) (subscription.List, error) {
	if err := validateUTASubscription(s, a); err != nil {
		return nil, err
	}
	if !ch.perSymbol {
		sub := s.Clone()
		sub.Asset = a
		sub.Authenticated = ch.private
		sub.QualifiedChannel = ch.name
		return subscription.List{sub}, nil
	}
	pairs := s.Pairs
	if len(pairs) == 0 {
		var err error
		if pairs, err = e.GetEnabledPairs(a); err != nil {
			return nil, err
		}
	}
	pairFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	subs := make(subscription.List, 0, len(pairs))
	for _, p := range pairs {
		sub := s.Clone()
		sub.Asset = a
		sub.Pairs = currency.Pairs{p}
		sub.Authenticated = ch.private
		sub.QualifiedChannel = ch.name + ":" + pairFmt.Format(p)
		subs = append(subs, sub)
	}
	return subs, nil
}

// validateUTASubscription checks that a configured UTA channel's interval or orderbook depth is one KuCoin offers for
// the asset
func validateUTASubscription(s *subscription.Subscription, a asset.Item) error {
	var err error
	switch s.Channel {
	case UTAKlineChannel:
		_, err = utaKlineInterval(s.Interval, a)
	case UTAOrderbookChannel:
		_, err = utaOrderbookDepth(s.Levels)
	}
	return err
}

// utaKlineInterval returns KuCoin's name for a kline interval; futures klines have no 6 hour interval
func utaKlineInterval(i kline.Interval, a asset.Item) (string, error) {
	name, ok := utaKlineIntervals[i]
	if !ok || (a == asset.Futures && i == kline.SixHour) {
		return "", fmt.Errorf("%w: %s for %s", subscription.ErrInvalidInterval, i, a)
	}
	return name, nil
}

// utaOrderbookDepth returns the orderbook depth a subscription's levels select: 1, 5 and 50 push snapshots, while 500
// selects the best 500 levels as a snapshot followed by deltas every 10ms. The plain increment depth is not offered:
// KuCoin deprecated it from 15 July 2026 in favour of increment@10ms
func utaOrderbookDepth(levels int) (string, error) {
	switch levels {
	case 1, 5, 50:
		return strconv.Itoa(levels), nil
	case 500:
		return "increment@10ms", nil
	}
	return "", fmt.Errorf("%w: %d", subscription.ErrInvalidLevel, levels)
}

// UTASubscribe subscribes a UTA push connection to channels
func (e *Exchange) UTASubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageUTASubscriptions(ctx, conn, subs, utaSubscribeAction)
}

// UTAUnsubscribe unsubscribes a UTA push connection from channels
func (e *Exchange) UTAUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.manageUTASubscriptions(ctx, conn, subs, utaUnsubscribeAction)
}

// manageUTASubscriptions sends one subscribe or unsubscribe message per subscription, as most UTA channels take a
// single symbol, and waits for the replies together
func (e *Exchange) manageUTASubscriptions(ctx context.Context, conn websocket.Connection, subs subscription.List, action string) error {
	errs := make([]error, len(subs))
	var wg sync.WaitGroup
	for i, s := range subs {
		wg.Go(func() { errs[i] = e.manageUTASubscription(ctx, conn, s, action) })
	}
	wg.Wait()
	var err error
	for _, subErr := range errs {
		err = common.AppendError(err, subErr)
	}
	return err
}

// manageUTASubscription subscribes to or unsubscribes from one subscription and records the outcome
func (e *Exchange) manageUTASubscription(ctx context.Context, conn websocket.Connection, s *subscription.Subscription, action string) error {
	req, err := e.utaSubscriptionRequest(s, action)
	if err != nil {
		return err
	}
	reply, err := conn.SendMessageReturnResponse(ctx, request.Unset, req.ID, req)
	if err != nil {
		return err
	}
	var resp UTAWebsocketAckResponse
	if err := json.Unmarshal(reply, &resp); err != nil {
		return err
	}
	if !resp.Result.Bool() {
		return fmt.Errorf("%w: %s %s: %s", errUTARequestRejected, action, s.QualifiedChannel, cmp.Or(resp.Message, resp.Reason))
	}
	if action == utaUnsubscribeAction {
		return e.Websocket.RemoveSubscriptions(conn, s)
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, s)
}

// utaSubscriptionRequest returns the message subscribing to or unsubscribing from a UTA subscription
func (e *Exchange) utaSubscriptionRequest(s *subscription.Subscription, action string) (*UTAWebsocketSubscriptionRequest, error) {
	ch, ok := utaChannels[s.Channel]
	if !ok {
		return nil, fmt.Errorf("%w: %q", subscription.ErrNotSupported, s.Channel)
	}
	req := &UTAWebsocketSubscriptionRequest{ID: e.utaMessageID(), Action: action, Channel: ch.name}
	switch {
	case s.Channel == UTABalanceChannel:
		// The unified account holds the funds of spot, margin and futures trading; the funding and isolated margin
		// accounts the docs also offer are not subscribed to
		req.AccountType = utaUnifiedAccountType
	case ch.private:
		req.TradeType = utaUnifiedAccountType
	case ch.tradeType && s.Asset == asset.Futures:
		req.TradeType = "FUTURES"
	case ch.tradeType:
		req.TradeType = "SPOT"
	}
	if ch.perSymbol {
		if len(s.Pairs) != 1 {
			return nil, fmt.Errorf("%w: %s", subscription.ErrNotSinglePair, s)
		}
		symbol, err := e.FormatSymbol(s.Pairs[0], s.Asset)
		if err != nil {
			return nil, err
		}
		req.Symbol = symbol
	}
	var err error
	switch s.Channel {
	case UTAKlineChannel:
		req.Interval, err = utaKlineInterval(s.Interval, s.Asset)
	case UTAOrderbookChannel:
		req.Depth, err = utaOrderbookDepth(s.Levels)
	}
	if err != nil {
		return nil, err
	}
	return req, nil
}

// utaMessageID returns a request ID that fits every UTA connection: a UUID without its dashes, as trade requests take at
// most 32 bytes. Public subscriptions would take the dashes, in IDs of up to 40 characters
func (e *Exchange) utaMessageID() string {
	return strings.ReplaceAll(e.MessageID(), "-", "")
}

// wsHandleUTAData routes a frame from a UTA public or private push connection
func (e *Exchange) wsHandleUTAData(ctx context.Context, conn websocket.Connection, respData []byte) error {
	var msg UTAWebsocketMessage
	if err := json.Unmarshal(respData, &msg); err != nil {
		return err
	}
	switch {
	case msg.Topic != "":
		return e.processUTAPush(ctx, conn, &msg)
	case msg.Operation == "pong", msg.Message == "welcome":
		return nil
	case msg.ID != "":
		return conn.RequireMatchWithData(msg.ID, respData)
	}
	return fmt.Errorf("%w: %s", errUTAUnexpectedMessage, respData)
}

// processUTAPush routes a push to its channel's handler. A topic is the channel name followed by the market or account
// type the push belongs to, except on futures channels serving one market
func (e *Exchange) processUTAPush(ctx context.Context, conn websocket.Connection, msg *UTAWebsocketMessage) error {
	channel, suffix, _ := strings.CutLast(msg.Topic, ".")
	switch channel {
	case "ticker":
		return e.processUTATicker(ctx, msg, suffix)
	case "kline":
		return e.processUTAKline(ctx, msg, suffix)
	case "trade":
		return e.processUTATrade(msg, suffix)
	case "obu":
		return e.processUTAOrderbook(ctx, conn, msg, suffix)
	case "mark-price":
		return e.processUTAMarkPrice(ctx, msg)
	case "funding-fee":
		return e.processUTAFundingFee(ctx, msg)
	case "funding-fee-all-symbols":
		return e.processUTAAllFundingFeeRates(ctx, msg)
	case "callAuctionInfo":
		return e.processUTACallAuctionInfo(ctx, msg)
	case "orderAll", "order":
		return e.processUTAOrder(ctx, msg)
	case "execution.lite":
		return e.processUTAExecutionLite(msg)
	case "execution":
		return e.processUTAExecution(ctx, msg)
	case "balance":
		return e.processUTABalance(ctx, msg, suffix)
	case "positionAll", "position":
		return e.processUTAPosition(ctx, msg)
	case "lw", "risk": // The docs' example pushes lw.UNIFIED and the docs' schema names risk.UNIFIED
		return e.processUTALiquidationWarning(ctx, msg)
	case "leverage":
		return e.processUTALeverage(ctx, msg)
	}
	return fmt.Errorf("%w: %q", errUTAUnknownTopic, msg.Topic)
}

// utaPairAndAssets returns the pair a public push symbol names and the assets its data is stored under. Futures
// symbols carry no delimiter, so they are matched against the available contracts; a spot symbol is stored under
// spot and margin wherever the pair is enabled, as both trade on the spot market, and under spot otherwise
func (e *Exchange) utaPairAndAssets(symbol, market string) (currency.Pair, []asset.Item, error) {
	switch market {
	case "SPOT":
		pair, err := currency.NewPairFromString(symbol)
		if err != nil {
			return currency.EMPTYPAIR, nil, err
		}
		assets := make([]asset.Item, 0, 2)
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			if enabled, err := e.IsPairEnabled(pair, a); err == nil && enabled {
				assets = append(assets, a)
			}
		}
		if len(assets) == 0 {
			assets = append(assets, asset.Spot)
		}
		return pair, assets, nil
	case "FUTURES":
		pair, err := e.MatchSymbolWithAvailablePairs(symbol, asset.Futures, false)
		if err != nil {
			return currency.EMPTYPAIR, nil, err
		}
		return pair, []asset.Item{asset.Futures}, nil
	}
	return currency.EMPTYPAIR, nil, fmt.Errorf("%w: market %q", errUTAUnknownTopic, market)
}
