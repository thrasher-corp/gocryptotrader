package ftx

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
func (f *FTX) GetDefaultConfig() (*config.ExchangeConfig, error) {
	f.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = f.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = f.BaseCurrencies

	err := f.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if f.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = f.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for FTX
func (f *FTX) SetDefaults() {
	f.Name = "FTX"
	f.Enabled = true
	f.Verbose = true
	f.API.CredentialsValidator.RequiresKey = true
	f.API.CredentialsValidator.RequiresSecret = true
	f.CurrencyPairs = currency.PairsManager{
		AssetTypes: asset.Items{
			asset.Spot,
			asset.Futures,
		},
	}
	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "/",
		},
	}
	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}
	f.CurrencyPairs.Store(asset.Spot, spot)
	f.CurrencyPairs.Store(asset.Futures, futures)
	f.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				OrderbookFetching: true,
				TradeFetching:     true,
				Subscribe:         true,
				Unsubscribe:       true,
				GetOrders:         true,
				GetOrder:          true,
			},
			WithdrawPermissions: exchange.NoAPIWithdrawalMethods,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	f.Requester = request.New(f.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(ratePeriod, rateLimit)))

	f.API.Endpoints.URLDefault = ftxAPIURL
	f.API.Endpoints.URL = f.API.Endpoints.URLDefault
	f.Websocket = wshandler.New()
	f.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	f.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	f.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (f *FTX) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		f.SetEnabled(false)
		return nil
	}

	err := f.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = f.Websocket.Setup(
		&wshandler.WebsocketSetup{
			Enabled:                          exch.Features.Enabled.Websocket,
			Verbose:                          exch.Verbose,
			AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
			WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
			DefaultURL:                       ftxWSURL,
			ExchangeName:                     exch.Name,
			RunningURL:                       exch.API.Endpoints.WebsocketURL,
			Connector:                        f.WsConnect,
			Subscriber:                       f.Subscribe,
			UnSubscriber:                     f.Unsubscribe,
			Features:                         &f.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}

	f.WebsocketConn = &wshandler.WebsocketConnection{
		ExchangeName:         f.Name,
		URL:                  f.Websocket.GetWebsocketURL(),
		ProxyURL:             f.Websocket.GetProxyAddress(),
		Verbose:              f.Verbose,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	}

	f.Websocket.Orderbook.Setup(
		exch.WebsocketOrderbookBufferLimit,
		false,
		false,
		false,
		false,
		exch.Name)
	return nil
}

// Start starts the FTX go routine
func (f *FTX) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		f.Run()
		wg.Done()
	}()
}

// Run implements the FTX wrapper
func (f *FTX) Run() {
	if f.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			f.Name,
			common.IsEnabled(f.Websocket.IsEnabled()))
		f.PrintEnabledPairs()
	}

	if !f.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := f.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			f.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (f *FTX) FetchTradablePairs(a asset.Item) ([]string, error) {
	if !f.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, f.Name)
	}
	markets, err := f.GetMarkets()
	if err != nil {
		return nil, err
	}
	var pairs []string
	switch a {
	case asset.Spot:
		for x := range markets.Result {
			if markets.Result[x].MarketType == spotString {
				pairs = append(pairs, markets.Result[x].Name)
			}
		}
	case asset.Futures:
		for x := range markets.Result {
			if markets.Result[x].MarketType == futuresString {
				pairs = append(pairs, markets.Result[x].Name)
			}
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (f *FTX) UpdateTradablePairs(forceUpdate bool) error {
	for x := range f.CurrencyPairs.AssetTypes {
		pairs, err := f.FetchTradablePairs(f.CurrencyPairs.AssetTypes[x])
		if err != nil {
			return err
		}
		err = f.UpdatePairs(currency.NewPairsFromStrings(pairs),
			f.CurrencyPairs.AssetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (f *FTX) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	f.Verbose = true
	var marketNames []string
	allPairs := f.GetEnabledPairs(assetType)
	for a := range allPairs {
		marketNames = append(marketNames, f.FormatExchangeCurrency(allPairs[a], assetType).String())
	}
	markets, err := f.GetMarkets()
	if err != nil {
		return nil, err
	}
	for x := range markets.Result {
		marketName := currency.NewPairFromString(markets.Result[x].Name)
		if !common.StringDataCompareInsensitive(marketNames, marketName.String()) {
			continue
		}
		var resp ticker.Price
		resp.Pair = marketName
		resp.Last = markets.Result[x].Last
		resp.Bid = markets.Result[x].Bid
		resp.Ask = markets.Result[x].Ask
		resp.LastUpdated = time.Now()
		err = ticker.ProcessTicker(f.Name, &resp, assetType)
		if err != nil {
			return nil, err
		}
	}
	return ticker.GetTicker(f.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (f *FTX) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(f.Name, p, assetType)
	if err != nil {
		return f.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (f *FTX) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(f.Name, currency, assetType)
	if err != nil {
		return f.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (f *FTX) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderBook := new(orderbook.Base)
	tempResp, err := f.GetOrderbook(f.FormatExchangeCurrency(p, assetType).String(), 0)
	if err != nil {
		return orderBook, err
	}
	for x := range tempResp.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Amount: tempResp.Bids[x].Size,
			Price:  tempResp.Bids[x].Price})
	}
	for y := range tempResp.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Amount: tempResp.Asks[y].Size,
			Price:  tempResp.Asks[y].Price})
	}
	orderBook.Pair = p
	orderBook.ExchangeName = f.Name
	orderBook.AssetType = assetType
	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}
	return orderbook.Get(f.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (f *FTX) UpdateAccountInfo() (account.Holdings, error) {
	var resp account.Holdings
	data, err := f.GetBalances()
	if err != nil {
		return resp, err
	}
	var acc account.SubAccount
	for i := range data.Result {
		c := currency.NewCode(data.Result[i].Coin)
		hold := data.Result[i].Total - data.Result[i].Free
		total := data.Result[i].Total
		acc.Currencies = append(acc.Currencies,
			account.Balance{CurrencyName: c,
				TotalValue: total,
				Hold:       hold})
	}
	resp.Accounts = append(resp.Accounts, acc)
	resp.Exchange = f.Name

	err = account.Process(&resp)
	if err != nil {
		return account.Holdings{}, err
	}

	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (f *FTX) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(f.Name)
	if err != nil {
		return f.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (f *FTX) GetFundingHistory() ([]exchange.FundHistory, error) {
	var resp []exchange.FundHistory
	depositData, err := f.FetchDepositHistory()
	if err != nil {
		return resp, err
	}
	for x := range depositData.Result {
		var tempData exchange.FundHistory
		tempData.Fee = depositData.Result[x].Fee
		tempData.Timestamp = depositData.Result[x].Time
		tempData.ExchangeName = f.Name
		tempData.CryptoTxID = depositData.Result[x].TxID
		tempData.Status = depositData.Result[x].Status
		tempData.Amount = depositData.Result[x].Size
		tempData.Currency = depositData.Result[x].Coin
		tempData.TransferID = strconv.FormatInt(depositData.Result[x].ID, 10)
		resp = append(resp, tempData)
	}
	withdrawalData, err := f.FetchWithdrawalHistory()
	if err != nil {
		return resp, err
	}
	for y := range withdrawalData.Result {
		var tempData exchange.FundHistory
		tempData.Fee = depositData.Result[y].Fee
		tempData.Timestamp = depositData.Result[y].Time
		tempData.ExchangeName = f.Name
		tempData.CryptoTxID = depositData.Result[y].TxID
		tempData.Status = depositData.Result[y].Status
		tempData.Amount = depositData.Result[y].Size
		tempData.Currency = depositData.Result[y].Coin
		tempData.TransferID = strconv.FormatInt(depositData.Result[y].ID, 10)
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (f *FTX) GetExchangeHistory(p currency.Pair, assetType asset.Item) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (f *FTX) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	if s.Side == order.Sell {
		s.Side = order.Ask
	}
	if s.Side == order.Buy {
		s.Side = order.Bid
	}

	tempResp, err := f.Order(f.FormatExchangeCurrency(s.Pair, s.AssetType).String(),
		s.Side.String(),
		s.Type.String(),
		"",
		"",
		"",
		s.ClientOrderID,
		s.Price,
		s.Amount)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = strconv.FormatInt(tempResp.Result.ID, 10)
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (f *FTX) ModifyOrder(action *order.Modify) (string, error) {
	if action.TriggerPrice != 0 {
		a, err := f.ModifyTriggerOrder(action.ID,
			action.Type.String(),
			action.Amount,
			action.TriggerPrice,
			action.Price,
			0)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(a.Result.ID, 10), err
	}
	var o ModifyOrder
	var err error
	switch action.ID {
	case "":
		o, err = f.ModifyOrderByClientID(action.ClientOrderID, action.ClientOrderID, action.Price, action.Amount)
		if err != nil {
			return "", err
		}
	default:
		o, err = f.ModifyPlacedOrder(action.ID, action.ClientOrderID, action.Price, action.Amount)
		if err != nil {
			return "", err
		}
	}
	return strconv.FormatInt(o.Result.ID, 10), err
}

// CancelOrder cancels an order by its corresponding ID number
func (f *FTX) CancelOrder(order *order.Cancel) error {
	_, err := f.DeleteOrder(order.ID)
	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (f *FTX) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	tempMap := make(map[string]string)
	orders, err := f.GetOpenOrders(f.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType).String())
	if err != nil {
		return resp, err
	}
	for x := range orders.Result {
		_, err := f.DeleteOrder(strconv.FormatInt(orders.Result[x].ID, 10))
		if err != nil {
			tempMap[strconv.FormatInt(orders.Result[x].ID, 10)] = "Cancellation Failed"
			continue
		}
		tempMap[strconv.FormatInt(orders.Result[x].ID, 10)] = "Success"
	}
	resp.Status = tempMap
	return resp, nil
}

// GetCompatible gets compatible variables for order vars
func (s *OrderStatus) GetCompatible(f *FTX) (OrderVars, error) {
	var resp OrderVars
	switch s.Result.Side {
	case order.Buy.Lower():
		resp.Side = order.Buy
	case order.Sell.Lower():
		resp.Side = order.Sell
	}
	switch s.Result.Status {
	case strings.ToLower(order.New.String()):
		resp.Status = order.New
	case strings.ToLower(order.Open.String()):
		resp.Status = order.Open
	case closedStatus:
		if s.Result.FilledSize != 0 && s.Result.FilledSize != s.Result.Size {
			resp.Status = order.PartiallyCancelled
		}
		if s.Result.FilledSize == 0 {
			resp.Status = order.Cancelled
		}
		if s.Result.FilledSize == s.Result.Size {
			resp.Status = order.Filled
		}
	}
	var feeBuilder exchange.FeeBuilder
	feeBuilder.PurchasePrice = s.Result.AvgFillPrice
	feeBuilder.Amount = s.Result.Size
	resp.OrderType = order.Market
	if strings.EqualFold(s.Result.OrderType, order.Limit.String()) {
		resp.OrderType = order.Limit
		feeBuilder.IsMaker = true
	}
	fee, err := f.GetFee(&feeBuilder)
	if err != nil {
		return resp, err
	}
	resp.Fee = fee
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (f *FTX) GetOrderInfo(orderID string) (order.Detail, error) {
	var resp order.Detail
	orderData, err := f.GetOrderStatus(orderID)
	if err != nil {
		return resp, err
	}
	resp.ID = strconv.FormatInt(orderData.Result.ID, 10)
	resp.Amount = orderData.Result.Size
	resp.AssetType = asset.Spot
	resp.ClientOrderID = orderData.Result.ClientID
	resp.Date = orderData.Result.CreatedAt
	resp.Exchange = f.Name
	resp.ExecutedAmount = orderData.Result.Size - orderData.Result.RemainingSize
	resp.Pair = currency.NewPairFromString(orderData.Result.Market)
	resp.Price = orderData.Result.Price
	resp.RemainingAmount = orderData.Result.RemainingSize
	orderVars, err := orderData.GetCompatible(f)
	if err != nil {
		return resp, err
	}
	resp.Status = orderVars.Status
	resp.Side = orderVars.Side
	resp.Type = orderVars.OrderType
	resp.Fee = orderVars.Fee
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (f *FTX) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	a, err := f.FetchDepositAddress(cryptocurrency.String())
	if err != nil {
		return "", err
	}
	return a.Result.Address, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *FTX) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	var address, addressTag string
	if withdrawRequest.Crypto != nil {
		address = withdrawRequest.Crypto.Address
		addressTag = withdrawRequest.Crypto.AddressTag
	}
	resp := withdraw.ExchangeResponse{}
	a, err := f.Withdraw(withdrawRequest.Currency.String(),
		address,
		addressTag,
		withdrawRequest.TradePassword,
		strconv.FormatInt(withdrawRequest.OneTimePassword, 10),
		withdrawRequest.Amount)
	if err != nil {
		return &resp, err
	}
	resp.ID = strconv.FormatInt(a.Result.ID, 10)
	resp.Status = a.Result.Status
	return &resp, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *FTX) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	var resp *withdraw.ExchangeResponse
	return resp, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (f *FTX) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (f *FTX) GetWebsocket() (*wshandler.Websocket, error) {
	return f.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (f *FTX) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		var tempResp order.Detail
		orderData, err := f.GetOpenOrders(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String())
		if err != nil {
			return resp, err
		}
		for y := range orderData.Result {
			tempResp.ID = strconv.FormatInt(orderData.Result[y].ID, 10)
			tempResp.Amount = orderData.Result[y].Size
			tempResp.AssetType = asset.Spot
			tempResp.ClientOrderID = orderData.Result[y].ClientID
			tempResp.Date = orderData.Result[y].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = orderData.Result[y].Size - orderData.Result[y].RemainingSize
			tempResp.Pair = currency.NewPairFromString(orderData.Result[y].Market)
			tempResp.Price = orderData.Result[y].Price
			tempResp.RemainingAmount = orderData.Result[y].RemainingSize
			var orderVars OrderVars
			orderVars, err = f.compatibleOrderVars(orderData.Result[y].Side,
				orderData.Result[y].Status,
				orderData.Result[y].OrderType,
				orderData.Result[y].FilledSize,
				orderData.Result[y].Size,
				orderData.Result[y].AvgFillPrice)
			if err != nil {
				return resp, err
			}
			tempResp.Status = orderVars.Status
			tempResp.Side = orderVars.Side
			tempResp.Type = orderVars.OrderType
			tempResp.Fee = orderVars.Fee
			resp = append(resp, tempResp)
		}
		triggerOrderData, err := f.GetOpenTriggerOrders(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String(), getOrdersRequest.Type.String())
		if err != nil {
			return resp, err
		}
		for z := range triggerOrderData.Result {
			tempResp.ID = strconv.FormatInt(triggerOrderData.Result[z].ID, 10)
			tempResp.Amount = triggerOrderData.Result[z].Size
			tempResp.AssetType = asset.Spot
			tempResp.Date = triggerOrderData.Result[z].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = triggerOrderData.Result[z].FilledSize
			tempResp.Pair = currency.NewPairFromString(triggerOrderData.Result[z].Market)
			tempResp.Price = triggerOrderData.Result[z].AvgFillPrice
			tempResp.RemainingAmount = triggerOrderData.Result[z].Size - triggerOrderData.Result[z].FilledSize
			tempResp.TriggerPrice = triggerOrderData.Result[z].TriggerPrice
			orderVars, err := f.compatibleOrderVars(triggerOrderData.Result[z].Side,
				triggerOrderData.Result[z].Status,
				triggerOrderData.Result[z].OrderType,
				triggerOrderData.Result[z].FilledSize,
				triggerOrderData.Result[z].Size,
				triggerOrderData.Result[z].AvgFillPrice)
			if err != nil {
				return resp, err
			}
			tempResp.Status = orderVars.Status
			tempResp.Side = orderVars.Side
			tempResp.Type = orderVars.OrderType
			tempResp.Fee = orderVars.Fee
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (f *FTX) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		var tempResp order.Detail
		orderData, err := f.FetchOrderHistory(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String(),
			getOrdersRequest.StartTicks, getOrdersRequest.EndTicks, "")
		if err != nil {
			return resp, err
		}
		for y := range orderData.Result {
			tempResp.ID = strconv.FormatInt(orderData.Result[y].ID, 10)
			tempResp.Amount = orderData.Result[y].Size
			tempResp.AssetType = asset.Spot
			tempResp.ClientOrderID = orderData.Result[y].ClientID
			tempResp.Date = orderData.Result[y].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = orderData.Result[y].Size - orderData.Result[y].RemainingSize
			tempResp.Pair = currency.NewPairFromString(orderData.Result[y].Market)
			tempResp.Price = orderData.Result[y].Price
			tempResp.RemainingAmount = orderData.Result[y].RemainingSize
			var orderVars OrderVars
			orderVars, err = f.compatibleOrderVars(orderData.Result[y].Side,
				orderData.Result[y].Status,
				orderData.Result[y].OrderType,
				orderData.Result[y].FilledSize,
				orderData.Result[y].Size,
				orderData.Result[y].AvgFillPrice)
			if err != nil {
				return resp, err
			}
			tempResp.Status = orderVars.Status
			tempResp.Side = orderVars.Side
			tempResp.Type = orderVars.OrderType
			tempResp.Fee = orderVars.Fee
			resp = append(resp, tempResp)
		}
		triggerOrderData, err := f.GetTriggerOrderHistory(f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Spot).String(),
			getOrdersRequest.StartTicks, getOrdersRequest.EndTicks, getOrdersRequest.Side.String(), getOrdersRequest.Type.String(), "")
		if err != nil {
			return resp, err
		}
		for z := range triggerOrderData.Result {
			tempResp.ID = strconv.FormatInt(triggerOrderData.Result[z].ID, 10)
			tempResp.Amount = triggerOrderData.Result[z].Size
			tempResp.AssetType = asset.Spot
			tempResp.Date = triggerOrderData.Result[z].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = triggerOrderData.Result[z].FilledSize
			tempResp.Pair = currency.NewPairFromString(triggerOrderData.Result[z].Market)
			tempResp.Price = triggerOrderData.Result[z].AvgFillPrice
			tempResp.RemainingAmount = triggerOrderData.Result[z].Size - triggerOrderData.Result[z].FilledSize
			tempResp.TriggerPrice = triggerOrderData.Result[z].TriggerPrice
			orderVars, err := f.compatibleOrderVars(triggerOrderData.Result[z].Side,
				triggerOrderData.Result[z].Status,
				triggerOrderData.Result[z].OrderType,
				triggerOrderData.Result[z].FilledSize,
				triggerOrderData.Result[z].Size,
				triggerOrderData.Result[z].AvgFillPrice)
			if err != nil {
				return resp, err
			}
			tempResp.Status = orderVars.Status
			tempResp.Side = orderVars.Side
			tempResp.Type = orderVars.OrderType
			tempResp.Fee = orderVars.Fee
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (f *FTX) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return f.GetFee(feeBuilder)
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (f *FTX) SubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	f.Websocket.SubscribeToChannels(channels)
	return nil
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (f *FTX) UnsubscribeToWebsocketChannels(channels []wshandler.WebsocketChannelSubscription) error {
	f.Websocket.RemoveSubscribedChannels(channels)
	return nil
}

// GetSubscriptions returns a copied list of subscriptions
func (f *FTX) GetSubscriptions() ([]wshandler.WebsocketChannelSubscription, error) {
	return f.Websocket.GetSubscriptions(), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (f *FTX) AuthenticateWebsocket() error {
	return f.WsAuth()
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (f *FTX) ValidateCredentials() error {
	_, err := f.UpdateAccountInfo()
	return f.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (f *FTX) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval time.Duration) (kline.Item, error) {
	intervalToString, err := parseInterval(interval)
	if err != nil {
		return kline.Item{}, err
	}
	var resp kline.Item
	ohlcData, err := f.GetHistoricalData(f.FormatExchangeCurrency(pair, a).String(),
		string(intervalToString), "", start, end)
	if err != nil {
		return resp, err
	}
	resp.Exchange = f.Name
	resp.Asset = a
	resp.Pair = pair
	for x := range ohlcData.Result {
		var tempData kline.Candle
		tempData.Open = ohlcData.Result[x].Open
		tempData.High = ohlcData.Result[x].High
		tempData.Low = ohlcData.Result[x].Low
		tempData.Close = ohlcData.Result[x].Close
		tempData.Volume = ohlcData.Result[x].Volume
		tempData.Time = ohlcData.Result[x].StartTime
		resp.Candles = append(resp.Candles, tempData)
	}
	return resp, nil
}
