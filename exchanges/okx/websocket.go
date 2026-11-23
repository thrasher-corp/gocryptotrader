package okx

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errWebsocketRequestRejected  = errors.New("websocket request rejected")
	errChannelConnectionLimit    = errors.New("channel connection limit reached")
	errInvalidWebsocketRequest   = errors.New("invalid websocket request")
	errOperationFailed           = errors.New("operation failed")
	errPartialSuccess            = errors.New("bulk operation partially succeeded")
	errMultipleItemsReturned     = errors.New("multiple items returned")
	errUnsupportedCandleInterval = errors.New("unsupported candlestick interval")
	errSubscriptionParameter     = errors.New("invalid subscription parameter")
)

var (
	errSubscriptionNotAcknowledged = errors.New("subscription not acknowledged")
	errSubscriptionArgumentTooLong = errors.New("subscription argument too long for a request")
)

// The message filters of the connections OKX serves channels and operations
// on. The channels which need a login are served by their own connections to
// the public and business URLs, so the others subscribe without one.
const (
	publicConnection        = "public"
	publicLoginConnection   = "public-login"
	privateConnection       = "private"
	businessConnection      = "business"
	businessLoginConnection = "business-login"
)

const okxBusinessWebsocketURL = websocketURL + "business"

var (
	pingMsg = []byte("ping")
	pongMsg = []byte("pong")
)

// authConnErrorCodes are the error codes OKX answers a login with
// See: https://www.okx.com/docs-v5/en/#error-code-websocket-public
var authConnErrorCodes = []string{
	"60007", "60022", "60023", "60024", "60026", "63999", "60032", "60011", "60009",
	"60005", "60021", "60031", "50110", "60033",
}

// noticeServiceUpgrade is the code of the notice OKX sends a minute before it
// closes a connection for a service upgrade
// See: https://www.okx.com/docs-v5/en/#overview-websocket-notification
const noticeServiceUpgrade = "64008"

const (
	wsOrderbookSnapshot = "snapshot"
	wsOrderbookUpdate   = "update"
)

// maxConnByteLen is the documented maximum length of a subscription request's
// arguments
const maxConnByteLen = 64 * 1024

// Channel names
const (
	channelAccount                    = "account"
	channelPositions                  = "positions"
	channelBalanceAndPosition         = "balance_and_position"
	channelLiquidationWarning         = "liquidation-warning"
	channelAccountGreeks              = "account-greeks"
	channelOrders                     = "orders"
	channelFills                      = "fills"
	channelAlgoOrders                 = "orders-algo"
	channelAlgoAdvance                = "algo-advance"
	channelSpotGridOrders             = "grid-orders-spot"
	channelContractGridOrders         = "grid-orders-contract"
	channelGridPositions              = "grid-positions"
	channelGridSubOrders              = "grid-sub-orders"
	channelRecurringBuy               = "algo-recurring-buy"
	channelLeadTradingNotification    = "copytrading-lead-notification"
	channelTickers                    = "tickers"
	channelCandle                     = "candle"
	channelTrades                     = "trades"
	channelAllTrades                  = "trades-all"
	channelOrderBooks                 = "books"
	channelOrderBooksRPI              = "books-rpi"
	channelOrderBooks5                = "books5"
	channelBBOTBT                     = "bbo-tbt"
	channelBBORPITBT                  = "bbo-rpi-tbt"
	channelOrderBooks50TBT            = "books50-l2-tbt"
	channelOrderBooksTBT              = "books-l2-tbt"
	channelOptionTrades               = "option-trades"
	channelCallAuctionDetails         = "call-auction-details"
	channelRequestForQuotes           = "rfqs"
	channelQuotes                     = "quotes"
	channelStructureBlockTrades       = "struc-block-trades"
	channelPublicStructureBlockTrades = "public-struc-block-trades"
	channelPublicBlockTrades          = "public-block-trades"
	channelBlockTickers               = "block-tickers"
	channelSpreadOrders               = "sprd-orders"
	channelSpreadTrades               = "sprd-trades"
	channelSpreadOrderBooks5          = "sprd-books5"
	channelSpreadBBOTBT               = "sprd-bbo-tbt"
	channelSpreadOrderBooksTBT        = "sprd-books-l2-tbt"
	channelSpreadPublicTrades         = "sprd-public-trades"
	channelSpreadTickers              = "sprd-tickers"
	channelSpreadCandle               = "sprd-candle"
	channelInstruments                = "instruments"
	channelEventContractMarkets       = "event-contract-markets"
	channelOpenInterest               = "open-interest"
	channelFundingRate                = "funding-rate"
	channelPriceLimit                 = "price-limit"
	channelOptionSummary              = "opt-summary"
	channelEstimatedPrice             = "estimated-price"
	channelMarkPrice                  = "mark-price"
	channelIndexTickers               = "index-tickers"
	channelMarkPriceCandle            = "mark-price-candle"
	channelIndexCandle                = "index-candle"
	channelLiquidationOrders          = "liquidation-orders"
	channelADLWarning                 = "adl-warning"
	channelEconomicCalendar           = "economic-calendar"
	channelDepositInfo                = "deposit-info"
	channelWithdrawalInfo             = "withdrawal-info"
	channelStatus                     = "status"
)

// channelScope describes how a channel's subscription arguments are formed
type channelScope uint8

const (
	channelScopeNone              channelScope = iota // The channel and any configured parameters
	channelScopeAsset                                 // One per asset, with the asset's instType; spreads have none
	channelScopeInstrument                            // One per pair, as instId
	channelScopeInstrumentAndType                     // One per pair, as instId beside the asset's instType
	channelScopeIndex                                 // One per pair, as the instId of the pair's index
	channelScopeSpread                                // One per spread pair, as sprdId
)

// websocketChannel describes the connection serving a channel and how its
// subscription arguments are formed
type websocketChannel struct {
	connection string
	scope      channelScope
	assets     []asset.Item // The assets a scoped channel accepts
}

var (
	contractAssets   = []asset.Item{asset.PerpetualSwap, asset.Futures, asset.Options}
	marginAssets     = []asset.Item{asset.Margin, asset.PerpetualSwap, asset.Futures, asset.Options}
	instrumentAssets = []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures, asset.Options}
	algoAssets       = []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures}
	spreadAssets     = []asset.Item{asset.Spread}
)

var websocketChannels = map[string]websocketChannel{
	channelAccount:                    {connection: privateConnection},
	channelPositions:                  {privateConnection, channelScopeAsset, marginAssets},
	channelBalanceAndPosition:         {connection: privateConnection},
	channelLiquidationWarning:         {privateConnection, channelScopeAsset, marginAssets},
	channelAccountGreeks:              {connection: privateConnection},
	channelOrders:                     {privateConnection, channelScopeAsset, instrumentAssets},
	channelFills:                      {privateConnection, channelScopeInstrument, instrumentAssets},
	channelAlgoOrders:                 {businessLoginConnection, channelScopeAsset, algoAssets},
	channelAlgoAdvance:                {businessLoginConnection, channelScopeAsset, algoAssets},
	channelSpotGridOrders:             {businessLoginConnection, channelScopeAsset, []asset.Item{asset.Spot}},
	channelContractGridOrders:         {businessLoginConnection, channelScopeAsset, []asset.Item{asset.PerpetualSwap, asset.Futures}},
	channelGridPositions:              {connection: businessLoginConnection},
	channelGridSubOrders:              {connection: businessLoginConnection},
	channelRecurringBuy:               {businessLoginConnection, channelScopeAsset, []asset.Item{asset.Spot}},
	channelLeadTradingNotification:    {businessLoginConnection, channelScopeAsset, []asset.Item{asset.PerpetualSwap}},
	channelTickers:                    {publicConnection, channelScopeInstrument, instrumentAssets},
	channelTrades:                     {publicConnection, channelScopeInstrument, instrumentAssets},
	channelAllTrades:                  {businessConnection, channelScopeInstrument, instrumentAssets},
	channelOrderBooks:                 {publicConnection, channelScopeInstrument, instrumentAssets},
	channelOrderBooksRPI:              {publicConnection, channelScopeInstrument, instrumentAssets},
	channelOrderBooks5:                {publicConnection, channelScopeInstrument, instrumentAssets},
	channelBBOTBT:                     {publicConnection, channelScopeInstrument, instrumentAssets},
	channelBBORPITBT:                  {publicConnection, channelScopeInstrument, instrumentAssets},
	channelOrderBooks50TBT:            {publicLoginConnection, channelScopeInstrument, instrumentAssets},
	channelOrderBooksTBT:              {publicLoginConnection, channelScopeInstrument, instrumentAssets},
	channelOptionTrades:               {publicConnection, channelScopeInstrumentAndType, []asset.Item{asset.Options}},
	channelCallAuctionDetails:         {publicConnection, channelScopeInstrument, instrumentAssets},
	channelRequestForQuotes:           {connection: businessLoginConnection},
	channelQuotes:                     {connection: businessLoginConnection},
	channelStructureBlockTrades:       {connection: businessLoginConnection},
	channelPublicStructureBlockTrades: {connection: businessConnection},
	channelPublicBlockTrades:          {businessConnection, channelScopeInstrument, instrumentAssets},
	channelBlockTickers:               {businessConnection, channelScopeInstrument, instrumentAssets},
	channelSpreadOrders:               {businessLoginConnection, channelScopeAsset, spreadAssets},
	channelSpreadTrades:               {businessLoginConnection, channelScopeAsset, spreadAssets},
	channelSpreadOrderBooks5:          {businessConnection, channelScopeSpread, spreadAssets},
	channelSpreadBBOTBT:               {businessConnection, channelScopeSpread, spreadAssets},
	channelSpreadOrderBooksTBT:        {businessConnection, channelScopeSpread, spreadAssets},
	channelSpreadPublicTrades:         {businessConnection, channelScopeSpread, spreadAssets},
	channelSpreadTickers:              {businessConnection, channelScopeSpread, spreadAssets},
	channelInstruments:                {publicConnection, channelScopeAsset, instrumentAssets},
	channelEventContractMarkets:       {connection: publicConnection},
	channelOpenInterest:               {publicConnection, channelScopeInstrument, contractAssets},
	channelFundingRate:                {publicConnection, channelScopeInstrument, []asset.Item{asset.PerpetualSwap, asset.Futures}},
	channelPriceLimit:                 {publicConnection, channelScopeInstrument, instrumentAssets},
	channelOptionSummary:              {connection: publicConnection},
	channelEstimatedPrice:             {publicConnection, channelScopeInstrumentAndType, contractAssets},
	channelMarkPrice:                  {publicConnection, channelScopeInstrument, instrumentAssets},
	channelIndexTickers:               {publicConnection, channelScopeIndex, instrumentAssets},
	channelLiquidationOrders:          {publicConnection, channelScopeAsset, marginAssets},
	channelADLWarning:                 {publicConnection, channelScopeAsset, contractAssets},
	channelEconomicCalendar:           {connection: businessLoginConnection},
	channelDepositInfo:                {connection: businessLoginConnection},
	channelWithdrawalInfo:             {connection: businessLoginConnection},
	channelStatus:                     {connection: publicConnection},
}

// candleIntervals maps the documented candlestick channel suffixes to their
// intervals; the suffixes from 6H up without utc follow Hong Kong time
var candleIntervals = map[string]kline.Interval{
	"1s": kline.ThousandMilliseconds, "1m": kline.OneMin, "3m": kline.ThreeMin, "5m": kline.FiveMin, "15m": kline.FifteenMin,
	"30m": kline.ThirtyMin, "1H": kline.OneHour, "2H": kline.TwoHour, "4H": kline.FourHour, "6H": kline.SixHour,
	"12H": kline.TwelveHour, "1D": kline.OneDay, "2D": kline.TwoDay, "3D": kline.ThreeDay, "5D": kline.FiveDay,
	"1W": kline.OneWeek, "1M": kline.OneMonth, "3M": kline.ThreeMonth, "6Hutc": kline.SixHour, "12Hutc": kline.TwelveHour,
	"1Dutc": kline.OneDay, "2Dutc": kline.TwoDay, "3Dutc": kline.ThreeDay, "5Dutc": kline.FiveDay, "1Wutc": kline.OneWeek,
	"1Mutc": kline.OneMonth, "3Mutc": kline.ThreeMonth,
}

func init() {
	for suffix := range candleIntervals {
		websocketChannels[channelCandle+suffix] = websocketChannel{businessConnection, channelScopeInstrument, instrumentAssets}
		websocketChannels[channelIndexCandle+suffix] = websocketChannel{businessConnection, channelScopeIndex, instrumentAssets}
		if suffix == "1s" {
			continue // OKX rejects mark price and spread candlesticks of one second
		}
		websocketChannels[channelMarkPriceCandle+suffix] = websocketChannel{businessConnection, channelScopeInstrument, instrumentAssets}
		websocketChannels[channelSpreadCandle+suffix] = websocketChannel{businessConnection, channelScopeSpread, spreadAssets}
	}
}

// candleChannelSuffix returns the channel suffix of a candlestick interval,
// preferring the UTC aligned channels
func candleChannelSuffix(interval kline.Interval) (string, error) {
	var suffix string
	for s, i := range candleIntervals {
		if i != interval {
			continue
		}
		if suffix == "" || strings.HasSuffix(s, "utc") {
			suffix = s
		}
	}
	if suffix == "" {
		return "", fmt.Errorf("%w: %s", errUnsupportedCandleInterval, interval)
	}
	return suffix, nil
}

// candleChannelInterval returns the interval of a candlestick channel
func candleChannelInterval(channel, prefix string) (kline.Interval, error) {
	if suffix, ok := strings.CutPrefix(channel, prefix); ok {
		if interval, ok := candleIntervals[suffix]; ok {
			return interval, nil
		}
	}
	return 0, fmt.Errorf("%w: %s", kline.ErrInvalidInterval, channel)
}

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
}

// WebsocketAPI holds the order operations OKX serves over its websocket
// connections
type WebsocketAPI struct {
	e *Exchange
}

// WebsocketAPI returns the exchange's websocket order operations
func (e *Exchange) WebsocketAPI() WebsocketAPI {
	return WebsocketAPI{e: e}
}

// setupWebsocket sets up the websocket manager and the connections OKX serves
// channels and operations on
func (e *Exchange) setupWebsocket(exch *config.Exchange) error {
	// The websocket draws on the requester's limits, as OKX shares an
	// operation's budget between REST and the websocket
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		MaxWebsocketSubscriptionsPerConnection: 30, // see: https://www.okx.com/docs-v5/en/#overview-websocket-connection-count-limit
		RateLimitDefinitions:                   e.Requester.GetRateLimiterDefinitions(),
		UseMultiConnectionManagement:           true,
		FillsFeed:                              e.Features.Enabled.FillsFeed,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
	}); err != nil {
		return err
	}
	for _, c := range []struct {
		endpoint exchange.URL
		filter   string
		login    bool
	}{
		{exchange.WebsocketSpot, publicConnection, false},
		{exchange.WebsocketSpot, publicLoginConnection, true},
		{exchange.WebsocketPrivate, privateConnection, true},
		{exchange.WebsocketSpotSupplementary, businessConnection, false},
		{exchange.WebsocketSpotSupplementary, businessLoginConnection, true},
	} {
		u, err := e.API.Endpoints.GetURL(c.endpoint)
		if err != nil {
			return err
		}
		setup := &websocket.ConnectionSetup{
			URL:                   u,
			ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:      websocketResponseMaxLimit,
			ConnectionRateLimiter: websocketControlLimiter,
			Connector:             e.wsConnect,
			GenerateSubscriptions: func() (subscription.List, error) { return e.generateSubscriptions(c.filter) },
			Subscriber:            e.Subscribe,
			Unsubscriber:          e.Unsubscribe,
			Handler:               e.wsHandleData,
			MessageFilter:         c.filter,
		}
		if c.login {
			setup.Authenticate = e.wsAuthenticateConnection
		} else {
			setup.TrackOnExistingConnection = e.trackEquivalentSubscriptionsOnExistingConnection
		}
		if err := e.Websocket.SetupNewConnection(setup); err != nil {
			return err
		}
	}
	return nil
}

// wsConnect dials a websocket connection and keeps it alive
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := e.Requester.InitiateRateLimit(ctx, websocketConnectionEPL); err != nil {
		return err
	}
	if err := conn.Dial(ctx, &gws.Dialer{ReadBufferSize: 8192, WriteBufferSize: 8192}, nil, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     pingMsg,
		Delay:       time.Second * 20,
	})
	return nil
}

// wsAuthenticateConnection logs a connection in
func (e *Exchange) wsAuthenticateConnection(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ts := time.Now().Unix()
	hmac, err := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(strconv.FormatInt(ts, 10)+http.MethodGet+"/users/self/verify"),
		[]byte(creds.Secret),
	)
	if err != nil {
		return err
	}
	req := &WebsocketLoginRequest{
		Operation: operationLogin,
		Arguments: []WebsocketLoginArgument{{
			APIKey:     creds.Key,
			Passphrase: creds.ClientID,
			Timestamp:  uint64(ts), //nolint:gosec // The current Unix time is positive
			Sign:       base64.StdEncoding.EncodeToString(hmac),
		}},
	}
	resp, err := conn.SendMessageReturnResponse(ctx, websocketRequestEPL, "login-response", req)
	if err != nil {
		return err
	}
	var event *WebsocketEventResponse
	if err := json.Unmarshal(resp, &event); err != nil {
		return err
	}
	if event == nil {
		return fmt.Errorf("%w: login reply", common.ErrNoResponse)
	}
	if event.Code == "" || event.Code == "0" {
		e.recordWebsocketLogin(conn, creds)
		return nil
	}
	code, err := strconv.ParseUint(event.Code, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: login code %q: %s", errWebsocketRequestRejected, event.Code, event.Message)
	}
	return getStatusError(code, event.Message)
}

// websocketLogin holds the credentials a websocket connection logged in with
type websocketLogin struct {
	credentials accounts.Credentials
}

func newWebsocketLogin(creds *accounts.Credentials) websocketLogin {
	return websocketLogin{credentials: *creds}
}

// namesAccount returns whether creds name the account of the login. OKX signs
// a login with the key, secret and passphrase alone, so a sub-account or a
// one-time password beside them names the same account.
func (l *websocketLogin) namesAccount(creds *accounts.Credentials) bool {
	return creds.Key == l.credentials.Key && creds.Secret == l.credentials.Secret && creds.ClientID == l.credentials.ClientID
}

// recordWebsocketLogin records the credentials conn logged in with, and drops
// the records of connections which have since closed
func (e *Exchange) recordWebsocketLogin(conn websocket.Connection, creds *accounts.Credentials) {
	e.websocketLoginsMu.Lock()
	defer e.websocketLoginsMu.Unlock()
	if e.websocketLogins == nil {
		e.websocketLogins = make(map[websocket.Connection]websocketLogin)
	}
	for c := range e.websocketLogins {
		if closed, ok := c.(interface{ IsConnected() bool }); ok && !closed.IsConnected() {
			delete(e.websocketLogins, c)
		}
	}
	e.websocketLogins[conn] = newWebsocketLogin(creds)
}

// checkWebsocketLogin returns errWebsocketOtherAccount unless conn logged in
// with the credentials the request acts for, its context's or the exchange's
// own: an operation on the connection acts for the account it logged in as.
func (e *Exchange) checkWebsocketLogin(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.websocketLoginsMu.Lock()
	login, loggedIn := e.websocketLogins[conn]
	e.websocketLoginsMu.Unlock()
	switch {
	case !loggedIn:
		return fmt.Errorf("%w: the connection has not logged in", errWebsocketOtherAccount)
	case !login.namesAccount(creds):
		return errWebsocketOtherAccount
	}
	return nil
}

// websocketLoginCredentials returns a copy of the credentials conn logged in
// with, as the account its private pushes belong to
func (e *Exchange) websocketLoginCredentials(conn websocket.Connection) (*accounts.Credentials, error) {
	e.websocketLoginsMu.Lock()
	login, loggedIn := e.websocketLogins[conn]
	e.websocketLoginsMu.Unlock()
	if !loggedIn {
		return nil, errUnknownPushAccount
	}
	return new(login.credentials), nil
}

// sendWebsocketRequest sends an order operation on the connection the message
// filter selects and decodes the reply's data into result, a pointer to a
// slice of pointers to the operation's result rows. A partial success or a
// failure still decodes the rows OKX answered with.
func (e *Exchange) sendWebsocketRequest(ctx context.Context, connection string, epl request.EndpointLimit, operation string, payload, result any) error {
	if operation == "" || payload == nil {
		return errInvalidWebsocketRequest
	}
	conn, err := e.Websocket.GetConnection(connection)
	if err != nil {
		return err
	}
	if err := e.checkWebsocketLogin(ctx, conn); err != nil {
		return err
	}
	id := e.MessageID()
	incoming, err := conn.SendMessageReturnResponse(ctx, epl, id, &WebsocketOperationRequest{ID: id, Operation: operation, Arguments: payload})
	if err != nil {
		return err
	}
	resp := &WebsocketOperationResponse{Data: result}
	if err := json.Unmarshal(incoming, resp); err != nil {
		return err
	}
	switch resp.Code {
	case 0:
		return nil
	case 1:
		return parseWebsocketResponseErrors(result, errOperationFailed)
	case 2:
		return parseWebsocketResponseErrors(result, errPartialSuccess)
	default:
		return getStatusError(resp.Code, resp.Message)
	}
}

// parseWebsocketResponseErrors collects the errors of a batch reply's rows
func parseWebsocketResponseErrors(result any, err error) error {
	s := reflect.ValueOf(result).Elem()
	for i := range s.Len() {
		v := s.Index(i)
		if v.Kind() == reflect.Pointer && v.IsNil() {
			continue
		}
		if subErr, ok := reflect.TypeAssert[interface{ Error() error }](v); ok && subErr.Error() != nil {
			err = common.AppendError(err, fmt.Errorf("%s[%d]: %w", v.Type(), i+1, subErr.Error()))
		}
	}
	return err
}

// singleItem returns the one row a single order operation answers with
func singleItem[T any](resp []*T) (*T, error) {
	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}
	if len(resp) > 1 {
		return nil, fmt.Errorf("%w, received %d", errMultipleItemsReturned, len(resp))
	}
	if resp[0] == nil {
		// A success reply reports one row for the one requested operation; a
		// null row leaves its outcome unknown.
		return nil, fmt.Errorf("%w, success reply has a null row", common.ErrInvalidResponse)
	}
	return resp[0], nil
}

// Subscribe sends a websocket subscription request to several channels to receive data.
func (e *Exchange) Subscribe(ctx context.Context, conn websocket.Connection, channelsToSubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, operationSubscribe, channelsToSubscribe)
}

// Unsubscribe sends a websocket unsubscription request to several channels to receive data.
func (e *Exchange) Unsubscribe(ctx context.Context, conn websocket.Connection, channelsToUnsubscribe subscription.List) error {
	return e.handleSubscription(ctx, conn, operationUnsubscribe, channelsToUnsubscribe)
}

// subscriptionRequests holds the subscribe and unsubscribe requests sent on
// the websocket connections until their outcome is tracked, routing OKX's
// replies to each by its ID while it awaits them. It also counts the
// arguments of the subscriptions being subscribed, which keep an order book as
// tracked subscriptions do.
type subscriptionRequests struct {
	mu                 sync.Mutex
	sent               map[string]*sentSubscriptionRequest
	requestedArguments map[string]int
	requested          atomic.Int64
}

// sentSubscriptionRequest is a request sent on a connection, with the errors
// OKX ended channels on the connection with since it was sent
type sentSubscriptionRequest struct {
	conn     websocket.Connection
	request  *WebsocketSubscriptionRequest
	awaiting bool
	ended    map[string]error
}

// send records a request sent on conn as awaiting OKX's replies
func (r *subscriptionRequests) send(conn websocket.Connection, req *WebsocketSubscriptionRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sent == nil {
		r.sent = make(map[string]*sentSubscriptionRequest)
	}
	r.sent[req.ID] = &sentSubscriptionRequest{conn: conn, request: req, awaiting: true}
}

// route passes a reply to the request awaiting it, returning false when no
// request awaits a reply with its ID or it already holds one for each argument
func (r *subscriptionRequests) route(id string, reply []byte) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	sent, ok := r.sent[id]
	return ok && sent.awaiting && sent.request.deliver(reply)
}

// stopAwaiting stops routing replies to a request
func (r *subscriptionRequests) stopAwaiting(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sent, ok := r.sent[id]; ok {
		sent.awaiting = false
	}
}

// settle forgets a request once its outcome is tracked, returning the errors
// OKX ended channels on its connection with since it was sent, by channel
func (r *subscriptionRequests) settle(id string) map[string]error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sent, ok := r.sent[id]
	if !ok {
		return nil
	}
	delete(r.sent, id)
	return sent.ended
}

// end records the error OKX ended a channel on conn with for the subscribe
// requests sent on it
func (r *subscriptionRequests) end(conn websocket.Connection, channel string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, sent := range r.sent {
		if sent.conn != conn || sent.request.Operation != operationSubscribe {
			continue
		}
		if sent.ended == nil {
			sent.ended = make(map[string]error)
		}
		sent.ended[channel] = err
	}
}

// subscribing returns the subscriptions of the subscribe requests sent on conn
// which match
func (r *subscriptionRequests) subscribing(conn websocket.Connection, match func(*subscription.Subscription) bool) subscription.List {
	r.mu.Lock()
	defer r.mu.Unlock()
	var subs subscription.List
	for _, sent := range r.sent {
		if sent.conn != conn || sent.request.Operation != operationSubscribe {
			continue
		}
		for _, s := range sent.request.subs {
			if match(s) {
				subs = append(subs, s)
			}
		}
	}
	return subs
}

// request counts the arguments of subscriptions being subscribed until they
// are released
func (r *subscriptionRequests) request(arguments []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.requestedArguments == nil {
		r.requestedArguments = make(map[string]int)
	}
	for _, a := range arguments {
		r.requestedArguments[a]++
	}
	r.requested.Add(int64(len(arguments)))
}

// release stops counting the arguments of subscriptions as being subscribed
func (r *subscriptionRequests) release(arguments []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range arguments {
		r.requestedArguments[a]--
		if r.requestedArguments[a] <= 0 {
			delete(r.requestedArguments, a)
		}
	}
	r.requested.Add(-int64(len(arguments)))
}

// isRequested returns whether a subscription to an argument is being
// subscribed
func (r *subscriptionRequests) isRequested(argument string) bool {
	if r.requested.Load() == 0 {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requestedArguments[argument] > 0
}

// deliver queues a reply to the request, returning false when it already holds
// one for each argument
func (r *WebsocketSubscriptionRequest) deliver(reply []byte) bool {
	select {
	case r.replies <- reply:
		return true
	default:
		return false
	}
}

// handleSubscription sends subscribe or unsubscribe requests for subs, and
// tracks or stops tracking those OKX acknowledges. Subscriptions count as
// requested until the outcome of every request is tracked, so the deepest
// order book channel requested for an instrument keeps its book from the
// first push.
func (e *Exchange) handleSubscription(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	requests, err := e.chunkRequests(conn, subs, operation)
	if err != nil {
		return err
	}
	if operation == operationSubscribe {
		arguments := subs.QualifiedChannels()
		e.subscriptionRequests.request(arguments)
		defer e.subscriptionRequests.release(arguments)
	}
	outcomes := make(map[SubscriptionArgument]bool)
	var errs error
	for i := range requests {
		if err := ctx.Err(); err != nil {
			return common.AppendError(errs, err)
		}
		done, err := e.sendSubscriptionRequest(ctx, conn, &requests[i], outcomes)
		errs = common.AppendError(errs, err)
		if operation == operationUnsubscribe {
			err = e.Websocket.RemoveSubscriptions(conn, done...)
		} else {
			err = e.Websocket.AddSuccessfulSubscriptions(conn, done...)
		}
		ended, endErr := endedSubscriptions(done, e.subscriptionRequests.settle(requests[i].ID))
		if operation == operationSubscribe && len(ended) != 0 {
			errs = common.AppendError(errs, endErr)
			err = common.AppendError(err, e.Websocket.RemoveSubscriptions(conn, ended...))
		}
		if err != nil {
			return common.AppendError(errs, err)
		}
	}
	return errs
}

// endedSubscriptions returns the subscriptions to the channels OKX ended, with
// an error naming them
func endedSubscriptions(subs subscription.List, ended map[string]error) (subscription.List, error) {
	if len(ended) == 0 {
		return nil, nil
	}
	var endedSubs subscription.List
	var errs error
	for _, s := range subs {
		if err, ok := ended[subscriptionChannel(s)]; ok {
			endedSubs = append(endedSubs, s)
			errs = common.AppendError(errs, fmt.Errorf("%w; ended %s", err, s.QualifiedChannel))
		}
	}
	return endedSubs, errs
}

// sendSubscriptionRequest sends a subscribe or unsubscribe request and returns
// its subscriptions which OKX acknowledged, with an error naming the others.
// OKX answers each argument with an event echoing it or with an error, both
// carrying the request's ID; no two arguments of a request are echoed alike.
// The replies received within the connection's response limit decide the
// outcome of their arguments, and the arguments without one fail. outcomes
// records the outcome of each argument sent; a subscription whose argument
// was not sent shares the outcome of the request which sent it, or else the
// connection's subscription to it. The request stays recorded as sent until
// it is settled.
func (e *Exchange) sendSubscriptionRequest(ctx context.Context, conn websocket.Connection, req *WebsocketSubscriptionRequest, outcomes map[SubscriptionArgument]bool) (subscription.List, error) {
	var errs error
	if len(req.Arguments) != 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for i := range req.Arguments {
			outcomes[req.Arguments[i]] = false
		}
		var replies [][]byte
		replies, errs = e.awaitSubscriptionReplies(ctx, conn, req)
		for _, reply := range replies {
			var event WebsocketEventResponse
			if err := json.Unmarshal(reply, &event); err != nil {
				errs = common.AppendError(errs, err)
				continue
			}
			switch event.Event {
			case req.Operation:
				identity := subscriptionArgumentIdentity(&event.Argument)
				for i := range req.Arguments {
					if subscriptionArgumentIdentity(&req.Arguments[i]) == identity {
						outcomes[req.Arguments[i]] = true
					}
				}
			case "error":
				errs = common.AppendError(errs, fmt.Errorf("%w: code %q: %s", errWebsocketRequestRejected, event.Code, event.Message))
			}
		}
	}
	done := make(subscription.List, 0, len(req.subs))
	var failed []string
	for _, s := range req.subs {
		var arg SubscriptionArgument
		if err := json.Unmarshal([]byte(s.QualifiedChannel), &arg); err != nil {
			return nil, common.AppendError(errs, err)
		}
		if ok, sent := outcomes[arg]; sent && !ok {
			failed = append(failed, s.QualifiedChannel)
			continue
		}
		done = append(done, s)
	}
	if len(failed) != 0 {
		errs = common.AppendError(fmt.Errorf("%w: %s %s", errSubscriptionNotAcknowledged, req.Operation, strings.Join(failed, ", ")), errs)
	}
	return done, errs
}

// awaitSubscriptionReplies sends a subscribe or unsubscribe request and returns
// the replies OKX sent it until each argument has one, the connection's
// response limit passes or the context ends, with an error for the last two
func (e *Exchange) awaitSubscriptionReplies(ctx context.Context, conn websocket.Connection, req *WebsocketSubscriptionRequest) ([][]byte, error) {
	req.replies = make(chan []byte, len(req.Arguments))
	e.subscriptionRequests.send(conn, req)
	if err := conn.SendJSONMessage(ctx, websocketRequestEPL, req); err != nil {
		e.subscriptionRequests.stopAwaiting(req.ID)
		return nil, err
	}
	timer := time.NewTimer(websocketResponseMaxLimit)
	defer timer.Stop()
	replies := make([][]byte, 0, len(req.Arguments))
	var err error
	for err == nil && len(replies) < len(req.Arguments) {
		select {
		case reply := <-req.replies:
			replies = append(replies, reply)
		case <-timer.C:
			err = fmt.Errorf("%s %w %s: %d of %d replies", e.Name, websocket.ErrSignatureTimeout, req.ID, len(replies), len(req.Arguments))
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	e.subscriptionRequests.stopAwaiting(req.ID)
	for len(replies) < len(req.Arguments) {
		select {
		case reply := <-req.replies:
			replies = append(replies, reply)
		default:
			return replies, err
		}
	}
	return replies, nil
}

// subscriptionArgumentIdentity returns a subscription argument without its
// configuration and user ID, as OKX's replies need not echo them
func subscriptionArgumentIdentity(a *SubscriptionArgument) SubscriptionArgument {
	identity := *a
	identity.ExtraParams, identity.UserID = "", ""
	return identity
}

// chunkRequests splits subscriptions into subscribe or unsubscribe requests
// within maxConnByteLen as the connection sends them. An argument OKX may
// echo as another of a request's is sent in a later request, so each reply
// names its argument, and a subscription whose argument another sends, or
// keeps subscribed on the connection, sends none. An argument which OKX would
// refuse to subscribe, or which no request can carry, is refused before any
// request is sent.
func (e *Exchange) chunkRequests(conn websocket.Connection, subs subscription.List, operation string) ([]WebsocketSubscriptionRequest, error) {
	sent := e.sentSubscriptions(conn, subs)
	var requests []WebsocketSubscriptionRequest
	for len(subs) > 0 {
		req := WebsocketSubscriptionRequest{ID: e.MessageID(), Operation: operation}
		identities := make(map[SubscriptionArgument]bool)
		n := 0
		for ; n < len(subs); n++ {
			var arg SubscriptionArgument
			if err := json.Unmarshal([]byte(subs[n].QualifiedChannel), &arg); err != nil {
				return nil, err
			}
			if operation == operationSubscribe {
				if err := checkSubscriptionArgument(&arg); err != nil {
					return nil, err
				}
			}
			if sent[subs[n]] {
				identity := subscriptionArgumentIdentity(&arg)
				if identities[identity] {
					break
				}
				size, err := subscriptionRequestSize(&WebsocketSubscriptionRequest{ID: req.ID, Operation: operation, Arguments: append(req.Arguments, arg)})
				if err != nil {
					return nil, err
				}
				if size > maxConnByteLen {
					if len(req.Arguments) == 0 {
						return nil, fmt.Errorf("%w: a %s request of a %s argument alone is %d bytes, beyond %d", errSubscriptionArgumentTooLong, operation, arg.Channel, size, maxConnByteLen)
					}
					break
				}
				identities[identity] = true
				req.Arguments = append(req.Arguments, arg)
			}
			req.subs = append(req.subs, subs[n])
		}
		requests = append(requests, req)
		subs = subs[n:]
	}
	return requests, nil
}

// subscriptionRequestSize returns the length of a subscription request as a
// connection sends it: the connection's JSON writer ends the encoding with a
// newline
func subscriptionRequestSize(req *WebsocketSubscriptionRequest) (int, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(req); err != nil {
		return 0, err
	}
	return b.Len(), nil
}

// sentSubscriptions returns which of subs send their argument: the first with
// each argument, unless a subscription conn holds outside subs has it, as one
// subscription to an argument on a connection serves every subscription
// having it. A subscription in subs stands for the one conn holds with its
// key and argument, as an equivalent generated again does.
func (e *Exchange) sentSubscriptions(conn websocket.Connection, subs subscription.List) map[*subscription.Subscription]bool {
	listed := make(map[*subscription.Subscription]bool, len(subs))
	for _, s := range subs {
		listed[s] = true
	}
	subscribed := make(map[string]bool)
	if conn != nil {
		store := conn.Subscriptions()
		for _, s := range subs {
			if held := store.Get(subscriptionKey{s}); held != nil {
				listed[held] = true
			}
		}
		for _, s := range store.List() {
			if !listed[s] && e.Websocket.GetSubscription(s) == s {
				subscribed[s.QualifiedChannel] = true
			}
		}
	}
	sent := make(map[*subscription.Subscription]bool, len(subs))
	for _, s := range subs {
		if !subscribed[s.QualifiedChannel] {
			subscribed[s.QualifiedChannel] = true
			sent[s] = true
		}
	}
	return sent
}

// wsHandleData routes an incoming frame: replies to requests by their ID,
// login replies, errors and notices by their event, and pushes by their channel
func (e *Exchange) wsHandleData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	if bytes.Equal(respRaw, pongMsg) {
		return nil
	}
	if id, _ := jsonparser.GetString(respRaw, "id"); id != "" {
		if e.subscriptionRequests.route(id, respRaw) {
			return nil
		}
		return conn.RequireMatchWithData(id, respRaw)
	}
	var resp WebsocketEventResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return fmt.Errorf("%w unmarshalling %s", err, respRaw)
	}
	switch resp.Event {
	case operationLogin:
		return conn.RequireMatchWithData("login-response", respRaw)
	case "error":
		if slices.Contains(authConnErrorCodes, resp.Code) {
			return conn.RequireMatchWithData("login-response", respRaw)
		}
		return fmt.Errorf("%w: code %q channel %q: %s", errWebsocketRequestRejected, resp.Code, resp.Argument.Channel, resp.Message)
	case "notice":
		if resp.Code == noticeServiceUpgrade {
			e.reconnectWebsocket()
		}
		return e.Websocket.DataHandler.Send(ctx, &WebsocketNotice{Code: resp.Code, Message: resp.Message, ConnectionID: resp.ConnectionID})
	case "channel-conn-count-error":
		return e.endChannelSubscriptions(conn, resp.Channel, fmt.Errorf("%w: channel %q connection count %v", errChannelConnectionLimit, resp.Channel, resp.ConnectionCount))
	}
	// Event replies carry no data; pushes are decoded by their channel's handler.
	if _, _, _, err := jsonparser.Get(respRaw, "data"); err != nil {
		if errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return nil
		}
		return err
	}
	return e.wsHandlePush(ctx, conn, resp.Argument.Channel, respRaw)
}

// reconnectWebsocket shuts the websocket down for its connection monitor to
// reconnect, as OKX asks before it closes a connection for a service upgrade.
// The manager reconnects every connection together. The shutdown runs apart
// from the reader which received the notice, as it waits for the readers.
func (e *Exchange) reconnectWebsocket() {
	if (!e.Websocket.IsConnected() && !e.Websocket.IsConnecting()) || !e.websocketReconnecting.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer e.websocketReconnecting.Store(false)
		if err := e.Websocket.Shutdown(); err != nil && !errors.Is(err, websocket.ErrNotConnected) {
			log.Errorf(log.WebsocketMgr, "%s websocket: reconnecting for a service upgrade: %v", e.Name, err)
		}
	}()
}

// endChannelSubscriptions stops tracking the subscriptions to a channel which
// OKX ended on conn after acknowledging them, as it does on a connection
// beyond its limit of connections to the channel, and returns err naming
// them. A subscription to the channel still being subscribed on conn is left
// untracked once its request is answered.
func (e *Exchange) endChannelSubscriptions(conn websocket.Connection, channel string, err error) error {
	if conn == nil {
		return err
	}
	e.subscriptionRequests.end(conn, channel, err)
	store := conn.Subscriptions()
	var ended subscription.List
	for _, s := range store.List() {
		if subscriptionChannel(s) == channel && e.Websocket.GetSubscription(s) == s {
			ended = append(ended, s)
		}
	}
	if len(ended) == 0 {
		return err
	}
	errs := e.Websocket.RemoveSubscriptions(conn, ended...)
	for _, s := range ended {
		errs = common.AppendError(errs, store.Remove(s))
	}
	names := ended.QualifiedChannels()
	slices.Sort(names)
	return common.AppendError(fmt.Errorf("%w; ended %s", err, strings.Join(names, ", ")), errs)
}

// wsHandlePush decodes a push by its channel and relays it
func (e *Exchange) wsHandlePush(ctx context.Context, conn websocket.Connection, channel string, respRaw []byte) error {
	switch channel {
	case channelAccount:
		return e.wsProcessAccount(ctx, conn, respRaw)
	case channelPositions:
		return relayPush[WebsocketPositionsPush](ctx, e, respRaw)
	case channelBalanceAndPosition:
		return e.wsProcessBalanceAndPosition(ctx, respRaw)
	case channelLiquidationWarning:
		return relayPush[WebsocketPositionRiskWarningPush](ctx, e, respRaw)
	case channelAccountGreeks:
		return relayPush[WebsocketAccountGreeksPush](ctx, e, respRaw)
	case channelOrders:
		return e.wsProcessOrders(ctx, respRaw)
	case channelFills:
		return e.wsProcessFills(respRaw)
	case channelAlgoOrders:
		return relayPush[WebsocketAlgoOrdersPush](ctx, e, respRaw)
	case channelAlgoAdvance:
		return relayPush[WebsocketAdvanceAlgoOrdersPush](ctx, e, respRaw)
	case channelSpotGridOrders:
		return relayPush[WebsocketSpotGridAlgoOrdersPush](ctx, e, respRaw)
	case channelContractGridOrders:
		return relayPush[WebsocketContractGridAlgoOrdersPush](ctx, e, respRaw)
	case channelGridPositions:
		return relayPushWithRows(ctx, e, respRaw, func(p *WebsocketGridPositionsPush) int { return len(p.Data) })
	case channelGridSubOrders:
		return relayPushWithRows(ctx, e, respRaw, func(p *WebsocketGridSubOrdersPush) int { return len(p.Data) })
	case channelRecurringBuy:
		return relayPush[WebsocketRecurringBuyOrdersPush](ctx, e, respRaw)
	case channelLeadTradingNotification:
		return relayPush[WebsocketLeadTradingNotificationPush](ctx, e, respRaw)
	case channelTickers:
		return e.wsProcessTickers(ctx, respRaw)
	case channelTrades, channelAllTrades:
		return e.wsProcessTrades(ctx, respRaw)
	case channelOrderBooks, channelOrderBooksRPI, channelOrderBooks5, channelBBOTBT, channelOrderBooks50TBT, channelOrderBooksTBT:
		return e.wsProcessOrderBooks(ctx, conn, respRaw)
	case channelBBORPITBT:
		// The best levels of retail price improvement orders alone, which the
		// instrument's order book does not hold
		return relayPush[WebsocketOrderBookPush](ctx, e, respRaw)
	case channelOptionTrades:
		return e.wsProcessOptionTrades(ctx, respRaw)
	case channelCallAuctionDetails:
		return relayPush[WebsocketCallAuctionDetailsPush](ctx, e, respRaw)
	case channelRequestForQuotes:
		return relayPush[WebsocketRequestForQuotesPush](ctx, e, respRaw)
	case channelQuotes:
		return relayPush[WebsocketQuotesPush](ctx, e, respRaw)
	case channelStructureBlockTrades:
		return relayPush[WebsocketStructureBlockTradesPush](ctx, e, respRaw)
	case channelPublicStructureBlockTrades:
		return relayPush[WebsocketPublicStructureBlockTradesPush](ctx, e, respRaw)
	case channelPublicBlockTrades:
		return e.wsProcessPublicBlockTrades(ctx, respRaw)
	case channelBlockTickers:
		return relayPush[WebsocketBlockTickersPush](ctx, e, respRaw)
	case channelSpreadOrders:
		return e.wsProcessSpreadOrders(ctx, respRaw)
	case channelSpreadTrades:
		return e.wsProcessSpreadTrades(respRaw)
	case channelSpreadOrderBooks5, channelSpreadBBOTBT, channelSpreadOrderBooksTBT:
		return e.wsProcessSpreadOrderBooks(ctx, conn, respRaw)
	case channelSpreadPublicTrades:
		return e.wsProcessSpreadPublicTrades(ctx, respRaw)
	case channelSpreadTickers:
		return e.wsProcessSpreadTickers(ctx, respRaw)
	case channelInstruments:
		return e.wsProcessInstruments(ctx, respRaw)
	case channelEventContractMarkets:
		return relayPush[WebsocketEventContractMarketsPush](ctx, e, respRaw)
	case channelOpenInterest:
		return relayPush[WebsocketOpenInterestPush](ctx, e, respRaw)
	case channelFundingRate:
		return relayPush[WebsocketFundingRatePush](ctx, e, respRaw)
	case channelPriceLimit:
		return relayPush[WebsocketPriceLimitPush](ctx, e, respRaw)
	case channelOptionSummary:
		return relayPush[WebsocketOptionSummaryPush](ctx, e, respRaw)
	case channelEstimatedPrice:
		return relayPush[WebsocketEstimatedPricePush](ctx, e, respRaw)
	case channelMarkPrice:
		return relayPush[WebsocketMarkPricePush](ctx, e, respRaw)
	case channelIndexTickers:
		return relayPush[WebsocketIndexTickersPush](ctx, e, respRaw)
	case channelLiquidationOrders:
		return relayPush[WebsocketLiquidationOrdersPush](ctx, e, respRaw)
	case channelADLWarning:
		return relayPush[WebsocketADLWarningPush](ctx, e, respRaw)
	case channelEconomicCalendar:
		return relayPush[WebsocketEconomicCalendarPush](ctx, e, respRaw)
	case channelDepositInfo:
		return relayPush[WebsocketDepositInfoPush](ctx, e, respRaw)
	case channelWithdrawalInfo:
		return relayPush[WebsocketWithdrawalInfoPush](ctx, e, respRaw)
	case channelStatus:
		return relayPush[WebsocketStatusPush](ctx, e, respRaw)
	}
	switch {
	case strings.HasPrefix(channel, channelSpreadCandle):
		return e.wsProcessSpreadCandles(ctx, respRaw)
	case strings.HasPrefix(channel, channelMarkPriceCandle):
		return relayPush[WebsocketMarkPriceCandlesticksPush](ctx, e, respRaw)
	case strings.HasPrefix(channel, channelIndexCandle):
		return relayPush[WebsocketIndexCandlesticksPush](ctx, e, respRaw)
	case strings.HasPrefix(channel, channelCandle):
		return e.wsProcessCandles(ctx, respRaw)
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: e.Name + websocket.UnhandledMessage + string(respRaw)})
}

// relayPush decodes a push into its type and relays it to the data handler
func relayPush[T any](ctx context.Context, e *Exchange, respRaw []byte) error {
	resp := new(T)
	if err := json.Unmarshal(respRaw, resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// relayPushWithRows decodes a push into its type and relays it if it holds any
// rows; OKX says to ignore the empty pushes of the grid positions and sub
// orders channels
func relayPushWithRows[T any](ctx context.Context, e *Exchange, respRaw []byte, rows func(*T) int) error {
	resp := new(T)
	if err := json.Unmarshal(respRaw, resp); err != nil {
		return err
	}
	if rows(resp) == 0 {
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// channelName returns the OKX channel a subscription selects for an asset
func channelName(s *subscription.Subscription, a asset.Item) (string, error) {
	spread := a == asset.Spread
	switch s.Channel {
	case subscription.TickerChannel:
		if spread {
			return channelSpreadTickers, nil
		}
		return channelTickers, nil
	case subscription.AllTradesChannel:
		if spread {
			return channelSpreadPublicTrades, nil
		}
		return channelTrades, nil
	case subscription.OrderbookChannel:
		if spread {
			return channelSpreadOrderBooks5, nil
		}
		return channelOrderBooks, nil
	case subscription.CandlesChannel:
		suffix, err := candleChannelSuffix(s.Interval)
		if err != nil {
			return "", err
		}
		if spread {
			if suffix == "1s" {
				return "", fmt.Errorf("%w: %s for spreads", errUnsupportedCandleInterval, s.Interval)
			}
			return channelSpreadCandle + suffix, nil
		}
		return channelCandle + suffix, nil
	case subscription.MyOrdersChannel:
		if spread {
			return channelSpreadOrders, nil
		}
		return channelOrders, nil
	case subscription.MyTradesChannel:
		if spread {
			return channelSpreadTrades, nil
		}
		return channelFills, nil
	case subscription.MyAccountChannel:
		return channelAccount, nil
	}
	return s.Channel, nil
}

// subscriptionChannel returns the OKX channel of an expanded subscription
func subscriptionChannel(s *subscription.Subscription) string {
	channel, err := jsonparser.GetString([]byte(s.QualifiedChannel), "channel")
	if err != nil {
		return ""
	}
	return channel
}

// channelConnection returns the message filter of the connection serving a
// channel; channels the package does not describe are sent on the public
// connection
func channelConnection(channel string) string {
	if c, ok := websocketChannels[channel]; ok {
		return c.connection
	}
	return publicConnection
}

// requiresLogin returns whether a channel is served by a connection which logs in
func requiresLogin(channel string) bool {
	switch channelConnection(channel) {
	case publicLoginConnection, privateConnection, businessLoginConnection:
		return true
	}
	return false
}

// subscriptionKey identifies a subscription by its argument beside the
// channel, asset, pairs, levels and interval subscription.ExactKey compares,
// so subscriptions differing only in their parameters, such as account
// channels of two update intervals, are told apart
type subscriptionKey struct {
	*subscription.Subscription
}

// Match implements subscription.MatchableKey
func (k subscriptionKey) Match(eachKey subscription.MatchableKey) bool {
	if eachKey == nil {
		return false
	}
	s := eachKey.GetSubscription()
	return s != nil && s.QualifiedChannel == k.QualifiedChannel && subscription.ExactKey{Subscription: k.Subscription}.Match(eachKey)
}

// GetSubscription implements subscription.MatchableKey
func (k subscriptionKey) GetSubscription() *subscription.Subscription {
	return k.Subscription
}

// String implements subscription.MatchableKey
func (k subscriptionKey) String() string {
	return subscription.ExactKey{Subscription: k.Subscription}.String() + " " + k.QualifiedChannel
}

// GetQualifiedChannel implements subscription.QualifiedChannelKey
func (k subscriptionKey) GetQualifiedChannel() string {
	return k.QualifiedChannel
}

// generateSubscriptions returns the configured subscriptions served by the
// connection the message filter selects, each keyed by its argument
func (e *Exchange) generateSubscriptions(connection string) (subscription.List, error) {
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, s := range e.Features.Subscriptions {
		if s.QualifiedChannel != "" {
			// Expansion passes an expanded subscription through as it is, and the
			// configuration's own must not take the key set below
			s = s.Clone()
		}
		if !s.Authenticated {
			// Login channels are only generated when the websocket can log in
			a := s.Asset
			if a == asset.All || a == asset.Empty {
				a = asset.Spot
			}
			if channel, err := channelName(s, a); err == nil && requiresLogin(channel) {
				s = s.Clone()
				s.Authenticated = true
			}
		}
		subs = append(subs, s)
	}
	expanded, err := subs.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	served := make(subscription.List, 0, len(expanded))
	for _, s := range expanded {
		if channelConnection(subscriptionChannel(s)) == connection {
			s.SetKey(subscriptionKey{s})
			served = append(served, s)
		}
	}
	return served, nil
}

// GetSubscriptionTemplate returns a subscription channel template
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":     channelName,
		"channelScope":    channelScopeName,
		"channelArgument": e.channelArgument,
	}).Parse(subTplText)
}

const subTplText = `
{{- range $asset, $pairs := $.AssetPairs }}
	{{- $name := channelName $.S $asset }}
	{{- $scope := channelScope $name $asset }}
	{{- if eq $scope "pair" }}
		{{- range $p := $pairs -}}
			{{ channelArgument $.S $name $asset $p }}
			{{ $.PairSeparator }}
		{{- end -}}
	{{- else if eq $scope "asset" -}}
		{{ channelArgument $.S $name $asset nil }}
	{{- end }}
	{{- $.AssetSeparator }}
{{- end }}
`

// channelScopeName returns whether a channel subscribes once per pair, once
// for an asset, or not at all for an asset it does not accept
func channelScopeName(channel string, a asset.Item) string {
	c, ok := websocketChannels[channel]
	if !ok || c.scope == channelScopeNone {
		return "asset"
	}
	if a != asset.Empty && !slices.Contains(c.assets, a) {
		return "none"
	}
	if c.scope == channelScopeAsset || a == asset.Empty {
		return "asset"
	}
	return "pair"
}

// subscriptionParameters are the subscription Params passed into a channel's argument
var subscriptionParameters = []string{"instType", "instFamily", "instId", "sprdId", "algoId", "ccy", "extraParams"}

// channelArgument returns the JSON argument of a subscription to a channel for
// an asset and, for channels subscribed per pair, a pair
func (e *Exchange) channelArgument(s *subscription.Subscription, channel string, a asset.Item, p any) (string, error) {
	arg := SubscriptionArgument{Channel: channel}
	for _, param := range subscriptionParameters {
		v, ok := s.Params[param]
		if !ok {
			continue
		}
		str, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %w", errSubscriptionParameter, common.GetTypeAssertError("string", v, param))
		}
		switch param {
		case "instType":
			arg.InstrumentType = str
		case "instFamily":
			arg.InstrumentFamily = str
		case "instId":
			pair, err := currency.NewPairFromString(str)
			if err != nil {
				return "", fmt.Errorf("%w: instId %q: %w", errSubscriptionParameter, str, err)
			}
			arg.InstrumentID = pair
		case "sprdId":
			pair, err := currency.NewPairFromString(str)
			if err != nil {
				return "", fmt.Errorf("%w: sprdId %q: %w", errSubscriptionParameter, str, err)
			}
			arg.SpreadID = pair
		case "algoId":
			arg.AlgoID = str
		case "ccy":
			arg.Currency = currency.NewCode(str)
		case "extraParams":
			arg.ExtraParams = str
		}
	}
	if c, ok := websocketChannels[channel]; ok && a != asset.Empty {
		instType := ""
		if a != asset.Spread {
			instType = GetInstrumentTypeFromAssetItem(a)
		}
		pair, _ := p.(currency.Pair)
		switch c.scope {
		case channelScopeAsset:
			arg.InstrumentType = instType
		case channelScopeInstrument:
			arg.InstrumentID = pair
		case channelScopeInstrumentAndType:
			arg.InstrumentType = instType
			arg.InstrumentID = pair
		case channelScopeIndex:
			arg.InstrumentID = indexInstrumentID(pair)
		case channelScopeSpread:
			arg.SpreadID = pair
		}
	}
	if channel == channelEventContractMarkets && arg.InstrumentType == "" {
		arg.InstrumentType = instTypeEvents
	}
	if err := checkSubscriptionArgument(&arg); err != nil {
		return "", err
	}
	b, err := json.Marshal(arg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// checkSubscriptionArgument returns an error when a subscription argument lacks
// a parameter its channel requires, or names an instrument type its channel
// does not take
func checkSubscriptionArgument(arg *SubscriptionArgument) error {
	if c, ok := websocketChannels[arg.Channel]; ok {
		switch {
		case (c.scope == channelScopeInstrument && arg.Channel != channelFills) || c.scope == channelScopeIndex:
			if arg.InstrumentID.IsEmpty() {
				return fmt.Errorf("%w: %s needs an instrument", currency.ErrCurrencyPairEmpty, arg.Channel)
			}
		case c.scope == channelScopeInstrumentAndType:
			if arg.InstrumentType == "" || (arg.InstrumentID.IsEmpty() && arg.InstrumentFamily == "") {
				return fmt.Errorf("%w: %s needs an instrument type and an instrument or instrument family", errSubscriptionParameter, arg.Channel)
			}
		case c.scope == channelScopeSpread:
			if arg.SpreadID.IsEmpty() {
				return fmt.Errorf("%w: %s needs a spread", currency.ErrCurrencyPairEmpty, arg.Channel)
			}
		case c.scope == channelScopeAsset && !slices.Equal(c.assets, spreadAssets):
			if arg.InstrumentType == "" {
				return fmt.Errorf("%w: %s needs an instrument type", errInvalidInstrumentType, arg.Channel)
			}
		}
	}
	switch arg.Channel {
	case channelEventContractMarkets:
		if arg.InstrumentType != instTypeEvents {
			return fmt.Errorf("%w: %s takes instrument type %s, not %q", errSubscriptionParameter, arg.Channel, instTypeEvents, arg.InstrumentType)
		}
	case channelOptionSummary:
		if arg.InstrumentFamily == "" {
			return fmt.Errorf("%w: %s needs an instrument family", errSubscriptionParameter, arg.Channel)
		}
	case channelGridPositions, channelGridSubOrders:
		if arg.AlgoID == "" {
			return fmt.Errorf("%w: %s needs an algo ID", errSubscriptionParameter, arg.Channel)
		}
	}
	return nil
}

// indexInstrumentID returns the ID of the index an instrument is priced from:
// the instrument itself for spot, and the underlying for derivatives, such as
// BTC-USD for BTC-USD-SWAP, BTC-USD_UM-261225 and BTC-USD-261225-60000-C
func indexInstrumentID(p currency.Pair) currency.Pair {
	quote := p.Quote.String()
	if i := strings.IndexAny(quote, "-_"); i != -1 {
		quote = quote[:i]
	}
	return currency.NewPairWithDelimiter(p.Base.String(), quote, currency.DashDelimiter)
}

// isAssetChannel returns whether a subscription's channel subscribes once per
// asset, so spot and margin subscriptions to it differ in their arguments
func isAssetChannel(s *subscription.Subscription) bool {
	a := s.Asset
	if a == asset.All || a == asset.Empty {
		a = asset.Spot
	}
	channel, err := channelName(s, a)
	if err != nil {
		return false
	}
	c, ok := websocketChannels[channel]
	return ok && c.scope == channelScopeAsset
}

func inverseSpotMarginSubscription(sub *subscription.Subscription) (*subscription.Subscription, bool) {
	if sub == nil || isAssetChannel(sub) || (sub.Asset != asset.Spot && sub.Asset != asset.Margin) {
		return nil, false
	}
	inverse := sub.Clone()
	switch sub.Asset {
	case asset.Spot:
		inverse.Asset = asset.Margin
	case asset.Margin:
		inverse.Asset = asset.Spot
	}
	return inverse, true
}

// trackEquivalentSubscriptionsOnExistingConnection identifies spot/margin equivalent subscriptions that can be logically attached to an existing connection, sends any required outbound subscribe payloads, and returns both remaining and tracked subscriptions for manager-level bookkeeping.
func (e *Exchange) trackEquivalentSubscriptionsOnExistingConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) (remaining, tracked subscription.List, err error) {
	if conn == nil || len(subs) == 0 {
		return subs, nil, nil
	}

	remaining = make(subscription.List, 0, len(subs))
	for _, sub := range subs {
		inverse, ok := inverseSpotMarginSubscription(sub)
		if !ok || conn.Subscriptions().Get(inverse) == nil {
			remaining = append(remaining, sub)
			continue
		}
		tracked = append(tracked, sub)
	}
	if len(tracked) == 0 {
		return remaining, nil, nil
	}
	requests, err := e.chunkRequests(conn, tracked, operationSubscribe)
	if err != nil {
		return nil, nil, err
	}
	outcomes := make(map[SubscriptionArgument]bool)
	for i := range requests {
		done, err := e.sendSubscriptionRequest(ctx, conn, &requests[i], outcomes)
		if _, endErr := endedSubscriptions(done, e.subscriptionRequests.settle(requests[i].ID)); err != nil || endErr != nil {
			return nil, nil, common.AppendError(err, endErr)
		}
	}
	for _, sub := range tracked {
		if err := e.refreshEquivalentOrderbookSnapshot(ctx, sub); err != nil {
			return nil, nil, err
		}
	}
	return remaining, tracked, nil
}
