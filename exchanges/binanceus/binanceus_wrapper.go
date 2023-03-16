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

	err := bi.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
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
	bi.Verbose = true
	bi.API.CredentialsValidator.RequiresKey = true
	bi.API.CredentialsValidator.RequiresSecret = true
	bi.SetValues()

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}
	err := bi.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	bi.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				SubmitOrders:          true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				TradeFetching:         true,
				UserTradeHistory:      true,
				TradeFee:              true,
				CryptoDepositFee:      true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrders:              true,
				TradeFetching:          true,
				KlineFetching:          true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.OneMin,
					kline.ThreeMin,
					kline.FiveMin,
					kline.FifteenMin,
					kline.ThirtyMin,
					kline.OneHour,
					kline.TwoHour,
					kline.FourHour,
					kline.SixHour,
					kline.EightHour,
					kline.TwelveHour,
					kline.OneDay,
					kline.ThreeDay,
					kline.OneWeek,
					kline.OneMonth,
				),
				ResultLimit: 1000,
			},
		},
	}
	bi.Requester, err = request.New(bi.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	bi.API.Endpoints = bi.NewEndpoints()
	if err := bi.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   binanceusAPIURL,
		exchange.RestSpotSupplementary:      binanceusAPIURL,
		exchange.WebsocketSpot:              binanceusDefaultWebsocketURL,
		exchange.WebsocketSpotSupplementary: binanceusDefaultWebsocketURL,
	}); err != nil {
		log.Errorf(log.ExchangeSys,
			"%s setting default endpoints error %v",
			bi.Name, err)
	}
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
		ExchangeConfig:         exch,
		DefaultURL:             binanceusDefaultWebsocketURL,
		RunningURL:             ePoint,
		Connector:              bi.WsConnect,
		Subscriber:             bi.Subscribe,
		Unsubscriber:           bi.Unsubscribe,
		GenerateSubscriptions:  bi.GenerateSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &bi.Features.Supports.WebsocketCapabilities,
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
func (bi *Binanceus) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !bi.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, bi.Name)
	}
	info, err := bi.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(info.Symbols))
	for x := range info.Symbols {
		if info.Symbols[x].Status != "TRADING" ||
			!info.Symbols[x].IsSpotTradingAllowed {
			continue
		}
		var pair currency.Pair
		pair, err = currency.NewPairFromStrings(info.Symbols[x].BaseAsset,
			info.Symbols[x].QuoteAsset)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
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
	return bi.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (bi *Binanceus) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w '%v'", asset.ErrNotSupported, a)
	}
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
	return ticker.GetTicker(bi.Name, p, a)
}

// UpdateTickers updates all currency pairs of a given asset type
func (bi *Binanceus) UpdateTickers(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("assetType not supported: %v", a)
	}
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
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (bi *Binanceus) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPairs, er := bi.FormatExchangeCurrency(p, assetType)
	if er != nil {
		return nil, er
	}

	tickerNew, er := ticker.GetTicker(bi.Name, fPairs, assetType)
	if er != nil {
		return bi.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (bi *Binanceus) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := bi.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	ob, err := orderbook.Get(bi.Name, fPair, assetType)
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
	if assetType != asset.Spot {
		return info, fmt.Errorf("%v  assetType is not supported", assetType)
	}
	theAccount, err := bi.GetAccount(ctx)
	if err != nil {
		return info, err
	}
	currencyBalance := make([]account.Balance, len(theAccount.Balances))
	for i := range theAccount.Balances {
		freeBalance := theAccount.Balances[i].Free.InexactFloat64()
		locked := theAccount.Balances[i].Locked.InexactFloat64()

		currencyBalance[i] = account.Balance{
			Currency: currency.NewCode(theAccount.Balances[i].Asset),
			Total:    freeBalance + locked,
			Hold:     locked,
			Free:     freeBalance,
		}
	}
	acc.Currencies = currencyBalance
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return info, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (bi *Binanceus) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(bi.Name, creds, assetType)
	if err != nil {
		return bi.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and withdrawals
func (bi *Binanceus) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (bi *Binanceus) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) (resp []exchange.WithdrawalHistory, err error) {
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
			AssetType:    assetType,
			CurrencyPair: p,
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
		StartTime: timestampStart.UnixMilli(),
		EndTime:   timestampEnd.UnixMilli(),
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
func (bi *Binanceus) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	var timeInForce RequestParamsTimeForceType
	var sideType string
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	if s.AssetType != asset.Spot {
		return nil, fmt.Errorf("%s %w", s.AssetType, asset.ErrNotSupported)
	}
	if s.Side == order.Buy {
		sideType = order.Buy.String()
	} else {
		sideType = order.Sell.String()
	}
	var requestParamOrderType RequestParamsOrderType
	switch s.Type {
	case order.Market:
		requestParamOrderType = BinanceRequestParamsOrderMarket
	case order.Limit:
		timeInForce = BinanceRequestParamsTimeGTC
		requestParamOrderType = BinanceRequestParamsOrderLimit
	default:
		return nil, errors.New(bi.Name + " unsupported order type")
	}
	var response NewOrderResponse
	response, err = bi.NewOrder(ctx, &NewOrderRequest{
		Symbol:           s.Pair,
		Side:             sideType,
		Price:            s.Price,
		Quantity:         s.Amount,
		TradeType:        requestParamOrderType,
		TimeInForce:      timeInForce,
		NewClientOrderID: s.ClientOrderID,
	})
	if err != nil {
		return nil, err
	}
	if response.OrderID > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.OrderID, 10)
	}
	if response.ExecutedQty == response.OrigQty {
		submitOrderResponse.Status = order.Filled
	}
	for i := range response.Fills {
		submitOrderResponse.Trades = append(submitOrderResponse.Trades, order.TradeHistory{
			Price:    response.Fills[i].Price,
			Amount:   response.Fills[i].Qty,
			Fee:      response.Fills[i].Commission,
			FeeAsset: response.Fills[i].CommissionAsset,
			Exchange: bi.Name,
		})
	}

	return &submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (bi *Binanceus) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (bi *Binanceus) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	if o.AssetType != asset.Spot {
		return fmt.Errorf("%w '%v'", asset.ErrNotSupported, o.AssetType)
	}
	_, err := bi.CancelExistingOrder(ctx,
		&CancelOrderRequestParams{
			Symbol:                o.Pair,
			OrderID:               o.OrderID,
			ClientSuppliedOrderID: o.ClientOrderID,
		})
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (bi *Binanceus) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (bi *Binanceus) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	if orderCancellation.AssetType == asset.Spot {
		symbolValue, err := bi.FormatSymbol(orderCancellation.Pair, asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		openOrders, err := bi.GetAllOpenOrders(ctx, symbolValue)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for ind := range openOrders {
			pair, err := currency.NewPairFromString(openOrders[ind].Symbol)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			_, err = bi.CancelExistingOrder(ctx, &CancelOrderRequestParams{
				Symbol:                pair,
				OrderID:               strconv.FormatUint(openOrders[ind].OrderID, 10),
				ClientSuppliedOrderID: openOrders[ind].ClientOrderID,
			})
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	} else {
		return cancelAllOrdersResponse, fmt.Errorf("%w '%v'", asset.ErrNotSupported, orderCancellation.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (bi *Binanceus) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var respData order.Detail
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return respData, fmt.Errorf("invalid orderID %w", err)
	}
	symbolValue, err := bi.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return respData, err
	}
	if assetType != asset.Spot {
		return respData, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	var orderType order.Type
	resp, err := bi.GetOrder(ctx, &OrderRequestParams{
		Symbol:  symbolValue,
		OrderID: uint64(orderIDInt),
	})
	if err != nil {
		return respData, err
	}
	orderSide, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
	}
	status, err := order.StringToOrderStatus(resp.Status)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
	}
	orderType, err = order.StringToOrderType(resp.Type)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
	}

	return order.Detail{
		Amount:         resp.OrigQty,
		Exchange:       bi.Name,
		OrderID:        strconv.FormatInt(int64(resp.OrderID), 10),
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
}

// GetDepositAddress returns a deposit address for a specified currency
func (bi *Binanceus) GetDepositAddress(ctx context.Context, c currency.Code, _ /*accountID*/, chain string) (*deposit.Address, error) {
	address, err := bi.GetDepositAddressForCurrency(ctx, c.String(), chain)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: address.Address,
		Tag:     address.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (bi *Binanceus) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawID, err := bi.WithdrawCrypto(ctx, withdrawRequest)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: withdrawID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted. But, GCT has no concept of withdrawal via SEN
// the fiat withdrawal end point of Binance.US is built to submit a USD withdraw request via Silvergate Exchange Network (SEN).
// So, this method is not implemented.
func (bi *Binanceus) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
// But, GCT has no concept of withdrawal via SEN the fiat withdrawal end point of Binance.US is built to submit a USD withdraw request via Silvergate Exchange Network (SEN).
func (bi *Binanceus) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (bi *Binanceus) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	var symbol string
	var pair currency.Pair
	var selectedOrders []Order
	if getOrdersRequest.AssetType != asset.Spot {
		return nil, fmt.Errorf("%s %w", getOrdersRequest.AssetType, asset.ErrNotSupported)
	}
	if len(getOrdersRequest.Pairs) != 1 {
		symbol = ""
	} else {
		symbol, err = bi.FormatSymbol(getOrdersRequest.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
	}
	resp, err := bi.GetAllOpenOrders(ctx, symbol)
	if err != nil {
		return nil, err
	}
	for s := range resp {
		ord := resp[s]
		pair, err = currency.NewPairFromString(ord.Symbol)
		if err != nil {
			continue
		}
		for p := range getOrdersRequest.Pairs {
			if getOrdersRequest.Pairs[p].Equal(pair) {
				selectedOrders = append(selectedOrders, ord)
			}
		}
	}
	orders := make([]order.Detail, len(selectedOrders))
	for x := range selectedOrders {
		var orderSide order.Side
		var orderType order.Type
		var orderStatus order.Status
		orderSide, err = order.StringToOrderSide(strings.ToUpper(resp[x].Side))
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
		}
		orderType, err = order.StringToOrderType(strings.ToUpper(resp[x].Type))
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
		}
		orderStatus, err = order.StringToOrderStatus(resp[x].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", bi.Name, err)
		}
		orders[x] = order.Detail{
			Amount:        resp[x].OrigQty,
			Date:          resp[x].Time,
			Exchange:      bi.Name,
			OrderID:       strconv.FormatInt(int64(resp[x].OrderID), 10),
			ClientOrderID: resp[x].ClientOrderID,
			Side:          orderSide,
			Type:          orderType,
			Price:         resp[x].Price,
			Status:        orderStatus,
			Pair:          getOrdersRequest.Pairs[0],
			AssetType:     getOrdersRequest.AssetType,
			LastUpdated:   resp[x].UpdateTime,
		}
	}
	return getOrdersRequest.Filter(bi.Name, orders), nil
}

// GetOrderHistory retrieves account order information Can Limit response to specific order status
func (bi *Binanceus) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	// An endpoint like /api/v3/allOrders does not exist in the binance us
	// so This end point is left unimplemented
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (bi *Binanceus) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!bi.AreCredentialsValid(ctx) || bi.SkipAuthCheck) &&
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
func (bi *Binanceus) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := bi.GetKlineRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	candles, err := bi.GetSpotKline(ctx, &KlinesRequestParams{
		Interval:  bi.GetIntervalEnum(req.ExchangeInterval),
		Symbol:    req.Pair,
		StartTime: req.Start,
		EndTime:   req.End,
		Limit:     int64(bi.Features.Enabled.Kline.ResultLimit),
	})
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (bi *Binanceus) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := bi.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles []CandleStick
		candles, err = bi.GetSpotKline(ctx, &KlinesRequestParams{
			Interval:  bi.GetIntervalEnum(req.ExchangeInterval),
			Symbol:    req.Pair,
			StartTime: req.RangeHolder.Ranges[x].Start.Time,
			EndTime:   req.RangeHolder.Ranges[x].End.Time,
			Limit:     int64(bi.Features.Enabled.Kline.ResultLimit),
		})
		if err != nil {
			return nil, err
		}

		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].OpenTime,
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (bi *Binanceus) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	coinInfo, err := bi.GetAssetFeesAndWalletStatus(ctx)
	if err != nil {
		return nil, err
	}

	var availableChains []string
	for x := range coinInfo {
		if strings.EqualFold(coinInfo[x].Coin, cryptocurrency.String()) {
			for y := range coinInfo[x].NetworkList {
				if coinInfo[x].NetworkList[y].DepositEnable {
					availableChains = append(availableChains, coinInfo[x].NetworkList[y].Network)
				}
			}
		}
	}
	return availableChains, nil
}
