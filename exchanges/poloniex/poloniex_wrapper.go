package poloniex

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (p *Poloniex) GetDefaultConfig() (*config.Exchange, error) {
	p.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = p.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = p.BaseCurrencies

	err := p.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if p.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = p.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default settings for poloniex
func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = true
	p.Verbose = true
	p.API.CredentialsValidator.RequiresKey = true
	p.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	configFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}

	err := p.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	p.Features = exchange.Features{
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
				CancelOrder:           true,
				CancelOrders:          true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
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
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.TwoHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.OneDay.Word():     true,
				},
			},
		},
	}

	p.Requester, err = request.New(p.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	p.API.Endpoints = p.NewEndpoints()
	err = p.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      poloniexAPIURL,
		exchange.WebsocketSpot: poloniexWebsocketAddress,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	p.Websocket = stream.New()
	p.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	p.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	p.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (p *Poloniex) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		p.SetEnabled(false)
		return nil
	}
	err = p.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := p.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = p.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            poloniexWebsocketAddress,
		RunningURL:            wsRunningURL,
		Connector:             p.WsConnect,
		Subscriber:            p.Subscribe,
		Unsubscriber:          p.Unsubscribe,
		GenerateSubscriptions: p.GenerateDefaultSubscriptions,
		Features:              &p.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	return p.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Poloniex go routine
func (p *Poloniex) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		p.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Poloniex wrapper
func (p *Poloniex) Run() {
	if p.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s (url: %s).\n",
			p.Name,
			common.IsEnabled(p.Websocket.IsEnabled()),
			poloniexWebsocketAddress)
		p.PrintEnabledPairs()
	}

	forceUpdate := false

	avail, err := p.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			p.Name,
			err)
		return
	}

	if common.StringDataCompare(avail.Strings(), "BTC_USDT") {
		log.Warnf(log.ExchangeSys,
			"%s contains invalid pair, forcing upgrade of available currencies.\n",
			p.Name)
		forceUpdate = true
	}

	if !p.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = p.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			p.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (p *Poloniex) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	resp, err := p.GetTicker(ctx)
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range resp {
		currencies = append(currencies, x)
	}

	return currencies, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (p *Poloniex) UpdateTradablePairs(ctx context.Context, forceUpgrade bool) error {
	pairs, err := p.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	ps, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return p.UpdatePairs(ps, asset.Spot, false, forceUpgrade)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (p *Poloniex) UpdateTickers(ctx context.Context, a asset.Item) error {
	tick, err := p.GetTicker(ctx)
	if err != nil {
		return err
	}

	enabledPairs, err := p.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range enabledPairs {
		fpair, err := p.FormatExchangeCurrency(enabledPairs[i], a)
		if err != nil {
			return err
		}
		curr := fpair.String()
		if _, ok := tick[curr]; !ok {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Pair:         enabledPairs[i],
			Ask:          tick[curr].LowestAsk,
			Bid:          tick[curr].HighestBid,
			High:         tick[curr].High24Hr,
			Last:         tick[curr].Last,
			Low:          tick[curr].Low24Hr,
			Volume:       tick[curr].BaseVolume,
			QuoteVolume:  tick[curr].QuoteVolume,
			ExchangeName: p.Name,
			AssetType:    a})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *Poloniex) UpdateTicker(ctx context.Context, currencyPair currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := p.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(p.Name, currencyPair, a)
}

// FetchTicker returns the ticker for a currency pair
func (p *Poloniex) FetchTicker(ctx context.Context, currencyPair currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateTicker(ctx, currencyPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (p *Poloniex) FetchOrderbook(ctx context.Context, currencyPair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateOrderbook(ctx, currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *Poloniex) UpdateOrderbook(ctx context.Context, c currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	callingBook := &orderbook.Base{
		Exchange:        p.Name,
		Pair:            c,
		Asset:           assetType,
		VerifyOrderbook: p.CanVerifyOrderbook,
	}
	orderbookNew, err := p.GetOrderbook(ctx, "", poloniexMaxOrderbookDepth)
	if err != nil {
		return callingBook, err
	}

	enabledPairs, err := p.GetEnabledPairs(assetType)
	if err != nil {
		return callingBook, err
	}
	for i := range enabledPairs {
		book := &orderbook.Base{
			Exchange:        p.Name,
			Pair:            enabledPairs[i],
			Asset:           assetType,
			VerifyOrderbook: p.CanVerifyOrderbook,
		}

		fpair, err := p.FormatExchangeCurrency(enabledPairs[i], assetType)
		if err != nil {
			return book, err
		}
		data, ok := orderbookNew.Data[fpair.String()]
		if !ok {
			continue
		}

		for y := range data.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: data.Bids[y].Amount,
				Price:  data.Bids[y].Price,
			})
		}

		for y := range data.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: data.Asks[y].Amount,
				Price:  data.Asks[y].Price,
			})
		}
		err = book.Process()
		if err != nil {
			return book, err
		}
	}
	return orderbook.Get(p.Name, c, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (p *Poloniex) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = p.Name
	accountBalance, err := p.GetBalances(ctx)
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for x, y := range accountBalance.Currency {
		currencies = append(currencies, account.Balance{
			CurrencyName: currency.NewCode(x),
			Total:        y,
		})
	}

	response.Accounts = append(response.Accounts, account.SubAccount{
		Currencies: currencies,
	})

	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (p *Poloniex) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(p.Name, assetType)
	if err != nil {
		return p.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (p *Poloniex) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (p *Poloniex) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (p *Poloniex) GetRecentTrades(ctx context.Context, pair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return p.GetHistoricTrades(ctx, pair, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (p *Poloniex) GetHistoricTrades(ctx context.Context, pair currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	pair, err = p.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	ts := timestampStart
allTrades:
	for {
		var tradeData []TradeHistory
		tradeData, err = p.GetTradeHistory(ctx,
			pair.String(),
			ts.Unix(),
			timestampEnd.Unix())
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			var tt time.Time
			tt, err = time.Parse(common.SimpleTimeFormat, tradeData[i].Date)
			if err != nil {
				return nil, err
			}
			if (tt.Before(timestampStart) && !timestampStart.IsZero()) || (tt.After(timestampEnd) && !timestampEnd.IsZero()) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Type)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				TID:          strconv.FormatInt(tradeData[i].TradeID, 10),
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Rate,
				Amount:       tradeData[i].Amount,
				Timestamp:    tt,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tt) {
					// reached end of trades to crawl
					break allTrades
				}
				if timestampStart.IsZero() {
					break allTrades
				}
				ts = tt
			}
		}
	}

	err = p.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	resp = trade.FilterTradesByTime(resp, timestampStart, timestampEnd)

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (p *Poloniex) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}

	fPair, err := p.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	fillOrKill := s.Type == order.Market
	isBuyOrder := s.Side == order.Buy
	response, err := p.PlaceOrder(ctx,
		fPair.String(),
		s.Price,
		s.Amount,
		false,
		fillOrKill,
		isBuyOrder)
	if err != nil {
		return submitOrderResponse, err
	}
	if response.OrderNumber > 0 {
		submitOrderResponse.OrderID = strconv.FormatInt(response.OrderNumber, 10)
	}

	submitOrderResponse.IsOrderPlaced = true
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (p *Poloniex) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	if err := action.Validate(); err != nil {
		return order.Modify{}, err
	}

	oID, err := strconv.ParseInt(action.ID, 10, 64)
	if err != nil {
		return order.Modify{}, err
	}

	resp, err := p.MoveOrder(ctx,
		oID,
		action.Price,
		action.Amount,
		action.PostOnly,
		action.ImmediateOrCancel)
	if err != nil {
		return order.Modify{}, err
	}

	return order.Modify{
		Exchange:  action.Exchange,
		AssetType: action.AssetType,
		Pair:      action.Pair,
		ID:        strconv.FormatInt(resp.OrderNumber, 10),

		Price:             action.Price,
		Amount:            action.Amount,
		PostOnly:          action.PostOnly,
		ImmediateOrCancel: action.ImmediateOrCancel,
	}, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (p *Poloniex) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	return p.CancelExistingOrder(ctx, orderIDInt)
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (p *Poloniex) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (p *Poloniex) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	openOrders, err := p.GetOpenOrdersForAllCurrencies(ctx)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for key := range openOrders.Data {
		for i := range openOrders.Data[key] {
			err = p.CancelExistingOrder(ctx, openOrders.Data[key][i].OrderNumber)
			if err != nil {
				id := strconv.FormatInt(openOrders.Data[key][i].OrderNumber, 10)
				cancelAllOrdersResponse.Status[id] = err.Error()
			}
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (p *Poloniex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	orderInfo := order.Detail{
		Exchange: p.Name,
		Pair:     pair,
	}

	trades, err := p.GetAuthenticatedOrderTrades(ctx, orderID)
	if err != nil && !strings.Contains(err.Error(), "Order not found") {
		return orderInfo, err
	}

	for i := range trades {
		var tradeHistory order.TradeHistory
		tradeHistory.Exchange = p.Name
		tradeHistory.Side, err = order.StringToOrderSide(trades[i].Type)
		if err != nil {
			return orderInfo, err
		}
		tradeHistory.TID = strconv.FormatInt(trades[i].GlobalTradeID, 10)
		tradeHistory.Timestamp, err = time.Parse(common.SimpleTimeFormat, trades[i].Date)
		if err != nil {
			return orderInfo, err
		}
		tradeHistory.Price = trades[i].Rate
		tradeHistory.Amount = trades[i].Amount
		tradeHistory.Total = trades[i].Total
		tradeHistory.Fee = trades[i].Fee
		orderInfo.Trades = append(orderInfo.Trades, tradeHistory)
	}

	resp, err := p.GetAuthenticatedOrderStatus(ctx, orderID)
	if err != nil {
		if len(orderInfo.Trades) > 0 { // on closed orders return trades only
			if strings.Contains(err.Error(), "Order not found") {
				orderInfo.Status = order.Closed
			}
			return orderInfo, nil
		}
		return orderInfo, err
	}

	if orderInfo.Status, err = order.StringToOrderStatus(resp.Status); err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", p.Name, err)
	}
	orderInfo.Price = resp.Rate
	orderInfo.Amount = resp.Amount
	orderInfo.Cost = resp.Total
	orderInfo.Fee = resp.Fee
	orderInfo.QuoteAmount = resp.StartingAmount

	orderInfo.Side, err = order.StringToOrderSide(resp.Type)
	if err != nil {
		return orderInfo, err
	}

	orderInfo.Date, err = time.Parse(common.SimpleTimeFormat, resp.Date)
	if err != nil {
		return orderInfo, err
	}

	return orderInfo, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (p *Poloniex) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	depositAddrs, err := p.GetDepositAddresses(ctx)
	if err != nil {
		return nil, err
	}

	// Some coins use a main address, so we must use this in conjunction with the returned
	// deposit address to produce the full deposit address and tag
	currencies, err := p.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	coinParams, ok := currencies[cryptocurrency.Upper().String()]
	if !ok {
		return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
	}

	// Handle coins with payment ID's like XRP
	var address, tag string
	if coinParams.CurrencyType == "address-payment-id" && coinParams.DepositAddress != "" {
		address = coinParams.DepositAddress
		tag, ok = depositAddrs.Addresses[cryptocurrency.Upper().String()]
		if !ok {
			newAddr, err := p.GenerateNewAddress(ctx, cryptocurrency.Upper().String())
			if err != nil {
				return nil, err
			}
			tag = newAddr
		}
		return &deposit.Address{
			Address: address,
			Tag:     tag,
		}, nil
	}

	// Handle coins like BTC or multichain coins
	targetCurrency := cryptocurrency.String()
	if chain != "" && !strings.EqualFold(chain, cryptocurrency.String()) {
		targetCurrency = chain
	}

	address, ok = depositAddrs.Addresses[strings.ToUpper(targetCurrency)]
	if !ok {
		if len(coinParams.ChildChains) > 1 && chain != "" && !common.StringDataCompare(coinParams.ChildChains, targetCurrency) {
			// rather than assume, return an error
			return nil, fmt.Errorf("currency %s has %v chains available, one of these must be specified",
				cryptocurrency,
				coinParams.ChildChains)
		}

		coinParams, ok = currencies[strings.ToUpper(targetCurrency)]
		if !ok {
			return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
		}
		if coinParams.WithdrawalDepositDisabled == 1 {
			return nil, fmt.Errorf("deposits and withdrawals for %v are currently disabled", targetCurrency)
		}

		newAddr, err := p.GenerateNewAddress(ctx, targetCurrency)
		if err != nil {
			return nil, err
		}
		address = newAddr
	}
	return &deposit.Address{Address: address}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (p *Poloniex) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}

	targetCurrency := withdrawRequest.Currency.String()
	if withdrawRequest.Crypto.Chain != "" {
		targetCurrency = withdrawRequest.Crypto.Chain
	}
	v, err := p.Withdraw(ctx, targetCurrency, withdrawRequest.Crypto.Address, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v.Response,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!p.AreCredentialsValid(ctx) || p.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return p.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (p *Poloniex) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := p.GetOpenOrdersForAllCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	format, err := p.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key := range resp.Data {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}
		for i := range resp.Data[key] {
			orderSide := order.Side(strings.ToUpper(resp.Data[key][i].Type))
			orderDate, err := time.Parse(common.SimpleTimeFormat, resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name,
					"GetActiveOrders",
					resp.Data[key][i].OrderNumber,
					resp.Data[key][i].Date)
			}

			orders = append(orders, order.Detail{
				ID:       strconv.FormatInt(resp.Data[key][i].OrderNumber, 10),
				Side:     orderSide,
				Amount:   resp.Data[key][i].Amount,
				Date:     orderDate,
				Price:    resp.Data[key][i].Rate,
				Pair:     symbol,
				Exchange: p.Name,
			})
		}
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersBySide(&orders, req.Side)

	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (p *Poloniex) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := p.GetAuthenticatedTradeHistory(ctx,
		req.StartTime.Unix(),
		req.EndTime.Unix(),
		10000)
	if err != nil {
		return nil, err
	}

	format, err := p.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for key := range resp.Data {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(key, format.Delimiter)
		if err != nil {
			return nil, err
		}

		for i := range resp.Data[key] {
			orderSide := order.Side(strings.ToUpper(resp.Data[key][i].Type))
			orderDate, err := time.Parse(common.SimpleTimeFormat,
				resp.Data[key][i].Date)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse date to unix with value of %v",
					p.Name,
					"GetActiveOrders",
					resp.Data[key][i].OrderNumber,
					resp.Data[key][i].Date)
			}

			detail := order.Detail{
				ID:                   strconv.FormatInt(resp.Data[key][i].GlobalTradeID, 10),
				Side:                 orderSide,
				Amount:               resp.Data[key][i].Amount,
				ExecutedAmount:       resp.Data[key][i].Amount,
				Date:                 orderDate,
				Price:                resp.Data[key][i].Rate,
				AverageExecutedPrice: resp.Data[key][i].Rate,
				Pair:                 pair,
				Status:               order.Filled,
				Exchange:             p.Name,
			}
			detail.InferCostsAndTimes()
			orders = append(orders, detail)
		}
	}

	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	order.FilterOrdersBySide(&orders, req.Side)

	return orders, nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (p *Poloniex) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := p.UpdateAccountInfo(ctx, assetType)
	return p.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := p.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := p.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	// we use Truncate here to round star to the nearest even number that matches the requested interval
	// example 10:17 with an interval of 15 minutes will go down 10:15
	// this is due to poloniex returning a non-complete candle if the time does not match
	candles, err := p.GetChartData(ctx,
		formattedPair.String(),
		start.Truncate(interval.Duration()), end,
		p.FormatExchangeKlineInterval(interval))
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: p.Name,
		Interval: interval,
		Pair:     pair,
		Asset:    a,
	}

	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   time.Unix(candles[x].Date, 0),
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		})
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return p.GetHistoricCandles(ctx, pair, a, start, end, interval)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (p *Poloniex) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	currencies, err := p.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	curr, ok := currencies[cryptocurrency.Upper().String()]
	if !ok {
		return nil, errors.New("unable to locate currency in map")
	}

	return curr.ChildChains, nil
}
