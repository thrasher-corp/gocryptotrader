package binance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
func (b *Binance) GetDefaultConfig() (*config.Exchange, error) {
	b.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Binance
func (b *Binance) SetDefaults() {
	b.Name = "Binance"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.SetValues()

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}
	coinFutures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
	}
	usdtFutures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
	}
	err := b.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.Margin, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.DisableAssetWebsocketSupport(asset.Margin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.CoinMarginedFutures, coinFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.DisableAssetWebsocketSupport(asset.CoinMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.USDTMarginedFutures, usdtFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.DisableAssetWebsocketSupport(asset.USDTMarginedFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
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
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.OneMin,
					kline.ThreeMin,
					kline.FiveMin,
					kline.FifteenMin,
					kline.ThirtyMin,
					kline.OneHour,
					kline.TwoHour,
					kline.FourHour,
					kline.SixHour,
					kline.EightHour,
					kline.TwelveHour,
					kline.OneDay,
					kline.ThreeDay,
					kline.OneWeek,
					kline.OneMonth,
				),
				ResultLimit: 1000,
			},
		},
	}

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
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

	b.Websocket = stream.New()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Binance) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	err = b.SetupDefaults(exch)
	if err != nil {
		return err
	}
	ePoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             binanceDefaultWebsocketURL,
		RunningURL:             ePoint,
		Connector:              b.WsConnect,
		Subscriber:             b.Subscribe,
		Unsubscriber:           b.Unsubscribe,
		GenerateSubscriptions:  b.GenerateSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &b.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
		TradeFeed: b.Features.Enabled.TradeFeed,
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            wsRateLimitMilliseconds,
	})
}

// Start starts the Binance go routine
func (b *Binance) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Binance wrapper
func (b *Binance) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()),
			b.Websocket.GetWebsocketURL())
		b.PrintEnabledPairs()
	}

	forceUpdate := false
	a := b.GetAssetTypes(true)
	for x := range a {
		if err := b.UpdateOrderExecutionLimits(context.TODO(), a[x]); err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to set exchange order execution limits. Err: %v",
				b.Name,
				err)
		}
		if a[x] == asset.USDTMarginedFutures && !b.BypassConfigFormatUpgrades {
			format, err := b.GetPairFormat(asset.USDTMarginedFutures, false)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s failed to get enabled currencies. Err %s\n",
					b.Name,
					err)
				return
			}
			var enabled, avail currency.Pairs
			enabled, err = b.CurrencyPairs.GetPairs(asset.USDTMarginedFutures, true)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s failed to get enabled currencies. Err %s\n",
					b.Name,
					err)
				return
			}

			avail, err = b.CurrencyPairs.GetPairs(asset.USDTMarginedFutures, false)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s failed to get available currencies. Err %s\n",
					b.Name,
					err)
				return
			}
			if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
				!common.StringDataContains(avail.Strings(), format.Delimiter) {
				var enabledPairs currency.Pairs
				enabledPairs, err = currency.NewPairsFromStrings([]string{
					currency.BTC.String() + format.Delimiter + currency.USDT.String(),
				})
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s failed to update currencies. Err %s\n",
						b.Name,
						err)
				} else {
					log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, b.Name, a[x], enabledPairs)
					forceUpdate = true
					err = b.UpdatePairs(enabledPairs, a[x], true, true)
					if err != nil {
						log.Errorf(log.ExchangeSys,
							"%s failed to update currencies. Err: %s\n",
							b.Name,
							err)
					}
				}
			}
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	if err := b.UpdateTradablePairs(context.TODO(), forceUpdate); err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Binance) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !b.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, b.Name)
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
	return nil
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
					Last:         tick[y].LastPrice,
					High:         tick[y].HighPrice,
					Low:          tick[y].LowPrice,
					Bid:          tick[y].BidPrice,
					Ask:          tick[y].AskPrice,
					Volume:       tick[y].Volume,
					QuoteVolume:  tick[y].QuoteVolume,
					Open:         tick[y].OpenPrice,
					Close:        tick[y].PrevClosePrice,
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
	default:
		return fmt.Errorf("assetType not supported: %v", a)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Binance) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	switch a {
	case asset.Spot, asset.Margin:
		tick, err := b.GetPriceChangeStats(ctx, p)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick.LastPrice,
			High:         tick.HighPrice,
			Low:          tick.LowPrice,
			Bid:          tick.BidPrice,
			Ask:          tick.AskPrice,
			Volume:       tick.Volume,
			QuoteVolume:  tick.QuoteVolume,
			Open:         tick.OpenPrice,
			Close:        tick.PrevClosePrice,
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

	default:
		return nil, fmt.Errorf("assetType not supported: %v", a)
	}
	return ticker.GetTicker(b.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (b *Binance) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Binance) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		// TODO: Disconnect update orderbook functionality from fetch orderbook
		// functionality across all wrappers as this mutes potential errors.
		return b.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}
	var orderbookNew *OrderBook
	var err error
	switch assetType {
	case asset.Spot, asset.Margin:
		orderbookNew, err = b.GetOrderBook(ctx,
			OrderBookDataRequestParams{
				Symbol: p,
				Limit:  1000})
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

	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Quantity,
			Price:  orderbookNew.Bids[x].Price,
		}
	}
	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
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
		return info, fmt.Errorf("%v assetType not supported", assetType)
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

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Binance) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(b.Name, creds, assetType)
	if err != nil {
		return b.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Binance) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Binance) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	w, err := b.WithdrawHistory(ctx, c, "", time.Time{}, time.Time{}, 0, 10000)
	if err != nil {
		return nil, err
	}

	for i := range w {
		tm, err := time.Parse(binanceSAPITimeLayout, w[i].ApplyTime)
		if err != nil {
			return nil, err
		}
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          strconv.FormatInt(w[i].Status, 10),
			TransferID:      w[i].ID,
			Currency:        w[i].Coin,
			Amount:          w[i].Amount,
			Fee:             w[i].TransactionFee,
			CryptoToAddress: w[i].Address,
			CryptoTxID:      w[i].TransactionID,
			CryptoChain:     w[i].Network,
			Timestamp:       tm,
		})
	}

	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Binance) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	const limit = 1000
	tradeData, err := b.GetMostRecentTrades(ctx,
		RecentTradeRequestParams{p, limit})
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		resp[i] = trade.Data{
			TID:          strconv.FormatInt(tradeData[i].ID, 10),
			Exchange:     b.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Quantity,
			Timestamp:    tradeData[i].Time,
		}
	}
	if b.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(b.Name, resp...)
		if err != nil {
			return nil, err
		}
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Binance) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, from, to time.Time) ([]trade.Data, error) {
	req := AggregatedTradeRequestParams{
		Symbol:    p,
		StartTime: from,
		EndTime:   to,
	}
	trades, err := b.GetAggregatedTrades(ctx, &req)
	if err != nil {
		return nil, err
	}
	result := make([]trade.Data, len(trades))
	exName := b.GetName()
	for i := range trades {
		t := trades[i].toTradeData(p, exName, a)
		result[i] = *t
	}
	return result, nil
}

func (a *AggregatedTrade) toTradeData(p currency.Pair, exchange string, aType asset.Item) *trade.Data {
	return &trade.Data{
		CurrencyPair: p,
		TID:          strconv.FormatInt(a.ATradeID, 10),
		Amount:       a.Quantity,
		Exchange:     exchange,
		Price:        a.Price,
		Timestamp:    a.TimeStamp,
		AssetType:    aType,
		Side:         order.AnySide,
	}
}

// SubmitOrder submits a new order
func (b *Binance) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	var orderID string
	status := order.New
	var trades []order.TradeHistory
	switch s.AssetType {
	case asset.Spot, asset.Margin:
		var sideType string
		if s.Side == order.Buy {
			sideType = order.Buy.String()
		} else {
			sideType = order.Sell.String()
		}

		timeInForce := BinanceRequestParamsTimeGTC
		var requestParamsOrderType RequestParamsOrderType
		switch s.Type {
		case order.Market:
			timeInForce = ""
			requestParamsOrderType = BinanceRequestParamsOrderMarket
		case order.Limit:
			if s.ImmediateOrCancel {
				timeInForce = BinanceRequestParamsTimeIOC
			}
			requestParamsOrderType = BinanceRequestParamsOrderLimit
		default:
			return nil, errors.New("unsupported order type")
		}

		var orderRequest = NewOrderRequest{
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
			return nil, fmt.Errorf("invalid side")
		}

		var (
			oType       string
			timeInForce RequestParamsTimeForceType
		)

		switch s.Type {
		case order.Limit:
			oType = cfuturesLimit
			timeInForce = BinanceRequestParamsTimeGTC
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
			return nil, fmt.Errorf("invalid side")
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
		order, err := b.UFuturesNewOrder(ctx,
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
		orderID = strconv.FormatInt(order.OrderID, 10)
	default:
		return nil, fmt.Errorf("assetType not supported")
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
func (b *Binance) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
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
		return cancelAllOrdersResponse, fmt.Errorf("assetType not supported: %v", req.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Binance) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var respData order.Detail
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return respData, err
	}
	switch assetType {
	case asset.Spot:
		resp, err := b.QueryOrder(ctx, pair, "", orderIDInt)
		if err != nil {
			return respData, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(resp.Side)
		if err != nil {
			return respData, err
		}
		status, err := order.StringToOrderStatus(resp.Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}
		orderType := order.Limit
		if resp.Type == "MARKET" {
			orderType = order.Market
		}

		return order.Detail{
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
			Date:           resp.Time,
			LastUpdated:    resp.UpdateTime,
		}, nil
	case asset.CoinMarginedFutures:
		orderData, err := b.FuturesOpenOrderData(ctx, pair, orderID, "")
		if err != nil {
			return respData, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := b.GetFee(ctx, &feeBuilder)
		if err != nil {
			return respData, err
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
		respData.Date = orderData.Time
		respData.LastUpdated = orderData.UpdateTime
	case asset.USDTMarginedFutures:
		orderData, err := b.UGetOrderData(ctx, pair, orderID, "")
		if err != nil {
			return respData, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := b.GetFee(ctx, &feeBuilder)
		if err != nil {
			return respData, err
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
		respData.Date = orderData.Time
		respData.LastUpdated = orderData.UpdateTime
	default:
		return respData, fmt.Errorf("assetType %s not supported", assetType)
	}
	return respData, nil
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
func (b *Binance) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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
					Date:          resp[x].Time,
					Exchange:      b.Name,
					OrderID:       strconv.FormatInt(resp[x].OrderID, 10),
					ClientOrderID: resp[x].ClientOrderID,
					Side:          side,
					Type:          orderType,
					Price:         resp[x].Price,
					Status:        orderStatus,
					Pair:          req.Pairs[i],
					AssetType:     req.AssetType,
					LastUpdated:   resp[x].UpdateTime,
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
					Date:            openOrders[y].Time,
					LastUpdated:     openOrders[y].UpdateTime,
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
					Date:            openOrders[y].Time,
					LastUpdated:     openOrders[y].UpdateTime,
				})
			}
		default:
			return orders, fmt.Errorf("assetType not supported")
		}
	}
	return req.Filter(b.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Binance) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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
					Date:            resp[i].Time,
					LastUpdated:     resp[i].UpdateTime,
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
			case !req.StartTime.IsZero() && !req.EndTime.IsZero() && req.OrderID == "":
				if req.EndTime.Before(req.StartTime) {
					return nil, errors.New("endTime cannot be before startTime")
				}
				if time.Since(req.StartTime) > time.Hour*24*30 {
					return nil, fmt.Errorf("can only fetch orders 30 days out")
				}
				orderHistory, err = b.GetAllFuturesOrders(ctx,
					req.Pairs[i], "", req.StartTime, req.EndTime, 0, 0)
				if err != nil {
					return nil, err
				}
			case req.OrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.OrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = b.GetAllFuturesOrders(ctx,
					req.Pairs[i], "", time.Time{}, time.Time{}, fromID, 0)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("invalid combination of input params")
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
					Date:            orderHistory[y].Time,
				})
			}
		}
	case asset.USDTMarginedFutures:
		for i := range req.Pairs {
			var orderHistory []UFuturesOrderData
			var err error
			switch {
			case !req.StartTime.IsZero() && !req.EndTime.IsZero() && req.OrderID == "":
				if req.EndTime.Before(req.StartTime) {
					return nil, errors.New("endTime cannot be before startTime")
				}
				if time.Since(req.StartTime) > time.Hour*24*7 {
					return nil, fmt.Errorf("can only fetch orders 7 days out")
				}
				orderHistory, err = b.UAllAccountOrders(ctx,
					req.Pairs[i], 0, 0, req.StartTime, req.EndTime)
				if err != nil {
					return nil, err
				}
			case req.OrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.OrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = b.UAllAccountOrders(ctx,
					req.Pairs[i], fromID, 0, time.Time{}, time.Time{})
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("invalid combination of input params")
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
					Date:            orderHistory[y].Time,
				})
			}
		}
	default:
		return orders, fmt.Errorf("assetType not supported")
	}
	return req.Filter(b.Name, orders), nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Binance) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
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
	req, err := b.GetKlineRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	if a != asset.Spot {
		// TODO: Add support for other asset types.
		return nil, common.ErrNotYetImplemented
	}

	candles, err := b.GetSpotKline(ctx, &KlinesRequestParams{
		Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
		Symbol:    req.Pair,
		StartTime: req.Start,
		EndTime:   req.End,
		Limit:     int(b.Features.Enabled.Kline.ResultLimit),
	})
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		}
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

	if a != asset.Spot {
		// TODO: Add support for other asset types.
		return nil, common.ErrNotYetImplemented
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles []CandleStick
		candles, err = b.GetSpotKline(ctx, &KlinesRequestParams{
			Interval:  b.FormatExchangeKlineInterval(req.ExchangeInterval),
			Symbol:    req.Pair,
			StartTime: req.RangeHolder.Ranges[x].Start.Time,
			EndTime:   req.RangeHolder.Ranges[x].End.Time,
			Limit:     int(b.Features.Enabled.Kline.ResultLimit),
		})
		if err != nil {
			return nil, err
		}

		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].OpenTime,
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
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
		limits, err = b.FetchSpotExchangeLimits(ctx)
	case asset.USDTMarginedFutures:
		limits, err = b.FetchUSDTMarginExchangeLimits(ctx)
	case asset.CoinMarginedFutures:
		limits, err = b.FetchCoinMarginExchangeLimits(ctx)
	case asset.Margin:
		if err = b.CurrencyPairs.IsAssetEnabled(asset.Spot); err != nil {
			limits, err = b.FetchSpotExchangeLimits(ctx)
		} else {
			return nil
		}
	default:
		err = fmt.Errorf("unhandled asset type %s", a)
	}
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %v", err)
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
		return info.Servertime, nil
	case asset.CoinMarginedFutures:
		info, err := b.FuturesExchangeInfo(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return time.UnixMilli(info.ServerTime), nil
	}
	return time.Time{}, fmt.Errorf("%s %w", ai, asset.ErrNotSupported)
}
