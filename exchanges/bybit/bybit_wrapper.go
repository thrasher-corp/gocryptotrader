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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
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
func (by *Bybit) SetDefaults() {
	by.Name = "Bybit"
	by.Enabled = true
	by.Verbose = true
	by.API.CredentialsValidator.RequiresKey = true
	by.API.CredentialsValidator.RequiresSecret = true

	for _, n := range assetPairFmts {
		ps := currency.PairStore{AssetEnabled: true, RequestFormat: n.reqFmt, ConfigFormat: n.cfgFmt}
		if err := by.SetAssetPairStore(n.asset, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", by.Name, n.asset, err)
		}
	}

	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.Options} {
		if err := by.DisableAssetWebsocketSupport(a); err != nil {
			log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", by.Name, a, err)
		}
	}

	by.Features = exchange.Features{
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

	by.API.Endpoints = by.NewEndpoints()
	err := by.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         bybitAPIURL,
		exchange.RestCoinMargined: bybitAPIURL,
		exchange.RestUSDTMargined: bybitAPIURL,
		exchange.RestFutures:      bybitAPIURL,
		exchange.RestUSDCMargined: bybitAPIURL,
		exchange.WebsocketSpot:    spotPublic,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	if by.Requester, err = request.New(by.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()),
	); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	by.Websocket = websocket.NewManager()
	by.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	by.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	by.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (by *Bybit) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		by.SetEnabled(false)
		return nil
	}

	err = by.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := by.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = by.Websocket.Setup(
		&websocket.ManagerSetup{
			ExchangeConfig:        exch,
			DefaultURL:            spotPublic,
			RunningURL:            wsRunningEndpoint,
			RunningURLAuth:        websocketPrivate,
			Connector:             by.WsConnect,
			Subscriber:            by.Subscribe,
			Unsubscriber:          by.Unsubscribe,
			GenerateSubscriptions: by.generateSubscriptions,
			Features:              &by.Features.Supports.WebsocketCapabilities,
			OrderbookBufferConfig: buffer.Config{
				SortBuffer:            true,
				SortBufferByUpdateIDs: true,
			},
			TradeFeed: by.Features.Enabled.TradeFeed,
		})
	if err != nil {
		return err
	}
	err = by.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  by.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     bybitWebsocketTimer,
	})
	if err != nil {
		return err
	}

	return by.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  websocketPrivate,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (by *Bybit) AuthenticateWebsocket(ctx context.Context) error {
	return by.WsAuth(ctx)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (by *Bybit) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !by.SupportsAsset(a) {
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	var pair currency.Pair
	var category string
	format, err := by.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	var (
		pairs    currency.Pairs
		allPairs []InstrumentInfo
		response *InstrumentsInfo
	)
	var nextPageCursor string
	switch a {
	case asset.Spot, asset.CoinMarginedFutures, asset.USDCMarginedFutures, asset.USDTMarginedFutures:
		category = getCategoryName(a)
		for {
			response, err = by.GetInstrumentInfo(ctx, category, "", "Trading", "", nextPageCursor, 1000)
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
				response, err = by.GetInstrumentInfo(ctx, category, "", "Trading", supportedOptionsTypes[x], nextPageCursor, 1000)
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
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
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
		return "inverse"
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		return "linear"
	case asset.Spot:
		return a.String()
	case asset.Options:
		return "option"
	default:
		return ""
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (by *Bybit) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := by.GetAssetTypes(true)
	for i := range assetTypes {
		pairs, err := by.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return err
		}
		err = by.UpdatePairs(pairs, assetTypes[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return by.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (by *Bybit) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	enabled, err := by.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	format, err := by.GetPairFormat(assetType, false)
	if err != nil {
		return err
	}
	var ticks *TickerData
	switch assetType {
	case asset.Spot, asset.USDCMarginedFutures,
		asset.USDTMarginedFutures,
		asset.CoinMarginedFutures:
		ticks, err = by.GetTickers(ctx, getCategoryName(assetType), "", "", time.Time{})
		if err != nil {
			return err
		}
		for x := range ticks.List {
			var pair currency.Pair
			pair, err = by.MatchSymbolWithAvailablePairs(ticks.List[x].Symbol, assetType, true)
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
				ExchangeName: by.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	case asset.Options:
		for x := range supportedOptionsTypes {
			ticks, err = by.GetTickers(ctx, getCategoryName(assetType), "", supportedOptionsTypes[x], time.Time{})
			if err != nil {
				return err
			}
			for x := range ticks.List {
				var pair currency.Pair
				pair, err = by.MatchSymbolWithAvailablePairs(ticks.List[x].Symbol, assetType, true)
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
					ExchangeName: by.Name,
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
func (by *Bybit) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if err := by.UpdateTickers(ctx, assetType); err != nil {
		return nil, err
	}
	return ticker.GetTicker(by.Name, p, assetType)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (by *Bybit) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := by.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	var orderbookNew *Orderbook
	var err error
	p, err = by.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures,
		asset.USDCMarginedFutures,
		asset.CoinMarginedFutures,
		asset.Options:
		if assetType == asset.USDCMarginedFutures && !p.Quote.Equal(currency.PERP) {
			p.Delimiter = currency.DashDelimiter
		}
		orderbookNew, err = by.GetOrderBook(ctx, getCategoryName(assetType), p.String(), 0)
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          by.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: by.ValidateOrderbook,
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
	return orderbook.Get(by.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	var accountType string
	info.Exchange = by.Name
	at, err := by.FetchAccountType(ctx)
	if err != nil {
		return info, err
	}
	switch assetType {
	case asset.Spot, asset.Options,
		asset.USDCMarginedFutures,
		asset.USDTMarginedFutures:
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
		return info, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	balances, err := by.GetWalletBalance(ctx, accountType, "")
	if err != nil {
		return info, err
	}
	currencyBalance := []account.Balance{}
	for i := range balances.List {
		for c := range balances.List[i].Coin {
			balance := account.Balance{
				Currency: balances.List[i].Coin[c].Coin,
				Total:    balances.List[i].Coin[c].WalletBalance.Float64(),
				Free:     balances.List[i].Coin[c].AvailableToWithdraw.Float64(),
				Borrowed: balances.List[i].Coin[c].BorrowAmount.Float64(),
				Hold:     balances.List[i].Coin[c].WalletBalance.Float64() - balances.List[i].Coin[c].AvailableToWithdraw.Float64(),
			}
			if assetType == asset.Spot && balances.List[i].Coin[c].AvailableBalanceForSpot.Float64() != 0 {
				balance.Free = balances.List[i].Coin[c].AvailableBalanceForSpot.Float64()
			}
			currencyBalance = append(currencyBalance, balance)
		}
	}
	acc.Currencies = currencyBalance
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&info, creds)
	if err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (by *Bybit) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (by *Bybit) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	switch a {
	case asset.Spot, asset.Options, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		withdrawals, err := by.GetWithdrawalRecords(ctx, c, "", "2", "", time.Time{}, time.Time{}, 0)
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
func (by *Bybit) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	formattedPair, err := by.FormatExchangeCurrency(p, assetType)
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
		tradeData, err = by.GetPublicTradingHistory(ctx, getCategoryName(assetType), formattedPair.String(), "", "", limit)
	case asset.Options:
		tradeData, err = by.GetPublicTradingHistory(ctx, getCategoryName(assetType), formattedPair.String(), formattedPair.Base.String(), "", limit)
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
			Exchange:     by.Name,
			CurrencyPair: formattedPair,
			AssetType:    assetType,
			Price:        tradeData.List[i].Price.Float64(),
			Amount:       tradeData.List[i].Size.Float64(),
			Timestamp:    tradeData.List[i].TradeTime.Time(),
			TID:          tradeData.List[i].ExecutionID,
			Side:         side,
		}
	}

	if by.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (by *Bybit) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, _, _ time.Time) ([]trade.Data, error) {
	var err error
	p, err = by.FormatExchangeCurrency(p, assetType)
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
		tradeHistoryResponse, err = by.GetPublicTradingHistory(ctx, getCategoryName(assetType), p.String(), "", "", limit)
		if err != nil {
			return nil, err
		}
	case asset.Options:
		tradeHistoryResponse, err = by.GetPublicTradingHistory(ctx, getCategoryName(assetType), p.String(), p.Base.String(), "", limit)
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
			Exchange:     by.Name,
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
func (by *Bybit) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(by.GetTradingRequirements())
	if err != nil {
		return nil, err
	}
	formattedPair, err := by.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	var sideType string
	switch {
	case s.Side.IsLong():
		sideType = sideBuy
	case s.Side.IsShort():
		sideType = sideSell
	default:
		return nil, order.ErrSideIsInvalid
	}
	status := order.New
	switch s.AssetType {
	case asset.Spot, asset.Options, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		if s.AssetType == asset.USDCMarginedFutures && !formattedPair.Quote.Equal(currency.PERP) {
			formattedPair.Delimiter = currency.DashDelimiter
		}
		var response *OrderResponse
		arg := &PlaceOrderParams{
			Category:        getCategoryName(s.AssetType),
			Symbol:          formattedPair,
			Side:            sideType,
			OrderType:       orderTypeToString(s.Type),
			OrderQuantity:   s.Amount,
			Price:           s.Price,
			OrderLinkID:     s.ClientOrderID,
			WhetherToBorrow: s.AssetType == asset.Margin,
			ReduceOnly:      s.ReduceOnly,
			OrderFilter: func() string {
				if s.RiskManagementModes.TakeProfit.Price != 0 || s.RiskManagementModes.TakeProfit.LimitPrice != 0 ||
					s.RiskManagementModes.StopLoss.Price != 0 || s.RiskManagementModes.StopLoss.LimitPrice != 0 {
					return ""
				} else if s.TriggerPrice != 0 {
					return "tpslOrder"
				}
				return "Order"
			}(),
			TriggerPrice: s.TriggerPrice,
		}
		if arg.TriggerPrice != 0 {
			arg.TriggerPriceType = s.TriggerPriceType.String()
		}
		if s.RiskManagementModes.TakeProfit.Price != 0 {
			arg.TakeProfitPrice = s.RiskManagementModes.TakeProfit.Price
			arg.TakeProfitTriggerBy = s.RiskManagementModes.TakeProfit.TriggerPriceType.String()
			arg.TpOrderType = getOrderTypeString(s.RiskManagementModes.TakeProfit.OrderType)
			arg.TpLimitPrice = s.RiskManagementModes.TakeProfit.LimitPrice
		}
		if s.RiskManagementModes.StopLoss.Price != 0 {
			arg.StopLossPrice = s.RiskManagementModes.StopLoss.Price
			arg.StopLossTriggerBy = s.RiskManagementModes.StopLoss.TriggerPriceType.String()
			arg.SlOrderType = getOrderTypeString(s.RiskManagementModes.StopLoss.OrderType)
			arg.SlLimitPrice = s.RiskManagementModes.StopLoss.LimitPrice
		}
		response, err = by.PlaceOrder(ctx, arg)
		if err != nil {
			return nil, err
		}
		resp, err := s.DeriveSubmitResponse(response.OrderID)
		if err != nil {
			return nil, err
		}
		resp.Status = status
		return resp, nil
	default:
		return nil, fmt.Errorf("%s %w", s.AssetType, asset.ErrNotSupported)
	}
}

func getOrderTypeString(oType order.Type) string {
	switch oType {
	case order.UnknownType:
		return ""
	default:
		return oType.String()
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (by *Bybit) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var (
		result *OrderResponse
		err    error
	)
	action.Pair, err = by.FormatExchangeCurrency(action.Pair, action.AssetType)
	if err != nil {
		return nil, err
	}
	switch action.AssetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		if action.AssetType == asset.USDCMarginedFutures && !action.Pair.Quote.Equal(currency.PERP) {
			action.Pair.Delimiter = currency.DashDelimiter
		}
		arg := &AmendOrderParams{
			Category:             getCategoryName(action.AssetType),
			Symbol:               action.Pair,
			OrderID:              action.OrderID,
			OrderLinkID:          action.ClientOrderID,
			OrderQuantity:        action.Amount,
			Price:                action.Price,
			TriggerPrice:         action.TriggerPrice,
			TriggerPriceType:     action.TriggerPriceType.String(),
			TakeProfitPrice:      action.RiskManagementModes.TakeProfit.Price,
			TakeProfitTriggerBy:  getOrderTypeString(action.RiskManagementModes.TakeProfit.OrderType),
			TakeProfitLimitPrice: action.RiskManagementModes.TakeProfit.LimitPrice,
			StopLossPrice:        action.RiskManagementModes.StopLoss.Price,
			StopLossTriggerBy:    action.RiskManagementModes.StopLoss.TriggerPriceType.String(),
			StopLossLimitPrice:   action.RiskManagementModes.StopLoss.LimitPrice,
		}
		result, err = by.AmendOrder(ctx, arg)
		if err != nil {
			return nil, err
		}
	default:
		err = fmt.Errorf("%s %w", action.AssetType, asset.ErrNotSupported)
	}
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

// CancelOrder cancels an order by its corresponding ID number
func (by *Bybit) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	format, err := by.GetPairFormat(ord.AssetType, true)
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		if ord.AssetType == asset.USDCMarginedFutures && !ord.Pair.Quote.Equal(currency.PERP) {
			ord.Pair.Delimiter = currency.DashDelimiter
		}
		_, err = by.CancelTradeOrder(ctx, &CancelOrderParams{
			Category:    getCategoryName(ord.AssetType),
			Symbol:      ord.Pair.Format(format),
			OrderID:     ord.OrderID,
			OrderLinkID: ord.ClientOrderID,
		})
	default:
		return fmt.Errorf("%s %w", ord.AssetType, asset.ErrNotSupported)
	}
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (by *Bybit) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	requests := make([]CancelOrderParams, len(o))
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
			o[i].Pair, err = by.FormatExchangeCurrency(o[i].Pair, category)
			if err != nil {
				return nil, err
			}
			requests[i] = CancelOrderParams{
				OrderID:     o[i].OrderID,
				OrderLinkID: o[i].ClientOrderID,
				Symbol:      o[i].Pair,
			}
		}
	}
	cancelledOrders, err := by.CancelBatchOrder(ctx, &CancelBatchOrder{
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
func (by *Bybit) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	err := orderCancellation.Validate()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	orderCancellation.Pair, err = by.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
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
		activeOrder, err := by.CancelAllTradeOrders(ctx, &CancelAllOrdersParam{
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
func (by *Bybit) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	} else if err := by.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	pair, err := by.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	switch assetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		if assetType == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
			pair.Delimiter = currency.DashDelimiter
		}
		resp, err := by.GetOpenOrders(ctx, getCategoryName(assetType), pair.String(), "", "", orderID, "", "", "", 0, 1)
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
			Exchange:        by.Name,
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
func (by *Bybit) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	dAddressInfo, err := by.GetMasterDepositAddress(ctx, cryptocurrency, chain)
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

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (by *Bybit) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	info, err := by.GetCoinInfo(ctx, cryptocurrency)
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
func (by *Bybit) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	wID, err := by.WithdrawCurrency(ctx,
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
func (by *Bybit) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (by *Bybit) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (by *Bybit) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	format, err := by.GetPairFormat(req.AssetType, true)
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
			openOrders, err := by.GetOpenOrders(ctx, getCategoryName(req.AssetType), "", baseCoin.String(), "", req.FromOrderID, "", "", "", 0, 50)
			if err != nil {
				return nil, err
			}
			newOpenOrders, err := by.ConstructOrderDetails(openOrders.List, req.AssetType, currency.EMPTYPAIR, req.Pairs)
			if err != nil {
				return nil, err
			}
			orders = append(orders, newOpenOrders...)
		} else {
			for y := range req.Pairs {
				if req.AssetType == asset.USDCMarginedFutures && !req.Pairs[y].Quote.Equal(currency.PERP) {
					req.Pairs[y].Delimiter = currency.DashDelimiter
				}
				openOrders, err := by.GetOpenOrders(ctx, getCategoryName(req.AssetType), req.Pairs[y].String(), "", "", req.FromOrderID, "", "", "", 0, 50)
				if err != nil {
					return nil, err
				}
				newOpenOrders, err := by.ConstructOrderDetails(openOrders.List, req.AssetType, req.Pairs[y], currency.Pairs{})
				if err != nil {
					return nil, err
				}
				orders = append(orders, newOpenOrders...)
			}
		}
	default:
		return orders, fmt.Errorf("%s %w", req.AssetType, asset.ErrNotSupported)
	}
	return req.Filter(by.Name, orders), nil
}

// ConstructOrderDetails constructs list of order.Detail instances given list of TradeOrder and other filtering information
func (by *Bybit) ConstructOrderDetails(tradeOrders []TradeOrder, assetType asset.Item, pair currency.Pair, filterPairs currency.Pairs) (order.FilteredOrders, error) {
	orders := make([]order.Detail, 0, len(tradeOrders))
	var err error
	var ePair currency.Pair
	for x := range tradeOrders {
		ePair, err = by.MatchSymbolWithAvailablePairs(tradeOrders[x].Symbol, assetType, true)
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
			Exchange:             by.Name,
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
func (by *Bybit) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	limit := int64(200)
	if req.AssetType == asset.Options {
		limit = 25
	}
	format, err := by.GetPairFormat(req.AssetType, false)
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		resp, err := by.GetTradeOrderHistory(ctx, getCategoryName(req.AssetType), "", req.FromOrderID, "", "", "", "", "", "", req.StartTime, req.EndTime, limit)
		if err != nil {
			return nil, err
		}

		for i := range resp.List {
			// here, we are not using getSide because in sample response's sides are in upper
			var side order.Side
			side, err = order.StringToOrderSide(resp.List[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", by.Name, err)
			}

			var pair currency.Pair
			pair, err = by.MatchSymbolWithAvailablePairs(resp.List[i].Symbol, req.AssetType, true)
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
				Exchange:             by.Name,
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
		resp, err := by.GetTradeOrderHistory(ctx, getCategoryName(req.AssetType), "", req.FromOrderID, "", "", "", "", "", "", req.StartTime, req.EndTime, limit)
		if err != nil {
			return nil, err
		}

		for i := range resp.List {
			// here, we are not using getSide because in sample response's sides are in upper
			var side order.Side
			side, err = order.StringToOrderSide(resp.List[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", by.Name, err)
			}
			var pair currency.Pair
			pair, err = by.MatchSymbolWithAvailablePairs(resp.List[i].Symbol, req.AssetType, true)
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
				Exchange:             by.Name,
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
	return req.Filter(by.Name, orders), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (by *Bybit) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder.Pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	if (!by.AreCredentialsValid(ctx) || by.SkipAuthCheck) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		return getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount), nil
	default:
		assets := by.getCategoryFromPair(feeBuilder.Pair)
		var err error
		var baseCoin, pairString string
		if assets[0] == asset.Options {
			baseCoin = feeBuilder.Pair.Base.String()
		} else {
			pairString, err = by.FormatSymbol(feeBuilder.Pair, assets[0])
			if err != nil {
				return 0, err
			}
		}
		accountFee, err := by.GetFeeRate(ctx, getCategoryName(assets[0]), pairString, baseCoin)
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

func (by *Bybit) getCategoryFromPair(pair currency.Pair) []asset.Item {
	assets := by.GetAssetTypes(true)
	containingAssets := make([]asset.Item, 0, len(assets))
	for a := range assets {
		pairs, err := by.GetAvailablePairs(assets[a])
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
func (by *Bybit) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := by.UpdateAccountInfo(ctx, assetType)
	return by.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (by *Bybit) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
		req, err := by.GetKlineRequest(pair, a, interval, start, end, false)
		if err != nil {
			return nil, err
		}
		var timeSeries []kline.Candle
		if a == asset.USDCMarginedFutures && !pair.Quote.Equal(currency.PERP) {
			req.RequestFormatted.Delimiter = currency.DashDelimiter
		}
		var candles []KlineItem
		candles, err = by.GetKlines(ctx, getCategoryName(req.Asset), req.RequestFormatted.String(), req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
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
func (by *Bybit) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		req, err := by.GetKlineExtendedRequest(pair, a, interval, start, end)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, 0, req.Size())
		for x := range req.RangeHolder.Ranges {
			if req.Asset == asset.USDCMarginedFutures && !req.RequestFormatted.Quote.Equal(currency.PERP) {
				req.RequestFormatted.Delimiter = currency.DashDelimiter
			}
			var klineItems []KlineItem
			klineItems, err = by.GetKlines(ctx,
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
func (by *Bybit) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	info, err := by.GetBybitServerTime(ctx)
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
func (by *Bybit) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var (
		allInstrumentsInfo InstrumentsInfo
		nextPageCursor     string
	)
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		for {
			instrumentInfo, err := by.GetInstrumentInfo(ctx, getCategoryName(a), "", "", "", nextPageCursor, 1000)
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
				instrumentInfo, err := by.GetInstrumentInfo(ctx, getCategoryName(a), "", "", supportedOptionsTypes[i], nextPageCursor, 1000)
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
	limits := make([]order.MinMaxLevel, 0, len(allInstrumentsInfo.List))
	for x := range allInstrumentsInfo.List {
		if allInstrumentsInfo.List[x].Status != "Trading" {
			continue
		}
		symbol := allInstrumentsInfo.List[x].transformSymbol(a)
		pair, err := by.MatchSymbolWithAvailablePairs(symbol, a, true)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s unable to load limits for %s %v, pair data missing", by.Name, a, symbol)
			continue
		}
		limits = append(limits, order.MinMaxLevel{
			Asset:                   a,
			Pair:                    pair,
			MinimumBaseAmount:       allInstrumentsInfo.List[x].LotSizeFilter.MinOrderQty.Float64(),
			MaximumBaseAmount:       allInstrumentsInfo.List[x].LotSizeFilter.MaxOrderQty.Float64(),
			MinPrice:                allInstrumentsInfo.List[x].PriceFilter.MinPrice.Float64(),
			MaxPrice:                allInstrumentsInfo.List[x].PriceFilter.MaxPrice.Float64(),
			PriceStepIncrementSize:  allInstrumentsInfo.List[x].PriceFilter.TickSize.Float64(),
			AmountStepIncrementSize: allInstrumentsInfo.List[x].LotSizeFilter.BasePrecision.Float64(),
			QuoteStepIncrementSize:  allInstrumentsInfo.List[x].LotSizeFilter.QuotePrecision.Float64(),
			MinimumQuoteAmount:      allInstrumentsInfo.List[x].LotSizeFilter.MinOrderQty.Float64() * allInstrumentsInfo.List[x].PriceFilter.MinPrice.Float64(),
			MaximumQuoteAmount:      allInstrumentsInfo.List[x].LotSizeFilter.MaxOrderQty.Float64() * allInstrumentsInfo.List[x].PriceFilter.MaxPrice.Float64(),
		})
	}
	return by.LoadLimits(limits)
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (by *Bybit) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, amount float64, orderSide order.Side) error {
	switch item {
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		var err error
		pair, err = by.FormatExchangeCurrency(pair, item)
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
		return by.SetLeverageLevel(ctx, params)
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (by *Bybit) IsPerpetualFutureCurrency(a asset.Item, p currency.Pair) (bool, error) {
	if !a.IsFutures() {
		return false, nil
	}
	return p.Quote.Equal(currency.PERP) ||
		p.Quote.Equal(currency.USD) ||
		p.Quote.Equal(currency.USDC) ||
		p.Quote.Equal(currency.USDT), nil
}

// GetFuturesContractDetails returns details about futures contracts
func (by *Bybit) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !by.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	inverseContracts, err := by.GetInstrumentInfo(ctx, getCategoryName(item), "", "", "", "", 1000)
	if err != nil {
		return nil, err
	}
	format, err := by.GetPairFormat(item, false)
	if err != nil {
		return nil, err
	}
	switch item {
	case asset.CoinMarginedFutures:
		resp := make([]futures.Contract, 0, len(inverseContracts.List))
		for i := range inverseContracts.List {
			if inverseContracts.List[i].SettleCoin == "USDT" || inverseContracts.List[i].SettleCoin == "USDC" {
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
			var s, e time.Time
			if inverseContracts.List[i].LaunchTime.Time().UnixMilli() > 0 {
				s = inverseContracts.List[i].LaunchTime.Time()
			}
			if inverseContracts.List[i].DeliveryTime.Time().UnixMilli() > 0 {
				e = inverseContracts.List[i].DeliveryTime.Time()
			}

			var ct futures.ContractType
			switch contractType {
			case "inverseperpetual":
				ct = futures.Perpetual
			case "inversefutures":
				ct, err = getContractLength(e.Sub(s))
				if err != nil {
					return nil, fmt.Errorf("%w %v %v %v %v-%v", err, by.Name, item, cp, inverseContracts.List[i].LaunchTime.Time(), inverseContracts.List[i].DeliveryTime)
				}
			default:
				if by.Verbose {
					log.Warnf(log.ExchangeSys, "%v unhandled contract type for %v %v %v-%v", by.Name, item, cp, s, e)
				}
				ct = futures.Unknown
			}

			resp = append(resp, futures.Contract{
				Exchange:             by.Name,
				Name:                 cp.Format(format),
				Underlying:           underlying,
				Asset:                item,
				StartDate:            s,
				EndDate:              e,
				SettlementType:       futures.Inverse,
				IsActive:             strings.EqualFold(inverseContracts.List[i].Status, "trading"),
				Status:               inverseContracts.List[i].Status,
				Type:                 ct,
				SettlementCurrencies: currency.Currencies{currency.NewCode(inverseContracts.List[i].SettleCoin)},
				MaxLeverage:          inverseContracts.List[i].LeverageFilter.MaxLeverage.Float64(),
			})
		}
		return resp, nil
	case asset.USDCMarginedFutures:
		linearContracts, err := by.GetInstrumentInfo(ctx, "linear", "", "", "", "", 1000)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(inverseContracts.List)+len(linearContracts.List))

		var instruments []InstrumentInfo
		for i := range linearContracts.List {
			if linearContracts.List[i].SettleCoin != "USDC" {
				continue
			}
			instruments = append(instruments, linearContracts.List[i])
		}
		for i := range inverseContracts.List {
			if inverseContracts.List[i].SettleCoin != "USDC" {
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
					return nil, fmt.Errorf("%w %v %v %v %v-%v", err, by.Name, item, cp, instruments[i].LaunchTime.Time(), instruments[i].DeliveryTime.Time())
				}
				cp, err = by.MatchSymbolWithAvailablePairs(instruments[i].Symbol, item, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
			default:
				if by.Verbose {
					log.Warnf(log.ExchangeSys, "%v unhandled contract type for %v %v %v-%v", by.Name, item, cp, instruments[i].LaunchTime.Time(), instruments[i].DeliveryTime.Time())
				}
				ct = futures.Unknown
				cp, err = by.MatchSymbolWithAvailablePairs(instruments[i].Symbol, item, true)
				if err != nil {
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
			}

			resp = append(resp, futures.Contract{
				Exchange:             by.Name,
				Name:                 cp.Format(format),
				Underlying:           underlying,
				Asset:                item,
				StartDate:            instruments[i].LaunchTime.Time(),
				EndDate:              instruments[i].DeliveryTime.Time(),
				SettlementType:       futures.Linear,
				IsActive:             strings.EqualFold(instruments[i].Status, "trading"),
				Status:               instruments[i].Status,
				Type:                 ct,
				SettlementCurrencies: currency.Currencies{currency.USDC},
				MaxLeverage:          instruments[i].LeverageFilter.MaxLeverage.Float64(),
				Multiplier:           instruments[i].LeverageFilter.LeverageStep.Float64(),
			})
		}
		return resp, nil
	case asset.USDTMarginedFutures:
		linearContracts, err := by.GetInstrumentInfo(ctx, "linear", "", "", "", "", 1000)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.Contract, 0, len(inverseContracts.List)+len(linearContracts.List))

		var instruments []InstrumentInfo
		for i := range linearContracts.List {
			if linearContracts.List[i].SettleCoin != "USDT" {
				continue
			}
			instruments = append(instruments, linearContracts.List[i])
		}
		for i := range inverseContracts.List {
			if inverseContracts.List[i].SettleCoin != "USDT" {
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
			var s, e time.Time
			if !instruments[i].LaunchTime.Time().IsZero() {
				s = instruments[i].LaunchTime.Time()
			}
			if !instruments[i].DeliveryTime.Time().IsZero() {
				e = instruments[i].DeliveryTime.Time()
			}

			var ct futures.ContractType
			switch contractType {
			case "linearperpetual":
				ct = futures.Perpetual
			case "linearfutures":
				ct, err = getContractLength(e.Sub(s))
				if err != nil {
					return nil, fmt.Errorf("%w %v %v %v %v-%v", err, by.Name, item, cp, s, e)
				}
			default:
				if by.Verbose {
					log.Warnf(log.ExchangeSys, "%v unhandled contract type for %v %v %v-%v", by.Name, item, cp, s, e)
				}
				ct = futures.Unknown
			}

			resp = append(resp, futures.Contract{
				Exchange:             by.Name,
				Name:                 cp.Format(format),
				Underlying:           underlying,
				Asset:                item,
				StartDate:            s,
				EndDate:              e,
				SettlementType:       futures.Linear,
				IsActive:             strings.EqualFold(instruments[i].Status, "trading"),
				Status:               instruments[i].Status,
				Type:                 ct,
				SettlementCurrencies: currency.Currencies{currency.USDT},
				MaxLeverage:          instruments[i].LeverageFilter.MaxLeverage.Float64(),
				Multiplier:           instruments[i].LeverageFilter.LeverageStep.Float64(),
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
func (by *Bybit) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
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
			format, err := by.GetPairFormat(r.Asset, true)
			if err != nil {
				return nil, err
			}
			symbol = r.Pair.Format(format).String()
		}
		ticks, err := by.GetTickers(ctx, getCategoryName(r.Asset), symbol, "", time.Time{})
		if err != nil {
			return nil, err
		}

		instrumentInfo, err := by.GetInstrumentInfo(ctx, getCategoryName(r.Asset), symbol, "", "", "", 1000)
		if err != nil {
			return nil, err
		}

		resp := make([]fundingrate.LatestRateResponse, 0, len(ticks.List))
		for i := range ticks.List {
			var cp currency.Pair
			var isEnabled bool
			cp, isEnabled, err = by.MatchSymbolCheckEnabled(ticks.List[i].Symbol, r.Asset, false)
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
				Exchange:    by.Name,
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
func (by *Bybit) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.USDCMarginedFutures &&
			k[i].Asset != asset.USDTMarginedFutures &&
			k[i].Asset != asset.CoinMarginedFutures {
			return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, k[i].Asset)
		}
	}
	if len(k) == 1 {
		formattedPair, err := by.FormatExchangeCurrency(k[0].Pair(), k[0].Asset)
		if err != nil {
			return nil, err
		}
		if _, parseErr := time.Parse(longDatedFormat, k[0].Quote.Symbol); parseErr == nil {
			// long-dated contracts have a delimiter
			formattedPair.Delimiter = currency.DashDelimiter
		}
		pFmt := formattedPair.String()
		var ticks *TickerData
		ticks, err = by.GetTickers(ctx, getCategoryName(k[0].Asset), pFmt, "", time.Time{})
		if err != nil {
			return nil, err
		}
		for i := range ticks.List {
			if ticks.List[i].Symbol != pFmt {
				continue
			}
			return []futures.OpenInterest{{
				Key: key.ExchangePairAsset{
					Exchange: by.Name,
					Asset:    k[0].Asset,
					Base:     k[0].Base,
					Quote:    k[0].Quote,
				},
				OpenInterest: ticks.List[i].OpenInterest.Float64(),
			}}, nil
		}
	}
	assets := []asset.Item{asset.USDCMarginedFutures, asset.USDTMarginedFutures, asset.CoinMarginedFutures}
	var resp []futures.OpenInterest
	for i := range assets {
		ticks, err := by.GetTickers(ctx, getCategoryName(assets[i]), "", "", time.Time{})
		if err != nil {
			return nil, err
		}
		for x := range ticks.List {
			var pair currency.Pair
			var isEnabled bool
			// only long-dated contracts have a delimiter
			pair, isEnabled, err = by.MatchSymbolCheckEnabled(ticks.List[x].Symbol, assets[i], strings.Contains(ticks.List[x].Symbol, currency.DashDelimiter))
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
				Key: key.ExchangePairAsset{
					Exchange: by.Name,
					Base:     pair.Base.Item,
					Quote:    pair.Quote.Item,
					Asset:    assets[i],
				},
				OpenInterest: ticks.List[i].OpenInterest.Float64(),
			})
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (by *Bybit) GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := by.CurrencyPairs.IsPairEnabled(cp, a)
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
		symbol, err = by.FormatSymbol(cp, a)
		if err != nil {
			return "", err
		}
		// convert long-dated to static contracts
		var io *InstrumentsInfo
		io, err = by.GetInstrumentInfo(ctx, getCategoryName(a), symbol, "", "", "", 1000)
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
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
