package okgroup

import (
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the OKEX go routine
func (o *OKGroup) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKGroup) Run() {
	if o.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.WebsocketURL)
		log.Debugf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	prods, err := o.GetSpotInstruments()
	if err != nil {
		log.Errorf("OKEX failed to obtain available spot instruments. Err: %d", err)
		return
	}

	var pairs []string
	for x := range prods {
		pairs = append(pairs, prods[x].BaseCurrency+"_"+prods[x].QuoteCurrency)
	}

	err = o.UpdateCurrencies(pairs, false, false)
	if err != nil {
		log.Errorf("OKEX failed to update available currencies. Err: %s", err)
		return
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKGroup) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	return ticker.Price{}, common.ErrNotYetImplemented
}

// GetTickerPrice returns the ticker for a currency pair
func (o *OKGroup) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	return ticker.Price{}, common.ErrNotYetImplemented
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *OKGroup) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrNotYetImplemented
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKGroup) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrNotYetImplemented
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// OKEX exchange
func (o *OKGroup) GetAccountInfo() (exchange.AccountInfo, error) {
	return exchange.AccountInfo{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *OKGroup) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *OKGroup) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (o *OKGroup) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	return exchange.SubmitOrderResponse{}, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *OKGroup) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (o *OKGroup) CancelOrder(order exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders for all enabled currencies
func (o *OKGroup) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (o *OKGroup) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	return exchange.OrderDetail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *OKGroup) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *OKGroup) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (o *OKGroup) GetWebsocket() (*exchange.Websocket, error) {
	return o.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return o.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (o *OKGroup) GetWithdrawCapabilities() uint32 {
	return o.GetWithdrawPermissions()
}
