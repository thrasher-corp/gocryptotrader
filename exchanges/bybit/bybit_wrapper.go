package bybit

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
func (by *Bybit) GetDefaultConfig() (*config.Exchange, error) {
	by.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = by.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = by.BaseCurrencies

	err := by.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if by.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := by.UpdateTradablePairs(context.TODO(), true)
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

	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
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
				ModifyOrder:       true,
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

	by.Requester, err = request.New(by.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	by.API.Endpoints = by.NewEndpoints()
	err = by.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         bybitAPIURL,
		exchange.RestCoinMargined: bybitAPIURL,
		exchange.RestUSDTMargined: bybitAPIURL,
		exchange.RestFutures:      bybitAPIURL,
		exchange.WebsocketSpot:    bybitWSBaseURL + wsSpotPublicTopicV2,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	by.Websocket = stream.New()
	by.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	by.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	by.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (by *Bybit) Setup(exch *config.Exchange) error {
	if !exch.Enabled {
		by.SetEnabled(false)
		return nil
	}

	err := by.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := by.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = by.Websocket.Setup(
		&stream.WebsocketSetup{
			ExchangeConfig:        exch,
			DefaultURL:            bybitWSBaseURL + wsSpotPublicTopicV2,
			RunningURL:            wsRunningEndpoint,
			RunningURLAuth:        bybitWSBaseURL + wsSpotPrivate,
			Connector:             by.WsConnect,
			Subscriber:            by.Subscribe,
			Unsubscriber:          by.Unsubscribe,
			GenerateSubscriptions: by.GenerateDefaultSubscriptions,
			Features:              &by.Features.Supports.WebsocketCapabilities,
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
			TradeFeed:             by.Features.Enabled.TradeFeed,
		})
	if err != nil {
		return err
	}

	err = by.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  by.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}

	return by.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  bybitWSBaseURL + wsSpotPrivate,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (by *Bybit) AuthenticateWebsocket(_ context.Context) error {
	return by.WsAuth()
}

// Start starts the Bybit go routine
func (by *Bybit) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		by.Run()
		wg.Done()
	}()
	return nil
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

	err := by.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			by.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (by *Bybit) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	if !by.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, by.Name)
	}
	var pairs []string
	switch a {
	case asset.Spot:
		allPairs, err := by.GetAllSpotPairs(ctx)
		if err != nil {
			return nil, err
		}
		for x := range allPairs {
			pairs = append(pairs, allPairs[x].BaseCurrency+currency.DashDelimiter+allPairs[x].QuoteCurrency)
		}
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		allPairs, err := by.GetSymbolsInfo(ctx)
		if err != nil {
			return pairs, err
		}
		for x := range allPairs {
			if allPairs[x].Status == "Trading" {
				pairs = append(pairs, allPairs[x].BaseCurrency+currency.DashDelimiter+allPairs[x].QuoteCurrency)
			}
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (by *Bybit) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := by.GetAssetTypes(false)
	for i := range assetTypes {
		pairs, err := by.FetchTradablePairs(ctx, assetTypes[i])
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

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (by *Bybit) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	allPairs, err := by.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	switch assetType {
	case asset.Spot:
		tick, err := by.Get24HrsChange(ctx, "")
		if err != nil {
			return err
		}
		for p := range allPairs {
			formattedPair, err := by.FormatExchangeCurrency(allPairs[p], assetType)
			if err != nil {
				return err
			}

			for y := range tick {
				if tick[y].Symbol != formattedPair.String() {
					continue
				}

				cp, err := by.extractCurrencyPair(tick[y].Symbol, assetType)
				if err != nil {
					return err
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
					return err
				}
			}
		}

	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		tick, err := by.GetFuturesSymbolPriceTicker(ctx, currency.Pair{})
		if err != nil {
			return err
		}

		for p := range allPairs {
			formattedPair, err := by.FormatExchangeCurrency(allPairs[p], assetType)
			if err != nil {
				return err
			}

			for y := range tick {
				if tick[y].Symbol != formattedPair.String() {
					continue
				}
				cp, err := by.extractCurrencyPair(tick[y].Symbol, assetType)
				if err != nil {
					return err
				}
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         tick[y].LastPrice,
					High:         tick[y].HighPrice24h,
					Low:          tick[y].LowPrice24h,
					Bid:          tick[y].BidPrice,
					Ask:          tick[y].AskPrice,
					Volume:       tick[y].Volume24h,
					Open:         tick[y].OpenValue,
					Pair:         cp,
					ExchangeName: by.Name,
					AssetType:    assetType})
				if err != nil {
					return err
				}
			}
		}

	default:
		return fmt.Errorf("assetType not supported: %v", assetType)
	}

	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (by *Bybit) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	formattedPair, err := by.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	switch assetType {
	case asset.Spot:
		tick, err := by.Get24HrsChange(ctx, formattedPair.String())
		if err != nil {
			return nil, err
		}

		for y := range tick {
			cp, err := by.extractCurrencyPair(tick[y].Symbol, assetType)
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
		tick, err := by.GetFuturesSymbolPriceTicker(ctx, formattedPair)
		if err != nil {
			return nil, err
		}

		for y := range tick {
			cp, err := by.extractCurrencyPair(tick[y].Symbol, assetType)
			if err != nil {
				return nil, err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[y].LastPrice,
				High:         tick[y].HighPrice24h,
				Low:          tick[y].LowPrice24h,
				Bid:          tick[y].BidPrice,
				Ask:          tick[y].AskPrice,
				Volume:       tick[y].Volume24h,
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
func (by *Bybit) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := by.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(by.Name, fPair, assetType)
	if err != nil {
		return by.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (by *Bybit) FetchOrderbook(ctx context.Context, currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(by.Name, currency, assetType)
	if err != nil {
		return by.UpdateOrderbook(ctx, currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (by *Bybit) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	var orderbookNew Orderbook
	var err error

	formattedPair, err := by.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	book := &orderbook.Base{
		Exchange:        by.Name,
		Pair:            formattedPair,
		Asset:           assetType,
		VerifyOrderbook: by.CanVerifyOrderbook,
	}

	switch assetType {
	case asset.Spot:
		orderbookNew, err = by.GetOrderBook(ctx, formattedPair.String(), 0)
	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		orderbookNew, err = by.GetFuturesOrderbook(ctx, formattedPair)
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
	return orderbook.Get(by.Name, formattedPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = by.Name
	switch assetType {
	case asset.Spot:
		balances, err := by.GetWalletBalance(ctx)
		if err != nil {
			return info, err
		}

		currencyBalance := make([]account.Balance, len(balances))
		for i := range balances {
			currencyBalance[i] = account.Balance{
				CurrencyName: currency.NewCode(balances[i].CoinName),
				Total:        balances[i].Total,
				Hold:         balances[i].Locked,
				Free:         balances[i].Total - balances[i].Locked,
			}
		}

		acc.Currencies = currencyBalance

	case asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.Futures:
		balances, err := by.GetFutureWalletBalance(ctx, "")
		if err != nil {
			return info, err
		}

		var i int
		currencyBalance := make([]account.Balance, len(balances))
		for coinName, data := range balances {
			currencyBalance[i] = account.Balance{
				CurrencyName: currency.NewCode(coinName),
				Total:        data.WalletBalance,
				Hold:         data.WalletBalance - data.AvailableBalance,
				Free:         data.AvailableBalance,
			}
			i++
		}

		acc.Currencies = currencyBalance

	default:
		return info, fmt.Errorf("assetType not supported: %v", assetType)
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)

	if err := account.Process(&info); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(by.Name, assetType)
	if err != nil {
		return by.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (by *Bybit) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (by *Bybit) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	w, err := by.GetWalletWithdrawalRecords(ctx, "", "", "", c, 0, 0)
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
func (by *Bybit) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var resp []trade.Data

	switch assetType {
	case asset.Spot:
		tradeData, err := by.GetTrades(ctx, "", 0)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			resp = append(resp, trade.Data{
				Exchange:     by.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Volume,
				Timestamp:    tradeData[i].Time,
			})
		}

	case asset.CoinMarginedFutures, asset.Futures:
		tradeData, err := by.GetPublicTrades(ctx, currency.Pair{}, 0)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			resp = append(resp, trade.Data{
				Exchange:     by.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Qty,
				Timestamp:    tradeData[i].Time,
			})
		}

	case asset.USDTMarginedFutures:
		tradeData, err := by.GetUSDTPublicTrades(ctx, currency.Pair{}, 0)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			resp = append(resp, trade.Data{
				Exchange:     by.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Qty,
				Timestamp:    tradeData[i].Time,
			})
		}

	default:
		return nil, fmt.Errorf("assetType not supported: %v", assetType)
	}

	if by.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(by.Name, resp...)
		if err != nil {
			return nil, err
		}
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (by *Bybit) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (by *Bybit) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	switch s.AssetType {
	case asset.Spot:
		var sideType string
		switch s.Side {
		case order.Buy:
			sideType = sideBuy
		case order.Sell:
			sideType = sideSell
		default:
			return submitOrderResponse, errInvalidSide
		}

		timeInForce := BybitRequestParamsTimeGTC
		var requestParamsOrderType RequestParamsOrderType
		switch s.Type {
		case order.Market:
			timeInForce = ""
			requestParamsOrderType = BybitRequestParamsOrderMarket
		case order.Limit:
			requestParamsOrderType = BybitRequestParamsOrderLimit
		default:
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, errUnsupportedOrderType
		}

		var orderRequest = PlaceOrderRequest{
			Symbol:      s.Pair.String(),
			Side:        sideType,
			Price:       s.Price,
			Quantity:    s.Amount,
			TradeType:   requestParamsOrderType,
			TimeInForce: timeInForce,
			OrderLinkID: s.ClientOrderID,
		}
		response, err := by.CreatePostOrder(ctx, &orderRequest)
		if err != nil {
			return submitOrderResponse, err
		}

		if response.OrderID > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response.OrderID, 10)
		}
		if response.ExecutedQty == response.Quantity {
			submitOrderResponse.FullyMatched = true
		}
		submitOrderResponse.IsOrderPlaced = true
	case asset.CoinMarginedFutures:
		var sideType string
		switch s.Side {
		case order.Buy:
			sideType = sideBuy
		case order.Sell:
			sideType = sideSell
		default:
			return submitOrderResponse, errInvalidSide
		}

		timeInForce := "GTC"
		var oType string
		switch s.Type {
		case order.Market:
			timeInForce = ""
			oType = "MARKET"
		case order.Limit:
			oType = "LIMIT"
		default:
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, errUnsupportedOrderType
		}

		o, err := by.CreateCoinFuturesOrder(ctx, s.Pair, sideType, oType, timeInForce,
			s.ClientOrderID, "", "",
			s.Amount, s.Price, 0, 0, false, s.ReduceOnly)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = o.OrderID
		submitOrderResponse.IsOrderPlaced = true
	case asset.USDTMarginedFutures:
		var sideType string
		switch s.Side {
		case order.Buy:
			sideType = sideBuy
		case order.Sell:
			sideType = sideSell
		default:
			return submitOrderResponse, errInvalidSide
		}

		timeInForce := "GTC"
		var oType string
		switch s.Type {
		case order.Market:
			timeInForce = ""
			oType = "MARKET"
		case order.Limit:
			oType = "LIMIT"
		default:
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, errUnsupportedOrderType
		}

		o, err := by.CreateUSDTFuturesOrder(ctx, s.Pair, sideType, oType, timeInForce,
			s.ClientOrderID, "", "",
			s.Amount, s.Price, 0, 0, false, s.ReduceOnly)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = o.OrderID
		submitOrderResponse.IsOrderPlaced = true
	case asset.Futures:
		var sideType string
		switch s.Side {
		case order.Buy:
			sideType = sideBuy
		case order.Sell:
			sideType = sideSell
		default:
			return submitOrderResponse, errInvalidSide
		}

		timeInForce := "GTC"
		var oType string
		switch s.Type {
		case order.Market:
			timeInForce = ""
			oType = "MARKET"
		case order.Limit:
			oType = "LIMIT"
		default:
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, errUnsupportedOrderType
		}

		// TODO: check position mode
		o, err := by.CreateFuturesOrder(ctx, 0, s.Pair, sideType, oType, timeInForce,
			s.ClientOrderID, "", "",
			s.Amount, s.Price, 0, 0, false, s.ReduceOnly)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = o.OrderID
		submitOrderResponse.IsOrderPlaced = true
	default:
		return submitOrderResponse, fmt.Errorf("assetType not supported: %v", s.AssetType)
	}

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (by *Bybit) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	if err := action.Validate(); err != nil {
		return order.Modify{}, err
	}

	var orderID string
	var err error
	switch action.AssetType {
	case asset.CoinMarginedFutures:
		orderID, err = by.ReplaceActiveCoinFuturesOrders(ctx, action.Pair, action.ID, action.ClientOrderID, "", "", int64(action.Amount), action.Price, 0, 0)
	case asset.USDTMarginedFutures:
		orderID, err = by.ReplaceActiveUSDTFuturesOrders(ctx, action.Pair, action.ID, action.ClientOrderID, "", "", int64(action.Amount), action.Price, 0, 0)

	case asset.Futures:
		orderID, err = by.ReplaceActiveFuturesOrders(ctx, action.Pair, action.ID, action.ClientOrderID, "", "", action.Amount, action.Price, 0, 0)
	default:
		err = fmt.Errorf("assetType not supported: %v", action.AssetType)
	}

	if err != nil {
		return order.Modify{}, err
	}
	return order.Modify{
		Exchange:  action.Exchange,
		AssetType: action.AssetType,
		Pair:      action.Pair,
		ID:        orderID,
		Price:     action.Price,
		Amount:    action.Amount,
	}, err
}

// CancelOrder cancels an order by its corresponding ID number
func (by *Bybit) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}

	var err error
	switch ord.AssetType {
	case asset.Spot:
		_, err = by.CancelExistingOrder(ctx, ord.ID, ord.ClientOrderID)
	case asset.CoinMarginedFutures:
		_, err = by.CancelActiveCoinFuturesOrders(ctx, ord.Pair, ord.ID, ord.ClientOrderID)
	case asset.USDTMarginedFutures:
		_, err = by.CancelActiveUSDTFuturesOrders(ctx, ord.Pair, ord.ID, ord.ClientOrderID)
	case asset.Futures:
		_, err = by.CancelActiveFuturesOrders(ctx, ord.Pair, ord.ID, ord.ClientOrderID)
	default:
		return fmt.Errorf("assetType not supported: %v", ord.AssetType)
	}
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (by *Bybit) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (by *Bybit) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch orderCancellation.AssetType {
	case asset.Spot:
		activeOrder, err := by.ListOpenOrders(ctx, orderCancellation.Symbol, "", 0)
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		successful, err := by.BatchCancelOrder(ctx, orderCancellation.Symbol, string(orderCancellation.Side), string(orderCancellation.Type))

		if !successful {
			return cancelAllOrdersResponse, fmt.Errorf("failed to cancelAllOrder")
		}
		for i := range activeOrder {
			cancelAllOrdersResponse.Status[strconv.FormatInt(activeOrder[i].OrderID, 10)] = err.Error()
		}

	case asset.CoinMarginedFutures:
		resp, err := by.CancelAllActiveCoinFuturesOrders(ctx, orderCancellation.Pair)

		for i := range resp {
			cancelAllOrdersResponse.Status[resp[i].OrderID] = err.Error()
		}
	case asset.USDTMarginedFutures:
		resp, err := by.CancelAllActiveUSDTFuturesOrders(ctx, orderCancellation.Pair)

		for i := range resp {
			cancelAllOrdersResponse.Status[resp[i]] = err.Error()
		}
	case asset.Futures:
		resp, err := by.CancelAllActiveFuturesOrders(ctx, orderCancellation.Pair)

		for i := range resp {
			cancelAllOrdersResponse.Status[resp[i].CancelOrderID] = err.Error()
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("assetType not supported: %v", orderCancellation.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (by *Bybit) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	switch assetType {
	case asset.Spot:
		resp, err := by.QueryOrder(ctx, orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		return order.Detail{
			Amount:         resp.Quantity,
			Exchange:       by.Name,
			ID:             strconv.FormatInt(resp.OrderID, 10),
			ClientOrderID:  resp.OrderLinkID,
			Side:           order.Side(resp.Side),
			Type:           order.Type(resp.TradeType),
			Pair:           pair,
			Cost:           resp.CummulativeQuoteQty,
			AssetType:      assetType,
			Status:         order.Status(resp.Status),
			Price:          resp.Price,
			ExecutedAmount: resp.ExecutedQty,
			Date:           resp.Time.Time(),
			LastUpdated:    resp.UpdateTime.Time(),
		}, nil
	case asset.CoinMarginedFutures:
		resp, err := by.GetActiveRealtimeCoinOrders(ctx, pair, orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		if len(resp) != 1 {
			return order.Detail{}, fmt.Errorf("invalid order's count found")
		}

		return order.Detail{
			Amount:         resp[0].Qty,
			Exchange:       by.Name,
			ID:             resp[0].OrderID,
			ClientOrderID:  resp[0].OrderLinkID,
			Side:           order.Side(resp[0].Side),
			Type:           order.Type(resp[0].OrderType),
			Pair:           pair,
			Cost:           resp[0].CumulativeQty,
			AssetType:      assetType,
			Status:         order.Status(resp[0].OrderStatus),
			Price:          resp[0].Price,
			ExecutedAmount: resp[0].Qty - resp[0].LeavesQty,
			Date:           resp[0].CreatedAt,
			LastUpdated:    resp[0].UpdatedAt,
		}, nil

	case asset.USDTMarginedFutures:
		resp, err := by.GetActiveUSDTRealtimeOrders(ctx, pair, orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		if len(resp) != 1 {
			return order.Detail{}, fmt.Errorf("invalid order's count found")
		}

		return order.Detail{
			Amount:         resp[0].Qty,
			Exchange:       by.Name,
			ID:             resp[0].OrderID,
			ClientOrderID:  resp[0].OrderLinkID,
			Side:           order.Side(resp[0].Side),
			Type:           order.Type(resp[0].OrderType),
			Pair:           pair,
			Cost:           resp[0].CumulativeQty,
			AssetType:      assetType,
			Status:         order.Status(resp[0].OrderStatus),
			Price:          resp[0].Price,
			ExecutedAmount: resp[0].Qty - resp[0].LeavesQty,
			Date:           resp[0].CreatedAt,
			LastUpdated:    resp[0].UpdatedAt,
		}, nil

	case asset.Futures:
		resp, err := by.GetActiveRealtimeOrders(ctx, pair, orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		if len(resp) != 1 {
			return order.Detail{}, fmt.Errorf("invalid order's count found")
		}

		return order.Detail{
			Amount:         resp[0].Qty,
			Exchange:       by.Name,
			ID:             resp[0].OrderID,
			ClientOrderID:  resp[0].OrderLinkID,
			Side:           order.Side(resp[0].Side),
			Type:           order.Type(resp[0].OrderType),
			Pair:           pair,
			Cost:           resp[0].CumulativeQty,
			AssetType:      assetType,
			Status:         order.Status(resp[0].OrderStatus),
			Price:          resp[0].Price,
			ExecutedAmount: resp[0].Qty - resp[0].LeavesQty,
			Date:           resp[0].CreatedAt,
			LastUpdated:    resp[0].UpdatedAt,
		}, nil

	default:
		return order.Detail{}, fmt.Errorf("assetType not supported: %v", assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (by *Bybit) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (by *Bybit) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (by *Bybit) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (by *Bybit) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (by *Bybit) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		// sending an empty currency pair retrieves data for all currencies
		req.Pairs = append(req.Pairs, currency.Pair{})
	}

	var orders []order.Detail
	for i := range req.Pairs {
		switch req.AssetType {
		case asset.Spot:
			openOrders, err := by.ListOpenOrders(ctx, req.Pairs[i].String(), "", 0)
			if err != nil {
				return nil, err
			}
			for x := range openOrders {
				orders = append(orders, order.Detail{
					Amount:        openOrders[x].Quantity,
					Date:          openOrders[x].Time.Time(),
					Exchange:      by.Name,
					ID:            strconv.FormatInt(openOrders[x].OrderID, 10),
					ClientOrderID: openOrders[x].OrderLinkID,
					Side:          order.Side(openOrders[x].Side),
					Type:          order.Type(openOrders[x].TradeType),
					Price:         openOrders[x].Price,
					Status:        order.Status(openOrders[x].Status),
					Pair:          req.Pairs[i],
					AssetType:     req.AssetType,
					LastUpdated:   openOrders[x].UpdateTime.Time(),
				})
			}
		case asset.CoinMarginedFutures:
			openOrders, err := by.GetActiveCoinFuturesOrders(ctx, req.Pairs[i], "", "", "", 0)
			if err != nil {
				return nil, err
			}

			for x := range openOrders {
				orders = append(orders, order.Detail{
					Price:           openOrders[x].Price,
					Amount:          openOrders[x].Qty,
					ExecutedAmount:  openOrders[x].Qty - openOrders[x].LeavesQty,
					RemainingAmount: openOrders[x].LeavesQty,
					Fee:             openOrders[x].CumulativeFee,
					Exchange:        by.Name,
					ID:              openOrders[x].OrderID,
					ClientOrderID:   openOrders[x].OrderLinkID,
					Type:            order.Type(openOrders[x].OrderType),
					Side:            order.Side(openOrders[x].Side),
					Status:          order.Status(openOrders[x].OrderStatus),
					Pair:            req.Pairs[i],
					AssetType:       req.AssetType,
					Date:            openOrders[x].CreatedAt,
				})
			}

		case asset.USDTMarginedFutures:
			openOrders, err := by.GetActiveUSDTFuturesOrders(ctx, req.Pairs[i], "", "", "", "", 0, 0)
			if err != nil {
				return nil, err
			}

			for x := range openOrders {
				orders = append(orders, order.Detail{
					Price:           openOrders[x].Price,
					Amount:          openOrders[x].Qty,
					ExecutedAmount:  openOrders[x].Qty - openOrders[x].LeavesQty,
					RemainingAmount: openOrders[x].LeaveValue,
					Fee:             openOrders[x].CumulativeFee,
					Exchange:        by.Name,
					ID:              openOrders[x].OrderID,
					ClientOrderID:   openOrders[x].OrderLinkID,
					Type:            order.Type(openOrders[x].OrderType),
					Side:            order.Side(openOrders[x].Side),
					Status:          order.Status(openOrders[x].OrderStatus),
					Pair:            req.Pairs[i],
					AssetType:       asset.USDTMarginedFutures,
					Date:            openOrders[x].CreatedAt,
				})
			}
		case asset.Futures:
			openOrders, err := by.GetActiveFuturesOrders(ctx, req.Pairs[i], "", "", "", 0)
			if err != nil {
				return nil, err
			}

			for x := range openOrders {
				orders = append(orders, order.Detail{
					Price:           openOrders[x].Price,
					Amount:          openOrders[x].Qty,
					ExecutedAmount:  openOrders[x].Qty - openOrders[x].LeavesQty,
					RemainingAmount: openOrders[x].LeavesQty,
					Fee:             openOrders[x].CumulativeFee,
					Exchange:        by.Name,
					ID:              openOrders[x].OrderID,
					ClientOrderID:   openOrders[x].OrderLinkID,
					Type:            order.Type(openOrders[x].OrderType),
					Side:            order.Side(openOrders[x].Side),
					Status:          order.Status(openOrders[x].OrderStatus),
					Pair:            req.Pairs[i],
					AssetType:       req.AssetType,
					Date:            openOrders[x].CreatedAt,
				})
			}
		default:
			return orders, fmt.Errorf("assetType not supported: %v", req.AssetType)
		}
	}
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (by *Bybit) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (by *Bybit) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (by *Bybit) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := by.UpdateAccountInfo(ctx, assetType)
	return by.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (by *Bybit) FormatExchangeKlineInterval(ctx context.Context, interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1h"
	case kline.TwoHour:
		return "2h"
	case kline.FourHour:
		return "4h"
	case kline.SixHour:
		return "4h"
	case kline.TwelveHour:
		return "12h"
	case kline.OneDay:
		return "1d"
	case kline.OneWeek:
		return "1w"
	case kline.OneMonth:
		return "1M"
	default:
		return interval.Short()
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (by *Bybit) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := by.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	klineItem := kline.Item{
		Exchange: by.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	switch a {
	case asset.Spot:
		candles, err := by.GetKlines(ctx, pair.String(), by.FormatExchangeKlineInterval(ctx, interval), int64(by.Features.Enabled.Kline.ResultLimit), start, end)
		if err != nil {
			return klineItem, err
		}

		for x := range candles {
			klineItem.Candles = append(klineItem.Candles, kline.Candle{
				Time:   candles[x].StartTime,
				Open:   candles[x].Open,
				High:   candles[x].High,
				Low:    candles[x].Low,
				Close:  candles[x].Close,
				Volume: candles[x].Volume,
			})
		}
	case asset.CoinMarginedFutures, asset.Futures:
		candles, err := by.GetFuturesKlineData(ctx, pair, by.FormatExchangeKlineInterval(ctx, interval), int64(by.Features.Enabled.Kline.ResultLimit), start)
		if err != nil {
			return klineItem, err
		}

		for x := range candles {
			klineItem.Candles = append(klineItem.Candles, kline.Candle{
				Time:   time.Unix(candles[x].OpenTime, 0),
				Open:   candles[x].Open,
				High:   candles[x].High,
				Low:    candles[x].Low,
				Close:  candles[x].Close,
				Volume: candles[x].Volume,
			})
		}
	case asset.USDTMarginedFutures:
		candles, err := by.GetUSDTFuturesKlineData(ctx, pair, by.FormatExchangeKlineInterval(ctx, interval), int64(by.Features.Enabled.Kline.ResultLimit), start)
		if err != nil {
			return klineItem, err
		}

		for x := range candles {
			klineItem.Candles = append(klineItem.Candles, kline.Candle{
				Time:   time.Unix(candles[x].OpenTime, 0),
				Open:   candles[x].Open,
				High:   candles[x].High,
				Low:    candles[x].Low,
				Close:  candles[x].Close,
				Volume: candles[x].Volume,
			})
		}

	default:
		return klineItem, fmt.Errorf("assetType not supported: %v", a)
	}

	klineItem.RemoveOutsideRange(start, end)
	klineItem.SortCandlesByTimestamp(false)
	return klineItem, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (by *Bybit) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := by.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: by.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, by.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}

	switch a {
	case asset.Spot:
		candles, err := by.GetKlines(ctx, pair.String(), by.FormatExchangeKlineInterval(ctx, interval), int64(by.Features.Enabled.Kline.ResultLimit), start, end)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range candles {
			for j := range ret.Candles {
				if ret.Candles[j].Time.Equal(time.Unix(int64(candles[i].Open), 0)) {
					continue
				}
			}
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   time.Unix(int64(candles[i].Open), 0),
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	case asset.CoinMarginedFutures, asset.Futures:
		candles, err := by.GetFuturesKlineData(ctx, pair, by.FormatExchangeKlineInterval(ctx, interval), int64(by.Features.Enabled.Kline.ResultLimit), start)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range candles {
			for j := range ret.Candles {
				if ret.Candles[j].Time.Equal(time.Unix(candles[i].OpenTime, 0)) {
					continue
				}
			}
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   time.Unix(candles[i].OpenTime, 0),
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	case asset.USDTMarginedFutures:
		candles, err := by.GetUSDTFuturesKlineData(ctx, pair, by.FormatExchangeKlineInterval(ctx, interval), int64(by.Features.Enabled.Kline.ResultLimit), start)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range candles {
			for j := range ret.Candles {
				if ret.Candles[j].Time.Equal(time.Unix(candles[i].OpenTime, 0)) {
					continue
				}
			}
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   time.Unix(candles[i].OpenTime, 0),
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}

	default:
		return kline.Item{}, fmt.Errorf("assetType not supported: %v", a)
	}

	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", by.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

func (b *Bybit) extractCurrencyPair(symbol string, item asset.Item) (currency.Pair, error) {
	if len(symbol) == 6 {
		return currency.NewPairFromString(symbol)
	}
	pairs, err := b.CurrencyPairs.GetPairs(item, true)
	if err != nil {
		return currency.Pair{}, err
	}
	pFmt, err := b.GetPairFormat(item, false)
	if err != nil {
		return currency.Pair{}, err
	}
	return currency.MatchPairsWithNoDelimiter(symbol, pairs, pFmt)
}
