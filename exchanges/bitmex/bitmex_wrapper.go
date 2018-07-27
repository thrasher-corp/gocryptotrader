package bitmex

import (
	"errors"
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Bitmex go routine
func (b *Bitmex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitmex wrapper
func (b *Bitmex) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket), b.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitmex) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	// tick, err := b.GetTrade()
	// if err != nil {
	// 	return tickerPrice, err
	// }
	//
	// for _, x := range b.GetEnabledCurrencies() {
	// 	curr := exchange.FormatExchangeCurrency(b.Name, x)
	// 	for y := range tick {
	// 		if tick[y].Symbol == curr.String() {
	// 			tickerPrice.Pair = x
	// 			tickerPrice.Ask = tick[y].AskPrice
	// 			tickerPrice.Bid = tick[y].BidPrice
	// 			tickerPrice.High = tick[y].HighPrice
	// 			tickerPrice.Last = tick[y].LastPrice
	// 			tickerPrice.Low = tick[y].LowPrice
	// 			tickerPrice.Volume = tick[y].Volume
	// 			ticker.ProcessTicker(b.Name, x, tickerPrice, assetType)
	// 		}
	// 	}
	// }
	// return ticker.GetTicker(b.Name, p, assetType)
	return tickerPrice, errors.New("Not yet implemented")
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bitmex) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *Bitmex) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), currency, assetType)
	if err != nil {
		return b.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitmex) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base

	orderbookNew, err := b.GetOrderbook(OrderBookGetL2Params{
		Symbol: exchange.FormatExchangeCurrency(b.Name, p).String(),
		Depth:  0})
	if err != nil {
		return orderBook, err
	}

	for _, ob := range orderbookNew {
		if ob.Side == "Sell" {
			orderBook.Asks = append(orderBook.Asks,
				orderbook.Item{Amount: float64(ob.Size), Price: ob.Price})
			continue
		}
		if ob.Side == "Buy" {
			orderBook.Bids = append(orderBook.Bids,
				orderbook.Item{Amount: float64(ob.Size), Price: ob.Price})
			continue
		}
	}
	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)

	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Bitmex exchange
func (b *Bitmex) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	return response, errors.New("not implemented")
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (b *Bitmex) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitmex) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *Bitmex) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitmex) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *Bitmex) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *Bitmex) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *Bitmex) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *Bitmex) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawExchangeFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawExchangeFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
