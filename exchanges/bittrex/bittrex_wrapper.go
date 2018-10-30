package bittrex

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (b *Bittrex) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults method assignes the default values for Bittrex
func (b *Bittrex) SetDefaults() {
	b.Name = "Bittrex"
	b.Enabled = true
	b.Verbose = true
	b.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.NoFiatWithdrawals
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, bittrexAuthRate),
		request.NewRateLimit(time.Second, bittrexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = bittrexAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
}

// Setup method sets current configuration details if enabled
func (b *Bittrex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	return b.SetupDefaults(exch)
}

// Start starts the Bittrex go routine
func (b *Bittrex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bittrex wrapper
func (b *Bittrex) Run() {
	if b.Verbose {
		b.PrintEnabledPairs()
	}

	forceUpdate := false
	if !common.StringDataContains(b.GetEnabledPairs(assets.AssetTypeSpot).Strings(), "-") ||
		!common.StringDataContains(b.GetAvailablePairs(assets.AssetTypeSpot).Strings(), "-") {
		forceUpdate = true
		enabledPairs := []string{"USDT-BTC"}
		log.Warn("WARNING: Available pairs for Bittrex reset due to config upgrade, please enable the ones you would like again")

		err := b.UpdatePairs(currency.NewPairsFromStrings(enabledPairs), assets.AssetTypeSpot, true, true)
		if err != nil {
			log.Errorf("%s failed to update currencies. Err: %s\n", b.Name, err)
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := b.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bittrex) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	markets, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range markets.Result {
		if !markets.Result[x].IsActive || markets.Result[x].MarketName == "" {
			continue
		}
		pairs = append(pairs, markets.Result[x].MarketName)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bittrex) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// GetAccountInfo Retrieves balances for all enabled currencies for the
// Bittrex exchange
func (b *Bittrex) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.GetName()
	accountBalance, err := b.GetAccountBalances()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance.Result); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance.Result[i].Currency)
		exchangeCurrency.TotalValue = accountBalance.Result[i].Balance
		exchangeCurrency.Hold = accountBalance.Result[i].Balance - accountBalance.Result[i].Available
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bittrex) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	tick, err := b.GetMarketSummaries()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledPairs(assetType) {
		curr := b.FormatExchangeCurrency(x, assetType)
		for y := range tick.Result {
			if tick.Result[y].MarketName != curr.String() {
				continue
			}
			tickerPrice.Pair = x
			tickerPrice.High = tick.Result[y].High
			tickerPrice.Low = tick.Result[y].Low
			tickerPrice.Ask = tick.Result[y].Ask
			tickerPrice.Bid = tick.Result[y].Bid
			tickerPrice.Last = tick.Result[y].Last
			tickerPrice.Volume = tick.Result[y].Volume
			ticker.ProcessTicker(b.GetName(), &tickerPrice, assetType)
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bittrex) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bittrex) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bittrex) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := b.GetOrderbook(b.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Result.Buy {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Amount: orderbookNew.Result.Buy[x].Quantity,
				Price:  orderbookNew.Result.Buy[x].Rate,
			},
		)
	}

	for x := range orderbookNew.Result.Sell {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Amount: orderbookNew.Result.Sell[x].Quantity,
				Price:  orderbookNew.Result.Sell[x].Rate,
			},
		)
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

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bittrex) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bittrex) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bittrex) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	buy := side == exchange.BuyOrderSide
	var response UUID
	var err error

	if orderType != exchange.LimitOrderType {
		return submitOrderResponse, errors.New("not supported on exchange")
	}

	if buy {
		response, err = b.PlaceBuyLimit(p.String(), amount, price)
	} else {
		response, err = b.PlaceSellLimit(p.String(), amount, price)
	}

	if response.Result.ID != "" {
		submitOrderResponse.OrderID = response.Result.ID
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bittrex) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bittrex) CancelOrder(order *exchange.OrderCancellation) error {
	_, err := b.CancelExistingOrder(order.OrderID)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bittrex) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	openOrders, err := b.GetOpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range openOrders.Result {
		_, err := b.CancelExistingOrder(openOrders.Result[i].OrderUUID)
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[openOrders.Result[i].OrderUUID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Bittrex) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bittrex) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	depositAddr, err := b.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	return depositAddr.Result.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bittrex) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	uuid, err := b.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.AddressTag, withdrawRequest.Address, withdrawRequest.Amount)
	return fmt.Sprintf("%v", uuid), err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bittrex) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bittrex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)

}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bittrex) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var currPair string
	if len(getOrdersRequest.Currencies) == 1 {
		currPair = getOrdersRequest.Currencies[0].String()
	}

	resp, err := b.GetOpenOrders(currPair)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp.Result {
		orderDate, err := time.Parse(time.RFC3339, resp.Result[i].Opened)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				b.Name, "GetActiveOrders", resp.Result[i].OrderUUID, resp.Result[i].Opened)
		}

		pair := currency.NewPairDelimiter(resp.Result[i].Exchange,
			b.CurrencyPairs.ConfigFormat.Delimiter)
		orderType := exchange.OrderType(strings.ToUpper(resp.Result[i].Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:          resp.Result[i].Quantity,
			RemainingAmount: resp.Result[i].QuantityRemaining,
			Price:           resp.Result[i].Price,
			OrderDate:       orderDate,
			ID:              resp.Result[i].OrderUUID,
			Exchange:        b.Name,
			OrderType:       orderType,
			CurrencyPair:    pair,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bittrex) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var currPair string
	if len(getOrdersRequest.Currencies) == 1 {
		currPair = getOrdersRequest.Currencies[0].String()
	}

	resp, err := b.GetOrderHistoryForCurrency(currPair)
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range resp.Result {
		orderDate, err := time.Parse(time.RFC3339, resp.Result[i].TimeStamp)
		if err != nil {
			log.Warnf("Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				b.Name, "GetActiveOrders", resp.Result[i].OrderUUID, resp.Result[i].Opened)
		}

		pair := currency.NewPairDelimiter(resp.Result[i].Exchange,
			b.CurrencyPairs.ConfigFormat.Delimiter)
		orderType := exchange.OrderType(strings.ToUpper(resp.Result[i].Type))

		orders = append(orders, exchange.OrderDetail{
			Amount:          resp.Result[i].Quantity,
			RemainingAmount: resp.Result[i].QuantityRemaining,
			Price:           resp.Result[i].Price,
			OrderDate:       orderDate,
			ID:              resp.Result[i].OrderUUID,
			Exchange:        b.Name,
			OrderType:       orderType,
			Fee:             resp.Result[i].Commission,
			CurrencyPair:    pair,
		})
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}
