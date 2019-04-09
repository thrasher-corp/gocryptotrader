package btcc

import (
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

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
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if common.StringDataContains(b.EnabledPairs.Strings(), "CNY") ||
		common.StringDataContains(b.AvailablePairs.Strings(), "CNY") ||
		common.StringDataContains(b.BaseCurrencies.Strings(), "CNY") {
		log.Warn("BTCC only supports BTCUSD now, upgrading available, enabled and base currencies to BTCUSD/USD")
		pairs := currency.Pairs{currency.Pair{Base: currency.BTC,
			Quote: currency.USD}}
		cfg := config.GetConfig()
		exchCfg, err := cfg.GetExchangeConfig(b.Name)
		if err != nil {
			log.Errorf("%s failed to get exchange config. %s\n", b.Name, err)
			return
		}

		exchCfg.BaseCurrencies = currency.Currencies{currency.USD}
		exchCfg.AvailablePairs = pairs
		exchCfg.EnabledPairs = pairs
		b.BaseCurrencies = currency.Currencies{currency.USD}

		err = b.UpdateCurrencies(pairs, false, true)
		if err != nil {
			log.Errorf("%s failed to update available currencies. %s\n", b.Name, err)
		}

		err = b.UpdateCurrencies(pairs, true, true)
		if err != nil {
			log.Errorf("%s failed to update enabled currencies. %s\n", b.Name, err)
		}

		err = cfg.UpdateExchangeConfig(&exchCfg)
		if err != nil {
			log.Errorf("%s failed to update config. %s\n", b.Name, err)
			return
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCC) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	return ticker.Price{}, common.ErrFunctionNotSupported
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCC) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	return ticker.Price{}, common.ErrFunctionNotSupported
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *BTCC) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrFunctionNotSupported
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCC) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
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
func (b *BTCC) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
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
	if (b.APIKey == "" || b.APISecret == "") && // Todo check connection status
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
