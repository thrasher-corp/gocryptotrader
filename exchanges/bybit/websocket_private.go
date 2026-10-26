package bybit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// wsHandleAuthenticatedData handles a message from the private stream
func (e *Exchange) wsHandleAuthenticatedData(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	topic, err := jsonparser.GetString(respRaw, "topic")
	if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
		return err
	}
	if topic == "" {
		return e.handleNoTopicWebsocketResponse(ctx, conn, respRaw)
	}
	switch name, _, _ := strings.Cut(topic, "."); {
	case topic == ChannelSpreadOrder, topic == ChannelSpreadExecution:
		return e.wsHandleSpreadPrivateData(ctx, respRaw)
	case topic == ChannelRFQQuotes, topic == ChannelRFQTrades, topic == ChannelRFQInquiries:
		return e.wsHandleRFQPrivateData(ctx, respRaw)
	// Event contract executions and orders also arrive on the generic execution and order topics, with category event
	case topic == ChannelEventPosition, topic == ChannelEventExecution, topic == ChannelEventOrder,
		(topic == chanExecution || topic == chanOrder) && isEventContractFrame(respRaw):
		return e.wsHandleEventPrivateData(ctx, respRaw)
	case name == chanPositions:
		return e.wsProcessPosition(ctx, respRaw)
	case topic == chanFastExecution, strings.HasPrefix(topic, chanFastExecution+"."):
		return e.wsProcessFastExecution(ctx, respRaw)
	case name == chanExecution:
		return e.wsProcessExecution(ctx, respRaw)
	case name == chanOrder:
		return e.wsProcessOrder(ctx, respRaw)
	case name == chanWallet:
		return e.wsProcessWalletPushData(ctx, respRaw)
	case name == chanGreeks:
		return relayWebsocketResponse[WebsocketGreeksUpdate](ctx, e, respRaw)
	case name == chanStrategy:
		return relayWebsocketResponse[WebsocketStrategyUpdate](ctx, e, respRaw)
	}
	return fmt.Errorf("%w: %s", errUnhandledStreamData, topic)
}

// wsProcessPosition emits canonical positions from Bybit's account-wide private stream so consumers do not depend on
// exchange wire types
func (e *Exchange) wsProcessPosition(ctx context.Context, respRaw []byte) error {
	data, errs := websocketEntries[WebsocketPosition](respRaw, "position index", "positionIdx")
	positions := make([]futures.Position, 0, len(data))
	for i := range data {
		p := &data[i]
		report := func(err error) {
			errs = common.AppendError(errs, fmt.Errorf("%s %s position index %d: %w", p.Category, p.Symbol, p.PositionIndex, err))
		}
		pair, a, err := e.matchPairAssetFromResponse(p.Category, p.Symbol)
		if err != nil {
			report(err)
			continue
		}
		size := p.Size.Decimal()
		isSizeZero := size.IsZero()
		direction := order.UnknownSide
		if !isSizeZero || p.Side != "" {
			direction, err = order.StringToOrderSide(p.Side)
			if err != nil {
				report(err)
				continue
			}
			direction, err = direction.Position()
			if err != nil {
				report(err)
				continue
			}
		} else {
			switch p.PositionIndex {
			case 1:
				direction = order.Long
			case 2:
				direction = order.Short
			}
		}
		collateralCurrency, err := positionCollateralCurrency(a, pair, p.Symbol)
		if err != nil {
			report(err)
			continue
		}
		status := order.Closed
		if !isSizeZero {
			status = order.Open
		}
		riskStatus := order.UnknownStatus
		switch p.PositionStatus {
		case "Liq":
			riskStatus = order.Liquidated
		case "Adl":
			riskStatus = order.AutoDeleverage
		}
		var closeDate time.Time
		if isSizeZero {
			closeDate = p.UpdatedTime.Time()
		}
		positionMargin := p.PositionInitialMargin.Decimal()
		// positionMM includes estimated closing fees, so dividing it by positionValue does not recover the tier rate.
		// MaintenanceMarginFraction requires the matching riskId tier from Bybit's /v5/market/risk-limit endpoint and
		// remains unset until that data is available here.
		positions = append(positions, futures.Position{
			Exchange:           e.Name,
			Asset:              a,
			Pair:               pair,
			Underlying:         pair.Base,
			CollateralCurrency: collateralCurrency,
			Leverage:           p.Leverage.Decimal(),
			NotionalSize:       p.PositionValue.Decimal(),
			PositionMargin:     positionMargin,
			// Bybit defines positionIM as the position's initial margin, so both canonical fields share the source value
			InitialMarginRequirement:     positionMargin,
			MaintenanceMarginRequirement: p.PositionMaintenanceMargin.Decimal(),
			EstimatedLiquidationPrice:    p.LiquidationPrice.Decimal(),
			UpdateID:                     p.Sequence,
			RealisedPNL:                  p.CurrentRealisedPNL.Decimal(),
			UnrealisedPNL:                p.UnrealisedPNL.Decimal(),
			Status:                       status,
			RiskStatus:                   riskStatus,
			OpeningDate:                  p.OpenTime.Time(),
			OpeningPrice:                 p.EntryPrice.Decimal(),
			OpeningDirection:             direction,
			LatestPrice:                  p.MarkPrice.Decimal(),
			LatestSize:                   size,
			LatestDirection:              direction,
			LastUpdated:                  p.UpdatedTime.Time(),
			CloseDate:                    closeDate,
		})
	}
	if len(positions) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, positions))
}

func positionCollateralCurrency(a asset.Item, pair currency.Pair, symbol string) (currency.Code, error) {
	switch a {
	case asset.CoinMarginedFutures:
		return pair.Base, nil
	case asset.USDTMarginedFutures:
		return currency.USDT, nil
	case asset.USDCMarginedFutures:
		return currency.USDC, nil
	case asset.Options:
		if strings.HasSuffix(symbol, currency.DashDelimiter+currency.USDT.String()) {
			return currency.USDT, nil
		}
		return currency.USDC, nil
	default:
		return currency.EMPTYCODE, fmt.Errorf("%w: %s", asset.ErrNotSupported, a)
	}
}

func (e *Exchange) wsProcessExecution(ctx context.Context, respRaw []byte) error {
	data, errs := websocketEntries[WebsocketExecution](respRaw, "execution", "execId")
	fills := make([]fill.Data, 0, len(data))
	for i := range data {
		x := &data[i]
		if !isTradeExecution(x.ExecutionType) {
			continue
		}
		f, err := e.executionFill(x.Category, x.Symbol, x.Side, x.ExecutionID, x.OrderID, x.OrderLinkID)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s %s execution %s: %w", x.Category, x.Symbol, x.ExecutionID, err))
			continue
		}
		f.Timestamp = x.ExecutionTime.Time()
		f.Price = x.ExecutionPrice.Float64()
		f.Amount = x.ExecutionQuantity.Float64()
		fills = append(fills, f)
	}
	if len(fills) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, fills))
}

// websocketEntries decodes each entry of a private push's data on its own, so that an entry that cannot be decoded is
// reported, named by its category, symbol and the value of idKey, without losing the push's other entries
func websocketEntries[T any](respRaw []byte, kind, idKey string) ([]T, error) {
	var resp struct {
		CreationTime types.Time        `json:"creationTime"`
		Data         []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return nil, err
	}
	entries := make([]T, 0, len(resp.Data))
	var errs error
	for _, raw := range resp.Data {
		var entry T
		if err := json.Unmarshal(raw, &entry); err != nil {
			category, _ := jsonparser.GetString(raw, "category")
			symbol, _ := jsonparser.GetString(raw, "symbol")
			id, _, _, _ := jsonparser.Get(raw, idKey)
			errs = common.AppendError(errs, fmt.Errorf("%s %s %s %s: %w", category, symbol, kind, id, err))
			continue
		}
		entries = append(entries, entry)
	}
	return entries, errs
}

// isTradeExecution returns whether an execution type is a trade, as opposed to a funding payment, a settlement, a
// delivery or a stock split adjustment, which the execution stream also pushes
func isTradeExecution(execType string) bool {
	switch execType {
	case "Trade", "AdlTrade", "BustTrade", "BlockTrade", "MovePosition", "FutureSpread":
		return true
	}
	return false
}

func (e *Exchange) wsProcessFastExecution(ctx context.Context, respRaw []byte) error {
	data, errs := websocketEntries[WebsocketFastExecution](respRaw, "execution", "execId")
	fills := make([]fill.Data, 0, len(data))
	for i := range data {
		x := &data[i]
		f, err := e.executionFill(x.Category, x.Symbol, x.Side, x.ExecutionID, x.OrderID, x.OrderLinkID)
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s %s execution %s: %w", x.Category, x.Symbol, x.ExecutionID, err))
			continue
		}
		f.Timestamp = x.ExecutionTime.Time()
		f.Price = x.ExecutionPrice.Float64()
		f.Amount = x.ExecutionQuantity.Float64()
		fills = append(fills, f)
	}
	if len(fills) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, fills))
}

// executionFill returns a fill with the identifying fields an execution and a fast execution share
func (e *Exchange) executionFill(category, symbol, side, executionID, orderID, orderLinkID string) (fill.Data, error) {
	cp, a, err := e.matchPairAssetFromResponse(category, symbol)
	if err != nil {
		return fill.Data{}, err
	}
	s, err := order.StringToOrderSide(side)
	if err != nil {
		return fill.Data{}, err
	}
	return fill.Data{
		ID:            executionID,
		Exchange:      e.Name,
		AssetType:     a,
		CurrencyPair:  cp,
		Side:          s,
		OrderID:       orderID,
		ClientOrderID: orderLinkID,
	}, nil
}

// wsProcessOrder relays order pushes as order details
func (e *Exchange) wsProcessOrder(ctx context.Context, respRaw []byte) error {
	data, errs := websocketEntries[WebsocketOrder](respRaw, "order", "orderId")
	details := make([]order.Detail, 0, len(data))
	for i := range data {
		o := &data[i]
		report := func(err error) {
			errs = common.AppendError(errs, fmt.Errorf("%s %s order %s: %w", o.Category, o.Symbol, o.OrderID, err))
		}
		cp, a, err := e.matchPairAssetFromResponse(o.Category, o.Symbol)
		if err != nil {
			report(err)
			continue
		}
		side, err := order.StringToOrderSide(o.Side)
		if err != nil {
			report(err)
			continue
		}
		orderType, err := orderTypeFromString(o.OrderType)
		if err != nil {
			report(err)
			continue
		}
		orderType = conditionalOrderType(orderType, o.StopOrderType)
		tif, err := timeInForceFromString(o.TimeInForce)
		if err != nil {
			report(err)
			continue
		}
		var fees order.Fees
		settle, err := orderCostCurrency(a, cp, o.Symbol)
		if err == nil {
			fees, err = orderFees(a, settle, o.CumulativeFeeDetail, o.CumulativeExecutedFee)
		}
		if err != nil {
			report(err)
			continue
		}
		d := order.Detail{
			TimeInForce:          tif,
			ReduceOnly:           o.ReduceOnly,
			Price:                o.Price.Float64(),
			Amount:               o.Quantity.Float64(),
			TriggerPrice:         o.TriggerPrice.Float64(),
			AverageExecutedPrice: o.AveragePrice.Float64(),
			ExecutedAmount:       o.CumulativeExecutedQuantity.Float64(),
			RemainingAmount:      o.LeavesQuantity.Float64(),
			Cost:                 o.CumulativeExecutedValue.Float64(),
			CostAsset:            settle,
			Exchange:             e.Name,
			OrderID:              o.OrderID,
			ClientOrderID:        o.OrderLinkID,
			Type:                 orderType,
			Side:                 side,
			Status:               StringToOrderStatus(o.OrderStatus),
			AssetType:            a,
			Date:                 o.CreatedTime.Time(),
			LastUpdated:          o.UpdatedTime.Time(),
			Pair:                 cp,
		}
		if a != asset.Spot {
			d.SettlementCurrency = settle
		}
		// A spot market order sized by value holds a quote amount in qty, and the base amount it bought in cumExecQty
		if quoteSized(a, o.OrderType, o.Side, o.MarketUnit) {
			d.QuoteAmount, d.Amount = d.Amount, d.ExecutedAmount
		}
		d.SetFees(fees)
		details = append(details, d)
	}
	if len(details) == 0 {
		return errs
	}
	return common.AppendError(errs, e.Websocket.DataHandler.Send(ctx, details))
}

// timeInForceFromString returns the time in force Bybit names; an RPI order rests on the book as a post only order that
// only non-algorithmic users can match
func timeInForceFromString(s string) (order.TimeInForce, error) {
	if s == "RPI" {
		return order.PostOnly, nil
	}
	return order.StringToTimeInForce(s)
}

// wsProcessWalletPushData stores and emits the unified trading account's balances as the spot sub-account, where
// UpdateAccountBalances stores the account for every asset type
func (e *Exchange) wsProcessWalletPushData(ctx context.Context, respRaw []byte) error {
	var resp WebsocketWalletUpdate
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	subAcct := accounts.NewSubAccount(asset.Spot, "")
	for x := range resp.Data {
		for y := range resp.Data[x].Coins {
			coin := &resp.Data[x].Coins[y]
			update := unifiedBalance(coin)
			balance, err := e.Accounts.UpdateBalance(ctx, "", asset.Spot, coin.Coin, func(balance *accounts.Balance) {
				balance.Total = update.Total
				balance.Hold = update.Hold
				balance.Free = update.Free
				balance.Borrowed = update.Borrowed
				balance.AvailableWithoutBorrow = update.AvailableWithoutBorrow
			})
			if err != nil {
				return err
			}
			subAcct.Balances.Set(coin.Coin, balance)
		}
	}
	return e.Websocket.DataHandler.Send(ctx, accounts.SubAccounts{subAcct})
}

// generateAuthSubscriptions generates default subscription for the dedicated auth websocket connection. These are
// agnostic to the asset type and pair as all account level data will be routed through this connection. Setup disables
// any authenticated subscription in the configuration, since these are not configurable
func (e *Exchange) generateAuthSubscriptions() (subscription.List, error) {
	if !e.Websocket.CanUseAuthenticatedEndpoints() {
		return nil, nil
	}

	// The dcp, fast execution, greeks and strategy topics are left for callers to subscribe to: a dcp subscription
	// changes when Bybit cancels the account's orders on disconnection
	var subscriptions subscription.List
	for _, channel := range []string{chanPositions, chanExecution, chanOrder, chanWallet} {
		subscriptions = append(subscriptions, &subscription.Subscription{Channel: channel, Asset: asset.All})
	}
	return subscriptions, nil
}

func (e *Exchange) authSubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.sendSubscriptions(ctx, conn, subscribeOperation, qualifyByChannelName(subs))
}

func (e *Exchange) authUnsubscribe(ctx context.Context, conn websocket.Connection, subs subscription.List) error {
	return e.sendSubscriptions(ctx, conn, unsubscribeOperation, qualifyByChannelName(subs))
}

// qualifyByChannelName sets the topic of each subscription that has none to its channel name. Private and system topics
// take no symbol, so a subscription names its topic, such as position, order.spot or dcp.future, by its channel
func qualifyByChannelName(subs subscription.List) subscription.List {
	for _, s := range subs {
		if s.QualifiedChannel == "" {
			s.QualifiedChannel = channelName(s)
		}
	}
	return subs
}
