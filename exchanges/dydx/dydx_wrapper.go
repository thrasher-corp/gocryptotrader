package dydx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
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
func (dy *DYDX) GetDefaultConfig() (*config.Exchange, error) {
	dy.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = dy.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = dy.BaseCurrencies

	dy.SetupDefaults(exchCfg)

	if dy.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := dy.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Dydx
func (dy *DYDX) SetDefaults() {
	dy.Name = "Dydx"
	dy.Enabled = true
	dy.Verbose = true
	dy.API.CredentialsValidator.RequiresKey = true
	dy.API.CredentialsValidator.RequiresSecret = true

	// If using only one pair format for request and configuration, across all
	// supported asset types either SPOT and FUTURES etc. You can use the
	// example below:

	// Request format denotes what the pair as a string will be, when you send
	// a request to an exchange.
	requestFmt := &currency.PairFormat{ /*Set pair request formatting details here for e.g.*/ Uppercase: true, Delimiter: ":"}
	// Config format denotes what the pair as a string will be, when saved to
	// the config.json file.
	configFmt := &currency.PairFormat{ /*Set pair request formatting details here*/ }
	err := dy.SetGlobalPairsManager(requestFmt, configFmt /*multiple assets can be set here using the asset package ie asset.Spot*/)
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

	err = dy.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = dy.StoreAssetPairFormat(asset.Margin, fmt2)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	dy.Features = exchange.Features{
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
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.OneDay.Word():     true,
				},
				ResultLimit: 200,
			},
		},
	}
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	dy.Requester, err = request.New(dy.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// NOTE: SET THE URLs HERE
	dy.API.Endpoints = dy.NewEndpoints()
	dy.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      dydxAPIURL,
		exchange.WebsocketSpot: dydxWSAPIURL,
	})
	dy.Websocket = stream.New()
	dy.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	dy.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	dy.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (dy *DYDX) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		dy.SetEnabled(false)
		return nil
	}
	err = dy.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := dy.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	// If websocket is supported, please fill out the following

	err = dy.Websocket.Setup(
		&stream.WebsocketSetup{
			ExchangeConfig:        exch,
			DefaultURL:            dydxWSAPIURL,
			RunningURL:            wsRunningEndpoint,
			Connector:             dy.WsConnect,
			Subscriber:            dy.Subscribe,
			Unsubscriber:          dy.Unsubscribe,
			GenerateSubscriptions: dy.GenerateDefaultSubscriptions,
			Features:              &dy.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}

	dy.Websocket.Conn = &stream.WebsocketConnection{
		ExchangeName: dy.Name,
		URL:          dy.Websocket.GetWebsocketURL(),
		ProxyURL:     dy.Websocket.GetProxyAddress(),
		Verbose:      dy.Verbose,
		// ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit: exch.WebsocketResponseMaxLimit,
	}
	return nil
}

// Start starts the Dydx go routine
func (dy *DYDX) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		dy.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Dydx wrapper
func (dy *DYDX) Run() {
	if dy.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			dy.Name,
			common.IsEnabled(dy.Websocket.IsEnabled()))
		dy.PrintEnabledPairs()
	}

	if !dy.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := dy.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			dy.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (dy *DYDX) FetchTradablePairs(ctx context.Context, _ asset.Item) ([]currency.Pair, error) {
	instruments, err := dy.GetMarkets(ctx, "")
	if err != nil {
		return nil, err
	}
	pairs := make(currency.Pairs, len(instruments.Markets))
	count := 0
	for key, _ := range instruments.Markets {
		cp, err := currency.NewPairFromString(key)
		if err != nil {
			return nil, err
		}
		pairs[count] = cp
		count++
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (dy *DYDX) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := dy.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	return dy.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (dy *DYDX) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	// NOTE: EXAMPLE FOR GETTING TICKER PRICE
	/*
		tickerPrice := new(ticker.Price)
		tick, err := dy.GetTicker(p.String())
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
		err = ticker.ProcessTicker(dy.Name, tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	*/
	return ticker.GetTicker(dy.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (dy *DYDX) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	// NOTE: EXAMPLE FOR GETTING TICKER PRICE
	/*
			tick, err := dy.GetTickers()
			if err != nil {
				return err
			}
		    for y := range tick {
		        cp, err := currency.NewPairFromString(tick[y].Symbol)
		        if err != nil {
		            return err
		        }
		        err = ticker.ProcessTicker(&ticker.Price{
		            Last:         tick[y].LastPrice,
		            High:         tick[y].HighPrice,
		            Low:          tick[y].LowPrice,
		            Bid:          tick[y].BidPrice,
		            Ask:          tick[y].AskPrice,
		            Volume:       tick[y].Volume,
		            QuoteVolume:  tick[y].QuoteVolume,
		            Open:         tick[y].OpenPrice,
		            Close:        tick[y].PrevClosePrice,
		            Pair:         cp,
		            ExchangeName: b.Name,
		            AssetType:    assetType,
		        })
		        if err != nil {
		            return err
		        }
		    }
	*/
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (dy *DYDX) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(dy.Name, p, assetType)
	if err != nil {
		return dy.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (dy *DYDX) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(dy.Name, pair, assetType)
	if err != nil {
		return dy.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (dy *DYDX) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        dy.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: dy.CanVerifyOrderbook,
	}

	// NOTE: UPDATE ORDERBOOK EXAMPLE
	/*
		orderbookNew, err := dy.GetOrderBook(exchange.FormatExchangeCurrency(dy.Name, p).String(), 1000)
		if err != nil {
			return book, err
		}

		book.Bids = make([]orderbook.Item, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			book.Bids[x] = orderbook.Item{
				Amount: orderbookNew.Bids[x].Quantity,
				Price: orderbookNew.Bids[x].Price,
			}
		}

		book.Asks = make([]orderbook.Item, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			book.Asks[x] = orderbook.Item{
				Amount: orderBookNew.Asks[x].Quantity,
				Price: orderBookNew.Asks[x].Price,
			}
		}
	*/

	err := book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(dy.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (dy *DYDX) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	// If fetching requires more than one asset type please set
	// HasAssetTypeAccountSegregation to true in RESTCapabilities above.
	return account.Holdings{}, common.ErrNotYetImplemented
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (dy *DYDX) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	// Example implementation below:
	// 	creds, err := dy.GetCredentials(ctx)
	// 	if err != nil {
	// 		return account.Holdings{}, err
	// 	}
	// 	acc, err := account.GetHoldings(dy.Name, creds, assetType)
	// 	if err != nil {
	// 		return dy.UpdateAccountInfo(ctx, assetType)
	// 	}
	// 	return acc, nil
	return account.Holdings{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (dy *DYDX) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (dy *DYDX) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (dy *DYDX) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (dy *DYDX) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (dy *DYDX) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	// When an order has been submitted you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.SubmitResponse if you think they are applicable.
	// resp, err := s.DeriveSubmitResponse( /*newOrderID*/)
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.Date = exampleTime // e.g. If this is supplied by the exchanges API.
	// return resp, nil
	return nil, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (dy *DYDX) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	// When an order has been modified you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.ModifyResponse if you think they are applicable.
	// resp, err := action.DeriveModifyResponse()
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.OrderID = maybeANewOrderID // e.g. If this is supplied by the exchanges API.
	return nil, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (dy *DYDX) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (dy *DYDX) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (dy *DYDX) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (dy *DYDX) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (dy *DYDX) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (dy *DYDX) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (dy *DYDX) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (dy *DYDX) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (dy *DYDX) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (dy *DYDX) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (dy *DYDX) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (dy *DYDX) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := dy.UpdateAccountInfo(ctx, assetType)
	return dy.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (dy *DYDX) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (dy *DYDX) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
