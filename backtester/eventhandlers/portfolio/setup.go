package portfolio

import (
	"strings"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
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
func (p *Portfolio) Reset() {
	if p == nil {
		return
	}
	p.exchangeAssetPairSettings = nil
}

// SetupCurrencySettingsMap ensures a map is created and no panics happen
func (p *Portfolio) SetupCurrencySettingsMap(setup *exchange.Settings) error {
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
	if p.exchangeAssetPairSettings == nil {
		p.exchangeAssetPairSettings = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	}
	name := strings.ToLower(setup.Exchange.GetName())
	if p.exchangeAssetPairSettings[name] == nil {
		p.exchangeAssetPairSettings[name] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*Settings)
	}
	if p.exchangeAssetPairSettings[name][setup.Asset] == nil {
		p.exchangeAssetPairSettings[name][setup.Asset] = make(map[*currency.Item]map[*currency.Item]*Settings)
	}
	if p.exchangeAssetPairSettings[name][setup.Asset][setup.Pair.Base.Item] == nil {
		p.exchangeAssetPairSettings[name][setup.Asset][setup.Pair.Base.Item] = make(map[*currency.Item]*Settings)
	}
	if _, ok := p.exchangeAssetPairSettings[name][setup.Asset][setup.Pair.Base.Item][setup.Pair.Quote.Item]; ok {
		return nil
	}
	collateralCurrency, _, err := setup.Exchange.GetCollateralCurrencyForContract(setup.Asset, setup.Pair)
	if err != nil {
		return err
	}
	settings := &Settings{
		BuySideSizing:     setup.BuySide,
		SellSideSizing:    setup.SellSide,
		Leverage:          setup.Leverage,
		Exchange:          setup.Exchange,
		ComplianceManager: compliance.Manager{},
	}
	if setup.Asset.IsFutures() {
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
	p.exchangeAssetPairSettings[name][setup.Asset][setup.Pair.Base.Item][setup.Pair.Quote.Item] = settings
	return nil
}
