package okx

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
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
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
	okxWebsocketResponseMaxLimit = time.Second * 3
)

// SetDefaults sets the basic defaults for Okx
func (ok *Okx) SetDefaults() {
	ok.Name = "Okx"
	ok.Enabled = true
	ok.Verbose = true

	ok.API.CredentialsValidator.RequiresKey = true
	ok.API.CredentialsValidator.RequiresSecret = true
	ok.API.CredentialsValidator.RequiresClientID = true

	cpf := &currency.PairFormat{
		Delimiter: currency.DashDelimiter,
		Uppercase: true,
	}

	err := ok.SetGlobalPairsManager(cpf, cpf, asset.Spot, asset.Futures, asset.PerpetualSwap, asset.Options, asset.Margin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	ok.Features = exchange.Features{
		CurrencyTranslations: currency.NewTranslations(map[currency.Code]currency.Code{
			currency.NewCode("USDT-SWAP"): currency.USDT,
			currency.NewCode("USD-SWAP"):  currency.USD,
			currency.NewCode("USDC-SWAP"): currency.USDC,
		}),
		Supports: exchange.FeaturesSupported{
			REST:                true,
			Websocket:           true,
			MaximumOrderHistory: kline.OneDay.Duration() * 90,
			RESTCapabilities: protocol.Features{
				TickerFetching:        true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				CryptoDeposit:         true,
				CryptoWithdrawalFee:   true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				SubmitOrder:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				CancelOrders:          true,
				TradeFetching:         true,
				UserTradeHistory:      true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
				KlineFetching:         true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				ModifyOrder:           true,
				FundingRateFetching:   true,
				PredictedFundingRate:  true,
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
				SubmitOrder:            true,
				CancelOrder:            true,
				CancelOrders:           true,
				ModifyOrder:            true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:      true,
				Leverage:       true,
				CollateralMode: true,
				OpenInterest: exchange.OpenInterestSupport{
					Supported:         true,
					SupportsRestBatch: true,
				},
				FundingRates:              true,
				MaximumFundingRateHistory: kline.ThreeMonth.Duration(),
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
				},
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
					kline.IntervalCapacity{Interval: kline.FiveDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
					kline.IntervalCapacity{Interval: kline.ThreeMonth},
					kline.IntervalCapacity{Interval: kline.SixMonth},
					kline.IntervalCapacity{Interval: kline.OneYear},
				),
				GlobalResultLimit: 100, // Reference: https://www.okx.com/docs-v5/en/#rest-api-market-data-get-candlesticks-history
			},
		},
	}
	ok.Requester, err = request.New(ok.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
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

	ok.Websocket = stream.NewWebsocket()
	ok.WebsocketResponseMaxLimit = okxWebsocketResponseMaxLimit
	ok.WebsocketResponseCheckTimeout = okxWebsocketResponseMaxLimit
	ok.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (ok *Okx) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		ok.SetEnabled(false)
		return nil
	}
	if err := ok.SetupDefaults(exch); err != nil {
		return err
	}

	wsRunningEndpoint, err := ok.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	if err := ok.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:                         exch,
		DefaultURL:                             okxAPIWebsocketPublicURL,
		RunningURL:                             wsRunningEndpoint,
		Connector:                              ok.WsConnect,
		Subscriber:                             ok.Subscribe,
		Unsubscriber:                           ok.Unsubscribe,
		GenerateSubscriptions:                  ok.GenerateDefaultSubscriptions,
		Features:                               &ok.Features.Supports.WebsocketCapabilities,
		MaxWebsocketSubscriptionsPerConnection: 240,
		OrderbookBufferConfig: buffer.Config{
			Checksum: ok.CalculateUpdateOrderbookChecksum,
		},
	}); err != nil {
		return err
	}

	if err := ok.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                      okxAPIWebsocketPublicURL,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         okxWebsocketResponseMaxLimit,
		RateLimit:                request.NewRateLimitWithWeight(time.Second, 2, 1),
		BespokeGenerateMessageID: func(bool) int64 { return ok.Counter.IncrementAndGet() },
	}); err != nil {
		return err
	}

	return ok.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                      okxAPIWebsocketPrivateURL,
		ResponseCheckTimeout:     exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         okxWebsocketResponseMaxLimit,
		Authenticated:            true,
		RateLimit:                request.NewRateLimitWithWeight(time.Second, 2, 1),
		BespokeGenerateMessageID: func(bool) int64 { return ok.Counter.IncrementAndGet() },
	})
}

// GetServerTime returns the current exchange server time.
func (ok *Okx) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return ok.GetSystemTime(ctx)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (ok *Okx) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	insts, err := ok.getInstrumentsForAsset(ctx, a)
	if err != nil {
		return nil, err
	}
	pf, err := ok.CurrencyPairs.GetFormat(a, false)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, len(insts))
	for x := range insts {
		pairs[x], err = currency.NewPairDelimiter(insts[x].InstrumentID, pf.Delimiter)
		if err != nil {
			return nil, err
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (ok *Okx) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := ok.GetAssetTypes(false)
	for i := range assetTypes {
		pairs, err := ok.FetchTradablePairs(ctx, assetTypes[i])
		if err != nil {
			return fmt.Errorf("%w for asset %v", err, assetTypes[i])
		}
		err = ok.UpdatePairs(pairs, assetTypes[i], false, forceUpdate)
		if err != nil {
			return fmt.Errorf("%w for asset %v", err, assetTypes[i])
		}
	}
	return ok.EnsureOnePairEnabled()
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (ok *Okx) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	insts, err := ok.getInstrumentsForAsset(ctx, a)
	if err != nil {
		return err
	}
	if len(insts) == 0 {
		return errNoInstrumentFound
	}
	limits := make([]order.MinMaxLevel, len(insts))
	for x := range insts {
		pair, err := currency.NewPairFromString(insts[x].InstrumentID)
		if err != nil {
			return err
		}

		limits[x] = order.MinMaxLevel{
			Pair:                   pair,
			Asset:                  a,
			PriceStepIncrementSize: insts[x].TickSize.Float64(),
			MinimumBaseAmount:      insts[x].MinimumOrderSize.Float64(),
		}
	}

	return ok.LoadLimits(limits)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (ok *Okx) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	pairFormat, err := ok.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(p)
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}
	mdata, err := ok.GetTicker(ctx, instrumentID)
	if err != nil {
		return nil, err
	}
	var baseVolume, quoteVolume float64
	switch a {
	case asset.Spot, asset.Margin:
		baseVolume = mdata.Vol24H.Float64()
		quoteVolume = mdata.VolCcy24H.Float64()
	case asset.PerpetualSwap, asset.Futures, asset.Options:
		baseVolume = mdata.VolCcy24H.Float64()
		quoteVolume = mdata.Vol24H.Float64()
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	err = ticker.ProcessTicker(&ticker.Price{
		Last:         mdata.LastTradePrice.Float64(),
		High:         mdata.High24H.Float64(),
		Low:          mdata.Low24H.Float64(),
		Bid:          mdata.BestBidPrice.Float64(),
		BidSize:      mdata.BestBidSize.Float64(),
		Ask:          mdata.BestAskPrice.Float64(),
		AskSize:      mdata.BestAskSize.Float64(),
		Volume:       baseVolume,
		QuoteVolume:  quoteVolume,
		Open:         mdata.Open24H.Float64(),
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
	if assetType == asset.Margin {
		instrumentType = okxInstTypeSpot
	}
	ticks, err := ok.GetTickers(ctx, instrumentType, "", "")
	if err != nil {
		return err
	}

	for y := range ticks {
		pair, err := currency.NewPairFromString(ticks[y].InstrumentID)
		if err != nil {
			return err
		}
		for i := range pairs {
			pairFmt, err := ok.FormatExchangeCurrency(pairs[i], assetType)
			if err != nil {
				return err
			}
			if !pair.Equal(pairFmt) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks[y].LastTradePrice.Float64(),
				High:         ticks[y].High24H.Float64(),
				Low:          ticks[y].Low24H.Float64(),
				Bid:          ticks[y].BestBidPrice.Float64(),
				BidSize:      ticks[y].BestBidSize.Float64(),
				Ask:          ticks[y].BestAskPrice.Float64(),
				AskSize:      ticks[y].BestAskSize.Float64(),
				Volume:       ticks[y].Vol24H.Float64(),
				QuoteVolume:  ticks[y].VolCcy24H.Float64(),
				Open:         ticks[y].Open24H.Float64(),
				Pair:         pairFmt,
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
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := ok.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        ok.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: ok.CanVerifyOrderbook,
	}
	var orderbookNew *OrderBookResponse
	var err error
	err = ok.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	var instrumentID string
	pairFormat, err := ok.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if !pair.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID = pairFormat.Format(pair)
	orderbookNew, err = ok.GetOrderBookDepth(ctx, instrumentID, 400)
	if err != nil {
		return book, err
	}

	orderBookD, err := orderbookNew.GetOrderBookResponseDetail()
	if err != nil {
		return nil, err
	}
	book.Bids = make(orderbook.Tranches, len(orderBookD.Bids))
	for x := range orderBookD.Bids {
		book.Bids[x] = orderbook.Tranche{
			Amount: orderBookD.Bids[x].BaseCurrencies,
			Price:  orderBookD.Bids[x].DepthPrice,
		}
	}
	book.Asks = make(orderbook.Tranches, len(orderBookD.Asks))
	for x := range orderBookD.Asks {
		book.Asks[x] = orderbook.Tranche{
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
	if err := ok.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return account.Holdings{}, err
	}

	var info account.Holdings
	var acc account.SubAccount
	info.Exchange = ok.Name
	if !ok.SupportsAsset(assetType) {
		return info, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	accountBalances, err := ok.AccountBalance(ctx, "")
	if err != nil {
		return info, err
	}
	currencyBalances := []account.Balance{}
	for i := range accountBalances {
		for j := range accountBalances[i].Details {
			currencyBalances = append(currencyBalances, account.Balance{
				Currency: accountBalances[i].Details[j].Currency,
				Total:    accountBalances[i].Details[j].EquityOfCurrency.Float64(),
				Hold:     accountBalances[i].Details[j].FrozenBalance.Float64(),
				Free:     accountBalances[i].Details[j].AvailableBalance.Float64(),
			})
		}
	}
	acc.Currencies = currencyBalances
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

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (ok *Okx) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	depositHistories, err := ok.GetCurrencyDepositHistory(ctx, "", "", "", time.Time{}, time.Time{}, -1, 0)
	if err != nil {
		return nil, err
	}
	withdrawalHistories, err := ok.GetWithdrawalHistory(ctx, "", "", "", "", "", time.Time{}, time.Time{}, -5)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(depositHistories)+len(withdrawalHistories))
	for x := range depositHistories {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    ok.Name,
			Status:          strconv.Itoa(depositHistories[x].State),
			Timestamp:       depositHistories[x].Timestamp.Time(),
			Currency:        depositHistories[x].Currency,
			Amount:          depositHistories[x].Amount.Float64(),
			TransferType:    "deposit",
			CryptoToAddress: depositHistories[x].ToDepositAddress,
			CryptoTxID:      depositHistories[x].TransactionID,
		})
	}
	for x := range withdrawalHistories {
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    ok.Name,
			Status:          withdrawalHistories[x].StateOfWithdrawal,
			Timestamp:       withdrawalHistories[x].Timestamp.Time(),
			Currency:        withdrawalHistories[x].Currency,
			Amount:          withdrawalHistories[x].Amount.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalHistories[x].ToReceivingAddress,
			CryptoTxID:      withdrawalHistories[x].TransactionID,
			TransferID:      withdrawalHistories[x].WithdrawalID,
			Fee:             withdrawalHistories[x].WithdrawalFee.Float64(),
			CryptoChain:     withdrawalHistories[x].ChainName,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (ok *Okx) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := ok.GetWithdrawalHistory(ctx, c.String(), "", "", "", "", time.Time{}, time.Time{}, -5)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for x := range withdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:          withdrawals[x].StateOfWithdrawal,
			Timestamp:       withdrawals[x].Timestamp.Time(),
			Currency:        withdrawals[x].Currency,
			Amount:          withdrawals[x].Amount.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawals[x].ToReceivingAddress,
			CryptoTxID:      withdrawals[x].TransactionID,
			CryptoChain:     withdrawals[x].ChainName,
			TransferID:      withdrawals[x].WithdrawalID,
			Fee:             withdrawals[x].WithdrawalFee.Float64(),
		})
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (ok *Okx) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	pairFormat, err := ok.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(p)
	tradeData, err := ok.GetTrades(ctx, instrumentID, 1000)
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
			Side:         tradeData[x].Side,
			Price:        tradeData[x].Price.Float64(),
			Amount:       tradeData[x].Quantity.Float64(),
			Timestamp:    tradeData[x].Timestamp.Time(),
		}
	}
	if ok.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(ok.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades retrieves historic trade data within the timeframe provided
func (ok *Okx) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if timestampStart.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	const limit = 100
	pairFormat, err := ok.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp []trade.Data
	instrumentID := pairFormat.Format(p)
	tradeIDEnd := ""
allTrades:
	for {
		var trades []TradeResponse
		trades, err = ok.GetTradesHistory(ctx, instrumentID, "", tradeIDEnd, limit)
		if err != nil {
			return nil, err
		}
		if len(trades) == 0 {
			break
		}
		for i := range trades {
			if timestampStart.Equal(trades[i].Timestamp.Time()) ||
				trades[i].Timestamp.Time().Before(timestampStart) ||
				tradeIDEnd == trades[len(trades)-1].TradeID {
				// reached end of trades to crawl
				break allTrades
			}
			resp = append(resp, trade.Data{
				TID:          trades[i].TradeID,
				Exchange:     ok.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        trades[i].Price.Float64(),
				Amount:       trades[i].Quantity.Float64(),
				Timestamp:    trades[i].Timestamp.Time(),
				Side:         trades[i].Side,
			})
		}
		tradeIDEnd = trades[len(trades)-1].TradeID
	}
	if ok.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(ok.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (ok *Okx) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(ok.GetTradingRequirements()); err != nil {
		return nil, err
	}
	if !ok.SupportsAsset(s.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, s.AssetType)
	}
	if s.Amount <= 0 {
		return nil, errors.New("amount, or size (sz) of quantity to buy or sell hast to be greater than zero")
	}
	pairFormat, err := ok.GetPairFormat(s.AssetType, true)
	if err != nil {
		return nil, err
	}
	if s.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(s.Pair)
	tradeMode := ok.marginTypeToString(s.MarginType)
	if s.Leverage != 0 && s.Leverage != 1 {
		return nil, fmt.Errorf("%w received '%v'", order.ErrSubmitLeverageNotSupported, s.Leverage)
	}
	var sideType string
	if s.Side.IsLong() {
		sideType = order.Buy.Lower()
	} else {
		sideType = order.Sell.Lower()
	}

	amount := s.Amount
	var targetCurrency string
	if s.AssetType == asset.Spot && s.Type == order.Market {
		targetCurrency = "base_ccy" // Default to base currency
		if s.QuoteAmount > 0 {
			amount = s.QuoteAmount
			targetCurrency = "quote_ccy"
		}
	}

	var orderRequest = &PlaceOrderRequestParam{
		InstrumentID:  instrumentID,
		TradeMode:     tradeMode,
		Side:          sideType,
		OrderType:     s.Type.Lower(),
		Amount:        amount,
		ClientOrderID: s.ClientOrderID,
		Price:         s.Price,
		QuantityType:  targetCurrency,
	}
	switch s.Type.Lower() {
	case OkxOrderLimit, OkxOrderPostOnly, OkxOrderFOK, OkxOrderIOC:
		orderRequest.Price = s.Price
	}
	var placeOrderResponse *OrderData
	if s.AssetType == asset.PerpetualSwap || s.AssetType == asset.Futures {
		if s.Type.Lower() == "" {
			orderRequest.OrderType = OkxOrderOptimalLimitIOC
		}
		// TODO: handle positionSideLong while side is Short and positionSideShort while side is Long
		if s.Side.IsLong() {
			orderRequest.PositionSide = positionSideLong
		} else {
			orderRequest.PositionSide = positionSideShort
		}
	}
	if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		placeOrderResponse, err = ok.WsPlaceOrder(ctx, orderRequest)
		if err != nil {
			return nil, err
		}
	} else {
		placeOrderResponse, err = ok.PlaceOrder(ctx, orderRequest, s.AssetType)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(placeOrderResponse.OrderID)
}

func (ok *Okx) marginTypeToString(m margin.Type) string {
	switch m {
	case margin.Isolated:
		return "isolated"
	case margin.Multi:
		return "cross"
	default:
		return "cash"
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
func (ok *Okx) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var err error
	if math.Trunc(action.Amount) != action.Amount {
		return nil, errors.New("okx contract amount can not be decimal")
	}
	pairFormat, err := ok.GetPairFormat(action.AssetType, true)
	if err != nil {
		return nil, err
	}
	if action.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	amendRequest := AmendOrderRequestParams{
		InstrumentID:  pairFormat.Format(action.Pair),
		NewQuantity:   action.Amount,
		OrderID:       action.OrderID,
		ClientOrderID: action.ClientOrderID,
	}
	if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = ok.WsAmendOrder(ctx, &amendRequest)
	} else {
		_, err = ok.AmendOrder(ctx, &amendRequest)
	}
	if err != nil {
		return nil, err
	}
	return action.DeriveModifyResponse()
}

// CancelOrder cancels an order by its corresponding ID number
func (ok *Okx) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	if !ok.SupportsAsset(ord.AssetType) {
		return fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
	}
	pairFormat, err := ok.GetPairFormat(ord.AssetType, true)
	if err != nil {
		return err
	}
	if ord.Pair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	instrumentID := pairFormat.Format(ord.Pair)
	req := CancelOrderRequestParam{
		InstrumentID:  instrumentID,
		OrderID:       ord.OrderID,
		ClientOrderID: ord.ClientOrderID,
	}
	if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		_, err = ok.WsCancelOrder(ctx, req)
	} else {
		_, err = ok.CancelSingleOrder(ctx, req)
	}
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (ok *Okx) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) > 20 {
		return nil, fmt.Errorf("%w, cannot cancel more than 20 orders", errExceedLimit)
	} else if len(o) == 0 {
		return nil, fmt.Errorf("%w, must have at least 1 cancel order", order.ErrCancelOrderIsNil)
	}
	cancelOrderParams := make([]CancelOrderRequestParam, len(o))
	var err error
	for x := range o {
		ord := o[x]
		err = ord.Validate(ord.StandardCancel())
		if err != nil {
			return nil, err
		}
		if !ok.SupportsAsset(ord.AssetType) {
			return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, ord.AssetType)
		}
		var pairFormat currency.PairFormat
		pairFormat, err = ok.GetPairFormat(ord.AssetType, true)
		if err != nil {
			return nil, err
		}
		if !ord.Pair.IsPopulated() {
			return nil, errIncompleteCurrencyPair
		}
		cancelOrderParams[x] = CancelOrderRequestParam{
			InstrumentID:  pairFormat.Format(ord.Pair),
			OrderID:       ord.OrderID,
			ClientOrderID: ord.ClientOrderID,
		}
	}
	var canceledOrders []OrderData
	if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		canceledOrders, err = ok.WsCancelMultipleOrder(ctx, cancelOrderParams)
	} else {
		canceledOrders, err = ok.CancelMultipleOrders(ctx, cancelOrderParams)
	}
	if err != nil {
		return nil, err
	}
	resp := &order.CancelBatchResponse{Status: make(map[string]string)}
	for x := range canceledOrders {
		resp.Status[canceledOrders[x].OrderID] = func() string {
			if canceledOrders[x].SCode > 0 {
				return ""
			}
			return order.Cancelled.String()
		}()
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (ok *Okx) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	err := orderCancellation.Validate()
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	cancelAllResponse := order.CancelAllResponse{
		Status: map[string]string{},
	}
	var instrumentType string
	if orderCancellation.AssetType.IsValid() {
		err = ok.CurrencyPairs.IsAssetEnabled(orderCancellation.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		instrumentType = ok.GetInstrumentTypeFromAssetItem(orderCancellation.AssetType)
	}
	var oType string
	if orderCancellation.Type != order.UnknownType && orderCancellation.Type != order.AnyType {
		oType, err = ok.OrderTypeString(orderCancellation.Type)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
	}
	var curr string
	if orderCancellation.Pair.IsPopulated() {
		curr = orderCancellation.Pair.Upper().String()
	}
	myOrders, err := ok.GetOrderList(ctx, &OrderListRequestParams{
		InstrumentType: instrumentType,
		OrderType:      oType,
		InstrumentID:   curr,
	})
	if err != nil {
		return cancelAllResponse, err
	}
	cancelAllOrdersRequestParams := make([]CancelOrderRequestParam, len(myOrders))
ordersLoop:
	for x := range myOrders {
		switch {
		case orderCancellation.OrderID != "" || orderCancellation.ClientOrderID != "":
			if myOrders[x].OrderID == orderCancellation.OrderID ||
				myOrders[x].ClientOrderID == orderCancellation.ClientOrderID {
				cancelAllOrdersRequestParams[x] = CancelOrderRequestParam{
					OrderID:       myOrders[x].OrderID,
					ClientOrderID: myOrders[x].ClientOrderID,
				}
				break ordersLoop
			}
		case orderCancellation.Side == order.Buy || orderCancellation.Side == order.Sell:
			if myOrders[x].Side == order.Buy || myOrders[x].Side == order.Sell {
				cancelAllOrdersRequestParams[x] = CancelOrderRequestParam{
					OrderID:       myOrders[x].OrderID,
					ClientOrderID: myOrders[x].ClientOrderID,
				}
				continue
			}
		default:
			cancelAllOrdersRequestParams[x] = CancelOrderRequestParam{
				OrderID:       myOrders[x].OrderID,
				ClientOrderID: myOrders[x].ClientOrderID,
			}
		}
	}
	remaining := cancelAllOrdersRequestParams
	loop := int(math.Ceil(float64(len(remaining)) / 20.0))
	for range loop {
		var response []OrderData
		if len(remaining) > 20 {
			if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				response, err = ok.WsCancelMultipleOrder(ctx, remaining[:20])
			} else {
				response, err = ok.CancelMultipleOrders(ctx, remaining[:20])
			}
			remaining = remaining[20:]
		} else {
			if ok.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				response, err = ok.WsCancelMultipleOrder(ctx, remaining)
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
			if response[y].SCode == 0 {
				cancelAllResponse.Status[response[y].OrderID] = order.Cancelled.String()
			} else {
				cancelAllResponse.Status[response[y].OrderID] = response[y].SMessage
			}
		}
	}
	return cancelAllResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (ok *Okx) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := ok.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	pairFormat, err := ok.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}
	if !pair.IsPopulated() {
		return nil, errIncompleteCurrencyPair
	}
	instrumentID := pairFormat.Format(pair)
	if !ok.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, assetType)
	}
	orderDetail, err := ok.GetOrderDetail(ctx, &OrderDetailRequestParam{
		InstrumentID: instrumentID,
		OrderID:      orderID,
	})
	if err != nil {
		return nil, err
	}
	status, err := order.StringToOrderStatus(orderDetail.State)
	if err != nil {
		return nil, err
	}
	orderType, err := ok.OrderTypeFromString(orderDetail.OrderType)
	if err != nil {
		return nil, err
	}

	return &order.Detail{
		Amount:         orderDetail.Size.Float64(),
		Exchange:       ok.Name,
		OrderID:        orderDetail.OrderID,
		ClientOrderID:  orderDetail.ClientOrderID,
		Side:           orderDetail.Side,
		Type:           orderType,
		Pair:           pair,
		Cost:           orderDetail.Price.Float64(),
		AssetType:      assetType,
		Status:         status,
		Price:          orderDetail.Price.Float64(),
		ExecutedAmount: orderDetail.RebateAmount.Float64(),
		Date:           orderDetail.CreationTime,
		LastUpdated:    orderDetail.UpdateTime,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (ok *Okx) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	response, err := ok.GetCurrencyDepositAddress(ctx, c.String())
	if err != nil {
		return nil, err
	}

	// Check if a specific chain was requested
	if chain != "" {
		for x := range response {
			if !strings.EqualFold(response[x].Chain, chain) {
				continue
			}
			return &deposit.Address{
				Address: response[x].Address,
				Tag:     response[x].Tag,
				Chain:   response[x].Chain,
			}, nil
		}
		return nil, fmt.Errorf("specified chain %s not found", chain)
	}

	// If no specific chain was requested, return the first selected address (mainnet addresses are returned first by default)
	for x := range response {
		if !response[x].Selected {
			continue
		}

		return &deposit.Address{
			Address: response[x].Address,
			Tag:     response[x].Tag,
			Chain:   response[x].Chain,
		}, nil
	}
	return nil, errDepositAddressNotFound
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (ok *Okx) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	input := WithdrawalInput{
		ChainName:             withdrawRequest.Crypto.Chain,
		Amount:                withdrawRequest.Amount,
		Currency:              withdrawRequest.Currency.String(),
		ToAddress:             withdrawRequest.Crypto.Address,
		TransactionFee:        withdrawRequest.Crypto.FeeAmount,
		WithdrawalDestination: "3",
	}
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
func (ok *Okx) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (ok *Okx) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (ok *Okx) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.StartTime.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	if !ok.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}
	instrumentType := ok.GetInstrumentTypeFromAssetItem(req.AssetType)
	var orderType string
	if req.Type != order.UnknownType && req.Type != order.AnyType {
		orderType, err = ok.OrderTypeString(req.Type)
		if err != nil {
			return nil, err
		}
	}
	endTime := req.EndTime
	var resp []order.Detail
allOrders:
	for {
		requestParam := &OrderListRequestParams{
			OrderType:      orderType,
			End:            endTime,
			InstrumentType: instrumentType,
		}
		var orderList []OrderDetail
		orderList, err = ok.GetOrderList(ctx, requestParam)
		if err != nil {
			return nil, err
		}
		if len(orderList) == 0 {
			break
		}
		for i := range orderList {
			if req.StartTime.Equal(orderList[i].CreationTime) ||
				orderList[i].CreationTime.Before(req.StartTime) ||
				endTime == orderList[i].CreationTime {
				// reached end of orders to crawl
				break allOrders
			}
			orderSide := orderList[i].Side
			pair, err := currency.NewPairFromString(orderList[i].InstrumentID)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) > 0 {
				x := 0
				for x = range req.Pairs {
					if req.Pairs[x].Equal(pair) {
						break
					}
				}
				if !req.Pairs[x].Equal(pair) {
					continue
				}
			}
			var orderStatus order.Status
			orderStatus, err = order.StringToOrderStatus(strings.ToUpper(orderList[i].State))
			if err != nil {
				return nil, err
			}
			var oType order.Type
			oType, err = ok.OrderTypeFromString(orderList[i].OrderType)
			if err != nil {
				return nil, err
			}
			resp = append(resp, order.Detail{
				Amount:          orderList[i].Size.Float64(),
				Pair:            pair,
				Price:           orderList[i].Price.Float64(),
				ExecutedAmount:  orderList[i].FillSize.Float64(),
				RemainingAmount: orderList[i].Size.Float64() - orderList[i].FillSize.Float64(),
				Fee:             orderList[i].TransactionFee.Float64(),
				FeeAsset:        currency.NewCode(orderList[i].FeeCurrency),
				Exchange:        ok.Name,
				OrderID:         orderList[i].OrderID,
				ClientOrderID:   orderList[i].ClientOrderID,
				Type:            oType,
				Side:            orderSide,
				Status:          orderStatus,
				AssetType:       req.AssetType,
				Date:            orderList[i].CreationTime,
				LastUpdated:     orderList[i].UpdateTime,
			})
		}
		if len(orderList) < 100 {
			// Since the we passed a limit of 0 to the method GetOrderList,
			// we expect 100 orders to be retrieved if the number of orders are more that 100.
			// If not, break out of the loop to not send another request.
			break
		}
		endTime = orderList[len(orderList)-1].CreationTime
	}
	return req.Filter(ok.Name, resp), nil
}

// GetOrderHistory retrieves account order information Can Limit response to specific order status
func (ok *Okx) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && req.StartTime.Before(time.Now().Add(-kline.ThreeMonth.Duration())) {
		return nil, errOnlyThreeMonthsSupported
	}
	if len(req.Pairs) == 0 {
		return nil, errMissingAtLeast1CurrencyPair
	}
	if !ok.SupportsAsset(req.AssetType) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.AssetType)
	}
	instrumentType := ok.GetInstrumentTypeFromAssetItem(req.AssetType)
	endTime := req.EndTime
	var resp []order.Detail
allOrders:
	for {
		orderList, err := ok.Get3MonthOrderHistory(ctx, &OrderHistoryRequestParams{
			OrderListRequestParams: OrderListRequestParams{
				InstrumentType: instrumentType,
				End:            endTime,
			},
		})
		if err != nil {
			return nil, err
		}
		if len(orderList) == 0 {
			break
		}
		for i := range orderList {
			if req.StartTime.Equal(orderList[i].CreationTime) ||
				orderList[i].CreationTime.Before(req.StartTime) ||
				endTime == orderList[i].CreationTime {
				// reached end of orders to crawl
				break allOrders
			}
			pair, err := currency.NewPairFromString(orderList[i].InstrumentID)
			if err != nil {
				return nil, err
			}
			for j := range req.Pairs {
				if !req.Pairs[j].Equal(pair) {
					continue
				}
				var orderStatus order.Status
				orderStatus, err = order.StringToOrderStatus(strings.ToUpper(orderList[i].State))
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
				}
				if orderStatus == order.Active {
					continue
				}
				orderSide := orderList[i].Side
				var oType order.Type
				oType, err = ok.OrderTypeFromString(orderList[i].OrderType)
				if err != nil {
					return nil, err
				}
				orderAmount := orderList[i].Size
				if orderList[i].QuantityType == "quote_ccy" {
					// Size is quote amount.
					orderAmount /= orderList[i].AveragePrice
				}

				remainingAmount := float64(0)
				if orderStatus != order.Filled {
					remainingAmount = orderAmount.Float64() - orderList[i].AccumulatedFillSize.Float64()
				}
				resp = append(resp, order.Detail{
					Price:                orderList[i].Price.Float64(),
					AverageExecutedPrice: orderList[i].AveragePrice.Float64(),
					Amount:               orderAmount.Float64(),
					ExecutedAmount:       orderList[i].AccumulatedFillSize.Float64(),
					RemainingAmount:      remainingAmount,
					Fee:                  orderList[i].TransactionFee.Float64(),
					FeeAsset:             currency.NewCode(orderList[i].FeeCurrency),
					Exchange:             ok.Name,
					OrderID:              orderList[i].OrderID,
					ClientOrderID:        orderList[i].ClientOrderID,
					Type:                 oType,
					Side:                 orderSide,
					Status:               orderStatus,
					AssetType:            req.AssetType,
					Date:                 orderList[i].CreationTime,
					LastUpdated:          orderList[i].UpdateTime,
					Pair:                 pair,
					Cost:                 orderList[i].AveragePrice.Float64() * orderList[i].AccumulatedFillSize.Float64(),
					CostAsset:            currency.NewCode(orderList[i].RebateCurrency),
				})
			}
		}
		if len(orderList) < 100 {
			break
		}
		endTime = orderList[len(orderList)-1].CreationTime
	}
	return req.Filter(ok.Name, resp), nil
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

// ValidateAPICredentials validates current credentials used for wrapper
func (ok *Okx) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := ok.UpdateAccountInfo(ctx, assetType)
	return ok.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := ok.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	candles, err := ok.GetCandlesticksHistory(ctx,
		req.RequestFormatted.Base.String()+
			currency.DashDelimiter+
			req.RequestFormatted.Quote.String(),
		req.ExchangeInterval,
		start.Add(-time.Nanosecond), // Start time not inclusive of candle.
		end,
		300)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, len(candles))
	for x := range candles {
		timeSeries[x] = kline.Candle{
			Time:   candles[x].OpenTime,
			Open:   candles[x].OpenPrice,
			High:   candles[x].HighestPrice,
			Low:    candles[x].LowestPrice,
			Close:  candles[x].ClosePrice,
			Volume: candles[x].Volume,
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (ok *Okx) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := ok.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	count := kline.TotalCandlesPerInterval(req.Start, req.End, req.ExchangeInterval)
	if count > 1440 {
		return nil,
			fmt.Errorf("candles count: %d max lookback: %d, %w",
				count, 1440, kline.ErrRequestExceedsMaxLookback)
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for y := range req.RangeHolder.Ranges {
		var candles []CandleStick
		candles, err = ok.GetCandlesticksHistory(ctx,
			req.RequestFormatted.Base.String()+
				currency.DashDelimiter+
				req.RequestFormatted.Quote.String(),
			req.ExchangeInterval,
			req.RangeHolder.Ranges[y].Start.Time.Add(-time.Nanosecond), // Start time not inclusive of candle.
			req.RangeHolder.Ranges[y].End.Time,
			300)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   candles[x].OpenTime,
				Open:   candles[x].OpenPrice,
				High:   candles[x].HighestPrice,
				Low:    candles[x].LowestPrice,
				Close:  candles[x].ClosePrice,
				Volume: candles[x].Volume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (ok *Okx) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	currencyChains, err := ok.GetFundingCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	chains := make([]string, 0, len(currencyChains))
	for x := range currencyChains {
		if (!cryptocurrency.IsEmpty() && !strings.EqualFold(cryptocurrency.String(), currencyChains[x].Currency)) ||
			(!currencyChains[x].CanDeposit && !currencyChains[x].CanWithdraw) ||
			// Lightning network is currently not supported by transfer chains
			// as it is an invoice string which is generated per request and is
			// not a static address. TODO: Add a hook to generate a new invoice
			// string per request.
			(currencyChains[x].Chain != "" && currencyChains[x].Chain == "BTC-Lightning") {
			continue
		}
		chains = append(chains, currencyChains[x].Chain)
	}
	return chains, nil
}

// getInstrumentsForOptions returns the instruments for options asset type
func (ok *Okx) getInstrumentsForOptions(ctx context.Context) ([]Instrument, error) {
	underlyings, err := ok.GetPublicUnderlyings(context.Background(), okxInstTypeOption)
	if err != nil {
		return nil, err
	}
	var insts []Instrument
	for x := range underlyings {
		var instruments []Instrument
		instruments, err = ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeOption,
			Underlying:     underlyings[x],
		})
		if err != nil {
			return nil, err
		}
		insts = append(insts, instruments...)
	}
	return insts, nil
}

// getInstrumentsForAsset returns the instruments for an asset type
func (ok *Okx) getInstrumentsForAsset(ctx context.Context, a asset.Item) ([]Instrument, error) {
	if !ok.SupportsAsset(a) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
	}

	var instType string
	switch a {
	case asset.Options:
		return ok.getInstrumentsForOptions(ctx)
	case asset.Spot:
		instType = okxInstTypeSpot
	case asset.Futures:
		instType = okxInstTypeFutures
	case asset.PerpetualSwap:
		instType = okxInstTypeSwap
	case asset.Margin:
		instType = okxInstTypeMargin
	}

	return ok.GetInstruments(ctx, &InstrumentsFetchParams{
		InstrumentType: instType,
	})
}

// GetLatestFundingRates returns the latest funding rates data
func (ok *Okx) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.PerpetualSwap {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w, pair required", currency.ErrCurrencyPairEmpty)
	}
	format, err := ok.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := r.Pair.Format(format)
	pairRate := fundingrate.LatestRateResponse{
		TimeChecked: time.Now(),
		Exchange:    ok.Name,
		Asset:       r.Asset,
		Pair:        fPair,
	}
	fr, err := ok.GetSingleFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	var fri time.Duration
	if len(ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
		// can infer funding rate interval from the only funding rate frequency defined
		for k := range ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
			fri = k.Duration()
		}
	}
	pairRate.LatestRate = fundingrate.Rate{
		// okx funding rate is settlement time, not when it started
		Time: fr.FundingTime.Time().Add(-fri),
		Rate: fr.FundingRate.Decimal(),
	}
	if r.IncludePredictedRate {
		pairRate.TimeOfNextRate = fr.NextFundingTime.Time()
		pairRate.PredictedUpcomingRate = fundingrate.Rate{
			Time: fr.NextFundingTime.Time().Add(-fri),
			Rate: fr.NextFundingRate.Decimal(),
		}
	}
	return []fundingrate.LatestRateResponse{pairRate}, nil
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (ok *Okx) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	requestLimit := 100
	sd := r.StartDate
	maxLookback := time.Now().Add(-ok.Features.Supports.FuturesCapabilities.MaximumFundingRateHistory)
	if r.StartDate.Before(maxLookback) {
		if r.RespectHistoryLimits {
			r.StartDate = maxLookback
		} else {
			return nil, fmt.Errorf("%w earliest date is %v", fundingrate.ErrFundingRateOutsideLimits, maxLookback)
		}
		if r.EndDate.Before(maxLookback) {
			return nil, futures.ErrGetFundingDataRequired
		}
		r.StartDate = maxLookback
	}
	format, err := ok.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := r.Pair.Format(format)
	pairRate := fundingrate.HistoricalRates{
		Exchange:  ok.Name,
		Asset:     r.Asset,
		Pair:      fPair,
		StartDate: r.StartDate,
		EndDate:   r.EndDate,
	}
	// map of time indexes, allowing for easy lookup of slice index from unix time data
	mti := make(map[int64]int)
	for {
		if sd.Equal(r.EndDate) || sd.After(r.EndDate) {
			break
		}
		var frh []FundingRateResponse
		frh, err = ok.GetFundingRateHistory(ctx, fPair.String(), sd, r.EndDate, int64(requestLimit))
		if err != nil {
			return nil, err
		}
		if len(frh) == 0 {
			break
		}
		for i := range frh {
			if r.IncludePayments {
				mti[frh[i].FundingTime.Time().Unix()] = i
			}
			pairRate.FundingRates = append(pairRate.FundingRates, fundingrate.Rate{
				Time: frh[i].FundingTime.Time(),
				Rate: frh[i].RealisedRate.Decimal(),
			})
		}
		if len(frh) < requestLimit {
			break
		}
		sd = frh[len(frh)-1].FundingTime.Time()
	}
	var fr *FundingRateResponse
	fr, err = ok.GetSingleFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	if fr == nil {
		return nil, fmt.Errorf("%w GetSingleFundingRate", common.ErrNilPointer)
	}
	pairRate.LatestRate = fundingrate.Rate{
		Time: fr.FundingTime.Time(),
		Rate: fr.FundingRate.Decimal(),
	}
	pairRate.TimeOfNextRate = fr.NextFundingTime.Time()
	if r.IncludePredictedRate {
		pairRate.PredictedUpcomingRate = fundingrate.Rate{
			Time: fr.NextFundingTime.Time(),
			Rate: fr.NextFundingRate.Decimal(),
		}
	}
	if r.IncludePayments {
		pairRate.PaymentCurrency = r.Pair.Base
		if !r.PaymentCurrency.IsEmpty() {
			pairRate.PaymentCurrency = r.PaymentCurrency
		}
		sd = r.StartDate
		billDetailsFunc := ok.GetBillsDetail3Months
		if time.Since(r.StartDate) < kline.OneWeek.Duration() {
			billDetailsFunc = ok.GetBillsDetailLast7Days
		}
		for {
			if sd.Equal(r.EndDate) || sd.After(r.EndDate) {
				break
			}
			var fri time.Duration
			if len(ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
				// can infer funding rate interval from the only funding rate frequency defined
				for k := range ok.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
					fri = k.Duration()
				}
			}
			var billDetails []BillsDetailResponse
			billDetails, err = billDetailsFunc(ctx, &BillsDetailQueryParameter{
				InstrumentType: ok.GetInstrumentTypeFromAssetItem(r.Asset),
				Currency:       pairRate.PaymentCurrency.String(),
				BillType:       137,
				BeginTime:      sd,
				EndTime:        r.EndDate,
				Limit:          int64(requestLimit),
			})
			if err != nil {
				return nil, err
			}
			for i := range billDetails {
				if index, okay := mti[billDetails[i].Timestamp.Time().Truncate(fri).Unix()]; okay {
					pairRate.FundingRates[index].Payment = billDetails[i].ProfitAndLoss.Decimal()
					continue
				}
			}
			if len(billDetails) < requestLimit {
				break
			}
			sd = billDetails[len(billDetails)-1].Timestamp.Time()
		}

		for i := range pairRate.FundingRates {
			pairRate.PaymentSum = pairRate.PaymentSum.Add(pairRate.FundingRates[i].Payment)
		}
	}
	return &pairRate, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (ok *Okx) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.PerpetualSwap, nil
}

// SetMarginType sets the default margin type for when opening a new position
// okx allows this to be set with an order, however this sets a default
func (ok *Okx) SetMarginType(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type) error {
	return fmt.Errorf("%w margin type is set per order", common.ErrFunctionNotSupported)
}

// SetCollateralMode sets the collateral type for your account
func (ok *Okx) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// GetCollateralMode returns the collateral type for your account
func (ok *Okx) GetCollateralMode(ctx context.Context, item asset.Item) (collateral.Mode, error) {
	if !ok.SupportsAsset(item) {
		return 0, fmt.Errorf("%w: %v", asset.ErrNotSupported, item)
	}
	cfg, err := ok.GetAccountConfiguration(ctx)
	if err != nil {
		return 0, err
	}
	switch cfg[0].AccountLevel {
	case 1:
		if item != asset.Spot {
			return 0, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
		}
		fallthrough
	case 2:
		return collateral.SingleMode, nil
	case 3:
		return collateral.MultiMode, nil
	case 4:
		return collateral.PortfolioMode, nil
	default:
		return collateral.UnknownMode, fmt.Errorf("%w %v", order.ErrCollateralInvalid, cfg[0].AccountLevel)
	}
}

// ChangePositionMargin will modify a position/currencies margin parameters
func (ok *Okx) ChangePositionMargin(ctx context.Context, req *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionChangeRequest", common.ErrNilPointer)
	}
	if !ok.SupportsAsset(req.Asset) {
		return nil, fmt.Errorf("%w: %v", asset.ErrNotSupported, req.Asset)
	}
	if req.NewAllocatedMargin == 0 {
		return nil, fmt.Errorf("%w %v %v", margin.ErrNewAllocatedMarginRequired, req.Asset, req.Pair)
	}
	if req.OriginalAllocatedMargin == 0 {
		return nil, margin.ErrOriginalPositionMarginRequired
	}
	if req.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, req.MarginType)
	}
	pairFormat, err := ok.GetPairFormat(req.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := req.Pair.Format(pairFormat)
	marginType := "add"
	amt := req.NewAllocatedMargin - req.OriginalAllocatedMargin
	if req.NewAllocatedMargin < req.OriginalAllocatedMargin {
		marginType = "reduce"
		amt = req.OriginalAllocatedMargin - req.NewAllocatedMargin
	}
	if req.MarginSide == "" {
		req.MarginSide = "net"
	}
	r := &IncreaseDecreaseMarginInput{
		InstrumentID: fPair.String(),
		PositionSide: req.MarginSide,
		Type:         marginType,
		Amount:       amt,
	}

	if req.Asset == asset.Margin {
		r.Currency = req.Pair.Base.Item.Symbol
	}

	resp, err := ok.IncreaseDecreaseMargin(ctx, r)
	if err != nil {
		return nil, err
	}

	return &margin.PositionChangeResponse{
		Exchange:        ok.Name,
		Pair:            req.Pair,
		Asset:           req.Asset,
		AllocatedMargin: resp.Amount.Float64(),
		MarginType:      req.MarginType,
	}, nil
}

// GetFuturesPositionSummary returns position summary details for an active position
func (ok *Okx) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if req.CalculateOffline {
		return nil, common.ErrCannotCalculateOffline
	}
	if !ok.SupportsAsset(req.Asset) || !req.Asset.IsFutures() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	fPair, err := ok.FormatExchangeCurrency(req.Pair, req.Asset)
	if err != nil {
		return nil, err
	}
	instrumentType := ok.GetInstrumentTypeFromAssetItem(req.Asset)

	var contracts []futures.Contract
	contracts, err = ok.GetFuturesContractDetails(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	multiplier := 1.0
	var contractSettlementType futures.ContractSettlementType
	for i := range contracts {
		if !contracts[i].Name.Equal(fPair) {
			continue
		}
		multiplier = contracts[i].Multiplier
		contractSettlementType = contracts[i].SettlementType
		break
	}

	positionSummaries, err := ok.GetPositions(ctx, instrumentType, fPair.String(), "")
	if err != nil {
		return nil, err
	}
	var positionSummary *AccountPosition
	for i := range positionSummaries {
		if positionSummaries[i].QuantityOfPosition.Float64() <= 0 {
			continue
		}
		positionSummary = &positionSummaries[i]
		break
	}
	if positionSummary == nil {
		return nil, fmt.Errorf("%w, received '%v', no positions found", errOnlyOneResponseExpected, len(positionSummaries))
	}
	marginMode := margin.Isolated
	if positionSummary.MarginMode == "cross" {
		marginMode = margin.Multi
	}

	acc, err := ok.AccountBalance(ctx, "")
	if err != nil {
		return nil, err
	}
	if len(acc) != 1 {
		return nil, fmt.Errorf("%w, received '%v'", errOnlyOneResponseExpected, len(acc))
	}
	var (
		freeCollateral, totalCollateral, equityOfCurrency, frozenBalance,
		availableEquity, cashBalance, discountEquity,
		equityUSD, totalEquity, isolatedEquity, isolatedLiabilities,
		isolatedUnrealisedProfit, notionalLeverage,
		strategyEquity decimal.Decimal
	)

	for i := range acc[0].Details {
		if !acc[0].Details[i].Currency.Equal(positionSummary.Currency) {
			continue
		}
		freeCollateral = acc[0].Details[i].AvailableBalance.Decimal()
		frozenBalance = acc[0].Details[i].FrozenBalance.Decimal()
		totalCollateral = freeCollateral.Add(frozenBalance)
		equityOfCurrency = acc[0].Details[i].EquityOfCurrency.Decimal()
		availableEquity = acc[0].Details[i].AvailableEquity.Decimal()
		cashBalance = acc[0].Details[i].CashBalance.Decimal()
		discountEquity = acc[0].Details[i].DiscountEquity.Decimal()
		equityUSD = acc[0].Details[i].EquityUsd.Decimal()
		totalEquity = acc[0].Details[i].TotalEquity.Decimal()
		isolatedEquity = acc[0].Details[i].IsoEquity.Decimal()
		isolatedLiabilities = acc[0].Details[i].IsolatedLiabilities.Decimal()
		isolatedUnrealisedProfit = acc[0].Details[i].IsoUpl.Decimal()
		notionalLeverage = acc[0].Details[i].NotionalLever.Decimal()
		strategyEquity = acc[0].Details[i].StrategyEquity.Decimal()

		break
	}
	collateralMode, err := ok.GetCollateralMode(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	return &futures.PositionSummary{
		Pair:            req.Pair,
		Asset:           req.Asset,
		MarginType:      marginMode,
		CollateralMode:  collateralMode,
		Currency:        positionSummary.Currency,
		AvailableEquity: availableEquity,
		CashBalance:     cashBalance,
		DiscountEquity:  discountEquity,
		EquityUSD:       equityUSD,

		IsolatedEquity:               isolatedEquity,
		IsolatedLiabilities:          isolatedLiabilities,
		IsolatedUPL:                  isolatedUnrealisedProfit,
		NotionalLeverage:             notionalLeverage,
		TotalEquity:                  totalEquity,
		StrategyEquity:               strategyEquity,
		IsolatedMargin:               positionSummary.Margin.Decimal(),
		NotionalSize:                 positionSummary.NotionalUsd.Decimal(),
		Leverage:                     positionSummary.Leverage.Decimal(),
		MaintenanceMarginRequirement: positionSummary.MaintenanceMarginRequirement.Decimal(),
		InitialMarginRequirement:     positionSummary.InitialMarginRequirement.Decimal(),
		EstimatedLiquidationPrice:    positionSummary.LiquidationPrice.Decimal(),
		CollateralUsed:               positionSummary.Margin.Decimal(),
		MarkPrice:                    positionSummary.MarkPrice.Decimal(),
		CurrentSize:                  positionSummary.QuantityOfPosition.Decimal().Mul(decimal.NewFromFloat(multiplier)),
		ContractSize:                 positionSummary.QuantityOfPosition.Decimal(),
		ContractMultiplier:           decimal.NewFromFloat(multiplier),
		ContractSettlementType:       contractSettlementType,
		AverageOpenPrice:             positionSummary.AveragePrice.Decimal(),
		UnrealisedPNL:                positionSummary.UPNL.Decimal(),
		MaintenanceMarginFraction:    positionSummary.MarginRatio.Decimal(),
		FreeCollateral:               freeCollateral,
		TotalCollateral:              totalCollateral,
		FrozenBalance:                frozenBalance,
		EquityOfCurrency:             equityOfCurrency,
	}, nil
}

// GetFuturesPositionOrders returns the orders for futures positions
func (ok *Okx) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w PositionSummaryRequest", common.ErrNilPointer)
	}
	if !ok.SupportsAsset(req.Asset) || !req.Asset.IsFutures() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, req.Asset)
	}
	if time.Since(req.StartDate) > ok.Features.Supports.MaximumOrderHistory {
		if req.RespectOrderHistoryLimits {
			req.StartDate = time.Now().Add(-ok.Features.Supports.MaximumOrderHistory)
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, time.Now().Add(-ok.Features.Supports.MaximumOrderHistory))
		}
	}
	err := common.StartEndTimeCheck(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(req.Pairs))
	var contracts []futures.Contract
	contracts, err = ok.GetFuturesContractDetails(ctx, req.Asset)
	if err != nil {
		return nil, err
	}
	for i := range req.Pairs {
		fPair, err := ok.FormatExchangeCurrency(req.Pairs[i], req.Asset)
		if err != nil {
			return nil, err
		}
		instrumentType := ok.GetInstrumentTypeFromAssetItem(req.Asset)

		multiplier := 1.0
		var contractSettlementType futures.ContractSettlementType
		if req.Asset.IsFutures() {
			for j := range contracts {
				if !contracts[j].Name.Equal(fPair) {
					continue
				}
				multiplier = contracts[j].Multiplier
				contractSettlementType = contracts[j].SettlementType
				break
			}
		}

		resp[i] = futures.PositionResponse{
			Pair:                   req.Pairs[i],
			Asset:                  req.Asset,
			ContractSettlementType: contractSettlementType,
		}

		var positions []OrderDetail
		historyRequest := &OrderHistoryRequestParams{
			OrderListRequestParams: OrderListRequestParams{
				InstrumentType: instrumentType,
				InstrumentID:   fPair.String(),
				Start:          req.StartDate,
				End:            req.EndDate,
			},
		}
		if time.Since(req.StartDate) <= time.Hour*24*7 {
			positions, err = ok.Get7DayOrderHistory(ctx, historyRequest)
		} else {
			positions, err = ok.Get3MonthOrderHistory(ctx, historyRequest)
		}
		if err != nil {
			return nil, err
		}
		for j := range positions {
			if req.Pairs[i].String() != positions[j].InstrumentID {
				continue
			}
			var orderStatus order.Status
			orderStatus, err = order.StringToOrderStatus(strings.ToUpper(positions[j].State))
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ok.Name, err)
			}
			orderSide := positions[j].Side
			var oType order.Type
			oType, err = ok.OrderTypeFromString(positions[j].OrderType)
			if err != nil {
				return nil, err
			}
			orderAmount := positions[j].Size
			if positions[j].QuantityType == "quote_ccy" {
				// Size is quote amount.
				orderAmount /= positions[j].AveragePrice
			}

			remainingAmount := float64(0)
			if orderStatus != order.Filled {
				remainingAmount = orderAmount.Float64() - positions[j].AccumulatedFillSize.Float64()
			}
			cost := positions[j].AveragePrice.Float64() * positions[j].AccumulatedFillSize.Float64()
			if multiplier != 1 {
				cost *= multiplier
			}
			resp[i].Orders = append(resp[i].Orders, order.Detail{
				Price:                positions[j].Price.Float64(),
				AverageExecutedPrice: positions[j].AveragePrice.Float64(),
				Amount:               orderAmount.Float64() * multiplier,
				ContractAmount:       orderAmount.Float64(),
				ExecutedAmount:       positions[j].AccumulatedFillSize.Float64(),
				RemainingAmount:      remainingAmount,
				Fee:                  positions[j].TransactionFee.Float64(),
				FeeAsset:             currency.NewCode(positions[j].FeeCurrency),
				Exchange:             ok.Name,
				OrderID:              positions[j].OrderID,
				ClientOrderID:        positions[j].ClientOrderID,
				Type:                 oType,
				Side:                 orderSide,
				Status:               orderStatus,
				AssetType:            req.Asset,
				Date:                 positions[j].CreationTime,
				LastUpdated:          positions[j].UpdateTime,
				Pair:                 req.Pairs[i],
				Cost:                 cost,
				CostAsset:            currency.NewCode(positions[j].RebateCurrency),
			})
		}
	}
	return resp, nil
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (ok *Okx) SetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, amount float64, orderSide order.Side) error {
	posSide := "net"
	switch item {
	case asset.Futures, asset.PerpetualSwap:
		if marginType == margin.Isolated {
			switch {
			case orderSide == order.UnknownSide:
				return errOrderSideRequired
			case orderSide.IsLong():
				posSide = "long"
			case orderSide.IsShort():
				posSide = "short"
			default:
				return fmt.Errorf("%w %v requires long/short", errInvalidOrderSide, orderSide)
			}
		}
		fallthrough
	case asset.Margin, asset.Options:
		instrumentID, err := ok.FormatSymbol(pair, item)
		if err != nil {
			return err
		}

		marginMode := ok.marginTypeToString(marginType)
		_, err = ok.SetLeverageRate(ctx, SetLeverageInput{
			Leverage:     amount,
			MarginMode:   marginMode,
			InstrumentID: instrumentID,
			PositionSide: posSide,
		})
		return err
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (ok *Okx) GetLeverage(ctx context.Context, item asset.Item, pair currency.Pair, marginType margin.Type, orderSide order.Side) (float64, error) {
	var inspectLeverage bool
	switch item {
	case asset.Futures, asset.PerpetualSwap:
		if marginType == margin.Isolated {
			switch {
			case orderSide == order.UnknownSide:
				return 0, errOrderSideRequired
			case orderSide.IsLong(), orderSide.IsShort():
				inspectLeverage = true
			default:
				return 0, fmt.Errorf("%w %v requires long/short", errInvalidOrderSide, orderSide)
			}
		}
		fallthrough
	case asset.Margin, asset.Options:
		instrumentID, err := ok.FormatSymbol(pair, item)
		if err != nil {
			return -1, err
		}
		marginMode := ok.marginTypeToString(marginType)
		lev, err := ok.GetLeverageRate(ctx, instrumentID, marginMode)
		if err != nil {
			return -1, err
		}
		if len(lev) == 0 {
			return -1, fmt.Errorf("%w %v %v %s", futures.ErrPositionNotFound, item, pair, marginType)
		}
		if inspectLeverage {
			for i := range lev {
				if lev[i].PositionSide == orderSide.Lower() {
					return lev[i].Leverage.Float64(), nil
				}
			}
		}

		// leverage is the same across positions
		return lev[0].Leverage.Float64(), nil
	default:
		return -1, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
}

// GetFuturesContractDetails returns details about futures contracts
func (ok *Okx) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !ok.SupportsAsset(item) || item == asset.Options {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	instType := ok.GetInstrumentTypeFromAssetItem(item)
	result, err := ok.GetInstruments(ctx, &InstrumentsFetchParams{
		InstrumentType: instType,
	})
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, len(result))
	for i := range result {
		var cp, underlying currency.Pair
		underlying, err = currency.NewPairFromString(result[i].Underlying)
		if err != nil {
			return nil, err
		}
		cp, err = currency.NewPairFromString(result[i].InstrumentID)
		if err != nil {
			return nil, err
		}
		settleCurr := currency.NewCode(result[i].SettlementCurrency)
		var ct futures.ContractType
		if item == asset.PerpetualSwap {
			ct = futures.Perpetual
		} else {
			switch result[i].Alias {
			case "this_week", "next_week":
				ct = futures.Weekly
			case "quarter", "next_quarter":
				ct = futures.Quarterly
			}
		}
		contractSettlementType := futures.Linear
		if result[i].SettlementCurrency == result[i].BaseCurrency {
			contractSettlementType = futures.Inverse
		}
		resp[i] = futures.Contract{
			Exchange:             ok.Name,
			Name:                 cp,
			Underlying:           underlying,
			Asset:                item,
			StartDate:            result[i].ListTime.Time,
			EndDate:              result[i].ExpTime.Time,
			IsActive:             result[i].State == "live",
			Status:               result[i].State,
			Type:                 ct,
			SettlementType:       contractSettlementType,
			SettlementCurrencies: currency.Currencies{settleCurr},
			MarginCurrency:       settleCurr,
			Multiplier:           result[i].ContractValue.Float64(),
			MaxLeverage:          result[i].MaxLeverage.Float64(),
		}
	}
	return resp, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (ok *Okx) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures && k[i].Asset != asset.PerpetualSwap {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) != 1 {
		var resp []futures.OpenInterest
		// TODO: Options support
		instTypes := map[string]asset.Item{
			"SWAP":    asset.PerpetualSwap,
			"FUTURES": asset.Futures,
		}
		for instType, v := range instTypes {
			oid, err := ok.GetOpenInterestData(ctx, instType, "", "")
			if err != nil {
				return nil, err
			}
			for j := range oid {
				p, isEnabled, err := ok.MatchSymbolCheckEnabled(oid[j].InstrumentID, v, true)
				if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
					return nil, err
				}
				if !isEnabled {
					continue
				}
				var appendData bool
				for j := range k {
					if k[j].Pair().Equal(p) {
						appendData = true
						break
					}
				}
				if len(k) > 0 && !appendData {
					continue
				}
				resp = append(resp, futures.OpenInterest{
					Key: key.ExchangePairAsset{
						Exchange: ok.Name,
						Base:     p.Base.Item,
						Quote:    p.Quote.Item,
						Asset:    v,
					},
					OpenInterest: oid[j].OpenInterest.Float64(),
				})
			}
		}
		return resp, nil
	}
	resp := make([]futures.OpenInterest, 1)
	instTypes := map[asset.Item]string{
		asset.PerpetualSwap: "SWAP",
		asset.Futures:       "FUTURES",
	}
	pFmt, err := ok.FormatSymbol(k[0].Pair(), k[0].Asset)
	if err != nil {
		return nil, err
	}
	oid, err := ok.GetOpenInterestData(ctx, instTypes[k[0].Asset], "", pFmt)
	if err != nil {
		return nil, err
	}
	for i := range oid {
		p, isEnabled, err := ok.MatchSymbolCheckEnabled(oid[i].InstrumentID, k[0].Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		resp[0] = futures.OpenInterest{
			Key: key.ExchangePairAsset{
				Exchange: ok.Name,
				Base:     p.Base.Item,
				Quote:    p.Quote.Item,
				Asset:    k[0].Asset,
			},
			OpenInterest: oid[i].OpenInterest.Float64(),
		}
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (ok *Okx) GetCurrencyTradeURL(ctx context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := ok.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	switch a {
	case asset.Spot:
		return baseURL + tradeSpot + cp.Lower().String(), nil
	case asset.Margin:
		return baseURL + tradeMargin + cp.Lower().String(), nil
	case asset.PerpetualSwap:
		return baseURL + tradePerps + cp.Lower().String(), nil
	case asset.Options:
		return baseURL + tradeOptions + cp.Base.Lower().String() + "-usd", nil
	case asset.Futures:
		cp, err = ok.FormatExchangeCurrency(cp, a)
		if err != nil {
			return "", err
		}
		insts, err := ok.GetInstruments(ctx, &InstrumentsFetchParams{
			InstrumentType: okxInstTypeFutures,
			InstrumentID:   cp.String(),
		})
		if err != nil {
			return "", err
		}
		if len(insts) != 1 {
			return "", fmt.Errorf("%w response len: %v currency expected: %v", errOnlyOneResponseExpected, len(insts), cp)
		}
		var ct string
		switch insts[0].Alias {
		case "this_week":
			ct = "-weekly"
		case "next_week":
			ct = "-biweekly"
		case "this_month":
			ct = "-monthly"
		case "next_month":
			ct = "-bimonthly"
		case "quarter":
			ct = "-quarterly"
		case "next_quarter":
			ct = "-biquarterly"
		}
		return baseURL + tradeFutures + strings.ToLower(insts[0].Underlying) + ct, nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
