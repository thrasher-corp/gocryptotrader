package sharedtestvalues

import (
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

func (c *CustomEx) ValidateCredentials(a asset.Item) error {
	return nil
}

func (c *CustomEx) FetchTicker(p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return nil, nil
}

func (c *CustomEx) UpdateTickers(a asset.Item) error {
	return nil
}

func (c *CustomEx) UpdateTicker(p currency.Pair, a asset.Item) (*ticker.Price, error) {
	return nil, nil
}

func (c *CustomEx) FetchOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

func (c *CustomEx) UpdateOrderbook(p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	return nil, nil
}

func (c *CustomEx) FetchTradablePairs(a asset.Item) ([]string, error) {
	return nil, nil
}

func (c *CustomEx) UpdateTradablePairs(forceUpdate bool) error {
	return nil
}

func (c *CustomEx) GetEnabledPairs(a asset.Item) (currency.Pairs, error) {
	return nil, nil
}

func (c *CustomEx) GetAvailablePairs(a asset.Item) (currency.Pairs, error) {
	return nil, nil
}

func (c *CustomEx) FetchAccountInfo(a asset.Item) (account.Holdings, error) {
	return account.Holdings{}, nil
}

func (c *CustomEx) UpdateAccountInfo(a asset.Item) (account.Holdings, error) {
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

func (c *CustomEx) GetRecentTrades(p currency.Pair, a asset.Item) ([]trade.Data, error) {
	return nil, nil
}

func (c *CustomEx) GetHistoricTrades(p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	return nil, nil
}

func (c *CustomEx) SupportsAutoPairUpdates() bool {
	return false
}

func (c *CustomEx) SupportsRESTTickerBatchUpdates() bool {
	return false
}

func (c *CustomEx) GetFeeByType(f *exchange.FeeBuilder) (float64, error) {
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

func (c *CustomEx) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, nil
}

func (c *CustomEx) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	return order.SubmitResponse{}, nil
}

func (c *CustomEx) ModifyOrder(action *order.Modify) (order.Modify, error) {
	return order.Modify{}, nil
}

func (c *CustomEx) CancelOrder(o *order.Cancel) error {
	return nil
}

func (c *CustomEx) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, nil
}

func (c *CustomEx) CancelAllOrders(orders *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, nil
}

func (c *CustomEx) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, nil
}

func (c *CustomEx) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", nil
}

func (c *CustomEx) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, nil
}

func (c *CustomEx) GetWithdrawalsHistory(code currency.Code) ([]exchange.WithdrawalHistory, error) {
	return []exchange.WithdrawalHistory{}, nil
}

func (c *CustomEx) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return []order.Detail{}, nil
}

func (c *CustomEx) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (c *CustomEx) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (c *CustomEx) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
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

func (c *CustomEx) GetHistoricCandles(p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, nil
}

func (c *CustomEx) GetHistoricCandlesExtended(p currency.Pair, a asset.Item, timeStart, timeEnd time.Time, interval kline.Interval) (kline.Item, error) {
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

func (c *CustomEx) AuthenticateWebsocket() error {
	return nil
}

func (c *CustomEx) GetOrderExecutionLimits(a asset.Item, cp currency.Pair) (*order.Limits, error) {
	return nil, nil
}

func (c *CustomEx) CheckOrderExecutionLimits(a asset.Item, cp currency.Pair, price, amount float64, orderType order.Type) error {
	return nil
}

func (c *CustomEx) UpdateOrderExecutionLimits(a asset.Item) error {
	return nil
}
