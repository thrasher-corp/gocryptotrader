package binance

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for Binance
func (b *Binance) SetDefaults() {
	b.Name = "Binance"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.SetValues()

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}
	coinFutures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
	}
	usdtFutures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
	}
	europeanOptions := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
	}
	err := b.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.Margin, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.DisableAssetWebsocketSupport(asset.Margin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.CoinMarginedFutures, coinFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.DisableAssetWebsocketSupport(asset.CoinMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.USDTMarginedFutures, usdtFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.DisableAssetWebsocketSupport(asset.USDTMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.Options, europeanOptions)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.DisableAssetWebsocketSupport(asset.Options)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:                true,
			Websocket:           true,
			MaximumOrderHistory: kline.OneDay.Duration() * 7,
			RESTCapabilities: protocol.Features{
				TickerBatching:                 true,
				TickerFetching:                 true,
				KlineFetching:                  true,
				OrderbookFetching:              true,
				AutoPairUpdates:                true,
				AccountInfo:                    true,
				CryptoDeposit:                  true,
				CryptoWithdrawal:               true,
				GetOrder:                       true,
				GetOrders:                      true,
				CancelOrders:                   true,
				CancelOrder:                    true,
				SubmitOrder:                    true,
				DepositHistory:                 true,
				WithdrawalHistory:              true,
				TradeFetching:                  true,
				UserTradeHistory:               true,
				TradeFee:                       true,
				CryptoWithdrawalFee:            true,
				MultiChainDeposits:             true,
				MultiChainWithdrawals:          true,
				HasAssetTypeAccountSegregation: true,
				FundingRateFetching:            true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				TickerFetching:         true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrder:               true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
				FundingRateFetching:    false, // supported but not implemented // TODO when multi-websocket support added
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:      true,
				Leverage:       true,
				CollateralMode: true,
				FundingRates:   true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.FourHour:  true,
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.USDTMarginedFutures: true,
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
					kline.IntervalCapacity{Interval: kline.FiveHundredMilliseconds},
					kline.IntervalCapacity{Interval: kline.ThousandMilliseconds},
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
		Subscriptions: subscription.List{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
		},
	}

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                 apiURL,
		exchange.RestOptions:              eOptionAPIURL,
		exchange.RestUSDTMargined:         ufuturesAPIURL,
		exchange.RestCoinMargined:         cfuturesAPIURL,
		exchange.RestFuturesSupplementary: pMarginAPIURL,
		exchange.EdgeCase1:                "https://api.binance.com",
		exchange.WebsocketSpot:            binanceDefaultWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.Websocket = stream.NewWebsocket()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Binance) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	if err := b.SetupDefaults(exch); err != nil {
		return err
	}
	ePoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            binanceDefaultWebsocketURL,
		RunningURL:            ePoint,
		Connector:             b.WsConnect,
		Subscriber:            b.Subscribe,
		Unsubscriber:          b.Unsubscribe,
		GenerateSubscriptions: b.generateSubscriptions,
		Features:              &b.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
		TradeFeed: b.Features.Enabled.TradeFeed,
	})
	if err != nil {
		return err
	}

	err = b.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            request.NewWeightedRateLimitByDuration(250 * time.Millisecond),
		URL:                  binanceWebsocketAPIURL,
		Authenticated:        true,
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            request.NewWeightedRateLimitByDuration(250 * time.Millisecond),
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Binance) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !b.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	tradingStatus := "TRADING"
	var pairs currency.Pairs
	switch a {
	case asset.Spot, asset.Margin:
		info, err := b.GetExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make([]currency.Pair, 0, len(info.Symbols))
		for x := range info.Symbols {
			if info.Symbols[x].Status != tradingStatus {
				continue
			}
			pair, err := currency.NewPairFromStrings(info.Symbols[x].BaseAsset,
				info.Symbols[x].QuoteAsset)
			if err != nil {
				return nil, err
			}
			if a == asset.Spot && info.Symbols[x].IsSpotTradingAllowed {
				pairs = append(pairs, pair)
			}
			if a == asset.Margin && info.Symbols[x].IsMarginTradingAllowed {
				pairs = append(pairs, pair)
			}
		}
	case asset.CoinMarginedFutures:
		cInfo, err := b.FuturesExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make([]currency.Pair, 0, len(cInfo.Symbols))
		for z := range cInfo.Symbols {
			if cInfo.Symbols[z].ContractStatus != tradingStatus {
				continue
			}
			pair, err := currency.NewPairFromString(cInfo.Symbols[z].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.USDTMarginedFutures:
		uInfo, err := b.UExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make([]currency.Pair, 0, len(uInfo.Symbols))
		for u := range uInfo.Symbols {
			if uInfo.Symbols[u].Status != tradingStatus {
				continue
			}
			var pair currency.Pair
			if uInfo.Symbols[u].ContractType == "PERPETUAL" {
				pair, err = currency.NewPairFromStrings(uInfo.Symbols[u].BaseAsset,
					uInfo.Symbols[u].QuoteAsset)
			} else {
				pair, err = currency.NewPairFromString(uInfo.Symbols[u].Symbol)
			}
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.Options:
		exchangeInformation, err := b.GetOptionsExchangeInformation(ctx)
		if err != nil {
			return nil, err
		}
		pairs = make([]currency.Pair, len(exchangeInformation.OptionSymbols))
		for a := range exchangeInformation.OptionSymbols {
			pairs[a], err = currency.NewPairFromString(exchangeInformation.OptionSymbols[a].Symbol)
			if err != nil {
				return nil, err
			}
		}
	}
	format, err := b.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	return pairs.Format(format), nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Binance) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := b.GetAssetTypes(true)
	for i := range assetTypes {
		pairs, err := b.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return err
		}

		err = b.UpdatePairs(pairs, assetTypes[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return b.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Binance) UpdateTickers(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot, asset.Margin:
		format, err := b.GetPairFormat(a, true)
		if err != nil {
			return err
		}
		pairs, err := b.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		pairs = pairs.Format(format)
		batchNo := len(pairs)/2 + (len(pairs)%2 - 1)
		var tick []PriceChangeStats
		for batch := 0; batch <= batchNo; batch++ {
			var selectedPairs currency.Pairs
			if batch*2+2 >= len(pairs) {
				selectedPairs = pairs[batch*2:]
			} else {
				selectedPairs = pairs[batch*2 : batch*2+2]
			}
			if b.IsAPIStreamConnected() {
				tick, err = b.GetWsTradingDayTickers(&PriceChangeRequestParam{Symbols: selectedPairs, TickerType: "FULL"})
			} else {
				tick, err = b.GetPriceChangeStats(ctx, currency.EMPTYPAIR, selectedPairs)
			}
			if err != nil {
				return err
			}

			for y := range tick {
				pair, err := currency.NewPairFromString(tick[y].Symbol)
				if err != nil {
					return err
				}
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         tick[y].LastPrice.Float64(),
					High:         tick[y].HighPrice.Float64(),
					Low:          tick[y].LowPrice.Float64(),
					Bid:          tick[y].BidPrice.Float64(),
					Ask:          tick[y].AskPrice.Float64(),
					Volume:       tick[y].Volume.Float64(),
					QuoteVolume:  tick[y].QuoteVolume.Float64(),
					Open:         tick[y].OpenPrice.Float64(),
					Close:        tick[y].PrevClosePrice.Float64(),
					Pair:         pair.Format(format),
					ExchangeName: b.Name,
					AssetType:    a,
				})
				if err != nil {
					return err
				}
			}
		}
	case asset.USDTMarginedFutures:
		tick, err := b.U24HTickerPriceChangeStats(ctx, currency.EMPTYPAIR)
		if err != nil {
			return err
		}

		for y := range tick {
			cp, err := currency.NewPairFromString(tick[y].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[y].LastPrice,
				High:         tick[y].HighPrice,
				Low:          tick[y].LowPrice,
				Volume:       tick[y].Volume,
				QuoteVolume:  tick[y].QuoteVolume,
				Open:         tick[y].OpenPrice,
				Close:        tick[y].PrevClosePrice,
				Pair:         cp,
				ExchangeName: b.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.CoinMarginedFutures:
		tick, err := b.GetFuturesSwapTickerChangeStats(ctx, currency.EMPTYPAIR, "")
		if err != nil {
			return err
		}

		for y := range tick {
			cp, err := currency.NewPairFromString(tick[y].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[y].LastPrice.Float64(),
				High:         tick[y].HighPrice.Float64(),
				Low:          tick[y].LowPrice.Float64(),
				Volume:       tick[y].Volume.Float64(),
				QuoteVolume:  tick[y].QuoteVolume.Float64(),
				Open:         tick[y].OpenPrice.Float64(),
				Close:        tick[y].PrevClosePrice.Float64(),
				Pair:         cp,
				ExchangeName: b.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.Options:
		tick, err := b.GetEOptions24hrTickerPriceChangeStatistics(ctx, "")
		if err != nil {
			return err
		}
		for a := range tick {
			cp, err := currency.NewPairFromString(tick[a].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[a].LastPrice.Float64(),
				High:         tick[a].High.Float64(),
				Low:          tick[a].Low.Float64(),
				Volume:       tick[a].Volume.Float64(),
				Open:         tick[a].Open.Float64(),
				Pair:         cp,
				ExchangeName: b.Name,
				AssetType:    asset.Options,
			})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Binance) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if enabled, err := b.IsPairEnabled(p, a); !enabled {
		return nil, err
	}
	var err error
	switch a {
	case asset.Spot, asset.Margin:
		p, err = b.FormatExchangeCurrency(p, a)
		if err != nil {
			return nil, err
		}
		var ticks []PriceChangeStats
		if b.IsAPIStreamConnected() {
			ticks, err = b.GetWsTradingDayTickers(&PriceChangeRequestParam{
				Symbol:     p.String(),
				TickerType: "FULL",
			})
			if err != nil {
				return nil, err
			}
		} else {
			ticks, err = b.GetPriceChangeStats(ctx, p, currency.Pairs{})
			if err != nil {
				return nil, err
			}
		}
		if len(ticks) != 1 {
			return nil, ticker.ErrNoTickerFound
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         ticks[0].LastPrice.Float64(),
			High:         ticks[0].HighPrice.Float64(),
			Low:          ticks[0].LowPrice.Float64(),
			Bid:          ticks[0].BidPrice.Float64(),
			Ask:          ticks[0].AskPrice.Float64(),
			Volume:       ticks[0].Volume.Float64(),
			QuoteVolume:  ticks[0].QuoteVolume.Float64(),
			Open:         ticks[0].OpenPrice.Float64(),
			Close:        ticks[0].PrevClosePrice.Float64(),
			Pair:         p,
			ExchangeName: b.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.USDTMarginedFutures:
		var tick []U24HrPriceChangeStats
		tick, err = b.U24HTickerPriceChangeStats(ctx, p)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick[0].LastPrice,
			High:         tick[0].HighPrice,
			Low:          tick[0].LowPrice,
			Volume:       tick[0].Volume,
			QuoteVolume:  tick[0].QuoteVolume,
			Open:         tick[0].OpenPrice,
			Close:        tick[0].PrevClosePrice,
			Pair:         p,
			ExchangeName: b.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.CoinMarginedFutures:
		var tick []PriceChangeStats
		tick, err = b.GetFuturesSwapTickerChangeStats(ctx, p, "")
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick[0].LastPrice.Float64(),
			High:         tick[0].HighPrice.Float64(),
			Low:          tick[0].LowPrice.Float64(),
			Volume:       tick[0].Volume.Float64(),
			QuoteVolume:  tick[0].QuoteVolume.Float64(),
			Open:         tick[0].OpenPrice.Float64(),
			Close:        tick[0].PrevClosePrice.Float64(),
			Pair:         p,
			ExchangeName: b.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.Options:
		tick, err := b.GetEOptions24hrTickerPriceChangeStatistics(ctx, p.String())
		if err != nil {
			return nil, err
		}
		if len(tick) == 0 {
			return nil, fmt.Errorf("%w, pair: %v", ticker.ErrNoTickerFound, p)
		}
		for a := range tick {
			cp, err := currency.NewPairFromString(tick[a].Symbol)
			if err != nil {
				return nil, err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[a].LastPrice.Float64(),
				High:         tick[a].High.Float64(),
				Low:          tick[a].Low.Float64(),
				Volume:       tick[a].Volume.Float64(),
				Open:         tick[a].Open.Float64(),
				Pair:         cp,
				ExchangeName: b.Name,
				AssetType:    asset.Options,
			})
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	return ticker.GetTicker(b.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (b *Binance) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Binance) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	err := b.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	p = p.Upper()
	isEnabled, err := b.CurrencyPairs.IsPairEnabled(p, assetType)
	if !isEnabled || err != nil {
		return nil, fmt.Errorf("%w pair: %v", currency.ErrPairNotEnabled, p)
	}
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}
	var orderbookNew *OrderBook
	var orderbookPopulated bool
	switch assetType {
	case asset.Spot, asset.Margin:
		if b.IsAPIStreamConnected() {
			orderbookNew, err = b.GetWsOrderbook(
				&OrderBookDataRequestParams{
					Symbol: p,
					Limit:  1000})
		} else {
			orderbookNew, err = b.GetOrderBook(ctx,
				OrderBookDataRequestParams{
					Symbol: p,
					Limit:  1000})
		}

	case asset.USDTMarginedFutures:
		orderbookNew, err = b.UFuturesOrderbook(ctx, p.String(), 1000)
	case asset.CoinMarginedFutures:
		orderbookNew, err = b.GetFuturesOrderbook(ctx, p, 1000)
	case asset.Options:
		var resp *EOptionsOrderbook
		resp, err = b.GetEOptionsOrderbook(ctx, p.String(), 1000)
		if err != nil {
			return nil, err
		}
		book.Bids = make(orderbook.Tranches, len(resp.Bids))
		for x := range resp.Bids {
			book.Bids[x] = orderbook.Tranche{
				Price:  resp.Bids[x][0].Float64(),
				Amount: resp.Bids[x][1].Float64(),
			}
		}
		book.Asks = make(orderbook.Tranches, len(resp.Asks))
		for x := range resp.Asks {
			book.Asks[x] = orderbook.Tranche{
				Price:  resp.Asks[x][0].Float64(),
				Amount: resp.Asks[x][1].Float64(),
			}
		}
		orderbookPopulated = true
	default:
		return nil, fmt.Errorf("[%s] %w", assetType, asset.ErrNotSupported)
	}
	if err != nil {
		return book, err
	}
	if !orderbookPopulated {
		book.Bids = make(orderbook.Tranches, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			book.Bids[x] = orderbook.Tranche{
				Amount: orderbookNew.Bids[x].Quantity,
				Price:  orderbookNew.Bids[x].Price,
			}
		}
		book.Asks = make(orderbook.Tranches, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			book.Asks[x] = orderbook.Tranche{
				Amount: orderbookNew.Asks[x].Quantity,
				Price:  orderbookNew.Asks[x].Price,
			}
		}
	}

	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Binance exchange
func (b *Binance) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	acc.AssetType = assetType
	info.Exchange = b.Name
	switch assetType {
	case asset.Spot:
		var raw *Account
		var err error
		if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			raw, err = b.GetWsAccountInfo(0)
		} else {
			raw, err = b.GetAccount(ctx, false)
		}
		if err != nil {
			return info, err
		}

		var currencyBalance []account.Balance
		for i := range raw.Balances {
			free := raw.Balances[i].Free.InexactFloat64()
			locked := raw.Balances[i].Locked.InexactFloat64()

			currencyBalance = append(currencyBalance, account.Balance{
				Currency: currency.NewCode(raw.Balances[i].Asset),
				Total:    free + locked,
				Hold:     locked,
				Free:     free,
			})
		}

		acc.Currencies = currencyBalance

	case asset.CoinMarginedFutures:
		accData, err := b.GetFuturesAccountInfo(ctx)
		if err != nil {
			return info, err
		}
		var currencyDetails []account.Balance
		for i := range accData.Assets {
			currencyDetails = append(currencyDetails, account.Balance{
				Currency: currency.NewCode(accData.Assets[i].Asset),
				Total:    accData.Assets[i].WalletBalance,
				Hold:     accData.Assets[i].WalletBalance - accData.Assets[i].AvailableBalance,
				Free:     accData.Assets[i].AvailableBalance,
			})
		}

		acc.Currencies = currencyDetails

	case asset.USDTMarginedFutures:
		accData, err := b.UAccountBalanceV2(ctx)
		if err != nil {
			return info, err
		}
		accountCurrencyDetails := make(map[string][]account.Balance)
		for i := range accData {
			currencyDetails := accountCurrencyDetails[accData[i].AccountAlias]
			accountCurrencyDetails[accData[i].AccountAlias] = append(
				currencyDetails, account.Balance{
					Currency: currency.NewCode(accData[i].Asset),
					Total:    accData[i].Balance,
					Hold:     accData[i].Balance - accData[i].AvailableBalance,
					Free:     accData[i].AvailableBalance,
				},
			)
		}

		if info.Accounts, err = account.CollectBalances(accountCurrencyDetails, assetType); err != nil {
			return account.Holdings{}, err
		}
	case asset.Margin:
		accData, err := b.GetMarginAccount(ctx)
		if err != nil {
			return info, err
		}
		currencyDetails := make([]account.Balance, len(accData.UserAssets))
		for i := range accData.UserAssets {
			currencyDetails[i] = account.Balance{
				Currency:               currency.NewCode(accData.UserAssets[i].Asset),
				Total:                  accData.UserAssets[i].Free + accData.UserAssets[i].Locked,
				Hold:                   accData.UserAssets[i].Locked,
				Free:                   accData.UserAssets[i].Free,
				AvailableWithoutBorrow: accData.UserAssets[i].Free - accData.UserAssets[i].Borrowed,
				Borrowed:               accData.UserAssets[i].Borrowed,
			}
		}
		acc.Currencies = currencyDetails
	case asset.Options:
		accData, err := b.GetOptionsAccountInformation(ctx)
		if err != nil {
			return info, err
		}
		currencyDetails := make([]account.Balance, len(accData.Asset))
		for i := range accData.Asset {
			currencyDetails[i] = account.Balance{
				Currency: currency.NewCode(accData.Asset[i].AssetType),
				Total:    accData.Asset[i].MarginBalance.Float64(),
				Hold:     accData.Asset[i].Locked.Float64(),
				Free:     accData.Asset[i].AvailableFunds.Float64(),
			}
		}
		acc.Currencies = currencyDetails
	default:
		return info, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Binance) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(b.Name, creds, assetType)
	if err != nil {
		return b.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (b *Binance) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Binance) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := b.WithdrawHistory(ctx, c, "", time.Time{}, time.Time{}, 0, 10000)
	if err != nil {
		return nil, err
	}

	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		resp[i] = exchange.WithdrawalHistory{
			Status:          strconv.FormatInt(withdrawals[i].Status, 10),
			TransferID:      withdrawals[i].ID,
			Currency:        withdrawals[i].Coin,
			Amount:          withdrawals[i].Amount,
			Fee:             withdrawals[i].TransactionFee,
			CryptoToAddress: withdrawals[i].Address,
			CryptoTxID:      withdrawals[i].TransactionID,
			CryptoChain:     withdrawals[i].Network,
			Timestamp:       withdrawals[i].ApplyTime.Time(),
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Binance) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	const limit = 1000
	rFmt, err := b.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	pFmt := p.Format(rFmt)
	resp := make([]trade.Data, 0, limit)
	switch a {
	case asset.Spot, asset.Margin:
		var tradeData []RecentTrade
		if b.IsAPIStreamConnected() {
			tradeData, err = b.GetWsMostRecentTrades(&RecentTradeRequestParams{Symbol: pFmt, Limit: limit})
		} else {
			tradeData, err = b.GetMostRecentTrades(ctx, &RecentTradeRequestParams{Symbol: pFmt, Limit: limit})
		}
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     b.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Quantity,
				Timestamp:    tradeData[i].Time.Time(),
			})
		}
	case asset.USDTMarginedFutures:
		tradeData, err := b.URecentTrades(ctx, pFmt.String(), "", limit)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     b.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Qty,
				Timestamp:    tradeData[i].Time.Time(),
			})
		}
	case asset.CoinMarginedFutures:
		tradeData, err := b.GetFuturesPublicTrades(ctx, pFmt, limit)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     b.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Qty,
				Timestamp:    tradeData[i].Time.Time(),
			})
		}
	case asset.Options:
		tradeData, err := b.GetEOptionsRecentTrades(ctx, p.String(), limit)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     b.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Quantity.Float64(),
				Timestamp:    tradeData[i].Time.Time(),
			})
		}
	}

	if b.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(b.Name, resp...)
		if err != nil {
			return nil, err
		}
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Binance) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, from, to time.Time) ([]trade.Data, error) {
	if err := b.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	rFmt, err := b.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	pFmt := p.Format(rFmt)
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var trades []AggregatedTrade
		if b.IsAPIStreamConnected() {
			trades, err = b.GetWsAggregatedTrades(&WsAggregateTradeRequestParams{
				Symbol:    pFmt.String(),
				StartTime: from.UnixMilli(),
				EndTime:   to.UnixMilli(),
			})
		} else {
			trades, err = b.GetAggregatedTrades(ctx, &AggregatedTradeRequestParams{
				Symbol:    pFmt.String(),
				StartTime: from,
				EndTime:   to,
			})
		}
		if err != nil {
			return nil, fmt.Errorf("%w %v", err, pFmt)
		}
		result := make([]trade.Data, len(trades))
		for i := range trades {
			result[i] = trade.Data{
				CurrencyPair: p,
				TID:          strconv.FormatInt(trades[i].ATradeID, 10),
				Amount:       trades[i].Quantity,
				Exchange:     b.Name,
				Price:        trades[i].Price,
				Timestamp:    trades[i].TimeStamp.Time(),
				AssetType:    a,
				Side:         order.AnySide,
			}
		}
		return result, nil
	case asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		var trades []UPublicTradesData
		if a == asset.USDTMarginedFutures {
			trades, err = b.UFuturesHistoricalTrades(ctx, pFmt.String(), "", 0)
		} else {
			trades, err = b.GetFuturesHistoricalTrades(ctx, pFmt, "", 0)
		}
		if err != nil {
			return nil, err
		}
		result := make([]trade.Data, len(trades))
		for i := range trades {
			result[i] = trade.Data{
				CurrencyPair: p,
				TID:          strconv.FormatInt(trades[i].ID, 10),
				Amount:       trades[i].Qty,
				Exchange:     b.Name,
				Price:        trades[i].Price,
				Timestamp:    trades[i].Time.Time(),
				AssetType:    a,
				Side:         order.AnySide,
			}
		}
		return result, nil
	case asset.Options:
		trades, err := b.GetEOptionsTradeHistory(ctx, pFmt.String(), 0, 0)
		if err != nil {
			return nil, err
		}
		result := make([]trade.Data, len(trades))
		for i := range trades {
			result[i] = trade.Data{
				CurrencyPair: p,
				TID:          strconv.FormatInt(trades[i].ID, 10),
				Amount:       trades[i].Quantity.Float64(),
				Exchange:     b.Name,
				Price:        trades[i].Price.Float64(),
				Timestamp:    trades[i].Time.Time(),
				AssetType:    a,
				Side:         order.AnySide,
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

func (b *Binance) orderTypeToString(orderType order.Type) (string, error) {
	switch orderType {
	case order.Limit:
		return cfuturesLimit, nil
	case order.Market:
		return cfuturesMarket, nil
	case order.Stop:
		return cfuturesStop, nil
	case order.StopMarket:
		return cfuturesStopMarket, nil
	case order.TakeProfit:
		return cfuturesTakeProfit, nil
	case order.TakeProfitMarket:
		return cfuturesTakeProfitMarket, nil
	case order.TrailingStop:
		return cfuturesTrailingStopMarket, nil
	case order.OCO, order.SOR:
		return orderType.String(), nil
	}
	return "", fmt.Errorf("%w, order type %v", order.ErrTypeIsInvalid, orderType)
}

// SubmitOrder submits a new order
func (b *Binance) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(b.GetTradingRequirements())
	if err != nil {
		return nil, err
	}
	var orderID string
	status := order.New
	var trades []order.TradeHistory
	if s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w received '%v'", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	switch s.AssetType {
	case asset.Spot, asset.Margin:
		var sideType string
		if s.Side.IsLong() {
			sideType = order.Buy.String()
		} else {
			sideType = order.Sell.String()
		}
		timeInForce := BinanceRequestParamsTimeGTC
		var requestParamsOrderType RequestParamsOrderType
		switch s.Type {
		case order.Market:
			timeInForce = ""
			requestParamsOrderType = BinanceRequestParamsOrderMarket
		case order.Limit:
			if s.ImmediateOrCancel {
				timeInForce = BinanceRequestParamsTimeIOC
			}
			requestParamsOrderType = BinanceRequestParamsOrderLimit
		default:
			return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, s.Type)
		}
		switch {
		case s.Type == order.SOR:
			if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				var resp []OSROrder
				resp, err = b.WsPlaceNewSOROrder(&WsOSRPlaceOrderParams{
					Symbol:           s.Pair.String(),
					Side:             sideType,
					OrderType:        string(requestParamsOrderType),
					TimeInForce:      string(timeInForce),
					Price:            s.Price,
					Quantity:         s.Amount,
					NewClientOrderID: s.ClientOrderID,
				})
				if err != nil {
					return nil, err
				}
				if len(resp) != 0 {
					return nil, order.ErrUnableToPlaceOrder
				}
				orderID = strconv.FormatInt(resp[0].OrderID, 10)
			} else {
				var resp *SOROrderResponse
				resp, err = b.NewOrderUsingSOR(ctx, &SOROrderRequestParams{
					Symbol:           s.Pair,
					Side:             sideType,
					OrderType:        string(requestParamsOrderType),
					TimeInForce:      string(timeInForce),
					Quantity:         s.Amount,
					Price:            s.Price,
					NewClientOrderID: s.ClientOrderID,
				})
				if err != nil {
					return nil, err
				}
				orderID = strconv.FormatInt(resp.OrderID, 10)
			}
		case s.Type == order.OCO:
			var ocoOrder *OCOOrder
			if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				ocoOrder, err = b.WsPlaceOCOOrder(&PlaceOCOOrderParam{
					Symbol:               s.Pair.String(),
					Side:                 sideType,
					Price:                s.Price,
					Quantity:             s.Amount,
					ListClientOrderID:    "list-" + s.ClientOrderID,
					LimitClientOrderID:   "limit-" + s.ClientOrderID,
					StopPrice:            s.TriggerPrice,
					StopLimitTimeInForce: string(timeInForce),
				})
				if err != nil {
					return nil, err
				}
				orderID = strconv.FormatInt(ocoOrder.OrderListID, 10)
			} else {
				ocoOrder, err = b.NewOCOOrder(
					ctx,
					&OCOOrderParam{
						Symbol:               s.Pair,
						Side:                 sideType,
						Amount:               s.Amount,
						Price:                s.Price,
						StopPrice:            s.TriggerPrice,
						LimitClientOrderID:   "limit-" + s.ClientOrderID,
						StopClientOrderID:    "stop-" + s.ClientOrderID,
						StopLimitPrice:       s.RiskManagementModes.StopLoss.LimitPrice,
						StopLimitTimeInForce: string(timeInForce),
					})
				if err != nil {
					return nil, err
				}
				orderID = strconv.FormatInt(ocoOrder.OrderListID, 10)
			}
		case b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper():
			var results *TradeOrderResponse
			results, err = b.WsPlaceNewOrder(&TradeOrderRequestParam{
				Symbol:      s.Pair.String(),
				Side:        sideType,
				OrderType:   string(requestParamsOrderType),
				TimeInForce: string(timeInForce),
				Price:       s.Price,
				Quantity:    s.Amount,
			})
			if err != nil {
				return nil, err
			}
			orderID = strconv.FormatInt(results.OrderID, 10)
		default:
			var response NewOrderResponse
			response, err = b.NewOrder(ctx, &NewOrderRequest{
				Symbol:           s.Pair,
				Side:             sideType,
				Price:            s.Price,
				Quantity:         s.Amount,
				TradeType:        requestParamsOrderType,
				TimeInForce:      timeInForce,
				NewClientOrderID: s.ClientOrderID,
			})
			if err != nil {
				return nil, err
			}

			orderID = strconv.FormatInt(response.OrderID, 10)
			if response.ExecutedQty == response.OrigQty {
				status = order.Filled
			}

			trades = make([]order.TradeHistory, len(response.Fills))
			for i := range response.Fills {
				trades[i] = order.TradeHistory{
					Price:    response.Fills[i].Price,
					Amount:   response.Fills[i].Quantity,
					Fee:      response.Fills[i].Commission,
					FeeAsset: response.Fills[i].CommissionAsset,
				}
			}
		}
	case asset.CoinMarginedFutures,
		asset.USDTMarginedFutures:
		var reqSide string
		switch s.Side {
		case order.Buy, order.Sell:
			reqSide = s.Side.String()
		default:
			return nil, order.ErrSideIsInvalid
		}

		var (
			oType       string
			timeInForce RequestParamsTimeForceType
		)
		oType, err = b.orderTypeToString(s.Type)
		if err != nil {
			return nil, err
		}
		if s.Type == order.Limit {
			timeInForce = BinanceRequestParamsTimeGTC
		}
		if s.AssetType == asset.CoinMarginedFutures {
			var o *FuturesOrderPlaceData
			o, err = b.FuturesNewOrder(ctx, &FuturesNewOrderRequest{
				Symbol:           s.Pair,
				Side:             reqSide,
				OrderType:        oType,
				TimeInForce:      timeInForce,
				NewClientOrderID: s.ClientOrderID,
				Quantity:         s.Amount,
				Price:            s.Price,
				ReduceOnly:       s.ReduceOnly,
			})
			if err != nil {
				return nil, err
			}
			orderID = strconv.FormatInt(o.OrderID, 10)
		} else {
			var o *UOrderData
			o, err = b.UFuturesNewOrder(ctx, &UFuturesNewOrderRequest{
				Symbol:           s.Pair,
				Side:             reqSide,
				OrderType:        oType,
				TimeInForce:      string(timeInForce),
				NewClientOrderID: s.ClientOrderID,
				Quantity:         s.Amount,
				Price:            s.Price,
				ReduceOnly:       s.ReduceOnly,
			})
			if err != nil {
				return nil, err
			}
			orderID = strconv.FormatInt(o.OrderID, 10)
		}
	case asset.Options:
		var result *OptionOrder
		result, err = b.NewOptionsOrder(ctx, &OptionsOrderParams{
			Symbol:               s.Pair,
			Side:                 s.Side.String(),
			OrderType:            strings.ToUpper(s.Type.String()),
			Amount:               s.Amount,
			Price:                s.Price,
			ReduceOnly:           s.ReduceOnly,
			PostOnly:             s.PostOnly,
			NewOrderResponseType: "RESULT",
			ClientOrderID:        s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		orderID = strconv.FormatInt(result.OrderID, 10)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, s.AssetType)
	}

	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Trades = trades
	resp.Status = status
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Binance) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Binance) CancelOrder(ctx context.Context, o *order.Cancel) error {
	err := o.Validate(o.StandardCancel())
	if err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot, asset.Margin:
		var orderIDInt int64
		switch {
		case o.Type == order.OCO:
			if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				_, err = b.WsCancelOCOOrder(o.Pair, o.OrderID, o.ClientOrderID, "")
			} else {
				_, err = b.CancelOCOOrder(ctx, o.Pair.String(), o.OrderID, o.ClientOrderID, "")
			}
		case b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper():
			orderIDInt, err = strconv.ParseInt(o.OrderID, 10, 64)
			if err != nil {
				return err
			}
			_, err = b.WsCancelOrder(&QueryOrderParam{
				Symbol:            o.Pair.String(),
				OrderID:           orderIDInt,
				OrigClientOrderID: o.ClientOrderID,
			})
		default:
			orderIDInt, err = strconv.ParseInt(o.OrderID, 10, 64)
			if err != nil {
				return err
			}
			_, err = b.CancelExistingOrder(ctx,
				o.Pair,
				orderIDInt,
				o.AccountID)
		}
		if err != nil {
			return err
		}
	case asset.CoinMarginedFutures:
		_, err := b.FuturesCancelOrder(ctx, o.Pair, o.OrderID, "")
		if err != nil {
			return err
		}
	case asset.USDTMarginedFutures:
		_, err := b.UCancelOrder(ctx, o.Pair.String(), o.OrderID, "")
		if err != nil {
			return err
		}
	case asset.Options:
		reg := regexp.MustCompile(`^\d+$`)
		if !reg.MatchString(o.OrderID) {
			return fmt.Errorf("%w, invalid orderID", order.ErrOrderIDNotSet)
		}
		orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
		if err != nil {
			return err
		}
		_, err = b.CancelOptionsOrder(ctx, o.Pair.String(), o.ClientOrderID, orderIDInt)
		if err != nil {
			return err
		}
	}
	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Binance) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Binance) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	var err error
	err = req.Validate()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch req.AssetType {
	case asset.Spot, asset.Margin:
		var openOrders []TradeOrder
		if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			openOrders, err = b.WsCurrentOpenOrders(req.Pair, 0)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range openOrders {
				_, err = b.WsCancelOpenOrders(req.Pair, 0)
				if err != nil {
					cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders[i].OrderID, 10)] = err.Error()
				}
			}
		} else {
			openOrders, err = b.OpenOrders(ctx, req.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range openOrders {
				_, err = b.CancelExistingOrder(ctx,
					req.Pair,
					openOrders[i].OrderID,
					"")
				if err != nil {
					cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders[i].OrderID, 10)] = err.Error()
				}
			}
		}
	case asset.CoinMarginedFutures:
		if req.Pair.IsEmpty() {
			var enabledPairs currency.Pairs
			enabledPairs, err = b.GetEnabledPairs(req.AssetType)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				_, err = b.FuturesCancelAllOpenOrders(ctx, enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
			}
		} else {
			_, err = b.FuturesCancelAllOpenOrders(ctx, req.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	case asset.USDTMarginedFutures:
		var enabledPairs currency.Pairs
		if req.Pair.IsEmpty() {
			enabledPairs, err = b.GetEnabledPairs(asset.USDTMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				_, err = b.UCancelAllOpenOrders(ctx, enabledPairs[i].String())
				if err != nil {
					return cancelAllOrdersResponse, err
				}
			}
		} else {
			_, err = b.UCancelAllOpenOrders(ctx, req.Pair.String())
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	case asset.Options:
		if req.Pair.IsEmpty() {
			err = b.CancelAllOptionOrdersOnSpecificSymbol(ctx, "")
		} else {
			err = b.CancelAllOptionOrdersOnSpecificSymbol(ctx, req.Pair.String())
		}
		if err != nil {
			return cancelAllOrdersResponse, err
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Binance) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		var resp *TradeOrder
		if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			var trades []TradeOrder
			trades, err = b.WsQueryAccountOrderHistory(&AccountOrderRequestParam{
				Symbol:  pair.String(),
				OrderID: orderIDInt,
			})
			if err != nil {
				return nil, err
			}
			resp = &trades[0]
		} else {
			resp, err = b.QueryOrder(ctx, pair, "", orderIDInt)
			if err != nil {
				return nil, err
			}
		}
		var side order.Side
		side, err = order.StringToOrderSide(resp.Side)
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(resp.Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}
		orderType := order.Limit
		if resp.Type == "MARKET" {
			orderType = order.Market
		}
		return &order.Detail{
			Amount:         resp.OrigQty.Float64(),
			Exchange:       b.Name,
			OrderID:        strconv.FormatInt(resp.OrderID, 10),
			ClientOrderID:  resp.ClientOrderID,
			Side:           side,
			Type:           orderType,
			Pair:           pair,
			Cost:           resp.CummulativeQuoteQty.Float64(),
			AssetType:      assetType,
			Status:         status,
			Price:          resp.Price.Float64(),
			ExecutedAmount: resp.ExecutedQty.Float64(),
			Date:           resp.Time.Time(),
			LastUpdated:    resp.UpdateTime.Time(),
		}, nil
	case asset.CoinMarginedFutures:
		orderData, err := b.FuturesOpenOrderData(ctx, pair, orderID, "")
		if err != nil {
			return nil, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := b.GetFee(ctx, &feeBuilder)
		if err != nil {
			return nil, err
		}
		orderVars := compatibleOrderVars(orderData.Side, orderData.Status, orderData.OrderType)
		return &order.Detail{
			Amount:          orderData.OriginalQuantity,
			AssetType:       assetType,
			ClientOrderID:   orderData.ClientOrderID,
			Exchange:        b.Name,
			ExecutedAmount:  orderData.ExecutedQuantity,
			Fee:             fee,
			OrderID:         orderID,
			Pair:            pair,
			Price:           orderData.Price,
			RemainingAmount: orderData.OriginalQuantity - orderData.ExecutedQuantity,
			Side:            orderVars.Side,
			Status:          orderVars.Status,
			Type:            orderVars.OrderType,
			Date:            orderData.Time.Time(),
			LastUpdated:     orderData.UpdateTime.Time()}, nil
	case asset.USDTMarginedFutures:
		orderData, err := b.UGetOrderData(ctx, pair.String(), orderID, "")
		if err != nil {
			return nil, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := b.GetFee(ctx, &feeBuilder)
		if err != nil {
			return nil, err
		}
		orderVars := compatibleOrderVars(orderData.Side, orderData.Status, orderData.OrderType)
		return &order.Detail{
			Amount:          orderData.OriginalQuantity,
			AssetType:       assetType,
			ClientOrderID:   orderData.ClientOrderID,
			Exchange:        b.Name,
			ExecutedAmount:  orderData.ExecutedQuantity,
			Fee:             fee,
			OrderID:         orderID,
			Pair:            pair,
			Price:           orderData.Price,
			RemainingAmount: orderData.OriginalQuantity - orderData.ExecutedQuantity,
			Side:            orderVars.Side,
			Status:          orderVars.Status,
			Type:            orderVars.OrderType,
			Date:            orderData.Time.Time(),
			LastUpdated:     orderData.UpdateTime.Time(),
		}, nil
	case asset.Options:
		orderData, err := b.GetSingleEOptionsOrder(ctx, pair.String(), "", orderIDInt)
		if err != nil {
			return nil, err
		}
		oType, err := order.StringToOrderType(orderData.Type)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(orderData.Side)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(orderData.Status)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			PostOnly:             orderData.PostOnly,
			ReduceOnly:           orderData.ReduceOnly,
			Price:                orderData.Price.Float64(),
			Amount:               orderData.Quantity.Float64(),
			AverageExecutedPrice: orderData.AvgPrice.Float64(),
			QuoteAmount:          orderData.Quantity.Float64() * orderData.AvgPrice.Float64(),
			ExecutedAmount:       orderData.ExecutedQty.Float64(),
			RemainingAmount:      orderData.Quantity.Float64() - orderData.ExecutedQty.Float64(),
			Fee:                  orderData.Fee.Float64(),
			FeeAsset:             currency.NewCode(orderData.QuoteAsset),
			Exchange:             b.Name,
			OrderID:              strconv.FormatInt(orderData.OrderID, 10),
			ClientOrderID:        orderData.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            assetType,
			LastUpdated:          orderData.UpdateTime.Time(),
			Pair:                 pair,
		}, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Binance) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := b.GetDepositAddressForCurrency(ctx, cryptocurrency.String(), chain)
	if err != nil {
		return nil, err
	}

	return &deposit.Address{
		Chain:   chain,
		Address: addr.Address,
		Tag:     addr.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Binance) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := b.WithdrawCrypto(ctx,
		withdrawRequest.Currency,
		"", // withdrawal order ID
		withdrawRequest.Crypto.Chain,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Description,
		withdrawRequest.Amount, false)
	return &withdraw.ExchangeResponse{
		ID: v,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Binance) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!b.AreCredentialsValid(ctx) || b.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Binance) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 || len(req.Pairs) >= 40 {
		// sending an empty currency pair retrieves data for all currencies
		req.Pairs = append(req.Pairs, currency.EMPTYPAIR)
	}
	var orders []order.Detail
	for i := range req.Pairs {
		switch req.AssetType {
		case asset.Spot, asset.Margin:
			if req.Type == order.OCO {
				var resp []OCOOrder
				if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
					resp, err = b.WsCurrentOpenOCOOrders(defaultRecvWindow.Milliseconds())
				} else {
					resp, err = b.GetOpenOCOList(ctx)
				}
				if err != nil {
					return nil, err
				}
				for x := range resp {
					for a := range resp[x].OrderReports {
						var side order.Side
						side, err = order.StringToOrderSide(resp[x].OrderReports[a].Side)
						if err != nil {
							log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
						}
						var orderType order.Type
						orderType, err = order.StringToOrderType(resp[x].OrderReports[a].Type)
						if err != nil {
							log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
						}
						var orderStatus order.Status
						orderStatus, err = order.StringToOrderStatus(resp[x].OrderReports[a].Status)
						if err != nil {
							log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
						}
						var cp currency.Pair
						cp, err = currency.NewPairFromString(resp[x].OrderReports[a].Symbol)
						if err != nil {
							return nil, err
						}
						orders = append(orders, order.Detail{
							Exchange:        b.Name,
							Amount:          resp[x].OrderReports[a].OrigQty.Float64(),
							Price:           resp[x].OrderReports[a].Price.Float64(),
							OrderID:         strconv.FormatInt(resp[x].OrderReports[a].OrderID, 10),
							ClientOrderID:   resp[x].ListClientOrderID,
							Side:            side,
							Type:            orderType,
							Status:          orderStatus,
							Pair:            cp,
							AssetType:       req.AssetType,
							LastUpdated:     resp[x].OrderReports[a].TransactTime.Time(),
							TriggerPrice:    resp[x].OrderReports[a].StopPrice.Float64(),
							QuoteAmount:     resp[x].OrderReports[a].CummulativeQuoteQty.Float64(),
							ExecutedAmount:  resp[x].OrderReports[a].ExecutedQty.Float64(),
							RemainingAmount: resp[x].OrderReports[a].OrigQty.Float64() - resp[x].OrderReports[a].ExecutedQty.Float64(),
						})
					}
				}
			} else {
				var resp []TradeOrder
				if b.IsAPIStreamConnected() && b.Websocket.CanUseAuthenticatedEndpoints() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
					resp, err = b.WsCurrentOpenOrders(req.Pairs[i], 0)
				} else {
					resp, err = b.OpenOrders(ctx, req.Pairs[i])
				}
				if err != nil {
					return nil, err
				}
				for x := range resp {
					var side order.Side
					side, err = order.StringToOrderSide(resp[x].Side)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					var orderType order.Type
					orderType, err = order.StringToOrderType(resp[x].Type)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					orderStatus, err := order.StringToOrderStatus(resp[x].Status)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					orders = append(orders, order.Detail{
						Amount:        resp[x].OrigQty.Float64(),
						Date:          resp[x].Time.Time(),
						Exchange:      b.Name,
						OrderID:       strconv.FormatInt(resp[x].OrderID, 10),
						ClientOrderID: resp[x].ClientOrderID,
						Side:          side,
						Type:          orderType,
						Price:         resp[x].Price.Float64(),
						Status:        orderStatus,
						Pair:          req.Pairs[i],
						AssetType:     req.AssetType,
						LastUpdated:   resp[x].UpdateTime.Time(),
					})
				}
			}
		case asset.CoinMarginedFutures:
			openOrders, err := b.GetFuturesAllOpenOrders(ctx, req.Pairs[i], "")
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = openOrders[y].ExecutedQty
				feeBuilder.PurchasePrice = openOrders[y].AvgPrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := b.GetFee(ctx, &feeBuilder)
				if err != nil {
					return orders, err
				}
				orderVars := compatibleOrderVars(openOrders[y].Side, openOrders[y].Status, openOrders[y].OrderType)
				orders = append(orders, order.Detail{
					Price:           openOrders[y].Price,
					Amount:          openOrders[y].OrigQty,
					ExecutedAmount:  openOrders[y].ExecutedQty,
					RemainingAmount: openOrders[y].OrigQty - openOrders[y].ExecutedQty,
					Fee:             fee,
					Exchange:        b.Name,
					OrderID:         strconv.FormatInt(openOrders[y].OrderID, 10),
					ClientOrderID:   openOrders[y].ClientOrderID,
					Type:            orderVars.OrderType,
					Side:            orderVars.Side,
					Status:          orderVars.Status,
					Pair:            req.Pairs[i],
					AssetType:       asset.CoinMarginedFutures,
					Date:            openOrders[y].Time.Time(),
					LastUpdated:     openOrders[y].UpdateTime.Time(),
				})
			}
		case asset.USDTMarginedFutures:
			openOrders, err := b.UAllAccountOpenOrders(ctx, req.Pairs[i])
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = openOrders[y].ExecutedQuantity
				feeBuilder.PurchasePrice = openOrders[y].AveragePrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := b.GetFee(ctx, &feeBuilder)
				if err != nil {
					return orders, err
				}
				orderVars := compatibleOrderVars(openOrders[y].Side, openOrders[y].Status, openOrders[y].OrderType)
				orders = append(orders, order.Detail{
					Price:           openOrders[y].Price,
					Amount:          openOrders[y].OriginalQuantity,
					ExecutedAmount:  openOrders[y].ExecutedQuantity,
					RemainingAmount: openOrders[y].OriginalQuantity - openOrders[y].ExecutedQuantity,
					Fee:             fee,
					Exchange:        b.Name,
					OrderID:         strconv.FormatInt(openOrders[y].OrderID, 10),
					ClientOrderID:   openOrders[y].ClientOrderID,
					Type:            orderVars.OrderType,
					Side:            orderVars.Side,
					Status:          orderVars.Status,
					Pair:            req.Pairs[i],
					AssetType:       asset.USDTMarginedFutures,
					Date:            openOrders[y].Time.Time(),
					LastUpdated:     openOrders[y].UpdateTime.Time(),
				})
			}
		case asset.Options:
			openOrders, err := b.GetCurrentOpenOptionsOrders(ctx, req.Pairs[i].String(), req.StartTime, req.EndTime, 0, 0)
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				orderVars := compatibleOrderVars(openOrders[y].Side, openOrders[y].Status, openOrders[y].Type)
				orders = append(orders, order.Detail{
					Price:           openOrders[y].Price.Float64(),
					Amount:          openOrders[y].Quantity.Float64(),
					ExecutedAmount:  openOrders[y].ExecutedQty.Float64(),
					RemainingAmount: openOrders[y].Quantity.Float64() - openOrders[y].ExecutedQty.Float64(),
					Fee:             openOrders[y].Fee.Float64(),
					Exchange:        b.Name,
					OrderID:         strconv.FormatInt(openOrders[y].OrderID, 10),
					ClientOrderID:   openOrders[y].ClientOrderID,
					Type:            orderVars.OrderType,
					Side:            orderVars.Side,
					Status:          orderVars.Status,
					Pair:            req.Pairs[i],
					AssetType:       asset.USDTMarginedFutures,
					LastUpdated:     openOrders[y].UpdateTime.Time(),
				})
			}
		default:
			return orders, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
		}
	}
	return req.Filter(b.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Binance) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, fmt.Errorf("%w at least one currency is required", currency.ErrCurrencyPairsEmpty)
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot, asset.Margin:
		if req.Type == order.OCO {
			var resp []OCOOrder
			resp, err = b.GetAllOCOOrders(ctx, req.FromOrderID, req.StartTime, req.EndTime, 0)
			if err != nil {
				return nil, err
			}
			for x := range resp {
				for a := range resp[x].OrderReports {
					var side order.Side
					side, err = order.StringToOrderSide(resp[x].OrderReports[a].Side)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					var orderType order.Type
					orderType, err = order.StringToOrderType(resp[x].OrderReports[a].Type)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					var orderStatus order.Status
					orderStatus, err = order.StringToOrderStatus(resp[x].OrderReports[a].Status)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					cp, err := currency.NewPairFromString(resp[x].OrderReports[a].Symbol)
					if err != nil {
						return nil, err
					}
					orders = append(orders, order.Detail{
						Amount:        resp[x].OrderReports[a].OrigQty.Float64(),
						Exchange:      b.Name,
						OrderID:       strconv.FormatInt(resp[x].OrderReports[a].OrderID, 10),
						ClientOrderID: resp[x].ListClientOrderID,
						Side:          side,
						Type:          orderType,
						Price:         resp[x].OrderReports[a].Price.Float64(),
						Status:        orderStatus,
						Pair:          cp,
						AssetType:     req.AssetType,
						LastUpdated:   resp[x].OrderReports[a].TransactTime.Time(),
					})
				}
			}
		} else {
			for x := range req.Pairs {
				resp, err := b.AllOrders(ctx,
					req.Pairs[x],
					"",
					"1000")
				if err != nil {
					return nil, err
				}

				for i := range resp {
					var side order.Side
					side, err = order.StringToOrderSide(resp[i].Side)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					var orderType order.Type
					orderType, err = order.StringToOrderType(resp[i].Type)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					orderStatus, err := order.StringToOrderStatus(resp[i].Status)
					if err != nil {
						log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
					}
					// New orders are covered in GetOpenOrders
					if orderStatus == order.New {
						continue
					}

					var cost float64
					// For some historical orders cummulativeQuoteQty will be < 0,
					// meaning the data is not available at this time.
					if resp[i].CummulativeQuoteQty > 0 {
						cost = resp[i].CummulativeQuoteQty.Float64()
					}
					detail := order.Detail{
						Amount:          resp[i].OrigQty.Float64(),
						ExecutedAmount:  resp[i].ExecutedQty.Float64(),
						RemainingAmount: resp[i].OrigQty.Float64() - resp[i].ExecutedQty.Float64(),
						Cost:            cost,
						CostAsset:       req.Pairs[x].Quote,
						Date:            resp[i].Time.Time(),
						LastUpdated:     resp[i].UpdateTime.Time(),
						Exchange:        b.Name,
						OrderID:         strconv.FormatInt(resp[i].OrderID, 10),
						Side:            side,
						Type:            orderType,
						Price:           resp[i].Price.Float64(),
						Pair:            req.Pairs[x],
						Status:          orderStatus,
					}
					detail.InferCostsAndTimes()
					orders = append(orders, detail)
				}
			}
		}
	case asset.CoinMarginedFutures:
		for i := range req.Pairs {
			var orderHistory []FuturesOrderData
			var err error
			switch {
			case !req.StartTime.IsZero() && !req.EndTime.IsZero() && req.FromOrderID == "":
				if req.EndTime.Before(req.StartTime) {
					return nil, errors.New("endTime cannot be before startTime")
				}
				if time.Since(req.StartTime) > time.Hour*24*30 {
					return nil, errors.New("can only fetch orders 30 days out")
				}
				orderHistory, err = b.GetAllFuturesOrders(ctx,
					req.Pairs[i], currency.EMPTYPAIR, req.StartTime, req.EndTime, 0, 0)
				if err != nil {
					return nil, err
				}
			case req.FromOrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.FromOrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = b.GetAllFuturesOrders(ctx,
					req.Pairs[i], currency.EMPTYPAIR, time.Time{}, time.Time{}, fromID, 0)
				if err != nil {
					return nil, err
				}
			default:
				return nil, errors.New("invalid combination of input params")
			}
			for y := range orderHistory {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = orderHistory[y].ExecutedQty
				feeBuilder.PurchasePrice = orderHistory[y].AvgPrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := b.GetFee(ctx, &feeBuilder)
				if err != nil {
					return orders, err
				}
				orderVars := compatibleOrderVars(orderHistory[y].Side, orderHistory[y].Status, orderHistory[y].OrderType)
				orders = append(orders, order.Detail{
					Price:           orderHistory[y].Price,
					Amount:          orderHistory[y].OrigQty,
					ExecutedAmount:  orderHistory[y].ExecutedQty,
					RemainingAmount: orderHistory[y].OrigQty - orderHistory[y].ExecutedQty,
					Fee:             fee,
					Exchange:        b.Name,
					OrderID:         strconv.FormatInt(orderHistory[y].OrderID, 10),
					ClientOrderID:   orderHistory[y].ClientOrderID,
					Type:            orderVars.OrderType,
					Side:            orderVars.Side,
					Status:          orderVars.Status,
					Pair:            req.Pairs[i],
					AssetType:       asset.CoinMarginedFutures,
					Date:            orderHistory[y].Time.Time(),
				})
			}
		}
	case asset.USDTMarginedFutures:
		for i := range req.Pairs {
			var orderHistory []UFuturesOrderData
			var err error
			switch {
			case !req.StartTime.IsZero() && !req.EndTime.IsZero() && req.FromOrderID == "":
				if req.EndTime.Before(req.StartTime) {
					return nil, errors.New("endTime cannot be before startTime")
				}
				if time.Since(req.StartTime) > time.Hour*24*7 {
					return nil, errors.New("can only fetch orders 7 days out")
				}
				orderHistory, err = b.UAllAccountOrders(ctx,
					req.Pairs[i].String(), 0, 0, req.StartTime, req.EndTime)
				if err != nil {
					return nil, err
				}
			case req.FromOrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.FromOrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = b.UAllAccountOrders(ctx,
					req.Pairs[i].String(), fromID, 0, time.Time{}, time.Time{})
				if err != nil {
					return nil, err
				}
			default:
				return nil, errors.New("invalid combination of input params")
			}
			for y := range orderHistory {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = orderHistory[y].ExecutedQty
				feeBuilder.PurchasePrice = orderHistory[y].AvgPrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := b.GetFee(ctx, &feeBuilder)
				if err != nil {
					return orders, err
				}
				orderVars := compatibleOrderVars(orderHistory[y].Side, orderHistory[y].Status, orderHistory[y].OrderType)
				orders = append(orders, order.Detail{
					Price:           orderHistory[y].Price,
					Amount:          orderHistory[y].OrigQty,
					ExecutedAmount:  orderHistory[y].ExecutedQty,
					RemainingAmount: orderHistory[y].OrigQty - orderHistory[y].ExecutedQty,
					Fee:             fee,
					Exchange:        b.Name,
					OrderID:         strconv.FormatInt(orderHistory[y].OrderID, 10),
					ClientOrderID:   orderHistory[y].ClientOrderID,
					Type:            orderVars.OrderType,
					Side:            orderVars.Side,
					Status:          orderVars.Status,
					Pair:            req.Pairs[i],
					AssetType:       asset.USDTMarginedFutures,
					Date:            orderHistory[y].Time.Time(),
				})
			}
		}
	case asset.Options:
		if len(req.Pairs) == 0 {
			req.Pairs = append(req.Pairs, currency.EMPTYPAIR)
		}
		for i := range req.Pairs {
			openOrders, err := b.GetCurrentOpenOptionsOrders(ctx, req.Pairs[i].String(), req.StartTime, req.EndTime, 0, 0)
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				orderVars := compatibleOrderVars(openOrders[y].Side, openOrders[y].Status, openOrders[y].Type)
				orders = append(orders, order.Detail{
					Price:           openOrders[y].Price.Float64(),
					Amount:          openOrders[y].Quantity.Float64(),
					ExecutedAmount:  openOrders[y].ExecutedQty.Float64(),
					RemainingAmount: openOrders[y].Quantity.Float64() - openOrders[y].ExecutedQty.Float64(),
					Fee:             openOrders[y].Fee.Float64(),
					Exchange:        b.Name,
					OrderID:         strconv.FormatInt(openOrders[y].OrderID, 10),
					ClientOrderID:   openOrders[y].ClientOrderID,
					Type:            orderVars.OrderType,
					Side:            orderVars.Side,
					Status:          orderVars.Status,
					Pair:            req.Pairs[i],
					AssetType:       asset.USDTMarginedFutures,
					LastUpdated:     openOrders[y].UpdateTime.Time(),
				})
			}
		}
	default:
		return orders, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(b.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (b *Binance) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (b *Binance) FormatExchangeKlineInterval(interval kline.Interval) string {
	switch interval {
	case kline.OneDay:
		return "1d"
	case kline.ThreeDay:
		return "3d"
	case kline.OneWeek:
		return "1w"
	case kline.OneMonth:
		return "1M"
	default:
		return interval.Short()
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Binance) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	switch a {
	case asset.Spot, asset.Margin:
		var candles []CandleStick
		if b.IsAPIStreamConnected() {
			candles, err = b.GetWsOptimizedCandlestick(&KlinesRequestParams{
				Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
				Symbol:    req.RequestFormatted,
				StartTime: req.Start,
				EndTime:   req.End,
				Limit:     req.RequestLimit,
			})
		} else {
			candles, err = b.GetSpotKline(ctx, &KlinesRequestParams{
				Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
				Symbol:    req.Pair,
				StartTime: req.Start,
				EndTime:   req.End,
				Limit:     req.RequestLimit,
			})
		}
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
	case asset.USDTMarginedFutures:
		var candles []FuturesCandleStick
		candles, err = b.UKlineData(ctx,
			req.RequestFormatted.String(),
			b.FormatExchangeKlineInterval(interval),
			req.RequestLimit,
			req.Start,
			req.End)
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
	case asset.CoinMarginedFutures:
		var candles []FuturesCandleStick
		candles, err = b.GetFuturesKlineData(ctx,
			req.RequestFormatted,
			b.FormatExchangeKlineInterval(interval),
			req.RequestLimit,
			req.Start,
			req.End)
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
	case asset.Options:
		candles, err := b.GetEOptionsCandlesticks(ctx, req.RequestFormatted.String(),
			interval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}
		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].CloseTime.Time(),
				Open:   candles[i].Open.Float64(),
				High:   candles[i].High.Float64(),
				Low:    candles[i].Low.Float64(),
				Close:  candles[i].Close.Float64(),
				Volume: candles[i].Volume.Float64(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set
// time interval
func (b *Binance) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin:
			var candles []CandleStick
			if b.IsAPIStreamConnected() {
				candles, err = b.GetWsCandlestick(&KlinesRequestParams{
					Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
					Symbol:    req.RequestFormatted,
					StartTime: req.RangeHolder.Ranges[x].Start.Time,
					EndTime:   req.RangeHolder.Ranges[x].End.Time,
					Limit:     req.RequestLimit,
				})
			} else {
				candles, err = b.GetSpotKline(ctx, &KlinesRequestParams{
					Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
					Symbol:    req.Pair,
					StartTime: req.RangeHolder.Ranges[x].Start.Time,
					EndTime:   req.RangeHolder.Ranges[x].End.Time,
					Limit:     req.RequestLimit,
				})
			}
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
		case asset.USDTMarginedFutures:
			var candles []FuturesCandleStick
			candles, err = b.UKlineData(ctx,
				req.RequestFormatted.String(),
				b.FormatExchangeKlineInterval(interval),
				int64(req.RangeHolder.Limit),
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time)
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
		case asset.CoinMarginedFutures:
			var candles []FuturesCandleStick
			candles, err = b.GetFuturesKlineData(ctx,
				req.RequestFormatted,
				b.FormatExchangeKlineInterval(interval),
				int64(req.RangeHolder.Limit),
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time)
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
		case asset.Options:
			candles, err := b.GetEOptionsCandlesticks(ctx, req.RequestFormatted.String(),
				interval, req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time,
				int64(req.RangeHolder.Limit))
			if err != nil {
				return nil, err
			}
			for i := range candles {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[i].CloseTime.Time(),
					Open:   candles[i].Open.Float64(),
					High:   candles[i].High.Float64(),
					Low:    candles[i].Low.Float64(),
					Close:  candles[i].Close.Float64(),
					Volume: candles[i].Volume.Float64(),
				})
			}
		default:
			return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
		}
	}
	return req.ProcessResponse(timeSeries)
}

func compatibleOrderVars(side, status, orderType string) OrderVars {
	var resp OrderVars
	switch side {
	case order.Buy.String():
		resp.Side = order.Buy
	case order.Sell.String():
		resp.Side = order.Sell
	default:
		resp.Side = order.UnknownSide
	}
	switch status {
	case "NEW":
		resp.Status = order.New
	case "PARTIALLY_FILLED":
		resp.Status = order.PartiallyFilled
	case "FILLED":
		resp.Status = order.Filled
	case "CANCELED":
		resp.Status = order.Cancelled
	case "EXPIRED":
		resp.Status = order.Expired
	case "NEW_ADL":
		resp.Status = order.AutoDeleverage
	default:
		resp.Status = order.UnknownStatus
	}
	switch orderType {
	case "MARKET":
		resp.OrderType = order.Market
	case "LIMIT":
		resp.OrderType = order.Limit
	case "STOP":
		resp.OrderType = order.Stop
	case "TAKE_PROFIT":
		resp.OrderType = order.TakeProfit
	case "LIQUIDATION":
		resp.OrderType = order.Liquidation
	default:
		resp.OrderType = order.UnknownType
	}
	return resp
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (b *Binance) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var limits []order.MinMaxLevel
	var err error
	switch a {
	case asset.Spot,
		asset.Margin:
		limits, err = b.FetchExchangeLimits(ctx, a)
	case asset.USDTMarginedFutures:
		limits, err = b.FetchUSDTMarginExchangeLimits(ctx)
	case asset.CoinMarginedFutures:
		limits, err = b.FetchCoinMarginExchangeLimits(ctx)
	case asset.Options:
		limits, err = b.FetchOptionsExchangeLimits(ctx)
	default:
		err = fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %w", err)
	}
	return b.LoadLimits(limits)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (b *Binance) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	coinInfo, err := b.GetAllCoinsInfo(ctx)
	if err != nil {
		return nil, err
	}

	var availableChains []string
	for x := range coinInfo {
		if strings.EqualFold(coinInfo[x].Coin, cryptocurrency.String()) {
			for y := range coinInfo[x].NetworkList {
				availableChains = append(availableChains, coinInfo[x].NetworkList[y].Network)
			}
		}
	}
	return availableChains, nil
}

// FormatExchangeCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
// overrides default implementation to use optional delimiter
func (b *Binance) FormatExchangeCurrency(p currency.Pair, a asset.Item) (currency.Pair, error) {
	pairFmt, err := b.GetPairFormat(a, true)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	if a == asset.USDTMarginedFutures {
		return b.formatUSDTMarginedFuturesPair(p, pairFmt), nil
	}
	return p.Format(pairFmt), nil
}

// FormatSymbol formats the given pair to a string suitable for exchange API requests
// overrides default implementation to use optional delimiter
func (b *Binance) FormatSymbol(p currency.Pair, a asset.Item) (string, error) {
	pairFmt, err := b.GetPairFormat(a, true)
	if err != nil {
		return p.String(), err
	}
	if a == asset.USDTMarginedFutures {
		p = b.formatUSDTMarginedFuturesPair(p, pairFmt)
		return p.String(), nil
	}
	return pairFmt.Format(p), nil
}

// formatUSDTMarginedFuturesPair Binance USDTMarginedFutures pairs have a delimiter
// only if the contract has an expiry date
func (b *Binance) formatUSDTMarginedFuturesPair(p currency.Pair, pairFmt currency.PairFormat) currency.Pair {
	quote := p.Quote.String()
	for _, c := range quote {
		if c < '0' || c > '9' {
			// character rune is alphabetic, cannot be expiring contract
			return p.Format(pairFmt)
		}
	}
	pairFmt.Delimiter = currency.UnderscoreDelimiter
	return p.Format(pairFmt)
}

// GetServerTime returns the current exchange server time.
func (b *Binance) GetServerTime(ctx context.Context, ai asset.Item) (time.Time, error) {
	switch ai {
	case asset.USDTMarginedFutures:
		return b.UServerTime(ctx)
	case asset.Spot, asset.Margin:
		info, err := b.GetExchangeInfo(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return info.ServerTime.Time(), nil
	case asset.CoinMarginedFutures:
		info, err := b.FuturesExchangeInfo(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return info.ServerTime.Time(), nil
	case asset.Options:
		info, err := b.CheckEOptionsServerTime(context.Background())
		if err != nil {
			return time.Time{}, err
		}
		return info.Time(), nil
	}
	return time.Time{}, fmt.Errorf("%s %w", ai, asset.ErrNotSupported)
}

// GetLatestFundingRates returns the latest funding rates data
func (b *Binance) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}
	fPair := r.Pair
	var err error
	if !fPair.IsEmpty() {
		var format currency.PairFormat
		format, err = b.GetPairFormat(r.Asset, true)
		if err != nil {
			return nil, err
		}
		fPair = r.Pair.Format(format)
	}

	switch r.Asset {
	case asset.Spot:
	case asset.USDTMarginedFutures:
		var mp []UMarkPrice
		var fri []FundingRateInfoResponse
		fri, err = b.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}

		mp, err = b.UGetMarkPrice(ctx, fPair.String())
		if err != nil {
			return nil, err
		}
		resp := make([]fundingrate.LatestRateResponse, 0, len(mp))
		for i := range mp {
			var cp currency.Pair
			var isEnabled bool
			cp, isEnabled, err = b.MatchSymbolCheckEnabled(mp[i].Symbol, r.Asset, true)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return nil, err
			}
			if !isEnabled {
				continue
			}
			var isPerp bool
			isPerp, err = b.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}
			var fundingRateFrequency int64
			for x := range fri {
				if fri[x].Symbol != mp[i].Symbol {
					continue
				}
				fundingRateFrequency = fri[x].FundingIntervalHours
				break
			}
			nft := mp[i].NextFundingTime.Time()
			rate := fundingrate.LatestRateResponse{
				TimeChecked: time.Now(),
				Exchange:    b.Name,
				Asset:       r.Asset,
				Pair:        cp,
				LatestRate: fundingrate.Rate{
					Time: mp[i].Time.Time().Truncate(time.Hour * time.Duration(fundingRateFrequency)),
					Rate: decimal.NewFromFloat(mp[i].LastFundingRate),
				},
			}
			if nft.Year() == rate.TimeChecked.Year() {
				rate.TimeOfNextRate = nft
			}
			resp = append(resp, rate)
		}
		if len(resp) == 0 {
			return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
		return resp, nil
	case asset.CoinMarginedFutures:
		var mp []IndexMarkPrice
		mp, err = b.GetIndexAndMarkPrice(ctx, fPair.String(), "")
		if err != nil {
			return nil, err
		}
		var fri []FundingRateInfoResponse
		fri, err = b.GetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}

		resp := make([]fundingrate.LatestRateResponse, 0, len(mp))
		for i := range mp {
			var cp currency.Pair
			cp, err = currency.NewPairFromString(mp[i].Symbol)
			if err != nil {
				return nil, err
			}
			var isPerp bool
			isPerp, err = b.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}
			var fundingRateFrequency int64
			for x := range fri {
				if fri[x].Symbol != mp[i].Symbol {
					continue
				}
				fundingRateFrequency = fri[x].FundingIntervalHours
				break
			}
			nft := mp[i].NextFundingTime.Time()
			rate := fundingrate.LatestRateResponse{
				TimeChecked: time.Now(),
				Exchange:    b.Name,
				Asset:       r.Asset,
				Pair:        cp,
				LatestRate: fundingrate.Rate{
					Time: mp[i].Time.Time().Truncate(time.Duration(fundingRateFrequency) * time.Hour),
					Rate: mp[i].LastFundingRate.Decimal(),
				},
			}
			if nft.Year() == rate.TimeChecked.Year() {
				rate.TimeOfNextRate = nft
			}
			resp = append(resp, rate)
		}
		if len(resp) == 0 {
			return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%s %w", r.Asset, asset.ErrNotSupported)
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (b *Binance) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w GetFundingRates IncludePredictedRate", common.ErrFunctionNotSupported)
	}
	if !r.PaymentCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w GetFundingRates PaymentCurrency", common.ErrFunctionNotSupported)
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	format, err := b.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := r.Pair.Format(format)
	pairRate := fundingrate.HistoricalRates{
		Exchange:  b.Name,
		Asset:     r.Asset,
		Pair:      fPair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	switch r.Asset {
	case asset.USDTMarginedFutures:
		requestLimit := 1000
		sd := r.StartDate
		var fri []FundingRateInfoResponse
		fri, err = b.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		var fundingRateFrequency int64
		fps := fPair.String()
		for x := range fri {
			if fri[x].Symbol != fps {
				continue
			}
			fundingRateFrequency = fri[x].FundingIntervalHours
			break
		}
		for {
			var frh []FundingRateHistory
			frh, err = b.UGetFundingHistory(ctx, fPair.String(), int64(requestLimit), sd, r.EndDate)
			if err != nil {
				return nil, err
			}
			for j := range frh {
				pairRate.FundingRates = append(pairRate.FundingRates, fundingrate.Rate{
					Time: frh[j].FundingTime.Time(),
					Rate: decimal.NewFromFloat(frh[j].FundingRate),
				})
			}
			if len(frh) < requestLimit {
				break
			}
			sd = frh[len(frh)-1].FundingTime.Time()
		}
		var mp []UMarkPrice
		mp, err = b.UGetMarkPrice(ctx, fPair.String())
		if err != nil {
			return nil, err
		}
		pairRate.LatestRate = fundingrate.Rate{
			Time: mp[len(mp)-1].Time.Time().Truncate(time.Duration(fundingRateFrequency) * time.Hour),
			Rate: decimal.NewFromFloat(mp[len(mp)-1].LastFundingRate),
		}
		pairRate.TimeOfNextRate = mp[len(mp)-1].NextFundingTime.Time()
		if r.IncludePayments {
			var income []UAccountIncomeHistory
			income, err = b.UAccountIncomeHistory(ctx, fPair.String(), "FUNDING_FEE", int64(requestLimit), r.StartDate, r.EndDate)
			if err != nil {
				return nil, err
			}
			for j := range income {
				for x := range pairRate.FundingRates {
					tt := income[j].Time.Time()
					tt = tt.Truncate(time.Duration(fundingRateFrequency) * time.Hour)
					if !tt.Equal(pairRate.FundingRates[x].Time) {
						continue
					}
					if pairRate.PaymentCurrency.IsEmpty() {
						pairRate.PaymentCurrency = currency.NewCode(income[j].Asset)
					}
					pairRate.FundingRates[x].Payment = decimal.NewFromFloat(income[j].Income)
					pairRate.PaymentSum = pairRate.PaymentSum.Add(pairRate.FundingRates[x].Payment)
					break
				}
			}
		}
	case asset.CoinMarginedFutures:
		requestLimit := 1000
		sd := r.StartDate
		var fri []FundingRateInfoResponse
		fri, err = b.GetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		var fundingRateFrequency int64
		fps := fPair.String()
		for x := range fri {
			if fri[x].Symbol != fps {
				continue
			}
			fundingRateFrequency = fri[x].FundingIntervalHours
			break
		}
		for {
			var frh []FundingRateHistory
			frh, err = b.FuturesGetFundingHistory(ctx, fPair, int64(requestLimit), sd, r.EndDate)
			if err != nil {
				return nil, err
			}
			for j := range frh {
				pairRate.FundingRates = append(pairRate.FundingRates, fundingrate.Rate{
					Time: frh[j].FundingTime.Time(),
					Rate: decimal.NewFromFloat(frh[j].FundingRate),
				})
			}
			if len(frh) < requestLimit {
				break
			}
			sd = frh[len(frh)-1].FundingTime.Time()
		}
		var mp []IndexMarkPrice
		mp, err = b.GetIndexAndMarkPrice(ctx, fPair.String(), "")
		if err != nil {
			return nil, err
		}
		pairRate.LatestRate = fundingrate.Rate{
			Time: mp[len(mp)-1].Time.Time().Truncate(time.Duration(fundingRateFrequency) * time.Hour),
			Rate: mp[len(mp)-1].LastFundingRate.Decimal(),
		}
		pairRate.TimeOfNextRate = mp[len(mp)-1].NextFundingTime.Time()
		if r.IncludePayments {
			var income []FuturesIncomeHistoryData
			income, err = b.FuturesIncomeHistory(ctx, fPair, "FUNDING_FEE", r.StartDate, r.EndDate, int64(requestLimit))
			if err != nil {
				return nil, err
			}
			for j := range income {
				for x := range pairRate.FundingRates {
					tt := income[j].Timestamp.Time()
					tt = tt.Truncate(time.Duration(fundingRateFrequency) * time.Hour)
					if !tt.Equal(pairRate.FundingRates[x].Time) {
						continue
					}
					if pairRate.PaymentCurrency.IsEmpty() {
						pairRate.PaymentCurrency = currency.NewCode(income[j].Asset)
					}
					pairRate.FundingRates[x].Payment = decimal.NewFromFloat(income[j].Income)
					pairRate.PaymentSum = pairRate.PaymentSum.Add(pairRate.FundingRates[x].Payment)
					break
				}
			}
		}
	default:
		return nil, fmt.Errorf("%s %w", r.Asset, asset.ErrNotSupported)
	}
	return &pairRate, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (b *Binance) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	if a == asset.CoinMarginedFutures {
		return cp.Quote.Equal(currency.PERP), nil
	}
	if a == asset.USDTMarginedFutures {
		return cp.Quote.Equal(currency.USDT) || cp.Quote.Equal(currency.BUSD), nil
	}
	return false, nil
}

// SetCollateralMode sets the account's collateral mode for the asset type
func (b *Binance) SetCollateralMode(ctx context.Context, a asset.Item, collateralMode collateral.Mode) error {
	if a != asset.USDTMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	if collateralMode != collateral.MultiMode && collateralMode != collateral.SingleMode {
		return fmt.Errorf("%w %v", order.ErrCollateralInvalid, collateralMode)
	}
	return b.SetAssetsMode(ctx, collateralMode == collateral.MultiMode)
}

// GetCollateralMode returns the account's collateral mode for the asset type
func (b *Binance) GetCollateralMode(ctx context.Context, a asset.Item) (collateral.Mode, error) {
	if a != asset.USDTMarginedFutures {
		return collateral.UnknownMode, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	isMulti, err := b.GetAssetsMode(ctx)
	if err != nil {
		return collateral.UnknownMode, err
	}
	if isMulti {
		return collateral.MultiMode, nil
	}
	return collateral.SingleMode, nil
}

// SetMarginType sets the default margin type for when opening a new position
func (b *Binance) SetMarginType(ctx context.Context, item asset.Item, pair currency.Pair, tp margin.Type) error {
	if item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	if !tp.Valid() {
		return fmt.Errorf("%w %v", margin.ErrInvalidMarginType, tp)
	}
	mt, err := b.marginTypeToString(tp)
	if err != nil {
		return err
	}
	switch item {
	case asset.CoinMarginedFutures:
		_, err = b.FuturesChangeMarginType(ctx, pair, mt)
	case asset.USDTMarginedFutures:
		err = b.UChangeInitialMarginType(ctx, pair, mt)
	}
	if err != nil {
		return err
	}

	return nil
}

// ChangePositionMargin will modify a position/currencies margin parameters
func (b *Binance) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionChangeRequest", common.ErrNilPointer)
	}
	if req.Asset != asset.USDTMarginedFutures && req.Asset != asset.CoinMarginedFutures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	if req.NewAllocatedMargin == 0 {
		return nil, fmt.Errorf("%w %v %v", margin.ErrNewAllocatedMarginRequired, req.Asset, req.Pair)
	}
	if req.OriginalAllocatedMargin == 0 {
		return nil, fmt.Errorf("%w %v %v", margin.ErrOriginalPositionMarginRequired, req.Asset, req.Pair)
	}
	if req.MarginType == margin.Multi {
		return nil, fmt.Errorf("%w %v %v", margin.ErrMarginTypeUnsupported, req.Asset, req.Pair)
	}

	marginType := "add"
	if req.NewAllocatedMargin < req.OriginalAllocatedMargin {
		marginType = "reduce"
	}
	var side string
	if req.MarginSide != "" {
		side = req.MarginSide
	}
	var err error
	switch req.Asset {
	case asset.CoinMarginedFutures:
		_, err = b.ModifyIsolatedPositionMargin(ctx, req.Pair, side, marginType, req.NewAllocatedMargin)
	case asset.USDTMarginedFutures:
		_, err = b.UModifyIsolatedPositionMarginReq(ctx, req.Pair.String(), side, marginType, req.NewAllocatedMargin)
	}
	if err != nil {
		return nil, err
	}

	return &margin.PositionChangeResponse{
		Exchange:        b.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		MarginType:      req.MarginType,
		AllocatedMargin: req.NewAllocatedMargin,
	}, nil
}

// marginTypeToString converts the GCT margin type to Binance's string
func (b *Binance) marginTypeToString(mt margin.Type) (string, error) {
	switch mt {
	case margin.Isolated:
		return margin.Isolated.Upper(), nil
	case margin.Multi:
		return "CROSSED", nil
	}
	return "", fmt.Errorf("%w %v", margin.ErrInvalidMarginType, mt)
}

// GetFuturesPositionSummary returns the account's position summary for the asset type and pair
// it can be used to calculate potential positions
func (b *Binance) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionSummary", common.ErrNilPointer)
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	fPair, err := b.FormatExchangeCurrency(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	switch req.Asset {
	case asset.USDTMarginedFutures:
		ai, err := b.UAccountInformationV2(ctx)
		if err != nil {
			return nil, err
		}
		collateralMode := collateral.SingleMode
		if ai.MultiAssetsMargin {
			collateralMode = collateral.MultiMode
		}
		var accountPosition *UPosition
		var leverage, maintenanceMargin, initialMargin,
			liquidationPrice, markPrice, positionSize,
			collateralTotal, collateralUsed, collateralAvailable,
			unrealisedPNL, openPrice, isolatedMargin float64

		for i := range ai.Positions {
			if ai.Positions[i].Symbol != fPair.String() {
				continue
			}
			accountPosition = &ai.Positions[i]
			break
		}
		if accountPosition == nil {
			return nil, fmt.Errorf("%w %v %v position info", currency.ErrCurrencyNotFound, req.Asset, req.Pair)
		}

		var usdtAsset, busdAsset *UAsset
		for i := range ai.Assets {
			if usdtAsset != nil && busdAsset != nil {
				break
			}
			if strings.EqualFold(ai.Assets[i].Asset, currency.USDT.Item.Symbol) {
				usdtAsset = &ai.Assets[i]
				continue
			}
			if strings.EqualFold(ai.Assets[i].Asset, currency.BUSD.Item.Symbol) {
				busdAsset = &ai.Assets[i]
			}
		}
		if usdtAsset == nil && busdAsset == nil {
			return nil, fmt.Errorf("%w %v %v asset info", currency.ErrCurrencyNotFound, req.Asset, req.Pair)
		}

		leverage = accountPosition.Leverage
		openPrice = accountPosition.EntryPrice
		maintenanceMargin = accountPosition.MaintenanceMargin
		initialMargin = accountPosition.PositionInitialMargin
		marginType := margin.Multi
		if accountPosition.Isolated {
			marginType = margin.Isolated
		}

		var contracts []futures.Contract
		contracts, err = b.GetFuturesContractDetails(ctx, req.Asset)
		if err != nil {
			return nil, err
		}
		var contractSettlementType futures.ContractSettlementType
		for i := range contracts {
			if !contracts[i].Name.Equal(fPair) {
				continue
			}
			contractSettlementType = contracts[i].SettlementType
			break
		}

		var c currency.Code
		if collateralMode == collateral.SingleMode {
			var collateralAsset *UAsset
			if strings.Contains(accountPosition.Symbol, usdtAsset.Asset) {
				collateralAsset = usdtAsset
			} else if strings.Contains(accountPosition.Symbol, busdAsset.Asset) {
				collateralAsset = busdAsset
			}
			collateralTotal = collateralAsset.WalletBalance
			collateralAvailable = collateralAsset.AvailableBalance
			unrealisedPNL = collateralAsset.UnrealizedProfit
			c = currency.NewCode(collateralAsset.Asset)
			if marginType == margin.Multi {
				isolatedMargin = collateralAsset.CrossUnPnl
				collateralUsed = collateralTotal + isolatedMargin
			} else {
				isolatedMargin = accountPosition.IsolatedWallet
				collateralUsed = isolatedMargin
			}
		} else if collateralMode == collateral.MultiMode {
			collateralTotal = ai.TotalWalletBalance
			collateralUsed = ai.TotalWalletBalance - ai.AvailableBalance
			collateralAvailable = ai.AvailableBalance
			unrealisedPNL = accountPosition.UnrealisedProfit
		}

		var maintenanceMarginFraction decimal.Decimal
		if collateralTotal != 0 {
			maintenanceMarginFraction = decimal.NewFromFloat(maintenanceMargin).Div(decimal.NewFromFloat(collateralTotal)).Mul(decimal.NewFromInt32(100))
		}

		// binance so fun, some prices exclusively here
		positionsInfo, err := b.UPositionsInfoV2(ctx, fPair)
		if err != nil {
			return nil, err
		}
		var relevantPosition *UPositionInformationV2
		fps := fPair.String()
		for i := range positionsInfo {
			if positionsInfo[i].Symbol != fps {
				continue
			}
			relevantPosition = &positionsInfo[i]
		}
		if relevantPosition == nil {
			return nil, fmt.Errorf("%w %v %v", futures.ErrNoPositionsFound, req.Asset, req.Pair)
		}

		return &futures.PositionSummary{
			Pair:                         req.Pair,
			Asset:                        req.Asset,
			MarginType:                   marginType,
			CollateralMode:               collateralMode,
			Currency:                     c,
			ContractSettlementType:       contractSettlementType,
			IsolatedMargin:               decimal.NewFromFloat(isolatedMargin),
			Leverage:                     decimal.NewFromFloat(leverage),
			MaintenanceMarginRequirement: decimal.NewFromFloat(maintenanceMargin),
			InitialMarginRequirement:     decimal.NewFromFloat(initialMargin),
			EstimatedLiquidationPrice:    decimal.NewFromFloat(liquidationPrice),
			CollateralUsed:               decimal.NewFromFloat(collateralUsed),
			MarkPrice:                    decimal.NewFromFloat(markPrice),
			CurrentSize:                  decimal.NewFromFloat(positionSize),
			AverageOpenPrice:             decimal.NewFromFloat(openPrice),
			UnrealisedPNL:                decimal.NewFromFloat(unrealisedPNL),
			MaintenanceMarginFraction:    maintenanceMarginFraction,
			FreeCollateral:               decimal.NewFromFloat(collateralAvailable),
			TotalCollateral:              decimal.NewFromFloat(collateralTotal),
			NotionalSize:                 decimal.NewFromFloat(positionSize).Mul(decimal.NewFromFloat(markPrice)),
		}, nil
	case asset.CoinMarginedFutures:
		ai, err := b.GetFuturesAccountInfo(ctx)
		if err != nil {
			return nil, err
		}
		collateralMode := collateral.SingleMode
		var leverage, maintenanceMargin, initialMargin,
			liquidationPrice, markPrice, positionSize,
			collateralTotal, collateralUsed, collateralAvailable,
			pnl, openPrice, isolatedMargin float64

		var accountPosition *FuturesAccountInformationPosition
		fps := fPair.String()
		for i := range ai.Positions {
			if ai.Positions[i].Symbol != fps {
				continue
			}
			accountPosition = &ai.Positions[i]
			break
		}
		if accountPosition == nil {
			return nil, fmt.Errorf("%w %v %v position info", currency.ErrCurrencyNotFound, req.Asset, req.Pair)
		}
		var accountAsset *FuturesAccountAsset
		for i := range ai.Assets {
			// TODO: utilise contract data to discern the underlying currency
			// instead of having a user provide it
			if ai.Assets[i].Asset != req.UnderlyingPair.Base.Upper().String() {
				continue
			}
			accountAsset = &ai.Assets[i]
			break
		}
		if accountAsset == nil {
			return nil, fmt.Errorf("could not get asset info: %w %v %v, please verify underlying pair: '%v'", currency.ErrCurrencyNotFound, req.Asset, req.Pair, req.UnderlyingPair)
		}

		leverage = accountPosition.Leverage
		openPrice = accountPosition.EntryPrice
		maintenanceMargin = accountPosition.MaintenanceMargin
		initialMargin = accountPosition.PositionInitialMargin
		marginType := margin.Multi
		if accountPosition.Isolated {
			marginType = margin.Isolated
		}
		collateralTotal = accountAsset.WalletBalance
		frozenBalance := decimal.NewFromFloat(accountAsset.WalletBalance).Sub(decimal.NewFromFloat(accountAsset.AvailableBalance))
		collateralAvailable = accountAsset.AvailableBalance
		pnl = accountAsset.UnrealizedProfit
		if marginType == margin.Multi {
			isolatedMargin = accountAsset.CrossUnPNL
			collateralUsed = collateralTotal + isolatedMargin
		} else {
			isolatedMargin = accountPosition.IsolatedWallet
			collateralUsed = isolatedMargin
		}

		// binance so fun, some prices exclusively here
		positionsInfo, err := b.FuturesPositionsInfo(ctx, "", req.Pair.Base.String())
		if err != nil {
			return nil, err
		}
		if len(positionsInfo) == 0 {
			return nil, fmt.Errorf("%w %v", futures.ErrNoPositionsFound, fPair)
		}
		var relevantPosition *FuturesPositionInformation
		for i := range positionsInfo {
			if positionsInfo[i].Symbol != fps {
				continue
			}
			relevantPosition = &positionsInfo[i]
		}
		if relevantPosition == nil {
			return nil, fmt.Errorf("%w %v %v", futures.ErrNoPositionsFound, req.Asset, req.Pair)
		}
		liquidationPrice = relevantPosition.LiquidationPrice
		markPrice = relevantPosition.MarkPrice
		positionSize = relevantPosition.PositionAmount
		var mmf, tc decimal.Decimal
		if collateralTotal != 0 {
			tc = decimal.NewFromFloat(collateralTotal)
			mmf = decimal.NewFromFloat(maintenanceMargin).Div(tc).Mul(decimal.NewFromInt(100))
		}

		var contracts []futures.Contract
		contracts, err = b.GetFuturesContractDetails(ctx, req.Asset)
		if err != nil {
			return nil, err
		}
		var contractSettlementType futures.ContractSettlementType
		for i := range contracts {
			if !contracts[i].Name.Equal(fPair) {
				continue
			}
			contractSettlementType = contracts[i].SettlementType
			break
		}

		return &futures.PositionSummary{
			Pair:                         req.Pair,
			Asset:                        req.Asset,
			MarginType:                   marginType,
			CollateralMode:               collateralMode,
			ContractSettlementType:       contractSettlementType,
			Currency:                     currency.NewCode(accountAsset.Asset),
			IsolatedMargin:               decimal.NewFromFloat(isolatedMargin),
			NotionalSize:                 decimal.NewFromFloat(positionSize).Mul(decimal.NewFromFloat(markPrice)),
			Leverage:                     decimal.NewFromFloat(leverage),
			MaintenanceMarginRequirement: decimal.NewFromFloat(maintenanceMargin),
			InitialMarginRequirement:     decimal.NewFromFloat(initialMargin),
			EstimatedLiquidationPrice:    decimal.NewFromFloat(liquidationPrice),
			CollateralUsed:               decimal.NewFromFloat(collateralUsed),
			MarkPrice:                    decimal.NewFromFloat(markPrice),
			CurrentSize:                  decimal.NewFromFloat(positionSize),
			AverageOpenPrice:             decimal.NewFromFloat(openPrice),
			UnrealisedPNL:                decimal.NewFromFloat(pnl),
			MaintenanceMarginFraction:    mmf,
			FreeCollateral:               decimal.NewFromFloat(collateralAvailable),
			TotalCollateral:              tc,
			FrozenBalance:                frozenBalance,
		}, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
}

// GetFuturesPositionOrders returns the orders for futures positions
func (b *Binance) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionOrders", common.ErrNilPointer)
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if time.Since(req.StartDate) > b.Features.Supports.MaximumOrderHistory+time.Hour {
		if req.RespectOrderHistoryLimits {
			req.StartDate = time.Now().Add(-b.Features.Supports.MaximumOrderHistory)
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, time.Now().Add(-b.Features.Supports.MaximumOrderHistory))
		}
	}
	if req.EndDate.IsZero() {
		req.EndDate = time.Now()
	}

	var resp []futures.PositionResponse
	sd := req.StartDate
	switch req.Asset {
	case asset.USDTMarginedFutures:
		var orderLimit = 1000
		for x := range req.Pairs {
			fPair, err := b.FormatExchangeCurrency(req.Pairs[x], req.Asset)
			if err != nil {
				return nil, err
			}
			result, err := b.UPositionsInfoV2(ctx, fPair)
			if err != nil {
				return nil, err
			}
			for y := range result {
				currencyPosition := futures.PositionResponse{
					Asset: req.Asset,
					Pair:  req.Pairs[x],
				}
				for {
					var orders []UFuturesOrderData
					orders, err = b.UAllAccountOrders(ctx, fPair.String(), 0, int64(orderLimit), sd, req.EndDate)
					if err != nil {
						return nil, err
					}
					for i := range orders {
						if orders[i].Time.Time().After(req.EndDate) {
							continue
						}
						orderVars := compatibleOrderVars(orders[i].Side, orders[i].Status, orders[i].OrderType)
						var mt margin.Type
						mt, err = margin.StringToMarginType(result[y].MarginType)
						if err != nil {
							if !errors.Is(err, margin.ErrInvalidMarginType) {
								return nil, err
							}
						}
						currencyPosition.Orders = append(currencyPosition.Orders, order.Detail{
							ReduceOnly:           orders[i].ClosePosition,
							Price:                orders[i].Price,
							Amount:               orders[i].ExecutedQty,
							TriggerPrice:         orders[i].ActivatePrice,
							AverageExecutedPrice: orders[i].AvgPrice,
							ExecutedAmount:       orders[i].ExecutedQty,
							RemainingAmount:      orders[i].OrigQty - orders[i].ExecutedQty,
							CostAsset:            req.Pairs[x].Quote,
							Leverage:             result[y].Leverage,
							Exchange:             b.Name,
							OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
							ClientOrderID:        orders[i].ClientOrderID,
							Type:                 orderVars.OrderType,
							Side:                 orderVars.Side,
							Status:               orderVars.Status,
							AssetType:            asset.USDTMarginedFutures,
							Date:                 orders[i].Time.Time(),
							LastUpdated:          orders[i].UpdateTime.Time(),
							Pair:                 req.Pairs[x],
							MarginType:           mt,
						})
					}
					if len(orders) < orderLimit {
						break
					}
					sd = currencyPosition.Orders[len(currencyPosition.Orders)-1].Date
				}
				resp = append(resp, currencyPosition)
			}
		}
	case asset.CoinMarginedFutures:
		var orderLimit = 100
		for x := range req.Pairs {
			fPair, err := b.FormatExchangeCurrency(req.Pairs[x], req.Asset)
			if err != nil {
				return nil, err
			}
			// "pair" for coinmarginedfutures is the pair.Base
			// eg ADAUSD_PERP the pair is ADAUSD
			result, err := b.FuturesPositionsInfo(ctx, "", fPair.Base.String())
			if err != nil {
				return nil, err
			}
			currencyPosition := futures.PositionResponse{
				Asset: req.Asset,
				Pair:  req.Pairs[x],
			}
			for y := range result {
				if result[y].PositionAmount == 0 {
					continue
				}
				for {
					var orders []FuturesOrderData
					orders, err = b.GetAllFuturesOrders(ctx, fPair, currency.EMPTYPAIR, sd, req.EndDate, 0, int64(orderLimit))
					if err != nil {
						return nil, err
					}
					for i := range orders {
						if orders[i].Time.Time().After(req.EndDate) {
							continue
						}
						var orderPair currency.Pair
						orderPair, err = currency.NewPairFromString(orders[i].Pair)
						if err != nil {
							return nil, err
						}
						orderVars := compatibleOrderVars(orders[i].Side, orders[i].Status, orders[i].OrderType)
						var mt margin.Type
						mt, err = margin.StringToMarginType(result[y].MarginType)
						if err != nil {
							if !errors.Is(err, margin.ErrInvalidMarginType) {
								return nil, err
							}
						}
						currencyPosition.Orders = append(currencyPosition.Orders, order.Detail{
							ReduceOnly:           orders[i].ClosePosition,
							Price:                orders[i].Price,
							Amount:               orders[i].ExecutedQty,
							TriggerPrice:         orders[i].ActivatePrice,
							AverageExecutedPrice: orders[i].AvgPrice,
							ExecutedAmount:       orders[i].ExecutedQty,
							RemainingAmount:      orders[i].OrigQty - orders[i].ExecutedQty,
							Leverage:             result[y].Leverage,
							CostAsset:            orderPair.Base,
							Exchange:             b.Name,
							OrderID:              strconv.FormatInt(orders[i].OrderID, 10),
							ClientOrderID:        orders[i].ClientOrderID,
							Type:                 orderVars.OrderType,
							Side:                 orderVars.Side,
							Status:               orderVars.Status,
							AssetType:            asset.CoinMarginedFutures,
							Date:                 orders[i].Time.Time(),
							LastUpdated:          orders[i].UpdateTime.Time(),
							Pair:                 req.Pairs[x],
							MarginType:           mt,
						})
					}
					if len(orders) < orderLimit {
						break
					}
					sd = currencyPosition.Orders[len(currencyPosition.Orders)-1].Date
				}
				resp = append(resp, currencyPosition)
			}
		}
	default:
		return nil, fmt.Errorf("%w futures position for %v is not", asset.ErrNotSupported, req.Asset)
	}
	return resp, nil
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (b *Binance) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, amount float64, _ order.Side) error {
	switch item {
	case asset.USDTMarginedFutures:
		_, err := b.UChangeInitialLeverageRequest(ctx, pair.String(), amount)
		return err
	case asset.CoinMarginedFutures:
		_, err := b.FuturesChangeInitialLeverage(ctx, pair, amount)
		return err
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (b *Binance) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, _ order.Side) (float64, error) {
	if pair.IsEmpty() {
		return -1, currency.ErrCurrencyPairEmpty
	}
	switch item {
	case asset.USDTMarginedFutures:
		resp, err := b.UPositionsInfoV2(ctx, pair)
		if err != nil {
			return -1, err
		}
		if len(resp) == 0 {
			return -1, fmt.Errorf("%w %v %v", futures.ErrPositionNotFound, item, pair)
		}
		// leverage is the same across positions
		return resp[0].Leverage, nil
	case asset.CoinMarginedFutures:
		resp, err := b.FuturesPositionsInfo(ctx, "", pair.Base.String())
		if err != nil {
			return -1, err
		}
		if len(resp) == 0 {
			return -1, fmt.Errorf("%w %v %v", futures.ErrPositionNotFound, item, pair)
		}
		// leverage is the same across positions
		return resp[0].Leverage, nil
	default:
		return -1, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetFuturesContractDetails returns details about futures contracts
func (b *Binance) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	switch item {
	case asset.USDTMarginedFutures:
		fri, err := b.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}

		ei, err := b.UExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(ei.Symbols))
		for i := range ei.Symbols {
			var fundingRateFloor, fundingRateCeil decimal.Decimal
			for j := range fri {
				if fri[j].Symbol != ei.Symbols[i].Symbol {
					continue
				}
				fundingRateFloor = fri[j].AdjustedFundingRateFloor.Decimal()
				fundingRateCeil = fri[j].AdjustedFundingRateCap.Decimal()
				break
			}
			var cp currency.Pair
			cp, err = currency.NewPairFromStrings(ei.Symbols[i].BaseAsset, ei.Symbols[i].Symbol[len(ei.Symbols[i].BaseAsset):])
			if err != nil {
				return nil, err
			}
			var ct futures.ContractType
			var ed time.Time
			if cp.Quote.Equal(currency.USDT) || cp.Quote.Equal(currency.BUSD) {
				ct = futures.Perpetual
			} else {
				ct = futures.Quarterly
				ed = ei.Symbols[i].DeliveryDate.Time()
			}
			resp = append(resp, futures.Contract{
				Exchange:           b.Name,
				Name:               cp,
				Underlying:         currency.NewPair(currency.NewCode(ei.Symbols[i].BaseAsset), currency.NewCode(ei.Symbols[i].QuoteAsset)),
				Asset:              item,
				SettlementType:     futures.Linear,
				StartDate:          ei.Symbols[i].OnboardDate.Time(),
				EndDate:            ed,
				IsActive:           ei.Symbols[i].Status == "TRADING",
				Status:             ei.Symbols[i].Status,
				MarginCurrency:     currency.NewCode(ei.Symbols[i].MarginAsset),
				Type:               ct,
				FundingRateFloor:   fundingRateFloor,
				FundingRateCeiling: fundingRateCeil,
			})
		}
		return resp, nil
	case asset.CoinMarginedFutures:
		fri, err := b.GetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		ei, err := b.FuturesExchangeInfo(ctx)
		if err != nil {
			return nil, err
		}

		resp := make([]futures.Contract, 0, len(ei.Symbols))
		for i := range ei.Symbols {
			var fundingRateFloor, fundingRateCeil decimal.Decimal
			for j := range fri {
				if fri[j].Symbol != ei.Symbols[i].Symbol {
					continue
				}
				fundingRateFloor = fri[j].AdjustedFundingRateFloor.Decimal()
				fundingRateCeil = fri[j].AdjustedFundingRateCap.Decimal()
				break
			}
			var cp currency.Pair
			cp, err = currency.NewPairFromString(ei.Symbols[i].Symbol)
			if err != nil {
				return nil, err
			}

			var ct futures.ContractType
			var ed time.Time
			if cp.Quote.Equal(currency.PERP) {
				ct = futures.Perpetual
			} else {
				ct = futures.Quarterly
				ed = ei.Symbols[i].DeliveryDate.Time()
			}
			resp = append(resp, futures.Contract{
				Exchange:           b.Name,
				Name:               cp,
				Underlying:         currency.NewPair(currency.NewCode(ei.Symbols[i].BaseAsset), currency.NewCode(ei.Symbols[i].QuoteAsset)),
				Asset:              item,
				StartDate:          ei.Symbols[i].OnboardDate.Time(),
				EndDate:            ed,
				IsActive:           ei.Symbols[i].ContractStatus == "TRADING",
				MarginCurrency:     currency.NewCode(ei.Symbols[i].MarginAsset),
				SettlementType:     futures.Inverse,
				Type:               ct,
				FundingRateFloor:   fundingRateFloor,
				FundingRateCeiling: fundingRateCeil,
			})
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (b *Binance) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	if len(k) == 0 {
		return nil, fmt.Errorf("%w requires pair", common.ErrFunctionNotSupported)
	}
	for i := range k {
		if k[i].Asset != asset.USDTMarginedFutures && k[i].Asset != asset.CoinMarginedFutures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	result := make([]futures.OpenInterest, len(k))
	for i := range k {
		switch k[i].Asset {
		case asset.USDTMarginedFutures:
			oi, err := b.UOpenInterest(ctx, k[i].Pair().String())
			if err != nil {
				return nil, err
			}
			result[i] = futures.OpenInterest{
				Key: key.ExchangePairAsset{
					Exchange: b.Name,
					Base:     k[i].Base,
					Quote:    k[i].Quote,
					Asset:    k[i].Asset,
				},
				OpenInterest: oi.OpenInterest,
			}
		case asset.CoinMarginedFutures:
			oi, err := b.OpenInterest(ctx, k[i].Pair())
			if err != nil {
				return nil, err
			}
			result[i] = futures.OpenInterest{
				Key: key.ExchangePairAsset{
					Exchange: b.Name,
					Base:     k[i].Base,
					Quote:    k[i].Quote,
					Asset:    k[i].Asset,
				},
				OpenInterest: oi.OpenInterest,
			}
		}
	}
	return result, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *Binance) GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := b.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	symbol, err := b.FormatSymbol(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.USDTMarginedFutures:
		var ct string
		if !cp.Quote.Equal(currency.USDT) && !cp.Quote.Equal(currency.BUSD) {
			ei, err := b.UExchangeInfo(ctx)
			if err != nil {
				return "", err
			}
			for i := range ei.Symbols {
				if ei.Symbols[i].Symbol != symbol {
					continue
				}
				switch ei.Symbols[i].ContractType {
				case "CURRENT_QUARTER":
					ct = "_QUARTER"
				case "NEXT_QUARTER":
					ct = "_BI-QUARTER"
				}
				symbol = ei.Symbols[i].Pair
				break
			}
		}
		return tradeBaseURL + "futures/" + symbol + ct, nil
	case asset.CoinMarginedFutures:
		var ct string
		if !cp.Quote.Equal(currency.USDT) && !cp.Quote.Equal(currency.BUSD) {
			ei, err := b.FuturesExchangeInfo(ctx)
			if err != nil {
				return "", err
			}
			for i := range ei.Symbols {
				if ei.Symbols[i].Symbol != symbol {
					continue
				}
				switch ei.Symbols[i].ContractType {
				case "CURRENT_QUARTER":
					ct = "_QUARTER"
				case "NEXT_QUARTER":
					ct = "_BI-QUARTER"
				}
				symbol = ei.Symbols[i].Pair
				break
			}
		}
		return tradeBaseURL + "delivery/" + symbol + ct, nil
	case asset.Spot:
		return tradeBaseURL + "trade/" + symbol + "?type=spot", nil
	case asset.Margin:
		return tradeBaseURL + "trade/" + symbol + "?type=cross", nil
	case asset.Options:
		var underlying string
		ei, err := b.GetOptionsExchangeInformation(ctx)
		if err != nil {
			return "", err
		}
		for i := range ei.OptionSymbols {
			if ei.OptionSymbols[i].Symbol != symbol {
				continue
			}
			underlying = ei.OptionSymbols[i].Underlying
			break
		}
		return tradeBaseURL + "eoptions/" + underlying + "/" + symbol, nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
