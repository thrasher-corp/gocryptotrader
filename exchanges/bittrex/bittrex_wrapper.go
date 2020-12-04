package bittrex

import (
	"errors"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (b *Bittrex) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults method assignes the default values for Bittrex
func (b *Bittrex) SetDefaults() {
	b.Name = "Bittrex"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := b.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				GetOrders:           true,
				CancelOrder:         true,
				SubmitOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(bittrexRateInterval, bittrexRequestRate)))

	b.API.Endpoints.URLDefault = bittrexAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
}

// Setup method sets current configuration details if enabled
func (b *Bittrex) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	return b.SetupDefaults(exch)
}

// Start starts the Bittrex go routine
func (b *Bittrex) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bittrex wrapper
func (b *Bittrex) Run() {
	if b.Verbose {
		b.PrintEnabledPairs()
	}

	forceUpdate := false
	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update currencies. Err: %s\n",
			b.Name,
			err)
		return
	}

	pairs, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update currencies. Err: %s\n",
			b.Name,
			err)
		return
	}

	avail, err := b.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update currencies. Err: %s\n",
			b.Name,
			err)
		return
	}

	if !common.StringDataContains(pairs.Strings(), format.Delimiter) ||
		!common.StringDataContains(avail.Strings(), format.Delimiter) {
		forceUpdate = true
		log.Warn(log.ExchangeSys, "Available pairs for Bittrex reset due to config upgrade, please enable the ones you would like again")
		pairs, err = currency.NewPairsFromStrings([]string{currency.USDT.String() +
			format.Delimiter +
			currency.BTC.String()})
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				b.Name,
				err)
		} else {
			err = b.UpdatePairs(pairs, asset.Spot, true, true)
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
func (b *Bittrex) FetchTradablePairs(asset asset.Item) ([]string, error) {
	markets, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range markets.Result {
		if !markets.Result[x].IsActive || markets.Result[x].MarketName == "" {
			continue
		}
		pairs = append(pairs, markets.Result[x].MarketName)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bittrex) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}
	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return b.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateAccountInfo Retrieves balances for all enabled currencies for the
// Bittrex exchange
func (b *Bittrex) UpdateAccountInfo() (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = b.Name
	accountBalance, err := b.GetAccountBalances()
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for i := range accountBalance.Result {
		var exchangeCurrency account.Balance
		exchangeCurrency.CurrencyName = currency.NewCode(accountBalance.Result[i].Currency)
		exchangeCurrency.TotalValue = accountBalance.Result[i].Balance
		exchangeCurrency.Hold = accountBalance.Result[i].Balance - accountBalance.Result[i].Available
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
func (b *Bittrex) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name)
	if err != nil {
		return b.UpdateAccountInfo()
	}

	return acc, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bittrex) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	ticks, err := b.GetMarketSummaries()
	if err != nil {
		return nil, err
	}

	pairs, err := b.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}

	for j := range ticks.Result {
		cp, err := currency.NewPairFromString(ticks.Result[j].MarketName)
		if err != nil {
			return nil, err
		}
		if !pairs.Contains(cp, true) {
			continue
		}
		tickerTime, err := parseTime(ticks.Result[j].TimeStamp)
		if err != nil {
			return nil, err
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Last:         ticks.Result[j].Last,
			High:         ticks.Result[j].High,
			Low:          ticks.Result[j].Low,
			Bid:          ticks.Result[j].Bid,
			Ask:          ticks.Result[j].Ask,
			Volume:       ticks.Result[j].BaseVolume,
			QuoteVolume:  ticks.Result[j].Volume,
			Close:        ticks.Result[j].PrevDay,
			Pair:         cp,
			LastUpdated:  tickerTime,
			ExchangeName: b.Name,
			AssetType:    assetType})
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bittrex) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bittrex) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bittrex) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fpair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	orderbookNew, err := b.GetOrderbook(fpair.String())
	if err != nil {
		return nil, err
	}

	orderBook := new(orderbook.Base)
	for x := range orderbookNew.Result.Buy {
		orderBook.Bids = append(orderBook.Bids,
			orderbook.Item{
				Amount: orderbookNew.Result.Buy[x].Quantity,
				Price:  orderbookNew.Result.Buy[x].Rate,
			},
		)
	}

	for x := range orderbookNew.Result.Sell {
		orderBook.Asks = append(orderBook.Asks,
			orderbook.Item{
				Amount: orderbookNew.Result.Sell[x].Quantity,
				Price:  orderbookNew.Result.Sell[x].Rate,
			},
		)
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

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bittrex) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bittrex) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Bittrex) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tradeData, err := b.GetMarketHistory(p.String())
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData.Result {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData.Result[i].OrderType)
		if err != nil {
			return nil, err
		}
		var ts time.Time
		ts, err = time.Parse("2006-01-02T15:04:05.999999999", tradeData.Result[i].Timestamp)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			Exchange:     b.Name,
			TID:          strconv.FormatInt(tradeData.Result[i].ID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData.Result[i].Price,
			Amount:       tradeData.Result[i].Quantity,
			Timestamp:    ts,
		})
	}

	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bittrex) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *Bittrex) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	buy := s.Side == order.Buy
	if s.Type != order.Limit {
		return submitOrderResponse,
			errors.New("limit orders only supported on exchange")
	}

	fPair, err := b.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	var response UUID
	if buy {
		response, err = b.PlaceBuyLimit(fPair.String(),
			s.Amount,
			s.Price)
	} else {
		response, err = b.PlaceSellLimit(fPair.String(),
			s.Amount,
			s.Price)
	}
	if err != nil {
		return submitOrderResponse, err
	}
	if response.Result.ID != "" {
		submitOrderResponse.OrderID = response.Result.ID
	}

	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bittrex) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bittrex) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	_, err := b.CancelExistingOrder(o.ID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Bittrex) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bittrex) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := b.GetOpenOrders("")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range openOrders.Result {
		_, err := b.CancelExistingOrder(openOrders.Result[i].OrderUUID)
		if err != nil {
			cancelAllOrdersResponse.Status[openOrders.Result[i].OrderUUID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (b *Bittrex) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bittrex) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	depositAddr, err := b.GetCryptoDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	return depositAddr.Result.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bittrex) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}

	uuid, err := b.Withdraw(withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: uuid.Result.ID,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bittrex) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bittrex) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var currPair string
	if len(req.Pairs) == 1 {
		fPair, err := b.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = fPair.String()
	}

	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	resp, err := b.GetOpenOrders(currPair)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp.Result {
		orderDate, err := parseTime(resp.Result[i].Opened)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				b.Name,
				"GetActiveOrders",
				resp.Result[i].OrderUUID,
				resp.Result[i].Opened)
		}

		pair, err := currency.NewPairDelimiter(resp.Result[i].Exchange,
			format.Delimiter)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse currency pair %v",
				b.Name,
				"GetActiveOrders",
				resp.Result[i].OrderUUID,
				err)
		}
		orderType := order.Type(strings.ToUpper(resp.Result[i].Type))

		orders = append(orders, order.Detail{
			Amount:          resp.Result[i].Quantity,
			RemainingAmount: resp.Result[i].QuantityRemaining,
			Price:           resp.Result[i].Price,
			Date:            orderDate,
			ID:              resp.Result[i].OrderUUID,
			Exchange:        b.Name,
			Type:            orderType,
			Pair:            pair,
		})
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bittrex) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var currPair string
	if len(req.Pairs) == 1 {
		fPair, err := b.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = fPair.String()
	}

	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	resp, err := b.GetOrderHistoryForCurrency(currPair)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp.Result {
		orderDate, err := parseTime(resp.Result[i].TimeStamp)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				b.Name,
				"GetOrderHistory",
				resp.Result[i].OrderUUID,
				resp.Result[i].Opened)
		}

		pair, err := currency.NewPairDelimiter(resp.Result[i].Exchange,
			format.Delimiter)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse currency pair %v",
				b.Name,
				"GetOrderHistory",
				resp.Result[i].OrderUUID,
				err)
		}
		orderType := order.Type(strings.ToUpper(resp.Result[i].Type))

		orders = append(orders, order.Detail{
			Amount:          resp.Result[i].Quantity,
			RemainingAmount: resp.Result[i].QuantityRemaining,
			Price:           resp.Result[i].Price,
			Date:            orderDate,
			ID:              resp.Result[i].OrderUUID,
			Exchange:        b.Name,
			Type:            orderType,
			Fee:             resp.Result[i].Commission,
			Pair:            pair,
		})
	}

	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Bittrex) ValidateCredentials() error {
	_, err := b.UpdateAccountInfo()
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bittrex) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *Bittrex) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
