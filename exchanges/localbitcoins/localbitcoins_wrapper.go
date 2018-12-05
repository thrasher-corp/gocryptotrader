package localbitcoins

import (
	"errors"
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
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
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	currencies, err := l.GetTradableCurrencies()
	if err != nil {
		log.Printf("%s failed to obtain available tradable currencies. Err: %s", l.Name, err)
		return
	}

	var pairs []string
	for x := range currencies {
		pairs = append(pairs, "BTC"+currencies[x])
	}

	err = l.UpdateCurrencies(pairs, false, false)
	if err != nil {
		log.Printf("%s failed to update available currencies. Err %s", l.Name, err)
	}

}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LocalBitcoins) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := l.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range l.GetEnabledCurrencies() {
		currency := x.SecondCurrency.String()
		var tp ticker.Price
		tp.Pair = x
		tp.Last = tick[currency].Avg24h
		tp.Volume = tick[currency].VolumeBTC
		ticker.ProcessTicker(l.GetName(), x, tp, assetType)
	}

	return ticker.GetTicker(l.GetName(), p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (l *LocalBitcoins) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (l *LocalBitcoins) GetOrderbookEx(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(l.GetName(), p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LocalBitcoins) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := l.GetOrderbook(p.SecondCurrency.String())
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

	orderbook.ProcessOrderbook(l.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(l.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// LocalBitcoins exchange
func (l *LocalBitcoins) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = l.GetName()
	accountBalance, err := l.GetWalletBalance()
	if err != nil {
		return response, err
	}
	var exchangeCurrency exchange.AccountCurrencyInfo
	exchangeCurrency.CurrencyName = "BTC"
	exchangeCurrency.TotalValue = accountBalance.Total.Balance

	response.Currencies = append(response.Currencies, exchangeCurrency)
	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LocalBitcoins) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LocalBitcoins) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (l *LocalBitcoins) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
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
		Currency:                   p.SecondCurrency.String(),
		AccountInfo:                "-",
		BankName:                   "Bank",
		MSG:                        fmt.Sprintf("%s", side.ToString()),
		SMSVerficationRequired:     true,
		TrackMaxAmount:             true,
		RequireTrustedByAdvertiser: true,
		RequireIdentification:      true,
		OnlineProvider:             "",
		TradeType:                  "",
		MinAmount:                  int(math.Round(amount)),
	}

	// Does not return any orderID, so create the add, then get the order
	err := l.CreateAd(params)
	if err != nil {
		return submitOrderResponse, err
	}
	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	// Now to figure out what ad we just submitted
	// The only details we have are the params above
	var adID string
	ads, err := l.Getads()
	for _, i := range ads.AdList {
		if i.Data.PriceEquation == params.PriceEquation &&
			i.Data.Lat == float64(params.Latitude) &&
			i.Data.Lon == float64(params.Longitude) &&
			i.Data.City == params.City &&
			i.Data.Location == params.Location &&
			i.Data.CountryCode == params.CountryCode &&
			i.Data.Currency == params.Currency &&
			i.Data.AccountInfo == params.AccountInfo &&
			i.Data.BankName == params.BankName &&
			i.Data.SMSVerficationRequired == params.SMSVerficationRequired &&
			i.Data.TrackMaxAmount == params.TrackMaxAmount &&
			i.Data.RequireTrustedByAdvertiser == params.RequireTrustedByAdvertiser &&
			i.Data.OnlineProvider == params.OnlineProvider &&
			i.Data.TradeType == params.TradeType &&
			i.Data.MinAmount == fmt.Sprintf("%v", params.MinAmount) {
			adID = fmt.Sprintf("%v", i.Data.AdID)
		}
	}

	if adID != "" {
		submitOrderResponse.OrderID = adID
	} else {
		return submitOrderResponse, errors.New("Ad placed, but not found via API")
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LocalBitcoins) ModifyOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LocalBitcoins) CancelOrder(order exchange.OrderCancellation) error {
	return l.DeleteAd(order.OrderID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LocalBitcoins) CancelAllOrders(orderCancellation exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (l *LocalBitcoins) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LocalBitcoins) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LocalBitcoins) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *LocalBitcoins) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LocalBitcoins) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return l.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (l *LocalBitcoins) GetWithdrawCapabilities() uint32 {
	return l.GetWithdrawPermissions()
}
