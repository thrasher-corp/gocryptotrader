package bithumb

import (
	"errors"
	"fmt"
	"math"
	"strconv"
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
func (b *Bithumb) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets the basic defaults for Bithumb
func (b *Bithumb) SetDefaults() {
	b.Name = "Bithumb"
	b.Enabled = true
	b.Verbose = true
	b.APIWithdrawPermissions = exchange.AutoWithdrawCrypto |
		exchange.AutoWithdrawFiat
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},

		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Index:     "KRW",
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST: true,
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
		request.NewRateLimit(time.Second, bithumbAuthRate),
		request.NewRateLimit(time.Second, bithumbUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = apiURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bithumb) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	return b.SetupDefaults(exch)
}

// Start starts the Bithumb go routine
func (b *Bithumb) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bithumb wrapper
func (b *Bithumb) Run() {
	if b.Verbose {
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bithumb) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	currencies, err := b.GetTradablePairs()
	if err != nil {
		return nil, err
	}

	for x := range currencies {
		currencies[x] += "KRW"
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bithumb) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bithumb) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price

	tickers, err := b.GetAllTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range b.GetEnabledPairs(assetType) {
		currency := x.Base.String()
		var tp ticker.Price
		tp.Pair = x
		tp.Ask = tickers[currency].SellPrice
		tp.Bid = tickers[currency].BuyPrice
		tp.Low = tickers[currency].MinPrice
		tp.Last = tickers[currency].ClosingPrice
		tp.Volume = tickers[currency].Volume1Day
		tp.High = tickers[currency].MaxPrice

		err = ticker.ProcessTicker(b.Name, &tp, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bithumb) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Bithumb) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bithumb) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := p.Base.String()

	orderbookNew, err := b.GetOrderBook(currency)
	if err != nil {
		return orderBook, err
	}

	for _, bids := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: bids.Quantity, Price: bids.Price})
	}

	for _, asks := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: asks.Quantity, Price: asks.Price})
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
// Bithumb exchange
func (b *Bithumb) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	bal, err := b.GetAccountBalance("ALL")
	if err != nil {
		return info, err
	}

	var exchangeBalances []exchange.AccountCurrencyInfo
	for key, totalAmount := range bal.Total {
		hold, ok := bal.InUse[key]
		if !ok {
			return info, fmt.Errorf("getAccountInfo error - in use item not found for currency %s",
				key)
		}

		exchangeBalances = append(exchangeBalances, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(key),
			TotalValue:   totalAmount,
			Hold:         hold,
		})
	}

	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: exchangeBalances,
	})

	info.Exchange = b.GetName()
	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bithumb) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bithumb) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
// TODO: Fill this out to support limit orders
func (b *Bithumb) SubmitOrder(p currency.Pair, side exchange.OrderSide, _ exchange.OrderType, amount, _ float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var err error
	var orderID string
	if side == exchange.BuyOrderSide {
		var result MarketBuy
		result, err = b.MarketBuyOrder(p.Base.String(), amount)
		orderID = result.OrderID
	} else if side == exchange.SellOrderSide {
		var result MarketSell
		result, err = b.MarketSellOrder(p.Base.String(), amount)
		orderID = result.OrderID
	}

	if orderID != "" {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", orderID)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bithumb) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	order, err := b.ModifyTrade(action.OrderID,
		action.CurrencyPair.Base.String(),
		common.StringToLower(action.OrderSide.ToString()),
		action.Amount,
		int64(action.Price))

	if err != nil {
		return "", err
	}

	return order.Data[0].ContID, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bithumb) CancelOrder(order *exchange.OrderCancellation) error {
	_, err := b.CancelTrade(order.Side.ToString(),
		order.OrderID,
		order.CurrencyPair.Base.String())
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bithumb) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var allOrders []OrderData

	for _, currency := range b.GetEnabledPairs(assets.AssetTypeSpot) {
		orders, err := b.GetOrders("",
			orderCancellation.Side.ToString(),
			"100",
			"",
			currency.Base.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		allOrders = append(allOrders, orders.Data...)
	}

	for i := range allOrders {
		_, err := b.CancelTrade(orderCancellation.Side.ToString(),
			allOrders[i].OrderID,
			orderCancellation.CurrencyPair.Base.String())
		if err != nil {
			cancelAllOrdersResponse.OrderStatus[allOrders[i].OrderID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Bithumb) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bithumb) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	addr, err := b.GetWalletAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	return addr.Data.WalletAddress, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bithumb) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	_, err := b.WithdrawCrypto(withdrawRequest.Address, withdrawRequest.AddressTag, withdrawRequest.Currency.String(), withdrawRequest.Amount)
	return "", err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bithumb) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	if math.Mod(withdrawRequest.Amount, 1) != 0 {
		return "", errors.New("currency KRW does not support decimal places")
	}
	if withdrawRequest.Currency != currency.KRW {
		return "", errors.New("only KRW is supported")
	}
	bankDetails := fmt.Sprintf("%v_%v", withdrawRequest.BankCode, withdrawRequest.BankName)
	bankAccountNumber := strconv.FormatFloat(withdrawRequest.BankAccountNumber, 'f', -1, 64)
	withdrawAmountInt := int64(withdrawRequest.Amount)
	resp, err := b.RequestKRWWithdraw(bankDetails, bankAccountNumber, withdrawAmountInt)
	if err != nil {
		return "", err
	}
	if resp.Status != "0000" {
		return "", errors.New(resp.Message)
	}

	return resp.Message, nil
}

// WithdrawFiatFundsToInternationalBank is not supported as Bithumb only withdraws KRW to South Korean banks
func (b *Bithumb) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bithumb) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bithumb) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bithumb) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	resp, err := b.GetOrders("", "", "1000", "", "")
	if err != nil {
		return nil, err
	}

	for i := range resp.Data {
		if resp.Data[i].Status != "placed" {
			continue
		}

		orderDate := time.Unix(resp.Data[i].OrderDate, 0)
		orderDetail := exchange.OrderDetail{
			Amount:          resp.Data[i].Units,
			Exchange:        b.Name,
			ID:              resp.Data[i].OrderID,
			OrderDate:       orderDate,
			Price:           resp.Data[i].Price,
			RemainingAmount: resp.Data[i].UnitsRemaining,
			Status:          string(exchange.ActiveOrderStatus),
			CurrencyPair: currency.NewPairWithDelimiter(resp.Data[i].OrderCurrency,
				resp.Data[i].PaymentCurrency,
				b.CurrencyPairs.ConfigFormat.Delimiter),
		}

		if resp.Data[i].Type == "bid" {
			orderDetail.OrderSide = exchange.BuyOrderSide
		} else if resp.Data[i].Type == "ask" {
			orderDetail.OrderSide = exchange.SellOrderSide
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bithumb) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	resp, err := b.GetOrders("", "", "1000", "", "")
	if err != nil {
		return nil, err
	}

	for i := range resp.Data {
		if resp.Data[i].Status == "placed" {
			continue
		}

		orderDate := time.Unix(resp.Data[i].OrderDate, 0)
		orderDetail := exchange.OrderDetail{
			Amount:          resp.Data[i].Units,
			Exchange:        b.Name,
			ID:              resp.Data[i].OrderID,
			OrderDate:       orderDate,
			Price:           resp.Data[i].Price,
			RemainingAmount: resp.Data[i].UnitsRemaining,
			CurrencyPair: currency.NewPairWithDelimiter(resp.Data[i].OrderCurrency,
				resp.Data[i].PaymentCurrency,
				b.CurrencyPairs.ConfigFormat.Delimiter),
		}

		if resp.Data[i].Type == "bid" {
			orderDetail.OrderSide = exchange.BuyOrderSide
		} else if resp.Data[i].Type == "ask" {
			orderDetail.OrderSide = exchange.SellOrderSide
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}
