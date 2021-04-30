package localbitcoins

import (
	"errors"
	"fmt"
	"math"
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
func (l *LocalBitcoins) GetDefaultConfig() (*config.ExchangeConfig, error) {
	l.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = l.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = l.BaseCurrencies

	err := l.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if l.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = l.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the package defaults for localbitcoins
func (l *LocalBitcoins) SetDefaults() {
	l.Name = "LocalBitcoins"
	l.Enabled = true
	l.Verbose = true
	l.API.CredentialsValidator.RequiresKey = true
	l.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Uppercase: true}
	err := l.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	l.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: protocol.Features{
				TickerBatching:    true,
				TickerFetching:    true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrder:          true,
				CancelOrder:       true,
				SubmitOrder:       true,
				DepositHistory:    true,
				WithdrawalHistory: true,
				UserTradeHistory:  true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.WithdrawFiatViaWebsiteOnly,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	l.Requester = request.New(l.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	l.API.Endpoints = l.NewEndpoints()
	err = l.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: localbitcoinsAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup sets exchange configuration parameters
func (l *LocalBitcoins) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		l.SetEnabled(false)
		return nil
	}
	return l.SetupDefaults(exch)
}

// Start starts the LocalBitcoins go routine
func (l *LocalBitcoins) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the LocalBitcoins wrapper
func (l *LocalBitcoins) Run() {
	if l.Verbose {
		l.PrintEnabledPairs()
	}

	if !l.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := l.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", l.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (l *LocalBitcoins) FetchTradablePairs(asset asset.Item) ([]string, error) {
	currencies, err := l.GetTradableCurrencies()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range currencies {
		pairs = append(pairs, "BTC"+currencies[x])
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (l *LocalBitcoins) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := l.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}
	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return l.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LocalBitcoins) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tick, err := l.GetTicker()
	if err != nil {
		return nil, err
	}

	pairs, err := l.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}
	for i := range pairs {
		curr := pairs[i].Quote.String()
		if _, ok := tick[curr]; !ok {
			continue
		}
		var tp ticker.Price
		tp.Pair = pairs[i]
		tp.Last = tick[curr].Avg24h
		tp.Volume = tick[curr].VolumeBTC
		tp.ExchangeName = l.Name
		tp.AssetType = assetType

		err = ticker.ProcessTicker(&tp)
		if err != nil {
			return nil, err
		}
	}

	return ticker.GetTicker(l.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (l *LocalBitcoins) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.Name, p, assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (l *LocalBitcoins) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(l.Name, p, assetType)
	if err != nil {
		return l.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *LocalBitcoins) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        l.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: l.CanVerifyOrderbook,
	}

	orderbookNew, err := l.GetOrderbook(p.Quote.String())
	if err != nil {
		return book, err
	}

	for x := range orderbookNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount / orderbookNew.Bids[x].Price,
			Price:  orderbookNew.Bids[x].Price,
		})
	}

	for x := range orderbookNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount / orderbookNew.Asks[x].Price,
			Price:  orderbookNew.Asks[x].Price,
		})
	}

	book.PriceDuplication = true
	err = book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(l.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// LocalBitcoins exchange
func (l *LocalBitcoins) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = l.Name
	accountBalance, err := l.GetWalletBalance()
	if err != nil {
		return response, err
	}
	var exchangeCurrency account.Balance
	exchangeCurrency.CurrencyName = currency.BTC
	exchangeCurrency.TotalValue = accountBalance.Total.Balance

	response.Accounts = append(response.Accounts, account.SubAccount{
		Currencies: []account.Balance{exchangeCurrency},
	})

	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (l *LocalBitcoins) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(l.Name, assetType)
	if err != nil {
		return l.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LocalBitcoins) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (l *LocalBitcoins) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (l *LocalBitcoins) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = l.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData []Trade
	tradeData, err = l.GetTrades(p.Quote.String(), nil)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData {
		resp = append(resp, trade.Data{
			Exchange:     l.Name,
			TID:          strconv.FormatInt(tradeData[i].TID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    time.Unix(tradeData[i].Date, 0),
		})
	}

	err = l.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (l *LocalBitcoins) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (l *LocalBitcoins) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	fPair, err := l.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	// These are placeholder details
	// TODO store a user's localbitcoin details to use here
	var params = AdCreate{
		PriceEquation:              "USD_in_AUD",
		Latitude:                   1,
		Longitude:                  1,
		City:                       "City",
		Location:                   "Location",
		CountryCode:                "US",
		Currency:                   fPair.Quote.String(),
		AccountInfo:                "-",
		BankName:                   "Bank",
		MSG:                        s.Side.String(),
		SMSVerficationRequired:     true,
		TrackMaxAmount:             true,
		RequireTrustedByAdvertiser: true,
		RequireIdentification:      true,
		OnlineProvider:             "",
		TradeType:                  "",
		MinAmount:                  int(math.Round(s.Amount)),
	}

	// Does not return any orderID, so create the add, then get the order
	err = l.CreateAd(&params)
	if err != nil {
		return submitOrderResponse, err
	}

	submitOrderResponse.IsOrderPlaced = true

	// Now to figure out what ad we just submitted
	// The only details we have are the params above
	var adID string
	ads, err := l.Getads()
	for i := range ads.AdList {
		if ads.AdList[i].Data.PriceEquation == params.PriceEquation &&
			ads.AdList[i].Data.Lat == float64(params.Latitude) &&
			ads.AdList[i].Data.Lon == float64(params.Longitude) &&
			ads.AdList[i].Data.City == params.City &&
			ads.AdList[i].Data.Location == params.Location &&
			ads.AdList[i].Data.CountryCode == params.CountryCode &&
			ads.AdList[i].Data.Currency == params.Currency &&
			ads.AdList[i].Data.AccountInfo == params.AccountInfo &&
			ads.AdList[i].Data.BankName == params.BankName &&
			ads.AdList[i].Data.SMSVerficationRequired == params.SMSVerficationRequired &&
			ads.AdList[i].Data.TrackMaxAmount == params.TrackMaxAmount &&
			ads.AdList[i].Data.RequireTrustedByAdvertiser == params.RequireTrustedByAdvertiser &&
			ads.AdList[i].Data.OnlineProvider == params.OnlineProvider &&
			ads.AdList[i].Data.TradeType == params.TradeType &&
			ads.AdList[i].Data.MinAmount == strconv.FormatInt(int64(params.MinAmount), 10) {
			adID = strconv.FormatInt(ads.AdList[i].Data.AdID, 10)
		}
	}

	if adID != "" {
		submitOrderResponse.OrderID = adID
	} else {
		return submitOrderResponse, errors.New("ad placed, but not found via API")
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *LocalBitcoins) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *LocalBitcoins) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	return l.DeleteAd(o.ID)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (l *LocalBitcoins) CancelBatchOrders(o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *LocalBitcoins) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	ads, err := l.Getads()
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range ads.AdList {
		adIDString := strconv.FormatInt(ads.AdList[i].Data.AdID, 10)
		err = l.DeleteAd(adIDString)
		if err != nil {
			cancelAllOrdersResponse.Status[adIDString] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (l *LocalBitcoins) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *LocalBitcoins) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	if !strings.EqualFold(currency.BTC.String(), cryptocurrency.String()) {
		return "", fmt.Errorf("%s does not have support for currency %s, it only supports bitcoin",
			l.Name, cryptocurrency)
	}

	return l.GetWalletAddress()
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *LocalBitcoins) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	err := l.WalletSend(withdrawRequest.Crypto.Address,
		withdrawRequest.Amount,
		withdrawRequest.PIN)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (l *LocalBitcoins) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if (!l.AllowAuthenticatedRequest() || l.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return l.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (l *LocalBitcoins) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}

	resp, err := l.GetDashboardInfo()
	if err != nil {
		return nil, err
	}

	format, err := l.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp {
		orderDate, err := time.Parse(time.RFC3339, resp[i].Data.CreatedAt)
		if err != nil {
			log.Errorf(log.ExchangeSys, "Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				l.Name,
				"GetActiveOrders",
				resp[i].Data.Advertisement.ID,
				resp[i].Data.CreatedAt)
		}

		var side order.Side
		if resp[i].Data.IsBuying {
			side = order.Buy
		} else if resp[i].Data.IsSelling {
			side = order.Sell
		}

		orders = append(orders, order.Detail{
			Amount: resp[i].Data.AmountBTC,
			Price:  resp[i].Data.Amount,
			ID:     strconv.FormatInt(int64(resp[i].Data.Advertisement.ID), 10),
			Date:   orderDate,
			Fee:    resp[i].Data.FeeBTC,
			Side:   side,
			Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
				resp[i].Data.Currency,
				format.Delimiter),
			Exchange: l.Name,
		})
	}

	order.FilterOrdersByTimeRange(&orders, getOrdersRequest.StartTime,
		getOrdersRequest.EndTime)
	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *LocalBitcoins) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}

	var allTrades []DashBoardInfo
	resp, err := l.GetDashboardCancelledTrades()
	if err != nil {
		return nil, err
	}
	allTrades = append(allTrades, resp...)

	resp, err = l.GetDashboardClosedTrades()
	if err != nil {
		return nil, err
	}
	allTrades = append(allTrades, resp...)

	resp, err = l.GetDashboardReleasedTrades()
	if err != nil {
		return nil, err
	}
	allTrades = append(allTrades, resp...)

	format, err := l.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range allTrades {
		orderDate, err := time.Parse(time.RFC3339, allTrades[i].Data.CreatedAt)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
				l.Name,
				"GetActiveOrders",
				allTrades[i].Data.Advertisement.ID,
				allTrades[i].Data.CreatedAt)
		}

		var side order.Side
		if allTrades[i].Data.IsBuying {
			side = order.Buy
		} else if allTrades[i].Data.IsSelling {
			side = order.Sell
		}

		status := ""

		switch {
		case allTrades[i].Data.ReleasedAt != "" &&
			allTrades[i].Data.ReleasedAt != null:
			status = "Released"
		case allTrades[i].Data.CanceledAt != "" &&
			allTrades[i].Data.CanceledAt != null:
			status = "Cancelled"
		case allTrades[i].Data.ClosedAt != "" &&
			allTrades[i].Data.ClosedAt != null:
			status = "Closed"
		}

		orders = append(orders, order.Detail{
			Amount: allTrades[i].Data.AmountBTC,
			Price:  allTrades[i].Data.Amount,
			ID:     strconv.FormatInt(int64(allTrades[i].Data.Advertisement.ID), 10),
			Date:   orderDate,
			Fee:    allTrades[i].Data.FeeBTC,
			Side:   side,
			Status: order.Status(status),
			Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
				allTrades[i].Data.Currency,
				format.Delimiter),
			Exchange: l.Name,
		})
	}

	order.FilterOrdersByTimeRange(&orders, getOrdersRequest.StartTime,
		getOrdersRequest.EndTime)
	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)

	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (l *LocalBitcoins) ValidateCredentials(assetType asset.Item) error {
	_, err := l.UpdateAccountInfo(assetType)
	return l.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (l *LocalBitcoins) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (l *LocalBitcoins) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
