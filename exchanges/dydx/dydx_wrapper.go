package dydx

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
func (dy *DYDX) GetDefaultConfig() (*config.Exchange, error) {
	dy.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = dy.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = dy.BaseCurrencies

	err := dy.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if dy.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := dy.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Dydx
func (dy *DYDX) SetDefaults() {
	dy.Name = "Dydx"
	dy.Enabled = true
	dy.Verbose = true
	dy.API.CredentialsValidator.RequiresKey = true
	dy.API.CredentialsValidator.RequiresSecret = true
	dy.API.CredentialsValidator.RequiresClientID = true
	dy.API.CredentialsValidator.RequiresPEM = true

	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := dy.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	dy.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.OneMin,
					kline.FiveMin,
					kline.FifteenMin,
					kline.ThirtyMin,
					kline.OneHour,
					kline.FourHour,
					kline.OneDay,
				),
				ResultLimit: 5000,
			},
		},
	}
	dy.Requester, err = request.New(dy.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetupRateLimiter()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	dy.API.Endpoints = dy.NewEndpoints()
	err = dy.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              dydxAPIURL,
		exchange.RestSpotSupplementary: dydxOnlySignOnDomainMainnet,
		exchange.WebsocketSpot:         dydxWSAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	dy.Websocket = stream.New()
	dy.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	dy.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	dy.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (dy *DYDX) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		dy.SetEnabled(false)
		return nil
	}
	err = dy.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := dy.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = dy.Websocket.Setup(
		&stream.WebsocketSetup{
			ExchangeConfig:        exch,
			DefaultURL:            dydxWSAPIURL,
			RunningURL:            wsRunningEndpoint,
			Connector:             dy.WsConnect,
			Subscriber:            dy.Subscribe,
			Unsubscriber:          dy.Unsubscribe,
			GenerateSubscriptions: dy.GenerateDefaultSubscriptions,
			Features:              &dy.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}
	return dy.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  dy.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Dydx go routine
func (dy *DYDX) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		dy.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Dydx wrapper
func (dy *DYDX) Run() {
	if dy.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			dy.Name,
			common.IsEnabled(dy.Websocket.IsEnabled()))
		dy.PrintEnabledPairs()
	}

	if !dy.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := dy.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			dy.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (dy *DYDX) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !dy.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, dy.Name)
	}
	instruments, err := dy.GetMarkets(ctx, "")
	if err != nil {
		return nil, err
	}
	pairs := make(currency.Pairs, len(instruments.Markets))
	count := 0
	for key := range instruments.Markets {
		cp, err := currency.NewPairFromString(key)
		if err != nil {
			return nil, err
		}
		pairs[count] = cp
		count++
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (dy *DYDX) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := dy.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	return dy.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (dy *DYDX) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := dy.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	stats, err := dy.GetMarketStats(ctx, fPair.String(), 1)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		return nil, fmt.Errorf("missing ticker data for instrument %s", fPair.String())
	}
	for key, tick := range stats {
		if !fPair.IsEmpty() && !strings.EqualFold(fPair.String(), key) {
			continue
		}
		cp, err := currency.NewPairFromString(tick.Market)
		if err != nil {
			return nil, err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         cp,
			High:         tick.High,
			Low:          tick.Low,
			Close:        tick.Close,
			Open:         tick.Open,
			Volume:       tick.BaseVolume,
			QuoteVolume:  tick.QuoteVolume,
			ExchangeName: dy.Name,
			AssetType:    assetType,
		})
		if err != nil {
			return nil, err
		}
		if !fPair.IsEmpty() && cp.Equal(fPair) {
			return ticker.GetTicker(dy.Name, p, assetType)
		}
	}
	return ticker.GetTicker(dy.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (dy *DYDX) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	pairs, err := dy.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	if !dy.SupportsAsset(assetType) {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	stats, err := dy.GetMarketStats(ctx, "", 30)
	if err != nil {
		return err
	}

	for x := range stats {
		pair, err := currency.NewPairFromString(stats[x].Market)
		if err != nil {
			return err
		}
		for i := range pairs {
			if !pair.Equal(pairs[i]) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         pair,
				High:         stats[x].High,
				Low:          stats[x].Low,
				Close:        stats[x].Close,
				Open:         stats[x].Open,
				Volume:       stats[x].BaseVolume,
				QuoteVolume:  stats[x].QuoteVolume,
				ExchangeName: dy.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (dy *DYDX) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(dy.Name, p, assetType)
	if err != nil {
		return dy.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (dy *DYDX) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(dy.Name, pair, assetType)
	if err != nil {
		return dy.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (dy *DYDX) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        dy.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: dy.CanVerifyOrderbook,
	}
	fPair, err := dy.FormatSymbol(pair, assetType)
	if err != nil {
		return nil, err
	}
	books, err := dy.GetOrderbooks(ctx, fPair)
	if err != nil {
		return nil, err
	}
	book.Asks = books.Asks.generateOrderbookItem()
	book.Bids = books.Bids.generateOrderbookItem()
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(dy.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (dy *DYDX) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := dy.GetAccounts(ctx)
	if err != nil {
		return account.Holdings{}, err
	}

	// TODO: incomplete implementation.

	var resp account.Holdings
	for x := range acc.Accounts {
		var subAcc = account.SubAccount{ID: acc.Accounts[x].AccountNumber, AssetType: asset.Spot}
		if err != nil {
			return account.Holdings{}, err
		}
		subAcc.Currencies = append(subAcc.Currencies, account.Balance{
			Currency: currency.USDC,
			Total:    acc.Accounts[x].QuoteBalance,
			Hold:     acc.Accounts[x].PendingWithdrawals,
			Free:     acc.Accounts[x].FreeCollateral,
		})
		resp.Accounts = append(resp.Accounts, subAcc)
	}
	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (dy *DYDX) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := dy.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(dy.Name, creds, assetType)
	if err != nil {
		return dy.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (dy *DYDX) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	transfers, err := dy.GetTransfers(ctx, "", 0, time.Time{})
	if err != nil {
		return nil, err
	}
	fundingDatas := make([]exchange.FundHistory, len(transfers.Transfers))
	for x := range transfers.Transfers {
		switch transfers.Transfers[x].Type {
		case "DEPOSIT":
			fundingDatas[x] = exchange.FundHistory{
				Timestamp:         transfers.Transfers[x].CreatedAt,
				TransferType:      transfers.Transfers[x].Type,
				ExchangeName:      dy.Name,
				CryptoFromAddress: transfers.Transfers[x].FromAddress,
				CryptoToAddress:   transfers.Transfers[x].ToAddress,
				CryptoTxID:        transfers.Transfers[x].ID,
				Status:            transfers.Transfers[x].Status,
				Amount:            transfers.Transfers[x].CreditAmount,
				Currency:          transfers.Transfers[x].CreditAsset,
			}
		case "WITHDRAWAL", "FAST_WITHDRAWAL":
			fundingDatas[x] = exchange.FundHistory{
				Timestamp:         transfers.Transfers[x].CreatedAt,
				TransferType:      transfers.Transfers[x].Type,
				ExchangeName:      dy.Name,
				CryptoFromAddress: transfers.Transfers[x].FromAddress,
				CryptoToAddress:   transfers.Transfers[x].ToAddress,
				CryptoTxID:        transfers.Transfers[x].ID,
				Status:            transfers.Transfers[x].Status,
				Amount:            transfers.Transfers[x].DebitAmount,
				Currency:          transfers.Transfers[x].DebitAsset,
			}
		}
	}
	return fundingDatas, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (dy *DYDX) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	if !dy.SupportsAsset(a) {
		return nil, fmt.Errorf("asset %v not supported", a)
	}
	transfers, err := dy.GetTransfers(ctx, "", 0, time.Time{})
	if err != nil {
		return nil, err
	}
	withdrawalHistory := []exchange.WithdrawalHistory{}
	for x := range transfers.Transfers {
		if transfers.Transfers[x].Type == "WITHDRAWAL" || transfers.Transfers[x].Type == "FAST_WITHDRAWAL" {
			withdrawalHistory = append(withdrawalHistory, exchange.WithdrawalHistory{
				Timestamp:       transfers.Transfers[x].CreatedAt,
				TransferType:    transfers.Transfers[x].Type,
				CryptoToAddress: transfers.Transfers[x].ToAddress,
				CryptoTxID:      transfers.Transfers[x].ID,
				Status:          transfers.Transfers[x].Status,
				Amount:          transfers.Transfers[x].DebitAmount,
				Currency:        transfers.Transfers[x].DebitAsset,
			})
		}
	}
	return withdrawalHistory, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (dy *DYDX) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if !dy.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	format, err := dy.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := format.Format(p)
	trades, err := dy.GetTrades(ctx, instrumentID, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades))
	for x := range trades {
		var side order.Side
		side, err = order.StringToOrderSide(trades[x].Side)
		if err != nil {
			return nil, err
		}
		resp[x] = trade.Data{
			Exchange:     dy.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        trades[x].Price,
			Amount:       trades[x].Size,
			Timestamp:    trades[x].CreatedAt,
		}
	}
	if dy.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(dy.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (dy *DYDX) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, _ time.Time) ([]trade.Data, error) {
	if !dy.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	format, err := dy.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := format.Format(p)
	trades, err := dy.GetTrades(ctx, instrumentID, timestampStart, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades))
	for x := range trades {
		var side order.Side
		side, err = order.StringToOrderSide(trades[x].Side)
		if err != nil {
			return nil, err
		}
		resp[x] = trade.Data{
			Exchange:     dy.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        trades[x].Price,
			Amount:       trades[x].Size,
			Timestamp:    trades[x].CreatedAt,
		}
	}
	if dy.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(dy.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (dy *DYDX) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	formattedPair, err := dy.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	ord, err := dy.CreateNewOrder(ctx, &CreateOrderRequestParams{
		Market:     formattedPair.String(),
		Side:       s.Side.Lower(),
		Type:       s.Type.Lower(),
		PostOnly:   s.PostOnly,
		Size:       s.Amount,
		Price:      s.Price,
		ReduceOnly: s.ReduceOnly,
		Expiration: time.Now().Add(time.Hour * 24 * 3).UTC().Format("2006-01-02T15:04:05.999Z"),
	})
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(ord.ID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (dy *DYDX) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (dy *DYDX) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	if ord.OrderID == "" {
		return errors.New("Order ID is required")
	}
	_, err := dy.CancelOrderByID(ctx, ord.OrderID)
	if err != nil {
		return err
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (dy *DYDX) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	var resp order.CancelBatchResponse
	resp.Status = map[string]string{}
	var err error
	for x := range orders {
		if !dy.SupportsAsset(orders[x].AssetType) {
			return resp, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, orders[x].AssetType)
		}
		if orders[x].Pair.IsEmpty() && orders[x].OrderID == "" {
			return resp, errors.New("either market or order ID must to be specified")
		}
		if orders[x].OrderID != "" && (orders[x].Side == order.UnknownSide || orders[x].Side == order.AnySide) {
			return resp, errors.New("if id is present in the request then side is required")
		}
		var formattedPair string
		if !orders[x].Pair.IsEmpty() {
			formattedPair, err = dy.FormatSymbol(orders[x].Pair, orders[x].AssetType)
			if err != nil {
				return resp, err
			}
		}
		var orderSide string
		if orders[x].Side != order.UnknownSide && orders[x].Side != order.AnySide {
			orderSide = orders[x].Side.Lower()
		}
		result, err := dy.CancelActiveOrders(ctx, formattedPair, orderSide, orders[x].OrderID)
		if err != nil {
			return resp, err
		}
		for i := range result {
			resp.Status[result[i].ID] = result[i].Status
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (dy *DYDX) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrFunctionNotSupported
}

// GetOrderInfo returns order information based on order ID
func (dy *DYDX) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail
	if !dy.SupportsAsset(assetType) {
		return resp, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	if orderID == "" {
		return resp, errors.New("order ID is required")
	}
	orderDetail, err := dy.GetOrderByID(ctx, orderID)
	if err != nil {
		return resp, err
	}
	pair, err := currency.NewPairFromString(orderDetail.Market)
	if err != nil {
		return resp, err
	}
	orderStatus, err := order.StringToOrderStatus(orderDetail.Status)
	if err != nil {
		return resp, err
	}
	orderSide, err := order.StringToOrderSide(orderDetail.Side)
	if err != nil {
		return resp, err
	}
	orderType, err := order.StringToOrderType(orderDetail.Type)
	if err != nil {
		return resp, err
	}
	return order.Detail{
		OrderID:         orderDetail.ID,
		Amount:          orderDetail.Size,
		ClientOrderID:   orderDetail.ClientAssignedID,
		Date:            orderDetail.CreatedAt,
		Exchange:        dy.Name,
		ExecutedAmount:  orderDetail.Size - orderDetail.RemainingSize,
		Pair:            pair,
		RemainingAmount: orderDetail.RemainingSize,
		AssetType:       asset.Spot,
		Status:          orderStatus,
		Side:            orderSide,
		Type:            orderType,
		Fee:             orderDetail.LimitFee,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (dy *DYDX) GetDepositAddress(ctx context.Context, c currency.Code, accountID, chain string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (dy *DYDX) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	response, err := dy.CreateWithdrawal(ctx, WithdrawalParam{
		Asset:      withdrawRequest.Currency.String(),
		Amount:     withdrawRequest.Amount,
		Expiration: time.Now().Add(time.Hour * 24 * 20).UTC().Format(timeFormat),
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   dy.Name,
		ID:     response.ID,
		Status: response.Status,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (dy *DYDX) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (dy *DYDX) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (dy *DYDX) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	if !dy.SupportsAsset(getOrdersRequest.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
	var filteredOrders order.FilteredOrders
	for i := range getOrdersRequest.Pairs {
		market, err := dy.FormatSymbol(getOrdersRequest.Pairs[i], getOrdersRequest.AssetType)
		if err != nil {
			return nil, err
		}
		orders, err := dy.GetOpenOrders(ctx, market, getOrdersRequest.Side.String(), getOrdersRequest.OrderID)
		if err != nil {
			return nil, err
		}
		for x := range orders {
			cp, err := currency.NewPairFromString(orders[x].Market)
			if err != nil {
				return nil, err
			}
			orderStatus, err := order.StringToOrderStatus(orders[x].Status)
			if err != nil {
				return nil, err
			}
			orderType, err := order.StringToOrderType(orders[x].Type)
			if err != nil {
				return nil, err
			}
			orderSide, err := order.StringToOrderSide(orders[x].Side)
			if err != nil {
				return nil, err
			}
			filteredOrders = append(filteredOrders, order.Detail{
				OrderID:         orders[x].ID,
				Amount:          orders[x].Size,
				AssetType:       asset.Spot,
				TriggerPrice:    orders[x].TriggerPrice,
				ClientOrderID:   orders[x].ClientAssignedID,
				Date:            orders[x].CreatedAt,
				Exchange:        dy.Name,
				Pair:            cp,
				Price:           orders[x].Price,
				RemainingAmount: orders[x].RemainingSize,
				Status:          orderStatus,
				Type:            orderType,
				Side:            orderSide,
				Fee:             orders[x].LimitFee,
			})
		}
	}
	return getOrdersRequest.Filter(dy.Name, filteredOrders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (dy *DYDX) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	if !dy.SupportsAsset(getOrdersRequest.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
	var market string
	if len(getOrdersRequest.Pairs) == 1 {
		market, err = dy.FormatSymbol(getOrdersRequest.Pairs[0], getOrdersRequest.AssetType)
		if err != nil {
			return nil, err
		}
	}
	var orderTypeString string
	switch getOrdersRequest.Type {
	case order.Limit, order.Stop, order.TrailingStop:
		orderTypeString = getOrdersRequest.Type.String()
	case order.TakeProfit:
		orderTypeString = "TAKE_PROFIT"
	}
	orders, err := dy.GetOrders(ctx, market, "", getOrdersRequest.Side.String(), orderTypeString, 0, getOrdersRequest.EndTime, true)
	if err != nil {
		return nil, err
	}
	var filteredOrders order.FilteredOrders
	for x := range orders {
		cp, err := currency.NewPairFromString(orders[x].Market)
		if err != nil {
			return nil, err
		}
		if len(getOrdersRequest.Pairs) > 0 {
			for p := range getOrdersRequest.Pairs {
				if getOrdersRequest.Pairs[p].Equal(cp) {
					goto EXIST
				}
			}
			continue
		}
	EXIST:
		orderStatus, err := order.StringToOrderStatus(orders[x].Status)
		if err != nil {
			return nil, err
		}
		orderType, err := order.StringToOrderType(orders[x].Type)
		if err != nil {
			return nil, err
		}
		orderSide, err := order.StringToOrderSide(orders[x].Side)
		if err != nil {
			return nil, err
		}
		filteredOrders = append(filteredOrders, order.Detail{
			OrderID:         orders[x].ID,
			Amount:          orders[x].Size,
			AssetType:       asset.Spot,
			TriggerPrice:    orders[x].TriggerPrice,
			ClientOrderID:   orders[x].ClientAssignedID,
			Date:            orders[x].CreatedAt,
			Exchange:        dy.Name,
			Pair:            cp,
			Price:           orders[x].Price,
			RemainingAmount: orders[x].RemainingSize,
			Status:          orderStatus,
			Type:            orderType,
			Side:            orderSide,
			Fee:             orders[x].LimitFee,
		})
	}
	return getOrdersRequest.Filter(dy.Name, filteredOrders), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
// https://dydxprotocol.github.io/v3-teacher/?json#order-limitfee
func (dy *DYDX) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	if !dy.IsRESTAuthenticationSupported() {
		return fee, errors.New("order limit fee is authenticated")
	}
	user, err := dy.GetUsers(ctx)
	if err != nil {
		return fee, err
	}
	switch {
	case feeBuilder.IsMaker, feeBuilder.PostOnly:
		fee = user.User.MakerFeeRate * feeBuilder.Amount * feeBuilder.PurchasePrice
	case !feeBuilder.IsMaker,
		feeBuilder.OrderType == order.FillOrKill ||
			feeBuilder.OrderType == order.ImmediateOrCancel:
		fee = user.User.TakerFeeRate * feeBuilder.Amount * feeBuilder.PurchasePrice
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// ValidateCredentials validates current credentials used for wrapper
func (dy *DYDX) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := dy.UpdateAccountInfo(ctx, assetType)
	return dy.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (dy *DYDX) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !pair.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	req, err := dy.GetKlineRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	candles, err := dy.GetCandlesForMarket(ctx, req.RequestFormatted.String(), interval, "", "", 0)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].UpdatedAt,
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].BaseTokenVolume,
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (dy *DYDX) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (dy *DYDX) GetServerTime(ctx context.Context, ai asset.Item) (time.Time, error) {
	serverTime, err := dy.GetAPIServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return serverTime.Epoch, nil
}
