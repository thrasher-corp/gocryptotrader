package htx

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	wsFuturesURL              = "wss://api.hbdm.com/ws"
	wsFuturesPrivateURL       = "wss://api.hbdm.com/notification"
	wsCoinMarginedURL         = "wss://api.hbdm.com/swap-ws"
	wsCoinMarginedPrivateURL  = "wss://api.hbdm.com/swap-notification"
	wsUSDTMarginedURL         = "wss://api.hbdm.com/linear-swap-ws"
	wsUSDTMarginedPrivateURL  = "wss://api.hbdm.com/ws/v5/notification"
	wsUSDTMarginedTradeURL    = "wss://api.hbdm.com/linear-swap-trade"
	wsPositionsChannel        = "positions"
	wsTriggerOrdersChannel    = "triggerOrders"
	wsTradeUpdatesChannel     = "tradeUpdates"
	wsExecutionDetailsChannel = "executionDetails"
	wsFuturesSignatureVersion = "2"
	orderOffsetClose          = "close"
)

var defaultFuturesSubscriptions = subscription.List{
	{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.Futures, Channel: subscription.AllTradesChannel},
	{Enabled: false, Asset: asset.Futures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.Futures, Channel: wsTriggerOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: wsTriggerOrdersChannel, Authenticated: true},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel},
	{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsTradeUpdatesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsExecutionDetailsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsPositionsChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.MyTradesChannel, Authenticated: true},
	{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: wsTriggerOrdersChannel, Authenticated: true},
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
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		if errors.Is(err, asset.ErrNotEnabled) {
			return subscription.List{}, nil
		}
		return nil, err
	}
	subs := make(subscription.List, 0, len(e.Features.Subscriptions))
	var unsupported error
	for _, sub := range e.Features.Subscriptions {
		if sub.Enabled && sub.Asset == a && sub.Authenticated == private {
			if sub.Channel == wsFundingRateChannel || sub.Channel == wsFundingRateTopic {
				unsupported = common.AppendError(unsupported, fmt.Errorf("%w: %w: %s", websocket.ErrSubscriptionPartial, common.ErrNotYetImplemented, sub.Channel))
				continue
			}
			cloned := sub.Clone()
			if a == asset.Futures && !private {
				pairs := cloned.Pairs
				if len(pairs) == 0 {
					var err error
					pairs, err = e.GetEnabledPairs(a)
					if err != nil {
						return nil, err
					}
				}
				for _, pair := range pairs {
					delimiter := ""
					if slices.Contains(validContractExpiryCodes, pair.Quote.Upper().String()) {
						delimiter = "_"
					}
					topicSub := cloned.Clone()
					topicSub.Pairs = currency.Pairs{currency.NewPairWithDelimiter(pair.Base.String(), pair.Quote.String(), delimiter)}
					expanded, errs := subscription.List{topicSub}.ExpandTemplates(e)
					if errs != nil {
						return nil, errs
					}
					for _, expandedSub := range expanded {
						expandedSub.Pairs = currency.Pairs{pair}
					}
					subs = append(subs, expanded...)
				}
				continue
			}
			subs = append(subs, cloned)
		}
	}
	expanded, err := subs.ExpandTemplates(e)
	return expanded, common.AppendError(unsupported, err)
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
	if err := getErrResp(resp); err != nil {
		return err
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return nil
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
	if ping.Timestamp == "" {
		return fmt.Errorf("%w: missing futures websocket timestamp", common.ErrParsingWSField)
	}
	ping.Operation = "pong"
	if err := conn.SendJSONMessage(ctx, request.Unset, ping); err != nil {
		return fmt.Errorf("error sending futures pong response: %w", err)
	}
	return nil
}

// getFuturesPrivateSubscription resolves concrete notification topics against wildcard subscriptions.
func (e *Exchange) getFuturesPrivateSubscription(conn websocket.Connection, topic string) *subscription.Subscription {
	if conn != nil {
		if sub := e.Websocket.GetSubscription(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: topic}}, connection: conn}); sub != nil {
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
		return e.Websocket.GetSubscription(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: wildcard}}, connection: conn})
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
	switch sub.Channel {
	case subscription.MyOrdersChannel:
		response := new(FWsSubOrderData)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		detail, err := e.formatLegacyFuturesWSOrder(&legacyFuturesWSOrder{
			asset:          sub.Asset,
			contractCode:   response.ContractCode,
			direction:      response.Direction,
			orderPriceType: response.OrderPriceType,
			status:         response.Status,
			orderID:        response.OrderID,
			orderIDString:  response.OrderIDString,
			clientOrderID:  response.ClientOrderID,
			volume:         response.Volume,
			price:          response.Price,
			tradeVolume:    response.TradeVolume,
			tradeTurnover:  response.TradeTurnover,
			averagePrice:   response.TradeAvgPrice,
			trades:         response.Trade,
			fee:            response.Fee,
			feeAsset:       response.FeeAsset,
			leverage:       float64(response.LeverageRate),
			createdAt:      response.CreatedAt,
			cancelledAt:    response.CancelledAt,
			reduceOnly:     response.Offset == orderOffsetClose,
		})
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &detail)
	case subscription.MyTradesChannel:
		response := new(FWsSubMatchOrderData)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		if response.Status == 9 || response.Status == 10 {
			return nil
		}
		detail, err := e.formatLegacyFuturesWSOrder(&legacyFuturesWSOrder{
			asset:          sub.Asset,
			contractCode:   response.ContractCode,
			direction:      response.Direction,
			orderPriceType: response.OrderPriceType,
			orderType:      response.OrderType,
			status:         response.Status,
			orderID:        response.OrderID,
			orderIDString:  response.OrderIDString,
			clientOrderID:  response.ClientOrderID,
			volume:         response.Volume,
			price:          response.Price,
			tradeVolume:    response.TradeVolume,
			leverage:       response.LeverageRate,
			createdAt:      response.CreatedAt,
			reduceOnly:     response.Offset == orderOffsetClose,
		})
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &detail)
	case subscription.MyAccountChannel:
		response := new(FWsSubEquityUpdates)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		changes := make([]accounts.Change, 0, len(response.Data))
		for i := range response.Data {
			changes = append(changes, accounts.Change{
				AssetType: sub.Asset,
				Balance: accounts.Balance{
					Currency:  currency.NewCode(response.Data[i].Symbol),
					Total:     response.Data[i].MarginBalance,
					Hold:      response.Data[i].MarginFrozen,
					Free:      response.Data[i].MarginAvailable,
					UpdatedAt: response.Timestamp.Time(),
				},
			})
		}
		if err := e.storeWebsocketBalances(ctx, sub.Asset, changes); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, changes)
	case wsPositionsChannel:
		response := new(FWsSubPositionUpdates)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case wsTriggerOrdersChannel:
		response := new(FWsSubTriggerOrderUpdates)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
}

// legacyFuturesWSOrder holds the common order fields shared by delivery and coin-margined notifications.
type legacyFuturesWSOrder struct {
	asset          asset.Item
	contractCode   string
	direction      string
	orderPriceType string
	feeAsset       string
	status         int64
	orderID        int64
	clientOrderID  int64
	orderType      int64
	orderIDString  string
	volume         float64
	price          float64
	tradeVolume    float64
	tradeTurnover  float64
	averagePrice   float64
	trades         []LegacyFuturesWSTrade
	fee            float64
	leverage       float64
	createdAt      int64
	cancelledAt    int64
	reduceOnly     bool
}

// formatLegacyFuturesWSOrder converts legacy derivative notifications into the canonical order type.
func (e *Exchange) formatLegacyFuturesWSOrder(data *legacyFuturesWSOrder) (order.Detail, error) {
	priceType := data.orderPriceType
	if priceType == "" {
		switch data.orderType {
		case 1:
			priceType = orderPriceTypeLimit
		case 3:
			priceType = orderPriceTypeOpponent
		case 6:
			priceType = orderPriceTypePostOnly
		default:
			return order.Detail{}, errInvalidOrderPriceType
		}
	}
	orderVars, err := compatibleVars(data.direction, priceType, data.status)
	if err != nil {
		return order.Detail{}, err
	}
	pair, err := currency.NewPairFromString(data.contractCode)
	if err != nil {
		return order.Detail{}, err
	}
	orderID := data.orderIDString
	if orderID == "" {
		orderID = strconv.FormatInt(data.orderID, 10)
	}
	var clientOrderID string
	if data.clientOrderID != 0 {
		clientOrderID = strconv.FormatInt(data.clientOrderID, 10)
	}
	var createdAt, cancelledAt time.Time
	if data.createdAt != 0 {
		createdAt = time.UnixMilli(data.createdAt)
	}
	if data.cancelledAt != 0 {
		cancelledAt = time.UnixMilli(data.cancelledAt)
	}
	trades := make([]order.TradeHistory, 0, len(data.trades))
	for _, fill := range data.trades {
		id := fill.ID
		if id == "" {
			id = strconv.FormatInt(fill.TradeID, 10)
		}
		feeAsset := fill.FeeAsset
		if feeAsset == "" {
			feeAsset = data.feeAsset
		}
		trades = append(trades, order.TradeHistory{TID: id, Exchange: e.Name, Price: fill.TradePrice, Amount: fill.TradeVolume, Fee: -fill.TradeFee, FeeAsset: feeAsset, Total: fill.TradeTurnover, Timestamp: time.UnixMilli(fill.CreatedAt), IsMaker: strings.EqualFold(fill.Role, "maker"), Type: orderVars.OrderType, Side: orderVars.Side})
	}
	return order.Detail{
		AverageExecutedPrice: data.averagePrice,
		Trades:               trades,
		Exchange:             e.Name,
		OrderID:              orderID,
		ClientOrderID:        clientOrderID,
		Pair:                 pair,
		Type:                 orderVars.OrderType,
		Side:                 orderVars.Side,
		TimeInForce:          orderVars.TimeInForce,
		Date:                 createdAt,
		CloseTime:            cancelledAt,
		Status:               orderVars.Status,
		Price:                data.price,
		Amount:               data.volume,
		ExecutedAmount:       data.tradeVolume,
		RemainingAmount:      data.volume - data.tradeVolume,
		Cost:                 data.tradeTurnover,
		Fee:                  -data.fee,
		FeeAsset:             currency.NewCode(data.feeAsset),
		Leverage:             data.leverage,
		ReduceOnly:           data.reduceOnly,
		AssetType:            data.asset,
		MarginType:           margin.Isolated,
	}, nil
}

// storeWebsocketBalances records balances at their local arrival time so REST snapshots and venue clock skew cannot reject subsequent updates.
func (e *Exchange) storeWebsocketBalances(ctx context.Context, a asset.Item, changes []accounts.Change) error {
	for _, change := range changes {
		b := change.Balance
		if _, err := e.Accounts.UpdateBalance(ctx, "", a, b.Currency, func(stored *accounts.Balance) {
			stored.Total, stored.Hold, stored.Free = b.Total, b.Hold, b.Free
		}); err != nil {
			return err
		}
	}
	return nil
}
