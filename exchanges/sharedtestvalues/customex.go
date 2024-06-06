package sharedtestvalues

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// CustomEx creates a mock custom exchange
type CustomEx struct {
	exchange.Base
}

// Setup is a mock method for CustomEx
func (c *CustomEx) Setup(_ *config.Exchange) error {
	return nil
}

// SetDefaults is a mock method for CustomEx
func (c *CustomEx) SetDefaults() {
}

// GetName is a mock method for CustomEx
func (c *CustomEx) GetName() string {
	return "customex"
}

// IsEnabled is a mock method for CustomEx
func (c *CustomEx) IsEnabled() bool {
	return true
}

// SetEnabled is a mock method for CustomEx
func (c *CustomEx) SetEnabled(bool) {
}

// ValidateAPICredentials is a mock method for CustomEx
func (c *CustomEx) ValidateAPICredentials(_ context.Context, _ asset.Item) error {
	return nil
}

// FetchTicker is a mock method for CustomEx
func (c *CustomEx) FetchTicker(_ context.Context, _ currency.Pair, _ asset.Item) (*ticker.Price, error) {
	return nil, nil
}

// UpdateTickers is a mock method for CustomEx
func (c *CustomEx) UpdateTickers(_ context.Context, _ asset.Item) error {
	return nil
}

// UpdateTicker is a mock method for CustomEx
func (c *CustomEx) UpdateTicker(_ context.Context, _ currency.Pair, _ asset.Item) (*ticker.Price, error) {
	return nil, nil
}

// FetchOrderbook is a mock method for CustomEx
func (c *CustomEx) FetchOrderbook(_ context.Context, _ currency.Pair, _ asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

// UpdateOrderbook is a mock method for CustomEx
func (c *CustomEx) UpdateOrderbook(_ context.Context, _ currency.Pair, _ asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

// FetchTradablePairs is a mock method for CustomEx
func (c *CustomEx) FetchTradablePairs(_ context.Context, _ asset.Item) (currency.Pairs, error) {
	return nil, nil
}

// UpdateTradablePairs is a mock method for CustomEx
func (c *CustomEx) UpdateTradablePairs(_ context.Context, _ bool) error {
	return nil
}

// GetEnabledPairs is a mock method for CustomEx
func (c *CustomEx) GetEnabledPairs(_ asset.Item) (currency.Pairs, error) {
	return nil, nil
}

// GetAvailablePairs is a mock method for CustomEx
func (c *CustomEx) GetAvailablePairs(_ asset.Item) (currency.Pairs, error) {
	return nil, nil
}

// FetchAccountInfo is a mock method for CustomEx
func (c *CustomEx) FetchAccountInfo(_ context.Context, _ asset.Item) (account.Holdings, error) {
	return account.Holdings{}, nil
}

// UpdateAccountInfo is a mock method for CustomEx
func (c *CustomEx) UpdateAccountInfo(_ context.Context, _ asset.Item) (account.Holdings, error) {
	return account.Holdings{}, nil
}

// SetPairs is a mock method for CustomEx
func (c *CustomEx) SetPairs(_ currency.Pairs, _ asset.Item, _ bool) error {
	return nil
}

// GetAssetTypes is a mock method for CustomEx
func (c *CustomEx) GetAssetTypes(_ bool) asset.Items {
	return nil
}

// GetRecentTrades is a mock method for CustomEx
func (c *CustomEx) GetRecentTrades(_ context.Context, _ currency.Pair, _ asset.Item) ([]trade.Data, error) {
	return nil, nil
}

// GetHistoricTrades is a mock method for CustomEx
func (c *CustomEx) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, nil
}

// SupportsAutoPairUpdates is a mock method for CustomEx
func (c *CustomEx) SupportsAutoPairUpdates() bool {
	return false
}

// SupportsRESTTickerBatchUpdates is a mock method for CustomEx
func (c *CustomEx) SupportsRESTTickerBatchUpdates() bool {
	return false
}

// GetServerTime is a mock method for CustomEx
func (c *CustomEx) GetServerTime(context.Context, asset.Item) (time.Time, error) {
	return time.Now(), nil
}

// GetFeeByType is a mock method for CustomEx
func (c *CustomEx) GetFeeByType(_ context.Context, _ *exchange.FeeBuilder) (float64, error) {
	return 0.0, nil
}

// GetLastPairsUpdateTime is a mock method for CustomEx
func (c *CustomEx) GetLastPairsUpdateTime() int64 {
	return 0
}

// GetWithdrawPermissions is a mock method for CustomEx
func (c *CustomEx) GetWithdrawPermissions() uint32 {
	return 0
}

// FormatWithdrawPermissions is a mock method for CustomEx
func (c *CustomEx) FormatWithdrawPermissions() string {
	return ""
}

// SupportsWithdrawPermissions is a mock method for CustomEx
func (c *CustomEx) SupportsWithdrawPermissions(_ uint32) bool {
	return false
}

// GetAccountFundingHistory is a mock method for CustomEx
func (c *CustomEx) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, nil
}

// SubmitOrder is a mock method for CustomEx
func (c *CustomEx) SubmitOrder(_ context.Context, _ *order.Submit) (*order.SubmitResponse, error) {
	return nil, nil
}

// ModifyOrder is a mock method for CustomEx
func (c *CustomEx) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, nil
}

// CancelOrder is a mock method for CustomEx
func (c *CustomEx) CancelOrder(_ context.Context, _ *order.Cancel) error {
	return nil
}

// CancelBatchOrders is a mock method for CustomEx
func (c *CustomEx) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, nil
}

// CancelAllOrders is a mock method for CustomEx
func (c *CustomEx) CancelAllOrders(_ context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, nil
}

// GetOrderInfo is a mock method for CustomEx
func (c *CustomEx) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, nil
}

// GetDepositAddress is a mock method for CustomEx
func (c *CustomEx) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, nil
}

// GetOrderHistory is a mock method for CustomEx
func (c *CustomEx) GetOrderHistory(_ context.Context, _ *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return nil, nil
}

// GetWithdrawalsHistory is a mock method for CustomEx
func (c *CustomEx) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return []exchange.WithdrawalHistory{}, nil
}

// GetActiveOrders is a mock method for CustomEx
func (c *CustomEx) GetActiveOrders(_ context.Context, _ *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return []order.Detail{}, nil
}

// WithdrawCryptocurrencyFunds is a mock method for CustomEx
func (c *CustomEx) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

// WithdrawFiatFunds is a mock method for CustomEx
func (c *CustomEx) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

// WithdrawFiatFundsToInternationalBank is a mock method for CustomEx
func (c *CustomEx) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

// SetHTTPClientUserAgent is a mock method for CustomEx
func (c *CustomEx) SetHTTPClientUserAgent(_ string) error {
	return nil
}

// GetHTTPClientUserAgent is a mock method for CustomEx
func (c *CustomEx) GetHTTPClientUserAgent() (string, error) {
	return "", nil
}

// SetClientProxyAddress is a mock method for CustomEx
func (c *CustomEx) SetClientProxyAddress(_ string) error {
	return nil
}

// SupportsREST is a mock method for CustomEx
func (c *CustomEx) SupportsREST() bool {
	return true
}

// GetSubscriptions is a mock method for CustomEx
func (c *CustomEx) GetSubscriptions() (subscription.List, error) {
	return nil, nil
}

// GetBase is a mock method for CustomEx
func (c *CustomEx) GetBase() *exchange.Base {
	return nil
}

// SupportsAsset is a mock method for CustomEx
func (c *CustomEx) SupportsAsset(_ asset.Item) bool {
	return false
}

// GetHistoricCandles is a mock method for CustomEx
func (c *CustomEx) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return &kline.Item{}, nil
}

// GetHistoricCandlesExtended is a mock method for CustomEx
func (c *CustomEx) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return &kline.Item{}, nil
}

// DisableRateLimiter is a mock method for CustomEx
func (c *CustomEx) DisableRateLimiter() error {
	return nil
}

// EnableRateLimiter is a mock method for CustomEx
func (c *CustomEx) EnableRateLimiter() error {
	return nil
}

// GetWebsocket is a mock method for CustomEx
func (c *CustomEx) GetWebsocket() (*stream.Websocket, error) {
	return nil, nil
}

// IsWebsocketEnabled is a mock method for CustomEx
func (c *CustomEx) IsWebsocketEnabled() bool {
	return false
}

// SupportsWebsocket is a mock method for CustomEx
func (c *CustomEx) SupportsWebsocket() bool {
	return false
}

// SubscribeToWebsocketChannels is a mock method for CustomEx
func (c *CustomEx) SubscribeToWebsocketChannels(_ subscription.List) error {
	return nil
}

// UnsubscribeToWebsocketChannels is a mock method for CustomEx
func (c *CustomEx) UnsubscribeToWebsocketChannels(_ subscription.List) error {
	return nil
}

// IsAssetWebsocketSupported is a mock method for CustomEx
func (c *CustomEx) IsAssetWebsocketSupported(_ asset.Item) bool {
	return false
}

// FlushWebsocketChannels is a mock method for CustomEx
func (c *CustomEx) FlushWebsocketChannels() error {
	return nil
}

// AuthenticateWebsocket is a mock method for CustomEx
func (c *CustomEx) AuthenticateWebsocket(_ context.Context) error {
	return nil
}

// GetOrderExecutionLimits is a mock method for CustomEx
func (c *CustomEx) GetOrderExecutionLimits(_ asset.Item, _ currency.Pair) (order.MinMaxLevel, error) {
	return order.MinMaxLevel{}, nil
}

// CheckOrderExecutionLimits is a mock method for CustomEx
func (c *CustomEx) CheckOrderExecutionLimits(_ asset.Item, _ currency.Pair, _, _ float64, _ order.Type) error {
	return nil
}

// UpdateOrderExecutionLimits is a mock method for CustomEx
func (c *CustomEx) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return nil
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (c *CustomEx) GetHistoricalFundingRates(_ context.Context, _ *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	return nil, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (c *CustomEx) GetLatestFundingRates(_ context.Context, _ *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, nil
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (c *CustomEx) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}
