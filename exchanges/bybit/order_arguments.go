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

	orderFilter := "" // If "Order" is not passed, "Order" by default.
	if s.AssetType == asset.Spot && s.TriggerPrice != 0 {
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

	if s.RiskManagementModes.TakeProfit.Price != 0 {
		arg.TakeProfitPrice = s.RiskManagementModes.TakeProfit.Price
		arg.TakeProfitTriggerBy = s.RiskManagementModes.TakeProfit.TriggerPriceType.String()
		arg.TpOrderType = getOrderTypeString(s.RiskManagementModes.TakeProfit.OrderType)
		arg.TpLimitPrice = s.RiskManagementModes.TakeProfit.LimitPrice
	}
	if s.RiskManagementModes.StopLoss.Price != 0 {
		arg.StopLossPrice = s.RiskManagementModes.StopLoss.Price
		arg.StopLossTriggerBy = s.RiskManagementModes.StopLoss.TriggerPriceType.String()
		arg.SlOrderType = getOrderTypeString(s.RiskManagementModes.StopLoss.OrderType)
		arg.SlLimitPrice = s.RiskManagementModes.StopLoss.LimitPrice
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

	return &AmendOrderRequest{
		Category:             getCategoryName(action.AssetType),
		Symbol:               pair,
		OrderID:              action.OrderID,
		OrderLinkID:          action.ClientOrderID,
		OrderQuantity:        action.Amount,
		Price:                action.Price,
		TriggerPrice:         action.TriggerPrice,
		TriggerPriceType:     action.TriggerPriceType.String(),
		TakeProfitPrice:      action.RiskManagementModes.TakeProfit.Price,
		TakeProfitTriggerBy:  getOrderTypeString(action.RiskManagementModes.TakeProfit.OrderType),
		TakeProfitLimitPrice: action.RiskManagementModes.TakeProfit.LimitPrice,
		StopLossPrice:        action.RiskManagementModes.StopLoss.Price,
		StopLossTriggerBy:    action.RiskManagementModes.StopLoss.TriggerPriceType.String(),
		StopLossLimitPrice:   action.RiskManagementModes.StopLoss.LimitPrice,
	}, nil
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
