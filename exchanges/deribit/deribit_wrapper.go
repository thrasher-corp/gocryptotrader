package deribit

import (
	"context"
	"fmt"
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
func (d *Deribit) GetDefaultConfig() (*config.Exchange, error) {
	d.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = d.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = d.BaseCurrencies
	err := d.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}
	if d.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := d.UpdateTradablePairs(context.Background(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Deribit
func (d *Deribit) SetDefaults() {
	d.Name = "Deribit"
	d.Enabled = true
	d.Verbose = true
	d.API.CredentialsValidator.RequiresKey = true
	d.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := d.SetGlobalPairsManager(requestFmt, configFmt, asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	d.Features = exchange.Features{
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
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
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
	d.Requester, err = request.New(d.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	d.API.Endpoints = d.NewEndpoints()
	err = d.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestFutures: deribitAPIURL,
		exchange.RestSpot:    deribitTestAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	d.Websocket = stream.New()
	d.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	d.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	d.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (d *Deribit) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		d.SetEnabled(false)
		return nil
	}
	err = d.SetupDefaults(exch)
	if err != nil {
		return err
	}
	err = d.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            deribitWebsocketAddress,
		RunningURL:            deribitWebsocketAddress,
		Connector:             d.WsConnect,
		Subscriber:            d.Subscribe,
		Unsubscriber:          d.Unsubscribe,
		GenerateSubscriptions: d.GenerateDefaultSubscriptions,
		Features:              &d.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}
	return d.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            rateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Deribit go routine
func (d *Deribit) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return common.ErrNilPointer
	}
	wg.Add(1)
	go func() {
		d.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Deribit wrapper
func (d *Deribit) Run() {
	if d.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			d.Name,
			common.IsEnabled(d.Websocket.IsEnabled()))
		d.PrintEnabledPairs()
	}
	if !d.GetEnabledFeatures().AutoPairUpdates {
		return
	}
	err := d.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			d.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (d *Deribit) FetchTradablePairs(ctx context.Context, assetType asset.Item) ([]string, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, assetType.String())
	}
	var resp []string
	switch assetType {
	case asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo:
		for _, x := range []string{"BTC", "SOL", "ETH", "USDC"} {
			instrumentsData, err := d.GetInstrumentsData(ctx, x, d.GetAssetKind(assetType), false)
			if err != nil && len(resp) == 0 {
				return nil, err
			}
			for y := range instrumentsData {
				resp = append(resp, instrumentsData[y].InstrumentName)
			}
		}
	}
	return resp, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (d *Deribit) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := d.GetAssetTypes(false)
	for x := range assets {
		pairs, err := d.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}
		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}
		err = d.UpdatePairs(p, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (d *Deribit) UpdateTickers(ctx context.Context, a asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (d *Deribit) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, assetType)
	}
	switch assetType {
	case asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo:
		if p.IsEmpty() {
			return nil, fmt.Errorf("pair provided is empty")
		}
		fmtPair, err := d.FormatExchangeCurrency(p, asset.Futures)
		if err != nil {
			return nil, err
		}
		tickerData, err := d.GetPublicTicker(ctx, fmtPair.String())
		if err != nil {
			return nil, err
		}
		var resp ticker.Price
		resp.ExchangeName = d.Name
		resp.Pair = p
		resp.AssetType = assetType
		resp.Ask = tickerData.BestAskPrice
		resp.AskSize = tickerData.BestAskAmount
		resp.Bid = tickerData.BestBidPrice
		resp.BidSize = tickerData.BestBidAmount
		resp.High = tickerData.Stats.High
		resp.Low = tickerData.Stats.Low
		resp.Last = tickerData.LastPrice
		err = ticker.ProcessTicker(&resp)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, assetType)
	}
	return ticker.GetTicker(d.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (d *Deribit) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(d.Name, p, assetType)
	if err != nil {
		return d.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (d *Deribit) FetchOrderbook(ctx context.Context, currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(d.Name, currency, assetType)
	if err != nil {
		return d.UpdateOrderbook(ctx, currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (d *Deribit) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        d.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: d.CanVerifyOrderbook,
	}

	switch assetType {
	case asset.Futures:
		fmtPair, err := d.FormatExchangeCurrency(p, assetType)
		if err != nil {
			return nil, err
		}

		obData, err := d.GetOrderbookData(ctx, fmtPair.String(), 50)
		if err != nil {
			return nil, err
		}

		for x := range obData.Asks {
			book.Asks = append(book.Asks, orderbook.Item{
				Price:  obData.Asks[x][0],
				Amount: obData.Asks[x][1],
			})
		}

		for x := range obData.Bids {
			book.Bids = append(book.Bids, orderbook.Item{
				Price:  obData.Bids[x][0],
				Amount: obData.Bids[x][1],
			})
		}

		err = book.Process()
		if err != nil {
			return book, err
		}
	default:
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	return orderbook.Get(d.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (d *Deribit) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	if !d.SupportsAsset(assetType) {
		return account.Holdings{}, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, assetType)
	}
	var resp account.Holdings
	resp.Exchange = d.Name
	currencies, err := d.GetCurrencies(ctx)
	if err != nil {
		return resp, err
	}
	for x := range currencies {
		data, err := d.GetAccountSummary(ctx, currencies[x].Currency, false)
		if err != nil {
			return resp, err
		}

		var subAcc account.SubAccount
		subAcc.AssetType = asset.Futures
		subAcc.Currencies = append(subAcc.Currencies, account.Balance{
			CurrencyName: currency.NewCode(currencies[x].Currency),
			Total:        data.Balance,
			Hold:         data.Balance - data.AvailableFunds,
		})
	}
	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (d *Deribit) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := d.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	accountData, err := account.GetHoldings(d.Name, creds, assetType)
	if err != nil {
		return d.UpdateAccountInfo(ctx, assetType)
	}
	return accountData, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (d *Deribit) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	currencies, err := d.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	var resp []exchange.FundHistory
	for x := range currencies {
		deposits, err := d.GetDeposits(ctx, currencies[x].Currency, 100, 0)
		if err != nil {
			return nil, err
		}
		for y := range deposits.Data {
			resp = append(resp, exchange.FundHistory{
				ExchangeName:    d.Name,
				Status:          deposits.Data[y].State,
				TransferID:      deposits.Data[y].TransactionID,
				Timestamp:       time.Unix(deposits.Data[y].UpdatedTimestamp/1000, 0),
				Currency:        currencies[x].Currency,
				Amount:          deposits.Data[y].Amount,
				CryptoToAddress: deposits.Data[y].Address,
				TransferType:    "deposit",
			})
		}
		withdrawalData, err := d.GetWithdrawals(ctx, currencies[x].Currency, 100, 0)
		if err != nil {
			return nil, err
		}

		for z := range withdrawalData.Data {
			resp = append(resp, exchange.FundHistory{
				ExchangeName:    d.Name,
				Status:          withdrawalData.Data[z].State,
				TransferID:      withdrawalData.Data[z].TransactionID,
				Timestamp:       time.Unix(withdrawalData.Data[z].UpdatedTimestamp/1000, 0),
				Currency:        currencies[x].Currency,
				Amount:          withdrawalData.Data[z].Amount,
				CryptoToAddress: withdrawalData.Data[z].Address,
				TransferType:    "deposit",
			})
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (d *Deribit) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	currencies, err := d.GetCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	var resp []exchange.WithdrawalHistory
	for x := range currencies {
		if !strings.EqualFold(currencies[x].Currency, c.String()) {
			continue
		}
		withdrawalData, err := d.GetWithdrawals(ctx, currencies[x].Currency, 100, 0)
		if err != nil {
			return nil, err
		}
		for y := range withdrawalData.Data {
			resp = append(resp, exchange.WithdrawalHistory{
				Status:          withdrawalData.Data[y].State,
				TransferID:      withdrawalData.Data[y].TransactionID,
				Timestamp:       time.Unix(withdrawalData.Data[y].UpdatedTimestamp/1000, 0),
				Currency:        currencies[x].Currency,
				Amount:          withdrawalData.Data[y].Amount,
				CryptoToAddress: withdrawalData.Data[y].Address,
				TransferType:    "deposit",
			})
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (d *Deribit) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, d.Name)
	}
	format, err := d.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for _, x := range []string{"BTC", "SOL", "ETH", "USDC"} {
		instrumentsData, err := d.GetInstrumentsData(ctx, x, d.GetAssetKind(assetType), false)
		if err != nil {
			return nil, err
		}
		for y := range instrumentsData {
			if strings.EqualFold(format.Format(p), instrumentsData[y].InstrumentName) {
				trades, err := d.GetLastTradesByInstrument(
					ctx,
					instrumentsData[y].InstrumentName,
					"",
					"",
					"",
					0,
					false)
				if err != nil {
					return nil, err
				}
				for a := range trades.Trades {
					sideData := order.Sell
					if trades.Trades[a].Direction == sideBUY {
						sideData = order.Buy
					}
					resp = append(resp, trade.Data{
						TID:          trades.Trades[a].TradeID,
						Exchange:     d.Name,
						Price:        trades.Trades[a].Price,
						Amount:       trades.Trades[a].Amount,
						Timestamp:    time.Unix(trades.Trades[a].Timestamp/1000, 0),
						AssetType:    assetType,
						Side:         sideData,
						CurrencyPair: p,
					})
				}
			}
		}
	}
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (d *Deribit) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if timestampStart.Equal(timestampEnd) ||
		timestampEnd.After(time.Now()) ||
		timestampEnd.Before(timestampStart) ||
		(timestampStart.IsZero() && !timestampEnd.IsZero()) {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v",
			timestampStart,
			timestampEnd)
	}
	fmtPair, err := d.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	var tradesData *PublicTradesData
	var hasMore = true
	for hasMore {
		tradesData, err = d.GetLastTradesByInstrumentAndTime(ctx, fmtPair.String(),
			"asc",
			100,
			false,
			timestampStart,
			timestampEnd)
		if err != nil {
			return nil, err
		}
		if len(tradesData.Trades) != 100 {
			hasMore = false
		}
		for t := range tradesData.Trades {
			if t == 99 {
				timestampStart = time.Unix(tradesData.Trades[t].Timestamp/1000, 0)
			}
			sideData := order.Sell
			if tradesData.Trades[t].Direction == sideBUY {
				sideData = order.Buy
			}
			resp = append(resp, trade.Data{
				TID:          tradesData.Trades[t].TradeID,
				Exchange:     d.Name,
				Price:        tradesData.Trades[t].Price,
				Amount:       tradesData.Trades[t].Amount,
				Timestamp:    time.Unix(tradesData.Trades[t].Timestamp/1000, 0),
				AssetType:    assetType,
				Side:         sideData,
				CurrencyPair: p,
			})
		}
	}
	return resp, nil
}

// SubmitOrder submits a new order
func (d *Deribit) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	var orderID string
	var fmtPair currency.Pair
	status := order.New
	switch s.AssetType {
	case asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo:
		fmtPair, err = d.FormatExchangeCurrency(s.Pair, asset.Futures)
		if err != nil {
			return nil, err
		}
		timeInForce := ""
		if s.ImmediateOrCancel {
			timeInForce = "immediate_or_cancel"
		}
		var data *PrivateTradeData
		switch s.Side {
		case order.Bid, order.Buy:
			if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				data, err = d.wsPlaceOrder(fmtPair.String(),
					strings.ToLower(s.Type.String()),
					s.ClientOrderID,
					timeInForce, "", "",
					s.Amount,
					s.Price,
					0,
					s.TriggerPrice,
					s.PostOnly,
					false,
					s.ReduceOnly,
					false)
				if err != nil {
					return nil, err
				}
				orderID = data.Order.OrderID
			} else {
				data, err = d.SubmitBuy(ctx, fmtPair.String(),
					strings.ToLower(s.Type.String()),
					s.ClientOrderID,
					timeInForce, "", "",
					s.Amount,
					s.Price,
					0,
					s.TriggerPrice,
					s.PostOnly,
					false,
					s.ReduceOnly,
					false)
				if err != nil {
					return nil, err
				}
				orderID = data.Order.OrderID
			}
		case order.Sell, order.Ask:
			data, err = d.SubmitSell(ctx, fmtPair.String(),
				s.Type.String(),
				s.ClientOrderID,
				"", "", "",
				s.Amount,
				s.Price,
				0,
				s.TriggerPrice,
				s.PostOnly,
				false,
				s.ReduceOnly,
				false)
			if err != nil {
				return nil, err
			}
			orderID = data.Order.OrderID
		}
	default:
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, s.AssetType)
	}
	resp, err := s.DeriveSubmitResponse(orderID)
	if err != nil {
		return nil, err
	}
	resp.Status = status
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (d *Deribit) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var modify *PrivateTradeData
	var err error
	switch action.AssetType {
	case asset.Futures:
		modify, err = d.SubmitEdit(ctx, action.OrderID,
			"",
			action.Amount,
			action.Price,
			action.TriggerPrice,
			action.PostOnly,
			false,
			false,
			false)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, action.AssetType)
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = modify.Order.OrderID
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (d *Deribit) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Futures:
		_, err := d.SubmitCancel(ctx, ord.OrderID)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, ord.AssetType)
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (d *Deribit) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	var resp = order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	for x := range orders {
		if orders[x].AssetType.IsValid() {
			_, err := d.SubmitCancel(ctx, orders[x].OrderID)
			if err != nil {
				resp.Status[orders[x].OrderID] = err.Error()
			} else {
				resp.Status[orders[x].OrderID] = "successfully cancelled"
			}
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (d *Deribit) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var cancelData int64
	switch orderCancellation.AssetType {
	case asset.Futures:
		pairFmt, err := d.GetPairFormat(orderCancellation.AssetType, true)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		var orderTypeStr string
		switch orderCancellation.Type {
		case order.Limit:
			orderTypeStr = order.Limit.String()
		case order.Market:
			orderTypeStr = order.Market.String()
		case order.AnyType:
			orderTypeStr = "all"
		default:
			return order.CancelAllResponse{}, fmt.Errorf("%s: orderType %v is not valid", d.Name, orderCancellation.Type)
		}
		cancelData, err = d.SubmitCancelAllByInstrument(ctx, pairFmt.Format(orderCancellation.Pair), orderTypeStr)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
	default:
		return order.CancelAllResponse{}, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, orderCancellation.AssetType)
	}
	return order.CancelAllResponse{Count: cancelData}, nil
}

// GetOrderInfo returns order information based on order ID
func (d *Deribit) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail
	switch assetType {
	case asset.Futures:
		orderInfo, err := d.GetOrderState(ctx, orderID)
		if err != nil {
			return resp, err
		}
		orderSide := order.Sell
		if orderInfo.Direction == sideBUY {
			orderSide = order.Buy
		}
		var orderType order.Type
		switch orderInfo.OrderType {
		case "market":
			orderType = order.Market
		case "limit":
			orderType = order.Limit
		case "stop_limit":
			orderType = order.StopLimit
		case "stop_market":
			orderType = order.StopMarket
		default:
			return resp, fmt.Errorf("%v: orderType %s not supported", d.Name, orderInfo.OrderType)
		}
		var orderStatus order.Status
		switch orderInfo.OrderState {
		case "open":
			orderStatus = order.Active
		case "filled":
			orderStatus = order.Filled
		case "rejected":
			orderStatus = order.Rejected
		case "cancelled":
			orderStatus = order.Cancelled
		case "untriggered":
			orderStatus = order.UnknownStatus
		default:
			return resp, fmt.Errorf("%v: orderStatus %s not supported", d.Name, orderInfo.OrderState)
		}
		resp = order.Detail{
			AssetType:       asset.Futures,
			Exchange:        d.Name,
			PostOnly:        orderInfo.PostOnly,
			Price:           orderInfo.Price,
			Amount:          orderInfo.Amount,
			ExecutedAmount:  orderInfo.FilledAmount,
			Fee:             orderInfo.Commission,
			RemainingAmount: orderInfo.Amount - orderInfo.FilledAmount,
			OrderID:         orderInfo.OrderID,
			Pair:            pair,
			LastUpdated:     time.Unix(orderInfo.LastUpdateTimestamp/1000, 0),
			Side:            orderSide,
			Type:            orderType,
			Status:          orderStatus,
		}
	default:
		return resp, fmt.Errorf("%s: orderType %v is not valid", d.Name, assetType)
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (d *Deribit) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, accountID string, chain string) (*deposit.Address, error) {
	addressData, err := d.GetCurrentDepositAddress(ctx, cryptocurrency.String())
	return &deposit.Address{
		Address: addressData.Address,
		Chain:   addressData.Currency,
	}, err
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (d *Deribit) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawData, err := d.SubmitWithdraw(
		ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		"",
		withdrawRequest.Amount)
	return &withdraw.ExchangeResponse{
		ID:     strconv.FormatInt(withdrawData.ID, 10),
		Status: withdrawData.State,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (d *Deribit) WithdrawFiatFunds(ctx context.Context, request *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (d *Deribit) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (d *Deribit) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail
	switch getOrdersRequest.AssetType {
	case asset.Futures, asset.Options, asset.FutureCombo, asset.OptionCombo:
		for x := range getOrdersRequest.Pairs {
			fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Futures)
			if err != nil {
				return nil, err
			}
			ordersData, err := d.GetOpenOrdersByInstrument(ctx, fmtPair.String(), getOrdersRequest.Type.Lower())
			if err != nil {
				return nil, err
			}
			for y := range ordersData {
				orderSide := order.Sell
				if ordersData[y].Direction == sideBUY {
					orderSide = order.Buy
				}
				if getOrdersRequest.Side != orderSide || getOrdersRequest.Side != order.AnySide {
					continue
				}
				var orderType order.Type
				switch ordersData[y].OrderType {
				case "market":
					orderType = order.Market
				case "limit":
					orderType = order.Limit
				case "stop_limit":
					orderType = order.StopLimit
				case "stop_market":
					orderType = order.StopMarket
				default:
					return resp, fmt.Errorf("%v: orderType %s not supported", d.Name, ordersData[y].OrderType)
				}
				if getOrdersRequest.Type != orderType || getOrdersRequest.Type != order.AnyType {
					continue
				}
				var orderStatus order.Status
				if !strings.EqualFold(ordersData[y].OrderState, "open") {
					continue
				}
				// switch ordersData[y].OrderState {
				// case "filled":
				// 	orderStatus = order.Filled
				// case "rejected":
				// 	orderStatus = order.Rejected
				// case "cancelled":
				// 	orderStatus = order.Cancelled
				// case "untriggered":
				// 	orderStatus = order.UnknownStatus
				// default:
				// 	return resp, fmt.Errorf("%v: orderStatus %s not supported", d.Name, ordersData[y].OrderState)
				// }
				resp = append(resp, order.Detail{
					AssetType:       asset.Futures,
					Exchange:        d.Name,
					PostOnly:        ordersData[y].PostOnly,
					Price:           ordersData[y].Price,
					Amount:          ordersData[y].Amount,
					ExecutedAmount:  ordersData[y].FilledAmount,
					Fee:             ordersData[y].Commission,
					RemainingAmount: ordersData[y].Amount - ordersData[y].FilledAmount,
					OrderID:         ordersData[y].OrderID,
					Pair:            getOrdersRequest.Pairs[x],
					LastUpdated:     time.Unix(ordersData[y].LastUpdateTimestamp/1000, 0),
					Side:            orderSide,
					Type:            orderType,
					Status:          orderStatus,
				})
			}
		}
	default:
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (d *Deribit) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Futures)
		if err != nil {
			return nil, err
		}
		ordersData, err := d.GetOrderHistoryByInstrument(ctx, fmtPair.String(), 100, 0, true, true)
		if err != nil {
			return nil, err
		}
		for y := range ordersData {
			orderSide := order.Sell
			if ordersData[y].Direction == sideBUY {
				orderSide = order.Buy
			}
			if getOrdersRequest.Side != orderSide || getOrdersRequest.Side != order.AnySide {
				continue
			}
			var orderType order.Type
			switch ordersData[y].OrderType {
			case "market":
				orderType = order.Market
			case "limit":
				orderType = order.Limit
			case "stop_limit":
				orderType = order.StopLimit
			case "stop_market":
				orderType = order.StopMarket
			default:
				return resp, fmt.Errorf("%v: orderType %s not supported", d.Name, ordersData[y].OrderType)
			}
			if getOrdersRequest.Type != orderType || getOrdersRequest.Type != order.AnyType {
				continue
			}
			var orderStatus order.Status
			switch ordersData[y].OrderState {
			case "open":
				orderStatus = order.Active
			case "filled":
				orderStatus = order.Filled
			case "rejected":
				orderStatus = order.Rejected
			case "cancelled":
				orderStatus = order.Cancelled
			case "untriggered":
				orderStatus = order.UnknownStatus
			default:
				return resp, fmt.Errorf("%v: orderStatus %s not supported", d.Name, ordersData[y].OrderState)
			}
			resp = append(resp, order.Detail{
				AssetType:       asset.Futures,
				Exchange:        d.Name,
				PostOnly:        ordersData[y].PostOnly,
				Price:           ordersData[y].Price,
				Amount:          ordersData[y].Amount,
				ExecutedAmount:  ordersData[y].FilledAmount,
				Fee:             ordersData[y].Commission,
				RemainingAmount: ordersData[y].Amount - ordersData[y].FilledAmount,
				OrderID:         ordersData[y].OrderID,
				Pair:            getOrdersRequest.Pairs[x],
				LastUpdated:     time.Unix(ordersData[y].LastUpdateTimestamp/1000, 0),
				Side:            orderSide,
				Type:            orderType,
				Status:          orderStatus,
			})
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (d *Deribit) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (d *Deribit) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := d.UpdateAccountInfo(ctx, assetType)
	return d.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (d *Deribit) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	fmtPair, err := d.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	min := strconv.Itoa(int(interval.Duration().Minutes()))
	if min == "1440" {
		min = "1D"
	}

	tradingViewData, err := d.GetTradingViewChartData(
		ctx,
		fmtPair.String(),
		min,
		start,
		end)
	if err != nil {
		return kline.Item{}, err
	}
	checkLen := len(tradingViewData.Ticks)
	if len(tradingViewData.Open) != checkLen ||
		len(tradingViewData.High) != checkLen ||
		len(tradingViewData.Low) != checkLen ||
		len(tradingViewData.Close) != checkLen ||
		len(tradingViewData.Volume) != checkLen {
		return kline.Item{}, fmt.Errorf("%s - %s - %v: invalid trading view chart data received", d.Name, a, pair)
	}
	var resp kline.Item
	for x := range tradingViewData.Ticks {
		resp.Candles = append(resp.Candles, kline.Candle{
			Time:   time.Unix(int64(tradingViewData.Ticks[x])/1000, 0),
			Open:   tradingViewData.Open[x],
			High:   tradingViewData.High[x],
			Low:    tradingViewData.Low[x],
			Close:  tradingViewData.Close[x],
			Volume: tradingViewData.Volume[x],
		})
	}
	resp.Pair = pair
	resp.Asset = a
	resp.Interval = interval
	resp.Exchange = d.Name
	return resp, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (d *Deribit) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
