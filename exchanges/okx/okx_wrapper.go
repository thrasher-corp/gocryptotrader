package okx

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

const (
	okxWebsocketResponseMaxLimit = time.Second * 15
)

// GetDefaultConfig returns a default exchange config
func (ok *Okx) GetDefaultConfig() (*config.Exchange, error) {
	ok.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = ok.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = ok.BaseCurrencies

	err := ok.Setup(exchCfg)
	if err != nil {
		return nil, err
	}

	if ok.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := ok.UpdateTradablePairs(context.TODO(), false)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Okx
func (ok *Okx) SetDefaults() {
	ok.Name = "Okx"
	ok.Enabled = true
	ok.Verbose = true

	ok.WsResponseMultiplexer = wsRequestDataChannelsMultiplexer{
		WsResponseChannelsMap: make(map[string]*wsRequestInfo),
		Register:              make(chan *wsRequestInfo),
		Unregister:            make(chan string),
		Message:               make(chan *wsIncomingData),
	}
	ok.WsRequestSemaphore = make(chan int, 5)
	ok.API.CredentialsValidator.RequiresKey = true
	ok.API.CredentialsValidator.RequiresSecret = true
	ok.API.CredentialsValidator.RequiresClientID = true

	fmt1 := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}

	err := ok.SetGlobalPairsManager(fmt1.RequestFormat, fmt1.ConfigFormat, asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Option, asset.Margin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	ok.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:      true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				CryptoDeposit:       true,
				CryptoWithdrawalFee: true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				SubmitOrder:         true,
				CancelOrder:         true,
				CancelOrders:        true,
				TradeFetching:       true,
				UserTradeHistory:    true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrders:              true,
				TradeFetching:          true,
				KlineFetching:          true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
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
					kline.ThreeDay.Word():   true,
					kline.OneWeek.Word():    true,
					kline.OneMonth.Word():   true,
					kline.ThreeMonth.Word(): true,
					kline.SixMonth.Word():   true,
					kline.OneYear.Word():    true,
				},
				ResultLimit: 300,
			},
		},
	}
	ok.Requester, err = request.New(ok.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	ok.API.Endpoints = ok.NewEndpoints()
	err = ok.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      okxAPIURL,
		exchange.WebsocketSpot: okxAPIWebsocketPublicURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	ok.Websocket = stream.New()
	ok.WebsocketResponseMaxLimit = okxWebsocketResponseMaxLimit
	ok.WebsocketResponseCheckTimeout = okxWebsocketResponseMaxLimit
	ok.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (ok *Okx) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		ok.SetEnabled(false)
		return nil
	}
	err = ok.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := ok.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = ok.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            okxAPIWebsocketPublicURL,
		RunningURL:            wsRunningEndpoint,
		Connector:             ok.WsConnect,
		Subscriber:            ok.Subscribe,
		Unsubscriber:          ok.Unsubscribe,
		GenerateSubscriptions: ok.GenerateDefaultSubscriptions,
		Features:              &ok.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			Checksum: ok.CalculateUpdateOrderbookChecksum,
		},
	})

	if err != nil {
		return err
	}
	err = ok.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  okxAPIWebsocketPublicURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     okxWebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}
	return ok.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  okxAPIWebsocketPrivateURL,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     okxWebsocketResponseMaxLimit,
		Authenticated:        true,
	})
}

// Start starts the Okx go routine
func (ok *Okx) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		ok.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Okx wrapper
func (ok *Okx) Run() {
	if ok.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			ok.Name,
			common.IsEnabled(ok.Websocket.IsEnabled()))
		ok.PrintEnabledPairs()
	}

	if !ok.GetEnabledFeatures().AutoPairUpdates {
		return
	}
	err := ok.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			ok.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (ok *Okx) FetchTradablePairs(ctx context.Context, a asset.Item) ([]string, error) {
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", a, ok.Name)
	}
	format, err := ok.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	pairs := []string{}
	insts := []Instrument{}
	switch a {
	case asset.Spot:
		insts, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeSpot,
		})
	case asset.Futures:
		insts, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeFutures,
		})
	case asset.PerpetualSwap:
		insts, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeSwap,
		})
	case asset.Option:
		var instsb []Instrument
		var instsc []Instrument
		insts, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeOption,
			Underlying:     "BTC-USD",
		})
		if err != nil {
			return nil, err
		}
		instsb, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeOption,
			Underlying:     "ETH-USD",
		})
		if err != nil {
			return nil, err
		}
		instsc, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeOption,
			Underlying:     "SOL-USD",
		})
		insts = append(insts, instsb...)
		insts = append(insts, instsc...)
	case asset.Margin:
		insts, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeMargin,
		})
	}
	if err != nil {
		return nil, err
	}
	if len(insts) == 0 {
		return nil, errNoInstrumentFound
	}
	for x := range insts {
		var pair string
		switch insts[x].InstrumentType {
		case asset.Spot, asset.Margin:
			pair = insts[x].BaseCurrency + format.Delimiter + insts[x].QuoteCurrency
			if pair == "-" {
				return nil, fmt.Errorf("%v, invalid currency pair data", errMalformedData)
			}
		case asset.Futures, asset.PerpetualSwap, asset.Option:
			c, err := currency.NewPairFromString(insts[x].InstrumentID)
			if err != nil {
				return nil, err
			}
			pair = c.Base.String() + format.Delimiter + c.Quote.String()
		}
		pairs = append(pairs, pair)
	}
	selectedPairs := []string{}
	pairsMap := map[string]int{}
	for i := range pairs {
		if pairs[i] == "" {
			return nil, errInvalidCurrencyPair
		}
		count, ok := pairsMap[pairs[i]]
		if !ok || count == 0 {
			pairsMap[pairs[i]] = 1
			selectedPairs = append(selectedPairs, pairs[i])
		}
	}
	return selectedPairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (ok *Okx) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := ok.GetAssetTypes(false)
	for i := range assetTypes {
		p, err := ok.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return err
		}
		pairs, err := currency.NewPairsFromStrings(p)
		if err != nil {
			return err
		}
		err = ok.UpdatePairs(pairs, assetTypes[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (ok *Okx) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	format, err := ok.GetPairFormat(a, false)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID := format.Format(p)
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}
	mdata, err := ok.GetTicker(ctx, instrumentID)
	if err != nil {
		return nil, err
	}
	var baseVolume float64
	var quoteVolume float64
	switch a {
	case asset.Spot, asset.Margin:
		baseVolume = mdata.Vol24H
		quoteVolume = mdata.VolCcy24H
	case asset.PerpetualSwap, asset.Futures, asset.Option:
		baseVolume = mdata.VolCcy24H
		quoteVolume = mdata.Vol24H
	default:
		return nil, fmt.Errorf("%w, asset type %s is not supported", errInvalidInstrumentType, a.String())
	}
	err = ticker.ProcessTicker(&ticker.Price{
		Last:         mdata.LastTradePrice,
		High:         mdata.High24H,
		Low:          mdata.Low24H,
		Bid:          mdata.BidPrice,
		Ask:          mdata.BestAskPrice,
		Volume:       baseVolume,
		QuoteVolume:  quoteVolume,
		Open:         mdata.Open24H,
		Pair:         p,
		ExchangeName: ok.Name,
		AssetType:    a,
	})
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(ok.Name, p, a)
}

// UpdateTickers updates all currency pairs of a given asset type
func (ok *Okx) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	pairs, err := ok.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	instrumentType := ok.GetInstrumentTypeFromAssetItem(assetType)
	ticks, err := ok.GetTickers(ctx, strings.ToUpper(instrumentType), "", "")
	if err != nil {
		return err
	}
	format, err := ok.GetPairFormat(assetType, false)
	if err != nil {
		return err
	}
	for i := range pairs {
		pairFmt, err := ok.FormatExchangeCurrency(pairs[i], assetType)
		if err != nil {
			return err
		}
		pairFmt, err = ok.GetPairFromInstrumentID(format.Format(pairFmt))
		if err != nil {
			return err
		}
		for y := range ticks {
			pair, err := ok.GetPairFromInstrumentID(ticks[y].InstrumentID)
			if err != nil {
				return err
			}
			if pair.String() != pairFmt.String() {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks[y].LastTradePrice,
				High:         ticks[y].High24H,
				Low:          ticks[y].Low24H,
				Bid:          ticks[y].BidPrice,
				Ask:          ticks[y].BestAskPrice,
				Volume:       ticks[y].Vol24H,
				QuoteVolume:  ticks[y].VolCcy24H,
				Open:         ticks[y].Open24H,
				Pair:         pair,
				ExchangeName: ok.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (ok *Okx) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	formattedPair, err := ok.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tickerNew, err := ticker.GetTicker(ok.Name, formattedPair, assetType)
	if err != nil {
		return ok.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (ok *Okx) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(ok.Name, pair, assetType)
	if err != nil {
		return ok.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (ok *Okx) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        ok.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: ok.CanVerifyOrderbook,
	}
	var orderbookNew *OrderBookResponse
	var err error
	if !ok.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	var instrumentID string
	format, err := ok.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !pair.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID = format.Format(pair)
	orderbookNew, err = ok.GetOrderBookDepth(ctx, instrumentID, 0)
	if err != nil {
		return book, err
	}

	orderBookD, err := orderbookNew.GetOrderBookResponseDetail()
	if err != nil {
		return nil, err
	}
	book.Bids = make(orderbook.Items, len(orderBookD.Bids))
	for x := range orderBookD.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderBookD.Bids[x].BaseCurrencies,
			Price:  orderBookD.Bids[x].DepthPrice,
		}
	}
	book.Asks = make(orderbook.Items, len(orderBookD.Asks))
	for x := range orderBookD.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: orderBookD.Asks[x].NumberOfContracts,
			Price:  orderBookD.Asks[x].DepthPrice,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(ok.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies.
func (ok *Okx) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = ok.Name
	if !ok.SupportsAsset(assetType) {
		return info, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	balances, err := ok.GetBalance(ctx, "")
	if err != nil {
		return info, err
	}
	currencyBalance := make([]account.Balance, len(balances))
	for i := range balances {
		free := balances[i].AvailBal
		locked := balances[i].FrozenBalance
		currencyBalance[i] = account.Balance{
			CurrencyName: currency.NewCode(balances[i].Currency),
			Total:        balances[i].Balance,
			Hold:         locked,
			Free:         free,
		}
	}
	acc.Currencies = currencyBalance

	acc.AssetType = assetType
	info.Accounts = append(info.Accounts, acc)
	creds, err := ok.GetCredentials(ctx)
	if err != nil {
		return info, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}
	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (ok *Okx) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := ok.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(ok.Name, creds, assetType)
	if err != nil {
		return ok.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and withdrawals
func (ok *Okx) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	depositHistories, err := ok.GetCurrencyDepositHistory(ctx, "", "", "", time.Time{}, time.Time{}, -1, 0)
	if err != nil {
		return nil, err
	}
	withdrawalHistories, err := ok.GetWithdrawalHistory(ctx, "", "", "", "", time.Time{}, time.Time{}, -5, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundHistory, 0, len(depositHistories)+len(withdrawalHistories))
	for x := range depositHistories {
		resp = append(resp, exchange.FundHistory{
			ExchangeName:    ok.Name,
			Status:          strconv.Itoa(depositHistories[x].State),
			Timestamp:       depositHistories[x].Timestamp,
			Currency:        depositHistories[x].Currency,
			Amount:          depositHistories[x].Amount,
			TransferType:    "deposit",
			CryptoToAddress: depositHistories[x].ToDepositAddress,
			CryptoTxID:      depositHistories[x].TransactionID,
		})
	}
	for x := range withdrawalHistories {
		resp = append(resp, exchange.FundHistory{
			ExchangeName:    ok.Name,
			Status:          withdrawalHistories[x].StateOfWithdrawal,
			Timestamp:       withdrawalHistories[x].Timestamp,
			Currency:        withdrawalHistories[x].Currency,
			Amount:          withdrawalHistories[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalHistories[x].ToReceivingAddress,
			CryptoTxID:      withdrawalHistories[x].TransactionID,
			TransferID:      withdrawalHistories[x].WithdrawalID,
			Fee:             withdrawalHistories[x].WithdrawalFee,
			CryptoChain:     withdrawalHistories[x].ChainName,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (ok *Okx) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	withdrawals, err := ok.GetWithdrawalHistory(ctx, c.String(), "", "", "", time.Time{}, time.Time{}, -5, 0)
	if err != nil {
		return nil, err
	}
	resp = make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for x := range withdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          withdrawals[x].StateOfWithdrawal,
			Timestamp:       withdrawals[x].Timestamp,
			Currency:        withdrawals[x].Currency,
			Amount:          withdrawals[x].Amount,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[x].ToReceivingAddress,
			CryptoTxID:      withdrawals[x].TransactionID,
			CryptoChain:     withdrawals[x].ChainName,
			TransferID:      withdrawals[x].WithdrawalID,
			Fee:             withdrawals[x].WithdrawalFee,
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (ok *Okx) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	format, err := ok.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID := format.Format(p)
	tradeData, err := ok.GetTrades(ctx, instrumentID, 1000)
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for x := range tradeData {
		side, err := order.StringToOrderSide(tradeData[x].Side)
		if err != nil {
			return nil, err
		}
		resp[x] = trade.Data{
			TID:          tradeData[x].TradeID,
			Exchange:     ok.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[x].Price,
			Amount:       tradeData[x].Quantity,
			Timestamp:    tradeData[x].Timestamp,
		}
	}
	if ok.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(ok.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (ok *Okx) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	const limit = 1000
	format, err := ok.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !p.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID := format.Format(p)
	tradeData, err := ok.GetTradesHistory(ctx, instrumentID, "", "", limit)
	if err != nil {
		return nil, err
	}
	resp := make([]trade.Data, len(tradeData))
	for x := range tradeData {
		resp[x] = trade.Data{
			TID:          tradeData[x].TradeID,
			Exchange:     ok.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Price:        tradeData[x].Price,
			Amount:       tradeData[x].Quantity,
			Timestamp:    tradeData[x].Timestamp,
		}
	}
	if ok.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(ok.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (ok *Okx) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if !ok.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
	if s.Amount <= 0 {
		return nil, fmt.Errorf("amount, or size (sz) of quantity to buy or sell hast to be greater than zero ")
	}
	format, err := ok.GetPairFormat(s.AssetType, false)
	if err != nil {
		return nil, err
	}
	if !s.Pair.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID := format.Format(s.Pair)
	var tradeMode string
	if s.AssetType != asset.Margin {
		tradeMode = "cash"
	}
	var sideType string
	if s.Side == order.Buy {
		sideType = order.Buy.Lower()
	} else {
		sideType = order.Sell.Lower()
	}
	var orderRequest = &PlaceOrderRequestParam{
		InstrumentID:          instrumentID,
		TradeMode:             tradeMode,
		Side:                  sideType,
		OrderType:             s.Type.Lower(),
		Amount:                s.Amount,
		ClientSupplierOrderID: s.ClientOrderID,
		Price:                 s.Price,
	}
	switch s.Type.Lower() {
	case OkxOrderLimit, OkxOrderPostOnly, OkxOrderFOK, OkxOrderIOC:
		orderRequest.Price = s.Price
	}
	var placeOrderResponse *OrderData
	switch s.AssetType {
	case asset.Spot, asset.Option, asset.Margin:
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			placeOrderResponse, err = ok.WsPlaceOrder(orderRequest)
		} else {
			placeOrderResponse, err = ok.PlaceOrder(ctx, orderRequest, s.AssetType)
		}
	case asset.PerpetualSwap, asset.Futures:
		if s.Type.Lower() == "" {
			orderRequest.OrderType = OkxOrderOptimalLimitIOC // only applicable for Futures and Perpetual Swap Types.
		}
		orderRequest.PositionSide = positionSideLong
		if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			placeOrderResponse, err = ok.WsPlaceOrder(orderRequest)
		} else {
			placeOrderResponse, err = ok.PlaceOrder(ctx, orderRequest, s.AssetType)
		}
	default:
		return nil, errInvalidInstrumentType
	}
	if err != nil {
		return nil, err
	}
	if placeOrderResponse.OrderID != "0" && placeOrderResponse.OrderID != "" {
		submitOrderResponse.OrderID = placeOrderResponse.OrderID
	}
	return &submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (ok *Okx) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var amendRequest AmendOrderRequestParams
	var err error
	if math.Mod(action.Amount, 1) != 0 {
		return nil, errors.New("Okx contract amount can not be decimal")
	}
	format, err := ok.GetPairFormat(action.AssetType, false)
	if err != nil {
		return nil, err
	}
	if !action.Pair.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID := format.Format(action.Pair)
	if err != nil {
		return nil, err
	}
	amendRequest.InstrumentID = instrumentID
	amendRequest.NewQuantity = action.Amount
	amendRequest.OrderID = action.OrderID
	amendRequest.ClientSuppliedOrderID = action.ClientOrderID
	var response *OrderData
	if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		response, err = ok.WsAmendOrder(&amendRequest)
	} else {
		response, err = ok.AmendOrder(ctx, &amendRequest)
	}
	if err != nil {
		return nil, err
	}
	return &order.ModifyResponse{
		Exchange:  action.Exchange,
		AssetType: action.AssetType,
		Pair:      action.Pair,
		OrderID:   response.OrderID,
		Price:     action.Price,
		Amount:    amendRequest.NewQuantity,
	}, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (ok *Okx) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	if !ok.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	format, err := ok.GetPairFormat(ord.AssetType, false)
	if err != nil {
		return err
	}
	if !ord.Pair.IsPopulated() {
		return errIncompleteCurrencyPair
	}
	instrumentID := format.Format(ord.Pair)
	req := CancelOrderRequestParam{
		InstrumentID:          instrumentID,
		OrderID:               ord.OrderID,
		ClientSupplierOrderID: ord.ClientOrderID,
	}
	if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = ok.WsCancelOrder(req)
	} else {
		_, err = ok.CancelSingleOrder(ctx, req)
	}
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (ok *Okx) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	var cancelBatchResponse order.CancelBatchResponse
	cancelOrderParams := []CancelOrderRequestParam{}
	var err error
	for x := range orders {
		ord := orders[x]
		err = ord.Validate(ord.StandardCancel())
		if err != nil {
			return cancelBatchResponse, err
		}
		if !ok.SupportsAsset(ord.AssetType) {
			return cancelBatchResponse, fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
		}
		var instrumentID string
		var format currency.PairFormat
		format, err = ok.GetPairFormat(ord.AssetType, false)
		if err != nil {
			return cancelBatchResponse, err
		}
		if !ord.Pair.IsPopulated() {
			return cancelBatchResponse, errIncompleteCurrencyPair
		}
		instrumentID = format.Format(ord.Pair)
		if err != nil {
			return cancelBatchResponse, err
		}
		req := CancelOrderRequestParam{
			InstrumentID:          instrumentID,
			OrderID:               ord.OrderID,
			ClientSupplierOrderID: ord.ClientOrderID,
		}
		cancelOrderParams = append(cancelOrderParams, req)
	}
	var canceledOrders []OrderData
	if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		canceledOrders, err = ok.WsCancelMultipleOrder(cancelOrderParams)
	} else {
		canceledOrders, err = ok.CancelMultipleOrders(ctx, cancelOrderParams)
	}
	if err != nil {
		return cancelBatchResponse, err
	}
	for x := range canceledOrders {
		cancelBatchResponse.Status[canceledOrders[x].OrderID] = func() string {
			if canceledOrders[x].SCode != "0" && canceledOrders[x].SCode != "2" {
				return ""
			}
			return order.Cancelled.String()
		}()
	}
	return cancelBatchResponse, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (ok *Okx) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}
	myOrders, err := ok.GetOrderList(ctx, &OrderListRequestParams{})
	if err != nil {
		return cancelAllResponse, err
	}
	cancelAllOrdersRequestParams := make([]CancelOrderRequestParam, len(myOrders))
	for x := range myOrders {
		cancelAllOrdersRequestParams[x] = CancelOrderRequestParam{
			OrderID:               myOrders[x].OrderID,
			ClientSupplierOrderID: myOrders[x].ClientSupplierOrderID,
		}
	}
	remaining := cancelAllOrdersRequestParams
	loop := int(math.Ceil(float64(len(remaining)) / 20.0))
	for b := 0; b < loop; b++ {
		var response []OrderData
		if len(remaining) > 20 {
			if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				response, err = ok.WsCancelMultipleOrder(remaining[:20])
			} else {
				response, err = ok.CancelMultipleOrders(ctx, remaining[:20])
			}
			remaining = remaining[20:]
		} else {
			if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				response, err = ok.WsCancelMultipleOrder(remaining)
			} else {
				response, err = ok.CancelMultipleOrders(ctx, remaining)
			}
		}
		if err != nil {
			if len(cancelAllResponse.Status) == 0 {
				return cancelAllResponse, err
			}
		}
		for y := range response {
			if response[y].SCode == "0" {
				cancelAllResponse.Status[response[y].OrderID] = order.Cancelled.String()
			} else {
				cancelAllResponse.Status[response[y].OrderID] = response[y].SMessage
			}
		}
	}
	return cancelAllResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (ok *Okx) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var respData order.Detail
	format, err := ok.GetPairFormat(assetType, false)
	if err != nil {
		return respData, err
	}
	if !pair.IsPopulated() {
		return respData, errIncompleteCurrencyPair
	}
	instrumentID := format.Format(pair)
	if !ok.SupportsAsset(assetType) {
		return respData, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	orderDetail, err := ok.GetOrderDetail(ctx, &OrderDetailRequestParam{
		InstrumentID: instrumentID,
		OrderID:      orderID,
	})
	if err != nil {
		return respData, err
	}
	status, err := order.StringToOrderStatus(orderDetail.State)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
	}
	orderType, err := ok.OrderTypeFromString(orderDetail.OrderType)
	if err != nil {
		return respData, err
	}
	return order.Detail{
		Amount:         orderDetail.Size,
		Exchange:       ok.Name,
		OrderID:        orderDetail.OrderID,
		ClientOrderID:  orderDetail.ClientSupplierOrderID,
		Side:           orderDetail.Side,
		Type:           orderType,
		Pair:           pair,
		Cost:           orderDetail.Price,
		AssetType:      assetType,
		Status:         status,
		Price:          orderDetail.Price,
		ExecutedAmount: orderDetail.RebateAmount,
		Date:           orderDetail.CreationTime,
		LastUpdated:    orderDetail.UpdateTime,
	}, err
}

// GetDepositAddress returns a deposit address for a specified currency
func (ok *Okx) GetDepositAddress(ctx context.Context, c currency.Code, accountID, chain string) (*deposit.Address, error) {
	response, err := ok.GetCurrencyDepositAddress(ctx, c.String())
	if err != nil {
		return nil, err
	}

	for x := range response {
		if accountID == response[x].Address && (strings.EqualFold(response[x].Chain, chain) || strings.HasPrefix(response[x].Chain, c.String()+"-"+chain)) {
			return &deposit.Address{
				Address: response[x].Address,
				Tag:     response[x].Tag,
				Chain:   response[x].Chain,
			}, nil
		}
	}
	if len(response) > 0 {
		return &deposit.Address{
			Address: response[0].Address,
			Tag:     response[0].Tag,
			Chain:   response[0].Chain,
		}, nil
	}
	return nil, errDepositAddressNotFound
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (ok *Okx) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	var input WithdrawalInput
	input.ChainName = withdrawRequest.Crypto.Chain
	input.Amount = withdrawRequest.Amount
	input.Currency = withdrawRequest.Currency.String()
	input.ToAddress = withdrawRequest.Crypto.Address
	input.TransactionFee = withdrawRequest.Crypto.FeeAmount
	input.WithdrawalDestination = "3"
	resp, err := ok.Withdrawal(ctx, &input)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: resp.WithdrawalID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (ok *Okx) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (ok *Okx) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (ok *Okx) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if !ok.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}
	instrumentType := ok.GetInstrumentTypeFromAssetItem(req.AssetType)
	orderType, _ := ok.OrderTypeString(req.Type)
	requestParam := &OrderListRequestParams{
		OrderType:      orderType,
		After:          req.StartTime,
		Before:         req.EndTime,
		InstrumentType: instrumentType,
	}
	response, err := ok.GetOrderList(ctx, requestParam)
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, len(response))
	for x := range response {
		orderSide := response[x].Side
		var pair currency.Pair
		pair, err = ok.GetPairFromInstrumentID(response[x].InstrumentID)
		if err != nil {
			return nil, err
		}
		for i := range req.Pairs {
			if !req.Pairs[i].Equal(pair) {
				continue
			}
			orderStatus, err := order.StringToOrderStatus(strings.ToUpper(response[x].State))
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
			}
			var oType order.Type
			oType, err = ok.OrderTypeFromString(response[x].OrderType)
			if err != nil {
				return nil, err
			}
			orders[x] = order.Detail{
				Amount:          response[x].Size,
				Pair:            pair,
				Price:           response[x].Price,
				ExecutedAmount:  response[x].FillSize,
				RemainingAmount: response[x].Size - response[x].FillSize,
				Fee:             response[i].TransactionFee,
				FeeAsset:        currency.NewCode(response[x].FeeCurrency),
				Exchange:        ok.Name,
				OrderID:         response[x].OrderID,
				ClientOrderID:   response[x].ClientSupplierOrderID,
				Type:            oType,
				Side:            orderSide,
				Status:          orderStatus,
				AssetType:       req.AssetType,
				Date:            response[x].CreationTime,
				LastUpdated:     response[x].UpdateTime,
			}
		}
	}
	order.FilterOrdersByPairs(&orders, req.Pairs)
	order.FilterOrdersByType(&orders, req.Type)
	order.FilterOrdersBySide(&orders, req.Side)
	return orders, order.FilterOrdersByTimeRange(&orders, req.StartTime, req.EndTime)
}

// GetOrderHistory retrieves account order information Can Limit response to specific order status
func (ok *Okx) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, errMissingAtLeast1CurrencyPair
	}
	if !ok.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}
	instrumentType := strings.ToUpper(ok.GetInstrumentTypeFromAssetItem(req.AssetType))
	response, err := ok.Get3MonthOrderHistory(ctx, &OrderHistoryRequestParams{
		OrderListRequestParams: OrderListRequestParams{
			InstrumentType: instrumentType,
		},
	})
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(response))
	for i := range response {
		var pair currency.Pair
		pair, err = ok.GetPairFromInstrumentID(response[i].InstrumentID)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
			continue
		}
		for j := range req.Pairs {
			if !req.Pairs[j].Equal(pair) {
				continue
			}
			var orderStatus order.Status
			orderStatus, err = order.StringToOrderStatus(strings.ToUpper(response[i].State))
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
			}
			if orderStatus == order.New {
				continue
			}
			orderSide := response[i].Side
			var oType order.Type
			oType, err = ok.OrderTypeFromString(response[i].OrderType)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Price:           response[i].Price,
				Amount:          response[i].Size,
				ExecutedAmount:  response[i].FillSize,
				RemainingAmount: response[i].Size - response[i].FillSize,
				Fee:             response[i].TransactionFee,
				FeeAsset:        currency.NewCode(response[i].FeeCurrency),
				Exchange:        ok.Name,
				OrderID:         response[i].OrderID,
				ClientOrderID:   response[i].ClientSupplierOrderID,
				Type:            oType,
				Side:            orderSide,
				Status:          orderStatus,
				AssetType:       req.AssetType,
				Date:            response[i].CreationTime,
				LastUpdated:     response[i].UpdateTime,
				Pair:            pair,
			})
		}
	}
	return orders, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (ok *Okx) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !ok.AreCredentialsValid(ctx) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return ok.GetFee(ctx, feeBuilder)
}

// ValidateCredentials validates current credentials used for wrapper
func (ok *Okx) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := ok.UpdateAccountInfo(ctx, assetType)
	return ok.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := ok.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	if kline.TotalCandlesPerInterval(start, end, interval) > 100 {
		return kline.Item{}, errors.New(kline.ErrRequestExceedsExchangeLimits)
	}
	format, err := ok.GetPairFormat(a, false)
	if err != nil {
		return kline.Item{}, err
	}
	if !pair.IsPopulated() {
		return kline.Item{}, errIncompleteCurrencyPair
	}
	instrumentID := format.Format(pair)
	candles, err := ok.GetCandlesticksHistory(ctx, instrumentID, interval, start, end, 100)
	if err != nil {
		return kline.Item{}, err
	}
	response := kline.Item{
		Exchange: ok.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	for x := range candles {
		response.Candles = append(response.Candles, kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].OpenPrice,
			High:   candles[x].HighestPrice,
			Low:    candles[x].LowestPrice,
			Close:  candles[x].ClosePrice,
			Volume: candles[x].Volume,
		})
	}
	response.SortCandlesByTimestamp(false)
	return response, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	format, err := ok.GetPairFormat(a, false)
	if err != nil {
		return kline.Item{}, err
	}
	err = ok.ValidateKline(pair, a, interval)
	if err != nil {
		return kline.Item{}, err
	}
	instrumentID := format.Format(pair)
	if err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: ok.Name,
		Pair:     pair,
		Interval: interval,
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, 100)
	if err != nil {
		return kline.Item{}, err
	}
	for y := range dates.Ranges {
		candles, err := ok.GetCandlesticksHistory(ctx, instrumentID, interval, dates.Ranges[y].Start.Time, dates.Ranges[y].End.Time, 100)
		if err != nil {
			return kline.Item{}, err
		}
		for x := range candles {
			ret.Candles = append(ret.Candles, kline.Candle{
				Time:   candles[x].OpenTime,
				Open:   candles[x].OpenPrice,
				High:   candles[x].HighestPrice,
				Low:    candles[x].LowestPrice,
				Close:  candles[x].ClosePrice,
				Volume: candles[x].Volume,
			})
		}
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
