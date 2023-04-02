package sharedtestvalues

import (
	"context"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// CustomEx creates a mock custom exchange
type CustomEx struct {
	exchange.Base
}

// Setup is a mock method for CustomEx
func (c *CustomEx) Setup(exch *config.Exchange) error {
	return nil
}

// Start is a mock method for CustomEx
func (c *CustomEx) Start(ctx context.Context, wg *sync.WaitGroup) error {
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

// ValidateCredentials is a mock method for CustomEx
func (c *CustomEx) ValidateCredentials(ctx context.Context, a asset.Item) error {
	return nil
}

// FetchTicker is a mock method for CustomEx
func (c *CustomEx) FetchTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return nil, nil
}

// UpdateTickers is a mock method for CustomEx
func (c *CustomEx) UpdateTickers(ctx context.Context, a asset.Item) error {
	return nil
}

// UpdateTicker is a mock method for CustomEx
func (c *CustomEx) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return nil, nil
}

// FetchOrderbook is a mock method for CustomEx
func (c *CustomEx) FetchOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

// UpdateOrderbook is a mock method for CustomEx
func (c *CustomEx) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

// FetchTradablePairs is a mock method for CustomEx
func (c *CustomEx) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	return nil, nil
}

// UpdateTradablePairs is a mock method for CustomEx
func (c *CustomEx) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	return nil
}

// GetEnabledPairs is a mock method for CustomEx
func (c *CustomEx) GetEnabledPairs(a asset.Item) (currency.Pairs, error) {
	return nil, nil
}

// GetAvailablePairs is a mock method for CustomEx
func (c *CustomEx) GetAvailablePairs(a asset.Item) (currency.Pairs, error) {
	return nil, nil
}

// FetchAccountInfo is a mock method for CustomEx
func (c *CustomEx) FetchAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	return account.Holdings{}, nil
}

// UpdateAccountInfo is a mock method for CustomEx
func (c *CustomEx) UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	return account.Holdings{}, nil
}

// SetPairs is a mock method for CustomEx
func (c *CustomEx) SetPairs(pairs currency.Pairs, a asset.Item, enabled bool) error {
	return nil
}

// GetAssetTypes is a mock method for CustomEx
func (c *CustomEx) GetAssetTypes(enabled bool) asset.Items {
	return nil
}

// GetRecentTrades is a mock method for CustomEx
func (c *CustomEx) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	return nil, nil
}

// GetHistoricTrades is a mock method for CustomEx
func (c *CustomEx) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
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

// GetFeeByType is a mock method for CustomEx
func (c *CustomEx) GetFeeByType(ctx context.Context, f *exchange.FeeBuilder) (float64, error) {
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
func (c *CustomEx) SupportsWithdrawPermissions(permissions uint32) bool {
	return false
}

// GetFundingHistory is a mock method for CustomEx
func (c *CustomEx) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, nil
}

// SubmitOrder is a mock method for CustomEx
func (c *CustomEx) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	return nil, nil
}

// ModifyOrder is a mock method for CustomEx
func (c *CustomEx) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, nil
}

// CancelOrder is a mock method for CustomEx
func (c *CustomEx) CancelOrder(ctx context.Context, o *order.Cancel) error {
	return nil
}

// CancelBatchOrders is a mock method for CustomEx
func (c *CustomEx) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, nil
}

// CancelAllOrders is a mock method for CustomEx
func (c *CustomEx) CancelAllOrders(ctx context.Context, orders *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, nil
}

// GetOrderInfo is a mock method for CustomEx
func (c *CustomEx) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, nil
}

// GetDepositAddress is a mock method for CustomEx
func (c *CustomEx) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, accountID, chain string) (*deposit.Address, error) {
	return nil, nil
}

// GetOrderHistory is a mock method for CustomEx
func (c *CustomEx) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	return nil, nil
}

// GetWithdrawalsHistory is a mock method for CustomEx
func (c *CustomEx) GetWithdrawalsHistory(ctx context.Context, code currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	return []exchange.WithdrawalHistory{}, nil
}

// GetActiveOrders is a mock method for CustomEx
func (c *CustomEx) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	return []order.Detail{}, nil
}

// WithdrawCryptocurrencyFunds is a mock method for CustomEx
func (c *CustomEx) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

// WithdrawFiatFunds is a mock method for CustomEx
func (c *CustomEx) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

// WithdrawFiatFundsToInternationalBank is a mock method for CustomEx
func (c *CustomEx) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

// SetHTTPClientUserAgent is a mock method for CustomEx
func (c *CustomEx) SetHTTPClientUserAgent(ua string) error {
	return nil
}

// GetHTTPClientUserAgent is a mock method for CustomEx
func (c *CustomEx) GetHTTPClientUserAgent() (string, error) {
	return "", nil
}

// SetClientProxyAddress is a mock method for CustomEx
func (c *CustomEx) SetClientProxyAddress(addr string) error {
	return nil
}

// SupportsREST is a mock method for CustomEx
func (c *CustomEx) SupportsREST() bool {
	return true
}

// GetSubscriptions is a mock method for CustomEx
func (c *CustomEx) GetSubscriptions() ([]stream.ChannelSubscription, error) {
	return nil, nil
}

// GetDefaultConfig is a mock method for CustomEx
func (c *CustomEx) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	return nil, nil
}

// GetBase is a mock method for CustomEx
func (c *CustomEx) GetBase() *exchange.Base {
	return nil
}

// SupportsAsset is a mock method for CustomEx
func (c *CustomEx) SupportsAsset(assetType asset.Item) bool {
	return false
}

// GetHistoricCandles is a mock method for CustomEx
func (c *CustomEx) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, timeStart, timeEnd time.Time) (*kline.Item, error) {
	return &kline.Item{}, nil
}

// GetHistoricCandlesExtended is a mock method for CustomEx
func (c *CustomEx) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, interval kline.Interval, timeStart, timeEnd time.Time) (*kline.Item, error) {
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
func (c *CustomEx) SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	return nil
}

// UnsubscribeToWebsocketChannels is a mock method for CustomEx
func (c *CustomEx) UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	return nil
}

// IsAssetWebsocketSupported is a mock method for CustomEx
func (c *CustomEx) IsAssetWebsocketSupported(aType asset.Item) bool {
	return false
}

// FlushWebsocketChannels is a mock method for CustomEx
func (c *CustomEx) FlushWebsocketChannels() error {
	return nil
}

// AuthenticateWebsocket is a mock method for CustomEx
func (c *CustomEx) AuthenticateWebsocket(ctx context.Context) error {
	return nil
}

// GetOrderExecutionLimits is a mock method for CustomEx
func (c *CustomEx) GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (order.MinMaxLevel, error) {
	return order.MinMaxLevel{}, nil
}

// CheckOrderExecutionLimits is a mock method for CustomEx
func (c *CustomEx) CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error {
	return nil
}

// UpdateOrderExecutionLimits is a mock method for CustomEx
func (c *CustomEx) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	return nil
}
