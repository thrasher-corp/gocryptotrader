package engine

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/engine"
)

var (
	errNilConfig           = errors.New("unable to setup backtester with nil config")
	errAmbiguousDataSource = errors.New("ambiguous settings received. Only one data type can be set")
	errNoDataSource        = errors.New("no data settings set in config")
	errIntervalUnset       = errors.New("candle interval unset")
	errUnhandledDatatype   = errors.New("unhandled datatype")
	errNilData             = errors.New("nil data received")
	errLiveOnly            = errors.New("close all positions is only supported by live data type")
)

// BackTest is the main holder of all backtesting functionality
type BackTest struct {
	verbose                  bool
	hasProcessedAnEvent      bool
	hasShutdown              bool
	shutdown                 chan struct{}
	DataHolder               data.Holder
	LiveDataHandler          Handler
	Strategy                 strategies.Handler
	Portfolio                portfolio.Handler
	Exchange                 exchange.ExecutionHandler
	Statistic                statistics.Handler
	EventQueue               eventholder.EventHolder
	Reports                  report.Handler
	Funding                  funding.IFundingManager
	exchangeManager          *engine.ExchangeManager
	orderManager             *engine.OrderManager
	databaseManager          *engine.DatabaseConnectionManager
	hasProcessedDataAtOffset map[int64]bool
}
