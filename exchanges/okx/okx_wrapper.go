package okx

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	websocketResponseMaxLimit = time.Second * 3
	instrumentStateLive       = "live"
)

// SetDefaults sets the basic defaults for Okx
func (e *Exchange) SetDefaults() {
	e.Name = "Okx"
	e.Enabled = true
	e.Verbose = true

	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresClientID = true

	e.instrumentsInfoMap = make(map[string][]Instrument)
	e.instrumentIDCodeMap = make(map[string]uint64)

	cpf := &currency.PairFormat{
		Delimiter: currency.DashDelimiter,
		Uppercase: true,
	}

	// In this exchange, we represent deliverable futures contracts as 'FUTURES'/asset.Futures and perpetual futures as 'SWAP'/asset.PerpetualSwap
	err := e.SetGlobalPairsManager(cpf, cpf, asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options, asset.Margin, asset.Spread)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// TODO: Remove when spread/business websocket templates across connections are completed
	if err := e.DisableAssetWebsocketSupport(asset.Spread); err != nil {
		log.Errorf(log.ExchangeSys, "%s error disabling %q asset websocket support: %s", e.Name, asset.Spread.String(), err)
	}

	// Fill out the capabilities/features that the exchange supports
	e.Features = exchange.Features{
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
				SubmitOrders:           true,
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
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(rateLimits))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   apiURL,
		exchange.WebsocketSpot:              apiWebsocketPublicURL,
		exchange.WebsocketPrivate:           apiWebsocketPrivateURL,
		exchange.WebsocketSpotSupplementary: okxBusinessWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = websocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = websocketResponseMaxLimit
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *Exchange) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	if err := e.SetupDefaults(exch); err != nil {
		return err
	}

	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		MaxWebsocketSubscriptionsPerConnection: 30, // see: https://www.okx.com/docs-v5/en/#overview-websocket-connection-count-limit
		RateLimitDefinitions:                   rateLimits,
		UseMultiConnectionManagement:           true,
	}); err != nil {
		return err
	}

	wsPublic, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                       wsPublic,
		ResponseCheckTimeout:      exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:          websocketResponseMaxLimit,
		ConnectionRateLimiter:     func() *request.RateLimiterWithWeight { return request.NewRateLimitWithWeight(time.Hour, 480, 1) }, // see: https://www.okx.com/docs-v5/en/#overview-websocket-connect
		Connector:                 e.wsConnect,
		GenerateSubscriptions:     func() (subscription.List, error) { return e.generateSubscriptions(true) },
		Handler:                   e.wsHandleData,
		Subscriber:                e.Subscribe,
		TrackOnExistingConnection: e.trackEquivalentSubscriptionsOnExistingConnection,
		Unsubscriber:              e.Unsubscribe,
	}); err != nil {
		return err
	}

	wsPrivate, err := e.API.Endpoints.GetURL(exchange.WebsocketPrivate)
	if err != nil {
		return err
	}

	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsPrivate,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      websocketResponseMaxLimit,
		ConnectionRateLimiter: func() *request.RateLimiterWithWeight { return request.NewRateLimitWithWeight(time.Hour, 480, 1) }, // see: https://www.okx.com/docs-v5/en/#overview-websocket-connect
		Connector:             e.wsConnect,
		GenerateSubscriptions: func() (subscription.List, error) { return e.generateSubscriptions(false) },
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		Handler:               e.wsHandleData,
		Authenticate:          e.wsAuthenticateConnection,
		MessageFilter:         privateConnection,
	}); err != nil {
		return err
	}

	wsBusiness, err := e.API.Endpoints.GetURL(exchange.WebsocketSpotSupplementary)
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsBusiness,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      websocketResponseMaxLimit,
		ConnectionRateLimiter: func() *request.RateLimiterWithWeight { return request.NewRateLimitWithWeight(time.Hour, 480, 1) }, // see: https://www.okx.com/docs-v5/en/#overview-websocket-connect
		Connector:             e.wsConnect,
		GenerateSubscriptions: e.GenerateDefaultBusinessSubscriptions,
		Subscriber:            e.BusinessSubscribe,
		Unsubscriber:          e.BusinessUnsubscribe,
		Handler:               e.wsHandleData,
		Authenticate:          e.wsAuthenticateConnection,
		MessageFilter:         businessConnection,
	})
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	t, err := e.GetSystemTime(ctx)
	return t.Time(), err
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Options, asset.Futures, asset.Spot, asset.PerpetualSwap, asset.Margin:
		format, err := e.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		insts, err := e.getInstrumentsForAsset(ctx, a)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(insts))
		for x := range insts {
			if insts[x].State != instrumentStateLive {
				continue
			}
			pairs = append(pairs, insts[x].InstrumentID.Format(format))
		}
		return pairs, nil
	case asset.Spread:
		format, err := e.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		spreadInstruments, err := e.GetPublicSpreads(ctx, "", "", "", instrumentStateLive)
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
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assetTypes := e.GetAssetTypes(true)
	for i := range assetTypes {
		pairs, err := e.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return fmt.Errorf("%w for asset %v", err, assetTypes[i])
		}
		if err := e.UpdatePairs(pairs, assetTypes[i], false); err != nil {
			return fmt.Errorf("%w for asset %v", err, assetTypes[i])
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot, asset.Margin, asset.Options,
		asset.PerpetualSwap, asset.Futures:
		insts, err := e.getInstrumentsForAsset(ctx, a)
		if err != nil {
			return err
		}
		return e.loadInstrumentOrderExecutionLimits(a, insts)
	case asset.Spread:
		insts, err := e.GetPublicSpreads(ctx, "", "", "", instrumentStateLive)
		if err != nil {
			return err
		}
		if len(insts) == 0 {
			return common.ErrNoResponse
		}
		l := make([]limits.MinMaxLevel, len(insts))
		for i := range insts {
			l[i] = limits.MinMaxLevel{
				Key:                    key.NewExchangeAssetPair(e.Name, a, insts[i].SpreadID),
				PriceStepIncrementSize: insts[i].MinSize.Float64(),
				MinimumBaseAmount:      insts[i].MinSize.Float64(),
				QuoteStepIncrementSize: insts[i].TickSize.Float64(),
			}
		}
		return limits.Load(l)
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

func (e *Exchange) loadInstrumentOrderExecutionLimits(a asset.Item, insts []Instrument) error {
	if len(insts) == 0 {
		return common.ErrNoResponse
	}
	l := make([]limits.MinMaxLevel, 0, len(insts))
	for i := range insts {
		if insts[i].State != instrumentStateLive || insts[i].InstrumentID.IsEmpty() {
			continue
		}
		l = append(l, limits.MinMaxLevel{
			Key:                    key.NewExchangeAssetPair(e.Name, a, insts[i].InstrumentID),
			PriceStepIncrementSize: insts[i].TickSize.Float64(),
			MinimumBaseAmount:      insts[i].MinimumOrderSize.Float64(),
		})
	}
	if len(l) == 0 {
		return common.ErrInvalidResponse
	}
	return limits.Load(l)
}

// tickerVolumes maps a ticker's two volume figures onto base and quote. For spot and margin
// vol24h is the base currency and volCcy24h the quote; the two swap over on the derivative
// instrument types, where volCcy24h is the base currency and vol24h counts contracts. A contract
// is neither currency, being worth ctVal of ctValCcy and the ticker carrying neither, so a
// derivative reports no quote volume rather than a count the field name would misdescribe. Some
// futures instruments serve volCcy24h badly and report no base volume either. An asset type the
// exchange does not price this way reports neither, so a new one must be added here to carry volume
func tickerVolumes(tick *TickerResponse, a asset.Item) (baseVolume, quoteVolume float64) {
	switch a {
	case asset.Spot, asset.Margin:
		return tick.TradingVolume24HourInContract.Float64(), tick.TradingVolume24HourInCurrency.Float64()
	case asset.PerpetualSwap, asset.Futures, asset.Options:
		return tick.TradingVolume24HourInCurrency.Float64(), 0
	}
	return 0, 0
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	p, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	if a == asset.Spread {
		spreadTicker, err := e.GetPublicSpreadTickers(ctx, p.String())
		if err != nil {
			return nil, err
		}

		if len(spreadTicker) == 0 {
			return nil, fmt.Errorf("no ticker data for %s", p.String())
		}

		if err := ticker.ProcessTicker(&ticker.Price{
			Last:         spreadTicker[0].Last.Float64(),
			High:         spreadTicker[0].HighestPrice24Hour.Float64(),
			Low:          spreadTicker[0].LowestPrice24Hour.Float64(),
			Bid:          spreadTicker[0].BidPrice.Float64(),
			BidSize:      spreadTicker[0].BidSize.Float64(),
			Ask:          spreadTicker[0].AskPrice.Float64(),
			AskSize:      spreadTicker[0].AskSize.Float64(),
			Open:         spreadTicker[0].OpenPrice24Hour.Float64(),
			LastUpdated:  spreadTicker[0].Timestamp.Time(),
			Pair:         p,
			AssetType:    a,
			ExchangeName: e.Name,
		}); err != nil {
			return nil, err
		}
	} else {
		mdata, err := e.GetTicker(ctx, p.String())
		if err != nil {
			return nil, err
		}
		baseVolume, quoteVolume := tickerVolumes(mdata, a)
		if err := ticker.ProcessTicker(&ticker.Price{
			Last:         mdata.LastTradePrice.Float64(),
			High:         mdata.HighestPrice24Hour.Float64(),
			Low:          mdata.LowestPrice24Hour.Float64(),
			Bid:          mdata.BestBidPrice.Float64(),
			BidSize:      mdata.BestBidSize.Float64(),
			Ask:          mdata.BestAskPrice.Float64(),
			AskSize:      mdata.BestAskSize.Float64(),
			BaseVolume:   baseVolume,
			QuoteVolume:  quoteVolume,
			Open:         mdata.OpenPrice24Hour.Float64(),
			LastUpdated:  mdata.TickerDataGenerationTime.Time(),
			Pair:         p,
			ExchangeName: e.Name,
			AssetType:    a,
		}); err != nil {
			return nil, err
		}
	}

	return ticker.GetTicker(e.Name, p, a)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spread:
		format, err := e.GetPairFormat(asset.Spread, true)
		if err != nil {
			return err
		}
		pairs, err := e.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}
		for y := range pairs {
			var spreadTickers []SpreadTicker
			spreadTickers, err = e.GetPublicSpreadTickers(ctx, format.Format(pairs[y]))
			if err != nil {
				return err
			}
			for x := range spreadTickers {
				pair, err := currency.NewPairDelimiter(spreadTickers[x].SpreadID, format.Delimiter)
				if err != nil {
					return err
				}
				// the store overwrites wholesale, so every field UpdateTicker sets must be set
				// here too; sprd/ticker sends no 24h figures today but market/sprd-ticker does.
				// vol24h is deliberately not mapped: OKX reports it in USD on an inverse spread,
				// so it is not a base volume, and SpreadTicker carries no sprdType to separate them
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         spreadTickers[x].Last.Float64(),
					Bid:          spreadTickers[x].BidPrice.Float64(),
					BidSize:      spreadTickers[x].BidSize.Float64(),
					Ask:          spreadTickers[x].AskPrice.Float64(),
					AskSize:      spreadTickers[x].AskSize.Float64(),
					High:         spreadTickers[x].HighestPrice24Hour.Float64(),
					Low:          spreadTickers[x].LowestPrice24Hour.Float64(),
					Open:         spreadTickers[x].OpenPrice24Hour.Float64(),
					LastUpdated:  spreadTickers[x].Timestamp.Time(),
					Pair:         pair,
					ExchangeName: e.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return err
				}
			}
		}
	case asset.Spot, asset.PerpetualSwap, asset.Futures, asset.Options, asset.Margin:
		pairs, err := e.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}

		instrumentType := GetInstrumentTypeFromAssetItem(assetType)
		if assetType == asset.Margin {
			instrumentType = instTypeSpot
		}
		ticks, err := e.GetTickers(ctx, instrumentType, "", "")
		if err != nil {
			return err
		}

		for y := range ticks {
			pair, err := e.GetPairFromInstrumentID(ticks[y].InstrumentID.String())
			if err != nil {
				return err
			}
			for i := range pairs {
				pairFmt, err := e.FormatExchangeCurrency(pairs[i], assetType)
				if err != nil {
					return err
				}
				if !pair.Equal(pairFmt) {
					continue
				}
				baseVolume, quoteVolume := tickerVolumes(&ticks[y], assetType)
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         ticks[y].LastTradePrice.Float64(),
					High:         ticks[y].HighestPrice24Hour.Float64(),
					Low:          ticks[y].LowestPrice24Hour.Float64(),
					Bid:          ticks[y].BestBidPrice.Float64(),
					BidSize:      ticks[y].BestBidSize.Float64(),
					Ask:          ticks[y].BestAskPrice.Float64(),
					AskSize:      ticks[y].BestAskSize.Float64(),
					BaseVolume:   baseVolume,
					QuoteVolume:  quoteVolume,
					Open:         ticks[y].OpenPrice24Hour.Float64(),
					LastUpdated:  ticks[y].TickerDataGenerationTime.Time(),
					Pair:         pairFmt,
					ExchangeName: e.Name,
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
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
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
		pairFormat, err = e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		spreadOrderbook, err = e.GetPublicSpreadOrderBooks(ctx, pairFormat.Format(pair), 50)
		if err != nil {
			return nil, err
		}
		for y := range spreadOrderbook {
			book := &orderbook.Book{
				Exchange:          e.Name,
				Pair:              pair,
				Asset:             assetType,
				ValidateOrderbook: e.ValidateOrderbook,
				LastUpdated:       spreadOrderbook[y].Timestamp.Time(),
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
		err = e.CurrencyPairs.IsAssetEnabled(assetType)
		if err != nil {
			return nil, err
		}
		var instrumentID string
		pairFormat, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		if !pair.IsPopulated() {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		instrumentID = pairFormat.Format(pair)
		book := &orderbook.Book{
			Exchange:          e.Name,
			Pair:              pair,
			Asset:             assetType,
			ValidateOrderbook: e.ValidateOrderbook,
		}
		var orderBookD *OrderBookResponseDetail
		orderBookD, err = e.GetOrderBookDepth(ctx, instrumentID, 400)
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
		book.LastUpdated = orderBookD.GenerationTimestamp.Time()
		err = book.Process()
		if err != nil {
			return book, err
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	resp, err := e.AccountBalance(ctx, currency.EMPTYCODE)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for i := range resp {
		for j := range resp[i].Details {
			subAccts[0].Balances.Set(resp[i].Details[j].Currency, accounts.Balance{
				Total: resp[i].Details[j].EquityOfCurrency.Float64(),
				Hold:  resp[i].Details[j].FrozenBalance.Float64(),
				Free:  resp[i].Details[j].AvailableBalance.Float64(),
			})
		}
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	depositHistories, err := e.GetCurrencyDepositHistory(ctx, currency.EMPTYCODE, "", "", "", "", time.Time{}, time.Time{}, -1, 0)
	if err != nil {
		return nil, err
	}

	withdrawalHistories, err := e.GetWithdrawalHistory(ctx, currency.EMPTYCODE, "", "", "", "", time.Time{}, time.Time{}, -5)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(depositHistories)+len(withdrawalHistories))
	for x := range depositHistories {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
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
			ExchangeName:    e.Name,
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
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.GetWithdrawalHistory(ctx, c, "", "", "", "", time.Time{}, time.Time{}, -5)
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
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	format, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Spread:
		var spreadTrades []SpreadPublicTradeItem
		spreadTrades, err = e.GetPublicSpreadTrades(ctx, "")
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
				Exchange:     e.Name,
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
		tradeData, err = e.GetTrades(ctx, instrumentID, 1000)
		if err != nil {
			return nil, err
		}

		resp = make([]trade.Data, len(tradeData))
		for x := range tradeData {
			resp[x] = trade.Data{
				TID:          tradeData[x].TradeID,
				Exchange:     e.Name,
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
	if e.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}
	trade.SortByDate(resp)
	return resp, nil
}

// GetHistoricTrades retrieves historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if !e.SupportsAsset(assetType) || assetType == asset.Spread {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}

	if timestampStart.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	const limit = 100
	pairFormat, err := e.GetPairFormat(assetType, true)
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
		trades, err = e.GetTradesHistory(ctx, instrumentID, "", tradeIDEnd, limit)
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
				Exchange:     e.Name,
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
	if e.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}
	trade.SortByDate(resp)
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// submitPrelude carries the validated and formatted values shared by REST and
// websocket order submission.
type submitPrelude struct {
	pairString     string
	tradeMode      string
	sideType       string
	positionSide   string
	positionMode   string
	amount         float64
	targetCurrency string
}

// validateSubmitPrelude performs the validation and formatting common to REST
// and websocket order submission.
func (e *Exchange) validateSubmitPrelude(ctx context.Context, s *order.Submit) (*submitPrelude, error) {
	if s == nil {
		return nil, order.ErrSubmissionIsNil
	}
	if !e.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
	if s.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	pairFormat, err := e.GetPairFormat(s.AssetType, true)
	if err != nil {
		return nil, err
	}
	if s.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if s.AssetType.IsFutures() && s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w received '%v'", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	tradeMode := e.marginTypeToString(s.MarginType)
	if tradeMode == "" {
		// OKX requires tdMode, and the zero margin type leaves it empty: spot
		// defaults to cash and perpetual swap to cross.
		switch s.AssetType {
		case asset.Spot:
			tradeMode = TradeModeCash
		case asset.Margin, asset.Futures, asset.PerpetualSwap, asset.Options:
			tradeMode = TradeModeCross
		}
	}
	p := &submitPrelude{
		pairString: pairFormat.Format(s.Pair),
		tradeMode:  tradeMode,
		amount:     s.Amount,
	}
	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.Spread:
		p.sideType = s.Side.String()
	case asset.Futures, asset.PerpetualSwap, asset.Options:
		// OKX requires side for every instrument type, and contracts carry
		// the direction in side as well as posSide, which only futures and
		// perpetual swap accept.
		if s.Side.IsLong() {
			p.sideType = order.Buy.Lower()
		} else if s.Side.IsShort() {
			p.sideType = order.Sell.Lower()
		}
		if s.AssetType != asset.Options {
			p.positionSide = s.Side.Lower()
		}
	}
	if s.AssetType == asset.Futures || s.AssetType == asset.PerpetualSwap {
		// Futures and perpetual swap placement branches on the account's
		// position mode: net mode pairs reduceOnly with posSide net, long/short
		// mode pairs the side with the position side instead.
		mode, err := e.contractPositionMode(ctx)
		if err != nil {
			return nil, err
		}
		p.positionMode = mode
	}
	if s.AssetType == asset.Spot && s.Type == order.Market {
		p.targetCurrency = "base_ccy" // Default to base currency
		if s.QuoteAmount > 0 {
			p.amount = s.QuoteAmount
			p.targetCurrency = "quote_ccy"
		}
	}
	return p, nil
}

// deriveSpreadOrderParam converts a validated submit into a spread order
// parameter, with the formatted pair identifying the spread as the sprdId.
func deriveSpreadOrderParam(s *order.Submit, p *submitPrelude) (*SpreadOrderParam, error) {
	spreadOrderType, err := spreadOrderTypeString(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	return &SpreadOrderParam{
		SpreadID:      p.pairString,
		ClientOrderID: s.ClientOrderID,
		Side:          p.sideType,
		OrderType:     spreadOrderType,
		Size:          s.Amount,
		Price:         s.Price,
	}, nil
}

// derivePlaceOrderRequest converts a validated submit into a place order
// request for the supplied OKX order type string.
func derivePlaceOrderRequest(s *order.Submit, p *submitPrelude, oType string) *PlaceOrderRequestParam {
	orderRequest := &PlaceOrderRequestParam{
		InstrumentID:   p.pairString,
		TradeMode:      p.tradeMode,
		Side:           p.sideType,
		PositionSide:   p.positionSide,
		OrderType:      oType,
		Amount:         p.amount,
		ClientOrderID:  s.ClientOrderID,
		TargetCurrency: p.targetCurrency,
		AssetType:      s.AssetType,
	}
	// px only applies to the limit-style order types, so a market order
	// carries no price even when the submit set one.
	switch oType {
	case orderLimit, orderPostOnly, orderFOK, orderIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly:
		orderRequest.Price = s.Price
	}
	// OKX applies reduceOnly to MARGIN orders and FUTURES/SWAP orders in net
	// mode only.
	switch s.AssetType {
	case asset.Margin:
		orderRequest.ReduceOnly = s.ReduceOnly
	case asset.Futures, asset.PerpetualSwap:
		// The account's position mode decides the placement: net mode pairs
		// reduceOnly with posSide net, and long/short mode pairs the side
		// with the position side, expressing the close through it.
		orderRequest.PositionSide = positionSideForMode(s.Side, s.ReduceOnly, p.positionMode)
		orderRequest.ReduceOnly = s.ReduceOnly && p.positionMode == positionModeNet
	}
	return orderRequest
}

// positionSideForMode returns the posSide a futures or perpetual swap order
// sends under the account's position mode. Net mode takes net, where
// reduceOnly carries the close intent. Long/short mode pairs the side with
// the position side instead: an open points at its own direction and a close
// at the opposite one, so sell plus long closes a long position.
func positionSideForMode(side order.Side, reduceOnly bool, mode string) string {
	if mode != positionModeLongShort {
		return positionSideNet
	}
	if reduceOnly {
		if side.IsLong() {
			return positionSideShort
		}
		return positionSideLong
	}
	if side.IsLong() {
		return positionSideLong
	}
	return positionSideShort
}

// SubmitOrder submits a new order via the exchange REST API.
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	p, err := e.validateSubmitPrelude(ctx, s)
	if err != nil {
		return nil, err
	}
	if s.AssetType == asset.Spread {
		spreadParam, err := deriveSpreadOrderParam(s, p)
		if err != nil {
			return nil, err
		}
		placeSpreadOrderResponse, err := e.PlaceSpreadOrder(ctx, spreadParam)
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(placeSpreadOrderResponse.OrderID)
	}
	orderTypeStr, err := orderTypeString(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	// Algo orders carry posSide only for futures and perpetual swap, where the
	// account's position mode decides it.
	var positionSide string
	if s.AssetType == asset.Futures || s.AssetType == asset.PerpetualSwap {
		positionSide = positionSideForMode(s.Side, s.ReduceOnly, p.positionMode)
	}
	var result *AlgoOrder
	switch orderTypeStr {
	case orderLimit, orderMarket, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly:
		placeOrderResponse, err := e.PlaceOrder(ctx, derivePlaceOrderRequest(s, p, orderTypeStr))
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(placeOrderResponse.OrderID)
	case orderTrigger:
		if s.Price == 0 {
			// OKX requires orderPx on trigger orders: -1 submits a market
			// order when triggered, any other value is the limit price.
			return nil, fmt.Errorf("%w, order price is required, -1 submits a market order when triggered", limits.ErrPriceBelowMin)
		}
		result, err = e.PlaceTriggerAlgoOrder(ctx, &AlgoOrderParams{
			InstrumentID:     p.pairString,
			TradeMode:        p.tradeMode,
			Side:             p.sideType,
			PositionSide:     positionSide,
			OrderType:        orderTypeStr,
			Size:             s.Amount,
			ReduceOnly:       s.ReduceOnly,
			TriggerPrice:     s.TriggerPrice,
			OrderPrice:       s.Price,
			TriggerPriceType: priceTypeString(s.TriggerPriceType),
		})
	case orderConditional:
		// Trigger Price and type are used as a stop losss trigger price and type.
		result, err = e.PlaceTakeProfitStopLossOrder(ctx, &AlgoOrderParams{
			InstrumentID:             p.pairString,
			TradeMode:                p.tradeMode,
			Side:                     p.sideType,
			PositionSide:             positionSide,
			OrderType:                orderTypeStr,
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
			return nil, fmt.Errorf("%w, tracking value required", limits.ErrAmountBelowMin)
		}
		result, err = e.PlaceChaseAlgoOrder(ctx, &AlgoOrderParams{
			InstrumentID:  p.pairString,
			TradeMode:     p.tradeMode,
			Side:          p.sideType,
			PositionSide:  positionSide,
			OrderType:     orderTypeStr,
			Size:          s.Amount,
			ReduceOnly:    s.ReduceOnly,
			MaxChaseType:  chaseTypeString(s.TrackingMode),
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
		result, err = e.PlaceTrailingStopOrder(ctx, &AlgoOrderParams{
			InstrumentID:           p.pairString,
			TradeMode:              p.tradeMode,
			Side:                   p.sideType,
			PositionSide:           positionSide,
			OrderType:              orderTypeStr,
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
		result, err = e.PlaceTWAPOrder(ctx, &AlgoOrderParams{
			InstrumentID:  p.pairString,
			TradeMode:     p.tradeMode,
			Side:          p.sideType,
			PositionSide:  positionSide,
			OrderType:     orderTypeStr,
			Size:          s.Amount,
			ReduceOnly:    s.ReduceOnly,
			PriceVariance: priceVar,
			PriceSpread:   priceSpread,
			SizeLimit:     s.Amount,
			LimitPrice:    s.Price,
			// OKX documents no input field for the interval, so the wrapper
			// keeps the 15 minute default and sends it as seconds.
			TimeInterval: strconv.FormatInt(int64(kline.FifteenMin.Duration().Seconds()), 10),
		})
	case orderOCO:
		switch {
		case s.RiskManagementModes.TakeProfit.Price <= 0:
			return nil, fmt.Errorf("%w, take profit price is required", limits.ErrPriceBelowMin)
		case s.RiskManagementModes.StopLoss.Price <= 0:
			return nil, fmt.Errorf("%w, stop loss price is required", limits.ErrPriceBelowMin)
		}
		result, err = e.PlaceAlgoOrder(ctx, &AlgoOrderParams{
			InstrumentID: p.pairString,
			TradeMode:    p.tradeMode,
			Side:         p.sideType,
			PositionSide: positionSide,
			OrderType:    orderTypeStr,
			Size:         s.Amount,
			ReduceOnly:   s.ReduceOnly,

			TakeProfitTriggerPrice:     s.RiskManagementModes.TakeProfit.Price,
			TakeProfitOrderPrice:       s.RiskManagementModes.TakeProfit.LimitPrice,
			TakeProfitTriggerPriceType: priceTypeString(s.TriggerPriceType),

			StopLossTriggerPrice:     s.RiskManagementModes.StopLoss.Price,
			StopLossOrderPrice:       s.RiskManagementModes.StopLoss.LimitPrice,
			StopLossTriggerPriceType: priceTypeString(s.TriggerPriceType),
		})
	default:
		return nil, fmt.Errorf("%w, order type %s", order.ErrTypeIsInvalid, orderTypeStr)
	}
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(result.AlgoID)
}

// contractPositionMode returns the account's contract position mode,
// fetching and caching it on first use: net mode and long/short mode place
// futures and perpetual swap orders differently.
func (e *Exchange) contractPositionMode(ctx context.Context) (string, error) {
	e.accountPositionModeMu.RLock()
	mode := e.accountPositionMode
	e.accountPositionModeMu.RUnlock()
	if mode != "" {
		return mode, nil
	}
	accountConfig, err := e.GetAccountConfiguration(ctx)
	if err != nil {
		return "", fmt.Errorf("error fetching the account position mode: %w", err)
	}
	if accountConfig == nil {
		return "", fmt.Errorf("error fetching the account position mode: %w", common.ErrNoResponse)
	}
	if accountConfig.PositionMode != positionModeNet && accountConfig.PositionMode != positionModeLongShort {
		return "", fmt.Errorf("%w %q", errInvalidPositionMode, accountConfig.PositionMode)
	}
	e.accountPositionModeMu.Lock()
	e.accountPositionMode = accountConfig.PositionMode
	e.accountPositionModeMu.Unlock()
	return accountConfig.PositionMode, nil
}

// chaseTypeString maps the tracking mode to OKX's maxChaseType values, which
// name a ratio rather than the percentage the tracking mode string carries.
func chaseTypeString(mode order.TrackingMode) string {
	switch mode {
	case order.Distance:
		return "distance"
	case order.Percentage:
		return "ratio"
	default:
		return ""
	}
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

func (e *Exchange) marginTypeToString(m margin.Type) string {
	// Unset is the zero value, so the mask subset check alone would let it
	// through and its empty String() would only coincidentally be rejected
	// downstream; exclude it explicitly.
	if m != margin.Unset && allowedMarginTypes&m == m {
		return m.String()
	}
	if margin.Multi == m {
		return TradeModeCross
	}
	return ""
}

// ModifyOrder modifies an existing order via the exchange REST API.
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var err error
	// When asset type is asset.Spread
	if action.AssetType == asset.Spread {
		_, err = e.AmendSpreadOrder(ctx, &AmendSpreadOrderParam{
			OrderID:       action.OrderID,
			ClientOrderID: action.ClientOrderID,
			NewSize:       action.Amount,
			NewPrice:      action.Price,
		})
		if err != nil {
			return nil, err
		}
		return action.DeriveModifyResponse()
	}

	// For other asset type instances.
	pairFormat, err := e.GetPairFormat(action.AssetType, true)
	if err != nil {
		return nil, err
	}
	if action.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(action.Pair)
	switch action.Type {
	case order.UnknownType, order.Market, order.Limit, order.LimitMaker, order.OptimalLimit, order.MarketMakerProtection:
		_, err = e.AmendOrder(ctx, &AmendOrderRequestParams{
			InstrumentID:  instrumentID,
			NewQuantity:   action.Amount,
			OrderID:       action.OrderID,
			ClientOrderID: action.ClientOrderID,
			NewPrice:      action.Price,
		})
		if err != nil {
			return nil, err
		}
	case order.Trigger:
		if action.TriggerPrice == 0 {
			return nil, fmt.Errorf("%w, trigger price required", limits.ErrPriceBelowMin)
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
		_, err = e.AmendAlgoOrder(ctx, &AmendAlgoOrderParam{
			InstrumentID:              instrumentID,
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
			return nil, fmt.Errorf("%w, either take profit trigger price or order price is required", limits.ErrPriceBelowMin)
		case action.RiskManagementModes.StopLoss.Price <= 0 &&
			action.RiskManagementModes.StopLoss.LimitPrice <= 0:
			return nil, fmt.Errorf("%w, either stop loss trigger price or order price is required", limits.ErrPriceBelowMin)
		}
		_, err = e.AmendAlgoOrder(ctx, &AmendAlgoOrderParam{
			InstrumentID:              instrumentID,
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

// CancelOrder cancels an order by its corresponding ID number via the exchange REST API.
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if ord == nil {
		return order.ErrCancelOrderIsNil
	}
	if !e.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	if ord.AssetType == asset.Spread {
		_, err := e.CancelSpreadOrder(ctx, ord.OrderID, ord.ClientOrderID)
		return err
	}
	pairFormat, err := e.GetPairFormat(ord.AssetType, true)
	if err != nil {
		return err
	}
	if ord.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(ord.Pair)
	switch ord.Type {
	case order.UnknownType, order.Market, order.Limit, order.LimitMaker, order.OptimalLimit, order.MarketMakerProtection:
		_, err = e.CancelSingleOrder(ctx, &CancelOrderRequestParam{
			InstrumentID:  instrumentID,
			OrderID:       ord.OrderID,
			ClientOrderID: ord.ClientOrderID,
		})
	case order.Trigger, order.OCO, order.ConditionalStop, order.TWAP, order.TrailingStop, order.Chase:
		var response []AlgoOrder
		response, err = e.CancelAlgoOrder(ctx, []AlgoOrderCancelParams{
			{
				AlgoOrderID:  ord.OrderID,
				InstrumentID: instrumentID,
			},
		})
		if err != nil {
			return err
		}
		if len(response) == 0 {
			return fmt.Errorf("%w for algo order %s", common.ErrNoResponse, ord.OrderID)
		}
		return getStatusError(response[0].StatusCode, response[0].StatusMessage)
	default:
		return fmt.Errorf("%w, order type %v", order.ErrUnsupportedOrderType, ord.Type)
	}
	return err
}

// maxCancelAlgosPerRequest is the documented maximum number of algo orders
// POST /api/v5/trade/cancel-algos accepts per request.
const maxCancelAlgosPerRequest = 10

// CancelBatchOrders cancels orders by their corresponding ID numbers via the
// exchange REST API.
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) > 20 {
		return nil, fmt.Errorf("%w, cannot cancel more than 20 orders", errExceedLimit)
	} else if len(o) == 0 {
		return nil, fmt.Errorf("%w, must have at least 1 cancel order", order.ErrCancelOrderIsNil)
	}
	cancelOrderParams := make([]CancelOrderRequestParam, 0, len(o))
	cancelAlgoOrderParams := make([]AlgoOrderCancelParams, 0, len(o))
	cancelSpreadOrderParams := make([]order.Cancel, 0, len(o))
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	var err error
	// The whole batch is validated before any cancel is sent, so an invalid
	// entry cannot leave a partially executed batch behind.
	for x := range o {
		ord := o[x]
		if !e.SupportsAsset(ord.AssetType) {
			return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
		}
		var pairFormat currency.PairFormat
		pairFormat, err = e.GetPairFormat(ord.AssetType, true)
		if err != nil {
			return nil, err
		}
		if !ord.Pair.IsPopulated() {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		if ord.AssetType == asset.Spread {
			if ord.OrderID == "" && ord.ClientOrderID == "" {
				return nil, fmt.Errorf("%w, order ID required for spread order cancel", order.ErrOrderIDNotSet)
			}
			cancelSpreadOrderParams = append(cancelSpreadOrderParams, ord)
			continue
		}
		switch ord.Type {
		case order.UnknownType, order.Market, order.Limit, order.LimitMaker, order.OptimalLimit, order.MarketMakerProtection:
			if o[x].ClientOrderID == "" && o[x].OrderID == "" {
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
	// Cancels from here on execute, so a failure returns resp with the
	// statuses already recorded rather than discarding them.
	for x := range cancelSpreadOrderParams {
		ord := cancelSpreadOrderParams[x]
		// OKX accepts spread operations only on its business websocket, and
		// WSCancelSpreadOrder sends on the private one, so spread cancels use REST.
		var cancelled *SpreadOrderResponse
		cancelled, err = e.CancelSpreadOrder(ctx, ord.OrderID, ord.ClientOrderID)
		switch {
		case err != nil:
			return resp, err
		case cancelled == nil:
			return resp, fmt.Errorf("%w cancelling spread order ID %q client order ID %q", common.ErrNoResponse, ord.OrderID, ord.ClientOrderID)
		case cancelled.StatusCode != 0:
			return resp, getStatusError(cancelled.StatusCode, cancelled.StatusMessage)
		case cancelled.OrderID == "":
			return resp, fmt.Errorf("%w: no order ID cancelling spread order ID %q client order ID %q", common.ErrInvalidResponse, ord.OrderID, ord.ClientOrderID)
		}
		// Status keys are exchange order IDs, as on the ordinary path below, so
		// each cancel is keyed by the ordId OKX returns, even one sent by client
		// order ID.
		resp.Status[cancelled.OrderID] = order.Cancelled.String()
	}
	if len(cancelOrderParams) > 0 {
		canceledOrders, err := e.CancelMultipleOrders(ctx, cancelOrderParams)
		if cancelResultsUsable(err) {
			for x := range canceledOrders {
				if canceledOrders[x] == nil || canceledOrders[x].OrderID == "" {
					continue
				}
				if canceledOrders[x].StatusCode == 0 {
					resp.Status[canceledOrders[x].OrderID] = order.Cancelled.String()
				} else {
					resp.Status[canceledOrders[x].OrderID] = canceledOrders[x].StatusMessage
				}
			}
		}
		if err != nil {
			return resp, err
		}
	}
	if len(cancelAlgoOrderParams) > 0 {
		// cancel-advance-algos is no longer in OKX's documentation and the
		// documented cancel-algos accepts at most maxCancelAlgosPerRequest
		// orders per request, so the batch is sent in chunks. A partially
		// successful chunk leaves the chunks after it to cancel, as
		// cancelInBatches does for ordinary orders.
		var batchErr error
		for start := 0; start < len(cancelAlgoOrderParams); start += maxCancelAlgosPerRequest {
			end := min(start+maxCancelAlgosPerRequest, len(cancelAlgoOrderParams))
			algoResults, err := e.CancelAlgoOrder(ctx, cancelAlgoOrderParams[start:end])
			if cancelResultsUsable(err) {
				// OKX reports one result per requested algo order; failed
				// cancels are reported with their status message instead of a
				// false Cancelled.
				for x := range algoResults {
					if algoResults[x].AlgoID == "" {
						continue
					}
					if algoResults[x].StatusCode == 0 {
						resp.Status[algoResults[x].AlgoID] = order.Cancelled.String()
					} else {
						resp.Status[algoResults[x].AlgoID] = algoResults[x].StatusMessage
					}
				}
			}
			batchErr = common.AppendError(batchErr, err)
			if err != nil && !cancelResultsUsable(err) {
				// A failed chunk stops the chunks after it, as the ordinary
				// batch branch above does.
				return resp, batchErr
			}
		}
		return resp, batchErr
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair via the
// exchange REST API.
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	err := orderCancellation.Validate()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}

	// For asset.Spread asset orders cancellation. OKX's mass-cancel scopes to
	// one spread instrument via its sprdId, the spread pair itself such as
	// BTC-USDT_BTC-USDT-SWAP, and cancels every spread order when sprdId is
	// omitted, so a populated pair scopes the cancel instead of the order ID,
	// which names a single order rather than a spread. The pair's own
	// underscore-delimited form is used as-is: the configured spread pair
	// format's dash delimiter would mangle the legs.
	if orderCancellation.AssetType == asset.Spread {
		var spreadID string
		if orderCancellation.Pair.IsPopulated() {
			spreadID = orderCancellation.Pair.Upper().String()
		}
		var success bool
		success, err = e.CancelAllSpreadOrders(ctx, spreadID)
		if err != nil {
			return cancelAllResponse, err
		}
		// The result is keyed by the scope the request sent, since the order ID
		// and client order ID scope nothing on a mass cancel.
		cancelAllResponse.Status[spreadID] = strconv.FormatBool(success)
		return cancelAllResponse, nil
	}

	cancelAllOrdersRequestParams, err := e.pendingOrdersToCancel(ctx, orderCancellation)
	if err != nil {
		return cancelAllResponse, err
	}
	return cancelInBatches(ctx, cancelAllOrdersRequestParams, e.CancelMultipleOrders)
}

// errSpreadWebsocketUnsupported reports that a spread order operation cannot
// run over the order websocket: OKX accepts spread trading only on its
// business connection, while order operations transmit on the private
// connection.
var errSpreadWebsocketUnsupported = fmt.Errorf("%w: spread orders require the OKX business websocket connection", common.ErrFunctionNotSupported)

// requireWebsocketInstrumentIDCode returns the cached instrument ID code a
// websocket order operation requires. An explicit websocket call has no
// transport to fall back to, so an uncached instrument must fail before any
// request transmits: OKX ignores instId on order operation frames, and newly
// listed instruments carry a null code until OKX generates one.
func (e *Exchange) requireWebsocketInstrumentIDCode(instID string) (uint64, error) {
	code, ok := e.websocketInstrumentIDCode(instID)
	if !ok {
		return 0, fmt.Errorf("%w: %s", errMissingInstrumentIDCode, instID)
	}
	return code, nil
}

// prepareWebsocketPlaceOrder validates a submit and converts it into a
// private-connection place order request. Algo order types and spread orders
// return common.ErrFunctionNotSupported before any request transmits; algo
// orders have no websocket equivalent, and spread orders require the business
// connection.
func (e *Exchange) prepareWebsocketPlaceOrder(ctx context.Context, s *order.Submit) (*PlaceOrderRequestParam, error) {
	p, err := e.validateSubmitPrelude(ctx, s)
	if err != nil {
		return nil, err
	}
	if s.AssetType == asset.Spread {
		return nil, errSpreadWebsocketUnsupported
	}
	orderTypeStr, err := orderTypeString(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	switch orderTypeStr {
	case orderLimit, orderMarket, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly:
		orderRequest := derivePlaceOrderRequest(s, p, orderTypeStr)
		orderRequest.InstrumentIDCode, err = e.requireWebsocketInstrumentIDCode(orderRequest.InstrumentID)
		if err != nil {
			return nil, err
		}
		return orderRequest, nil
	default:
		return nil, fmt.Errorf("%w: algo order type %s", common.ErrFunctionNotSupported, orderTypeStr)
	}
}

// WebsocketSubmitOrder submits an order through the authenticated private
// websocket connection. Algo order types and spread orders return
// common.ErrFunctionNotSupported before any request is transmitted.
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	orderRequest, err := e.prepareWebsocketPlaceOrder(ctx, s)
	if err != nil {
		return nil, err
	}
	placeOrderResponse, err := e.WSPlaceOrder(ctx, orderRequest)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(placeOrderResponse.OrderID)
}

// WebsocketSubmitOrders submits orders in a single batch-orders frame through
// the authenticated private websocket connection. Every order is validated
// and converted before the batch transmits, so an invalid order cannot leave
// a partially submitted batch behind. Algo order types and spread orders
// return common.ErrFunctionNotSupported before any request is transmitted.
func (e *Exchange) WebsocketSubmitOrders(ctx context.Context, orders []*order.Submit) ([]*order.SubmitResponse, error) {
	if len(orders) == 0 {
		return nil, fmt.Errorf("%T: %w", orders, order.ErrSubmissionIsNil)
	}
	if len(orders) > 20 {
		return nil, fmt.Errorf("%w, cannot submit more than 20 orders", errExceedLimit)
	}
	args := make([]PlaceOrderRequestParam, len(orders))
	for i := range orders {
		orderRequest, err := e.prepareWebsocketPlaceOrder(ctx, orders[i])
		if err != nil {
			return nil, err
		}
		args[i] = *orderRequest
	}
	placed, err := e.WSPlaceMultipleOrders(ctx, args)
	if (err == nil || len(placed) != 0) && len(placed) != len(orders) {
		return nil, common.AppendError(err, fmt.Errorf("%w: %d results for %d orders", common.ErrInvalidResponse, len(placed), len(orders)))
	}
	responses := make([]*order.SubmitResponse, len(orders))
	for i := range placed {
		if placed[i] == nil || placed[i].OrderID == "" {
			continue
		}
		// A partial success keeps its transport error alongside the per-order
		// results, matching the batch cancel contract above; a derive failure
		// joins it instead of replacing it.
		var derr error
		responses[i], derr = orders[i].DeriveSubmitResponse(placed[i].OrderID)
		if derr != nil {
			err = common.AppendError(err, derr)
		}
	}
	return responses, err
}

// WebsocketModifyOrder amends an existing order through the authenticated
// private websocket connection. Algo orders and spread orders return
// common.ErrFunctionNotSupported before any request is transmitted.
func (e *Exchange) WebsocketModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if action.AssetType == asset.Spread {
		return nil, errSpreadWebsocketUnsupported
	}
	pairFormat, err := e.GetPairFormat(action.AssetType, true)
	if err != nil {
		return nil, err
	}
	if action.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(action.Pair)
	switch action.Type {
	case order.UnknownType, order.Market, order.Limit, order.LimitMaker, order.OptimalLimit, order.MarketMakerProtection:
		code, err := e.requireWebsocketInstrumentIDCode(instrumentID)
		if err != nil {
			return nil, err
		}
		_, err = e.WSAmendOrder(ctx, &AmendOrderRequestParams{
			InstrumentID:     instrumentID,
			InstrumentIDCode: code,
			NewQuantity:      action.Amount,
			OrderID:          action.OrderID,
			ClientOrderID:    action.ClientOrderID,
			NewPrice:         action.Price,
		})
		if err != nil {
			return nil, err
		}
		return action.DeriveModifyResponse()
	default:
		return nil, fmt.Errorf("%w: amend of order type %v", common.ErrFunctionNotSupported, action.Type)
	}
}

// WebsocketCancelOrder cancels an order through the authenticated private
// websocket connection. Algo orders and spread orders return
// common.ErrFunctionNotSupported before any request is transmitted.
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, ord *order.Cancel) error {
	if ord == nil {
		return order.ErrCancelOrderIsNil
	}
	if !e.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	if ord.AssetType == asset.Spread {
		return errSpreadWebsocketUnsupported
	}
	pairFormat, err := e.GetPairFormat(ord.AssetType, true)
	if err != nil {
		return err
	}
	if ord.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(ord.Pair)
	switch ord.Type {
	case order.UnknownType, order.Market, order.Limit, order.LimitMaker, order.OptimalLimit, order.MarketMakerProtection:
		code, err := e.requireWebsocketInstrumentIDCode(instrumentID)
		if err != nil {
			return err
		}
		_, err = e.WSCancelOrder(ctx, &CancelOrderRequestParam{
			InstrumentID:     instrumentID,
			InstrumentIDCode: code,
			OrderID:          ord.OrderID,
			ClientOrderID:    ord.ClientOrderID,
		})
		return err
	default:
		return fmt.Errorf("%w: cancel of order type %v", common.ErrFunctionNotSupported, ord.Type)
	}
}

// WebsocketCancelBatchOrders cancels a batch of orders through the
// authenticated private websocket connection. Every order is validated and
// converted before the batch transmits, so an invalid order cannot leave a
// partially executed batch behind. Algo order types and spread orders return
// common.ErrFunctionNotSupported before any request is transmitted.
func (e *Exchange) WebsocketCancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) > 20 {
		return nil, fmt.Errorf("%w, cannot cancel more than 20 orders", errExceedLimit)
	}
	if len(o) == 0 {
		return nil, fmt.Errorf("%w, must have at least 1 cancel order", order.ErrCancelOrderIsNil)
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	args := make([]CancelOrderRequestParam, 0, len(o))
	for x := range o {
		ord := o[x]
		if !e.SupportsAsset(ord.AssetType) {
			return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
		}
		if ord.AssetType == asset.Spread {
			return nil, errSpreadWebsocketUnsupported
		}
		pairFormat, err := e.GetPairFormat(ord.AssetType, true)
		if err != nil {
			return nil, err
		}
		if !ord.Pair.IsPopulated() {
			return nil, currency.ErrCurrencyPairsEmpty
		}
		switch ord.Type {
		case order.UnknownType, order.Market, order.Limit, order.LimitMaker, order.OptimalLimit, order.MarketMakerProtection:
			if ord.OrderID == "" && ord.ClientOrderID == "" {
				return nil, fmt.Errorf("%w: order ID required for order of type %v", order.ErrOrderIDNotSet, ord.Type)
			}
			instrumentID := pairFormat.Format(ord.Pair)
			code, err := e.requireWebsocketInstrumentIDCode(instrumentID)
			if err != nil {
				return nil, err
			}
			args = append(args, CancelOrderRequestParam{
				InstrumentID:     instrumentID,
				InstrumentIDCode: code,
				OrderID:          ord.OrderID,
				ClientOrderID:    ord.ClientOrderID,
			})
		case order.Trigger, order.OCO, order.ConditionalStop,
			order.TWAP, order.TrailingStop, order.Chase:
			return nil, fmt.Errorf("%w: cancel of algo order type %v", common.ErrFunctionNotSupported, ord.Type)
		default:
			return nil, fmt.Errorf("%w: order of type %v not supported", order.ErrUnsupportedOrderType, ord.Type)
		}
	}
	cancelled, err := e.WSCancelMultipleOrders(ctx, args)
	if cancelResultsUsable(err) {
		for x := range cancelled {
			if cancelled[x] == nil || cancelled[x].OrderID == "" {
				continue
			}
			if cancelled[x].StatusCode == 0 {
				resp.Status[cancelled[x].OrderID] = order.Cancelled.String()
			} else {
				resp.Status[cancelled[x].OrderID] = cancelled[x].StatusMessage
			}
		}
	}
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// WebsocketCancelAllOrders cancels all orders associated with a currency pair
// through the authenticated private websocket connection. Algo order types and
// spread orders return common.ErrFunctionNotSupported before any request is
// transmitted: the pending order list holds no algo orders, and spread orders
// require the business websocket connection.
func (e *Exchange) WebsocketCancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}
	if orderCancellation == nil {
		return cancelAllResponse, order.ErrCancelOrderIsNil
	}
	if orderCancellation.AssetType == asset.Spread {
		return cancelAllResponse, errSpreadWebsocketUnsupported
	}
	switch orderCancellation.Type {
	case order.Trigger, order.OCO, order.ConditionalStop, order.TWAP, order.TrailingStop, order.Chase:
		return cancelAllResponse, fmt.Errorf("%w: cancel of algo order type %v", common.ErrFunctionNotSupported, orderCancellation.Type)
	}
	err := orderCancellation.Validate()
	if err != nil {
		return cancelAllResponse, err
	}

	cancelAllOrdersRequestParams, err := e.pendingOrdersToCancel(ctx, orderCancellation)
	if err != nil {
		return cancelAllResponse, err
	}
	// An order whose instrument has no cached code cannot be cancelled over
	// the websocket. It is reported in the error rather than stopping the
	// cancels that can be sent, as a failed batch does below.
	var errs error
	sendable := make([]CancelOrderRequestParam, 0, len(cancelAllOrdersRequestParams))
	for i := range cancelAllOrdersRequestParams {
		code, codeErr := e.requireWebsocketInstrumentIDCode(cancelAllOrdersRequestParams[i].InstrumentID)
		if codeErr != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w cancelling order %s", codeErr, cancelAllOrdersRequestParams[i].OrderID))
			continue
		}
		cancelAllOrdersRequestParams[i].InstrumentIDCode = code
		sendable = append(sendable, cancelAllOrdersRequestParams[i])
	}
	resp, batchErrs := cancelInBatches(ctx, sendable, e.WSCancelMultipleOrders)
	errs = common.AppendError(errs, batchErrs)
	if errs != nil {
		return resp, errs
	}
	return resp, nil
}

// pendingOrdersToCancel pages through the pending orders a cancel-all selects:
// OKX caps the pending order list at 100 records per request, so the full list
// is crawled before cancelling to reach accounts holding more open orders than
// a single page, and the orders the cancellation scopes to are kept.
func (e *Exchange) pendingOrdersToCancel(ctx context.Context, c *order.Cancel) ([]CancelOrderRequestParam, error) {
	var err error
	var instrumentType string
	if c.AssetType.IsValid() {
		err = e.CurrencyPairs.IsAssetEnabled(c.AssetType)
		if err != nil {
			return nil, err
		}
		instrumentType = GetInstrumentTypeFromAssetItem(c.AssetType)
	}
	var oType string
	if c.Type != order.UnknownType && c.Type != order.AnyType {
		oType, err = orderTypeFilter(c.Type, c.TimeInForce)
		if err != nil {
			return nil, err
		}
	}
	var curr string
	if c.Pair.IsPopulated() {
		if c.AssetType.IsValid() {
			// Format through the exchange's pair format so callers passing a
			// differently delimited pair still resolve their instrument; OKX
			// rejects an unmatched instId with error 51001.
			var pairFormat currency.PairFormat
			pairFormat, err = e.GetPairFormat(c.AssetType, true)
			if err != nil {
				return nil, err
			}
			curr = pairFormat.Format(c.Pair)
		} else {
			curr = c.Pair.Upper().String()
		}
	}
	var myOrders []OrderDetail
	for after := ""; ; {
		var page []OrderDetail
		page, err = e.GetOrderList(ctx, &OrderListRequestParams{
			InstrumentType: instrumentType,
			OrderType:      oType,
			InstrumentID:   curr,
			After:          after,
		})
		if err != nil {
			return nil, err
		}
		myOrders = append(myOrders, page...)
		if len(page) < orderListPageSize {
			break
		}
		after = page[len(page)-1].OrderID
	}
	cancelAllOrdersRequestParams := make([]CancelOrderRequestParam, 0, len(myOrders))
ordersLoop:
	for x := range myOrders {
		switch {
		case c.OrderID != "" || c.ClientOrderID != "":
			// Every supplied discriminator must match, so supplying both IDs
			// cannot cancel an order matching only one of them.
			if (c.OrderID == "" || myOrders[x].OrderID == c.OrderID) &&
				(c.ClientOrderID == "" || myOrders[x].ClientOrderID == c.ClientOrderID) {
				cancelAllOrdersRequestParams = append(cancelAllOrdersRequestParams, CancelOrderRequestParam{
					InstrumentID:  myOrders[x].InstrumentID,
					OrderID:       myOrders[x].OrderID,
					ClientOrderID: myOrders[x].ClientOrderID,
				})
				break ordersLoop
			}
		case c.Side == order.Buy || c.Side == order.Sell:
			if myOrders[x].Side == c.Side {
				cancelAllOrdersRequestParams = append(cancelAllOrdersRequestParams, CancelOrderRequestParam{
					InstrumentID:  myOrders[x].InstrumentID,
					OrderID:       myOrders[x].OrderID,
					ClientOrderID: myOrders[x].ClientOrderID,
				})
			}
		default:
			cancelAllOrdersRequestParams = append(cancelAllOrdersRequestParams, CancelOrderRequestParam{
				InstrumentID:  myOrders[x].InstrumentID,
				OrderID:       myOrders[x].OrderID,
				ClientOrderID: myOrders[x].ClientOrderID,
			})
		}
	}
	return cancelAllOrdersRequestParams, nil
}

// cancelInBatches cancels orders 20 at a time with cancel, recording each
// result; a failed batch does not stop the batches after it.
func cancelInBatches(ctx context.Context, params []CancelOrderRequestParam, cancel func(context.Context, []CancelOrderRequestParam) ([]*OrderData, error)) (order.CancelAllResponse, error) {
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}
	remaining := params
	loop := int(math.Ceil(float64(len(remaining)) / 20.0))
	var errs error
	for range loop {
		if ctxErr := ctx.Err(); ctxErr != nil {
			// A dead context stops the loop; the statuses and errors collected
			// so far are still returned so the caller sees what was cancelled.
			// The failed batch's error already carries the context error, so
			// append it only once.
			if !errors.Is(errs, ctxErr) {
				errs = common.AppendError(errs, ctxErr)
			}
			break
		}
		batch := remaining
		if len(batch) > 20 {
			batch = batch[:20]
			remaining = remaining[20:]
		} else {
			remaining = nil
		}
		response, err := cancel(ctx, batch)
		// A failed batch does not stop later batches; the errors are joined so
		// a cancel-all still reaches every remaining order.
		errs = common.AppendError(errs, err)
		if !cancelResultsUsable(err) {
			continue
		}
		for y := range response {
			if response[y] == nil || response[y].OrderID == "" {
				continue
			}
			if response[y].StatusCode == 0 {
				cancelAllResponse.Status[response[y].OrderID] = order.Cancelled.String()
			} else {
				cancelAllResponse.Status[response[y].OrderID] = response[y].StatusMessage
			}
		}
	}
	if errs != nil {
		return cancelAllResponse, errs
	}
	return cancelAllResponse, nil
}

// cancelResultsUsable reports whether per-order cancel results can be recorded
// alongside err. A partial success on the websocket or over REST returns fully
// decoded results with its error; any other error, a decode error included,
// can leave them half populated.
func cancelResultsUsable(err error) bool {
	return err == nil || errors.Is(err, errPartialSuccess)
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if assetType == asset.Spread {
		var resp *SpreadOrder
		resp, err := e.GetSpreadOrderDetails(ctx, orderID, "")
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(resp.Side)
		if err != nil {
			return nil, err
		}
		oType, tif, err := orderTypeFromString(resp.OrderType)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(resp.State)
		if err != nil {
			return nil, err
		}
		format, err := e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		// OKX's spread order response documents sprdId, not instId, so the
		// pair comes from the spread ID like the other spread listings.
		cp, err := currency.NewPairDelimiter(resp.SpreadID, format.Delimiter)
		if err != nil {
			return nil, err
		}
		if !pair.IsEmpty() && !cp.Equal(pair) {
			return nil, fmt.Errorf("%w, unexpected instrument ID %v for order ID %s", order.ErrOrderNotFound, pair, orderID)
		}
		spreadAmt := resp.Size.Float64()
		spreadExec := resp.AccFillSize.Float64()
		spreadRemaining := float64(0)
		if oStatus != order.Filled && spreadAmt > spreadExec {
			spreadRemaining = spreadAmt - spreadExec
		}
		return &order.Detail{
			Amount:               spreadAmt,
			Exchange:             e.Name,
			OrderID:              resp.OrderID,
			ClientOrderID:        resp.ClientOrderID,
			Side:                 oSide,
			Type:                 oType,
			Pair:                 cp,
			Cost:                 resp.Price.Float64(),
			AssetType:            assetType,
			Status:               oStatus,
			Price:                resp.Price.Float64(),
			ExecutedAmount:       spreadExec,
			Date:                 resp.CreationTime.Time(),
			LastUpdated:          resp.UpdateTime.Time(),
			AverageExecutedPrice: resp.AveragePrice.Float64(),
			RemainingAmount:      spreadRemaining,
			TimeInForce:          tif,
		}, nil
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !pair.IsPopulated() {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	instrumentID := pairFormat.Format(pair)
	orderDetail, err := e.GetOrderDetail(ctx, &OrderDetailRequestParam{
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

	amount, remaining, quoteAmount := orderAmounts(orderDetail, status)
	return &order.Detail{
		Amount:          amount,
		Exchange:        e.Name,
		OrderID:         orderDetail.OrderID,
		ClientOrderID:   orderDetail.ClientOrderID,
		Side:            orderDetail.Side,
		Type:            orderType,
		Pair:            pair,
		Cost:            orderDetail.Price.Float64(),
		AssetType:       assetType,
		Status:          status,
		Price:           orderDetail.Price.Float64(),
		ExecutedAmount:  orderDetail.AccumulatedFillSize.Float64(),
		RemainingAmount: remaining,
		QuoteAmount:     quoteAmount,
		Date:            orderDetail.CreationTime.Time(),
		LastUpdated:     orderDetail.UpdateTime.Time(),
		TimeInForce:     tif,
	}, nil
}

// targetCurrencyQuote is the tgtCcy value that sizes an order in its quote
// currency.
const targetCurrencyQuote = "quote_ccy"

// orderAmounts returns an order's size and unfilled size in the base currency,
// and its size in the quote currency when OKX states it that way: tgtCcy
// quote_ccy sizes a spot market order in the quote currency, while accFillSz
// is always in the base currency. It sizes them as wsProcessOrders does.
func orderAmounts(o *OrderDetail, status order.Status) (amount, remaining, quoteAmount float64) {
	amount = o.Size.Float64()
	executed := o.AccumulatedFillSize.Float64()
	if o.QuantityType == targetCurrencyQuote {
		quoteAmount = amount
		switch avgPrice := o.AveragePrice.Float64(); {
		case status == order.Filled:
			amount = executed
		case avgPrice > 0:
			amount = quoteAmount / avgPrice
		default:
			amount = 0
		}
	}
	if status != order.Filled && amount > executed {
		remaining = amount - executed
	}
	return amount, remaining, quoteAmount
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	response, err := e.GetCurrencyDepositAddress(ctx, c)
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
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
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
	resp, err := e.Withdrawal(ctx, &input)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.WithdrawalID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.StartTime.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	if !e.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}

	var resp []order.Detail
	var format currency.PairFormat
	if req.AssetType == asset.Spread {
		var spreadOrderType string
		if req.Type != order.UnknownType && req.Type != order.AnyType {
			spreadOrderType, err = spreadOrderTypeFilter(req.Type, req.TimeInForce)
			if err != nil {
				return nil, err
			}
		}
		// OKX caps the pending spread order response at orderListPageSize
		// records and pages the remainder with the endId cursor: endId returns
		// records earlier than the order ID, the direction the newest-first
		// listing pages. beginId returns records newer than the order ID and
		// cannot walk the pages.
		var spreads []SpreadOrder
		for endID := ""; ; {
			var page []SpreadOrder
			page, err = e.GetActiveSpreadOrders(ctx, "", spreadOrderType, "", req.FromOrderID, endID, 0)
			if err != nil {
				return nil, err
			}
			spreads = append(spreads, page...)
			if len(page) < orderListPageSize {
				break
			}
			endID = page[len(page)-1].OrderID
		}
		for x := range spreads {
			format, err = e.GetPairFormat(asset.Spread, true)
			if err != nil {
				return nil, err
			}
			var (
				pair    currency.Pair
				oType   order.Type
				tif     order.TimeInForce
				oSide   order.Side
				oStatus order.Status
			)

			pair, err = currency.NewPairDelimiter(spreads[x].SpreadID, format.Delimiter)
			if err != nil {
				return nil, err
			}
			oType, tif, err = orderTypeFromString(spreads[x].OrderType)
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
			spreadAmt := spreads[x].Size.Float64()
			spreadExec := spreads[x].AccFillSize.Float64()
			spreadRemaining := float64(0)
			if oStatus != order.Filled && spreadAmt > spreadExec {
				spreadRemaining = spreadAmt - spreadExec
			}
			resp = append(resp, order.Detail{
				Amount:          spreadAmt,
				Pair:            pair,
				Price:           spreads[x].Price.Float64(),
				ExecutedAmount:  spreadExec,
				RemainingAmount: spreadRemaining,
				Exchange:        e.Name,
				OrderID:         spreads[x].OrderID,
				ClientOrderID:   spreads[x].ClientOrderID,
				Type:            oType,
				Side:            oSide,
				Status:          oStatus,
				AssetType:       req.AssetType,
				Date:            spreads[x].CreationTime.Time(),
				LastUpdated:     spreads[x].UpdateTime.Time(),
				TimeInForce:     tif,
			})
		}
		return req.Filter(e.Name, resp), nil
	}

	instrumentType := GetInstrumentTypeFromAssetItem(req.AssetType)
	var orderType string
	if req.Type != order.UnknownType && req.Type != order.AnyType {
		orderType, err = orderTypeFilter(req.Type, req.TimeInForce)
		if err != nil {
			return nil, err
		}
	}
	// OKX pages the pending order list with the after order ID cursor; the End
	// timestamp is not a documented parameter, so paginating with it re-fetched
	// the first page, losing every order past the first hundred.
allOrders:
	for after := ""; ; {
		var orderList []OrderDetail
		orderList, err = e.GetOrderList(ctx, &OrderListRequestParams{
			OrderType:      orderType,
			InstrumentType: instrumentType,
			After:          after,
		})
		if err != nil {
			return nil, err
		}
		if len(orderList) == 0 {
			break
		}
		for i := range orderList {
			if orderList[i].CreationTime.Time().Before(req.StartTime) {
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
			amount, remaining, quoteAmount := orderAmounts(&orderList[i], orderStatus)
			resp = append(resp, order.Detail{
				Amount:          amount,
				Pair:            pair,
				Price:           orderList[i].Price.Float64(),
				ExecutedAmount:  orderList[i].AccumulatedFillSize.Float64(),
				RemainingAmount: remaining,
				QuoteAmount:     quoteAmount,
				Fee:             orderList[i].TransactionFee.Float64(),
				FeeAsset:        currency.NewCode(orderList[i].FeeCurrency),
				Exchange:        e.Name,
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
		if len(orderList) < orderListPageSize {
			break
		}
		after = orderList[len(orderList)-1].OrderID
	}
	return req.Filter(e.Name, resp), nil
}

// GetOrderHistory retrieves account order information Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.StartTime.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	if !e.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}
	var resp []order.Detail
	var err error
	if req.AssetType == asset.Spread {
		resp, err = e.getSpreadOrderHistoryDetails(ctx, req)
	} else {
		resp, err = e.getStandardOrderHistoryDetails(ctx, req)
	}
	if err != nil {
		return nil, err
	}
	return req.Filter(e.Name, resp), nil
}

// getSpreadOrderHistoryDetails retrieves completed spread orders, paging the
// 21 day listing and, when the requested window reaches beyond it, the 3 month
// archive, de-duplicating the overlap between the two listings.
func (e *Exchange) getSpreadOrderHistoryDetails(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	var spreadOrderType string
	if req.Type != order.UnknownType && req.Type != order.AnyType {
		var err error
		spreadOrderType, err = spreadOrderTypeFilter(req.Type, req.TimeInForce)
		if err != nil {
			return nil, err
		}
	}
	// OKX caps the spread order history response at orderListPageSize
	// records and pages the remainder with the same earlier-than order ID
	// endId cursor as the pending spread order listing. The 21 day listing
	// carries the freshest orders, which the archive lags, and OKX confirmed
	// its begin filter reaches the full 21 day window, so the archive is only
	// crawled when the requested window extends past it.
	// req.FromOrderID seeds the endId cursor so the crawl returns records
	// earlier than it, matching the standard history's after semantics;
	// beginId, which returns records newer than an order ID, is never sent.
	var spreadOrders []SpreadOrder
	seen := make(map[string]struct{})
	record := func(page []SpreadOrder) {
		for i := range page {
			if _, ok := seen[page[i].OrderID]; ok {
				continue
			}
			seen[page[i].OrderID] = struct{}{}
			spreadOrders = append(spreadOrders, page[i])
		}
	}
	// The 21 day listing cannot hold orders older than its window, so it is
	// crawled only when the requested window reaches into it.
	if req.EndTime.IsZero() || req.EndTime.After(time.Now().Add(-kline.ThreeWeek.Duration())) {
		for endID := req.FromOrderID; ; {
			var page []SpreadOrder
			page, err := e.GetCompletedSpreadOrdersLast21Days(ctx, "", spreadOrderType, "", "", endID, req.StartTime, req.EndTime, 0)
			if err != nil {
				return nil, err
			}
			record(page)
			if len(page) < orderListPageSize {
				break
			}
			next := page[len(page)-1].OrderID
			if next == endID {
				// The page did not advance past the cursor; stop rather than
				// request the same page forever.
				break
			}
			endID = next
		}
	}
	if req.StartTime.IsZero() || req.StartTime.Before(time.Now().Add(-kline.ThreeWeek.Duration())) {
		// The 21 day listing cannot reach past its documented window, so the
		// archive covers the remainder of the documented 3 month window.
		for endID := req.FromOrderID; ; {
			var page []SpreadOrder
			page, err := e.GetCompletedSpreadOrdersLast3Months(ctx, "", spreadOrderType, "", "", endID, req.StartTime, req.EndTime, 0)
			if err != nil {
				return nil, err
			}
			record(page)
			if len(page) < orderListPageSize {
				break
			}
			next := page[len(page)-1].OrderID
			if next == endID {
				// The page did not advance past the cursor; stop rather than
				// request the same page forever.
				break
			}
			endID = next
		}
	}
	format, err := e.GetPairFormat(asset.Spread, true)
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range spreadOrders {
		detail, err := e.spreadOrderToDetail(&spreadOrders[x], req.AssetType, format)
		if err != nil {
			return nil, err
		}
		resp = append(resp, detail)
	}
	return resp, nil
}

// spreadOrderToDetail converts a completed spread order listing row into an
// order Detail. OKX documents sprdId, not instId, so the pair comes from the
// spread ID like the other spread listings; cutting at the first delimiter
// keeps the same encoding the pair manager stores for asset.Spread, so
// formatting the pair reproduces the spread ID.
func (e *Exchange) spreadOrderToDetail(so *SpreadOrder, assetType asset.Item, format currency.PairFormat) (order.Detail, error) {
	pair, err := currency.NewPairDelimiter(so.SpreadID, format.Delimiter)
	if err != nil {
		return order.Detail{}, err
	}
	oType, tif, err := orderTypeFromString(so.OrderType)
	if err != nil {
		return order.Detail{}, err
	}
	oSide, err := order.StringToOrderSide(so.Side)
	if err != nil {
		return order.Detail{}, err
	}
	oStatus, err := order.StringToOrderStatus(so.State)
	if err != nil {
		return order.Detail{}, err
	}
	amount := so.Size.Float64()
	executed := so.AccFillSize.Float64()
	remaining := float64(0)
	if oStatus != order.Filled && amount > executed {
		remaining = amount - executed
	}
	return order.Detail{
		Price:                so.Price.Float64(),
		AverageExecutedPrice: so.AveragePrice.Float64(),
		Amount:               amount,
		ExecutedAmount:       executed,
		RemainingAmount:      remaining,
		Exchange:             e.Name,
		OrderID:              so.OrderID,
		ClientOrderID:        so.ClientOrderID,
		Type:                 oType,
		Side:                 oSide,
		Status:               oStatus,
		AssetType:            assetType,
		Date:                 so.CreationTime.Time(),
		LastUpdated:          so.UpdateTime.Time(),
		Pair:                 pair,
		TimeInForce:          tif,
	}, nil
}

// getStandardOrderHistoryDetails retrieves the standard order history for the
// requested asset type, filtered to the requested pairs when they are set. The
// history endpoints require only the instrument type from OKX, so empty pairs
// returns every instrument the listings carry. Pair narrowing happens in the
// result set only: instId is a single-value parameter on both endpoints (a
// comma-separated list is rejected with 51000), so pushing pairs down would
// multiply the crawls. The 3 month archive is the primary source; the 7 day
// listing is crawled beside it when the requested window reaches its reach.
func (e *Exchange) getStandardOrderHistoryDetails(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	instrumentType := GetInstrumentTypeFromAssetItem(req.AssetType)
	// OKX returns both listings newest first and pages the remainder with the
	// after cursor, which returns the records earlier than the requested
	// order ID. Unlike the end timestamp, the order ID cursor is exclusive,
	// so orders sharing one creation millisecond cannot straddle a page
	// boundary and reappear at the head of the next page. The seen set only
	// guards against OKX repeating a row the cursor already passed, and
	// de-duplicates the rows the two listings share.
	var resp []order.Detail
	seen := make(map[string]struct{})
	crawl := func(fetch func(after string) ([]OrderDetail, error)) error {
	allOrders:
		for after := req.FromOrderID; ; {
			orderList, err := fetch(after)
			if err != nil {
				return err
			}
			if len(orderList) == 0 {
				break
			}
			for i := range orderList {
				if orderList[i].CreationTime.Time().Before(req.StartTime.Add(-time.Second)) {
					// Reached the end of the crawl: rows arrive newest
					// first, so every row after is older than StartTime.
					// The one second margin keeps rows the second-granular
					// time filter in req.Filter would accept.
					break allOrders
				}
				if _, ok := seen[orderList[i].OrderID]; ok {
					continue
				}
				seen[orderList[i].OrderID] = struct{}{}
				pair, err := currency.NewPairFromString(orderList[i].InstrumentID)
				if err != nil {
					return err
				}
				if len(req.Pairs) > 0 && !slices.ContainsFunc(req.Pairs, pair.Equal) {
					continue
				}
				orderStatus, err := order.StringToOrderStatus(strings.ToUpper(orderList[i].State))
				if err != nil {
					return err
				}
				if orderStatus == order.Active {
					continue
				}
				oType, tif, err := orderTypeFromString(orderList[i].OrderType)
				if err != nil {
					return err
				}
				amount, remaining, quoteAmount := orderAmounts(&orderList[i], orderStatus)
				resp = append(resp, order.Detail{
					Price:                orderList[i].Price.Float64(),
					AverageExecutedPrice: orderList[i].AveragePrice.Float64(),
					Amount:               amount,
					ExecutedAmount:       orderList[i].AccumulatedFillSize.Float64(),
					RemainingAmount:      remaining,
					QuoteAmount:          quoteAmount,
					Fee:                  orderList[i].TransactionFee.Float64(),
					FeeAsset:             currency.NewCode(orderList[i].FeeCurrency),
					Exchange:             e.Name,
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
					CostAsset:            pair.Quote,
					TimeInForce:          tif,
				})
			}
			if len(orderList) < orderListPageSize {
				break
			}
			next := orderList[len(orderList)-1].OrderID
			if next == after {
				// The page did not advance past the cursor; stop rather than
				// request the same page forever.
				break
			}
			after = next
		}
		return nil
	}
	if err := crawl(func(after string) ([]OrderDetail, error) {
		return e.Get3MonthOrderHistory(ctx, &OrderHistoryRequestParams{
			InstrumentType: instrumentType,
			After:          after,
			Start:          req.StartTime,
			End:            req.EndTime,
		})
	}); err != nil {
		return nil, err
	}
	// The archive does not contain canceled orders without any fills; the
	// 7 day listing does, retaining them for 2 hours, and carries the
	// freshest orders before they reach the archive. It is crawled when the
	// requested window reaches the listing's 7 day reach.
	if req.EndTime.IsZero() || req.EndTime.After(time.Now().Add(-kline.SevenDay.Duration())) {
		if err := crawl(func(after string) ([]OrderDetail, error) {
			return e.Get7DayOrderHistory(ctx, &OrderHistoryRequestParams{
				InstrumentType: instrumentType,
				After:          after,
				Start:          req.StartTime,
				End:            req.EndTime,
			})
		}); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// ValidateAPICredentials validates current credentials used for wrapper
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	var timeSeries []kline.Candle
	switch a {
	case asset.Spread:
		candles, err := e.GetSpreadCandlesticksHistory(ctx, req.RequestFormatted.String(), req.ExchangeInterval, start.Add(-time.Nanosecond), end, 100)
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
		candles, err := e.GetCandlesticksHistory(ctx,
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
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
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
			candles, err := e.GetSpreadCandlesticksHistory(ctx,
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
			candles, err := e.GetCandlesticksHistory(ctx,
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

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	currencyChains, err := e.GetFundingCurrencies(ctx, cryptocurrency)
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
func (e *Exchange) getInstrumentsForOptions(ctx context.Context) ([]Instrument, error) {
	underlyings, err := e.GetPublicUnderlyings(ctx, instTypeOption)
	if err != nil {
		return nil, err
	}
	var insts []Instrument
	for x := range underlyings {
		var instruments []Instrument
		instruments, err = e.GetInstruments(ctx, &InstrumentsFetchParams{
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
func (e *Exchange) getInstrumentsForAsset(ctx context.Context, a asset.Item) ([]Instrument, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	var instruments []Instrument
	var instType string
	var err error
	switch a {
	case asset.Options:
		instruments, err = e.getInstrumentsForOptions(ctx)
		if err != nil {
			return nil, err
		}
		e.cacheInstruments(instTypeOption, instruments)
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

	instruments, err = e.GetInstruments(ctx, &InstrumentsFetchParams{
		InstrumentType: instType,
	})
	if err != nil {
		return nil, err
	}
	e.cacheInstruments(instType, instruments)
	return instruments, nil
}

// cacheInstruments stores instruments by instrument type and indexes their
// instrument ID codes by instrument ID for getInstrumentIDCode lookups
func (e *Exchange) cacheInstruments(instType string, instruments []Instrument) {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	e.instrumentsInfoMap[instType] = instruments
	e.cacheInstrumentIDCodesLocked(instruments)
}

// cacheInstrumentIDCodes upserts instrument ID codes from instruments channel
// pushes, so instruments listed after the startup fetch resolve codes in a
// running process. The instruments channel is not part of
// defaultSubscriptions, so this only runs when it is subscribed explicitly.
// The instruments list itself is left alone because the channel pushes deltas,
// not full snapshots.
func (e *Exchange) cacheInstrumentIDCodes(instruments []Instrument) {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	e.cacheInstrumentIDCodesLocked(instruments)
}

func (e *Exchange) cacheInstrumentIDCodesLocked(instruments []Instrument) {
	for x := range instruments {
		// OKX sends a null instIdCode until it generates one, including for a
		// relisted instrument, whose code changes. Dropping the cached code
		// fails its websocket order operations instead of sending a stale code.
		if instruments[x].InstrumentIDCode == 0 {
			delete(e.instrumentIDCodeMap, instruments[x].InstrumentID.String())
			continue
		}
		e.instrumentIDCodeMap[instruments[x].InstrumentID.String()] = instruments[x].InstrumentIDCode
	}
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.PerpetualSwap {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w, pair required", currency.ErrCurrencyPairEmpty)
	}
	format, err := e.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := r.Pair.Format(format)
	pairRate := fundingrate.LatestRateResponse{
		TimeChecked: time.Now(),
		Exchange:    e.Name,
		Asset:       r.Asset,
		Pair:        fPair,
	}
	fr, err := e.GetSingleFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	var fri time.Duration
	if len(e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
		// can infer funding rate interval from the only funding rate frequency defined
		for k := range e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
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
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	requestLimit := 100
	sd := r.StartDate
	maxLookback := time.Now().Add(-e.Features.Supports.FuturesCapabilities.MaximumFundingRateHistory)
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
	format, err := e.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := r.Pair.Format(format)
	pairRate := fundingrate.HistoricalRates{
		Exchange:  e.Name,
		Asset:     r.Asset,
		Pair:      fPair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	// map of time indexes, allowing for easy lookup of slice index from unix time data
	mti := make(map[int64]int)
	for sd.Before(r.EndDate) {
		var frh []FundingRateResponse
		frh, err = e.GetFundingRateHistory(ctx, fPair.String(), sd, r.EndDate, int64(requestLimit))
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
	fr, err = e.GetSingleFundingRate(ctx, fPair.String())
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
		billDetailsFunc := e.GetBillsDetail3Months
		if time.Since(r.StartDate) < kline.OneWeek.Duration() {
			billDetailsFunc = e.GetBillsDetailLast7Days
		}
		for sd.Before(r.EndDate) {
			var fri time.Duration
			if len(e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
				// can infer funding rate interval from the only funding rate frequency defined
				for k := range e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
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
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.PerpetualSwap, nil
}

// SetMarginType sets the default margin type for when opening a new position
// okx allows this to be set with an order, however this sets a default
func (e *Exchange) SetMarginType(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type) error {
	return fmt.Errorf("%w margin type is set per order", common.ErrFunctionNotSupported)
}

// SetCollateralMode sets the collateral type for your account
func (e *Exchange) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// GetCollateralMode returns the collateral type for your account
func (e *Exchange) GetCollateralMode(ctx context.Context, item asset.Item) (collateral.Mode, error) {
	if !e.SupportsAsset(item) {
		return 0, fmt.Errorf("%w: %v", asset.ErrNotSupported, item)
	}
	cfg, err := e.GetAccountConfiguration(ctx)
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
func (e *Exchange) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionChangeRequest", common.ErrNilPointer)
	}
	if !e.SupportsAsset(req.Asset) {
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
	pairFormat, err := e.GetPairFormat(req.Asset, true)
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

	resp, err := e.IncreaseDecreaseMargin(ctx, r)
	if err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		AllocatedMargin: resp.Amount.Float64(),
		MarginType:      req.MarginType,
	}, nil
}

// GetFuturesPositionSummary returns position summary details for an active position
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	if !e.SupportsAsset(req.Asset) || !req.Asset.IsFutures() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	fPair, err := e.FormatExchangeCurrency(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	instrumentType := GetInstrumentTypeFromAssetItem(req.Asset)

	var contracts []futures.Contract
	contracts, err = e.GetFuturesContractDetails(ctx, req.Asset)
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

	positionSummaries, err := e.GetPositions(ctx, instrumentType, fPair.String(), "")
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

	acc, err := e.AccountBalance(ctx, currency.EMPTYCODE)
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
	collateralMode, err := e.GetCollateralMode(ctx, req.Asset)
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
		CurrentSize:                  positionSummary.QuantityOfPosition.Decimal().Mul(decimal.MustFromFloat(multiplier)),
		ContractSize:                 positionSummary.QuantityOfPosition.Decimal(),
		ContractMultiplier:           decimal.MustFromFloat(multiplier),
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
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if !e.SupportsAsset(req.Asset) || !req.Asset.IsFutures() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	if time.Since(req.StartDate) > e.Features.Supports.MaximumOrderHistory {
		if req.RespectOrderHistoryLimits {
			req.StartDate = time.Now().Add(-e.Features.Supports.MaximumOrderHistory)
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, time.Now().Add(-e.Features.Supports.MaximumOrderHistory))
		}
	}
	err := common.StartEndTimeCheck(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(req.Pairs))
	var contracts []futures.Contract
	contracts, err = e.GetFuturesContractDetails(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	contractsMap := make(map[currency.Pair]*futures.Contract)
	for i := range contracts {
		contractsMap[contracts[i].Name] = &contracts[i]
	}
	for i := range req.Pairs {
		fPair, err := e.FormatExchangeCurrency(req.Pairs[i], req.Asset)
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
			InstrumentType: instrumentType,
			InstrumentID:   fPair.String(),
			Start:          req.StartDate,
			End:            req.EndDate,
		}
		if time.Since(req.StartDate) <= time.Hour*24*7 {
			positions, err = e.Get7DayOrderHistory(ctx, historyRequest)
		} else {
			positions, err = e.Get3MonthOrderHistory(ctx, historyRequest)
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
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
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
				Exchange:             e.Name,
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
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, orderSide order.Side) error {
	switch item {
	case asset.Futures, asset.PerpetualSwap, asset.Margin:
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	marginMode := e.marginTypeToString(marginType)
	if marginMode == "" {
		return fmt.Errorf("%w: %v", margin.ErrMarginTypeUnsupported, marginType)
	}
	var posSide string
	if marginMode == TradeModeIsolated && item != asset.Margin {
		// OKX requires posSide for isolated contract leverage only in
		// long/short mode, where the side selects the position; net mode
		// leaves it unset, which OKX reads as net.
		mode, err := e.contractPositionMode(ctx)
		if err != nil {
			return err
		}
		if mode == positionModeLongShort {
			switch {
			case orderSide.IsLong():
				posSide = positionSideLong
			case orderSide.IsShort():
				posSide = positionSideShort
			default:
				return fmt.Errorf("%w: %v, isolated leverage in long/short mode requires a long or short side", order.ErrSideIsInvalid, orderSide)
			}
		}
	}
	instrumentID, err := e.FormatSymbol(pair, item)
	if err != nil {
		return err
	}
	_, err = e.SetLeverageRate(ctx, &SetLeverageInput{
		Leverage:     amount,
		MarginMode:   marginMode,
		InstrumentID: instrumentID,
		PositionSide: posSide,
	})
	return err
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (e *Exchange) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, orderSide order.Side) (float64, error) {
	switch item {
	case asset.Futures, asset.PerpetualSwap, asset.Margin:
	default:
		return -1, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	marginMode := e.marginTypeToString(marginType)
	if marginMode == "" {
		return -1, fmt.Errorf("%w: %v", margin.ErrMarginTypeUnsupported, marginType)
	}
	var posSide string
	if marginMode == TradeModeIsolated && item != asset.Margin {
		// Isolated contract leverage is reported per position side in
		// long/short mode and as a single net row in net mode.
		mode, err := e.contractPositionMode(ctx)
		if err != nil {
			return -1, err
		}
		if mode == positionModeLongShort {
			switch {
			case orderSide.IsLong():
				posSide = positionSideLong
			case orderSide.IsShort():
				posSide = positionSideShort
			default:
				return -1, fmt.Errorf("%w: %v, isolated leverage in long/short mode requires a long or short side", order.ErrSideIsInvalid, orderSide)
			}
		} else {
			posSide = positionSideNet
		}
	}
	instrumentID, err := e.FormatSymbol(pair, item)
	if err != nil {
		return -1, err
	}
	lev, err := e.GetLeverageRate(ctx, instrumentID, marginMode, currency.EMPTYCODE)
	if err != nil {
		return -1, err
	}
	if len(lev) == 0 {
		return -1, fmt.Errorf("%w %v %v %s", futures.ErrPositionNotFound, item, pair, marginType)
	}
	if posSide != "" {
		for i := range lev {
			if lev[i].PositionSide == posSide {
				return lev[i].Leverage.Float64(), nil
			}
		}
		// A per-side request must not silently fall back to a row belonging
		// to the opposite side or a stale net row.
		return -1, fmt.Errorf("%w %v %v %s posSide %s", futures.ErrPositionNotFound, item, pair, marginType, posSide)
	}

	// leverage is the same across positions
	return lev[0].Leverage.Float64(), nil
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	switch item {
	case asset.Futures, asset.PerpetualSwap:
		instType := GetInstrumentTypeFromAssetItem(item)
		result, err := e.GetInstruments(ctx, &InstrumentsFetchParams{
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
				Exchange:       e.Name,
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
				resp[i].SettlementCurrency = settleCurr
			}
		}
		return resp, nil
	case asset.Spread:
		results, err := e.GetPublicSpreads(ctx, "", "", "", "")
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
				Exchange:       e.Name,
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
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
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
				underlyings, err = e.GetPublicUnderlyings(ctx, instTypeOption)
				if err != nil {
					return nil, err
				}
				for u := range underlyings {
					var incOID []OpenInterest
					incOID, err = e.GetOpenInterestData(ctx, instType, underlyings[u], "", "")
					if err != nil {
						return nil, err
					}
					oid = append(oid, incOID...)
				}
			case instTypeSwap,
				instTypeFutures:
				oid, err = e.GetOpenInterestData(ctx, instType, "", "", "")
				if err != nil {
					return nil, err
				}
			}
			for j := range oid {
				var isEnabled bool
				var p currency.Pair
				p, isEnabled, err = e.MatchSymbolCheckEnabled(oid[j].InstrumentID, v, true)
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
					Key:          key.NewExchangeAssetPair(e.Name, v, p),
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
	pFmt, err := e.FormatSymbol(k[0].Pair(), k[0].Asset)
	if err != nil {
		return nil, err
	}
	var oid []OpenInterest
	switch instTypes[k[0].Asset] {
	case instTypeOption:
		var underlyings []string
		underlyings, err = e.GetPublicUnderlyings(ctx, instTypeOption)
		if err != nil {
			return nil, err
		}
		for u := range underlyings {
			var incOID []OpenInterest
			incOID, err = e.GetOpenInterestData(ctx, instTypes[k[0].Asset], underlyings[u], "", "")
			if err != nil {
				return nil, err
			}
			oid = append(oid, incOID...)
		}
	case instTypeSwap, instTypeFutures:
		oid, err = e.GetOpenInterestData(ctx, instTypes[k[0].Asset], "", "", pFmt)
		if err != nil {
			return nil, err
		}
	}
	for i := range oid {
		p, isEnabled, err := e.MatchSymbolCheckEnabled(oid[i].InstrumentID, k[0].Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		resp[0] = futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, p),
			OpenInterest: oid[i].OpenInterest.Float64(),
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
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
		cp, err = e.FormatExchangeCurrency(cp, a)
		if err != nil {
			return "", err
		}
		insts, err := e.GetInstruments(ctx, &InstrumentsFetchParams{
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
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// MessageID returns a universally unique ID using UUID V7, with hyphens removed to fit the maximum 32-character field for okx
func (e *Exchange) MessageID() string {
	u := uuid.NewV7()
	var buf [32]byte
	hex.Encode(buf[:], u[:])
	return string(buf[:])
}

// getInstrumentIDCode returns the OKX instrument ID code for an instrument
// ID, or zero if it is not cached. Websocket order, amend and cancel
// operations resolve codes here because OKX ignores instId on those frames,
// while no REST order endpoint documents instIdCode, so REST request bodies
// must stay free of the field.
func (e *Exchange) getInstrumentIDCode(instID string) uint64 {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	return e.instrumentIDCodeMap[instID]
}

// websocketInstrumentIDCode returns the cached instrument ID code for an
// instrument and whether it is available. An uncached instrument reports ok as
// false: OKX requires the code on websocket operations, and newly listed
// instruments carry a null code until OKX generates one.
func (e *Exchange) websocketInstrumentIDCode(instID string) (uint64, bool) {
	code := e.getInstrumentIDCode(instID)
	return code, code != 0
}
