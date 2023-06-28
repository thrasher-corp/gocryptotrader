package coinbasepro

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
func (c *CoinbasePro) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
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
		err = c.UpdateTradablePairs(ctx, true)
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
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 300,
			},
		},
	}

	c.Requester, err = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
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
func (c *CoinbasePro) Setup(exch *config.Exchange) error {
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
		ExchangeConfig:         exch,
		DefaultURL:             coinbaseproWebsocketURL,
		RunningURL:             wsRunningURL,
		Connector:              c.WsConnect,
		Subscriber:             c.Subscribe,
		Unsubscriber:           c.Unsubscribe,
		GenerateSubscriptions:  c.GenerateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &c.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer: true,
		},
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
func (c *CoinbasePro) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		c.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the coinbasepro wrapper
func (c *CoinbasePro) Run(ctx context.Context) {
	if c.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			c.Name,
			common.IsEnabled(c.Websocket.IsEnabled()),
			coinbaseproWebsocketURL)
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
			p, err = currency.NewPairsFromStrings([]string{currency.BTC.String() +
				format.Delimiter +
				currency.USD.String()})
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					c.Name,
					err)
			} else {
				forceUpdate = true
				log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, c.Name, asset.Spot, p)

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

	err := c.UpdateTradablePairs(ctx, forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *CoinbasePro) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	products, err := c.GetProducts(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(products))
	for x := range products {
		if products[x].TradingDisabled {
			continue
		}
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(products[x].ID, currency.DashDelimiter)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *CoinbasePro) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := c.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = c.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return c.EnsureOnePairEnabled()
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// coinbasepro exchange
func (c *CoinbasePro) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = c.Name
	accountBalance, err := c.GetAccounts(ctx)
	if err != nil {
		return response, err
	}

	accountCurrencies := make(map[string][]account.Balance)
	for i := range accountBalance {
		profileID := accountBalance[i].ProfileID
		currencies := accountCurrencies[profileID]
		accountCurrencies[profileID] = append(currencies, account.Balance{
			Currency:               currency.NewCode(accountBalance[i].Currency),
			Total:                  accountBalance[i].Balance,
			Hold:                   accountBalance[i].Hold,
			Free:                   accountBalance[i].Available,
			AvailableWithoutBorrow: accountBalance[i].Available - accountBalance[i].FundedAmount,
			Borrowed:               accountBalance[i].FundedAmount,
		})
	}

	if response.Accounts, err = account.CollectBalances(accountCurrencies, assetType); err != nil {
		return account.Holdings{}, err
	}

	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (c *CoinbasePro) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(c.Name, creds, assetType)
	if err != nil {
		return c.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (c *CoinbasePro) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := c.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tick, err := c.GetTicker(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	stats, err := c.GetStats(ctx, fPair.String())
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
		AssetType:    a}

	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(c.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (c *CoinbasePro) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *CoinbasePro) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(c.Name, p, assetType)
	if err != nil {
		return c.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *CoinbasePro) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := c.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        c.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: c.CanVerifyOrderbook,
	}
	fPair, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := c.GetOrderbook(ctx, fPair.String(), 2)
	if err != nil {
		return book, err
	}

	obNew, ok := orderbookNew.(OrderbookL1L2)
	if !ok {
		return book, common.GetTypeAssertError("OrderbookL1L2", orderbookNew)
	}

	book.Bids = make(orderbook.Items, len(obNew.Bids))
	for x := range obNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: obNew.Bids[x].Amount,
			Price:  obNew.Bids[x].Price,
		}
	}

	book.Asks = make(orderbook.Items, len(obNew.Asks))
	for x := range obNew.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: obNew.Asks[x].Amount,
			Price:  obNew.Asks[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(c.Name, p, assetType)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (c *CoinbasePro) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (c *CoinbasePro) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	// while fetching withdrawal history is possible, the API response lacks any useful information
	// like the currency withdrawn and thus is unsupported. If that position changes, use GetTransfers(...)
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (c *CoinbasePro) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData []Trade
	tradeData, err = c.GetTrades(ctx, p.String())
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     c.Name,
			TID:          strconv.FormatInt(tradeData[i].TradeID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Size,
			Timestamp:    tradeData[i].Time,
		}
	}

	err = c.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (c *CoinbasePro) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	fPair, err := c.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	var orderID string
	switch s.Type {
	case order.Market:
		orderID, err = c.PlaceMarketOrder(ctx,
			"",
			s.Amount,
			s.QuoteAmount,
			s.Side.Lower(),
			fPair.String(),
			"")
	case order.Limit:
		timeInForce := CoinbaseRequestParamsTimeGTC
		if s.ImmediateOrCancel {
			timeInForce = CoinbaseRequestParamsTimeIOC
		}
		orderID, err = c.PlaceLimitOrder(ctx,
			"",
			s.Price,
			s.Amount,
			s.Side.Lower(),
			timeInForce,
			"",
			fPair.String(),
			"",
			false)
	default:
		err = fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, s.Type)
	}
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(orderID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *CoinbasePro) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *CoinbasePro) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	return c.CancelExistingOrder(ctx, o.OrderID)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (c *CoinbasePro) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	// CancellAllExisting orders returns a list of successful cancellations, we're only interested in failures
	_, err := c.CancelAllExistingOrders(ctx, "")
	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns order information based on order ID
func (c *CoinbasePro) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	genOrderDetail, err := c.GetOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving order %s : %w", orderID, err)
	}
	orderStatus, err := order.StringToOrderStatus(genOrderDetail.Status)
	if err != nil {
		return nil, fmt.Errorf("error parsing order status: %w", err)
	}
	orderType, err := order.StringToOrderType(genOrderDetail.Type)
	if err != nil {
		return nil, fmt.Errorf("error parsing order type: %w", err)
	}
	orderSide, err := order.StringToOrderSide(genOrderDetail.Side)
	if err != nil {
		return nil, fmt.Errorf("error parsing order side: %w", err)
	}
	pair, err := currency.NewPairDelimiter(genOrderDetail.ProductID, "-")
	if err != nil {
		return nil, fmt.Errorf("error parsing order pair: %w", err)
	}

	response := order.Detail{
		Exchange:        c.GetName(),
		OrderID:         genOrderDetail.ID,
		Pair:            pair,
		Side:            orderSide,
		Type:            orderType,
		Date:            genOrderDetail.DoneAt,
		Status:          orderStatus,
		Price:           genOrderDetail.Price,
		Amount:          genOrderDetail.Size,
		ExecutedAmount:  genOrderDetail.FilledSize,
		RemainingAmount: genOrderDetail.Size - genOrderDetail.FilledSize,
		Fee:             genOrderDetail.FillFees,
	}
	fillResponse, err := c.GetFills(ctx, orderID, genOrderDetail.ProductID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving the order fills: %w", err)
	}
	for i := range fillResponse {
		var fillSide order.Side
		fillSide, err = order.StringToOrderSide(fillResponse[i].Side)
		if err != nil {
			return nil, fmt.Errorf("error parsing order Side: %w", err)
		}
		response.Trades = append(response.Trades, order.TradeHistory{
			Timestamp: fillResponse[i].CreatedAt,
			TID:       strconv.FormatInt(fillResponse[i].TradeID, 10),
			Price:     fillResponse[i].Price,
			Amount:    fillResponse[i].Size,
			Exchange:  c.GetName(),
			Type:      orderType,
			Side:      fillSide,
			Fee:       fillResponse[i].Fee,
		})
	}
	return &response, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *CoinbasePro) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := c.WithdrawCrypto(ctx,
		withdrawRequest.Amount,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.ID,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	paymentMethods, err := c.GetPayMethods(ctx)
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

	resp, err := c.WithdrawViaPaymentMethod(ctx,
		withdrawRequest.Amount,
		withdrawRequest.Currency.String(),
		selectedWithdrawalMethod.ID)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		Status: resp.ID,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := c.WithdrawFiatFunds(ctx, withdrawRequest)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     v.ID,
		Status: v.Status,
	}, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !c.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *CoinbasePro) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var respOrders []GeneralizedOrderResponse
	var fPair currency.Pair
	for i := range req.Pairs {
		fPair, err = c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}

		var resp []GeneralizedOrderResponse
		resp, err = c.GetOrders(ctx,
			[]string{"open", "pending", "active"},
			fPair.String())
		if err != nil {
			return nil, err
		}
		respOrders = append(respOrders, resp...)
	}

	format, err := c.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(respOrders))
	for i := range respOrders {
		var curr currency.Pair
		curr, err = currency.NewPairDelimiter(respOrders[i].ProductID,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(respOrders[i].Side)
		if err != nil {
			return nil, err
		}
		var orderType order.Type
		orderType, err = order.StringToOrderType(respOrders[i].Type)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", c.Name, err)
		}
		orders[i] = order.Detail{
			OrderID:        respOrders[i].ID,
			Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			Type:           orderType,
			Date:           respOrders[i].CreatedAt,
			Side:           side,
			Pair:           curr,
			Exchange:       c.Name,
		}
	}
	return req.Filter(c.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *CoinbasePro) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var respOrders []GeneralizedOrderResponse
	if len(req.Pairs) > 0 {
		var fPair currency.Pair
		var resp []GeneralizedOrderResponse
		for i := range req.Pairs {
			fPair, err = c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
			if err != nil {
				return nil, err
			}
			resp, err = c.GetOrders(ctx, []string{"done"}, fPair.String())
			if err != nil {
				return nil, err
			}
			respOrders = append(respOrders, resp...)
		}
	} else {
		respOrders, err = c.GetOrders(ctx, []string{"done"}, "")
		if err != nil {
			return nil, err
		}
	}

	format, err := c.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(respOrders))
	for i := range respOrders {
		var curr currency.Pair
		curr, err = currency.NewPairDelimiter(respOrders[i].ProductID,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(respOrders[i].Side)
		if err != nil {
			return nil, err
		}
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(respOrders[i].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", c.Name, err)
		}
		var orderType order.Type
		orderType, err = order.StringToOrderType(respOrders[i].Type)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", c.Name, err)
		}
		detail := order.Detail{
			OrderID:         respOrders[i].ID,
			Amount:          respOrders[i].Size,
			ExecutedAmount:  respOrders[i].FilledSize,
			RemainingAmount: respOrders[i].Size - respOrders[i].FilledSize,
			Cost:            respOrders[i].ExecutedValue,
			CostAsset:       curr.Quote,
			Type:            orderType,
			Date:            respOrders[i].CreatedAt,
			CloseTime:       respOrders[i].DoneAt,
			Fee:             respOrders[i].FillFees,
			FeeAsset:        curr.Quote,
			Side:            side,
			Status:          orderStatus,
			Pair:            curr,
			Price:           respOrders[i].Price,
			Exchange:        c.Name,
		}
		detail.InferCostsAndTimes()
		orders[i] = detail
	}
	return req.Filter(c.Name, orders), nil
}

// GetHistoricCandles returns a set of candle between two time periods for a
// designated time period
func (c *CoinbasePro) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := c.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	history, err := c.GetHistoricRates(ctx,
		req.RequestFormatted.String(),
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		int64(req.ExchangeInterval.Duration().Seconds()))
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(history))
	for x := range history {
		timeSeries[x] = kline.Candle{
			Time:   history[x].Time,
			Low:    history[x].Low,
			High:   history[x].High,
			Open:   history[x].Open,
			Close:  history[x].Close,
			Volume: history[x].Volume,
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (c *CoinbasePro) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := c.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var history []History
		history, err = c.GetHistoricRates(ctx,
			req.RequestFormatted.String(),
			req.RangeHolder.Ranges[x].Start.Time.Format(time.RFC3339),
			req.RangeHolder.Ranges[x].End.Time.Format(time.RFC3339),
			int64(req.ExchangeInterval.Duration().Seconds()))
		if err != nil {
			return nil, err
		}

		for i := range history {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   history[i].Time,
				Low:    history[i].Low,
				High:   history[i].High,
				Open:   history[i].Open,
				Close:  history[i].Close,
				Volume: history[i].Volume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (c *CoinbasePro) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := c.UpdateAccountInfo(ctx, assetType)
	return c.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (c *CoinbasePro) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	st, err := c.GetCurrentServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return st.ISO, nil
}
