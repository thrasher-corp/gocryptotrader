package lbank

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

// SetDefaults sets the basic defaults for Lbank
func (e *Exchange) SetDefaults() {
	e.Name = "Lbank"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
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
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					// NOTE: The supported time intervals below are returned
					// offset to the Asia/HongKong time zone. This may lead to
					// issues with candle quality and conversion as the
					// intervals may be broken up. The below intervals
					// are constructed from hourly -> 4 hourly candles.
					// kline.IntervalCapacity{Interval: kline.EightHour}, // The docs suggest this is supported, but it isn't.
					// kline.IntervalCapacity{Interval: kline.TwelveHour}, // The docs suggest this is supported, but it isn't.
					// kline.IntervalCapacity{Interval: kline.OneDay},
					// kline.IntervalCapacity{Interval: kline.OneWeek},
					// kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 2000,
			},
		},
	}
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: lbankAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup sets exchange configuration profile
func (e *Exchange) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	err = e.SetupDefaults(exch)
	if err != nil {
		return err
	}

	if e.API.AuthenticatedSupport {
		err = e.loadPrivKey(context.TODO())
		if err != nil {
			e.API.AuthenticatedSupport = false
			log.Errorf(log.ExchangeSys, "%s couldn't load private key, setting authenticated support to false", e.Name)
		}
	}
	return nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	currencies, err := e.GetCurrencyPairs(ctx)
	if err != nil {
		return nil, err
	}
	return currency.NewPairsFromStrings(currencies)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	if err := e.UpdatePairs(pairs, asset.Spot, false); err != nil {
		return err
	}
	return e.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	tickerInfo, err := e.GetTickers(ctx)
	if err != nil {
		return err
	}
	pairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range pairs {
		for j := range tickerInfo {
			if !pairs[i].Equal(tickerInfo[j].Symbol) {
				continue
			}

			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         tickerInfo[j].Ticker.Latest,
				High:         tickerInfo[j].Ticker.High,
				Low:          tickerInfo[j].Ticker.Low,
				Volume:       tickerInfo[j].Ticker.Volume,
				Pair:         tickerInfo[j].Symbol,
				LastUpdated:  tickerInfo[j].Timestamp.Time(),
				ExchangeName: e.Name,
				AssetType:    a,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if !e.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}

	fPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	d, err := e.GetMarketDepths(ctx, fPair.String(), 60)
	if err != nil {
		return nil, err
	}

	ob := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
		Asks:              d.Data.Asks.Levels(),
		Bids:              d.Data.Bids.Levels(),
	}
	if err := ob.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	resp, err := e.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for k, val := range resp.Info.Asset {
		hold, ok := resp.Info.Freeze[k]
		if !ok {
			return nil, fmt.Errorf("hold data not found with %s", k)
		}
		totalVal := val.Float64()
		totalHold := hold.Float64()
		subAccts[0].Balances.Set(currency.NewCode(k), accounts.Balance{
			Total: totalVal,
			Hold:  totalHold,
			Free:  totalVal - totalHold,
		})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	withdrawalRecords, err := e.GetWithdrawalRecords(ctx, c.String(), 1, 0, 100)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawalRecords.List))
	for i := range withdrawalRecords.List {
		id := strconv.FormatInt(withdrawalRecords.List[i].ID, 10)
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawalRecords.List[i].Status,
			TransferID:      id,
			Timestamp:       withdrawalRecords.List[i].Time.Time(),
			Currency:        withdrawalRecords.List[i].AssetCode,
			Amount:          withdrawalRecords.List[i].Amount,
			Fee:             withdrawalRecords.List[i].Fee,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalRecords.List[i].Address,
			CryptoTxID:      withdrawalRecords.List[i].TXHash,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return e.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	ts := timestampStart
	const limit uint64 = 600
allTrades:
	for {
		var tradeData []TradeResponse
		tradeData, err = e.GetTrades(ctx, p.String(), limit, ts)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			tradeTime := tradeData[i].DateMS.Time()
			if tradeTime.Before(timestampStart) || tradeTime.After(timestampEnd) {
				break allTrades
			}
			side := order.Buy
			if strings.Contains(tradeData[i].Type, "sell") {
				side = order.Sell
			}
			resp = append(resp, trade.Data{
				Exchange:     e.Name,
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
		if len(tradeData) != int(limit) {
			break allTrades
		}
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	if !s.Side.IsLong() && !s.Side.IsShort() {
		return nil,
			fmt.Errorf("%s order side is not supported by the exchange",
				s.Side)
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	tempResp, err := e.CreateOrder(ctx,
		fPair.String(),
		s.Side.String(),
		s.Amount,
		s.Price)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(tempResp.OrderID)
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	fPair, err := e.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	_, err = e.RemoveOrder(ctx, fPair.String(), o.OrderID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.GetTimestamp(ctx)
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, o *order.Cancel) (order.CancelAllResponse, error) {
	if err := o.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var resp order.CancelAllResponse
	orderIDs, err := e.getAllOpenOrderID(ctx)
	if err != nil {
		return resp, err
	}

	for key := range orderIDs {
		if key != o.Pair.String() {
			continue
		}
		x, y := 0, 0
		var input string
		var tempSlice []string
		for x <= len(orderIDs[key]) {
			x++
			for y != x {
				tempSlice = append(tempSlice, orderIDs[key][y])
				if y%3 == 0 {
					input = strings.Join(tempSlice, ",")
					CancelResponse, err2 := e.RemoveOrder(ctx, key, input)
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
			CancelResponse, err2 := e.RemoveOrder(ctx, key, input)
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
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	var resp order.Detail
	orderIDs, err := e.getAllOpenOrderID(ctx)
	if err != nil {
		return nil, err
	}

	for key, val := range orderIDs {
		for i := range val {
			if val[i] != orderID {
				continue
			}
			tempResp, err := e.QueryOrder(ctx, key, orderID)
			if err != nil {
				return nil, err
			}
			resp.Exchange = e.Name
			resp.Pair, err = currency.NewPairFromString(key)
			if err != nil {
				return nil, err
			}

			if strings.EqualFold(tempResp.Orders[0].Type, order.Buy.String()) {
				resp.Side = order.Buy
			} else {
				resp.Side = order.Sell
			}

			resp.Status = e.GetStatus(tempResp.Orders[0].Status)
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = e.GetFeeByType(ctx, &exchange.FeeBuilder{
				FeeType:       exchange.CryptocurrencyTradeFee,
				Amount:        tempResp.Orders[0].Amount,
				PurchasePrice: tempResp.Orders[0].Price,
			})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
		}
	}
	return &resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.Withdraw(ctx,
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
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}

	var finalResp []order.Detail
	var resp order.Detail
	tempData, err := e.getAllOpenOrderID(ctx)
	if err != nil {
		return finalResp, err
	}

	for key, val := range tempData {
		for x := range val {
			tempResp, err := e.QueryOrder(ctx, key, val[x])
			if err != nil {
				return finalResp, err
			}
			resp.Exchange = e.Name
			resp.Pair, err = currency.NewPairFromString(key)
			if err != nil {
				return nil, err
			}

			if strings.EqualFold(tempResp.Orders[0].Type, order.Buy.String()) {
				resp.Side = order.Buy
			} else {
				resp.Side = order.Sell
			}
			resp.Status = e.GetStatus(tempResp.Orders[0].Status)
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.Date = tempResp.Orders[0].CreateTime.Time()
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = e.GetFeeByType(ctx,
				&exchange.FeeBuilder{
					FeeType:       exchange.CryptocurrencyTradeFee,
					Amount:        tempResp.Orders[0].Amount,
					PurchasePrice: tempResp.Orders[0].Price,
				})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
			for y := range getOrdersRequest.Pairs {
				if getOrdersRequest.Pairs[y].String() != key {
					continue
				}
				if getOrdersRequest.Side == order.AnySide {
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
	return getOrdersRequest.Filter(e.Name, finalResp), nil
}

// GetOrderHistory retrieves account order information *
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}

	var finalResp []order.Detail
	var resp order.Detail
	var tempCurr currency.Pairs
	if len(getOrdersRequest.Pairs) == 0 {
		var err error
		tempCurr, err = e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
	} else {
		tempCurr = getOrdersRequest.Pairs
	}
	for a := range tempCurr {
		fPair, err := e.FormatExchangeCurrency(tempCurr[a], asset.Spot)
		if err != nil {
			return nil, err
		}

		b := int64(1)
		tempResp, err := e.QueryOrderHistory(ctx,
			fPair.String(), strconv.FormatInt(b, 10), "200")
		if err != nil {
			return finalResp, err
		}
		for len(tempResp.Orders) != 0 {
			tempResp, err = e.QueryOrderHistory(ctx,
				fPair.String(), strconv.FormatInt(b, 10), "200")
			if err != nil {
				return finalResp, err
			}
			for x := range tempResp.Orders {
				resp.Exchange = e.Name
				resp.Pair, err = currency.NewPairFromString(tempResp.Orders[x].Symbol)
				if err != nil {
					return nil, err
				}

				if strings.EqualFold(tempResp.Orders[x].Type, order.Buy.String()) {
					resp.Side = order.Buy
				} else {
					resp.Side = order.Sell
				}
				resp.Status = e.GetStatus(tempResp.Orders[x].Status)
				resp.Price = tempResp.Orders[x].Price
				resp.AverageExecutedPrice = tempResp.Orders[x].AvgPrice
				resp.Amount = tempResp.Orders[x].Amount
				resp.Date = tempResp.Orders[x].CreateTime.Time()
				resp.ExecutedAmount = tempResp.Orders[x].DealAmount
				resp.RemainingAmount = tempResp.Orders[x].Amount - tempResp.Orders[x].DealAmount
				resp.Fee, err = e.GetFeeByType(ctx,
					&exchange.FeeBuilder{
						FeeType:       exchange.CryptocurrencyTradeFee,
						Amount:        tempResp.Orders[x].Amount,
						PurchasePrice: tempResp.Orders[x].Price,
					})
				if err != nil {
					resp.Fee = lbankFeeNotFound
				}
				resp.InferCostsAndTimes()
				finalResp = append(finalResp, resp)
				b++
			}
		}
	}
	return getOrdersRequest.Filter(e.Name, finalResp), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction *
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	var resp float64
	if feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.002, nil
	}
	if feeBuilder.FeeType == exchange.CryptocurrencyWithdrawalFee {
		withdrawalFee, err := e.GetWithdrawConfig(ctx, feeBuilder.Pair.Base)
		if err != nil {
			return resp, err
		}
		for i := range withdrawalFee {
			if !withdrawalFee[i].AssetCode.Equal(feeBuilder.Pair.Base) {
				continue
			}
			resp = withdrawalFee[i].Fee
			break
		}
	}
	return resp, nil
}

// getAllOpenOrderID returns all open orders by currency pairs
func (e *Exchange) getAllOpenOrderID(ctx context.Context) (map[string][]string, error) {
	allPairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]string)
	for a := range allPairs {
		fPair, err := e.FormatExchangeCurrency(allPairs[a], asset.Spot)
		if err != nil {
			return nil, err
		}
		b := int64(1)
		tempResp, err := e.GetOpenOrders(ctx,
			fPair.String(),
			strconv.FormatInt(b, 10),
			"200")
		if err != nil {
			return resp, err
		}
		tempData := len(tempResp.Orders)
		for tempData != 0 {
			tempResp, err = e.GetOpenOrders(ctx,
				fPair.String(),
				strconv.FormatInt(b, 10),
				"200")
			if err != nil {
				return resp, err
			}

			if len(tempResp.Orders) == 0 {
				return resp, nil
			}

			for c := range tempData {
				resp[fPair.String()] = append(resp[fPair.String()], tempResp.Orders[c].OrderID)
			}
			tempData = len(tempResp.Orders)
			b++
		}
	}
	return resp, nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (e *Exchange) FormatExchangeKlineInterval(in kline.Interval) string {
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
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}

	data, err := e.GetKlines(ctx,
		req.RequestFormatted.String(),
		strconv.FormatUint(req.RequestLimit, 10),
		e.FormatExchangeKlineInterval(req.ExchangeInterval),
		req.Start)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(data))
	for x := range data {
		timeSeries[x] = kline.Candle{
			Time:   data[x].TimeStamp,
			Open:   data[x].OpenPrice,
			High:   data[x].HighestPrice,
			Low:    data[x].LowestPrice,
			Close:  data[x].ClosePrice,
			Volume: data[x].TradingVolume,
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var data []KlineResponse
		data, err = e.GetKlines(ctx,
			req.RequestFormatted.String(),
			strconv.FormatUint(req.RequestLimit, 10),
			e.FormatExchangeKlineInterval(req.ExchangeInterval),
			req.RangeHolder.Ranges[x].Start.Time)
		if err != nil {
			return nil, err
		}
		for i := range data {
			if (data[i].TimeStamp.Unix() < req.RangeHolder.Ranges[x].Start.Ticks) ||
				(data[i].TimeStamp.Unix() > req.RangeHolder.Ranges[x].End.Ticks) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   data[i].TimeStamp,
				Open:   data[i].OpenPrice,
				High:   data[i].HighestPrice,
				Low:    data[i].LowestPrice,
				Close:  data[i].ClosePrice,
				Volume: data[i].TradingVolume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetStatus returns the order.Status from the int representation.
func (e *Exchange) GetStatus(status int64) order.Status {
	var oStatus order.Status
	switch status {
	case -1:
		// "cancelled"
		oStatus = order.Cancelled
	case 0:
		// "on trading"
		oStatus = order.Active
	case 1:
		// "filled partially"
		oStatus = order.PartiallyFilled
	case 2:
		// "filled totally"
		oStatus = order.Filled
	case 4:
		// "Cancelling"
		oStatus = order.Cancelling
	default:
		log.Errorf(log.Global, "%s Unhandled Order Status '%v'", e.GetName(), status)
	}
	return oStatus
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.UnderscoreDelimiter
	return tradeBaseURL + cp.Lower().String(), nil
}
