package bitfinex

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
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
		log.Debugf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()))
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	exchangeProducts, err := b.GetSymbols()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", b.GetName())
	} else {
		err = b.UpdateCurrencies(exchangeProducts, false, false)
		if err != nil {
			log.Errorf("%s Failed to update available symbols.\n", b.GetName())
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

// GetAccountInfo retrieves balances for all enabled currencies on the
// Bitfinex exchange
func (b *Bitfinex) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()
	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}

	var Accounts = []exchange.Account{
		{ID: "deposit"},
		{ID: "exchange"},
		{ID: "trading"},
	}

	for _, bal := range accountBalance {
		for i := range Accounts {
			if Accounts[i].ID == bal.Type {
				Accounts[i].Currencies = append(Accounts[i].Currencies,
					exchange.AccountCurrencyInfo{
						CurrencyName: bal.Currency,
						TotalValue:   bal.Amount,
						Hold:         bal.Amount - bal.Available,
					})
			}
		}
	}

	response.Accounts = Accounts
	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitfinex) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitfinex) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitfinex) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var isBuying bool

	if side == exchange.Buy {
		isBuying = true
	}

	response, err := b.NewOrder(p.Pair().String(), amount, price, isBuying, orderType.ToString(), false)

	if response.OrderID > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response.OrderID)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitfinex) ModifyOrder(action exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitfinex) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder(orderIDInt)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitfinex) CancelAllOrders(orderCancellation exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	_, err := b.CancelAllExistingOrders()
	return exchange.CancelAllOrdersResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (b *Bitfinex) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitfinex) GetDepositAddress(cryptocurrency pair.CurrencyItem, accountID string) (string, error) {
	method, err := b.ConvertSymbolToDepositMethod(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	resp, err := b.NewDeposit(method, accountID, 0)
	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *Bitfinex) WithdrawCryptocurrencyFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	withdrawalType := b.ConvertSymbolToWithdrawalType(withdrawRequest.Currency.String())
	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawCryptocurrency(withdrawalType, walletType, withdrawRequest.Address, withdrawRequest.Currency.String(), withdrawRequest.Description, withdrawRequest.Amount)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errors.New("No withdrawID returned. Check order status")
	}

	return fmt.Sprintf("%v", resp[0].WithdrawalID), err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
// Returns comma delimited withdrawal IDs
func (b *Bitfinex) WithdrawFiatFunds(withdrawRequest exchange.WithdrawRequest) (string, error) {
	withdrawalType := "wire"
	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawFIAT(withdrawalType, walletType, withdrawRequest.WireCurrency,
		withdrawRequest.BankAccountName, withdrawRequest.BankName, withdrawRequest.BankAddress,
		withdrawRequest.BankCity, withdrawRequest.BankCountry, withdrawRequest.SwiftCode,
		withdrawRequest.Description, withdrawRequest.IntermediaryBankName, withdrawRequest.IntermediaryBankAddress,
		withdrawRequest.IntermediaryBankCity, withdrawRequest.IntermediaryBankCountry, withdrawRequest.IntermediarySwiftCode,
		withdrawRequest.Amount, withdrawRequest.BankAccountNumber, withdrawRequest.IntermediaryBankAccountNumber,
		withdrawRequest.IsExpressWire, withdrawRequest.RequiresIntermediaryBank)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errors.New("No withdrawID returned. Check order status")
	}

	var withdrawalSuccesses string
	var withdrawalErrors string
	for _, withdrawal := range resp {
		if withdrawal.Status == "error" {
			withdrawalErrors += fmt.Sprintf("%v ", withdrawal.Message)
		}
		if withdrawal.Status == "success" {
			withdrawalSuccesses += fmt.Sprintf("%v,", withdrawal.WithdrawalID)
		}
	}
	if len(withdrawalErrors) > 0 {
		return withdrawalSuccesses, errors.New(withdrawalErrors)
	}

	return withdrawalSuccesses, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
// Returns comma delimited withdrawal IDs
func (b *Bitfinex) WithdrawFiatFundsToInternationalBank(withdrawRequest exchange.WithdrawRequest) (string, error) {
	return b.WithdrawFiatFunds(withdrawRequest)
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitfinex) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return b.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (b *Bitfinex) GetWithdrawCapabilities() uint32 {
	return b.GetWithdrawPermissions()
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitfinex) GetOrderHistory(orderHistoryRequest exchange.OrderHistoryRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}
