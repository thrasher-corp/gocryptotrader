package itbit

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
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
func (i *ItBit) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	i.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = i.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = i.BaseCurrencies

	err := i.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if i.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = i.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the defaults for the exchange
func (i *ItBit) SetDefaults() {
	i.Name = "ITBIT"
	i.Enabled = true
	i.Verbose = true
	i.API.CredentialsValidator.RequiresClientID = true
	i.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := i.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	i.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AccountInfo:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrder:       true,
				SubmitOrder:       true,
				DepositHistory:    true,
				WithdrawalHistory: true,
				UserTradeHistory:  true,
				CryptoDeposit:     true,
				TradeFee:          true,
				FiatWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.WithdrawCryptoViaWebsiteOnly |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
	}

	i.Requester, err = request.New(i.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	i.API.Endpoints = i.NewEndpoints()
	err = i.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: itbitAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup sets the exchange parameters from exchange config
func (i *ItBit) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		i.SetEnabled(false)
		return nil
	}
	return i.SetupDefaults(exch)
}

// Start starts the ItBit go routine
func (i *ItBit) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		i.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the ItBit wrapper
func (i *ItBit) Run(_ context.Context) {
	if i.Verbose {
		i.PrintEnabledPairs()
	}
}

// GetServerTime returns the current exchange server time.
func (i *ItBit) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (i *ItBit) FetchTradablePairs(_ context.Context, _ asset.Item) (currency.Pairs, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (i *ItBit) UpdateTradablePairs(_ context.Context, _ bool) error {
	return common.ErrFunctionNotSupported
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (i *ItBit) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (i *ItBit) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := i.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tick, err := i.GetTicker(ctx, fPair.String())
	if err != nil {
		return nil, err
	}

	err = ticker.ProcessTicker(&ticker.Price{
		Last:         tick.LastPrice,
		High:         tick.High24h,
		Low:          tick.Low24h,
		Bid:          tick.Bid,
		Ask:          tick.Ask,
		Volume:       tick.Volume24h,
		Open:         tick.OpenToday,
		Pair:         p,
		LastUpdated:  tick.ServertimeUTC,
		ExchangeName: i.Name,
		AssetType:    a})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(i.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (i *ItBit) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(i.Name, p, assetType)
	if err != nil {
		return i.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (i *ItBit) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(i.Name, p, assetType)
	if err != nil {
		return i.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (i *ItBit) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := i.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:         i.Name,
		Pair:             p,
		Asset:            assetType,
		PriceDuplication: true,
		VerifyOrderbook:  i.CanVerifyOrderbook,
	}
	fPair, err := i.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := i.GetOrderbook(ctx, fPair.String())
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		var price, amount float64
		price, err = strconv.ParseFloat(orderbookNew.Bids[x][0], 64)
		if err != nil {
			return book, err
		}
		amount, err = strconv.ParseFloat(orderbookNew.Bids[x][1], 64)
		if err != nil {
			return book, err
		}
		book.Bids[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}

	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		var price, amount float64
		price, err = strconv.ParseFloat(orderbookNew.Asks[x][0], 64)
		if err != nil {
			return book, err
		}
		amount, err = strconv.ParseFloat(orderbookNew.Asks[x][1], 64)
		if err != nil {
			return book, err
		}
		book.Asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(i.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (i *ItBit) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	info.Exchange = i.Name

	wallets, err := i.GetWallets(ctx, url.Values{})
	if err != nil {
		return info, err
	}

	var amounts = make(map[string]*account.Balance)

	for x := range wallets {
		for _, cb := range wallets[x].Balances {
			if _, ok := amounts[cb.Currency]; !ok {
				amounts[cb.Currency] = &account.Balance{}
			}

			amounts[cb.Currency].Total += cb.TotalBalance
			amounts[cb.Currency].Hold += cb.TotalBalance - cb.AvailableBalance
			amounts[cb.Currency].Free += cb.AvailableBalance
		}
	}

	fullBalance := make([]account.Balance, 0, len(amounts))
	for key := range amounts {
		fullBalance = append(fullBalance, account.Balance{
			Currency: currency.NewCode(key),
			Total:    amounts[key].Total,
			Hold:     amounts[key].Hold,
			Free:     amounts[key].Free,
		})
	}

	info.Accounts = append(info.Accounts, account.SubAccount{
		AssetType:  assetType,
		Currencies: fullBalance,
	})

	creds, err := i.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&info, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (i *ItBit) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := i.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(i.Name, creds, assetType)
	if err != nil {
		return i.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (i *ItBit) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (i *ItBit) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (i *ItBit) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = i.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData Trades
	tradeData, err = i.GetTradeHistory(ctx, p.String(), "")
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData.RecentTrades))
	for x := range tradeData.RecentTrades {
		resp[x] = trade.Data{
			Exchange:     i.Name,
			TID:          tradeData.RecentTrades[x].MatchNumber,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData.RecentTrades[x].Price,
			Amount:       tradeData.RecentTrades[x].Amount,
			Timestamp:    tradeData.RecentTrades[x].Timestamp,
		}
	}

	err = i.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (i *ItBit) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	// cannot do time based retrieval of trade data
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (i *ItBit) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	var wallet string
	wallets, err := i.GetWallets(ctx, url.Values{})
	if err != nil {
		return nil, err
	}

	// Determine what wallet ID to use if there is any actual available currency to make the trade!
	for i := range wallets {
		for j := range wallets[i].Balances {
			if wallets[i].Balances[j].Currency == s.Pair.Base.String() &&
				wallets[i].Balances[j].AvailableBalance >= s.Amount {
				wallet = wallets[i].ID
			}
		}
	}

	if wallet == "" {
		return nil,
			fmt.Errorf("no wallet found with currency: %s with amount >= %v",
				s.Pair.Base,
				s.Amount)
	}

	fPair, err := i.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	response, err := i.PlaceOrder(ctx,
		wallet,
		s.Side.String(),
		s.Type.String(),
		fPair.Base.String(),
		s.Amount,
		s.Price,
		fPair.String(),
		"")
	if err != nil {
		return nil, err
	}
	subResp, err := s.DeriveSubmitResponse(response.ID)
	if err != nil {
		return nil, err
	}
	if response.AmountFilled == s.Amount {
		subResp.Status = order.Filled
	}
	return subResp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (i *ItBit) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (i *ItBit) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	return i.CancelExistingOrder(ctx, o.WalletAddress, o.OrderID)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (i *ItBit) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (i *ItBit) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := i.GetOrders(ctx,
		orderCancellation.WalletAddress,
		"",
		"open",
		0,
		0)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for j := range openOrders {
		err = i.CancelExistingOrder(ctx,
			orderCancellation.WalletAddress,
			openOrders[j].ID)
		if err != nil {
			cancelAllOrdersResponse.Status[openOrders[j].ID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (i *ItBit) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
// NOTE: This has not been implemented due to the fact you need to generate a
// specific wallet ID, and they restrict the amount of deposit addresses you can
// request limiting them to 2.
func (i *ItBit) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (i *ItBit) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (i *ItBit) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !i.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return i.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (i *ItBit) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	wallets, err := i.GetWallets(ctx, url.Values{})
	if err != nil {
		return nil, err
	}

	var allOrders []Order
	for x := range wallets {
		var resp []Order
		resp, err = i.GetOrders(ctx, wallets[x].ID, "", "open", 0, 0)
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := i.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, 0, len(allOrders))
	for j := range allOrders {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(allOrders[j].Instrument,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(allOrders[j].Side)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", i.Name, err)
		}
		var orderDate time.Time
		orderDate, err = time.Parse(time.RFC3339, allOrders[j].CreatedTime)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				i.Name,
				"GetActiveOrders",
				allOrders[j].ID,
				allOrders[j].CreatedTime)
		}

		orders = append(orders, order.Detail{
			OrderID:         allOrders[j].ID,
			Side:            side,
			Amount:          allOrders[j].Amount,
			ExecutedAmount:  allOrders[j].AmountFilled,
			RemainingAmount: allOrders[j].Amount - allOrders[j].AmountFilled,
			Exchange:        i.Name,
			Date:            orderDate,
			Pair:            symbol,
		})
	}
	return req.Filter(i.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (i *ItBit) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	wallets, err := i.GetWallets(ctx, url.Values{})
	if err != nil {
		return nil, err
	}

	var allOrders []Order
	for x := range wallets {
		var resp []Order
		resp, err = i.GetOrders(ctx, wallets[x].ID, "", "", 0, 0)
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := i.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, 0, len(allOrders))
	for j := range allOrders {
		if allOrders[j].Type == "open" {
			continue
		}
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(allOrders[j].Instrument,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(allOrders[j].Side)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", i.Name, err)
		}
		var status order.Status
		status, err = order.StringToOrderStatus(allOrders[j].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", i.Name, err)
		}
		var orderDate time.Time
		orderDate, err = time.Parse(time.RFC3339, allOrders[j].CreatedTime)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				i.Name,
				"GetActiveOrders",
				allOrders[j].ID,
				allOrders[j].CreatedTime)
		}

		detail := order.Detail{
			OrderID:              allOrders[j].ID,
			Side:                 side,
			Status:               status,
			Amount:               allOrders[j].Amount,
			ExecutedAmount:       allOrders[j].AmountFilled,
			RemainingAmount:      allOrders[j].Amount - allOrders[j].AmountFilled,
			Price:                allOrders[j].Price,
			AverageExecutedPrice: allOrders[j].VolumeWeightedAveragePrice,
			Exchange:             i.Name,
			Date:                 orderDate,
			Pair:                 symbol,
		}
		detail.InferCostsAndTimes()
		orders = append(orders, detail)
	}
	return req.Filter(i.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (i *ItBit) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := i.UpdateAccountInfo(ctx, assetType)
	return i.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (i *ItBit) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (i *ItBit) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}
