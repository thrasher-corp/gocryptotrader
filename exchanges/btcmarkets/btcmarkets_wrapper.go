package btcmarkets

import (
	"errors"
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
		b.PrintEnabledPairs()
	}

	forceUpdate := false
	if !common.StringDataContains(b.GetEnabledPairs(asset.Spot).Strings(), "-") ||
		!common.StringDataContains(b.GetAvailablePairs(asset.Spot).Strings(), "-") {
		enabledPairs := []string{"BTC-AUD"}
		log.Warnln(log.ExchangeSys, "Available pairs for BTC Markets reset due to config upgrade, please enable the pairs you would like again.")
		forceUpdate = true

		err := b.UpdatePairs(currency.NewPairsFromStrings(enabledPairs), asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update currencies. Err: %s", b.Name, err)
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := b.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *BTCMarkets) FetchTradablePairs(asset asset.Item) ([]string, error) {
	markets, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range markets {
		pairs = append(pairs, fmt.Sprintf("%v%v%v", markets[x].Instrument, b.GetPairFormat(asset, false).Delimiter, markets[x].Currency))
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
	var tickerPrice ticker.Price
	tick, err := b.GetTicker(p.Base.String(), p.Quote.String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice = ticker.Price{
		Last:        tick.LastPrice,
		High:        tick.High24h,
		Low:         tick.Low24h,
		Bid:         tick.BestBid,
		Ask:         tick.BestAsk,
		Volume:      tick.Volume24h,
		Pair:        p,
		LastUpdated: time.Unix(tick.Timestamp, 0),
	}

	err = ticker.ProcessTicker(b.Name, &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
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
	orderbookNew, err := b.GetOrderbook(p.Base.String(),
		p.Quote.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x][1],
			Price:  orderbookNew.Bids[x][0],
		})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x][1],
			Price:  orderbookNew.Asks[x][0],
		})
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

// GetAccountInfo retrieves balances for all enabled currencies for the
// BTCMarkets exchange
func (b *BTCMarkets) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.Name

	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency exchange.AccountCurrencyInfo
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].PendingFunds

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, exchange.Account{
		Currencies: currencies,
	})

	return response, nil
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
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	if strings.EqualFold(s.OrderSide.String(), order.Sell.String()) {
		s.OrderSide = order.Ask
	}
	if strings.EqualFold(s.OrderSide.String(), order.Buy.String()) {
		s.OrderSide = order.Bid
	}

	response, err := b.NewOrder(s.Pair.Base.Upper().String(),
		s.Pair.Quote.Upper().String(),
		s.Price,
		s.Amount,
		s.OrderSide.String(),
		s.OrderType.String(),
		s.ClientID)

	if response > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response, 10)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTCMarkets) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTCMarkets) CancelOrder(order *order.Cancel) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder([]int64{orderIDInt})
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *BTCMarkets) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := b.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var orderList []int64
	for i := range openOrders {
		orderList = append(orderList, openOrders[i].ID)
	}

	if len(orderList) > 0 {
		orders, err := b.CancelExistingOrder(orderList)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for i := range orders {
			if !orders[i].Success {
				cancelAllOrdersResponse.Status[strconv.FormatInt(orders[i].ID, 10)] = orders[i].ErrorMessage
			}
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTCMarkets) GetOrderInfo(orderID string) (order.Detail, error) {
	var OrderDetail order.Detail

	o, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return OrderDetail, err
	}

	orders, err := b.GetOrderDetail([]int64{o})
	if err != nil {
		return OrderDetail, err
	}

	if len(orders) > 1 {
		return OrderDetail, errors.New("too many orders returned")
	}

	if len(orders) == 0 {
		return OrderDetail, errors.New("no orders found")
	}

	for i := range orders {
		var side order.Side
		if strings.EqualFold(orders[i].OrderSide, order.Ask.String()) {
			side = order.Sell
		} else if strings.EqualFold(orders[i].OrderSide, order.Bid.String()) {
			side = order.Buy
		}
		orderDate := time.Unix(int64(orders[i].CreationTime), 0)
		orderType := order.Type(strings.ToUpper(orders[i].OrderType))

		OrderDetail.Amount = orders[i].Volume
		OrderDetail.OrderDate = orderDate
		OrderDetail.Exchange = b.Name
		OrderDetail.ID = strconv.FormatInt(orders[i].ID, 10)
		OrderDetail.RemainingAmount = orders[i].OpenVolume
		OrderDetail.OrderSide = side
		OrderDetail.OrderType = orderType
		OrderDetail.Price = orders[i].Price
		OrderDetail.Status = order.Status(orders[i].Status)
		OrderDetail.CurrencyPair = currency.NewPairWithDelimiter(orders[i].Instrument,
			orders[i].Currency,
			b.GetPairFormat(asset.Spot, false).Delimiter)
	}

	return OrderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTCMarkets) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *BTCMarkets) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	return b.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Address)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *BTCMarkets) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	if withdrawRequest.Currency != currency.AUD {
		return "", errors.New("only AUD is supported for withdrawals")
	}
	return b.WithdrawAUD(withdrawRequest.BankAccountName,
		strconv.FormatFloat(withdrawRequest.BankAccountNumber, 'f', -1, 64),
		withdrawRequest.BankName,
		strconv.FormatFloat(withdrawRequest.BankCode, 'f', -1, 64),
		withdrawRequest.Amount)
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
	resp, err := b.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		var side order.Side
		if strings.EqualFold(resp[i].OrderSide, order.Ask.String()) {
			side = order.Sell
		} else if strings.EqualFold(resp[i].OrderSide, order.Bid.String()) {
			side = order.Buy
		}
		orderDate := time.Unix(int64(resp[i].CreationTime), 0)
		orderType := order.Type(strings.ToUpper(resp[i].OrderType))

		openOrder := order.Detail{
			ID:              strconv.FormatInt(resp[i].ID, 10),
			Amount:          resp[i].Volume,
			Exchange:        b.Name,
			RemainingAmount: resp[i].OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           resp[i].Price,
			Status:          order.Status(resp[i].Status),
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].Instrument,
				resp[i].Currency,
				b.GetPairFormat(asset.Spot, false).Delimiter),
		}

		for j := range resp[i].Trades {
			tradeDate := time.Unix(int64(resp[i].Trades[j].CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, order.TradeHistory{
				Amount:      resp[i].Trades[j].Volume,
				Exchange:    b.Name,
				Price:       resp[i].Trades[j].Price,
				TID:         resp[i].Trades[j].ID,
				Timestamp:   tradeDate,
				Fee:         resp[i].Trades[j].Fee,
				Description: resp[i].Trades[j].Description,
			})
		}

		orders = append(orders, openOrder)
	}

	order.FilterOrdersByType(&orders, req.OrderType)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTCMarkets) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(req.Currencies) == 0 {
		return nil, errors.New("requires at least one currency pair to retrieve history")
	}

	var respOrders []Order
	for i := range req.Currencies {
		resp, err := b.GetOrders(req.Currencies[i].Base.String(),
			req.Currencies[i].Quote.String(),
			200,
			0,
			true)
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	var orders []order.Detail
	for i := range respOrders {
		var side order.Side
		if strings.EqualFold(respOrders[i].OrderSide, order.Ask.String()) {
			side = order.Sell
		} else if strings.EqualFold(respOrders[i].OrderSide, order.Bid.String()) {
			side = order.Buy
		}
		orderDate := time.Unix(int64(respOrders[i].CreationTime), 0)
		orderType := order.Type(strings.ToUpper(respOrders[i].OrderType))

		openOrder := order.Detail{
			ID:              strconv.FormatInt(respOrders[i].ID, 10),
			Amount:          respOrders[i].Volume,
			Exchange:        b.Name,
			RemainingAmount: respOrders[i].OpenVolume,
			OrderDate:       orderDate,
			OrderSide:       side,
			OrderType:       orderType,
			Price:           respOrders[i].Price,
			Status:          order.Status(respOrders[i].Status),
			CurrencyPair: currency.NewPairWithDelimiter(respOrders[i].Instrument,
				respOrders[i].Currency,
				b.GetPairFormat(asset.Spot, false).Delimiter),
		}

		for j := range respOrders[i].Trades {
			tradeDate := time.Unix(int64(respOrders[i].Trades[j].CreationTime), 0)
			openOrder.Trades = append(openOrder.Trades, order.TradeHistory{
				Amount:      respOrders[i].Trades[j].Volume,
				Exchange:    b.Name,
				Price:       respOrders[i].Trades[j].Price,
				TID:         respOrders[i].Trades[j].ID,
				Timestamp:   tradeDate,
				Fee:         respOrders[i].Trades[j].Fee,
				Description: respOrders[i].Trades[j].Description,
			})
		}
		orders = append(orders, openOrder)
	}

	order.FilterOrdersByType(&orders, req.OrderType)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.OrderSide)
	return orders, nil
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
