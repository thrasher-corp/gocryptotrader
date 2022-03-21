package kraken

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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
func (k *Kraken) GetDefaultConfig() (*config.Exchange, error) {
	k.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = k.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = k.BaseCurrencies

	err := k.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if k.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = k.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default settings
func (k *Kraken) SetDefaults() {
	k.Name = "Kraken"
	k.Enabled = true
	k.Verbose = true
	k.API.CredentialsValidator.RequiresKey = true
	k.API.CredentialsValidator.RequiresSecret = true
	k.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	pairStore := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Separator: ",",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
			Separator: ",",
		},
	}

	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Delimiter: currency.UnderscoreDelimiter,
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.UnderscoreDelimiter,
		},
	}

	err := k.StoreAssetPairFormat(asset.Spot, pairStore)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = k.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = k.DisableAssetWebsocketSupport(asset.Futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	k.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				SubmitOrder:           true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				FiatDeposit:           true,
				FiatWithdraw:          true,
				TradeFee:              true,
				FiatDepositFee:        true,
				FiatWithdrawalFee:     true,
				CryptoDepositFee:      true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:     true,
				TradeFetching:      true,
				KlineFetching:      true,
				OrderbookFetching:  true,
				Subscribe:          true,
				Unsubscribe:        true,
				MessageCorrelation: true,
				SubmitOrder:        true,
				CancelOrder:        true,
				CancelOrders:       true,
				GetOrders:          true,
				GetOrder:           true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawCryptoWith2FA |
				exchange.AutoWithdrawFiatWithSetup |
				exchange.WithdrawFiatWith2FA,
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
					kline.FourHour.Word():   true,
					kline.OneDay.Word():     true,
					kline.FifteenDay.Word(): true,
					kline.OneWeek.Word():    true,
				},
			},
		},
	}

	k.Requester, err = request.New(k.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(krakenRateInterval, krakenRequestRate)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	k.API.Endpoints = k.NewEndpoints()
	err = k.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      krakenAPIURL,
		exchange.RestFutures:   futuresURL,
		exchange.WebsocketSpot: krakenWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	k.Websocket = stream.New()
	k.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	k.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	k.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets current exchange configuration
func (k *Kraken) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		k.SetEnabled(false)
		return nil
	}
	err = k.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = k.SeedAssets(context.TODO())
	if err != nil {
		return err
	}

	wsRunningURL, err := k.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = k.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            krakenWSURL,
		RunningURL:            wsRunningURL,
		Connector:             k.WsConnect,
		Subscriber:            k.Subscribe,
		Unsubscriber:          k.Unsubscribe,
		GenerateSubscriptions: k.GenerateDefaultSubscriptions,
		Features:              &k.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{SortBuffer: true},
	})
	if err != nil {
		return err
	}

	err = k.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            krakenWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  krakenWSURL,
	})
	if err != nil {
		return err
	}

	return k.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            krakenWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  krakenAuthWSURL,
		Authenticated:        true,
	})
}

// Start starts the Kraken go routine
func (k *Kraken) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		k.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Kraken wrapper
func (k *Kraken) Run() {
	if k.Verbose {
		k.PrintEnabledPairs()
	}

	forceUpdate := false
	if !k.BypassConfigFormatUpgrades {
		format, err := k.GetPairFormat(asset.UseDefault(), false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update tradable pairs. Err: %s",
				k.Name,
				err)
			return
		}
		enabled, err := k.GetEnabledPairs(asset.UseDefault())
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update tradable pairs. Err: %s",
				k.Name,
				err)
			return
		}

		avail, err := k.GetAvailablePairs(asset.UseDefault())
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update tradable pairs. Err: %s",
				k.Name,
				err)
			return
		}

		if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
			!common.StringDataContains(avail.Strings(), format.Delimiter) ||
			common.StringDataContains(avail.Strings(), "ZUSD") {
			var p currency.Pairs
			p, err = currency.NewPairsFromStrings([]string{currency.XBT.String() +
				format.Delimiter +
				currency.USD.String()})
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					k.Name,
					err)
			} else {
				log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, k.Name, asset.UseDefault(), p)
				forceUpdate = true

				err = k.UpdatePairs(p, asset.UseDefault(), true, true)
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%s failed to update currencies. Err: %s\n",
						k.Name,
						err)
				}
			}
		}
	}

	if !k.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := k.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			k.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (k *Kraken) FetchTradablePairs(ctx context.Context, assetType asset.Item) ([]string, error) {
	var products []string
	format, err := k.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Spot:
		if !assetTranslator.Seeded() {
			if err := k.SeedAssets(ctx); err != nil {
				return nil, err
			}
		}
		pairs, err := k.GetAssetPairs(ctx, []string{}, "")
		if err != nil {
			return nil, err
		}
		for i := range pairs {
			if strings.Contains(pairs[i].Altname, ".d") {
				continue
			}
			base := assetTranslator.LookupAltname(pairs[i].Base)
			if base == "" {
				log.Warnf(log.ExchangeSys,
					"%s unable to lookup altname for base currency %s",
					k.Name,
					pairs[i].Base)
				continue
			}
			quote := assetTranslator.LookupAltname(pairs[i].Quote)
			if quote == "" {
				log.Warnf(log.ExchangeSys,
					"%s unable to lookup altname for quote currency %s",
					k.Name,
					pairs[i].Quote)
				continue
			}
			products = append(products, base+format.Delimiter+quote)
		}
	case asset.Futures:
		pairs, err := k.GetFuturesMarkets(ctx)
		if err != nil {
			return nil, err
		}
		for x := range pairs.Instruments {
			if pairs.Instruments[x].Tradable {
				curr, err := currency.NewPairFromString(pairs.Instruments[x].Symbol)
				if err != nil {
					return nil, err
				}
				products = append(products, format.Format(curr))
			}
		}
	}
	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (k *Kraken) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := k.GetAssetTypes(false)
	for x := range assets {
		pairs, err := k.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}
		err = k.UpdatePairs(p, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (k *Kraken) UpdateTickers(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.Spot:
		pairs, err := k.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		pairsCollated, err := k.FormatExchangeCurrencies(pairs, a)
		if err != nil {
			return err
		}
		tickers, err := k.GetTickers(ctx, pairsCollated)
		if err != nil {
			return err
		}

		for i := range pairs {
			for c, t := range tickers {
				pairFmt, err := k.FormatExchangeCurrency(pairs[i], a)
				if err != nil {
					return err
				}
				if !strings.EqualFold(pairFmt.String(), c) {
					altCurrency := assetTranslator.LookupAltname(c)
					if altCurrency == "" {
						continue
					}
					if !strings.EqualFold(pairFmt.String(), altCurrency) {
						continue
					}
				}

				err = ticker.ProcessTicker(&ticker.Price{
					Last:         t.Last,
					High:         t.High,
					Low:          t.Low,
					Bid:          t.Bid,
					Ask:          t.Ask,
					Volume:       t.Volume,
					Open:         t.Open,
					Pair:         pairs[i],
					ExchangeName: k.Name,
					AssetType:    a})
				if err != nil {
					return err
				}
			}
		}
	case asset.Futures:
		t, err := k.GetFuturesTickers(ctx)
		if err != nil {
			return err
		}
		for x := range t.Tickers {
			pair, err := currency.NewPairFromString(t.Tickers[x].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         t.Tickers[x].Last,
				Bid:          t.Tickers[x].Bid,
				Ask:          t.Tickers[x].Ask,
				Volume:       t.Tickers[x].Vol24h,
				Open:         t.Tickers[x].Open24H,
				Pair:         pair,
				ExchangeName: k.Name,
				AssetType:    a})
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
func (k *Kraken) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := k.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(k.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (k *Kraken) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(k.Name, p, assetType)
	if err != nil {
		return k.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (k *Kraken) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(k.Name, p, assetType)
	if err != nil {
		return k.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (k *Kraken) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        k.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: k.CanVerifyOrderbook,
	}
	var err error
	switch assetType {
	case asset.Spot:
		var orderbookNew Orderbook
		orderbookNew, err = k.GetDepth(ctx, p)
		if err != nil {
			return nil, err
		}
		for x := range orderbookNew.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: orderbookNew.Bids[x].Amount,
				Price:  orderbookNew.Bids[x].Price,
			})
		}
		for y := range orderbookNew.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: orderbookNew.Asks[y].Amount,
				Price:  orderbookNew.Asks[y].Price,
			})
		}
	case asset.Futures:
		var futuresOB FuturesOrderbookData
		futuresOB, err = k.GetFuturesOrderbook(ctx, p)
		if err != nil {
			return nil, err
		}
		for x := range futuresOB.Orderbook.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Price:  futuresOB.Orderbook.Asks[x][0],
				Amount: futuresOB.Orderbook.Asks[x][1],
			})
		}
		for y := range futuresOB.Orderbook.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Price:  futuresOB.Orderbook.Bids[y][0],
				Amount: futuresOB.Orderbook.Bids[y][1],
			})
		}
	default:
		return book, fmt.Errorf("invalid assetType: %v", assetType)
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(k.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Kraken exchange - to-do
func (k *Kraken) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var balances []account.Balance
	info.Exchange = k.Name
	switch assetType {
	case asset.Spot:
		bal, err := k.GetBalance(ctx)
		if err != nil {
			return info, err
		}
		for key := range bal {
			translatedCurrency := assetTranslator.LookupAltname(key)
			if translatedCurrency == "" {
				log.Warnf(log.ExchangeSys, "%s unable to translate currency: %s\n",
					k.Name,
					key)
				continue
			}
			balances = append(balances, account.Balance{
				CurrencyName: currency.NewCode(translatedCurrency),
				Total:        bal[key],
			})
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			Currencies: balances,
		})
	case asset.Futures:
		bal, err := k.GetFuturesAccountData(ctx)
		if err != nil {
			return info, err
		}
		for name := range bal.Accounts {
			for code := range bal.Accounts[name].Balances {
				balances = append(balances, account.Balance{
					CurrencyName: currency.NewCode(code).Upper(),
					Total:        bal.Accounts[name].Balances[code],
				})
			}
			info.Accounts = append(info.Accounts, account.SubAccount{
				ID:         name,
				AssetType:  asset.Futures,
				Currencies: balances,
			})
		}
	}
	if err := account.Process(&info); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (k *Kraken) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(k.Name, assetType)
	if err != nil {
		return k.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (k *Kraken) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (k *Kraken) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	withdrawals, err := k.WithdrawStatus(ctx, c, "")
	for i := range withdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          withdrawals[i].Status,
			TransferID:      withdrawals[i].Refid,
			Timestamp:       time.Unix(int64(withdrawals[i].Time), 0),
			Amount:          withdrawals[i].Amount,
			Fee:             withdrawals[i].Fee,
			CryptoToAddress: withdrawals[i].Info,
			CryptoTxID:      withdrawals[i].TxID,
			Currency:        c.String(),
		})
	}

	return
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (k *Kraken) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	var tradeData []RecentTrades
	tradeData, err = k.GetTrades(ctx, p)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData {
		side := order.Buy
		if tradeData[i].BuyOrSell == "s" {
			side = order.Sell
		}
		resp = append(resp, trade.Data{
			Exchange:     k.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Volume,
			Timestamp:    convert.TimeFromUnixTimestampDecimal(tradeData[i].Time),
		})
	}

	err = k.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (k *Kraken) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (k *Kraken) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	switch s.AssetType {
	case asset.Spot:
		if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			var resp string
			s.Pair.Delimiter = "/" // required pair format: ISO 4217-A3
			resp, err := k.wsAddOrder(&WsAddOrderRequest{
				OrderType: s.Type.Lower(),
				OrderSide: s.Side.Lower(),
				Pair:      s.Pair.String(),
				Price:     s.Price,
				Volume:    s.Amount,
			})
			if err != nil {
				return submitOrderResponse, err
			}
			submitOrderResponse.OrderID = resp
			submitOrderResponse.IsOrderPlaced = true
		} else {
			var response AddOrderResponse
			response, err := k.AddOrder(ctx,
				s.Pair,
				s.Side.String(),
				s.Type.String(),
				s.Amount,
				s.Price,
				0,
				0,
				&AddOrderOptions{})
			if err != nil {
				return submitOrderResponse, err
			}
			if len(response.TransactionIds) > 0 {
				submitOrderResponse.OrderID = strings.Join(response.TransactionIds, ", ")
			}
		}
		if s.Type == order.Market {
			submitOrderResponse.FullyMatched = true
		}
		submitOrderResponse.IsOrderPlaced = true
	case asset.Futures:
		order, err := k.FuturesSendOrder(ctx,
			s.Type,
			s.Pair,
			s.Side.Lower(),
			"",
			s.ClientOrderID,
			"",
			s.ImmediateOrCancel,
			s.Amount,
			s.Price,
			0,
		)
		if err != nil {
			return submitOrderResponse, err
		}

		// check the status, anything that is not placed we error out
		if order.SendStatus.Status != "placed" {
			return submitOrderResponse,
				fmt.Errorf("submit order failed: %s",
					order.SendStatus.Status)
		}

		submitOrderResponse.OrderID = order.SendStatus.OrderID
		submitOrderResponse.IsOrderPlaced = true
	default:
		return submitOrderResponse, fmt.Errorf("invalid assetType")
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (k *Kraken) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (k *Kraken) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot:
		if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			return k.wsCancelOrders([]string{o.ID})
		}
		_, err := k.CancelExistingOrder(ctx, o.ID)
		return err
	case asset.Futures:
		_, err := k.FuturesCancelOrder(ctx, o.ID, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (k *Kraken) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	var ordersList []string
	for i := range orders {
		if err := orders[i].Validate(orders[i].StandardCancel()); err != nil {
			return order.CancelBatchResponse{}, err
		}
		ordersList = append(ordersList, orders[i].ID)
	}

	if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		err := k.wsCancelOrders(ordersList)
		return order.CancelBatchResponse{}, err
	}

	return order.CancelBatchResponse{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (k *Kraken) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	if err := req.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	switch req.AssetType {
	case asset.Spot:
		if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			resp, err := k.wsCancelAllOrders()
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			cancelAllOrdersResponse.Count = resp.Count
			return cancelAllOrdersResponse, err
		}

		var emptyOrderOptions OrderInfoOptions
		openOrders, err := k.GetOpenOrders(ctx, emptyOrderOptions)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for orderID := range openOrders.Open {
			var err error
			if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				err = k.wsCancelOrders([]string{orderID})
			} else {
				_, err = k.CancelExistingOrder(ctx, orderID)
			}
			if err != nil {
				cancelAllOrdersResponse.Status[orderID] = err.Error()
			}
		}
	case asset.Futures:
		cancelData, err := k.FuturesCancelAllOrders(ctx, req.Pair)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for x := range cancelData.CancelStatus.CancelledOrders {
			cancelAllOrdersResponse.Status[cancelData.CancelStatus.CancelledOrders[x].OrderID] = "cancelled"
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (k *Kraken) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	switch assetType {
	case asset.Spot:
		resp, err := k.QueryOrdersInfo(ctx,
			OrderInfoOptions{
				Trades: true,
			}, orderID)
		if err != nil {
			return orderDetail, err
		}

		orderInfo, ok := resp[orderID]
		if !ok {
			return orderDetail, fmt.Errorf("order %s not found in response", orderID)
		}

		if !assetType.IsValid() {
			assetType = asset.UseDefault()
		}

		avail, err := k.GetAvailablePairs(assetType)
		if err != nil {
			return orderDetail, err
		}

		format, err := k.GetPairFormat(assetType, true)
		if err != nil {
			return orderDetail, err
		}

		var trades []order.TradeHistory
		for i := range orderInfo.Trades {
			trades = append(trades, order.TradeHistory{
				TID: orderInfo.Trades[i],
			})
		}
		side, err := order.StringToOrderSide(orderInfo.Description.Type)
		if err != nil {
			return orderDetail, err
		}
		status, err := order.StringToOrderStatus(orderInfo.Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", k.Name, err)
		}
		oType, err := order.StringToOrderType(orderInfo.Description.OrderType)
		if err != nil {
			return orderDetail, err
		}

		p, err := currency.NewPairFromFormattedPairs(orderInfo.Description.Pair,
			avail,
			format)
		if err != nil {
			return orderDetail, err
		}

		price := orderInfo.Price
		if orderInfo.Status == statusOpen {
			price = orderInfo.Description.Price
		}

		orderDetail = order.Detail{
			Exchange:        k.Name,
			ID:              orderID,
			Pair:            p,
			Side:            side,
			Type:            oType,
			Date:            convert.TimeFromUnixTimestampDecimal(orderInfo.OpenTime),
			CloseTime:       convert.TimeFromUnixTimestampDecimal(orderInfo.CloseTime),
			Status:          status,
			Price:           price,
			Amount:          orderInfo.Volume,
			ExecutedAmount:  orderInfo.VolumeExecuted,
			RemainingAmount: orderInfo.Volume - orderInfo.VolumeExecuted,
			Fee:             orderInfo.Fee,
			Trades:          trades,
			Cost:            orderInfo.Cost,
		}
	case asset.Futures:
		orderInfo, err := k.FuturesGetFills(ctx, time.Time{})
		if err != nil {
			return orderDetail, err
		}
		for y := range orderInfo.Fills {
			if orderInfo.Fills[y].OrderID != orderID {
				continue
			}
			pair, err := currency.NewPairFromString(orderInfo.Fills[y].Symbol)
			if err != nil {
				return orderDetail, err
			}
			oSide, err := compatibleOrderSide(orderInfo.Fills[y].Side)
			if err != nil {
				return orderDetail, err
			}
			fillOrderType, err := compatibleFillOrderType(orderInfo.Fills[y].FillType)
			if err != nil {
				return orderDetail, err
			}
			timeVar, err := time.Parse(krakenFormat, orderInfo.Fills[y].FillTime)
			if err != nil {
				return orderDetail, err
			}
			orderDetail = order.Detail{
				ID:       orderID,
				Price:    orderInfo.Fills[y].Price,
				Amount:   orderInfo.Fills[y].Size,
				Side:     oSide,
				Type:     fillOrderType,
				Date:     timeVar,
				Pair:     pair,
				Exchange: k.Name,
			}
		}
	}
	return orderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (k *Kraken) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	if chain == "" {
		methods, err := k.GetDepositMethods(ctx, cryptocurrency.String())
		if err != nil {
			return nil, err
		}
		if len(methods) == 0 {
			return nil, errors.New("unable to get any deposit methods")
		}
		chain = methods[0].Method
	}

	depositAddr, err := k.GetCryptoDepositAddress(ctx, chain, cryptocurrency.String(), false)
	if err != nil {
		if strings.Contains(err.Error(), "no addresses returned") {
			depositAddr, err = k.GetCryptoDepositAddress(ctx, chain, cryptocurrency.String(), true)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &deposit.Address{
		Address: depositAddr[0].Address,
		Tag:     depositAddr[0].Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal
// Populate exchange.WithdrawRequest.TradePassword with withdrawal key name, as set up on your account
func (k *Kraken) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := k.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.TradePassword,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := k.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.TradePassword,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := k.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.TradePassword,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (k *Kraken) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !k.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return k.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (k *Kraken) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		resp, err := k.GetOpenOrders(ctx, OrderInfoOptions{})
		if err != nil {
			return nil, err
		}

		assetType := req.AssetType
		if !req.AssetType.IsValid() {
			assetType = asset.UseDefault()
		}

		avail, err := k.GetAvailablePairs(assetType)
		if err != nil {
			return nil, err
		}

		format, err := k.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}
		for i := range resp.Open {
			p, err := currency.NewPairFromFormattedPairs(resp.Open[i].Description.Pair,
				avail,
				format)
			if err != nil {
				return nil, err
			}
			side := order.Side(strings.ToUpper(resp.Open[i].Description.Type))
			orderType := order.Type(strings.ToUpper(resp.Open[i].Description.OrderType))
			orders = append(orders, order.Detail{
				ID:              i,
				Amount:          resp.Open[i].Volume,
				RemainingAmount: (resp.Open[i].Volume - resp.Open[i].VolumeExecuted),
				ExecutedAmount:  resp.Open[i].VolumeExecuted,
				Exchange:        k.Name,
				Date:            convert.TimeFromUnixTimestampDecimal(resp.Open[i].OpenTime),
				Price:           resp.Open[i].Description.Price,
				Side:            side,
				Type:            orderType,
				Pair:            p,
			})
		}
	case asset.Futures:
		var err error
		var pairs currency.Pairs
		if len(req.Pairs) > 0 {
			pairs = req.Pairs
		} else {
			pairs, err = k.GetEnabledPairs(asset.Futures)
			if err != nil {
				return orders, err
			}
		}
		activeOrders, err := k.FuturesOpenOrders(ctx)
		if err != nil {
			return orders, err
		}
		for i := range pairs {
			fPair, err := k.FormatExchangeCurrency(pairs[i], asset.Futures)
			if err != nil {
				return orders, err
			}
			for a := range activeOrders.OpenOrders {
				if activeOrders.OpenOrders[a].Symbol != fPair.String() {
					continue
				}
				oSide, err := compatibleOrderSide(activeOrders.OpenOrders[a].Side)
				if err != nil {
					return orders, err
				}
				oType, err := compatibleOrderType(activeOrders.OpenOrders[a].OrderType)
				if err != nil {
					return orders, err
				}
				timeVar, err := time.Parse(krakenFormat, activeOrders.OpenOrders[a].ReceivedTime)
				if err != nil {
					return orders, err
				}
				orders = append(orders, order.Detail{
					ID:       activeOrders.OpenOrders[a].OrderID,
					Price:    activeOrders.OpenOrders[a].LimitPrice,
					Amount:   activeOrders.OpenOrders[a].FilledSize,
					Side:     oSide,
					Type:     oType,
					Date:     timeVar,
					Pair:     fPair,
					Exchange: k.Name,
				})
			}
		}
	default:
		return nil, fmt.Errorf("%s assetType not supported", req.AssetType)
	}
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (k *Kraken) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		req := GetClosedOrdersOptions{}
		if getOrdersRequest.StartTime.Unix() > 0 {
			req.Start = strconv.FormatInt(getOrdersRequest.StartTime.Unix(), 10)
		}
		if getOrdersRequest.EndTime.Unix() > 0 {
			req.End = strconv.FormatInt(getOrdersRequest.EndTime.Unix(), 10)
		}

		assetType := getOrdersRequest.AssetType
		if !getOrdersRequest.AssetType.IsValid() {
			assetType = asset.UseDefault()
		}

		avail, err := k.GetAvailablePairs(assetType)
		if err != nil {
			return nil, err
		}

		format, err := k.GetPairFormat(assetType, true)
		if err != nil {
			return nil, err
		}

		resp, err := k.GetClosedOrders(ctx, req)
		if err != nil {
			return nil, err
		}

		for i := range resp.Closed {
			p, err := currency.NewPairFromFormattedPairs(resp.Closed[i].Description.Pair,
				avail,
				format)
			if err != nil {
				return nil, err
			}

			side := order.Side(strings.ToUpper(resp.Closed[i].Description.Type))
			status, err := order.StringToOrderStatus(resp.Closed[i].Status)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", k.Name, err)
			}
			orderType := order.Type(strings.ToUpper(resp.Closed[i].Description.OrderType))
			detail := order.Detail{
				ID:              i,
				Amount:          resp.Closed[i].Volume,
				ExecutedAmount:  resp.Closed[i].VolumeExecuted,
				RemainingAmount: resp.Closed[i].Volume - resp.Closed[i].VolumeExecuted,
				Cost:            resp.Closed[i].Cost,
				CostAsset:       p.Quote,
				Exchange:        k.Name,
				Date:            convert.TimeFromUnixTimestampDecimal(resp.Closed[i].OpenTime),
				CloseTime:       convert.TimeFromUnixTimestampDecimal(resp.Closed[i].CloseTime),
				Price:           resp.Closed[i].Description.Price,
				Side:            side,
				Status:          status,
				Type:            orderType,
				Pair:            p,
			}
			detail.InferCostsAndTimes()
			orders = append(orders, detail)
		}
	case asset.Futures:
		var orderHistory FuturesRecentOrdersData
		var err error
		var pairs currency.Pairs
		if len(getOrdersRequest.Pairs) > 0 {
			pairs = getOrdersRequest.Pairs
		} else {
			pairs, err = k.GetEnabledPairs(asset.Futures)
			if err != nil {
				return orders, err
			}
		}
		for p := range pairs {
			orderHistory, err = k.FuturesRecentOrders(ctx, pairs[p])
			if err != nil {
				return orders, err
			}
			for o := range orderHistory.OrderEvents {
				switch {
				case orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.UID != "":
					timeVar, err := time.Parse(krakenFormat,
						orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Timestamp)
					if err != nil {
						return orders, err
					}
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Quantity -
							orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.Filled,
						ID:        orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.ExecutionEvent.Execution.TakerOrder.ClientID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      timeVar,
						Side:      oDirection,
						Exchange:  k.Name,
						Pair:      pairs[p],
					})
				case orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.UID != "":
					timeVar, err := time.Parse(krakenFormat,
						orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Timestamp)
					if err != nil {
						return orders, err
					}
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Quantity -
							orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.Filled,
						ID:        orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.OrderRejected.RecentOrder.AccountID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      timeVar,
						Side:      oDirection,
						Exchange:  k.Name,
						Pair:      pairs[p],
						Status:    order.Rejected,
					})
				case orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.UID != "":
					timeVar, err := time.Parse(krakenFormat,
						orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Timestamp)
					if err != nil {
						return orders, err
					}
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Quantity -
							orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.Filled,
						ID:        orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.OrderCancelled.RecentOrder.AccountID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      timeVar,
						Side:      oDirection,
						Exchange:  k.Name,
						Pair:      pairs[p],
						Status:    order.Cancelled,
					})
				case orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.UID != "":
					timeVar, err := time.Parse(krakenFormat,
						orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Timestamp)
					if err != nil {
						return orders, err
					}
					oDirection, err := compatibleOrderSide(orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Direction)
					if err != nil {
						return orders, err
					}
					oType, err := compatibleOrderType(orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.OrderType)
					if err != nil {
						return orders, err
					}
					orders = append(orders, order.Detail{
						Price:          orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.LimitPrice,
						Amount:         orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Quantity,
						ExecutedAmount: orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Filled,
						RemainingAmount: orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Quantity -
							orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.Filled,
						ID:        orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.UID,
						ClientID:  orderHistory.OrderEvents[o].Event.OrderPlaced.RecentOrder.AccountID,
						AssetType: asset.Futures,
						Type:      oType,
						Date:      timeVar,
						Side:      oDirection,
						Exchange:  k.Name,
						Pair:      pairs[p],
					})
				default:
					return orders, fmt.Errorf("invalid orderHistory data")
				}
			}
		}
	}

	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)
	order.FilterOrdersByCurrencies(&orders, getOrdersRequest.Pairs)
	return orders, nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (k *Kraken) AuthenticateWebsocket(ctx context.Context) error {
	resp, err := k.GetWebsocketToken(ctx)
	if resp != "" {
		authToken = resp
	}
	return err
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (k *Kraken) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := k.UpdateAccountInfo(ctx, assetType)
	return k.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (k *Kraken) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Minutes(), 'f', -1, 64)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (k *Kraken) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := k.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: k.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	candles, err := k.GetOHLC(ctx, pair, k.FormatExchangeKlineInterval(interval))
	if err != nil {
		return kline.Item{}, err
	}
	for x := range candles {
		timeValue, err := convert.TimeFromUnixTimestampFloat(candles[x].Time * 1000)
		if err != nil {
			return kline.Item{}, err
		}
		if timeValue.Before(start) || timeValue.After(end) {
			continue
		}
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   timeValue,
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
func (k *Kraken) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := k.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: k.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	candles, err := k.GetOHLC(ctx, pair, k.FormatExchangeKlineInterval(interval))
	if err != nil {
		return kline.Item{}, err
	}
	for i := range candles {
		timeValue, err := convert.TimeFromUnixTimestampFloat(candles[i].Time * 1000)
		if err != nil {
			return kline.Item{}, err
		}
		if timeValue.Before(start) || timeValue.After(end) {
			continue
		}
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   timeValue,
			Open:   candles[i].Open,
			High:   candles[i].High,
			Low:    candles[i].Low,
			Close:  candles[i].Close,
			Volume: candles[i].Volume,
		})
	}
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

func compatibleOrderSide(side string) (order.Side, error) {
	switch {
	case strings.EqualFold(order.Buy.String(), side):
		return order.Buy, nil
	case strings.EqualFold(order.Sell.String(), side):
		return order.Sell, nil
	}
	return order.AnySide, fmt.Errorf("invalid side received")
}

func compatibleOrderType(orderType string) (order.Type, error) {
	var resp order.Type
	switch orderType {
	case "lmt":
		resp = order.Limit
	case "stp":
		resp = order.Stop
	case "take_profit":
		resp = order.TakeProfit
	default:
		return resp, fmt.Errorf("invalid orderType")
	}
	return resp, nil
}

func compatibleFillOrderType(fillType string) (order.Type, error) {
	var resp order.Type
	switch fillType {
	case "maker":
		resp = order.Limit
	case "taker":
		resp = order.Market
	case "liquidation":
		resp = order.Liquidation
	default:
		return resp, fmt.Errorf("invalid orderPriceType")
	}
	return resp, nil
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (k *Kraken) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	methods, err := k.GetDepositMethods(ctx, cryptocurrency.String())
	if err != nil {
		return nil, err
	}

	var availableChains []string
	for x := range methods {
		availableChains = append(availableChains, methods[x].Method)
	}
	return availableChains, nil
}
