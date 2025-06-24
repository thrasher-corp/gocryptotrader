package okx

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	websocketResponseMaxLimit = time.Second * 3
)

// SetDefaults sets the basic defaults for Okx
func (ok *Okx) SetDefaults() {
	ok.Name = "Okx"
	ok.Enabled = true
	ok.Verbose = true

	ok.API.CredentialsValidator.RequiresKey = true
	ok.API.CredentialsValidator.RequiresSecret = true
	ok.API.CredentialsValidator.RequiresClientID = true

	ok.instrumentsInfoMap = make(map[string][]Instrument)

	cpf := &currency.PairFormat{
		Delimiter: currency.DashDelimiter,
		Uppercase: true,
	}

	// In this exchange, we represent deliverable futures contracts as 'FUTURES'/asset.Futures and perpetual futures as 'SWAP'/asset.PerpetualSwap
	err := ok.SetGlobalPairsManager(cpf, cpf, asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options, asset.Margin, asset.Spread)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// TODO: Disabled until spread/business websocket is implemented
	if err := ok.DisableAssetWebsocketSupport(asset.Spread); err != nil {
		log.Errorf(log.ExchangeSys, "%s error disabling %q asset websocket support: %s", ok.Name, asset.Spread.String(), err)
	}

	// Fill out the capabilities/features that the exchange supports
	ok.Features = exchange.Features{
		CurrencyTranslations: currency.NewTranslations(map[currency.Code]currency.Code{
			currency.NewCode("USDT-SWAP"): currency.USDT,
			currency.NewCode("USD-SWAP"):  currency.USD,
			currency.NewCode("USDC-SWAP"): currency.USDC,
		}),
		Supports: exchange.FeaturesSupported{
			REST:                true,
			Websocket:           true,
			MaximumOrderHistory: kline.OneDay.Duration() * 90,
			RESTCapabilities: protocol.Features{
				TickerFetching:        true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				CryptoDeposit:         true,
				CryptoWithdrawalFee:   true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				SubmitOrder:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				CancelOrders:          true,
				TradeFetching:         true,
				UserTradeHistory:      true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
				KlineFetching:         true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				ModifyOrder:           true,
				FundingRateFetching:   true,
				PredictedFundingRate:  true,
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
				SubmitOrder:            true,
				CancelOrder:            true,
				CancelOrders:           true,
				ModifyOrder:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:      true,
				Leverage:       true,
				CollateralMode: true,
				OpenInterest: exchange.OpenInterestSupport{
					Supported:         true,
					SupportsRestBatch: true,
				},
				FundingRates:              true,
				MaximumFundingRateHistory: kline.ThreeMonth.Duration(),
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.TwoDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.FiveDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
					kline.IntervalCapacity{Interval: kline.ThreeMonth},
					kline.IntervalCapacity{Interval: kline.SixMonth},
					kline.IntervalCapacity{Interval: kline.OneYear},
				),
				GlobalResultLimit: 100, // Reference: https://www.okx.com/docs-v5/en/#rest-api-market-data-get-candlesticks-history
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}
	ok.Requester, err = request.New(ok.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(rateLimits))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	ok.API.Endpoints = ok.NewEndpoints()
	err = ok.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      apiURL,
		exchange.WebsocketSpot: apiWebsocketPublicURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	ok.Websocket = websocket.NewManager()
	ok.WebsocketResponseMaxLimit = websocketResponseMaxLimit
	ok.WebsocketResponseCheckTimeout = websocketResponseMaxLimit
	ok.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (ok *Okx) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		ok.SetEnabled(false)
		return nil
	}
	if err := ok.SetupDefaults(exch); err != nil {
		return err
	}

	wsRunningEndpoint, err := ok.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	if err := ok.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		DefaultURL:                             apiWebsocketPublicURL,
		RunningURL:                             wsRunningEndpoint,
		Connector:                              ok.WsConnect,
		Subscriber:                             ok.Subscribe,
		Unsubscriber:                           ok.Unsubscribe,
		GenerateSubscriptions:                  ok.generateSubscriptions,
		Features:                               &ok.Features.Supports.WebsocketCapabilities,
		MaxWebsocketSubscriptionsPerConnection: 240,
		RateLimitDefinitions:                   rateLimits,
	}); err != nil {
		return err
	}

	if err := ok.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                      apiWebsocketPublicURL,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         websocketResponseMaxLimit,
		RateLimit:                request.NewRateLimitWithWeight(time.Second, 2, 1),
		BespokeGenerateMessageID: func(bool) int64 { return ok.messageIDSeq.IncrementAndGet() },
	}); err != nil {
		return err
	}

	return ok.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                      apiWebsocketPrivateURL,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         websocketResponseMaxLimit,
		Authenticated:            true,
		RateLimit:                request.NewRateLimitWithWeight(time.Second, 2, 1),
		BespokeGenerateMessageID: func(bool) int64 { return ok.messageIDSeq.IncrementAndGet() },
	})
}

// GetServerTime returns the current exchange server time.
func (ok *Okx) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	t, err := ok.GetSystemTime(ctx)
	return t.Time(), err
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (ok *Okx) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Options, asset.Futures, asset.Spot, asset.PerpetualSwap, asset.Margin:
		format, err := ok.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		insts, err := ok.getInstrumentsForAsset(ctx, a)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(insts))
		for x := range insts {
			if insts[x].State != "live" {
				continue
			}
			pairs = append(pairs, insts[x].InstrumentID.Format(format))
		}
		return pairs, nil
	case asset.Spread:
		format, err := ok.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		spreadInstruments, err := ok.GetPublicSpreads(ctx, "", "", "", "live")
		if err != nil {
			return nil, fmt.Errorf("%w asset type: %v", err, a)
		}
		pairs := make(currency.Pairs, len(spreadInstruments))
		for x := range spreadInstruments {
			pairs[x] = spreadInstruments[x].SpreadID.Format(format)
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (ok *Okx) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := ok.GetAssetTypes(true)
	for i := range assetTypes {
		pairs, err := ok.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return fmt.Errorf("%w for asset %v", err, assetTypes[i])
		}
		err = ok.UpdatePairs(pairs, assetTypes[i], false, forceUpdate)
		if err != nil {
			return fmt.Errorf("%w for asset %v", err, assetTypes[i])
		}
	}
	return ok.EnsureOnePairEnabled()
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (ok *Okx) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot, asset.Margin, asset.Options,
		asset.PerpetualSwap, asset.Futures:
		insts, err := ok.getInstrumentsForAsset(ctx, a)
		if err != nil {
			return err
		}
		if len(insts) == 0 {
			return common.ErrNoResponse
		}
		limits := make([]order.MinMaxLevel, len(insts))
		for x := range insts {
			limits[x] = order.MinMaxLevel{
				Pair:                   insts[x].InstrumentID,
				Asset:                  a,
				PriceStepIncrementSize: insts[x].TickSize.Float64(),
				MinimumBaseAmount:      insts[x].MinimumOrderSize.Float64(),
			}
		}
		return ok.LoadLimits(limits)
	case asset.Spread:
		insts, err := ok.GetPublicSpreads(ctx, "", "", "", "live")
		if err != nil {
			return err
		}
		if len(insts) == 0 {
			return common.ErrNoResponse
		}
		limits := make([]order.MinMaxLevel, len(insts))
		for x := range insts {
			limits[x] = order.MinMaxLevel{
				Pair:                   insts[x].SpreadID,
				Asset:                  a,
				PriceStepIncrementSize: insts[x].MinSize.Float64(),
				MinimumBaseAmount:      insts[x].MinSize.Float64(),
				QuoteStepIncrementSize: insts[x].TickSize.Float64(),
			}
		}
		return ok.LoadLimits(limits)
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (ok *Okx) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	p, err := ok.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	if a == asset.Spread {
		spreadTicker, err := ok.GetPublicSpreadTickers(ctx, p.String())
		if err != nil {
			return nil, err
		}

		if len(spreadTicker) == 0 {
			return nil, fmt.Errorf("no ticker data for %s", p.String())
		}

		if err := ticker.ProcessTicker(&ticker.Price{
			Last:         spreadTicker[0].Last.Float64(),
			High:         spreadTicker[0].High24Hour.Float64(),
			Low:          spreadTicker[0].Low24Hour.Float64(),
			Bid:          spreadTicker[0].BidPrice.Float64(),
			BidSize:      spreadTicker[0].BidSize.Float64(),
			Ask:          spreadTicker[0].AskPrice.Float64(),
			AskSize:      spreadTicker[0].AskSize.Float64(),
			Volume:       spreadTicker[0].Volume24Hour.Float64(),
			Open:         spreadTicker[0].Open24Hour.Float64(),
			LastUpdated:  spreadTicker[0].Timestamp.Time(),
			Pair:         p,
			AssetType:    a,
			ExchangeName: ok.Name,
		}); err != nil {
			return nil, err
		}
	} else {
		mdata, err := ok.GetTicker(ctx, p.String())
		if err != nil {
			return nil, err
		}
		var baseVolume, quoteVolume float64
		switch a {
		case asset.Spot, asset.Margin:
			baseVolume = mdata.Vol24H.Float64()
			quoteVolume = mdata.VolCcy24H.Float64()
		case asset.PerpetualSwap, asset.Futures, asset.Options:
			baseVolume = mdata.VolCcy24H.Float64()
			quoteVolume = mdata.Vol24H.Float64()
		}
		if err := ticker.ProcessTicker(&ticker.Price{
			Last:         mdata.LastTradePrice.Float64(),
			High:         mdata.High24H.Float64(),
			Low:          mdata.Low24H.Float64(),
			Bid:          mdata.BestBidPrice.Float64(),
			BidSize:      mdata.BestBidSize.Float64(),
			Ask:          mdata.BestAskPrice.Float64(),
			AskSize:      mdata.BestAskSize.Float64(),
			Volume:       baseVolume,
			QuoteVolume:  quoteVolume,
			Open:         mdata.Open24H.Float64(),
			Pair:         p,
			ExchangeName: ok.Name,
			AssetType:    a,
		}); err != nil {
			return nil, err
		}
	}

	return ticker.GetTicker(ok.Name, p, a)
}

// UpdateTickers updates all currency pairs of a given asset type
func (ok *Okx) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spread:
		format, err := ok.GetPairFormat(asset.Spread, true)
		if err != nil {
			return err
		}
		pairs, err := ok.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}
		for y := range pairs {
			var spreadTickers []SpreadTicker
			spreadTickers, err = ok.GetPublicSpreadTickers(ctx, format.Format(pairs[y]))
			if err != nil {
				return err
			}
			for x := range spreadTickers {
				pair, err := currency.NewPairDelimiter(spreadTickers[x].SpreadID, format.Delimiter)
				if err != nil {
					return err
				}
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         spreadTickers[x].Last.Float64(),
					Bid:          spreadTickers[x].BidPrice.Float64(),
					BidSize:      spreadTickers[x].BidSize.Float64(),
					Ask:          spreadTickers[x].AskPrice.Float64(),
					AskSize:      spreadTickers[x].AskSize.Float64(),
					Pair:         pair,
					ExchangeName: ok.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return err
				}
			}
		}
	case asset.Spot, asset.PerpetualSwap, asset.Futures, asset.Options, asset.Margin:
		pairs, err := ok.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}

		instrumentType := GetInstrumentTypeFromAssetItem(assetType)
		if assetType == asset.Margin {
			instrumentType = instTypeSpot
		}
		ticks, err := ok.GetTickers(ctx, instrumentType, "", "")
		if err != nil {
			return err
		}

		for y := range ticks {
			pair, err := ok.GetPairFromInstrumentID(ticks[y].InstrumentID.String())
			if err != nil {
				return err
			}
			for i := range pairs {
				pairFmt, err := ok.FormatExchangeCurrency(pairs[i], assetType)
				if err != nil {
					return err
				}
				if !pair.Equal(pairFmt) {
					continue
				}
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         ticks[y].LastTradePrice.Float64(),
					High:         ticks[y].High24H.Float64(),
					Low:          ticks[y].Low24H.Float64(),
					Bid:          ticks[y].BestBidPrice.Float64(),
					BidSize:      ticks[y].BestBidSize.Float64(),
					Ask:          ticks[y].BestAskPrice.Float64(),
					AskSize:      ticks[y].BestAskSize.Float64(),
					Volume:       ticks[y].Vol24H.Float64(),
					QuoteVolume:  ticks[y].VolCcy24H.Float64(),
					Open:         ticks[y].Open24H.Float64(),
					Pair:         pairFmt,
					ExchangeName: ok.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (ok *Okx) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var err error
	switch assetType {
	case asset.Spread:
		var (
			pairFormat      currency.PairFormat
			spreadOrderbook []SpreadOrderbook
		)
		pairFormat, err = ok.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		spreadOrderbook, err = ok.GetPublicSpreadOrderBooks(ctx, pairFormat.Format(pair), 50)
		if err != nil {
			return nil, err
		}
		for y := range spreadOrderbook {
			book := &orderbook.Book{
				Exchange:          ok.Name,
				Pair:              pair,
				Asset:             assetType,
				ValidateOrderbook: ok.ValidateOrderbook,
			}
			book.Bids = make(orderbook.Levels, 0, len(spreadOrderbook[y].Bids))
			for b := range spreadOrderbook[y].Bids {
				// Skip order book bid depths where the price value is zero.
				if spreadOrderbook[y].Bids[b][0].Float64() == 0 {
					continue
				}
				book.Bids = append(book.Bids, orderbook.Level{
					Price:      spreadOrderbook[y].Bids[b][0].Float64(),
					Amount:     spreadOrderbook[y].Bids[b][1].Float64(),
					OrderCount: spreadOrderbook[y].Bids[b][2].Int64(),
				})
			}
			book.Asks = make(orderbook.Levels, 0, len(spreadOrderbook[y].Asks))
			for a := range spreadOrderbook[y].Asks {
				// Skip order book ask depths where the price value is zero.
				if spreadOrderbook[y].Asks[a][0].Float64() == 0 {
					continue
				}
				book.Asks = append(book.Asks, orderbook.Level{
					Price:      spreadOrderbook[y].Asks[a][0].Float64(),
					Amount:     spreadOrderbook[y].Asks[a][1].Float64(),
					OrderCount: spreadOrderbook[y].Asks[a][2].Int64(),
				})
			}
			err = book.Process()
			if err != nil {
				return book, err
			}
		}
	case asset.Spot, asset.Options, asset.Margin, asset.PerpetualSwap, asset.Futures:
		err = ok.CurrencyPairs.IsAssetEnabled(assetType)
		if err != nil {
			return nil, err
		}
		var instrumentID string
		pairFormat, err := ok.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		if !pair.IsPopulated() {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		instrumentID = pairFormat.Format(pair)
		book := &orderbook.Book{
			Exchange:          ok.Name,
			Pair:              pair,
			Asset:             assetType,
			ValidateOrderbook: ok.ValidateOrderbook,
		}
		var orderBookD *OrderBookResponseDetail
		orderBookD, err = ok.GetOrderBookDepth(ctx, instrumentID, 400)
		if err != nil {
			return book, err
		}

		book.Bids = make(orderbook.Levels, len(orderBookD.Bids))
		for x := range orderBookD.Bids {
			book.Bids[x] = orderbook.Level{
				Amount: orderBookD.Bids[x].Amount.Float64(),
				Price:  orderBookD.Bids[x].DepthPrice.Float64(),
			}
		}
		book.Asks = make(orderbook.Levels, len(orderBookD.Asks))
		for x := range orderBookD.Asks {
			book.Asks[x] = orderbook.Level{
				Amount: orderBookD.Asks[x].Amount.Float64(),
				Price:  orderBookD.Asks[x].DepthPrice.Float64(),
			}
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return orderbook.Get(ok.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies.
func (ok *Okx) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	if err := ok.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return account.Holdings{}, err
	}

	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = ok.Name
	if !ok.SupportsAsset(assetType) {
		return info, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	accountBalances, err := ok.AccountBalance(ctx, currency.EMPTYCODE)
	if err != nil {
		return info, err
	}
	currencyBalances := []account.Balance{}
	for i := range accountBalances {
		for j := range accountBalances[i].Details {
			currencyBalances = append(currencyBalances, account.Balance{
				Currency: accountBalances[i].Details[j].Currency,
				Total:    accountBalances[i].Details[j].EquityOfCurrency.Float64(),
				Hold:     accountBalances[i].Details[j].FrozenBalance.Float64(),
				Free:     accountBalances[i].Details[j].AvailableBalance.Float64(),
			})
		}
	}
	acc.Currencies = currencyBalances
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	creds, err := ok.GetCredentials(ctx)
	if err != nil {
		return info, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (ok *Okx) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	depositHistories, err := ok.GetCurrencyDepositHistory(ctx, currency.EMPTYCODE, "", "", "", "", time.Time{}, time.Time{}, -1, 0)
	if err != nil {
		return nil, err
	}

	withdrawalHistories, err := ok.GetWithdrawalHistory(ctx, currency.EMPTYCODE, "", "", "", "", time.Time{}, time.Time{}, -5)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(depositHistories)+len(withdrawalHistories))
	for x := range depositHistories {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    ok.Name,
			Status:          strconv.FormatInt(depositHistories[x].State.Int64(), 10),
			Timestamp:       depositHistories[x].Timestamp.Time(),
			Currency:        depositHistories[x].Currency,
			Amount:          depositHistories[x].Amount.Float64(),
			TransferType:    "deposit",
			CryptoToAddress: depositHistories[x].ToDepositAddress,
			CryptoTxID:      depositHistories[x].TransactionID,
		})
	}
	for x := range withdrawalHistories {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    ok.Name,
			Status:          withdrawalHistories[x].StateOfWithdrawal,
			Timestamp:       withdrawalHistories[x].Timestamp.Time(),
			Currency:        withdrawalHistories[x].Currency,
			Amount:          withdrawalHistories[x].Amount.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalHistories[x].ToReceivingAddress,
			CryptoTxID:      withdrawalHistories[x].TransactionID,
			TransferID:      withdrawalHistories[x].WithdrawalID,
			Fee:             withdrawalHistories[x].WithdrawalFee.Float64(),
			CryptoChain:     withdrawalHistories[x].ChainName,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (ok *Okx) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := ok.GetWithdrawalHistory(ctx, c, "", "", "", "", time.Time{}, time.Time{}, -5)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for x := range withdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          withdrawals[x].StateOfWithdrawal,
			Timestamp:       withdrawals[x].Timestamp.Time(),
			Currency:        withdrawals[x].Currency,
			Amount:          withdrawals[x].Amount.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[x].ToReceivingAddress,
			CryptoTxID:      withdrawals[x].TransactionID,
			CryptoChain:     withdrawals[x].ChainName,
			TransferID:      withdrawals[x].WithdrawalID,
			Fee:             withdrawals[x].WithdrawalFee.Float64(),
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (ok *Okx) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	format, err := ok.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Spread:
		var spreadTrades []SpreadPublicTradeItem
		spreadTrades, err = ok.GetPublicSpreadTrades(ctx, "")
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(spreadTrades))
		var oSide order.Side
		for x := range spreadTrades {
			oSide, err = order.StringToOrderSide(spreadTrades[x].Side)
			if err != nil {
				return nil, err
			}
			resp[x] = trade.Data{
				TID:          spreadTrades[x].TradeID,
				Exchange:     ok.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         oSide,
				Price:        spreadTrades[x].Price.Float64(),
				Amount:       spreadTrades[x].Size.Float64(),
				Timestamp:    spreadTrades[x].Timestamp.Time(),
			}
		}
	case asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options:
		if p.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		instrumentID := format.Format(p)
		var tradeData []TradeResponse
		tradeData, err = ok.GetTrades(ctx, instrumentID, 1000)
		if err != nil {
			return nil, err
		}

		resp = make([]trade.Data, len(tradeData))
		for x := range tradeData {
			resp[x] = trade.Data{
				TID:          tradeData[x].TradeID,
				Exchange:     ok.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         tradeData[x].Side,
				Price:        tradeData[x].Price.Float64(),
				Amount:       tradeData[x].Quantity.Float64(),
				Timestamp:    tradeData[x].Timestamp.Time(),
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if ok.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades retrieves historic trade data within the timeframe provided
func (ok *Okx) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if !ok.SupportsAsset(assetType) || assetType == asset.Spread {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}

	if timestampStart.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	const limit = 100
	pairFormat, err := ok.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp []trade.Data
	instrumentID := pairFormat.Format(p)
	tradeIDEnd := ""
allTrades:
	for {
		var trades []TradeResponse
		trades, err = ok.GetTradesHistory(ctx, instrumentID, "", tradeIDEnd, limit)
		if err != nil {
			return nil, err
		}
		if len(trades) == 0 {
			break
		}
		for i := range trades {
			if timestampStart.Equal(trades[i].Timestamp.Time()) ||
				trades[i].Timestamp.Time().Before(timestampStart) ||
				tradeIDEnd == trades[len(trades)-1].TradeID {
				// reached end of trades to crawl
				break allTrades
			}
			resp = append(resp, trade.Data{
				TID:          trades[i].TradeID,
				Exchange:     ok.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Quantity.Float64(),
				Timestamp:    trades[i].Timestamp.Time(),
				Side:         trades[i].Side,
			})
		}
		tradeIDEnd = trades[len(trades)-1].TradeID
	}
	if ok.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (ok *Okx) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if !ok.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
	if s.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	pairFormat, err := ok.GetPairFormat(s.AssetType, true)
	if err != nil {
		return nil, err
	}
	if s.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	pairString := pairFormat.Format(s.Pair)
	tradeMode := ok.marginTypeToString(s.MarginType)
	if s.AssetType.IsFutures() && s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w received '%v'", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	var sideType, positionSide string
	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.Spread:
		sideType = s.Side.String()
	case asset.Futures, asset.PerpetualSwap, asset.Options:
		positionSide = s.Side.Lower()
	}
	amount := s.Amount
	var targetCurrency string
	if s.AssetType == asset.Spot && s.Type == order.Market {
		targetCurrency = "base_ccy" // Default to base currency
		if s.QuoteAmount > 0 {
			amount = s.QuoteAmount
			targetCurrency = "quote_ccy"
		}
	}
	// If asset type is spread
	if s.AssetType == asset.Spread {
		spreadParam := &SpreadOrderParam{
			SpreadID:      pairString,
			ClientOrderID: s.ClientOrderID,
			Side:          sideType,
			OrderType:     s.Type.Lower(),
			Size:          s.Amount,
			Price:         s.Price,
		}
		var placeSpreadOrderResponse *SpreadOrderResponse
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			placeSpreadOrderResponse, err = ok.WSPlaceSpreadOrder(ctx, spreadParam)
			if err != nil {
				return nil, err
			}
		} else {
			placeSpreadOrderResponse, err = ok.PlaceSpreadOrder(ctx, spreadParam)
			if err != nil {
				return nil, err
			}
		}
		return s.DeriveSubmitResponse(placeSpreadOrderResponse.OrderID)
	}
	orderTypeString, err := orderTypeString(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	var placeOrderResponse *OrderData
	var result *AlgoOrder
	switch orderTypeString {
	case orderLimit, orderMarket, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, "mmp", "mmp_and_post_only":
		orderRequest := &PlaceOrderRequestParam{
			InstrumentID:   pairString,
			TradeMode:      tradeMode,
			Side:           sideType,
			PositionSide:   positionSide,
			OrderType:      orderTypeString,
			Amount:         amount,
			ClientOrderID:  s.ClientOrderID,
			Price:          s.Price,
			TargetCurrency: targetCurrency,
			AssetType:      s.AssetType,
		}
		switch s.Type.Lower() {
		case orderLimit, orderPostOnly, orderFOK, orderIOC:
			orderRequest.Price = s.Price
		}
		if s.AssetType == asset.PerpetualSwap || s.AssetType == asset.Futures {
			if s.Type.Lower() == "" {
				orderRequest.OrderType = orderOptimalLimitIOC
			}
			// TODO: handle positionSideLong while side is Short and positionSideShort while side is Long
			if s.Side.IsLong() {
				orderRequest.PositionSide = positionSideLong
			} else {
				orderRequest.PositionSide = positionSideShort
			}
		}
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			placeOrderResponse, err = ok.WSPlaceOrder(ctx, orderRequest)
		} else {
			placeOrderResponse, err = ok.PlaceOrder(ctx, orderRequest)
		}
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(placeOrderResponse.OrderID)
	case orderTrigger:
		result, err = ok.PlaceTriggerAlgoOrder(ctx, &AlgoOrderParams{
			InstrumentID:     pairString,
			TradeMode:        tradeMode,
			Side:             s.Side.Lower(),
			PositionSide:     positionSide,
			OrderType:        orderTypeString,
			Size:             s.Amount,
			ReduceOnly:       s.ReduceOnly,
			TriggerPrice:     s.TriggerPrice,
			TriggerPriceType: priceTypeString(s.TriggerPriceType),
		})
	case orderConditional:
		// Trigger Price and type are used as a stop losss trigger price and type.
		result, err = ok.PlaceTakeProfitStopLossOrder(ctx, &AlgoOrderParams{
			InstrumentID:             pairString,
			TradeMode:                tradeMode,
			Side:                     s.Side.Lower(),
			PositionSide:             positionSide,
			OrderType:                orderTypeString,
			Size:                     s.Amount,
			ReduceOnly:               s.ReduceOnly,
			StopLossTriggerPrice:     s.TriggerPrice,
			StopLossOrderPrice:       s.Price,
			StopLossTriggerPriceType: priceTypeString(s.TriggerPriceType),
		})
	case orderChase:
		if s.TrackingMode == order.UnknownTrackingMode {
			return nil, fmt.Errorf("%w, tracking mode unset", order.ErrUnknownTrackingMode)
		}
		if s.TrackingValue == 0 {
			return nil, fmt.Errorf("%w, tracking value required", order.ErrAmountBelowMin)
		}
		result, err = ok.PlaceChaseAlgoOrder(ctx, &AlgoOrderParams{
			InstrumentID:  pairString,
			TradeMode:     tradeMode,
			Side:          s.Side.Lower(),
			PositionSide:  positionSide,
			OrderType:     orderTypeString,
			Size:          s.Amount,
			ReduceOnly:    s.ReduceOnly,
			MaxChaseType:  s.TrackingMode.String(),
			MaxChaseValue: s.TrackingValue,
		})
	case orderMoveOrderStop:
		if s.TrackingMode == order.UnknownTrackingMode {
			return nil, fmt.Errorf("%w, tracking mode unset", order.ErrUnknownTrackingMode)
		}
		var callbackSpread, callbackRatio float64
		switch s.TrackingMode {
		case order.Distance:
			callbackSpread = s.TrackingValue
		case order.Percentage:
			callbackRatio = s.TrackingValue
		}
		result, err = ok.PlaceTrailingStopOrder(ctx, &AlgoOrderParams{
			InstrumentID:           pairString,
			TradeMode:              tradeMode,
			Side:                   sideType,
			PositionSide:           positionSide,
			OrderType:              orderTypeString,
			Size:                   s.Amount,
			ReduceOnly:             s.ReduceOnly,
			CallbackRatio:          callbackRatio,
			CallbackSpreadVariance: callbackSpread,
			ActivePrice:            s.TriggerPrice,
		})
	case orderTWAP:
		if s.TrackingMode == order.UnknownTrackingMode {
			return nil, fmt.Errorf("%w, tracking mode unset", order.ErrUnknownTrackingMode)
		}
		var priceVar, priceSpread float64
		switch s.TrackingMode {
		case order.Distance:
			priceSpread = s.TrackingValue
		case order.Percentage:
			priceVar = s.TrackingValue
		}
		result, err = ok.PlaceTWAPOrder(ctx, &AlgoOrderParams{
			InstrumentID:  pairString,
			TradeMode:     tradeMode,
			Side:          sideType,
			PositionSide:  positionSide,
			OrderType:     orderTypeString,
			Size:          s.Amount,
			ReduceOnly:    s.ReduceOnly,
			PriceVariance: priceVar,
			PriceSpread:   priceSpread,
			SizeLimit:     s.Amount,
			LimitPrice:    s.Price,
			TimeInterval:  kline.FifteenMin,
		})
	case orderOCO:
		switch {
		case s.RiskManagementModes.TakeProfit.Price <= 0:
			return nil, fmt.Errorf("%w, take profit price is required", order.ErrPriceBelowMin)
		case s.RiskManagementModes.StopLoss.Price <= 0:
			return nil, fmt.Errorf("%w, stop loss price is required", order.ErrPriceBelowMin)
		}
		result, err = ok.PlaceAlgoOrder(ctx, &AlgoOrderParams{
			InstrumentID: pairString,
			TradeMode:    tradeMode,
			Side:         sideType,
			PositionSide: positionSide,
			OrderType:    orderTypeString,
			Size:         s.Amount,
			ReduceOnly:   s.ReduceOnly,

			TakeProfitTriggerPrice:     s.RiskManagementModes.TakeProfit.Price,
			TakeProfitOrderPrice:       s.RiskManagementModes.TakeProfit.LimitPrice,
			TakeProfitTriggerPriceType: priceTypeString(s.TriggerPriceType),

			StopLossTriggerPrice:     s.RiskManagementModes.TakeProfit.Price,
			StopLossOrderPrice:       s.RiskManagementModes.StopLoss.LimitPrice,
			StopLossTriggerPriceType: priceTypeString(s.TriggerPriceType),
		})
	default:
		return nil, fmt.Errorf("%w, order type %s", order.ErrTypeIsInvalid, orderTypeString)
	}
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(result.AlgoID)
}

func priceTypeString(pt order.PriceType) string {
	switch pt {
	case order.LastPrice:
		return "last"
	case order.IndexPrice:
		return "index"
	case order.MarkPrice:
		return "mark"
	default:
		return ""
	}
}

var allowedMarginTypes = margin.Isolated | margin.NoMargin | margin.SpotIsolated

func (ok *Okx) marginTypeToString(m margin.Type) string {
	if allowedMarginTypes&m == m {
		return m.String()
	} else if margin.Multi == m {
		return TradeModeCross
	}
	return ""
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (ok *Okx) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var err error
	if math.Trunc(action.Amount) != action.Amount {
		return nil, errors.New("contract amount can not be decimal")
	}
	// When asset type is asset.Spread
	if action.AssetType == asset.Spread {
		amendSpreadOrder := &AmendSpreadOrderParam{
			OrderID:       action.OrderID,
			ClientOrderID: action.ClientOrderID,
			NewSize:       action.Amount,
			NewPrice:      action.Price,
		}
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			_, err = ok.WSAmendSpreadOrder(ctx, amendSpreadOrder)
		} else {
			_, err = ok.AmendSpreadOrder(ctx, amendSpreadOrder)
		}
		if err != nil {
			return nil, err
		}
		return action.DeriveModifyResponse()
	}

	// For other asset type instances.
	pairFormat, err := ok.GetPairFormat(action.AssetType, true)
	if err != nil {
		return nil, err
	}
	if action.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch action.Type {
	case order.UnknownType, order.Market, order.Limit, order.OptimalLimit, order.MarketMakerProtection:
		amendRequest := AmendOrderRequestParams{
			InstrumentID:  pairFormat.Format(action.Pair),
			NewQuantity:   action.Amount,
			OrderID:       action.OrderID,
			ClientOrderID: action.ClientOrderID,
		}
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			_, err = ok.WSAmendOrder(ctx, &amendRequest)
		} else {
			_, err = ok.AmendOrder(ctx, &amendRequest)
		}
		if err != nil {
			return nil, err
		}
	case order.Trigger:
		if action.TriggerPrice == 0 {
			return nil, fmt.Errorf("%w, trigger price required", order.ErrPriceBelowMin)
		}
		var postTriggerTPSLOrders []SubTPSLParams
		if action.RiskManagementModes.StopLoss.Price > 0 && action.RiskManagementModes.TakeProfit.Price > 0 {
			postTriggerTPSLOrders = []SubTPSLParams{
				{
					NewTakeProfitTriggerPrice:     action.RiskManagementModes.TakeProfit.Price,
					NewTakeProfitOrderPrice:       action.RiskManagementModes.TakeProfit.LimitPrice,
					NewStopLossTriggerPrice:       action.RiskManagementModes.StopLoss.Price,
					NewStopLossOrderPrice:         action.RiskManagementModes.StopLoss.Price,
					NewTakeProfitTriggerPriceType: priceTypeString(action.RiskManagementModes.TakeProfit.TriggerPriceType),
					NewStopLossTriggerPriceType:   priceTypeString(action.RiskManagementModes.StopLoss.TriggerPriceType),
				},
			}
		}
		_, err = ok.AmendAlgoOrder(ctx, &AmendAlgoOrderParam{
			InstrumentID:              pairFormat.Format(action.Pair),
			AlgoID:                    action.OrderID,
			ClientSuppliedAlgoOrderID: action.ClientOrderID,
			NewSize:                   action.Amount,

			NewTriggerPrice:     action.TriggerPrice,
			NewOrderPrice:       action.Price,
			NewTriggerPriceType: priceTypeString(action.TriggerPriceType),

			// An one-cancel-other order to be placed after executing the trigger order
			AttachAlgoOrders: postTriggerTPSLOrders,
		})
		if err != nil {
			return nil, err
		}
	case order.OCO:
		switch {
		case action.RiskManagementModes.TakeProfit.Price <= 0 &&
			action.RiskManagementModes.TakeProfit.LimitPrice <= 0:
			return nil, fmt.Errorf("%w, either take profit trigger price or order price is required", order.ErrPriceBelowMin)
		case action.RiskManagementModes.StopLoss.Price <= 0 &&
			action.RiskManagementModes.StopLoss.LimitPrice <= 0:
			return nil, fmt.Errorf("%w, either stop loss trigger price or order price is required", order.ErrPriceBelowMin)
		}
		_, err = ok.AmendAlgoOrder(ctx, &AmendAlgoOrderParam{
			InstrumentID:              pairFormat.Format(action.Pair),
			AlgoID:                    action.OrderID,
			ClientSuppliedAlgoOrderID: action.ClientOrderID,
			NewSize:                   action.Amount,

			NewTakeProfitTriggerPrice: action.RiskManagementModes.TakeProfit.Price,
			NewTakeProfitOrderPrice:   action.RiskManagementModes.TakeProfit.LimitPrice,

			NewStopLossTriggerPrice: action.RiskManagementModes.StopLoss.Price,
			NewStopLossOrderPrice:   action.RiskManagementModes.StopEntry.LimitPrice,

			NewTakeProfitTriggerPriceType: priceTypeString(action.RiskManagementModes.TakeProfit.TriggerPriceType),
			NewStopLossTriggerPriceType:   priceTypeString(action.RiskManagementModes.StopLoss.TriggerPriceType),
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w, could not amend order of type %v", order.ErrUnsupportedOrderType, action.Type)
	}
	return action.DeriveModifyResponse()
}

// CancelOrder cancels an order by its corresponding ID number
func (ok *Okx) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if !ok.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	var err error
	if ord.AssetType == asset.Spread {
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			_, err = ok.WSCancelSpreadOrder(ctx, ord.OrderID, ord.ClientOrderID)
		} else {
			_, err = ok.CancelSpreadOrder(ctx, ord.OrderID, ord.ClientOrderID)
		}
		return err
	}
	pairFormat, err := ok.GetPairFormat(ord.AssetType, true)
	if err != nil {
		return err
	}
	if ord.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(ord.Pair)
	switch ord.Type {
	case order.UnknownType, order.Market, order.Limit, order.OptimalLimit, order.MarketMakerProtection:
		req := CancelOrderRequestParam{
			InstrumentID:  instrumentID,
			OrderID:       ord.OrderID,
			ClientOrderID: ord.ClientOrderID,
		}
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			_, err = ok.WSCancelOrder(ctx, &req)
		} else {
			_, err = ok.CancelSingleOrder(ctx, &req)
		}
	case order.Trigger, order.OCO, order.ConditionalStop, order.TWAP, order.TrailingStop, order.Chase:
		var response *AlgoOrder
		response, err = ok.CancelAdvanceAlgoOrder(ctx, []AlgoOrderCancelParams{
			{
				AlgoOrderID:  ord.OrderID,
				InstrumentID: instrumentID,
			},
		})
		if err != nil {
			return err
		}
		return getStatusError(response.StatusCode, response.StatusMessage)
	default:
		return fmt.Errorf("%w, order type %v", order.ErrUnsupportedOrderType, ord.Type)
	}
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (ok *Okx) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) > 20 {
		return nil, fmt.Errorf("%w, cannot cancel more than 20 orders", errExceedLimit)
	} else if len(o) == 0 {
		return nil, fmt.Errorf("%w, must have at least 1 cancel order", order.ErrCancelOrderIsNil)
	}
	cancelOrderParams := make([]CancelOrderRequestParam, 0, len(o))
	cancelAlgoOrderParams := make([]AlgoOrderCancelParams, 0, len(o))
	var err error
	for x := range o {
		ord := o[x]
		if !ok.SupportsAsset(ord.AssetType) {
			return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
		}
		var pairFormat currency.PairFormat
		pairFormat, err = ok.GetPairFormat(ord.AssetType, true)
		if err != nil {
			return nil, err
		}
		if !ord.Pair.IsPopulated() {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		switch ord.Type {
		case order.UnknownType, order.Market, order.Limit, order.OptimalLimit, order.MarketMakerProtection:
			if o[x].ClientID == "" && o[x].OrderID == "" {
				return nil, fmt.Errorf("%w, order ID required for order of type %v", order.ErrOrderIDNotSet, o[x].Type)
			}
			cancelOrderParams = append(cancelOrderParams, CancelOrderRequestParam{
				InstrumentID:  pairFormat.Format(ord.Pair),
				OrderID:       ord.OrderID,
				ClientOrderID: ord.ClientOrderID,
			})
		case order.Trigger, order.OCO, order.ConditionalStop,
			order.TWAP, order.TrailingStop, order.Chase:
			if o[x].OrderID == "" {
				return nil, fmt.Errorf("%w, order ID required for order of type %v", order.ErrOrderIDNotSet, o[x].Type)
			}
			cancelAlgoOrderParams = append(cancelAlgoOrderParams, AlgoOrderCancelParams{
				AlgoOrderID:  o[x].OrderID,
				InstrumentID: pairFormat.Format(ord.Pair),
			})
		default:
			return nil, fmt.Errorf("%w order of type %v not supported", order.ErrUnsupportedOrderType, o[x].Type)
		}
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	if len(cancelOrderParams) > 0 {
		var canceledOrders []*OrderData
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			canceledOrders, err = ok.WSCancelMultipleOrders(ctx, cancelOrderParams)
		} else {
			canceledOrders, err = ok.CancelMultipleOrders(ctx, cancelOrderParams)
		}
		if err != nil {
			return nil, err
		}
		for x := range canceledOrders {
			resp.Status[canceledOrders[x].OrderID] = func() string {
				if canceledOrders[x].StatusCode != 0 {
					return ""
				}
				return order.Cancelled.String()
			}()
		}
	}
	if len(cancelAlgoOrderParams) > 0 {
		cancelationResponse, err := ok.CancelAdvanceAlgoOrder(ctx, cancelAlgoOrderParams)
		if err != nil {
			if len(resp.Status) > 0 {
				return resp, nil
			}
			return nil, err
		} else if cancelationResponse.StatusCode != 0 {
			if len(resp.Status) > 0 {
				return resp, nil
			}
			return resp, getStatusError(cancelationResponse.StatusCode, cancelationResponse.StatusMessage)
		}
		for x := range cancelAlgoOrderParams {
			resp.Status[cancelAlgoOrderParams[x].AlgoOrderID] = order.Cancelled.String()
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (ok *Okx) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	err := orderCancellation.Validate()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}

	// For asset.Spread asset orders cancellation
	if orderCancellation.AssetType == asset.Spread {
		var success bool
		success, err = ok.CancelAllSpreadOrders(ctx, orderCancellation.OrderID)
		if err != nil {
			return cancelAllResponse, err
		}
		cancelAllResponse.Status[orderCancellation.OrderID] = strconv.FormatBool(success)
		return cancelAllResponse, nil
	}

	var instrumentType string
	if orderCancellation.AssetType.IsValid() {
		err = ok.CurrencyPairs.IsAssetEnabled(orderCancellation.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		instrumentType = GetInstrumentTypeFromAssetItem(orderCancellation.AssetType)
	}
	var oType string
	if orderCancellation.Type != order.UnknownType && orderCancellation.Type != order.AnyType {
		oType, err = orderTypeString(orderCancellation.Type, orderCancellation.TimeInForce)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
	}
	var curr string
	if orderCancellation.Pair.IsPopulated() {
		curr = orderCancellation.Pair.Upper().String()
	}
	myOrders, err := ok.GetOrderList(ctx, &OrderListRequestParams{
		InstrumentType: instrumentType,
		OrderType:      oType,
		InstrumentID:   curr,
	})
	if err != nil {
		return cancelAllResponse, err
	}
	cancelAllOrdersRequestParams := make([]CancelOrderRequestParam, len(myOrders))
ordersLoop:
	for x := range myOrders {
		switch {
		case orderCancellation.OrderID != "" || orderCancellation.ClientOrderID != "":
			if myOrders[x].OrderID == orderCancellation.OrderID ||
				myOrders[x].ClientOrderID == orderCancellation.ClientOrderID {
				cancelAllOrdersRequestParams[x] = CancelOrderRequestParam{
					OrderID:       myOrders[x].OrderID,
					ClientOrderID: myOrders[x].ClientOrderID,
				}
				break ordersLoop
			}
		case orderCancellation.Side == order.Buy || orderCancellation.Side == order.Sell:
			if myOrders[x].Side == order.Buy || myOrders[x].Side == order.Sell {
				cancelAllOrdersRequestParams[x] = CancelOrderRequestParam{
					OrderID:       myOrders[x].OrderID,
					ClientOrderID: myOrders[x].ClientOrderID,
				}
				continue
			}
		default:
			cancelAllOrdersRequestParams[x] = CancelOrderRequestParam{
				OrderID:       myOrders[x].OrderID,
				ClientOrderID: myOrders[x].ClientOrderID,
			}
		}
	}
	remaining := cancelAllOrdersRequestParams
	loop := int(math.Ceil(float64(len(remaining)) / 20.0))
	for range loop {
		var response []*OrderData
		if len(remaining) > 20 {
			if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				response, err = ok.WSCancelMultipleOrders(ctx, remaining[:20])
			} else {
				response, err = ok.CancelMultipleOrders(ctx, remaining[:20])
			}
			remaining = remaining[20:]
		} else {
			if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				response, err = ok.WSCancelMultipleOrders(ctx, remaining)
			} else {
				response, err = ok.CancelMultipleOrders(ctx, remaining)
			}
		}
		if err != nil {
			if len(cancelAllResponse.Status) == 0 {
				return cancelAllResponse, err
			}
		}
		for y := range response {
			if response[y].StatusCode == 0 {
				cancelAllResponse.Status[response[y].OrderID] = order.Cancelled.String()
			} else {
				cancelAllResponse.Status[response[y].OrderID] = response[y].StatusMessage
			}
		}
	}
	return cancelAllResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (ok *Okx) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if !ok.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if assetType == asset.Spread {
		var resp *SpreadOrder
		resp, err := ok.GetSpreadOrderDetails(ctx, orderID, "")
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(resp.Side)
		if err != nil {
			return nil, err
		}
		oType, err := order.StringToOrderType(resp.OrderType)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(resp.State)
		if err != nil {
			return nil, err
		}
		cp, err := currency.NewPairFromString(resp.InstrumentID)
		if err != nil {
			return nil, err
		}
		if !pair.IsEmpty() && !cp.Equal(pair) {
			return nil, fmt.Errorf("%w, unexpected instrument ID %v for order ID %s", order.ErrOrderNotFound, pair, orderID)
		}
		return &order.Detail{
			Amount:               resp.Size.Float64(),
			Exchange:             ok.Name,
			OrderID:              resp.OrderID,
			ClientOrderID:        resp.ClientOrderID,
			Side:                 oSide,
			Type:                 oType,
			Pair:                 cp,
			Cost:                 resp.Price.Float64(),
			AssetType:            assetType,
			Status:               oStatus,
			Price:                resp.Price.Float64(),
			ExecutedAmount:       resp.FillSize.Float64(),
			Date:                 resp.CreationTime.Time(),
			LastUpdated:          resp.UpdateTime.Time(),
			AverageExecutedPrice: resp.AveragePrice.Float64(),
			RemainingAmount:      resp.Size.Float64() - resp.FillSize.Float64(),
		}, nil
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := ok.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	pairFormat, err := ok.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !pair.IsPopulated() {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	instrumentID := pairFormat.Format(pair)
	orderDetail, err := ok.GetOrderDetail(ctx, &OrderDetailRequestParam{
		InstrumentID: instrumentID,
		OrderID:      orderID,
	})
	if err != nil {
		return nil, err
	}
	status, err := order.StringToOrderStatus(orderDetail.State)
	if err != nil {
		return nil, err
	}
	orderType, tif, err := orderTypeFromString(orderDetail.OrderType)
	if err != nil {
		return nil, err
	}

	return &order.Detail{
		Amount:         orderDetail.Size.Float64(),
		Exchange:       ok.Name,
		OrderID:        orderDetail.OrderID,
		ClientOrderID:  orderDetail.ClientOrderID,
		Side:           orderDetail.Side,
		Type:           orderType,
		Pair:           pair,
		Cost:           orderDetail.Price.Float64(),
		AssetType:      assetType,
		Status:         status,
		Price:          orderDetail.Price.Float64(),
		ExecutedAmount: orderDetail.RebateAmount.Float64(),
		Date:           orderDetail.CreationTime.Time(),
		LastUpdated:    orderDetail.UpdateTime.Time(),
		TimeInForce:    tif,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (ok *Okx) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	response, err := ok.GetCurrencyDepositAddress(ctx, c)
	if err != nil {
		return nil, err
	}

	// Check if a specific chain was requested
	if chain != "" {
		for x := range response {
			if !strings.EqualFold(response[x].Chain, chain) {
				continue
			}
			return &deposit.Address{
				Address: response[x].Address,
				Tag:     response[x].Tag,
				Chain:   response[x].Chain,
			}, nil
		}
		return nil, fmt.Errorf("specified chain %s not found", chain)
	}

	// If no specific chain was requested, return the first selected address (mainnet addresses are returned first by default)
	for x := range response {
		if !response[x].Selected {
			continue
		}

		return &deposit.Address{
			Address: response[x].Address,
			Tag:     response[x].Tag,
			Chain:   response[x].Chain,
		}, nil
	}
	return nil, deposit.ErrAddressNotFound
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (ok *Okx) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	input := WithdrawalInput{
		ChainName:             withdrawRequest.Crypto.Chain,
		Amount:                withdrawRequest.Amount,
		Currency:              withdrawRequest.Currency,
		ToAddress:             withdrawRequest.Crypto.Address,
		TransactionFee:        withdrawRequest.Crypto.FeeAmount,
		WithdrawalDestination: "3",
	}
	resp, err := ok.Withdrawal(ctx, &input)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.WithdrawalID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (ok *Okx) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (ok *Okx) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (ok *Okx) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.StartTime.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	if !ok.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}

	var resp []order.Detail
	var format currency.PairFormat
	if req.AssetType == asset.Spread {
		var spreads []SpreadOrder
		spreads, err = ok.GetActiveSpreadOrders(ctx, "", req.Type.String(), "", req.FromOrderID, "", 0)
		if err != nil {
			return nil, err
		}
		for x := range spreads {
			format, err = ok.GetPairFormat(asset.Spread, true)
			if err != nil {
				return nil, err
			}
			var (
				pair    currency.Pair
				oType   order.Type
				oSide   order.Side
				oStatus order.Status
			)

			pair, err = currency.NewPairDelimiter(spreads[x].SpreadID, format.Delimiter)
			if err != nil {
				return nil, err
			}
			oType, err = order.StringToOrderType(spreads[x].OrderType)
			if err != nil {
				return nil, err
			}
			oSide, err = order.StringToOrderSide(spreads[x].Side)
			if err != nil {
				return nil, err
			}
			oStatus, err = order.StringToOrderStatus(spreads[x].State)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				Amount:          spreads[x].Size.Float64(),
				Pair:            pair,
				Price:           spreads[x].Price.Float64(),
				ExecutedAmount:  spreads[x].FillSize.Float64(),
				RemainingAmount: spreads[x].Size.Float64() - spreads[x].FillSize.Float64(),
				Exchange:        ok.Name,
				OrderID:         spreads[x].OrderID,
				ClientOrderID:   spreads[x].ClientOrderID,
				Type:            oType,
				Side:            oSide,
				Status:          oStatus,
				AssetType:       req.AssetType,
				Date:            spreads[x].CreationTime.Time(),
				LastUpdated:     spreads[x].UpdateTime.Time(),
			})
		}
		return req.Filter(ok.Name, resp), nil
	}

	instrumentType := GetInstrumentTypeFromAssetItem(req.AssetType)
	var orderType string
	if req.Type != order.UnknownType && req.Type != order.AnyType {
		orderType, err = orderTypeString(req.Type, req.TimeInForce)
		if err != nil {
			return nil, err
		}
	}
	endTime := req.EndTime
allOrders:
	for {
		requestParam := &OrderListRequestParams{
			OrderType:      orderType,
			End:            endTime,
			InstrumentType: instrumentType,
		}
		var orderList []OrderDetail
		orderList, err = ok.GetOrderList(ctx, requestParam)
		if err != nil {
			return nil, err
		}
		if len(orderList) == 0 {
			break
		}
		for i := range orderList {
			if req.StartTime.Equal(orderList[i].CreationTime.Time()) ||
				orderList[i].CreationTime.Time().Before(req.StartTime) ||
				endTime.Equal(orderList[i].CreationTime.Time()) {
				// reached end of orders to crawl
				break allOrders
			}
			orderSide := orderList[i].Side
			pair, err := currency.NewPairFromString(orderList[i].InstrumentID)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) > 0 {
				x := 0
				for x = range req.Pairs {
					if req.Pairs[x].Equal(pair) {
						break
					}
				}
				if !req.Pairs[x].Equal(pair) {
					continue
				}
			}
			orderStatus, err := order.StringToOrderStatus(strings.ToUpper(orderList[i].State))
			if err != nil {
				return nil, err
			}
			oType, tif, err := orderTypeFromString(orderList[i].OrderType)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				Amount:          orderList[i].Size.Float64(),
				Pair:            pair,
				Price:           orderList[i].Price.Float64(),
				ExecutedAmount:  orderList[i].FillSize.Float64(),
				RemainingAmount: orderList[i].Size.Float64() - orderList[i].FillSize.Float64(),
				Fee:             orderList[i].TransactionFee.Float64(),
				FeeAsset:        currency.NewCode(orderList[i].FeeCurrency),
				Exchange:        ok.Name,
				OrderID:         orderList[i].OrderID,
				ClientOrderID:   orderList[i].ClientOrderID,
				Type:            oType,
				Side:            orderSide,
				Status:          orderStatus,
				AssetType:       req.AssetType,
				Date:            orderList[i].CreationTime.Time(),
				LastUpdated:     orderList[i].UpdateTime.Time(),
				TimeInForce:     tif,
			})
		}
		if len(orderList) < 100 {
			// Since the we passed a limit of 0 to the method GetOrderList,
			// we expect 100 orders to be retrieved if the number of orders are more that 100.
			// If not, break out of the loop to not send another request.
			break
		}
		endTime = orderList[len(orderList)-1].CreationTime.Time()
	}
	return req.Filter(ok.Name, resp), nil
}

// GetOrderHistory retrieves account order information Can Limit response to specific order status
func (ok *Okx) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.StartTime.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	if !ok.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}
	var resp []order.Detail
	// For Spread orders.
	if req.AssetType == asset.Spread {
		oType, err := orderTypeString(req.Type, req.TimeInForce)
		if err != nil {
			return nil, err
		}
		spreadOrders, err := ok.GetCompletedSpreadOrdersLast7Days(ctx, "", oType, "", req.FromOrderID, "", req.StartTime, req.EndTime, 0)
		if err != nil {
			return nil, err
		}
		for x := range spreadOrders {
			var format currency.PairFormat
			format, err = ok.GetPairFormat(asset.Spread, true)
			if err != nil {
				return nil, err
			}
			var pair currency.Pair
			pair, err = currency.NewPairDelimiter(spreadOrders[x].SpreadID, format.Delimiter)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(spreadOrders[x].OrderType)
			if err != nil {
				return nil, err
			}
			oSide, err := order.StringToOrderSide(spreadOrders[x].Side)
			if err != nil {
				return nil, err
			}
			oStatus, err := order.StringToOrderStatus(spreadOrders[x].State)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				Price:                spreadOrders[x].Price.Float64(),
				AverageExecutedPrice: spreadOrders[x].AveragePrice.Float64(),
				Amount:               spreadOrders[x].Size.Float64(),
				ExecutedAmount:       spreadOrders[x].FillSize.Float64(),
				RemainingAmount:      spreadOrders[x].PendingFillSize.Float64(),
				Exchange:             ok.Name,
				OrderID:              spreadOrders[x].OrderID,
				ClientOrderID:        spreadOrders[x].ClientOrderID,
				Type:                 oType,
				Side:                 oSide,
				Status:               oStatus,
				AssetType:            req.AssetType,
				Date:                 spreadOrders[x].CreationTime.Time(),
				LastUpdated:          spreadOrders[x].UpdateTime.Time(),
				Pair:                 pair,
			})
		}
		return req.Filter(ok.Name, resp), nil
	}

	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	instrumentType := GetInstrumentTypeFromAssetItem(req.AssetType)
	endTime := req.EndTime
allOrders:
	for {
		orderList, err := ok.Get3MonthOrderHistory(ctx, &OrderHistoryRequestParams{
			OrderListRequestParams: OrderListRequestParams{
				InstrumentType: instrumentType,
				End:            endTime,
			},
		})
		if err != nil {
			return nil, err
		}
		if len(orderList) == 0 {
			break
		}
		for i := range orderList {
			if req.StartTime.Equal(orderList[i].CreationTime.Time()) ||
				orderList[i].CreationTime.Time().Before(req.StartTime) ||
				endTime.Equal(orderList[i].CreationTime.Time()) {
				// reached end of orders to crawl
				break allOrders
			}
			pair, err := currency.NewPairFromString(orderList[i].InstrumentID)
			if err != nil {
				return nil, err
			}
			for j := range req.Pairs {
				if !req.Pairs[j].Equal(pair) {
					continue
				}
				orderStatus, err := order.StringToOrderStatus(strings.ToUpper(orderList[i].State))
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
				}
				if orderStatus == order.Active {
					continue
				}
				oType, tif, err := orderTypeFromString(orderList[i].OrderType)
				if err != nil {
					return nil, err
				}
				orderAmount := orderList[i].Size
				if orderList[i].QuantityType == "quote_ccy" {
					// Size is quote amount.
					orderAmount /= orderList[i].AveragePrice
				}

				remainingAmount := float64(0)
				if orderStatus != order.Filled {
					remainingAmount = orderAmount.Float64() - orderList[i].AccumulatedFillSize.Float64()
				}
				resp = append(resp, order.Detail{
					Price:                orderList[i].Price.Float64(),
					AverageExecutedPrice: orderList[i].AveragePrice.Float64(),
					Amount:               orderAmount.Float64(),
					ExecutedAmount:       orderList[i].AccumulatedFillSize.Float64(),
					RemainingAmount:      remainingAmount,
					Fee:                  orderList[i].TransactionFee.Float64(),
					FeeAsset:             currency.NewCode(orderList[i].FeeCurrency),
					Exchange:             ok.Name,
					OrderID:              orderList[i].OrderID,
					ClientOrderID:        orderList[i].ClientOrderID,
					Type:                 oType,
					Side:                 orderList[i].Side,
					Status:               orderStatus,
					AssetType:            req.AssetType,
					Date:                 orderList[i].CreationTime.Time(),
					LastUpdated:          orderList[i].UpdateTime.Time(),
					Pair:                 pair,
					Cost:                 orderList[i].AveragePrice.Float64() * orderList[i].AccumulatedFillSize.Float64(),
					CostAsset:            currency.NewCode(orderList[i].RebateCurrency),
					TimeInForce:          tif,
				})
			}
		}
		if len(orderList) < 100 {
			break
		}
		endTime = orderList[len(orderList)-1].CreationTime.Time()
	}
	return req.Filter(ok.Name, resp), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (ok *Okx) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !ok.AreCredentialsValid(ctx) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return ok.GetFee(ctx, feeBuilder)
}

// ValidateAPICredentials validates current credentials used for wrapper
func (ok *Okx) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := ok.UpdateAccountInfo(ctx, assetType)
	return ok.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	req, err := ok.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	var timeSeries []kline.Candle
	switch a {
	case asset.Spread:
		candles, err := ok.GetSpreadCandlesticksHistory(ctx, req.RequestFormatted.String(), req.ExchangeInterval, start.Add(-time.Nanosecond), end, 100)
		if err != nil {
			return nil, err
		}
		timeSeries = make([]kline.Candle, len(candles))
		for x := range candles {
			timeSeries[x] = kline.Candle{
				Time:   candles[x].Timestamp.Time(),
				Open:   candles[x].Open.Float64(),
				High:   candles[x].High.Float64(),
				Low:    candles[x].Low.Float64(),
				Close:  candles[x].Close.Float64(),
				Volume: candles[x].Volume.Float64(),
			}
		}
	default:
		candles, err := ok.GetCandlesticksHistory(ctx,
			req.RequestFormatted.String(),
			req.ExchangeInterval,
			start.Add(-time.Nanosecond), // Start time not inclusive of candle.
			end,
			100)
		if err != nil {
			return nil, err
		}

		timeSeries = make([]kline.Candle, len(candles))
		for x := range candles {
			timeSeries[x] = kline.Candle{
				Time:   candles[x].OpenTime.Time(),
				Open:   candles[x].OpenPrice.Float64(),
				High:   candles[x].HighestPrice.Float64(),
				Low:    candles[x].LowestPrice.Float64(),
				Close:  candles[x].ClosePrice.Float64(),
				Volume: candles[x].Volume.Float64(),
			}
		}
	}

	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	req, err := ok.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	count := kline.TotalCandlesPerInterval(req.Start, req.End, req.ExchangeInterval)
	if count > 1440 {
		return nil,
			fmt.Errorf("candles count: %d max lookback: %d, %w",
				count, 1440, kline.ErrRequestExceedsMaxLookback)
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for y := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spread:
			candles, err := ok.GetSpreadCandlesticksHistory(ctx,
				req.RequestFormatted.String(),
				req.ExchangeInterval,
				req.RangeHolder.Ranges[y].Start.Time.Add(-time.Nanosecond), // Start time not inclusive of candle.
				req.RangeHolder.Ranges[y].End.Time,
				100)
			if err != nil {
				return nil, err
			}
			for x := range candles {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[x].Timestamp.Time(),
					Open:   candles[x].Open.Float64(),
					High:   candles[x].High.Float64(),
					Low:    candles[x].Low.Float64(),
					Close:  candles[x].Close.Float64(),
					Volume: candles[x].Volume.Float64(),
				})
			}
		default:
			candles, err := ok.GetCandlesticksHistory(ctx,
				req.RequestFormatted.String(),
				req.ExchangeInterval,
				req.RangeHolder.Ranges[y].Start.Time.Add(-time.Nanosecond), // Start time not inclusive of candle.
				req.RangeHolder.Ranges[y].End.Time,
				100)
			if err != nil {
				return nil, err
			}
			for x := range candles {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[x].OpenTime.Time(),
					Open:   candles[x].OpenPrice.Float64(),
					High:   candles[x].HighestPrice.Float64(),
					Low:    candles[x].LowestPrice.Float64(),
					Close:  candles[x].ClosePrice.Float64(),
					Volume: candles[x].Volume.Float64(),
				})
			}
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (ok *Okx) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	currencyChains, err := ok.GetFundingCurrencies(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	chains := make([]string, 0, len(currencyChains))
	for x := range currencyChains {
		if (!cryptocurrency.IsEmpty() && !strings.EqualFold(cryptocurrency.String(), currencyChains[x].Currency)) ||
			(!currencyChains[x].CanDeposit && !currencyChains[x].CanWithdraw) ||
			// Lightning network is currently not supported by transfer chains
			// as it is an invoice string which is generated per request and is
			// not a static address. TODO: Add a hook to generate a new invoice
			// string per request.
			(currencyChains[x].Chain != "" && currencyChains[x].Chain == "BTC-Lightning") {
			continue
		}
		chains = append(chains, currencyChains[x].Chain)
	}
	return chains, nil
}

// getInstrumentsForOptions returns the instruments for options asset type
func (ok *Okx) getInstrumentsForOptions(ctx context.Context) ([]Instrument, error) {
	underlyings, err := ok.GetPublicUnderlyings(ctx, instTypeOption)
	if err != nil {
		return nil, err
	}
	var insts []Instrument
	for x := range underlyings {
		var instruments []Instrument
		instruments, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: instTypeOption,
			Underlying:     underlyings[x],
		})
		if err != nil {
			return nil, err
		}
		insts = append(insts, instruments...)
	}
	return insts, nil
}

// getInstrumentsForAsset returns the instruments for an asset type
func (ok *Okx) getInstrumentsForAsset(ctx context.Context, a asset.Item) ([]Instrument, error) {
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	var instruments []Instrument
	var instType string
	var err error
	switch a {
	case asset.Options:
		instruments, err = ok.getInstrumentsForOptions(ctx)
		if err != nil {
			return nil, err
		}
		ok.instrumentsInfoMapLock.Lock()
		ok.instrumentsInfoMap[instTypeOption] = instruments
		ok.instrumentsInfoMapLock.Unlock()
		return instruments, nil
	case asset.Spot:
		instType = instTypeSpot
	case asset.Futures:
		instType = instTypeFutures
	case asset.PerpetualSwap:
		instType = instTypeSwap
	case asset.Margin:
		instType = instTypeMargin
	}

	instruments, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
		InstrumentType: instType,
	})
	if err != nil {
		return nil, err
	}
	ok.instrumentsInfoMapLock.Lock()
	ok.instrumentsInfoMap[instType] = instruments
	ok.instrumentsInfoMapLock.Unlock()
	return instruments, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (ok *Okx) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.PerpetualSwap {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w, pair required", currency.ErrCurrencyPairEmpty)
	}
	format, err := ok.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := r.Pair.Format(format)
	pairRate := fundingrate.LatestRateResponse{
		TimeChecked: time.Now(),
		Exchange:    ok.Name,
		Asset:       r.Asset,
		Pair:        fPair,
	}
	fr, err := ok.GetSingleFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	var fri time.Duration
	if len(ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
		// can infer funding rate interval from the only funding rate frequency defined
		for k := range ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
			fri = k.Duration()
		}
	}
	pairRate.LatestRate = fundingrate.Rate{
		// okx funding rate is settlement time, not when it started
		Time: fr.FundingTime.Time().Add(-fri),
		Rate: fr.FundingRate.Decimal(),
	}
	if r.IncludePredictedRate {
		pairRate.TimeOfNextRate = fr.NextFundingTime.Time()
		pairRate.PredictedUpcomingRate = fundingrate.Rate{
			Time: fr.NextFundingTime.Time().Add(-fri),
			Rate: fr.NextFundingRate.Decimal(),
		}
	}
	return []fundingrate.LatestRateResponse{pairRate}, nil
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (ok *Okx) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	requestLimit := 100
	sd := r.StartDate
	maxLookback := time.Now().Add(-ok.Features.Supports.FuturesCapabilities.MaximumFundingRateHistory)
	if r.StartDate.Before(maxLookback) {
		if r.RespectHistoryLimits {
			r.StartDate = maxLookback
		} else {
			return nil, fmt.Errorf("%w earliest date is %v", fundingrate.ErrFundingRateOutsideLimits, maxLookback)
		}
		if r.EndDate.Before(maxLookback) {
			return nil, futures.ErrGetFundingDataRequired
		}
		r.StartDate = maxLookback
	}
	format, err := ok.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := r.Pair.Format(format)
	pairRate := fundingrate.HistoricalRates{
		Exchange:  ok.Name,
		Asset:     r.Asset,
		Pair:      fPair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	// map of time indexes, allowing for easy lookup of slice index from unix time data
	mti := make(map[int64]int)
	for sd.Before(r.EndDate) {
		var frh []FundingRateResponse
		frh, err = ok.GetFundingRateHistory(ctx, fPair.String(), sd, r.EndDate, int64(requestLimit))
		if err != nil {
			return nil, err
		}
		if len(frh) == 0 {
			break
		}
		for i := range frh {
			if r.IncludePayments {
				mti[frh[i].FundingTime.Time().Unix()] = i
			}
			pairRate.FundingRates = append(pairRate.FundingRates, fundingrate.Rate{
				Time: frh[i].FundingTime.Time(),
				Rate: frh[i].FundingRate.Decimal(),
			})
		}
		if len(frh) < requestLimit {
			break
		}
		sd = frh[len(frh)-1].FundingTime.Time()
	}
	var fr *FundingRateResponse
	fr, err = ok.GetSingleFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	if fr == nil {
		return nil, fmt.Errorf("%w GetSingleFundingRate", common.ErrNilPointer)
	}
	pairRate.LatestRate = fundingrate.Rate{
		Time: fr.FundingTime.Time(),
		Rate: fr.FundingRate.Decimal(),
	}
	pairRate.TimeOfNextRate = fr.NextFundingTime.Time()
	if r.IncludePredictedRate {
		pairRate.PredictedUpcomingRate = fundingrate.Rate{
			Time: fr.NextFundingTime.Time(),
			Rate: fr.NextFundingRate.Decimal(),
		}
	}
	if r.IncludePayments {
		pairRate.PaymentCurrency = r.Pair.Base
		if !r.PaymentCurrency.IsEmpty() {
			pairRate.PaymentCurrency = r.PaymentCurrency
		}
		sd = r.StartDate
		billDetailsFunc := ok.GetBillsDetail3Months
		if time.Since(r.StartDate) < kline.OneWeek.Duration() {
			billDetailsFunc = ok.GetBillsDetailLast7Days
		}
		for sd.Before(r.EndDate) {
			var fri time.Duration
			if len(ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
				// can infer funding rate interval from the only funding rate frequency defined
				for k := range ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
					fri = k.Duration()
				}
			}
			var billDetails []BillsDetailResponse
			billDetails, err = billDetailsFunc(ctx, &BillsDetailQueryParameter{
				InstrumentType: GetInstrumentTypeFromAssetItem(r.Asset),
				Currency:       pairRate.PaymentCurrency,
				BillType:       137,
				BeginTime:      sd,
				EndTime:        r.EndDate,
				Limit:          int64(requestLimit),
			})
			if err != nil {
				return nil, err
			}
			for i := range billDetails {
				if index, okay := mti[billDetails[i].Timestamp.Time().Truncate(fri).Unix()]; okay {
					pairRate.FundingRates[index].Payment = billDetails[i].ProfitAndLoss.Decimal()
					continue
				}
			}
			if len(billDetails) < requestLimit {
				break
			}
			sd = billDetails[len(billDetails)-1].Timestamp.Time()
		}

		for i := range pairRate.FundingRates {
			pairRate.PaymentSum = pairRate.PaymentSum.Add(pairRate.FundingRates[i].Payment)
		}
	}
	return &pairRate, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (ok *Okx) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.PerpetualSwap, nil
}

// SetMarginType sets the default margin type for when opening a new position
// okx allows this to be set with an order, however this sets a default
func (ok *Okx) SetMarginType(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type) error {
	return fmt.Errorf("%w margin type is set per order", common.ErrFunctionNotSupported)
}

// SetCollateralMode sets the collateral type for your account
func (ok *Okx) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// GetCollateralMode returns the collateral type for your account
func (ok *Okx) GetCollateralMode(ctx context.Context, item asset.Item) (collateral.Mode, error) {
	if !ok.SupportsAsset(item) {
		return 0, fmt.Errorf("%w: %v", asset.ErrNotSupported, item)
	}
	cfg, err := ok.GetAccountConfiguration(ctx)
	if err != nil {
		return 0, err
	}
	switch cfg.AccountLevel {
	case 1:
		if item != asset.Spot {
			return 0, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
		}
		fallthrough
	case 2:
		return collateral.SpotFuturesMode, nil
	case 3:
		return collateral.MultiMode, nil
	case 4:
		return collateral.PortfolioMode, nil
	default:
		return collateral.UnknownMode, fmt.Errorf("%w %v", order.ErrCollateralInvalid, cfg.AccountLevel)
	}
}

// ChangePositionMargin will modify a position/currencies margin parameters
func (ok *Okx) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionChangeRequest", common.ErrNilPointer)
	}
	if !ok.SupportsAsset(req.Asset) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.Asset)
	}
	if req.NewAllocatedMargin == 0 {
		return nil, fmt.Errorf("%w %v %v", margin.ErrNewAllocatedMarginRequired, req.Asset, req.Pair)
	}
	if req.OriginalAllocatedMargin == 0 {
		return nil, margin.ErrOriginalPositionMarginRequired
	}
	if req.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, req.MarginType)
	}
	pairFormat, err := ok.GetPairFormat(req.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := req.Pair.Format(pairFormat)
	marginType := "add"
	amt := req.NewAllocatedMargin - req.OriginalAllocatedMargin
	if req.NewAllocatedMargin < req.OriginalAllocatedMargin {
		marginType = "reduce"
		amt = req.OriginalAllocatedMargin - req.NewAllocatedMargin
	}
	if req.MarginSide == "" {
		req.MarginSide = "net"
	}
	r := &IncreaseDecreaseMarginInput{
		InstrumentID:      fPair.String(),
		PositionSide:      req.MarginSide,
		MarginBalanceType: marginType,
		Amount:            amt,
	}

	if req.Asset == asset.Margin {
		r.Currency = req.Pair.Base.Item.Symbol
	}

	resp, err := ok.IncreaseDecreaseMargin(ctx, r)
	if err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        ok.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		AllocatedMargin: resp.Amount.Float64(),
		MarginType:      req.MarginType,
	}, nil
}

// GetFuturesPositionSummary returns position summary details for an active position
func (ok *Okx) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	if !ok.SupportsAsset(req.Asset) || !req.Asset.IsFutures() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	fPair, err := ok.FormatExchangeCurrency(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	instrumentType := GetInstrumentTypeFromAssetItem(req.Asset)

	var contracts []futures.Contract
	contracts, err = ok.GetFuturesContractDetails(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	multiplier := 1.0
	var contractSettlementType futures.ContractSettlementType
	for i := range contracts {
		if !contracts[i].Name.Equal(fPair) {
			continue
		}
		multiplier = contracts[i].Multiplier
		contractSettlementType = contracts[i].SettlementType
		break
	}

	positionSummaries, err := ok.GetPositions(ctx, instrumentType, fPair.String(), "")
	if err != nil {
		return nil, err
	}
	var positionSummary *AccountPosition
	for i := range positionSummaries {
		if positionSummaries[i].QuantityOfPosition.Float64() <= 0 {
			continue
		}
		positionSummary = &positionSummaries[i]
		break
	}
	if positionSummary == nil {
		return nil, fmt.Errorf("%w, received '%v', no positions found", errOnlyOneResponseExpected, len(positionSummaries))
	}
	marginMode := margin.Isolated
	if positionSummary.MarginMode == TradeModeCross {
		marginMode = margin.Multi
	}

	acc, err := ok.AccountBalance(ctx, currency.EMPTYCODE)
	if err != nil {
		return nil, err
	}
	if len(acc) != 1 {
		return nil, fmt.Errorf("%w, received '%v'", errOnlyOneResponseExpected, len(acc))
	}
	var freeCollateral, totalCollateral, equityOfCurrency, frozenBalance,
		availableEquity, cashBalance, discountEquity,
		equityUSD, totalEquity, isolatedEquity, isolatedLiabilities,
		isolatedUnrealisedProfit, notionalLeverage,
		strategyEquity decimal.Decimal

	for i := range acc[0].Details {
		if !acc[0].Details[i].Currency.Equal(positionSummary.Currency) {
			continue
		}
		freeCollateral = acc[0].Details[i].AvailableBalance.Decimal()
		frozenBalance = acc[0].Details[i].FrozenBalance.Decimal()
		totalCollateral = freeCollateral.Add(frozenBalance)
		equityOfCurrency = acc[0].Details[i].EquityOfCurrency.Decimal()
		availableEquity = acc[0].Details[i].AvailableEquity.Decimal()
		cashBalance = acc[0].Details[i].CashBalance.Decimal()
		discountEquity = acc[0].Details[i].DiscountEquity.Decimal()
		equityUSD = acc[0].Details[i].EquityUsd.Decimal()
		totalEquity = acc[0].Details[i].TotalEquity.Decimal()
		isolatedEquity = acc[0].Details[i].IsoEquity.Decimal()
		isolatedLiabilities = acc[0].Details[i].IsolatedLiabilities.Decimal()
		isolatedUnrealisedProfit = acc[0].Details[i].IsoUpl.Decimal()
		notionalLeverage = acc[0].Details[i].NotionalLever.Decimal()
		strategyEquity = acc[0].Details[i].StrategyEquity.Decimal()

		break
	}
	collateralMode, err := ok.GetCollateralMode(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	return &futures.PositionSummary{
		Pair:            req.Pair,
		Asset:           req.Asset,
		MarginType:      marginMode,
		CollateralMode:  collateralMode,
		Currency:        positionSummary.Currency,
		AvailableEquity: availableEquity,
		CashBalance:     cashBalance,
		DiscountEquity:  discountEquity,
		EquityUSD:       equityUSD,

		IsolatedEquity:               isolatedEquity,
		IsolatedLiabilities:          isolatedLiabilities,
		IsolatedUPL:                  isolatedUnrealisedProfit,
		NotionalLeverage:             notionalLeverage,
		TotalEquity:                  totalEquity,
		StrategyEquity:               strategyEquity,
		IsolatedMargin:               positionSummary.Margin.Decimal(),
		NotionalSize:                 positionSummary.NotionalUsd.Decimal(),
		Leverage:                     positionSummary.Leverage.Decimal(),
		MaintenanceMarginRequirement: positionSummary.MaintenanceMarginRequirement.Decimal(),
		InitialMarginRequirement:     positionSummary.InitialMarginRequirement.Decimal(),
		EstimatedLiquidationPrice:    positionSummary.LiquidationPrice.Decimal(),
		CollateralUsed:               positionSummary.Margin.Decimal(),
		MarkPrice:                    positionSummary.MarkPrice.Decimal(),
		CurrentSize:                  positionSummary.QuantityOfPosition.Decimal().Mul(decimal.NewFromFloat(multiplier)),
		ContractSize:                 positionSummary.QuantityOfPosition.Decimal(),
		ContractMultiplier:           decimal.NewFromFloat(multiplier),
		ContractSettlementType:       contractSettlementType,
		AverageOpenPrice:             positionSummary.AveragePrice.Decimal(),
		UnrealisedPNL:                positionSummary.UPNL.Decimal(),
		MaintenanceMarginFraction:    positionSummary.MarginRatio.Decimal(),
		FreeCollateral:               freeCollateral,
		TotalCollateral:              totalCollateral,
		FrozenBalance:                frozenBalance,
		EquityOfCurrency:             equityOfCurrency,
	}, nil
}

// GetFuturesPositionOrders returns the orders for futures positions
func (ok *Okx) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if !ok.SupportsAsset(req.Asset) || !req.Asset.IsFutures() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	if time.Since(req.StartDate) > ok.Features.Supports.MaximumOrderHistory {
		if req.RespectOrderHistoryLimits {
			req.StartDate = time.Now().Add(-ok.Features.Supports.MaximumOrderHistory)
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, time.Now().Add(-ok.Features.Supports.MaximumOrderHistory))
		}
	}
	err := common.StartEndTimeCheck(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(req.Pairs))
	var contracts []futures.Contract
	contracts, err = ok.GetFuturesContractDetails(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	contractsMap := make(map[currency.Pair]*futures.Contract)
	for i := range contracts {
		contractsMap[contracts[i].Name] = &contracts[i]
	}
	for i := range req.Pairs {
		fPair, err := ok.FormatExchangeCurrency(req.Pairs[i], req.Asset)
		if err != nil {
			return nil, err
		}
		instrumentType := GetInstrumentTypeFromAssetItem(req.Asset)

		contract, exist := contractsMap[fPair]
		if !exist {
			return nil, fmt.Errorf("%w %v", futures.ErrContractNotSupported, fPair)
		}
		multiplier := contract.Multiplier
		contractSettlementType := contract.SettlementType

		resp[i] = futures.PositionResponse{
			Pair:                   req.Pairs[i],
			Asset:                  req.Asset,
			ContractSettlementType: contractSettlementType,
		}

		var positions []OrderDetail
		historyRequest := &OrderHistoryRequestParams{
			OrderListRequestParams: OrderListRequestParams{
				InstrumentType: instrumentType,
				InstrumentID:   fPair.String(),
				Start:          req.StartDate,
				End:            req.EndDate,
			},
		}
		if time.Since(req.StartDate) <= time.Hour*24*7 {
			positions, err = ok.Get7DayOrderHistory(ctx, historyRequest)
		} else {
			positions, err = ok.Get3MonthOrderHistory(ctx, historyRequest)
		}
		if err != nil {
			return nil, err
		}
		for j := range positions {
			if fPair.String() != positions[j].InstrumentID {
				continue
			}
			orderStatus, err := order.StringToOrderStatus(strings.ToUpper(positions[j].State))
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
			}
			oType, tif, err := orderTypeFromString(positions[j].OrderType)
			if err != nil {
				return nil, err
			}
			orderAmount := positions[j].Size
			if positions[j].QuantityType == "quote_ccy" {
				// Size is quote amount.
				orderAmount /= positions[j].AveragePrice
			}

			remainingAmount := float64(0)
			if orderStatus != order.Filled {
				remainingAmount = orderAmount.Float64() - positions[j].AccumulatedFillSize.Float64()
			}
			cost := positions[j].AveragePrice.Float64() * positions[j].AccumulatedFillSize.Float64()
			if multiplier != 1 {
				cost *= multiplier
			}
			resp[i].Orders = append(resp[i].Orders, order.Detail{
				Price:                positions[j].Price.Float64(),
				AverageExecutedPrice: positions[j].AveragePrice.Float64(),
				Amount:               orderAmount.Float64() * multiplier,
				ContractAmount:       orderAmount.Float64(),
				ExecutedAmount:       positions[j].AccumulatedFillSize.Float64(),
				RemainingAmount:      remainingAmount,
				Fee:                  positions[j].TransactionFee.Float64(),
				FeeAsset:             currency.NewCode(positions[j].FeeCurrency),
				Exchange:             ok.Name,
				OrderID:              positions[j].OrderID,
				ClientOrderID:        positions[j].ClientOrderID,
				Type:                 oType,
				Side:                 positions[j].Side,
				Status:               orderStatus,
				AssetType:            req.Asset,
				Date:                 positions[j].CreationTime.Time(),
				LastUpdated:          positions[j].UpdateTime.Time(),
				Pair:                 req.Pairs[i],
				Cost:                 cost,
				CostAsset:            currency.NewCode(positions[j].RebateCurrency),
				TimeInForce:          tif,
			})
		}
	}
	return resp, nil
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (ok *Okx) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, orderSide order.Side) error {
	posSide := "net"
	switch item {
	case asset.Futures, asset.PerpetualSwap:
		if marginType == margin.Isolated {
			switch {
			case orderSide == order.UnknownSide:
				return order.ErrSideIsInvalid
			case orderSide.IsLong():
				posSide = "long"
			case orderSide.IsShort():
				posSide = "short"
			default:
				return fmt.Errorf("%w %v requires long/short", order.ErrSideIsInvalid, orderSide)
			}
		}
		fallthrough
	case asset.Margin, asset.Options:
		instrumentID, err := ok.FormatSymbol(pair, item)
		if err != nil {
			return err
		}

		marginMode := ok.marginTypeToString(marginType)
		_, err = ok.SetLeverageRate(ctx, &SetLeverageInput{
			Leverage:     amount,
			MarginMode:   marginMode,
			InstrumentID: instrumentID,
			PositionSide: posSide,
		})
		return err
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (ok *Okx) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, orderSide order.Side) (float64, error) {
	var inspectLeverage bool
	switch item {
	case asset.Futures, asset.PerpetualSwap:
		if marginType == margin.Isolated {
			switch {
			case orderSide == order.UnknownSide:
				return 0, order.ErrSideIsInvalid
			case orderSide.IsLong(), orderSide.IsShort():
				inspectLeverage = true
			default:
				return 0, fmt.Errorf("%w '%v', requires long/short", order.ErrSideIsInvalid, orderSide)
			}
		}
		fallthrough
	case asset.Margin, asset.Options:
		instrumentID, err := ok.FormatSymbol(pair, item)
		if err != nil {
			return -1, err
		}
		marginMode := ok.marginTypeToString(marginType)
		lev, err := ok.GetLeverageRate(ctx, instrumentID, marginMode, currency.EMPTYCODE)
		if err != nil {
			return -1, err
		}
		if len(lev) == 0 {
			return -1, fmt.Errorf("%w %v %v %s", futures.ErrPositionNotFound, item, pair, marginType)
		}
		if inspectLeverage {
			for i := range lev {
				if lev[i].PositionSide == orderSide.Lower() {
					return lev[i].Leverage.Float64(), nil
				}
			}
		}

		// leverage is the same across positions
		return lev[0].Leverage.Float64(), nil
	default:
		return -1, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetFuturesContractDetails returns details about futures contracts
func (ok *Okx) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	switch item {
	case asset.Futures, asset.PerpetualSwap:
		instType := GetInstrumentTypeFromAssetItem(item)
		result, err := ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: instType,
		})
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(result))
		for i := range result {
			var (
				underlying             currency.Pair
				settleCurr             currency.Code
				contractSettlementType futures.ContractSettlementType
			)

			if result[i].State == "live" {
				underlying, err = currency.NewPairFromString(result[i].Underlying)
				if err != nil {
					return nil, err
				}

				settleCurr = currency.NewCode(result[i].SettlementCurrency)

				contractSettlementType = futures.Linear
				if result[i].SettlementCurrency == result[i].BaseCurrency {
					contractSettlementType = futures.Inverse
				}
			}

			var ct futures.ContractType
			if item == asset.PerpetualSwap {
				ct = futures.Perpetual
			} else {
				switch result[i].Alias {
				case "this_week", "next_week":
					ct = futures.Weekly
				case "quarter", "next_quarter":
					ct = futures.Quarterly
				}
			}

			resp[i] = futures.Contract{
				Exchange:       ok.Name,
				Name:           result[i].InstrumentID,
				Underlying:     underlying,
				Asset:          item,
				StartDate:      result[i].ListTime.Time(),
				EndDate:        result[i].ExpTime.Time(),
				IsActive:       result[i].State == "live",
				Status:         result[i].State,
				Type:           ct,
				SettlementType: contractSettlementType,
				MarginCurrency: settleCurr,
				Multiplier:     result[i].ContractValue.Float64(),
				MaxLeverage:    result[i].MaxLeverage.Float64(),
			}

			if !settleCurr.IsEmpty() {
				resp[i].SettlementCurrencies = currency.Currencies{settleCurr}
			}
		}
		return resp, nil
	case asset.Spread:
		results, err := ok.GetPublicSpreads(ctx, "", "", "", "")
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(results))
		for s := range results {
			contractSettlementType, err := futures.StringToContractSettlementType(results[s].SpreadType)
			if err != nil {
				return nil, err
			}
			resp[s] = futures.Contract{
				Exchange:       ok.Name,
				Name:           results[s].SpreadID,
				Asset:          asset.Spread,
				StartDate:      results[s].ListTime.Time(),
				EndDate:        results[s].ExpTime.Time(),
				IsActive:       results[s].State == "live",
				Status:         results[s].State,
				Type:           futures.LongDated,
				SettlementType: contractSettlementType,
				MarginCurrency: currency.NewCode(results[s].QuoteCurrency),
			}
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (ok *Okx) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		switch k[i].Asset {
		case asset.Futures, asset.PerpetualSwap, asset.Options:
		default:
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) != 1 {
		var resp []futures.OpenInterest
		// TODO: Options support
		instTypes := map[string]asset.Item{
			instTypeSwap:    asset.PerpetualSwap,
			instTypeFutures: asset.Futures,
			instTypeOption:  asset.Options,
		}
		for instType, v := range instTypes {
			var oid []OpenInterest
			var err error
			switch instType {
			case instTypeOption:
				var underlyings []string
				underlyings, err = ok.GetPublicUnderlyings(ctx, instTypeOption)
				if err != nil {
					return nil, err
				}
				for u := range underlyings {
					var incOID []OpenInterest
					incOID, err = ok.GetOpenInterestData(ctx, instType, underlyings[u], "", "")
					if err != nil {
						return nil, err
					}
					oid = append(oid, incOID...)
				}
			case instTypeSwap,
				instTypeFutures:
				oid, err = ok.GetOpenInterestData(ctx, instType, "", "", "")
				if err != nil {
					return nil, err
				}
			}
			for j := range oid {
				var isEnabled bool
				var p currency.Pair
				p, isEnabled, err = ok.MatchSymbolCheckEnabled(oid[j].InstrumentID, v, true)
				if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
					return nil, err
				}
				if !isEnabled {
					continue
				}
				var appendData bool
				for j := range k {
					if k[j].Pair().Equal(p) {
						appendData = true
						break
					}
				}
				if len(k) > 0 && !appendData {
					continue
				}
				resp = append(resp, futures.OpenInterest{
					Key: key.ExchangePairAsset{
						Exchange: ok.Name,
						Base:     p.Base.Item,
						Quote:    p.Quote.Item,
						Asset:    v,
					},
					OpenInterest: oid[j].OpenInterest.Float64(),
				})
			}
		}
		return resp, nil
	}
	resp := make([]futures.OpenInterest, 1)
	instTypes := map[asset.Item]string{
		asset.PerpetualSwap: "SWAP",
		asset.Futures:       "FUTURES",
	}
	pFmt, err := ok.FormatSymbol(k[0].Pair(), k[0].Asset)
	if err != nil {
		return nil, err
	}
	var oid []OpenInterest
	switch instTypes[k[0].Asset] {
	case instTypeOption:
		var underlyings []string
		underlyings, err = ok.GetPublicUnderlyings(ctx, instTypeOption)
		if err != nil {
			return nil, err
		}
		for u := range underlyings {
			var incOID []OpenInterest
			incOID, err = ok.GetOpenInterestData(ctx, instTypes[k[0].Asset], underlyings[u], "", "")
			if err != nil {
				return nil, err
			}
			oid = append(oid, incOID...)
		}
	case instTypeSwap, instTypeFutures:
		oid, err = ok.GetOpenInterestData(ctx, instTypes[k[0].Asset], "", "", pFmt)
		if err != nil {
			return nil, err
		}
	}
	for i := range oid {
		p, isEnabled, err := ok.MatchSymbolCheckEnabled(oid[i].InstrumentID, k[0].Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		resp[0] = futures.OpenInterest{
			Key: key.ExchangePairAsset{
				Exchange: ok.Name,
				Base:     p.Base.Item,
				Quote:    p.Quote.Item,
				Asset:    k[0].Asset,
			},
			OpenInterest: oid[i].OpenInterest.Float64(),
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (ok *Okx) GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := ok.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	switch a {
	case asset.Spot:
		return baseURL + "trade-spot/" + cp.Lower().String(), nil
	case asset.Margin:
		return baseURL + "trade-margin/" + cp.Lower().String(), nil
	case asset.PerpetualSwap:
		return baseURL + "trade-swap/" + cp.Lower().String(), nil
	case asset.Options:
		return baseURL + "trade-option/" + cp.Base.Lower().String() + "-usd", nil
	case asset.Spread:
		return baseURL, nil
	case asset.Futures:
		cp, err = ok.FormatExchangeCurrency(cp, a)
		if err != nil {
			return "", err
		}
		insts, err := ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: instTypeFutures,
			InstrumentID:   cp.String(),
		})
		if err != nil {
			return "", err
		}
		if len(insts) != 1 {
			return "", fmt.Errorf("%w response len: %v currency expected: %v", errOnlyOneResponseExpected, len(insts), cp)
		}
		var ct string
		switch insts[0].Alias {
		case "this_week":
			ct = "-weekly"
		case "next_week":
			ct = "-biweekly"
		case "this_month":
			ct = "-monthly"
		case "next_month":
			ct = "-bimonthly"
		case "quarter":
			ct = "-quarterly"
		case "next_quarter":
			ct = "-biquarterly"
		}
		return baseURL + "trade-futures/" + strings.ToLower(insts[0].Underlying) + ct, nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
