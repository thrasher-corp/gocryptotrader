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
func (b *Bitfinex) GetDefaultConfig() (*config.Exchange, error) {
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
		err = b.UpdateTradablePairs(context.TODO(), true)
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
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
	}

	fmt2 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: ":"},
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
					kline.OneWeek.Word():    true,
					kline.TwoWeek.Word():    true,
				},
				ResultLimit: 10000,
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
		ExchangeConfig:        exch,
		DefaultURL:            publicBitfinexWebsocketEndpoint,
		RunningURL:            wsEndpoint,
		Connector:             b.WsConnect,
		Subscriber:            b.Subscribe,
		Unsubscriber:          b.Unsubscribe,
		GenerateSubscriptions: b.GenerateDefaultSubscriptions,
		Features:              &b.Features.Supports.WebsocketCapabilities,
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
func (b *Bitfinex) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Bitfinex wrapper
func (b *Bitfinex) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()))
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitfinex) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	items, err := b.GetTickerBatch(ctx)
	if err != nil {
		return nil, err
	}

	var symbols []string
	switch a {
	case asset.Spot:
		for k := range items {
			if !strings.HasPrefix(k, "t") {
				continue
			}
			symbols = append(symbols, k[1:])
		}
	case asset.Margin:
		for k := range items {
			if !strings.Contains(k, ":") {
				continue
			}
			symbols = append(symbols, k[1:])
		}
	case asset.MarginFunding:
		for k := range items {
			if !strings.HasPrefix(k, "f") {
				continue
			}
			symbols = append(symbols, k[1:])
		}
	default:
		return nil, errors.New("asset type not supported by this endpoint")
	}

	return symbols, nil
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

		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}

		err = b.UpdatePairs(p, assets[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bitfinex) UpdateTickers(ctx context.Context, a asset.Item) error {
	enabledPairs, err := b.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	tickerNew, err := b.GetTickerBatch(ctx)
	if err != nil {
		return err
	}

	for k, v := range tickerNew {
		pair, err := currency.NewPairFromString(k[1:]) // Remove prefix
		if err != nil {
			return err
		}

		if !enabledPairs.Contains(pair, true) {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Last:         v.Last,
			High:         v.High,
			Low:          v.Low,
			Bid:          v.Bid,
			Ask:          v.Ask,
			Volume:       v.Volume,
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

	b.appendOptionalDelimiter(&fPair)
	tick, err := ticker.GetTicker(b.Name, fPair, asset.Spot)
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

	b.appendOptionalDelimiter(&fPair)
	ob, err := orderbook.Get(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateOrderbook(ctx, fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitfinex) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
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
		return o, fmt.Errorf("assetType not supported: %v", assetType)
	}
	b.appendOptionalDelimiter(&fPair)
	var prefix = "t"
	if assetType == asset.MarginFunding {
		prefix = "f"
	}
	var orderbookNew Orderbook
	orderbookNew, err = b.GetOrderbook(ctx, prefix+fPair.String(), "R0", 100)
	if err != nil {
		return nil, err
	}
	if assetType == asset.MarginFunding {
		o.IsFundingRate = true
		for x := range orderbookNew.Asks {
			o.Asks = append(o.Asks, orderbook.Item{
				ID:     orderbookNew.Asks[x].OrderID,
				Price:  orderbookNew.Asks[x].Rate,
				Amount: orderbookNew.Asks[x].Amount,
				Period: int64(orderbookNew.Asks[x].Period),
			})
		}
		for x := range orderbookNew.Bids {
			o.Bids = append(o.Bids, orderbook.Item{
				ID:     orderbookNew.Bids[x].OrderID,
				Price:  orderbookNew.Bids[x].Rate,
				Amount: orderbookNew.Bids[x].Amount,
				Period: int64(orderbookNew.Bids[x].Period),
			})
		}
	} else {
		for x := range orderbookNew.Asks {
			o.Asks = append(o.Asks, orderbook.Item{
				ID:     orderbookNew.Asks[x].OrderID,
				Price:  orderbookNew.Asks[x].Price,
				Amount: orderbookNew.Asks[x].Amount,
			})
		}
		for x := range orderbookNew.Bids {
			o.Bids = append(o.Bids, orderbook.Item{
				ID:     orderbookNew.Bids[x].OrderID,
				Price:  orderbookNew.Bids[x].Price,
				Amount: orderbookNew.Bids[x].Amount,
			})
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
		{ID: "deposit"},
		{ID: "exchange"},
		{ID: "trading"},
		{ID: "margin"},
		{ID: "funding "},
	}

	for x := range accountBalance {
		for i := range Accounts {
			if Accounts[i].ID == accountBalance[x].Type {
				Accounts[i].Currencies = append(Accounts[i].Currencies,
					account.Balance{
						CurrencyName: currency.NewCode(accountBalance[x].Currency),
						Total:        accountBalance[x].Amount,
						Hold:         accountBalance[x].Amount - accountBalance[x].Available,
						Free:         accountBalance[x].Available,
					})
			}
		}
	}

	response.Accounts = Accounts
	err = account.Process(&response)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Bitfinex) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name, assetType)
	if err != nil {
		return b.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitfinex) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitfinex) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Bitfinex) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return b.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bitfinex) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if assetType == asset.MarginFunding {
		return nil, fmt.Errorf("asset type '%v' not supported", assetType)
	}
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var currString string
	currString, err = b.fixCasing(p, assetType)
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
				AssetType:    assetType,
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
func (b *Bitfinex) SubmitOrder(ctx context.Context, o *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	err := o.Validate()
	if err != nil {
		return submitOrderResponse, err
	}

	fpair, err := b.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return submitOrderResponse, err
	}

	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		submitOrderResponse.OrderID, err = b.WsNewOrder(&WsNewOrderRequest{
			CustomID: b.Websocket.AuthConn.GenerateMessageID(false),
			Type:     o.Type.String(),
			Symbol:   fpair.String(),
			Amount:   o.Amount,
			Price:    o.Price,
		})
		if err != nil {
			return submitOrderResponse, err
		}
	} else {
		var response Order
		isBuying := o.Side == order.Buy
		b.appendOptionalDelimiter(&fpair)
		orderType := o.Type.Lower()
		if o.AssetType == asset.Spot {
			orderType = "exchange " + orderType
		}
		response, err = b.NewOrder(ctx,
			fpair.String(),
			orderType,
			o.Amount,
			o.Price,
			isBuying,
			false)
		if err != nil {
			return submitOrderResponse, err
		}
		if response.ID > 0 {
			submitOrderResponse.OrderID = strconv.FormatInt(response.ID, 10)
		}
		if response.RemainingAmount == 0 {
			submitOrderResponse.FullyMatched = true
		}

		submitOrderResponse.IsOrderPlaced = true
	}
	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitfinex) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	if err := action.Validate(); err != nil {
		return order.Modify{}, err
	}

	orderIDInt, err := strconv.ParseInt(action.ID, 10, 64)
	if err != nil {
		return order.Modify{ID: action.ID}, err
	}
	if b.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		request := WsUpdateOrderRequest{
			OrderID: orderIDInt,
			Price:   action.Price,
			Amount:  action.Amount,
		}
		if action.Side == order.Sell && action.Amount > 0 {
			request.Amount *= -1
		}
		err = b.WsModifyOrder(&request)
		return order.Modify{
			Exchange:  action.Exchange,
			AssetType: action.AssetType,
			Pair:      action.Pair,
			ID:        action.ID,

			Price:  action.Price,
			Amount: action.Amount,
		}, err
	}
	return order.Modify{}, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitfinex) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.ID, 10, 64)
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
func (b *Bitfinex) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
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

// GetOrderInfo returns order information based on order ID
func (b *Bitfinex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
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
		return nil, errors.New("unsupported currency")
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
func (b *Bitfinex) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var orders []order.Detail
	resp, err := b.GetOpenOrders(ctx)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := order.Side(strings.ToUpper(resp[i].Side))
		timestamp, err := strconv.ParseFloat(resp[i].Timestamp, 64)
		if err != nil {
			log.Warnf(log.ExchangeSys,
				"Unable to convert timestamp '%s', leaving blank",
				resp[i].Timestamp)
		}

		pair, err := currency.NewPairFromString(resp[i].Symbol)
		if err != nil {
			return nil, err
		}

		orderDetail := order.Detail{
			Amount:          resp[i].OriginalAmount,
			Date:            time.Unix(int64(timestamp), 0),
			Exchange:        b.Name,
			ID:              strconv.FormatInt(resp[i].ID, 10),
			Side:            orderSide,
			Price:           resp[i].Price,
			RemainingAmount: resp[i].RemainingAmount,
			Pair:            pair,
			ExecutedAmount:  resp[i].ExecutedAmount,
		}

		switch {
		case resp[i].IsLive:
			orderDetail.Status = order.Active
		case resp[i].IsCancelled:
			orderDetail.Status = order.Cancelled
		case resp[i].IsHidden:
			orderDetail.Status = order.Hidden
		default:
			orderDetail.Status = order.UnknownStatus
		}

		// API docs discrepancy. Example contains prefixed "exchange "
		// Return type suggests “market” / “limit” / “stop” / “trailing-stop”
		orderType := strings.Replace(resp[i].Type, "exchange ", "", 1)
		if orderType == "trailing-stop" {
			orderDetail.Type = order.TrailingStop
		} else {
			orderDetail.Type = order.Type(strings.ToUpper(orderType))
		}

		orders = append(orders, orderDetail)
	}

	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitfinex) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var orders []order.Detail
	resp, err := b.GetInactiveOrders(ctx)
	if err != nil {
		return nil, err
	}

	for i := range resp {
		orderSide := order.Side(strings.ToUpper(resp[i].Side))
		timestamp, err := strconv.ParseInt(resp[i].Timestamp, 10, 64)
		if err != nil {
			log.Warnf(log.ExchangeSys, "Unable to convert timestamp '%v', leaving blank", resp[i].Timestamp)
		}
		orderDate := time.Unix(timestamp, 0)

		pair, err := currency.NewPairFromString(resp[i].Symbol)
		if err != nil {
			return nil, err
		}

		orderDetail := order.Detail{
			Amount:               resp[i].OriginalAmount,
			Date:                 orderDate,
			Exchange:             b.Name,
			ID:                   strconv.FormatInt(resp[i].ID, 10),
			Side:                 orderSide,
			Price:                resp[i].Price,
			AverageExecutedPrice: resp[i].AverageExecutionPrice,
			RemainingAmount:      resp[i].RemainingAmount,
			ExecutedAmount:       resp[i].ExecutedAmount,
			Pair:                 pair,
		}
		orderDetail.InferCostsAndTimes()

		switch {
		case resp[i].IsLive:
			orderDetail.Status = order.Active
		case resp[i].IsCancelled:
			orderDetail.Status = order.Cancelled
		case resp[i].IsHidden:
			orderDetail.Status = order.Hidden
		default:
			orderDetail.Status = order.UnknownStatus
		}

		// API docs discrepency. Example contains prefixed "exchange "
		// Return type suggests “market” / “limit” / “stop” / “trailing-stop”
		orderType := strings.Replace(resp[i].Type, "exchange ", "", 1)
		if orderType == "trailing-stop" {
			orderDetail.Type = order.TrailingStop
		} else {
			orderDetail.Type = order.Type(strings.ToUpper(orderType))
		}

		orders = append(orders, orderDetail)
	}

	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
	for i := range req.Pairs {
		b.appendOptionalDelimiter(&req.Pairs[i])
	}
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Bitfinex) AuthenticateWebsocket(ctx context.Context) error {
	return b.WsSendAuth(ctx)
}

// appendOptionalDelimiter ensures that a delimiter is present for long character currencies
func (b *Bitfinex) appendOptionalDelimiter(p *currency.Pair) {
	if len(p.Quote.String()) > 3 ||
		len(p.Base.String()) > 3 {
		p.Delimiter = ":"
	}
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Bitfinex) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (b *Bitfinex) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneDay:
		return "1D"
	case kline.OneWeek:
		return "7D"
	case kline.OneWeek * 2:
		return "14D"
	default:
		return in.Short()
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitfinex) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := b.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	if kline.TotalCandlesPerInterval(start, end, interval) > float64(b.Features.Enabled.Kline.ResultLimit) {
		return kline.Item{}, errors.New(kline.ErrRequestExceedsExchangeLimits)
	}

	cf, err := b.fixCasing(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	candles, err := b.GetCandles(ctx,
		cf, b.FormatExchangeKlineInterval(interval),
		start.Unix()*1000, end.Unix()*1000,
		b.Features.Enabled.Kline.ResultLimit, true)
	if err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	for x := range candles {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   candles[x].Timestamp,
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
func (b *Bitfinex) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := b.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	dates, err := kline.CalculateCandleDateRanges(start, end, interval, b.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	cf, err := b.fixCasing(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range dates.Ranges {
		var candles []Candle
		candles, err = b.GetCandles(ctx,
			cf, b.FormatExchangeKlineInterval(interval),
			dates.Ranges[x].Start.Ticks*1000, dates.Ranges[x].End.Ticks*1000,
			b.Features.Enabled.Kline.ResultLimit, true)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range candles {
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   candles[i].Timestamp,
				Open:   candles[i].Open,
				High:   candles[i].High,
				Low:    candles[i].Low,
				Close:  candles[i].Close,
				Volume: candles[i].Volume,
			})
		}
	}
	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", b.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

func (b *Bitfinex) fixCasing(in currency.Pair, a asset.Item) (string, error) {
	var checkString [2]byte
	if a == asset.Spot || a == asset.Margin {
		checkString[0] = 't'
		checkString[1] = 'T'
	} else if a == asset.MarginFunding {
		checkString[0] = 'f'
		checkString[1] = 'F'
	}

	fmt, err := b.FormatExchangeCurrency(in, a)
	if err != nil {
		return "", err
	}

	y := in.Base.String()
	if (y[0] != checkString[0] && y[0] != checkString[1]) ||
		(y[0] == checkString[1] && y[1] == checkString[1]) || in.Base == currency.TNB {
		if fmt.Quote.IsEmpty() {
			return string(checkString[0]) + fmt.Base.Upper().String(), nil
		}
		return string(checkString[0]) + fmt.Upper().String(), nil
	}

	runes := []rune(fmt.Upper().String())
	if fmt.Quote.IsEmpty() {
		runes = []rune(fmt.Base.Upper().String())
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
