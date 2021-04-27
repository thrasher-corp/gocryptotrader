package opendax

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
func (op *Opendax) GetDefaultConfig() (*config.ExchangeConfig, error) {
	op.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = op.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = op.BaseCurrencies

	op.SetupDefaults(exchCfg)

	if op.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := op.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Opendax
func (op *Opendax) SetDefaults() {
	op.Name = "Opendax"
	op.Enabled = true
	op.Verbose = true
	op.API.CredentialsValidator.RequiresKey = true
	op.API.CredentialsValidator.RequiresSecret = true

	// If using only one pair format for request and configuration, across all
	// supported asset types either SPOT and FUTURES etc. You can use the
	// example below:

	// Request format denotes what the pair as a string will be, when you send
	// a request to an exchange.
	requestFmt := &currency.PairFormat{ /*Set pair request formatting details here for e.g.*/ Uppercase: true, Delimiter: ":"}
	// Config format denotes what the pair as a string will be, when saved to
	// the config.json file.
	configFmt := &currency.PairFormat{ /*Set pair request formatting details here*/ }
	err := op.SetGlobalPairsManager(requestFmt, configFmt /*multiple assets can be set here using the asset package ie asset.Spot*/)
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

	err = op.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = op.StoreAssetPairFormat(asset.Margin, fmt2)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	op.Features = exchange.Features{
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
	op.Requester = request.New(op.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	// NOTE: SET THE URLs HERE
	op.API.Endpoints = op.NewEndpoints()
	op.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: opendaxAPIURL,
		// exchange.WebsocketSpot: opendaxWSAPIURL,
	})
	op.Websocket = stream.New()
	op.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	op.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	op.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (op *Opendax) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		op.SetEnabled(false)
		return nil
	}

	op.SetupDefaults(exch)

	/*
		wsRunningEndpoint, err := op.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			return err
		}

		// If websocket is supported, please fill out the following

		err = op.Websocket.Setup(
			&stream.WebsocketSetup{
				Enabled:                          exch.Features.Enabled.Websocket,
				Verbose:                          exch.Verbose,
				AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
				WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
				DefaultURL:                       opendaxWSAPIURL,
				ExchangeName:                     exch.Name,
				RunningURL:                       wsRunningEndpoint,
				Connector:                        op.WsConnect,
				Subscriber:                       op.Subscribe,
				UnSubscriber:                     op.Unsubscribe,
				Features:                         &op.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		op.WebsocketConn = &stream.WebsocketConnection{
			ExchangeName:         op.Name,
			URL:                  op.Websocket.GetWebsocketURL(),
			ProxyURL:             op.Websocket.GetProxyAddress(),
			Verbose:              op.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}

		// NOTE: PLEASE ENSURE YOU SET THE ORDERBOOK BUFFER SETTINGS CORRECTLY
		op.Websocket.Orderbook.Setup(
			exch.OrderbookConfig.WebsocketBufferLimit,
			true,
			true,
			false,
			false,
			exch.Name)
	*/
	return nil
}

// Start starts the Opendax go routine
func (op *Opendax) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		op.Run()
		wg.Done()
	}()
}

// Run implements the Opendax wrapper
func (op *Opendax) Run() {
	if op.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			op.Name,
			common.IsEnabled(op.Websocket.IsEnabled()))
		op.PrintEnabledPairs()
	}

	if !op.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := op.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			op.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (op *Opendax) FetchTradablePairs(asset asset.Item) ([]string, error) {
	// Implement fetching the exchange available pairs if supported
	return nil, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (op *Opendax) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := op.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return op.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (op *Opendax) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	// NOTE: EXAMPLE FOR GETTING TICKER PRICE
	/*
		tickerPrice := new(ticker.Price)
		tick, err := op.GetTicker(p.String())
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
		err = ticker.ProcessTicker(op.Name, tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	*/
	return ticker.GetTicker(op.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (op *Opendax) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(op.Name, p, assetType)
	if err != nil {
		return op.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (op *Opendax) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(op.Name, currency, assetType)
	if err != nil {
		return op.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (op *Opendax) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        op.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: op.CanVerifyOrderbook,
	}

	// NOTE: UPDATE ORDERBOOK EXAMPLE
	/*
		orderbookNew, err := op.GetOrderBook(exchange.FormatExchangeCurrency(op.Name, p).String(), 1000)
		if err != nil {
			return book, err
		}

		for x := range orderbookNew.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: orderbookNew.Bids[x].Quantity,
				Price: orderbookNew.Bids[x].Price,
			})
		}

		for x := range orderbookNew.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: orderBookNew.Asks[x].Quantity,
				Price: orderBookNew.Asks[x].Price,
			})
		}
	*/

	err := book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(op.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (op *Opendax) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	return account.Holdings{}, common.ErrNotYetImplemented
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (op *Opendax) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	return account.Holdings{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (op *Opendax) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (op *Opendax) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (op *Opendax) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (op *Opendax) GetHistoricTrades(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (op *Opendax) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (op *Opendax) ModifyOrder(action *order.Modify) (string, error) {
	// if err := action.Validate(); err != nil {
	// 	return "", err
	// }
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (op *Opendax) CancelOrder(ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (op *Opendax) CancelBatchOrders(orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (op *Opendax) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (op *Opendax) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (op *Opendax) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (op *Opendax) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (op *Opendax) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (op *Opendax) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (op *Opendax) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (op *Opendax) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (op *Opendax) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (op *Opendax) ValidateCredentials(assetType asset.Item) error {
	_, err := op.UpdateAccountInfo(assetType)
	return op.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (op *Opendax) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (op *Opendax) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
