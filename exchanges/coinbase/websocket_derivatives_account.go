package coinbase

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errDerivativesUnknownOrderState = errors.New("unknown gateway order state")

// wsHandleDerivativesOrders relays a user.orders or user.isolated.orders notification's orders. A raw channel sends each
// order as it changes, and the others a list of the orders changed over their interval; the gateway documents an
// isolated subaccount's notification as one order, so either form is accepted
func (e *Exchange) wsHandleDerivativesOrders(ctx context.Context, respRaw []byte) error {
	var data json.RawMessage
	if err := decodeDerivativesNotification(respRaw, &data); err != nil {
		return err
	}
	var orders []DerivativesOrder
	if trimmed := bytes.TrimSpace(data); len(trimmed) != 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &orders); err != nil {
			return fmt.Errorf("%w: %w", errDerivativesMalformedNotification, err)
		}
	} else {
		orders = make([]DerivativesOrder, 1)
		if err := json.Unmarshal(trimmed, &orders[0]); err != nil {
			return fmt.Errorf("%w: %w", errDerivativesMalformedNotification, err)
		}
	}
	details := make([]order.Detail, 0, len(orders))
	var errs error
	for i := range orders {
		detail, err := e.derivativesOrderDetail(&orders[i])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		details = append(details, *detail)
	}
	if len(details) != 0 {
		errs = common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, details))
	}
	return errs
}

// wsHandleDerivativesUserTrades passes the account's trades to the fills feed
func (e *Exchange) wsHandleDerivativesUserTrades(respRaw []byte) error {
	var trades []DerivativesUserTrade
	if err := decodeDerivativesNotification(respRaw, &trades); err != nil {
		return err
	}
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	fills := make([]fill.Data, 0, len(trades))
	var errs error
	for i := range trades {
		data, err := e.derivativesTradeFill(&trades[i])
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		fills = append(fills, *data)
	}
	if len(fills) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.Fills.Update(fills...))
}

// derivativesOrderDetail converts a gateway order; its label is its client order ID
func (e *Exchange) derivativesOrderDetail(o *DerivativesOrder) (*order.Detail, error) {
	a, pair, err := derivativesInstrumentPair(o.InstrumentName)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(o.Direction)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	orderType, err := derivativesOrderType(o.OrderType)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	status, err := derivativesOrderStatus(o.OrderState, o.FilledAmount)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	tif, err := order.StringToTimeInForce(o.TimeInForce)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", o.OrderID, err)
	}
	if o.PostOnly {
		tif |= order.PostOnly
	}
	// An isolated margin subaccount's orders carry isolated; the main account's are cross margined
	marginType := margin.Multi
	if o.Isolated {
		marginType = margin.Isolated
	}
	return &order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID,
		ClientOrderID:        o.Label,
		Pair:                 pair,
		AssetType:            a,
		Side:                 side,
		Type:                 orderType,
		Status:               status,
		TimeInForce:          tif,
		ReduceOnly:           o.ReduceOnly,
		MarginType:           marginType,
		Price:                o.Price.Value,
		TriggerPrice:         o.TriggerPrice,
		Amount:               o.Amount,
		ExecutedAmount:       o.FilledAmount,
		RemainingAmount:      o.Amount - o.FilledAmount,
		AverageExecutedPrice: o.AveragePrice,
		Date:                 o.CreationTimestamp.Time(),
		LastUpdated:          o.LastUpdateTimestamp.Time(),
	}, nil
}

// derivativesTradeFill converts one of the account's trades to a fill
func (e *Exchange) derivativesTradeFill(t *DerivativesUserTrade) (*fill.Data, error) {
	a, pair, err := derivativesInstrumentPair(t.InstrumentName)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(t.Direction)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", t.TradeID, err)
	}
	return &fill.Data{
		ID:            t.TradeID,
		TradeID:       t.TradeID,
		Timestamp:     t.Timestamp.Time(),
		Exchange:      e.Name,
		AssetType:     a,
		CurrencyPair:  pair,
		Side:          side,
		OrderID:       t.OrderID,
		ClientOrderID: t.Label,
		Price:         t.Price,
		Amount:        t.Amount,
	}, nil
}

// derivativesOrderType converts a gateway order type; a market_limit order is a market order whose unfilled remainder
// rests as a limit order, liquidation is only a trade's order type, and an isolated subaccount's order notification
// may leave it out, as the docs show
func derivativesOrderType(orderType string) (order.Type, error) {
	switch orderType {
	case "":
		return order.UnknownType, nil
	case DerivativesOrderTypeLimit:
		return order.Limit, nil
	case DerivativesOrderTypeMarket, DerivativesOrderTypeMarketLimit:
		return order.Market, nil
	case DerivativesOrderTypeStopLimit:
		return order.StopLimit, nil
	case DerivativesOrderTypeStopMarket:
		return order.StopMarket, nil
	case DerivativesOrderTypeTakeLimit:
		return order.TakeProfit, nil
	case DerivativesOrderTypeTakeMarket:
		return order.TakeProfitMarket, nil
	case DerivativesOrderTypeTrailingStop:
		return order.TrailingStop, nil
	case "liquidation":
		return order.Liquidation, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, orderType)
	}
}

// derivativesOrderStatus converts a gateway order state; an open or cancelled order that has started filling is
// partially filled
func derivativesOrderStatus(state string, filledAmount float64) (order.Status, error) {
	switch state {
	case DerivativesOrderStateOpen, DerivativesOrderStateTriggered:
		if filledAmount > 0 {
			return order.PartiallyFilled, nil
		}
		return order.Open, nil
	case DerivativesOrderStateUntriggered:
		return order.Pending, nil
	case DerivativesOrderStateFilled:
		return order.Filled, nil
	case DerivativesOrderStateCancelled:
		if filledAmount > 0 {
			return order.PartiallyFilledCancelled, nil
		}
		return order.Cancelled, nil
	case DerivativesOrderStateRejected:
		return order.Rejected, nil
	case "archive":
		return order.Closed, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", errDerivativesUnknownOrderState, state)
	}
}
