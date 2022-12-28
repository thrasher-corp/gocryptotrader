package gateio

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
		err = g.UpdateTradablePairs(context.TODO(), forceUpdate)
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
					kline.HundredMilliseconds.Word():  true,
					kline.ThousandMilliseconds.Word(): true,
					kline.TenSecond.Word():            true,
					kline.ThirtySecond.Word():         true,
					kline.OneMin.Word():               true,
					kline.FiveMin.Word():              true,
					kline.FifteenMin.Word():           true,
					kline.ThirtyMin.Word():            true,
					kline.OneHour.Word():              true,
					kline.TwoHour.Word():              true,
					kline.FourHour.Word():             true,
					kline.EightHour.Word():            true,
					kline.TwelveHour.Word():           true,
					kline.OneDay.Word():               true,
					kline.OneWeek.Word():              true,
					kline.OneMonth.Word():             true,
				},
				ResultLimit: 1000,
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
	g.WsChannelsMultiplexer = &WsMultiplexer{
		Channels:   map[string]chan *WsEventResponse{},
		Register:   make(chan *wsChanReg),
		Unregister: make(chan string),
		Message:    make(chan *WsEventResponse),
	}
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
		Unsubscriber:          g.Unsubscribe,
		GenerateSubscriptions: g.GenerateDefaultSubscriptions,
		Features:              &g.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}
	return g.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  gateioWebsocketEndpoint,
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
	err := g.UpdateTradablePairs(context.TODO(), forceUpdate)
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
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair = fPair.Upper()
	var tickerData *ticker.Price
	switch a {
	case asset.Margin, asset.Spot, asset.CrossMargin:
		var tickerNew *Ticker
		tickerNew, err = g.GetTicker(ctx, fPair.String(), "")
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
		if !strings.HasPrefix(fPair.Quote.String(), currency.USD.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.USDT.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.BTC.Upper().String()) {
			return nil, errUnsupportedSettleValue
		}
		var tickers []FuturesTicker
		tickers, err = g.GetFuturesTickers(ctx, fPair.Quote.String(), fPair.Upper().String())
		if err != nil {
			return nil, err
		}
		var tick *FuturesTicker
		for x := range tickers {
			if tickers[x].Contract == strings.ToUpper(fPair.String()) {
				tick = &tickers[x]
				break
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
		var underlying string
		var tickers []OptionsTicker
		underlying, err = g.GetUnderlyingFromCurrencyPair(fPair)
		if err != nil {
			return nil, err
		}
		tickers, err = g.GetOptionsTickers(ctx, underlying)
		if err != nil {
			return nil, err
		}
		for x := range tickers {
			if !fPair.IsEmpty() && !strings.EqualFold(tickers[x].Name, strings.ToUpper(fPair.String())) {
				continue
			}
			tick := &tickers[x]
			var cp currency.Pair
			cp, err = currency.NewPairFromString((strings.ReplaceAll(tick.Name, currency.DashDelimiter, currency.UnderscoreDelimiter)))
			cp.Quote = currency.NewCode(strings.ReplaceAll(cp.Quote.String(), currency.UnderscoreDelimiter, currency.DashDelimiter))
			if err != nil {
				return nil, err
			}
			if tick == nil {
				return nil, errNoTickerData
			}
			tickerData = &ticker.Price{
				Pair:         cp,
				Last:         tick.LastPrice,
				Bid:          tick.Bid1Price,
				Ask:          tick.Ask1Price,
				AskSize:      tick.Ask1Size,
				BidSize:      tick.Bid1Size,
				ExchangeName: g.Name,
				AssetType:    a,
			}
			err = ticker.ProcessTicker(tickerData)
			if err != nil {
				return nil, err
			}
		}
		return ticker.GetTicker(g.Name, fPair, a)
	case asset.DeliveryFutures:
		if !strings.HasPrefix(fPair.Quote.String(), currency.USD.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.USDT.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.BTC.Upper().String()) {
			return nil, errUnsupportedSettleValue
		}
		var settle string
		settle, err = g.getSettlementFromCurrency(fPair)
		if err != nil {
			return nil, err
		}
		var tickers []FuturesTicker
		tickers, err = g.GetDeliveryFutureTickers(ctx, settle, fPair)
		if err != nil {
			return nil, err
		}
		var tick *FuturesTicker
		for x := range tickers {
			if !fPair.IsEmpty() && tickers[x].Contract == fPair.Upper().String() {
				tick = &tickers[x]
				break
			} else if !fPair.IsEmpty() {
				continue
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
func (g *Gateio) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !g.SupportsAsset(a) {
		return nil, fmt.Errorf("%s does not support %s", g.Name, a)
	}
	switch a {
	case asset.Spot:
		tradables, err := g.ListAllCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		for x := range tradables {
			p := strings.ToUpper(tradables[x].Base + currency.UnderscoreDelimiter + tradables[x].Quote)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	case asset.Margin, asset.CrossMargin:
		tradables, err := g.GetMarginSupportedCurrencyPairs(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]currency.Pair, 0, len(tradables))
		for x := range tradables {
			p := strings.ToUpper(tradables[x].Base + currency.UnderscoreDelimiter + tradables[x].Quote)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	case asset.Futures:
		btcContracts, err := g.GetAllFutureContracts(ctx, settleBTC)
		if err != nil {
			return nil, err
		}
		usdtContracts, err := g.GetAllFutureContracts(ctx, settleUSDT)
		if err != nil {
			return nil, err
		}
		btcContracts = append(btcContracts, usdtContracts...)
		pairs := make([]currency.Pair, 0, len(btcContracts))
		for x := range btcContracts {
			p := strings.ToUpper(btcContracts[x].Name)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	case asset.DeliveryFutures:
		btcContracts, err := g.GetAllDeliveryContracts(ctx, settleBTC)
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return nil, err
		}
		usdContracts, err := g.GetAllDeliveryContracts(ctx, settleUSD)
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return nil, err
		}
		usdtContracts, err := g.GetAllDeliveryContracts(ctx, settleUSDT)
		if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
			return nil, err
		}
		btcContracts = append(btcContracts, usdtContracts...)
		btcContracts = append(btcContracts, usdContracts...)
		pairs := make([]currency.Pair, 0, len(btcContracts))
		for x := range btcContracts {
			p := strings.ToUpper(btcContracts[x].Name)
			if !g.IsValidPairString(p) {
				continue
			}
			cp, err := currency.NewPairFromString(p)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	case asset.Options:
		underlyings, err := g.GetAllOptionsUnderlyings(ctx)
		if err != nil {
			return nil, err
		}
		pairs := []currency.Pair{}
		for x := range underlyings {
			contracts, err := g.GetAllContractOfUnderlyingWithinExpiryDate(ctx, underlyings[x].Name, time.Time{})
			if err != nil {
				return nil, err
			}
			for c := range contracts {
				if !g.IsValidPairString(contracts[c].Name) {
					continue
				}
				cp, err := currency.NewPairFromString(strings.ReplaceAll(contracts[c].Name, currency.DashDelimiter, currency.UnderscoreDelimiter))
				cp.Quote = currency.NewCode(strings.ReplaceAll(cp.Quote.String(), currency.UnderscoreDelimiter, currency.DashDelimiter))
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, cp)
			}
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
		if len(pairs) == 0 {
			continue
		}
		err = g.UpdatePairs(pairs, assets[x], false, forceUpdate)
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
	var err error
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var tickers []Ticker
		tickers, err = g.GetTickers(ctx, currency.EMPTYPAIR.String(), "")
		if err != nil {
			return err
		}
		for x := range tickers {
			var currencyPair currency.Pair
			currencyPair, err = currency.NewPairFromString(tickers[x].CurrencyPair)
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
	case asset.Futures, asset.DeliveryFutures:
		var tickers []FuturesTicker
		var ticks []FuturesTicker
		for _, settle := range []string{settleBTC, settleUSDT, settleUSD} {
			if a == asset.Futures {
				ticks, err = g.GetFuturesTickers(ctx, settle, currency.EMPTYPAIR.String())
			} else {
				ticks, err = g.GetDeliveryFutureTickers(ctx, settle, currency.EMPTYPAIR)
			}
			if err != nil && !strings.Contains(err.Error(), "404 Not Found") {
				return err
			}
			tickers = append(tickers, ticks...)
		}
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
	case asset.Options:
		pairs, err := g.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		for i := range pairs {
			if pairs[i].Base.IsEmpty() || pairs[i].Quote.IsEmpty() {
				return currency.ErrCurrencyPairEmpty
			}
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
					Last:         tickers[x].LastPrice,
					Ask:          tickers[x].Ask1Price,
					AskSize:      tickers[x].Ask1Size,
					Bid:          tickers[x].Bid1Price,
					BidSize:      tickers[x].Bid1Size,
					Pair:         currencyPair,
					ExchangeName: g.Name,
					AssetType:    a,
				})
				if err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("%s does not support %s", g.Name, a)
	}
	return nil
}

// FetchOrderbook returns orderbook base on the currency pair
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
	fPair.Delimiter = currency.UnderscoreDelimiter
	book.Pair = fPair.Upper()
	fPair = fPair.Upper()
	var orderbookNew *Orderbook
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		orderbookNew, err = g.GetOrderbook(ctx, fPair, "", 0, true)
	case asset.Futures:
		if !strings.HasPrefix(fPair.Quote.String(), currency.USD.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.USDT.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.BTC.Upper().String()) {
			return nil, errUnsupportedSettleValue
		}
		orderbookNew, err = g.GetFuturesOrderbook(ctx, fPair.Quote.String(), fPair.Upper().String(), "", 0, true)
	case asset.DeliveryFutures:
		if !strings.HasPrefix(fPair.Quote.String(), currency.USD.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.USDT.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.BTC.Upper().String()) {
			return nil, errUnsupportedSettleValue
		}
		var quote string
		if strings.HasPrefix(fPair.Quote.String(), currency.USDT.Upper().String()) {
			quote = settleUSDT
		} else {
			quote = settleBTC
		}
		orderbookNew, err = g.GetDeliveryOrderbook(ctx, quote, fPair.Upper().String(), "", 0, true)
	case asset.Options:
		if !strings.HasPrefix(fPair.Quote.String(), currency.USD.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.USDT.Upper().String()) &&
			!strings.HasPrefix(fPair.Quote.String(), currency.BTC.Upper().String()) {
			return nil, errUnsupportedSettleValue
		}
		orderbookNew, err = g.GetOptionsOrderbook(ctx, fPair, "", 0, true)
	}
	if err != nil {
		return book, err
	}
	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}
	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
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
	return orderbook.Get(g.Name, fPair, a)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
func (g *Gateio) UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	var info account.Holdings
	info.Exchange = g.Name
	var err error
	switch a {
	case asset.Spot:
		var balances []SpotAccount
		balances, err = g.GetSpotAccounts(ctx, currency.EMPTYCODE)
		currencies := make([]account.Balance, len(balances))
		if err != nil {
			return info, err
		}
		for x := range balances {
			currencies[x] = account.Balance{
				CurrencyName: currency.NewCode(balances[x].Currency),
				Total:        balances[x].Available - balances[x].Locked,
				Hold:         balances[x].Locked,
				Free:         balances[x].Available,
			}
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType:  a,
			Currencies: currencies,
		})
	case asset.Margin, asset.CrossMargin:
		var balances []MarginAccountItem
		balances, err = g.GetMarginAccountList(ctx, currency.EMPTYPAIR)
		if err != nil {
			return info, err
		}
		var currencies []account.Balance
		for x := range balances {
			currencies = append(currencies, account.Balance{
				CurrencyName: currency.NewCode(balances[x].Base.Currency),
				Total:        balances[x].Base.Available + balances[x].Base.Locked,
				Hold:         balances[x].Base.Locked,
				Free:         balances[x].Base.Available,
			}, account.Balance{
				CurrencyName: currency.NewCode(balances[x].Quote.Currency),
				Total:        balances[x].Quote.Available + balances[x].Quote.Locked,
				Hold:         balances[x].Quote.Locked,
				Free:         balances[x].Quote.Available,
			})
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType:  a,
			Currencies: currencies,
		})
	case asset.Futures, asset.DeliveryFutures:
		currencies := make([]account.Balance, 3)
		settles := []currency.Code{currency.BTC, currency.USD, currency.USDT}
		for x := range settles {
			var balance *FuturesAccount
			if a == asset.Futures {
				balance, err = g.QueryFuturesAccount(ctx, settles[x].String())
			} else {
				balance, err = g.GetDeliveryFuturesAccounts(ctx, settles[x].String())
			}
			if err != nil {
				return info, err
			}
			currencies[0] = account.Balance{
				CurrencyName: currency.NewCode(balance.Currency),
				Total:        balance.Total,
				Hold:         balance.Total - balance.Available,
				Free:         balance.Available,
			}
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType:  a,
			Currencies: currencies,
		})
	case asset.Options:
		var balance *OptionAccount
		balance, err = g.GetOptionAccounts(ctx)
		if err != nil {
			return info, err
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			AssetType: a,
			Currencies: []account.Balance{
				{
					CurrencyName: currency.NewCode(balance.Currency),
					Total:        balance.Total,
					Hold:         balance.Total - balance.Available,
					Free:         balance.Available,
				},
			},
		})
	default:
		return info, fmt.Errorf("%s does not support %s", g.Name, a)
	}
	creds, err := g.GetCredentials(ctx)
	if err != nil {
		return info, err
	}
	err = account.Process(&info, creds)
	if err != nil {
		return info, err
	}
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

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gateio) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (g *Gateio) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	records, err := g.GetWithdrawalRecords(ctx, c, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return nil, err
	}
	withdrawalHistories := make([]exchange.WithdrawalHistory, len(records))
	for x := range records {
		withdrawalHistories[x] = exchange.WithdrawalHistory{
			Status:          records[x].Status,
			TransferID:      records[x].ID,
			Currency:        records[x].Currency,
			Amount:          records[x].Amount,
			CryptoTxID:      records[x].TransactionID,
			CryptoToAddress: records[x].Address,
			Timestamp:       records[x].Timestamp,
		}
	}
	return withdrawalHistories, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (g *Gateio) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	p, err := g.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var tradeData []Trade
		if p.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		var fPair currency.PairFormat
		fPair, err = g.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		tradeData, err = g.GetMarketTrades(ctx, fPair.Format(p), 0, "", false, time.Time{}, time.Time{}, 0)
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(tradeData))
		for i := range tradeData {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp[i] = trade.Data{
				Exchange:     g.Name,
				TID:          tradeData[i].OrderID,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Amount,
				Timestamp:    tradeData[i].CreateTimeMs,
			}
		}
	case asset.Futures:
		if p.Quote != currency.USD &&
			p.Quote != currency.USDT &&
			p.Quote != currency.BTC {
			return nil, errUnsupportedSettleValue
		}
		var futuresTrades []TradingHistoryItem
		futuresTrades, err = g.GetFuturesTradingHistory(ctx, p.Quote.String(), p, 0, 0, "", time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(futuresTrades))
		for i := range futuresTrades {
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(futuresTrades[i].ID, 10),
				Exchange:     g.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        futuresTrades[i].Price,
				Amount:       futuresTrades[i].Size,
				Timestamp:    futuresTrades[i].CreateTime,
			}
		}
	case asset.DeliveryFutures:
		if !strings.HasPrefix(p.Quote.Upper().String(), currency.USD.String()) &&
			!strings.HasPrefix(p.Quote.Upper().String(), currency.USDT.String()) &&
			!strings.HasPrefix(p.Quote.Upper().String(), currency.BTC.String()) {
			return nil, errUnsupportedSettleValue
		}
		var settle string
		settle, err = g.getSettlementFromCurrency(p)
		if err != nil {
			return nil, err
		}
		var deliveryTrades []DeliveryTradingHistory
		deliveryTrades, err = g.GetDeliveryTradingHistory(ctx, settle, p.Upper().String(), 0, "", time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(deliveryTrades))
		for i := range deliveryTrades {
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(deliveryTrades[i].ID, 10),
				Exchange:     g.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        deliveryTrades[i].Price,
				Amount:       deliveryTrades[i].Size,
				Timestamp:    deliveryTrades[i].CreateTime,
			}
		}
	case asset.Options:
		var trades []TradingHistoryItem
		trades, err = g.GetOptionsTradeHistory(ctx, p.Upper().String(), "", 0, 0, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, len(trades))
		for i := range trades {
			resp[i] = trade.Data{
				TID:          strconv.FormatInt(trades[i].ID, 10),
				Exchange:     g.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        trades[i].Price,
				Amount:       trades[i].Size,
				Timestamp:    trades[i].CreateTime,
			}
		}
	default:
		return nil, fmt.Errorf("%s does not support %s", g.Name, a)
	}
	err = g.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (g *Gateio) GetHistoricTrades(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
// TODO: support multiple order types (IOC)
func (g *Gateio) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	var fPair currency.Pair
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	var orderTypeFormat string
	switch s.Side {
	case order.Buy:
		orderTypeFormat = order.Buy.Lower()
	case order.Sell:
		orderTypeFormat = order.Sell.Lower()
	case order.Bid:
		orderTypeFormat = order.Bid.Lower()
	case order.Ask:
		orderTypeFormat = order.Ask.Lower()
	default:
		return nil, errInvalidOrderSide
	}
	fPair, err = g.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		if s.Type != order.Limit {
			return nil, errOnlyLimitOrderType
		}
		sOrder, err := g.PlaceSpotOrder(ctx, &CreateOrderRequestData{
			Side:         orderTypeFormat,
			Type:         s.Type.Lower(),
			Account:      g.assetTypeToString(s.AssetType),
			Amount:       s.Amount,
			Price:        s.Price,
			CurrencyPair: fPair,
			Text:         s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(sOrder.ID)
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(sOrder.Side)
		if err != nil {
			return nil, err
		}
		response.Side = side
		status, err := order.StringToOrderStatus(sOrder.Status)
		if err != nil {
			return nil, err
		}
		response.Status = status
		response.Fee = sOrder.Fee
		response.Pair = fPair
		response.Date = sOrder.CreateTime
		return response, nil
	case asset.Futures:
		if !fPair.Quote.Equal(currency.USD) &&
			!fPair.Quote.Equal(currency.USDT) &&
			!fPair.Quote.Equal(currency.BTC) {
			return nil, errUnsupportedSettleValue
		}
		if orderTypeFormat == "bid" && s.Price < 0 {
			s.Price = -s.Price
		} else if orderTypeFormat == "ask" && s.Price > 0 {
			s.Price = -s.Price
		}
		fOrder, err := g.PlaceFuturesOrder(ctx, &OrderCreateParams{
			Contract:    fPair,
			Size:        s.Amount,
			Price:       s.Price,
			Settle:      fPair.Quote.String(),
			ReduceOnly:  s.ReduceOnly,
			Text:        s.ClientOrderID,
			TimeInForce: "gtc",
		})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(strconv.FormatInt(fOrder.ID, 10))
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(fOrder.Status)
		if err != nil {
			return nil, err
		}
		response.Status = status
		response.Pair = fPair
		response.Date = fOrder.CreateTime
		return response, nil
	case asset.DeliveryFutures:
		if fPair.Quote != currency.USD &&
			fPair.Quote != currency.USDT &&
			fPair.Quote != currency.BTC {
			return nil, errUnsupportedSettleValue
		}
		if orderTypeFormat == "bid" && s.Price < 0 {
			s.Price = -s.Price
		} else if orderTypeFormat == "ask" && s.Price > 0 {
			s.Price = -s.Price
		}
		dOrder, err := g.PlaceDeliveryOrder(ctx, &OrderCreateParams{
			Contract:    fPair,
			Size:        s.Amount,
			Price:       s.Price,
			Settle:      fPair.Quote.String(),
			ReduceOnly:  s.ReduceOnly,
			Text:        s.ClientOrderID,
			TimeInForce: "gtc",
		})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(strconv.FormatInt(dOrder.ID, 10))
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(dOrder.Status)
		if err != nil {
			return nil, err
		}
		response.Status = status
		response.Pair = fPair
		response.Date = dOrder.CreateTime
		return response, nil
	case asset.Options:
		optionOrder, err := g.PlaceOptionOrder(ctx, OptionOrderParam{
			Contract:   fPair.String(),
			OrderSize:  s.Amount,
			Price:      s.Price,
			ReduceOnly: s.ReduceOnly,
			Text:       s.ClientOrderID,
		})
		if err != nil {
			return nil, err
		}
		response, err := s.DeriveSubmitResponse(strconv.FormatInt(optionOrder.OptionOrderID, 10))
		if err != nil {
			return nil, err
		}
		status, err := order.StringToOrderStatus(optionOrder.Status)
		if err != nil {
			return nil, err
		}
		response.Status = status
		response.Pair = fPair
		response.Date = optionOrder.CreateTime
		return response, nil
	default:
		return nil, fmt.Errorf("%s does not support %s", g.Name, s.AssetType)
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (g *Gateio) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gateio) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	fPair, err := g.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	switch o.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		_, err = g.CancelSingleSpotOrder(ctx, o.OrderID, fPair.String(), o.AssetType)
	case asset.Futures, asset.DeliveryFutures:
		if fPair.Quote != currency.USD &&
			fPair.Quote != currency.USDT &&
			fPair.Quote != currency.BTC {
			return errUnsupportedSettleValue
		}
		if o.AssetType == asset.Futures {
			_, err = g.CancelSingleFuturesOrder(ctx, fPair.Quote.String(), o.OrderID)
		} else {
			_, err = g.CancelSingleDeliveryOrder(ctx, fPair.Quote.String(), o.OrderID)
		}
	case asset.Options:
		_, err = g.CancelOptionSingleOrder(ctx, o.OrderID)
	default:
		return fmt.Errorf("%s does not support %s", g.Name, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (g *Gateio) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	var response order.CancelBatchResponse
	if len(o) == 0 {
		return response, errors.New("no cancel order passed")
	}
	var cancelSpotOrdersParam []CancelOrderByIDParam
	a := o[0].AssetType
	for x := range o {
		if a != o[x].AssetType {
			return response, errors.New("cannot cancel orders of different asset types")
		}
		if a == asset.Spot || a == asset.Margin || a == asset.CrossMargin {
			cancelSpotOrdersParam = append(cancelSpotOrdersParam, CancelOrderByIDParam{
				ID:           o[x].OrderID,
				CurrencyPair: o[x].Pair,
			})
			continue
		}
		if err := o[x].Validate(o[x].StandardCancel()); err != nil {
			return response, err
		}
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		loop := int(math.Ceil(float64(len(cancelSpotOrdersParam)) / 10))
		for count := 0; count < loop; count++ {
			var input []CancelOrderByIDParam
			if (count + 1) == loop {
				input = cancelSpotOrdersParam[count*10:]
			} else {
				input = cancelSpotOrdersParam[count*10 : (count*10)+10]
			}
			cancel, err := g.CancelBatchOrdersWithIDList(ctx, input)
			if err != nil {
				return response, err
			}
			for x := range cancel {
				response.Status[cancel[x].ID] = func() string {
					if cancel[x].Succeeded {
						return order.Cancelled.String()
					}
					return ""
				}()
			}
		}
	case asset.Futures:
		for a := range o {
			cancel, err := g.CancelMultipleFuturesOpenOrders(ctx, o[a].Pair, o[a].Side.Lower(), o[a].Pair.Quote.String())
			if err != nil {
				return response, err
			}
			for x := range cancel {
				response.Status[strconv.FormatInt(cancel[x].ID, 10)] = cancel[x].Status
			}
		}
	case asset.DeliveryFutures:
		for a := range o {
			if o[a].Pair.Quote != currency.USD &&
				o[a].Pair.Quote != currency.USDT &&
				o[a].Pair.Quote != currency.BTC {
				return response, errUnsupportedSettleValue
			}
			cancel, err := g.CancelMultipleDeliveryOrders(ctx, o[a].Pair, o[a].Side.Lower(), o[a].Pair.Quote.Lower().String())
			if err != nil {
				return response, err
			}
			for x := range cancel {
				response.Status[strconv.FormatInt(cancel[x].ID, 10)] = cancel[x].Status
			}
		}
	case asset.Options:
		for a := range o {
			cancel, err := g.CancelMultipleOptionOpenOrders(ctx, o[a].Pair, o[a].Pair.String(), o[a].Side.Lower())
			if err != nil {
				return response, err
			}
			for x := range cancel {
				response.Status[strconv.FormatInt(cancel[x].OptionOrderID, 10)] = cancel[x].Status
			}
		}
	default:
		return response, fmt.Errorf("%s does not support %s", g.Name, a)
	}
	return response, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gateio) CancelAllOrders(ctx context.Context, o *order.Cancel) (order.CancelAllResponse, error) {
	var cancelAllOrdersResponse order.CancelAllResponse
	switch o.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		cancel, err := g.CancelMultipleSpotOpenOrders(ctx, currency.EMPTYPAIR, o.AssetType)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for x := range cancel {
			cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[x].ID, 10)] = cancel[0].Status
		}
	case asset.Futures:
		contracts, err := g.FetchTradablePairs(ctx, asset.Futures)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range contracts {
			contracts[i] = contracts[i].Upper()
			if contracts[i].Quote != currency.USD &&
				contracts[i].Quote != currency.USDT &&
				contracts[i].Quote != currency.BTC {
				continue
			}
			cancel, err := g.CancelMultipleFuturesOpenOrders(ctx, contracts[i], o.Side.Lower(), contracts[i].Quote.String())
			if err != nil && len(cancelAllOrdersResponse.Status) != 0 {
				return cancelAllOrdersResponse, err
			}
			for f := range cancel {
				cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[f].ID, 10)] = cancel[f].Status
			}
		}
	case asset.DeliveryFutures:
		contracts, err := g.FetchTradablePairs(ctx, asset.DeliveryFutures)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range contracts {
			contracts[i] = contracts[i].Upper()
			if contracts[i].Quote != currency.USD &&
				contracts[i].Quote != currency.USDT &&
				contracts[i].Quote != currency.BTC {
				continue
			}
			cancel, err := g.CancelMultipleDeliveryOrders(ctx, contracts[i], o.Side.Lower(), contracts[i].Quote.String())
			if err != nil && len(cancelAllOrdersResponse.Status) != 0 {
				return cancelAllOrdersResponse, err
			}
			for f := range cancel {
				cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[f].ID, 10)] = cancel[f].Status
			}
		}
	case asset.Options:
		contracts, err := g.FetchTradablePairs(ctx, asset.DeliveryFutures)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range contracts {
			cancel, err := g.CancelMultipleOptionOpenOrders(ctx, contracts[i], contracts[i].String(), o.Side.Lower())
			if err != nil && len(cancelAllOrdersResponse.Status) != 0 {
				return cancelAllOrdersResponse, err
			}
			for x := range cancel {
				cancelAllOrdersResponse.Status[strconv.FormatInt(cancel[x].OptionOrderID, 10)] = cancel[x].Status
			}
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%s does not support %s", g.Name, o.AssetType)
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (g *Gateio) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, a asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	pair, err := g.FormatExchangeCurrency(pair, a)
	if err != nil {
		return orderDetail, err
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		spotOrder, err := g.GetSpotOrder(ctx, orderID, pair.String(), a)
		if err != nil {
			return orderDetail, err
		}
		side, err := order.StringToOrderSide(spotOrder.Side)
		if err != nil {
			return orderDetail, err
		}
		orderType, err := order.StringToOrderType(spotOrder.Type)
		if err != nil {
			return orderDetail, err
		}
		orderStatus, err := order.StringToOrderStatus(spotOrder.Status)
		if err != nil {
			return orderDetail, err
		}
		return order.Detail{
			Amount:         spotOrder.Amount,
			Exchange:       g.Name,
			OrderID:        spotOrder.ID,
			Side:           side,
			Type:           orderType,
			Pair:           pair,
			Cost:           spotOrder.Fee,
			AssetType:      a,
			Status:         orderStatus,
			Price:          spotOrder.Price,
			ExecutedAmount: spotOrder.Amount - spotOrder.Left,
			Date:           spotOrder.CreateTimeMs,
			LastUpdated:    spotOrder.UpdateTimeMs,
		}, nil
	case asset.Futures, asset.DeliveryFutures:
		pair = pair.Upper()
		if pair.Quote != currency.USD &&
			pair.Quote != currency.USDT &&
			pair.Quote != currency.BTC {
			return orderDetail, errUnsupportedSettleValue
		}
		var fOrder *Order
		var err error
		if asset.Futures == a {
			fOrder, err = g.GetSingleFuturesOrder(ctx, pair.Quote.Lower().String(), orderID)
		} else {
			fOrder, err = g.GetSingleDeliveryOrder(ctx, pair.Quote.Lower().String(), orderID)
		}
		if err != nil {
			return orderDetail, err
		}
		orderStatus, err := order.StringToOrderStatus(fOrder.Status)
		if err != nil {
			return orderDetail, err
		}
		pair, err = currency.NewPairFromString(fOrder.Contract)
		if err != nil {
			return orderDetail, err
		}
		return order.Detail{
			Amount:         fOrder.Size,
			ExecutedAmount: fOrder.Size - fOrder.Left,
			Exchange:       g.Name,
			OrderID:        orderID,
			Status:         orderStatus,
			Price:          fOrder.Price,
			Date:           fOrder.CreateTime,
			LastUpdated:    fOrder.FinishTime,
			Pair:           pair,
			AssetType:      a,
		}, nil
	case asset.Options:
		optionOrder, err := g.GetSingleOptionOrder(ctx, orderID)
		if err != nil {
			return orderDetail, err
		}
		orderStatus, err := order.StringToOrderStatus(optionOrder.Status)
		if err != nil {
			return orderDetail, err
		}
		pair, err = currency.NewPairFromString(optionOrder.Contract)
		if err != nil {
			return orderDetail, err
		}
		return order.Detail{
			Amount:         optionOrder.Size,
			ExecutedAmount: optionOrder.Size - optionOrder.Left,
			Exchange:       g.Name,
			OrderID:        orderID,
			Status:         orderStatus,
			Price:          optionOrder.Price,
			Date:           optionOrder.CreateTime,
			LastUpdated:    optionOrder.FinishTime,
			Pair:           pair,
			AssetType:      a,
		}, nil
	default:
		return orderDetail, fmt.Errorf("%s does not support %s", g.Name, a)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gateio) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := g.GenerateCurrencyDepositAddress(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	if chain != "" {
		for x := range addr.MultichainAddresses {
			if addr.MultichainAddresses[x].ObtainFailed == 1 {
				continue
			}
			if addr.MultichainAddresses[x].Chain == chain {
				return &deposit.Address{
					Chain:   addr.MultichainAddresses[x].Chain,
					Address: addr.MultichainAddresses[x].Address,
					Tag:     addr.MultichainAddresses[x].PaymentName,
				}, nil
			}
		}
		return nil, fmt.Errorf("network %s not found", chain)
	}
	return &deposit.Address{
		Address: addr.Address,
		Chain:   chain,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gateio) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	response, err := g.WithdrawCurrency(ctx,
		WithdrawalRequestParam{
			Amount:   withdrawRequest.Amount,
			Currency: withdrawRequest.Currency,
			Address:  withdrawRequest.Crypto.Address,
		},
	)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   response.Chain,
		ID:     response.TransactionID,
		Status: response.Status,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (g *Gateio) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gateio) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !g.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gateio) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	var orders []order.Detail
	format, err := g.GetPairFormat(req.AssetType, false)
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var spotOrders []SpotOrdersDetail
		spotOrders, err = g.GetSpotOpenOrders(ctx, 0, 0, req.AssetType == asset.CrossMargin)
		if err != nil {
			return nil, err
		}
		for x := range spotOrders {
			var symbol currency.Pair
			symbol, err = currency.NewPairDelimiter(spotOrders[x].CurrencyPair, format.Delimiter)
			if err != nil {
				return nil, err
			}
			for y := range spotOrders[x].Orders {
				if spotOrders[x].Orders[y].Status != "open" {
					continue
				}
				var side order.Side
				side, err = order.StringToOrderSide(spotOrders[x].Orders[x].Side)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
				}
				if req.Side != order.AnySide && req.Side != side {
					continue
				}
				var status order.Status
				status, err = order.StringToOrderStatus(spotOrders[x].Orders[y].Status)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
				}
				orders = append(orders, order.Detail{
					Side:            side,
					Status:          status,
					Pair:            symbol,
					OrderID:         spotOrders[x].Orders[y].ID,
					Amount:          spotOrders[x].Orders[y].Amount,
					ExecutedAmount:  spotOrders[x].Orders[y].Amount - spotOrders[x].Orders[y].Left,
					RemainingAmount: spotOrders[x].Orders[y].Left,
					Price:           spotOrders[x].Orders[y].Price,
					Date:            spotOrders[x].Orders[y].CreateTimeMs,
					LastUpdated:     spotOrders[x].Orders[y].UpdateTimeMs,
					Exchange:        g.Name,
					AssetType:       req.AssetType,
				})
			}
		}
	case asset.Futures, asset.DeliveryFutures:
		var pairs []currency.Pair
		if len(req.Pairs) == 0 {
			pairs, err = g.FetchTradablePairs(ctx, req.AssetType)
			if err != nil {
				return nil, err
			}
		}
		for z := range pairs {
			if pairs[z].Quote != currency.USD &&
				pairs[z].Quote != currency.USDT &&
				pairs[z].Quote != currency.BTC {
				return nil, errUnsupportedSettleValue
			}
			var futuresOrders []Order
			if req.AssetType == asset.Futures {
				futuresOrders, err = g.GetFuturesOrders(ctx, pairs[z], "open", 0, 0, "", 0, pairs[z].Quote.Lower().String())
			} else {
				futuresOrders, err = g.GetDeliveryOrders(ctx, pairs[z], "open", 0, 0, "", 0, pairs[z].Quote.Lower().String())
			}
			if err != nil {
				return nil, err
			}
			for x := range futuresOrders {
				if futuresOrders[x].Status != "open" {
					continue
				}
				var status order.Status
				status, err = order.StringToOrderStatus(futuresOrders[x].Status)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
				}
				orders = append(orders, order.Detail{
					Status:          status,
					Amount:          futuresOrders[x].Size,
					Pair:            pairs[x],
					OrderID:         strconv.FormatInt(futuresOrders[x].ID, 10),
					Price:           futuresOrders[x].Price,
					ExecutedAmount:  futuresOrders[x].Size - futuresOrders[x].Left,
					RemainingAmount: futuresOrders[x].Left,
					LastUpdated:     futuresOrders[x].FinishTime,
					Date:            futuresOrders[x].CreateTime,
					Exchange:        g.Name,
					AssetType:       req.AssetType,
				})
			}
		}
	case asset.Options:
		var optionsOrders []OptionOrderResponse
		optionsOrders, err = g.GetOptionFuturesOrders(ctx, "", "", "open", 0, 0, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range optionsOrders {
			var currencyPair currency.Pair
			var status order.Status
			currencyPair, err = currency.NewPairFromString(optionsOrders[x].Contract)
			if err != nil {
				return nil, err
			}
			status, err = order.StringToOrderStatus(optionsOrders[x].Status)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Status:          status,
				Amount:          optionsOrders[x].Size,
				Pair:            currencyPair,
				OrderID:         strconv.FormatInt(optionsOrders[x].OptionOrderID, 10),
				Price:           optionsOrders[x].Price,
				ExecutedAmount:  optionsOrders[x].Size - optionsOrders[x].Left,
				RemainingAmount: optionsOrders[x].Left,
				LastUpdated:     optionsOrders[x].FinishTime,
				Date:            optionsOrders[x].CreateTime,
				Exchange:        g.Name,
				AssetType:       req.AssetType,
			})
		}
	default:
		return nil, fmt.Errorf("%s does not support %s", g.Name, req.AssetType)
	}
	return req.Filter(g.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gateio) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	format, err := g.GetPairFormat(req.AssetType, true)
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		for x := range req.Pairs {
			var spotOrders []SpotPersonalTradeHistory
			spotOrders, err = g.GetPersonalTradingHistory(ctx, req.Pairs[x], req.OrderID, 0, 0, req.AssetType, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			for o := range spotOrders {
				var side order.Side
				side, err = order.StringToOrderSide(spotOrders[o].Side)
				if err != nil {
					return nil, err
				}
				req.Pairs[x], err = g.FormatExchangeCurrency(req.Pairs[x], req.AssetType)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					OrderID:        spotOrders[o].OrderID,
					Amount:         spotOrders[o].Amount,
					ExecutedAmount: spotOrders[o].Amount,
					Price:          spotOrders[o].Price,
					Date:           spotOrders[o].CreateTime,
					Side:           side,
					Exchange:       g.Name,
					Pair:           req.Pairs[x],
					AssetType:      req.AssetType,
					Fee:            spotOrders[o].Fee,
					FeeAsset:       currency.NewCode(spotOrders[o].FeeCurrency),
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
		}
	case asset.Futures, asset.DeliveryFutures:
		for x := range req.Pairs {
			if req.Pairs[x].Quote != currency.USD &&
				req.Pairs[x].Quote != currency.USDT &&
				req.Pairs[x].Quote != currency.BTC {
				return nil, errUnsupportedSettleValue
			}
			var futuresOrder []TradingHistoryItem
			if req.AssetType == asset.Futures {
				futuresOrder, err = g.GetMyPersonalTradingHistory(ctx, req.Pairs[x].Quote.String(), req.Pairs[x], req.OrderID, 0, 0, 0, "")
			} else {
				futuresOrder, err = g.GetDeliveryPersonalTradingHistory(ctx, req.Pairs[x].Quote.String(), req.Pairs[x], req.OrderID, 0, 0, 0, "")
			}
			if err != nil {
				return nil, err
			}
			for o := range futuresOrder {
				detail := order.Detail{
					OrderID:   strconv.FormatInt(futuresOrder[o].ID, 10),
					Amount:    futuresOrder[o].Size,
					Price:     futuresOrder[o].Price,
					Date:      futuresOrder[o].CreateTime,
					Exchange:  g.Name,
					Pair:      req.Pairs[x].Format(format),
					AssetType: req.AssetType,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
		}
	case asset.Options:
		for x := range req.Pairs {
			var optionOrders []OptionTradingHistory
			optionOrders, err = g.GetOptionsPersonalTradingHistory(ctx, format.Format(req.Pairs[x]), req.Pairs[x].Upper().String(), 0, 0, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			for o := range optionOrders {
				detail := order.Detail{
					OrderID:   strconv.FormatInt(optionOrders[o].OrderID, 10),
					Amount:    optionOrders[o].Size,
					Price:     optionOrders[o].Price,
					Date:      optionOrders[o].CreateTime,
					Exchange:  g.Name,
					Pair:      req.Pairs[x].Format(format),
					AssetType: req.AssetType,
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
		}
	default:
		return nil, fmt.Errorf("%s does not support %s", g.Name, req.AssetType)
	}
	return req.Filter(g.Name, orders), nil
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (g *Gateio) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	formattedPair, err := g.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}
	formattedPair = formattedPair.Upper()
	err = g.ValidateKline(formattedPair, a, interval)
	if err != nil {
		return kline.Item{}, err
	}
	klineData := kline.Item{
		Interval: interval,
		Asset:    a,
		Pair:     formattedPair.Upper(),
		Exchange: g.Name,
	}
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		var candles []Candlestick
		if formattedPair.IsEmpty() {
			return klineData, currency.ErrCurrencyPairEmpty
		}
		var fPair currency.PairFormat
		fPair, err = g.GetPairFormat(a, true)
		if err != nil {
			return klineData, err
		}
		candles, err = g.GetCandlesticks(ctx, fPair.Format(formattedPair), 0, start, end, interval)
		if err != nil {
			return klineData, err
		}
		klineData.Candles = make([]kline.Candle, len(candles))
		for x := range candles {
			klineData.Candles[x] = kline.Candle{
				Time:   candles[x].Timestamp,
				Open:   candles[x].OpenPrice,
				High:   candles[x].HighestPrice,
				Low:    candles[x].LowestPrice,
				Close:  candles[x].ClosePrice,
				Volume: candles[x].QuoteCcyVolume,
			}
		}
		return klineData, nil
	case asset.Futures, asset.DeliveryFutures:
		if formattedPair.Quote != currency.USD &&
			formattedPair.Quote != currency.USDT &&
			formattedPair.Quote != currency.BTC {
			return klineData, errUnsupportedSettleValue
		}

		var candles []FuturesCandlestick
		if a == asset.Futures {
			candles, err = g.GetFuturesCandlesticks(ctx, formattedPair.Quote.Lower().String(), formattedPair.String(), start, end, 0, interval)
		} else {
			candles, err = g.GetDeliveryFuturesCandlesticks(ctx, formattedPair.Quote.Lower().String(), formattedPair.Upper().String(), start, end, 0, interval)
		}
		if err != nil {
			return klineData, err
		}
		klineData.Candles = make([]kline.Candle, len(candles))
		for x := range candles {
			klineData.Candles[x] = kline.Candle{
				Time:   candles[x].Timestamp,
				Open:   candles[x].OpenPrice,
				High:   candles[x].HighestPrice,
				Low:    candles[x].LowestPrice,
				Close:  candles[x].ClosePrice,
				Volume: candles[x].Volume,
			}
		}
		return klineData, nil
	case asset.Options:
		candles, err := g.GetOptionFuturesCandlesticks(ctx, formattedPair.Upper().String(), 0, start, end, interval)
		if err != nil {
			return klineData, err
		}
		klineData.Candles = make([]kline.Candle, len(candles))
		for x := range candles {
			klineData.Candles[x] = kline.Candle{
				Time:   candles[x].Timestamp,
				Open:   candles[x].OpenPrice,
				High:   candles[x].HighestPrice,
				Low:    candles[x].LowestPrice,
				Close:  candles[x].ClosePrice,
				Volume: candles[x].Volume,
			}
		}
		return klineData, nil
	default:
		return klineData, fmt.Errorf("%s does not support %s", g.Name, a)
	}
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (g *Gateio) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	err := g.ValidateKline(pair, a, interval)
	if err != nil {
		return kline.Item{}, err
	}
	var formattedPair currency.Pair
	formattedPair, err = g.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}
	klineData := kline.Item{
		Interval: interval,
		Asset:    a,
		Pair:     formattedPair.Upper(),
		Exchange: g.Name,
	}
	var dates *kline.IntervalRangeHolder
	dates, err = kline.CalculateCandleDateRanges(start, end, interval, g.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	var candlestickItems []kline.Candle
	var fPair currency.PairFormat
	fPair, err = g.GetPairFormat(a, true)
	if err != nil {
		return kline.Item{}, err
	}
	for b := range dates.Ranges {
		switch a {
		case asset.Spot, asset.Margin, asset.CrossMargin:
			var candles []Candlestick
			if formattedPair.IsEmpty() {
				return klineData, currency.ErrCurrencyPairEmpty
			}
			candles, err = g.GetCandlesticks(ctx, fPair.Format(formattedPair), 0, dates.Ranges[b].Start.Time, dates.Ranges[b].End.Time, interval)
			if err != nil {
				return klineData, err
			}
			for x := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   candles[x].Timestamp,
					Open:   candles[x].OpenPrice,
					High:   candles[x].HighestPrice,
					Low:    candles[x].LowestPrice,
					Close:  candles[x].ClosePrice,
					Volume: candles[x].QuoteCcyVolume,
				})
			}
		case asset.Futures, asset.DeliveryFutures:
			if formattedPair.Quote != currency.USD &&
				formattedPair.Quote != currency.USDT &&
				formattedPair.Quote != currency.BTC {
				return klineData, errUnsupportedSettleValue
			}

			var candles []FuturesCandlestick
			if a == asset.Futures {
				candles, err = g.GetFuturesCandlesticks(ctx, formattedPair.Quote.Lower().String(), formattedPair.String(), dates.Ranges[b].Start.Time, dates.Ranges[b].End.Time, uint64(g.Features.Enabled.Kline.ResultLimit), interval)
			} else {
				candles, err = g.GetDeliveryFuturesCandlesticks(ctx, formattedPair.Quote.Lower().String(), formattedPair.Upper().String(), dates.Ranges[b].Start.Time, dates.Ranges[b].End.Time, uint64(g.Features.Enabled.Kline.ResultLimit), interval)
			}
			if err != nil {
				return klineData, err
			}
			for x := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   candles[x].Timestamp,
					Open:   candles[x].OpenPrice,
					High:   candles[x].HighestPrice,
					Low:    candles[x].LowestPrice,
					Close:  candles[x].ClosePrice,
					Volume: candles[x].Volume,
				})
			}
			return klineData, nil
		case asset.Options:
			candles, err := g.GetOptionFuturesCandlesticks(ctx, formattedPair.Upper().String(), uint64(g.Features.Enabled.Kline.ResultLimit), start, end, interval)
			if err != nil {
				return klineData, err
			}
			for x := range candles {
				candlestickItems = append(candlestickItems, kline.Candle{
					Time:   candles[x].Timestamp,
					Open:   candles[x].OpenPrice,
					High:   candles[x].HighestPrice,
					Low:    candles[x].LowestPrice,
					Close:  candles[x].ClosePrice,
					Volume: candles[x].Volume,
				})
			}
			return klineData, nil
		default:
			return klineData, fmt.Errorf("%s does not support %s", g.Name, a)
		}
	}
	klineData.Candles = candlestickItems
	if start.IsZero() || end.IsZero() {
		klineData.SortCandlesByTimestamp(false)
		klineData.RemoveOutsideRange(start, end)
	}
	return klineData, nil
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (g *Gateio) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	chains, err := g.ListCurrencyChain(ctx, cryptocurrency.Upper())
	if err != nil {
		return nil, err
	}
	availableChains := make([]string, 0, len(chains))
	for x := range chains {
		if chains[x].IsDisabled == 0 {
			availableChains = append(availableChains, chains[x].Chain)
		}
	}
	return availableChains, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (g *Gateio) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := g.UpdateAccountInfo(ctx, assetType)
	return g.CheckTransientError(err)
}
