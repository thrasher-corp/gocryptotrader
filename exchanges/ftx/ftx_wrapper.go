package ftx

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
func (f *Ftx) GetDefaultConfig() (*config.ExchangeConfig, error) {
	f.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = f.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = f.BaseCurrencies

	err := f.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if f.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = f.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Ftx
func (f *Ftx) SetDefaults() {
	f.Name = "Ftx"
	f.Enabled = true
	f.Verbose = true
	f.API.CredentialsValidator.RequiresKey = true
	f.API.CredentialsValidator.RequiresSecret = true
	f.CurrencyPairs = currency.PairsManager{
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
	// Fill out the capabilities/features that the exchange supports
	f.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
	f.Requester = request.New(f.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		nil)
	f.API.Endpoints.URLDefault = ftxAPIURL
	f.API.Endpoints.URL = f.API.Endpoints.URLDefault
	f.Websocket = wshandler.New()
	f.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	f.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	f.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (f *Ftx) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		f.SetEnabled(false)
		return nil
	}

	err := f.SetupDefaults(exch)
	if err != nil {
		return err
	}

	// If websocket is supported, please fill out the following
	/*
		err = f.Websocket.Setup(
			&wshandler.WebsocketSetup{
				Enabled:                          exch.Features.Enabled.Websocket,
				Verbose:                          exch.Verbose,
				AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
				WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
				DefaultURL:                       ftxWSURL,
				ExchangeName:                     exch.Name,
				RunningURL:                       exch.API.Endpoints.WebsocketURL,
				Connector:                        f.WsConnect,
				Subscriber:                       f.Subscribe,
				UnSubscriber:                     f.Unsubscribe,
				Features:                         &f.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		f.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         f.Name,
			URL:                  f.Websocket.GetWebsocketURL(),
			ProxyURL:             f.Websocket.GetProxyAddress(),
			Verbose:              f.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}

		// NOTE: PLEASE ENSURE YOU SET THE ORDERBOOK BUFFER SETTINGS CORRECTLY
		f.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			false,
			false,
			exch.Name)
	*/
	return nil
}

// Start starts the Ftx go routine
func (f *Ftx) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		f.Run()
		wg.Done()
	}()
}

// Run implements the Ftx wrapper
func (f *Ftx) Run() {
	if f.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			f.Name,
			common.IsEnabled(f.Websocket.IsEnabled()))
		f.PrintEnabledPairs()
	}

	if !f.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := f.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			f.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (f *Ftx) FetchTradablePairs(a asset.Item) ([]string, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, f.Name)
	}
	markets, err := f.GetMarkets()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range markets.Result {
		pairs = append(pairs, markets.Result[x].Name)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (f *Ftx) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := f.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}
	return f.UpdatePairs(currency.NewPairsFromStrings(pairs),
		asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (f *Ftx) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	var marketNames []string
	allPairs := f.GetEnabledPairs(assetType)
	for a := range allPairs {
		marketNames = append(marketNames, f.FormatExchangeCurrency(allPairs[a], asset.Spot).String())
	}
	markets, err := f.GetMarkets()
	if err != nil {
		return nil, err
	}
	for x := range markets.Result {
		marketName := currency.NewPairFromString(markets.Result[x].Name)
		if common.StringDataCompareInsensitive(marketNames, marketName.String()) {
			continue
		}
		var resp ticker.Price
		resp.Pair = marketName
		resp.Last = markets.Result[x].Last
		resp.Bid = markets.Result[x].Bid
		resp.Ask = markets.Result[x].Ask
		resp.LastUpdated = time.Now()
		err = ticker.ProcessTicker(f.Name, &resp, assetType)
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(f.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (f *Ftx) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(f.Name, p, assetType)
	if err != nil {
		return f.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (f *Ftx) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(f.Name, currency, assetType)
	if err != nil {
		return f.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (f *Ftx) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	tempResp, err := f.GetOrderbook(f.FormatExchangeCurrency(p, assetType).String(), 0)
	if err != nil {
		return orderBook, err
	}
	for x := range tempResp.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Amount: tempResp.Bids[x].Size,
			Price:  tempResp.Bids[x].Price})
	}
	for y := range tempResp.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Amount: tempResp.Asks[y].Size,
			Price:  tempResp.Asks[y].Price})
	}
	orderBook.Pair = p
	orderBook.ExchangeName = f.Name
	orderBook.AssetType = assetType
	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}
	return orderbook.Get(f.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (f *Ftx) UpdateAccountInfo() (account.Holdings, error) {
	var resp account.Holdings
	data, err := f.GetBalances()
	if err != nil {
		return resp, err
	}
	var acc account.SubAccount
	for key := range data.Result {
		c := currency.NewCode(data.Result[key].Coin)
		hold := data.Result[key].Total - data.Result[key].Free
		total := data.Result[key].Total
		acc.Currencies = append(acc.Currencies,
			account.Balance{CurrencyName: c,
				TotalValue: total,
				Hold:       hold})
	}
	resp.Accounts = append(resp.Accounts, acc)
	resp.Exchange = f.Name

	err = account.Process(&resp)
	if err != nil {
		return account.Holdings{}, err
	}

	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (f *Ftx) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(f.Name)
	if err != nil {
		return f.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (f *Ftx) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (f *Ftx) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (f *Ftx) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	if s.Side == order.Sell {
		s.Side = order.Ask
	}
	if s.Side == order.Buy {
		s.Side = order.Bid
	}

	tempResp, err := f.Order(f.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
		s.Side.String(),
		s.Type.String(),
		"",
		"",
		"",
		s.ClientID,
		s.Price,
		s.Amount)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = strconv.FormatInt(tempResp.Result.ID, 10)
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (f *Ftx) ModifyOrder(action *order.Modify) (string, error) {
	if action.TriggerPrice != 0 {
		a, err := f.ModifyTriggerOrder(action.ID,
			action.Type.String(),
			action.Amount,
			action.TriggerPrice,
			action.Price,
			0)
		return strconv.FormatInt(a.Result.ID, 10), err
	}
	var o ModifyOrder
	var err error
	switch action.ID {
	case "":
		o, err = f.ModifyOrderByClientID(action.ClientOrderID, action.ClientID, action.Price, action.Amount)
	default:
		o, err = f.ModifyPlacedOrder(action.ID, action.ClientID, action.Price, action.Amount)
	}
	return strconv.FormatInt(o.Result.ID, 10), err
}

// CancelOrder cancels an order by its corresponding ID number
func (f *Ftx) CancelOrder(order *order.Cancel) error {
	_, err := f.DeleteOrder(order.ID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (f *Ftx) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	tempMap := make(map[string]string)
	orders, err := f.GetOpenOrders(f.FormatExchangeCurrency(orderCancellation.Pair, asset.Spot).String())
	if err != nil {
		return resp, err
	}
	for x := range orders.Result {
		_, err := f.DeleteOrder(strconv.FormatInt(orders.Result[x].ID, 10))
		if err != nil {
			tempMap[strconv.FormatInt(orders.Result[x].ID, 10)] = "Cancellation Failed"
			continue
		}
		tempMap[strconv.FormatInt(orders.Result[x].ID, 10)] = "Success"
	}
	resp.Status = tempMap
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (f *Ftx) GetOrderInfo(orderID string) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (f *Ftx) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *Ftx) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	var resp *withdraw.ExchangeResponse
	return resp, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *Ftx) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	var resp *withdraw.ExchangeResponse
	return resp, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (f *Ftx) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (f *Ftx) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (f *Ftx) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		var tempResp order.Detail
		orderData, err := f.GetOpenOrders(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String())
		if err != nil {
			return resp, err
		}
		for y := range orderData.Result {
			tempResp.ID = strconv.FormatInt(orderData.Result[y].ID, 10)
			tempResp.Amount = orderData.Result[y].Size
			tempResp.AssetType = asset.Spot
			tempResp.ClientID = orderData.Result[y].ClientID
			tempResp.Date = orderData.Result[y].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = orderData.Result[y].Size - orderData.Result[y].RemainingSize
			// tempResp.Fee = Fee
			tempResp.Pair = currency.NewPairFromString(orderData.Result[y].Market)
			tempResp.Price = orderData.Result[y].Price
			tempResp.RemainingAmount = orderData.Result[y].RemainingSize
			switch orderData.Result[y].Side {
			case buy:
				tempResp.Side = order.Buy
			case sell:
				tempResp.Side = order.Sell
			}
			switch orderData.Result[y].Status {
			case newStatus:
				tempResp.Status = order.New
			case openStatus:
				tempResp.Status = order.Open
			case closedStatus:
				if orderData.Result[y].FilledSize != 0 && orderData.Result[y].FilledSize != orderData.Result[y].Size {
					tempResp.Status = order.PartiallyCancelled
				}
				if orderData.Result[y].FilledSize == 0 {
					tempResp.Status = order.Cancelled
				}
				if orderData.Result[y].FilledSize == orderData.Result[y].Size {
					tempResp.Status = order.Filled
				}
			}
			switch orderData.Result[y].OrderType {
			case marketOrder:
				tempResp.Type = order.Market
			case limitOrder:
				tempResp.Type = order.Limit
			}
			resp = append(resp, tempResp)
		}
		triggerOrderData, err := f.GetOpenTriggerOrders(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String(), getOrdersRequest.Type.String())
		if err != nil {
			return resp, err
		}
		for z := range triggerOrderData.Result {
			tempResp.ID = strconv.FormatInt(triggerOrderData.Result[z].ID, 10)
			tempResp.Amount = triggerOrderData.Result[z].Size
			tempResp.AssetType = asset.Spot
			tempResp.Date = triggerOrderData.Result[z].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = triggerOrderData.Result[z].FilledSize
			// tempResp.Fee = Fee
			tempResp.Pair = currency.NewPairFromString(triggerOrderData.Result[z].Market)
			tempResp.Price = triggerOrderData.Result[z].AvgFillPrice
			tempResp.RemainingAmount = triggerOrderData.Result[z].Size - triggerOrderData.Result[z].FilledSize
			tempResp.TriggerPrice = triggerOrderData.Result[z].TriggerPrice
			switch triggerOrderData.Result[z].Side {
			case buy:
				tempResp.Side = order.Buy
			case sell:
				tempResp.Side = order.Sell
			}
			switch orderData.Result[z].Status {
			case newStatus:
				tempResp.Status = order.New
			case openStatus:
				tempResp.Status = order.Open
			case closedStatus:
				if triggerOrderData.Result[z].FilledSize != 0 && triggerOrderData.Result[z].FilledSize != triggerOrderData.Result[z].Size {
					tempResp.Status = order.PartiallyCancelled
				}
				if triggerOrderData.Result[z].FilledSize == 0 {
					tempResp.Status = order.Cancelled
				}
				if triggerOrderData.Result[z].FilledSize == triggerOrderData.Result[z].Size {
					tempResp.Status = order.Filled
				}
			}
			switch triggerOrderData.Result[z].OrderType {
			case marketOrder:
				tempResp.Type = order.Market
			case limitOrder:
				tempResp.Type = order.Limit
			}
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (f *Ftx) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		var tempResp order.Detail
		fmt.Println(getOrdersRequest.EndTicks.String())
		fmt.Println(getOrdersRequest.StartTicks.String())
		orderData, err := f.FetchOrderHistory(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String(),
			getOrdersRequest.StartTicks.String(), getOrdersRequest.EndTicks.String(), "")
		if err != nil {
			return resp, err
		}
		for y := range orderData.Result {
			tempResp.ID = strconv.FormatInt(orderData.Result[y].ID, 10)
			tempResp.Amount = orderData.Result[y].Size
			tempResp.AssetType = asset.Spot
			tempResp.ClientID = orderData.Result[y].ClientID
			tempResp.Date = orderData.Result[y].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = orderData.Result[y].Size - orderData.Result[y].RemainingSize
			// tempResp.Fee = Fee
			tempResp.Pair = currency.NewPairFromString(orderData.Result[y].Market)
			tempResp.Price = orderData.Result[y].Price
			tempResp.RemainingAmount = orderData.Result[y].RemainingSize
			switch orderData.Result[y].Side {
			case buy:
				tempResp.Side = order.Buy
			case sell:
				tempResp.Side = order.Sell
			}
			switch orderData.Result[y].Status {
			case newStatus:
				tempResp.Status = order.New
			case openStatus:
				tempResp.Status = order.Open
			case closedStatus:
				if orderData.Result[y].FilledSize != 0 && orderData.Result[y].FilledSize != orderData.Result[y].Size {
					tempResp.Status = order.PartiallyCancelled
				}
				if orderData.Result[y].FilledSize == 0 {
					tempResp.Status = order.Cancelled
				}
				if orderData.Result[y].FilledSize == orderData.Result[y].Size {
					tempResp.Status = order.Filled
				}
			}
			switch orderData.Result[y].OrderType {
			case marketOrder:
				tempResp.Type = order.Market
			case limitOrder:
				tempResp.Type = order.Limit
			}
			resp = append(resp, tempResp)
		}
		triggerOrderData, err := f.GetTriggerOrderHistory(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String(),
			getOrdersRequest.StartTicks.String(), getOrdersRequest.EndTicks.String(), getOrdersRequest.Side.String(), getOrdersRequest.Type.String(), "")
		if err != nil {
			return resp, err
		}
		for z := range triggerOrderData.Result {
			tempResp.ID = strconv.FormatInt(triggerOrderData.Result[z].ID, 10)
			tempResp.Amount = triggerOrderData.Result[z].Size
			tempResp.AssetType = asset.Spot
			tempResp.Date = triggerOrderData.Result[z].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = triggerOrderData.Result[z].FilledSize
			// tempResp.Fee = Fee
			tempResp.Pair = currency.NewPairFromString(triggerOrderData.Result[z].Market)
			tempResp.Price = triggerOrderData.Result[z].AvgFillPrice
			tempResp.RemainingAmount = triggerOrderData.Result[z].Size - triggerOrderData.Result[z].FilledSize
			tempResp.TriggerPrice = triggerOrderData.Result[z].TriggerPrice
			switch triggerOrderData.Result[z].Side {
			case buy:
				tempResp.Side = order.Buy
			case sell:
				tempResp.Side = order.Sell
			}
			switch orderData.Result[z].Status {
			case newStatus:
				tempResp.Status = order.New
			case openStatus:
				tempResp.Status = order.Open
			case closedStatus:
				if triggerOrderData.Result[z].FilledSize != 0 && triggerOrderData.Result[z].FilledSize != triggerOrderData.Result[z].Size {
					tempResp.Status = order.PartiallyCancelled
				}
				if triggerOrderData.Result[z].FilledSize == 0 {
					tempResp.Status = order.Cancelled
				}
				if triggerOrderData.Result[z].FilledSize == triggerOrderData.Result[z].Size {
					tempResp.Status = order.Filled
				}
			}
			switch triggerOrderData.Result[z].OrderType {
			case marketOrder:
				tempResp.Type = order.Market
			case limitOrder:
				tempResp.Type = order.Limit
			}
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (f *Ftx) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (f *Ftx) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	f.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (f *Ftx) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	f.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (f *Ftx) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrNotYetImplemented
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (f *Ftx) AuthenticateWebsocket() error {
	return common.ErrNotYetImplemented
}
