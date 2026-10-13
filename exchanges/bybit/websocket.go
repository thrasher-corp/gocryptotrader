package bybit

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
	"unicode"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	// websocketPingInterval follows Bybit's recommendation; a connection with no pings and no data is closed after 10
	// minutes
	websocketPingInterval = 20 * time.Second

	spotPublic            = "wss://stream.bybit.com/v5/public/spot"
	linearPublic          = "wss://stream.bybit.com/v5/public/linear"  // USDT and USDC perpetuals and futures
	inversePublic         = "wss://stream.bybit.com/v5/public/inverse" // Inverse perpetuals and futures
	optionPublic          = "wss://stream.bybit.com/v5/public/option"  // USDT and USDC options
	websocketPrivate      = "wss://stream.bybit.com/v5/private"
	websocketTrade        = "wss://stream.bybit.com/v5/trade"
	websocketSystemStatus = "wss://stream.bybit.com/v5/public/misc/status"

	// subscriptionBatchSize is the most topics spot accepts in one request; the other streams accept more, but share the
	// limit so a rejected request costs at most ten topics a resend
	subscriptionBatchSize = 10

	subscribeOperation   = "subscribe"
	unsubscribeOperation = "unsubscribe"
	authOperation        = "auth"

	// commandResponseType marks option and system status replies, which echo no request ID and so are matched on it
	commandResponseType = "COMMAND_RESP"

	snapshotType = "snapshot"
)

// Public, private and system topic names
const (
	chanOrderbook      = "orderbook"
	chanRPIOrderbook   = "orderbook.rpi"
	chanFullOrderbook  = "orderbook.full"
	chanPublicTrade    = "publicTrade"
	chanPublicTicker   = "tickers"
	chanKline          = "kline"
	chanAllLiquidation = "allLiquidation"
	chanInsurancePool  = "insurance"
	chanPriceLimit     = "priceLimit"
	chanADLAlert       = "adlAlert"

	chanPositions     = "position"
	chanExecution     = "execution"
	chanFastExecution = "execution.fast"
	chanOrder         = "order"
	chanWallet        = "wallet"
	chanGreeks        = "greeks"
	chanDCP           = "dcp"
	chanStrategy      = "strategy"

	chanSystemStatus = "system.status"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 50},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneHour},
}

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    chanPublicTicker,
	subscription.OrderbookChannel: chanOrderbook,
	subscription.AllTradesChannel: chanPublicTrade,
	subscription.MyOrdersChannel:  chanOrder,
	subscription.MyWalletChannel:  chanWallet,
	subscription.MyTradesChannel:  chanExecution,
	subscription.CandlesChannel:   chanKline,
}

// Errors from subscribing, authenticating and handling pushes on the websocket streams
var (
	errUnhandledStreamData        = errors.New("unhandled stream data")
	errUnsupportedCategory        = errors.New("unsupported category")
	errWebsocketAuthFailed        = errors.New("websocket authentication failed")
	errContractGroupNotSupported  = errors.New("topic has no contract group for asset")
	errOptionTradesNotOneBaseCoin = errors.New("option trade subscriptions must share one base coin")
)

// commandResponseMu lets one option or system status request at a time wait for its reply, since the replies carry no
// request ID to tell them apart
var commandResponseMu sync.Mutex

// WsConnect connects to a websocket stream and starts its heartbeat
func (e *Exchange) WsConnect(ctx context.Context, conn websocket.Connection) error {
	if err := conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil); err != nil {
		return err
	}
	conn.SetupPingHandler(wsSubscriptionEPL, websocket.PingHandler{
		MessageType: gws.TextMessage,
		Message:     []byte(`{"op":"ping"}`),
		Delay:       websocketPingInterval,
	})
	return nil
}

// WebsocketAuthenticatePrivateConnection authenticates the private stream connection (Connect: Authentication)
func (e *Exchange) WebsocketAuthenticatePrivateConnection(ctx context.Context, conn websocket.Connection) error {
	req, err := e.GetAuthenticationPayload(ctx, e.MessageID())
	if err != nil {
		return err
	}
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
	if err != nil {
		return err
	}
	var resp SubscriptionResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%w: %s", errWebsocketAuthFailed, resp.ReturnMessage)
	}
	return nil
}

// GetAuthenticationPayload returns the request that authenticates a private stream connection (Connect: Authentication)
func (e *Exchange) GetAuthenticationPayload(ctx context.Context, requestID string) (*AuthenticationRequest, error) {
	args, err := e.websocketAuthenticationArguments(ctx)
	if err != nil {
		return nil, err
	}
	return &AuthenticationRequest{RequestID: requestID, Operation: authOperation, Arguments: args}, nil
}

// websocketAuthenticationArguments returns the API key, expiry and signature that authenticate a private stream or
// trade connection
func (e *Exchange) websocketAuthenticationArguments(ctx context.Context) ([]any, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(time.Hour * 6).UnixMilli()
	return []any{creds.Key, expires, getSign("GET/realtime"+strconv.FormatInt(expires, 10), creds.Secret)}, nil
}

// sendSubscriptions sends subscribe or unsubscribe requests for subs in batches and records the subscriptions Bybit
// accepts. Bybit rejects a whole request when one of its topics is invalid and names that topic, so a rejected batch is
// resent without the topics it names and the batches after it are still sent; every rejected topic is reported in the
// returned error
func (e *Exchange) sendSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	var errs error
	for _, batch := range common.Batch(subs, subscriptionBatchSize) {
		errs = common.AppendError(errs, e.sendSubscriptionBatch(ctx, conn, operation, batch))
	}
	return errs
}

func (e *Exchange) sendSubscriptionBatch(ctx context.Context, conn websocket.Connection, operation string, batch subscription.List) error {
	var errs error
	for len(batch) != 0 {
		req := &SubscriptionRequest{RequestID: e.MessageID(), Operation: operation, Arguments: uniqueTopics(batch)}
		respRaw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, req.RequestID, req)
		if err != nil {
			return common.AppendError(errs, fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err))
		}
		var resp SubscriptionResponse
		if err := json.Unmarshal(respRaw, &resp); err != nil {
			return common.AppendError(errs, fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err))
		}
		if resp.Success {
			return common.AppendError(errs, e.recordSubscriptions(conn, operation, batch))
		}
		var rejected subscription.List
		rejected, batch = splitRejectedSubscriptions(batch, resp.ReturnMessage)
		errs = common.AppendError(errs, fmt.Errorf("%w: %s %s rejected: %s", websocket.ErrSubscriptionFailure, operation, strings.Join(uniqueTopics(rejected), ","), resp.ReturnMessage))
	}
	return errs
}

// splitRejectedSubscriptions returns the subscriptions whose topics a rejection message names, and the rest. A message
// that names none of the topics rejects them all
func splitRejectedSubscriptions(batch subscription.List, message string) (rejected, remaining subscription.List) {
	named := strings.FieldsFunc(message, func(r rune) bool {
		return r == '[' || r == ']' || r == ',' || r == ':' || unicode.IsSpace(r)
	})
	for _, s := range batch {
		if slices.Contains(named, s.QualifiedChannel) {
			rejected = append(rejected, s)
		} else {
			remaining = append(remaining, s)
		}
	}
	if len(rejected) == 0 {
		return batch, nil
	}
	return rejected, remaining
}

// sendCommandSubscriptions sends subscribe or unsubscribe requests for subs on the option and system status streams,
// whose replies list the topics applied and rejected
func (e *Exchange) sendCommandSubscriptions(ctx context.Context, conn websocket.Connection, operation string, subs subscription.List) error {
	var errs error
	for _, batch := range common.Batch(subs, subscriptionBatchSize) {
		errs = common.AppendError(errs, e.sendCommandSubscriptionBatch(ctx, conn, operation, batch))
	}
	return errs
}

func (e *Exchange) sendCommandSubscriptionBatch(ctx context.Context, conn websocket.Connection, operation string, batch subscription.List) error {
	req := &SubscriptionRequest{RequestID: e.MessageID(), Operation: operation, Arguments: uniqueTopics(batch)}
	commandResponseMu.Lock()
	respRaw, err := conn.SendMessageReturnResponse(ctx, wsSubscriptionEPL, commandResponseType, req)
	commandResponseMu.Unlock()
	if err != nil {
		return fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err)
	}
	var resp SubscriptionResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return fmt.Errorf("%w: %s %s: %w", websocket.ErrSubscriptionFailure, operation, strings.Join(req.Arguments, ","), err)
	}
	var accepted, rejected subscription.List
	for _, s := range batch {
		if !resp.Success || slices.Contains(resp.Data.FailTopics, s.QualifiedChannel) {
			rejected = append(rejected, s)
		} else {
			accepted = append(accepted, s)
		}
	}
	var errs error
	if len(accepted) != 0 {
		errs = e.recordSubscriptions(conn, operation, accepted)
	}
	if len(rejected) != 0 {
		errs = common.AppendError(errs, fmt.Errorf("%w: %s %s rejected", websocket.ErrSubscriptionFailure, operation, strings.Join(uniqueTopics(rejected), ",")))
	}
	return errs
}

// recordSubscriptions stores subscriptions Bybit subscribed, or removes ones it unsubscribed
func (e *Exchange) recordSubscriptions(conn websocket.Connection, operation string, subs subscription.List) error {
	if operation == unsubscribeOperation {
		return e.Websocket.RemoveSubscriptions(conn, subs...)
	}
	return e.Websocket.AddSuccessfulSubscriptions(conn, subs...)
}

// uniqueTopics returns the topics of subs once each, in order; option trade subscriptions for one base coin share a topic
func uniqueTopics(subs subscription.List) []string {
	topics := make([]string, 0, len(subs))
	for _, s := range subs {
		if !slices.Contains(topics, s.QualifiedChannel) {
			topics = append(topics, s.QualifiedChannel)
		}
	}
	return topics
}

// handleNoTopicWebsocketResponse routes replies to subscribe, unsubscribe, auth and ping requests
func (e *Exchange) handleNoTopicWebsocketResponse(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	var resp SubscriptionResponse
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	switch {
	case resp.Type == commandResponseType:
		if !conn.IncomingWithData(commandResponseType, respRaw) && len(resp.Data.FailTopics) != 0 {
			return fmt.Errorf("%w: %s rejected", websocket.ErrSubscriptionFailure, strings.Join(resp.Data.FailTopics, ","))
		}
		return nil
	case resp.Operation == subscribeOperation, resp.Operation == unsubscribeOperation, resp.Operation == authOperation:
		if resp.RequestID == "" {
			return nil
		}
		return conn.RequireMatchWithData(resp.RequestID, respRaw)
	case resp.Operation == "ping", resp.Operation == "pong":
		return nil
	}
	return e.Websocket.DataHandler.Send(ctx, websocket.UnhandledMessageWarning{Message: string(respRaw)})
}

// generateSubscriptions returns the configured spot subscriptions for the spot connection; each other public connection
// generates its own
func (e *Exchange) generateSubscriptions() (subscription.List, error) {
	var configured subscription.List
	for _, s := range e.Features.Subscriptions {
		if s.Asset == asset.Spot || s.Asset == asset.All {
			configured = append(configured, s)
		}
	}
	expanded, err := configured.ExpandTemplates(e)
	if err != nil {
		return nil, err
	}
	subs := make(subscription.List, 0, len(expanded))
	for _, s := range expanded {
		if s.Asset == asset.Spot {
			subs = append(subs, s)
		}
	}
	return subs, nil
}

// GetSubscriptionTemplate returns the template that expands spot subscriptions into topics
func (e *Exchange) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(template.FuncMap{
		"channelName":      channelName,
		"isSymbolChannel":  isSymbolChannel,
		"intervalToString": intervalToString,
	}).Parse(subTplText)
}

// channelName converts global channel names to exchange specific names
func channelName(s *subscription.Subscription) string {
	if name, ok := subscriptionNames[s.Channel]; ok {
		return name
	}
	return s.Channel
}

// isSymbolChannel returns whether the channel accepts a symbol parameter
func isSymbolChannel(name string) bool {
	switch name {
	case chanPositions, chanExecution, chanOrder, chanWallet, chanInsurancePool, chanADLAlert, chanSystemStatus:
		return false
	}
	return true
}

const subTplText = `
{{ with $name := channelName $.S }}
	{{- range $asset, $pairs := $.AssetPairs }}
		{{- if isSymbolChannel $name }}
			{{- range $p := $pairs }}
				{{- $name -}} .
				{{- if eq $name "orderbook" -}} {{- $.S.Levels -}} . {{- end }}
				{{- if eq $name "kline" -}} {{- intervalToString $.S.Interval -}} . {{- end }}
				{{- $p }}
				{{- $.PairSeparator }}
			{{- end }}
		{{- end }}
		{{- $.AssetSeparator }}
	{{- end }}
{{- end }}
`

// symbol returns the symbol Bybit names a pair by for an asset
func (e *Exchange) symbol(p currency.Pair, a asset.Item) (string, error) {
	pairFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return "", err
	}
	// USDC futures are named with a dash, unlike USDC perpetuals, which the request format serves
	if a == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
		pairFmt.Delimiter = currency.DashDelimiter
	}
	return pairFmt.Format(p), nil
}

// matchPairAssetFromResponse returns the currency pair and asset type based on the category and symbol. Used with a dedicated
// auth connection where multiple asset type changes are piped through a single connection.
func (e *Exchange) matchPairAssetFromResponse(category, symbol string) (currency.Pair, asset.Item, error) {
	assets := make([]asset.Item, 0, 2)
	switch category {
	case cSpot:
		assets = append(assets, asset.Spot)
	case cInverse:
		assets = append(assets, asset.CoinMarginedFutures)
	case cLinear:
		assets = append(assets, asset.USDTMarginedFutures, asset.USDCMarginedFutures)
	case cOption:
		assets = append(assets, asset.Options)
	default:
		return currency.EMPTYPAIR, 0, fmt.Errorf("incoming symbol %q %w: %q", symbol, errUnsupportedCategory, category)
	}
	for _, a := range assets {
		cp, err := e.MatchSymbolWithAvailablePairs(symbol, a, hasPotentialDelimiter(a))
		if err != nil {
			if !errors.Is(err, currency.ErrPairNotFound) {
				return currency.EMPTYPAIR, 0, fmt.Errorf("%w for %s symbol %q", err, category, symbol)
			}
			continue
		}
		return cp, a, nil
	}
	return currency.EMPTYPAIR, 0, currency.ErrPairNotFound
}

// hasPotentialDelimiter returns if the asset has a potential delimiter on the pairs being returned.
func hasPotentialDelimiter(a asset.Item) bool {
	return a == asset.Options || a == asset.USDCMarginedFutures
}

// stringToInterval returns a kline.Interval instance from string.
func stringToInterval(s string) (kline.Interval, error) {
	interval, okay := stringToIntervalMap[s]
	if okay {
		return interval, nil
	}
	return 0, kline.ErrInvalidInterval
}
