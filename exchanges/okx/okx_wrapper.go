package okx

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (ok *Okx) GetDefaultConfig() (*config.Exchange, error) {
	ok.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = ok.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = ok.BaseCurrencies

	err := ok.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if ok.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := ok.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Okx
func (ok *Okx) SetDefaults() {
	ok.Name = "Okx"
	ok.Enabled = true
	ok.Verbose = true

	ok.API.CredentialsValidator.RequiresKey = true
	ok.API.CredentialsValidator.RequiresSecret = true
	ok.API.CredentialsValidator.RequiresClientID = true

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}

	err := ok.SetGlobalPairsManager(fmt1.RequestFormat, fmt1.ConfigFormat, asset.Spot, asset.Futures, asset.PerpetualSwap)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	ok.Features = exchange.Features{
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
	ok.Requester, err = request.New(ok.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// NOTE: SET THE URLs HERE
	ok.API.Endpoints = ok.NewEndpoints()
	ok.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      okxAPIURL,
		exchange.WebsocketSpot: okxWebsocketURL,
	})
	ok.Websocket = stream.New()
	ok.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	ok.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	ok.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (ok *Okx) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		ok.SetEnabled(false)
		return nil
	}
	err = ok.SetupDefaults(exch)
	if err != nil {
		return err
	}

	/*
		wsRunningEndpoint, err := ok.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			return err
		}

		// If websocket is supported, please fill out the following

		err = ok.Websocket.Setup(
			&stream.WebsocketSetup{
				ExchangeConfig:  exch,
				DefaultURL:      okxWSAPIURL,
				RunningURL:      wsRunningEndpoint,
				Connector:       ok.WsConnect,
				Subscriber:      ok.Subscribe,
				UnSubscriber:    ok.Unsubscribe,
				Features:        &ok.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		ok.WebsocketConn = &stream.WebsocketConnection{
			ExchangeName:         ok.Name,
			URL:                  ok.Websocket.GetWebsocketURL(),
			ProxyURL:             ok.Websocket.GetProxyAddress(),
			Verbose:              ok.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
	*/
	return nil
}

// Start starts the Okx go routine
func (ok *Okx) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		ok.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Okx wrapper
func (ok *Okx) Run() {
	if ok.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			ok.Name,
			common.IsEnabled(ok.Websocket.IsEnabled()))
		ok.PrintEnabledPairs()
	}
	if !ok.GetEnabledFeatures().AutoPairUpdates {
		return
	}
	err := ok.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			ok.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (ok *Okx) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, ok.Name)
	}
	format, err := ok.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	pairs := []string{}
	insts := []*Instrument{}
	var er error
	switch a {
	case asset.Spot:
		insts, er = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: "SPOT",
		})
	case asset.Futures:
		insts, er = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: "FUTURES",
		})
	case asset.PerpetualSwap:
		insts, er = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: "SWAP",
		})
	case asset.Option:
		insts, er = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: "OPTION",
		})
	case asset.Margin:
		insts, er = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: "MARGIN",
		})
	}
	if er != nil || len(insts) == 0 {
		return pairs, er
	}
	for x := range insts {
		var pair string
		println(insts[x].InstrumentID)
		switch insts[x].InstrumentType {
		case asset.Spot:
			pair = insts[x].BaseCurrency + format.Delimiter + insts[x].QuoteCurrency
		case asset.Futures, asset.PerpetualSwap, asset.Option:
			currency, err := currency.NewPairFromString(insts[x].Underlying)
			if err != nil {
				continue
			}
			pair = currency.Base.String() + format.Delimiter + currency.Quote.String()
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (ok *Okx) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := ok.GetAssetTypes(false)
	for i := range assetTypes {
		p, err := ok.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return err
		}
		pairs, err := currency.NewPairsFromStrings(p)
		if err != nil {
			return err
		}
		err = ok.UpdatePairs(pairs, assetTypes[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (ok *Okx) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	var mdata *MarketDataResponse
	var er error
	var instrumentID string
	instrumentID, er = ok.GetInstrumentIDFromPair(p, a)
	if er != nil {
		return nil, er
	}
	switch a {
	case asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Futures, asset.Option:
		mdata, er = ok.GetTicker(context.Background(), instrumentID)
		if er != nil {
			return nil, er
		}
		er = ticker.ProcessTicker(&ticker.Price{
			Last:         mdata.LastTradePrice,
			High:         mdata.High24H,
			Low:          mdata.Low24H,
			Bid:          mdata.BidPrice,
			Ask:          mdata.BestAskPrice,
			Volume:       mdata.Vol24H,
			QuoteVolume:  mdata.VolCcy24H,
			Open:         mdata.Open24H,
			Pair:         p,
			ExchangeName: ok.Name,
			AssetType:    a,
		})
		if er != nil {
			return nil, er
		}
	default:
		return nil, fmt.Errorf("assetType not supported: %v", a)
	}
	return ticker.GetTicker(ok.Name, p, a)
}

// UpdateTickers updates all currency pairs of a given asset type
func (ok *Okx) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot, asset.Margin, asset.Futures, asset.PerpetualSwap, asset.Option:
		instrumentType := ""
		switch assetType {
		case asset.PerpetualSwap:
			instrumentType = "SWAP"
		default:
			instrumentType = assetType.String()
		}
		ticks, er := ok.GetTickers(ctx, strings.ToUpper(instrumentType), "", "")
		if er != nil {
			return er
		}
		pairs, er := ok.GetEnabledPairs(assetType)
		if er != nil {
			return er
		}
		for i := range pairs {
			for y := range ticks {
				pairFmt, err := ok.FormatExchangeCurrency(pairs[i], assetType)
				if err != nil {
					return err
				}
				pair, er := ok.GetPairFromInstrumentID(ticks[y].InstrumentID, assetType)
				if er != nil {
					return er
				}
				if pair.String() != pairFmt.String() {
					continue
				}
				er = ticker.ProcessTicker(&ticker.Price{
					Last:         ticks[y].LastTradePrice,
					High:         ticks[y].High24H,
					Low:          ticks[y].Low24H,
					Bid:          ticks[y].BidPrice,
					Ask:          ticks[y].BestAskPrice,
					Volume:       ticks[y].Vol24H,
					QuoteVolume:  ticks[y].VolCcy24H,
					Open:         ticks[y].Open24H,
					Pair:         pair,
					ExchangeName: ok.Name,
					AssetType:    assetType,
				})
				if er != nil {
					return er
				}
			}
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (ok *Okx) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	formatedPair, er := ok.FormatExchangeCurrency(p, assetType)
	if er != nil {
		return nil, er
	}
	tickerNew, err := ticker.GetTicker(ok.Name, formatedPair, assetType)
	if err != nil {
		return ok.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (ok *Okx) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(ok.Name, pair, assetType)
	if err != nil {
		return ok.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (ok *Okx) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        ok.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: ok.CanVerifyOrderbook,
	}

	var orderbookNew *OrderBookResponse
	var er error
	switch assetType {
	case asset.Spot, asset.Margin, asset.PerpetualSwap, asset.Option, asset.Futures:
		instrumentID, er := ok.GetInstrumentIDFromPair(pair, assetType)
		if er != nil {
			return book, er
		}
		orderbookNew, er = ok.GetOrderBookDepth(ctx, instrumentID, 0)
		if er != nil {
			return book, er
		}
	default:
		return nil, errInvalidInstrumentType
	}
	orderBookD := orderbookNew.GetOrderBookResponseDetail()
	book.Bids = make(orderbook.Items, len(orderBookD.Bids))
	for x := range orderBookD.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderBookD.Bids[x].BaseCurrencies,
			Price:  orderBookD.Bids[x].DepthPrice,
		}
	}
	book.Asks = make(orderbook.Items, len(orderBookD.Asks))
	for x := range orderBookD.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: orderBookD.Asks[x].NumberOfContracts,
			Price:  orderBookD.Asks[x].DepthPrice,
		}
	}
	er = book.Process()
	if er != nil {
		return book, er
	}
	return orderbook.Get(ok.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies.
func (ok *Okx) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = ok.Name
	switch assetType {
	case asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Option:
		balances, er := ok.GetBalance(context.Background(), "")
		if er != nil {
			return info, er
		}
		var currencyBalance []account.Balance
		for i := range balances {
			free := balances[i].AvailBal
			locked := balances[i].FrozenBalance
			currencyBalance = append(currencyBalance, account.Balance{
				CurrencyName: currency.NewCode(balances[i].Currency),
				Total:        balances[i].Balance,
				Hold:         locked,
				Free:         free,
			})
		}
		acc.Currencies = currencyBalance
	default:
		return info, errInvalidInstrumentType
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	if er := account.Process(&info); er != nil {
		return account.Holdings{}, er
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (ok *Okx) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, er := account.GetHoldings(ok.Name, assetType)
	if er != nil {
		return ok.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and withdrawals
func (ok *Okx) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	depositHistories, er := ok.GetCurrencyDepositHistory(ctx, "", "", "", -1, time.Time{}, time.Time{}, 0)
	if er != nil {
		return nil, er
	}
	withdrawalHistories, er := ok.GetWithdrawalHistory(ctx, "", "", "", "", -5, time.Time{}, time.Time{}, 0)
	if er != nil {
		return nil, er
	}
	resp := make([]exchange.FundHistory, 0, len(depositHistories)+len(withdrawalHistories))
	for x := range depositHistories {
		resp = append(resp, exchange.FundHistory{
			ExchangeName:    ok.Name,
			Status:          strconv.Itoa(depositHistories[x].State),
			Timestamp:       depositHistories[x].Timestamp,
			Currency:        depositHistories[x].Currency,
			Amount:          depositHistories[x].Amount,
			TransferType:    "deposit",
			CryptoToAddress: depositHistories[x].ToDepositAddress,
			CryptoTxID:      depositHistories[x].TransactionID,
		})
	}
	for x := range withdrawalHistories {
		resp = append(resp, exchange.FundHistory{
			ExchangeName:    ok.Name,
			Status:          withdrawalHistories[x].StateOfWithdrawal,
			Timestamp:       withdrawalHistories[x].Timestamp,
			Currency:        withdrawalHistories[x].Currency,
			Amount:          withdrawalHistories[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalHistories[x].ToReceivingAddress,
			CryptoTxID:      withdrawalHistories[x].TransactionID,
			TransferID:      withdrawalHistories[x].WithdrawalID,
			Fee:             withdrawalHistories[x].WithdrawalFee,
			CryptoChain:     withdrawalHistories[x].ChainName,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (ok *Okx) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	withdrawals, er := ok.GetWithdrawalHistory(ctx, c.String(), "", "", "", -5, time.Time{}, time.Time{}, 0)
	if er != nil {
		return nil, er
	}
	resp = make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for x := range withdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          withdrawals[x].StateOfWithdrawal,
			Timestamp:       withdrawals[x].Timestamp,
			Currency:        withdrawals[x].Currency,
			Amount:          withdrawals[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[x].ToReceivingAddress,
			CryptoTxID:      withdrawals[x].TransactionID,
			CryptoChain:     withdrawals[x].ChainName,
			TransferID:      withdrawals[x].WithdrawalID,
			Fee:             withdrawals[x].WithdrawalFee,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (ok *Okx) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	const limit = 1000
	instrumentID, er := ok.GetInstrumentIDFromPair(p, assetType)
	if er != nil {
		return nil, er
	}
	tradeData, er := ok.GetTrades(ctx, instrumentID, limit)
	if er != nil {
		return nil, er
	}
	resp := make([]trade.Data, len(tradeData))
	for x := range tradeData {
		resp[x] = trade.Data{
			TID:          strconv.FormatInt(int64(tradeData[x].TradeID), 10),
			Exchange:     ok.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData[x].Price,
			Amount:       tradeData[x].Quantity,
			Timestamp:    tradeData[x].Timestamp,
		}
	}
	if ok.IsSaveTradeDataEnabled() {
		er := trade.AddTradesToBuffer(ok.Name, resp...)
		if er != nil {
			return nil, er
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (ok *Okx) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	const limit = 1000
	instrumentID, er := ok.GetInstrumentIDFromPair(p, assetType)
	if er != nil {
		return nil, er
	}
	tradeData, er := ok.GetTradesHistory(ctx, instrumentID, "", "", limit)
	if er != nil {
		return nil, er
	}
	resp := make([]trade.Data, len(tradeData))
	for x := range tradeData {
		resp[x] = trade.Data{
			TID:          strconv.FormatInt(int64(tradeData[x].TradeID), 10),
			Exchange:     ok.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData[x].Price,
			Amount:       tradeData[x].Quantity,
			Timestamp:    tradeData[x].Timestamp,
		}
	}
	if ok.IsSaveTradeDataEnabled() {
		er := trade.AddTradesToBuffer(ok.Name, resp...)
		if er != nil {
			return nil, er
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (ok *Okx) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	var orderType string
	switch s.Type {
	case order.Market:
		orderType = "market"
	case order.Limit:
		orderType = "limit"
	case order.FillOrKill:
		orderType = "fok"
	case order.PostOnly:
		orderType = "post_only"
	case order.IOS:
		orderType = "ioc"
	default:
		if !(s.AssetType == asset.PerpetualSwap || s.AssetType == asset.Futures) {
			return submitOrderResponse, errInvalidOrderType
		}
		orderType = ""
	}
	instrumentID, er := ok.GetInstrumentIDFromPair(s.Pair, s.AssetType)
	if er != nil {
		return submitOrderResponse, er
	}
	tradeMode := "cash"
	var sideType string
	if s.Side == order.Buy {
		sideType = order.Buy.String()
	} else {
		sideType = order.Sell.String()
	}
	sideType = strings.ToLower(sideType)
	var orderRequest = PlaceOrderRequestParam{
		InstrumentID:          instrumentID,
		TradeMode:             tradeMode,
		Side:                  sideType,
		OrderType:             orderType,
		QuantityToBuyOrSell:   s.Amount,
		ClientSupplierOrderID: s.ClientOrderID,
	}
	switch orderType {
	case "limit", "post_only", "fok", "ioc":
		orderRequest.OrderPrice = s.Price
	}
	var placeOrderResponse *PlaceOrderResponse
	switch s.AssetType {
	case asset.Spot, asset.Option:
		placeOrderResponse, er = ok.PlaceOrder(ctx, orderRequest)
	case asset.PerpetualSwap, asset.Futures:
		if orderType == "" {
			orderType = "optimal_limit_ioc" // only applicable for Futures and Perpetual Swap Types.
		}
		orderRequest.PositionSide = "long"
		placeOrderResponse, er = ok.PlaceOrder(ctx, orderRequest)
	default:
		return submitOrderResponse, errInvalidInstrumentType
	}
	if er != nil {
		return submitOrderResponse, er
	}
	if placeOrderResponse.OrderID != "0" && placeOrderResponse.OrderID != "" {
		submitOrderResponse.OrderID = placeOrderResponse.OrderID
	}
	submitOrderResponse.IsOrderPlaced = true
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (ok *Okx) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	if err := action.Validate(); err != nil {
		return order.Modify{}, err
	}
	var modify order.Modify
	var amendRequest AmendOrderRequestParams
	var er error
	if math.Mod(action.Amount, 1) != 0 {
		return modify, errors.New("Okx contract amount can not be decimal")
	}
	instrumentID, er := ok.GetInstrumentIDFromPair(action.Pair, action.AssetType)
	if er != nil {
		return modify, er
	}
	amendRequest.InstrumentID = instrumentID
	amendRequest.NewQuantity = action.Amount
	amendRequest.OrderID = action.ID
	amendRequest.ClientSuppliedOrderID = action.ClientOrderID
	response, er := ok.AmendOrder(ctx, &amendRequest)
	if er != nil {
		return modify, er
	}
	return order.Modify{
		Exchange:  action.Exchange,
		AssetType: action.AssetType,
		Pair:      action.Pair,
		ID:        response.OrderID,
		Price:     action.Price,
		Amount:    amendRequest.NewQuantity,
	}, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (ok *Okx) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Option:
		instrumentID, er := ok.GetInstrumentIDFromPair(ord.Pair, ord.AssetType)
		if er != nil {
			return er
		}
		req := CancelOrderRequestParam{
			InstrumentID:          instrumentID,
			OrderID:               ord.ID,
			ClientSupplierOrderID: ord.ClientOrderID,
		}
		_, er = ok.CancelSingleOrder(ctx, req)
		return er
	default:
		return errInvalidInstrumentType
	}
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (ok *Okx) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	var cancelBatchResponse order.CancelBatchResponse
	cancelOrderParams := []CancelOrderRequestParam{}
	var er error
	for x := range orders {
		ord := orders[x]
		if err := ord.Validate(ord.StandardCancel()); err != nil {
			return cancelBatchResponse, err
		}
		switch ord.AssetType {
		case asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Option:
			instrumentID, er := ok.GetInstrumentIDFromPair(ord.Pair, ord.AssetType)
			if er != nil {
				return cancelBatchResponse, er
			}
			req := CancelOrderRequestParam{
				InstrumentID:          instrumentID,
				OrderID:               ord.ID,
				ClientSupplierOrderID: ord.ClientOrderID,
			}
			cancelOrderParams = append(cancelOrderParams, req)
		default:
			return cancelBatchResponse, errInvalidInstrumentType
		}
	}
	_, er = ok.CancelMultipleOrders(ctx, cancelOrderParams)
	return cancelBatchResponse, er
}

// CancelAllOrders cancels all orders associated with a currency pair
func (ok *Okx) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (ok *Okx) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var respData order.Detail
	instrumentID, er := ok.GetInstrumentIDFromPair(pair, assetType)
	if er != nil {
		return respData, er
	}
	switch assetType {
	case asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Option:
		orderDetail, er := ok.GetOrderDetail(context.Background(), &OrderDetailRequestParam{
			InstrumentID: instrumentID,
			OrderID:      orderID,
		})
		if er != nil {
			return respData, er
		}
		status, err := order.StringToOrderStatus(orderDetail.State)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
		}
		var orderType order.Type
		switch orderDetail.OrderType {
		case "market":
			orderType = order.Market
		case "limit":
			orderType = order.Limit
		case "post_only":
			orderType = order.PostOnly
		case "fok":
			orderType = order.FillOrKill
		case "ioc":
			orderType = order.IOS
		case "optimal_limit_ioc":
			orderType = order.UnknownType
		}
		orderSide := order.Side(orderDetail.Side)
		return order.Detail{
			Amount:         orderDetail.Size,
			Exchange:       ok.Name,
			ID:             orderDetail.OrderID,
			ClientOrderID:  orderDetail.ClientSupplierOrderID,
			Side:           orderSide,
			Type:           orderType,
			Pair:           pair,
			Cost:           orderDetail.Price,
			AssetType:      assetType,
			Status:         status,
			Price:          orderDetail.Price,
			ExecutedAmount: orderDetail.RebateAmount,
			Date:           orderDetail.CreationTime,
			LastUpdated:    orderDetail.UpdateTime,
		}, er
	default:
		return respData, errInvalidInstrumentType
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (ok *Okx) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (ok *Okx) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (ok *Okx) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (ok *Okx) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (ok *Okx) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (ok *Okx) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (ok *Okx) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (ok *Okx) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := ok.UpdateAccountInfo(ctx, assetType)
	return ok.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
