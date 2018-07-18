package bittrex

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

// Start starts the Bittrex go routine
func (b *Bittrex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bittrex wrapper
func (b *Bittrex) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	exchangeProducts, err := b.GetMarkets()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(b.EnabledPairs, "-") || !common.StringDataContains(b.AvailablePairs, "-") {
			forceUpgrade = true
		}
		var currencies []string
		for x := range exchangeProducts.Result {
			if !exchangeProducts.Result[x].IsActive || exchangeProducts.Result[x].MarketName == "" {
				continue
			}
			currencies = append(currencies, exchangeProducts.Result[x].MarketName)
		}

		if forceUpgrade {
			enabledPairs := []string{"USDT-BTC"}
			log.Println("WARNING: Available pairs for Bittrex reset due to config upgrade, please enable the ones you would like again")

			err = b.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Printf("%s Failed to get config.\n", b.GetName())
			}
		}
		err = b.UpdateCurrencies(currencies, false, forceUpgrade)
		if err != nil {
			log.Printf("%s Failed to get config.\n", b.GetName())
		}
	}
}

// GetExchangeAccountInfo Retrieves balances for all enabled currencies for the
// Bittrex exchange
func (b *Bittrex) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetAccountBalances()
	if err != nil {
		return response, err
	}

	for i := 0; i < len(accountBalance.Result); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance.Result[i].Currency
		exchangeCurrency.TotalValue = accountBalance.Result[i].Balance
		exchangeCurrency.Hold = accountBalance.Result[i].Balance - accountBalance.Result[i].Available
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bittrex) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetMarketSummaries()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledCurrencies() {
		curr := exchange.FormatExchangeCurrency(b.Name, x)
		for y := range tick.Result {
			if tick.Result[y].MarketName == curr.String() {
				tickerPrice.Pair = x
				tickerPrice.High = tick.Result[y].High
				tickerPrice.Low = tick.Result[y].Low
				tickerPrice.Ask = tick.Result[y].Ask
				tickerPrice.Bid = tick.Result[y].Bid
				tickerPrice.Last = tick.Result[y].Last
				tickerPrice.Volume = tick.Result[y].Volume
				ticker.ProcessTicker(b.GetName(), x, tickerPrice, assetType)
			}
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bittrex) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, ticker.Spot)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *Bittrex) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bittrex) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(exchange.FormatExchangeCurrency(b.GetName(), p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Result.Buy {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Amount: orderbookNew.Result.Buy[x].Quantity,
				Price:  orderbookNew.Result.Buy[x].Rate,
			},
		)
	}

	for x := range orderbookNew.Result.Sell {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Amount: orderbookNew.Result.Sell[x].Quantity,
				Price:  orderbookNew.Result.Sell[x].Rate,
			},
		)
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (b *Bittrex) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bittrex) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *Bittrex) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bittrex) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *Bittrex) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *Bittrex) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *Bittrex) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *Bittrex) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bittrex) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
