package bybit

import (
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func (e *Exchange) deriveSubmitOrderArguments(s *order.Submit) (*PlaceOrderRequest, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	formattedPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	side := sideBuy
	if s.Side.IsShort() {
		side = sideSell
	}

	if s.AssetType == asset.USDCMarginedFutures && !formattedPair.Quote.Equal(currency.PERP) {
		formattedPair.Delimiter = currency.DashDelimiter
	}

	timeInForce := "GTC"
	if s.Type == order.Market {
		timeInForce = "IOC"
	} else {
		switch {
		case s.TimeInForce.Is(order.FillOrKill):
			timeInForce = "FOK"
		case s.TimeInForce.Is(order.PostOnly):
			timeInForce = "PostOnly"
		case s.TimeInForce.Is(order.ImmediateOrCancel):
			timeInForce = "IOC"
		}
	}

	orderFilter := "Order" // If "Order" is not passed, "Order" by default.
	if s.TakeProfit.Price != 0 || s.TakeProfit.LimitPrice != 0 ||
		s.StopLoss.Price != 0 || s.StopLoss.LimitPrice != 0 {
		orderFilter = ""
	} else if s.TriggerPrice != 0 {
		orderFilter = "tpslOrder"
	}

	var triggerPriceType string
	if s.TriggerPrice != 0 {
		triggerPriceType = s.TriggerPriceType.String()
	}

	arg := &PlaceOrderRequest{
		Category:         getCategoryName(s.AssetType),
		Symbol:           formattedPair,
		Side:             side,
		OrderType:        orderTypeToString(s.Type),
		OrderQuantity:    s.Amount,
		Price:            s.Price,
		OrderLinkID:      s.ClientOrderID,
		EnableBorrow:     s.AssetType == asset.Margin,
		ReduceOnly:       s.ReduceOnly,
		OrderFilter:      orderFilter,
		TriggerPrice:     s.TriggerPrice,
		TimeInForce:      timeInForce,
		TriggerPriceType: triggerPriceType,
	}
	if arg.TriggerPrice != 0 {
		arg.TriggerPriceType = s.TriggerPriceType.String()
	}
	if s.TakeProfit.Price != 0 {
		arg.TakeProfitPrice = s.TakeProfit.Price
		arg.TakeProfitTriggerBy = s.TakeProfit.TriggerPriceType.String()
		arg.TpLimitPrice = s.TakeProfit.LimitPrice
		if s.TakeProfit.LimitPrice != 0 {
			arg.TpOrderType = getOrderTypeString(order.Limit)
		} else {
			arg.TpOrderType = getOrderTypeString(order.Market)
		}
	}
	if s.StopLoss.Price != 0 {
		arg.StopLossPrice = s.StopLoss.Price
		arg.StopLossTriggerBy = s.StopLoss.TriggerPriceType.String()
		arg.SlLimitPrice = s.StopLoss.LimitPrice
		if s.StopLoss.LimitPrice != 0 {
			arg.SlOrderType = getOrderTypeString(order.Limit)
		} else {
			arg.SlOrderType = getOrderTypeString(order.Market)
		}
	}
	return arg, nil
}

func (e *Exchange) deriveAmendOrderArguments(action *order.Modify) (*AmendOrderRequest, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}

	pair, err := e.FormatExchangeCurrency(action.Pair, action.AssetType)
	if err != nil {
		return nil, err
	}

	if action.AssetType == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
		pair.Delimiter = currency.DashDelimiter
	}

	arg := &AmendOrderRequest{
		Category:         getCategoryName(action.AssetType),
		Symbol:           pair,
		OrderID:          action.OrderID,
		OrderLinkID:      action.ClientOrderID,
		OrderQuantity:    action.Amount,
		Price:            action.Price,
		TriggerPrice:     action.TriggerPrice,
		TriggerPriceType: action.TriggerPriceType.String(),
	}
	if arg.TriggerPrice != 0 {
		arg.TriggerPriceType = action.TriggerPriceType.String()
	}
	if action.TakeProfit.Price != 0 {
		arg.TakeProfitPrice = action.TakeProfit.Price
		arg.TakeProfitTriggerBy = action.TakeProfit.TriggerPriceType.String()
		arg.TakeProfitLimitPrice = action.TakeProfit.LimitPrice
	}
	if action.StopLoss.Price != 0 {
		arg.StopLossPrice = action.StopLoss.Price
		arg.StopLossTriggerBy = action.StopLoss.TriggerPriceType.String()
		arg.StopLossLimitPrice = action.StopLoss.LimitPrice
	}
	return arg, nil
}

func (e *Exchange) deriveCancelOrderArguments(ord *order.Cancel) (*CancelOrderRequest, error) {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return nil, err
	}
	pair, err := e.FormatExchangeCurrency(ord.Pair, ord.AssetType)
	if err != nil {
		return nil, err
	}
	if ord.AssetType == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
		pair.Delimiter = currency.DashDelimiter
	}
	return &CancelOrderRequest{
		Category:    getCategoryName(ord.AssetType),
		Symbol:      pair,
		OrderID:     ord.OrderID,
		OrderLinkID: ord.ClientOrderID,
	}, nil
}
