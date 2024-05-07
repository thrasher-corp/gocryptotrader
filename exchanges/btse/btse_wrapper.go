package btse

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for BTSE
func (b *BTSE) SetDefaults() {
	b.Name = "BTSE"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
	}
	err := b.StoreAssetPairFormat(asset.Spot, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	fmt2 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
	}
	err = b.StoreAssetPairFormat(asset.Futures, fmt2)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				TickerBatching:      true,
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
				FundingRateFetching: true,
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
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.OneHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.Futures: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportsRestBatch:  true,
					SupportedViaTicker: true,
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
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 300,
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
		exchange.RestSpot:      btseAPIURL,
		exchange.RestFutures:   btseAPIURL,
		exchange.WebsocketSpot: btseWebsocket,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.Websocket = stream.NewWebsocket()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *BTSE) Setup(exch *config.Exchange) error {
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

	wsRunningURL, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            btseWebsocket,
		RunningURL:            wsRunningURL,
		Connector:             b.WsConnect,
		Subscriber:            b.Subscribe,
		Unsubscriber:          b.Unsubscribe,
		GenerateSubscriptions: b.GenerateDefaultSubscriptions,
		Features:              &b.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	err = b.seedOrderSizeLimits(context.TODO())
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *BTSE) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	m, err := b.GetMarketSummary(ctx, "", a == asset.Spot)
	if err != nil {
		return nil, err
	}
	pairs := make(currency.Pairs, 0, len(m))
	mPairs := m.MillionPairs()
	for _, l := range m {
		if !l.Active || !l.HasLiquidity() ||
			(a == asset.Spot && !l.IsMarketOpenToSpot) { // Skip OTC assets only tradable on web UI
			continue
		}
		if mPairs[l.Symbol] {
			// BTSE lists M_ symbols for very small pairs, in millions. For those listings, we want to take the M_ listing in preference
			// to the native listing, since they're often going to appear as locked markets due to size (bid == ask, e.g. 0.0000000003)
			continue
		}
		baseCurr := l.Base
		var quoteCurr string
		if a == asset.Futures {
			s := strings.Split(l.Symbol, l.Base) // e.g. RUNEPFC for RUNE-USD futures pair
			if len(s) <= 1 {
				continue
			}
			quoteCurr = s[1]
		} else {
			s := strings.Split(l.Symbol, currency.DashDelimiter)
			if len(s) != 2 {
				continue
			}
			baseCurr = s[0]
			quoteCurr = s[1]
		}
		pair, err := currency.NewPairFromStrings(baseCurr, quoteCurr)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *BTSE) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	a := b.GetAssetTypes(false)
	for i := range a {
		pairs, err := b.FetchTradablePairs(ctx, a[i])
		if err != nil {
			return err
		}
		err = b.UpdatePairs(pairs, a[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return b.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *BTSE) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !b.SupportsAsset(a) {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	tickers, err := b.GetMarketSummary(ctx, "", a == asset.Spot)
	if err != nil {
		return err
	}
	var errs error
	for x := range tickers {
		pair, err := currency.NewPairFromString(tickers[x].Symbol)
		if err == nil {
			err = ticker.ProcessTicker(&ticker.Price{
				Pair:         pair,
				Ask:          tickers[x].LowestAsk,
				Bid:          tickers[x].HighestBid,
				Low:          tickers[x].Low24Hr,
				Last:         tickers[x].Last,
				Volume:       tickers[x].Volume,
				High:         tickers[x].High24Hr,
				OpenInterest: tickers[x].OpenInterest,
				ExchangeName: b.Name,
				AssetType:    a})
		}
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}

	return errs
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *BTSE) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !b.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	ticks, err := b.GetMarketSummary(ctx, p.String(), a == asset.Spot)
	if err != nil {
		return nil, err
	}
	if len(ticks) != 1 {
		return nil, errors.New("market_summary should return 1 tick for a single ticker")
	}
	err = ticker.ProcessTicker(&ticker.Price{
		Pair:         p,
		Ask:          ticks[0].LowestAsk,
		Bid:          ticks[0].HighestBid,
		Low:          ticks[0].Low24Hr,
		Last:         ticks[0].Last,
		Volume:       ticks[0].Volume,
		High:         ticks[0].High24Hr,
		ExchangeName: b.Name,
		AssetType:    a})
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(b.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (b *BTSE) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *BTSE) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(b.Name, p, assetType)
	if err != nil {
		return b.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *BTSE) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}
	a, err := b.FetchOrderBook(ctx, fPair.String(), 0, 0, 0, assetType == asset.Spot)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Items, 0, len(a.BuyQuote))
	for x := range a.BuyQuote {
		if b.orderbookFilter(a.BuyQuote[x].Price, a.BuyQuote[x].Size) {
			continue
		}
		book.Bids = append(book.Bids, orderbook.Item{
			Price:  a.BuyQuote[x].Price,
			Amount: a.BuyQuote[x].Size,
		})
	}
	book.Asks = make(orderbook.Items, 0, len(a.SellQuote))
	for x := range a.SellQuote {
		if b.orderbookFilter(a.SellQuote[x].Price, a.SellQuote[x].Size) {
			continue
		}
		book.Asks = append(book.Asks, orderbook.Item{
			Price:  a.SellQuote[x].Price,
			Amount: a.SellQuote[x].Size,
		})
	}
	book.Asks.SortAsks()
	book.Pair = p
	book.Exchange = b.Name
	book.Asset = assetType
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// BTSE exchange
func (b *BTSE) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var a account.Holdings
	balance, err := b.GetWalletInformation(ctx)
	if err != nil {
		return a, err
	}

	currencies := make([]account.Balance, len(balance))
	for b := range balance {
		currencies[b] = account.Balance{
			Currency: currency.NewCode(balance[b].Currency),
			Total:    balance[b].Total,
			Hold:     balance[b].Total - balance[b].Available,
			Free:     balance[b].Available,
		}
	}
	a.Exchange = b.Name
	a.Accounts = []account.SubAccount{
		{
			AssetType:  assetType,
			Currencies: currencies,
		},
	}

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&a, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return a, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *BTSE) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
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
func (b *BTSE) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

func (b *BTSE) withinLimits(pair currency.Pair, amount float64) error {
	val, found := OrderSizeLimits(pair.String())
	if !found {
		return fmt.Errorf("%w for pair %v", order.ErrExchangeLimitNotLoaded, pair)
	}
	if math.Mod(amount, val.MinSizeIncrement) < 0 {
		return fmt.Errorf("%w %v %v %v", order.ErrAmountBelowMin, pair, amount, val.MinSizeIncrement)
	}
	if amount < val.MinOrderSize {
		return fmt.Errorf("%w %v %v %v", order.ErrAmountBelowMin, pair, amount, val.MinOrderSize)
	}
	if amount > val.MaxOrderSize {
		return fmt.Errorf("%w %v %v %v", order.ErrAmountExceedsMax, pair, amount, val.MinSizeIncrement)
	}
	return nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *BTSE) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *BTSE) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	const limit = 500
	var tradeData []Trade
	tradeData, err = b.GetTrades(ctx,
		p.String(),
		time.Time{}, time.Time{},
		0, 0, limit,
		false,
		assetType == asset.Spot)
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		tradeTimestamp := time.UnixMilli(tradeData[i].Time)
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     b.Name,
			TID:          strconv.FormatInt(tradeData[i].SerialID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeTimestamp,
		}
	}
	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *BTSE) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *BTSE) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	fPair, err := b.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	err = b.withinLimits(fPair, s.Amount)
	if err != nil {
		return nil, err
	}

	r, err := b.CreateOrder(ctx,
		s.ClientID, 0.0,
		false,
		s.Price,
		s.Side.String(),
		s.Amount, 0, 0,
		fPair.String(),
		goodTillCancel,
		0.0,
		s.TriggerPrice,
		"",
		s.Type.String())
	if err != nil {
		return nil, err
	}

	var orderID string
	if len(r) > 0 {
		orderID = r[0].OrderID
	}
	return s.DeriveSubmitResponse(orderID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *BTSE) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *BTSE) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	fPair, err := b.FormatExchangeCurrency(o.Pair, o.AssetType)
	if err != nil {
		return err
	}

	_, err = b.CancelExistingOrder(ctx, o.OrderID, fPair.String(), o.ClientOrderID)
	if err != nil {
		return err
	}

	return nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *BTSE) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
// If product ID is sent, all orders of that specified market will be cancelled
// If not specified, all orders of all markets will be cancelled
func (b *BTSE) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var resp order.CancelAllResponse

	fPair, err := b.FormatExchangeCurrency(orderCancellation.Pair,
		orderCancellation.AssetType)
	if err != nil {
		return resp, err
	}

	allOrders, err := b.CancelExistingOrder(ctx, "", fPair.String(), "")
	if err != nil {
		return resp, err
	}

	resp.Status = make(map[string]string)
	for x := range allOrders {
		if allOrders[x].Status == orderCancelled {
			resp.Status[allOrders[x].OrderID] = order.Cancelled.String()
		}
	}
	return resp, nil
}

func orderIntToType(i int) order.Type {
	if i == 77 {
		return order.Market
	} else if i == 76 {
		return order.Limit
	}
	return order.UnknownType
}

// GetOrderInfo returns order information based on order ID
func (b *BTSE) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	o, err := b.GetOrders(ctx, "", orderID, "")
	if err != nil {
		return nil, err
	}

	var od order.Detail
	if len(o) == 0 {
		return nil, errors.New("no orders found")
	}

	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	for i := range o {
		if o[i].OrderID != orderID {
			continue
		}

		var side = order.Buy
		if strings.EqualFold(o[i].Side, order.Ask.String()) {
			side = order.Sell
		}

		od.Pair, err = currency.NewPairDelimiter(o[i].Symbol,
			format.Delimiter)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetOrderInfo unable to parse currency pair: %s\n",
				b.Name,
				err)
		}
		od.Exchange = b.Name
		od.Amount = o[i].Size
		od.OrderID = o[i].OrderID
		od.Date = time.Unix(o[i].Timestamp, 0)
		od.Side = side

		od.Type = orderIntToType(o[i].OrderType)

		od.Price = o[i].Price
		if od.Status, err = order.StringToOrderStatus(o[i].OrderState); err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}

		th, err := b.TradeHistory(ctx,
			"",
			time.Time{}, time.Time{},
			0, 0, 0,
			false,
			"", orderID)
		if err != nil {
			return nil, fmt.Errorf("unable to get order fills for orderID %s", orderID)
		}

		for i := range th {
			createdAt, err := parseOrderTime(th[i].TradeID)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s GetOrderInfo unable to parse time: %s\n", b.Name, err)
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(th[i].Side)
			if err != nil {
				return nil, err
			}
			od.Trades = append(od.Trades, order.TradeHistory{
				Timestamp: createdAt,
				TID:       th[i].TradeID,
				Price:     th[i].Price,
				Amount:    th[i].Size,
				Exchange:  b.Name,
				Side:      orderSide,
				Fee:       th[i].FeeAmount,
			})
		}
	}
	return &od, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *BTSE) GetDepositAddress(ctx context.Context, c currency.Code, _, _ string) (*deposit.Address, error) {
	address, err := b.GetWalletAddress(ctx, c.String())
	if err != nil {
		return nil, err
	}

	extractor := func(addr string) (string, string) {
		if strings.Contains(addr, ":") {
			split := strings.Split(addr, ":")
			return split[0], split[1]
		}
		return addr, ""
	}

	if len(address) == 0 {
		addressCreate, err := b.CreateWalletAddress(ctx, c.String())
		if err != nil {
			return nil, err
		}
		if len(addressCreate) != 0 {
			addr, tag := extractor(addressCreate[0].Address)
			return &deposit.Address{
				Address: addr,
				Tag:     tag,
			}, nil
		}
		return nil, errors.New("address not found")
	}
	addr, tag := extractor(address[0].Address)
	return &deposit.Address{
		Address: addr,
		Tag:     tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	amountToString := strconv.FormatFloat(withdrawRequest.Amount, 'f', 8, 64)
	resp, err := b.WalletWithdrawal(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		amountToString)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name: b.Name,
		ID:   resp.WithdrawID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *BTSE) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (b *BTSE) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	if len(req.Pairs) == 0 {
		return nil, errors.New("no pair provided")
	}

	var orders []order.Detail
	for x := range req.Pairs {
		formattedPair, err := b.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		resp, err := b.GetOrders(ctx, formattedPair.String(), "", "")
		if err != nil {
			return nil, err
		}

		format, err := b.GetPairFormat(asset.Spot, false)
		if err != nil {
			return nil, err
		}

		for i := range resp {
			var side = order.Buy
			if strings.EqualFold(resp[i].Side, order.Ask.String()) {
				side = order.Sell
			}

			status, err := order.StringToOrderStatus(resp[i].OrderState)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
			}

			p, err := currency.NewPairDelimiter(resp[i].Symbol,
				format.Delimiter)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s GetActiveOrders unable to parse currency pair: %s\n",
					b.Name,
					err)
			}

			openOrder := order.Detail{
				Pair:            p,
				Exchange:        b.Name,
				Amount:          resp[i].Size,
				ExecutedAmount:  resp[i].FilledSize,
				RemainingAmount: resp[i].Size - resp[i].FilledSize,
				OrderID:         resp[i].OrderID,
				Date:            time.Unix(resp[i].Timestamp, 0),
				Side:            side,
				Price:           resp[i].Price,
				Status:          status,
			}

			if resp[i].OrderType == 77 {
				openOrder.Type = order.Market
			} else if resp[i].OrderType == 76 {
				openOrder.Type = order.Limit
			}

			fills, err := b.TradeHistory(ctx,
				"",
				time.Time{}, time.Time{},
				0, 0, 0,
				false,
				"", resp[i].OrderID)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s: Unable to get order fills for orderID %s",
					b.Name,
					resp[i].OrderID)
				continue
			}

			for i := range fills {
				createdAt, err := parseOrderTime(fills[i].Timestamp)
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%s GetActiveOrders unable to parse time: %s\n",
						b.Name,
						err)
				}
				var orderSide order.Side
				orderSide, err = order.StringToOrderSide(fills[i].Side)
				if err != nil {
					return nil, err
				}
				openOrder.Trades = append(openOrder.Trades, order.TradeHistory{
					Timestamp: createdAt,
					TID:       fills[i].TradeID,
					Price:     fills[i].Price,
					Amount:    fills[i].Size,
					Exchange:  b.Name,
					Side:      orderSide,
					Fee:       fills[i].FeeAmount,
				})
			}
			orders = append(orders, openOrder)
		}
	}
	return req.Filter(b.Name, orders), nil
}

func matchType(input int, required order.Type) bool {
	if (required == order.AnyType) || (input == 76 && required == order.Limit) || input == 77 && required == order.Market {
		return true
	}
	return false
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *BTSE) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}

	var resp []order.Detail
	if len(getOrdersRequest.Pairs) == 0 {
		var err error
		getOrdersRequest.Pairs, err = b.GetEnabledPairs(asset.Spot)
		if err != nil {
			return nil, err
		}
	}
	orderDeref := *getOrdersRequest
	for x := range orderDeref.Pairs {
		fPair, err := b.FormatExchangeCurrency(orderDeref.Pairs[x], asset.Spot)
		if err != nil {
			return nil, err
		}
		currentOrder, err := b.GetOrders(ctx, fPair.String(), "", "")
		if err != nil {
			return nil, err
		}
		for y := range currentOrder {
			if !matchType(currentOrder[y].OrderType, orderDeref.Type) {
				continue
			}
			orderStatus, err := order.StringToOrderStatus(currentOrder[y].OrderState)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(currentOrder[y].Side)
			if err != nil {
				return nil, err
			}
			orderTime := time.UnixMilli(currentOrder[y].Timestamp)
			tempOrder := order.Detail{
				OrderID:              currentOrder[y].OrderID,
				ClientID:             currentOrder[y].ClOrderID,
				Exchange:             b.Name,
				Price:                currentOrder[y].Price,
				AverageExecutedPrice: currentOrder[y].AverageFillPrice,
				Amount:               currentOrder[y].Size,
				ExecutedAmount:       currentOrder[y].FilledSize,
				RemainingAmount:      currentOrder[y].Size - currentOrder[y].FilledSize,
				Date:                 orderTime,
				Side:                 orderSide,
				Status:               orderStatus,
				Pair:                 orderDeref.Pairs[x],
			}
			tempOrder.InferCostsAndTimes()
			resp = append(resp, tempOrder)
		}
	}
	return getOrdersRequest.Filter(b.Name, resp), nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *BTSE) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !b.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(ctx, feeBuilder)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (b *BTSE) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// FormatExchangeKlineInterval formats kline interval to exchange requested type
func (b *BTSE) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Minutes(), 'f', 0, 64)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *BTSE) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.Futures:
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	req, err := b.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	intervalInt, err := strconv.Atoi(b.FormatExchangeKlineInterval(req.ExchangeInterval))
	if err != nil {
		return nil, err
	}

	candles, err := b.GetOHLCV(ctx,
		req.RequestFormatted.String(),
		req.Start,
		req.End.Add(-req.ExchangeInterval.Duration()), // End time is inclusive, so we need to subtract the interval.
		intervalInt,
		a)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   time.Unix(int64(candles[x][0]), 0),
			Open:   candles[x][1],
			High:   candles[x][2],
			Low:    candles[x][3],
			Close:  candles[x][4],
			Volume: candles[x][5],
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *BTSE) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	switch a {
	case asset.Spot, asset.Futures:
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	req, err := b.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	intervalInt, err := strconv.Atoi(b.FormatExchangeKlineInterval(req.ExchangeInterval))
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, req.Size())
	for i := range req.RangeHolder.Ranges {
		var candles OHLCV
		candles, err = b.GetOHLCV(ctx,
			req.RequestFormatted.String(),
			req.RangeHolder.Ranges[i].Start.Time,
			req.RangeHolder.Ranges[i].End.Time,
			intervalInt,
			a)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			timeSeries[x] = kline.Candle{
				Time:   time.Unix(int64(candles[x][0]), 0),
				Open:   candles[x][1],
				High:   candles[x][2],
				Low:    candles[x][3],
				Close:  candles[x][4],
				Volume: candles[x][5],
			}
		}
	}

	return req.ProcessResponse(timeSeries)
}

func (b *BTSE) seedOrderSizeLimits(ctx context.Context) error {
	pairs, err := b.GetMarketSummary(ctx, "", true)
	if err != nil {
		return err
	}
	for x := range pairs {
		tempValues := OrderSizeLimit{
			MinOrderSize:     pairs[x].MinOrderSize,
			MaxOrderSize:     pairs[x].MaxOrderSize,
			MinSizeIncrement: pairs[x].MinSizeIncrement,
		}
		orderSizeLimitMap.Store(pairs[x].Symbol, tempValues)
	}

	pairs, err = b.GetMarketSummary(ctx, "", false)
	if err != nil {
		return err
	}
	for x := range pairs {
		tempValues := OrderSizeLimit{
			MinOrderSize:     pairs[x].MinOrderSize,
			MaxOrderSize:     pairs[x].MaxOrderSize,
			MinSizeIncrement: pairs[x].MinSizeIncrement,
		}
		orderSizeLimitMap.Store(pairs[x].Symbol, tempValues)
	}
	return nil
}

// OrderSizeLimits looks up currency pair in orderSizeLimitMap and returns OrderSizeLimit
func OrderSizeLimits(pair string) (limits OrderSizeLimit, found bool) {
	resp, ok := orderSizeLimitMap.Load(pair)
	if !ok {
		return
	}
	val, ok := resp.(OrderSizeLimit)
	return val, ok
}

// GetServerTime returns the current exchange server time.
func (b *BTSE) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	st, err := b.GetCurrentServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return st.ISO, nil
}

// GetFuturesContractDetails returns details about futures contracts
func (b *BTSE) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	marketSummary, err := b.GetMarketSummary(ctx, "", false)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, 0, len(marketSummary))
	for i := range marketSummary {
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(marketSummary[i].Base, marketSummary[i].Symbol[len(marketSummary[i].Base):])
		if err != nil {
			return nil, err
		}
		settlementCurrencies := make(currency.Currencies, len(marketSummary[i].AvailableSettlement))
		var s, e time.Time
		var ct futures.ContractType
		if marketSummary[i].OpenTime > 0 {
			s = time.UnixMilli(marketSummary[i].OpenTime)
		}
		if marketSummary[i].CloseTime > 0 {
			e = time.UnixMilli(marketSummary[i].CloseTime)
		}
		if marketSummary[i].TimeBasedContract {
			if e.Sub(s) > kline.OneMonth.Duration() {
				ct = futures.Quarterly
			} else {
				ct = futures.Monthly
			}
		} else {
			ct = futures.Perpetual
		}
		var contractSettlementType futures.ContractSettlementType
		for j := range marketSummary[i].AvailableSettlement {
			settlementCurrencies[j] = currency.NewCode(marketSummary[i].AvailableSettlement[j])
			if contractSettlementType == futures.LinearOrInverse {
				continue
			}
			containsUSD := strings.Contains(marketSummary[i].AvailableSettlement[j], "USD")
			if !containsUSD {
				contractSettlementType = futures.LinearOrInverse
				continue
			}
			if containsUSD {
				contractSettlementType = futures.Linear
			}
		}

		c := futures.Contract{
			Exchange:             b.Name,
			Name:                 cp,
			Underlying:           currency.NewPair(currency.NewCode(marketSummary[i].Base), currency.NewCode(marketSummary[i].Quote)),
			Asset:                item,
			SettlementCurrencies: settlementCurrencies,
			StartDate:            s,
			EndDate:              e,
			SettlementType:       contractSettlementType,
			IsActive:             marketSummary[i].Active,
			Type:                 ct,
		}
		if marketSummary[i].FundingRate > 0 {
			c.LatestRate = fundingrate.Rate{
				Rate: decimal.NewFromFloat(marketSummary[i].FundingRate),
				Time: time.Now().Truncate(time.Hour),
			}
		}

		resp = append(resp, c)
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (b *BTSE) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}

	format, err := b.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := format.Format(r.Pair)
	rates, err := b.GetMarketSummary(ctx, fPair, false)
	if err != nil {
		return nil, err
	}

	resp := make([]fundingrate.LatestRateResponse, 0, len(rates))
	for i := range rates {
		var cp currency.Pair
		var isEnabled bool
		cp, isEnabled, err = b.MatchSymbolCheckEnabled(rates[i].Symbol, r.Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		var isPerp bool
		isPerp, err = b.IsPerpetualFutureCurrency(r.Asset, cp)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			continue
		}
		tt := time.Now().Truncate(time.Hour)
		resp = append(resp, fundingrate.LatestRateResponse{
			Exchange: b.Name,
			Asset:    r.Asset,
			Pair:     cp,
			LatestRate: fundingrate.Rate{
				Time: time.Now().Truncate(time.Hour),
				Rate: decimal.NewFromFloat(rates[i].FundingRate),
			},
			TimeOfNextRate: tt.Add(time.Hour),
			TimeChecked:    time.Now(),
		})
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (b *BTSE) IsPerpetualFutureCurrency(a asset.Item, p currency.Pair) (bool, error) {
	return a == asset.Futures && p.Quote.Equal(currency.PFC), nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (b *BTSE) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (b *BTSE) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	tickers, err := b.GetMarketSummary(ctx, "", false)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(tickers))
	for i := range tickers {
		var symbol currency.Pair
		var enabled bool
		symbol, enabled, err = b.MatchSymbolCheckEnabled(tickers[i].Symbol, asset.Futures, false)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !enabled {
			continue
		}
		var appendData bool
		for j := range k {
			if k[j].Pair().Equal(symbol) {
				appendData = true
				break
			}
		}
		if len(k) > 0 && !appendData {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key: key.ExchangePairAsset{
				Exchange: b.Name,
				Base:     symbol.Base.Item,
				Quote:    symbol.Quote.Item,
				Asset:    asset.Futures,
			},
			OpenInterest: tickers[i].OpenInterest,
		})
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *BTSE) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := b.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	switch a {
	case asset.Spot:
		return tradeBaseURL + tradeSpot + cp.Upper().String(), nil
	case asset.Futures:
		return tradeBaseURL + tradeFutures + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
