package localbitcoins

import (
	"errors"
	"fmt"
	"math"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
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

	l.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
		},

		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
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

	l.API.Endpoints.URLDefault = localbitcoinsAPIURL
	l.API.Endpoints.URL = l.API.Endpoints.URLDefault
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
	return l.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *LocalBitcoins) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	tick, err := l.GetTicker()
	if err != nil {
		return tickerPrice, err
	}

	pairs := l.GetEnabledPairs(assetType)
	for i := range pairs {
		curr := pairs[i].Quote.String()
		if _, ok := tick[curr]; !ok {
			continue
		}
		var tp ticker.Price
		tp.Pair = pairs[i]
		tp.Last = tick[curr].Avg24h
		tp.Volume = tick[curr].VolumeBTC

		err = ticker.ProcessTicker(l.Name, &tp, assetType)
		if err != nil {
			log.Error(log.Ticker, err)
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
	orderBook := new(orderbook.Base)
	orderbookNew, err := l.GetOrderbook(p.Quote.String())
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount / orderbookNew.Bids[x].Price,
			Price:  orderbookNew.Bids[x].Price,
		})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount / orderbookNew.Asks[x].Price,
			Price:  orderbookNew.Asks[x].Price,
		})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = l.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(l.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// LocalBitcoins exchange
func (l *LocalBitcoins) UpdateAccountInfo() (account.Holdings, error) {
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
func (l *LocalBitcoins) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(l.Name)
	if err != nil {
		return l.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *LocalBitcoins) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *LocalBitcoins) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (l *LocalBitcoins) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
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
		Currency:                   s.Pair.Quote.String(),
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
	err := l.CreateAd(&params)
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
func (l *LocalBitcoins) CancelOrder(order *order.Cancel) error {
	return l.DeleteAd(order.ID)
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

// GetOrderInfo returns information on a current open order
func (l *LocalBitcoins) GetOrderInfo(orderID string) (order.Detail, error) {
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
func (l *LocalBitcoins) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (l *LocalBitcoins) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *LocalBitcoins) GetWebsocket() (*wshandler.Websocket, error) {
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
	resp, err := l.GetDashboardInfo()
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
				l.GetPairFormat(asset.Spot, false).Delimiter),
			Exchange: l.Name,
		})
	}

	order.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (l *LocalBitcoins) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
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
				l.GetPairFormat(asset.Spot, false).Delimiter),
			Exchange: l.Name,
		})
	}

	order.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks,
		getOrdersRequest.EndTicks)
	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)

	return orders, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (l *LocalBitcoins) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (l *LocalBitcoins) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrFunctionNotSupported
}

// GetSubscriptions returns a copied list of subscriptions
func (l *LocalBitcoins) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrFunctionNotSupported
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (l *LocalBitcoins) AuthenticateWebsocket() error {
	return common.ErrFunctionNotSupported
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (l *LocalBitcoins) ValidateCredentials() error {
	_, err := l.UpdateAccountInfo()
	return l.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (l *LocalBitcoins) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
