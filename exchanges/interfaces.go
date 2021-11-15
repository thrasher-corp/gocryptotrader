package exchange

import (
	"context"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
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
	Setup(exch *config.Exchange) error
	Start(wg *sync.WaitGroup) error
	SetDefaults()
	GetName() string
	IsEnabled() bool
	SetEnabled(bool)
	ValidateCredentials(ctx context.Context, a asset.Item) error
	FetchTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTickers(ctx context.Context, a asset.Item) error
	FetchOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error)
	UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error)
	FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error)
	UpdateTradablePairs(ctx context.Context, forceUpdate bool) error
	GetEnabledPairs(a asset.Item) (currency.Pairs, error)
	GetAvailablePairs(a asset.Item) (currency.Pairs, error)
	FetchAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error)
	UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error)
	GetAuthenticatedAPISupport(endpoint uint8) bool
	SetPairs(pairs currency.Pairs, a asset.Item, enabled bool) error
	GetAssetTypes(enabled bool) asset.Items
	GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error)
	GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error)
	SupportsAutoPairUpdates() bool
	SupportsRESTTickerBatchUpdates() bool
	GetFeeByType(ctx context.Context, f *FeeBuilder) (float64, error)
	GetLastPairsUpdateTime() int64
	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	SupportsWithdrawPermissions(permissions uint32) bool
	GetFundingHistory(ctx context.Context) ([]FundHistory, error)
	SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error)
	ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error)
	CancelOrder(ctx context.Context, o *order.Cancel) error
	CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error)
	CancelAllOrders(ctx context.Context, orders *order.Cancel) (order.CancelAllResponse, error)
	GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error)
	GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, accountID, chain string) (*deposit.Address, error)
	GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error)
	GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error)
	GetWithdrawalsHistory(ctx context.Context, code currency.Code) ([]WithdrawalHistory, error)
	GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error)
	WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	SetHTTPClientUserAgent(ua string)
	GetHTTPClientUserAgent() string
	SetClientProxyAddress(addr string) error
	SupportsREST() bool
	GetSubscriptions() ([]stream.ChannelSubscription, error)
	GetDefaultConfig() (*config.Exchange, error)
	GetBase() *Base
	SupportsAsset(assetType asset.Item) bool
	GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error)
	GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error)
	DisableRateLimiter() error
	EnableRateLimiter() error
	CalculatePNL(*PNLCalculator) (*PNLResult, error)
	CurrencyStateManagement

	GetWebsocket() (*stream.Websocket, error)
	IsWebsocketEnabled() bool
	SupportsWebsocket() bool
	SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error
	UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error
	IsAssetWebsocketSupported(aType asset.Item) bool
	FlushWebsocketChannels() error
	AuthenticateWebsocket(ctx context.Context) error

	GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (*order.Limits, error)
	CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error
	UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error
}

// CurrencyStateManagement defines functionality for currency state management
type CurrencyStateManagement interface {
	GetCurrencyStateSnapshot() ([]currencystate.Snapshot, error)
	UpdateCurrencyStates(ctx context.Context, a asset.Item) error
	CanTradePair(p currency.Pair, a asset.Item) error
	CanTrade(c currency.Code, a asset.Item) error
	CanWithdraw(c currency.Code, a asset.Item) error
	CanDeposit(c currency.Code, a asset.Item) error
}
