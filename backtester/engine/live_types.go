package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	// ErrLiveDataTimeout returns when an event has not been processed within the timeframe
	ErrLiveDataTimeout = errors.New("no data processed within timeframe")

	errDataSourceExists             = errors.New("data source already exists")
	errNoCredsNoLive                = errors.New("cannot use real orders without credentials to fulfil those real orders")
	errNoDataSetForClosingPositions = errors.New("no data was set for closing positions")
	errNilError                     = errors.New("nil error received when expecting an error")
)

var (
	defaultEventTimeout              = time.Minute
	defaultDataCheckInterval         = time.Second
	defaultDataRetryAttempts   int64 = 1
	defaultDataRequestWaitTime       = time.Millisecond * 500
)

// Handler is all the functionality required in order to
// run a backtester with live data
type Handler interface {
	AppendDataSource(*liveDataSourceSetup) error
	FetchLatestData() (bool, error)
	Start() error
	IsRunning() bool
	DataFetcher() error
	Stop() error
	Reset() error
	Updated() chan bool
	HasShutdown() chan bool
	HasShutdownFromError() chan bool
	SetDataForClosingAllPositions(events ...signal.Event) error
	UpdateFunding(force bool) error
	IsRealOrders() bool
}

// dataChecker is responsible for managing all data retrieval
// for a live data option
type dataChecker struct {
	m                 sync.Mutex
	wg                sync.WaitGroup
	started           uint32
	updatingFunding   uint32
	verboseDataCheck  bool
	realOrders        bool
	hasUpdatedFunding bool
	exchangeManager   *engine.ExchangeManager
	sourcesToCheck    []*liveDataSourceDataHandler
	eventTimeout      time.Duration
	dataCheckInterval time.Duration
	dataHolder        data.Holder
	shutdownErr       chan bool
	shutdown          chan bool
	dataUpdated       chan bool
	report            report.Handler
	funding           funding.IFundingManager
}

// liveDataSourceSetup is used to add new data sources
// to retrieve live data
type liveDataSourceSetup struct {
	exchange                  gctexchange.IBotExchange
	interval                  gctkline.Interval
	asset                     asset.Item
	pair                      currency.Pair
	underlyingPair            currency.Pair
	dataType                  int64
	dataRequestRetryTolerance int64
	dataRequestRetryWaitTime  time.Duration
	verboseExchangeRequest    bool
}

// liveDataSourceDataHandler is used to collect
// and store live data
type liveDataSourceDataHandler struct {
	exchange                  gctexchange.IBotExchange
	exchangeName              string
	asset                     asset.Item
	pair                      currency.Pair
	underlyingPair            currency.Pair
	dataType                  int64
	pairCandles               *kline.DataFromKline
	processedData             map[int64]struct{}
	candlesToAppend           *gctkline.Item
	dataRequestRetryTolerance int64
	dataRequestRetryWaitTime  time.Duration
	verboseExchangeRequest    bool
}
