package binance

import (
	"errors"
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

	err := b.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.Margin, fmt1)
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
	info, err := b.GetExchangeInfo()
	if err != nil {
		return nil, err
	}

	format, err := b.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}

	var pairs []string
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
	tick, err := b.GetTickers()
	if err != nil {
		return nil, err
	}

	pairs, err := b.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}

	for i := range pairs {
		for y := range tick {
			pairFmt, err := b.FormatExchangeCurrency(pairs[i], assetType)
			if err != nil {
				return nil, err
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
				Pair:         pairs[i],
				ExchangeName: b.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return nil, err
			}
		}
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
	fpair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	orderbookNew, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: fpair.String(),
		Limit:  1000})
	if err != nil {
		return nil, err
	}

	orderBook := new(orderbook.Base)
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

	orderBook.Pair = p
	orderBook.ExchangeName = b.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Bithumb exchange
func (b *Binance) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
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

	info.Exchange = b.Name
	info.Accounts = append(info.Accounts, account.SubAccount{
		Currencies: currencyBalance,
	})

	err = account.Process(&info)
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

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Binance) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	limit := 1000
	tradeData, err := b.GetMostRecentTrades(RecentTradeRequestParams{p.String(), limit})
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
			Timestamp:    time.Unix(0, tradeData[i].Time*int64(time.Millisecond)),
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
func (b *Binance) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *Binance) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

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

	fPair, err := b.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	var orderRequest = NewOrderRequest{
		Symbol:      fPair.String(),
		Side:        sideType,
		Price:       s.Price,
		Quantity:    s.Amount,
		TradeType:   requestParamsOrderType,
		TimeInForce: timeInForce,
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

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	fpair, err := b.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder(fpair.String(),
		orderIDInt,
		o.AccountID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Binance) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
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

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (b *Binance) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (o order.Detail, err error) {
	if assetType == "" {
		assetType = asset.Spot
	}

	formattedPair, err := b.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return
	}

	orderIDInt64, err := convert.Int64FromString(orderID)
	if err != nil {
		return
	}

	resp, err := b.QueryOrder(formattedPair.String(), "", orderIDInt64)
	if err != nil {
		return
	}

	orderSide := order.Side(resp.Side)
	orderDate, err := convert.TimeFromUnixTimestampFloat(resp.Time)
	if err != nil {
		return
	}

	orderCloseDate, err := convert.TimeFromUnixTimestampFloat(float64(resp.UpdateTime))
	if err != nil {
		return
	}

	status, err := order.StringToOrderStatus(resp.Status)
	if err != nil {
		return
	}

	orderType := order.Limit
	if resp.Type == "MARKET" {
		orderType = order.Market
	}

	return order.Detail{
		Amount:         resp.OrigQty,
		Date:           orderDate,
		Exchange:       b.Name,
		ID:             strconv.FormatInt(resp.OrderID, 10),
		Side:           orderSide,
		Type:           orderType,
		Pair:           formattedPair,
		Cost:           resp.CummulativeQuoteQty,
		AssetType:      assetType,
		CloseTime:      orderCloseDate,
		Status:         status,
		Price:          resp.Price,
		ExecutedAmount: resp.ExecutedQty,
	}, nil
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
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	var orders []order.Detail
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
			orderDate, err := convert.TimeFromUnixTimestampFloat(resp[i].Time)
			if err != nil {
				return nil, err
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
				Status:   order.Status(resp[i].Status),
				Pair:     pair,
			})
		}
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
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
			orderDate, err := convert.TimeFromUnixTimestampFloat(resp[i].Time)
			if err != nil {
				return nil, err
			}
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
	if err := b.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
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
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
