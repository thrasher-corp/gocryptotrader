package okgroup

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Note: GoCryptoTrader wrapper funcs currently only support SPOT trades.
// Therefore this OKGroup_Wrapper can be shared between OKEX and OKCoin.
// When circumstances change, wrapper funcs can be split appropriately

// Setup sets user exchange configuration settings
func (o *OKGroup) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		o.SetEnabled(false)
		return nil
	}

	err := o.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return o.WebsocketSetup(o.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		o.WebsocketURL,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the OKGroup go routine
func (o *OKGroup) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKGroup) Run() {
	if o.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.WebsocketURL)
	}

	prods, err := o.GetSpotTokenPairDetails()
	if err != nil {
		log.Errorf("%v failed to obtain available spot instruments. Err: %s", o.Name, err)
		return
	}

	var pairs currency.Pairs
	for x := range prods {
		pairs = append(pairs, currency.NewPairFromString(prods[x].BaseCurrency+"_"+prods[x].QuoteCurrency))
	}

	err = o.UpdatePairs(pairs, assets.AssetTypeSpot, false, false)
	if err != nil {
		log.Errorf("%v failed to update available currencies. Err: %s", o.Name, err)
		return
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKGroup) UpdateTicker(p currency.Pair, assetType assets.AssetType) (tickerData ticker.Price, err error) {
	resp, err := o.GetSpotAllTokenPairsInformationForCurrency(o.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return
	}
	tickerData = ticker.Price{
		Ask:         resp.BestAsk,
		Bid:         resp.BestBid,
		High:        resp.High24h,
		Last:        resp.Last,
		LastUpdated: resp.Timestamp,
		Low:         resp.Low24h,
		Pair:        o.FormatExchangeCurrency(p, assetType),
		Volume:      resp.BaseVolume24h,
	}

	err = ticker.ProcessTicker(o.Name, &tickerData, assetType)
	return
}

// FetchTicker returns the ticker for a currency pair
func (o *OKGroup) FetchTicker(p currency.Pair, assetType assets.AssetType) (tickerData ticker.Price, err error) {
	tickerData, err = ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return
}

// FetchOrderbook returns orderbook base on the currency pair
func (o *OKGroup) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (resp orderbook.Base, err error) {
	ob, err := orderbook.Get(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKGroup) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (resp orderbook.Base, err error) {
	orderbookNew, err := o.GetSpotOrderBook(GetSpotOrderBookRequest{
		InstrumentID: o.FormatExchangeCurrency(p, assetType).String(),
	})
	if err != nil {
		return
	}

	for x := range orderbookNew.Bids {
		amount, convErr := strconv.ParseFloat(orderbookNew.Bids[x][1], 64)
		if convErr != nil {
			log.Errorf("Could not convert %v to float64", orderbookNew.Bids[x][1])
		}
		price, convErr := strconv.ParseFloat(orderbookNew.Bids[x][0], 64)
		if convErr != nil {
			log.Errorf("Could not convert %v to float64", orderbookNew.Bids[x][0])
		}
		resp.Bids = append(resp.Bids, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
	}

	for x := range orderbookNew.Asks {
		amount, convErr := strconv.ParseFloat(orderbookNew.Asks[x][1], 64)
		if convErr != nil {
			log.Errorf("Could not convert %v to float64", orderbookNew.Asks[x][1])
		}
		price, convErr := strconv.ParseFloat(orderbookNew.Asks[x][0], 64)
		if convErr != nil {
			log.Errorf("Could not convert %v to float64", orderbookNew.Asks[x][0])
		}
		resp.Asks = append(resp.Asks, orderbook.Item{
			Amount: amount,
			Price:  price,
		})
	}

	resp.Pair = p
	resp.AssetType = assetType
	resp.ExchangeName = o.Name

	err = resp.Process()
	if err != nil {
		return
	}

	return orderbook.Get(o.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies
func (o *OKGroup) GetAccountInfo() (resp exchange.AccountInfo, err error) {
	resp.Exchange = o.Name
	currencies, err := o.GetSpotTradingAccounts()
	currencyAccount := exchange.Account{}

	for _, curr := range currencies {
		hold, err := strconv.ParseFloat(curr.Hold, 64)
		if err != nil {
			log.Errorf("Could not convert %v to float64", curr.Hold)
		}
		totalValue, err := strconv.ParseFloat(curr.Balance, 64)
		if err != nil {
			log.Errorf("Could not convert %v to float64", curr.Balance)
		}
		currencyAccount.Currencies = append(currencyAccount.Currencies, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(curr.Currency),
			Hold:         hold,
			TotalValue:   totalValue,
		})
	}

	resp.Accounts = append(resp.Accounts, currencyAccount)
	return
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *OKGroup) GetFundingHistory() (resp []exchange.FundHistory, err error) {
	accountDepositHistory, err := o.GetAccountDepositHistory("")
	if err != nil {
		return
	}
	for _, deposit := range accountDepositHistory {
		orderStatus := ""
		switch deposit.Status {
		case 0:
			orderStatus = "waiting"
		case 1:
			orderStatus = "confirmation account"
		case 2:
			orderStatus = "recharge success"
		}

		resp = append(resp, exchange.FundHistory{
			Amount:       deposit.Amount,
			Currency:     deposit.Currency,
			ExchangeName: o.Name,
			Status:       orderStatus,
			Timestamp:    deposit.Timestamp,
			TransferID:   deposit.TransactionID,
			TransferType: "deposit",
		})
	}
	accountWithdrawlHistory, err := o.GetAccountWithdrawalHistory("")
	for i := range accountWithdrawlHistory {
		resp = append(resp, exchange.FundHistory{
			Amount:       accountWithdrawlHistory[i].Amount,
			Currency:     accountWithdrawlHistory[i].Currency,
			ExchangeName: o.Name,
			Status:       OrderStatus[accountWithdrawlHistory[i].Status],
			Timestamp:    accountWithdrawlHistory[i].Timestamp,
			TransferID:   accountWithdrawlHistory[i].Txid,
			TransferType: "withdrawal",
		})
	}
	return resp, err
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *OKGroup) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (o *OKGroup) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (resp exchange.SubmitOrderResponse, err error) {
	request := PlaceSpotOrderRequest{
		ClientOID:    clientID,
		InstrumentID: o.FormatExchangeCurrency(p, assets.AssetTypeSpot).String(),
		Side:         strings.ToLower(side.ToString()),
		Type:         strings.ToLower(orderType.ToString()),
		Size:         strconv.FormatFloat(amount, 'f', -1, 64),
	}
	if orderType == exchange.LimitOrderType {
		request.Price = strconv.FormatFloat(price, 'f', -1, 64)
	}

	orderResponse, err := o.PlaceSpotOrder(&request)
	if err != nil {
		return
	}
	resp.IsOrderPlaced = orderResponse.Result
	resp.OrderID = orderResponse.OrderID

	return
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *OKGroup) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (o *OKGroup) CancelOrder(orderCancellation *exchange.OrderCancellation) (err error) {
	orderID, err := strconv.ParseInt(orderCancellation.OrderID, 10, 64)
	if err != nil {
		return
	}
	orderCancellationResponse, err := o.CancelSpotOrder(CancelSpotOrderRequest{
		InstrumentID: o.FormatExchangeCurrency(orderCancellation.CurrencyPair,
			assets.AssetTypeSpot).String(),
		OrderID: orderID,
	})
	if !orderCancellationResponse.Result {
		err = fmt.Errorf("order %v failed to be cancelled", orderCancellationResponse.OrderID)
	}

	return
}

// CancelAllOrders cancels all orders associated with a currency pair
func (o *OKGroup) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (resp exchange.CancelAllOrdersResponse, _ error) {
	orderIDs := strings.Split(orderCancellation.OrderID, ",")
	var orderIDNumbers []int64
	for _, i := range orderIDs {
		orderIDNumber, err := strconv.ParseInt(i, 10, 64)
		if err != nil {
			return resp, err
		}
		orderIDNumbers = append(orderIDNumbers, orderIDNumber)
	}

	cancelOrdersResponse, err := o.CancelMultipleSpotOrders(CancelMultipleSpotOrdersRequest{
		InstrumentID: o.FormatExchangeCurrency(orderCancellation.CurrencyPair,
			assets.AssetTypeSpot).String(),
		OrderIDs: orderIDNumbers,
	})
	if err != nil {
		return
	}

	for _, orderMap := range cancelOrdersResponse {
		for _, cancelledOrder := range orderMap {
			resp.OrderStatus[fmt.Sprintf("%v", cancelledOrder.OrderID)] = fmt.Sprintf("%v", cancelledOrder.Result)
		}
	}

	return
}

// GetOrderInfo returns information on a current open order
func (o *OKGroup) GetOrderInfo(orderID string) (resp exchange.OrderDetail, err error) {
	order, err := o.GetSpotOrder(GetSpotOrderRequest{OrderID: orderID})
	if err != nil {
		return
	}
	resp = exchange.OrderDetail{
		Amount: order.Size,
		CurrencyPair: currency.NewPairDelimiter(order.InstrumentID,
			o.CurrencyPairs.ConfigFormat.Delimiter),
		Exchange:       o.Name,
		OrderDate:      order.Timestamp,
		ExecutedAmount: order.FilledSize,
		Status:         order.Status,
		OrderSide:      exchange.OrderSide(order.Side),
	}
	return
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *OKGroup) GetDepositAddress(p currency.Code, accountID string) (_ string, err error) {
	wallet, err := o.GetAccountDepositAddressForCurrency(p.Lower().String())
	if err != nil {
		return
	}
	return wallet[0].Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *OKGroup) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	withdrawal, err := o.AccountWithdraw(AccountWithdrawRequest{
		Amount:      withdrawRequest.Amount,
		Currency:    withdrawRequest.Currency.Lower().String(),
		Destination: 4, // 1, 2, 3 are all internal
		Fee:         withdrawRequest.FeeAmount,
		ToAddress:   withdrawRequest.Address,
		TradePwd:    withdrawRequest.TradePassword,
	})
	if err != nil {
		return "", err
	}
	if !withdrawal.Result {
		return fmt.Sprintf("%v", withdrawal.WithdrawalID), fmt.Errorf("could not withdraw currency %v to %v, no error specified", withdrawRequest.Currency.String(), withdrawRequest.Address)
	}

	return fmt.Sprintf("%v", withdrawal.WithdrawalID), nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKGroup) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (o *OKGroup) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) (resp []exchange.OrderDetail, err error) {
	for _, currency := range getOrdersRequest.Currencies {
		spotOpenOrders, err := o.GetSpotOpenOrders(GetSpotOpenOrdersRequest{
			InstrumentID: o.FormatExchangeCurrency(currency,
				assets.AssetTypeSpot).String(),
		})
		if err != nil {
			return resp, err
		}
		for i := range spotOpenOrders {
			resp = append(resp, exchange.OrderDetail{
				ID:             spotOpenOrders[i].OrderID,
				Price:          spotOpenOrders[i].Price,
				Amount:         spotOpenOrders[i].Size,
				CurrencyPair:   currency,
				Exchange:       o.Name,
				OrderSide:      exchange.OrderSide(spotOpenOrders[i].Side),
				OrderType:      exchange.OrderType(spotOpenOrders[i].Type),
				ExecutedAmount: spotOpenOrders[i].FilledSize,
				OrderDate:      spotOpenOrders[i].Timestamp,
				Status:         spotOpenOrders[i].Status,
			})
		}
	}

	return
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (o *OKGroup) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) (resp []exchange.OrderDetail, err error) {
	for _, currency := range getOrdersRequest.Currencies {
		spotOpenOrders, err := o.GetSpotOrders(GetSpotOrdersRequest{
			Status: strings.Join([]string{"filled", "cancelled", "failure"}, "|"),
			InstrumentID: o.FormatExchangeCurrency(currency,
				assets.AssetTypeSpot).String(),
		})
		if err != nil {
			return resp, err
		}
		for i := range spotOpenOrders {
			resp = append(resp, exchange.OrderDetail{
				ID:             spotOpenOrders[i].OrderID,
				Price:          spotOpenOrders[i].Price,
				Amount:         spotOpenOrders[i].Size,
				CurrencyPair:   currency,
				Exchange:       o.Name,
				OrderSide:      exchange.OrderSide(spotOpenOrders[i].Side),
				OrderType:      exchange.OrderType(spotOpenOrders[i].Type),
				ExecutedAmount: spotOpenOrders[i].FilledSize,
				OrderDate:      spotOpenOrders[i].Timestamp,
				Status:         spotOpenOrders[i].Status,
			})
		}
	}

	return
}

// GetWebsocket returns a pointer to the exchange websocket
func (o *OKGroup) GetWebsocket() (*exchange.Websocket, error) {
	return o.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *OKGroup) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !o.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return o.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (o *OKGroup) GetWithdrawCapabilities() uint32 {
	return o.GetWithdrawPermissions()
}
