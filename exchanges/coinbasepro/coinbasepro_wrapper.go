package coinbasepro

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
func (c *CoinbasePro) GetDefaultConfig() (*config.ExchangeConfig, error) {
	c.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = c.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = c.BaseCurrencies

	err := c.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if c.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = c.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (c *CoinbasePro) SetDefaults() {
	c.Name = "CoinbasePro"
	c.Enabled = true
	c.Verbose = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresSecret = true
	c.API.CredentialsValidator.RequiresClientID = true
	c.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
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
				DepositHistory:    true,
				WithdrawalHistory: true,
				UserTradeHistory:  true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
				FiatDeposit:       true,
				FiatWithdraw:      true,
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
				CandleHistory:     true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageSequenceNumbers: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.AutoWithdrawFiatWithAPIPermission,
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
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.OneHour.Word():    true,
					kline.SixHour.Word():    true,
					kline.OneDay.Word():     true,
				},
				ResultLimit: 300,
			},
		},
	}

	c.Requester = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	c.API.Endpoints = c.NewEndpoints()
	err = c.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      coinbaseproAPIURL,
		exchange.RestSandbox:   coinbaseproSandboxAPIURL,
		exchange.WebsocketSpot: coinbaseproWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	c.Websocket = stream.New()
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup initialises the exchange parameters with the current configuration
func (c *CoinbasePro) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		c.SetEnabled(false)
		return nil
	}

	err := c.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := c.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = c.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       coinbaseproWebsocketURL,
		ExchangeName:                     exch.Name,
		RunningURL:                       wsRunningURL,
		Connector:                        c.WsConnect,
		Subscriber:                       c.Subscribe,
		UnSubscriber:                     c.Unsubscribe,
		GenerateSubscriptions:            c.GenerateDefaultSubscriptions,
		Features:                         &c.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.OrderbookConfig.WebsocketBufferLimit,
		BufferEnabled:                    exch.OrderbookConfig.WebsocketBufferEnabled,
		SortBuffer:                       true,
	})
	if err != nil {
		return err
	}

	return c.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the coinbasepro go routine
func (c *CoinbasePro) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		c.Run()
		wg.Done()
	}()
}

// Run implements the coinbasepro wrapper
func (c *CoinbasePro) Run() {
	if c.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			c.Name,
			common.IsEnabled(c.Websocket.IsEnabled()),
			coinbaseproWebsocketURL)
		c.PrintEnabledPairs()
	}

	forceUpdate := false
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
		p, err = currency.NewPairsFromStrings([]string{currency.BTC.String() +
			format.Delimiter +
			currency.USD.String()})
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				c.Name,
				err)
		} else {
			log.Warn(log.ExchangeSys,
				"Enabled pairs for CoinbasePro reset due to config upgrade, please enable the ones you would like to use again")
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

	if !c.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = c.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *CoinbasePro) FetchTradablePairs(asset asset.Item) ([]string, error) {
	pairs, err := c.GetProducts()
	if err != nil {
		return nil, err
	}

	format, err := c.GetPairFormat(asset, false)
	if err != nil {
		return nil, err
	}

	var products []string
	for x := range pairs {
		products = append(products, pairs[x].BaseCurrency+
			format.Delimiter+
			pairs[x].QuoteCurrency)
	}

	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *CoinbasePro) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(asset.Spot)
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
// coinbasepro exchange
func (c *CoinbasePro) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = c.Name
	accountBalance, err := c.GetAccounts()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for i := range accountBalance {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance[i].Currency)
		exchangeCurrency.TotalValue = accountBalance[i].Available
		exchangeCurrency.Hold = accountBalance[i].Hold

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
func (c *CoinbasePro) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(c.Name, assetType)
	if err != nil {
		return c.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fpair, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tick, err := c.GetTicker(fpair.String())
	if err != nil {
		return nil, err
	}
	stats, err := c.GetStats(fpair.String())
	if err != nil {
		return nil, err
	}

	tickerPrice := &ticker.Price{
		Last:         stats.Last,
		High:         stats.High,
		Low:          stats.Low,
		Bid:          tick.Bid,
		Ask:          tick.Ask,
		Volume:       tick.Volume,
		Open:         stats.Open,
		Pair:         p,
		LastUpdated:  tick.Time,
		ExchangeName: c.Name,
		AssetType:    assetType}

	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(c.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (c *CoinbasePro) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *CoinbasePro) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(c.Name, p, assetType)
	if err != nil {
		return c.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *CoinbasePro) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        c.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: c.CanVerifyOrderbook,
	}
	fpair, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := c.GetOrderbook(fpair.String(), 2)
	if err != nil {
		return book, err
	}

	obNew := orderbookNew.(OrderbookL1L2)
	for x := range obNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: obNew.Bids[x].Amount,
			Price:  obNew.Bids[x].Price})
	}

	for x := range obNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: obNew.Asks[x].Amount,
			Price:  obNew.Asks[x].Price})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(c.Name, p, assetType)
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (c *CoinbasePro) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (c *CoinbasePro) GetWithdrawalsHistory(cur currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (c *CoinbasePro) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData []Trade
	tradeData, err = c.GetTrades(p.String())
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].Side)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			Exchange:     c.Name,
			TID:          strconv.FormatInt(tradeData[i].TradeID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Size,
			Timestamp:    tradeData[i].Time,
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
func (c *CoinbasePro) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	fpair, err := c.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return submitOrderResponse, err
	}

	var response string
	switch s.Type {
	case order.Market:
		response, err = c.PlaceMarketOrder("",
			s.Amount,
			s.Amount,
			s.Side.Lower(),
			fpair.String(),
			"")
	case order.Limit:
		response, err = c.PlaceLimitOrder("",
			s.Price,
			s.Amount,
			s.Side.Lower(),
			"",
			"",
			fpair.String(),
			"",
			false)
	default:
		err = errors.New("order type not supported")
	}
	if err != nil {
		return submitOrderResponse, err
	}
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	if response != "" {
		submitOrderResponse.OrderID = response
	}

	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *CoinbasePro) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *CoinbasePro) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	return c.CancelExistingOrder(o.ID)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (c *CoinbasePro) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	// CancellAllExisting orders returns a list of successful cancellations, we're only interested in failures
	_, err := c.CancelAllExistingOrders("")
	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns order information based on order ID
func (c *CoinbasePro) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	genOrderDetail, errGo := c.GetOrder(orderID)
	if errGo != nil {
		return order.Detail{}, fmt.Errorf("error retrieving order %s : %s", orderID, errGo)
	}
	od, errOd := time.Parse(time.RFC3339, genOrderDetail.DoneAt)
	if errOd != nil {
		return order.Detail{}, fmt.Errorf("error parsing order done at time: %s", errOd)
	}
	os, errOs := order.StringToOrderStatus(genOrderDetail.Status)
	if errOs != nil {
		return order.Detail{}, fmt.Errorf("error parsing order status: %s", errOs)
	}
	tt, errOt := order.StringToOrderType(genOrderDetail.Type)
	if errOt != nil {
		return order.Detail{}, fmt.Errorf("error parsing order type: %s", errOt)
	}
	ss, errOss := order.StringToOrderSide(genOrderDetail.Side)
	if errOss != nil {
		return order.Detail{}, fmt.Errorf("error parsing order side: %s", errOss)
	}
	p, errP := currency.NewPairDelimiter(genOrderDetail.ProductID, "-")
	if errP != nil {
		return order.Detail{}, fmt.Errorf("error parsing order side: %s", errP)
	}

	response := order.Detail{
		Exchange:        c.GetName(),
		ID:              genOrderDetail.ID,
		Pair:            p,
		Side:            ss,
		Type:            tt,
		Date:            od,
		Status:          os,
		Price:           genOrderDetail.Price,
		Amount:          genOrderDetail.Size,
		ExecutedAmount:  genOrderDetail.FilledSize,
		RemainingAmount: genOrderDetail.Size - genOrderDetail.FilledSize,
		Fee:             genOrderDetail.FillFees,
	}
	fillResponse, errGF := c.GetFills(orderID, genOrderDetail.ProductID)
	if errGF != nil {
		return response, fmt.Errorf("error retrieving the order fills: %s", errGF)
	}
	for i := range fillResponse {
		trSi, errTSi := order.StringToOrderSide(fillResponse[i].Side)
		if errTSi != nil {
			return response, fmt.Errorf("error parsing order Side: %s", errTSi)
		}
		response.Trades = append(response.Trades, order.TradeHistory{
			Timestamp: fillResponse[i].CreatedAt,
			TID:       strconv.FormatInt(fillResponse[i].TradeID, 10),
			Price:     fillResponse[i].Price,
			Amount:    fillResponse[i].Size,
			Exchange:  c.GetName(),
			Type:      tt,
			Side:      trSi,
			Fee:       fillResponse[i].Fee,
		})
	}
	return response, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *CoinbasePro) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := c.WithdrawCrypto(withdrawRequest.Amount, withdrawRequest.Currency.String(), withdrawRequest.Crypto.Address)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.ID,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	paymentMethods, err := c.GetPayMethods()
	if err != nil {
		return nil, err
	}

	selectedWithdrawalMethod := PaymentMethod{}
	for i := range paymentMethods {
		if withdrawRequest.Fiat.Bank.BankName == paymentMethods[i].Name {
			selectedWithdrawalMethod = paymentMethods[i]
			break
		}
	}
	if selectedWithdrawalMethod.ID == "" {
		return nil, fmt.Errorf("could not find payment method '%v'. Check the name via the website and try again", withdrawRequest.Fiat.Bank.BankName)
	}

	resp, err := c.WithdrawViaPaymentMethod(withdrawRequest.Amount, withdrawRequest.Currency.String(), selectedWithdrawalMethod.ID)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		Status: resp.ID,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := c.WithdrawFiatFunds(withdrawRequest)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     v.ID,
		Status: v.Status,
	}, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !c.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *CoinbasePro) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var respOrders []GeneralizedOrderResponse
	for i := range req.Pairs {
		fpair, err := c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}

		resp, err := c.GetOrders([]string{"open", "pending", "active"},
			fpair.String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	format, err := c.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range respOrders {
		var curr currency.Pair
		curr, err = currency.NewPairDelimiter(respOrders[i].ProductID,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderSide := order.Side(strings.ToUpper(respOrders[i].Side))
		orderType := order.Type(strings.ToUpper(respOrders[i].Type))
		orders = append(orders, order.Detail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			Type:           orderType,
			Date:           respOrders[i].CreatedAt,
			Side:           orderSide,
			Pair:           curr,
			Exchange:       c.Name,
		})
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *CoinbasePro) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var respOrders []GeneralizedOrderResponse
	for i := range req.Pairs {
		fpair, err := c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}
		resp, err := c.GetOrders([]string{"done", "settled"},
			fpair.String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	format, err := c.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range respOrders {
		var curr currency.Pair
		curr, err = currency.NewPairDelimiter(respOrders[i].ProductID,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderSide := order.Side(strings.ToUpper(respOrders[i].Side))
		orderType := order.Type(strings.ToUpper(respOrders[i].Type))
		orders = append(orders, order.Detail{
			ID:             respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			Type:           orderType,
			Date:           respOrders[i].CreatedAt,
			Side:           orderSide,
			Pair:           curr,
			Exchange:       c.Name,
		})
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// checkInterval checks allowable interval
func checkInterval(i time.Duration) (int64, error) {
	switch i.Seconds() {
	case 60:
		return 60, nil
	case 300:
		return 300, nil
	case 900:
		return 900, nil
	case 3600:
		return 3600, nil
	case 21600:
		return 21600, nil
	case 86400:
		return 86400, nil
	}
	return 0, fmt.Errorf("interval not allowed %v", i.Seconds())
}

// GetHistoricCandles returns a set of candle between two time periods for a
// designated time period
func (c *CoinbasePro) GetHistoricCandles(p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := c.ValidateKline(p, a, interval); err != nil {
		return kline.Item{}, err
	}

	if kline.TotalCandlesPerInterval(start, end, interval) > float64(c.Features.Enabled.Kline.ResultLimit) {
		return kline.Item{}, errors.New(kline.ErrRequestExceedsExchangeLimits)
	}

	candles := kline.Item{
		Exchange: c.Name,
		Pair:     p,
		Asset:    a,
		Interval: interval,
	}

	gran, err := strconv.ParseInt(c.FormatExchangeKlineInterval(interval), 10, 64)
	if err != nil {
		return kline.Item{}, err
	}

	formatP, err := c.FormatExchangeCurrency(p, a)
	if err != nil {
		return kline.Item{}, err
	}

	history, err := c.GetHistoricRates(formatP.String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		gran)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range history {
		candles.Candles = append(candles.Candles, kline.Candle{
			Time:   time.Unix(history[x].Time, 0),
			Low:    history[x].Low,
			High:   history[x].High,
			Open:   history[x].Open,
			Close:  history[x].Close,
			Volume: history[x].Volume,
		})
	}

	candles.SortCandlesByTimestamp(false)
	return candles, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (c *CoinbasePro) GetHistoricCandlesExtended(p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := c.ValidateKline(p, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: c.Name,
		Pair:     p,
		Asset:    a,
		Interval: interval,
	}

	gran, err := strconv.ParseInt(c.FormatExchangeKlineInterval(interval), 10, 64)
	if err != nil {
		return kline.Item{}, err
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, c.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := c.FormatExchangeCurrency(p, a)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range dates.Ranges {
		var history []History
		history, err = c.GetHistoricRates(formattedPair.String(),
			dates.Ranges[x].Start.Time.Format(time.RFC3339),
			dates.Ranges[x].End.Time.Format(time.RFC3339),
			gran)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range history {
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   time.Unix(history[i].Time, 0),
				Low:    history[i].Low,
				High:   history[i].High,
				Open:   history[i].Open,
				Close:  history[i].Close,
				Volume: history[i].Volume,
			})
		}
	}
	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", c.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (c *CoinbasePro) ValidateCredentials(assetType asset.Item) error {
	_, err := c.UpdateAccountInfo(assetType)
	return c.CheckTransientError(err)
}
