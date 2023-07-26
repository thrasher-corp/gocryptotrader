package bybit

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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
func (by *Bybit) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	by.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = by.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = by.BaseCurrencies

	err := by.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if by.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := by.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Bybit
func (by *Bybit) SetDefaults() {
	by.Name = "Bybit"
	by.Enabled = true
	by.Verbose = true
	by.API.CredentialsValidator.RequiresKey = true
	by.API.CredentialsValidator.RequiresSecret = true

	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: ":"}
	requestFormat := &currency.PairFormat{Uppercase: true}
	spotPairStore := currency.PairStore{RequestFormat: requestFormat, ConfigFormat: configFmt}
	err := by.StoreAssetPairFormat(asset.Spot, spotPairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.Spot, err)
	}
	// marginPairStore := currency.PairStore{RequestFormat: requestFormat, ConfigFormat: configFmt}
	// err = by.StoreAssetPairFormat(asset.Margin, marginPairStore)
	// if err != nil {
	// 	log.Errorf(log.ExchangeSys, "%v %v", asset.Margin, err)
	// }
	linearPairStore := currency.PairStore{RequestFormat: requestFormat, ConfigFormat: configFmt}
	err = by.StoreAssetPairFormat(asset.Linear, linearPairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.Linear, err)
	}
	inversePairStore := currency.PairStore{RequestFormat: requestFormat, ConfigFormat: configFmt}
	err = by.StoreAssetPairFormat(asset.Inverse, inversePairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.Inverse, err)
	}
	optionPairStore := currency.PairStore{RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}}
	err = by.StoreAssetPairFormat(asset.Options, optionPairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.Options, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.Inverse)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.Linear)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.Options)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	by.Features = exchange.Features{
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
				GlobalResultLimit: 200,
			},
		},
	}

	by.Requester, err = request.New(by.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	by.API.Endpoints = by.NewEndpoints()
	err = by.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         bybitAPIURL,
		exchange.RestCoinMargined: bybitAPIURL,
		exchange.RestUSDTMargined: bybitAPIURL,
		exchange.RestFutures:      bybitAPIURL,
		exchange.RestUSDCMargined: bybitAPIURL,
		exchange.WebsocketSpot:    bybitWSBaseURL + wsSpotPublicTopicV2,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	by.Websocket = stream.New()
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
		&stream.WebsocketSetup{
			ExchangeConfig:         exch,
			DefaultURL:             bybitWSBaseURL + wsSpotPublicTopicV2,
			RunningURL:             wsRunningEndpoint,
			RunningURLAuth:         bybitWSBaseURL + wsSpotPrivate,
			Connector:              by.WsConnect,
			Subscriber:             by.Subscribe,
			Unsubscriber:           by.Unsubscribe,
			GenerateSubscriptions:  by.GenerateDefaultSubscriptions,
			ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
			Features:               &by.Features.Supports.WebsocketCapabilities,
			OrderbookBufferConfig: buffer.Config{
				SortBuffer:            true,
				SortBufferByUpdateIDs: true,
			},
			TradeFeed: by.Features.Enabled.TradeFeed,
		})
	if err != nil {
		return err
	}

	err = by.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  by.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}

	return by.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  bybitWSBaseURL + wsSpotPrivate,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (by *Bybit) AuthenticateWebsocket(ctx context.Context) error {
	return by.WsAuth(ctx)
}

// Start starts the Bybit go routine
func (by *Bybit) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		by.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the Bybit wrapper
func (by *Bybit) Run(ctx context.Context) {
	if by.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			by.Name,
			common.IsEnabled(by.Websocket.IsEnabled()))
		by.PrintEnabledPairs()
	}

	if !by.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := by.UpdateTradablePairs(ctx, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			by.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (by *Bybit) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	var pair currency.Pair
	switch a {
	case asset.Spot, asset.Linear, asset.Inverse:
		allPairs, err := by.GetInstruments(ctx, getCategoryName(a), "", "Trading", "", "", 0)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, len(allPairs.List))
		for x := range allPairs.List {
			pair, err = currency.NewPairFromString(allPairs.List[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs[x] = pair
		}
		return pairs, nil
	case asset.Options:
		baseCoins := []string{"BTC"}
		pairs := []currency.Pair{}
		for b := range baseCoins {
			allPairs, err := by.GetInstruments(ctx, getCategoryName(a), "", "Trading", baseCoins[0], "", 0)
			if err != nil {
				return nil, err
			}
			println(baseCoins[b], len(allPairs.List))
			for x := range allPairs.List {
				pair, err = currency.NewPairFromString(allPairs.List[x].Symbol)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

func getCategoryName(a asset.Item) string {
	switch a {
	case asset.Spot, asset.Linear, asset.Inverse:
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
	println("\n\n\n", assetTypes.JoinToString(","), "\n\n\n")
	for i := range assetTypes {
		pairs, err := by.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return err
		}
		println(pairs.Join())
		err = by.UpdatePairs(pairs, assetTypes[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return by.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (by *Bybit) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	avail, err := by.GetAvailablePairs(assetType)
	if err != nil {
		return err
	}
	println("\n\n")
	println(avail.Join())
	println("\n\n")

	enabled, err := by.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	var ticks *TickerData
	switch assetType {
	case asset.Spot, asset.Linear, asset.Inverse, asset.Options:
		var baseCoin string
		if assetType == asset.Options {
			baseCoin = "BTC"
		}
		ticks, err = by.GetTickers(ctx, getCategoryName(assetType), "", baseCoin, time.Time{})
		if err != nil {
			return err
		}
		for x := range ticks.List {
			pair, err := avail.DeriveFrom(ticks.List[x].Symbol)
			if err != nil {
				return err
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
				Pair:         pair,
				ExchangeName: by.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
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

// FetchTicker returns the ticker for a currency pair
func (by *Bybit) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := by.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(by.Name, fPair, assetType)
	if err != nil {
		return by.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (by *Bybit) FetchOrderbook(ctx context.Context, currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(by.Name, currency, assetType)
	if err != nil {
		return by.UpdateOrderbook(ctx, currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (by *Bybit) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := by.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	var orderbookNew *Orderbook
	var err error

	formattedPair, err := by.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	switch assetType {
	case asset.Spot, asset.Linear, asset.Inverse, asset.Options:
		orderbookNew, err = by.GetOrderBook(ctx, getCategoryName(assetType), formattedPair.String(), 0)
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	if err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        by.Name,
		Pair:            formattedPair,
		Asset:           assetType,
		VerifyOrderbook: by.CanVerifyOrderbook,
		Bids:            make([]orderbook.Item, len(orderbookNew.Bids)),
		Asks:            make([]orderbook.Item, len(orderbookNew.Asks)),
	}
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(by.Name, formattedPair, assetType)
}

func getAccountType(a asset.Item) string {
	switch a {
	case asset.Spot, asset.Linear, asset.Options:
		return "UNIFIED"
	default:
		return "CONTRACT"
	}
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = by.Name
	switch assetType {
	case asset.Spot, asset.Options, asset.Linear, asset.Inverse:
		balances, err := by.GetWalletBalance(ctx, getAccountType(assetType), "")
		if err != nil {
			return info, err
		}

		currencyBalance := []account.Balance{}
		for i := range balances.List {
			for c := range balances.List[i].Coin {
				balance := account.Balance{
					Currency: currency.NewCode(balances.List[i].Coin[0].Coin),
					Total:    balances.List[i].TotalWalletBalance.Float64(),
					Free:     balances.List[i].Coin[0].AvailableToWithdraw.Float64(),
					// AvailableWithoutBorrow: balances.List[i].Coin[c].AvailableToWithdraw.Float64(),
					Borrowed: balances.List[i].Coin[c].BorrowAmount.Float64(),
					Hold:     balances.List[i].Coin[c].WalletBalance.Float64() - balances.List[i].Coin[c].AvailableToWithdraw.Float64(),
				}
				if assetType == asset.Spot && balances.List[i].Coin[c].AvailableBalanceForSpot.Float64() != 0 {
					balance.Free = balances.List[i].Coin[0].AvailableBalanceForSpot.Float64()
				}
				currencyBalance = append(currencyBalance, balance)
			}
		}
		acc.Currencies = currencyBalance
	default:
		return info, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)

	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(by.Name, creds, assetType)
	if err != nil {
		return by.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (by *Bybit) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (by *Bybit) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	switch a {
	case asset.Spot, asset.Options, asset.Linear, asset.Inverse:
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
	var resp []trade.Data

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
	case asset.Spot, asset.Linear, asset.Inverse:
		tradeData, err = by.GetPublicTradingHistory(ctx, getCategoryName(assetType), formattedPair.String(), "", "", limit)
	case asset.Options:
		tradeData, err = by.GetPublicTradingHistory(ctx, getCategoryName(assetType), formattedPair.String(), formattedPair.Base.String(), "", limit)
	default:
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	if err != nil {
		return nil, err
	}
	for i := range tradeData.List {
		side, err := order.StringToOrderSide(tradeData.List[i].Side)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			Exchange:     by.Name,
			CurrencyPair: formattedPair,
			AssetType:    assetType,
			Price:        tradeData.List[i].Price.Float64(),
			Amount:       tradeData.List[i].Size.Float64(),
			Timestamp:    tradeData.List[i].TradeTime.Time(),
			TID:          tradeData.List[i].ExecutionID,
			Side:         side,
		})
	}

	if by.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(by.Name, resp...)
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
	case asset.Spot, asset.Linear, asset.Inverse:
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

// SubmitOrder submits a new order
func (by *Bybit) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}

	formattedPair, err := by.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	var sideType string
	switch s.Side {
	case order.Buy:
		sideType = sideBuy
	case order.Sell:
		sideType = sideSell
	default:
		return nil, errInvalidSide
	}
	var orderID string
	status := order.New
	switch s.AssetType {
	case asset.Spot, asset.Options, asset.Linear, asset.Inverse:
		by.PlaceOrder(ctx, &PlaceOrderParams{
			Category:        getCategoryName(s.AssetType),
			Symbol:          formattedPair,
			Side:            sideType,
			OrderType:       s.Type.String(),
			OrderQuantity:   s.Amount,
			Price:           s.Price,
			OrderLinkID:     s.ClientOrderID,
			WhetherToBorrow: s.AssetType == asset.Margin,
			ReduceOnly:      s.ReduceOnly,
			// OrderFilter: required if not empty
			// TriggerDirection: s.TriggerPrice.Float64()  1 for increasing 2 for decreasing 0 for none
			TriggerPrice:     s.TriggerPrice,
			TriggerPriceType: s.TriggerPriceType.String(),
			// OrderImpliedVolatility
			// PositionIdx:,
			TakeProfitPrice:     s.RiskManagementModes.TakeProfit.Price,
			TakeProfitTriggerBy: s.RiskManagementModes.TakeProfit.TriggerPriceType.String(),
			StopLossTriggerBy:   s.RiskManagementModes.StopLoss.TriggerPriceType.String(),
			StopLossPrice:       s.RiskManagementModes.StopLoss.Price,
			// SMPExecutionType
			// MarketMakerProtection
			// TpslMode
			TpOrderType:  s.RiskManagementModes.TakeProfit.OrderType.String(),
			SlOrderType:  s.RiskManagementModes.StopLoss.OrderType.String(),
			TpLimitPrice: s.RiskManagementModes.TakeProfit.LimitPrice,
			SlLimitPrice: s.RiskManagementModes.StopLoss.LimitPrice,
		})
		// timeInForce := BybitRequestParamsTimeGTC
		// var requestParamsOrderType string
		// switch s.Type {
		// case order.Market:
		// 	timeInForce = ""
		// 	requestParamsOrderType = BybitRequestParamsOrderMarket
		// case order.Limit:
		// 	requestParamsOrderType = BybitRequestParamsOrderLimit
		// default:
		// 	return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, s.Type)
		// }

		// var orderRequest = PlaceOrderRequest{
		// 	Symbol:      formattedPair.String(),
		// 	Side:        sideType,
		// 	Price:       s.Price,
		// 	Quantity:    s.Amount,
		// 	TradeType:   requestParamsOrderType,
		// 	TimeInForce: timeInForce,
		// 	OrderLinkID: s.ClientOrderID,
		// }
		var response *OrderResponse
		// response, err = by.CreatePostOrder(ctx, &orderRequest)
		// if err != nil {
		// 	return nil, err
		// }
		orderID = response.OrderID
		// if response.ExecutedQty == response.Quantity {
		// 	status = order.Filled
		// }
	default:
		return nil, fmt.Errorf("%s %w", s.AssetType, asset.ErrNotSupported)
	}

	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = status
	return resp, nil
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
	switch action.AssetType {
	case asset.Spot, asset.Linear, asset.Inverse, asset.Options:
		result, err = by.AmendOrder(ctx, &AmendOrderParams{
			Category:             getCategoryName(action.AssetType),
			Symbol:               action.Pair,
			OrderID:              action.OrderID,
			OrderLinkID:          action.ClientOrderID,
			TriggerPrice:         action.TriggerPrice,
			OrderQuantity:        action.Amount,
			Price:                action.Price,
			TakeProfitPrice:      action.RiskManagementModes.TakeProfit.Price,
			StopLossPrice:        action.RiskManagementModes.StopLoss.Price,
			TakeProfitTriggerBy:  action.RiskManagementModes.TakeProfit.OrderType.String(),
			StopLossTriggerBy:    action.RiskManagementModes.StopLoss.TriggerPriceType.String(),
			TriggerPriceType:     action.TriggerPriceType.String(),
			TakeProfitLimitPrice: action.RiskManagementModes.TakeProfit.LimitPrice,
			StopLossLimitPrice:   action.RiskManagementModes.StopLoss.LimitPrice,
		})
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
	var err error
	switch ord.AssetType {
	case asset.Spot, asset.Linear, asset.Inverse, asset.Options:
		_, err = by.CancelTradeOrder(ctx, &CancelOrderParams{
			Category:    getCategoryName(ord.AssetType),
			Symbol:      ord.Pair,
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
	case asset.Spot, asset.Linear, asset.Inverse, asset.Options:
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
	}
	if err := by.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	switch assetType {
	case asset.Spot, asset.Linear, asset.Inverse, asset.Options:
		resp, err := by.GetOpenOrders(ctx, getCategoryName(asset.Spot), pair.String(), "", "", orderID, "", "", "", 0, 1)
		if err != nil {
			return nil, err
		}
		if len(resp.List) != 1 {
			return nil, order.ErrOrderNotFound
		}
		return &order.Detail{
			Amount:         resp.List[0].OrderQuantity.Float64(),
			Exchange:       by.Name,
			OrderID:        resp.List[0].OrderID,
			ClientOrderID:  resp.List[0].OrderLinkID,
			Side:           getSide(resp.List[0].Side),
			Type:           getTradeType(resp.List[0].OrderType),
			Pair:           pair,
			Cost:           resp.List[0].CumulativeExecQuantity.Float64() * resp.List[0].AveragePrice.Float64(),
			AssetType:      assetType,
			Status:         getOrderStatus(resp.List[0].OrderStatus),
			Price:          resp.List[0].Price.Float64(),
			ExecutedAmount: resp.List[0].CumulativeExecQuantity.Float64(),
			Date:           resp.List[0].CreatedTime.Time(),
			LastUpdated:    resp.List[0].UpdatedTime.Time(),
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
		if currency.NewCode(info.Rows[x].Coin) == cryptocurrency {
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

	if len(req.Pairs) == 0 && req.AssetType != asset.Spot {
		return nil, fmt.Errorf("GetActiveOrders: zero pairs found")
	}

	var symbol string
	switch len(req.Pairs) {
	case 0:
		// sending an empty currency pair retrieves data for all currencies
		req.Pairs = append(req.Pairs, currency.EMPTYPAIR)
	case 1:
		symbol, err = by.FormatSymbol(req.Pairs[0], req.AssetType)
		if err != nil {
			return nil, err
		}
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot, asset.Linear, asset.Inverse, asset.Options:
		openOrders, err := by.GetOpenOrders(ctx, getCategoryName(req.AssetType), symbol, "", "", req.FromOrderID, "", "", "", 0, 50)
		if err != nil {
			return nil, err
		}
		for x := range openOrders.List {
			for i := range req.Pairs {
				if req.Pairs[i].String() == openOrders.List[x].Symbol {
					orders = append(orders, order.Detail{
						Amount:               openOrders.List[x].OrderQuantity.Float64(),
						Date:                 openOrders.List[x].CreatedTime.Time(),
						Exchange:             by.Name,
						OrderID:              openOrders.List[x].OrderID,
						ClientOrderID:        openOrders.List[x].OrderLinkID,
						Side:                 getSide(openOrders.List[x].Side),
						Type:                 getTradeType(openOrders.List[x].OrderType),
						Price:                openOrders.List[x].Price.Float64(),
						Status:               getOrderStatus(openOrders.List[x].OrderStatus),
						Pair:                 req.Pairs[i],
						AssetType:            req.AssetType,
						LastUpdated:          openOrders.List[x].UpdatedTime.Time(),
						ReduceOnly:           openOrders.List[x].ReduceOnly,
						ExecutedAmount:       openOrders.List[i].CumulativeExecQuantity.Float64(),
						RemainingAmount:      openOrders.List[i].LeavesQuantity.Float64(),
						TriggerPrice:         openOrders.List[i].TriggerPrice.Float64(),
						AverageExecutedPrice: openOrders.List[i].AveragePrice.Float64(),
						Cost:                 openOrders.List[i].AveragePrice.Float64() * openOrders.List[i].CumulativeExecQuantity.Float64(),
						Fee:                  openOrders.List[i].CumulativeExecFee.Float64(),
					})
				}
			}
		}
	default:
		return orders, fmt.Errorf("%s %w", req.AssetType, asset.ErrNotSupported)
	}
	return req.Filter(by.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (by *Bybit) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Linear, asset.Inverse, asset.Options:
		resp, err := by.GetTradeOrderHistory(ctx, getCategoryName(req.AssetType), "", req.FromOrderID, "", "", "", "", "", "", req.StartTime, req.EndTime, 50)
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
			pair, err = currency.NewPairFromString(resp.List[i].Symbol)
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
				Type:                 getTradeType(resp.List[i].OrderType),
				Price:                resp.List[i].Price.Float64(),
				Pair:                 pair,
				Status:               getOrderStatus(resp.List[i].OrderStatus),
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
		resp, err := by.GetTradeOrderHistory(ctx, getCategoryName(req.AssetType), "", req.FromOrderID, "", "", "", "", "", "", req.StartTime, req.EndTime, 50)
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
			pair, err = currency.NewPairFromString(resp.List[i].Symbol)
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
				Type:                 getTradeType(resp.List[i].OrderType),
				Price:                resp.List[i].Price.Float64(),
				Pair:                 pair,
				Status:               getOrderStatus(resp.List[i].OrderStatus),
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
	// TODO: Upgrade from v1 spot API
	// TODO: give FeeBuilder asset property to distinguish between endpoints
	// results, err := by.GetFeeRate(ctx, feeBuilder)
	return 0, common.ErrFunctionNotSupported
}

// ValidateAPICredentials validates current credentials used for wrapper
func (by *Bybit) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := by.UpdateAccountInfo(ctx, assetType)
	return by.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (by *Bybit) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := by.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	var timeSeries []kline.Candle
	switch req.Asset {
	case asset.Spot, asset.Inverse, asset.Linear:
		var candles []KlineItem
		candles, err = by.GetKlines(ctx, getCategoryName(req.Asset), req.RequestFormatted.String(), req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}

		timeSeries = make([]kline.Candle, len(candles))
		for x := range candles {
			timeSeries[x] = kline.Candle{
				Time:   candles[x].StartTime,
				Open:   candles[x].Open,
				High:   candles[x].High,
				Low:    candles[x].Low,
				Close:  candles[x].Close,
				Volume: candles[x].TradeVolume,
			}
		}
	default:
		return nil, fmt.Errorf("%s %w", req.Asset, asset.ErrNotSupported)
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (by *Bybit) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := by.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		switch req.Asset {
		case asset.Spot, asset.Linear, asset.Inverse:
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
					Time:   klineItems[i].StartTime,
					Open:   klineItems[i].Open,
					High:   klineItems[i].High,
					Low:    klineItems[i].Low,
					Close:  klineItems[i].Close,
					Volume: klineItems[i].TradeVolume,
				})
			}
		default:
			return nil, fmt.Errorf("%s %w", req.Asset, asset.ErrNotSupported)
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetServerTime returns the current exchange server time.
func (by *Bybit) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	info, err := by.GetBybitServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return info.TimeNano.Time(), err
}

func (by *Bybit) extractCurrencyPair(symbol string, item asset.Item) (currency.Pair, error) {
	pairs, err := by.CurrencyPairs.GetPairs(item, true)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	return pairs.DeriveFrom(symbol)
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (by *Bybit) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	avail, err := by.GetAvailablePairs(a)
	if err != nil {
		return err
	}
	var limits []order.MinMaxLevel
	var instrumentsInfo *InstrumentsInfo
	switch a {
	case asset.Spot, asset.Linear, asset.Inverse:
		instrumentsInfo, err = by.GetInstruments(ctx, getCategoryName(a), "", "", "", "", 400)
		if err != nil {
			return err
		}
	case asset.Options:
		instrumentsInfo, err = by.GetInstruments(ctx, getCategoryName(a), "", "", "BTC", "", 400)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	limits = make([]order.MinMaxLevel, 0, len(instrumentsInfo.List))
	for x := range instrumentsInfo.List {
		var pair currency.Pair
		pair, err = avail.DeriveFrom(instrumentsInfo.List[x].Symbol)
		if err != nil {
			return err
		}

		limits = append(limits, order.MinMaxLevel{
			Asset:                   a,
			Pair:                    pair,
			MinimumBaseAmount:       instrumentsInfo.List[x].LotSizeFilter.MinOrderQty.Float64(),
			MaximumBaseAmount:       instrumentsInfo.List[x].LotSizeFilter.MaxOrderQty.Float64(),
			MinPrice:                instrumentsInfo.List[x].PriceFilter.MinPrice.Float64(),
			MaxPrice:                instrumentsInfo.List[x].PriceFilter.MaxPrice.Float64(),
			PriceStepIncrementSize:  instrumentsInfo.List[x].PriceFilter.TickSize.Float64(),
			AmountStepIncrementSize: instrumentsInfo.List[x].LotSizeFilter.QtyStep.Float64(),
		})
	}
	return by.LoadLimits(limits)
}
