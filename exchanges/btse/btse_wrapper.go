package btse

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// GetDefaultConfig returns a default exchange config
func (b *BTSE) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for BTSE
func (b *BTSE) SetDefaults() {
	b.Name = "BTSE"
	b.Enabled = true
	b.Verbose = true
	b.APIWithdrawPermissions = exchange.NoAPIWithdrawalMethods
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	b.CurrencyPairs = currency.PairsManager{
		AssetTypes: assets.AssetTypes{
			assets.AssetTypeSpot,
		},
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: exchange.ProtocolFeatures{
				AutoPairUpdates: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, 0),
		request.NewRateLimit(time.Second, 0),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	b.API.Endpoints.URLDefault = btseAPIURL
	b.API.Endpoints.URL = b.API.Endpoints.URLDefault
	b.WebsocketInit()
	b.Websocket.Functionality = exchange.WebsocketTickerSupported |
		exchange.WebsocketOrderbookSupported
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *BTSE) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}

	err := b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	return b.WebsocketSetup(b.WsConnect,
		exch.Name,
		exch.Features.Enabled.Websocket,
		btseWebsocket,
		exch.API.Endpoints.WebsocketURL)
}

// Start starts the BTSE go routine
func (b *BTSE) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the BTSE wrapper
func (b *BTSE) Run() {
	if b.Verbose {
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf("%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *BTSE) FetchTradablePairs(asset assets.AssetType) ([]string, error) {
	markets, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	var pairs []string
	for _, m := range *markets {
		pairs = append(pairs, m.ID)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *BTSE) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(assets.AssetTypeSpot)
	if err != nil {
		return err
	}

	return b.UpdatePairs(currency.NewPairsFromStrings(pairs), assets.AssetTypeSpot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTSE) UpdateTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	var tickerPrice ticker.Price

	t, err := b.GetTicker(b.FormatExchangeCurrency(p,
		assetType).String())
	if err != nil {
		return tickerPrice, err
	}

	s, err := b.GetMarketStatistics(b.FormatExchangeCurrency(p,
		assetType).String())
	if err != nil {
		return tickerPrice, err

	}

	tickerPrice.Pair = p
	tickerPrice.Ask = t.Ask
	tickerPrice.Bid = t.Bid
	tickerPrice.Low = s.Low
	tickerPrice.Last = t.Price
	tickerPrice.Volume = s.Volume
	tickerPrice.High = s.High

	err = ticker.ProcessTicker(b.GetName(), &tickerPrice, assetType)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(b.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *BTSE) FetchTicker(p currency.Pair, assetType assets.AssetType) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *BTSE) FetchOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	ob, err := orderbook.Get(b.GetName(), p, assetType)
	if err != nil {
		return b.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTSE) UpdateOrderbook(p currency.Pair, assetType assets.AssetType) (orderbook.Base, error) {
	return orderbook.Base{}, common.ErrFunctionNotSupported
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// BTSE exchange
func (b *BTSE) GetAccountInfo() (exchange.AccountInfo, error) {
	var a exchange.AccountInfo
	balance, err := b.GetAccountBalance()
	if err != nil {
		return a, err
	}

	var currencies []exchange.AccountCurrencyInfo
	for _, b := range *balance {
		currencies = append(currencies,
			exchange.AccountCurrencyInfo{
				CurrencyName: currency.NewCode(b.Currency),
				TotalValue:   b.Total,
				Hold:         b.Available,
			},
		)
	}
	a.Exchange = b.Name
	a.Accounts = []exchange.Account{
		{
			Currencies: currencies,
		},
	}
	return a, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *BTSE) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (b *BTSE) GetExchangeHistory(p currency.Pair, assetType assets.AssetType) ([]exchange.TradeHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (b *BTSE) SubmitOrder(p currency.Pair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var resp exchange.SubmitOrderResponse
	r, err := b.CreateOrder(amount, price, side.ToString(),
		orderType.ToString(), b.FormatExchangeCurrency(p,
			assets.AssetTypeSpot).String(), "GTC", clientID)
	if err != nil {
		return resp, err
	}

	if *r != "" {
		resp.IsOrderPlaced = true
		resp.OrderID = *r
	}

	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTSE) ModifyOrder(action *exchange.ModifyOrder) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTSE) CancelOrder(order *exchange.OrderCancellation) error {
	r, err := b.CancelExistingOrder(order.OrderID,
		b.FormatExchangeCurrency(order.CurrencyPair,
			assets.AssetTypeSpot).String())
	if err != nil {
		return err
	}

	switch r.Code {
	case -1:
		return errors.New("order cancellation unsuccessful")
	case 4:
		return errors.New("order cancellation timeout")
	}

	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
// If product ID is sent, all orders of that specified market will be cancelled
// If not specified, all orders of all markets will be cancelled
func (b *BTSE) CancelAllOrders(orderCancellation *exchange.OrderCancellation) (exchange.CancelAllOrdersResponse, error) {
	r, err := b.CancelOrders(b.FormatExchangeCurrency(
		orderCancellation.CurrencyPair, assets.AssetTypeSpot).String(),
	)
	if err != nil {
		return exchange.CancelAllOrdersResponse{}, err
	}

	var resp exchange.CancelAllOrdersResponse
	switch r.Code {
	case -1:
		return resp, errors.New("order cancellation unsuccessful")
	case 4:
		return resp, errors.New("order cancellation timeout")
	}

	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (b *BTSE) GetOrderInfo(orderID string) (exchange.OrderDetail, error) {
	o, err := b.GetOrders("")
	if err != nil {
		return exchange.OrderDetail{}, err
	}

	var od exchange.OrderDetail
	if len(*o) == 0 {
		return od, errors.New("no orders found")
	}

	for i := range *o {
		o := (*o)[i]
		if o.ID != orderID {
			continue
		}

		var side = exchange.BuyOrderSide
		if strings.EqualFold(o.Side, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		}

		od.CurrencyPair = currency.NewPairDelimiter(o.ProductID,
			b.CurrencyPairs.ConfigFormat.Delimiter)
		od.Exchange = b.Name
		od.Amount = o.Amount
		od.ID = o.ID
		od.OrderDate = parseOrderTime(o.CreatedAt)
		od.OrderSide = side
		od.OrderType = exchange.OrderType(strings.ToUpper(o.Type))
		od.Price = o.Price
		od.Status = o.Status

		fills, err := b.GetFills(orderID, "", "", "", "")
		if err != nil {
			return od, fmt.Errorf("unable to get order fills for orderID %s", orderID)
		}

		for i := range *fills {
			f := (*fills)[i]
			createdAt, _ := time.Parse(time.RFC3339, f.CreatedAt)
			od.Trades = append(od.Trades, exchange.TradeHistory{
				Timestamp: createdAt,
				TID:       f.ID,
				Price:     f.Price,
				Amount:    f.Amount,
				Exchange:  b.Name,
				Type:      exchange.OrderSide(f.Side).ToString(),
				Fee:       f.Fee,
			})
		}
	}
	return od, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTSE) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawCryptocurrencyFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFunds(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFundsToInternationalBank(withdrawRequest *exchange.WithdrawRequest) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (b *BTSE) GetWebsocket() (*exchange.Websocket, error) {
	return b.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTSE) GetActiveOrders(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	resp, err := b.GetOrders("")
	if err != nil {
		return nil, err
	}

	var orders []exchange.OrderDetail
	for i := range *resp {
		order := (*resp)[i]
		var side = exchange.BuyOrderSide
		if strings.EqualFold(order.Side, exchange.AskOrderSide.ToString()) {
			side = exchange.SellOrderSide
		}

		openOrder := exchange.OrderDetail{
			CurrencyPair: currency.NewPairDelimiter(order.ProductID,
				b.CurrencyPairs.ConfigFormat.Delimiter),
			Exchange:  b.Name,
			Amount:    order.Amount,
			ID:        order.ID,
			OrderDate: parseOrderTime(order.CreatedAt),
			OrderSide: side,
			OrderType: exchange.OrderType(strings.ToUpper(order.Type)),
			Price:     order.Price,
			Status:    order.Status,
		}

		fills, err := b.GetFills(order.ID, "", "", "", "")
		if err != nil {
			log.Errorf("unable to get order fills for orderID %s", order.ID)
			continue
		}

		for i := range *fills {
			f := (*fills)[i]
			createdAt, _ := time.Parse(time.RFC3339, f.CreatedAt)
			openOrder.Trades = append(openOrder.Trades, exchange.TradeHistory{
				Timestamp: createdAt,
				TID:       f.ID,
				Price:     f.Price,
				Amount:    f.Amount,
				Exchange:  b.Name,
				Type:      exchange.OrderSide(f.Side).ToString(),
				Fee:       f.Fee,
			})
		}
		orders = append(orders, openOrder)
	}

	exchange.FilterOrdersByType(&orders, getOrdersRequest.OrderType)
	exchange.FilterOrdersByTickRange(&orders, getOrdersRequest.StartTicks, getOrdersRequest.EndTicks)
	exchange.FilterOrdersBySide(&orders, getOrdersRequest.OrderSide)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTSE) GetOrderHistory(getOrdersRequest *exchange.GetOrdersRequest) ([]exchange.OrderDetail, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTSE) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}
