package ftx

import (
	"sync"

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
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (f *Ftx) GetDefaultConfig() (*config.ExchangeConfig, error) {
	f.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = f.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = f.BaseCurrencies

	err := f.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if f.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = f.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Ftx
func (f *Ftx) SetDefaults() {
	f.Name = "Ftx"
	f.Enabled = true
	f.Verbose = true
	f.API.CredentialsValidator.RequiresKey = true
	f.API.CredentialsValidator.RequiresSecret = true
	f.CurrencyPairs = currency.PairsManager{
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
	// Fill out the capabilities/features that the exchange supports
	f.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}
	f.Requester = request.New(f.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		nil)
	f.API.Endpoints.URLDefault = ftxAPIURL
	f.API.Endpoints.URL = f.API.Endpoints.URLDefault
	f.Websocket = wshandler.New()
	f.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	f.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	f.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (f *Ftx) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		f.SetEnabled(false)
		return nil
	}

	err := f.SetupDefaults(exch)
	if err != nil {
		return err
	}

	// If websocket is supported, please fill out the following
	/*
		err = f.Websocket.Setup(
			&wshandler.WebsocketSetup{
				Enabled:                          exch.Features.Enabled.Websocket,
				Verbose:                          exch.Verbose,
				AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
				WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
				DefaultURL:                       ftxWSURL,
				ExchangeName:                     exch.Name,
				RunningURL:                       exch.API.Endpoints.WebsocketURL,
				Connector:                        f.WsConnect,
				Subscriber:                       f.Subscribe,
				UnSubscriber:                     f.Unsubscribe,
				Features:                         &f.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		f.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         f.Name,
			URL:                  f.Websocket.GetWebsocketURL(),
			ProxyURL:             f.Websocket.GetProxyAddress(),
			Verbose:              f.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}

		// NOTE: PLEASE ENSURE YOU SET THE ORDERBOOK BUFFER SETTINGS CORRECTLY
		f.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			false,
			false,
			exch.Name)
	*/
	return nil
}

// Start starts the Ftx go routine
func (f *Ftx) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		f.Run()
		wg.Done()
	}()
}

// Run implements the Ftx wrapper
func (f *Ftx) Run() {
	if f.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			f.Name,
			common.IsEnabled(f.Websocket.IsEnabled()))
		f.PrintEnabledPairs()
	}

	if !f.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := f.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			f.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (f *Ftx) FetchTradablePairs(asset asset.Item) ([]string, error) {
	// Implement fetching the exchange available pairs if supported
	return nil, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (f *Ftx) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := f.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}
	return f.UpdatePairs(currency.NewPairsFromStrings(pairs),
		asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (f *Ftx) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	// NOTE: EXAMPLE FOR GETTING TICKER PRICE
	/*
		tickerPrice := new(ticker.Price)
		tick, err := f.GetTicker(p.String())
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice = &ticker.Price{
			High:  tick.High,
			Low:   tick.Low,
			Bid:   tick.Bid,
			Ask:   tick.Ask,
			Open:  tick.Open,
			Close: tick.Close,
			Pair:  p,
		}
		err = ticker.ProcessTicker(f.Name, tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	*/
	return ticker.GetTicker(f.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (f *Ftx) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(f.Name, p, assetType)
	if err != nil {
		return f.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (f *Ftx) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(f.Name, currency, assetType)
	if err != nil {
		return f.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (f *Ftx) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	// NOTE: UPDATE ORDERBOOK EXAMPLE
	/*
		orderbookNew, err := f.GetOrderBook(exchange.FormatExchangeCurrency(f.Name, p).String(), 1000)
		if err != nil {
			return orderBook, err
		}

		for x := range orderbookNew.Bids {
			orderBook.Bids = append(orderBook.Bids, orderbook.Item{
				Amount: orderbookNew.Bids[x].Quantity,
				Price: orderbookNew.Bids[x].Price,
			})
		}

		for x := range orderbookNew.Asks {
			orderBook.Asks = append(orderBook.Asks, orderbook.Item{
				Amount: orderBook.Asks[x].Quantity,
				Price: orderBook.Asks[x].Price,
			})
		}
	*/

	orderBook.Pair = p
	orderBook.ExchangeName = f.Name
	orderBook.AssetType = assetType

	err := orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(f.Name, p, assetType)
}

// // GetAccountInfo retrieves balances for all enabled currencies for the
// // Ftx exchange
// func (f *Ftx) GetAccountInfo() (exchange.AccountInfo, error) {
// 	return exchange.AccountInfo{}, common.ErrNotYetImplemented
// }

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (f *Ftx) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (f *Ftx) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (f *Ftx) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (f *Ftx) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (f *Ftx) CancelOrder(order *order.Cancel) error {
	return common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (f *Ftx) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (f *Ftx) GetOrderInfo(orderID string) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (f *Ftx) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *Ftx) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	var resp *withdraw.ExchangeResponse
	return resp, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *Ftx) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	var resp *withdraw.ExchangeResponse
	return resp, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (f *Ftx) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (f *Ftx) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (f *Ftx) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (f *Ftx) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (f *Ftx) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (f *Ftx) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	f.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (f *Ftx) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	f.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (f *Ftx) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrNotYetImplemented
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (f *Ftx) AuthenticateWebsocket() error {
	return common.ErrNotYetImplemented
}
