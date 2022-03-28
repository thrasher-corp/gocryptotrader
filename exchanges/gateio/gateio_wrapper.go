package gateio

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (g *Gateio) GetDefaultConfig() (*config.Exchange, error) {
	g.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = g.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = g.BaseCurrencies

	err := g.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if g.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = g.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (g *Gateio) SetDefaults() {
	g.Name = "GateIO"
	g.Enabled = true
	g.Verbose = true
	g.API.CredentialsValidator.RequiresKey = true
	g.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Delimiter: currency.UnderscoreDelimiter, Uppercase: true}
	err := g.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	g.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrders:          true,
				CancelOrder:           true,
				SubmitOrder:           true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				TradeFetching:          true,
				KlineFetching:          true,
				FullPayloadSubscribe:   true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrder:               true,
				AccountBalance:         true,
				Subscribe:              true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.ThreeMin.Word():   true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.SixHour.Word():    true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
				},
			},
		},
	}
	g.Requester, err = request.New(g.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.API.Endpoints = g.NewEndpoints()
	err = g.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              gateioTradeURL,
		exchange.RestSpotSupplementary: gateioMarketURL,
		exchange.WebsocketSpot:         gateioWebsocketEndpoint,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	g.Websocket = stream.New()
	g.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	g.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	g.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user configuration
func (g *Gateio) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		g.SetEnabled(false)
		return nil
	}
	err = g.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := g.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = g.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            gateioWebsocketEndpoint,
		RunningURL:            wsRunningURL,
		Connector:             g.WsConnect,
		Subscriber:            g.Subscribe,
		GenerateSubscriptions: g.GenerateDefaultSubscriptions,
		Features:              &g.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return g.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            gateioWebsocketRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the GateIO go routine
func (g *Gateio) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		g.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the GateIO wrapper
func (g *Gateio) Run() {
	if g.Verbose {
		g.PrintEnabledPairs()
	}

	if !g.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := g.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", g.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (g *Gateio) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	return g.GetSymbols(ctx)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (g *Gateio) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := g.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return g.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (g *Gateio) UpdateTickers(ctx context.Context, a asset.Item) error {
	result, err := g.GetTickers(ctx)
	if err != nil {
		return err
	}
	pairs, err := g.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for p := range pairs {
		for k := range result {
			if !strings.EqualFold(k, pairs[p].String()) {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         result[k].Last,
				High:         result[k].High,
				Low:          result[k].Low,
				Volume:       result[k].BaseVolume,
				QuoteVolume:  result[k].QuoteVolume,
				Open:         result[k].Open,
				Close:        result[k].Close,
				Pair:         pairs[p],
				ExchangeName: g.Name,
				AssetType:    a})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (g *Gateio) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := g.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(g.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (g *Gateio) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(g.Name, p, assetType)
	if err != nil {
		return g.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (g *Gateio) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(g.Name, p, assetType)
	if err != nil {
		return g.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (g *Gateio) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        g.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: g.CanVerifyOrderbook,
	}
	curr, err := g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := g.GetOrderbook(ctx, curr.String())
	if err != nil {
		return book, err
	}

	for x := range orderbookNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		})
	}

	for x := range orderbookNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(g.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (g *Gateio) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var balances []account.Balance

	if g.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		resp, err := g.wsGetBalance([]string{})
		if err != nil {
			return info, err
		}
		var currData []account.Balance
		for k := range resp.Result {
			currData = append(currData, account.Balance{
				CurrencyName: currency.NewCode(k),
				Total:        resp.Result[k].Available + resp.Result[k].Freeze,
				Hold:         resp.Result[k].Freeze,
				Free:         resp.Result[k].Available,
			})
		}
		info.Accounts = append(info.Accounts, account.SubAccount{
			Currencies: currData,
		})
	} else {
		balance, err := g.GetBalances(ctx)
		if err != nil {
			return info, err
		}

		switch l := balance.Locked.(type) {
		case map[string]interface{}:
			for x := range l {
				var lockedF float64
				lockedF, err = strconv.ParseFloat(l[x].(string), 64)
				if err != nil {
					return info, err
				}

				balances = append(balances, account.Balance{
					CurrencyName: currency.NewCode(x),
					Hold:         lockedF,
				})
			}
		default:
			break
		}

		switch v := balance.Available.(type) {
		case map[string]interface{}:
			for x := range v {
				var availAmount float64
				availAmount, err = strconv.ParseFloat(v[x].(string), 64)
				if err != nil {
					return info, err
				}

				var updated bool
				for i := range balances {
					if !balances[i].CurrencyName.Equal(currency.NewCode(x)) {
						continue
					}
					balances[i].Total = balances[i].Hold + availAmount
					balances[i].Free = availAmount
					balances[i].AvailableWithoutBorrow = availAmount
					updated = true
					break
				}
				if !updated {
					balances = append(balances, account.Balance{
						CurrencyName: currency.NewCode(x),
						Total:        availAmount,
					})
				}
			}
		default:
			break
		}

		info.Accounts = append(info.Accounts, account.SubAccount{
			Currencies: balances,
		})
	}

	info.Exchange = g.Name
	if err := account.Process(&info); err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (g *Gateio) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(g.Name, assetType)
	if err != nil {
		return g.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (g *Gateio) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (g *Gateio) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (g *Gateio) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = g.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData TradeHistory
	tradeData, err = g.GetTrades(ctx, p.String())
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData.Data {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData.Data[i].Type)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			Exchange:     g.Name,
			TID:          tradeData.Data[i].TradeID,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData.Data[i].Rate,
			Amount:       tradeData.Data[i].Amount,
			Timestamp:    time.Unix(tradeData.Data[i].Timestamp, 0),
		})
	}

	err = g.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (g *Gateio) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
// TODO: support multiple order types (IOC)
func (g *Gateio) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	var orderTypeFormat string
	if s.Side == order.Buy {
		orderTypeFormat = order.Buy.Lower()
	} else {
		orderTypeFormat = order.Sell.Lower()
	}

	fPair, err := g.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	var spotNewOrderRequestParams = SpotNewOrderRequestParams{
		Amount: s.Amount,
		Price:  s.Price,
		Symbol: fPair.String(),
		Type:   orderTypeFormat,
	}

	response, err := g.SpotNewOrder(ctx, spotNewOrderRequestParams)
	if err != nil {
		return submitOrderResponse, err
	}
	if response.OrderNumber > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.OrderNumber, 10)
	}
	if response.LeftAmount == 0 {
		submitOrderResponse.FullyMatched = true
	}
	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (g *Gateio) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (g *Gateio) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	fpair, err := g.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}

	_, err = g.CancelExistingOrder(ctx, orderIDInt, fpair.String())
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (g *Gateio) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (g *Gateio) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := g.GetOpenOrders(ctx, "")
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	uniqueSymbols := make(map[string]int)
	for i := range openOrders.Orders {
		uniqueSymbols[openOrders.Orders[i].CurrencyPair]++
	}

	for unique := range uniqueSymbols {
		err = g.CancelAllExistingOrders(ctx, -1, unique)
		if err != nil {
			cancelAllOrdersResponse.Status[unique] = err.Error()
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (g *Gateio) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	orders, err := g.GetOpenOrders(ctx, "")
	if err != nil {
		return orderDetail, errors.New("failed to get open orders")
	}

	if assetType == "" {
		assetType = asset.Spot
	}

	format, err := g.GetPairFormat(assetType, false)
	if err != nil {
		return orderDetail, err
	}

	for x := range orders.Orders {
		if orders.Orders[x].OrderNumber != orderID {
			continue
		}
		orderDetail.Exchange = g.Name
		orderDetail.ID = orders.Orders[x].OrderNumber
		orderDetail.RemainingAmount = orders.Orders[x].InitialAmount - orders.Orders[x].FilledAmount
		orderDetail.ExecutedAmount = orders.Orders[x].FilledAmount
		orderDetail.Amount = orders.Orders[x].InitialAmount
		orderDetail.Date = time.Unix(orders.Orders[x].Timestamp, 0)
		if orderDetail.Status, err = order.StringToOrderStatus(orders.Orders[x].Status); err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
		}
		orderDetail.Price = orders.Orders[x].Rate
		orderDetail.Pair, err = currency.NewPairDelimiter(orders.Orders[x].CurrencyPair,
			format.Delimiter)
		if err != nil {
			return orderDetail, err
		}
		if strings.EqualFold(orders.Orders[x].Type, order.Ask.String()) {
			orderDetail.Side = order.Ask
		} else if strings.EqualFold(orders.Orders[x].Type, order.Bid.String()) {
			orderDetail.Side = order.Buy
		}
		return orderDetail, nil
	}
	return orderDetail, fmt.Errorf("no order found with id %v", orderID)
}

// GetDepositAddress returns a deposit address for a specified currency
func (g *Gateio) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	addr, err := g.GetCryptoDepositAddress(ctx, cryptocurrency.String())
	if err != nil {
		return nil, err
	}

	if addr.Address == gateioGenerateAddress {
		return nil,
			errors.New("new deposit address is being generated, please retry again shortly")
	}

	if chain != "" {
		for x := range addr.MultichainAddresses {
			if strings.EqualFold(addr.MultichainAddresses[x].Chain, chain) {
				return &deposit.Address{
					Address: addr.MultichainAddresses[x].Address,
					Tag:     addr.MultichainAddresses[x].PaymentName,
				}, nil
			}
		}
		return nil, fmt.Errorf("network %s not found", chain)
	}
	return &deposit.Address{
		Address: addr.Address,
		Tag:     addr.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (g *Gateio) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	return g.WithdrawCrypto(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Crypto.Chain,
		withdrawRequest.Amount,
	)
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (g *Gateio) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (g *Gateio) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !g.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return g.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (g *Gateio) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var orders []order.Detail
	var currPair string
	if len(req.Pairs) == 1 {
		fPair, err := g.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = fPair.String()
	}
	if g.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		for i := 0; ; i += 100 {
			resp, err := g.wsGetOrderInfo(req.Type.String(), i, 100)
			if err != nil {
				return orders, err
			}

			for j := range resp.WebSocketOrderQueryRecords {
				orderSide := order.Buy
				if resp.WebSocketOrderQueryRecords[j].Type == 1 {
					orderSide = order.Sell
				}
				orderType := order.Market
				if resp.WebSocketOrderQueryRecords[j].OrderType == 1 {
					orderType = order.Limit
				}
				p, err := currency.NewPairFromString(resp.WebSocketOrderQueryRecords[j].Market)
				if err != nil {
					return nil, err
				}
				orders = append(orders, order.Detail{
					Exchange:        g.Name,
					AccountID:       strconv.FormatInt(resp.WebSocketOrderQueryRecords[j].User, 10),
					ID:              strconv.FormatInt(resp.WebSocketOrderQueryRecords[j].ID, 10),
					Pair:            p,
					Side:            orderSide,
					Type:            orderType,
					Date:            convert.TimeFromUnixTimestampDecimal(resp.WebSocketOrderQueryRecords[j].Ctime),
					Price:           resp.WebSocketOrderQueryRecords[j].Price,
					Amount:          resp.WebSocketOrderQueryRecords[j].Amount,
					ExecutedAmount:  resp.WebSocketOrderQueryRecords[j].FilledAmount,
					RemainingAmount: resp.WebSocketOrderQueryRecords[j].Left,
					Fee:             resp.WebSocketOrderQueryRecords[j].DealFee,
				})
			}
			if len(resp.WebSocketOrderQueryRecords) < 100 {
				break
			}
		}
	} else {
		resp, err := g.GetOpenOrders(ctx, currPair)
		if err != nil {
			return nil, err
		}

		format, err := g.GetPairFormat(asset.Spot, false)
		if err != nil {
			return nil, err
		}

		for i := range resp.Orders {
			if resp.Orders[i].Status != "open" {
				continue
			}
			var symbol currency.Pair
			symbol, err = currency.NewPairDelimiter(resp.Orders[i].CurrencyPair,
				format.Delimiter)
			if err != nil {
				return nil, err
			}
			side := order.Side(strings.ToUpper(resp.Orders[i].Type))
			status, err := order.StringToOrderStatus(resp.Orders[i].Status)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", g.Name, err)
			}
			orderDate := time.Unix(resp.Orders[i].Timestamp, 0)
			orders = append(orders, order.Detail{
				ID:              resp.Orders[i].OrderNumber,
				Amount:          resp.Orders[i].Amount,
				ExecutedAmount:  resp.Orders[i].Amount - resp.Orders[i].FilledAmount,
				RemainingAmount: resp.Orders[i].FilledAmount,
				Price:           resp.Orders[i].Rate,
				Date:            orderDate,
				Side:            side,
				Exchange:        g.Name,
				Pair:            symbol,
				Status:          status,
			})
		}
	}
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (g *Gateio) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var trades []TradesResponse
	for i := range req.Pairs {
		resp, err := g.GetTradeHistory(ctx, req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		trades = append(trades, resp.Trades...)
	}

	format, err := g.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range trades {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(trades[i].Pair, format.Delimiter)
		if err != nil {
			return nil, err
		}
		side := order.Side(strings.ToUpper(trades[i].Type))
		orderDate := time.Unix(trades[i].TimeUnix, 0)
		detail := order.Detail{
			ID:                   strconv.FormatInt(trades[i].OrderID, 10),
			Amount:               trades[i].Amount,
			ExecutedAmount:       trades[i].Amount,
			Price:                trades[i].Rate,
			AverageExecutedPrice: trades[i].Rate,
			Date:                 orderDate,
			Side:                 side,
			Exchange:             g.Name,
			Pair:                 pair,
		}
		detail.InferCostsAndTimes()
		orders = append(orders, detail)
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (g *Gateio) AuthenticateWebsocket(ctx context.Context) error {
	return g.wsServerSignIn(ctx)
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (g *Gateio) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := g.UpdateAccountInfo(ctx, assetType)
	return g.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (g *Gateio) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Seconds(), 'f', 0, 64)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (g *Gateio) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := g.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	hours := time.Since(start).Hours()
	formattedPair, err := g.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	params := KlinesRequestParams{
		Symbol:   formattedPair.String(),
		GroupSec: g.FormatExchangeKlineInterval(interval),
		HourSize: int(hours),
	}

	klineData, err := g.GetSpotKline(ctx, params)
	if err != nil {
		return kline.Item{}, err
	}
	klineData.Interval = interval
	klineData.Pair = pair
	klineData.Asset = a

	klineData.SortCandlesByTimestamp(false)
	klineData.RemoveOutsideRange(start, end)
	return klineData, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (g *Gateio) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return g.GetHistoricCandles(ctx, pair, a, start, end, interval)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (g *Gateio) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	chains, err := g.GetCryptoDepositAddress(ctx, cryptocurrency.String())
	if err != nil {
		return nil, err
	}

	var availableChains []string
	for x := range chains.MultichainAddresses {
		availableChains = append(availableChains, chains.MultichainAddresses[x].Chain)
	}
	return availableChains, nil
}
