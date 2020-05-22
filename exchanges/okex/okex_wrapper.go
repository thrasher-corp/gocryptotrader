package okex

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	delimiterDash       = "-"
	delimiterUnderscore = "_"
)

// GetDefaultConfig returns a default exchange config
func (o *OKEX) GetDefaultConfig() (*config.ExchangeConfig, error) {
	o.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = o.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = o.BaseCurrencies

	err := o.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if o.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = o.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults method assignes the default values for OKEX
func (o *OKEX) SetDefaults() {
	o.SetErrorDefaults()
	o.SetCheckVarDefaults()
	o.Name = okExExchangeName
	o.Enabled = true
	o.Verbose = true
	o.API.CredentialsValidator.RequiresKey = true
	o.API.CredentialsValidator.RequiresSecret = true
	o.API.CredentialsValidator.RequiresClientID = true

	o.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
			asset.Futures,
			asset.PerpetualSwap,
			asset.Index,
		},
	}
	// Same format used for perpetual swap and futures
	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: delimiterDash,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: delimiterUnderscore,
		},
	}
	o.CurrencyPairs.Store(asset.PerpetualSwap, fmt1)
	o.CurrencyPairs.Store(asset.Futures, fmt1)

	index := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: delimiterDash,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: delimiterDash,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: delimiterDash,
		},
	}
	o.CurrencyPairs.Store(asset.Spot, spot)
	o.CurrencyPairs.Store(asset.Index, index)

	o.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				CancelOrders:        true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrders:              true,
				GetOrder:               true,
				AccountBalance:         true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			KlineCapabilities: kline.ExchangeCapabilitiesSupported{
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
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.ThreeDay.Word():   true,
					kline.OneWeek.Word():    true,
				},
				ResultLimit: 1440,
			},
		},
	}

	o.Requester = request.New(o.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		// TODO: Specify each individual endpoint rate limits as per docs
		request.WithLimiter(request.NewBasicRateLimit(okExRateInterval, okExRequestRate)),
	)

	o.API.Endpoints.URLDefault = okExAPIURL
	o.API.Endpoints.URL = okExAPIURL
	o.API.Endpoints.WebsocketURL = OkExWebsocketURL
	o.Websocket = wshandler.New()
	o.APIVersion = okExAPIVersion
	o.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	o.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	o.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Start starts the OKGroup go routine
func (o *OKEX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (o *OKEX) Run() {
	if o.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			o.Name,
			common.IsEnabled(o.Websocket.IsEnabled()),
			o.API.Endpoints.WebsocketURL)
	}

	delim := o.GetPairFormat(asset.Spot, false).Delimiter
	forceUpdate := false
	if !common.StringDataContains(o.GetEnabledPairs(asset.Spot).Strings(), delim) ||
		!common.StringDataContains(o.GetAvailablePairs(asset.Spot).Strings(), delim) {
		forceUpdate = true
		enabledPairs := currency.NewPairsFromStrings(
			[]string{currency.BTC.String() + delim + currency.USDT.String()},
		)
		log.Warnf(log.ExchangeSys,
			"Enabled pairs for %v reset due to config upgrade, please enable the ones you would like again.",
			o.Name)

		err := o.UpdatePairs(enabledPairs, asset.Spot, true, forceUpdate)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies.\n",
				o.Name)
			return
		}
	}

	if !o.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := o.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (o *OKEX) FetchTradablePairs(i asset.Item) ([]string, error) {
	var pairs []string
	switch i {
	case asset.Spot:
		prods, err := o.GetSpotTokenPairDetails()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			pairs = append(pairs,
				currency.NewPairWithDelimiter(prods[x].BaseCurrency,
					prods[x].QuoteCurrency,
					o.GetPairFormat(i, false).Delimiter).String())
		}
		return pairs, nil
	case asset.Futures:
		prods, err := o.GetFuturesContractInformation()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			p := strings.Split(prods[x].InstrumentID, delimiterDash)
			pairs = append(pairs,
				p[0]+delimiterDash+p[1]+o.GetPairFormat(i, false).Delimiter+p[2])
		}
		return pairs, nil

	case asset.PerpetualSwap:
		prods, err := o.GetSwapContractInformation()
		if err != nil {
			return nil, err
		}

		for x := range prods {
			pairs = append(pairs,
				prods[x].UnderlyingIndex+
					delimiterDash+
					prods[x].QuoteCurrency+
					o.GetPairFormat(i, false).Delimiter+
					"SWAP")
		}
		return pairs, nil
	case asset.Index:
		// This is updated in futures index
		return nil, errors.New("index updated in futures")
	}

	return nil, fmt.Errorf("%s invalid asset type", o.Name)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (o *OKEX) UpdateTradablePairs(forceUpdate bool) error {
	for x := range o.CurrencyPairs.AssetTypes {
		if o.CurrencyPairs.AssetTypes[x] == asset.Index {
			// Update from futures
			continue
		}

		pairs, err := o.FetchTradablePairs(o.CurrencyPairs.AssetTypes[x])
		if err != nil {
			return err
		}

		if o.CurrencyPairs.AssetTypes[x] == asset.Futures {
			var indexPairs []string
			for i := range pairs {
				indexPairs = append(indexPairs,
					strings.Split(pairs[i], delimiterUnderscore)[0])
			}
			err = o.UpdatePairs(currency.NewPairsFromStrings(indexPairs),
				asset.Index,
				false,
				forceUpdate)
			if err != nil {
				return err
			}
		}

		err = o.UpdatePairs(currency.NewPairsFromStrings(pairs),
			o.CurrencyPairs.AssetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKEX) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	switch assetType {
	case asset.Spot:
		resp, err := o.GetSpotAllTokenPairsInformation()
		if err != nil {
			return tickerPrice, err
		}
		for j := range resp {
			if !o.GetEnabledPairs(assetType).Contains(resp[j].InstrumentID, true) {
				continue
			}
			tickerPrice = &ticker.Price{
				Last:        resp[j].Last,
				High:        resp[j].High24h,
				Low:         resp[j].Low24h,
				Bid:         resp[j].BestBid,
				Ask:         resp[j].BestAsk,
				Volume:      resp[j].BaseVolume24h,
				QuoteVolume: resp[j].QuoteVolume24h,
				Open:        resp[j].Open24h,
				Pair:        resp[j].InstrumentID,
				LastUpdated: resp[j].Timestamp,
			}
			err = ticker.ProcessTicker(o.Name, tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}

	case asset.PerpetualSwap:
		resp, err := o.GetAllSwapTokensInformation()
		if err != nil {
			return nil, err
		}

		for j := range resp {
			p := strings.Split(resp[j].InstrumentID, delimiterDash)
			nC := currency.NewPairWithDelimiter(p[0]+delimiterDash+p[1],
				p[2],
				delimiterUnderscore)
			if !o.GetEnabledPairs(assetType).Contains(nC, true) {
				continue
			}
			tickerPrice = &ticker.Price{
				Last:        resp[j].Last,
				High:        resp[j].High24H,
				Low:         resp[j].Low24H,
				Bid:         resp[j].BestBid,
				Ask:         resp[j].BestAsk,
				Volume:      resp[j].Volume24H,
				Pair:        nC,
				LastUpdated: resp[j].Timestamp,
			}
			err = ticker.ProcessTicker(o.Name, tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}

	case asset.Futures:
		resp, err := o.GetAllFuturesTokenInfo()
		if err != nil {
			return nil, err
		}

		for j := range resp {
			p := strings.Split(resp[j].InstrumentID, delimiterDash)
			nC := currency.NewPairWithDelimiter(p[0]+delimiterDash+p[1],
				p[2],
				delimiterUnderscore)
			if !o.GetEnabledPairs(assetType).Contains(nC, true) {
				continue
			}
			tickerPrice = &ticker.Price{
				Last:        resp[j].Last,
				High:        resp[j].High24h,
				Low:         resp[j].Low24h,
				Bid:         resp[j].BestBid,
				Ask:         resp[j].BestAsk,
				Volume:      resp[j].Volume24h,
				Pair:        nC,
				LastUpdated: resp[j].Timestamp,
			}
			err = ticker.ProcessTicker(o.Name, tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}
	}

	return ticker.GetTicker(o.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (o *OKEX) FetchTicker(p currency.Pair, assetType asset.Item) (tickerData *ticker.Price, err error) {
	if assetType == asset.Index {
		return tickerData, errors.New("ticker fetching not supported for index")
	}
	tickerData, err = ticker.GetTicker(o.Name, p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (o *OKEX) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if !o.KlineIntervalEnabled(interval) {
		return kline.Item{}, kline.ErrorKline{
			Interval: interval,
		}
	}

	req := &okgroup.GetSpotMarketDataRequest{
		Asset:        a,
		Start:        start.UTC().Format(time.RFC3339),
		End:          end.UTC().Format(time.RFC3339),
		Granularity:  o.FormatExchangeKlineInterval(interval),
		InstrumentID: o.FormatExchangeCurrency(pair, a).String(),
	}

	candles, err := o.GetSpotMarketData(req)
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: o.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	for x := range candles {
		t := candles[x].([]interface{})
		tempCandle := kline.Candle{}
		v, ok := t[0].(string)
		if !ok {
			return kline.Item{}, errors.New("unexpected value receved")
		}
		tempCandle.Time, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return kline.Item{}, err
		}
		tempCandle.Open, err = convert.FloatFromString(t[1])
		if err != nil {
			return kline.Item{}, err
		}
		tempCandle.High, err = convert.FloatFromString(t[2])
		if err != nil {
			return kline.Item{}, err
		}

		tempCandle.Low, err = convert.FloatFromString(t[3])
		if err != nil {
			return kline.Item{}, err
		}

		tempCandle.Close, err = convert.FloatFromString(t[4])
		if err != nil {
			return kline.Item{}, err
		}

		tempCandle.Volume, err = convert.FloatFromString(t[5])
		if err != nil {
			return kline.Item{}, err
		}
		ret.Candles = append(ret.Candles, tempCandle)
	}
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (o *OKEX) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if !o.KlineIntervalEnabled(interval) {
		return kline.Item{}, kline.ErrorKline{
			Interval: interval,
		}
	}

	ret := kline.Item{
		Exchange: o.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	dates := kline.CalcDateRanges(start, end, interval, o.Features.Enabled.Kline.ResultLimit)
	for x := range dates {
		req := &okgroup.GetSpotMarketDataRequest{
			Asset:        a,
			Start:        dates[x].Start.UTC().Format(time.RFC3339),
			End:          dates[x].End.UTC().Format(time.RFC3339),
			Granularity:  o.FormatExchangeKlineInterval(interval),
			InstrumentID: o.FormatExchangeCurrency(pair, a).String(),
		}

		candles, err := o.GetSpotMarketData(req)
		if err != nil {
			return kline.Item{}, err
		}

		for x := range candles {
			t := candles[x].([]interface{})
			tempCandle := kline.Candle{}
			v, ok := t[0].(string)
			if !ok {
				return kline.Item{}, errors.New("unexpected value received")
			}
			tempCandle.Time, err = time.Parse(time.RFC3339, v)
			if err != nil {
				return kline.Item{}, err
			}
			tempCandle.Open, err = convert.FloatFromString(t[1])
			if err != nil {
				return kline.Item{}, err
			}
			tempCandle.High, err = convert.FloatFromString(t[2])
			if err != nil {
				return kline.Item{}, err
			}

			tempCandle.Low, err = convert.FloatFromString(t[3])
			if err != nil {
				return kline.Item{}, err
			}

			tempCandle.Close, err = convert.FloatFromString(t[4])
			if err != nil {
				return kline.Item{}, err
			}

			tempCandle.Volume, err = convert.FloatFromString(t[5])
			if err != nil {
				return kline.Item{}, err
			}
			ret.Candles = append(ret.Candles, tempCandle)
		}
	}
	return ret, nil
}
