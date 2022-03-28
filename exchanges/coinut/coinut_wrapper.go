package coinut

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (c *COINUT) GetDefaultConfig() (*config.Exchange, error) {
	c.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = c.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = c.BaseCurrencies

	err := c.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if c.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = c.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default values
func (c *COINUT) SetDefaults() {
	c.Name = "COINUT"
	c.Enabled = true
	c.Verbose = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresClientID = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := c.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				SubmitOrders:      true,
				UserTradeHistory:  true,
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				GetOrders:              true,
				CancelOrders:           true,
				CancelOrder:            true,
				SubmitOrder:            true,
				SubmitOrders:           true,
				UserTradeHistory:       true,
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				AccountInfo:            true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
			},
			WithdrawPermissions: exchange.WithdrawCryptoViaWebsiteOnly |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	c.Requester, err = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	c.API.Endpoints = c.NewEndpoints()
	err = c.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      coinutAPIURL,
		exchange.WebsocketSpot: coinutWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	c.Websocket = stream.New()
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
	rand.Seed(time.Now().UnixNano())
}

// Setup sets the current exchange configuration
func (c *COINUT) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		c.SetEnabled(false)
		return nil
	}
	err = c.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := c.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = c.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            coinutWebsocketURL,
		RunningURL:            wsRunningURL,
		Connector:             c.WsConnect,
		Subscriber:            c.Subscribe,
		Unsubscriber:          c.Unsubscribe,
		GenerateSubscriptions: c.GenerateDefaultSubscriptions,
		Features:              &c.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	return c.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            wsRateLimitInMilliseconds,
	})
}

// Start starts the COINUT go routine
func (c *COINUT) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the COINUT wrapper
func (c *COINUT) Run() {
	if c.Verbose {
		log.Debugf(log.ExchangeSys, "%s Websocket: %s. (url: %s).\n", c.Name, common.IsEnabled(c.Websocket.IsEnabled()), coinutWebsocketURL)
		c.PrintEnabledPairs()
	}

	forceUpdate := false
	if !c.BypassConfigFormatUpgrades {
		format, err := c.GetPairFormat(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				c.Name,
				err)
			return
		}

		enabled, err := c.CurrencyPairs.GetPairs(asset.Spot, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				c.Name,
				err)
			return
		}
		avail, err := c.CurrencyPairs.GetPairs(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				c.Name,
				err)
			return
		}

		if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
			!common.StringDataContains(avail.Strings(), format.Delimiter) {
			var p currency.Pairs
			p, err = currency.NewPairsFromStrings([]string{currency.LTC.String() +
				format.Delimiter +
				currency.USDT.String()})
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					c.Name,
					err)
			} else {
				log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, c.Name, asset.Spot, p)
				forceUpdate = true

				err = c.UpdatePairs(p, asset.Spot, true, true)
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%s failed to update currencies. Err: %s\n",
						c.Name,
						err)
				}
			}
		}
	}

	if !c.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := c.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *COINUT) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	var instruments map[string][]InstrumentBase
	var resp Instruments
	var err error
	if c.Websocket.IsConnected() {
		resp, err = c.WsGetInstruments()
		if err != nil {
			return nil, err
		}
	} else {
		resp, err = c.GetInstruments(ctx)
		if err != nil {
			return nil, err
		}
	}

	format, err := c.GetPairFormat(asset, false)
	if err != nil {
		return nil, err
	}

	instruments = resp.Instruments
	var pairs []string
	for i := range instruments {
		c.instrumentMap.Seed(instruments[i][0].Base+instruments[i][0].Quote, instruments[i][0].InstrumentID)
		p := instruments[i][0].Base + format.Delimiter + instruments[i][0].Quote
		pairs = append(pairs, p)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *COINUT) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return c.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// COINUT exchange
func (c *COINUT) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var bal *UserBalance
	var err error
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var resp *UserBalance
		resp, err = c.wsGetAccountBalance()
		if err != nil {
			return info, err
		}
		bal = resp
	} else {
		bal, err = c.GetUserBalance(ctx)
		if err != nil {
			return info, err
		}
	}

	var balances = []account.Balance{
		{
			CurrencyName: currency.BCH,
			Total:        bal.BCH,
		},
		{
			CurrencyName: currency.BTC,
			Total:        bal.BTC,
		},
		{
			CurrencyName: currency.BTG,
			Total:        bal.BTG,
		},
		{
			CurrencyName: currency.CAD,
			Total:        bal.CAD,
		},
		{
			CurrencyName: currency.ETC,
			Total:        bal.ETC,
		},
		{
			CurrencyName: currency.ETH,
			Total:        bal.ETH,
		},
		{
			CurrencyName: currency.LCH,
			Total:        bal.LCH,
		},
		{
			CurrencyName: currency.LTC,
			Total:        bal.LTC,
		},
		{
			CurrencyName: currency.MYR,
			Total:        bal.MYR,
		},
		{
			CurrencyName: currency.SGD,
			Total:        bal.SGD,
		},
		{
			CurrencyName: currency.USD,
			Total:        bal.USD,
		},
		{
			CurrencyName: currency.USDT,
			Total:        bal.USDT,
		},
		{
			CurrencyName: currency.XMR,
			Total:        bal.XMR,
		},
		{
			CurrencyName: currency.ZEC,
			Total:        bal.ZEC,
		},
	}
	info.Exchange = c.Name
	info.Accounts = append(info.Accounts, account.SubAccount{
		Currencies: balances,
	})

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (c *COINUT) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(c.Name, assetType)
	if err != nil {
		return c.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (c *COINUT) UpdateTickers(ctx context.Context, a asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *COINUT) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return nil, err
	}

	fpair, err := c.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	instID := c.instrumentMap.LookupID(fpair.String())
	if instID == 0 {
		return nil, errors.New("unable to lookup instrument ID")
	}
	var tick Ticker
	tick, err = c.GetInstrumentTicker(ctx, instID)
	if err != nil {
		return nil, err
	}

	err = ticker.ProcessTicker(&ticker.Price{
		Last:         tick.Last,
		High:         tick.High24,
		Low:          tick.Low24,
		Bid:          tick.HighestBuy,
		Ask:          tick.LowestSell,
		Volume:       tick.Volume24,
		Pair:         p,
		LastUpdated:  time.Unix(0, tick.Timestamp),
		ExchangeName: c.Name,
		AssetType:    a})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(c.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (c *COINUT) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *COINUT) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(c.Name, p, assetType)
	if err != nil {
		return c.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *COINUT) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        c.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: c.CanVerifyOrderbook,
	}
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return book, err
	}

	fpair, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	instID := c.instrumentMap.LookupID(fpair.String())
	if instID == 0 {
		return book, errLookupInstrumentID
	}

	orderbookNew, err := c.GetInstrumentOrderbook(ctx, instID, 200)
	if err != nil {
		return book, err
	}

	for x := range orderbookNew.Buy {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: orderbookNew.Buy[x].Quantity,
			Price:  orderbookNew.Buy[x].Price})
	}

	for x := range orderbookNew.Sell {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: orderbookNew.Sell[x].Quantity,
			Price:  orderbookNew.Sell[x].Price})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(c.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *COINUT) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (c *COINUT) GetWithdrawalsHistory(_ context.Context, _ currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (c *COINUT) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	currencyID := c.instrumentMap.LookupID(p.String())
	if currencyID == 0 {
		return nil, errLookupInstrumentID
	}
	var tradeData Trades
	tradeData, err = c.GetTrades(ctx, currencyID)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData.Trades {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData.Trades[i].Side)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			Exchange:     c.Name,
			TID:          strconv.FormatInt(tradeData.Trades[i].TransactionID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData.Trades[i].Price,
			Amount:       tradeData.Trades[i].Quantity,
			Timestamp:    time.Unix(0, tradeData.Trades[i].Timestamp*int64(time.Microsecond)),
		})
	}

	err = c.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (c *COINUT) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (c *COINUT) SubmitOrder(ctx context.Context, o *order.Submit) (order.SubmitResponse, error) {
	if err := o.Validate(); err != nil {
		return order.SubmitResponse{}, err
	}

	var submitOrderResponse order.SubmitResponse
	var err error
	if _, err = strconv.Atoi(o.ClientID); err != nil {
		return submitOrderResponse, fmt.Errorf("%s - ClientID must be a number, received: %s", c.Name, o.ClientID)
	}

	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var response *order.Detail
		response, err = c.wsSubmitOrder(&WsSubmitOrderParameters{
			Currency: o.Pair,
			Side:     o.Side,
			Amount:   o.Amount,
			Price:    o.Price,
		})
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = response.ID
		submitOrderResponse.IsOrderPlaced = true
	} else {
		err = c.loadInstrumentsIfNotLoaded()
		if err != nil {
			return submitOrderResponse, err
		}

		fpair, err := c.FormatExchangeCurrency(o.Pair, asset.Spot)
		if err != nil {
			return submitOrderResponse, err
		}

		currencyID := c.instrumentMap.LookupID(fpair.String())
		if currencyID == 0 {
			return submitOrderResponse, errLookupInstrumentID
		}

		var APIResponse interface{}
		var clientIDInt uint64
		isBuyOrder := o.Side == order.Buy
		clientIDInt, err = strconv.ParseUint(o.ClientID, 0, 32)
		if err != nil {
			return submitOrderResponse, err
		}
		clientIDUint := uint32(clientIDInt)
		APIResponse, err = c.NewOrder(ctx,
			currencyID,
			o.Amount,
			o.Price,
			isBuyOrder,
			clientIDUint)
		if err != nil {
			return submitOrderResponse, err
		}
		responseMap, ok := APIResponse.(map[string]interface{})
		if !ok {
			return submitOrderResponse, errors.New("unable to type assert responseMap")
		}
		orderType, ok := responseMap["reply"].(string)
		if !ok {
			return submitOrderResponse, errors.New("unable to type assert orderType")
		}
		switch orderType {
		case "order_rejected":
			return submitOrderResponse, fmt.Errorf("clientOrderID: %v was rejected: %v", o.ClientID, responseMap["reasons"])
		case "order_filled":
			orderID, ok := responseMap["order_id"].(float64)
			if !ok {
				return submitOrderResponse, errors.New("unable to type assert orderID")
			}
			submitOrderResponse.OrderID = strconv.FormatFloat(orderID, 'f', -1, 64)
			submitOrderResponse.IsOrderPlaced = true
			submitOrderResponse.FullyMatched = true
			return submitOrderResponse, nil
		case "order_accepted":
			orderID, ok := responseMap["order_id"].(float64)
			if !ok {
				return submitOrderResponse, errors.New("unable to type assert orderID")
			}
			submitOrderResponse.OrderID = strconv.FormatFloat(orderID, 'f', -1, 64)
			submitOrderResponse.IsOrderPlaced = true
			return submitOrderResponse, nil
		}
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *COINUT) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *COINUT) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return err
	}
	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	fpair, err := c.FormatExchangeCurrency(o.Pair, asset.Spot)
	if err != nil {
		return err
	}

	currencyID := c.instrumentMap.LookupID(fpair.String())

	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var resp *CancelOrdersResponse
		resp, err = c.wsCancelOrder(&WsCancelOrderParameters{
			Currency: o.Pair,
			OrderID:  orderIDInt,
		})
		if err != nil {
			return err
		}
		if len(resp.Status) >= 1 && resp.Status[0] != "OK" {
			return errors.New(c.Name + " - Failed to cancel order " + o.ID)
		}
	} else {
		if currencyID == 0 {
			return errLookupInstrumentID
		}
		_, err = c.CancelExistingOrder(ctx, currencyID, orderIDInt)
		if err != nil {
			return err
		}
	}

	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (c *COINUT) CancelBatchOrders(_ context.Context, _ []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *COINUT) CancelAllOrders(ctx context.Context, details *order.Cancel) (order.CancelAllResponse, error) {
	if err := details.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var cancelAllOrdersResponse order.CancelAllResponse
	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return cancelAllOrdersResponse, err
	}
	cancelAllOrdersResponse.Status = make(map[string]string)
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		openOrders, err := c.wsGetOpenOrders(details.Pair.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		var ordersToCancel []WsCancelOrderParameters
		for i := range openOrders.Orders {
			var fpair currency.Pair
			fpair, err = c.FormatExchangeCurrency(details.Pair, asset.Spot)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if openOrders.Orders[i].InstrumentID == c.instrumentMap.LookupID(fpair.String()) {
				ordersToCancel = append(ordersToCancel, WsCancelOrderParameters{
					Currency: details.Pair,
					OrderID:  openOrders.Orders[i].OrderID,
				})
			}
		}
		resp, err := c.wsCancelOrders(ordersToCancel)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for i := range resp.Results {
			if openOrders.Orders[i].Status[0] != "OK" {
				cancelAllOrdersResponse.Status[strconv.FormatInt(openOrders.Orders[i].OrderID, 10)] = strings.Join(openOrders.Orders[i].Status, ",")
			}
		}
	} else {
		var allTheOrders []OrderResponse
		ids := c.instrumentMap.GetInstrumentIDs()
		for x := range ids {
			fpair, err := c.FormatExchangeCurrency(details.Pair, asset.Spot)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			if ids[x] == c.instrumentMap.LookupID(fpair.String()) {
				openOrders, err := c.GetOpenOrders(ctx, ids[x])
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				allTheOrders = append(allTheOrders, openOrders.Orders...)
			}
		}

		var allTheOrdersToCancel []CancelOrders
		for i := range allTheOrders {
			cancelOrder := CancelOrders{
				InstrumentID: allTheOrders[i].InstrumentID,
				OrderID:      allTheOrders[i].OrderID,
			}
			allTheOrdersToCancel = append(allTheOrdersToCancel, cancelOrder)
		}

		if len(allTheOrdersToCancel) > 0 {
			resp, err := c.CancelOrders(ctx, allTheOrdersToCancel)
			if err != nil {
				return cancelAllOrdersResponse, err
			}

			for i := range resp.Results {
				if resp.Results[i].Status != "OK" {
					cancelAllOrdersResponse.Status[strconv.FormatInt(resp.Results[i].OrderID, 10)] = resp.Results[i].Status
				}
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (c *COINUT) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *COINUT) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *COINUT) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *COINUT) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *COINUT) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !c.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *COINUT) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return nil, err
	}
	var orders []order.Detail
	var currenciesToCheck []string
	if len(req.Pairs) == 0 {
		for i := range req.Pairs {
			fpair, err := c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
			if err != nil {
				return nil, err
			}
			currenciesToCheck = append(currenciesToCheck, fpair.String())
		}
	} else {
		for k := range c.instrumentMap.Instruments {
			currenciesToCheck = append(currenciesToCheck, k)
		}
	}
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for x := range currenciesToCheck {
			openOrders, err := c.wsGetOpenOrders(currenciesToCheck[x])
			if err != nil {
				return nil, err
			}
			for i := range openOrders.Orders {
				p, err := currency.NewPairFromString(currenciesToCheck[x])
				if err != nil {
					return nil, err
				}

				fpair, err := c.FormatExchangeCurrency(p, asset.Spot)
				if err != nil {
					return nil, err
				}

				orders = append(orders, order.Detail{
					Exchange:        c.Name,
					ID:              strconv.FormatInt(openOrders.Orders[i].OrderID, 10),
					Pair:            fpair,
					Side:            order.Side(openOrders.Orders[i].Side),
					Date:            time.Unix(0, openOrders.Orders[i].Timestamp),
					Status:          order.Active,
					Price:           openOrders.Orders[i].Price,
					Amount:          openOrders.Orders[i].Quantity,
					ExecutedAmount:  openOrders.Orders[i].Quantity - openOrders.Orders[i].OpenQuantity,
					RemainingAmount: openOrders.Orders[i].OpenQuantity,
				})
			}
		}
	} else {
		var instrumentsToUse []int64
		for x := range req.Pairs {
			curr, err := c.FormatExchangeCurrency(req.Pairs[x],
				asset.Spot)
			if err != nil {
				return nil, err
			}
			instrumentsToUse = append(instrumentsToUse,
				c.instrumentMap.LookupID(curr.String()))
		}
		if len(instrumentsToUse) == 0 {
			instrumentsToUse = c.instrumentMap.GetInstrumentIDs()
		}

		pairs, err := c.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}

		format, err := c.GetPairFormat(asset.Spot, true)
		if err != nil {
			return nil, err
		}

		for x := range instrumentsToUse {
			openOrders, err := c.GetOpenOrders(ctx, instrumentsToUse[x])
			if err != nil {
				return nil, err
			}
			for y := range openOrders.Orders {
				curr := c.instrumentMap.LookupInstrument(instrumentsToUse[x])
				p, err := currency.NewPairFromFormattedPairs(curr,
					pairs,
					format)
				if err != nil {
					return nil, err
				}

				orderSide := order.Side(strings.ToUpper(openOrders.Orders[y].Side))
				orderDate := time.Unix(openOrders.Orders[y].Timestamp, 0)
				orders = append(orders, order.Detail{
					ID:       strconv.FormatInt(openOrders.Orders[y].OrderID, 10),
					Amount:   openOrders.Orders[y].Quantity,
					Price:    openOrders.Orders[y].Price,
					Exchange: c.Name,
					Side:     orderSide,
					Date:     orderDate,
					Pair:     p,
				})
			}
		}
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *COINUT) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	err := c.loadInstrumentsIfNotLoaded()
	if err != nil {
		return nil, err
	}
	var allOrders []order.Detail
	if c.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for i := range req.Pairs {
			for j := int64(0); ; j += 100 {
				trades, err := c.wsGetTradeHistory(req.Pairs[i], j, 100)
				if err != nil {
					return allOrders, err
				}
				for x := range trades.Trades {
					curr := c.instrumentMap.LookupInstrument(trades.Trades[x].InstrumentID)
					p, err := currency.NewPairFromString(curr)
					if err != nil {
						return nil, err
					}

					detail := order.Detail{
						Exchange:        c.Name,
						ID:              strconv.FormatInt(trades.Trades[x].OrderID, 10),
						Pair:            p,
						Side:            order.Side(trades.Trades[x].Side),
						Date:            time.Unix(0, trades.Trades[x].Timestamp),
						Status:          order.Filled,
						Price:           trades.Trades[x].Price,
						Amount:          trades.Trades[x].Quantity,
						ExecutedAmount:  trades.Trades[x].Quantity - trades.Trades[x].OpenQuantity,
						RemainingAmount: trades.Trades[x].OpenQuantity,
					}
					detail.InferCostsAndTimes()
					allOrders = append(allOrders, detail)
				}
				if len(trades.Trades) < 100 {
					break
				}
			}
		}
	} else {
		var instrumentsToUse []int64
		for x := range req.Pairs {
			curr, err := c.FormatExchangeCurrency(req.Pairs[x],
				asset.Spot)
			if err != nil {
				return nil, err
			}

			instrumentID := c.instrumentMap.LookupID(curr.String())
			if instrumentID > 0 {
				instrumentsToUse = append(instrumentsToUse, instrumentID)
			}
		}
		if len(instrumentsToUse) == 0 {
			instrumentsToUse = c.instrumentMap.GetInstrumentIDs()
		}

		pairs, err := c.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}

		format, err := c.GetPairFormat(asset.Spot, true)
		if err != nil {
			return nil, err
		}

		for x := range instrumentsToUse {
			orders, err := c.GetTradeHistory(ctx, instrumentsToUse[x], -1, -1)
			if err != nil {
				return nil, err
			}
			for y := range orders.Trades {
				curr := c.instrumentMap.LookupInstrument(instrumentsToUse[x])
				p, err := currency.NewPairFromFormattedPairs(curr,
					pairs,
					format)
				if err != nil {
					return nil, err
				}

				orderSide := order.Side(strings.ToUpper(orders.Trades[y].Order.Side))
				orderDate := time.Unix(orders.Trades[y].Order.Timestamp, 0)
				allOrders = append(allOrders, order.Detail{
					ID:       strconv.FormatInt(orders.Trades[y].Order.OrderID, 10),
					Amount:   orders.Trades[y].Order.Quantity,
					Price:    orders.Trades[y].Order.Price,
					Exchange: c.Name,
					Side:     orderSide,
					Date:     orderDate,
					Pair:     p,
				})
			}
		}
	}

	order.FilterOrdersByTimeRange(&allOrders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&allOrders, req.Side)
	return allOrders, nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (c *COINUT) AuthenticateWebsocket(ctx context.Context) error {
	return c.wsAuthenticate(ctx)
}

func (c *COINUT) loadInstrumentsIfNotLoaded() error {
	if !c.instrumentMap.IsLoaded() {
		if c.Websocket.IsConnected() {
			_, err := c.WsGetInstruments()
			if err != nil {
				return err
			}
		} else {
			err := c.SeedInstruments(context.TODO())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (c *COINUT) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := c.UpdateAccountInfo(ctx, assetType)
	return c.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (c *COINUT) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (c *COINUT) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
