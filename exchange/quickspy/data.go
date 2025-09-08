package quickspy

import (
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

func (q *QuickSpy) handleWSAccountChange(data *account.Change) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(AccountHoldingsFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, AccountHoldingsFocusType)
	}
	if data.AssetType != q.key.ExchangeAssetPair.Asset ||
		(!data.Balance.Currency.Equal(q.key.ExchangeAssetPair.Pair().Base) && !data.Balance.Currency.Equal(q.key.ExchangeAssetPair.Pair().Quote)) {
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

func (q *QuickSpy) handleWSAccountChanges(data []account.Change) error {
	focus := q.focuses.GetByFocusType(AccountHoldingsFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, AccountHoldingsFocusType)
	}
	var payload []account.Balance
	for i := range data {
		if data[i].AssetType == q.key.ExchangeAssetPair.Asset &&
			(data[i].Balance.Currency.Equal(q.key.ExchangeAssetPair.Pair().Base) || data[i].Balance.Currency.Equal(q.key.ExchangeAssetPair.Pair().Quote)) {
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

func (q *QuickSpy) handleWSOrderDetail(data *order.Detail) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	if data.AssetType != q.key.ExchangeAssetPair.Asset || !data.Pair.Equal(q.key.ExchangeAssetPair.Pair()) {
		// these WS checks are here due to the inability to fully know how a subscription is transformed
		// it is not an error to get other data, just ignore it
		return nil
	}
	focus := q.focuses.GetByFocusType(ActiveOrdersFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, ActiveOrdersFocusType)
	}
	q.m.Lock()
	// managing an order list properly goes against the simplicity of quickspy.
	// If you're trying to track everything effectively, use our order manager or PRs welcome
	q.data.Orders = []order.Detail{*data}
	q.m.Unlock()
	focus.stream(data)
	focus.setSuccessful()
	return nil
}

func (q *QuickSpy) handleWSOrderDetails(data []order.Detail) error {
	focus := q.focuses.GetByFocusType(ActiveOrdersFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, ActiveOrdersFocusType)
	}
	payload := make([]order.Detail, 0, len(data))
	for i := range data {
		if data[i].Pair.Equal(q.key.ExchangeAssetPair.Pair()) &&
			data[i].AssetType == q.key.ExchangeAssetPair.Asset {
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

func (q *QuickSpy) handleWSTickers(data []ticker.Price) error {
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
			if data[i].Pair.Equal(q.key.ExchangeAssetPair.Pair()) &&
				data[i].AssetType == q.key.ExchangeAssetPair.Asset {
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

func (q *QuickSpy) handleWSTicker(data *ticker.Price) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(TickerFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, TickerFocusType)
	}
	if data.AssetType != q.key.ExchangeAssetPair.Asset || !data.Pair.Equal(q.key.ExchangeAssetPair.Pair()) {
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

func (q *QuickSpy) handleWSOrderbook(data *orderbook.Depth) error {
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
	if payload.Asset != q.key.ExchangeAssetPair.Asset || !payload.Pair.Equal(q.key.ExchangeAssetPair.Pair()) {
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

func (q *QuickSpy) handleWSTrade(data *trade.Data) error {
	if err := common.NilGuard(data); err != nil {
		return err
	}
	focus := q.focuses.GetByFocusType(TradesFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, TradesFocusType)
	}
	if data.AssetType != q.key.ExchangeAssetPair.Asset || !data.CurrencyPair.Equal(q.key.ExchangeAssetPair.Pair()) {
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

func (q *QuickSpy) handleWSTrades(data []trade.Data) error {
	focus := q.focuses.GetByFocusType(TradesFocusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, TradesFocusType)
	}
	if len(data) == 0 {
		return nil
	}
	relevantTrades := make([]trade.Data, 0, len(data))
	for i := range data {
		if data[i].AssetType == q.key.ExchangeAssetPair.Asset && data[i].CurrencyPair.Equal(q.key.ExchangeAssetPair.Pair()) {
			relevantTrades = append(relevantTrades, data[i])
		}
	}
	if len(relevantTrades) == 0 {
		return nil
	}
	q.m.Lock()
	q.data.Trades = data
	payload := q.data.Trades
	q.m.Unlock()
	focus.stream(payload)
	focus.setSuccessful()
	return nil
}

func (q *QuickSpy) handleURLFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	resp, err := q.exch.GetCurrencyTradeURL(q.credContext, q.key.ExchangeAssetPair.Asset, q.key.ExchangeAssetPair.Pair())
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	if resp == "" {
		return nil
	}
	focus.m.Lock()
	q.data.URL = resp
	focus.m.Unlock()
	focus.stream(resp)
	return nil
}

func (q *QuickSpy) handleContractFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	contracts, err := q.exch.GetFuturesContractDetails(q.credContext, q.key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	var contractOfFocus *futures.Contract
	for i := range contracts {
		if !contracts[i].Name.Equal(q.key.ExchangeAssetPair.Pair()) {
			continue
		}
		contractOfFocus = &contracts[i]
		break
	}
	if contractOfFocus == nil {
		return fmt.Errorf("no contract found for %s %s", q.key.ExchangeAssetPair, focus.focusType)
	}
	focus.m.Lock()
	q.data.Contract = contractOfFocus
	focus.m.Unlock()
	focus.stream(contractOfFocus)
	return nil
}

func (q *QuickSpy) handleKlineFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	ett := time.Now()
	stt := ett.Add(-kline.OneMonth.Duration())
	k, err := q.exch.GetHistoricCandlesExtended(q.credContext, q.key.ExchangeAssetPair.Pair(), q.key.ExchangeAssetPair.Asset, kline.OneHour, stt, ett)
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			k, err = q.exch.GetHistoricCandles(q.credContext, q.key.ExchangeAssetPair.Pair(), q.key.ExchangeAssetPair.Asset, kline.OneHour, stt, ett)
		}
		if err != nil {
			return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
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
	focus.m.Lock()
	q.data.Kline = wsConvertedCandles
	focus.m.Unlock()
	focus.stream(wsConvertedCandles)
	return nil
}

func (q *QuickSpy) handleOpenInterestFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	oi, err := q.exch.GetOpenInterest(q.credContext, key.PairAsset{
		Base:  q.key.ExchangeAssetPair.Pair().Base.Item,
		Quote: q.key.ExchangeAssetPair.Pair().Quote.Item,
		Asset: q.key.ExchangeAssetPair.Asset,
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	if len(oi) != 1 {
		return nil
	}
	resp := oi[0].OpenInterest
	focus.m.Lock()
	q.data.OpenInterest = resp
	focus.m.Unlock()
	focus.stream(resp)
	return nil
}

func (q *QuickSpy) handleTickerFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	resp, err := q.exch.UpdateTicker(q.credContext, q.key.ExchangeAssetPair.Pair(), q.key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	focus.m.Lock()
	q.data.Ticker = resp
	focus.m.Unlock()
	focus.stream(resp)

	return nil
}

func (q *QuickSpy) handleOrdersFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	resp, err := q.exch.GetActiveOrders(q.credContext, &order.MultiOrderRequest{
		Pairs:     []currency.Pair{q.key.ExchangeAssetPair.Pair()},
		AssetType: q.key.ExchangeAssetPair.Asset,
		Side:      order.AnySide,
		Type:      order.AnyType,
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	focus.m.Lock()
	q.data.Orders = resp
	focus.m.Unlock()
	focus.stream(resp)
	return nil
}

func (q *QuickSpy) handleAccountHoldingsFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	ais, err := q.exch.UpdateAccountInfo(q.credContext, q.key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w",
			q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	// filter results only to passed in key currencies
	sa := make([]account.Balance, 0, 2)
	for _, a := range ais.Accounts {
		if a.AssetType != q.key.ExchangeAssetPair.Asset {
			continue
		}
		for _, c := range a.Currencies {
			if c.Currency.Equal(q.key.ExchangeAssetPair.Base.Currency()) {
				sa = append(sa, c)
			}
			if c.Currency.Equal(q.key.ExchangeAssetPair.Quote.Currency()) {
				sa = append(sa, c)
			}
		}
	}
	focus.m.Lock()
	q.data.AccountBalance = sa
	focus.m.Unlock()
	focus.stream(sa)
	return nil
}

func (q *QuickSpy) handleOrderBookFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	ob, err := q.exch.UpdateOrderbook(q.credContext, q.key.ExchangeAssetPair.Pair(), q.key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	focus.m.Lock()
	q.data.Orderbook = ob
	focus.m.Unlock()
	focus.stream(ob)
	return nil
}

func (q *QuickSpy) handleTradesFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	tr, err := q.exch.GetRecentTrades(q.credContext, q.key.ExchangeAssetPair.Pair(), q.key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	focus.m.Lock()
	q.data.Trades = tr
	focus.m.Unlock()
	focus.stream(tr)
	return nil
}

func (q *QuickSpy) handleOrderExecutionFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	el, err := q.exch.GetOrderExecutionLimits(q.key.ExchangeAssetPair.Asset, q.key.ExchangeAssetPair.Pair())
	if err != nil {
		err = q.exch.UpdateOrderExecutionLimits(q.credContext, q.key.ExchangeAssetPair.Asset)
		if err != nil {
			return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
		}
		el, err = q.exch.GetOrderExecutionLimits(q.key.ExchangeAssetPair.Asset, q.key.ExchangeAssetPair.Pair())
		if err != nil {
			return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
		}
	}
	focus.m.Lock()
	q.data.ExecutionLimits = &el
	focus.m.Unlock()
	focus.stream(&el)
	return nil
}

func (q *QuickSpy) handleFundingRateFocus(focus *FocusData) error {
	if err := common.NilGuard(focus); err != nil {
		return err
	}
	isPerp, err := q.exch.IsPerpetualFutureCurrency(q.key.ExchangeAssetPair.Asset, q.key.ExchangeAssetPair.Pair())
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType, err)
	}
	if !isPerp {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType, futures.ErrNotPerpetualFuture)
	}
	fr, err := q.exch.GetLatestFundingRates(q.credContext, &fundingrate.LatestRateRequest{
		Asset: q.key.ExchangeAssetPair.Asset,
		Pair:  q.key.ExchangeAssetPair.Pair(),
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.key.ExchangeAssetPair, focus.focusType.String(), err)
	}
	if len(fr) != 1 {
		return fmt.Errorf("expected 1 funding rate for %s %q, got %d", q.key.ExchangeAssetPair, focus.focusType.String(), len(fr))
	}
	focus.m.Lock()
	q.data.FundingRate = &fr[0]
	focus.m.Unlock()
	focus.stream(&fr[0])
	return nil
}
