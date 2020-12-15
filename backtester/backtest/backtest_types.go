package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// BackTest is the main hodler of all backtesting
type BackTest struct {
	Bot *engine.Engine

	shutdown   chan struct{}
	Datas      data.Holder
	Strategy   strategies.Handler
	Portfolio  portfolio.Handler
	Exchange   exchange.ExecutionHandler
	Statistic  statistics.Handler
	EventQueue eventholder.EventHolder
	Reports    report.Handler
}

// UltimateHolderOfAllThings is to hold all specific currency pair related things in one location.
type UltimateHolderOfAllThings struct {
	Hi map[string]map[asset.Item]map[currency.Pair]*AllTheThings
}

// AllTheThings conceptually holds all data related to specific currencies
// it is a way of passing related data to each handler as required
//
// This is to minimise the amount of maps used throughout the entire backtester
type AllTheThings struct {
	Data                      data.Handler
	Holdings                  holdings.Snapshots
	Compliance                compliance.Manager
	Events                    currencystatstics.CurrencyStatistic
	ExchangeAssetPairSettings portfolio.ExchangeAssetPairSettings
	RiskSettings              risk.Settings
}

var hasHandledAnEvent bool
