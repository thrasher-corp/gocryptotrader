package subsystem

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

const (
	// MsgStarting message to return when subsystem is starting up
	MsgStarting = "starting..."
	// MsgStarted message to return when subsystem has started
	MsgStarted = "started."
	// MsgShuttingDown message to return when a subsystem is shutting down
	MsgShuttingDown = "shutting down..."
	// MsgShutdown message to return when a subsystem has shutdown
	MsgShutdown = "shutdown."
)

var (
	// ErrAlreadyStarted message to return when a subsystem is already started
	ErrAlreadyStarted = errors.New("subsystem already started")
	// ErrNotStarted message to return when subsystem not started
	ErrNotStarted = errors.New("subsystem not started")
	// ErrNil is returned when a subsystem hasn't had its Setup() func run
	ErrNil                          = errors.New("subsystem not setup")
	ErrNilWaitGroup                 = errors.New("nil wait group received")
	ErrNilExchangeManager           = errors.New("cannot start with nil exchange manager")
	ErrNilDatabaseConnectionManager = errors.New("cannot start with nil database connection manager")
	ErrNilConfig                    = errors.New("received nil config")
)

// ExchangeManager limits exposure of accessible functions to exchange manager
// so that subsystems can use some functionality
type ExchangeManager interface {
	GetExchanges() ([]exchange.IBotExchange, error)
	GetExchangeByName(string) (exchange.IBotExchange, error)
}

// CommsManager limits exposure of accessible functions to communication manager
type CommsManager interface {
	PushEvent(evt base.Event)
}

// OrderManager defines a limited scoped order manager
type OrderManager interface {
	Exists(*order.Detail) bool
	Add(*order.Detail) error
	Cancel(context.Context, *order.Cancel) error
	GetByExchangeAndID(string, string) (*order.Detail, error)
	UpdateExistingOrder(*order.Detail) error
}

// PortfolioManager limits exposure of accessible functions to portfolio manager
type PortfolioManager interface {
	GetPortfolioSummary() portfolio.Summary
	IsWhiteListed(string) bool
	IsExchangeSupported(string, string) bool
}

// Bot limits exposure of accessible functions to engine bot
type Bot interface {
	SetupExchanges() error
}

// CurrencyPairSyncer defines a limited scoped currency pair syncer
type CurrencyPairSyncer interface {
	IsRunning() bool
	PrintTickerSummary(*ticker.Price, string, error)
	PrintOrderbookSummary(*orderbook.Base, string, error)
	Update(exchangeName, protocol string, pair currency.Pair, a asset.Item, syncType int, incomingErr error) error
}

// DatabaseConnectionManager defines a limited scoped databaseConnectionManager
type DatabaseConnectionManager interface {
	GetInstance() database.IDatabase
}
