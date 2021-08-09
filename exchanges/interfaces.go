package exchange

import (
	"context"
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

	APIManager

	GetEnabledPairs(a asset.Item) (currency.Pairs, error)
	GetAvailablePairs(a asset.Item) (currency.Pairs, error)

	GetAuthenticatedAPISupport(endpoint uint8) bool
	SetPairs(pairs currency.Pairs, a asset.Item, enabled bool) error
	GetAssetTypes(enabled bool) asset.Items

	SupportsAutoPairUpdates() bool
	SupportsRESTTickerBatchUpdates() bool
	GetLastPairsUpdateTime() int64
	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	SupportsWithdrawPermissions(permissions uint32) bool

	SetHTTPClientUserAgent(ua string)
	GetHTTPClientUserAgent() string
	SetClientProxyAddress(addr string) error
	SupportsREST() bool
	GetSubscriptions() ([]stream.ChannelSubscription, error)
	GetDefaultConfig() (*config.ExchangeConfig, error)
	GetBase() *Base
	SupportsAsset(assetType asset.Item) bool

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

// APIManager defines required API management functionality for exchange
// implementations.
type APIManager interface {
	ValidateCredentials(ctx context.Context, a asset.Item) error

	FetchAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error)
	UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error)

	FetchTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error)

	FetchOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error)
	UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error)

	FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error)
	UpdateTradablePairs(ctx context.Context, forceUpdate bool) error

	GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error)
	GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error)

	GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, accountID string) (string, error)

	GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error)
	GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error)

	SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error)
	ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error)

	CancelOrder(ctx context.Context, o *order.Cancel) error
	CancelAllOrders(ctx context.Context, orders *order.Cancel) (order.CancelAllResponse, error)

	CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error)

	GetWithdrawalsHistory(ctx context.Context, code currency.Code) ([]WithdrawalHistory, error)
	GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error)

	WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)

	GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error)
	GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error)

	GetFundingHistory(ctx context.Context) ([]FundHistory, error)

	GetFeeByType(ctx context.Context, f *FeeBuilder) (float64, error)
}
