package lakebtc

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
func (l *LakeBTC) GetDefaultConfig() (*config.ExchangeConfig, error) {
	l.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = l.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = l.BaseCurrencies

	err := l.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if l.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = l.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets LakeBTC defaults
func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = true
	l.Verbose = true
	l.API.CredentialsValidator.RequiresKey = true
	l.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Uppercase: true}
	err := l.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	l.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:    true,
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				UserTradeHistory:  true,
				CryptoWithdrawal:  true,
				TradeFee:          true,
				CryptoDepositFee:  true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:     true,
				OrderbookFetching: true,
				Subscribe:         true,
				Unsubscribe:       true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	l.Requester = request.New(l.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	l.API.Endpoints.URLDefault = lakeBTCAPIURL
	l.API.Endpoints.URL = l.API.Endpoints.URLDefault
	l.Websocket = stream.New()
	l.API.Endpoints.WebsocketURL = lakeBTCWSURL
	l.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	l.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	l.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets exchange configuration profile
func (l *LakeBTC) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		l.SetEnabled(false)
		return nil
	}

	err := l.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return l.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       lakeBTCWSURL,
		ExchangeName:                     exch.Name,
		RunningURL:                       exch.API.Endpoints.WebsocketURL,
		Connector:                        l.WsConnect,
		Subscriber:                       l.Subscribe,
		UnSubscriber:                     l.Unsubscribe,
		GenerateSubscriptions:            l.GenerateDefaultSubscriptions,
		Features:                         &l.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.WebsocketOrderbookBufferLimit,
	})
}

// Start starts the LakeBTC go routine
func (l *LakeBTC) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the LakeBTC wrapper
func (l *LakeBTC) Run() {
	if l.Verbose {
		l.PrintEnabledPairs()
	}

	if !l.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := l.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", l.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (l *LakeBTC) FetchTradablePairs(asset asset.Item) ([]string, error) {
	result, err := l.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result {
		currencies = append(currencies, strings.ToUpper(x))
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (l *LakeBTC) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := l.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return l.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LakeBTC) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	ticks, err := l.GetTicker()
	if err != nil {
		return nil, err
	}

	pairs, err := l.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}

	for i := range pairs {
		fpair, err := l.FormatExchangeCurrency(pairs[i], assetType)
		if err != nil {
			return nil, err
		}

		c, ok := ticks[fpair.String()]
		if !ok {
			continue
		}

		tickerPrice := new(ticker.Price)
		tickerPrice.Pair = pairs[i]
		tickerPrice.Ask = c.Ask
		tickerPrice.Bid = c.Bid
		tickerPrice.Volume = c.Volume
		tickerPrice.High = c.High
		tickerPrice.Low = c.Low
		tickerPrice.Last = c.Last
		tickerPrice.ExchangeName = l.Name
		tickerPrice.AssetType = assetType

		err = ticker.ProcessTicker(tickerPrice)
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(l.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (l *LakeBTC) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(l.Name, fPair, assetType)
	if err != nil {
		return l.UpdateTicker(fPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (l *LakeBTC) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	ob, err := orderbook.Get(l.Name, fPair, assetType)
	if err != nil {
		return l.UpdateOrderbook(fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LakeBTC) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	orderBook := new(orderbook.Base)
	orderbookNew, err := l.GetOrderBook(fPair.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: orderbookNew.Bids[x].Amount, Price: orderbookNew.Bids[x].Price})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: orderbookNew.Asks[x].Amount, Price: orderbookNew.Asks[x].Price})
	}

	orderBook.Pair = fPair
	orderBook.ExchangeName = l.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(l.Name, fPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// LakeBTC exchange
func (l *LakeBTC) UpdateAccountInfo() (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = l.Name
	accountInfo, err := l.GetAccountInformation()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for x, y := range accountInfo.Balance {
		for z, w := range accountInfo.Locked {
			if z != x {
				continue
			}
			var exchangeCurrency account.Balance
			exchangeCurrency.CurrencyName = currency.NewCode(x)
			exchangeCurrency.TotalValue, _ = strconv.ParseFloat(y, 64)
			exchangeCurrency.Hold, _ = strconv.ParseFloat(w, 64)
			currencies = append(currencies, exchangeCurrency)
		}
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
func (l *LakeBTC) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(l.Name)
	if err != nil {
		return l.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LakeBTC) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (l *LakeBTC) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	var tradeData []TradeHistory
	tradeData, err = l.GetTradeHistory(p.String())
	if err != nil {
		return nil, err
	}
	for i := range tradeData {
		tradeTS := time.Unix(tradeData[i].Date, 0)
		resp = append(resp, trade.Data{
			TID:          strconv.FormatInt(tradeData[i].TID, 10),
			Exchange:     l.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeTS,
		})
	}

	err = l.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (l *LakeBTC) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (l *LakeBTC) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	fPair, err := l.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	isBuyOrder := s.Side == order.Buy
	response, err := l.Trade(isBuyOrder,
		s.Amount,
		s.Price,
		fPair.Lower().String())
	if err != nil {
		return submitOrderResponse, err
	}
	if response.ID > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.ID, 10)
	}

	submitOrderResponse.IsOrderPlaced = true
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LakeBTC) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LakeBTC) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	return l.CancelExistingOrder(orderIDInt)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LakeBTC) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	var cancelAllOrdersResponse order.CancelAllResponse
	openOrders, err := l.GetOpenOrders()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	var ordersToCancel []string
	for i := range openOrders {
		ordersToCancel = append(ordersToCancel, strconv.FormatInt(openOrders[i].ID, 10))
	}

	return cancelAllOrdersResponse, l.CancelExistingOrders(ordersToCancel)
}

// GetOrderInfo returns order information based on order ID
func (l *LakeBTC) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LakeBTC) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	if !strings.EqualFold(cryptocurrency.String(), currency.BTC.String()) {
		return "", fmt.Errorf("unsupported currency %s deposit address can only be BTC, manual deposit is required for other currencies",
			cryptocurrency.String())
	}

	info, err := l.GetAccountInformation()
	if err != nil {
		return "", err
	}

	return info.Profile.BTCDepositAddress, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LakeBTC) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}

	if withdrawRequest.Currency != currency.BTC {
		return nil, errors.New("only BTC supported for withdrawals")
	}

	resp, err := l.CreateWithdraw(withdrawRequest.Amount, withdrawRequest.Description)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp.ID, 10),
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LakeBTC) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LakeBTC) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !l.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return l.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (l *LakeBTC) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := l.GetOpenOrders()
	if err != nil {
		return nil, err
	}

	format, err := l.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(resp[i].Symbol,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(resp[i].At, 0)
		side := order.Side(strings.ToUpper(resp[i].Type))

		orders = append(orders, order.Detail{
			Amount:   resp[i].Amount,
			ID:       strconv.FormatInt(resp[i].ID, 10),
			Price:    resp[i].Price,
			Side:     side,
			Date:     orderDate,
			Pair:     symbol,
			Exchange: l.Name,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *LakeBTC) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := l.GetOrders([]int64{})
	if err != nil {
		return nil, err
	}

	format, err := l.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		if resp[i].State == "active" {
			continue
		}
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(resp[i].Symbol, format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(resp[i].At, 0)
		side := order.Side(strings.ToUpper(resp[i].Type))

		orders = append(orders, order.Detail{
			Amount:   resp[i].Amount,
			ID:       strconv.FormatInt(resp[i].ID, 10),
			Price:    resp[i].Price,
			Side:     side,
			Date:     orderDate,
			Pair:     symbol,
			Exchange: l.Name,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)

	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (l *LakeBTC) ValidateCredentials() error {
	_, err := l.UpdateAccountInfo()
	return l.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (l *LakeBTC) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (l *LakeBTC) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
