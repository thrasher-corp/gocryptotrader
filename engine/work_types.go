package engine

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
	"golang.org/x/time/rate"
)

// Priority constants determines API function prirority. This allows
// reorganisation to occur on job queue heap e.g. priority1 for all order
// operations
const (
	// Cancel is lowest priority so we execute higher priority jobs under heavy
	// work loads and cleanup remnants when workloads drop
	cancel Priority = iota
	low
	medium
	high
	extreme

	defaultWorkerCount = int32(10)
)

var (
	errWorkManagerStopped = errors.New("work manager has stopped")
	errWorkManagerStarted = errors.New("work manager already started")
	errExchangNotFound    = errors.New("exchange not found")
	errJobCancelled       = errors.New("job cancelled")
)

// WorkManager defines a prioritised job queue manager for generalised API calls
// that will also act as a security layer i.e. general exchange rate limits and
// client call permission sets
type WorkManager struct {
	Jobs    PriorityJobQueue
	jobsMtx sync.Mutex

	workAvailable chan struct{}

	shutdown chan struct{}
	p        *sync.Pool

	wg          sync.WaitGroup
	workerCount int32
	started     int32
	running     int32
	verbose     bool
}

// Priority defines an explicit priority level
type Priority int

// Exchange couples a calling systems intended trading API functionality with
// an exchange
type Exchange struct {
	e  exchange.IBotExchange
	wm *WorkManager
}

// Command wraps execute functionality for our priority work queue
type Command interface {
	Execute()
	GetReservation() *rate.Reservation
	// IsCancelled() bool
}

// FetchTicker defines a coupler to an exchange REST request
type FetchTicker struct {
	exchange.IBotExchange
	Pair        currency.Pair
	Asset       asset.Item
	Price       *ticker.Price
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// UpdateTicker defines a coupler to an exchange REST request
type UpdateTicker struct {
	exchange.IBotExchange
	Pair        currency.Pair
	Asset       asset.Item
	Price       *ticker.Price
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// FetchOrderbook defines a coupler to an exchange REST request
type FetchOrderbook struct {
	exchange.IBotExchange
	Pair        currency.Pair
	Asset       asset.Item
	Orderbook   *orderbook.Base
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// UpdateOrderbook defines a coupler to an exchange REST request
type UpdateOrderbook struct {
	exchange.IBotExchange
	Pair        currency.Pair
	Asset       asset.Item
	Orderbook   *orderbook.Base
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetAccountInfo defines a coupler to an exchange REST request
type GetAccountInfo struct {
	exchange.IBotExchange
	AccountInfo exchange.AccountInfo
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetExchangeHistory defines a coupler to an exchange REST request
type GetExchangeHistory struct {
	exchange.IBotExchange
	Request     *exchange.TradeHistoryRequest
	Response    []exchange.TradeHistory
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetFeeByType defines a coupler to an exchange REST request
type GetFeeByType struct {
	exchange.IBotExchange
	Request     *exchange.FeeBuilder
	Response    float64
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetFundingHistory defines a coupler to an exchange REST request
type GetFundingHistory struct {
	exchange.IBotExchange
	Response    []exchange.FundHistory
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// SubmitOrder defines a coupler to an exchange REST request
type SubmitOrder struct {
	exchange.IBotExchange
	Request     *order.Submit
	Response    order.SubmitResponse
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// ModifyOrder defines a coupler to an exchange REST request
type ModifyOrder struct {
	exchange.IBotExchange
	Request     *order.Modify
	Response    string
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// CancelOrder defines a coupler to an exchange REST request
type CancelOrder struct {
	exchange.IBotExchange
	Request     *order.Cancel
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// CancelAllOrders defines a coupler to an exchange REST request
type CancelAllOrders struct {
	exchange.IBotExchange
	Request     *order.Cancel
	Response    order.CancelAllResponse
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetOrderInfo defines a coupler to an exchange REST request
type GetOrderInfo struct {
	exchange.IBotExchange
	Request     string
	Response    order.Detail
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetDepositAddress defines a coupler to an exchange REST request
type GetDepositAddress struct {
	exchange.IBotExchange
	Crypto      currency.Code
	AccountID   string
	Response    string
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetOrderHistory defines a coupler to an exchange REST request
type GetOrderHistory struct {
	exchange.IBotExchange
	Request     *order.GetOrdersRequest
	Response    []order.Detail
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// GetActiveOrders defines a coupler to an exchange REST request
type GetActiveOrders struct {
	exchange.IBotExchange
	Request     *order.GetOrdersRequest
	Response    []order.Detail
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// WithdrawCryptocurrencyFunds defines a coupler to an exchange REST request
type WithdrawCryptocurrencyFunds struct {
	exchange.IBotExchange
	Request     *withdraw.CryptoRequest
	Response    string
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// WithdrawFiatFunds defines a coupler to an exchange REST request
type WithdrawFiatFunds struct {
	exchange.IBotExchange
	Request     *withdraw.FiatRequest
	Response    string
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}

// WithdrawFiatFundsToInternationalBank defines a coupler to an exchange REST
// request
type WithdrawFiatFundsToInternationalBank struct {
	exchange.IBotExchange
	Request     *withdraw.FiatRequest
	Response    string
	Reservation *rate.Reservation
	CancelMe    chan int
	Error       error
}
