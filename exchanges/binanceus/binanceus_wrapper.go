package binanceus

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
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
func (bi *Binanceus) GetDefaultConfig() (*config.Exchange, error) {
	bi.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = bi.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = bi.BaseCurrencies

	er := bi.SetupDefaults(exchCfg)
	if er != nil {
		return nil, er
	}

	if bi.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := bi.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Binanceus
func (bi *Binanceus) SetDefaults() {
	bi.Name = "Binanceus"
	bi.Enabled = true
	bi.Verbose = false
	bi.API.CredentialsValidator.RequiresKey = true
	bi.API.CredentialsValidator.RequiresSecret = true
	bi.SetValues()

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
	}
	err := bi.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	// Fill out the capabilities/features that the exchange supports
	bi.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:    true,
				TickerFetching:    true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				SubmitOrders:      true,
				DepositHistory:    true,
				WithdrawalHistory: true,
				TradeFetching:     true,
				UserTradeHistory:  true,
				TradeFee:          true,
				// FiatDepositFee:      true,
				// FiatWithdrawalFee:   true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
				Subscribe:         true,
				Unsubscribe:       true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.ThreeMin.Word():   true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.SixHour.Word():    true,
					kline.EightHour.Word():  true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.ThreeDay.Word():   true,
					kline.OneWeek.Word():    true,
					kline.OneMonth.Word():   true,
				},
				ResultLimit: 1000,
			},
		},
	}
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	bi.Requester, err = request.New(bi.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// NOTE: SET THE URLs HERE
	bi.API.Endpoints = bi.NewEndpoints()
	bi.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   binanceusAPIURL,
		exchange.RestSpotSupplementary:      binanceusAPIURL,
		exchange.WebsocketSpot:              binanceusDefaultWebsocketURL,
		exchange.WebsocketSpotSupplementary: binanceusDefaultWebsocketURL,
	})
	bi.Websocket = stream.New()
	bi.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	bi.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	bi.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (bi *Binanceus) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		bi.SetEnabled(false)
		return nil
	}
	err = bi.SetupDefaults(exch)
	if err != nil {
		return err
	}

	ePoint, err := bi.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = bi.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            binanceusDefaultWebsocketURL,
		RunningURL:            ePoint,
		Connector:             bi.WsConnect,
		Subscriber:            bi.Subscribe,
		Unsubscriber:          bi.Unsubscribe,
		GenerateSubscriptions: bi.GenerateSubscriptions,
		Features:              &bi.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
		TradeFeed: bi.Features.Enabled.TradeFeed,
	})
	if err != nil {
		return err
	}

	return bi.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            wsRateLimitMilliseconds,
	})
}

// Start starts the Binanceus go routine
func (bi *Binanceus) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		bi.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Binanceus wrapper
func (bi *Binanceus) Run() {
	if bi.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			bi.Name,
			common.IsEnabled(bi.Websocket.IsEnabled()))
		bi.PrintEnabledPairs()
	}

	if !bi.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := bi.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			bi.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (bi *Binanceus) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	if !bi.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, bi.Name)
	}
	format, err := bi.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	tradingStatus := "TRADING"
	var pairs []string

	switch a {
	case asset.Spot:
		var info ExchangeInfo
		info, err = bi.GetExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		for x := range info.Symbols {
			if info.Symbols[x].Status != tradingStatus {
				continue
			}
			pair := info.Symbols[x].BaseAsset +
				format.Delimiter +
				info.Symbols[x].QuoteAsset
			if a == asset.Spot && info.Symbols[x].IsSpotTradingAllowed {
				pairs = append(pairs, pair)
			}
			if a == asset.Margin && info.Symbols[x].IsMarginTradingAllowed {
				pairs = append(pairs, pair)
			}
		}
		var cInfo ExchangeInfo
		cInfo, err = bi.GetExchangeInfo(ctx)
		if err != nil {
			return pairs, err
		}
		for z := range cInfo.Symbols {
			if cInfo.Symbols[z].Status != tradingStatus {
				continue
			}
			var curr currency.Pair
			curr, err = currency.NewPairFromString(cInfo.Symbols[z].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, format.Format(curr))
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (bi *Binanceus) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := bi.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return bi.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (bi *Binanceus) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	switch a {
	case asset.Spot:
		tick, err := bi.GetPriceChangeStats(ctx, p)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick.LastPrice,
			High:         tick.HighPrice,
			Low:          tick.LowPrice,
			Bid:          tick.BidPrice,
			Ask:          tick.AskPrice,
			Volume:       tick.Volume,
			QuoteVolume:  tick.QuoteVolume,
			Open:         tick.OpenPrice,
			Close:        tick.PrevClosePrice,
			Pair:         p,
			ExchangeName: bi.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("assetType not supported: %v", a)
	}
	return ticker.GetTicker(bi.Name, p, a)
}

// UpdateTickers updates all currency pairs of a given asset type
func (bi *Binanceus) UpdateTickers(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot:
		tick, err := bi.GetTickers(ctx)
		if err != nil {
			return err
		}

		pairs, err := bi.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		for i := range pairs {
			for y := range tick {
				pairFmt, err := bi.FormatExchangeCurrency(pairs[i], a)
				if err != nil {
					return err
				}
				if tick[y].Symbol != pairFmt.String() {
					continue
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
					Pair:         pairFmt,
					ExchangeName: bi.Name,
					AssetType:    a,
				})
				if err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("assetType not supported: %v", a)
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (bi *Binanceus) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {

	fpairs, er := bi.FormatExchangeCurrency(p, assetType)
	if er != nil {
		return nil, er
	}
	bi.appendOptionalDelimiter(&fpairs)

	tickerNew, err := ticker.GetTicker(bi.Name, p, assetType)
	if err != nil {
		return bi.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// appendOptionalDelimiter ensures that a delimiter is
// present for long character currencies
func (b *Binanceus) appendOptionalDelimiter(p *currency.Pair) {
	if len(p.Quote.String()) > 3 ||
		len(p.Base.String()) > 3 {
		p.Delimiter = ":"
	}
}

// FetchOrderbook returns orderbook base on the currency pair
func (bi *Binanceus) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := bi.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	bi.appendOptionalDelimiter(&fPair)
	ob, err := orderbook.Get(bi.Name, pair, assetType)
	if err != nil {
		return bi.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (bi *Binanceus) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        bi.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: bi.CanVerifyOrderbook,
	}

	orderbookNew, err := bi.GetOrderBookDepth(ctx, &OrderBookDataRequestParams{
		Symbol: pair,
		Limit:  1000})
	if err != nil {
		return book, err
	}

	book.Bids = make([]orderbook.Item, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Quantity,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	book.Asks = make([]orderbook.Item, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: orderbookNew.Asks[x].Quantity,
			Price:  orderbookNew.Asks[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(bi.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (bi *Binanceus) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = bi.Name
	switch assetType {
	case asset.Spot:
		theaccount, err := bi.GetAccount(ctx)
		if err != nil {
			return info, err
		}
		var currencyBalance []account.Balance
		for i := range theaccount.Balances {
			freeBalance := theaccount.Balances[i].Free.InexactFloat64()
			locked := theaccount.Balances[i].Locked.InexactFloat64()

			currencyBalance = append(currencyBalance, account.Balance{
				CurrencyName: currency.NewCode(theaccount.Balances[i].Asset),
				Total:        freeBalance + locked,
				Hold:         locked, // This are the locked account balances quantity.
				Free:         freeBalance,
			})
		}
		acc.Currencies = currencyBalance
	default:
		return info, fmt.Errorf("%v  assetType is not supported", assetType)
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	if er := account.Process(&info); er != nil {
		return account.Holdings{}, er
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (bi *Binanceus) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, er := account.GetHoldings(bi.Name, assetType)
	if er != nil {
		return bi.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (bi *Binanceus) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	// Not Implemented in the Binanceus endpoint
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (bi *Binanceus) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	w, err := bi.WithdrawalHistory(ctx, c, "", time.Time{}, time.Time{}, 0, 10000)
	if err != nil {
		return nil, err
	}
	for i := range w {
		tm, err := time.Parse(binanceUSAPITimeLayout, w[i].ApplyTime)
		if err != nil {
			return nil, err
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          fmt.Sprint(w[i].Status),
			TransferID:      w[i].ID,
			Currency:        w[i].Coin,
			Amount:          w[i].Amount,
			Fee:             w[i].TransactionFee,
			CryptoToAddress: w[i].Address,
			CryptoTxID:      w[i].ID,
			CryptoChain:     w[i].Network,
			Timestamp:       tm,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (bi *Binanceus) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	const limit = 1000
	tradeData, err := bi.GetMostRecentTrades(ctx, RecentTradeRequestParams{p, limit})
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		resp[i] = trade.Data{
			TID:          fmt.Sprint(tradeData[i].ID),
			Exchange:     bi.Name,
			AssetType:    assetType, //  always the asset type is Spot,
			CurrencyPair: p,         // This is the currency pair input we used.
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Quantity,
			Timestamp:    tradeData[i].Time,
		}
	}

	if bi.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(bi.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (bi *Binanceus) GetHistoricTrades(ctx context.Context, p currency.Pair,
	assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	req := AggregatedTradeRequestParams{
		Symbol:    p,
		StartTime: uint64(timestampStart.UnixMilli()),
		EndTime:   uint64(timestampEnd.UnixMilli()),
	}
	trades, err := bi.GetAggregateTrades(ctx, &req)
	if err != nil {
		return nil, err
	}
	result := make([]trade.Data, len(trades))
	exName := bi.Name
	for i := range trades {
		t := trades[i].toTradeData(p, exName, assetType)
		result[i] = *t
	}
	return result, nil
}

// SubmitOrder submits a new order
func (bi *Binanceus) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	var timeInForce RequestParamsTimeForceType
	switch s.AssetType {
	case asset.Spot:
		var sideType string
		if s.Side == order.Buy {
			sideType = order.Buy.String()
		} else {
			sideType = order.Sell.String()
		}
		timeInForce = BinanceRequestParamsTimeGTC
		var requestParamOrderType RequestParamsOrderType
		switch s.Type {
		case order.Market:
			timeInForce = ""
			requestParamOrderType = BinanceRequestParamsOrderMarket
		default:
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, errors.New("unsupported order type")
		}

		var orderRequest = NewOrderRequest{
			Symbol:           s.Pair,
			Side:             sideType,
			Price:            s.Price,
			Quantity:         s.Amount,
			TradeType:        requestParamOrderType,
			TimeInForce:      timeInForce,
			NewClientOrderID: s.ClientOrderID,
		}
		response, er := bi.NewOrder(ctx, &orderRequest)
		if er != nil {
			return submitOrderResponse, er
		}
		if response.OrderID > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response.OrderID, 10)
		}
		if response.ExecutedQty == response.OrigQty {
			submitOrderResponse.FullyMatched = true
		}
		submitOrderResponse.IsOrderPlaced = true
		for i := range response.Fills {
			submitOrderResponse.Trades = append(submitOrderResponse.Trades, order.TradeHistory{
				Price:    response.Fills[i].Price,
				Amount:   response.Fills[i].Qty,
				Fee:      response.Fills[i].Commission,
				FeeAsset: response.Fills[i].CommissionAsset,
			})
		}
	default:
		return submitOrderResponse, fmt.Errorf("assetType not supported")
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (bi *Binanceus) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (bi *Binanceus) CancelOrder(ctx context.Context, order *order.Cancel) error {
	if err := order.Validate(order.StandardCancel()); err != nil {
		return err
	}
	switch order.AssetType {
	case asset.Spot:
		orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
		if err != nil {
			return err
		}
		_, err = bi.CancelExistingOrder(ctx,
			CancelOrderRequestParams{
				Symbol:            order.Pair,
				OrderID:           uint64(orderIDInt),
				OrigClientOrderID: order.AccountID,
			})
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("assetType not supported")
	}
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (bi *Binanceus) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (bi *Binanceus) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch orderCancellation.AssetType {
	case asset.Spot:
		symbolValue, err := bi.FormatSymbol(orderCancellation.Pair, asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		openOrders, er := bi.GetAllOpenOrders(ctx, symbolValue)
		if er != nil {
			return cancelAllOrdersResponse, er
		}
		for _, openO := range openOrders {
			pair, er := currency.NewPairFromString(openO.Symbol)
			if er != nil {
				return cancelAllOrdersResponse, er
			}
			_, err := bi.CancelExistingOrder(ctx, CancelOrderRequestParams{
				Symbol:            pair,
				OrderID:           openO.OrderID,
				OrigClientOrderID: openO.ClientOrderID,
			})
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("assetType not supported: %v", orderCancellation.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (bi *Binanceus) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var respData order.Detail
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return respData, err
	}
	symbolValue, err := bi.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return respData, err
	}
	switch assetType {
	case asset.Spot:
		resp, err := bi.GetOrder(ctx, OrderRequestParams{
			Symbol:            symbolValue,
			OrderID:           uint64(orderIDInt),
			OrigClientOrderId: "",
			// RecvWindow : 60000,
		})
		if err != nil {
			return respData, err
		}
		orderSide := order.Side(resp.Side)
		status, err := order.StringToOrderStatus(resp.Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
		}
		orderType := order.Limit
		if strings.ToUpper(resp.Type) == "MARKET" {
			orderType = order.Market
		}

		return order.Detail{
			Amount:         resp.OrigQty,
			Exchange:       bi.Name,
			ID:             strconv.FormatInt(int64(resp.OrderID), 10),
			ClientOrderID:  resp.ClientOrderID,
			Side:           orderSide,
			Type:           orderType,
			Pair:           pair,
			Cost:           resp.CummulativeQuoteQty,
			AssetType:      assetType,
			Status:         status,
			Price:          resp.Price,
			ExecutedAmount: resp.ExecutedQty,
			Date:           resp.Time,
			LastUpdated:    resp.UpdateTime,
		}, nil
	default:
		return respData, fmt.Errorf("assetType %s not supported", assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (bi *Binanceus) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	address, err := bi.GetDepositAddressForCurrency(ctx, c.String(), chain)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: address.Address,
		Tag:     address.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (bi *Binanceus) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, er := bi.WithdrawCrypto(ctx, WithdrawalRequestParam{
		Coin:            withdrawRequest.Currency.String(),
		WithdrawOrderId: "",
		Network:         withdrawRequest.Crypto.Chain,
		Address:         withdrawRequest.Crypto.Address,
		AddressTag:      withdrawRequest.Crypto.AddressTag,
		Amount:          withdrawRequest.Amount,
	})
	if er != nil {
		return nil, er
	}
	return &withdraw.ExchangeResponse{
		ID: resp,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
// But, GCT has no concept of withdrawal via SEN
// the fiat withdrawal end point of Binance.US is built to submit a USD withdraw request via Silvergate Exchange Network (SEN).
// So, this method is not implemented.
func (bi *Binanceus) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	// 	return nil, err
	// }

	// resp, err := bi.WithdrawFiat(ctx, WithdrawFiatRequestParams{
	// 	PaymentAccount: withdrawRequest.Fiat.IntermediaryBankAccountNumber,
	// 	FiatCurrency:   "",
	// 	Amount:         withdrawRequest.Amount,
	// })

	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
// But, GCT has no concept of withdrawal via SEN
// the fiat withdrawal end point of Binance.US is built to submit a USD withdraw request via Silvergate Exchange Network (SEN).
func (bi *Binanceus) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (bi *Binanceus) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	if len(getOrdersRequest.Pairs) == 0 || len(getOrdersRequest.Pairs) >= 40 {
		getOrdersRequest.Pairs = append(getOrdersRequest.Pairs, currency.EMPTYPAIR)
	}
	var orders []order.Detail
	for i := range getOrdersRequest.Pairs {
		switch getOrdersRequest.AssetType {
		case asset.Spot:
			symbol, err := bi.FormatSymbol(getOrdersRequest.Pairs[i], asset.Spot)
			if err != nil {
				return orders, err
			}
			resp, err := bi.GetAllOpenOrders(ctx, symbol)
			if err != nil {
				return nil, err
			}
			for x := range resp {
				orderSide := order.Side(strings.ToUpper(resp[x].Side))
				orderType := order.Type(strings.ToUpper(resp[x].Type))
				orderStatus, err := order.StringToOrderStatus(resp[i].Status)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
				}
				orders = append(orders, order.Detail{
					Amount:        resp[x].OrigQty,
					Date:          resp[x].Time,
					Exchange:      bi.Name,
					ID:            strconv.FormatInt(int64(resp[x].OrderID), 10),
					ClientOrderID: resp[x].ClientOrderID,
					Side:          orderSide,
					Type:          orderType,
					Price:         resp[x].Price,
					Status:        orderStatus,
					Pair:          getOrdersRequest.Pairs[i],
					AssetType:     getOrdersRequest.AssetType,
					LastUpdated:   resp[x].UpdateTime,
				})
			}
		default:
			return orders, fmt.Errorf("assetType not supported")
		}
	}
	order.FilterOrdersByCurrencies(&orders, getOrdersRequest.Pairs)
	order.FilterOrdersByType(&orders, getOrdersRequest.Type)
	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)
	order.FilterOrdersByTimeRange(&orders, getOrdersRequest.StartTime, getOrdersRequest.EndTime)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (bi *Binanceus) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// An endpoint like /api/v3/allOrders does not exist in the binance us
	// so This end point is left Un Implemented
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (bi *Binanceus) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!bi.AreCredentialsValid(ctx) || bi.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return bi.GetFee(ctx, feeBuilder)
}

// ValidateCredentials validates current credentials used for wrapper
func (bi *Binanceus) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := bi.UpdateAccountInfo(ctx, assetType)
	return bi.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (bi *Binanceus) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := bi.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	if kline.TotalCandlesPerInterval(start, end, interval) > float64(bi.Features.Enabled.Kline.ResultLimit) {
		return kline.Item{}, errors.New(kline.ErrRequestExceedsExchangeLimits)
	}
	req := KlinesRequestParams{
		Interval:  bi.GetIntervalEnum(interval),
		Symbol:    pair,
		StartTime: start,
		EndTime:   end,
		Limit:     int(bi.Features.Enabled.Kline.ResultLimit),
	}
	ret := kline.Item{
		Exchange: bi.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	candles, err := bi.GetSpotKline(ctx, &req)
	if err != nil {
		return kline.Item{}, err
	}
	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		})
	}
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (bi *Binanceus) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := bi.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: bi.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, bi.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	var candles []CandleStick
	for x := range dates.Ranges {
		req := KlinesRequestParams{
			Interval:  bi.FormatExchangeKlineInterval(interval),
			Symbol:    pair,
			StartTime: dates.Ranges[x].Start.Time,
			EndTime:   dates.Ranges[x].End.Time,
			Limit:     int(bi.Features.Enabled.Kline.ResultLimit),
		}

		candles, err = bi.GetSpotKline(ctx, &req)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range candles {
			for j := range ret.Candles {
				if ret.Candles[j].Time.Equal(candles[i].OpenTime) {
					continue
				}
			}
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   candles[i].OpenTime,
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	}
	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", bi.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
