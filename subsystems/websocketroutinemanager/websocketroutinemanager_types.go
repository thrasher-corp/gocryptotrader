package websocketroutinemanager

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

type Manager struct {
	started         int32
	verbose         bool
	exchangeManager iExchangeManager
	orderManager    iOrderManager
	syncer          iCurrencyPairSyncer
	currencyConfig  *config.CurrencyConfig
	shutdown        chan struct{}
	wg              sync.WaitGroup
}

type iExchangeManager interface {
	GetExchanges() []exchange.IBotExchange
}

type iCurrencyPairSyncer interface {
	IsRunning() bool
	PrintTickerSummary(*ticker.Price, string, error)
	PrintOrderbookSummary(*orderbook.Base, string, error)
	Update(string, currency.Pair, asset.Item, int, error) error
}

type iOrderManager interface {
	Exists(*order.Detail) bool
	Add(*order.Detail) error
	Cancel(*order.Cancel) error
	GetByExchangeAndID(string, string) (*order.Detail, error)
}
