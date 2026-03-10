package funding

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding/trackingcurrencies"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupFundingManager creates the funding holder. It carries knowledge about levels of funding
// across all execution handlers and enables fund transfers
func SetupFundingManager(exchManager *engine.ExchangeManager, usingExchangeLevelFunding, disableUSDTracking, verbose bool) (*FundManager, error) {
	if exchManager == nil {
		return nil, errExchangeManagerRequired
	}
	return &FundManager{
		usingExchangeLevelFunding: usingExchangeLevelFunding,
		disableUSDTracking:        disableUSDTracking,
		exchangeManager:           exchManager,
		verbose:                   verbose,
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
	if item == nil {
		return fmt.Errorf("%w missing item", gctcommon.ErrNilPointer)
	}
	if code.IsEmpty() {
		return fmt.Errorf("%w unset currency", gctcommon.ErrNilPointer)
	}
	if !item.asset.IsFutures() {
		return errNotFutures
	}
	if item.pairedWith != nil {
		return fmt.Errorf("%w item already paired with %v", ErrAlreadyExists, item.pairedWith.currency)
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
func (f *FundManager) CreateSnapshot(t time.Time) error {
	if t.IsZero() {
		return gctcommon.ErrDateUnset
	}
	for i := range f.items {
		if f.items[i].snapshot == nil {
			f.items[i].snapshot = make(map[int64]ItemSnapshot)
		}
		iss, ok := f.items[i].snapshot[t.UnixNano()]
		if !ok {
			iss = ItemSnapshot{
				Time: t,
			}
		}
		iss.Available = f.items[i].available
		if !f.disableUSDTracking {
			if f.items[i].trackingCandles == nil {
				continue
			}
			var usdClosePrice decimal.Decimal
			usdCandles, err := f.items[i].trackingCandles.GetStream()
			if err != nil {
				return err
			}
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
	return nil
}

// AddUSDTrackingData adds USD tracking data to a funding item
// only in the event that it is not USD and there is data
func (f *FundManager) AddUSDTrackingData(k *kline.DataFromKline) error {
	if f == nil || f.items == nil {
		return gctcommon.ErrNilPointer
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
				err := f.setUSDCandles(k, f.items[i])
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
				if trackingcurrencies.CurrencyIsUSDTracked(k.Item.Pair.Quote) {
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
				err := f.setUSDCandles(k, f.items[i])
				if err != nil {
					return err
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
func (f *FundManager) setUSDCandles(k *kline.DataFromKline, i *Item) error {
	usdCandles := gctkline.Item{
		Exchange: k.Item.Exchange,
		Pair:     currency.Pair{Delimiter: k.Item.Pair.Delimiter, Base: i.currency, Quote: currency.USD},
		Asset:    k.Item.Asset,
		Interval: k.Item.Interval,
		Candles:  make([]gctkline.Candle, len(k.Item.Candles)),
	}
	for x := range usdCandles.Candles {
		usdCandles.Candles[x] = gctkline.Candle{
			Time:  k.Item.Candles[x].Time,
			Open:  1,
			High:  1,
			Low:   1,
			Close: 1,
		}
	}
	cpy := *k
	cpy.Item = &usdCandles
	cpy.Base = &data.Base{}
	if err := cpy.Load(); err != nil {
		return err
	}
	i.trackingCandles = &cpy
	return nil
}

// CreatePair adds two funding items and associates them with one another
// the association allows for the same currency to be used multiple times when
// usingExchangeLevelFunding is false. eg BTC-USDT and LTC-USDT do not share the same
// USDT level funding
func CreatePair(base, quote *Item) (*SpotPair, error) {
	if base == nil {
		return nil, fmt.Errorf("base %w", gctcommon.ErrNilPointer)
	}
	if quote == nil {
		return nil, fmt.Errorf("quote %w", gctcommon.ErrNilPointer)
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
		return nil, fmt.Errorf("base %w", gctcommon.ErrNilPointer)
	}
	if collateral == nil {
		return nil, fmt.Errorf("quote %w", gctcommon.ErrNilPointer)
	}
	collateral.isCollateral = true
	// copy to prevent the off chance of sending in the same base OR quote
	// to create a new pair with a new base OR quote
	bCopy := *contract
	qCopy := *collateral
	bCopy.pairedWith = &qCopy
	qCopy.pairedWith = &bCopy
	return &CollateralPair{contract: &bCopy, collateral: &qCopy}, nil
}

// Reset clears all settings
func (f *FundManager) Reset() error {
	if f == nil {
		return gctcommon.ErrNilPointer
	}
	*f = FundManager{}
	return nil
}

// USDTrackingDisabled clears all settings
func (f *FundManager) USDTrackingDisabled() bool {
	return f.disableUSDTracking
}

// GenerateReport builds report data for result HTML report
func (f *FundManager) GenerateReport() (*Report, error) {
	report := Report{
		UsingExchangeLevelFunding: f.usingExchangeLevelFunding,
		DisableUSDTracking:        f.disableUSDTracking,
	}
	items := make([]ReportItem, len(f.items))
	for x := range f.items {
		item := ReportItem{
			Exchange:       f.items[x].exchange,
			Asset:          f.items[x].asset,
			Currency:       f.items[x].currency,
			InitialFunds:   f.items[x].initialFunds,
			TransferFee:    f.items[x].transferFee,
			FinalFunds:     f.items[x].available,
			IsCollateral:   f.items[x].isCollateral,
			AppendedViaAPI: f.items[x].appendedViaAPI,
		}

		if !f.disableUSDTracking &&
			f.items[x].trackingCandles != nil {
			usdStream, err := f.items[x].trackingCandles.GetStream()
			if err != nil {
				return nil, err
			}
			last, err := usdStream.Last()
			if err != nil {
				log.Errorf(common.FundManager, "USD tracking data is nil for %v %v %v, please ensure data is present", f.items[x].exchange, f.items[x].asset, f.items[x].currency)
			}
			first, err := usdStream.First()
			if err != nil {
				log.Errorf(common.FundManager, "USD tracking data is nil for %v %v %v, please ensure data is present", f.items[x].exchange, f.items[x].asset, f.items[x].currency)
			}
			if !item.IsCollateral {
				item.USDInitialFunds = f.items[x].initialFunds.Mul(first.GetClosePrice())
				item.USDFinalFunds = f.items[x].available.Mul(last.GetClosePrice())
			}

			item.USDInitialCostForOne = first.GetClosePrice()
			item.USDFinalCostForOne = last.GetClosePrice()
			item.USDPairCandle = f.items[x].trackingCandles
		}

		// create a breakdown of USD values and currency contributions over the span of run
		var pricingOverTime []ItemSnapshot
	snaps:
		for _, snapshot := range f.items[x].snapshot {
			pricingOverTime = append(pricingOverTime, snapshot)
			if f.items[x].asset.IsFutures() || f.disableUSDTracking {
				// futures contracts / collateral does not contribute to USD value
				// no USD tracking means no USD values to breakdown
				continue
			}
			for y := range report.USDTotalsOverTime {
				if !report.USDTotalsOverTime[y].Time.Equal(snapshot.Time) {
					continue
				}
				report.USDTotalsOverTime[y].USDValue = report.USDTotalsOverTime[y].USDValue.Add(snapshot.USDValue)
				report.USDTotalsOverTime[y].Breakdown = append(report.USDTotalsOverTime[y].Breakdown, CurrencyContribution{
					Currency:        f.items[x].currency,
					USDContribution: snapshot.USDValue,
				})
				continue snaps
			}
			report.USDTotalsOverTime = append(report.USDTotalsOverTime, ItemSnapshot{
				Time:     snapshot.Time,
				USDValue: snapshot.USDValue,
				Breakdown: []CurrencyContribution{
					{
						Currency:        f.items[x].currency,
						USDContribution: snapshot.USDValue,
					},
				},
			})
		}

		sort.Slice(pricingOverTime, func(i, j int) bool {
			return pricingOverTime[i].Time.Before(pricingOverTime[j].Time)
		})
		item.Snapshots = pricingOverTime

		if f.items[x].initialFunds.IsZero() {
			item.ShowInfinite = true
		} else {
			item.Difference = f.items[x].available.Sub(f.items[x].initialFunds).Div(f.items[x].initialFunds).Mul(decimal.NewFromInt(100))
		}
		if f.items[x].pairedWith != nil {
			item.PairedWith = f.items[x].pairedWith.currency
		}
		report.InitialFunds = report.InitialFunds.Add(item.USDInitialFunds)

		items[x] = item
	}

	if len(report.USDTotalsOverTime) > 0 {
		sort.Slice(report.USDTotalsOverTime, func(i, j int) bool {
			return report.USDTotalsOverTime[i].Time.Before(report.USDTotalsOverTime[j].Time)
		})
		report.FinalFunds = report.USDTotalsOverTime[len(report.USDTotalsOverTime)-1].USDValue
	}

	report.Items = items
	return &report, nil
}

// Transfer allows transferring funds from one pretend exchange to another
func (f *FundManager) Transfer(amount decimal.Decimal, sender, receiver *Item, inclusiveFee bool) error {
	if sender == nil || receiver == nil {
		return gctcommon.ErrNilPointer
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
func (f *FundManager) GetFundingForEvent(ev common.Event) (IFundingPair, error) {
	return f.getFundingForEAP(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
}

// getFundingForEAP constructs a funding based on the exchange, asset, currency pair
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

// Liquidate will remove all funding for all items belonging to an exchange
func (f *FundManager) Liquidate(ev common.Event) error {
	if ev == nil {
		return fmt.Errorf("%w event", gctcommon.ErrNilPointer)
	}
	for i := range f.items {
		if f.items[i].exchange == ev.GetExchange() {
			f.items[i].reserved = decimal.Zero
			f.items[i].available = decimal.Zero
			f.items[i].isLiquidated = true
		}
	}
	return nil
}

// GetAllFunding returns basic representations of all current
// holdings from the latest point
func (f *FundManager) GetAllFunding() ([]BasicItem, error) {
	result := make([]BasicItem, len(f.items))
	for i := range f.items {
		var usd decimal.Decimal
		if f.items[i].trackingCandles != nil {
			latest, err := f.items[i].trackingCandles.Latest()
			if err != nil {
				return nil, err
			}
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
	return result, nil
}

// UpdateFundingFromLiveData forcefully updates funding from a live source
func (f *FundManager) UpdateFundingFromLiveData(initialFundsSet bool) error {
	exchanges, err := f.exchangeManager.GetExchanges()
	if err != nil {
		return err
	}
	for _, e := range exchanges {
		eName := e.GetName()
		for _, a := range e.GetAssetTypes(false) {
			if a.IsFutures() {
				// we set all holdings as spot
				// futures currency holdings are collateral in the collateral currency
				continue
			}
			subAccts, err := e.UpdateAccountBalances(context.TODO(), a)
			if err != nil {
				return err
			}
			for _, subAcct := range subAccts {
				for _, bal := range subAcct.Balances {
					if err := f.SetFunding(eName, a, &bal, initialFundsSet); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// UpdateAllCollateral will update the collateral values
// of all stored exchanges
func (f *FundManager) UpdateAllCollateral(isLive, initialFundsSet bool) error {
	exchanges, err := f.exchangeManager.GetExchanges()
	if err != nil {
		return err
	}

	for x := range exchanges {
		exchName := strings.ToLower(exchanges[x].GetName())
		exchangeCollateralCalculator := &futures.TotalCollateralCalculator{
			CalculateOffline: !isLive,
		}
		for y := range f.items {
			if f.items[y].exchange != exchName {
				continue
			}
			if f.items[y].asset.IsFutures() {
				// futures positions aren't collateral, they utilise it
				continue
			}
			var usd decimal.Decimal
			if f.items[y].trackingCandles != nil {
				var latest data.Event
				latest, err = f.items[y].trackingCandles.Latest()
				if err != nil {
					return err
				}
				if latest != nil {
					usd = latest.GetClosePrice()
				}
			}
			if usd.IsZero() && exchangeCollateralCalculator.CalculateOffline {
				continue
			}
			side := gctorder.Buy
			if !f.items[y].available.GreaterThan(decimal.Zero) {
				side = gctorder.Sell
			}

			exchangeCollateralCalculator.CollateralAssets = append(exchangeCollateralCalculator.CollateralAssets, futures.CollateralCalculator{
				CalculateOffline:   !isLive,
				CollateralCurrency: f.items[y].currency,
				Asset:              f.items[y].asset,
				Side:               side,
				FreeCollateral:     f.items[y].available,
				LockedCollateral:   f.items[y].reserved,
				USDPrice:           usd,
			})
		}

		var collateral *futures.TotalCollateralResponse
		collateral, err = exchanges[x].CalculateTotalCollateral(context.TODO(), exchangeCollateralCalculator)
		if err != nil {
			return err
		}
		for y := range f.items {
			if f.items[y].exchange == exchName &&
				f.items[y].isCollateral {
				if f.verbose {
					log.Infof(common.FundManager, "Setting collateral %v %v %v to %v", f.items[y].exchange, f.items[y].asset, f.items[y].currency, collateral.AvailableCollateral)
				}
				f.items[y].available = collateral.AvailableCollateral
				if !initialFundsSet {
					f.items[y].initialFunds = collateral.AvailableCollateral
				}
				return nil
			}
		}
	}

	return nil
}

// UpdateCollateralForEvent will recalculate collateral for an exchange
// based on the event passed in
func (f *FundManager) UpdateCollateralForEvent(ev common.Event, isLive bool) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if !f.HasFutures() {
		// no collateral, no need to update
		return nil
	}

	exchMap := make(map[string]exchange.IBotExchange)
	var collateralAmount decimal.Decimal
	var err error
	calculator := futures.TotalCollateralCalculator{
		CalculateOffline: !isLive,
	}

	for i := range f.items {
		if f.items[i].asset.IsFutures() {
			// futures positions aren't collateral, they utilise it
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
			var latest data.Event
			latest, err = f.items[i].trackingCandles.Latest()
			if err != nil {
				return err
			}
			if latest != nil {
				usd = latest.GetClosePrice()
			}
		}
		if usd.IsZero() {
			continue
		}
		side := gctorder.Buy
		if !f.items[i].available.GreaterThan(decimal.Zero) {
			side = gctorder.Sell
		}

		calculator.CollateralAssets = append(calculator.CollateralAssets, futures.CollateralCalculator{
			CalculateOffline:   !isLive,
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
			if f.verbose {
				log.Infof(common.FundManager, "Setting collateral %v %v %v to %v", f.items[i].exchange, f.items[i].asset, f.items[i].currency, collat.AvailableCollateral)
			}
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
func (f *FundManager) HasExchangeBeenLiquidated(ev common.Event) bool {
	for i := range f.items {
		if ev.GetExchange() == f.items[i].exchange {
			return f.items[i].isLiquidated
		}
	}
	return false
}

// SetFunding overwrites a funding setting. This is for live trading
// where external wallet amounts need to be synced
// As external sources may have additional currencies and balances
// versus the strategy currencies, they must be appended to
// help calculate collateral
func (f *FundManager) SetFunding(exchName string, item asset.Item, balance *accounts.Balance, initialFundsSet bool) error {
	if exchName == "" {
		return gctcommon.ErrExchangeNameNotSet
	}
	if !item.IsValid() {
		return asset.ErrNotSupported
	}
	if balance == nil {
		return gctcommon.ErrNilPointer
	}
	if balance.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}

	exchName = strings.ToLower(exchName)
	amount := decimal.NewFromFloat(balance.Total)
	for i := range f.items {
		if f.items[i].asset.IsFutures() {
			continue
		}
		if f.items[i].exchange != exchName ||
			f.items[i].asset != item ||
			!f.items[i].currency.Equal(balance.Currency) {
			continue
		}
		if f.verbose {
			log.Infof(common.FundManager, "Setting %v %v %v balance to %v", exchName, item, balance.Currency, balance.Total)
		}
		if !initialFundsSet {
			f.items[i].initialFunds = amount
		}
		f.items[i].available = amount
		return nil
	}
	if f.verbose {
		log.Debugf(common.FundManager, "Appending balance %v %v %v to %v", exchName, item, balance.Currency, balance.Total)
	}
	f.items = append(f.items, &Item{
		exchange:       exchName,
		asset:          item,
		currency:       balance.Currency,
		initialFunds:   amount,
		available:      amount,
		appendedViaAPI: true,
	})
	return nil
}
