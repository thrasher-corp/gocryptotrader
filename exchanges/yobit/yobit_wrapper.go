package yobit

import (
	"errors"
	"math"
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
func (y *Yobit) GetDefaultConfig() (*config.ExchangeConfig, error) {
	y.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = y.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = y.BaseCurrencies

	err := y.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if y.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = y.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default value for Yobit
func (y *Yobit) SetDefaults() {
	y.Name = "Yobit"
	y.Enabled = true
	y.Verbose = true
	y.API.CredentialsValidator.RequiresKey = true
	y.API.CredentialsValidator.RequiresSecret = true

	y.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: false,
			Separator: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	y.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
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
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	y.Requester = request.New(y.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		// Server responses are cached every 2 seconds.
		request.WithLimiter(request.NewBasicRateLimit(time.Second, 1)))

	y.API.Endpoints.URLDefault = apiPublicURL
	y.API.Endpoints.URL = y.API.Endpoints.URLDefault
	y.API.Endpoints.URLSecondaryDefault = apiPrivateURL
	y.API.Endpoints.URLSecondary = y.API.Endpoints.URLSecondaryDefault
}

// Setup sets exchange configuration parameters for Yobit
func (y *Yobit) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		y.SetEnabled(false)
		return nil
	}

	return y.SetupDefaults(exch)
}

// Start starts the WEX go routine
func (y *Yobit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		y.Run()
		wg.Done()
	}()
}

// Run implements the Yobit wrapper
func (y *Yobit) Run() {
	if y.Verbose {
		y.PrintEnabledPairs()
	}

	if !y.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := y.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			y.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (y *Yobit) FetchTradablePairs(asset asset.Item) ([]string, error) {
	info, err := y.GetInfo()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range info.Pairs {
		currencies = append(currencies, strings.ToUpper(x))
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (y *Yobit) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := y.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return y.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (y *Yobit) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	pairsCollated, err := y.FormatExchangeCurrencies(y.GetEnabledPairs(assetType), assetType)
	if err != nil {
		return tickerPrice, err
	}

	result, err := y.GetTicker(pairsCollated)
	if err != nil {
		return tickerPrice, err
	}

	enabledPairs := y.GetEnabledPairs(assetType)
	for i := range enabledPairs {
		curr := y.FormatExchangeCurrency(enabledPairs[i], assetType).Lower().String()
		if _, ok := result[curr]; !ok {
			continue
		}
		resultCurr := result[curr]
		tickerPrice := new(ticker.Price)
		tickerPrice.Pair = enabledPairs[i]
		tickerPrice.Last = resultCurr.Last
		tickerPrice.Ask = resultCurr.Sell
		tickerPrice.Bid = resultCurr.Buy
		tickerPrice.Last = resultCurr.Last
		tickerPrice.Low = resultCurr.Low
		tickerPrice.QuoteVolume = resultCurr.VolumeCurrent
		tickerPrice.Volume = resultCurr.Vol

		err = ticker.ProcessTicker(y.Name, tickerPrice, assetType)
		if err != nil {
			log.Error(log.Ticker, err)
		}
	}
	return ticker.GetTicker(y.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (y *Yobit) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := ticker.GetTicker(y.Name, p, assetType)
	if err != nil {
		return y.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (y *Yobit) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(y.Name, p, assetType)
	if err != nil {
		return y.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (y *Yobit) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := y.GetDepth(y.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return orderBook, err
	}

	for i := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Price:  orderbookNew.Bids[i][0],
				Amount: orderbookNew.Bids[i][1],
			})
	}

	for i := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Price:  orderbookNew.Asks[i][0],
				Amount: orderbookNew.Asks[i][1],
			})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = y.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(y.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Yobit exchange
func (y *Yobit) UpdateAccountInfo() (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = y.Name
	accountBalance, err := y.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for x, y := range accountBalance.FundsInclOrders {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		for z, w := range accountBalance.Funds {
			if z == x {
				exchangeCurrency.Hold = y - w
			}
		}

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
func (y *Yobit) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(y.Name)
	if err != nil {
		return y.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (y *Yobit) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (y *Yobit) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
// Yobit only supports limit orders
func (y *Yobit) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	if s.Type != order.Limit {
		return submitOrderResponse, errors.New("only limit orders are allowed")
	}

	response, err := y.Trade(s.Pair.String(),
		s.Side.String(),
		s.Amount,
		s.Price)
	if err != nil {
		return submitOrderResponse, err
	}
	if response > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response, 10)
	}

	submitOrderResponse.IsOrderPlaced = true
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (y *Yobit) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (y *Yobit) CancelOrder(order *order.Cancel) error {
	orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
	if err != nil {
		return err
	}

	return y.CancelExistingOrder(orderIDInt)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (y *Yobit) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	var allActiveOrders []map[string]ActiveOrders
	enabledPairs := y.GetEnabledPairs(asset.Spot)
	for i := range enabledPairs {
		fCurr := y.FormatExchangeCurrency(enabledPairs[i], asset.Spot).String()
		activeOrdersForPair, err := y.GetOpenOrders(fCurr)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		allActiveOrders = append(allActiveOrders, activeOrdersForPair)
	}

	for i := range allActiveOrders {
		for key := range allActiveOrders[i] {
			orderIDInt, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				cancelAllOrdersResponse.Status[key] = err.Error()
				continue
			}

			err = y.CancelExistingOrder(orderIDInt)
			if err != nil {
				cancelAllOrdersResponse.Status[key] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (y *Yobit) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (y *Yobit) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	a, err := y.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	return a.Return.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (y *Yobit) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	resp, err := y.WithdrawCoinsToAddress(withdrawRequest.Currency.String(), withdrawRequest.Amount, withdrawRequest.Crypto.Address)
	if err != nil {
		return nil, err
	}
	if len(resp.Error) > 0 {
		return nil, errors.New(resp.Error)
	}
	return &withdraw.ExchangeResponse{}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (y *Yobit) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (y *Yobit) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !y.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return y.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (y *Yobit) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var orders []order.Detail
	for x := range req.Pairs {
		fCurr := y.FormatExchangeCurrency(req.Pairs[x], asset.Spot).String()
		resp, err := y.GetOpenOrders(fCurr)
		if err != nil {
			return nil, err
		}

		for id := range resp {
			symbol := currency.NewPairDelimiter(resp[id].Pair,
				y.GetPairFormat(asset.Spot, false).Delimiter)
			orderDate := time.Unix(int64(resp[id].TimestampCreated), 0)
			side := order.Side(strings.ToUpper(resp[id].Type))
			orders = append(orders, order.Detail{
				ID:       id,
				Amount:   resp[id].Amount,
				Price:    resp[id].Rate,
				Side:     side,
				Date:     orderDate,
				Pair:     symbol,
				Exchange: y.Name,
			})
		}
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (y *Yobit) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var allOrders []TradeHistory
	for x := range req.Pairs {
		resp, err := y.GetTradeHistory(0,
			10000,
			math.MaxInt64,
			req.StartTicks.Unix(),
			req.EndTicks.Unix(),
			"DESC",
			y.FormatExchangeCurrency(req.Pairs[x], asset.Spot).String())
		if err != nil {
			return nil, err
		}

		for key := range resp {
			allOrders = append(allOrders, resp[key])
		}
	}

	var orders []order.Detail
	for i := range allOrders {
		symbol := currency.NewPairDelimiter(allOrders[i].Pair,
			y.GetPairFormat(asset.Spot, false).Delimiter)
		orderDate := time.Unix(int64(allOrders[i].Timestamp), 0)
		side := order.Side(strings.ToUpper(allOrders[i].Type))
		orders = append(orders, order.Detail{
			ID:       strconv.FormatFloat(allOrders[i].OrderID, 'f', -1, 64),
			Amount:   allOrders[i].Amount,
			Price:    allOrders[i].Rate,
			Side:     side,
			Date:     orderDate,
			Pair:     symbol,
			Exchange: y.Name,
		})
	}

	order.FilterOrdersBySide(&orders, req.Side)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (y *Yobit) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (y *Yobit) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (y *Yobit) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (y *Yobit) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (y *Yobit) ValidateCredentials() error {
	_, err := y.UpdateAccountInfo()
	return y.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (y *Yobit) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
