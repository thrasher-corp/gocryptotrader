package okx

import (
	"cmp"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// timeNow is the clock the wrapper judges request windows and chooses history
// listings by; the mock tests set it to their recording time
var timeNow = time.Now

// GetFee returns Cryptocurrency trade fee, and offline trade fee
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, common.ErrNilPointer
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		// The spot underlying doubles as the instrument ID the fee rates are
		// queried by, e.g. BTC-USDT.
		instID, err := e.GetUnderlying(feeBuilder.Pair, asset.Spot)
		if err != nil {
			return 0, err
		}
		responses, err := e.GetTradeFee(ctx, &TradeFeeRequest{InstrumentType: instTypeSpot, InstrumentID: instID})
		if err != nil {
			return 0, err
		}
		if len(responses) == 0 {
			return 0, common.ErrNoResponse
		}
		if len(responses) != 1 || (responses[0].InstrumentType != "" && responses[0].InstrumentType != instTypeSpot) {
			return 0, fmt.Errorf("%w: expected one spot fee response for %s", errFeeRateUnavailable, instID)
		}
		rate := &responses[0]
		var selectedRate *types.Number
		switch {
		case len(rate.FeeGroup) > 0:
			// instId scopes the returned rates, including market maker incentives.
			// If multiple groups remain, resolve the instrument's group explicitly.
			group := &rate.FeeGroup[0]
			if len(rate.FeeGroup) > 1 {
				instruments, err := e.GetInstruments(ctx, &InstrumentsRequest{InstrumentType: instTypeSpot, InstrumentID: instID})
				if err != nil {
					return 0, err
				}
				if len(instruments) != 1 || instruments[0].InstrumentID.String() != instID || instruments[0].InstrumentType != instTypeSpot || instruments[0].TradingFeeGroupID == "" {
					return 0, fmt.Errorf("%w: instrument fee group missing for %s", errFeeRateUnavailable, instID)
				}
				group = nil
				for i := range rate.FeeGroup {
					if rate.FeeGroup[i].GroupID != instruments[0].TradingFeeGroupID {
						continue
					}
					if group != nil {
						return 0, fmt.Errorf("%w: duplicate fee group for %s", errFeeRateUnavailable, instID)
					}
					group = &rate.FeeGroup[i]
				}
			}
			if group == nil || group.GroupID == "" {
				return 0, fmt.Errorf("%w: matching fee group missing for %s", errFeeRateUnavailable, instID)
			}
			selectedRate = group.TakerRate
			if feeBuilder.IsMaker {
				selectedRate = group.MakerRate
			}
		case feeBuilder.Pair.Quote.Equal(currency.USDT):
			selectedRate = rate.FeeRateTaker
			if feeBuilder.IsMaker {
				selectedRate = rate.FeeRateMaker
			}
		case !feeBuilder.Pair.Quote.IsFiatCurrency():
			// Legacy USDC fields also cover other stablecoin and crypto quotes.
			// makerU/takerU describe derivatives and cannot substitute for spot fees.
			selectedRate = rate.FeeRateTakerUSDC
			if feeBuilder.IsMaker {
				selectedRate = rate.FeeRateMakerUSDC
			}
		}
		if selectedRate == nil {
			return 0, fmt.Errorf("%w: maker=%t instrument=%s", errFeeRateUnavailable, feeBuilder.IsMaker, instID)
		}
		// OKX uses negative rates for commission charges and positive rates for rebates.
		return -selectedRate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	case exchange.OfflineTradeFee:
		return 0.0015 * feeBuilder.PurchasePrice * feeBuilder.Amount, nil
	default:
		return 0, errFeeTypeUnsupported
	}
}

const websocketResponseMaxLimit = time.Second * 3

const instrumentStateLive = "live"

const instrumentStatePreOpen = "preopen"

// SetDefaults sets the basic defaults for Okx
func (e *Exchange) SetDefaults() {
	e.Name = "Okx"
	e.Enabled = true
	e.Verbose = true

	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresClientID = true

	e.instrumentsInfoMap = make(map[string][]InstrumentResponse)
	e.instrumentIDCodeMap = make(map[string]uint64)
	e.accountPositionModes = make(map[string]string)
	e.accountLevels = make(map[string]uint64)

	cpf := &currency.PairFormat{
		Delimiter: currency.DashDelimiter,
		Uppercase: true,
	}

	// In this exchange, we represent deliverable futures contracts as 'FUTURES'/asset.Futures and perpetual futures as 'SWAP'/asset.PerpetualSwap
	err := e.SetGlobalPairsManager(cpf, cpf, asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options, asset.Margin, asset.Spread)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
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

	return e.setupWebsocket(exch)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.GetSystemTime(ctx)
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
		spreadInstruments, err := e.GetPublicSpreads(ctx, &GetPublicSpreadsRequest{State: instrumentStateLive})
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
		if _, err := e.getInstrumentsForAsset(ctx, a); err != nil {
			return err
		}
		return e.loadStoredInstrumentOrderExecutionLimits(a)
	case asset.Spread:
		insts, err := e.GetPublicSpreads(ctx, &GetPublicSpreadsRequest{State: instrumentStateLive})
		if err != nil {
			return err
		}
		if len(insts) == 0 {
			return common.ErrNoResponse
		}
		l := make([]limits.MinMaxLevel, len(insts))
		for i := range insts {
			l[i] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, insts[i].SpreadID),
				PriceStepIncrementSize:  insts[i].TickSize.Float64(),
				MinimumBaseAmount:       insts[i].MinSize.Float64(),
				AmountStepIncrementSize: insts[i].LotSize.Float64(),
			}
		}
		return limits.Load(l)
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// loadStoredInstrumentOrderExecutionLimits loads the execution limits of the
// instrument list stored for an asset, holding the instruments lock so that the
// limits of a list a newer listing replaced are not loaded over the newer's
func (e *Exchange) loadStoredInstrumentOrderExecutionLimits(a asset.Item) error {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	return e.loadInstrumentOrderExecutionLimits(a, e.instrumentsInfoMap[GetInstrumentTypeFromAssetItem(a)])
}

func (e *Exchange) loadInstrumentOrderExecutionLimits(a asset.Item, insts []InstrumentResponse) error {
	if len(insts) == 0 {
		return common.ErrNoResponse
	}
	l := make([]limits.MinMaxLevel, 0, len(insts))
	for i := range insts {
		if insts[i].State != instrumentStateLive || insts[i].InstrumentID.IsEmpty() {
			continue
		}
		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, insts[i].InstrumentID),
			PriceStepIncrementSize:  insts[i].TickSize.Float64(),
			MinimumBaseAmount:       insts[i].MinimumOrderSize.Float64(),
			AmountStepIncrementSize: insts[i].LotSize.Float64(),
			MultiplierDecimal:       insts[i].ContractValue.Float64() * insts[i].ContractMultiplier.Float64(),
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

// candleVolume returns a candlestick's volume in the base currency, as tickerVolumes reads a ticker's: vol is the base
// currency on spot and margin, while on the derivative instrument types vol counts contracts and volCcy is the base
// currency, with the same futures caveat
func candleVolume(vol, volCcy types.Number, a asset.Item) float64 {
	switch a {
	case asset.PerpetualSwap, asset.Futures, asset.Options:
		return volCcy.Float64()
	}
	return vol.Float64()
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
		if mdata == nil || mdata.InstrumentID.IsEmpty() {
			return nil, fmt.Errorf("%w: ticker reply has no instrument for %s", common.ErrInvalidResponse, p)
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
			PreOpen:      e.instrumentPreOpen(p.String(), mdata.TickerDataGenerationTime.Time()),
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
			var spreadTickers []SpreadTickerResponse
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
		ticks, err := e.GetTickers(ctx, &TickersRequest{InstrumentType: instrumentType})
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
					PreOpen:      e.instrumentPreOpen(ticks[y].InstrumentID.String(), ticks[y].TickerDataGenerationTime.Time()),
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
			spreadOrderbook []SpreadOrderBookResponse
		)
		pairFormat, err = e.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		spreadOrderbook, err = e.GetPublicSpreadOrderBooks(ctx, &GetPublicSpreadOrderBooksRequest{SpreadID: pairFormat.Format(pair), Depth: 50})
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
			// A spread's price is the difference between its legs' prices, so
			// a level can be priced at zero
			book.Bids = make(orderbook.Levels, 0, len(spreadOrderbook[y].Bids))
			for b := range spreadOrderbook[y].Bids {
				book.Bids = append(book.Bids, orderbook.Level{
					Price:      spreadOrderbook[y].Bids[b][0].Float64(),
					Amount:     spreadOrderbook[y].Bids[b][1].Float64(),
					OrderCount: spreadOrderbook[y].Bids[b][2].Int64(),
				})
			}
			book.Asks = make(orderbook.Levels, 0, len(spreadOrderbook[y].Asks))
			for a := range spreadOrderbook[y].Asks {
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
		var orderBookD *OrderBookResponse
		orderBookD, err = e.GetOrderBookDepth(ctx, &OrderBookRequest{InstrumentID: instrumentID, Depth: 400})
		if err != nil {
			return book, err
		}
		if orderBookD == nil {
			return book, fmt.Errorf("%w: order book reply has no book for %s", common.ErrInvalidResponse, instrumentID)
		}

		book.Bids = make(orderbook.Levels, len(orderBookD.Bids))
		for x := range orderBookD.Bids {
			book.Bids[x] = orderbook.Level{
				Amount:     orderBookD.Bids[x].Size.Float64(),
				Price:      orderBookD.Bids[x].Price.Float64(),
				OrderCount: orderBookD.Bids[x].OrderCount.Int64(),
			}
		}
		book.Asks = make(orderbook.Levels, len(orderBookD.Asks))
		for x := range orderBookD.Asks {
			book.Asks[x] = orderbook.Level{
				Amount:     orderBookD.Asks[x].Size.Float64(),
				Price:      orderBookD.Asks[x].Price.Float64(),
				OrderCount: orderBookD.Asks[x].OrderCount.Int64(),
			}
		}
		book.LastUpdated = orderBookD.Timestamp.Time()
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
	resp, err := e.GetAccountBalance(ctx, &AccountBalanceRequest{})
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w: account balance reply has no account", common.ErrInvalidResponse)
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for i := range resp {
		for j := range resp[i].Details {
			subAccts[0].Balances.Set(resp[i].Details[j].Currency, accounts.Balance{
				Total:     resp[i].Details[j].EquityOfCurrency.Float64(),
				Hold:      resp[i].Details[j].FrozenBalance.Float64(),
				Free:      resp[i].Details[j].AvailableBalance.Float64(),
				Borrowed:  resp[i].Details[j].Liabilities.Float64(),
				UpdatedAt: resp[i].Details[j].UpdateTime.Time(),
			})
		}
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	depositHistories, err := e.GetCurrencyDepositHistory(ctx, &DepositHistoryRequest{})
	if err != nil {
		return nil, err
	}

	withdrawalHistories, err := e.GetWithdrawalHistory(ctx, &WithdrawalHistoryRequest{})
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(depositHistories)+len(withdrawalHistories))
	for x := range depositHistories {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          depositHistories[x].State,
			Timestamp:       depositHistories[x].Timestamp.Time(),
			Currency:        depositHistories[x].Currency.String(),
			Amount:          depositHistories[x].Amount.Float64(),
			TransferType:    "deposit",
			CryptoToAddress: depositHistories[x].To,
			CryptoTxID:      depositHistories[x].TransactionID,
		})
	}
	for x := range withdrawalHistories {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          withdrawalHistories[x].State,
			Timestamp:       withdrawalHistories[x].Timestamp.Time(),
			Currency:        withdrawalHistories[x].Currency.String(),
			Amount:          withdrawalHistories[x].Amount.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalHistories[x].To,
			CryptoTxID:      withdrawalHistories[x].TransactionID,
			TransferID:      withdrawalHistories[x].WithdrawalID,
			Fee:             withdrawalHistories[x].Fee.Float64(),
			CryptoChain:     withdrawalHistories[x].Chain,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.GetWithdrawalHistory(ctx, &WithdrawalHistoryRequest{Currency: c})
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for x := range withdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          withdrawals[x].State,
			Timestamp:       withdrawals[x].Timestamp.Time(),
			Currency:        withdrawals[x].Currency.String(),
			Amount:          withdrawals[x].Amount.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[x].To,
			CryptoTxID:      withdrawals[x].TransactionID,
			CryptoChain:     withdrawals[x].Chain,
			TransferID:      withdrawals[x].WithdrawalID,
			Fee:             withdrawals[x].Fee.Float64(),
		})
	}
	return resp, nil
}

// maxTradesPerRequest is the most trades OKX's trades endpoint returns
const maxTradesPerRequest = 500

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	format, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Spread:
		if p.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		var spreadTrades []SpreadPublicTradeResponse
		spreadTrades, err = e.GetPublicSpreadTrades(ctx, &GetPublicSpreadTradesRequest{SpreadID: format.Format(p)})
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
	case asset.Spot, asset.Margin, asset.Futures, asset.PerpetualSwap, asset.Options:
		if p.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		instrumentID := format.Format(p)
		var tradeData []TradeResponse
		tradeData, err = e.GetTrades(ctx, &TradesRequest{InstrumentID: instrumentID, Limit: maxTradesPerRequest})
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
				Amount:       tradeData[x].Size.Float64(),
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

	if timestampStart.Before(timeNow().Add(-kline.ThreeMonth.Duration())) {
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
		trades, err = e.GetTradesHistory(ctx, &TradesHistoryRequest{InstrumentID: instrumentID, After: tradeIDEnd, Limit: limit})
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
				Amount:       trades[i].Size.Float64(),
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
// websocket order submission. marginCurrency is the ccy an isolated MARGIN
// order, or a cross MARGIN order in Futures mode, sends: the quote currency for
// a buy and the base for a sell.
type submitPrelude struct {
	pairString     string
	tradeMode      string
	sideType       string
	positionMode   string
	marginCurrency currency.Code
}

// validateSubmitPrelude performs the validation and formatting common to REST
// and websocket order submission, with the values the account's configuration
// decides.
func (e *Exchange) validateSubmitPrelude(ctx context.Context, s *order.Submit) (*submitPrelude, error) {
	p, err := e.checkSubmitPrelude(s)
	if err != nil {
		return nil, err
	}
	if err := e.resolveSubmitAccount(ctx, s, p); err != nil {
		return nil, err
	}
	return p, nil
}

// checkSubmitPrelude performs the validation and formatting common to REST and
// websocket order submission that needs no request, so an order refused for
// itself sends none. The trade mode of an order without a margin type is left
// for resolveSubmitAccount.
func (e *Exchange) checkSubmitPrelude(s *order.Submit) (*submitPrelude, error) {
	if s == nil {
		return nil, order.ErrSubmissionIsNil
	}
	if !e.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
	switch {
	case s.TimeInForce.Is(order.GoodTillTime), s.TimeInForce.Is(order.GoodTillDay):
		return nil, fmt.Errorf("%w: %s, OKX orders carry no expiry time", order.ErrUnsupportedTimeInForce, s.TimeInForce)
	case s.TimeInForce.Is(order.StopOrReduce):
		return nil, fmt.Errorf("%w: %s, OKX has no stop or reduce instruction", order.ErrUnsupportedTimeInForce, s.TimeInForce)
	case s.TimeInForce.Is(order.GoodTillCrossing) && !s.TimeInForce.Is(order.PostOnly):
		// GCT documents no meaning for good till crossing, so it is not read as post only unless PostOnly asks for it
		return nil, fmt.Errorf("%w: %s, set PostOnly for an order which must not cross the book", order.ErrUnsupportedTimeInForce, s.TimeInForce)
	}
	if s.Type == order.OCO {
		// An OCO order's legs decide the currency OKX reads its size in
		if _, _, err := ocoLegOrderPrices(&s.RiskManagementModes); err != nil {
			return nil, err
		}
	}
	switch {
	case s.Amount < 0:
		return nil, limits.ErrAmountBelowMin
	case quoteSizedOnly(s):
		if s.QuoteAmount <= 0 {
			return nil, fmt.Errorf("%w: OKX sizes a %s %s %s in the quote currency, which only QuoteAmount states", order.ErrAmountMustBeSet, s.AssetType, s.Type, s.Side)
		}
	case s.Amount == 0 && (s.QuoteAmount <= 0 || !placesSpotMarketOrder(s)):
		return nil, limits.ErrAmountBelowMin
	}
	// Spread orders are checked on their own path, as their prices are
	// differentials between legs and may be zero or negative. Otherwise a
	// market order's price is optional, an optimal limit order takes none and
	// OKX offers it on futures and perpetual swaps only, as it does chase
	// orders, Trigger and ConditionalStop orders send -1 for a market fill, OCO,
	// Chase and TrailingStop orders take their prices from other fields, and
	// every other order type requires a positive price.
	switch s.Type {
	case order.OptimalLimit:
		if s.AssetType != asset.Spread && s.AssetType != asset.Futures && s.AssetType != asset.PerpetualSwap {
			return nil, fmt.Errorf("%w: %v on %v", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
		}
	case order.Market:
		// OKX takes no market orders on options, but does take the ioc or fok
		// a market order places as with that time in force.
		if s.AssetType == asset.Options && placesMarketOrder(s) {
			return nil, fmt.Errorf("%w: %v on %v", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
		}
	case order.Chase:
		if s.AssetType != asset.Futures && s.AssetType != asset.PerpetualSwap {
			return nil, fmt.Errorf("%w: %v on %v", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
		}
	case order.OCO, order.TrailingStop, order.Trigger, order.ConditionalStop:
	case order.MarketMakerProtection:
		if s.AssetType != asset.Options {
			return nil, fmt.Errorf("%w: %v on %v", order.ErrUnsupportedOrderType, s.Type, s.AssetType)
		}
		if s.Price <= 0 {
			return nil, limits.ErrPriceBelowMin
		}
	default:
		if s.AssetType != asset.Spread && s.Price <= 0 {
			return nil, limits.ErrPriceBelowMin
		}
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
	switch {
	case tradeMode == "" && s.MarginType != margin.Unset:
		// Only the zero margin type takes a default: an explicit one OKX
		// cannot express must not silently become another margin mode.
		return nil, fmt.Errorf("%w: %v", margin.ErrMarginTypeUnsupported, s.MarginType)
	case s.AssetType == asset.Margin && (tradeMode == TradeModeCash || tradeMode == TradeModeSpotIsolated):
		// OKX would place a spot order for a non-margin trade mode
		return nil, fmt.Errorf("%w: %v for a margin order", margin.ErrMarginTypeUnsupported, s.MarginType)
	}
	p := &submitPrelude{
		pairString: pairFormat.Format(s.Pair),
		tradeMode:  tradeMode,
	}
	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.Spread:
		p.sideType = s.Side.String()
	case asset.Futures, asset.PerpetualSwap, asset.Options:
		// OKX requires side for every instrument type, and a contract's
		// posSide comes from the account's position mode in the order
		// derivation instead, so the prelude carries the side alone.
		if s.Side.IsLong() {
			p.sideType = order.Buy.Lower()
		} else if s.Side.IsShort() {
			p.sideType = order.Sell.Lower()
		}
	}
	return p, nil
}

// resolveSubmitAccount fills in the prelude values the account's configuration
// decides: the default trade mode of an order without a margin type, the
// position mode of a futures or perpetual swap order, and the margin currency
// of a margin order.
func (e *Exchange) resolveSubmitAccount(ctx context.Context, s *order.Submit, p *submitPrelude) error {
	if p.tradeMode == "" {
		tradeMode, err := e.defaultTradeMode(ctx, s.AssetType)
		if err != nil {
			return err
		}
		p.tradeMode = tradeMode
	}
	switch s.AssetType {
	case asset.Futures, asset.PerpetualSwap:
		// Futures and perpetual swap placement branches on the account's
		// position mode: net mode pairs reduceOnly with posSide net, long/short
		// mode pairs the side with the position side instead.
		mode, err := e.contractPositionMode(ctx)
		if err != nil {
			return err
		}
		p.positionMode = mode
	case asset.Spot, asset.Margin:
		// OKX requires the margin currency on isolated MARGIN orders and on
		// cross MARGIN orders in Futures mode.
		requiresCurrency := p.tradeMode == TradeModeIsolated
		if p.tradeMode == TradeModeCross {
			level, err := e.accountLevel(ctx)
			if err != nil {
				return err
			}
			requiresCurrency = level == accountLevelFutures
		}
		if requiresCurrency {
			switch {
			case s.Side.IsLong():
				p.marginCurrency = s.Pair.Quote.Upper()
			case s.Side.IsShort():
				p.marginCurrency = s.Pair.Base.Upper()
			}
		}
	}
	return nil
}

// defaultTradeMode returns the trade mode an order without a margin type trades in, following OKX's trade modes by
// account mode: spot orders trade in cash in Spot and Futures mode and in cross in the margin modes, option orders in
// cash in Spot mode, where only option buyers trade, and in cross otherwise, and margin, futures and perpetual swap
// orders in cross. A spread order sends no trade mode.
func (e *Exchange) defaultTradeMode(ctx context.Context, a asset.Item) (string, error) {
	switch a {
	case asset.Margin, asset.Futures, asset.PerpetualSwap:
		return TradeModeCross, nil
	case asset.Spot, asset.Options:
	default:
		return "", nil
	}
	level, err := e.accountLevel(ctx)
	if err != nil {
		return "", err
	}
	switch level {
	case accountLevelSpot:
		return TradeModeCash, nil
	case accountLevelFutures:
		if a == asset.Spot {
			return TradeModeCash, nil
		}
		return TradeModeCross, nil
	case accountLevelMultiCurrencyMargin, accountLevelPortfolioMargin:
		return TradeModeCross, nil
	}
	return "", fmt.Errorf("%w %d: OKX documents no default trade mode for it, so the order needs a margin type", errInvalidAccountLevel, level)
}

// deriveSpreadOrderParam converts a validated submit into a spread order
// parameter, with the formatted pair identifying the spread as the sprdId.
// OKX attaches no take profit or stop loss to a spread order.
func deriveSpreadOrderParam(s *order.Submit, p *submitPrelude) (*PlaceSpreadOrderRequest, error) {
	if riskManagementSet(&s.RiskManagementModes) {
		return nil, fmt.Errorf("%w: a spread order carries none", errAttachedOrderUnsupported)
	}
	spreadOrderType, err := spreadOrderTypeString(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	return &PlaceSpreadOrderRequest{
		SpreadID:      p.pairString,
		ClientOrderID: s.ClientOrderID,
		Side:          p.sideType,
		OrderType:     spreadOrderType,
		Size:          s.Amount,
		Price:         s.Price,
	}, nil
}

// placesMarketOrder reports whether a market submit places as an OKX market
// order, rather than as the ioc or fok its time in force asks for.
func placesMarketOrder(s *order.Submit) bool {
	return s.Type == order.Market && s.TimeInForce != order.ImmediateOrCancel && s.TimeInForce != order.FillOrKill
}

// spotMarginOrder reports whether a spot asset submit places what OKX calls a
// MARGIN order: one on a spot instrument in the cross or isolated trade mode.
func spotMarginOrder(s *order.Submit) bool {
	return s.AssetType == asset.Spot && (s.MarginType == margin.Multi || s.MarginType == margin.Isolated)
}

// placesSpotMarketOrder reports whether a submit places a SPOT market order,
// which tgtCcy sizes in the base or quote currency. OKX scopes tgtCcy to SPOT,
// so a spot asset submit traded on margin is sized as a MARGIN order.
func placesSpotMarketOrder(s *order.Submit) bool {
	return s.AssetType == asset.Spot && !spotMarginOrder(s) && placesMarketOrder(s)
}

// quoteSizedOnly reports whether OKX reads a submit's size in the quote
// currency with no tgtCcy to name another: a MARGIN market buy, as documented,
// and a spot or margin algo buy placing a market buy once triggered, which OKX
// sizes as that market buy: a trailing stop, a trigger order at market, an OCO
// order with both legs at market, and a conditional order at market but for a
// SPOT one, whose tgtCcy states its base currency size.
func quoteSizedOnly(s *order.Submit) bool {
	if !s.Side.IsLong() || (s.AssetType != asset.Spot && s.AssetType != asset.Margin) {
		return false
	}
	marginOrder := s.AssetType == asset.Margin || spotMarginOrder(s)
	switch s.Type {
	case order.Market:
		return marginOrder && placesMarketOrder(s)
	case order.TrailingStop:
		return true
	case order.Trigger:
		return s.Price == marketOrderPrice
	case order.ConditionalStop:
		return marginOrder && legOrderPrice(s.TriggerPrice, s.Price) == marketOrderPrice
	case order.OCO:
		takeProfitAtMarket, stopLossAtMarket := ocoLegsAtMarket(s)
		return takeProfitAtMarket && stopLossAtMarket
	}
	return false
}

// ocoLegsAtMarket reports whether an OCO order's take-profit and stop-loss legs execute at market when triggered; a
// leg ocoLegOrderPrices refuses does not
func ocoLegsAtMarket(s *order.Submit) (takeProfit, stopLoss bool) {
	tpPrice, tpErr := attachedLegOrderPrice(&s.RiskManagementModes.TakeProfit)
	slPrice, slErr := attachedLegOrderPrice(&s.RiskManagementModes.StopLoss)
	return tpErr == nil && tpPrice == marketOrderPrice, slErr == nil && slPrice == marketOrderPrice
}

// ocoLegOrderPrices returns the order prices an OCO order's take-profit and stop-loss legs are placed at once
// triggered, refusing a leg as an attached take profit or stop loss is refused
func ocoLegOrderPrices(modes *order.RiskManagementModes) (takeProfit, stopLoss float64, err error) {
	if takeProfit, err = attachedLegOrderPrice(&modes.TakeProfit); err != nil {
		return 0, 0, fmt.Errorf("take profit: %w", err)
	}
	if stopLoss, err = attachedLegOrderPrice(&modes.StopLoss); err != nil {
		return 0, 0, fmt.Errorf("stop loss: %w", err)
	}
	return takeProfit, stopLoss, nil
}

// algoOrderSize returns the size an algo order sends: its QuoteAmount where OKX
// reads the size in the quote currency alone, and its Amount otherwise.
func algoOrderSize(s *order.Submit) float64 {
	if quoteSizedOnly(s) {
		return s.QuoteAmount
	}
	return s.Amount
}

// derivePlaceOrderRequest converts a validated submit into a place order
// request for the supplied OKX order type string.
func derivePlaceOrderRequest(s *order.Submit, p *submitPrelude, oType string) *PlaceOrderRequest {
	orderRequest := &PlaceOrderRequest{
		InstrumentID:  p.pairString,
		TradeMode:     p.tradeMode,
		Currency:      p.marginCurrency,
		Side:          p.sideType,
		OrderType:     oType,
		Size:          s.Amount,
		ClientOrderID: s.ClientOrderID,
		AssetType:     s.AssetType,
	}
	// The price only applies to the limit-style order types, so a market
	// order carries no price even when the submit set one.
	switch oType {
	case orderLimit, orderPostOnly, orderFOK, orderIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly:
		orderRequest.Price = s.Price
	case orderMarket:
		// OKX reads tgtCcy on SPOT market orders only, so a market submit
		// placed as ioc or fok keeps its base currency size.
		switch {
		case placesSpotMarketOrder(s):
			orderRequest.TargetCurrency = targetCurrencyBase
			if s.QuoteAmount > 0 {
				orderRequest.Size = s.QuoteAmount
				orderRequest.TargetCurrency = targetCurrencyQuote
			}
		case quoteSizedOnly(s):
			orderRequest.Size = s.QuoteAmount
		}
	}
	// OKX applies reduceOnly to MARGIN orders, which a spot instrument places
	// in a margin trade mode, and FUTURES/SWAP orders in net mode only.
	switch s.AssetType {
	case asset.Spot, asset.Margin:
		orderRequest.ReduceOnly = s.ReduceOnly && (p.tradeMode == TradeModeCross || p.tradeMode == TradeModeIsolated)
	case asset.Futures, asset.PerpetualSwap:
		// The account's position mode decides the placement: net mode pairs
		// reduceOnly with posSide net, and long/short mode pairs the side
		// with the position side, expressing the close through it.
		orderRequest.PositionSide = positionSideForMode(s.Side, s.ReduceOnly, p.positionMode)
		orderRequest.ReduceOnly = s.ReduceOnly && p.positionMode == positionModeNet
	}
	return orderRequest
}

// marketOrderPrice is OKX's order price that executes a triggered leg at
// market, accepted by a trigger order's orderPx and an OCO or conditional
// order's tpOrdPx and slOrdPx.
const marketOrderPrice = -1

// legOrderPrice returns the order price a take-profit or stop-loss leg sends:
// its limit price, or market when the leg sets only a trigger price, as OKX
// requires an order price beside each trigger price.
func legOrderPrice(triggerPrice, limitPrice float64) float64 {
	if limitPrice == 0 && triggerPrice != 0 {
		return marketOrderPrice
	}
	return limitPrice
}

// spotAlgoTargetCurrency returns the tgtCcy a conditional order sends, the only
// algo order OKX takes tgtCcy on: it reads a SPOT conditional market buy's sz
// in the quote currency unless tgtCcy names base_ccy, so a buy leg executing at
// market names base_ccy. A sell's sz is in the base currency already, and OKX
// has refused a conditional sell carrying tgtCcy, so a sell sends none, as do a
// leg keeping its limit price, a MARGIN order, whose instrument type OKX
// documents no tgtCcy for, and other assets.
func spotAlgoTargetCurrency(s *order.Submit, marketLeg bool) string {
	if s.AssetType == asset.Spot && !spotMarginOrder(s) && s.Side.IsLong() && marketLeg {
		return targetCurrencyBase
	}
	return ""
}

// errAttachedOrderUnsupported is returned for a take-profit, stop-loss or stop entry value OKX cannot attach to the
// order or amendment carrying it
var errAttachedOrderUnsupported = fmt.Errorf("%w: attached take profit or stop loss", common.ErrFunctionNotSupported)

// riskManagementSet reports whether a submission or amendment sets any take-profit, stop-loss or stop entry value
func riskManagementSet(modes *order.RiskManagementModes) bool {
	return *modes != (order.RiskManagementModes{})
}

// checkRiskManagementExtras refuses a stop entry order and a take-profit and stop-loss mode, which OKX has no field for
func checkRiskManagementExtras(modes *order.RiskManagementModes) error {
	if modes.StopEntry != (order.RiskManagement{}) || modes.Mode != "" {
		return fmt.Errorf("%w: OKX takes no stop entry order or take profit and stop loss mode", errAttachedOrderUnsupported)
	}
	return nil
}

// attachedAlgoOrder returns the take-profit and stop-loss legs an order attaches, each placed at its order price once
// its trigger price is crossed, or nil when it attaches neither
func attachedAlgoOrder(modes *order.RiskManagementModes) (*AttachAlgoOrderRequest, error) {
	if err := checkRiskManagementExtras(modes); err != nil {
		return nil, err
	}
	var attached AttachAlgoOrderRequest
	takeProfit, stopLoss := &modes.TakeProfit, &modes.StopLoss
	if *takeProfit != (order.RiskManagement{}) {
		orderPrice, err := attachedLegOrderPrice(takeProfit)
		if err != nil {
			return nil, fmt.Errorf("take profit: %w", err)
		}
		attached.TakeProfitTriggerPrice = takeProfit.Price
		attached.TakeProfitOrderPrice = orderPrice
		attached.TakeProfitTriggerPriceType = priceTypeString(takeProfit.TriggerPriceType)
	}
	if *stopLoss != (order.RiskManagement{}) {
		orderPrice, err := attachedLegOrderPrice(stopLoss)
		if err != nil {
			return nil, fmt.Errorf("stop loss: %w", err)
		}
		attached.StopLossTriggerPrice = stopLoss.Price
		attached.StopLossOrderPrice = orderPrice
		attached.StopLossTriggerPriceType = priceTypeString(stopLoss.TriggerPriceType)
	}
	if attached == (AttachAlgoOrderRequest{}) {
		return nil, nil
	}
	return &attached, nil
}

// attachedLegOrderPrice returns the order price a take-profit or stop-loss leg is placed at once its trigger price is
// crossed: market for a market leg and its limit price for a limit leg. A leg naming no order type is placed at its
// limit price when it has one and at market otherwise, as an OCO order's legs are.
func attachedLegOrderPrice(leg *order.RiskManagement) (float64, error) {
	if leg.Price <= 0 {
		return 0, fmt.Errorf("%w: the trigger price is required", limits.ErrPriceBelowMin)
	}
	switch leg.OrderType {
	case order.Market:
		if leg.LimitPrice != 0 && leg.LimitPrice != marketOrderPrice {
			return 0, fmt.Errorf("%w: a market order takes no limit price", order.ErrUnsupportedOrderType)
		}
		return marketOrderPrice, nil
	case order.Limit:
		if leg.LimitPrice <= 0 {
			return 0, fmt.Errorf("%w: a limit order needs its limit price", limits.ErrPriceBelowMin)
		}
		return leg.LimitPrice, nil
	case order.UnknownType:
		if leg.LimitPrice < 0 && leg.LimitPrice != marketOrderPrice {
			return 0, fmt.Errorf("%w: the limit price must be positive", limits.ErrPriceBelowMin)
		}
		return legOrderPrice(leg.Price, leg.LimitPrice), nil
	}
	return 0, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, leg.OrderType)
}

// amendedAttachedAlgoOrder returns the take-profit and stop-loss legs a trigger order's amendment sends, as
// attachedAlgoOrder returns those an order attaches, or nil when it sends neither
func amendedAttachedAlgoOrder(modes *order.RiskManagementModes) (*AmendAttachedAlgoOrderRequest, error) {
	attached, err := attachedAlgoOrder(modes)
	if attached == nil || err != nil {
		return nil, err
	}
	return &AmendAttachedAlgoOrderRequest{
		NewTakeProfitTriggerPrice:     attached.TakeProfitTriggerPrice,
		NewTakeProfitOrderPrice:       attached.TakeProfitOrderPrice,
		NewTakeProfitTriggerPriceType: attached.TakeProfitTriggerPriceType,
		NewStopLossTriggerPrice:       attached.StopLossTriggerPrice,
		NewStopLossOrderPrice:         attached.StopLossOrderPrice,
		NewStopLossTriggerPriceType:   attached.StopLossTriggerPriceType,
	}, nil
}

// checkTriggerAttachmentAccountLevel refuses a trigger order's take-profit and stop-loss legs in Spot mode, the
// account mode OKX attaches none to a trigger order in
func (e *Exchange) checkTriggerAttachmentAccountLevel(ctx context.Context) error {
	level, err := e.accountLevel(ctx)
	if err != nil {
		return err
	}
	if level == accountLevelSpot {
		return fmt.Errorf("%w: OKX attaches none to a trigger order in Spot mode", errAttachedOrderUnsupported)
	}
	return nil
}

// refuseRiskManagement refuses a take-profit, stop-loss or stop entry value an order of the type cannot carry
func refuseRiskManagement(modes *order.RiskManagementModes, orderType order.Type) error {
	if riskManagementSet(modes) {
		return fmt.Errorf("%w: a %s order carries none", errAttachedOrderUnsupported, orderType)
	}
	return nil
}

// deriveAmendSpreadOrderRequest converts an amendment of a spread order into OKX's spread amendment. A zero price
// leaves the order's price as it is, as an order modification has no other way to state that, so a spread is
// repriced to zero through AmendSpreadOrder alone.
func deriveAmendSpreadOrderRequest(action *order.Modify) (*AmendSpreadOrderRequest, error) {
	if riskManagementSet(&action.RiskManagementModes) {
		return nil, fmt.Errorf("%w: a spread order carries none", errAttachedOrderUnsupported)
	}
	amendRequest := &AmendSpreadOrderRequest{
		OrderID:       action.OrderID,
		ClientOrderID: action.ClientOrderID,
		NewSize:       action.Amount,
	}
	if action.Price != 0 {
		amendRequest.NewPrice = new(action.Price)
	}
	return amendRequest, nil
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
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	p, err := e.checkSubmitPrelude(s)
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
		orderID, err := placedOrderID(spreadOrderOutcome(placeSpreadOrderResponse))
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(orderID)
	}
	orderTypeStr, err := orderTypeString(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	var advanceOrderType string
	if orderTypeStr == orderTrigger {
		if advanceOrderType, err = triggerAdvanceOrderType(s.TimeInForce, s.Price); err != nil {
			return nil, err
		}
	}
	if err := e.resolveSubmitAccount(ctx, s, p); err != nil {
		return nil, err
	}
	// Algo orders carry posSide only for futures and perpetual swap, where the
	// account's position mode decides it. OKX only honours reduceOnly for
	// MARGIN orders and FUTURES/SWAP orders in net mode; in long/short mode
	// the position side expresses the close intent instead.
	var positionSide string
	reduceOnly := s.ReduceOnly
	if s.AssetType == asset.Futures || s.AssetType == asset.PerpetualSwap {
		positionSide = positionSideForMode(s.Side, s.ReduceOnly, p.positionMode)
		reduceOnly = s.ReduceOnly && p.positionMode == positionModeNet
	}
	var result *AlgoOrderActionResponse
	switch orderTypeStr {
	case orderLimit, orderMarket, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly:
		orderRequest := derivePlaceOrderRequest(s, p, orderTypeStr)
		attached, err := attachedAlgoOrder(&s.RiskManagementModes)
		if err != nil {
			return nil, err
		}
		if attached != nil {
			if (orderRequest.TargetCurrency == targetCurrencyBase && s.Side.IsLong()) || (orderRequest.TargetCurrency == targetCurrencyQuote && s.Side.IsShort()) {
				return nil, fmt.Errorf("%w: OKX attaches none to a market %s sized by %s", errAttachedOrderUnsupported, s.Side, orderRequest.TargetCurrency)
			}
			orderRequest.AttachAlgoOrders = []AttachAlgoOrderRequest{*attached}
		}
		placeOrderResponse, err := e.PlaceOrder(ctx, orderRequest)
		if err != nil {
			return nil, err
		}
		orderID, err := placedOrderID(orderActionOutcome(placeOrderResponse))
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(orderID)
	case orderTrigger:
		if s.Price == 0 {
			// OKX requires orderPx on trigger orders: -1 submits a market
			// order when triggered, any other value is the limit price.
			return nil, fmt.Errorf("%w, order price is required, -1 submits a market order when triggered", limits.ErrPriceBelowMin)
		}
		var attached *AttachAlgoOrderRequest
		if attached, err = attachedAlgoOrder(&s.RiskManagementModes); err != nil {
			return nil, err
		}
		var attachedOrders []AttachAlgoOrderRequest
		if attached != nil {
			if err := e.checkTriggerAttachmentAccountLevel(ctx); err != nil {
				return nil, err
			}
			attachedOrders = []AttachAlgoOrderRequest{*attached}
		}
		result, err = e.PlaceTriggerAlgoOrder(ctx, &PlaceAlgoOrderRequest{
			InstrumentID:      p.pairString,
			AlgoClientOrderID: s.ClientOrderID,
			TradeMode:         p.tradeMode,
			Currency:          p.marginCurrency,
			Side:              p.sideType,
			PositionSide:      positionSide,
			OrderType:         orderTypeStr,
			Size:              algoOrderSize(s),
			ReduceOnly:        reduceOnly,
			TriggerPrice:      s.TriggerPrice,
			OrderPrice:        s.Price,
			AdvanceOrderType:  advanceOrderType,
			TriggerPriceType:  priceTypeString(s.TriggerPriceType),
			AttachAlgoOrders:  attachedOrders,
		})
	case orderConditional:
		if err := refuseRiskManagement(&s.RiskManagementModes, s.Type); err != nil {
			return nil, err
		}
		// Trigger Price and type are used as a stop losss trigger price and type.
		orderPrice := legOrderPrice(s.TriggerPrice, s.Price)
		result, err = e.PlaceTakeProfitStopLossOrder(ctx, &PlaceAlgoOrderRequest{
			InstrumentID:             p.pairString,
			AlgoClientOrderID:        s.ClientOrderID,
			TradeMode:                p.tradeMode,
			Currency:                 p.marginCurrency,
			Side:                     p.sideType,
			PositionSide:             positionSide,
			OrderType:                orderTypeStr,
			Size:                     algoOrderSize(s),
			TargetCurrency:           spotAlgoTargetCurrency(s, orderPrice == marketOrderPrice),
			ReduceOnly:               reduceOnly,
			StopLossTriggerPrice:     s.TriggerPrice,
			StopLossOrderPrice:       orderPrice,
			StopLossTriggerPriceType: priceTypeString(s.TriggerPriceType),
		})
	case orderChase:
		if err := refuseRiskManagement(&s.RiskManagementModes, s.Type); err != nil {
			return nil, err
		}
		// The tracking value is the chase's distance from the best bid or ask, where a zero distance, OKX's default,
		// tracks the best price itself; the maximum chase is an auto-cancel bound order.Submit has no field for.
		chaseType := chaseTypeString(s.TrackingMode)
		switch {
		case s.TrackingValue < 0:
			return nil, fmt.Errorf("%w, tracking value %v is negative", limits.ErrAmountBelowMin, s.TrackingValue)
		case chaseType == "" && (s.TrackingMode != order.UnknownTrackingMode || s.TrackingValue != 0):
			return nil, fmt.Errorf("%w, tracking mode %d", order.ErrUnknownTrackingMode, s.TrackingMode)
		}
		result, err = e.PlaceChaseAlgoOrder(ctx, &PlaceAlgoOrderRequest{
			InstrumentID:      p.pairString,
			AlgoClientOrderID: s.ClientOrderID,
			TradeMode:         p.tradeMode,
			Currency:          p.marginCurrency,
			Side:              p.sideType,
			PositionSide:      positionSide,
			OrderType:         orderTypeStr,
			Size:              s.Amount,
			ReduceOnly:        reduceOnly,
			ChaseType:         chaseType,
			ChaseValue:        s.TrackingValue,
		})
	case orderMoveOrderStop:
		if err := refuseRiskManagement(&s.RiskManagementModes, s.Type); err != nil {
			return nil, err
		}
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
		result, err = e.PlaceTrailingStopOrder(ctx, &PlaceAlgoOrderRequest{
			InstrumentID:      p.pairString,
			AlgoClientOrderID: s.ClientOrderID,
			TradeMode:         p.tradeMode,
			Currency:          p.marginCurrency,
			Side:              p.sideType,
			PositionSide:      positionSide,
			OrderType:         orderTypeStr,
			Size:              algoOrderSize(s),
			ReduceOnly:        reduceOnly,
			CallbackRatio:     callbackRatio,
			CallbackSpread:    callbackSpread,
			ActivePrice:       s.TriggerPrice,
		})
	case orderTWAP:
		if err := refuseRiskManagement(&s.RiskManagementModes, s.Type); err != nil {
			return nil, err
		}
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
		result, err = e.PlaceTWAPOrder(ctx, &PlaceAlgoOrderRequest{
			InstrumentID:      p.pairString,
			AlgoClientOrderID: s.ClientOrderID,
			TradeMode:         p.tradeMode,
			Currency:          p.marginCurrency,
			Side:              p.sideType,
			PositionSide:      positionSide,
			OrderType:         orderTypeStr,
			Size:              s.Amount,
			ReduceOnly:        reduceOnly,
			PriceVariance:     priceVar,
			PriceSpread:       priceSpread,
			SizeLimit:         s.Amount,
			PriceLimit:        s.Price,
			// order.Submit has no interval field, so the wrapper keeps the 15
			// minute default, sent in seconds as OKX's timeInterval takes it.
			TimeInterval: uint64(kline.FifteenMin.Duration().Seconds()),
		})
	case orderOCO:
		if err := checkRiskManagementExtras(&s.RiskManagementModes); err != nil {
			return nil, err
		}
		var takeProfitOrderPrice, stopLossOrderPrice float64
		if takeProfitOrderPrice, stopLossOrderPrice, err = ocoLegOrderPrices(&s.RiskManagementModes); err != nil {
			return nil, err
		}
		if (takeProfitOrderPrice == marketOrderPrice) != (stopLossOrderPrice == marketOrderPrice) && s.Side.IsLong() && (s.AssetType == asset.Spot || s.AssetType == asset.Margin) {
			// OKX reads a market buy's size in the quote currency and a limit buy's in the base currency, so one size
			// cannot serve both legs
			return nil, fmt.Errorf("%w: an OCO buy with one leg at market and one at a limit price", order.ErrUnknownSubmissionAmountType)
		}
		result, err = e.PlaceAlgoOrder(ctx, &PlaceAlgoOrderRequest{
			InstrumentID:      p.pairString,
			AlgoClientOrderID: s.ClientOrderID,
			TradeMode:         p.tradeMode,
			Currency:          p.marginCurrency,
			Side:              p.sideType,
			PositionSide:      positionSide,
			OrderType:         orderTypeStr,
			Size:              algoOrderSize(s),
			ReduceOnly:        reduceOnly,

			TakeProfitTriggerPrice:     s.RiskManagementModes.TakeProfit.Price,
			TakeProfitOrderPrice:       takeProfitOrderPrice,
			TakeProfitTriggerPriceType: priceTypeString(s.RiskManagementModes.TakeProfit.TriggerPriceType),

			StopLossTriggerPrice:     s.RiskManagementModes.StopLoss.Price,
			StopLossOrderPrice:       stopLossOrderPrice,
			StopLossTriggerPriceType: priceTypeString(s.RiskManagementModes.StopLoss.TriggerPriceType),
		})
	default:
		return nil, fmt.Errorf("%w, order type %s", order.ErrTypeIsInvalid, orderTypeStr)
	}
	if err != nil {
		return nil, err
	}
	algoID, err := placedOrderID(algoOrderOutcome(result))
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(algoID)
}

// contractPositionMode returns the account's contract position mode,
// fetching and caching it on first use: net mode and long/short mode place
// futures and perpetual swap orders differently. The cache is keyed by the
// API key the request acts for, as a request can carry another account's
// credentials in its context. A fetch that overlaps a confirmed switch errors
// rather than report a mode that may predate it, so the next order fetches
// again.
func (e *Exchange) contractPositionMode(ctx context.Context) (string, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return "", err
	}
	apiKey := e.positionModeKey(ctx)
	e.accountPositionModeMu.RLock()
	mode := e.accountPositionModes[apiKey]
	switches := e.accountPositionModeSwitches
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
	if level := accountConfig.AccountLevel.Int64(); level > 0 {
		e.cacheAccountLevel(apiKey, switches, uint64(level))
	}
	settled, ok := e.cachePositionMode(apiKey, switches, accountConfig.PositionMode)
	if !ok {
		// A switch through another key, which may belong to the same account,
		// landed while the fetch was in flight, so its answer may predate the
		// switch.
		return "", errPositionModeSwitchedDuringFetch
	}
	return settled, nil
}

// cachePositionMode caches a mode fetched for apiKey unless a switch landed
// after switches was read, reporting the mode to use and whether it is
// settled. A switch confirmed for apiKey itself is newer than the fetched
// mode, so it is kept.
func (e *Exchange) cachePositionMode(apiKey string, switches uint64, fetched string) (string, bool) {
	e.accountPositionModeMu.Lock()
	defer e.accountPositionModeMu.Unlock()
	if confirmed := e.accountPositionModes[apiKey]; confirmed != "" {
		return confirmed, true
	}
	if e.accountPositionModeSwitches != switches {
		return "", false
	}
	e.accountPositionModes[apiKey] = fetched
	return fetched, true
}

// accountLevel returns the account level (acctLv) of the account a request
// acts for, from the account configuration contractPositionMode reads the
// position mode from, fetching it on first use and caching both: OKX requires
// the margin currency on cross MARGIN orders in Futures mode. As with the
// position mode, a fetch that overlaps a confirmed switch errors rather than
// report a level that may predate it.
func (e *Exchange) accountLevel(ctx context.Context) (uint64, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return 0, err
	}
	apiKey := e.positionModeKey(ctx)
	e.accountPositionModeMu.RLock()
	level := e.accountLevels[apiKey]
	switches := e.accountPositionModeSwitches
	e.accountPositionModeMu.RUnlock()
	if level != 0 {
		return level, nil
	}
	accountConfig, err := e.GetAccountConfiguration(ctx)
	if err != nil {
		return 0, fmt.Errorf("error fetching the account level: %w", err)
	}
	if accountConfig == nil {
		return 0, fmt.Errorf("error fetching the account level: %w", common.ErrNoResponse)
	}
	fetched := accountConfig.AccountLevel.Int64()
	if fetched <= 0 {
		return 0, fmt.Errorf("%w %q", errInvalidAccountLevel, accountConfig.AccountLevel)
	}
	if accountConfig.PositionMode == positionModeNet || accountConfig.PositionMode == positionModeLongShort {
		e.cachePositionMode(apiKey, switches, accountConfig.PositionMode)
	}
	settled, ok := e.cacheAccountLevel(apiKey, switches, uint64(fetched))
	if !ok {
		return 0, errAccountConfigurationSwitchedDuringFetch
	}
	return settled, nil
}

// cacheAccountLevel caches a level fetched for apiKey unless a switch landed
// after switches was read, reporting the level to use and whether it is
// settled. A switch confirmed for apiKey itself is newer than the fetched
// level, so it is kept.
func (e *Exchange) cacheAccountLevel(apiKey string, switches, fetched uint64) (uint64, bool) {
	e.accountPositionModeMu.Lock()
	defer e.accountPositionModeMu.Unlock()
	if confirmed := e.accountLevels[apiKey]; confirmed != 0 {
		return confirmed, true
	}
	if e.accountPositionModeSwitches != switches {
		return 0, false
	}
	e.accountLevels[apiKey] = fetched
	return fetched, true
}

// positionModeKey returns the API key of the account a request acts for,
// from the request's context credentials or the exchange's own, keying that
// account's cached position mode. Without usable credentials the account
// configuration request fails on its own, so an empty key is returned.
func (e *Exchange) positionModeKey(ctx context.Context) string {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return ""
	}
	return creds.Key
}

// chaseTypeString maps the tracking mode to OKX's chaseType values, which name
// a ratio rather than the percentage the tracking mode string carries.
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

// marginTypeToString returns the OKX trade mode for a single margin type, and
// empty for anything else: a combination of margin types names no mode.
func (e *Exchange) marginTypeToString(m margin.Type) string {
	switch m {
	case margin.Isolated:
		return TradeModeIsolated
	case margin.NoMargin:
		return TradeModeCash
	case margin.SpotIsolated:
		return TradeModeSpotIsolated
	case margin.Multi:
		return TradeModeCross
	default:
		return ""
	}
}

// amendResponse returns the response to an amendment OKX's reply row confirms,
// claiming only what the amendment sent: the order's new size and prices and,
// but for a spread, its instrument. An amendment sends no side, type, time in
// force or asset type, none of which OKX amends, so the response states none.
// The order ID in OKX's reply identifies an order amended by client order ID
// alone.
func amendResponse(action *order.Modify, row *orderOutcome) (*order.ModifyResponse, error) {
	orderID, err := amendedOrderID(row, action.OrderID, action.ClientOrderID)
	if err != nil {
		return nil, err
	}
	resp := &order.ModifyResponse{
		Exchange:      action.Exchange,
		OrderID:       orderID,
		ClientOrderID: action.ClientOrderID,
		Pair:          action.Pair,
		Price:         action.Price,
		Amount:        action.Amount,
		TriggerPrice:  action.TriggerPrice,
	}
	if action.AssetType == asset.Spread {
		resp.Pair = currency.EMPTYPAIR
	}
	switch action.Type {
	case order.Trigger, order.ConditionalStop:
	case order.OCO:
		// An OCO amendment moves its legs' prices, which the response has no
		// fields for.
		resp.Price, resp.TriggerPrice = 0, 0
	default:
		resp.TriggerPrice = 0
	}
	return resp, nil
}

// errReplyOrderMismatch reports reply rows which do not match the orders an
// order operation requested
var errReplyOrderMismatch = fmt.Errorf("%w: reply does not match the order requested", common.ErrInvalidResponse)

// errDuplicateReplyOrderID reports reply rows giving one order ID to orders
// which cannot share it, such as two orders of a batch placement or the orders
// of two client order IDs
var errDuplicateReplyOrderID = fmt.Errorf("%w: one order ID answers distinct orders", errReplyOrderMismatch)

// orderOutcome is what a row of OKX's reply to an order operation reports: the
// order it names, an algo order by its algo IDs, and the operation's outcome
type orderOutcome struct {
	orderID       string
	clientOrderID string
	status        error // nil for success
}

// orderActionOutcome returns what an order action's reply row reports, nil for
// no row
func orderActionOutcome(row *OrderActionResponse) *orderOutcome {
	if row == nil {
		return nil
	}
	return &orderOutcome{orderID: row.OrderID, clientOrderID: row.ClientOrderID, status: row.Error()}
}

// spreadOrderOutcome returns what a spread order action's reply row reports,
// nil for no row
func spreadOrderOutcome(row *SpreadOrderResponse) *orderOutcome {
	if row == nil {
		return nil
	}
	return &orderOutcome{orderID: row.OrderID, clientOrderID: row.ClientOrderID, status: row.Error()}
}

// algoOrderOutcome returns what an algo order placement's or cancellation's
// reply row reports, nil for no row
func algoOrderOutcome(row *AlgoOrderActionResponse) *orderOutcome {
	if row == nil {
		return nil
	}
	return &orderOutcome{
		orderID:       row.AlgoID,
		clientOrderID: row.AlgoClientOrderID,
		status:        replyRowStatusError(row.StatusCode, row.StatusMessage, row.statusCodeMissing),
	}
}

// amendAlgoOrderOutcome returns what an algo order amendment's reply row
// reports, nil for no row
func amendAlgoOrderOutcome(row *AmendAlgoOrderResponse) *orderOutcome {
	if row == nil {
		return nil
	}
	return &orderOutcome{
		orderID:       row.AlgoID,
		clientOrderID: row.AlgoClientOrderID,
		status:        replyRowStatusError(row.StatusCode, row.StatusMessage, row.statusCodeMissing),
	}
}

// placedOrderID returns the ID of the order a placement's reply row confirms
// placed. OKX can give an order it rejects an ID, so the row's status decides
// before its ID does.
func placedOrderID(row *orderOutcome) (string, error) {
	switch {
	case row == nil:
		return "", fmt.Errorf("%w: placement reply has no row", common.ErrInvalidResponse)
	case row.status != nil:
		return "", row.status
	case row.orderID == "":
		return "", fmt.Errorf("%w: placement reply has no order ID", common.ErrInvalidResponse)
	}
	return row.orderID, nil
}

// amendedOrderID returns the ID of the order an amendment's reply row confirms
// amending: OKX amends the order the amendment's order ID names, or its client
// order ID without one, so a row naming another order confirms nothing.
func amendedOrderID(row *orderOutcome, orderID, clientOrderID string) (string, error) {
	switch {
	case row == nil:
		return "", fmt.Errorf("%w: amend reply has no row for order ID %q client order ID %q", common.ErrInvalidResponse, orderID, clientOrderID)
	case row.status != nil:
		return "", row.status
	case orderID != "" && row.orderID != orderID:
		return "", fmt.Errorf("%w: amend reply names order ID %q amending order ID %q", errReplyOrderMismatch, row.orderID, orderID)
	case orderID == "" && row.clientOrderID != clientOrderID:
		return "", fmt.Errorf("%w: amend reply names client order ID %q amending client order ID %q", errReplyOrderMismatch, row.clientOrderID, clientOrderID)
	case row.orderID == "":
		return "", fmt.Errorf("%w: amend reply has no order ID for client order ID %q", common.ErrInvalidResponse, clientOrderID)
	}
	return row.orderID, nil
}

// ModifyOrder modifies an existing order via the exchange REST API.
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if action.AssetType == asset.Spread {
		amendRequest, err := deriveAmendSpreadOrderRequest(action)
		if err != nil {
			return nil, err
		}
		reply, err := e.AmendSpreadOrder(ctx, amendRequest)
		if err != nil {
			return nil, err
		}
		return amendResponse(action, spreadOrderOutcome(reply))
	}

	pairFormat, err := e.GetPairFormat(action.AssetType, true)
	if err != nil {
		return nil, err
	}
	if action.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(action.Pair)
	var row *orderOutcome
	switch action.Type {
	case order.UnknownType, order.Market, order.Limit, order.LimitMaker, order.OptimalLimit, order.MarketMakerProtection:
		if riskManagementSet(&action.RiskManagementModes) {
			return nil, fmt.Errorf("%w: OKX amends one by its attachAlgoId, which an order modification cannot name", errAttachedOrderUnsupported)
		}
		var reply *OrderActionResponse
		reply, err = e.AmendOrder(ctx, &AmendOrderRequest{
			InstrumentID:  instrumentID,
			NewSize:       action.Amount,
			OrderID:       action.OrderID,
			ClientOrderID: action.ClientOrderID,
			NewPrice:      action.Price,
		})
		if err != nil {
			return nil, err
		}
		row = orderActionOutcome(reply)
	case order.Trigger, order.ConditionalStop, order.OCO:
		var algoAmend *AmendAlgoOrderRequest
		if algoAmend, err = deriveAmendAlgoOrderRequest(action); err != nil {
			return nil, err
		}
		if len(algoAmend.AttachAlgoOrders) != 0 {
			if err := e.checkTriggerAttachmentAccountLevel(ctx); err != nil {
				return nil, err
			}
		}
		algoAmend.InstrumentID = instrumentID
		var reply *AmendAlgoOrderResponse
		if reply, err = e.AmendAlgoOrder(ctx, algoAmend); err != nil {
			return nil, err
		}
		row = amendAlgoOrderOutcome(reply)
	default:
		return nil, fmt.Errorf("%w, could not amend order of type %v", order.ErrUnsupportedOrderType, action.Type)
	}
	return amendResponse(action, row)
}

// deriveAmendAlgoOrderRequest converts an amendment of a trigger, conditional
// or OCO order into an algo order amendment without its instrument ID.
func deriveAmendAlgoOrderRequest(action *order.Modify) (*AmendAlgoOrderRequest, error) {
	algoAmend := &AmendAlgoOrderRequest{
		AlgoID:            action.OrderID,
		AlgoClientOrderID: action.ClientOrderID,
		NewSize:           action.Amount,
	}
	takeProfit, stopLoss := action.RiskManagementModes.TakeProfit, action.RiskManagementModes.StopLoss
	switch action.Type {
	case order.Trigger:
		if action.TriggerPrice == 0 {
			return nil, fmt.Errorf("%w, trigger price required", limits.ErrPriceBelowMin)
		}
		if action.Price == 0 {
			// OKX requires newOrdPx beside newTriggerPx, and reading an unset
			// price as -1 would turn a limit trigger into a market one
			return nil, fmt.Errorf("%w, order price is required, -1 amends to a market order when triggered", limits.ErrPriceBelowMin)
		}
		algoAmend.NewTriggerPrice = action.TriggerPrice
		algoAmend.NewOrderPrice = action.Price
		algoAmend.NewTriggerPriceType = priceTypeString(action.TriggerPriceType)
		attached, err := amendedAttachedAlgoOrder(&action.RiskManagementModes)
		if err != nil {
			return nil, err
		}
		if attached != nil {
			algoAmend.AttachAlgoOrders = []AmendAttachedAlgoOrderRequest{*attached}
		}
	case order.ConditionalStop:
		if err := refuseRiskManagement(&action.RiskManagementModes, action.Type); err != nil {
			return nil, err
		}
		// SubmitOrder places a conditional order's stop-loss leg from the
		// trigger price and price, which amend it in turn.
		if action.Amount <= 0 && action.TriggerPrice <= 0 && action.Price == 0 {
			return nil, fmt.Errorf("%w: a conditional order amends its size, trigger price or order price", errInvalidNewSizeOrPriceInformation)
		}
		algoAmend.NewStopLossTriggerPrice = action.TriggerPrice
		algoAmend.NewStopLossOrderPrice = action.Price
		algoAmend.NewStopLossTriggerPriceType = amendedTriggerPriceType(action.TriggerPrice, action.TriggerPriceType)
	case order.OCO:
		if err := checkRiskManagementExtras(&action.RiskManagementModes); err != nil {
			return nil, err
		}
		legsUnchanged := takeProfit.Price == 0 && takeProfit.LimitPrice == 0 && stopLoss.Price == 0 && stopLoss.LimitPrice == 0
		if action.Amount <= 0 || !legsUnchanged {
			switch {
			case takeProfit.Price <= 0 && takeProfit.LimitPrice <= 0:
				return nil, fmt.Errorf("%w, either take profit trigger price or order price is required", limits.ErrPriceBelowMin)
			case stopLoss.Price <= 0 && stopLoss.LimitPrice <= 0:
				return nil, fmt.Errorf("%w, either stop loss trigger price or order price is required", limits.ErrPriceBelowMin)
			}
		}
		algoAmend.NewTakeProfitTriggerPrice = takeProfit.Price
		algoAmend.NewTakeProfitOrderPrice = takeProfit.LimitPrice
		algoAmend.NewTakeProfitTriggerPriceType = amendedTriggerPriceType(takeProfit.Price, takeProfit.TriggerPriceType)
		algoAmend.NewStopLossTriggerPrice = stopLoss.Price
		algoAmend.NewStopLossOrderPrice = stopLoss.LimitPrice
		algoAmend.NewStopLossTriggerPriceType = amendedTriggerPriceType(stopLoss.Price, stopLoss.TriggerPriceType)
	default:
		return nil, fmt.Errorf("%w, could not amend order of type %v", order.ErrUnsupportedOrderType, action.Type)
	}
	return algoAmend, nil
}

// amendedTriggerPriceType returns the trigger price type an amendment sends
// beside a new trigger price, and none without one: the zero price type is the
// last price, so sending it alone would reset a type the order already has.
func amendedTriggerPriceType(triggerPrice float64, pt order.PriceType) string {
	if triggerPrice <= 0 {
		return ""
	}
	return priceTypeString(pt)
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
		cancelled, err := e.CancelSpreadOrder(ctx, &CancelSpreadOrderRequest{OrderID: ord.OrderID, ClientOrderID: ord.ClientOrderID})
		if err != nil {
			return err
		}
		return cancelReplyError(spreadOrderOutcome(cancelled), ord, true)
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
		cancelled, err := e.CancelSingleOrder(ctx, &CancelOrderRequest{
			InstrumentID:  instrumentID,
			OrderID:       ord.OrderID,
			ClientOrderID: ord.ClientOrderID,
		})
		if err != nil {
			return err
		}
		return cancelReplyError(orderActionOutcome(cancelled), ord, true)
	case order.Trigger, order.OCO, order.ConditionalStop, order.TWAP, order.TrailingStop, order.Chase:
		response, err := e.CancelAlgoOrder(ctx, []CancelAlgoOrderRequest{
			{
				InstrumentID:      instrumentID,
				AlgoID:            ord.OrderID,
				AlgoClientOrderID: ord.ClientOrderID,
			},
		})
		if err != nil {
			return err
		}
		// OKX answers an accepted algo cancel with one row for the one order
		// requested
		switch {
		case len(response) == 0:
			return fmt.Errorf("%w for algo order ID %q client order ID %q", common.ErrNoResponse, ord.OrderID, ord.ClientOrderID)
		case len(response) > 1:
			return fmt.Errorf("%w: %d rows cancelling algo order ID %q client order ID %q", common.ErrInvalidResponse, len(response), ord.OrderID, ord.ClientOrderID)
		}
		return cancelReplyError(algoOrderOutcome(&response[0]), ord, false)
	default:
		return fmt.Errorf("%w, order type %v", order.ErrUnsupportedOrderType, ord.Type)
	}
}

// cancelReplyError returns the outcome a single cancel's reply row reports.
// OKX acknowledges a cancel with the order's ID and sCode 0, so a row without
// them, or naming another order than the one requested, leaves the cancel
// unconfirmed. A cancel naming its order by client order ID alone is matched
// by the client order ID the row echoes, which must be present when
// echoesClientOrderID is set: an algo cancel's reply may leave it out, as OKX
// has deprecated its algoClOrdId.
func cancelReplyError(row *orderOutcome, c *order.Cancel, echoesClientOrderID bool) error {
	switch {
	case row == nil:
		return fmt.Errorf("%w cancelling order ID %q client order ID %q", common.ErrNoResponse, c.OrderID, c.ClientOrderID)
	case row.status != nil:
		return row.status
	case row.orderID == "":
		return fmt.Errorf("%w: no order ID cancelling order ID %q client order ID %q", common.ErrInvalidResponse, c.OrderID, c.ClientOrderID)
	case c.OrderID != "" && row.orderID != c.OrderID,
		c.OrderID == "" && row.clientOrderID != c.ClientOrderID && (echoesClientOrderID || row.clientOrderID != ""):
		return fmt.Errorf("%w: reply names order ID %q client order ID %q cancelling order ID %q client order ID %q", errReplyOrderMismatch, row.orderID, row.clientOrderID, c.OrderID, c.ClientOrderID)
	}
	return nil
}

// maxCancelAlgosPerRequest is the documented maximum number of algo orders
// POST /api/v5/trade/cancel-algos accepts per request.
const maxCancelAlgosPerRequest = 10

// errSpreadBatchCancelUnsupported reports that spread orders cannot join a
// batch cancel: OKX takes each spread cancel as its own request, so the batch
// endpoint carries nothing to batch for them.
var errSpreadBatchCancelUnsupported = fmt.Errorf("%w: spread orders cancel one request at a time", common.ErrFunctionNotSupported)

// CancelBatchOrders cancels orders by their corresponding ID numbers via the
// exchange REST API. Ordinary orders cancel in one request of at most 20 and
// algo orders in one request of at most maxCancelAlgosPerRequest; spread
// orders cannot batch, so they are rejected. An algo order group is still
// attempted after an ordinary order group fails, while the context is live.
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if len(o) > 20 {
		return nil, fmt.Errorf("%w, cannot cancel more than 20 orders", errExceedLimit)
	} else if len(o) == 0 {
		return nil, fmt.Errorf("%w, must have at least 1 cancel order", order.ErrCancelOrderIsNil)
	}
	cancelOrderParams := make([]CancelOrderRequest, 0, len(o))
	cancelAlgoOrderParams := make([]CancelAlgoOrderRequest, 0, len(o))
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	// The whole batch is validated before any cancel is sent, so an invalid
	// entry cannot leave a partially executed batch behind.
	for x := range o {
		ord := o[x]
		if !e.SupportsAsset(ord.AssetType) {
			return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
		}
		if ord.AssetType == asset.Spread {
			return nil, errSpreadBatchCancelUnsupported
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
			if ord.ClientOrderID == "" && ord.OrderID == "" {
				return nil, fmt.Errorf("%w, order ID required for order of type %v", order.ErrOrderIDNotSet, ord.Type)
			}
			cancelOrderParams = append(cancelOrderParams, CancelOrderRequest{
				InstrumentID:  pairFormat.Format(ord.Pair),
				OrderID:       ord.OrderID,
				ClientOrderID: ord.ClientOrderID,
			})
		case order.Trigger, order.OCO, order.ConditionalStop,
			order.TWAP, order.TrailingStop, order.Chase:
			if ord.OrderID == "" && ord.ClientOrderID == "" {
				return nil, fmt.Errorf("%w, order ID required for order of type %v", order.ErrOrderIDNotSet, ord.Type)
			}
			cancelAlgoOrderParams = append(cancelAlgoOrderParams, CancelAlgoOrderRequest{
				InstrumentID:      pairFormat.Format(ord.Pair),
				AlgoID:            ord.OrderID,
				AlgoClientOrderID: ord.ClientOrderID,
			})
		default:
			return nil, fmt.Errorf("%w order of type %v not supported", order.ErrUnsupportedOrderType, ord.Type)
		}
	}
	if len(cancelAlgoOrderParams) > maxCancelAlgosPerRequest {
		return nil, fmt.Errorf("%w, cannot cancel more than %d algo orders", errExceedLimit, maxCancelAlgosPerRequest)
	}
	// Cancels from here on execute, so a failure returns resp with the
	// statuses already recorded rather than discarding them.
	var errs error
	if len(cancelOrderParams) > 0 {
		canceledOrders, err := e.CancelMultipleOrders(ctx, cancelOrderParams)
		errs = err
		if cancelResultsUsable(err) {
			errs = common.AppendError(errs, recordOrderCancels(resp.Status, cancelOrderParams, canceledOrders))
		}
	}
	if len(cancelAlgoOrderParams) > 0 {
		if ctx.Err() != nil {
			return resp, withContextError(ctx, errs)
		}
		algoResults, err := e.CancelAlgoOrder(ctx, cancelAlgoOrderParams)
		errs = common.AppendError(errs, err)
		if cancelResultsUsable(err) {
			errs = common.AppendError(errs, recordAlgoCancels(resp.Status, cancelAlgoOrderParams, algoResults))
		}
	}
	return resp, errs
}

// checkSpreadMassCancelFilters returns common.ErrFunctionNotSupported for a
// cancel-all filter OKX's spread mass cancel cannot apply: it cancels every
// pending order of a spread, or of every spread, whatever order ID, client
// order ID, side, order type, time in force or margin type the cancellation
// names. A pair naming one currency is refused too, as no spread ID would
// scope the cancel to every spread.
func checkSpreadMassCancelFilters(c *order.Cancel) error {
	switch {
	case c.OrderID != "" || c.ClientOrderID != "":
		return fmt.Errorf("%w: spread mass cancel cannot filter by order ID or client order ID", common.ErrFunctionNotSupported)
	case c.Side != order.UnknownSide && c.Side != order.AnySide:
		return fmt.Errorf("%w: spread mass cancel cannot filter by side %s", common.ErrFunctionNotSupported, c.Side)
	case c.Type != order.UnknownType && c.Type != order.AnyType:
		return fmt.Errorf("%w: spread mass cancel cannot filter by order type %s", common.ErrFunctionNotSupported, c.Type)
	case c.TimeInForce != order.UnknownTIF:
		return fmt.Errorf("%w: spread mass cancel cannot filter by time in force %s", common.ErrFunctionNotSupported, c.TimeInForce)
	case c.MarginType != margin.Unset:
		return fmt.Errorf("%w: spread mass cancel cannot filter by margin type %s", common.ErrFunctionNotSupported, c.MarginType)
	case !c.Pair.IsEmpty() && !c.Pair.IsPopulated():
		return fmt.Errorf("%w: pair %q names one currency", currency.ErrCurrencyCodeEmpty, c.Pair)
	}
	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair via the
// exchange REST API: the pending ordinary orders and the untriggered algo
// orders which the cancellation selects, icebergs included, or a spread's
// orders through the spread mass cancel. asset.All and an unset asset cancel
// the orders of every instrument type but spreads, which asset.Spread cancels.
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	err = orderCancellation.Validate()
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
		if filterErr := checkSpreadMassCancelFilters(orderCancellation); filterErr != nil {
			return cancelAllResponse, filterErr
		}
		var spreadID string
		if orderCancellation.Pair.IsPopulated() {
			spreadID = orderCancellation.Pair.Upper().String()
		}
		var cancelled *CancelAllSpreadOrdersResponse
		cancelled, err = e.CancelAllSpreadOrders(ctx, &CancelAllSpreadOrdersRequest{SpreadID: spreadID})
		if err != nil {
			return cancelAllResponse, err
		}
		if cancelled == nil {
			return cancelAllResponse, fmt.Errorf("%w cancelling all spread orders for spread ID %q", common.ErrNoResponse, spreadID)
		}
		// The result is keyed by the scope the request sent, since the order ID
		// and client order ID scope nothing on a mass cancel.
		cancelAllResponse.Status[spreadID] = strconv.FormatBool(cancelled.Result)
		return cancelAllResponse, nil
	}

	scope, err := e.newCancelAllScope(orderCancellation)
	if err != nil {
		return cancelAllResponse, err
	}
	var errs error
	if scope.ordinary {
		cancelAllOrdersRequestParams, err := e.pendingOrdersToCancel(ctx, scope)
		errs = err
		if len(cancelAllOrdersRequestParams) != 0 {
			cancelled, err := cancelInBatches(ctx, cancelAllOrdersRequestParams, e.CancelMultipleOrders)
			maps.Copy(cancelAllResponse.Status, cancelled.Status)
			errs = common.AppendError(errs, err)
			if scope.idScoped() {
				return cancelAllResponse, errs
			}
		}
	}
	if len(scope.algoTypes) == 0 {
		return cancelAllResponse, errs
	}
	if ctx.Err() != nil {
		return cancelAllResponse, withContextError(ctx, errs)
	}
	algoParams, err := e.pendingAlgoOrdersToCancel(ctx, scope)
	errs = common.AppendError(errs, err)
	if ctx.Err() != nil {
		return cancelAllResponse, withContextError(ctx, errs)
	}
	errs = common.AppendError(errs, cancelBatches(ctx, algoParams, maxCancelAlgosPerRequest, cancelAllResponse.Status, e.CancelAlgoOrder, recordAlgoCancels))
	return cancelAllResponse, errs
}

// errSpreadWebsocketBatchUnsupported reports that spread orders cannot join a
// websocket batch: OKX has no batch spread operation.
var errSpreadWebsocketBatchUnsupported = fmt.Errorf("%w: spread orders are placed one request at a time", common.ErrFunctionNotSupported)

// errWebsocketOtherAccount reports a websocket order request for another
// account than the one its connection logged in as, which OKX acts for
var errWebsocketOtherAccount = fmt.Errorf("%w: a websocket operation acts for the account its connection logged in as, so a request for another account must use REST", common.ErrFunctionNotSupported)

// checkWebsocketAccount refuses a request for another account than the one the
// connection the request uses logged in as, before any account state such as
// the position mode is read for it. An unavailable connection is left to the
// operation, which reports it when sending.
func (e *Exchange) checkWebsocketAccount(ctx context.Context, connection string) error {
	if conn, err := e.Websocket.GetConnection(connection); err == nil {
		return e.checkWebsocketLogin(ctx, conn)
	}
	return nil
}

// withAccountCredentials returns ctx carrying a private copy of the credentials
// the request acts for, so that every account lookup and request of a
// compound operation acts for the one account even if the exchange's own
// credentials, or a credentials store the caller's context carries, change
// meanwhile. The error of credentials which cannot be used is returned, as
// those they become while the operation runs may be another account's.
// Disabled authentication is left to the requests, which refuse it, as
// context credentials would bypass the setting.
func (e *Exchange) withAccountCredentials(ctx context.Context) (context.Context, error) {
	creds, err := e.GetCredentials(ctx)
	switch {
	case errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled):
		return ctx, nil
	case err != nil:
		return ctx, err
	}
	return accounts.DeployCredentialsToContext(ctx, creds), nil
}

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
// orders have no websocket equivalent, and spread orders cannot join a batch.
func (e *Exchange) prepareWebsocketPlaceOrder(ctx context.Context, s *order.Submit) (*PlaceOrderRequest, error) {
	p, err := e.checkSubmitPrelude(s)
	if err != nil {
		return nil, err
	}
	if s.AssetType == asset.Spread {
		return nil, errSpreadWebsocketBatchUnsupported
	}
	if riskManagementSet(&s.RiskManagementModes) {
		return nil, fmt.Errorf("%w: OKX's websocket order operation takes none", errAttachedOrderUnsupported)
	}
	orderTypeStr, err := orderTypeString(s.Type, s.TimeInForce)
	if err != nil {
		return nil, err
	}
	switch orderTypeStr {
	case orderLimit, orderMarket, orderPostOnly, orderFOK, orderIOC, orderOptimalLimitIOC, orderMarketMakerProtection, orderMarketMakerProtectionAndPostOnly:
		code, err := e.requireWebsocketInstrumentIDCode(p.pairString)
		if err != nil {
			return nil, err
		}
		if err := e.resolveSubmitAccount(ctx, s, p); err != nil {
			return nil, err
		}
		orderRequest := derivePlaceOrderRequest(s, p, orderTypeStr)
		orderRequest.InstrumentIDCode = code
		return orderRequest, nil
	default:
		return nil, fmt.Errorf("%w: algo order type %s", common.ErrFunctionNotSupported, orderTypeStr)
	}
}

// WebsocketSubmitOrder submits an order through the authenticated private
// websocket connection, or a spread order through the authenticated business
// connection. Algo order types return common.ErrFunctionNotSupported before
// any request is transmitted.
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	connection := privateConnection
	if s != nil && s.AssetType == asset.Spread {
		connection = businessLoginConnection
	}
	if err := e.checkWebsocketAccount(ctx, connection); err != nil {
		return nil, err
	}
	if s != nil && s.AssetType == asset.Spread {
		p, err := e.validateSubmitPrelude(ctx, s)
		if err != nil {
			return nil, err
		}
		spreadParam, err := deriveSpreadOrderParam(s, p)
		if err != nil {
			return nil, err
		}
		placed, err := e.WebsocketAPI().PlaceSpreadOrder(ctx, spreadParam)
		if err != nil {
			return nil, err
		}
		orderID, err := placedOrderID(spreadOrderOutcome(placed))
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(orderID)
	}
	orderRequest, err := e.prepareWebsocketPlaceOrder(ctx, s)
	if err != nil {
		return nil, err
	}
	placeOrderResponse, err := e.WebsocketAPI().PlaceOrder(ctx, orderRequest)
	if err != nil {
		return nil, err
	}
	orderID, err := placedOrderID(orderActionOutcome(placeOrderResponse))
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(orderID)
}

// WebsocketSubmitOrders submits orders in a single batch-orders frame through
// the authenticated private websocket connection. Every order is validated
// and converted before the batch transmits, so an invalid order cannot leave
// a partially submitted batch behind. Algo order types and spread orders
// return common.ErrFunctionNotSupported before any request is transmitted. An
// order without a client order ID is sent with one of its own, which its
// response reports, so that OKX's reply row for every order is known by the
// client order ID it echoes.
func (e *Exchange) WebsocketSubmitOrders(ctx context.Context, orders []*order.Submit) ([]*order.SubmitResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if err := e.checkWebsocketAccount(ctx, privateConnection); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, fmt.Errorf("%T: %w", orders, order.ErrSubmissionIsNil)
	}
	if len(orders) > 20 {
		return nil, fmt.Errorf("%w, cannot submit more than 20 orders", errExceedLimit)
	}
	args := make([]PlaceOrderRequest, len(orders))
	clientOrderIDs := make(common.Seen[string], len(orders))
	for i := range orders {
		orderRequest, err := e.prepareWebsocketPlaceOrder(ctx, orders[i])
		if err != nil {
			return nil, err
		}
		if orderRequest.ClientOrderID != "" && clientOrderIDs.CheckAndRecord(orderRequest.ClientOrderID) {
			return nil, fmt.Errorf("%w: %q", errDuplicateClientOrderID, orderRequest.ClientOrderID)
		}
		args[i] = *orderRequest
	}
	for i := range args {
		for args[i].ClientOrderID == "" {
			if clientOrderID := e.MessageID(); !clientOrderIDs.CheckAndRecord(clientOrderID) {
				args[i].ClientOrderID = clientOrderID
			}
		}
	}
	placed, err := e.WebsocketAPI().PlaceMultipleOrders(ctx, args)
	if (err == nil || len(placed) != 0) && len(placed) != len(orders) {
		return nil, common.AppendError(err, fmt.Errorf("%w: %d results for %d orders", common.ErrInvalidResponse, len(placed), len(orders)))
	}
	responses := make([]*order.SubmitResponse, len(orders))
	if len(placed) == 0 {
		return responses, err
	}
	// A partial success keeps its transport error alongside the per-order
	// results, matching the batch cancel contract above; every other failure
	// joins it instead of replacing it.
	rows, errs := batchPlacementReplyRows(args, placed)
	errs = common.AppendError(err, errs)
	for i, row := range rows {
		if row == nil {
			continue
		}
		orderID, rowErr := placedOrderID(orderActionOutcome(row))
		if rowErr != nil {
			// A failed or partly failed batch's error states its failed rows'
			// status codes already
			if err == nil || row.Error() == nil {
				errs = common.AppendError(errs, fmt.Errorf("order %d: %w", i+1, rowErr))
			}
			continue
		}
		response, derr := orders[i].DeriveSubmitResponse(orderID)
		if derr != nil {
			errs = common.AppendError(errs, derr)
			continue
		}
		response.ClientOrderID = args[i].ClientOrderID
		responses[i] = response
	}
	return responses, errs
}

// errDuplicateClientOrderID reports a batch giving two orders one client order
// ID, which would leave OKX's replies to them indistinguishable
var errDuplicateClientOrderID = errors.New("duplicate client order ID")

// batchPlacementReplyRows pairs each order of a batch placement with OKX's
// reply row for it: OKX does not promise its rows follow the batch's order, so
// an order takes the one row echoing its client order ID. An order whose row
// is missing, null, repeated or echoes another client order ID is left without
// one, beside an error, as is a row no order takes. An order ID which any two
// of the reply's successful rows give two client order IDs is reported, and an
// order whose successful row names it is left without its row, rather than
// given another order's ID; a rejected row's order ID claims no order, as OKX
// can give an order it rejects an ID.
func batchPlacementReplyRows(args []PlaceOrderRequest, rows []*OrderActionResponse) ([]*OrderActionResponse, error) {
	if len(rows) != len(args) {
		return nil, fmt.Errorf("%w: %d reply rows for %d orders", common.ErrInvalidResponse, len(rows), len(args))
	}
	byClientOrderID := make(map[string][]int, len(rows))
	for i, row := range rows {
		if row != nil && row.ClientOrderID != "" {
			byClientOrderID[row.ClientOrderID] = append(byClientOrderID[row.ClientOrderID], i)
		}
	}
	matched := make([]*OrderActionResponse, len(args))
	taken := make([]bool, len(rows))
	var errs error
	for i := range args {
		clientOrderID := args[i].ClientOrderID
		switch echoed := byClientOrderID[clientOrderID]; len(echoed) {
		case 0:
			errs = common.AppendError(errs, fmt.Errorf("%w: no reply row echoes client order ID %q of order %d", errReplyOrderMismatch, clientOrderID, i+1))
		case 1:
			matched[i], taken[echoed[0]] = rows[echoed[0]], true
		default:
			for _, j := range echoed {
				taken[j] = true
			}
			errs = common.AppendError(errs, fmt.Errorf("%w: %d reply rows echo client order ID %q of order %d", errReplyOrderMismatch, len(echoed), clientOrderID, i+1))
		}
	}
	for j, row := range rows {
		if row != nil && !taken[j] {
			errs = common.AppendError(errs, fmt.Errorf("%w: no order takes reply row %d naming order ID %q client order ID %q", errReplyOrderMismatch, j+1, row.OrderID, row.ClientOrderID))
		}
	}
	claims := make(map[string][]string, len(rows))
	var contested []string
	for _, row := range rows {
		if row == nil || row.OrderID == "" || row.ClientOrderID == "" || row.Error() != nil || slices.Contains(claims[row.OrderID], row.ClientOrderID) {
			continue
		}
		claims[row.OrderID] = append(claims[row.OrderID], row.ClientOrderID)
		if len(claims[row.OrderID]) == 2 {
			contested = append(contested, row.OrderID)
		}
	}
	for _, orderID := range contested {
		errs = common.AppendError(errs, fmt.Errorf("%w: reply rows give order ID %q to client order IDs %q", errDuplicateReplyOrderID, orderID, claims[orderID]))
	}
	for i, row := range matched {
		if row != nil && row.Error() == nil && len(claims[row.OrderID]) > 1 {
			matched[i] = nil
		}
	}
	return matched, errs
}

// WebsocketModifyOrder amends an existing order through the authenticated
// private websocket connection, or a spread order through the authenticated
// business connection. Algo orders return common.ErrFunctionNotSupported
// before any request is transmitted.
func (e *Exchange) WebsocketModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	connection := privateConnection
	if action != nil && action.AssetType == asset.Spread {
		connection = businessLoginConnection
	}
	if err := e.checkWebsocketAccount(ctx, connection); err != nil {
		return nil, err
	}
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if action.AssetType == asset.Spread {
		amendRequest, err := deriveAmendSpreadOrderRequest(action)
		if err != nil {
			return nil, err
		}
		reply, err := e.WebsocketAPI().AmendSpreadOrder(ctx, amendRequest)
		if err != nil {
			return nil, err
		}
		return amendResponse(action, spreadOrderOutcome(reply))
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
		if riskManagementSet(&action.RiskManagementModes) {
			return nil, fmt.Errorf("%w: OKX's websocket amend operation takes none", errAttachedOrderUnsupported)
		}
		code, err := e.requireWebsocketInstrumentIDCode(instrumentID)
		if err != nil {
			return nil, err
		}
		reply, err := e.WebsocketAPI().AmendOrder(ctx, &AmendOrderRequest{
			InstrumentID:     instrumentID,
			InstrumentIDCode: code,
			NewSize:          action.Amount,
			OrderID:          action.OrderID,
			ClientOrderID:    action.ClientOrderID,
			NewPrice:         action.Price,
		})
		if err != nil {
			return nil, err
		}
		return amendResponse(action, orderActionOutcome(reply))
	default:
		return nil, fmt.Errorf("%w: amend of order type %v", common.ErrFunctionNotSupported, action.Type)
	}
}

// WebsocketCancelOrder cancels an order through the authenticated private
// websocket connection, or a spread order through the authenticated business
// connection. Algo orders return common.ErrFunctionNotSupported before any
// request is transmitted.
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, ord *order.Cancel) error {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return err
	}
	connection := privateConnection
	if ord != nil && ord.AssetType == asset.Spread {
		connection = businessLoginConnection
	}
	if err := e.checkWebsocketAccount(ctx, connection); err != nil {
		return err
	}
	if ord == nil {
		return order.ErrCancelOrderIsNil
	}
	if !e.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	if ord.AssetType == asset.Spread {
		cancelled, err := e.WebsocketAPI().CancelSpreadOrder(ctx, ord.OrderID, ord.ClientOrderID)
		if err != nil {
			return err
		}
		return cancelReplyError(spreadOrderOutcome(cancelled), ord, true)
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
		cancelled, err := e.WebsocketAPI().CancelSingleOrder(ctx, &CancelOrderRequest{
			InstrumentID:     instrumentID,
			InstrumentIDCode: code,
			OrderID:          ord.OrderID,
			ClientOrderID:    ord.ClientOrderID,
		})
		if err != nil {
			return err
		}
		return cancelReplyError(orderActionOutcome(cancelled), ord, true)
	default:
		return fmt.Errorf("%w: cancel of order type %v", common.ErrFunctionNotSupported, ord.Type)
	}
}

// WebsocketCancelBatchOrders cancels a batch of orders through the
// authenticated private websocket connection. Every order is validated and
// converted before the batch transmits, so an invalid order cannot leave a
// partially executed batch behind. Algo order types and spread orders, which
// OKX cannot cancel in a batch, return common.ErrFunctionNotSupported before
// any request is transmitted.
func (e *Exchange) WebsocketCancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if err := e.checkWebsocketAccount(ctx, privateConnection); err != nil {
		return nil, err
	}
	if len(o) > 20 {
		return nil, fmt.Errorf("%w, cannot cancel more than 20 orders", errExceedLimit)
	}
	if len(o) == 0 {
		return nil, fmt.Errorf("%w, must have at least 1 cancel order", order.ErrCancelOrderIsNil)
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	args := make([]CancelOrderRequest, 0, len(o))
	for x := range o {
		ord := o[x]
		if !e.SupportsAsset(ord.AssetType) {
			return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
		}
		if ord.AssetType == asset.Spread {
			return nil, errSpreadBatchCancelUnsupported
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
			args = append(args, CancelOrderRequest{
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
	cancelled, err := e.WebsocketAPI().CancelMultipleOrders(ctx, args)
	if cancelResultsUsable(err) {
		err = common.AppendError(err, recordOrderCancels(resp.Status, args, cancelled))
	}
	return resp, err
}

// errWebsocketAlgoCancelUnsupported reports untriggered algo orders a
// websocket cancel-all selects but cannot cancel: OKX cancels algo orders over
// REST only
var errWebsocketAlgoCancelUnsupported = fmt.Errorf("%w: algo orders cannot be cancelled over the websocket, cancel them over REST", common.ErrFunctionNotSupported)

// WebsocketCancelAllOrders cancels all orders associated with a currency pair
// through the authenticated private websocket connection, or the spread's
// orders through the authenticated business connection. OKX cancels no algo
// order over the websocket: algo order types return
// common.ErrFunctionNotSupported before any request is transmitted, and the
// untriggered algo orders a cancellation of every order type selects are
// named in an errWebsocketAlgoCancelUnsupported error beside the results.
func (e *Exchange) WebsocketCancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	ctx, err := e.withAccountCredentials(ctx)
	if err != nil {
		return order.CancelAllResponse{Status: map[string]string{}}, err
	}
	connection := privateConnection
	if orderCancellation != nil && orderCancellation.AssetType == asset.Spread {
		connection = businessLoginConnection
	}
	if err := e.checkWebsocketAccount(ctx, connection); err != nil {
		return order.CancelAllResponse{Status: map[string]string{}}, err
	}
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}
	if orderCancellation == nil {
		return cancelAllResponse, order.ErrCancelOrderIsNil
	}
	if orderCancellation.AssetType == asset.Spread {
		if err := orderCancellation.Validate(); err != nil {
			return cancelAllResponse, err
		}
		if err := checkSpreadMassCancelFilters(orderCancellation); err != nil {
			return cancelAllResponse, err
		}
		// The spread pair scopes the cancel as CancelAllOrders does
		var spreadID string
		if orderCancellation.Pair.IsPopulated() {
			spreadID = orderCancellation.Pair.Upper().String()
		}
		// A cancel OKX reports unsuccessful is a result, as CancelAllOrders keys it
		err := e.WebsocketAPI().CancelAllSpreadOrders(ctx, spreadID)
		if err != nil && !errors.Is(err, errCancelAllSpreadOrdersFailed) {
			return cancelAllResponse, err
		}
		cancelAllResponse.Status[spreadID] = strconv.FormatBool(err == nil)
		return cancelAllResponse, nil
	}
	switch orderCancellation.Type {
	case order.Trigger, order.OCO, order.ConditionalStop, order.TWAP, order.TrailingStop, order.Chase:
		return cancelAllResponse, fmt.Errorf("%w: cancel of algo order type %v", common.ErrFunctionNotSupported, orderCancellation.Type)
	}
	err = orderCancellation.Validate()
	if err != nil {
		return cancelAllResponse, err
	}
	scope, err := e.newCancelAllScope(orderCancellation)
	if err != nil {
		return cancelAllResponse, err
	}

	cancelAllOrdersRequestParams, errs := e.pendingOrdersToCancel(ctx, scope)
	if len(cancelAllOrdersRequestParams) != 0 {
		// An order whose instrument has no cached code cannot be cancelled over
		// the websocket. It is reported in the error rather than stopping the
		// cancels that can be sent, as a failed batch does below.
		sendable := make([]CancelOrderRequest, 0, len(cancelAllOrdersRequestParams))
		for i := range cancelAllOrdersRequestParams {
			code, codeErr := e.requireWebsocketInstrumentIDCode(cancelAllOrdersRequestParams[i].InstrumentID)
			if codeErr != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w cancelling order %s", codeErr, cancelAllOrdersRequestParams[i].OrderID))
				continue
			}
			cancelAllOrdersRequestParams[i].InstrumentIDCode = code
			sendable = append(sendable, cancelAllOrdersRequestParams[i])
		}
		cancelled, err := cancelInBatches(ctx, sendable, e.WebsocketAPI().CancelMultipleOrders)
		cancelAllResponse.Status = cancelled.Status
		errs = common.AppendError(errs, err)
		if scope.idScoped() {
			return cancelAllResponse, errs
		}
	}
	if len(scope.algoTypes) == 0 {
		return cancelAllResponse, errs
	}
	if ctx.Err() != nil {
		return cancelAllResponse, withContextError(ctx, errs)
	}
	algoParams, err := e.pendingAlgoOrdersToCancel(ctx, scope)
	errs = common.AppendError(errs, err)
	if len(algoParams) != 0 {
		algoIDs := make([]string, len(algoParams))
		for i := range algoParams {
			algoIDs[i] = algoParams[i].AlgoID
		}
		errs = common.AppendError(errs, fmt.Errorf("%w: %s", errWebsocketAlgoCancelUnsupported, strings.Join(algoIDs, ", ")))
	}
	return cancelAllResponse, errs
}

// cancelAllScope is the selection of a cancel-all other than a spread's: the
// pending order listings it reads and the filters each listed order must pass
// to be cancelled.
type cancelAllScope struct {
	instrumentType string
	instrumentID   string
	orderID        string
	clientOrderID  string
	side           order.Side
	tradeMode      string
	// ordinary selects pending ordinary orders, of the ordinaryTypes ordType
	// filter when one is set
	ordinary      bool
	ordinaryTypes string
	// algoTypes are the ordType values the algo order list is requested with,
	// one request each
	algoTypes []string
}

// algoListingTypes maps the algo order types SubmitOrder places to the ordType
// the algo order list takes for each
var algoListingTypes = map[order.Type]string{
	order.ConditionalStop: orderConditional,
	order.OCO:             orderOCO,
	order.Trigger:         orderTrigger,
	order.TrailingStop:    orderMoveOrderStop,
	order.TWAP:            orderTWAP,
	order.Chase:           orderChase,
}

// cancelAllAlgoTypes returns the algo order list requests a cancel-all of
// every order type makes for an asset: one for conditional and oco orders,
// which OKX lists together, and one for each other algo order type OKX lists,
// the icebergs the exported placement methods place among them. OKX lists the
// algo orders of spot, margin, futures and perpetual swap instruments, and
// places chase orders on futures and perpetual swaps only.
func cancelAllAlgoTypes(a asset.Item) []string {
	switch a {
	case asset.Spot, asset.Margin:
		return []string{orderConditional + "," + orderOCO, orderTrigger, orderMoveOrderStop, orderTWAP, orderIceberg, orderSmartIceberg}
	case asset.Futures, asset.PerpetualSwap, asset.Empty, asset.All:
		return []string{orderConditional + "," + orderOCO, orderTrigger, orderMoveOrderStop, orderTWAP, orderIceberg, orderSmartIceberg, orderChase}
	default:
		return nil
	}
}

// newCancelAllScope returns the selection of a cancel-all other than a
// spread's. A filter the listings cannot apply is refused, as dropping it would
// widen the cancel: a pair naming one currency, an asset which is neither
// valid nor a wildcard, a margin type with no OKX trade mode, an order type
// and time in force, or a time in force alone, no pending order reads back
// with, and a time in force beside an algo order type, as algo orders read
// back without one.
func (e *Exchange) newCancelAllScope(c *order.Cancel) (*cancelAllScope, error) {
	s := &cancelAllScope{orderID: c.OrderID, clientOrderID: c.ClientOrderID}
	switch {
	case c.Side == order.UnknownSide || c.Side == order.AnySide:
	case c.Side.IsLong():
		s.side = order.Buy
	case c.Side.IsShort():
		s.side = order.Sell
	default:
		return nil, fmt.Errorf("%w: %s", order.ErrSideIsInvalid, c.Side)
	}
	switch {
	case c.AssetType == asset.Empty || c.AssetType == asset.All:
	case !c.AssetType.IsValid():
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, c.AssetType)
	default:
		if err := e.CurrencyPairs.IsAssetEnabled(c.AssetType); err != nil {
			return nil, err
		}
		s.instrumentType = GetInstrumentTypeFromAssetItem(c.AssetType)
	}
	switch {
	case c.Pair.IsEmpty():
	case !c.Pair.IsPopulated():
		return nil, fmt.Errorf("%w: pair %q names one currency", currency.ErrCurrencyCodeEmpty, c.Pair)
	case s.instrumentType != "":
		// Format through the exchange's pair format so callers passing a
		// differently delimited pair still resolve their instrument; OKX
		// rejects an unmatched instId with error 51001.
		pairFormat, err := e.GetPairFormat(c.AssetType, true)
		if err != nil {
			return nil, err
		}
		s.instrumentID = pairFormat.Format(c.Pair)
	default:
		s.instrumentID = c.Pair.Upper().String()
	}
	if c.MarginType != margin.Unset {
		if s.tradeMode = e.marginTypeToString(c.MarginType); s.tradeMode == "" {
			return nil, fmt.Errorf("%w: %v", margin.ErrMarginTypeUnsupported, c.MarginType)
		}
	}
	var err error
	switch algoType, isAlgo := algoListingTypes[c.Type]; {
	case isAlgo:
		if c.TimeInForce != order.UnknownTIF {
			return nil, fmt.Errorf("%w: %s, algo orders read back without a time in force", order.ErrUnsupportedTimeInForce, c.TimeInForce)
		}
		s.algoTypes = []string{algoType}
	case c.Type != order.UnknownType && c.Type != order.AnyType:
		s.ordinary = true
		if s.ordinaryTypes, err = orderTypeFilter(c.Type, c.TimeInForce); err != nil {
			return nil, err
		}
	case c.TimeInForce != order.UnknownTIF:
		s.ordinary = true
		if s.ordinaryTypes, err = timeInForceFilter(c.TimeInForce); err != nil {
			return nil, err
		}
	default:
		s.ordinary = true
		s.algoTypes = cancelAllAlgoTypes(c.AssetType)
	}
	return s, nil
}

// idScoped reports whether the cancel-all names a single order
func (s *cancelAllScope) idScoped() bool {
	return s.orderID != "" || s.clientOrderID != ""
}

// selects reports whether a listed order passes every filter of the scope, so
// that an order ID cannot cancel an order of another side, and supplying both
// IDs cannot cancel an order matching only one of them. The listing's own
// instrument and order type filters are checked again so that a listing
// ignoring one cancels nothing it did not select.
func (s *cancelAllScope) selects(instrumentID, orderID, clientOrderID string, side order.Side, tradeMode, orderType, listedTypes string) bool {
	return (s.orderID == "" || orderID == s.orderID) &&
		(s.clientOrderID == "" || clientOrderID == s.clientOrderID) &&
		(s.side == order.UnknownSide || side == s.side) &&
		(s.instrumentID == "" || strings.EqualFold(instrumentID, s.instrumentID)) &&
		(s.tradeMode == "" || strings.EqualFold(tradeMode, s.tradeMode)) &&
		(listedTypes == "" || slices.ContainsFunc(strings.Split(listedTypes, ","), func(t string) bool { return strings.EqualFold(t, orderType) }))
}

// errListedOtherInstrumentType reports an order a pending order listing
// returned for another instrument type than it asked for, which a cancel-all
// leaves alone
var errListedOtherInstrumentType = fmt.Errorf("%w: a listed order is of another instrument type than the listing asked for", common.ErrInvalidResponse)

// errListingStalled reports an order listing whose pages stopped advancing
// before its last page, which leaves the orders past them unread
var errListingStalled = errors.New("order listing stopped advancing before its last page")

// otherInstrumentType returns an error naming a listed order of another
// instrument type than the scope's listing asked for, which a listing ignoring
// its instType filter returns, and nil for an order of the scope's instrument
// type or one stating none
func (s *cancelAllScope) otherInstrumentType(orderID, instrumentType string) error {
	if s.instrumentType == "" || instrumentType == "" || strings.EqualFold(instrumentType, s.instrumentType) {
		return nil
	}
	return fmt.Errorf("%w: order %q is %s, not %s", errListedOtherInstrumentType, orderID, instrumentType, s.instrumentType)
}

// pendingOrdersToCancel pages through the pending ordinary orders a cancel-all
// selects: OKX caps the pending order list at 100 records per request, so the
// full list is crawled before cancelling to reach accounts holding more open
// orders than a single page, and the orders the scope selects are kept. A page
// holding no order read before, or whose last order is a cursor requested
// already, leaves the listing incomplete even when it is short, as OKX's
// cursor is exclusive: the orders read are returned beside errListingStalled
// rather than any cursor requested again. An order listed for another
// instrument type than requested is left out and reported.
func (e *Exchange) pendingOrdersToCancel(ctx context.Context, s *cancelAllScope) ([]CancelOrderRequest, error) {
	var myOrders []OrderDetailResponse
	var errs error
	seen := make(common.Seen[string])
	requested := make(common.Seen[string])
	for after := ""; ; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		requested[after] = true
		page, err := e.GetOrderList(ctx, &OrderListRequest{
			InstrumentType: s.instrumentType,
			OrderType:      s.ordinaryTypes,
			InstrumentID:   s.instrumentID,
			After:          after,
		})
		if err != nil {
			return nil, err
		}
		unread := unreadOrders(seen, page, orderDetailID)
		myOrders = append(myOrders, unread...)
		if len(page) == 0 {
			break
		}
		next := page[len(page)-1].OrderID
		if len(unread) == 0 || requested[next] {
			errs = fmt.Errorf("%w: the page after order ID %q leads to order ID %q", errListingStalled, after, next)
			break
		}
		if len(page) < orderListPageSize {
			break
		}
		after = next
	}
	cancelAllOrdersRequestParams := make([]CancelOrderRequest, 0, len(myOrders))
	for x := range myOrders {
		o := &myOrders[x]
		if !s.selects(o.InstrumentID, o.OrderID, o.ClientOrderID, o.Side, o.TradeMode, o.OrderType, s.ordinaryTypes) {
			continue
		}
		if err := s.otherInstrumentType(o.OrderID, o.InstrumentType); err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		cancelAllOrdersRequestParams = append(cancelAllOrdersRequestParams, CancelOrderRequest{
			InstrumentID:  o.InstrumentID,
			OrderID:       o.OrderID,
			ClientOrderID: o.ClientOrderID,
		})
		if s.idScoped() {
			break
		}
	}
	return cancelAllOrdersRequestParams, errs
}

// pendingAlgoOrdersToCancel pages through the untriggered algo orders a
// cancel-all selects, one listing for each of the scope's algo order types. A
// failed listing, or one which stops advancing as pendingOrdersToCancel's
// does, does not stop the others: the orders selected are returned beside the
// joined errors. An order listed for another instrument type than requested is
// left out and reported.
func (e *Exchange) pendingAlgoOrdersToCancel(ctx context.Context, s *cancelAllScope) ([]CancelAlgoOrderRequest, error) {
	var params []CancelAlgoOrderRequest
	var errs error
	for _, algoType := range s.algoTypes {
		seen := make(common.Seen[string])
		requested := make(common.Seen[string])
		for after := ""; ; {
			if ctx.Err() != nil {
				return params, withContextError(ctx, errs)
			}
			requested[after] = true
			page, err := e.GetAlgoOrderList(ctx, &AlgoOrderListRequest{
				OrderType:      algoType,
				InstrumentType: s.instrumentType,
				InstrumentID:   s.instrumentID,
				After:          after,
			})
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("listing %s algo orders: %w", algoType, err))
				break
			}
			unread := unreadOrders(seen, page, algoOrderID)
			for i := range unread {
				o := &unread[i]
				if !s.selects(o.InstrumentID, o.AlgoID, o.AlgoClientOrderID, o.Side, o.TradeMode, o.OrderType, algoType) {
					continue
				}
				if err := s.otherInstrumentType(o.AlgoID, o.InstrumentType); err != nil {
					errs = common.AppendError(errs, err)
					continue
				}
				params = append(params, CancelAlgoOrderRequest{
					InstrumentID:      o.InstrumentID,
					AlgoID:            o.AlgoID,
					AlgoClientOrderID: o.AlgoClientOrderID,
				})
				if s.idScoped() {
					return params, errs
				}
			}
			if len(page) == 0 {
				break
			}
			next := page[len(page)-1].AlgoID
			if len(unread) == 0 || requested[next] {
				errs = common.AppendError(errs, fmt.Errorf("listing %s algo orders: %w: the page after algo ID %q leads to algo ID %q", algoType, errListingStalled, after, next))
				break
			}
			if len(page) < orderListPageSize {
				break
			}
			after = next
		}
	}
	return params, errs
}

// cancelInBatches cancels the orders a cancel-all selects 20 at a time — the
// documented per-request maximum the batch cancel endpoints enforce, so the
// caller of cancel supplies each batch within it — recording each result; a
// failed batch does not stop the batches after it.
func cancelInBatches(ctx context.Context, params []CancelOrderRequest, cancel func(context.Context, []CancelOrderRequest) ([]*OrderActionResponse, error)) (order.CancelAllResponse, error) {
	status := map[string]string{}
	err := cancelBatches(ctx, params, maxOrdersPerBatch, status, cancel, recordOrderCancels)
	return order.CancelAllResponse{Status: status}, err
}

// cancelBatches cancels params size at a time, recording each batch's results
// in status. A failed batch does not stop the batches after it, and a dead
// context stops before the next batch: the errors are joined so the caller
// sees what was and was not cancelled.
func cancelBatches[T, R any](ctx context.Context, params []T, size int, status map[string]string, cancel func(context.Context, []T) (R, error), record func(map[string]string, []T, R) error) error {
	var errs error
	for batch := range slices.Chunk(params, size) {
		if ctx.Err() != nil {
			errs = withContextError(ctx, errs)
			break
		}
		response, err := cancel(ctx, batch)
		errs = common.AppendError(errs, err)
		if cancelResultsUsable(err) {
			errs = common.AppendError(errs, record(status, batch, response))
		}
	}
	return errs
}

// withContextError returns errs joined with the error of a done ctx, unless
// errs already carries it, as the request a dead context failed does
func withContextError(ctx context.Context, errs error) error {
	if ctxErr := ctx.Err(); ctxErr != nil && !errors.Is(errs, ctxErr) {
		return common.AppendError(errs, ctxErr)
	}
	return errs
}

// cancelTarget is an order a batch cancel requests, by exchange order ID or,
// without one, by client order ID: OKX cancels by the exchange order ID when
// both are given
type cancelTarget struct {
	orderID, clientOrderID string
}

// String identifies the order as the cancel requested it
func (t cancelTarget) String() string {
	if t.orderID != "" {
		return "order ID " + strconv.Quote(t.orderID)
	}
	return "client order ID " + strconv.Quote(t.clientOrderID)
}

// cancelOutcome is a batch cancel reply row: the order IDs it names, and its
// status error and message, the error nil when the cancel was accepted
type cancelOutcome struct {
	orderID, clientOrderID string
	err                    error
	message                string
}

// recordOrderCancels records an ordinary order batch cancel's results in status
// by exchange order ID, returning an error for each requested order the reply
// does not account for
func recordOrderCancels(status map[string]string, requested []CancelOrderRequest, rows []*OrderActionResponse) error {
	targets := make([]cancelTarget, len(requested))
	for i := range requested {
		targets[i] = cancelTarget{orderID: requested[i].OrderID, clientOrderID: requested[i].ClientOrderID}
	}
	outcomes := make([]cancelOutcome, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			outcomes = append(outcomes, cancelOutcome{orderID: row.OrderID, clientOrderID: row.ClientOrderID, err: row.Error(), message: row.StatusMessage})
		}
	}
	return recordCancelOutcomes(status, targets, outcomes, false)
}

// recordAlgoCancels records an algo order batch cancel's results in status by
// algo ID, returning an error for each requested order the reply does not
// account for. OKX documents the algo client order ID echo as deprecated, so
// rows echoing none may account for orders requested by it by elimination.
func recordAlgoCancels(status map[string]string, requested []CancelAlgoOrderRequest, rows []AlgoOrderActionResponse) error {
	targets := make([]cancelTarget, len(requested))
	for i := range requested {
		targets[i] = cancelTarget{orderID: requested[i].AlgoID, clientOrderID: requested[i].AlgoClientOrderID}
	}
	outcomes := make([]cancelOutcome, len(rows))
	for i := range rows {
		outcomes[i] = cancelOutcome{orderID: rows[i].AlgoID, clientOrderID: rows[i].AlgoClientOrderID, err: rows[i].Error(), message: rows[i].StatusMessage}
	}
	return recordCancelOutcomes(status, targets, outcomes, true)
}

// cancelTargets are the orders a batch cancel's row may answer: the one order
// it is matched to, or the orders requested by client order ID alone which the
// rows echoing no client order ID cannot be told apart among
type cancelTargets []cancelTarget

// String identifies the orders as the cancels requested them
func (ts cancelTargets) String() string {
	if len(ts) == 1 {
		return ts[0].String()
	}
	names := make([]string, len(ts))
	for i := range ts {
		names[i] = ts[i].String()
	}
	return "one of " + strings.Join(names, ", ")
}

// recordCancelOutcomes records the outcome of each order a batch cancel
// requested in status, keyed by the exchange order ID its row names, and
// returns an error for every requested order the rows do not account for. A
// row accounts for the order whose exchange order ID it names, or whose client
// order ID it echoes when the order was requested by client order ID alone.
// With byElimination, as OKX answers each requested order with one row, the
// rows naming no requested order and echoing no client order ID account for the
// orders requested by client order ID alone when their counts match: each such
// row answers the one client order ID left, or, when several are left, any of
// them, so its result is keyed by its exchange order ID alone and its failure
// is reported naming them all. An order requested more than once reports its
// acceptance over its failures. An accounted order whose row names no exchange
// order ID or states no outcome is reported in the error, as status holds
// exchange order IDs and outcomes only, and so is a row accounting for no
// order. Rows giving one exchange order ID to the orders of two client order
// IDs, by their echoes or by elimination, are reported rather than recorded,
// as one of them is not that order.
func recordCancelOutcomes(status map[string]string, targets []cancelTarget, outcomes []cancelOutcome, byElimination bool) error {
	var errs error
	record := func(requests cancelTargets, o *cancelOutcome) {
		switch {
		case o.orderID == "" && o.err != nil:
			errs = common.AppendError(errs, fmt.Errorf("cancelling %s: %w", requests, o.err))
		case o.orderID == "":
			errs = common.AppendError(errs, fmt.Errorf("%w: the cancel result for %s names no order ID", common.ErrInvalidResponse, requests))
		case errors.Is(o.err, errMissingStatusCode):
			errs = common.AppendError(errs, fmt.Errorf("cancelling %s: %w", requests, o.err))
		case o.err != nil:
			if _, recorded := status[o.orderID]; !recorded {
				status[o.orderID] = cmp.Or(o.message, o.err.Error())
			}
		default:
			// OKX's sCode 0 acknowledges the cancel request only; the order
			// reaches CANCELED once the exchange settles it, which the order
			// streams and listings report.
			status[o.orderID] = order.PendingCancel.String()
		}
	}
	answers := make([]cancelTargets, len(outcomes))
	take := func(matches func(*cancelOutcome) bool) int {
		for i := range outcomes {
			if answers[i] == nil && matches(&outcomes[i]) {
				return i
			}
		}
		return -1
	}
	requestedIDs := make(map[string]bool, len(targets))
	for _, t := range targets {
		if t.orderID != "" {
			requestedIDs[t.orderID] = true
		}
	}
	var byClientID []cancelTarget
	for _, t := range targets {
		var i int
		if t.orderID != "" {
			i = take(func(o *cancelOutcome) bool { return o.orderID == t.orderID })
		} else {
			i = take(func(o *cancelOutcome) bool { return o.clientOrderID == t.clientOrderID })
		}
		switch {
		case i >= 0:
			answers[i] = cancelTargets{t}
		case t.orderID == "":
			byClientID = append(byClientID, t)
		default:
			errs = common.AppendError(errs, fmt.Errorf("%w: no cancel result for %s", common.ErrInvalidResponse, t))
		}
	}
	var unnamed []int
	for i := range outcomes {
		if answers[i] == nil && outcomes[i].clientOrderID == "" && !requestedIDs[outcomes[i].orderID] {
			unnamed = append(unnamed, i)
		}
	}
	eliminated := make(map[string]int, len(byClientID))
	if byElimination && len(byClientID) != 0 && len(unnamed) == len(byClientID) {
		var candidates cancelTargets
		for _, t := range byClientID {
			eliminated[t.clientOrderID]++
			if eliminated[t.clientOrderID] == 1 {
				candidates = append(candidates, t)
			}
		}
		for _, i := range unnamed {
			answers[i] = candidates
		}
		byClientID = nil
	}
	for _, t := range byClientID {
		errs = common.AppendError(errs, fmt.Errorf("%w: no cancel result for %s", common.ErrInvalidResponse, t))
	}
	for i := range outcomes {
		if answers[i] == nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: a cancel result for order ID %q client order ID %q, which was not requested", common.ErrInvalidResponse, outcomes[i].orderID, outcomes[i].clientOrderID))
		}
	}
	duplicates := duplicateCancelOrderIDs(outcomes, answers, eliminated)
	for i, requests := range answers {
		if requests != nil && !duplicates[outcomes[i].orderID] {
			record(requests, &outcomes[i])
		}
	}
	reported := make(map[string]bool, len(duplicates))
	for i := range outcomes {
		orderID := outcomes[i].orderID
		if !duplicates[orderID] || reported[orderID] {
			continue
		}
		reported[orderID] = true
		var requests []string
		for j := range outcomes {
			if outcomes[j].orderID != orderID {
				continue
			}
			for _, t := range answers[j] {
				if name := t.String(); !slices.Contains(requests, name) {
					requests = append(requests, name)
				}
			}
		}
		errs = common.AppendError(errs, fmt.Errorf("%w: order ID %q answers the cancels of %s", errDuplicateReplyOrderID, orderID, strings.Join(requests, ", ")))
	}
	return errs
}

// duplicateCancelOrderIDs returns the exchange order IDs batch cancel rows give
// to the orders of two client order IDs, as a client order ID names one order.
// The rows echoing client order IDs give the order ID they name to each,
// whatever their outcomes, as status keys outcomes by order ID alone. The rows
// answering by elimination give the order ID they name to one client order ID
// left for elimination, once for each of its requests left: the one a row
// echoes for that order ID, or else one which no row echoes for any. An order
// ID they cannot give one is returned, and so is every order ID competing for
// the same client order IDs, as they cannot be told apart. eliminated counts
// the requests of each client order ID left for elimination.
func duplicateCancelOrderIDs(outcomes []cancelOutcome, answers []cancelTargets, eliminated map[string]int) map[string]bool {
	owners := make(map[string]map[string]bool)
	echoed := make(map[string]bool)
	eliminatedRows := make(map[string]int)
	var eliminatedIDs []string
	for i := range outcomes {
		switch o := &outcomes[i]; {
		case o.orderID == "":
		case o.clientOrderID != "":
			if owners[o.orderID] == nil {
				owners[o.orderID] = make(map[string]bool)
			}
			owners[o.orderID][o.clientOrderID] = true
			echoed[o.clientOrderID] = true
		case answers[i] != nil && answers[i][0].orderID == "":
			if eliminatedRows[o.orderID] == 0 {
				eliminatedIDs = append(eliminatedIDs, o.orderID)
			}
			eliminatedRows[o.orderID]++
		}
	}
	duplicates := make(map[string]bool)
	for orderID, clientOrderIDs := range owners {
		if len(clientOrderIDs) > 1 {
			duplicates[orderID] = true
		}
	}
	var unowned []string
	for _, orderID := range eliminatedIDs {
		switch clientOrderIDs := owners[orderID]; len(clientOrderIDs) {
		case 0:
			unowned = append(unowned, orderID)
		case 1:
			for clientOrderID := range clientOrderIDs {
				if eliminatedRows[orderID] > eliminated[clientOrderID] {
					duplicates[orderID] = true
				}
			}
		}
	}
	// Ranked largest first, the order IDs can each take a client order ID of
	// their own only if none needs more requests than the one of its rank has;
	// each order ID needing as many as one which does competes for too few
	var requests []int
	for clientOrderID, n := range eliminated {
		if !echoed[clientOrderID] {
			requests = append(requests, n)
		}
	}
	rows := make([]int, len(unowned))
	for i, orderID := range unowned {
		rows[i] = eliminatedRows[orderID]
	}
	descending := func(a, b int) int { return cmp.Compare(b, a) }
	slices.SortFunc(requests, descending)
	slices.SortFunc(rows, descending)
	var unplaceable int
	for i, n := range rows {
		if i >= len(requests) || n > requests[i] {
			unplaceable = n
		}
	}
	for _, orderID := range unowned {
		if unplaceable > 0 && eliminatedRows[orderID] >= unplaceable {
			duplicates[orderID] = true
		}
	}
	return duplicates
}

// cancelResultsUsable reports whether per-order cancel results can be recorded
// alongside err. A partial success on the websocket or over REST returns fully
// decoded results with its error; any other error, a decode error included,
// can leave them half populated.
func cancelResultsUsable(err error) bool {
	return err == nil || errors.Is(err, errPartialSuccess)
}

// unreadOrders returns the orders of a page that the crawl has not read yet,
// on an earlier page or earlier in the same page, and records their order IDs
// in seen: a page whose cursor did not advance repeats orders already read
func unreadOrders[T any](seen common.Seen[string], page []T, orderID func(*T) string) []T {
	unread := make([]T, 0, len(page))
	for i := range page {
		if !seen.CheckAndRecord(orderID(&page[i])) {
			unread = append(unread, page[i])
		}
	}
	return unread
}

func orderDetailID(o *OrderDetailResponse) string { return o.OrderID }

func spreadOrderDetailID(o *SpreadOrderDetailResponse) string { return o.OrderID }

func algoOrderID(o *AlgoOrderResponse) string { return o.AlgoID }

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if assetType == asset.Spread {
		var resp *SpreadOrderDetailResponse
		resp, err := e.GetSpreadOrderDetails(ctx, &GetSpreadOrderDetailsRequest{OrderID: orderID})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			// A success reply reports the requested order; a missing row
			// leaves the order's state unknown.
			return nil, fmt.Errorf("%w: spread order details reply has no row", common.ErrInvalidResponse)
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
		spreadExec := resp.AccumulatedFillSize.Float64()
		spreadRemaining := spreadRemainingSize(resp.PendingFillSize, spreadAmt, spreadExec, oStatus)
		return &order.Detail{
			Amount:               spreadAmt,
			Exchange:             e.Name,
			OrderID:              resp.OrderID,
			ClientOrderID:        resp.ClientOrderID,
			Side:                 oSide,
			Type:                 oType,
			Pair:                 cp,
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
	orderDetail, err := e.GetOrderDetail(ctx, &OrderDetailRequest{
		InstrumentID: instrumentID,
		OrderID:      orderID,
	})
	if err != nil {
		var respErr *ResponseError
		if !errors.As(err, &respErr) || respErr.Code != orderDoesNotExistCode {
			return nil, err
		}
		// SubmitOrder returns an algo order's algo ID, which the order details
		// endpoint does not know
		algoDetail, algoErr := e.algoOrderInfo(ctx, orderID, instrumentID, pair, assetType)
		if algoErr != nil {
			return nil, common.AppendError(err, algoErr)
		}
		return algoDetail, nil
	}
	if orderDetail == nil || orderDetail.OrderID == "" {
		return nil, fmt.Errorf("%w: order details reply has no order", common.ErrInvalidResponse)
	}
	a, err := replyAsset(orderDetail.InstrumentType, assetType)
	if err != nil {
		return nil, fmt.Errorf("%w for order ID %s", err, orderID)
	}
	status, err := order.StringToOrderStatus(orderDetail.State)
	if err != nil {
		return nil, err
	}
	orderType, tif, err := orderTypeFromString(orderDetail.OrderType)
	if err != nil {
		return nil, err
	}

	quoteSizedOrder := quoteSized(a, orderDetail.OrderType, orderDetail.Side, orderDetail.TargetCurrency)
	amount, remaining, quoteAmount := orderAmounts(orderDetail.Size.Float64(), orderDetail.AccumulatedFillSize.Float64(), orderDetail.AveragePrice.Float64(), quoteSizedOrder, status)
	cost, costAsset := orderCost(a, orderDetail.AveragePrice, orderDetail.AccumulatedFillSize, pair, orderDetail.TradeQuoteCurrency)
	fee, feeAsset := orderFee(orderDetail.TransactionFee, orderDetail.FeeCurrency, orderDetail.RebateAmount, orderDetail.RebateCurrency)
	return &order.Detail{
		Amount:               amount,
		Exchange:             e.Name,
		OrderID:              orderDetail.OrderID,
		ClientOrderID:        orderDetail.ClientOrderID,
		Side:                 orderDetail.Side,
		Type:                 orderType,
		Pair:                 pair,
		Cost:                 cost,
		CostAsset:            costAsset,
		AssetType:            a,
		ReduceOnly:           orderDetail.ReduceOnly,
		Status:               status,
		Price:                orderDetail.Price.Float64(),
		AverageExecutedPrice: orderDetail.AveragePrice.Float64(),
		ExecutedAmount:       orderDetail.AccumulatedFillSize.Float64(),
		RemainingAmount:      remaining,
		QuoteAmount:          quoteAmount,
		Fee:                  fee,
		FeeAsset:             feeAsset,
		Date:                 orderDetail.CreationTime.Time(),
		LastUpdated:          orderDetail.UpdateTime.Time(),
		Leverage:             orderDetail.Leverage.Float64(),
		TimeInForce:          tif,
	}, nil
}

// orderDoesNotExistCode is the error code OKX answers an order lookup with when
// it knows no order by the ID
const orderDoesNotExistCode = 51603

// algoOrderInfo returns the details of an algo order by its algo ID, provided
// it trades the requested instrument
func (e *Exchange) algoOrderInfo(ctx context.Context, algoID, instrumentID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	algo, err := e.GetAlgoOrderDetail(ctx, &AlgoOrderDetailRequest{AlgoID: algoID})
	if err != nil {
		return nil, err
	}
	if algo == nil {
		return nil, fmt.Errorf("%w: algo order details reply has no row", common.ErrInvalidResponse)
	}
	if algo.InstrumentID != instrumentID {
		return nil, fmt.Errorf("%w, unexpected instrument ID %s for algo order ID %s", order.ErrOrderNotFound, algo.InstrumentID, algoID)
	}
	a, err := replyAsset(algo.InstrumentType, assetType)
	if err != nil {
		return nil, fmt.Errorf("%w for algo order ID %s", err, algoID)
	}
	return e.algoOrderDetail(algo, pair, a)
}

// algoOrderDetail returns an algo order's details as SubmitOrder places it
func (e *Exchange) algoOrderDetail(algo *AlgoOrderResponse, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	oType, err := algoOrderType(algo.OrderType)
	if err != nil {
		return nil, err
	}
	status, err := algoOrderStatus(algo.State)
	if err != nil {
		return nil, err
	}
	d := &order.Detail{
		Amount:        algo.Size.Float64(),
		Exchange:      e.Name,
		OrderID:       algo.AlgoID,
		ClientOrderID: algo.AlgoClientOrderID,
		Side:          algo.Side,
		Type:          oType,
		Pair:          pair,
		AssetType:     a,
		Status:        status,
		ReduceOnly:    algo.ReduceOnly == "true",
		Leverage:      algo.Leverage.Float64(),
		Date:          algo.CreationTime.Time(),
		LastUpdated:   algo.UpdateTime.Time(),
	}
	if algoQuoteSized(a, algo) {
		d.Amount, d.QuoteAmount = 0, d.Amount
	}
	// Each algo order type carries its prices where SubmitOrder places them
	switch oType {
	case order.Trigger:
		d.Price, d.TriggerPrice, d.TimeInForce = algo.OrderPrice.Float64(), algo.TriggerPrice.Float64(), triggerTimeInForce(algo.AdvanceOrderType)
	case order.ConditionalStop:
		d.Price, d.TriggerPrice = conditionalLeg(algo)
	case order.TrailingStop:
		d.TriggerPrice = algo.ActivePrice.Float64()
	case order.TWAP:
		d.Price = algo.PriceLimit.Float64()
	}
	return d, nil
}

// algoOrderType returns the order type of an algo order's ordType, for the
// algo order types SubmitOrder places
func algoOrderType(ordType string) (order.Type, error) {
	switch strings.ToLower(ordType) {
	case orderTrigger:
		return order.Trigger, nil
	case orderConditional:
		return order.ConditionalStop, nil
	case orderOCO:
		return order.OCO, nil
	case orderMoveOrderStop:
		return order.TrailingStop, nil
	case orderTWAP:
		return order.TWAP, nil
	case orderChase:
		return order.Chase, nil
	default:
		return order.UnknownType, fmt.Errorf("%w %q", order.ErrTypeIsInvalid, ordType)
	}
}

// algoOrderStatus returns the order status of an algo order's state. An algo
// order takes effect by placing other orders, which carry its executions, so
// an effective order is closed rather than filled, and a partially effective
// one is still open.
func algoOrderStatus(state string) (order.Status, error) {
	switch strings.ToLower(state) {
	case "live", "pause":
		return order.Active, nil
	case "partially_effective":
		return order.Open, nil
	case "effective":
		return order.Closed, nil
	case "canceled":
		return order.Cancelled, nil
	case "order_failed":
		return order.Rejected, nil
	case "partially_failed":
		return order.PartiallyCancelled, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w %q", order.ErrUnsupportedStatusType, state)
	}
}

// orderFee returns an order's fee, a charge positive and a rebate negative, and
// its currency. OKX reports fee negative when charged and nets any rebate into
// it, apart from spot and margin orders other than maker sells, whose maker
// rebate it reports separately as rebate and rebateCcy. The two are combined
// when one is zero or they share a currency; amounts in two currencies, such as
// a maker sell's, cannot be summed, so the fee alone is reported. Subtracting
// from zero keeps a zero fee from reading as -0.
func orderFee(fee types.Number, feeCurrency currency.Code, rebate types.Number, rebateCurrency currency.Code) (float64, currency.Code) {
	charged, rebated := fee.Float64(), rebate.Float64()
	switch {
	case rebated == 0:
		return 0 - charged, feeCurrency
	case charged == 0:
		return 0 - rebated, rebateCurrency
	case feeCurrency.Equal(rebateCurrency):
		return 0 - (charged + rebated), feeCurrency
	default:
		return 0 - charged, feeCurrency
	}
}

// orderCost returns the cost of an order's fills and the currency it is in.
// OKX states accFillSz in the base currency for spot and margin orders, so the
// average price times it is a cost in the currency the order traded against:
// tradeQuoteCcy when OKX names one, as a book such as BTC-USD also trades
// against USDC, and the pair's quote otherwise. Futures, perpetual swap and
// option orders fill in contracts, whose cost needs each contract's value, so
// theirs is left unset rather than reported in contracts.
func orderCost(a asset.Item, averagePrice, accumulatedFillSize types.Number, pair currency.Pair, tradeQuoteCurrency currency.Code) (float64, currency.Code) {
	if a != asset.Spot && a != asset.Margin {
		return 0, currency.EMPTYCODE
	}
	costAsset := pair.Quote
	if !tradeQuoteCurrency.IsEmpty() {
		costAsset = tradeQuoteCurrency
	}
	return averagePrice.Float64() * accumulatedFillSize.Float64(), costAsset
}

// tgtCcy values name the currency a spot market order states its size in.
const (
	targetCurrencyBase  = "base_ccy"
	targetCurrencyQuote = "quote_ccy"
)

// quoteSized reports whether OKX states an ordinary order's sz in the quote
// currency: tgtCcy says which when the order names one, and without one OKX
// sizes spot and margin market buys in the quote currency.
func quoteSized(a asset.Item, ordType string, side order.Side, targetCurrency string) bool {
	if targetCurrency != "" {
		return targetCurrency == targetCurrencyQuote
	}
	if !side.IsLong() {
		return false
	}
	return strings.EqualFold(ordType, orderMarket) && (a == asset.Spot || a == asset.Margin)
}

// algoQuoteSized reports whether OKX reads an algo order's size in the quote
// currency. A stated target currency decides; without one, a spot or margin
// buy that places a market order once triggered is sized in the quote
// currency, as OKX sizes market buys and as SubmitOrder places them.
func algoQuoteSized(a asset.Item, algo *AlgoOrderResponse) bool {
	if algo.TargetCurrency != "" {
		return algo.TargetCurrency == targetCurrencyQuote
	}
	if !algo.Side.IsLong() || (a != asset.Spot && a != asset.Margin) {
		return false
	}
	switch strings.ToLower(algo.OrderType) {
	case orderMoveOrderStop:
		return true
	case orderTrigger:
		return algo.OrderPrice.Float64() == marketOrderPrice
	case orderConditional:
		orderPrice, triggerPrice := conditionalLeg(algo)
		return triggerPrice != 0 && orderPrice == marketOrderPrice
	case orderOCO:
		takeProfit, stopLoss := algo.TakeProfitTriggerPrice.Float64() != 0, algo.StopLossTriggerPrice.Float64() != 0
		takeProfitAtMarket, stopLossAtMarket := algo.TakeProfitOrderPrice.Float64() == marketOrderPrice, algo.StopLossOrderPrice.Float64() == marketOrderPrice
		return (takeProfit || stopLoss) && (!takeProfit || takeProfitAtMarket) && (!stopLoss || stopLossAtMarket)
	}
	return false
}

// conditionalLeg returns the order and trigger prices of the leg a conditional order executes: the leg OKX states
// triggered, or else its stop loss, which SubmitOrder places and which OKX executes alone when both legs are set, or
// the take profit of an order without one
func conditionalLeg(algo *AlgoOrderResponse) (orderPrice, triggerPrice float64) {
	switch strings.ToLower(algo.ActualSide) {
	case "tp":
		return algo.TakeProfitOrderPrice.Float64(), algo.TakeProfitTriggerPrice.Float64()
	case "sl":
		return algo.StopLossOrderPrice.Float64(), algo.StopLossTriggerPrice.Float64()
	}
	orderPrice, triggerPrice = algo.StopLossOrderPrice.Float64(), algo.StopLossTriggerPrice.Float64()
	if orderPrice == 0 && triggerPrice == 0 {
		return algo.TakeProfitOrderPrice.Float64(), algo.TakeProfitTriggerPrice.Float64()
	}
	return orderPrice, triggerPrice
}

// orderAmounts returns an order's size and unfilled size in the base currency,
// and its size in the quote currency when OKX states it that way, while
// accFillSz is always in the base currency. The REST readers and
// wsProcessOrders size orders with it alike.
func orderAmounts(size, executed, averagePrice float64, quoteSizedOrder bool, status order.Status) (amount, remaining, quoteAmount float64) {
	amount = size
	if quoteSizedOrder {
		quoteAmount = size
		switch {
		case status == order.Filled:
			// The accumulated fill avoids the rounding of dividing the quote
			// amount by the average price
			amount = executed
		case averagePrice > 0:
			amount = quoteAmount / averagePrice
		default:
			amount = 0
		}
	}
	// The accumulated fill can exceed the derived amount by a rounding error,
	// and an order can fill completely before its status says so
	if status != order.Filled && amount > executed {
		remaining = amount - executed
	}
	return amount, remaining, quoteAmount
}

// replyAsset returns the asset of the order an order lookup's reply describes:
// the requested asset, or for a reply stating another instrument type, the
// other of spot and margin, as OKX's spot and margin orders share instrument
// IDs. A reply stating no instrument type keeps the requested asset.
func replyAsset(instType string, requested asset.Item) (asset.Item, error) {
	if instType == "" {
		return requested, nil
	}
	a, err := assetTypeFromInstrumentType(instType)
	if err != nil {
		return asset.Empty, err
	}
	switch {
	case a == requested:
	case (a == asset.Spot || a == asset.Margin) && (requested == asset.Spot || requested == asset.Margin):
	default:
		return asset.Empty, fmt.Errorf("%w, unexpected instrument type %s for %s", order.ErrOrderNotFound, instType, requested)
	}
	return a, nil
}

// spreadRemainingSize returns a spread order's size still to fill: OKX's
// pendingFillSz when the order states it, a zero included, as a cancelled or
// rejected part of the size is neither filled nor left to fill; otherwise what
// its fills leave of its size.
func spreadRemainingSize(pendingFillSize OptionalNumber, size, executed float64, status order.Status) float64 {
	if pendingFillSize.Value != nil {
		return pendingFillSize.Value.Float64()
	}
	if status != order.Filled && size > executed {
		return size - executed
	}
	return 0
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

// Withdrawal methods, as a withdrawal's dest names them
const (
	withdrawalDestinationInternal = "3"
	withdrawalDestinationOnChain  = "4"
)

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted: an on-chain withdrawal, or an internal transfer to another OKX
// account when the request says so. The request's client order ID is sent as
// the withdrawal's client ID, and an on-chain withdrawal carries the travel
// rule recipient information OKX takes, as withdrawalRecipient maps it
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	input := WithdrawalRequest{
		Chain:       withdrawRequest.Crypto.Chain,
		Amount:      withdrawRequest.Amount,
		Currency:    withdrawRequest.Currency,
		ToAddress:   withdrawRequest.Crypto.Address,
		Destination: withdrawalDestinationOnChain,
		ClientID:    withdrawRequest.ClientOrderID,
	}
	if withdrawRequest.InternalTransfer {
		input.Destination = withdrawalDestinationInternal
	} else {
		if withdrawRequest.Crypto.AddressTag != "" {
			// OKX takes an on-chain withdrawal's tag in toAddr as address:tag
			input.ToAddress += ":" + withdrawRequest.Crypto.AddressTag
		}
		recipient, err := withdrawalRecipient(&withdrawRequest.Travel)
		if err != nil {
			return nil, err
		}
		input.RecipientInformation = recipient
	}
	resp, err := e.Withdrawal(ctx, &input)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.WithdrawalID == "" {
		return nil, fmt.Errorf("%w: OKX accepted the withdrawal request but returned no withdrawal ID, so check the withdrawal history for client ID %q before retrying", common.ErrInvalidResponse, withdrawRequest.ClientOrderID)
	}
	return &withdraw.ExchangeResponse{
		ID: resp.WithdrawalID,
	}, nil
}

// withdrawalRecipient returns the recipient information OKX takes with an
// on-chain withdrawal for a travel rule, or nil for an empty one: the wallet
// type and the recipient's country, state, town and street. OKX needs nothing
// more for a private wallet, but an exchange wallet's recipient also needs a
// first and last name and an exchange ID, which a travel rule cannot carry,
// so an exchange wallet is refused
func withdrawalRecipient(travel *withdraw.TravelRule) (*WithdrawalRecipientInformation, error) {
	if *travel == (withdraw.TravelRule{}) {
		return nil, nil
	}
	if strings.EqualFold(travel.BeneficiaryWalletType, "exchange") {
		return nil, fmt.Errorf("%w: an exchange wallet recipient needs a first and last name and an exchange ID", common.ErrFunctionNotSupported)
	}
	address := &travel.BeneficiaryAddress
	return &WithdrawalRecipientInformation{
		WalletType:                 travel.BeneficiaryWalletType,
		ReceiverCountry:            address.Country,
		ReceiverCountrySubDivision: address.State,
		ReceiverTownName:           address.City,
		ReceiverStreetName:         strings.Join(slices.DeleteFunc([]string{address.Address1, address.Address2, address.Address3}, func(line string) bool { return line == "" }), ", "),
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

// GetActiveOrders retrieves any orders that are active/open. OKX lists open
// orders whenever they were created, so the request's start and end bound
// their creation times without the order history's three month limit. An algo
// order listing which stops advancing is reported beside the orders of every
// listing read.
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if !e.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}

	var resp []order.Detail
	var format currency.PairFormat
	if req.AssetType == asset.Spread {
		var spreadOrderType string
		if listingNarrowsOrderTypes(req) {
			spreadOrderType, err = spreadOrderTypeFilter(req.Type, req.TimeInForce)
			if err != nil {
				return nil, err
			}
		}
		// OKX caps the pending spread order response at orderListPageSize
		// records and pages the remainder with the endId cursor: endId returns
		// records earlier than the order ID, the direction the newest-first
		// listing pages. beginId returns records newer than the order ID and
		// cannot walk the pages. req.FromOrderID seeds the endId cursor so the
		// crawl returns records earlier than it, matching the standard listing's
		// after cursor and the spread history crawls.
		var spreads []SpreadOrderDetailResponse
		seen := make(common.Seen[string])
		for endID := req.FromOrderID; ; {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var page []SpreadOrderDetailResponse
			page, err = e.GetActiveSpreadOrders(ctx, &GetActiveSpreadOrdersRequest{OrderType: spreadOrderType, EndID: endID})
			if err != nil {
				return nil, err
			}
			spreads = append(spreads, unreadOrders(seen, page, spreadOrderDetailID)...)
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
			if created := spreads[x].CreationTime.Time(); createdBeforeStart(created, req.StartTime) || createdAfterEnd(created, req.EndTime) {
				continue
			}
			spreadAmt := spreads[x].Size.Float64()
			spreadExec := spreads[x].AccumulatedFillSize.Float64()
			spreadRemaining := spreadRemainingSize(spreads[x].PendingFillSize, spreadAmt, spreadExec, oStatus)
			resp = append(resp, order.Detail{
				Amount:               spreadAmt,
				Pair:                 pair,
				Price:                spreads[x].Price.Float64(),
				AverageExecutedPrice: spreads[x].AveragePrice.Float64(),
				ExecutedAmount:       spreadExec,
				RemainingAmount:      spreadRemaining,
				Exchange:             e.Name,
				OrderID:              spreads[x].OrderID,
				ClientOrderID:        spreads[x].ClientOrderID,
				Type:                 oType,
				Side:                 oSide,
				Status:               oStatus,
				AssetType:            req.AssetType,
				Date:                 spreads[x].CreationTime.Time(),
				LastUpdated:          spreads[x].UpdateTime.Time(),
				TimeInForce:          tif,
			})
		}
		asLimitMaker(req, resp)
		return req.Filter(e.Name, resp), nil
	}

	instrumentType := GetInstrumentTypeFromAssetItem(req.AssetType)
	algoOrderTypes, err := algoListingOrderTypes(req.AssetType, req.Type, req.TimeInForce)
	if err != nil {
		return nil, err
	}
	algoSeen := make(common.Seen[string])
	var stalled error
	for _, ordType := range algoOrderTypes {
		algos, err := e.algoOrderDetails(ctx, req, algoSeen, func(after string) ([]AlgoOrderResponse, error) {
			return e.GetAlgoOrderList(ctx, &AlgoOrderListRequest{OrderType: ordType, InstrumentType: instrumentType, After: after})
		})
		if err != nil {
			if !errors.Is(err, errListingStalled) {
				return nil, err
			}
			stalled = common.AppendError(stalled, fmt.Errorf("listing %s algo orders: %w", ordType, err))
		}
		resp = append(resp, algos...)
	}
	if algoOrderTypes != nil && req.Type != order.AnyType {
		// The algo order listings alone hold orders of an algo order type
		return req.Filter(e.Name, resp), stalled
	}
	var orderType string
	if listingNarrowsOrderTypes(req) {
		orderType, err = listingOrderTypeFilter(req.Type, req.TimeInForce)
		if err != nil {
			return nil, err
		}
	}
	// OKX pages the pending order list with the after order ID cursor; the End
	// timestamp is not a documented parameter, so paginating with it re-fetched
	// the first page, losing every order past the first hundred.
	seen := make(common.Seen[string])
allOrders:
	for after := req.FromOrderID; ; {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var page []OrderDetailResponse
		page, err = e.GetOrderList(ctx, &OrderListRequest{
			OrderType:      orderType,
			InstrumentType: instrumentType,
			After:          after,
		})
		if err != nil {
			return nil, err
		}
		orderList := unreadOrders(seen, page, orderDetailID)
		for i := range orderList {
			created := orderList[i].CreationTime.Time()
			if createdBeforeStart(created, req.StartTime) {
				// reached end of orders to crawl
				break allOrders
			}
			if createdAfterEnd(created, req.EndTime) {
				continue
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
			quoteSizedOrder := quoteSized(req.AssetType, orderList[i].OrderType, orderSide, orderList[i].TargetCurrency)
			amount, remaining, quoteAmount := orderAmounts(orderList[i].Size.Float64(), orderList[i].AccumulatedFillSize.Float64(), orderList[i].AveragePrice.Float64(), quoteSizedOrder, orderStatus)
			cost, costAsset := orderCost(req.AssetType, orderList[i].AveragePrice, orderList[i].AccumulatedFillSize, pair, orderList[i].TradeQuoteCurrency)
			fee, feeAsset := orderFee(orderList[i].TransactionFee, orderList[i].FeeCurrency, orderList[i].RebateAmount, orderList[i].RebateCurrency)
			resp = append(resp, order.Detail{
				Amount:               amount,
				Pair:                 pair,
				Price:                orderList[i].Price.Float64(),
				AverageExecutedPrice: orderList[i].AveragePrice.Float64(),
				Cost:                 cost,
				CostAsset:            costAsset,
				ExecutedAmount:       orderList[i].AccumulatedFillSize.Float64(),
				RemainingAmount:      remaining,
				QuoteAmount:          quoteAmount,
				Fee:                  fee,
				FeeAsset:             feeAsset,
				Exchange:             e.Name,
				OrderID:              orderList[i].OrderID,
				ClientOrderID:        orderList[i].ClientOrderID,
				Type:                 oType,
				Side:                 orderSide,
				Status:               orderStatus,
				AssetType:            req.AssetType,
				ReduceOnly:           orderList[i].ReduceOnly,
				Date:                 orderList[i].CreationTime.Time(),
				LastUpdated:          orderList[i].UpdateTime.Time(),
				TimeInForce:          tif,
			})
		}
		if len(page) < orderListPageSize {
			break
		}
		next := page[len(page)-1].OrderID
		if next == after {
			// The page did not advance past the cursor; stop rather than
			// request the same page forever.
			break
		}
		after = next
	}
	asLimitMaker(req, resp)
	return req.Filter(e.Name, resp), stalled
}

// listingNarrowsOrderTypes reports whether a listing request narrows the order
// types it lists: by its order type, or by its time in force whatever the type
func listingNarrowsOrderTypes(req *order.MultiOrderRequest) bool {
	return req.Type != order.UnknownType && (req.Type != order.AnyType || req.TimeInForce != order.UnknownTIF)
}

// createdBeforeStart reports whether an order created at created precedes a
// listing request's start. OKX states creation times to the millisecond, so an
// order created within the start's millisecond is kept.
func createdBeforeStart(created, start time.Time) bool {
	return created.Before(start.Truncate(time.Millisecond))
}

// createdAfterEnd reports whether an order created at created follows a
// listing request's end, when it sets one
func createdAfterEnd(created, end time.Time) bool {
	return !end.IsZero() && created.After(end)
}

// algoListingOrderTypes returns the ordType of each algo order listing a
// listing request reads: one for each algo order type the asset's orders can
// be for AnyType, the requested one for an algo order type, and none for the
// other order types, which the ordinary listings hold. OKX lists one ordType a
// request, apart from conditional and OCO orders together. Algo orders read
// back with no time in force but trigger orders, which read back with the fill
// or kill or immediate or cancel of the order they place, so a request for a
// time in force lists trigger orders at most.
func algoListingOrderTypes(a asset.Item, t order.Type, tif order.TimeInForce) ([]string, error) {
	var listed []string
	switch a {
	case asset.Spot, asset.Margin:
		listed = []string{orderTrigger, orderConditional + "," + orderOCO, orderMoveOrderStop, orderTWAP}
	case asset.Futures, asset.PerpetualSwap:
		listed = []string{orderTrigger, orderConditional + "," + orderOCO, orderMoveOrderStop, orderTWAP, orderChase}
	}
	var ordType string
	switch t {
	case order.AnyType:
		switch {
		case tif == order.UnknownTIF:
			return listed, nil
		case len(listed) > 0 && (tif.Is(order.FillOrKill) || tif.Is(order.ImmediateOrCancel)):
			return []string{orderTrigger}, nil
		}
		return nil, nil
	case order.Trigger:
		ordType = orderTrigger
	case order.ConditionalStop:
		ordType = orderConditional
	case order.OCO:
		ordType = orderOCO
	case order.TrailingStop:
		ordType = orderMoveOrderStop
	case order.TWAP:
		ordType = orderTWAP
	case order.Chase:
		ordType = orderChase
	default:
		return nil, nil
	}
	switch {
	case tif == order.UnknownTIF:
	case t != order.Trigger:
		return nil, fmt.Errorf("%w: %s, %s orders read back with no time in force", order.ErrUnsupportedTimeInForce, tif, t)
	case tif != order.FillOrKill && tif != order.ImmediateOrCancel:
		return nil, fmt.Errorf("%w: %s, trigger orders read back with fill or kill or immediate or cancel alone", order.ErrUnsupportedTimeInForce, tif)
	}
	if !slices.ContainsFunc(listed, func(l string) bool { return slices.Contains(strings.Split(l, ","), ordType) }) {
		return nil, fmt.Errorf("%w: %s orders on %s", order.ErrUnsupportedOrderType, t, a)
	}
	return []string{ordType}, nil
}

// algoOrderDetails pages through an algo order listing with the after algo ID
// cursor, newest first, and returns the details of the requested pairs' orders
// created within the request's bounds and read back with its time in force.
// seen keeps an order the listings of one request repeat once. A page whose
// last order is a cursor requested already, even a short one, as OKX's cursor
// is exclusive, leaves the listing incomplete: the details read are returned
// beside errListingStalled rather than any cursor requested again.
func (e *Exchange) algoOrderDetails(ctx context.Context, req *order.MultiOrderRequest, seen common.Seen[string], fetch func(after string) ([]AlgoOrderResponse, error)) ([]order.Detail, error) {
	var resp []order.Detail
	requested := make(common.Seen[string])
	for after := req.FromOrderID; ; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		requested[after] = true
		page, err := fetch(after)
		if err != nil {
			return nil, err
		}
		algos := unreadOrders(seen, page, algoOrderID)
		for i := range algos {
			created := algos[i].CreationTime.Time()
			if createdBeforeStart(created, req.StartTime) {
				return resp, nil
			}
			if createdAfterEnd(created, req.EndTime) {
				continue
			}
			pair, err := currency.NewPairFromString(algos[i].InstrumentID)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) > 0 && !slices.ContainsFunc(req.Pairs, pair.Equal) {
				continue
			}
			d, err := e.algoOrderDetail(&algos[i], pair, req.AssetType)
			if err != nil {
				return nil, err
			}
			if req.TimeInForce != order.UnknownTIF && !req.TimeInForce.Is(d.TimeInForce) {
				continue
			}
			resp = append(resp, *d)
		}
		if len(page) == 0 {
			return resp, nil
		}
		next := page[len(page)-1].AlgoID
		if requested[next] {
			return resp, fmt.Errorf("%w: the page after algo ID %q leads to algo ID %q", errListingStalled, after, next)
		}
		if len(page) < orderListPageSize {
			return resp, nil
		}
		after = next
	}
}

// asLimitMaker reports post-only Limit orders as LimitMaker when the request
// asks for LimitMaker: OKX places a LimitMaker order as post_only, which reads
// back as a post-only Limit, so the request filter would otherwise drop it.
func asLimitMaker(req *order.MultiOrderRequest, orders []order.Detail) {
	if req.Type != order.LimitMaker {
		return
	}
	for i := range orders {
		if orders[i].Type == order.Limit && orders[i].TimeInForce == order.PostOnly {
			orders[i].Type = order.LimitMaker
		}
	}
}

// GetOrderHistory retrieves account order information Can Limit response to specific order status. An algo order
// listing which stops advancing is reported beside the orders of every listing read.
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.StartTime.Before(timeNow().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	if !e.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}
	var resp []order.Detail
	var stalled error
	if req.AssetType == asset.Spread {
		var err error
		resp, err = e.getSpreadOrderHistoryDetails(ctx, req)
		if err != nil {
			return nil, err
		}
	} else {
		algoOrderTypes, err := algoListingOrderTypes(req.AssetType, req.Type, req.TimeInForce)
		if err != nil {
			return nil, err
		}
		resp, err = e.getAlgoOrderHistoryDetails(ctx, req, algoOrderTypes)
		if err != nil && !errors.Is(err, errListingStalled) {
			return nil, err
		}
		stalled = err
		if algoOrderTypes == nil || req.Type == order.AnyType {
			orders, err := e.getStandardOrderHistoryDetails(ctx, req)
			if err != nil {
				return nil, err
			}
			resp = append(resp, orders...)
		}
	}
	asLimitMaker(req, resp)
	return req.Filter(e.Name, resp), stalled
}

// algoOrderHistoryStates are the states OKX's algo order history lists by, as
// it requires one a request
var algoOrderHistoryStates = []string{"effective", "canceled", "order_failed"}

// getAlgoOrderHistoryDetails retrieves the completed algo orders of each algo
// order listing ordType, in each state the history lists. A listing which
// stops advancing does not stop the others: the orders read are returned
// beside the errListingStalled errors naming it.
func (e *Exchange) getAlgoOrderHistoryDetails(ctx context.Context, req *order.MultiOrderRequest, ordTypes []string) ([]order.Detail, error) {
	instrumentType := GetInstrumentTypeFromAssetItem(req.AssetType)
	var resp []order.Detail
	var stalled error
	seen := make(common.Seen[string])
	for _, ordType := range ordTypes {
		for _, state := range algoOrderHistoryStates {
			algos, err := e.algoOrderDetails(ctx, req, seen, func(after string) ([]AlgoOrderResponse, error) {
				return e.GetAlgoOrderHistory(ctx, &AlgoOrderHistoryRequest{OrderType: ordType, State: state, InstrumentType: instrumentType, After: after})
			})
			if err != nil {
				if !errors.Is(err, errListingStalled) {
					return nil, err
				}
				stalled = common.AppendError(stalled, fmt.Errorf("listing %s algo orders in the %s state: %w", ordType, state, err))
			}
			resp = append(resp, algos...)
		}
	}
	return resp, stalled
}

// getSpreadOrderHistoryDetails retrieves completed spread orders, paging the
// 21 day listing and, when the requested window reaches beyond it, the 3 month
// archive, de-duplicating the overlap between the two listings.
func (e *Exchange) getSpreadOrderHistoryDetails(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	var spreadOrderType string
	if listingNarrowsOrderTypes(req) {
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
	var spreadOrders []SpreadOrderDetailResponse
	seen := make(common.Seen[string])
	record := func(page []SpreadOrderDetailResponse) {
		for i := range page {
			if seen.CheckAndRecord(page[i].OrderID) {
				continue
			}
			spreadOrders = append(spreadOrders, page[i])
		}
	}
	// The 21 day listing cannot hold orders older than its window, so it is
	// crawled only when the requested window reaches into it.
	if req.EndTime.IsZero() || req.EndTime.After(timeNow().Add(-kline.ThreeWeek.Duration())) {
		for endID := req.FromOrderID; ; {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var page []SpreadOrderDetailResponse
			page, err := e.GetCompletedSpreadOrdersLast21Days(ctx, &GetCompletedSpreadOrdersRequest{OrderType: spreadOrderType, EndID: endID, BeginTime: req.StartTime, EndTime: req.EndTime})
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
	if req.StartTime.IsZero() || req.StartTime.Before(timeNow().Add(-kline.ThreeWeek.Duration())) {
		// The 21 day listing cannot reach past its documented window, so the
		// archive covers the remainder of the documented 3 month window.
		for endID := req.FromOrderID; ; {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var page []SpreadOrderDetailResponse
			page, err := e.GetCompletedSpreadOrdersLast3Months(ctx, &GetCompletedSpreadOrdersArchiveRequest{OrderType: spreadOrderType, EndID: endID, BeginTime: req.StartTime, EndTime: req.EndTime})
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
// formatting the pair reproduces the spread ID. No cost is reported: a
// spread's price is the difference between its legs, and OKX sizes inverse
// spreads in USD, so price times size is not what the fills cost.
func (e *Exchange) spreadOrderToDetail(so *SpreadOrderDetailResponse, assetType asset.Item, format currency.PairFormat) (order.Detail, error) {
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
	executed := so.AccumulatedFillSize.Float64()
	remaining := spreadRemainingSize(so.PendingFillSize, amount, executed, oStatus)
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
// listing is crawled beside it.
func (e *Exchange) getStandardOrderHistoryDetails(ctx context.Context, req *order.MultiOrderRequest) ([]order.Detail, error) {
	instrumentType := GetInstrumentTypeFromAssetItem(req.AssetType)
	// The crawls send no ordType and req.Filter narrows by type only, so a
	// requested time in force keeps the OKX order types GetActiveOrders' ordType
	// filter asks for: a post-only Limit query must not return IOC orders, and
	// one no order reads back as, such as a GoodTillCancel query, is refused.
	// This narrows the raw ordType, as the read-back time in force of a limit,
	// rpi and op_fok row cannot tell them apart.
	var ordTypes []string
	if req.TimeInForce != order.UnknownTIF && req.Type != order.UnknownType {
		filter, err := listingOrderTypeFilter(req.Type, req.TimeInForce)
		if err != nil {
			return nil, err
		}
		ordTypes = strings.Split(filter, ",")
	}
	// OKX returns both listings newest first and pages the remainder with the
	// after cursor, which returns the records earlier than the requested
	// order ID. Unlike the end timestamp, the order ID cursor is exclusive,
	// so orders sharing one creation millisecond cannot straddle a page
	// boundary and reappear at the head of the next page. The seen set only
	// guards against OKX repeating a row the cursor already passed, and
	// de-duplicates the rows the two listings share.
	var resp []order.Detail
	seen := make(common.Seen[string])
	crawl := func(fetch func(after string) ([]OrderDetailResponse, error)) error {
	allOrders:
		for after := req.FromOrderID; ; {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			orderList, err := fetch(after)
			if err != nil {
				return err
			}
			if len(orderList) == 0 {
				break
			}
			for i := range orderList {
				if orderList[i].CreationTime.Time().Before(req.StartTime.Truncate(time.Millisecond)) {
					// Reached the end of the crawl: rows arrive newest first,
					// so every row after is older than StartTime. Without
					// begin this is the window's only lower bound, so it
					// matches begin's inclusive millisecond.
					break allOrders
				}
				if seen.CheckAndRecord(orderList[i].OrderID) {
					continue
				}
				pair, err := currency.NewPairFromString(orderList[i].InstrumentID)
				if err != nil {
					return err
				}
				if len(req.Pairs) > 0 && !slices.ContainsFunc(req.Pairs, pair.Equal) {
					continue
				}
				if ordTypes != nil && !slices.Contains(ordTypes, strings.ToLower(orderList[i].OrderType)) {
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
				quoteSizedOrder := quoteSized(req.AssetType, orderList[i].OrderType, orderList[i].Side, orderList[i].TargetCurrency)
				amount, remaining, quoteAmount := orderAmounts(orderList[i].Size.Float64(), orderList[i].AccumulatedFillSize.Float64(), orderList[i].AveragePrice.Float64(), quoteSizedOrder, orderStatus)
				cost, costAsset := orderCost(req.AssetType, orderList[i].AveragePrice, orderList[i].AccumulatedFillSize, pair, orderList[i].TradeQuoteCurrency)
				fee, feeAsset := orderFee(orderList[i].TransactionFee, orderList[i].FeeCurrency, orderList[i].RebateAmount, orderList[i].RebateCurrency)
				resp = append(resp, order.Detail{
					Price:                orderList[i].Price.Float64(),
					AverageExecutedPrice: orderList[i].AveragePrice.Float64(),
					Amount:               amount,
					ExecutedAmount:       orderList[i].AccumulatedFillSize.Float64(),
					RemainingAmount:      remaining,
					QuoteAmount:          quoteAmount,
					Fee:                  fee,
					FeeAsset:             feeAsset,
					ReduceOnly:           orderList[i].ReduceOnly,
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
					Cost:                 cost,
					CostAsset:            costAsset,
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
	// OKX answers begin without end or after with the records closest to
	// begin, the oldest, and the crawl pages towards older ones, so begin is
	// left out and the crawl's own StartTime stop bounds the window instead.
	if err := crawl(func(after string) ([]OrderDetailResponse, error) {
		return e.Get3MonthOrderHistory(ctx, &OrderHistoryRequest{
			InstrumentType: instrumentType,
			After:          after,
			End:            req.EndTime,
		})
	}); err != nil {
		return nil, err
	}
	// The archive does not contain canceled orders without any fills; the
	// 7 day listing does, retaining them for 2 hours, and carries the
	// freshest orders before they reach the archive. It holds every order
	// completed in the last 7 days whenever it was created, so the window's
	// creation time bounds cannot rule it out.
	if err := crawl(func(after string) ([]OrderDetailResponse, error) {
		return e.Get7DayOrderHistory(ctx, &OrderHistoryRequest{
			InstrumentType: instrumentType,
			After:          after,
			End:            req.EndTime,
		})
	}); err != nil {
		return nil, err
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
		candles, err := e.GetSpreadCandlesticksHistory(ctx, &GetSpreadCandlesticksRequest{SpreadID: req.RequestFormatted.String(), Interval: req.ExchangeInterval, Before: start.Add(-time.Nanosecond), After: end, Limit: 100})
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
		candles, err := e.GetCandlesticksHistory(ctx, &CandlesticksRequest{
			InstrumentID: req.RequestFormatted.String(),
			Bar:          req.ExchangeInterval,
			Before:       start.Add(-time.Nanosecond), // Start time not inclusive of candle.
			After:        end,
			Limit:        100,
		})
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
				Volume: candleVolume(candles[x].Volume, candles[x].VolumeCurrency, a),
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
			candles, err := e.GetSpreadCandlesticksHistory(ctx, &GetSpreadCandlesticksRequest{
				SpreadID: req.RequestFormatted.String(),
				Interval: req.ExchangeInterval,
				Before:   req.RangeHolder.Ranges[y].Start.Time.Add(-time.Nanosecond), // Start time not inclusive of candle.
				After:    req.RangeHolder.Ranges[y].End.Time,
				Limit:    100,
			})
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
			candles, err := e.GetCandlesticksHistory(ctx, &CandlesticksRequest{
				InstrumentID: req.RequestFormatted.String(),
				Bar:          req.ExchangeInterval,
				Before:       req.RangeHolder.Ranges[y].Start.Time.Add(-time.Nanosecond), // Start time not inclusive of candle.
				After:        req.RangeHolder.Ranges[y].End.Time,
				Limit:        100,
			})
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
					Volume: candleVolume(candles[x].Volume, candles[x].VolumeCurrency, a),
				})
			}
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	req := &FundingCurrenciesRequest{}
	if !cryptocurrency.IsEmpty() {
		req.Currencies = currency.Currencies{cryptocurrency}
	}
	currencyChains, err := e.GetFundingCurrencies(ctx, req)
	if err != nil {
		return nil, err
	}
	chains := make([]string, 0, len(currencyChains))
	for x := range currencyChains {
		if (!cryptocurrency.IsEmpty() && !strings.EqualFold(cryptocurrency.String(), currencyChains[x].Currency.String())) ||
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
func (e *Exchange) getInstrumentsForOptions(ctx context.Context) ([]InstrumentResponse, error) {
	underlyings, err := e.GetPublicUnderlyings(ctx, instTypeOption)
	if err != nil {
		return nil, err
	}
	var insts []InstrumentResponse
	for x := range underlyings {
		var instruments []InstrumentResponse
		instruments, err = e.GetInstruments(ctx, &InstrumentsRequest{
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

// getInstrumentsForAsset lists the instruments of an asset type and returns
// the asset's stored list, which a listing requested after this one and
// answered first keeps
func (e *Exchange) getInstrumentsForAsset(ctx context.Context, a asset.Item) ([]InstrumentResponse, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	generation := e.nextInstrumentGeneration()
	var instruments []InstrumentResponse
	var instType string
	var err error
	switch a {
	case asset.Options:
		instruments, err = e.getInstrumentsForOptions(ctx)
		if err != nil {
			return nil, err
		}
		return e.cacheInstruments(instTypeOption, instruments, generation), nil
	case asset.Spot:
		instType = instTypeSpot
	case asset.Futures:
		instType = instTypeFutures
	case asset.PerpetualSwap:
		instType = instTypeSwap
	case asset.Margin:
		instType = instTypeMargin
	}

	instruments, err = e.GetInstruments(ctx, &InstrumentsRequest{
		InstrumentType: instType,
	})
	if err != nil {
		return nil, err
	}
	return e.cacheInstruments(instType, instruments, generation), nil
}

// nextInstrumentGeneration counts an instrument listing requested now and
// returns its count, which orders it among the other listings and the
// instruments channel pushes
func (e *Exchange) nextInstrumentGeneration() uint64 {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	e.instrumentGeneration++
	return e.instrumentGeneration
}

// cacheInstruments stores instruments by instrument type, caches their ID
// codes and pre-open states for lookups by instrument ID, and returns the list
// stored for the instrument type. The listing was requested at generation, so
// it is discarded whole when a later listing stored the type's list, and an
// instrument a later push set keeps the newer code and state.
func (e *Exchange) cacheInstruments(instType string, instruments []InstrumentResponse, generation uint64) []InstrumentResponse {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	if e.instrumentListGenerations[instType] > generation {
		return e.instrumentsInfoMap[instType]
	}
	if e.instrumentListGenerations == nil {
		e.instrumentListGenerations = make(map[string]uint64)
	}
	e.instrumentListGenerations[instType] = generation
	e.instrumentsInfoMap[instType] = instruments
	e.cacheInstrumentsLocked(instruments, generation)
	return instruments
}

// cacheInstrumentPush caches the ID codes and pre-open states of an instruments
// channel push, so instruments listed after the startup fetch resolve codes in
// a running process. The instruments channel is not part of
// defaultSubscriptions, so this only runs when it is subscribed explicitly.
// The instruments list itself is left alone because the channel pushes deltas,
// not full snapshots.
func (e *Exchange) cacheInstrumentPush(instruments []InstrumentResponse) {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	e.instrumentGeneration++
	e.cacheInstrumentsLocked(instruments, e.instrumentGeneration)
}

// cacheInstrumentsLocked caches the ID code and pre-open state of each
// instrument of the listing or push counted at generation, except those a
// later listing or push set
func (e *Exchange) cacheInstrumentsLocked(instruments []InstrumentResponse, generation uint64) {
	for x := range instruments {
		instID := instruments[x].InstrumentID.String()
		if e.instrumentGenerations[instID] > generation {
			continue
		}
		if e.instrumentGenerations == nil {
			e.instrumentGenerations = make(map[string]uint64)
		}
		e.instrumentGenerations[instID] = generation
		state := instrumentTradingState{
			preOpen:                     instruments[x].State == instrumentStatePreOpen,
			continuousTradingSwitchTime: instruments[x].ContinuousTradingSwitchTime.Time(),
		}
		if state.continuousTradingSwitchTime.IsZero() {
			switch instruments[x].InstrumentType {
			case instTypeSpot, instTypeMargin, instTypeSwap, instTypeFutures:
				// OKX switches these from pre-open to live at their listing time,
				// and an option only once it has processed the listing
				state.continuousTradingSwitchTime = instruments[x].ListTime.Time()
			}
		}
		if state.preOpen || !state.continuousTradingSwitchTime.IsZero() {
			if e.instrumentTradingStates == nil {
				e.instrumentTradingStates = make(map[string]instrumentTradingState)
			}
			e.instrumentTradingStates[instID] = state
		} else {
			delete(e.instrumentTradingStates, instID)
		}
		// OKX sends a null instIdCode until it generates one, including for a
		// relisted instrument, whose code changes. Dropping the cached code
		// fails its websocket order operations instead of sending a stale code.
		if instruments[x].InstrumentIDCode == 0 {
			delete(e.instrumentIDCodeMap, instID)
			continue
		}
		e.instrumentIDCodeMap[instID] = instruments[x].InstrumentIDCode
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
	rates, err := e.GetFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	if len(rates) != 1 {
		return nil, fmt.Errorf("%w: got %d funding rates for %s", errOnlyOneResponseExpected, len(rates), fPair)
	}
	fr := &rates[0]
	// ts is the observation time; fundingTime is the imminent settlement.
	// nextFundingTime refers to the following period, not this indicative rate.
	pairRate.LatestRate = fundingrate.Rate{Time: fr.Timestamp.Time(), Rate: fr.FundingRate.Decimal()}
	pairRate.TimeOfNextRate = fr.FundingTime.Time()
	if r.IncludePredictedRate {
		pairRate.PredictedUpcomingRate = fundingrate.Rate{Time: fr.FundingTime.Time(), Rate: fr.FundingRate.Decimal()}
	}
	return []fundingrate.LatestRateResponse{pairRate}, nil
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	// OKX publishes funding history for perpetual swaps and X-Perps futures
	if r.Asset != asset.PerpetualSwap && r.Asset != asset.Futures {
		return nil, futures.ErrNotPerpetualFuture
	}
	if r.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w, pair required", currency.ErrCurrencyPairEmpty)
	}
	if r.EndDate.IsZero() {
		// An unset end reads as open-ended, spanning up to now; a before-only
		// query would make OKX answer with the oldest page after the start
		// date, silently truncating the history.
		r.EndDate = timeNow()
	}
	requestLimit := 100
	maxLookback := timeNow().Add(-e.Features.Supports.FuturesCapabilities.MaximumFundingRateHistory)
	if r.StartDate.Before(maxLookback) {
		if r.RespectHistoryLimits {
			r.StartDate = maxLookback
		} else {
			return nil, fmt.Errorf("%w earliest date is %v", fundingrate.ErrFundingRateOutsideLimits, maxLookback)
		}
		if r.EndDate.Before(maxLookback) {
			return nil, futures.ErrGetFundingDataRequired
		}
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
	// The history arrives newest first, so after is the documented cursor for
	// records earlier than the page's oldest funding time, while before holds
	// the requested window floor.
	sd := r.StartDate
	historyCursor := r.EndDate
	seen := make(common.Seen[int64])
	for {
		var history []FundingRateResponse
		history, err = e.GetFundingRateHistory(ctx, &FundingRateHistoryRequest{InstrumentID: fPair.String(), Before: sd, After: historyCursor, Limit: uint64(requestLimit)})
		if err != nil {
			return nil, err
		}
		if len(history) == 0 {
			break
		}
		for i := range history {
			settledRate := &history[i]
			settlementTime := settledRate.FundingTime.Time()
			if settlementTime.Before(r.StartDate) || settlementTime.After(r.EndDate) {
				continue
			}
			if seen.CheckAndRecord(settlementTime.Unix()) {
				continue
			}
			if settledRate.RealisedRate == nil {
				return nil, fmt.Errorf("%w: realised funding rate missing", common.ErrNoResponse)
			}
			pairRate.FundingRates = append(pairRate.FundingRates, fundingrate.Rate{
				Time: settlementTime,
				Rate: settledRate.RealisedRate.Decimal(),
			})
		}
		if len(history) < requestLimit {
			break
		}
		oldest := history[len(history)-1].FundingTime.Time()
		if !sd.Before(oldest) {
			break
		}
		// A full page above the requested start still needs an older cursor.
		// Repeated or newer timestamps cannot establish that history is complete.
		if !oldest.Before(historyCursor) {
			return nil, fmt.Errorf("%w: funding history page did not advance before %s", common.ErrNoResponse, historyCursor)
		}
		historyCursor = oldest
	}
	var rates []FundingRateResponse
	rates, err = e.GetFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	if len(rates) != 1 {
		return nil, fmt.Errorf("%w: got %d funding rates for %s", errOnlyOneResponseExpected, len(rates), fPair)
	}
	fr := &rates[0]
	pairRate.LatestRate = fundingrate.Rate{
		Time: fr.Timestamp.Time(),
		Rate: fr.FundingRate.Decimal(),
	}
	pairRate.TimeOfNextRate = fr.FundingTime.Time()
	if r.IncludePredictedRate {
		pairRate.PredictedUpcomingRate = fundingrate.Rate{
			Time: fr.FundingTime.Time(),
			Rate: fr.FundingRate.Decimal(),
		}
	}
	if r.IncludePayments {
		pairRate.PaymentCurrency = r.PaymentCurrency
		if pairRate.PaymentCurrency.IsEmpty() {
			// Linear and inverse swaps settle in different currencies; the pair
			// name alone does not identify the account bills' payment currency.
			instType := GetInstrumentTypeFromAssetItem(r.Asset)
			instruments, err := e.GetInstruments(ctx, &InstrumentsRequest{InstrumentType: instType, InstrumentID: fPair.String()})
			if err != nil {
				return nil, err
			}
			if len(instruments) != 1 || instruments[0].InstrumentID.String() != fPair.String() || instruments[0].InstrumentType != instType || instruments[0].SettlementCurrency.IsEmpty() {
				return nil, fmt.Errorf("%w: settlement currency unavailable for %s", common.ErrNoResponse, fPair)
			}
			pairRate.PaymentCurrency = instruments[0].SettlementCurrency
		}
		slices.SortFunc(pairRate.FundingRates, func(a, b fundingrate.Rate) int { return b.Time.Compare(a.Time) })
		billDetailsFunc := e.GetBillsDetail3Months
		if timeNow().Sub(r.StartDate) < kline.OneWeek.Duration() {
			billDetailsFunc = e.GetBillsDetailLast7Days
		}
		var after string
		seenBills := make(map[string]struct{})
		seenCursors := make(map[string]struct{})
		for {
			var billDetails []BillsDetailResponse
			billDetails, err = billDetailsFunc(ctx, &BillsDetailRequest{
				InstrumentType: GetInstrumentTypeFromAssetItem(r.Asset),
				InstrumentID:   fPair.String(),
				Currency:       pairRate.PaymentCurrency,
				BillType:       billTypeFundingFee,
				BeginTime:      r.StartDate,
				EndTime:        r.EndDate,
				After:          after,
				Limit:          uint64(requestLimit),
			})
			if err != nil {
				return nil, err
			}
			for i := range billDetails {
				bill := &billDetails[i]
				billTime := bill.Timestamp.Time()
				if billTime.Before(r.StartDate) || billTime.After(r.EndDate) {
					continue
				}
				if bill.BillID == "" {
					return nil, fmt.Errorf("%w: funding bill ID missing", common.ErrNoResponse)
				}
				if _, duplicate := seenBills[bill.BillID]; duplicate {
					continue
				}
				seenBills[bill.BillID] = struct{}{}
				// Bills report the balance update time, which can follow settlement.
				// Match the latest actual settlement at or before that update, so
				// changes in funding frequency do not shift payments into another period.
				index := sort.Search(len(pairRate.FundingRates), func(i int) bool {
					return !pairRate.FundingRates[i].Time.After(billTime)
				})
				if index < len(pairRate.FundingRates) {
					pairRate.FundingRates[index].Payment = pairRate.FundingRates[index].Payment.Add(bill.ProfitAndLoss.Decimal())
				}
			}
			if len(billDetails) < requestLimit {
				break
			}
			// Use the bill ID rather than its timestamp: multiple payments can
			// share the same balance update time.
			after = billDetails[len(billDetails)-1].BillID
			if _, repeated := seenCursors[after]; repeated || after == "" {
				return nil, fmt.Errorf("%w: funding bills pagination did not advance", common.ErrNoResponse)
			}
			seenCursors[after] = struct{}{}
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
	if cfg == nil {
		return 0, fmt.Errorf("error fetching the account level: %w", common.ErrNoResponse)
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

// ChangePositionMargin changes the margin allocated to an isolated margin,
// perpetual swap, futures or options position. An isolated margin position's
// margin is adjusted in the currency the position reports it holds its margin
// in, which may be either currency of its pair
func (e *Exchange) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionChangeRequest", common.ErrNilPointer)
	}
	switch req.Asset {
	case asset.Margin, asset.PerpetualSwap, asset.Futures, asset.Options:
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.Asset)
	}
	if !e.SupportsAsset(req.Asset) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.Asset)
	}
	if req.NewAllocatedMargin <= 0 || math.IsNaN(req.NewAllocatedMargin) || math.IsInf(req.NewAllocatedMargin, 0) {
		return nil, fmt.Errorf("%w %v %v: %v", margin.ErrNewAllocatedMarginRequired, req.Asset, req.Pair, req.NewAllocatedMargin)
	}
	if req.OriginalAllocatedMargin <= 0 || math.IsNaN(req.OriginalAllocatedMargin) || math.IsInf(req.OriginalAllocatedMargin, 0) {
		return nil, fmt.Errorf("%w: %v", margin.ErrOriginalPositionMarginRequired, req.OriginalAllocatedMargin)
	}
	if req.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, req.MarginType)
	}
	pairFormat, err := e.GetPairFormat(req.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := req.Pair.Format(pairFormat)
	// Float subtraction would send OKX its rounding error: 5 - 4.0695 is 0.9305000000000003
	change := decimal.MustFromFloat(req.NewAllocatedMargin).Sub(decimal.MustFromFloat(req.OriginalAllocatedMargin))
	marginType := marginBalanceAdd
	if change.IsNegative() {
		marginType = marginBalanceReduce
		change = change.Neg()
	}
	if req.MarginSide == "" {
		req.MarginSide = "net"
	}
	r := &IncreaseDecreaseMarginRequest{
		InstrumentID:      fPair.String(),
		PositionSide:      req.MarginSide,
		MarginBalanceType: marginType,
		Amount:            types.Number(change.InexactFloat64()),
	}

	if req.Asset == asset.Margin {
		if r.Currency, err = e.isolatedMarginCurrency(ctx, r.InstrumentID, r.PositionSide); err != nil {
			return nil, err
		}
	}

	if _, err := e.IncreaseDecreaseMargin(ctx, r); err != nil {
		return nil, err
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		AllocatedMargin: req.NewAllocatedMargin,
		MarginType:      req.MarginType,
	}, nil
}

// isolatedMarginCurrency returns the currency the account's isolated MARGIN
// position on an instrument and side holds its margin in, which OKX requires
// with an adjustment of that margin. OKX also lists an instrument's closed
// positions, so a position holding no size, margin or debt is left out; one
// holding margin transferred in before it opened is not.
func (e *Exchange) isolatedMarginCurrency(ctx context.Context, instrumentID, positionSide string) (currency.Code, error) {
	positions, err := e.GetPositions(ctx, &PositionsRequest{InstrumentType: instTypeMargin, InstrumentIDs: []string{instrumentID}})
	if err != nil {
		return currency.EMPTYCODE, err
	}
	var marginCurrency currency.Code
	for i := range positions {
		if positions[i].InstrumentID != instrumentID || positions[i].MarginMode != TradeModeIsolated || positions[i].PositionSide != positionSide {
			continue
		}
		if positions[i].QuantityOfPosition.Float64() == 0 && positions[i].Margin.Float64() == 0 && positions[i].Liabilities.Float64() == 0 {
			continue
		}
		if !marginCurrency.IsEmpty() && !marginCurrency.Equal(positions[i].Currency) {
			return currency.EMPTYCODE, fmt.Errorf("%w: isolated %s positions on %s hold margin in %s and %s", errAmbiguousPosition, positionSide, instrumentID, marginCurrency, positions[i].Currency)
		}
		marginCurrency = positions[i].Currency
	}
	if marginCurrency.IsEmpty() {
		return currency.EMPTYCODE, fmt.Errorf("%w: no isolated %s position on %s", futures.ErrPositionNotFound, positionSide, instrumentID)
	}
	return marginCurrency, nil
}

// validContractValue reports whether a contract's value can size its positions: a finite value above zero, which a
// missing or malformed ctVal is not
func validContractValue(value float64) bool {
	return value > 0 && !math.IsInf(value, 1)
}

// GetFuturesPositionSummary returns position summary details for an active
// position. In long/short position mode an instrument can hold a long and a
// short position at once: the request's Direction selects one, and a request
// that would match more than one position is refused
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	if !e.SupportsAsset(req.Asset) || !req.Asset.IsFutures() || req.Asset == asset.Spread {
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
	var multiplier float64
	var contractSettlementType futures.ContractSettlementType
	for i := range contracts {
		if !contracts[i].Name.Equal(fPair) {
			continue
		}
		multiplier = contracts[i].Multiplier
		contractSettlementType = contracts[i].SettlementType
		break
	}
	if !validContractValue(multiplier) {
		return nil, fmt.Errorf("%w %v: no contract value to size its position by", futures.ErrContractNotSupported, fPair)
	}

	positionSummaries, err := e.GetPositions(ctx, &PositionsRequest{InstrumentType: instrumentType, InstrumentIDs: []string{fPair.String()}})
	if err != nil {
		return nil, err
	}
	var positionSummary *AccountPositionResponse
	for i := range positionSummaries {
		if positionSummaries[i].QuantityOfPosition.Float64() == 0 {
			// A zero size marks a closed position; a negative size is a short
			// in net position mode and keeps its sign, matching how the
			// futures summary types carry short sizes.
			continue
		}
		if !positionHasDirection(&positionSummaries[i], req.Direction) {
			continue
		}
		if positionSummary != nil {
			return nil, fmt.Errorf("%w on %s", errAmbiguousPosition, fPair)
		}
		positionSummary = &positionSummaries[i]
	}
	if positionSummary == nil {
		return nil, fmt.Errorf("%w, received '%v', no positions found", errOnlyOneResponseExpected, len(positionSummaries))
	}
	marginMode := margin.Isolated
	if positionSummary.MarginMode == TradeModeCross {
		marginMode = margin.Multi
	}

	acc, err := e.GetAccountBalance(ctx, &AccountBalanceRequest{})
	if err != nil {
		return nil, err
	}
	if len(acc) != 1 {
		return nil, fmt.Errorf("%w, received '%v'", errOnlyOneResponseExpected, len(acc))
	}
	var freeCollateral, totalCollateral, equityOfCurrency, frozenBalance,
		availableEquity, cashBalance, discountEquity,
		equityUSD, isolatedEquity, isolatedLiabilities,
		isolatedUnrealisedProfit, notionalLeverage,
		strategyEquity decimal.Decimal
	totalEquity := acc[0].TotalEquity.Decimal()

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
		equityUSD = acc[0].Details[i].EquityUSD.Decimal()
		isolatedEquity = acc[0].Details[i].IsolatedEquity.Decimal()
		isolatedLiabilities = acc[0].Details[i].IsolatedLiabilities.Decimal()
		isolatedUnrealisedProfit = acc[0].Details[i].IsolatedUnrealisedProfitAndLoss.Decimal()
		notionalLeverage = acc[0].Details[i].NotionalLeverage.Decimal()
		strategyEquity = acc[0].Details[i].StrategyEquity.Decimal()

		break
	}
	collateralMode, err := e.GetCollateralMode(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	positionSize := positionSummary.QuantityOfPosition.Decimal()
	if positionSummary.PositionSide == positionSideShort {
		// In long/short mode OKX reports both sides' sizes positive and
		// carries the direction in posSide; a net mode short is already
		// negative, so only a posSide short flips the sign.
		positionSize = positionSize.Neg()
	}
	return &futures.PositionSummary{
		Pair:            req.Pair,
		Asset:           req.Asset,
		MarginType:      marginMode,
		CollateralMode:  collateralMode,
		Currency:        positionSummary.Currency,
		StartDate:       positionSummary.CreationTime.Time(),
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
		NotionalSize:                 positionSummary.NotionalUSD.Decimal(),
		Leverage:                     positionSummary.Leverage.Decimal(),
		MaintenanceMarginRequirement: positionSummary.MaintenanceMarginRequirement.Decimal(),
		InitialMarginRequirement:     positionSummary.InitialMarginRequirement.Decimal(),
		EstimatedLiquidationPrice:    positionSummary.LiquidationPrice.Decimal(),
		CollateralUsed:               positionSummary.Margin.Decimal(),
		MarkPrice:                    positionSummary.MarkPrice.Decimal(),
		CurrentSize:                  positionSize.Mul(decimal.MustFromFloat(multiplier)),
		ContractSize:                 positionSize,
		ContractMultiplier:           decimal.MustFromFloat(multiplier),
		ContractSettlementType:       contractSettlementType,
		AverageOpenPrice:             positionSummary.AveragePrice.Decimal(),
		UnrealisedPNL:                positionSummary.UnrealisedProfitAndLoss.Decimal(),
		RealisedPNL:                  positionSummary.RealisedProfitAndLoss.Decimal(),
		MaintenanceMarginFraction:    positionSummary.MarginRatio.Decimal(),
		FreeCollateral:               freeCollateral,
		TotalCollateral:              totalCollateral,
		FrozenBalance:                frozenBalance,
		EquityOfCurrency:             equityOfCurrency,
	}, nil
}

// positionHasDirection reports whether a position is on the side a direction
// names, a net mode position's side being the sign of its size; a direction
// that is neither long nor short names every side
func positionHasDirection(p *AccountPositionResponse, direction order.Side) bool {
	switch p.PositionSide {
	case positionSideLong:
		return !direction.IsShort()
	case positionSideShort:
		return !direction.IsLong()
	}
	if p.QuantityOfPosition.Float64() < 0 {
		return !direction.IsLong()
	}
	return !direction.IsShort()
}

// startEndTimeCheckAt checks a window as common.StartEndTimeCheck does, but
// judges a future start by now rather than the wall clock
func startEndTimeCheckAt(start, end, now time.Time) error {
	if err := common.StartEndTimeCheck(start, end); err != nil && !errors.Is(err, common.ErrStartAfterTimeNow) {
		return err
	}
	if start.After(now) {
		return common.ErrStartAfterTimeNow
	}
	return nil
}

// GetFuturesPositionOrders returns the orders that make up futures and
// perpetual swap positions: the orders that executed something. OKX states
// their amounts in contracts, which are scaled by the contract's value, so a
// contract the instrument list does not value, such as an expired one, is
// refused, as GetFuturesPositionSummary refuses it
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if !e.SupportsAsset(req.Asset) || !req.Asset.IsFutures() || req.Asset == asset.Spread {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	now := timeNow()
	if earliest := now.Add(-e.Features.Supports.MaximumOrderHistory); req.StartDate.Before(earliest) {
		if req.RespectOrderHistoryLimits {
			req.StartDate = earliest
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, earliest)
		}
	}
	err := startEndTimeCheckAt(req.StartDate, req.EndDate, now)
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
		if !exist || !validContractValue(contract.Multiplier) {
			return nil, fmt.Errorf("%w %v: no contract value to size its position orders by", futures.ErrContractNotSupported, fPair)
		}
		multiplier, contractSettlementType, settlementCurrency := contract.Multiplier, contract.SettlementType, contract.SettlementCurrency

		resp[i] = futures.PositionResponse{
			Pair:                   req.Pairs[i],
			Asset:                  req.Asset,
			ContractSettlementType: contractSettlementType,
		}

		var positions []OrderDetailResponse
		historyRequest := &OrderHistoryRequest{
			InstrumentType: instrumentType,
			InstrumentID:   fPair.String(),
			Begin:          req.StartDate,
			End:            req.EndDate,
		}
		// Both history listings page at 100 rows, so each is followed with the
		// after order ID cursor until a short page ends the instrument's
		// window. The 7 day listing holds every order completed in the last 7
		// days, whenever it was created, the newest of which the archive lags;
		// only the archive holds orders created and completed before them. A
		// window starting before them reads both, the 7 day listing first,
		// keeping each order once.
		sevenDaysAgo := timeNow().Add(-7 * 24 * time.Hour)
		listings := []func(context.Context, *OrderHistoryRequest) ([]OrderDetailResponse, error){e.Get7DayOrderHistory}
		if req.StartDate.Before(sevenDaysAgo) {
			listings = append(listings, e.Get3MonthOrderHistory)
		}
		seen := make(common.Seen[string])
		for _, getHistory := range listings {
			for after := ""; ; {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				historyRequest.After = after
				var page []OrderDetailResponse
				page, err = getHistory(ctx, historyRequest)
				if err != nil {
					return nil, err
				}
				positions = append(positions, unreadOrders(seen, page, orderDetailID)...)
				if len(page) < orderListPageSize {
					break
				}
				next := page[len(page)-1].OrderID
				if next == after {
					// The page did not advance past the cursor; stop rather
					// than request the same page forever.
					break
				}
				after = next
			}
		}
		for j := range positions {
			if fPair.String() != positions[j].InstrumentID {
				continue
			}
			if positions[j].AccumulatedFillSize.Float64() <= 0 {
				// An order that executed nothing changed no position
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
			if positions[j].TargetCurrency == "quote_ccy" {
				// Size is quote amount.
				orderAmount /= positions[j].AveragePrice
			}

			remainingAmount := float64(0)
			if orderStatus != order.Filled {
				remainingAmount = orderAmount.Float64() - positions[j].AccumulatedFillSize.Float64()
			}
			// OKX sizes fills in contracts: a linear contract's notional is
			// ctVal x contracts x price in the settlement currency, an
			// inverse contract's is ctVal x contracts in USD, so its cost in
			// the base settlement currency divides by the price.
			var cost float64
			switch contractSettlementType {
			case futures.Inverse:
				if avgPx := positions[j].AveragePrice.Float64(); avgPx > 0 {
					cost = multiplier * positions[j].AccumulatedFillSize.Float64() / avgPx
				}
			default:
				cost = positions[j].AveragePrice.Float64() * positions[j].AccumulatedFillSize.Float64() * multiplier
			}
			fee, feeAsset := orderFee(positions[j].TransactionFee, positions[j].FeeCurrency, positions[j].RebateAmount, positions[j].RebateCurrency)
			// The amount, its executed and remaining parts are scaled by the
			// contract value so they share a unit, while ContractAmount keeps
			// the contract count OKX states them in.
			resp[i].Orders = append(resp[i].Orders, order.Detail{
				Price:                positions[j].Price.Float64(),
				AverageExecutedPrice: positions[j].AveragePrice.Float64(),
				Amount:               orderAmount.Float64() * multiplier,
				ContractAmount:       orderAmount.Float64(),
				ExecutedAmount:       positions[j].AccumulatedFillSize.Float64() * multiplier,
				RemainingAmount:      remainingAmount * multiplier,
				Fee:                  fee,
				FeeAsset:             feeAsset,
				Exchange:             e.Name,
				OrderID:              positions[j].OrderID,
				ClientOrderID:        positions[j].ClientOrderID,
				Type:                 oType,
				Side:                 positions[j].Side,
				Status:               orderStatus,
				AssetType:            req.Asset,
				ReduceOnly:           positions[j].ReduceOnly,
				Date:                 positions[j].CreationTime.Time(),
				LastUpdated:          positions[j].UpdateTime.Time(),
				Pair:                 req.Pairs[i],
				Cost:                 cost,
				CostAsset:            settlementCurrency,
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
	_, err = e.SetLeverageRate(ctx, &SetLeverageRequest{
		Leverage:     types.Number(amount),
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
	lev, err := e.GetLeverageRate(ctx, &LeverageInfoRequest{InstrumentIDs: []string{instrumentID}, MarginMode: marginMode})
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
		result, err := e.GetInstruments(ctx, &InstrumentsRequest{
			InstrumentType: instType,
		})
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(result))
		for i := range result {
			instrument := &result[i]
			if instrument.InstrumentType != "" && !strings.EqualFold(instrument.InstrumentType, instType) {
				return nil, fmt.Errorf("%w: instrument type %s", asset.ErrNotSupported, instrument.InstrumentType)
			}
			var underlying currency.Pair
			// Inactive instruments still have settlement terms needed to value open positions.
			// Only active instruments require an underlying; preserve it whenever supplied.
			if instrument.Underlying != "" || instrument.State == "live" {
				underlying, err = currency.NewPairFromString(instrument.Underlying)
				if err != nil {
					return nil, err
				}
			}
			settlementType, err := futures.StringToContractSettlementType(instrument.ContractType)
			if err != nil {
				return nil, err
			}
			if settlementType == futures.UnsetSettlementType {
				settlementType = futures.Linear
				// Derivative baseCcy is often empty; use the underlying only as a fallback.
				if !instrument.SettlementCurrency.IsEmpty() && instrument.SettlementCurrency.Equal(underlying.Base) {
					settlementType = futures.Inverse
				}
			}

			var contractType futures.ContractType
			if item == asset.PerpetualSwap {
				contractType = futures.Perpetual
			} else {
				switch instrument.Alias {
				case "this_week", "next_week":
					contractType = futures.Weekly
				case "quarter", "next_quarter":
					contractType = futures.Quarterly
				}
			}

			resp[i] = futures.Contract{
				Exchange:           e.Name,
				Name:               instrument.InstrumentID,
				Underlying:         underlying,
				Asset:              item,
				StartDate:          instrument.ListTime.Time(),
				EndDate:            instrument.ExpiryTime.Time(),
				IsActive:           instrument.State == "live",
				Status:             instrument.State,
				Type:               contractType,
				SettlementType:     settlementType,
				MarginCurrency:     instrument.SettlementCurrency,
				SettlementCurrency: instrument.SettlementCurrency,
				Multiplier:         instrument.ContractValue.Float64(),
				MaxLeverage:        instrument.MaxLeverage.Float64(),
			}
		}
		return resp, nil
	case asset.Spread:
		results, err := e.GetPublicSpreads(ctx, &GetPublicSpreadsRequest{})
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, len(results))
		for s := range results {
			contractSettlementType, err := futures.StringToContractSettlementType(results[s].SpreadType)
			if err != nil {
				return nil, err
			}
			// quoteCcy is the currency a spread is priced in, while an inverse
			// spread's legs settle in its base currency.
			settlementCurrency := results[s].QuoteCurrency
			if contractSettlementType == futures.Inverse {
				settlementCurrency = results[s].BaseCurrency
			}
			resp[s] = futures.Contract{
				Exchange:           e.Name,
				Name:               results[s].SpreadID,
				Asset:              asset.Spread,
				StartDate:          results[s].ListTime.Time(),
				EndDate:            results[s].ExpiryTime.Time(),
				IsActive:           results[s].State == "live",
				Status:             results[s].State,
				Type:               futures.LongDated,
				SettlementType:     contractSettlementType,
				MarginCurrency:     results[s].QuoteCurrency,
				SettlementCurrency: settlementCurrency,
			}
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// optionInstrumentFamilies lists the option families on OKX's instrument
// tick bands table, which the open interest queries iterate: instFamily is
// the documented filter and an underlying spans several families (BTC-USD
// spans BTC-USD and BTC-USD_UM), so it cannot stand in for one.
func (e *Exchange) optionInstrumentFamilies(ctx context.Context) ([]string, error) {
	tickBands, err := e.GetInstrumentTickBands(ctx, &InstrumentTickBandsRequest{InstrumentType: instTypeOption})
	if err != nil {
		return nil, err
	}
	seen := make(common.Seen[string], len(tickBands))
	families := make([]string, 0, len(tickBands))
	for i := range tickBands {
		family := tickBands[i].InstrumentFamily
		if family == "" {
			continue
		}
		if seen.CheckAndRecord(family) {
			continue
		}
		families = append(families, family)
	}
	if len(families) == 0 {
		return nil, common.ErrNoResponse
	}
	return families, nil
}

// GetOpenInterest returns the open interest of the instruments the keys name,
// whether or not their pairs are enabled, or of every enabled pair when no key
// is given
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		switch k[i].Asset {
		case asset.Futures, asset.PerpetualSwap, asset.Options:
		default:
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) > 1 {
		return e.openInterestForKeys(ctx, k)
	}
	if len(k) == 0 {
		var resp []futures.OpenInterest
		instTypes := map[string]asset.Item{
			instTypeSwap:    asset.PerpetualSwap,
			instTypeFutures: asset.Futures,
			instTypeOption:  asset.Options,
		}
		for instType, v := range instTypes {
			var oid []OpenInterestResponse
			var err error
			switch instType {
			case instTypeOption:
				var families []string
				families, err = e.optionInstrumentFamilies(ctx)
				if err != nil {
					return nil, err
				}
				for u := range families {
					var incOID []OpenInterestResponse
					incOID, err = e.GetOpenInterestData(ctx, &OpenInterestRequest{InstrumentType: instType, InstrumentFamily: families[u]})
					if err != nil {
						return nil, err
					}
					oid = append(oid, incOID...)
				}
			case instTypeSwap, instTypeFutures:
				oid, err = e.GetOpenInterestData(ctx, &OpenInterestRequest{InstrumentType: instType})
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
				resp = append(resp, futures.OpenInterest{
					Key:          key.NewExchangeAssetPair(e.Name, v, p),
					OpenInterest: oid[j].OpenInterest.Float64(),
				})
			}
		}
		return resp, nil
	}
	instTypes := map[asset.Item]string{
		asset.PerpetualSwap: "SWAP",
		asset.Futures:       "FUTURES",
		asset.Options:       instTypeOption,
	}
	pFmt, err := e.FormatSymbol(k[0].Pair(), k[0].Asset)
	if err != nil {
		return nil, err
	}
	var oid []OpenInterestResponse
	switch instTypes[k[0].Asset] {
	case instTypeOption:
		// OKX rejects a family that doesn't match the instrument with 51002, so
		// only the instrument's own family, the prefix of its ID, is queried
		var families []string
		families, err = e.optionInstrumentFamilies(ctx)
		if err != nil {
			return nil, err
		}
		family := optionInstrumentFamily(families, pFmt)
		if family == "" {
			return nil, fmt.Errorf("%w for %s", errInstrumentFamilyRequired, pFmt)
		}
		oid, err = e.GetOpenInterestData(ctx, &OpenInterestRequest{InstrumentType: instTypeOption, InstrumentFamily: family, InstrumentID: pFmt})
		if err != nil {
			return nil, err
		}
	case instTypeSwap, instTypeFutures:
		oid, err = e.GetOpenInterestData(ctx, &OpenInterestRequest{InstrumentType: instTypes[k[0].Asset], InstrumentID: pFmt})
		if err != nil {
			return nil, err
		}
	}
	// The requested instrument is reported whether or not its pair is enabled, as the caller named it
	for i := range oid {
		if !strings.EqualFold(oid[i].InstrumentID, pFmt) {
			continue
		}
		return []futures.OpenInterest{{
			Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, k[0].Pair()),
			OpenInterest: oid[i].OpenInterest.Float64(),
		}}, nil
	}
	return nil, fmt.Errorf("%w: no open interest for %s", common.ErrNoResponse, pFmt)
}

// optionInstrumentFamily returns the one of families that an option's
// instrument ID begins with, or an empty family when none does
func optionInstrumentFamily(families []string, instrumentID string) string {
	for _, f := range families {
		if strings.HasPrefix(instrumentID, f+currency.DashDelimiter) {
			return f
		}
	}
	return ""
}

// openInterestForKeys returns the open interest of each instrument the keys
// name that OKX reports, whether or not its pair is enabled, reading one
// listing for each instrument type and option family the keys need
func (e *Exchange) openInterestForKeys(ctx context.Context, keys []key.PairAsset) ([]futures.OpenInterest, error) {
	type listing struct{ instrumentType, family string }
	instrumentIDs := make([]string, len(keys))
	listings := make([]listing, len(keys))
	var families []string
	for i := range keys {
		instrumentID, err := e.FormatSymbol(keys[i].Pair(), keys[i].Asset)
		if err != nil {
			return nil, err
		}
		l := listing{instrumentType: GetInstrumentTypeFromAssetItem(keys[i].Asset)}
		if keys[i].Asset == asset.Options {
			if families == nil {
				if families, err = e.optionInstrumentFamilies(ctx); err != nil {
					return nil, err
				}
			}
			// OKX rejects a family that doesn't match the instrument with 51002
			if l.family = optionInstrumentFamily(families, instrumentID); l.family == "" {
				return nil, fmt.Errorf("%w for %s", errInstrumentFamilyRequired, instrumentID)
			}
		}
		instrumentIDs[i], listings[i] = strings.ToUpper(instrumentID), l
	}
	openInterests := make(map[listing]map[string]float64)
	seen := make(common.Seen[key.PairAsset], len(keys))
	resp := make([]futures.OpenInterest, 0, len(keys))
	for i := range keys {
		if seen.CheckAndRecord(keys[i]) {
			continue
		}
		openInterest, ok := openInterests[listings[i]]
		if !ok {
			rows, err := e.GetOpenInterestData(ctx, &OpenInterestRequest{InstrumentType: listings[i].instrumentType, InstrumentFamily: listings[i].family})
			if err != nil {
				return nil, err
			}
			openInterest = make(map[string]float64, len(rows))
			for j := range rows {
				openInterest[strings.ToUpper(rows[j].InstrumentID)] = rows[j].OpenInterest.Float64()
			}
			openInterests[listings[i]] = openInterest
		}
		if oi, ok := openInterest[instrumentIDs[i]]; ok {
			resp = append(resp, futures.OpenInterest{
				Key:          key.NewExchangeAssetPair(e.Name, keys[i].Asset, keys[i].Pair()),
				OpenInterest: oi,
			})
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
		insts, err := e.GetInstruments(ctx, &InstrumentsRequest{
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

// instrumentTradingState holds whether OKX last listed or pushed an instrument
// in its pre-open state, and when the instrument switches from its call
// auction, pre-quote or pre-open listing to continuous trading, when OKX gives
// the time
type instrumentTradingState struct {
	preOpen                     bool
	continuousTradingSwitchTime time.Time
}

// preOpenAt returns whether the instrument is pre-open at a time: before its
// continuous trading switch when OKX gives the switch time, and otherwise in
// the state OKX last listed or pushed
func (s instrumentTradingState) preOpenAt(t time.Time) bool {
	if s.continuousTradingSwitchTime.IsZero() || t.IsZero() {
		return s.preOpen
	}
	return t.Before(s.continuousTradingSwitchTime)
}

// instrumentPreOpen returns whether an instrument is in OKX's pre-open state
// at a time, as last listed or pushed
func (e *Exchange) instrumentPreOpen(instID string, at time.Time) bool {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	return e.instrumentTradingStates[instID].preOpenAt(at)
}
