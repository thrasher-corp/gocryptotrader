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
func (a *Alphapoint) SetDefaults() {
	a.Name = "Alphapoint"
	a.Enabled = true
	a.Verbose = true
	a.API.Endpoints = a.NewEndpoints()
	err := a.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      alphapointDefaultAPIURL,
		exchange.WebsocketSpot: alphapointDefaultWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	a.API.CredentialsValidator.RequiresKey = true
	a.API.CredentialsValidator.RequiresSecret = true

	a.Features = exchange.Features{
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

	a.Requester, err = request.New(a.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup takes in the supplied exchange configuration details and sets params
func (a *Alphapoint) Setup(_ *config.Exchange) error {
	return common.ErrFunctionNotSupported
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (a *Alphapoint) FetchTradablePairs(_ context.Context, _ asset.Item) (currency.Pairs, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (a *Alphapoint) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (a *Alphapoint) UpdateTradablePairs(_ context.Context, _ bool) error {
	return common.ErrFunctionNotSupported
}

// UpdateAccountInfo retrieves balances for all enabled currencies on the
// Alphapoint exchange
func (a *Alphapoint) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = a.Name
	acc, err := a.GetAccountInformation(ctx)
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

	creds, err := a.GetCredentials(ctx)
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
func (a *Alphapoint) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *Alphapoint) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := a.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	tick, err := a.GetTicker(ctx, p.String())
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
		ExchangeName: a.Name,
		AssetType:    assetType,
	})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(a.Name, p, assetType)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *Alphapoint) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := a.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	orderBook := new(orderbook.Book)
	orderbookNew, err := a.GetOrderbook(ctx, p.String())
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
	orderBook.Exchange = a.Name
	orderBook.Asset = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(a.Name, p, assetType)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (a *Alphapoint) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	// https://alphapoint.github.io/slate/#generatetreasuryactivityreport
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (a *Alphapoint) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (a *Alphapoint) GetRecentTrades(_ context.Context, _ currency.Pair, _ asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (a *Alphapoint) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order and returns a true value when
// successfully submitted
func (a *Alphapoint) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(a.GetTradingRequirements()); err != nil {
		return nil, err
	}

	fPair, err := a.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	response, err := a.CreateOrder(ctx,
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
func (a *Alphapoint) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (a *Alphapoint) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}
	_, err = a.CancelExistingOrder(ctx, orderIDInt, o.AccountID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (a *Alphapoint) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders for a given account
func (a *Alphapoint) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	return order.CancelAllResponse{},
		a.CancelAllExistingOrders(ctx, orderCancellation.AccountID)
}

// GetOrderInfo returns order information based on order ID
func (a *Alphapoint) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (a *Alphapoint) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	addresses, err := a.GetDepositAddresses(ctx)
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
func (a *Alphapoint) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (a *Alphapoint) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (a *Alphapoint) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *Alphapoint) GetFeeByType(_ context.Context, _ *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (a *Alphapoint) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	resp, err := a.GetOrders(ctx)
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
				Exchange:        a.Name,
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
	return req.Filter(a.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (a *Alphapoint) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	resp, err := a.GetOrders(ctx)
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
				Exchange:        a.Name,
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
	return req.Filter(a.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (a *Alphapoint) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := a.UpdateAccountInfo(ctx, assetType)
	return a.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (a *Alphapoint) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set
// time interval
func (a *Alphapoint) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (a *Alphapoint) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (a *Alphapoint) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (a *Alphapoint) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}
