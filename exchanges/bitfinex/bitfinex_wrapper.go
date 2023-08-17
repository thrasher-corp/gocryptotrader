package bitfinex

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

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
func (b *Bitfinex) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	b.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets the basic defaults for bitfinex
func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = true
	b.Verbose = true
	b.WebsocketSubdChannels = make(map[int]WebsocketChanInfo)
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}

	fmt2 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: ":"},
	}

	err := b.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.Margin, fmt2)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = b.StoreAssetPairFormat(asset.MarginFunding, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
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
	}

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
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
	b.Websocket = stream.New()
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

	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             publicBitfinexWebsocketEndpoint,
		RunningURL:             wsEndpoint,
		Connector:              b.WsConnect,
		Subscriber:             b.Subscribe,
		Unsubscriber:           b.Unsubscribe,
		GenerateSubscriptions:  b.GenerateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &b.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			UpdateEntriesByID: true,
		},
	})
	if err != nil {
		return err
	}

	err = b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  publicBitfinexWebsocketEndpoint,
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  authenticatedBitfinexWebsocketEndpoint,
		Authenticated:        true,
	})
}

// Start starts the Bitfinex go routine
func (b *Bitfinex) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the Bitfinex wrapper
func (b *Bitfinex) Run(ctx context.Context) {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()))
		b.PrintEnabledPairs()
	}

	if b.GetEnabledFeatures().AutoPairUpdates {
		if err := b.UpdateTradablePairs(ctx, false); err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update tradable pairs. Err: %s",
				b.Name,
				err)
		}
	}
	for _, a := range b.GetAssetTypes(true) {
		if err := b.UpdateOrderExecutionLimits(ctx, a); err != nil && err != common.ErrNotYetImplemented {
			log.Errorln(log.ExchangeSys, err.Error())
		}
	}

	err := b.UpdateTradablePairs(ctx, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
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
	enabled, err := b.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	tickerNew, err := b.GetTickerBatch(ctx)
	if err != nil {
		return err
	}

	for key, val := range tickerNew {
		pair, err := enabled.DeriveFrom(strings.Replace(key, ":", "", 1)[1:])
		if err != nil {
			// GetTickerBatch returns all pairs in call across all asset types.
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
			ExchangeName: b.Name})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitfinex) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := b.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(b.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitfinex) FetchTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	DFPair := fPair
	b.appendOptionalDelimiter(&DFPair)
	tick, err := ticker.GetTicker(b.Name, DFPair, a)
	if err != nil {
		return b.UpdateTicker(ctx, fPair, a)
	}
	return tick, nil
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitfinex) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	DFPair := fPair
	b.appendOptionalDelimiter(&DFPair)
	ob, err := orderbook.Get(b.Name, DFPair, assetType)
	if err != nil {
		return b.UpdateOrderbook(ctx, fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitfinex) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	o := &orderbook.Base{
		Exchange:         b.Name,
		Pair:             p,
		Asset:            assetType,
		PriceDuplication: true,
		VerifyOrderbook:  b.CanVerifyOrderbook,
	}

	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return o, err
	}
	if assetType != asset.Spot && assetType != asset.Margin && assetType != asset.MarginFunding {
		return o, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	b.appendOptionalDelimiter(&fPair)
	var prefix = "t"
	if assetType == asset.MarginFunding {
		prefix = "f"
	}

	orderbookNew, err := b.GetOrderbook(ctx, prefix+fPair.String(), "R0", 100)
	if err != nil {
		return o, err
	}
	if assetType == asset.MarginFunding {
		o.IsFundingRate = true
		o.Asks = make(orderbook.Items, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			o.Asks[x] = orderbook.Item{
				ID:     orderbookNew.Asks[x].OrderID,
				Price:  orderbookNew.Asks[x].Rate,
				Amount: orderbookNew.Asks[x].Amount,
				Period: int64(orderbookNew.Asks[x].Period),
			}
		}
		o.Bids = make(orderbook.Items, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			o.Bids[x] = orderbook.Item{
				ID:     orderbookNew.Bids[x].OrderID,
				Price:  orderbookNew.Bids[x].Rate,
				Amount: orderbookNew.Bids[x].Amount,
				Period: int64(orderbookNew.Bids[x].Period),
			}
		}
	} else {
		o.Asks = make(orderbook.Items, len(orderbookNew.Asks))
		for x := range orderbookNew.Asks {
			o.Asks[x] = orderbook.Item{
				ID:     orderbookNew.Asks[x].OrderID,
				Price:  orderbookNew.Asks[x].Price,
				Amount: orderbookNew.Asks[x].Amount,
			}
		}
		o.Bids = make(orderbook.Items, len(orderbookNew.Bids))
		for x := range orderbookNew.Bids {
			o.Bids[x] = orderbook.Item{
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

	var Accounts = []account.SubAccount{
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

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Bitfinex) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(b.Name, creds, assetType)
	if err != nil {
		return b.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
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
			Description:     history[i].Description,
			Timestamp:       time.UnixMilli(int64(history[i].Timestamp)),
			Currency:        history[i].Currency,
			Amount:          history[i].Amount,
			Fee:             history[i].Fee,
			TransferType:    history[i].Type,
			CryptoToAddress: history[i].Address,
			CryptoTxID:      history[i].TxID,
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
	var err error
	p, err = b.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var currString string
	currString, err = b.fixCasing(p, a)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	ts := timestampEnd
	limit := 10000
allTrades:
	for {
		var tradeData []Trade
		tradeData, err = b.GetTrades(ctx,
			currString, int64(limit), 0, ts.Unix()*1000, false)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			tradeTS := time.UnixMilli(tradeData[i].Timestamp)
			if tradeTS.Before(timestampStart) && !timestampStart.IsZero() {
				break allTrades
			}
			tID := strconv.FormatInt(tradeData[i].TID, 10)
			resp = append(resp, trade.Data{
				TID:          tID,
				Exchange:     b.Name,
				CurrencyPair: p,
				AssetType:    a,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Amount,
				Timestamp:    time.UnixMilli(tradeData[i].Timestamp),
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

	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (b *Bitfinex) SubmitOrder(ctx context.Context, o *order.Submit) (*order.SubmitResponse, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}

	fPair, err := b.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return nil, err
	}

	var orderID string
	status := order.New
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		symbolStr, err := b.fixCasing(fPair, o.AssetType) //nolint:govet // intentional shadow of err
		if err != nil {
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
		orderID, err = b.WsNewOrder(req)
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
		err = b.WsModifyOrder(&wsRequest)
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
		err = b.WsCancelOrder(orderIDInt)
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
		err = b.WsCancelAllOrders()
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
	var timestamp float64
	timestamp, err = strconv.ParseFloat(o.Timestamp, 64)
	if err != nil {
		log.Warnf(log.ExchangeSys,
			"%s Unable to convert timestamp '%s', leaving blank",
			b.Name, o.Timestamp)
	}

	var pair currency.Pair
	pair, err = currency.NewPairFromString(o.Symbol)
	if err != nil {
		return nil, err
	}

	orderDetail := &order.Detail{
		Amount:          o.OriginalAmount,
		Date:            time.Unix(int64(timestamp), 0),
		Exchange:        b.Name,
		OrderID:         strconv.FormatInt(o.ID, 10),
		Side:            side,
		Price:           o.Price,
		RemainingAmount: o.RemainingAmount,
		Pair:            pair,
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

	if c == currency.USDT {
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
	if tmpCurr == currency.USDT {
		// USDT is UST on Bitfinex
		tmpCurr = currency.NewCode("UST")
	}

	methods := acceptableMethods.lookup(tmpCurr)
	if len(methods) == 0 {
		return nil, errors.New("no transfer methods returned for currency")
	}
	method := methods[0]
	if len(methods) > 1 && withdrawRequest.Crypto.Chain != "" {
		if !common.StringDataCompareInsensitive(methods, withdrawRequest.Crypto.Chain) {
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
	if (len(p.Base.String()) > 3 && len(p.Quote.String()) > 0) ||
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
	candles, err := b.GetCandles(ctx, cf, fInterval, req.Start.UnixMilli(), req.End.UnixMilli(), uint32(req.RequestLimit), true)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].Timestamp,
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
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
		candles, err = b.GetCandles(ctx, cf, fInterval, req.RangeHolder.Ranges[x].Start.Time.UnixMilli(), req.RangeHolder.Ranges[x].End.Time.UnixMilli(), uint32(req.RequestLimit), true)
		if err != nil {
			return nil, err
		}

		for i := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[i].Timestamp,
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

func (b *Bitfinex) fixCasing(in currency.Pair, a asset.Item) (string, error) {
	if in.IsEmpty() || in.Base.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	var checkString [2]byte
	if a == asset.Spot || a == asset.Margin {
		checkString[0] = 't'
		checkString[1] = 'T'
	} else if a == asset.MarginFunding {
		checkString[0] = 'f'
		checkString[1] = 'F'
	}

	cFmt, err := b.FormatExchangeCurrency(in, a)
	if err != nil {
		return "", err
	}

	y := in.Base.String()
	if (y[0] != checkString[0] && y[0] != checkString[1]) ||
		(y[0] == checkString[1] && y[1] == checkString[1]) || in.Base == currency.TNB {
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

	if cryptocurrency == currency.USDT {
		// USDT is UST on Bitfinex
		cryptocurrency = currency.NewCode("UST")
	}

	availChains := acceptableMethods.lookup(cryptocurrency)
	if len(availChains) == 0 {
		return nil, fmt.Errorf("unable to find any available chains")
	}
	return availChains, nil
}

// GetServerTime returns the current exchange server time.
func (b *Bitfinex) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}
