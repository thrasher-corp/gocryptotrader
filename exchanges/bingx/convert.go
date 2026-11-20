package bingx

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// convertOrderStatus converts the status of a spot or perpetual order. Spot reports NEW, PENDING, PARTIALLY_FILLED,
// FILLED, CANCELED and FAILED. Perpetual order queries spell two of theirs PARTIALLYFILLED and CANCELLED, and perpetual
// order events add EXPIRED. The coin-M order event example reports Unknown, which converts to order.UnknownStatus: order
// stores keep the status they hold for it
func convertOrderStatus(status string) (order.Status, error) {
	switch status {
	case "NEW":
		return order.New, nil
	case "PENDING":
		return order.Pending, nil
	case "PARTIALLY_FILLED", "PARTIALLYFILLED":
		return order.PartiallyFilled, nil
	case "FILLED":
		return order.Filled, nil
	case "CANCELED", "CANCELLED":
		return order.Cancelled, nil
	case "EXPIRED":
		return order.Expired, nil
	case "FAILED":
		return order.Rejected, nil
	case "Unknown":
		return order.UnknownStatus, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", order.ErrUnsupportedStatusType, status)
	}
}

// convertOrderType converts the type of a spot or perpetual order. Spot's TAKE_STOP_LIMIT and TAKE_STOP_MARKET, the
// TRIGGER_LIMIT and TRIGGER_MARKET both products take, and perpetual STOP and STOP_MARKET are stop orders; perpetual
// TAKE_PROFIT is a take profit limit order, and TRAILING_TP_SL trails as a take profit or a stop loss
func convertOrderType(orderType OrderType) (order.Type, error) {
	switch orderType {
	case OrderTypeLimit:
		return order.Limit, nil
	case OrderTypeMarket:
		return order.Market, nil
	case OrderTypeStop, OrderTypeTakeStopLimit, OrderTypeTriggerLimit:
		return order.StopLimit, nil
	case OrderTypeStopMarket, OrderTypeTakeStopMarket, OrderTypeTriggerMarket:
		return order.StopMarket, nil
	case OrderTypeTakeProfit:
		return order.TakeProfit, nil
	case OrderTypeTakeProfitMarket:
		return order.TakeProfitMarket, nil
	case OrderTypeTrailingStopMarket, OrderTypeTrailingTakeProfitOrStopLoss:
		return order.TrailingStop, nil
	case OrderTypeLiquidation:
		return order.Liquidation, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, orderType)
	}
}

// convertOrderSide converts the side of an order, which both products report as BUY or SELL
func convertOrderSide(side Side) (order.Side, error) {
	switch side {
	case SideBuy:
		return order.Buy, nil
	case SideSell:
		return order.Sell, nil
	default:
		return order.UnknownSide, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, side)
	}
}

// convertTimeInForce converts the time in force of an order, which both products report as GTC, IOC, FOK or PostOnly.
// An order reporting none, such as a market order, has order.UnknownTIF
func convertTimeInForce(timeInForce TimeInForce) (order.TimeInForce, error) {
	switch timeInForce {
	case TimeInForceGoodTillCancel:
		return order.GoodTillCancel, nil
	case TimeInForceImmediateOrCancel:
		return order.ImmediateOrCancel, nil
	case TimeInForceFillOrKill:
		return order.FillOrKill, nil
	case TimeInForcePostOnly:
		return order.PostOnly, nil
	case "":
		return order.UnknownTIF, nil
	default:
		return order.UnknownTIF, fmt.Errorf("%w: %q", order.ErrInvalidTimeInForce, timeInForce)
	}
}

// convertCoinMarginedOrderStatus converts the status of a coin-M order, which coin-M replies spell in upper case, such
// as PENDING, or in mixed case, such as Pending
func convertCoinMarginedOrderStatus(status string) (order.Status, error) {
	if strings.EqualFold(status, "Unknown") {
		return order.UnknownStatus, nil
	}
	return convertOrderStatus(strings.ToUpper(status))
}

// convertPositionSide converts a position side, which coin-M replies spell in upper case, such as LONG, or in mixed
// case, such as Long
func convertPositionSide(positionSide string) PositionSide {
	return PositionSide(strings.ToUpper(positionSide))
}

// formatOrderSide returns the side BingX takes for an order's direction
func formatOrderSide(side order.Side) (Side, error) {
	switch {
	case side.IsLong():
		return SideBuy, nil
	case side.IsShort():
		return SideSell, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}
}

// resolveOrderType returns the type an order is placed as: a bare stop or take profit order is the limit form when it
// has a price and the market form otherwise
func resolveOrderType(orderType order.Type, price float64) order.Type {
	switch orderType {
	case order.Stop:
		if price > 0 {
			return order.StopLimit
		}
		return order.StopMarket
	case order.TakeProfit:
		if price > 0 {
			return order.TakeProfit
		}
		return order.TakeProfitMarket
	default:
		return orderType
	}
}

// formatOrderType returns the order type of an asset BingX takes for a resolved order type, the inverse of
// convertOrderType: spot stop orders are its TAKE_STOP orders, which trigger in either direction, perpetual stop and
// take profit limit orders STOP and TAKE_PROFIT, and USDT-M trailing stops TRAILING_STOP_MARKET
func formatOrderType(a asset.Item, orderType order.Type) (OrderType, error) {
	switch orderType {
	case order.Limit:
		return OrderTypeLimit, nil
	case order.Market:
		return OrderTypeMarket, nil
	}
	switch a {
	case asset.Spot:
		switch orderType {
		case order.StopLimit:
			return OrderTypeTakeStopLimit, nil
		case order.StopMarket:
			return OrderTypeTakeStopMarket, nil
		}
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		switch orderType {
		case order.StopLimit:
			return OrderTypeStop, nil
		case order.StopMarket:
			return OrderTypeStopMarket, nil
		case order.TakeProfit:
			return OrderTypeTakeProfit, nil
		case order.TakeProfitMarket:
			return OrderTypeTakeProfitMarket, nil
		case order.TrailingStop:
			if a == asset.USDTMarginedFutures {
				return OrderTypeTrailingStopMarket, nil
			}
		}
	}
	return "", fmt.Errorf("%w: %s on %s", order.ErrUnsupportedOrderType, orderType, a)
}

// orderTypeTakesPrice reports whether a BingX order type takes a limit price
func orderTypeTakesPrice(orderType OrderType) bool {
	switch orderType {
	case OrderTypeLimit, OrderTypeStop, OrderTypeTakeProfit, OrderTypeTakeStopLimit, OrderTypeTriggerLimit:
		return true
	default:
		return false
	}
}

// orderTypeTakesTrigger reports whether a BingX order type triggers at a stop price
func orderTypeTakesTrigger(orderType OrderType) bool {
	switch orderType {
	case OrderTypeStop, OrderTypeStopMarket, OrderTypeTakeProfit, OrderTypeTakeProfitMarket, OrderTypeTakeStopLimit,
		OrderTypeTakeStopMarket, OrderTypeTriggerLimit, OrderTypeTriggerMarket:
		return true
	default:
		return false
	}
}

// orderTypeTrails reports whether a BingX order type trails the price, by the distance its price holds or by its price
// rate, from its activation price, which its trigger price stands for
func orderTypeTrails(orderType OrderType) bool {
	return orderType == OrderTypeTrailingStopMarket || orderType == OrderTypeTrailingTakeProfitOrStopLoss
}

// formatTimeInForce returns the time in force BingX takes, the inverse of convertTimeInForce; an unset one is left out,
// which BingX takes as GTC
func formatTimeInForce(timeInForce order.TimeInForce) (TimeInForce, error) {
	switch timeInForce {
	case order.UnknownTIF:
		return "", nil
	case order.GoodTillCancel:
		return TimeInForceGoodTillCancel, nil
	case order.ImmediateOrCancel:
		return TimeInForceImmediateOrCancel, nil
	case order.FillOrKill:
		return TimeInForceFillOrKill, nil
	case order.PostOnly, order.PostOnly | order.GoodTillCancel:
		return TimeInForcePostOnly, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, timeInForce)
	}
}

// formatWorkingType returns the price a perpetual conditional order triggers on: GCT's last price is BingX's contract
// price
func formatWorkingType(priceType order.PriceType) (WorkingType, error) {
	switch priceType {
	case order.LastPrice:
		return WorkingTypeContractPrice, nil
	case order.MarkPrice:
		return WorkingTypeMarkPrice, nil
	case order.IndexPrice:
		return WorkingTypeIndexPrice, nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnknownPriceType, priceType)
	}
}
