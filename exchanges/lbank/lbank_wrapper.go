package lbank

import (
	"context"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (l *Lbank) GetDefaultConfig() (*config.Exchange, error) {
	l.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = l.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = l.BaseCurrencies

	err := l.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if l.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = l.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Lbank
func (l *Lbank) SetDefaults() {
	l.Name = "Lbank"
	l.Enabled = true
	l.Verbose = true
	l.API.CredentialsValidator.RequiresKey = true
	l.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	err := l.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	l.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST: true,
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
				SubmitOrder:         true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.EightHour.Word():  true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.OneWeek.Word():    true,
				},
				ResultLimit: 2000,
			},
		},
	}
	l.Requester, err = request.New(l.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	l.API.Endpoints = l.NewEndpoints()
	err = l.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: lbankAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup sets exchange configuration profile
func (l *Lbank) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		l.SetEnabled(false)
		return nil
	}
	err = l.SetupDefaults(exch)
	if err != nil {
		return err
	}

	if l.API.AuthenticatedSupport {
		err = l.loadPrivKey(context.TODO())
		if err != nil {
			l.API.AuthenticatedSupport = false
			log.Errorf(log.ExchangeSys, "%s couldn't load private key, setting authenticated support to false", l.Name)
		}
	}
	return nil
}

// Start starts the Lbank go routine
func (l *Lbank) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Lbank wrapper
func (l *Lbank) Run() {
	if l.Verbose {
		l.PrintEnabledPairs()
	}

	if !l.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := l.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", l.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (l *Lbank) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	currencies, err := l.GetCurrencyPairs(ctx)
	if err != nil {
		return nil, err
	}
	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (l *Lbank) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := l.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return l.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (l *Lbank) UpdateTickers(ctx context.Context, a asset.Item) error {
	tickerInfo, err := l.GetTickers(ctx)
	if err != nil {
		return err
	}
	pairs, err := l.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range pairs {
		for j := range tickerInfo {
			if !pairs[i].Equal(tickerInfo[j].Symbol) {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickerInfo[j].Ticker.Latest,
				High:         tickerInfo[j].Ticker.High,
				Low:          tickerInfo[j].Ticker.Low,
				Volume:       tickerInfo[j].Ticker.Volume,
				Pair:         tickerInfo[j].Symbol,
				LastUpdated:  time.Unix(0, tickerInfo[j].Timestamp),
				ExchangeName: l.Name,
				AssetType:    a})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *Lbank) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := l.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(l.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (l *Lbank) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fpair, err := l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(l.Name, fpair, assetType)
	if err != nil {
		return l.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (l *Lbank) FetchOrderbook(ctx context.Context, c currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(l.Name, c, assetType)
	if err != nil {
		return l.UpdateOrderbook(ctx, c, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *Lbank) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        l.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: l.CanVerifyOrderbook,
	}
	fpair, err := l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	a, err := l.GetMarketDepths(ctx, fpair.String(), "60", "1")
	if err != nil {
		return book, err
	}
	for i := range a.Data.Asks {
		price, convErr := strconv.ParseFloat(a.Data.Asks[i][0], 64)
		if convErr != nil {
			return book, convErr
		}
		amount, convErr := strconv.ParseFloat(a.Data.Asks[i][1], 64)
		if convErr != nil {
			return book, convErr
		}
		book.Asks = append(book.Asks, orderbook.Item{
			Price:  price,
			Amount: amount})
	}
	for i := range a.Data.Bids {
		price, convErr := strconv.ParseFloat(a.Data.Bids[i][0], 64)
		if convErr != nil {
			return book, convErr
		}
		amount, convErr := strconv.ParseFloat(a.Data.Bids[i][1], 64)
		if convErr != nil {
			return book, convErr
		}
		book.Bids = append(book.Bids, orderbook.Item{
			Price:  price,
			Amount: amount})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(l.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Lbank exchange
func (l *Lbank) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	data, err := l.GetUserInfo(ctx)
	if err != nil {
		return info, err
	}
	var acc account.SubAccount
	for key, val := range data.Info.Asset {
		c := currency.NewCode(key)
		hold, ok := data.Info.Freeze[key]
		if !ok {
			return info, fmt.Errorf("hold data not found with %s", key)
		}
		totalVal, parseErr := strconv.ParseFloat(val, 64)
		if parseErr != nil {
			return info, parseErr
		}
		totalHold, parseErr := strconv.ParseFloat(hold, 64)
		if parseErr != nil {
			return info, parseErr
		}
		acc.Currencies = append(acc.Currencies, account.Balance{
			CurrencyName: c,
			Total:        totalVal,
			Hold:         totalHold,
			Free:         totalVal - totalHold,
		})
	}

	info.Accounts = append(info.Accounts, acc)
	info.Exchange = l.Name

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (l *Lbank) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(l.Name, assetType)
	if err != nil {
		return l.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *Lbank) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (l *Lbank) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (l *Lbank) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return l.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (l *Lbank) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	ts := timestampStart
	limit := 600
allTrades:
	for {
		var tradeData []TradeResponse
		tradeData, err = l.GetTrades(ctx,
			p.String(),
			int64(limit),
			ts.UnixMilli())
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			tradeTime := time.UnixMilli(tradeData[i].DateMS)
			if tradeTime.Before(timestampStart) || tradeTime.After(timestampEnd) {
				break allTrades
			}
			side := order.Buy
			if strings.Contains(tradeData[i].Type, "sell") {
				side = order.Sell
			}
			resp = append(resp, trade.Data{
				Exchange:     l.Name,
				TID:          tradeData[i].TID,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Amount,
				Timestamp:    tradeTime,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tradeTime) {
					// reached end of trades to crawl
					break allTrades
				}
				ts = tradeTime
			}
		}
		if len(tradeData) != limit {
			break allTrades
		}
	}

	err = l.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (l *Lbank) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	if s.Side != order.Buy && s.Side != order.Sell {
		return resp,
			fmt.Errorf("%s order side is not supported by the exchange",
				s.Side)
	}

	fpair, err := l.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return resp, err
	}

	tempResp, err := l.CreateOrder(ctx,
		fpair.String(),
		s.Side.String(),
		s.Amount,
		s.Price)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = tempResp.OrderID
	if s.Type == order.Market {
		resp.FullyMatched = true
	}
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *Lbank) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *Lbank) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	fpair, err := l.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	_, err = l.RemoveOrder(ctx, fpair.String(), o.ID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (l *Lbank) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *Lbank) CancelAllOrders(ctx context.Context, o *order.Cancel) (order.CancelAllResponse, error) {
	if err := o.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var resp order.CancelAllResponse
	orderIDs, err := l.getAllOpenOrderID(ctx)
	if err != nil {
		return resp, err
	}

	for key := range orderIDs {
		if key != o.Pair.String() {
			continue
		}
		var x, y = 0, 0
		var input string
		var tempSlice []string
		for x <= len(orderIDs[key]) {
			x++
			for y != x {
				tempSlice = append(tempSlice, orderIDs[key][y])
				if y%3 == 0 {
					input = strings.Join(tempSlice, ",")
					CancelResponse, err2 := l.RemoveOrder(ctx, key, input)
					if err2 != nil {
						return resp, err2
					}
					tempStringSuccess := strings.Split(CancelResponse.Success, ",")
					for k := range tempStringSuccess {
						resp.Status[tempStringSuccess[k]] = "Cancelled"
					}
					tempStringError := strings.Split(CancelResponse.Err, ",")
					for l := range tempStringError {
						resp.Status[tempStringError[l]] = "Failed"
					}
					tempSlice = tempSlice[:0]
					y++
				}
				y++
			}
			input = strings.Join(tempSlice, ",")
			CancelResponse, err2 := l.RemoveOrder(ctx, key, input)
			if err2 != nil {
				return resp, err2
			}
			tempStringSuccess := strings.Split(CancelResponse.Success, ",")
			for k := range tempStringSuccess {
				resp.Status[tempStringSuccess[k]] = "Cancelled"
			}
			tempStringError := strings.Split(CancelResponse.Err, ",")
			for l := range tempStringError {
				resp.Status[tempStringError[l]] = "Failed"
			}
			tempSlice = tempSlice[:0]
		}
	}
	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (l *Lbank) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail
	orderIDs, err := l.getAllOpenOrderID(ctx)
	if err != nil {
		return resp, err
	}

	for key, val := range orderIDs {
		for i := range val {
			if val[i] != orderID {
				continue
			}
			tempResp, err := l.QueryOrder(ctx, key, orderID)
			if err != nil {
				return resp, err
			}
			resp.Exchange = l.Name
			resp.Pair, err = currency.NewPairFromString(key)
			if err != nil {
				return order.Detail{}, err
			}

			if strings.EqualFold(tempResp.Orders[0].Type, order.Buy.String()) {
				resp.Side = order.Buy
			} else {
				resp.Side = order.Sell
			}
			z := tempResp.Orders[0].Status
			switch {
			case z == -1:
				resp.Status = "cancelled"
			case z == 0:
				resp.Status = "on trading"
			case z == 1:
				resp.Status = "filled partially"
			case z == 2:
				resp.Status = "Filled totally"
			case z == 4:
				resp.Status = "Cancelling"
			default:
				resp.Status = "Invalid Order Status"
			}
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(ctx,
				&exchange.FeeBuilder{
					FeeType:       exchange.CryptocurrencyTradeFee,
					Amount:        tempResp.Orders[0].Amount,
					PurchasePrice: tempResp.Orders[0].Price})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
		}
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *Lbank) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := l.Withdraw(ctx,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Currency.String(),
		strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64),
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Description,
		"")
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.WithdrawID,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (l *Lbank) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}

	var finalResp []order.Detail
	var resp order.Detail
	tempData, err := l.getAllOpenOrderID(ctx)
	if err != nil {
		return finalResp, err
	}

	for key, val := range tempData {
		for x := range val {
			tempResp, err := l.QueryOrder(ctx, key, val[x])
			if err != nil {
				return finalResp, err
			}
			resp.Exchange = l.Name
			resp.Pair, err = currency.NewPairFromString(key)
			if err != nil {
				return nil, err
			}

			if strings.EqualFold(tempResp.Orders[0].Type, order.Buy.String()) {
				resp.Side = order.Buy
			} else {
				resp.Side = order.Sell
			}
			z := tempResp.Orders[0].Status
			switch {
			case z == -1:
				resp.Status = "cancelled"
			case z == 1:
				resp.Status = "on trading"
			case z == 2:
				resp.Status = "filled partially"
			case z == 3:
				resp.Status = "Filled totally"
			case z == 4:
				resp.Status = "Cancelling"
			default:
				resp.Status = "Invalid Order Status"
			}
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.Date = time.Unix(tempResp.Orders[0].CreateTime, 0)
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(ctx,
				&exchange.FeeBuilder{
					FeeType:       exchange.CryptocurrencyTradeFee,
					Amount:        tempResp.Orders[0].Amount,
					PurchasePrice: tempResp.Orders[0].Price})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
			for y := int(0); y < len(getOrdersRequest.Pairs); y++ {
				if getOrdersRequest.Pairs[y].String() != key {
					continue
				}
				if getOrdersRequest.Side == "ANY" {
					finalResp = append(finalResp, resp)
					continue
				}
				if strings.EqualFold(getOrdersRequest.Side.String(),
					tempResp.Orders[0].Type) {
					finalResp = append(finalResp, resp)
				}
			}
		}
	}
	return finalResp, nil
}

// GetOrderHistory retrieves account order information *
// Can Limit response to specific order status
func (l *Lbank) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}

	var finalResp []order.Detail
	var resp order.Detail
	var tempCurr currency.Pairs
	if len(getOrdersRequest.Pairs) == 0 {
		var err error
		tempCurr, err = l.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
	} else {
		tempCurr = getOrdersRequest.Pairs
	}
	for a := range tempCurr {
		fpair, err := l.FormatExchangeCurrency(tempCurr[a], asset.Spot)
		if err != nil {
			return nil, err
		}

		b := int64(1)
		tempResp, err := l.QueryOrderHistory(ctx,
			fpair.String(), strconv.FormatInt(b, 10), "200")
		if err != nil {
			return finalResp, err
		}
		for len(tempResp.Orders) != 0 {
			tempResp, err = l.QueryOrderHistory(ctx,
				fpair.String(), strconv.FormatInt(b, 10), "200")
			if err != nil {
				return finalResp, err
			}
			for x := 0; x < len(tempResp.Orders); x++ {
				resp.Exchange = l.Name
				resp.Pair, err = currency.NewPairFromString(tempResp.Orders[x].Symbol)
				if err != nil {
					return nil, err
				}

				if strings.EqualFold(tempResp.Orders[x].Type, order.Buy.String()) {
					resp.Side = order.Buy
				} else {
					resp.Side = order.Sell
				}
				z := tempResp.Orders[x].Status
				switch {
				case z == -1:
					resp.Status = "cancelled"
				case z == 1:
					resp.Status = "on trading"
				case z == 2:
					resp.Status = "filled partially"
				case z == 3:
					resp.Status = "Filled totally"
				case z == 4:
					resp.Status = "Cancelling"
				default:
					resp.Status = "Invalid Order Status"
				}
				resp.Price = tempResp.Orders[x].Price
				resp.AverageExecutedPrice = tempResp.Orders[x].AvgPrice
				resp.Amount = tempResp.Orders[x].Amount
				resp.Date = time.Unix(tempResp.Orders[x].CreateTime, 0)
				resp.ExecutedAmount = tempResp.Orders[x].DealAmount
				resp.RemainingAmount = tempResp.Orders[x].Amount - tempResp.Orders[x].DealAmount
				resp.Fee, err = l.GetFeeByType(ctx,
					&exchange.FeeBuilder{
						FeeType:       exchange.CryptocurrencyTradeFee,
						Amount:        tempResp.Orders[x].Amount,
						PurchasePrice: tempResp.Orders[x].Price})
				if err != nil {
					resp.Fee = lbankFeeNotFound
				}
				resp.InferCostsAndTimes()
				finalResp = append(finalResp, resp)
				b++
			}
		}
	}
	return finalResp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction *
func (l *Lbank) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	var resp float64
	if feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.002, nil
	}
	if feeBuilder.FeeType == exchange.CryptocurrencyWithdrawalFee {
		withdrawalFee, err := l.GetWithdrawConfig(ctx,
			feeBuilder.Pair.Base.Lower().String())
		if err != nil {
			return resp, err
		}
		for i := range withdrawalFee {
			if !strings.EqualFold(withdrawalFee[i].AssetCode, feeBuilder.Pair.Base.String()) {
				continue
			}
			if withdrawalFee[i].Fee == "" {
				return 0, nil
			}
			resp, err = strconv.ParseFloat(withdrawalFee[i].Fee, 64)
			if err != nil {
				return resp, err
			}
		}
	}
	return resp, nil
}

// GetAllOpenOrderID returns all open orders by currency pairs
func (l *Lbank) getAllOpenOrderID(ctx context.Context) (map[string][]string, error) {
	allPairs, err := l.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]string)
	for a := range allPairs {
		fpair, err := l.FormatExchangeCurrency(allPairs[a], asset.Spot)
		if err != nil {
			return nil, err
		}
		b := int64(1)
		tempResp, err := l.GetOpenOrders(ctx,
			fpair.String(),
			strconv.FormatInt(b, 10),
			"200")
		if err != nil {
			return resp, err
		}
		tempData := len(tempResp.Orders)
		for tempData != 0 {
			tempResp, err = l.GetOpenOrders(ctx,
				fpair.String(),
				strconv.FormatInt(b, 10),
				"200")
			if err != nil {
				return resp, err
			}

			if len(tempResp.Orders) == 0 {
				return resp, nil
			}

			for c := 0; c < tempData; c++ {
				resp[fpair.String()] = append(resp[fpair.String()],
					tempResp.Orders[c].OrderID)
			}
			tempData = len(tempResp.Orders)
			b++
		}
	}
	return resp, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (l *Lbank) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := l.UpdateAccountInfo(ctx, assetType)
	return l.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (l *Lbank) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneMin, kline.ThreeMin,
		kline.FiveMin, kline.FifteenMin, kline.ThirtyMin:
		return "minute" + in.Short()[:len(in.Short())-1]
	case kline.OneHour, kline.FourHour,
		kline.EightHour, kline.TwelveHour:
		return "hour" + in.Short()[:len(in.Short())-1]
	case kline.OneDay:
		return "day1"
	case kline.OneWeek:
		return "week1"
	}
	return ""
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (l *Lbank) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := l.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := l.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	data, err := l.GetKlines(ctx,
		formattedPair.String(),
		strconv.FormatInt(int64(l.Features.Enabled.Kline.ResultLimit), 10),
		l.FormatExchangeKlineInterval(interval),
		strconv.FormatInt(start.Unix(), 10))
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: l.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	for x := range data {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   data[x].TimeStamp,
			Open:   data[x].OpenPrice,
			High:   data[x].HigestPrice,
			Low:    data[x].LowestPrice,
			Close:  data[x].ClosePrice,
			Volume: data[x].TradingVolume,
		})
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (l *Lbank) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := l.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: l.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	dates, err := kline.CalculateCandleDateRanges(start, end, interval, l.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	formattedPair, err := l.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range dates.Ranges {
		var data []KlineResponse
		data, err = l.GetKlines(ctx,
			formattedPair.String(),
			strconv.FormatInt(int64(l.Features.Enabled.Kline.ResultLimit), 10),
			l.FormatExchangeKlineInterval(interval),
			strconv.FormatInt(dates.Ranges[x].Start.Ticks, 10))
		if err != nil {
			return kline.Item{}, err
		}
		for i := range data {
			if data[i].TimeStamp.Unix() < dates.Ranges[x].Start.Ticks || data[i].TimeStamp.Unix() > dates.Ranges[x].End.Ticks {
				continue
			}
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   data[i].TimeStamp,
				Open:   data[i].OpenPrice,
				High:   data[i].HigestPrice,
				Low:    data[i].LowestPrice,
				Close:  data[i].ClosePrice,
				Volume: data[i].TradingVolume,
			})
		}
	}

	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", l.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
