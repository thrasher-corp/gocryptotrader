package gateio

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// GetDefaultConfig returns a default exchange config
func (g *Gateio) GetDefaultConfig() (*config.Exchange, error) {
	g.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = g.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = g.BaseCurrencies

	err := g.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if g.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = g.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (g *Gateio) SetDefaults() {
	g.Name = "GateIO"
	g.Enabled = true
	g.Verbose = true
	g.API.CredentialsValidator.RequiresKey = true
	g.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	err := g.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Futures, asset.Margin, asset.CrossMargin, asset.DeliveryFutures, asset.Options)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	g.Features = exchange.Features{
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
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				KlineFetching:          true,
				FullPayloadSubscribe:   true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrder:               true,
				AccountBalance:         true,
				Subscribe:              true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.TenSecond.Word():    true,
					kline.ThirtySecond.Word(): true,
					kline.OneMin.Word():       true,
					kline.FiveMin.Word():      true,
					kline.FifteenMin.Word():   true,
					kline.ThirtyMin.Word():    true,
					kline.OneHour.Word():      true,
					kline.TwoHour.Word():      true,
					kline.FourHour.Word():     true,
					kline.EightHour.Word():    true,
					kline.TwelveHour.Word():   true,
					kline.OneDay.Word():       true,
					kline.OneWeek.Word():      true,
					kline.ThirtyDay.Word():    true,
				},
			},
		},
	}
	g.Requester, err = request.New(g.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.API.Endpoints = g.NewEndpoints()
	err = g.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              gateioTradeURL,
		exchange.RestSpotSupplementary: gateioFuturesTestnetTrading,
		exchange.WebsocketSpot:         gateioWebsocketEndpoint,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.Websocket = stream.New()
	g.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	g.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	g.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user configuration
func (g *Gateio) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		g.SetEnabled(false)
		return nil
	}
	err = g.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := g.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = g.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            gateioWebsocketEndpoint,
		RunningURL:            wsRunningURL,
		Connector:             g.WsConnect,
		Subscriber:            g.Subscribe,
		GenerateSubscriptions: g.GenerateDefaultSubscriptions,
		Features:              &g.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return g.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            gateioWebsocketRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the GateIO go routine
func (g *Gateio) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the GateIO wrapper
func (g *Gateio) Run() {
	if g.Verbose {
		g.PrintEnabledPairs()
	}

	if !g.GetEnabledFeatures().AutoPairUpdates {
		return
	}
	err := g.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", g.Name, err)
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gateio) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if !g.SupportsAsset(a) {
		return nil, fmt.Errorf("%s does not support %s", g.Name, a.String())
	}
	fPair, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	if fPair.IsEmpty() || fPair.Quote.IsEmpty() {
		return nil, errInvalidOrEmptyCurrencyPair
	}
	var tickerData *ticker.Price
	switch a {
	case asset.Margin, asset.Spot, asset.CrossMargin:
		tickerNew, err := g.GetTicker(ctx, fPair, "")
		if err != nil {
			return nil, err
		}
		tickerData = &ticker.Price{
			Pair:         fPair,
			Low:          tickerNew.Low24H,
			High:         tickerNew.High24H,
			Bid:          tickerNew.HighestBid,
			Ask:          tickerNew.LowestAsk,
			Last:         tickerNew.Last,
			ExchangeName: g.Name,
			AssetType:    a,
		}
	case asset.Futures:
		if !(strings.EqualFold(fPair.Quote.String(), currency.USD.String()) || strings.EqualFold(fPair.Quote.String(), currency.USDT.String()) || strings.EqualFold(fPair.Quote.String(), currency.BTC.String())) {
			return nil, errUnsupportedSettleValue
		}
		tickers, err := g.GetFuturesTickers(ctx, fPair.Quote.String(), fPair)
		if err != nil {
			return nil, err
		}
		var tick *FuturesTicker
		for x := range tickers {
			if tickers[x].Contract == strings.ToUpper(fPair.String()) {
				tick = &tickers[x]
			}
		}
		if tick == nil {
			return nil, errNoTickerData
		}
		tickerData = &ticker.Price{
			Pair:         fPair,
			Low:          tick.Low24H,
			High:         tick.High24H,
			Last:         tick.Last,
			Volume:       tick.Volume24HBase,
			QuoteVolume:  tick.Volume24HQuote,
			ExchangeName: g.Name,
			AssetType:    a,
		}
	case asset.Options:
		tickers, err := g.GetOptionsTickers(ctx, fPair.String())
		if err != nil {
			println("ERROR:", err)
			return nil, err
		}
		var tick *OptionsTicker
		for x := range tickers {
			if strings.HasPrefix(tickers[x].Name, strings.ToUpper(fPair.String())) {
				tick = &tickers[x]
			}
		}
		if tick == nil {
			return nil, errNoTickerData
		}
		tickerData = &ticker.Price{
			Pair:         fPair,
			Last:         tick.LastPrice,
			Bid:          tick.Bid1Price,
			Ask:          tick.Ask1Price,
			AskSize:      tick.Ask1Size,
			BidSize:      tick.Bid1Size,
			ExchangeName: g.Name,
			AssetType:    a,
		}
	case asset.DeliveryFutures:
		if !(strings.EqualFold(fPair.Quote.String(), currency.USD.String()) || strings.EqualFold(fPair.Quote.String(), currency.USDT.String()) || strings.EqualFold(fPair.Quote.String(), currency.BTC.String())) {
			return nil, errUnsupportedSettleValue
		}
		tickers, err := g.GetDeliveryFutureTickers(ctx, fPair.Quote.String(), fPair)
		if err != nil {
			return nil, err
		}
		var tick *FuturesTicker
		for x := range tickers {
			if strings.EqualFold(tickers[x].Contract, strings.ToUpper(fPair.String())) {
				tick = &tickers[x]
			}
		}
		if tick == nil {
			return nil, errNoTickerData
		}
		tickerData = &ticker.Price{
			Pair:         fPair,
			Last:         tick.Last,
			High:         tick.High24H,
			Low:          tick.Low24H,
			Volume:       tick.Volume24H,
			QuoteVolume:  tick.Volume24HQuote,
			ExchangeName: g.Name,
			AssetType:    a,
		}
	}
	err = ticker.ProcessTicker(tickerData)
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(g.Name, fPair, a)
}

// FetchTicker retrives a list of tickers.
func (g *Gateio) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tickerNew, err := ticker.GetTicker(g.Name, fPair, assetType)
	if err != nil {
		return g.UpdateTicker(ctx, fPair, assetType)
	}
	return tickerNew, nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gateio) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	if !g.SupportsAsset(a) {
		return nil, fmt.Errorf("%s does not support %s", g.Name, a)
	}
	switch a {
	case asset.Spot:
		tradables, err := g.ListAllCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := []string{}
		for x := range tradables {
			p := strings.ToUpper(tradables[x].Base + currency.UnderscoreDelimiter + tradables[x].Quote)
			if !g.IsValidPairString(p) {
				continue
			}
			pairs = append(pairs, p)
		}
		return pairs, nil
	case asset.Margin, asset.CrossMargin:
		tradables, err := g.GetMarginSupportedCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := []string{}
		for x := range tradables {
			p := strings.ToUpper(tradables[x].Base + currency.UnderscoreDelimiter + tradables[x].Quote)
			if !g.IsValidPairString(p) {
				continue
			}
			pairs = append(pairs, p)
		}
		return pairs, nil
	case asset.Futures:
		btcContracts, err := g.GetAllFutureContracts(ctx, "btc")
		if err != nil {
			return nil, err
		}
		usdContracts, err := g.GetAllFutureContracts(ctx, "usd")
		if err != nil {
			return nil, err
		}
		usdtContracts, err := g.GetAllFutureContracts(ctx, "usdt")
		if err != nil {
			return nil, err
		}
		btcContracts = append(btcContracts, usdtContracts...)
		btcContracts = append(btcContracts, usdContracts...)
		pairs := []string{}
		for x := range btcContracts {
			p := btcContracts[x].Name
			if !g.IsValidPairString(p) {
				continue
			}
			pairs = append(pairs, p)
		}
		return pairs, nil
	case asset.DeliveryFutures:
		btcContracts, err := g.GetAllDeliveryContracts(ctx, "btc")
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return nil, err
		}
		usdContracts, err := g.GetAllDeliveryContracts(ctx, "usd")
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return nil, err
		}
		usdtContracts, err := g.GetAllDeliveryContracts(ctx, "usdt")
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return nil, err
		}
		btcContracts = append(btcContracts, usdtContracts...)
		btcContracts = append(btcContracts, usdContracts...)
		pairs := []string{}
		for x := range btcContracts {
			p := btcContracts[x].Name
			if !g.IsValidPairString(p) {
				continue
			}
			pairs = append(pairs, p)
		}
		return pairs, nil
	case asset.Options:
		underlyings, err := g.GetAllUnderlyings(ctx)
		if err != nil {
			return nil, err
		}
		pairs := []string{}
		for x := range underlyings {
			p := underlyings[x].Name
			if !g.IsValidPairString(p) {
				continue
			}
			pairs = append(pairs, p)
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%s does not support %s", g.Name, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (g *Gateio) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := g.GetAssetTypes(false)
	for x := range assets {
		pairs, err := g.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}
		err = g.UpdatePairs(p, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (g *Gateio) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !g.SupportsAsset(a) {
		return fmt.Errorf("%s does not support %s", g.Name, a)
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		tickers, err := g.GetTickers(ctx, currency.EMPTYPAIR, "")
		if err != nil {
			return err
		}
		for x := range tickers {
			currencyPair, err := currency.NewPairFromString(tickers[x].CurrencyPair)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers[x].Last,
				High:         tickers[x].High24H,
				Low:          tickers[x].Low24H,
				Bid:          tickers[x].HighestBid,
				Ask:          tickers[x].LowestAsk,
				QuoteVolume:  tickers[x].QuoteVolume,
				Volume:       tickers[x].BaseVolume,
				ExchangeName: g.Name,
				Pair:         currencyPair,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		tickers, err := g.GetFuturesTickers(ctx, "btc", currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		tickerUSD, err := g.GetFuturesTickers(ctx, "usd", currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		tickerUSDT, err := g.GetFuturesTickers(ctx, "usd", currency.EMPTYPAIR)
		if err != nil {
			return err
		}
		tickers = append(tickers, tickerUSD...)
		tickers = append(tickers, tickerUSDT...)
		for x := range tickers {
			currencyPair, err := currency.NewPairFromString(tickers[x].Contract)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers[x].Last,
				High:         tickers[x].High24H,
				Low:          tickers[x].Low24H,
				Volume:       tickers[x].Volume24H,
				QuoteVolume:  tickers[x].Volume24HQuote,
				ExchangeName: g.Name,
				Pair:         currencyPair,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.DeliveryFutures:
		tickers, err := g.GetDeliveryFutureTickers(ctx, "btc", currency.EMPTYPAIR)
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return err
		}
		tickerUSD, err := g.GetDeliveryFutureTickers(ctx, "usd", currency.EMPTYPAIR)
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return err
		}
		tickerUSDT, err := g.GetDeliveryFutureTickers(ctx, "usdt", currency.EMPTYPAIR)
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return err
		}
		tickers = append(tickers, tickerUSD...)
		tickers = append(tickers, tickerUSDT...)
		for x := range tickers {
			currencyPair, err := currency.NewPairFromString(tickers[x].Contract)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers[x].Last,
				High:         tickers[x].High24H,
				Low:          tickers[x].Low24H,
				Volume:       tickers[x].Volume24H,
				QuoteVolume:  tickers[x].Volume24HQuote,
				Pair:         currencyPair,
				ExchangeName: g.Name,
				AssetType:    a,
			})
			if err != nil {
				return err
			}
		}
	case asset.Options:
		pairs, err := g.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		for i := range pairs {
			tickers, err := g.GetOptionsTickers(ctx, pairs[i].String())
			if err != nil {
				return err
			}
			for x := range tickers {
				currencyPair, err := currency.NewPairFromString(tickers[x].Name)
				if err != nil {
					return err
				}
				err = ticker.ProcessTicker(&ticker.Price{
					Last:    tickers[x].LastPrice,
					Ask:     tickers[x].Ask1Price,
					AskSize: tickers[x].Ask1Size,
					Bid:     tickers[x].Bid1Price,
					BidSize: tickers[x].Bid1Size,

					Pair:         currencyPair,
					ExchangeName: g.Name,
					AssetType:    a,
				})
			}
		}
	default:
		return fmt.Errorf("%s does not support %s", g.Name, a)
	}
	return nil
}

// // FetchOrderbook returns orderbook base on the currency pair
func (g *Gateio) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(g.Name, p, assetType)
	if err != nil {
		return g.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gateio) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        g.Name,
		Asset:           a,
		VerifyOrderbook: g.CanVerifyOrderbook,
	}
	fPair, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return book, err
	}
	book.Pair = fPair
	var orderbookNew *Orderbook
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		orderbookNew, err = g.GetOrderbook(ctx, fPair, "", 0, true)
	case asset.Futures:
		if !(strings.EqualFold(fPair.Quote.String(), currency.USD.String()) || strings.EqualFold(fPair.Quote.String(), currency.USDT.String()) || strings.EqualFold(fPair.Quote.String(), currency.BTC.String())) {
			return nil, errUnsupportedSettleValue
		}
		orderbookNew, err = g.GetFuturesOrderbook(ctx, fPair.Quote.String(), fPair, "", 0, true)
	case asset.DeliveryFutures:
		if !(strings.EqualFold(fPair.Quote.String(), currency.USD.String()) || strings.EqualFold(fPair.Quote.String(), currency.USDT.String()) || strings.EqualFold(fPair.Quote.String(), currency.BTC.String())) {
			return nil, errUnsupportedSettleValue
		}
		contract, err := g.GetContractFromCurrencyPair(ctx, fPair, a)
		if err != nil {
			return nil, err
		}
		orderbookNew, err = g.GetDeliveryOrderbook(ctx, fPair.Quote.String(), contract, "", 0, true)
	case asset.Options:
		if !(strings.EqualFold(fPair.Quote.String(), currency.USD.String()) || strings.EqualFold(fPair.Quote.String(), currency.USDT.String()) || strings.EqualFold(fPair.Quote.String(), currency.BTC.String())) {
			return nil, errUnsupportedSettleValue
		}
		orderbookNew, err = g.GetOptionsOrderbook(ctx, fPair, "", 0, true)
	}
	if err != nil {
		return book, err
	}
	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	leng := len(orderbookNew.Bids)
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[leng-1-x].Amount,
			Price:  orderbookNew.Bids[leng-1-x].Price,
		}
	}
	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	leng = len(orderbookNew.Asks)
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: orderbookNew.Asks[leng-1-x].Amount,
			Price:  orderbookNew.Asks[leng-1-x].Price,
		}
	}
	if err = book.Process(); err != nil {
		return book, err
	}
	return orderbook.Get(g.Name, fPair, a)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
func (g *Gateio) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	// var balances []account.Balance
	// if g.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
	// 	resp, err := g.wsGetBalance([]string{})
	// 	if err != nil {
	// 		return info, err
	// 	}
	// 	var currData []account.Balance
	// 	for k := range resp.Result {
	// 		currData = append(currData, account.Balance{
	// 			CurrencyName: currency.NewCode(k),
	// 			Total:        resp.Result[k].Available + resp.Result[k].Freeze,
	// 			Hold:         resp.Result[k].Freeze,
	// 			Free:         resp.Result[k].Available,
	// 		})
	// 	}
	// 	info.Accounts = append(info.Accounts, account.SubAccount{
	// 		Currencies: currData,
	// 		AssetType:  assetType,
	// 	})
	// } else {
	// balance, err := g.GetSpotAccounts(ctx, currency.EMPTYCODE)
	// if err != nil {
	// 	return info, err
	// }
	// switch l := balance.Locked.(type) {
	// case map[string]interface{}:
	// 	for x := range l {
	// 		var lockedF float64
	// 		lockedF, err = strconv.ParseFloat(l[x].(string), 64)
	// 		if err != nil {
	// 			return info, err
	// 		}

	// 		balances = append(balances, account.Balance{
	// 			CurrencyName: currency.NewCode(x),
	// 			Hold:         lockedF,
	// 		})
	// 	}
	// default:
	// 	break
	// }
	// switch v := balance.Available.(type) {
	// case map[string]interface{}:
	// 	for x := range v {
	// 		var availAmount float64
	// 		availAmount, err = strconv.ParseFloat(v[x].(string), 64)
	// 		if err != nil {
	// 			return info, err
	// 		}

	// 		var updated bool
	// 		for i := range balances {
	// 			if !balances[i].CurrencyName.Equal(currency.NewCode(x)) {
	// 				continue
	// 			}
	// 			balances[i].Total = balances[i].Hold + availAmount
	// 			balances[i].Free = availAmount
	// 			balances[i].AvailableWithoutBorrow = availAmount
	// 			updated = true
	// 			break
	// 		}
	// 		if !updated {
	// 			balances = append(balances, account.Balance{
	// 				CurrencyName: currency.NewCode(x),
	// 				Total:        availAmount,
	// 			})
	// 		}
	// 	}
	// default:
	// 	break
	// }
	// info.Accounts = append(info.Accounts, account.SubAccount{
	// 	AssetType:  assetType,
	// 	Currencies: balances,
	// })
	// // }
	// info.Exchange = g.Name
	// creds, err := g.GetCredentials(ctx)
	// if err != nil {
	// 	return account.Holdings{}, err
	// }
	// if err := account.Process(&info, creds); err != nil {
	// 	return account.Holdings{}, err
	// }
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (g *Gateio) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(g.Name, creds, assetType)
	if err != nil {
		return g.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// // GetFundingHistory returns funding history, deposits and
// // withdrawals
// func (g *Gateio) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
// 	return nil, common.ErrFunctionNotSupported
// }

// // GetWithdrawalsHistory returns previous withdrawals data
// func (g *Gateio) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) (resp []exchange.WithdrawalHistory, err error) {
// 	return nil, common.ErrNotYetImplemented
// }

// // GetRecentTrades returns the most recent trades for a currency and asset
// func (g *Gateio) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
// 	var err error
// 	p, err = g.FormatExchangeCurrency(p, assetType)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var tradeData TradeHistory
// 	tradeData, err = g.GetTrades(ctx, p.String())
// 	if err != nil {
// 		return nil, err
// 	}
// 	resp := make([]trade.Data, len(tradeData.Data))
// 	for i := range tradeData.Data {
// 		var side order.Side
// 		side, err = order.StringToOrderSide(tradeData.Data[i].Type)
// 		if err != nil {
// 			return nil, err
// 		}
// 		resp[i] = trade.Data{
// 			Exchange:     g.Name,
// 			TID:          tradeData.Data[i].TradeID,
// 			CurrencyPair: p,
// 			AssetType:    assetType,
// 			Side:         side,
// 			Price:        tradeData.Data[i].Rate,
// 			Amount:       tradeData.Data[i].Amount,
// 			Timestamp:    time.Unix(tradeData.Data[i].Timestamp, 0),
// 		}
// 	}

// 	err = g.AddTradesToBuffer(resp...)
// 	if err != nil {
// 		return nil, err
// 	}

// 	sort.Sort(trade.ByDate(resp))
// 	return resp, nil
// }

// // GetHistoricTrades returns historic trade data within the timeframe provided
// func (g *Gateio) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
// 	return nil, common.ErrFunctionNotSupported
// }

// // SubmitOrder submits a new order
// // TODO: support multiple order types (IOC)
// func (g *Gateio) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
// 	if err := s.Validate(); err != nil {
// 		return nil, err
// 	}

// 	var orderTypeFormat string
// 	if s.Side == order.Buy {
// 		orderTypeFormat = order.Buy.Lower()
// 	} else {
// 		orderTypeFormat = order.Sell.Lower()
// 	}

// 	fPair, err := g.FormatExchangeCurrency(s.Pair, s.AssetType)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var spotNewOrderRequestParams = SpotNewOrderRequestParams{
// 		Amount: s.Amount,
// 		Price:  s.Price,
// 		Symbol: fPair.String(),
// 		Type:   orderTypeFormat,
// 	}

// 	response, err := g.SpotNewOrder(ctx, spotNewOrderRequestParams)
// 	if err != nil {
// 		return nil, err
// 	}
// 	subResp, err := s.DeriveSubmitResponse(strconv.FormatInt(response.OrderNumber, 10))
// 	if err != nil {
// 		return nil, err
// 	}
// 	if response.LeftAmount == 0 {
// 		subResp.Status = order.Filled
// 	}
// 	return subResp, nil
// }

// // ModifyOrder will allow of changing orderbook placement and limit to
// // market conversion
// func (g *Gateio) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
// 	return nil, common.ErrFunctionNotSupported
// }

// // CancelOrder cancels an order by its corresponding ID number
// func (g *Gateio) CancelOrder(ctx context.Context, o *order.Cancel) error {
// 	if err := o.Validate(o.StandardCancel()); err != nil {
// 		return err
// 	}

// 	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
// 	if err != nil {
// 		return err
// 	}

// 	fpair, err := g.FormatExchangeCurrency(o.Pair, o.AssetType)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = g.CancelExistingOrder(ctx, orderIDInt, fpair.String())
// 	return err
// }

// // CancelBatchOrders cancels an orders by their corresponding ID numbers
// func (g *Gateio) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
// 	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
// }

// // CancelAllOrders cancels all orders associated with a currency pair
// func (g *Gateio) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
// 	cancelAllOrdersResponse := order.CancelAllResponse{
// 		Status: make(map[string]string),
// 	}
// 	openOrders, err := g.GetOpenOrders(ctx, "")
// 	if err != nil {
// 		return cancelAllOrdersResponse, err
// 	}

// 	uniqueSymbols := make(map[string]int)
// 	for i := range openOrders.Orders {
// 		uniqueSymbols[openOrders.Orders[i].CurrencyPair]++
// 	}

// 	for unique := range uniqueSymbols {
// 		err = g.CancelAllExistingOrders(ctx, -1, unique)
// 		if err != nil {
// 			cancelAllOrdersResponse.Status[unique] = err.Error()
// 		}
// 	}

// 	return cancelAllOrdersResponse, nil
// }

// // GetOrderInfo returns order information based on order ID
// func (g *Gateio) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
// 	var orderDetail order.Detail
// 	orders, err := g.GetOpenOrders(ctx, "")
// 	if err != nil {
// 		return orderDetail, errors.New("failed to get open orders")
// 	}

// 	if assetType == asset.Empty {
// 		assetType = asset.Spot
// 	}

// 	format, err := g.GetPairFormat(assetType, false)
// 	if err != nil {
// 		return orderDetail, err
// 	}

// 	for x := range orders.Orders {
// 		if orders.Orders[x].OrderNumber != orderID {
// 			continue
// 		}
// 		orderDetail.Exchange = g.Name
// 		orderDetail.OrderID = orders.Orders[x].OrderNumber
// 		orderDetail.RemainingAmount = orders.Orders[x].InitialAmount - orders.Orders[x].FilledAmount
// 		orderDetail.ExecutedAmount = orders.Orders[x].FilledAmount
// 		orderDetail.Amount = orders.Orders[x].InitialAmount
// 		orderDetail.Date = time.Unix(orders.Orders[x].Timestamp, 0)
// 		if orderDetail.Status, err = order.StringToOrderStatus(orders.Orders[x].Status); err != nil {
// 			log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
// 		}
// 		orderDetail.Price = orders.Orders[x].Rate
// 		orderDetail.Pair, err = currency.NewPairDelimiter(orders.Orders[x].CurrencyPair,
// 			format.Delimiter)
// 		if err != nil {
// 			return orderDetail, err
// 		}
// 		if strings.EqualFold(orders.Orders[x].Type, order.Ask.String()) {
// 			orderDetail.Side = order.Ask
// 		} else if strings.EqualFold(orders.Orders[x].Type, order.Bid.String()) {
// 			orderDetail.Side = order.Buy
// 		}
// 		return orderDetail, nil
// 	}
// 	return orderDetail, fmt.Errorf("no order found with id %v", orderID)
// }

// // GetDepositAddress returns a deposit address for a specified currency
// func (g *Gateio) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
// 	addr, err := g.GetCryptoDepositAddress(ctx, cryptocurrency.String())
// 	if err != nil {
// 		return nil, err
// 	}

// 	if addr.Address == gateioGenerateAddress {
// 		return nil,
// 			errors.New("new deposit address is being generated, please retry again shortly")
// 	}

// 	if chain != "" {
// 		for x := range addr.MultichainAddresses {
// 			if strings.EqualFold(addr.MultichainAddresses[x].Chain, chain) {
// 				return &deposit.Address{
// 					Address: addr.MultichainAddresses[x].Address,
// 					Tag:     addr.MultichainAddresses[x].PaymentName,
// 				}, nil
// 			}
// 		}
// 		return nil, fmt.Errorf("network %s not found", chain)
// 	}
// 	return &deposit.Address{
// 		Address: addr.Address,
// 		Tag:     addr.Tag,
// 	}, nil
// }

// // WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// // submitted
// func (g *Gateio) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
// 	if err := withdrawRequest.Validate(); err != nil {
// 		return nil, err
// 	}
// 	return g.WithdrawCrypto(ctx,
// 		withdrawRequest.Currency.String(),
// 		withdrawRequest.Crypto.Address,
// 		withdrawRequest.Crypto.AddressTag,
// 		withdrawRequest.Crypto.Chain,
// 		withdrawRequest.Amount,
// 	)
// }

// // WithdrawFiatFunds returns a withdrawal ID when a
// // withdrawal is submitted
// func (g *Gateio) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
// 	return nil, common.ErrFunctionNotSupported
// }

// // WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// // withdrawal is submitted
// func (g *Gateio) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
// 	return nil, common.ErrFunctionNotSupported
// }

// // GetFeeByType returns an estimate of fee based on type of transaction
// func (g *Gateio) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
// 	if feeBuilder == nil {
// 		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
// 	}
// 	if !g.AreCredentialsValid(ctx) && // Todo check connection status
// 		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
// 		feeBuilder.FeeType = exchange.OfflineTradeFee
// 	}
// 	return g.GetFee(ctx, feeBuilder)
// }

// // GetActiveOrders retrieves any orders that are active/open
// func (g *Gateio) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
// 	if err := req.Validate(); err != nil {
// 		return nil, err
// 	}

// 	var orders []order.Detail
// 	var currPair string
// 	if len(req.Pairs) == 1 {
// 		fPair, err := g.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
// 		if err != nil {
// 			return nil, err
// 		}
// 		currPair = fPair.String()
// 	}
// 	if g.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
// 		for i := 0; ; i += 100 {
// 			resp, err := g.wsGetOrderInfo(req.Type.String(), i, 100)
// 			if err != nil {
// 				return orders, err
// 			}

// 			for j := range resp.WebSocketOrderQueryRecords {
// 				orderSide := order.Buy
// 				if resp.WebSocketOrderQueryRecords[j].Type == 1 {
// 					orderSide = order.Sell
// 				}
// 				orderType := order.Market
// 				if resp.WebSocketOrderQueryRecords[j].OrderType == 1 {
// 					orderType = order.Limit
// 				}
// 				p, err := currency.NewPairFromString(resp.WebSocketOrderQueryRecords[j].Market)
// 				if err != nil {
// 					return nil, err
// 				}
// 				orders = append(orders, order.Detail{
// 					Exchange:        g.Name,
// 					AccountID:       strconv.FormatInt(resp.WebSocketOrderQueryRecords[j].User, 10),
// 					OrderID:         strconv.FormatInt(resp.WebSocketOrderQueryRecords[j].ID, 10),
// 					Pair:            p,
// 					Side:            orderSide,
// 					Type:            orderType,
// 					Date:            convert.TimeFromUnixTimestampDecimal(resp.WebSocketOrderQueryRecords[j].Ctime),
// 					Price:           resp.WebSocketOrderQueryRecords[j].Price,
// 					Amount:          resp.WebSocketOrderQueryRecords[j].Amount,
// 					ExecutedAmount:  resp.WebSocketOrderQueryRecords[j].FilledAmount,
// 					RemainingAmount: resp.WebSocketOrderQueryRecords[j].Left,
// 					Fee:             resp.WebSocketOrderQueryRecords[j].DealFee,
// 				})
// 			}
// 			if len(resp.WebSocketOrderQueryRecords) < 100 {
// 				break
// 			}
// 		}
// 	} else {
// 		resp, err := g.GetOpenOrders(ctx, currPair)
// 		if err != nil {
// 			return nil, err
// 		}

// 		format, err := g.GetPairFormat(asset.Spot, false)
// 		if err != nil {
// 			return nil, err
// 		}

// 		for i := range resp.Orders {
// 			if resp.Orders[i].Status != "open" {
// 				continue
// 			}
// 			var symbol currency.Pair
// 			symbol, err = currency.NewPairDelimiter(resp.Orders[i].CurrencyPair,
// 				format.Delimiter)
// 			if err != nil {
// 				return nil, err
// 			}
// 			var side order.Side
// 			side, err = order.StringToOrderSide(resp.Orders[i].Type)
// 			if err != nil {
// 				log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
// 			}
// 			status, err := order.StringToOrderStatus(resp.Orders[i].Status)
// 			if err != nil {
// 				log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
// 			}
// 			orderDate := time.Unix(resp.Orders[i].Timestamp, 0)
// 			orders = append(orders, order.Detail{
// 				OrderID:         resp.Orders[i].OrderNumber,
// 				Amount:          resp.Orders[i].Amount,
// 				ExecutedAmount:  resp.Orders[i].Amount - resp.Orders[i].FilledAmount,
// 				RemainingAmount: resp.Orders[i].FilledAmount,
// 				Price:           resp.Orders[i].Rate,
// 				Date:            orderDate,
// 				Side:            side,
// 				Exchange:        g.Name,
// 				Pair:            symbol,
// 				Status:          status,
// 			})
// 		}
// 	}
// 	err := order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
// 	if err != nil {
// 		log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
// 	}
// 	order.FilterOrdersBySide(&orders, req.Side)
// 	return orders, nil
// }

// // GetOrderHistory retrieves account order information
// // Can Limit response to specific order status
// func (g *Gateio) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
// 	if err := req.Validate(); err != nil {
// 		return nil, err
// 	}

// 	var trades []TradesResponse
// 	for i := range req.Pairs {
// 		resp, err := g.GetTradeHistory(ctx, req.Pairs[i].String())
// 		if err != nil {
// 			return nil, err
// 		}
// 		trades = append(trades, resp.Trades...)
// 	}

// 	format, err := g.GetPairFormat(asset.Spot, false)
// 	if err != nil {
// 		return nil, err
// 	}

// 	orders := make([]order.Detail, len(trades))
// 	for i := range trades {
// 		var pair currency.Pair
// 		pair, err = currency.NewPairDelimiter(trades[i].Pair, format.Delimiter)
// 		if err != nil {
// 			return nil, err
// 		}
// 		var side order.Side
// 		side, err = order.StringToOrderSide(trades[i].Type)
// 		if err != nil {
// 			log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
// 		}
// 		orderDate := time.Unix(trades[i].TimeUnix, 0)
// 		detail := order.Detail{
// 			OrderID:              strconv.FormatInt(trades[i].OrderID, 10),
// 			Amount:               trades[i].Amount,
// 			ExecutedAmount:       trades[i].Amount,
// 			Price:                trades[i].Rate,
// 			AverageExecutedPrice: trades[i].Rate,
// 			Date:                 orderDate,
// 			Side:                 side,
// 			Exchange:             g.Name,
// 			Pair:                 pair,
// 		}
// 		detail.InferCostsAndTimes()
// 		orders[i] = detail
// 	}

// 	err = order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
// 	if err != nil {
// 		log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
// 	}
// 	order.FilterOrdersBySide(&orders, req.Side)
// 	return orders, nil
// }

// // AuthenticateWebsocket sends an authentication message to the websocket
// func (g *Gateio) AuthenticateWebsocket(ctx context.Context) error {
// 	return g.wsServerSignIn(ctx)
// }

// // ValidateCredentials validates current credentials used for wrapper
// // functionality
// func (g *Gateio) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
// 	_, err := g.UpdateAccountInfo(ctx, assetType)
// 	return g.CheckTransientError(err)
// }

// // FormatExchangeKlineInterval returns Interval to exchange formatted string
// func (g *Gateio) FormatExchangeKlineInterval(in kline.Interval) string {
// 	return strconv.FormatFloat(in.Duration().Seconds(), 'f', 0, 64)
// }

// // GetHistoricCandles returns candles between a time period for a set time interval
// func (g *Gateio) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
// 	if err := g.ValidateKline(pair, a, interval); err != nil {
// 		return kline.Item{}, err
// 	}

// 	hours := time.Since(start).Hours()
// 	formattedPair, err := g.FormatExchangeCurrency(pair, a)
// 	if err != nil {
// 		return kline.Item{}, err
// 	}

// 	params := KlinesRequestParams{
// 		Symbol:   formattedPair.String(),
// 		GroupSec: g.FormatExchangeKlineInterval(interval),
// 		HourSize: int(hours),
// 	}

// 	klineData, err := g.GetSpotKline(ctx, params)
// 	if err != nil {
// 		return kline.Item{}, err
// 	}
// 	klineData.Interval = interval
// 	klineData.Pair = pair
// 	klineData.Asset = a

// 	klineData.SortCandlesByTimestamp(false)
// 	klineData.RemoveOutsideRange(start, end)
// 	return klineData, nil
// }

// // GetHistoricCandlesExtended returns candles between a time period for a set time interval
// func (g *Gateio) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
// 	return g.GetHistoricCandles(ctx, pair, a, start, end, interval)
// }

// // GetAvailableTransferChains returns the available transfer blockchains for the specific
// // cryptocurrency
// func (g *Gateio) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
// 	chains, err := g.GetCryptoDepositAddress(ctx, cryptocurrency.String())
// 	if err != nil {
// 		return nil, err
// 	}

// 	availableChains := make([]string, len(chains.MultichainAddresses))
// 	for x := range chains.MultichainAddresses {
// 		availableChains[x] = chains.MultichainAddresses[x].Chain
// 	}
// 	return availableChains, nil
// }
