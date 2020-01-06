package anx

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/exchanges/withdraw"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config for Alphapoint
func (a *ANX) GetDefaultConfig() (*config.ExchangeConfig, error) {
	a.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = a.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = a.BaseCurrencies

	err := a.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if a.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = a.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default settings
func (a *ANX) SetDefaults() {
	a.Name = "ANX"
	a.Enabled = true
	a.Verbose = true
	a.BaseCurrencies = currency.Currencies{
		currency.USD,
		currency.HKD,
		currency.EUR,
		currency.CAD,
		currency.AUD,
		currency.SGD,
		currency.JPY,
		currency.GBP,
		currency.NZD,
	}
	a.API.CredentialsValidator.RequiresKey = true
	a.API.CredentialsValidator.RequiresSecret = true
	a.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	a.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
			Uppercase: true,
		},
	}

	a.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				TradeFee:            true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.WithdrawCryptoWithEmail |
				exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawCryptoWith2FA |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: false,
		},
	}

	a.Requester = request.New(a.Name,
		request.NewRateLimit(time.Second, anxAuthRate),
		request.NewRateLimit(time.Second, anxUnauthRate),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	a.API.Endpoints.URLDefault = anxAPIURL
	a.API.Endpoints.URL = a.API.Endpoints.URLDefault
}

// Setup is run on startup to setup exchange with config values
func (a *ANX) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		a.SetEnabled(false)
		return nil
	}

	return a.SetupDefaults(exch)
}

// Start starts the ANX go routine
func (a *ANX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		a.Run()
		wg.Done()
	}()
}

// Run implements the ANX wrapper
func (a *ANX) Run() {
	if a.Verbose {
		a.PrintEnabledPairs()
	}

	forceUpdate := false
	delim := a.GetPairFormat(asset.Spot, false).Delimiter
	if !common.StringDataContains(a.GetEnabledPairs(asset.Spot).Strings(), delim) ||
		!common.StringDataContains(a.GetAvailablePairs(asset.Spot).Strings(), delim) {
		enabledPairs := currency.NewPairsFromStrings(
			[]string{currency.BTC.String() + delim + currency.USD.String()},
		)
		log.Warn(log.ExchangeSys,
			"Enabled pairs for ANX reset due to config upgrade, please enable the ones you would like again.")
		forceUpdate = true
		err := a.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update currencies.\n", a.Name)
			return
		}
	}

	if !a.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := a.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", a.Name, err)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (a *ANX) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := a.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return a.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (a *ANX) FetchTradablePairs(asset asset.Item) ([]string, error) {
	result, err := a.GetCurrencies()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result.CurrencyPairs {
		currencies = append(currencies, result.CurrencyPairs[x].TradedCcy+
			a.GetPairFormat(asset, false).Delimiter+
			result.CurrencyPairs[x].SettlementCcy)
	}

	return currencies, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *ANX) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	tick, err := a.GetTicker(a.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return tickerPrice, err
	}
	last, _ := convert.FloatFromString(tick.Data.Last.Value)
	high, _ := convert.FloatFromString(tick.Data.High.Value)
	low, _ := convert.FloatFromString(tick.Data.Low.Value)
	bid, _ := convert.FloatFromString(tick.Data.Buy.Value)
	ask, _ := convert.FloatFromString(tick.Data.Sell.Value)
	volume, _ := convert.FloatFromString(tick.Data.Volume.Value)

	tickerPrice = &ticker.Price{
		Last:        last,
		High:        high,
		Low:         low,
		Bid:         bid,
		Ask:         ask,
		Volume:      volume,
		Pair:        p,
		LastUpdated: time.Unix(0, tick.Data.UpdateTime),
	}

	err = ticker.ProcessTicker(a.Name, tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(a.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (a *ANX) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(a.Name, p, assetType)
	if err != nil {
		return a.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (a *ANX) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(a.Name, p, assetType)
	if err != nil {
		return a.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *ANX) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := a.GetDepth(a.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Data.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Price:  orderbookNew.Data.Asks[x].Price,
				Amount: orderbookNew.Data.Asks[x].Amount})
	}

	for x := range orderbookNew.Data.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Price:  orderbookNew.Data.Bids[x].Price,
				Amount: orderbookNew.Data.Bids[x].Amount})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = a.Name
	orderBook.AssetType = assetType
	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(a.Name, p, assetType)
}

// FetchTrade returns the trades for a currency pair
func (a *ANX) FetchTrades(p currency.Pair, assetType asset.Item) ([]order.Trade, error) {
	return nil, errors.New("NOT DONE")
}

// UpdateTrade updates and returns the trades for a currency pair
func (a *ANX) UpdateTrades(p currency.Pair, assetType asset.Item) ([]order.Trade, error) {
	return nil, errors.New("NOT DONE")
}

// GetAccountInfo retrieves balances for all enabled currencies on the
// exchange
func (a *ANX) GetAccountInfo() (exchange.AccountInfo, error) {
	var info exchange.AccountInfo

	raw, err := a.GetAccountInformation()
	if err != nil {
		return info, err
	}

	var balance []exchange.AccountCurrencyInfo
	for c := range raw.Wallets {
		balance = append(balance, exchange.AccountCurrencyInfo{
			CurrencyName: currency.NewCode(c),
			TotalValue:   raw.Wallets[c].AvailableBalance.Value,
			Hold:         raw.Wallets[c].Balance.Value,
		})
	}

	info.Exchange = a.Name
	info.Accounts = append(info.Accounts, exchange.Account{
		Currencies: balance,
	})

	return info, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (a *ANX) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (a *ANX) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (a *ANX) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var isBuying bool
	var limitPriceInSettlementCurrency float64

	if s.OrderSide == order.Buy {
		isBuying = true
	}

	if s.OrderType == order.Limit {
		limitPriceInSettlementCurrency = s.Price
	}

	response, err := a.NewOrder(s.OrderType.String(),
		isBuying,
		s.Pair.Base.String(),
		s.Amount,
		s.Pair.Quote.String(),
		s.Amount,
		limitPriceInSettlementCurrency,
		false,
		"",
		false)
	if err != nil {
		return submitOrderResponse, err
	}
	if response != "" {
		submitOrderResponse.OrderID = response
	}
	if s.OrderType == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (a *ANX) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (a *ANX) CancelOrder(order *order.Cancel) error {
	orderIDs := []string{order.OrderID}
	_, err := a.CancelOrderByIDs(orderIDs)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (a *ANX) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	placedOrders, err := a.GetOrderList(true)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var orderIDs []string
	for i := range placedOrders {
		orderIDs = append(orderIDs, placedOrders[i].OrderID)
	}

	resp, err := a.CancelOrderByIDs(orderIDs)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range resp.OrderCancellationResponses {
		if resp.OrderCancellationResponses[i].Error != CancelRequestSubmitted {
			cancelAllOrdersResponse.Status[resp.OrderCancellationResponses[i].UUID] = resp.OrderCancellationResponses[i].Error
		}
	}

	return cancelAllOrdersResponse, err
}

// GetOrderInfo returns information on a current open order
func (a *ANX) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (a *ANX) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return a.GetDepositAddressByCurrency(cryptocurrency.String(), "", false)
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.CryptoRequest) (string, error) {
	return a.Send(withdrawRequest.Currency.String(), withdrawRequest.Address, "", strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64))
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFunds(withdrawRequest *withdraw.FiatRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *ANX) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.FiatRequest) (string, error) {
	// Fiat withdrawals available via website
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (a *ANX) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *ANX) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!a.AllowAuthenticatedRequest() || a.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return a.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (a *ANX) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	resp, err := a.GetOrderList(true)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		orderDate := time.Unix(resp[i].Timestamp, 0)
		orderType := order.Type(strings.ToUpper(resp[i].OrderType))

		orderDetail := order.Detail{
			Amount: resp[i].TradedCurrencyAmount,
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].TradedCurrency,
				resp[i].SettlementCurrency,
				a.GetPairFormat(asset.Spot, false).Delimiter),
			OrderDate: orderDate,
			Exchange:  a.Name,
			ID:        resp[i].OrderID,
			OrderType: orderType,
			Price:     resp[i].SettlementCurrencyAmount,
			Status:    order.Status(resp[i].OrderStatus),
		}

		orders = append(orders, orderDetail)
	}

	order.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	order.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	order.FilterOrdersByCurrencies(&orders, getOrdersRequest.Currencies)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (a *ANX) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	resp, err := a.GetOrderList(false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		orderDate := time.Unix(resp[i].Timestamp, 0)
		orderType := order.Type(strings.ToUpper(resp[i].OrderType))

		orderDetail := order.Detail{
			Amount:    resp[i].TradedCurrencyAmount,
			OrderDate: orderDate,
			Exchange:  a.Name,
			ID:        resp[i].OrderID,
			OrderType: orderType,
			Price:     resp[i].SettlementCurrencyAmount,
			Status:    order.Status(resp[i].OrderStatus),
			CurrencyPair: currency.NewPairWithDelimiter(resp[i].TradedCurrency,
				resp[i].SettlementCurrency,
				a.GetPairFormat(asset.Spot, false).Delimiter),
		}

		orders = append(orders, orderDetail)
	}

	order.FilterOrdersByType(&orders, req.OrderType)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersByCurrencies(&orders, req.Currencies)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (a *ANX) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (a *ANX) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (a *ANX) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (a *ANX) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}
