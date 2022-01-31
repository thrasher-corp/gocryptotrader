package bybit

import (
	"errors"
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
	var resp []trade.Data

	switch assetType {
	case asset.Spot:
		tradeData, err := by.GetTrades("", 0)
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
		tradeData, err := by.GetPublicTrades(currency.Pair{}, 0, 0)
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
		tradeData, err := by.GetUSDTPublicTrades(currency.Pair{}, 0)
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
		return nil, fmt.Errorf("%v assetType not supported", assetType)
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
func (by *Bybit) GetHistoricTrades(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (by *Bybit) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
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
			return submitOrderResponse, fmt.Errorf("invalid side")
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
			return submitOrderResponse, errors.New("unsupported order type")
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
		response, err := by.CreatePostOrder(&orderRequest)
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
			return submitOrderResponse, fmt.Errorf("invalid side")
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
			return submitOrderResponse, errors.New("unsupported order type")
		}

		o, err := by.CreateCoinFuturesOrder(s.Pair, sideType, oType, timeInForce,
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
			return submitOrderResponse, fmt.Errorf("invalid side")
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
			return submitOrderResponse, errors.New("unsupported order type")
		}

		o, err := by.CreateUSDTFuturesOrder(s.Pair, sideType, oType, timeInForce,
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
			return submitOrderResponse, fmt.Errorf("invalid side")
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
			return submitOrderResponse, errors.New("unsupported order type")
		}

		// TODO: check position mode
		o, err := by.CreateFuturesOrder(0, s.Pair, sideType, oType, timeInForce,
			s.ClientOrderID, "", "",
			s.Amount, s.Price, 0, 0, false, s.ReduceOnly)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = o.OrderID
		submitOrderResponse.IsOrderPlaced = true
	default:
		return submitOrderResponse, fmt.Errorf("assetType not supported")
	}

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (by *Bybit) ModifyOrder(action *order.Modify) (string, error) {
	if err := action.Validate(); err != nil {
		return "", err
	}

	var order string
	var err error
	switch action.AssetType {
	case asset.CoinMarginedFutures:
		order, err = by.ReplaceActiveCoinFuturesOrders(action.Pair, action.ID, action.ClientOrderID, "", "", int64(action.Amount), action.Price, 0, 0)
	case asset.USDTMarginedFutures:
		order, err = by.ReplaceActiveUSDTFuturesOrders(action.Pair, action.ID, action.ClientOrderID, "", "", int64(action.Amount), action.Price, 0, 0)

	case asset.Futures:
		order, err = by.ReplaceActiveFuturesOrders(action.Pair, action.ID, action.ClientOrderID, "", "", action.Amount, action.Price, 0, 0)
	default:
		return "", fmt.Errorf("assetType not supported")
	}

	if err != nil {
		return "", err
	}
	return order, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (by *Bybit) CancelOrder(ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}

	var err error
	switch ord.AssetType {
	case asset.Spot:
		_, err = by.CancelExistingOrder(ord.ID, ord.ClientOrderID)
	case asset.CoinMarginedFutures:
		_, err = by.CancelActiveCoinFuturesOrders(ord.Pair, ord.ID, ord.ClientOrderID)
	case asset.USDTMarginedFutures:
		_, err = by.CancelActiveUSDTFuturesOrders(ord.Pair, ord.ID, ord.ClientOrderID)
	case asset.Futures:
		_, err = by.CancelActiveFuturesOrders(ord.Pair, ord.ID, ord.ClientOrderID)
	default:
		return fmt.Errorf("assetType not supported")
	}
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (by *Bybit) CancelBatchOrders(orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (by *Bybit) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch orderCancellation.AssetType {
	case asset.Spot:
		activeOrder, err := by.ListOpenOrders(orderCancellation.Symbol, "", 0)

		successful, err := by.BatchCancelOrder(orderCancellation.Symbol, string(orderCancellation.Side), string(orderCancellation.Type))

		if successful {
			for i := range activeOrder {
				cancelAllOrdersResponse.Status[strconv.FormatInt(activeOrder[i].OrderID, 10)] = err.Error()
			}
		} else {
			return cancelAllOrdersResponse, fmt.Errorf("failed to cancelAllOrder")
		}
	case asset.CoinMarginedFutures:
		resp, err := by.CancelAllActiveCoinFuturesOrders(orderCancellation.Pair)

		for i := range resp {
			cancelAllOrdersResponse.Status[resp[i].OrderID] = err.Error()
		}
	case asset.USDTMarginedFutures:
		resp, err := by.CancelAllActiveUSDTFuturesOrders(orderCancellation.Pair)

		for i := range resp {
			cancelAllOrdersResponse.Status[resp[i]] = err.Error()
		}
	case asset.Futures:
		resp, err := by.CancelAllActiveFuturesOrders(orderCancellation.Pair)

		for i := range resp {
			cancelAllOrdersResponse.Status[resp[i].CancelOrderID] = err.Error()
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("assetType not supported")
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (by *Bybit) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	switch assetType {
	case asset.Spot:
		resp, err := by.QueryOrder(orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		cummulativeQuoteQty, err := strconv.ParseFloat(resp.CummulativeQuoteQty, 64)
		if err != nil {
			return order.Detail{}, err
		}

		executedQuoteQty, err := strconv.ParseFloat(resp.ExecutedQty, 64)
		if err != nil {
			return order.Detail{}, err
		}

		// TODO: check if auto data type conversion can cause any issue
		return order.Detail{
			Amount:         resp.Quantity,
			Exchange:       by.Name,
			ID:             strconv.FormatInt(resp.OrderID, 10),
			ClientOrderID:  resp.OrderLinkID,
			Side:           order.Side(resp.Side),
			Type:           order.Type(resp.TradeType),
			Pair:           pair,
			Cost:           cummulativeQuoteQty,
			AssetType:      assetType,
			Status:         order.Status(resp.Status),
			Price:          resp.Price,
			ExecutedAmount: executedQuoteQty,
			Date:           time.Unix(resp.Time, 0),
			LastUpdated:    time.Unix(resp.UpdateTime, 0),
		}, nil
	case asset.CoinMarginedFutures:
		resp, err := by.GetActiveRealtimeCoinOrders(pair, orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		if len(resp) != 1 {
			fmt.Errorf("invalid order's count found")
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
		resp, err := by.GetActiveUSDTRealtimeOrders(pair, orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		if len(resp) != 1 {
			fmt.Errorf("invalid order's count found")
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
		resp, err := by.GetActiveRealtimeOrders(pair, orderID, "")
		if err != nil {
			return order.Detail{}, err
		}

		if len(resp) != 1 {
			fmt.Errorf("invalid order's count found")
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
		return order.Detail{}, fmt.Errorf("assetType not supported")
	}
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
func (by *Bybit) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
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
			openOrders, err := by.ListOpenOrders(req.Pairs[i].String(), "", 0)
			if err != nil {
				return nil, err
			}
			for x := range openOrders {
				orders = append(orders, order.Detail{
					Amount:        openOrders[x].Quantity,
					Date:          time.Unix(openOrders[x].Time, 0),
					Exchange:      by.Name,
					ID:            strconv.FormatInt(openOrders[x].OrderID, 10),
					ClientOrderID: openOrders[x].OrderLinkID,
					Side:          order.Side(openOrders[x].Side),
					Type:          order.Type(openOrders[x].TradeType),
					Price:         openOrders[x].Price,
					Status:        order.Status(openOrders[x].Status),
					Pair:          req.Pairs[i],
					AssetType:     req.AssetType,
					LastUpdated:   time.Unix(openOrders[x].UpdateTime, 0),
				})
			}
		case asset.CoinMarginedFutures:
			openOrders, err := by.GetActiveCoinFuturesOrders(req.Pairs[i], "", "", "", 0)
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
			openOrders, err := by.GetActiveUSDTFuturesOrders(req.Pairs[i], "", "", "", "", 0, 0)
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
			openOrders, err := by.GetActiveFuturesOrders(req.Pairs[i], "", "", "", 0)
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
			return orders, fmt.Errorf("assetType not supported")
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
