package exchange

import (
	"sync"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// IBotExchange enforces standard functions for all exchanges supported in
// GoCryptoTrader
type IBotExchange interface {
	Setup(exch *config.ExchangeConfig) error
	Start(wg *sync.WaitGroup)
	SetDefaults()
	GetName() string
	IsEnabled() bool
	SetEnabled(bool)
	FetchTicker(currency currency.Pair, assetType assets.AssetType) (ticker.Price, error)
	UpdateTicker(currency currency.Pair, assetType assets.AssetType) (ticker.Price, error)
	FetchOrderbook(currency currency.Pair, assetType assets.AssetType) (orderbook.Base, error)
	UpdateOrderbook(currency currency.Pair, assetType assets.AssetType) (orderbook.Base, error)
	FetchTradablePairs(assetType assets.AssetType) ([]string, error)
	UpdateTradablePairs(forceUpdate bool) error
	GetEnabledPairs(assetType assets.AssetType) currency.Pairs
	GetAvailablePairs(assetType assets.AssetType) currency.Pairs
	GetAccountInfo() (AccountInfo, error)
	GetAuthenticatedAPISupport() bool
	SetPairs(pairs currency.Pairs, assetType assets.AssetType, enabled bool) error
	GetAssetTypes() assets.AssetTypes
	GetExchangeHistory(currencyPair currency.Pair, assetType assets.AssetType) ([]TradeHistory, error)
	SupportsAutoPairUpdates() bool
	SupportsRESTTickerBatchUpdates() bool
	GetFeeByType(feeBuilder *FeeBuilder) (float64, error)
	GetLastPairsUpdateTime() int64
	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	SupportsWithdrawPermissions(permissions uint32) bool
	GetFundingHistory() ([]FundHistory, error)
	SubmitOrder(p currency.Pair, side OrderSide, orderType OrderType, amount, price float64, clientID string) (SubmitOrderResponse, error)
	ModifyOrder(action *ModifyOrder) (string, error)
	CancelOrder(order *OrderCancellation) error
	CancelAllOrders(orders *OrderCancellation) (CancelAllOrdersResponse, error)
	GetOrderInfo(orderID string) (OrderDetail, error)
	GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error)
	GetOrderHistory(getOrdersRequest *GetOrdersRequest) ([]OrderDetail, error)
	GetActiveOrders(getOrdersRequest *GetOrdersRequest) ([]OrderDetail, error)
	WithdrawCryptocurrencyFunds(withdrawRequest *WithdrawRequest) (string, error)
	WithdrawFiatFunds(withdrawRequest *WithdrawRequest) (string, error)
	WithdrawFiatFundsToInternationalBank(withdrawRequest *WithdrawRequest) (string, error)
	SetHTTPClientUserAgent(ua string)
	GetHTTPClientUserAgent() string
	SetClientProxyAddress(addr string) error
	SupportsWebsocket() bool
	SupportsREST() bool
	IsWebsocketEnabled() bool
	GetWebsocket() (*Websocket, error)
	GetDefaultConfig() (*config.ExchangeConfig, error)
}
