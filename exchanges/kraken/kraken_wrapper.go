package kraken

import (
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the Kraken go routine
func (k *Kraken) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		k.Run()
		wg.Done()
	}()
}

// Run implements the Kraken wrapper
func (k *Kraken) Run() {
	if k.Verbose {
		log.Printf("%s polling delay: %ds.\n", k.GetName(), k.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", k.GetName(), len(k.EnabledPairs), k.EnabledPairs)
	}

	assetPairs, err := k.GetAssetPairs()
	if err != nil {
		log.Printf("%s Failed to get available symbols.\n", k.GetName())
	} else {
		forceUpgrade := false
		if !common.StringDataContains(k.EnabledPairs, "-") || !common.StringDataContains(k.AvailablePairs, "-") {
			forceUpgrade = true
		}

		var exchangeProducts []string
		for _, v := range assetPairs {
			if common.StringContains(v.Altname, ".d") {
				continue
			}
			if v.Base[0] == 'X' {
				if len(v.Base) > 3 {
					v.Base = v.Base[1:]
				}
			}
			if v.Quote[0] == 'Z' || v.Quote[0] == 'X' {
				v.Quote = v.Quote[1:]
			}
			exchangeProducts = append(exchangeProducts, v.Base+"-"+v.Quote)
		}

		if forceUpgrade {
			enabledPairs := []string{"XBT-USD"}
			log.Println("WARNING: Available pairs for Kraken reset due to config upgrade, please enable the ones you would like again")

			err = k.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Printf("%s Failed to get config.\n", k.GetName())
			}
		}
		err = k.UpdateCurrencies(exchangeProducts, false, forceUpgrade)
		if err != nil {
			log.Printf("%s Failed to get config.\n", k.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (k *Kraken) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	pairs := k.GetEnabledCurrencies()
	pairsCollated, err := exchange.GetAndFormatExchangeCurrencies(k.Name, pairs)
	if err != nil {
		return tickerPrice, err
	}
	tickers, err := k.GetTickers(pairsCollated.String())
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range pairs {
		for y, z := range tickers {
			if common.StringContains(y, x.FirstCurrency.Upper().String()) && common.StringContains(y, x.SecondCurrency.Upper().String()) {
				var tp ticker.Price
				tp.Pair = x
				tp.Last = z.Last
				tp.Ask = z.Ask
				tp.Bid = z.Bid
				tp.High = z.High
				tp.Low = z.Low
				tp.Volume = z.Volume
				ticker.ProcessTicker(k.GetName(), x, tp, assetType)
			}
		}
	}
	return ticker.GetTicker(k.GetName(), p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (k *Kraken) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(k.GetName(), p, assetType)
	if err != nil {
		return k.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (k *Kraken) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(k.GetName(), p, assetType)
	if err != nil {
		return k.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (k *Kraken) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := k.GetDepth(exchange.FormatExchangeCurrency(k.GetName(), p).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderbook.ProcessOrderbook(k.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(k.Name, p, assetType)
}

// GetExchangeAccountInfo retrieves balances for all enabled currencies for the
// Kraken exchange - to-do
func (k *Kraken) GetExchangeAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = k.GetName()
	return response, nil
}

// GetExchangeFundTransferHistory returns funding history, deposits and
// withdrawals
func (k *Kraken) GetExchangeFundTransferHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, errors.New("not supported on exchange")
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (k *Kraken) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, errors.New("trade history not yet implemented")
}

// SubmitExchangeOrder submits a new order
func (k *Kraken) SubmitExchangeOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (string, error) {
	var args = AddOrderOptions{}

	response, err := k.AddOrder(p.Pair().String(), side.Format(k.Name), orderType.Format(k.Name), amount, price, 0, 0, args)
	orderIds := strings.Join(response.TransactionIds, ",")

	return orderIds, err
}

// ModifyExchangeOrder will allow of changing orderbook placement and limit to
// market conversion
func (k *Kraken) ModifyExchangeOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, errors.New("not yet implemented")
}

// CancelExchangeOrder cancels an order by its corresponding ID number
func (k *Kraken) CancelExchangeOrder(orderID int64) error {
	return errors.New("not yet implemented")
}

// CancelAllExchangeOrders cancels all orders associated with a currency pair
func (k *Kraken) CancelAllExchangeOrders() error {
	return errors.New("not yet implemented")
}

// GetExchangeOrderInfo returns information on a current open order
func (k *Kraken) GetExchangeOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, errors.New("not yet implemented")
}

// GetExchangeDepositAddress returns a deposit address for a specified currency
func (k *Kraken) GetExchangeDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawCryptoExchangeFunds returns a withdrawal ID when a withdrawal is
// submitted
func (k *Kraken) WithdrawCryptoExchangeFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFunds returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatExchangeFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// WithdrawFiatExchangeFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatExchangeFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", errors.New("not yet implemented")
}

// GetWebsocket returns a pointer to the exchange websocket
func (k *Kraken) GetWebsocket() (*exchange.Websocket, error) {
	return nil, errors.New("not yet implemented")
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (k *Kraken) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return k.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (k *Kraken) GetWithdrawCapabilities() uint32 {
	return k.GetWithdrawPermissions()
}
