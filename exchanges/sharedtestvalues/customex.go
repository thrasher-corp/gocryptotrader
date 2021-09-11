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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

type CustomEx struct {
	exchange.Base
}

func (c *CustomEx) Setup(exch *config.ExchangeConfig) error {
	return nil
}

func (c *CustomEx) Start(wg *sync.WaitGroup) {
}

func (c *CustomEx) SetDefaults() {
}

func (c *CustomEx) GetName() string {
	return "customex"
}

func (c *CustomEx) IsEnabled() bool {
	return true
}

func (c *CustomEx) SetEnabled(bool) {
}

func (c *CustomEx) ValidateCredentials(ctx context.Context, a asset.Item) error {
	return nil
}

func (c *CustomEx) FetchTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return nil, nil
}

func (c *CustomEx) UpdateTickers(ctx context.Context, a asset.Item) error {
	return nil
}

func (c *CustomEx) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return nil, nil
}

func (c *CustomEx) FetchOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

func (c *CustomEx) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

func (c *CustomEx) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	return nil, nil
}

func (c *CustomEx) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	return nil
}

func (c *CustomEx) GetEnabledPairs(a asset.Item) (currency.Pairs, error) {
	return nil, nil
}

func (c *CustomEx) GetAvailablePairs(a asset.Item) (currency.Pairs, error) {
	return nil, nil
}

func (c *CustomEx) FetchAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	return account.Holdings{}, nil
}

func (c *CustomEx) UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	return account.Holdings{}, nil
}

func (c *CustomEx) GetAuthenticatedAPISupport(endpoint uint8) bool {
	return false
}

func (c *CustomEx) SetPairs(pairs currency.Pairs, a asset.Item, enabled bool) error {
	return nil
}

func (c *CustomEx) GetAssetTypes(enabled bool) asset.Items {
	return nil
}

func (c *CustomEx) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	return nil, nil
}

func (c *CustomEx) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	return nil, nil
}

func (c *CustomEx) SupportsAutoPairUpdates() bool {
	return false
}

func (c *CustomEx) SupportsRESTTickerBatchUpdates() bool {
	return false
}

func (c *CustomEx) GetFeeByType(ctx context.Context, f *exchange.FeeBuilder) (float64, error) {
	return 0.0, nil
}

func (c *CustomEx) GetLastPairsUpdateTime() int64 {
	return 0
}

func (c *CustomEx) GetWithdrawPermissions() uint32 {
	return 0
}

func (c *CustomEx) FormatWithdrawPermissions() string {
	return ""
}

func (c *CustomEx) SupportsWithdrawPermissions(permissions uint32) bool {
	return false
}

func (c *CustomEx) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, nil
}

func (c *CustomEx) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	return order.SubmitResponse{}, nil
}

func (c *CustomEx) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, nil
}

func (c *CustomEx) CancelOrder(ctx context.Context, o *order.Cancel) error {
	return nil
}

func (c *CustomEx) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, nil
}

func (c *CustomEx) CancelAllOrders(ctx context.Context, orders *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, nil
}

func (c *CustomEx) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, nil
}

func (c *CustomEx) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, accountID string) (string, error) {
	return "", nil
}

func (c *CustomEx) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, nil
}

func (c *CustomEx) GetWithdrawalsHistory(ctx context.Context, code currency.Code) ([]exchange.WithdrawalHistory, error) {
	return []exchange.WithdrawalHistory{}, nil
}

func (c *CustomEx) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return []order.Detail{}, nil
}

func (c *CustomEx) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (c *CustomEx) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (c *CustomEx) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (c *CustomEx) SetHTTPClientUserAgent(ua string) {
}

func (c *CustomEx) GetHTTPClientUserAgent() string {
	return ""
}

func (c *CustomEx) SetClientProxyAddress(addr string) error {
	return nil
}

func (c *CustomEx) SupportsREST() bool {
	return true
}

func (c *CustomEx) GetSubscriptions() ([]stream.ChannelSubscription, error) {
	return nil, nil
}

func (c *CustomEx) GetDefaultConfig() (*config.ExchangeConfig, error) {
	return nil, nil
}

func (c *CustomEx) GetBase() *exchange.Base {
	return nil
}

func (c *CustomEx) SupportsAsset(assetType asset.Item) bool {
	return false
}

func (c *CustomEx) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, nil
}

func (c *CustomEx) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, nil
}

func (c *CustomEx) DisableRateLimiter() error {
	return nil
}

func (c *CustomEx) EnableRateLimiter() error {
	return nil
}

func (c *CustomEx) GetWebsocket() (*stream.Websocket, error) {
	return nil, nil
}

func (c *CustomEx) IsWebsocketEnabled() bool {
	return false
}

func (c *CustomEx) SupportsWebsocket() bool {
	return false
}

func (c *CustomEx) SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	return nil
}

func (c *CustomEx) UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	return nil
}

func (c *CustomEx) IsAssetWebsocketSupported(aType asset.Item) bool {
	return false
}

func (c *CustomEx) FlushWebsocketChannels() error {
	return nil
}

func (c *CustomEx) AuthenticateWebsocket(ctx context.Context) error {
	return nil
}

func (c *CustomEx) GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (*order.Limits, error) {
	return nil, nil
}

func (c *CustomEx) CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error {
	return nil
}

func (c *CustomEx) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	return nil
}
