package kucoin

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

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
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// AccountToTradeTypeString returns KuCoin's trade type for an asset: TRADE for spot, and for margin
// MARGIN_ISOLATED_TRADE for isolated margin or MARGIN_TRADE for cross margin, which any other margin type selects.
// Other assets have no trade type, so it returns an empty string for them
func (e *Exchange) AccountToTradeTypeString(a asset.Item, marginType margin.Type) string {
	switch a {
	case asset.Spot:
		return SpotTradeType
	case asset.Margin:
		if marginType == margin.Isolated {
			return IsolatedMarginTradeType
		}
		return CrossMarginTradeType
	default:
		return ""
	}
}

// OrderSideString converts an order.Side instance to a string representation
func (e *Exchange) OrderSideString(side order.Side) (string, error) {
	switch {
	case side.IsLong():
		return order.Buy.Lower(), nil
	case side.IsShort():
		return order.Sell.Lower(), nil
	case side == order.AnySide:
		return "", nil
	default:
		return "", fmt.Errorf("%w, side:%s", order.ErrSideIsInvalid, side.Lower())
	}
}

// SetDefaults sets the basic defaults for Kucoin
func (e *Exchange) SetDefaults() {
	e.Name = "Kucoin"
	e.Enabled = true
	e.Verbose = false

	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresClientID = true

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		if a == asset.Futures {
			ps.RequestFormat.Delimiter = ""
			ps.ConfigFormat.Delimiter = currency.UnderscoreDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		CurrencyTranslations: currency.NewTranslations(map[currency.Code]currency.Code{
			currency.XBT:   currency.BTC,
			currency.USDTM: currency.USDT,
			currency.USDM:  currency.USD,
			currency.USDCM: currency.USDC,
		}),
		TradingRequirements: protocol.TradingRequirements{
			ClientOrderID: true,
		},
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:        true,
				TickerBatching:        true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				SubmitOrder:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				CancelOrders:          true,
				ModifyOrder:           true,
				TradeFetching:         true,
				UserTradeHistory:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				KlineFetching:         true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
				// GetDepositAddress cannot choose among addresses on several chains, so the engine names the chain
				MultiChainDepositRequiresChainSet: true,
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
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:                 true,
				Leverage:                  true,
				FundingRates:              true,
				MaximumFundingRateHistory: kline.ThreeMonth.Duration(),
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour:   true,
					kline.FourHour:  true,
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.Futures: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportedViaTicker: true,
					SupportsRestBatch:  true,
				},
			},
			MaximumOrderHistory: kline.OneDay.Duration() * 7,
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
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
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
				),
				GlobalResultLimit: 500,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              kucoinAPIURL,
		exchange.RestFutures:           kucoinFuturesAPIURL,
		exchange.RestSpotSupplementary: kucoinBrokerAPIURL, // Exchange Broker endpoints are served from their own host
		exchange.WebsocketSpot:         kucoinWebsocketURL,
		exchange.WebsocketFutures:      kucoinFuturesWebsocketURL,
	})
	if err != nil {
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

	if err := e.checkSubscriptions(); err != nil {
		return err
	}

	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:                         exch,
		Features:                               &e.Features.Supports.WebsocketCapabilities,
		TradeFeed:                              e.Features.Enabled.TradeFeed,
		UseMultiConnectionManagement:           true,
		MaxWebsocketSubscriptionsPerConnection: 400,
	}); err != nil {
		return err
	}
	e.wsOBUpdateMgr = e.newOrderbookUpdateManager()

	wsSpotURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	wsFuturesURL, err := e.API.Endpoints.GetURL(exchange.WebsocketFutures)
	if err != nil {
		return err
	}
	// The UTA connections generate subscriptions only from configured UTA channels, so they stay down by default. Every
	// connection but spot's is optional: one that fails to connect or drops leaves the others up while the manager
	// retries it
	setups := []*websocket.ConnectionSetup{
		e.newSpotConnectionSetup(exch, wsSpotURL),
		e.newFuturesConnectionSetup(exch, wsFuturesURL),
		e.newUTASpotPublicConnection(exch, kucoinUTASpotPublicWebsocketURL),
		e.newUTAFuturesPublicConnection(exch, kucoinUTAFuturesPublicWebsocketURL),
		e.newUTAPrivateConnection(exch, kucoinUTAPrivateWebsocketURL),
	}
	// The order connections carry no subscriptions, so the manager always dials them, and they sign in with the
	// account's keys. They are set up only for authenticated websocket use with credentials, and the UTA one only for an
	// account a configured UTA channel shows to be unified
	if e.Websocket.CanUseAuthenticatedEndpoints() && e.AreCredentialsValid(context.Background()) {
		setups = append(setups, e.newOrderConnectionSetup(exch, kucoinOrderWebsocketURL))
		if e.hasUTASubscriptions() {
			setups = append(setups, e.newUTATradeConnection(exch, kucoinUTATradeWebsocketURL))
		}
	}
	for _, s := range setups {
		if err := e.Websocket.SetupNewConnection(s); err != nil {
			return err
		}
	}
	return nil
}

// errContractSymbolMismatch is returned when a futures contract's symbol does not start with its base currency, which
// the contract's pair is derived from
var errContractSymbolMismatch = errors.New("contract symbol does not start with its base currency")

// futuresContractPair returns a futures contract's pair: its base currency, and the rest of its symbol as the quote,
// such as XBT and USDTM for XBTUSDTM or XBT and MZ26 for the dated XBTMZ26
func futuresContractPair(c *ContractResponse) (currency.Pair, error) {
	base := c.BaseCurrency.String()
	if base == "" || len(c.Symbol) <= len(base) || !strings.EqualFold(c.Symbol[:len(base)], base) {
		return currency.EMPTYPAIR, fmt.Errorf("%w: %q %q", errContractSymbolMismatch, c.Symbol, base)
	}
	return currency.NewPair(c.BaseCurrency, currency.NewCode(c.Symbol[len(base):])), nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	switch assetType {
	case asset.Futures:
		contracts, err := e.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(contracts))
		for i := range contracts {
			if !strings.EqualFold(contracts[i].Status, "open") {
				continue
			}
			pair, err := futuresContractPair(&contracts[i])
			if err != nil {
				return nil, err
			}
			pairs = pairs.Add(pair)
		}
		return pairs, nil
	case asset.Spot, asset.Margin:
		myPairs, err := e.GetSymbols(ctx, "")
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(myPairs))
		for x := range myPairs {
			if !myPairs[x].EnableTrading || (assetType == asset.Margin && !myPairs[x].IsMarginEnabled) {
				continue
			}
			// Symbol field must be used to generate pair as this is the symbol
			// to fetch data from the API. e.g. BSV-USDT name is BCHSV-USDT as symbol.
			pair, err := currency.NewPairFromString(myPairs[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = pairs.Add(pair)
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(true)
	for a := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[a])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assets[a], false); err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if err := e.UpdateTickers(ctx, assetType); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// contractVolumes maps a futures contract's two volume figures onto base and quote. KuCoin reports
// volumeOf24h in the base currency and turnoverOf24h in the quote, except on an inverse contract,
// which is worth one unit of its quote currency, where the two swap over. Both count the maker and taker
// sides of each trade, so they are halved to the one-sided volume the symbol snapshot push and the klines
// report
func contractVolumes(c *ContractResponse) (baseVolume, quoteVolume float64) {
	if c.IsInverse {
		return c.TurnoverOf24Hour / 2, c.VolumeOf24Hour / 2
	}
	return c.VolumeOf24Hour / 2, c.TurnoverOf24Hour / 2
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	var errs error
	switch assetType {
	case asset.Futures:
		tickers, err := e.GetFuturesTickers(ctx)
		for i := range tickers {
			errs = common.AppendError(errs, ticker.ProcessTicker(tickers[i]))
		}
		return common.AppendError(errs, err)
	case asset.Spot, asset.Margin:
		ticks, err := e.GetTickers(ctx)
		if err != nil {
			return err
		}
		for t := range ticks.Ticker {
			pair, enabled, err := e.MatchSymbolCheckEnabled(ticks.Ticker[t].Symbol, assetType, true)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return err
			}
			if !enabled {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks.Ticker[t].Last.Float64(),
				LastSize:     ticks.Ticker[t].LastSize.Float64(),
				High:         ticks.Ticker[t].High.Float64(),
				Low:          ticks.Ticker[t].Low.Float64(),
				BaseVolume:   ticks.Ticker[t].Volume.Float64(),
				QuoteVolume:  ticks.Ticker[t].VolumeValue.Float64(),
				Ask:          ticks.Ticker[t].Sell.Float64(),
				AskSize:      ticks.Ticker[t].BestAskSize.Float64(),
				Bid:          ticks.Ticker[t].Buy.Float64(),
				BidSize:      ticks.Ticker[t].BestBidSize.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    assetType,
				LastUpdated:  ticks.Time.Time(),
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return errs
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	err := e.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return nil, err
	}
	p, err = e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var bids, asks orderbook.Levels
	var lastUpdated time.Time
	switch a {
	case asset.Futures:
		book, err := e.GetFuturesOrderbook(ctx, p.String())
		if err != nil {
			return nil, err
		}
		if book == nil {
			return nil, common.ErrNoResponse
		}
		bids, asks, lastUpdated = book.Bids.Levels(), book.Asks.Levels(), book.Timestamp.Time()
	case asset.Spot, asset.Margin:
		book, err := e.GetPartOrderbook(ctx, p.String(), 100)
		if err != nil {
			return nil, err
		}
		bids, asks, lastUpdated = book.Bids.Levels(), book.Asks.Levels(), book.Time.Time()
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}

	ob := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
		Asks:              mergeRoundedOrderbookLevels(asks),
		Bids:              mergeRoundedOrderbookLevels(bids),
		LastUpdated:       lastUpdated,
	}

	if err := ob.Process(); err != nil {
		return nil, err
	}

	return orderbook.Get(e.Name, p, a)
}

// KuCoin's rounded books can emit adjacent levels that can be collapsed to the same
// price, so we consolidate them before loading the local book to avoid
// duplicate-price levels downstream.
func mergeRoundedOrderbookLevels(levels []orderbook.Level) []orderbook.Level {
	if len(levels) < 2 {
		return levels
	}

	writeIndex := 1
	for i := 1; i < len(levels); i++ {
		if levels[writeIndex-1].Price == levels[i].Price {
			levels[writeIndex-1].Amount += levels[i].Amount
			continue
		}
		levels[writeIndex] = levels[i]
		writeIndex++
	}
	return levels[:writeIndex]
}

// futuresSettlementCurrencies are the currencies KuCoin settles futures contracts in, each with its own futures account,
// as Get All Symbols listed them on 2026-10-14
var futuresSettlementCurrencies = []currency.Code{currency.USDT, currency.USDC, currency.XBT, currency.ETH, currency.SOL, currency.XRP}

// UpdateAccountBalances retrieves currency balances. For margin it returns the cross margin account, and saves each
// isolated margin account beside it as the margin sub-account of its symbol
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	switch assetType {
	case asset.Futures:
		for _, settlement := range futuresSettlementCurrencies {
			resp, err := e.GetFuturesAccountOverview(ctx, settlement)
			if err != nil {
				return nil, err
			}
			if resp == nil {
				return nil, common.ErrNoResponse
			}
			// Account equity is position margin, order margin, frozen funds and the available balance together
			subAccts[0].Balances.Set(resp.Currency, accounts.Balance{
				Total: resp.AccountEquity,
				Hold:  resp.PositionMargin + resp.OrderMargin + resp.FrozenFunds,
				Free:  resp.AvailableBalance,
			})
		}
	case asset.Margin:
		// Stamped before the requests, so that a push arriving while they are answered is neither undone nor refused, and
		// to the millisecond, as pushes are, so that a push in the listing's millisecond does not read as older
		listedAt := time.Now().Truncate(time.Millisecond)
		resp, err := e.GetCrossMarginAccountsDetail(ctx, currency.EMPTYCODE, "")
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		for i := range resp.Accounts {
			subAccts[0].Balances.Set(resp.Accounts[i].Currency, marginAssetBalance(&resp.Accounts[i], listedAt))
		}
		isolated, err := e.GetIsolatedMarginAccountDetail(ctx, &IsolatedMarginAccountDetailRequest{QueryType: "ISOLATED"})
		if err != nil {
			return nil, err
		}
		if isolated == nil {
			return nil, common.ErrNoResponse
		}
		snapshot := slices.Clone(subAccts)
		for i := range isolated.Assets {
			// The cross margin account is sub-account "", which a symbol must not overwrite
			if isolated.Assets[i].Symbol == "" {
				return nil, currency.ErrSymbolStringEmpty
			}
			s := accounts.NewSubAccount(asset.Margin, isolated.Assets[i].Symbol)
			s.Balances.Set(isolated.Assets[i].BaseAsset.Currency, marginAssetBalance(&isolated.Assets[i].BaseAsset, listedAt))
			s.Balances.Set(isolated.Assets[i].QuoteAsset.Currency, marginAssetBalance(&isolated.Assets[i].QuoteAsset, listedAt))
			snapshot = append(snapshot, s)
		}
		// The isolated margin accounts are saved beside the cross margin account, which alone is returned, with any
		// balance kept from a push, as callers such as the backtester's live funding set each currency from one returned
		// sub-account after another
		return subAccts, e.saveMarginSnapshot(ctx, snapshot, listedAt)
	case asset.Spot:
		// Stamped before the request, so that a push arriving while it is answered keeps its account's share, and to the
		// millisecond, as pushes are, so that a push in the listing's millisecond does not read as older
		listedAt := time.Now().Truncate(time.Millisecond)
		resp, err := e.GetAllAccounts(ctx, currency.EMPTYCODE, "")
		if err != nil {
			return nil, err
		}
		shares := make(spotAccountShares)
		for i := range resp {
			// The account list's margin and isolated types are abandoned; the main and trade accounts both hold spot funds
			if resp[i].AccountType == "margin" || resp[i].AccountType == "isolated" {
				continue
			}
			if resp[i].Currency.IsEmpty() {
				return nil, currency.ErrCurrencyCodeEmpty
			}
			// Balance pushes count the HF trade account, trade_hf, which an account trading at high frequency since before
			// 2024 uses in place of trade, as the trade account
			account := resp[i].AccountType
			if account == "trade_hf" {
				account = "trade"
			}
			if shares[account] == nil {
				shares[account] = make(map[*currency.Item]accounts.Balance)
			}
			b := accounts.Balance{
				Total:     resp[i].Balance.Float64(),
				Hold:      resp[i].Holds.Float64(),
				Free:      resp[i].Available.Float64(),
				UpdatedAt: listedAt,
			}
			if listed, ok := shares[account][resp[i].Currency.Item]; ok {
				b = b.Add(listed)
			}
			b.Currency = resp[i].Currency
			shares[account][resp[i].Currency.Item] = b
		}
		return e.saveSpotAccountShares(ctx, shares, listedAt)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// marginAssetBalance returns the balance of a cross or isolated margin account's currency as of its listing's time
func marginAssetBalance(a *MarginAssetDetail, listedAt time.Time) accounts.Balance {
	return accounts.Balance{
		Total:     a.Total.Float64(),
		Hold:      a.Hold.Float64(),
		Free:      a.Available.Float64(),
		Borrowed:  a.Liability.Float64(),
		UpdatedAt: listedAt,
	}
}

// saveMarginSnapshot saves the cross margin account and each isolated margin account REST listed as the margin
// sub-accounts, and empties the sub-account of an isolated margin symbol the account no longer lists. A balance a push
// saved after the listing's time is kept, set on the listed sub-account it belongs to, and a listing older than one
// already saved is left unsaved
func (e *Exchange) saveMarginSnapshot(ctx context.Context, subAccts accounts.SubAccounts, listedAt time.Time) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	e.marginBalanceMu.Lock()
	defer e.marginBalanceMu.Unlock()
	// A listing answered after a newer one was saved holds older figures than those saved, liabilities included
	if listedAt.Before(e.marginAccountsListedAt[*creds]) {
		return nil
	}
	saved, err := e.Accounts.SubAccounts(creds, asset.Margin)
	if err != nil && !errors.Is(err, accounts.ErrNoSubAccounts) {
		return err
	}
	snapshot := slices.Clone(subAccts)
	for _, s := range saved {
		i := slices.IndexFunc(snapshot, func(listed *accounts.SubAccount) bool { return listed.ID == s.ID })
		if i == -1 {
			snapshot = append(snapshot, accounts.NewSubAccount(asset.Margin, s.ID))
			i = len(snapshot) - 1
		}
		for c, b := range s.Balances {
			if !b.UpdatedAt.After(listedAt) {
				continue
			}
			// A balance a push saved after the listing's time is kept over the listing's figures, as for spot, as the
			// listing cannot show whether KuCoin took them before or after the push, though the liability, which a push
			// carries none of, is the listing's: none for a currency it leaves out
			b.Borrowed = snapshot[i].Balances[c].Borrowed
			snapshot[i].Balances.Set(c, b)
		}
	}
	if err := e.Accounts.Save(ctx, snapshot, true); err != nil {
		return err
	}
	if e.marginAccountsListedAt == nil {
		e.marginAccountsListedAt = make(map[accounts.Credentials]time.Time)
	}
	if listedAt.After(e.marginAccountsListedAt[*creds]) {
		e.marginAccountsListedAt[*creds] = listedAt
	}
	return nil
}

// spotAccountShares holds each account's balance of each currency, keyed by account type: main and trade
type spotAccountShares map[string]map[*currency.Item]accounts.Balance

// saveSpotAccountShares saves the account balances REST listed as the spot balance, each currency the accounts'
// balances together as of the latest of their times. A balance push newer than the listing keeps its account's share
func (e *Exchange) saveSpotAccountShares(ctx context.Context, shares spotAccountShares, listedAt time.Time) (accounts.SubAccounts, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	e.spotAccountBalancesMu.Lock()
	defer e.spotAccountBalancesMu.Unlock()
	for account, saved := range e.spotAccountBalances[*creds] {
		for item, b := range saved {
			if b.UpdatedAt.After(listedAt) {
				if shares[account] == nil {
					shares[account] = make(map[*currency.Item]accounts.Balance)
				}
				shares[account][item] = b
			}
		}
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, "")}
	for _, accountShares := range shares {
		for _, b := range accountShares {
			if err := subAccts[0].Balances.Add(b.Currency, b); err != nil {
				return nil, err
			}
		}
	}
	if err := e.Accounts.Save(ctx, subAccts, true); err != nil {
		return nil, err
	}
	if e.spotAccountBalances == nil {
		e.spotAccountBalances = make(map[accounts.Credentials]spotAccountShares)
	}
	e.spotAccountBalances[*creds] = shares
	if e.spotAccountsListedAt == nil {
		e.spotAccountsListedAt = make(map[accounts.Credentials]time.Time)
	}
	if listedAt.After(e.spotAccountsListedAt[*creds]) {
		e.spotAccountsListedAt[*creds] = listedAt
	}
	return subAccts, nil
}

// fundingHistoryPageSize is the most records Get Deposit History and Get Withdrawal History return per page
const fundingHistoryPageSize = 500

// depositHistory returns every page of a currency's deposits, latest first
func (e *Exchange) depositHistory(ctx context.Context, ccy currency.Code) ([]DepositRecord, error) {
	req := &DepositHistoryRequest{Currency: ccy, PageSize: fundingHistoryPageSize}
	var deposits []DepositRecord
	for req.CurrentPage = 1; ; req.CurrentPage++ {
		resp, err := e.GetDepositHistory(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		deposits = append(deposits, resp.Items...)
		if req.CurrentPage >= resp.TotalPage {
			return deposits, nil
		}
	}
}

// withdrawalHistory returns every page of a currency's withdrawals, latest first
func (e *Exchange) withdrawalHistory(ctx context.Context, ccy currency.Code) ([]Withdrawal, error) {
	req := &WithdrawalHistoryRequest{Currency: ccy, PageSize: fundingHistoryPageSize}
	var withdrawals []Withdrawal
	for req.CurrentPage = 1; ; req.CurrentPage++ {
		resp, err := e.GetWithdrawalHistory(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		withdrawals = append(withdrawals, resp.Items...)
		if req.CurrentPage >= resp.TotalPage {
			return withdrawals, nil
		}
	}
}

// GetAccountFundingHistory returns the deposits and withdrawals of each currency in the spot account list, since
// KuCoin's deposit and withdrawal histories are queried by currency
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	accts, err := e.GetAllAccounts(ctx, currency.EMPTYCODE, "")
	if err != nil {
		return nil, err
	}
	var currencies currency.Currencies
	for i := range accts {
		currencies = currencies.Add(accts[i].Currency)
	}
	var resp []exchange.FundingHistory
	for _, ccy := range currencies {
		deposits, err := e.depositHistory(ctx, ccy)
		if err != nil {
			return nil, err
		}
		for i := range deposits {
			resp = append(resp, exchange.FundingHistory{
				ExchangeName:    e.Name,
				Status:          deposits[i].Status,
				TransferID:      deposits[i].ID,
				Description:     deposits[i].Remark,
				Timestamp:       deposits[i].CreatedAt.Time(),
				Currency:        deposits[i].Currency.String(),
				Amount:          deposits[i].Amount.Float64(),
				Fee:             deposits[i].Fee.Float64(),
				TransferType:    "deposit",
				CryptoToAddress: deposits[i].Address,
				CryptoTxID:      deposits[i].WalletTxID,
				CryptoChain:     deposits[i].Chain,
			})
		}
		withdrawals, err := e.withdrawalHistory(ctx, ccy)
		if err != nil {
			return nil, err
		}
		for i := range withdrawals {
			resp = append(resp, exchange.FundingHistory{
				ExchangeName:    e.Name,
				Status:          withdrawals[i].Status,
				TransferID:      withdrawals[i].ID,
				Description:     withdrawals[i].Remark,
				Timestamp:       withdrawals[i].CreatedAt.Time(),
				Currency:        withdrawals[i].Currency.String(),
				Amount:          withdrawals[i].Amount.Float64(),
				Fee:             withdrawals[i].Fee.Float64(),
				TransferType:    "withdrawal",
				CryptoToAddress: withdrawals[i].Address,
				CryptoTxID:      withdrawals[i].WalletTxID,
				CryptoChain:     withdrawals[i].Chain,
			})
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns every withdrawal of a currency, latest first
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, assetType asset.Item) ([]exchange.WithdrawalHistory, error) {
	if !e.SupportsAsset(assetType) {
		return nil, asset.ErrNotSupported
	}
	withdrawals, err := e.withdrawalHistory(ctx, c)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals))
	for i := range withdrawals {
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawals[i].Status,
			TransferID:      withdrawals[i].ID,
			Description:     withdrawals[i].Remark,
			Timestamp:       withdrawals[i].CreatedAt.Time(),
			Currency:        withdrawals[i].Currency.String(),
			Amount:          withdrawals[i].Amount.Float64(),
			Fee:             withdrawals[i].Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[i].Address,
			CryptoTxID:      withdrawals[i].WalletTxID,
			CryptoChain:     withdrawals[i].Chain,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Futures:
		tradeData, err := e.GetFuturesTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(tradeData))
		for i := range tradeData {
			side, err := order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp[i] = trade.Data{
				TID:          tradeData[i].TradeID,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price.Float64(),
				Amount:       float64(tradeData[i].Size),
				Timestamp:    tradeData[i].Timestamp.Time(),
				Side:         side,
			}
		}
	case asset.Spot, asset.Margin:
		tradeData, err := e.GetTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(tradeData))
		for i := range tradeData {
			side, err := order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp[i] = trade.Data{
				TID:          tradeData[i].TradeID,
				Exchange:     e.Name,
				CurrencyPair: p,
				Side:         side,
				AssetType:    assetType,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Size.Float64(),
				Timestamp:    tradeData[i].Time.Time(),
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// Wrapper order errors
var (
	errFuturesSizeNotWholeLots   = errors.New("futures order size must be a whole number of lots")
	errStopModesExclusive        = errors.New("stop loss and stop entry cannot both be enabled")
	errTakeProfitPriceRequired   = errors.New("take profit price is required")
	errStopLossPriceRequired     = errors.New("stop loss price is required")
	errOCOPricesInverted         = errors.New("stop loss price is on the wrong side of the take profit price")
	errOCOPriceMismatch          = errors.New("order price differs from the take profit price")
	errRiskModeUnsupported       = errors.New("risk management mode not supported")
	errOrderNotCancelled         = errors.New("order was not cancelled")
	errOrderPairMismatch         = errors.New("order belongs to another pair")
	errLeverageNotModified       = errors.New("leverage was not modified")
	errMarginUnchanged           = errors.New("new allocated margin equals the original allocated margin")
	errPositionDirectionRequired = errors.New("position direction is required in hedge mode")
)

// requestFormat returns the pair format of an enabled asset's requests
func (e *Exchange) requestFormat(a asset.Item) (currency.PairFormat, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return currency.EMPTYFORMAT, err
	}
	return e.GetPairFormat(a, true)
}

// SubmitOrder submits an order over REST. Spot and margin stop orders take their trigger from RiskManagementModes:
// StopLoss triggers as the price falls to its price and StopEntry as the price rises to it. An OCO order places a
// limit order at the TakeProfit price and a stop order, which the StopLoss price triggers, placing a limit order at
// the StopLoss LimitPrice. Futures orders are sized in whole lots by Amount, or in the quote currency by QuoteAmount,
// and a futures stop or take profit order triggers at TriggerPrice; a futures limit or market buy can carry TakeProfit
// and StopLoss triggers
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	pFmt, err := e.requestFormat(s.AssetType)
	if err != nil {
		return nil, err
	}
	symbol := pFmt.Format(s.Pair)
	side := order.Buy.Lower()
	if s.Side.IsShort() {
		side = order.Sell.Lower()
	}
	switch s.AssetType {
	case asset.Futures:
		r, err := futuresOrderRequest(s, symbol, side)
		if err != nil {
			return nil, err
		}
		var resp *FuturesOrderResponse
		if s.RiskManagementModes.TakeProfit.Enabled || s.RiskManagementModes.StopLoss.Enabled {
			var tpsl *FuturesTakeProfitStopLossOrderRequest
			if tpsl, err = futuresTakeProfitStopLossRequest(r, &s.RiskManagementModes); err != nil {
				return nil, err
			}
			resp, err = e.PostFuturesTakeProfitStopLossOrder(ctx, tpsl)
		} else {
			resp, err = e.PostFuturesOrder(ctx, r)
		}
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		return s.DeriveSubmitResponse(resp.OrderID)
	case asset.Margin:
		return e.submitMarginOrder(ctx, s, symbol, side)
	default: // Spot, the only other asset requestFormat accepts
		return e.submitSpotOrder(ctx, s, symbol, side)
	}
}

// futuresOrderRequest returns the futures order a submission describes. A stop order triggers as the price moves
// against its side, and a take profit order as the price moves in its favour
func futuresOrderRequest(s *order.Submit, symbol, side string) (*FuturesOrderRequest, error) {
	if s.RiskManagementModes.StopEntry.Enabled {
		return nil, fmt.Errorf("%w: a futures order has no stop entry, and a futures stop order triggers at TriggerPrice", errRiskModeUnsupported)
	}
	if s.Amount != math.Trunc(s.Amount) {
		return nil, fmt.Errorf("%w: %v", errFuturesSizeNotWholeLots, s.Amount)
	}
	if s.Leverage < 0 || s.Leverage != math.Trunc(s.Leverage) {
		return nil, fmt.Errorf("%w: KuCoin takes whole leverage, not %v", errInvalidLeverage, s.Leverage)
	}
	timeInForce, err := timeInForceString(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	r := &FuturesOrderRequest{
		ClientOrderID: s.ClientOrderID,
		Side:          side,
		Symbol:        symbol,
		Leverage:      uint64(s.Leverage),
		ReduceOnly:    s.ReduceOnly,
		Price:         s.Price,
		Size:          uint64(s.Amount),
		TimeInForce:   timeInForce,
		PostOnly:      s.TimeInForce.Is(order.PostOnly),
		ValueQuantity: s.QuoteAmount,
	}
	if s.MarginType != margin.Unset {
		if r.MarginMode, err = futuresMarginMode(s.MarginType); err != nil {
			return nil, err
		}
	}
	switch s.Type {
	case order.Limit, order.Market:
		r.Type = s.Type.Lower()
		return r, nil
	case order.Stop, order.StopLimit, order.TakeProfit:
		r.Type = order.Limit.Lower()
	case order.StopMarket, order.TakeProfitMarket:
		r.Type = order.Market.Lower()
	default:
		return nil, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, s.Type)
	}
	if s.TriggerPrice <= 0 {
		return nil, errStopPriceRequired
	}
	takeProfit := s.Type == order.TakeProfit || s.Type == order.TakeProfitMarket
	r.Stop = "down"
	if s.Side.IsLong() != takeProfit {
		r.Stop = "up"
	}
	r.StopPrice = s.TriggerPrice
	r.StopPriceType = futuresStopPriceType(s.TriggerPriceType)
	return r, nil
}

// futuresStopPriceType returns KuCoin's futures trigger price type of a price type: the index or mark price, or the
// last traded price otherwise
func futuresStopPriceType(pt order.PriceType) string {
	switch pt {
	case order.IndexPrice:
		return "IP"
	case order.MarkPrice:
		return "MP"
	default:
		return "TP"
	}
}

// futuresTakeProfitStopLossRequest returns the Add Take Profit And Stop Loss Order request of the futures limit or market
// order r with the TakeProfit and StopLoss triggers of modes. KuCoin names the triggers by the way the price moves to
// reach them, yet documents triggerStopUpPrice as the take profit price and triggerStopDownPrice as the stop loss price
// for either side. The two readings agree for a buy, whose take profit is above its stop loss, and conflict for a sell;
// until KuCoin's own reading is known, a sell's triggers are refused rather than guessed, as the wrong guess sends both
// triggers already reached
func futuresTakeProfitStopLossRequest(r *FuturesOrderRequest, modes *order.RiskManagementModes) (*FuturesTakeProfitStopLossOrderRequest, error) {
	switch {
	case r.Stop != "":
		return nil, fmt.Errorf("%w: a futures stop order takes no take profit or stop loss triggers", errRiskModeUnsupported)
	case r.Side != order.Buy.Lower():
		return nil, fmt.Errorf("%w: a sell's take profit and stop loss triggers, for which KuCoin's trigger names and their descriptions disagree", errRiskModeUnsupported)
	case r.TimeInForce == order.FillOrKill.String():
		return nil, fmt.Errorf("%w: an order with take profit and stop loss triggers is good till cancelled or immediate or cancel", order.ErrUnsupportedTimeInForce)
	}
	tpsl := &FuturesTakeProfitStopLossOrderRequest{
		ClientOrderID: r.ClientOrderID,
		Side:          r.Side,
		Symbol:        r.Symbol,
		Leverage:      r.Leverage,
		Type:          r.Type,
		ReduceOnly:    r.ReduceOnly,
		MarginMode:    r.MarginMode,
		Price:         r.Price,
		Size:          r.Size,
		TimeInForce:   r.TimeInForce,
		PostOnly:      r.PostOnly,
		ValueQuantity: r.ValueQuantity,
	}
	var err error
	if modes.TakeProfit.Enabled {
		if tpsl.TriggerStopUpPrice, err = takeProfitStopLossTrigger(&modes.TakeProfit, errTakeProfitPriceRequired); err != nil {
			return nil, err
		}
		tpsl.StopPriceType = futuresStopPriceType(modes.TakeProfit.TriggerPriceType)
	}
	if modes.StopLoss.Enabled {
		if tpsl.TriggerStopDownPrice, err = takeProfitStopLossTrigger(&modes.StopLoss, errStopLossPriceRequired); err != nil {
			return nil, err
		}
		stopPriceType := futuresStopPriceType(modes.StopLoss.TriggerPriceType)
		if tpsl.StopPriceType != "" && tpsl.StopPriceType != stopPriceType {
			return nil, fmt.Errorf("%w: a take profit and a stop loss trigger on one price type", errRiskModeUnsupported)
		}
		tpsl.StopPriceType = stopPriceType
	}
	if tpsl.TriggerStopUpPrice != 0 && tpsl.TriggerStopDownPrice != 0 && tpsl.TriggerStopUpPrice <= tpsl.TriggerStopDownPrice {
		return nil, fmt.Errorf("%w: a buy order's take profit price must be above its stop loss price", errOCOPricesInverted)
	}
	return tpsl, nil
}

// takeProfitStopLossTrigger returns the trigger price of a take profit or stop loss of an Add Take Profit And Stop Loss
// Order, refusing a limit price or limit order type, for which the order has no field
func takeProfitStopLossTrigger(m *order.RiskManagement, errPriceRequired error) (float64, error) {
	switch {
	case m.Price <= 0:
		return 0, errPriceRequired
	case m.LimitPrice != 0 || (m.OrderType != order.UnknownType && m.OrderType != order.Market):
		return 0, fmt.Errorf("%w: a take profit or stop loss trigger takes no limit price", errRiskModeUnsupported)
	}
	return m.Price, nil
}

// timeInForceString returns the time in force KuCoin takes for a limit order, or an empty string for its default, good
// till cancelled. Post only is a flag of its own
func timeInForceString(tif order.TimeInForce) (string, error) {
	switch tif &^ order.PostOnly {
	case order.UnknownTIF, order.GoodTillCancel:
		return "", nil
	case order.ImmediateOrCancel:
		return order.ImmediateOrCancel.String(), nil
	case order.FillOrKill:
		return order.FillOrKill.String(), nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
	}
}

// futuresMarginMode returns KuCoin's futures margin mode of a margin type
func futuresMarginMode(m margin.Type) (string, error) {
	switch m {
	case margin.Isolated:
		return "ISOLATED", nil
	case margin.Multi:
		return "CROSS", nil
	default:
		return "", fmt.Errorf("%w: %s", margin.ErrMarginTypeUnsupported, m)
	}
}

// submitSpotOrder places a spot OCO order, a stop order or a limit or market order. KuCoin documents no Classic
// replacement for the abandoned stop and OCO order endpoints, so those orders still go through them
func (e *Exchange) submitSpotOrder(ctx context.Context, s *order.Submit, symbol, side string) (*order.SubmitResponse, error) {
	if s.Type == order.OCO {
		price, stopPrice, limitPrice, err := ocoPrices(s)
		if err != nil {
			return nil, err
		}
		resp, err := e.PlaceOCOOrder(ctx, &PlaceOCOOrderRequest{
			ClientOrderID: s.ClientOrderID,
			Side:          side,
			Symbol:        symbol,
			Price:         price,
			Size:          s.Amount,
			StopPrice:     stopPrice,
			LimitPrice:    limitPrice,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		return ocoSubmitResponse(s, resp.OrderID, price)
	}
	r, stop, stopPrice, err := hfOrderRequest(s, symbol, side)
	if err != nil {
		return nil, err
	}
	if stop != "" {
		resp, err := e.PlaceStopOrder(ctx, &PlaceStopOrderRequest{
			ClientOrderID: r.ClientOrderID,
			Side:          r.Side,
			Symbol:        r.Symbol,
			Type:          r.Type,
			Price:         r.Price,
			Size:          r.Size,
			TimeInForce:   r.TimeInForce,
			PostOnly:      r.PostOnly,
			Funds:         r.Funds,
			StopPrice:     stopPrice,
			TradeType:     SpotTradeType,
			Stop:          stop,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		return stopSubmitResponse(s, resp.OrderID, r.Type, stopPrice)
	}
	resp, err := e.PlaceHFOrder(ctx, r)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return s.DeriveSubmitResponse(resp.OrderID)
}

// submitMarginOrder places a margin OCO order, a stop order or a limit or market order, in the isolated margin account
// for an isolated margin submission and in the cross margin account otherwise
func (e *Exchange) submitMarginOrder(ctx context.Context, s *order.Submit, symbol, side string) (*order.SubmitResponse, error) {
	isolated := s.MarginType == margin.Isolated
	if s.Type == order.OCO {
		price, stopPrice, limitPrice, err := ocoPrices(s)
		if err != nil {
			return nil, err
		}
		resp, err := e.PlaceMarginOCOOrder(ctx, &PlaceMarginOCOOrderRequest{
			ClientOrderID: s.ClientOrderID,
			Side:          side,
			Symbol:        symbol,
			Price:         price,
			Size:          s.Amount,
			StopPrice:     stopPrice,
			LimitPrice:    limitPrice,
			IsIsolated:    isolated,
			AutoRepay:     s.AutoRepay,
			AutoBorrow:    s.AutoBorrow,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		return ocoSubmitResponse(s, resp.OrderID, price)
	}
	r, stop, stopPrice, err := hfOrderRequest(s, symbol, side)
	if err != nil {
		return nil, err
	}
	if stop != "" {
		resp, err := e.PlaceMarginStopOrder(ctx, &PlaceMarginStopOrderRequest{
			ClientOrderID: r.ClientOrderID,
			Side:          r.Side,
			Symbol:        r.Symbol,
			Type:          r.Type,
			Price:         r.Price,
			Size:          r.Size,
			TimeInForce:   r.TimeInForce,
			PostOnly:      r.PostOnly,
			Funds:         r.Funds,
			IsIsolated:    isolated,
			AutoBorrow:    s.AutoBorrow,
			AutoRepay:     s.AutoRepay,
			Stop:          stop,
			StopPrice:     stopPrice,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, common.ErrNoResponse
		}
		return stopSubmitResponse(s, resp.OrderID, r.Type, stopPrice)
	}
	resp, err := e.PlaceMarginHFOrder(ctx, &PlaceMarginHFOrderRequest{
		ClientOrderID: r.ClientOrderID,
		Side:          r.Side,
		Symbol:        r.Symbol,
		Type:          r.Type,
		Price:         r.Price,
		Size:          r.Size,
		TimeInForce:   r.TimeInForce,
		PostOnly:      r.PostOnly,
		Funds:         r.Funds,
		IsIsolated:    isolated,
		AutoBorrow:    s.AutoBorrow,
		AutoRepay:     s.AutoRepay,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	submitted, err := s.DeriveSubmitResponse(resp.OrderID)
	if err != nil {
		return nil, err
	}
	submitted.BorrowSize = resp.BorrowSize.Float64()
	submitted.LoanApplyID = resp.LoanApplyID
	return submitted, nil
}

// hfOrderRequest returns the spot or margin limit or market order a submission describes, with the stop condition,
// loss or entry, and the trigger price of a stop order, which its StopLoss or StopEntry risk management mode gives
func hfOrderRequest(s *order.Submit, symbol, side string) (r *PlaceHFOrderRequest, stop string, stopPrice float64, err error) {
	var orderType string
	switch s.Type {
	case order.Limit, order.StopLimit:
		orderType = order.Limit.Lower()
	case order.Market, order.StopMarket:
		orderType = order.Market.Lower()
	default:
		return nil, "", 0, fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, s.Type)
	}
	switch {
	case s.RiskManagementModes.StopLoss.Enabled && s.RiskManagementModes.StopEntry.Enabled:
		return nil, "", 0, errStopModesExclusive
	case s.RiskManagementModes.StopEntry.Enabled:
		stop, stopPrice = "entry", s.RiskManagementModes.StopEntry.Price
	case s.RiskManagementModes.StopLoss.Enabled:
		stop, stopPrice = "loss", s.RiskManagementModes.StopLoss.Price
	case s.Type == order.StopLimit || s.Type == order.StopMarket:
		return nil, "", 0, fmt.Errorf("%w: a stop order takes its trigger from a StopLoss or StopEntry risk management mode", errStopPriceRequired)
	}
	r = &PlaceHFOrderRequest{
		ClientOrderID: s.ClientOrderID,
		Side:          side,
		Symbol:        symbol,
		Type:          orderType,
		Size:          s.Amount,
	}
	if orderType == order.Market.Lower() {
		r.Funds = s.QuoteAmount
		return r, stop, stopPrice, nil
	}
	if r.TimeInForce, err = timeInForceString(s.TimeInForce); err != nil {
		return nil, "", 0, err
	}
	r.Price = s.Price
	r.PostOnly = s.TimeInForce.Is(order.PostOnly)
	return r, stop, stopPrice, nil
}

// stopSubmitResponse returns the response to a submission placed as a stop order, of KuCoin order type orderType,
// which triggers at stopPrice, taken from the submission's risk management modes. Until it triggers it is a stop limit
// or stop market order, whatever type the submission named, and has not filled
func stopSubmitResponse(s *order.Submit, orderID, orderType string, stopPrice float64) (*order.SubmitResponse, error) {
	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Type, resp.Status, resp.TriggerPrice = order.StopLimit, order.New, stopPrice
	if orderType == order.Market.Lower() {
		resp.Type = order.StopMarket
	}
	return resp, nil
}

// ocoSubmitResponse returns the response to a submission placed as an OCO order whose limit order is at price, which
// the submission's Price may leave unset
func ocoSubmitResponse(s *order.Submit, orderID string, price float64) (*order.SubmitResponse, error) {
	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Price = price
	return resp, nil
}

// ocoPrices returns the prices of the OCO order a submission describes: its limit order's price, the TakeProfit price,
// and its stop order's trigger and limit prices, the StopLoss price and LimitPrice. The take profit must be on the
// order's favourable side of the stop loss. KuCoin's OCO stop order is always a limit order, and the limit order holds
// the order's only other price, so a Price that differs from the TakeProfit price is refused rather than dropped
func ocoPrices(s *order.Submit) (price, stopPrice, limitPrice float64, err error) {
	r := &s.RiskManagementModes
	switch {
	case !r.TakeProfit.Enabled || r.TakeProfit.Price <= 0:
		return 0, 0, 0, errTakeProfitPriceRequired
	case s.Price != 0 && s.Price != r.TakeProfit.Price:
		return 0, 0, 0, fmt.Errorf("%w: price %v, take profit price %v", errOCOPriceMismatch, s.Price, r.TakeProfit.Price)
	case !r.StopLoss.Enabled || r.StopLoss.Price <= 0:
		return 0, 0, 0, errStopLossPriceRequired
	case r.StopLoss.LimitPrice <= 0:
		return 0, 0, 0, fmt.Errorf("%w: an OCO order's stop order is a limit order at the StopLoss LimitPrice", errLimitPriceRequired)
	case s.Side.IsShort() && r.TakeProfit.Price <= r.StopLoss.Price:
		return 0, 0, 0, fmt.Errorf("%w: a sell order's stop loss price must be below its take profit price", errOCOPricesInverted)
	case s.Side.IsLong() && r.TakeProfit.Price >= r.StopLoss.Price:
		return 0, 0, 0, fmt.Errorf("%w: a buy order's stop loss price must be above its take profit price", errOCOPricesInverted)
	}
	return r.TakeProfit.Price, r.StopLoss.Price, r.StopLoss.LimitPrice, nil
}

// ModifyOrder changes the price or amount of a spot order over REST. KuCoin cancels the order and places a new one,
// whose order ID the response carries, with its price, remaining size and status as KuCoin reports them or, when the
// new order cannot be read back, its price as requested; it cannot modify margin or futures orders. The amount is the
// one requested, as KuCoin sizes the new order by what the requested amount leaves after the original order's fills,
// which the order's executed amount still holds
func (e *Exchange) ModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	pFmt, err := e.requestFormat(m.AssetType)
	if err != nil {
		return nil, err
	}
	if m.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w: KuCoin modifies spot orders only, not %s orders", asset.ErrNotSupported, m.AssetType)
	}
	r := &ModifyHFOrderRequest{
		Symbol:   pFmt.Format(m.Pair),
		NewPrice: m.Price,
		NewSize:  m.Amount,
	}
	if m.OrderID != "" {
		r.OrderID = m.OrderID
	} else {
		r.ClientOrderID = m.ClientOrderID
	}
	resp, err := e.ModifyHFOrder(ctx, r)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	modified, err := m.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	modified.OrderID = resp.NewOrderID
	modified.ClientOrderID = resp.ClientOrderID
	// KuCoin has replaced the order by now, so failing to read the new order back must not hide its order ID
	if placed, err := e.GetHFOrderByOrderID(ctx, resp.NewOrderID, r.Symbol); err == nil {
		modified.Price = placed.Price.Float64()
		modified.RemainingAmount = placed.RemainSize.Float64()
		modified.Status = orderStatus(placed.Active, placed.CancelExist, placed.DealSize.Float64())
	}
	return modified, nil
}

// CancelOrder cancels an order over REST by its order ID, or by its client order ID when it has none. Futures orders
// cancelled by client order ID, and spot and margin orders other than stop and OCO orders, need the pair. KuCoin has
// abandoned the endpoints that cancel one spot stop order, so spot stop orders are cancelled through Batch Cancel
// Stop Orders, which takes order IDs only
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(); err != nil {
		return err
	}
	if ord.OrderID == "" && ord.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	pFmt, err := e.requestFormat(ord.AssetType)
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Futures:
		if ord.OrderID != "" {
			_, err = e.CancelFuturesOrderByOrderID(ctx, ord.OrderID)
			return err
		}
	case asset.Margin:
		switch {
		case ord.Type == order.OCO && ord.OrderID != "":
			_, err = e.CancelMarginOCOOrderByOrderID(ctx, ord.OrderID)
			return err
		case ord.Type == order.OCO:
			_, err = e.CancelMarginOCOOrderByClientOrderID(ctx, ord.ClientOrderID)
			return err
		case isStopOrderType(ord.Type) && ord.OrderID != "":
			_, err = e.CancelMarginStopOrderByOrderID(ctx, ord.OrderID)
			return err
		case isStopOrderType(ord.Type):
			_, err = e.CancelMarginStopOrderByClientOrderID(ctx, ord.ClientOrderID)
			return err
		}
	default: // Spot, the only other asset requestFormat accepts
		switch {
		case ord.Type == order.OCO && ord.OrderID != "":
			_, err = e.CancelOCOOrderByOrderID(ctx, ord.OrderID)
			return err
		case ord.Type == order.OCO:
			_, err = e.CancelOCOOrderByClientOrderID(ctx, ord.ClientOrderID)
			return err
		case isStopOrderType(ord.Type) && ord.OrderID != "":
			return e.cancelSpotStopOrders(ctx, []string{ord.OrderID})
		case isStopOrderType(ord.Type):
			var symbol string
			if !ord.Pair.IsEmpty() {
				symbol = pFmt.Format(ord.Pair)
			}
			stops, err := e.GetStopOrderByClientOrderID(ctx, ord.ClientOrderID, symbol)
			if err != nil {
				return err
			}
			if len(stops) == 0 {
				return fmt.Errorf("%w: client order ID %s", order.ErrOrderNotFound, ord.ClientOrderID)
			}
			orderIDs := make([]string, len(stops))
			for i := range stops {
				orderIDs[i] = stops[i].ID
			}
			return e.cancelSpotStopOrders(ctx, orderIDs)
		}
	}
	if ord.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	symbol := pFmt.Format(ord.Pair)
	switch {
	case ord.AssetType == asset.Futures:
		_, err = e.CancelFuturesOrderByClientOrderID(ctx, symbol, ord.ClientOrderID)
	case ord.AssetType == asset.Margin && ord.OrderID != "":
		_, err = e.CancelMarginHFOrderByOrderID(ctx, ord.OrderID, symbol)
	case ord.AssetType == asset.Margin:
		_, err = e.CancelMarginHFOrderByClientOrderID(ctx, ord.ClientOrderID, symbol)
	case ord.OrderID != "":
		_, err = e.CancelHFOrderByOrderID(ctx, ord.OrderID, symbol)
	default:
		_, err = e.CancelHFOrderByClientOrderID(ctx, ord.ClientOrderID, symbol)
	}
	return err
}

// isStopOrderType returns whether KuCoin keeps orders of a type as stop orders until they trigger
func isStopOrderType(t order.Type) bool {
	switch t {
	case order.Stop, order.StopLimit, order.StopMarket, order.ConditionalStop:
		return true
	default:
		return false
	}
}

// cancelSpotStopOrders cancels spot stop orders by order ID through Batch Cancel Stop Orders, which reports the orders
// it cancelled rather than failing for the others. KuCoin cancels every stop order when no order ID is given, so an
// empty list is refused
func (e *Exchange) cancelSpotStopOrders(ctx context.Context, orderIDs []string) error {
	if len(orderIDs) == 0 {
		return order.ErrOrderIDNotSet
	}
	resp, err := e.BatchCancelStopOrders(ctx, &BatchCancelStopOrdersRequest{TradeType: SpotTradeType, OrderIDs: orderIDs})
	if err != nil {
		return err
	}
	var cancelled []string
	if resp != nil {
		cancelled = resp.CancelledOrderIDs
	}
	for _, orderID := range orderIDs {
		if !slices.Contains(cancelled, orderID) {
			return fmt.Errorf("%w: stop order %s", errOrderNotCancelled, orderID)
		}
	}
	return nil
}

// CancelBatchOrders cancels futures orders over REST through Batch Cancel Orders, ten orders a request, each by its
// order ID or, when it has none, by its pair and client order ID, sending an order listed twice once. Each order's status is order.Cancelled or KuCoin's
// reason for not cancelling it, keyed by the order ID or client order ID KuCoin echoes, or the error of its request
// when that request failed, keyed by the ID the request sent. A failed request does not stop the later ones; the
// statuses are returned with the failed requests' errors joined. KuCoin has no batch cancel for spot or margin orders
func (e *Exchange) CancelBatchOrders(ctx context.Context, cancels []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(cancels) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	pFmt, err := e.requestFormat(asset.Futures)
	if err != nil {
		return nil, err
	}
	var orderIDs []string
	var clientOrders []BatchCancelFuturesClientOrder
	for i := range cancels {
		switch {
		case cancels[i].AssetType != asset.Futures:
			return nil, fmt.Errorf("%w: KuCoin batch cancels futures orders only, not %s orders", asset.ErrNotSupported, cancels[i].AssetType)
		case cancels[i].OrderID != "":
			// An order listed twice would go in two requests, and a later failure would overwrite its cancellation
			if !slices.Contains(orderIDs, cancels[i].OrderID) {
				orderIDs = append(orderIDs, cancels[i].OrderID)
			}
		case cancels[i].ClientOrderID == "":
			return nil, order.ErrOrderIDNotSet
		case cancels[i].Pair.IsEmpty():
			return nil, currency.ErrCurrencyPairEmpty
		default:
			clientOrder := BatchCancelFuturesClientOrder{Symbol: pFmt.Format(cancels[i].Pair), ClientOrderID: cancels[i].ClientOrderID}
			if !slices.Contains(clientOrders, clientOrder) {
				clientOrders = append(clientOrders, clientOrder)
			}
		}
	}
	// KuCoin ignores the client orders of a request that lists order IDs, so each kind is cancelled in requests of its own
	requests := make([]*BatchCancelFuturesOrdersRequest, 0, len(cancels)/batchCancelFuturesOrdersLimit+2)
	for _, batch := range common.Batch(orderIDs, batchCancelFuturesOrdersLimit) {
		requests = append(requests, &BatchCancelFuturesOrdersRequest{OrderIDs: batch})
	}
	for _, batch := range common.Batch(clientOrders, batchCancelFuturesOrdersLimit) {
		requests = append(requests, &BatchCancelFuturesOrdersRequest{ClientOrders: batch})
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(cancels))}
	var errs error
	for _, r := range requests {
		results, err := e.BatchCancelFuturesOrders(ctx, r)
		if err != nil {
			errs = common.AppendError(errs, err)
			for _, orderID := range r.OrderIDs {
				resp.Status[orderID] = err.Error()
			}
			for i := range r.ClientOrders {
				resp.Status[r.ClientOrders[i].ClientOrderID] = err.Error()
			}
			continue
		}
		sent := slices.Clone(r.OrderIDs)
		for i := range r.ClientOrders {
			sent = append(sent, r.ClientOrders[i].ClientOrderID)
		}
		named := make(map[string]struct{}, len(results))
		for i := range results {
			id := results[i].OrderID
			if id == "" {
				id = results[i].ClientOrderID
			}
			if !slices.Contains(sent, id) {
				continue // A result naming no order this request sent reports nothing about its orders
			}
			named[id] = struct{}{}
			status := results[i].Message
			if results[i].Code == "200" {
				status = order.Cancelled.String()
			}
			resp.Status[id] = status
		}
		// An order the reply leaves out was not reported cancelled
		for _, id := range sent {
			if _, ok := named[id]; !ok {
				err := fmt.Errorf("%w for order %s", common.ErrNoResponse, id)
				errs = common.AppendError(errs, err)
				resp.Status[id] = err.Error()
			}
		}
	}
	return resp, errs
}

// batchCancelFuturesOrdersLimit is the most orders a Batch Cancel Orders request takes
const batchCancelFuturesOrdersLimit = 10

// CancelAllOrders cancels an asset's orders over REST, those of one pair when the request names one. Futures limit and
// stop orders are cancelled together. Spot and margin stop or OCO orders are cancelled for a stop or OCO request, only
// its order when it names one. KuCoin returns no order IDs for the other spot and margin orders it cancels, so their
// statuses are keyed by symbol. KuCoin cancels margin and futures limit orders by symbol, so without a pair those of
// each symbol with open orders are cancelled, and when a later request fails the statuses of the orders earlier ones
// cancelled are returned with its error. Margin orders are those of the request's margin type, cross margin unless it
// is isolated
func (e *Exchange) CancelAllOrders(ctx context.Context, c *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	if err := c.Validate(); err != nil {
		return resp, err
	}
	pFmt, err := e.requestFormat(c.AssetType)
	if err != nil {
		return resp, err
	}
	var symbol string
	if !c.Pair.IsEmpty() {
		symbol = pFmt.Format(c.Pair)
	}
	var orderIDs []string
	if c.OrderID != "" {
		orderIDs = []string{c.OrderID}
	}
	cancelled := order.Cancelled.String()
	tradeType := e.AccountToTradeTypeString(c.AssetType, c.MarginType)
	var cancelledIDs []string
	switch c.AssetType {
	case asset.Futures:
		symbols := []string{symbol}
		if symbol == "" {
			active, err := e.allFuturesOrders(ctx, &GetFuturesOrdersRequest{Status: "active"})
			if err != nil {
				return resp, err
			}
			symbols = nil
			for i := range active {
				if !slices.Contains(symbols, active[i].Symbol) {
					symbols = append(symbols, active[i].Symbol)
				}
			}
		}
		// Orders earlier requests cancelled stay cancelled when a later request fails, so their statuses go with its error
		for _, s := range symbols {
			limitOrders, err := e.CancelAllFuturesOrders(ctx, s)
			if err != nil {
				return addCancelledOrders(resp, cancelledIDs), err
			}
			if limitOrders != nil {
				cancelledIDs = append(cancelledIDs, limitOrders.CancelledOrderIDs...)
			}
		}
		stopOrders, err := e.CancelAllFuturesStopOrders(ctx, symbol)
		if err != nil {
			return addCancelledOrders(resp, cancelledIDs), err
		}
		if stopOrders != nil {
			cancelledIDs = append(cancelledIDs, stopOrders.CancelledOrderIDs...)
		}
	case asset.Margin:
		switch {
		case c.Type == order.OCO:
			ocoOrders, err := e.CancelMarginOCOOrders(ctx, &CancelMarginOCOOrdersRequest{OrderIDs: orderIDs, Symbol: symbol, TradeType: tradeType})
			if err != nil {
				return resp, err
			}
			if ocoOrders != nil {
				cancelledIDs = ocoOrders.CancelledOrderIDs
			}
		case isStopOrderType(c.Type):
			stopOrders, err := e.CancelMarginStopOrders(ctx, &CancelMarginStopOrdersRequest{Symbol: symbol, TradeType: tradeType, OrderIDs: orderIDs})
			if err != nil {
				return resp, err
			}
			if stopOrders != nil {
				cancelledIDs = stopOrders.CancelledOrderIDs
			}
		default:
			symbols := []string{symbol}
			if symbol == "" {
				if symbols, err = e.openOrderSymbols(ctx, asset.Margin, tradeType); err != nil {
					return resp, err
				}
			}
			for _, s := range symbols {
				if _, err := e.CancelAllMarginHFOrdersBySymbol(ctx, s, tradeType); err != nil {
					return resp, err
				}
				resp.Add(s, cancelled)
			}
		}
	default: // Spot, the only other asset requestFormat accepts
		switch {
		case c.Type == order.OCO:
			ocoOrders, err := e.CancelOCOMultipleOrders(ctx, orderIDs, symbol)
			if err != nil {
				return resp, err
			}
			if ocoOrders != nil {
				cancelledIDs = ocoOrders.CancelledOrderIDs
			}
		case isStopOrderType(c.Type):
			stopOrders, err := e.BatchCancelStopOrders(ctx, &BatchCancelStopOrdersRequest{Symbol: symbol, TradeType: tradeType, OrderIDs: orderIDs})
			if err != nil {
				return resp, err
			}
			if stopOrders != nil {
				cancelledIDs = stopOrders.CancelledOrderIDs
			}
		case symbol != "":
			if _, err := e.CancelAllHFOrdersBySymbol(ctx, symbol); err != nil {
				return resp, err
			}
			resp.Add(symbol, cancelled)
		default:
			symbols, err := e.CancelAllHFOrders(ctx)
			if err != nil {
				return resp, err
			}
			if symbols != nil {
				for _, s := range symbols.SucceedSymbols {
					resp.Add(s, cancelled)
				}
				for i := range symbols.FailedSymbols {
					resp.Add(symbols.FailedSymbols[i].Symbol, symbols.FailedSymbols[i].Error)
				}
			}
		}
	}
	return addCancelledOrders(resp, cancelledIDs), nil
}

// addCancelledOrders adds the orders KuCoin reported cancelled to a CancelAllOrders response
func addCancelledOrders(resp order.CancelAllResponse, orderIDs []string) order.CancelAllResponse {
	for _, orderID := range orderIDs {
		resp.Add(orderID, order.Cancelled.String())
	}
	return resp
}

// openOrderSymbols returns the symbols KuCoin reports open spot orders for, or open margin orders of a trade type
func (e *Exchange) openOrderSymbols(ctx context.Context, a asset.Item, tradeType string) ([]string, error) {
	if a == asset.Margin {
		resp, err := e.GetActiveMarginHFOrderSymbols(ctx, tradeType)
		if err != nil || resp == nil {
			return nil, err
		}
		return resp.Symbols, nil
	}
	resp, err := e.GetSymbolsWithOpenHFOrder(ctx)
	if err != nil || resp == nil {
		return nil, err
	}
	return resp.Symbols, nil
}

// GetOrderInfo returns an order by its order ID over REST. Spot and margin orders need their pair, by which KuCoin
// looks them up; KuCoin keeps cancelled spot orders for two days and filled ones for seven. KuCoin looks up spot and
// margin stop and OCO orders apart from the others, so an order the order lookup does not find is looked up as one
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	pFmt, err := e.requestFormat(a)
	if err != nil {
		return nil, err
	}
	var d order.Detail
	switch a {
	case asset.Futures:
		resp, err := e.GetFuturesOrderDetailsByOrderID(ctx, orderID)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
		}
		if d, err = e.futuresOrderDetail(resp); err != nil {
			return nil, err
		}
	case asset.Margin:
		if pair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		resp, err := e.GetMarginHFOrderDetailByOrderID(ctx, orderID, pFmt.Format(pair))
		if err != nil {
			if d, err = e.conditionalOrderInfo(ctx, orderID, a, err); err != nil {
				return nil, err
			}
			break
		}
		if d, err = e.marginHFOrderDetail(&resp.MarginHFOrderResponse); err != nil {
			return nil, err
		}
	default: // Spot, the only other asset requestFormat accepts
		if pair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		resp, err := e.GetHFOrderByOrderID(ctx, orderID, pFmt.Format(pair))
		if err != nil {
			if d, err = e.conditionalOrderInfo(ctx, orderID, a, err); err != nil {
				return nil, err
			}
			break
		}
		if d, err = e.hfOrderDetail(&resp.HFOrderResponse); err != nil {
			return nil, err
		}
	}
	if !pair.IsEmpty() && !d.Pair.Equal(pair) {
		return nil, fmt.Errorf("%w: order %s is a %s order", errOrderPairMismatch, orderID, d.Pair)
	}
	return &d, nil
}

// conditionalOrderInfo returns the spot or margin stop or OCO order an order ID names, after the order lookup failed
// with lookupErr. KuCoin answers an order ID it does not hold with no data or with an error response, so either sends
// the ID to the stop order lookup and then the OCO order lookup, and lookupErr is returned when neither holds it
func (e *Exchange) conditionalOrderInfo(ctx context.Context, orderID string, a asset.Item, lookupErr error) (order.Detail, error) {
	if !lookupMissed(lookupErr) {
		return order.Detail{}, lookupErr
	}
	var stop *StopOrderResponse
	var err error
	if a == asset.Margin {
		var resp *MarginStopOrderResponse
		resp, err = e.GetMarginStopOrderByOrderID(ctx, orderID)
		// Margin and spot stop orders hold the same fields
		stop = (*StopOrderResponse)(resp)
	} else {
		stop, err = e.GetStopOrderByOrderID(ctx, orderID)
	}
	switch {
	case err != nil && !lookupMissed(err):
		return order.Detail{}, err
	case err == nil && stop != nil:
		return e.stopOrderDetail(stop, a, strings.EqualFold(stop.Stop, "oco"))
	}
	var oco *OCOOrder
	if a == asset.Margin {
		var resp *MarginOCOOrderResponse
		if resp, err = e.GetMarginOCOOrderByOrderID(ctx, orderID); resp != nil {
			oco = &OCOOrder{OrderID: resp.OrderID, Symbol: resp.Symbol, ClientOrderID: resp.ClientOrderID, OrderTime: resp.OrderTime, Status: resp.Status}
		}
	} else {
		// KuCoin abandoned its spot OCO order lookups along with the order endpoints the spot OCO orders still use, and the
		// list, which the package already uses for them, finds an order by its ID as well
		var resp *OCOOrderListResponse
		if resp, err = e.GetOCOOrderList(ctx, &OCOOrderListRequest{OrderIDs: []string{orderID}}); resp != nil {
			for i := range resp.Items {
				if resp.Items[i].OrderID == orderID {
					oco = &resp.Items[i]
					break
				}
			}
		}
	}
	switch {
	case err != nil && !lookupMissed(err):
		return order.Detail{}, err
	case err == nil && oco != nil:
		return e.ocoOrderDetail(oco, a)
	}
	return order.Detail{}, lookupErr
}

// lookupMissed returns whether an order lookup's error may mean KuCoin holds no order of the ID it looked up: it
// returned no data, or KuCoin answered with an error, as it does for an order ID it does not know
func lookupMissed(err error) bool {
	var respErr *responseError
	return errors.Is(err, order.ErrOrderNotFound) || errors.As(err, &respErr)
}

// hfOrderDetail converts a spot order
func (e *Exchange) hfOrderDetail(o *HFOrderResponse) (order.Detail, error) {
	pair, err := currency.NewPairFromString(o.Symbol)
	if err != nil {
		return order.Detail{}, err
	}
	oType, err := orderType(o.Type, false)
	if err != nil {
		return order.Detail{}, err
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return order.Detail{}, err
	}
	d := order.Detail{
		Exchange:        e.Name,
		OrderID:         o.ID,
		ClientOrderID:   o.ClientOrderID,
		Pair:            pair,
		AssetType:       asset.Spot,
		Type:            oType,
		Side:            side,
		Status:          orderStatus(o.Active, o.CancelExist, o.DealSize.Float64()),
		Price:           o.Price.Float64(),
		Amount:          o.Size.Float64(),
		QuoteAmount:     o.Funds.Float64(),
		ExecutedAmount:  o.DealSize.Float64(),
		RemainingAmount: o.RemainSize.Float64(),
		Cost:            o.DealFunds.Float64(),
		CostAsset:       pair.Quote,
		Fee:             o.Fee.Float64(),
		FeeAsset:        o.FeeCurrency,
		TimeInForce:     StringToTimeInForce(o.TimeInForce, o.PostOnly),
		HiddenOrder:     o.Hidden,
		Date:            o.CreatedAt.Time(),
		LastUpdated:     o.LastUpdatedAt.Time(),
	}
	if !o.Active {
		d.CloseTime = d.LastUpdated
	}
	if d.ExecutedAmount > 0 {
		d.AverageExecutedPrice = d.Cost / d.ExecutedAmount
	}
	return d, nil
}

// marginHFOrderDetail converts a margin order
func (e *Exchange) marginHFOrderDetail(o *MarginHFOrderResponse) (order.Detail, error) {
	pair, err := currency.NewPairFromString(o.Symbol)
	if err != nil {
		return order.Detail{}, err
	}
	// A triggered stop order carries on as the limit or market order it placed, which the order pushes report it as
	untriggered := o.Stop != "" && !o.StopTriggered
	oType, err := orderType(o.Type, untriggered)
	if err != nil {
		return order.Detail{}, err
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return order.Detail{}, err
	}
	status := orderStatus(o.Active, o.CancelExist, o.DealSize.Float64())
	if untriggered && status == order.Open {
		// A stop order waiting for its trigger is NEW, as Get Stop Order and the stop order pushes report it
		status = order.New
	}
	d := order.Detail{
		Exchange:        e.Name,
		OrderID:         o.ID,
		ClientOrderID:   o.ClientOrderID,
		Pair:            pair,
		AssetType:       asset.Margin,
		MarginType:      tradeTypeMarginType(o.TradeType),
		Type:            oType,
		Side:            side,
		Status:          status,
		Price:           o.Price.Float64(),
		TriggerPrice:    o.StopPrice.Float64(),
		Amount:          o.Size.Float64(),
		QuoteAmount:     o.Funds.Float64(),
		ExecutedAmount:  o.DealSize.Float64(),
		RemainingAmount: o.RemainSize.Float64(),
		Cost:            o.DealFunds.Float64(),
		CostAsset:       pair.Quote,
		Fee:             o.Fee.Float64(),
		FeeAsset:        o.FeeCurrency,
		TimeInForce:     StringToTimeInForce(o.TimeInForce, o.PostOnly),
		HiddenOrder:     o.Hidden,
		Date:            o.CreatedAt.Time(),
		LastUpdated:     o.LastUpdatedAt.Time(),
	}
	switch {
	case !o.Active:
		e.marginOrderIDs.Delete(o.ID)
		if o.ClientOrderID != "" {
			// A reply still to come for its placement must not remember the finished order
			e.notePlacingMarginOrder(o.ClientOrderID, o.ID, true)
		}
		d.CloseTime = d.LastUpdated
	case untriggered:
		e.marginOrderIDs.LoadOrStore(o.ID, false)
	default:
		// Read past any trigger, so a triggered push arriving later is not relayed over it
		e.marginOrderIDs.Store(o.ID, true)
	}
	if d.ExecutedAmount > 0 {
		d.AverageExecutedPrice = d.Cost / d.ExecutedAmount
	}
	return d, nil
}

// takeProfitType returns the take profit type of a futures stop order that triggers as the price moves in its favour, up
// for a sell and down for a buy, as futuresOrderRequest places a take profit order; a stop order triggering as the price
// moves against it keeps its type
func takeProfitType(oType order.Type, stop string, side order.Side) order.Type {
	if (stop == "up" && side.IsShort()) || (stop == "down" && side.IsLong()) {
		switch oType {
		case order.StopLimit:
			return order.TakeProfit
		case order.StopMarket:
			return order.TakeProfitMarket
		}
	}
	return oType
}

// futuresFinalPushWait is how long after a futures order ends its final pushes are waited for: its final Orders push,
// which reports a triggered stop order by the type its stop push or a REST read gave it, and its stop pushes, which come
// on a topic of their own and can arrive after its final Orders push
const futuresFinalPushWait = time.Minute

// futuresOrderDetail converts a futures order, whose amounts are in lots
func (e *Exchange) futuresOrderDetail(o *FuturesOrderDetailResponse) (order.Detail, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(o.Symbol, asset.Futures, false)
	if err != nil {
		return order.Detail{}, err
	}
	oType, err := orderType(o.Type, o.Stop != "")
	if err != nil {
		return order.Detail{}, err
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return order.Detail{}, err
	}
	oType = takeProfitType(oType, o.Stop, side)
	switch {
	case !o.IsActive && time.Since(o.EndAt.Time()) > futuresFinalPushWait:
		e.futuresStopOrderTypes.Delete(o.ID)
	case o.Stop != "":
		// An order that has only just ended keeps its type for its final Orders push, which may still be on its way
		e.rememberFuturesStopOrderRead(o.ID, oType, !o.IsActive)
	}
	marginType := margin.Isolated
	if o.MarginMode == "CROSS" {
		marginType = margin.Multi
	}
	return order.Detail{
		Exchange:             e.Name,
		OrderID:              o.ID,
		ClientOrderID:        o.ClientOrderID,
		Pair:                 pair,
		AssetType:            asset.Futures,
		MarginType:           marginType,
		Type:                 oType,
		Side:                 side,
		Status:               orderStatus(o.IsActive, o.CancelExist, float64(o.DealSize)),
		Price:                o.Price.Float64(),
		TriggerPrice:         o.StopPrice.Float64(),
		Amount:               float64(o.Size),
		ContractAmount:       float64(o.Size),
		ExecutedAmount:       float64(o.DealSize),
		RemainingAmount:      float64(o.Size) - float64(o.DealSize),
		AverageExecutedPrice: o.AverageDealPrice.Float64(),
		Cost:                 o.DealValue.Float64(),
		CostAsset:            o.SettleCurrency,
		SettlementCurrency:   o.SettleCurrency,
		Leverage:             o.Leverage.Float64(),
		ReduceOnly:           o.ReduceOnly,
		TimeInForce:          StringToTimeInForce(o.TimeInForce, o.PostOnly),
		HiddenOrder:          o.Hidden,
		Date:                 o.CreatedAt.Time(),
		LastUpdated:          o.UpdatedAt.Time(),
		CloseTime:            o.EndAt.Time(),
	}, nil
}

// stopOrderDetail converts a spot or margin stop order, or an OCO order, which KuCoin lists with them. A triggered stop
// order, which only a lookup by its order ID returns, has placed its order and is closed
func (e *Exchange) stopOrderDetail(o *StopOrderResponse, a asset.Item, oco bool) (order.Detail, error) {
	pair, err := currency.NewPairFromString(o.Symbol)
	if err != nil {
		return order.Detail{}, err
	}
	oType := order.OCO
	if !oco {
		if oType, err = orderType(o.Type, true); err != nil {
			return order.Detail{}, err
		}
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return order.Detail{}, err
	}
	status := order.New
	if strings.EqualFold(o.Status, "TRIGGERED") {
		status = order.Closed
	}
	if a == asset.Margin && !oco && status == order.New {
		// A margin stop order places its order under its own order ID once triggered, which the Order V2 pushes follow
		e.marginOrderIDs.LoadOrStore(o.ID, false)
	}
	return order.Detail{
		Exchange:      e.Name,
		OrderID:       o.ID,
		ClientOrderID: o.ClientOrderID,
		Pair:          pair,
		AssetType:     a,
		MarginType:    tradeTypeMarginType(o.TradeType),
		Type:          oType,
		Side:          side,
		Status:        status,
		Price:         o.Price.Float64(),
		TriggerPrice:  o.StopPrice.Float64(),
		Amount:        o.Size.Float64(),
		QuoteAmount:   o.Funds.Float64(),
		TimeInForce:   StringToTimeInForce(o.TimeInForce, o.PostOnly),
		HiddenOrder:   o.Hidden,
		Date:          o.CreatedAt.Time(),
		LastUpdated:   o.OrderTime.Time(),
	}, nil
}

// ocoOrderDetail converts an OCO order of KuCoin's OCO order lists
func (e *Exchange) ocoOrderDetail(o *OCOOrder, a asset.Item) (order.Detail, error) {
	pair, err := currency.NewPairFromString(o.Symbol)
	if err != nil {
		return order.Detail{}, err
	}
	var status order.Status
	switch strings.ToUpper(o.Status) {
	case "NEW":
		status = order.New
	case "DONE", "TRIGGERED":
		status = order.Closed
	case "CANCELED", "CANCELLED":
		status = order.Cancelled
	default:
		return order.Detail{}, fmt.Errorf("%w: %q", order.ErrUnsupportedStatusType, o.Status)
	}
	return order.Detail{
		Exchange:      e.Name,
		OrderID:       o.OrderID,
		ClientOrderID: o.ClientOrderID,
		Pair:          pair,
		AssetType:     a,
		Type:          order.OCO,
		Status:        status,
		Date:          o.OrderTime.Time(),
		LastUpdated:   o.OrderTime.Time(),
	}, nil
}

// orderType returns the order type of KuCoin's limit or market order type, a stop order type for a stop order
func orderType(kind string, stop bool) (order.Type, error) {
	switch strings.ToLower(kind) {
	case order.Limit.Lower():
		if stop {
			return order.StopLimit, nil
		}
		return order.Limit, nil
	case order.Market.Lower():
		if stop {
			return order.StopMarket, nil
		}
		return order.Market, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", order.ErrUnrecognisedOrderType, kind)
	}
}

// orderStatus returns the status of an order from whether KuCoin still has it active, whether it has a cancellation
// record and how much of it filled
func orderStatus(active, cancelled bool, executed float64) order.Status {
	switch {
	case active && executed > 0:
		return order.PartiallyFilled
	case active:
		return order.Open
	case cancelled && executed > 0:
		return order.PartiallyFilledCancelled
	case cancelled:
		return order.Cancelled
	default:
		return order.Filled
	}
}

// tradeTypeMarginType returns the margin type of KuCoin's trade type: isolated or cross for margin, unset for spot
func tradeTypeMarginType(tradeType string) margin.Type {
	switch tradeType {
	case IsolatedMarginTradeType:
		return margin.Isolated
	case CrossMarginTradeType:
		return margin.Multi
	default:
		return margin.Unset
	}
}

// GetDepositAddress returns a currency's deposit address, on the chain given by its chain ID (as
// GetAvailableTransferChains returns them) or name, or its only address when no chain is given
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	addresses, err := e.GetDepositAddressV3(ctx, c, "")
	if err != nil {
		return nil, err
	}
	if chain != "" {
		for i := range addresses {
			if strings.EqualFold(addresses[i].ChainID, chain) || strings.EqualFold(addresses[i].ChainName, chain) {
				return &deposit.Address{Address: addresses[i].Address, Tag: addresses[i].Memo, Chain: addresses[i].ChainID}, nil
			}
		}
		return nil, fmt.Errorf("%w on chain %s", errNoDepositAddress, chain)
	}
	switch len(addresses) {
	case 0:
		return nil, errNoDepositAddress
	case 1:
		return &deposit.Address{Address: addresses[0].Address, Tag: addresses[0].Memo, Chain: addresses[0].ChainID}, nil
	default:
		return nil, errMultipleDepositAddress
	}
}

// WithdrawCryptocurrencyFunds withdraws to an address, deducting the fee from the amount, and returns the withdrawal
// ID. The chain is a chain ID, as GetAvailableTransferChains returns them. Futures funds must first be transferred to
// the main account
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.WithdrawV3(ctx, &WithdrawV3Request{
		Currency:      withdrawRequest.Currency,
		Chain:         withdrawRequest.Crypto.Chain,
		Amount:        withdrawRequest.Amount,
		Memo:          withdrawRequest.Crypto.AddressTag,
		IsInner:       withdrawRequest.InternalTransfer,
		Remark:        withdrawRequest.Description,
		FeeDeductType: "INTERNAL",
		ToAddress:     withdrawRequest.Crypto.Address,
		WithdrawType:  "ADDRESS",
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return &withdraw.ExchangeResponse{ID: resp.WithdrawalID}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// OrderTypeToString returns an order type instance from string.
func OrderTypeToString(oType order.Type) (string, error) {
	switch oType {
	case order.Limit:
		return "limit", nil
	case order.Market:
		return "market", nil
	case order.StopLimit:
		return "limit_stop", nil
	case order.StopMarket:
		return "market_stop", nil
	case order.AnyType, order.UnknownType:
		return "", nil
	default:
		return "", order.ErrUnsupportedOrderType
	}
}

// Page sizes of KuCoin's order lists
const (
	futuresOrdersPageSize = 1000
	stopOrdersPageSize    = 500
	ocoOrdersPageSize     = 500
	closedOrdersLimit     = 100
)

// collectPages gathers the items of each page of a paged list, from page 1 to the last page fetch reports
func collectPages[T any](fetch func(page uint64) (items []T, totalPages uint64, err error)) ([]T, error) {
	var all []T
	for page := uint64(1); ; page++ {
		items, totalPages, err := fetch(page)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if page >= totalPages {
			return all, nil
		}
	}
}

// collectByLastID gathers the items of each page of a list that pages back from the ID of the previous page's last
// item, until a page holds fewer than closedOrdersLimit items or the ID stops changing
func collectByLastID[T any](fetch func(lastID uint64) (items []T, nextLastID uint64, err error)) ([]T, error) {
	var all []T
	var lastID uint64
	for {
		items, nextLastID, err := fetch(lastID)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < closedOrdersLimit || nextLastID == lastID {
			return all, nil
		}
		lastID = nextLastID
	}
}

// allFuturesOrders returns the futures orders of every page a Get Order List request matches, setting its paging
func (e *Exchange) allFuturesOrders(ctx context.Context, r *GetFuturesOrdersRequest) ([]FuturesOrderDetailResponse, error) {
	r.PageSize = futuresOrdersPageSize
	return collectPages(func(page uint64) ([]FuturesOrderDetailResponse, uint64, error) {
		r.CurrentPage = page
		resp, err := e.GetFuturesOrders(ctx, r)
		if err != nil || resp == nil {
			return nil, 0, err
		}
		return resp.Items, resp.TotalPage, nil
	})
}

// futuresOrders returns the futures orders of a status, active or done, that a request matches, those of one symbol or
// of every symbol when symbol is empty
func (e *Exchange) futuresOrders(ctx context.Context, req *order.MultiOrderRequest, status, symbol, side string) ([]order.Detail, error) {
	oType, err := OrderTypeToString(req.Type)
	if err != nil {
		return nil, err
	}
	orders, err := e.allFuturesOrders(ctx, &GetFuturesOrdersRequest{
		Status:  status,
		Symbol:  symbol,
		Side:    side,
		Type:    oType,
		StartAt: req.StartTime,
		EndAt:   req.EndTime,
	})
	if err != nil {
		return nil, err
	}
	details := make([]order.Detail, len(orders))
	for i := range orders {
		if details[i], err = e.futuresOrderDetail(&orders[i]); err != nil {
			return nil, err
		}
	}
	return details, nil
}

// activeFuturesOrders returns the active futures orders a request matches, with the untriggered stop orders KuCoin lists
// apart from them. A stop order that triggers between the two lists moves from one to the other, so the stop orders are
// listed first and one the active orders hold already is left out
func (e *Exchange) activeFuturesOrders(ctx context.Context, req *order.MultiOrderRequest, symbol, side string) ([]order.Detail, error) {
	var stopType string
	listed := req
	switch req.Type {
	case order.StopLimit, order.TakeProfit:
		stopType = order.Limit.Lower()
	case order.StopMarket, order.TakeProfitMarket:
		stopType = order.Market.Lower()
	case order.AnyType, order.UnknownType:
	default:
		// Limit and market orders are no stop orders, and futuresOrders refuses the types KuCoin does not list
		return e.futuresOrders(ctx, req, "active", symbol, side)
	}
	r := &GetUntriggeredFuturesStopOrdersRequest{Symbol: symbol, Side: side, Type: stopType, StartAt: req.StartTime, EndAt: req.EndTime, PageSize: futuresOrdersPageSize}
	stops, err := collectPages(func(page uint64) ([]FuturesOrderDetailResponse, uint64, error) {
		r.CurrentPage = page
		resp, err := e.GetUntriggeredFuturesStopOrders(ctx, r)
		if err != nil || resp == nil {
			return nil, 0, err
		}
		return resp.Items, resp.TotalPage, nil
	})
	if err != nil {
		return nil, err
	}
	// KuCoin lists a triggered take profit order among the stop orders of its type, which takeProfitType then names
	switch req.Type {
	case order.TakeProfit:
		r := *req
		r.Type = order.StopLimit
		listed = &r
	case order.TakeProfitMarket:
		r := *req
		r.Type = order.StopMarket
		listed = &r
	}
	orders, err := e.futuresOrders(ctx, listed, "active", symbol, side)
	if err != nil {
		return nil, err
	}
	active := make(map[string]struct{}, len(orders))
	for i := range orders {
		active[orders[i].OrderID] = struct{}{}
	}
	for i := range stops {
		if _, ok := active[stops[i].ID]; ok {
			continue
		}
		d, err := e.futuresOrderDetail(&stops[i])
		if err != nil {
			return nil, err
		}
		orders = append(orders, d)
	}
	return orders, nil
}

// GetActiveOrders returns open orders over REST. KuCoin lists open spot and margin orders by symbol, so they are listed
// for the request's pair, or for each symbol with open orders, among the request's pairs when it has several. Spot and
// margin stop and OCO orders come from KuCoin's lists of untriggered stop orders, and futures orders are listed with the
// untriggered futures stop orders unless the request is for limit or market orders. Margin orders are those of the
// request's margin type, cross margin unless it is isolated
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pFmt, err := e.requestFormat(req.AssetType)
	if err != nil {
		return nil, err
	}
	side, err := e.OrderSideString(req.Side)
	if err != nil {
		return nil, err
	}
	var symbol string
	if len(req.Pairs) == 1 {
		symbol = pFmt.Format(req.Pairs[0])
	}
	var orders []order.Detail
	switch {
	case req.AssetType == asset.Futures:
		orders, err = e.activeFuturesOrders(ctx, req, symbol, side)
	case req.Type == order.OCO || isStopOrderType(req.Type):
		orders, err = e.untriggeredStopOrders(ctx, req, symbol, side)
	default:
		orders, err = e.openHFOrders(ctx, req, pFmt)
	}
	if err != nil {
		return nil, err
	}
	return e.filterOrders(req, orders), nil
}

// openHFOrders returns the open spot or margin limit and market orders of the request's pair, or of each symbol with
// open orders, among the request's pairs when it has several. KuCoin lists open orders by symbol, so this spares a
// request for each pair without open orders
func (e *Exchange) openHFOrders(ctx context.Context, req *order.MultiOrderRequest, pFmt currency.PairFormat) ([]order.Detail, error) {
	tradeType := e.AccountToTradeTypeString(req.AssetType, req.MarginType)
	var symbols []string
	if len(req.Pairs) == 1 {
		symbols = []string{pFmt.Format(req.Pairs[0])}
	} else {
		open, err := e.openOrderSymbols(ctx, req.AssetType, tradeType)
		if err != nil {
			return nil, err
		}
		requested := make(map[string]bool, len(req.Pairs))
		for _, p := range req.Pairs {
			requested[pFmt.Format(p)] = true
		}
		for _, symbol := range open {
			if len(requested) == 0 || requested[symbol] {
				symbols = append(symbols, symbol)
			}
		}
	}
	var orders []order.Detail
	for _, symbol := range symbols {
		if req.AssetType == asset.Margin {
			open, err := e.GetActiveMarginHFOrders(ctx, symbol, tradeType)
			if err != nil {
				return nil, err
			}
			for i := range open {
				d, err := e.marginHFOrderDetail(&open[i])
				if err != nil {
					return nil, err
				}
				orders = append(orders, d)
			}
			continue
		}
		open, err := e.GetOpenHFOrders(ctx, symbol)
		if err != nil {
			return nil, err
		}
		for i := range open {
			d, err := e.hfOrderDetail(&open[i])
			if err != nil {
				return nil, err
			}
			orders = append(orders, d)
		}
	}
	return orders, nil
}

// untriggeredStopOrders returns the request's untriggered spot or margin stop orders, or its OCO orders for an OCO
// request, of one symbol or of every symbol when symbol is empty. Triggered orders are listed with the open orders, so
// they are left out
func (e *Exchange) untriggeredStopOrders(ctx context.Context, req *order.MultiOrderRequest, symbol, side string) ([]order.Detail, error) {
	oco := req.Type == order.OCO
	stop := "stop"
	if oco {
		stop = "oco"
	}
	tradeType := e.AccountToTradeTypeString(req.AssetType, req.MarginType)
	var stops []StopOrderResponse
	var err error
	if req.AssetType == asset.Margin {
		stops, err = collectPages(func(page uint64) ([]StopOrderResponse, uint64, error) {
			resp, err := e.GetMarginStopOrders(ctx, &GetMarginStopOrdersRequest{
				Symbol:      symbol,
				Side:        side,
				TradeType:   tradeType,
				StartAt:     req.StartTime,
				EndAt:       req.EndTime,
				CurrentPage: page,
				PageSize:    stopOrdersPageSize,
				Stop:        stop,
			})
			if err != nil || resp == nil {
				return nil, 0, err
			}
			items := make([]StopOrderResponse, len(resp.Items))
			for i := range resp.Items {
				// Margin and spot stop orders hold the same fields
				items[i] = StopOrderResponse(resp.Items[i].MarginStopOrderResponse)
			}
			return items, resp.TotalPage, nil
		})
	} else {
		stops, err = collectPages(func(page uint64) ([]StopOrderResponse, uint64, error) {
			resp, err := e.GetStopOrderList(ctx, &StopOrderListRequest{
				Symbol:      symbol,
				Side:        side,
				TradeType:   tradeType,
				StartAt:     req.StartTime,
				EndAt:       req.EndTime,
				CurrentPage: page,
				PageSize:    stopOrdersPageSize,
				Stop:        stop,
			})
			if err != nil || resp == nil {
				return nil, 0, err
			}
			items := make([]StopOrderResponse, len(resp.Items))
			for i := range resp.Items {
				items[i] = resp.Items[i].StopOrderResponse
			}
			return items, resp.TotalPage, nil
		})
	}
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(stops))
	for i := range stops {
		if !strings.EqualFold(stops[i].Status, "NEW") {
			continue
		}
		d, err := e.stopOrderDetail(&stops[i], req.AssetType, oco)
		if err != nil {
			return nil, err
		}
		orders = append(orders, d)
	}
	return orders, nil
}

// filterOrders applies a request's filters to orders, matching a Long or Bid request to buy orders, a Short or Ask
// request to sell orders, and a Stop or ConditionalStop request to every stop order type
func (e *Exchange) filterOrders(req *order.MultiOrderRequest, orders []order.Detail) order.FilteredOrders {
	filter := *req
	switch {
	case filter.Side.IsLong():
		filter.Side = order.Buy
	case filter.Side.IsShort():
		filter.Side = order.Sell
	}
	if filter.Type == order.Stop || filter.Type == order.ConditionalStop {
		filter.Type = order.AnyType
	}
	return filter.Filter(e.Name, orders)
}

// GetOrderHistory returns closed orders over REST. KuCoin lists closed spot and margin limit and market orders by
// symbol, so they need the request's pairs, and keeps cancelled spot orders for two days and filled ones for seven.
// Futures and OCO orders are listed for the request's pair, or for every pair when it has several or none. KuCoin keeps
// no history of stop orders: triggered ones are listed with the other closed orders and untriggered ones are open.
// Margin orders are those of the request's margin type, cross margin unless it is isolated
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	pFmt, err := e.requestFormat(req.AssetType)
	if err != nil {
		return nil, err
	}
	side, err := e.OrderSideString(req.Side)
	if err != nil {
		return nil, err
	}
	var symbol string
	if len(req.Pairs) == 1 {
		symbol = pFmt.Format(req.Pairs[0])
	}
	var orders []order.Detail
	switch {
	case req.AssetType == asset.Futures:
		orders, err = e.futuresOrders(ctx, req, "done", symbol, side)
	case isStopOrderType(req.Type):
		return nil, fmt.Errorf("%w: KuCoin keeps no stop order history", order.ErrUnsupportedOrderType)
	case req.Type == order.OCO:
		orders, err = e.ocoOrderHistory(ctx, req, symbol)
	default:
		orders, err = e.closedHFOrders(ctx, req, pFmt, side)
	}
	if err != nil {
		return nil, err
	}
	return e.filterOrders(req, orders), nil
}

// closedHFOrders returns the closed spot or margin limit and market orders of the request's pairs
func (e *Exchange) closedHFOrders(ctx context.Context, req *order.MultiOrderRequest, pFmt currency.PairFormat, side string) ([]order.Detail, error) {
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	oType, err := OrderTypeToString(req.Type)
	if err != nil {
		return nil, err
	}
	tradeType := e.AccountToTradeTypeString(req.AssetType, req.MarginType)
	var orders []order.Detail
	for i := range req.Pairs {
		symbol := pFmt.Format(req.Pairs[i])
		if req.AssetType == asset.Margin {
			closed, err := collectByLastID(func(lastID uint64) ([]MarginHFOrderDetailResponse, uint64, error) {
				resp, err := e.GetClosedMarginHFOrders(ctx, &GetClosedMarginHFOrdersRequest{
					Symbol:    symbol,
					TradeType: tradeType,
					Side:      side,
					Type:      oType,
					LastID:    lastID,
					Limit:     closedOrdersLimit,
					StartAt:   req.StartTime,
					EndAt:     req.EndTime,
				})
				if err != nil || resp == nil {
					return nil, 0, err
				}
				return resp.Items, resp.LastID, nil
			})
			if err != nil {
				return nil, err
			}
			for j := range closed {
				d, err := e.marginHFOrderDetail(&closed[j].MarginHFOrderResponse)
				if err != nil {
					return nil, err
				}
				orders = append(orders, d)
			}
			continue
		}
		closed, err := collectByLastID(func(lastID uint64) ([]HFOrderDetailResponse, uint64, error) {
			resp, err := e.GetClosedHFOrders(ctx, &ClosedHFOrdersRequest{
				Symbol:  symbol,
				Side:    side,
				Type:    oType,
				LastID:  lastID,
				Limit:   closedOrdersLimit,
				StartAt: req.StartTime,
				EndAt:   req.EndTime,
			})
			if err != nil || resp == nil {
				return nil, 0, err
			}
			return resp.Items, resp.LastID, nil
		})
		if err != nil {
			return nil, err
		}
		for j := range closed {
			d, err := e.hfOrderDetail(&closed[j].HFOrderResponse)
			if err != nil {
				return nil, err
			}
			orders = append(orders, d)
		}
	}
	return orders, nil
}

// ocoOrderHistory returns the request's spot or margin OCO orders that are no longer open, of one symbol or of every
// symbol when symbol is empty. KuCoin documents no replacement for its abandoned spot OCO order list, which lists the
// OCO orders that have finished, so spot OCO orders still come from it
func (e *Exchange) ocoOrderHistory(ctx context.Context, req *order.MultiOrderRequest, symbol string) ([]order.Detail, error) {
	var ocos []OCOOrder
	var err error
	if req.AssetType == asset.Margin {
		tradeType := e.AccountToTradeTypeString(req.AssetType, req.MarginType)
		ocos, err = collectPages(func(page uint64) ([]OCOOrder, uint64, error) {
			resp, err := e.GetMarginOCOOrders(ctx, &GetMarginOCOOrdersRequest{
				PageSize:    ocoOrdersPageSize,
				CurrentPage: page,
				Symbol:      symbol,
				StartAt:     req.StartTime,
				EndAt:       req.EndTime,
				TradeType:   tradeType,
			})
			if err != nil || resp == nil {
				return nil, 0, err
			}
			items := make([]OCOOrder, len(resp.Items))
			for i := range resp.Items {
				items[i] = OCOOrder{
					OrderID:       resp.Items[i].OrderID,
					Symbol:        resp.Items[i].Symbol,
					ClientOrderID: resp.Items[i].ClientOrderID,
					OrderTime:     resp.Items[i].OrderTime,
					Status:        resp.Items[i].Status,
				}
			}
			return items, resp.TotalPage, nil
		})
	} else {
		ocos, err = collectPages(func(page uint64) ([]OCOOrder, uint64, error) {
			resp, err := e.GetOCOOrderList(ctx, &OCOOrderListRequest{
				Symbol:      symbol,
				StartAt:     req.StartTime,
				EndAt:       req.EndTime,
				PageSize:    ocoOrdersPageSize,
				CurrentPage: page,
			})
			if err != nil || resp == nil {
				return nil, 0, err
			}
			return resp.Items, resp.TotalPage, nil
		})
	}
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(ocos))
	for i := range ocos {
		if strings.EqualFold(ocos[i].Status, "NEW") {
			continue
		}
		d, err := e.ocoOrderDetail(&ocos[i], req.AssetType)
		if err != nil {
			return nil, err
		}
		orders = append(orders, d)
	}
	return orders, nil
}

// errFeeTypeNotSupported is returned for a fee type GetFeeByType cannot estimate
var errFeeTypeNotSupported = errors.New("fee type not supported")

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	if feeBuilder.Pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		// Get Actual Fee takes spot symbols, which are upper case and dash delimited
		symbol := feeBuilder.Pair.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}).String()
		fees, err := e.GetTradingPairActualFees(ctx, []string{symbol})
		if err != nil {
			return 0, err
		}
		if len(fees) == 0 {
			return 0, fmt.Errorf("%w: no fee rates for %s", common.ErrNoResponse, symbol)
		}
		if feeBuilder.IsMaker {
			return feeBuilder.Amount * fees[0].MakerFeeRate.Float64(), nil
		}
		return feeBuilder.Amount * fees[0].TakerFeeRate.Float64(), nil
	case exchange.CryptocurrencyWithdrawalFee:
		quota, err := e.GetWithdrawalQuotas(ctx, feeBuilder.Pair.Base, "")
		if err != nil {
			return 0, err
		}
		if quota == nil {
			return 0, common.ErrNoResponse
		}
		return quota.WithdrawMinFee.Float64(), nil
	case exchange.OfflineTradeFee:
		return feeBuilder.Amount * 0.001, nil
	case exchange.CryptocurrencyDepositFee:
		return 0, nil
	default:
		// KuCoin's API offers no bank deposits or withdrawals, so their fees cannot be estimated
		return 0, fmt.Errorf("%w: %v", errFeeTypeNotSupported, feeBuilder.FeeType)
	}
}

// futuresKlineLimit is the most candles one futures Get Klines request returns; the docs say 500, but the server stops
// at 200
const futuresKlineLimit = 200

// futuresCandles returns a futures contract's candles between start and end, requesting them limit candles at a time
func (e *Exchange) futuresCandles(ctx context.Context, symbol string, interval kline.Interval, start, end time.Time, limit uint64) ([]kline.Candle, error) {
	ranges, err := kline.CalculateCandleDateRanges(start, end, interval, limit)
	if err != nil {
		return nil, err
	}
	var candles []kline.Candle
	for i := range ranges.Ranges {
		resp, err := e.GetFuturesKline(ctx, &FuturesKlineRequest{Symbol: symbol, Granularity: interval, From: ranges.Ranges[i].Start.Time, To: ranges.Ranges[i].End.Time})
		if err != nil {
			return nil, err
		}
		for j := range resp {
			candles = append(candles, kline.Candle{
				Time:   resp[j].StartTime.Time(),
				Open:   resp[j].Open,
				High:   resp[j].High,
				Low:    resp[j].Low,
				Close:  resp[j].Close,
				Volume: resp[j].Volume,
			})
		}
	}
	return candles, nil
}

// getCandles returns a symbol's candles between start and end at an interval the asset's Get Klines endpoint serves
func (e *Exchange) getCandles(ctx context.Context, symbol string, a asset.Item, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	switch a {
	case asset.Futures:
		if interval == kline.SixHour {
			return nil, fmt.Errorf("%w: futures candles have no %s granularity", kline.ErrUnsupportedInterval, interval)
		}
		return e.futuresCandles(ctx, symbol, interval, start, end, futuresKlineLimit)
	case asset.Spot, asset.Margin:
		intervalString, err := IntervalToString(interval)
		if err != nil {
			return nil, err
		}
		resp, err := e.GetKlines(ctx, &KlinesRequest{Symbol: symbol, Type: intervalString, StartAt: start, EndAt: end})
		if err != nil {
			return nil, err
		}
		candles := make([]kline.Candle, len(resp))
		for i := range resp {
			candles[i] = kline.Candle{
				Time:   resp[i].StartTime.Time(),
				Open:   resp[i].Open.Float64(),
				High:   resp[i].High.Float64(),
				Low:    resp[i].Low.Float64(),
				Close:  resp[i].Close.Float64(),
				Volume: resp[i].Volume.Float64(),
			}
		}
		return candles, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	candles, err := e.getCandles(ctx, req.RequestFormatted.String(), a, req.ExchangeInterval, req.Start, req.End)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var timeSeries []kline.Candle
	for i := range req.RangeHolder.Ranges {
		candles, err := e.getCandles(ctx, req.RequestFormatted.String(), a, req.ExchangeInterval, req.RangeHolder.Ranges[i].Start.Time, req.RangeHolder.Ranges[i].End.Time)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	switch a {
	case asset.Spot, asset.Margin:
		return e.GetCurrentServerTime(ctx)
	case asset.Futures:
		return e.GetFuturesServerTime(ctx)
	default:
		return time.Time{}, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// GetAvailableTransferChains returns the IDs of the chains a cryptocurrency can be deposited and withdrawn on, which
// GetDepositAddress and WithdrawCryptocurrencyFunds take
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	currencyDetail, err := e.GetCurrencyDetailV3(ctx, cryptocurrency, "")
	if err != nil {
		return nil, err
	}
	if currencyDetail == nil {
		return nil, common.ErrNoResponse
	}
	chains := make([]string, len(currencyDetail.Chains))
	for i := range currencyDetail.Chains {
		chains[i] = currencyDetail.Chains[i].ChainID
	}
	return chains, nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// perpetualContractType is the type of KuCoin's perpetual futures contracts; dated futures are FFICSX
const perpetualContractType = "FFWCSX"

// contractFundingRate returns a contract's current funding rate, timed at the start of its funding period. KuCoin
// leaves the current funding interval null on some contracts, which then follow their configured interval
func contractFundingRate(c *ContractResponse) fundingrate.Rate {
	interval := c.CurrentFundingRateGranularity
	if interval == 0 {
		interval = c.FundingRateGranularity
	}
	rate := fundingrate.Rate{Rate: decimal.MustFromFloat(c.FundingFeeRate)}
	if next := c.NextFundingRateDateTime.Time(); !next.IsZero() {
		rate.Time = next.Add(-time.Duration(interval) * time.Millisecond) //nolint:gosec // Funding intervals are hours in milliseconds, far inside int64's range
	}
	return rate
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	contracts, err := e.GetFuturesOpenContracts(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(contracts))
	for i := range contracts {
		name, err := futuresContractPair(&contracts[i])
		if err != nil {
			return nil, err
		}
		contractType := futures.Quarterly
		if contracts[i].Type == perpetualContractType {
			contractType = futures.Perpetual
		}
		settlementType := futures.Linear
		if contracts[i].IsInverse {
			settlementType = futures.Inverse
		}
		resp[i] = futures.Contract{
			Exchange:           e.Name,
			Name:               name,
			Underlying:         currency.NewPair(contracts[i].BaseCurrency, contracts[i].QuoteCurrency),
			Asset:              item,
			StartDate:          contracts[i].FirstOpenDate.Time(),
			EndDate:            contracts[i].ExpireDate.Time(),
			IsActive:           !strings.EqualFold(contracts[i].Status, "closed"),
			Status:             contracts[i].Status,
			Type:               contractType,
			SettlementType:     settlementType,
			SettlementCurrency: contracts[i].SettleCurrency,
			MarginCurrency:     contracts[i].SettleCurrency,
			Multiplier:         contracts[i].Multiplier,
			MaxLeverage:        float64(contracts[i].MaxLeverage),
			LatestRate:         contractFundingRate(&contracts[i]),
			FundingRateFloor:   decimal.MustFromFloat(contracts[i].FundingRateFloor),
			FundingRateCeiling: decimal.MustFromFloat(contracts[i].FundingRateCap),
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
	// KuCoin documents a predicted funding rate, but never sends one
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}
	if r.Pair.IsEmpty() {
		contracts, err := e.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		timeChecked := time.Now()
		resp := make([]fundingrate.LatestRateResponse, 0, len(contracts))
		for i := range contracts {
			if contracts[i].Type != perpetualContractType {
				continue
			}
			pair, err := futuresContractPair(&contracts[i])
			if err != nil {
				return nil, err
			}
			resp = append(resp, fundingrate.LatestRateResponse{
				Exchange:       e.Name,
				Asset:          r.Asset,
				Pair:           pair,
				LatestRate:     contractFundingRate(&contracts[i]),
				TimeOfNextRate: contracts[i].NextFundingRateDateTime.Time(),
				TimeChecked:    timeChecked,
			})
		}
		return resp, nil
	}
	if !isPerpetualPair(r.Pair) {
		return nil, fmt.Errorf("%w %s %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	fundingRate, err := e.GetFuturesCurrentFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	if fundingRate == nil {
		return nil, common.ErrNoResponse
	}
	return []fundingrate.LatestRateResponse{{
		Exchange: e.Name,
		Asset:    r.Asset,
		Pair:     r.Pair,
		LatestRate: fundingrate.Rate{
			Time: fundingRate.TimePoint.Time(),
			Rate: decimal.MustFromFloat(fundingRate.Value),
		},
		TimeOfNextRate: fundingRate.FundingTime.Time(),
		TimeChecked:    time.Now(),
	}}, nil
}

// isPerpetualPair reports whether a futures pair is a perpetual contract's, whose quote is USDTM, USDCM or USDM; a
// dated contract's quote is its delivery month
func isPerpetualPair(cp currency.Pair) bool {
	return cp.Quote.Equal(currency.USDTM) || cp.Quote.Equal(currency.USDCM) || cp.Quote.Equal(currency.USDM)
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	return a == asset.Futures && isPerpetualPair(cp), nil
}

// GetHistoricalFundingRates returns a perpetual contract's funding rates over a window, newest first. KuCoin returns
// the newest hundred rates of a window a request, so a longer window is read back from its end a hundred at a time
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
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
	var err error
	r.Pair, err = e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}

	// KuCoin returns the newest rates of a window, at most publicFundingRatesLimit, so each later request asks for the
	// rates before the oldest one the last returned. Both edges of a window are inclusive
	var records []FuturesPublicFundingHistoryResponse
	for end := r.EndDate; ; {
		page, err := e.GetPublicFundingRate(ctx, r.Pair.String(), r.StartDate, end)
		if err != nil {
			return nil, err
		}
		// A rate after the window's end is none of its rates, so it neither counts nor moves the window back
		var oldest time.Time
		for i := range page {
			t := page[i].Timepoint.Time()
			if t.After(end) {
				continue
			}
			records = append(records, page[i])
			if oldest.IsZero() || t.Before(oldest) {
				oldest = t
			}
		}
		// A short page holds the window's last rates. The next window ends before the oldest rate this one held, which
		// leaves none once that is at the window's start; a page of no rate in the window leaves oldest unset, which is
		// before any start, as such a page cannot move the window back
		next := oldest.Add(-time.Millisecond)
		if len(page) < publicFundingRatesLimit || !next.After(r.StartDate) {
			break
		}
		end = next
	}

	if len(records) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	fundingRates := make([]fundingrate.Rate, 0, len(records))
	for i := range records {
		if (!r.EndDate.IsZero() && r.EndDate.Before(records[i].Timepoint.Time())) ||
			(!r.StartDate.IsZero() && r.StartDate.After(records[i].Timepoint.Time())) {
			continue
		}

		fundingRates = append(fundingRates, fundingrate.Rate{
			Rate: decimal.MustFromFloat(records[i].FundingRate),
			Time: records[i].Timepoint.Time(),
		})
	}

	if len(fundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}

	return &fundingrate.HistoricalRates{
		Exchange:        e.Name,
		Asset:           r.Asset,
		Pair:            r.Pair,
		FundingRates:    fundingRates,
		StartDate:       fundingRates[len(fundingRates)-1].Time,
		EndDate:         fundingRates[0].Time,
		LatestRate:      fundingRates[0],
		PaymentCurrency: r.PaymentCurrency,
	}, nil
}

// GetLeverage returns a futures contract's cross margin leverage. KuCoin sets isolated margin leverage per order, so it
// has no leverage to return for isolated margin, or for other assets
func (e *Exchange) GetLeverage(ctx context.Context, a asset.Item, pair currency.Pair, marginType margin.Type, _ order.Side) (float64, error) {
	if a != asset.Futures {
		return 0, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	if marginType != margin.Multi {
		return 0, fmt.Errorf("%w: KuCoin returns futures cross margin leverage only, not %q", margin.ErrMarginTypeUnsupported, marginType)
	}
	if pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	pFmt, err := e.requestFormat(a)
	if err != nil {
		return 0, err
	}
	resp, err := e.GetFuturesCrossMarginLeverage(ctx, pFmt.Format(pair))
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, common.ErrNoResponse
	}
	return resp.Leverage.Float64(), nil
}

// SetLeverage sets a futures contract's cross margin leverage, or the leverage of the cross margin account or of an
// isolated margin pair. KuCoin sets isolated futures leverage per order
func (e *Exchange) SetLeverage(ctx context.Context, a asset.Item, pair currency.Pair, marginType margin.Type, amount float64, _ order.Side) error {
	pFmt, err := e.requestFormat(a)
	if err != nil {
		return err
	}
	switch a {
	case asset.Futures:
		if marginType != margin.Multi {
			return fmt.Errorf("%w: KuCoin sets futures cross margin leverage only, not %q", margin.ErrMarginTypeUnsupported, marginType)
		}
		if pair.IsEmpty() {
			return currency.ErrCurrencyPairEmpty
		}
		modified, err := e.ModifyFuturesCrossMarginLeverage(ctx, pFmt.Format(pair), amount)
		if err != nil {
			return err
		}
		if !modified {
			return errLeverageNotModified
		}
		return nil
	case asset.Margin:
		r := &ModifyLeverageMultiplierRequest{IsIsolated: marginType == margin.Isolated, Leverage: amount}
		if r.IsIsolated {
			if pair.IsEmpty() {
				return currency.ErrCurrencyPairEmpty
			}
			r.Symbol = pFmt.Format(pair)
		}
		return e.ModifyLeverageMultiplier(ctx, r)
	default:
		return fmt.Errorf("%w: KuCoin sets futures and margin leverage only, not %s leverage", asset.ErrNotSupported, a)
	}
}

// SetMarginType switches a futures contract between isolated and cross margin
func (e *Exchange) SetMarginType(ctx context.Context, a asset.Item, pair currency.Pair, marginType margin.Type) error {
	if a != asset.Futures {
		return fmt.Errorf("%w: KuCoin sets the margin type of futures only, not %s", asset.ErrNotSupported, a)
	}
	mode, err := futuresMarginMode(marginType)
	if err != nil {
		return err
	}
	if pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	pFmt, err := e.requestFormat(a)
	if err != nil {
		return err
	}
	_, err = e.SwitchFuturesMarginMode(ctx, pFmt.Format(pair), mode)
	return err
}

// SetCollateralMode sets the collateral type for your account
func (e *Exchange) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// GetCollateralMode returns the collateral type for your account
func (e *Exchange) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return collateral.UnknownMode, fmt.Errorf("%w only via website", common.ErrFunctionNotSupported)
}

// ChangePositionMargin moves an isolated margin futures position's margin from OriginalAllocatedMargin to
// NewAllocatedMargin, adding or withdrawing the difference. MarginSide is the position side: both, or in hedge mode
// buy or long for the long position and sell or short for the short one
func (e *Exchange) ChangePositionMargin(ctx context.Context, r *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotFuturesAsset, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if r.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, r.MarginType)
	}
	if r.NewAllocatedMargin <= 0 {
		return nil, margin.ErrNewAllocatedMarginRequired
	}
	if r.OriginalAllocatedMargin <= 0 {
		return nil, margin.ErrOriginalPositionMarginRequired
	}
	change := decimal.MustFromFloat(r.NewAllocatedMargin).Sub(decimal.MustFromFloat(r.OriginalAllocatedMargin))
	if change.IsZero() {
		return nil, errMarginUnchanged
	}
	pFmt, err := e.requestFormat(r.Asset)
	if err != nil {
		return nil, err
	}
	symbol := pFmt.Format(r.Pair)
	var positionSide string
	switch strings.ToLower(r.MarginSide) {
	case "": // KuCoin defaults to BOTH, the one-way mode position
	case "both":
		positionSide = "BOTH"
	case "buy", "long":
		positionSide = "LONG"
	case "sell", "short":
		positionSide = "SHORT"
	default:
		return nil, fmt.Errorf("%w: margin side %q", order.ErrSideIsInvalid, r.MarginSide)
	}
	resp := &margin.PositionChangeResponse{Exchange: e.Name, Pair: r.Pair, Asset: r.Asset, MarginType: r.MarginType}
	if change.IsPositive() {
		// The business number makes KuCoin add the margin once, however often the request is sent
		position, err := e.AddMargin(ctx, &AddMarginRequest{Symbol: symbol, Margin: change.InexactFloat64(), BusinessNumber: rand.Text(), PositionSide: positionSide})
		if err != nil {
			return nil, err
		}
		if position == nil {
			return nil, common.ErrNoResponse
		}
		resp.AllocatedMargin = position.PositionMargin
		return resp, nil
	}
	withdrawn, err := e.RemoveMarginManually(ctx, &RemoveMarginManuallyRequest{Symbol: symbol, WithdrawAmount: change.Neg().InexactFloat64(), PositionSide: positionSide})
	if err != nil {
		return nil, err
	}
	resp.AllocatedMargin = decimal.MustFromFloat(r.OriginalAllocatedMargin).Sub(decimal.MustFromFloat(withdrawn)).InexactFloat64()
	return resp, nil
}

// GetFuturesPositionSummary returns a summary of a futures position and its settlement currency's account. In hedge
// mode, Direction chooses the long or short position. ContractSize is in what the position's lots are worth: the base
// currency for a linear contract and USD for an inverse one
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	pFmt, err := e.requestFormat(r.Asset)
	if err != nil {
		return nil, err
	}
	contracts, err := e.GetFuturesContractDetails(ctx, r.Asset)
	if err != nil {
		return nil, err
	}
	var multiplier float64
	var settlementType futures.ContractSettlementType
	for i := range contracts {
		if contracts[i].Name.Equal(r.Pair) {
			multiplier = contracts[i].Multiplier
			settlementType = contracts[i].SettlementType
			break
		}
	}
	positions, err := e.GetFuturesPositionDetails(ctx, pFmt.Format(r.Pair))
	if err != nil {
		return nil, err
	}
	pos, err := futuresPosition(positions, r.Direction)
	if err != nil {
		return nil, err
	}
	account, err := e.GetFuturesAccountOverview(ctx, pos.SettleCurrency)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, common.ErrNoResponse
	}
	marginType := margin.Isolated
	if pos.CrossMode {
		marginType = margin.Multi
	}
	size := decimal.NewFromInt(pos.CurrentQuantity)
	return &futures.PositionSummary{
		Pair:                         r.Pair,
		Asset:                        r.Asset,
		MarginType:                   marginType,
		CollateralMode:               collateral.MultiMode,
		Currency:                     pos.SettleCurrency,
		StartDate:                    pos.OpeningTimestamp.Time(),
		AvailableEquity:              decimal.MustFromFloat(account.AccountEquity),
		MarginBalance:                decimal.MustFromFloat(account.MarginBalance),
		NotionalSize:                 pos.MarkValue.Decimal(),
		Leverage:                     pos.Leverage.Decimal(),
		MaintenanceMarginRequirement: pos.PositionMaintenanceMargin.Decimal(),
		InitialMarginRequirement:     pos.PositionInitialMargin.Decimal(),
		EstimatedLiquidationPrice:    pos.LiquidationPrice.Decimal(),
		CollateralUsed:               pos.PositionMargin.Decimal(),
		MarkPrice:                    pos.MarkPrice.Decimal(),
		CurrentSize:                  size,
		ContractSize:                 size.Mul(lotValue(multiplier)),
		ContractMultiplier:           lotValue(multiplier),
		ContractSettlementType:       settlementType,
		AverageOpenPrice:             pos.AverageEntryPrice.Decimal(),
		UnrealisedPNL:                pos.UnrealisedPNL.Decimal(),
		RealisedPNL:                  pos.RealisedPNL.Decimal(),
		MaintenanceMarginFraction:    pos.MaintenanceMarginRequirement.Decimal(),
		FreeCollateral:               decimal.MustFromFloat(account.AvailableBalance),
		TotalCollateral:              decimal.MustFromFloat(account.AccountEquity),
		FrozenBalance:                decimal.MustFromFloat(account.FrozenFunds),
	}, nil
}

// futuresPosition returns a contract's position of a direction: the position in one-way mode, and in hedge mode the
// long or short one, which an unset direction leaves ambiguous
func futuresPosition(positions []FuturesPositionDetailResponse, direction order.Side) (*FuturesPositionDetailResponse, error) {
	var found *FuturesPositionDetailResponse
	for i := range positions {
		switch {
		case direction.IsLong() && positions[i].PositionSide == "SHORT", direction.IsShort() && positions[i].PositionSide == "LONG":
			continue
		case found != nil:
			return nil, errPositionDirectionRequired
		}
		found = &positions[i]
	}
	if found == nil {
		return nil, futures.ErrPositionNotFound
	}
	return found, nil
}

// lotValue returns what one lot of a futures contract is worth: its multiplier of the base currency for a linear
// contract, and 1 USD for an inverse one, whose multiplier KuCoin reports as -1
func lotValue(multiplier float64) decimal.Decimal {
	return decimal.MustFromFloat(multiplier).Abs()
}

// GetFuturesPositionOrders returns each pair's closed futures orders over REST, in a window of at most seven days, which
// RespectOrderHistoryLimits shortens from its start rather than erroring. The window ends now when EndDate is unset.
// ContractAmount is in lots, and Amount, ExecutedAmount and RemainingAmount are in what the lots are worth: the base
// currency for a linear contract and USD for an inverse one. They are zero for a contract the list of active contracts
// no longer holds, such as a settled one, whose lot is unknown
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if err := common.NilGuard(r); err != nil {
		return nil, err
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if r.EndDate.IsZero() {
		// The order manager sets only the start, from which KuCoin would list a single day
		r.EndDate = time.Now()
	}
	if err := common.StartEndTimeCheck(r.StartDate, r.EndDate); err != nil {
		return nil, err
	}
	if window := e.Features.Supports.MaximumOrderHistory; r.EndDate.Sub(r.StartDate) > window {
		if !r.RespectOrderHistoryLimits {
			return nil, fmt.Errorf("%w: KuCoin lists closed futures orders over at most %s", futures.ErrOrderHistoryTooLarge, window)
		}
		r.StartDate = r.EndDate.Add(-window)
	}
	pFmt, err := e.requestFormat(r.Asset)
	if err != nil {
		return nil, err
	}
	contracts, err := e.GetFuturesContractDetails(ctx, r.Asset)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for x := range r.Pairs {
		resp[x].Pair = r.Pairs[x]
		resp[x].Asset = r.Asset
		var worth decimal.Decimal
		for i := range contracts {
			if contracts[i].Name.Equal(r.Pairs[x]) {
				worth = lotValue(contracts[i].Multiplier)
				resp[x].ContractSettlementType = contracts[i].SettlementType
				break
			}
		}
		orders, err := e.allFuturesOrders(ctx, &GetFuturesOrdersRequest{Symbol: pFmt.Format(r.Pairs[x]), StartAt: r.StartDate, EndAt: r.EndDate})
		if err != nil {
			return nil, err
		}
		resp[x].Orders = make([]order.Detail, len(orders))
		for i := range orders {
			if resp[x].Orders[i], err = e.futuresOrderDetail(&orders[i]); err != nil {
				return nil, err
			}
			// futuresOrderDetail reports lots, which a decimal product converts without float rounding
			o := &resp[x].Orders[i]
			o.Amount = decimal.MustFromFloat(o.ContractAmount).Mul(worth).InexactFloat64()
			o.ExecutedAmount = decimal.MustFromFloat(o.ExecutedAmount).Mul(worth).InexactFloat64()
			o.RemainingAmount = decimal.MustFromFloat(o.RemainingAmount).Mul(worth).InexactFloat64()
		}
	}
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}

	var l []limits.MinMaxLevel
	switch a {
	case asset.Spot, asset.Margin:
		symbols, err := e.GetSymbols(ctx, "")
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(symbols))
		for x := range symbols {
			if a == asset.Margin && !symbols[x].IsMarginEnabled {
				continue
			}
			pair, err := currency.NewPairFromString(symbols[x].Symbol)
			if err != nil {
				return err
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				AmountStepIncrementSize: symbols[x].BaseIncrement.Float64(),
				QuoteStepIncrementSize:  symbols[x].QuoteIncrement.Float64(),
				PriceStepIncrementSize:  symbols[x].PriceIncrement.Float64(),
				MinimumBaseAmount:       symbols[x].BaseMinSize.Float64(),
				MaximumBaseAmount:       symbols[x].BaseMaxSize.Float64(),
				MinimumQuoteAmount:      symbols[x].QuoteMinSize.Float64(),
				MaximumQuoteAmount:      symbols[x].QuoteMaxSize.Float64(),
				// minFunds is the least an order may be worth, its amount times its price for a limit order
				MinNotional: symbols[x].MinFunds.Float64(),
				Listed:      symbols[x].TradingStartTime.Time(),
			})
		}
	case asset.Futures:
		contracts, err := e.GetFuturesOpenContracts(ctx)
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(contracts))
		for i := range contracts {
			pair, err := futuresContractPair(&contracts[i])
			if err != nil {
				return err
			}
			priceDivisor := 1.0
			if strings.HasPrefix(contracts[i].Symbol, "10") { // handle 1000SHIBUSDTM, 1000PEPEUSDTM etc; exclude 1INCHUSDTM
				for _, r := range contracts[i].Symbol[1:] {
					if r != '0' {
						break
					}
					priceDivisor *= 10
				}
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				MaxPrice:                contracts[i].MaxPrice,
				PriceStepIncrementSize:  contracts[i].TickSize,
				PriceDivisor:            priceDivisor,
				MultiplierDecimal:       contracts[i].Multiplier,
				MinimumBaseAmount:       float64(contracts[i].LotSize),
				MaximumBaseAmount:       float64(contracts[i].MaxOrderQuantity),
				AmountStepIncrementSize: float64(contracts[i].LotSize),
				MarketMaxQty:            float64(contracts[i].MarketMaxOrderQuantity),
				Listed:                  contracts[i].FirstOpenDate.Time(),
				Delisted:                contracts[i].ExpireDate.Time(),
				Expiry:                  contracts[i].SettleDate.Time(),
			})
		}
	}

	return limits.Load(l)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	contracts, err := e.GetFuturesOpenContracts(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(contracts))
	for i := range contracts {
		pair, err := futuresContractPair(&contracts[i])
		if err != nil {
			return nil, err
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
			Key:          key.NewExchangeAssetPair(e.Name, asset.Futures, pair),
			OpenInterest: contracts[i].OpenInterest.Float64(),
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
	cp.Delimiter = currency.DashDelimiter
	switch a {
	case asset.Spot:
		return tradeBaseURL + tradeSpot + cp.Upper().String(), nil
	case asset.Margin:
		return tradeBaseURL + tradeSpot + tradeMargin + cp.Upper().String(), nil
	case asset.Futures:
		cp.Delimiter = ""
		return tradeBaseURL + tradeFutures + tradeSpot + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// StringToTimeInForce returns an order.TimeInForce instance from string
func StringToTimeInForce(tif string, postOnly bool) order.TimeInForce {
	tif = strings.ToUpper(tif)
	var out order.TimeInForce
	switch tif {
	case "GTT":
		out = order.GoodTillTime
	case "IOC":
		out = order.ImmediateOrCancel
	case "FOK":
		out = order.FillOrKill
	default:
		out = order.GoodTillCancel
	}
	if postOnly {
		out |= order.PostOnly
	}
	return out
}
