package cryptodotcom

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for Cryptodotcom
func (e *Exchange) SetDefaults() {
	e.Name = "Cryptodotcom"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.PerpetualSwap} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		}
		if a == asset.PerpetualSwap {
			ps.RequestFormat.Delimiter = currency.DashDelimiter
			ps.ConfigFormat.Delimiter = currency.DashDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}
	// Fill out the capabilities/features that the exchange supports
	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				TradeFetching:       true,
				KlineFetching:       true,
				OrderbookFetching:   true,
				CryptoWithdrawal:    true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				CancelOrders:        true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				UserTradeHistory:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerBatching:         true,
				TickerFetching:         true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				CryptoWithdrawal:       true,
				TradeFetching:          true,
				AccountBalance:         true,
				SubmitOrder:            true,
				SubmitOrders:           true,
				CancelOrder:            true,
				CancelOrders:           true,
				GetOrder:               true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			FillsFeed:       true,
			TradeFeed:       true,
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.SevenDay},
					kline.IntervalCapacity{Interval: kline.TwoWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 300,
			},
		},
	}
	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   apiURLV1,
		exchange.RestSpotSupplementary:      apiURLV1Supplementary,
		exchange.RestFutures:                restURL,
		exchange.WebsocketSpot:              cryptodotcomWebsocketMarketAPI,
		exchange.WebsocketSpotSupplementary: cryptodotcomWebsocketUserAPI,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = time.Second * 15
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *Exchange) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	err = e.SetupDefaults(exch)
	if err != nil {
		return err
	}
	wsRunningEndpoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = e.Websocket.Setup(
		&websocket.ManagerSetup{
			ExchangeConfig:        exch,
			DefaultURL:            cryptodotcomWebsocketUserAPI,
			RunningURL:            wsRunningEndpoint,
			Connector:             e.WsConnect,
			Subscriber:            e.Subscribe,
			Unsubscriber:          e.Unsubscribe,
			GenerateSubscriptions: e.GenerateDefaultSubscriptions,
			Features:              &e.Features.Supports.WebsocketCapabilities,
			FillsFeed:             exch.Features.Enabled.FillsFeed,
			TradeFeed:             exch.Features.Enabled.TradeFeed,
		})
	if err != nil {
		return err
	}
	err = e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  cryptodotcomWebsocketMarketAPI,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  cryptodotcomWebsocketUserAPI,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, e.Name)
	}
	switch a {
	case asset.Spot, asset.Margin:
		instruments, err := e.GetInstruments(ctx)
		if err != nil {
			return nil, err
		}
		pairs := currency.Pairs{}
		for x := range instruments.Instruments {
			if instruments.Instruments[x].InstrumentType != "CCY_PAIR" {
				continue
			}
			pair, err := currency.NewPairFromString(instruments.Instruments[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
		return pairs, nil
	case asset.PerpetualSwap:
		instruments, err := e.GetInstruments(ctx)
		if err != nil {
			return nil, err
		}
		pairs := currency.Pairs{}
		for x := range instruments.Instruments {
			if instruments.Instruments[x].InstrumentType != "PERPETUAL_SWAP" {
				continue
			}
			pair, err := currency.NewPairFromString(instruments.Instruments[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w asset type: %s", asset.ErrNotSupported, a.String())
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assetTypes := e.GetAssetTypes(true)
	for _, assetType := range assetTypes {
		pairs, err := e.FetchTradablePairs(ctx, assetType)
		if err != nil {
			return err
		}
		if assetType == asset.OTC && !e.IsRESTAuthenticationSupported() {
			continue
		}
		return e.UpdatePairs(pairs, assetType, false)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tick, err := e.GetTickers(ctx, p.String())
	if err != nil {
		return nil, err
	}
	if len(tick.Data) != 1 {
		return nil, errInvalidResponseFromServer
	}
	tickerPrice := &ticker.Price{
		High:         tick.Data[0].HighestTradePrice.Float64(),
		Low:          tick.Data[0].LowestTradePrice.Float64(),
		Bid:          tick.Data[0].BestBidPrice.Float64(),
		Ask:          tick.Data[0].BestAskPrice.Float64(),
		Last:         tick.Data[0].LatestTradePrice.Float64(),
		Volume:       tick.Data[0].TradedVolume.Float64(),
		LastUpdated:  tick.Data[0].TradeTimestamp.Time(),
		AssetType:    assetType,
		ExchangeName: e.Name,
		Pair:         p,
	}
	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	if !e.SupportsAsset(assetType) {
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	tick, err := e.GetTickers(ctx, "")
	if err != nil {
		return err
	}
	for y := range tick.Data {
		cp, err := currency.NewPairFromString(tick.Data[y].InstrumentName)
		if err != nil {
			return err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick.Data[y].LatestTradePrice.Float64(),
			High:         tick.Data[y].HighestTradePrice.Float64(),
			Low:          tick.Data[y].LowestTradePrice.Float64(),
			Bid:          tick.Data[y].BestBidPrice.Float64(),
			Ask:          tick.Data[y].BestAskPrice.Float64(),
			Volume:       tick.Data[y].TradedVolume.Float64(),
			QuoteVolume:  tick.Data[y].TradedVolumeInUSD24H.Float64(),
			AssetType:    assetType,
			ExchangeName: e.Name,
			Pair:         cp,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (e *Exchange) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	tickerNew, err := ticker.GetTicker(e.Name, p, assetType)
	if err != nil {
		return e.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (e *Exchange) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	ob, err := orderbook.Get(e.Name, pair, assetType)
	if err != nil {
		return e.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	pair, err := e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	orderbookNew, err := e.GetOrderbook(ctx, pair.String(), 0)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	if len(orderbookNew.Data) == 0 {
		return nil, fmt.Errorf("%w, missing orderbook data", orderbook.ErrOrderbookInvalid)
	}
	book.Bids = make([]orderbook.Level, len(orderbookNew.Data[0].Bids))
	for x := range orderbookNew.Data[0].Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Data[0].Bids[x][1].Float64(),
			Price:  orderbookNew.Data[0].Bids[x][0].Float64(),
		}
	}
	book.Asks = make([]orderbook.Level, len(orderbookNew.Data[0].Asks))
	for x := range orderbookNew.Data[0].Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Data[0].Asks[x][1].Float64(),
			Price:  orderbookNew.Data[0].Asks[x][0].Float64(),
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (e *Exchange) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	if !e.SupportsAsset(assetType) {
		return info, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	var accs *Accounts
	var err error
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		accs, err = e.WsRetriveAccountSummary(currency.EMPTYCODE)
	} else {
		accs, err = e.GetAccountSummary(ctx, currency.EMPTYCODE)
	}
	if err != nil {
		return info, err
	}
	balances := make([]account.Balance, len(accs.Accounts))
	for i := range accs.Accounts {
		balances[i] = account.Balance{
			Currency: currency.NewCode(accs.Accounts[i].Currency),
			Total:    accs.Accounts[i].Balance,
			Hold:     accs.Accounts[i].Stake + accs.Accounts[i].Order,
			Free:     accs.Accounts[i].Available,
		}
	}
	acc := account.SubAccount{
		Currencies: balances,
		AssetType:  assetType,
	}
	info.Accounts = []account.SubAccount{acc}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return info, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}
	info.Exchange = e.Name
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (e *Exchange) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(e.Name, creds, assetType)
	if err != nil {
		return e.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	var err error
	var withdrawals *WithdrawalResponse
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		withdrawals, err = e.WsRetriveWithdrawalHistory()
	} else {
		withdrawals, err = e.GetWithdrawalHistory(ctx)
	}
	if err != nil {
		return nil, err
	}
	deposits, err := e.GetDepositHistory(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(withdrawals.WithdrawalList)+len(deposits.DepositList))
	for x := range withdrawals.WithdrawalList {
		resp = append(resp, exchange.FundingHistory{
			Status:          translateWithdrawalStatus(withdrawals.WithdrawalList[x].Status),
			Timestamp:       withdrawals.WithdrawalList[x].UpdateTime.Time(),
			Currency:        withdrawals.WithdrawalList[x].Currency,
			Amount:          withdrawals.WithdrawalList[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals.WithdrawalList[x].Address,
			TransferID:      withdrawals.WithdrawalList[x].TransactionID,
			Fee:             withdrawals.WithdrawalList[x].Fee,
		})
	}
	for x := range deposits.DepositList {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          translateDepositStatus(deposits.DepositList[x].Status),
			Timestamp:       deposits.DepositList[x].UpdateTime.Time(),
			Currency:        deposits.DepositList[x].Currency,
			Amount:          deposits.DepositList[x].Amount,
			TransferType:    "deposit",
			CryptoToAddress: deposits.DepositList[x].Address,
			CryptoTxID:      deposits.DepositList[x].ID,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.GetWithdrawalHistory(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals.WithdrawalList))
	for x := range withdrawals.WithdrawalList {
		resp[x] = exchange.WithdrawalHistory{
			Status:          translateWithdrawalStatus(withdrawals.WithdrawalList[x].Status),
			Timestamp:       withdrawals.WithdrawalList[x].UpdateTime.Time(),
			Currency:        withdrawals.WithdrawalList[x].Currency,
			Amount:          withdrawals.WithdrawalList[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals.WithdrawalList[x].Address,
			TransferID:      withdrawals.WithdrawalList[x].TransactionID,
			Fee:             withdrawals.WithdrawalList[x].Fee,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	trades, err := e.GetTrades(ctx, p.String(), 0, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades.Data))
	for x := range trades.Data {
		var side order.Side
		side, err = order.StringToOrderSide(trades.Data[x].Side)
		if err != nil {
			return nil, err
		}
		resp[x] = trade.Data{
			TID:          trades.Data[x].TradeID,
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        trades.Data[x].TradePrice.Float64(),
			Amount:       trades.Data[x].TradeQuantity.Float64(),
			Timestamp:    trades.Data[x].TradeTimestamp.Time(),
		}
	}
	if e.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w, asset type %v not supported", asset.ErrNotSupported, assetType)
	}
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", startTime, endTime, err)
	}
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	limit := 1000
	ts := startTime
	var resp []trade.Data
allTrades:
	for {
		var tradeData *TradesResponse
		tradeData, err = e.GetTrades(ctx, p.String(), 0, startTime, endTime)
		if err != nil {
			return nil, err
		}
		for i := range tradeData.Data {
			if tradeData.Data[i].TradeTimestamp.Time().Before(startTime) || tradeData.Data[i].TradeTimestamp.Time().After(endTime) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData.Data[i].Side)
			if err != nil {
				return nil, err
			}
			if tradeData.Data[i].TradePrice == 0 {
				continue
			}
			resp = append(resp, trade.Data{
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData.Data[i].TradePrice.Float64(),
				Amount:       tradeData.Data[i].TradeQuantity.Float64(),
				Timestamp:    tradeData.Data[i].TradeTimestamp.Time(),
				TID:          tradeData.Data[i].TradeID,
			})
			if i == len(tradeData.Data)-1 {
				if ts.Equal(tradeData.Data[i].TradeTimestamp.Time()) {
					// reached end of trades to crawl
					break allTrades
				}
				ts = tradeData.Data[i].TradeTimestamp.Time()
			}
		}
		if len(tradeData.Data) != limit {
			break allTrades
		}
	}
	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, startTime, endTime), nil
}

func timeInForceToString(tif order.TimeInForce) string {
	switch {
	case tif.Is(order.GoodTillCancel):
		return tifGTC
	case tif.Is(order.ImmediateOrCancel):
		return tifIOC
	case tif.Is(order.FillOrKill):
		return tifFOK
	}
	return ""
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if !e.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
	if s.Amount <= 0 {
		return nil, fmt.Errorf("%w, amount to buy or sell hast to be greater than zero ", order.ErrAmountIsInvalid)
	}
	format, err := e.GetPairFormat(s.AssetType, false)
	if err != nil {
		return nil, err
	}
	if !s.Pair.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var notional float64
	switch s.Type {
	case order.Market, order.Stop, order.TakeProfit:
		// For MARKET (BUY), STOP_LOSS (BUY), TAKE_PROFIT (BUY) orders only: Amount to spend
		notional = s.Amount
	}
	priceTypeString, err := priceTypeToString(s.TriggerPriceType)
	if err != nil {
		return nil, err
	}
	var ordersResp *CreateOrderResponse
	arg := &OrderParam{Symbol: format.Format(s.Pair), Side: s.Side, OrderType: s.Type, Price: s.Price, Quantity: s.Amount, ClientOrderID: s.ClientOrderID, Notional: notional, PostOnly: s.TimeInForce.Is(order.PostOnly), TriggerPrice: s.TriggerPrice, TriggerPriceType: priceTypeString, TimeInForce: timeInForceToString(s.TimeInForce)}
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		ordersResp, err = e.WsPlaceOrder(arg)
	} else {
		ordersResp, err = e.CreateOrder(ctx, arg)
	}
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(ordersResp.OrderID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *Exchange) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	err := ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	if !e.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	format, err := e.GetPairFormat(ord.AssetType, false)
	if err != nil {
		return err
	}
	if !ord.Pair.IsPopulated() {
		return currency.ErrCurrencyPairEmpty
	}
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		return e.WsCancelExistingOrder(format.Format(ord.Pair), ord.OrderID)
	}
	return e.CancelExistingOrder(ctx, format.Format(ord.Pair), ord.OrderID)
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	cancelBatchResponse := &order.CancelBatchResponse{
		Status: map[string]string{},
	}
	cancelOrderParams := []CancelOrderParam{}
	format, err := e.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	for x := range orders {
		cancelOrderParams = append(cancelOrderParams, CancelOrderParam{
			InstrumentName: format.Format(orders[x].Pair),
			OrderID:        orders[x].OrderID,
		})
	}
	var cancelResp *CancelOrdersResponse
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		cancelResp, err = e.WsCancelOrderList(cancelOrderParams)
	} else {
		cancelResp, err = e.CancelOrderList(ctx, cancelOrderParams)
	}
	if err != nil {
		return nil, err
	}
	for x := range cancelResp.ResultList {
		if cancelResp.ResultList[x].Code != 0 {
			cancelBatchResponse.Status[cancelOrderParams[cancelResp.ResultList[x].Index].InstrumentName] = ""
		} else {
			cancelBatchResponse.Status[cancelOrderParams[cancelResp.ResultList[x].Index].InstrumentName] = order.Cancelled.String()
		}
	}
	return cancelBatchResponse, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}
	err := orderCancellation.Validate()
	if err != nil {
		return cancelAllResponse, err
	}
	format, err := e.GetPairFormat(orderCancellation.AssetType, true)
	if err != nil {
		return cancelAllResponse, err
	}
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		return order.CancelAllResponse{}, e.WsCancelAllPersonalOrders(orderCancellation.Pair.Format(format).String(), OrderTypeToString(orderCancellation.Type))
	}
	return order.CancelAllResponse{}, e.CancelAllPersonalOrders(ctx, orderCancellation.Pair.Format(format).String(), OrderTypeToString(orderCancellation.Type))
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	if !pair.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	orderDetail, err := e.GetOrderDetail(ctx, orderID, "")
	if err != nil {
		return nil, err
	}
	status, err := order.StringToOrderStatus(orderDetail.Status)
	if err != nil {
		return nil, err
	}
	orderType, err := StringToOrderType(orderDetail.OrderType)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(orderDetail.Side)
	if err != nil {
		return nil, err
	}
	pair, err = e.FormatExchangeCurrency(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	return &order.Detail{
		ExecutedAmount: orderDetail.CumulativeQuantity.Float64() - orderDetail.Quantity.Float64(),
		Cost:           orderDetail.CumulativeValue.Float64(),
		Date:           orderDetail.CreateTime.Time(),
		LastUpdated:    orderDetail.UpdateTime.Time(),
		Amount:         orderDetail.Quantity.Float64(),
		ClientOrderID:  orderDetail.ClientOrderID,
		OrderID:        orderDetail.OrderID,
		Type:           orderType,
		Exchange:       e.Name,
		Side:           side,
		Pair:           pair,
		AssetType:      assetType,
		Status:         status,
		Price:          orderDetail.Price.Float64(),
		TimeInForce:    orderDetail.TimeInForce,
	}, err
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, accountID, chain string) (*deposit.Address, error) {
	dAddresses, err := e.GetPersonalDepositAddress(ctx, c)
	if err != nil {
		return nil, err
	}
	for x := range dAddresses.DepositAddressList {
		if dAddresses.DepositAddressList[x].Currency == c.String() &&
			(accountID == "" || accountID == dAddresses.DepositAddressList[x].ID) &&
			(chain == "" || chain == dAddresses.DepositAddressList[x].Network) {
			return &deposit.Address{
				Address: dAddresses.DepositAddressList[x].Address,
				Chain:   dAddresses.DepositAddressList[x].Network,
			}, nil
		}
	}
	return nil, fmt.Errorf("deposit address not found for currency: %s chain: %s", c, chain)
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	err := withdrawRequest.Validate()
	if err != nil {
		return nil, err
	}
	var withdrawalResp *WithdrawalItem
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		withdrawalResp, err = e.WsCreateWithdrawal(withdrawRequest.Currency, withdrawRequest.Amount, withdrawRequest.Crypto.Address, withdrawRequest.Crypto.AddressTag, withdrawRequest.Crypto.Chain, withdrawRequest.ClientOrderID)
	} else {
		withdrawalResp, err = e.WithdrawFunds(ctx, withdrawRequest.Currency, withdrawRequest.Amount, withdrawRequest.Crypto.Address, withdrawRequest.Crypto.AddressTag, withdrawRequest.Crypto.Chain, withdrawRequest.ClientOrderID)
	}
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   e.Name,
		ID:     withdrawalResp.ID,
		Status: withdrawalResp.Status,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}

	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, false)
	if err != nil {
		return nil, err
	}
	switch getOrdersRequest.AssetType {
	case asset.Margin, asset.Spot:
		var orders *PersonalOrdersResponse
		var err error
		if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			orders, err = e.WsRetrivePersonalOpenOrders("")
		} else {
			orders, err = e.GetPersonalOpenOrders(ctx, "")
		}
		if err != nil {
			return nil, err
		}
		resp := []order.Detail{}
		for x := range orders.OrderList {
			cp, err := currency.NewPairFromString(orders.OrderList[x].InstrumentName)
			if err != nil {
				return nil, err
			}
			if len(orders.OrderList) != 0 {
				found := false
				for b := range getOrdersRequest.Pairs {
					if cp.Equal(getOrdersRequest.Pairs[b].Format(pairFormat)) {
						found = true
					}
				}
				if !found {
					continue
				}
			}
			orderType, err := StringToOrderType(orders.OrderList[x].Type)
			if err != nil {
				return nil, err
			}
			side, err := order.StringToOrderSide(orders.OrderList[x].Side)
			if err != nil {
				return nil, err
			}
			status, err := order.StringToOrderStatus(orders.OrderList[x].Status)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				Price:                orders.OrderList[x].Price,
				AverageExecutedPrice: orders.OrderList[x].AvgPrice,
				Amount:               orders.OrderList[x].CumulativeQuantity,
				ExecutedAmount:       orders.OrderList[x].Quantity,
				RemainingAmount:      orders.OrderList[x].CumulativeQuantity - orders.OrderList[x].Quantity,
				Exchange:             e.Name,
				OrderID:              orders.OrderList[x].OrderID,
				ClientOrderID:        orders.OrderList[x].ClientOid,
				Status:               status,
				Side:                 side,
				Type:                 orderType,
				AssetType:            getOrdersRequest.AssetType,
				Date:                 orders.OrderList[x].CreateTime.Time(),
				LastUpdated:          orders.OrderList[x].UpdateTime.Time(),
				Pair:                 cp,
				TimeInForce:          orders.OrderList[x].TimeInForce,
			})
		}
		return getOrdersRequest.Filter(e.Name, resp), nil
	case asset.PerpetualSwap:
		var contingencyType string
		if getOrdersRequest.Type == order.OCO {
			contingencyType = "OCO"
		}
		var symbol string
		if len(getOrdersRequest.Pairs) == 1 {
			symbol = pairFormat.Format(getOrdersRequest.Pairs[0])
		}
		result, err := e.GetFuturesOrderList(ctx, contingencyType, "", symbol)
		if err != nil {
			return nil, err
		}
		resp := make([]order.Detail, 0, len(result.Data))
		for d := range result.Data {
			if len(getOrdersRequest.Pairs) == 1 && result.Data[d].InstrumentName != symbol {
				continue
			}
			cp, err := currency.NewPairFromString(result.Data[d].InstrumentName)
			if err != nil {
				return nil, err
			}
			if len(getOrdersRequest.Pairs) > 0 {
				found := false
				for p := range getOrdersRequest.Pairs {
					if getOrdersRequest.Pairs[p].Equal(cp) {
						found = true
					}
				}
				if !found {
					continue
				}
			}
			oType, err := StringToOrderType(result.Data[d].OrderType)
			if err != nil {
				return nil, err
			}
			oSide, err := order.StringToOrderSide(result.Data[d].Side)
			if err != nil {
				return nil, err
			}
			oStatus, err := order.StringToOrderStatus(result.Data[d].Status)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				TimeInForce:          result.Data[d].TimeInForce,
				Price:                result.Data[d].Price.Float64(),
				Amount:               result.Data[d].Quantity.Float64(),
				ContractAmount:       result.Data[d].CumulativeValue.Float64(),
				AverageExecutedPrice: result.Data[d].AvgPrice.Float64(),
				Exchange:             e.Name,
				OrderID:              result.Data[d].OrderID,
				ClientOrderID:        result.Data[d].ClientOrderID,
				AccountID:            result.Data[d].AccountID,
				Type:                 oType,
				Side:                 oSide,
				Status:               oStatus,
				AssetType:            asset.PerpetualSwap,
				LastUpdated:          result.Data[d].UpdateTime.Time(),
				Pair:                 cp,
			})
		}
		return getOrdersRequest.Filter(e.Name, resp), nil
	default:
		return nil, fmt.Errorf("%w; asset type: %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(getOrdersRequest.AssetType, false)
	if err != nil {
		return nil, err
	}
	var orders *PersonalOrdersResponse
	if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		orders, err = e.WsRetrivePersonalOrderHistory("", getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0, 0)
	} else {
		orders, err = e.GetPersonalOrderHistory(ctx, "", getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0, 0)
	}
	if err != nil {
		return nil, err
	}
	resp := []order.Detail{}
	for x := range orders.OrderList {
		cp, err := currency.NewPairFromString(orders.OrderList[x].InstrumentName)
		if err != nil {
			return nil, err
		}
		if len(orders.OrderList) != 0 {
			found := false
			for b := range getOrdersRequest.Pairs {
				if cp.Equal(getOrdersRequest.Pairs[b].Format(pairFormat)) {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		orderType, err := StringToOrderType(orders.OrderList[x].Type)
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(orders.OrderList[x].Side)
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(orders.OrderList[x].Status)
		if err != nil {
			return nil, err
		}
		resp = append(resp, order.Detail{
			Price:                orders.OrderList[x].Price,
			AverageExecutedPrice: orders.OrderList[x].AvgPrice,
			Amount:               orders.OrderList[x].CumulativeQuantity,
			ExecutedAmount:       orders.OrderList[x].Quantity,
			RemainingAmount:      orders.OrderList[x].CumulativeQuantity - orders.OrderList[x].Quantity,
			Exchange:             e.Name,
			OrderID:              orders.OrderList[x].OrderID,
			ClientOrderID:        orders.OrderList[x].ClientOid,
			Status:               status,
			Side:                 side,
			Type:                 orderType,
			AssetType:            getOrdersRequest.AssetType,
			Date:                 orders.OrderList[x].CreateTime.Time(),
			LastUpdated:          orders.OrderList[x].UpdateTime.Time(),
			Pair:                 cp,
			TimeInForce:          orders.OrderList[x].TimeInForce,
		})
	}
	return getOrdersRequest.Filter(e.Name, resp), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder) * feeBuilder.Amount * feeBuilder.PurchasePrice
	case exchange.CryptocurrencyWithdrawalFee:
		fee = 0.5 * feeBuilder.PurchasePrice * feeBuilder.Amount
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0750 * price * amount
}

// calculateTradingFee return fee based on users current fee tier or default values
func calculateTradingFee(feeBuilder *exchange.FeeBuilder) float64 {
	switch {
	case feeBuilder.Amount*feeBuilder.PurchasePrice <= 250:
		return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.075
	case feeBuilder.Amount*feeBuilder.PurchasePrice < 1000000:
		if feeBuilder.IsMaker {
			return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.07
		}
		return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.072
	case feeBuilder.Amount*feeBuilder.PurchasePrice < 5000000:
		if feeBuilder.IsMaker {
			return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.065
		}
		return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.069
	case feeBuilder.Amount*feeBuilder.PurchasePrice <= 10000000:
		if feeBuilder.IsMaker {
			return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.06
		}
		return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.065
	default:
		if !feeBuilder.IsMaker {
			return feeBuilder.PurchasePrice * feeBuilder.Amount * 0.05
		}
		return 0
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.GetCandlestickDetail(ctx, req.RequestFormatted.String(), interval, 0, start, end)
	if err != nil {
		return nil, err
	}
	candleElements := make([]kline.Candle, len(candles.Data))
	for x := range candles.Data {
		candleElements[x] = kline.Candle{
			Time:   candles.Data[x].EndTime.Time(),
			Open:   candles.Data[x].Open,
			High:   candles.Data[x].High,
			Low:    candles.Data[x].Low,
			Close:  candles.Data[x].Close,
			Volume: candles.Data[x].Volume,
		}
	}
	return req.ProcessResponse(candleElements)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// ValidateAPICredentials validates current credentials used for wrapper
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountInfo(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(_ context.Context, _ *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w asset type %v", asset.ErrNotSupported, a)
	}
	instrumentsResponse, err := e.GetInstruments(ctx)
	if err != nil {
		return err
	}

	ls := make([]limits.MinMaxLevel, 0, len(instrumentsResponse.Instruments))
	for x := range instrumentsResponse.Instruments {
		pair, err := currency.NewPairFromString(instrumentsResponse.Instruments[x].Symbol)
		if err != nil {
			return err
		}
		ls = append(ls, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
			AmountStepIncrementSize: instrumentsResponse.Instruments[x].QtyTickSize.Float64(),
			PriceStepIncrementSize:  instrumentsResponse.Instruments[x].PriceTickSize.Float64(),
		})
	}
	return limits.Load(ls)
}

func priceTypeToString(pt order.PriceType) (string, error) {
	switch pt {
	case order.IndexPrice:
		return "INDEX_PRICE", nil
	case order.MarkPrice:
		return "MARK_PRICE", nil
	case order.LastPrice:
		return "LAST_PRICE", nil
	case order.UnsetPriceType:
		return "", nil
	default:
		return "", fmt.Errorf("%w, price type: %v", order.ErrUnknownPriceType, pt.String())
	}
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
// differs by exchange
func (e *Exchange) IsPerpetualFutureCurrency(assetType asset.Item, pair currency.Pair) (bool, error) {
	if pair.IsEmpty() {
		return false, currency.ErrCurrencyPairEmpty
	}
	if assetType != asset.Futures {
		// deribit considers future combo, even if ending in "PERP" to not be a perpetual
		return false, nil
	}
	return strings.HasSuffix(pair.Quote.String(), "PERP"), nil
}
