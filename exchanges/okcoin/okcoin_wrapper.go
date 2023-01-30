package okcoin

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
func (o *OKCoin) GetDefaultConfig() (*config.Exchange, error) {
	o.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = o.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = o.BaseCurrencies

	err := o.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if o.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = o.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults method assignes the default values for OKCoin
func (o *OKCoin) SetDefaults() {
	o.SetErrorDefaults()
	o.Name = okCoinExchangeName
	o.Enabled = true
	o.Verbose = true

	o.API.CredentialsValidator.RequiresKey = true
	o.API.CredentialsValidator.RequiresSecret = true
	o.API.CredentialsValidator.RequiresClientID = true

	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := o.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Margin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	o.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
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
				CancelOrders:        true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrders:              true,
				GetOrder:               true,
				AccountBalance:         true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.OneMin,
					kline.ThreeMin,
					kline.FiveMin,
					kline.FifteenMin,
					kline.ThirtyMin,
					kline.OneHour,
					kline.TwoHour,
					kline.FourHour,
					kline.SixHour,
					kline.TwelveHour,
					kline.OneDay,
					kline.ThreeDay,
					kline.OneWeek,
				),
				ResultLimit: 1440,
			},
		},
	}

	o.Requester, err = request.New(o.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		// TODO: Specify each individual endpoint rate limits as per docs
		request.WithLimiter(request.NewBasicRateLimit(okCoinRateInterval, okCoinStandardRequestRate)),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	o.API.Endpoints = o.NewEndpoints()
	err = o.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      okCoinAPIURL,
		exchange.WebsocketSpot: okCoinWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	o.Websocket = stream.New()
	o.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	o.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	o.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (o *OKCoin) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		o.SetEnabled(false)
		return nil
	}
	err = o.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsEndpoint, err := o.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = o.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             wsEndpoint,
		RunningURL:             wsEndpoint,
		Connector:              o.WsConnect,
		Subscriber:             o.Subscribe,
		Unsubscriber:           o.Unsubscribe,
		GenerateSubscriptions:  o.GenerateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &o.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return o.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            okcoinWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the OKCoin go routine
func (o *OKCoin) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the OKCoin wrapper
func (o *OKCoin) Run() {
	if o.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			o.Name,
			common.IsEnabled(o.Websocket.IsEnabled()))
		o.PrintEnabledPairs()
	}

	forceUpdate := false
	var err error
	if !o.BypassConfigFormatUpgrades {
		var format currency.PairFormat
		format, err = o.GetPairFormat(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				o.Name,
				err)
			return
		}
		var enabled, avail currency.Pairs
		enabled, err = o.CurrencyPairs.GetPairs(asset.Spot, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				o.Name,
				err)
			return
		}

		avail, err = o.CurrencyPairs.GetPairs(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				o.Name,
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
					"%s failed to update currencies.\n",
					o.Name)
			} else {
				log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, o.Name, asset.Spot, p)
				forceUpdate = true

				err = o.UpdatePairs(p, asset.Spot, true, true)
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%s failed to update currencies. Err: %s\n",
						o.Name,
						err)
					return
				}
			}
		}
	}

	if !o.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = o.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (o *OKCoin) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	prods, err := o.GetSpotTokenPairDetails(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, len(prods))
	for x := range prods {
		var pair currency.Pair
		pair, err = currency.NewPairFromStrings(prods[x].BaseCurrency, prods[x].QuoteCurrency)
		if err != nil {
			return nil, err
		}
		pairs[x] = pair
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (o *OKCoin) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := o.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	return o.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (o *OKCoin) UpdateTickers(ctx context.Context, a asset.Item) error {
	if a == asset.Spot {
		resp, err := o.GetSpotAllTokenPairsInformation(ctx)
		if err != nil {
			return err
		}
		pairs, err := o.GetEnabledPairs(a)
		if err != nil {
			return err
		}
		for i := range pairs {
			for j := range resp {
				if !pairs[i].Equal(resp[j].InstrumentID) {
					continue
				}

				err = ticker.ProcessTicker(&ticker.Price{
					Last:         resp[j].Last,
					High:         resp[j].High24h,
					Low:          resp[j].Low24h,
					Bid:          resp[j].BestBid,
					Ask:          resp[j].BestAsk,
					Volume:       resp[j].BaseVolume24h,
					QuoteVolume:  resp[j].QuoteVolume24h,
					Open:         resp[j].Open24h,
					Pair:         pairs[i],
					LastUpdated:  resp[j].Timestamp,
					ExchangeName: o.Name,
					AssetType:    a})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKCoin) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := o.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(o.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (o *OKCoin) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerData, err := ticker.GetTicker(o.Name, p, assetType)
	if err != nil {
		return o.UpdateTicker(ctx, p, assetType)
	}
	return tickerData, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (o *OKCoin) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = o.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		var tradeData []GetSpotFilledOrdersInformationResponse
		tradeData, err = o.GetSpotFilledOrdersInformation(ctx,
			&GetSpotFilledOrdersInformationRequest{
				InstrumentID: p.String(),
			})
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     o.Name,
				TID:          tradeData[i].TradeID,
				CurrencyPair: p,
				Side:         side,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].Timestamp,
			})
		}
	default:
		return nil, fmt.Errorf("%s asset type %v unsupported", o.Name, assetType)
	}
	err = o.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (o *OKCoin) CancelBatchOrders(_ context.Context, _ []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// FetchOrderbook returns orderbook base on the currency pair
func (o *OKCoin) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := o.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	ob, err := orderbook.Get(o.Name, fPair, assetType)
	if err != nil {
		return o.UpdateOrderbook(ctx, fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKCoin) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        o.Name,
		Pair:            p,
		Asset:           a,
		VerifyOrderbook: o.CanVerifyOrderbook,
	}

	fPair, err := o.FormatExchangeCurrency(p, a)
	if err != nil {
		return book, err
	}

	orderbookNew, err := o.GetOrderBook(ctx,
		&GetOrderBookRequest{
			InstrumentID: fPair.String(),
			Size:         200,
		}, a)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		amount, convErr := strconv.ParseFloat(orderbookNew.Bids[x][1], 64)
		if convErr != nil {
			return book, err
		}
		price, convErr := strconv.ParseFloat(orderbookNew.Bids[x][0], 64)
		if convErr != nil {
			return book, err
		}

		var liquidationOrders, orderCount int64
		// Contract specific variables
		if len(orderbookNew.Bids[x]) == 4 {
			liquidationOrders, convErr = strconv.ParseInt(orderbookNew.Bids[x][2], 10, 64)
			if convErr != nil {
				return book, err
			}

			orderCount, convErr = strconv.ParseInt(orderbookNew.Bids[x][3], 10, 64)
			if convErr != nil {
				return book, err
			}
		}

		book.Bids[x] = orderbook.Item{
			Amount:            amount,
			Price:             price,
			LiquidationOrders: liquidationOrders,
			OrderCount:        orderCount,
		}
	}

	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		amount, convErr := strconv.ParseFloat(orderbookNew.Asks[x][1], 64)
		if convErr != nil {
			return book, err
		}
		price, convErr := strconv.ParseFloat(orderbookNew.Asks[x][0], 64)
		if convErr != nil {
			return book, err
		}

		var liquidationOrders, orderCount int64
		// Contract specific variables
		if len(orderbookNew.Asks[x]) == 4 {
			liquidationOrders, convErr = strconv.ParseInt(orderbookNew.Asks[x][2], 10, 64)
			if convErr != nil {
				return book, err
			}

			orderCount, convErr = strconv.ParseInt(orderbookNew.Asks[x][3], 10, 64)
			if convErr != nil {
				return book, err
			}
		}

		book.Asks[x] = orderbook.Item{
			Amount:            amount,
			Price:             price,
			LiquidationOrders: liquidationOrders,
			OrderCount:        orderCount,
		}
	}

	err = book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(o.Name, fPair, a)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (o *OKCoin) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	currencies, err := o.GetSpotTradingAccounts(ctx)
	if err != nil {
		return account.Holdings{}, err
	}

	var resp account.Holdings
	resp.Exchange = o.Name
	currencyAccount := account.SubAccount{AssetType: assetType}

	for i := range currencies {
		hold, parseErr := strconv.ParseFloat(currencies[i].Hold, 64)
		if parseErr != nil {
			return resp, parseErr
		}
		totalValue, parseErr := strconv.ParseFloat(currencies[i].Balance, 64)
		if parseErr != nil {
			return resp, parseErr
		}
		currencyAccount.Currencies = append(currencyAccount.Currencies,
			account.Balance{
				Currency: currency.NewCode(currencies[i].Currency),
				Total:    totalValue,
				Hold:     hold,
				Free:     totalValue - hold,
			})
	}

	resp.Accounts = append(resp.Accounts, currencyAccount)

	creds, err := o.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&resp, creds)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (o *OKCoin) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := o.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(o.Name, creds, assetType)
	if err != nil {
		return o.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *OKCoin) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	accountDepositHistory, err := o.GetAccountDepositHistory(ctx, "")
	if err != nil {
		return nil, err
	}
	accountWithdrawlHistory, err := o.GetAccountWithdrawalHistory(ctx, "")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundHistory, len(accountDepositHistory)+len(accountWithdrawlHistory))
	for x := range accountDepositHistory {
		orderStatus := ""
		switch accountDepositHistory[x].Status {
		case 0:
			orderStatus = "waiting"
		case 1:
			orderStatus = "confirmation account"
		case 2:
			orderStatus = "recharge success"
		}

		resp[x] = exchange.FundHistory{
			Amount:       accountDepositHistory[x].Amount,
			Currency:     accountDepositHistory[x].Currency,
			ExchangeName: o.Name,
			Status:       orderStatus,
			Timestamp:    accountDepositHistory[x].Timestamp,
			TransferID:   accountDepositHistory[x].TransactionID,
			TransferType: "deposit",
		}
	}

	for i := range accountWithdrawlHistory {
		resp[len(accountDepositHistory)+i] = exchange.FundHistory{
			Amount:       accountWithdrawlHistory[i].Amount,
			Currency:     accountWithdrawlHistory[i].Currency,
			ExchangeName: o.Name,
			Status:       OrderStatus[accountWithdrawlHistory[i].Status],
			Timestamp:    accountWithdrawlHistory[i].Timestamp,
			TransferID:   accountWithdrawlHistory[i].TransactionID,
			TransferType: "withdrawal",
		}
	}
	return resp, nil
}

// SubmitOrder submits a new order
func (o *OKCoin) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}

	fPair, err := o.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	req := PlaceOrderRequest{
		ClientOID:    s.ClientID,
		InstrumentID: fPair.String(),
		Side:         s.Side.Lower(),
		Type:         s.Type.Lower(),
		Size:         strconv.FormatFloat(s.Amount, 'f', -1, 64),
	}
	if s.Type == order.Limit {
		req.Price = strconv.FormatFloat(s.Price, 'f', -1, 64)
	}

	orderResponse, err := o.PlaceSpotOrder(ctx, &req)
	if err != nil {
		return nil, err
	}

	if !orderResponse.Result {
		return nil, order.ErrUnableToPlaceOrder
	}
	return s.DeriveSubmitResponse(orderResponse.OrderID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *OKCoin) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (o *OKCoin) CancelOrder(ctx context.Context, cancel *order.Cancel) error {
	err := cancel.Validate(cancel.StandardCancel())
	if err != nil {
		return err
	}

	orderID, err := strconv.ParseInt(cancel.OrderID, 10, 64)
	if err != nil {
		return err
	}

	fpair, err := o.FormatExchangeCurrency(cancel.Pair,
		cancel.AssetType)
	if err != nil {
		return err
	}

	orderCancellationResponse, err := o.CancelSpotOrder(ctx,
		&CancelSpotOrderRequest{
			InstrumentID: fpair.String(),
			OrderID:      orderID,
		})
	if err != nil {
		return err
	}
	if !orderCancellationResponse.Result {
		return fmt.Errorf("order %d failed to be cancelled",
			orderCancellationResponse.OrderID)
	}

	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (o *OKCoin) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	orderIDs := strings.Split(orderCancellation.OrderID, ",")
	resp := order.CancelAllResponse{}
	resp.Status = make(map[string]string)
	orderIDNumbers := make([]int64, 0, len(orderIDs))
	for i := range orderIDs {
		orderIDNumber, err := strconv.ParseInt(orderIDs[i], 10, 64)
		if err != nil {
			resp.Status[orderIDs[i]] = err.Error()
			continue
		}
		orderIDNumbers = append(orderIDNumbers, orderIDNumber)
	}

	fpair, err := o.FormatExchangeCurrency(orderCancellation.Pair,
		orderCancellation.AssetType)
	if err != nil {
		return resp, err
	}

	cancelOrdersResponse, err := o.CancelMultipleSpotOrders(ctx,
		&CancelMultipleSpotOrdersRequest{
			InstrumentID: fpair.String(),
			OrderIDs:     orderIDNumbers,
		})
	if err != nil {
		return resp, err
	}

	for x := range cancelOrdersResponse {
		for y := range cancelOrdersResponse[x] {
			resp.Status[strconv.FormatInt(cancelOrdersResponse[x][y].OrderID, 10)] = strconv.FormatBool(cancelOrdersResponse[x][y].Result)
		}
	}

	return resp, err
}

// GetOrderInfo returns order information based on order ID
func (o *OKCoin) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail
	if assetType != asset.Spot {
		return resp, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}

	mOrder, err := o.GetSpotOrder(ctx, &GetSpotOrderRequest{OrderID: orderID})
	if err != nil {
		return resp, err
	}

	format, err := o.GetPairFormat(assetType, false)
	if err != nil {
		return resp, err
	}

	p, err := currency.NewPairDelimiter(mOrder.InstrumentID, format.Delimiter)
	if err != nil {
		return resp, err
	}

	status, err := order.StringToOrderStatus(mOrder.Status)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
	}

	side, err := order.StringToOrderSide(mOrder.Side)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
	}
	resp = order.Detail{
		Amount:         mOrder.Size,
		Pair:           p,
		Exchange:       o.Name,
		Date:           mOrder.Timestamp,
		ExecutedAmount: mOrder.FilledSize,
		Status:         status,
		Side:           side,
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *OKCoin) GetDepositAddress(ctx context.Context, c currency.Code, _, _ string) (*deposit.Address, error) {
	wallet, err := o.GetAccountDepositAddressForCurrency(ctx, c.Lower().String())
	if err != nil {
		return nil, err
	}
	if len(wallet) == 0 {
		return nil, fmt.Errorf("%w for currency %s",
			errNoAccountDepositAddress,
			c)
	}
	return &deposit.Address{
		Address: wallet[0].Address,
		Tag:     wallet[0].Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *OKCoin) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawal, err := o.AccountWithdraw(ctx,
		&AccountWithdrawRequest{
			Amount:      withdrawRequest.Amount,
			Currency:    withdrawRequest.Currency.Lower().String(),
			Destination: 4, // 1, 2, 3 are all internal
			Fee:         withdrawRequest.Crypto.FeeAmount,
			ToAddress:   withdrawRequest.Crypto.Address,
			TradePwd:    withdrawRequest.TradePassword,
		})
	if err != nil {
		return nil, err
	}
	if !withdrawal.Result {
		return nil,
			fmt.Errorf("could not withdraw currency %s to %s, no error specified",
				withdrawRequest.Currency,
				withdrawRequest.Crypto.Address)
	}

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(withdrawal.WithdrawalID, 10),
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKCoin) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKCoin) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (o *OKCoin) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (o *OKCoin) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var resp []order.Detail
	for x := range req.Pairs {
		var fPair currency.Pair
		fPair, err = o.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		var spotOpenOrders []GetSpotOrderResponse
		spotOpenOrders, err = o.GetSpotOpenOrders(ctx,
			&GetSpotOpenOrdersRequest{
				InstrumentID: fPair.String(),
			})
		if err != nil {
			return nil, err
		}
		for i := range spotOpenOrders {
			var status order.Status
			status, err = order.StringToOrderStatus(spotOpenOrders[i].Status)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var side order.Side
			side, err = order.StringToOrderSide(spotOpenOrders[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(spotOpenOrders[i].Type)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			resp = append(resp, order.Detail{
				OrderID:        spotOpenOrders[i].OrderID,
				Price:          spotOpenOrders[i].Price,
				Amount:         spotOpenOrders[i].Size,
				Pair:           req.Pairs[x],
				Exchange:       o.Name,
				Side:           side,
				Type:           orderType,
				ExecutedAmount: spotOpenOrders[i].FilledSize,
				Date:           spotOpenOrders[i].Timestamp,
				Status:         status,
			})
		}
	}
	return req.Filter(o.Name, resp), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (o *OKCoin) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var resp []order.Detail
	for x := range req.Pairs {
		var fPair currency.Pair
		fPair, err = o.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		var spotOrders []GetSpotOrderResponse
		spotOrders, err = o.GetSpotOrders(ctx,
			&GetSpotOrdersRequest{
				Status:       strings.Join([]string{"filled", "cancelled", "failure"}, "|"),
				InstrumentID: fPair.String(),
			})
		if err != nil {
			return nil, err
		}
		for i := range spotOrders {
			var status order.Status
			status, err = order.StringToOrderStatus(spotOrders[i].Status)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var side order.Side
			side, err = order.StringToOrderSide(spotOrders[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(spotOrders[i].Type)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			detail := order.Detail{
				OrderID:              spotOrders[i].OrderID,
				Price:                spotOrders[i].Price,
				AverageExecutedPrice: spotOrders[i].PriceAvg,
				Amount:               spotOrders[i].Size,
				ExecutedAmount:       spotOrders[i].FilledSize,
				RemainingAmount:      spotOrders[i].Size - spotOrders[i].FilledSize,
				Pair:                 req.Pairs[x],
				Exchange:             o.Name,
				Side:                 side,
				Type:                 orderType,
				Date:                 spotOrders[i].Timestamp,
				Status:               status,
			}
			detail.InferCostsAndTimes()
			resp = append(resp, detail)
		}
	}
	return req.Filter(o.Name, resp), nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *OKCoin) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !o.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return o.GetFee(ctx, feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (o *OKCoin) GetWithdrawCapabilities() uint32 {
	return o.GetWithdrawPermissions()
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (o *OKCoin) AuthenticateWebsocket(ctx context.Context) error {
	return o.WsLogin(ctx)
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (o *OKCoin) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := o.UpdateAccountInfo(ctx, assetType)
	return o.CheckTransientError(err)
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (o *OKCoin) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (o *OKCoin) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := o.GetKlineRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries, err := o.GetMarketData(ctx, &GetMarketDataRequest{
		Asset:        a,
		Start:        start.UTC().Format(time.RFC3339),
		End:          end.UTC().Format(time.RFC3339),
		Granularity:  o.FormatExchangeKlineInterval(interval),
		InstrumentID: req.RequestFormatted.String(),
	})
	if err != nil {
		return nil, err
	}

	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (o *OKCoin) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := o.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	gran := o.FormatExchangeKlineInterval(interval)
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles []kline.Candle
		candles, err = o.GetMarketData(ctx, &GetMarketDataRequest{
			Asset:        a,
			Start:        req.RangeHolder.Ranges[x].Start.Time.UTC().Format(time.RFC3339),
			End:          req.RangeHolder.Ranges[x].End.Time.UTC().Format(time.RFC3339),
			Granularity:  gran,
			InstrumentID: req.RequestFormatted.String(),
		})
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, candles...)
	}
	return req.ProcessResponse(timeSeries)
}
