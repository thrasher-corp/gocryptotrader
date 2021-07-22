package deribit

import (
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
func (d *Deribit) GetDefaultConfig() (*config.ExchangeConfig, error) {
	d.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = d.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = d.BaseCurrencies

	d.SetupDefaults(exchCfg)

	if d.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := d.UpdateTradablePairs(true)
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

	// If using only one pair format for request and configuration, across all
	// supported asset types either SPOT and FUTURES etc. You can use the
	// example below:

	// Request format denotes what the pair as a string will be, when you send
	// a request to an exchange.
	requestFmt := &currency.PairFormat{ /*Set pair request formatting details here for e.g.*/ Uppercase: true, Delimiter: ":"}
	// Config format denotes what the pair as a string will be, when saved to
	// the config.json file.
	configFmt := &currency.PairFormat{ /*Set pair request formatting details here*/ }
	err := d.SetGlobalPairsManager(requestFmt, configFmt /*multiple assets can be set here using the asset package ie asset.Spot*/)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// If assets require multiple differences in formating for request and
	// configuration, another exchange method can be be used e.g. futures
	// contracts require a dash as a delimiter rather than an underscore. You
	// can use this example below:

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{Uppercase: true},
	}

	err = d.StoreAssetPairFormat(asset.Futures, fmt1)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	d.Features = exchange.Features{
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
		},
	}
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	d.Requester = request.New(d.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))

	// NOTE: SET THE URLs HERE
	d.API.Endpoints = d.NewEndpoints()
	d.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestFutures: deribitAPIURL,
		// exchange.WebsocketSpot: deribitWSAPIURL,
	})
	d.Websocket = stream.New()
	d.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	d.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	d.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (d *Deribit) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		d.SetEnabled(false)
		return nil
	}

	d.SetupDefaults(exch)

	/*
		wsRunningEndpoint, err := d.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			return err
		}

		// If websocket is supported, please fill out the following

		err = d.Websocket.Setup(
			&stream.WebsocketSetup{
				Enabled:                          exch.Features.Enabled.Websocket,
				Verbose:                          exch.Verbose,
				AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
				WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
				DefaultURL:                       deribitWSAPIURL,
				ExchangeName:                     exch.Name,
				RunningURL:                       wsRunningEndpoint,
				Connector:                        d.WsConnect,
				Subscriber:                       d.Subscribe,
				UnSubscriber:                     d.Unsubscribe,
				Features:                         &d.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		d.WebsocketConn = &stream.WebsocketConnection{
			ExchangeName:         d.Name,
			URL:                  d.Websocket.GetWebsocketURL(),
			ProxyURL:             d.Websocket.GetProxyAddress(),
			Verbose:              d.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}

		// NOTE: PLEASE ENSURE YOU SET THE ORDERBOOK BUFFER SETTINGS CORRECTLY
		d.Websocket.Orderbook.Setup(
			exch.OrderbookConfig.WebsocketBufferLimit,
			true,
			true,
			false,
			false,
			exch.Name)
	*/
	return nil
}

// Start starts the Deribit go routine
func (d *Deribit) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		d.Run()
		wg.Done()
	}()
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

	err := d.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			d.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (d *Deribit) FetchTradablePairs(assetType asset.Item) ([]string, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, d.Name)
	}
	format, err := d.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	currs, err := d.GetCurrencies()
	if err != nil {
		return nil, err
	}
	var resp []string
	for x := range currs {
		instrumentsData, err := d.GetInstrumentsData(currs[x].Currency, "", false)
		if err != nil {
			return nil, err
		}
		for y := range instrumentsData {
			curr, err := currency.NewPairFromString(instrumentsData[y].InstrumentName)
			if err != nil {
				return nil, err
			}
			resp = append(resp, format.Format(curr))
		}
	}
	return resp, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (d *Deribit) UpdateTradablePairs(forceUpdate bool) error {
	assets := d.GetAssetTypes()
	fmt.Println(assets)
	for x := range assets {
		pairs, err := d.FetchTradablePairs(assets[x])
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

// UpdateTicker updates and returns the ticker for a currency pair
func (d *Deribit) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, assetType)
	}

	switch assetType {
	case asset.Futures:
		if p.IsEmpty() {
			return nil, fmt.Errorf("pair provided is empty")
		}
		fmtPair, err := d.FormatExchangeCurrency(p, asset.Futures)
		if err != nil {
			return nil, err
		}
		tickerData, err := d.GetPublicTicker(fmtPair.String())
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
func (d *Deribit) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(d.Name, p, assetType)
	if err != nil {
		return d.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (d *Deribit) FetchOrderbook(currency currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(d.Name, currency, assetType)
	if err != nil {
		return d.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (d *Deribit) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
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

		obData, err := d.GetOrderbookData(fmtPair.String(), 50)
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
func (d *Deribit) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	var resp account.Holdings
	resp.Exchange = d.Name
	switch assetType {
	case asset.Futures:
		currencies, err := d.GetCurrencies()
		if err != nil {
			return resp, err
		}
		for x := range currencies {
			data, err := d.GetAccountSummary(currencies[x].Currency, false)
			if err != nil {
				return resp, err
			}

			var subAcc account.SubAccount
			subAcc.AssetType = asset.Futures
			subAcc.Currencies = append(subAcc.Currencies, account.Balance{
				CurrencyName: currency.NewCode(currencies[x].Currency),
				TotalValue:   data.Balance,
				Hold:         data.Balance - data.AvailableFunds,
			})
		}
	default:
		return resp, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, assetType)
	}
	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (d *Deribit) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	accountData, err := account.GetHoldings(d.Name, assetType)
	if err != nil {
		return d.UpdateAccountInfo(assetType)
	}
	return accountData, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (d *Deribit) GetFundingHistory() ([]exchange.FundHistory, error) {
	currencies, err := d.GetCurrencies()
	if err != nil {
		return nil, err
	}
	var resp []exchange.FundHistory
	for x := range currencies {
		deposits, err := d.GetDeposits(currencies[x].Currency, 100, 0)
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
		withdrawalData, err := d.GetWithdrawals(currencies[x].Currency, 100, 0)
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
func (d *Deribit) GetWithdrawalsHistory(c currency.Code) ([]exchange.WithdrawalHistory, error) {
	currencies, err := d.GetCurrencies()
	if err != nil {
		return nil, err
	}
	var resp []exchange.WithdrawalHistory
	for x := range currencies {
		if !strings.EqualFold(currencies[x].Currency, c.String()) {
			continue
		}
		withdrawalData, err := d.GetWithdrawals(currencies[x].Currency, 100, 0)
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
func (d *Deribit) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if !d.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s: %w - %s", d.Name, asset.ErrNotSupported, d.Name)
	}
	format, err := d.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	currs, err := d.GetCurrencies()
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for x := range currs {
		instrumentsData, err := d.GetInstrumentsData(currs[x].Currency, "", false)
		if err != nil {
			return nil, err
		}
		for y := range instrumentsData {
			if strings.EqualFold(format.Format(p), instrumentsData[y].InstrumentName) {
				trades, err := d.GetLastTradesByInstrument(
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
					if trades.Trades[a].Direction == "buy" {
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
func (d *Deribit) GetHistoricTrades(p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
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
	var tradesData PublicTradesData
	var hasMore = true
	for hasMore {
		tradesData, err = d.GetLastTradesByInstrumentAndTime(fmtPair.String(),
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
			if tradesData.Trades[t].Direction == "buy" {
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
func (d *Deribit) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	switch s.AssetType {
	case asset.Futures:
		fmtPair, err := d.FormatExchangeCurrency(s.Pair, asset.Futures)
		if err != nil {
			return submitOrderResponse, err
		}
		switch s.Side {
		case order.Bid, order.Buy:
			data, err := d.SubmitBuy(fmtPair.String(),
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
				return submitOrderResponse, err
			}
			submitOrderResponse.Cost = data.Order.AveragePrice * data.Order.FilledAmount
			submitOrderResponse.Rate = data.Order.AveragePrice
			var feeTotal float64
			submitOrderResponse.FullyMatched = data.Order.Amount == data.Order.FilledAmount
			submitOrderResponse.OrderID = data.Order.OrderID
			for t := range data.Trades {
				typeData := order.Market
				if data.Trades[t].OrderType == "limit" {
					typeData = order.Limit
				}
				feeTotal += data.Trades[t].Fee
				submitOrderResponse.Trades = append(submitOrderResponse.Trades, order.TradeHistory{
					Price:     data.Trades[t].Price,
					Amount:    data.Trades[t].Amount,
					Fee:       data.Trades[t].Fee,
					Exchange:  d.Name,
					Side:      order.Buy,
					Type:      typeData,
					FeeAsset:  data.Trades[t].FeeCurrency,
					Timestamp: time.Unix(data.Trades[t].Timestamp/1000, 0),
					TID:       data.Trades[t].OrderID,
				})
			}
		case order.Sell, order.Ask:
			data, err := d.SubmitSell(fmtPair.String(),
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
				return submitOrderResponse, err
			}
			submitOrderResponse.Cost = data.Order.AveragePrice * data.Order.FilledAmount
			submitOrderResponse.Rate = data.Order.AveragePrice
			var feeTotal float64
			submitOrderResponse.FullyMatched = data.Order.Amount == data.Order.FilledAmount
			submitOrderResponse.OrderID = data.Order.OrderID
			for t := range data.Trades {
				typeData := order.Market
				if data.Trades[t].OrderType == "limit" {
					typeData = order.Limit
				}
				feeTotal += data.Trades[t].Fee
				submitOrderResponse.Trades = append(submitOrderResponse.Trades, order.TradeHistory{
					Price:     data.Trades[t].Price,
					Amount:    data.Trades[t].Amount,
					Fee:       data.Trades[t].Fee,
					Exchange:  d.Name,
					Side:      order.Sell,
					Type:      typeData,
					FeeAsset:  data.Trades[t].FeeCurrency,
					Timestamp: time.Unix(data.Trades[t].Timestamp/1000, 0),
					TID:       data.Trades[t].OrderID,
				})
			}
		}
	default:
		return submitOrderResponse, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, s.AssetType)
	}
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (d *Deribit) ModifyOrder(action *order.Modify) (string, error) {
	if err := action.Validate(); err != nil {
		return "", err
	}
	var modify PrivateTradeData
	var err error
	switch action.AssetType {
	case asset.Futures:
		modify, err = d.SubmitEdit(action.ID,
			"",
			action.Amount,
			action.Price,
			action.TriggerPrice,
			action.PostOnly,
			false,
			false,
			false)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, action.AssetType)
	}
	return modify.Order.OrderID, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (d *Deribit) CancelOrder(ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Futures:
		_, err := d.SubmitCancel(ord.ID)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, ord.AssetType)
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (d *Deribit) CancelBatchOrders(orders []order.Cancel) (order.CancelBatchResponse, error) {
	var resp = order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	for x := range orders {
		if orders[x].AssetType.IsValid() {
			_, err := d.SubmitCancel(orders[x].ID)
			if err != nil {
				resp.Status[orders[x].ID] = err.Error()
			} else {
				resp.Status[orders[x].ID] = "successfully cancelled"
			}
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (d *Deribit) CancelAllOrders(orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
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
		cancelData, err = d.SubmitCancelAllByInstrument(pairFmt.Format(orderCancellation.Pair), orderTypeStr)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
	default:
		return order.CancelAllResponse{}, fmt.Errorf("%s: %w - %v", d.Name, asset.ErrNotSupported, orderCancellation.AssetType)
	}
	return order.CancelAllResponse{Count: cancelData}, nil
}

// GetOrderInfo returns order information based on order ID
func (d *Deribit) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail
	switch assetType {
	case asset.Futures:
		orderInfo, err := d.GetOrderState(orderID)
		if err != nil {
			return resp, err
		}
		orderSide := order.Sell
		if orderInfo.Direction == "buy" {
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
			ID:              orderInfo.OrderID,
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
func (d *Deribit) GetDepositAddress(cryptocurrency currency.Code, accountID string) (string, error) {
	addressData, err := d.GetCurrentDepositAddress(cryptocurrency.String())
	return addressData.Address, err
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (d *Deribit) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawData, err := d.SubmitWithdraw(
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.Address,
		"",
		strconv.FormatInt(withdrawRequest.OneTimePassword, 10),
		withdrawRequest.Amount)
	return &withdraw.ExchangeResponse{
		ID:     strconv.FormatInt(withdrawData.ID, 10),
		Status: withdrawData.State,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (d *Deribit) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (d *Deribit) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (d *Deribit) GetActiveOrders(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail
	switch getOrdersRequest.AssetType {
	case asset.Futures:
		for x := range getOrdersRequest.Pairs {
			fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Futures)
			if err != nil {
				return nil, err
			}
			ordersData, err := d.GetOpenOrdersByInstrument(fmtPair.String(), getOrdersRequest.Type.Lower())
			if err != nil {
				return nil, err
			}
			for y := range ordersData {
				orderSide := order.Sell
				if ordersData[y].Direction == "buy" {
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
					ID:              ordersData[y].OrderID,
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
func (d *Deribit) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range getOrdersRequest.Pairs {
		fmtPair, err := d.FormatExchangeCurrency(getOrdersRequest.Pairs[x], asset.Futures)
		if err != nil {
			return nil, err
		}
		ordersData, err := d.GetOrderHistoryByInstrument(fmtPair.String(), 100, 0, true, true)
		if err != nil {
			return nil, err
		}
		for y := range ordersData {
			orderSide := order.Sell
			if ordersData[y].Direction == "buy" {
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
				ID:              ordersData[y].OrderID,
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
func (d *Deribit) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateCredentials validates current credentials used for wrapper
func (d *Deribit) ValidateCredentials(assetType asset.Item) error {
	_, err := d.UpdateAccountInfo(assetType)
	return d.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (d *Deribit) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	fmtPair, err := d.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}
	tradingViewData, err := d.GetTradingViewChartData(fmtPair.String(),
		interval.String(),
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
func (d *Deribit) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
