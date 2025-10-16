package quickdata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func (q *QuickData) handleWSAccountChange(data *account.Change) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(AccountHoldingsFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, AccountHoldingsFocusType)
	}
	if data.AssetType != q.key.Asset ||
		(!data.Balance.Currency.Equal(q.key.Pair().Base) && !data.Balance.Currency.Equal(q.key.Pair().Quote)) {
		// these WS checks are here due to the inability to fully know how a subscription is transformed
		// it is not an error to get other data, just ignore it
		return nil
	}
	payload := make([]account.Balance, 1)
	payload[0] = *data.Balance
	q.m.Lock()
	q.data.AccountBalance = payload
	q.m.Unlock()
	focus.stream(payload)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSAccountChanges(data []account.Change) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(AccountHoldingsFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, AccountHoldingsFocusType)
	}
	var payload []account.Balance
	for i := range data {
		if data[i].AssetType == q.key.Asset &&
			(data[i].Balance.Currency.Equal(q.key.Pair().Base) || data[i].Balance.Currency.Equal(q.key.Pair().Quote)) {
			payload = append(payload, *data[i].Balance)
		}
	}
	if len(payload) == 0 {
		return nil
	}
	q.m.Lock()
	q.data.AccountBalance = payload
	q.m.Unlock()
	focus.stream(payload)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSOrderDetail(data *order.Detail) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	if data.AssetType != q.key.Asset || !data.Pair.Equal(q.key.Pair()) {
		// these WS checks are here due to the inability to fully know how a subscription is transformed
		// it is not an error to get other data, just ignore it
		return nil
	}
	focus := q.focuses.GetByFocusType(ActiveOrdersFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, ActiveOrdersFocusType)
	}
	q.m.Lock()
	// managing an order list properly goes against the simplicity of quickData.
	// If you're trying to track everything effectively, use our order manager or PRs welcome
	q.data.Orders = []order.Detail{*data}
	q.m.Unlock()
	focus.stream(data)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSOrderDetails(data []order.Detail) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(ActiveOrdersFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, ActiveOrdersFocusType)
	}
	payload := make([]order.Detail, 0, len(data))
	for i := range data {
		if data[i].Pair.Equal(q.key.Pair()) &&
			data[i].AssetType == q.key.Asset {
			payload = append(payload, data[i])
		}
	}
	if len(payload) == 0 {
		return nil
	}
	q.m.Lock()
	q.data.Orders = payload
	q.m.Unlock()
	focus.stream(payload)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSTickers(data []ticker.Price) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(TickerFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, TickerFocusType)
	}
	var payload *ticker.Price
	switch {
	case len(data) == 0:
	case len(data) == 1:
		payload = &data[0]
	case len(data) > 1:
		for i := range data {
			if data[i].Pair.Equal(q.key.Pair()) &&
				data[i].AssetType == q.key.Asset {
				payload = &data[i]
				break
			}
		}
	}
	if payload == nil {
		return nil
	}
	q.m.Lock()
	q.data.Ticker = payload
	q.m.Unlock()
	focus.stream(payload)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSTicker(data *ticker.Price) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(TickerFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, TickerFocusType)
	}
	if data.AssetType != q.key.Asset || !data.Pair.Equal(q.key.Pair()) {
		// these WS checks are here due to the inability to fully know how a subscription is transformed
		// it is not an error to get other data, just ignore it
		return nil
	}
	q.m.Lock()
	q.data.Ticker = data
	q.m.Unlock()
	focus.stream(data)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSOrderbook(data *orderbook.Depth) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(OrderBookFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, OrderBookFocusType)
	}
	payload, err := data.Retrieve()
	if err != nil {
		focus.stream(err)
		return err
	}
	if payload.Asset != q.key.Asset || !payload.Pair.Equal(q.key.Pair()) {
		// these WS checks are here due to the inability to fully know how a subscription is transformed
		// it is not an error to get other data, just ignore it
		return nil
	}
	q.m.Lock()
	q.data.Orderbook = payload
	q.m.Unlock()
	focus.stream(payload)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSTrade(data *trade.Data) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(TradesFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, TradesFocusType)
	}
	if data.AssetType != q.key.Asset || !data.CurrencyPair.Equal(q.key.Pair()) {
		// these WS checks are here due to the inability to fully know how a subscription is transformed
		// it is not an error to get other data, just ignore it
		return nil
	}

	q.m.Lock()
	q.data.Trades = []trade.Data{*data}
	payload := q.data.Trades
	q.m.Unlock()
	focus.stream(payload)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleWSTrades(data []trade.Data) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(TradesFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, TradesFocusType)
	}
	if len(data) == 0 {
		return nil
	}
	relevantTrades := make([]trade.Data, 0, len(data))
	for i := range data {
		if data[i].AssetType == q.key.Asset && data[i].CurrencyPair.Equal(q.key.Pair()) {
			relevantTrades = append(relevantTrades, data[i])
		}
	}
	if len(relevantTrades) == 0 {
		return nil
	}
	q.m.Lock()
	q.data.Trades = relevantTrades
	q.m.Unlock()
	focus.stream(q.data.Trades)
	focus.setSuccessful()
	return nil
}

func (q *QuickData) handleURLFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	resp, err := q.exch.GetCurrencyTradeURL(ctx, q.key.Asset, q.key.Pair())
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	if resp == "" {
		return nil
	}
	q.m.Lock()
	q.data.URL = resp
	q.m.Unlock()
	focus.stream(resp)
	return nil
}

func (q *QuickData) handleContractFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	contracts, err := q.exch.GetFuturesContractDetails(ctx, q.key.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	var contractOfFocus *futures.Contract
	for i := range contracts {
		if !contracts[i].Name.Equal(q.key.Pair()) {
			continue
		}
		contractOfFocus = &contracts[i]
		break
	}
	if contractOfFocus == nil {
		return fmt.Errorf("no contract found for %s %s", q.key, focus.focusType)
	}
	q.m.Lock()
	q.data.Contract = contractOfFocus
	q.m.Unlock()
	focus.stream(contractOfFocus)
	return nil
}

func (q *QuickData) handleKlineFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	ett := time.Now()
	stt := ett.Add(-kline.OneMonth.Duration())
	k, err := q.exch.GetHistoricCandlesExtended(ctx, q.key.Pair(), q.key.Asset, kline.OneHour, stt, ett)
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			k, err = q.exch.GetHistoricCandles(ctx, q.key.Pair(), q.key.Asset, kline.OneHour, stt, ett)
		}
		if err != nil {
			return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
		}
	}
	if len(k.Candles) == 0 {
		return nil
	}
	start := k.Candles[0].Time
	end := k.Candles[len(k.Candles)-1].Time
	wsConvertedCandles := make([]websocket.KlineData, len(k.Candles))
	for i := range k.Candles {
		wsConvertedCandles[i] = websocket.KlineData{
			Timestamp:  k.Candles[i].Time,
			Pair:       k.Pair,
			AssetType:  k.Asset,
			Exchange:   k.Exchange,
			StartTime:  start,
			CloseTime:  end,
			Interval:   k.Interval.String(),
			OpenPrice:  k.Candles[i].Open,
			ClosePrice: k.Candles[i].Close,
			HighPrice:  k.Candles[i].High,
			LowPrice:   k.Candles[i].Low,
			Volume:     k.Candles[i].Volume,
		}
	}
	q.m.Lock()
	q.data.Kline = wsConvertedCandles
	q.m.Unlock()
	focus.stream(wsConvertedCandles)
	return nil
}

func (q *QuickData) handleOpenInterestFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	oi, err := q.exch.GetOpenInterest(ctx, key.PairAsset{
		Base:  q.key.Pair().Base.Item,
		Quote: q.key.Pair().Quote.Item,
		Asset: q.key.Asset,
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	if len(oi) != 1 {
		return nil
	}
	resp := oi[0].OpenInterest
	q.m.Lock()
	q.data.OpenInterest = resp
	q.m.Unlock()
	focus.stream(resp)
	return nil
}

func (q *QuickData) handleTickerFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	resp, err := q.exch.UpdateTicker(ctx, q.key.Pair(), q.key.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	q.m.Lock()
	q.data.Ticker = resp
	q.m.Unlock()
	focus.stream(resp)
	return nil
}

func (q *QuickData) handleOrdersFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	resp, err := q.exch.GetActiveOrders(ctx, &order.MultiOrderRequest{
		Pairs:     []currency.Pair{q.key.Pair()},
		AssetType: q.key.Asset,
		Side:      order.AnySide,
		Type:      order.AnyType,
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	q.m.Lock()
	q.data.Orders = resp
	q.m.Unlock()
	focus.stream(resp)
	return nil
}

func (q *QuickData) handleAccountHoldingsFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	ais, err := q.exch.UpdateAccountInfo(ctx, q.key.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w",
			q.key, focus.focusType.String(), err)
	}
	// filter results only to passed in key currencies
	sa := make([]account.Balance, 0, 2)
	// iterate on account index as it is not a pointer
	for i := range ais.Accounts {
		if ais.Accounts[i].AssetType != q.key.Asset {
			continue
		}
		for _, c := range ais.Accounts[i].Currencies {
			if c.Currency.Equal(q.key.Base.Currency()) {
				sa = append(sa, c)
			}
			if c.Currency.Equal(q.key.Quote.Currency()) {
				sa = append(sa, c)
			}
		}
	}
	q.m.Lock()
	q.data.AccountBalance = sa
	q.m.Unlock()
	focus.stream(sa)
	return nil
}

func (q *QuickData) handleOrderBookFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	ob, err := q.exch.UpdateOrderbook(ctx, q.key.Pair(), q.key.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	q.m.Lock()
	q.data.Orderbook = ob
	q.m.Unlock()
	focus.stream(ob)
	return nil
}

func (q *QuickData) handleTradesFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	tr, err := q.exch.GetRecentTrades(ctx, q.key.Pair(), q.key.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	q.m.Lock()
	q.data.Trades = tr
	q.m.Unlock()
	focus.stream(tr)
	return nil
}

func (q *QuickData) handleOrderExecutionFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	el, err := q.exch.GetOrderExecutionLimits(q.key.Asset, q.key.Pair())
	if err != nil {
		err = q.exch.UpdateOrderExecutionLimits(ctx, q.key.Asset)
		if err != nil {
			return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
		}
		el, err = q.exch.GetOrderExecutionLimits(q.key.Asset, q.key.Pair())
		if err != nil {
			return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
		}
	}
	q.m.Lock()
	q.data.ExecutionLimits = &el
	q.m.Unlock()
	focus.stream(&el)
	return nil
}

func (q *QuickData) handleFundingRateFocus(ctx context.Context, focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	isPerp, err := q.exch.IsPerpetualFutureCurrency(q.key.Asset, q.key.Pair())
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType, err)
	}
	if !isPerp {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType, futures.ErrNotPerpetualFuture)
	}
	fr, err := q.exch.GetLatestFundingRates(ctx, &fundingrate.LatestRateRequest{
		Asset: q.key.Asset,
		Pair:  q.key.Pair(),
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key, focus.focusType.String(), err)
	}
	if len(fr) != 1 {
		return fmt.Errorf("expected 1 funding rate for %s %q, got %d", q.key, focus.focusType.String(), len(fr))
	}
	q.m.Lock()
	q.data.FundingRate = &fr[0]
	q.m.Unlock()
	focus.stream(&fr[0])
	return nil
}
