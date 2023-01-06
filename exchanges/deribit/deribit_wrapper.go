package deribit

import (
	"context"
	"errors"
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
					kline.TenMin.Word():     true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.TwoHour.Word():    true,
					kline.ThreeHour.Word():  true,
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
		URL:                  deribitWebsocketAddress,
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
func (d *Deribit) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, assetType.String())
	}
	var resp []currency.Pair
	for _, x := range []string{"BTC", "SOL", "ETH", "USDC"} {
		var instrumentsData []InstrumentData
		var err error
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			instrumentsData, err = d.WSRetriveInstrumentsData(x, d.GetAssetKind(assetType), false)
		} else {
			instrumentsData, err = d.GetInstrumentsData(ctx, x, d.GetAssetKind(assetType), false)
		}
		if err != nil && len(resp) == 0 {
			return nil, err
		}
		for y := range instrumentsData {
			cp, err := currency.NewPairFromString(instrumentsData[y].InstrumentName)
			if err != nil {
				return nil, err
			}
			resp = append(resp, cp)
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
		err = d.UpdatePairs(pairs, assets[x], false, forceUpdate)
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
	if p.IsEmpty() {
		return nil, fmt.Errorf("pair provided is empty")
	}
	fmtPair, err := d.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tickerData *TickerData
	if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		tickerData, err = d.WSRetrivePublicTicker(fmtPair.String())
	} else {
		tickerData, err = d.GetPublicTicker(ctx, fmtPair.String())
	}
	if err != nil {
		return nil, err
	}
	resp := ticker.Price{
		ExchangeName: d.Name,
		Pair:         p,
		AssetType:    assetType,
		Ask:          tickerData.BestAskPrice,
		AskSize:      tickerData.BestAskAmount,
		Bid:          tickerData.BestBidPrice,
		BidSize:      tickerData.BestBidAmount,
		High:         tickerData.Stats.High,
		Low:          tickerData.Stats.Low,
		Last:         tickerData.LastPrice,
		Volume:       tickerData.Stats.Volume,
	}
	err = ticker.ProcessTicker(&resp)

	if err != nil {
		return nil, err
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
	fmtPair, err := d.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var obData *Orderbook
	if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		obData, err = d.WSRetriveOrderbookData(fmtPair.String(), 50)
	} else {
		obData, err = d.GetOrderbookData(ctx, fmtPair.String(), 50)
	}
	if err != nil {
		return nil, err
	}
	book.Asks = make([]orderbook.Item, len(obData.Asks))
	for x := range book.Asks {
		book.Asks[x] = orderbook.Item{
			Price:  obData.Asks[x][0],
			Amount: obData.Asks[x][1],
		}
		if book.Asks[x].Price == 0 {
			return nil, errors.New("asks price cannot be zero")
		}
	}
	book.Bids = make([]orderbook.Item, len(obData.Bids))
	for x := range book.Bids {
		book.Bids[x] = orderbook.Item{
			Price:  obData.Bids[x][0],
			Amount: obData.Bids[x][1],
		}
		if book.Bids[x].Price == 0 {
			return nil, errors.New("bids price cannot be zero")
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(d.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (d *Deribit) UpdateAccountInfo(ctx context.Context, _ asset.Item) (account.Holdings, error) {
	var resp account.Holdings
	resp.Exchange = d.Name
	currencies, err := d.GetCurrencies(ctx)
	if err != nil {
		return resp, err
	}
	resp.Accounts = make([]account.SubAccount, len(currencies))
	for x := range currencies {
		var data *AccountSummaryData
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			data, err = d.WSRetriveAccountSummary(currencies[x].Currency, false)
		} else {
			data, err = d.GetAccountSummary(ctx, currencies[x].Currency, false)
		}
		if err != nil {
			return resp, err
		}
		var subAcc account.SubAccount
		subAcc.Currencies = append(subAcc.Currencies, account.Balance{
			Currency: currency.NewCode(currencies[x].Currency),
			Total:    data.Balance,
			Hold:     data.Balance - data.AvailableFunds,
		})
		resp.Accounts[x] = subAcc
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

// GetFundingHistory returns funding history, deposits and withdrawals
func (d *Deribit) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	var currencies []CurrencyData
	var err error
	if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		currencies, err = d.WSRetriveCurrencies()
	} else {
		currencies, err = d.GetCurrencies(ctx)
	}
	if err != nil {
		return nil, err
	}
	var resp []exchange.FundHistory
	for x := range currencies {
		var deposits *DepositsData
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			deposits, err = d.WSRetriveDeposits(currencies[x].Currency, 100, 0)
		} else {
			deposits, err = d.GetDeposits(ctx, currencies[x].Currency, 100, 0)
		}
		if err != nil {
			return nil, err
		}
		for y := range deposits.Data {
			resp = append(resp, exchange.FundHistory{
				ExchangeName:    d.Name,
				Status:          deposits.Data[y].State,
				TransferID:      deposits.Data[y].TransactionID,
				Timestamp:       time.UnixMilli(deposits.Data[y].UpdatedTimestamp),
				Currency:        currencies[x].Currency,
				Amount:          deposits.Data[y].Amount,
				CryptoToAddress: deposits.Data[y].Address,
				TransferType:    "deposit",
			})
		}
		var withdrawalData *WithdrawalsData
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			withdrawalData, err = d.WSRetriveWithdrawals(currencies[x].Currency, 100, 0)
		} else {
			withdrawalData, err = d.GetWithdrawals(ctx, currencies[x].Currency, 100, 0)
		}
		if err != nil {
			return nil, err
		}
		for z := range withdrawalData.Data {
			resp = append(resp, exchange.FundHistory{
				ExchangeName:    d.Name,
				Status:          withdrawalData.Data[z].State,
				TransferID:      withdrawalData.Data[z].TransactionID,
				Timestamp:       time.UnixMilli(withdrawalData.Data[z].UpdatedTimestamp),
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
	var currencies []CurrencyData
	var err error
	if d.Websocket.IsConnected() {
		currencies, err = d.WSRetriveCurrencies()
	} else {
		currencies, err = d.GetCurrencies(ctx)
	}
	if err != nil {
		return nil, err
	}
	resp := []exchange.WithdrawalHistory{}
	for x := range currencies {
		if !strings.EqualFold(currencies[x].Currency, c.String()) {
			continue
		}
		var withdrawalData *WithdrawalsData
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			withdrawalData, err = d.WSRetriveWithdrawals(currencies[x].Currency, 100, 0)
		} else {
			withdrawalData, err = d.GetWithdrawals(ctx, currencies[x].Currency, 100, 0)
		}
		if err != nil {
			return nil, err
		}
		for y := range withdrawalData.Data {
			resp = append(resp, exchange.WithdrawalHistory{
				Status:          withdrawalData.Data[y].State,
				TransferID:      withdrawalData.Data[y].TransactionID,
				Timestamp:       time.UnixMilli(withdrawalData.Data[y].UpdatedTimestamp),
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
	resp := []trade.Data{}
	var trades *PublicTradesData
	if d.Websocket.IsConnected() {
		trades, err = d.WSRetriveLastTradesByInstrument(
			format.Format(p), "", "", "", 0, false)
	} else {
		trades, err = d.GetLastTradesByInstrument(
			ctx,
			format.Format(p), "", "", "", 0, false)
	}
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
			Timestamp:    time.UnixMilli(trades.Trades[a].Timestamp),
			AssetType:    assetType,
			Side:         sideData,
			CurrencyPair: p,
		})
	}
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (d *Deribit) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if common.StartEndTimeCheck(timestampStart, timestampEnd) != nil {
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
		tradesData, err = d.GetLastTradesByInstrumentAndTime(ctx, fmtPair.String(), "asc", 100, false, timestampStart, timestampEnd)
		if err != nil {
			return nil, err
		}
		if len(tradesData.Trades) != 100 {
			hasMore = false
		}
		for t := range tradesData.Trades {
			if t == 99 {
				if timestampStart.Equal(time.UnixMilli(tradesData.Trades[t].Timestamp)) {
					hasMore = false
				}
				timestampStart = time.UnixMilli(tradesData.Trades[t].Timestamp)
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
				Timestamp:    time.UnixMilli(tradesData.Trades[t].Timestamp),
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
	if !d.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%s: orderType %v is not valid", d.Name, s.AssetType)
	}
	var orderID string
	var fmtPair currency.Pair
	status := order.New
	fmtPair, err = d.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	timeInForce := ""
	if s.ImmediateOrCancel {
		timeInForce = "immediate_or_cancel"
	}
	var data *PrivateTradeData
	reqParams := &OrderBuyAndSellParams{
		Instrument:   fmtPair.String(),
		OrderType:    strings.ToLower(s.Type.String()),
		Label:        s.ClientOrderID,
		TimeInForce:  timeInForce,
		Amount:       s.Amount,
		Price:        s.Price,
		TriggerPrice: s.TriggerPrice,
		PostOnly:     s.PostOnly,
		ReduceOnly:   s.ReduceOnly,
	}
	switch {
	case s.Side.IsLong():
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			data, err = d.WSSubmitBuy(reqParams)
		} else {
			data, err = d.SubmitBuy(ctx, reqParams)
		}
		if err != nil {
			return nil, err
		}
		orderID = data.Order.OrderID
	case s.Side.IsShort():
		var data *PrivateTradeData
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			data, err = d.WSSubmitSell(reqParams)
		} else {
			data, err = d.SubmitSell(ctx, reqParams)
		}
		if err != nil {
			return nil, err
		}
		orderID = data.Order.OrderID
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
	if !d.SupportsAsset(action.AssetType) || action.AssetType == asset.Combo {
		return nil, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, action.AssetType)
	}
	var modify *PrivateTradeData
	var err error
	switch action.AssetType {
	case asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo:
		reqParam := &OrderBuyAndSellParams{
			TriggerPrice: action.TriggerPrice,
			PostOnly:     action.PostOnly,
			Amount:       action.Amount,
			OrderID:      action.OrderID,
			Price:        action.Price,
		}
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			modify, err = d.WSSubmitEdit(reqParam)
		} else {
			modify, err = d.SubmitEdit(ctx, reqParam)
		}
		if err != nil {
			return nil, err
		}
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
	err := ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo:
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			_, err = d.WSSubmitCancel(ord.OrderID)
		} else {
			_, err = d.SubmitCancel(ctx, ord.OrderID)
		}
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
			var err error
			if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				_, err = d.WSSubmitCancel(orders[x].OrderID)
			} else {
				_, err = d.SubmitCancel(ctx, orders[x].OrderID)
			}
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
	case asset.Futures, asset.Options, asset.OptionCombo, asset.FutureCombo:
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
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			cancelData, err = d.WSSubmitCancelAllByInstrument(pairFmt.Format(orderCancellation.Pair), orderTypeStr)
		} else {
			cancelData, err = d.SubmitCancelAllByInstrument(ctx, pairFmt.Format(orderCancellation.Pair), orderTypeStr)
		}
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
	if !d.SupportsAsset(assetType) {
		return resp, fmt.Errorf("%s: orderType %v is not valid", d.Name, assetType)
	}
	var orderInfo *OrderData
	var err error
	if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		orderInfo, err = d.WSRetrivesOrderState(orderID)
	} else {
		orderInfo, err = d.GetOrderState(ctx, orderID)
	}
	if err != nil {
		return resp, err
	}
	orderSide := order.Sell
	if orderInfo.Direction == sideBUY {
		orderSide = order.Buy
	}
	orderType, err := order.StringToOrderType(orderInfo.OrderType)
	if err != nil {
		return resp, err
	}
	var orderStatus order.Status
	if orderInfo.OrderState == "untriggered" {
		orderStatus = order.UnknownStatus
	} else {
		orderStatus, err = order.StringToOrderStatus(orderInfo.OrderState)
		if err != nil {
			return resp, fmt.Errorf("%v: orderStatus %s not supported", d.Name, orderInfo.OrderState)
		}
	}
	resp = order.Detail{
		AssetType:       assetType,
		Exchange:        d.Name,
		PostOnly:        orderInfo.PostOnly,
		Price:           orderInfo.Price,
		Amount:          orderInfo.Amount,
		ExecutedAmount:  orderInfo.FilledAmount,
		Fee:             orderInfo.Commission,
		RemainingAmount: orderInfo.Amount - orderInfo.FilledAmount,
		OrderID:         orderInfo.OrderID,
		Pair:            pair,
		LastUpdated:     time.UnixMilli(orderInfo.LastUpdateTimestamp),
		Side:            orderSide,
		Type:            orderType,
		Status:          orderStatus,
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (d *Deribit) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, accountID, chain string) (*deposit.Address, error) {
	var addressData *DepositAddressData
	var err error
	if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		addressData, err = d.WSRetriveCurrentDepositAddress(cryptocurrency.String())
	} else {
		addressData, err = d.GetCurrentDepositAddress(ctx, cryptocurrency.String())
	}
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: addressData.Address,
		Chain:   addressData.Currency,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (d *Deribit) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	err := withdrawRequest.Validate()
	if err != nil {
		return nil, err
	}
	var withdrawData *WithdrawData
	if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		withdrawData, err = d.WSSubmitWithdraw(withdrawRequest.Currency.String(), withdrawRequest.Crypto.Address, "", withdrawRequest.Amount)
	} else {
		withdrawData, err = d.SubmitWithdraw(ctx, withdrawRequest.Currency.String(), withdrawRequest.Crypto.Address, "", withdrawRequest.Amount)
	}
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
func (d *Deribit) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail
	switch getOrdersRequest.AssetType {
	case asset.Futures, asset.Options, asset.FutureCombo, asset.OptionCombo:
		for x := range getOrdersRequest.Pairs {
			fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
			if err != nil {
				return nil, err
			}
			var ordersData []OrderData
			if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				ordersData, err = d.WSRetriveOpenOrdersByInstrument(fmtPair.String(), getOrdersRequest.Type.Lower())
			} else {
				ordersData, err = d.GetOpenOrdersByInstrument(ctx, fmtPair.String(), getOrdersRequest.Type.Lower())
			}
			if err != nil {
				return nil, err
			}
			for y := range ordersData {
				orderSide := order.Sell
				if ordersData[y].Direction == sideBUY {
					orderSide = order.Buy
				}
				if getOrdersRequest.Side != orderSide && getOrdersRequest.Side != order.AnySide {
					continue
				}
				orderType, err := order.StringToOrderType(ordersData[y].OrderType)
				if err != nil {
					return nil, err
				}
				if getOrdersRequest.Type != orderType && getOrdersRequest.Type != order.AnyType {
					continue
				}
				var orderStatus order.Status
				ordersData[y].OrderState = strings.ToLower(ordersData[y].OrderState)
				if ordersData[y].OrderState != "open" {
					continue
				}
				resp = append(resp, order.Detail{
					AssetType:       getOrdersRequest.AssetType,
					Exchange:        d.Name,
					PostOnly:        ordersData[y].PostOnly,
					Price:           ordersData[y].Price,
					Amount:          ordersData[y].Amount,
					ExecutedAmount:  ordersData[y].FilledAmount,
					Fee:             ordersData[y].Commission,
					RemainingAmount: ordersData[y].Amount - ordersData[y].FilledAmount,
					OrderID:         ordersData[y].OrderID,
					Pair:            getOrdersRequest.Pairs[x],
					LastUpdated:     time.UnixMilli(ordersData[y].LastUpdateTimestamp),
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
func (d *Deribit) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
		if err != nil {
			return nil, err
		}
		var ordersData []OrderData
		if d.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			ordersData, err = d.WSRetriveOrderHistoryByInstrument(fmtPair.String(), 100, 0, true, true)
		} else {
			ordersData, err = d.GetOrderHistoryByInstrument(ctx, fmtPair.String(), 100, 0, true, true)
		}
		if err != nil {
			return nil, err
		}
		for y := range ordersData {
			orderSide := order.Sell
			if ordersData[y].Direction == sideBUY {
				orderSide = order.Buy
			}
			if getOrdersRequest.Side != orderSide && getOrdersRequest.Side != order.AnySide {
				continue
			}
			orderType, err := order.StringToOrderType(ordersData[y].OrderType)
			if err != nil {
				return nil, err
			}
			if getOrdersRequest.Type != orderType && getOrdersRequest.Type != order.AnyType {
				continue
			}
			var orderStatus order.Status
			if ordersData[y].OrderState == "untriggered" {
				orderStatus = order.UnknownStatus
			} else {
				orderStatus, err = order.StringToOrderStatus(ordersData[y].OrderState)
				if err != nil {
					return resp, fmt.Errorf("%v: orderStatus %s not supported", d.Name, ordersData[y].OrderState)
				}
			}
			resp = append(resp, order.Detail{
				AssetType:       getOrdersRequest.AssetType,
				Exchange:        d.Name,
				PostOnly:        ordersData[y].PostOnly,
				Price:           ordersData[y].Price,
				Amount:          ordersData[y].Amount,
				ExecutedAmount:  ordersData[y].FilledAmount,
				Fee:             ordersData[y].Commission,
				RemainingAmount: ordersData[y].Amount - ordersData[y].FilledAmount,
				OrderID:         ordersData[y].OrderID,
				Pair:            getOrdersRequest.Pairs[x],
				LastUpdated:     time.UnixMilli(ordersData[y].LastUpdateTimestamp),
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
	return 0, common.ErrFunctionNotSupported
}

// ValidateCredentials validates current credentials used for wrapper
func (d *Deribit) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := d.UpdateAccountInfo(ctx, assetType)
	return d.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (d *Deribit) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := d.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	fmtPair, err := d.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}
	intervalString, err := d.GetResolutionFromInterval(interval)
	if err != nil {
		return kline.Item{}, err
	}
	var tradingViewData *TVChartData
	if d.Websocket.IsConnected() {
		tradingViewData, err = d.WSRetrivesTradingViewChartData(fmtPair.String(), intervalString, start, end)
	} else {
		tradingViewData, err = d.GetTradingViewChartData(ctx, fmtPair.String(), intervalString, start, end)
	}
	if err != nil {
		return kline.Item{}, err
	}
	checkLen := len(tradingViewData.Ticks)
	if len(tradingViewData.Open) != checkLen ||
		len(tradingViewData.High) != checkLen ||
		len(tradingViewData.Low) != checkLen ||
		len(tradingViewData.Close) != checkLen ||
		len(tradingViewData.Volume) != checkLen {
		return kline.Item{}, fmt.Errorf("%s - %s - %v: invalid trading view chart data received", d.Name, a, fmtPair)
	}
	resp := kline.Item{
		Pair:     fmtPair,
		Asset:    a,
		Interval: interval,
		Exchange: d.Name,
	}
	resp.Candles = make([]kline.Candle, len(tradingViewData.Ticks))
	for x := range tradingViewData.Ticks {
		resp.Candles[x] = kline.Candle{
			Time:   time.UnixMilli(int64(tradingViewData.Ticks[x])),
			Open:   tradingViewData.Open[x],
			High:   tradingViewData.High[x],
			Low:    tradingViewData.Low[x],
			Close:  tradingViewData.Close[x],
			Volume: tradingViewData.Volume[x],
		}
	}
	return resp, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (d *Deribit) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := d.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	fmtPair, err := d.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}
	resp := kline.Item{
		Pair:     fmtPair,
		Asset:    a,
		Interval: interval,
		Exchange: d.Name,
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, d.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	var tradingViewData *TVChartData
	for x := range dates.Ranges {
		intervalString, err := d.GetResolutionFromInterval(interval)
		if err != nil {
			return kline.Item{}, err
		}
		if d.Websocket.IsConnected() {
			tradingViewData, err = d.WSRetrivesTradingViewChartData(fmtPair.String(), intervalString, dates.Ranges[x].Start.Time, dates.Ranges[x].End.Time)
		} else {
			tradingViewData, err = d.GetTradingViewChartData(ctx, fmtPair.String(), intervalString, dates.Ranges[x].Start.Time, dates.Ranges[x].End.Time)
		}
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
		for x := range tradingViewData.Ticks {
			resp.Candles = append(resp.Candles, kline.Candle{
				Time:   time.UnixMilli(int64(tradingViewData.Ticks[x])),
				Open:   tradingViewData.Open[x],
				High:   tradingViewData.High[x],
				Low:    tradingViewData.Low[x],
				Close:  tradingViewData.Close[x],
				Volume: tradingViewData.Volume[x],
			})
		}
	}
	return resp, nil
}
