package deribit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
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

// SetDefaults sets the basic defaults for Deribit
func (d *Deribit) SetDefaults() {
	d.Name = "Deribit"
	d.Enabled = true
	d.Verbose = true
	d.API.CredentialsValidator.RequiresKey = true
	d.API.CredentialsValidator.RequiresSecret = true

	dashFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	underscoreFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	if err := d.SetAssetPairStore(asset.Spot, currency.PairStore{AssetEnabled: true, RequestFormat: underscoreFormat, ConfigFormat: underscoreFormat}); err != nil {
		log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", d.Name, asset.Spot, err)
	}
	for _, a := range []asset.Item{asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo} {
		if err := d.SetAssetPairStore(a, currency.PairStore{AssetEnabled: true, RequestFormat: dashFormat, ConfigFormat: dashFormat}); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", d.Name, a, err)
		}
	}

	// Fill out the capabilities/features that the exchange supports
	d.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:    true,
				Leverage:     true,
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour:   true,
					kline.EightHour: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.HundredMilliseconds},
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.TenMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					// NOTE: The supported time intervals below are returned
					// offset to +8 hours. This may lead to
					// issues with candle quality and conversion as the
					// intervals may be broken up. The below intervals
					// are therefore constructed from the intervals above.
					// kline.IntervalCapacity{Interval: kline.ThreeHour},
					// kline.IntervalCapacity{Interval: kline.SixHour},
					// kline.IntervalCapacity{Interval: kline.TwelveHour},
					// kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 500,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	d.Requester, err = request.New(d.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	for _, assetType := range []asset.Item{asset.Options, asset.OptionCombo, asset.FutureCombo} {
		if err = d.DisableAssetWebsocketSupport(assetType); err != nil {
			log.Errorln(log.ExchangeSys, err)
		}
	}
	d.API.Endpoints = d.NewEndpoints()
	err = d.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestFutures:           "https://www.deribit.com",
		exchange.RestSpot:              "https://www.deribit.com",
		exchange.RestSpotSupplementary: "https://test.deribit.com",
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	d.Websocket = websocket.NewManager()
	d.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	d.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	d.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (d *Deribit) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		d.SetEnabled(false)
		return nil
	}
	err = d.SetupDefaults(exch)
	if err != nil {
		return err
	}
	err = d.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            deribitWebsocketAddress,
		RunningURL:            deribitWebsocketAddress,
		Connector:             d.WsConnect,
		Subscriber:            d.Subscribe,
		Unsubscriber:          d.Unsubscribe,
		GenerateSubscriptions: d.generateSubscriptions,
		Features:              &d.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	return d.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  d.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (d *Deribit) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, assetType)
	}

	instruments, err := d.GetInstruments(ctx, currency.EMPTYCODE, d.GetAssetKind(assetType), false)
	if err != nil {
		return nil, err
	}

	resp := make(currency.Pairs, 0, len(instruments))
	for _, inst := range instruments {
		if !inst.IsActive {
			continue
		}
		cp, err := currency.NewPairFromString(inst.InstrumentName)
		if err != nil {
			return nil, err
		}
		resp = resp.Add(cp)
	}
	return resp, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (d *Deribit) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := d.GetAssetTypes(false)
	errs := common.CollectErrors(len(assets))
	for x := range assets {
		go func(x int) {
			defer errs.Wg.Done()
			pairs, err := d.FetchTradablePairs(ctx, assets[x])
			if err != nil {
				errs.C <- err
				return
			}
			errs.C <- d.UpdatePairs(pairs, assets[x], false, forceUpdate)
		}(x)
	}
	return errs.Collect()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (d *Deribit) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (d *Deribit) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, assetType)
	}
	p, err := d.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	instrumentID := d.formatPairString(assetType, p)
	var tickerData *TickerData
	if d.Websocket.IsConnected() {
		tickerData, err = d.WSRetrievePublicTicker(ctx, instrumentID)
	} else {
		tickerData, err = d.GetPublicTicker(ctx, instrumentID)
	}
	if err != nil {
		return nil, err
	}
	resp := ticker.Price{
		ExchangeName: d.Name,
		Pair:         p,
		AssetType:    assetType,
		Ask:          tickerData.BestAskPrice,
		AskSize:      tickerData.BestAskAmount,
		Bid:          tickerData.BestBidPrice,
		BidSize:      tickerData.BestBidAmount,
		High:         tickerData.Stats.High,
		Low:          tickerData.Stats.Low,
		Last:         tickerData.LastPrice,
		Volume:       tickerData.Stats.Volume,
		Close:        tickerData.LastPrice,
		IndexPrice:   tickerData.IndexPrice,
		MarkPrice:    tickerData.MarkPrice,
		QuoteVolume:  tickerData.Stats.VolumeUSD,
	}
	err = ticker.ProcessTicker(&resp)
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(d.Name, p, assetType)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (d *Deribit) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	p, err := d.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	instrumentID := d.formatPairString(assetType, p)
	var obData *Orderbook
	if d.Websocket.IsConnected() {
		obData, err = d.WSRetrieveOrderbookData(ctx, instrumentID, 50)
	} else {
		obData, err = d.GetOrderbook(ctx, instrumentID, 50)
	}
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          d.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: d.ValidateOrderbook,
	}
	book.Asks = make(orderbook.Levels, 0, len(obData.Asks))
	for x := range obData.Asks {
		if obData.Asks[x][0] == 0 || obData.Asks[x][1] == 0 {
			continue
		}
		book.Asks = append(book.Asks, orderbook.Level{
			Price:  obData.Asks[x][0],
			Amount: obData.Asks[x][1],
		})
	}
	book.Bids = make(orderbook.Levels, 0, len(obData.Bids))
	for x := range obData.Bids {
		if obData.Bids[x][0] == 0 || obData.Bids[x][1] == 0 {
			continue
		}
		book.Bids = append(book.Bids, orderbook.Level{
			Price:  obData.Bids[x][0],
			Amount: obData.Bids[x][1],
		})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(d.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (d *Deribit) UpdateAccountInfo(ctx context.Context, _ asset.Item) (account.Holdings, error) {
	var resp account.Holdings
	resp.Exchange = d.Name
	currencies, err := d.GetCurrencies(ctx)
	if err != nil {
		return resp, err
	}
	resp.Accounts = make([]account.SubAccount, len(currencies))
	for x := range currencies {
		var data *AccountSummaryData
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			data, err = d.WSRetrieveAccountSummary(ctx, currency.NewCode(currencies[x].Currency), false)
		} else {
			data, err = d.GetAccountSummary(ctx, currency.NewCode(currencies[x].Currency), false)
		}
		if err != nil {
			return resp, err
		}
		var subAcc account.SubAccount
		subAcc.Currencies = append(subAcc.Currencies, account.Balance{
			Currency: currency.NewCode(currencies[x].Currency),
			Total:    data.Balance,
			Hold:     data.Balance - data.AvailableFunds,
		})
		resp.Accounts[x] = subAcc
	}
	return resp, nil
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (d *Deribit) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	var currencies []CurrencyData
	var err error
	if d.Websocket.IsConnected() {
		currencies, err = d.WSRetrieveCurrencies(ctx)
	} else {
		currencies, err = d.GetCurrencies(ctx)
	}
	if err != nil {
		return nil, err
	}
	var resp []exchange.FundingHistory
	for x := range currencies {
		var deposits *DepositsData
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			deposits, err = d.WSRetrieveDeposits(ctx, currency.NewCode(currencies[x].Currency), 100, 0)
		} else {
			deposits, err = d.GetDeposits(ctx, currency.NewCode(currencies[x].Currency), 100, 0)
		}
		if err != nil {
			return nil, err
		}
		for y := range deposits.Data {
			resp = append(resp, exchange.FundingHistory{
				ExchangeName:    d.Name,
				Status:          deposits.Data[y].State,
				TransferID:      deposits.Data[y].TransactionID,
				Timestamp:       deposits.Data[y].UpdatedTimestamp.Time(),
				Currency:        currencies[x].Currency,
				Amount:          deposits.Data[y].Amount,
				CryptoToAddress: deposits.Data[y].Address,
				TransferType:    "deposit",
			})
		}
		var withdrawalData *WithdrawalsData
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			withdrawalData, err = d.WSRetrieveWithdrawals(ctx, currency.NewCode(currencies[x].Currency), 100, 0)
		} else {
			withdrawalData, err = d.GetWithdrawals(ctx, currency.NewCode(currencies[x].Currency), 100, 0)
		}
		if err != nil {
			return nil, err
		}
		for z := range withdrawalData.Data {
			resp = append(resp, exchange.FundingHistory{
				ExchangeName:    d.Name,
				Status:          withdrawalData.Data[z].State,
				TransferID:      withdrawalData.Data[z].TransactionID,
				Timestamp:       withdrawalData.Data[z].UpdatedTimestamp.Time(),
				Currency:        currencies[x].Currency,
				Amount:          withdrawalData.Data[z].Amount,
				CryptoToAddress: withdrawalData.Data[z].Address,
				TransferType:    "withdrawal",
			})
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (d *Deribit) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	var currencies []CurrencyData
	var err error
	if d.Websocket.IsConnected() {
		currencies, err = d.WSRetrieveCurrencies(ctx)
	} else {
		currencies, err = d.GetCurrencies(ctx)
	}
	if err != nil {
		return nil, err
	}
	resp := []exchange.WithdrawalHistory{}
	for x := range currencies {
		if !strings.EqualFold(currencies[x].Currency, c.String()) {
			continue
		}
		var withdrawalData *WithdrawalsData
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			withdrawalData, err = d.WSRetrieveWithdrawals(ctx, currency.NewCode(currencies[x].Currency), 100, 0)
		} else {
			withdrawalData, err = d.GetWithdrawals(ctx, currency.NewCode(currencies[x].Currency), 100, 0)
		}
		if err != nil {
			return nil, err
		}
		for y := range withdrawalData.Data {
			resp = append(resp, exchange.WithdrawalHistory{
				Status:          withdrawalData.Data[y].State,
				TransferID:      withdrawalData.Data[y].TransactionID,
				Timestamp:       withdrawalData.Data[y].UpdatedTimestamp.Time(),
				Currency:        currencies[x].Currency,
				Amount:          withdrawalData.Data[y].Amount,
				CryptoToAddress: withdrawalData.Data[y].Address,
				TransferType:    "deposit",
			})
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (d *Deribit) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, assetType)
	}
	p, err := d.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	instrumentID := d.formatPairString(assetType, p)
	resp := []trade.Data{}
	var trades *PublicTradesData
	if d.Websocket.IsConnected() {
		trades, err = d.WSRetrieveLastTradesByInstrument(ctx, instrumentID, "", "", "", 0, false)
	} else {
		trades, err = d.GetLastTradesByInstrument(ctx, instrumentID, "", "", "", 0, false)
	}
	if err != nil {
		return nil, err
	}
	for a := range trades.Trades {
		sideData := order.Sell
		if trades.Trades[a].Direction == sideBUY {
			sideData = order.Buy
		}
		resp = append(resp, trade.Data{
			TID:          trades.Trades[a].TradeID,
			Exchange:     d.Name,
			Price:        trades.Trades[a].Price,
			Amount:       trades.Trades[a].Amount,
			Timestamp:    trades.Trades[a].Timestamp.Time(),
			AssetType:    assetType,
			Side:         sideData,
			CurrencyPair: p,
		})
	}
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (d *Deribit) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if common.StartEndTimeCheck(timestampStart, timestampEnd) != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v",
			timestampStart,
			timestampEnd)
	}
	p, err := d.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var instrumentID string
	switch assetType {
	case asset.Futures, asset.Options, asset.Spot:
		instrumentID = d.formatPairString(assetType, p)
	default:
		return nil, fmt.Errorf("%w asset type %v", asset.ErrNotSupported, assetType)
	}
	var resp []trade.Data
	var tradesData *PublicTradesData
	hasMore := true
	for hasMore {
		if d.Websocket.IsConnected() {
			tradesData, err = d.WSRetrieveLastTradesByInstrumentAndTime(ctx, instrumentID, "asc", 100, true, timestampStart, timestampEnd)
		} else {
			tradesData, err = d.GetLastTradesByInstrumentAndTime(ctx, instrumentID, "asc", 100, timestampStart, timestampEnd)
		}
		if err != nil {
			return nil, err
		}
		if len(tradesData.Trades) != 100 {
			hasMore = false
		}
		for t := range tradesData.Trades {
			if t == 99 {
				if timestampStart.Equal(tradesData.Trades[t].Timestamp.Time()) {
					hasMore = false
				}
				timestampStart = tradesData.Trades[t].Timestamp.Time()
			}
			sideData := order.Sell
			if tradesData.Trades[t].Direction == sideBUY {
				sideData = order.Buy
			}
			resp = append(resp, trade.Data{
				TID:          tradesData.Trades[t].TradeID,
				Exchange:     d.Name,
				Price:        tradesData.Trades[t].Price,
				Amount:       tradesData.Trades[t].Amount,
				Timestamp:    tradesData.Trades[t].Timestamp.Time(),
				AssetType:    assetType,
				Side:         sideData,
				CurrencyPair: p,
			})
		}
	}
	return resp, nil
}

// SubmitOrder submits a new order
func (d *Deribit) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(d.GetTradingRequirements())
	if err != nil {
		return nil, err
	}
	if !d.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%s: orderType %v is not valid", d.Name, s.AssetType)
	}
	var orderID string
	var fmtPair currency.Pair
	status := order.New
	fmtPair, err = d.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	timeInForce := ""
	if s.TimeInForce.Is(order.ImmediateOrCancel) {
		timeInForce = "immediate_or_cancel"
	}
	var data *PrivateTradeData
	reqParams := &OrderBuyAndSellParams{
		Instrument:   fmtPair.String(),
		OrderType:    strings.ToLower(s.Type.String()),
		Label:        s.ClientOrderID,
		TimeInForce:  timeInForce,
		Amount:       s.Amount,
		Price:        s.Price,
		TriggerPrice: s.TriggerPrice,
		PostOnly:     s.TimeInForce.Is(order.PostOnly),
		ReduceOnly:   s.ReduceOnly,
	}
	switch {
	case s.Side.IsLong():
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			data, err = d.WSSubmitBuy(ctx, reqParams)
		} else {
			data, err = d.SubmitBuy(ctx, reqParams)
		}
		if err != nil {
			return nil, err
		}
		if data == nil {
			return nil, common.ErrNoResponse
		}
		orderID = data.Order.OrderID
	case s.Side.IsShort():
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			data, err = d.WSSubmitSell(ctx, reqParams)
		} else {
			data, err = d.SubmitSell(ctx, reqParams)
		}
		if err != nil {
			return nil, err
		}
		if data == nil {
			return nil, common.ErrNoResponse
		}
		orderID = data.Order.OrderID
	}
	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = status
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (d *Deribit) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if !d.SupportsAsset(action.AssetType) {
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, action.AssetType)
	}
	var modify *PrivateTradeData
	var err error
	reqParam := &OrderBuyAndSellParams{
		TriggerPrice: action.TriggerPrice,
		PostOnly:     action.TimeInForce.Is(order.PostOnly),
		Amount:       action.Amount,
		OrderID:      action.OrderID,
		Price:        action.Price,
	}
	if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		modify, err = d.WSSubmitEdit(ctx, reqParam)
	} else {
		modify, err = d.SubmitEdit(ctx, reqParam)
	}
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = modify.Order.OrderID
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (d *Deribit) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if !d.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, ord.AssetType)
	}
	err := ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = d.WSSubmitCancel(ctx, ord.OrderID)
	} else {
		_, err = d.SubmitCancel(ctx, ord.OrderID)
	}
	if err != nil {
		return err
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (d *Deribit) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (d *Deribit) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelData *MultipleCancelResponse
	pairFmt, err := d.GetPairFormat(orderCancellation.AssetType, true)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	var orderTypeStr string
	switch orderCancellation.Type {
	case order.Limit:
		orderTypeStr = order.Limit.String()
	case order.Market:
		orderTypeStr = order.Market.String()
	case order.AnyType, order.UnknownType:
		orderTypeStr = "all"
	default:
		return order.CancelAllResponse{}, fmt.Errorf("%s: orderType %v is not valid", d.Name, orderCancellation.Type)
	}
	if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		cancelData, err = d.WSSubmitCancelAllByInstrument(ctx, pairFmt.Format(orderCancellation.Pair), orderTypeStr, true, true)
	} else {
		cancelData, err = d.SubmitCancelAllByInstrument(ctx, pairFmt.Format(orderCancellation.Pair), orderTypeStr, true, true)
	}
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	response := order.CancelAllResponse{Count: cancelData.CancelCount}
	if len(cancelData.CancelDetails) > 0 {
		response.Status = make(map[string]string)
		for a := range cancelData.CancelDetails {
			for b := range cancelData.CancelDetails[a].Result {
				response.Status[cancelData.CancelDetails[a].Result[b].OrderID] = cancelData.CancelDetails[a].Result[b].OrderState
			}
		}
	}
	return response, nil
}

// GetOrderInfo returns order information based on order ID
func (d *Deribit) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w assetType %v", asset.ErrNotSupported, assetType)
	}
	var orderInfo *OrderData
	var err error
	if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		orderInfo, err = d.WSRetrievesOrderState(ctx, orderID)
	} else {
		orderInfo, err = d.GetOrderState(ctx, orderID)
	}
	if err != nil {
		return nil, err
	}
	orderSide := order.Sell
	if orderInfo.Direction == sideBUY {
		orderSide = order.Buy
	}
	orderType, err := order.StringToOrderType(orderInfo.OrderType)
	if err != nil {
		return nil, err
	}
	var pair currency.Pair
	pair, err = currency.NewPairFromString(orderInfo.InstrumentName)
	if err != nil {
		return nil, err
	}
	var orderStatus order.Status
	if orderInfo.OrderState == "untriggered" {
		orderStatus = order.UnknownStatus
	} else {
		orderStatus, err = order.StringToOrderStatus(orderInfo.OrderState)
		if err != nil {
			return nil, fmt.Errorf("%v: orderStatus %s not supported", d.Name, orderInfo.OrderState)
		}
	}
	var tif order.TimeInForce
	tif, err = timeInForceFromString(orderInfo.TimeInForce, orderInfo.PostOnly)
	if err != nil {
		return nil, err
	}
	return &order.Detail{
		AssetType:       assetType,
		Exchange:        d.Name,
		TimeInForce:     tif,
		Price:           orderInfo.Price,
		Amount:          orderInfo.Amount,
		ExecutedAmount:  orderInfo.FilledAmount,
		Fee:             orderInfo.Commission,
		RemainingAmount: orderInfo.Amount - orderInfo.FilledAmount,
		OrderID:         orderInfo.OrderID,
		Pair:            pair,
		LastUpdated:     orderInfo.LastUpdateTimestamp.Time(),
		Side:            orderSide,
		Type:            orderType,
		Status:          orderStatus,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (d *Deribit) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	var addressData *DepositAddressData
	var err error
	if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		addressData, err = d.WSRetrieveCurrentDepositAddress(ctx, cryptocurrency)
	} else {
		addressData, err = d.GetCurrentDepositAddress(ctx, cryptocurrency)
	}
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: addressData.Address,
		Chain:   addressData.Currency,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (d *Deribit) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	err := withdrawRequest.Validate()
	if err != nil {
		return nil, err
	}
	var withdrawData *WithdrawData
	if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		withdrawData, err = d.WSSubmitWithdraw(ctx, withdrawRequest.Currency, withdrawRequest.Crypto.Address, "", withdrawRequest.Amount)
	} else {
		withdrawData, err = d.SubmitWithdraw(ctx, withdrawRequest.Currency, withdrawRequest.Crypto.Address, "", withdrawRequest.Amount)
	}
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     strconv.FormatInt(withdrawData.ID, 10),
		Status: withdrawData.State,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (d *Deribit) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (d *Deribit) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (d *Deribit) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	if !d.SupportsAsset(getOrdersRequest.AssetType) {
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
	if len(getOrdersRequest.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	resp := []order.Detail{}
	for x := range getOrdersRequest.Pairs {
		fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
		if err != nil {
			return nil, err
		}
		var oTypeString string
		switch getOrdersRequest.Type {
		case order.AnyType, order.UnknownType:
			oTypeString = "all"
		default:
			oTypeString = getOrdersRequest.Type.Lower()
		}
		var ordersData []OrderData
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			ordersData, err = d.WSRetrieveOpenOrdersByInstrument(ctx, fmtPair.String(), oTypeString)
		} else {
			ordersData, err = d.GetOpenOrdersByInstrument(ctx, fmtPair.String(), oTypeString)
		}
		if err != nil {
			return nil, err
		}
		for y := range ordersData {
			orderSide := order.Sell
			if ordersData[y].Direction == sideBUY {
				orderSide = order.Buy
			}
			if getOrdersRequest.Side != orderSide && getOrdersRequest.Side != order.AnySide {
				continue
			}
			orderType, err := order.StringToOrderType(ordersData[y].OrderType)
			if err != nil {
				return nil, err
			}
			if getOrdersRequest.Type != orderType && getOrdersRequest.Type != order.AnyType {
				continue
			}
			var orderStatus order.Status
			ordersData[y].OrderState = strings.ToLower(ordersData[y].OrderState)
			if ordersData[y].OrderState != "open" {
				continue
			}

			var tif order.TimeInForce
			tif, err = timeInForceFromString(ordersData[y].TimeInForce, ordersData[y].PostOnly)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				AssetType:       getOrdersRequest.AssetType,
				Exchange:        d.Name,
				Price:           ordersData[y].Price,
				Amount:          ordersData[y].Amount,
				ExecutedAmount:  ordersData[y].FilledAmount,
				Fee:             ordersData[y].Commission,
				RemainingAmount: ordersData[y].Amount - ordersData[y].FilledAmount,
				OrderID:         ordersData[y].OrderID,
				Pair:            getOrdersRequest.Pairs[x],
				LastUpdated:     ordersData[y].LastUpdateTimestamp.Time(),
				Side:            orderSide,
				Type:            orderType,
				Status:          orderStatus,
				TimeInForce:     tif,
			})
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (d *Deribit) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	if len(getOrdersRequest.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
		if err != nil {
			return nil, err
		}
		var ordersData []OrderData
		if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			ordersData, err = d.WSRetrieveOrderHistoryByInstrument(ctx, fmtPair.String(), 100, 0, true, true)
		} else {
			ordersData, err = d.GetOrderHistoryByInstrument(ctx, fmtPair.String(), 100, 0, true, true)
		}
		if err != nil {
			return nil, err
		}
		for y := range ordersData {
			orderSide := order.Sell
			if ordersData[y].Direction == sideBUY {
				orderSide = order.Buy
			}
			if getOrdersRequest.Side != orderSide && getOrdersRequest.Side != order.AnySide {
				continue
			}
			orderType, err := order.StringToOrderType(ordersData[y].OrderType)
			if err != nil {
				return nil, err
			}
			if getOrdersRequest.Type != orderType && getOrdersRequest.Type != order.AnyType {
				continue
			}
			var orderStatus order.Status
			if ordersData[y].OrderState == "untriggered" {
				orderStatus = order.UnknownStatus
			} else {
				orderStatus, err = order.StringToOrderStatus(ordersData[y].OrderState)
				if err != nil {
					return resp, fmt.Errorf("%v: orderStatus %s not supported", d.Name, ordersData[y].OrderState)
				}
			}

			var tif order.TimeInForce
			tif, err = timeInForceFromString(ordersData[y].TimeInForce, ordersData[y].PostOnly)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				AssetType:       getOrdersRequest.AssetType,
				Exchange:        d.Name,
				Price:           ordersData[y].Price,
				Amount:          ordersData[y].Amount,
				ExecutedAmount:  ordersData[y].FilledAmount,
				Fee:             ordersData[y].Commission,
				RemainingAmount: ordersData[y].Amount - ordersData[y].FilledAmount,
				OrderID:         ordersData[y].OrderID,
				Pair:            getOrdersRequest.Pairs[x],
				LastUpdated:     ordersData[y].LastUpdateTimestamp.Time(),
				Side:            orderSide,
				Type:            orderType,
				Status:          orderStatus,
				TimeInForce:     tif,
			})
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (d *Deribit) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !d.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	var fee float64
	var err error
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee, err = calculateTradingFee(feeBuilder)
		if err != nil {
			return 0, err
		}
	case exchange.CryptocurrencyDepositFee:
	case exchange.CryptocurrencyWithdrawalFee:
		// Withdrawals are processed instantly if the balance in our hot wallet permits so. We keep only a small percentage of coins in hot storage,
		// therefore there is a chance that your withdrawal cannot be processed immediately. If needed, once a day we will replenish the balance of the hot wallet from the cold storage.
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (d *Deribit) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := d.UpdateAccountInfo(ctx, assetType)
	return d.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (d *Deribit) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := d.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	intervalString, err := d.GetResolutionFromInterval(req.ExchangeInterval)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Futures, asset.Spot:
		var tradingViewData *TVChartData
		if d.Websocket.IsConnected() {
			tradingViewData, err = d.WSRetrievesTradingViewChartData(ctx, d.formatFuturesTradablePair(req.RequestFormatted), intervalString, start, end)
		} else {
			tradingViewData, err = d.GetTradingViewChart(ctx, d.formatFuturesTradablePair(req.RequestFormatted), intervalString, start, end)
		}
		if err != nil {
			return nil, err
		} else if len(tradingViewData.Ticks) == 0 {
			return nil, kline.ErrNoTimeSeriesDataToConvert
		}
		checkLen := len(tradingViewData.Ticks)
		if len(tradingViewData.Open) != checkLen ||
			len(tradingViewData.High) != checkLen ||
			len(tradingViewData.Low) != checkLen ||
			len(tradingViewData.Close) != checkLen ||
			len(tradingViewData.Volume) != checkLen {
			return nil, fmt.Errorf("%s - %v: invalid trading view chart data received", a, req.RequestFormatted)
		}
		listCandles := make([]kline.Candle, 0, len(tradingViewData.Ticks))
		for x := range tradingViewData.Ticks {
			timeInfo := time.UnixMilli(tradingViewData.Ticks[x]).UTC()
			if timeInfo.Before(start) {
				continue
			}
			listCandles = append(listCandles, kline.Candle{
				Open:   tradingViewData.Open[x],
				High:   tradingViewData.High[x],
				Low:    tradingViewData.Low[x],
				Close:  tradingViewData.Close[x],
				Volume: tradingViewData.Volume[x],
				Time:   timeInfo,
			})
		}
		return req.ProcessResponse(listCandles)
	case asset.OptionCombo, asset.FutureCombo, asset.Options:
		// TODO: candlestick data for asset item option_combo, future_combo, and option not supported yet
	}
	return nil, fmt.Errorf("%w candlestick data for asset type %v", asset.ErrNotSupported, a)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (d *Deribit) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := d.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var tradingViewData *TVChartData
	timeSeries := make([]kline.Candle, 0, req.Size())
	switch a {
	case asset.Futures, asset.Spot:
		for x := range req.RangeHolder.Ranges {
			intervalString, err := d.GetResolutionFromInterval(req.ExchangeInterval)
			if err != nil {
				return nil, err
			}
			if d.Websocket.IsConnected() {
				tradingViewData, err = d.WSRetrievesTradingViewChartData(ctx, d.formatFuturesTradablePair(req.RequestFormatted), intervalString, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			} else {
				tradingViewData, err = d.GetTradingViewChart(ctx, d.formatFuturesTradablePair(req.RequestFormatted), intervalString, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			}
			if err != nil {
				return nil, err
			}
			checkLen := len(tradingViewData.Ticks)
			if len(tradingViewData.Open) != checkLen ||
				len(tradingViewData.High) != checkLen ||
				len(tradingViewData.Low) != checkLen ||
				len(tradingViewData.Close) != checkLen ||
				len(tradingViewData.Volume) != checkLen {
				return nil, fmt.Errorf("%s - %v: invalid trading view chart data received", a, d.formatFuturesTradablePair(req.RequestFormatted))
			}
			for i := range tradingViewData.Ticks {
				timeInfo := time.UnixMilli(tradingViewData.Ticks[i]).UTC()
				if timeInfo.Before(start) {
					continue
				}
				timeSeries = append(timeSeries, kline.Candle{
					Open:   tradingViewData.Open[i],
					High:   tradingViewData.High[i],
					Low:    tradingViewData.Low[i],
					Close:  tradingViewData.Close[i],
					Volume: tradingViewData.Volume[i],
					Time:   timeInfo,
				})
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.OptionCombo, asset.FutureCombo, asset.Options:
		// TODO: candlestick data for asset item option_combo, future_combo, and option not supported yet
	}
	return nil, fmt.Errorf("%w candlestick data for asset type %v", asset.ErrNotSupported, a)
}

// GetServerTime returns the current exchange server time.
func (d *Deribit) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return d.GetTime(ctx)
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (d *Deribit) AuthenticateWebsocket(ctx context.Context) error {
	return d.wsLogin(ctx)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (d *Deribit) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	resp := []futures.Contract{}
	for _, ccy := range baseCurrencies {
		var marketSummary []*InstrumentData
		var err error
		if d.Websocket.IsConnected() {
			marketSummary, err = d.WSRetrieveInstrumentsData(ctx, currency.NewCode(ccy), d.GetAssetKind(item), false)
		} else {
			marketSummary, err = d.GetInstruments(ctx, currency.NewCode(ccy), d.GetAssetKind(item), false)
		}
		if err != nil {
			return nil, err
		}
		for _, inst := range marketSummary {
			if inst.Kind != "future" && inst.Kind != "future_combo" {
				continue
			}
			cp, err := currency.NewPairFromString(inst.InstrumentName)
			if err != nil {
				return nil, err
			}
			var ct futures.ContractType
			switch inst.SettlementPeriod {
			case "day":
				ct = futures.Daily
			case "week":
				ct = futures.Weekly
			case "month":
				ct = futures.Monthly
			case "perpetual":
				ct = futures.Perpetual
			}
			var contractSettlementType futures.ContractSettlementType
			if inst.InstrumentType == "reversed" {
				contractSettlementType = futures.Inverse
			} else {
				contractSettlementType = futures.Linear
			}
			resp = append(resp, futures.Contract{
				Exchange:             d.Name,
				Name:                 cp,
				Underlying:           currency.NewPair(currency.NewCode(inst.BaseCurrency), currency.NewCode(inst.QuoteCurrency)),
				Asset:                item,
				SettlementCurrencies: []currency.Code{currency.NewCode(inst.SettlementCurrency)},
				StartDate:            inst.CreationTimestamp.Time(),
				EndDate:              inst.ExpirationTimestamp.Time(),
				Type:                 ct,
				SettlementType:       contractSettlementType,
				IsActive:             inst.IsActive,
				MaxLeverage:          inst.MaxLeverage,
				Multiplier:           inst.ContractSize,
			})
		}
	}
	return resp, nil
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (d *Deribit) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !d.SupportsAsset(a) {
		return fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, a)
	}
	for _, x := range baseCurrencies {
		var instrumentsData []*InstrumentData
		var err error
		if d.Websocket.IsConnected() {
			instrumentsData, err = d.WSRetrieveInstrumentsData(ctx, currency.NewCode(x), d.GetAssetKind(a), false)
		} else {
			instrumentsData, err = d.GetInstruments(ctx, currency.NewCode(x), d.GetAssetKind(a), false)
		}
		if err != nil {
			return err
		} else if len(instrumentsData) == 0 {
			continue
		}

		limits := make([]order.MinMaxLevel, len(instrumentsData))
		for x, inst := range instrumentsData {
			var pair currency.Pair
			pair, err = currency.NewPairFromString(inst.InstrumentName)
			if err != nil {
				return err
			}
			limits[x] = order.MinMaxLevel{
				Pair:                   pair,
				Asset:                  a,
				PriceStepIncrementSize: inst.TickSize,
				MinimumBaseAmount:      inst.MinimumTradeAmount,
			}
		}
		err = d.LoadLimits(limits)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFuturesPositionSummary returns position summary details for an active position
func (d *Deribit) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair, err := d.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	var pos []PositionData
	if d.Websocket.IsConnected() && d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		pos, err = d.WSRetrievePositions(ctx, fPair.Base, d.GetAssetKind(r.Asset))
	} else {
		pos, err = d.GetPositions(ctx, fPair.Base, d.GetAssetKind(r.Asset))
	}
	if err != nil {
		return nil, err
	}
	index := -1
	for a := range pos {
		if pos[a].InstrumentName == fPair.String() {
			index = a
			break
		}
	}
	if index == -1 {
		return nil, errors.New("position information for the instrument not found")
	}
	contracts, err := d.GetFuturesContractDetails(ctx, r.Asset)
	if err != nil {
		return nil, err
	}
	var multiplier, contractSize float64
	var settlementType futures.ContractSettlementType
	for i := range contracts {
		if !contracts[i].Name.Equal(fPair) {
			continue
		}
		multiplier = contracts[i].Multiplier
		settlementType = contracts[i].SettlementType
		break
	}

	var baseSize float64
	switch r.Asset {
	case asset.Futures:
		baseSize = pos[index].SizeCurrency
	case asset.Options:
		baseSize = pos[index].Size
	}
	contractSize = multiplier * baseSize

	return &futures.PositionSummary{
		Pair:                      r.Pair,
		Asset:                     r.Asset,
		Currency:                  fPair.Base,
		NotionalSize:              decimal.NewFromFloat(pos[index].MarkPrice),
		Leverage:                  decimal.NewFromFloat(pos[index].Leverage),
		InitialMarginRequirement:  decimal.NewFromFloat(pos[index].InitialMargin),
		EstimatedLiquidationPrice: decimal.NewFromFloat(pos[index].EstimatedLiquidationPrice),
		MarkPrice:                 decimal.NewFromFloat(pos[index].MarkPrice),
		CurrentSize:               decimal.NewFromFloat(baseSize),
		ContractSize:              decimal.NewFromFloat(contractSize),
		ContractMultiplier:        decimal.NewFromFloat(multiplier),
		ContractSettlementType:    settlementType,
		AverageOpenPrice:          decimal.NewFromFloat(pos[index].AveragePrice),
		UnrealisedPNL:             decimal.NewFromFloat(pos[index].TotalProfitLoss - pos[index].RealizedProfitLoss),
		RealisedPNL:               decimal.NewFromFloat(pos[index].RealizedProfitLoss),
		MaintenanceMarginFraction: decimal.NewFromFloat(pos[index].MaintenanceMargin),
	}, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (d *Deribit) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	if len(k) == 0 {
		return nil, fmt.Errorf("%w requires pair", common.ErrFunctionNotSupported)
	}
	for i := range k {
		if k[i].Asset == asset.Spot ||
			!d.SupportsAsset(k[i].Asset) {
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	result := make([]futures.OpenInterest, 0, len(k))
	for i := range k {
		pFmt, err := d.CurrencyPairs.GetFormat(k[i].Asset, true)
		if err != nil {
			return nil, err
		}
		cp := k[i].Pair().Format(pFmt)
		p := d.formatPairString(k[i].Asset, cp)
		var oi []BookSummaryData
		if d.Websocket.IsConnected() {
			oi, err = d.WSRetrieveBookSummaryByInstrument(ctx, p)
		} else {
			oi, err = d.GetBookSummaryByInstrument(ctx, p)
		}
		if err != nil {
			return nil, err
		}
		for a := range oi {
			result = append(result, futures.OpenInterest{
				Key: key.ExchangePairAsset{
					Exchange: d.Name,
					Base:     k[i].Base,
					Quote:    k[i].Quote,
					Asset:    k[i].Asset,
				},
				OpenInterest: oi[a].OpenInterest,
			})
			break
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("%w, no data found for %v", currency.ErrCurrencyNotFound, k)
	}
	return result, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (d *Deribit) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if cp.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	switch a {
	case asset.Futures:
		isPerp, err := d.IsPerpetualFutureCurrency(a, cp)
		if err != nil {
			return "", err
		}
		if isPerp {
			return tradeBaseURL + tradeFutures + cp.Base.Upper().String() + currency.UnderscoreDelimiter + cp.Quote.Upper().String(), nil
		}
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	case asset.Spot:
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeBaseURL + tradeSpot + cp.Upper().String(), nil
	case asset.Options:
		baseString := cp.Base.Upper().String()
		quoteString := cp.Quote.Upper().String()
		quoteSplit := strings.Split(quoteString, currency.DashDelimiter)
		if len(quoteSplit) > 1 &&
			(quoteSplit[len(quoteSplit)-1] == "C" || quoteSplit[len(quoteSplit)-1] == "P") {
			return tradeBaseURL + tradeOptions + baseString + "/" + baseString + currency.DashDelimiter + quoteSplit[0], nil
		}
		return tradeBaseURL + tradeOptions + baseString, nil
	case asset.FutureCombo:
		return tradeBaseURL + tradeFuturesCombo + cp.Upper().String(), nil
	case asset.OptionCombo:
		return tradeBaseURL + tradeOptionsCombo + cp.Base.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
// differs by exchange
func (d *Deribit) IsPerpetualFutureCurrency(assetType asset.Item, pair currency.Pair) (bool, error) {
	if pair.IsEmpty() {
		return false, currency.ErrCurrencyPairEmpty
	}
	if assetType != asset.Futures {
		// deribit considers future combo, even if ending in "PERP" to not be a perpetual
		return false, nil
	}
	pqs := strings.Split(pair.Quote.Upper().String(), currency.DashDelimiter)
	return pqs[len(pqs)-1] == perpString, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (d *Deribit) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if !d.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%s %w", r.Asset, asset.ErrNotSupported)
	}
	isPerpetual, err := d.IsPerpetualFutureCurrency(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	if !isPerpetual {
		return nil, fmt.Errorf("%w %q", futures.ErrNotPerpetualFuture, r.Pair)
	}
	pFmt, err := d.CurrencyPairs.GetFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	cp := r.Pair.Format(pFmt)
	p := d.formatPairString(r.Asset, cp)
	var fri []FundingRateHistory
	fri, err = d.GetFundingRateHistory(ctx, p, time.Now().Add(-time.Hour*16), time.Now())
	if err != nil {
		return nil, err
	}

	resp := make([]fundingrate.LatestRateResponse, 1)
	latestTime := fri[0].Timestamp.Time()
	for i := range fri {
		if fri[i].Timestamp.Time().Before(latestTime) {
			continue
		}
		resp[0] = fundingrate.LatestRateResponse{
			TimeChecked: time.Now(),
			Exchange:    d.Name,
			Asset:       r.Asset,
			Pair:        r.Pair,
			LatestRate: fundingrate.Rate{
				Time: fri[i].Timestamp.Time(),
				Rate: decimal.NewFromFloat(fri[i].Interest8H),
			},
		}
		latestTime = fri[i].Timestamp.Time()
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	return resp, nil
}

// GetHistoricalFundingRates returns historical funding rates for a future
func (d *Deribit) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if !r.StartDate.IsZero() && !r.EndDate.IsZero() {
		err := common.StartEndTimeCheck(r.StartDate, r.EndDate)
		if err != nil {
			return nil, err
		}
	}
	if r.IncludePayments {
		return nil, fmt.Errorf("include payments %w", common.ErrNotYetImplemented)
	}
	pFmt, err := d.CurrencyPairs.GetFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	cp := r.Pair.Format(pFmt)
	p := d.formatPairString(r.Asset, cp)
	ed := r.EndDate

	var fundingRates []fundingrate.Rate
	mfr := make(map[int64]struct{})
	for ed.After(r.StartDate) {
		var records []FundingRateHistory
		if d.Websocket.IsConnected() {
			records, err = d.WSRetrieveFundingRateHistory(ctx, p, r.StartDate, ed)
		} else {
			records, err = d.GetFundingRateHistory(ctx, p, r.StartDate, ed)
		}
		if err != nil {
			return nil, err
		}
		if len(records) == 0 || ed.Equal(records[0].Timestamp.Time()) {
			break
		}
		for i := range records {
			rt := records[i].Timestamp.Time()
			if rt.Before(r.StartDate) || rt.After(r.EndDate) {
				continue
			}
			if _, ok := mfr[rt.UnixMilli()]; ok {
				continue
			}
			fundingRates = append(fundingRates, fundingrate.Rate{
				Rate: decimal.NewFromFloat(records[i].Interest1H),
				Time: rt,
			})
			mfr[rt.UnixMilli()] = struct{}{}
		}
		ed = records[0].Timestamp.Time()
	}
	if len(fundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	sort.Slice(fundingRates, func(i, j int) bool {
		return fundingRates[i].Time.Before(fundingRates[j].Time)
	})
	return &fundingrate.HistoricalRates{
		Exchange:        d.Name,
		Asset:           r.Asset,
		Pair:            r.Pair,
		FundingRates:    fundingRates,
		StartDate:       fundingRates[0].Time,
		EndDate:         r.EndDate,
		LatestRate:      fundingRates[len(fundingRates)-1],
		PaymentCurrency: r.PaymentCurrency,
	}, nil
}

func (d *Deribit) formatPairString(assetType asset.Item, pair currency.Pair) string {
	switch assetType {
	case asset.Futures:
		return d.formatFuturesTradablePair(pair)
	case asset.Options:
		return d.optionPairToString(pair)
	}
	return pair.String()
}

func timeInForceFromString(timeInForceString string, postOnly bool) (order.TimeInForce, error) {
	tif, err := order.StringToTimeInForce(timeInForceString)
	if err != nil {
		return order.UnknownTIF, err
	}
	if postOnly {
		tif |= order.PostOnly
	}
	return tif, nil
}
