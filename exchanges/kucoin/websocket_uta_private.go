package kucoin

import (
	"context"
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// processUTAOrder relays an order push as an order detail
func (e *Exchange) processUTAOrder(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketOrder
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	a, err := utaTradeTypeAsset(resp.TradeType)
	if err != nil {
		return err
	}
	pair, err := e.utaSymbolPair(resp.Symbol, a)
	if err != nil {
		return err
	}
	side, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return err
	}
	orderType, err := order.StringToOrderType(resp.OrderType)
	if err != nil {
		return err
	}
	status, err := utaOrderStatus(resp.OrderStatus)
	if err != nil {
		return err
	}
	timeInForce, err := utaTimeInForce(resp.TimeInForce, resp.PostOnly)
	if err != nil {
		return err
	}
	marginType, err := margin.StringToMarginType(resp.MarginMode)
	if err != nil {
		return err
	}
	detail := &order.Detail{
		TimeInForce:          timeInForce,
		ReduceOnly:           resp.ReduceOnly,
		Price:                resp.Price.Float64(),
		TriggerPrice:         resp.TriggerPrice.Float64(),
		AverageExecutedPrice: resp.AveragePrice.Float64(),
		Fee:                  resp.Fee.Float64(),
		FeeAsset:             resp.FeeCurrency,
		Exchange:             e.Name,
		OrderID:              resp.OrderID,
		ClientOrderID:        resp.ClientOrderID,
		Type:                 orderType,
		Side:                 side,
		Status:               status,
		AssetType:            a,
		Date:                 resp.CreationTimestamp.Time(),
		LastUpdated:          resp.UpdateTimestamp.Time(),
		Pair:                 pair,
		MarginType:           marginType,
	}
	// The quantity, filled size and remaining size are all in the quantity unit
	switch resp.QuantityUnit {
	case "QUOTECCY":
		// No base currency amounts are pushed: the filled size is what the order has spent
		detail.QuoteAmount = resp.Quantity.Float64()
		detail.Cost = resp.FilledSize.Float64()
		detail.CostAsset = pair.Quote
	case "UNIT":
		detail.ContractAmount = resp.Quantity.Float64()
		fallthrough
	default:
		detail.Amount = resp.Quantity.Float64()
		detail.ExecutedAmount = resp.FilledSize.Float64()
		detail.RemainingAmount = resp.RemainingSize.Float64()
	}
	return e.Websocket.DataHandler.Send(ctx, detail)
}

// utaTradeTypeAsset returns the asset a UTA trade type names
func utaTradeTypeAsset(tradeType string) (asset.Item, error) {
	switch tradeType {
	case "SPOT":
		return asset.Spot, nil
	case "MARGIN":
		return asset.Margin, nil
	case "FUTURES":
		return asset.Futures, nil
	}
	return asset.Empty, fmt.Errorf("%w: trade type %q", asset.ErrNotSupported, tradeType)
}

// utaSymbolPair returns the pair a private push symbol names: spot and margin symbols carry a dash, while futures
// symbols are matched against the available contracts
func (e *Exchange) utaSymbolPair(symbol string, a asset.Item) (currency.Pair, error) {
	if a == asset.Futures {
		return e.MatchSymbolWithAvailablePairs(symbol, asset.Futures, false)
	}
	return currency.NewPairFromString(symbol)
}

// utaOrderStatus converts an order push's status. A UTA conditional order ends once triggered, as the order it places
// takes over under its own ID
func utaOrderStatus(status uint64) (order.Status, error) {
	switch status {
	case 0:
		return order.Pending, nil
	case 1:
		return order.Closed, nil
	case 2:
		return order.Open, nil
	case 3:
		return order.Filled, nil
	case 4:
		return order.PartiallyFilled, nil
	case 5:
		return order.Cancelled, nil
	case 6:
		return order.PartiallyFilledCancelled, nil
	}
	return order.UnknownStatus, fmt.Errorf("%w: %d", order.ErrUnsupportedStatusType, status)
}

// utaTimeInForce converts an order push's time in force and post-only flag. GoCryptoTrader has no equivalent of RPI,
// the retail price improvement time in force only some market makers may use
func utaTimeInForce(timeInForce string, postOnly bool) (order.TimeInForce, error) {
	var tif order.TimeInForce
	if timeInForce != "RPI" {
		var err error
		if tif, err = order.StringToTimeInForce(timeInForce); err != nil {
			return order.UnknownTIF, err
		}
	}
	if postOnly {
		tif |= order.PostOnly
	}
	return tif, nil
}

// processUTAExecutionLite relays an execution lite push as a fill, when fills are fed
func (e *Exchange) processUTAExecutionLite(msg *UTAWebsocketMessage) error {
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	var resp UTAWebsocketExecutionLite
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	f, err := e.utaFill(&resp)
	if err != nil {
		return err
	}
	return e.Websocket.Fills.Update(f)
}

// processUTAExecution relays an execution push, which carries the fee and fill details a fill lacks, and relays its
// fill when fills are fed
func (e *Exchange) processUTAExecution(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketExecution
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	if err := e.Websocket.DataHandler.Send(ctx, &resp); err != nil {
		return err
	}
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	f, err := e.utaFill(&resp.UTAWebsocketExecutionLite)
	if err != nil {
		return err
	}
	return e.Websocket.Fills.Update(f)
}

// utaFill converts an execution push into a fill. Executions name no trade type: futures symbols are told apart by
// having no dash, and margin fills report as spot since they share spot symbols
func (e *Exchange) utaFill(x *UTAWebsocketExecutionLite) (fill.Data, error) {
	a := asset.Futures
	if strings.Contains(x.Symbol, currency.DashDelimiter) {
		a = asset.Spot
	}
	pair, err := e.utaSymbolPair(x.Symbol, a)
	if err != nil {
		return fill.Data{}, err
	}
	side, err := order.StringToOrderSide(x.Side)
	if err != nil {
		return fill.Data{}, err
	}
	return fill.Data{
		Timestamp:     x.ExecutionTimestamp.Time(),
		Exchange:      e.Name,
		AssetType:     a,
		CurrencyPair:  pair,
		Side:          side,
		OrderID:       x.OrderID,
		ClientOrderID: x.ClientOrderID,
		TradeID:       string(x.TradeID),
		Price:         x.Price.Float64(),
		Amount:        x.Size.Float64(),
	}, nil
}

// processUTABalance saves a unified account balance push under spot, as the unified account's funds back its spot,
// margin and futures trading alike, and relays the saved balance. The balance is saved in a sub-account of its own,
// named for the unified account type, so the Classic spot balances REST saves neither overwrite nor remove it. Only the
// unified account's balance is subscribed to
func (e *Exchange) processUTABalance(ctx context.Context, msg *UTAWebsocketMessage, accountType string) error {
	if accountType != utaUnifiedAccountType {
		return fmt.Errorf("%w: %q", errUTAUnknownTopic, msg.Topic)
	}
	var resp UTAWebsocketBalance
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, utaUnifiedAccountType)}
	subAccts[0].Balances.Set(resp.Currency, accounts.Balance{
		Total:     resp.Balance.Float64(),
		Hold:      resp.Hold.Float64(),
		Free:      resp.Available.Float64(),
		Borrowed:  resp.Liability.Float64(),
		UpdatedAt: resp.UpdateTimestamp.Time(),
	})
	if err := e.Accounts.Save(ctx, subAccts, false); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, subAccts)
}

// processUTAPosition relays a futures position push as a position
func (e *Exchange) processUTAPosition(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketPosition
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	pair, err := e.MatchSymbolWithAvailablePairs(resp.Symbol, asset.Futures, false)
	if err != nil {
		return err
	}
	size := resp.Size.Decimal()
	position := futures.Position{
		Exchange:                     e.Name,
		Asset:                        asset.Futures,
		Pair:                         pair,
		Underlying:                   pair.Base,
		Leverage:                     resp.Leverage.Decimal(),
		NotionalSize:                 resp.PositionValue.Decimal(),
		PositionMargin:               resp.PositionMargin.Decimal(),
		InitialMarginRequirement:     resp.InitialMargin.Decimal(),
		MaintenanceMarginRequirement: resp.MaintenanceMargin.Decimal(),
		MaintenanceMarginFraction:    resp.MaintenanceMarginRate.Decimal(),
		EstimatedLiquidationPrice:    resp.LiquidationPrice.Decimal(),
		RealisedPNL:                  resp.RealisedPNL.Decimal(),
		UnrealisedPNL:                resp.UnrealisedPNL.Decimal(),
		Status:                       order.Open,
		OpeningDate:                  resp.CreationTimestamp.Time(),
		OpeningPrice:                 resp.EntryPrice.Decimal(),
		LatestPrice:                  resp.MarkPrice.Decimal(),
		LatestSize:                   size.Abs(),
		LastUpdated:                  resp.UpdateTimestamp.Time(),
	}
	switch {
	case size.IsPositive():
		position.OpeningDirection, position.LatestDirection = order.Long, order.Long
	case size.IsNegative():
		position.OpeningDirection, position.LatestDirection = order.Short, order.Short
	default:
		position.Status = order.Closed
		position.CloseDate = position.LastUpdated
	}
	return e.Websocket.DataHandler.Send(ctx, []futures.Position{position})
}

// processUTALiquidationWarning relays a liquidation warning push
func (e *Exchange) processUTALiquidationWarning(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketLiquidationWarning
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}

// processUTALeverage relays a leverage push
func (e *Exchange) processUTALeverage(ctx context.Context, msg *UTAWebsocketMessage) error {
	var resp UTAWebsocketLeverage
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	return e.Websocket.DataHandler.Send(ctx, &resp)
}
