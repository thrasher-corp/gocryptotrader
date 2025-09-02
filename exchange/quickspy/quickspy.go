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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

// NewQuickSpy creates a new QuickSpy
func NewQuickSpy(ctx context.Context, k *CredentialsKey, focuses []FocusData, verbose bool) (*QuickSpy, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	q := &QuickSpy{
		Key:                k,
		dataHandlerChannel: make(chan any),
		Focuses:            sm,
		credContext:        ctx,
		Data:               &Data{Key: k},
		m:                  new(sync.RWMutex),
		verbose:            verbose,
	}
	err := q.setupExchange()
	if err != nil {
		return nil, err
	}
	if q.AnyRequiresAuth() {
		if k.Credentials.IsEmpty() {
			return nil, fmt.Errorf("%w for %s", errNoCredentials, k.ExchangeAssetPair)
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

	e, err := engine.NewSupportedExchangeByName(q.Key.ExchangeAssetPair.Exchange)
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
		if !errors.Is(err, errNoSubSwitchingToREST) {
			return err
		}
		log.Warnf(log.QuickSpy, "%s websocket setup failed: %v. Disabling websocket focuses", q.Key.ExchangeAssetPair, err)
		q.Focuses.DisableWebsocketFocuses()
	}
	q.Exch = e
	return nil
}

func (q *QuickSpy) setupExchangeDefaults(e exchange.IBotExchange, b *exchange.Base) error {
	e.SetDefaults()
	exchCfg, err := b.GetStandardConfig()
	if err != nil {
		return fmt.Errorf("%s: %w", q.Key.ExchangeAssetPair, err)
	}
	exchCfg.Verbose = q.verbose
	if err := b.SetupDefaults(exchCfg); err != nil {
		return fmt.Errorf("%s: %w", q.Key.ExchangeAssetPair, err)
	}
	b.Verbose = q.verbose
	if err := e.Setup(exchCfg); err != nil {
		return fmt.Errorf("%s: %w", q.Key.ExchangeAssetPair, err)
	}
	return nil
}

func (q *QuickSpy) setupCurrencyPairs(b *exchange.Base) error {
	var rFmt, cFmt *currency.PairFormat
	if b.CurrencyPairs.UseGlobalFormat {
		rFmt = b.CurrencyPairs.RequestFormat
		cFmt = b.CurrencyPairs.ConfigFormat
	} else {
		rFmt = b.CurrencyPairs.Pairs[q.Key.ExchangeAssetPair.Asset].RequestFormat
		cFmt = b.CurrencyPairs.Pairs[q.Key.ExchangeAssetPair.Asset].ConfigFormat
	}
	b.CurrencyPairs.DisableAllPairs()
	// no formatting occurs for websocket subscription generation
	// so do it here to cover for it
	cFmtPair := q.Key.ExchangeAssetPair.Pair().Format(*cFmt)
	b.CurrencyPairs.Pairs[q.Key.ExchangeAssetPair.Asset] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: rFmt,
		ConfigFormat:  cFmt,
	}

	if err := b.CurrencyPairs.StorePairs(q.Key.ExchangeAssetPair.Asset, currency.Pairs{cFmtPair}, false); err != nil {
		return err
	}
	if err := b.CurrencyPairs.StorePairs(q.Key.ExchangeAssetPair.Asset, currency.Pairs{cFmtPair}, true); err != nil {
		return err
	}
	return nil
}

func (q *QuickSpy) checkRateLimits(b *exchange.Base) error {
	if len(b.GetRateLimiterDefinitions()) == 0 {
		return fmt.Errorf("%s %w", q.Key.ExchangeAssetPair, errNoRateLimits)
	}
	return nil
}

func (q *QuickSpy) setupWebsocket(e exchange.IBotExchange, b *exchange.Base) error {
	if q.AnyRequiresWebsocket() {
		if !e.SupportsWebsocket() {
			return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.Key.ExchangeAssetPair.Exchange)
		}
	} else {
		return nil
	}

	if !b.Features.Supports.Websocket {
		return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.Key.ExchangeAssetPair.Exchange)
	}
	if err := common.NilGuard(b.Websocket); err != nil {
		return fmt.Errorf("%s %w", q.Key.ExchangeAssetPair, err)
	}
	// allows routing of all websocket data to our custom one
	b.Websocket.ToRoutine = q.dataHandlerChannel
	var newSubs []*subscription.Subscription
	for _, f := range q.Focuses.List() {
		if !f.RequiresWebsocket() {
			continue
		}
		ch, ok := focusToSub[f.Type]
		if !ok || ch == "" {
			return fmt.Errorf("%s %s %w", q.Key.ExchangeAssetPair, f.Type, errNoWebsocketSupportForFocusType)
		}
		var sub *subscription.Subscription
		for _, s := range b.Config.Features.Subscriptions {
			if s.Channel != ch {
				continue
			}
			if s.Asset != q.Key.ExchangeAssetPair.Asset &&
				s.Asset != asset.All && s.Asset != asset.Empty {
				continue
			}
			sub = s
		}
		if sub == nil {
			return fmt.Errorf("%s %s %w", q.Key.ExchangeAssetPair, f.Type, errNoSubSwitchingToREST)
		}
		s := sub.Clone()
		rFmtPair := q.Key.ExchangeAssetPair.Pair().Format(*b.CurrencyPairs.Pairs[q.Key.ExchangeAssetPair.Asset].RequestFormat)
		s.Pairs.Add(rFmtPair)
		newSubs = append(newSubs, s)
	}
	b.Config.Features.Subscriptions = newSubs
	b.Features.Subscriptions = newSubs
	if err := b.Websocket.EnableAndConnect(); err != nil {
		if !errors.Is(err, websocket.ErrWebsocketAlreadyEnabled) {
			return fmt.Errorf("%s: %w", q.Key.ExchangeAssetPair, err)
		}
		// Because EnableAndConnect errors if its already enabled, but also wants to connect
		// we have to do this silly handling, making everyone suffer
		// the complaint was generated by AI, there must be a lot of bitching in scraped comments
		err = b.Websocket.Connect()
		if err != nil {
			return fmt.Errorf("%s: %w", q.Key.ExchangeAssetPair, err)
		}
	}
	return nil
}

func (q *QuickSpy) Run() error {
	if q.AnyRequiresWebsocket() {
		q.wg.Go(func() {
			err := q.HandleWS()
			if err != nil {
				panic(err)
			}
		})
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
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %s failed, focus type: %q err: %v",
					i+1, q.Key.ExchangeAssetPair, f.Type, err)
			}
		}(focus)
	}
	return nil
}

func (q *QuickSpy) HandleWS() error {
	for {
		select {
		case <-q.credContext.Done():
			return q.credContext.Err()
		case d := <-q.dataHandlerChannel:
			switch data := d.(type) {
			case account.Change:
				focus := q.Focuses.GetByFocusType(AccountHoldingsFocusType)
				if focus == nil {
					// these should never happen, but I've had Stranger Things happen
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, AccountHoldingsFocusType)
					continue
				}
				if data.AssetType != q.Key.ExchangeAssetPair.Asset &&
					!data.Balance.Currency.Equal(q.Key.ExchangeAssetPair.Pair().Base) &&
					!data.Balance.Currency.Equal(q.Key.ExchangeAssetPair.Pair().Quote) {
					continue
				}
				a := make([]account.Balance, 1)
				a[0] = *data.Balance
				q.m.Lock()
				q.Data.AccountBalance = a
				q.m.Unlock()
				select {
				case focus.Stream <- a:
				default:
					// drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case []account.Change:
				focus := q.Focuses.GetByFocusType(AccountHoldingsFocusType)
				if focus == nil {
					// these should never happen, but I've had Stranger Things happen
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, AccountHoldingsFocusType)
					continue
				}
				var a []account.Balance
				for i := range data {
					if data[i].AssetType == q.Key.ExchangeAssetPair.Asset &&
						(data[i].Balance.Currency.Equal(q.Key.ExchangeAssetPair.Pair().Base) || data[i].Balance.Currency.Equal(q.Key.ExchangeAssetPair.Pair().Quote)) {
						a = append(a, *data[i].Balance)
					}
				}
				if len(a) == 0 {
					continue
				}
				q.m.Lock()
				q.Data.AccountBalance = a
				q.m.Unlock()
				select {
				case focus.Stream <- a:
				default:
					// drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case *order.Detail:
				focus := q.Focuses.GetByFocusType(ActiveOrdersFocusType)
				if focus == nil {
					// these should never happen, but I've had Stranger Things happen
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, ActiveOrdersFocusType)
					continue
				}
				q.m.Lock()
				// managing an order list properly goes against the simplicity of quickspy.
				// If you're trying to track everything effectively, PRs welcome with map based management or use something else
				q.Data.Orders = []order.Detail{*data}
				q.m.Unlock()
				select {
				case focus.Stream <- data:
				default:
					// drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case []order.Detail:
				focus := q.Focuses.GetByFocusType(ActiveOrdersFocusType)
				if focus == nil {
					// these should never happen, but I've had Stranger Things happen
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, ActiveOrdersFocusType)
					continue
				}
				o := make([]order.Detail, 0, len(data))
				for i := range data {
					if data[i].Pair.Equal(q.Key.ExchangeAssetPair.Pair()) &&
						data[i].AssetType == q.Key.ExchangeAssetPair.Asset {
						o = append(o, data[i])
					}
				}
				if len(o) == 0 {
					continue
				}
				q.m.Lock()
				q.Data.Orders = o
				q.m.Unlock()
				select {
				case focus.Stream <- o:
				default:
					// drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case []ticker.Price:
				focus := q.Focuses.GetByFocusType(TickerFocusType)
				if focus == nil {
					// these should never happen, but I've had Stranger Things happen
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, TickerFocusType)
					continue
				}
				var td *ticker.Price
				switch {
				case len(data) == 0:
					continue
				case len(data) == 1:
					td = &data[0]
				case len(data) > 1:
					for i := range data {
						if data[i].Pair.Equal(q.Key.ExchangeAssetPair.Pair()) &&
							data[i].AssetType == q.Key.ExchangeAssetPair.Asset {
							td = &data[i]
							break
						}
					}
				}
				if td == nil {
					continue
				}
				q.m.Lock()
				q.Data.Ticker = td
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
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, TickerFocusType)
					continue
				}
				q.m.Lock()
				q.Data.Ticker = data
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
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, OrderBookFocusType)
					continue
				}

				// Retrieve without holding q.m to avoid lock contention
				ob, err := data.Retrieve()
				if err != nil {
					select {
					case focus.Stream <- err:
					default: // drop data that doesn't fit or get listened to
					}
					continue
				}
				q.m.Lock()
				q.Data.Orderbook = ob
				q.m.Unlock()

				select {
				case focus.Stream <- ob:
				default: // drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case trade.Data:
				focus := q.Focuses.GetByFocusType(TradesFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, TradesFocusType)
					continue
				}
				q.m.Lock()
				q.Data.Trades = []trade.Data{data}
				payload := q.Data.Trades
				q.m.Unlock()
				select {
				case focus.Stream <- payload:
				default: // drop data that doesn't fit or get listened to
				}
				focus.SetSuccessful()
			case []trade.Data:
				focus := q.Focuses.GetByFocusType(TradesFocusType)
				if focus == nil {
					log.Errorf(log.QuickSpy, "Quickspy data attempt: %s failed, focus type: %s not found",
						q.Key.ExchangeAssetPair, TradesFocusType)
					continue
				}
				if len(data) == 0 {
					continue
				}
				q.m.Lock()
				q.Data.Trades = data
				payload := q.Data.Trades
				q.m.Unlock()
				select {
				case focus.Stream <- payload:
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
		case <-q.credContext.Done():
			return q.credContext.Err()
		case <-timer.C:
			err := q.handleFocusType(focusType, focus, timer)
			if err != nil {
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %s failed, focus type: %s err: %v",
					failures+1, q.Key.ExchangeAssetPair, focusType, err)
				if focus.IsOnceOff {
					return nil
				}
				if !focus.hasBeenSuccessful {
					if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
						q.successfulSpy(focus, timer)
						return nil
					}
					if failures == 5 {
						return fmt.Errorf("Quickspy data attempt: %v/5 %s failed, focus type: %s err: %v ", failures, q.Key.ExchangeAssetPair, focusType, err)
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
	resp, err := q.Exch.GetCurrencyTradeURL(q.credContext, q.Key.ExchangeAssetPair.Asset, q.Key.ExchangeAssetPair.Pair())
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
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
	contracts, err := q.Exch.GetFuturesContractDetails(q.credContext, q.Key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
	}
	var contractOfFocus *futures.Contract
	for i := range contracts {
		if !contracts[i].Name.Equal(q.Key.ExchangeAssetPair.Pair()) {
			continue
		}
		contractOfFocus = &contracts[i]
		break
	}
	if contractOfFocus == nil {
		return fmt.Errorf("no contract found for %s %s", q.Key.ExchangeAssetPair, focus.Type)
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
	k, err := q.Exch.GetHistoricCandlesExtended(q.credContext, q.Key.ExchangeAssetPair.Pair(), q.Key.ExchangeAssetPair.Asset, kline.OneHour, tt, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			k, err = q.Exch.GetHistoricCandles(q.credContext, q.Key.ExchangeAssetPair.Pair(), q.Key.ExchangeAssetPair.Asset, kline.OneHour, tt, time.Now())
		}
		if err != nil {
			return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
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
		Base:  q.Key.ExchangeAssetPair.Pair().Base.Item,
		Quote: q.Key.ExchangeAssetPair.Pair().Quote.Item,
		Asset: q.Key.ExchangeAssetPair.Asset,
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
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
	tick, err := q.Exch.UpdateTicker(q.credContext, q.Key.ExchangeAssetPair.Pair(), q.Key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Ticker = tick
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleOrdersFocus(focus *FocusData) error {
	resp, err := q.Exch.GetActiveOrders(q.credContext, &order.MultiOrderRequest{
		Pairs:     []currency.Pair{q.Key.ExchangeAssetPair.Pair()},
		AssetType: q.Key.ExchangeAssetPair.Asset,
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Orders = resp
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleAccountHoldingsFocus(focus *FocusData) error {
	ais, err := q.Exch.UpdateAccountInfo(q.credContext, q.Key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w",
			q.Key.ExchangeAssetPair, focus.Type.String(), err)
	}
	// filter results only to passed in key currencies
	sa := make([]account.Balance, 0, 2)
	for _, a := range ais.Accounts {
		if a.AssetType != q.Key.ExchangeAssetPair.Asset {
			continue
		}
		for _, c := range a.Currencies {
			if c.Currency.Equal(q.Key.ExchangeAssetPair.Base.Currency()) {
				sa = append(sa, c)
			}
			if c.Currency.Equal(q.Key.ExchangeAssetPair.Quote.Currency()) {
				sa = append(sa, c)
			}
		}
	}
	focus.m.Lock()
	q.Data.AccountBalance = sa
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleOrderBookFocus(focus *FocusData) error {
	ob, err := q.Exch.UpdateOrderbook(q.credContext, q.Key.ExchangeAssetPair.Pair(), q.Key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Orderbook = ob
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleTradesFocus(focus *FocusData) error {
	tr, err := q.Exch.GetRecentTrades(q.credContext, q.Key.ExchangeAssetPair.Pair(), q.Key.ExchangeAssetPair.Asset)
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
	}
	focus.m.Lock()
	q.Data.Trades = tr
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleOrderExecutionFocus(focus *FocusData) error {
	el, err := q.Exch.GetOrderExecutionLimits(q.Key.ExchangeAssetPair.Asset, q.Key.ExchangeAssetPair.Pair())
	if err != nil {
		err = q.Exch.UpdateOrderExecutionLimits(q.credContext, q.Key.ExchangeAssetPair.Asset)
		if err != nil {
			return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
		}
		el, err = q.Exch.GetOrderExecutionLimits(q.Key.ExchangeAssetPair.Asset, q.Key.ExchangeAssetPair.Pair())
		if err != nil {
			return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
		}
	}
	focus.m.Lock()
	q.Data.ExecutionLimits = &el
	focus.m.Unlock()
	return nil
}

func (q *QuickSpy) handleFundingRateFocus(focus *FocusData) error {
	isPerp, err := q.Exch.IsPerpetualFutureCurrency(q.Key.ExchangeAssetPair.Asset, q.Key.ExchangeAssetPair.Pair())
	if err != nil && !errors.Is(err, futures.ErrNotPerpetualFuture) {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
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
		Asset:                q.Key.ExchangeAssetPair.Asset,
		Pair:                 q.Key.ExchangeAssetPair.Pair(),
		IncludePredictedRate: true,
	})
	if err != nil {
		return fmt.Errorf("%s %q %w", q.Key.ExchangeAssetPair, focus.Type.String(), err)
	}
	if len(fr) != 1 {
		return fmt.Errorf("expected 1 funding rate for %s %q, got %d", q.Key.ExchangeAssetPair, focus.Type.String(), len(fr))
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
	q.credContext.Done()
}

func (q *QuickSpy) Dump() (*ExportedData, error) {
	q.m.RLock()
	defer q.m.RUnlock()
	return q.Data.Dump(q.Key.ExchangeAssetPair, !q.Key.Credentials.IsEmpty())
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

	var holdings []account.Balance
	if d.AccountBalance != nil {
		holdings = make([]account.Balance, len(d.AccountBalance))
		for i := range d.AccountBalance {
			holdings[i] = d.AccountBalance[i]
		}
	}
	var nextFundingRateTime, currentFundingRateTime time.Time
	var execLimitsCopy limits.MinMaxLevel
	if d.ExecutionLimits != nil {
		execLimitsCopy = *d.ExecutionLimits
	}
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
		ExecutionLimits:        execLimitsCopy,
		URL:                    d.URL,
		ContractSettlement:     contractSettlement,
	}, nil
}

func (q *QuickSpy) HasBeenSuccessful(focusType FocusType) (bool, error) {
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return false, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	return focus.HasBeenSuccessful(), nil
}

// LatestData returns the latest focus-specific payload guarded by the
// internal read lock. It returns an error if no data has been collected yet
// for the requested focus type.
func (q *QuickSpy) LatestData(focusType FocusType) (any, error) {
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return false, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	if !focus.HasBeenSuccessful() {
		return nil, fmt.Errorf("%q %w", focusType, errNoDataYet)
	}

	q.m.RLock()
	defer q.m.RUnlock()
	switch focusType {
	case TickerFocusType:
		return q.Data.Ticker, nil
	case OrderBookFocusType:
		return q.Data.Orderbook, nil
	case KlineFocusType:
		return q.Data.Kline, nil
	case TradesFocusType:
		return q.Data.Trades, nil
	case AccountHoldingsFocusType:
		return q.Data.AccountBalance, nil
	case ActiveOrdersFocusType:
		return q.Data.Orders, nil
	case OpenInterestFocusType:
		return q.Data.OpenInterest, nil
	case FundingRateFocusType:
		return q.Data.FundingRate, nil
	case ContractFocusType:
		return q.Data.Contract, nil
	case URLFocusType:
		return q.Data.URL, nil
	case OrderExecutionFocusType:
		return q.Data.ExecutionLimits, nil
	default:
		return nil, fmt.Errorf("unsupported focus: %s", focusType.String())
	}
}

// WaitForInitialData allows a caller to wait for a response before doing other actions
func (q *QuickSpy) WaitForInitialData(ctx context.Context, focusType FocusType) error {
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-focus.HasBeenSuccessfulChan:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitForInitialDataWithTimer waits for initial data for a focus type or cancels when ctx is done.
func (q *QuickSpy) WaitForInitialDataWithTimer(ctx context.Context, focusType FocusType, tt time.Duration) error {
	if tt == 0 {
		return fmt.Errorf("%w: timer cannot be 0", errTimerNotSet)
	}
	focus := q.Focuses.GetByFocusType(focusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	t := time.NewTimer(tt)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-focus.HasBeenSuccessfulChan:
		return nil
	case <-t.C:
		return fmt.Errorf("%w %q", errFocusDataTimeout, focusType)
	}
}
