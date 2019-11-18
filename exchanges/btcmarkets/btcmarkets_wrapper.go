package btcmarkets

import (
	"errors"
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
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (b *BTCMarkets) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets basic defaults
func (b *BTCMarkets) SetDefaults() {
	b.Name = "BTC Markets"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.Endpoints.URLDefault = btcMarketsAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault

	b.CurrencyPairs = currency.PairsManager{
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

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				SubmitOrder:         true,
				UserTradeHistory:    true,
				CryptoWithdrawal:    true,
				FiatWithdraw:        true,
				TradeFee:            true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				AccountInfo:            true,
				Subscribe:              true,
				AuthenticatedEndpoints: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second*10, btcmarketsAuthLimit),
		request.NewRateLimit(time.Second*10, btcmarketsUnauthLimit),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.WebsocketURL = btcMarketsWSURL
	b.Websocket = wshandler.New()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in an exchange configuration and sets all parameters
func (b *BTCMarkets) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       btcMarketsWSURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        b.WsConnect,
			Subscriber:                       b.Subscribe,
			Features:                         &b.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}

	b.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         b.Name,
		URL:                  b.Websocket.GetWebsocketURL(),
		ProxyURL:             b.Websocket.GetProxyAddress(),
		Verbose:              b.Verbose,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	return nil
}

// Start starts the BTC Markets go routine
func (b *BTCMarkets) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTC Markets wrapper
func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s (url: %s).\n",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()),
			btcMarketsWSURL)
		b.PrintEnabledPairs()
	}
	forceUpdate := false
	if !common.StringDataContains(b.GetEnabledPairs(asset.Spot).Strings(), "-") ||
		!common.StringDataContains(b.GetAvailablePairs(asset.Spot).Strings(), "-") {
		log.Warnln(log.ExchangeSys, "Available pairs for BTC Markets reset due to config upgrade, please enable the pairs you would like again.")
		forceUpdate = true
	}
	if forceUpdate {
		enabledPairs := currency.Pairs{currency.Pair{
			Base:      currency.BTC.Lower(),
			Quote:     currency.AUD.Lower(),
			Delimiter: "-",
		},
		}
		err := b.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s Failed to update enabled currencies.\n",
				b.Name)
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := b.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *BTCMarkets) FetchTradablePairs(a asset.Item) ([]string, error) {
	markets, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range markets {
		pairs = append(pairs, markets[x].MarketID)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *BTCMarkets) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTCMarkets) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var resp ticker.Price
	allPairs, err := b.GetMarkets()
	if err != nil {
		return resp, err
	}
	for x := range allPairs {
		tick, err := b.GetTicker(allPairs[x].MarketID)
		if err != nil {
			return resp, err
		}
		resp.Pair = currency.NewPairFromString(allPairs[x].MarketID)
		resp.Last = tick.LastPrice
		resp.High = tick.High24h
		resp.Low = tick.Low24h
		resp.Bid = tick.BestBID
		resp.Ask = tick.BestAsk
		resp.Volume = tick.Volume
		resp.LastUpdated = time.Now()
		err = ticker.ProcessTicker(b.Name, &resp, assetType)
		if err != nil {
			return resp, err
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *BTCMarkets) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *BTCMarkets) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCMarkets) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	var orderBook orderbook.Base
	tempResp, err := b.GetOrderbook(b.FormatExchangeCurrency(p, assetType).String(), 2)
	if err != nil {
		return orderBook, err
	}

	for x := range tempResp.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Amount: tempResp.Bids[x].Volume,
			Price:  tempResp.Bids[x].Price})
	}

	for y := range tempResp.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Amount: tempResp.Asks[y].Volume,
			Price:  tempResp.Asks[y].Price})
		orderBook.Pair = p
		orderBook.ExchangeName = b.Name
		orderBook.AssetType = assetType

		err = orderBook.Process()
		if err != nil {
			return orderBook, err
		}
	}
	return orderbook.Get(b.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies
func (b *BTCMarkets) GetAccountInfo() (exchange.AccountInfo, error) {
	var resp exchange.AccountInfo
	data, err := b.GetAccountBalance()
	if err != nil {
		return resp, err
	}
	var account exchange.Account
	for key := range data {
		c := currency.NewCode(data[key].AssetName)
		hold := data[key].Locked
		total := data[key].Balance
		account.Currencies = append(account.Currencies,
			exchange.AccountCurrencyInfo{CurrencyName: c,
				TotalValue: total,
				Hold:       hold})
	}
	resp.Accounts = append(resp.Accounts, account)
	resp.Exchange = b.Name
	return resp, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCMarkets) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTCMarkets) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTCMarkets) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	if s.OrderSide == order.Sell {
		s.OrderSide = order.Ask
	}
	if s.OrderSide == order.Buy {
		s.OrderSide = order.Bid
	}

	tempResp, err := b.NewOrder(b.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
		s.Amount,
		s.Price,
		s.OrderSide.String(),
		s.OrderType.String(),
		s.TriggerPrice,
		s.TargetAmount,
		"",
		false,
		"",
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
func (b *BTCMarkets) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCMarkets) CancelOrder(o *order.Cancel) error {
	_, err := b.RemoveOrder(o.OrderID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCMarkets) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	tempMap := make(map[string]string)
	orders, err := b.GetOrders("")
	if err != nil {
		return resp, err
	}
	for x := range orders {
		_, err := b.RemoveOrder(orders[x].OrderID)
		if err != nil {
			tempMap[orders[x].OrderID] = "Failed"
		} else {
			tempMap[orders[x].OrderID] = "Success"
		}
	}
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTCMarkets) GetOrderInfo(orderID string) (order.Detail, error) {
	var resp order.Detail
	o, err := b.FetchOrder(orderID)
	if err != nil {
		return resp, err
	}
	resp.Exchange = b.Name
	resp.ID = orderID
	resp.CurrencyPair = currency.NewPairFromString(o.MarketID)
	resp.Price = o.Price
	resp.OrderDate = o.CreationTime
	resp.ExecutedAmount = o.Amount - o.OpenAmount
	resp.OrderSide = order.Bid
	if o.Side == "Ask" {
		resp.OrderSide = order.Ask
	}
	switch o.Type {
	case limit:
		resp.OrderType = order.Limit
	case market:
		resp.OrderType = order.Market
	case stopLimit:
		resp.OrderType = order.Stop
	case stop:
		resp.OrderType = order.Stop
	case takeProfit:
		resp.OrderType = order.ImmediateOrCancel
	default:
		resp.OrderType = order.Unknown
	}
	resp.RemainingAmount = o.OpenAmount
	switch o.Status {
	case accepted:
		resp.Status = order.Active
	case placed:
		resp.Status = order.Active
	case partiallyMatched:
		resp.Status = order.PartiallyFilled
	case fullyMatched:
		resp.Status = order.Filled
	case cancelled:
		resp.Status = order.Cancelled
	case partiallyCancelled:
		resp.Status = order.PartiallyFilled
	case failed:
		resp.Status = order.Rejected
	default:
		resp.Status = order.UnknownStatus
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCMarkets) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	a, err := b.RequestWithdraw(withdrawRequest.Currency.String(), withdrawRequest.Amount,
		withdrawRequest.Address, "",
		"",
		"",
		"")
	if err != nil {
		return "", err
	}
	return a.Status, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	if withdrawRequest.Currency != currency.AUD {
		return "", errors.New("only AUD is supported for withdrawals")
	}
	a, err := b.RequestWithdraw(withdrawRequest.GenericWithdrawRequestInfo.Currency.String(),
		withdrawRequest.GenericWithdrawRequestInfo.Amount,
		"",
		withdrawRequest.BankAccountName,
		withdrawRequest.BankAccountNumber,
		"",
		withdrawRequest.BankName)
	if err != nil {
		return "", err
	}
	return a.Status, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTCMarkets) GetWebsocket() (*wshandler.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTCMarkets) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTCMarkets) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	var tempResp order.Detail
	var tempData []OrderData
	if len(req.Currencies) == 0 {
		allPairs, err := b.GetMarkets()
		if err != nil {
			return resp, err
		}
		for a := range allPairs {
			req.Currencies = append(req.Currencies,
				currency.NewPairFromString(allPairs[a].MarketID))
		}
	}
	var err error
	for x := range req.Currencies {
		tempData, err = b.GetOrders(b.FormatExchangeCurrency(req.Currencies[x], asset.Spot).String())
		if err != nil {
			return resp, err
		}
		for y := range tempData {
			tempResp.Exchange = b.Name
			tempResp.CurrencyPair = req.Currencies[x]
			tempResp.OrderSide = order.Bid
			if tempData[y].Side == "Ask" {
				tempResp.OrderSide = order.Ask
			}
			tempResp.OrderDate = tempData[y].CreationTime
			switch tempData[y].Status {
			case accepted:
				tempResp.Status = order.Active
			case placed:
				tempResp.Status = order.Active
			case partiallyMatched:
				tempResp.Status = order.PartiallyFilled
			case fullyMatched:
				tempResp.Status = order.Filled
			case cancelled:
				tempResp.Status = order.Cancelled
			case partiallyCancelled:
				tempResp.Status = order.PartiallyFilled
			case failed:
				tempResp.Status = order.Rejected
			}
			tempResp.Price = tempData[y].Price
			tempResp.Amount = tempData[y].Amount
			tempResp.ExecutedAmount = tempData[y].Amount - tempData[y].OpenAmount
			tempResp.RemainingAmount = tempData[y].OpenAmount
			resp = append(resp, tempResp)
		}
		return resp, nil
	}

	order.FilterOrdersByType(&resp, req.OrderType)
	order.FilterOrdersByTickRange(&resp, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&resp, req.OrderSide)
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCMarkets) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	var tempResp order.Detail
	var tempArray []string
	if len(req.Currencies) == 0 {
		orders, err := b.GetOrders("")
		if err != nil {
			return resp, err
		}
		for x := range orders {
			tempArray = append(tempArray, orders[x].OrderID)
		}
	}
	for y := range req.Currencies {
		orders, err := b.GetOrders(b.FormatExchangeCurrency(req.Currencies[y], asset.Spot).String())
		if err != nil {
			return resp, err
		}
		for z := range orders {
			tempArray = append(tempArray, orders[z].OrderID)
		}
	}
	tempData, err := b.GetBatchTrades(tempArray)
	if err != nil {
		return resp, err
	}
	for c := range tempData.Orders {
		tempResp.Exchange = b.Name
		tempResp.CurrencyPair = currency.NewPairFromString(tempData.Orders[c].MarketID)
		tempResp.OrderSide = order.Bid
		if tempData.Orders[c].Side == "Ask" {
			tempResp.OrderSide = order.Ask
		}
		tempResp.OrderDate = tempData.Orders[c].CreationTime
		tempResp.Status = order.Filled
		tempResp.Price = tempData.Orders[c].Price
		tempResp.ExecutedAmount = tempData.Orders[c].Amount
		resp = append(resp, tempResp)
	}
	return resp, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *BTCMarkets) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *BTCMarkets) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (b *BTCMarkets) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *BTCMarkets) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
