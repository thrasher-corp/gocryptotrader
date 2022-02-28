package zb

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (z *ZB) GetDefaultConfig() (*config.Exchange, error) {
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
		err = z.UpdateTradablePairs(context.TODO(), true)
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
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.ThreeMin.Word():   true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.SixHour.Word():    true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.ThreeDay.Word():   true,
					kline.OneWeek.Word():    true,
				},
				ResultLimit: 1000,
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
		ExchangeConfig:        exch,
		DefaultURL:            zbWebsocketAPI,
		RunningURL:            wsRunningURL,
		Connector:             z.WsConnect,
		GenerateSubscriptions: z.GenerateDefaultSubscriptions,
		Subscriber:            z.Subscribe,
		Features:              &z.Features.Supports.WebsocketCapabilities,
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

// Start starts the OKEX go routine
func (z *ZB) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		z.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the OKEX wrapper
func (z *ZB) Run() {
	if z.Verbose {
		z.PrintEnabledPairs()
	}

	if !z.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := z.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", z.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (z *ZB) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	markets, err := z.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range markets {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (z *ZB) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := z.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return z.UpdatePairs(p, asset.Spot, false, forceUpdate)
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
		curr := enabledPairs[x].Format("", false).String()
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

	for x := range orderbookNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x][1],
			Price:  orderbookNew.Bids[x][0],
		})
	}

	for x := range orderbookNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x][1],
			Price:  orderbookNew.Asks[x][0],
		})
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
	var balances []account.Balance
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

	for i := range coins {
		hold, err := strconv.ParseFloat(coins[i].Freeze, 64)
		if err != nil {
			return info, err
		}

		avail, err := strconv.ParseFloat(coins[i].Available, 64)
		if err != nil {
			return info, err
		}

		balances = append(balances, account.Balance{
			CurrencyName: currency.NewCode(coins[i].EnName),
			Total:        hold + avail,
			Hold:         hold,
			Free:         avail,
		})
	}

	info.Exchange = z.Name
	info.Accounts = append(info.Accounts, account.SubAccount{
		Currencies: balances,
	})

	if err := account.Process(&info); err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (z *ZB) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(z.Name, assetType)
	if err != nil {
		return z.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (z *ZB) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (z *ZB) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
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
	var resp []trade.Data
	for i := range tradeData {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].Type)
		if err != nil {
			return nil, err
		}

		resp = append(resp, trade.Data{
			Exchange:     z.Name,
			TID:          strconv.FormatInt(tradeData[i].Tid, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    time.Unix(tradeData[i].Date, 0),
		})
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
func (z *ZB) SubmitOrder(ctx context.Context, o *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	err := o.Validate()
	if err != nil {
		return submitOrderResponse, err
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
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = strconv.FormatInt(response.Data.EntrustID, 10)
	} else {
		var oT SpotNewOrderRequestParamsType
		if o.Side == order.Buy {
			oT = SpotNewOrderRequestParamsTypeBuy
		} else {
			oT = SpotNewOrderRequestParamsTypeSell
		}

		fPair, err := z.FormatExchangeCurrency(o.Pair, o.AssetType)
		if err != nil {
			return submitOrderResponse, err
		}

		var params = SpotNewOrderRequestParams{
			Amount: o.Amount,
			Price:  o.Price,
			Symbol: fPair.Lower().String(),
			Type:   oT,
		}
		var response int64
		response, err = z.SpotNewOrder(ctx, params)
		if err != nil {
			return submitOrderResponse, err
		}
		if response > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response, 10)
		}
	}
	submitOrderResponse.IsOrderPlaced = true
	if o.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (z *ZB) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (z *ZB) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
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
			return fmt.Errorf("%v - Could not cancel order %v", z.Name, o.ID)
		}
		return nil
	}
	fpair, err := z.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}
	return z.CancelExistingOrder(ctx, orderIDInt, fpair.String())
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (z *ZB) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
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
			cancelAllOrdersResponse.Status[strconv.FormatInt(allOpenOrders[i].ID, 10)] = err.Error()
			continue
		}

		err = z.CancelOrder(ctx,
			&order.Cancel{
				ID:        strconv.FormatInt(allOpenOrders[i].ID, 10),
				Pair:      p,
				AssetType: asset.Spot,
			})
		if err != nil {
			cancelAllOrdersResponse.Status[strconv.FormatInt(allOpenOrders[i].ID, 10)] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (z *ZB) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
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
		return nil, fmt.Errorf("%s does not support chain %s", cryptocurrency.String(), chain)
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
func (z *ZB) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var allOrders []Order
	for x := range req.Pairs {
		for i := int64(1); ; i++ {
			fPair, err := z.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
			if err != nil {
				return nil, err
			}
			resp, err := z.GetUnfinishedOrdersIgnoreTradeType(ctx,
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

	var orders []order.Detail
	for i := range allOrders {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(allOrders[i].Currency,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(int64(allOrders[i].TradeDate), 0)
		orderSide := orderSideMap[allOrders[i].Type]
		orders = append(orders, order.Detail{
			ID:       strconv.FormatInt(allOrders[i].ID, 10),
			Amount:   allOrders[i].TotalAmount,
			Exchange: z.Name,
			Date:     orderDate,
			Price:    allOrders[i].Price,
			Side:     orderSide,
			Pair:     symbol,
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (z *ZB) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if req.Side == order.AnySide || req.Side == "" {
		return nil, errors.New("specific order side is required")
	}
	var allOrders []Order
	var orders []order.Detail
	var side int64

	if z.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for x := range req.Pairs {
			for y := int64(1); ; y++ {
				resp, err := z.wsGetOrdersIgnoreTradeType(ctx, req.Pairs[x], y, 10)
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
		if req.Side == order.Buy {
			side = 1
		}
		for x := range req.Pairs {
			for y := int64(1); ; y++ {
				fPair, err := z.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
				if err != nil {
					return nil, err
				}
				resp, err := z.GetOrders(ctx, fPair.String(), y, side)
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

	format, err := z.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	for i := range allOrders {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(allOrders[i].Currency,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		orderDate := time.Unix(int64(allOrders[i].TradeDate), 0)
		orderSide := orderSideMap[allOrders[i].Type]
		detail := order.Detail{
			ID:                   strconv.FormatInt(allOrders[i].ID, 10),
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
		orders = append(orders, detail)
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (z *ZB) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
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
func (z *ZB) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	ret, err := z.validateCandlesRequest(p, a, start, end, interval)
	if err != nil {
		return kline.Item{}, err
	}

	p, err = z.FormatExchangeCurrency(p, a)
	if err != nil {
		return kline.Item{}, err
	}

	klineParams := KlinesRequestParams{
		Type:   z.FormatExchangeKlineInterval(interval),
		Symbol: p.String(),
		Since:  start.UnixMilli(),
		Size:   int64(z.Features.Enabled.Kline.ResultLimit),
	}
	var candles KLineResponse
	candles, err = z.GetSpotKline(ctx, klineParams)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range candles.Data {
		if candles.Data[x].KlineTime.Before(start) || candles.Data[x].KlineTime.After(end) {
			continue
		}
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   candles.Data[x].KlineTime,
			Open:   candles.Data[x].Open,
			High:   candles.Data[x].High,
			Low:    candles.Data[x].Low,
			Close:  candles.Data[x].Close,
			Volume: candles.Data[x].Volume,
		})
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (z *ZB) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	ret, err := z.validateCandlesRequest(p, a, start, end, interval)
	if err != nil {
		return kline.Item{}, err
	}

	p, err = z.FormatExchangeCurrency(p, a)
	if err != nil {
		return kline.Item{}, err
	}

	startTime := start
allKlines:
	for {
		klineParams := KlinesRequestParams{
			Type:   z.FormatExchangeKlineInterval(interval),
			Symbol: p.String(),
			Since:  startTime.UnixMilli(),
			Size:   int64(z.Features.Enabled.Kline.ResultLimit),
		}

		candles, err := z.GetSpotKline(ctx, klineParams)
		if err != nil {
			return kline.Item{}, err
		}

		for x := range candles.Data {
			if candles.Data[x].KlineTime.Before(start) || candles.Data[x].KlineTime.After(end) {
				continue
			}
			if startTime.Equal(candles.Data[x].KlineTime) {
				// no new data has been sent
				break allKlines
			}
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   candles.Data[x].KlineTime,
				Open:   candles.Data[x].Open,
				High:   candles.Data[x].High,
				Low:    candles.Data[x].Low,
				Close:  candles.Data[x].Close,
				Volume: candles.Data[x].Volume,
			})
			if x == len(candles.Data)-1 {
				startTime = candles.Data[x].KlineTime
			}
		}
		if len(candles.Data) != int(z.Features.Enabled.Kline.ResultLimit) {
			break allKlines
		}
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

func (z *ZB) validateCandlesRequest(p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return kline.Item{}, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", start, end, err)
	}
	if err := z.ValidateKline(p, a, interval); err != nil {
		return kline.Item{}, err
	}

	return kline.Item{
		Exchange: z.Name,
		Pair:     p,
		Asset:    a,
		Interval: interval,
	}, nil
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

	var availableChains []string
	for x := range chains {
		availableChains = append(availableChains, chains[x].Blockchain)
	}
	return availableChains, nil
}
