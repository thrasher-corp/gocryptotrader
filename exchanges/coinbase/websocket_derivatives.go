package coinbase

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
	"sync"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// The gateway's subscription channels, which a subscription names, other than through a standard channel, to subscribe
// to; each needs a subscription to a USDC or coin margined futures asset, or to no asset
const (
	ChannelDerivativesAnnouncements                  = "announcements"
	ChannelDerivativesBook                           = "book"
	ChannelDerivativesChartTrades                    = "chart.trades"
	ChannelDerivativesPriceIndex                     = "deribit_price_index"
	ChannelDerivativesPriceRanking                   = "deribit_price_ranking"
	ChannelDerivativesPriceStatistics                = "deribit_price_statistics"
	ChannelDerivativesVolatilityIndex                = "deribit_volatility_index"
	ChannelDerivativesEstimatedExpirationPrice       = "estimated_expiration_price"
	ChannelDerivativesIncrementalTicker              = "incremental_ticker"
	ChannelDerivativesInstrumentCreation             = "instrument.creation"
	ChannelDerivativesInstrumentState                = "instrument.state"
	ChannelDerivativesMarkPriceOptions               = "markprice.options"
	ChannelDerivativesPerpetual                      = "perpetual"
	ChannelDerivativesPlatformState                  = "platform_state"
	ChannelDerivativesPublicMethodsState             = "platform_state.public_methods_state"
	ChannelDerivativesQuote                          = "quote"
	ChannelDerivativesTicker                         = "ticker"
	ChannelDerivativesTrades                         = "trades"
	ChannelDerivativesUserAccessLog                  = "user.access_log"
	ChannelDerivativesUserChanges                    = "user.changes"
	ChannelDerivativesUserComboTrades                = "user.combo_trades"
	ChannelDerivativesUserIsolatedChanges            = "user.isolated.changes"
	ChannelDerivativesUserIsolatedLiquidation        = "user.isolated.liquidation"
	ChannelDerivativesUserIsolatedOrders             = "user.isolated.orders"
	ChannelDerivativesUserIsolatedPortfolio          = "user.isolated.portfolio"
	ChannelDerivativesUserIsolatedTrades             = "user.isolated.trades"
	ChannelDerivativesUserLock                       = "user.lock"
	ChannelDerivativesUserOrders                     = "user.orders"
	ChannelDerivativesUserPortfolio                  = "user.portfolio"
	ChannelDerivativesUserPositionLock               = "user.position_lock"
	ChannelDerivativesUserTrades                     = "user.trades"
	ChannelDerivativesUserTrailingOrdersTriggerPrice = "user.trailing_orders.trigger_price"
)

const (
	// derivativesHeartbeatInterval has the gateway send a heartbeat every 15 seconds, so a quiet connection keeps its
	// traffic up and a stale one is noticed within the traffic timeout
	derivativesHeartbeatInterval = 15 * time.Second
	// derivativesReauthenticateMargin is how long before a websocket session expires it is authenticated again, and
	// derivativesReauthenticateRetry how long after a failed renewal it is tried again
	derivativesReauthenticateMargin = 5 * time.Minute
	derivativesReauthenticateRetry  = 30 * time.Second

	derivativesMethodSubscription = "subscription"
	derivativesMethodHeartbeat    = "heartbeat"
	// derivativesHeartbeatTestRequest is the heartbeat type the gateway closes the connection for unless a public/test
	// request answers it
	derivativesHeartbeatTestRequest = "test_request"

	// Channel intervals: raw sends each event as it happens, 100ms aggregates over 100 milliseconds and agg2 over about
	// a second
	derivativesIntervalRaw       = "raw"
	derivativesIntervalHundredMS = "100ms"
	derivativesIntervalAgg2      = "agg2"

	// derivativesChannelAny is the value a kind or currency channel parameter takes to select every kind or currency
	derivativesChannelAny = "any"
)

var (
	errDerivativesWebsocketAuthentication = errors.New("gateway websocket authentication failed")
	errDerivativesChannelParameter        = errors.New("invalid gateway channel parameter")
	errDerivativesChannelNeedsAuth        = errors.New("gateway channel needs an authenticated subscription")
	errDerivativesChannelNeedsInstrument  = errors.New("gateway channel needs an asset and pairs")
	errDerivativesSubscriptionUnconfirmed = errors.New("gateway channel not acknowledged")
	errDerivativesUnknownInstrument       = errors.New("unknown gateway instrument")
)

// derivativesChannel describes how a gateway channel is qualified
type derivativesChannel struct {
	// instrument channels are qualified per pair by its instrument name, for a subscription with an asset
	instrument bool
	// kindCurrency channels are qualified by the kind and currency parameters, each any when unset, for a subscription
	// without an asset
	kindCurrency bool
	// interval channels end with the subscription's interval: raw, 100ms, or agg2 for a second; unset is 100ms
	interval bool
	// resolution channels end with the subscription's interval as a candle resolution
	resolution bool
	// indexName channels are qualified by the index_name parameter, such as btc_usd
	indexName bool
	// currency channels are qualified by the currency parameter, any when unset
	currency bool
	// private channels need an authenticated subscription, and stream on the gateway's private connections
	private bool
}

// derivativesChannels lists every gateway channel a subscription may name
var derivativesChannels = map[string]derivativesChannel{
	ChannelDerivativesAnnouncements:                  {},
	ChannelDerivativesBook:                           {instrument: true, interval: true},
	ChannelDerivativesChartTrades:                    {instrument: true, resolution: true},
	ChannelDerivativesPriceIndex:                     {indexName: true},
	ChannelDerivativesPriceRanking:                   {indexName: true},
	ChannelDerivativesPriceStatistics:                {indexName: true},
	ChannelDerivativesVolatilityIndex:                {indexName: true},
	ChannelDerivativesEstimatedExpirationPrice:       {indexName: true},
	ChannelDerivativesIncrementalTicker:              {instrument: true},
	ChannelDerivativesInstrumentCreation:             {kindCurrency: true},
	ChannelDerivativesInstrumentState:                {kindCurrency: true},
	ChannelDerivativesMarkPriceOptions:               {indexName: true},
	ChannelDerivativesPerpetual:                      {instrument: true, interval: true},
	ChannelDerivativesPlatformState:                  {},
	ChannelDerivativesPublicMethodsState:             {},
	ChannelDerivativesQuote:                          {instrument: true},
	ChannelDerivativesTicker:                         {instrument: true, interval: true},
	ChannelDerivativesTrades:                         {instrument: true, kindCurrency: true, interval: true},
	ChannelDerivativesUserAccessLog:                  {private: true},
	ChannelDerivativesUserChanges:                    {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelDerivativesUserComboTrades:                {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelDerivativesUserIsolatedChanges:            {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelDerivativesUserIsolatedLiquidation:        {private: true},
	ChannelDerivativesUserIsolatedOrders:             {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelDerivativesUserIsolatedPortfolio:          {currency: true, private: true},
	ChannelDerivativesUserIsolatedTrades:             {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelDerivativesUserLock:                       {private: true},
	ChannelDerivativesUserOrders:                     {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelDerivativesUserPortfolio:                  {currency: true, private: true},
	ChannelDerivativesUserPositionLock:               {private: true},
	ChannelDerivativesUserTrades:                     {instrument: true, kindCurrency: true, interval: true, private: true},
	ChannelDerivativesUserTrailingOrdersTriggerPrice: {currency: true, private: true},
}

// derivativesStandardChannels maps the standard channels the gateway serves to its channels; the account channel is
// Advanced Trade's user channel
var derivativesStandardChannels = map[string]string{
	subscription.TickerChannel:    ChannelDerivativesTicker,
	subscription.OrderbookChannel: ChannelDerivativesBook,
	subscription.AllTradesChannel: ChannelDerivativesTrades,
	subscription.CandlesChannel:   ChannelDerivativesChartTrades,
	subscription.MyOrdersChannel:  ChannelDerivativesUserOrders,
	subscription.MyTradesChannel:  ChannelDerivativesUserTrades,
}

// derivativesChannelsByLength lists the channel names longest first, so a notification's channel is matched to the
// most specific name it starts with
var derivativesChannelsByLength = func() []string {
	names := slices.Collect(maps.Keys(derivativesChannels))
	slices.SortFunc(names, func(a, b string) int { return cmp.Or(len(b)-len(a), strings.Compare(a, b)) })
	return names
}()

// derivativesChannelName returns the gateway channel a subscription refers to: a standard channel's gateway channel, or
// the channel it names
func derivativesChannelName(s *subscription.Subscription) (string, error) {
	if name, ok := derivativesStandardChannels[s.Channel]; ok {
		return name, nil
	}
	if _, ok := derivativesChannels[s.Channel]; ok {
		return s.Channel, nil
	}
	return "", fmt.Errorf("%w: %s", subscription.ErrNotSupported, s.Channel)
}

// qualifyDerivativesInstrumentChannel returns a subscription's channel for one of its instruments
func qualifyDerivativesInstrumentChannel(s *subscription.Subscription, p currency.Pair) (string, error) {
	name, channel, err := validateDerivativesSubscription(s)
	if err != nil {
		return "", err
	}
	qualified := name + "." + p.String()
	if name == ChannelDerivativesBook && s.Levels != 0 {
		group, err := derivativesBookGroup(s.Params)
		if err != nil {
			return "", err
		}
		switch s.Levels {
		case 1, 10, 20:
		default:
			return "", fmt.Errorf("%w: %d, as a grouped book sends 1, 10 or 20 levels", subscription.ErrInvalidLevel, s.Levels)
		}
		qualified += "." + group + "." + strconv.Itoa(s.Levels)
	}
	return appendDerivativesInterval(qualified, s, &channel)
}

// qualifyDerivativesChannel returns a subscription's channel when it is not qualified per instrument
func qualifyDerivativesChannel(s *subscription.Subscription) (string, error) {
	name, channel, err := validateDerivativesSubscription(s)
	if err != nil {
		return "", err
	}
	qualified := name
	switch {
	case channel.kindCurrency:
		kind, err := derivativesChannelParameter(s.Params, "kind", derivativesChannelAny)
		if err != nil {
			return "", err
		}
		ccy, err := derivativesChannelCurrency(s.Params)
		if err != nil {
			return "", err
		}
		qualified += "." + kind + "." + ccy
	case channel.instrument:
		return "", fmt.Errorf("%w: %s", errDerivativesChannelNeedsInstrument, name)
	case channel.indexName:
		index, err := derivativesChannelParameter(s.Params, "index_name", "")
		if err != nil {
			return "", err
		}
		qualified += "." + index
	case channel.currency:
		ccy, err := derivativesChannelCurrency(s.Params)
		if err != nil {
			return "", err
		}
		qualified += "." + ccy
	}
	return appendDerivativesInterval(qualified, s, &channel)
}

// validateDerivativesSubscription returns the gateway channel a subscription refers to, checking a private channel is
// subscribed authenticated
func validateDerivativesSubscription(s *subscription.Subscription) (string, derivativesChannel, error) {
	name, err := derivativesChannelName(s)
	if err != nil {
		return "", derivativesChannel{}, err
	}
	channel := derivativesChannels[name]
	if channel.private && !s.Authenticated {
		return "", derivativesChannel{}, fmt.Errorf("%w: %s", errDerivativesChannelNeedsAuth, name)
	}
	return name, channel, nil
}

// appendDerivativesInterval appends the subscription's interval to a channel that takes one; an unset interval
// aggregates over 100 milliseconds. The gateway's streams send raw channels unauthenticated, although its docs, as
// Deribit's, say otherwise
func appendDerivativesInterval(qualified string, s *subscription.Subscription, channel *derivativesChannel) (string, error) {
	switch {
	case channel.resolution:
		resolution, err := derivativesChartResolution(s.Interval)
		if err != nil {
			return "", err
		}
		// A candle GCT handles starts at a multiple of its interval in UTC, which the gateway's longer candles do not
		if !derivativesCandleAligned(s.Interval) {
			return "", fmt.Errorf("%w: %s, as the gateway aligns its candles of that interval to its 08:00 UTC daily settlement", kline.ErrUnsupportedInterval, s.Interval)
		}
		return qualified + "." + resolution, nil
	case channel.interval:
		var interval string
		switch s.Interval {
		case 0, kline.HundredMilliseconds:
			interval = derivativesIntervalHundredMS
		case kline.ThousandMilliseconds:
			interval = derivativesIntervalAgg2
		case kline.Raw:
			interval = derivativesIntervalRaw
		default:
			return "", fmt.Errorf("%w: %s, as the gateway aggregates over 100ms, 1s or raw", kline.ErrUnsupportedInterval, s.Interval)
		}
		return qualified + "." + interval, nil
	}
	return qualified, nil
}

// derivativesResolutionInterval returns the interval a chart resolution names
func derivativesResolutionInterval(resolution string) (kline.Interval, error) {
	if resolution == "1D" {
		return kline.OneDay, nil
	}
	minutes, err := strconv.ParseUint(resolution, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: resolution %q", kline.ErrInvalidInterval, resolution)
	}
	interval := kline.Interval(time.Duration(minutes) * time.Minute) //nolint:gosec // Resolutions are a few minutes
	if _, err := derivativesChartResolution(interval); err != nil {
		return 0, err
	}
	return interval, nil
}

// derivativesBookGroup returns a grouped book's price grouping from the group parameter, none when unset; a number, as
// configuration supplies it, is formatted as the gateway names it
func derivativesBookGroup(params map[string]any) (string, error) {
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
		return "", fmt.Errorf("%w: group %v", errDerivativesChannelParameter, value)
	}
	switch group {
	case "none", "1", "2", "5", "10", "25", "100", "250":
		return group, nil
	default:
		return "", fmt.Errorf("%w: group %q", errDerivativesChannelParameter, group)
	}
}

// derivativesChannelParameter returns a string channel parameter, or the default when it is unset; a parameter without
// a default is required
func derivativesChannelParameter(params map[string]any, name, defaultValue string) (string, error) {
	switch value := params[name].(type) {
	case nil:
		if defaultValue == "" {
			return "", fmt.Errorf("%w: %s is required", errDerivativesChannelParameter, name)
		}
		return defaultValue, nil
	case string:
		if value == "" || strings.Contains(value, ".") {
			return "", fmt.Errorf("%w: %s %q", errDerivativesChannelParameter, name, value)
		}
		return value, nil
	default:
		return "", fmt.Errorf("%w: %s %v", errDerivativesChannelParameter, name, value)
	}
}

// derivativesChannelCurrency returns the currency parameter, upper case as the gateway names currencies, or any when it
// is unset
func derivativesChannelCurrency(params map[string]any) (string, error) {
	var ccy string
	switch value := params["currency"].(type) {
	case nil:
		return derivativesChannelAny, nil
	case string:
		ccy = value
	case currency.Code:
		ccy = value.String()
	default:
		return "", fmt.Errorf("%w: currency %v", errDerivativesChannelParameter, value)
	}
	if strings.EqualFold(ccy, derivativesChannelAny) {
		return derivativesChannelAny, nil
	}
	if ccy == "" || strings.Contains(ccy, ".") {
		return "", fmt.Errorf("%w: currency %q", errDerivativesChannelParameter, ccy)
	}
	return strings.ToUpper(ccy), nil
}

// derivativesChannelOf returns the gateway channel an expanded subscription streams on, if any
func derivativesChannelOf(s *subscription.Subscription) (derivativesChannel, bool) {
	if s.Asset != asset.Empty && !isDerivativesAsset(s.Asset) {
		return derivativesChannel{}, false
	}
	if s.Asset == asset.Empty {
		if _, ok := advancedTradeChannels[s.Channel]; ok {
			return derivativesChannel{}, false
		}
	}
	name, err := derivativesChannelName(s)
	if err != nil {
		return derivativesChannel{}, false
	}
	return derivativesChannels[name], true
}

// generateDerivativesStreamsSubscriptions returns the subscriptions the gateway's public streams carry
func (e *Exchange) generateDerivativesStreamsSubscriptions() (subscription.List, error) {
	return e.generateSubscriptions(func(s *subscription.Subscription) bool {
		ch, ok := derivativesChannelOf(s)
		return ok && !ch.private
	})
}

// generateDerivativesPrivateSubscriptions returns the subscriptions the gateway's private connections carry
func (e *Exchange) generateDerivativesPrivateSubscriptions() (subscription.List, error) {
	return e.generateSubscriptions(func(s *subscription.Subscription) bool {
		ch, ok := derivativesChannelOf(s)
		return ok && ch.private
	})
}

// derivativesInstrumentPair returns the asset and pair of a perpetual the gateway names: a USDC margined one is named
// as BTC_USDC-PERPETUAL, its pair BTC_USDC-PERPETUAL with an underscore, and a coin margined one as BTC-PERPETUAL
func derivativesInstrumentPair(instrument string) (asset.Item, currency.Pair, error) {
	if base, ok := strings.CutSuffix(instrument, "_USDC-PERPETUAL"); ok && base != "" && !strings.ContainsAny(base, "_-") {
		return asset.USDCMarginedFutures, currency.NewPairWithDelimiter(base, "USDC-PERPETUAL", currency.UnderscoreDelimiter), nil
	}
	if base, ok := strings.CutSuffix(instrument, "-PERPETUAL"); ok && base != "" && !strings.ContainsAny(base, "_-") {
		return asset.CoinMarginedFutures, currency.NewPairWithDelimiter(base, "PERPETUAL", currency.DashDelimiter), nil
	}
	return asset.Empty, currency.EMPTYPAIR, fmt.Errorf("%w: %q", errDerivativesUnknownInstrument, instrument)
}

// wsDerivativesConnect dials a gateway connection and asks it for heartbeats once the connection's reader runs, as the
// request waits for its reply
func (e *Exchange) wsDerivativesConnect(ctx context.Context, conn websocket.Connection) error {
	e.wsConnections.pruneClosed()
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	go func() {
		params := &DerivativesHeartbeatWireRequest{Interval: derivativesHeartbeatInterval.Seconds()}
		var resp string
		// A connection closed before heartbeats are set needs none
		if err := e.WebsocketAPI().sendRequestOnConnection(ctx, conn, derivativesNonMatchingEPL, "public/set_heartbeat", params, &resp); err != nil && ctx.Err() == nil && isConnected(conn) {
			log.Errorf(log.WebsocketMgr, "%s gateway websocket heartbeats could not be set: %v", e.Name, err)
		}
	}()
	return nil
}

// DerivativesHeartbeatWireRequest holds public/set_heartbeat's params, the interval in seconds
type DerivativesHeartbeatWireRequest struct {
	Interval float64 `json:"interval"`
}

// wsDerivativesAuthenticate authenticates a gateway private connection with a CDP JWT, and authenticates it again
// before its session expires. An account without the gateway's derivatives cannot authenticate; that leaves the other
// connections, and authenticated Advanced Trade channels, up, and the gateway refuses the connection's private
// subscriptions
func (e *Exchange) wsDerivativesAuthenticate(ctx context.Context, conn websocket.Connection) error {
	e.derivativesSessions.set(conn, "")
	if err := e.wsDerivativesSendAuth(ctx, conn); err != nil {
		log.Errorf(log.ExchangeSys, "%s gateway websocket authentication failed: %v", e.Name, err)
	}
	return nil
}

// wsDerivativesSendAuth sends public/auth on a connection and schedules the next before the session expires
func (e *Exchange) wsDerivativesSendAuth(ctx context.Context, conn websocket.Connection) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	// A websocket JWT is not bound to a request, as the connection it authenticates outlives any one request
	jwt, _, err := e.GetJWT(ctx, "")
	if err != nil {
		return err
	}
	var resp *DerivativesAuthResponse
	if err := e.WebsocketAPI().sendRequestOnConnection(ctx, conn, derivativesNonMatchingEPL, derivativesAuthMethod, &DerivativesAuthWireRequest{GrantType: derivativesGrantType, Token: jwt}, &resp); err != nil {
		return fmt.Errorf("%w: %w", errDerivativesWebsocketAuthentication, err)
	}
	if resp == nil {
		return fmt.Errorf("%w: no session issued", errDerivativesWebsocketAuthentication)
	}
	authentication := e.derivativesSessions.authenticated(conn, creds.Key)
	if resp.ExpiresIn > 0 {
		e.wsDerivativesScheduleReauthentication(ctx, conn, authentication, time.Duration(resp.ExpiresIn)*time.Second)
	}
	return nil
}

// wsDerivativesScheduleReauthentication authenticates a connection again before the session an authentication began
// expires, and again after each failure for as long as wsDerivativesRenewSession asks. The renewal replaces any the
// connection has pending, as each authentication schedules one, and AuthenticateWebsocket authenticates a connection
// already authenticated. It belongs to the connection, not to the authentication's caller, so it outlives the
// caller's context, and ends once the connection closes
func (e *Exchange) wsDerivativesScheduleReauthentication(ctx context.Context, conn websocket.Connection, authentication uint64, lifetime time.Duration) {
	ctx = context.WithoutCancel(ctx)
	expiry := time.Now().Add(lifetime)
	var renew func()
	renew = func() {
		if retry := e.wsDerivativesRenewSession(ctx, conn, authentication, expiry); retry > 0 {
			e.derivativesSessions.scheduleRenewal(conn, authentication, retry, renew)
		}
	}
	e.derivativesSessions.scheduleRenewal(conn, authentication, max(lifetime-derivativesReauthenticateMargin, lifetime/2), renew)
}

// wsDerivativesRenewSession authenticates a connection again while it stays connected and no later authentication has
// superseded the one renewed, returning how long to wait before trying again after a failure. A failure moves the
// wrapper's private calls to HTTP; as the gateway drops a session's private subscriptions once it expires, a connection
// whose session could not be renewed in time is closed, for the manager to reconnect, authenticating and subscribing
// afresh. A later authentication that succeeds while a renewal is under way keeps its session
func (e *Exchange) wsDerivativesRenewSession(ctx context.Context, conn websocket.Connection, authentication uint64, expiry time.Time) time.Duration {
	if ctx.Err() != nil || !isConnected(conn) || e.derivativesSessions.superseded(conn, authentication) {
		return 0
	}
	err := e.wsDerivativesSendAuth(ctx, conn)
	if err == nil || !e.derivativesSessions.clearUnlessSuperseded(conn, authentication) {
		return 0
	}
	if time.Until(expiry) > derivativesReauthenticateRetry {
		log.Errorf(log.WebsocketMgr, "%s gateway websocket session could not be renewed, trying again in %s: %v", e.Name, derivativesReauthenticateRetry, err)
		return derivativesReauthenticateRetry
	}
	log.Errorf(log.WebsocketMgr, "%s gateway websocket session could not be renewed before it expires, reconnecting: %v", e.Name, err)
	e.wsCloseForReconnection(ctx, conn, derivativesNonMatchingEPL, "session expired")
	return 0
}

// derivativesWebsocketSessions holds the API key each of the gateway's private connections is authenticated with, as
// the manager opens another once a connection holds its most subscriptions, and a session acts for its key's account
// alone, and each connection's pending session renewal and latest authentication, which numbers renewals, so that one
// a later authentication superseded is left alone
type derivativesWebsocketSessions struct {
	m               sync.Mutex
	keys            map[websocket.Connection]string
	renewals        map[websocket.Connection]*time.Timer
	authentications map[websocket.Connection]uint64
	lastAuth        uint64
}

// authenticated records the API key a connection authenticated with, returning the authentication's number
func (s *derivativesWebsocketSessions) authenticated(conn websocket.Connection, key string) uint64 {
	s.m.Lock()
	defer s.m.Unlock()
	if s.keys == nil {
		s.keys = make(map[websocket.Connection]string)
	}
	if s.authentications == nil {
		s.authentications = make(map[websocket.Connection]uint64)
	}
	s.pruneClosed(conn)
	s.keys[conn] = key
	s.lastAuth++
	s.authentications[conn] = s.lastAuth
	return s.lastAuth
}

// superseded reports whether a connection authenticated again since an authentication
func (s *derivativesWebsocketSessions) superseded(conn websocket.Connection, authentication uint64) bool {
	s.m.Lock()
	defer s.m.Unlock()
	return s.authentications[conn] != authentication
}

// clearUnlessSuperseded forgets a connection's API key after its renewal failed, unless a later authentication
// succeeded meanwhile, and reports whether it did
func (s *derivativesWebsocketSessions) clearUnlessSuperseded(conn websocket.Connection, authentication uint64) bool {
	s.m.Lock()
	defer s.m.Unlock()
	if s.authentications[conn] != authentication {
		return false
	}
	s.keys[conn] = ""
	return true
}

// set records the API key a connection is authenticated with, none when it is not, and forgets connections that have
// closed, which the manager replaces with new ones
func (s *derivativesWebsocketSessions) set(conn websocket.Connection, key string) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.keys == nil {
		s.keys = make(map[websocket.Connection]string)
	}
	s.pruneClosed(conn)
	s.keys[conn] = key
}

// scheduleRenewal runs a connection's session renewal after a delay, in place of any renewal it has pending, unless a
// later authentication superseded the one it renews
func (s *derivativesWebsocketSessions) scheduleRenewal(conn websocket.Connection, authentication uint64, after time.Duration, renew func()) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.authentications[conn] != authentication {
		return
	}
	if s.renewals == nil {
		s.renewals = make(map[websocket.Connection]*time.Timer)
	}
	s.pruneClosed(conn)
	if pending, ok := s.renewals[conn]; ok {
		pending.Stop()
	}
	s.renewals[conn] = time.AfterFunc(after, renew)
}

// pruneClosed forgets the closed connections other than conn, stopping their pending renewals; the caller holds s.m
func (s *derivativesWebsocketSessions) pruneClosed(conn websocket.Connection) {
	for c := range s.keys {
		if c != conn && !isConnected(c) {
			delete(s.keys, c)
		}
	}
	for c, pending := range s.renewals {
		if c != conn && !isConnected(c) {
			pending.Stop()
			delete(s.renewals, c)
		}
	}
	for c := range s.authentications {
		if c != conn && !isConnected(c) {
			delete(s.authentications, c)
		}
	}
}

// authenticatedKey returns the API key a connection is authenticated with, none when it is not
func (s *derivativesWebsocketSessions) authenticatedKey(conn websocket.Connection) string {
	s.m.Lock()
	defer s.m.Unlock()
	return s.keys[conn]
}

// isConnected reports whether a connection is still connected; the manager's connections report it
func isConnected(conn websocket.Connection) bool {
	c, ok := conn.(interface{ IsConnected() bool })
	return ok && c.IsConnected()
}

// wsDerivativesSubscribe subscribes a gateway connection to channels in one request, public/subscribe on the public
// streams and private/subscribe on a private connection, which refuses public/subscribe, and records each channel the
// gateway acknowledged; it leaves out an instrument it does not list
func (e *Exchange) wsDerivativesSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.wsDerivativesManage(ctx, conn, true, subs)
}

// wsDerivativesUnsubscribe unsubscribes a gateway connection from channels in one request, and removes each channel the
// gateway acknowledged
func (e *Exchange) wsDerivativesUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.wsDerivativesManage(ctx, conn, false, subs)
}

// wsDerivativesManage subscribes to or unsubscribes from channels on a gateway connection, then records each the
// gateway acknowledged and reports the rest
func (e *Exchange) wsDerivativesManage(ctx context.Context, conn websocket.Connection, subscribe bool, subs subscription.List) error {
	if len(subs) == 0 {
		return nil
	}
	private := slices.ContainsFunc(subs, func(s *subscription.Subscription) bool {
		ch, ok := derivativesChannelOf(s)
		return ok && ch.private
	})
	method, epl := "public/unsubscribe", derivativesNonMatchingEPL
	switch {
	case subscribe && private:
		method, epl = "private/subscribe", derivativesSubscribeEPL
	case subscribe:
		method, epl = "public/subscribe", derivativesSubscribeEPL
	case private:
		method = "private/unsubscribe"
	}
	var acknowledged []string
	if err := e.WebsocketAPI().sendRequestOnConnection(ctx, conn, epl, method, &DerivativesSubscriptionWireRequest{Channels: subs.QualifiedChannels()}, &acknowledged); err != nil {
		return fmt.Errorf("%s %s: %w", method, subs, err)
	}
	var errs error
	for _, s := range subs {
		if !slices.Contains(acknowledged, s.QualifiedChannel) {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", errDerivativesSubscriptionUnconfirmed, s.QualifiedChannel))
			continue
		}
		if subscribe {
			errs = common.AppendError(errs, e.Websocket.AddSuccessfulSubscriptions(conn, s))
		} else {
			errs = common.AppendError(errs, e.Websocket.RemoveSubscriptions(conn, s))
		}
	}
	return errs
}

// wsHandleDerivatives passes a gateway response to the request awaiting it, answers a heartbeat test request and
// processes a subscription notification
func (e *Exchange) wsHandleDerivatives(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var envelope DerivativesEnvelope
	if err := json.Unmarshal(respRaw, &envelope); err != nil {
		return fmt.Errorf("%s could not parse gateway websocket data: %w: %s", e.Name, err, respRaw)
	}
	if envelope.ID != 0 {
		return conn.RequireMatchWithData(envelope.ID, respRaw)
	}
	switch envelope.Method {
	case derivativesMethodHeartbeat:
		if envelope.Params.Type == derivativesHeartbeatTestRequest {
			go e.wsDerivativesAnswerHeartbeat(ctx, conn)
		}
		return nil
	case derivativesMethodSubscription:
		return e.wsHandleDerivativesNotification(ctx, conn, envelope.Params.Channel, respRaw)
	default:
		return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{
			Message: e.Name + websocket.UnhandledMessage + string(respRaw),
		})
	}
}

// wsDerivativesAnswerHeartbeat answers a heartbeat test request with a public/test request, without which the gateway
// closes the connection
func (e *Exchange) wsDerivativesAnswerHeartbeat(ctx context.Context, conn websocket.Connection) {
	var resp json.RawMessage
	if err := e.WebsocketAPI().sendRequestOnConnection(ctx, conn, derivativesNonMatchingEPL, "public/test", nil, &resp); err != nil {
		log.Errorf(log.WebsocketMgr, "%s gateway websocket heartbeat test request could not be answered: %v", e.Name, err)
	}
}

// derivativesNotificationChannel returns the gateway channel a notification's qualified channel belongs to, or an
// empty string for an unknown channel
func derivativesNotificationChannel(channel string) string {
	for _, name := range derivativesChannelsByLength {
		if channel == name || strings.HasPrefix(channel, name+".") {
			return name
		}
	}
	return ""
}
