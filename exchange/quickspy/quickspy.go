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
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

var focusToSub = map[FocusType]string{
	OrderBookFocusType: subscription.OrderbookChannel,
	TickerFocusType:    subscription.TickerChannel,
	KlineFocusType:     subscription.CandlesChannel,
}

// NewQuickSpy creates a new QuickSpy
func NewQuickSpy(k CredKey, focuses []FocusData) (*QuickSpy, error) {
	k.Exchange = strings.ToLower(k.Exchange)
	sm := slicemap.NewSliceMap[FocusType, FocusData]()
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
		focus := focuses[i]
		focus.m = &sync.RWMutex{}
		focus.HasBeenSuccessfulChan = make(chan any)
		sm.Upsert(focuses[i].Type, &focus)
	}
	credContext := context.Background()
	q := &QuickSpy{
		CredKey:            k,
		dataHandlerChannel: make(chan any),
		shutUP:             make(chan any),
		Focuses:            sm,
		credContext:        credContext,
		Data:               Data{Key: k},
		RWMutex:            &sync.RWMutex{},
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
	focuses := q.Focuses.CopyList()
	for i := range focuses {
		if focuses[i].UseWebsocket {
			return true
		}
	}
	return false
}

func (q *QuickSpy) Error() error {
	if q.CredKey.Key.Asset.IsFutures() {
		return fmt.Errorf("%s %s %s denom: %s val: %s l: %s v: %s",
			q.CredKey.Key.Exchange,
			q.CredKey.Key.Asset,
			q.CredKey.Pair(),
			q.Data.ContractSettlementDenomination.String(),
			q.Data.ContractValueDenomination.String(),
			q.Data.LastPrice,
			q.Data.Volume)
	}
	return fmt.Errorf("%s %s %s l: %s v: %s",
		q.CredKey.Key.Exchange,
		q.CredKey.Key.Asset,
		q.CredKey.Pair(),
		q.Data.LastPrice,
		q.Data.Volume)
}

func (q *QuickSpy) RequiresAuth() bool {
	focuses := q.Focuses.CopyList()
	for i := range focuses {
		if focuses[i].Type == AccountHoldingsFocusType || focuses[i].Type == OrdersFocusType || focuses[i].Type == OrderPlacementFocusType {
			return true
		}
	}
	return false
}

func (q *QuickSpy) FocusRequiresWebsocket(focusType FocusType) bool {
	focus, _, err := q.Focuses.GetByKey(focusType)
	if err != nil {
		return false
	}
	return focus.UseWebsocket
}

func (q *QuickSpy) GetAndWaitForFocusByKey(focusType FocusType) (*FocusData, error) {
	focus, err := q.GetFocusByKey(focusType)
	if err != nil {
		return nil, err
	}
	<-focus.HasBeenSuccessfulChan
	return focus, nil
}

func (q *QuickSpy) GetFocusByKey(focusType FocusType) (*FocusData, error) {
	focus, _, err := q.Focuses.GetByKey(focusType)
	if err != nil {
		return nil, err
	}
	return focus, nil
}

func (q *QuickSpy) setupExchange(k types.CredKey) error {
	e, err := engine.NewSupportedExchangeByName(k.Key.Exchange)
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
		return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", k.Key.Exchange)
	}
	ratelimit.DoForAnExchange(e)

	var rFmt, cFmt *currency.PairFormat
	if b.CurrencyPairs.UseGlobalFormat {
		rFmt = b.CurrencyPairs.RequestFormat
		cFmt = b.CurrencyPairs.ConfigFormat
	} else {
		rFmt = b.CurrencyPairs.Pairs[k.Key.Asset].RequestFormat
		cFmt = b.CurrencyPairs.Pairs[k.Key.Asset].ConfigFormat
	}
	b.CurrencyPairs.DisableAllPairs()
	cp := currency.Pair{Quote: k.Key.Quote.Currency(), Base: k.Key.Base.Currency()}
	b.CurrencyPairs.Pairs[k.Key.Asset] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: rFmt,
		ConfigFormat:  cFmt,
	}
	err = b.CurrencyPairs.StorePairs(k.Key.Asset, currency.Pairs{cp}, false)
	if err != nil {
		return err
	}
	err = b.CurrencyPairs.StorePairs(k.Key.Asset, currency.Pairs{cp}, true)
	if err != nil {
		return err
	}
	if b.Features.Supports.Websocket && q.RequiresWebsocket() {
		b.Websocket.ToRoutine = q.dataHandlerChannel
		err = b.Websocket.EnableAndConnectNoSubs()
		if err != nil {
			return err
		}
		var newSubs []*subscription.Subscription
		if q.FocusRequiresWebsocket(TickerFocusType) {
			newSubs = append(newSubs, &subscription.Subscription{Channel: focusToSub[TickerFocusType]})
		}
		if q.FocusRequiresWebsocket(TradesFocusType) && strings.Contains(subs[i].Channel, "trade") {
			newSubs = append(newSubs, subs[i])
		}
		if q.FocusRequiresWebsocket(OrderBookFocusType) && common.StringSliceCompareInsensitive(bookNames, subs[i].Channel) {
			newSubs = append(newSubs, subs[i])
		}
		if q.FocusRequiresWebsocket(AccountHoldingsFocusType) && strings.Contains(subs[i].Channel, "account") {
			newSubs = append(newSubs, subs[i])
		}
		b.GetSubscriptionTemplate(&subscription.Subscription{Channel: })
		if len(newSubs) > 0 {
			err = b.Websocket.Subscriber(newSubs)
			if err != nil {
				return err
			}
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
	focuses := q.Focuses.CopyList()
	for i := range focuses {
		if focuses[i].UseWebsocket {
			continue
		}
		q.wg.Add(1)
		go func(f FocusData) {
			defer q.wg.Done()
			err := q.RunRESTFocus(f.Type)
			if err != nil {
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %s %s %s failed, focus type: %s err: %v",
					i+1, q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), f.Type.String(), err)
			}
		}(focuses[i])
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
				focus, _, err := q.Focuses.GetByKey(TickerFocusType)
				if err != nil {
					return err
				}
				focus.m.Lock()
				q.Data.LastPrice, _ = udecimal.NewFromFloat64(data.Last)
				q.Data.MarkPrice, _ = udecimal.NewFromFloat64(data.MarkPrice)
				q.Data.IndexPrice, _ = udecimal.NewFromFloat64(data.IndexPrice)
				q.Data.Volume, _ = udecimal.NewFromFloat64(data.Volume)
				q.Data.QuoteVolume, _ = udecimal.NewFromFloat64(data.QuoteVolume)
				focus.m.Unlock()
				focus.SetSuccessful()
			case *orderbook.Depth:
				focus, _, err := q.Focuses.GetByKey(OrderBookFocusType)
				if err != nil {
					return err
				}
				focus.m.RLock()
				if q.Data.OB != nil {
					q.RWMutex.RUnlock()
					continue
				}
				q.RWMutex.RUnlock()
				q.RWMutex.Lock()
				q.Data.OB = data
				q.RWMutex.Unlock()
				focus.SetSuccessful()
			case []trade.Data:
				focus, _, err := q.Focuses.GetByKey(TradesFocusType)
				if err != nil {
					return err
				}
				focus.m.Lock()
				q.Data.LastTradePrice, _ = udecimal.NewFromFloat64(data[len(data)-1].Price)
				q.Data.LastTradeSize, _ = udecimal.NewFromFloat64(data[len(data)-1].Amount)
				focus.m.Unlock()
				focus.SetSuccessful()
			}
		}
	}
}

func (q *QuickSpy) RunRESTFocus(focusType FocusType) error {
	focus, _, err := q.Focuses.GetByKey(focusType)
	if err != nil {
		return err
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
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %s %s %s failed, focus type: %s err: %v",
					failures+1, q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focusType, err)
				if focus.IsOnceOff {
					return nil
				}
				if !focus.hasBeenSuccessful {
					if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
						q.successfulSpy(focus, timer)
						return nil
					}
					if failures == 5 {
						return fmt.Errorf("Quickspy data attempt: %v/5 %s %s %s failed, focus type: %s err: %v ", failures, q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focusType, err)
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
	resp, err := q.Exch.GetCurrencyTradeURL(q.credContext, q.CredKey.Key.Asset, q.CredKey.Pair())
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
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
	if !q.CredKey.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	contracts, err := q.Exch.GetFuturesContractDetails(q.credContext, q.CredKey.Key.Asset)
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
	}
	for i := range contracts {
		if !contracts[i].Name.Equal(q.CredKey.Pair()) {
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
			q.Data.ContractDecimals, _ = udecimal.NewFromFloat64(contracts[i].Multiplier)
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
	if !q.CredKey.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	isPerp, err := q.Exch.IsPerpetualFutureCurrency(q.CredKey.Key.Asset, q.CredKey.Pair())
	if err != nil {
		log.Errorf(log.QuickSpy, "Quickspy data attempt: %s %s %s failed, focus type: %s err: %v",
			q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
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
		ContractPair:   q.CredKey.Pair(),
		UnderlyingPair: currency.Pair{Base: q.Data.UnderlyingBase.Currency(), Quote: q.Data.UnderlyingQuote.Currency()},
		Asset:          q.CredKey.Key.Asset,
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
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
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
	k, err := q.Exch.GetHistoricCandlesExtended(q.credContext, q.CredKey.Pair(), q.CredKey.Key.Asset, kline.OneHour, tt, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			k, err = q.Exch.GetHistoricCandles(q.credContext, q.CredKey.Pair(), q.CredKey.Key.Asset, kline.OneHour, tt, time.Now())
		}
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
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
	if !q.CredKey.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	if moi, err := memstore.GetOpenInterest(q.CredKey.KeyNoCreds()); moi != nil && err == nil && time.Since(moi.LastUpdated) < focus.RESTPollTime {
		focus.m.Lock()
		q.Data.OpenInterest, err = udecimal.NewFromFloat64(moi.OpenInterest)
		focus.m.Unlock()
		if err != nil {
			return err
		}
		q.successfulSpy(focus, timer)
		return nil
	}
	oi, err := q.Exch.GetOpenInterest(q.credContext, key.PairAsset{
		Base:  q.CredKey.Key.Base,
		Quote: q.CredKey.Key.Quote,
		Asset: q.CredKey.Key.Asset,
	})
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
	}
	if len(oi) != 1 {
		return nil
	}
	focus.m.Lock()
	q.Data.OpenInterest, err = udecimal.NewFromFloat64(oi[0].OpenInterest)
	focus.m.Unlock()
	if err != nil {
		return err
	}
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleTickerFocus(focus *FocusData, timer *time.Timer) error {
	tick, err := memstore.GetTicker(q.CredKey.KeyNoCreds())
	if err != nil || time.Since(tick.LastUpdated) > focus.RESTPollTime {
		tick, err = ticker.GetTicker(q.CredKey.Key.Exchange, q.CredKey.Pair(), q.CredKey.Key.Asset)
		if err != nil || time.Since(tick.LastUpdated) > focus.RESTPollTime {
			tick, err = q.Exch.UpdateTicker(q.credContext, q.CredKey.Pair(), q.CredKey.Key.Asset)
			if err != nil {
				return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
			}
		}
	}
	focus.m.Lock()
	q.Data.LastPrice, _ = udecimal.NewFromFloat64(tick.Last)
	q.Data.MarkPrice, _ = udecimal.NewFromFloat64(tick.MarkPrice)
	q.Data.Volume, _ = udecimal.NewFromFloat64(tick.Volume)
	q.Data.QuoteVolume, _ = udecimal.NewFromFloat64(tick.QuoteVolume)
	q.Data.IndexPrice, _ = udecimal.NewFromFloat64(tick.IndexPrice)
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleOrdersFocus(focus *FocusData, timer *time.Timer) error {
	resp, err := q.Exch.GetActiveOrders(q.credContext, &order.MultiOrderRequest{
		Pairs:     []currency.Pair{q.CredKey.Pair()},
		AssetType: q.CredKey.Key.Asset,
	})
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Orders = resp
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleAccountHoldingsFocus(focus *FocusData, timer *time.Timer) error {
	ais := accounts.AccountManager.GetByCredentials(q.CredKey.Credentials)
	focus.m.Lock()
	q.Data.Holdings = ais
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleOrderBookFocus(focus *FocusData, timer *time.Timer) error {
	ob, err := orderbook.Get(q.CredKey.Key.Exchange, q.CredKey.Pair(), q.CredKey.Key.Asset)
	if err != nil || time.Since(ob.LastUpdated) > focus.RESTPollTime {
		ob, err = q.Exch.UpdateOrderbook(q.credContext, q.CredKey.Pair(), q.CredKey.Key.Asset)
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
		}
	}
	focus.m.Lock()
	q.Data.OB, err = ob.GetDepth()
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
	}
	err = q.unSafeOBDataSet()
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
	}
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleTradesFocus(focus *FocusData, timer *time.Timer) error {
	tr, err := q.Exch.GetRecentTrades(q.credContext, q.CredKey.Pair(), q.CredKey.Key.Asset)
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.LastTradeSize, _ = udecimal.NewFromFloat64(tr[len(tr)-1].Amount)
	q.Data.LastTradePrice, _ = udecimal.NewFromFloat64(tr[len(tr)-1].Price)
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleOrderExecutionFocus(focus *FocusData, timer *time.Timer) error {
	el, err := q.Exch.GetOrderExecutionLimits(q.CredKey.Key.Asset, q.CredKey.Pair())
	if err != nil {
		err = q.Exch.UpdateOrderExecutionLimits(q.credContext, q.CredKey.Key.Asset)
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
		}
		el, err = q.Exch.GetOrderExecutionLimits(q.CredKey.Key.Asset, q.CredKey.Pair())
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
		}
	}
	focus.m.Lock()
	q.Data.ExecutionLimits = el
	focus.m.Unlock()
	q.successfulSpy(focus, timer)
	return nil
}

func (q *QuickSpy) handleFundingRateFocus(focus *FocusData, timer *time.Timer) error {
	if !q.CredKey.Key.Asset.IsFutures() {
		q.successfulSpy(focus, timer)
		return nil
	}
	if !focus.hasBeenSuccessful {
		isPerp, err := q.Exch.IsPerpetualFutureCurrency(q.CredKey.Key.Asset, q.CredKey.Pair())
		if err != nil {
			return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
		}
		if !isPerp {
			q.successfulSpy(focus, timer)
			return nil
		}
	}
	if moi, err := memstore.GetFundingRate(q.CredKey.Key.Exchange, q.CredKey.Pair(), q.CredKey.Key.Asset); moi != nil && err == nil && time.Since(moi.TimeChecked) < focus.RESTPollTime {
		focus.m.Lock()
		q.Data.FundingRate, _ = udecimal.NewFromFloat64(moi.LatestRate.Rate.InexactFloat64())
		q.Data.CurrentFundingRateTime = moi.LatestRate.Time
		q.Data.EstimatedFundingRate, _ = udecimal.NewFromFloat64(moi.PredictedUpcomingRate.Rate.InexactFloat64())
		q.Data.NextFundingRateTime = moi.PredictedUpcomingRate.Time
		if q.Data.NextFundingRateTime.IsZero() {
			q.Data.NextFundingRateTime = moi.TimeOfNextRate
		}
		focus.m.Unlock()
		q.successfulSpy(focus, timer)
		return nil
	}
	fr, err := q.Exch.GetLatestFundingRates(q.credContext, &fundingrate.LatestRateRequest{
		Asset:                q.CredKey.Key.Asset,
		Pair:                 q.CredKey.Pair(),
		IncludePredictedRate: true,
	})
	if err != nil {
		return fmt.Errorf("%v %v %v %v %w", q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String(), err)
	}
	if len(fr) != 1 {
		log.Errorf(log.QuickSpy, "Quickspy data attempt: %s %s %s failed, focus type: %s funding rate length not 1",
			q.CredKey.Key.Exchange, q.CredKey.Key.Asset, q.CredKey.Pair(), focus.Type.String())
		return nil
	}
	focus.m.Lock()
	q.Data.FundingRate, _ = udecimal.NewFromFloat64(fr[0].LatestRate.Rate.InexactFloat64())
	q.Data.CurrentFundingRateTime = fr[0].LatestRate.Time
	q.Data.EstimatedFundingRate, _ = udecimal.NewFromFloat64(fr[0].PredictedUpcomingRate.Rate.InexactFloat64())
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
	q.Data.Spread, _ = udecimal.NewFromFloat64(spread)

	spreadPercent, err = q.Data.OB.GetSpreadPercentage()
	if err != nil {
		return err
	}
	q.Data.SpreadPercent, _ = udecimal.NewFromFloat64(spreadPercent)

	bidLiquidity, bidValue, err = q.Data.OB.TotalBidAmounts()
	if err != nil {
		return err
	}
	q.Data.BidLiquidity, _ = udecimal.NewFromFloat64(bidLiquidity)
	q.Data.BidValue, _ = udecimal.NewFromFloat64(bidValue)

	askLiquidity, askLiquidity, err = q.Data.OB.TotalAskAmounts()
	if err != nil {
		return err
	}
	q.Data.AskLiquidity, _ = udecimal.NewFromFloat64(askLiquidity)
	q.Data.AskValue, _ = udecimal.NewFromFloat64(askValue)
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
	q.RWMutex.RLock()
	defer q.RWMutex.RUnlock()
	return q.Data.Dump(key.ExchangePairAsset{
		Exchange: q.CredKey.Key.Exchange,
		Asset:    q.CredKey.Key.Asset,
		Base:     q.CredKey.Key.Base,
		Quote:    q.CredKey.Key.Quote,
	}, !q.CredKey.Credentials.IsEmpty())
}

func (d *Data) createOrderbookEntries(t orderbook.Tranches) ([]OrderBookEntry, error) {
	entries := make([]OrderBookEntry, len(t))
	for i := range t {
		p, err := udecimal.NewFromFloat64(t[i].Price)
		if err != nil {
			return nil, err
		}
		a, err := udecimal.NewFromFloat64(t[i].Amount)
		if err != nil {
			return nil, err
		}
		total := p.Mul(a)
		entries[i] = OrderBookEntry{
			Price:            p,
			Amount:           a,
			OrderAmount:      t[i].OrderCount,
			ContractDecimals: d.ContractDecimals,
			Total:            total,
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
