package coinbasepro

import (
	"errors"
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
func (c *CoinbasePro) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets default values for the exchange
func (c *CoinbasePro) SetDefaults() {
	c.Name = "CoinbasePro"
	c.Enabled = true
	c.Verbose = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresSecret = true
	c.API.CredentialsValidator.RequiresClientID = true
	c.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	c.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
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

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				KlineFetching:     true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				DepositHistory:    true,
				WithdrawalHistory: true,
				UserTradeHistory:  true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
				FiatDeposit:       true,
				FiatWithdraw:      true,
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
				CandleHistory:     true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageSequenceNumbers: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.AutoWithdrawFiatWithAPIPermission,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	c.Requester = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		SetRateLimit())

	c.API.Endpoints.URLDefault = coinbaseproAPIURL
	c.API.Endpoints.URL = c.API.Endpoints.URLDefault
	c.API.Endpoints.WebsocketURL = coinbaseproWebsocketURL
	c.Websocket = wshandler.New()
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup initialises the exchange parameters with the current configuration
func (c *CoinbasePro) Setup(exch *config.ExchangeConfig) error {
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
			DefaultURL:                       coinbaseproWebsocketURL,
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

// Start starts the coinbasepro go routine
func (c *CoinbasePro) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the coinbasepro wrapper
func (c *CoinbasePro) Run() {
	if c.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			c.Name,
			common.IsEnabled(c.Websocket.IsEnabled()),
			coinbaseproWebsocketURL)
		c.PrintEnabledPairs()
	}

	forceUpdate := false
	delim := c.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(c.CurrencyPairs.GetPairs(asset.Spot,
		true).Strings(), delim) ||
		!common.StringDataContains(c.CurrencyPairs.GetPairs(asset.Spot,
			false).Strings(), delim) {
		enabledPairs := currency.NewPairsFromStrings(
			[]string{currency.BTC.String() + delim + currency.USD.String()},
		)
		log.Warn(log.ExchangeSys,
			"Enabled pairs for CoinbasePro reset due to config upgrade, please enable the ones you would like to use again")
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
func (c *CoinbasePro) FetchTradablePairs(asset asset.Item) ([]string, error) {
	pairs, err := c.GetProducts()
	if err != nil {
		return nil, err
	}

	var products []string
	for x := range pairs {
		products = append(products, pairs[x].BaseCurrency+
			c.GetPairFormat(asset, false).Delimiter+
			pairs[x].QuoteCurrency)
	}

	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *CoinbasePro) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return c.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// coinbasepro exchange
func (c *CoinbasePro) UpdateAccountInfo() (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = c.Name
	accountBalance, err := c.GetAccounts()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for i := range accountBalance {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Available
		exchangeCurrency.Hold = accountBalance[i].Hold

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		Currencies: currencies,
	})

	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (c *CoinbasePro) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(c.Name)
	if err != nil {
		return c.UpdateAccountInfo()
	}

	return acc, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := c.GetTicker(c.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return nil, err
	}
	stats, err := c.GetStats(c.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return nil, err
	}

	tickerPrice := &ticker.Price{
		Last:        tick.Size,
		High:        stats.High,
		Low:         stats.Low,
		Bid:         tick.Bid,
		Ask:         tick.Ask,
		Volume:      tick.Volume,
		Open:        stats.Open,
		Pair:        p,
		LastUpdated: tick.Time,
	}

	err = ticker.ProcessTicker(c.Name, tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(c.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (c *CoinbasePro) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *CoinbasePro) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(c.Name, p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *CoinbasePro) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := c.GetOrderbook(c.FormatExchangeCurrency(p,
		assetType).String(), 2)
	if err != nil {
		return orderBook, err
	}

	obNew := orderbookNew.(OrderbookL1L2)

	for x := range obNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: obNew.Bids[x].Amount, Price: obNew.Bids[x].Price})
	}

	for x := range obNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: obNew.Asks[x].Amount, Price: obNew.Asks[x].Price})
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
func (c *CoinbasePro) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (c *CoinbasePro) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var response string
	var err error
	switch s.Type {
	case order.Market:
		response, err = c.PlaceMarketOrder("",
			s.Amount,
			s.Amount,
			s.Side.Lower(),
			c.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
			"")
	case order.Limit:
		response, err = c.PlaceLimitOrder("",
			s.Price,
			s.Amount,
			s.Side.Lower(),
			"",
			"",
			c.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
			"",
			false)
	default:
		err = errors.New("order type not supported")
	}
	if err != nil {
		return submitOrderResponse, err
	}
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	if response != "" {
		submitOrderResponse.OrderID = response
	}

	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *CoinbasePro) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *CoinbasePro) CancelOrder(order *order.Cancel) error {
	return c.CancelExistingOrder(order.ID)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	// CancellAllExisting orders returns a list of successful cancellations, we're only interested in failures
	_, err := c.CancelAllExistingOrders("")
	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (c *CoinbasePro) GetOrderInfo(orderID string) (order.Detail, error) {
	genOrderDetail, errGo := c.GetOrder(orderID)
	if errGo != nil {
		return order.Detail{}, fmt.Errorf("error retrieving order %s : %s", orderID, errGo)
	}
	od, errOd := time.Parse(time.RFC3339, genOrderDetail.DoneAt)
	if errOd != nil {
		return order.Detail{}, fmt.Errorf("error parsing order done at time: %s", errOd)
	}
	os, errOs := order.StringToOrderStatus(genOrderDetail.Status)
	if errOs != nil {
		return order.Detail{}, fmt.Errorf("error parsing order status: %s", errOs)
	}
	tt, errOt := order.StringToOrderType(genOrderDetail.Type)
	if errOt != nil {
		return order.Detail{}, fmt.Errorf("error parsing order type: %s", errOt)
	}
	ss, errOss := order.StringToOrderSide(genOrderDetail.Side)
	if errOss != nil {
		return order.Detail{}, fmt.Errorf("error parsing order side: %s", errOss)
	}
	response := order.Detail{
		Exchange:        c.GetName(),
		ID:              genOrderDetail.ID,
		Pair:            currency.NewPairDelimiter(genOrderDetail.ProductID, "-"),
		Side:            ss,
		Type:            tt,
		Date:            od,
		Status:          os,
		Price:           genOrderDetail.Price,
		Amount:          genOrderDetail.Size,
		ExecutedAmount:  genOrderDetail.FilledSize,
		RemainingAmount: genOrderDetail.Size - genOrderDetail.FilledSize,
		Fee:             genOrderDetail.FillFees,
	}
	fillResponse, errGF := c.GetFills(orderID, genOrderDetail.ProductID)
	if errGF != nil {
		return response, fmt.Errorf("error retrieving the order fills: %s", errGF)
	}
	for i := range fillResponse {
		trSi, errTSi := order.StringToOrderSide(fillResponse[i].Side)
		if errTSi != nil {
			return response, fmt.Errorf("error parsing order Side: %s", errTSi)
		}
		td, errTd := time.Parse(time.RFC3339, fillResponse[i].CreatedAt)
		if errTd != nil {
			return response, fmt.Errorf("error parsing trade created time: %s", errTd)
		}
		response.Trades = append(response.Trades, order.TradeHistory{
			Timestamp: td,
			TID:       string(fillResponse[i].TradeID),
			Price:     fillResponse[i].Price,
			Amount:    fillResponse[i].Size,
			Exchange:  c.GetName(),
			Type:      tt,
			Side:      trSi,
			Fee:       fillResponse[i].Fee,
		})
	}
	return response, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *CoinbasePro) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	resp, err := c.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Crypto.Address)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.ID,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	paymentMethods, err := c.GetPayMethods()
	if err != nil {
		return nil, err
	}

	selectedWithdrawalMethod := PaymentMethod{}
	for i := range paymentMethods {
		if withdrawRequest.Fiat.Bank.BankName == paymentMethods[i].Name {
			selectedWithdrawalMethod = paymentMethods[i]
			break
		}
	}
	if selectedWithdrawalMethod.ID == "" {
		return nil, fmt.Errorf("could not find payment method '%v'. Check the name via the website and try again", withdrawRequest.Fiat.Bank.BankName)
	}

	resp, err := c.WithdrawViaPaymentMethod(withdrawRequest.Amount, withdrawRequest.Currency.String(), selectedWithdrawalMethod.ID)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		Status: resp.ID,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	v, err := c.WithdrawFiatFunds(withdrawRequest)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     v.ID,
		Status: v.Status,
	}, nil
}

// GetWebsocket returns a pointer to the exchange websocket
func (c *CoinbasePro) GetWebsocket() (*wshandler.Websocket, error) {
	return c.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !c.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *CoinbasePro) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var respOrders []GeneralizedOrderResponse
	for i := range req.Pairs {
		resp, err := c.GetOrders([]string{"open", "pending", "active"},
			c.FormatExchangeCurrency(req.Pairs[i], asset.Spot).String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []order.Detail
	for i := range respOrders {
		curr := currency.NewPairDelimiter(respOrders[i].ProductID,
			c.GetPairFormat(asset.Spot, false).Delimiter)
		orderSide := order.Side(strings.ToUpper(respOrders[i].Side))
		orderType := order.Type(strings.ToUpper(respOrders[i].Type))
		orderDate, err := time.Parse(time.RFC3339, respOrders[i].CreatedAt)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				c.Name,
				"GetActiveOrders",
				respOrders[i].ID,
				respOrders[i].CreatedAt)
		}

		orders = append(orders, order.Detail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			Type:           orderType,
			Date:           orderDate,
			Side:           orderSide,
			Pair:           curr,
			Exchange:       c.Name,
		})
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *CoinbasePro) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var respOrders []GeneralizedOrderResponse
	for i := range req.Pairs {
		resp, err := c.GetOrders([]string{"done", "settled"},
			c.FormatExchangeCurrency(req.Pairs[i], asset.Spot).String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []order.Detail
	for i := range respOrders {
		curr := currency.NewPairDelimiter(respOrders[i].ProductID,
			c.GetPairFormat(asset.Spot, false).Delimiter)
		orderSide := order.Side(strings.ToUpper(respOrders[i].Side))
		orderType := order.Type(strings.ToUpper(respOrders[i].Type))
		orderDate, err := time.Parse(time.RFC3339, respOrders[i].CreatedAt)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				c.Name,
				"GetActiveOrders",
				respOrders[i].ID,
				respOrders[i].CreatedAt)
		}

		orders = append(orders, order.Detail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			Type:           orderType,
			Date:           orderDate,
			Side:           orderSide,
			Pair:           curr,
			Exchange:       c.Name,
		})
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (c *CoinbasePro) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (c *CoinbasePro) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	c.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (c *CoinbasePro) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return c.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (c *CoinbasePro) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}

// checkInterval checks allowable interval
func checkInterval(i time.Duration) (int64, error) {
	switch i.Seconds() {
	case 60:
		return 60, nil
	case 300:
		return 300, nil
	case 900:
		return 900, nil
	case 3600:
		return 3600, nil
	case 21600:
		return 21600, nil
	case 86400:
		return 86400, nil
	}
	return 0, fmt.Errorf("interval not allowed %v", i.Seconds())
}

// GetHistoricCandles returns a set of candle between two time periods for a
// designated time period
func (c *CoinbasePro) GetHistoricCandles(p currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	i, err := checkInterval(interval)
	if err != nil {
		return kline.Item{}, err
	}

	history, err := c.GetHistoricRates(c.FormatExchangeCurrency(p, a).String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		i)
	if err != nil {
		return kline.Item{}, err
	}

	var candles kline.Item
	candles.Asset = a
	candles.Exchange = c.Name
	candles.Interval = interval
	candles.Pair = p

	for x := range history {
		candles.Candles = append(candles.Candles, kline.Candle{
			Time:   time.Unix(history[x].Time, 0),
			Low:    history[x].Low,
			High:   history[x].High,
			Open:   history[x].Open,
			Close:  history[x].Close,
			Volume: history[x].Volume,
		})
	}
	return candles, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (c *CoinbasePro) ValidateCredentials() error {
	_, err := c.UpdateAccountInfo()
	return c.CheckTransientError(err)
}
