package binance

import (
	"errors"
	"fmt"
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
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "_",
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
	err = b.StoreAssetPairFormat(asset.CoinMarginedFutures, coinFutures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.USDTMarginedFutures, usdtFutures)
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

	b.API.Endpoints.URLDefault = apiURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.Websocket = stream.New()
	b.API.Endpoints.WebsocketURL = binanceDefaultWebsocketURL
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Binance) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
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
		RunningURL:                       exch.API.Endpoints.WebsocketURL,
		Connector:                        b.WsConnect,
		Subscriber:                       b.Subscribe,
		UnSubscriber:                     b.Unsubscribe,
		GenerateSubscriptions:            b.GenerateSubscriptions,
		Features:                         &b.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.WebsocketOrderbookBufferLimit,
		SortBuffer:                       true,
		SortBufferByUpdateIDs:            true,
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
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
	var pairs []string
	switch a {
	case asset.Spot, asset.Margin:
		info, err := b.GetExchangeInfo()
		if err != nil {
			return nil, err
		}
		format, err := b.GetPairFormat(a, false)
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
		fmt.Printf("HILOOOOOOOOOOO\n\n\n")
		cInfo, err := b.FuturesExchangeInfo()
		fmt.Println(cInfo)
		if err != nil {
			return pairs, nil
		}
		for z := range cInfo.Symbols {
			if cInfo.Symbols[z].ContractStatus == "TRADING" {
				pairs = append(pairs, cInfo.Symbols[z].Symbol)
			}
		}
	case asset.USDTMarginedFutures:
		uInfo, err := b.UExchangeInfo()
		if err != nil {
			return pairs, nil
		}
		for u := range uInfo.Symbols {
			if uInfo.Symbols[u].Status == "TRADING" {
				pairs = append(pairs, uInfo.Symbols[u].Symbol)
			}
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Binance) UpdateTradablePairs(forceUpdate bool) error {
	assetTypes := b.GetAssetTypes()
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
	if !b.SupportsAsset(assetType) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", assetType, b.Name)
	}
	switch assetType {
	case asset.Spot, asset.Margin:
		tick, err := b.GetTickers()
		if err != nil {
			return nil, err
		}

		pairs, err := b.GetEnabledPairs(assetType)
		if err != nil {
			return nil, err
		}

		for i := range pairs {

			pairFmt, err := b.FormatExchangeCurrency(pairs[i], assetType)
			if err != nil {
				return nil, err
			}

			for y := range tick {

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
					Pair:         pairs[i],
					ExchangeName: b.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return nil, err
				}
			}
		}
	case asset.USDTMarginedFutures:
		tick, err := b.U24HTickerPriceChangeStats("")
		if err != nil {
			return nil, err
		}

		pairs, err := b.GetEnabledPairs(assetType)
		if err != nil {
			return nil, err
		}
		for i := range pairs {

			pairFmt, err := b.FormatExchangeCurrency(pairs[i], assetType)
			if err != nil {
				return nil, err
			}

			for y := range tick {

				if tick[y].Symbol != pairFmt.String() {
					continue
				}

				tickData, err := b.USymbolOrderbookTicker(tick[y].Symbol)
				if err != nil {
					return nil, err
				}

				if len(tickData) != 1 {
					return nil, fmt.Errorf("invalid tickData response: only requested tick data for the given symbol")
				}

				err = ticker.ProcessTicker(&ticker.Price{
					Last:         tick[y].LastPrice,
					High:         tick[y].HighPrice,
					Low:          tick[y].LowPrice,
					Bid:          tickData[0].BidPrice,
					Ask:          tickData[0].AskPrice,
					Volume:       tick[y].Volume,
					QuoteVolume:  tick[y].QuoteVolume,
					Open:         tick[y].OpenPrice,
					Close:        tick[y].PrevClosePrice,
					Pair:         pairs[i],
					ExchangeName: b.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return nil, err
				}
			}
		}
	case asset.CoinMarginedFutures:
		tick, err := b.GetFuturesSwapTickerChangeStats("", "")
		if err != nil {
			return nil, err
		}

		pairs, err := b.GetEnabledPairs(assetType)
		if err != nil {
			return nil, err
		}

		for i := range pairs {

			pairFmt, err := b.FormatExchangeCurrency(pairs[i], assetType)
			if err != nil {
				return nil, err
			}

			for y := range tick {

				if tick[y].Symbol != pairFmt.String() {
					continue
				}

				tickData, err := b.GetFuturesOrderbookTicker(tick[y].Symbol, "")
				if err != nil {
					return nil, err
				}

				if len(tickData) != 1 {
					return nil, fmt.Errorf("invalid tickData response: only requested tick data for the given symbol")
				}

				err = ticker.ProcessTicker(&ticker.Price{
					Last:         tick[y].LastPrice,
					High:         tick[y].HighPrice,
					Low:          tick[y].LowPrice,
					Bid:          tickData[0].BidPrice,
					Ask:          tickData[0].AskPrice,
					Volume:       tick[y].Volume,
					QuoteVolume:  tick[y].QuoteVolume,
					Open:         tick[y].OpenPrice,
					Close:        tick[y].PrevClosePrice,
					Pair:         pairs[i],
					ExchangeName: b.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Binance) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.Name, p, assetType)
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
	fpair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	orderBook := new(orderbook.Base)
	orderBook.Pair = p
	orderBook.ExchangeName = b.Name
	orderBook.AssetType = assetType
	var orderbookNew OrderBook
	switch assetType {
	case asset.Spot, asset.Margin:

		orderbookNew, err = b.GetOrderBook(OrderBookDataRequestParams{
			Symbol: fpair.String(),
			Limit:  1000})

	case asset.USDTMarginedFutures:

		orderbookNew, err = b.UFuturesOrderbook(fpair.String(), 1000)

	case asset.CoinMarginedFutures:

		orderbookNew, err = b.GetFuturesOrderbook(fpair.String(), 1000)

	}

	if err != nil {
		return nil, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Amount: orderbookNew.Bids[x].Quantity,
				Price:  orderbookNew.Bids[x].Price,
			})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Amount: orderbookNew.Asks[x].Quantity,
				Price:  orderbookNew.Asks[x].Price,
			})
	}

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Binance exchange
func (b *Binance) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = b.Name
	assetTypes := b.GetAssetTypes()
	for x := range assetTypes {
		switch assetTypes[x] {
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

			acc.AssetType = asset.Spot
			acc.Currencies = currencyBalance
			info.Accounts = append(info.Accounts, acc)

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

			acc.AssetType = asset.CoinMarginedFutures
			acc.Currencies = currencyDetails
			info.Accounts = append(info.Accounts, acc)

		case asset.USDTMarginedFutures:

			fmt.Printf("HI\n\n\n")

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

			acc.AssetType = asset.USDTMarginedFutures
			acc.Currencies = currencyDetails
			info.Accounts = append(info.Accounts, acc)

		default:
			continue
		}
	}
	err := account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Binance) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name)
	if err != nil {
		return b.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Binance) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data within the timeframe provided.
func (b *Binance) GetExchangeHistory(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *Binance) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse

	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	switch s.AssetType {

	case asset.Spot:

		var sideType string
		if s.Side == order.Buy {
			sideType = order.Buy.String()
		} else {
			sideType = order.Sell.String()
		}

		var requestParamsOrderType RequestParamsOrderType
		switch s.Type {
		case order.Market:
			requestParamsOrderType = BinanceRequestParamsOrderMarket
		case order.Limit:
			requestParamsOrderType = BinanceRequestParamsOrderLimit
		default:
			submitOrderResponse.IsOrderPlaced = false
			return submitOrderResponse, errors.New("unsupported order type")
		}

		var orderRequest = NewOrderRequest{
			Symbol:      s.Pair.Base.String() + s.Pair.Quote.String(),
			Side:        sideType,
			Price:       s.Price,
			Quantity:    s.Amount,
			TradeType:   requestParamsOrderType,
			TimeInForce: BinanceRequestParamsTimeGTC,
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

	case asset.CoinMarginedFutures:

		fPair, err := b.FormatExchangeCurrency(s.Pair, asset.CoinMarginedFutures)
		if err != nil {
			return submitOrderResponse, err
		}

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
		}

		order, err := b.FuturesNewOrder(fPair.String(), reqSide,
			"", oType, "GTC", "",
			s.ClientOrderID, "", "", "",
			s.Amount, s.Price, 0, 0, 0)
		if err != nil {
			return submitOrderResponse, err
		}

		submitOrderResponse.OrderID = strconv.FormatInt(order.OrderID, 10)
		submitOrderResponse.IsOrderPlaced = true

	case asset.USDTMarginedFutures:

		fPair, err := b.FormatExchangeCurrency(s.Pair, asset.USDTMarginedFutures)
		if err != nil {
			return submitOrderResponse, err
		}

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
		}

		order, err := b.UFuturesNewOrder(fPair.String(), reqSide,
			"", oType, "GTC", "",
			s.ClientOrderID, "", "", "",
			s.Amount, s.Price, 0, 0, 0)
		if err != nil {
			return submitOrderResponse, err
		}

		submitOrderResponse.OrderID = strconv.FormatInt(order.OrderID, 10)
		submitOrderResponse.IsOrderPlaced = true

	}

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Binance) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Binance) CancelOrder(order *order.Cancel) error {

	fpair, err := b.FormatExchangeCurrency(order.Pair, order.AssetType)
	if err != nil {
		return err
	}

	switch order.AssetType {
	case asset.Spot:

		orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
		if err != nil {
			return err
		}

		_, err = b.CancelExistingOrder(fpair.String(),
			orderIDInt,
			order.AccountID)
		if err != nil {
			return err
		}

	case asset.CoinMarginedFutures:

		_, err := b.FuturesCancelOrder(fpair.String(), order.ID, "")
		if err != nil {
			return err
		}

	case asset.USDTMarginedFutures:

		_, err := b.UCancelOrder(fpair.String(), order.ID, "")
		if err != nil {
			return err
		}

	}
	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Binance) CancelAllOrders(req *order.Cancel) (order.CancelAllResponse, error) {

	var cancelAllOrdersResponse order.CancelAllResponse

	switch req.AssetType {

	case asset.Spot:
		openOrders, err := b.OpenOrders("")
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		for i := range openOrders {
			_, err = b.CancelExistingOrder(openOrders[i].Symbol,
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

				fPair, err := b.FormatExchangeCurrency(enabledPairs[i], asset.CoinMarginedFutures)
				if err != nil {
					return cancelAllOrdersResponse, err
				}

				_, err = b.CancelAllOpenOrders(fPair.String())
				if err != nil {
					return cancelAllOrdersResponse, err
				}

			}

		} else {

			fPair, err := b.FormatExchangeCurrency(req.Pair, asset.CoinMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			_, err = b.CancelAllOpenOrders(fPair.String())
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

				fPair, err := b.FormatExchangeCurrency(enabledPairs[i], asset.CoinMarginedFutures)
				if err != nil {
					return cancelAllOrdersResponse, err
				}

				_, err = b.UCancelAllOpenOrders(fPair.String())
				if err != nil {
					return cancelAllOrdersResponse, err
				}

			}

		} else {

			fPair, err := b.FormatExchangeCurrency(req.Pair, asset.USDTMarginedFutures)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			_, err = b.UCancelAllOpenOrders(fPair.String())
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		}

	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (b *Binance) GetOrderInfo(orderID string, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail

	orderIDInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return resp, err
	}

	switch assetType {

	case asset.Spot:

	case asset.CoinMarginedFutures:

		orderData, err := b.GetAllFuturesOrders("", "", time.Time{}, time.Time{}, orderIDInt, 0)
		if err != nil {
			return resp, err
		}
		fmt.Println(orderData)

		if len(orderData) != 1 {
			return resp, fmt.Errorf("invalid data recieved")
		}

		p, err := currency.NewPairFromString(orderData[0].Pair)
		if err != nil {
			return resp, err
		}

		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData[0].ExecutedQty
		feeBuilder.PurchasePrice = orderData[0].AvgPrice
		feeBuilder.Pair = p

		fee, err := b.GetFee(&feeBuilder)
		if err != nil {
			return resp, err
		}

		orderVars, err := compatibleOrderVars(orderData[0].Side, orderData[0].Status, orderData[0].OrderType)
		if err != nil {
			return resp, err
		}

		resp.Amount = orderData[0].OrigQty
		resp.AssetType = assetType
		resp.ClientOrderID = orderData[0].ClientOrderID
		resp.Exchange = b.Name
		resp.ExecutedAmount = orderData[0].ExecutedQty
		resp.Fee = fee
		resp.ID = orderID
		resp.Pair = p
		resp.Price = orderData[0].Price
		resp.RemainingAmount = orderData[0].OrigQty - orderData[0].ExecutedQty
		resp.Side = orderVars.Side
		resp.Status = orderVars.Status
		resp.Type = orderVars.OrderType

	case asset.USDTMarginedFutures:

		orderData, err := b.UAllAccountOrders("", 0, 0, time.Time{}, time.Time{})
		if err != nil {
			return resp, err
		}
		fmt.Println(orderData)

		if len(orderData) != 1 {
			return resp, fmt.Errorf("invalid data recieved")
		}

		p, err := currency.NewPairFromString(orderData[0].Symbol)
		if err != nil {
			return resp, err
		}

		var feeBuilder exchange.FeeBuilder
		feeBuilder.Amount = orderData[0].ExecutedQty
		feeBuilder.PurchasePrice = orderData[0].AvgPrice
		feeBuilder.Pair = p

		fee, err := b.GetFee(&feeBuilder)
		if err != nil {
			return resp, err
		}

		orderVars, err := compatibleOrderVars(orderData[0].Side, orderData[0].Status, orderData[0].OrderType)
		if err != nil {
			return resp, err
		}

		resp.Amount = orderData[0].OrigQty
		resp.AssetType = assetType
		resp.ClientOrderID = orderData[0].ClientOrderID
		resp.Exchange = b.Name
		resp.ExecutedAmount = orderData[0].ExecutedQty
		resp.Fee = fee
		resp.ID = orderID
		resp.Pair = p
		resp.Price = orderData[0].Price
		resp.RemainingAmount = orderData[0].OrigQty - orderData[0].ExecutedQty
		resp.Side = orderVars.Side
		resp.Status = orderVars.Status
		resp.Type = orderVars.OrderType

	}

	return resp, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Binance) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	return b.GetDepositAddressForCurrency(cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Binance) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
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
func (b *Binance) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Binance) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
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
	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	var orders []order.Detail

	switch req.AssetType {

	case asset.Spot:

		for x := range req.Pairs {
			fpair, err := b.FormatExchangeCurrency(req.Pairs[x],
				asset.Spot)

			if err != nil {
				return nil, err
			}

			resp, err := b.OpenOrders(fpair.String())
			if err != nil {
				return nil, err
			}

			for i := range resp {
				orderSide := order.Side(strings.ToUpper(resp[i].Side))
				orderType := order.Type(strings.ToUpper(resp[i].Type))
				orderDate := time.Unix(0, int64(resp[i].Time)*int64(time.Millisecond))

				pair, err := currency.NewPairFromString(resp[i].Symbol)
				if err != nil {
					return nil, err
				}

				orders = append(orders, order.Detail{
					Amount:   resp[i].OrigQty,
					Date:     orderDate,
					Exchange: b.Name,
					ID:       strconv.FormatInt(resp[i].OrderID, 10),
					Side:     orderSide,
					Type:     orderType,
					Price:    resp[i].Price,
					Status:   order.Status(resp[i].Status),
					Pair:     pair,
				})
			}
		}

	case asset.CoinMarginedFutures:

		var openOrders []FuturesOrderData
		var err error

		var pairs currency.Pairs

		if len(req.Pairs) > 0 {
			pairs = req.Pairs
		} else {
			pairs, err = b.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return orders, err
			}
		}

		for i := range pairs {

			fPair, err := b.FormatExchangeCurrency(pairs[i], req.AssetType)
			if err != nil {
				return orders, err
			}

			openOrders, err = b.GetFuturesAllOpenOrders(fPair.String(), "")
			if err != nil {
				return orders, err
			}

			for y := range openOrders {

				var feeBuilder exchange.FeeBuilder
				feeBuilder.Amount = openOrders[y].ExecutedQty
				feeBuilder.PurchasePrice = openOrders[y].AvgPrice
				feeBuilder.Pair = fPair

				fee, err := b.GetFee(&feeBuilder)
				if err != nil {
					return orders, err
				}

				var orderVars OrderVars
				orderVars, err = compatibleOrderVars(openOrders[y].Side, openOrders[y].Status, openOrders[y].OrderType)
				if err != nil {
					return orders, err
				}

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
					Pair:            fPair,
					AssetType:       asset.CoinMarginedFutures,
				})
			}
		}

	case asset.USDTMarginedFutures:

		var openOrders []UOrderData
		var err error
		var pairs currency.Pairs

		if len(req.Pairs) > 0 {
			pairs = req.Pairs
		} else {
			pairs, err = b.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return orders, err
			}
		}

		for i := range pairs {

			fPair, err := b.FormatExchangeCurrency(pairs[i], req.AssetType)
			if err != nil {
				return orders, err
			}

			var complete bool

			for !complete {
				openOrders, err = b.UAllAccountOpenOrders(fPair.String())
				if err != nil {
					return orders, err
				}

				for y := range openOrders {

					var feeBuilder exchange.FeeBuilder
					feeBuilder.Amount = openOrders[y].ExecutedQty
					feeBuilder.PurchasePrice = openOrders[y].AvgPrice
					feeBuilder.Pair = fPair

					fee, err := b.GetFee(&feeBuilder)
					if err != nil {
						return orders, err
					}

					var orderVars OrderVars
					orderVars, err = compatibleOrderVars(openOrders[y].Side, openOrders[y].Status, openOrders[y].OrderType)
					if err != nil {
						return orders, err
					}

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
						Pair:            fPair,
						AssetType:       asset.CoinMarginedFutures,
					})
				}
				if len(openOrders) < 100 {
					complete = true
				}
			}
		}
	default:
		return orders, fmt.Errorf("assetType not supported")
	}
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Binance) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	var orders []order.Detail

	switch req.AssetType {

	case asset.Spot:

		for x := range req.Pairs {
			fpair, err := b.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
			if err != nil {
				return nil, err
			}
			resp, err := b.AllOrders(fpair.String(),
				"",
				"1000")
			if err != nil {
				return nil, err
			}

			for i := range resp {
				orderSide := order.Side(strings.ToUpper(resp[i].Side))
				orderType := order.Type(strings.ToUpper(resp[i].Type))
				orderDate := time.Unix(0, int64(resp[i].Time)*int64(time.Millisecond))
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
					Date:     orderDate,
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

		var orderHistory []FuturesOrderData
		var err error

		var pairs currency.Pairs

		if len(req.Pairs) > 0 {
			pairs = req.Pairs
		} else {
			pairs, err = b.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return orders, err
			}
		}

		for i := range pairs {

			fPair, err := b.FormatExchangeCurrency(pairs[i], req.AssetType)
			if err != nil {
				return orders, err
			}

			var complete bool

			for !complete {
				orderHistory, err = b.GetAllFuturesOrders(fPair.String(), "", req.StartTicks, req.EndTicks, 0, 100)
				if err != nil {
					return orders, err
				}

				for y := range orderHistory {

					var feeBuilder exchange.FeeBuilder
					feeBuilder.Amount = orderHistory[y].ExecutedQty
					feeBuilder.PurchasePrice = orderHistory[y].AvgPrice
					feeBuilder.Pair = fPair

					fee, err := b.GetFee(&feeBuilder)
					if err != nil {
						return orders, err
					}

					var orderVars OrderVars
					orderVars, err = compatibleOrderVars(orderHistory[y].Side, orderHistory[y].Status, orderHistory[y].OrderType)
					if err != nil {
						return orders, err
					}

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
						Pair:            fPair,
						AssetType:       asset.CoinMarginedFutures,
					})
				}
				if len(orderHistory) < 100 {
					complete = true
				}
			}
		}

	case asset.USDTMarginedFutures:

		var orderHistory []UFuturesOrderData
		var err error

		var pairs currency.Pairs

		if len(req.Pairs) > 0 {
			pairs = req.Pairs
		} else {
			pairs, err = b.GetEnabledPairs(asset.CoinMarginedFutures)
			if err != nil {
				return orders, err
			}
		}

		for i := range pairs {

			fPair, err := b.FormatExchangeCurrency(pairs[i], req.AssetType)
			if err != nil {
				return orders, err
			}

			var complete bool

			for !complete {
				orderHistory, err = b.UAllAccountOrders(fPair.String(), 0, 100, req.StartTicks, req.EndTicks)
				if err != nil {
					return orders, err
				}

				for y := range openOrders {

					var feeBuilder exchange.FeeBuilder
					feeBuilder.Amount = orderHistory[y].ExecutedQty
					feeBuilder.PurchasePrice = orderHistory[y].AvgPrice
					feeBuilder.Pair = fPair

					fee, err := b.GetFee(&feeBuilder)
					if err != nil {
						return orders, err
					}

					var orderVars OrderVars
					orderVars, err = compatibleOrderVars(orderHistory[y].Side, orderHistory[y].Status, orderHistory[y].OrderType)
					if err != nil {
						return orders, err
					}

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
						Pair:            fPair,
						AssetType:       asset.CoinMarginedFutures,
					})
				}
				if len(orderHistory) < 100 {
					complete = true
				}
			}
		}
	default:
		return orders, fmt.Errorf("assetType not supported")
	}
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Binance) ValidateCredentials() error {
	_, err := b.UpdateAccountInfo()
	return b.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (b *Binance) FormatExchangeKlineInterval(in kline.Interval) string {
	if in == kline.OneDay {
		return "1d"
	}
	if in == kline.OneMonth {
		return "1M"
	}
	return in.Short()
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Binance) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if !b.KlineIntervalEnabled(interval) {
		return kline.Item{}, kline.ErrorKline{
			Interval: interval,
		}
	}

	if kline.TotalCandlesPerInterval(start, end, interval) > b.Features.Enabled.Kline.ResultLimit {
		return kline.Item{}, errors.New(kline.ErrRequestExceedsExchangeLimits)
	}

	fpair, err := b.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}
	req := KlinesRequestParams{
		Interval:  b.FormatExchangeKlineInterval(interval),
		Symbol:    fpair.String(),
		StartTime: start.Unix() * 1000,
		EndTime:   end.Unix() * 1000,
		Limit:     int(b.Features.Enabled.Kline.ResultLimit),
	}

	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	candles, err := b.GetSpotKline(req)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].Open,
			High:   candles[x].Close,
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
	if !b.KlineIntervalEnabled(interval) {
		return kline.Item{}, kline.ErrorKline{
			Interval: interval,
		}
	}

	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	formattedPair, err := b.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	dates := kline.CalcDateRanges(start, end, interval, b.Features.Enabled.Kline.ResultLimit)
	for x := range dates {
		req := KlinesRequestParams{
			Interval:  b.FormatExchangeKlineInterval(interval),
			Symbol:    formattedPair.String(),
			StartTime: dates[x].Start.UTC().Unix() * 1000,
			EndTime:   dates[x].End.UTC().Unix() * 1000,
			Limit:     int(b.Features.Enabled.Kline.ResultLimit),
		}

		candles, err := b.GetSpotKline(req)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range candles {
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   candles[i].OpenTime,
				Open:   candles[i].Open,
				High:   candles[i].Close,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

func compatibleOrderVars(side, status, orderType string) (OrderVars, error) {
	var resp OrderVars

	switch side {
	case "BUY":
		resp.Side = order.Buy
	case "SELL":
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

	return resp, nil
}
