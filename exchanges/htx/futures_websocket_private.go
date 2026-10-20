package htx

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	wsPositionsChannel        = "positions"
	wsTriggerOrdersChannel    = "triggerOrders"
	wsTradeUpdatesChannel     = "tradeUpdates"
	wsExecutionDetailsChannel = "executionDetails"
	wsFuturesSignatureVersion = "2"
	orderOffsetClose          = "close"
)

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
		response := new(FuturesWebsocketSubOrder)
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
		response := new(FuturesWebsocketSubMatchOrder)
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
			leverage:       float64(response.LeverageRate),
			createdAt:      response.CreatedAt,
			reduceOnly:     response.Offset == orderOffsetClose,
		})
		if err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, &detail)
	case subscription.MyAccountChannel:
		response := new(FuturesWebsocketSubEquityUpdates)
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
		response := new(FuturesWebsocketSubPositionUpdates)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	case wsTriggerOrdersChannel:
		response := new(FuturesWebsocketSubTriggerOrderUpdates)
		if err := json.Unmarshal(raw, response); err != nil {
			return err
		}
		return e.Websocket.DataHandler.Send(ctx, response)
	default:
		return fmt.Errorf("%w: %s", common.ErrNotYetImplemented, sub.Channel)
	}
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
		orderID = strconv.FormatUint(data.orderID, 10)
	}
	var clientOrderID string
	if data.clientOrderID != 0 {
		clientOrderID = strconv.FormatUint(data.clientOrderID, 10)
	}
	createdAt, cancelledAt := data.createdAt.Time(), data.cancelledAt.Time()
	trades := make([]order.TradeHistory, 0, len(data.trades))
	for _, fill := range data.trades {
		id := fill.ID
		if id == "" {
			id = strconv.FormatUint(fill.TradeID, 10)
		}
		feeAsset := fill.FeeAsset
		if feeAsset.IsEmpty() {
			feeAsset = data.feeAsset
		}
		trades = append(trades, order.TradeHistory{
			TID:       id,
			Exchange:  e.Name,
			Price:     fill.TradePrice,
			Amount:    fill.TradeVolume,
			Fee:       -fill.TradeFee,
			FeeAsset:  feeAsset.String(),
			Total:     fill.TradeTurnover,
			Timestamp: fill.CreatedAt.Time(),
			IsMaker:   strings.EqualFold(fill.Role, "maker"),
			Type:      orderVars.OrderType,
			Side:      orderVars.Side,
		})
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
		CostAsset:            currency.USD,
		Fee:                  -data.fee,
		FeeAsset:             data.feeAsset,
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
