package localbitcoins

import (
	"errors"
	"fmt"
	"math"
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

// Start starts the LocalBitcoins go routine
func (l *LocalBitcoins) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the LocalBitcoins wrapper
func (l *LocalBitcoins) Run() {
	if l.Verbose {
		log.Debugf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Debugf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	currencies, err := l.GetTradableCurrencies()
	if err != nil {
		log.Errorf("%s failed to obtain available tradable currencies. Err: %s", l.Name, err)
		return
	}

	var pairs []string
	for x := range currencies {
		pairs = append(pairs, "BTC"+currencies[x])
	}

	var newExchangeProducts currency.Pairs
	for _, p := range pairs {
		newExchangeProducts = append(newExchangeProducts,
			currency.NewPairFromString(p))
	}

	err = l.UpdateCurrencies(newExchangeProducts, false, false)
	if err != nil {
		log.Errorf("%s failed to update available currencies. Err %s", l.Name, err)
	}

}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LocalBitcoins) UpdateTicker(p currency.Pair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := l.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range l.GetEnabledCurrencies() {
		currency := x.Quote.String()
		var tp ticker.Price
		tp.Pair = x
		tp.Last = tick[currency].Avg24h
		tp.Volume = tick[currency].VolumeBTC

		err = ticker.ProcessTicker(l.GetName(), &tp, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}

	return ticker.GetTicker(l.GetName(), p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *LocalBitcoins) GetTickerPrice(p currency.Pair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *LocalBitcoins) GetOrderbookEx(p currency.Pair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.Get(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LocalBitcoins) UpdateOrderbook(p currency.Pair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := l.GetOrderbook(p.Quote.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data.Amount / data.Price, Price: data.Price})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data.Amount / data.Price, Price: data.Price})
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
// LocalBitcoins exchange
func (l *LocalBitcoins) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = l.GetName()
	accountBalance, err := l.GetWalletBalance()
	if err != nil {
		return response, err
	}
	var exchangeCurrency exchange.AccountCurrencyInfo
	exchangeCurrency.CurrencyName = currency.BTC
	exchangeCurrency.TotalValue = accountBalance.Total.Balance

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: []exchange.AccountCurrencyInfo{exchangeCurrency},
	})
	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LocalBitcoins) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LocalBitcoins) GetExchangeHistory(p currency.Pair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (l *LocalBitcoins) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, _ float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	// These are placeholder details
	// TODO store a user's localbitcoin details to use here
	var params = AdCreate{
		PriceEquation:              "USD_in_AUD",
		Latitude:                   1,
		Longitude:                  1,
		City:                       "City",
		Location:                   "Location",
		CountryCode:                "US",
		Currency:                   p.Quote.String(),
		AccountInfo:                "-",
		BankName:                   "Bank",
		MSG:                        side.ToString(),
		SMSVerficationRequired:     true,
		TrackMaxAmount:             true,
		RequireTrustedByAdvertiser: true,
		RequireIdentification:      true,
		OnlineProvider:             "",
		TradeType:                  "",
		MinAmount:                  int(math.Round(amount)),
	}

	// Does not return any orderID, so create the add, then get the order
	err := l.CreateAd(&params)
	if err != nil {
		return submitOrderResponse, err
	}

	submitOrderResponse.IsOrderPlaced = true

	// Now to figure out what ad we just submitted
	// The only details we have are the params above
	var adID string
	ads, err := l.Getads()
	for i := range ads.AdList {
		if ads.AdList[i].Data.PriceEquation == params.PriceEquation &&
			ads.AdList[i].Data.Lat == float64(params.Latitude) &&
			ads.AdList[i].Data.Lon == float64(params.Longitude) &&
			ads.AdList[i].Data.City == params.City &&
			ads.AdList[i].Data.Location == params.Location &&
			ads.AdList[i].Data.CountryCode == params.CountryCode &&
			ads.AdList[i].Data.Currency == params.Currency &&
			ads.AdList[i].Data.AccountInfo == params.AccountInfo &&
			ads.AdList[i].Data.BankName == params.BankName &&
			ads.AdList[i].Data.SMSVerficationRequired == params.SMSVerficationRequired &&
			ads.AdList[i].Data.TrackMaxAmount == params.TrackMaxAmount &&
			ads.AdList[i].Data.RequireTrustedByAdvertiser == params.RequireTrustedByAdvertiser &&
			ads.AdList[i].Data.OnlineProvider == params.OnlineProvider &&
			ads.AdList[i].Data.TradeType == params.TradeType &&
			ads.AdList[i].Data.MinAmount == fmt.Sprintf("%v", params.MinAmount) {
			adID = fmt.Sprintf("%v", ads.AdList[i].Data.AdID)
		}
	}

	if adID != "" {
		submitOrderResponse.OrderID = adID
	} else {
		return submitOrderResponse, errors.New("ad placed, but not found via API")
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LocalBitcoins) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LocalBitcoins) CancelOrder(order *exchange.OrderCancellation) error {
	return l.DeleteAd(order.OrderID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LocalBitcoins) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	ads, err := l.Getads()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range ads.AdList {
		adIDString := strconv.FormatInt(ads.AdList[i].Data.AdID, 10)
		err = l.DeleteAd(adIDString)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[adIDString] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (l *LocalBitcoins) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LocalBitcoins) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	if !strings.EqualFold(currency.BTC.String(), cryptocurrency.String()) {
		return "", fmt.Errorf("%s does not have support for currency %s, it only supports bitcoin",
			l.Name, cryptocurrency)
	}

	return l.GetWalletAddress()
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LocalBitcoins) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "",
		l.WalletSend(withdrawRequest.Address,
			withdrawRequest.Amount,
			withdrawRequest.PIN)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *LocalBitcoins) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LocalBitcoins) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (l.APIKey == "" || l.APISecret == "") && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return l.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (l *LocalBitcoins) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := l.GetDashboardInfo()
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp {
		orderDate, err := time.Parse(time.RFC3339, resp[i].Data.CreatedAt)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				l.Name,
				"GetActiveOrders",
				resp[i].Data.Advertisement.ID,
				resp[i].Data.CreatedAt)
		}

		var side exchange.OrderSide
		if resp[i].Data.IsBuying {
			side = exchange.BuyOrderSide
		} else if resp[i].Data.IsSelling {
			side = exchange.SellOrderSide
		}

		orders = append(orders, exchange.OrderDetail{
			Amount:    resp[i].Data.AmountBTC,
			Price:     resp[i].Data.Amount,
			ID:        fmt.Sprintf("%v", resp[i].Data.Advertisement.ID),
			OrderDate: orderDate,
			Fee:       resp[i].Data.FeeBTC,
			OrderSide: side,
			CurrencyPair: currency.NewPairWithDelimiter(currency.BTC.String(),
				resp[i].Data.Currency,
				l.ConfigCurrencyPairFormat.Delimiter),
			Exchange: l.Name,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *LocalBitcoins) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var allTrades []DashBoardInfo
	resp, err := l.GetDashboardCancelledTrades()
	if err != nil {
		return nil, err
	}
	allTrades = append(allTrades, resp...)

	resp, err = l.GetDashboardClosedTrades()
	if err != nil {
		return nil, err
	}
	allTrades = append(allTrades, resp...)

	resp, err = l.GetDashboardReleasedTrades()
	if err != nil {
		return nil, err
	}
	allTrades = append(allTrades, resp...)

	var orders []exchange.OrderDetail
	for i := range allTrades {
		orderDate, err := time.Parse(time.RFC3339, allTrades[i].Data.CreatedAt)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				l.Name,
				"GetActiveOrders",
				allTrades[i].Data.Advertisement.ID,
				allTrades[i].Data.CreatedAt)
		}

		var side exchange.OrderSide
		if allTrades[i].Data.IsBuying {
			side = exchange.BuyOrderSide
		} else if allTrades[i].Data.IsSelling {
			side = exchange.SellOrderSide
		}

		status := ""

		switch {
		case allTrades[i].Data.ReleasedAt != "" && allTrades[i].Data.ReleasedAt != null:
			status = "Released"
		case allTrades[i].Data.CanceledAt != "" && allTrades[i].Data.CanceledAt != null:
			status = "Cancelled"
		case allTrades[i].Data.ClosedAt != "" && allTrades[i].Data.ClosedAt != null:
			status = "Closed"
		}

		orders = append(orders, exchange.OrderDetail{
			Amount:    allTrades[i].Data.AmountBTC,
			Price:     allTrades[i].Data.Amount,
			ID:        fmt.Sprintf("%v", allTrades[i].Data.Advertisement.ID),
			OrderDate: orderDate,
			Fee:       allTrades[i].Data.FeeBTC,
			OrderSide: side,
			Status:    status,
			CurrencyPair: currency.NewPairWithDelimiter(currency.BTC.String(),
				allTrades[i].Data.Currency,
				l.ConfigCurrencyPairFormat.Delimiter),
			Exchange: l.Name,
		})
	}

	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (l *LocalBitcoins) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (l *LocalBitcoins) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (l *LocalBitcoins) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (l *LocalBitcoins) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
