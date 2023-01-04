package portfolio

import (
	"strings"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Setup creates a portfolio manager instance and sets private fields
func Setup(sh SizeHandler, r risk.Handler, riskFreeRate decimal.Decimal) (*Portfolio, error) {
	if sh == nil {
		return nil, errSizeManagerUnset
	}
	if riskFreeRate.IsNegative() {
		return nil, errNegativeRiskFreeRate
	}
	if r == nil {
		return nil, errRiskManagerUnset
	}
	p := &Portfolio{}
	p.sizeManager = sh
	p.riskManager = r
	p.riskFreeRate = riskFreeRate

	return p, nil
}

// Reset returns the portfolio manager to its default state
func (p *Portfolio) Reset() error {
	if p == nil {
		return gctcommon.ErrNilPointer
	}
	p.exchangeAssetPairPortfolioSettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	p.riskFreeRate = decimal.Zero
	p.sizeManager = nil
	p.riskManager = nil
	return nil
}

// SetCurrencySettingsMap ensures a map is created and no panics happen
func (p *Portfolio) SetCurrencySettingsMap(setup *exchange.Settings) error {
	if setup == nil {
		return errNoPortfolioSettings
	}
	if setup.Exchange == nil {
		return errExchangeUnset
	}
	if setup.Asset == asset.Empty {
		return errAssetUnset
	}
	if setup.Pair.IsEmpty() {
		return errCurrencyPairUnset
	}

	if p.exchangeAssetPairPortfolioSettings == nil {
		p.exchangeAssetPairPortfolioSettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	}
	name := strings.ToLower(setup.Exchange.GetName())
	m, ok := p.exchangeAssetPairPortfolioSettings[name]
	if !ok {
		m = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
		p.exchangeAssetPairPortfolioSettings[name] = m
	}
	m2, ok := m[setup.Asset]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]*Settings)
		m[setup.Asset] = m2
	}
	m3, ok := m2[setup.Pair.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]*Settings)
		m2[setup.Pair.Base.Item] = m3
	}
	settings := &Settings{
		Exchange:          setup.Exchange,
		exchangeName:      name,
		assetType:         setup.Asset,
		pair:              setup.Pair,
		BuySideSizing:     setup.BuySide,
		SellSideSizing:    setup.SellSide,
		Leverage:          setup.Leverage,
		HoldingsSnapshots: make(map[int64]*holdings.Holding),
	}
	if setup.Asset.IsFutures() {
		collateralCurrency, _, err := setup.Exchange.GetCollateralCurrencyForContract(setup.Asset, setup.Pair)
		if err != nil {
			return err
		}
		futureTrackerSetup := &gctorder.MultiPositionTrackerSetup{
			Exchange:                  name,
			Asset:                     setup.Asset,
			Pair:                      setup.Pair,
			Underlying:                setup.Pair.Base,
			OfflineCalculation:        true,
			UseExchangePNLCalculation: setup.UseExchangePNLCalculation,
			CollateralCurrency:        collateralCurrency,
		}
		if setup.UseExchangePNLCalculation {
			futureTrackerSetup.ExchangePNLCalculation = setup.Exchange
		}
		var tracker *gctorder.MultiPositionTracker
		tracker, err = gctorder.SetupMultiPositionTracker(futureTrackerSetup)
		if err != nil {
			return err
		}
		settings.FuturesTracker = tracker
	}
	m3[setup.Pair.Quote.Item] = settings
	return nil
}
