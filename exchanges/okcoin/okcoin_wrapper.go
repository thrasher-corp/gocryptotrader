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
func (o *OKCoin) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
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

// SetDefaults method assigns the default values for OKCoin
func (o *OKCoin) SetDefaults() {
	o.SetErrorDefaults()
	o.Name = okCoinExchangeName
	o.Enabled = true
	o.Verbose = true

	o.API.CredentialsValidator.RequiresKey = true
	o.API.CredentialsValidator.RequiresSecret = true
	o.API.CredentialsValidator.RequiresClientID = true
	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	err := o.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Margin)
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
				GlobalResultLimit: 1440,
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
		exchange.RestSpot:      okCoinAPIURL,
		exchange.WebsocketSpot: okCoinWebsocketURL,
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
func (o *OKCoin) Setup(exch *config.Exchange) error {
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
		RunningURLAuth:         okCoinPrivateWebsocketURL,
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
		URL:                  okCoinWebsocketURL,
	})
	if err != nil {
		return err
	}
	return o.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            okcoinWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
		URL:                  okCoinPrivateWebsocketURL,
	})
}

// Start starts the OKCoin go routine
func (o *OKCoin) Start(ctx context.Context, wg *sync.WaitGroup) error {
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

// Run implements the OKCoin wrapper
func (o *OKCoin) Run(ctx context.Context) {
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
		var format currency.PairFormat
		format, err = o.GetPairFormat(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				o.Name,
				err)
			return
		}
		var enabled, avail currency.Pairs
		enabled, err = o.CurrencyPairs.GetPairs(asset.Spot, true)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				o.Name,
				err)
			return
		}

		avail, err = o.CurrencyPairs.GetPairs(asset.Spot, false)
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
				log.Warnf(log.ExchangeSys, exchange.ResetConfigPairsWarningMessage, o.Name, asset.Spot, p)
				forceUpdate = true

				err = o.UpdatePairs(p, asset.Spot, true, true)
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
func (o *OKCoin) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w, asset: %v", asset.ErrNotSupported, a)
	}
	prods, err := o.GetInstruments(ctx, "SPOT", "")
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, len(prods))
	for x := range prods {
		var pair currency.Pair
		pair, err = currency.NewPairFromString(prods[x].InstrumentID)
		if err != nil {
			return nil, err
		}
		pairs[x] = pair
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (o *OKCoin) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := o.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	return o.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (o *OKCoin) UpdateTickers(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%w, asset: %v", asset.ErrNotSupported, a)
	}
	tickers, err := o.GetTickers(ctx, "SPOT")
	if err != nil {
		return err
	}
	enabledPairs, err := o.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	for p := range enabledPairs {
		for i := range tickers {
			cp, err := currency.NewPairFromString(tickers[i].InstrumentID)
			if err != nil {
				return err
			}
			if !enabledPairs[p].Equal(cp) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         tickers[i].LastTradedPrice,
				High:         tickers[i].High24H,
				Bid:          tickers[i].BestBidPrice,
				BidSize:      tickers[i].BestBidSize,
				Ask:          tickers[i].BestAskPrice,
				AskSize:      tickers[i].BestAskPrice,
				QuoteVolume:  tickers[i].VolCcy24H,
				LastUpdated:  tickers[i].Timestamp.Time(),
				Volume:       tickers[i].Vol24H,
				Open:         tickers[i].Open24H,
				AssetType:    asset.Spot,
				ExchangeName: o.Name,
				Pair:         cp,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKCoin) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := o.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(o.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (o *OKCoin) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerData, err := ticker.GetTicker(o.Name, p, assetType)
	if err != nil {
		return o.UpdateTicker(ctx, p, assetType)
	}
	return tickerData, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (o *OKCoin) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if assetType != asset.Spot {
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
			Price:        tradeData[i].TradePrice,
			Amount:       tradeData[i].TradeSize,
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
func (o *OKCoin) CancelBatchOrders(ctx context.Context, args []order.Cancel) (order.CancelBatchResponse, error) {
	var err error
	cancelBatchResponse := order.CancelBatchResponse{
		Status: make(map[string]string, len(args)),
	}
	params := make([]CancelTradeOrderRequest, len(args))
	for x := range args {
		if args[x].AssetType != asset.Spot {
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
				return ""
			}
			return order.Cancelled.String()
		}()
	}
	return cancelBatchResponse, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (o *OKCoin) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
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
func (o *OKCoin) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Base, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, a)
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
	orderbookLite, err := o.GetOrderbook(ctx, p.String(), 0)
	if err != nil {
		return nil, err
	}
	book.Bids = make(orderbook.Items, len(orderbookLite.Bids))
	for x := range orderbookLite.Bids {
		book.Bids[x].Amount, err = strconv.ParseFloat(orderbookLite.Bids[x][1], 64)
		if err != nil {
			return nil, err
		}
		book.Bids[x].Price, err = strconv.ParseFloat(orderbookLite.Bids[x][0], 64)
		if err != nil {
			return book, err
		}
	}
	book.Asks = make(orderbook.Items, len(orderbookLite.Asks))
	for x := range orderbookLite.Bids {
		book.Asks[x].Amount, err = strconv.ParseFloat(orderbookLite.Asks[x][1], 64)
		if err != nil {
			return nil, err
		}
		book.Asks[x].Price, err = strconv.ParseFloat(orderbookLite.Asks[x][0], 64)
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
func (o *OKCoin) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	if assetType != asset.Spot {
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
			hold, parseErr := strconv.ParseFloat(currencies[i].Details[x].FrozenBalance, 64)
			if parseErr != nil {
				return resp, parseErr
			}
			totalValue, parseErr := strconv.ParseFloat(currencies[i].Details[x].AvailableBalance, 64)
			if parseErr != nil {
				return resp, parseErr
			}
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
func (o *OKCoin) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
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

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *OKCoin) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	accountDepositHistory, err := o.GetDepositHistory(ctx, currency.EMPTYCODE, "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	accountWithdrawlHistory, err := o.GetWithdrawalHistory(ctx, currency.EMPTYCODE, "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundHistory, len(accountDepositHistory)+len(accountWithdrawlHistory))
	for x := range accountDepositHistory {
		orderStatus := ""
		switch accountDepositHistory[x].State {
		case 0:
			orderStatus = "waiting for confirmation"
		case 1:
			orderStatus = "deposit credited"
		case 2:
			orderStatus = "deposit successful"
		case 8:
			orderStatus = "pending due to temporary deposit suspension "
		case 12:
			orderStatus = "account or deposit is frozen"
		case 13:
			orderStatus = "sub-account deposit interception"
		}
		resp[x] = exchange.FundHistory{
			Amount:       accountDepositHistory[x].Amount,
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
		case -3:
			orderStatus = "pending cancel"
		case -2:
			orderStatus = "canceled"
		case -1:
			orderStatus = "failed"
		case 0:
			orderStatus = "pending"
		case 1:
			orderStatus = "sending"
		case 2:
			orderStatus = "sent"
		case 3:
			orderStatus = "awaiting email verification"
		case 4:
			orderStatus = "awaiting manual verification"
		case 5:
			orderStatus = "awaiting identity verification"
		}
		resp[len(accountDepositHistory)+i] = exchange.FundHistory{
			Amount:          accountWithdrawlHistory[i].Amt,
			Currency:        accountWithdrawlHistory[i].Ccy,
			ExchangeName:    o.Name,
			Status:          orderStatus,
			Timestamp:       accountWithdrawlHistory[i].Timestamp.Time(),
			TransferID:      accountWithdrawlHistory[i].TransactionID,
			Fee:             accountWithdrawlHistory[i].Fee,
			CryptoToAddress: accountWithdrawlHistory[i].To,
			CryptoTxID:      accountWithdrawlHistory[i].TransactionID,
			CryptoChain:     accountWithdrawlHistory[i].Chain,
			TransferType:    "withdrawal",
		}
	}
	return resp, nil
}

// SubmitOrder submits a new order
func (o *OKCoin) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s.AssetType != asset.Spot {
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
func (o *OKCoin) ModifyOrder(ctx context.Context, req *order.Modify) (*order.ModifyResponse, error) {
	if req.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w, asset: %v", asset.ErrNotSupported, req.AssetType)
	}
	var err error
	req.Pair, err = o.FormatExchangeCurrency(req.Pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	err = req.Validate()
	if err != nil {
		return nil, err
	}
	request := &AmendTradeOrderRequestParam{
		OrderID:       req.OrderID,
		InstrumentID:  req.Pair.String(),
		ClientOrderID: req.ClientOrderID,
		NewSize:       req.Amount,
		NewPrice:      req.Price}
	if o.Websocket.IsConnected() && o.Websocket.CanUseAuthenticatedEndpoints() && o.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = o.WsAmendOrder(request)
	} else {
		_, err = o.AmendOrder(ctx, request)
	}
	if err != nil {
		return nil, err
	}
	return req.DeriveModifyResponse()
}

// CancelOrder cancels an order by its corresponding ID number
func (o *OKCoin) CancelOrder(ctx context.Context, cancel *order.Cancel) error {
	err := cancel.Validate(cancel.StandardCancel())
	if err != nil {
		return err
	}
	cancel.Pair, err = o.FormatExchangeCurrency(cancel.Pair, cancel.AssetType)
	if err != nil {
		return err
	}
	request := &CancelTradeOrderRequest{
		InstrumentID:  cancel.Pair.String(),
		OrderID:       cancel.OrderID,
		ClientOrderID: cancel.ClientOrderID,
	}
	if o.Websocket.IsConnected() && o.Websocket.CanUseAuthenticatedEndpoints() && o.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = o.WsCancelTradeOrder(request)
	} else {
		_, err = o.CancelTradeOrder(ctx, request)
	}
	if err != nil {
		return err
	}
	return nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (o *OKCoin) CancelAllOrders(_ context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrFunctionNotSupported
}

// GetOrderInfo returns order information based on order ID
func (o *OKCoin) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var resp order.Detail
	if assetType != asset.Spot {
		return resp, fmt.Errorf("%s %w", assetType, asset.ErrNotSupported)
	}
	pair, err := o.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return resp, err
	}
	tradeOrder, err := o.GetPersonalOrderDetail(ctx, pair.String(), orderID, "")
	if err != nil {
		return resp, err
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
		return resp, err
	}
	return order.Detail{
		Amount:               tradeOrder.Size,
		Pair:                 pair,
		Exchange:             o.Name,
		Date:                 tradeOrder.CreationTime.Time(),
		LastUpdated:          tradeOrder.UpdateTime.Time(),
		ExecutedAmount:       tradeOrder.AccFillSize,
		Status:               status,
		Side:                 side,
		Leverage:             tradeOrder.Leverage,
		ReduceOnly:           tradeOrder.ReduceOnly,
		Price:                tradeOrder.Price,
		AverageExecutedPrice: tradeOrder.AveragePrice,
		RemainingAmount:      tradeOrder.Size - tradeOrder.AccFillSize,
		Fee:                  tradeOrder.Fee,
		FeeAsset:             currency.NewCode(tradeOrder.FeeCurrency),
		OrderID:              tradeOrder.OrderID,
		ClientOrderID:        tradeOrder.ClientOrdID,
		Type:                 orderType,
		AssetType:            assetType,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *OKCoin) GetDepositAddress(ctx context.Context, c currency.Code, _, _ string) (*deposit.Address, error) {
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
func (o *OKCoin) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	param := &WithdrawalRequest{
		Amount:         withdrawRequest.Amount,
		Ccy:            withdrawRequest.Currency,
		Chain:          withdrawRequest.Crypto.Chain,
		ToAddress:      withdrawRequest.Crypto.Address,
		TransactionFee: withdrawRequest.Crypto.FeeAmount,
	}
	if withdrawRequest.Crypto.Chain != "" {
		param.WithdrawalMethod = "4"
	} else {
		param.WithdrawalMethod = "3"
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
func (o *OKCoin) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKCoin) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (o *OKCoin) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w, asseet: %v", asset.ErrNotSupported, a)
	}
	withdrawals, err := o.GetWithdrawalHistory(ctx, c, "", "", "", "", "", time.Time{}, time.Time{}, 0)
	if err != nil {
		return nil, err
	}
	wHistories := make([]exchange.WithdrawalHistory, len(withdrawals))
	for x := range withdrawals {
		orderStatus := ""
		switch withdrawals[x].State {
		case -3:
			orderStatus = "pending cancel"
		case -2:
			orderStatus = "canceled"
		case -1:
			orderStatus = "failed"
		case 0:
			orderStatus = "pending"
		case 1:
			orderStatus = "sending"
		case 2:
			orderStatus = "sent"
		case 3:
			orderStatus = "awaiting email verification"
		case 4:
			orderStatus = "awaiting manual verification"
		case 5:
			orderStatus = "awaiting identity verification"
		}
		wHistories[x] = exchange.WithdrawalHistory{
			Status:          orderStatus,
			TransferID:      withdrawals[x].WithdrawalID,
			Timestamp:       withdrawals[x].Timestamp.Time(),
			Currency:        withdrawals[x].Ccy,
			Amount:          withdrawals[x].Amt,
			Fee:             withdrawals[x].Fee,
			CryptoToAddress: withdrawals[x].To,
			CryptoTxID:      withdrawals[x].TransactionID,
			CryptoChain:     withdrawals[x].Chain,
			TransferType:    "withdrawal",
		}
	}
	return wHistories, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (o *OKCoin) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range req.Pairs {
		req.Pairs[x], err = o.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
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
				Price:          tradeOrders[i].AveragePrice,
				ExecutedAmount: tradeOrders[i].AccFillSize,
				OrderID:        tradeOrders[i].OrderID,
				Amount:         tradeOrders[i].Size,
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
func (o *OKCoin) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var resp []order.Detail
	for x := range req.Pairs {
		req.Pairs[x], err = o.FormatExchangeCurrency(req.Pairs[x], asset.Spot)
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
				Price:                spotOrders[i].Price,
				AverageExecutedPrice: spotOrders[i].AveragePrice,
				Amount:               spotOrders[i].Size,
				ExecutedAmount:       spotOrders[i].AccFillSize,
				RemainingAmount:      spotOrders[i].Size - spotOrders[i].AccFillSize,
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
func (o *OKCoin) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
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
func (o *OKCoin) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := o.UpdateAccountInfo(ctx, assetType)
	return o.CheckTransientError(err)
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (o *OKCoin) GetHistoricTrades(ctx context.Context, pair currency.Pair, assetType asset.Item, start, end time.Time) ([]trade.Data, error) {
	if assetType != asset.Spot {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, assetType)
	}
	const limit = 100
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
				Price:        trades[i].TradePrice,
				Amount:       trades[i].TradeSize,
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
func (o *OKCoin) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, a)
	}
	req, err := o.GetKlineRequest(pair, a, interval, start, end, true)
	if err != nil {
		return nil, err
	}
	pair, err = o.FormatExchangeCurrency(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	timeSeries, err := o.GetCandlesticks(ctx, pair.String(), interval, req.End, req.Start, 300)
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
func (o *OKCoin) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if a != asset.Spot {
		return nil, fmt.Errorf("%w, asset type %v", asset.ErrNotSupported, a)
	}
	req, err := o.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles []CandlestickData
		candles, err = o.GetCandlestickHistory(ctx, req.RequestFormatted.String(), req.RangeHolder.Ranges[x].End.Time, req.RangeHolder.Ranges[x].Start.Time, interval, 0)
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
	return req.ProcessResponse(timeSeries)
}

// ValidateAPICredentials validates current credentials used for wrapper
func (o *OKCoin) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := o.UpdateAccountInfo(ctx, assetType)
	return o.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (o *OKCoin) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return o.GetSystemTime(ctx)
}
