package bitstamp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Start starts the Bitstamp go routine
func (b *Bitstamp) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitstamp wrapper
func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Debugf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()))
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	pairs, err := b.GetTradingPairs()
	if err != nil {
		log.Errorf("%s failed to get trading pairs. Err: %s", b.Name, err)
	} else {
		var currencies []string
		for x := range pairs {
			if pairs[x].Trading != "Enabled" {
				continue
			}
			p := strings.Split(pairs[x].Name, "/")
			currencies = append(currencies, p[0]+p[1])
		}

		var newCurrencies currency.Pairs
		for _, p := range currencies {
			newCurrencies = append(newCurrencies,
				currency.NewCurrencyPairFromString(p))
		}

		err = b.UpdateCurrencies(newCurrencies, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", b.Name)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitstamp) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.String(), false)
	if err != nil {
		return tickerPrice, err

	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.Last = tick.Last
	tickerPrice.Volume = tick.Volume
	tickerPrice.High = tick.High
	ticker.ProcessTicker(b.GetName(), p, tickerPrice, assetType)
	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *Bitstamp) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitstamp) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)

}

// GetOrderbookEx returns the orderbook for a currency pair
func (b *Bitstamp) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount, Price: data.Price})
	}

	err = orderbook.ProcessOrderbook(b.GetName(), orderBook, assetType)
	if err != nil {
		return orderBook, err
	}

	return orderbook.GetOrderbook(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Bitstamp exchange
func (b *Bitstamp) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()
	accountBalance, err := b.GetBalance()
	if err != nil {
		return response, err
	}

	var currencies = []exchange.AccountCurrencyInfo{
		{
			CurrencyName: "BTC",
			TotalValue:   accountBalance.BTCAvailable,
			Hold:         accountBalance.BTCReserved,
		},
		{
			CurrencyName: "XRP",
			TotalValue:   accountBalance.XRPAvailable,
			Hold:         accountBalance.XRPReserved,
		},
		{
			CurrencyName: "USD",
			TotalValue:   accountBalance.USDAvailable,
			Hold:         accountBalance.USDReserved,
		},
		{
			CurrencyName: "EUR",
			TotalValue:   accountBalance.EURAvailable,
			Hold:         accountBalance.EURReserved,
		},
	}
	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitstamp) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitstamp) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitstamp) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	buy := side == exchange.BuyOrderSide
	market := orderType == exchange.MarketOrderType
	response, err := b.PlaceOrder(p.String(), price, amount, buy, market)

	if response.ID > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.ID)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitstamp) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitstamp) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}
	_, err = b.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitstamp) CancelAllOrders(_ exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	isCancelAllSuccessful, err := b.CancelAllExistingOrders()
	if !isCancelAllSuccessful {
		err = errors.New("cancel all orders failed. Bitstamp provides no further information. Check order status to verify")
	}

	return exchange.CancelAllOrdersResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (b *Bitstamp) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitstamp) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return b.GetCryptoDepositAddress(cryptocurrency)
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitstamp) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := b.CryptoWithdrawal(withdrawRequest.Amount, withdrawRequest.Address, withdrawRequest.Currency.String(), withdrawRequest.AddressTag, true)
	if err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", errors.New(resp.Error)
	}

	return resp.ID, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := b.OpenBankWithdrawal(withdrawRequest.Amount, withdrawRequest.Currency.String(),
		withdrawRequest.BankAccountName, withdrawRequest.IBAN, withdrawRequest.SwiftCode, withdrawRequest.BankAddress,
		withdrawRequest.BankPostalCode, withdrawRequest.BankCity, withdrawRequest.BankCountry,
		withdrawRequest.Description, sepaWithdrawal)
	if err != nil {
		return "", err
	}
	if resp.Status == errStr {
		return "", errors.New(resp.Reason)
	}

	return resp.ID, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	resp, err := b.OpenInternationalBankWithdrawal(withdrawRequest.Amount, withdrawRequest.Currency.String(),
		withdrawRequest.BankAccountName, withdrawRequest.IBAN, withdrawRequest.SwiftCode, withdrawRequest.BankAddress,
		withdrawRequest.BankPostalCode, withdrawRequest.BankCity, withdrawRequest.BankCountry,
		withdrawRequest.IntermediaryBankName, withdrawRequest.IntermediaryBankAddress, withdrawRequest.IntermediaryBankPostalCode,
		withdrawRequest.IntermediaryBankCity, withdrawRequest.IntermediaryBankCountry, withdrawRequest.WireCurrency,
		withdrawRequest.Description, internationalWithdrawal)
	if err != nil {
		return "", err
	}
	if resp.Status == errStr {
		return "", errors.New(resp.Reason)
	}

	return resp.ID, nil
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitstamp) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitstamp) GetActiveOrders(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	var currPair string
	if len(getOrdersRequest.Currencies) != 1 {
		currPair = "all"
	} else {
		currPair = getOrdersRequest.Currencies[0].String()
	}

	resp, err := b.GetOpenOrders(currPair)
	if err != nil {
		return nil, err
	}

	for _, order := range resp {
		symbolOne := currency.Code(order.Currency[0:3])
		symbolTwo := currency.Code(order.Currency[len(order.Currency)-3:])
		orderDate := time.Unix(order.Date, 0)

		orders = append(orders, exchange.OrderDetail{
			Amount:       order.Amount,
			ID:           fmt.Sprintf("%v", order.ID),
			Price:        order.Price,
			OrderDate:    orderDate,
			CurrencyPair: currency.NewCurrencyPair(symbolOne, symbolTwo),
			Exchange:     b.Name,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitstamp) GetOrderHistory(getOrdersRequest exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var currPair string
	if len(getOrdersRequest.Currencies) == 1 {
		currPair = getOrdersRequest.Currencies[0].String()
	}
	resp, err := b.GetUserTransactions(currPair)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for _, order := range resp {
		if order.Type != 2 {
			continue
		}
		var quoteCurrency, baseCurrency currency.Code

		switch {
		case order.BTC > 0:
			baseCurrency = currency.BTC
		case order.XRP > 0:
			baseCurrency = currency.XRP
		default:
			log.Warnf("no base currency found for OrderID '%v'", order.OrderID)
		}

		switch {
		case order.USD > 0:
			quoteCurrency = currency.USD
		case order.EUR > 0:
			quoteCurrency = currency.EUR
		default:
			log.Warnf("no quote currency found for orderID '%v'", order.OrderID)
		}

		var currPair currency.Pair
		if quoteCurrency.String() != "" && baseCurrency.String() != "" {
			currPair = currency.NewCurrencyPairWithDelimiter(baseCurrency.String(),
				quoteCurrency.String(),
				b.ConfigCurrencyPairFormat.Delimiter)
		}
		orderDate := time.Unix(order.Date, 0)

		orders = append(orders, exchange.OrderDetail{
			ID:           fmt.Sprintf("%v", order.OrderID),
			OrderDate:    orderDate,
			Exchange:     b.Name,
			CurrencyPair: currPair,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)

	return orders, nil
}
