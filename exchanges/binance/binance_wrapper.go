package binance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var defaultAssetPairStores = map[asset.Item]currency.PairStore{
	asset.Spot: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true},
	},
	asset.Margin: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true},
	},
	asset.CoinMarginedFutures: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	},
	asset.USDTMarginedFutures: {
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	},
}

// SetDefaults sets the basic defaults for Binance
func (e *Exchange) SetDefaults() {
	e.Name = "Binance"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	for a, ps := range defaultAssetPairStores {
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	for _, a := range []asset.Item{asset.Margin, asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		if err := e.DisableAssetWebsocketSupport(a); err != nil {
			log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
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

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              spotAPIURL,
		exchange.RestSpotSupplementary: apiURL,
		exchange.RestUSDTMargined:      ufuturesAPIURL,
		exchange.RestCoinMargined:      cfuturesAPIURL,
		exchange.EdgeCase1:             "https://www.binance.com",
		exchange.WebsocketSpot:         binanceDefaultWebsocketURL,
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
	ePoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            binanceDefaultWebsocketURL,
		RunningURL:            ePoint,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
		TradeFeed: e.Features.Enabled.TradeFeed,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            request.NewWeightedRateLimitByDuration(250 * time.Millisecond),
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	tradingStatus := "TRADING"
	var pairs []currency.Pair
	switch a {
	case asset.Spot, asset.Margin:
		info, err := e.GetExchangeInfo(ctx)
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
		cInfo, err := e.FuturesExchangeInfo(ctx)
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
		uInfo, err := e.UExchangeInfo(ctx)
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
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assetTypes := e.GetAssetTypes(false)
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
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot, asset.Margin:
		tick, err := e.GetTickers(ctx)
		if err != nil {
			return err
		}

		pairs, err := e.GetEnabledPairs(a)
		if err != nil {
			return err
		}

		for i := range pairs {
			for y := range tick {
				pairFmt, err := e.FormatExchangeCurrency(pairs[i], a)
				if err != nil {
					return err
				}

				if tick[y].Symbol != pairFmt.String() {
					continue
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
					Pair:         pairFmt,
					ExchangeName: e.Name,
					AssetType:    a,
				})
				if err != nil {
					return err
				}
			}
		}
	case asset.USDTMarginedFutures:
		tick, err := e.U24HTickerPriceChangeStats(ctx, currency.EMPTYPAIR)
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
				ExchangeName: e.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.CoinMarginedFutures:
		tick, err := e.GetFuturesSwapTickerChangeStats(ctx, currency.EMPTYPAIR, "")
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
				ExchangeName: e.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch a {
	case asset.Spot, asset.Margin:
		tick, err := e.GetPriceChangeStats(ctx, p)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick.LastPrice.Float64(),
			High:         tick.HighPrice.Float64(),
			Low:          tick.LowPrice.Float64(),
			Bid:          tick.BidPrice.Float64(),
			Ask:          tick.AskPrice.Float64(),
			Volume:       tick.Volume.Float64(),
			QuoteVolume:  tick.QuoteVolume.Float64(),
			Open:         tick.OpenPrice.Float64(),
			Close:        tick.PrevClosePrice.Float64(),
			Pair:         p,
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.USDTMarginedFutures:
		tick, err := e.U24HTickerPriceChangeStats(ctx, p)
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
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.CoinMarginedFutures:
		tick, err := e.GetFuturesSwapTickerChangeStats(ctx, p, "")
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
			ExchangeName: e.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}

	var orderbookNew *OrderBookResponse
	var err error
	switch a {
	case asset.Spot, asset.Margin:
		orderbookNew, err = e.GetOrderBook(ctx, p, 1000)
	case asset.USDTMarginedFutures:
		orderbookNew, err = e.UFuturesOrderbook(ctx, p, 1000)
	case asset.CoinMarginedFutures:
		orderbookNew, err = e.GetFuturesOrderbook(ctx, p, 1000)
	default:
		return nil, fmt.Errorf("[%s] %w", a, asset.ErrNotSupported)
	}
	if err != nil {
		return nil, err
	}

	ob := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              orderbookNew.Bids.Levels(),
		Asks:              orderbookNew.Asks.Levels(),
	}

	if err := ob.Process(); err != nil {
		return nil, err
	}

	return orderbook.Get(e.Name, p, a)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (subAccts accounts.SubAccounts, err error) {
	switch assetType {
	case asset.Spot:
		creds, err := e.GetCredentials(ctx)
		if err != nil {
			return nil, err
		}
		if creds.SubAccount != "" {
			// TODO: implement sub-account endpoints
			return nil, common.ErrNotYetImplemented
		}
		resp, err := e.GetAccount(ctx)
		if err != nil {
			return nil, err
		}
		subAccts = accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
		for i := range resp.Balances {
			free := resp.Balances[i].Free.InexactFloat64()
			locked := resp.Balances[i].Locked.InexactFloat64()
			subAccts[0].Balances.Set(resp.Balances[i].Asset, accounts.Balance{
				Total: free + locked,
				Hold:  locked,
				Free:  free,
			})
		}
	case asset.CoinMarginedFutures:
		resp, err := e.GetFuturesAccountInfo(ctx)
		if err != nil {
			return nil, err
		}
		subAccts = accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
		for i := range resp.Assets {
			subAccts[0].Balances.Set(resp.Assets[i].Asset, accounts.Balance{
				Total: resp.Assets[i].WalletBalance,
				Hold:  resp.Assets[i].WalletBalance - resp.Assets[i].AvailableBalance,
				Free:  resp.Assets[i].AvailableBalance,
			})
		}
	case asset.USDTMarginedFutures:
		resp, err := e.UAccountBalanceV2(ctx)
		if err != nil {
			return nil, err
		}
		subAccts = make(accounts.SubAccounts, 0, len(resp))
		for i := range resp {
			a := accounts.NewSubAccount(assetType, resp[i].AccountAlias)
			a.Balances.Set(resp[i].Asset, accounts.Balance{
				Total: resp[i].Balance,
				Hold:  resp[i].Balance - resp[i].AvailableBalance,
				Free:  resp[i].AvailableBalance,
			})
			subAccts = subAccts.Merge(a)
		}
	case asset.Margin:
		resp, err := e.GetMarginAccount(ctx)
		if err != nil {
			return nil, err
		}
		subAccts = accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
		for i := range resp.UserAssets {
			subAccts[0].Balances.Set(resp.UserAssets[i].Asset, accounts.Balance{
				Total:                  resp.UserAssets[i].Free + resp.UserAssets[i].Locked,
				Hold:                   resp.UserAssets[i].Locked,
				Free:                   resp.UserAssets[i].Free,
				AvailableWithoutBorrow: resp.UserAssets[i].Free - resp.UserAssets[i].Borrowed,
				Borrowed:               resp.UserAssets[i].Borrowed,
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.WithdrawHistory(ctx, c, "", time.Time{}, time.Time{}, 0, 10000)
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
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	const limit = 1000
	rFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	pFmt := p.Format(rFmt)
	resp := make([]trade.Data, 0, limit)
	switch a {
	case asset.Spot:
		tradeData, err := e.GetMostRecentTrades(ctx,
			RecentTradeRequestParams{pFmt, limit})
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			td := trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Quantity,
				Timestamp:    tradeData[i].Time.Time(),
			}
			if tradeData[i].IsBuyerMaker { // Seller is Taker
				td.Side = order.Sell
			} else { // Buyer is Taker
				td.Side = order.Buy
			}
			resp = append(resp, td)
		}
	case asset.USDTMarginedFutures:
		tradeData, err := e.URecentTrades(ctx, pFmt, "", limit)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			td := trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Qty,
				Timestamp:    tradeData[i].Time.Time(),
			}
			if tradeData[i].IsBuyerMaker { // Seller is Taker
				td.Side = order.Sell
			} else { // Buyer is Taker
				td.Side = order.Buy
			}
			resp = append(resp, td)
		}
	case asset.CoinMarginedFutures:
		tradeData, err := e.GetFuturesPublicTrades(ctx, pFmt, limit)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			td := trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Qty,
				Timestamp:    tradeData[i].Time.Time(),
			}
			if tradeData[i].IsBuyerMaker { // Seller is Taker
				td.Side = order.Sell
			} else { // Buyer is Taker
				td.Side = order.Buy
			}
			resp = append(resp, td)
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
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, from, to time.Time) ([]trade.Data, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	rFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	pFmt := p.Format(rFmt)
	req := AggregatedTradeRequestParams{
		Symbol:    pFmt,
		StartTime: from,
		EndTime:   to,
	}
	trades, err := e.GetAggregatedTrades(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("%w %v", err, pFmt)
	}
	result := make([]trade.Data, len(trades))
	for i := range trades {
		td := trade.Data{
			CurrencyPair: p,
			TID:          strconv.FormatInt(trades[i].ATradeID, 10),
			Amount:       trades[i].Quantity,
			Exchange:     e.Name,
			Price:        trades[i].Price,
			Timestamp:    trades[i].TimeStamp.Time(),
			AssetType:    a,
		}
		if trades[i].IsBuyerMaker { // Seller is Taker
			td.Side = order.Sell
		} else { // Buyer is Taker
			td.Side = order.Buy
		}
		result[i] = td
	}
	return result, nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
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
		timeInForce := order.GoodTillCancel.String()
		var requestParamsOrderType RequestParamsOrderType
		switch s.Type {
		case order.Market:
			timeInForce = ""
			requestParamsOrderType = BinanceRequestParamsOrderMarket
		case order.Limit:
			if s.TimeInForce.Is(order.ImmediateOrCancel) {
				timeInForce = order.ImmediateOrCancel.String()
			}
			requestParamsOrderType = BinanceRequestParamsOrderLimit
		default:
			return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, s.Type)
		}

		orderRequest := NewOrderRequest{
			Symbol:           s.Pair,
			Side:             sideType,
			Price:            s.Price,
			Quantity:         s.Amount,
			TradeType:        requestParamsOrderType,
			TimeInForce:      timeInForce,
			NewClientOrderID: s.ClientOrderID,
		}
		response, err := e.NewOrder(ctx, &orderRequest)
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
				Amount:   response.Fills[i].Qty,
				Fee:      response.Fills[i].Commission,
				FeeAsset: response.Fills[i].CommissionAsset,
			}
		}
	case asset.CoinMarginedFutures:
		var reqSide string
		switch s.Side {
		case order.Buy:
			reqSide = "BUY"
		case order.Sell:
			reqSide = "SELL"
		default:
			return nil, errors.New("invalid side")
		}

		var oType, timeInForce string
		switch s.Type {
		case order.Limit:
			oType = cfuturesLimit
			timeInForce = order.GoodTillCancel.String()
		case order.Market:
			oType = cfuturesMarket
		case order.Stop:
			oType = cfuturesStop
		case order.TakeProfit:
			oType = cfuturesTakeProfit
		case order.StopMarket:
			oType = cfuturesStopMarket
		case order.TakeProfitMarket:
			oType = cfuturesTakeProfitMarket
		case order.TrailingStop:
			oType = cfuturesTrailingStopMarket
		default:
			return nil, errors.New("invalid type, check api docs for updates")
		}

		o, err := e.FuturesNewOrder(
			ctx,
			&FuturesNewOrderRequest{
				Symbol:           s.Pair,
				Side:             reqSide,
				OrderType:        oType,
				TimeInForce:      timeInForce,
				NewClientOrderID: s.ClientOrderID,
				Quantity:         s.Amount,
				Price:            s.Price,
				ReduceOnly:       s.ReduceOnly,
			},
		)
		if err != nil {
			return nil, err
		}
		orderID = strconv.FormatInt(o.OrderID, 10)
	case asset.USDTMarginedFutures:
		var reqSide string
		switch s.Side {
		case order.Buy:
			reqSide = "BUY"
		case order.Sell:
			reqSide = "SELL"
		default:
			return nil, errors.New("invalid side")
		}
		var oType string
		switch s.Type {
		case order.Limit:
			oType = "LIMIT"
		case order.Market:
			oType = "MARKET"
		case order.Stop:
			oType = "STOP"
		case order.TakeProfit:
			oType = "TAKE_PROFIT"
		case order.StopMarket:
			oType = "STOP_MARKET"
		case order.TakeProfitMarket:
			oType = "TAKE_PROFIT_MARKET"
		case order.TrailingStop:
			oType = "TRAILING_STOP_MARKET"
		default:
			return nil, errors.New("invalid type, check api docs for updates")
		}
		o, err := e.UFuturesNewOrder(ctx,
			&UFuturesNewOrderRequest{
				Symbol:           s.Pair,
				Side:             reqSide,
				OrderType:        oType,
				TimeInForce:      "GTC",
				NewClientOrderID: s.ClientOrderID,
				Quantity:         s.Amount,
				Price:            s.Price,
				ReduceOnly:       s.ReduceOnly,
			},
		)
		if err != nil {
			return nil, err
		}
		orderID = strconv.FormatInt(o.OrderID, 10)
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

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot, asset.Margin:
		orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
		if err != nil {
			return err
		}
		_, err = e.CancelExistingOrder(ctx,
			o.Pair,
			orderIDInt,
			o.AccountID)
		if err != nil {
			return err
		}
	case asset.CoinMarginedFutures:
		_, err := e.FuturesCancelOrder(ctx, o.Pair, o.OrderID, "")
		if err != nil {
			return err
		}
	case asset.USDTMarginedFutures:
		_, err := e.UCancelOrder(ctx, o.Pair, o.OrderID, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch req.AssetType {
	case asset.Spot, asset.Margin:
		openOrders, err := e.OpenOrders(ctx, req.Pair)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range openOrders {
			_, err = e.CancelExistingOrder(ctx,
				req.Pair,
				openOrders[i].OrderID,
				"")
			if err != nil {
				cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders[i].OrderID, 10)] = err.Error()
			}
		}
	case asset.CoinMarginedFutures:
		if req.Pair.IsEmpty() {
			enabledPairs, err := e.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				_, err = e.FuturesCancelAllOpenOrders(ctx, enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
			}
		} else {
			_, err := e.FuturesCancelAllOpenOrders(ctx, req.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	case asset.USDTMarginedFutures:
		if req.Pair.IsEmpty() {
			enabledPairs, err := e.GetEnabledPairs(asset.USDTMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				_, err = e.UCancelAllOpenOrders(ctx, enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
			}
		} else {
			_, err := e.UCancelAllOpenOrders(ctx, req.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	var respData order.Detail
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		resp, err := e.QueryOrder(ctx, pair, "", orderIDInt)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(resp.Side)
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(resp.Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
		}
		orderType := order.Limit
		if resp.Type == "MARKET" {
			orderType = order.Market
		}

		return &order.Detail{
			Amount:         resp.OrigQty,
			Exchange:       e.Name,
			OrderID:        strconv.FormatInt(resp.OrderID, 10),
			ClientOrderID:  resp.ClientOrderID,
			Side:           side,
			Type:           orderType,
			Pair:           pair,
			Cost:           resp.CummulativeQuoteQty,
			AssetType:      assetType,
			Status:         status,
			Price:          resp.Price,
			ExecutedAmount: resp.ExecutedQty,
			Date:           resp.Time.Time(),
			LastUpdated:    resp.UpdateTime.Time(),
		}, nil
	case asset.CoinMarginedFutures:
		orderData, err := e.FuturesOpenOrderData(ctx, pair, orderID, "")
		if err != nil {
			return nil, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := e.GetFee(ctx, &feeBuilder)
		if err != nil {
			return nil, err
		}
		orderVars := compatibleOrderVars(orderData.Side, orderData.Status, orderData.OrderType)
		respData.Amount = orderData.OriginalQuantity
		respData.AssetType = assetType
		respData.ClientOrderID = orderData.ClientOrderID
		respData.Exchange = e.Name
		respData.ExecutedAmount = orderData.ExecutedQuantity
		respData.Fee = fee
		respData.OrderID = orderID
		respData.Pair = pair
		respData.Price = orderData.Price
		respData.RemainingAmount = orderData.OriginalQuantity - orderData.ExecutedQuantity
		respData.Side = orderVars.Side
		respData.Status = orderVars.Status
		respData.Type = orderVars.OrderType
		respData.Date = orderData.Time.Time()
		respData.LastUpdated = orderData.UpdateTime.Time()
	case asset.USDTMarginedFutures:
		orderData, err := e.UGetOrderData(ctx, pair, orderID, "")
		if err != nil {
			return nil, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := e.GetFee(ctx, &feeBuilder)
		if err != nil {
			return nil, err
		}
		orderVars := compatibleOrderVars(orderData.Side, orderData.Status, orderData.OrderType)
		respData.Amount = orderData.OriginalQuantity
		respData.AssetType = assetType
		respData.ClientOrderID = orderData.ClientOrderID
		respData.Exchange = e.Name
		respData.ExecutedAmount = orderData.ExecutedQuantity
		respData.Fee = fee
		respData.OrderID = orderID
		respData.Pair = pair
		respData.Price = orderData.Price
		respData.RemainingAmount = orderData.OriginalQuantity - orderData.ExecutedQuantity
		respData.Side = orderVars.Side
		respData.Status = orderVars.Status
		respData.Type = orderVars.OrderType
		respData.Date = orderData.Time.Time()
		respData.LastUpdated = orderData.UpdateTime.Time()
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return &respData, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := e.GetDepositAddressForCurrency(ctx, cryptocurrency.String(), chain)
	if err != nil {
		return nil, err
	}

	return &deposit.Address{
		Address: addr.Address,
		Tag:     addr.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	amountStr := strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64)
	v, err := e.WithdrawCrypto(ctx,
		withdrawRequest.Currency.String(),
		"", // withdrawal order ID
		withdrawRequest.Crypto.Chain,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Description,
		amountStr,
		false)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
			resp, err := e.OpenOrders(ctx, req.Pairs[i])
			if err != nil {
				return nil, err
			}
			for x := range resp {
				var side order.Side
				side, err = order.StringToOrderSide(resp[x].Side)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				var orderType order.Type
				orderType, err = order.StringToOrderType(resp[x].Type)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				orderStatus, err := order.StringToOrderStatus(resp[x].Status)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				orders = append(orders, order.Detail{
					Amount:        resp[x].OrigQty,
					Date:          resp[x].Time.Time(),
					Exchange:      e.Name,
					OrderID:       strconv.FormatInt(resp[x].OrderID, 10),
					ClientOrderID: resp[x].ClientOrderID,
					Side:          side,
					Type:          orderType,
					Price:         resp[x].Price,
					Status:        orderStatus,
					Pair:          req.Pairs[i],
					AssetType:     req.AssetType,
					LastUpdated:   resp[x].UpdateTime.Time(),
				})
			}
		case asset.CoinMarginedFutures:
			openOrders, err := e.GetFuturesAllOpenOrders(ctx, req.Pairs[i], "")
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = openOrders[y].ExecutedQty
				feeBuilder.PurchasePrice = openOrders[y].AvgPrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := e.GetFee(ctx, &feeBuilder)
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
					Exchange:        e.Name,
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
			openOrders, err := e.UAllAccountOpenOrders(ctx, req.Pairs[i])
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = openOrders[y].ExecutedQuantity
				feeBuilder.PurchasePrice = openOrders[y].AveragePrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := e.GetFee(ctx, &feeBuilder)
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
					Exchange:        e.Name,
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
		default:
			return orders, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
		}
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot, asset.Margin:
		for x := range req.Pairs {
			resp, err := e.AllOrders(ctx,
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
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				var orderType order.Type
				orderType, err = order.StringToOrderType(resp[i].Type)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				orderStatus, err := order.StringToOrderStatus(resp[i].Status)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				// New orders are covered in GetOpenOrders
				if orderStatus == order.New {
					continue
				}

				var cost float64
				// For some historical orders cummulativeQuoteQty will be < 0,
				// meaning the data is not available at this time.
				if resp[i].CummulativeQuoteQty > 0 {
					cost = resp[i].CummulativeQuoteQty
				}
				detail := order.Detail{
					Amount:          resp[i].OrigQty,
					ExecutedAmount:  resp[i].ExecutedQty,
					RemainingAmount: resp[i].OrigQty - resp[i].ExecutedQty,
					Cost:            cost,
					CostAsset:       req.Pairs[x].Quote,
					Date:            resp[i].Time.Time(),
					LastUpdated:     resp[i].UpdateTime.Time(),
					Exchange:        e.Name,
					OrderID:         strconv.FormatInt(resp[i].OrderID, 10),
					Side:            side,
					Type:            orderType,
					Price:           resp[i].Price,
					Pair:            req.Pairs[x],
					Status:          orderStatus,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
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
				orderHistory, err = e.GetAllFuturesOrders(ctx,
					req.Pairs[i], currency.EMPTYPAIR, req.StartTime, req.EndTime, 0, 0)
				if err != nil {
					return nil, err
				}
			case req.FromOrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.FromOrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = e.GetAllFuturesOrders(ctx,
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
				fee, err := e.GetFee(ctx, &feeBuilder)
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
					Exchange:        e.Name,
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
				orderHistory, err = e.UAllAccountOrders(ctx,
					req.Pairs[i], 0, 0, req.StartTime, req.EndTime)
				if err != nil {
					return nil, err
				}
			case req.FromOrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.FromOrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = e.UAllAccountOrders(ctx,
					req.Pairs[i], fromID, 0, time.Time{}, time.Time{})
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
				fee, err := e.GetFee(ctx, &feeBuilder)
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
					Exchange:        e.Name,
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
	default:
		return orders, fmt.Errorf("%w %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineInterval(interval kline.Interval) string {
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
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	switch a {
	case asset.Spot, asset.Margin:
		var candles []CandleStick
		candles, err = e.GetSpotKline(ctx, &KlinesRequestParams{
			Interval:  e.FormatExchangeKlineInterval(req.ExchangeInterval),
			Symbol:    req.Pair,
			StartTime: req.Start,
			EndTime:   req.End,
			Limit:     req.RequestLimit,
		})
		if err != nil {
			return nil, err
		}
		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].OpenTime.Time(),
				Open:   candles[i].Open.Float64(),
				High:   candles[i].High.Float64(),
				Low:    candles[i].Low.Float64(),
				Close:  candles[i].Close.Float64(),
				Volume: candles[i].Volume.Float64(),
			})
		}
	case asset.USDTMarginedFutures:
		var candles []FuturesCandleStick
		candles, err = e.UKlineData(ctx,
			req.RequestFormatted,
			e.FormatExchangeKlineInterval(interval),
			req.RequestLimit,
			req.Start,
			req.End)
		if err != nil {
			return nil, err
		}
		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].OpenTime.Time(),
				Open:   candles[i].Open.Float64(),
				High:   candles[i].High.Float64(),
				Low:    candles[i].Low.Float64(),
				Close:  candles[i].Close.Float64(),
				Volume: candles[i].Volume.Float64(),
			})
		}
	case asset.CoinMarginedFutures:
		var candles []FuturesCandleStick
		candles, err = e.GetFuturesKlineData(ctx,
			req.RequestFormatted,
			e.FormatExchangeKlineInterval(interval),
			req.RequestLimit,
			req.Start,
			req.End)
		if err != nil {
			return nil, err
		}
		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].OpenTime.Time(),
				Open:   candles[i].Open.Float64(),
				High:   candles[i].High.Float64(),
				Low:    candles[i].Low.Float64(),
				Close:  candles[i].Close.Float64(),
				Volume: candles[i].Volume.Float64(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set
// time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin:
			var candles []CandleStick
			candles, err = e.GetSpotKline(ctx, &KlinesRequestParams{
				Interval:  e.FormatExchangeKlineInterval(req.ExchangeInterval),
				Symbol:    req.Pair,
				StartTime: req.RangeHolder.Ranges[x].Start.Time,
				EndTime:   req.RangeHolder.Ranges[x].End.Time,
				Limit:     req.RequestLimit,
			})
			if err != nil {
				return nil, err
			}
			for i := range candles {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[i].OpenTime.Time(),
					Open:   candles[i].Open.Float64(),
					High:   candles[i].High.Float64(),
					Low:    candles[i].Low.Float64(),
					Close:  candles[i].Close.Float64(),
					Volume: candles[i].Volume.Float64(),
				})
			}
		case asset.USDTMarginedFutures:
			var candles []FuturesCandleStick
			candles, err = e.UKlineData(ctx,
				req.RequestFormatted,
				e.FormatExchangeKlineInterval(interval),
				req.RangeHolder.Limit,
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for i := range candles {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[i].OpenTime.Time(),
					Open:   candles[i].Open.Float64(),
					High:   candles[i].High.Float64(),
					Low:    candles[i].Low.Float64(),
					Close:  candles[i].Close.Float64(),
					Volume: candles[i].Volume.Float64(),
				})
			}
		case asset.CoinMarginedFutures:
			var candles []FuturesCandleStick
			candles, err = e.GetFuturesKlineData(ctx,
				req.RequestFormatted,
				e.FormatExchangeKlineInterval(interval),
				req.RangeHolder.Limit,
				req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for i := range candles {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[i].OpenTime.Time(),
					Open:   candles[i].Open.Float64(),
					High:   candles[i].High.Float64(),
					Low:    candles[i].Low.Float64(),
					Close:  candles[i].Close.Float64(),
					Volume: candles[i].Volume.Float64(),
				})
			}
		default:
			return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
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
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var l []limits.MinMaxLevel
	var err error
	switch a {
	case asset.Spot:
		l, err = e.FetchExchangeLimits(ctx, asset.Spot)
	case asset.USDTMarginedFutures:
		l, err = e.FetchUSDTMarginExchangeLimits(ctx)
	case asset.CoinMarginedFutures:
		l, err = e.FetchCoinMarginExchangeLimits(ctx)
	case asset.Margin:
		l, err = e.FetchExchangeLimits(ctx, asset.Margin)
	default:
		err = fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %w", err)
	}
	return limits.Load(l)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	coinInfo, err := e.GetAllCoinsInfo(ctx)
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
func (e *Exchange) FormatExchangeCurrency(p currency.Pair, a asset.Item) (currency.Pair, error) {
	pairFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	if a == asset.USDTMarginedFutures {
		return e.formatUSDTMarginedFuturesPair(p, pairFmt), nil
	}
	return p.Format(pairFmt), nil
}

// FormatSymbol formats the given pair to a string suitable for exchange API requests
// overrides default implementation to use optional delimiter
func (e *Exchange) FormatSymbol(p currency.Pair, a asset.Item) (string, error) {
	pairFmt, err := e.GetPairFormat(a, true)
	if err != nil {
		return p.String(), err
	}
	if a == asset.USDTMarginedFutures {
		p = e.formatUSDTMarginedFuturesPair(p, pairFmt)
		return p.String(), nil
	}
	return pairFmt.Format(p), nil
}

// formatUSDTMarginedFuturesPair Binance USDTMarginedFutures pairs have a delimiter
// only if the contract has an expiry date
func (e *Exchange) formatUSDTMarginedFuturesPair(p currency.Pair, pairFmt currency.PairFormat) currency.Pair {
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
func (e *Exchange) GetServerTime(ctx context.Context, ai asset.Item) (time.Time, error) {
	switch ai {
	case asset.USDTMarginedFutures:
		return e.UServerTime(ctx)
	case asset.Spot:
		info, err := e.GetExchangeInfo(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return info.ServerTime.Time(), nil
	case asset.CoinMarginedFutures:
		info, err := e.FuturesExchangeInfo(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return info.ServerTime.Time(), nil
	}
	return time.Time{}, fmt.Errorf("%s %w", ai, asset.ErrNotSupported)
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
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
		format, err = e.GetPairFormat(r.Asset, true)
		if err != nil {
			return nil, err
		}
		fPair = r.Pair.Format(format)
	}

	switch r.Asset {
	case asset.USDTMarginedFutures:
		var mp []UMarkPrice
		var fri []FundingRateInfoResponse
		fri, err = e.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		mp, err = e.UGetMarkPrice(ctx, fPair)
		if err != nil {
			return nil, err
		}
		resp := make([]fundingrate.LatestRateResponse, 0, len(mp))
		for i := range mp {
			var cp currency.Pair
			var isEnabled bool
			cp, isEnabled, err = e.MatchSymbolCheckEnabled(mp[i].Symbol, r.Asset, true)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return nil, err
			}
			if !isEnabled {
				continue
			}
			var isPerp bool
			isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}
			var fundingRateFrequency int64 = 8
			for x := range fri {
				if fri[x].Symbol != mp[i].Symbol {
					continue
				}
				fundingRateFrequency = fri[x].FundingIntervalHours
				break
			}
			nft := mp[i].NextFundingTime.Time()
			cft := nft.Add(-time.Hour * time.Duration(fundingRateFrequency))
			rate := fundingrate.LatestRateResponse{
				TimeChecked: time.Now(),
				Exchange:    e.Name,
				Asset:       r.Asset,
				Pair:        cp,
				LatestRate: fundingrate.Rate{
					Time: cft,
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
		var fri []FundingRateInfoResponse
		fri, err = e.GetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		var mp []IndexMarkPrice
		mp, err = e.GetIndexAndMarkPrice(ctx, fPair.String(), "")
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
			isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}
			var fundingRateFrequency int64 = 8
			for x := range fri {
				if fri[x].Symbol != mp[i].Symbol {
					continue
				}
				fundingRateFrequency = fri[x].FundingIntervalHours
				break
			}
			nft := mp[i].NextFundingTime.Time()
			cft := nft.Add(-time.Hour * time.Duration(fundingRateFrequency))
			rate := fundingrate.LatestRateResponse{
				TimeChecked: time.Now(),
				Exchange:    e.Name,
				Asset:       r.Asset,
				Pair:        cp,
				LatestRate: fundingrate.Rate{
					Time: cft,
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
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
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
	switch r.Asset {
	case asset.USDTMarginedFutures:
		requestLimit := 1000
		sd := r.StartDate
		var fri []FundingRateInfoResponse
		fri, err = e.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		var fundingRateFrequency int64 = 8
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
			frh, err = e.UGetFundingHistory(ctx, fPair, int64(requestLimit), sd, r.EndDate)
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
		mp, err = e.UGetMarkPrice(ctx, fPair)
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
			income, err = e.UAccountIncomeHistory(ctx, fPair, "FUNDING_FEE", int64(requestLimit), r.StartDate, r.EndDate)
			if err != nil {
				return nil, err
			}
			for j := range income {
				for x := range pairRate.FundingRates {
					tt := income[j].Time.Time().Truncate(time.Duration(fundingRateFrequency) * time.Hour)
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
		fri, err = e.GetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		var fundingRateFrequency int64 = 8
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
			frh, err = e.FuturesGetFundingHistory(ctx, fPair, int64(requestLimit), sd, r.EndDate)
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
		mp, err = e.GetIndexAndMarkPrice(ctx, fPair.String(), "")
		if err != nil {
			return nil, err
		}
		pairRate.LatestRate = fundingrate.Rate{
			Time: mp[len(mp)-1].NextFundingTime.Time().Add(-time.Hour * time.Duration(fundingRateFrequency)),
			Rate: mp[len(mp)-1].LastFundingRate.Decimal(),
		}
		pairRate.TimeOfNextRate = mp[len(mp)-1].NextFundingTime.Time()
		if r.IncludePayments {
			var income []FuturesIncomeHistoryData
			income, err = e.FuturesIncomeHistory(ctx, fPair, "FUNDING_FEE", r.StartDate, r.EndDate, int64(requestLimit))
			if err != nil {
				return nil, err
			}
			for j := range income {
				for x := range pairRate.FundingRates {
					tt := income[j].Timestamp.Time().Truncate(8 * time.Hour)
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
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	if a == asset.CoinMarginedFutures {
		return cp.Quote.Equal(currency.PERP), nil
	}
	if a == asset.USDTMarginedFutures {
		return cp.Quote.Equal(currency.USDT) || cp.Quote.Equal(currency.BUSD), nil
	}
	return false, nil
}

// SetCollateralMode sets the account's collateral mode for the asset type
func (e *Exchange) SetCollateralMode(ctx context.Context, a asset.Item, collateralMode collateral.Mode) error {
	if a != asset.USDTMarginedFutures {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if collateralMode != collateral.MultiMode && collateralMode != collateral.SingleMode {
		return fmt.Errorf("%w %v", order.ErrCollateralInvalid, collateralMode)
	}
	return e.SetAssetsMode(ctx, collateralMode == collateral.MultiMode)
}

// GetCollateralMode returns the account's collateral mode for the asset type
func (e *Exchange) GetCollateralMode(ctx context.Context, a asset.Item) (collateral.Mode, error) {
	if a != asset.USDTMarginedFutures {
		return collateral.UnknownMode, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	isMulti, err := e.GetAssetsMode(ctx)
	if err != nil {
		return collateral.UnknownMode, err
	}
	if isMulti {
		return collateral.MultiMode, nil
	}
	return collateral.SingleMode, nil
}

// SetMarginType sets the default margin type for when opening a new position
func (e *Exchange) SetMarginType(ctx context.Context, item asset.Item, pair currency.Pair, tp margin.Type) error {
	if item != asset.USDTMarginedFutures && item != asset.CoinMarginedFutures {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	if !tp.Valid() {
		return fmt.Errorf("%w %v", margin.ErrInvalidMarginType, tp)
	}
	mt, err := e.marginTypeToString(tp)
	if err != nil {
		return err
	}
	switch item {
	case asset.CoinMarginedFutures:
		_, err = e.FuturesChangeMarginType(ctx, pair, mt)
	case asset.USDTMarginedFutures:
		err = e.UChangeInitialMarginType(ctx, pair, mt)
	}
	if err != nil {
		return err
	}

	return nil
}

// ChangePositionMargin will modify a position/currencies margin parameters
func (e *Exchange) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
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
		_, err = e.ModifyIsolatedPositionMargin(ctx, req.Pair, side, marginType, req.NewAllocatedMargin)
	case asset.USDTMarginedFutures:
		_, err = e.UModifyIsolatedPositionMarginReq(ctx, req.Pair, side, marginType, req.NewAllocatedMargin)
	}
	if err != nil {
		return nil, err
	}

	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		MarginType:      req.MarginType,
		AllocatedMargin: req.NewAllocatedMargin,
	}, nil
}

// marginTypeToString converts the GCT margin type to Binance's string
func (e *Exchange) marginTypeToString(mt margin.Type) (string, error) {
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
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionSummary", common.ErrNilPointer)
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	fPair, err := e.FormatExchangeCurrency(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	switch req.Asset {
	case asset.USDTMarginedFutures:
		ai, err := e.UAccountInformationV2(ctx)
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
		contracts, err = e.GetFuturesContractDetails(ctx, req.Asset)
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

		switch collateralMode {
		case collateral.SingleMode:
			var collateralAsset *UAsset
			switch {
			case strings.Contains(accountPosition.Symbol, usdtAsset.Asset):
				collateralAsset = usdtAsset
			case strings.Contains(accountPosition.Symbol, busdAsset.Asset):
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

		case collateral.MultiMode:
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
		positionsInfo, err := e.UPositionsInfoV2(ctx, fPair)
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
		ai, err := e.GetFuturesAccountInfo(ctx)
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
			// TODO: utilise contract data to discern the underlying currency instead of having a user provide it
			if !ai.Assets[i].Asset.Equal(req.UnderlyingPair.Base) {
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
		positionsInfo, err := e.FuturesPositionsInfo(ctx, "", req.Pair.Base.String())
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
		contracts, err = e.GetFuturesContractDetails(ctx, req.Asset)
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
			Currency:                     accountAsset.Asset,
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
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionOrders", common.ErrNilPointer)
	}
	if len(req.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if time.Since(req.StartDate) > e.Features.Supports.MaximumOrderHistory+time.Hour {
		if req.RespectOrderHistoryLimits {
			req.StartDate = time.Now().Add(-e.Features.Supports.MaximumOrderHistory)
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, time.Now().Add(-e.Features.Supports.MaximumOrderHistory))
		}
	}
	if req.EndDate.IsZero() {
		req.EndDate = time.Now()
	}

	var resp []futures.PositionResponse
	sd := req.StartDate
	switch req.Asset {
	case asset.USDTMarginedFutures:
		orderLimit := 1000
		for x := range req.Pairs {
			fPair, err := e.FormatExchangeCurrency(req.Pairs[x], req.Asset)
			if err != nil {
				return nil, err
			}
			result, err := e.UPositionsInfoV2(ctx, fPair)
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
					orders, err = e.UAllAccountOrders(ctx, fPair, 0, int64(orderLimit), sd, req.EndDate)
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
							Exchange:             e.Name,
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
		orderLimit := 100
		for x := range req.Pairs {
			fPair, err := e.FormatExchangeCurrency(req.Pairs[x], req.Asset)
			if err != nil {
				return nil, err
			}
			// "pair" for coinmarginedfutures is the pair.Base
			// eg ADAUSD_PERP the pair is ADAUSD
			result, err := e.FuturesPositionsInfo(ctx, "", fPair.Base.String())
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
					orders, err = e.GetAllFuturesOrders(ctx, fPair, currency.EMPTYPAIR, sd, req.EndDate, 0, int64(orderLimit))
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
							Exchange:             e.Name,
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
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	return resp, nil
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (e *Exchange) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, amount float64, _ order.Side) error {
	switch item {
	case asset.USDTMarginedFutures:
		_, err := e.UChangeInitialLeverageRequest(ctx, pair, amount)
		return err
	case asset.CoinMarginedFutures:
		_, err := e.FuturesChangeInitialLeverage(ctx, pair, amount)
		return err
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (e *Exchange) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, _ order.Side) (float64, error) {
	if pair.IsEmpty() {
		return -1, currency.ErrCurrencyPairEmpty
	}
	switch item {
	case asset.USDTMarginedFutures:
		resp, err := e.UPositionsInfoV2(ctx, pair)
		if err != nil {
			return -1, err
		}
		if len(resp) == 0 {
			return -1, fmt.Errorf("%w %v %v", futures.ErrPositionNotFound, item, pair)
		}
		// leverage is the same across positions
		return resp[0].Leverage, nil
	case asset.CoinMarginedFutures:
		resp, err := e.FuturesPositionsInfo(ctx, "", pair.Base.String())
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
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	switch item {
	case asset.USDTMarginedFutures:
		fri, err := e.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		ei, err := e.UExchangeInfo(ctx)
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
				Exchange:           e.Name,
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
		fri, err := e.GetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		ei, err := e.FuturesExchangeInfo(ctx)
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
				Exchange:           e.Name,
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
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
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
			oi, err := e.UOpenInterest(ctx, k[i].Pair())
			if err != nil {
				return nil, err
			}
			result[i] = futures.OpenInterest{
				Key:          key.NewExchangeAssetPair(e.Name, k[i].Asset, k[i].Pair()),
				OpenInterest: oi.OpenInterest,
			}
		case asset.CoinMarginedFutures:
			oi, err := e.OpenInterest(ctx, k[i].Pair())
			if err != nil {
				return nil, err
			}
			result[i] = futures.OpenInterest{
				Key:          key.NewExchangeAssetPair(e.Name, k[i].Asset, k[i].Pair()),
				OpenInterest: oi.OpenInterest,
			}
		}
	}
	return result, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	symbol, err := e.FormatSymbol(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.USDTMarginedFutures:
		var ct string
		if !cp.Quote.Equal(currency.USDT) && !cp.Quote.Equal(currency.BUSD) {
			ei, err := e.UExchangeInfo(ctx)
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
			ei, err := e.FuturesExchangeInfo(ctx)
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
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}
