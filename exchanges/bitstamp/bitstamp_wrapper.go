package bitstamp

import (
	"errors"
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
func (b *Bitstamp) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets default for Bitstamp
func (b *Bitstamp) SetDefaults() {
	b.Name = "Bitstamp"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.API.CredentialsValidator.RequiresClientID = true

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
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
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
				CryptoDepositFee:  true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:     true,
				OrderbookFetching: true,
				Subscribe:         true,
				Unsubscribe:       true,
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
		request.WithLimiter(request.NewBasicRateLimit(bitstampRateInterval, bitstampRequestRate)))

	b.API.Endpoints.URLDefault = bitstampAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.API.Endpoints.WebsocketURL = bitstampWSURL
	b.Websocket = wshandler.New()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets configuration values to bitstamp
func (b *Bitstamp) Setup(exch *config.ExchangeConfig) error {
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
			DefaultURL:                       bitstampWSURL,
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

	return nil
}

// Start starts the Bitstamp go routine
func (b *Bitstamp) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitstamp wrapper
func (b *Bitstamp) Run() {
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
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitstamp) FetchTradablePairs(asset asset.Item) ([]string, error) {
	pairs, err := b.GetTradingPairs()
	if err != nil {
		return nil, err
	}

	var products []string
	for x := range pairs {
		if pairs[x].Trading != "Enabled" {
			continue
		}

		pair := strings.Split(pairs[x].Name, "/")
		products = append(products, pair[0]+pair[1])
	}

	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitstamp) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitstamp) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	tick, err := b.GetTicker(p.String(), false)
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice = &ticker.Price{
		Last:        tick.Last,
		High:        tick.High,
		Low:         tick.Low,
		Bid:         tick.Bid,
		Ask:         tick.Ask,
		Volume:      tick.Volume,
		Open:        tick.Open,
		Pair:        p,
		LastUpdated: time.Unix(tick.Timestamp, 0),
	}

	err = ticker.ProcessTicker(b.Name, tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitstamp) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitstamp) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!b.AllowAuthenticatedRequest() || b.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitstamp) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := b.GetOrderbook(p.String())
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

	orderBook.Pair = p
	orderBook.ExchangeName = b.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Bitstamp exchange
func (b *Bitstamp) UpdateAccountInfo() (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = b.Name
	accountBalance, err := b.GetBalance()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for k, v := range accountBalance {
		currencies = append(currencies, account.Balance{
			CurrencyName: currency.NewCode(k),
			TotalValue:   v.Available,
			Hold:         v.Reserved,
		})
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
func (b *Bitstamp) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name)
	if err != nil {
		return b.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitstamp) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *Bitstamp) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Bitstamp) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	buy := s.Side == order.Buy
	market := s.Type == order.Market
	response, err := b.PlaceOrder(s.Pair.String(),
		s.Price,
		s.Amount,
		buy,
		market)
	if err != nil {
		return submitOrderResponse, err
	}
	if response.ID > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.ID, 10)
	}

	submitOrderResponse.IsOrderPlaced = true
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitstamp) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitstamp) CancelOrder(order *order.Cancel) error {
	orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
	if err != nil {
		return err
	}
	_, err = b.CancelExistingOrder(orderIDInt)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitstamp) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	success, err := b.CancelAllExistingOrders()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	if !success {
		err = errors.New("cancel all orders failed. Bitstamp provides no further information. Check order status to verify")
	}

	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns information on a current open order
func (b *Bitstamp) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitstamp) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return b.GetCryptoDepositAddress(cryptocurrency)
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitstamp) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	resp, err := b.CryptoWithdrawal(withdrawRequest.Amount,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.AddressTag,
		true)
	if err != nil {
		return nil, err
	}
	if len(resp.Error) != 0 {
		var details strings.Builder
		for x := range resp.Error {
			details.WriteString(strings.Join(resp.Error[x], ""))
		}
		return nil, errors.New(details.String())
	}

	return &withdraw.ExchangeResponse{
		ID: resp.ID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	resp, err := b.OpenBankWithdrawal(withdrawRequest.Amount,
		withdrawRequest.Currency.String(),
		withdrawRequest.Fiat.Bank.AccountName,
		withdrawRequest.Fiat.Bank.IBAN,
		withdrawRequest.Fiat.Bank.SWIFTCode,
		withdrawRequest.Fiat.Bank.BankAddress,
		withdrawRequest.Fiat.Bank.BankPostalCode,
		withdrawRequest.Fiat.Bank.BankPostalCity,
		withdrawRequest.Fiat.Bank.BankCountry,
		withdrawRequest.Description,
		sepaWithdrawal)
	if err != nil {
		return nil, err
	}
	if resp.Status == errStr {
		var details strings.Builder
		for x := range resp.Reason {
			details.WriteString(strings.Join(resp.Reason[x], ""))
		}
		return nil, errors.New(details.String())
	}

	return &withdraw.ExchangeResponse{
		ID:     resp.ID,
		Status: resp.Status,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	resp, err := b.OpenInternationalBankWithdrawal(withdrawRequest.Amount,
		withdrawRequest.Currency.String(),
		withdrawRequest.Fiat.Bank.AccountName,
		withdrawRequest.Fiat.Bank.IBAN,
		withdrawRequest.Fiat.Bank.SWIFTCode,
		withdrawRequest.Fiat.Bank.BankAddress,
		withdrawRequest.Fiat.Bank.BankPostalCode,
		withdrawRequest.Fiat.Bank.BankPostalCity,
		withdrawRequest.Fiat.Bank.BankCountry,
		withdrawRequest.Fiat.IntermediaryBankName,
		withdrawRequest.Fiat.IntermediaryBankAddress,
		withdrawRequest.Fiat.IntermediaryBankPostalCode,
		withdrawRequest.Fiat.IntermediaryBankCity,
		withdrawRequest.Fiat.IntermediaryBankCountry,
		withdrawRequest.Fiat.WireCurrency,
		withdrawRequest.Description,
		internationalWithdrawal)
	if err != nil {
		return nil, err
	}
	if resp.Status == errStr {
		var details strings.Builder
		for x := range resp.Reason {
			details.WriteString(strings.Join(resp.Reason[x], ""))
		}
		return nil, errors.New(details.String())
	}

	return &withdraw.ExchangeResponse{
		ID:     resp.ID,
		Status: resp.Status,
	}, nil
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *Bitstamp) GetWebsocket() (*wshandler.Websocket, error) {
	return b.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitstamp) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var currPair string
	if len(req.Pairs) != 1 {
		currPair = "all"
	} else {
		currPair = req.Pairs[0].String()
	}

	resp, err := b.GetOpenOrders(currPair)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		orderSide := order.Buy
		if resp[i].Type == SellOrder {
			orderSide = order.Sell
		}

		tm, err := parseTime(resp[i].DateTime)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetActiveOrders unable to parse time: %s\n", b.Name, err)
		}

		orders = append(orders, order.Detail{
			Amount:   resp[i].Amount,
			ID:       strconv.FormatInt(resp[i].ID, 10),
			Price:    resp[i].Price,
			Type:     order.Limit,
			Side:     orderSide,
			Date:     tm,
			Pair:     currency.NewPairFromString(resp[i].Currency),
			Exchange: b.Name,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitstamp) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	var currPair string
	if len(req.Pairs) == 1 {
		currPair = req.Pairs[0].String()
	}
	resp, err := b.GetUserTransactions(currPair)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		if resp[i].Type != MarketTrade {
			continue
		}
		var quoteCurrency, baseCurrency currency.Code

		switch {
		case resp[i].BTC > 0:
			baseCurrency = currency.BTC
		case resp[i].XRP > 0:
			baseCurrency = currency.XRP
		default:
			log.Warnf(log.ExchangeSys,
				"%s No base currency found for ID '%d'\n",
				b.Name,
				resp[i].OrderID)
		}

		switch {
		case resp[i].USD > 0:
			quoteCurrency = currency.USD
		case resp[i].EUR > 0:
			quoteCurrency = currency.EUR
		default:
			log.Warnf(log.ExchangeSys,
				"%s No quote currency found for orderID '%d'\n",
				b.Name,
				resp[i].OrderID)
		}

		var currPair currency.Pair
		if quoteCurrency.String() != "" && baseCurrency.String() != "" {
			currPair = currency.NewPairWithDelimiter(baseCurrency.String(),
				quoteCurrency.String(),
				b.GetPairFormat(asset.Spot, false).Delimiter)
		}

		tm, err := parseTime(resp[i].Date)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetOrderHistory unable to parse time: %s\n", b.Name, err)
		}

		orders = append(orders, order.Detail{
			ID:       strconv.FormatInt(resp[i].OrderID, 10),
			Date:     tm,
			Exchange: b.Name,
			Pair:     currPair,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *Bitstamp) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *Bitstamp) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (b *Bitstamp) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return b.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Bitstamp) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Bitstamp) ValidateCredentials() error {
	_, err := b.UpdateAccountInfo()
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitstamp) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
