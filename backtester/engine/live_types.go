package engine

import (
	"errors"
	"github.com/thrasher-corp/gocryptotrader/exchanges/alert"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
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

	errDataSourceExists = errors.New("data source already exists")
)

var (
	defaultEventTimeout      = time.Minute
	defaultDataCheckInterval = time.Second
)

// Handler is all the functionality required in order to
// run a backtester with live data
type Handler interface {
	AppendDataSource(exch gctexchange.IBotExchange, interval gctkline.Interval, item asset.Item, curr, underlying currency.Pair, dataType int64) error
	FetchLatestData() (bool, error)
	Start() error
	IsRunning() bool
	DataFetcher() error
	Stop() error
	Reset()
	Updated() <-chan bool
}

// DataChecker is responsible for managing all data retrieval
// for a live data option
type DataChecker struct {
	m                 sync.Mutex
	wg                sync.WaitGroup
	started           uint32
	verbose           bool
	exchangeManager   *engine.ExchangeManager
	exchangesToCheck  []*liveExchangeDataHandler
	eventTimeout      time.Duration
	dataCheckInterval time.Duration
	dataHolder        data.Holder
	notice            alert.Notice
	shutdown          chan struct{}
	report            report.Handler
	funding           funding.IFundingManager
}

type liveExchangeDataHandler struct {
	m              sync.Mutex
	exchange       gctexchange.IBotExchange
	exchangeName   string
	asset          asset.Item
	pair           currency.Pair
	underlyingPair currency.Pair
	pairCandles    kline.DataFromKline
	dataType       int64
}
