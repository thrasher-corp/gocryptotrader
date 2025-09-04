package quickspy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// NewQuickestSpy spins up a quickspy with a single focus and the least fuss
// For those who gotta go fast
func NewQuickestSpy(ctx context.Context, exchangeName string, assetType asset.Item, pair currency.Pair, focus FocusType, credentials *account.Credentials) (*QuickSpy, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sm := NewFocusStore()
	useWS := slices.Contains(wsSupportedFocusList, focus)
	focusData := NewFocusData(focus, false, useWS, time.Second)
	sm.Upsert(focus, focusData)
	k := &CredentialsKey{
		ExchangeAssetPair: key.NewExchangeAssetPair(exchangeName, assetType, pair),
		Credentials:       credentials,
	}
	q, err := NewQuickSpy(ctx, k, sm.List())
	if err != nil {
		return nil, err
	}
	if err := q.run(); err != nil {
		return nil, err
	}
	return q, nil
}

// NewQuickSpy returns a running quickspy if everything passed in is valid
func NewQuickSpy(ctx context.Context, k *CredentialsKey, focuses []*FocusData) (*QuickSpy, error) {
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
			return nil, fmt.Errorf("focus %q %w: %w", focuses[i].focusType, errValidationFailed, err)
		}
		sm.Upsert(focuses[i].focusType, focuses[i])
	}
	q := &QuickSpy{
		key:                k,
		dataHandlerChannel: make(chan any),
		focuses:            sm,
		credContext:        ctx,
		data:               &Data{Key: k},
		m:                  new(sync.RWMutex),
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
		b := q.exch.GetBase()
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
	}
	if err := q.run(); err != nil {
		return nil, err
	}
	return q, nil
}

// AnyRequiresWebsocket tells a quickspy whether to setup the websocket
func (q *QuickSpy) AnyRequiresWebsocket() bool {
	for _, focus := range q.focuses.List() {
		if focus.RequiresWebsocket() {
			return true
		}
	}
	return false
}

// AnyRequiresAuth tells quickspy if valid credentials should be present
func (q *QuickSpy) AnyRequiresAuth() bool {
	for _, focus := range q.focuses.List() {
		if focus.RequiresAuth() {
			return true
		}
	}
	return false
}

// FocusTypeRequiresWebsocket checks a focus type to see if it has been set for a websocket subscription
func (q *QuickSpy) FocusTypeRequiresWebsocket(focusType FocusType) bool {
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return false
	}
	return focus.useWebsocket
}

// GetAndWaitForFocusByKey is a convenience function to wait for a quickspy to be setup and have data
// before utilising the desired focus type
func (q *QuickSpy) GetAndWaitForFocusByKey(ctx context.Context, focusType FocusType, timeout time.Duration) (*FocusData, error) {
	focus, err := q.GetFocusByKey(focusType)
	if err != nil {
		return nil, err
	}
	timer := time.NewTimer(timeout)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-focus.hasBeenSuccessfulChan:
		return focus, nil
	case <-timer.C:
		return nil, fmt.Errorf("%w %q", errFocusDataTimeout, focusType)
	}
}

// GetFocusByKey returns FocusData based on its type allowing for streaming data results
func (q *QuickSpy) GetFocusByKey(focusType FocusType) (*FocusData, error) {
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return nil, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	return focus, nil
}

func (q *QuickSpy) setupExchange() error {
	q.m.Lock()
	defer q.m.Unlock()

	e, err := engine.NewSupportedExchangeByName(q.key.ExchangeAssetPair.Exchange)
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
		log.Warnf(log.QuickSpy, "%s websocket setup failed: %v. Disabling websocket focuses", q.key.ExchangeAssetPair, err)
		q.focuses.DisableWebsocketFocuses()
	}
	q.exch = e
	return nil
}

func (q *QuickSpy) setupExchangeDefaults(e exchange.IBotExchange, b *exchange.Base) error {
	e.SetDefaults()
	exchCfg, err := b.GetStandardConfig()
	if err != nil {
		return fmt.Errorf("%s: %w", q.key.ExchangeAssetPair, err)
	}
	if err := b.SetupDefaults(exchCfg); err != nil {
		return fmt.Errorf("%s: %w", q.key.ExchangeAssetPair, err)
	}
	if err := e.Setup(exchCfg); err != nil {
		return fmt.Errorf("%s: %w", q.key.ExchangeAssetPair, err)
	}
	return nil
}

func (q *QuickSpy) setupCurrencyPairs(b *exchange.Base) error {
	var rFmt, cFmt *currency.PairFormat
	if b.CurrencyPairs.UseGlobalFormat {
		rFmt = b.CurrencyPairs.RequestFormat
		cFmt = b.CurrencyPairs.ConfigFormat
	} else {
		rFmt = b.CurrencyPairs.Pairs[q.key.ExchangeAssetPair.Asset].RequestFormat
		cFmt = b.CurrencyPairs.Pairs[q.key.ExchangeAssetPair.Asset].ConfigFormat
	}
	b.CurrencyPairs.DisableAllPairs()
	// no formatting occurs for websocket subscription generation
	// so do it here to cover for it
	cFmtPair := q.key.ExchangeAssetPair.Pair().Format(*cFmt)
	b.CurrencyPairs.Pairs[q.key.ExchangeAssetPair.Asset] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: rFmt,
		ConfigFormat:  cFmt,
	}

	if err := b.CurrencyPairs.StorePairs(q.key.ExchangeAssetPair.Asset, currency.Pairs{cFmtPair}, false); err != nil {
		return err
	}
	if err := b.CurrencyPairs.StorePairs(q.key.ExchangeAssetPair.Asset, currency.Pairs{cFmtPair}, true); err != nil {
		return err
	}
	return nil
}

func (q *QuickSpy) checkRateLimits(b *exchange.Base) error {
	if len(b.GetRateLimiterDefinitions()) == 0 {
		return fmt.Errorf("%s %w", q.key.ExchangeAssetPair, errNoRateLimits)
	}
	return nil
}

func (q *QuickSpy) setupWebsocket(e exchange.IBotExchange, b *exchange.Base) error {
	if q.AnyRequiresWebsocket() {
		if !e.SupportsWebsocket() {
			return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.key.ExchangeAssetPair.Exchange)
		}
	} else {
		return nil
	}

	if !b.Features.Supports.Websocket {
		return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.key.ExchangeAssetPair.Exchange)
	}
	if err := common.NilGuard(b.Websocket); err != nil {
		return fmt.Errorf("%s %w", q.key.ExchangeAssetPair, err)
	}
	// allows routing of all websocket data to our custom one
	b.Websocket.ToRoutine = q.dataHandlerChannel
	var newSubs []*subscription.Subscription
	for _, f := range q.focuses.List() {
		if !f.RequiresWebsocket() {
			continue
		}
		ch, ok := focusToSub[f.focusType]
		if !ok || ch == "" {
			return fmt.Errorf("%s %s %w", q.key.ExchangeAssetPair, f.focusType, errNoWebsocketSupportForFocusType)
		}
		var sub *subscription.Subscription
		for _, s := range b.Config.Features.Subscriptions {
			if s.Channel != ch {
				continue
			}
			if s.Asset != q.key.ExchangeAssetPair.Asset &&
				s.Asset != asset.All && s.Asset != asset.Empty {
				continue
			}
			sub = s
		}
		if sub == nil {
			return fmt.Errorf("%s %s %w", q.key.ExchangeAssetPair, f.focusType, errNoSubSwitchingToREST)
		}
		s := sub.Clone()
		rFmtPair := q.key.ExchangeAssetPair.Pair().Format(*b.CurrencyPairs.Pairs[q.key.ExchangeAssetPair.Asset].RequestFormat)
		s.Pairs.Add(rFmtPair)
		newSubs = append(newSubs, s)
	}
	b.Config.Features.Subscriptions = newSubs
	b.Features.Subscriptions = newSubs
	if err := b.Websocket.EnableAndConnect(); err != nil {
		if !errors.Is(err, websocket.ErrWebsocketAlreadyEnabled) {
			return fmt.Errorf("%s: %w", q.key.ExchangeAssetPair, err)
		}
		// Because EnableAndConnect errors if its already enabled, but also wants to connect
		// we have to do this silly handling, making everyone suffer
		// the complaint was generated by AI, there must be a lot of complaining in scraped comments
		err = b.Websocket.Connect()
		if err != nil {
			return fmt.Errorf("%s: %w", q.key.ExchangeAssetPair, err)
		}
	}
	return nil
}

func (q *QuickSpy) run() error {
	if q.AnyRequiresWebsocket() {
		q.wg.Go(func() {
			err := q.handleWS()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				log.Errorf(log.QuickSpy, "%s websocket handler error: %v", q.key.ExchangeAssetPair, err)
			}
		})
	}
	for i, focus := range q.focuses.List() {
		if focus.useWebsocket {
			continue
		}
		q.wg.Add(1)
		go func(f *FocusData) {
			defer q.wg.Done()
			err := q.runRESTFocus(f)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %s failed, focus type: %q err: %v",
					i+1, q.key.ExchangeAssetPair, f.focusType, err)
			}
		}(focus)
	}
	return nil
}

func (q *QuickSpy) handleWS() error {
	for {
		select {
		case <-q.credContext.Done():
			return q.credContext.Err()
		case d := <-q.dataHandlerChannel:
			switch data := d.(type) {
			case account.Change:
				if err := q.handleWSAccountChange(&data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case []account.Change:
				if err := q.handleWSAccountChanges(data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case *order.Detail:
				if err := q.handleWSOrderDetail(data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case []order.Detail:
				if err := q.handleWSOrderDetails(data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case []ticker.Price:
				if err := q.handleWSTickers(data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case *ticker.Price:
				if err := q.handleWSTicker(data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case *orderbook.Depth:
				if err := q.handleWSOrderbook(data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case trade.Data:
				if err := q.handleWSTrade(&data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			case []trade.Data:
				if err := q.handleWSTrades(data); err != nil {
					log.Errorf(log.QuickSpy, "%s %s", q.key.ExchangeAssetPair, err)
				}
			}
		}
	}
}

func (q *QuickSpy) runRESTFocus(f *FocusData) error {
	if f.useWebsocket {
		return nil
	}
	timer := time.NewTimer(0)
	failures := 0
	for {
		select {
		case <-q.credContext.Done():
			return q.credContext.Err()
		case <-timer.C:
			err := q.handleFocusType(f.focusType, f, timer)
			if err != nil {
				log.Errorf(log.QuickSpy, "Quickspy data attempt: %v %s failed, focus type: %s err: %v",
					failures+1, q.key.ExchangeAssetPair, f.focusType, err)
				if f.isOnceOff {
					return nil
				}
				if !f.hasBeenSuccessful {
					if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
						q.successfulSpy(f, timer)
						return nil
					}
					if failures == 5 {
						return fmt.Errorf("Quickspy data attempt: %v/5 %s failed, focus type: %s err: %v ", failures, q.key.ExchangeAssetPair, f.focusType, err)
					}
					failures++
					timer.Reset(f.restPollTime)
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
		err = q.handleRESTContractFocus(focus)
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
	case OrderBookFocusType:
		err = q.handleOrderBookFocus(focus)
	case TradesFocusType:
		err = q.handleTradesFocus(focus)
	case OrderLimitsFocusType:
		err = q.handleOrderExecutionFocus(focus)
	case FundingRateFocusType:
		err = q.handleFundingRateFocus(focus)
	default:
		return fmt.Errorf("unknown focus type %v", focusType)
	}
	if err != nil {
		focus.stream(err)
	}
	q.successfulSpy(focus, timer)

	return nil
}

func (q *QuickSpy) successfulSpy(focus *FocusData, timer *time.Timer) {
	focus.setSuccessful()
	focus.m.RLock()
	defer focus.m.RUnlock()
	if focus.isOnceOff {
		return
	}
	timer.Reset(focus.restPollTime)
}

func (q *QuickSpy) Shutdown() {
	q.credContext.Done()
}

func (q *QuickSpy) HasBeenSuccessful(focusType FocusType) (bool, error) {
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return false, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	return focus.HasBeenSuccessful(), nil
}

// LatestData returns the latest focus-specific payload guarded by the
// internal read lock. It returns an error if no data has been collected yet
// for the requested focus type.
func (q *QuickSpy) LatestData(focusType FocusType) (any, error) {
	focus := q.focuses.GetByFocusType(focusType)
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
		return q.data.Ticker, nil
	case OrderBookFocusType:
		return q.data.Orderbook, nil
	case KlineFocusType:
		return q.data.Kline, nil
	case TradesFocusType:
		return q.data.Trades, nil
	case AccountHoldingsFocusType:
		return q.data.AccountBalance, nil
	case ActiveOrdersFocusType:
		return q.data.Orders, nil
	case OpenInterestFocusType:
		return q.data.OpenInterest, nil
	case FundingRateFocusType:
		return q.data.FundingRate, nil
	case ContractFocusType:
		return q.data.Contract, nil
	case URLFocusType:
		return q.data.URL, nil
	case OrderLimitsFocusType:
		return q.data.ExecutionLimits, nil
	default:
		return nil, fmt.Errorf("%w %q", ErrUnsupportedFocusType, focusType.String())
	}
}

// DumpJSON conveniently gives you JSON output of all gathered data
func (q *QuickSpy) DumpJSON() ([]byte, error) {
	q.m.RLock()
	defer q.m.RUnlock()
	b, err := json.MarshalIndent(q.data, "", "  ")
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Data returns the internal Data struct pointer and is unsafe while quickspy is running
func (q *QuickSpy) Data() *Data {
	return q.data
}

// WaitForInitialData allows a caller to wait for a response before doing other actions
func (q *QuickSpy) WaitForInitialData(ctx context.Context, focusType FocusType) error {
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-focus.hasBeenSuccessfulChan:
		return nil
	}
}

// WaitForInitialDataWithTimer waits for initial data for a focus type or cancels when ctx is done.
func (q *QuickSpy) WaitForInitialDataWithTimer(ctx context.Context, focusType FocusType, tt time.Duration) error {
	if tt == 0 {
		return fmt.Errorf("%w: timer cannot be 0", errTimerNotSet)
	}
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	t := time.NewTimer(tt)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-focus.hasBeenSuccessfulChan:
		return nil
	case <-t.C:
		return fmt.Errorf("%w %q", errFocusDataTimeout, focusType)
	}
}
