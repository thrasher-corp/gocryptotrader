package kraken

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
func (k *Kraken) GetDefaultConfig() (*config.ExchangeConfig, error) {
	k.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = k.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = k.BaseCurrencies

	err := k.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if k.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = k.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default settings
func (k *Kraken) SetDefaults() {
	k.Name = "Kraken"
	k.Enabled = true
	k.Verbose = true
	k.API.CredentialsValidator.RequiresKey = true
	k.API.CredentialsValidator.RequiresSecret = true
	k.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	k.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},

		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Separator: ",",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
			Separator: ",",
		},
	}

	k.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				SubmitOrder:         true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				FiatDeposit:         true,
				FiatWithdraw:        true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:     true,
				TradeFetching:      true,
				KlineFetching:      true,
				OrderbookFetching:  true,
				Subscribe:          true,
				Unsubscribe:        true,
				MessageCorrelation: true,
				SubmitOrder:        true,
				CancelOrder:        true,
				CancelOrders:       true,
				GetOrders:          true,
				GetOrder:           true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawCryptoWith2FA |
				exchange.AutoWithdrawFiatWithSetup |
				exchange.WithdrawFiatWith2FA,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	k.Requester = request.New(k.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(krakenRateInterval, krakenRequestRate)))

	k.API.Endpoints.URLDefault = krakenAPIURL
	k.API.Endpoints.URL = k.API.Endpoints.URLDefault
	k.Websocket = wshandler.New()
	k.API.Endpoints.WebsocketURL = krakenWSURL
	k.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	k.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	k.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets current exchange configuration
func (k *Kraken) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		k.SetEnabled(false)
		return nil
	}

	err := k.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = k.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       krakenWSURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        k.WsConnect,
			Subscriber:                       k.Subscribe,
			UnSubscriber:                     k.Unsubscribe,
			Features:                         &k.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}

	k.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         k.Name,
		URL:                  k.Websocket.GetWebsocketURL(),
		ProxyURL:             k.Websocket.GetProxyAddress(),
		Verbose:              k.Verbose,
		RateLimit:            krakenWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	k.AuthenticatedWebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         k.Name,
		URL:                  krakenAuthWSURL,
		ProxyURL:             k.Websocket.GetProxyAddress(),
		Verbose:              k.Verbose,
		RateLimit:            krakenWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	k.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		true,
		true,
		false,
		false,
		exch.Name)
	return nil
}

// Start starts the Kraken go routine
func (k *Kraken) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		k.Run()
		wg.Done()
	}()
}

// Run implements the Kraken wrapper
func (k *Kraken) Run() {
	if k.Verbose {
		k.PrintEnabledPairs()
	}

	forceUpdate := false
	delim := k.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(k.GetEnabledPairs(asset.Spot).Strings(), delim) ||
		!common.StringDataContains(k.GetAvailablePairs(asset.Spot).Strings(), delim) {
		enabledPairs := currency.NewPairsFromStrings(
			[]string{currency.XBT.String() + delim + currency.USD.String()},
		)
		log.Warn(log.ExchangeSys, "Available pairs for Kraken reset due to config upgrade, please enable the ones you would like again")
		forceUpdate = true

		err := k.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				k.Name,
				err)
		}
	}

	if !k.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := k.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			k.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (k *Kraken) FetchTradablePairs(asset asset.Item) ([]string, error) {
	pairs, err := k.GetAssetPairs()
	if err != nil {
		return nil, err
	}

	var products []string
	for i := range pairs {
		v := pairs[i]
		if strings.Contains(v.Altname, ".d") {
			continue
		}
		if v.Base[0] == 'X' {
			if len(v.Base) > 3 {
				v.Base = v.Base[1:]
			}
		}
		if v.Quote[0] == 'Z' || v.Quote[0] == 'X' {
			v.Quote = v.Quote[1:]
		}
		products = append(products, v.Base+
			k.GetPairFormat(asset, false).Delimiter+
			v.Quote)
	}
	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (k *Kraken) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := k.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return k.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (k *Kraken) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	pairs := k.GetEnabledPairs(assetType)
	pairsCollated, err := k.FormatExchangeCurrencies(pairs, assetType)
	if err != nil {
		return tickerPrice, err
	}
	tickers, err := k.GetTickers(pairsCollated)
	if err != nil {
		return tickerPrice, err
	}

	for i := range pairs {
		for c, t := range tickers {
			pairFmt := k.FormatExchangeCurrency(pairs[i], assetType).String()
			if !strings.EqualFold(pairFmt, c) {
				altCurrency, ok := assetPairMap[c]
				if !ok {
					continue
				}
				if !strings.EqualFold(pairFmt, altCurrency) {
					continue
				}
			}

			tickerPrice = &ticker.Price{
				Last:   t.Last,
				High:   t.High,
				Low:    t.Low,
				Bid:    t.Bid,
				Ask:    t.Ask,
				Volume: t.Volume,
				Open:   t.Open,
				Pair:   pairs[i],
			}
			err = ticker.ProcessTicker(k.Name, tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}
	}
	return ticker.GetTicker(k.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (k *Kraken) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(k.Name, p, assetType)
	if err != nil {
		return k.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (k *Kraken) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(k.Name, p, assetType)
	if err != nil {
		return k.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (k *Kraken) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := k.GetDepth(k.FormatExchangeCurrency(p,
		assetType).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = k.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(k.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Kraken exchange - to-do
func (k *Kraken) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
	info.Exchange = k.Name

	bal, err := k.GetBalance()
	if err != nil {
		return info, err
	}

	var balances []account.Balance
	for key := range bal {
		balances = append(balances, account.Balance{
			CurrencyName: currency.NewCode(key),
			TotalValue:   bal[key],
		})
	}

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
func (k *Kraken) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(k.Name)
	if err != nil {
		return k.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (k *Kraken) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (k *Kraken) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (k *Kraken) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	err := s.Validate()
	if err != nil {
		return submitOrderResponse, err
	}

	if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var resp string
		resp, err = k.wsAddOrder(&WsAddOrderRequest{
			OrderType: s.Type.String(),
			OrderSide: s.Side.String(),
			Pair:      s.Pair.String(),
			Price:     s.Price,
			Volume:    s.Amount,
		})
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = resp
		submitOrderResponse.IsOrderPlaced = true
	} else {
		var response AddOrderResponse
		response, err = k.AddOrder(s.Pair.String(),
			s.Side.String(),
			s.Type.String(),
			s.Amount,
			s.Price,
			0,
			0,
			&AddOrderOptions{})
		if err != nil {
			return submitOrderResponse, err
		}
		if len(response.TransactionIds) > 0 {
			submitOrderResponse.OrderID = strings.Join(response.TransactionIds, ", ")
		}
	}
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	submitOrderResponse.IsOrderPlaced = true
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (k *Kraken) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (k *Kraken) CancelOrder(order *order.Cancel) error {
	if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		return k.wsCancelOrders([]string{order.ID})
	}
	_, err := k.CancelExistingOrder(order.ID)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (k *Kraken) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	var emptyOrderOptions OrderInfoOptions
	openOrders, err := k.GetOpenOrders(emptyOrderOptions)
	if err != nil {
		return cancelAllOrdersResponse, err
	}
	for orderID := range openOrders.Open {
		var err error
		if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			err = k.wsCancelOrders([]string{orderID})
		} else {
			_, err = k.CancelExistingOrder(orderID)
		}
		if err != nil {
			cancelAllOrdersResponse.Status[orderID] = err.Error()
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (k *Kraken) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	var emptyOrderOptions OrderInfoOptions
	openOrders, err := k.GetOpenOrders(emptyOrderOptions)
	if err != nil {
		return orderDetail, err
	}
	if orderInfo, ok := openOrders.Open[orderID]; ok {
		var trades []order.TradeHistory
		for i := range orderInfo.Trades {
			trades = append(trades, order.TradeHistory{
				TID: orderInfo.Trades[i],
			})
		}
		firstNum, decNum, err := convert.SplitFloatDecimals(orderInfo.StartTime)
		if err != nil {
			return orderDetail, err
		}
		side, err := order.StringToOrderSide(orderInfo.Description.Type)
		if err != nil {
			return orderDetail, err
		}
		status, err := order.StringToOrderStatus(orderInfo.Status)
		if err != nil {
			return orderDetail, err
		}
		oType, err := order.StringToOrderType(orderInfo.Description.OrderType)
		if err != nil {
			return orderDetail, err
		}

		orderDetail = order.Detail{
			Exchange:        k.Name,
			ID:              orderID,
			Pair:            currency.NewPairFromString(orderInfo.Description.Pair),
			Side:            side,
			Type:            oType,
			Date:            time.Unix(firstNum, decNum),
			Status:          status,
			Price:           orderInfo.Price,
			Amount:          orderInfo.Volume,
			ExecutedAmount:  orderInfo.VolumeExecuted,
			RemainingAmount: orderInfo.Volume - orderInfo.VolumeExecuted,
			Fee:             orderInfo.Fee,
			Trades:          trades,
		}
	} else {
		return orderDetail, errors.New(k.Name + " - Order ID not found: " + orderID)
	}

	return orderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (k *Kraken) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	methods, err := k.GetDepositMethods(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	var method string
	for _, m := range methods {
		method = m.Method
	}

	if method == "" {
		return "", errors.New("method not found")
	}

	return k.GetCryptoDepositAddress(method, cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal
// Populate exchange.WithdrawRequest.TradePassword with withdrawal key name, as set up on your account
func (k *Kraken) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	v, err := k.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.TradePassword, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	v, err := k.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.TradePassword, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	v, err := k.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.TradePassword, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// GetWebsocket returns a pointer to the exchange websocket
func (k *Kraken) GetWebsocket() (*wshandler.Websocket, error) {
	return k.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (k *Kraken) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !k.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return k.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (k *Kraken) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	resp, err := k.GetOpenOrders(OrderInfoOptions{})
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp.Open {
		symbol := currency.NewPairFromString(resp.Open[i].Description.Pair)
		orderDate := time.Unix(int64(resp.Open[i].StartTime), 0)
		side := order.Side(strings.ToUpper(resp.Open[i].Description.Type))
		orderType := order.Type(strings.ToUpper(resp.Open[i].Description.OrderType))

		orders = append(orders, order.Detail{
			ID:              i,
			Amount:          resp.Open[i].Volume,
			RemainingAmount: (resp.Open[i].Volume - resp.Open[i].VolumeExecuted),
			ExecutedAmount:  resp.Open[i].VolumeExecuted,
			Exchange:        k.Name,
			Date:            orderDate,
			Price:           resp.Open[i].Description.Price,
			Side:            side,
			Type:            orderType,
			Pair:            symbol,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (k *Kraken) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	req := GetClosedOrdersOptions{}
	if getOrdersRequest.StartTicks.Unix() > 0 {
		req.Start = strconv.FormatInt(getOrdersRequest.StartTicks.Unix(), 10)
	}
	if getOrdersRequest.EndTicks.Unix() > 0 {
		req.End = strconv.FormatInt(getOrdersRequest.EndTicks.Unix(), 10)
	}

	resp, err := k.GetClosedOrders(req)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp.Closed {
		symbol := currency.NewPairFromString(resp.Closed[i].Description.Pair)
		orderDate := time.Unix(int64(resp.Closed[i].StartTime), 0)
		side := order.Side(strings.ToUpper(resp.Closed[i].Description.Type))
		orderType := order.Type(strings.ToUpper(resp.Closed[i].Description.OrderType))

		orders = append(orders, order.Detail{
			ID:              i,
			Amount:          resp.Closed[i].Volume,
			RemainingAmount: (resp.Closed[i].Volume - resp.Closed[i].VolumeExecuted),
			ExecutedAmount:  resp.Closed[i].VolumeExecuted,
			Exchange:        k.Name,
			Date:            orderDate,
			Price:           resp.Closed[i].Description.Price,
			Side:            side,
			Type:            orderType,
			Pair:            symbol,
		})
	}

	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)
	order.FilterOrdersByCurrencies(&orders, getOrdersRequest.Pairs)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (k *Kraken) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	k.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (k *Kraken) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	k.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (k *Kraken) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return k.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (k *Kraken) AuthenticateWebsocket() error {
	resp, err := k.GetWebsocketToken()
	if resp != "" {
		authToken = resp
	}
	return err
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (k *Kraken) ValidateCredentials() error {
	_, err := k.UpdateAccountInfo()
	return k.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (k *Kraken) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
