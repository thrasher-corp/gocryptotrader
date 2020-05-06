package lbank

import (
	"fmt"
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
func (l *Lbank) GetDefaultConfig() (*config.ExchangeConfig, error) {
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

// SetDefaults sets the basic defaults for Lbank
func (l *Lbank) SetDefaults() {
	l.Name = "Lbank"
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
			Delimiter: "_",
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "_",
		},
	}

	l.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST: true,
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
				SubmitOrder:         true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	l.Requester = request.New(l.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	l.API.Endpoints.URLDefault = lbankAPIURL
	l.API.Endpoints.URL = l.API.Endpoints.URLDefault
}

// Setup sets exchange configuration profile
func (l *Lbank) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		l.SetEnabled(false)
		return nil
	}

	err := l.SetupDefaults(exch)
	if err != nil {
		return err
	}

	if l.API.AuthenticatedSupport {
		err = l.loadPrivKey()
		if err != nil {
			l.API.AuthenticatedSupport = false
			log.Errorf(log.ExchangeSys, "%s couldn't load private key, setting authenticated support to false", l.Name)
		}
	}
	return nil
}

// Start starts the LakeBTC go routine
func (l *Lbank) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		l.Run()
		wg.Done()
	}()
}

// Run implements the Lbank wrapper
func (l *Lbank) Run() {
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
func (l *Lbank) FetchTradablePairs(asset asset.Item) ([]string, error) {
	currencies, err := l.GetCurrencyPairs()
	if err != nil {
		return nil, err
	}
	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (l *Lbank) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := l.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	return l.UpdatePairs(currency.NewPairsFromStrings(pairs), asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (l *Lbank) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	tickerInfo, err := l.GetTickers()
	if err != nil {
		return tickerPrice, err
	}
	pairs := l.GetEnabledPairs(assetType)
	for i := range pairs {
		for j := range tickerInfo {
			if !pairs[i].Equal(tickerInfo[j].Symbol) {
				continue
			}
			tickerPrice = &ticker.Price{
				Last:        tickerInfo[j].Ticker.Latest,
				High:        tickerInfo[j].Ticker.High,
				Low:         tickerInfo[j].Ticker.Low,
				Volume:      tickerInfo[j].Ticker.Volume,
				Pair:        tickerInfo[j].Symbol,
				LastUpdated: time.Unix(0, tickerInfo[j].Timestamp),
			}
			err = ticker.ProcessTicker(l.Name, tickerPrice, assetType)
			if err != nil {
				log.Error(log.Ticker, err)
			}
		}
	}
	return ticker.GetTicker(l.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (l *Lbank) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(l.Name,
		l.FormatExchangeCurrency(p, assetType), assetType)
	if err != nil {
		return l.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (l *Lbank) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(l.Name, currency, assetType)
	if err != nil {
		return l.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (l *Lbank) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	a, err := l.GetMarketDepths(l.FormatExchangeCurrency(p, assetType).String(), "60", "1")
	if err != nil {
		return orderBook, err
	}
	for i := range a.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Price:  a.Asks[i][0],
			Amount: a.Asks[i][1]})
	}
	for i := range a.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Price:  a.Bids[i][0],
			Amount: a.Bids[i][1]})
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
// Lbank exchange
func (l *Lbank) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
	data, err := l.GetUserInfo()
	if err != nil {
		return info, err
	}
	var acc account.SubAccount
	for key, val := range data.Info.Asset {
		c := currency.NewCode(key)
		hold, ok := data.Info.Freeze[key]
		if !ok {
			return info, fmt.Errorf("hold data not found with %s", key)
		}
		totalVal, parseErr := strconv.ParseFloat(val, 64)
		if parseErr != nil {
			return info, parseErr
		}
		totalHold, parseErr := strconv.ParseFloat(hold, 64)
		if parseErr != nil {
			return info, parseErr
		}
		acc.Currencies = append(acc.Currencies, account.Balance{
			CurrencyName: c,
			TotalValue:   totalVal,
			Hold:         totalHold})
	}

	info.Accounts = append(info.Accounts, acc)
	info.Exchange = l.Name

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (l *Lbank) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(l.Name)
	if err != nil {
		return l.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (l *Lbank) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (l *Lbank) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (l *Lbank) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	if s.Side != order.Buy && s.Side != order.Sell {
		return resp,
			fmt.Errorf("%s order side is not supported by the exchange",
				s.Side)
	}
	tempResp, err := l.CreateOrder(
		l.FormatExchangeCurrency(s.Pair, asset.Spot).String(),
		s.Side.String(),
		s.Amount,
		s.Price)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = tempResp.OrderID
	if s.Type == order.Market {
		resp.FullyMatched = true
	}
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (l *Lbank) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (l *Lbank) CancelOrder(order *order.Cancel) error {
	_, err := l.RemoveOrder(l.FormatExchangeCurrency(order.Pair,
		order.AssetType).String(), order.ID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (l *Lbank) CancelAllOrders(orders *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	orderIDs, err := l.getAllOpenOrderID()
	if err != nil {
		return resp, nil
	}

	for key := range orderIDs {
		if key != orders.Pair.String() {
			continue
		}
		var x, y = 0, 0
		var input string
		var tempSlice []string
		for x <= len(orderIDs[key]) {
			x++
			for y != x {
				tempSlice = append(tempSlice, orderIDs[key][y])
				if y%3 == 0 {
					input = strings.Join(tempSlice, ",")
					CancelResponse, err2 := l.RemoveOrder(key, input)
					if err2 != nil {
						return resp, err2
					}
					tempStringSuccess := strings.Split(CancelResponse.Success, ",")
					for k := range tempStringSuccess {
						resp.Status[tempStringSuccess[k]] = "Cancelled"
					}
					tempStringError := strings.Split(CancelResponse.Err, ",")
					for l := range tempStringError {
						resp.Status[tempStringError[l]] = "Failed"
					}
					tempSlice = tempSlice[:0]
					y++
				}
				y++
			}
			input = strings.Join(tempSlice, ",")
			CancelResponse, err2 := l.RemoveOrder(key, input)
			if err2 != nil {
				return resp, err2
			}
			tempStringSuccess := strings.Split(CancelResponse.Success, ",")
			for k := range tempStringSuccess {
				resp.Status[tempStringSuccess[k]] = "Cancelled"
			}
			tempStringError := strings.Split(CancelResponse.Err, ",")
			for l := range tempStringError {
				resp.Status[tempStringError[l]] = "Failed"
			}
			tempSlice = tempSlice[:0]
		}
	}
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (l *Lbank) GetOrderInfo(orderID string) (order.Detail, error) {
	var resp order.Detail
	orderIDs, err := l.getAllOpenOrderID()
	if err != nil {
		return resp, err
	}

	for key, val := range orderIDs {
		for i := range val {
			if val[i] != orderID {
				continue
			}
			tempResp, err := l.QueryOrder(key, orderID)
			if err != nil {
				return resp, err
			}
			resp.Exchange = l.Name
			resp.Pair = currency.NewPairFromString(key)
			if strings.EqualFold(tempResp.Orders[0].Type, order.Buy.String()) {
				resp.Side = order.Buy
			} else {
				resp.Side = order.Sell
			}
			z := tempResp.Orders[0].Status
			switch {
			case z == -1:
				resp.Status = "cancelled"
			case z == 0:
				resp.Status = "on trading"
			case z == 1:
				resp.Status = "filled partially"
			case z == 2:
				resp.Status = "Filled totally"
			case z == 4:
				resp.Status = "Cancelling"
			default:
				resp.Status = "Invalid Order Status"
			}
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(&exchange.FeeBuilder{
				FeeType:       exchange.CryptocurrencyTradeFee,
				Amount:        tempResp.Orders[0].Amount,
				PurchasePrice: tempResp.Orders[0].Price})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
		}
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (l *Lbank) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	resp, err := l.Withdraw(withdrawRequest.Crypto.Address, withdrawRequest.Currency.String(),
		strconv.FormatFloat(withdrawRequest.Amount, 'f', -1, 64), "",
		withdrawRequest.Description, "")
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.WithdrawID,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (l *Lbank) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (l *Lbank) GetWebsocket() (*wshandler.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (l *Lbank) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var finalResp []order.Detail
	var resp order.Detail
	tempData, err := l.getAllOpenOrderID()
	if err != nil {
		return finalResp, err
	}

	for key, val := range tempData {
		for x := range val {
			tempResp, err := l.QueryOrder(key, val[x])
			if err != nil {
				return finalResp, err
			}
			resp.Exchange = l.Name
			resp.Pair = currency.NewPairFromString(key)
			if strings.EqualFold(tempResp.Orders[0].Type, order.Buy.String()) {
				resp.Side = order.Buy
			} else {
				resp.Side = order.Sell
			}
			z := tempResp.Orders[0].Status
			switch {
			case z == -1:
				resp.Status = "cancelled"
			case z == 1:
				resp.Status = "on trading"
			case z == 2:
				resp.Status = "filled partially"
			case z == 3:
				resp.Status = "Filled totally"
			case z == 4:
				resp.Status = "Cancelling"
			default:
				resp.Status = "Invalid Order Status"
			}
			resp.Price = tempResp.Orders[0].Price
			resp.Amount = tempResp.Orders[0].Amount
			resp.Date = time.Unix(tempResp.Orders[0].CreateTime, 9)
			resp.ExecutedAmount = tempResp.Orders[0].DealAmount
			resp.RemainingAmount = tempResp.Orders[0].Amount - tempResp.Orders[0].DealAmount
			resp.Fee, err = l.GetFeeByType(&exchange.FeeBuilder{
				FeeType:       exchange.CryptocurrencyTradeFee,
				Amount:        tempResp.Orders[0].Amount,
				PurchasePrice: tempResp.Orders[0].Price})
			if err != nil {
				resp.Fee = lbankFeeNotFound
			}
			for y := int(0); y < len(getOrdersRequest.Pairs); y++ {
				if getOrdersRequest.Pairs[y].String() != key {
					continue
				}
				if getOrdersRequest.Side == "ANY" {
					finalResp = append(finalResp, resp)
					continue
				}
				if strings.EqualFold(getOrdersRequest.Side.String(),
					tempResp.Orders[0].Type) {
					finalResp = append(finalResp, resp)
				}
			}
		}
	}
	return finalResp, nil
}

// GetOrderHistory retrieves account order information *
// Can Limit response to specific order status
func (l *Lbank) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var finalResp []order.Detail
	var resp order.Detail
	var tempCurr currency.Pairs
	if len(getOrdersRequest.Pairs) == 0 {
		tempCurr = l.GetEnabledPairs(asset.Spot)
	} else {
		tempCurr = getOrdersRequest.Pairs
	}
	for a := range tempCurr {
		p := l.FormatExchangeCurrency(tempCurr[a], asset.Spot).String()
		b := int64(1)
		tempResp, err := l.QueryOrderHistory(p, strconv.FormatInt(b, 10), "200")
		if err != nil {
			return finalResp, err
		}
		for len(tempResp.Orders) != 0 {
			tempResp, err = l.QueryOrderHistory(p, strconv.FormatInt(b, 10), "200")
			if err != nil {
				return finalResp, err
			}
			for x := 0; x < len(tempResp.Orders); x++ {
				resp.Exchange = l.Name
				resp.Pair = currency.NewPairFromString(tempResp.Orders[x].Symbol)
				if strings.EqualFold(tempResp.Orders[x].Type, order.Buy.String()) {
					resp.Side = order.Buy
				} else {
					resp.Side = order.Sell
				}
				z := tempResp.Orders[x].Status
				switch {
				case z == -1:
					resp.Status = "cancelled"
				case z == 1:
					resp.Status = "on trading"
				case z == 2:
					resp.Status = "filled partially"
				case z == 3:
					resp.Status = "Filled totally"
				case z == 4:
					resp.Status = "Cancelling"
				default:
					resp.Status = "Invalid Order Status"
				}
				resp.Price = tempResp.Orders[x].Price
				resp.Amount = tempResp.Orders[x].Amount
				resp.Date = time.Unix(tempResp.Orders[x].CreateTime, 9)
				resp.ExecutedAmount = tempResp.Orders[x].DealAmount
				resp.RemainingAmount = tempResp.Orders[x].Price - tempResp.Orders[x].DealAmount
				resp.Fee, err = l.GetFeeByType(&exchange.FeeBuilder{
					FeeType:       exchange.CryptocurrencyTradeFee,
					Amount:        tempResp.Orders[x].Amount,
					PurchasePrice: tempResp.Orders[x].Price})
				if err != nil {
					resp.Fee = lbankFeeNotFound
				}
				finalResp = append(finalResp, resp)
				b++
			}
		}
	}
	return finalResp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction *
func (l *Lbank) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var resp float64
	if feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.002, nil
	}
	if feeBuilder.FeeType == exchange.CryptocurrencyWithdrawalFee {
		withdrawalFee, err := l.GetWithdrawConfig(feeBuilder.Pair.Base.Lower().String())
		if err != nil {
			return resp, err
		}
		var tempFee string
		temp := strings.Split(withdrawalFee[0].Fee, ":\"")
		if len(temp) > 1 {
			tempFee = strings.TrimRight(temp[1], ",\"type")
		} else {
			tempFee = temp[0]
		}
		resp, err = strconv.ParseFloat(tempFee, 64)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// GetAllOpenOrderID returns all open orders by currency pairs
func (l *Lbank) getAllOpenOrderID() (map[string][]string, error) {
	allPairs := l.GetEnabledPairs(asset.Spot)
	resp := make(map[string][]string)
	for a := range allPairs {
		p := l.FormatExchangeCurrency(allPairs[a], asset.Spot).String()
		b := int64(1)
		tempResp, err := l.GetOpenOrders(p, strconv.FormatInt(b, 10), "200")
		if err != nil {
			return resp, err
		}
		tempData := len(tempResp.Orders)
		for tempData != 0 {
			tempResp, err = l.GetOpenOrders(p, strconv.FormatInt(b, 10), "200")
			if err != nil {
				return resp, err
			}

			if len(tempResp.Orders) == 0 {
				return resp, nil
			}

			for c := 0; c < tempData; c++ {
				resp[p] = append(resp[p], tempResp.Orders[c].OrderID)
			}
			tempData = len(tempResp.Orders)
			b++
		}
	}
	return resp, nil
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (l *Lbank) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrNotYetImplemented
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (l *Lbank) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	return common.ErrNotYetImplemented
}

// AuthenticateWebsocket authenticates it
func (l *Lbank) AuthenticateWebsocket() error {
	return common.ErrNotYetImplemented
}

// GetSubscriptions gets subscriptions
func (l *Lbank) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return nil, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (l *Lbank) ValidateCredentials() error {
	_, err := l.UpdateAccountInfo()
	return l.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (l *Lbank) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
