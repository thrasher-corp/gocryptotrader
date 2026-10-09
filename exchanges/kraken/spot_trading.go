package kraken

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// addOrderTypes are the order types Add Order and Add Order Batch take
	addOrderTypes = []string{"market", "limit", "iceberg", "stop-loss", "take-profit", "stop-loss-limit", "take-profit-limit", "trailing-stop", "trailing-stop-limit", "settle-position"}
	// conditionalCloseOrderTypes are the order types a conditional close order takes
	conditionalCloseOrderTypes = []string{"limit", "iceberg", "stop-loss", "take-profit", "stop-loss-limit", "take-profit-limit", "trailing-stop", "trailing-stop-limit"}
)

// AddOrder calls Add Order. Get Tradable Asset Pairs gives each pair's order and cost minimums and its price and volume
// precisions
func (e *Exchange) AddOrder(ctx context.Context, req *AddOrderRequest) (*AddOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	body, closeOrder, err := req.Order.params()
	if err != nil {
		return nil, err
	}
	// Add Order names the conditional close order's parameters with form style keys, where Add Order Batch nests them
	for k, v := range closeOrder {
		body["close["+k+"]"] = v
	}
	symbol, err := e.FormatSymbol(req.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	body["pair"] = symbol
	if req.AssetClass != "" {
		body["asset_class"] = req.AssetClass
	}
	if !req.Deadline.IsZero() {
		body["deadline"] = req.Deadline.UTC().Format(time.RFC3339)
	}
	if req.Validate {
		body["validate"] = true
	}
	if req.BrokerIIBAN != "" {
		body["broker"] = req.BrokerIIBAN
	}
	var resp *AddOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/AddOrder", nil, body, &resp)
}

// AmendOrder calls Amend Order, changing an open order in place: it keeps its identifiers, its fills and, where it can,
// its queue priority. Amending the quantity below the filled quantity cancels the remainder, and an order with a
// conditional close order cannot be amended
func (e *Exchange) AmendOrder(ctx context.Context, req *AmendOrderRequest) (*AmendOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.TransactionID == "" && req.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if req.TransactionID != "" && req.ClientOrderID != "" {
		return nil, fmt.Errorf("%w: transaction ID and client order ID", errMultipleOrderIdentifiers)
	}
	if req.OrderQuantity < 0 || req.DisplayQuantity < 0 {
		return nil, fmt.Errorf("%w: quantities cannot be negative", order.ErrAmountIsInvalid)
	}
	if err := checkOrderPrice(req.LimitPrice, "+-"); err != nil {
		return nil, err
	}
	if err := checkOrderPrice(req.TriggerPrice, "+-"); err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if req.TransactionID != "" {
		body["txid"] = req.TransactionID
	} else {
		body["cl_ord_id"] = req.ClientOrderID
	}
	if req.OrderQuantity != 0 {
		body["order_qty"] = strconv.FormatFloat(req.OrderQuantity, 'f', -1, 64)
	}
	if req.DisplayQuantity != 0 {
		body["display_qty"] = strconv.FormatFloat(req.DisplayQuantity, 'f', -1, 64)
	}
	if req.LimitPrice != (OrderPrice{}) {
		body["limit_price"] = req.LimitPrice.String()
	}
	if req.TriggerPrice != (OrderPrice{}) {
		body["trigger_price"] = req.TriggerPrice.String()
	}
	if !req.Pair.IsEmpty() {
		symbol, err := e.FormatSymbol(req.Pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		body["pair"] = symbol
	}
	if req.PostOnly {
		body["post_only"] = true
	}
	if !req.Deadline.IsZero() {
		body["deadline"] = req.Deadline.UTC().Format(time.RFC3339)
	}
	var resp *AmendOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/AmendOrder", nil, body, &resp)
}

// EditOrder calls Edit Order, which cancels an open order and places its replacement under a new transaction ID,
// losing its queue priority and leaving its fills with the original order. It cannot edit triggered stop-loss and
// take-profit orders, orders with a conditional close order or orders already filled beyond the new volume, which
// AmendOrder avoids by changing an order in place
func (e *Exchange) EditOrder(ctx context.Context, req *EditOrderRequest) (*EditOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.TransactionID == "" && req.OrderUserReference == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if req.TransactionID != "" && req.OrderUserReference != 0 {
		return nil, fmt.Errorf("%w: transaction ID and user reference", errMultipleOrderIdentifiers)
	}
	if req.Volume < 0 || req.DisplayVolume < 0 {
		return nil, fmt.Errorf("%w: volumes cannot be negative", order.ErrAmountIsInvalid)
	}
	if err := checkOrderPrice(req.Price, "+-#"); err != nil {
		return nil, err
	}
	if err := checkOrderPrice(req.SecondaryPrice, "+-#"); err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(req.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"pair": symbol}
	if req.TransactionID != "" {
		body["txid"] = req.TransactionID
	} else {
		body["txid"] = req.OrderUserReference
	}
	if req.NewUserReference != 0 {
		body["userref"] = req.NewUserReference
	}
	if req.AssetClass != "" {
		body["asset_class"] = req.AssetClass
	}
	if req.Volume != 0 {
		body["volume"] = strconv.FormatFloat(req.Volume, 'f', -1, 64)
	}
	if req.DisplayVolume != 0 {
		body["displayvol"] = strconv.FormatFloat(req.DisplayVolume, 'f', -1, 64)
	}
	if req.Price != (OrderPrice{}) {
		body["price"] = req.Price.String()
	}
	if req.SecondaryPrice != (OrderPrice{}) {
		body["price2"] = req.SecondaryPrice.String()
	}
	if req.PostOnly {
		body["oflags"] = "post"
	}
	if !req.Deadline.IsZero() {
		body["deadline"] = req.Deadline.UTC().Format(time.RFC3339)
	}
	if req.CancelResponse {
		body["cancel_response"] = true
	}
	if req.Validate {
		body["validate"] = true
	}
	var resp *EditOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/EditOrder", nil, body, &resp)
}

// CancelExistingOrder calls Cancel Order
func (e *Exchange) CancelExistingOrder(ctx context.Context, req *CancelExistingOrderRequest) (*CancelOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	switch {
	case req.TransactionID == "" && req.UserReference == 0 && req.ClientOrderID == "":
		return nil, order.ErrOrderIDNotSet
	case req.TransactionID != "" && (req.UserReference != 0 || req.ClientOrderID != ""), req.UserReference != 0 && req.ClientOrderID != "":
		return nil, fmt.Errorf("%w: cancel takes one of a transaction ID, user reference or client order ID", errMultipleOrderIdentifiers)
	}
	body := make(map[string]any, 1)
	switch {
	case req.TransactionID != "":
		body["txid"] = req.TransactionID
	case req.UserReference != 0:
		body["txid"] = req.UserReference
	default:
		body["cl_ord_id"] = req.ClientOrderID
	}
	var resp *CancelOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/CancelOrder", nil, body, &resp)
}

// CancelAllOpenOrders calls Cancel All Orders
func (e *Exchange) CancelAllOpenOrders(ctx context.Context) (*CancelOrderResponse, error) {
	var resp *CancelOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/CancelAll", nil, nil, &resp)
}

// CancelAllOrdersAfter calls Cancel All Orders After X, a dead man's switch that cancels every order once timeout passes
// without another call. A zero timeout disables it, as does its expiry until the next call. Kraken recommends a minute,
// renewed every 15 to 30 seconds, and disabling the switch before scheduled maintenance, as every order is otherwise
// cancelled when the engine returns
func (e *Exchange) CancelAllOrdersAfter(ctx context.Context, timeout time.Duration) (*CancelAllOrdersAfterResponse, error) {
	// Kraken takes whole seconds, so a timeout under a second would disable the switch rather than arm it
	if timeout < 0 || (timeout > 0 && timeout < time.Second) || timeout >= 24*time.Hour {
		return nil, fmt.Errorf("%w: %s is neither 0 nor from 1 second to under 24 hours", errInvalidCancelTimeout, timeout)
	}
	body := map[string]any{"timeout": uint64(timeout / time.Second)}
	var resp *CancelAllOrdersAfterResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/CancelAllOrdersAfter", nil, body, &resp)
}

// AddOrderBatch calls Add Order Batch, placing 2 to 15 orders on one pair. Kraken rejects the whole batch if an order
// fails validation, but an order failing pre-match checks such as funding is rejected alone, with its error in its
// result
func (e *Exchange) AddOrderBatch(ctx context.Context, req *AddOrderBatchRequest) (*AddOrderBatchResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(req.Orders) < 2 || len(req.Orders) > 15 {
		return nil, fmt.Errorf("%w: %d orders, a batch takes 2 to 15", errInvalidCount, len(req.Orders))
	}
	orders := make([]map[string]any, len(req.Orders))
	for i := range req.Orders {
		params, closeOrder, err := req.Orders[i].params()
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		if closeOrder != nil {
			params["close"] = closeOrder
		}
		orders[i] = params
	}
	symbol, err := e.FormatSymbol(req.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"orders": orders, "pair": symbol}
	if req.AssetClass != "" {
		body["asset_class"] = req.AssetClass
	}
	if !req.Deadline.IsZero() {
		body["deadline"] = req.Deadline.UTC().Format(time.RFC3339)
	}
	if req.Validate {
		body["validate"] = true
	}
	if req.BrokerIIBAN != "" {
		body["broker"] = req.BrokerIIBAN
	}
	var resp *AddOrderBatchResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/AddOrderBatch", nil, body, &resp)
}

// CancelOrderBatch calls Cancel Order Batch
func (e *Exchange) CancelOrderBatch(ctx context.Context, req *CancelOrderBatchRequest) (*CancelOrderBatchResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	count := len(req.TransactionIDs) + len(req.UserReferences) + len(req.ClientOrderIDs)
	if count == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if count > 50 {
		return nil, fmt.Errorf("%w: %d identifiers exceed 50", errInvalidCount, count)
	}
	// The schema wraps each identifier in an object, but its own example, like the clients trading with it, lists them
	// bare
	orders := make([]any, 0, len(req.TransactionIDs)+len(req.UserReferences))
	for _, id := range req.TransactionIDs {
		if id == "" {
			return nil, fmt.Errorf("%w: empty transaction ID", order.ErrOrderIDNotSet)
		}
		orders = append(orders, id)
	}
	for _, ref := range req.UserReferences {
		if ref == 0 {
			return nil, fmt.Errorf("%w: user reference 0", order.ErrOrderIDNotSet)
		}
		orders = append(orders, ref)
	}
	if slices.Contains(req.ClientOrderIDs, "") {
		return nil, fmt.Errorf("%w: empty client order ID", order.ErrOrderIDNotSet)
	}
	body := make(map[string]any, 2)
	if len(orders) != 0 {
		body["orders"] = orders
	}
	if len(req.ClientOrderIDs) != 0 {
		body["cl_ord_ids"] = req.ClientOrderIDs
	}
	var resp *CancelOrderBatchResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/CancelOrderBatch", nil, body, &resp)
}

// GetWebsocketToken calls Get Websockets Token, returning the token that authenticates websocket connections. It must
// be used within 15 minutes, and does not expire while a connection keeps a private subscription
func (e *Exchange) GetWebsocketToken(ctx context.Context) (*WebsocketTokenResponse, error) {
	var resp *WebsocketTokenResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/GetWebSocketsToken", nil, nil, &resp)
}

// params validates an order and returns its parameters, with its conditional close order's apart, as Add Order and Add
// Order Batch send those differently
func (o *OrderParameters) params() (body, closeOrder map[string]any, err error) {
	if o.Side != "buy" && o.Side != "sell" {
		return nil, nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, o.Side)
	}
	if !slices.Contains(addOrderTypes, o.OrderType) {
		return nil, nil, fmt.Errorf("%w: %q", order.ErrTypeIsInvalid, o.OrderType)
	}
	if o.Volume < 0 || o.DisplayVolume < 0 {
		return nil, nil, fmt.Errorf("%w: volumes cannot be negative", order.ErrAmountIsInvalid)
	}
	if o.Volume == 0 && o.Leverage == 0 && o.OrderType != "settle-position" {
		return nil, nil, fmt.Errorf("%w: only margin and settle-position orders take a volume of 0", order.ErrAmountIsInvalid)
	}
	if o.UserReference != 0 && o.ClientOrderID != "" {
		return nil, nil, fmt.Errorf("%w: user reference and client order ID", errMultipleOrderIdentifiers)
	}
	if o.FeeInBase && o.FeeInQuote {
		return nil, nil, fmt.Errorf("%w: fee in base and fee in quote currency", errConflictingOrderFlags)
	}
	if o.TimeInForce != "" && !slices.Contains([]string{timeInForceGTC, timeInForceIOC, timeInForceGTD, timeInForceFOK}, o.TimeInForce) {
		return nil, nil, fmt.Errorf("%w: %q", order.ErrInvalidTimeInForce, o.TimeInForce)
	}
	if o.TimeInForce == timeInForceGTD && o.ExpireTime.IsZero() && o.ExpireDelay == 0 {
		return nil, nil, fmt.Errorf("%w: GTD orders need an expiry", order.ErrInvalidTimeInForce)
	}
	if err := checkOrderPrices(o.OrderType, o.Price, o.SecondaryPrice); err != nil {
		return nil, nil, err
	}
	startTime, err := formatOrderTime(o.StartTime, o.StartDelay, time.Second)
	if err != nil {
		return nil, nil, err
	}
	expireTime, err := formatOrderTime(o.ExpireTime, o.ExpireDelay, 5*time.Second)
	if err != nil {
		return nil, nil, err
	}
	if o.Close != nil {
		if closeOrder, err = o.Close.params(); err != nil {
			return nil, nil, err
		}
	}
	body = map[string]any{
		"ordertype": o.OrderType,
		"type":      o.Side,
		"volume":    strconv.FormatFloat(o.Volume, 'f', -1, 64),
	}
	if o.UserReference != 0 {
		body["userref"] = o.UserReference
	}
	if o.ClientOrderID != "" {
		body["cl_ord_id"] = o.ClientOrderID
	}
	if o.DisplayVolume != 0 {
		body["displayvol"] = strconv.FormatFloat(o.DisplayVolume, 'f', -1, 64)
	}
	if o.Price != (OrderPrice{}) {
		body["price"] = o.Price.String()
	}
	if o.SecondaryPrice != (OrderPrice{}) {
		body["price2"] = o.SecondaryPrice.String()
	}
	if o.TriggerSignal != "" {
		body["trigger"] = o.TriggerSignal
	}
	if o.Leverage != 0 {
		body["leverage"] = strconv.FormatUint(o.Leverage, 10)
	}
	if o.ReduceOnly {
		body["reduce_only"] = true
	}
	if o.SelfTradePrevention != "" {
		body["stptype"] = o.SelfTradePrevention
	}
	var flags []string
	if o.PostOnly {
		flags = append(flags, "post")
	}
	if o.FeeInBase {
		flags = append(flags, "fcib")
	}
	if o.FeeInQuote {
		flags = append(flags, "fciq")
	}
	if o.VolumeInQuote {
		flags = append(flags, "viqc")
	}
	if len(flags) != 0 {
		body["oflags"] = strings.Join(flags, ",")
	}
	if o.TimeInForce != "" {
		body["timeinforce"] = o.TimeInForce
	}
	if startTime != "" {
		body["starttm"] = startTime
	}
	if expireTime != "" {
		body["expiretm"] = expireTime
	}
	return body, closeOrder, nil
}

// params validates a conditional close order and returns its parameters
func (c *ConditionalCloseOrder) params() (map[string]any, error) {
	if !slices.Contains(conditionalCloseOrderTypes, c.OrderType) {
		return nil, fmt.Errorf("%w: conditional close order type %q", order.ErrTypeIsInvalid, c.OrderType)
	}
	if err := checkOrderPrices(c.OrderType, c.Price, c.SecondaryPrice); err != nil {
		return nil, fmt.Errorf("conditional close order: %w", err)
	}
	params := map[string]any{"ordertype": c.OrderType}
	if c.Price != (OrderPrice{}) {
		params["price"] = c.Price.String()
	}
	if c.SecondaryPrice != (OrderPrice{}) {
		params["price2"] = c.SecondaryPrice.String()
	}
	return params, nil
}

// checkOrderPrices validates an order's prices, which a trailing stop order gives relative to the market and its
// limit variant's limit price relative to the trigger price
func checkOrderPrices(orderType string, price, secondaryPrice OrderPrice) error {
	if err := checkOrderPrice(price, "+-#"); err != nil {
		return err
	}
	if err := checkOrderPrice(secondaryPrice, "+-#"); err != nil {
		return err
	}
	switch orderType {
	case "trailing-stop-limit":
		if secondaryPrice.Offset != "+" && secondaryPrice.Offset != "-" {
			return fmt.Errorf("%w: a trailing stop limit order's limit price is a + or - offset from its trigger price", errInvalidOrderPrice)
		}
		fallthrough
	case "trailing-stop":
		if price.Offset == "" {
			return fmt.Errorf("%w: a trailing stop order's trigger price is relative", errInvalidOrderPrice)
		}
	}
	return nil
}

// checkOrderPrice validates a price against the offsets its parameter takes
func checkOrderPrice(p OrderPrice, offsets string) error {
	if p.Value < 0 {
		return fmt.Errorf("%w: %v is negative, where an offset gives a relative price's direction", errInvalidOrderPrice, p.Value)
	}
	if p.Offset != "" && (len(p.Offset) != 1 || !strings.Contains(offsets, p.Offset)) {
		return fmt.Errorf("%w: offset %q is not one of %q", errInvalidOrderPrice, p.Offset, offsets)
	}
	return nil
}

// formatOrderTime returns an order's start or expiry as Kraken takes it: a Unix time, or a delay in whole seconds after
// Kraken receives the order, prefixed with +
func formatOrderTime(t time.Time, delay, minimumDelay time.Duration) (string, error) {
	switch {
	case delay == 0 && t.IsZero():
		return "", nil
	case delay == 0:
		return strconv.FormatInt(t.Unix(), 10), nil
	case !t.IsZero():
		return "", fmt.Errorf("%w: a time and a delay cannot both be set", errInvalidOrderTime)
	case delay < 0 || delay.Truncate(time.Second) < minimumDelay:
		return "", fmt.Errorf("%w: delay %s is under %s", errInvalidOrderTime, delay, minimumDelay)
	}
	return "+" + strconv.FormatInt(int64(delay/time.Second), 10), nil
}
