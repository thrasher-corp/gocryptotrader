package testexch

import (
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (te *Testexch) GetDefaultConfig() (*config.ExchangeConfig, error) {
	te.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = te.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = te.BaseCurrencies

	te.SetupDefaults(exchCfg)

	if te.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := te.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Testexch
func (te *Testexch) SetDefaults() {
	te.Name = "Testexch"
	te.Enabled = true
	te.Verbose = true
	te.API.CredentialsValidator.RequiresKey = true
	te.API.CredentialsValidator.RequiresSecret = true

	// If using only one pair format for request and configuration, across all
	// supported asset types either SPOT and FUTURES etc. You can use the
	// example below:

	// Request format denotes what the pair as a string will be, when you send
	// a request to an exchange.
	requestFmt := &currency.PairFormat{ /*Set pair request formatting details here for e.g.*/ Uppercase: true, Delimiter: ":"}
	// Config format denotes what the pair as a string will be, when saved to
	// the config.json file.
	configFmt := &currency.PairFormat{ /*Set pair request formatting details here*/ }
	err := te.SetGlobalPairsManager(requestFmt, configFmt /*multiple assets can be set here using the asset package ie asset.Spot*/)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// If assets require multiple differences in formating for request and
	// configuration, another exchange method can be be used e.g. futures
	// contracts require a dash as a delimiter rather than an underscore. You
	// can use this example below:

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
	}

	fmt2 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: ":"},
	}

	err = te.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = te.StoreAssetPairFormat(asset.Margin, fmt2)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	te.Features = exchange.Features{
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
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	te.Requester = request.New(te.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	// NOTE: SET THE URLs HERE
	te.API.Endpoints = te.NewEndpoints()
	te.API.Endpoints.CreateMap(map[exchange.URL]string{
		exchange.RestSpot: testexchAPIURL,
		// exchange.WebsocketSpot: testexchWSAPIURL,
	})
	te.Websocket = stream.New()
	te.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	te.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	te.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (te *Testexch) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		te.SetEnabled(false)
		return nil
	}

	te.SetupDefaults(exch)

	/*
		wsRunningEndpoint, err := te.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			return err
		}

		// If websocket is supported, please fill out the following

		err = te.Websocket.Setup(
			&stream.WebsocketSetup{
				Enabled:                          exch.Features.Enabled.Websocket,
				Verbose:                          exch.Verbose,
				AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
				WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
				DefaultURL:                       testexchWSAPIURL,
				ExchangeName:                     exch.Name,
				RunningURL:                       wsRunningEndpoint,
				Connector:                        te.WsConnect,
				Subscriber:                       te.Subscribe,
				UnSubscriber:                     te.Unsubscribe,
				Features:                         &te.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		te.WebsocketConn = &stream.WebsocketConnection{
			ExchangeName:         te.Name,
			URL:                  te.Websocket.GetWebsocketURL(),
			ProxyURL:             te.Websocket.GetProxyAddress(),
			Verbose:              te.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}

		// NOTE: PLEASE ENSURE YOU SET THE ORDERBOOK BUFFER SETTINGS CORRECTLY
		te.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			false,
			false,
			exch.Name)
	*/
	return nil
}

// Start starts the Testexch go routine
func (te *Testexch) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		te.Run()
		wg.Done()
	}()
}

// Run implements the Testexch wrapper
func (te *Testexch) Run() {
	if te.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			te.Name,
			common.IsEnabled(te.Websocket.IsEnabled()))
		te.PrintEnabledPairs()
	}

	if !te.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := te.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			te.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (te *Testexch) FetchTradablePairs(asset asset.Item) ([]string, error) {
	// Implement fetching the exchange available pairs if supported
	return nil, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (te *Testexch) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := te.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return te.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (te *Testexch) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	// NOTE: EXAMPLE FOR GETTING TICKER PRICE
	/*
		tickerPrice := new(ticker.Price)
		tick, err := te.GetTicker(p.String())
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice = &ticker.Price{
			High:    tick.High,
			Low:     tick.Low,
			Bid:     tick.Bid,
			Ask:     tick.Ask,
			Open:    tick.Open,
			Close:   tick.Close,
			Pair:    p,
		}
		err = ticker.ProcessTicker(te.Name, tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	*/
	return ticker.GetTicker(te.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (te *Testexch) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(te.Name, p, assetType)
	if err != nil {
		return te.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (te *Testexch) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(te.Name, currency, assetType)
	if err != nil {
		return te.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (te *Testexch) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	// NOTE: UPDATE ORDERBOOK EXAMPLE
	/*
		orderbookNew, err := te.GetOrderBook(exchange.FormatExchangeCurrency(te.Name, p).String(), 1000)
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
	orderBook.ExchangeName = te.Name
	orderBook.AssetType = assetType

	err := orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(te.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (te *Testexch) UpdateAccountInfo() (account.Holdings, error) {
	return account.Holdings{}, common.ErrNotYetImplemented
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (te *Testexch) FetchAccountInfo() (account.Holdings, error) {
	return account.Holdings{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (te *Testexch) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (te *Testexch) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (te *Testexch) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (te *Testexch) GetHistoricTrades(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (te *Testexch) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (te *Testexch) ModifyOrder(action *order.Modify) (string, error) {
	// if err := action.Validate(); err != nil {
	// 	return "", err
	// }
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (te *Testexch) CancelOrder(ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (te *Testexch) CancelBatchOrders(orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (te *Testexch) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (te *Testexch) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (te *Testexch) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (te *Testexch) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (te *Testexch) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (te *Testexch) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (te *Testexch) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (te *Testexch) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (te *Testexch) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (te *Testexch) ValidateCredentials() error {
	_, err := te.UpdateAccountInfo()
	return te.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (te *Testexch) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (te *Testexch) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
