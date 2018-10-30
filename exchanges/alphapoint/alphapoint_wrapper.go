package alphapoint

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// GetDefaultConfig returns a default exchange config for Alphapoint
func (a *Alphapoint) GetDefaultConfig() (*config.ExchangeConfig, error) {
	return nil, common.ErrFunctionNotSupported
}

// SetDefaults sets current default settings
func (a *Alphapoint) SetDefaults() {
	a.Name = "Alphapoint"
	a.Enabled = true
	a.Verbose = true
	a.API.Endpoints.URL = alphapointDefaultAPIURL
	a.API.Endpoints.WebsocketURL = alphapointDefaultWebsocketURL
	a.API.CredentialsValidator.RequiresKey = true
	a.API.CredentialsValidator.RequiresSecret = true

	a.CurrencyPairs.AssetTypes = assets.AssetTypes{
		assets.AssetTypeSpot,
	}

	a.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AccountInfo: true,
			},

			WebsocketCapabilities: exchange.ProtocolFeatures{
				TickerFetching: true,
			},
		},
	}

	a.APIWithdrawPermissions = exchange.WithdrawCryptoWith2FA |
		exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.NoFiatWithdrawals

	a.Requester = request.New(a.Name,
		request.NewRateLimit(time.Minute*10, alphapointAuthRate),
		request.NewRateLimit(time.Minute*10, alphapointUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	a.WebsocketInit()
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (a *Alphapoint) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (a *Alphapoint) UpdateTradablePairs(forceUpdate bool) error {
	return common.ErrFunctionNotSupported
}

// GetAccountInfo retrieves balances for all enabled currencies on the
// Alphapoint exchange
func (a *Alphapoint) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = a.GetName()
	account, err := a.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(account.Currencies); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(account.Currencies[i].Name)
		exchangeCurrency.TotalValue = float64(account.Currencies[i].Balance)
		exchangeCurrency.Hold = float64(account.Currencies[i].Hold)

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *Alphapoint) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := a.GetTicker(p.String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = tick.Ask
	tickerPrice.Bid = tick.Bid
	tickerPrice.Low = tick.Low
	tickerPrice.High = tick.High
	tickerPrice.Volume = tick.Volume
	tickerPrice.Last = tick.Last

	err = ticker.ProcessTicker(a.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(a.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (a *Alphapoint) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tick, err := ticker.GetTicker(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *Alphapoint) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := a.GetOrderbook(p.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{Amount: data.Quantity, Price: data.Price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = a.GetName()
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(a.Name, p, assetType)
}

// FetchOrderbook returns the orderbook for a currency pair
func (a *Alphapoint) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(a.GetName(), p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (a *Alphapoint) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	// https://alphapoint.github.io/slate/#generatetreasuryactivityreport
	return fundHistory, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *Alphapoint) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order and returns a true value when
// successfully submitted
func (a *Alphapoint) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	response, err := a.CreateOrder(p.String(),
		side.ToString(),
		orderType.ToString(),
		amount, price)

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
func (a *Alphapoint) ModifyOrder(_ *exchange.ModifyOrder) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (a *Alphapoint) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = a.CancelExistingOrder(orderIDInt, order.AccountID)

	return err
}

// CancelAllOrders cancels all orders for a given account
func (a *Alphapoint) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	return exchange.CancelAllOrdersResponse{}, a.CancelAllExistingOrders(orderCancellation.AccountID)
}

// GetOrderInfo returns information on a current open order
func (a *Alphapoint) GetOrderInfo(orderID string) (float64, error) {
	orders, err := a.GetOrders()
	if err != nil {
		return 0, err
	}

	for x := range orders {
		for y := range orders[x].OpenOrders {
			if strconv.Itoa(orders[x].OpenOrders[y].ServerOrderID) == orderID {
				return orders[x].OpenOrders[y].QtyRemaining, nil
			}
		}
	}
	return 0, errors.New("order not found")
}

// GetDepositAddress returns a deposit address for a specified currency
func (a *Alphapoint) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	addreses, err := a.GetDepositAddresses()
	if err != nil {
		return "", err
	}

	for x := range addreses {
		if addreses[x].Name == cryptocurrency.String() {
			return addreses[x].DepositAddress, nil
		}
	}
	return "", errors.New("associated currency address not found")
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *Alphapoint) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (a *Alphapoint) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *Alphapoint) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (a *Alphapoint) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *Alphapoint) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (a *Alphapoint) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := a.GetOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for x := range resp {
		for _, order := range resp[x].OpenOrders {
			if order.State != 1 {
				continue
			}

			orderDetail := exchange.OrderDetail{
				Amount:          order.QtyTotal,
				Exchange:        a.Name,
				AccountID:       fmt.Sprintf("%v", order.AccountID),
				ID:              fmt.Sprintf("%v", order.ServerOrderID),
				Price:           order.Price,
				RemainingAmount: order.QtyRemaining,
			}

			orderDetail.OrderSide = orderSideMap[order.Side]
			orderDetail.OrderDate = time.Unix(order.ReceiveTime, 0)
			orderDetail.OrderType = orderTypeMap[order.OrderType]
			if orderDetail.OrderType == "" {
				orderDetail.OrderType = exchange.UnknownOrderType
			}

			orders = append(orders, orderDetail)
		}
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (a *Alphapoint) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := a.GetOrders()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for x := range resp {
		for _, order := range resp[x].OpenOrders {
			if order.State == 1 {
				continue
			}

			orderDetail := exchange.OrderDetail{
				Amount:          order.QtyTotal,
				AccountID:       fmt.Sprintf("%v", order.AccountID),
				Exchange:        a.Name,
				ID:              fmt.Sprintf("%v", order.ServerOrderID),
				Price:           order.Price,
				RemainingAmount: order.QtyRemaining,
			}

			orderDetail.OrderSide = orderSideMap[order.Side]
			orderDetail.OrderDate = time.Unix(order.ReceiveTime, 0)
			orderDetail.OrderType = orderTypeMap[order.OrderType]
			if orderDetail.OrderType == "" {
				orderDetail.OrderType = exchange.UnknownOrderType
			}

			orders = append(orders, orderDetail)
		}
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	return orders, nil
}
