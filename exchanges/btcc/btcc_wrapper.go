package btcc

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
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
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if !common.StringDataContains(b.EnabledPairs, "_") || !common.StringDataContains(b.AvailablePairs, "_") {
		var availP []string
		for _, p := range b.GetAvailableCurrencies() {
			availP = append(availP, p.Display("_", true).String())
		}

		var enabledP []string
		for _, p := range b.GetEnabledCurrencies() {
			enabledP = append(enabledP, p.Display("_", true).String())
		}

		err := b.UpdateCurrencies(availP, false, true)
		if err != nil {
			log.Printf("%s failed to update available currencies. %s\n", b.Name, err)
		}

		err = b.UpdateCurrencies(enabledP, true, true)
		if err != nil {
			log.Printf("%s failed to update enabled currencies. %s\n", b.Name, err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCC) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	return ticker.Price{}, errors.New("REST NOT SUPPORTED")
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCC) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	return ticker.Price{}, errors.New("REST NOT SUPPORTED")
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *BTCC) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, errors.New("REST NOT SUPPORTED")
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCC) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	return orderbook.Base{}, errors.New("REST NOT SUPPORTED")
}

// GetAccountInfo retrieves balances for all enabled currencies for
// the BTCC exchange - TODO
func (b *BTCC) GetAccountInfo() (exchange.AccountInfo, error) {
	// var response exchange.AccountInfo
	// response.ExchangeName = b.GetName()
	// return response, nil
	return exchange.AccountInfo{}, errors.New("REST NOT SUPPORTED")
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCC) GetFundingHistory() ([]exchange.FundHistory, error) {
	// var fundHistory []exchange.FundHistory
	// return fundHistory, common.ErrFunctionNotSupported
	return nil, errors.New("REST NOT SUPPORTED")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCC) GetExchangeHistory(p pair.CurrencyPair, assetType string, timestampStart time.Time, tradeID int64) ([]exchange.TradeHistory, error) {
	return nil, errors.New("REST NOT SUPPORTED")
}

// SubmitOrder submits a new order
func (b *BTCC) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCC) ModifyOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCC) CancelOrder(order exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCC) CancelAllOrders() error {
	return common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (b *BTCC) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCC) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTCC) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTCC) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCC) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *BTCC) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}
