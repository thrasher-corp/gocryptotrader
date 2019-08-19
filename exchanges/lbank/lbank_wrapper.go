package lbank

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Start starts the Lbank go routine
func (l *Lbank) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the Lbank wrapper
func (l *Lbank) Run() {
	if l.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", l.GetName(), common.IsEnabled(l.Websocket.IsEnabled()), l.Websocket.GetWebsocketURL())
		log.Debugf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}
	exchangeCurrencies, err := l.GetCurrencyPairs()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", l.GetName())
	} else {
		var newExchangeCurrencies currency.Pairs
		for _, p := range exchangeCurrencies {
			newExchangeCurrencies = append(newExchangeCurrencies,
				currency.NewPairFromString(p))
		}
		err = l.UpdateCurrencies(newExchangeCurrencies, false, true)
		if err != nil {
			log.Errorf("%s Failed to update available currencies %s.\n", l.GetName(), err)
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *Lbank) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tickerInfo, err := l.GetTicker(exchange.FormatExchangeCurrency(l.Name, p).String())
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Pair = p
	tickerPrice.Last = tickerInfo.Ticker.Latest
	tickerPrice.High = tickerInfo.Ticker.High
	tickerPrice.Volume = tickerInfo.Ticker.Volume
	tickerPrice.Low = tickerInfo.Ticker.Low

	err = ticker.ProcessTicker(l.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(l.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *Lbank) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), exchange.FormatExchangeCurrency(l.Name, p), assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *Lbank) GetOrderbookEx(currency currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(l.GetName(), currency, assetType)
	if err != nil {
		return l.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *Lbank) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	a, err := l.GetMarketDepths(exchange.FormatExchangeCurrency(l.Name, p).String(), "60", "1")
	if err != nil {
		return orderBook, err
	}
	for i := range a.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Price:  a.Asks[i][0],
			Amount: a.Asks[i][1]})
	}
	for i := range a.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Price:  a.Bids[i][0],
			Amount: a.Bids[i][1]})
	}
	orderBook.Pair = p
	orderBook.ExchangeName = l.GetName()
	orderBook.AssetType = assetType
	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(l.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Lbank exchange
func (l *Lbank) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	data, err := l.GetUserInfo()
	if err != nil {
		return info, err
	}
	var account exchange.Account
	for key, val := range data.Info.Asset {
		c := currency.NewCode(key)
		hold, ok := data.Info.Freeze[key]
		if !ok {
			return info, fmt.Errorf("hold data not found with %s", key)
		}
		totalVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return info, err
		}
		totalHold, err := strconv.ParseFloat(hold, 64)
		if err != nil {
			return info, err
		}
		account.Currencies = append(account.Currencies,
			exchange.AccountCurrencyInfo{CurrencyName: c,
				TotalValue: totalVal,
				Hold:       totalHold})
	}

	info.Accounts = append(info.Accounts, account)
	info.Exchange = l.GetName()
	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *Lbank) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *Lbank) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (l *Lbank) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var resp exchange.SubmitOrderResponse
	if side != exchange.BuyOrderSide && side != exchange.SellOrderSide {
		return resp, fmt.Errorf("%s orderside is not supported by the exchange", side)
	}
	tempResp, err := l.CreateOrder(exchange.FormatExchangeCurrency(l.Name, p).String(), side.ToString(), amount, price)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = tempResp.OrderID
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *Lbank) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *Lbank) CancelOrder(order *exchange.OrderCancellation) error {
	_, err := l.RemoveOrder(exchange.FormatExchangeCurrency(l.Name, order.CurrencyPair).String(), order.OrderID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *Lbank) CancelAllOrders(orders *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	var resp exchange.CancelAllOrdersResponse
	orderIDs, err := l.getAllOpenOrderID()
	if err != nil {
		return resp, nil
	}

	for key := range orderIDs {
		if key != orders.CurrencyPair.String() {
			continue
		}
		var x, y = 0, 0
		var input string
		var tempSlice []string
		for x <= len(orderIDs[key]) {
			x++
			for y != x {
				tempSlice = append(tempSlice, orderIDs[key][y])
				if y%3 == 0 {
					input = strings.Join(tempSlice, ",")
					CancelResponse, err2 := l.RemoveOrder(key, input)
					if err2 != nil {
						return resp, err2
					}
					tempStringSuccess := strings.Split(CancelResponse.Success, ",")
					for k := range tempStringSuccess {
						resp.OrderStatus[tempStringSuccess[k]] = "Cancelled"
					}
					tempStringError := strings.Split(CancelResponse.Err, ",")
					for l := range tempStringError {
						resp.OrderStatus[tempStringError[l]] = "Failed"
					}
					tempSlice = tempSlice[:0]
					y++
				}
				y++
			}
			input = strings.Join(tempSlice, ",")
			CancelResponse, err2 := l.RemoveOrder(key, input)
			if err2 != nil {
				return resp, err2
			}
			tempStringSuccess := strings.Split(CancelResponse.Success, ",")
			for k := range tempStringSuccess {
				resp.OrderStatus[tempStringSuccess[k]] = "Cancelled"
			}
			tempStringError := strings.Split(CancelResponse.Err, ",")
			for l := range tempStringError {
				resp.OrderStatus[tempStringError[l]] = "Failed"
			}
			tempSlice = tempSlice[:0]
		}
	}
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (l *Lbank) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var resp exchange.OrderDetail
	orderIDs, err := l.getAllOpenOrderID()
	if err != nil {
		return resp, err
	}

	for key, val := range orderIDs {
		for i := range val {
			if val[i] != orderID {
				continue
			}
			tempResp, err := l.QueryOrder(key, orderID)
			if err != nil {
				return resp, err
			}
			resp.Exchange = l.GetName()
			resp.CurrencyPair = currency.NewPairFromString(key)
			if strings.EqualFold(tempResp.Orders[0].Type, "buy") {
				resp.OrderSide = exchange.BuyOrderSide
			} else {
				resp.OrderSide = exchange.SellOrderSide
			}
			z := tempResp.Orders[0].Status
			switch {
			case z == -1:
				resp.Status = "cancelled"
			case z == 0:
				resp.Status = "on trading"
			case z == 1:
				resp.Status = "filled partially"
			case z == 2:
				resp.Status = "Filled totally"
			case z == 4:
				resp.Status = "Cancelling"
			default:
				resp.Status = "Invalid Order Status"
			}
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(&exchange.FeeBuilder{
				FeeType:       exchange.CryptocurrencyTradeFee,
				Amount:        tempResp.Orders[0].Amount,
				PurchasePrice: tempResp.Orders[0].Price})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
		}
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *Lbank) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	resp, err := l.Withdraw(withdrawRequest.Address, withdrawRequest.Currency.String(), strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64), "", withdrawRequest.Description)
	return resp.WithdrawID, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *Lbank) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (l *Lbank) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var finalResp []exchange.OrderDetail
	var resp exchange.OrderDetail
	tempData, err := l.getAllOpenOrderID()
	if err != nil {
		return finalResp, err
	}

	for key, val := range tempData {
		for x := range val {
			tempResp, err := l.QueryOrder(key, val[x])
			if err != nil {
				return finalResp, err
			}
			resp.Exchange = l.GetName()
			resp.CurrencyPair = currency.NewPairFromString(key)
			if strings.EqualFold(tempResp.Orders[0].Type, "buy") {
				resp.OrderSide = exchange.BuyOrderSide
			} else {
				resp.OrderSide = exchange.SellOrderSide
			}
			z := tempResp.Orders[0].Status
			switch {
			case z == -1:
				resp.Status = "cancelled"
			case z == 1:
				resp.Status = "on trading"
			case z == 2:
				resp.Status = "filled partially"
			case z == 3:
				resp.Status = "Filled totally"
			case z == 4:
				resp.Status = "Cancelling"
			default:
				resp.Status = "Invalid Order Status"
			}
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.OrderDate = time.Unix(tempResp.Orders[0].CreateTime, 9)
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(&exchange.FeeBuilder{
				FeeType:       exchange.CryptocurrencyTradeFee,
				Amount:        tempResp.Orders[0].Amount,
				PurchasePrice: tempResp.Orders[0].Price})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
			for y := int(0); y < len(getOrdersRequest.Currencies); y++ {
				if getOrdersRequest.Currencies[y].String() != key {
					continue
				}
				if getOrdersRequest.OrderSide == "ANY" {
					finalResp = append(finalResp, resp)
					continue
				}
				if strings.EqualFold(getOrdersRequest.OrderSide.ToString(), tempResp.Orders[0].Type) {
					finalResp = append(finalResp, resp)
				}
			}
		}
	}
	return finalResp, nil
}

// GetOrderHistory retrieves account order information *
// Can Limit response to specific order status
func (l *Lbank) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var finalResp []exchange.OrderDetail
	var resp exchange.OrderDetail
	var tempCurr currency.Pairs
	var x int
	if len(getOrdersRequest.Currencies) == 0 {
		tempCurr = l.GetEnabledCurrencies()
	} else {
		for x < len(getOrdersRequest.Currencies) {
			tempCurr = getOrdersRequest.Currencies
		}
	}
	for a := range tempCurr {
		p := exchange.FormatExchangeCurrency(l.Name, tempCurr[a])
		b := int64(1)
		tempResp, err := l.QueryOrderHistory(exchange.FormatExchangeCurrency(l.Name, p).String(), strconv.FormatInt(b, 10), "200")
		if err != nil {
			return finalResp, err
		}
		for len(tempResp.Orders) != 0 {
			tempResp, err = l.QueryOrderHistory(exchange.FormatExchangeCurrency(l.Name, p).String(), strconv.FormatInt(b, 10), "200")
			if err != nil {
				return finalResp, err
			}
			for x := 0; x < len(tempResp.Orders); x++ {
				resp.Exchange = l.GetName()
				resp.CurrencyPair = currency.NewPairFromString(tempResp.Orders[x].Symbol)
				if strings.EqualFold(tempResp.Orders[x].Type, "buy") {
					resp.OrderSide = exchange.BuyOrderSide
				} else {
					resp.OrderSide = exchange.SellOrderSide
				}
				z := tempResp.Orders[x].Status
				switch {
				case z == -1:
					resp.Status = "cancelled"
				case z == 1:
					resp.Status = "on trading"
				case z == 2:
					resp.Status = "filled partially"
				case z == 3:
					resp.Status = "Filled totally"
				case z == 4:
					resp.Status = "Cancelling"
				default:
					resp.Status = "Invalid Order Status"
				}
				resp.Price = tempResp.Orders[x].Price
				resp.Amount = tempResp.Orders[x].Amount
				resp.OrderDate = time.Unix(tempResp.Orders[x].CreateTime, 9)
				resp.ExecutedAmount = tempResp.Orders[x].DealAmount
				resp.RemainingAmount = tempResp.Orders[x].Price - tempResp.Orders[x].DealAmount
				resp.Fee, err = l.GetFeeByType(&exchange.FeeBuilder{
					FeeType:       exchange.CryptocurrencyTradeFee,
					Amount:        tempResp.Orders[x].Amount,
					PurchasePrice: tempResp.Orders[x].Price})
				if err != nil {
					resp.Fee = lbankFeeNotFound
				}
				finalResp = append(finalResp, resp)
				b++
			}
		}
	}
	return finalResp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction *
func (l *Lbank) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var resp float64
	if feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		return feeBuilder.Amount * feeBuilder.PurchasePrice * l.Fee, nil
	}
	if feeBuilder.FeeType == exchange.CryptocurrencyWithdrawalFee {
		withdrawalFee, err := l.GetWithdrawConfig(feeBuilder.Pair.Base.Lower().String())
		if err != nil {
			return resp, err
		}
		var tempFee string
		temp := strings.Split(withdrawalFee[0].Fee, ":\"")
		if len(temp) > 1 {
			tempFee = strings.TrimRight(temp[1], ",\"type")
		} else {
			tempFee = temp[0]
		}
		resp, err = strconv.ParseFloat(tempFee, 64)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// GetAllOpenOrderID returns all open orders by currency pairs
func (l *Lbank) getAllOpenOrderID() (map[string][]string, error) {
	allPairs := l.GetEnabledCurrencies()
	resp := make(map[string][]string)
	for a := range allPairs {
		p := exchange.FormatExchangeCurrency(l.Name, allPairs[a])
		b := int64(1)
		tempResp, err := l.GetOpenOrders(exchange.FormatExchangeCurrency(l.Name, p).String(), strconv.FormatInt(b, 10), "200")
		if err != nil {
			return resp, err
		}
		tempData := len(tempResp.Orders)
		for tempData != 0 {
			tempResp, err = l.GetOpenOrders(exchange.FormatExchangeCurrency(l.Name, p).String(), strconv.FormatInt(b, 10), "200")
			if err != nil {
				return resp, err
			}

			if len(tempResp.Orders) == 0 {
				return resp, nil
			}

			for c := 0; c < tempData; c++ {
				resp[exchange.FormatExchangeCurrency(l.Name, p).String()] = append(resp[exchange.FormatExchangeCurrency(l.Name, p).String()], tempResp.Orders[c].OrderID)

			}
			tempData = len(tempResp.Orders)
			b++
		}
	}
	return resp, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (l *Lbank) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrNotYetImplemented
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (l *Lbank) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrNotYetImplemented
}

// AuthenticateWebsocket authenticates it
func (l *Lbank) AuthenticateWebsocket() error {
	return common.ErrNotYetImplemented
}

// GetSubscriptions gets subscriptions
func (l *Lbank) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrNotYetImplemented
}
