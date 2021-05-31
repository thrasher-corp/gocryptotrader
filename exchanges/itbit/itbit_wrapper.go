package itbit

import (
	"fmt"
	"net/url"
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
func (i *ItBit) GetDefaultConfig() (*config.ExchangeConfig, error) {
	i.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = i.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = i.BaseCurrencies

	err := i.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if i.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = i.UpdateTradablePairs(true)
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
	configFmt := &currency.PairFormat{Uppercase: true}
	err := i.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	i.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
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
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: false,
		},
	}

	i.Requester = request.New(i.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	i.API.Endpoints = i.NewEndpoints()
	err = i.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: itbitAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup sets the exchange parameters from exchange config
func (i *ItBit) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		i.SetEnabled(false)
		return nil
	}
	return i.SetupDefaults(exch)
}

// Start starts the ItBit go routine
func (i *ItBit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		i.Run()
		wg.Done()
	}()
}

// Run implements the ItBit wrapper
func (i *ItBit) Run() {
	if i.Verbose {
		i.PrintEnabledPairs()
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (i *ItBit) FetchTradablePairs(asset asset.Item) ([]string, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (i *ItBit) UpdateTradablePairs(forceUpdate bool) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (i *ItBit) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fpair, err := i.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tick, err := i.GetTicker(fpair.String())
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
		AssetType:    assetType})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(i.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (i *ItBit) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(i.Name, p, assetType)
	if err != nil {
		return i.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (i *ItBit) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(i.Name, p, assetType)
	if err != nil {
		return i.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (i *ItBit) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:         i.Name,
		Pair:             p,
		Asset:            assetType,
		PriceDuplication: true,
		VerifyOrderbook:  i.CanVerifyOrderbook,
	}
	fpair, err := i.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := i.GetOrderbook(fpair.String())
	if err != nil {
		return nil, err
	}

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
		book.Bids = append(book.Bids,
			orderbook.Item{
				Amount: amount,
				Price:  price,
			})
	}

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
		book.Asks = append(book.Asks,
			orderbook.Item{
				Amount: amount,
				Price:  price,
			})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(i.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (i *ItBit) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	info.Exchange = i.Name

	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return info, err
	}

	type balance struct {
		TotalValue float64
		Hold       float64
	}

	var amounts = make(map[string]*balance)

	for x := range wallets {
		for _, cb := range wallets[x].Balances {
			if _, ok := amounts[cb.Currency]; !ok {
				amounts[cb.Currency] = &balance{}
			}

			amounts[cb.Currency].TotalValue += cb.TotalBalance
			amounts[cb.Currency].Hold += cb.TotalBalance - cb.AvailableBalance
		}
	}

	var fullBalance []account.Balance
	for key := range amounts {
		fullBalance = append(fullBalance, account.Balance{
			CurrencyName: currency.NewCode(key),
			TotalValue:   amounts[key].TotalValue,
			Hold:         amounts[key].Hold,
		})
	}

	info.Accounts = append(info.Accounts, account.SubAccount{
		Currencies: fullBalance,
	})

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (i *ItBit) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(i.Name, assetType)
	if err != nil {
		return i.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (i *ItBit) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (i *ItBit) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (i *ItBit) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = i.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData Trades
	tradeData, err = i.GetTradeHistory(p.String(), "")
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for x := range tradeData.RecentTrades {
		resp = append(resp, trade.Data{
			Exchange:     i.Name,
			TID:          tradeData.RecentTrades[x].MatchNumber,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData.RecentTrades[x].Price,
			Amount:       tradeData.RecentTrades[x].Amount,
			Timestamp:    tradeData.RecentTrades[x].Timestamp,
		})
	}

	err = i.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (i *ItBit) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	// cannot do time based retrieval of trade data
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (i *ItBit) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var wallet string
	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return submitOrderResponse, err
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
		return submitOrderResponse,
			fmt.Errorf("no wallet found with currency: %s with amount >= %v",
				s.Pair.Base,
				s.Amount)
	}

	fPair, err := i.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	response, err := i.PlaceOrder(wallet,
		s.Side.String(),
		s.Type.String(),
		fPair.Base.String(),
		s.Amount,
		s.Price,
		fPair.String(),
		"")
	if err != nil {
		return submitOrderResponse, err
	}
	if response.ID != "" {
		submitOrderResponse.OrderID = response.ID
	}

	if response.AmountFilled == s.Amount {
		submitOrderResponse.FullyMatched = true
	}
	submitOrderResponse.IsOrderPlaced = true
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (i *ItBit) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (i *ItBit) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	return i.CancelExistingOrder(o.WalletAddress, o.ID)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (i *ItBit) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (i *ItBit) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := i.GetOrders(orderCancellation.WalletAddress, "", "open", 0, 0)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for j := range openOrders {
		err = i.CancelExistingOrder(orderCancellation.WalletAddress, openOrders[j].ID)
		if err != nil {
			cancelAllOrdersResponse.Status[openOrders[j].ID] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (i *ItBit) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
// NOTE: This has not been implemented due to the fact you need to generate a
// a specific wallet ID and they restrict the amount of deposit address you can
// request limiting them to 2.
func (i *ItBit) GetDepositAddress(_ currency.Code, _ string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (i *ItBit) WithdrawCryptocurrencyFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (i *ItBit) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (i *ItBit) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !i.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return i.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (i *ItBit) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return nil, err
	}

	var allOrders []Order
	for x := range wallets {
		var resp []Order
		resp, err = i.GetOrders(wallets[x].ID, "", "open", 0, 0)
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := i.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for j := range allOrders {
		var symbol currency.Pair
		symbol, err := currency.NewPairDelimiter(allOrders[j].Instrument,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		side := order.Side(strings.ToUpper(allOrders[j].Side))
		orderDate, err := time.Parse(time.RFC3339, allOrders[j].CreatedTime)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				i.Name,
				"GetActiveOrders",
				allOrders[j].ID,
				allOrders[j].CreatedTime)
		}

		orders = append(orders, order.Detail{
			ID:              allOrders[j].ID,
			Side:            side,
			Amount:          allOrders[j].Amount,
			ExecutedAmount:  allOrders[j].AmountFilled,
			RemainingAmount: (allOrders[j].Amount - allOrders[j].AmountFilled),
			Exchange:        i.Name,
			Date:            orderDate,
			Pair:            symbol,
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (i *ItBit) GetOrderHistory(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	wallets, err := i.GetWallets(url.Values{})
	if err != nil {
		return nil, err
	}

	var allOrders []Order
	for x := range wallets {
		var resp []Order
		resp, err = i.GetOrders(wallets[x].ID, "", "", 0, 0)
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := i.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
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

		side := order.Side(strings.ToUpper(allOrders[j].Side))
		orderDate, err := time.Parse(time.RFC3339, allOrders[j].CreatedTime)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				i.Name,
				"GetActiveOrders",
				allOrders[j].ID,
				allOrders[j].CreatedTime)
		}

		orders = append(orders, order.Detail{
			ID:              allOrders[j].ID,
			Side:            side,
			Amount:          allOrders[j].Amount,
			ExecutedAmount:  allOrders[j].AmountFilled,
			RemainingAmount: (allOrders[j].Amount - allOrders[j].AmountFilled),
			Exchange:        i.Name,
			Date:            orderDate,
			Pair:            symbol,
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (i *ItBit) ValidateCredentials(assetType asset.Item) error {
	_, err := i.UpdateAccountInfo(assetType)
	return i.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (i *ItBit) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (i *ItBit) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
