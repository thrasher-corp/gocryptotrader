package bitmex

import (
	"errors"
	"fmt"
	"math"
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
func (b *Bitmex) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets the basic defaults for Bitmex
func (b *Bitmex) SetDefaults() {
	b.Name = "Bitmex"
	b.Enabled = true
	b.Verbose = true
	b.APIWithdrawPermissions = exchange.AutoWithdrawCryptoWithAPIPermission |
		exchange.WithdrawCryptoWithEmail |
		exchange.WithdrawCryptoWith2FA |
		exchange.NoFiatWithdrawals
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypePerpetualContract,
			assets.AssetTypeFutures,
			assets.AssetTypeDownsideProfitContract,
			assets.AssetTypeUpsideProfitContract,
		},
		UseGlobalFormat: false,
	}

	// Same format used for perpetual contracts and futures
	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}
	b.CurrencyPairs.Store(assets.AssetTypePerpetualContract, fmt1)
	b.CurrencyPairs.Store(assets.AssetTypeFutures, fmt1)

	// Upside and Downside profit contracts use the same format
	fmt2 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "-",
			Uppercase: true,
		},
	}
	b.CurrencyPairs.Store(assets.AssetTypeDownsideProfitContract, fmt2)
	b.CurrencyPairs.Store(assets.AssetTypeUpsideProfitContract, fmt2)

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
				TickerBatching:  false,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, bitmexAuthRate),
		request.NewRateLimit(time.Second, bitmexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = bitmexAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.WebsocketInit()
	b.Websocket.Functionality = exchange.WebsocketTradeDataSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitmex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return b.WebsocketSetup(b.WsConnector,
		exch.Name,
		exch.Features.Enabled.Websocket,
		bitmexWSURL,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the Bitmex go routine
func (b *Bitmex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitmex wrapper
func (b *Bitmex) Run() {
	if b.Verbose {
		log.Debugf("%s Websocket: %s. (url: %s).\n", b.GetName(), common.IsEnabled(b.Websocket.IsEnabled()), b.API.Endpoints.WebsocketURL)
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
func (b *Bitmex) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	marketInfo, err := b.GetActiveInstruments(&GenericRequestParams{})
	if err != nil {
		return nil, err
	}

	var products []string
	for x := range marketInfo {
		products = append(products, marketInfo[x].Symbol)
	}

	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitmex) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	var assetPairs []string
	for x := range b.CurrencyPairs.AssetTypes {
		switch b.CurrencyPairs.AssetTypes[x] {
		case assets.AssetTypePerpetualContract:
			for y := range pairs {
				if strings.Contains(pairs[y], "USD") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		case assets.AssetTypeFutures:
			for y := range pairs {
				if strings.Contains(pairs[y], "19") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		case assets.AssetTypeDownsideProfitContract:
			for y := range pairs {
				if strings.Contains(pairs[y], "_D") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		case assets.AssetTypeUpsideProfitContract:
			for y := range pairs {
				if strings.Contains(pairs[y], "_U") {
					assetPairs = append(assetPairs, pairs[y])
				}
			}
		}

		err = b.UpdatePairs(currency.NewPairsFromStrings(assetPairs), b.CurrencyPairs.AssetTypes[x], false, false)
		if err != nil {
			log.Warnf("%s failed to update available pairs. Err: %v", b.Name, err)
		}
		assetPairs = nil
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitmex) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price
	currency := b.FormatExchangeCurrency(p, assetType)

	tick, err := b.GetTrade(&GenericRequestParams{
		Symbol:  currency.String(),
		Reverse: true,
		Count:   1})
	if err != nil {
		return tickerPrice, err
	}

	if len(tick) == 0 {
		return tickerPrice, fmt.Errorf("%s REST error: no ticker return", b.Name)
	}

	tickerPrice.Pair = p
	tickerPrice.Last = tick[0].Price
	tickerPrice.Volume = float64(tick[0].Size)
	return tickerPrice, ticker.ProcessTicker(b.Name, &tickerPrice, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitmex) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Bitmex) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitmex) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	var orderBook orderbook.Base

	orderbookNew, err := b.GetOrderbook(OrderBookGetL2Params{
		Symbol: b.FormatExchangeCurrency(p, assetType).String(),
		Depth:  500})
	if err != nil {
		return orderBook, err
	}

	for _, ob := range orderbookNew {
		if strings.EqualFold(ob.Side, exchange.SellOrderSide.ToString()) {
			orderBook.Asks = append(orderBook.Asks,
				orderbook.Item{Amount: float64(ob.Size), Price: ob.Price})
			continue
		}
		if strings.EqualFold(ob.Side, exchange.BuyOrderSide.ToString()) {
			orderBook.Bids = append(orderBook.Bids,
				orderbook.Item{Amount: float64(ob.Size), Price: ob.Price})
			continue
		}
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
// Bitmex exchange
func (b *Bitmex) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	bal, err := b.GetAllUserMargin()
	if err != nil {
		return info, err
	}

	// Need to update to add Margin/Liquidity availibilty
	var balances []exchange.AccountCurrencyInfo
	for i := range bal {
		balances = append(balances, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(bal[i].Currency),
			TotalValue:   float64(bal[i].WalletBalance),
		})
	}

	info.Exchange = b.GetName()
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balances,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitmex) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitmex) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitmex) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, _ string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse

	if math.Mod(amount, 1) != 0 {
		return submitOrderResponse,
			errors.New("contract amount can not have decimals")
	}

	var orderNewParams = OrderNewParams{
		OrdType:  side.ToString(),
		Symbol:   p.String(),
		OrderQty: amount,
		Side:     side.ToString(),
	}

	if orderType == exchange.LimitOrderType {
		orderNewParams.Price = price
	}

	response, err := b.CreateOrder(&orderNewParams)
	if response.OrderID != "" {
		submitOrderResponse.OrderID = response.OrderID
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitmex) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	var params OrderAmendParams

	if math.Mod(action.Amount, 1) != 0 {
		return "", errors.New("contract amount can not have decimals")
	}

	params.OrderID = action.OrderID
	params.OrderQty = int32(action.Amount)
	params.Price = action.Price

	order, err := b.AmendOrder(&params)
	if err != nil {
		return "", err
	}

	return order.OrderID, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitmex) CancelOrder(order *exchange.OrderCancellation) error {
	var params = OrderCancelParams{
		OrderID: order.OrderID,
	}
	_, err := b.CancelOrders(&params)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitmex) CancelAllOrders(_ *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	cancelAllOrdersResponse := exchange.CancelAllOrdersResponse{
		OrderStatus: make(map[string]string),
	}
	var emptyParams OrderCancelAllParams
	orders, err := b.CancelAllExistingOrders(emptyParams)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range orders {
		cancelAllOrdersResponse.OrderStatus[orders[i].OrderID] = orders[i].OrdRejReason
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Bitmex) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitmex) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return b.GetCryptoDepositAddress(cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	var request = UserRequestWithdrawalParams{
		Address:  withdrawRequest.Address,
		Amount:   withdrawRequest.Amount,
		Currency: withdrawRequest.Currency.String(),
		OtpToken: withdrawRequest.OneTimePassword,
	}
	if withdrawRequest.FeeAmount > 0 {
		request.Fee = withdrawRequest.FeeAmount
	}

	resp, err := b.UserRequestWithdrawal(request)
	if err != nil {
		return "", err
	}

	return resp.TransactID, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitmex) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitmex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (b *Bitmex) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	params := OrdersRequest{}
	params.Filter = "{\"open\":true}"

	resp, err := b.GetOrders(&params)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := orderSideMap[resp[i].Side]
		orderType := orderTypeMap[resp[i].OrdType]
		if orderType == "" {
			orderType = exchange.UnknownOrderType
		}

		orderDetail := exchange.OrderDetail{
			Price:     resp[i].Price,
			Amount:    float64(resp[i].OrderQty),
			Exchange:  b.Name,
			ID:        resp[i].OrderID,
			OrderSide: orderSide,
			OrderType: orderType,
			Status:    resp[i].OrdStatus,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].Symbol,
				resp[i].SettlCurrency,
				b.CurrencyPairs.ConfigFormat.Delimiter),
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (b *Bitmex) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	var orders []exchange.OrderDetail
	params := OrdersRequest{}
	resp, err := b.GetOrders(&params)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := orderSideMap[resp[i].Side]
		orderType := orderTypeMap[resp[i].OrdType]
		if orderType == "" {
			orderType = exchange.UnknownOrderType
		}

		orderDetail := exchange.OrderDetail{
			Price:     resp[i].Price,
			Amount:    float64(resp[i].OrderQty),
			Exchange:  b.Name,
			ID:        resp[i].OrderID,
			OrderSide: orderSide,
			OrderType: orderType,
			Status:    resp[i].OrdStatus,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].Symbol,
				resp[i].SettlCurrency,
				b.CurrencyPairs.ConfigFormat.Delimiter),
		}

		orders = append(orders, orderDetail)
	}

	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}
