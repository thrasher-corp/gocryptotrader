package coinbene

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
			asset.PerpetualSwap,
		},
	}

	c.CurrencyPairs.Store(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
	})

	c.CurrencyPairs.Store(asset.PerpetualSwap, currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
	})

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
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
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.NoFiatWithdrawals |
				exchange.WithdrawCryptoViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
	c.Requester = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))

	c.API.Endpoints.URLDefault = coinbeneAPIURL
	c.API.Endpoints.URL = c.API.Endpoints.URLDefault
	c.API.Endpoints.WebsocketURL = wsContractURL
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

	err = c.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       wsContractURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        c.WsConnect,
			Subscriber:                       c.Subscribe,
			UnSubscriber:                     c.Unsubscribe,
			Features:                         &c.Features.Supports.WebsocketCapabilities,
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
	if !c.SupportsAsset(a) {
		return nil, fmt.Errorf("%s does not support asset type %s", c.Name, a)
	}

	var currencies []string
	switch a {
	case asset.Spot:
		pairs, err := c.GetAllPairs()
		if err != nil {
			return nil, err
		}

		for x := range pairs {
			currencies = append(currencies, pairs[x].Symbol)
		}
	case asset.PerpetualSwap:
		tickers, err := c.GetSwapTickers()
		if err != nil {
			return nil, err
		}
		for t := range tickers {
			idx := strings.Index(t, currency.USDT.String())
			if idx == 0 {
				return nil, fmt.Errorf("%s SWAP currency does not contain USDT", c.Name)
			}
			currencies = append(currencies,
				t[0:idx]+c.GetPairFormat(a, false).Delimiter+t[idx:])
		}
	}
	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them
func (c *Coinbene) UpdateTradablePairs(forceUpdate bool) error {
	assets := c.GetAssetTypes()
	for x := range assets {
		pairs, err := c.FetchTradablePairs(assets[x])
		if err != nil {
			return err
		}
		err = c.UpdatePairs(currency.NewPairsFromStrings(pairs),
			assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *Coinbene) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	resp := new(ticker.Price)
	if !c.SupportsAsset(assetType) {
		return nil,
			fmt.Errorf("%s does not support asset type %s", c.Name, assetType)
	}

	switch assetType {
	case asset.Spot:
		allPairs := c.GetEnabledPairs(assetType)
		for x := range allPairs {
			tempResp, err := c.GetTicker(c.FormatExchangeCurrency(allPairs[x],
				assetType).String())
			if err != nil {
				return nil, err
			}
			resp.Pair = allPairs[x]
			resp.Last = tempResp.LatestPrice
			resp.High = tempResp.DailyHigh
			resp.Low = tempResp.DailyLow
			resp.Bid = tempResp.BestBid
			resp.Ask = tempResp.BestAsk
			resp.Volume = tempResp.DailyVolume
			resp.LastUpdated = time.Now()
			err = ticker.ProcessTicker(c.Name, resp, assetType)
			if err != nil {
				return nil, err
			}
		}
	case asset.PerpetualSwap:
		tickers, err := c.GetSwapTickers()
		if err != nil {
			return nil, err
		}

		allPairs := c.GetEnabledPairs(assetType)
		for x := range allPairs {
			tick, ok := tickers[c.FormatExchangeCurrency(allPairs[x],
				assetType).String()]
			if !ok {
				log.Warnf(log.ExchangeSys,
					"%s SWAP ticker item was not found", c.Name)
				continue
			}
			resp.Pair = allPairs[x]
			resp.Last = tick.LastPrice
			resp.High = tick.High24Hour
			resp.Low = tick.Low24Hour
			resp.Bid = tick.BestBidPrice
			resp.Ask = tick.BestAskPrice
			resp.Volume = tick.Volume24Hour
			resp.LastUpdated = tick.Timestamp
			err = ticker.ProcessTicker(c.Name, resp, assetType)
			if err != nil {
				return nil, err
			}
		}
	}
	return ticker.GetTicker(c.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (c *Coinbene) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if !c.SupportsAsset(assetType) {
		return nil,
			fmt.Errorf("%s does not support asset type %s", c.Name, assetType)
	}

	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *Coinbene) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if !c.SupportsAsset(assetType) {
		return nil,
			fmt.Errorf("%s does not support asset type %s", c.Name, assetType)
	}

	ob, err := orderbook.Get(c.Name, currency, assetType)
	if err != nil {
		return c.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *Coinbene) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	resp := new(orderbook.Base)
	if !c.SupportsAsset(assetType) {
		return nil,
			fmt.Errorf("%s does not support asset type %s", c.Name, assetType)
	}

	var tempResp Orderbook
	var err error

	switch assetType {
	case asset.Spot:
		tempResp, err = c.GetOrderbook(
			c.FormatExchangeCurrency(p, assetType).String(),
			100, // TO-DO: Update this once we support configurable orderbook depth
		)
	case asset.PerpetualSwap:
		tempResp, err = c.GetSwapOrderbook(
			c.FormatExchangeCurrency(p, assetType).String(),
			100, // TO-DO: Update this once we support configurable orderbook depth
		)
	}
	if err != nil {
		return nil, err
	}
	resp.ExchangeName = c.Name
	resp.Pair = p
	resp.AssetType = assetType
	for x := range tempResp.Asks {
		item := orderbook.Item{
			Price:  tempResp.Asks[x].Price,
			Amount: tempResp.Asks[x].Amount,
		}
		if assetType == asset.PerpetualSwap {
			item.OrderCount = tempResp.Asks[x].Count
		}
		resp.Asks = append(resp.Asks, item)
	}
	for x := range tempResp.Bids {
		item := orderbook.Item{
			Price:  tempResp.Bids[x].Price,
			Amount: tempResp.Bids[x].Amount,
		}
		if assetType == asset.PerpetualSwap {
			item.OrderCount = tempResp.Bids[x].Count
		}
		resp.Bids = append(resp.Bids, item)
	}
	err = resp.Process()
	if err != nil {
		return nil, err
	}
	return orderbook.Get(c.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Coinbene exchange
func (c *Coinbene) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
	balance, err := c.GetAccountBalances()
	if err != nil {
		return info, err
	}
	var acc account.SubAccount
	for key := range balance {
		c := currency.NewCode(balance[key].Asset)
		hold := balance[key].Reserved
		available := balance[key].Available
		acc.Currencies = append(acc.Currencies,
			account.Balance{
				CurrencyName: c,
				TotalValue:   hold + available,
				Hold:         hold,
			})
	}
	info.Accounts = append(info.Accounts, acc)
	info.Exchange = c.Name

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (c *Coinbene) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(c.Name)
	if err != nil {
		return c.UpdateAccountInfo()
	}

	return acc, nil
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

	if s.Side != order.Buy && s.Side != order.Sell {
		return resp,
			fmt.Errorf("%s orderside is not supported by this exchange",
				s.Side)
	}
	if s.Type != order.Limit {
		return resp, fmt.Errorf("only limit order is supported by this exchange")
	}

	tempResp, err := c.PlaceSpotOrder(s.Price,
		s.Amount,
		c.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
		s.Side.String(),
		s.Type.String(),
		s.ClientID,
		0)
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
	_, err := c.CancelSpotOrder(order.ID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *Coinbene) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	orders, err := c.FetchOpenSpotOrders(
		c.FormatExchangeCurrency(orderCancellation.Pair,
			asset.Spot).String(),
	)
	if err != nil {
		return resp, err
	}
	tempMap := make(map[string]string)
	for x := range orders {
		_, err := c.CancelSpotOrder(orders[x].OrderID)
		if err != nil {
			tempMap[orders[x].OrderID] = "Failed"
		} else {
			tempMap[orders[x].OrderID] = "Success"
		}
	}
	resp.Status = tempMap
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (c *Coinbene) GetOrderInfo(orderID string) (order.Detail, error) {
	var resp order.Detail
	tempResp, err := c.FetchSpotOrderInfo(orderID)
	if err != nil {
		return resp, err
	}
	resp.Exchange = c.Name
	resp.ID = orderID
	resp.Pair = currency.NewPairWithDelimiter(tempResp.BaseAsset,
		"/",
		tempResp.QuoteAsset)
	resp.Price = tempResp.OrderPrice
	resp.Date = tempResp.OrderTime
	resp.ExecutedAmount = tempResp.FilledAmount
	resp.Fee = tempResp.TotalFee
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *Coinbene) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (c *Coinbene) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *Coinbene) GetWebsocket() (*wshandler.Websocket, error) {
	return c.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (c *Coinbene) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(getOrdersRequest.Pairs) == 0 {
		allPairs, err := c.GetAllPairs()
		if err != nil {
			return nil, err
		}
		for a := range allPairs {
			getOrdersRequest.Pairs = append(getOrdersRequest.Pairs,
				currency.NewPairFromString(allPairs[a].Symbol))
		}
	}

	var err error
	var resp []order.Detail

	for x := range getOrdersRequest.Pairs {
		var tempData OrdersInfo
		tempData, err = c.FetchOpenSpotOrders(
			c.FormatExchangeCurrency(
				getOrdersRequest.Pairs[x],
				asset.Spot).String(),
		)
		if err != nil {
			return nil, err
		}

		for y := range tempData {
			var tempResp order.Detail
			tempResp.Exchange = c.Name
			tempResp.Pair = getOrdersRequest.Pairs[x]
			tempResp.Side = order.Buy
			if strings.EqualFold(tempData[y].OrderType, order.Sell.String()) {
				tempResp.Side = order.Sell
			}
			tempResp.Date = tempData[y].OrderTime
			tempResp.Status = order.Status(tempData[y].OrderStatus)
			tempResp.Price = tempData[y].OrderPrice
			tempResp.Amount = tempData[y].Amount
			tempResp.ExecutedAmount = tempData[y].FilledAmount
			tempResp.RemainingAmount = tempData[y].Amount - tempData[y].FilledAmount
			tempResp.Fee = tempData[y].TotalFee
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *Coinbene) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(getOrdersRequest.Pairs) == 0 {
		allPairs, err := c.GetAllPairs()
		if err != nil {
			return nil, err
		}
		for a := range allPairs {
			getOrdersRequest.Pairs = append(getOrdersRequest.Pairs,
				currency.NewPairFromString(allPairs[a].Symbol))
		}
	}

	var resp []order.Detail
	var tempData OrdersInfo
	var err error

	for x := range getOrdersRequest.Pairs {
		tempData, err = c.FetchClosedOrders(
			c.FormatExchangeCurrency(
				getOrdersRequest.Pairs[x],
				asset.Spot).String(),
			"",
		)
		if err != nil {
			return nil, err
		}

		for y := range tempData {
			var tempResp order.Detail
			tempResp.Exchange = c.Name
			tempResp.Pair = getOrdersRequest.Pairs[x]
			tempResp.Side = order.Buy
			if strings.EqualFold(tempData[y].OrderType, order.Sell.String()) {
				tempResp.Side = order.Sell
			}
			tempResp.Date = tempData[y].OrderTime
			tempResp.Status = order.Status(tempData[y].OrderStatus)
			tempResp.Price = tempData[y].OrderPrice
			tempResp.Amount = tempData[y].Amount
			tempResp.ExecutedAmount = tempData[y].FilledAmount
			tempResp.RemainingAmount = tempData[y].Amount - tempData[y].FilledAmount
			tempResp.Fee = tempData[y].TotalFee
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
		fee = feeBuilder.PurchasePrice * feeBuilder.Amount * tempData.MakerFeeRate
	case false:
		fee = feeBuilder.PurchasePrice * feeBuilder.Amount * tempData.TakerFeeRate
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

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (c *Coinbene) ValidateCredentials() error {
	_, err := c.UpdateAccountInfo()
	return c.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (c *Coinbene) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
