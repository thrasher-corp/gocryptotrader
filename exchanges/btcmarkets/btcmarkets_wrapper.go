package btcmarkets

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
				TickerBatching:      true,
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
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))

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

	b.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		true,
		true,
		false,
		false,
		exch.Name)

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
	delim := b.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(b.GetEnabledPairs(asset.Spot).Strings(), delim) ||
		!common.StringDataContains(b.GetAvailablePairs(asset.Spot).Strings(), delim) {
		log.Warnln(log.ExchangeSys, "Available pairs for BTC Markets reset due to config upgrade, please enable the pairs you would like again.")
		forceUpdate = true
	}
	if forceUpdate {
		enabledPairs := currency.Pairs{currency.Pair{
			Base:      currency.BTC.Lower(),
			Quote:     currency.AUD.Lower(),
			Delimiter: delim,
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
	if a != asset.Spot {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, b.Name)
	}
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
func (b *BTCMarkets) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	allPairs := b.GetEnabledPairs(assetType)
	tickers, err := b.GetTickers(allPairs.Slice())
	if err != nil {
		return nil, err
	}
	for x := range tickers {
		var resp ticker.Price
		resp.Pair = currency.NewPairFromString(tickers[x].MarketID)
		resp.Last = tickers[x].LastPrice
		resp.High = tickers[x].High24h
		resp.Low = tickers[x].Low24h
		resp.Bid = tickers[x].BestBID
		resp.Ask = tickers[x].BestAsk
		resp.Volume = tickers[x].Volume
		resp.LastUpdated = time.Now()
		err = ticker.ProcessTicker(b.Name, &resp, assetType)
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *BTCMarkets) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *BTCMarkets) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTCMarkets) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
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
	}
	orderBook.Pair = p
	orderBook.ExchangeName = b.Name
	orderBook.AssetType = assetType
	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}
	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (b *BTCMarkets) UpdateAccountInfo() (account.Holdings, error) {
	var resp account.Holdings
	data, err := b.GetAccountBalance()
	if err != nil {
		return resp, err
	}
	var acc account.SubAccount
	for key := range data {
		c := currency.NewCode(data[key].AssetName)
		hold := data[key].Locked
		total := data[key].Balance
		acc.Currencies = append(acc.Currencies,
			account.Balance{CurrencyName: c,
				TotalValue: total,
				Hold:       hold})
	}
	resp.Accounts = append(resp.Accounts, acc)
	resp.Exchange = b.Name

	err = account.Process(&resp)
	if err != nil {
		return account.Holdings{}, err
	}

	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *BTCMarkets) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name)
	if err != nil {
		return b.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTCMarkets) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
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

	if s.Side == order.Sell {
		s.Side = order.Ask
	}
	if s.Side == order.Buy {
		s.Side = order.Bid
	}

	tempResp, err := b.NewOrder(b.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
		s.Price,
		s.Amount,
		s.Type.String(),
		s.Side.String(),
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
	_, err := b.RemoveOrder(o.ID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCMarkets) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	tempMap := make(map[string]string)
	var orderIDs []string
	orders, err := b.GetOrders("", -1, -1, -1, true)
	if err != nil {
		return resp, err
	}
	for x := range orders {
		orderIDs = append(orderIDs, orders[x].OrderID)
	}
	splitOrders := common.SplitStringSliceByLimit(orderIDs, 20)
	for z := range splitOrders {
		tempResp, err := b.CancelBatchOrders(splitOrders[z])
		if err != nil {
			return resp, err
		}
		for y := range tempResp.CancelOrders {
			tempMap[tempResp.CancelOrders[y].OrderID] = "Success"
		}
		for z := range tempResp.UnprocessedRequests {
			tempMap[tempResp.UnprocessedRequests[z].RequestID] = "Cancellation Failed"
		}
	}
	resp.Status = tempMap
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
	resp.Pair = currency.NewPairFromString(o.MarketID)
	resp.Price = o.Price
	resp.Date = o.CreationTime
	resp.ExecutedAmount = o.Amount - o.OpenAmount
	resp.Side = order.Bid
	if o.Side == ask {
		resp.Side = order.Ask
	}
	switch o.Type {
	case limit:
		resp.Type = order.Limit
	case market:
		resp.Type = order.Market
	case stopLimit:
		resp.Type = order.Stop
	case stop:
		resp.Type = order.Stop
	case takeProfit:
		resp.Type = order.ImmediateOrCancel
	default:
		resp.Type = order.UnknownType
	}
	resp.RemainingAmount = o.OpenAmount
	switch o.Status {
	case orderAccepted:
		resp.Status = order.Active
	case orderPlaced:
		resp.Status = order.Active
	case orderPartiallyMatched:
		resp.Status = order.PartiallyFilled
	case orderFullyMatched:
		resp.Status = order.Filled
	case orderCancelled:
		resp.Status = order.Cancelled
	case orderPartiallyCancelled:
		resp.Status = order.PartiallyCancelled
	case orderFailed:
		resp.Status = order.Rejected
	default:
		resp.Status = order.UnknownStatus
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCMarkets) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	temp, err := b.FetchDepositAddress(strings.ToUpper(cryptocurrency.String()), -1, -1, -1)
	if err != nil {
		return "", err
	}
	return temp.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	a, err := b.RequestWithdraw(withdrawRequest.Currency.String(),
		withdrawRequest.Amount,
		withdrawRequest.Crypto.Address,
		"",
		"",
		"",
		"")
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     a.ID,
		Status: a.Status,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if withdrawRequest.Currency != currency.AUD {
		return nil, errors.New("only aud is supported for withdrawals")
	}
	a, err := b.RequestWithdraw(withdrawRequest.Currency.String(),
		withdrawRequest.Amount,
		"",
		withdrawRequest.Fiat.Bank.AccountName,
		withdrawRequest.Fiat.Bank.AccountNumber,
		withdrawRequest.Fiat.Bank.BSBNumber,
		withdrawRequest.Fiat.Bank.BankName)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     a.ID,
		Status: a.Status,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
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
	if len(req.Pairs) == 0 {
		allPairs := b.GetEnabledPairs(asset.Spot)
		for a := range allPairs {
			req.Pairs = append(req.Pairs,
				allPairs[a])
		}
	}

	var resp []order.Detail
	for x := range req.Pairs {
		tempData, err := b.GetOrders(b.FormatExchangeCurrency(req.Pairs[x], asset.Spot).String(), -1, -1, -1, true)
		if err != nil {
			return resp, err
		}
		for y := range tempData {
			var tempResp order.Detail
			tempResp.Exchange = b.Name
			tempResp.Pair = req.Pairs[x]
			tempResp.ID = tempData[y].OrderID
			tempResp.Side = order.Bid
			if tempData[y].Side == ask {
				tempResp.Side = order.Ask
			}
			tempResp.Date = tempData[y].CreationTime

			switch tempData[y].Type {
			case limit:
				tempResp.Type = order.Limit
			case market:
				tempResp.Type = order.Market
			default:
				log.Errorf(log.ExchangeSys,
					"%s unknown order type %s getting order",
					b.Name,
					tempData[y].Type)
				tempResp.Type = order.UnknownType
			}
			switch tempData[y].Status {
			case orderAccepted:
				tempResp.Status = order.Active
			case orderPlaced:
				tempResp.Status = order.Active
			case orderPartiallyMatched:
				tempResp.Status = order.PartiallyFilled
			default:
				log.Errorf(log.ExchangeSys,
					"%s unexpected status %s on order %v",
					b.Name,
					tempData[y].Status,
					tempData[y].OrderID)
				tempResp.Status = order.UnknownStatus
			}
			tempResp.Price = tempData[y].Price
			tempResp.Amount = tempData[y].Amount
			tempResp.ExecutedAmount = tempData[y].Amount - tempData[y].OpenAmount
			tempResp.RemainingAmount = tempData[y].OpenAmount
			resp = append(resp, tempResp)
		}
	}
	order.FilterOrdersByType(&resp, req.Type)
	order.FilterOrdersByTickRange(&resp, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&resp, req.Side)
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCMarkets) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	var tempResp order.Detail
	var tempArray []string
	if len(req.Pairs) == 0 {
		orders, err := b.GetOrders("", -1, -1, -1, false)
		if err != nil {
			return resp, err
		}
		for x := range orders {
			tempArray = append(tempArray, orders[x].OrderID)
		}
	}
	for y := range req.Pairs {
		orders, err := b.GetOrders(b.FormatExchangeCurrency(req.Pairs[y], asset.Spot).String(), -1, -1, -1, false)
		if err != nil {
			return resp, err
		}
		for z := range orders {
			tempArray = append(tempArray, orders[z].OrderID)
		}
	}
	splitOrders := common.SplitStringSliceByLimit(tempArray, 50)
	for x := range splitOrders {
		tempData, err := b.GetBatchTrades(splitOrders[x])
		if err != nil {
			return resp, err
		}
		for c := range tempData.Orders {
			switch tempData.Orders[c].Status {
			case orderFailed:
				tempResp.Status = order.Rejected
			case orderPartiallyCancelled:
				tempResp.Status = order.PartiallyCancelled
			case orderCancelled:
				tempResp.Status = order.Cancelled
			case orderFullyMatched:
				tempResp.Status = order.Filled
			case orderPartiallyMatched:
				continue
			case orderPlaced:
				continue
			case orderAccepted:
				continue
			}
			tempResp.Exchange = b.Name
			tempResp.Pair = currency.NewPairFromString(tempData.Orders[c].MarketID)
			tempResp.Side = order.Bid
			if tempData.Orders[c].Side == ask {
				tempResp.Side = order.Ask
			}
			tempResp.ID = tempData.Orders[c].OrderID
			tempResp.Date = tempData.Orders[c].CreationTime
			tempResp.Price = tempData.Orders[c].Price
			tempResp.ExecutedAmount = tempData.Orders[c].Amount
			resp = append(resp, tempResp)
		}
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

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *BTCMarkets) ValidateCredentials() error {
	_, err := b.UpdateAccountInfo()
	if err != nil {
		if b.CheckTransientError(err) == nil {
			return nil
		}
		// Check for specific auth errors; all other errors can be disregarded
		// as this does not affect authenticated requests.
		if strings.Contains(err.Error(), "InvalidAPIKey") ||
			strings.Contains(err.Error(), "InvalidAuthTimestamp") ||
			strings.Contains(err.Error(), "InvalidAuthSignature") ||
			strings.Contains(err.Error(), "InsufficientAPIPermission") {
			return err
		}
	}

	return nil
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *BTCMarkets) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return b.GetMarketCandles(pair.String(), interval, start, end, -1, -1, 0)
}
