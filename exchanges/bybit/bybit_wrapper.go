package bybit

import (
	"fmt"
	"strconv"
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
func (by *Bybit) GetDefaultConfig() (*config.ExchangeConfig, error) {
	by.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = by.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = by.BaseCurrencies

	err := by.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if by.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := by.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Bybit
func (by *Bybit) SetDefaults() {
	by.Name = "Bybit"
	by.Enabled = true
	by.Verbose = true
	by.API.CredentialsValidator.RequiresKey = true
	by.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true}

	configFmt := &currency.PairFormat{Uppercase: true}
	err := by.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.CoinMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.USDTMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.Futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	by.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TradeFetching:     true,
				KlineFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
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
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
				CryptoDepositFee:  true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				TickerFetching:         true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	by.Requester = request.New(by.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))

	by.API.Endpoints = by.NewEndpoints()
	by.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         bybitAPIURL,
		exchange.RestCoinMargined: bybitAPIURL,
		exchange.RestUSDTMargined: bybitAPIURL,
		exchange.RestFutures:      bybitAPIURL,
		exchange.WebsocketSpot:    bybitWSBaseURL + wsSpotPublicTopicV2,
	})
	by.Websocket = stream.New()
	by.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	by.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	by.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (by *Bybit) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		by.SetEnabled(false)
		return nil
	}

	by.SetupDefaults(exch)

	wsRunningEndpoint, err := by.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	// If websocket is supported, please fill out the following
	err = by.Websocket.Setup(
		&stream.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       bybitWSBaseURL + wsSpotPublicTopicV2,
			ExchangeName:                     exch.Name,
			RunningURL:                       wsRunningEndpoint,
			Connector:                        by.WsConnect,
			Subscriber:                       by.Subscribe,
			UnSubscriber:                     by.Unsubscribe,
			Features:                         &by.Features.Supports.WebsocketCapabilities,
			OrderbookBufferLimit:             exch.OrderbookConfig.WebsocketBufferLimit,
			BufferEnabled:                    exch.OrderbookConfig.WebsocketBufferEnabled,
			SortBuffer:                       true,
			SortBufferByUpdateIDs:            true,
		})
	if err != nil {
		return err
	}

	return by.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  by.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Bybit go routine
func (by *Bybit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		by.Run()
		wg.Done()
	}()
}

// Run implements the Bybit wrapper
func (by *Bybit) Run() {
	if by.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			by.Name,
			common.IsEnabled(by.Websocket.IsEnabled()))
		by.PrintEnabledPairs()
	}

	if !by.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := by.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			by.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (by *Bybit) FetchTradablePairs(a asset.Item) ([]string, error) {
	if !by.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, by.Name)
	}
	var pairs []string
	switch a {
	case asset.Spot:
		allPairs, err := by.GetAllPairs()
		if err != nil {
			return nil, err
		}
		for x := range allPairs {
			pairs = append(pairs, allPairs[x].Name)
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		allPairs, err := by.GetSymbolsInfo()
		if err != nil {
			return pairs, nil
		}
		for x := range allPairs {
			if allPairs[x].Status == "Trading" {
				pairs = append(pairs, allPairs[x].Name)
			}
		}
	}
	return pairs, nil

}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (by *Bybit) UpdateTradablePairs(forceUpdate bool) error {
	assetTypes := by.GetAssetTypes(false)
	for i := range assetTypes {
		pairs, err := by.FetchTradablePairs(assetTypes[i])
		if err != nil {
			return err
		}

		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}

		err = by.UpdatePairs(p, assetTypes[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (by *Bybit) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	switch assetType {
	case asset.Spot:
		tick, err := by.Get24HrsChange("")
		if err != nil {
			return nil, err
		}

		for y := range tick {
			cp, err := currency.NewPairFromString(tick[y].Symbol)
			if err != nil {
				return nil, err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[y].LastPrice,
				High:         tick[y].HighPrice,
				Low:          tick[y].LowPrice,
				Bid:          tick[y].BestBidPrice,
				Ask:          tick[y].BestAskPrice,
				Volume:       tick[y].Volume,
				QuoteVolume:  tick[y].QuoteVolume,
				Open:         tick[y].OpenPrice,
				Pair:         cp,
				LastUpdated:  tick[y].Time,
				ExchangeName: by.Name,
				AssetType:    assetType})
			if err != nil {
				return nil, err
			}
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		tick, err := by.GetFuturesSymbolPriceTicker(currency.Pair{})
		if err != nil {
			return nil, err
		}

		for y := range tick {
			cp, err := currency.NewPairFromString(tick[y].Symbol)
			if err != nil {
				return nil, err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[y].LastPrice,
				High:         tick[y].HighPrice24h,
				Low:          tick[y].LowPrice24h,
				Bid:          tick[y].BidPrice,
				Ask:          tick[y].AskPrice,
				Volume:       float64(tick[y].Volume24h),
				Open:         tick[y].OpenValue,
				Pair:         cp,
				ExchangeName: by.Name,
				AssetType:    assetType})
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("assetType not supported: %v", assetType)
	}

	return ticker.GetTicker(by.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (by *Bybit) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := by.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(by.Name, fPair, assetType)
	if err != nil {
		return by.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (by *Bybit) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(by.Name, currency, assetType)
	if err != nil {
		return by.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (by *Bybit) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        by.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: by.CanVerifyOrderbook,
	}

	var orderbookNew Orderbook
	var err error
	switch assetType {
	case asset.Spot:
		orderbookNew, err = by.GetOrderBook(p.String(), 0)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		orderbookNew, err = by.GetFuturesOrderbook(p)
	default:
		return nil, fmt.Errorf("assetType not supported: %v", assetType)
	}
	if err != nil {
		return book, err
	}

	for x := range orderbookNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		})
	}

	for x := range orderbookNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(by.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = by.Name
	switch assetType {
	case asset.Spot:
		balances, err := by.GetWalletBalance()
		if err != nil {
			return info, err
		}

		var currencyBalance []account.Balance
		for i := range balances {
			currencyBalance = append(currencyBalance, account.Balance{
				CurrencyName: currency.NewCode(balances[i].CoinName),
				TotalValue:   balances[i].Total,
				Hold:         balances[i].Locked,
			})
		}

		acc.Currencies = currencyBalance

	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		balances, err := by.GetFutureWalletBalance("")
		if err != nil {
			return info, err
		}

		var currencyBalance []account.Balance
		for coinName, data := range balances {
			currencyBalance = append(currencyBalance, account.Balance{
				CurrencyName: currency.NewCode(coinName),
				TotalValue:   data.Equity,
				Hold:         data.Equity - data.AvailableBalance,
			})
		}

		acc.Currencies = currencyBalance

	default:
		return info, fmt.Errorf("%v assetType not supported", assetType)
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	err := account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(by.Name, assetType)
	if err != nil {
		return by.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// TODO: check again
// GetFundingHistory returns funding history, deposits and
// withdrawals
func (by *Bybit) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (by *Bybit) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	w, err := by.GetWalletWithdrawalRecords("", "", "", "", 0, 0)
	if err != nil {
		return nil, err
	}

	for i := range w {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          w[i].Status,
			TransferID:      strconv.FormatInt(w[i].ID, 10),
			Currency:        w[i].Coin,
			Amount:          w[i].Amount,
			Fee:             w[i].Fee,
			CryptoToAddress: w[i].Address,
			CryptoTxID:      w[i].TxID,
			Timestamp:       w[i].UpdatedAt,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (by *Bybit) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (by *Bybit) GetHistoricTrades(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (by *Bybit) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	return submitOrderResponse, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (by *Bybit) ModifyOrder(action *order.Modify) (string, error) {
	// if err := action.Validate(); err != nil {
	// 	return "", err
	// }
	return "", common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (by *Bybit) CancelOrder(ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (by *Bybit) CancelBatchOrders(orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (by *Bybit) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (by *Bybit) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (by *Bybit) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (by *Bybit) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (by *Bybit) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (by *Bybit) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (by *Bybit) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (by *Bybit) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (by *Bybit) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (by *Bybit) ValidateCredentials(assetType asset.Item) error {
	_, err := by.UpdateAccountInfo(assetType)
	return by.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (by *Bybit) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (by *Bybit) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
