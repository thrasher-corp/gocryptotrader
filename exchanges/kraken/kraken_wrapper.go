package kraken

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
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

// SetDefaults sets current default settings
func (e *Exchange) SetDefaults() {
	e.Name = "Kraken"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		}
		if a == asset.Futures {
			ps.RequestFormat.Delimiter = currency.UnderscoreDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	if err := e.DisableAssetWebsocketSupport(asset.Futures); err != nil {
		log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", e.Name, asset.Futures, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                 true,
				TickerFetching:                 true,
				KlineFetching:                  true,
				TradeFetching:                  true,
				OrderbookFetching:              true,
				AutoPairUpdates:                true,
				AccountInfo:                    true,
				GetOrder:                       true,
				GetOrders:                      true,
				CancelOrder:                    true,
				SubmitOrder:                    true,
				UserTradeHistory:               true,
				CryptoDeposit:                  true,
				CryptoWithdrawal:               true,
				FiatDeposit:                    true,
				FiatWithdraw:                   true,
				TradeFee:                       true,
				FiatDepositFee:                 true,
				FiatWithdrawalFee:              true,
				CryptoDepositFee:               true,
				CryptoWithdrawalFee:            true,
				MultiChainDeposits:             true,
				MultiChainWithdrawals:          true,
				HasAssetTypeAccountSegregation: true,
				FundingRateFetching:            true,
				PredictedFundingRate:           true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:      true,
				TradeFetching:       true,
				KlineFetching:       true,
				OrderbookFetching:   true,
				Subscribe:           true,
				Unsubscribe:         true,
				MessageCorrelation:  true,
				SubmitOrder:         true,
				CancelOrder:         true,
				CancelOrders:        true,
				GetOrders:           true,
				GetOrder:            true,
				FundingRateFetching: false, // has capability but is not supported // TODO when multi-websocket support added
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawCryptoWith2FA |
				exchange.AutoWithdrawFiatWithSetup |
				exchange.WithdrawFiatWith2FA,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.FourHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.Futures: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportsRestBatch:  true,
					SupportedViaTicker: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.FifteenDay},
				),
				GlobalResultLimit: 720,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(krakenRateInterval, krakenRequestRate, 1)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   krakenAPIURL,
		exchange.RestFutures:                krakenFuturesURL,
		exchange.WebsocketSpot:              krakenWSURL,
		exchange.WebsocketSpotSupplementary: krakenAuthWSURL,
		exchange.RestFuturesSupplementary:   krakenFuturesSupplementaryURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets current exchange configuration
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

	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            krakenWSURL,
		RunningURL:            wsRunningURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{SortBuffer: true},
	})
	if err != nil {
		return err
	}

	err = e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            request.NewWeightedRateLimitByDuration(50 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}

	wsRunningAuthURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpotSupplementary)
	if err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		RateLimit:            request.NewWeightedRateLimitByDuration(50 * time.Millisecond),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
		URL:                  wsRunningAuthURL,
	})
}

// Bootstrap provides initialisation for an exchange
func (e *Exchange) Bootstrap(ctx context.Context) (continueBootstrap bool, err error) {
	continueBootstrap = true

	if err = e.SeedAssets(ctx); err != nil {
		err = fmt.Errorf("failed to Seed Assets: %w", err)
	}

	return
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return common.ErrNotYetImplemented
	}

	if !assetTranslator.Seeded() {
		if err := e.SeedAssets(ctx); err != nil {
			return err
		}
	}

	pairInfo, err := e.fetchSpotPairInfo(ctx)
	if err != nil {
		return fmt.Errorf("%s failed to load %s pair execution limits. Err: %s", e.Name, a, err)
	}

	l := make([]limits.MinMaxLevel, 0, len(pairInfo))

	for pair, info := range pairInfo {
		l = append(l, limits.MinMaxLevel{
			Key:                    key.NewExchangeAssetPair(e.Name, a, pair),
			PriceStepIncrementSize: info.TickSize,
			MinimumBaseAmount:      info.OrderMinimum,
		})
	}

	if err := limits.Load(l); err != nil {
		return fmt.Errorf("%s Error loading %s exchange limits: %w", e.Name, a, err)
	}

	return nil
}

func (e *Exchange) fetchSpotPairInfo(ctx context.Context) (map[currency.Pair]*AssetPairs, error) {
	pairs := make(map[currency.Pair]*AssetPairs)

	pairInfo, err := e.GetAssetPairs(ctx, nil, "")
	if err != nil {
		return pairs, err
	}

	for _, info := range pairInfo {
		if info.Status != "online" {
			continue
		}
		base := assetTranslator.LookupAltName(info.Base)
		if base == "" {
			log.Warnf(log.ExchangeSys,
				"%s unable to lookup altname for base currency %s",
				e.Name,
				info.Base)
			continue
		}
		quote := assetTranslator.LookupAltName(info.Quote)
		if quote == "" {
			log.Warnf(log.ExchangeSys,
				"%s unable to lookup altname for quote currency %s",
				e.Name,
				info.Quote)
			continue
		}
		pair, err := currency.NewPairFromStrings(base, quote)
		if err != nil {
			return pairs, err
		}
		pairs[pair] = info
	}

	return pairs, nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	pairs := currency.Pairs{}
	switch a {
	case asset.Spot:
		if !assetTranslator.Seeded() {
			if err := e.SeedAssets(ctx); err != nil {
				return nil, err
			}
		}
		pairInfo, err := e.fetchSpotPairInfo(ctx)
		if err != nil {
			return pairs, err
		}
		pairs = make(currency.Pairs, 0, len(pairInfo))
		for pair := range pairInfo {
			pairs = append(pairs, pair)
		}
	case asset.Futures:
		symbols, err := e.GetInstruments(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make([]currency.Pair, 0, len(symbols.Instruments))
		for x := range symbols.Instruments {
			if !symbols.Instruments[x].Tradable {
				continue
			}
			pair, err := currency.NewPairFromString(symbols.Instruments[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(false)
	for x := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assets[x], false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot:
		tickers, err := e.GetTickers(ctx, "")
		if err != nil {
			return err
		}
		for c, t := range tickers {
			var cp currency.Pair
			cp, err = e.MatchSymbolWithAvailablePairs(c, a, false)
			if err != nil {
				if !errors.Is(err, currency.ErrPairNotFound) {
					return err
				}
				altName := assetTranslator.LookupAltName(c)
				if altName == "" {
					continue
				}
				cp, err = e.MatchSymbolWithAvailablePairs(altName, a, false)
				if err != nil {
					continue
				}
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         t.Last,
				High:         t.High,
				Low:          t.Low,
				Bid:          t.Bid,
				BidSize:      t.BidSize,
				Ask:          t.Ask,
				AskSize:      t.AskSize,
				Volume:       t.Volume,
				Open:         t.Open,
				Pair:         cp,
				ExchangeName: e.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		t, err := e.GetFuturesTickers(ctx)
		if err != nil {
			return err
		}
		for x := range t.Tickers {
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         t.Tickers[x].Last,
				Bid:          t.Tickers[x].Bid,
				BidSize:      t.Tickers[x].BidSize,
				Ask:          t.Tickers[x].Ask,
				AskSize:      t.Tickers[x].AskSize,
				Volume:       t.Tickers[x].Vol24h,
				Open:         t.Tickers[x].Open24H,
				OpenInterest: t.Tickers[x].OpenInterest,
				MarkPrice:    t.Tickers[x].MarkPrice,
				IndexPrice:   t.Tickers[x].IndexPrice,
				Pair:         t.Tickers[x].Symbol,
				ExchangeName: e.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	switch assetType {
	case asset.Spot:
		orderbookNew, err := e.GetDepth(ctx, p)
		if err != nil {
			return book, err
		}
		book.Bids = make([]orderbook.Level, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			book.Bids[x] = orderbook.Level{
				Amount: orderbookNew.Bids[x].Amount.Float64(),
				Price:  orderbookNew.Bids[x].Price.Float64(),
			}
		}
		book.Asks = make([]orderbook.Level, len(orderbookNew.Asks))
		for y := range orderbookNew.Asks {
			book.Asks[y] = orderbook.Level{
				Amount: orderbookNew.Asks[y].Amount.Float64(),
				Price:  orderbookNew.Asks[y].Price.Float64(),
			}
		}
	case asset.Futures:
		futuresOB, err := e.GetFuturesOrderbook(ctx, p)
		if err != nil {
			return book, err
		}
		book.Asks = make([]orderbook.Level, len(futuresOB.Orderbook.Asks))
		for x := range futuresOB.Orderbook.Asks {
			book.Asks[x] = orderbook.Level{
				Price:  futuresOB.Orderbook.Asks[x][0],
				Amount: futuresOB.Orderbook.Asks[x][1],
			}
		}
		book.Bids = make([]orderbook.Level, len(futuresOB.Orderbook.Bids))
		for y := range futuresOB.Orderbook.Bids {
			book.Bids[y] = orderbook.Level{
				Price:  futuresOB.Orderbook.Bids[y][0],
				Amount: futuresOB.Orderbook.Bids[y][1],
			}
		}
		book.Bids.SortBids()
	default:
		return book, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if err := book.Process(); err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (subAccts accounts.SubAccounts, err error) {
	if !assetTranslator.Seeded() {
		if err := e.SeedAssets(ctx); err != nil {
			return nil, err
		}
	}
	switch assetType {
	case asset.Spot:
		resp, err := e.GetBalance(ctx)
		if err != nil {
			return nil, err
		}
		subAccts = accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
		for key, bal := range resp {
			c := assetTranslator.LookupAltName(key)
			if c == "" {
				log.Warnf(log.ExchangeSys, "%s unable to translate currency: %s", e.Name, key)
				continue
			}
			subAccts[0].Balances.Set(currency.NewCode(c), accounts.Balance{
				Total: bal.Total,
				Hold:  bal.Hold,
				Free:  bal.Total - bal.Hold,
			})
		}
	case asset.Futures:
		resp, err := e.GetFuturesAccountData(ctx)
		if err != nil {
			return nil, err
		}
		for name, v := range resp.Accounts {
			a := accounts.NewSubAccount(assetType, name)
			for curr, bal := range v.Balances {
				a.Balances.Set(currency.NewCode(curr), accounts.Balance{Total: bal})
			}
			subAccts = subAccts.Merge(a)
		}
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.WithdrawStatus(ctx, c, "")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawals[i].Status,
			TransferID:      withdrawals[i].Refid,
			Timestamp:       withdrawals[i].Time.Time(),
			Amount:          withdrawals[i].Amount,
			Fee:             withdrawals[i].Fee,
			CryptoToAddress: withdrawals[i].Info,
			CryptoTxID:      withdrawals[i].TxID,
			Currency:        c.String(),
		}
	}

	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		tradeData, err := e.GetTrades(ctx, p, time.Time{}, 1000)
		if err != nil {
			return nil, err
		}
		trades, ok := tradeData.Trades[assetTranslator.LookupCurrency(p.String())]
		if !ok {
			return nil, fmt.Errorf("unable to find symbol %s in trade data", p.String())
		}
		for i := range trades {
			side := order.Buy
			if trades[i].BuyOrSell == "s" {
				side = order.Sell
			}
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(trades[i].TradeID.Int64(), 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Volume.Float64(),
				Timestamp:    trades[i].Time.Time(),
			})
		}
	case asset.Futures:
		var tradeData *FuturesPublicTrades
		tradeData, err = e.GetFuturesTrades(ctx, p, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		for i := range tradeData.Elements {
			side := order.Buy
			if strings.EqualFold(tradeData.Elements[i].ExecutionEvent.OuterExecutionHolder.Execution.MakerOrder.Direction, "sell") {
				side = order.Sell
			}
			resp = append(resp, trade.Data{
				TID:          tradeData.Elements[i].UID,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData.Elements[i].ExecutionEvent.OuterExecutionHolder.Execution.MakerOrder.LimitPrice,
				Amount:       tradeData.Elements[i].ExecutionEvent.OuterExecutionHolder.Execution.MakerOrder.Quantity,
				Timestamp:    tradeData.Elements[i].ExecutionEvent.OuterExecutionHolder.Execution.MakerOrder.Timestamp.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(e.GetTradingRequirements())
	if err != nil {
		return nil, err
	}

	var orderID string
	status := order.New
	switch s.AssetType {
	case asset.Spot:
		var timeInForce string
		switch {
		case s.TimeInForce.Is(order.GoodTillDay):
			timeInForce = "GTD"
		case s.TimeInForce.Is(order.ImmediateOrCancel):
			timeInForce = "IOC"
		}
		if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			orderID, err = e.wsAddOrder(ctx, &WsAddOrderRequest{
				OrderType:   s.Type.Lower(),
				OrderSide:   s.Side.Lower(),
				Pair:        s.Pair.Format(currency.PairFormat{Uppercase: true, Delimiter: "/"}).String(), // required pair format: ISO 4217-A3
				Price:       s.Price,
				Volume:      s.Amount,
				TimeInForce: timeInForce,
			})
			if err != nil {
				return nil, err
			}
		} else {
			var response *AddOrderResponse
			response, err = e.AddOrder(ctx,
				s.Pair,
				s.Side.String(),
				s.Type.String(),
				s.Amount,
				s.Price,
				0,
				0,
				&AddOrderOptions{
					TimeInForce: timeInForce,
				})
			if err != nil {
				return nil, err
			}
			if len(response.TransactionIDs) > 0 {
				orderID = strings.Join(response.TransactionIDs, ", ")
			}
		}
		if s.Type == order.Market {
			status = order.Filled
		}
	case asset.Futures:
		var fOrder FuturesSendOrderData
		fOrder, err = e.FuturesSendOrder(ctx,
			s.Type,
			s.Pair,
			s.Side.Lower(),
			"",
			s.ClientOrderID,
			"",
			s.TimeInForce,
			s.Amount,
			s.Price,
			0,
		)
		if err != nil {
			return nil, err
		}

		// check the status, anything that is not placed we error out
		if fOrder.SendStatus.Status != "placed" {
			return nil, fmt.Errorf("submit order failed: %s", fOrder.SendStatus.Status)
		}
		orderID = fOrder.SendStatus.OrderID
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, s.AssetType)
	}
	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = status
	return resp, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot:
		if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			return e.wsCancelOrders(ctx, []string{o.OrderID})
		}
		_, err := e.CancelExistingOrder(ctx, o.OrderID)
		return err
	case asset.Futures:
		_, err := e.FuturesCancelOrder(ctx, o.OrderID, "")
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, o.AssetType)
	}

	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if !e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		return nil, common.ErrFunctionNotSupported
	}

	ordersList := make([]string, len(o))
	for i := range o {
		if err := o[i].Validate(o[i].StandardCancel()); err != nil {
			return nil, err
		}
		ordersList[i] = o[i].OrderID
	}

	err := e.wsCancelOrders(ctx, ordersList)
	return nil, err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	if err := req.Validate(); err != nil {
		return resp, err
	}
	switch req.AssetType {
	case asset.Spot:
		if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			cancel, err := e.wsCancelAllOrders(ctx)
			if err != nil {
				return resp, err
			}
			for i := range cancel.Count {
				resp.Add(fmt.Sprintf("Unknown:%d", i+1), "cancelled")
			}
			return resp, err
		}
		openOrders, err := e.GetOpenOrders(ctx, OrderInfoOptions{})
		if err != nil {
			return resp, err
		}
		for orderID := range openOrders.Open {
			if e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				err = e.wsCancelOrders(ctx, []string{orderID})
			} else {
				_, err = e.CancelExistingOrder(ctx, orderID)
			}
			if err != nil {
				resp.Add(orderID, err.Error())
				continue
			}
			resp.Add(orderID, "cancelled")
		}
	case asset.Futures:
		cancelData, err := e.FuturesCancelAllOrders(ctx, req.Pair)
		if err != nil {
			return resp, err
		}
		for x := range cancelData.CancelStatus.CancelledOrders {
			resp.Add(cancelData.CancelStatus.CancelledOrders[x].OrderID, "cancelled")
		}
	}
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	var orderDetail order.Detail
	switch assetType {
	case asset.Spot:
		resp, err := e.QueryOrdersInfo(ctx,
			OrderInfoOptions{
				Trades: true,
			}, orderID)
		if err != nil {
			return nil, err
		}

		orderInfo, ok := resp[orderID]
		if !ok {
			return nil, fmt.Errorf("order %s not found in response", orderID)
		}

		if !assetType.IsValid() {
			assetType = asset.UseDefault()
		}

		avail, err := e.GetAvailablePairs(assetType)
		if err != nil {
			return nil, err
		}

		format, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}

		var trades []order.TradeHistory
		for i := range orderInfo.Trades {
			trades = append(trades, order.TradeHistory{
				TID: orderInfo.Trades[i],
			})
		}
		side, err := order.StringToOrderSide(orderInfo.Description.Type)
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(orderInfo.Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		oType, err := order.StringToOrderType(orderInfo.Description.OrderType)
		if err != nil {
			return nil, err
		}

		p, err := currency.NewPairFromFormattedPairs(orderInfo.Description.Pair,
			avail,
			format)
		if err != nil {
			return nil, err
		}

		price := orderInfo.Price
		if orderInfo.Status == statusOpen {
			price = orderInfo.Description.Price
		}

		orderDetail = order.Detail{
			Exchange:        e.Name,
			OrderID:         orderID,
			Pair:            p,
			Side:            side,
			Type:            oType,
			Date:            orderInfo.OpenTime.Time(),
			CloseTime:       orderInfo.CloseTime.Time(),
			Status:          status,
			Price:           price,
			Amount:          orderInfo.Volume,
			ExecutedAmount:  orderInfo.VolumeExecuted,
			RemainingAmount: orderInfo.Volume - orderInfo.VolumeExecuted,
			Fee:             orderInfo.Fee,
			Trades:          trades,
			Cost:            orderInfo.Cost,
			AssetType:       asset.Spot,
		}
	case asset.Futures:
		orderInfo, err := e.FuturesGetFills(ctx, time.Time{})
		if err != nil {
			return nil, err
		}
		for y := range orderInfo.Fills {
			if orderInfo.Fills[y].OrderID != orderID {
				continue
			}
			pair, err := currency.NewPairFromString(orderInfo.Fills[y].Symbol)
			if err != nil {
				return nil, err
			}
			oSide, err := compatibleOrderSide(orderInfo.Fills[y].Side)
			if err != nil {
				return nil, err
			}
			fillOrderType, err := compatibleFillOrderType(orderInfo.Fills[y].FillType)
			if err != nil {
				return nil, err
			}
			orderDetail = order.Detail{
				OrderID:   orderID,
				Price:     orderInfo.Fills[y].Price,
				Amount:    orderInfo.Fills[y].Size,
				Side:      oSide,
				Type:      fillOrderType,
				Date:      orderInfo.Fills[y].FillTime,
				Pair:      pair,
				Exchange:  e.Name,
				AssetType: asset.Futures,
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return &orderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	if chain == "" {
		methods, err := e.GetDepositMethods(ctx, cryptocurrency.String())
		if err != nil {
			return nil, err
		}
		if len(methods) == 0 {
			return nil, errors.New("unable to get any deposit methods")
		}
		chain = methods[0].Method
	}

	depositAddr, err := e.GetCryptoDepositAddress(ctx, chain, cryptocurrency.String(), false)
	if err != nil {
		if strings.Contains(err.Error(), "no addresses returned") {
			depositAddr, err = e.GetCryptoDepositAddress(ctx, chain, cryptocurrency.String(), true)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &deposit.Address{
		Address: depositAddr[0].Address,
		Tag:     depositAddr[0].Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal
// Populate exchange.WithdrawRequest.TradePassword with withdrawal key name, as set up on your account
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := e.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.TradePassword,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := e.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.TradePassword,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := e.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.TradePassword,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		resp, err := e.GetOpenOrders(ctx, OrderInfoOptions{})
		if err != nil {
			return nil, err
		}

		assetType := req.AssetType
		if !req.AssetType.IsValid() {
			assetType = asset.UseDefault()
		}

		avail, err := e.GetAvailablePairs(assetType)
		if err != nil {
			return nil, err
		}

		format, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		for i := range resp.Open {
			p, err := currency.NewPairFromFormattedPairs(resp.Open[i].Description.Pair,
				avail,
				format)
			if err != nil {
				return nil, err
			}
			var side order.Side
			side, err = order.StringToOrderSide(resp.Open[i].Description.Type)
			if err != nil {
				return nil, err
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(resp.Open[i].Description.OrderType)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}
			orders = append(orders, order.Detail{
				OrderID:         i,
				Amount:          resp.Open[i].Volume,
				RemainingAmount: resp.Open[i].Volume - resp.Open[i].VolumeExecuted,
				ExecutedAmount:  resp.Open[i].VolumeExecuted,
				Exchange:        e.Name,
				Date:            resp.Open[i].OpenTime.Time(),
				Price:           resp.Open[i].Description.Price,
				Side:            side,
				Type:            orderType,
				Pair:            p,
				AssetType:       asset.Spot,
				Status:          order.Open,
			})
		}
	case asset.Futures:
		var err error
		var pairs currency.Pairs
		if len(req.Pairs) > 0 {
			pairs = req.Pairs
		} else {
			pairs, err = e.GetEnabledPairs(asset.Futures)
			if err != nil {
				return orders, err
			}
		}
		activeOrders, err := e.FuturesOpenOrders(ctx)
		if err != nil {
			return orders, err
		}
		for i := range pairs {
			fPair, err := e.FormatExchangeCurrency(pairs[i], asset.Futures)
			if err != nil {
				return orders, err
			}
			for a := range activeOrders.OpenOrders {
				if activeOrders.OpenOrders[a].Symbol != fPair.String() {
					continue
				}
				oSide, err := compatibleOrderSide(activeOrders.OpenOrders[a].Side)
				if err != nil {
					return orders, err
				}
				oType, err := compatibleOrderType(activeOrders.OpenOrders[a].OrderType)
				if err != nil {
					return orders, err
				}
				orders = append(orders, order.Detail{
					OrderID:   activeOrders.OpenOrders[a].OrderID,
					Price:     activeOrders.OpenOrders[a].LimitPrice,
					Amount:    activeOrders.OpenOrders[a].FilledSize,
					Side:      oSide,
					Type:      oType,
					Date:      activeOrders.OpenOrders[a].ReceivedTime,
					Pair:      fPair,
					Exchange:  e.Name,
					AssetType: asset.Futures,
					Status:    order.Open,
				})
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		req := GetClosedOrdersOptions{}
		if getOrdersRequest.StartTime.Unix() > 0 {
			req.Start = strconv.FormatInt(getOrdersRequest.StartTime.Unix(), 10)
		}
		if getOrdersRequest.EndTime.Unix() > 0 {
			req.End = strconv.FormatInt(getOrdersRequest.EndTime.Unix(), 10)
		}

		assetType := getOrdersRequest.AssetType
		if !getOrdersRequest.AssetType.IsValid() {
			assetType = asset.UseDefault()
		}

		avail, err := e.GetAvailablePairs(assetType)
		if err != nil {
			return nil, err
		}

		format, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}

		resp, err := e.GetClosedOrders(ctx, req)
		if err != nil {
			return nil, err
		}

		for i := range resp.Closed {
			p, err := currency.NewPairFromFormattedPairs(resp.Closed[i].Description.Pair,
				avail,
				format)
			if err != nil {
				return nil, err
			}

			var side order.Side
			side, err = order.StringToOrderSide(resp.Closed[i].Description.Type)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}
			status, err := order.StringToOrderStatus(resp.Closed[i].Status)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(resp.Closed[i].Description.OrderType)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}
			detail := order.Detail{
				OrderID:         i,
				Amount:          resp.Closed[i].Volume,
				ExecutedAmount:  resp.Closed[i].VolumeExecuted,
				RemainingAmount: resp.Closed[i].Volume - resp.Closed[i].VolumeExecuted,
				Cost:            resp.Closed[i].Cost,
				CostAsset:       p.Quote,
				Exchange:        e.Name,
				Date:            resp.Closed[i].OpenTime.Time(),
				CloseTime:       resp.Closed[i].CloseTime.Time(),
				Price:           resp.Closed[i].Description.Price,
				Side:            side,
				Status:          status,
				Type:            orderType,
				Pair:            p,
			}
			detail.InferCostsAndTimes()
			orders = append(orders, detail)
		}
	case asset.Futures:
		var orderHistory FuturesRecentOrdersData
		var err error
		var pairs currency.Pairs
		if len(getOrdersRequest.Pairs) > 0 {
			pairs = getOrdersRequest.Pairs
		} else {
			pairs, err = e.GetEnabledPairs(asset.Futures)
			if err != nil {
				return orders, err
			}
		}
		for p := range pairs {
			orderHistory, err = e.FuturesRecentOrders(ctx, pairs[p])
			if err != nil {
				return orders, err
			}
			for o := range orderHistory.OrderEvents {
				switch {
				case orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.UID != "":
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Quantity -
							orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Filled,
						OrderID:   orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.ClientID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Timestamp,
						Side:      oDirection,
						Exchange:  e.Name,
						Pair:      pairs[p],
					})
				case orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.UID != "":
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Quantity -
							orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Filled,
						OrderID:   orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.AccountID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Timestamp,
						Side:      oDirection,
						Exchange:  e.Name,
						Pair:      pairs[p],
						Status:    order.Rejected,
					})
				case orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.UID != "":
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Quantity -
							orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Filled,
						OrderID:   orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.AccountID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Timestamp,
						Side:      oDirection,
						Exchange:  e.Name,
						Pair:      pairs[p],
						Status:    order.Cancelled,
					})
				case orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.UID != "":
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Quantity -
							orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Filled,
						OrderID:   orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.AccountID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Timestamp,
						Side:      oDirection,
						Exchange:  e.Name,
						Pair:      pairs[p],
					})
				default:
					return orders, errors.New("invalid orderHistory data")
				}
			}
		}
	}
	return getOrdersRequest.Filter(e.Name, orders), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	resp, err := e.GetWebsocketToken(ctx)
	if err != nil {
		return err
	}

	e.setWebsocketAuthToken(resp)
	return nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Minutes(), 'f', -1, 64)
}

// FormatExchangeKlineIntervalFutures returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineIntervalFutures(in kline.Interval) string {
	switch in {
	case kline.OneDay:
		return "1d"
	default:
		return in.Short()
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	switch a {
	case asset.Spot:
		candles, err := e.GetOHLC(ctx,
			req.RequestFormatted,
			e.FormatExchangeKlineInterval(req.ExchangeInterval))
		if err != nil {
			return nil, err
		}

		for x := range candles {
			if candles[x].Time.Before(req.Start) || candles[x].Time.After(req.End) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[x].Time,
				Open:   candles[x].Open,
				High:   candles[x].High,
				Low:    candles[x].Low,
				Close:  candles[x].Close,
				Volume: candles[x].Volume,
			})
		}
	default:
		// TODO add new Futures API support
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}

	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

func compatibleOrderSide(side string) (order.Side, error) {
	switch {
	case strings.EqualFold(order.Buy.String(), side):
		return order.Buy, nil
	case strings.EqualFold(order.Sell.String(), side):
		return order.Sell, nil
	}
	return order.AnySide, errors.New("invalid side received")
}

func compatibleOrderType(orderType string) (order.Type, error) {
	var resp order.Type
	switch orderType {
	case "lmt":
		resp = order.Limit
	case "stp":
		resp = order.Stop
	case "take_profit":
		resp = order.TakeProfit
	default:
		return resp, errors.New("invalid orderType")
	}
	return resp, nil
}

func compatibleFillOrderType(fillType string) (order.Type, error) {
	var resp order.Type
	switch fillType {
	case "maker":
		resp = order.Limit
	case "taker":
		resp = order.Market
	case "liquidation":
		resp = order.Liquidation
	default:
		return resp, errors.New("invalid orderPriceType")
	}
	return resp, nil
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	methods, err := e.GetDepositMethods(ctx, cryptocurrency.String())
	if err != nil {
		return nil, err
	}

	availableChains := make([]string, len(methods))
	for x := range methods {
		availableChains[x] = methods[x].Method
	}
	return availableChains, nil
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	st, err := e.GetCurrentServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse("Mon, 02 Jan 06 15:04:05 -0700", st.Rfc1123)
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	result, err := e.GetInstruments(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(result.Instruments))
	for i := range result.Instruments {
		var cp, underlying currency.Pair
		var underlyingStr string
		cp, err = currency.NewPairFromString(result.Instruments[i].Symbol)
		if err != nil {
			return nil, err
		}
		var symbolToSplit string
		if result.Instruments[i].Underlying != "" {
			symbolToSplit = result.Instruments[i].Underlying
		} else {
			symbolToSplit = result.Instruments[i].Symbol
		}

		underlyingBase := strings.Split(symbolToSplit, "_")
		if len(underlyingBase) <= 1 {
			underlyingStr = symbolToSplit
		} else {
			underlyingStr = underlyingBase[1]
		}
		usdIndex := strings.LastIndex(strings.ToLower(underlyingStr), "usd")
		if usdIndex <= 0 {
			log.Warnf(log.ExchangeSys, "%v unable to find USD index in %v to process contract", e.Name, underlyingStr)
			continue
		}
		underlying, err = currency.NewPairFromStrings(underlyingStr[0:usdIndex], underlyingStr[usdIndex:])
		if err != nil {
			return nil, err
		}
		var startTime, endTime time.Time
		if !result.Instruments[i].OpeningDate.IsZero() {
			startTime = result.Instruments[i].OpeningDate
		}
		var ct futures.ContractType
		if result.Instruments[i].LastTradingTime.IsZero() || item == asset.PerpetualSwap {
			ct = futures.Perpetual
		} else {
			endTime = result.Instruments[i].LastTradingTime
			switch {
			// three day is used for generosity for contract date ranges
			case endTime.Sub(startTime) <= kline.OneMonth.Duration()+kline.ThreeDay.Duration():
				ct = futures.Monthly
			case endTime.Sub(startTime) <= kline.ThreeMonth.Duration()+kline.ThreeDay.Duration():
				ct = futures.Quarterly
			default:
				ct = futures.SemiAnnually
			}
		}
		contractSettlementType := futures.Linear
		if cp.Base.Equal(currency.PI) || cp.Base.Equal(currency.FI) {
			contractSettlementType = futures.Inverse
		}
		resp[i] = futures.Contract{
			Exchange:       e.Name,
			Name:           cp,
			Underlying:     underlying,
			Asset:          item,
			StartDate:      startTime,
			EndDate:        endTime,
			SettlementType: contractSettlementType,
			IsActive:       result.Instruments[i].Tradable,
			Type:           ct,
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	if !r.Pair.IsEmpty() {
		if ok, err := e.CurrencyPairs.IsPairAvailable(r.Pair, r.Asset); err != nil {
			return nil, err
		} else if !ok {
			return nil, currency.ErrPairNotContainedInAvailablePairs
		}
	}

	t, err := e.GetFuturesTickers(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]fundingrate.LatestRateResponse, 0, len(t.Tickers))
	for i := range t.Tickers {
		if !r.Pair.IsEmpty() && !r.Pair.Equal(t.Tickers[i].Symbol) {
			continue
		}
		var isPerp bool
		isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, t.Tickers[i].Symbol)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			continue
		}
		rate := fundingrate.LatestRateResponse{
			Exchange: e.Name,
			Asset:    r.Asset,
			Pair:     t.Tickers[i].Symbol,
			LatestRate: fundingrate.Rate{
				Rate: decimal.NewFromFloat(t.Tickers[i].FundingRate),
			},
			TimeChecked: time.Now(),
		}
		if r.IncludePredictedRate {
			rate.PredictedUpcomingRate = fundingrate.Rate{
				Rate: decimal.NewFromFloat(t.Tickers[i].FundingRatePrediction),
			}
		}
		resp = append(resp, rate)
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	return cp.Base.Equal(currency.PF) && a == asset.Futures, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range keys {
		if keys[i].Asset != asset.Futures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, keys[i].Asset, keys[i].Pair())
		}
	}
	futuresTickersData, err := e.GetFuturesTickers(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(futuresTickersData.Tickers))
	for i := range futuresTickersData.Tickers {
		var p currency.Pair
		var isEnabled bool
		p, isEnabled, err = e.MatchSymbolCheckEnabled(futuresTickersData.Tickers[i].Symbol.String(), asset.Futures, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		var appendData bool
		for j := range keys {
			if keys[j].Pair().Equal(p) {
				appendData = true
				break
			}
		}
		if len(keys) > 0 && !appendData {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.Futures, p),
			OpenInterest: futuresTickersData.Tickers[i].OpenInterest,
		})
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		cp.Delimiter = currency.DashDelimiter
		return tradeBaseURL + cp.Lower().String(), nil
	case asset.Futures:
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeFuturesURL + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
