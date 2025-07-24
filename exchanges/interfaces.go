package exchange

import (
	"context"
	"text/template"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// IBotExchange enforces standard functions for all exchanges supported in
// GoCryptoTrader
type IBotExchange interface {
	Setup(exch *config.Exchange) error
	Bootstrap(context.Context) (continueBootstrap bool, err error)
	SetDefaults()
	Shutdown() error
	GetName() string
	SetEnabled(bool)

	GetEnabledFeatures() FeaturesEnabled
	GetSupportedFeatures() FeaturesSupported
	// GetTradingRequirements returns trading requirements for the exchange
	GetTradingRequirements() protocol.TradingRequirements

	GetCachedTicker(p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error)
	UpdateTickers(ctx context.Context, a asset.Item) error
	GetCachedOrderbook(p currency.Pair, a asset.Item) (*orderbook.Book, error)
	UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error)
	FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error)
	UpdateTradablePairs(ctx context.Context, forceUpdate bool) error
	GetEnabledPairs(a asset.Item) (currency.Pairs, error)
	GetAvailablePairs(a asset.Item) (currency.Pairs, error)
	GetPairFormat(asset.Item, bool) (currency.PairFormat, error)
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
	GetBase() *Base
	GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error)
	GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error)
	DisableRateLimiter() error
	EnableRateLimiter() error
	GetServerTime(ctx context.Context, ai asset.Item) (time.Time, error)
	GetWebsocket() (*websocket.Manager, error)
	SubscribeToWebsocketChannels(channels subscription.List) error
	UnsubscribeToWebsocketChannels(channels subscription.List) error
	GetSubscriptions() (subscription.List, error)
	GetSubscriptionTemplate(*subscription.Subscription) (*template.Template, error)
	FlushWebsocketChannels() error
	AuthenticateWebsocket(ctx context.Context) error
	CanUseAuthenticatedWebsocketEndpoints() bool
	GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (order.MinMaxLevel, error)
	CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error
	UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error
	GetCredentials(ctx context.Context) (*account.Credentials, error)
	EnsureOnePairEnabled() error
	PrintEnabledPairs()
	IsVerbose() bool
	GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error)

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
	MarginManagement

	// MatchSymbolWithAvailablePairs returns a currency pair based on the supplied
	// symbol and asset type. If the string is expected to have a delimiter this
	// will attempt to screen it out.
	MatchSymbolWithAvailablePairs(symbol string, a asset.Item, hasDelimiter bool) (currency.Pair, error)
	// MatchSymbolCheckEnabled returns a currency pair based on the supplied symbol
	// and asset type against the available pairs list. If the string is expected to
	// have a delimiter this will attempt to screen it out. It will also check if
	// the pair is enabled.
	MatchSymbolCheckEnabled(symbol string, a asset.Item, hasDelimiter bool) (pair currency.Pair, enabled bool, err error)
	// IsPairEnabled checks if a pair is enabled for an enabled asset type
	IsPairEnabled(pair currency.Pair, a asset.Item) (bool, error)
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
	WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error)
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
	GetCachedAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error)
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
	GetOpenInterest(context.Context, ...key.PairAsset) ([]futures.OpenInterest, error)
	ScaleCollateral(ctx context.Context, calculator *futures.CollateralCalculator) (*collateral.ByCurrency, error)
	GetPositionSummary(context.Context, *futures.PositionSummaryRequest) (*futures.PositionSummary, error)
	CalculateTotalCollateral(context.Context, *futures.TotalCollateralCalculator) (*futures.TotalCollateralResponse, error)
	GetFuturesPositions(context.Context, *futures.PositionsRequest) ([]futures.PositionDetails, error)
	GetHistoricalFundingRates(context.Context, *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error)
	GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error)
	IsPerpetualFutureCurrency(asset.Item, currency.Pair) (bool, error)
	GetCollateralCurrencyForContract(asset.Item, currency.Pair) (currency.Code, asset.Item, error)

	GetFuturesPositionSummary(context.Context, *futures.PositionSummaryRequest) (*futures.PositionSummary, error)
	GetFuturesPositionOrders(context.Context, *futures.PositionsRequest) ([]futures.PositionResponse, error)
	SetCollateralMode(ctx context.Context, item asset.Item, mode collateral.Mode) error
	GetCollateralMode(ctx context.Context, item asset.Item) (collateral.Mode, error)
	SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, orderSide order.Side) error
	GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, orderSide order.Side) (float64, error)
}

// MarginManagement manages margin positions and rates
type MarginManagement interface {
	SetMarginType(ctx context.Context, item asset.Item, pair currency.Pair, tp margin.Type) error
	ChangePositionMargin(ctx context.Context, change *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error)
	GetMarginRatesHistory(context.Context, *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error)
	futures.PNLCalculation
	GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error)
}
