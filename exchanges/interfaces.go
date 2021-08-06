package exchange

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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

	FetchTicker(p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTicker(p currency.Pair, a asset.Item) (*ticker.Price, error)
	FetchOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error)
	UpdateOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error)
	FetchTradablePairs(a asset.Item) ([]string, error)
	UpdateTradablePairs(forceUpdate bool) error
	GetEnabledPairs(a asset.Item) (currency.Pairs, error)
	GetAvailablePairs(a asset.Item) (currency.Pairs, error)

	// GetAccounts returns the different account names associated with the
	// supplied credentials
	GetAccounts() ([]account.Designation, error)
	// FetchAccountInfo initially fetches account info, if not found will
	// execute UpdateAccountInfo
	FetchAccountInfo(a account.Designation, assetType asset.Item) (account.HoldingsSnapshot, error)
	// UpdateAccountInfo specifically fetches and updates account holdings from
	// the exchange
	UpdateAccountInfo(a account.Designation, assetType asset.Item) (account.HoldingsSnapshot, error)
	// GetFullAccountSnapshot returns a full snapshot of all accounts associated
	// with supplied credentials
	GetFullAccountSnapshot() (account.FullSnapshot, error)
	// AccountValid verifies if the account supplied is valid
	AccountValid(a account.Designation) error
	// ClaimAccountFunds allows for a strategy or sub-system to claim on a specific account
	// holding associated with the supplied credentials. If totalRequired param
	// is false will allow the claim of less than or equal to the request amount
	// in the event multiple strategies are working on the same holdings.
	ClaimAccountFunds(a account.Designation, assetType asset.Item, c currency.Code, amount float64, totalRequired bool) (*account.Claim, error)

	GetAuthenticatedAPISupport(endpoint uint8) bool
	SetPairs(pairs currency.Pairs, a asset.Item, enabled bool) error
	GetAssetTypes(enabled bool) asset.Items
	GetRecentTrades(p currency.Pair, a asset.Item) ([]trade.Data, error)
	GetHistoricTrades(p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error)
	SupportsAutoPairUpdates() bool
	SupportsRESTTickerBatchUpdates() bool
	GetFeeByType(f *FeeBuilder) (float64, error)
	GetLastPairsUpdateTime() int64
	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	SupportsWithdrawPermissions(permissions uint32) bool
	GetFundingHistory() ([]FundHistory, error)
	SubmitOrder(s *order.Submit) (order.SubmitResponse, error)
	ModifyOrder(action *order.Modify) (order.Modify, error)
	CancelOrder(o *order.Cancel) error
	CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error)
	CancelAllOrders(orders *order.Cancel) (order.CancelAllResponse, error)
	GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error)
	GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error)
	GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error)
	GetWithdrawalsHistory(code currency.Code) ([]WithdrawalHistory, error)
	GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error)
	WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	SetHTTPClientUserAgent(ua string)
	GetHTTPClientUserAgent() string
	SetClientProxyAddress(addr string) error
	SupportsREST() bool
	GetSubscriptions() ([]stream.ChannelSubscription, error)
	GetDefaultConfig() (*config.ExchangeConfig, error)
	GetBase() *Base
	SupportsAsset(assetType asset.Item) bool
	GetHistoricCandles(p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error)
	GetHistoricCandlesExtended(p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error)
	DisableRateLimiter() error
	EnableRateLimiter() error
	// Websocket specific wrapper functionality
	// GetWebsocket returns a pointer to the websocket
	GetWebsocket() (*stream.Websocket, error)
	IsWebsocketEnabled() bool
	SupportsWebsocket() bool
	SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error
	UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error
	IsAssetWebsocketSupported(aType asset.Item) bool
	// FlushWebsocketChannels checks and flushes subscriptions if there is a
	// pair,asset, url/proxy or subscription change
	FlushWebsocketChannels() error
	AuthenticateWebsocket() error
	// Exchange order related execution limits
	GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (*order.Limits, error)
	CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error
	UpdateOrderExecutionLimits(a asset.Item) error
}
