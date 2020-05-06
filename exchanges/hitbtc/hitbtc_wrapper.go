package hitbtc

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
func (h *HitBTC) GetDefaultConfig() (*config.ExchangeConfig, error) {
	h.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = h.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = h.BaseCurrencies

	err := h.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if h.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = h.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default settings for hitbtc
func (h *HitBTC) SetDefaults() {
	h.Name = "HitBTC"
	h.Enabled = true
	h.Verbose = true
	h.API.CredentialsValidator.RequiresKey = true
	h.API.CredentialsValidator.RequiresSecret = true

	h.CurrencyPairs = currency.PairsManager{
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

	h.Features = exchange.Features{
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
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				ModifyOrder:         true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				SubmitOrder:            true,
				CancelOrder:            true,
				MessageSequenceNumbers: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	h.Requester = request.New(h.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))

	h.API.Endpoints.URLDefault = apiURL
	h.API.Endpoints.URL = h.API.Endpoints.URLDefault
	h.API.Endpoints.WebsocketURL = hitbtcWebsocketAddress
	h.Websocket = wshandler.New()
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (h *HitBTC) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		h.SetEnabled(false)
		return nil
	}

	err := h.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = h.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       hitbtcWebsocketAddress,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        h.WsConnect,
			Subscriber:                       h.Subscribe,
			UnSubscriber:                     h.Unsubscribe,
			Features:                         &h.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}

	h.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         h.Name,
		URL:                  h.Websocket.GetWebsocketURL(),
		ProxyURL:             h.Websocket.GetProxyAddress(),
		Verbose:              h.Verbose,
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	h.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		true,
		true,
		true,
		false,
		exch.Name)
	return nil
}

// Start starts the HitBTC go routine
func (h *HitBTC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
}

// Run implements the HitBTC wrapper
func (h *HitBTC) Run() {
	if h.Verbose {
		log.Debugf(log.ExchangeSys, "%s Websocket: %s (url: %s).\n", h.Name, common.IsEnabled(h.Websocket.IsEnabled()), hitbtcWebsocketAddress)
		h.PrintEnabledPairs()
	}

	forceUpdate := false
	delim := h.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(h.GetEnabledPairs(asset.Spot).Strings(), delim) ||
		!common.StringDataContains(h.GetAvailablePairs(asset.Spot).Strings(), delim) {
		enabledPairs := []string{currency.BTC.String() + delim + currency.USD.String()}
		log.Warn(log.ExchangeSys, "Available pairs for HitBTC reset due to config upgrade, please enable the ones you would like again.")
		forceUpdate = true

		err := h.UpdatePairs(currency.NewPairsFromStrings(enabledPairs), asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update enabled currencies.\n", h.Name)
		}
	}

	if !h.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := h.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", h.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (h *HitBTC) FetchTradablePairs(asset asset.Item) ([]string, error) {
	symbols, err := h.GetSymbolsDetailed()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range symbols {
		pairs = append(pairs, symbols[x].BaseCurrency+
			h.GetPairFormat(asset, false).Delimiter+symbols[x].QuoteCurrency)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (h *HitBTC) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := h.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return h.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HitBTC) UpdateTicker(currencyPair currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	tick, err := h.GetTickers()
	if err != nil {
		return tickerPrice, err
	}
	pairs := h.GetEnabledPairs(assetType)
	for i := range pairs {
		for j := range tick {
			pairFmt := h.FormatExchangeCurrency(pairs[i], assetType).String()
			if tick[j].Symbol != pairFmt {
				found := false
				if strings.Contains(tick[j].Symbol, "USDT") {
					if pairFmt == tick[j].Symbol[0:len(tick[j].Symbol)-1] {
						found = true
					}
				}
				if !found {
					continue
				}
			}
			tickerPrice := &ticker.Price{
				Last:        tick[j].Last,
				High:        tick[j].High,
				Low:         tick[j].Low,
				Bid:         tick[j].Bid,
				Ask:         tick[j].Ask,
				Volume:      tick[j].Volume,
				QuoteVolume: tick[j].VolumeQuote,
				Open:        tick[j].Open,
				Pair:        pairs[i],
				LastUpdated: tick[j].Timestamp,
			}
			err = ticker.ProcessTicker(h.Name, tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}
	}
	return ticker.GetTicker(h.Name, currencyPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (h *HitBTC) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.Name, p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (h *HitBTC) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(h.Name, p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HitBTC) UpdateOrderbook(currencyPair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := h.GetOrderbook(h.FormatExchangeCurrency(currencyPair, assetType).String(), 1000)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		})
	}

	orderBook.Pair = currencyPair
	orderBook.ExchangeName = h.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(h.Name, currencyPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// HitBTC exchange
func (h *HitBTC) UpdateAccountInfo() (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = h.Name
	accountBalance, err := h.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for i := range accountBalance {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Available
		exchangeCurrency.Hold = accountBalance[i].Reserved
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
func (h *HitBTC) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(h.Name)
	if err != nil {
		return h.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HitBTC) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (h *HitBTC) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (h *HitBTC) SubmitOrder(o *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	err := o.Validate()
	if err != nil {
		return submitOrderResponse, err
	}
	if h.Websocket.IsConnected() && h.Websocket.CanUseAuthenticatedEndpoints() {
		var response *WsSubmitOrderSuccessResponse
		response, err = h.wsPlaceOrder(o.Pair, o.Side.String(), o.Amount, o.Price)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = strconv.FormatInt(response.ID, 10)
		if response.Result.CumQuantity == o.Amount {
			submitOrderResponse.FullyMatched = true
		}
	} else {
		var response OrderResponse
		response, err = h.PlaceOrder(o.Pair.String(),
			o.Price,
			o.Amount,
			strings.ToLower(o.Type.String()),
			strings.ToLower(o.Side.String()))
		if err != nil {
			return submitOrderResponse, err
		}
		if response.OrderNumber > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response.OrderNumber, 10)
		}
		if o.Type == order.Market {
			submitOrderResponse.FullyMatched = true
		}
	}
	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (h *HitBTC) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HitBTC) CancelOrder(order *order.Cancel) error {
	orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(orderIDInt)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HitBTC) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	resp, err := h.CancelAllExistingOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range resp {
		if resp[i].Status != "canceled" {
			cancelAllOrdersResponse.Status[strconv.FormatInt(resp[i].ID, 10)] =
				fmt.Sprintf("Could not cancel order %v. Status: %v",
					resp[i].ID,
					resp[i].Status)
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (h *HitBTC) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HitBTC) GetDepositAddress(currency currency.Code, _ string) (string, error) {
	resp, err := h.GetDepositAddresses(currency.String())
	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HitBTC) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	v, err := h.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.Crypto.Address, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: common.IsEnabled(v),
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (h *HitBTC) GetWebsocket() (*wshandler.Websocket, error) {
	return h.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !h.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return h.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HitBTC) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for i := range req.Pairs {
		resp, err := h.GetOpenOrders(req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []order.Detail
	for i := range allOrders {
		symbol := currency.NewPairDelimiter(allOrders[i].Symbol,
			h.GetPairFormat(asset.Spot, false).Delimiter)
		side := order.Side(strings.ToUpper(allOrders[i].Side))
		orders = append(orders, order.Detail{
			ID:       allOrders[i].ID,
			Amount:   allOrders[i].Quantity,
			Exchange: h.Name,
			Price:    allOrders[i].Price,
			Date:     allOrders[i].CreatedAt,
			Side:     side,
			Pair:     symbol,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HitBTC) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for i := range req.Pairs {
		resp, err := h.GetOrders(req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	var orders []order.Detail
	for i := range allOrders {
		symbol := currency.NewPairDelimiter(allOrders[i].Symbol,
			h.GetPairFormat(asset.Spot, false).Delimiter)
		side := order.Side(strings.ToUpper(allOrders[i].Side))
		orders = append(orders, order.Detail{
			ID:       allOrders[i].ID,
			Amount:   allOrders[i].Quantity,
			Exchange: h.Name,
			Price:    allOrders[i].Price,
			Date:     allOrders[i].CreatedAt,
			Side:     side,
			Pair:     symbol,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (h *HitBTC) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	h.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (h *HitBTC) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	h.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (h *HitBTC) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return h.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (h *HitBTC) AuthenticateWebsocket() error {
	return h.wsLogin()
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (h *HitBTC) ValidateCredentials() error {
	_, err := h.UpdateAccountInfo()
	return h.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (h *HitBTC) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
