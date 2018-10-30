package btcc

import (
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (b *BTCC) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (b *BTCC) SetDefaults() {
	b.Name = "BTCC"
	b.Enabled = true
	b.Verbose = true
	b.APIWithdrawPermissions = exchange.NoAPIWithdrawalMethods
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      false,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: false,
				TickerBatching:  false,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: false,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, btccAuthRate),
		request.NewRateLimit(time.Second, btccUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.WebsocketInit()
}

// Setup is run on startup to setup exchange with config values
func (b *BTCC) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return b.WebsocketSetup(b.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		btccSocketioAddress,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the BTCC go routine
func (b *BTCC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTCC wrapper
func (b *BTCC) Run() {
	if b.Verbose {
		log.Debugf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()))
		b.PrintEnabledPairs()
	}

	if common.StringDataContains(b.GetEnabledPairs(assets.AssetTypeSpot).Strings(), "CNY") ||
		common.StringDataContains(b.GetAvailablePairs(assets.AssetTypeSpot).Strings(), "CNY") ||
		common.StringDataContains(b.BaseCurrencies.Strings(), "CNY") {
		log.Warn("WARNING: BTCC only supports BTCUSD now, upgrading available, enabled and base currencies to BTCUSD/USD")
		pairs := currency.NewPairsFromStrings([]string{"BTCUSD"})
		cfg := config.GetConfig()
		exchCfg, err := cfg.GetExchangeConfig(b.Name)
		if err != nil {
			log.Errorf("%s failed to get exchange config. %s\n", b.Name, err)
			return
		}

		exchCfg.BaseCurrencies = currency.Currencies{currency.USD}
		exchCfg.CurrencyPairs.StorePairs(assets.AssetTypeSpot, pairs, true)
		exchCfg.CurrencyPairs.StorePairs(assets.AssetTypeSpot, pairs, false)
		b.BaseCurrencies = currency.Currencies{currency.USD}

		err = b.UpdatePairs(pairs, assets.AssetTypeSpot, false, true)
		if err != nil {
			log.Errorf("%s failed to update available currencies. %s\n", b.Name, err)
		}

		err = b.UpdatePairs(pairs, assets.AssetTypeSpot, true, true)
		if err != nil {
			log.Errorf("%s failed to update enabled currencies. %s\n", b.Name, err)
		}

		err = cfg.UpdateExchangeConfig(exchCfg)
		if err != nil {
			log.Errorf("%s failed to update config. %s\n", b.Name, err)
			return
		}
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *BTCC) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *BTCC) UpdateTradablePairs(forceUpdate bool) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCC) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	return ticker.Price{}, common.ErrFunctionNotSupported
}

// FetchTicker returns the ticker for a currency pair
func (b *BTCC) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	return ticker.Price{}, common.ErrFunctionNotSupported
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *BTCC) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrFunctionNotSupported
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCC) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrFunctionNotSupported
}

// GetAccountInfo : Retrieves balances for all enabled currencies for
// the Kraken exchange - TODO
func (b *BTCC) GetAccountInfo() (exchange.AccountInfo, error) {
	return exchange.AccountInfo{}, common.ErrFunctionNotSupported
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCC) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCC) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *BTCC) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	return exchange.SubmitOrderResponse{}, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCC) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCC) CancelOrder(order *exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCC) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (b *BTCC) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	return exchange.OrderDetail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCC) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTCC) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTCC) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCC) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTCC) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCC) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
