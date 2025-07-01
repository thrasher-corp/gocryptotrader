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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
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
func (b *Binance) SetDefaults() {
	b.Name = "Binance"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	for a, ps := range defaultAssetPairStores {
		if err := b.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", b.Name, a, err)
		}
	}

	for _, a := range []asset.Item{asset.Margin, asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		if err := b.DisableAssetWebsocketSupport(a); err != nil {
			log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", b.Name, a, err)
		}
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
	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
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

	b.Websocket = websocket.NewManager()
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
	err = b.Websocket.Setup(&websocket.ManagerSetup{
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

	return b.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
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
	var pairs []currency.Pair
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
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Binance) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := b.GetAssetTypes(false)
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
		tick, err := b.GetTickers(ctx)
		if err != nil {
			return err
		}

		pairs, err := b.GetEnabledPairs(a)
		if err != nil {
			return err
		}

		for i := range pairs {
			for y := range tick {
				pairFmt, err := b.FormatExchangeCurrency(pairs[i], a)
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
	switch a {
	case asset.Spot, asset.Margin:
		tick, err := b.GetPriceChangeStats(ctx, p)
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
			ExchangeName: b.Name,
			AssetType:    a,
		})
		if err != nil {
			return nil, err
		}
	case asset.USDTMarginedFutures:
		tick, err := b.U24HTickerPriceChangeStats(ctx, p)
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
		tick, err := b.GetFuturesSwapTickerChangeStats(ctx, p, "")
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

	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	return ticker.GetTicker(b.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          b.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: b.ValidateOrderbook,
	}
	var orderbookNew *OrderBook
	var err error

	switch assetType {
	case asset.Spot, asset.Margin:
		orderbookNew, err = b.GetOrderBook(ctx,
			OrderBookDataRequestParams{
				Symbol: p,
				Limit:  1000,
			})
	case asset.USDTMarginedFutures:
		orderbookNew, err = b.UFuturesOrderbook(ctx, p, 1000)
	case asset.CoinMarginedFutures:
		orderbookNew, err = b.GetFuturesOrderbook(ctx, p, 1000)
	default:
		return nil, fmt.Errorf("[%s] %w", assetType, asset.ErrNotSupported)
	}
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Bids[x].Quantity,
			Price:  orderbookNew.Bids[x].Price,
		}
	}
	book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Asks[x].Quantity,
			Price:  orderbookNew.Asks[x].Price,
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
		creds, err := b.GetCredentials(ctx)
		if err != nil {
			return info, err
		}
		if creds.SubAccount != "" {
			// TODO: implement sub-account endpoints
			return info, common.ErrNotYetImplemented
		}
		raw, err := b.GetAccount(ctx)
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
		var currencyDetails []account.Balance
		for i := range accData.UserAssets {
			currencyDetails = append(currencyDetails, account.Balance{
				Currency:               currency.NewCode(accData.UserAssets[i].Asset),
				Total:                  accData.UserAssets[i].Free + accData.UserAssets[i].Locked,
				Hold:                   accData.UserAssets[i].Locked,
				Free:                   accData.UserAssets[i].Free,
				AvailableWithoutBorrow: accData.UserAssets[i].Free - accData.UserAssets[i].Borrowed,
				Borrowed:               accData.UserAssets[i].Borrowed,
			})
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
	case asset.Spot:
		tradeData, err := b.GetMostRecentTrades(ctx,
			RecentTradeRequestParams{pFmt, limit})
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			td := trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     b.Name,
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
		tradeData, err := b.URecentTrades(ctx, pFmt, "", limit)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			td := trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     b.Name,
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
		tradeData, err := b.GetFuturesPublicTrades(ctx, pFmt, limit)
		if err != nil {
			return nil, err
		}

		for i := range tradeData {
			td := trade.Data{
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				Exchange:     b.Name,
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

	if b.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(resp...)
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
	if a != asset.Spot {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	rFmt, err := b.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	pFmt := p.Format(rFmt)
	req := AggregatedTradeRequestParams{
		Symbol:    pFmt,
		StartTime: from,
		EndTime:   to,
	}
	trades, err := b.GetAggregatedTrades(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("%w %v", err, pFmt)
	}
	result := make([]trade.Data, len(trades))
	for i := range trades {
		td := trade.Data{
			CurrencyPair: p,
			TID:          strconv.FormatInt(trades[i].ATradeID, 10),
			Amount:       trades[i].Quantity,
			Exchange:     b.Name,
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
func (b *Binance) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(b.GetTradingRequirements()); err != nil {
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
		response, err := b.NewOrder(ctx, &orderRequest)
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

		o, err := b.FuturesNewOrder(
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
		o, err := b.UFuturesNewOrder(ctx,
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

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Binance) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Binance) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot, asset.Margin:
		orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
		if err != nil {
			return err
		}
		_, err = b.CancelExistingOrder(ctx,
			o.Pair,
			orderIDInt,
			o.AccountID)
		if err != nil {
			return err
		}
	case asset.CoinMarginedFutures:
		_, err := b.FuturesCancelOrder(ctx, o.Pair, o.OrderID, "")
		if err != nil {
			return err
		}
	case asset.USDTMarginedFutures:
		_, err := b.UCancelOrder(ctx, o.Pair, o.OrderID, "")
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
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch req.AssetType {
	case asset.Spot, asset.Margin:
		openOrders, err := b.OpenOrders(ctx, req.Pair)
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
	case asset.CoinMarginedFutures:
		if req.Pair.IsEmpty() {
			enabledPairs, err := b.GetEnabledPairs(asset.CoinMarginedFutures)
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
			_, err := b.FuturesCancelAllOpenOrders(ctx, req.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}
	case asset.USDTMarginedFutures:
		if req.Pair.IsEmpty() {
			enabledPairs, err := b.GetEnabledPairs(asset.USDTMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				_, err = b.UCancelAllOpenOrders(ctx, enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
			}
		} else {
			_, err := b.UCancelAllOpenOrders(ctx, req.Pair)
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
func (b *Binance) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	var respData order.Detail
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		resp, err := b.QueryOrder(ctx, pair, "", orderIDInt)
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
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}
		orderType := order.Limit
		if resp.Type == "MARKET" {
			orderType = order.Market
		}

		return &order.Detail{
			Amount:         resp.OrigQty,
			Exchange:       b.Name,
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
		respData.Amount = orderData.OriginalQuantity
		respData.AssetType = assetType
		respData.ClientOrderID = orderData.ClientOrderID
		respData.Exchange = b.Name
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
		orderData, err := b.UGetOrderData(ctx, pair, orderID, "")
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
		respData.Amount = orderData.OriginalQuantity
		respData.AssetType = assetType
		respData.ClientOrderID = orderData.ClientOrderID
		respData.Exchange = b.Name
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
func (b *Binance) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := b.GetDepositAddressForCurrency(ctx, cryptocurrency.String(), chain)
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
func (b *Binance) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	amountStr := strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64)
	v, err := b.WithdrawCrypto(ctx,
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
			resp, err := b.OpenOrders(ctx, req.Pairs[i])
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
					Amount:        resp[x].OrigQty,
					Date:          resp[x].Time.Time(),
					Exchange:      b.Name,
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
		return nil, errors.New("at least one currency is required to fetch order history")
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot, asset.Margin:
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
					Exchange:        b.Name,
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
					req.Pairs[i], 0, 0, req.StartTime, req.EndTime)
				if err != nil {
					return nil, err
				}
			case req.FromOrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.FromOrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = b.UAllAccountOrders(ctx,
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
		candles, err = b.GetSpotKline(ctx, &KlinesRequestParams{
			Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
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
		candles, err = b.UKlineData(ctx,
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
				Time:   candles[i].OpenTime.Time(),
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
			candles, err = b.GetSpotKline(ctx, &KlinesRequestParams{
				Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
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
			candles, err = b.UKlineData(ctx,
				req.RequestFormatted,
				b.FormatExchangeKlineInterval(interval),
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
			candles, err = b.GetFuturesKlineData(ctx,
				req.RequestFormatted,
				b.FormatExchangeKlineInterval(interval),
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
	case asset.Spot:
		limits, err = b.FetchExchangeLimits(ctx, asset.Spot)
	case asset.USDTMarginedFutures:
		limits, err = b.FetchUSDTMarginExchangeLimits(ctx)
	case asset.CoinMarginedFutures:
		limits, err = b.FetchCoinMarginExchangeLimits(ctx)
	case asset.Margin:
		limits, err = b.FetchExchangeLimits(ctx, asset.Margin)
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
	case asset.Spot:
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
	case asset.USDTMarginedFutures:
		var mp []UMarkPrice
		var fri []FundingRateInfoResponse
		fri, err = b.UGetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		mp, err = b.UGetMarkPrice(ctx, fPair)
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
				Exchange:    b.Name,
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
		fri, err = b.GetFundingRateInfo(ctx)
		if err != nil {
			return nil, err
		}
		var mp []IndexMarkPrice
		mp, err = b.GetIndexAndMarkPrice(ctx, fPair.String(), "")
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
				Exchange:    b.Name,
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
			frh, err = b.UGetFundingHistory(ctx, fPair, int64(requestLimit), sd, r.EndDate)
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
		mp, err = b.UGetMarkPrice(ctx, fPair)
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
			income, err = b.UAccountIncomeHistory(ctx, fPair, "FUNDING_FEE", int64(requestLimit), r.StartDate, r.EndDate)
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
		fri, err = b.GetFundingRateInfo(ctx)
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
			Time: mp[len(mp)-1].NextFundingTime.Time().Add(-time.Hour * time.Duration(fundingRateFrequency)),
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
		_, err = b.UModifyIsolatedPositionMarginReq(ctx, req.Pair, side, marginType, req.NewAllocatedMargin)
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
		orderLimit := 1000
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
					orders, err = b.UAllAccountOrders(ctx, fPair, 0, int64(orderLimit), sd, req.EndDate)
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
		orderLimit := 100
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
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	return resp, nil
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (b *Binance) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, _ margin.Type, amount float64, _ order.Side) error {
	switch item {
	case asset.USDTMarginedFutures:
		_, err := b.UChangeInitialLeverageRequest(ctx, pair, amount)
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
			oi, err := b.UOpenInterest(ctx, k[i].Pair())
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
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
