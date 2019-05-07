package btcmarkets

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
		log.Debugf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	markets, err := b.GetMarkets()
	if err != nil {
		log.Errorf("%s failed to get active market. Err: %s", b.Name, err)
	} else {
		forceUpgrade := false
		if !common.StringDataContains(b.EnabledPairs.Strings(), "-") ||
			!common.StringDataContains(b.AvailablePairs.Strings(), "-") {
			forceUpgrade = true
		}

		var currencies currency.Pairs
		for x := range markets {
			currencies = append(currencies,
				currency.NewPairWithDelimiter(markets[x].Instrument,
					markets[x].Currency, "-"))
		}

		if forceUpgrade {
			enabledPairs := currency.Pairs{currency.Pair{Base: currency.BTC,
				Quote: currency.AUD, Delimiter: "-"}}

			log.Warn("Available pairs for BTC Makrets reset due to config upgrade, please enable the pairs you would like again.")

			err = b.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Errorf("%s failed to update currencies. Err: %s", b.Name, err)
			}
		}
		err = b.UpdateCurrencies(currencies, false, forceUpgrade)
		if err != nil {
			log.Errorf("%s failed to update currencies. Err: %s", b.Name, err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCMarkets) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.Base.String(), p.Quote.String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Ask = tick.BestAsk
	tickerPrice.Bid = tick.BestBID
	tickerPrice.Last = tick.LastPrice

	err = ticker.ProcessTicker(b.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(b.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (b *BTCMarkets) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (b *BTCMarkets) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCMarkets) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(p.Base.String(),
		p.Quote.String())
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

	orderBook.Pair = p
	orderBook.ExchangeName = b.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// BTCMarkets exchange
func (b *BTCMarkets) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()

	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].PendingFunds

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCMarkets) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCMarkets) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTCMarkets) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	response, err := b.NewOrder(p.Base.Upper().String(),
		p.Quote.Upper().String(),
		price,
		amount,
		side.ToString(),
		orderType.ToString(),
		clientID)

	if response > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCMarkets) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCMarkets) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder([]int64{orderIDInt})
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCMarkets) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := b.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var orderList []int64
	for i := range openOrders {
		orderIDInt, err := strconv.ParseInt(openOrders[i].ID, 10, 64)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[openOrders[i].ID] = err.Error()
		}
		orderList = append(orderList, orderIDInt)
	}

	if len(orderList) > 0 {
		orders, err := b.CancelExistingOrder(orderList)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for i := range orders {
			if err != nil {
				cancelAllOrdersResponse.OrderStatus[strconv.FormatInt(orders[i].ID, 10)] = err.Error()
			}
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTCMarkets) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var OrderDetail exchange.OrderDetail

	o, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return OrderDetail, err
	}

	orders, err := b.GetOrderDetail([]int64{o})
	if err != nil {
		return OrderDetail, err
	}

	if len(orders) > 1 {
		return OrderDetail, errors.New("too many orders returned")
	}

	if len(orders) == 0 {
		return OrderDetail, errors.New("no orders found")
	}

	for i := range orders {
		var side exchange.OrderSide
		if strings.EqualFold(orders[i].OrderSide, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		} else if strings.EqualFold(orders[i].OrderSide, exchange.BidOrderSide.ToString()) {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(orders[i].CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(orders[i].OrderType))

		OrderDetail.Amount = orders[i].Volume
		OrderDetail.OrderDate = orderDate
		OrderDetail.Exchange = b.GetName()
		OrderDetail.ID = orders[i].ID
		OrderDetail.RemainingAmount = orders[i].OpenVolume
		OrderDetail.OrderSide = side
		OrderDetail.OrderType = orderType
		OrderDetail.Price = orders[i].Price
		OrderDetail.Status = orders[i].Status
		OrderDetail.CurrencyPair = currency.NewPairWithDelimiter(orders[i].Instrument,
			orders[i].Currency,
			b.ConfigCurrencyPairFormat.Delimiter)
	}

	return OrderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCMarkets) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return b.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Address)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	if withdrawRequest.Currency != currency.AUD {
		return "", errors.New("only AUD is supported for withdrawals")
	}
	return b.WithdrawAUD(withdrawRequest.BankAccountName, fmt.Sprintf("%v", withdrawRequest.BankAccountNumber), withdrawRequest.BankName, fmt.Sprintf("%v", withdrawRequest.BankCode), withdrawRequest.Amount)
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTCMarkets) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCMarkets) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (b.APIKey == "" || b.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTCMarkets) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := b.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp {
		var side exchange.OrderSide
		if strings.EqualFold(resp[i].OrderSide, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		} else if strings.EqualFold(resp[i].OrderSide, exchange.BidOrderSide.ToString()) {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(resp[i].CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(resp[i].OrderType))

		openOrder := exchange.OrderDetail{
			ID:              resp[i].ID,
			Amount:          resp[i].Volume,
			Exchange:        b.Name,
			RemainingAmount: resp[i].OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           resp[i].Price,
			Status:          resp[i].Status,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].Instrument,
				resp[i].Currency,
				b.ConfigCurrencyPairFormat.Delimiter),
		}

		for j := range resp[i].Trades {
			tradeDate := time.Unix(int64(resp[i].Trades[j].CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Amount:      resp[i].Trades[j].Volume,
				Exchange:    b.Name,
				Price:       resp[i].Trades[j].Price,
				TID:         resp[i].Trades[j].ID,
				Timestamp:   tradeDate,
				Fee:         resp[i].Trades[j].Fee,
				Description: resp[i].Trades[j].Description,
			})
		}

		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCMarkets) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	if len(getOrdersRequest.Currencies) == 0 {
		return nil, errors.New("requires at least one currency pair to retrieve history")
	}

	var respOrders []Order
	for _, currency := range getOrdersRequest.Currencies {
		resp, err := b.GetOrders(currency.Base.String(),
			currency.Quote.String(),
			200,
			0,
			true)
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []exchange.OrderDetail
	for i := range respOrders {
		var side exchange.OrderSide
		if strings.EqualFold(respOrders[i].OrderSide, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		} else if strings.EqualFold(respOrders[i].OrderSide, exchange.BidOrderSide.ToString()) {
			side = exchange.BuyOrderSide
		}
		orderDate := time.Unix(int64(respOrders[i].CreationTime), 0)
		orderType := exchange.OrderType(strings.ToUpper(respOrders[i].OrderType))

		openOrder := exchange.OrderDetail{
			ID:              respOrders[i].ID,
			Amount:          respOrders[i].Volume,
			Exchange:        b.Name,
			RemainingAmount: respOrders[i].OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           respOrders[i].Price,
			Status:          respOrders[i].Status,
			CurrencyPair: currency.NewPairWithDelimiter(respOrders[i].Instrument,
				respOrders[i].Currency,
				b.ConfigCurrencyPairFormat.Delimiter),
		}

		for j := range respOrders[i].Trades {
			tradeDate := time.Unix(int64(respOrders[i].Trades[j].CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Amount:      respOrders[i].Trades[j].Volume,
				Exchange:    b.Name,
				Price:       respOrders[i].Trades[j].Price,
				TID:         respOrders[i].Trades[j].ID,
				Timestamp:   tradeDate,
				Fee:         respOrders[i].Trades[j].Fee,
				Description: respOrders[i].Trades[j].Description,
			})
		}

		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *BTCMarkets) SubscribeToWebsocketChannels(channels []exchange.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *BTCMarkets) UnsubscribeToWebsocketChannels(channels []exchange.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}