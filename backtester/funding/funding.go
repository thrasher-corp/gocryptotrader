package funding

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding/trackingcurrencies"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrFundsNotFound used when funds are requested but the funding is not found in the manager
	ErrFundsNotFound = errors.New("funding not found")
	// ErrAlreadyExists used when a matching item or pair is already in the funding manager
	ErrAlreadyExists = errors.New("funding already exists")
	// ErrUSDTrackingDisabled used when attempting to track USD values when disabled
	ErrUSDTrackingDisabled = errors.New("USD tracking disabled")

	errCannotAllocate             = errors.New("cannot allocate funds")
	errZeroAmountReceived         = errors.New("amount received less than or equal to zero")
	errNegativeAmountReceived     = errors.New("received negative decimal")
	errNotEnoughFunds             = errors.New("not enough funds")
	errCannotTransferToSameFunds  = errors.New("cannot send funds to self")
	errTransferMustBeSameCurrency = errors.New("cannot transfer to different currency")
	errCannotMatchTrackingToItem  = errors.New("cannot match tracking data to funding items")
	errNotFutures                 = errors.New("item linking collateral currencies must be a futures asset")
	errExchangeManagerRequired    = errors.New("exchange manager required")
)

// SetupFundingManager creates the funding holder. It carries knowledge about levels of funding
// across all execution handlers and enables fund transfers
func SetupFundingManager(exchManager *engine.ExchangeManager, usingExchangeLevelFunding, disableUSDTracking bool) (*FundManager, error) {
	if exchManager == nil {
		return nil, errExchangeManagerRequired
	}
	return &FundManager{
		usingExchangeLevelFunding: usingExchangeLevelFunding,
		disableUSDTracking:        disableUSDTracking,
		exchangeManager:           exchManager,
	}, nil
}

// CreateFuturesCurrencyCode converts a currency pair into a code
// The main reasoning is that as a contract, it exists as an item even if
// it is formatted as BTC-1231. To treat it as a pair in the funding system
// would cause an increase in funds for BTC, when it is an increase in contracts
// This function is basic, but is important be explicit in why this is occurring
func CreateFuturesCurrencyCode(b, q currency.Code) currency.Code {
	return currency.NewCode(fmt.Sprintf("%s-%s", b, q))
}

// CreateItem creates a new funding item
func CreateItem(exch string, a asset.Item, ci currency.Code, initialFunds, transferFee decimal.Decimal) (*Item, error) {
	if initialFunds.IsNegative() {
		return nil, fmt.Errorf("%v %v %v %w initial funds: %v", exch, a, ci, errNegativeAmountReceived, initialFunds)
	}
	if transferFee.IsNegative() {
		return nil, fmt.Errorf("%v %v %v %w transfer fee: %v", exch, a, ci, errNegativeAmountReceived, transferFee)
	}

	return &Item{
		exchange:     strings.ToLower(exch),
		asset:        a,
		currency:     ci,
		initialFunds: initialFunds,
		available:    initialFunds,
		transferFee:  transferFee,
		snapshot:     make(map[int64]ItemSnapshot),
	}, nil
}

// LinkCollateralCurrency links an item to an existing currency code
// for collateral purposes
func (f *FundManager) LinkCollateralCurrency(item *Item, code currency.Code) error {
	if !item.asset.IsFutures() {
		return errNotFutures
	}
	for i := range f.items {
		if f.items[i].currency.Equal(code) && f.items[i].asset == item.asset {
			item.pairedWith = f.items[i]
			return nil
		}
	}
	collateral := &Item{
		exchange:     item.exchange,
		asset:        item.asset,
		currency:     code,
		pairedWith:   item,
		isCollateral: true,
	}
	if err := f.AddItem(collateral); err != nil {
		return err
	}
	item.pairedWith = collateral
	return nil
}

// CreateSnapshot creates a Snapshot for an event's point in time
// as funding.snapshots is a map, it allows for the last event
// in the chronological list to establish the canon at X time
func (f *FundManager) CreateSnapshot(t time.Time) {
	for i := range f.items {
		if f.items[i].snapshot == nil {
			f.items[i].snapshot = make(map[int64]ItemSnapshot)
			if f.items[i].isCollateral {
				f.items[i].initialFunds = f.items[i].available
			}
		} else if _, ok := f.items[i].snapshot[t.UnixNano()]; ok {
			f.items[i].snapshot[t.UnixNano()] = ItemSnapshot{}
		}

		iss := ItemSnapshot{
			Available: f.items[i].available,
			Time:      t,
		}

		if !f.disableUSDTracking {
			var usdClosePrice decimal.Decimal
			if f.items[i].trackingCandles == nil {
				continue
			}
			usdCandles := f.items[i].trackingCandles.GetStream()
			for j := range usdCandles {
				if usdCandles[j].GetTime().Equal(t) {
					usdClosePrice = usdCandles[j].GetClosePrice()
					break
				}
			}
			iss.USDClosePrice = usdClosePrice
			iss.USDValue = usdClosePrice.Mul(f.items[i].available)
		}

		f.items[i].snapshot[t.UnixNano()] = iss
	}
}

// AddUSDTrackingData adds USD tracking data to a funding item
// only in the event that it is not USD and there is data
func (f *FundManager) AddUSDTrackingData(k *kline.DataFromKline) error {
	if f == nil || f.items == nil {
		return common.ErrNilArguments
	}
	if f.disableUSDTracking {
		return ErrUSDTrackingDisabled
	}
	baseSet := false
	quoteSet := false
	var basePairedWith currency.Code
	for i := range f.items {
		if baseSet && quoteSet {
			return nil
		}
		if f.items[i].asset.IsFutures() && k.Item.Asset.IsFutures() {
			if f.items[i].isCollateral {
				err := f.setUSDCandles(k, i)
				if err != nil {
					return err
				}
			} else {
				f.items[i].trackingCandles = k
				baseSet = true
			}
			continue
		}

		if strings.EqualFold(f.items[i].exchange, k.Item.Exchange) &&
			f.items[i].asset == k.Item.Asset {
			if f.items[i].currency.Equal(k.Item.Pair.Base) {
				if f.items[i].trackingCandles == nil &&
					trackingcurrencies.CurrencyIsUSDTracked(k.Item.Pair.Quote) {
					f.items[i].trackingCandles = k
					if f.items[i].pairedWith != nil {
						basePairedWith = f.items[i].pairedWith.currency
					}
				}
				baseSet = true
			}
			if trackingcurrencies.CurrencyIsUSDTracked(f.items[i].currency) {
				if f.items[i].pairedWith != nil && !f.items[i].currency.Equal(basePairedWith) {
					continue
				}
				if f.items[i].trackingCandles == nil {
					err := f.setUSDCandles(k, i)
					if err != nil {
						return err
					}
				}
				quoteSet = true
			}
		}
	}
	if baseSet {
		return nil
	}
	return fmt.Errorf("%w %v %v %v", errCannotMatchTrackingToItem, k.Item.Exchange, k.Item.Asset, k.Item.Pair)
}

// setUSDCandles sets usd tracking candles
// usd stablecoins do not always match in value,
// this is a simplified implementation that can allow
// USD tracking for many currencies across many exchanges
func (f *FundManager) setUSDCandles(k *kline.DataFromKline, i int) error {
	usdCandles := gctkline.Item{
		Exchange: k.Item.Exchange,
		Pair:     currency.Pair{Delimiter: k.Item.Pair.Delimiter, Base: f.items[i].currency, Quote: currency.USD},
		Asset:    k.Item.Asset,
		Interval: k.Item.Interval,
		Candles:  make([]gctkline.Candle, len(k.Item.Candles)),
	}
	for j := range usdCandles.Candles {
		usdCandles.Candles[j] = gctkline.Candle{
			Time:  k.Item.Candles[j].Time,
			Open:  1,
			High:  1,
			Low:   1,
			Close: 1,
		}
	}
	cpy := *k
	cpy.Item = usdCandles
	if err := cpy.Load(); err != nil {
		return err
	}
	f.items[i].trackingCandles = &cpy
	return nil
}

// CreatePair adds two funding items and associates them with one another
// the association allows for the same currency to be used multiple times when
// usingExchangeLevelFunding is false. eg BTC-USDT and LTC-USDT do not share the same
// USDT level funding
func CreatePair(base, quote *Item) (*SpotPair, error) {
	if base == nil {
		return nil, fmt.Errorf("base %w", common.ErrNilArguments)
	}
	if quote == nil {
		return nil, fmt.Errorf("quote %w", common.ErrNilArguments)
	}
	// copy to prevent the off chance of sending in the same base OR quote
	// to create a new pair with a new base OR quote
	bCopy := *base
	qCopy := *quote
	bCopy.pairedWith = &qCopy
	qCopy.pairedWith = &bCopy
	return &SpotPair{base: &bCopy, quote: &qCopy}, nil
}

// CreateCollateral adds two funding items and associates them with one another
// the association allows for the same currency to be used multiple times when
// usingExchangeLevelFunding is false. eg BTC-USDT and LTC-USDT do not share the same
// USDT level funding
func CreateCollateral(contract, collateral *Item) (*CollateralPair, error) {
	if contract == nil {
		return nil, fmt.Errorf("base %w", common.ErrNilArguments)
	}
	if collateral == nil {
		return nil, fmt.Errorf("quote %w", common.ErrNilArguments)
	}
	// copy to prevent the off chance of sending in the same base OR quote
	// to create a new pair with a new base OR quote
	bCopy := *contract
	qCopy := *collateral
	bCopy.pairedWith = &qCopy
	qCopy.pairedWith = &bCopy
	return &CollateralPair{contract: &bCopy, collateral: &qCopy}, nil
}

// Reset clears all settings
func (f *FundManager) Reset() {
	*f = FundManager{}
}

// USDTrackingDisabled clears all settings
func (f *FundManager) USDTrackingDisabled() bool {
	return f.disableUSDTracking
}

// GenerateReport builds report data for result HTML report
func (f *FundManager) GenerateReport() *Report {
	report := Report{
		UsingExchangeLevelFunding: f.usingExchangeLevelFunding,
		DisableUSDTracking:        f.disableUSDTracking,
	}
	items := make([]ReportItem, len(f.items))
	for i := range f.items {
		item := ReportItem{
			Exchange:     f.items[i].exchange,
			Asset:        f.items[i].asset,
			Currency:     f.items[i].currency,
			InitialFunds: f.items[i].initialFunds,
			TransferFee:  f.items[i].transferFee,
			FinalFunds:   f.items[i].available,
			IsCollateral: f.items[i].isCollateral,
		}

		if !f.disableUSDTracking &&
			f.items[i].trackingCandles != nil {
			usdStream := f.items[i].trackingCandles.GetStream()
			item.USDInitialFunds = f.items[i].initialFunds.Mul(usdStream[0].GetClosePrice())
			item.USDFinalFunds = f.items[i].available.Mul(usdStream[len(usdStream)-1].GetClosePrice())
			item.USDInitialCostForOne = usdStream[0].GetClosePrice()
			item.USDFinalCostForOne = usdStream[len(usdStream)-1].GetClosePrice()
			item.USDPairCandle = f.items[i].trackingCandles
		}

		var pricingOverTime []ItemSnapshot
	snaps:
		for _, v := range f.items[i].snapshot {
			pricingOverTime = append(pricingOverTime, v)
			if !f.items[i].asset.IsFutures() && !f.disableUSDTracking {
				for j := range report.USDTotalsOverTime {
					if report.USDTotalsOverTime[j].Time.Equal(v.Time) {
						report.USDTotalsOverTime[j].USDValue = report.USDTotalsOverTime[j].USDValue.Add(v.USDValue)
						report.USDTotalsOverTime[j].Breakdown = append(report.USDTotalsOverTime[j].Breakdown, CurrencyContribution{
							Currency: f.items[i].currency,
							USD:      v.USDValue,
						})
						continue snaps
					} else {
						continue
					}
				}
				report.USDTotalsOverTime = append(report.USDTotalsOverTime, ItemSnapshot{
					Time:     v.Time,
					USDValue: v.USDValue,
					Breakdown: []CurrencyContribution{
						{
							Currency: f.items[i].currency,
							USD:      v.USDValue,
						},
					},
				})
			}
		}

		sort.Slice(pricingOverTime, func(i, j int) bool {
			return pricingOverTime[i].Time.Before(pricingOverTime[j].Time)
		})
		item.Snapshots = pricingOverTime

		if f.items[i].initialFunds.IsZero() {
			item.ShowInfinite = true
		} else {
			item.Difference = f.items[i].available.Sub(f.items[i].initialFunds).Div(f.items[i].initialFunds).Mul(decimal.NewFromInt(100))
		}
		if f.items[i].pairedWith != nil {
			item.PairedWith = f.items[i].pairedWith.currency
		}

		items[i] = item
	}

	sort.Slice(report.USDTotalsOverTime, func(i, j int) bool {
		return report.USDTotalsOverTime[i].Time.Before(report.USDTotalsOverTime[j].Time)
	})

	report.Items = items
	return &report
}

// Transfer allows transferring funds from one pretend exchange to another
func (f *FundManager) Transfer(amount decimal.Decimal, sender, receiver *Item, inclusiveFee bool) error {
	if sender == nil || receiver == nil {
		return common.ErrNilArguments
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return errZeroAmountReceived
	}
	if inclusiveFee {
		if sender.available.LessThan(amount) {
			return fmt.Errorf("%w for %v", errNotEnoughFunds, sender.currency)
		}
	} else {
		if sender.available.LessThan(amount.Add(sender.transferFee)) {
			return fmt.Errorf("%w for %v", errNotEnoughFunds, sender.currency)
		}
	}

	if !sender.currency.Equal(receiver.currency) {
		return errTransferMustBeSameCurrency
	}
	if sender.currency.Equal(receiver.currency) &&
		sender.exchange == receiver.exchange &&
		sender.asset == receiver.asset {
		return fmt.Errorf("%v %v %v %w", sender.exchange, sender.asset, sender.currency, errCannotTransferToSameFunds)
	}

	sendAmount := amount
	receiveAmount := amount
	if inclusiveFee {
		receiveAmount = amount.Sub(sender.transferFee)
	} else {
		sendAmount = amount.Add(sender.transferFee)
	}
	err := sender.Reserve(sendAmount)
	if err != nil {
		return err
	}
	err = receiver.IncreaseAvailable(receiveAmount)
	if err != nil {
		return err
	}
	return sender.Release(sendAmount, decimal.Zero)
}

// AddItem appends a new funding item. Will reject if exists by exchange asset currency
func (f *FundManager) AddItem(item *Item) error {
	if f.Exists(item) {
		return fmt.Errorf("cannot add item %v %v %v %w", item.exchange, item.asset, item.currency, ErrAlreadyExists)
	}
	f.items = append(f.items, item)
	return nil
}

// Exists verifies whether there is a funding item that exists
// with the same exchange, asset and currency
func (f *FundManager) Exists(item *Item) bool {
	for i := range f.items {
		if f.items[i].Equal(item) {
			return true
		}
	}
	return false
}

// AddPair adds a pair to the fund manager if it does not exist
func (f *FundManager) AddPair(p *SpotPair) error {
	if f.Exists(p.base) {
		return fmt.Errorf("%w %v", ErrAlreadyExists, p.base)
	}
	if f.Exists(p.quote) {
		return fmt.Errorf("%w %v", ErrAlreadyExists, p.quote)
	}
	f.items = append(f.items, p.base, p.quote)
	return nil
}

// IsUsingExchangeLevelFunding returns if using usingExchangeLevelFunding
func (f *FundManager) IsUsingExchangeLevelFunding() bool {
	return f.usingExchangeLevelFunding
}

// GetFundingForEvent This will construct a funding based on a backtesting event
func (f *FundManager) GetFundingForEvent(ev common.EventHandler) (IFundingPair, error) {
	return f.getFundingForEAP(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
}

// GetFundingForEAP This will construct a funding based on the exchange, asset, currency pair
func (f *FundManager) getFundingForEAP(exch string, a asset.Item, p currency.Pair) (IFundingPair, error) {
	if a.IsFutures() {
		var collat CollateralPair
		for i := range f.items {
			if f.items[i].MatchesCurrency(currency.NewCode(p.String())) {
				collat.contract = f.items[i]
				collat.collateral = f.items[i].pairedWith
				return &collat, nil
			}
		}
	} else {
		var resp SpotPair
		for i := range f.items {
			if f.items[i].BasicEqual(exch, a, p.Base, p.Quote) {
				resp.base = f.items[i]
				continue
			}
			if f.items[i].BasicEqual(exch, a, p.Quote, p.Base) {
				resp.quote = f.items[i]
			}
		}
		if resp.base == nil {
			return nil, fmt.Errorf("base %v %w", p.Base, ErrFundsNotFound)
		}
		if resp.quote == nil {
			return nil, fmt.Errorf("quote %v %w", p.Quote, ErrFundsNotFound)
		}
		return &resp, nil
	}

	return nil, fmt.Errorf("%v %v %v %w", exch, a, p, ErrFundsNotFound)
}

// GetFundingForEAC This will construct a funding based on the exchange, asset, currency code
func (f *FundManager) getFundingForEAC(exch string, a asset.Item, c currency.Code) (*Item, error) {
	for i := range f.items {
		if f.items[i].BasicEqual(exch, a, c, currency.EMPTYCODE) {
			return f.items[i], nil
		}
	}
	return nil, ErrFundsNotFound
}

// Liquidate will remove all funding for all items belonging to an exchange
func (f *FundManager) Liquidate(ev common.EventHandler) {
	if ev == nil {
		return
	}
	for i := range f.items {
		if f.items[i].exchange == ev.GetExchange() {
			f.items[i].reserved = decimal.Zero
			f.items[i].available = decimal.Zero
			f.items[i].isLiquidated = true
		}
	}
}

// GetAllFunding returns basic representations of all current
// holdings from the latest point
func (f *FundManager) GetAllFunding() []BasicItem {
	result := make([]BasicItem, len(f.items))
	for i := range f.items {
		var usd decimal.Decimal
		if f.items[i].trackingCandles != nil {
			latest := f.items[i].trackingCandles.Latest()
			if latest != nil {
				usd = latest.GetClosePrice()
			}
		}
		result[i] = BasicItem{
			Exchange:     f.items[i].exchange,
			Asset:        f.items[i].asset,
			Currency:     f.items[i].currency,
			InitialFunds: f.items[i].initialFunds,
			Available:    f.items[i].available,
			Reserved:     f.items[i].reserved,
			USDPrice:     usd,
		}
	}
	return result
}

// UpdateCollateral will recalculate collateral for an exchange
// based on the event passed in
func (f *FundManager) UpdateCollateral(ev common.EventHandler) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	exchMap := make(map[string]exchange.IBotExchange)
	var collateralAmount decimal.Decimal
	var err error
	calculator := gctorder.TotalCollateralCalculator{
		CalculateOffline: true,
	}

	for i := range f.items {
		if f.items[i].asset.IsFutures() {
			// futures positions aren't collateral, they use it
			continue
		}
		_, ok := exchMap[f.items[i].exchange]
		if !ok {
			var exch exchange.IBotExchange
			exch, err = f.exchangeManager.GetExchangeByName(f.items[i].exchange)
			if err != nil {
				return err
			}
			exchMap[f.items[i].exchange] = exch
		}
		var usd decimal.Decimal
		if f.items[i].trackingCandles != nil {
			latest := f.items[i].trackingCandles.Latest()
			if latest != nil {
				usd = latest.GetClosePrice()
			}
		}
		if usd.IsZero() {
			continue
		}
		var side = gctorder.Buy
		if !f.items[i].available.GreaterThan(decimal.Zero) {
			side = gctorder.Sell
		}

		calculator.CollateralAssets = append(calculator.CollateralAssets, gctorder.CollateralCalculator{
			CalculateOffline:   true,
			CollateralCurrency: f.items[i].currency,
			Asset:              f.items[i].asset,
			Side:               side,
			FreeCollateral:     f.items[i].available,
			LockedCollateral:   f.items[i].reserved,
			USDPrice:           usd,
		})
	}
	exch, ok := exchMap[ev.GetExchange()]
	if !ok {
		return fmt.Errorf("%v %w", ev.GetExchange(), engine.ErrExchangeNotFound)
	}
	futureCurrency, futureAsset, err := exch.GetCollateralCurrencyForContract(ev.GetAssetType(), ev.Pair())
	if err != nil {
		return err
	}

	collat, err := exchMap[ev.GetExchange()].CalculateTotalCollateral(context.TODO(), &calculator)
	if err != nil {
		return err
	}

	for i := range f.items {
		if f.items[i].exchange == ev.GetExchange() &&
			f.items[i].asset == futureAsset &&
			f.items[i].currency.Equal(futureCurrency) {
			f.items[i].available = collat.AvailableCollateral
			return nil
		}
	}
	return fmt.Errorf("%w to allocate %v to %v %v %v", ErrFundsNotFound, collateralAmount, ev.GetExchange(), ev.GetAssetType(), futureCurrency)
}

// HasFutures returns whether the funding manager contains any futures assets
func (f *FundManager) HasFutures() bool {
	for i := range f.items {
		if f.items[i].isCollateral || f.items[i].asset.IsFutures() {
			return true
		}
	}
	return false
}

// RealisePNL adds the realised PNL to a receiving exchange asset pair
func (f *FundManager) RealisePNL(receivingExchange string, receivingAsset asset.Item, receivingCurrency currency.Code, realisedPNL decimal.Decimal) error {
	for i := range f.items {
		if f.items[i].exchange == receivingExchange &&
			f.items[i].asset == receivingAsset &&
			f.items[i].currency.Equal(receivingCurrency) {
			return f.items[i].TakeProfit(realisedPNL)
		}
	}
	return fmt.Errorf("%w to allocate %v to %v %v %v", ErrFundsNotFound, realisedPNL, receivingExchange, receivingAsset, receivingCurrency)
}

// HasExchangeBeenLiquidated checks for any items with a matching exchange
// and returns whether it has been liquidated
func (f *FundManager) HasExchangeBeenLiquidated(ev common.EventHandler) bool {
	for i := range f.items {
		if ev.GetExchange() == f.items[i].exchange {
			return f.items[i].isLiquidated
		}
	}
	return false
}
