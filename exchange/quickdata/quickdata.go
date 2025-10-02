package quickdata

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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errKeyNotFound                    = errors.New("key not found")
	errNoKey                          = errors.New("no key provided")
	errNoFocus                        = errors.New("no focuses provided")
	errValidationFailed               = errors.New("validation failed")
	errNoCredentials                  = errors.New("credentials required but none provided")
	errFocusDataTimeout               = errors.New("focus did not receive data in time")
	errNoRateLimits                   = errors.New("exchange has no rate limits set")
	errNoWebsocketSupportForFocusType = errors.New("quickData does not support websocket for this focus type")
	errNoSubSwitchingToREST           = errors.New("no subscription found, switching to REST")
	errTimerNotSet                    = errors.New("timer not set")
	errNoDataYet                      = errors.New("no data received yet")
	errOverMaxFailures                = errors.New("exceeded maximum allowed failures")
	errUnhandledWebsocketData         = errors.New("unhandled websocket data type")

	// ErrContextMustBeAbleToFinish returned as quickspy must be able to close
	ErrContextMustBeAbleToFinish = errors.New("context must be able to finish via timeout, deadline or cancellation, view package readme for details")
)

// QuickData is a helper struct to gather data from an exchange quickly
// It supports both public and private data (if credentials are provided)
// It can gather data via websocket (if supported by the exchange) or REST
type QuickData struct {
	exch exchange.IBotExchange
	key  *key.ExchangeAssetPair
	// focuses is a map of focus types to focus options
	// Don't access directly, use functions to handle locking
	focuses *FocusStore
	// dataHandlerChannel is used for receiving data from websockets
	dataHandlerChannel chan any
	// m is used for concurrent read/write operations
	m    sync.RWMutex
	wg   sync.WaitGroup
	data *Data
}

// NewQuickData returns a running quickData if everything passed in is valid
// imbue context with credentials to utilise private endpoints
func NewQuickData(k *key.ExchangeAssetPair, focuses []*FocusData) (*QuickData, error) {
	if k == nil {
		return nil, errNoKey
	}
	if len(focuses) == 0 {
		return nil, errNoFocus
	}
	sm := NewFocusStore()
	for i := range focuses {
		if err := focuses[i].focusType.Valid(); err != nil {
			return nil, err
		}
		focuses[i].Init()
		if err := focuses[i].Validate(k); err != nil {
			return nil, fmt.Errorf("focus %q %w: %w", focuses[i].focusType, errValidationFailed, err)
		}
		sm.Upsert(focuses[i].focusType, focuses[i])
	}

	q := &QuickData{
		key:                k,
		dataHandlerChannel: make(chan any, 10),
		focuses:            sm,
		data:               &Data{Key: k},
	}
	err := q.setupExchange()
	if err != nil {
		return nil, err
	}
	if q.AnyRequiresAuth() {
		b := q.exch.GetBase()
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
	}
	return q, nil
}

// NewQuickerData spins up a quickData with a single focus to quickly return data to the user
// auto opt-in to use websocket as it has REST fallback
func NewQuickerData(k *key.ExchangeAssetPair, focus FocusType) (*QuickData, error) {
	if err := common.NilGuard(k); err != nil {
		return nil, err
	}
	if err := focus.Valid(); err != nil {
		return nil, err
	}
	useWS := slices.Contains(wsSupportedFocusList, focus)
	focusData := NewFocusData(focus, false, useWS, time.Second)
	return NewQuickData(k, []*FocusData{focusData})
}

// NewQuickestData spins up a quickData with a single focus and returns the data channel which streams results
// auto opt-in to use websocket as it has REST fallback
// imbue context with credentials to utilise private endpoints
// Automatically starts gathering data as only a channel is used
func NewQuickestData(ctx context.Context, k *key.ExchangeAssetPair, focus FocusType) (chan any, error) {
	if err := common.NilGuard(k); err != nil {
		return nil, err
	}
	if err := focus.Valid(); err != nil {
		return nil, err
	}
	useWS := slices.Contains(wsSupportedFocusList, focus)
	focusData := NewFocusData(focus, false, useWS, time.Second)
	q, err := NewQuickData(k, []*FocusData{focusData})
	if err != nil {
		return nil, err
	}
	if err := q.Run(ctx); err != nil {
		return nil, err
	}
	fd, err := q.GetFocusByKey(focus)
	if err != nil {
		return nil, err
	}
	return fd.Stream, nil
}

// Run starts all routines to gather data based on the focuses provided
// imbue context with credentials to utilise private endpoints
func (q *QuickData) Run(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok && ctx.Done() == nil {
		return ErrContextMustBeAbleToFinish
	}
	if creds := account.GetCredentialsFromContext(ctx); q.AnyRequiresAuth() && creds == nil {
		return fmt.Errorf("%w for %s", errNoCredentials, q.key)
	}
	if q.AnyRequiresWebsocket() {
		q.wg.Go(func() {
			if err := q.handleWS(ctx); err != nil {
				log.Errorf(log.QuickData, "%s websocket handler error: %v", q.key, err)
			}
		})
	}
	for _, focus := range q.focuses.List() {
		if focus.UseWebsocket() {
			continue
		}
		q.wg.Go(func() {
			if err := q.runRESTRoutine(ctx, focus); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				focus.stream(err)
			}
		})
	}
	q.wg.Go(func() {
		<-ctx.Done()
		if err := q.exch.Shutdown(); err != nil {
			log.Errorf(log.QuickData, "%s exchange shutdown error: %v", q.key, err)
		}
	})

	return nil
}

// AnyRequiresWebsocket tells a quickData whether to setup the websocket
func (q *QuickData) AnyRequiresWebsocket() bool {
	for _, focus := range q.focuses.List() {
		if focus.UseWebsocket() {
			return true
		}
	}
	return false
}

// AnyRequiresAuth tells quickData if valid credentials should be present
func (q *QuickData) AnyRequiresAuth() bool {
	for _, focus := range q.focuses.List() {
		if RequiresAuth(focus.focusType) {
			return true
		}
	}
	return false
}

// GetAndWaitForFocusByKey is a convenience function to wait for a quickData to be setup and have data
// before utilising the desired focus type
func (q *QuickData) GetAndWaitForFocusByKey(ctx context.Context, focusType FocusType, timeout time.Duration) (*FocusData, error) {
	focus, err := q.GetFocusByKey(focusType)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-focus.hasBeenSuccessfulChan:
		return focus, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("%w %q", errFocusDataTimeout, focusType)
	}
}

// GetFocusByKey returns FocusData based on its type allowing for streaming data results
func (q *QuickData) GetFocusByKey(focusType FocusType) (*FocusData, error) {
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return nil, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	return focus, nil
}

func (q *QuickData) setupExchange() error {
	q.m.Lock()
	defer q.m.Unlock()

	e, err := engine.NewSupportedExchangeByName(q.key.Exchange)
	if err != nil {
		return err
	}
	q.exch = e
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
		log.Warnf(log.QuickData, "%s websocket setup failed: %v. Disabling websocket focuses", q.key, err)
		q.focuses.DisableWebsocketFocuses()
	}
	return nil
}

func (q *QuickData) setupExchangeDefaults(e exchange.IBotExchange, b *exchange.Base) error {
	e.SetDefaults()
	exchCfg, err := b.GetStandardConfig()
	if err != nil {
		return fmt.Errorf("%s: %w", q.key, err)
	}
	if err := b.SetupDefaults(exchCfg); err != nil {
		return fmt.Errorf("%s: %w", q.key, err)
	}
	exchCfg.Features.Enabled.TradeFeed = true
	if err := e.Setup(exchCfg); err != nil {
		return fmt.Errorf("%s: %w", q.key, err)
	}
	return nil
}

func (q *QuickData) setupCurrencyPairs(b *exchange.Base) error {
	var rFmt, cFmt *currency.PairFormat
	if b.CurrencyPairs.UseGlobalFormat {
		rFmt = b.CurrencyPairs.RequestFormat
		cFmt = b.CurrencyPairs.ConfigFormat
	} else {
		rFmt = b.CurrencyPairs.Pairs[q.key.Asset].RequestFormat
		cFmt = b.CurrencyPairs.Pairs[q.key.Asset].ConfigFormat
	}
	b.CurrencyPairs.DisableAllPairs()
	// no formatting occurs for websocket subscription generation
	// so do it here to cover for it
	cFmtPair := q.key.Pair().Format(*cFmt)
	b.CurrencyPairs.Pairs[q.key.Asset] = &currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: rFmt,
		ConfigFormat:  cFmt,
	}

	if err := b.CurrencyPairs.StorePairs(q.key.Asset, currency.Pairs{cFmtPair}, false); err != nil {
		return err
	}
	if err := b.CurrencyPairs.StorePairs(q.key.Asset, currency.Pairs{cFmtPair}, true); err != nil {
		return err
	}
	return nil
}

func (q *QuickData) checkRateLimits(b *exchange.Base) error {
	if len(b.GetRateLimiterDefinitions()) == 0 {
		return fmt.Errorf("%s %w", q.key, errNoRateLimits)
	}
	return nil
}

func (q *QuickData) setupWebsocket(e exchange.IBotExchange, b *exchange.Base) error {
	if q.AnyRequiresWebsocket() {
		if !e.SupportsWebsocket() {
			return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.key.Exchange)
		}
	} else {
		return nil
	}

	if !b.Features.Supports.Websocket {
		return fmt.Errorf("exchange %s has no websocket. Websocket requirement was enabled", q.key.Exchange)
	}
	if err := common.NilGuard(b.Websocket); err != nil {
		return fmt.Errorf("%s %w", q.key, err)
	}
	// allows routing of all websocket data to our custom one
	q.dataHandlerChannel = b.Websocket.ToRoutine
	focusList := q.focuses.List()
	newSubs := make([]*subscription.Subscription, 0, len(focusList))
	for _, f := range focusList {
		if !f.UseWebsocket() {
			continue
		}
		ch, ok := focusToSub[f.focusType]
		if !ok || ch == "" {
			return fmt.Errorf("%s %s %w", q.key, f.focusType, errNoWebsocketSupportForFocusType)
		}
		var sub *subscription.Subscription
		for _, s := range b.Config.Features.Subscriptions {
			if s.Channel != ch {
				continue
			}
			if s.Asset != q.key.Asset &&
				s.Asset != asset.All && s.Asset != asset.Empty {
				continue
			}
			sub = s
		}
		if sub == nil {
			return fmt.Errorf("%s %s %w", q.key, f.focusType, errNoSubSwitchingToREST)
		}
		s := sub.Clone()
		rFmtPair := q.key.Pair().Format(*b.CurrencyPairs.Pairs[q.key.Asset].RequestFormat)
		s.Pairs.Add(rFmtPair)
		newSubs = append(newSubs, s)
	}
	b.Config.Features.Subscriptions = newSubs
	b.Features.Subscriptions = newSubs
	if err := b.Websocket.EnableAndConnect(); err != nil {
		if !errors.Is(err, websocket.ErrWebsocketAlreadyEnabled) {
			return fmt.Errorf("%s: %w", q.key, err)
		}
		// EnableAndConnect returns an error if the websocket is already enabled,
		// but a connection still needs to be established. In this case, we manually
		// call Connect to ensure the websocket is connected.
		if err := b.Websocket.Connect(); err != nil {
			return fmt.Errorf("%s: %w", q.key, err)
		}
	}
	return q.validateSubscriptions(newSubs)
}

func (q *QuickData) validateSubscriptions(newSubs []*subscription.Subscription) error {
	if len(newSubs) == 0 {
		if err := q.stopWebsocket(); err != nil {
			return err
		}
		return fmt.Errorf("%s %w", q.key, errNoSubSwitchingToREST)
	}
	b := q.exch.GetBase()
	generatedSubs := b.Websocket.GetSubscriptions()
	if len(generatedSubs) != len(newSubs) {
		if err := q.stopWebsocket(); err != nil {
			return err
		}
		return fmt.Errorf("%s %w", q.key, errNoSubSwitchingToREST)
	}
	for i := range generatedSubs {
		for _, f := range q.focuses.List() {
			if !f.UseWebsocket() {
				continue
			}
			ch, ok := focusToSub[f.focusType]
			if !ok || ch == "" {
				continue
			}
			if generatedSubs[i].Channel != ch {
				continue
			}
			if generatedSubs[i].Asset != q.key.Asset &&
				generatedSubs[i].Asset != asset.All && generatedSubs[i].Asset != asset.Empty {
				if err := q.stopWebsocket(); err != nil {
					return err
				}
				return fmt.Errorf("%s %s %w", q.key, f.focusType, errNoSubSwitchingToREST)
			}
			if !generatedSubs[i].Pairs.Contains(q.key.Pair(), false) {
				if err := q.stopWebsocket(); err != nil {
					return err
				}
				return fmt.Errorf("%s %s %w", q.key, f.focusType, errNoSubSwitchingToREST)
			}
		}
	}
	return nil
}

// stopWebsocket reverts all focuses to REST when websocket does not utilise proper subscriptions
// eg multi connection websockets
func (q *QuickData) stopWebsocket() error {
	b := q.exch.GetBase()
	if err := b.Websocket.Shutdown(); err != nil && !errors.Is(err, websocket.ErrNotConnected) {
		return err
	}
	for _, f := range q.focuses.List() {
		f.m.Lock()
		f.useWebsocket = false
		f.m.Unlock()
	}
	return nil
}

func (q *QuickData) handleWS(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d := <-q.dataHandlerChannel:
			if err := q.handleWSData(d); err != nil {
				log.Errorf(log.QuickData, "%s %s", q.key, err)
			}
		}
	}
}

func (q *QuickData) handleWSData(d any) error {
	if err := common.NilGuard(d); err != nil {
		return err
	}
	switch data := d.(type) {
	case account.Change:
		return q.handleWSAccountChange(&data)
	case []account.Change:
		return q.handleWSAccountChanges(data)
	case *order.Detail:
		return q.handleWSOrderDetail(data)
	case []order.Detail:
		return q.handleWSOrderDetails(data)
	case []ticker.Price:
		return q.handleWSTickers(data)
	case *ticker.Price:
		return q.handleWSTicker(data)
	case *orderbook.Depth:
		return q.handleWSOrderbook(data)
	case trade.Data:
		return q.handleWSTrade(&data)
	case []trade.Data:
		return q.handleWSTrades(data)
	default:
		return fmt.Errorf("%w %T", errUnhandledWebsocketData, data)
	}
}

func (q *QuickData) runRESTRoutine(ctx context.Context, f *FocusData) error {
	if f.UseWebsocket() {
		return nil
	}
	timer := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if err := q.processRESTFocus(ctx, f); err != nil {
				return err
			}
			if f.isOnceOff {
				return nil
			}
			timer.Reset(f.restPollTime)
		}
	}
}

func (q *QuickData) processRESTFocus(ctx context.Context, f *FocusData) error {
	err := q.handleFocusType(ctx, f.focusType, f)
	if err != nil {
		if errors.Is(err, common.ErrFunctionNotSupported) || errors.Is(err, common.ErrNotYetImplemented) {
			return err
		}
		if !f.hasBeenSuccessful {
			f.failures++
			if f.failures >= f.FailureTolerance {
				return fmt.Errorf("%w: %v/%v %s failed, focus type: %s err: %w",
					errOverMaxFailures, f.failures, f.FailureTolerance, q.key, f.focusType, err)
			}
		}
		f.stream(err)
	}

	return nil
}

func (q *QuickData) handleFocusType(ctx context.Context, focusType FocusType, focus *FocusData) error {
	var err error
	switch focusType {
	case URLFocusType:
		err = q.handleURLFocus(ctx, focus)
	case ContractFocusType:
		err = q.handleContractFocus(ctx, focus)
	case KlineFocusType:
		err = q.handleKlineFocus(ctx, focus)
	case OpenInterestFocusType:
		err = q.handleOpenInterestFocus(ctx, focus)
	case TickerFocusType:
		err = q.handleTickerFocus(ctx, focus)
	case ActiveOrdersFocusType:
		err = q.handleOrdersFocus(ctx, focus)
	case AccountHoldingsFocusType:
		err = q.handleAccountHoldingsFocus(ctx, focus)
	case OrderBookFocusType:
		err = q.handleOrderBookFocus(ctx, focus)
	case TradesFocusType:
		err = q.handleTradesFocus(ctx, focus)
	case OrderLimitsFocusType:
		err = q.handleOrderExecutionFocus(ctx, focus)
	case FundingRateFocusType:
		err = q.handleFundingRateFocus(ctx, focus)
	default:
		return fmt.Errorf("%w %v", ErrUnsupportedFocusType, focusType)
	}
	if err != nil {
		return err
	}
	focus.setSuccessful()
	return nil
}

// HasBeenSuccessful returns whether a focus type has ever been successful
// or an error if the focus type does not exist
func (q *QuickData) HasBeenSuccessful(focusType FocusType) (bool, error) {
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return false, fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	return focus.HasBeenSuccessful(), nil
}

// Data holds the GCT types that QuickData gathers
type Data struct {
	Key             *key.ExchangeAssetPair          `json:"key"`
	Contract        *futures.Contract               `json:"contract,omitempty"`
	Orderbook       *orderbook.Book                 `json:"orderbook,omitempty"`
	Ticker          *ticker.Price                   `json:"ticker,omitempty"`
	Kline           []websocket.KlineData           `json:"kline,omitempty"`
	AccountBalance  []account.Balance               `json:"accountBalance,omitempty"`
	Orders          []order.Detail                  `json:"orders,omitempty"`
	FundingRate     *fundingrate.LatestRateResponse `json:"fundingRate,omitempty"`
	Trades          []trade.Data                    `json:"trades,omitempty"`
	ExecutionLimits *limits.MinMaxLevel             `json:"executionLimits,omitempty"`
	URL             string                          `json:"url,omitzero"`
	OpenInterest    float64                         `json:"openInterest,omitzero"`
}

// LatestData returns the latest focus-specific payload guarded by the
// internal read lock. It returns an error if no data has been collected yet
// for the requested focus type.
func (q *QuickData) LatestData(focusType FocusType) (any, error) {
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
func (q *QuickData) DumpJSON() ([]byte, error) {
	q.m.RLock()
	defer q.m.RUnlock()
	return json.MarshalIndent(q.data, "", "  ")
}

// Data returns the internal Data struct pointer and is unsafe while quickData is running
func (q *QuickData) Data() *Data {
	return q.data
}

// WaitForInitialData allows a caller to wait for a response before doing other actions
func (q *QuickData) WaitForInitialData(ctx context.Context, focusType FocusType) error {
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

// WaitForInitialDataWithTimeout waits for initial data for a focus type or cancels when ctx is done.
func (q *QuickData) WaitForInitialDataWithTimeout(ctx context.Context, focusType FocusType, timeout time.Duration) error {
	if timeout == 0 {
		return fmt.Errorf("%w: timer cannot be 0", errTimerNotSet)
	}
	focus := q.focuses.GetByFocusType(focusType)
	if focus == nil {
		return fmt.Errorf("%w %q", errKeyNotFound, focusType)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-focus.hasBeenSuccessfulChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("%w %q", errFocusDataTimeout, focusType)
	}
}
