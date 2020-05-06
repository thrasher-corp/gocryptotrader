package btse

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
func (b *BTSE) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets the basic defaults for BTSE
func (b *BTSE) SetDefaults() {
	b.Name = "BTSE"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
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
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				OrderbookFetching: true,
				TradeFetching:     true,
				Subscribe:         true,
				Unsubscribe:       true,
				GetOrders:         true,
				GetOrder:          true,
			},
			WithdrawPermissions: exchange.NoAPIWithdrawalMethods,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = btseAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.Websocket = wshandler.New()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *BTSE) Setup(exch *config.ExchangeConfig) error {
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
			DefaultURL:                       btseWebsocket,
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

	b.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		false,
		false,
		false,
		false,
		exch.Name)
	return nil
}

// Start starts the BTSE go routine
func (b *BTSE) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTSE wrapper
func (b *BTSE) Run() {
	if b.Verbose {
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s Failed to update tradable pairs. Error: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *BTSE) FetchTradablePairs(asset asset.Item) ([]string, error) {
	m, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range m {
		if m[x].Status != "active" {
			continue
		}
		currencies = append(currencies, m[x].Symbol)
	}
	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *BTSE) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTSE) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)

	t, err := b.GetTicker(b.FormatExchangeCurrency(p,
		assetType).String())
	if err != nil {
		return tickerPrice, err
	}

	s, err := b.GetMarketStatistics(b.FormatExchangeCurrency(p,
		assetType).String())
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Pair = p
	tickerPrice.Ask = t.Ask
	tickerPrice.Bid = t.Bid
	tickerPrice.Low = s.Low
	tickerPrice.Last = t.Price
	tickerPrice.Volume = s.Volume
	tickerPrice.High = s.High
	tickerPrice.LastUpdated = s.Time

	err = ticker.ProcessTicker(b.Name, tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *BTSE) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *BTSE) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTSE) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	a, err := b.FetchOrderBook(b.FormatExchangeCurrency(p, assetType).String())
	if err != nil {
		return orderBook, err
	}
	for x := range a.BuyQuote {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Price:  a.BuyQuote[x].Price,
			Amount: a.BuyQuote[x].Size})
	}
	for x := range a.SellQuote {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Price:  a.SellQuote[x].Price,
			Amount: a.SellQuote[x].Size})
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
// BTSE exchange
func (b *BTSE) UpdateAccountInfo() (account.Holdings, error) {
	var a account.Holdings
	balance, err := b.GetAccountBalance()
	if err != nil {
		return a, err
	}

	var currencies []account.Balance
	for b := range balance {
		currencies = append(currencies,
			account.Balance{
				CurrencyName: currency.NewCode(balance[b].Currency),
				TotalValue:   balance[b].Total,
				Hold:         balance[b].Available,
			},
		)
	}
	a.Exchange = b.Name
	a.Accounts = []account.SubAccount{
		{
			Currencies: currencies,
		},
	}

	err = account.Process(&a)
	if err != nil {
		return account.Holdings{}, err
	}

	return a, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *BTSE) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name)
	if err != nil {
		return b.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTSE) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTSE) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTSE) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	r, err := b.CreateOrder(s.Amount,
		s.Price,
		s.Side.String(),
		s.Type.String(),
		b.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
		goodTillCancel,
		s.ClientID)
	if err != nil {
		return resp, err
	}

	if *r != "" {
		resp.IsOrderPlaced = true
		resp.OrderID = *r
	}
	if s.Type == order.Market {
		resp.FullyMatched = true
	}
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTSE) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTSE) CancelOrder(order *order.Cancel) error {
	r, err := b.CancelExistingOrder(order.ID,
		b.FormatExchangeCurrency(order.Pair,
			asset.Spot).String())
	if err != nil {
		return err
	}

	switch r.Code {
	case -1:
		return errors.New("order cancellation unsuccessful")
	case 4:
		return errors.New("order cancellation timeout")
	}

	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
// If product ID is sent, all orders of that specified market will be cancelled
// If not specified, all orders of all markets will be cancelled
func (b *BTSE) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	markets, err := b.GetMarkets()
	if err != nil {
		return resp, err
	}

	resp.Status = make(map[string]string)
	for x := range markets {
		strPair := b.FormatExchangeCurrency(orderCancellation.Pair,
			orderCancellation.AssetType).String()
		checkPair := currency.NewPairWithDelimiter(markets[x].BaseCurrency,
			markets[x].QuoteCurrency,
			b.GetPairFormat(asset.Spot, false).Delimiter).String()
		if strPair != "" && strPair != checkPair {
			continue
		} else {
			orders, err := b.GetOrders(checkPair)
			if err != nil {
				return resp, err
			}
			for y := range orders {
				success := "Order Cancelled"
				_, err = b.CancelExistingOrder(orders[y].Order.ID, checkPair)
				if err != nil {
					success = "Order Cancellation Failed"
				}
				resp.Status[orders[y].Order.ID] = success
			}
		}
	}
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTSE) GetOrderInfo(orderID string) (order.Detail, error) {
	o, err := b.GetOrders("")
	if err != nil {
		return order.Detail{}, err
	}

	var od order.Detail
	if len(o) == 0 {
		return od, errors.New("no orders found")
	}

	for i := range o {
		if o[i].ID != orderID {
			continue
		}

		var side = order.Buy
		if strings.EqualFold(o[i].Side, order.Ask.String()) {
			side = order.Sell
		}

		od.Pair = currency.NewPairDelimiter(o[i].Symbol,
			b.GetPairFormat(asset.Spot, false).Delimiter)
		od.Exchange = b.Name
		od.Amount = o[i].Amount
		od.ID = o[i].ID
		od.Date, err = parseOrderTime(o[i].CreatedAt)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetOrderInfo unable to parse time: %s\n", b.Name, err)
		}
		od.Side = side
		od.Type = order.Type(strings.ToUpper(o[i].Type))
		od.Price = o[i].Price
		od.Status = order.Status(o[i].Status)

		fills, err := b.GetFills(orderID, "", "", "", "", "")
		if err != nil {
			return od,
				fmt.Errorf("unable to get order fills for orderID %s",
					orderID)
		}

		for i := range fills {
			createdAt, err := parseOrderTime(fills[i].CreatedAt)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s GetOrderInfo unable to parse time: %s\n", b.Name, err)
			}
			od.Trades = append(od.Trades, order.TradeHistory{
				Timestamp: createdAt,
				TID:       strconv.FormatInt(fills[i].ID, 10),
				Price:     fills[i].Price,
				Amount:    fills[i].Amount,
				Exchange:  b.Name,
				Side:      order.Side(fills[i].Side),
				Fee:       fills[i].Fee,
			})
		}
	}
	return od, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTSE) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTSE) GetWebsocket() (*wshandler.Websocket, error) {
	return b.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTSE) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	resp, err := b.GetOrders("")
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		var side = order.Buy
		if strings.EqualFold(resp[i].Side, order.Ask.String()) {
			side = order.Sell
		}

		tm, err := parseOrderTime(resp[i].CreatedAt)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetActiveOrders unable to parse time: %s\n",
				b.Name,
				err)
		}

		openOrder := order.Detail{
			Pair: currency.NewPairDelimiter(resp[i].Symbol,
				b.GetPairFormat(asset.Spot, false).Delimiter),
			Exchange: b.Name,
			Amount:   resp[i].Amount,
			ID:       resp[i].ID,
			Date:     tm,
			Side:     side,
			Type:     order.Type(strings.ToUpper(resp[i].Type)),
			Price:    resp[i].Price,
			Status:   order.Status(resp[i].Status),
		}

		fills, err := b.GetFills(resp[i].ID, "", "", "", "", "")
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s: Unable to get order fills for orderID %s",
				b.Name,
				resp[i].ID)
			continue
		}

		for i := range fills {
			createdAt, err := parseOrderTime(fills[i].CreatedAt)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s GetActiveOrders unable to parse time: %s\n",
					b.Name,
					err)
			}
			openOrder.Trades = append(openOrder.Trades, order.TradeHistory{
				Timestamp: createdAt,
				TID:       strconv.FormatInt(fills[i].ID, 10),
				Price:     fills[i].Price,
				Amount:    fills[i].Amount,
				Exchange:  b.Name,
				Side:      order.Side(fills[i].Side),
				Fee:       fills[i].Fee,
			})
		}
		orders = append(orders, openOrder)
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTSE) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTSE) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (b *BTSE) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (b *BTSE) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	b.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (b *BTSE) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return b.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *BTSE) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *BTSE) ValidateCredentials() error {
	_, err := b.UpdateAccountInfo()
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *BTSE) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
