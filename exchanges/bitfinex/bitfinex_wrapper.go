package bitfinex

import (
	"errors"
	"log"
	"net/url"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Bitfinex go routine
func (b *Bitfinex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitfinex wrapper
func (b *Bitfinex) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.WebsocketClient()
	}

	exchangeProducts, err := b.GetSymbols()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		err = b.UpdateCurrencies(exchangeProducts, false, false)
		if err != nil {
			log.Printf("%s Failed to update available symbols.\n", b.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitfinex) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	enabledPairs := b.GetEnabledCurrencies()

	var pairs []string
	for x := range enabledPairs {
		pairs = append(pairs, "t"+enabledPairs[x].Pair().String())
	}

	tickerNew, err := b.GetTickersV2(common.JoinStrings(pairs, ","))
	if err != nil {
		return tickerPrice, err
	}

	for x := range tickerNew {
		newP := pair.NewCurrencyPair(tickerNew[x].Symbol[1:4], tickerNew[x].Symbol[4:])
		var tick ticker.Price
		tick.Pair = newP
		tick.Ask = tickerNew[x].Ask
		tick.Bid = tickerNew[x].Bid
		tick.Low = tickerNew[x].Low
		tick.Last = tickerNew[x].Last
		tick.Volume = tickerNew[x].Volume
		tick.High = tickerNew[x].High
		ticker.ProcessTicker(b.Name, tick.Pair, tick, assetType)
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bitfinex) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, ticker.Spot)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *Bitfinex) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitfinex) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	urlVals := url.Values{}
	urlVals.Set("limit_bids", "100")
	urlVals.Set("limit_asks", "100")
	orderbookNew, err := b.GetOrderbook(p.Pair().String(), urlVals)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Price: orderbookNew.Asks[x].Price, Amount: orderbookNew.Asks[x].Amount})
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Price: orderbookNew.Bids[x].Price, Amount: orderbookNew.Bids[x].Amount})
	}

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies on the
// Bitfinex exchange
func (b *Bitfinex) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = b.GetName()
	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}
	if !b.Enabled {
		return response, nil
	}

	type bfxCoins struct {
		OnHold    float64
		Available float64
	}

	accounts := make(map[string]bfxCoins)

	for i := range accountBalance {
		onHold := accountBalance[i].Amount - accountBalance[i].Available
		coins := bfxCoins{
			OnHold:    onHold,
			Available: accountBalance[i].Available,
		}
		result, ok := accounts[accountBalance[i].Currency]
		if !ok {
			accounts[accountBalance[i].Currency] = coins
		} else {
			result.Available += accountBalance[i].Available
			result.OnHold += onHold
			accounts[accountBalance[i].Currency] = result
		}
	}

	for x, y := range accounts {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = common.StringToUpper(x)
		exchangeCurrency.TotalValue = y.Available + y.OnHold
		exchangeCurrency.Hold = y.OnHold
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}

	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (b *Bitfinex) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitfinex) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (b *Bitfinex) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitfinex) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (b *Bitfinex) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (b *Bitfinex) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (b *Bitfinex) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (b *Bitfinex) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is submitted
func (b *Bitfinex) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitfinex) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitfinex) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}
