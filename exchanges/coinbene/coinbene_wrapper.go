package coinbene

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (c *Coinbene) GetDefaultConfig() (*config.ExchangeConfig, error) {
	c.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = c.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = c.BaseCurrencies

	err := c.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if c.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = c.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Coinbene
func (c *Coinbene) SetDefaults() {
	c.Name = "Coinbene"
	c.Enabled = true
	c.Verbose = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresSecret = true

	c.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
	}

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false, // Purposely disabled until SWAP is supported
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AccountBalance:    true,
				AutoPairUpdates:   true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrder:       true,
				CancelOrders:      true,
				SubmitOrder:       true,
				TradeFee:          true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				AccountBalance:         true,
				AccountInfo:            true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				KlineFetching:          true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
			},
			WithdrawPermissions: exchange.NoFiatWithdrawals |
				exchange.WithdrawCryptoViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
	c.Requester = request.New(c.Name,
		request.NewRateLimit(time.Minute, authRateLimit),
		request.NewRateLimit(time.Second, unauthRateLimit),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	c.API.Endpoints.URLDefault = coinbeneAPIURL
	c.API.Endpoints.URL = c.API.Endpoints.URLDefault
	c.API.Endpoints.WebsocketURL = coinbeneWsURL
	c.Websocket = wshandler.New()
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (c *Coinbene) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		c.SetEnabled(false)
		return nil
	}

	err := c.SetupDefaults(exch)
	if err != nil {
		return err
	}

	// TO-DO: Remove this once SWAP is supported
	if exch.Features.Enabled.Websocket {
		log.Warnf(log.ExchangeSys,
			"%s websocket only supports SWAP which GoCryptoTrader currently "+
				"does not. Disabling.\n",
			c.Name)
		exch.Features.Enabled.Websocket = false
	}

	err = c.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       coinbeneWsURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        c.WsConnect,
			Subscriber:                       c.Subscribe,
			UnSubscriber:                     c.Unsubscribe,
		})
	if err != nil {
		return err
	}

	c.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         c.Name,
		URL:                  c.Websocket.GetWebsocketURL(),
		ProxyURL:             c.Websocket.GetProxyAddress(),
		Verbose:              c.Verbose,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	c.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		true,
		true,
		false,
		false,
		exch.Name)

	return nil
}

// Start starts the Coinbene go routine
func (c *Coinbene) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the Coinbene wrapper
func (c *Coinbene) Run() {
	if c.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			c.Name,
			common.IsEnabled(c.Websocket.IsEnabled()),
			c.Websocket.GetWebsocketURL(),
		)
		c.PrintEnabledPairs()
	}

	if !c.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := c.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s Failed to update tradable pairs. Error: %s",
			c.Name,
			err)
	}
}

// FetchTradablePairs returns a list of exchange tradable pairs
func (c *Coinbene) FetchTradablePairs(a asset.Item) ([]string, error) {
	pairs, err := c.GetAllPairs()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range pairs.Data {
		currencies = append(currencies, pairs.Data[x].Symbol)
	}
	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them
func (c *Coinbene) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return c.UpdatePairs(currency.NewPairsFromStrings(pairs),
		asset.Spot,
		false,
		forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *Coinbene) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var resp ticker.Price
	allPairs := c.GetEnabledPairs(assetType)
	for x := range allPairs {
		tempResp, err := c.GetTicker(c.FormatExchangeCurrency(allPairs[x],
			assetType).String())
		if err != nil {
			return resp, err
		}
		resp.Pair = allPairs[x]
		resp.Last = tempResp.TickerData.LatestPrice
		resp.High = tempResp.TickerData.DailyHigh
		resp.Low = tempResp.TickerData.DailyLow
		resp.Bid = tempResp.TickerData.BestBid
		resp.Ask = tempResp.TickerData.BestAsk
		resp.Volume = tempResp.TickerData.DailyVolume
		resp.LastUpdated = time.Now()
		err = ticker.ProcessTicker(c.Name, &resp, assetType)
		if err != nil {
			return resp, err
		}
	}
	return ticker.GetTicker(c.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (c *Coinbene) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *Coinbene) FetchOrderbook(currency currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(c.Name, currency, assetType)
	if err != nil {
		return c.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *Coinbene) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var resp orderbook.Base
	tempResp, err := c.GetOrderbook(
		c.FormatExchangeCurrency(p, assetType).String(),
		100,
	)
	if err != nil {
		return resp, err
	}
	resp.ExchangeName = c.Name
	resp.Pair = p
	resp.AssetType = assetType
	var amount, price float64
	for i := range tempResp.Orderbook.Asks {
		amount, err = strconv.ParseFloat(tempResp.Orderbook.Asks[i][1], 64)
		if err != nil {
			return resp, err
		}
		price, err = strconv.ParseFloat(tempResp.Orderbook.Asks[i][0], 64)
		if err != nil {
			return resp, err
		}
		resp.Asks = append(resp.Asks, orderbook.Item{
			Price:  price,
			Amount: amount})
	}
	for j := range tempResp.Orderbook.Bids {
		amount, err = strconv.ParseFloat(tempResp.Orderbook.Bids[j][1], 64)
		if err != nil {
			return resp, err
		}
		price, err = strconv.ParseFloat(tempResp.Orderbook.Bids[j][0], 64)
		if err != nil {
			return resp, err
		}
		resp.Bids = append(resp.Bids, orderbook.Item{
			Price:  price,
			Amount: amount})
	}
	err = resp.Process()
	if err != nil {
		return resp, err
	}
	return orderbook.Get(c.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// Coinbene exchange
func (c *Coinbene) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo
	data, err := c.GetUserBalance()
	if err != nil {
		return info, err
	}
	var account exchange.Account
	for key := range data.Data {
		c := currency.NewCode(data.Data[key].Asset)
		hold := data.Data[key].Reserved
		available := data.Data[key].Available
		account.Currencies = append(account.Currencies,
			exchange.AccountCurrencyInfo{CurrencyName: c,
				TotalValue: hold + available,
				Hold:       hold})
	}
	info.Accounts = append(info.Accounts, account)
	info.Exchange = c.Name
	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *Coinbene) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *Coinbene) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (c *Coinbene) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	if s.OrderSide != order.Buy && s.OrderSide != order.Sell {
		return resp,
			fmt.Errorf("%s orderside is not supported by this exchange",
				s.OrderSide)
	}

	if s.OrderType != order.Limit {
		return resp, fmt.Errorf("only limit order is supported by this exchange")
	}
	tempResp, err := c.PlaceOrder(s.Price,
		s.Amount,
		c.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
		s.OrderType.String(),
		s.ClientID)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = tempResp.OrderID
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *Coinbene) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *Coinbene) CancelOrder(order *order.Cancel) error {
	_, err := c.RemoveOrder(order.OrderID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *Coinbene) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	tempMap := make(map[string]string)
	orders, err := c.FetchOpenOrders(
		c.FormatExchangeCurrency(orderCancellation.CurrencyPair,
			asset.Spot).String(),
	)
	if err != nil {
		return resp, err
	}
	for x := range orders.OpenOrders {
		_, err := c.RemoveOrder(orders.OpenOrders[x].OrderID)
		if err != nil {
			tempMap[orders.OpenOrders[x].OrderID] = "Failed"
		} else {
			tempMap[orders.OpenOrders[x].OrderID] = "Success"
		}
	}
	resp.Status = tempMap
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (c *Coinbene) GetOrderInfo(orderID string) (order.Detail, error) {
	var resp order.Detail
	tempResp, err := c.FetchOrderInfo(orderID)
	if err != nil {
		return resp, err
	}
	var t time.Time
	resp.Exchange = c.Name
	resp.ID = orderID
	resp.CurrencyPair = currency.NewPairWithDelimiter(tempResp.Order.BaseAsset,
		"/",
		tempResp.Order.QuoteAsset)
	t, err = time.Parse(time.RFC3339, tempResp.Order.OrderTime)
	if err != nil {
		return resp, err
	}
	resp.Price = tempResp.Order.OrderPrice
	resp.OrderDate = t
	resp.ExecutedAmount = tempResp.Order.FilledAmount
	resp.Fee = tempResp.Order.TotalFee
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *Coinbene) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.CryptoWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFunds(withdrawRequest *withdraw.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *Coinbene) GetWebsocket() (*wshandler.Websocket, error) {
	return c.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (c *Coinbene) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	var tempResp order.Detail
	var tempData OpenOrderResponse
	if len(getOrdersRequest.Currencies) == 0 {
		allPairs, err := c.GetAllPairs()
		if err != nil {
			return resp, err
		}
		for a := range allPairs.Data {
			getOrdersRequest.Currencies = append(getOrdersRequest.Currencies, currency.NewPairFromString(allPairs.Data[a].Symbol))
		}
	}
	var err error
	for x := range getOrdersRequest.Currencies {
		tempData, err = c.FetchOpenOrders(
			c.FormatExchangeCurrency(
				getOrdersRequest.Currencies[x],
				asset.Spot).String(),
		)
		if err != nil {
			return resp, err
		}
		var t time.Time
		for y := range tempData.OpenOrders {
			tempResp.Exchange = c.Name
			tempResp.CurrencyPair = getOrdersRequest.Currencies[x]
			tempResp.OrderSide = order.Buy
			if strings.EqualFold(tempData.OpenOrders[y].OrderType, order.Sell.String()) {
				tempResp.OrderSide = order.Sell
			}
			t, err = time.Parse(time.RFC3339, tempData.OpenOrders[y].OrderTime)
			if err != nil {
				return resp, err
			}
			tempResp.OrderDate = t
			tempResp.Status = order.Status(tempData.OpenOrders[y].OrderStatus)
			tempResp.Price = tempData.OpenOrders[y].OrderPrice
			tempResp.Amount = tempData.OpenOrders[y].Amount
			tempResp.ExecutedAmount = tempData.OpenOrders[y].FilledAmount
			tempResp.RemainingAmount = tempData.OpenOrders[y].Amount - tempData.OpenOrders[y].FilledAmount
			tempResp.Fee = tempData.OpenOrders[y].TotalFee
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *Coinbene) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	var tempResp order.Detail
	var tempData ClosedOrderResponse
	if len(getOrdersRequest.Currencies) == 0 {
		allPairs, err := c.GetAllPairs()
		if err != nil {
			return resp, err
		}
		for a := range allPairs.Data {
			getOrdersRequest.Currencies = append(getOrdersRequest.Currencies, currency.NewPairFromString(allPairs.Data[a].Symbol))
		}
	}
	var err error
	for x := range getOrdersRequest.Currencies {
		tempData, err = c.FetchClosedOrders(
			c.FormatExchangeCurrency(
				getOrdersRequest.Currencies[x],
				asset.Spot).String(),
			"",
		)
		if err != nil {
			return resp, err
		}
		var t time.Time
		for y := range tempData.Data {
			tempResp.Exchange = c.Name
			tempResp.CurrencyPair = getOrdersRequest.Currencies[x]
			tempResp.OrderSide = order.Buy
			if strings.EqualFold(tempData.Data[y].OrderType, order.Sell.String()) {
				tempResp.OrderSide = order.Sell
			}
			t, err = time.Parse(time.RFC3339, tempData.Data[y].OrderTime)
			if err != nil {
				return resp, err
			}
			tempResp.OrderDate = t
			tempResp.Status = order.Status(tempData.Data[y].OrderStatus)
			tempResp.Price = tempData.Data[y].OrderPrice
			tempResp.Amount = tempData.Data[y].Amount
			tempResp.ExecutedAmount = tempData.Data[y].FilledAmount
			tempResp.RemainingAmount = tempData.Data[y].Amount - tempData.Data[y].FilledAmount
			tempResp.Fee = tempData.Data[y].TotalFee
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (c *Coinbene) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	tempData, err := c.GetPairInfo(
		c.FormatExchangeCurrency(
			feeBuilder.Pair, asset.Spot).String(),
	)
	if err != nil {
		return fee, err
	}
	switch feeBuilder.IsMaker {
	case true:
		fee = feeBuilder.PurchasePrice * feeBuilder.Amount * tempData.Data.MakerFeeRate
	case false:
		fee = feeBuilder.PurchasePrice * feeBuilder.Amount * tempData.Data.TakerFeeRate
	}
	return fee, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (c *Coinbene) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (c *Coinbene) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (c *Coinbene) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return c.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (c *Coinbene) AuthenticateWebsocket() error {
	return c.Login()
}
