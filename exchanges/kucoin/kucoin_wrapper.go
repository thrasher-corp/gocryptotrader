package kucoin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (ku *Kucoin) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	ku.SetDefaults()
	exchCfg, err := ku.GetStandardConfig()
	if err != nil {
		return nil, err
	}

	err = ku.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if ku.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := ku.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Kucoin
func (ku *Kucoin) SetDefaults() {
	ku.Name = "Kucoin"
	ku.Enabled = true
	ku.Verbose = false

	ku.API.CredentialsValidator.RequiresKey = true
	ku.API.CredentialsValidator.RequiresSecret = true
	ku.API.CredentialsValidator.RequiresClientID = true

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}
	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	}
	err := ku.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = ku.StoreAssetPairFormat(asset.Margin, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = ku.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	ku.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TickerBatching:    true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				CryptoWithdrawal:  true,
				SubmitOrder:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrder:       true,
				CancelOrders:      true,
				TradeFetching:     true,
				UserTradeHistory:  true,
				KlineFetching:     true,
				DepositHistory:    true,
				WithdrawalHistory: true,
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
				CollateralMode:            true,
				FundingRates:              true,
				MaximumFundingRateHistory: kline.ThreeMonth.Duration(),
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
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
				GlobalResultLimit: 1500,
			},
		},
	}
	ku.Requester, err = request.New(ku.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	ku.API.Endpoints = ku.NewEndpoints()
	err = ku.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      kucoinAPIURL,
		exchange.RestFutures:   kucoinFuturesAPIURL,
		exchange.WebsocketSpot: kucoinWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	ku.Websocket = stream.New()
	ku.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	ku.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	ku.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (ku *Kucoin) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		ku.SetEnabled(false)
		return nil
	}
	err = ku.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := ku.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = ku.Websocket.Setup(
		&stream.WebsocketSetup{
			ExchangeConfig:        exch,
			DefaultURL:            kucoinWebsocketURL,
			RunningURL:            wsRunningEndpoint,
			Connector:             ku.WsConnect,
			Subscriber:            ku.Subscribe,
			Unsubscriber:          ku.Unsubscribe,
			GenerateSubscriptions: ku.GenerateDefaultSubscriptions,
			Features:              &ku.Features.Supports.WebsocketCapabilities,
			OrderbookBufferConfig: buffer.Config{
				SortBuffer:            true,
				SortBufferByUpdateIDs: true,
				UpdateIDProgression:   true,
			},
			TradeFeed: ku.Features.Enabled.TradeFeed,
		})
	if err != nil {
		return err
	}
	return ku.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            500,
	})
}

// Start starts the Kucoin go routine
func (ku *Kucoin) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		ku.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the Kucoin wrapper
func (ku *Kucoin) Run(ctx context.Context) {
	if ku.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			ku.Name,
			common.IsEnabled(ku.Websocket.IsEnabled()))
		ku.PrintEnabledPairs()
	}

	assetTypes := ku.GetAssetTypes(false)
	for i := range assetTypes {
		if err := ku.UpdateOrderExecutionLimits(ctx, assetTypes[i]); err != nil && !errors.Is(err, common.ErrNotYetImplemented) {
			log.Errorf(log.ExchangeSys,
				"%s failed to set exchange order execution limits. Err: %v",
				ku.Name,
				err)
		}
	}

	if !ku.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := ku.UpdateTradablePairs(ctx, true)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			ku.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (ku *Kucoin) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	var cp currency.Pair
	switch assetType {
	case asset.Futures:
		myPairs, err := ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(myPairs))
		for x := range myPairs {
			if strings.ToLower(myPairs[x].Status) != "open" { //nolint:gocritic // strings.ToLower is faster
				continue
			}
			cp, err = currency.NewPairFromStrings(myPairs[x].BaseCurrency, myPairs[x].Symbol[len(myPairs[x].BaseCurrency):])
			if err != nil {
				return nil, err
			}
			pairs = pairs.Add(cp)
		}
		configFormat, err := ku.GetPairFormat(asset.Futures, false)
		if err != nil {
			return nil, err
		}
		return pairs.Format(configFormat), nil
	case asset.Spot, asset.Margin:
		myPairs, err := ku.GetSymbols(ctx, "")
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
			cp, err = currency.NewPairFromString(strings.ToUpper(myPairs[x].Symbol))
			if err != nil {
				return nil, err
			}
			pairs = pairs.Add(cp)
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (ku *Kucoin) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := ku.GetAssetTypes(true)
	for a := range assets {
		pairs, err := ku.FetchTradablePairs(ctx, assets[a])
		if err != nil {
			return err
		}
		if len(pairs) == 0 {
			return fmt.Errorf("%v; no tradable pairs", currency.ErrCurrencyPairsEmpty)
		}
		err = ku.UpdatePairs(pairs, assets[a], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (ku *Kucoin) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	p, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if err := ku.UpdateTickers(ctx, assetType); err != nil {
		return nil, err
	}
	return ticker.GetTicker(ku.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (ku *Kucoin) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Futures:
		ticks, err := ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return err
		}
		pairs, err := ku.GetEnabledPairs(asset.Futures)
		if err != nil {
			return err
		}
		for x := range ticks {
			var pair currency.Pair
			pair, err = currency.NewPairFromStrings(ticks[x].BaseCurrency, ticks[x].Symbol[len(ticks[x].BaseCurrency):])
			if err != nil {
				return err
			}
			if !pairs.Contains(pair, true) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks[x].LastTradePrice,
				High:         ticks[x].HighPrice,
				Low:          ticks[x].LowPrice,
				Volume:       ticks[x].VolumeOf24h,
				OpenInterest: ticks[x].OpenInterest.Float64(),
				Pair:         pair,
				ExchangeName: ku.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
		return nil
	case asset.Spot, asset.Margin:
		ticks, err := ku.GetTickers(ctx)
		if err != nil {
			return err
		}
		pairs, err := ku.GetEnabledPairs(assetType)
		if err != nil {
			return err
		}
		for t := range ticks.Tickers {
			pair, err := currency.NewPairFromString(ticks.Tickers[t].Symbol)
			if err != nil {
				return err
			}
			if !pairs.Contains(pair, true) {
				continue
			}
			for _, assetType := range ku.listOfAssetsCurrencyPairEnabledFor(pair) {
				err = ticker.ProcessTicker(&ticker.Price{
					Last:         ticks.Tickers[t].Last,
					High:         ticks.Tickers[t].High,
					Low:          ticks.Tickers[t].Low,
					Volume:       ticks.Tickers[t].Volume,
					Ask:          ticks.Tickers[t].Sell,
					Bid:          ticks.Tickers[t].Buy,
					Pair:         pair,
					ExchangeName: ku.Name,
					AssetType:    assetType,
					LastUpdated:  ticks.Time.Time(),
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

// FetchTicker returns the ticker for a currency pair
func (ku *Kucoin) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	p, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tickerNew, err := ticker.GetTicker(ku.Name, p, assetType)
	if err != nil {
		return ku.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (ku *Kucoin) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	pair, err := ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	ob, err := orderbook.Get(ku.Name, pair, assetType)
	if err != nil {
		return ku.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (ku *Kucoin) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	err := ku.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	pair, err = ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	var ordBook *Orderbook
	switch assetType {
	case asset.Futures:
		ordBook, err = ku.GetFuturesOrderbook(ctx, pair.String())
	case asset.Spot, asset.Margin:
		if ku.IsRESTAuthenticationSupported() && ku.AreCredentialsValid(ctx) {
			ordBook, err = ku.GetOrderbook(ctx, pair.String())
			if err != nil {
				return nil, err
			}
		} else {
			ordBook, err = ku.GetPartOrderbook100(ctx, pair.String())
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if err != nil {
		return nil, err
	}

	book := &orderbook.Base{
		Exchange:        ku.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: ku.CanVerifyOrderbook,
		Asks:            ordBook.Asks,
		Bids:            ordBook.Bids,
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(ku.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (ku *Kucoin) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	holding := account.Holdings{
		Exchange: ku.Name,
	}
	err := ku.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return holding, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	switch assetType {
	case asset.Futures:
		accountH, err := ku.GetFuturesAccountOverview(ctx, "")
		if err != nil {
			return account.Holdings{}, err
		}
		holding.Accounts = append(holding.Accounts, account.SubAccount{
			AssetType: assetType,
			Currencies: []account.Balance{{
				Currency: currency.NewCode(accountH.Currency),
				Total:    accountH.AvailableBalance + accountH.FrozenFunds,
				Hold:     accountH.FrozenFunds,
				Free:     accountH.AvailableBalance,
			}},
		})
	case asset.Spot, asset.Margin:
		accountH, err := ku.GetAllAccounts(ctx, "", ku.accountTypeToString(assetType))
		if err != nil {
			return account.Holdings{}, err
		}
		for x := range accountH {
			holding.Accounts = append(holding.Accounts, account.SubAccount{
				AssetType: assetType,
				Currencies: []account.Balance{
					{
						Currency: currency.NewCode(accountH[x].Currency),
						Total:    accountH[x].Balance,
						Hold:     accountH[x].Holds,
						Free:     accountH[x].Available,
					}},
			})
		}
	default:
		return holding, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return holding, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (ku *Kucoin) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := ku.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(ku.Name, creds, assetType)
	if err != nil {
		return ku.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (ku *Kucoin) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	withdrawalsData, err := ku.GetWithdrawalList(ctx, "", "", time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	depositsData, err := ku.GetHistoricalDepositList(ctx, "", "", time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	fundingData := make([]exchange.FundingHistory, len(withdrawalsData.Items)+len(depositsData.Items))
	for x := range depositsData.Items {
		fundingData[x] = exchange.FundingHistory{
			Timestamp:    depositsData.Items[x].CreatedAt.Time(),
			ExchangeName: ku.Name,
			TransferType: "deposit",
			CryptoTxID:   depositsData.Items[x].WalletTxID,
			Status:       depositsData.Items[x].Status,
			Amount:       depositsData.Items[x].Amount,
			Currency:     depositsData.Items[x].Currency,
		}
	}
	length := len(depositsData.Items)
	for x := range withdrawalsData.Items {
		fundingData[length+x] = exchange.FundingHistory{
			Fee:             withdrawalsData.Items[x].Fee,
			Timestamp:       withdrawalsData.Items[x].UpdatedAt.Time(),
			ExchangeName:    ku.Name,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalsData.Items[x].Address,
			CryptoTxID:      withdrawalsData.Items[x].WalletTxID,
			Status:          withdrawalsData.Items[x].Status,
			Amount:          withdrawalsData.Items[x].Amount,
			Currency:        withdrawalsData.Items[x].Currency,
			TransferID:      withdrawalsData.Items[x].ID,
		}
	}
	return fundingData, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (ku *Kucoin) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	err := ku.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		var withdrawals *HistoricalDepositWithdrawalResponse
		withdrawals, err = ku.GetHistoricalWithdrawalList(ctx, c.String(), "", time.Time{}, time.Time{}, 0, 0)
		if err != nil {
			return nil, err
		}
		resp := make([]exchange.WithdrawalHistory, len(withdrawals.Items))
		for x := range withdrawals.Items {
			resp[x] = exchange.WithdrawalHistory{
				Status:       withdrawals.Items[x].Status,
				CryptoTxID:   withdrawals.Items[x].WalletTxID,
				Timestamp:    withdrawals.Items[x].CreatedAt.Time(),
				Amount:       withdrawals.Items[x].Amount,
				TransferType: "withdrawal",
				Currency:     c.String(),
			}
		}
		return resp, nil
	case asset.Futures:
		var futuresWithdrawals *FuturesWithdrawalsListResponse
		futuresWithdrawals, err = ku.GetFuturesWithdrawalList(ctx, c.String(), "", time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp := make([]exchange.WithdrawalHistory, len(futuresWithdrawals.Items))
		for y := range futuresWithdrawals.Items {
			resp[y] = exchange.WithdrawalHistory{
				Status:       futuresWithdrawals.Items[y].Status,
				CryptoTxID:   futuresWithdrawals.Items[y].WalletTxID,
				Timestamp:    futuresWithdrawals.Items[y].CreatedAt.Time(),
				Amount:       futuresWithdrawals.Items[y].Amount,
				Currency:     c.String(),
				TransferType: "withdrawal",
			}
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("withdrawal %w for asset type %v", asset.ErrNotSupported, a)
	}
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (ku *Kucoin) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Futures:
		tradeData, err := ku.GetFuturesTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		var side order.Side
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[0].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          tradeData[i].TradeID,
				Exchange:     ku.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].FilledTime.Time(),
				Side:         side,
			})
		}
	case asset.Spot, asset.Margin:
		tradeData, err := ku.GetTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		var side order.Side
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[0].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          tradeData[i].Sequence,
				Exchange:     ku.Name,
				CurrencyPair: p,
				Side:         side,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].Time.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if ku.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(ku.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (ku *Kucoin) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (ku *Kucoin) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	sideString, err := ku.orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	if s.Type != order.UnknownType && s.Type != order.Limit && s.Type != order.Market {
		return nil, fmt.Errorf("%w only limit and market are supported", order.ErrTypeIsInvalid)
	}
	s.Pair, err = ku.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Futures:
		if s.Leverage == 0 {
			s.Leverage = 1
		}
		o, err := ku.PostFuturesOrder(ctx, &FuturesOrderParam{
			ClientOrderID: s.ClientOrderID, Side: sideString, Symbol: s.Pair,
			OrderType: s.Type.Lower(), Size: s.Amount, Price: s.Price, StopPrice: s.TriggerPrice,
			Leverage: s.Leverage, VisibleSize: 0, ReduceOnly: s.ReduceOnly,
			PostOnly: s.PostOnly, Hidden: s.Hidden})
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(o)
	case asset.Spot:
		timeInForce := ""
		if s.Type == order.Limit {
			switch {
			case s.FillOrKill:
				timeInForce = "FOK"
			case s.ImmediateOrCancel:
				timeInForce = "IOC"
			case s.PostOnly:
			default:
				timeInForce = "GTC"
			}
		}
		o, err := ku.PostOrder(ctx, &SpotOrderParam{
			ClientOrderID: s.ClientOrderID,
			Side:          sideString,
			Symbol:        s.Pair,
			OrderType:     s.Type.Lower(),
			Size:          s.Amount,
			Price:         s.Price,
			PostOnly:      s.PostOnly,
			Hidden:        s.Hidden,
			TimeInForce:   timeInForce,
		})
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(o)
	case asset.Margin:
		o, err := ku.PostMarginOrder(ctx,
			&MarginOrderParam{ClientOrderID: s.ClientOrderID,
				Side: sideString, Symbol: s.Pair,
				OrderType: s.Type.Lower(), MarginMode: marginModeToString(s.MarginType),
				Price: s.Price, Size: s.Amount,
				VisibleSize: s.Amount, PostOnly: s.PostOnly,
				Hidden: s.Hidden, AutoBorrow: s.AutoBorrow})
		if err != nil {
			return nil, err
		}
		ret, err := s.DeriveSubmitResponse(o.OrderID)
		if err != nil {
			return nil, err
		}
		ret.BorrowSize = o.BorrowSize
		ret.LoanApplyID = o.LoanApplyID
		return ret, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, s.AssetType)
	}
}

func marginModeToString(mType margin.Type) string {
	switch mType {
	case margin.Isolated:
		return mType.String()
	case margin.Multi:
		return "cross"
	default:
		return ""
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (ku *Kucoin) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (ku *Kucoin) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if ord == nil {
		return common.ErrNilPointer
	}
	err := ku.CurrencyPairs.IsAssetEnabled(ord.AssetType)
	if err != nil {
		return err
	}
	err = ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot, asset.Margin:
		if ord.OrderID == "" && ord.ClientOrderID == "" {
			return errors.New("either OrderID or ClientSuppliedOrderID must be specified")
		}
		if ord.OrderID != "" {
			_, err = ku.CancelSingleOrder(ctx, ord.OrderID)
		} else if ord.ClientOrderID != "" || ord.ClientID != "" {
			if ord.ClientID != "" && ord.ClientOrderID == "" {
				ord.ClientOrderID = ord.ClientID
			}
			_, err = ku.CancelOrderByClientOID(ctx, ord.ClientOrderID)
		}
		return err
	case asset.Futures:
		_, err := ku.CancelFuturesOrder(ctx, ord.OrderID)
		if err != nil {
			return err
		}
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (ku *Kucoin) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (ku *Kucoin) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if orderCancellation == nil {
		return order.CancelAllResponse{}, common.ErrNilPointer
	}
	if err := ku.CurrencyPairs.IsAssetEnabled(orderCancellation.AssetType); err != nil {
		return order.CancelAllResponse{}, err
	}
	result := order.CancelAllResponse{}
	err := orderCancellation.Validate()
	if err != nil {
		return result, err
	}
	var pairString string
	if !orderCancellation.Pair.IsEmpty() {
		orderCancellation.Pair, err = ku.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
		if err != nil {
			return result, err
		}
		pairString = orderCancellation.Pair.String()
	}
	var values []string
	switch orderCancellation.AssetType {
	case asset.Margin, asset.Spot:
		tradeType := ku.accountToTradeTypeString(orderCancellation.AssetType, marginModeToString(orderCancellation.MarginType))
		values, err = ku.CancelAllOpenOrders(ctx, pairString, tradeType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
	case asset.Futures:
		values, err = ku.CancelAllFuturesOpenOrders(ctx, orderCancellation.Pair.String())
		if err != nil {
			return result, err
		}
		stopOrders, err := ku.CancelAllFuturesStopOrders(ctx, orderCancellation.Pair.String())
		if err != nil {
			return result, err
		}
		values = append(values, stopOrders...)
	default:
		return order.CancelAllResponse{}, fmt.Errorf("%w %v", asset.ErrNotSupported, orderCancellation.AssetType)
	}
	result.Status = map[string]string{}
	for x := range values {
		result.Status[values[x]] = order.Cancelled.String()
	}
	return result, nil
}

// GetOrderInfo returns order information based on order ID
func (ku *Kucoin) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if err := ku.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	pair, err := ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Futures:
		orderDetail, err := ku.GetFuturesOrderDetails(ctx, orderID)
		if err != nil {
			return nil, err
		}
		var nPair currency.Pair
		nPair, err = ku.MatchSymbolWithAvailablePairs(orderDetail.Symbol, assetType, true)
		if err != nil {
			return nil, err
		}
		oType, err := order.StringToOrderType(orderDetail.OrderType)
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return nil, err
		}
		if !pair.IsEmpty() && !nPair.Equal(pair) {
			return nil, fmt.Errorf("order with id %s and currency Pair %v does not exist", orderID, pair)
		}
		return &order.Detail{
			Exchange:        ku.Name,
			OrderID:         orderDetail.ID,
			Pair:            pair,
			Type:            oType,
			Side:            side,
			AssetType:       assetType,
			ExecutedAmount:  orderDetail.DealSize,
			RemainingAmount: orderDetail.Size - orderDetail.DealSize,
			Amount:          orderDetail.Size,
			Price:           orderDetail.Price,
			Date:            orderDetail.CreatedAt.Time()}, nil
	case asset.Spot, asset.Margin:
		orderDetail, err := ku.GetOrderByID(ctx, orderID)
		if err != nil {
			return nil, err
		}
		nPair, err := currency.NewPairFromString(orderDetail.Symbol)
		if err != nil {
			return nil, err
		}
		oType, err := order.StringToOrderType(orderDetail.Type)
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return nil, err
		}
		if !pair.IsEmpty() && !nPair.Equal(pair) {
			return nil, fmt.Errorf("order with id %s and currency Pair %v does not exist", orderID, pair)
		}
		return &order.Detail{
			Exchange:        ku.Name,
			OrderID:         orderDetail.ID,
			Pair:            pair,
			Type:            oType,
			Side:            side,
			Fee:             orderDetail.Fee,
			AssetType:       assetType,
			ExecutedAmount:  orderDetail.DealSize,
			RemainingAmount: orderDetail.Size - orderDetail.DealSize,
			Amount:          orderDetail.Size,
			Price:           orderDetail.Price,
			Date:            orderDetail.CreatedAt.Time(),
		}, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (ku *Kucoin) GetDepositAddress(ctx context.Context, c currency.Code, _, _ string) (*deposit.Address, error) {
	ad, err := ku.GetDepositAddressesV2(ctx, c.Upper().String())
	if err != nil {
		fad, err := ku.GetFuturesDepositAddress(ctx, c.String())
		if err != nil {
			return nil, err
		}
		return &deposit.Address{
			Address: fad.Address,
			Chain:   fad.Chain,
			Tag:     fad.Memo,
		}, nil
	}
	if len(ad) > 1 {
		return nil, errMultipleDepositAddress
	} else if len(ad) == 0 {
		return nil, errNoDepositAddress
	}
	return &deposit.Address{
		Address: ad[0].Address,
		Chain:   ad[0].Chain,
		Tag:     ad[0].Memo,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
// The endpoint was deprecated for futures, please transfer assets from the FUTURES account to the MAIN account first,
// and then withdraw from the MAIN account
func (ku *Kucoin) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawalID, err := ku.ApplyWithdrawal(ctx, withdrawRequest.Currency.String(), withdrawRequest.Crypto.Address, withdrawRequest.Crypto.AddressTag, withdrawRequest.Description, withdrawRequest.Crypto.Chain, "INTERNAL", false, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: withdrawalID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (ku *Kucoin) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (ku *Kucoin) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

func orderTypeToString(oType order.Type) (string, error) {
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

// GetActiveOrders retrieves any orders that are active/open
func (ku *Kucoin) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if getOrdersRequest == nil {
		return nil, common.ErrNilPointer
	}
	err := ku.CurrencyPairs.IsAssetEnabled(getOrdersRequest.AssetType)
	if err != nil {
		return nil, err
	}
	if getOrdersRequest.Validate() != nil {
		return nil, err
	}
	format, err := ku.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	pair := ""
	orders := []order.Detail{}
	switch getOrdersRequest.AssetType {
	case asset.Futures:
		if len(getOrdersRequest.Pairs) == 1 {
			pair = format.Format(getOrdersRequest.Pairs[0])
		}
		sideString, err := ku.orderSideString(getOrdersRequest.Side)
		if err != nil {
			return nil, err
		}
		oType, err := orderTypeToString(getOrdersRequest.Type)
		if err != nil {
			return nil, err
		}
		futuresOrders, err := ku.GetFuturesOrders(ctx, "active", pair, sideString, oType, getOrdersRequest.StartTime, getOrdersRequest.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range futuresOrders.Items {
			if !futuresOrders.Items[x].IsActive {
				continue
			}
			var dPair currency.Pair
			var isEnabled bool
			dPair, isEnabled, err = ku.MatchSymbolCheckEnabled(futuresOrders.Items[x].Symbol, getOrdersRequest.AssetType, true)
			if err != nil {
				return nil, err
			}
			if !isEnabled {
				continue
			}
			for i := range getOrdersRequest.Pairs {
				if !getOrdersRequest.Pairs[i].Equal(dPair) {
					continue
				}
				side, err := order.StringToOrderSide(futuresOrders.Items[x].Side)
				if err != nil {
					return nil, err
				}
				oType, err := order.StringToOrderType(futuresOrders.Items[x].OrderType)
				if err != nil {
					return nil, fmt.Errorf("asset type: %v order type: %v err: %w", getOrdersRequest.AssetType, getOrdersRequest.Type, err)
				}
				orders = append(orders, order.Detail{
					OrderID:         futuresOrders.Items[x].ID,
					Amount:          futuresOrders.Items[x].Size,
					RemainingAmount: futuresOrders.Items[x].Size - futuresOrders.Items[x].FilledSize,
					ExecutedAmount:  futuresOrders.Items[x].FilledSize,
					Exchange:        ku.Name,
					Date:            futuresOrders.Items[x].CreatedAt.Time(),
					LastUpdated:     futuresOrders.Items[x].UpdatedAt.Time(),
					Price:           futuresOrders.Items[x].Price,
					Side:            side,
					Type:            oType,
					Pair:            dPair,
				})
			}
		}
	case asset.Spot, asset.Margin:
		if len(getOrdersRequest.Pairs) == 1 {
			pair = format.Format(getOrdersRequest.Pairs[0])
		}
		sideString, err := ku.orderSideString(getOrdersRequest.Side)
		if err != nil {
			return nil, err
		}
		oType, err := ku.orderTypeToString(getOrdersRequest.Type)
		if err != nil {
			return nil, fmt.Errorf("asset type: %v order type: %v err: %w", getOrdersRequest.AssetType, getOrdersRequest.Type, err)
		}
		spotOrders, err := ku.ListOrders(ctx, "active", pair, sideString, oType, "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
		if err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		for x := range spotOrders.Items {
			if !spotOrders.Items[x].IsActive {
				continue
			}
			var dPair currency.Pair
			var isEnabled bool
			dPair, isEnabled, err = ku.MatchSymbolCheckEnabled(spotOrders.Items[x].Symbol, getOrdersRequest.AssetType, true)
			if err != nil {
				return nil, err
			}
			if !isEnabled {
				continue
			}
			if len(getOrdersRequest.Pairs) > 0 && !getOrdersRequest.Pairs.Contains(dPair, true) {
				continue
			}
			side, err := order.StringToOrderSide(spotOrders.Items[x].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(spotOrders.Items[x].TradeType)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				OrderID:         spotOrders.Items[x].ID,
				Amount:          spotOrders.Items[x].Size,
				RemainingAmount: spotOrders.Items[x].Size - spotOrders.Items[x].DealSize,
				ExecutedAmount:  spotOrders.Items[x].DealSize,
				Exchange:        ku.Name,
				Date:            spotOrders.Items[x].CreatedAt.Time(),
				Price:           spotOrders.Items[x].Price,
				Side:            side,
				Type:            oType,
				Pair:            dPair,
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (ku *Kucoin) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if getOrdersRequest == nil {
		return nil, common.ErrNilPointer
	}
	err := ku.CurrencyPairs.IsAssetEnabled(getOrdersRequest.AssetType)
	if err != nil {
		return nil, err
	}
	if getOrdersRequest.Validate() != nil {
		return nil, err
	}
	var sideString string
	sideString, err = ku.orderSideString(getOrdersRequest.Side)
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	var orderSide order.Side
	var orderStatus order.Status
	var oType order.Type
	var pair currency.Pair
	switch getOrdersRequest.AssetType {
	case asset.Futures:
		var futuresOrders *FutureOrdersResponse
		var newOrders *FutureOrdersResponse
		if len(getOrdersRequest.Pairs) == 0 {
			futuresOrders, err = ku.GetFuturesOrders(ctx, "", "", sideString, getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
			if err != nil {
				return nil, err
			}
		} else {
			for x := range getOrdersRequest.Pairs {
				getOrdersRequest.Pairs[x], err = ku.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
				if err != nil {
					return nil, err
				}
				newOrders, err = ku.GetFuturesOrders(ctx, "", getOrdersRequest.Pairs[x].String(), sideString, getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
				if err != nil {
					return nil, fmt.Errorf("%w while fetching for symbol %s", err, getOrdersRequest.Pairs[x].String())
				}
				if futuresOrders == nil {
					futuresOrders = newOrders
				} else {
					futuresOrders.Items = append(futuresOrders.Items, newOrders.Items...)
				}
			}
		}
		orders = make(order.FilteredOrders, 0, len(futuresOrders.Items))
		for i := range orders {
			orderSide, err = order.StringToOrderSide(futuresOrders.Items[i].Side)
			if err != nil {
				return nil, err
			}
			var isEnabled bool
			pair, isEnabled, err = ku.MatchSymbolCheckEnabled(futuresOrders.Items[i].Symbol, getOrdersRequest.AssetType, true)
			if err != nil {
				return nil, err
			}
			if !isEnabled {
				continue
			}
			oType, err = order.StringToOrderType(futuresOrders.Items[i].OrderType)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ku.Name, err)
			}
			orders = append(orders, order.Detail{
				Price:           futuresOrders.Items[i].Price,
				Amount:          futuresOrders.Items[i].Size,
				ExecutedAmount:  futuresOrders.Items[i].DealSize,
				RemainingAmount: futuresOrders.Items[i].Size - futuresOrders.Items[i].DealSize,
				Date:            futuresOrders.Items[i].CreatedAt.Time(),
				Exchange:        ku.Name,
				OrderID:         futuresOrders.Items[i].ID,
				Side:            orderSide,
				Status:          orderStatus,
				Type:            oType,
				Pair:            pair,
			})
			orders[i].InferCostsAndTimes()
		}
	case asset.Spot, asset.Margin:
		var responseOrders *OrdersListResponse
		var newOrders *OrdersListResponse
		if len(getOrdersRequest.Pairs) == 0 {
			responseOrders, err = ku.ListOrders(ctx, "", "", sideString, getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
			if err != nil {
				return nil, err
			}
		} else {
			for x := range getOrdersRequest.Pairs {
				newOrders, err = ku.ListOrders(ctx, "", getOrdersRequest.Pairs[x].String(), sideString, getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
				if err != nil {
					return nil, fmt.Errorf("%w while fetching for symbol %s", err, getOrdersRequest.Pairs[x].String())
				}
				if responseOrders == nil {
					responseOrders = newOrders
				} else {
					responseOrders.Items = append(responseOrders.Items, newOrders.Items...)
				}
			}
		}
		orders = make([]order.Detail, len(responseOrders.Items))
		for i := range orders {
			orderSide, err = order.StringToOrderSide(responseOrders.Items[i].Side)
			if err != nil {
				return nil, err
			}
			var orderStatus order.Status
			pair, err = currency.NewPairFromString(responseOrders.Items[i].Symbol)
			if err != nil {
				return nil, err
			}
			var oType order.Type
			oType, err = order.StringToOrderType(responseOrders.Items[i].Type)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ku.Name, err)
			}
			orders[i] = order.Detail{
				Price:           responseOrders.Items[i].Price,
				Amount:          responseOrders.Items[i].Size,
				ExecutedAmount:  responseOrders.Items[i].DealSize,
				RemainingAmount: responseOrders.Items[i].Size - responseOrders.Items[i].DealSize,
				Date:            responseOrders.Items[i].CreatedAt.Time(),
				Exchange:        ku.Name,
				OrderID:         responseOrders.Items[i].ID,
				Side:            orderSide,
				Status:          orderStatus,
				Type:            oType,
				Pair:            pair,
			}
			orders[i].InferCostsAndTimes()
		}
	}
	return getOrdersRequest.Filter(ku.Name, orders), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (ku *Kucoin) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !ku.AreCredentialsValid(ctx) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	if feeBuilder.Pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyWithdrawalFee,
		exchange.CryptocurrencyTradeFee:
		fee, err := ku.GetTradingFee(ctx, currency.Pairs{feeBuilder.Pair})
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			return feeBuilder.Amount * fee[0].MakerFeeRate, nil
		}
		return feeBuilder.Amount * fee[0].TakerFeeRate, nil
	case exchange.OfflineTradeFee:
		return feeBuilder.Amount * 0.001, nil
	case exchange.CryptocurrencyDepositFee:
		return 0, nil
	default:
		if !feeBuilder.FiatCurrency.IsEmpty() {
			fee, err := ku.GetBasicFee(ctx, "1")
			if err != nil {
				return 0, err
			}
			if feeBuilder.IsMaker {
				return feeBuilder.Amount * fee.MakerFeeRate, nil
			}
			return feeBuilder.Amount * fee.TakerFeeRate, nil
		}
		return 0, fmt.Errorf("can't construct fee")
	}
}

// ValidateCredentials validates current credentials used for wrapper
func (ku *Kucoin) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	err := ku.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return err
	}
	_, err = ku.UpdateAccountInfo(ctx, assetType)
	return ku.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (ku *Kucoin) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := ku.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	var timeseries []kline.Candle
	switch a {
	case asset.Futures:
		var candles []FuturesKline
		candles, err := ku.GetFuturesKline(ctx, int64(interval.Duration().Minutes()), req.RequestFormatted.String(), req.Start, req.End)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			timeseries = append(
				timeseries, kline.Candle{
					Time:   candles[x].StartTime,
					Open:   candles[x].Open,
					High:   candles[x].High,
					Low:    candles[x].Low,
					Close:  candles[x].Close,
					Volume: candles[x].Volume,
				})
		}
	case asset.Spot, asset.Margin:
		intervalString, err := ku.intervalToString(interval)
		if err != nil {
			return nil, err
		}
		var candles []Kline
		candles, err = ku.GetKlines(ctx, req.RequestFormatted.String(), intervalString, req.Start, req.End)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			timeseries = append(
				timeseries, kline.Candle{
					Time:   candles[x].StartTime,
					Open:   candles[x].Open,
					High:   candles[x].High,
					Low:    candles[x].Low,
					Close:  candles[x].Close,
					Volume: candles[x].Volume,
				})
		}
	}
	return req.ProcessResponse(timeseries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (ku *Kucoin) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := ku.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var timeSeries []kline.Candle
	for x := range req.RangeHolder.Ranges {
		switch a {
		case asset.Futures:
			var candles []FuturesKline
			candles, err = ku.GetFuturesKline(ctx, int64(interval.Duration().Minutes()), req.RequestFormatted.String(), req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for x := range candles {
				timeSeries = append(
					timeSeries, kline.Candle{
						Time:   candles[x].StartTime,
						Open:   candles[x].Open,
						High:   candles[x].High,
						Low:    candles[x].Low,
						Close:  candles[x].Close,
						Volume: candles[x].Volume,
					})
			}
		case asset.Spot, asset.Margin:
			var intervalString string
			intervalString, err = ku.intervalToString(interval)
			if err != nil {
				return nil, err
			}
			var candles []Kline
			candles, err = ku.GetKlines(ctx, req.RequestFormatted.String(), intervalString, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for x := range candles {
				timeSeries = append(
					timeSeries, kline.Candle{
						Time:   candles[x].StartTime,
						Open:   candles[x].Open,
						High:   candles[x].High,
						Low:    candles[x].Low,
						Close:  candles[x].Close,
						Volume: candles[x].Volume,
					})
			}
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetServerTime returns the current exchange server time.
func (ku *Kucoin) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	switch a {
	case asset.Spot, asset.Margin:
		return ku.GetCurrentServerTime(ctx)
	case asset.Futures:
		return ku.GetFuturesServerTime(ctx)
	default:
		return time.Time{}, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (ku *Kucoin) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencyDetail, err := ku.GetCurrencyDetail(ctx, cryptocurrency.String(), "")
	if err != nil {
		return nil, err
	}
	chains := make([]string, 0, len(currencyDetail.Chains))
	for x := range currencyDetail.Chains {
		chains = append(chains, currencyDetail.Chains[x].Name)
	}
	return chains, nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (ku *Kucoin) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := ku.UpdateAccountInfo(ctx, assetType)
	return ku.CheckTransientError(err)
}

// GetFuturesContractDetails returns details about futures contracts
func (ku *Kucoin) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !ku.SupportsAsset(item) || item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}

	contracts, err := ku.GetFuturesOpenContracts(ctx)
	if err != nil {
		return nil, err
	}

	resp := make([]futures.Contract, len(contracts))
	for i := range contracts {
		var cp, underlying currency.Pair
		underlying, err = currency.NewPairFromStrings(contracts[i].BaseCurrency, contracts[i].QuoteCurrency)
		if err != nil {
			return nil, err
		}
		cp, err = currency.NewPairFromStrings(contracts[i].BaseCurrency, contracts[i].Symbol[len(contracts[i].BaseCurrency):])
		if err != nil {
			return nil, err
		}
		settleCurr := currency.NewCode(contracts[i].SettleCurrency)
		var ct futures.ContractType
		if contracts[i].ContractType == "FFWCSX" {
			ct = futures.Perpetual
		} else {
			ct = futures.Quarterly
		}
		contractSettlementType := futures.Linear
		if contracts[i].IsInverse {
			contractSettlementType = futures.Inverse
		}
		var fri time.Duration
		if len(ku.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
			// can infer funding rate interval from the only funding rate frequency defined
			for k := range ku.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
				fri = k.Duration()
			}
		}
		timeOfCurrentFundingRate := time.Now().Add((time.Duration(contracts[i].NextFundingRateTime) * time.Millisecond) - fri).Truncate(time.Hour).UTC()
		resp[i] = futures.Contract{
			Exchange:             ku.Name,
			Name:                 cp,
			Underlying:           underlying,
			SettlementCurrencies: currency.Currencies{settleCurr},
			MarginCurrency:       settleCurr,
			Asset:                item,
			StartDate:            contracts[i].FirstOpenDate.Time(),
			EndDate:              contracts[i].ExpireDate.Time(),
			IsActive:             !strings.EqualFold(contracts[i].Status, "closed"),
			Status:               contracts[i].Status,
			Multiplier:           contracts[i].Multiplier,
			MaxLeverage:          contracts[i].MaxLeverage,
			SettlementType:       contractSettlementType,
			LatestRate: fundingrate.Rate{
				Rate: decimal.NewFromFloat(contracts[i].FundingFeeRate),
				Time: timeOfCurrentFundingRate, // kucoin pays every 8 hours
			},
			Type: ct,
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (ku *Kucoin) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	var fri time.Duration
	if len(ku.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
		// can infer funding rate interval from the only funding rate frequency defined
		for k := range ku.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
			fri = k.Duration()
		}
	}
	if r.Pair.IsEmpty() {
		contracts, err := ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		if r.IncludePredictedRate {
			log.Warnf(log.ExchangeSys, "%s predicted rate for all currencies requires an additional %v requests", ku.Name, len(contracts))
		}
		timeChecked := time.Now()
		resp := make([]fundingrate.LatestRateResponse, 0, len(contracts))
		for i := range contracts {
			timeOfNextFundingRate := time.Now().Add(time.Duration(contracts[i].NextFundingRateTime) * time.Millisecond).Truncate(time.Hour).UTC()
			var cp currency.Pair
			cp, err = currency.NewPairFromStrings(contracts[i].BaseCurrency, contracts[i].Symbol[len(contracts[i].BaseCurrency):])
			if err != nil {
				return nil, err
			}
			var isPerp bool
			isPerp, err = ku.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}

			rate := fundingrate.LatestRateResponse{
				Exchange: ku.Name,
				Asset:    r.Asset,
				Pair:     cp,
				LatestRate: fundingrate.Rate{
					Time: timeOfNextFundingRate.Add(-fri),
					Rate: decimal.NewFromFloat(contracts[i].FundingFeeRate),
				},
				TimeOfNextRate: timeOfNextFundingRate,
				TimeChecked:    timeChecked,
			}
			if r.IncludePredictedRate {
				var fr *FuturesFundingRate
				fr, err = ku.GetFuturesCurrentFundingRate(ctx, contracts[i].Symbol)
				if err != nil {
					return nil, err
				}
				rate.PredictedUpcomingRate = fundingrate.Rate{
					Time: timeOfNextFundingRate,
					Rate: decimal.NewFromFloat(fr.PredictedValue),
				}
			}
			resp = append(resp, rate)
		}
		return resp, nil
	}
	resp := make([]fundingrate.LatestRateResponse, 1)
	is, err := ku.IsPerpetualFutureCurrency(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	if !is {
		return nil, fmt.Errorf("%w %s %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	fPair, err := ku.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	var fr *FuturesFundingRate
	fr, err = ku.GetFuturesCurrentFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	rate := fundingrate.LatestRateResponse{
		Exchange: ku.Name,
		Asset:    r.Asset,
		Pair:     r.Pair,
		LatestRate: fundingrate.Rate{
			Time: fr.TimePoint.Time(),
			Rate: decimal.NewFromFloat(fr.Value),
		},
		TimeOfNextRate: fr.TimePoint.Time().Add(fri).Truncate(time.Hour).UTC(),
		TimeChecked:    time.Now(),
	}
	if r.IncludePredictedRate {
		rate.PredictedUpcomingRate = fundingrate.Rate{
			Time: rate.TimeOfNextRate,
			Rate: decimal.NewFromFloat(fr.PredictedValue),
		}
	}
	resp[0] = rate
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (ku *Kucoin) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	return a == asset.Futures && (cp.Quote.Equal(currency.USDTM) || cp.Quote.Equal(currency.USDM)), nil
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (ku *Kucoin) GetHistoricalFundingRates(_ context.Context, _ *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (ku *Kucoin) GetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ order.Side) (float64, error) {
	return -1, fmt.Errorf("%w leverage is set during order placement, view orders to view leverage", common.ErrFunctionNotSupported)
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (ku *Kucoin) SetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ float64, _ order.Side) error {
	return fmt.Errorf("%w leverage is set during order placement", common.ErrFunctionNotSupported)
}

// SetMarginType sets the default margin type for when opening a new position
func (ku *Kucoin) SetMarginType(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// SetCollateralMode sets the collateral type for your account
func (ku *Kucoin) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// GetCollateralMode returns the collateral type for your account
func (ku *Kucoin) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return collateral.UnknownMode, fmt.Errorf("%w only via website", common.ErrFunctionNotSupported)
}

// ChangePositionMargin will modify a position/currencies margin parameters
func (ku *Kucoin) ChangePositionMargin(ctx context.Context, r *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
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
	fPair, err := ku.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}

	resp, err := ku.AddMargin(ctx, fPair.String(), fmt.Sprintf("%s%v%v", r.Pair, r.NewAllocatedMargin, time.Now().Unix()), r.NewAllocatedMargin)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%s - %s", ku.Name, "no response received")
	}
	return &margin.PositionChangeResponse{
		Exchange:        ku.Name,
		Pair:            r.Pair,
		Asset:           r.Asset,
		AllocatedMargin: resp.PosMargin,
		MarginType:      r.MarginType,
	}, nil
}

// GetFuturesPositionSummary returns position summary details for an active position
func (ku *Kucoin) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair, err := ku.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	pos, err := ku.GetFuturesPosition(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	marginType := margin.Isolated
	if pos.CrossMode {
		marginType = margin.Multi
	}
	contracts, err := ku.GetFuturesContractDetails(ctx, r.Asset)
	if err != nil {
		return nil, err
	}
	var multiplier, contractSize float64
	var settlementType futures.ContractSettlementType
	for i := range contracts {
		if !contracts[i].Name.Equal(fPair) {
			continue
		}
		multiplier = contracts[i].Multiplier
		contractSize = multiplier * pos.CurrentQty
		settlementType = contracts[i].SettlementType
	}

	ao, err := ku.GetFuturesAccountOverview(ctx, fPair.String())
	if err != nil {
		return nil, err
	}

	return &futures.PositionSummary{
		Pair:                         r.Pair,
		Asset:                        r.Asset,
		MarginType:                   marginType,
		CollateralMode:               collateral.MultiMode,
		Currency:                     currency.NewCode(pos.SettleCurrency),
		StartDate:                    pos.OpeningTimestamp.Time(),
		AvailableEquity:              decimal.NewFromFloat(ao.AccountEquity),
		MarginBalance:                decimal.NewFromFloat(ao.MarginBalance),
		NotionalSize:                 decimal.NewFromFloat(pos.MarkValue),
		Leverage:                     decimal.NewFromFloat(pos.RealLeverage),
		MaintenanceMarginRequirement: decimal.NewFromFloat(pos.MaintMarginReq),
		InitialMarginRequirement:     decimal.NewFromFloat(pos.PosInit),
		EstimatedLiquidationPrice:    decimal.NewFromFloat(pos.LiquidationPrice),
		CollateralUsed:               decimal.NewFromFloat(pos.PosCost),
		MarkPrice:                    decimal.NewFromFloat(pos.MarkPrice),
		CurrentSize:                  decimal.NewFromFloat(pos.CurrentQty),
		ContractSize:                 decimal.NewFromFloat(contractSize),
		ContractMultiplier:           decimal.NewFromFloat(multiplier),
		ContractSettlementType:       settlementType,
		AverageOpenPrice:             decimal.NewFromFloat(pos.AvgEntryPrice),
		UnrealisedPNL:                decimal.NewFromFloat(pos.UnrealisedPnl),
		RealisedPNL:                  decimal.NewFromFloat(pos.RealisedPnl),
		MaintenanceMarginFraction:    decimal.NewFromFloat(pos.MaintMarginReq),
		FreeCollateral:               decimal.NewFromFloat(ao.AvailableBalance),
		TotalCollateral:              decimal.NewFromFloat(ao.AccountEquity),
		FrozenBalance:                decimal.NewFromFloat(ao.FrozenFunds),
	}, nil
}

// GetFuturesPositionOrders returns the orders for futures positions
func (ku *Kucoin) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairEmpty
	}
	err := common.StartEndTimeCheck(r.StartDate, r.EndDate)
	if err != nil {
		return nil, err
	}
	if !r.EndDate.IsZero() && r.EndDate.Sub(r.StartDate) > ku.Features.Supports.MaximumOrderHistory {
		if r.RespectOrderHistoryLimits {
			r.StartDate = time.Now().Add(-ku.Features.Supports.MaximumOrderHistory)
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, time.Now().Add(-ku.Features.Supports.MaximumOrderHistory))
		}
	}
	contracts, err := ku.GetFuturesContractDetails(ctx, r.Asset)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for x := range r.Pairs {
		var multiplier float64
		fPair, err := ku.FormatExchangeCurrency(r.Pairs[x], r.Asset)
		if err != nil {
			return nil, err
		}
		for i := range contracts {
			if !contracts[i].Name.Equal(fPair) {
				continue
			}
			multiplier = contracts[i].Multiplier
		}

		positionOrders, err := ku.GetFuturesOrders(ctx, "", fPair.String(), "", "", r.StartDate, r.EndDate)
		if err != nil {
			return nil, err
		}
		resp[x].Orders = make([]order.Detail, len(positionOrders.Items))
		for y := range positionOrders.Items {
			side, err := order.StringToOrderSide(positionOrders.Items[y].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(positionOrders.Items[y].OrderType)
			if err != nil {
				return nil, fmt.Errorf("asset type: %v err: %w", r.Asset, err)
			}
			oStatus, err := order.StringToOrderStatus(positionOrders.Items[y].Status)
			if err != nil {
				return nil, fmt.Errorf("asset type: %v err: %w", r.Asset, err)
			}
			resp[x].Orders[y] = order.Detail{
				Leverage:        positionOrders.Items[y].Leverage,
				Price:           positionOrders.Items[y].Price,
				Amount:          positionOrders.Items[y].Size * multiplier,
				ContractAmount:  positionOrders.Items[y].Size,
				ExecutedAmount:  positionOrders.Items[y].FilledSize,
				RemainingAmount: positionOrders.Items[y].Size - positionOrders.Items[y].FilledSize,
				CostAsset:       currency.NewCode(positionOrders.Items[y].SettleCurrency),
				Exchange:        ku.Name,
				OrderID:         positionOrders.Items[y].ID,
				ClientOrderID:   positionOrders.Items[y].ClientOid,
				Type:            oType,
				Side:            side,
				Status:          oStatus,
				AssetType:       asset.Futures,
				Date:            positionOrders.Items[y].CreatedAt.Time(),
				CloseTime:       positionOrders.Items[y].EndAt.Time(),
				LastUpdated:     positionOrders.Items[y].UpdatedAt.Time(),
				Pair:            fPair,
			}
		}
	}
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (ku *Kucoin) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !ku.SupportsAsset(a) {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}

	var limits []order.MinMaxLevel
	switch a {
	case asset.Spot, asset.Margin:
		symbols, err := ku.GetSymbols(ctx, "")
		if err != nil {
			return err
		}
		limits = make([]order.MinMaxLevel, 0, len(symbols))
		for x := range symbols {
			if a == asset.Margin && !symbols[x].IsMarginEnabled {
				continue
			}
			pair, enabled, err := ku.MatchSymbolCheckEnabled(symbols[x].Symbol, a, true)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return err
			}
			if !enabled {
				continue
			}
			limits = append(limits, order.MinMaxLevel{
				Pair:                    pair,
				Asset:                   a,
				AmountStepIncrementSize: symbols[x].BaseIncrement,
				QuoteStepIncrementSize:  symbols[x].QuoteIncrement,
				PriceStepIncrementSize:  symbols[x].PriceIncrement,
				MinimumBaseAmount:       symbols[x].BaseMinSize,
				MaximumBaseAmount:       symbols[x].BaseMaxSize,
				MinimumQuoteAmount:      symbols[x].QuoteMinSize,
				MaximumQuoteAmount:      symbols[x].QuoteMaxSize,
			})
		}
	case asset.Futures:
		contract, err := ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return err
		}
		limits = make([]order.MinMaxLevel, 0, len(contract))
		for x := range contract {
			pair, enabled, err := ku.MatchSymbolCheckEnabled(contract[x].Symbol, a, false)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return err
			}
			if !enabled {
				continue
			}
			limits = append(limits, order.MinMaxLevel{
				Pair:                    pair,
				Asset:                   a,
				AmountStepIncrementSize: contract[x].LotSize,
				QuoteStepIncrementSize:  contract[x].TickSize,
				MaximumBaseAmount:       contract[x].MaxOrderQty,
				MaximumQuoteAmount:      contract[x].MaxPrice,
			})
		}
	}

	return ku.LoadLimits(limits)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (ku *Kucoin) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	ticks, err := ku.GetCachedOpenInterest(ctx, k...)
	if err == nil && len(ticks) > 0 {
		return ticks, nil
	}

	if len(k) == 0 || len(k) > 1 {
		var contracts []Contract
		contracts, err = ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.OpenInterest, 0, len(contracts))
		for i := range contracts {
			var symbol currency.Pair
			var enabled bool
			symbol, enabled, err = ku.MatchSymbolCheckEnabled(contracts[i].Symbol, asset.Futures, true)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return nil, err
			}
			if !enabled {
				continue
			}
			var appendData bool
			for j := range k {
				if k[j].Pair().Equal(symbol) {
					appendData = true
					break
				}
			}
			if len(k) > 0 && !appendData {
				continue
			}
			resp = append(resp, futures.OpenInterest{
				Key: key.ExchangePairAsset{
					Exchange: ku.Name,
					Base:     symbol.Base.Item,
					Quote:    symbol.Quote.Item,
					Asset:    asset.Futures,
				},
				OpenInterest: contracts[i].OpenInterest.Float64(),
			})
		}
		return resp, nil
	}

	resp := make([]futures.OpenInterest, 1)
	p, isEnabled, err := ku.MatchSymbolCheckEnabled(k[0].Pair().String(), k[0].Asset, false)
	if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
		return nil, err
	}
	if !isEnabled {
		return nil, fmt.Errorf("%v %w", p, currency.ErrPairNotEnabled)
	}
	symbolStr, err := ku.FormatSymbol(k[0].Pair(), k[0].Asset)
	if err != nil {
		return nil, err
	}
	instrument, err := ku.GetFuturesContract(ctx, symbolStr)
	if err != nil {
		return nil, err
	}
	resp[0] = futures.OpenInterest{
		Key: key.ExchangePairAsset{
			Exchange: ku.Name,
			Base:     k[0].Base,
			Quote:    k[0].Quote,
			Asset:    k[0].Asset,
		},
		OpenInterest: instrument.OpenInterest.Float64(),
	}
	return resp, nil
}
