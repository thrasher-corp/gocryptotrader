package okx

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// errUnknownPushAccount reports a private push on a connection with no
// recorded login, which leaves the account the push belongs to unknown
var errUnknownPushAccount = errors.New("the push's connection has not logged in, so its account is unknown")

// wsProcessAccount saves the balances of an account push under the account
// its connection logged in as, as UpdateAccountBalances saves the trading
// account's, and relays the push. Every push, a snapshot's pages included,
// merges its currencies into the stored balances, which UpdateAccountBalances
// replaces.
func (e *Exchange) wsProcessAccount(ctx context.Context, conn websocket.Connection, respRaw []byte) error {
	resp := new(WebsocketAccountPush)
	if err := json.Unmarshal(respRaw, resp); err != nil {
		return err
	}
	subAcct := accounts.NewSubAccount(asset.Spot, "")
	for i := range resp.Data {
		for j := range resp.Data[i].Details {
			detail := &resp.Data[i].Details[j]
			if detail.Currency.IsEmpty() {
				continue
			}
			subAcct.Balances.Set(detail.Currency, accounts.Balance{
				Total:     detail.Equity.Float64(),
				Hold:      detail.FrozenBalance.Float64(),
				Free:      detail.AvailableBalance.Float64(),
				Borrowed:  detail.Liabilities.Float64(),
				UpdatedAt: detail.UpdateTime.Time(),
			})
		}
	}
	if len(subAcct.Balances) != 0 {
		creds, err := e.websocketLoginCredentials(conn)
		if err != nil {
			return err
		}
		if err := e.Accounts.Save(accounts.DeployCredentialsToContext(ctx, creds), accounts.SubAccounts{subAcct}, false); err != nil {
			return err
		}
	}
	return e.Websocket.DataHandler.Send(ctx, resp)
}

// wsProcessBalanceAndPosition relays a balance_and_position push carrying
// balances, positions or trades. The push carries a currency's cash alone,
// which the account channel and UpdateAccountBalances report beside the
// equity, available and frozen amounts it changes, so the balance store is
// left to them.
func (e *Exchange) wsProcessBalanceAndPosition(ctx context.Context, data []byte) error {
	resp := new(WebsocketBalanceAndPositionPush)
	if err := json.Unmarshal(data, resp); err != nil {
		return err
	}
	for i := range resp.Data {
		if len(resp.Data[i].Balances) != 0 || len(resp.Data[i].Positions) != 0 || len(resp.Data[i].Trades) != 0 {
			return e.Websocket.DataHandler.Send(ctx, resp)
		}
	}
	return nil
}

// wsProcessOrders relays the orders of an orders push as order details
func (e *Exchange) wsProcessOrders(ctx context.Context, respRaw []byte) error {
	var resp WebsocketOrdersPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	a, err := assetTypeFromInstrumentType(resp.Argument.InstrumentType)
	if err != nil {
		return err
	}
	for x := range resp.Data {
		orderType, tif, err := orderTypeFromString(resp.Data[x].OrderType)
		if err != nil {
			return err
		}
		orderStatus, err := order.StringToOrderStatus(resp.Data[x].State)
		if err != nil {
			return err
		}
		avgPrice := resp.Data[x].AveragePrice.Float64()
		execAmount := resp.Data[x].AccumulatedFillSize.Float64()
		quoteSizedOrder := quoteSized(a, resp.Data[x].OrderType, resp.Data[x].Side, resp.Data[x].SizeType)
		orderAmount, remainingAmount, quoteAmount := orderAmounts(resp.Data[x].Size.Float64(), execAmount, avgPrice, quoteSizedOrder, orderStatus)
		cost, costAsset := orderCost(a, resp.Data[x].AveragePrice, resp.Data[x].AccumulatedFillSize, resp.Data[x].InstrumentID, resp.Data[x].TradeQuoteCurrency)
		fee, feeAsset := orderFee(resp.Data[x].Fee, resp.Data[x].FeeCurrency, resp.Data[x].Rebate, resp.Data[x].RebateCurrency)

		d := &order.Detail{
			Amount:               orderAmount,
			AssetType:            a,
			AverageExecutedPrice: avgPrice,
			ClientOrderID:        resp.Data[x].ClientOrderID,
			Cost:                 cost,
			CostAsset:            costAsset,
			Date:                 resp.Data[x].CreationTime.Time(),
			Exchange:             e.Name,
			ExecutedAmount:       execAmount,
			Fee:                  fee,
			FeeAsset:             feeAsset,
			LastUpdated:          resp.Data[x].UpdateTime.Time(),
			Leverage:             resp.Data[x].Leverage.Float64(),
			OrderID:              resp.Data[x].OrderID,
			Pair:                 resp.Data[x].InstrumentID,
			Price:                resp.Data[x].Price.Float64(),
			QuoteAmount:          quoteAmount,
			ReduceOnly:           resp.Data[x].ReduceOnly,
			RemainingAmount:      remainingAmount,
			Side:                 resp.Data[x].Side,
			Status:               orderStatus,
			Type:                 orderType,
			TimeInForce:          tif,
		}
		if orderStatus == order.Filled {
			d.CloseTime = resp.Data[x].FillTime.Time()
			if d.Amount == 0 {
				d.Amount = d.ExecutedAmount
			}
		}
		if err := e.Websocket.DataHandler.Send(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// wsProcessFills relays the fills of a fills push to the fills feed when it
// is enabled. A spot instrument's fill does not say whether it was a spot or
// margin trade, so it is relayed once, under the instrument's first enabled
// asset, rather than once for each. The fills channel pushes every
// instrument's fills, so a fill of an instrument with no enabled asset is
// reported without holding back the push's other fills.
func (e *Exchange) wsProcessFills(respRaw []byte) error {
	var resp WebsocketFillsPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if !e.IsFillsFeedEnabled() {
		return nil
	}
	fills := make([]fill.Data, 0, len(resp.Data))
	var errs error
	for i := range resp.Data {
		assets, err := e.getAssetsFromInstrumentID(resp.Data[i].InstrumentID.String())
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("fill trade ID %s: %w", resp.Data[i].TradeID, err))
			continue
		}
		fills = append(fills, fill.Data{
			ID:            resp.Data[i].TradeID,
			Timestamp:     resp.Data[i].Timestamp.Time(),
			Exchange:      e.Name,
			AssetType:     assets[0],
			CurrencyPair:  resp.Data[i].InstrumentID,
			Side:          resp.Data[i].Side,
			OrderID:       resp.Data[i].OrderID,
			ClientOrderID: resp.Data[i].ClientOrderID,
			TradeID:       resp.Data[i].TradeID,
			Price:         resp.Data[i].FillPrice.Float64(),
			Amount:        resp.Data[i].FillSize.Float64(),
		})
	}
	return common.AppendError(errs, e.Websocket.Fills.Update(fills...))
}

// wsProcessSpreadOrders relays the spread orders of a sprd-orders push as order details.
// No quote amount is reported: a spread's price is the difference between its
// legs, and OKX sizes inverse spreads in USD, so size times price is not a
// notional.
func (e *Exchange) wsProcessSpreadOrders(ctx context.Context, respRaw []byte) error {
	var resp WebsocketSpreadOrdersPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return kline.ErrNoTimeSeriesDataToConvert
	}
	orderDetails := make([]order.Detail, len(resp.Data))
	for x := range resp.Data {
		oStatus, err := order.StringToOrderStatus(resp.Data[x].State)
		if err != nil {
			return err
		}
		oType, tif, err := orderTypeFromString(resp.Data[x].OrderType)
		if err != nil {
			return err
		}
		spreadAmt := resp.Data[x].Size.Float64()
		spreadExec := resp.Data[x].AccumulatedFillSize.Float64()
		spreadRemaining := spreadRemainingSize(resp.Data[x].PendingFillSize, spreadAmt, spreadExec, oStatus)
		orderDetails[x] = order.Detail{
			AssetType:            asset.Spread,
			Amount:               spreadAmt,
			AverageExecutedPrice: resp.Data[x].AveragePrice.Float64(),
			ClientOrderID:        resp.Data[x].ClientOrderID,
			Date:                 resp.Data[x].CreationTime.Time(),
			Exchange:             e.Name,
			ExecutedAmount:       spreadExec,
			OrderID:              resp.Data[x].OrderID,
			Pair:                 resp.Data[x].SpreadID,
			Price:                resp.Data[x].Price.Float64(),
			RemainingAmount:      spreadRemaining,
			Side:                 resp.Data[x].Side,
			Status:               oStatus,
			Type:                 oType,
			LastUpdated:          resp.Data[x].UpdateTime.Time(),
			TimeInForce:          tif,
		}
	}
	return e.Websocket.DataHandler.Send(ctx, orderDetails)
}

// wsProcessSpreadTrades relays the fills of a sprd-trades push to the fills
// feed when it is enabled, and saves them as spread trades when trade saving
// is enabled; a failed save does not hold back the fills
func (e *Exchange) wsProcessSpreadTrades(respRaw []byte) error {
	var resp WebsocketSpreadTradesPush
	if err := json.Unmarshal(respRaw, &resp); err != nil {
		return err
	}
	var saveErr error
	if e.IsSaveTradeDataEnabled() {
		trades := make([]trade.Data, len(resp.Data))
		for i := range resp.Data {
			trades[i] = trade.Data{
				TID:          resp.Data[i].TradeID,
				Exchange:     e.Name,
				CurrencyPair: resp.Data[i].SpreadID,
				AssetType:    asset.Spread,
				Side:         resp.Data[i].Side,
				Price:        resp.Data[i].FillPrice.Float64(),
				Amount:       resp.Data[i].FillSize.Float64(),
				Timestamp:    resp.Data[i].Timestamp.Time(),
			}
		}
		saveErr = trade.AddTradesToBuffer(trades...)
	}
	if !e.IsFillsFeedEnabled() {
		return saveErr
	}
	fills := make([]fill.Data, len(resp.Data))
	for i := range resp.Data {
		fills[i] = fill.Data{
			ID:            resp.Data[i].TradeID,
			Timestamp:     resp.Data[i].Timestamp.Time(),
			Exchange:      e.Name,
			AssetType:     asset.Spread,
			CurrencyPair:  resp.Data[i].SpreadID,
			Side:          resp.Data[i].Side,
			OrderID:       resp.Data[i].OrderID,
			ClientOrderID: resp.Data[i].ClientOrderID,
			TradeID:       resp.Data[i].TradeID,
			Price:         resp.Data[i].FillPrice.Float64(),
			Amount:        resp.Data[i].FillSize.Float64(),
		}
	}
	return common.AppendError(saveErr, e.Websocket.Fills.Update(fills...))
}
