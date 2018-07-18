package btcmarkets

import (
	"errors"
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the BTC Markets go routine
func (b *BTCMarkets) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTC Markets wrapper
func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if !common.StringDataContains(b.EnabledPairs, "AUD") || !common.StringDataContains(b.EnabledPairs, "AUD") {
		enabledPairs := []string{}
		for x := range b.EnabledPairs {
			enabledPairs = append(enabledPairs, b.EnabledPairs[x]+"AUD")
		}

		availablePairs := []string{}
		for x := range b.AvailablePairs {
			availablePairs = append(availablePairs, b.AvailablePairs[x]+"AUD")
		}

		log.Println("BTCMarkets: Upgrading available and enabled pairs")

		err := b.UpdateCurrencies(enabledPairs, true, true)
		if err != nil {
			log.Printf("%s Failed to get config.\n", b.GetName())
			return
		}

		err = b.UpdateCurrencies(availablePairs, false, true)
		if err != nil {
			log.Printf("%s Failed to get config.\n", b.GetName())
			return
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCMarkets) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.GetFirstCurrency().String(),
		p.GetSecondCurrency().String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.BestAsk
	tickerPrice.Bid = tick.BestBID
	tickerPrice.Last = tick.LastPrice
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCMarkets) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *BTCMarkets) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCMarkets) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.GetFirstCurrency().String(),
		p.GetSecondCurrency().String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// BTCMarkets exchange
func (b *BTCMarkets) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()

	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}

	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].PendingFunds

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (b *BTCMarkets) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCMarkets) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *BTCMarkets) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return b.NewOrder(p.GetFirstCurrency().Upper().String(), p.GetSecondCurrency().Upper().String(), price, amount, side.Format(b.GetName()), orderType.Format(b.GetName()), clientID)
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCMarkets) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not supported on exchange")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *BTCMarkets) CancelExchangeOrder(orderID int64) error {
	_, err := b.CancelOrder([]int64{orderID})
	if err != nil {
		return err
	}
	return nil
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *BTCMarkets) CancelAllExchangeOrders() error {
	orders, err := b.GetOrders("", "", 0, 0, true)
	if err != nil {
		return err
	}

	var orderList []int64
	for _, order := range orders {
		orderList = append(orderList, order.ID)
	}

	_, err = b.CancelOrder(orderList)
	if err != nil {
		return err
	}
	return nil
}

// GetExchangeOrderInfo returns information on a current open order
func (b *BTCMarkets) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var OrderDetail exchange.OrderDetail

	orders, err := b.GetOrderDetail([]int64{orderID})
	if err != nil {
		return OrderDetail, err
	}

	if len(orders) > 1 {
		return OrderDetail, errors.New("too many orders returned")
	}

	if len(orders) == 0 {
		return OrderDetail, errors.New("no orders found")
	}

	for _, order := range orders {
		OrderDetail.Amount = order.Volume
		OrderDetail.BaseCurrency = order.Currency
		OrderDetail.CreationTime = int64(order.CreationTime)
		OrderDetail.Exchange = b.GetName()
		OrderDetail.ID = order.ID
		OrderDetail.OpenVolume = order.OpenVolume
		OrderDetail.OrderSide = order.OrderSide
		OrderDetail.OrderType = order.OrderType
		OrderDetail.Price = order.Price
		OrderDetail.QuoteCurrency = order.Instrument
		OrderDetail.Status = order.Status
	}

	return OrderDetail, nil
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *BTCMarkets) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not supported on exchange")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return b.WithdrawCrypto(amount, cryptocurrency.String(), address)
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	bd, err := b.GetClientBankAccounts(b.Name, currency.Upper().String())
	if err != nil {
		return "", err
	}
	return b.WithdrawAUD(bd.AccountName, bd.AccountNumber, bd.BankName, bd.BSBNumber, amount)
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
