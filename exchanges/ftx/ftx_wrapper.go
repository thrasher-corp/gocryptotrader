package ftx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
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
func (f *FTX) GetDefaultConfig() (*config.Exchange, error) {
	f.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = f.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = f.BaseCurrencies

	err := f.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if f.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = f.UpdateTradablePairs(context.TODO(), true)
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

	err := f.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = f.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	f.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:        true,
				TickerBatching:        true,
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
				TradeFee:              true,
				FiatDepositFee:        true,
				FiatWithdrawalFee:     true,
				CryptoWithdrawalFee:   true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				OrderbookFetching: true,
				TradeFetching:     true,
				Subscribe:         true,
				Unsubscribe:       true,
				GetOrders:         true,
				GetOrder:          true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.FifteenSecond.Word(): true,
					kline.OneMin.Word():        true,
					kline.FiveMin.Word():       true,
					kline.FifteenMin.Word():    true,
					kline.OneHour.Word():       true,
					kline.FourHour.Word():      true,
					kline.OneDay.Word():        true,
				},
				ResultLimit: 5000,
			},
		},
	}

	f.Requester = request.New(f.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(ratePeriod, rateLimit)))
	f.API.Endpoints = f.NewEndpoints()
	err = f.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      ftxAPIURL,
		exchange.WebsocketSpot: ftxWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	f.Websocket = stream.New()
	f.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	f.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	f.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (f *FTX) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		f.SetEnabled(false)
		return nil
	}
	err = f.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsEndpoint, err := f.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	if exch.Websocket != nil && *exch.Websocket {
		err = f.Websocket.Setup(&stream.WebsocketSetup{
			ExchangeConfig:        exch,
			DefaultURL:            ftxWSURL,
			RunningURL:            wsEndpoint,
			Connector:             f.WsConnect,
			Subscriber:            f.Subscribe,
			Unsubscriber:          f.Unsubscribe,
			GenerateSubscriptions: f.GenerateDefaultSubscriptions,
			Features:              &f.Features.Supports.WebsocketCapabilities,
			TradeFeed:             f.Features.Enabled.TradeFeed,
			FillsFeed:             f.Features.Enabled.FillsFeed,
		})
		if err != nil {
			return err
		}
	}

	if err = f.CurrencyPairs.IsAssetEnabled(asset.Futures); err == nil {
		err = f.LoadCollateralWeightings()
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to store collateral weightings. Err: %s",
				f.Name,
				err)
		}
	}
	if exch.Websocket != nil && *exch.Websocket {
		return f.Websocket.SetupNewConnection(stream.ConnectionSetup{
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		})
	}
	return nil
}

// Start starts the FTX go routine
func (f *FTX) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		f.Run()
		wg.Done()
	}()
	return nil
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

	err := f.UpdateOrderExecutionLimits(context.TODO(), "")
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to set exchange order execution limits. Err: %v",
			f.Name,
			err)
	}

	if !f.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err = f.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			f.Name,
			err)
	}

}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (f *FTX) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	if !f.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, f.Name)
	}
	markets, err := f.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}
	format, err := f.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	var pairs []string
	switch a {
	case asset.Spot:
		for x := range markets {
			if markets[x].MarketType == spotString {
				curr, err := currency.NewPairFromString(markets[x].Name)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, format.Format(curr))
			}
		}
	case asset.Futures:
		for x := range markets {
			if markets[x].MarketType == futuresString {
				curr, err := currency.NewPairFromString(markets[x].Name)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, format.Format(curr))
			}
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (f *FTX) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := f.GetAssetTypes(false)
	for x := range assets {
		pairs, err := f.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}
		err = f.UpdatePairs(p, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (f *FTX) UpdateTickers(ctx context.Context, a asset.Item) error {
	allPairs, err := f.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	markets, err := f.GetMarkets(ctx)
	if err != nil {
		return err
	}
	for p := range allPairs {
		formattedPair, err := f.FormatExchangeCurrency(allPairs[p], a)
		if err != nil {
			return err
		}

		for x := range markets {
			if markets[x].Name != formattedPair.String() {
				continue
			}
			var resp ticker.Price
			resp.Pair, err = currency.NewPairFromString(markets[x].Name)
			if err != nil {
				return err
			}
			resp.Last = markets[x].Last
			resp.Bid = markets[x].Bid
			resp.Ask = markets[x].Ask
			resp.LastUpdated = time.Now()
			resp.AssetType = a
			resp.ExchangeName = f.Name
			err = ticker.ProcessTicker(&resp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (f *FTX) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	formattedPair, err := f.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	market, err := f.GetMarket(ctx, formattedPair.String())
	if err != nil {
		return nil, err
	}

	var resp ticker.Price
	resp.Pair, err = currency.NewPairFromString(market.Name)
	if err != nil {
		return nil, err
	}
	resp.Last = market.Last
	resp.Bid = market.Bid
	resp.Ask = market.Ask
	resp.LastUpdated = time.Now()
	resp.AssetType = a
	resp.ExchangeName = f.Name
	err = ticker.ProcessTicker(&resp)
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(f.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (f *FTX) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(f.Name, p, assetType)
	if err != nil {
		return f.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (f *FTX) FetchOrderbook(ctx context.Context, c currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(f.Name, c, assetType)
	if err != nil {
		return f.UpdateOrderbook(ctx, c, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (f *FTX) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        f.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: f.CanVerifyOrderbook,
	}
	formattedPair, err := f.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}
	tempResp, err := f.GetOrderbook(ctx, formattedPair.String(), 100)
	if err != nil {
		return book, err
	}
	for x := range tempResp.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Amount: tempResp.Bids[x].Size,
			Price:  tempResp.Bids[x].Price})
	}
	for y := range tempResp.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Amount: tempResp.Asks[y].Size,
			Price:  tempResp.Asks[y].Price})
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(f.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (f *FTX) UpdateAccountInfo(ctx context.Context, a asset.Item) (account.Holdings, error) {
	var resp account.Holdings

	var data AllWalletBalances
	if f.API.Credentials.Subaccount != "" {
		balances, err := f.GetBalances(ctx)
		if err != nil {
			return resp, err
		}
		data = make(AllWalletBalances)
		data[f.API.Credentials.Subaccount] = balances
	} else {
		// Get all wallet balances used so we can transfer between accounts if
		// needed.
		var err error
		data, err = f.GetAllWalletBalances(ctx)
		if err != nil {
			return resp, err
		}
	}

	for subName, balances := range data {
		// "main" defines the main account in the sub account list
		var acc = account.SubAccount{ID: subName, AssetType: a}
		for x := range balances {
			c := currency.NewCode(balances[x].Coin)
			hold := balances[x].Total - balances[x].Free
			acc.Currencies = append(acc.Currencies,
				account.Balance{CurrencyName: c,
					TotalValue: balances[x].Total,
					Hold:       hold})
		}
		resp.Accounts = append(resp.Accounts, acc)
	}

	resp.Exchange = f.Name
	if err := account.Process(&resp); err != nil {
		return account.Holdings{}, err
	}

	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (f *FTX) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(f.Name, assetType)
	if err != nil {
		return f.UpdateAccountInfo(ctx, assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (f *FTX) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	var resp []exchange.FundHistory
	depositData, err := f.FetchDepositHistory(ctx)
	if err != nil {
		return resp, err
	}
	for x := range depositData {
		var tempData exchange.FundHistory
		tempData.Fee = depositData[x].Fee
		tempData.Timestamp = depositData[x].Time
		tempData.ExchangeName = f.Name
		tempData.CryptoToAddress = depositData[x].Address.Address
		tempData.CryptoTxID = depositData[x].TxID
		tempData.CryptoChain = depositData[x].Address.Method
		tempData.Status = depositData[x].Status
		tempData.Amount = depositData[x].Size
		tempData.Currency = depositData[x].Coin
		tempData.TransferID = strconv.FormatInt(depositData[x].ID, 10)
		resp = append(resp, tempData)
	}
	withdrawalData, err := f.FetchWithdrawalHistory(ctx)
	if err != nil {
		return resp, err
	}
	for y := range withdrawalData {
		var tempData exchange.FundHistory
		tempData.Fee = withdrawalData[y].Fee
		tempData.Timestamp = withdrawalData[y].Time
		tempData.ExchangeName = f.Name
		tempData.CryptoToAddress = withdrawalData[y].Address
		tempData.CryptoTxID = withdrawalData[y].TXID
		tempData.CryptoChain = withdrawalData[y].Method
		tempData.Status = withdrawalData[y].Status
		tempData.Amount = withdrawalData[y].Size
		tempData.Currency = withdrawalData[y].Coin
		tempData.TransferID = strconv.FormatInt(withdrawalData[y].ID, 10)
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (f *FTX) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (f *FTX) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return f.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
// FTX returns trades from the end date and iterates towards the start date
func (f *FTX) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = f.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	endTime := timestampEnd
	var resp []trade.Data
allTrades:
	for {
		var trades []TradeData
		trades, err = f.GetTrades(ctx,
			p.String(),
			timestampStart.Unix(),
			endTime.Unix(),
			0)
		if err != nil {
			if errors.Is(err, errStartTimeCannotBeAfterEndTime) {
				break
			}
			return nil, err
		}
		if len(trades) == 0 {
			break
		}
		for i := 0; i < len(trades); i++ {
			if timestampStart.Equal(trades[i].Time) || trades[i].Time.Before(timestampStart) {
				// reached end of trades to crawl
				break allTrades
			}
			if trades[i].Time.After(endTime) {
				continue
			}
			var side order.Side
			side, err = order.StringToOrderSide(trades[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          strconv.FormatInt(trades[i].ID, 10),
				Exchange:     f.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        trades[i].Price,
				Amount:       trades[i].Size,
				Timestamp:    trades[i].Time,
			})
		}
		endTime = trades[len(trades)-1].Time
	}

	err = f.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (f *FTX) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var resp order.SubmitResponse
	if err := s.Validate(); err != nil {
		return resp, err
	}

	if s.Side == order.Ask {
		s.Side = order.Sell
	}
	if s.Side == order.Bid {
		s.Side = order.Buy
	}

	fPair, err := f.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return resp, err
	}

	tempResp, err := f.Order(ctx,
		fPair.String(),
		s.Side.Lower(),
		s.Type.Lower(),
		s.ReduceOnly,
		s.ImmediateOrCancel,
		s.PostOnly,
		s.ClientOrderID,
		s.Price,
		s.Amount)
	if err != nil {
		return resp, err
	}
	resp.IsOrderPlaced = true
	resp.OrderID = strconv.FormatInt(tempResp.ID, 10)
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (f *FTX) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	if err := action.Validate(); err != nil {
		return order.Modify{}, err
	}

	if action.TriggerPrice != 0 {
		a, err := f.ModifyTriggerOrder(ctx,
			action.ID,
			action.Type.String(),
			action.Amount,
			action.TriggerPrice,
			action.Price,
			0)
		if err != nil {
			return order.Modify{}, err
		}
		return order.Modify{
			Exchange:  action.Exchange,
			AssetType: action.AssetType,
			Pair:      action.Pair,
			ID:        strconv.FormatInt(a.ID, 10),

			Price:        action.Price,
			Amount:       action.Amount,
			TriggerPrice: action.TriggerPrice,
			Type:         action.Type,
		}, err
	}
	var o OrderData
	var err error
	if action.ID == "" {
		o, err = f.ModifyOrderByClientID(ctx,
			action.ClientOrderID,
			action.ClientOrderID,
			action.Price,
			action.Amount)
		if err != nil {
			return order.Modify{}, err
		}
	} else {
		o, err = f.ModifyPlacedOrder(ctx,
			action.ID,
			action.ClientOrderID,
			action.Price,
			action.Amount)
		if err != nil {
			return order.Modify{}, err
		}
	}
	return order.Modify{
		Exchange:  action.Exchange,
		AssetType: action.AssetType,
		Pair:      action.Pair,
		ID:        strconv.FormatInt(o.ID, 10),

		Price:  action.Price,
		Amount: action.Amount,
	}, err
}

// CancelOrder cancels an order by its corresponding ID number
func (f *FTX) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	if o.ClientOrderID != "" {
		_, err := f.DeleteOrderByClientID(ctx, o.ClientOrderID)
		return err
	}

	_, err := f.DeleteOrder(ctx, o.ID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (f *FTX) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (f *FTX) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}

	var resp order.CancelAllResponse
	formattedPair, err := f.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
	if err != nil {
		return resp, err
	}
	orders, err := f.GetOpenOrders(ctx, formattedPair.String())
	if err != nil {
		return resp, err
	}

	tempMap := make(map[string]string)
	for x := range orders {
		_, err := f.DeleteOrder(ctx, strconv.FormatInt(orders[x].ID, 10))
		if err != nil {
			tempMap[strconv.FormatInt(orders[x].ID, 10)] = "Cancellation Failed"
			continue
		}
		tempMap[strconv.FormatInt(orders[x].ID, 10)] = "Success"
	}
	resp.Status = tempMap
	return resp, nil
}

// GetCompatible gets compatible variables for order vars
func (s *OrderData) GetCompatible(ctx context.Context, f *FTX) (OrderVars, error) {
	var resp OrderVars
	switch s.Side {
	case order.Buy.Lower():
		resp.Side = order.Buy
	case order.Sell.Lower():
		resp.Side = order.Sell
	default:
		resp.Side = order.UnknownSide
	}
	switch s.Status {
	case strings.ToLower(order.New.String()):
		resp.Status = order.New
	case strings.ToLower(order.Open.String()):
		resp.Status = order.Open
	case closedStatus:
		if s.FilledSize != 0 && s.FilledSize != s.Size {
			resp.Status = order.PartiallyCancelled
		}
		if s.FilledSize == 0 {
			resp.Status = order.Cancelled
		}
		if s.FilledSize == s.Size {
			resp.Status = order.Filled
		}
	default:
		resp.Status = order.AnyStatus
	}
	var feeBuilder exchange.FeeBuilder
	feeBuilder.PurchasePrice = s.AvgFillPrice
	feeBuilder.Amount = s.Size
	resp.OrderType = order.Market
	if strings.EqualFold(s.OrderType, order.Limit.String()) {
		resp.OrderType = order.Limit
		feeBuilder.IsMaker = true
	}
	fee, err := f.GetFee(ctx, &feeBuilder)
	if err != nil {
		return resp, err
	}
	resp.Fee = fee
	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (f *FTX) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail
	orderData, err := f.GetOrderStatus(ctx, orderID)
	if err != nil {
		return resp, err
	}
	p, err := currency.NewPairFromString(orderData.Market)
	if err != nil {
		return resp, err
	}
	orderAssetType, err := f.GetPairAssetType(p)
	if err != nil {
		return resp, err
	}
	resp.ID = strconv.FormatInt(orderData.ID, 10)
	resp.Amount = orderData.Size
	resp.ClientOrderID = orderData.ClientID
	resp.Date = orderData.CreatedAt
	resp.Exchange = f.Name
	resp.ExecutedAmount = orderData.Size - orderData.RemainingSize
	resp.Pair = p
	resp.AssetType = orderAssetType
	resp.Price = orderData.Price
	resp.RemainingAmount = orderData.RemainingSize
	orderVars, err := orderData.GetCompatible(ctx, f)
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
func (f *FTX) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	a, err := f.FetchDepositAddress(ctx, cryptocurrency, chain)
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: a.Address,
		Tag:     a.Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *FTX) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := f.Withdraw(ctx,
		withdrawRequest.Currency,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.TradePassword,
		withdrawRequest.Crypto.Chain,
		strconv.FormatInt(withdrawRequest.OneTimePassword, 10),
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID:     strconv.FormatInt(resp.ID, 10),
		Status: resp.Status,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (f *FTX) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (f *FTX) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWebsocket returns a pointer to the exchange websocket
func (f *FTX) GetWebsocket() (*stream.Websocket, error) {
	return f.Websocket, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (f *FTX) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}

	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		assetType, err := f.GetPairAssetType(getOrdersRequest.Pairs[x])
		if err != nil {
			return resp, err
		}

		formattedPair, err := f.FormatExchangeCurrency(getOrdersRequest.Pairs[x], assetType)
		if err != nil {
			return nil, err
		}

		var tempResp order.Detail
		orderData, err := f.GetOpenOrders(ctx, formattedPair.String())
		if err != nil {
			return resp, err
		}
		for y := range orderData {
			var p currency.Pair
			p, err = currency.NewPairFromString(orderData[y].Market)
			if err != nil {
				return nil, err
			}

			tempResp.ID = strconv.FormatInt(orderData[y].ID, 10)
			tempResp.Amount = orderData[y].Size
			tempResp.AssetType = assetType
			tempResp.ClientOrderID = orderData[y].ClientID
			tempResp.Date = orderData[y].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = orderData[y].Size - orderData[y].RemainingSize
			tempResp.Pair = p
			tempResp.Price = orderData[y].Price
			tempResp.RemainingAmount = orderData[y].RemainingSize
			var orderVars OrderVars
			orderVars, err = f.compatibleOrderVars(ctx,
				orderData[y].Side,
				orderData[y].Status,
				orderData[y].OrderType,
				orderData[y].Size,
				orderData[y].FilledSize,
				orderData[y].AvgFillPrice)
			if err != nil {
				return resp, err
			}
			tempResp.Status = orderVars.Status
			tempResp.Side = orderVars.Side
			tempResp.Type = orderVars.OrderType
			tempResp.Fee = orderVars.Fee
			resp = append(resp, tempResp)
		}

		triggerOrderData, err := f.GetOpenTriggerOrders(ctx,
			formattedPair.String(),
			getOrdersRequest.Type.String())
		if err != nil {
			return resp, err
		}
		for z := range triggerOrderData {
			var p currency.Pair
			p, err = currency.NewPairFromString(triggerOrderData[z].Market)
			if err != nil {
				return nil, err
			}
			tempResp.ID = strconv.FormatInt(triggerOrderData[z].ID, 10)
			tempResp.Amount = triggerOrderData[z].Size
			tempResp.AssetType = assetType
			tempResp.Date = triggerOrderData[z].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = triggerOrderData[z].FilledSize
			tempResp.Pair = p
			tempResp.Price = triggerOrderData[z].AvgFillPrice
			tempResp.RemainingAmount = triggerOrderData[z].Size - triggerOrderData[z].FilledSize
			tempResp.TriggerPrice = triggerOrderData[z].TriggerPrice
			orderVars, err := f.compatibleOrderVars(ctx,
				triggerOrderData[z].Side,
				triggerOrderData[z].Status,
				triggerOrderData[z].OrderType,
				triggerOrderData[z].Size,
				triggerOrderData[z].FilledSize,
				triggerOrderData[z].AvgFillPrice)
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
func (f *FTX) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		var tempResp order.Detail
		assetType, err := f.GetPairAssetType(getOrdersRequest.Pairs[x])
		if err != nil {
			return resp, err
		}

		formattedPair, err := f.FormatExchangeCurrency(getOrdersRequest.Pairs[x],
			assetType)
		if err != nil {
			return nil, err
		}

		orderData, err := f.FetchOrderHistory(ctx,
			formattedPair.String(),
			getOrdersRequest.StartTime,
			getOrdersRequest.EndTime,
			"")
		if err != nil {
			return resp, err
		}
		for y := range orderData {
			var p currency.Pair
			p, err = currency.NewPairFromString(orderData[y].Market)
			if err != nil {
				return nil, err
			}
			tempResp.ID = strconv.FormatInt(orderData[y].ID, 10)
			tempResp.Amount = orderData[y].Size
			tempResp.AssetType = assetType
			tempResp.AverageExecutedPrice = orderData[y].AvgFillPrice
			tempResp.ClientOrderID = orderData[y].ClientID
			tempResp.Date = orderData[y].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = orderData[y].Size - orderData[y].RemainingSize
			tempResp.Pair = p
			tempResp.Price = orderData[y].Price
			tempResp.RemainingAmount = orderData[y].RemainingSize
			var orderVars OrderVars
			orderVars, err = f.compatibleOrderVars(ctx,
				orderData[y].Side,
				orderData[y].Status,
				orderData[y].OrderType,
				orderData[y].Size,
				orderData[y].FilledSize,
				orderData[y].AvgFillPrice)
			if err != nil {
				return resp, err
			}
			tempResp.Status = orderVars.Status
			tempResp.Side = orderVars.Side
			tempResp.Type = orderVars.OrderType
			tempResp.Fee = orderVars.Fee
			resp = append(resp, tempResp)
		}
		triggerOrderData, err := f.GetTriggerOrderHistory(ctx,
			formattedPair.String(),
			getOrdersRequest.StartTime,
			getOrdersRequest.EndTime,
			strings.ToLower(getOrdersRequest.Side.String()),
			strings.ToLower(getOrdersRequest.Type.String()),
			"")
		if err != nil {
			return resp, err
		}
		for z := range triggerOrderData {
			var p currency.Pair
			p, err = currency.NewPairFromString(triggerOrderData[z].Market)
			if err != nil {
				return nil, err
			}
			tempResp.ID = strconv.FormatInt(triggerOrderData[z].ID, 10)
			tempResp.Amount = triggerOrderData[z].Size
			tempResp.AssetType = assetType
			tempResp.Date = triggerOrderData[z].CreatedAt
			tempResp.Exchange = f.Name
			tempResp.ExecutedAmount = triggerOrderData[z].FilledSize
			tempResp.Pair = p
			tempResp.Price = triggerOrderData[z].AvgFillPrice
			tempResp.RemainingAmount = triggerOrderData[z].Size - triggerOrderData[z].FilledSize
			tempResp.TriggerPrice = triggerOrderData[z].TriggerPrice
			orderVars, err := f.compatibleOrderVars(ctx,
				triggerOrderData[z].Side,
				triggerOrderData[z].Status,
				triggerOrderData[z].OrderType,
				triggerOrderData[z].Size,
				triggerOrderData[z].FilledSize,
				triggerOrderData[z].AvgFillPrice)
			if err != nil {
				return resp, err
			}
			tempResp.Status = orderVars.Status
			tempResp.Side = orderVars.Side
			tempResp.Type = orderVars.OrderType
			tempResp.Fee = orderVars.Fee
			tempResp.InferCostsAndTimes()
			resp = append(resp, tempResp)
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (f *FTX) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	return f.GetFee(ctx, feeBuilder)
}

// SubscribeToWebsocketChannels appends to ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle subscribing
func (f *FTX) SubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	return f.Websocket.SubscribeToChannels(channels)
}

// UnsubscribeToWebsocketChannels removes from ChannelsToSubscribe
// which lets websocket.manageSubscriptions handle unsubscribing
func (f *FTX) UnsubscribeToWebsocketChannels(channels []stream.ChannelSubscription) error {
	return f.Websocket.UnsubscribeChannels(channels)
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (f *FTX) AuthenticateWebsocket(_ context.Context) error {
	return f.WsAuth()
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (f *FTX) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := f.UpdateAccountInfo(ctx, assetType)
	return f.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (f *FTX) GetHistoricCandles(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := f.ValidateKline(p, a, interval); err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := f.FormatExchangeCurrency(p, a)
	if err != nil {
		return kline.Item{}, err
	}

	ohlcData, err := f.GetHistoricalData(ctx,
		formattedPair.String(),
		int64(interval.Duration().Seconds()),
		int64(f.Features.Enabled.Kline.ResultLimit),
		start,
		end)
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: f.Name,
		Pair:     p,
		Asset:    a,
		Interval: interval,
	}

	for x := range ohlcData {
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   ohlcData[x].StartTime,
			Open:   ohlcData[x].Open,
			High:   ohlcData[x].High,
			Low:    ohlcData[x].Low,
			Close:  ohlcData[x].Close,
			Volume: ohlcData[x].Volume,
		})
	}
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (f *FTX) GetHistoricCandlesExtended(ctx context.Context, p currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := f.ValidateKline(p, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: f.Name,
		Pair:     p,
		Asset:    a,
		Interval: interval,
	}

	dates, err := kline.CalculateCandleDateRanges(start, end, interval, f.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}

	formattedPair, err := f.FormatExchangeCurrency(p, a)
	if err != nil {
		return kline.Item{}, err
	}

	for x := range dates.Ranges {
		var ohlcData []OHLCVData
		ohlcData, err = f.GetHistoricalData(ctx,
			formattedPair.String(),
			int64(interval.Duration().Seconds()),
			int64(f.Features.Enabled.Kline.ResultLimit),
			dates.Ranges[x].Start.Time,
			dates.Ranges[x].End.Time)
		if err != nil {
			return kline.Item{}, err
		}

		for i := range ohlcData {
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   ohlcData[i].StartTime,
				Open:   ohlcData[i].Open,
				High:   ohlcData[i].High,
				Low:    ohlcData[i].Low,
				Close:  ohlcData[i].Close,
				Volume: ohlcData[i].Volume,
			})
		}
	}
	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", f.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (f *FTX) UpdateOrderExecutionLimits(ctx context.Context, _ asset.Item) error {
	limits, err := f.FetchExchangeLimits(ctx)
	if err != nil {
		return fmt.Errorf("cannot update exchange execution limits: %w", err)
	}
	return f.LoadLimits(limits)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (f *FTX) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	coins, err := f.GetCoins(ctx)
	if err != nil {
		return nil, err
	}

	var availableChains []string
	for x := range coins {
		if strings.EqualFold(coins[x].ID, cryptocurrency.String()) {
			for y := range coins[x].Methods {
				availableChains = append(availableChains, coins[x].Methods[y])
			}
		}
	}
	return availableChains, nil
}

// CalculatePNL uses a high-tech algorithm to calculate your pnl
func (f *FTX) CalculatePNL(pnl *order.PNLCalculatorRequest) (*order.PNLResult, error) {
	if pnl == nil {
		return nil, fmt.Errorf("%v %w", f.Name, order.ErrNilPNLCalculator)
	}
	var result order.PNLResult
	result.Time = pnl.Time
	if pnl.CalculateOffline {
		result.Fee = pnl.Fee
		result.UnrealisedPNL = pnl.Amount.Mul(pnl.CurrentPrice.Sub(pnl.PreviousPrice))
		result.Price = pnl.CurrentPrice
		ftxPNLCalculation := pnl.Amount.Mul(pnl.CurrentPrice.Sub(pnl.EntryPrice))
		result.RealisedPNLBeforeFees = ftxPNLCalculation.Sub(result.UnrealisedPNL)
		return &result, nil
	}

	ep := pnl.EntryPrice.InexactFloat64()
	info, err := f.GetAccountInfo(context.Background())
	if err != nil {
		return nil, err
	}
	if info.Liquidating || info.Collateral == 0 {
		return &result, order.ErrPositionLiquidated
	}
	for i := range info.Positions {
		pair, err := currency.NewPairFromString(info.Positions[i].Future)
		if err != nil {
			return nil, err
		}
		if !pnl.Pair.Equal(pair) {
			continue
		}
		if info.Positions[i].EntryPrice == ep {
			result.UnrealisedPNL = decimal.NewFromFloat(info.Positions[i].UnrealizedPNL)
			result.RealisedPNLBeforeFees = decimal.NewFromFloat(info.Positions[i].RealizedPNL)
			result.Price = decimal.NewFromFloat(info.Positions[i].Cost)
		}
	}
	return &result, nil
}

// ScaleCollateral takes your totals and scales them according to FTX's rules
func (f *FTX) ScaleCollateral(calc *order.CollateralCalculator) (decimal.Decimal, error) {
	var result decimal.Decimal
	collateralWeight, ok := f.collateralWeight[calc.CollateralCurrency.Upper().String()]
	if !ok {
		return decimal.Zero, errCoinMustBeSpecified
	}
	if calc.CollateralAmount.IsPositive() {
		if collateralWeight.IMFFactor == 0 {
			return decimal.Zero, errCoinMustBeSpecified
		}
		var scaling decimal.Decimal
		if calc.IsLiquidating {
			scaling = decimal.NewFromFloat(collateralWeight.Total)
		} else {
			scaling = decimal.NewFromFloat(collateralWeight.Initial)
		}
		weight := decimal.NewFromFloat(1.1 / (1 + collateralWeight.IMFFactor*math.Sqrt(calc.CollateralAmount.InexactFloat64())))
		result = calc.CollateralAmount.Mul(calc.USDPrice).Mul(decimal.Min(scaling, weight))
	} else {
		result = result.Add(calc.CollateralAmount.Mul(calc.USDPrice))
	}
	return result, nil
}

// CalculateTotalCollateral scales collateral and determines how much you have in USD (maybe)
func (f *FTX) CalculateTotalCollateral(collateralAssets []order.CollateralCalculator) (decimal.Decimal, error) {
	var result decimal.Decimal
	for i := range collateralAssets {
		collateral, err := f.ScaleCollateral(&collateralAssets[i])
		if err != nil {
			return decimal.Zero, err
		}
		result = result.Add(collateral)
	}
	return result, nil
}

func (f *FTX) GetFuturesPositions(a asset.Item, cp currency.Pair, start, end time.Time) ([]order.Detail, error) {
	if !a.IsFutures() {
		return nil, fmt.Errorf("%w futures asset type only", common.ErrFunctionNotSupported)
	}
	fills, err := f.GetFills(context.Background(), cp, "200", start, end)
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	var side order.Side
	for i := range fills {
		price := fills[i].Price
		side, err = order.StringToOrderSide(fills[i].Side)
		if err != nil {
			return nil, err
		}
		resp = append(resp, order.Detail{
			Side:      side,
			Pair:      cp,
			ID:        fmt.Sprintf("%v", fills[i].ID),
			Price:     price,
			Amount:    fills[i].Size,
			AssetType: a,
			Exchange:  f.Name,
			Fee:       fills[i].Fee,
			Date:      fills[i].Time,
		})
	}

	return resp, nil
}
