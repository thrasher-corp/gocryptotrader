package bybit

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
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

	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.ColonDelimiter}
	requestFormat := &currency.PairFormat{Uppercase: true}
	spotPairStore := currency.PairStore{RequestFormat: requestFormat, ConfigFormat: configFmt}
	err := by.StoreAssetPairFormat(asset.Spot, spotPairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.Spot, err)
	}
	linearPairStore := currency.PairStore{RequestFormat: requestFormat, ConfigFormat: configFmt}
	err = by.StoreAssetPairFormat(asset.USDTMarginedFutures, linearPairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.USDTMarginedFutures, err)
	}
	err = by.StoreAssetPairFormat(asset.USDCMarginedFutures, linearPairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.USDCMarginedFutures, err)
	}
	inversePairStore := currency.PairStore{RequestFormat: requestFormat, ConfigFormat: configFmt}
	err = by.StoreAssetPairFormat(asset.CoinMarginedFutures, inversePairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.CoinMarginedFutures, err)
	}
	optionPairStore := currency.PairStore{RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}, ConfigFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}}
	err = by.StoreAssetPairFormat(asset.Options, optionPairStore)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%v %v", asset.Options, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.CoinMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = by.DisableAssetWebsocketSupport(asset.USDTMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = by.DisableAssetWebsocketSupport(asset.USDCMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = by.DisableAssetWebsocketSupport(asset.Options)
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
					kline.IntervalCapacity{Interval: kline.SevenHour},
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
		exchange.WebsocketSpot:    spotPublic,
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
			ExchangeConfig:        exch,
			DefaultURL:            spotPublic,
			RunningURL:            wsRunningEndpoint,
			RunningURLAuth:        websocketPrivate,
			Connector:             by.WsConnect,
			Subscriber:            by.Subscribe,
			Unsubscriber:          by.Unsubscribe,
			GenerateSubscriptions: by.GenerateDefaultSubscriptions,
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
	err = by.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  by.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     bybitWebsocketTimer,
	})
	if err != nil {
		return err
	}

	return by.Websocket.SetupNewConnection(stream.ConnectionSetup{
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

	by.RetrieveAndSetAccountType(ctx)
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
	var category string
	switch a {
	case asset.Spot, asset.Options, asset.CoinMarginedFutures, asset.USDCMarginedFutures, asset.USDTMarginedFutures:
		category = getCategoryName(a)
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	allPairs, err := by.GetInstruments(ctx, category, "", "Trading", "", "", 0)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(allPairs.List))
	switch a {
	case asset.Spot, asset.Options:
		for x := range allPairs.List {
			pair, err = currency.NewPairFromString(allPairs.List[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.CoinMarginedFutures:
		for x := range allPairs.List {
			if allPairs.List[x].Status != "Trading" || allPairs.List[x].QuoteCoin != "USD" {
				continue
			}
			pair, err = currency.NewPairFromString(allPairs.List[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.USDCMarginedFutures:
		for x := range allPairs.List {
			if allPairs.List[x].Status != "Trading" || allPairs.List[x].QuoteCoin != "USDC" {
				continue
			}
			pair, err = currency.NewPairFromString(allPairs.List[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	case asset.USDTMarginedFutures:
		for x := range allPairs.List {
			if allPairs.List[x].Status != "Trading" || allPairs.List[x].QuoteCoin != "USDT" {
				continue
			}
			pair, err = currency.NewPairFromString(allPairs.List[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	}
	return pairs, nil
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
	var ticks *TickerData
	switch assetType {
	case asset.Spot, asset.USDCMarginedFutures,
		asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		var baseCoin string
		if assetType == asset.Options {
			baseCoin = "BTC"
		}
		ticks, err = by.GetTickers(ctx, getCategoryName(assetType), "", baseCoin, time.Time{})
		if err != nil {
			return err
		}
		for x := range ticks.List {
			var pair currency.Pair
			pair, err = currency.NewPairFromString(ticks.List[x].Symbol)
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
	case asset.Spot, asset.USDTMarginedFutures,
		asset.USDCMarginedFutures,
		asset.CoinMarginedFutures,
		asset.Options:
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

// UpdateAccountInfo retrieves balances for all enabled currencies
func (by *Bybit) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	var accountType string
	info.Exchange = by.Name
	switch assetType {
	case asset.Spot, asset.Options,
		asset.USDCMarginedFutures,
		asset.USDTMarginedFutures:
		switch by.AccountType {
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
				Currency: currency.NewCode(balances.List[i].Coin[0].Coin),
				Total:    balances.List[i].TotalWalletBalance.Float64(),
				Free:     balances.List[i].Coin[0].AvailableToWithdraw.Float64(),
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
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
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
	err := s.Validate()
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
		return nil, errInvalidSide
	}
	status := order.New
	switch s.AssetType {
	case asset.Spot, asset.Options, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
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
	var err error
	switch ord.AssetType {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
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
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
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
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		resp, err := by.GetOpenOrders(ctx, getCategoryName(asset.Spot), pair.String(), "", "", orderID, "", "", "", 0, 1)
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
		return &order.Detail{
			Amount:         resp.List[0].OrderQuantity.Float64(),
			Exchange:       by.Name,
			OrderID:        resp.List[0].OrderID,
			ClientOrderID:  resp.List[0].OrderLinkID,
			Side:           getSide(resp.List[0].Side),
			Type:           orderType,
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
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
		openOrders, err := by.GetOpenOrders(ctx, getCategoryName(req.AssetType), symbol, "", "", req.FromOrderID, "", "", "", 0, 50)
		if err != nil {
			return nil, err
		}
		for x := range openOrders.List {
			for i := range req.Pairs {
				if req.Pairs[i].String() == openOrders.List[x].Symbol {
					orderType, err := order.StringToOrderType(openOrders.List[x].OrderType)
					if err != nil {
						return nil, err
					}
					orders = append(orders, order.Detail{
						Amount:               openOrders.List[x].OrderQuantity.Float64(),
						Date:                 openOrders.List[x].CreatedTime.Time(),
						Exchange:             by.Name,
						OrderID:              openOrders.List[x].OrderID,
						ClientOrderID:        openOrders.List[x].OrderLinkID,
						Side:                 getSide(openOrders.List[x].Side),
						Type:                 orderType,
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
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures, asset.Options:
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
	req, err := by.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	var timeSeries []kline.Candle
	switch req.Asset {
	case asset.Spot, asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.USDCMarginedFutures:
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
		case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
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

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (by *Bybit) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	avail, err := by.GetAvailablePairs(a)
	if err != nil {
		return err
	}
	var instrumentsInfo *InstrumentsInfo
	switch a {
	case asset.Spot, asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
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
	limits := make([]order.MinMaxLevel, 0, len(instrumentsInfo.List))
	for x := range instrumentsInfo.List {
		var pair currency.Pair
		pair, err = avail.DeriveFrom(instrumentsInfo.List[x].Symbol)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s unable to load limits for %v, pair data missing", by.Name, instrumentsInfo.List[x].Symbol)
			continue
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
			QuoteStepIncrementSize:  instrumentsInfo.List[x].PriceFilter.TickSize.Float64(),
			MinimumQuoteAmount:      instrumentsInfo.List[x].LotSizeFilter.MinOrderQty.Float64() * instrumentsInfo.List[x].PriceFilter.MinPrice.Float64(),
			MaximumQuoteAmount:      instrumentsInfo.List[x].LotSizeFilter.MaxOrderQty.Float64() * instrumentsInfo.List[x].PriceFilter.MaxPrice.Float64(),
		})
	}
	return by.LoadLimits(limits)
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (by *Bybit) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, amount float64, orderSide order.Side) error {
	switch item {
	case asset.USDTMarginedFutures, asset.USDCMarginedFutures, asset.CoinMarginedFutures:
		symbol, err := by.FormatSymbol(pair, item)
		if err != nil {
			return err
		}
		params := &SetLeverageParams{
			Category: getCategoryName(item),
			Symbol:   symbol,
		}
		switch orderSide {
		case order.Buy, order.Sell:
			// Unified account: buyLeverage must be the same as sellLeverage all the time
			// Classic account: under one-way mode, buyLeverage must be the same as sellLeverage
			params.BuyLeverage, params.SellLeverage = amount, amount
		case order.UnknownSide:
			return errOrderSideRequired
		default:
			return order.ErrSideIsInvalid
		}
		return by.SetLeverageLevel(ctx, params)
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetFuturesContractDetails returns details about futures contracts
func (by *Bybit) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !by.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}

	inverseContracts, err := by.GetInstruments(ctx, getCategoryName(item), "", "", "", "", 1000)
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
			splitCoin := strings.Split(inverseContracts.List[i].Symbol, inverseContracts.List[i].BaseCoin)
			if len(splitCoin) <= 1 {
				continue
			}

			cp, err = currency.NewPairFromStrings(inverseContracts.List[i].BaseCoin, splitCoin[1])
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
				Name:                 cp,
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
		linearContracts, err := by.GetInstruments(ctx, "linear", "", "", "", "", 1000)
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
				splitCoin := strings.Split(instruments[i].Symbol, instruments[i].BaseCoin)
				if len(splitCoin) <= 1 {
					continue
				}
				cp, err = currency.NewPairFromStrings(instruments[i].BaseCoin, splitCoin[1])
				if err != nil {
					return nil, err
				}
			case "linearfutures":
				ct, err = getContractLength(instruments[i].DeliveryTime.Time().Sub(instruments[i].LaunchTime.Time()))
				if err != nil {
					return nil, fmt.Errorf("%w %v %v %v %v-%v", err, by.Name, item, cp, instruments[i].LaunchTime.Time(), instruments[i].DeliveryTime.Time())
				}
				cp, err = currency.NewPairFromString(instruments[i].Symbol)
				if err != nil {
					return nil, err
				}
			default:
				if by.Verbose {
					log.Warnf(log.ExchangeSys, "%v unhandled contract type for %v %v %v-%v", by.Name, item, cp, instruments[i].LaunchTime.Time(), instruments[i].DeliveryTime.Time())
				}
				ct = futures.Unknown
				cp, err = currency.NewPairFromString(instruments[i].Symbol)
				if err != nil {
					return nil, err
				}
			}

			resp = append(resp, futures.Contract{
				Exchange:             by.Name,
				Name:                 cp,
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
		linearContracts, err := by.GetInstruments(ctx, "linear", "", "", "", "", 1000)
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
			splitCoin := strings.Split(instruments[i].Symbol, instruments[i].BaseCoin)
			if len(splitCoin) <= 1 {
				continue
			}
			var cp, underlying currency.Pair
			cp, err = currency.NewPairFromStrings(instruments[i].BaseCoin, splitCoin[1])
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
				Name:                 cp,
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
	case contractLength <= kline.ThreeMonth.Duration()+kline.ThreeWeek.Duration():
		ct = futures.Quarterly
	case contractLength <= kline.SixMonth.Duration()+kline.ThreeWeek.Duration():
		ct = futures.HalfYearly
	case contractLength <= kline.NineMonth.Duration()+kline.ThreeWeek.Duration():
		ct = futures.NineMonthly
	default:
		ct = futures.SemiAnnually
	}
	return ct, nil
}
