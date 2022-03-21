package coinbasepro

import (
	"context"
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
func (c *CoinbasePro) GetDefaultConfig() (*config.Exchange, error) {
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
		ExchangeConfig:        exch,
		DefaultURL:            coinbaseproWebsocketURL,
		RunningURL:            wsRunningURL,
		Connector:             c.WsConnect,
		Subscriber:            c.Subscribe,
		Unsubscriber:          c.Unsubscribe,
		GenerateSubscriptions: c.GenerateDefaultSubscriptions,
		Features:              &c.Features.Supports.WebsocketCapabilities,
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
func (c *CoinbasePro) Start(wg *sync.WaitGroup) error {
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

	err := c.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *CoinbasePro) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	pairs, err := c.GetProducts(ctx)
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
func (c *CoinbasePro) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
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
			CurrencyName:           currency.NewCode(accountBalance[i].Currency),
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

	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (c *CoinbasePro) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(c.Name, assetType)
	if err != nil {
		return c.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (c *CoinbasePro) UpdateTickers(ctx context.Context, a asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fpair, err := c.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tick, err := c.GetTicker(ctx, fpair.String())
	if err != nil {
		return nil, err
	}
	stats, err := c.GetStats(ctx, fpair.String())
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

	orderbookNew, err := c.GetOrderbook(ctx, fpair.String(), 2)
	if err != nil {
		return book, err
	}

	obNew, ok := orderbookNew.(OrderbookL1L2)
	if !ok {
		return book, errors.New("unable to type assert orderbook data")
	}
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
func (c *CoinbasePro) GetFundingHistory(_ context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (c *CoinbasePro) GetWithdrawalsHistory(_ context.Context, _ currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
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
func (c *CoinbasePro) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
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
		response, err = c.PlaceMarketOrder(ctx,
			"",
			s.Amount,
			s.Amount,
			s.Side.Lower(),
			fpair.String(),
			"")
	case order.Limit:
		response, err = c.PlaceLimitOrder(ctx,
			"",
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
func (c *CoinbasePro) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (c *CoinbasePro) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	return c.CancelExistingOrder(ctx, o.ID)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (c *CoinbasePro) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	// CancellAllExisting orders returns a list of successful cancellations, we're only interested in failures
	_, err := c.CancelAllExistingOrders(ctx, "")
	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns order information based on order ID
func (c *CoinbasePro) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	genOrderDetail, errGo := c.GetOrder(ctx, orderID)
	if errGo != nil {
		return order.Detail{}, fmt.Errorf("error retrieving order %s : %s", orderID, errGo)
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
		Date:            genOrderDetail.DoneAt,
		Status:          os,
		Price:           genOrderDetail.Price,
		Amount:          genOrderDetail.Size,
		ExecutedAmount:  genOrderDetail.FilledSize,
		RemainingAmount: genOrderDetail.Size - genOrderDetail.FilledSize,
		Fee:             genOrderDetail.FillFees,
	}
	fillResponse, errGF := c.GetFills(ctx, orderID, genOrderDetail.ProductID)
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
func (c *CoinbasePro) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var respOrders []GeneralizedOrderResponse
	for i := range req.Pairs {
		fpair, err := c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}

		resp, err := c.GetOrders(ctx,
			[]string{"open", "pending", "active"},
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
func (c *CoinbasePro) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var respOrders []GeneralizedOrderResponse
	if len(req.Pairs) > 0 {
		for i := range req.Pairs {
			fpair, err := c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
			if err != nil {
				return nil, err
			}
			resp, err := c.GetOrders(ctx,
				[]string{"done"},
				fpair.String())
			if err != nil {
				return nil, err
			}
			respOrders = append(respOrders, resp...)
		}
	} else {
		resp, err := c.GetOrders(ctx,
			[]string{"done"},
			"")
		if err != nil {
			return nil, err
		}
		respOrders = resp
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
		orderStatus, err := order.StringToOrderStatus(respOrders[i].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", c.Name, err)
		}
		orderType := order.Type(strings.ToUpper(respOrders[i].Type))
		detail := order.Detail{
			ID:              respOrders[i].ID,
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
			Side:            orderSide,
			Status:          orderStatus,
			Pair:            curr,
			Price:           respOrders[i].Price,
			Exchange:        c.Name,
		}
		detail.InferCostsAndTimes()
		orders = append(orders, detail)
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
func (c *CoinbasePro) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
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

	history, err := c.GetHistoricRates(ctx,
		formatP.String(),
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
func (c *CoinbasePro) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
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
		history, err = c.GetHistoricRates(ctx,
			formattedPair.String(),
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
func (c *CoinbasePro) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := c.UpdateAccountInfo(ctx, assetType)
	return c.CheckTransientError(err)
}
