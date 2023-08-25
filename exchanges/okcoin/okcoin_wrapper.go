package okcoin

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
func (o *Okcoin) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	o.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = o.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = o.BaseCurrencies

	err := o.Setup(exchCfg)
	if err != nil {
		return nil, err
	}
	if o.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = o.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults method assigns the default values for Okcoin
func (o *Okcoin) SetDefaults() {
	o.SetErrorDefaults()
	o.Name = okcoinExchangeName
	o.Enabled = true
	o.Verbose = true

	o.API.CredentialsValidator.RequiresKey = true
	o.API.CredentialsValidator.RequiresSecret = true
	o.API.CredentialsValidator.RequiresClientID = true
	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := o.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	o.Features = exchange.Features{
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
				CancelOrder:         true,
				CancelOrders:        true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				KlineFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageCorrelation:     true,
				GetOrders:              true,
				GetOrder:               true,
				AccountBalance:         true,
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
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.TwoDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
					kline.IntervalCapacity{Interval: kline.ThreeMonth},
				),
				GlobalResultLimit: 100,
			},
		},
	}
	o.Requester, err = request.New(o.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()),
	)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	o.API.Endpoints = o.NewEndpoints()
	err = o.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      okcoinAPIURL,
		exchange.WebsocketSpot: okcoinWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	o.Websocket = stream.New()
	o.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	o.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	o.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (o *Okcoin) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		o.SetEnabled(false)
		return nil
	}
	err = o.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsEndpoint, err := o.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = o.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             wsEndpoint,
		RunningURL:             wsEndpoint,
		RunningURLAuth:         okcoinPrivateWebsocketURL,
		Connector:              o.WsConnect,
		Subscriber:             o.Subscribe,
		Unsubscriber:           o.Unsubscribe,
		GenerateSubscriptions:  o.GenerateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &o.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}
	err = o.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            okcoinWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}
	return o.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  okcoinPrivateWebsocketURL,
		RateLimit:            okcoinWsRateLimit,
		Authenticated:        true,
	})
}

// Start starts the Okcoin go routine
func (o *Okcoin) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		o.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the Okcoin wrapper
func (o *Okcoin) Run(ctx context.Context) {
	if o.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			o.Name,
			common.IsEnabled(o.Websocket.IsEnabled()))
		o.PrintEnabledPairs()
	}

	forceUpdate := false
	var err error
	if !o.BypassConfigFormatUpgrades {
		for _, a := range o.GetAssetTypes(true) {
			var format currency.PairFormat
			format, err = o.GetPairFormat(a, false)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					o.Name,
					err)
				return
			}
			var enabled, avail currency.Pairs
			enabled, err = o.CurrencyPairs.GetPairs(a, true)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					o.Name,
					err)
				return
			}

			avail, err = o.CurrencyPairs.GetPairs(a, false)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					o.Name,
					err)
				return
			}

			if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
				!common.StringDataContains(avail.Strings(), format.Delimiter) {
				var p currency.Pairs
				p, err = currency.NewPairsFromStrings([]string{currency.BTC.String() +
					format.Delimiter +
					currency.USD.String()})
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%s failed to update currencies.\n",
						o.Name)
				} else {
					log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, o.Name, a, p)
					forceUpdate = true

					err = o.UpdatePairs(p, a, true, true)
					if err != nil {
						log.Errorf(log.ExchangeSys,
							"%s failed to update currencies. Err: %s\n",
							o.Name,
							err)
						return
					}
				}
			}
		}
	}
	if !o.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}
	err = o.UpdateTradablePairs(ctx, forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			o.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (o *Okcoin) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !o.SupportsAsset(a) {
		return nil, fmt.Errorf("%w, asset: %v", asset.ErrNotSupported, a)
	}
	prods, err := o.GetInstruments(ctx, "SPOT", "")
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(prods))
	for x := range prods {
		if prods[x].State != "live" {
			continue
		}
		var pair currency.Pair
		pair, err = currency.NewPairFromString(prods[x].InstrumentID)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (o *Okcoin) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := o.GetAssetTypes(true)
	for a := range assets {
		pairs, err := o.FetchTradablePairs(ctx, assets[a])
		if err != nil {
			return err
		}
		err = o.UpdatePairs(pairs, assets[a], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (o *Okcoin) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !o.SupportsAsset(a) {
		return fmt.Errorf("%w, asset: %v", asset.ErrNotSupported, a)
	}
	tickers, err := o.GetTickers(ctx, "SPOT")
	if err != nil {
		return err
	}
	enabledPairs, err := o.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	for i := range tickers {
		cp, err := currency.NewPairFromString(tickers[i].InstrumentID)
		if err != nil {
			return err
		}
		if !enabledPairs.Contains(cp, true) {
			continue
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tickers[i].LastTradedPrice.Float64(),
			High:         tickers[i].High24H.Float64(),
			Bid:          tickers[i].BestBidPrice.Float64(),
			BidSize:      tickers[i].BestBidSize.Float64(),
			Ask:          tickers[i].BestAskPrice.Float64(),
			AskSize:      tickers[i].BestAskPrice.Float64(),
			QuoteVolume:  tickers[i].VolCcy24H.Float64(),
			LastUpdated:  tickers[i].Timestamp.Time(),
			Volume:       tickers[i].Vol24H.Float64(),
			Open:         tickers[i].Open24H.Float64(),
			AssetType:    a,
			ExchangeName: o.Name,
			Pair:         cp,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (o *Okcoin) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	t, err := ticker.GetTicker(o.Name, p, assetType)
	if err != nil {
		return o.UpdateTicker(ctx, p, assetType)
	}
	return t, nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *Okcoin) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := o.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(o.Name, p, a)
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (o *Okcoin) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if !o.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, assetType)
	}
	var err error
	p, err = o.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData []SpotTrade
	tradeData, err = o.GetTrades(ctx, p.String(), 0)
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     o.Name,
			TID:          tradeData[i].TradeID,
			CurrencyPair: p,
			Side:         side,
			AssetType:    assetType,
			Price:        tradeData[i].TradePrice.Float64(),
			Amount:       tradeData[i].TradeSize.Float64(),
			Timestamp:    tradeData[i].Timestamp.Time(),
		}
	}
	err = o.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (o *Okcoin) CancelBatchOrders(ctx context.Context, args []order.Cancel) (*order.CancelBatchResponse, error) {
	var err error
	cancelBatchResponse := &order.CancelBatchResponse{
		Status: make(map[string]string, len(args)),
	}
	params := make([]CancelTradeOrderRequest, len(args))
	for x := range args {
		if !o.SupportsAsset(args[x].AssetType) {
			return cancelBatchResponse, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, args[x].AssetType)
		}
		err = args[x].Validate()
		if err != nil {
			return cancelBatchResponse, err
		}
		args[x].Pair, err = o.FormatExchangeCurrency(args[x].Pair, args[x].AssetType)
		if err != nil {
			return cancelBatchResponse, err
		}
		params[x] = CancelTradeOrderRequest{
			InstrumentID:  args[x].Pair.String(),
			OrderID:       args[x].OrderID,
			ClientOrderID: args[x].ClientOrderID,
		}
	}
	var responses []TradeOrderResponse
	if o.Websocket.IsConnected() && o.Websocket.CanUseAuthenticatedEndpoints() && o.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		responses, err = o.WsCancelMultipleOrders(params)
	} else {
		responses, err = o.CancelMultipleOrders(ctx, params)
	}
	if err != nil {
		return cancelBatchResponse, err
	}
	for x := range responses {
		cancelBatchResponse.Status[responses[x].OrderID] = func() string {
			if responses[x].SCode != "0" && responses[x].SCode != "2" {
				return fmt.Sprintf("error code: %s msg: %s", responses[x].SCode, responses[x].SMsg)
			}
			return order.Cancelled.String()
		}()
	}
	return cancelBatchResponse, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (o *Okcoin) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := o.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	ob, err := orderbook.Get(o.Name, fPair, assetType)
	if err != nil {
		return o.UpdateOrderbook(ctx, fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *Okcoin) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !o.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	book := &orderbook.Base{
		Exchange:        o.Name,
		Pair:            p,
		Asset:           a,
		VerifyOrderbook: o.CanVerifyOrderbook,
	}
	p, err := o.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	orderbookList, err := o.GetOrderbook(ctx, p.String(), 400)
	if err != nil {
		return nil, err
	}
	book.Bids = make(orderbook.Items, len(orderbookList.Bids))
	for x := range orderbookList.Bids {
		book.Bids[x].Amount, err = strconv.ParseFloat(orderbookList.Bids[x][1], 64)
		if err != nil {
			return nil, err
		}
		book.Bids[x].Price, err = strconv.ParseFloat(orderbookList.Bids[x][0], 64)
		if err != nil {
			return book, err
		}
	}
	book.Asks = make(orderbook.Items, len(orderbookList.Asks))
	for x := range orderbookList.Asks {
		book.Asks[x].Amount, err = strconv.ParseFloat(orderbookList.Asks[x][1], 64)
		if err != nil {
			return nil, err
		}
		book.Asks[x].Price, err = strconv.ParseFloat(orderbookList.Asks[x][0], 64)
		if err != nil {
			return book, err
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(o.Name, p, a)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (o *Okcoin) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	if !o.SupportsAsset(assetType) {
		return account.Holdings{}, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, assetType)
	}
	currencies, err := o.GetAccountBalance(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	var resp account.Holdings
	resp.Exchange = o.Name
	currencyAccount := account.SubAccount{AssetType: assetType}

	for i := range currencies {
		for x := range currencies[i].Details {
			hold := currencies[i].Details[x].FrozenBalance.Float64()
			totalValue := currencies[i].Details[x].AvailableBalance.Float64()
			currencyAccount.Currencies = append(currencyAccount.Currencies,
				account.Balance{
					Currency: currency.NewCode(currencies[i].Details[x].Currency),
					Total:    totalValue,
					Hold:     hold,
					Free:     totalValue - hold,
				})
		}
	}
	resp.Accounts = append(resp.Accounts, currencyAccount)
	creds, err := o.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&resp, creds)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (o *Okcoin) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := o.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(o.Name, creds, assetType)
	if err != nil {
		return o.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (o *Okcoin) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	accountDepositHistory, err := o.GetDepositHistory(ctx, currency.EMPTYCODE, "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	accountWithdrawlHistory, err := o.GetWithdrawalHistory(ctx, currency.EMPTYCODE, "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(accountDepositHistory)+len(accountWithdrawlHistory))
	for x := range accountDepositHistory {
		orderStatus := ""
		switch accountDepositHistory[x].State {
		case "0":
			orderStatus = "waiting for confirmation"
		case "1":
			orderStatus = "deposit credited"
		case "2":
			orderStatus = "deposit successful"
		case "8":
			orderStatus = "pending due to temporary deposit suspension "
		case "12":
			orderStatus = "account or deposit is frozen"
		case "13":
			orderStatus = "sub-account deposit interception"
		}
		resp[x] = exchange.FundingHistory{
			Amount:       accountDepositHistory[x].Amount.Float64(),
			Currency:     accountDepositHistory[x].Currency,
			ExchangeName: o.Name,
			Status:       orderStatus,
			Timestamp:    accountDepositHistory[x].Timestamp.Time(),
			TransferID:   accountDepositHistory[x].TransactionID,
			TransferType: "deposit",
		}
	}

	for i := range accountWithdrawlHistory {
		orderStatus := ""
		switch accountWithdrawlHistory[i].State {
		case "-3":
			orderStatus = "pending cancel"
		case "-2":
			orderStatus = "canceled"
		case "-1":
			orderStatus = "failed"
		case "0":
			orderStatus = "pending"
		case "1":
			orderStatus = "sending"
		case "2":
			orderStatus = "sent"
		case "3":
			orderStatus = "awaiting email verification"
		case "4":
			orderStatus = "awaiting manual verification"
		case "5":
			orderStatus = "awaiting identity verification"
		}
		resp[len(accountDepositHistory)+i] = exchange.FundingHistory{
			Amount:          accountWithdrawlHistory[i].Amount.Float64(),
			Currency:        accountWithdrawlHistory[i].Ccy,
			ExchangeName:    o.Name,
			Status:          orderStatus,
			Timestamp:       accountWithdrawlHistory[i].Timestamp.Time(),
			TransferID:      accountWithdrawlHistory[i].TransactionID,
			Fee:             accountWithdrawlHistory[i].Fee.Float64(),
			CryptoToAddress: accountWithdrawlHistory[i].ReceivingAddress,
			CryptoTxID:      accountWithdrawlHistory[i].TransactionID,
			CryptoChain:     accountWithdrawlHistory[i].Chain,
			TransferType:    "withdrawal",
		}
	}
	return resp, nil
}

// SubmitOrder submits a new order
func (o *Okcoin) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("%w, place order request parameter can not be null", common.ErrNilPointer)
	}
	if !o.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w, asset: %v", asset.ErrNotSupported, s.AssetType)
	}
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	s.Pair, err = o.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	if s.TradeMode == "" {
		s.TradeMode = "cash"
	}
	req := PlaceTradeOrderParam{
		ClientOrderID: s.ClientID,
		InstrumentID:  s.Pair,
		Side:          s.Side.Lower(),
		OrderType:     s.Type.Lower(),
		Size:          s.Amount,
		TradeMode:     s.TradeMode,
		Price:         s.Price,
		OrderTag:      "",
		BanAmend:      false,
	}
	if (s.Type == order.Limit || s.Type == order.PostOnly ||
		s.Type == order.FillOrKill || s.Type == order.ImmediateOrCancel) && s.Price <= 0 {
		return nil, fmt.Errorf("%w, price is required for order types %v,%v,%v, and %v", errInvalidPrice, order.Limit, order.PostOnly, order.ImmediateOrCancel, order.FillOrKill)
	}
	var orderResponse *TradeOrderResponse
	if o.Websocket.IsConnected() && o.Websocket.CanUseAuthenticatedEndpoints() && o.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		orderResponse, err = o.WsPlaceOrder(&req)
	} else {
		orderResponse, err = o.PlaceOrder(ctx, &req)
	}
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(orderResponse.OrderID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *Okcoin) ModifyOrder(ctx context.Context, req *order.Modify) (*order.ModifyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w, modify request parameter can not be null", common.ErrNilPointer)
	}
	if !o.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w, asset: %v", asset.ErrNotSupported, req.AssetType)
	}
	var err error
	req.Pair, err = o.FormatExchangeCurrency(req.Pair, req.AssetType)
	if err != nil {
		return nil, err
	}
	err = req.Validate()
	if err != nil {
		return nil, err
	}
	amendRequest := &AmendTradeOrderRequestParam{
		OrderID:       req.OrderID,
		InstrumentID:  req.Pair.String(),
		ClientOrderID: req.ClientOrderID,
		NewSize:       req.Amount,
		NewPrice:      req.Price}
	if o.Websocket.IsConnected() && o.Websocket.CanUseAuthenticatedEndpoints() && o.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = o.WsAmendOrder(amendRequest)
	} else {
		_, err = o.AmendOrder(ctx, amendRequest)
	}
	if err != nil {
		return nil, err
	}
	return req.DeriveModifyResponse()
}

// CancelOrder cancels an order by its corresponding ID number
func (o *Okcoin) CancelOrder(ctx context.Context, cancel *order.Cancel) error {
	err := cancel.Validate(cancel.StandardCancel())
	if err != nil {
		return err
	}
	cancel.Pair, err = o.FormatExchangeCurrency(cancel.Pair, cancel.AssetType)
	if err != nil {
		return err
	}
	amendRequest := &CancelTradeOrderRequest{
		InstrumentID:  cancel.Pair.String(),
		OrderID:       cancel.OrderID,
		ClientOrderID: cancel.ClientOrderID,
	}
	if o.Websocket.IsConnected() && o.Websocket.CanUseAuthenticatedEndpoints() && o.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = o.WsCancelTradeOrder(amendRequest)
	} else {
		_, err = o.CancelTradeOrder(ctx, amendRequest)
	}
	if err != nil {
		return err
	}
	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (o *Okcoin) CancelAllOrders(_ context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrFunctionNotSupported
}

// GetOrderInfo returns order information based on order ID
func (o *Okcoin) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if !o.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	if err := o.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	pair, err := o.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	tradeOrder, err := o.GetPersonalOrderDetail(ctx, pair.String(), orderID, "")
	if err != nil {
		return nil, err
	}
	status, err := order.StringToOrderStatus(tradeOrder.State)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
	}

	side, err := order.StringToOrderSide(tradeOrder.Side)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
	}
	orderType, err := order.StringToOrderType(tradeOrder.OrderType)
	if err != nil {
		return nil, err
	}
	return &order.Detail{
		Amount:               tradeOrder.Size.Float64(),
		Pair:                 pair,
		Exchange:             o.Name,
		Date:                 tradeOrder.CreationTime.Time(),
		LastUpdated:          tradeOrder.UpdateTime.Time(),
		ExecutedAmount:       tradeOrder.AccFillSize.Float64(),
		Status:               status,
		Side:                 side,
		Leverage:             tradeOrder.Leverage.Float64(),
		ReduceOnly:           tradeOrder.ReduceOnly,
		Price:                tradeOrder.Price.Float64(),
		AverageExecutedPrice: tradeOrder.AveragePrice.Float64(),
		RemainingAmount:      tradeOrder.Size.Float64() - tradeOrder.AccFillSize.Float64(),
		Fee:                  tradeOrder.Fee.Float64(),
		FeeAsset:             currency.NewCode(tradeOrder.FeeCurrency),
		OrderID:              tradeOrder.OrderID,
		ClientOrderID:        tradeOrder.ClientOrdID,
		Type:                 orderType,
		AssetType:            assetType,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *Okcoin) GetDepositAddress(ctx context.Context, c currency.Code, _, _ string) (*deposit.Address, error) {
	wallet, err := o.GetCurrencyDepositAddresses(ctx, c)
	if err != nil {
		return nil, err
	}
	if len(wallet) == 0 {
		return nil, fmt.Errorf("%w for currency %s",
			errNoAccountDepositAddress,
			c)
	}
	return &deposit.Address{
		Address: wallet[0].Address,
		Tag:     wallet[0].Tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *Okcoin) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	err := withdrawRequest.Validate()
	if err != nil {
		return nil, err
	}
	param := &WithdrawalRequest{
		Amount:         withdrawRequest.Amount,
		Ccy:            withdrawRequest.Currency,
		Chain:          withdrawRequest.Crypto.Chain,
		ToAddress:      withdrawRequest.Crypto.Address,
		TransactionFee: withdrawRequest.Crypto.FeeAmount,
	}
	if withdrawRequest.InternalTransfer {
		param.WithdrawalMethod = "3"
	} else {
		param.WithdrawalMethod = "4"
	}
	if param.TransactionFee == 0 {
		param.TransactionFee, err = o.GetFee(ctx, &exchange.FeeBuilder{
			FeeType: exchange.CryptocurrencyWithdrawalFee,
			Amount:  param.Amount,
		})
		if err != nil {
			return nil, err
		}
	}
	withdrawal, err := o.Withdrawal(ctx, param)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: withdrawal[0].WdID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *Okcoin) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *Okcoin) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (o *Okcoin) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	if c.IsFiatCurrency() {
		return nil, fmt.Errorf("%w for fiat currencies %v", common.ErrFunctionNotSupported, c)
	}
	withdrawals, err := o.GetWithdrawalHistory(ctx, c, "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	wHistories := make([]exchange.WithdrawalHistory, len(withdrawals))
	for x := range withdrawals {
		orderStatus := ""
		switch withdrawals[x].State {
		case "-3":
			orderStatus = "pending cancel"
		case "-2":
			orderStatus = "canceled"
		case "-1":
			orderStatus = "failed"
		case "0":
			orderStatus = "pending"
		case "1":
			orderStatus = "sending"
		case "2":
			orderStatus = "sent"
		case "3":
			orderStatus = "awaiting email verification"
		case "4":
			orderStatus = "awaiting manual verification"
		case "5":
			orderStatus = "awaiting identity verification"
		}
		wHistories[x] = exchange.WithdrawalHistory{
			Status:          orderStatus,
			TransferID:      withdrawals[x].WithdrawalID,
			Timestamp:       withdrawals[x].Timestamp.Time(),
			Currency:        withdrawals[x].Ccy,
			Amount:          withdrawals[x].Amount.Float64(),
			Fee:             withdrawals[x].Fee.Float64(),
			CryptoToAddress: withdrawals[x].ReceivingAddress,
			CryptoTxID:      withdrawals[x].TransactionID,
			CryptoChain:     withdrawals[x].Chain,
			TransferType:    "withdrawal",
		}
	}
	return wHistories, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (o *Okcoin) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range req.Pairs {
		req.Pairs[x], err = o.FormatExchangeCurrency(req.Pairs[x], req.AssetType)
		if err != nil {
			return nil, err
		}
		var tradeOrders []TradeOrder
		tradeOrders, err = o.GetPersonalOrderList(ctx, "SPOT", req.Pairs[x].String(), "", "", req.StartTime, req.EndTime, 0)
		if err != nil {
			return nil, err
		}
		for i := range tradeOrders {
			var status order.Status
			status, err = order.StringToOrderStatus(tradeOrders[i].State)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeOrders[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(tradeOrders[i].OrderType)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			resp = append(resp, order.Detail{
				Date:           tradeOrders[i].CreationTime.Time(),
				LastUpdated:    tradeOrders[i].UpdateTime.Time(),
				CloseTime:      tradeOrders[i].FillTime.Time(),
				Price:          tradeOrders[i].AveragePrice.Float64(),
				ExecutedAmount: tradeOrders[i].AccFillSize.Float64(),
				OrderID:        tradeOrders[i].OrderID,
				Amount:         tradeOrders[i].Size.Float64(),
				Pair:           req.Pairs[x],
				Type:           orderType,
				Status:         status,
				Exchange:       o.Name,
				Side:           side,
			})
		}
	}
	return req.Filter(o.Name, resp), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (o *Okcoin) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range req.Pairs {
		req.Pairs[x], err = o.FormatExchangeCurrency(req.Pairs[x], req.AssetType)
		if err != nil {
			return nil, err
		}
		var spotOrders []TradeOrder
		spotOrders, err = o.GetOrderHistory3Months(ctx, "SPOT", req.Pairs[x].String(), "", "", req.StartTime, req.EndTime, 0)
		if err != nil {
			return nil, err
		}
		for i := range spotOrders {
			var status order.Status
			status, err = order.StringToOrderStatus(spotOrders[i].State)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var side order.Side
			side, err = order.StringToOrderSide(spotOrders[i].Side)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			var orderType order.Type
			orderType, err = order.StringToOrderType(spotOrders[i].OrderType)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", o.Name, err)
			}
			detail := order.Detail{
				OrderID:              spotOrders[i].OrderID,
				Price:                spotOrders[i].Price.Float64(),
				AverageExecutedPrice: spotOrders[i].AveragePrice.Float64(),
				Amount:               spotOrders[i].Size.Float64(),
				ExecutedAmount:       spotOrders[i].AccFillSize.Float64(),
				RemainingAmount:      spotOrders[i].Size.Float64() - spotOrders[i].AccFillSize.Float64(),
				Pair:                 req.Pairs[x],
				Exchange:             o.Name,
				Side:                 side,
				Type:                 orderType,
				Date:                 spotOrders[i].CreationTime.Time(),
				LastUpdated:          spotOrders[i].UpdateTime.Time(),
				CloseTime:            spotOrders[i].FillTime.Time(),
				Status:               status,
			}
			detail.InferCostsAndTimes()
			resp = append(resp, detail)
		}
	}
	return req.Filter(o.Name, resp), nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *Okcoin) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !o.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return o.GetFee(ctx, feeBuilder)
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (o *Okcoin) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := o.UpdateAccountInfo(ctx, assetType)
	return o.CheckTransientError(err)
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (o *Okcoin) GetHistoricTrades(ctx context.Context, pair currency.Pair, assetType asset.Item, start, end time.Time) ([]trade.Data, error) {
	if !o.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, assetType)
	}
	pair, err := o.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	if !pair.IsPopulated() {
		return nil, fmt.Errorf("%w, %v", currency.ErrCurrencyPairEmpty, assetType)
	}
	var resp []trade.Data
	tradeIDEnd := ""
allTrades:
	for {
		var trades []SpotTrade
		trades, err = o.GetTradeHistory(ctx, pair.String(), "2", start, end, 100)
		if err != nil {
			return nil, err
		}
		if len(trades) == 0 {
			break
		}
		for i := 0; i < len(trades); i++ {
			if start.Equal(trades[i].Timestamp.Time()) ||
				trades[i].Timestamp.Time().Before(start) ||
				tradeIDEnd == trades[len(trades)-1].TradeID {
				// reached end of trades to crawl
				break allTrades
			}
			var tradeSide order.Side
			tradeSide, err = order.StringToOrderSide(trades[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          trades[i].TradeID,
				Exchange:     o.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Price:        trades[i].TradePrice.Float64(),
				Amount:       trades[i].TradeSize.Float64(),
				Timestamp:    trades[i].Timestamp.Time(),
				Side:         tradeSide,
			})
		}
		tradeIDEnd = trades[len(trades)-1].TradeID
	}
	if o.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(o.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, start, end), nil
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (o *Okcoin) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !o.SupportsAsset(a) {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, a)
	}
	req, err := o.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}
	pair, err = o.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	timeSeries, err := o.GetCandlesticks(ctx, pair.String(), interval, req.End, req.Start, 300, true)
	if err != nil {
		return nil, err
	}
	timeSeriesData := make([]kline.Candle, len(timeSeries))
	for x := range timeSeries {
		timeSeriesData[x] = kline.Candle{
			Time:   timeSeries[x].Timestamp.Time(),
			Open:   timeSeries[x].OpenPrice,
			High:   timeSeries[x].HighestPrice,
			Low:    timeSeries[x].LowestPrice,
			Close:  timeSeries[x].ClosePrice,
			Volume: timeSeries[x].TradingVolume,
		}
	}
	return req.ProcessResponse(timeSeriesData)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (o *Okcoin) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if !o.SupportsAsset(a) {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, a)
	}
	req, err := o.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		for a := range req.RangeHolder.Ranges[x].Intervals {
			var candles []CandlestickData
			candles, err = o.GetCandlestickHistory(ctx,
				req.RequestFormatted.String(),
				req.RangeHolder.Ranges[x].Intervals[a].Start.Time,
				req.RangeHolder.Ranges[x].Intervals[a].End.Time,
				interval, 0)
			if err != nil {
				return nil, err
			}
			for z := range candles {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[z].Timestamp.Time(),
					Open:   candles[z].OpenPrice,
					High:   candles[z].HighestPrice,
					Low:    candles[z].LowestPrice,
					Close:  candles[z].ClosePrice,
					Volume: candles[z].TradingVolume,
				})
			}
		}
	}
	return req.ProcessResponse(timeSeries)
}

// ValidateAPICredentials validates current credentials used for wrapper
func (o *Okcoin) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := o.UpdateAccountInfo(ctx, assetType)
	return o.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (o *Okcoin) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return o.GetSystemTime(ctx)
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (o *Okcoin) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !o.SupportsAsset(a) {
		return asset.ErrNotSupported
	}
	instrumentsList, err := o.GetInstruments(ctx, "SPOT", "")
	if err != nil {
		return fmt.Errorf("%s failed to load %s pair execution limits. Err: %s", o.Name, a, err)
	}

	limits := make([]order.MinMaxLevel, 0, len(instrumentsList))
	for index := range instrumentsList {
		pair, err := currency.NewPairFromString(instrumentsList[index].InstrumentID)
		if err != nil {
			return err
		}
		limits = append(limits, order.MinMaxLevel{
			Asset:                  a,
			Pair:                   pair,
			PriceStepIncrementSize: instrumentsList[index].TickSize.Float64(),
			MinimumBaseAmount:      instrumentsList[index].MinSize.Float64(),
			MaxIcebergParts:        instrumentsList[index].MaxIcebergSz.Int64(),
			MarketMaxQty:           instrumentsList[index].MaxMarketSize.Float64(),
		})
	}
	if err := o.LoadLimits(limits); err != nil {
		return fmt.Errorf("%s Error loading %s exchange limits: %v", o.Name, a, err)
	}
	return nil
}
