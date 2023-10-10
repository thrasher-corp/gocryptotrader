package coinbaseinternational

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (co *CoinbaseInternational) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	co.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = co.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = co.BaseCurrencies

	co.SetupDefaults(exchCfg)

	if co.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := co.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for CoinbaseInternational
func (co *CoinbaseInternational) SetDefaults() {
	co.Name = "CoinbaseInternational"
	co.Enabled = true
	co.Verbose = true
	co.API.CredentialsValidator.RequiresKey = true
	co.API.CredentialsValidator.RequiresSecret = true

	// If using only one pair format for request and configuration, across all
	// supported asset types either SPOT and FUTURES etc. You can use the
	// example below:

	// Request format denotes what the pair as a string will be, when you send
	// a request to an exchange.
	requestFmt := &currency.PairFormat{ /*Set pair request formatting details here for e.g.*/ Uppercase: true, Delimiter: ":"}
	// Config format denotes what the pair as a string will be, when saved to
	// the config.json file.
	configFmt := &currency.PairFormat{ /*Set pair request formatting details here*/ }
	err := co.SetGlobalPairsManager(requestFmt, configFmt /*multiple assets can be set here using the asset package ie asset.Spot*/)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// If assets require multiple differences in formatting for request and
	// configuration, another exchange method can be be used e.g. futures
	// contracts require a dash as a delimiter rather than an underscore. You
	// can use this example below:

	fmt := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}

	err = co.StoreAssetPairFormat(asset.Spot, fmt)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	co.Features = exchange.Features{
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
	co.Requester, err = request.New(co.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// NOTE: SET THE URLs HERE
	co.API.Endpoints = co.NewEndpoints()
	co.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      coinbaseInternationalAPIURL,
		exchange.WebsocketSpot: coinbaseinternationalWSAPIURL,
	})
	co.Websocket = stream.New()
	co.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	co.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	co.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (co *CoinbaseInternational) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		co.SetEnabled(false)
		return nil
	}
	err = co.SetupDefaults(exch)
	if err != nil {
		return err
	}
	wsRunningEndpoint, err := co.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = co.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            coinbaseinternationalWSAPIURL,
		RunningURL:            wsRunningEndpoint,
		Connector:             co.WsConnect,
		Subscriber:            co.Subscribe,
		Unsubscriber:          co.Unsubscribe,
		GenerateSubscriptions: co.GenerateDefaultSubscriptions,
		Features:              &co.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}
	return co.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the CoinbaseInternational go routine
func (co *CoinbaseInternational) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		co.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the CoinbaseInternational wrapper
func (co *CoinbaseInternational) Run(ctx context.Context) {
	if co.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			co.Name,
			common.IsEnabled(co.Websocket.IsEnabled()))
		co.PrintEnabledPairs()
	}

	if !co.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := co.UpdateTradablePairs(ctx, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			co.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (co *CoinbaseInternational) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !co.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	instruments, err := co.GetInstruments(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(instruments))
	for x := range instruments {
		if instruments[x].TradingState != "TRADING" {
			continue
		}
		cp, err := currency.NewPairFromString(instruments[x].Symbol)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, cp)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (co *CoinbaseInternational) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := co.GetAssetTypes(false)
	for x := range assetTypes {
		pairs, err := co.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}

		err = co.UpdatePairs(pairs, assetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (co *CoinbaseInternational) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	// NOTE: EXAMPLE FOR GETTING TICKER PRICE
	/*
		tickerPrice := new(ticker.Price)
		tick, err := co.GetTicker(p.String())
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
		err = ticker.ProcessTicker(co.Name, tickerPrice, assetType)
		if err != nil {
			return tickerPrice, err
		}
	*/
	return ticker.GetTicker(co.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (co *CoinbaseInternational) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	// NOTE: EXAMPLE FOR GETTING TICKER PRICE
	/*
			tick, err := co.GetTickers()
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
func (co *CoinbaseInternational) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(co.Name, p, assetType)
	if err != nil {
		return co.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (co *CoinbaseInternational) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(co.Name, pair, assetType)
	if err != nil {
		return co.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (co *CoinbaseInternational) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        co.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: co.CanVerifyOrderbook,
	}

	// NOTE: UPDATE ORDERBOOK EXAMPLE
	/*
		orderbookNew, err := co.GetOrderBook(exchange.FormatExchangeCurrency(co.Name, p).String(), 1000)
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

	return orderbook.Get(co.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (co *CoinbaseInternational) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	// If fetching requires more than one asset type please set
	// HasAssetTypeAccountSegregation to true in RESTCapabilities above.
	return account.Holdings{}, common.ErrNotYetImplemented
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (co *CoinbaseInternational) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	// Example implementation below:
	// 	creds, err := co.GetCredentials(ctx)
	// 	if err != nil {
	// 		return account.Holdings{}, err
	// 	}
	// 	acc, err := account.GetHoldings(co.Name, creds, assetType)
	// 	if err != nil {
	// 		return co.UpdateAccountInfo(ctx, assetType)
	// 	}
	// 	return acc, nil
	return account.Holdings{}, common.ErrNotYetImplemented
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (co *CoinbaseInternational) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (co *CoinbaseInternational) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (co *CoinbaseInternational) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (co *CoinbaseInternational) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetServerTime returns the current exchange server time.
func (co *CoinbaseInternational) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (co *CoinbaseInternational) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
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
func (co *CoinbaseInternational) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
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
func (co *CoinbaseInternational) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (co *CoinbaseInternational) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (co *CoinbaseInternational) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (co *CoinbaseInternational) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (co *CoinbaseInternational) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (co *CoinbaseInternational) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (co *CoinbaseInternational) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (co *CoinbaseInternational) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (co *CoinbaseInternational) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (co *CoinbaseInternational) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (co *CoinbaseInternational) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateAPICredentials validates current credentials used for wrapper
func (co *CoinbaseInternational) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := co.UpdateAccountInfo(ctx, assetType)
	return co.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (co *CoinbaseInternational) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (co *CoinbaseInternational) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (co *CoinbaseInternational) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}
