package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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

// SetDefaults sets the basic defaults for bitfinex
func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.MarginFunding} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		if a == asset.Margin {
			ps.ConfigFormat.Delimiter = ":"
		}
		if err := b.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", b.Name, a, err)
		}
	}

	// Margin WS Currently not fully implemented and causes subscription collisions with spot
	if err := b.DisableAssetWebsocketSupport(asset.Margin); err != nil {
		log.Errorf(log.ExchangeSys, "%s error disabling %q asset type websocket support: %s", b.Name, asset.Margin, err)
	}

	// TODO: Implement Futures and Securities asset types.

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                    true,
				TickerFetching:                    true,
				OrderbookFetching:                 true,
				AutoPairUpdates:                   true,
				AccountInfo:                       true,
				CryptoDeposit:                     true,
				CryptoWithdrawal:                  true,
				FiatWithdraw:                      true,
				GetOrder:                          true,
				GetOrders:                         true,
				CancelOrders:                      true,
				CancelOrder:                       true,
				SubmitOrder:                       true,
				SubmitOrders:                      true,
				DepositHistory:                    true,
				WithdrawalHistory:                 true,
				TradeFetching:                     true,
				UserTradeHistory:                  true,
				TradeFee:                          true,
				FiatDepositFee:                    true,
				FiatWithdrawalFee:                 true,
				CryptoDepositFee:                  true,
				CryptoWithdrawalFee:               true,
				MultiChainDeposits:                true,
				MultiChainWithdrawals:             true,
				MultiChainDepositRequiresChainSet: true,
				FundingRateFetching:               true,
			},
			WebsocketCapabilities: protocol.Features{
				AccountBalance:         true,
				CancelOrders:           true,
				CancelOrder:            true,
				SubmitOrder:            true,
				ModifyOrder:            true,
				TickerFetching:         true,
				KlineFetching:          true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				AccountInfo:            true,
				Subscribe:              true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				DeadMansSwitch:         true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.AutoWithdrawFiatWithAPIPermission,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.Margin: true,
				},
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.ThreeHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.TwoWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 10000,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      bitfinexAPIURLBase,
		exchange.WebsocketSpot: publicBitfinexWebsocketEndpoint,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.Websocket = websocket.NewManager()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitfinex) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	err = b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsEndpoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            publicBitfinexWebsocketEndpoint,
		RunningURL:            wsEndpoint,
		Connector:             b.WsConnect,
		Subscriber:            b.Subscribe,
		Unsubscriber:          b.Unsubscribe,
		GenerateSubscriptions: b.generateSubscriptions,
		Features:              &b.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	err = b.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  publicBitfinexWebsocketEndpoint,
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  authenticatedBitfinexWebsocketEndpoint,
		Authenticated:        true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitfinex) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	items, err := b.GetPairs(ctx, a)
	if err != nil {
		return nil, err
	}

	pairs := make(currency.Pairs, 0, len(items))
	for x := range items {
		if strings.Contains(items[x], "TEST") {
			continue
		}

		var pair currency.Pair
		if a == asset.MarginFunding {
			pair, err = currency.NewPairFromStrings(items[x], "")
		} else {
			pair, err = currency.NewPairFromString(items[x])
		}
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitfinex) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := b.CurrencyPairs.GetAssetTypes(false)
	for i := range assets {
		pairs, err := b.FetchTradablePairs(ctx, assets[i])
		if err != nil {
			return err
		}

		err = b.UpdatePairs(pairs, assets[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return b.EnsureOnePairEnabled()
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (b *Bitfinex) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return common.ErrNotYetImplemented
	}
	limits, err := b.GetSiteInfoConfigData(ctx, a)
	if err != nil {
		return err
	}
	if err := b.LoadLimits(limits); err != nil {
		return fmt.Errorf("%s Error loading exchange limits: %v", b.Name, err)
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bitfinex) UpdateTickers(ctx context.Context, a asset.Item) error {
	t, err := b.GetTickerBatch(ctx)
	if err != nil {
		return err
	}

	var errs error
	for key, val := range t {
		pair, enabled, err := b.MatchSymbolCheckEnabled(key[1:], a, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			errs = common.AppendError(errs, err)
			continue
		}
		if !enabled {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Last:         val.Last,
			High:         val.High,
			Low:          val.Low,
			Bid:          val.Bid,
			Ask:          val.Ask,
			Volume:       val.Volume,
			Pair:         pair,
			AssetType:    a,
			ExchangeName: b.Name,
		})
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitfinex) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := b.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(b.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitfinex) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	o := &orderbook.Book{
		Exchange:          b.Name,
		Pair:              p,
		Asset:             assetType,
		PriceDuplication:  true,
		ValidateOrderbook: b.ValidateOrderbook,
	}

	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return o, err
	}
	if assetType != asset.Spot && assetType != asset.Margin && assetType != asset.MarginFunding {
		return o, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	b.appendOptionalDelimiter(&fPair)
	prefix := "t"
	if assetType == asset.MarginFunding {
		prefix = "f"
	}

	orderbookNew, err := b.GetOrderbook(ctx, prefix+fPair.String(), "R0", 100)
	if err != nil {
		return o, err
	}
	if assetType == asset.MarginFunding {
		o.IsFundingRate = true
		o.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			o.Asks[x] = orderbook.Level{
				ID:     orderbookNew.Asks[x].OrderID,
				Price:  orderbookNew.Asks[x].Rate,
				Amount: orderbookNew.Asks[x].Amount,
				Period: int64(orderbookNew.Asks[x].Period),
			}
		}
		o.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			o.Bids[x] = orderbook.Level{
				ID:     orderbookNew.Bids[x].OrderID,
				Price:  orderbookNew.Bids[x].Rate,
				Amount: orderbookNew.Bids[x].Amount,
				Period: int64(orderbookNew.Bids[x].Period),
			}
		}
	} else {
		o.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			o.Asks[x] = orderbook.Level{
				ID:     orderbookNew.Asks[x].OrderID,
				Price:  orderbookNew.Asks[x].Price,
				Amount: orderbookNew.Asks[x].Amount,
			}
		}
		o.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			o.Bids[x] = orderbook.Level{
				ID:     orderbookNew.Bids[x].OrderID,
				Price:  orderbookNew.Bids[x].Price,
				Amount: orderbookNew.Bids[x].Amount,
			}
		}
	}
	err = o.Process()
	if err != nil {
		return nil, err
	}
	return orderbook.Get(b.Name, fPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies on the
// Bitfinex exchange
func (b *Bitfinex) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = b.Name

	accountBalance, err := b.GetAccountBalance(ctx)
	if err != nil {
		return response, err
	}

	Accounts := []account.SubAccount{
		{ID: "deposit", AssetType: assetType},
		{ID: "exchange", AssetType: assetType},
		{ID: "trading", AssetType: assetType},
		{ID: "margin", AssetType: assetType},
		{ID: "funding", AssetType: assetType},
	}

	for x := range accountBalance {
		for i := range Accounts {
			if Accounts[i].ID == accountBalance[x].Type {
				Accounts[i].Currencies = append(Accounts[i].Currencies,
					account.Balance{
						Currency: currency.NewCode(accountBalance[x].Currency),
						Total:    accountBalance[x].Amount,
						Hold:     accountBalance[x].Amount - accountBalance[x].Available,
						Free:     accountBalance[x].Available,
					})
			}
		}
	}

	response.Accounts = Accounts
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitfinex) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitfinex) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	history, err := b.GetMovementHistory(ctx, c.String(), "", time.Date(2012, 0, 0, 0, 0, 0, 0, time.Local), time.Now(), 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(history))
	for i := range history {
		resp[i] = exchange.WithdrawalHistory{
			Status:          history[i].Status,
			TransferID:      strconv.FormatInt(history[i].ID, 10),
			Description:     *history[i].TransactionID,
			Timestamp:       history[i].MTSStarted.Time(),
			Currency:        history[i].Currency,
			Amount:          history[i].Amount.Float64(),
			Fee:             history[i].Fees.Float64(),
			TransferType:    history[i].TransactionType,
			CryptoToAddress: history[i].DestinationAddress,
			CryptoTxID:      history[i].TXID,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Bitfinex) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return b.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bitfinex) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if a == asset.MarginFunding {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	p, err := b.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	currString, err := b.fixCasing(p, a)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	ts := timestampEnd
	const limit = 10000
allTrades:
	for {
		tradeData, err := b.GetTrades(ctx, currString, limit, time.Time{}, ts, false)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			tradeTS := tradeData[i].Timestamp.Time()
			if tradeTS.Before(timestampStart) && !timestampStart.IsZero() {
				break allTrades
			}
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(tradeData[i].TID, 10),
				Exchange:     b.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Amount,
				Timestamp:    tradeData[i].Timestamp.Time(),
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tradeTS) {
					// reached end of trades to crawl
					break allTrades
				}
				ts = tradeTS
			}
		}
		if len(tradeData) != limit {
			break allTrades
		}
	}

	if err := b.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (b *Bitfinex) SubmitOrder(ctx context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	if err := o.Validate(b.GetTradingRequirements()); err != nil {
		return nil, err
	}

	fPair, err := b.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return nil, err
	}

	var orderID string
	status := order.New
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var symbolStr string
		if symbolStr, err = b.fixCasing(fPair, o.AssetType); err != nil {
			return nil, err
		}
		orderType := strings.ToUpper(o.Type.String())
		if o.AssetType == asset.Spot {
			orderType = "EXCHANGE " + orderType
		}
		req := &WsNewOrderRequest{
			Type:   orderType,
			Symbol: symbolStr,
			Amount: o.Amount,
			Price:  o.Price,
		}
		if o.Side.IsShort() && o.Amount > 0 {
			// All v2 apis use negatives for Short side
			req.Amount *= -1
		}
		orderID, err = b.WsNewOrder(ctx, req)
		if err != nil {
			return nil, err
		}
	} else {
		var response Order
		b.appendOptionalDelimiter(&fPair)
		orderType := o.Type.Lower()
		if o.AssetType == asset.Spot {
			orderType = "exchange " + orderType
		}
		response, err = b.NewOrder(ctx,
			fPair.String(),
			orderType,
			o.Amount,
			o.Price,
			o.Side.IsLong(),
			false)
		if err != nil {
			return nil, err
		}
		orderID = strconv.FormatInt(response.ID, 10)

		if response.RemainingAmount == 0 {
			status = order.Filled
		}
	}
	resp, err := o.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = status
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitfinex) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}

	if b.Websocket.IsEnabled() && b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		orderIDInt, err := strconv.ParseInt(action.OrderID, 10, 64)
		if err != nil {
			return &order.ModifyResponse{OrderID: action.OrderID}, err
		}

		wsRequest := WsUpdateOrderRequest{
			OrderID: orderIDInt,
			Price:   action.Price,
			Amount:  action.Amount,
		}
		if action.Side.IsShort() && action.Amount > 0 {
			wsRequest.Amount *= -1
		}
		err = b.WsModifyOrder(ctx, &wsRequest)
		if err != nil {
			return nil, err
		}
		return action.DeriveModifyResponse()
	}

	_, err := b.OrderUpdate(ctx, action.OrderID, "", action.ClientOrderID, action.Amount, action.Price, -1)
	if err != nil {
		return nil, err
	}
	return action.DeriveModifyResponse()
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitfinex) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		err = b.WsCancelOrder(ctx, orderIDInt)
	} else {
		_, err = b.CancelExistingOrder(ctx, orderIDInt)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Bitfinex) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	// While bitfinex supports cancelling multiple orders, it is
	// done in a way that is not helpful for GCT, and it would be better instead
	// to use CancelAllOrders or CancelOrder
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitfinex) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	var err error
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		err = b.WsCancelAllOrders(ctx)
	} else {
		_, err = b.CancelAllExistingOrders(ctx)
	}
	return order.CancelAllResponse{}, err
}

func (b *Bitfinex) parseOrderToOrderDetail(o *Order) (*order.Detail, error) {
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return nil, err
	}

	orderDetail := &order.Detail{
		Amount:          o.OriginalAmount,
		Date:            o.Timestamp.Time(),
		Exchange:        b.Name,
		OrderID:         strconv.FormatInt(o.ID, 10),
		Side:            side,
		Price:           o.Price,
		RemainingAmount: o.RemainingAmount,
		Pair:            o.Symbol,
		ExecutedAmount:  o.ExecutedAmount,
	}

	switch {
	case o.IsLive:
		orderDetail.Status = order.Active
	case o.IsCancelled:
		orderDetail.Status = order.Cancelled
	case o.IsHidden:
		orderDetail.Status = order.Hidden
	default:
		orderDetail.Status = order.UnknownStatus
	}

	// API docs discrepancy. Example contains prefixed "exchange "
	// Return type suggests “market” / “limit” / “stop” / “trailing-stop”
	orderType := strings.Replace(o.Type, "exchange ", "", 1)
	if orderType == "trailing-stop" {
		orderDetail.Type = order.TrailingStop
	} else {
		orderDetail.Type, err = order.StringToOrderType(orderType)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}
	}

	return orderDetail, nil
}

// GetOrderInfo returns order information based on order ID
func (b *Bitfinex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}

	b.appendOptionalDelimiter(&pair)
	var cf string
	cf, err = b.fixCasing(pair, assetType)
	if err != nil {
		return nil, err
	}

	resp, err := b.GetInactiveOrders(ctx, cf, id)
	if err != nil {
		return nil, err
	}
	for i := range resp {
		if resp[i].OrderID != id {
			continue
		}
		var o *order.Detail
		o, err = b.parseOrderToOrderDetail(&resp[i])
		if err != nil {
			return nil, err
		}
		return o, nil
	}
	resp, err = b.GetOpenOrders(ctx, id)
	if err != nil {
		return nil, err
	}
	for i := range resp {
		if resp[i].OrderID != id {
			continue
		}
		var o *order.Detail
		o, err = b.parseOrderToOrderDetail(&resp[i])
		if err != nil {
			return nil, err
		}
		return o, nil
	}
	return nil, fmt.Errorf("%w %v", order.ErrOrderNotFound, orderID)
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitfinex) GetDepositAddress(ctx context.Context, c currency.Code, accountID, chain string) (*deposit.Address, error) {
	if accountID == "" {
		accountID = "funding"
	}

	if c.Equal(currency.USDT) {
		// USDT is UST on Bitfinex
		c = currency.NewCode("UST")
	}

	if err := b.PopulateAcceptableMethods(ctx); err != nil {
		return nil, err
	}

	methods := acceptableMethods.lookup(c)
	if len(methods) == 0 {
		return nil, currency.ErrCurrencyNotSupported
	}
	method := methods[0]
	if len(methods) > 1 && chain != "" {
		method = chain
	} else if len(methods) > 1 && chain == "" {
		return nil, fmt.Errorf("a chain must be specified, %s available", methods)
	}

	resp, err := b.NewDeposit(ctx, method, accountID, 0)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: resp.Address,
		Tag:     resp.PoolAddress,
	}, err
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (b *Bitfinex) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}

	if err := b.PopulateAcceptableMethods(ctx); err != nil {
		return nil, err
	}

	tmpCurr := withdrawRequest.Currency
	if tmpCurr.Equal(currency.USDT) {
		// USDT is UST on Bitfinex
		tmpCurr = currency.NewCode("UST")
	}

	methods := acceptableMethods.lookup(tmpCurr)
	if len(methods) == 0 {
		return nil, errors.New("no transfer methods returned for currency")
	}
	method := methods[0]
	if len(methods) > 1 && withdrawRequest.Crypto.Chain != "" {
		if !common.StringSliceCompareInsensitive(methods, withdrawRequest.Crypto.Chain) {
			return nil, fmt.Errorf("invalid chain %s supplied, %v available", withdrawRequest.Crypto.Chain, methods)
		}
		method = withdrawRequest.Crypto.Chain
	} else if len(methods) > 1 && withdrawRequest.Crypto.Chain == "" {
		return nil, fmt.Errorf("a chain must be specified, %s available", methods)
	}

	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawCryptocurrency(ctx,
		walletType,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		method,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID:     strconv.FormatInt(resp.WithdrawalID, 10),
		Status: resp.Status,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
// Returns comma delimited withdrawal IDs
func (b *Bitfinex) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawalType := "wire"
	// Bitfinex has support for three types, exchange, margin and deposit
	// As this is for trading, I've made the wrapper default 'exchange'
	// TODO: Discover an automated way to make the decision for wallet type to withdraw from
	walletType := "exchange"
	resp, err := b.WithdrawFIAT(ctx, withdrawalType, walletType, withdrawRequest)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID:     strconv.FormatInt(resp.WithdrawalID, 10),
		Status: resp.Status,
	}, err
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
// Returns comma delimited withdrawal IDs
func (b *Bitfinex) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := b.WithdrawFiatFunds(ctx, withdrawRequest)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     v.ID,
		Status: v.Status,
	}, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitfinex) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !b.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitfinex) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	resp, err := b.GetOpenOrders(ctx)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(resp))
	for i := range resp {
		var orderDetail *order.Detail
		orderDetail, err = b.parseOrderToOrderDetail(&resp[i])
		if err != nil {
			return nil, err
		}
		orders[i] = *orderDetail
	}
	return req.Filter(b.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitfinex) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range req.Pairs {
		b.appendOptionalDelimiter(&req.Pairs[i])
		var cf string
		cf, err = b.fixCasing(req.Pairs[i], req.AssetType)
		if err != nil {
			return nil, err
		}

		var resp []Order
		resp, err = b.GetInactiveOrders(ctx, cf)
		if err != nil {
			return nil, err
		}

		for j := range resp {
			var orderDetail *order.Detail
			orderDetail, err = b.parseOrderToOrderDetail(&resp[j])
			if err != nil {
				return nil, err
			}
			orders = append(orders, *orderDetail)
		}
	}

	return req.Filter(b.Name, orders), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Bitfinex) AuthenticateWebsocket(ctx context.Context) error {
	return b.WsSendAuth(ctx)
}

// appendOptionalDelimiter ensures that a delimiter is present for long character currencies
func (b *Bitfinex) appendOptionalDelimiter(p *currency.Pair) {
	if (len(p.Base.String()) > 3 && !p.Quote.IsEmpty()) ||
		len(p.Quote.String()) > 3 {
		p.Delimiter = ":"
	}
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (b *Bitfinex) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (b *Bitfinex) FormatExchangeKlineInterval(in kline.Interval) (string, error) {
	switch in {
	case kline.OneMin:
		return "1m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.ThreeHour:
		return "3h", nil
	case kline.SixHour:
		return "6h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1D", nil
	case kline.OneWeek:
		return "7D", nil
	case kline.OneWeek * 2:
		return "14D", nil
	case kline.OneMonth:
		return "1M", nil
	default:
		return "", fmt.Errorf("%w %v", kline.ErrInvalidInterval, in)
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitfinex) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	cf, err := b.fixCasing(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	fInterval, err := b.FormatExchangeKlineInterval(req.ExchangeInterval)
	if err != nil {
		return nil, err
	}
	candles, err := b.GetCandles(ctx, cf, fInterval, req.Start, req.End, req.RequestLimit, true)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].Timestamp.Time(),
			Open:   candles[x].Open.Float64(),
			High:   candles[x].High.Float64(),
			Low:    candles[x].Low.Float64(),
			Close:  candles[x].Close.Float64(),
			Volume: candles[x].Volume.Float64(),
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *Bitfinex) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	cf, err := b.fixCasing(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	fInterval, err := b.FormatExchangeKlineInterval(req.ExchangeInterval)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles []Candle
		candles, err = b.GetCandles(ctx, cf, fInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, req.RequestLimit, true)
		if err != nil {
			return nil, err
		}

		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].Timestamp.Time(),
				Open:   candles[i].Open.Float64(),
				High:   candles[i].High.Float64(),
				Low:    candles[i].Low.Float64(),
				Close:  candles[i].Close.Float64(),
				Volume: candles[i].Volume.Float64(),
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

func (b *Bitfinex) fixCasing(in currency.Pair, a asset.Item) (string, error) {
	if in.Base.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}

	// Convert input to lowercase to ensure consistent formatting.
	// Required for currencies that start with T or F eg tTNBUSD
	in = in.Lower()

	var checkString [2]byte
	switch a {
	case asset.Spot, asset.Margin:
		checkString[0] = 't'
		checkString[1] = 'T'
	case asset.MarginFunding:
		checkString[0] = 'f'
		checkString[1] = 'F'
	}

	cFmt, err := b.FormatExchangeCurrency(in, a)
	if err != nil {
		return "", err
	}

	y := in.Base.String()
	if (y[0] != checkString[0] && y[0] != checkString[1]) ||
		(y[0] == checkString[1] && y[1] == checkString[1]) || in.Base.Equal(currency.TNB) {
		if cFmt.Quote.IsEmpty() {
			return string(checkString[0]) + cFmt.Base.Upper().String(), nil
		}
		return string(checkString[0]) + cFmt.Upper().String(), nil
	}

	runes := []rune(cFmt.Upper().String())
	if cFmt.Quote.IsEmpty() {
		runes = []rune(cFmt.Base.Upper().String())
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes), nil
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (b *Bitfinex) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if err := b.PopulateAcceptableMethods(ctx); err != nil {
		return nil, err
	}

	if cryptocurrency.Equal(currency.USDT) {
		// USDT is UST on Bitfinex
		cryptocurrency = currency.NewCode("UST")
	}

	availChains := acceptableMethods.lookup(cryptocurrency)
	if len(availChains) == 0 {
		return nil, errors.New("unable to find any available chains")
	}
	return availChains, nil
}

// GetServerTime returns the current exchange server time.
func (b *Bitfinex) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (b *Bitfinex) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (b *Bitfinex) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	// TODO: Add futures support for Bitfinex
	return nil, common.ErrNotYetImplemented
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (b *Bitfinex) GetOpenInterest(context.Context, ...key.PairAsset) ([]futures.OpenInterest, error) {
	// TODO: Add futures support for Bitfinex
	return nil, common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *Bitfinex) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := b.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	symbol, err := b.FormatSymbol(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Margin, asset.MarginFunding:
		return tradeBaseURL + "/f/" + symbol, nil
	case asset.Spot:
		return tradeBaseURL + "/t/" + symbol, nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
