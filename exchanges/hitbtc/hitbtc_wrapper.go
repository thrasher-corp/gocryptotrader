package hitbtc

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
func (h *HitBTC) GetDefaultConfig() (*config.Exchange, error) {
	h.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = h.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = h.BaseCurrencies

	err := h.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if h.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = h.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default settings for hitbtc
func (h *HitBTC) SetDefaults() {
	h.Name = "HitBTC"
	h.Enabled = true
	h.Verbose = true
	h.API.CredentialsValidator.RequiresKey = true
	h.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := h.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	h.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
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
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				ModifyOrder:         true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				SubmitOrder:            true,
				CancelOrder:            true,
				MessageSequenceNumbers: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals:  true,
				DateRanges: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():    true,
					kline.ThreeMin.Word():  true,
					kline.FiveMin.Word():   true,
					kline.ThirtyMin.Word(): true,
					kline.OneHour.Word():   true,
					kline.FourHour.Word():  true,
					kline.OneDay.Word():    true,
					kline.SevenDay.Word():  true,
				},
				ResultLimit: 1000,
			},
		},
	}

	h.Requester, err = request.New(h.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	h.API.Endpoints = h.NewEndpoints()
	err = h.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      apiURL,
		exchange.WebsocketSpot: hitbtcWebsocketAddress,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	h.Websocket = stream.New()
	h.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	h.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	h.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (h *HitBTC) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		h.SetEnabled(false)
		return nil
	}
	err = h.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := h.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = h.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            hitbtcWebsocketAddress,
		RunningURL:            wsRunningURL,
		Connector:             h.WsConnect,
		Subscriber:            h.Subscribe,
		Unsubscriber:          h.Unsubscribe,
		GenerateSubscriptions: h.GenerateDefaultSubscriptions,
		Features:              &h.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	return h.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the HitBTC go routine
func (h *HitBTC) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the HitBTC wrapper
func (h *HitBTC) Run() {
	if h.Verbose {
		log.Debugf(log.ExchangeSys, "%s Websocket: %s (url: %s).\n", h.Name, common.IsEnabled(h.Websocket.IsEnabled()), hitbtcWebsocketAddress)
		h.PrintEnabledPairs()
	}

	forceUpdate := false
	if !h.BypassConfigFormatUpgrades {
		format, err := h.GetPairFormat(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update tradable pairs. Err: %s",
				h.Name,
				err)
			return
		}
		enabled, err := h.GetEnabledPairs(asset.Spot)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update tradable pairs. Err: %s",
				h.Name,
				err)
			return
		}

		avail, err := h.GetAvailablePairs(asset.Spot)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update tradable pairs. Err: %s",
				h.Name,
				err)
			return
		}

		if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
			!common.StringDataContains(avail.Strings(), format.Delimiter) {
			enabledPairs := []string{currency.BTC.String() + format.Delimiter + currency.USD.String()}
			log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, h.Name, asset.Spot, enabledPairs)
			forceUpdate = true
			var p currency.Pairs
			p, err = currency.NewPairsFromStrings(enabledPairs)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update tradable pairs. Err: %s",
					h.Name,
					err)
				return
			}
			err = h.UpdatePairs(p, asset.Spot, true, true)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update enabled currencies.\n",
					h.Name)
			}
		}
	}

	if !h.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := h.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			h.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (h *HitBTC) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	symbols, err := h.GetSymbolsDetailed(ctx)
	if err != nil {
		return nil, err
	}

	format, err := h.GetPairFormat(asset, false)
	if err != nil {
		return nil, err
	}

	var pairs []string
	for x := range symbols {
		pairs = append(pairs, symbols[x].BaseCurrency+
			format.Delimiter+
			symbols[x].QuoteCurrency)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (h *HitBTC) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := h.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return h.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (h *HitBTC) UpdateTickers(ctx context.Context, a asset.Item) error {
	tick, err := h.GetTickers(ctx)
	if err != nil {
		return err
	}
	pairs, err := h.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range pairs {
		for j := range tick {
			pairFmt, err := h.FormatExchangeCurrency(pairs[i], a)
			if err != nil {
				return err
			}

			if tick[j].Symbol != pairFmt.String() {
				found := false
				if strings.Contains(tick[j].Symbol, "USDT") {
					if pairFmt.String() == tick[j].Symbol[0:len(tick[j].Symbol)-1] {
						found = true
					}
				}
				if !found {
					continue
				}
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tick[j].Last,
				High:         tick[j].High,
				Low:          tick[j].Low,
				Bid:          tick[j].Bid,
				Ask:          tick[j].Ask,
				Volume:       tick[j].Volume,
				QuoteVolume:  tick[j].VolumeQuote,
				Open:         tick[j].Open,
				Pair:         pairs[i],
				LastUpdated:  tick[j].Timestamp,
				ExchangeName: h.Name,
				AssetType:    a})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (h *HitBTC) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := h.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(h.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (h *HitBTC) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(h.Name, p, assetType)
	if err != nil {
		return h.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (h *HitBTC) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(h.Name, p, assetType)
	if err != nil {
		return h.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (h *HitBTC) UpdateOrderbook(ctx context.Context, c currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        h.Name,
		Pair:            c,
		Asset:           assetType,
		VerifyOrderbook: h.CanVerifyOrderbook,
	}
	fpair, err := h.FormatExchangeCurrency(c, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := h.GetOrderbook(ctx, fpair.String(), 1000)
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
	return orderbook.Get(h.Name, c, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// HitBTC exchange
func (h *HitBTC) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = h.Name
	accountBalance, err := h.GetBalances(ctx)
	if err != nil {
		return response, err
	}

	var currencies []account.Balance
	for i := range accountBalance {
		currencies = append(currencies, account.Balance{
			CurrencyName: currency.NewCode(accountBalance[i].Currency),
			Total:        accountBalance[i].Available + accountBalance[i].Reserved,
			Hold:         accountBalance[i].Reserved,
			Free:         accountBalance[i].Available,
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
func (h *HitBTC) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(h.Name, assetType)
	if err != nil {
		return h.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (h *HitBTC) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (h *HitBTC) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (h *HitBTC) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return h.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (h *HitBTC) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = h.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	ts := timestampStart
	var resp []trade.Data
	limit := 1000
allTrades:
	for {
		var tradeData []TradeHistory
		tradeData, err = h.GetTrades(ctx,
			p.String(),
			"",
			"",
			ts.UnixMilli(),
			timestampEnd.UnixMilli(),
			int64(limit),
			0)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			if tradeData[i].Timestamp.Before(timestampStart) || tradeData[i].Timestamp.After(timestampEnd) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     h.Name,
				TID:          strconv.FormatInt(tradeData[i].ID, 10),
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Quantity,
				Timestamp:    tradeData[i].Timestamp,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tradeData[i].Timestamp) {
					// reached end of trades to crawl
					break allTrades
				}
				ts = tradeData[i].Timestamp
			}
		}
		if len(tradeData) != limit {
			break allTrades
		}
	}

	err = h.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (h *HitBTC) SubmitOrder(ctx context.Context, o *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	err := o.Validate()
	if err != nil {
		return submitOrderResponse, err
	}
	if h.Websocket.IsConnected() && h.Websocket.CanUseAuthenticatedEndpoints() {
		var response *WsSubmitOrderSuccessResponse
		response, err = h.wsPlaceOrder(o.Pair, o.Side.String(), o.Amount, o.Price)
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = strconv.FormatInt(response.ID, 10)
		if response.Result.CumQuantity == o.Amount {
			submitOrderResponse.FullyMatched = true
		}
	} else {
		fPair, err := h.FormatExchangeCurrency(o.Pair, o.AssetType)
		if err != nil {
			return submitOrderResponse, err
		}

		var response OrderResponse
		response, err = h.PlaceOrder(ctx,
			fPair.String(),
			o.Price,
			o.Amount,
			strings.ToLower(o.Type.String()),
			strings.ToLower(o.Side.String()))
		if err != nil {
			return submitOrderResponse, err
		}
		if response.OrderNumber > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response.OrderNumber, 10)
		}
		if o.Type == order.Market {
			submitOrderResponse.FullyMatched = true
		}
	}
	submitOrderResponse.IsOrderPlaced = true

	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (h *HitBTC) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (h *HitBTC) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
	if err != nil {
		return err
	}

	_, err = h.CancelExistingOrder(ctx, orderIDInt)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (h *HitBTC) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (h *HitBTC) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	resp, err := h.CancelAllExistingOrders(ctx)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range resp {
		if resp[i].Status != "canceled" {
			cancelAllOrdersResponse.Status[strconv.FormatInt(resp[i].ID, 10)] =
				fmt.Sprintf("Could not cancel order %v. Status: %v",
					resp[i].ID,
					resp[i].Status)
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (h *HitBTC) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (h *HitBTC) GetDepositAddress(ctx context.Context, currency currency.Code, _, _ string) (*deposit.Address, error) {
	resp, err := h.GetDepositAddresses(ctx, currency.String())
	if err != nil {
		return nil, err
	}

	return &deposit.Address{
		Address: resp.Address,
		Tag:     resp.PaymentID,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (h *HitBTC) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := h.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: common.IsEnabled(v),
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (h *HitBTC) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (h *HitBTC) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !h.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return h.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (h *HitBTC) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for i := range req.Pairs {
		resp, err := h.GetOpenOrders(ctx, req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := h.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range allOrders {
		var symbol currency.Pair
		symbol, err = currency.NewPairDelimiter(allOrders[i].Symbol,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		side := order.Side(strings.ToUpper(allOrders[i].Side))
		orders = append(orders, order.Detail{
			ID:       allOrders[i].ID,
			Amount:   allOrders[i].Quantity,
			Exchange: h.Name,
			Price:    allOrders[i].Price,
			Date:     allOrders[i].CreatedAt,
			Side:     side,
			Pair:     symbol,
		})
	}

	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (h *HitBTC) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("currency must be supplied")
	}

	var allOrders []OrderHistoryResponse
	for i := range req.Pairs {
		resp, err := h.GetOrders(ctx, req.Pairs[i].String())
		if err != nil {
			return nil, err
		}
		allOrders = append(allOrders, resp...)
	}

	format, err := h.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range allOrders {
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(allOrders[i].Symbol,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		side := order.Side(strings.ToUpper(allOrders[i].Side))
		status, err := order.StringToOrderStatus(allOrders[i].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", h.Name, err)
		}
		detail := order.Detail{
			ID:                   allOrders[i].ID,
			Amount:               allOrders[i].Quantity,
			ExecutedAmount:       allOrders[i].CumQuantity,
			RemainingAmount:      allOrders[i].Quantity - allOrders[i].CumQuantity,
			Exchange:             h.Name,
			Price:                allOrders[i].Price,
			AverageExecutedPrice: allOrders[i].AvgPrice,
			Date:                 allOrders[i].CreatedAt,
			LastUpdated:          allOrders[i].UpdatedAt,
			Side:                 side,
			Status:               status,
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
func (h *HitBTC) AuthenticateWebsocket(ctx context.Context) error {
	return h.wsLogin(ctx)
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (h *HitBTC) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := h.UpdateAccountInfo(ctx, assetType)
	return h.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (h *HitBTC) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneMin, kline.ThreeMin,
		kline.FiveMin, kline.FifteenMin, kline.ThirtyMin:
		return "M" + in.Short()[:len(in.Short())-1]
	case kline.OneDay:
		return "D1"
	case kline.SevenDay:
		return "D7"
	}
	return ""
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (h *HitBTC) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := h.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := h.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	data, err := h.GetCandles(ctx,
		formattedPair.String(),
		strconv.FormatInt(int64(h.Features.Enabled.Kline.ResultLimit), 10),
		h.FormatExchangeKlineInterval(interval),
		start,
		end)
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: h.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	for x := range data {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   data[x].Timestamp,
			Open:   data[x].Open,
			High:   data[x].Max,
			Low:    data[x].Min,
			Close:  data[x].Close,
			Volume: data[x].Volume,
		})
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (h *HitBTC) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := h.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: h.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	dates, err := kline.CalculateCandleDateRanges(start, end, interval, h.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	formattedPair, err := h.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	for y := range dates.Ranges {
		var data []ChartData
		data, err = h.GetCandles(ctx,
			formattedPair.String(),
			strconv.FormatInt(int64(h.Features.Enabled.Kline.ResultLimit), 10),
			h.FormatExchangeKlineInterval(interval),
			dates.Ranges[y].Start.Time,
			dates.Ranges[y].End.Time)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range data {
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   data[i].Timestamp,
				Open:   data[i].Open,
				High:   data[i].Max,
				Low:    data[i].Min,
				Close:  data[i].Close,
				Volume: data[i].Volume,
			})
		}
	}
	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", h.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
