package btcc

import (
	"errors"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
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

	if common.StringDataContains(b.EnabledPairs, "CNY") || common.StringDataContains(b.AvailablePairs, "CNY") || common.StringDataContains(b.BaseCurrencies, "CNY") {
		log.Warn("BTCC only supports BTCUSD now, upgrading available, enabled and base currencies to BTCUSD/USD")
		pairs := []string{"BTCUSD"}
		cfg := config.GetConfig()
		exchCfg, err := cfg.GetExchangeConfig(b.Name)
		if err != nil {
			log.Errorf("%s failed to get exchange config. %s\n", b.Name, err)
			return
		}

		exchCfg.BaseCurrencies = "USD"
		exchCfg.AvailablePairs = pairs[0]
		exchCfg.EnabledPairs = pairs[0]
		b.BaseCurrencies = []string{"USD"}

		err = b.UpdateCurrencies(pairs, false, true)
		if err != nil {
			log.Errorf("%s failed to update available currencies. %s\n", b.Name, err)
		}

		err = b.UpdateCurrencies(pairs, true, true)
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

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCC) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	// var tickerPrice ticker.Price
	// tick, err := b.GetTicker(exchange.FormatExchangeCurrency(b.GetName(), p).String())
	// if err != nil {
	// 	return tickerPrice, err
	// }
	// tickerPrice.Pair = p
	// tickerPrice.Ask = tick.AskPrice
	// tickerPrice.Bid = tick.BidPrice
	// tickerPrice.Low = tick.Low
	// tickerPrice.Last = tick.Last
	// tickerPrice.Volume = tick.Volume24H
	// tickerPrice.High = tick.High
	// ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	// return ticker.GetTicker(b.Name, p, assetType)
	return ticker.Price{}, errors.New("REST NOT SUPPORTED")
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCC) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	// tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	// if err != nil {
	// 	return b.UpdateTicker(p, assetType)
	// }
	// return tickerNew, nil
	return ticker.Price{}, errors.New("REST NOT SUPPORTED")
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *BTCC) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	// ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	// if err != nil {
	// 	return b.UpdateOrderbook(p, assetType)
	// }
	// return ob, nil
	return orderbook.Base{}, errors.New("REST NOT SUPPORTED")
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCC) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	// var orderBook orderbook.Base
	// orderbookNew, err := b.GetOrderBook(exchange.FormatExchangeCurrency(b.GetName(), p).String(), 100)
	// if err != nil {
	// 	return orderBook, err
	// }

	// for x := range orderbookNew.Bids {
	// 	data := orderbookNew.Bids[x]
	// 	orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: data[0], Amount: data[1]})
	// }

	// for x := range orderbookNew.Asks {
	// 	data := orderbookNew.Asks[x]
	// 	orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: data[0], Amount: data[1]})
	// }

	// orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	// return orderbook.GetOrderbook(b.Name, p, assetType)
	return orderbook.Base{}, errors.New("REST NOT SUPPORTED")
}

// GetAccountInfo : Retrieves balances for all enabled currencies for
// the Kraken exchange - TODO
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
func (b *BTCC) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	// var resp []exchange.TradeHistory

	// return resp, common.ErrNotYetImplemented
	return nil, errors.New("REST NOT SUPPORTED")
}

// SubmitOrder submits a new order
func (b *BTCC) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCC) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCC) CancelOrder(order exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCC) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (b *BTCC) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCC) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTCC) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCC) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
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

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCC) GetOrderHistory(orderHistoryRequest exchange.OrderHistoryRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
