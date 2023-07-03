package zb

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (z *ZB) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	z.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = z.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = z.BaseCurrencies

	err := z.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if z.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = z.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (z *ZB) SetDefaults() {
	z.Name = "ZB"
	z.Enabled = true
	z.Verbose = true
	z.API.CredentialsValidator.RequiresKey = true
	z.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	err := z.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	z.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
				MultiChainDeposits:  true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				CancelOrder:            true,
				SubmitOrder:            true,
				MessageCorrelation:     true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					// NOTE: The supported time intervals below are returned
					// offset to the Asia/Shanghai time zone. This may lead to
					// issues with candle quality and conversion as the
					// intervals may be broken up. Therefore the below intervals
					// are constructed from hourly candles.
					// kline.IntervalCapacity{Interval: kline.TwoHour},
					// kline.IntervalCapacity{Interval: kline.FourHour},
					// kline.IntervalCapacity{Interval: kline.SixHour},
					// kline.IntervalCapacity{Interval: kline.TwelveHour},
					// kline.IntervalCapacity{Interval: kline.OneDay},
					// kline.IntervalCapacity{Interval: kline.ThreeDay},
					// kline.IntervalCapacity{Interval: kline.OneWeek},
				),
				GlobalResultLimit: 1000,
			},
		},
	}

	z.Requester, err = request.New(z.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	z.API.Endpoints = z.NewEndpoints()
	err = z.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              zbTradeURL,
		exchange.RestSpotSupplementary: zbMarketURL,
		exchange.WebsocketSpot:         zbWebsocketAPI,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	z.Websocket = stream.New()
	z.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	z.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup sets user configuration
func (z *ZB) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		z.SetEnabled(false)
		return nil
	}
	err = z.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := z.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = z.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             zbWebsocketAPI,
		RunningURL:             wsRunningURL,
		Connector:              z.WsConnect,
		GenerateSubscriptions:  z.GenerateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Subscriber:             z.Subscribe,
		Features:               &z.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return z.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  z.Websocket.GetWebsocketURL(),
		RateLimit:            zbWebsocketRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the ZB go routine
func (z *ZB) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		z.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the ZB wrapper
func (z *ZB) Run(ctx context.Context) {
	if z.Verbose {
		z.PrintEnabledPairs()
	}

	if !z.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := z.UpdateTradablePairs(ctx, false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", z.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (z *ZB) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	markets, err := z.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, len(markets))
	var target int
	for key := range markets {
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
func (z *ZB) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := z.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = z.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return z.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (z *ZB) UpdateTickers(ctx context.Context, a asset.Item) error {
	result, err := z.GetTickers(ctx)
	if err != nil {
		return err
	}

	enabledPairs, err := z.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for x := range enabledPairs {
		// We can't use either pair format here, so format it to lower-
		// case and without any delimiter
		curr := enabledPairs[x].Format(currency.EMPTYFORMAT).String()
		if _, ok := result[curr]; !ok {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         enabledPairs[x],
			High:         result[curr].High,
			Last:         result[curr].Last,
			Ask:          result[curr].Sell,
			Bid:          result[curr].Buy,
			Low:          result[curr].Low,
			Volume:       result[curr].Volume,
			ExchangeName: z.Name,
			AssetType:    a})
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (z *ZB) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := z.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(z.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (z *ZB) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(z.Name, p, assetType)
	if err != nil {
		return z.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (z *ZB) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(z.Name, p, assetType)
	if err != nil {
		return z.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (z *ZB) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := z.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        z.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: z.CanVerifyOrderbook,
	}
	currFormat, err := z.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := z.GetOrderbook(ctx, currFormat.String())
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x][1],
			Price:  orderbookNew.Bids[x][0],
		}
	}

	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: orderbookNew.Asks[x][1],
			Price:  orderbookNew.Asks[x][0],
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(z.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (z *ZB) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var coins []AccountsResponseCoin
	if z.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		resp, err := z.wsGetAccountInfoRequest(ctx)
		if err != nil {
			return info, err
		}
		coins = resp.Data.Coins
	} else {
		bal, err := z.GetAccountInformation(ctx)
		if err != nil {
			return info, err
		}
		coins = bal.Result.Coins
	}

	balances := make([]account.Balance, len(coins))
	for i := range coins {
		hold, err := strconv.ParseFloat(coins[i].Freeze, 64)
		if err != nil {
			return info, err
		}

		avail, err := strconv.ParseFloat(coins[i].Available, 64)
		if err != nil {
			return info, err
		}

		balances[i] = account.Balance{
			Currency: currency.NewCode(coins[i].EnName),
			Total:    hold + avail,
			Hold:     hold,
			Free:     avail,
		}
	}

	info.Exchange = z.Name
	info.Accounts = append(info.Accounts, account.SubAccount{
		AssetType:  assetType,
		Currencies: balances,
	})

	creds, err := z.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (z *ZB) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := z.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(z.Name, creds, assetType)
	if err != nil {
		return z.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (z *ZB) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	pairs, err := z.GetEnabledPairs(asset.Spot)
	if err != nil {
		return nil, err
	}
	currs := pairs.GetCurrencies()

	var records []exchange.FundingHistory
	for x := range currs {
		totalPages := int64(1)
		for y := int64(0); y < totalPages; y++ {
			var deposits *DepositRecordsResponse
			deposits, err = z.GetDepositRecords(ctx, &WalletRecordsRequest{
				Currency:  currs[x],
				PageIndex: y,
			})
			if err != nil {
				return nil, err
			}
			totalPages = deposits.Total
			for i := range deposits.List {
				var status string
				switch deposits.List[i].Status {
				case 0:
					status = "pending confirm"
				case 1:
					status = "failed"
				case 2:
					status = "confirmed"
				}
				records = append(records, exchange.FundingHistory{
					ExchangeName:    z.GetName(),
					Status:          status,
					TransferID:      strconv.FormatInt(deposits.List[i].ID, 10),
					Description:     deposits.List[i].Description,
					Timestamp:       time.Unix(deposits.List[i].SubmitTime, 0),
					Currency:        deposits.List[i].Currency,
					Amount:          deposits.List[i].Amount,
					CryptoToAddress: deposits.List[i].Address,
					TransferType:    "deposit",
				})
			}
		}
	}

	for x := range currs {
		var resp []exchange.WithdrawalHistory
		resp, err = z.GetWithdrawalsHistory(ctx, currs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		for y := range resp {
			records = append(records, exchange.FundingHistory{
				ExchangeName:    z.GetName(),
				Status:          resp[y].Status,
				TransferID:      resp[y].TransferID,
				Timestamp:       resp[y].Timestamp,
				Currency:        resp[y].Currency,
				Amount:          resp[y].Amount,
				CryptoToAddress: resp[y].CryptoToAddress,
				Fee:             resp[y].Fee,
				TransferType:    resp[y].TransferType,
			})
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.Before(records[j].Timestamp)
	})
	return records, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (z *ZB) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	totalPages := int64(1)
	var records []exchange.WithdrawalHistory
	for y := int64(0); y < totalPages; y++ {
		withdrawals, err := z.GetWithdrawalRecords(ctx, &WalletRecordsRequest{
			Currency:  c,
			PageIndex: y,
		})
		if err != nil {
			return nil, err
		}
		totalPages = withdrawals.Total
		for i := range withdrawals.List {
			var status string
			switch withdrawals.List[i].Status {
			case 0:
				status = "submitted"
			case 1:
				status = "failed"
			case 2:
				status = "successful"
			case 3:
				status = "cancelled"
			case 5:
				status = "confirmed"
			}
			records = append(records, exchange.WithdrawalHistory{
				Status:          status,
				TransferID:      strconv.FormatInt(withdrawals.List[i].ID, 10),
				Timestamp:       time.Unix(withdrawals.List[i].SubmitTime, 0),
				Currency:        c.String(),
				Amount:          withdrawals.List[i].Amount,
				CryptoToAddress: withdrawals.List[i].ToAddress,
				Fee:             withdrawals.List[i].Fees,
				TransferType:    "withdrawal",
			})
		}
	}
	return records, nil
}

// GetServerTime returns the current exchange server time.
func (z *ZB) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (z *ZB) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = z.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData TradeHistory
	tradeData, err = z.GetTrades(ctx, p.String())
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].Type)
		if err != nil {
			return nil, err
		}

		resp[i] = trade.Data{
			Exchange:     z.Name,
			TID:          strconv.FormatInt(tradeData[i].Tid, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    time.Unix(tradeData[i].Date, 0),
		}
	}

	err = z.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (z *ZB) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (z *ZB) SubmitOrder(ctx context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	err := o.Validate()
	if err != nil {
		return nil, err
	}

	if z.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var isBuyOrder int64
		if o.Side == order.Buy {
			isBuyOrder = 1
		} else {
			isBuyOrder = 0
		}
		var response *WsSubmitOrderResponse
		response, err = z.wsSubmitOrder(ctx, o.Pair, o.Amount, o.Price, isBuyOrder)
		if err != nil {
			return nil, err
		}
		return o.DeriveSubmitResponse(strconv.FormatInt(response.Data.EntrustID, 10))
	}
	var orderType = SpotNewOrderRequestParamsTypeSell
	if o.Side == order.Buy {
		orderType = SpotNewOrderRequestParamsTypeBuy
	}

	fPair, err := z.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return nil, err
	}

	var params = SpotNewOrderRequestParams{
		Amount: o.Amount,
		Price:  o.Price,
		Symbol: fPair.Lower().String(),
		Type:   orderType,
	}
	var response int64
	response, err = z.SpotNewOrder(ctx, params)
	if err != nil {
		return nil, err
	}
	return o.DeriveSubmitResponse(strconv.FormatInt(response, 10))
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (z *ZB) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (z *ZB) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}

	if z.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var response *WsCancelOrderResponse
		response, err = z.wsCancelOrder(ctx, o.Pair, orderIDInt)
		if err != nil {
			return err
		}
		if !response.Success {
			return fmt.Errorf("%w %v %v", request.ErrAuthRequestFailed, o.OrderID, response.Message)
		}
		return nil
	}
	fPair, err := z.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	return z.CancelExistingOrder(ctx, orderIDInt, fPair.String())
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (z *ZB) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (z *ZB) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	var allOpenOrders []Order
	enabledPairs, err := z.GetEnabledPairs(asset.Spot)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for x := range enabledPairs {
		fPair, err := z.FormatExchangeCurrency(enabledPairs[x], asset.Spot)
		if err != nil {
			return cancelAllOrdersResponse, err
		}
		for y := int64(1); ; y++ {
			openOrders, err := z.GetUnfinishedOrdersIgnoreTradeType(ctx,
				fPair.String(), y, 10)
			if err != nil {
				if strings.Contains(err.Error(), "3001") {
					break
				}
				return cancelAllOrdersResponse, err
			}

			if len(openOrders) == 0 {
				break
			}

			allOpenOrders = append(allOpenOrders, openOrders...)

			if len(openOrders) != 10 {
				break
			}
		}
	}

	for i := range allOpenOrders {
		p, err := currency.NewPairFromString(allOpenOrders[i].Currency)
		if err != nil {
			cancelAllOrdersResponse.Status[allOpenOrders[i].ID] = err.Error()
			continue
		}

		err = z.CancelOrder(ctx, &order.Cancel{
			OrderID:   allOpenOrders[i].ID,
			Pair:      p,
			AssetType: asset.Spot,
		})
		if err != nil {
			cancelAllOrdersResponse.Status[allOpenOrders[i].ID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (z *ZB) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := z.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	resp, err := z.GetSingleOrder(ctx, orderID, "", pair)
	if err != nil {
		return nil, err
	}
	side := order.Sell
	if resp.Type == 1 {
		side = order.Buy
	}
	var status order.Status
	switch resp.Status {
	case 1:
		status = order.Cancelled
	case 2:
		status = order.Closed
	case 3:
		status = order.Pending
	}
	return &order.Detail{
		Price:           resp.Price,
		Amount:          resp.TotalAmount,
		ExecutedAmount:  resp.TradeAmount,
		RemainingAmount: resp.TotalAmount - resp.TradeAmount,
		Fee:             resp.Fees,
		FeeAsset:        pair.Quote,
		Exchange:        z.Name,
		OrderID:         resp.ID,
		Side:            side,
		Status:          status,
		AssetType:       assetType,
		Date:            time.Unix(resp.TradeDate, 0),
		Pair:            pair,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (z *ZB) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	if chain != "" {
		addresses, err := z.GetMultiChainDepositAddress(ctx, cryptocurrency)
		if err != nil {
			return nil, err
		}
		for x := range addresses {
			if strings.EqualFold(addresses[x].Blockchain, chain) {
				return &deposit.Address{
					Address: addresses[x].Address,
					Tag:     addresses[x].Memo,
				}, nil
			}
		}
		return nil, fmt.Errorf("%w %s does not support chain %s", request.ErrAuthRequestFailed, cryptocurrency.String(), chain)
	}
	address, err := z.GetCryptoAddress(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	return &deposit.Address{
		Address: address.Message.Data.Address,
		Tag:     address.Message.Data.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (z *ZB) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := z.Withdraw(ctx,
		withdrawRequest.Currency.Lower().String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.TradePassword,
		withdrawRequest.Amount,
		withdrawRequest.Crypto.FeeAmount,
		false)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (z *ZB) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (z *ZB) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (z *ZB) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!z.AreCredentialsValid(ctx) || z.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return z.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (z *ZB) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var allOrders []Order
	for x := range req.Pairs {
		for i := int64(1); ; i++ {
			var fPair currency.Pair
			fPair, err = z.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
			if err != nil {
				return nil, err
			}
			var resp []Order
			resp, err = z.GetUnfinishedOrdersIgnoreTradeType(ctx,
				fPair.String(), i, 10)
			if err != nil {
				if strings.Contains(err.Error(), "3001") {
					break
				}
				return nil, err
			}

			if len(resp) == 0 {
				break
			}

			allOrders = append(allOrders, resp...)

			if len(resp) != 10 {
				break
			}
		}
	}

	format, err := z.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(allOrders))
	for i := range allOrders {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(allOrders[i].Currency,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(allOrders[i].TradeDate, 0)
		orderSide := orderSideMap[allOrders[i].Type]
		orders[i] = order.Detail{
			OrderID:  allOrders[i].ID,
			Amount:   allOrders[i].TotalAmount,
			Exchange: z.Name,
			Date:     orderDate,
			Price:    allOrders[i].Price,
			Side:     orderSide,
			Pair:     symbol,
		}
	}
	return req.Filter(z.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (z *ZB) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	sides := make([]int64, 0, 2)
	switch {
	case req.Side == order.AnySide:
		sides = append(sides, 0, 1)
	case req.Side.IsLong():
		sides = append(sides, 1)
	case req.Side.IsShort():
		sides = append(sides, 0)
	default:
		return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, req.Side)
	}
	var allOrders []Order
	if z.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for x := range req.Pairs {
			for y := int64(1); ; y++ {
				var resp *WsGetOrdersIgnoreTradeTypeResponse
				resp, err = z.wsGetOrdersIgnoreTradeType(ctx, req.Pairs[x], y, 10)
				if err != nil {
					return nil, err
				}
				allOrders = append(allOrders, resp.Data...)
				if len(resp.Data) != 10 {
					break
				}
			}
		}
	} else {
		for i := range sides {
			for x := range req.Pairs {
				for y := int64(1); ; y++ {
					var fPair currency.Pair
					fPair, err = z.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
					if err != nil {
						return nil, err
					}
					var resp []Order
					resp, err = z.GetOrders(ctx, fPair.String(), y, sides[i])
					if err != nil {
						return nil, err
					}
					if len(resp) == 0 {
						break
					}
					allOrders = append(allOrders, resp...)
					if len(resp) != 10 {
						break
					}
				}
			}
		}
	}

	format, err := z.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(allOrders))
	for i := range allOrders {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(allOrders[i].Currency,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(allOrders[i].TradeDate, 0)
		orderSide := orderSideMap[allOrders[i].Type]
		detail := order.Detail{
			OrderID:              allOrders[i].ID,
			Amount:               allOrders[i].TotalAmount,
			ExecutedAmount:       allOrders[i].TradeAmount,
			RemainingAmount:      allOrders[i].TotalAmount - allOrders[i].TradeAmount,
			Exchange:             z.Name,
			Date:                 orderDate,
			Price:                allOrders[i].Price,
			AverageExecutedPrice: allOrders[i].TradePrice,
			Side:                 orderSide,
			Pair:                 pair,
		}
		detail.InferCostsAndTimes()
		orders[i] = detail
	}
	return req.Filter(z.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (z *ZB) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := z.UpdateAccountInfo(ctx, assetType)
	return z.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (z *ZB) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneMin, kline.ThreeMin,
		kline.FiveMin, kline.FifteenMin, kline.ThirtyMin:
		return in.Short() + "in"
	case kline.OneHour, kline.TwoHour, kline.FourHour, kline.SixHour, kline.TwelveHour:
		return in.Short()[:len(in.Short())-1] + "hour"
	case kline.OneDay:
		return "1day"
	case kline.ThreeDay:
		return "3day"
	case kline.OneWeek:
		return "1week"
	}
	return ""
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (z *ZB) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := z.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}

	candles, err := z.GetSpotKline(ctx, KlinesRequestParams{
		Type:   z.FormatExchangeKlineInterval(req.ExchangeInterval),
		Symbol: req.RequestFormatted.String(),
		Since:  start.UnixMilli(),
		Size:   req.RequestLimit,
	})
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, len(candles.Data))
	for x := range candles.Data {
		if candles.Data[x].KlineTime.Before(start) || candles.Data[x].KlineTime.After(end) {
			continue
		}
		timeSeries = append(timeSeries, kline.Candle{
			Time:   candles.Data[x].KlineTime,
			Open:   candles.Data[x].Open,
			High:   candles.Data[x].High,
			Low:    candles.Data[x].Low,
			Close:  candles.Data[x].Close,
			Volume: candles.Data[x].Volume,
		})
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (z *ZB) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := z.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	count := kline.TotalCandlesPerInterval(req.Start, req.End, req.ExchangeInterval)
	if count > 1000 {
		return nil,
			fmt.Errorf("candles count: %d max lookback: %d, %w",
				count, 1000, kline.ErrRequestExceedsMaxLookback)
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for i := range req.RangeHolder.Ranges {
		var candles KLineResponse
		candles, err = z.GetSpotKline(ctx, KlinesRequestParams{
			Type:   z.FormatExchangeKlineInterval(req.ExchangeInterval),
			Symbol: req.RequestFormatted.String(),
			Since:  req.RangeHolder.Ranges[i].Start.Time.UnixMilli(),
			Size:   int64(req.RangeHolder.Limit),
		})
		if err != nil {
			return nil, err
		}

		for x := range candles.Data {
			if candles.Data[x].KlineTime.Before(req.Start) || candles.Data[x].KlineTime.After(req.End) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles.Data[x].KlineTime,
				Open:   candles.Data[x].Open,
				High:   candles.Data[x].High,
				Low:    candles.Data[x].Low,
				Close:  candles.Data[x].Close,
				Volume: candles.Data[x].Volume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (z *ZB) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	chains, err := z.GetMultiChainDepositAddress(ctx, cryptocurrency)
	if err != nil {
		// returned on valid currencies like BTC, despite having a deposit
		// address created it will advise the user to create one via their
		// app or website. In this case, we'll just return nil transfer
		// chains and no error message
		if strings.Contains(err.Error(), "APP") {
			return nil, nil
		}
		return nil, err
	}

	availableChains := make([]string, len(chains))
	for x := range chains {
		availableChains[x] = chains[x].Blockchain
	}
	return availableChains, nil
}
