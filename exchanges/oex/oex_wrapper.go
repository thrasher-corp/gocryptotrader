package oex

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

// Start starts the Oex go routine
func (o *Oex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the Oex wrapper
func (o *Oex) Run() {
	if o.Verbose {

		log.Debugf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}
	exchangeCurrencies, err := o.GetAllPairs()
	if err != nil {
		log.Errorf("%s Failed to get available symbols.\n", o.GetName())
	}
	var newExchangeCurrencies currency.Pairs
	for x := 0; x < len(exchangeCurrencies.Data); x++ {
		newExchangeCurrencies = append(newExchangeCurrencies, currency.NewPairFromString(exchangeCurrencies.Data[x].Symbol))
	}
	err = o.UpdateCurrencies(newExchangeCurrencies, false, true)
	if err != nil {
		log.Errorf("%s Failed to update available currencies %s.\n", o.GetName(), err)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *Oex) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var resp ticker.Price

	strPair := p.Lower().String()
	tempResp, err := o.GetTicker(strPair)
	if err != nil {
		return resp, err
	}
	high, err := strconv.ParseFloat(tempResp.Data.High, 64)
	if err != nil {
		return resp, err
	}
	low, err := strconv.ParseFloat(tempResp.Data.Low, 64)
	if err != nil {
		return resp, err
	}
	resp.Pair = p
	resp.Last = tempResp.Data.Last
	resp.High = high
	resp.Low = low
	resp.Bid = tempResp.Data.Buy
	resp.Ask = tempResp.Data.Sell
	tempAmount, err := strconv.ParseFloat(tempResp.Data.Volume, 64)
	if err != nil {
		return resp, err
	}
	resp.Volume = tempAmount
	resp.LastUpdated = time.Unix(0, tempResp.Data.Time)
	return resp, nil // NOTE DO NOT USE AS RETURN
}

// GetTickerPrice returns the ticker for a currency pair
func (o *Oex) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *Oex) GetOrderbookEx(currency currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(o.GetName(), currency, assetType)
	if err != nil {
		return o.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *Oex) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var resp orderbook.Base
	strPair := p.Lower().String()
	tempResp, err := o.GetMarketDepth(strPair, "step2")
	if err != nil {
		return resp, err
	}
	resp.ExchangeName = o.GetName()
	resp.Pair = p
	for i := range tempResp.Data.Tick.Bids {
		var tempBids orderbook.Item
		tempBids.Amount = tempResp.Data.Tick.Bids[i][1]
		tempBids.Price = tempResp.Data.Tick.Bids[i][0]

		resp.Bids = append(resp.Bids, tempBids)
	}
	for j := range tempResp.Data.Tick.Asks {
		var tempAsks orderbook.Item
		tempAsks.Amount = tempResp.Data.Tick.Asks[j][1]
		tempAsks.Price = tempResp.Data.Tick.Asks[j][0]

		resp.Bids = append(resp.Bids, tempAsks)
	}
	return resp, nil // NOTE DO NOT USE AS RETURN
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Oex exchange
func (o *Oex) GetAccountInfo() (exchange.AccountInfo, error) {
	var resp exchange.AccountInfo
	resp.Exchange = o.GetName()
	var tempData exchange.Account
	tempResp, err := o.GetUserInfo()
	if err != nil {
		return resp, err
	}
	for x := range tempResp.Data.CoinData {
		totalVal, err := strconv.ParseFloat(tempResp.Data.CoinData[x].Normal, 64)
		if err != nil {
			return resp, err
		}
		holdVal, err2 := strconv.ParseFloat(tempResp.Data.CoinData[x].Locked, 64)
		if err2 != nil {
			return resp, err2
		}
		tempData.Currencies = append(tempData.Currencies,
			exchange.AccountCurrencyInfo{CurrencyName: currency.NewCode(tempResp.Data.CoinData[x].Coin),
				TotalValue: totalVal,
				Hold:       holdVal})
	}
	resp.Accounts = append(resp.Accounts, tempData)
	return resp, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *Oex) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *Oex) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (o *Oex) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var resp exchange.SubmitOrderResponse
	if side != exchange.BuyOrderSide && side != exchange.SellOrderSide {
		return resp, fmt.Errorf("%s orderside is not supported by %s", side, o.GetName())
	}
	if orderType != exchange.LimitOrderType && orderType != exchange.MarketOrderType {
		return resp, fmt.Errorf("%s ordertype is not supported by %s", orderType, o.GetName())
	}
	tempResp, err := o.CreateOrder(side.ToString(), orderType.ToString(), strconv.FormatFloat(amount, 'f', -1, 64), strconv.FormatFloat(price, 'f', -1, 64), exchange.FormatExchangeCurrency(o.Name, p).String(), "")
	if err != nil {
		return resp, err
	}
	if tempResp.ErrCapture.Error == "0" {
		resp.IsOrderPlaced = true
		resp.OrderID = strconv.FormatInt(tempResp.Data.OrderID, 10)
	} else {
		resp.IsOrderPlaced = false
	}
	return resp, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *Oex) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (o *Oex) CancelOrder(order *exchange.OrderCancellation) error {
	_, err := o.RemoveOrder(order.OrderID, exchange.FormatExchangeCurrency(o.Name, order.CurrencyPair).String())
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (o *Oex) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	var resp exchange.CancelAllOrdersResponse
	tempData, err := o.getAllOpenOrderID()
	if err != nil {
		return resp, err
	}
	for key, val := range tempData {
		if key != orderCancellation.CurrencyPair.String() {
			continue
		}
		for x := 0; x < len(val); x++ {
			_, err := o.RemoveOrder(strconv.FormatInt(val[x], 10), key)
			if err != nil {
				resp.OrderStatus[strconv.FormatInt(val[x], 10)] = "Order Cancel Failed"
			}
			resp.OrderStatus[strconv.FormatInt(val[x], 10)] = "Order Cancelled"
		}
	}
	return exchange.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (o *Oex) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var resp exchange.OrderDetail
	tempData, err := o.getAllOpenOrderID()
	if err != nil {
		return resp, err
	}
	floatID, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return resp, err
	}
	for key, val := range tempData {
		for i := range val {
			if val[i] != floatID {
				continue
			}
			tempResp, err := o.FetchOrderInfo(key, orderID)
			if err != nil {
				return resp, err
			}
			resp.Exchange = o.Name
			resp.CurrencyPair = currency.NewPairFromString(key)
			if strings.EqualFold(tempResp.Data.OrderInfo.Side, "BUY") {
				resp.OrderSide = exchange.BuyOrderSide
			} else {
				resp.OrderSide = exchange.SellOrderSide
			}
			resp.Price = tempResp.Data.OrderInfo.Price
			tempAmount, err := strconv.ParseFloat(tempResp.Data.OrderInfo.Volume, 64)
			if err != nil {
				return resp, err
			}
			resp.Amount = tempAmount
			tempTime, err := strconv.ParseInt(tempResp.Data.OrderInfo.CreatedAt, 10, 64)
			if err != nil {
				return resp, err
			}
			resp.OrderDate = time.Unix(tempTime, 9)
			resp.ID = orderID
			tempExecAmount, err := strconv.ParseFloat(tempResp.Data.OrderInfo.DealVolume, 64)
			if err != nil {
				return resp, err
			}
			resp.ExecutedAmount = tempExecAmount
			resp.RemainingAmount = tempAmount - tempExecAmount
			resp.Fee = tempResp.Data.OrderInfo.Fee
		}
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *Oex) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *Oex) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *Oex) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (o *Oex) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (o *Oex) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (o *Oex) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var resp []exchange.OrderDetail
	var tempResp exchange.OrderDetail
	tempData, err := o.getAllOpenOrderID()
	if err != nil {
		return resp, err
	}
	if len(getOrdersRequest.Currencies) == 0 {
		for pair := range tempData {
			getOrdersRequest.Currencies = append(getOrdersRequest.Currencies, currency.NewPairFromString(pair))
		}
		for x := range getOrdersRequest.Currencies {
			key := exchange.FormatExchangeCurrency(o.Name, getOrdersRequest.Currencies[x]).String()
			for tempKey, val := range tempData {
				if tempKey != key {
					continue
				}
				for y := 0; y < len(val); y++ {
					tempData2, err := o.FetchOrderInfo(strconv.FormatInt(tempData[key][val[y]], 10), key)
					if err != nil {
						return resp, err
					}
					tempResp.Exchange = o.Name
					tempResp.CurrencyPair = currency.NewPairFromString(key)
					if strings.EqualFold(tempData2.Data.OrderInfo.Side, "BUY") {
						tempResp.OrderSide = exchange.BuyOrderSide
					} else {
						tempResp.OrderSide = exchange.SellOrderSide
					}
					tempResp.Price = tempData2.Data.OrderInfo.Price
					tempAmount, err := strconv.ParseFloat(tempData2.Data.OrderInfo.Volume, 64)
					if err != nil {
						return resp, err
					}
					tempResp.Amount = tempAmount
					tempTime, err2 := strconv.ParseInt(tempData2.Data.OrderInfo.CreatedAt, 10, 64)
					if err2 != nil {
						return resp, err2
					}
					tempResp.OrderDate = time.Unix(tempTime, 9)
					tempResp.ID = strconv.FormatInt(val[y], 10)
					tempExecAmount, err2 := strconv.ParseFloat(tempData2.Data.OrderInfo.DealVolume, 64)
					if err2 != nil {
						return resp, err2
					}
					tempResp.ExecutedAmount = tempExecAmount
					tempResp.RemainingAmount = tempAmount - tempExecAmount
					tempResp.Fee = tempData2.Data.OrderInfo.Fee
					if getOrdersRequest.OrderSide == exchange.AnyOrderSide {
						resp = append(resp, tempResp)
						continue
					}
					if strings.EqualFold(getOrdersRequest.OrderSide.ToString(), tempResp.OrderSide.ToString()) {
						resp = append(resp, tempResp)
					}
				}
			}
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (o *Oex) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (o *Oex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (o *Oex) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	o.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (o *Oex) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrNotYetImplemented
}

// GetSubscriptions returns a copied list of subscriptions
func (o *Oex) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrNotYetImplemented
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (o *Oex) AuthenticateWebsocket() error {
	return common.ErrNotYetImplemented
}

// getAllOpenOrderID gets all the orderIDs for all currency pairs and stores them in map[string]string
func (o *Oex) getAllOpenOrderID() (map[string][]int64, error) {
	resp := make(map[string][]int64)
	var tempData2 OpenOrderResponse
	tempData, err := o.GetAllPairs()
	if err != nil {
		return resp, err
	}
	for x := range tempData.Data {
		tempData2, err = o.GetOpenOrders(tempData.Data[x].Symbol, "", "1")
		if err != nil {
			return resp, err
		}
		for y := int64(1); len(tempData2.Data.ResultList) != 0; y++ {
			tempData2, err = o.GetOpenOrders(tempData.Data[x].Symbol, "", strconv.FormatInt(y, 10))
			if err != nil {
				return resp, err
			}
			for z := 0; z < len(tempData2.Data.ResultList); z++ {
				resp[tempData.Data[x].Symbol] = append(resp[tempData.Data[x].Symbol], tempData2.Data.ResultList[z].ID)
			}
		}
	}
	return resp, nil
}
