package bitfinex

import (
	"errors"
	"fmt"
	"net/url"
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
func (b *Bitfinex) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets the basic defaults for bitfinex
func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = true
	b.Verbose = true
	b.WebsocketSubdChannels = make(map[int]WebsocketChanInfo)
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
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
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				FiatWithdraw:        true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				TradeFetching:       true,
				UserTradeHistory:    true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				CancelOrders:           true,
				CancelOrder:            true,
				SubmitOrder:            true,
				ModifyOrder:            true,
				TickerFetching:         true,
				KlineFetching:          true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				DeadMansSwitch:         true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.AutoWithdrawFiatWithAPIPermission,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second*60, bitfinexAuthRate),
		request.NewRateLimit(time.Second*60, bitfinexUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = bitfinexAPIURLBase
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.API.Endpoints.WebsocketURL = publicBitfinexWebsocketEndpoint
	b.Websocket = wshandler.New()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitfinex) Setup(exch *config.ExchangeConfig) error {
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
			DefaultURL:                       publicBitfinexWebsocketEndpoint,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        b.WsConnect,
			Subscriber:                       b.Subscribe,
			UnSubscriber:                     b.Unsubscribe,
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
	b.AuthenticatedWebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         b.Name,
		URL:                  authenticatedBitfinexWebsocketEndpoint,
		ProxyURL:             b.Websocket.GetProxyAddress(),
		Verbose:              b.Verbose,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	b.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		true,
		false,
		false,
		false,
		exch.Name)
	return nil
}

// Start starts the Bitfinex go routine
func (b *Bitfinex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitfinex wrapper
func (b *Bitfinex) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()))
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitfinex) FetchTradablePairs(asset asset.Item) ([]string, error) {
	return b.GetSymbols()
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitfinex) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitfinex) UpdateTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	var tickerPrice ticker.Price
	enabledPairs := b.GetEnabledPairs(assetType)
	var pairs []string
	for x := range enabledPairs {
		b.appendOptionalDelimiter(&enabledPairs[x])
		pairs = append(pairs, "t"+enabledPairs[x].String())
	}
	tickerNew, err := b.GetTickersV2(strings.Join(pairs, ","))
	if err != nil {
		return tickerPrice, err
	}
	for i := range tickerNew {
		newP := tickerNew[i].Symbol[1:] // Remove the "t" prefix
		tick := ticker.Price{
			Last:        tickerNew[i].Last,
			High:        tickerNew[i].High,
			Low:         tickerNew[i].Low,
			Bid:         tickerNew[i].Bid,
			Ask:         tickerNew[i].Ask,
			Volume:      tickerNew[i].Volume,
			Pair:        currency.NewPairFromString(newP),
			LastUpdated: tickerNew[i].Timestamp,
		}
		err = ticker.ProcessTicker(b.Name, &tick, assetType)
		if err != nil {
			log.Error(log.Ticker, err)
		}
	}

	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitfinex) FetchTicker(p currency.Pair, assetType asset.Item) (ticker.Price, error) {
	b.appendOptionalDelimiter(&p)
	tick, err := ticker.GetTicker(b.Name, p, asset.Spot)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitfinex) FetchOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	b.appendOptionalDelimiter(&p)
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitfinex) UpdateOrderbook(p currency.Pair, assetType asset.Item) (orderbook.Base, error) {
	b.appendOptionalDelimiter(&p)
	var orderBook orderbook.Base
	urlVals := url.Values{}
	urlVals.Set("limit_bids", "100")
	urlVals.Set("limit_asks", "100")
	orderbookNew, err := b.GetOrderbook(p.String(), urlVals)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{Price: orderbookNew.Asks[x].Price,
				Amount: orderbookNew.Asks[x].Amount})
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{Price: orderbookNew.Bids[x].Price,
				Amount: orderbookNew.Bids[x].Amount})
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

// GetAccountInfo retrieves balances for all enabled currencies on the
// Bitfinex exchange
func (b *Bitfinex) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.Exchange = b.Name

	accountBalance, err := b.GetAccountBalance()
	if err != nil {
		return response, err
	}

	var Accounts = []exchange.Account{
		{ID: "deposit"},
		{ID: "exchange"},
		{ID: "trading"},
	}

	for x := range accountBalance {
		for i := range Accounts {
			if Accounts[i].ID == accountBalance[x].Type {
				Accounts[i].Currencies = append(Accounts[i].Currencies,
					exchange.AccountCurrencyInfo{
						CurrencyName: currency.NewCode(accountBalance[x].Currency),
						TotalValue:   accountBalance[x].Amount,
						Hold:         accountBalance[x].Amount - accountBalance[x].Available,
					})
			}
		}
	}

	response.Accounts = Accounts
	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitfinex) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitfinex) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitfinex) SubmitOrder(o *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	err := o.Validate()
	if err != nil {
		return submitOrderResponse, err
	}
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		submitOrderResponse.OrderID, err = b.WsNewOrder(&WsNewOrderRequest{
			CustomID: b.AuthenticatedWebsocketConn.GenerateMessageID(false),
			Type:     o.OrderType.String(),
			Symbol:   b.FormatExchangeCurrency(o.Pair, asset.Spot).String(),
			Amount:   o.Amount,
			Price:    o.Price,
		})
		if err != nil {
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, err
		}
	} else {
		var response Order
		isBuying := o.OrderSide == order.Buy
		b.appendOptionalDelimiter(&o.Pair)
		response, err = b.NewOrder(o.Pair.String(),
			o.Amount,
			o.Price,
			isBuying,
			o.OrderType.String(),
			false)
		if err != nil {
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, err
		}
		if response.OrderID > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response.OrderID, 10)
		}
		if response.RemainingAmount == 0 {
			submitOrderResponse.FullyMatched = true
		}

		submitOrderResponse.IsOrderPlaced = true
	}
	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitfinex) ModifyOrder(action *order.Modify) (string, error) {
	orderIDInt, err := strconv.ParseInt(action.OrderID, 10, 64)
	if err != nil {
		return action.OrderID, err
	}
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		if action.Side == order.Sell && action.Amount > 0 {
			action.Amount = -1 * action.Amount
		}
		err = b.WsModifyOrder(&WsUpdateOrderRequest{
			OrderID: orderIDInt,
			Price:   action.Price,
			Amount:  action.Amount,
		})
		return action.OrderID, err
	}
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitfinex) CancelOrder(order *order.Cancel) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	if err != nil {
		return err
	}
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		err = b.WsCancelOrder(orderIDInt)
	} else {
		_, err = b.CancelExistingOrder(orderIDInt)
	}
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitfinex) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	var err error
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		err = b.WsCancelAllOrders()
	} else {
		_, err = b.CancelAllExistingOrders()
	}
	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (b *Bitfinex) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitfinex) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	method, err := b.ConvertSymbolToDepositMethod(cryptocurrency)
	if err != nil {
		return "", err
	}

	var resp DepositResponse
	resp, err = b.NewDeposit(method, accountID, 0)
	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *Bitfinex) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.CryptoWithdrawRequest) (string, error) {
	withdrawalType := b.ConvertSymbolToWithdrawalType(withdrawRequest.Currency)
	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawCryptocurrency(withdrawalType,
		walletType,
		withdrawRequest.Address,
		withdrawRequest.Description,
		withdrawRequest.Amount,
		withdrawRequest.Currency)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errors.New("no withdrawID returned. Check order status")
	}

	return strconv.FormatInt(resp[0].WithdrawalID, 10), err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
// Returns comma delimited withdrawal IDs
func (b *Bitfinex) WithdrawFiatFunds(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	withdrawalType := "wire"
	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawFIAT(withdrawalType, walletType, withdrawRequest)
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errors.New("no withdrawID returned. Check order status")
	}

	var withdrawalSuccesses string
	var withdrawalErrors string
	for _, withdrawal := range resp {
		if withdrawal.Status == "error" {
			withdrawalErrors += fmt.Sprintf("%v ", withdrawal.Message)
		}
		if withdrawal.Status == "success" {
			withdrawalSuccesses += fmt.Sprintf("%v,", withdrawal.WithdrawalID)
		}
	}
	if len(withdrawalErrors) > 0 {
		return withdrawalSuccesses, errors.New(withdrawalErrors)
	}

	return withdrawalSuccesses, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
// Returns comma delimited withdrawal IDs
func (b *Bitfinex) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.FiatWithdrawRequest) (string, error) {
	return b.WithdrawFiatFunds(withdrawRequest)
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitfinex) GetWebsocket() (*wshandler.Websocket, error) {
	return b.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitfinex) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var orders []order.Detail
	resp, err := b.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := order.Side(strings.ToUpper(resp[i].Side))
		timestamp, err := strconv.ParseInt(resp[i].Timestamp, 10, 64)
		if err != nil {
			log.Warnf(log.ExchangeSys,
				"Unable to convert timestamp '%s', leaving blank",
				resp[i].Timestamp)
		}
		orderDate := time.Unix(timestamp, 0)

		orderDetail := order.Detail{
			Amount:          resp[i].OriginalAmount,
			OrderDate:       orderDate,
			Exchange:        b.Name,
			ID:              strconv.FormatInt(resp[i].OrderID, 10),
			OrderSide:       orderSide,
			Price:           resp[i].Price,
			RemainingAmount: resp[i].RemainingAmount,
			CurrencyPair:    currency.NewPairFromString(resp[i].Symbol),
			ExecutedAmount:  resp[i].ExecutedAmount,
		}

		switch {
		case resp[i].IsLive:
			orderDetail.Status = order.Active
		case resp[i].IsCancelled:
			orderDetail.Status = order.Cancelled
		case resp[i].IsHidden:
			orderDetail.Status = order.Hidden
		default:
			orderDetail.Status = order.UnknownStatus
		}

		// API docs discrepancy. Example contains prefixed "exchange "
		// Return type suggests “market” / “limit” / “stop” / “trailing-stop”
		orderType := strings.Replace(resp[i].Type, "exchange ", "", 1)
		if orderType == "trailing-stop" {
			orderDetail.OrderType = order.TrailingStop
		} else {
			orderDetail.OrderType = order.Type(strings.ToUpper(orderType))
		}

		orders = append(orders, orderDetail)
	}

	order.FilterOrdersBySide(&orders, req.OrderSide)
	order.FilterOrdersByType(&orders, req.OrderType)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersByCurrencies(&orders, req.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitfinex) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var orders []order.Detail
	resp, err := b.GetInactiveOrders()
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := order.Side(strings.ToUpper(resp[i].Side))
		timestamp, err := strconv.ParseInt(resp[i].Timestamp, 10, 64)
		if err != nil {
			log.Warnf(log.ExchangeSys, "Unable to convert timestamp '%v', leaving blank", resp[i].Timestamp)
		}
		orderDate := time.Unix(timestamp, 0)

		orderDetail := order.Detail{
			Amount:          resp[i].OriginalAmount,
			OrderDate:       orderDate,
			Exchange:        b.Name,
			ID:              strconv.FormatInt(resp[i].OrderID, 10),
			OrderSide:       orderSide,
			Price:           resp[i].Price,
			RemainingAmount: resp[i].RemainingAmount,
			ExecutedAmount:  resp[i].ExecutedAmount,
			CurrencyPair:    currency.NewPairFromString(resp[i].Symbol),
		}

		switch {
		case resp[i].IsLive:
			orderDetail.Status = order.Active
		case resp[i].IsCancelled:
			orderDetail.Status = order.Cancelled
		case resp[i].IsHidden:
			orderDetail.Status = order.Hidden
		default:
			orderDetail.Status = order.UnknownStatus
		}

		// API docs discrepency. Example contains prefixed "exchange "
		// Return type suggests “market” / “limit” / “stop” / “trailing-stop”
		orderType := strings.Replace(resp[i].Type, "exchange ", "", 1)
		if orderType == "trailing-stop" {
			orderDetail.OrderType = order.TrailingStop
		} else {
			orderDetail.OrderType = order.Type(strings.ToUpper(orderType))
		}

		orders = append(orders, orderDetail)
	}

	order.FilterOrdersBySide(&orders, req.OrderSide)
	order.FilterOrdersByType(&orders, req.OrderType)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	for i := range req.Currencies {
		b.appendOptionalDelimiter(&req.Currencies[i])
	}
	order.FilterOrdersByCurrencies(&orders, req.Currencies)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *Bitfinex) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	for i := range channels {
		b.appendOptionalDelimiter(&channels[i].Currency)
	}
	b.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *Bitfinex) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	for i := range channels {
		b.appendOptionalDelimiter(&channels[i].Currency)
	}
	b.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (b *Bitfinex) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return b.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Bitfinex) AuthenticateWebsocket() error {
	return b.WsSendAuth()
}

// appendOptionalDelimiter ensures that a delimiter is present for long character currencies
func (b *Bitfinex) appendOptionalDelimiter(p *currency.Pair) {
	if len(p.Quote.String()) > 3 ||
		len(p.Base.String()) > 3 {
		p.Delimiter = ":"
	}
}
