package bitfinex

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
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
func (b *Bitfinex) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets the basic defaults for bitfinex
func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = true
	b.Verbose = true
	b.WebsocketSubdChannels = make(map[int]WebsocketChanInfo)
	b.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.AutoWithdrawFiatWithAPIPermission
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
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
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
		request.NewRateLimit(time.Second*60, bitfinexAuthRate),
		request.NewRateLimit(time.Second*60, bitfinexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = bitfinexAPIURLBase
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.WebsocketInit()
	b.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketTradeDataSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitfinex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return b.WebsocketSetup(b.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		bitfinexWebsocket,
		exch.API.Endpoints.WebsocketURL)
}

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
func (b *Bitfinex) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	return b.GetSymbols()
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitfinex) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitfinex) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	enabledPairs := b.GetEnabledPairs(assetType)

	var pairs []string
	for x := range enabledPairs {
		pairs = append(pairs, "t"+enabledPairs[x].String())
	}

	tickerNew, err := b.GetTickersV2(common.JoinStrings(pairs, ","))
	if err != nil {
		return tickerPrice, err
	}

	for x := range tickerNew {
		newP := currency.NewPairFromStrings(tickerNew[x].Symbol[1:4],
			tickerNew[x].Symbol[4:])

		var tick ticker.Price
		tick.Pair = newP
		tick.Ask = tickerNew[x].Ask
		tick.Bid = tickerNew[x].Bid
		tick.Low = tickerNew[x].Low
		tick.Last = tickerNew[x].Last
		tick.Volume = tickerNew[x].Volume
		tick.High = tickerNew[x].High

		err = ticker.ProcessTicker(b.Name, &tick, assetType)
		if err != nil {
			return tickerPrice, err
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitfinex) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tick, err := ticker.GetTicker(b.GetName(), p, assets.AssetTypeSpot)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitfinex) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitfinex) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base
	urlVals := url.Values{}
	urlVals.Set("limit_bids", "100")
	urlVals.Set("limit_asks", "100")
	orderbookNew, err := b.GetOrderbook(p.String(), urlVals)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{Price: orderbookNew.Asks[x].Price,
				Amount: orderbookNew.Asks[x].Amount})
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{Price: orderbookNew.Bids[x].Price,
				Amount: orderbookNew.Bids[x].Amount})
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
						CurrencyName: currency.NewCode(bal.Currency),
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
func (b *Bitfinex) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitfinex) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var isBuying bool

	if side == exchange.BuyOrderSide {
		isBuying = true
	}

	response, err := b.NewOrder(p.String(),
		amount,
		price,
		isBuying,
		orderType.ToString(),
		false)

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
func (b *Bitfinex) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitfinex) CancelOrder(order *exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder(orderIDInt)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitfinex) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	_, err := b.CancelAllExistingOrders()
	return exchange.CancelAllOrdersResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (b *Bitfinex) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitfinex) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	method, err := b.ConvertSymbolToDepositMethod(cryptocurrency)
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
func (b *Bitfinex) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	withdrawalType := b.ConvertSymbolToWithdrawalType(withdrawRequest.Currency)
	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawCryptocurrency(withdrawalType,
		walletType,
		withdrawRequest.Address,
		withdrawRequest.Description,
		withdrawRequest.Amount,
		withdrawRequest.Currency)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errors.New("no withdrawID returned. Check order status")
	}

	return fmt.Sprintf("%v", resp[0].WithdrawalID), err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
// Returns comma delimited withdrawal IDs
func (b *Bitfinex) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	withdrawalType := "wire"
	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawFIAT(withdrawalType, walletType, withdrawRequest)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errors.New("no withdrawID returned. Check order status")
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
func (b *Bitfinex) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return b.WithdrawFiatFunds(withdrawRequest)
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitfinex) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitfinex) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	resp, err := b.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := exchange.OrderSide(strings.ToUpper(resp[i].Side))
		timestamp, err := strconv.ParseInt(resp[i].Timestamp, 10, 64)
		if err != nil {
			log.Warnf("Unable to convert timestamp '%v', leaving blank", resp[i].Timestamp)
		}
		orderDate := time.Unix(timestamp, 0)

		orderDetail := exchange.OrderDetail{
			Amount:          resp[i].OriginalAmount,
			OrderDate:       orderDate,
			Exchange:        b.Name,
			ID:              fmt.Sprintf("%v", resp[i].OrderID),
			OrderSide:       orderSide,
			Price:           resp[i].Price,
			RemainingAmount: resp[i].RemainingAmount,
			CurrencyPair:    currency.NewPairFromString(resp[i].Symbol),
			ExecutedAmount:  resp[i].ExecutedAmount,
		}

		switch {
		case resp[i].IsLive:
			orderDetail.Status = string(exchange.ActiveOrderStatus)
		case resp[i].IsCancelled:
			orderDetail.Status = string(exchange.CancelledOrderStatus)
		case resp[i].IsHidden:
			orderDetail.Status = string(exchange.HiddenOrderStatus)
		default:
			orderDetail.Status = string(exchange.UnknownOrderStatus)
		}

		// API docs discrepency. Example contains prefixed "exchange "
		// Return type suggests “market” / “limit” / “stop” / “trailing-stop”
		orderType := strings.Replace(resp[i].Type, "exchange ", "", 1)
		if orderType == "trailing-stop" {
			orderDetail.OrderType = exchange.TrailingStopOrderType
		} else {
			orderDetail.OrderType = exchange.OrderType(strings.ToUpper(orderType))
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitfinex) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	resp, err := b.GetInactiveOrders()
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := exchange.OrderSide(strings.ToUpper(resp[i].Side))
		timestamp, err := strconv.ParseInt(resp[i].Timestamp, 10, 64)
		if err != nil {
			log.Warnf("Unable to convert timestamp '%v', leaving blank", resp[i].Timestamp)
		}
		orderDate := time.Unix(timestamp, 0)

		orderDetail := exchange.OrderDetail{
			Amount:          resp[i].OriginalAmount,
			OrderDate:       orderDate,
			Exchange:        b.Name,
			ID:              fmt.Sprintf("%v", resp[i].OrderID),
			OrderSide:       orderSide,
			Price:           resp[i].Price,
			RemainingAmount: resp[i].RemainingAmount,
			ExecutedAmount:  resp[i].ExecutedAmount,
			CurrencyPair:    currency.NewPairFromString(resp[i].Symbol),
		}

		switch {
		case resp[i].IsLive:
			orderDetail.Status = string(exchange.ActiveOrderStatus)
		case resp[i].IsCancelled:
			orderDetail.Status = string(exchange.CancelledOrderStatus)
		case resp[i].IsHidden:
			orderDetail.Status = string(exchange.HiddenOrderStatus)
		default:
			orderDetail.Status = string(exchange.UnknownOrderStatus)
		}

		// API docs discrepency. Example contains prefixed "exchange "
		// Return type suggests “market” / “limit” / “stop” / “trailing-stop”
		orderType := strings.Replace(resp[i].Type, "exchange ", "", 1)
		if orderType == "trailing-stop" {
			orderDetail.OrderType = exchange.TrailingStopOrderType
		} else {
			orderDetail.OrderType = exchange.OrderType(strings.ToUpper(orderType))
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}
