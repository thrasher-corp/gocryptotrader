package exchange

import (
	"sync"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// IBotExchange enforces standard functions for all exchanges supported in
// GoCryptoTrader
type IBotExchange interface {
	Setup(exch config.ExchangeConfig)
	Start(wg *sync.WaitGroup)
	SetDefaults()
	GetName() string
	IsEnabled() bool
	SetEnabled(bool)
	FetchTicker(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	UpdateTicker(currency pair.CurrencyPair, assetType string) (ticker.Price, error)
	FetchOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	UpdateOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error)
	GetEnabledCurrencies() []pair.CurrencyPair
	GetAvailableCurrencies() []pair.CurrencyPair
	GetExchangeAccountInfo() (AccountInfo, error)
	GetAuthenticatedAPISupport() bool
	SetCurrencies(pairs []pair.CurrencyPair, enabledPairs bool) error
	GetAssetTypes() []string
	GetExchangeHistory(pair.CurrencyPair, string) ([]TradeHistory, error)
	SupportsAutoPairUpdates() bool
	SupportsRESTTickerBatchUpdates() bool
	GetLastPairsUpdateTime() int64

	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	SupportsWithdrawPermissions(permissions uint32) bool

	GetExchangeFundTransferHistory() ([]FundHistory, error)
	SubmitExchangeOrder(p pair.CurrencyPair, side OrderSide, orderType OrderType, amount, price float64, clientID string) (int64, error)
	ModifyExchangeOrder(orderID int64, modify ModifyOrder) (int64, error)
	CancelExchangeOrder(orderID int64) error
	CancelAllExchangeOrders() error
	GetExchangeOrderInfo(orderID int64) (OrderDetail, error)
	GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error)

	WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error)
	WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error)

	SetHTTPClientUserAgent(ua string)
	GetHTTPClientUserAgent() string
	SetClientProxyAddress(addr string) error

	SupportsWebsocket() bool
	SupportsREST() bool
	IsWebsocketEnabled() bool
	GetWebsocket() (*Websocket, error)
}
