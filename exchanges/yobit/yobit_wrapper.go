package yobit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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

// SetDefaults sets current default value for Yobit
func (y *Yobit) SetDefaults() {
	y.Name = "Yobit"
	y.Enabled = true
	y.Verbose = true
	y.API.CredentialsValidator.RequiresKey = true
	y.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Separator: currency.DashDelimiter}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	err := y.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	y.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	y.Requester, err = request.New(y.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		// Server responses are cached every 2 seconds.
		request.WithLimiter(request.NewBasicRateLimit(time.Second, 1, 1)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	y.API.Endpoints = y.NewEndpoints()
	err = y.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              apiPublicURL,
		exchange.RestSpotSupplementary: apiPrivateURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup sets exchange configuration parameters for Yobit
func (y *Yobit) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		y.SetEnabled(false)
		return nil
	}
	return y.SetupDefaults(exch)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (y *Yobit) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	info, err := y.GetInfo(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, len(info.Pairs))
	var target int
	for key := range info.Pairs {
		var pair currency.Pair
		pair, err = currency.NewPairFromString(key)
		if err != nil {
			return nil, err
		}
		pairs[target] = pair
		target++
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (y *Yobit) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := y.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = y.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return y.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (y *Yobit) UpdateTickers(ctx context.Context, a asset.Item) error {
	enabledPairs, err := y.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	pairsCollated, err := y.FormatExchangeCurrencies(enabledPairs, a)
	if err != nil {
		return err
	}

	result, err := y.GetTicker(ctx, pairsCollated)
	if err != nil {
		return err
	}

	for i := range enabledPairs {
		fPair, err := y.FormatExchangeCurrency(enabledPairs[i], a)
		if err != nil {
			return err
		}
		curr := fPair.Lower().String()
		if _, ok := result[curr]; !ok {
			continue
		}

		resultCurr := result[curr]
		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         enabledPairs[i],
			Last:         resultCurr.Last,
			Ask:          resultCurr.Sell,
			Bid:          resultCurr.Buy,
			Low:          resultCurr.Low,
			QuoteVolume:  resultCurr.VolumeCurrent,
			Volume:       resultCurr.Vol,
			ExchangeName: y.Name,
			AssetType:    a,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (y *Yobit) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := y.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(y.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (y *Yobit) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := y.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          y.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: y.ValidateOrderbook,
	}
	fPair, err := y.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}
	orderbookNew, err := y.GetDepth(ctx, fPair.String())
	if err != nil {
		return book, err
	}

	for i := range orderbookNew.Bids {
		book.Bids = append(book.Bids,
			orderbook.Level{
				Price:  orderbookNew.Bids[i][0],
				Amount: orderbookNew.Bids[i][1],
			})
	}

	for i := range orderbookNew.Asks {
		book.Asks = append(book.Asks,
			orderbook.Level{
				Price:  orderbookNew.Asks[i][0],
				Amount: orderbookNew.Asks[i][1],
			})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(y.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Yobit exchange
func (y *Yobit) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = y.Name
	accountBalance, err := y.GetAccountInformation(ctx)
	if err != nil {
		return response, err
	}

	currencies := make([]account.Balance, 0, len(accountBalance.FundsInclOrders))
	for x, y := range accountBalance.FundsInclOrders {
		var exchangeCurrency account.Balance
		exchangeCurrency.Currency = currency.NewCode(x)
		exchangeCurrency.Total = y
		for z, w := range accountBalance.Funds {
			if z == x {
				exchangeCurrency.Hold = y - w
				exchangeCurrency.Free = w
			}
		}

		currencies = append(currencies, exchangeCurrency)
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		AssetType:  assetType,
		Currencies: currencies,
	})

	creds, err := y.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (y *Yobit) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (y *Yobit) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (y *Yobit) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = y.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	var tradeData []Trade
	tradeData, err = y.GetTrades(ctx, p.String())
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		side := order.Buy
		if tradeData[i].Type == "ask" {
			side = order.Sell
		}
		resp[i] = trade.Data{
			Exchange:     y.Name,
			TID:          strconv.FormatInt(tradeData[i].TID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeData[i].Timestamp.Time(),
		}
	}

	err = y.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (y *Yobit) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
// Yobit only supports limit orders
func (y *Yobit) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(y.GetTradingRequirements()); err != nil {
		return nil, err
	}

	if s.Type != order.Limit {
		return nil, errors.New("only limit orders are allowed")
	}

	fPair, err := y.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	response, err := y.Trade(ctx,
		fPair.String(),
		s.Side.String(),
		s.Amount,
		s.Price)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(strconv.FormatInt(response, 10))
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (y *Yobit) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (y *Yobit) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	return y.CancelExistingOrder(ctx, orderIDInt)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (y *Yobit) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (y *Yobit) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	enabledPairs, err := y.GetEnabledPairs(asset.Spot)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	allActiveOrders := make([]map[string]ActiveOrders, len(enabledPairs))
	for i := range enabledPairs {
		fCurr, err := y.FormatExchangeCurrency(enabledPairs[i], asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		activeOrdersForPair, err := y.GetOpenOrders(ctx, fCurr.String())
		if err != nil {
			return cancelAllOrdersResponse, err
		}

		allActiveOrders[i] = activeOrdersForPair
	}

	for i := range allActiveOrders {
		for key := range allActiveOrders[i] {
			orderIDInt, err := strconv.ParseInt(key, 10, 64)
			if err != nil {
				cancelAllOrdersResponse.Status[key] = err.Error()
				continue
			}

			err = y.CancelExistingOrder(ctx, orderIDInt)
			if err != nil {
				cancelAllOrdersResponse.Status[key] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (y *Yobit) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	iOID, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	format, err := y.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}
	resp, err := y.GetOrderInformation(ctx, iOID)
	if err != nil {
		return nil, err
	}

	for id, orderInfo := range resp {
		if id != orderID {
			continue
		}
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(orderInfo.Pair, format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(orderInfo.Type)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			OrderID:  id,
			Amount:   orderInfo.Amount,
			Price:    orderInfo.Rate,
			Side:     side,
			Date:     orderInfo.TimestampCreated.Time(),
			Pair:     symbol,
			Exchange: y.Name,
		}, nil
	}
	return nil, fmt.Errorf("%w %v", order.ErrOrderNotFound, orderID)
}

// GetDepositAddress returns a deposit address for a specified currency
func (y *Yobit) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	if cryptocurrency.Equal(currency.XRP) {
		// {"success":1,"return":{"status":"online","blocks":65778672,"address":996707783,"processed_amount":0.00000000,"server_time":1629425030}}
		return nil, errors.New("XRP isn't supported as the API does not return a valid address")
	}

	addr, err := y.GetCryptoDepositAddress(ctx, cryptocurrency.String(), false)
	if err != nil {
		return nil, err
	}

	return &deposit.Address{Address: addr.Return.Address}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (y *Yobit) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := y.WithdrawCoinsToAddress(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Amount,
		withdrawRequest.Crypto.Address)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return &withdraw.ExchangeResponse{}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (y *Yobit) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (y *Yobit) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !y.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return y.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (y *Yobit) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	format, err := y.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for x := range req.Pairs {
		var fCurr currency.Pair
		fCurr, err = y.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		var resp map[string]ActiveOrders
		resp, err = y.GetOpenOrders(ctx, fCurr.String())
		if err != nil {
			return nil, err
		}

		for id := range resp {
			var symbol currency.Pair
			symbol, err = currency.NewPairDelimiter(resp[id].Pair, format.Delimiter)
			if err != nil {
				return nil, err
			}
			var side order.Side
			side, err = order.StringToOrderSide(resp[id].Type)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				OrderID:  id,
				Amount:   resp[id].Amount,
				Price:    resp[id].Rate,
				Side:     side,
				Date:     resp[id].TimestampCreated.Time(),
				Pair:     symbol,
				Exchange: y.Name,
			})
		}
	}
	return req.Filter(y.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (y *Yobit) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var allOrders []TradeHistory
	for x := range req.Pairs {
		var fPair currency.Pair
		fPair, err = y.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		var resp map[string]TradeHistory
		resp, err = y.GetTradeHistory(ctx,
			0,
			10000,
			math.MaxInt64,
			req.StartTime.Unix(),
			req.EndTime.Unix(),
			"DESC",
			fPair.String())
		if err != nil {
			return nil, err
		}

		for key := range resp {
			allOrders = append(allOrders, resp[key])
		}
	}

	format, err := y.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(allOrders))
	for i := range allOrders {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(allOrders[i].Pair, format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(allOrders[i].Type)
		if err != nil {
			return nil, err
		}
		detail := order.Detail{
			OrderID:              strconv.FormatFloat(allOrders[i].OrderID, 'f', -1, 64),
			Amount:               allOrders[i].Amount,
			ExecutedAmount:       allOrders[i].Amount,
			Price:                allOrders[i].Rate,
			AverageExecutedPrice: allOrders[i].Rate,
			Side:                 side,
			Status:               order.Filled,
			Date:                 allOrders[i].Timestamp.Time(),
			Pair:                 pair,
			Exchange:             y.Name,
		}
		detail.InferCostsAndTimes()
		orders[i] = detail
	}
	return req.Filter(y.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (y *Yobit) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := y.UpdateAccountInfo(ctx, assetType)
	return y.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (y *Yobit) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (y *Yobit) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (y *Yobit) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	info, err := y.GetInfo(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return info.ServerTime.Time(), nil
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (y *Yobit) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (y *Yobit) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (y *Yobit) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (y *Yobit) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := y.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.ForwardSlashDelimiter
	return tradeBaseURL + cp.Upper().String(), nil
}
