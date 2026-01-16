package bybit

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
)

type assetPairFmt struct {
	asset  asset.Item
	cfgFmt *currency.PairFormat
	reqFmt *currency.PairFormat
}

var (
	underscoreFmt = &currency.PairFormat{Uppercase: true, Delimiter: "_"}
	dashFmt       = &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	plainFmt      = &currency.PairFormat{Uppercase: true}
	assetPairFmts = []assetPairFmt{
		{asset.Spot, underscoreFmt, plainFmt},
		{asset.USDTMarginedFutures, underscoreFmt, plainFmt},
		{asset.CoinMarginedFutures, underscoreFmt, plainFmt},
		{asset.USDCMarginedFutures, dashFmt, plainFmt},
		{asset.Options, dashFmt, dashFmt},
	}
)

// SetDefaults sets the basic defaults for Bybit
func (e *Exchange) SetDefaults() {
	e.Name = "Bybit"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for _, n := range assetPairFmts {
		ps := currency.PairStore{AssetEnabled: true, RequestFormat: n.reqFmt, ConfigFormat: n.cfgFmt}
		if err := e.SetAssetPairStore(n.asset, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, n.asset, err)
		}
	}

	e.Features = exchange.Features{
		CurrencyTranslations: currency.NewTranslations(
			map[currency.Code]currency.Code{
				currency.NewCode("10000000AIDOGE"):  currency.AIDOGE,
				currency.NewCode("1000000BABYDOGE"): currency.BABYDOGE,
				currency.NewCode("1000000MOG"):      currency.NewCode("MOG"),
				currency.NewCode("10000COQ"):        currency.NewCode("COQ"),
				currency.NewCode("10000LADYS"):      currency.NewCode("LADYS"),
				currency.NewCode("10000NFT"):        currency.NFT,
				currency.NewCode("10000SATS"):       currency.NewCode("SATS"),
				currency.NewCode("10000STARL"):      currency.STARL,
				currency.NewCode("10000WEN"):        currency.NewCode("WEN"),
				currency.NewCode("1000APU"):         currency.NewCode("APU"),
				currency.NewCode("1000BEER"):        currency.NewCode("BEER"),
				currency.NewCode("1000BONK"):        currency.BONK,
				currency.NewCode("1000BTT"):         currency.BTT,
				currency.NewCode("1000FLOKI"):       currency.FLOKI,
				currency.NewCode("1000IQ50"):        currency.NewCode("IQ50"),
				currency.NewCode("1000LUNC"):        currency.LUNC,
				currency.NewCode("1000PEPE"):        currency.PEPE,
				currency.NewCode("1000RATS"):        currency.NewCode("RATS"),
				currency.NewCode("1000TURBO"):       currency.NewCode("TURBO"),
				currency.NewCode("1000XEC"):         currency.XEC,
				currency.NewCode("LUNA2"):           currency.LUNA,
				currency.NewCode("SHIB1000"):        currency.SHIB,
			},
		),
		TradingRequirements: protocol.TradingRequirements{
			SpotMarketBuyQuotation: true,
		},
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:        true,
				TradeFetching:         true,
				KlineFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				FiatDepositFee:        true,
				FiatWithdrawalFee:     true,
				CryptoDepositFee:      true,
				ModifyOrder:           true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				TickerFetching:         true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrders:              true,
				Subscribe:              true,
				Unsubscribe:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				FundingRateBatching: map[asset.Item]bool{
					asset.USDCMarginedFutures: true,
					asset.USDTMarginedFutures: true,
					asset.CoinMarginedFutures: true,
				},
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.FourHour:  true,
					kline.EightHour: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportedViaTicker: true,
					SupportsRestBatch:  true,
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
					kline.IntervalCapacity{Interval: kline.SevenHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 1000,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	e.API.Endpoints = e.NewEndpoints()
	err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              bybitAPIURL,
		exchange.RestCoinMargined:      bybitAPIURL,
		exchange.RestUSDTMargined:      bybitAPIURL,
		exchange.RestFutures:           bybitAPIURL,
		exchange.RestUSDCMargined:      bybitAPIURL,
		exchange.WebsocketSpot:         spotPublic,
		exchange.WebsocketCoinMargined: inversePublic,
		exchange.WebsocketUSDTMargined: linearPublic,
		exchange.WebsocketUSDCMargined: linearPublic,
		exchange.WebsocketOptions:      optionPublic,
		exchange.WebsocketTrade:        websocketTrade,
		exchange.WebsocketPrivate:      websocketPrivate,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	if e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(rateLimits)); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
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
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig:        buffer.Config{SortBuffer: true, SortBufferByUpdateIDs: true},
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement: true,
		RateLimitDefinitions:         rateLimits,
	}); err != nil {
		return err
	}

	wsSpotURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	// Spot - Inbound public data.
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsSpotURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WsConnect,
		GenerateSubscriptions: e.generateSubscriptions,
		Subscriber:            e.SpotSubscribe,
		Unsubscriber:          e.SpotUnsubscribe,
		Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
			return e.wsHandleData(ctx, conn, asset.Spot, resp)
		},
	}); err != nil {
		return err
	}

	wsOptionsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketOptions)
	if err != nil {
		return err
	}

	// Options - Inbound public data.
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsOptionsURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WsConnect,
		GenerateSubscriptions: e.GenerateOptionsDefaultSubscriptions,
		Subscriber:            e.OptionsSubscribe,
		Unsubscriber:          e.OptionsUnsubscribe,
		Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
			return e.wsHandleData(ctx, conn, asset.Options, resp)
		},
	}); err != nil {
		return err
	}

	wsUSDTLinearURL, err := e.API.Endpoints.GetURL(exchange.WebsocketUSDTMargined)
	if err != nil {
		return err
	}

	// Linear - USDT margined futures inbound public data.
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsUSDTLinearURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Connector:            e.WsConnect,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateLinearDefaultSubscriptions(asset.USDTMarginedFutures)
		},
		Subscriber: func(ctx context.Context, conn websocket.Connection, sub subscription.List) error {
			return e.LinearSubscribe(ctx, conn, asset.USDTMarginedFutures, sub)
		},
		Unsubscriber: func(ctx context.Context, conn websocket.Connection, unsub subscription.List) error {
			return e.LinearUnsubscribe(ctx, conn, asset.USDTMarginedFutures, unsub)
		},
		Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
			return e.wsHandleData(ctx, conn, asset.USDTMarginedFutures, resp)
		},
		MessageFilter: asset.USDTMarginedFutures, // Unused but it allows us to differentiate between the two linear futures types.
	}); err != nil {
		return err
	}

	wsUSDCLinearURL, err := e.API.Endpoints.GetURL(exchange.WebsocketUSDCMargined)
	if err != nil {
		return err
	}

	// Linear - USDC margined futures inbound public data.
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsUSDCLinearURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Connector:            e.WsConnect,
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.GenerateLinearDefaultSubscriptions(asset.USDCMarginedFutures)
		},
		Subscriber: func(ctx context.Context, conn websocket.Connection, sub subscription.List) error {
			return e.LinearSubscribe(ctx, conn, asset.USDCMarginedFutures, sub)
		},
		Unsubscriber: func(ctx context.Context, conn websocket.Connection, unsub subscription.List) error {
			return e.LinearUnsubscribe(ctx, conn, asset.USDCMarginedFutures, unsub)
		},
		Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
			return e.wsHandleData(ctx, conn, asset.USDCMarginedFutures, resp)
		},
		MessageFilter: asset.USDCMarginedFutures, // Unused but it allows us to differentiate between the two linear futures types.
	}); err != nil {
		return err
	}

	wsInverseURL, err := e.API.Endpoints.GetURL(exchange.WebsocketCoinMargined)
	if err != nil {
		return err
	}

	// Inverse - Coin margined futures inbound public data.
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsInverseURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Connector:             e.WsConnect,
		GenerateSubscriptions: e.GenerateInverseDefaultSubscriptions,
		Subscriber:            e.InverseSubscribe,
		Unsubscriber:          e.InverseUnsubscribe,
		Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
			return e.wsHandleData(ctx, conn, asset.CoinMarginedFutures, resp)
		},
	}); err != nil {
		return err
	}

	wsTradeURL, err := e.API.Endpoints.GetURL(exchange.WebsocketTrade)
	if err != nil {
		return err
	}

	// Trade - Dedicated trade connection for all outbound trading requests.
	if err := e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  wsTradeURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Connector:            e.WsConnect,
		Handler: func(_ context.Context, conn websocket.Connection, resp []byte) error {
			return e.wsHandleTradeData(conn, resp)
		},
		Authenticate:             e.WebsocketAuthenticateTradeConnection,
		MessageFilter:            OutboundTradeConnection,
		SubscriptionsNotRequired: true,
	}); err != nil {
		return err
	}

	wsPrivateURL, err := e.API.Endpoints.GetURL(exchange.WebsocketPrivate)
	if err != nil {
		return err
	}

	// Private - Inbound private data connection for authenticated data
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   wsPrivateURL,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		Authenticated:         true,
		Connector:             e.WsConnect,
		GenerateSubscriptions: e.generateAuthSubscriptions,
		Subscriber:            e.authSubscribe,
		Unsubscriber:          e.authUnsubscribe,
		Handler:               e.wsHandleAuthenticatedData,
		Authenticate:          e.WebsocketAuthenticatePrivateConnection,
		MessageFilter:         InboundPrivateConnection,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	var pair currency.Pair
	var category string
	format, err := e.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	var (
		pairs    currency.Pairs
		allPairs []*InstrumentInfo
		response *InstrumentsInfo
	)
	var nextPageCursor string
	switch a {
	case asset.Spot, asset.CoinMarginedFutures, asset.USDCMarginedFutures, asset.USDTMarginedFutures:
		category = getCategoryName(a)
		for {
			response, err = e.GetInstrumentInfo(ctx, category, "", "Trading", "", nextPageCursor, 1000)
			if err != nil {
				return nil, err
			}
			allPairs = append(allPairs, response.List...)
			nextPageCursor = response.NextPageCursor
			if nextPageCursor == "" {
				break
			}
		}
	case asset.Options:
		category = getCategoryName(a)
		for x := range supportedOptionsTypes {
			nextPageCursor = ""
			for {
				response, err = e.GetInstrumentInfo(ctx, category, "", "Trading", supportedOptionsTypes[x], nextPageCursor, 1000)
				if err != nil {
					return nil, err
				}
				allPairs = append(allPairs, response.List...)
				if response.NextPageCursor == "" || (nextPageCursor != "" && nextPageCursor == response.NextPageCursor) || len(response.List) == 0 {
					break
				}
				nextPageCursor = response.NextPageCursor
			}
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	pairs = make(currency.Pairs, 0, len(allPairs))
	var filterSymbol string
	switch a {
	case asset.USDCMarginedFutures:
		filterSymbol = "USDC"
	case asset.USDTMarginedFutures:
		filterSymbol = "USDT"
	case asset.CoinMarginedFutures:
		filterSymbol = "USD"
	}
	for x := range allPairs {
		if allPairs[x].Status != "Trading" || (filterSymbol != "" && allPairs[x].QuoteCoin != filterSymbol) {
			continue
		}
		if a == asset.Options {
			_ = allPairs[x].transformSymbol(a)
		}
		pair, err = currency.NewPairFromString(allPairs[x].transformSymbol(a))
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}

	return pairs.Format(format), nil
}

func getCategoryName(a asset.Item) string {
	switch a {
	case asset.CoinMarginedFutures:
		return cInverse
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		return cLinear
	case asset.Spot:
		return a.String()
	case asset.Options:
		return cOption
	default:
		return ""
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assetTypes := e.GetAssetTypes(true)
	for i := range assetTypes {
		pairs, err := e.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assetTypes[i], false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	enabled, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	format, err := e.GetPairFormat(assetType, false)
	if err != nil {
		return err
	}
	var ticks *TickerData
	switch assetType {
	case asset.Spot, asset.USDCMarginedFutures,
		asset.USDTMarginedFutures,
		asset.CoinMarginedFutures:
		ticks, err = e.GetTickers(ctx, getCategoryName(assetType), "", "", time.Time{})
		if err != nil {
			return err
		}
		for x := range ticks.List {
			var pair currency.Pair
			pair, err = e.MatchSymbolWithAvailablePairs(ticks.List[x].Symbol, assetType, true)
			if err != nil {
				continue
			}
			if !enabled.Contains(pair, true) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks.List[x].LastPrice.Float64(),
				High:         ticks.List[x].HighPrice24H.Float64(),
				Low:          ticks.List[x].LowPrice24H.Float64(),
				Bid:          ticks.List[x].Bid1Price.Float64(),
				BidSize:      ticks.List[x].Bid1Size.Float64(),
				Ask:          ticks.List[x].Ask1Price.Float64(),
				AskSize:      ticks.List[x].Ask1Size.Float64(),
				Volume:       ticks.List[x].Volume24H.Float64(),
				Pair:         pair.Format(format),
				ExchangeName: e.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	case asset.Options:
		for x := range supportedOptionsTypes {
			ticks, err = e.GetTickers(ctx, getCategoryName(assetType), "", supportedOptionsTypes[x], time.Time{})
			if err != nil {
				return err
			}
			for x := range ticks.List {
				var pair currency.Pair
				pair, err = e.MatchSymbolWithAvailablePairs(ticks.List[x].Symbol, assetType, true)
				if err != nil {
					continue
				}
				if !enabled.Contains(pair, true) {
					continue
				}
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         ticks.List[x].LastPrice.Float64(),
					High:         ticks.List[x].HighPrice24H.Float64(),
					Low:          ticks.List[x].LowPrice24H.Float64(),
					Bid:          ticks.List[x].Bid1Price.Float64(),
					BidSize:      ticks.List[x].Bid1Size.Float64(),
					Ask:          ticks.List[x].Ask1Price.Float64(),
					AskSize:      ticks.List[x].Ask1Size.Float64(),
					Volume:       ticks.List[x].Volume24H.Float64(),
					Pair:         pair.Format(format),
					ExchangeName: e.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, assetType); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	var orderbookNew *Orderbook
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		if assetType == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
			p.Delimiter = currency.DashDelimiter
		}
		orderbookNew, err = e.GetOrderBook(ctx, getCategoryName(assetType), p.String(), 0)
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              make([]orderbook.Level, len(orderbookNew.Bids)),
		Asks:              make([]orderbook.Level, len(orderbookNew.Asks)),
	}
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	at, err := e.FetchAccountType(ctx)
	if err != nil {
		return nil, err
	}
	var accountType string
	switch assetType {
	case asset.Spot, asset.Options, asset.USDCMarginedFutures, asset.USDTMarginedFutures:
		switch at {
		case accountTypeUnified:
			accountType = "UNIFIED"
		case accountTypeNormal:
			if assetType == asset.Spot {
				accountType = "SPOT"
			} else {
				accountType = "CONTRACT"
			}
		}
	case asset.CoinMarginedFutures:
		accountType = "CONTRACT"
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	resp, err := e.GetWalletBalance(ctx, accountType, "")
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for i := range resp.List {
		for _, c := range resp.List[i].Coin {
			// borrow amounts get truncated to 8 dec places when total and equity are calculated on the exchange
			truncBorrow := c.BorrowAmount.Decimal().Truncate(8).InexactFloat64()

			// wallet balance can be negative when borrow is present, and wallet balance will be offset with spot holdings
			// e.g. borrow $10,000, wallet balance will be -$9,900 ∴ spot holding $100
			balanceDiff := truncBorrow + c.WalletBalance.Float64()

			freeBalance := balanceDiff - c.Locked.Float64()
			if assetType == asset.Spot && c.AvailableBalanceForSpot.Float64() != 0 {
				freeBalance = c.AvailableBalanceForSpot.Float64()
			}

			subAccts[0].Balances.Set(c.Coin, accounts.Balance{
				Total:                  c.WalletBalance.Float64(),
				Free:                   freeBalance,
				Borrowed:               c.BorrowAmount.Float64(),
				Hold:                   c.Locked.Float64(),
				AvailableWithoutBorrow: c.AvailableToWithdraw.Float64(),
			})
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
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	switch a {
	case asset.Spot, asset.Options, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		withdrawals, err := e.GetWithdrawalRecords(ctx, c, "", "2", "", time.Time{}, time.Time{}, 0)
		if err != nil {
			return nil, err
		}

		withdrawHistory := make([]exchange.WithdrawalHistory, len(withdrawals.Rows))
		for i := range withdrawals.Rows {
			withdrawHistory[i] = exchange.WithdrawalHistory{
				TransferID:      withdrawals.Rows[i].WithdrawID,
				Status:          withdrawals.Rows[i].Status,
				Currency:        withdrawals.Rows[i].Coin,
				Amount:          withdrawals.Rows[i].Amount.Float64(),
				Fee:             withdrawals.Rows[i].WithdrawFee.Float64(),
				CryptoToAddress: withdrawals.Rows[i].ToAddress,
				CryptoTxID:      withdrawals.Rows[i].TransactionID,
				CryptoChain:     withdrawals.Rows[i].Chain,
				Timestamp:       withdrawals.Rows[i].UpdateTime.Time(),
			}
		}
		return withdrawHistory, nil
	default:
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	formattedPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	limit := int64(500)
	if assetType == asset.Spot {
		limit = 60
	}
	var tradeData *TradingHistory
	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		if assetType == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
			formattedPair.Delimiter = currency.DashDelimiter
		}
		tradeData, err = e.GetPublicTradingHistory(ctx, getCategoryName(assetType), formattedPair.String(), "", "", limit)
	case asset.Options:
		tradeData, err = e.GetPublicTradingHistory(ctx, getCategoryName(assetType), formattedPair.String(), formattedPair.Base.String(), "", limit)
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData.List))
	for i := range tradeData.List {
		side, err := order.StringToOrderSide(tradeData.List[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			CurrencyPair: formattedPair,
			AssetType:    assetType,
			Price:        tradeData.List[i].Price.Float64(),
			Amount:       tradeData.List[i].Size.Float64(),
			Timestamp:    tradeData.List[i].TradeTime.Time(),
			TID:          tradeData.List[i].ExecutionID,
			Side:         side,
		}
	}

	if e.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, _, _ time.Time) ([]trade.Data, error) {
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	limit := int64(1000)
	if assetType == asset.Spot {
		limit = 60
	}
	var tradeHistoryResponse *TradingHistory
	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		if assetType == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
			p.Delimiter = currency.DashDelimiter
		}
		tradeHistoryResponse, err = e.GetPublicTradingHistory(ctx, getCategoryName(assetType), p.String(), "", "", limit)
		if err != nil {
			return nil, err
		}
	case asset.Options:
		tradeHistoryResponse, err = e.GetPublicTradingHistory(ctx, getCategoryName(assetType), p.String(), p.Base.String(), "", limit)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	resp := make([]trade.Data, len(tradeHistoryResponse.List))
	for x := range tradeHistoryResponse.List {
		side, err := order.StringToOrderSide(tradeHistoryResponse.List[x].Side)
		if err != nil {
			return nil, err
		}
		resp[x] = trade.Data{
			TID:          tradeHistoryResponse.List[x].ExecutionID,
			Exchange:     e.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeHistoryResponse.List[x].Price.Float64(),
			Amount:       tradeHistoryResponse.List[x].Size.Float64(),
			Timestamp:    tradeHistoryResponse.List[x].TradeTime.Time(),
		}
	}
	return resp, nil
}

func orderTypeToString(oType order.Type) string {
	switch oType {
	case order.Limit:
		return "Limit"
	case order.Market:
		return "Market"
	default:
		return oType.String()
	}
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	arg, err := e.deriveSubmitOrderArguments(s)
	if err != nil {
		return nil, err
	}
	response, err := e.PlaceOrder(ctx, arg)
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(response.OrderID)
	if err != nil {
		return nil, err
	}
	resp.Status = order.New
	return resp, nil
}

// WebsocketSubmitOrder submits a new order through the websocket connection
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	arg, err := e.deriveSubmitOrderArguments(s)
	if err != nil {
		return nil, err
	}
	orderDetails, err := e.WSCreateOrder(ctx, arg)
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(orderDetails.OrderID)
	if err != nil {
		return nil, err
	}
	resp.Status, err = order.StringToOrderStatus(orderDetails.OrderStatus)
	if err != nil {
		return nil, err
	}
	resp.TimeInForce, err = order.StringToTimeInForce(orderDetails.TimeInForce)
	if err != nil {
		return nil, err
	}

	resp.ReduceOnly = orderDetails.ReduceOnly
	resp.TriggerPrice = orderDetails.TriggerPrice.Float64()
	resp.AverageExecutedPrice = orderDetails.AveragePrice.Float64()
	resp.ClientOrderID = orderDetails.OrderLinkID
	resp.Fee = orderDetails.CumulativeExecutedFee.Float64()
	resp.Cost = orderDetails.CumulativeExecutedValue.Float64()
	return resp, nil
}

func getOrderTypeString(oType order.Type) string {
	switch oType {
	case order.UnknownType:
		return ""
	default:
		return oType.String()
	}
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	arg, err := e.deriveAmendOrderArguments(action)
	if err != nil {
		return nil, err
	}
	result, err := e.AmendOrder(ctx, arg)
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = result.OrderID
	return resp, nil
}

// WebsocketModifyOrder modifies an existing order
func (e *Exchange) WebsocketModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	arg, err := e.deriveAmendOrderArguments(action)
	if err != nil {
		return nil, err
	}
	result, err := e.WSAmendOrder(ctx, arg)
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = result.OrderID
	resp.ClientOrderID = result.OrderLinkID
	resp.Amount = result.Quantity.Float64()
	resp.Price = action.Price
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	arg, err := e.deriveCancelOrderArguments(ord)
	if err != nil {
		return err
	}
	_, err = e.CancelTradeOrder(ctx, arg)
	return err
}

// WebsocketCancelOrder cancels an order by ID
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, ord *order.Cancel) error {
	arg, err := e.deriveCancelOrderArguments(ord)
	if err != nil {
		return err
	}
	_, err = e.WSCancelOrder(ctx, arg)
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	requests := make([]CancelOrderRequest, len(o))
	category := asset.Options
	var err error
	for i := range o {
		switch o[i].AssetType {
		case asset.Options:
		default:
			return nil, fmt.Errorf("%w, only 'option' category is allowed, but given %v", asset.ErrNotSupported, o[i].AssetType)
		}
		switch {
		case o[i].Pair.IsEmpty():
			return nil, currency.ErrCurrencyPairEmpty
		case o[i].ClientOrderID == "" && o[i].OrderID == "":
			return nil, order.ErrOrderIDNotSet
		default:
			o[i].Pair, err = e.FormatExchangeCurrency(o[i].Pair, category)
			if err != nil {
				return nil, err
			}
			requests[i] = CancelOrderRequest{
				OrderID:     o[i].OrderID,
				OrderLinkID: o[i].ClientOrderID,
				Symbol:      o[i].Pair,
			}
		}
	}
	cancelledOrders, err := e.CancelBatchOrder(ctx, &CancelBatchOrder{
		Category: getCategoryName(category),
		Request:  requests,
	})
	if err != nil {
		return nil, err
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	for i := range cancelledOrders {
		resp.Status[cancelledOrders[i].OrderID] = "success"
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	err := orderCancellation.Validate()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	orderCancellation.Pair, err = e.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	status := "success"
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch orderCancellation.AssetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		if orderCancellation.AssetType == asset.USDCMarginedFutures && !orderCancellation.Pair.Quote.Equal(currency.PERP) {
			orderCancellation.Pair.Delimiter = currency.DashDelimiter
		}
		activeOrder, err := e.CancelAllTradeOrders(ctx, &CancelAllOrdersParam{
			Category: getCategoryName(orderCancellation.AssetType),
			Symbol:   orderCancellation.Pair,
			BaseCoin: orderCancellation.Pair.Base.String(),
		})
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range activeOrder {
			cancelAllOrdersResponse.Status[activeOrder[i].OrderID] = status
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%s %w", orderCancellation.AssetType, asset.ErrNotSupported)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	} else if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	pair, err := e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		if assetType == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
			pair.Delimiter = currency.DashDelimiter
		}
		resp, err := e.GetOpenOrders(ctx, getCategoryName(assetType), pair.String(), "", "", orderID, "", "", "", 0, 1)
		if err != nil {
			return nil, err
		}
		if len(resp.List) != 1 {
			return nil, order.ErrOrderNotFound
		}
		orderType, err := order.StringToOrderType(resp.List[0].OrderType)
		if err != nil {
			return nil, err
		}
		remainingAmt := resp.List[0].LeavesQuantity.Float64()
		if remainingAmt == 0 {
			remainingAmt = resp.List[0].OrderQuantity.Float64() - resp.List[0].CumulativeExecQuantity.Float64()
		}
		return &order.Detail{
			Amount:          resp.List[0].OrderQuantity.Float64(),
			Exchange:        e.Name,
			OrderID:         resp.List[0].OrderID,
			ClientOrderID:   resp.List[0].OrderLinkID,
			Side:            getSide(resp.List[0].Side),
			Type:            orderType,
			Pair:            pair,
			Cost:            resp.List[0].CumulativeExecQuantity.Float64() * resp.List[0].AveragePrice.Float64(),
			AssetType:       assetType,
			Status:          StringToOrderStatus(resp.List[0].OrderStatus),
			Price:           resp.List[0].Price.Float64(),
			ExecutedAmount:  resp.List[0].CumulativeExecQuantity.Float64(),
			RemainingAmount: remainingAmt,
			Date:            resp.List[0].CreatedTime.Time(),
			LastUpdated:     resp.List[0].UpdatedTime.Time(),
		}, nil
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	dAddressInfo, err := e.GetMasterDepositAddress(ctx, cryptocurrency, chain)
	if err != nil {
		return nil, err
	}

	for x := range dAddressInfo.Chains {
		if dAddressInfo.Chains[x].Chain == chain || chain == "" {
			return &deposit.Address{
				Address: dAddressInfo.Chains[x].AddressDeposit,
				Tag:     dAddressInfo.Chains[x].TagDeposit,
				Chain:   dAddressInfo.Chains[x].Chain,
			}, nil
		}
	}
	return nil, fmt.Errorf("%w for currency: %s chain: %s", deposit.ErrAddressNotFound, cryptocurrency, chain)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	info, err := e.GetCoinInfo(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	var availableChains []string
	for x := range info.Rows {
		if strings.EqualFold(info.Rows[x].Coin, cryptocurrency.String()) {
			for i := range info.Rows[x].Chains {
				availableChains = append(availableChains, info.Rows[x].Chains[i].Chain)
			}
		}
	}
	return availableChains, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	wID, err := e.WithdrawCurrency(ctx,
		&WithdrawalParam{
			Coin:      withdrawRequest.Currency,
			Chain:     withdrawRequest.Crypto.Chain,
			Address:   withdrawRequest.Crypto.Address,
			Tag:       withdrawRequest.Crypto.AddressTag,
			Amount:    withdrawRequest.Amount,
			Timestamp: time.Now().UnixMilli(),
		})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: wID,
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
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	format, err := e.GetPairFormat(req.AssetType, true)
	if err != nil {
		return nil, err
	}
	var baseCoin currency.Code
	req.Pairs = req.Pairs.Format(format)
	for i := range req.Pairs {
		if baseCoin != currency.EMPTYCODE && req.Pairs[i].Base != baseCoin {
			baseCoin = currency.EMPTYCODE
		} else if req.Pairs[i].Base != currency.EMPTYCODE {
			baseCoin = req.Pairs[i].Base
		}
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		if baseCoin != currency.EMPTYCODE {
			openOrders, err := e.GetOpenOrders(ctx, getCategoryName(req.AssetType), "", baseCoin.String(), "", req.FromOrderID, "", "", "", 0, 50)
			if err != nil {
				return nil, err
			}
			newOpenOrders, err := e.ConstructOrderDetails(openOrders.List, req.AssetType, currency.EMPTYPAIR, req.Pairs)
			if err != nil {
				return nil, err
			}
			orders = append(orders, newOpenOrders...)
		} else {
			for y := range req.Pairs {
				if req.AssetType == asset.USDCMarginedFutures && !req.Pairs[y].Quote.Equal(currency.PERP) {
					req.Pairs[y].Delimiter = currency.DashDelimiter
				}
				openOrders, err := e.GetOpenOrders(ctx, getCategoryName(req.AssetType), req.Pairs[y].String(), "", "", req.FromOrderID, "", "", "", 0, 50)
				if err != nil {
					return nil, err
				}
				newOpenOrders, err := e.ConstructOrderDetails(openOrders.List, req.AssetType, req.Pairs[y], currency.Pairs{})
				if err != nil {
					return nil, err
				}
				orders = append(orders, newOpenOrders...)
			}
		}
	default:
		return orders, fmt.Errorf("%s %w", req.AssetType, asset.ErrNotSupported)
	}
	return req.Filter(e.Name, orders), nil
}

// ConstructOrderDetails constructs list of order.Detail instances given list of TradeOrder and other filtering information
func (e *Exchange) ConstructOrderDetails(tradeOrders []TradeOrder, assetType asset.Item, pair currency.Pair, filterPairs currency.Pairs) (order.FilteredOrders, error) {
	orders := make([]order.Detail, 0, len(tradeOrders))
	var err error
	var ePair currency.Pair
	for x := range tradeOrders {
		ePair, err = e.MatchSymbolWithAvailablePairs(tradeOrders[x].Symbol, assetType, true)
		if err != nil {
			return nil, err
		}
		if (pair.IsEmpty() && len(filterPairs) > 0 && !filterPairs.Contains(ePair, true)) ||
			(!pair.IsEmpty() && !pair.Equal(ePair)) {
			continue
		}
		orderType, err := order.StringToOrderType(tradeOrders[x].OrderType)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order.Detail{
			Amount:               tradeOrders[x].OrderQuantity.Float64(),
			Date:                 tradeOrders[x].CreatedTime.Time(),
			Exchange:             e.Name,
			OrderID:              tradeOrders[x].OrderID,
			ClientOrderID:        tradeOrders[x].OrderLinkID,
			Side:                 getSide(tradeOrders[x].Side),
			Type:                 orderType,
			Price:                tradeOrders[x].Price.Float64(),
			Status:               StringToOrderStatus(tradeOrders[x].OrderStatus),
			Pair:                 ePair,
			AssetType:            assetType,
			LastUpdated:          tradeOrders[x].UpdatedTime.Time(),
			ReduceOnly:           tradeOrders[x].ReduceOnly,
			ExecutedAmount:       tradeOrders[x].CumulativeExecQuantity.Float64(),
			RemainingAmount:      tradeOrders[x].LeavesQuantity.Float64(),
			TriggerPrice:         tradeOrders[x].TriggerPrice.Float64(),
			AverageExecutedPrice: tradeOrders[x].AveragePrice.Float64(),
			Cost:                 tradeOrders[x].AveragePrice.Float64() * tradeOrders[x].CumulativeExecQuantity.Float64(),
			Fee:                  tradeOrders[x].CumulativeExecFee.Float64(),
		})
	}
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	limit := int64(200)
	if req.AssetType == asset.Options {
		limit = 25
	}
	format, err := e.GetPairFormat(req.AssetType, false)
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		resp, err := e.GetTradeOrderHistory(ctx, getCategoryName(req.AssetType), "", req.FromOrderID, "", "", "", "", "", "", req.StartTime, req.EndTime, limit)
		if err != nil {
			return nil, err
		}

		for i := range resp.List {
			// here, we are not using getSide because in sample response's sides are in upper
			var side order.Side
			side, err = order.StringToOrderSide(resp.List[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}

			var pair currency.Pair
			pair, err = e.MatchSymbolWithAvailablePairs(resp.List[i].Symbol, req.AssetType, true)
			if err != nil {
				return nil, err
			}
			orderType, err := order.StringToOrderType(resp.List[i].OrderType)
			if err != nil {
				return nil, err
			}
			detail := order.Detail{
				Amount:               resp.List[i].OrderQuantity.Float64(),
				ExecutedAmount:       resp.List[i].CumulativeExecQuantity.Float64(),
				RemainingAmount:      resp.List[i].LeavesQuantity.Float64(),
				Date:                 resp.List[i].CreatedTime.Time(),
				LastUpdated:          resp.List[i].UpdatedTime.Time(),
				Exchange:             e.Name,
				OrderID:              resp.List[i].OrderID,
				Side:                 side,
				Type:                 orderType,
				Price:                resp.List[i].Price.Float64(),
				Pair:                 pair.Format(format),
				Status:               StringToOrderStatus(resp.List[i].OrderStatus),
				ReduceOnly:           resp.List[i].ReduceOnly,
				TriggerPrice:         resp.List[i].TriggerPrice.Float64(),
				AverageExecutedPrice: resp.List[i].AveragePrice.Float64(),
				Cost:                 resp.List[i].AveragePrice.Float64() * resp.List[i].CumulativeExecQuantity.Float64(),
				CostAsset:            pair.Quote,
				Fee:                  resp.List[i].CumulativeExecFee.Float64(),
				ClientOrderID:        resp.List[i].OrderLinkID,
				AssetType:            req.AssetType,
			}
			orders = append(orders, detail)
		}
	case asset.Spot:
		resp, err := e.GetTradeOrderHistory(ctx, getCategoryName(req.AssetType), "", req.FromOrderID, "", "", "", "", "", "", req.StartTime, req.EndTime, limit)
		if err != nil {
			return nil, err
		}

		for i := range resp.List {
			// here, we are not using getSide because in sample response's sides are in upper
			var side order.Side
			side, err = order.StringToOrderSide(resp.List[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}
			var pair currency.Pair
			pair, err = e.MatchSymbolWithAvailablePairs(resp.List[i].Symbol, req.AssetType, true)
			if err != nil {
				return nil, err
			}
			orderType, err := order.StringToOrderType(resp.List[i].OrderType)
			if err != nil {
				return nil, err
			}
			detail := order.Detail{
				Amount:               resp.List[i].OrderQuantity.Float64(),
				ExecutedAmount:       resp.List[i].CumulativeExecQuantity.Float64(),
				RemainingAmount:      resp.List[i].CumulativeExecQuantity.Float64() - resp.List[i].CumulativeExecQuantity.Float64(),
				Cost:                 resp.List[i].AveragePrice.Float64() * resp.List[i].CumulativeExecQuantity.Float64(),
				Date:                 resp.List[i].CreatedTime.Time(),
				LastUpdated:          resp.List[i].UpdatedTime.Time(),
				Exchange:             e.Name,
				OrderID:              resp.List[i].OrderID,
				Side:                 side,
				Type:                 orderType,
				Price:                resp.List[i].Price.Float64(),
				Pair:                 pair.Format(format),
				Status:               StringToOrderStatus(resp.List[i].OrderStatus),
				ReduceOnly:           resp.List[i].ReduceOnly,
				TriggerPrice:         resp.List[i].TriggerPrice.Float64(),
				AverageExecutedPrice: resp.List[i].AveragePrice.Float64(),
				CostAsset:            pair.Quote,
				ClientOrderID:        resp.List[i].OrderLinkID,
				AssetType:            req.AssetType,
			}
			orders = append(orders, detail)
		}
	default:
		return orders, fmt.Errorf("%s %w", req.AssetType, asset.ErrNotSupported)
	}
	order.FilterOrdersByPairs(&orders, req.Pairs)
	return req.Filter(e.Name, orders), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder.Pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		return getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount), nil
	default:
		assets := e.getCategoryFromPair(feeBuilder.Pair)
		var err error
		var baseCoin, pairString string
		if assets[0] == asset.Options {
			baseCoin = feeBuilder.Pair.Base.String()
		} else {
			pairString, err = e.FormatSymbol(feeBuilder.Pair, assets[0])
			if err != nil {
				return 0, err
			}
		}
		accountFee, err := e.GetFeeRate(ctx, getCategoryName(assets[0]), pairString, baseCoin)
		if err != nil {
			return 0, err
		}
		if len(accountFee.List) == 0 {
			return 0, fmt.Errorf("no fee builder found for currency pair %s", pairString)
		}
		if feeBuilder.IsMaker {
			return accountFee.List[0].Maker.Float64() * feeBuilder.Amount, nil
		}
		return accountFee.List[0].Taker.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	}
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.01 * price * amount
}

func (e *Exchange) getCategoryFromPair(pair currency.Pair) []asset.Item {
	assets := e.GetAssetTypes(true)
	containingAssets := make([]asset.Item, 0, len(assets))
	for a := range assets {
		pairs, err := e.GetAvailablePairs(assets[a])
		if err != nil {
			continue
		}
		if pairs.Contains(pair, true) {
			containingAssets = append(containingAssets, assets[a])
		}
	}
	return containingAssets
}

// ValidateAPICredentials validates current credentials used for wrapper
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
		if err != nil {
			return nil, err
		}
		var timeSeries []kline.Candle
		if a == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
			req.RequestFormatted.Delimiter = currency.DashDelimiter
		}
		var candles []KlineItem
		candles, err = e.GetKlines(ctx, getCategoryName(req.Asset), req.RequestFormatted.String(), req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}

		timeSeries = make([]kline.Candle, len(candles))
		for x := range candles {
			timeSeries[x] = kline.Candle{
				Time:   candles[x].StartTime.Time(),
				Open:   candles[x].Open.Float64(),
				High:   candles[x].High.Float64(),
				Low:    candles[x].Low.Float64(),
				Close:  candles[x].Close.Float64(),
				Volume: candles[x].TradeVolume.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, 0, req.Size())
		for x := range req.RangeHolder.Ranges {
			if req.Asset == asset.USDCMarginedFutures && !req.RequestFormatted.Quote.Equal(currency.PERP) {
				req.RequestFormatted.Delimiter = currency.DashDelimiter
			}
			var klineItems []KlineItem
			klineItems, err = e.GetKlines(ctx,
				getCategoryName(req.Asset),
				req.RequestFormatted.String(),
				req.ExchangeInterval,
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time,
				req.RequestLimit)
			if err != nil {
				return nil, err
			}

			for i := range klineItems {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   klineItems[i].StartTime.Time(),
					Open:   klineItems[i].Open.Float64(),
					High:   klineItems[i].High.Float64(),
					Low:    klineItems[i].Low.Float64(),
					Close:  klineItems[i].Close.Float64(),
					Volume: klineItems[i].TradeVolume.Float64(),
				})
			}
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	info, err := e.GetBybitServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return info.TimeNano.Time(), err
}

// transformSymbol returns a symbol with a delimiter added if missing
// * Spot and Coin-M add "_"
// * Options, USDC-M USDT-M add "-"
// * CrossMargin is left without a delimiter
func (i *InstrumentInfo) transformSymbol(a asset.Item) string {
	switch a {
	case asset.Spot, asset.CoinMarginedFutures:
		quote := i.Symbol[len(i.BaseCoin):]
		return i.BaseCoin + "_" + quote
	case asset.Options:
		quote := strings.TrimPrefix(i.Symbol[len(i.BaseCoin):], currency.DashDelimiter)
		return i.BaseCoin + "-" + quote
	case asset.USDTMarginedFutures:
		quote := i.Symbol[len(i.BaseCoin):]
		return i.BaseCoin + "-" + quote
	case asset.USDCMarginedFutures:
		if i.ContractType != "LinearFutures" {
			quote := i.Symbol[len(i.BaseCoin):]
			return i.BaseCoin + "-" + quote
		}
		fallthrough // Contracts with linear futures already have a delimiter
	default:
		return i.Symbol
	}
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var (
		allInstrumentsInfo InstrumentsInfo
		nextPageCursor     string
	)
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		for {
			instrumentInfo, err := e.GetInstrumentInfo(ctx, getCategoryName(a), "", "", "", nextPageCursor, 1000)
			if err != nil {
				return err
			}
			switch a {
			case asset.USDTMarginedFutures:
				for i := range instrumentInfo.List {
					if instrumentInfo.List[i].QuoteCoin != "USDT" {
						continue
					}
					allInstrumentsInfo.List = append(allInstrumentsInfo.List, instrumentInfo.List[i])
				}
			case asset.USDCMarginedFutures:
				for i := range instrumentInfo.List {
					if instrumentInfo.List[i].QuoteCoin != "USDC" {
						continue
					}
					allInstrumentsInfo.List = append(allInstrumentsInfo.List, instrumentInfo.List[i])
				}
			default:
				allInstrumentsInfo.List = append(allInstrumentsInfo.List, instrumentInfo.List...)
			}
			nextPageCursor = instrumentInfo.NextPageCursor
			if nextPageCursor == "" {
				break
			}
		}
	case asset.Options:
		for i := range supportedOptionsTypes {
			nextPageCursor = ""
			for {
				instrumentInfo, err := e.GetInstrumentInfo(ctx, getCategoryName(a), "", "", supportedOptionsTypes[i], nextPageCursor, 1000)
				if err != nil {
					return fmt.Errorf("%w - %v", err, supportedOptionsTypes[i])
				}
				allInstrumentsInfo.List = append(allInstrumentsInfo.List, instrumentInfo.List...)
				nextPageCursor = instrumentInfo.NextPageCursor
				if nextPageCursor == "" {
					break
				}
			}
		}
	default:
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	l := make([]limits.MinMaxLevel, 0, len(allInstrumentsInfo.List))
	for _, inst := range allInstrumentsInfo.List {
		symbol := inst.transformSymbol(a)
		pair, err := e.MatchSymbolWithAvailablePairs(symbol, a, true)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s unable to load limits for %s %v, pair data missing", e.Name, a, symbol)
			continue
		}

		priceDivisor := 1.0
		if symbol[:2] == "10" { // handle 1000SHIBUSDT, 1000PEPEUSDT etc; screen 1INCHUSDT
			for _, r := range symbol[1:] {
				if r != '0' {
					break
				}
				priceDivisor *= 10
			}
		}

		var delistingAt time.Time
		var delistedAt time.Time
		var delivery time.Time
		if !inst.DeliveryTime.Time().IsZero() {
			switch a {
			case asset.Options:
				delivery = inst.DeliveryTime.Time()
			case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.USDCMarginedFutures:
				switch inst.ContractType {
				case "LinearFutures", "InverseFutures":
					delivery = inst.DeliveryTime.Time()
				default:
					delistedAt = inst.DeliveryTime.Time()
					// Not entirely accurate but from docs the system will use the average index price in the last
					// 30 minutes before the delisting time. See: https://www.bybit.com/en/help-center/article/Bybit-Derivatives-Delisting-Mechanism-DDM
					delistingAt = delistedAt.Add(-30 * time.Minute)
				}
			case asset.Spot:
				// asset.Spot does not return a delivery time and there is no API field for delisting time
				log.Warnf(log.ExchangeSys, "%s %s: delivery time returned for spot asset", e.Name, pair)
			}
		}

		baseStepAmount := inst.LotSizeFilter.QuantityStep.Float64()
		if a == asset.Spot {
			baseStepAmount = inst.LotSizeFilter.BasePrecision.Float64()
		}

		maxBaseAmount := inst.LotSizeFilter.MaxOrderQuantity.Float64()
		if a != asset.Spot && a != asset.Options {
			maxBaseAmount = inst.LotSizeFilter.MaxMarketOrderQuantity.Float64()
		}

		minQuoteAmount := inst.LotSizeFilter.MinOrderAmount.Float64()
		if a != asset.Spot {
			minQuoteAmount = inst.LotSizeFilter.MinNotionalValue.Float64()
		}

		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
			MinimumBaseAmount:       inst.LotSizeFilter.MinOrderQuantity.Float64(),
			MaximumBaseAmount:       maxBaseAmount,
			MinPrice:                inst.PriceFilter.MinPrice.Float64(),
			MaxPrice:                inst.PriceFilter.MaxPrice.Float64(),
			PriceStepIncrementSize:  inst.PriceFilter.TickSize.Float64(),
			AmountStepIncrementSize: baseStepAmount,
			QuoteStepIncrementSize:  inst.LotSizeFilter.QuotePrecision.Float64(),
			MinimumQuoteAmount:      minQuoteAmount,
			MaximumQuoteAmount:      inst.LotSizeFilter.MaxOrderAmount.Float64(),
			Delisting:               delistingAt,
			Delisted:                delistedAt,
			Expiry:                  delivery,
			PriceDivisor:            priceDivisor,
			Listed:                  inst.LaunchTime.Time(),
			MultiplierDecimal:       1, // All assets on Bybit are 1x
		})
	}
	return limits.Load(l)
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, amount float64, orderSide order.Side) error {
	switch item {
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		var err error
		pair, err = e.FormatExchangeCurrency(pair, item)
		if err != nil {
			return err
		}
		if item == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
			pair.Delimiter = currency.DashDelimiter
		}
		params := &SetLeverageParams{
			Category: getCategoryName(item),
			Symbol:   pair.String(),
		}
		switch orderSide {
		case order.Buy, order.Sell:
			// Unified account: buyLeverage must be the same as sellLeverage all the time
			// Classic account: under one-way mode, buyLeverage must be the same as sellLeverage
			params.BuyLeverage, params.SellLeverage = amount, amount
		case order.UnknownSide:
			return order.ErrSideIsInvalid
		default:
			return order.ErrSideIsInvalid
		}
		return e.SetLeverageLevel(ctx, params)
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, p currency.Pair) (bool, error) {
	if !a.IsFutures() {
		return false, nil
	}
	return p.Quote.Equal(currency.PERP) ||
		p.Quote.Equal(currency.USD) ||
		p.Quote.Equal(currency.USDC) ||
		p.Quote.Equal(currency.USDT), nil
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	inverseContracts, err := e.GetInstrumentInfo(ctx, getCategoryName(item), "", "", "", "", 1000)
	if err != nil {
		return nil, err
	}
	format, err := e.GetPairFormat(item, false)
	if err != nil {
		return nil, err
	}
	switch item {
	case asset.CoinMarginedFutures:
		resp := make([]futures.Contract, 0, len(inverseContracts.List))
		for i := range inverseContracts.List {
			if inverseContracts.List[i].SettleCoin.Equal(currency.USDT) || inverseContracts.List[i].SettleCoin.Equal(currency.USDC) {
				continue
			}
			var cp, underlying currency.Pair
			cp, err = currency.NewPairFromStrings(inverseContracts.List[i].BaseCoin, inverseContracts.List[i].Symbol[len(inverseContracts.List[i].BaseCoin):])
			if err != nil {
				return nil, err
			}

			underlying, err = currency.NewPairFromStrings(inverseContracts.List[i].BaseCoin, inverseContracts.List[i].QuoteCoin)
			if err != nil {
				return nil, err
			}
			contractType := strings.ToLower(inverseContracts.List[i].ContractType)
			var start, end time.Time
			if inverseContracts.List[i].LaunchTime.Time().UnixMilli() > 0 {
				start = inverseContracts.List[i].LaunchTime.Time()
			}
			if inverseContracts.List[i].DeliveryTime.Time().UnixMilli() > 0 {
				end = inverseContracts.List[i].DeliveryTime.Time()
			}

			var ct futures.ContractType
			switch contractType {
			case "inverseperpetual":
				ct = futures.Perpetual
			case "inversefutures":
				ct, err = getContractLength(end.Sub(start))
				if err != nil {
					return nil, fmt.Errorf("%w %v %v %v %v-%v", err, e.Name, item, cp, inverseContracts.List[i].LaunchTime.Time(), inverseContracts.List[i].DeliveryTime)
				}
			default:
				if e.Verbose {
					log.Warnf(log.ExchangeSys, "%v unhandled contract type for %v %v %v-%v", e.Name, item, cp, start, end)
				}
				ct = futures.Unknown
			}

			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp.Format(format),
				Underlying:         underlying,
				Asset:              item,
				StartDate:          start,
				EndDate:            end,
				SettlementType:     futures.Inverse,
				IsActive:           strings.EqualFold(inverseContracts.List[i].Status, "trading"),
				Status:             inverseContracts.List[i].Status,
				Type:               ct,
				SettlementCurrency: inverseContracts.List[i].SettleCoin,
				MaxLeverage:        inverseContracts.List[i].LeverageFilter.MaxLeverage.Float64(),
			})
		}
		return resp, nil
	case asset.USDCMarginedFutures:
		linearContracts, err := e.GetInstrumentInfo(ctx, cLinear, "", "", "", "", 1000)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(inverseContracts.List)+len(linearContracts.List))

		var instruments []*InstrumentInfo
		for i := range linearContracts.List {
			if !linearContracts.List[i].SettleCoin.Equal(currency.USDC) {
				continue
			}
			instruments = append(instruments, linearContracts.List[i])
		}
		for i := range inverseContracts.List {
			if !inverseContracts.List[i].SettleCoin.Equal(currency.USDC) {
				continue
			}
			instruments = append(instruments, inverseContracts.List[i])
		}
		for i := range instruments {
			var cp, underlying currency.Pair
			underlying, err = currency.NewPairFromStrings(instruments[i].BaseCoin, instruments[i].QuoteCoin)
			if err != nil {
				return nil, err
			}
			contractType := strings.ToLower(instruments[i].ContractType)

			var ct futures.ContractType
			switch contractType {
			case "linearperpetual":
				ct = futures.Perpetual
				cp, err = currency.NewPairFromStrings(instruments[i].BaseCoin, instruments[i].Symbol[len(instruments[i].BaseCoin):])
				if err != nil {
					return nil, err
				}
			case "linearfutures":
				ct, err = getContractLength(instruments[i].DeliveryTime.Time().Sub(instruments[i].LaunchTime.Time()))
				if err != nil {
					return nil, fmt.Errorf("%w %v %v %v %v-%v", err, e.Name, item, cp, instruments[i].LaunchTime.Time(), instruments[i].DeliveryTime.Time())
				}
				cp, err = e.MatchSymbolWithAvailablePairs(instruments[i].Symbol, item, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
			default:
				if e.Verbose {
					log.Warnf(log.ExchangeSys, "%v unhandled contract type for %v %v %v-%v", e.Name, item, cp, instruments[i].LaunchTime.Time(), instruments[i].DeliveryTime.Time())
				}
				ct = futures.Unknown
				cp, err = e.MatchSymbolWithAvailablePairs(instruments[i].Symbol, item, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
			}

			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp.Format(format),
				Underlying:         underlying,
				Asset:              item,
				StartDate:          instruments[i].LaunchTime.Time(),
				EndDate:            instruments[i].DeliveryTime.Time(),
				SettlementType:     futures.Linear,
				IsActive:           strings.EqualFold(instruments[i].Status, "trading"),
				Status:             instruments[i].Status,
				Type:               ct,
				SettlementCurrency: currency.USDC,
				MaxLeverage:        instruments[i].LeverageFilter.MaxLeverage.Float64(),
				Multiplier:         instruments[i].LeverageFilter.LeverageStep.Float64(),
			})
		}
		return resp, nil
	case asset.USDTMarginedFutures:
		linearContracts, err := e.GetInstrumentInfo(ctx, cLinear, "", "", "", "", 1000)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(inverseContracts.List)+len(linearContracts.List))

		var instruments []*InstrumentInfo
		for i := range linearContracts.List {
			if !linearContracts.List[i].SettleCoin.Equal(currency.USDT) {
				continue
			}
			instruments = append(instruments, linearContracts.List[i])
		}
		for i := range inverseContracts.List {
			if !inverseContracts.List[i].SettleCoin.Equal(currency.USDT) {
				continue
			}
			instruments = append(instruments, inverseContracts.List[i])
		}
		for i := range instruments {
			var cp, underlying currency.Pair
			cp, err = currency.NewPairFromStrings(instruments[i].BaseCoin, instruments[i].Symbol[len(instruments[i].BaseCoin):])
			if err != nil {
				return nil, err
			}

			underlying, err = currency.NewPairFromStrings(instruments[i].BaseCoin, instruments[i].QuoteCoin)
			if err != nil {
				return nil, err
			}
			contractType := strings.ToLower(instruments[i].ContractType)
			var start, end time.Time
			if !instruments[i].LaunchTime.Time().IsZero() {
				start = instruments[i].LaunchTime.Time()
			}
			if !instruments[i].DeliveryTime.Time().IsZero() {
				end = instruments[i].DeliveryTime.Time()
			}

			var ct futures.ContractType
			switch contractType {
			case "linearperpetual":
				ct = futures.Perpetual
			case "linearfutures":
				ct, err = getContractLength(end.Sub(start))
				if err != nil {
					return nil, fmt.Errorf("%w %v %v %v %v-%v", err, e.Name, item, cp, start, end)
				}
			default:
				if e.Verbose {
					log.Warnf(log.ExchangeSys, "%v unhandled contract type for %v %v %v-%v", e.Name, item, cp, start, end)
				}
				ct = futures.Unknown
			}

			resp = append(resp, futures.Contract{
				Exchange:           e.Name,
				Name:               cp.Format(format),
				Underlying:         underlying,
				Asset:              item,
				StartDate:          start,
				EndDate:            end,
				SettlementType:     futures.Linear,
				IsActive:           strings.EqualFold(instruments[i].Status, "trading"),
				Status:             instruments[i].Status,
				Type:               ct,
				SettlementCurrency: currency.USDT,
				MaxLeverage:        instruments[i].LeverageFilter.MaxLeverage.Float64(),
				Multiplier:         instruments[i].LeverageFilter.LeverageStep.Float64(),
			})
		}
		return resp, nil
	}

	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
}

func getContractLength(contractLength time.Duration) (futures.ContractType, error) {
	if contractLength <= 0 {
		return futures.Unknown, errInvalidContractLength
	}
	var ct futures.ContractType
	switch {
	case contractLength > 0 && contractLength <= kline.OneWeek.Duration()+kline.ThreeDay.Duration():
		ct = futures.Weekly
	case contractLength <= kline.TwoWeek.Duration()+kline.ThreeDay.Duration():
		ct = futures.Fortnightly
	case contractLength <= kline.ThreeWeek.Duration()+kline.ThreeDay.Duration():
		ct = futures.ThreeWeekly
	case contractLength <= kline.ThreeMonth.Duration()+kline.ThreeWeek.Duration():
		ct = futures.Quarterly
	case contractLength <= kline.SixMonth.Duration()+kline.ThreeWeek.Duration():
		ct = futures.HalfYearly
	case contractLength <= kline.NineMonth.Duration()+kline.ThreeWeek.Duration():
		ct = futures.NineMonthly
	case contractLength <= kline.OneYear.Duration()+kline.ThreeWeek.Duration():
		ct = futures.Yearly
	default:
		ct = futures.SemiAnnually
	}
	return ct, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}
	switch r.Asset {
	case asset.USDCMarginedFutures,
		asset.USDTMarginedFutures,
		asset.CoinMarginedFutures:

		symbol := ""
		if !r.Pair.IsEmpty() {
			format, err := e.GetPairFormat(r.Asset, true)
			if err != nil {
				return nil, err
			}
			symbol = r.Pair.Format(format).String()
		}
		ticks, err := e.GetTickers(ctx, getCategoryName(r.Asset), symbol, "", time.Time{})
		if err != nil {
			return nil, err
		}

		instrumentInfo, err := e.GetInstrumentInfo(ctx, getCategoryName(r.Asset), symbol, "", "", "", 1000)
		if err != nil {
			return nil, err
		}

		resp := make([]fundingrate.LatestRateResponse, 0, len(ticks.List))
		for i := range ticks.List {
			var cp currency.Pair
			var isEnabled bool
			cp, isEnabled, err = e.MatchSymbolCheckEnabled(ticks.List[i].Symbol, r.Asset, false)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return nil, err
			} else if !isEnabled {
				continue
			}
			var fundingInterval time.Duration
			for j := range instrumentInfo.List {
				if instrumentInfo.List[j].Symbol != ticks.List[i].Symbol {
					continue
				}
				fundingInterval = time.Duration(instrumentInfo.List[j].FundingInterval) * time.Minute
				break
			}
			var lrt time.Time
			if fundingInterval > 0 {
				lrt = ticks.List[i].NextFundingTime.Time().Add(-fundingInterval)
			}
			resp = append(resp, fundingrate.LatestRateResponse{
				Exchange:    e.Name,
				TimeChecked: time.Now(),
				Asset:       r.Asset,
				Pair:        cp,
				LatestRate: fundingrate.Rate{
					Time: lrt,
					Rate: decimal.NewFromFloat(ticks.List[i].FundingRate.Float64()),
				},
				TimeOfNextRate: ticks.List[i].NextFundingTime.Time(),
			})
		}
		if len(resp) == 0 {
			return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("%w %s", asset.ErrNotSupported, r.Asset)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.USDCMarginedFutures &&
			k[i].Asset != asset.USDTMarginedFutures &&
			k[i].Asset != asset.CoinMarginedFutures {
			return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, k[i].Asset)
		}
	}
	if len(k) == 1 {
		formattedPair, err := e.FormatExchangeCurrency(k[0].Pair(), k[0].Asset)
		if err != nil {
			return nil, err
		}
		if _, parseErr := time.Parse(longDatedFormat, k[0].Quote.Symbol); parseErr == nil {
			// long-dated contracts have a delimiter
			formattedPair.Delimiter = currency.DashDelimiter
		}
		pFmt := formattedPair.String()
		var ticks *TickerData
		ticks, err = e.GetTickers(ctx, getCategoryName(k[0].Asset), pFmt, "", time.Time{})
		if err != nil {
			return nil, err
		}
		for i := range ticks.List {
			if ticks.List[i].Symbol != pFmt {
				continue
			}
			return []futures.OpenInterest{{
				Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, k[0].Pair()),
				OpenInterest: ticks.List[i].OpenInterest.Float64(),
			}}, nil
		}
	}
	assets := []asset.Item{asset.USDCMarginedFutures, asset.USDTMarginedFutures, asset.CoinMarginedFutures}
	var resp []futures.OpenInterest
	for i := range assets {
		ticks, err := e.GetTickers(ctx, getCategoryName(assets[i]), "", "", time.Time{})
		if err != nil {
			return nil, err
		}
		for x := range ticks.List {
			var pair currency.Pair
			var isEnabled bool
			// only long-dated contracts have a delimiter
			pair, isEnabled, err = e.MatchSymbolCheckEnabled(ticks.List[x].Symbol, assets[i], strings.Contains(ticks.List[x].Symbol, currency.DashDelimiter))
			if err != nil || !isEnabled {
				continue
			}
			var appendData bool
			for j := range k {
				if k[j].Pair().Equal(pair) {
					appendData = true
					break
				}
			}
			if len(k) > 0 && !appendData {
				continue
			}
			resp = append(resp, futures.OpenInterest{
				Key:          key.NewExchangeAssetPair(e.Name, assets[i], pair),
				OpenInterest: ticks.List[i].OpenInterest.Float64(),
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
	switch a {
	case asset.Spot:
		cp.Delimiter = currency.ForwardSlashDelimiter
		return tradeBaseURL + "en/trade/spot/" + cp.Upper().String(), nil
	case asset.CoinMarginedFutures:
		if cp.Quote.Equal(currency.USD) {
			cp.Delimiter = ""
			return tradeBaseURL + "trade/inverse/" + cp.Upper().String(), nil
		}
		var symbol string
		symbol, err = e.FormatSymbol(cp, a)
		if err != nil {
			return "", err
		}
		// convert long-dated to static contracts
		var io *InstrumentsInfo
		io, err = e.GetInstrumentInfo(ctx, getCategoryName(a), symbol, "", "", "", 1000)
		if err != nil {
			return "", err
		}
		if len(io.List) != 1 {
			return "", fmt.Errorf("%w %v", currency.ErrCurrencyNotFound, cp)
		}
		var length futures.ContractType
		length, err = getContractLength(io.List[0].DeliveryTime.Time().Sub(io.List[0].LaunchTime.Time()))
		if err != nil {
			return "", err
		}
		// bybit inverse long-dated contracts are currently only quarterly or bi-quarterly
		if length == futures.Quarterly {
			cp = currency.NewPair(currency.NewCode(cp.Base.String()+currency.USD.String()), currency.NewCode("Q"))
		} else {
			cp = currency.NewPair(currency.NewCode(cp.Base.String()+currency.USD.String()), currency.NewCode("BIQ"))
		}
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeBaseURL + "trade/inverse/futures/" + cp.Upper().String(), nil
	case asset.USDTMarginedFutures:
		cp.Delimiter = ""
		return tradeBaseURL + "trade/usdt/" + cp.Upper().String(), nil
	case asset.USDCMarginedFutures:
		cp.Delimiter = currency.DashDelimiter
		return tradeBaseURL + "trade/futures/usdc/" + cp.Upper().String(), nil
	case asset.Options:
		return tradeBaseURL + "trade/option/usdc/" + cp.Base.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
