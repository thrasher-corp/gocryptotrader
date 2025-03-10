package bitget

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (bi *Bitget) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
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
		err = bi.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Bitget
func (bi *Bitget) SetDefaults() {
	bi.Name = "Bitget"
	bi.Enabled = true
	bi.Verbose = true
	bi.API.CredentialsValidator.RequiresKey = true
	bi.API.CredentialsValidator.RequiresSecret = true
	bi.API.CredentialsValidator.RequiresClientID = true
	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := bi.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Futures, asset.Margin, asset.CrossMargin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	bi.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                    true, // Supported for spot and futures, but not margin
				AutoPairUpdates:                   true,
				AccountBalance:                    true,
				CryptoDeposit:                     true,
				CryptoWithdrawal:                  true,
				FiatWithdraw:                      false,
				GetOrder:                          true,
				GetOrders:                         true,
				CancelOrders:                      true,
				CancelOrder:                       true,
				SubmitOrder:                       true,
				SubmitOrders:                      true,
				ModifyOrder:                       true,
				DepositHistory:                    true,
				WithdrawalHistory:                 true,
				TradeHistory:                      true,
				UserTradeHistory:                  true,
				TradeFee:                          true,
				FiatDepositFee:                    false,
				FiatWithdrawalFee:                 false,
				CryptoDepositFee:                  false,
				CryptoWithdrawalFee:               false,
				TickerFetching:                    true,
				KlineFetching:                     true,
				TradeFetching:                     true,
				OrderbookFetching:                 true,
				AccountInfo:                       true,
				FiatDeposit:                       false,
				DeadMansSwitch:                    false,
				FundingRateFetching:               true,
				AuthenticatedEndpoints:            true,
				CandleHistory:                     true,
				MultiChainDeposits:                true,
				MultiChainWithdrawals:             true,
				MultiChainDepositRequiresChainSet: true,
				HasAssetTypeAccountSegregation:    true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerBatching:                 false,
				AccountBalance:                 true,
				CryptoDeposit:                  false,
				CryptoWithdrawal:               false,
				FiatWithdraw:                   false,
				GetOrder:                       false,
				GetOrders:                      true,
				CancelOrders:                   false,
				CancelOrder:                    false,
				SubmitOrder:                    false,
				SubmitOrders:                   false,
				ModifyOrder:                    false,
				DepositHistory:                 false,
				WithdrawalHistory:              false,
				TradeHistory:                   false,
				UserTradeHistory:               false,
				TradeFee:                       false,
				FiatDepositFee:                 false,
				FiatWithdrawalFee:              false,
				CryptoDepositFee:               false,
				CryptoWithdrawalFee:            false,
				TickerFetching:                 true,
				KlineFetching:                  true,
				TradeFetching:                  true,
				OrderbookFetching:              true,
				AccountInfo:                    true,
				FiatDeposit:                    false,
				DeadMansSwitch:                 false,
				FundingRateFetching:            false,
				PredictedFundingRate:           false,
				Subscribe:                      true,
				Unsubscribe:                    true,
				AuthenticatedEndpoints:         true,
				MessageCorrelation:             false,
				MessageSequenceNumbers:         false,
				CandleHistory:                  false,
				MultiChainDeposits:             false,
				MultiChainWithdrawals:          false,
				HasAssetTypeAccountSegregation: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
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
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 200,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
		TradingRequirements: protocol.TradingRequirements{
			SpotMarketOrderAmountPurchaseQuotationOnly: false,
			SpotMarketOrderAmountSellBaseOnly:          true,
			ClientOrderID:                              false,
		},
	}
	bi.Requester, err = request.New(bi.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	bi.API.Endpoints = bi.NewEndpoints()
	err = bi.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      bitgetAPIURL,
		exchange.WebsocketSpot: bitgetPublicWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	bi.Websocket = stream.NewWebsocket()
	bi.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	bi.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	bi.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (bi *Bitget) Setup(exch *config.Exchange) error {
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
	wsRunningEndpoint, err := bi.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = bi.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:                         exch,
		DefaultURL:                             bitgetPublicWSURL,
		RunningURL:                             wsRunningEndpoint,
		Connector:                              bi.WsConnect,
		Subscriber:                             bi.Subscribe,
		Unsubscriber:                           bi.Unsubscribe,
		GenerateSubscriptions:                  bi.generateDefaultSubscriptions,
		Features:                               &bi.Features.Supports.WebsocketCapabilities,
		MaxWebsocketSubscriptionsPerConnection: 240,
		OrderbookBufferConfig: buffer.Config{
			Checksum: bi.CalculateUpdateOrderbookChecksum,
		},
		RateLimitDefinitions: GetRateLimits(),
	})
	if err != nil {
		return err
	}
	bi.Websocket.Conn = &stream.WebsocketConnection{
		ExchangeName:     bi.Name,
		URL:              bi.Websocket.GetWebsocketURL(),
		ProxyURL:         bi.Websocket.GetProxyAddress(),
		Verbose:          bi.Verbose,
		ResponseMaxLimit: exch.WebsocketResponseMaxLimit,
	}
	err = bi.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		URL:                  bitgetPublicWSURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            GetRateLimits()[RateSubscription],
	})
	if err != nil {
		return err
	}
	return bi.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		URL:                  bitgetPrivateWSURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
		RateLimit:            GetRateLimits()[RateSubscription],
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (bi *Bitget) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot:
		resp, err := bi.GetSymbolInfo(ctx, currency.Pair{})
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(resp))
		var filter int
		for x := range resp {
			if (resp[x].PricePrecision == 0 && resp[x].QuantityPrecision == 0 && resp[x].QuotePrecision == 0) || resp[x].OpenTime.Time().After(time.Now().Add(time.Hour*24*365)) {
				continue
			}
			pair := currency.NewPair(resp[x].BaseCoin, resp[x].QuoteCoin)
			pairs[filter] = pair
			filter++
		}
		return pairs[:filter], nil
	case asset.Futures:
		var resp []FutureTickerResp
		req := []string{"USDT-FUTURES", "COIN-FUTURES", "USDC-FUTURES"}
		for x := range req {
			resp2, err := bi.GetAllFuturesTickers(ctx, req[x])
			if err != nil {
				return nil, err
			}
			resp = append(resp, resp2...)
		}
		pairs := make(currency.Pairs, len(resp))
		for x := range resp {
			pair, err := pairFromStringHelper(resp[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs[x] = pair
		}
		return pairs, nil
	case asset.Margin, asset.CrossMargin:
		resp, err := bi.GetSupportedCurrencies(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(resp))
		for x := range resp {
			pairs[x] = currency.NewPair(resp[x].BaseCoin, resp[x].QuoteCoin)
		}
		return pairs, nil
	}
	return nil, asset.ErrNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (bi *Bitget) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := bi.GetAssetTypes(false)
	for x := range assetTypes {
		pairs, err := bi.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}
		for i := range pairs {
			pairs[i], err = bi.FormatExchangeCurrency(pairs[i], assetTypes[x])
			if err != nil {
				return err
			}
		}
		err = bi.UpdatePairs(pairs, assetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (bi *Bitget) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	// tickerPrice := new(ticker.Price)
	var tickerPrice *ticker.Price
	p, err := bi.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		tick, err := bi.GetSpotTickerInformation(ctx, p)
		if err != nil {
			return nil, err
		}
		if len(tick) == 0 {
			return nil, errReturnEmpty
		}
		tickerPrice = &ticker.Price{
			High:         tick[0].High24H,
			Low:          tick[0].Low24H,
			Bid:          tick[0].BidPrice,
			Ask:          tick[0].AskPrice,
			Volume:       tick[0].BaseVolume,
			QuoteVolume:  tick[0].QuoteVolume,
			Open:         tick[0].Open,
			Close:        tick[0].LastPrice,
			LastUpdated:  tick[0].Timestamp.Time(),
			ExchangeName: bi.Name,
			AssetType:    assetType,
			Pair:         p,
		}
	case asset.Futures:
		tick, err := bi.GetFuturesTicker(ctx, p, getProductType(p))
		if err != nil {
			return nil, err
		}
		if len(tick) == 0 {
			return nil, errReturnEmpty
		}
		tickerPrice = &ticker.Price{
			High:         tick[0].High24H,
			Low:          tick[0].Low24H,
			Bid:          tick[0].BidPrice,
			Ask:          tick[0].AskPrice,
			Volume:       tick[0].BaseVolume,
			QuoteVolume:  tick[0].QuoteVolume,
			Open:         tick[0].Open24H,
			Close:        tick[0].LastPrice,
			IndexPrice:   tick[0].IndexPrice,
			LastUpdated:  tick[0].Timestamp.Time(),
			ExchangeName: bi.Name,
			AssetType:    assetType,
			Pair:         p,
		}
	case asset.Margin, asset.CrossMargin:
		tick, err := bi.GetSpotCandlestickData(ctx, p, formatExchangeKlineIntervalSpot(kline.OneDay), time.Now().Add(-time.Hour*24), time.Now(), 2, false)
		if err != nil {
			return nil, err
		}
		if len(tick.SpotCandles) == 0 {
			return nil, errReturnEmpty
		}
		tickerPrice = &ticker.Price{
			High:         tick.SpotCandles[0].High,
			Low:          tick.SpotCandles[0].Low,
			Volume:       tick.SpotCandles[0].BaseVolume,
			QuoteVolume:  tick.SpotCandles[0].QuoteVolume,
			Open:         tick.SpotCandles[0].Open,
			Close:        tick.SpotCandles[0].Close,
			LastUpdated:  tick.SpotCandles[0].Timestamp,
			ExchangeName: bi.Name,
			AssetType:    assetType,
			Pair:         p,
		}
	default:
		return nil, asset.ErrNotSupported
	}
	tickerPrice.Pair = p
	tickerPrice.ExchangeName = bi.Name
	tickerPrice.AssetType = assetType
	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(bi.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (bi *Bitget) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot:
		tick, err := bi.GetSpotTickerInformation(ctx, currency.Pair{})
		if err != nil {
			return err
		}
		var filter int
		newTick := make([]TickerResp, len(tick))
		for i := range tick {
			if tick[i].Symbol == "BABYBONKUSDT" || tick[i].Symbol == "CARUSDT" {
				continue
			}
			newTick[filter] = tick[i]
			filter++
		}
		newTick = newTick[:filter]
		for x := range newTick {
			p, err := bi.MatchSymbolWithAvailablePairs(newTick[x].Symbol, assetType, false)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				High:         newTick[x].High24H,
				Low:          newTick[x].Low24H,
				Bid:          newTick[x].BidPrice,
				Ask:          newTick[x].AskPrice,
				Volume:       newTick[x].BaseVolume,
				QuoteVolume:  newTick[x].QuoteVolume,
				Open:         newTick[x].Open,
				Close:        newTick[x].LastPrice,
				LastUpdated:  newTick[x].Timestamp.Time(),
				Pair:         p,
				ExchangeName: bi.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		for i := range prodTypes {
			tick, err := bi.GetAllFuturesTickers(ctx, prodTypes[i])
			if err != nil {
				return err
			}
			for x := range tick {
				p, err := bi.MatchSymbolWithAvailablePairs(tick[x].Symbol, assetType, false)
				if err != nil {
					return err
				}
				err = ticker.ProcessTicker(&ticker.Price{
					High:         tick[x].High24H,
					Low:          tick[x].Low24H,
					Bid:          tick[x].BidPrice,
					Ask:          tick[x].AskPrice,
					Volume:       tick[x].BaseVolume,
					QuoteVolume:  tick[x].QuoteVolume,
					Open:         tick[x].Open24H,
					Close:        tick[x].LastPrice,
					IndexPrice:   tick[x].IndexPrice,
					LastUpdated:  tick[x].Timestamp.Time(),
					Pair:         p,
					ExchangeName: bi.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return err
				}
			}
		}
	case asset.Margin, asset.CrossMargin:
		pairs, err := bi.GetSupportedCurrencies(ctx)
		if err != nil {
			return err
		}
		check, err := bi.GetSymbolInfo(ctx, currency.Pair{})
		if err != nil {
			return err
		}
		checkSlice := make([]string, len(check))
		var filter int
		for i := range check {
			if (check[i].PricePrecision == 0 && check[i].QuantityPrecision == 0 && check[i].QuotePrecision == 0) || check[i].OpenTime.Time().After(time.Now().Add(time.Hour)) {
				continue
			}
			checkSlice[filter] = check[i].Symbol
			filter++
		}
		checkSlice = checkSlice[:filter]
		for x := range pairs {
			if !slices.Contains(checkSlice, pairs[x].Symbol) {
				continue
			}
			p, err := bi.MatchSymbolWithAvailablePairs(pairs[x].Symbol, assetType, false)
			if err != nil {
				return err
			}
			p, err = bi.FormatExchangeCurrency(p, assetType)
			if err != nil {
				return err
			}
			resp, err := bi.GetSpotCandlestickData(ctx, p, formatExchangeKlineIntervalSpot(kline.OneDay), time.Now().Add(-time.Hour*24), time.Now(), 2, false)
			if err != nil {
				return err
			}
			if len(resp.SpotCandles) == 0 {
				return errReturnEmpty
			}
			err = ticker.ProcessTicker(&ticker.Price{
				High:         resp.SpotCandles[0].High,
				Low:          resp.SpotCandles[0].Low,
				Volume:       resp.SpotCandles[0].BaseVolume,
				QuoteVolume:  resp.SpotCandles[0].QuoteVolume,
				Open:         resp.SpotCandles[0].Open,
				Close:        resp.SpotCandles[0].Close,
				LastUpdated:  resp.SpotCandles[0].Timestamp,
				Pair:         p,
				ExchangeName: bi.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (bi *Bitget) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(bi.Name, p, assetType)
	if err != nil {
		return bi.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (bi *Bitget) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(bi.Name, pair, assetType)
	if err != nil {
		return bi.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (bi *Bitget) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        bi.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: bi.CanVerifyOrderbook,
		MaxDepth:        150,
	}
	pair, err := bi.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		orderbookNew, err := bi.GetOrderbookDepth(ctx, pair, "", 150)
		if err != nil {
			return book, err
		}
		book.Bids = make([]orderbook.Tranche, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			book.Bids[x].Amount = orderbookNew.Bids[x][1].Float64()
			book.Bids[x].Price = orderbookNew.Bids[x][0].Float64()
		}
		book.Asks = make([]orderbook.Tranche, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			book.Asks[x].Amount = orderbookNew.Asks[x][1].Float64()
			book.Asks[x].Price = orderbookNew.Asks[x][0].Float64()
		}
	case asset.Futures:
		orderbookNew, err := bi.GetFuturesMergeDepth(ctx, pair, getProductType(pair), "", "max")
		if err != nil {
			return book, err
		}
		book.Bids = make([]orderbook.Tranche, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			book.Bids[x].Amount = orderbookNew.Bids[x][1]
			book.Bids[x].Price = orderbookNew.Bids[x][0]
		}
		book.Asks = make([]orderbook.Tranche, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			book.Asks[x].Amount = orderbookNew.Asks[x][1]
			book.Asks[x].Price = orderbookNew.Asks[x][0]
		}
	default:
		return book, asset.ErrNotSupported
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(bi.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (bi *Bitget) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc := account.Holdings{
		Exchange: bi.Name,
	}
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return acc, err
	}
	switch assetType {
	case asset.Spot:
		resp, err := bi.GetAccountAssets(ctx, currency.Code{}, "")
		if err != nil {
			return acc, err
		}
		acc.Accounts = make([]account.SubAccount, 1)
		acc.Accounts[0].Currencies = make([]account.Balance, len(resp))
		for x := range resp {
			acc.Accounts[0].Currencies[x].Currency = resp[x].Coin
			acc.Accounts[0].Currencies[x].Hold = resp[x].Frozen + resp[x].Locked + resp[x].LimitAvailable
			acc.Accounts[0].Currencies[x].Total = resp[x].Available + acc.Accounts[0].Currencies[x].Hold
			acc.Accounts[0].Currencies[x].Free = resp[x].Available
		}
	case asset.Futures:
		acc.Accounts = make([]account.SubAccount, len(prodTypes))
		for i := range prodTypes {
			resp, err := bi.GetAllFuturesAccounts(ctx, prodTypes[i])
			if err != nil {
				return acc, err
			}
			acc.Accounts[i].Currencies = make([]account.Balance, len(resp))
			for x := range resp {
				acc.Accounts[i].Currencies[x].Currency = resp[x].MarginCoin
				acc.Accounts[i].Currencies[x].Hold = resp[x].Locked
				acc.Accounts[i].Currencies[x].Total = resp[x].Locked + resp[x].Available
				acc.Accounts[i].Currencies[x].Free = resp[x].Available
			}
		}
	case asset.Margin:
		resp, err := bi.GetIsolatedAccountAssets(ctx, currency.Pair{})
		if err != nil {
			return acc, err
		}
		acc.Accounts = make([]account.SubAccount, 1)
		acc.Accounts[0].Currencies = make([]account.Balance, len(resp))
		for x := range resp {
			acc.Accounts[0].Currencies[x].Currency = resp[x].Coin
			acc.Accounts[0].Currencies[x].Hold = resp[x].Frozen
			acc.Accounts[0].Currencies[x].Total = resp[x].TotalAmount
			acc.Accounts[0].Currencies[x].Free = resp[x].Available
			acc.Accounts[0].Currencies[x].Borrowed = resp[x].Borrow
		}
	case asset.CrossMargin:
		resp, err := bi.GetCrossAccountAssets(ctx, currency.Code{})
		if err != nil {
			return acc, err
		}
		acc.Accounts = make([]account.SubAccount, 1)
		acc.Accounts[0].Currencies = make([]account.Balance, len(resp))
		for x := range resp {
			acc.Accounts[0].Currencies[x].Currency = resp[x].Coin
			acc.Accounts[0].Currencies[x].Hold = resp[x].Frozen
			acc.Accounts[0].Currencies[x].Total = resp[x].TotalAmount
			acc.Accounts[0].Currencies[x].Free = resp[x].Available
			acc.Accounts[0].Currencies[x].Borrowed = resp[x].Borrow
		}
	default:
		return acc, asset.ErrNotSupported
	}
	ID, err := bi.GetAccountInfo(ctx)
	if err != nil {
		return acc, err
	}
	for x := range acc.Accounts {
		acc.Accounts[x].ID = strconv.FormatUint(ID.UserID, 10)
		acc.Accounts[x].AssetType = assetType
	}
	err = account.Process(&acc, creds)
	if err != nil {
		return acc, err
	}
	return acc, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (bi *Bitget) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
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

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (bi *Bitget) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	// This exchange only allows requests covering the last 90 days
	resp, err := bi.withdrawalHistGrabber(ctx, currency.Code{})
	if err != nil {
		return nil, err
	}
	funHist := make([]exchange.FundingHistory, len(resp))
	for x := range resp {
		funHist[x] = exchange.FundingHistory{
			ExchangeName:      bi.Name,
			Status:            resp[x].Status,
			TransferID:        strconv.FormatInt(resp[x].OrderID, 10),
			Timestamp:         resp[x].CreationTime.Time(),
			Currency:          resp[x].Coin.String(),
			Amount:            resp[x].Size,
			TransferType:      "Withdrawal",
			CryptoToAddress:   resp[x].ToAddress,
			CryptoFromAddress: resp[x].FromAddress,
			CryptoChain:       resp[x].Chain,
		}
		if resp[x].Destination == "on_chain" {
			funHist[x].CryptoTxID = strconv.FormatInt(resp[x].TradeID, 10)
		}
	}
	var pagination int64
	pagination = 0
	for {
		resp, err := bi.GetDepositRecords(ctx, currency.Code{}, 0, pagination, 100, time.Now().Add(-time.Hour*24*90), time.Now())
		if err != nil {
			return nil, err
		}
		// Not sure that this is the right end to use for pagination
		if len(resp) == 0 || pagination == resp[len(resp)-1].OrderID {
			break
		}
		pagination = resp[len(resp)-1].OrderID
		tempHist := make([]exchange.FundingHistory, len(resp))
		for x := range resp {
			tempHist[x] = exchange.FundingHistory{
				ExchangeName:      bi.Name,
				Status:            resp[x].Status,
				TransferID:        strconv.FormatInt(resp[x].OrderID, 10),
				Timestamp:         resp[x].CreationTime.Time(),
				Currency:          resp[x].Coin.String(),
				Amount:            resp[x].Size,
				TransferType:      "Deposit",
				CryptoToAddress:   resp[x].ToAddress,
				CryptoFromAddress: resp[x].FromAddress,
				CryptoChain:       resp[x].Chain,
			}
			if resp[x].Destination == "on_chain" {
				tempHist[x].CryptoTxID = strconv.FormatInt(resp[x].TradeID, 10)
			}
		}
		funHist = slices.Concat(funHist, tempHist)
	}
	return funHist, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (bi *Bitget) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	// This exchange only allows requests covering the last 90 days
	resp, err := bi.withdrawalHistGrabber(ctx, c)
	if err != nil {
		return nil, err
	}
	funHist := make([]exchange.WithdrawalHistory, len(resp))
	for x := range resp {
		funHist[x] = exchange.WithdrawalHistory{
			Status:          resp[x].Status,
			TransferID:      strconv.FormatInt(resp[x].OrderID, 10),
			Timestamp:       resp[x].CreationTime.Time(),
			Currency:        resp[x].Coin.String(),
			Amount:          resp[x].Size,
			TransferType:    "Withdrawal",
			CryptoToAddress: resp[x].ToAddress,
			CryptoChain:     resp[x].Chain,
		}
		if resp[x].Destination == "on_chain" {
			funHist[x].CryptoTxID = strconv.FormatInt(resp[x].TradeID, 10)
		}
	}
	return funHist, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (bi *Bitget) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := bi.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		resp, err := bi.GetRecentSpotFills(ctx, p, 500)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price,
				Amount:       resp[x].Size,
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	case asset.Futures:
		resp, err := bi.GetRecentFuturesFills(ctx, p, getProductType(p), 100)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price,
				Amount:       resp[x].Size,
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	}
	return nil, asset.ErrNotSupported
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (bi *Bitget) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	// This exchange only allows requests covering the last 7 days
	p, err := bi.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		resp, err := bi.GetSpotMarketTrades(ctx, p, timestampStart, timestampEnd, 1000, 0)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price,
				Amount:       resp[x].Size,
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	case asset.Futures:
		resp, err := bi.GetFuturesMarketTrades(ctx, p, getProductType(p), 1000, 0, timestampStart, timestampEnd)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp))
		for x := range resp {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp[x].Side),
				Price:        resp[x].Price,
				Amount:       resp[x].Size,
				Timestamp:    resp[x].Timestamp.Time(),
			}
		}
		return trades, nil
	}
	return nil, asset.ErrNotSupported
}

// GetServerTime returns the current exchange server time.
func (bi *Bitget) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	resp, err := bi.GetTime(ctx)
	return resp.ServerTime.Time(), err
}

// SubmitOrder submits a new order
func (bi *Bitget) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(bi.GetTradingRequirements())
	if err != nil {
		return nil, err
	}
	var IDs *OrderIDStruct
	strategy, err := strategyTruthTable(s.ImmediateOrCancel, s.FillOrKill, s.PostOnly)
	if err != nil {
		return nil, err
	}
	cID, err := uuid.DefaultGenerator.NewV4()
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		IDs, err = bi.PlaceSpotOrder(ctx, s.Pair, s.Side.String(), s.Type.Lower(), strategy, cID.String(), "", s.Price, s.Amount, s.TriggerPrice, 0, 0, 0, 0, false, 0)
	case asset.Futures:
		IDs, err = bi.PlaceFuturesOrder(ctx, s.Pair, getProductType(s.Pair), marginStringer(s.MarginType), sideEncoder(s.Side, false), "", s.Type.Lower(), strategy, cID.String(), "", s.Pair.Quote, 0, 0, s.Amount, s.Price, s.ReduceOnly, false)
	case asset.Margin, asset.CrossMargin:
		loanType := "normal"
		if s.AutoBorrow {
			loanType = "autoLoan"
		}
		if s.AssetType == asset.Margin {
			IDs, err = bi.PlaceIsolatedOrder(ctx, s.Pair, s.Type.Lower(), loanType, strategy, cID.String(), s.Side.String(), "", s.Price, s.Amount, s.QuoteAmount)
		} else {
			IDs, err = bi.PlaceCrossOrder(ctx, s.Pair, s.Type.Lower(), loanType, strategy, cID.String(), s.Side.String(), "", s.Price, s.Amount, s.QuoteAmount)
		}
	default:
		return nil, asset.ErrNotSupported
	}
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(strconv.FormatInt(int64(IDs.OrderID), 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = IDs.ClientOrderID
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (bi *Bitget) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	err := action.Validate()
	if err != nil {
		return nil, err
	}
	var IDs *OrderIDStruct
	originalID, err := strconv.ParseInt(action.OrderID, 10, 64)
	if err != nil {
		return nil, err
	}
	switch action.AssetType {
	case asset.Spot:
		IDs, err = bi.ModifyPlanSpotOrder(ctx, originalID, action.ClientOrderID, action.Type.String(), action.TriggerPrice, action.Price, action.Amount)
	case asset.Futures:
		var cID uuid.UUID
		cID, err = uuid.DefaultGenerator.NewV4()
		if err != nil {
			return nil, err
		}
		IDs, err = bi.ModifyFuturesOrder(ctx, originalID, action.ClientOrderID, getProductType(action.Pair), cID.String(), action.Pair, action.Amount, action.Price, 0, 0)
	default:
		return nil, asset.ErrNotSupported
	}
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = strconv.FormatInt(int64(IDs.OrderID), 10)
	resp.ClientOrderID = IDs.ClientOrderID
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (bi *Bitget) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	err := ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	originalID, err := strconv.ParseInt(ord.OrderID, 10, 64)
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot:
		_, err = bi.CancelSpotOrderByID(ctx, ord.Pair, ord.ClientOrderID, "", originalID)
	case asset.Futures:
		_, err = bi.CancelFuturesOrder(ctx, ord.Pair, getProductType(ord.Pair), ord.ClientOrderID, ord.Pair.Quote, originalID)
	case asset.Margin:
		_, err = bi.CancelIsolatedOrder(ctx, ord.Pair, ord.ClientOrderID, originalID)
	case asset.CrossMargin:
		_, err = bi.CancelCrossOrder(ctx, ord.Pair, ord.ClientOrderID, originalID)
	default:
		return asset.ErrNotSupported
	}
	if err != nil {
		return err
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (bi *Bitget) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	batchByAsset := make(map[asset.Item][]order.Cancel)
	for i := range orders {
		batchByAsset[orders[i].AssetType] = append(batchByAsset[orders[i].AssetType], orders[i])
	}
	resp := &order.CancelBatchResponse{}
	resp.Status = make(map[string]string)
	for assetType, batch := range batchByAsset {
		var status *BatchOrderResp
		batchByPair, err := pairBatcher(batch)
		if err != nil {
			return nil, err
		}
		for pair, batch := range batchByPair {
			switch assetType {
			case asset.Spot:
				// This no longer needs to be batched by pair, refactor if many others get similar changes
				batchConv := make([]CancelSpotOrderStruct, len(batch))
				for i := range batch {
					batchConv[i] = CancelSpotOrderStruct{
						OrderID:       int64(batch[i].OrderID),
						ClientOrderID: batch[i].ClientOrderID,
					}
				}
				status, err = bi.BatchCancelOrders(ctx, pair, false, batchConv)
			case asset.Futures:
				status, err = bi.BatchCancelFuturesOrders(ctx, batch, pair, getProductType(pair), pair.Quote)
			case asset.Margin:
				status, err = bi.BatchCancelIsolatedOrders(ctx, pair, batch)
			case asset.CrossMargin:
				status, err = bi.BatchCancelCrossOrders(ctx, pair, batch)
			default:
				return nil, asset.ErrNotSupported
			}
			if err != nil {
				return nil, err
			}
			addStatuses(status, resp)
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (bi *Bitget) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	err := orderCancellation.Validate()
	if err != nil {
		return resp, err
	}
	switch orderCancellation.AssetType {
	case asset.Spot:
		_, err = bi.CancelOrdersBySymbol(ctx, orderCancellation.Pair)
		if err != nil {
			return resp, err
		}
	case asset.Futures:
		resp2, err := bi.CancelAllFuturesOrders(ctx, orderCancellation.Pair, getProductType(orderCancellation.Pair), orderCancellation.Pair.Quote, time.Second*60)
		if err != nil {
			return resp, err
		}
		resp.Status = make(map[string]string)
		for i := range resp2.SuccessList {
			resp.Status[resp2.SuccessList[i].ClientOrderID] = "success"
			resp.Status[strconv.FormatInt(int64(resp2.SuccessList[i].OrderID), 10)] = "success"
		}
		for i := range resp2.FailureList {
			resp.Status[resp2.FailureList[i].ClientOrderID] = resp2.FailureList[i].ErrorMessage
			resp.Status[strconv.FormatInt(int64(resp2.FailureList[i].OrderID), 10)] = resp2.FailureList[i].ErrorMessage
		}
	default:
		return resp, asset.ErrNotSupported
	}
	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (bi *Bitget) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	ordID, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	resp := &order.Detail{
		Exchange:  bi.Name,
		Pair:      pair,
		AssetType: assetType,
		OrderID:   orderID,
	}
	switch assetType {
	case asset.Spot:
		ordInfo, err := bi.GetSpotOrderDetails(ctx, ordID, "", time.Minute)
		if err != nil {
			return nil, err
		}
		if len(ordInfo) == 0 {
			return nil, errOrderNotFound
		}
		resp.AccountID = strconv.FormatUint(ordInfo[0].UserID, 10)
		resp.ClientOrderID = ordInfo[0].ClientOrderID
		resp.Price = ordInfo[0].Price
		resp.Amount = ordInfo[0].Size
		resp.Type = typeDecoder(ordInfo[0].OrderType)
		resp.Side = sideDecoder(ordInfo[0].Side)
		resp.Status = statusDecoder(ordInfo[0].Status)
		resp.AverageExecutedPrice = ordInfo[0].PriceAverage
		resp.QuoteAmount = ordInfo[0].QuoteVolume
		resp.Date = ordInfo[0].CreationTime.Time()
		resp.LastUpdated = ordInfo[0].UpdateTime.Time()
		for s, f := range ordInfo[0].FeeDetail {
			if s != "newFees" {
				resp.FeeAsset = f.FeeCoinCode
				resp.Fee = f.TotalFee
				break
			}
		}
		fillInfo, err := bi.GetSpotFills(ctx, pair, time.Time{}, time.Time{}, 0, 0, ordID)
		if err != nil {
			return nil, err
		}
		resp.Trades = make([]order.TradeHistory, len(fillInfo))
		for x := range fillInfo {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatInt(fillInfo[x].TradeID, 10),
				Type:      typeDecoder(fillInfo[x].OrderType),
				Side:      sideDecoder(fillInfo[x].Side),
				Price:     fillInfo[x].PriceAverage,
				Amount:    fillInfo[x].Size,
				Fee:       fillInfo[x].FeeDetail.TotalFee,
				FeeAsset:  fillInfo[x].FeeDetail.FeeCoin.String(),
				Timestamp: fillInfo[x].CreationTime.Time(),
			}
		}
	case asset.Futures:
		ordInfo, err := bi.GetFuturesOrderDetails(ctx, pair, getProductType(pair), "", ordID)
		if err != nil {
			return nil, err
		}
		resp.Amount = ordInfo.Size
		resp.ClientOrderID = ordInfo.ClientOrderID
		resp.AverageExecutedPrice = ordInfo.PriceAverage.Float64()
		resp.Fee = ordInfo.Fee.Float64()
		resp.Price = ordInfo.Price
		resp.Status = statusDecoder(ordInfo.State)
		resp.Side = sideDecoder(ordInfo.Side)
		resp.ImmediateOrCancel, resp.FillOrKill, resp.PostOnly = strategyDecoder(ordInfo.Force)
		resp.SettlementCurrency = ordInfo.MarginCoin
		resp.LimitPriceUpper = ordInfo.PresetStopSurplusPrice
		resp.LimitPriceLower = ordInfo.PresetStopLossPrice
		resp.QuoteAmount = ordInfo.QuoteVolume
		resp.Type = typeDecoder(ordInfo.OrderType)
		resp.Leverage = ordInfo.Leverage
		resp.MarginType = marginDecoder(ordInfo.MarginMode)
		resp.ReduceOnly = bool(ordInfo.ReduceOnly)
		resp.Date = ordInfo.CreationTime.Time()
		resp.LastUpdated = ordInfo.UpdateTime.Time()
		fillInfo, err := bi.GetFuturesFills(ctx, ordID, 0, 100, pair, getProductType(pair), time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp.Trades = make([]order.TradeHistory, len(fillInfo.FillList))
		for x := range fillInfo.FillList {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatInt(fillInfo.FillList[x].TradeID, 10),
				Price:     fillInfo.FillList[x].Price,
				Amount:    fillInfo.FillList[x].BaseVolume,
				Side:      sideDecoder(fillInfo.FillList[x].Side),
				Timestamp: fillInfo.FillList[x].CreationTime.Time(),
			}
			for i := range fillInfo.FillList[x].FeeDetail {
				resp.Trades[x].Fee += fillInfo.FillList[x].FeeDetail[i].TotalFee
				resp.Trades[x].FeeAsset = fillInfo.FillList[x].FeeDetail[i].FeeCoin.String()
			}
			if fillInfo.FillList[x].TradeScope == "maker" {
				resp.Trades[x].IsMaker = true
			}
		}
	case asset.Margin, asset.CrossMargin:
		var ordInfo *MarginOrders
		var fillInfo *MarginOrderFills
		if assetType == asset.Margin {
			ordInfo, err = bi.GetIsolatedOpenOrders(ctx, pair, "", ordID, 2, 0, time.Now().Add(-time.Hour*24*90), time.Now())
			if err != nil {
				return nil, err
			}
			fillInfo, err = bi.GetIsolatedOrderFills(ctx, pair, ordID, 0, 500, time.Now().Add(-time.Hour*24*90), time.Now())
		} else {
			ordInfo, err = bi.GetCrossOpenOrders(ctx, pair, "", ordID, 2, 0, time.Now().Add(-time.Hour*24*90), time.Now())
			if err != nil {
				return nil, err
			}
			fillInfo, err = bi.GetCrossOrderFills(ctx, pair, ordID, 0, 500, time.Now().Add(-time.Hour*24*90), time.Now())
		}
		if err != nil {
			return nil, err
		}
		if len(ordInfo.OrderList) == 0 {
			return nil, errOrderNotFound
		}
		resp.Type = typeDecoder(ordInfo.OrderList[0].OrderType)
		resp.ClientOrderID = ordInfo.OrderList[0].ClientOrderID
		resp.Price = ordInfo.OrderList[0].Price
		resp.Side = sideDecoder(ordInfo.OrderList[0].Side)
		resp.Status = statusDecoder(ordInfo.OrderList[0].Status)
		resp.Amount = ordInfo.OrderList[0].Size
		resp.QuoteAmount = ordInfo.OrderList[0].QuoteSize
		resp.ImmediateOrCancel, resp.FillOrKill, resp.PostOnly = strategyDecoder(ordInfo.OrderList[0].Force)
		resp.Date = ordInfo.OrderList[0].CreationTime.Time()
		resp.LastUpdated = ordInfo.OrderList[0].UpdateTime.Time()
		resp.Trades = make([]order.TradeHistory, len(fillInfo.Fills))
		for x := range fillInfo.Fills {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatInt(fillInfo.Fills[x].TradeID, 10),
				Type:      typeDecoder(fillInfo.Fills[x].OrderType),
				Side:      sideDecoder(fillInfo.Fills[x].Side),
				Price:     fillInfo.Fills[x].PriceAverage,
				Amount:    fillInfo.Fills[x].Size,
				Timestamp: fillInfo.Fills[x].CreationTime.Time(),
				Fee:       fillInfo.Fills[x].FeeDetail.TotalFee,
				FeeAsset:  fillInfo.Fills[x].FeeDetail.FeeCoin.String(),
			}
		}
	default:
		return nil, asset.ErrNotSupported
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (bi *Bitget) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	resp, err := bi.GetDepositAddressForCurrency(ctx, c, chain, 0)
	if err != nil {
		return nil, err
	}
	add := &deposit.Address{
		Address: resp.Address,
		Chain:   resp.Chain,
		Tag:     resp.Tag,
	}
	return add, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (bi *Bitget) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	err := withdrawRequest.Validate()
	if err != nil {
		return nil, err
	}
	resp, err := bi.WithdrawFunds(ctx, withdrawRequest.Currency, "on_chain", withdrawRequest.Crypto.Address, withdrawRequest.Crypto.Chain, "", "", withdrawRequest.Crypto.AddressTag, withdrawRequest.Description, "", "", "", "", "", "", withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	ret := &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(int64(resp.OrderID), 10),
	}
	return ret, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (bi *Bitget) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (bi *Bitget) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (bi *Bitget) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	for x := range getOrdersRequest.Pairs {
		getOrdersRequest.Pairs[x], err = bi.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
		if err != nil {
			return nil, err
		}
	}
	if len(getOrdersRequest.Pairs) == 0 {
		getOrdersRequest.Pairs = append(getOrdersRequest.Pairs, currency.Pair{})
	}
	var resp order.FilteredOrders
	for x := range getOrdersRequest.Pairs {
		switch getOrdersRequest.AssetType {
		case asset.Spot:
			var pagination int64
			for {
				genOrds, err := bi.GetUnfilledOrders(ctx, getOrdersRequest.Pairs[x], "", time.Time{}, time.Time{}, 100, pagination, 0, time.Minute)
				if err != nil {
					return nil, err
				}
				if len(genOrds) == 0 ||
					pagination == int64(genOrds[len(genOrds)-1].OrderID) {
					break
				}
				pagination = int64(genOrds[len(genOrds)-1].OrderID)
				tempOrds := make([]order.Detail, len(genOrds))
				for i := range genOrds {
					tempOrds[i] = order.Detail{
						Exchange:             bi.Name,
						AssetType:            asset.Spot,
						AccountID:            strconv.FormatUint(genOrds[i].UserID, 10),
						OrderID:              strconv.FormatInt(int64(genOrds[i].OrderID), 10),
						ClientOrderID:        genOrds[i].ClientOrderID,
						AverageExecutedPrice: genOrds[i].PriceAverage,
						Amount:               genOrds[i].Size,
						Type:                 typeDecoder(genOrds[i].OrderType),
						Side:                 sideDecoder(genOrds[i].Side),
						Status:               statusDecoder(genOrds[i].Status),
						Price:                genOrds[i].BasePrice,
						QuoteAmount:          genOrds[i].QuoteVolume,
						Date:                 genOrds[i].CreationTime.Time(),
						LastUpdated:          genOrds[i].UpdateTime.Time(),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
				}
				resp = append(resp, tempOrds...)
			}
			if !getOrdersRequest.Pairs[x].IsEmpty() {
				resp, err = bi.spotCurrentPlanOrdersHelper(ctx, getOrdersRequest.Pairs[x], resp)
				if err != nil {
					return nil, err
				}
			} else {
				newPairs, err := bi.FetchTradablePairs(ctx, asset.Spot)
				if err != nil {
					return nil, err
				}
				for y := range newPairs {
					callStr, err := bi.FormatExchangeCurrency(newPairs[y], asset.Spot)
					if err != nil {
						return nil, err
					}
					resp, err = bi.spotCurrentPlanOrdersHelper(ctx, callStr, resp)
					if err != nil {
						return nil, err
					}
				}
			}
		case asset.Futures:
			if !getOrdersRequest.Pairs[x].IsEmpty() {
				resp, err = bi.activeFuturesOrderHelper(ctx, getProductType(getOrdersRequest.Pairs[x]), getOrdersRequest.Pairs[x], resp)
				if err != nil {
					return nil, err
				}
			} else {
				for y := range prodTypes {
					resp, err = bi.activeFuturesOrderHelper(ctx, prodTypes[y], currency.Pair{}, resp)
					if err != nil {
						return nil, err
					}
				}
			}
		case asset.Margin, asset.CrossMargin:
			var pagination int64
			var genOrds *MarginOrders
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genOrds, err = bi.GetIsolatedOpenOrders(ctx, getOrdersRequest.Pairs[x], "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				} else {
					genOrds, err = bi.GetCrossOpenOrders(ctx, getOrdersRequest.Pairs[x], "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				}
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.OrderList) == 0 || pagination == int64(genOrds.MaximumID) {
					break
				}
				pagination = int64(genOrds.MaximumID)
				tempOrds := make([]order.Detail, len(genOrds.OrderList))
				for i := range genOrds.OrderList {
					tempOrds[i] = order.Detail{
						Exchange:      bi.Name,
						AssetType:     getOrdersRequest.AssetType,
						OrderID:       strconv.FormatInt(genOrds.OrderList[i].OrderID, 10),
						Type:          typeDecoder(genOrds.OrderList[i].OrderType),
						ClientOrderID: genOrds.OrderList[i].ClientOrderID,
						Price:         genOrds.OrderList[i].Price,
						Side:          sideDecoder(genOrds.OrderList[i].Side),
						Status:        statusDecoder(genOrds.OrderList[i].Status),
						QuoteAmount:   genOrds.OrderList[i].QuoteSize,
						Amount:        genOrds.OrderList[i].Size,
						Date:          genOrds.OrderList[i].CreationTime.Time(),
						LastUpdated:   genOrds.OrderList[i].UpdateTime.Time(),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds.OrderList[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
					tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly = strategyDecoder(genOrds.OrderList[i].Force)
				}
				resp = append(resp, tempOrds...)
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information. Can Limit response to specific order status
func (bi *Bitget) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	for x := range getOrdersRequest.Pairs {
		getOrdersRequest.Pairs[x], err = bi.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
	}
	if len(getOrdersRequest.Pairs) == 0 {
		getOrdersRequest.Pairs = append(getOrdersRequest.Pairs, currency.Pair{})
	}
	var resp order.FilteredOrders
	for x := range getOrdersRequest.Pairs {
		switch getOrdersRequest.AssetType {
		case asset.Spot:
			fillMap := make(map[int64][]order.TradeHistory)
			var pagination int64
			if !getOrdersRequest.Pairs[x].IsEmpty() {
				err = bi.spotFillsHelper(ctx, getOrdersRequest.Pairs[x], fillMap)
				if err != nil {
					return nil, err
				}
				resp, err = bi.spotHistoricPlanOrdersHelper(ctx, getOrdersRequest.Pairs[x], resp, fillMap)
				if err != nil {
					return nil, err
				}
			} else {
				newPairs, err := bi.FetchTradablePairs(ctx, asset.Spot)
				if err != nil {
					return nil, err
				}
				for y := range newPairs {
					callStr, err := bi.FormatExchangeCurrency(newPairs[y], asset.Spot)
					if err != nil {
						return nil, err
					}
					err = bi.spotFillsHelper(ctx, callStr, fillMap)
					if err != nil {
						return nil, err
					}
					resp, err = bi.spotHistoricPlanOrdersHelper(ctx, callStr, resp, fillMap)
					if err != nil {
						return nil, err
					}
				}
			}
			for {
				genOrds, err := bi.GetHistoricalSpotOrders(ctx, getOrdersRequest.Pairs[x], time.Time{}, time.Time{}, 100, pagination, 0, "", time.Minute)
				if err != nil {
					return nil, err
				}
				if len(genOrds) == 0 || pagination == int64(genOrds[len(genOrds)-1].OrderID) {
					break
				}
				pagination = int64(genOrds[len(genOrds)-1].OrderID)
				tempOrds := make([]order.Detail, len(genOrds))
				for i := range genOrds {
					tempOrds[i] = order.Detail{
						Exchange:             bi.Name,
						AssetType:            asset.Spot,
						AccountID:            strconv.FormatUint(genOrds[i].UserID, 10),
						OrderID:              strconv.FormatInt(int64(genOrds[i].OrderID), 10),
						ClientOrderID:        genOrds[i].ClientOrderID,
						Price:                genOrds[i].Price,
						Amount:               genOrds[i].Size,
						Type:                 typeDecoder(genOrds[i].OrderType),
						Side:                 sideDecoder(genOrds[i].Side),
						Status:               statusDecoder(genOrds[i].Status),
						AverageExecutedPrice: genOrds[i].PriceAverage,
						QuoteAmount:          genOrds[i].QuoteVolume,
						Date:                 genOrds[i].CreationTime.Time(),
						LastUpdated:          genOrds[i].UpdateTime.Time(),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
					for y := range genOrds[i].FeeDetail {
						tempOrds[i].Fee += genOrds[i].FeeDetail[y].TotalFee
						tempOrds[i].FeeAsset = genOrds[i].FeeDetail[y].FeeCoinCode
					}
					if len(fillMap[int64(genOrds[i].OrderID)]) > 0 {
						tempOrds[i].Trades = fillMap[int64(genOrds[i].OrderID)]
					}
				}
				resp = append(resp, tempOrds...)
			}
		case asset.Futures:
			if !getOrdersRequest.Pairs[x].IsEmpty() {
				resp, err = bi.historicalFuturesOrderHelper(ctx, getProductType(getOrdersRequest.Pairs[x]), getOrdersRequest.Pairs[x], resp)
				if err != nil {
					return nil, err
				}
			} else {
				for y := range prodTypes {
					resp, err = bi.historicalFuturesOrderHelper(ctx, prodTypes[y], currency.Pair{}, resp)
					if err != nil {
						return nil, err
					}
				}
			}
		case asset.Margin, asset.CrossMargin:
			var pagination int64
			var genFills *MarginOrderFills
			fillMap := make(map[int64][]order.TradeHistory)
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genFills, err = bi.GetIsolatedOrderFills(ctx, getOrdersRequest.Pairs[x], 0, pagination, 500, time.Now().Add(-time.Hour*24*90), time.Now())
				} else {
					genFills, err = bi.GetCrossOrderFills(ctx, getOrdersRequest.Pairs[x], 0, pagination, 500, time.Now().Add(-time.Hour*24*90), time.Now())
				}
				if err != nil {
					return nil, err
				}
				if genFills == nil || len(genFills.Fills) == 0 || pagination == int64(genFills.MaximumID) {
					break
				}
				pagination = int64(genFills.MaximumID)
				for i := range genFills.Fills {
					fillMap[genFills.Fills[i].TradeID] = append(fillMap[genFills.Fills[i].TradeID], order.TradeHistory{
						TID:       strconv.FormatInt(genFills.Fills[i].TradeID, 10),
						Type:      typeDecoder(genFills.Fills[i].OrderType),
						Side:      sideDecoder(genFills.Fills[i].Side),
						Price:     genFills.Fills[i].PriceAverage,
						Amount:    genFills.Fills[i].Size,
						Timestamp: genFills.Fills[i].CreationTime.Time(),
						Fee:       genFills.Fills[i].FeeDetail.TotalFee,
						FeeAsset:  genFills.Fills[i].FeeDetail.FeeCoin.String(),
					})
				}
			}
			pagination = 0
			var genOrds *MarginOrders
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genOrds, err = bi.GetIsolatedHistoricalOrders(ctx, getOrdersRequest.Pairs[x], "", "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				} else {
					genOrds, err = bi.GetCrossHistoricalOrders(ctx, getOrdersRequest.Pairs[x], "", "", 0, 500, pagination, time.Now().Add(-time.Hour*24*90), time.Time{})
				}
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.OrderList) == 0 || pagination == int64(genOrds.MaximumID) {
					break
				}
				pagination = int64(genOrds.MaximumID)
				tempOrds := make([]order.Detail, len(genOrds.OrderList))
				for i := range genOrds.OrderList {
					tempOrds[i] = order.Detail{
						Exchange:             bi.Name,
						AssetType:            getOrdersRequest.AssetType,
						OrderID:              strconv.FormatInt(genOrds.OrderList[i].OrderID, 10),
						Type:                 typeDecoder(genOrds.OrderList[i].OrderType),
						ClientOrderID:        genOrds.OrderList[i].ClientOrderID,
						Price:                genOrds.OrderList[i].Price,
						Side:                 sideDecoder(genOrds.OrderList[i].Side),
						Status:               statusDecoder(genOrds.OrderList[i].Status),
						Amount:               genOrds.OrderList[i].Size,
						QuoteAmount:          genOrds.OrderList[i].QuoteSize,
						AverageExecutedPrice: genOrds.OrderList[i].PriceAverage,
						Date:                 genOrds.OrderList[i].CreationTime.Time(),
						LastUpdated:          genOrds.OrderList[i].UpdateTime.Time(),
					}
					if !getOrdersRequest.Pairs[x].IsEmpty() {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds.OrderList[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
					tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly = strategyDecoder(genOrds.OrderList[i].Force)
					if len(fillMap[genOrds.OrderList[i].OrderID]) > 0 {
						tempOrds[i].Trades = fillMap[genOrds.OrderList[i].OrderID]
					}
				}
				resp = append(resp, tempOrds...)
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (bi *Bitget) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	fee, err := bi.GetTradeRate(ctx, feeBuilder.Pair, "spot")
	if err != nil {
		return 0, err
	}
	if feeBuilder.IsMaker {
		return fee.MakerFeeRate * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	}
	return fee.TakerFeeRate * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
}

// ValidateAPICredentials validates current credentials used for wrapper
func (bi *Bitget) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := bi.UpdateAccountInfo(ctx, assetType)
	return bi.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (bi *Bitget) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := bi.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	var resp []kline.Candle
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		cndl, err := bi.GetSpotCandlestickData(ctx, req.RequestFormatted, formatExchangeKlineIntervalSpot(req.ExchangeInterval), req.Start, req.End, 200, true)
		if err != nil {
			return nil, err
		}
		resp = make([]kline.Candle, len(cndl.SpotCandles))
		for i := range cndl.SpotCandles {
			resp[i] = kline.Candle{
				Time:   cndl.SpotCandles[i].Timestamp,
				Low:    cndl.SpotCandles[i].Low,
				High:   cndl.SpotCandles[i].High,
				Open:   cndl.SpotCandles[i].Open,
				Close:  cndl.SpotCandles[i].Close,
				Volume: cndl.SpotCandles[i].BaseVolume,
			}
		}
	case asset.Futures:
		cndl, err := bi.GetFuturesCandlestickData(ctx, req.RequestFormatted, getProductType(pair), formatExchangeKlineIntervalFutures(req.ExchangeInterval), "", req.Start, req.End, 200, CallModeHistory)
		if err != nil {
			return nil, err
		}
		resp = make([]kline.Candle, len(cndl.FuturesCandles))
		for i := range cndl.FuturesCandles {
			resp[i] = kline.Candle{
				Time:   cndl.FuturesCandles[i].Timestamp,
				Low:    cndl.FuturesCandles[i].Low,
				High:   cndl.FuturesCandles[i].High,
				Open:   cndl.FuturesCandles[i].Entry,
				Close:  cndl.FuturesCandles[i].Exit,
				Volume: cndl.FuturesCandles[i].BaseVolume,
			}
		}
	default:
		return nil, asset.ErrNotSupported
	}
	return req.ProcessResponse(resp)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (bi *Bitget) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := bi.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var resp []kline.Candle
	for x := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin, asset.CrossMargin:
			cndl, err := bi.GetSpotCandlestickData(ctx, req.RequestFormatted, formatExchangeKlineIntervalSpot(req.ExchangeInterval), req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, 200, true)
			if err != nil {
				return nil, err
			}
			temp := make([]kline.Candle, len(cndl.SpotCandles))
			for i := range cndl.SpotCandles {
				temp[i] = kline.Candle{
					Time:   cndl.SpotCandles[i].Timestamp,
					Low:    cndl.SpotCandles[i].Low,
					High:   cndl.SpotCandles[i].High,
					Open:   cndl.SpotCandles[i].Open,
					Close:  cndl.SpotCandles[i].Close,
					Volume: cndl.SpotCandles[i].BaseVolume,
				}
			}
			resp = append(resp, temp...)
		case asset.Futures:
			cndl, err := bi.GetFuturesCandlestickData(ctx, req.RequestFormatted, getProductType(pair), formatExchangeKlineIntervalFutures(req.ExchangeInterval), "", req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, 200, CallModeHistory)
			if err != nil {
				return nil, err
			}
			temp := make([]kline.Candle, len(cndl.FuturesCandles))
			for i := range cndl.FuturesCandles {
				temp[i] = kline.Candle{
					Time:   cndl.FuturesCandles[i].Timestamp,
					Low:    cndl.FuturesCandles[i].Low,
					High:   cndl.FuturesCandles[i].High,
					Open:   cndl.FuturesCandles[i].Entry,
					Close:  cndl.FuturesCandles[i].Exit,
					Volume: cndl.FuturesCandles[i].BaseVolume,
				}
			}
			resp = append(resp, temp...)
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return req.ProcessResponse(resp)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (bi *Bitget) GetFuturesContractDetails(ctx context.Context, _ asset.Item) ([]futures.Contract, error) {
	var contracts []futures.Contract
	for i := range prodTypes {
		resp, err := bi.GetContractConfig(ctx, currency.Pair{}, prodTypes[i])
		if err != nil {
			return nil, err
		}
		temp := make([]futures.Contract, len(resp))
		for x := range resp {
			temp[x] = futures.Contract{
				Exchange:    bi.Name,
				Name:        currency.NewPair(resp[x].BaseCoin, resp[x].QuoteCoin),
				Multiplier:  resp[x].SizeMultiplier,
				Asset:       itemDecoder(resp[x].SymbolType),
				Type:        contractTypeDecoder(resp[x].SymbolType),
				Status:      resp[x].SymbolStatus,
				StartDate:   resp[x].DeliveryStartTime.Time(),
				EndDate:     resp[x].DeliveryTime.Time(),
				MaxLeverage: resp[x].MaximumLeverage,
			}
			set := make(currency.Currencies, len(resp[x].SupportMarginCoins))
			for y := range resp[x].SupportMarginCoins {
				set[y] = currency.NewCode(resp[x].SupportMarginCoins[y])
			}
			temp[x].SettlementCurrencies = set
			if resp[x].SymbolStatus == "listed" || resp[x].SymbolStatus == "normal" {
				temp[x].IsActive = true
			}
		}
		contracts = append(contracts, temp...)
	}
	return contracts, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (bi *Bitget) GetLatestFundingRates(ctx context.Context, req *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	fPair, err := bi.FormatExchangeCurrency(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	curRate, err := bi.GetFundingCurrent(ctx, fPair, getProductType(fPair))
	if err != nil {
		return nil, err
	}
	nextTime, err := bi.GetNextFundingTime(ctx, fPair, getProductType(fPair))
	if err != nil {
		return nil, err
	}
	resp := []fundingrate.LatestRateResponse{
		{
			Exchange:       bi.Name,
			Pair:           fPair,
			TimeOfNextRate: nextTime[0].NextFundingTime.Time(),
			TimeChecked:    time.Now(),
		},
	}
	dec := decimal.NewFromFloat(curRate[0].FundingRate)
	resp[0].LatestRate.Rate = dec
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (bi *Bitget) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var limits []order.MinMaxLevel
	switch a {
	case asset.Spot:
		resp, err := bi.GetSymbolInfo(ctx, currency.Pair{})
		if err != nil {
			return err
		}
		limits = make([]order.MinMaxLevel, len(resp))
		for i := range resp {
			limits[i] = order.MinMaxLevel{
				Asset:                   a,
				Pair:                    currency.NewPair(resp[i].BaseCoin, resp[i].QuoteCoin),
				PriceStepIncrementSize:  math.Pow10(-int(resp[i].PricePrecision)),
				AmountStepIncrementSize: math.Pow10(-int(resp[i].QuantityPrecision)),
				QuoteStepIncrementSize:  math.Pow10(-int(resp[i].QuotePrecision)),
				MinNotional:             resp[i].MinimumTradeUSDT,
				MarketMinQty:            resp[i].MinimumTradeAmount,
				MarketMaxQty:            resp[i].MaximumTradeAmount,
			}
		}
	case asset.Futures:
		for i := range prodTypes {
			resp, err := bi.GetContractConfig(ctx, currency.Pair{}, prodTypes[i])
			if err != nil {
				return err
			}
			limits = make([]order.MinMaxLevel, len(resp))
			for i := range resp {
				limits[i] = order.MinMaxLevel{
					Asset:          a,
					Pair:           currency.NewPair(resp[i].BaseCoin, resp[i].QuoteCoin),
					MinNotional:    resp[i].MinimumTradeUSDT,
					MaxTotalOrders: resp[i].MaximumSymbolOrderNumber,
				}
			}
		}
	case asset.Margin, asset.CrossMargin:
		resp, err := bi.GetSupportedCurrencies(ctx)
		if err != nil {
			return err
		}
		limits = make([]order.MinMaxLevel, len(resp))
		for i := range resp {
			limits[i] = order.MinMaxLevel{
				Asset:                   a,
				Pair:                    currency.NewPair(resp[i].BaseCoin, resp[i].QuoteCoin),
				MinNotional:             resp[i].MinimumTradeUSDT,
				MarketMinQty:            resp[i].MinimumTradeAmount,
				MarketMaxQty:            resp[i].MaximumTradeAmount,
				QuoteStepIncrementSize:  math.Pow10(-int(resp[i].PricePrecision)),
				AmountStepIncrementSize: math.Pow10(-int(resp[i].QuantityPrecision)),
			}
		}
	default:
		return asset.ErrNotSupported
	}
	return bi.LoadLimits(limits)
}

// UpdateCurrencyStates updates currency states
func (bi *Bitget) UpdateCurrencyStates(ctx context.Context, a asset.Item) error {
	payload := make(map[currency.Code]currencystate.Options)
	resp, err := bi.GetCoinInfo(ctx, currency.Code{})
	if err != nil {
		return err
	}
	for i := range resp {
		var withdraw bool
		var deposit bool
		var trade bool
		for j := range resp[i].Chains {
			if resp[i].Chains[j].Withdrawable {
				withdraw = true
			}
			if resp[i].Chains[j].Rechargeable {
				deposit = true
			}
		}
		if withdraw && deposit {
			trade = true
		}
		payload[resp[i].Coin] = currencystate.Options{
			Withdraw: &withdraw,
			Deposit:  &deposit,
			Trade:    &trade,
		}
	}
	return bi.States.UpdateAll(a, payload)
}

// GetAvailableTransferChains returns a list of supported transfer chains based on the supplied cryptocurrency
func (bi *Bitget) GetAvailableTransferChains(ctx context.Context, cur currency.Code) ([]string, error) {
	if cur.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	resp, err := bi.GetCoinInfo(ctx, cur)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errReturnEmpty
	}
	chains := make([]string, len(resp[0].Chains))
	for i := range resp[0].Chains {
		chains[i] = resp[0].Chains[i].Chain
	}
	return chains, nil
}

// GetMarginRatesHistory returns the margin rate history for the supplied currency
func (bi *Bitget) GetMarginRatesHistory(ctx context.Context, req *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	var pagination int64
	rates := new(margin.RateHistoryResponse)
loop:
	for {
		switch req.Asset {
		case asset.Margin:
			resp, err := bi.GetIsolatedInterestHistory(ctx, req.Pair, req.Currency, req.StartDate, req.EndDate, 500, pagination)
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.ResultList) == 0 || pagination == int64(resp.MaximumID) {
				break loop
			}
			pagination = int64(resp.MaximumID)
			for i := range resp.ResultList {
				rates.Rates = append(rates.Rates, margin.Rate{
					DailyBorrowRate: decimal.NewFromFloat(resp.ResultList[i].DailyInterestRate),
					Time:            resp.ResultList[i].CreationTime.Time(),
				})
			}
		case asset.CrossMargin:
			resp, err := bi.GetCrossInterestHistory(ctx, req.Currency, req.StartDate, req.EndDate, 500, pagination)
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.ResultList) == 0 || pagination == int64(resp.MaximumID) {
				break loop
			}
			pagination = int64(resp.MaximumID)
			for i := range resp.ResultList {
				rates.Rates = append(rates.Rates, margin.Rate{
					DailyBorrowRate: decimal.NewFromFloat(resp.ResultList[i].DailyInterestRate),
					Time:            resp.ResultList[i].CreationTime.Time(),
				})
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return rates, nil
}

// GetFuturesPositionSummary returns stats for a future position
func (bi *Bitget) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	resp, err := bi.GetSinglePosition(ctx, getProductType(req.Pair), req.Pair, req.Pair.Quote)
	if err != nil {
		return nil, err
	}
	if len(resp) != 1 {
		// I'm not sure that it should actually return one data point in this case, replace this with a properly formatted error message once certain (i.e. once you can test GetSinglePosition properly)
		return nil, fmt.Errorf("expected 1 position, received %v", len(resp))
	}
	summary := &futures.PositionSummary{
		Pair:                         req.Pair,
		Asset:                        req.Asset,
		CurrentSize:                  decimal.NewFromFloat(resp[0].OpenDelegateSize),
		InitialMarginRequirement:     decimal.NewFromFloat(resp[0].MarginSize),
		AvailableEquity:              decimal.NewFromFloat(resp[0].Available),
		FrozenBalance:                decimal.NewFromFloat(resp[0].Locked),
		Leverage:                     decimal.NewFromFloat(resp[0].Leverage),
		RealisedPNL:                  decimal.NewFromFloat(resp[0].AchievedProfits),
		AverageOpenPrice:             decimal.NewFromFloat(resp[0].OpenPriceAverage),
		UnrealisedPNL:                decimal.NewFromFloat(resp[0].UnrealizedProfitLoss),
		MaintenanceMarginRequirement: decimal.NewFromFloat(resp[0].KeepMarginRate),
		MarkPrice:                    decimal.NewFromFloat(resp[0].MarkPrice),
		StartDate:                    resp[0].CreationTime.Time(),
	}
	return summary, nil
}

// GetFuturesPositions returns futures positions for all currencies
func (bi *Bitget) GetFuturesPositions(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionDetails, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	var resp []futures.PositionDetails
	// This exchange needs pairs to be passed through, since a MarginCoin has to be provided
	for i := range req.Pairs {
		temp, err := bi.GetAllPositions(ctx, getProductType(req.Pairs[i]), req.Pairs[i].Quote)
		if err != nil {
			return nil, err
		}
		for x := range temp {
			pair, err := pairFromStringHelper(temp[x].Symbol)
			if err != nil {
				return nil, err
			}
			ord := []order.Detail{
				{
					Exchange:             bi.Name,
					AssetType:            req.Asset,
					Pair:                 pair,
					Side:                 sideDecoder(temp[x].HoldSide),
					RemainingAmount:      temp[x].OpenDelegateSize,
					Amount:               temp[x].Total,
					Leverage:             temp[x].Leverage,
					AverageExecutedPrice: temp[x].OpenPriceAverage,
					MarginType:           marginDecoder(temp[x].MarginMode),
					Price:                temp[x].MarkPrice,
					Date:                 temp[x].CreationTime.Time(),
				},
			}
			resp = append(resp, futures.PositionDetails{
				Exchange: bi.Name,
				Pair:     pair,
				Asset:    req.Asset,
				Orders:   ord,
			})
		}
	}
	return resp, nil
}

// GetFuturesPositionOrders returns futures positions orders
func (bi *Bitget) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	pairs := make([]string, len(req.Pairs))
	for x := range req.Pairs {
		pairs[x] = req.Pairs[x].String()
	}
	var resp []futures.PositionResponse
	var err error
	if len(pairs) == 0 {
		for y := range prodTypes {
			resp, err = bi.allFuturesOrderHelper(ctx, prodTypes[y], currency.Pair{}, resp)
			if err != nil {
				return nil, err
			}
		}
	}
	for x := range pairs {
		resp, err = bi.allFuturesOrderHelper(ctx, getProductType(req.Pairs[x]), req.Pairs[x], resp)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// GetHistoricalFundingRates returns historical funding rates for a future
func (bi *Bitget) GetHistoricalFundingRates(ctx context.Context, req *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	resp, err := bi.GetFundingHistorical(ctx, req.Pair, getProductType(req.Pair), 100, 0)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, errReturnEmpty
	}
	rates := make([]fundingrate.Rate, len(resp))
	for i := range resp {
		rates[i] = fundingrate.Rate{
			Time: resp[i].FundingTime.Time(),
			Rate: decimal.NewFromFloat(resp[i].FundingRate),
		}
	}
	rateStruct := &fundingrate.HistoricalRates{
		Exchange:     bi.Name,
		Asset:        req.Asset,
		Pair:         req.Pair,
		StartDate:    rates[0].Time,
		EndDate:      rates[len(rates)-1].Time,
		LatestRate:   rates[0],
		FundingRates: rates,
	}
	if len(rates) > 1 {
		rateStruct.TimeOfNextRate = rates[0].Time.Add(rates[0].Time.Sub(rates[1].Time))
	}
	return rateStruct, nil
}

// SetCollateralMode sets the account's collateral mode for the asset type
func (bi *Bitget) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return common.ErrFunctionNotSupported
}

// GetCollateralMode returns the account's collateral mode for the asset type
func (bi *Bitget) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return 0, common.ErrFunctionNotSupported
}

// SetMarginType sets the account's margin type for the asset type
func (bi *Bitget) SetMarginType(ctx context.Context, a asset.Item, p currency.Pair, t margin.Type) error {
	switch a {
	case asset.Futures:
		var str string
		switch t {
		case margin.Isolated:
			str = "isolated"
		case margin.Multi:
			str = "crossed"
		}
		_, err := bi.ChangeMarginMode(ctx, p, getProductType(p), str, p.Quote)
		if err != nil {
			return err
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// ChangePositionMargin changes the margin type for a position
func (bi *Bitget) ChangePositionMargin(_ context.Context, _ *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (bi *Bitget) SetLeverage(ctx context.Context, a asset.Item, p currency.Pair, _ margin.Type, f float64, s order.Side) error {
	switch a {
	case asset.Futures:
		_, err := bi.ChangeLeverage(ctx, p, getProductType(p), sideEncoder(s, true), p.Quote, f)
		if err != nil {
			return err
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (bi *Bitget) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, t margin.Type, s order.Side) (float64, error) {
	lev := -1.1
	switch a {
	case asset.Futures:
		resp, err := bi.GetOneFuturesAccount(ctx, p, getProductType(p), p.Quote)
		if err != nil {
			return lev, err
		}
		switch t {
		case margin.Isolated:
			switch s {
			case order.Buy, order.Long:
				lev = resp.IsolatedLongLeverage
			case order.Sell, order.Short:
				lev = resp.IsolatedShortLeverage
			default:
				return lev, order.ErrSideIsInvalid
			}
		case margin.Multi:
			lev = resp.CrossedMarginleverage
		default:
			return lev, margin.ErrMarginTypeUnsupported
		}
	case asset.Margin:
		resp, err := bi.GetIsolatedInterestRateAndMaxBorrowable(ctx, p)
		if err != nil {
			return lev, err
		}
		if len(resp) == 0 {
			return lev, errReturnEmpty
		}
		lev = resp[0].Leverage
	case asset.CrossMargin:
		resp, err := bi.GetCrossInterestRateAndMaxBorrowable(ctx, p.Quote)
		if err != nil {
			return lev, err
		}
		if len(resp) == 0 {
			return lev, errReturnEmpty
		}
		lev = resp[0].Leverage
	default:
		return lev, asset.ErrNotSupported
	}
	return lev, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (bi *Bitget) GetOpenInterest(ctx context.Context, pairs ...key.PairAsset) ([]futures.OpenInterest, error) {
	openInterest := make([]futures.OpenInterest, len(pairs))
	for i := range pairs {
		resp, err := bi.GetOpenPositions(ctx, pairs[i].Pair(), getProductType(pairs[i].Pair()))
		if err != nil {
			return nil, err
		}
		if len(resp.OpenInterestList) == 0 {
			return nil, errReturnEmpty
		}
		openInterest[i] = futures.OpenInterest{
			OpenInterest: resp.OpenInterestList[0].Size,
			Key: key.ExchangePairAsset{
				Exchange: bi.Name,
				Base:     pairs[i].Base,
				Quote:    pairs[i].Quote,
				Asset:    pairs[i].Asset,
			},
		}
	}
	return openInterest, nil
}

// GetProductType is a helper function that returns the appropriate product type for a given currency pair
func getProductType(p currency.Pair) string {
	var prodType string
	switch p.Quote {
	case currency.USDT:
		prodType = "USDT-FUTURES"
	case currency.PERP, currency.USDC:
		prodType = "USDC-FUTURES"
	default:
		prodType = "COIN-FUTURES"
	}
	return prodType
}

// SideDecoder is a helper function that returns the appropriate order side for a given string
func sideDecoder(d string) order.Side {
	switch strings.ToLower(d) {
	case "buy", "long":
		return order.Buy
	case "sell", "short":
		return order.Sell
	}
	return order.UnknownSide
}

// StrategyTruthTable is a helper function that returns the appropriate strategy for a given set of booleans
func strategyTruthTable(ioc, fok, po bool) (string, error) {
	if (ioc && fok) || (fok && po) || (ioc && po) {
		return "", errStrategyMutex
	}
	if ioc {
		return "ioc", nil
	}
	if fok {
		return "fok", nil
	}
	if po {
		return "post_only", nil
	}
	return "gtc", nil
}

// ClientIDGenerator is a helper function that generates a unique client ID
func clientIDGenerator() string {
	// Making the bits corresponding to smaller timescales more significant, to cheaply increase uniqueness across small timeframes
	i := time.Now().UnixNano()>>29 + time.Now().UnixNano()<<35
	// ClientID supports alphanumeric characters, so use the largest prime bases that fit within 50 characters to minimize chance of collisions
	cID := strconv.FormatInt(i, 31) + strconv.FormatInt(i, 29) + strconv.FormatInt(i, 23) + strconv.FormatInt(i, 19)
	if len(cID) > 50 {
		cID = cID[:50]
	}
	return cID
}

// MarginStringer is a helper function that returns the appropriate string for a given margin type
func marginStringer(m margin.Type) string {
	switch m {
	case margin.Isolated:
		return "isolated"
	case margin.Multi:
		return "crossed"
	}
	return ""
}

// SideEncoder is a helper function that returns the appropriate string for a given order side
func sideEncoder(s order.Side, longshort bool) string {
	switch s {
	case order.Buy, order.Long:
		if longshort {
			return "long"
		}
		return "buy"
	case order.Sell, order.Short:
		if longshort {
			return "short"
		}
		return "sell"
	}
	return "unknown side"
}

// PairBatcher is a helper function that batches orders by currency pair
func pairBatcher(orders []order.Cancel) (map[currency.Pair][]OrderIDStruct, error) {
	batchByPair := make(map[currency.Pair][]OrderIDStruct)
	for i := range orders {
		originalID, err := strconv.ParseInt(orders[i].OrderID, 10, 64)
		if err != nil {
			return nil, err
		}
		batchByPair[orders[i].Pair] = append(batchByPair[orders[i].Pair], OrderIDStruct{
			ClientOrderID: orders[i].ClientOrderID,
			OrderID:       EmptyInt(originalID),
		})
	}
	return batchByPair, nil
}

// AddStatuses is a helper function that adds statuses to a response
func addStatuses(status *BatchOrderResp, resp *order.CancelBatchResponse) {
	for i := range status.SuccessList {
		resp.Status[status.SuccessList[i].ClientOrderID] = "success"
		resp.Status[strconv.FormatInt(int64(status.SuccessList[i].OrderID), 10)] = "success"
	}
	for i := range status.FailureList {
		resp.Status[status.FailureList[i].ClientOrderID] = status.FailureList[i].ErrorMessage
		resp.Status[strconv.FormatInt(int64(status.FailureList[i].OrderID), 10)] = status.FailureList[i].ErrorMessage
	}
}

// StatusDecoder is a helper function that returns the appropriate status for a given string
func statusDecoder(status string) order.Status {
	switch status {
	case "live":
		return order.Pending
	case "new":
		return order.New
	case "partially_filled", "partially_fill":
		return order.PartiallyFilled
	case "filled", "full_fill":
		return order.Filled
	case "cancelled", "canceled":
		return order.Cancelled
	case "not_trigger":
		return order.PendingTrigger
	}
	return order.UnknownStatus
}

// StrategyDecoder is a helper function that returns the appropriate strategy bools for a given string
func strategyDecoder(s string) (ioc, fok, po bool) {
	switch strings.ToLower(s) {
	case "ioc":
		ioc = true
	case "fok":
		fok = true
	case "post_only":
		po = true
	}
	return
}

// TypeDecoder is a helper function that returns the appropriate order type for a given string
func typeDecoder(s string) order.Type {
	switch s {
	case "limit":
		return order.Limit
	case "market":
		return order.Market
	}
	return order.UnknownType
}

// WithdrawalHistGrabber is a helper function that repeatedly calls GetWithdrawalRecords and returns all data
func (bi *Bitget) withdrawalHistGrabber(ctx context.Context, currency currency.Code) ([]WithdrawRecordsResp, error) {
	var allData []WithdrawRecordsResp
	var pagination int64
	for {
		resp, err := bi.GetWithdrawalRecords(ctx, currency, "", time.Now().Add(-time.Hour*24*90), time.Now(), pagination, 0, 100)
		if err != nil {
			return nil, err
		}
		if len(resp) == 0 || pagination == resp[len(resp)-1].OrderID {
			break
		}
		pagination = resp[len(resp)-1].OrderID
		allData = append(allData, resp...)
	}
	return allData, nil
}

// PairFromStringHelper is a helper function that does some checks to help with common ambiguous cases in this exchange
func pairFromStringHelper(s string) (currency.Pair, error) {
	pair := currency.Pair{}
	i := strings.LastIndex(s, "USD")
	if i == -1 {
		i = strings.Index(s, "PERP")
		if i == -1 {
			return pair, errUnknownPairQuote
		}
	}
	pair, err := currency.NewPairFromString(s[:i] + "-" + s[i:])
	if err != nil {
		return pair, err
	}
	pair = pair.Format(currency.PairFormat{Uppercase: true, Delimiter: ""})
	return pair, nil
}

// SpotPlanOrdersHelper is a helper function that repeatedly calls GetCurrentSpotPlanOrders and returns all data
func (bi *Bitget) spotCurrentPlanOrdersHelper(ctx context.Context, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination int64
	for {
		genOrds, err := bi.GetCurrentSpotPlanOrders(ctx, pairCan, time.Time{}, time.Time{}, 100, pagination)
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.OrderList) == 0 || pagination == int64(genOrds.IDLessThan) {
			break
		}
		pagination = int64(genOrds.IDLessThan)
		tempOrds := make([]order.Detail, len(genOrds.OrderList))
		for i := range genOrds.OrderList {
			tempOrds[i] = order.Detail{
				Exchange:      bi.Name,
				AssetType:     asset.Spot,
				OrderID:       strconv.FormatInt(genOrds.OrderList[i].OrderID, 10),
				ClientOrderID: genOrds.OrderList[i].ClientOrderID,
				TriggerPrice:  genOrds.OrderList[i].TriggerPrice,
				Type:          typeDecoder(genOrds.OrderList[i].OrderType),
				Price:         float64(genOrds.OrderList[i].ExecutePrice),
				Amount:        genOrds.OrderList[i].Size,
				Status:        statusDecoder(genOrds.OrderList[i].Status),
				Side:          sideDecoder(genOrds.OrderList[i].Side),
				Date:          genOrds.OrderList[i].CreationTime.Time(),
				LastUpdated:   genOrds.OrderList[i].UpdateTime.Time(),
			}
			tempOrds[i].Pair = pairCan
		}
		resp = append(resp, tempOrds...)
		if !genOrds.NextFlag {
			break
		}
	}
	return resp, nil
}

// MarginDecoder is a helper function that returns the appropriate margin type for a given string
func marginDecoder(s string) margin.Type {
	switch s {
	case "isolated":
		return margin.Isolated
	case "cross", "crossed":
		return margin.Multi
	}
	return margin.Unknown
}

// ActiveFuturesOrderHelper is a helper function that repeatedly calls GetPendingFuturesOrders and GetPendingFuturesTriggerOrders, returning the data formatted appropriately
func (bi *Bitget) activeFuturesOrderHelper(ctx context.Context, productType string, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination int64
	for {
		genOrds, err := bi.GetPendingFuturesOrders(ctx, 0, pagination, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == int64(genOrds.EndID) {
			break
		}
		pagination = int64(genOrds.EndID)
		tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
		for i := range genOrds.EntrustedList {
			tempOrds[i] = order.Detail{
				Exchange:             bi.Name,
				AssetType:            asset.Futures,
				Amount:               genOrds.EntrustedList[i].Size,
				OrderID:              strconv.FormatInt(genOrds.EntrustedList[i].OrderID, 10),
				ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.EntrustedList[i].Fee),
				Price:                float64(genOrds.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
				SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
				QuoteAmount:          genOrds.EntrustedList[i].QuoteVolume,
				Leverage:             genOrds.EntrustedList[i].Leverage,
				MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
				Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
			}
			if !pairCan.IsEmpty() {
				tempOrds[i].Pair = pairCan
			} else {
				tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly = strategyDecoder(genOrds.EntrustedList[i].Force)
		}
		resp = append(resp, tempOrds...)
	}
	for y := range planTypes {
		pagination = 0
		for {
			genOrds, err := bi.GetPendingTriggerFuturesOrders(ctx, 0, pagination, 100, "", planTypes[y], productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == int64(genOrds.EndID) {
				break
			}
			pagination = int64(genOrds.EndID)
			tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
			for i := range genOrds.EntrustedList {
				tempOrds[i] = order.Detail{
					Exchange:           bi.Name,
					AssetType:          asset.Futures,
					Amount:             genOrds.EntrustedList[i].Size,
					OrderID:            strconv.FormatInt(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:      genOrds.EntrustedList[i].ClientOrderID,
					Price:              float64(genOrds.EntrustedList[i].Price),
					TriggerPrice:       float64(genOrds.EntrustedList[i].TriggerPrice),
					Status:             statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Side:               sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency: genOrds.EntrustedList[i].MarginCoin,
					MarginType:         marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:               typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:               genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:        genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:    float64(genOrds.EntrustedList[i].TakeProfitExecutePrice),
					LimitPriceLower:    float64(genOrds.EntrustedList[i].StopLossExecutePrice),
				}
				if !pairCan.IsEmpty() {
					tempOrds[i].Pair = pairCan
				} else {
					tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
			}
			resp = append(resp, tempOrds...)
		}
	}
	return resp, nil
}

// SpotHistoricPlanOrdersHelper is a helper function that repeatedly calls GetHistoricalSpotOrders and returns all data formatted appropriately
func (bi *Bitget) spotHistoricPlanOrdersHelper(ctx context.Context, pairCan currency.Pair, resp []order.Detail, fillMap map[int64][]order.TradeHistory) ([]order.Detail, error) {
	var pagination int64
	for {
		genOrds, err := bi.GetSpotPlanOrderHistory(ctx, pairCan, time.Now().Add(-time.Hour*24*90), time.Now() /*.Add(-time.Second)*/, 100, pagination)
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.OrderList) == 0 || pagination == int64(genOrds.IDLessThan) {
			break
		}
		pagination = int64(genOrds.IDLessThan)
		tempOrds := make([]order.Detail, len(genOrds.OrderList))
		for i := range genOrds.OrderList {
			tempOrds[i] = order.Detail{
				Exchange:      bi.Name,
				AssetType:     asset.Spot,
				OrderID:       strconv.FormatInt(genOrds.OrderList[i].OrderID, 10),
				ClientOrderID: genOrds.OrderList[i].ClientOrderID,
				TriggerPrice:  genOrds.OrderList[i].TriggerPrice,
				Type:          typeDecoder(genOrds.OrderList[i].OrderType),
				Price:         float64(genOrds.OrderList[i].ExecutePrice),
				Amount:        genOrds.OrderList[i].Size,
				Status:        statusDecoder(genOrds.OrderList[i].Status),
				Side:          sideDecoder(genOrds.OrderList[i].Side),
				Date:          genOrds.OrderList[i].CreationTime.Time(),
				LastUpdated:   genOrds.OrderList[i].UpdateTime.Time(),
			}
			tempOrds[i].Pair = pairCan
			if len(fillMap[genOrds.OrderList[i].OrderID]) > 0 {
				tempOrds[i].Trades = fillMap[genOrds.OrderList[i].OrderID]
			}
		}
		resp = append(resp, tempOrds...)
		if !genOrds.NextFlag {
			break
		}
	}
	return resp, nil
}

// HistoricalFuturesOrderHelper is a helper function that repeatedly calls GetFuturesFills, GetHistoricalFuturesOrders, and GetHistoricalTriggerFuturesOrders, returning the data formatted appropriately
func (bi *Bitget) historicalFuturesOrderHelper(ctx context.Context, productType string, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination int64
	fillMap := make(map[int64][]order.TradeHistory)
	for {
		fillOrds, err := bi.GetFuturesFills(ctx, 0, pagination, 100, pairCan, productType, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if fillOrds == nil || len(fillOrds.FillList) == 0 || pagination == int64(fillOrds.EndID) {
			break
		}
		pagination = int64(fillOrds.EndID)
		for i := range fillOrds.FillList {
			tempFill := order.TradeHistory{
				TID:       strconv.FormatInt(fillOrds.FillList[i].TradeID, 10),
				Price:     fillOrds.FillList[i].Price,
				Amount:    fillOrds.FillList[i].BaseVolume,
				Side:      sideDecoder(fillOrds.FillList[i].Side),
				Timestamp: fillOrds.FillList[i].CreationTime.Time(),
			}
			for y := range fillOrds.FillList[i].FeeDetail {
				tempFill.Fee += fillOrds.FillList[i].FeeDetail[y].TotalFee
				tempFill.FeeAsset = fillOrds.FillList[i].FeeDetail[y].FeeCoin.String()
			}
			fillMap[fillOrds.FillList[i].OrderID] = append(fillMap[fillOrds.FillList[i].OrderID], tempFill)
		}
	}
	pagination = 0
	for {
		genOrds, err := bi.GetHistoricalFuturesOrders(ctx, 0, pagination, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == int64(genOrds.EndID) {
			break
		}
		pagination = int64(genOrds.EndID)
		tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
		for i := range genOrds.EntrustedList {
			tempOrds[i] = order.Detail{
				Exchange:             bi.Name,
				AssetType:            asset.Futures,
				Amount:               genOrds.EntrustedList[i].Size,
				OrderID:              strconv.FormatInt(genOrds.EntrustedList[i].OrderID, 10),
				ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.EntrustedList[i].Fee),
				Price:                float64(genOrds.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
				SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
				QuoteAmount:          genOrds.EntrustedList[i].QuoteVolume,
				Leverage:             genOrds.EntrustedList[i].Leverage,
				MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
				Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
			}
			if !pairCan.IsEmpty() {
				tempOrds[i].Pair = pairCan
			} else {
				tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly = strategyDecoder(genOrds.EntrustedList[i].Force)
			if len(fillMap[genOrds.EntrustedList[i].OrderID]) > 0 {
				tempOrds[i].Trades = fillMap[genOrds.EntrustedList[i].OrderID]
			}
		}
		resp = append(resp, tempOrds...)
	}
	for y := range planTypes {
		pagination = 0
		for {
			genOrds, err := bi.GetHistoricalTriggerFuturesOrders(ctx, 0, pagination, 100, "", planTypes[y], "", productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination == int64(genOrds.EndID) {
				break
			}
			pagination = int64(genOrds.EndID)
			tempOrds := make([]order.Detail, len(genOrds.EntrustedList))
			for i := range genOrds.EntrustedList {
				tempOrds[i] = order.Detail{
					Exchange:             bi.Name,
					AssetType:            asset.Futures,
					Amount:               genOrds.EntrustedList[i].Size,
					OrderID:              strconv.FormatInt(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
					Status:               statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Price:                float64(genOrds.EntrustedList[i].Price),
					AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
					TriggerPrice:         float64(genOrds.EntrustedList[i].TriggerPrice),
					Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
					MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				}
				if !pairCan.IsEmpty() {
					tempOrds[i].Pair = pairCan
				} else {
					tempOrds[i].Pair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				if len(fillMap[genOrds.EntrustedList[i].OrderID]) > 0 {
					tempOrds[i].Trades = fillMap[genOrds.EntrustedList[i].OrderID]
				}
			}
			resp = append(resp, tempOrds...)
		}
	}
	return resp, nil
}

// SpotFillsHelper is a helper function that repeatedly calls GetSpotFills, directly altering the supplied map with that data
func (bi *Bitget) spotFillsHelper(ctx context.Context, pair currency.Pair, fillMap map[int64][]order.TradeHistory) error {
	var pagination int64
	for {
		genFills, err := bi.GetSpotFills(ctx, pair, time.Time{}, time.Time{}, 100, pagination, 0)
		if err != nil {
			return err
		}
		if len(genFills) == 0 || pagination == genFills[len(genFills)-1].TradeID {
			break
		}
		pagination = genFills[len(genFills)-1].TradeID
		for i := range genFills {
			fillMap[genFills[i].TradeID] = append(fillMap[genFills[i].TradeID],
				order.TradeHistory{
					TID:       strconv.FormatInt(genFills[i].TradeID, 10),
					Type:      typeDecoder(genFills[i].OrderType),
					Side:      sideDecoder(genFills[i].Side),
					Price:     genFills[i].PriceAverage,
					Amount:    genFills[i].Size,
					Fee:       genFills[i].FeeDetail.TotalFee,
					FeeAsset:  genFills[i].FeeDetail.FeeCoin.String(),
					Timestamp: genFills[i].CreationTime.Time(),
				})
		}
	}
	return nil
}

// FormatExchangeKlineIntervalSpot is a helper function used to convert kline.Interval to the string format required by the spot API
func formatExchangeKlineIntervalSpot(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1min"
	case kline.FiveMin:
		return "5min"
	case kline.FifteenMin:
		return "15min"
	case kline.ThirtyMin:
		return "30min"
	case kline.OneHour:
		return "1h"
	case kline.FourHour:
		return "4h"
	case kline.SixHour:
		return "6h"
	case kline.TwelveHour:
		return "12h"
	case kline.OneDay:
		return "1day"
	case kline.ThreeDay:
		return "3day"
	case kline.OneWeek:
		return "1week"
	case kline.OneMonth:
		return "1M"
	}
	return errIntervalNotSupported
}

// FormatExchangeKlineIntervalFutures is a helper function used to convert kline.Interval to the string format required by the futures API
func formatExchangeKlineIntervalFutures(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1H"
	case kline.FourHour:
		return "4H"
	case kline.SixHour:
		return "6H"
	case kline.TwelveHour:
		return "12H"
	case kline.OneDay:
		return "1D"
	case kline.ThreeDay:
		return "3D"
	case kline.OneWeek:
		return "1W"
	case kline.OneMonth:
		return "1M"
	}
	return errIntervalNotSupported
}

// ItemDecoder is a helper function that returns the appropriate asset.Item for a given string
func itemDecoder(s string) asset.Item {
	switch s {
	case "spot", "SPOT":
		return asset.Spot
	case "margin", "MARGIN":
		return asset.Margin
	case "futures", "USDT-FUTURES", "COIN-FUTURES", "USDC-FUTURES", "SUSD-FUTURES", "SCOIN-FUTURES", "SUSDC-FUTURES":
		return asset.Futures
	case "perpetual":
		return asset.PerpetualContract
	case "delivery":
		return asset.DeliveryFutures
	}
	return asset.Empty
}

// contractTypeDecoder is a helper function that returns the appropriate contract type for a given string
func contractTypeDecoder(s string) futures.ContractType {
	switch s {
	case "delivery":
		return futures.LongDated
	case "perpetual":
		return futures.Perpetual
	}
	return futures.Unknown
}

// AllFuturesOrderHelper is a helper function that repeatedly calls GetPendingFuturesOrders and GetPendingFuturesTriggerOrders, returning the data formatted appropriately
func (bi *Bitget) allFuturesOrderHelper(ctx context.Context, productType string, pairCan currency.Pair, resp []futures.PositionResponse) ([]futures.PositionResponse, error) {
	var pagination1 int64
	var pagination2 int64
	var breakbool1 bool
	var breakbool2 bool
	tempOrds := make(map[currency.Pair][]order.Detail)
	for {
		var genOrds *FuturesOrdResp
		var err error
		if !breakbool1 {
			genOrds, err = bi.GetPendingFuturesOrders(ctx, 0, pagination1, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination1 == int64(genOrds.EndID) {
				breakbool1 = true
				genOrds = nil
			} else {
				pagination1 = int64(genOrds.EndID)
			}
		}
		if !breakbool2 {
			genOrds2, err := bi.GetHistoricalFuturesOrders(ctx, 0, pagination2, 100, "", productType, "", pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds2 == nil || len(genOrds2.EntrustedList) == 0 || pagination2 == int64(genOrds2.EndID) {
				breakbool2 = true
			} else {
				if genOrds == nil {
					genOrds = new(FuturesOrdResp)
				}
				genOrds.EntrustedList = append(genOrds.EntrustedList, genOrds2.EntrustedList...)
				pagination2 = int64(genOrds2.EndID)
			}
		}
		if breakbool1 && breakbool2 {
			break
		}
		for i := range genOrds.EntrustedList {
			var thisPair currency.Pair
			if !pairCan.IsEmpty() {
				thisPair = pairCan
			} else {
				thisPair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			ioc, fok, po := strategyDecoder(genOrds.EntrustedList[i].Force)
			tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
				Exchange:             bi.Name,
				Pair:                 thisPair,
				AssetType:            asset.Futures,
				Amount:               genOrds.EntrustedList[i].Size,
				OrderID:              strconv.FormatInt(genOrds.EntrustedList[i].OrderID, 10),
				ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.EntrustedList[i].Fee),
				Price:                float64(genOrds.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
				SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
				QuoteAmount:          genOrds.EntrustedList[i].QuoteVolume,
				Leverage:             genOrds.EntrustedList[i].Leverage,
				MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
				Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				ImmediateOrCancel:    ioc,
				FillOrKill:           fok,
				PostOnly:             po,
			})
		}
	}
	for y := range planTypes {
		pagination1 = 0
		for {
			genOrds, err := bi.GetPendingTriggerFuturesOrders(ctx, 0, pagination1, 100, "", planTypes[y], productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination1 == int64(genOrds.EndID) {
				break
			}
			pagination1 = int64(genOrds.EndID)
			for i := range genOrds.EntrustedList {
				var thisPair currency.Pair
				if !pairCan.IsEmpty() {
					thisPair = pairCan
				} else {
					thisPair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
					Exchange:           bi.Name,
					Pair:               thisPair,
					AssetType:          asset.Futures,
					Amount:             genOrds.EntrustedList[i].Size,
					OrderID:            strconv.FormatInt(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:      genOrds.EntrustedList[i].ClientOrderID,
					Price:              float64(genOrds.EntrustedList[i].Price),
					TriggerPrice:       float64(genOrds.EntrustedList[i].TriggerPrice),
					Status:             statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Side:               sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency: genOrds.EntrustedList[i].MarginCoin,
					MarginType:         marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:               typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:               genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:        genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:    float64(genOrds.EntrustedList[i].TakeProfitExecutePrice),
					LimitPriceLower:    float64(genOrds.EntrustedList[i].StopLossExecutePrice),
				})
			}
		}
		pagination1 = 0
		for {
			genOrds, err := bi.GetHistoricalTriggerFuturesOrders(ctx, 0, pagination1, 100, "", planTypes[y], "", productType, pairCan, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.EntrustedList) == 0 || pagination1 == int64(genOrds.EndID) {
				break
			}
			pagination1 = int64(genOrds.EndID)
			for i := range genOrds.EntrustedList {
				var thisPair currency.Pair
				if !pairCan.IsEmpty() {
					thisPair = pairCan
				} else {
					thisPair, err = pairFromStringHelper(genOrds.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
					Exchange:             bi.Name,
					Pair:                 thisPair,
					AssetType:            asset.Futures,
					Amount:               genOrds.EntrustedList[i].Size,
					OrderID:              strconv.FormatInt(genOrds.EntrustedList[i].OrderID, 10),
					ClientOrderID:        genOrds.EntrustedList[i].ClientOrderID,
					Status:               statusDecoder(genOrds.EntrustedList[i].PlanStatus),
					Price:                float64(genOrds.EntrustedList[i].Price),
					AverageExecutedPrice: float64(genOrds.EntrustedList[i].PriceAverage),
					TriggerPrice:         float64(genOrds.EntrustedList[i].TriggerPrice),
					Side:                 sideDecoder(genOrds.EntrustedList[i].Side),
					SettlementCurrency:   genOrds.EntrustedList[i].MarginCoin,
					MarginType:           marginDecoder(genOrds.EntrustedList[i].MarginMode),
					Type:                 typeDecoder(genOrds.EntrustedList[i].OrderType),
					Date:                 genOrds.EntrustedList[i].CreationTime.Time(),
					LastUpdated:          genOrds.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:      float64(genOrds.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:      float64(genOrds.EntrustedList[i].PresetStopLossPrice),
				})
			}
		}
	}
	for x, y := range tempOrds {
		resp = append(resp, futures.PositionResponse{
			Pair:   x,
			Orders: y,
			Asset:  asset.Futures,
		})
	}
	return resp, nil
}

// ItemEncoder encodes an asset.Item into a string
func itemEncoder(a asset.Item, pair currency.Pair) string {
	switch a {
	case asset.Spot:
		return "SPOT"
	case asset.Futures:
		return getProductType(pair)
	case asset.Margin, asset.CrossMargin:
		return "MARGIN"
	}
	return ""
}

// PositionModeDecoder is a helper function that returns the appropriate position mode for a given string
func positionModeDecoder(s string) futures.PositionMode {
	switch s {
	case "one_way_mode":
		return futures.OneWayMode
	case "hedge_mode":
		return futures.HedgeMode
	}
	return futures.UnknownMode
}
