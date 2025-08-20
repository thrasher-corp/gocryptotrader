package quickspy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var focusToSub = map[FocusType]string{
	OrderBookFocusType: subscription.OrderbookChannel,
	TickerFocusType:    subscription.TickerChannel,
	KlineFocusType:     subscription.CandlesChannel,
}

// NewQuickSpy creates a new QuickSpy
func NewQuickSpy(k Key, focuses []FocusData) (*QuickSpy, error) {
	k.Exchange = strings.ToLower(k.Exchange)
	sm := NewFocusStore()
	for i := range focuses {
		if !focuses[i].Enabled {
			log.Warnf(log.QuickSpy, "skipping focustype %q since its disabled", focuses[i].Type)
			continue
		}
		if k.Credentials.IsEmpty() {
			if focuses[i].Type == AccountHoldingsFocusType || focuses[i].Type == OrdersFocusType || focuses[i].Type == OrderPlacementFocusType {
				log.Warnf(log.QuickSpy, "skipping authenticated focustype %q without credentials", focuses[i].Type)
				continue
			}
		}

		focuses[i].Init()
		sm.Upsert(focuses[i].Type, &focuses[i])
	}
	credContext := context.Background()
	q := &QuickSpy{
		Key:                &k,
		dataHandlerChannel: make(chan any),
		shutUP:             make(chan any),
		Focuses:            sm,
		credContext:        credContext,
		Data:               Data{Key: &k},
		m:                  &sync.RWMutex{},
	}
	err := q.setupExchange(k)
	if err != nil {
		return nil, err
	}
	if q.RequiresAuth() {
		q.credContext = account.DeployCredentialsToContext(context.Background(), k.Credentials)
		b := q.Exch.GetBase()
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
	}

	return q, nil
}

func (q *QuickSpy) RequiresWebsocket() bool {
	q.m.RLock()
	defer q.m.RUnlock()
	for _, focus := range q.Focuses.List() {
		if focus.UseWebsocket {
			return true
		}
	}
	return false
}

func (q *QuickSpy) Error() error {
	return fmt.Errorf("%q %q %q l: %s v: %s",
		q.Key.Exchange,
		q.Key.Asset,
		q.Key.Pair,
		q.Data.LastPrice,
		q.Data.Volume)
}

func (q *QuickSpy) RequiresAuth() bool {
	q.m.RLock()
	defer q.m.RUnlock()
	for _, focus := range q.Focuses.List() {
		if focus.Type == AccountHoldingsFocusType || focus.Type == OrdersFocusType || focus.Type == OrderPlacementFocusType {
			return true
		}
	}
	return false
}

func (q *QuickSpy) FocusRequiresWebsocket(focusType FocusType) bool {
	q.m.RLock()
	defer q.m.RUnlock()
	focus := q.Focuses.GetByKey(focusType)
	if focus == nil {
		return false
	}
	return focus.UseWebsocket
}

func (q *QuickSpy) GetAndWaitForFocusByKey(focusType FocusType) (*FocusData, error) {
	q.m.RLock()
	defer q.m.RUnlock()
	focus, err := q.GetFocusByKey(focusType)
	if err != nil {
		return nil, err
	}
	<-focus.HasBeenSuccessfulChan
	return focus, nil
}

func (q *QuickSpy) GetFocusByKey(focusType FocusType) (*FocusData, error) {
	focus := q.Focuses.GetByKey(focusType)
	if focus == nil {
		return nil, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	return focus, nil
}

func (q *QuickSpy) setupExchange(k Key) error {
	e, err := engine.NewSupportedExchangeByName(k.Exchange)
	if err != nil {
		return err
	}
	b := e.GetBase()
	e.SetDefaults()
	exchCfg, err := b.GetStandardConfig()
	if err != nil {
		return err
	}
	err = b.SetupDefaults(exchCfg)
	if err != nil {
		return err
	}
	err = e.Setup(exchCfg)
	if err != nil {
		return err
	}
	if q.RequiresWebsocket() && !e.SupportsWebsocket() {
		return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", k.Exchange)
	}
	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(e.GetRateLimits()))

	var rFmt, cFmt *currency.PairFormat
	if b.CurrencyPairs.UseGlobalFormat {
		rFmt = b.CurrencyPairs.RequestFormat
		cFmt = b.CurrencyPairs.ConfigFormat
	} else {
		rFmt = b.CurrencyPairs.Pairs[k.Asset].RequestFormat
		cFmt = b.CurrencyPairs.Pairs[k.Asset].ConfigFormat
	}
	b.CurrencyPairs.DisableAllPairs()
	b.CurrencyPairs.Pairs[k.Asset] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: rFmt,
		ConfigFormat:  cFmt,
	}
	err = b.CurrencyPairs.StorePairs(k.Asset, currency.Pairs{k.Pair}, false)
	if err != nil {
		return err
	}
	err = b.CurrencyPairs.StorePairs(k.Asset, currency.Pairs{k.Pair}, true)
	if err != nil {
		return err
	}
	if b.Features.Supports.Websocket && q.RequiresWebsocket() {
		b.Websocket.ToRoutine = q.dataHandlerChannel

		err = b.Websocket.Connect()
		if err != nil {
			return err
		}
		var newSubs []*subscription.Subscription
		if q.FocusRequiresWebsocket(TickerFocusType) {
			newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[TickerFocusType]})
		}
		if q.FocusRequiresWebsocket(TradesFocusType) {
			newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[TradesFocusType]})
		}
		if q.FocusRequiresWebsocket(OrderBookFocusType)  {
			newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[OrderBookFocusType]})
		}
		if q.FocusRequiresWebsocket(AccountHoldingsFocusType) {
			newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[AccountHoldingsFocusType]})
		}
		b.Config.Features.Subscriptions = newSubs
		if err := b.Websocket.Connect(); err != nil {
			return fmt.Errorf("failed to connect websocket for %s: %w", k.Exchange, err)
		}
	}
	q.Exch = e
	return nil
}

func (q *QuickSpy) Run() error {
	if q.RequiresWebsocket() {
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
					i+1, q.Key.Exchange, q.Key.Asset, q.Key.Pair, f.Type, err)
			}
		}(focus)
	}
	return nil
}

func (q *QuickSpy) HandleWS() error {
	for {
		select {
		case <-q.shutUP:
			return nil
		case d := <-q.dataHandlerChannel:
			switch data := d.(type) {
			case *ticker.Price:
				focus := q.Focuses.GetByKey(TickerFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",)
					continue
				}
				q.m.Lock()
				focus.Stream <- data
				q.Data.
				q.m.Unlock()
				focus.SetSuccessful()
			case *orderbook.Depth:
				focus := q.Focuses.GetByKey(OrderBookFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found",)
					continue
				}
				focus.m.RLock()
				if q.Data.OB != nil {
					q.m.RUnlock()
					continue
				}
				q.m.RUnlock()
				q.m.Lock()
				q.Data.OB = data
				q.m.Unlock()
				focus.SetSuccessful()
			case []trade.Data:
				focus := q.Focuses.GetByKey(TradesFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s not found", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), TradesFocusType)
					continue
				}
				q.m.Lock()
				q.Data.LastTradePrice = data[len(data)-1].Price)
				q.Data.LastTradeSize = data[len(data)-1].Amount)
				q.m.Unlock()
				focus.SetSuccessful()
			}
		}
	}
}


func (q *QuickSpy) RunRESTFocus(focusType FocusType) error {
	focus, ok := q.Focuses[focusType]
	if !ok {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	if !focus.Enabled {
		return fmt.Errorf("focus type %v is not enabled", focusType)
	}
	if focus.UseWebsocket {
		return nil
	}
	timer := time.NewTimer(0)
	failures := 0
	for {
		select {
		case <-q.shutUP:
			return nil
		case <-timer.C:
			err := q.handleFocusType(focusType, focus, timer)
			if err != nil {
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %q %q %q failed, focus type: %s err: %v",
					failures+1, q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focusType, err)
				if focus.IsOnceOff {
					return nil
				}
				if !focus.hasBeenSuccessful {
					if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
						q.successfulSpy(focus, timer)
						return nil
					}
					if failures == 5 {
						return fmt.Errorf("Quickspy data attempt: %v/5 %q %q %q failed, focus type: %s err: %v ", failures, q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focusType, err)
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
		err = q.handleURLFocus(focus, timer)
	case ContractFocusType:
		err = q.handleContractFocus(focus, timer)
	case HistoricalContractKlineFocusType:
		err = q.handleHistoricalContractKlineFocus(focus, timer)
	case KlineFocusType:
		err = q.handleKlineFocus(focus, timer)
	case OpenInterestFocusType:
		err = q.handleOpenInterestFocus(focus, timer)
	case TickerFocusType:
		err = q.handleTickerFocus(focus, timer)
	case OrdersFocusType:
		err = q.handleOrdersFocus(focus, timer)
	case AccountHoldingsFocusType:
		err = q.handleAccountHoldingsFocus(focus, timer)
	case OrderPlacementFocusType:
		// No implementation provided in the original code
	case OrderBookFocusType:
		err = q.handleOrderBookFocus(focus, timer)
	case TradesFocusType:
		err = q.handleTradesFocus(focus, timer)
	case OrderExecutionFocusType:
		err = q.handleOrderExecutionFocus(focus, timer)
	case FundingRateFocusType:
		err = q.handleFundingRateFocus(focus, timer)
	default:
		return fmt.Errorf("unknown focus type %v", focusType)
	}
	return err
}

func (q *QuickSpy) handleURLFocus(focus *FocusData, timer *time.Timer) error {
	resp, err := q.Exch.GetCurrencyTradeURL(q.credContext, q.Key.Asset, q.Key.Pair())
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	if resp == "" {
		return nil
	}
	focus.m.Lock()
	q.Data.Url = resp
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleContractFocus(focus *FocusData, timer *time.Timer) error {
	if !q.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	contracts, err := q.Exch.GetFuturesContractDetails(q.credContext, q.Key.Asset)
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	for i := range contracts {
		if !contracts[i].Name.Equal(q.Key.Pair()) {
			continue
		}
		focus.m.Lock()
		q.Data.UnderlyingBase = contracts[i].Underlying.Base.Item
		q.Data.UnderlyingQuote = contracts[i].Underlying.Quote.Item
		q.Data.ContractExpirationTime = contracts[i].EndDate
		q.Data.ContractType = contracts[i].Type
		q.Data.ContractValueDenomination = contracts[i].ContractValueDenomination
		q.Data.ContractSettlementDenomination = contracts[i].ContractSettlementDenomination
		if contracts[i].Multiplier > 0 {
			q.Data.ContractDecimals, _ = contracts[i].Multiplier)
		}

		focus.m.Unlock()
		break
	}
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleHistoricalContractKlineFocus(focus *FocusData, timer *time.Timer) error {
	err := q.WaitForInitialData(ContractFocusType)
	if err != nil {
		return fmt.Errorf("%w %s requires %s", err, HistoricalContractKlineFocusType, ContractFocusType)
	}
	if !q.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	isPerp, err := q.Exch.IsPerpetualFutureCurrency(q.Key.Asset, q.Key.Pair())
	if err != nil {
		log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s err: %v",
			q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
		q.successfulSpy(focus, timer)
		return nil
	}
	if isPerp {
		q.successfulSpy(focus, timer)
		return nil
	}
	at := q.Exch.GetAssetTypes(true)
	if !at.Contains(asset.Spot) {
		b := q.Exch.GetBase()
		err = b.CurrencyPairs.SetAssetEnabled(asset.Spot, true)
		if err != nil {
			return err
		}
	}
	k, err := q.Exch.GetHistoricalContractKlineData(q.credContext, &futures.GetKlineContractRequest{
		ContractPair:   q.Key.Pair(),
		UnderlyingPair: currency.Pair{Base: q.Data.UnderlyingBase.Currency(), Quote: q.Data.UnderlyingQuote.Currency()},
		Asset:          q.Key.Asset,
		StartDate:      time.Now().Add(-time.Hour * 24 * 200),
		EndDate:        time.Now(),
		Interval:       kline.OneHour,
		Contract:       q.Data.ContractType,
	})
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			q.successfulSpy(focus, timer)
			return nil
		}
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	if len(k.Data) == 0 {
		return nil
	}
	k.Analyse()
	focus.m.Lock()
	q.Data.HistoricalKlines = *k
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleKlineFocus(focus *FocusData, timer *time.Timer) error {
	tt := time.Now().Add(-kline.ThreeMonth.Duration())
	k, err := q.Exch.GetHistoricCandlesExtended(q.credContext, q.Key.Pair(), q.Key.Asset, kline.OneHour, tt, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			k, err = q.Exch.GetHistoricCandles(q.credContext, q.Key.Pair(), q.Key.Asset, kline.OneHour, tt, time.Now())
		}
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
		}
	}
	if len(k.Candles) == 0 {
		return nil
	}
	focus.m.Lock()
	q.Data.Klines = k.Candles
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleOpenInterestFocus(focus *FocusData, timer *time.Timer) error {
	if !q.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	if moi, err := memstore.GetOpenInterest(q.KeyNoCreds()); moi != nil && err == nil && time.Since(moi.LastUpdated) < focus.RESTPollTime {
		focus.m.Lock()
		q.Data.OpenInterest, err = moi.OpenInterest)
		focus.m.Unlock()
		if err != nil {
			return err
		}
		q.successfulSpy(focus, timer)
		return nil
	}
	oi, err := q.Exch.GetOpenInterest(q.credContext, key.PairAsset{
		Base:  q.Key.Base,
		Quote: q.Key.Quote,
		Asset: q.Key.Asset,
	})
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	if len(oi) != 1 {
		return nil
	}
	focus.m.Lock()
	q.Data.OpenInterest, err = oi[0].OpenInterest)
	focus.m.Unlock()
	if err != nil {
		return err
	}
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleTickerFocus(focus *FocusData, timer *time.Timer) error {
	tick, err := memstore.GetTicker(q.KeyNoCreds())
	if err != nil || time.Since(tick.LastUpdated) > focus.RESTPollTime {
		tick, err = ticker.GetTicker(q.Key.Exchange, q.Key.Pair(), q.Key.Asset)
		if err != nil || time.Since(tick.LastUpdated) > focus.RESTPollTime {
			tick, err = q.Exch.UpdateTicker(q.credContext, q.Key.Pair(), q.Key.Asset)
			if err != nil {
				return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
			}
		}
	}
	focus.m.Lock()
	q.Data.LastPrice, _ = tick.Last)
	q.Data.MarkPrice, _ = tick.MarkPrice)
	q.Data.Volume, _ = tick.Volume)
	q.Data.QuoteVolume, _ = tick.QuoteVolume)
	q.Data.IndexPrice, _ = tick.IndexPrice)
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleOrdersFocus(focus *FocusData, timer *time.Timer) error {
	resp, err := q.Exch.GetActiveOrders(q.credContext, &order.MultiOrderRequest{
		Pairs:     []currency.Pair{q.Key.Pair()},
		AssetType: q.Key.Asset,
	})
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Orders = resp
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleAccountHoldingsFocus(focus *FocusData, timer *time.Timer) error {
	ais := accounts.AccountManager.GetByCredentials(q.Key.Credentials)
	focus.m.Lock()
	q.Data.Holdings = ais
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleOrderBookFocus(focus *FocusData, timer *time.Timer) error {
	ob, err := orderbook.Get(q.Key.Exchange, q.Key.Pair(), q.Key.Asset)
	if err != nil || time.Since(ob.LastUpdated) > focus.RESTPollTime {
		ob, err = q.Exch.UpdateOrderbook(q.credContext, q.Key.Pair(), q.Key.Asset)
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
		}
	}
	focus.m.Lock()
	q.Data.OB, err = ob.GetDepth()
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	err = q.unSafeOBDataSet()
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleTradesFocus(focus *FocusData, timer *time.Timer) error {
	tr, err := q.Exch.GetRecentTrades(q.credContext, q.Key.Pair(), q.Key.Asset)
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.LastTradeSize, _ = tr[len(tr)-1].Amount)
	q.Data.LastTradePrice, _ = tr[len(tr)-1].Price)
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleOrderExecutionFocus(focus *FocusData, timer *time.Timer) error {
	el, err := q.Exch.GetOrderExecutionLimits(q.Key.Asset, q.Key.Pair())
	if err != nil {
		err = q.Exch.UpdateOrderExecutionLimits(q.credContext, q.Key.Asset)
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
		}
		el, err = q.Exch.GetOrderExecutionLimits(q.Key.Asset, q.Key.Pair())
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
		}
	}
	focus.m.Lock()
	q.Data.ExecutionLimits = el
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleFundingRateFocus(focus *FocusData, timer *time.Timer) error {
	if !q.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	if !focus.hasBeenSuccessful {
		isPerp, err := q.Exch.IsPerpetualFutureCurrency(q.Key.Asset, q.Key.Pair())
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
		}
		if !isPerp {
			q.successfulSpy(focus, timer)
			return nil
		}
	}
	if moi, err := memstore.GetFundingRate(q.Key.Exchange, q.Key.Pair(), q.Key.Asset); moi != nil && err == nil && time.Since(moi.TimeChecked) < focus.RESTPollTime {
		focus.m.Lock()
		q.Data.FundingRate, _ = moi.LatestRate.Rate.InexactFloat64())
		q.Data.CurrentFundingRateTime = moi.LatestRate.Time
		q.Data.EstimatedFundingRate, _ = moi.PredictedUpcomingRate.Rate.InexactFloat64())
		q.Data.NextFundingRateTime = moi.PredictedUpcomingRate.Time
		if q.Data.NextFundingRateTime.IsZero() {
			q.Data.NextFundingRateTime = moi.TimeOfNextRate
		}
		focus.m.Unlock()
		q.successfulSpy(focus, timer)
		return nil
	}
	fr, err := q.Exch.GetLatestFundingRates(q.credContext, &fundingrate.LatestRateRequest{
		Asset:                q.Key.Asset,
		Pair:                 q.Key.Pair(),
		IncludePredictedRate: true,
	})
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String(), err)
	}
	if len(fr) != 1 {
		log.Errorf(log.QuickSpy, "Quickspy data attempt: %q %q %q failed, focus type: %s funding rate length not 1",
			q.Key.Exchange, q.Key.Asset, q.Key.Pair(), focus.Type.String())
		return nil
	}
	focus.m.Lock()
	q.Data.FundingRate, _ = fr[0].LatestRate.Rate.InexactFloat64())
	q.Data.CurrentFundingRateTime = fr[0].LatestRate.Time
	q.Data.EstimatedFundingRate, _ = fr[0].PredictedUpcomingRate.Rate.InexactFloat64())
	q.Data.NextFundingRateTime = fr[0].PredictedUpcomingRate.Time
	if q.Data.NextFundingRateTime.IsZero() {
		q.Data.NextFundingRateTime = fr[0].TimeOfNextRate
	}
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
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

func (q *QuickSpy) unSafeOBDataSet() error {
	var err error
	var (
		spread, spreadPercent, bidLiquidity, bidValue, askLiquidity, askValue float64
	)
	q.Data.Bids, q.Data.Asks, err = q.Data.OB.GetTranches(10)
	if err != nil {
		return err
	}
	spread, err = q.Data.OB.GetSpreadAmount()
	if err != nil {
		return err
	}
	q.Data.Spread, _ = spread)

	spreadPercent, err = q.Data.OB.GetSpreadPercentage()
	if err != nil {
		return err
	}
	q.Data.SpreadPercent, _ = spreadPercent)

	bidLiquidity, bidValue, err = q.Data.OB.TotalBidAmounts()
	if err != nil {
		return err
	}
	q.Data.BidLiquidity, _ = bidLiquidity)
	q.Data.BidValue, _ = bidValue)

	askLiquidity, askLiquidity, err = q.Data.OB.TotalAskAmounts()
	if err != nil {
		return err
	}
	q.Data.AskLiquidity, _ = askLiquidity)
	q.Data.AskValue, _ = askValue)
	return nil
}

func (q *QuickSpy) Shutdown() {
	close(q.shutUP)
}

func (f *FocusData) SetSuccessful() {
	f.m.RLock()
	if f.hasBeenSuccessful {
		f.m.RUnlock()
		return
	}
	f.m.RUnlock()
	f.m.Lock()
	if f.hasBeenSuccessful {
		f.m.Unlock()
		return
	}
	f.hasBeenSuccessful = true
	close(f.HasBeenSuccessfulChan)
	f.m.Unlock()
}

func (q *QuickSpy) Dump() (*ExportedData, error) {
	q.m.RLock()
	defer q.m.RUnlock()
	return q.Data.Dump(key.ExchangePairAsset{
		Exchange: q.Key.Exchange,
		Asset:    q.Key.Asset,
		Base:     q.Key.Pair.Base.Item,
		Quote:    q.Key.Pair.Quote.Item,
	}, !q.Key.Credentials.IsEmpty())
}

func (d *Data) createOrderbookEntries(t orderbook.Levels) ([]OrderBookEntry, error) {
	entries := make([]OrderBookEntry, len(t))
	for i := range t {
		entries[i] = OrderBookEntry{
			Price:           t[i].Price,
			Amount:           t[i].Amount,
			OrderAmount:      t[i].OrderCount,
			ContractDecimals: d.ContractDecimals,
			Total:            t[i].Price * t[i].Amount,
		}
	}
	return entries, nil
}

func (d *Data) Dump(k key.ExchangePairAsset, hasCredentials bool) (*ExportedData, error) {
	d.Asks.SortAsks()
	asks, err := d.createOrderbookEntries(d.Asks)
	if err != nil {
		return nil, err
	}
	d.Bids.SortBids()
	bids, err := d.createOrderbookEntries(d.Asks)
	if err != nil {
		return nil, err
	}
	resp := &ExportedData{
		Key:                    k,
		UnderlyingBase:         d.UnderlyingBase,
		UnderlyingQuote:        d.UnderlyingQuote,
		ContractExpirationTime: d.ContractExpirationTime,
		ContractType:           d.ContractType.String(),
		ContractDecimals:       d.ContractDecimals,
		HasValidCredentials:    hasCredentials,
		OpenInterest:           d.OpenInterest,
		LastPrice:              d.LastPrice,
		IndexPrice:             d.IndexPrice,
		MarkPrice:              d.MarkPrice,
		Volume:                 d.Volume,
		Bids:                   bids,
		Asks:                   asks,
		AskLiquidity:           d.AskLiquidity,
		AskValue:               d.AskValue,
		BidLiquidity:           d.BidLiquidity,
		BidValue:               d.BidValue,
		Spread:                 d.Spread,
		SpreadPercent:          d.SpreadPercent,
		FundingRate:            d.FundingRate,
		NextFundingRateTime:    d.NextFundingRateTime,
		CurrentFundingRateTime: d.CurrentFundingRateTime,
		EstimatedFundingRate:   d.EstimatedFundingRate,
		LastTradePrice:         d.LastTradePrice,
		LastTradeSize:          d.LastTradeSize,
		Holdings:               d.Holdings,
		Orders:                 d.Orders,
		ExecutionLimits:        d.ExecutionLimits,
		URL:                    d.Url,
	}
	return resp, nil
}

func (q *QuickSpy) WaitForInitialData(focusType FocusType) error {
	focus, _, err := q.Focuses.GetByKey(focusType)
	if err != nil {
		return err
	}
	<-focus.HasBeenSuccessfulChan
	return nil
}
