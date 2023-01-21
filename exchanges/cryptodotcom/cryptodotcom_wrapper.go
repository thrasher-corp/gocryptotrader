package cryptodotcom

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (cr *Cryptodotcom) GetDefaultConfig() (*config.Exchange, error) {
	cr.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = cr.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = cr.BaseCurrencies

	cr.SetupDefaults(exchCfg)

	if cr.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := cr.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Cryptodotcom
func (cr *Cryptodotcom) SetDefaults() {
	cr.Name = "Cryptodotcom"
	cr.Enabled = true
	cr.Verbose = true
	cr.API.CredentialsValidator.RequiresKey = true
	cr.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}
	err := cr.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	// Fill out the capabilities/features that the exchange supports
	cr.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.OneMin,
					kline.FiveMin,
					kline.FifteenMin,
					kline.ThirtyMin,
					kline.OneHour,
					kline.FourHour,
					kline.SixHour,
					kline.TwelveHour,
					kline.OneDay,
					kline.SevenDay,
					kline.TwoWeek,
					kline.OneMonth,
				),
				ResultLimit: 200,
			},
		},
	}
	cr.Requester, err = request.New(cr.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	cr.API.Endpoints = cr.NewEndpoints()
	cr.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      cryptodotcomAPIURL,
		exchange.WebsocketSpot: cryptodotcomWebsocketURL,
	})
	cr.Websocket = stream.New()
	cr.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	cr.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	cr.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (cr *Cryptodotcom) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		cr.SetEnabled(false)
		return nil
	}
	err = cr.SetupDefaults(exch)
	if err != nil {
		return err
	}
	wsRunningEndpoint, err := cr.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = cr.Websocket.Setup(
		&stream.WebsocketSetup{
			ExchangeConfig:        exch,
			DefaultURL:            cryptodotcomWebsocketURL,
			RunningURL:            wsRunningEndpoint,
			Connector:             cr.WsConnect,
			Subscriber:            cr.Subscribe,
			Unsubscriber:          cr.Unsubscribe,
			GenerateSubscriptions: cr.GenerateDefaultSubscriptions,
			Features:              &cr.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}
	return cr.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  cr.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Cryptodotcom go routine
func (cr *Cryptodotcom) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		cr.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Cryptodotcom wrapper
func (cr *Cryptodotcom) Run() {
	if cr.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			cr.Name,
			common.IsEnabled(cr.Websocket.IsEnabled()))
		cr.PrintEnabledPairs()
	}

	if !cr.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := cr.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			cr.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (cr *Cryptodotcom) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !cr.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, cr.Name)
	}
	instruments, err := cr.GetInstruments(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make(currency.Pairs, len(instruments))
	for x := range instruments {
		cp, err := currency.NewPairFromString(instruments[x].InstrumentName)
		if err != nil {
			return nil, err
		}
		pairs[x] = cp
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (cr *Cryptodotcom) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := cr.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	return cr.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (cr *Cryptodotcom) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	tick, err := cr.GetTicker(ctx, p.String())
	if err != nil {
		return tickerPrice, err
	}
	if len(tick.Data) != 1 {
		return tickerPrice, errInvalidResponseFromServer
	}
	tickerPrice = &ticker.Price{
		High:         tick.Data[0].HighestTradePrice,
		Low:          tick.Data[0].LowestTradePrice,
		Bid:          tick.Data[0].BestBidPrice,
		Ask:          tick.Data[0].BestAskPrice,
		Open:         tick.Data[0].OpenInterest,
		Last:         tick.Data[0].LatestTradePrice,
		Volume:       tick.Data[0].TradedVolume,
		LastUpdated:  tick.Data[0].TradeTimestamp.Time(),
		AssetType:    assetType,
		ExchangeName: cr.Name,
		Pair:         p,
	}
	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(cr.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (cr *Cryptodotcom) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	tick, err := cr.GetTicker(ctx, "")
	if err != nil {
		return err
	}
	for y := range tick.Data {
		cp, err := currency.NewPairFromString(tick.Data[y].InstrumentName)
		if err != nil {
			return err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick.Data[y].LatestTradePrice,
			High:         tick.Data[y].HighestTradePrice,
			Low:          tick.Data[y].LowestTradePrice,
			Bid:          tick.Data[y].BestBidPrice,
			Ask:          tick.Data[y].BestAskPrice,
			Volume:       tick.Data[y].TradedVolume,
			Open:         tick.Data[y].OpenInterest,
			Pair:         cp,
			ExchangeName: cr.Name,
			AssetType:    assetType,
			// QuoteVolume:  tick.Data[y].QuoteVolume,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (cr *Cryptodotcom) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(cr.Name, p, assetType)
	if err != nil {
		return cr.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (cr *Cryptodotcom) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(cr.Name, pair, assetType)
	if err != nil {
		return cr.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (cr *Cryptodotcom) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	orderbookNew, err := cr.GetOrderbook(context.Background(), pair.String(), 0)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        cr.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: cr.CanVerifyOrderbook,
	}
	book.Bids = make([]orderbook.Item, len(orderbookNew.Data[0].Bids))
	for x := range orderbookNew.Data[0].Bids {
		price, err := strconv.ParseFloat(orderbookNew.Data[0].Bids[x][0], 64)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseFloat(orderbookNew.Data[0].Bids[x][1], 64)
		if err != nil {
			return nil, err
		}
		book.Bids[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	book.Asks = make([]orderbook.Item, len(orderbookNew.Data[0].Asks))
	for x := range orderbookNew.Data[0].Asks {
		price, err := strconv.ParseFloat(orderbookNew.Data[0].Asks[x][0], 64)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseFloat(orderbookNew.Data[0].Asks[x][1], 64)
		if err != nil {
			return nil, err
		}
		book.Asks[x] = orderbook.Item{
			Amount: amount,
			Price:  price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(cr.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (cr *Cryptodotcom) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	info.Exchange = cr.Name
	if !cr.SupportsAsset(assetType) {
		return info, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	accs, err := cr.GetAccountSummary(ctx, currency.EMPTYCODE)
	if err != nil {
		return info, err
	}
	balances := make([]account.Balance, len(accs.Accounts))
	for i := range accs.Accounts {
		balances[i] = account.Balance{
			Currency: currency.NewCode(accs.Accounts[i].Currency),
			Total:    accs.Accounts[i].Balance,
			Hold:     accs.Accounts[i].Stake + accs.Accounts[i].Order,
			Free:     accs.Accounts[i].Available,
		}
	}
	acc := account.SubAccount{
		Currencies: balances,
		AssetType:  assetType,
	}
	info.Accounts = []account.SubAccount{acc}
	creds, err := cr.GetCredentials(ctx)
	if err != nil {
		return info, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (cr *Cryptodotcom) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := cr.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(cr.Name, creds, assetType)
	if err != nil {
		return cr.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (cr *Cryptodotcom) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	withdrawals, err := cr.GetWithdrawalHistory(ctx)
	if err != nil {
		return nil, err
	}
	deposits, err := cr.GetDepositHistory(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundHistory, 0, len(withdrawals.WithdrawalList)+len(deposits.DepositList))
	for x := range withdrawals.WithdrawalList {
		resp = append(resp, exchange.FundHistory{
			Status:          translateWithdrawalStatus(withdrawals.WithdrawalList[x].Status),
			Timestamp:       withdrawals.WithdrawalList[x].UpdateTime.Time(),
			Currency:        withdrawals.WithdrawalList[x].Currency,
			Amount:          withdrawals.WithdrawalList[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals.WithdrawalList[x].Address,
			TransferID:      withdrawals.WithdrawalList[x].TransactionID,
			Fee:             withdrawals.WithdrawalList[x].Fee,
		})
	}
	for x := range deposits.DepositList {
		resp = append(resp, exchange.FundHistory{
			ExchangeName:    cr.Name,
			Status:          translateDepositStatus(deposits.DepositList[x].Status),
			Timestamp:       deposits.DepositList[x].UpdateTime.Time(),
			Currency:        deposits.DepositList[x].Currency,
			Amount:          deposits.DepositList[x].Amount,
			TransferType:    "deposit",
			CryptoToAddress: deposits.DepositList[x].Address,
			CryptoTxID:      deposits.DepositList[x].ID,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (cr *Cryptodotcom) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := cr.GetWithdrawalHistory(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals.WithdrawalList))
	for x := range withdrawals.WithdrawalList {
		resp[x] = exchange.WithdrawalHistory{
			Status:          translateWithdrawalStatus(withdrawals.WithdrawalList[x].Status),
			Timestamp:       withdrawals.WithdrawalList[x].UpdateTime.Time(),
			Currency:        withdrawals.WithdrawalList[x].Currency,
			Amount:          withdrawals.WithdrawalList[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals.WithdrawalList[x].Address,
			TransferID:      withdrawals.WithdrawalList[x].TransactionID,
			Fee:             withdrawals.WithdrawalList[x].Fee,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (cr *Cryptodotcom) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	format, err := cr.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	trades, err := cr.GetTrades(ctx, format.Format(p))
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(trades.Data))
	for x := range trades.Data {
		side, err := order.StringToOrderSide(trades.Data[x].Side)
		if err != nil {
			return nil, err
		}
		resp[x] = trade.Data{
			TID:          trades.Data[x].TradeID,
			Exchange:     cr.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        trades.Data[x].TradePrice,
			Amount:       trades.Data[x].TradeQuantity,
			Timestamp:    trades.Data[x].DataTime.Time(),
		}
	}
	if cr.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(cr.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (cr *Cryptodotcom) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (cr *Cryptodotcom) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	// When an order has been submitted you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.SubmitResponse if you think they are applicable.
	// resp, err := s.DeriveSubmitResponse( /*newOrderID*/)
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.Date = exampleTime // e.g. If this is supplied by the exchanges API.
	// return resp, nil
	return nil, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (cr *Cryptodotcom) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	// When an order has been modified you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.ModifyResponse if you think they are applicable.
	// resp, err := action.DeriveModifyResponse()
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.OrderID = maybeANewOrderID // e.g. If this is supplied by the exchanges API.
	return nil, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (cr *Cryptodotcom) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (cr *Cryptodotcom) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (cr *Cryptodotcom) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (cr *Cryptodotcom) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (cr *Cryptodotcom) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (cr *Cryptodotcom) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (cr *Cryptodotcom) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (cr *Cryptodotcom) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (cr *Cryptodotcom) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (cr *Cryptodotcom) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (cr *Cryptodotcom) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (cr *Cryptodotcom) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := cr.UpdateAccountInfo(ctx, assetType)
	return cr.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (cr *Cryptodotcom) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, _, _ time.Time) (*kline.Item, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !a.IsValid() {
		return nil, asset.ErrNotSupported
	}
	formattedPair, err := cr.FormatSymbol(pair, a)
	if err != nil {
		return nil, err
	}
	candles, err := cr.GetCandlestickDetail(ctx, formattedPair, interval)
	if err != nil {
		return nil, err
	}
	candleElements := make([]kline.Candle, len(candles.Data))
	for x := range candles.Data {
		candleElements[x] = kline.Candle{
			Time:   candles.Data[x].EndTime.Time(),
			Open:   candles.Data[x].Open,
			High:   candles.Data[x].High,
			Low:    candles.Data[x].Low,
			Close:  candles.Data[x].Close,
			Volume: candles.Data[x].Volume,
		}
	}
	return &kline.Item{
		Exchange: cr.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
		Candles:  candleElements,
	}, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (cr *Cryptodotcom) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return cr.GetHistoricCandles(ctx, pair, a, interval, start, end)
}
