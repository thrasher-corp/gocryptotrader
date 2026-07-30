package htx

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	wsFuturesURL              = "wss://api.hbdm.com/ws"
	wsFuturesPrivateURL       = "wss://api.hbdm.com/notification"
	wsCoinMarginedURL         = "wss://api.hbdm.com/swap-ws"
	wsCoinMarginedPrivateURL  = "wss://api.hbdm.com/swap-notification"
	wsUSDTMarginedURL         = "wss://api.hbdm.com/linear-swap-ws"
	wsUSDTMarginedPrivateURL  = "wss://api.hbdm.com/linear-swap-notification"
	wsPositionsChannel        = "positions"
	wsTriggerOrdersChannel    = "triggerOrders"
	wsCrossOrdersChannel      = "crossOrders"
	wsCrossTradesChannel      = "crossTrades"
	wsCrossAccountsChannel    = "crossAccounts"
	wsCrossPositionsChannel   = "crossPositions"
	wsCrossTriggersChannel    = "crossTriggerOrders"
	wsFuturesSignatureVersion = "2"
)

var defaultFuturesSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: true, Asset: asset.Futures, Channel: wsTriggerOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: wsFundingRateChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: wsTriggerOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsFundingRateChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsTriggerOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsCrossOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsCrossTradesChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsCrossAccountsChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsCrossPositionsChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: wsCrossTriggersChannel, Authenticated: true},
}

// wsConnect establishes an HTX websocket connection for the asset selected by its connection setup.
func (e *Exchange) wsConnect(ctx context.Context, conn websocket.Connection) error {
	if !e.Websocket.IsEnabled() || !e.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	return conn.Dial(ctx, &gws.Dialer{}, http.Header{}, nil)
}

// generateSubscriptionsForAsset isolates configured subscriptions for one websocket connection.
func (e *Exchange) generateSubscriptionsForAsset(a asset.Item, private bool) (subscription.List, error) {
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	for _, sub := range e.Features.Subscriptions {
		if sub.Asset == a && sub.Authenticated == private {
			cloned := sub.Clone()
			if a == asset.Futures {
				pairs, err := e.GetEnabledPairs(a)
				if err != nil {
					return nil, err
				}
				e.futureContractCodesMutex.RLock()
				for _, pair := range pairs {
					quote := pair.Quote.String()
					for expiryCode, expiryDate := range e.futureContractCodes {
						if quote == expiryDate.String() {
							quote = expiryCode
							break
						}
					}
					cloned.Pairs = append(cloned.Pairs, currency.NewPairWithDelimiter(pair.Base.String(), quote, "_"))
				}
				e.futureContractCodesMutex.RUnlock()
			}
			subs = append(subs, cloned)
		}
	}
	return subs.ExpandTemplates(e)
}

// getWebsocketConnection resolves legacy subscription calls onto the asset-specific connection.
func (e *Exchange) getWebsocketConnection(sub *subscription.Subscription) (websocket.Connection, error) {
	switch sub.Asset {
	case asset.Spot:
		if sub.Authenticated {
			if e.Websocket.AuthConn != nil {
				return e.Websocket.AuthConn, nil
			}
			return e.Websocket.GetConnection(exchange.WebsocketPrivate)
		}
		if e.Websocket.Conn != nil {
			return e.Websocket.Conn, nil
		}
		return e.Websocket.GetConnection(exchange.WebsocketSpot)
	case asset.Futures:
		if sub.Authenticated {
			return e.Websocket.GetConnection(exchange.WebsocketFuturesPrivate)
		}
		return e.Websocket.GetConnection(exchange.WebsocketFutures)
	case asset.CoinMarginedFutures:
		if sub.Authenticated {
			return e.Websocket.GetConnection(exchange.WebsocketCoinMarginedPrivate)
		}
		return e.Websocket.GetConnection(exchange.WebsocketCoinMargined)
	case asset.USDTMarginedFutures:
		if sub.Authenticated {
			return e.Websocket.GetConnection(exchange.WebsocketUSDTMarginedPrivate)
		}
		return e.Websocket.GetConnection(exchange.WebsocketUSDTMargined)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, sub.Asset)
	}
}

// subscribeConnection subscribes through the connection selected by the websocket manager.
func (e *Exchange) subscribeConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, batch subscription.List) error {
		return e.manageSubs(ctx, conn, wsSubOp, batch)
	}, 1))
}

// unsubscribeConnection unsubscribes through the connection selected by the websocket manager.
func (e *Exchange) unsubscribeConnection(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	subs, errs := subs.ExpandTemplates(e)
	return common.AppendError(errs, e.ParallelChanOp(ctx, subs, func(ctx context.Context, batch subscription.List) error {
		return e.manageSubs(ctx, conn, wsUnsubOp, batch)
	}, 1))
}

// wsAuthenticateConnection authenticates a private connection using the protocol required by its endpoint.
func (e *Exchange) wsAuthenticateConnection(ctx context.Context, conn websocket.Connection) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	endpoint, err := url.Parse(conn.GetURL())
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidEndpoint, err)
	}
	if endpoint.Path == wsPrivatePath {
		err = e.wsLogin(ctx, conn)
	} else {
		err = e.wsFuturesLogin(ctx, conn)
	}
	if err != nil {
		return err
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return nil
}

// wsGenerateFuturesSignature signs authentication requests for derivative notification endpoints.
func (e *Exchange) wsGenerateFuturesSignature(conn websocket.Connection, creds *accounts.Credentials, timestamp string) ([]byte, error) {
	if err := common.NilGuard(conn, creds); err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(conn.GetURL())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidEndpoint, err)
	}
	if endpoint.Host == "" {
		return nil, fmt.Errorf("%w: missing host", errInvalidEndpoint)
	}
	signaturePath := endpoint.EscapedPath()
	if signaturePath == "" {
		signaturePath = "/"
	}
	values := url.Values{}
	values.Set("AccessKeyId", creds.Key)
	values.Set("SignatureMethod", signatureMethod)
	values.Set("SignatureVersion", wsFuturesSignatureVersion)
	values.Set("Timestamp", timestamp)
	payload := http.MethodGet + "\n" + endpoint.Host + "\n" + signaturePath + "\n" + values.Encode()
	return crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(creds.Secret))
}

// wsFuturesLogin authenticates a delivery, coin-margined, or USDT-margined notification connection.
func (e *Exchange) wsFuturesLogin(ctx context.Context, conn websocket.Connection) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	timestamp := time.Now().UTC().Format(wsDateTimeFormatting)
	signature, err := e.wsGenerateFuturesSignature(conn, creds, timestamp)
	if err != nil {
		return err
	}
	req := wsFuturesAuthRequest{
		Operation:        wsAuthChannel,
		AuthType:         "api",
		AccessKeyID:      creds.Key,
		SignatureMethod:  signatureMethod,
		SignatureVersion: wsFuturesSignatureVersion,
		Timestamp:        timestamp,
		Signature:        base64.StdEncoding.EncodeToString(signature),
	}
	resp, err := conn.SendMessageReturnResponse(ctx, request.Unset, wsAuthChannel, req)
	if err != nil {
		return err
	}
	return getErrResp(resp)
}

// wsHandleFuturesPing responds to notification endpoint heartbeats without changing timestamp representation.
func (e *Exchange) wsHandleFuturesPing(ctx context.Context, conn websocket.Connection, raw []byte) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	var ping wsFuturesPong
	if err := json.Unmarshal(raw, &ping); err != nil {
		return err
	}
	if len(ping.Timestamp) == 0 {
		return fmt.Errorf("%w: missing futures websocket timestamp", common.ErrParsingWSField)
	}
	ping.Operation = "pong"
	if err := conn.SendJSONMessage(ctx, request.Unset, ping); err != nil {
		return fmt.Errorf("error sending futures pong response: %w", err)
	}
	return nil
}

// wsHandleFuturesOperationResponse correlates derivative subscription acknowledgements with their request.
func (e *Exchange) wsHandleFuturesOperationResponse(conn websocket.Connection, operation string, raw []byte) error {
	if err := common.NilGuard(conn); err != nil {
		return err
	}
	topic, err := jsonparser.GetString(raw, "topic")
	if err != nil {
		return fmt.Errorf("%w 'topic': %w", common.ErrParsingWSField, err)
	}
	return conn.RequireMatchWithData(operation+":"+topic, raw)
}

// getFuturesPrivateSubscription resolves concrete notification topics against wildcard subscriptions.
func (e *Exchange) getFuturesPrivateSubscription(conn websocket.Connection, topic string) *subscription.Subscription {
	if conn != nil {
		if sub := conn.Subscriptions().Get(topic); sub != nil {
			return sub
		}
	} else if sub := e.Websocket.GetSubscription(topic); sub != nil {
		return sub
	}
	index := strings.IndexByte(topic, '.')
	wildcard := topic + ".*"
	if index != -1 {
		wildcard = topic[:index+1] + "*"
	}
	if conn != nil {
		return conn.Subscriptions().Get(wildcard)
	}
	return e.Websocket.GetSubscription(wildcard)
}

// wsHandleFuturesPrivateMessage dispatches private derivative notifications to their asset-specific schema.
func (e *Exchange) wsHandleFuturesPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	switch sub.Asset {
	case asset.Futures:
		return e.wsHandleDeliveryFuturesPrivateMessage(ctx, sub, raw)
	case asset.CoinMarginedFutures:
		return e.wsHandleCoinMarginedPrivateMessage(ctx, sub, raw)
	case asset.USDTMarginedFutures:
		return e.wsHandleUSDTMarginedPrivateMessage(ctx, sub, raw)
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, sub.Asset)
	}
}

// wsHandleDeliveryFuturesPrivateMessage decodes authenticated delivery-futures notifications.
func (e *Exchange) wsHandleDeliveryFuturesPrivateMessage(ctx context.Context, sub *subscription.Subscription, raw []byte) error {
	if err := common.NilGuard(sub); err != nil {
		return err
	}
	var response any
	switch sub.Channel {
	case subscription.MyOrdersChannel:
		response = new(FWsSubOrderData)
	case subscription.MyTradesChannel:
		response = new(FWsSubMatchOrderData)
	case subscription.MyAccountChannel:
		response = new(FWsSubEquityUpdates)
	case wsPositionsChannel:
		response = new(FWsSubPositionUpdates)
	case wsTriggerOrdersChannel:
		response = new(FWsSubTriggerOrderUpdates)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, response)
}

// wsHandleFundingRateMsg forwards derivative funding-rate updates to the websocket data handler.
func (e *Exchange) wsHandleFundingRateMsg(ctx context.Context, s *subscription.Subscription, raw []byte) error {
	if s.Asset != asset.CoinMarginedFutures && s.Asset != asset.USDTMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, s.Asset)
	}
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var response *WsFundingRate
	if err := json.Unmarshal(raw, &response); err != nil {
		return err
	}
	response.Asset = s.Asset
	response.Pair = s.Pairs[0]
	return e.Websocket.DataHandler.Send(ctx, response)
}
