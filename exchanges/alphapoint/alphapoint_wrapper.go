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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config for Alphapoint
func (a *Alphapoint) GetDefaultConfig() (*config.Exchange, error) {
	return nil, common.ErrFunctionNotSupported
}

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

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (a *Alphapoint) FetchTradablePairs(_ context.Context, _ asset.Item) (currency.Pairs, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (a *Alphapoint) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
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

// FetchAccountInfo retrieves balances for all enabled currencies on the
// Alphapoint exchange
func (a *Alphapoint) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := a.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(a.Name, creds, assetType)
	if err != nil {
		return a.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (a *Alphapoint) UpdateTickers(assetType asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (a *Alphapoint) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
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

// FetchTicker returns the ticker for a currency pair
func (a *Alphapoint) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := ticker.GetTicker(a.Name, p, assetType)
	if err != nil {
		return a.UpdateTicker(ctx, p, assetType)
	}
	return tick, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (a *Alphapoint) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	orderbookNew, err := a.GetOrderbook(ctx, p.String())
	if err != nil {
		return orderBook, err
	}

	orderBook.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		orderBook.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Quantity,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	orderBook.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		orderBook.Asks[x] = orderbook.Item{
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

// FetchOrderbook returns the orderbook for a currency pair
func (a *Alphapoint) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(a.Name, p, assetType)
	if err != nil {
		return a.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (a *Alphapoint) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	// https://alphapoint.github.io/slate/#generatetreasuryactivityreport
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (a *Alphapoint) GetWithdrawalsHistory(ctx context.Context, c currency.Code, as asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (a *Alphapoint) GetRecentTrades(_ currency.Pair, _ asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (a *Alphapoint) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order and returns a true value when
// successfully submitted
func (a *Alphapoint) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
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
func (a *Alphapoint) ModifyOrder(_ *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrNotYetImplemented
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
func (a *Alphapoint) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
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
func (a *Alphapoint) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (float64, error) {
	orders, err := a.GetOrders(ctx)
	if err != nil {
		return 0, err
	}

	for x := range orders {
		for y := range orders[x].OpenOrders {
			if strconv.Itoa(orders[x].OpenOrders[y].ServerOrderID) == orderID {
				return orders[x].OpenOrders[y].QtyRemaining, nil
			}
		}
	}
	return 0, errors.New("order not found")
}

// GetDepositAddress returns a deposit address for a specified currency
func (a *Alphapoint) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (string, error) {
	addreses, err := a.GetDepositAddresses(ctx)
	if err != nil {
		return "", err
	}

	for x := range addreses {
		if addreses[x].Name == cryptocurrency.String() {
			return addreses[x].DepositAddress, nil
		}
	}
	return "", errors.New("associated currency address not found")
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
func (a *Alphapoint) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (a *Alphapoint) GetFeeByType(_ *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (a *Alphapoint) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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

			orderDetail := order.Detail{
				Amount:          resp[x].OpenOrders[y].QtyTotal,
				Exchange:        a.Name,
				ExecutedAmount:  resp[x].OpenOrders[y].QtyTotal - resp[x].OpenOrders[y].QtyRemaining,
				AccountID:       strconv.FormatInt(int64(resp[x].OpenOrders[y].AccountID), 10),
				OrderID:         strconv.FormatInt(int64(resp[x].OpenOrders[y].ServerOrderID), 10),
				Price:           resp[x].OpenOrders[y].Price,
				RemainingAmount: resp[x].OpenOrders[y].QtyRemaining,
			}

			orderDetail.Side = orderSideMap[resp[x].OpenOrders[y].Side]
			orderDetail.Date = time.Unix(resp[x].OpenOrders[y].ReceiveTime, 0)
			orderDetail.Type = orderTypeMap[resp[x].OpenOrders[y].OrderType]
			orders = append(orders, orderDetail)
		}
	}
	return req.Filter(a.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (a *Alphapoint) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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

			orderDetail := order.Detail{
				Amount:          resp[x].OpenOrders[y].QtyTotal,
				AccountID:       strconv.FormatInt(int64(resp[x].OpenOrders[y].AccountID), 10),
				Exchange:        a.Name,
				ExecutedAmount:  resp[x].OpenOrders[y].QtyTotal - resp[x].OpenOrders[y].QtyRemaining,
				OrderID:         strconv.FormatInt(int64(resp[x].OpenOrders[y].ServerOrderID), 10),
				Price:           resp[x].OpenOrders[y].Price,
				RemainingAmount: resp[x].OpenOrders[y].QtyRemaining,
			}

			orderDetail.Side = orderSideMap[resp[x].OpenOrders[y].Side]
			orderDetail.Date = time.Unix(resp[x].OpenOrders[y].ReceiveTime, 0)
			orderDetail.Type = orderTypeMap[resp[x].OpenOrders[y].OrderType]
			orders = append(orders, orderDetail)
		}
	}
	return req.Filter(a.Name, orders), nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (a *Alphapoint) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := a.UpdateAccountInfo(ctx, assetType)
	return a.CheckTransientError(err)
}
