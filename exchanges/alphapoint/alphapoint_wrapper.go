package alphapoint

import (
	"context"
	"errors"
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

// SetDefaults sets current default settings
func (e *Exchange) SetDefaults() {
	e.Name = "Alphapoint"
	e.Enabled = true
	e.Verbose = true
	e.API.Endpoints = e.NewEndpoints()
	err := e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      alphapointDefaultAPIURL,
		exchange.WebsocketSpot: alphapointDefaultWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				AccountInfo:       true,
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				GetOrders:         true,
				CancelOrder:       true,
				CancelOrders:      true,
				SubmitOrder:       true,
				ModifyOrder:       true,
				UserTradeHistory:  true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
				TradeFee:          true,
			},

			WebsocketCapabilities: protocol.Features{
				AccountInfo: true,
			},

			WithdrawPermissions: exchange.WithdrawCryptoWith2FA |
				exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
		},
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *Exchange) Setup(_ *config.Exchange) error {
	return common.ErrFunctionNotSupported
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(_ context.Context, _ asset.Item) (currency.Pairs, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(_ context.Context, _ bool) error {
	return common.ErrFunctionNotSupported
}

// UpdateAccountInfo retrieves balances for all enabled currencies on the
// Alphapoint exchange
func (e *Exchange) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = e.Name
	acc, err := e.GetAccountInformation(ctx)
	if err != nil {
		return response, err
	}

	balances := make([]account.Balance, len(acc.Currencies))
	for i := range acc.Currencies {
		balances[i] = account.Balance{
			Currency: currency.NewCode(acc.Currencies[i].Name),
			Total:    float64(acc.Currencies[i].Balance),
			Hold:     float64(acc.Currencies[i].Hold),
			Free:     float64(acc.Currencies[i].Balance) - float64(acc.Currencies[i].Hold),
		}
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		Currencies: balances,
		AssetType:  assetType,
	})

	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}

	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	tick, err := e.GetTicker(ctx, p.String())
	if err != nil {
		return nil, err
	}

	err = ticker.ProcessTicker(&ticker.Price{
		Pair:         p,
		Ask:          tick.Ask,
		Bid:          tick.Bid,
		Low:          tick.Low,
		High:         tick.High,
		Volume:       tick.Volume,
		Last:         tick.Last,
		ExchangeName: e.Name,
		AssetType:    assetType,
	})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(e.Name, p, assetType)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	orderBook := new(orderbook.Book)
	orderbookNew, err := e.GetOrderbook(ctx, p.String())
	if err != nil {
		return orderBook, err
	}

	orderBook.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		orderBook.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Bids[x].Quantity,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	orderBook.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		orderBook.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Asks[x].Quantity,
			Price:  orderbookNew.Asks[x].Price,
		}
	}

	orderBook.Pair = p
	orderBook.Exchange = e.Name
	orderBook.Asset = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(e.Name, p, assetType)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	// https://alphapoint.github.io/slate/#generatetreasuryactivityreport
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(_ context.Context, _ currency.Pair, _ asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order and returns a true value when
// successfully submitted
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}

	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	response, err := e.CreateOrder(ctx,
		fPair.String(),
		s.Side.String(),
		s.Type.String(),
		s.Amount,
		s.Price)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(strconv.FormatInt(response, 10))
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *Exchange) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}
	_, err = e.CancelExistingOrder(ctx, orderIDInt, o.AccountID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders for a given account
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	return order.CancelAllResponse{},
		e.CancelAllExistingOrders(ctx, orderCancellation.AccountID)
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	addresses, err := e.GetDepositAddresses(ctx)
	if err != nil {
		return nil, err
	}

	for x := range addresses {
		if addresses[x].Name == cryptocurrency.String() {
			return &deposit.Address{
				Address: addresses[x].DepositAddress,
			}, nil
		}
	}
	return nil, errors.New("associated currency address not found")
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(_ context.Context, _ *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	resp, err := e.GetOrders(ctx)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for x := range resp {
		for y := range resp[x].OpenOrders {
			if resp[x].OpenOrders[y].State != 1 {
				continue
			}
			orders = append(orders, order.Detail{
				Amount:          resp[x].OpenOrders[y].QtyTotal,
				Exchange:        e.Name,
				ExecutedAmount:  resp[x].OpenOrders[y].QtyTotal - resp[x].OpenOrders[y].QtyRemaining,
				AccountID:       strconv.FormatInt(int64(resp[x].OpenOrders[y].AccountID), 10),
				OrderID:         strconv.FormatInt(int64(resp[x].OpenOrders[y].ServerOrderID), 10),
				Price:           resp[x].OpenOrders[y].Price,
				RemainingAmount: resp[x].OpenOrders[y].QtyRemaining,
				Side:            orderSideMap[resp[x].OpenOrders[y].Side],
				Date:            resp[x].OpenOrders[y].ReceiveTime.Time(),
				Type:            orderTypeMap[resp[x].OpenOrders[y].OrderType],
			})
		}
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	resp, err := e.GetOrders(ctx)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for x := range resp {
		for y := range resp[x].OpenOrders {
			if resp[x].OpenOrders[y].State == 1 {
				continue
			}

			orders = append(orders, order.Detail{
				Amount:          resp[x].OpenOrders[y].QtyTotal,
				AccountID:       strconv.FormatInt(int64(resp[x].OpenOrders[y].AccountID), 10),
				Exchange:        e.Name,
				ExecutedAmount:  resp[x].OpenOrders[y].QtyTotal - resp[x].OpenOrders[y].QtyRemaining,
				OrderID:         strconv.FormatInt(int64(resp[x].OpenOrders[y].ServerOrderID), 10),
				Price:           resp[x].OpenOrders[y].Price,
				RemainingAmount: resp[x].OpenOrders[y].QtyRemaining,
				Side:            orderSideMap[resp[x].OpenOrders[y].Side],
				Date:            resp[x].OpenOrders[y].ReceiveTime.Time(),
				Type:            orderTypeMap[resp[x].OpenOrders[y].OrderType],
			})
		}
	}
	return req.Filter(e.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountInfo(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set
// time interval
func (e *Exchange) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
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
