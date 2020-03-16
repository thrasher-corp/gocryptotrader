package coinut

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
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
func (c *COINUT) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets current default values
func (c *COINUT) SetDefaults() {
	c.Name = "COINUT"
	c.Enabled = true
	c.Verbose = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresClientID = true

	c.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				SubmitOrders:      true,
				UserTradeHistory:  true,
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				GetOrders:              true,
				CancelOrders:           true,
				CancelOrder:            true,
				SubmitOrder:            true,
				SubmitOrders:           true,
				UserTradeHistory:       true,
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
			},
			WithdrawPermissions: exchange.WithdrawCryptoViaWebsiteOnly |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	c.Requester = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		nil)

	c.API.Endpoints.URLDefault = coinutAPIURL
	c.API.Endpoints.URL = c.API.Endpoints.URLDefault
	c.API.Endpoints.WebsocketURL = coinutWebsocketURL
	c.Websocket = wshandler.New()
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
	rand.Seed(time.Now().UnixNano())
}

// Setup sets the current exchange configuration
func (c *COINUT) Setup(exch *config.ExchangeConfig) error {
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
			DefaultURL:                       coinutWebsocketURL,
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
		true,
		false,
		exch.Name)
	return nil
}

// Start starts the COINUT go routine
func (c *COINUT) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the COINUT wrapper
func (c *COINUT) Run() {
	if c.Verbose {
		log.Debugf(log.ExchangeSys, "%s Websocket: %s. (url: %s).\n", c.Name, common.IsEnabled(c.Websocket.IsEnabled()), coinutWebsocketURL)
		c.PrintEnabledPairs()
	}

	forceUpdate := false
	delim := c.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(c.CurrencyPairs.GetPairs(asset.Spot,
		true).Strings(), delim) ||
		!common.StringDataContains(c.CurrencyPairs.GetPairs(asset.Spot,
			false).Strings(), delim) {
		enabledPairs := currency.NewPairsFromStrings(
			[]string{currency.LTC.String() + delim + currency.USDT.String()},
		)
		log.Warn(log.ExchangeSys,
			"Enabled pairs for Coinut reset due to config upgrade, please enable the ones you would like to use again")
		forceUpdate = true

		err := c.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update currencies. Err: %s\n", c.Name, err)
		}
	}

	if !c.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := c.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *COINUT) FetchTradablePairs(asset asset.Item) ([]string, error) {
	var instruments map[string][]InstrumentBase
	var resp Instruments
	var err error
	if c.Websocket.IsConnected() {
		resp, err = c.WsGetInstruments()
		if err != nil {
			return nil, err
		}
	} else {
		resp, err = c.GetInstruments()
		if err != nil {
			return nil, err
		}
	}
	instruments = resp.Instruments
	var pairs []string
	for i := range instruments {
		c.instrumentMap.Seed(instruments[i][0].Base+instruments[i][0].Quote, instruments[i][0].InstrumentID)
		p := instruments[i][0].Base + c.GetPairFormat(asset, false).Delimiter + instruments[i][0].Quote
		pairs = append(pairs, p)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *COINUT) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return c.UpdatePairs(currency.NewPairsFromStrings(pairs),
		asset.Spot, false, forceUpdate)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// COINUT exchange
func (c *COINUT) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
	var bal *UserBalance
	var err error
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var resp *UserBalance
		resp, err = c.wsGetAccountBalance()
		if err != nil {
			return info, err
		}
		bal = resp
	} else {
		bal, err = c.GetUserBalance()
		if err != nil {
			return info, err
		}
	}

	var balances = []account.Balance{
		{
			CurrencyName: currency.BCH,
			TotalValue:   bal.BCH,
		},
		{
			CurrencyName: currency.BTC,
			TotalValue:   bal.BTC,
		},
		{
			CurrencyName: currency.BTG,
			TotalValue:   bal.BTG,
		},
		{
			CurrencyName: currency.CAD,
			TotalValue:   bal.CAD,
		},
		{
			CurrencyName: currency.ETC,
			TotalValue:   bal.ETC,
		},
		{
			CurrencyName: currency.ETH,
			TotalValue:   bal.ETH,
		},
		{
			CurrencyName: currency.LCH,
			TotalValue:   bal.LCH,
		},
		{
			CurrencyName: currency.LTC,
			TotalValue:   bal.LTC,
		},
		{
			CurrencyName: currency.MYR,
			TotalValue:   bal.MYR,
		},
		{
			CurrencyName: currency.SGD,
			TotalValue:   bal.SGD,
		},
		{
			CurrencyName: currency.USD,
			TotalValue:   bal.USD,
		},
		{
			CurrencyName: currency.USDT,
			TotalValue:   bal.USDT,
		},
		{
			CurrencyName: currency.XMR,
			TotalValue:   bal.XMR,
		},
		{
			CurrencyName: currency.ZEC,
			TotalValue:   bal.ZEC,
		},
	}
	info.Exchange = c.Name
	info.Accounts = append(info.Accounts, account.SubAccount{
		Currencies: balances,
	})

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (c *COINUT) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(c.Name)
	if err != nil {
		return c.UpdateAccountInfo()
	}

	return acc, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *COINUT) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return tickerPrice, err
	}

	instID := c.instrumentMap.LookupID(c.FormatExchangeCurrency(p,
		assetType).String())
	if instID == 0 {
		return tickerPrice, errors.New("unable to lookup instrument ID")
	}
	var tick Ticker
	tick, err = c.GetInstrumentTicker(instID)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice = &ticker.Price{
		Last:        tick.Last,
		High:        tick.High24,
		Low:         tick.Low24,
		Bid:         tick.HighestBuy,
		Ask:         tick.LowestSell,
		Volume:      tick.Volume24,
		Pair:        p,
		LastUpdated: time.Unix(0, tick.Timestamp),
	}
	err = ticker.ProcessTicker(c.Name, tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(c.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (c *COINUT) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *COINUT) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(c.Name, p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *COINUT) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return orderBook, err
	}

	instID := c.instrumentMap.LookupID(c.FormatExchangeCurrency(p,
		assetType).String())
	if instID == 0 {
		return orderBook, errLookupInstrumentID
	}

	orderbookNew, err := c.GetInstrumentOrderbook(instID, 200)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Buy {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Buy[x].Quantity, Price: orderbookNew.Buy[x].Price})
	}

	for x := range orderbookNew.Sell {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Sell[x].Quantity, Price: orderbookNew.Sell[x].Price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = c.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(c.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *COINUT) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *COINUT) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (c *COINUT) SubmitOrder(o *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	var err error
	if _, err = strconv.Atoi(o.ClientID); err != nil {
		return submitOrderResponse, fmt.Errorf("%s - ClientID must be a number, received: %s", c.Name, o.ClientID)
	}
	err = o.Validate()

	if err != nil {
		return submitOrderResponse, err
	}

	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var response *order.Detail
		response, err = c.wsSubmitOrder(&WsSubmitOrderParameters{
			Currency: o.Pair,
			Side:     o.Side,
			Amount:   o.Amount,
			Price:    o.Price,
		})
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = response.ID
		submitOrderResponse.IsOrderPlaced = true
	} else {
		err = c.loadInstrumentsIfNotLoaded()
		if err != nil {
			return submitOrderResponse, err
		}

		currencyID := c.instrumentMap.LookupID(c.FormatExchangeCurrency(o.Pair,
			asset.Spot).String())
		if currencyID == 0 {
			return submitOrderResponse, errLookupInstrumentID
		}

		var APIResponse interface{}
		var clientIDInt uint64
		isBuyOrder := o.Side == order.Buy
		clientIDInt, err = strconv.ParseUint(o.ClientID, 0, 32)
		if err != nil {
			return submitOrderResponse, err
		}
		clientIDUint := uint32(clientIDInt)
		APIResponse, err = c.NewOrder(currencyID, o.Amount, o.Price,
			isBuyOrder, clientIDUint)
		if err != nil {
			return submitOrderResponse, err
		}
		responseMap := APIResponse.(map[string]interface{})
		switch responseMap["reply"].(string) {
		case "order_rejected":
			return submitOrderResponse, fmt.Errorf("clientOrderID: %v was rejected: %v", o.ClientID, responseMap["reasons"])
		case "order_filled":
			orderID := responseMap["order_id"].(float64)
			submitOrderResponse.OrderID = strconv.FormatFloat(orderID, 'f', -1, 64)
			submitOrderResponse.IsOrderPlaced = true
			submitOrderResponse.FullyMatched = true
			return submitOrderResponse, nil
		case "order_accepted":
			orderID := responseMap["order_id"].(float64)
			submitOrderResponse.OrderID = strconv.FormatFloat(orderID, 'f', -1, 64)
			submitOrderResponse.IsOrderPlaced = true
			return submitOrderResponse, nil
		}
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *COINUT) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *COINUT) CancelOrder(o *order.Cancel) error {
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return err
	}
	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	currencyID := c.instrumentMap.LookupID(c.FormatExchangeCurrency(
		o.Pair,
		asset.Spot).String(),
	)
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var resp *CancelOrdersResponse
		resp, err = c.wsCancelOrder(&WsCancelOrderParameters{
			Currency: o.Pair,
			OrderID:  orderIDInt,
		})
		if err != nil {
			return err
		}
		if len(resp.Status) >= 1 && resp.Status[0] != "OK" {
			return errors.New(c.Name + " - Failed to cancel order " + o.ID)
		}
	} else {
		if currencyID == 0 {
			return errLookupInstrumentID
		}
		_, err = c.CancelExistingOrder(currencyID, orderIDInt)
		if err != nil {
			return err
		}
	}

	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *COINUT) CancelAllOrders(details *order.Cancel) (order.CancelAllResponse, error) {
	var cancelAllOrdersResponse order.CancelAllResponse
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return cancelAllOrdersResponse, err
	}
	cancelAllOrdersResponse.Status = make(map[string]string)
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		openOrders, err := c.wsGetOpenOrders(details.Pair.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		var ordersToCancel []WsCancelOrderParameters
		for i := range openOrders.Orders {
			if openOrders.Orders[i].InstrumentID == c.instrumentMap.LookupID(c.FormatExchangeCurrency(details.Pair, asset.Spot).String()) {
				ordersToCancel = append(ordersToCancel, WsCancelOrderParameters{
					Currency: details.Pair,
					OrderID:  openOrders.Orders[i].OrderID,
				})
			}
		}
		resp, err := c.wsCancelOrders(ordersToCancel)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range resp.Results {
			if openOrders.Orders[i].Status[0] != "OK" {
				cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders.Orders[i].OrderID, 10)] = strings.Join(openOrders.Orders[i].Status, ",")
			}
		}
	} else {
		var allTheOrders []OrderResponse
		ids := c.instrumentMap.GetInstrumentIDs()
		for x := range ids {
			if ids[x] == c.instrumentMap.LookupID(c.FormatExchangeCurrency(details.Pair, asset.Spot).String()) {
				openOrders, err := c.GetOpenOrders(ids[x])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				allTheOrders = append(allTheOrders, openOrders.Orders...)
			}
		}

		var allTheOrdersToCancel []CancelOrders
		for i := range allTheOrders {
			cancelOrder := CancelOrders{
				InstrumentID: allTheOrders[i].InstrumentID,
				OrderID:      allTheOrders[i].OrderID,
			}
			allTheOrdersToCancel = append(allTheOrdersToCancel, cancelOrder)
		}

		if len(allTheOrdersToCancel) > 0 {
			resp, err := c.CancelOrders(allTheOrdersToCancel)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			for i := range resp.Results {
				if resp.Results[i].Status != "OK" {
					cancelAllOrdersResponse.Status[strconv.FormatInt(resp.Results[i].OrderID, 10)] = resp.Results[i].Status
				}
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (c *COINUT) GetOrderInfo(orderID string) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *COINUT) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *COINUT) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *COINUT) GetWebsocket() (*wshandler.Websocket, error) {
	return c.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *COINUT) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !c.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *COINUT) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	var currenciesToCheck []string
	if len(req.Pairs) == 0 {
		for i := range req.Pairs {
			currenciesToCheck = append(currenciesToCheck, c.FormatExchangeCurrency(req.Pairs[i], asset.Spot).String())
		}
	} else {
		for k := range c.instrumentMap.Instruments {
			currenciesToCheck = append(currenciesToCheck, k)
		}
	}
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for x := range currenciesToCheck {
			openOrders, err := c.wsGetOpenOrders(currenciesToCheck[x])
			if err != nil {
				return nil, err
			}
			for i := range openOrders.Orders {
				orders = append(orders, order.Detail{
					Exchange:        c.Name,
					ID:              strconv.FormatInt(openOrders.Orders[i].OrderID, 10),
					Pair:            c.FormatExchangeCurrency(currency.NewPairFromString(currenciesToCheck[x]), asset.Spot),
					Side:            order.Side(openOrders.Orders[i].Side),
					Date:            time.Unix(0, openOrders.Orders[i].Timestamp),
					Status:          order.Active,
					Price:           openOrders.Orders[i].Price,
					Amount:          openOrders.Orders[i].Quantity,
					ExecutedAmount:  openOrders.Orders[i].Quantity - openOrders.Orders[i].OpenQuantity,
					RemainingAmount: openOrders.Orders[i].OpenQuantity,
				})
			}
		}
	} else {
		var instrumentsToUse []int64
		for x := range req.Pairs {
			curr := c.FormatExchangeCurrency(req.Pairs[x],
				asset.Spot).String()
			instrumentsToUse = append(instrumentsToUse,
				c.instrumentMap.LookupID(curr))
		}
		if len(instrumentsToUse) == 0 {
			instrumentsToUse = c.instrumentMap.GetInstrumentIDs()
		}

		for x := range instrumentsToUse {
			openOrders, err := c.GetOpenOrders(instrumentsToUse[x])
			if err != nil {
				return nil, err
			}
			for y := range openOrders.Orders {
				curr := c.instrumentMap.LookupInstrument(instrumentsToUse[x])
				p := currency.NewPairFromFormattedPairs(curr,
					c.GetEnabledPairs(asset.Spot),
					c.GetPairFormat(asset.Spot, true))
				orderSide := order.Side(strings.ToUpper(openOrders.Orders[y].Side))
				orderDate := time.Unix(openOrders.Orders[y].Timestamp, 0)
				orders = append(orders, order.Detail{
					ID:       strconv.FormatInt(openOrders.Orders[y].OrderID, 10),
					Amount:   openOrders.Orders[y].Quantity,
					Price:    openOrders.Orders[y].Price,
					Exchange: c.Name,
					Side:     orderSide,
					Date:     orderDate,
					Pair:     p,
				})
			}
		}
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *COINUT) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return nil, err
	}
	var allOrders []order.Detail
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for i := range req.Pairs {
			for j := int64(0); ; j += 100 {
				trades, err := c.wsGetTradeHistory(req.Pairs[i], j, 100)
				if err != nil {
					return allOrders, err
				}
				for x := range trades.Trades {
					curr := c.instrumentMap.LookupInstrument(trades.Trades[x].InstrumentID)
					allOrders = append(allOrders, order.Detail{
						Exchange:        c.Name,
						ID:              strconv.FormatInt(trades.Trades[x].OrderID, 10),
						Pair:            currency.NewPairFromString(curr),
						Side:            order.Side(trades.Trades[x].Side),
						Date:            time.Unix(0, trades.Trades[x].Timestamp),
						Status:          order.Filled,
						Price:           trades.Trades[x].Price,
						Amount:          trades.Trades[x].Quantity,
						ExecutedAmount:  trades.Trades[x].Quantity,
						RemainingAmount: trades.Trades[x].OpenQuantity,
					})
				}
				if len(trades.Trades) < 100 {
					break
				}
			}
		}
	} else {
		var instrumentsToUse []int64
		for x := range req.Pairs {
			curr := c.FormatExchangeCurrency(req.Pairs[x],
				asset.Spot).String()
			instrumentID := c.instrumentMap.LookupID(curr)
			if instrumentID > 0 {
				instrumentsToUse = append(instrumentsToUse, instrumentID)
			}
		}
		if len(instrumentsToUse) == 0 {
			instrumentsToUse = c.instrumentMap.GetInstrumentIDs()
		}
		for x := range instrumentsToUse {
			orders, err := c.GetTradeHistory(instrumentsToUse[x], -1, -1)
			if err != nil {
				return nil, err
			}
			for y := range orders.Trades {
				curr := c.instrumentMap.LookupInstrument(instrumentsToUse[x])
				p := currency.NewPairFromFormattedPairs(curr,
					c.GetEnabledPairs(asset.Spot),
					c.GetPairFormat(asset.Spot, true))
				orderSide := order.Side(strings.ToUpper(orders.Trades[y].Order.Side))
				orderDate := time.Unix(orders.Trades[y].Order.Timestamp, 0)
				allOrders = append(allOrders, order.Detail{
					ID:       strconv.FormatInt(orders.Trades[y].Order.OrderID, 10),
					Amount:   orders.Trades[y].Order.Quantity,
					Price:    orders.Trades[y].Order.Price,
					Exchange: c.Name,
					Side:     orderSide,
					Date:     orderDate,
					Pair:     p,
				})
			}
		}
	}

	order.FilterOrdersByTickRange(&allOrders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&allOrders, req.Side)
	return allOrders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (c *COINUT) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (c *COINUT) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (c *COINUT) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return c.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (c *COINUT) AuthenticateWebsocket() error {
	return c.wsAuthenticate()
}

func (c *COINUT) loadInstrumentsIfNotLoaded() error {
	if !c.instrumentMap.IsLoaded() {
		if c.Websocket.IsConnected() {
			_, err := c.WsGetInstruments()
			if err != nil {
				return err
			}
		} else {
			err := c.SeedInstruments()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (c *COINUT) ValidateCredentials() error {
	_, err := c.UpdateAccountInfo()
	return c.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (c *COINUT) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
