package huobi

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
func (h *HUOBI) GetDefaultConfig() (*config.ExchangeConfig, error) {
	h.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = h.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = h.BaseCurrencies

	err := h.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if h.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = h.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (h *HUOBI) SetDefaults() {
	h.Name = "Huobi"
	h.Enabled = true
	h.Verbose = true
	h.API.CredentialsValidator.RequiresKey = true
	h.API.CredentialsValidator.RequiresSecret = true

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: false},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}
	coinFutures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}
	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}
	err := h.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = h.StoreAssetPairFormat(asset.CoinMarginedFutures, coinFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = h.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	h.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				KlineFetching:     true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
				TradeFee:          true,
			},
			WebsocketCapabilities: protocol.Features{
				KlineFetching:          true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				MessageCorrelation:     true,
				GetOrder:               true,
				GetOrders:              true,
				TickerFetching:         true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.OneDay.Word():     true,
					kline.OneWeek.Word():    true,
					kline.OneMonth.Word():   true,
					kline.OneYear.Word():    true,
				},
				ResultLimit: 2000,
			},
		},
	}

	h.Requester = request.New(h.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	h.API.Endpoints = h.NewEndpoints()
	err = h.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:         huobiAPIURL,
		exchange.RestFutures:      huobiURL,
		exchange.RestCoinMargined: huobiFuturesURL,
		exchange.WebsocketSpot:    wsMarketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	h.Websocket = stream.New()
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user configuration
func (h *HUOBI) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		h.SetEnabled(false)
		return nil
	}

	err := h.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := h.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = h.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       wsMarketURL,
		ExchangeName:                     exch.Name,
		RunningURL:                       wsRunningURL,
		Connector:                        h.WsConnect,
		Subscriber:                       h.Subscribe,
		UnSubscriber:                     h.Unsubscribe,
		GenerateSubscriptions:            h.GenerateDefaultSubscriptions,
		Features:                         &h.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.OrderbookConfig.WebsocketBufferLimit,
		BufferEnabled:                    exch.OrderbookConfig.WebsocketBufferEnabled,
	})
	if err != nil {
		return err
	}

	err = h.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}

	return h.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  wsAccountsOrdersURL,
		Authenticated:        true,
	})
}

// Start starts the HUOBI go routine
func (h *HUOBI) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
}

// Run implements the HUOBI wrapper
func (h *HUOBI) Run() {
	if h.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s (url: %s).\n",
			h.Name,
			common.IsEnabled(h.Websocket.IsEnabled()),
			wsMarketURL)
		h.PrintEnabledPairs()
	}

	var forceUpdate bool
	enabled, err := h.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s Failed to update enabled currencies. Err:%s\n",
			h.Name,
			err)
	}

	avail, err := h.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s Failed to update enabled currencies. Err:%s\n",
			h.Name,
			err)
	}

	if common.StringDataContains(enabled.Strings(), currency.CNY.String()) ||
		common.StringDataContains(avail.Strings(), currency.CNY.String()) {
		forceUpdate = true
	}

	if common.StringDataContains(h.BaseCurrencies.Strings(), currency.CNY.String()) {
		cfg := config.GetConfig()
		var exchCfg *config.ExchangeConfig
		exchCfg, err = cfg.GetExchangeConfig(h.Name)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to get exchange config. %s\n",
				h.Name,
				err)
			return
		}
		exchCfg.BaseCurrencies = currency.Currencies{currency.USD}
		h.BaseCurrencies = currency.Currencies{currency.USD}
	}

	if forceUpdate {
		var format currency.PairFormat
		format, err = h.GetPairFormat(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to get exchange config. %s\n",
				h.Name,
				err)
			return
		}
		enabledPairs := currency.Pairs{
			currency.Pair{
				Base:      currency.BTC.Lower(),
				Quote:     currency.USDT.Lower(),
				Delimiter: format.Delimiter,
			},
		}
		log.Warn(log.ExchangeSys,
			"Available and enabled pairs for Huobi reset due to config upgrade, please enable the ones you would like again")

		err = h.UpdatePairs(enabledPairs, asset.Spot, true, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s Failed to update enabled currencies. Err:%s\n",
				h.Name,
				err)
		}
	}

	if !h.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = h.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			h.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (h *HUOBI) FetchTradablePairs(a asset.Item) ([]string, error) {
	if !h.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, h.Name)
	}

	var pairs []string

	switch a {
	case asset.Spot:
		symbols, err := h.GetSymbols()
		if err != nil {
			return nil, err
		}

		format, err := h.GetPairFormat(a, false)
		if err != nil {
			return nil, err
		}

		for x := range symbols {
			if symbols[x].State != "online" {
				continue
			}
			pairs = append(pairs, symbols[x].BaseCurrency+
				format.Delimiter+
				symbols[x].QuoteCurrency)
		}

	case asset.CoinMarginedFutures:
		symbols, err := h.GetSwapMarkets(currency.Pair{})
		if err != nil {
			return nil, err
		}

		for z := range symbols {
			if symbols[z].ContractStatus == 1 {
				pairs = append(pairs, symbols[z].ContractCode)
			}
		}

	case asset.Futures:
		symbols, err := h.FGetContractInfo("", "", currency.Pair{})
		if err != nil {
			return nil, err
		}

		for c := range symbols.Data {
			if symbols.Data[c].ContractStatus == 1 {
				pairs = append(pairs, symbols.Data[c].ContractCode)
			}
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (h *HUOBI) UpdateTradablePairs(forceUpdate bool) error {
	spotPairs, err := h.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}
	p, err := currency.NewPairsFromStrings(spotPairs)
	if err != nil {
		return err
	}
	err = h.UpdatePairs(p, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}

	futuresPairs, err := h.FetchTradablePairs(asset.Futures)
	if err != nil {
		return err
	}
	fp, err := currency.NewPairsFromStrings(futuresPairs)
	if err != nil {
		return err
	}
	err = h.UpdatePairs(fp, asset.Futures, false, forceUpdate)
	if err != nil {
		return err
	}

	coinmarginedFuturesPairs, err := h.FetchTradablePairs(asset.CoinMarginedFutures)
	if err != nil {
		return err
	}
	cp, err := currency.NewPairsFromStrings(coinmarginedFuturesPairs)
	if err != nil {
		return err
	}
	return h.UpdatePairs(cp, asset.CoinMarginedFutures, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HUOBI) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if !h.SupportsAsset(assetType) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", assetType, h.Name)
	}
	switch assetType {
	case asset.Spot:
		tickerData, err := h.Get24HrMarketSummary(p)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			High:         tickerData.Tick.High,
			Low:          tickerData.Tick.Low,
			Volume:       tickerData.Tick.Volume,
			Open:         tickerData.Tick.Open,
			Close:        tickerData.Tick.Close,
			Pair:         p,
			ExchangeName: h.Name,
			AssetType:    asset.Spot,
		})
		if err != nil {
			return nil, err
		}
	case asset.CoinMarginedFutures:
		marketData, err := h.GetSwapMarketOverview(p)
		if err != nil {
			return nil, err
		}

		if len(marketData.Tick.Bid) == 0 {
			return nil, fmt.Errorf("invalid data for bid")
		}
		if len(marketData.Tick.Ask) == 0 {
			return nil, fmt.Errorf("invalid data for Ask")
		}

		err = ticker.ProcessTicker(&ticker.Price{
			High:         marketData.Tick.High,
			Low:          marketData.Tick.Low,
			Volume:       marketData.Tick.Vol,
			Open:         marketData.Tick.Open,
			Close:        marketData.Tick.Close,
			Pair:         p,
			Bid:          marketData.Tick.Bid[0],
			Ask:          marketData.Tick.Ask[0],
			ExchangeName: h.Name,
			AssetType:    assetType,
		})
		if err != nil {
			return nil, err
		}
	case asset.Futures:
		marketData, err := h.FGetMarketOverviewData(p)
		if err != nil {
			return nil, err
		}

		err = ticker.ProcessTicker(&ticker.Price{
			High:         marketData.Tick.High,
			Low:          marketData.Tick.Low,
			Volume:       marketData.Tick.Vol,
			Open:         marketData.Tick.Open,
			Close:        marketData.Tick.Close,
			Pair:         p,
			Bid:          marketData.Tick.Bid[0],
			Ask:          marketData.Tick.Ask[0],
			ExchangeName: h.Name,
			AssetType:    assetType,
		})
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(h.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (h *HUOBI) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.Name, p, assetType)
	if err != nil {
		return h.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (h *HUOBI) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(h.Name, p, assetType)
	if err != nil {
		return h.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HUOBI) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        h.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: h.CanVerifyOrderbook,
	}
	var err error
	switch assetType {
	case asset.Spot:
		var orderbookNew Orderbook
		orderbookNew, err = h.GetDepth(OrderBookDataRequestParams{
			Symbol: p,
			Type:   OrderBookDataRequestParamsTypeStep0,
		})
		if err != nil {
			return nil, err
		}

		for x := range orderbookNew.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: orderbookNew.Bids[x][1],
				Price:  orderbookNew.Bids[x][0],
			})
		}

		for x := range orderbookNew.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: orderbookNew.Asks[x][1],
				Price:  orderbookNew.Asks[x][0],
			})
		}

	case asset.Futures:
		var orderbookNew OBData
		orderbookNew, err = h.FGetMarketDepth(p, "step0")
		if err != nil {
			return nil, err
		}

		for x := range orderbookNew.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: orderbookNew.Asks[x].Quantity,
				Price:  orderbookNew.Asks[x].Price,
			})
		}
		for y := range orderbookNew.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: orderbookNew.Bids[y].Quantity,
				Price:  orderbookNew.Bids[y].Price,
			})
		}

	case asset.CoinMarginedFutures:
		var orderbookNew SwapMarketDepthData
		orderbookNew, err = h.GetSwapMarketDepth(p, "step0")
		if err != nil {
			return nil, err
		}

		for x := range orderbookNew.Tick.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: orderbookNew.Tick.Asks[x][1],
				Price:  orderbookNew.Tick.Asks[x][0],
			})
		}
		for y := range orderbookNew.Tick.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: orderbookNew.Tick.Bids[y][1],
				Price:  orderbookNew.Tick.Bids[y][0],
			})
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(h.Name, p, assetType)
}

// GetAccountID returns the account ID for trades
func (h *HUOBI) GetAccountID() ([]Account, error) {
	acc, err := h.GetAccounts()
	if err != nil {
		return nil, err
	}

	if len(acc) < 1 {
		return nil, errors.New("no account returned")
	}

	return acc, nil
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// HUOBI exchange - to-do
func (h *HUOBI) UpdateAccountInfo(accountName string, assetType asset.Item) (account.HoldingsSnapshot, error) {
	m := make(account.HoldingsSnapshot)
	switch assetType {
	case asset.Spot:
		if h.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			resp, err := h.wsGetAccountsList()
			if err != nil {
				return nil, err
			}
			for i := range resp.Data {
				if len(resp.Data[i].List) == 0 {
					continue
				}

				var locked float64
				if len(resp.Data[i].List) > 1 && resp.Data[i].List[1].Type == "frozen" {
					locked = resp.Data[i].List[1].Balance
				}

				m[currency.NewCode(resp.Data[i].List[0].Currency)] = account.Balance{
					Total:  resp.Data[i].List[0].Balance,
					Locked: locked,
				}
			}

			// TODO: Add in ID
			err = h.LoadHoldings(account.Default, assetType, m)
			if err != nil {
				return nil, err
			}
		} else {
			accounts, err := h.GetAccountID()
			if err != nil {
				return nil, err
			}
			for x := range accounts {
				m := make(account.HoldingsSnapshot)
				accountID := strconv.FormatInt(accounts[x].ID, 10)
				balances, err := h.GetAccountBalance(accountID)
				if err != nil {
					return nil, err
				}

				for y := range balances {
					code := currency.NewCode(balances[y].Currency)
					if balances[y].Type == "frozen" {
						bal, ok := m[code]
						if !ok {
							m[code] = account.Balance{Locked: balances[y].Balance}
							continue
						}
						bal.Locked = balances[y].Balance
						m[code] = bal
						continue
					}
					bal, ok := m[code]
					if !ok {
						m[code] = account.Balance{Total: balances[y].Balance}
						continue
					}
					bal.Total = balances[y].Balance
					m[code] = bal
				}

				err = h.LoadHoldings(accountID, assetType, m)
				if err != nil {
					return nil, err
				}
			}
		}
	case asset.CoinMarginedFutures:
		cmfSubAccsData, err := h.GetSwapAllSubAccAssets(currency.Pair{})
		if err != nil {
			return nil, err
		}

		for x := range cmfSubAccsData.Data {
			m := make(account.HoldingsSnapshot)
			swapAcc, err := h.SwapSingleSubAccAssets(currency.Pair{}, cmfSubAccsData.Data[x].SubUID)
			if err != nil {
				return nil, err
			}
			for y := range swapAcc.Data {
				m[currency.NewCode(swapAcc.Data[y].Symbol)] = account.Balance{
					Total:  swapAcc.Data[y].MarginBalance + swapAcc.Data[y].MarginFrozen,
					Locked: swapAcc.Data[y].MarginFrozen,
				}
			}

			subAccID := strconv.FormatInt(cmfSubAccsData.Data[x].SubUID, 10)
			err = h.LoadHoldings(subAccID, asset.CoinMarginedFutures, m)
			if err != nil {
				return nil, err
			}
		}
	case asset.Futures:
		fsubAccsData, err := h.FGetAllSubAccountAssets(currency.Code{})
		if err != nil {
			return nil, err
		}

		for x := range fsubAccsData.Data {
			m := make(account.HoldingsSnapshot)
			subAccID := strconv.FormatInt(fsubAccsData.Data[x].SubUID, 10)
			subAcc, err := h.FGetSingleSubAccountInfo("", subAccID)
			if err != nil {
				return nil, err
			}
			for y := range subAcc.AssetsData {
				m[currency.NewCode(subAcc.AssetsData[y].Symbol)] = account.Balance{
					Total:  subAcc.AssetsData[y].MarginBalance + subAcc.AssetsData[y].MarginFrozen,
					Locked: subAcc.AssetsData[y].MarginFrozen,
				}
			}
			err = h.LoadHoldings(subAccID, asset.Futures, m)
			if err != nil {
				return nil, err
			}
		}
	}
	return h.GetHoldingsSnapshot(accountName, assetType)
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (h *HUOBI) FetchAccountInfo(accountName string, assetType asset.Item) (account.HoldingsSnapshot, error) {
	acc, err := h.GetHoldingsSnapshot(accountName, assetType)
	if err != nil {
		return h.UpdateAccountInfo(accountName, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HUOBI) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (h *HUOBI) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (h *HUOBI) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	var tradeData []TradeHistory
	tradeData, err = h.GetTradeHistory(p, 2000)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData {
		for j := range tradeData[i].Trades {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Trades[j].Direction)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     h.Name,
				TID:          strconv.FormatFloat(tradeData[i].Trades[j].TradeID, 'f', -1, 64),
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Trades[j].Price,
				Amount:       tradeData[i].Trades[j].Amount,
				Timestamp:    time.Unix(0, tradeData[i].Timestamp*int64(time.Millisecond)),
			})
		}
	}

	err = h.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (h *HUOBI) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (h *HUOBI) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	switch s.AssetType {
	case asset.Spot:
		accountID, err := strconv.ParseInt(s.ClientID, 10, 64)
		if err != nil {
			return submitOrderResponse, err
		}
		var formattedType SpotNewOrderRequestParamsType
		var params = SpotNewOrderRequestParams{
			Amount:    s.Amount,
			Source:    "api",
			Symbol:    s.Pair,
			AccountID: int(accountID),
		}
		switch {
		case s.Side == order.Buy && s.Type == order.Market:
			formattedType = SpotNewOrderRequestTypeBuyMarket
		case s.Side == order.Sell && s.Type == order.Market:
			formattedType = SpotNewOrderRequestTypeSellMarket
		case s.Side == order.Buy && s.Type == order.Limit:
			formattedType = SpotNewOrderRequestTypeBuyLimit
			params.Price = s.Price
		case s.Side == order.Sell && s.Type == order.Limit:
			formattedType = SpotNewOrderRequestTypeSellLimit
			params.Price = s.Price
		}
		params.Type = formattedType
		response, err := h.SpotNewOrder(&params)
		if err != nil {
			return submitOrderResponse, err
		}
		if response > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response, 10)
		}
		submitOrderResponse.IsOrderPlaced = true
		if s.Type == order.Market {
			submitOrderResponse.FullyMatched = true
		}
	case asset.CoinMarginedFutures:
		var oDirection string
		switch s.Side {
		case order.Buy:
			oDirection = "BUY"
		case order.Sell:
			oDirection = "SELL"
		}
		var oType string
		switch s.Type {
		case order.Limit:
			oType = "limit"
		case order.PostOnly:
			oType = "post_only"
		}
		order, err := h.PlaceSwapOrders(s.Pair, s.ClientOrderID, oDirection, s.Offset, oType, s.Price, s.Amount, s.Leverage)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = order.Data.OrderIDString
		submitOrderResponse.IsOrderPlaced = true
	case asset.Futures:
		var oDirection string
		switch s.Side {
		case order.Buy:
			oDirection = "BUY"
		case order.Sell:
			oDirection = "SELL"
		}
		var oType string
		switch s.Type {
		case order.Limit:
			oType = "limit"
		case order.PostOnly:
			oType = "post_only"
		}
		order, err := h.FOrder(s.Pair, "", "", s.ClientOrderID, oDirection, s.Offset, oType, s.Price, s.Amount, s.Leverage)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = order.Data.OrderIDStr
		submitOrderResponse.IsOrderPlaced = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (h *HUOBI) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HUOBI) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	var err error
	switch o.AssetType {
	case asset.Spot:
		var orderIDInt int64
		orderIDInt, err = strconv.ParseInt(o.ID, 10, 64)
		if err != nil {
			return err
		}
		_, err = h.CancelExistingOrder(orderIDInt)
	case asset.CoinMarginedFutures:
		_, err = h.CancelSwapOrder(o.ID, o.ClientID, o.Pair)
	case asset.Futures:
		_, err = h.FCancelOrder(o.Symbol, o.ClientID, o.ClientOrderID)
	default:
		return fmt.Errorf("%v assetType not supported", o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (h *HUOBI) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HUOBI) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelAllOrdersResponse order.CancelAllResponse
	cancelAllOrdersResponse.Status = make(map[string]string)
	switch orderCancellation.AssetType {
	case asset.Spot:
		enabledPairs, err := h.GetEnabledPairs(asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range enabledPairs {
			resp, err := h.CancelOpenOrdersBatch(orderCancellation.AccountID,
				enabledPairs[i])
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if resp.Data.FailedCount > 0 {
				return cancelAllOrdersResponse,
					fmt.Errorf("%v orders failed to cancel",
						resp.Data.FailedCount)
			}
			if resp.Status == "error" {
				return cancelAllOrdersResponse, errors.New(resp.ErrorMessage)
			}
		}
	case asset.CoinMarginedFutures:
		if orderCancellation.Pair.IsEmpty() {
			enabledPairs, err := h.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				a, err := h.CancelAllSwapOrders(enabledPairs[i])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				split := strings.Split(a.Successes, ",")
				for x := range split {
					cancelAllOrdersResponse.Status[split[x]] = "success"
				}
				for y := range a.Errors {
					cancelAllOrdersResponse.Status[a.Errors[y].OrderID] = fmt.Sprintf("fail: %s", a.Errors[y].ErrMsg)
				}
			}
		} else {
			a, err := h.CancelAllSwapOrders(orderCancellation.Pair)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			split := strings.Split(a.Successes, ",")
			for x := range split {
				cancelAllOrdersResponse.Status[split[x]] = "success"
			}
			for y := range a.Errors {
				cancelAllOrdersResponse.Status[a.Errors[y].OrderID] = fmt.Sprintf("fail: %s", a.Errors[y].ErrMsg)
			}
		}
	case asset.Futures:
		if orderCancellation.Pair.IsEmpty() {
			enabledPairs, err := h.GetEnabledPairs(asset.Futures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for i := range enabledPairs {
				a, err := h.FCancelAllOrders(enabledPairs[i], "", "")
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				split := strings.Split(a.Data.Successes, ",")
				for x := range split {
					cancelAllOrdersResponse.Status[split[x]] = "success"
				}
				for y := range a.Data.Errors {
					cancelAllOrdersResponse.Status[strconv.FormatInt(a.Data.Errors[y].OrderID, 10)] = fmt.Sprintf("fail: %s", a.Data.Errors[y].ErrMsg)
				}
			}
		} else {
			a, err := h.FCancelAllOrders(orderCancellation.Pair, "", "")
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			split := strings.Split(a.Data.Successes, ",")
			for x := range split {
				cancelAllOrdersResponse.Status[split[x]] = "success"
			}
			for y := range a.Data.Errors {
				cancelAllOrdersResponse.Status[strconv.FormatInt(a.Data.Errors[y].OrderID, 10)] = fmt.Sprintf("fail: %s", a.Data.Errors[y].ErrMsg)
			}
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (h *HUOBI) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	switch assetType {
	case asset.Spot:
		var respData *OrderInfo
		if h.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			resp, err := h.wsGetOrderDetails(orderID)
			if err != nil {
				return orderDetail, err
			}
			respData = &resp.Data
		} else {
			oID, err := strconv.ParseInt(orderID, 10, 64)
			if err != nil {
				return orderDetail, err
			}
			resp, err := h.GetOrder(oID)
			if err != nil {
				return orderDetail, err
			}
			respData = &resp
		}
		if respData.ID == 0 {
			return orderDetail, fmt.Errorf("%s - order not found for orderid %s", h.Name, orderID)
		}
		var responseID = strconv.FormatInt(respData.ID, 10)
		if responseID != orderID {
			return orderDetail, errors.New(h.Name + " - GetOrderInfo orderID mismatch. Expected: " +
				orderID + " Received: " + responseID)
		}
		typeDetails := strings.Split(respData.Type, "-")
		orderSide, err := order.StringToOrderSide(typeDetails[0])
		if err != nil {
			if h.Websocket.IsConnected() {
				h.Websocket.DataHandler <- order.ClassificationError{
					Exchange: h.Name,
					OrderID:  orderID,
					Err:      err,
				}
			} else {
				return orderDetail, err
			}
		}
		orderType, err := order.StringToOrderType(typeDetails[1])
		if err != nil {
			if h.Websocket.IsConnected() {
				h.Websocket.DataHandler <- order.ClassificationError{
					Exchange: h.Name,
					OrderID:  orderID,
					Err:      err,
				}
			} else {
				return orderDetail, err
			}
		}
		orderStatus, err := order.StringToOrderStatus(respData.State)
		if err != nil {
			if h.Websocket.IsConnected() {
				h.Websocket.DataHandler <- order.ClassificationError{
					Exchange: h.Name,
					OrderID:  orderID,
					Err:      err,
				}
			} else {
				return orderDetail, err
			}
		}
		var p currency.Pair
		var a asset.Item
		p, a, err = h.GetRequestFormattedPairAndAssetType(respData.Symbol)
		if err != nil {
			return orderDetail, err
		}
		orderDetail = order.Detail{
			Exchange:       h.Name,
			ID:             orderID,
			AccountID:      strconv.FormatInt(respData.AccountID, 10),
			Pair:           p,
			Type:           orderType,
			Side:           orderSide,
			Date:           time.Unix(0, respData.CreatedAt*int64(time.Millisecond)),
			Status:         orderStatus,
			Price:          respData.Price,
			Amount:         respData.Amount,
			ExecutedAmount: respData.FilledAmount,
			Fee:            respData.FilledFees,
			AssetType:      a,
		}
	case asset.CoinMarginedFutures:
		orderInfo, err := h.GetSwapOrderInfo(pair, orderID, "")
		if err != nil {
			return orderDetail, err
		}
		var orderVars OrderVars
		for x := range orderInfo.Data {
			orderVars, err = compatibleVars(orderInfo.Data[x].Direction, orderInfo.Data[x].OrderPriceType, orderInfo.Data[x].Status)
			if err != nil {
				return orderDetail, err
			}
			maker := true
			if orderVars.OrderType == order.Limit || orderVars.OrderType == order.PostOnly {
				maker = false
			}
			orderDetail.Trades = append(orderDetail.Trades, order.TradeHistory{
				Price:    orderInfo.Data[x].Price,
				Amount:   orderInfo.Data[x].Volume,
				Fee:      orderInfo.Data[x].Fee,
				Exchange: h.Name,
				TID:      orderInfo.Data[x].OrderIDString,
				Type:     orderVars.OrderType,
				Side:     orderVars.Side,
				IsMaker:  maker,
			})
		}
	case asset.Futures:
		orderInfo, err := h.FGetOrderInfo("", orderID, "")
		if err != nil {
			return orderDetail, err
		}
		var orderVars OrderVars
		for x := range orderInfo.Data {
			orderVars, err = compatibleVars(orderInfo.Data[x].Direction, orderInfo.Data[x].OrderPriceType, orderInfo.Data[x].Status)
			if err != nil {
				return orderDetail, err
			}

			orderDetail.Trades = append(orderDetail.Trades, order.TradeHistory{
				Price:    orderInfo.Data[x].Price,
				Amount:   orderInfo.Data[x].Volume,
				Fee:      orderInfo.Data[x].Fee,
				Exchange: h.Name,
				TID:      orderInfo.Data[x].OrderIDString,
				Type:     orderVars.OrderType,
				Side:     orderVars.Side,
				IsMaker:  orderVars.OrderType == order.Limit || orderVars.OrderType == order.PostOnly,
			})
		}
	}
	return orderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HUOBI) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	resp, err := h.QueryDepositAddress(cryptocurrency.Lower().String())
	return resp.Address, err
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HUOBI) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := h.Withdraw(withdrawRequest.Currency,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Amount,
		withdrawRequest.Crypto.FeeAmount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp, 10),
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBI) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HUOBI) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HUOBI) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !h.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return h.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HUOBI) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		if len(req.Pairs) == 0 {
			return nil, errors.New("currency must be supplied")
		}
		side := ""
		if req.Side == order.AnySide || req.Side == "" {
			side = ""
		} else if req.Side == order.Sell {
			side = req.Side.Lower()
		}
		if h.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			for i := range req.Pairs {
				resp, err := h.wsGetOrdersList(-1, req.Pairs[i])
				if err != nil {
					return orders, err
				}
				for j := range resp.Data {
					sideData := strings.Split(resp.Data[j].OrderState, "-")
					side = sideData[0]
					var orderID = strconv.FormatInt(resp.Data[j].OrderID, 10)
					orderSide, err := order.StringToOrderSide(side)
					if err != nil {
						h.Websocket.DataHandler <- order.ClassificationError{
							Exchange: h.Name,
							OrderID:  orderID,
							Err:      err,
						}
					}
					orderType, err := order.StringToOrderType(sideData[1])
					if err != nil {
						h.Websocket.DataHandler <- order.ClassificationError{
							Exchange: h.Name,
							OrderID:  orderID,
							Err:      err,
						}
					}
					orderStatus, err := order.StringToOrderStatus(resp.Data[j].OrderState)
					if err != nil {
						h.Websocket.DataHandler <- order.ClassificationError{
							Exchange: h.Name,
							OrderID:  orderID,
							Err:      err,
						}
					}
					orders = append(orders, order.Detail{
						Exchange:        h.Name,
						AccountID:       strconv.FormatInt(resp.Data[j].AccountID, 10),
						ID:              orderID,
						Pair:            req.Pairs[i],
						Type:            orderType,
						Side:            orderSide,
						Date:            time.Unix(0, resp.Data[j].CreatedAt*int64(time.Millisecond)),
						Status:          orderStatus,
						Price:           resp.Data[j].Price,
						Amount:          resp.Data[j].OrderAmount,
						ExecutedAmount:  resp.Data[j].FilledAmount,
						RemainingAmount: resp.Data[j].UnfilledAmount,
						Fee:             resp.Data[j].FilledFees,
					})
				}
			}
		} else {
			for i := range req.Pairs {
				resp, err := h.GetOpenOrders(req.Pairs[i],
					h.API.Credentials.ClientID,
					side,
					500)
				if err != nil {
					return nil, err
				}
				for x := range resp {
					orderDetail := order.Detail{
						ID:             strconv.FormatInt(resp[x].ID, 10),
						Price:          resp[x].Price,
						Amount:         resp[x].Amount,
						Pair:           req.Pairs[i],
						Exchange:       h.Name,
						ExecutedAmount: resp[x].FilledAmount,
						Date:           time.Unix(0, resp[x].CreatedAt*int64(time.Millisecond)),
						Status:         order.Status(resp[x].State),
						AccountID:      strconv.FormatInt(resp[x].AccountID, 10),
						Fee:            resp[x].FilledFees,
					}
					setOrderSideAndType(resp[x].Type, &orderDetail)
					orders = append(orders, orderDetail)
				}
			}
		}
	case asset.CoinMarginedFutures:
		for x := range req.Pairs {
			var currentPage int64 = 0
			for done := false; !done; {
				openOrders, err := h.GetSwapOpenOrders(req.Pairs[x], currentPage, 50)
				if err != nil {
					return orders, err
				}
				var orderVars OrderVars
				for x := range openOrders.Data.Orders {
					orderVars, err = compatibleVars(openOrders.Data.Orders[x].Direction,
						openOrders.Data.Orders[x].OrderPriceType,
						openOrders.Data.Orders[x].Status)
					if err != nil {
						return orders, err
					}
					p, err := currency.NewPairFromString(openOrders.Data.Orders[x].ContractCode)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						PostOnly:        (orderVars.OrderType == order.PostOnly),
						Leverage:        openOrders.Data.Orders[x].LeverageRate,
						Price:           openOrders.Data.Orders[x].Price,
						Amount:          openOrders.Data.Orders[x].Volume,
						ExecutedAmount:  openOrders.Data.Orders[x].TradeVolume,
						RemainingAmount: openOrders.Data.Orders[x].Volume - openOrders.Data.Orders[x].TradeVolume,
						Fee:             openOrders.Data.Orders[x].Fee,
						Exchange:        h.Name,
						AssetType:       req.AssetType,
						ID:              openOrders.Data.Orders[x].OrderIDString,
						Side:            orderVars.Side,
						Type:            orderVars.OrderType,
						Status:          orderVars.Status,
						Pair:            p,
					})
				}
			}
		}
	case asset.Futures:
		for x := range req.Pairs {
			var currentPage int64 = 0
			for done := false; !done; {
				openOrders, err := h.FGetOpenOrders(req.Pairs[x].Base, currentPage, 50)
				if err != nil {
					return orders, err
				}
				var orderVars OrderVars
				for x := range openOrders.Data.Orders {
					orderVars, err = compatibleVars(openOrders.Data.Orders[x].Direction,
						openOrders.Data.Orders[x].OrderPriceType,
						openOrders.Data.Orders[x].Status)
					if err != nil {
						return orders, err
					}
					p, err := currency.NewPairFromString(openOrders.Data.Orders[x].ContractCode)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						PostOnly:        (orderVars.OrderType == order.PostOnly),
						Leverage:        openOrders.Data.Orders[x].LeverageRate,
						Price:           openOrders.Data.Orders[x].Price,
						Amount:          openOrders.Data.Orders[x].Volume,
						ExecutedAmount:  openOrders.Data.Orders[x].TradeVolume,
						RemainingAmount: openOrders.Data.Orders[x].Volume - openOrders.Data.Orders[x].TradeVolume,
						Fee:             openOrders.Data.Orders[x].Fee,
						Exchange:        h.Name,
						AssetType:       req.AssetType,
						ID:              openOrders.Data.Orders[x].OrderIDString,
						Side:            orderVars.Side,
						Type:            orderVars.OrderType,
						Status:          orderVars.Status,
						Pair:            p,
					})
				}
			}
		}
	}
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HUOBI) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		if len(req.Pairs) == 0 {
			return nil, errors.New("currency must be supplied")
		}
		states := "partial-canceled,filled,canceled"
		for i := range req.Pairs {
			resp, err := h.GetOrders(
				req.Pairs[i],
				"",
				"",
				"",
				states,
				"",
				"",
				"")
			if err != nil {
				return nil, err
			}
			for x := range resp {
				orderDetail := order.Detail{
					ID:             strconv.FormatInt(resp[x].ID, 10),
					Price:          resp[x].Price,
					Amount:         resp[x].Amount,
					Pair:           req.Pairs[i],
					Exchange:       h.Name,
					ExecutedAmount: resp[x].FilledAmount,
					Date:           time.Unix(0, resp[x].CreatedAt*int64(time.Millisecond)),
					Status:         order.Status(resp[x].State),
					AccountID:      strconv.FormatInt(resp[x].AccountID, 10),
					Fee:            resp[x].FilledFees,
				}
				setOrderSideAndType(resp[x].Type, &orderDetail)
				orders = append(orders, orderDetail)
			}
		}
	case asset.CoinMarginedFutures:
		for x := range req.Pairs {
			var currentPage int64 = 0
			for done := false; !done; {
				orderHistory, err := h.GetSwapOrderHistory(req.Pairs[x], "all", "all", []order.Status{order.AnyStatus}, int64(req.EndTime.Sub(req.StartTime).Hours()/24), currentPage, 50)
				if err != nil {
					return orders, err
				}
				var orderVars OrderVars
				for x := range orderHistory.Data.Orders {
					p, err := currency.NewPairFromString(orderHistory.Data.Orders[x].ContractCode)
					if err != nil {
						return orders, err
					}

					orderVars, err = compatibleVars(orderHistory.Data.Orders[x].Direction,
						orderHistory.Data.Orders[x].OrderPriceType,
						orderHistory.Data.Orders[x].Status)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						PostOnly:        (orderVars.OrderType == order.PostOnly),
						Leverage:        orderHistory.Data.Orders[x].LeverageRate,
						Price:           orderHistory.Data.Orders[x].Price,
						Amount:          orderHistory.Data.Orders[x].Volume,
						ExecutedAmount:  orderHistory.Data.Orders[x].TradeVolume,
						RemainingAmount: orderHistory.Data.Orders[x].Volume - orderHistory.Data.Orders[x].TradeVolume,
						Fee:             orderHistory.Data.Orders[x].Fee,
						Exchange:        h.Name,
						AssetType:       req.AssetType,
						ID:              orderHistory.Data.Orders[x].OrderIDString,
						Side:            orderVars.Side,
						Type:            orderVars.OrderType,
						Status:          orderVars.Status,
						Pair:            p,
					})
				}
				currentPage++
				if currentPage == orderHistory.Data.TotalPage {
					done = true
				}
			}
		}
	case asset.Futures:
		for x := range req.Pairs {
			var currentPage int64 = 0
			for done := false; !done; {
				openOrders, err := h.FGetOrderHistory(req.Pairs[x], "", "all", "all", "limit", []order.Status{order.AnyStatus}, int64(req.EndTime.Sub(req.StartTime).Hours()/24), currentPage, 50)
				if err != nil {
					return orders, err
				}
				var orderVars OrderVars
				for x := range openOrders.Data.Orders {
					orderVars, err = compatibleVars(openOrders.Data.Orders[x].Direction,
						openOrders.Data.Orders[x].OrderPriceType,
						openOrders.Data.Orders[x].Status)
					if err != nil {
						return orders, err
					}
					if req.Side != orderVars.Side {
						continue
					}
					if req.Type != orderVars.OrderType {
						continue
					}
					orderCreateTime := time.Unix(openOrders.Data.Orders[x].CreateDate, 0)

					p, err := currency.NewPairFromString(openOrders.Data.Orders[x].ContractCode)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						PostOnly:        (orderVars.OrderType == order.PostOnly),
						Leverage:        openOrders.Data.Orders[x].LeverageRate,
						Price:           openOrders.Data.Orders[x].Price,
						Amount:          openOrders.Data.Orders[x].Volume,
						ExecutedAmount:  openOrders.Data.Orders[x].TradeVolume,
						RemainingAmount: openOrders.Data.Orders[x].Volume - openOrders.Data.Orders[x].TradeVolume,
						Fee:             openOrders.Data.Orders[x].Fee,
						Exchange:        h.Name,
						AssetType:       req.AssetType,
						ID:              openOrders.Data.Orders[x].OrderIDString,
						Side:            orderVars.Side,
						Type:            orderVars.OrderType,
						Status:          orderVars.Status,
						Pair:            p,
						Date:            orderCreateTime,
					})
				}
				currentPage++
				if currentPage == openOrders.Data.TotalPage {
					done = true
				}
			}
		}
	}
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	return orders, nil
}

func setOrderSideAndType(requestType string, orderDetail *order.Detail) {
	switch SpotNewOrderRequestParamsType(requestType) {
	case SpotNewOrderRequestTypeBuyMarket:
		orderDetail.Side = order.Buy
		orderDetail.Type = order.Market
	case SpotNewOrderRequestTypeSellMarket:
		orderDetail.Side = order.Sell
		orderDetail.Type = order.Market
	case SpotNewOrderRequestTypeBuyLimit:
		orderDetail.Side = order.Buy
		orderDetail.Type = order.Limit
	case SpotNewOrderRequestTypeSellLimit:
		orderDetail.Side = order.Sell
		orderDetail.Type = order.Limit
	}
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (h *HUOBI) AuthenticateWebsocket() error {
	return h.wsLogin()
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (h *HUOBI) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin:
		return in.Short() + "in"
	case kline.FourHour:
		return "4hour"
	case kline.OneDay:
		return "1day"
	case kline.OneMonth:
		return "1mon"
	case kline.OneWeek:
		return "1week"
	case kline.OneYear:
		return "1year"
	}
	return ""
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (h *HUOBI) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := h.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	klineParams := KlinesRequestParams{
		Period: h.FormatExchangeKlineInterval(interval),
		Symbol: pair,
	}
	candles, err := h.GetSpotKline(klineParams)
	if err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: h.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	for x := range candles {
		if time.Unix(candles[x].ID, 0).Before(start) ||
			time.Unix(candles[x].ID, 0).After(end) {
			continue
		}
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   time.Unix(candles[x].ID, 0),
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
func (h *HUOBI) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return h.GetHistoricCandles(pair, a, start, end, interval)
}

// compatibleVars gets compatible variables for order vars
func compatibleVars(side, orderPriceType string, status int64) (OrderVars, error) {
	var resp OrderVars
	switch side {
	case "buy":
		resp.Side = order.Buy
	case "sell":
		resp.Side = order.Sell
	default:
		return resp, fmt.Errorf("invalid orderSide")
	}
	switch orderPriceType {
	case "limit":
		resp.OrderType = order.Limit
	case "opponent":
		resp.OrderType = order.Market
	case "post_only":
		resp.OrderType = order.PostOnly
	default:
		return resp, fmt.Errorf("invalid orderPriceType")
	}
	switch status {
	case 1, 2, 11:
		resp.Status = order.UnknownStatus
	case 3:
		resp.Status = order.Active
	case 4:
		resp.Status = order.PartiallyFilled
	case 5:
		resp.Status = order.PartiallyCancelled
	case 6:
		resp.Status = order.Filled
	case 7:
		resp.Status = order.Cancelled
	default:
		return resp, fmt.Errorf("invalid orderStatus")
	}
	return resp, nil
}
