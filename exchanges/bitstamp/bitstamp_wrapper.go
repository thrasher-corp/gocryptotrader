package bitstamp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
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
			pair := strings.Split(pairs[x].Name, "/")
			currencies = append(currencies, pair[0]+pair[1])
		}
		err = b.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available currencies.\n", b.Name)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitstamp) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.Pair().String(), false)
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
func (b *Bitstamp) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
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
func (b *Bitstamp) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.Pair().String())
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

	orderbook.ProcessOrderbook(b.GetName(), p, orderBook, assetType)
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

	var currencies []exchange.AccountCurrencyInfo

	currencies = append(currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   accountBalance.BTCAvailable,
		Hold:         accountBalance.BTCReserved,
	})

	currencies = append(currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "XRP",
		TotalValue:   accountBalance.XRPAvailable,
		Hold:         accountBalance.XRPReserved,
	})

	currencies = append(currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   accountBalance.USDAvailable,
		Hold:         accountBalance.USDReserved,
	})

	currencies = append(currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "EUR",
		TotalValue:   accountBalance.EURAvailable,
		Hold:         accountBalance.EURReserved,
	})

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
func (b *Bitstamp) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitstamp) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	buy := side == exchange.Buy
	market := orderType == exchange.Market
	response, err := b.PlaceOrder(p.Pair().String(), price, amount, buy, market)

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
func (b *Bitstamp) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	isCancelAllSuccessful, err := b.CancelAllExistingOrders()
	if !isCancelAllSuccessful {
		err = errors.New("Cancel all failed. Bitstamp provides no further information. Check order status to verify")
	}

	return exchange.CancelAllOrdersResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (b *Bitstamp) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitstamp) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	return b.GetCryptoDepositAddress(cryptocurrency.String())
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

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *Bitstamp) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitstamp) GetOrderHistory(orderHistoryRequest exchange.OrderHistoryRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	var currPair string
	if orderHistoryRequest.OrderStatus == exchange.ActiveOrderStatus || orderHistoryRequest.OrderStatus == exchange.AnyOrderStatus {
		if len(orderHistoryRequest.Currencies) > 1 || len(orderHistoryRequest.Currencies) <= 0 {
			currPair = "all"
		} else {
			currPair = orderHistoryRequest.Currencies[0]
		}

		resp, err := b.GetOpenOrders(currPair)
		if err != nil {
			return nil, err
		}

		for _, order := range resp {
			symbolOne := order.Currency[0:3]
			symbolTwo := order.Currency[len(order.Currency)-3:]

			orders = append(orders, exchange.OrderDetail{
				Amount:              order.Amount,
				ID:                  fmt.Sprintf("%v", order.ID),
				Price:               order.Price,
				OrderPlacementTicks: order.Date,
				BaseCurrency:        symbolOne,
				Status:              string(exchange.ActiveOrderStatus),
				OrderType:           string(exchange.AnyOrderType),
				QuoteCurrency:       symbolTwo,
			})
		}
	}

	if orderHistoryRequest.OrderStatus != exchange.ActiveOrderStatus {
		if len(orderHistoryRequest.Currencies) > 1 || len(orderHistoryRequest.Currencies) <= 0 {
			currPair = ""
		} else {
			currPair = orderHistoryRequest.Currencies[0]
		}
		resp, err := b.GetUserTransactions(currPair)
		if err != nil {
			return nil, err
		}

		for _, order := range resp {
			if order.Type == 2 {
				orders = append(orders, exchange.OrderDetail{
					ID:                  fmt.Sprintf("%v", order.OrderID),
					OrderPlacementTicks: order.Date,
					Status:              string(exchange.FilledOrderStatus),
					OrderType:           string(exchange.AnyOrderType),
				})
			}
		}
	}

	b.FilterOrdersByStatusAndType(&orders, orderHistoryRequest.OrderType, orderHistoryRequest.OrderStatus)
	b.FilterOrdersByTickRange(&orders, orderHistoryRequest.StartTicks, orderHistoryRequest.EndTicks)
	b.FilterOrdersByCurrencies(&orders, orderHistoryRequest.Currencies)

	return orders, nil
}
