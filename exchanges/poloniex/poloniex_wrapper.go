package poloniex

import (
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
func (p *Poloniex) GetDefaultConfig() (*config.ExchangeConfig, error) {
	p.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = p.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = p.BaseCurrencies

	err := p.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if p.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = p.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default settings for poloniex
func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = true
	p.Verbose = true
	p.API.CredentialsValidator.RequiresKey = true
	p.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	configFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	err := p.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	p.Features = exchange.Features{
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
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.OneDay.Word():     true,
				},
			},
		},
	}

	p.Requester = request.New(p.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))

	p.API.Endpoints.URLDefault = poloniexAPIURL
	p.API.Endpoints.URL = p.API.Endpoints.URLDefault
	p.API.Endpoints.WebsocketURL = poloniexWebsocketAddress
	p.Websocket = stream.New()
	p.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	p.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	p.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (p *Poloniex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		p.SetEnabled(false)
		return nil
	}

	err := p.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = p.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       poloniexWebsocketAddress,
		ExchangeName:                     exch.Name,
		RunningURL:                       exch.API.Endpoints.WebsocketURL,
		Connector:                        p.WsConnect,
		Subscriber:                       p.Subscribe,
		UnSubscriber:                     p.Unsubscribe,
		GenerateSubscriptions:            p.GenerateDefaultSubscriptions,
		Features:                         &p.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.WebsocketOrderbookBufferLimit,
		SortBuffer:                       true,
		SortBufferByUpdateIDs:            true,
	})
	if err != nil {
		return err
	}

	return p.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Poloniex go routine
func (p *Poloniex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		p.Run()
		wg.Done()
	}()
}

// Run implements the Poloniex wrapper
func (p *Poloniex) Run() {
	if p.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s (url: %s).\n",
			p.Name,
			common.IsEnabled(p.Websocket.IsEnabled()),
			poloniexWebsocketAddress)
		p.PrintEnabledPairs()
	}

	forceUpdate := false

	avail, err := p.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			p.Name,
			err)
		return
	}

	if common.StringDataCompare(avail.Strings(), "BTC_USDT") {
		log.Warnf(log.ExchangeSys,
			"%s contains invalid pair, forcing upgrade of available currencies.\n",
			p.Name)
		forceUpdate = true
	}

	if !p.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = p.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			p.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (p *Poloniex) FetchTradablePairs(asset asset.Item) ([]string, error) {
	resp, err := p.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range resp {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (p *Poloniex) UpdateTradablePairs(forceUpgrade bool) error {
	pairs, err := p.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}
	ps, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return p.UpdatePairs(ps, asset.Spot, false, forceUpgrade)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *Poloniex) UpdateTicker(currencyPair currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := p.GetTicker()
	if err != nil {
		return nil, err
	}

	enabledPairs, err := p.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}
	for i := range enabledPairs {
		fpair, err := p.FormatExchangeCurrency(enabledPairs[i], assetType)
		if err != nil {
			return nil, err
		}
		curr := fpair.String()
		if _, ok := tick[curr]; !ok {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         enabledPairs[i],
			Ask:          tick[curr].LowestAsk,
			Bid:          tick[curr].HighestBid,
			High:         tick[curr].High24Hr,
			Last:         tick[curr].Last,
			Low:          tick[curr].Low24Hr,
			Volume:       tick[curr].BaseVolume,
			QuoteVolume:  tick[curr].QuoteVolume,
			ExchangeName: p.Name,
			AssetType:    assetType})
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(p.Name, currencyPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (p *Poloniex) FetchTicker(currencyPair currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateTicker(currencyPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (p *Poloniex) FetchOrderbook(currencyPair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateOrderbook(currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *Poloniex) UpdateOrderbook(currencyPair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderbookNew, err := p.GetOrderbook("", poloniexMaxOrderbookDepth)
	if err != nil {
		return nil, err
	}

	enabledPairs, err := p.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}
	for i := range enabledPairs {
		fpair, err := p.FormatExchangeCurrency(enabledPairs[i], assetType)
		if err != nil {
			return nil, err
		}
		data, ok := orderbookNew.Data[fpair.String()]
		if !ok {
			continue
		}

		orderBook := new(orderbook.Base)
		for y := range data.Bids {
			orderBook.Bids = append(orderBook.Bids, orderbook.Item{
				Amount: data.Bids[y].Amount,
				Price:  data.Bids[y].Price,
			})
		}

		for y := range data.Asks {
			orderBook.Asks = append(orderBook.Asks, orderbook.Item{
				Amount: data.Asks[y].Amount,
				Price:  data.Asks[y].Price,
			})
		}
		orderBook.Pair = enabledPairs[i]
		orderBook.ExchangeName = p.Name
		orderBook.AssetType = assetType

		err = orderBook.Process()
		if err != nil {
			return orderBook, err
		}
	}
	return orderbook.Get(p.Name, currencyPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (p *Poloniex) UpdateAccountInfo() (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = p.Name
	accountBalance, err := p.GetBalances()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for x, y := range accountBalance.Currency {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(x)
		exchangeCurrency.TotalValue = y
		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		Currencies: currencies,
	})

	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (p *Poloniex) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(p.Name)
	if err != nil {
		return p.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (p *Poloniex) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data within the timeframe provided.
func (p *Poloniex) GetExchangeHistory(currencyPair currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (p *Poloniex) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	fillOrKill := s.Type == order.Market
	isBuyOrder := s.Side == order.Buy
	response, err := p.PlaceOrder(s.Pair.String(),
		s.Price,
		s.Amount,
		false,
		fillOrKill,
		isBuyOrder)
	if err != nil {
		return submitOrderResponse, err
	}
	if response.OrderNumber > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.OrderNumber, 10)
	}

	submitOrderResponse.IsOrderPlaced = true
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (p *Poloniex) ModifyOrder(action *order.Modify) (string, error) {
	oID, err := strconv.ParseInt(action.ID, 10, 64)
	if err != nil {
		return "", err
	}

	resp, err := p.MoveOrder(oID,
		action.Price,
		action.Amount,
		action.PostOnly,
		action.ImmediateOrCancel)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(resp.OrderNumber, 10), nil
}

// CancelOrder cancels an order by its corresponding ID number
func (p *Poloniex) CancelOrder(order *order.Cancel) error {
	orderIDInt, err := strconv.ParseInt(order.ID, 10, 64)
	if err != nil {
		return err
	}

	return p.CancelExistingOrder(orderIDInt)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (p *Poloniex) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := p.GetOpenOrdersForAllCurrencies()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for key := range openOrders.Data {
		for i := range openOrders.Data[key] {
			err = p.CancelExistingOrder(openOrders.Data[key][i].OrderNumber)
			if err != nil {
				id := strconv.FormatInt(openOrders.Data[key][i].OrderNumber, 10)
				cancelAllOrdersResponse.Status[id] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns information on a current open order
func (p *Poloniex) GetOrderInfo(orderID string) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (p *Poloniex) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	a, err := p.GetDepositAddresses()
	if err != nil {
		return "", err
	}

	address, ok := a.Addresses[cryptocurrency.Upper().String()]
	if !ok {
		return "", fmt.Errorf("cannot find deposit address for %s",
			cryptocurrency)
	}

	return address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (p *Poloniex) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	v, err := p.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.Crypto.Address, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v.Response,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!p.AllowAuthenticatedRequest() || p.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return p.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (p *Poloniex) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	resp, err := p.GetOpenOrdersForAllCurrencies()
	if err != nil {
		return nil, err
	}

	format, err := p.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key := range resp.Data {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}
		for i := range resp.Data[key] {
			orderSide := order.Side(strings.ToUpper(resp.Data[key][i].Type))
			orderDate, err := time.Parse(common.SimpleTimeFormat, resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name,
					"GetActiveOrders",
					resp.Data[key][i].OrderNumber,
					resp.Data[key][i].Date)
			}

			orders = append(orders, order.Detail{
				ID:       strconv.FormatInt(resp.Data[key][i].OrderNumber, 10),
				Side:     orderSide,
				Amount:   resp.Data[key][i].Amount,
				Date:     orderDate,
				Price:    resp.Data[key][i].Rate,
				Pair:     symbol,
				Exchange: p.Name,
			})
		}
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersBySide(&orders, req.Side)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (p *Poloniex) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	resp, err := p.GetAuthenticatedTradeHistory(req.StartTicks.Unix(),
		req.EndTicks.Unix(),
		10000)
	if err != nil {
		return nil, err
	}

	format, err := p.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key := range resp.Data {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}

		for i := range resp.Data[key] {
			orderSide := order.Side(strings.ToUpper(resp.Data[key][i].Type))
			orderDate, err := time.Parse(common.SimpleTimeFormat,
				resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name,
					"GetActiveOrders",
					resp.Data[key][i].OrderNumber,
					resp.Data[key][i].Date)
			}

			orders = append(orders, order.Detail{
				ID:       strconv.FormatInt(resp.Data[key][i].GlobalTradeID, 10),
				Side:     orderSide,
				Amount:   resp.Data[key][i].Amount,
				Date:     orderDate,
				Price:    resp.Data[key][i].Rate,
				Pair:     symbol,
				Exchange: p.Name,
			})
		}
	}

	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersBySide(&orders, req.Side)

	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (p *Poloniex) ValidateCredentials() error {
	_, err := p.UpdateAccountInfo()
	return p.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if !p.KlineIntervalEnabled(interval) {
		return kline.Item{}, kline.ErrorKline{
			Interval: interval,
		}
	}

	fmtP, err := p.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	candles, err := p.GetChartData(fmtP.String(),
		start, end,
		p.FormatExchangeKlineInterval(interval))
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: p.Name,
		Interval: interval,
		Pair:     pair,
		Asset:    a,
	}

	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   time.Unix(candles[x].Date, 0),
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
func (p *Poloniex) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return p.GetHistoricCandles(pair, a, start, end, interval)
}
