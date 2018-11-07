package okcoin

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the OKCoin go routine
func (o *OKCoin) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKCoin wrapper
func (o *OKCoin) Run() {
	if o.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	if o.APIUrl == okcoinAPIURL {
		// OKCoin International
		forceUpgrade := false
		if !common.StringDataContains(o.EnabledPairs, "_") || !common.StringDataContains(o.AvailablePairs, "_") {
			forceUpgrade = true
		}

		var currencies []string
		for x := range o.AvailablePairs {
			currencies = append(currencies, o.AvailablePairs[x][0:3]+"_"+o.AvailablePairs[x][3:])
		}

		if forceUpgrade {
			enabledPairs := []string{"btc_usd"}
			log.Println("WARNING: Available pairs for OKCoin International reset due to config upgrade, please enable the pairs you would like again.")

			err := o.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Printf("%s failed to update currencies. Err: %s", o.Name, err)
			}

			err = o.UpdateCurrencies(currencies, false, true)
			if err != nil {
				log.Printf("%s failed to update currencies. Err: %s", o.Name, err)
			}
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKCoin) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	currency := exchange.FormatExchangeCurrency(o.Name, p).String()
	var tickerPrice ticker.Price

	if assetType != ticker.Spot && o.APIUrl == okcoinAPIURL {
		tick, err := o.GetFuturesTicker(currency, assetType)
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Sell
		tickerPrice.Bid = tick.Buy
		tickerPrice.Low = tick.Low
		tickerPrice.Last = tick.Last
		tickerPrice.Volume = tick.Vol
		tickerPrice.High = tick.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, assetType)
	} else {
		tick, err := o.GetTicker(currency)
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Sell
		tickerPrice.Bid = tick.Buy
		tickerPrice.Low = tick.Low
		tickerPrice.Last = tick.Last
		tickerPrice.Volume = tick.Vol
		tickerPrice.High = tick.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, ticker.Spot)

	}
	return ticker.GetTicker(o.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (o *OKCoin) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *OKCoin) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(o.GetName(), currency, assetType)
	if err != nil {
		return o.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKCoin) UpdateOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := o.GetOrderBook(exchange.FormatExchangeCurrency(o.Name, currency).String(), 200, false)
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

	orderbook.ProcessOrderbook(o.GetName(), currency, orderBook, assetType)
	return orderbook.GetOrderbook(o.Name, currency, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// OKCoin exchange
func (o *OKCoin) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = o.GetName()
	assets, err := o.GetUserInfo()
	if err != nil {
		return response, err
	}

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   assets.Info.Funds.Free.BTC,
		Hold:         assets.Info.Funds.Freezed.BTC,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "LTC",
		TotalValue:   assets.Info.Funds.Free.LTC,
		Hold:         assets.Info.Funds.Freezed.LTC,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   assets.Info.Funds.Free.USD,
		Hold:         assets.Info.Funds.Freezed.USD,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "CNY",
		TotalValue:   assets.Info.Funds.Free.CNY,
		Hold:         assets.Info.Funds.Freezed.CNY,
	})

	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (o *OKCoin) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *OKCoin) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (o *OKCoin) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (string, error) {
	var oT string
	if orderType == exchange.Limit {
		if side == exchange.Buy {
			oT = "buy"
		} else {
			oT = "sell"
		}
	} else if orderType == exchange.Market {
		if side == exchange.Buy {
			oT = "buy_market"
		} else {
			oT = "sell_market"
		}
	}

	response, err := o.Trade(amount, price, p.Pair().String(), oT)

	return fmt.Sprintf("%v", response), err
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *OKCoin) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (o *OKCoin) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (o *OKCoin) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (o *OKCoin) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (o *OKCoin) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *OKCoin) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKCoin) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKCoin) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// GetWebsocket returns a pointer to the exchange websocket
func (o *OKCoin) GetWebsocket() (*exchange.Websocket, error) {
	return o.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *OKCoin) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return o.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (o *OKCoin) GetWithdrawCapabilities() uint32 {
	return o.GetWithdrawPermissions()
}
