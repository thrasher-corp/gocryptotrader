package bybit

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func (e *Exchange) deriveSubmitOrderArguments(s *order.Submit) (*PlaceOrderRequest, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	symbol, err := e.orderSymbol(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	side := sideBuy
	if s.Side.IsShort() {
		side = sideSell
	}
	orderType, conditional, err := bybitOrderType(s.Type)
	if err != nil {
		return nil, err
	}
	market := orderType == "Market"
	timeInForce, err := bybitTimeInForce(s.TimeInForce, market)
	if err != nil {
		return nil, err
	}
	if conditional && s.TriggerPrice <= 0 {
		return nil, fmt.Errorf("%w for a %s order", errTriggerPriceRequired, s.Type)
	}
	arg := &PlaceOrderRequest{
		Category:    getCategoryName(s.AssetType),
		Symbol:      symbol,
		Side:        side,
		OrderType:   orderType,
		Quantity:    s.GetTradeAmount(e.GetTradingRequirements()), // Spot market buys are sized in the quote currency
		Price:       s.Price,
		OrderLinkID: s.ClientOrderID,
		IsLeverage:  s.AssetType == asset.Spot && s.AutoBorrow,
		ReduceOnly:  s.ReduceOnly,
		TimeInForce: timeInForce,
	}
	if s.TriggerPrice != 0 {
		arg.TriggerPrice = s.TriggerPrice
		arg.TriggerBy = s.TriggerPriceType.String()
		switch s.AssetType {
		case asset.Spot:
			arg.OrderFilter = "tpslOrder"
			// Bybit sizes a spot market buy in the quote currency unless told otherwise, and a stop order's amount is in
			// the base currency
			if conditional && market {
				arg.MarketUnit = "baseCoin"
			}
		case asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
			// Bybit rejects a futures conditional order that does not say which way the price must cross its trigger
			arg.TriggerDirection = 2
			if s.StopDirection == order.StopUp {
				arg.TriggerDirection = 1
			}
		}
	}
	if tp := &s.RiskManagementModes.TakeProfit; tp.Price != 0 {
		arg.TakeProfit = tp.Price
		arg.TPTriggerBy = tp.TriggerPriceType.String()
		if arg.TPOrderType, err = riskOrderType(tp.OrderType); err != nil {
			return nil, fmt.Errorf("take profit: %w", err)
		}
		arg.TPLimitPrice = tp.LimitPrice
	}
	if sl := &s.RiskManagementModes.StopLoss; sl.Price != 0 {
		arg.StopLoss = sl.Price
		arg.SLTriggerBy = sl.TriggerPriceType.String()
		if arg.SLOrderType, err = riskOrderType(sl.OrderType); err != nil {
			return nil, fmt.Errorf("stop loss: %w", err)
		}
		arg.SLLimitPrice = sl.LimitPrice
	}
	// Bybit requires a client order ID on every options order
	if s.AssetType == asset.Options && arg.OrderLinkID == "" {
		arg.OrderLinkID = uuid.NewV7().String()
	}
	// Futures take profit and stop loss orders that rest as limit orders apply to the filled quantity only, which Bybit
	// calls partial mode
	if s.AssetType != asset.Spot && (arg.TPOrderType == "Limit" || arg.SLOrderType == "Limit") {
		arg.TPSLMode = "Partial"
	}
	return arg, nil
}

// riskOrderType returns the take profit or stop loss order type Bybit names, or nothing to leave Bybit's default
func riskOrderType(t order.Type) (string, error) {
	switch t {
	case order.UnknownType:
		return "", nil
	case order.Limit, order.Market:
		return orderTypeToString(t), nil
	}
	return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
}

// bybitOrderType returns the order type Bybit names an order type by, and whether the order is conditional: Bybit
// places a stop order as a market or limit order with a trigger price
func bybitOrderType(t order.Type) (orderType string, conditional bool, err error) {
	switch t {
	case order.Limit:
		return "Limit", false, nil
	case order.Market:
		return "Market", false, nil
	case order.StopLimit:
		return "Limit", true, nil
	case order.Stop, order.StopMarket:
		return "Market", true, nil
	}
	return "", false, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
}

// bybitTimeInForce returns Bybit's name for the time in force of a market order, or of a limit order when market is
// false, or order.ErrUnsupportedTimeInForce for one Bybit cannot honour: Bybit executes every market order as IOC, and
// takes GTC, PostOnly, IOC or FOK for a limit order
func bybitTimeInForce(tif order.TimeInForce, market bool) (string, error) {
	if market {
		if tif != order.UnknownTIF && tif != order.ImmediateOrCancel {
			return "", fmt.Errorf("%w %q for a market order", order.ErrUnsupportedTimeInForce, tif)
		}
		return "IOC", nil
	}
	switch tif {
	case order.UnknownTIF, order.GoodTillCancel:
		return "GTC", nil
	case order.PostOnly, order.GoodTillCancel | order.PostOnly:
		return "PostOnly", nil
	case order.ImmediateOrCancel:
		return "IOC", nil
	case order.FillOrKill:
		return "FOK", nil
	}
	return "", fmt.Errorf("%w %q for a limit order", order.ErrUnsupportedTimeInForce, tif)
}

func (e *Exchange) deriveAmendOrderArguments(action *order.Modify) (*AmendOrderRequest, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	symbol, err := e.orderSymbol(action.Pair, action.AssetType)
	if err != nil {
		return nil, err
	}
	// Trigger types are sent only with the price they apply to, since sending one alone would change it, while a TP/SL
	// limit price can be amended without its trigger price
	arg := &AmendOrderRequest{
		Category:     getCategoryName(action.AssetType),
		Symbol:       symbol,
		OrderID:      action.OrderID,
		OrderLinkID:  action.ClientOrderID,
		Quantity:     action.Amount,
		Price:        action.Price,
		TPLimitPrice: action.RiskManagementModes.TakeProfit.LimitPrice,
		SLLimitPrice: action.RiskManagementModes.StopLoss.LimitPrice,
	}
	if action.TriggerPrice != 0 {
		arg.TriggerPrice = action.TriggerPrice
		arg.TriggerBy = action.TriggerPriceType.String()
	}
	if tp := &action.RiskManagementModes.TakeProfit; tp.Price != 0 {
		takeProfit := tp.Price
		arg.TakeProfit = &takeProfit
		arg.TPTriggerBy = tp.TriggerPriceType.String()
	}
	if sl := &action.RiskManagementModes.StopLoss; sl.Price != 0 {
		stopLoss := sl.Price
		arg.StopLoss = &stopLoss
		arg.SLTriggerBy = sl.TriggerPriceType.String()
	}
	return arg, nil
}

// orderIDRequired checks that a cancellation names its order by order ID or client order ID, either of which Bybit
// takes
func orderIDRequired(ord *order.Cancel) validate.Checker {
	return validate.Check(func() error {
		if ord.OrderID == "" && ord.ClientOrderID == "" {
			return order.ErrOrderIDNotSet
		}
		return nil
	})
}

func (e *Exchange) deriveCancelOrderArguments(ord *order.Cancel) (*CancelOrderRequest, error) {
	if err := ord.Validate(orderIDRequired(ord), ord.PairAssetRequired()); err != nil {
		return nil, err
	}
	symbol, err := e.orderSymbol(ord.Pair, ord.AssetType)
	if err != nil {
		return nil, err
	}
	arg := &CancelOrderRequest{
		Category:    getCategoryName(ord.AssetType),
		Symbol:      symbol,
		OrderID:     ord.OrderID,
		OrderLinkID: ord.ClientOrderID,
	}
	// SubmitOrder places spot stop orders as TP/SL orders, which Bybit looks for only when the cancellation says so
	if ord.AssetType == asset.Spot && ord.Type.Is(order.Stop) {
		arg.OrderFilter = "tpslOrder"
	}
	return arg, nil
}

// orderHistoryWindow is the longest time range Get Order History serves in one request
const orderHistoryWindow = 7 * 24 * time.Hour

// cryptoUnderlying is the option underlying type of crypto coins, as opposed to stocks, commodities, forex and oil,
// which need TradFi agreements
const cryptoUnderlying = 0

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
				WithdrawalHistory:     true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
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
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
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
					kline.OneHour:   true,
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
					kline.IntervalCapacity{Interval: kline.TwelveHour},
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
		exchange.RestSandbox:           bybitDemoURL,
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

	if e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(rateLimits), request.WithRetryPolicy(retryPolicy)); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
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
	e.disableUnusableSubscriptions()

	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
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
	wsOptionsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketOptions)
	if err != nil {
		return err
	}
	wsUSDTLinearURL, err := e.API.Endpoints.GetURL(exchange.WebsocketUSDTMargined)
	if err != nil {
		return err
	}
	wsUSDCLinearURL, err := e.API.Endpoints.GetURL(exchange.WebsocketUSDCMargined)
	if err != nil {
		return err
	}
	wsInverseURL, err := e.API.Endpoints.GetURL(exchange.WebsocketCoinMargined)
	if err != nil {
		return err
	}
	wsTradeURL, err := e.API.Endpoints.GetURL(exchange.WebsocketTrade)
	if err != nil {
		return err
	}
	wsPrivateURL, err := e.API.Endpoints.GetURL(exchange.WebsocketPrivate)
	if err != nil {
		return err
	}
	// The streams below without an endpoint of their own follow the spot stream to testnet
	auxiliaryURLs := make(map[string]string, 5)
	for _, path := range []string{systemStatusPath, advancedEarnPath, spreadPublicPath, rfqPublicPath, eventPublicPath} {
		if auxiliaryURLs[path], err = auxiliaryStreamURL(wsSpotURL, path); err != nil {
			return fmt.Errorf("%s: %w", exchange.WebsocketSpot, err)
		}
	}

	// Every URL has resolved before any connection is registered, so a configuration error registers none
	for _, setup := range []*websocket.ConnectionSetup{
		// Spot - Inbound public data.
		{
			URL:                   wsSpotURL,
			ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
			Connector:             e.WebsocketConnect,
			GenerateSubscriptions: e.generateSubscriptions,
			Subscriber:            e.SpotSubscribe,
			Unsubscriber:          e.SpotUnsubscribe,
			Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
				return e.wsHandleData(ctx, conn, asset.Spot, resp)
			},
			MaxSubscriptionsPerConnection: publicConnectionArgsLimit,
		},
		// Options - Inbound public data. Left uncapped, so that option trade subscriptions sharing a topic stay on one
		// connection
		{
			URL:                   wsOptionsURL,
			ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
			Connector:             e.WebsocketConnect,
			GenerateSubscriptions: e.GenerateOptionsDefaultSubscriptions,
			Subscriber:            e.OptionsSubscribe,
			Unsubscriber:          e.OptionsUnsubscribe,
			Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
				return e.wsHandleData(ctx, conn, asset.Options, resp)
			},
		},
		// Linear - USDT margined futures inbound public data.
		{
			URL:                  wsUSDTLinearURL,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
			Connector:            e.WebsocketConnect,
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
			MaxSubscriptionsPerConnection: publicConnectionArgsLimit,
			MessageFilter:                 asset.USDTMarginedFutures, // Unused but it allows us to differentiate between the two linear futures types.
		},
		// Linear - USDC margined futures inbound public data.
		{
			URL:                  wsUSDCLinearURL,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
			Connector:            e.WebsocketConnect,
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
			MaxSubscriptionsPerConnection: publicConnectionArgsLimit,
			MessageFilter:                 asset.USDCMarginedFutures, // Unused but it allows us to differentiate between the two linear futures types.
		},
		// Inverse - Coin margined futures inbound public data.
		{
			URL:                   wsInverseURL,
			ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
			Connector:             e.WebsocketConnect,
			GenerateSubscriptions: e.GenerateInverseDefaultSubscriptions,
			Subscriber:            e.InverseSubscribe,
			Unsubscriber:          e.InverseUnsubscribe,
			Handler: func(ctx context.Context, conn websocket.Connection, resp []byte) error {
				return e.wsHandleData(ctx, conn, asset.CoinMarginedFutures, resp)
			},
			MaxSubscriptionsPerConnection: publicConnectionArgsLimit,
		},
		// These connections subscribe only to the channels a user configures, none by default, so none of them is made
		// unless asked for
		e.SystemStatusConnectionSetup(auxiliaryURLs[systemStatusPath]),
		e.AdvancedEarnConnectionSetup(exch, auxiliaryURLs[advancedEarnPath]),
		e.SpreadPublicConnectionSetup(exch, auxiliaryURLs[spreadPublicPath]),
		e.RFQPublicConnectionSetup(exch, auxiliaryURLs[rfqPublicPath]),
		e.EventPublicConnectionSetup(exch, auxiliaryURLs[eventPublicPath]),
		// Trade - Dedicated trade connection for all outbound trading requests.
		{
			URL:                  wsTradeURL,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
			Connector:            e.WebsocketConnect,
			Handler: func(_ context.Context, conn websocket.Connection, resp []byte) error {
				return e.wsHandleTradeData(conn, resp)
			},
			Authenticate:             e.WebsocketAuthenticateTradeConnection,
			MessageFilter:            OutboundTradeConnection,
			SubscriptionsNotRequired: true,
		},
		// Private - Inbound private data connection for authenticated data
		{
			URL:                   wsPrivateURL,
			ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
			Authenticated:         true,
			Connector:             e.WebsocketConnect,
			GenerateSubscriptions: e.generateAuthSubscriptions,
			Subscriber:            e.authSubscribe,
			Unsubscriber:          e.authUnsubscribe,
			Handler:               e.wsHandleAuthenticatedData,
			Authenticate:          e.WebsocketAuthenticatePrivateConnection,
			MessageFilter:         InboundPrivateConnection,
		},
	} {
		if err := e.Websocket.SetupNewConnection(setup); err != nil {
			return err
		}
	}
	return nil
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
		allPairs []InstrumentInfo
		response *InstrumentsInfoResponse
	)
	var nextPageCursor string
	switch a {
	case asset.Spot, asset.CoinMarginedFutures, asset.USDCMarginedFutures, asset.USDTMarginedFutures:
		category = getCategoryName(a)
		for {
			response, err = e.GetInstrumentInfo(ctx, &InstrumentInfoRequest{Category: category, Status: tradingStatus, Cursor: nextPageCursor, Limit: 1000})
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
		baseCoins, err := e.optionBaseCoins(ctx)
		if err != nil {
			return nil, err
		}
		for _, baseCoin := range baseCoins {
			nextPageCursor = ""
			for {
				response, err = e.GetInstrumentInfo(ctx, &InstrumentInfoRequest{Category: category, Status: tradingStatus, BaseCoin: baseCoin.String(), Cursor: nextPageCursor, Limit: 1000})
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
		if allPairs[x].Status != tradingStatus || (filterSymbol != "" && allPairs[x].QuoteCoin.String() != filterSymbol) {
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

// optionBaseCoins returns the base coins that Bybit lists crypto options of, which option instruments are requested by
func (e *Exchange) optionBaseCoins(ctx context.Context) ([]currency.Code, error) {
	resp, err := e.GetOptionBaseCoins(ctx, []uint64{cryptoUnderlying})
	if err != nil {
		return nil, err
	}
	coins := make([]currency.Code, 0, len(resp.List))
	for i := range resp.List {
		if resp.List[i].HasSymbol {
			coins = append(coins, resp.List[i].BaseCoin)
		}
	}
	return coins, nil
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

// baseQuoteVolumes maps a ticker's or candle's volume and turnover onto its base and quote volumes. Bybit reports
// volume in the base currency and turnover in the quote, except on the inverse category, where a contract is worth one
// unit of the quote currency and the two swap over
func baseQuoteVolumes(volume, turnover types.Number, a asset.Item) (baseVolume, quoteVolume float64) {
	if a == asset.CoinMarginedFutures {
		return turnover.Float64(), volume.Float64()
	}
	return volume.Float64(), turnover.Float64()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	enabled, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	var ticks *TickersResponse
	switch assetType {
	case asset.Spot, asset.USDCMarginedFutures,
		asset.USDTMarginedFutures,
		asset.CoinMarginedFutures:
		ticks, err = e.GetTickers(ctx, &TickersRequest{Category: getCategoryName(assetType)})
		if err != nil {
			return err
		}
		for x := range ticks.List {
			var pair currency.Pair
			pair, err = e.matchSymbol(ticks.List[x].Symbol, assetType)
			if err != nil {
				continue
			}
			// Take the pair from the enabled list, so the stored ticker reports it in the configured format
			i := slices.IndexFunc(enabled, pair.Equal)
			if i == -1 {
				continue
			}
			pair = enabled[i]
			baseVolume, quoteVolume := baseQuoteVolumes(ticks.List[x].Volume24Hour, ticks.List[x].Turnover24Hour, assetType)
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks.List[x].LastPrice.Float64(),
				High:         ticks.List[x].HighPrice24Hour.Float64(),
				Low:          ticks.List[x].LowPrice24Hour.Float64(),
				Bid:          ticks.List[x].Bid1Price.Float64(),
				BidSize:      ticks.List[x].Bid1Size.Float64(),
				Ask:          ticks.List[x].Ask1Price.Float64(),
				AskSize:      ticks.List[x].Ask1Size.Float64(),
				BaseVolume:   baseVolume,
				QuoteVolume:  quoteVolume,
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	case asset.Options:
		// Option tickers are requested by base coin, so only the base coins of enabled pairs are needed
		var baseCoins []currency.Code
		for _, p := range enabled {
			if !slices.ContainsFunc(baseCoins, p.Base.Equal) {
				baseCoins = append(baseCoins, p.Base)
			}
		}
		for _, baseCoin := range baseCoins {
			ticks, err = e.GetTickers(ctx, &TickersRequest{Category: getCategoryName(assetType), BaseCoin: baseCoin})
			if err != nil {
				return err
			}
			for x := range ticks.List {
				var pair currency.Pair
				pair, err = e.matchSymbol(ticks.List[x].Symbol, assetType)
				if err != nil {
					continue
				}
				i := slices.IndexFunc(enabled, pair.Equal)
				if i == -1 {
					continue
				}
				pair = enabled[i]
				baseVolume, quoteVolume := baseQuoteVolumes(ticks.List[x].Volume24Hour, ticks.List[x].Turnover24Hour, assetType)
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         ticks.List[x].LastPrice.Float64(),
					High:         ticks.List[x].HighPrice24Hour.Float64(),
					Low:          ticks.List[x].LowPrice24Hour.Float64(),
					Bid:          ticks.List[x].Bid1Price.Float64(),
					BidSize:      ticks.List[x].Bid1Size.Float64(),
					Ask:          ticks.List[x].Ask1Price.Float64(),
					AskSize:      ticks.List[x].Ask1Size.Float64(),
					BaseVolume:   baseVolume,
					QuoteVolume:  quoteVolume,
					Pair:         pair,
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
	var orderbookNew *OrderbookResponse
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
		orderbookNew, err = e.GetOrderBook(ctx, getCategoryName(assetType), p.String(), orderbookDepth(assetType))
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
		LastUpdated:       orderbookNew.Timestamp.Time(),
		LastUpdateID:      int64(orderbookNew.UpdateID), //nolint:gosec // Update IDs stay far below the int64 limit
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

// orderbookDepth returns the deepest orderbook Get Orderbook serves for an asset; without a limit Bybit returns one
// level of spot and option books
func orderbookDepth(a asset.Item) uint64 {
	if a == asset.Options {
		return 25
	}
	return 1000
}

// UpdateAccountBalances returns the balances of the unified trading account, which every Bybit account has been since
// November 2025 and which holds the funds of every asset type. Whichever asset type is asked for, it is stored once, as
// the spot sub-account the wallet stream updates, and as of when it was requested, so a wallet push that arrives while
// the request is in flight is kept
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if getCategoryName(assetType) == "" {
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	requested := time.Now()
	resp, err := e.GetWalletBalance(ctx, &WalletBalanceRequest{AccountType: "UNIFIED"})
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, "")}
	for i := range resp.List {
		for j := range resp.List[i].Coin {
			c := &resp.List[i].Coin[j]
			subAccts[0].Balances.Set(c.Coin, unifiedBalance(c))
		}
	}
	return subAccts, e.Accounts.SaveSnapshot(ctx, subAccts, requested)
}

// GetCachedSubAccounts returns the stored unified trading account, which holds the funds of every asset type
func (e *Exchange) GetCachedSubAccounts(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if assetType != asset.All && getCategoryName(assetType) == "" {
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	return e.Base.GetCachedSubAccounts(ctx, asset.Spot)
}

// GetCachedCurrencyBalances returns the stored balances of the unified trading account, which holds the funds of every
// asset type
func (e *Exchange) GetCachedCurrencyBalances(ctx context.Context, assetType asset.Item) (accounts.CurrencyBalances, error) {
	if assetType != asset.All && getCategoryName(assetType) == "" {
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	return e.Base.GetCachedCurrencyBalances(ctx, asset.Spot)
}

// unifiedBalance returns a coin's balance in the unified trading account. Its wallet balance includes what spot margin
// trading and manual borrowing borrowed, and is negative while derivatives losses are owed. Spot open orders lock part
// of it and the initial margin of the coin's derivatives orders and positions holds more, as Bybit's isolated margin
// available balance counts; in portfolio margin mode Bybit leaves those margins empty, so only the spot locks are held.
// Bybit no longer reports what can be used without borrowing, so that is what the wallet holds beyond its loans, what
// is held and the bonus, which cannot be withdrawn
func unifiedBalance(c *WalletBalanceCoin) accounts.Balance {
	walletBalance := c.WalletBalance.Float64()
	hold := c.Locked.Float64() + c.TotalOrderInitialMargin.Float64() + c.TotalPositionInitialMargin.Float64()
	return accounts.Balance{
		Total:                  walletBalance,
		Hold:                   hold,
		Free:                   max(walletBalance-hold, 0),
		Borrowed:               c.BorrowAmount.Float64(),
		AvailableWithoutBorrow: max(walletBalance-c.SpotBorrow.Float64()-hold-c.Bonus.Float64(), 0),
	}
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns the account's on-chain and off-chain withdrawals of a coin, a page at a time
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if getCategoryName(a) == "" {
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	var withdrawHistory []exchange.WithdrawalHistory
	req := &WithdrawalRecordsRequest{Coin: c, WithdrawType: 2}
	for {
		withdrawals, err := e.GetWithdrawalRecords(ctx, req)
		if err != nil {
			return nil, err
		}
		for i := range withdrawals.Rows {
			withdrawHistory = append(withdrawHistory, exchange.WithdrawalHistory{
				TransferID:      withdrawals.Rows[i].WithdrawID,
				Status:          withdrawals.Rows[i].Status,
				Currency:        withdrawals.Rows[i].Coin.String(),
				Amount:          withdrawals.Rows[i].Amount.Float64(),
				Fee:             withdrawals.Rows[i].WithdrawFee.Float64(),
				CryptoToAddress: withdrawals.Rows[i].ToAddress,
				CryptoTxID:      withdrawals.Rows[i].TransactionID,
				CryptoChain:     withdrawals.Rows[i].Chain,
				Timestamp:       withdrawals.Rows[i].UpdateTime.Time(),
			})
		}
		if withdrawals.NextPageCursor == "" || withdrawals.NextPageCursor == req.Cursor || len(withdrawals.Rows) == 0 {
			return withdrawHistory, nil
		}
		req.Cursor = withdrawals.NextPageCursor
	}
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	formattedPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	limit := uint64(500)
	if assetType == asset.Spot {
		limit = 60
	}
	var tradeData *PublicTradingHistoryResponse
	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		if assetType == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
			formattedPair.Delimiter = currency.DashDelimiter
		}
		tradeData, err = e.GetPublicTradingHistory(ctx, &PublicTradingHistoryRequest{Category: getCategoryName(assetType), Symbol: formattedPair.String(), Limit: limit})
	case asset.Options:
		tradeData, err = e.GetPublicTradingHistory(ctx, &PublicTradingHistoryRequest{Category: getCategoryName(assetType), Symbol: formattedPair.String(), BaseCoin: formattedPair.Base, Limit: limit})
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
			Timestamp:    tradeData.List[i].Time.Time(),
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

	trade.SortByDate(resp)
	return resp, nil
}

// GetHistoricTrades returns the trades within the timeframe provided. Bybit serves only the most recent trades, up to
// 1000 (60 for spot), so older trades in the timeframe are not returned
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	limit := uint64(1000)
	if assetType == asset.Spot {
		limit = 60
	}
	var tradeHistoryResponse *PublicTradingHistoryResponse
	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		if assetType == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
			p.Delimiter = currency.DashDelimiter
		}
		tradeHistoryResponse, err = e.GetPublicTradingHistory(ctx, &PublicTradingHistoryRequest{Category: getCategoryName(assetType), Symbol: p.String(), Limit: limit})
		if err != nil {
			return nil, err
		}
	case asset.Options:
		tradeHistoryResponse, err = e.GetPublicTradingHistory(ctx, &PublicTradingHistoryRequest{Category: getCategoryName(assetType), Symbol: p.String(), BaseCoin: p.Base, Limit: limit})
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
			Timestamp:    tradeHistoryResponse.List[x].Time.Time(),
		}
	}
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
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
	resp.Type = submittedOrderType(s)
	resp.ClientOrderID = arg.OrderLinkID
	resp.Status = order.New
	return resp, nil
}

// submittedOrderType returns the type a submitted order reads back as. A limit or market order with a trigger price is
// placed as a conditional order, which Bybit reports with its condition, so storing it as such keeps the stored order
// in step with later updates and lets its cancellation carry the TP/SL filter a spot conditional order needs
func submittedOrderType(s *order.Submit) order.Type {
	switch {
	case s.Type == order.Stop:
		// A stop order is placed as a conditional market order, which Bybit reports as a stop market order
		return order.StopMarket
	case s.TriggerPrice != 0 && (s.Type == order.Limit || s.Type == order.Market):
		return order.Stop | s.Type
	}
	return s.Type
}

// WebsocketSubmitOrder submits a new order over the websocket trade connection. It returns once Bybit accepts the order,
// as SubmitOrder does; the order's status, fills and fees follow on the private stream
func (e *Exchange) WebsocketSubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := common.NilGuard(s); err != nil {
		return nil, err
	}
	if err := websocketOrderEntrySupported(s.AssetType); err != nil {
		return nil, err
	}
	arg, err := e.deriveSubmitOrderArguments(s)
	if err != nil {
		return nil, err
	}
	ack, err := e.WebsocketCreateOrder(ctx, arg)
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(ack.OrderID)
	if err != nil {
		return nil, err
	}
	resp.Type = submittedOrderType(s)
	resp.ClientOrderID = arg.OrderLinkID
	resp.Status = order.New
	return resp, nil
}

// websocketOrderEntrySupported returns common.ErrFunctionNotSupported for an asset Bybit's websocket order entry does
// not trade, so nothing is sent for it
func websocketOrderEntrySupported(a asset.Item) error {
	if getCategoryName(a) == "" {
		return fmt.Errorf("%w: websocket order entry for %s", common.ErrFunctionNotSupported, a)
	}
	return nil
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
	if resp != nil {
		resp.OrderID = result.OrderID
	}
	return resp, err
}

// WebsocketModifyOrder amends an existing order over the websocket trade connection. It returns once Bybit accepts the
// amendment, as ModifyOrder does
func (e *Exchange) WebsocketModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := common.NilGuard(action); err != nil {
		return nil, err
	}
	if err := websocketOrderEntrySupported(action.AssetType); err != nil {
		return nil, err
	}
	arg, err := e.deriveAmendOrderArguments(action)
	if err != nil {
		return nil, err
	}
	ack, err := e.WebsocketAmendOrder(ctx, arg)
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if resp != nil {
		resp.OrderID = ack.OrderID
	}
	return resp, err
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	arg, err := e.deriveCancelOrderArguments(ord)
	if err != nil {
		return err
	}
	return cancelWithStopOrderFallback(ctx, arg, e.CancelTradeOrder)
}

// WebsocketCancelOrder cancels an order over the websocket trade connection. It returns once Bybit accepts the
// cancellation, as CancelOrder does
func (e *Exchange) WebsocketCancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := common.NilGuard(ord); err != nil {
		return err
	}
	if err := websocketOrderEntrySupported(ord.AssetType); err != nil {
		return err
	}
	arg, err := e.deriveCancelOrderArguments(ord)
	if err != nil {
		return err
	}
	return cancelWithStopOrderFallback(ctx, arg, e.WebsocketCancelTradeOrder)
}

// cancelWithStopOrderFallback sends a cancellation, and sends it again with the StopOrder filter when Bybit finds no
// such TP/SL order, since a spot conditional order reads back as a stop order as a TP/SL order does
func cancelWithStopOrderFallback(ctx context.Context, arg *CancelOrderRequest, send func(context.Context, *CancelOrderRequest) (*OrderResponse, error)) error {
	_, err := send(ctx, arg)
	if errors.Is(err, order.ErrOrderNotFound) && arg.OrderFilter == "tpslOrder" {
		arg.OrderFilter = "StopOrder"
		_, err = send(ctx, arg)
	}
	return err
}

// CancelBatchOrders cancels orders by order ID or client order ID, in as few requests as Bybit's batch limits allow.
// Each order's status is "success" when Bybit accepted its cancellation, and Bybit's reason otherwise
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	// Bybit takes one category per request
	type batch struct {
		category string
		keys     []string
		items    []BatchCancelOrderItem
	}
	var batches []*batch
	byCategory := make(map[string]*batch)
	for i := range o {
		if err := o[i].Validate(orderIDRequired(&o[i]), o[i].PairAssetRequired()); err != nil {
			return nil, err
		}
		category := getCategoryName(o[i].AssetType)
		if category == "" {
			return nil, fmt.Errorf("%s %w", o[i].AssetType, asset.ErrNotSupported)
		}
		symbol, err := e.orderSymbol(o[i].Pair, o[i].AssetType)
		if err != nil {
			return nil, err
		}
		b, ok := byCategory[category]
		if !ok {
			b = &batch{category: category}
			byCategory[category] = b
			batches = append(batches, b)
		}
		id := o[i].OrderID
		if id == "" {
			id = o[i].ClientOrderID
		}
		b.keys = append(b.keys, id)
		b.items = append(b.items, BatchCancelOrderItem{Symbol: symbol, OrderID: o[i].OrderID, OrderLinkID: o[i].ClientOrderID})
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(o))}
	for _, b := range batches {
		size := batchOrderSize(b.category)
		for start := 0; start < len(b.items); start += size {
			end := min(start+size, len(b.items))
			result, err := e.CancelBatchOrder(ctx, &BatchCancelOrderRequest{Category: b.category, Request: b.items[start:end]})
			if err != nil {
				return resp, err
			}
			// Bybit reports each order's outcome in request order; a rejected order may come back without its IDs
			for i := range result.List {
				status := "success"
				if result.List[i].Code != 0 {
					status = result.List[i].Message
				}
				resp.Status[b.keys[start+i]] = status
			}
		}
	}
	return resp, nil
}

// batchOrderSize returns how many orders a batch order request of a category may hold: Bybit takes 5 options or 10 spot
// orders in one, and 20 linear or inverse orders, but counts those against a budget of 10 a second and fails the rest
func batchOrderSize(category string) int {
	if category == cOption {
		return 5
	}
	return 10
}

// spotOrderFilters are the kinds of spot order, which Cancel All Orders cancels one kind at a time
var spotOrderFilters = []string{"Order", "tpslOrder", "StopOrder", "OcoOrder", "BidirectionalTpslOrder"}

// CancelAllOrders cancels every open order of an asset, or of one pair when the cancellation names one. Coin margined
// futures need a pair, since Bybit cancels them by symbol, base coin or settle coin
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	category := getCategoryName(orderCancellation.AssetType)
	if category == "" {
		return order.CancelAllResponse{}, fmt.Errorf("%s %w", orderCancellation.AssetType, asset.ErrNotSupported)
	}
	req := &CancelAllOrdersRequest{Category: category}
	if !orderCancellation.Pair.IsEmpty() {
		symbol, err := e.orderSymbol(orderCancellation.Pair, orderCancellation.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		req.Symbol = symbol
	} else {
		switch orderCancellation.AssetType {
		case asset.CoinMarginedFutures:
			return order.CancelAllResponse{}, currency.ErrCurrencyPairEmpty
		case asset.USDTMarginedFutures, asset.USDCMarginedFutures:
			req.SettleCoin = orderSettleCoin(orderCancellation.AssetType)
		}
	}
	filters := []string{""}
	if orderCancellation.AssetType == asset.Spot {
		filters = spotOrderFilters
	}
	resp := order.CancelAllResponse{Status: make(map[string]string)}
	for _, filter := range filters {
		req.OrderFilter = filter
		for attempt := 1; ; attempt++ {
			cancelled, err := e.CancelAllTradeOrders(ctx, req)
			if err != nil {
				return resp, err
			}
			if !cancelled.Success {
				return resp, fmt.Errorf("%w: %s %s", errCancelAllOrdersFailed, category, req.OrderFilter)
			}
			var newlyCancelled int
			for i := range cancelled.List {
				if _, ok := resp.Status[cancelled.List[i].OrderID]; !ok {
					newlyCancelled++
				}
				resp.Status[cancelled.List[i].OrderID] = "success"
			}
			// Bybit cancels at most 500 futures orders a request, picked at random, so a full futures reply can leave
			// orders open; spot and option replies are never capped
			if len(cancelled.List) < cancelAllLimit || (category != cLinear && category != cInverse) {
				break
			}
			if newlyCancelled == 0 || attempt == cancelAllMaxRequests {
				return resp, fmt.Errorf("%w: %s %s orders may remain open after %d requests", errCancelAllOrdersFailed, category, req.OrderFilter, attempt)
			}
		}
	}
	return resp, nil
}

const (
	// cancelAllLimit is the most linear or inverse orders Cancel All Orders cancels in one request
	cancelAllLimit = 500
	// cancelAllMaxRequests bounds the requests CancelAllOrders sends for one kind of order while Bybit keeps returning
	// full replies
	cancelAllMaxRequests = 20
)

// GetOrderInfo returns order information based on order ID. Open orders and the most recent 500 closed ones per
// category come from Get Open & Closed Orders, and older orders from Get Order History
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	category := getCategoryName(assetType)
	if category == "" {
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	symbol, err := e.orderSymbol(pair, assetType)
	if err != nil {
		return nil, err
	}
	open, err := e.GetOpenOrders(ctx, &OpenOrdersRequest{Category: category, Symbol: symbol, OrderID: orderID})
	if err != nil {
		return nil, err
	}
	if len(open.List) != 0 {
		return e.orderDetail(&open.List[0], assetType, requestedPair(&open.List[0], symbol, pair))
	}
	history, err := e.GetTradeOrderHistory(ctx, &OrderHistoryRequest{Category: category, Symbol: symbol, OrderID: orderID})
	if err != nil {
		return nil, err
	}
	if len(history.List) == 0 {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
	}
	return e.orderDetail(&history.List[0].OpenOrder, assetType, requestedPair(&history.List[0].OpenOrder, symbol, pair))
}

// requestedPair returns the pair an order was requested by when Bybit returned an order of that pair's symbol, and no
// pair otherwise, since Get Open & Closed Orders ranks an order ID ahead of the symbol
func requestedPair(o *OpenOrder, symbol string, pair currency.Pair) currency.Pair {
	if o.Symbol != symbol {
		return currency.EMPTYPAIR
	}
	return pair
}

// orderSymbol returns the symbol Bybit names a pair's orders by
func (e *Exchange) orderSymbol(pair currency.Pair, a asset.Item) (string, error) {
	pair, err := e.FormatExchangeCurrency(pair, a)
	if err != nil {
		return "", err
	}
	if a == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
		pair.Delimiter = currency.DashDelimiter
	}
	return pair.String(), nil
}

// orderSettleCoin returns the settle coin that selects an asset's orders within its category: linear orders need
// one when no symbol is given, and USDT and USDC margined futures share the linear category
func orderSettleCoin(a asset.Item) currency.Code {
	switch a {
	case asset.USDTMarginedFutures:
		return currency.USDT
	case asset.USDCMarginedFutures:
		return currency.USDC
	}
	return currency.EMPTYCODE
}

// matchSymbol returns the available pair that a Bybit symbol names. USDC futures and option symbols carry a dash
// between base and quote that the stored pairs' format lacks, while dated USDT futures keep theirs (BTCUSDT-16OCT26)
func (e *Exchange) matchSymbol(symbol string, a asset.Item) (currency.Pair, error) {
	return e.MatchSymbolWithAvailablePairs(symbol, a, hasPotentialDelimiter(a))
}

// orderDetail converts a Bybit order into an order detail. known is the order's pair when the caller already knows it,
// which an order keeps once its contract has expired or been delisted and its symbol is no longer an available pair
func (e *Exchange) orderDetail(o *OpenOrder, a asset.Item, known currency.Pair) (*order.Detail, error) {
	pair, err := e.matchSymbol(o.Symbol, a)
	if err != nil {
		// The linear category returns USDT and USDC margined orders alike. A USDT margined symbol carries USDT after its
		// base coin (BTCUSDT, BTCUSDT-16OCT26) and a USDC margined one does not (BTCPERP, BTC-16OCT26), so the known pair
		// of one asset is not taken for the other's order
		otherMargin := (a == asset.USDTMarginedFutures || a == asset.USDCMarginedFutures) &&
			strings.Contains(o.Symbol, currency.USDT.String()) != (a == asset.USDTMarginedFutures)
		if known.IsEmpty() || otherMargin || !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		pair = known
	}
	orderType, err := orderTypeFromString(o.OrderType)
	if err != nil {
		return nil, err
	}
	orderType = conditionalOrderType(orderType, o.StopOrderType)
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return nil, err
	}
	tif, err := timeInForceFromString(o.TimeInForce)
	if err != nil {
		return nil, err
	}
	settle, err := orderCostCurrency(a, pair, o.Symbol)
	if err != nil {
		return nil, err
	}
	fees, err := orderFees(a, settle, o.CumulativeFeeDetail, o.CumulativeExecutedFee)
	if err != nil {
		return nil, err
	}
	d := &order.Detail{
		Exchange:             e.Name,
		OrderID:              o.OrderID,
		ClientOrderID:        o.OrderLinkID,
		Pair:                 pair,
		AssetType:            a,
		Type:                 orderType,
		Side:                 side,
		Status:               StringToOrderStatus(o.OrderStatus),
		TimeInForce:          tif,
		ReduceOnly:           o.ReduceOnly,
		Price:                o.Price.Float64(),
		Amount:               o.Quantity.Float64(),
		TriggerPrice:         o.TriggerPrice.Float64(),
		AverageExecutedPrice: o.AveragePrice.Float64(),
		ExecutedAmount:       o.CumulativeExecutedQuantity.Float64(),
		RemainingAmount:      o.LeavesQuantity.Float64(),
		Cost:                 o.CumulativeExecutedValue.Float64(),
		CostAsset:            settle,
		Date:                 o.CreatedTime.Time(),
		LastUpdated:          o.UpdatedTime.Time(),
	}
	if a != asset.Spot {
		d.SettlementCurrency = settle
	}
	// A spot market order sized by value holds a quote amount in qty, and the base amount it bought in cumExecQty
	if quoteSized(a, o.OrderType, o.Side, o.MarketUnit) {
		d.QuoteAmount, d.Amount = d.Amount, d.ExecutedAmount
	}
	d.SetFees(fees)
	return d, nil
}

// orderTypeFromString returns the order type of a Bybit order, REST or pushed
func orderTypeFromString(s string) (order.Type, error) {
	switch s {
	case "UNKNOWN", "Block trade Roll Back":
		// Bybit documents these as order types it only returns, for funding and for a block trade's roll back, with no
		// GoCryptoTrader counterpart
		return order.UnknownType, nil
	case "Block trade-Limit":
		return order.Limit, nil
	}
	return order.StringToOrderType(s)
}

// quoteSized reports whether an order's qty is a quote amount, which only a spot market order's can be. Bybit sizes a
// market buy by value unless its marketUnit is baseCoin, and reports an empty marketUnit for some buys sized that way,
// such as the legs of a bidirectional TP/SL order
func quoteSized(a asset.Item, orderType, side, marketUnit string) bool {
	if a != asset.Spot || orderType != "Market" {
		return false
	}
	switch marketUnit {
	case "quoteCoin":
		return true
	case "baseCoin":
		return false
	}
	return side == sideBuy
}

// conditionalOrderType returns the type of a conditional order, which Bybit reports as the type it takes once
// triggered, with the condition in stopOrderType
func conditionalOrderType(orderType order.Type, stopOrderType string) order.Type {
	switch stopOrderType {
	case "Stop", "StopLoss", "PartialStopLoss", "tpslOrder", "MmRateClose":
		return order.Stop | orderType
	case "TakeProfit", "PartialTakeProfit":
		if orderType == order.Market {
			return order.TakeProfitMarket
		}
		return order.TakeProfit
	case "TrailingStop":
		return order.TrailingStop
	case "OcoOrder", "BidirectionalTpslOrder":
		// Each record is one leg of a pair whose legs cancel each other when one triggers
		return order.OCO
	}
	return orderType
}

// orderCostCurrency returns the currency an order's executed value and fees are counted in: the quote currency for
// spot, and the settle coin for derivatives
func orderCostCurrency(a asset.Item, pair currency.Pair, symbol string) (currency.Code, error) {
	if a == asset.Spot {
		return pair.Quote, nil
	}
	return positionCollateralCurrency(a, pair, symbol)
}

// orderFees returns an order's cumulative fees. Linear and spot orders report them per currency in cumFeeDetail,
// which is absent from orders that predate it and empty when nothing was charged; inverse and option orders report
// them in cumExecFee, in the settle coin
func orderFees(a asset.Item, settle currency.Code, detail map[currency.Code]types.Number, cumExecFee types.Number) (order.Fees, error) {
	if a == asset.CoinMarginedFutures || a == asset.Options {
		return order.NewFees(order.FeeAmount{Currency: settle, Amount: cumExecFee.Float64()})
	}
	if detail == nil {
		return order.Fees{}, nil
	}
	amounts := make([]order.FeeAmount, 0, len(detail))
	for c, amount := range detail {
		amounts = append(amounts, order.FeeAmount{Currency: c, Amount: amount.Float64()})
	}
	return order.NewFees(amounts...)
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
		if info.Rows[x].Coin.Equal(cryptocurrency) {
			for i := range info.Rows[x].Chains {
				if info.Rows[x].Chains[i].ChainDeposit == "1" || info.Rows[x].Chains[i].ChainWithdraw == "1" {
					availableChains = append(availableChains, info.Rows[x].Chains[i].Chain)
				}
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
	resp, err := e.WithdrawCurrency(ctx,
		&WithdrawalRequest{
			Coin:      withdrawRequest.Currency,
			Chain:     withdrawRequest.Crypto.Chain,
			Address:   withdrawRequest.Crypto.Address,
			Tag:       withdrawRequest.Crypto.AddressTag,
			Amount:    withdrawRequest.Amount,
			Timestamp: time.Now(),
			// Bybit requires the wallet to withdraw from; deposits arrive in the funding wallet
			AccountType: "FUND",
		})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.ID,
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

// GetActiveOrders returns the open orders of an asset, those of req.Pairs when it lists any
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	category := getCategoryName(req.AssetType)
	if category == "" {
		return nil, fmt.Errorf("%s %w", req.AssetType, asset.ErrNotSupported)
	}
	var orders []order.Detail
	var cursor string
	for {
		resp, err := e.GetOpenOrders(ctx, &OpenOrdersRequest{Category: category, SettleCoin: orderSettleCoin(req.AssetType), OrderID: req.FromOrderID, Limit: 50, Cursor: cursor})
		if err != nil {
			return nil, err
		}
		details, err := e.ConstructOrderDetails(resp.List, req.AssetType, currency.EMPTYPAIR, req.Pairs)
		if err != nil {
			return nil, err
		}
		orders = append(orders, details...)
		if resp.NextPageCursor == "" || resp.NextPageCursor == cursor || len(resp.List) == 0 {
			break
		}
		cursor = resp.NextPageCursor
	}
	return req.Filter(e.Name, orders), nil
}

// ConstructOrderDetails converts Bybit orders into order details, keeping those of pair, or those of filterPairs when
// pair is empty and filterPairs lists any. Orders for symbols that are not available pairs of the asset are skipped
func (e *Exchange) ConstructOrderDetails(tradeOrders []OpenOrder, assetType asset.Item, pair currency.Pair, filterPairs currency.Pairs) (order.FilteredOrders, error) {
	orders := make([]order.Detail, 0, len(tradeOrders))
	for x := range tradeOrders {
		d, err := e.orderDetail(&tradeOrders[x], assetType, currency.EMPTYPAIR)
		if err != nil {
			if errors.Is(err, currency.ErrPairNotFound) {
				continue
			}
			return nil, err
		}
		if (!pair.IsEmpty() && !pair.Equal(d.Pair)) || (pair.IsEmpty() && len(filterPairs) > 0 && !filterPairs.Contains(d.Pair, true)) {
			continue
		}
		orders = append(orders, *d)
	}
	return orders, nil
}

// GetOrderHistory returns an asset's closed orders, those of req.Pairs when it lists any. Bybit returns at most 7 days
// of orders per request, so a longer time range is fetched a week at a time, and a start time alone, or a range that
// ends in the future, is fetched up to now; an end time alone returns the 7 days before it, and no time range the last
// 7 days
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	category := getCategoryName(req.AssetType)
	if category == "" {
		return nil, fmt.Errorf("%s %w", req.AssetType, asset.ErrNotSupported)
	}
	// An order of a requested pair keeps that pair once its contract has expired or been delisted, as GetOrderInfo's
	// does, unless unequal requested pairs share its symbol. The empty pair, which asks for every pair, names no symbol
	requested := make(map[string]currency.Pair, len(req.Pairs))
	for _, p := range req.Pairs {
		symbol, err := e.orderSymbol(p, req.AssetType)
		if err != nil {
			continue
		}
		if known, ok := requested[symbol]; ok && !known.Equal(p) {
			p = currency.EMPTYPAIR
		}
		requested[symbol] = p
	}
	windows := [][2]time.Time{{req.StartTime, req.EndTime}}
	if !req.StartTime.IsZero() {
		// No order is created after now, so a range ending later is fetched up to now
		endTime := req.EndTime
		if now := time.Now(); endTime.IsZero() || endTime.After(now) {
			endTime = now
		}
		windows = windows[:0]
		for start := req.StartTime; start.Before(endTime); start = start.Add(orderHistoryWindow) {
			end := start.Add(orderHistoryWindow)
			if end.After(endTime) {
				end = endTime
			}
			windows = append(windows, [2]time.Time{start, end})
		}
	}
	var orders []order.Detail
	for _, w := range windows {
		var cursor string
		for {
			resp, err := e.GetTradeOrderHistory(ctx, &OrderHistoryRequest{Category: category, SettleCoin: orderSettleCoin(req.AssetType), OrderID: req.FromOrderID, StartTime: w[0], EndTime: w[1], Limit: 50, Cursor: cursor})
			if err != nil {
				return nil, err
			}
			for i := range resp.List {
				d, err := e.orderDetail(&resp.List[i].OpenOrder, req.AssetType, requested[resp.List[i].Symbol])
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
				orders = append(orders, *d)
			}
			if resp.NextPageCursor == "" || resp.NextPageCursor == cursor || len(resp.List) == 0 {
				break
			}
			cursor = resp.NextPageCursor
		}
	}
	order.FilterOrdersByPairs(&orders, req.Pairs)
	return req.Filter(e.Name, orders), nil
}

// GetFeeByType returns an estimate of the fee of a trade of feeBuilder's pair: the account's fee rate when credentials
// are set, and a worst case rate otherwise. A pair that several assets trade takes the fee rate of the first of spot,
// USDT margined, USDC margined and coin margined futures, and options that lists it
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if err := common.NilGuard(feeBuilder); err != nil {
		return 0, err
	}
	if feeBuilder.Pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		price := feeBuilder.PurchasePrice
		if a, err := e.feeAsset(feeBuilder.Pair); err == nil {
			switch a {
			case asset.CoinMarginedFutures:
				// An inverse contract is worth one unit of the quote currency, so a trade's value is its number of
				// contracts
				price = 1
			case asset.Options:
				// Bybit charges an option trade up to 7% of the option's price, more than the usual worst case
				return maxOptionFeeCap * price * feeBuilder.Amount, nil
			}
		}
		return getOfflineTradeFee(price, feeBuilder.Amount), nil
	case exchange.CryptocurrencyTradeFee:
		a, err := e.feeAsset(feeBuilder.Pair)
		if err != nil {
			return 0, err
		}
		req := &FeeRateRequest{Category: getCategoryName(a)}
		if a == asset.Options {
			req.BaseCoin = feeBuilder.Pair.Base
		} else if req.Symbol, err = e.orderSymbol(feeBuilder.Pair, a); err != nil {
			return 0, err
		}
		rates, err := e.GetFeeRate(ctx, req)
		if err != nil {
			return 0, err
		}
		if len(rates.List) == 0 {
			return 0, fmt.Errorf("%w for %s %s", errFeeRateNotFound, a, feeBuilder.Pair)
		}
		rate := rates.List[0].TakerFeeRate
		if feeBuilder.IsMaker {
			rate = rates.List[0].MakerFeeRate
		}
		switch a {
		case asset.CoinMarginedFutures:
			// An inverse contract is worth one unit of the quote currency, so the fee in the quote currency is the rate
			// times the number of contracts
			return rate.Float64() * feeBuilder.Amount, nil
		case asset.Options:
			// Bybit charges an option trade the fee rate on its underlying's index price, up to a share of the option's price
			// that depends on the account's tier
			symbol, err := e.orderSymbol(feeBuilder.Pair, a)
			if err != nil {
				return 0, err
			}
			tickers, err := e.GetTickers(ctx, &TickersRequest{Category: cOption, Symbol: symbol})
			if err != nil {
				return 0, err
			}
			if len(tickers.List) == 0 {
				return 0, fmt.Errorf("%w for %s %s", ticker.ErrTickerNotFound, a, feeBuilder.Pair)
			}
			fee := rate.Float64() * tickers.List[0].IndexPrice.Float64()
			// No tier caps the fee below 4% of the option's price, so the account's tier matters only above that
			if fee > minOptionFeeCap*feeBuilder.PurchasePrice {
				info, err := e.GetAPIKeyInformation(ctx)
				if err != nil {
					return 0, err
				}
				fee = min(fee, optionFeeCap(info.VIPLevel)*feeBuilder.PurchasePrice)
			}
			return fee * feeBuilder.Amount, nil
		}
		return rate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	}
	return 0, fmt.Errorf("%w: fee type %v", common.ErrFunctionNotSupported, feeBuilder.FeeType)
}

// maxOptionFeeCap and minOptionFeeCap are the largest and smallest shares of an option's price that Bybit's account
// tiers cap the fee of trading it at
const (
	maxOptionFeeCap = 0.07
	minOptionFeeCap = 0.04
)

// optionFeeCap returns the most of an option's price Bybit charges an account of the VIP level as the fee of trading it:
// 7%, but 6% for Pro 2 to 4 and 4% for Pro 5 and 6
func optionFeeCap(vipLevel string) float64 {
	switch vipLevel {
	case "PRO-2", "PRO-3", "PRO-4":
		return 0.06
	case "PRO-5", "PRO-6":
		return minOptionFeeCap
	}
	return maxOptionFeeCap
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.01 * price * amount
}

// feeAsset returns the first of spot, USDT margined, USDC margined and coin margined futures, and options whose
// available pairs include pair
func (e *Exchange) feeAsset(pair currency.Pair) (asset.Item, error) {
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		pairs, err := e.GetAvailablePairs(a)
		if err != nil {
			continue
		}
		if pairs.Contains(pair, true) {
			return a, nil
		}
	}
	return asset.Empty, fmt.Errorf("%w: %s", currency.ErrPairNotFound, pair)
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
		if a == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
			req.RequestFormatted.Delimiter = currency.DashDelimiter
		}
		// Bybit's end is inclusive, and the candle starting at the exclusive end would push the first candle out of a full
		// page
		klines, err := e.GetKlines(ctx, &KlineRequest{Category: getCategoryName(req.Asset), Symbol: req.RequestFormatted.String(), Interval: req.ExchangeInterval, Start: req.Start, End: req.End.Add(-time.Millisecond), Limit: req.RequestLimit})
		if err != nil {
			return nil, err
		}
		return req.ProcessResponse(klineCandles(klines.List, a))
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
			// Bybit's end is inclusive, and the candle starting at the range's exclusive end would push the range's first
			// candle out of a full page
			klines, err := e.GetKlines(ctx, &KlineRequest{Category: getCategoryName(req.Asset), Symbol: req.RequestFormatted.String(), Interval: req.ExchangeInterval, Start: req.RangeHolder.Ranges[x].Start.Time, End: req.RangeHolder.Ranges[x].End.Time.Add(-time.Millisecond), Limit: req.RequestLimit})
			if err != nil {
				return nil, err
			}
			timeSeries = append(timeSeries, klineCandles(klines.List, a)...)
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
}

// klineCandles converts Bybit's candles of an asset to GCT candles
func klineCandles(items []KlineItem, a asset.Item) []kline.Candle {
	candles := make([]kline.Candle, len(items))
	for i := range items {
		volume, quoteVolume := baseQuoteVolumes(items[i].Volume, items[i].Turnover, a)
		candles[i] = kline.Candle{
			Time:        items[i].StartTime.Time(),
			Open:        items[i].OpenPrice.Float64(),
			High:        items[i].HighPrice.Float64(),
			Low:         items[i].LowPrice.Float64(),
			Close:       items[i].ClosePrice.Float64(),
			Volume:      volume,
			QuoteVolume: quoteVolume,
		}
	}
	return candles
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
		quote := i.Symbol[len(i.BaseCoin.String()):]
		return i.BaseCoin.String() + "_" + quote
	case asset.Options:
		quote := strings.TrimPrefix(i.Symbol[len(i.BaseCoin.String()):], currency.DashDelimiter)
		return i.BaseCoin.String() + "-" + quote
	case asset.USDTMarginedFutures:
		quote := i.Symbol[len(i.BaseCoin.String()):]
		return i.BaseCoin.String() + "-" + quote
	case asset.USDCMarginedFutures:
		if i.ContractType != "LinearFutures" {
			quote := i.Symbol[len(i.BaseCoin.String()):]
			return i.BaseCoin.String() + "-" + quote
		}
		fallthrough // Contracts with linear futures already have a delimiter
	default:
		return i.Symbol
	}
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var (
		allInstrumentsInfo InstrumentsInfoResponse
		nextPageCursor     string
	)
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		for {
			instrumentInfo, err := e.GetInstrumentInfo(ctx, &InstrumentInfoRequest{Category: getCategoryName(a), Cursor: nextPageCursor, Limit: 1000})
			if err != nil {
				return err
			}
			switch a {
			case asset.USDTMarginedFutures:
				for i := range instrumentInfo.List {
					if instrumentInfo.List[i].QuoteCoin.String() != "USDT" {
						continue
					}
					allInstrumentsInfo.List = append(allInstrumentsInfo.List, instrumentInfo.List[i])
				}
			case asset.USDCMarginedFutures:
				for i := range instrumentInfo.List {
					if instrumentInfo.List[i].QuoteCoin.String() != "USDC" {
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
		baseCoins, err := e.optionBaseCoins(ctx)
		if err != nil {
			return err
		}
		for _, baseCoin := range baseCoins {
			nextPageCursor = ""
			for {
				instrumentInfo, err := e.GetInstrumentInfo(ctx, &InstrumentInfoRequest{Category: getCategoryName(a), Status: tradingStatus, BaseCoin: baseCoin.String(), Cursor: nextPageCursor, Limit: 1000})
				if err != nil {
					return fmt.Errorf("%w - %v", err, baseCoin)
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
	for i := range allInstrumentsInfo.List {
		inst := &allInstrumentsInfo.List[i]
		if a == asset.Options && inst.Status != tradingStatus {
			continue
		}
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

		var minBaseAmount, maxBaseAmount, marketMaxQty, baseStepAmount, minQuoteAmount float64
		switch a {
		case asset.Spot:
			// Bybit checks a spot order's quantity against the maximum for its order type and its value against
			// minOrderAmt, and no longer checks the deprecated minOrderQty, maxOrderQty and maxOrderAmt
			maxBaseAmount = inst.LotSizeFilter.MaxLimitOrderQuantity.Float64()
			marketMaxQty = inst.LotSizeFilter.MaxMarketOrderQuantity.Float64()
			baseStepAmount = inst.LotSizeFilter.BasePrecision.Float64()
			minQuoteAmount = inst.LotSizeFilter.MinOrderAmount.Float64()
		default:
			minBaseAmount = inst.LotSizeFilter.MinOrderQuantity.Float64()
			maxBaseAmount = inst.LotSizeFilter.MaxOrderQuantity.Float64()
			baseStepAmount = inst.LotSizeFilter.QuantityStep.Float64()
			minQuoteAmount = inst.LotSizeFilter.MinNotionalValue.Float64()
			if a != asset.Options {
				marketMaxQty = inst.LotSizeFilter.MaxMktOrderQuantity.Float64()
			}
		}

		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
			MinimumBaseAmount:       minBaseAmount,
			MaximumBaseAmount:       maxBaseAmount,
			MarketMaxQty:            marketMaxQty,
			MinPrice:                inst.PriceFilter.MinPrice.Float64(),
			MaxPrice:                inst.PriceFilter.MaxPrice.Float64(),
			PriceStepIncrementSize:  inst.PriceFilter.TickSize.Float64(),
			AmountStepIncrementSize: baseStepAmount,
			QuoteStepIncrementSize:  inst.LotSizeFilter.QuotePrecision.Float64(),
			MinimumQuoteAmount:      minQuoteAmount,
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
		params := &SetLeverageRequest{
			Category: getCategoryName(item),
			Symbol:   pair.String(),
		}
		switch orderSide {
		case order.Buy, order.Sell, order.Long, order.Short, order.UnknownSide:
			// One-way mode and cross margin need equal buy and sell leverage, and only isolated margin hedge mode
			// positions may differ, so both sides are set
			params.BuyLeverage, params.SellLeverage = amount, amount
		default:
			return order.ErrSideIsInvalid
		}
		err = e.SetLeverageLevel(ctx, params)
		if apiErr := (*APIError)(nil); errors.As(err, &apiErr) && apiErr.Code == leverageNotModifiedCode {
			return nil
		}
		return err
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// leverageNotModifiedCode is Bybit's answer to a leverage that is already set
const leverageNotModifiedCode = 110043

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
	format, err := e.GetPairFormat(item, false)
	if err != nil {
		return nil, err
	}
	instruments, err := e.instruments(ctx, &InstrumentInfoRequest{Category: getCategoryName(item)})
	if err != nil {
		return nil, err
	}
	settlementType, settle := futures.Linear, orderSettleCoin(item)
	if item == asset.CoinMarginedFutures {
		settlementType = futures.Inverse
	}
	resp := make([]futures.Contract, 0, len(instruments))
	for i := range instruments {
		inst := &instruments[i]
		// USDT and USDC margined futures share the linear category
		if !settle.IsEmpty() && !inst.SettleCoin.Equal(settle) {
			continue
		}
		cp, err := currency.NewPairFromString(inst.transformSymbol(item))
		if err != nil {
			return nil, err
		}
		var start, end time.Time
		if inst.LaunchTime.Time().UnixMilli() > 0 {
			start = inst.LaunchTime.Time()
		}
		if inst.DeliveryTime.Time().UnixMilli() > 0 {
			end = inst.DeliveryTime.Time()
		}
		ct := futures.Unknown
		switch inst.ContractType {
		case "LinearPerpetual", "InversePerpetual":
			ct = futures.Perpetual
		case "LinearFutures", "InverseFutures":
			if ct, err = getContractLength(end.Sub(start)); err != nil {
				return nil, fmt.Errorf("%w %v %v %v %v-%v", err, e.Name, item, cp, start, end)
			}
		default:
			if e.Verbose {
				log.Warnf(log.ExchangeSys, "%v unhandled contract type %q for %v %v", e.Name, inst.ContractType, item, cp)
			}
		}
		resp = append(resp, futures.Contract{
			Exchange:           e.Name,
			Name:               cp.Format(format),
			Underlying:         currency.NewPair(inst.BaseCoin, inst.QuoteCoin),
			Asset:              item,
			StartDate:          start,
			EndDate:            end,
			SettlementType:     settlementType,
			IsActive:           inst.Status == tradingStatus,
			Status:             inst.Status,
			Type:               ct,
			SettlementCurrency: inst.SettleCoin,
			MarginCurrency:     inst.SettleCoin,
			// A Bybit contract is one unit of the base coin, or of the quote currency for inverse contracts
			Multiplier:         1,
			MaxLeverage:        inst.LeverageFilter.MaxLeverage.Float64(),
			FundingRateFloor:   inst.LowerFundingRate.Decimal(),
			FundingRateCeiling: inst.UpperFundingRate.Decimal(),
		})
	}
	return resp, nil
}

// instruments returns every instrument req selects, a page at a time
func (e *Exchange) instruments(ctx context.Context, req *InstrumentInfoRequest) ([]InstrumentInfo, error) {
	r := *req
	r.Limit = 1000
	var all []InstrumentInfo
	for {
		resp, err := e.GetInstrumentInfo(ctx, &r)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.List...)
		if resp.NextPageCursor == "" || resp.NextPageCursor == r.Cursor || len(resp.List) == 0 {
			return all, nil
		}
		r.Cursor = resp.NextPageCursor
	}
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
		ct = futures.LongDated
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
		ticks, err := e.GetTickers(ctx, &TickersRequest{Category: getCategoryName(r.Asset), Symbol: symbol})
		if err != nil {
			return nil, err
		}

		instruments, err := e.instruments(ctx, &InstrumentInfoRequest{Category: getCategoryName(r.Asset), Symbol: symbol})
		if err != nil {
			return nil, err
		}

		resp := make([]fundingrate.LatestRateResponse, 0, len(ticks.List))
		for i := range ticks.List {
			var cp currency.Pair
			var isEnabled bool
			cp, isEnabled, err = e.MatchSymbolCheckEnabled(ticks.List[i].Symbol, r.Asset, hasPotentialDelimiter(r.Asset))
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return nil, err
			} else if !isEnabled {
				continue
			}
			var fundingInterval time.Duration
			for j := range instruments {
				if instruments[j].Symbol != ticks.List[i].Symbol {
					continue
				}
				fundingInterval = time.Duration(instruments[j].FundingInterval) * time.Minute //nolint:gosec // A funding interval is a few hundred minutes
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
					Rate: decimal.MustFromFloat(ticks.List[i].FundingRate.Float64()),
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

// GetOpenInterest returns the open interest of the given futures pairs, or of every enabled futures pair when none
// are given. It reports Bybit's single side figure, as Bybit's openInterest counts each contract on both sides
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.USDCMarginedFutures && k[i].Asset != asset.USDTMarginedFutures && k[i].Asset != asset.CoinMarginedFutures {
			return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, k[i].Asset)
		}
	}
	if len(k) == 1 {
		symbol, err := e.orderSymbol(k[0].Pair(), k[0].Asset)
		if err != nil {
			return nil, err
		}
		ticks, err := e.GetTickers(ctx, &TickersRequest{Category: getCategoryName(k[0].Asset), Symbol: symbol})
		if err != nil {
			return nil, err
		}
		for i := range ticks.List {
			if ticks.List[i].Symbol == symbol {
				return []futures.OpenInterest{{
					Key:          key.NewExchangeAssetPair(e.Name, k[0].Asset, k[0].Pair()),
					OpenInterest: ticks.List[i].SingleOpenInterest.Float64(),
				}}, nil
			}
		}
		return nil, fmt.Errorf("%w: %s %s", currency.ErrPairNotFound, k[0].Asset, k[0].Pair())
	}
	var resp []futures.OpenInterest
	// USDT and USDC margined futures share the linear category's tickers
	tickers := make(map[string]*TickersResponse)
	for _, a := range []asset.Item{asset.USDCMarginedFutures, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		if len(k) != 0 && !slices.ContainsFunc(k, func(pa key.PairAsset) bool { return pa.Asset == a }) {
			continue
		}
		category := getCategoryName(a)
		ticks, ok := tickers[category]
		if !ok {
			var err error
			if ticks, err = e.GetTickers(ctx, &TickersRequest{Category: category}); err != nil {
				return nil, err
			}
			tickers[category] = ticks
		}
		for x := range ticks.List {
			pair, isEnabled, err := e.MatchSymbolCheckEnabled(ticks.List[x].Symbol, a, hasPotentialDelimiter(a))
			if err != nil || !isEnabled {
				continue
			}
			if len(k) != 0 && !slices.ContainsFunc(k, func(pa key.PairAsset) bool { return pa.Asset == a && pa.Pair().Equal(pair) }) {
				continue
			}
			resp = append(resp, futures.OpenInterest{
				Key:          key.NewExchangeAssetPair(e.Name, a, pair),
				OpenInterest: ticks.List[x].SingleOpenInterest.Float64(),
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
		var io *InstrumentsInfoResponse
		io, err = e.GetInstrumentInfo(ctx, &InstrumentInfoRequest{Category: getCategoryName(a), Symbol: symbol, Limit: 1000})
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
