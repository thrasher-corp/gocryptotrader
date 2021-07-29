package binance

import (
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (b *Binance) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
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
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				TradeFetching:       true,
				UserTradeHistory:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
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
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.ThreeMin.Word():   true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.SixHour.Word():    true,
					kline.EightHour.Word():  true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.ThreeDay.Word():   true,
					kline.OneWeek.Word():    true,
					kline.OneMonth.Word():   true,
				},
				ResultLimit: 1000,
			},
		},
	}

	b.Requester = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
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
func (b *Binance) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}
	ePoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = b.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       binanceDefaultWebsocketURL,
		ExchangeName:                     exch.Name,
		RunningURL:                       ePoint,
		Connector:                        b.WsConnect,
		Subscriber:                       b.Subscribe,
		UnSubscriber:                     b.Unsubscribe,
		GenerateSubscriptions:            b.GenerateSubscriptions,
		Features:                         &b.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.OrderbookConfig.WebsocketBufferLimit,
		BufferEnabled:                    exch.OrderbookConfig.WebsocketBufferEnabled,
		SortBuffer:                       true,
		SortBufferByUpdateIDs:            true,
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
func (b *Binance) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
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
	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to get enabled currencies. Err %s\n",
			b.Name,
			err)
		return
	}
	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to get enabled currencies. Err %s\n",
			b.Name,
			err)
		return
	}

	avail, err := b.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to get available currencies. Err %s\n",
			b.Name,
			err)
		return
	}

	if !common.StringDataContains(pairs.Strings(), format.Delimiter) ||
		!common.StringDataContains(avail.Strings(), format.Delimiter) {
		var enabledPairs currency.Pairs
		enabledPairs, err = currency.NewPairsFromStrings([]string{
			currency.BTC.String() +
				format.Delimiter +
				currency.USDT.String()})
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to update currencies. Err %s\n",
				b.Name,
				err)
		} else {
			log.Warn(log.ExchangeSys,
				"Available pairs for Binance reset due to config upgrade, please enable the ones you would like to use again")
			forceUpdate = true

			err = b.UpdatePairs(enabledPairs, asset.Spot, true, true)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					b.Name,
					err)
			}
		}
	}

	a := b.GetAssetTypes(true)
	for x := range a {
		err = b.UpdateOrderExecutionLimits(a[x])
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to set exchange order execution limits. Err: %v",
				b.Name,
				err)
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}
	err = b.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Binance) FetchTradablePairs(a asset.Item) ([]string, error) {
	if !b.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, b.Name)
	}
	format, err := b.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	var pairs []string
	switch a {
	case asset.Spot, asset.Margin:
		info, err := b.GetExchangeInfo()
		if err != nil {
			return nil, err
		}
		for x := range info.Symbols {
			if info.Symbols[x].Status == "TRADING" {
				pair := info.Symbols[x].BaseAsset +
					format.Delimiter +
					info.Symbols[x].QuoteAsset
				if a == asset.Spot && info.Symbols[x].IsSpotTradingAllowed {
					pairs = append(pairs, pair)
				}
				if a == asset.Margin && info.Symbols[x].IsMarginTradingAllowed {
					pairs = append(pairs, pair)
				}
			}
		}
	case asset.CoinMarginedFutures:
		cInfo, err := b.FuturesExchangeInfo()
		if err != nil {
			return pairs, nil
		}
		for z := range cInfo.Symbols {
			if cInfo.Symbols[z].ContractStatus == "TRADING" {
				curr, err := currency.NewPairFromString(cInfo.Symbols[z].Symbol)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, format.Format(curr))
			}
		}
	case asset.USDTMarginedFutures:
		uInfo, err := b.UExchangeInfo()
		if err != nil {
			return pairs, nil
		}
		for u := range uInfo.Symbols {
			if uInfo.Symbols[u].Status == "TRADING" {
				curr, err := currency.NewPairFromString(uInfo.Symbols[u].Symbol)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, format.Format(curr))
			}
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Binance) UpdateTradablePairs(forceUpdate bool) error {
	assetTypes := b.GetAssetTypes(false)
	for i := range assetTypes {
		p, err := b.FetchTradablePairs(assetTypes[i])
		if err != nil {
			return err
		}

		pairs, err := currency.NewPairsFromStrings(p)
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

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Binance) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	switch assetType {
	case asset.Spot, asset.Margin:
		tick, err := b.GetTickers()
		if err != nil {
			return nil, err
		}
		for y := range tick {
			cp, err := currency.NewPairFromString(tick[y].Symbol)
			if err != nil {
				return nil, err
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
				Pair:         cp,
				ExchangeName: b.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return nil, err
			}
		}
	case asset.USDTMarginedFutures:
		tick, err := b.U24HTickerPriceChangeStats(currency.Pair{})
		if err != nil {
			return nil, err
		}

		for y := range tick {
			cp, err := currency.NewPairFromString(tick[y].Symbol)
			if err != nil {
				return nil, err
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
				AssetType:    assetType,
			})
			if err != nil {
				return nil, err
			}
		}
	case asset.CoinMarginedFutures:
		tick, err := b.GetFuturesSwapTickerChangeStats(currency.Pair{}, "")
		if err != nil {
			return nil, err
		}

		for y := range tick {
			cp, err := currency.NewPairFromString(tick[y].Symbol)
			if err != nil {
				return nil, err
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
				AssetType:    assetType,
			})
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("assetType not supported: %v", assetType)
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Binance) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Binance) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Binance) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}
	var orderbookNew OrderBook
	var err error
	switch assetType {
	case asset.Spot, asset.Margin:
		orderbookNew, err = b.GetOrderBook(OrderBookDataRequestParams{
			Symbol: p,
			Limit:  1000})
	case asset.USDTMarginedFutures:
		orderbookNew, err = b.UFuturesOrderbook(p, 1000)
	case asset.CoinMarginedFutures:
		orderbookNew, err = b.GetFuturesOrderbook(p, 1000)
	}
	if err != nil {
		return book, err
	}
	for x := range orderbookNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Quantity,
			Price:  orderbookNew.Bids[x].Price,
		})
	}
	for x := range orderbookNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Quantity,
			Price:  orderbookNew.Asks[x].Price,
		})
	}

	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Binance exchange
func (b *Binance) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = b.Name
	switch assetType {
	case asset.Spot:
		raw, err := b.GetAccount()
		if err != nil {
			return info, err
		}

		var currencyBalance []account.Balance
		for i := range raw.Balances {
			freeCurrency, parseErr := strconv.ParseFloat(raw.Balances[i].Free, 64)
			if parseErr != nil {
				return info, parseErr
			}

			lockedCurrency, parseErr := strconv.ParseFloat(raw.Balances[i].Locked, 64)
			if parseErr != nil {
				return info, parseErr
			}

			currencyBalance = append(currencyBalance, account.Balance{
				CurrencyName: currency.NewCode(raw.Balances[i].Asset),
				TotalValue:   freeCurrency + lockedCurrency,
				Hold:         freeCurrency,
			})
		}

		acc.Currencies = currencyBalance

	case asset.CoinMarginedFutures:
		accData, err := b.GetFuturesAccountInfo()
		if err != nil {
			return info, err
		}
		var currencyDetails []account.Balance
		for i := range accData.Assets {
			currencyDetails = append(currencyDetails, account.Balance{
				CurrencyName: currency.NewCode(accData.Assets[i].Asset),
				TotalValue:   accData.Assets[i].WalletBalance,
				Hold:         accData.Assets[i].WalletBalance - accData.Assets[i].MarginBalance,
			})
		}

		acc.Currencies = currencyDetails

	case asset.USDTMarginedFutures:
		accData, err := b.UAccountBalanceV2()
		if err != nil {
			return info, err
		}
		var currencyDetails []account.Balance
		for i := range accData {
			currencyDetails = append(currencyDetails, account.Balance{
				CurrencyName: currency.NewCode(accData[i].Asset),
				TotalValue:   accData[i].Balance,
				Hold:         accData[i].Balance - accData[i].AvailableBalance,
			})
		}

		acc.Currencies = currencyDetails
	case asset.Margin:
		accData, err := b.GetMarginAccount()
		if err != nil {
			return info, err
		}
		var currencyDetails []account.Balance
		for i := range accData.UserAssets {
			currencyDetails = append(currencyDetails, account.Balance{
				CurrencyName: currency.NewCode(accData.UserAssets[i].Asset),
				TotalValue:   accData.UserAssets[i].Free + accData.UserAssets[i].Locked,
				Hold:         accData.UserAssets[i].Locked,
			})
		}

		acc.Currencies = currencyDetails

	default:
		return info, fmt.Errorf("%v assetType not supported", assetType)
	}
	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	err := account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Binance) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name, assetType)
	if err != nil {
		return b.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Binance) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Binance) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	w, err := b.WithdrawStatus(c, "", 0, 0)
	if err != nil {
		return nil, err
	}

	for i := range w {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          strconv.FormatInt(w[i].Status, 10),
			TransferID:      w[i].ID,
			Currency:        w[i].Asset,
			Amount:          w[i].Amount,
			Fee:             w[i].TransactionFee,
			CryptoToAddress: w[i].Address,
			CryptoTxID:      w[i].TxID,
			Timestamp:       time.Unix(w[i].ApplyTime/1000, 0),
		})
	}

	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Binance) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	limit := 1000
	tradeData, err := b.GetMostRecentTrades(RecentTradeRequestParams{p, limit})
	if err != nil {
		return nil, err
	}
	for i := range tradeData {
		resp = append(resp, trade.Data{
			TID:          strconv.FormatInt(tradeData[i].ID, 10),
			Exchange:     b.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Quantity,
			Timestamp:    tradeData[i].Time,
		})
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
func (b *Binance) GetHistoricTrades(p currency.Pair, a asset.Item, from, to time.Time) ([]trade.Data, error) {
	req := AggregatedTradeRequestParams{
		Symbol:    p,
		StartTime: from,
		EndTime:   to,
	}
	trades, err := b.GetAggregatedTrades(&req)
	if err != nil {
		return nil, err
	}
	var result []trade.Data
	exName := b.GetName()
	for i := range trades {
		t := trades[i].toTradeData(p, exName, a)
		result = append(result, *t)
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
func (b *Binance) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
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
			requestParamsOrderType = BinanceRequestParamsOrderLimit
		default:
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, errors.New("unsupported order type")
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
		response, err := b.NewOrder(&orderRequest)
		if err != nil {
			return submitOrderResponse, err
		}

		if response.OrderID > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response.OrderID, 10)
		}
		if response.ExecutedQty == response.OrigQty {
			submitOrderResponse.FullyMatched = true
		}
		submitOrderResponse.IsOrderPlaced = true

		for i := range response.Fills {
			submitOrderResponse.Trades = append(submitOrderResponse.Trades, order.TradeHistory{
				Price:    response.Fills[i].Price,
				Amount:   response.Fills[i].Qty,
				Fee:      response.Fills[i].Commission,
				FeeAsset: response.Fills[i].CommissionAsset,
			})
		}

	case asset.CoinMarginedFutures:
		var reqSide string
		switch s.Side {
		case order.Buy:
			reqSide = "BUY"
		case order.Sell:
			reqSide = "SELL"
		default:
			return submitOrderResponse, fmt.Errorf("invalid side")
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
			return submitOrderResponse, errors.New("invalid type, check api docs for updates")
		}
		o, err := b.FuturesNewOrder(s.Pair, reqSide,
			"", oType, "GTC", "",
			s.ClientOrderID, "", "",
			s.Amount, s.Price, 0, 0, 0, s.ReduceOnly)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = strconv.FormatInt(o.OrderID, 10)
		submitOrderResponse.IsOrderPlaced = true
	case asset.USDTMarginedFutures:
		var reqSide string
		switch s.Side {
		case order.Buy:
			reqSide = "BUY"
		case order.Sell:
			reqSide = "SELL"
		default:
			return submitOrderResponse, fmt.Errorf("invalid side")
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
			return submitOrderResponse, errors.New("invalid type, check api docs for updates")
		}
		order, err := b.UFuturesNewOrder(s.Pair, reqSide,
			"", oType, "GTC", "",
			s.ClientOrderID, "", "",
			s.Amount, s.Price, 0, 0, 0, s.ReduceOnly)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = strconv.FormatInt(order.OrderID, 10)
		submitOrderResponse.IsOrderPlaced = true
	default:
		return submitOrderResponse, fmt.Errorf("assetType not supported")
	}

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Binance) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Binance) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot, asset.Margin:
		orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
		if err != nil {
			return err
		}
		_, err = b.CancelExistingOrder(o.Pair,
			orderIDInt,
			o.AccountID)
		if err != nil {
			return err
		}
	case asset.CoinMarginedFutures:
		_, err := b.FuturesCancelOrder(o.Pair, o.ID, "")
		if err != nil {
			return err
		}
	case asset.USDTMarginedFutures:
		_, err := b.UCancelOrder(o.Pair, o.ID, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Binance) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Binance) CancelAllOrders(req *order.Cancel) (order.CancelAllResponse, error) {
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch req.AssetType {
	case asset.Spot, asset.Margin:
		openOrders, err := b.OpenOrders(req.Pair)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range openOrders {
			_, err = b.CancelExistingOrder(req.Pair,
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
				_, err = b.FuturesCancelAllOpenOrders(enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
			}
		} else {
			_, err := b.FuturesCancelAllOpenOrders(req.Pair)
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
				_, err = b.UCancelAllOpenOrders(enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
			}
		} else {
			_, err := b.UCancelAllOpenOrders(req.Pair)
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
func (b *Binance) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var respData order.Detail
	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return respData, err
	}
	switch assetType {
	case asset.Spot:
		resp, err := b.QueryOrder(pair, "", orderIDInt)
		if err != nil {
			return respData, err
		}
		orderSide := order.Side(resp.Side)
		status, err := order.StringToOrderStatus(resp.Status)
		if err != nil {
			return respData, err
		}
		orderType := order.Limit
		if resp.Type == "MARKET" {
			orderType = order.Market
		}

		return order.Detail{
			Amount:         resp.OrigQty,
			Exchange:       b.Name,
			ID:             strconv.FormatInt(resp.OrderID, 10),
			ClientOrderID:  resp.ClientOrderID,
			Side:           orderSide,
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
		orderData, err := b.FuturesOpenOrderData(pair, orderID, "")
		if err != nil {
			return respData, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := b.GetFee(&feeBuilder)
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
		respData.ID = orderID
		respData.Pair = pair
		respData.Price = orderData.Price
		respData.RemainingAmount = orderData.OriginalQuantity - orderData.ExecutedQuantity
		respData.Side = orderVars.Side
		respData.Status = orderVars.Status
		respData.Type = orderVars.OrderType
		respData.Date = orderData.Time
		respData.LastUpdated = orderData.UpdateTime
	case asset.USDTMarginedFutures:
		orderData, err := b.UGetOrderData(pair, orderID, "")
		if err != nil {
			return respData, err
		}
		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData.ExecutedQuantity
		feeBuilder.PurchasePrice = orderData.AveragePrice
		feeBuilder.Pair = pair
		fee, err := b.GetFee(&feeBuilder)
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
		respData.ID = orderID
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
func (b *Binance) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return b.GetDepositAddressForCurrency(cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Binance) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	amountStr := strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64)
	v, err := b.WithdrawCrypto(withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Description, amountStr)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Binance) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!b.AllowAuthenticatedRequest() || b.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Binance) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 || len(req.Pairs) >= 40 {
		// sending an empty currency pair retrieves data for all currencies
		req.Pairs = append(req.Pairs, currency.Pair{})
	}
	var orders []order.Detail
	for i := range req.Pairs {
		switch req.AssetType {
		case asset.Spot, asset.Margin:
			resp, err := b.OpenOrders(req.Pairs[i])
			if err != nil {
				return nil, err
			}
			for x := range resp {
				orderSide := order.Side(strings.ToUpper(resp[x].Side))
				orderType := order.Type(strings.ToUpper(resp[x].Type))
				orders = append(orders, order.Detail{
					Amount:        resp[x].OrigQty,
					Date:          resp[x].Time,
					Exchange:      b.Name,
					ID:            strconv.FormatInt(resp[x].OrderID, 10),
					ClientOrderID: resp[x].ClientOrderID,
					Side:          orderSide,
					Type:          orderType,
					Price:         resp[x].Price,
					Status:        order.Status(resp[x].Status),
					Pair:          req.Pairs[i],
					AssetType:     req.AssetType,
					LastUpdated:   resp[x].UpdateTime,
				})
			}
		case asset.CoinMarginedFutures:
			openOrders, err := b.GetFuturesAllOpenOrders(req.Pairs[i], "")
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = openOrders[y].ExecutedQty
				feeBuilder.PurchasePrice = openOrders[y].AvgPrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := b.GetFee(&feeBuilder)
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
					ID:              strconv.FormatInt(openOrders[y].OrderID, 10),
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
			openOrders, err := b.UAllAccountOpenOrders(req.Pairs[i])
			if err != nil {
				return nil, err
			}
			for y := range openOrders {
				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = openOrders[y].ExecutedQuantity
				feeBuilder.PurchasePrice = openOrders[y].AveragePrice
				feeBuilder.Pair = req.Pairs[i]
				fee, err := b.GetFee(&feeBuilder)
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
					ID:              strconv.FormatInt(openOrders[y].OrderID, 10),
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
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Binance) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot, asset.Margin:
		for x := range req.Pairs {
			resp, err := b.AllOrders(req.Pairs[x],
				"",
				"1000")
			if err != nil {
				return nil, err
			}

			for i := range resp {
				orderSide := order.Side(strings.ToUpper(resp[i].Side))
				orderType := order.Type(strings.ToUpper(resp[i].Type))
				// New orders are covered in GetOpenOrders
				if resp[i].Status == "NEW" {
					continue
				}

				pair, err := currency.NewPairFromString(resp[i].Symbol)
				if err != nil {
					return nil, err
				}
				orders = append(orders, order.Detail{
					Amount:   resp[i].OrigQty,
					Date:     resp[i].Time,
					Exchange: b.Name,
					ID:       strconv.FormatInt(resp[i].OrderID, 10),
					Side:     orderSide,
					Type:     orderType,
					Price:    resp[i].Price,
					Pair:     pair,
					Status:   order.Status(resp[i].Status),
				})
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
				orderHistory, err = b.GetAllFuturesOrders(req.Pairs[i], "", req.StartTime, req.EndTime, 0, 0)
				if err != nil {
					return nil, err
				}
			case req.OrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.OrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = b.GetAllFuturesOrders(req.Pairs[i], "", time.Time{}, time.Time{}, fromID, 0)
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
				fee, err := b.GetFee(&feeBuilder)
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
					ID:              strconv.FormatInt(orderHistory[y].OrderID, 10),
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
				orderHistory, err = b.UAllAccountOrders(req.Pairs[i], 0, 0, req.StartTime, req.EndTime)
				if err != nil {
					return nil, err
				}
			case req.OrderID != "" && req.StartTime.IsZero() && req.EndTime.IsZero():
				fromID, err := strconv.ParseInt(req.OrderID, 10, 64)
				if err != nil {
					return nil, err
				}
				orderHistory, err = b.UAllAccountOrders(req.Pairs[i], fromID, 0, time.Time{}, time.Time{})
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
				fee, err := b.GetFee(&feeBuilder)
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
					ID:              strconv.FormatInt(orderHistory[y].OrderID, 10),
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
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Binance) ValidateCredentials(assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(assetType)
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
func (b *Binance) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := b.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	if kline.TotalCandlesPerInterval(start, end, interval) > float64(b.Features.Enabled.Kline.ResultLimit) {
		return kline.Item{}, errors.New(kline.ErrRequestExceedsExchangeLimits)
	}
	req := KlinesRequestParams{
		Interval:  b.FormatExchangeKlineInterval(interval),
		Symbol:    pair,
		StartTime: start,
		EndTime:   end,
		Limit:     int(b.Features.Enabled.Kline.ResultLimit),
	}
	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	candles, err := b.GetSpotKline(&req)
	if err != nil {
		return kline.Item{}, err
	}
	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		})
	}
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *Binance) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := b.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, b.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	var candles []CandleStick
	for x := range dates.Ranges {
		req := KlinesRequestParams{
			Interval:  b.FormatExchangeKlineInterval(interval),
			Symbol:    pair,
			StartTime: dates.Ranges[x].Start.Time,
			EndTime:   dates.Ranges[x].End.Time,
			Limit:     int(b.Features.Enabled.Kline.ResultLimit),
		}

		candles, err = b.GetSpotKline(&req)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range candles {
			for j := range ret.Candles {
				if ret.Candles[j].Time.Equal(candles[i].OpenTime) {
					continue
				}
			}
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   candles[i].OpenTime,
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	}

	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", b.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
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
func (b *Binance) UpdateOrderExecutionLimits(a asset.Item) error {
	var limits []order.MinMaxLevel
	var err error
	switch a {
	case asset.Spot:
		limits, err = b.FetchSpotExchangeLimits()
	case asset.USDTMarginedFutures:
		limits, err = b.FetchUSDTMarginExchangeLimits()
	case asset.CoinMarginedFutures:
		limits, err = b.FetchCoinMarginExchangeLimits()
	case asset.Margin:
		if err = b.CurrencyPairs.IsAssetEnabled(asset.Spot); err != nil {
			limits, err = b.FetchSpotExchangeLimits()
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
