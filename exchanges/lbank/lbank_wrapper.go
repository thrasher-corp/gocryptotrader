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

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (l *Lbank) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	currencies, err := l.GetCurrencyPairs(ctx)
	if err != nil {
		return nil, err
	}
	return currency.NewPairsFromStrings(currencies)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (l *Lbank) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := l.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = l.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return l.EnsureOnePairEnabled()
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

			if err := ticker.ProcessTicker(&ticker.Price{
				Last:         tickerInfo[j].Ticker.Latest,
				High:         tickerInfo[j].Ticker.High,
				Low:          tickerInfo[j].Ticker.Low,
				Volume:       tickerInfo[j].Ticker.Volume,
				Pair:         tickerInfo[j].Symbol,
				LastUpdated:  tickerInfo[j].Timestamp.Time(),
				ExchangeName: l.Name,
				AssetType:    a,
			}); err != nil {
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

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *Lbank) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if !l.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}

	fPair, err := l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	d, err := l.GetMarketDepths(ctx, fPair.String(), 60)
	if err != nil {
		return nil, err
	}

	book := &orderbook.Book{
		Exchange:          l.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: l.ValidateOrderbook,
		Asks:              make(orderbook.Levels, len(d.Data.Asks)),
		Bids:              make(orderbook.Levels, len(d.Data.Bids)),
	}

	for i := range d.Data.Asks {
		book.Asks[i].Price = d.Data.Asks[i][0].Float64()
		book.Asks[i].Amount = d.Data.Asks[i][1].Float64()
	}
	for i := range d.Data.Bids {
		book.Bids[i].Price = d.Data.Bids[i][0].Float64()
		book.Bids[i].Amount = d.Data.Bids[i][1].Float64()
	}

	if err := book.Process(); err != nil {
		return nil, err
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
	acc := account.SubAccount{AssetType: assetType}
	for key, val := range data.Info.Asset {
		hold, ok := data.Info.Freeze[key]
		if !ok {
			return info, fmt.Errorf("hold data not found with %s", key)
		}
		totalVal := val.Float64()
		totalHold := hold.Float64()
		acc.Currencies = append(acc.Currencies, account.Balance{
			Currency: currency.NewCode(key),
			Total:    totalVal,
			Hold:     totalHold,
			Free:     totalVal - totalHold,
		})
	}

	info.Accounts = append(info.Accounts, acc)
	info.Exchange = l.Name

	creds, err := l.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&info, creds)
	if err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (l *Lbank) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (l *Lbank) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if err := l.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	withdrawalRecords, err := l.GetWithdrawalRecords(ctx, c.String(), 1, 0, 100)
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
	const limit uint64 = 600
allTrades:
	for {
		var tradeData []TradeResponse
		tradeData, err = l.GetTrades(ctx, p.String(), limit, ts)
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
		if len(tradeData) != int(limit) {
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
func (l *Lbank) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(l.GetTradingRequirements()); err != nil {
		return nil, err
	}

	if !s.Side.IsLong() && !s.Side.IsShort() {
		return nil,
			fmt.Errorf("%s order side is not supported by the exchange",
				s.Side)
	}

	fPair, err := l.FormatExchangeCurrency(s.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}

	tempResp, err := l.CreateOrder(ctx,
		fPair.String(),
		s.Side.String(),
		s.Amount,
		s.Price)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(tempResp.OrderID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *Lbank) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *Lbank) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	fPair, err := l.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	_, err = l.RemoveOrder(ctx, fPair.String(), o.OrderID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (l *Lbank) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (l *Lbank) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return l.GetTimestamp(ctx)
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
		x, y := 0, 0
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
func (l *Lbank) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	var resp order.Detail
	orderIDs, err := l.getAllOpenOrderID(ctx)
	if err != nil {
		return nil, err
	}

	for key, val := range orderIDs {
		for i := range val {
			if val[i] != orderID {
				continue
			}
			tempResp, err := l.QueryOrder(ctx, key, orderID)
			if err != nil {
				return nil, err
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

			resp.Status = l.GetStatus(tempResp.Orders[0].Status)
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(ctx, &exchange.FeeBuilder{
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
func (l *Lbank) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
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
			resp.Status = l.GetStatus(tempResp.Orders[0].Status)
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.Date = tempResp.Orders[0].CreateTime.Time()
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(ctx,
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
	return getOrdersRequest.Filter(l.Name, finalResp), nil
}

// GetOrderHistory retrieves account order information *
// Can Limit response to specific order status
func (l *Lbank) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
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
		fPair, err := l.FormatExchangeCurrency(tempCurr[a], asset.Spot)
		if err != nil {
			return nil, err
		}

		b := int64(1)
		tempResp, err := l.QueryOrderHistory(ctx,
			fPair.String(), strconv.FormatInt(b, 10), "200")
		if err != nil {
			return finalResp, err
		}
		for len(tempResp.Orders) != 0 {
			tempResp, err = l.QueryOrderHistory(ctx,
				fPair.String(), strconv.FormatInt(b, 10), "200")
			if err != nil {
				return finalResp, err
			}
			for x := range tempResp.Orders {
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
				resp.Status = l.GetStatus(tempResp.Orders[x].Status)
				resp.Price = tempResp.Orders[x].Price
				resp.AverageExecutedPrice = tempResp.Orders[x].AvgPrice
				resp.Amount = tempResp.Orders[x].Amount
				resp.Date = tempResp.Orders[x].CreateTime.Time()
				resp.ExecutedAmount = tempResp.Orders[x].DealAmount
				resp.RemainingAmount = tempResp.Orders[x].Amount - tempResp.Orders[x].DealAmount
				resp.Fee, err = l.GetFeeByType(ctx,
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
	return getOrdersRequest.Filter(l.Name, finalResp), nil
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
		withdrawalFee, err := l.GetWithdrawConfig(ctx, feeBuilder.Pair.Base)
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

// GetAllOpenOrderID returns all open orders by currency pairs
func (l *Lbank) getAllOpenOrderID(ctx context.Context) (map[string][]string, error) {
	allPairs, err := l.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	resp := make(map[string][]string)
	for a := range allPairs {
		fPair, err := l.FormatExchangeCurrency(allPairs[a], asset.Spot)
		if err != nil {
			return nil, err
		}
		b := int64(1)
		tempResp, err := l.GetOpenOrders(ctx,
			fPair.String(),
			strconv.FormatInt(b, 10),
			"200")
		if err != nil {
			return resp, err
		}
		tempData := len(tempResp.Orders)
		for tempData != 0 {
			tempResp, err = l.GetOpenOrders(ctx,
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

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (l *Lbank) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
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
func (l *Lbank) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := l.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}

	data, err := l.GetKlines(ctx,
		req.RequestFormatted.String(),
		strconv.FormatUint(req.RequestLimit, 10),
		l.FormatExchangeKlineInterval(req.ExchangeInterval),
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
func (l *Lbank) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := l.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var data []KlineResponse
		data, err = l.GetKlines(ctx,
			req.RequestFormatted.String(),
			strconv.FormatUint(req.RequestLimit, 10),
			l.FormatExchangeKlineInterval(req.ExchangeInterval),
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
func (l *Lbank) GetStatus(status int64) order.Status {
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
		log.Errorf(log.Global, "%s Unhandled Order Status '%v'", l.GetName(), status)
	}
	return oStatus
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (l *Lbank) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (l *Lbank) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (l *Lbank) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (l *Lbank) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := l.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.UnderscoreDelimiter
	return tradeBaseURL + cp.Lower().String(), nil
}
