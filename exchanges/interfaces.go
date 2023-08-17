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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
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
	Start(ctx context.Context, wg *sync.WaitGroup) error
	SetDefaults()
	Shutdown() error
	GetName() string
	SetEnabled(bool)
	FetchTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTickers(ctx context.Context, a asset.Item) error
	FetchOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error)
	UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error)
	FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error)
	UpdateTradablePairs(ctx context.Context, forceUpdate bool) error
	GetEnabledPairs(a asset.Item) (currency.Pairs, error)
	GetAvailablePairs(a asset.Item) (currency.Pairs, error)
	SetPairs(pairs currency.Pairs, a asset.Item, enabled bool) error
	GetAssetTypes(enabled bool) asset.Items
	GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error)
	GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error)
	GetFeeByType(ctx context.Context, f *FeeBuilder) (float64, error)
	GetLastPairsUpdateTime() int64
	GetWithdrawPermissions() uint32
	FormatWithdrawPermissions() string
	GetAccountFundingHistory(ctx context.Context) ([]FundingHistory, error)
	GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, accountID, chain string) (*deposit.Address, error)
	GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error)
	GetWithdrawalsHistory(ctx context.Context, code currency.Code, a asset.Item) ([]WithdrawalHistory, error)
	WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error)
	SetHTTPClientUserAgent(ua string) error
	GetHTTPClientUserAgent() (string, error)
	SetClientProxyAddress(addr string) error
	GetDefaultConfig(ctx context.Context) (*config.Exchange, error)
	GetBase() *Base
	GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error)
	GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error)
	DisableRateLimiter() error
	EnableRateLimiter() error
	GetServerTime(ctx context.Context, ai asset.Item) (time.Time, error)
	GetWebsocket() (*stream.Websocket, error)
	SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error
	UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error
	GetSubscriptions() ([]stream.ChannelSubscription, error)
	FlushWebsocketChannels() error
	AuthenticateWebsocket(ctx context.Context) error
	GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (order.MinMaxLevel, error)
	CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error
	UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error
	GetCredentials(ctx context.Context) (*account.Credentials, error)

	// ValidateAPICredentials function validates the API keys by sending an
	// authenticated REST request. See exchange specific wrapper implementation.
	ValidateAPICredentials(ctx context.Context, a asset.Item) error
	// VerifyAPICredentials determines if the credentials supplied have unset
	// required values. See exchanges/credentials.go Base method for
	// implementation.
	VerifyAPICredentials(creds *account.Credentials) error
	// GetDefaultCredentials returns the exchange.Base api credentials loaded by
	// config.json. See exchanges/credentials.go Base method for implementation.
	GetDefaultCredentials() *account.Credentials

	FunctionalityChecker
	AccountManagement
	OrderManagement
	CurrencyStateManagement
	FuturesManagement
}

// OrderManagement defines functionality for order management
type OrderManagement interface {
	SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error)
	ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error)
	CancelOrder(ctx context.Context, o *order.Cancel) error
	CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error)
	CancelAllOrders(ctx context.Context, orders *order.Cancel) (order.CancelAllResponse, error)
	GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error)
	GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error)
	GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error)
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

// AccountManagement defines functionality for exchange account management
type AccountManagement interface {
	UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error)
	FetchAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error)
	HasAssetTypeAccountSegregation() bool
}

// FunctionalityChecker defines functionality for retrieving exchange
// support/enabled features
type FunctionalityChecker interface {
	IsEnabled() bool
	IsAssetWebsocketSupported(a asset.Item) bool
	SupportsAsset(assetType asset.Item) bool
	SupportsREST() bool
	SupportsWithdrawPermissions(permissions uint32) bool
	SupportsRESTTickerBatchUpdates() bool
	IsWebsocketEnabled() bool
	SupportsWebsocket() bool
	SupportsAutoPairUpdates() bool
	IsWebsocketAuthenticationSupported() bool
	IsRESTAuthenticationSupported() bool
}

// FuturesManagement manages futures orders, pnl and collateral calculations
type FuturesManagement interface {
	GetPositionSummary(context.Context, *order.PositionSummaryRequest) (*order.PositionSummary, error)
	ScaleCollateral(ctx context.Context, calculator *order.CollateralCalculator) (*order.CollateralByCurrency, error)
	CalculateTotalCollateral(context.Context, *order.TotalCollateralCalculator) (*order.TotalCollateralResponse, error)
	GetFuturesPositions(context.Context, *order.PositionsRequest) ([]order.PositionDetails, error)
	GetFundingRates(context.Context, *fundingrate.RatesRequest) (*fundingrate.Rates, error)
	GetLatestFundingRate(context.Context, *fundingrate.LatestRateRequest) (*fundingrate.LatestRateResponse, error)
	IsPerpetualFutureCurrency(asset.Item, currency.Pair) (bool, error)
	GetCollateralCurrencyForContract(asset.Item, currency.Pair) (currency.Code, asset.Item, error)
	GetMarginRatesHistory(context.Context, *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error)
	order.PNLCalculation
}
