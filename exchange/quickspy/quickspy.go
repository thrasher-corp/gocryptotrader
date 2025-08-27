package quickspy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// NewQuickSpy creates a new QuickSpy
func NewQuickSpy(k *CredentialsKey, focuses []FocusData) (*QuickSpy, error) {
	if k == nil {
		return nil, errNoKey
	}
	if len(focuses) == 0 {
		return nil, errNoFocus
	}

	sm := NewFocusStore()
	for i := range focuses {
		focuses[i].Init()
		if err := focuses[i].Validate(k); err != nil {
			return nil, fmt.Errorf("focus %q %w: %w", focuses[i].Type, errValidationFailed, err)
		}
		sm.Upsert(focuses[i].Type, &focuses[i])
	}
	credContext := context.Background()
	q := &QuickSpy{
		Key:                k,
		dataHandlerChannel: make(chan any),
		shutdown:           make(chan any),
		Focuses:            sm,
		credContext:        credContext,
		Data:               &Data{Key: k},
		m:                  new(sync.RWMutex),
	}
	err := q.setupExchange()
	if err != nil {
		return nil, err
	}
	if q.AnyRequiresAuth() {
		if k.Credentials.IsEmpty() {
			return nil, fmt.Errorf("%w for %q %q %q", errNoCredentials, k.Key.Exchange, k.Key.Asset, k.Key.Pair())
		}
		q.credContext = account.DeployCredentialsToContext(context.Background(), k.Credentials)
		b := q.Exch.GetBase()
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
	}

	return q, nil
}

func (q *QuickSpy) AnyRequiresWebsocket() bool {
	for _, focus := range q.Focuses.List() {
		if focus.RequiresWebsocket() {
			return true
		}
	}
	return false
}

func (q *QuickSpy) AnyRequiresAuth() bool {
	for _, focus := range q.Focuses.List() {
		if focus.RequiresAuth() {
			return true
		}
	}
	return false
}

func (q *QuickSpy) FocusTypeRequiresWebsocket(focusType FocusType) bool {
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return false
	}
	return focus.UseWebsocket
}

func (q *QuickSpy) GetAndWaitForFocusByKey(focusType FocusType) (*FocusData, error) {
	focus, err := q.GetFocusByKey(focusType)
	if err != nil {
		return nil, err
	}
	timeout := time.NewTimer(focus.RESTPollTime)
	select {
	case <-focus.HasBeenSuccessfulChan:
		return focus, nil
	case <-timeout.C:
		return nil, fmt.Errorf("%w %q", errFocusDataTimeout, focusType)
	}
}

func (q *QuickSpy) GetFocusByKey(focusType FocusType) (*FocusData, error) {
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return nil, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	return focus, nil
}

func (q *QuickSpy) setupExchange() error {
	q.m.Lock()
	defer q.m.Unlock()

	e, err := engine.NewSupportedExchangeByName(q.Key.Key.Exchange)
	if err != nil {
		return err
	}

	b := e.GetBase()
	if err := q.setupExchangeDefaults(e, b); err != nil {
		return err
	}

	if err := q.setupCurrencyPairs(b); err != nil {
		return err
	}

	if err := q.checkRateLimits(b); err != nil {
		return err
	}

	if err := q.setupWebsocket(e, b); err != nil {
		return err
	}

	q.Exch = e
	return nil
}

func (q *QuickSpy) setupExchangeDefaults(e exchange.IBotExchange, b *exchange.Base) error {
	e.SetDefaults()
	exchCfg, err := b.GetStandardConfig()
	if err != nil {
		return fmt.Errorf("%q %q %q: %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), err)
	}
	if err := b.SetupDefaults(exchCfg); err != nil {
		return fmt.Errorf("%q %q %q: %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), err)
	}
	if err := e.Setup(exchCfg); err != nil {
		return fmt.Errorf("%q %q %q: %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), err)
	}
	return nil
}

func (q *QuickSpy) setupCurrencyPairs(b *exchange.Base) error {
	var rFmt, cFmt *currency.PairFormat
	if b.CurrencyPairs.UseGlobalFormat {
		rFmt = b.CurrencyPairs.RequestFormat
		cFmt = b.CurrencyPairs.ConfigFormat
	} else {
		rFmt = b.CurrencyPairs.Pairs[q.Key.Key.Asset].RequestFormat
		cFmt = b.CurrencyPairs.Pairs[q.Key.Key.Asset].ConfigFormat
	}
	b.CurrencyPairs.DisableAllPairs()
	b.CurrencyPairs.Pairs[q.Key.Key.Asset] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: rFmt,
		ConfigFormat:  cFmt,
	}
	if err := b.CurrencyPairs.StorePairs(q.Key.Key.Asset, currency.Pairs{q.Key.Key.Pair()}, false); err != nil {
		return err
	}
	return b.CurrencyPairs.StorePairs(q.Key.Key.Asset, currency.Pairs{q.Key.Key.Pair()}, true)
}

func (q *QuickSpy) checkRateLimits(b *exchange.Base) error {
	if len(b.GetRateLimiterDefinitions()) == 0 {
		return fmt.Errorf("exchange %s has no rate limits. Quickspy requires rate limits to be set", q.Key.Key.Exchange)
	}
	return nil
}

func (q *QuickSpy) setupWebsocket(e exchange.IBotExchange, b *exchange.Base) error {
	if q.AnyRequiresWebsocket() {
		if !e.SupportsWebsocket() {
			return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.Key.Key.Exchange)
		}
	} else {
		return nil
	}

	if !b.Features.Supports.Websocket {
		return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.Key.Key.Exchange)
	}

	b.Websocket.ToRoutine = q.dataHandlerChannel
	var newSubs []*subscription.Subscription
	if q.FocusTypeRequiresWebsocket(TickerFocusType) {
		newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[TickerFocusType]})
	}
	if q.FocusTypeRequiresWebsocket(TradesFocusType) {
		newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[TradesFocusType]})
	}
	if q.FocusTypeRequiresWebsocket(OrderBookFocusType) {
		newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[OrderBookFocusType]})
	}
	if q.FocusTypeRequiresWebsocket(AccountHoldingsFocusType) {
		newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[AccountHoldingsFocusType]})
	}
	b.Config.Features.Subscriptions = newSubs
	if err := b.Websocket.Connect(); err != nil {
		return fmt.Errorf("failed to connect websocket for %s: %w", q.Key.Key.Exchange, err)
	}
	return nil
}

func (q *QuickSpy) Run() error {
	if q.AnyRequiresWebsocket() {
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			err := q.HandleWS()
			if err != nil {
				panic(err)
			}
		}()
	}
	for i, focus := range q.Focuses.List() {
		if focus.UseWebsocket {
			continue
		}
		q.wg.Add(1)
		go func(f *FocusData) {
			defer q.wg.Done()
			err := q.RunRESTFocus(f.Type)
			if err != nil {
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %q %q %q failed, focus type: %q err: %v",
					i+1, q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), f.Type, err)
			}
		}(focus)
	}
	return nil
}

func (q *QuickSpy) HandleWS() error {
	for {
		select {
		case <-q.shutdown:
			return nil
		case d := <-q.dataHandlerChannel:
			switch data := d.(type) {
			case []ticker.Price:
				focus := q.Focuses.GetByFocusType(TickerFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",
						q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), TickerFocusType)
					continue
				}
				if len(data) != 1 {
					continue
				}
				q.m.Lock()
				q.Data.Ticker = &data[0]
				q.m.Unlock()
				select {
				case focus.Stream <- data:
				default:
					// drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case *ticker.Price:
				focus := q.Focuses.GetByFocusType(TickerFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",
						q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), TickerFocusType)
					continue
				}
				q.m.Lock()
				q.m.Unlock()
				select {
				case focus.Stream <- data:
				default:
					// drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case *orderbook.Depth:
				focus := q.Focuses.GetByFocusType(OrderBookFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",
						q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), OrderBookFocusType)
					continue
				}

				q.m.Lock()
				var err error
				q.Data.Orderbook, err = data.Retrieve()
				if err != nil {
					select {
					case focus.Stream <- err:
					default: // drop data that doesn't fit or get listened to
					}
					continue
				}

				select {
				case focus.Stream <- q.Data.Orderbook:
				default: // drop data that doesn't fit or get listened to
				}
				q.m.Unlock()
				focus.SetSuccessful()
			case trade.Data:
				focus := q.Focuses.GetByFocusType(TradesFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",
						q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), TradesFocusType)
					continue
				}
				q.m.Lock()
				q.Data.Trades = []trade.Data{data}
				select {
				case focus.Stream <- q.Data.Trades:
				default: // drop data that doesn't fit or get listened to
				}
				q.m.Unlock()
				focus.SetSuccessful()
			case []trade.Data:
				focus := q.Focuses.GetByFocusType(TradesFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",
						q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), TradesFocusType)
					continue
				}
				if len(data) == 0 {
					continue
				}
				q.m.Lock()
				q.Data.Trades = data
				select {
				case focus.Stream <- q.Data.Trades:
				default: // drop data that doesn't fit or get listened to
				}
				q.m.Unlock()
				focus.SetSuccessful()
			case []websocket.KlineData:
				focus := q.Focuses.GetByFocusType(KlineFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",
						q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), TradesFocusType)
					continue
				}
				q.m.Lock()
				q.Data.Kline = data
				q.m.Unlock()
				select {
				case focus.Stream <- data:
				default: // drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			}
		}
	}
}

func (q *QuickSpy) RunRESTFocus(focusType FocusType) error {
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	if focus.UseWebsocket {
		return nil
	}
	timer := time.NewTimer(0)
	failures := 0
	for {
		select {
		case <-q.shutdown:
			return nil
		case <-timer.C:
			err := q.handleFocusType(focusType, focus, timer)
			if err != nil {
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %q %q %q failed, focus type: %s err: %v",
					failures+1, q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focusType, err)
				if focus.IsOnceOff {
					return nil
				}
				if !focus.hasBeenSuccessful {
					if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
						q.successfulSpy(focus, timer)
						return nil
					}
					if failures == 5 {
						return fmt.Errorf("Quickspy data attempt: %v/5 %q %q %q failed, focus type: %s err: %v ", failures, q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focusType, err)
					}
					failures++
					timer.Reset(time.Second)
				}
			}
		}
	}
}

func (q *QuickSpy) handleFocusType(focusType FocusType, focus *FocusData, timer *time.Timer) error {
	var err error
	switch focusType {
	case URLFocusType:
		err = q.handleURLFocus(focus)
	case ContractFocusType:
		err = q.handleContractFocus(focus)
	case KlineFocusType:
		err = q.handleKlineFocus(focus)
	case OpenInterestFocusType:
		err = q.handleOpenInterestFocus(focus)
	case TickerFocusType:
		err = q.handleTickerFocus(focus)
	case ActiveOrdersFocusType:
		err = q.handleOrdersFocus(focus)
	case AccountHoldingsFocusType:
		err = q.handleAccountHoldingsFocus(focus)
	case OrderPlacementFocusType:
		// No implementation provided in the original code
	case OrderBookFocusType:
		err = q.handleOrderBookFocus(focus)
	case TradesFocusType:
		err = q.handleTradesFocus(focus)
	case OrderExecutionFocusType:
		err = q.handleOrderExecutionFocus(focus)
	case FundingRateFocusType:
		err = q.handleFundingRateFocus(focus)
	default:
		return fmt.Errorf("unknown focus type %v", focusType)
	}
	if err != nil {
		select {
		case focus.Stream <- err:
		default: // drop data that doesn't fit or get listened to
		}
		return err
	}
	q.successfulSpy(focus, timer)

	return nil
}

func (q *QuickSpy) handleURLFocus(focus *FocusData) error {
	resp, err := q.Exch.GetCurrencyTradeURL(q.credContext, q.Key.Key.Asset, q.Key.Key.Pair())
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	if resp == "" {
		return nil
	}
	focus.m.Lock()
	q.Data.URL = resp
	focus.m.Unlock()
	select {
	case focus.Stream <- resp:
	default: // drop data that doesn't fit or get listened to
	}
	return nil
}

func (q *QuickSpy) handleContractFocus(focus *FocusData) error {
	contracts, err := q.Exch.GetFuturesContractDetails(q.credContext, q.Key.Key.Asset)
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	var contractOfFocus *futures.Contract
	for i := range contracts {
		if !contracts[i].Name.Equal(q.Key.Key.Pair()) {
			continue
		}
		contractOfFocus = &contracts[i]
		break
	}
	if contractOfFocus == nil {
		return fmt.Errorf("no contract found for %v %v %v %v", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String())
	}
	focus.m.Lock()
	q.Data.Contract = contractOfFocus
	focus.m.Unlock()
	select {
	case focus.Stream <- contractOfFocus:
	default: // drop data that doesn't fit or get listened to
	}
	return nil
}

func (q *QuickSpy) handleKlineFocus(focus *FocusData) error {
	tt := time.Now().Add(-kline.ThreeMonth.Duration())
	k, err := q.Exch.GetHistoricCandlesExtended(q.credContext, q.Key.Key.Pair(), q.Key.Key.Asset, kline.OneHour, tt, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			k, err = q.Exch.GetHistoricCandles(q.credContext, q.Key.Key.Pair(), q.Key.Key.Asset, kline.OneHour, tt, time.Now())
		}
		if err != nil {
			return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
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
	q.Data.Kline = wsConvertedCandles
	focus.m.Unlock()
	select {
	case focus.Stream <- wsConvertedCandles:
	default: // drop data that doesn't fit or get listened to
	}
	return nil
}

func (q *QuickSpy) handleOpenInterestFocus(focus *FocusData) error {
	oi, err := q.Exch.GetOpenInterest(q.credContext, key.PairAsset{
		Base:  q.Key.Key.Pair().Base.Item,
		Quote: q.Key.Key.Pair().Quote.Item,
		Asset: q.Key.Key.Asset,
	})
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	if len(oi) != 1 {
		return nil
	}
	focus.m.Lock()
	q.Data.OpenInterest = oi[0].OpenInterest
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleTickerFocus(focus *FocusData) error {
	tick, err := q.Exch.UpdateTicker(q.credContext, q.Key.Key.Pair(), q.Key.Key.Asset)
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Ticker = tick
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleOrdersFocus(focus *FocusData) error {
	resp, err := q.Exch.GetActiveOrders(q.credContext, &order.MultiOrderRequest{
		Pairs:     []currency.Pair{q.Key.Key.Pair()},
		AssetType: q.Key.Key.Asset,
	})
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Orders = resp
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleAccountHoldingsFocus(focus *FocusData) error {
	ais, err := account.GetHoldings(q.Key.Key.Exchange, q.Key.Credentials, q.Key.Key.Asset)
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w",
			q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Account = &ais
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleOrderBookFocus(focus *FocusData) error {
	ob, err := q.Exch.UpdateOrderbook(q.credContext, q.Key.Key.Pair(), q.Key.Key.Asset)
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Orderbook = ob
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleTradesFocus(focus *FocusData) error {
	tr, err := q.Exch.GetRecentTrades(q.credContext, q.Key.Key.Pair(), q.Key.Key.Asset)
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Trades = tr
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleOrderExecutionFocus(focus *FocusData) error {
	el, err := q.Exch.GetOrderExecutionLimits(q.Key.Key.Asset, q.Key.Key.Pair())
	if err != nil {
		err = q.Exch.UpdateOrderExecutionLimits(q.credContext, q.Key.Key.Asset)
		if err != nil {
			return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
		}
		el, err = q.Exch.GetOrderExecutionLimits(q.Key.Key.Asset, q.Key.Key.Pair())
		if err != nil {
			return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
		}
	}
	focus.m.Lock()
	q.Data.ExecutionLimits = &el
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleFundingRateFocus(focus *FocusData) error {

	isPerp, err := q.Exch.IsPerpetualFutureCurrency(q.Key.Key.Asset, q.Key.Key.Pair())
	if err != nil && !errors.Is(err, futures.ErrNotPerpetualFuture) {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	if !isPerp {
		// Hard to validate if its a perp at startup
		// if its not a perp, just say its successful to
		// stop it polling
		// let the user feel bashful for their choices
		q.successfulSpy(focus, time.NewTimer(focus.RESTPollTime))
		return nil
	}
	fr, err := q.Exch.GetLatestFundingRates(q.credContext, &fundingrate.LatestRateRequest{
		Asset:                q.Key.Key.Asset,
		Pair:                 q.Key.Key.Pair(),
		IncludePredictedRate: true,
	})
	if err != nil {
		return fmt.Errorf("%q %q %q %q %w", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), err)
	}
	if len(fr) != 1 {
		return fmt.Errorf("expected 1 funding rate for %q %q %q %q, got %d", q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair(), focus.Type.String(), len(fr))
	}
	focus.m.Lock()
	q.Data.FundingRate = &fr[0]
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) successfulSpy(focus *FocusData, timer *time.Timer) {
	focus.SetSuccessful()
	focus.m.RLock()
	defer focus.m.RUnlock()
	if focus.IsOnceOff {
		return
	}
	timer.Reset(focus.RESTPollTime)
}

func (q *QuickSpy) Shutdown() {
	close(q.shutdown)
}

func (q *QuickSpy) Dump() (*ExportedData, error) {
	q.m.RLock()
	defer q.m.RUnlock()
	return q.Data.Dump(key.NewExchangeAssetPair(q.Key.Key.Exchange, q.Key.Key.Asset, q.Key.Key.Pair()), !q.Key.Credentials.IsEmpty())
}

func (d *Data) Dump(k key.ExchangeAssetPair, hasCredentials bool) (*ExportedData, error) {
	var (
		underlyingBase, underlyingQuote *currency.Item
		contractExpirationTime          time.Time
		contractType                    string
		contractSettlement              string
		contractDecimals                float64
	)
	if d.Contract != nil {
		underlyingBase = d.Contract.Underlying.Base.Item
		underlyingQuote = d.Contract.Underlying.Quote.Item
		contractExpirationTime = d.Contract.EndDate
		contractType = d.Contract.Type.String()
		contractDecimals = d.Contract.Multiplier
		contractSettlement = d.Contract.SettlementCurrencies.Join()
	}
	var (
		lastPrice, indexPrice, markPrice, volume,
		spread, spreadPercent, fundingRate, estimatedFundingRate,
		lastTradePrice, lastTradeSize float64
		bids, asks orderbook.Levels
	)
	if d.Ticker != nil {
		lastPrice = d.Ticker.Last
		indexPrice = d.Ticker.IndexPrice
		markPrice = d.Ticker.MarkPrice
		volume = d.Ticker.Volume
	}
	if d.Orderbook != nil {
		bids = d.Orderbook.Bids
		asks = d.Orderbook.Asks
	}
	if d.FundingRate != nil {
		fundingRate = d.FundingRate.LatestRate.Rate.InexactFloat64()
		estimatedFundingRate = d.FundingRate.PredictedUpcomingRate.Rate.InexactFloat64()
	}
	if len(d.Trades) > 0 {
		lastTrade := d.Trades[len(d.Trades)-1]
		lastTradePrice = lastTrade.Price
		lastTradeSize = lastTrade.Amount
	}
	holdings := []account.Holdings{}
	if d.Account != nil {
		holdings = append(holdings, *d.Account)
	}
	var (
		nextFundingRateTime, currentFundingRateTime time.Time
	)
	return &ExportedData{
		Key:                    k,
		UnderlyingBase:         underlyingBase,
		UnderlyingQuote:        underlyingQuote,
		ContractExpirationTime: contractExpirationTime,
		ContractType:           contractType,
		ContractDecimals:       contractDecimals,
		HasValidCredentials:    hasCredentials,
		LastPrice:              lastPrice,
		IndexPrice:             indexPrice,
		MarkPrice:              markPrice,
		Volume:                 volume,
		Spread:                 spread,
		SpreadPercent:          spreadPercent,
		FundingRate:            fundingRate,
		EstimatedFundingRate:   estimatedFundingRate,
		LastTradePrice:         lastTradePrice,
		LastTradeSize:          lastTradeSize,
		Holdings:               holdings,
		Orders:                 d.Orders,
		Bids:                   bids,
		Asks:                   asks,
		OpenInterest:           d.OpenInterest,
		NextFundingRateTime:    nextFundingRateTime,
		CurrentFundingRateTime: currentFundingRateTime,
		ExecutionLimits:        *d.ExecutionLimits,
		URL:                    d.URL,
		ContractSettlement:     contractSettlement,
	}, nil
}

// WaitForInitialData allows a caller to wait for a response before doing other actions
func (q *QuickSpy) WaitForInitialData(focusType FocusType) error {
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	<-focus.HasBeenSuccessfulChan
	return nil
}
