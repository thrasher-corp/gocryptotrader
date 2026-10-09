package kraken

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	errUnknownOrderType     = errors.New("unknown order type")
	errUnknownOrderStatus   = errors.New("unknown order status")
	errOrderEditFailed      = errors.New("order edit failed")
	errNoDepositMethods     = errors.New("no deposit methods")
	errDepositMethodUnknown = errors.New("deposit method not offered")
	errNoWithdrawalMethods  = errors.New("no withdrawal methods")
	errTradeFeeMissing      = errors.New("trade fee missing")
)

// offlineTakerFee is the taker fee of Kraken Pro's lowest volume tier, the most a spot trade is charged
const offlineTakerFee = 0.004

// SetDefaults sets current default settings
func (e *Exchange) SetDefaults() {
	e.Name = "Kraken"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		}
		if a == asset.Futures {
			ps.RequestFormat.Delimiter = currency.UnderscoreDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		// Add Order takes a quote amount for market buys alone
		TradingRequirements: protocol.TradingRequirements{SpotMarketSellBase: true},
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:                 true,
				TickerFetching:                 true,
				KlineFetching:                  true,
				TradeFetching:                  true,
				OrderbookFetching:              true,
				AutoPairUpdates:                true,
				AccountInfo:                    true,
				GetOrder:                       true,
				GetOrders:                      true,
				CancelOrder:                    true,
				CancelOrders:                   true,
				SubmitOrder:                    true,
				ModifyOrder:                    true,
				UserTradeHistory:               true,
				CryptoDeposit:                  true,
				CryptoWithdrawal:               true,
				FiatDeposit:                    true,
				FiatWithdraw:                   true,
				TradeFee:                       true,
				FiatDepositFee:                 true,
				FiatWithdrawalFee:              true,
				CryptoDepositFee:               true,
				CryptoWithdrawalFee:            true,
				MultiChainDeposits:             true,
				MultiChainWithdrawals:          true,
				HasAssetTypeAccountSegregation: true,
				FundingRateFetching:            true,
				PredictedFundingRate:           true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:     true,
				TradeFetching:      true,
				KlineFetching:      true,
				OrderbookFetching:  true,
				Subscribe:          true,
				Unsubscribe:        true,
				MessageCorrelation: true,
				SubmitOrder:        true,
				ModifyOrder:        true,
				CancelOrder:        true,
				CancelOrders:       true,
				GetOrders:          true,
				GetOrder:           true,
				AccountBalance:     true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawCryptoWith2FA |
				exchange.AutoWithdrawFiatWithSetup |
				exchange.WithdrawFiatWith2FA,
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
					kline.IntervalCapacity{Interval: kline.FourHour},
					// Kraken starts weekly and 15 day candles at multiples of their interval since the Unix epoch, a
					// Thursday, which kline does not align to, so they are built from daily candles
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				// Get OHLC Data serves the 720 most recent candles at most
				GlobalResultLimit: 720,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(krakenRateInterval, krakenRequestRate, 1)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   spotAPIURL,
		exchange.RestFutures:                futuresAPIURL,
		exchange.RestFuturesSupplementary:   futuresSupplementaryAPIURL,
		exchange.WebsocketSpot:              wsPublicURL,
		exchange.WebsocketSpotSupplementary: wsPrivateURL,
		// The level 3 order book endpoint is the other authenticated spot endpoint, as WebsocketSpotSupplementary holds
		// the private one
		exchange.WebsocketPrivate: wsLevel3URL,
		exchange.WebsocketFutures: wsFuturesURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
}

// Setup sets current exchange configuration
func (e *Exchange) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	if err := e.SetupDefaults(exch); err != nil {
		return err
	}
	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               exch,
		Features:                     &e.Features.Supports.WebsocketCapabilities,
		TradeFeed:                    e.Features.Enabled.TradeFeed,
		FillsFeed:                    e.Features.Enabled.FillsFeed,
		UseMultiConnectionManagement: true,
	}); err != nil {
		return err
	}
	setups, err := e.connectionSetups(exch)
	if err != nil {
		return err
	}
	for _, s := range setups {
		if err := e.Websocket.SetupNewConnection(s); err != nil {
			return err
		}
	}
	return nil
}

// Bootstrap seeds the asset names that spot pairs and balances are translated with
func (e *Exchange) Bootstrap(ctx context.Context) (continueBootstrap bool, err error) {
	if err := e.SeedAssets(ctx); err != nil {
		return true, fmt.Errorf("%s error seeding asset names: %w", e.Name, err)
	}
	return true, nil
}

// ensureAssetNames seeds the asset names when they have not been
func (e *Exchange) ensureAssetNames(ctx context.Context) error {
	if e.assetNames.seeded() {
		return nil
	}
	return e.SeedAssets(ctx)
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var l []limits.MinMaxLevel
	switch a {
	case asset.Spot:
		pairs, err := e.GetAssetPairs(ctx, nil)
		if err != nil {
			return fmt.Errorf("%s error fetching %s pairs: %w", e.Name, a, err)
		}
		l = make([]limits.MinMaxLevel, 0, len(pairs))
		for name := range pairs {
			info := pairs[name]
			pair, ok := spotPairFromWebsocketName(info.WebsocketName)
			if !ok {
				continue
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize:  info.TickSize.Float64(),
				MinimumBaseAmount:       info.OrderMinimum.Float64(),
				MinimumQuoteAmount:      info.CostMinimum.Float64(),
				AmountStepIncrementSize: math.Pow10(-int(info.LotDecimals)),  //nolint:gosec // Kraken's decimals are small
				QuoteStepIncrementSize:  math.Pow10(-int(info.CostDecimals)), //nolint:gosec // Kraken's decimals are small
			})
		}
	case asset.Futures:
		instruments, err := e.GetFuturesInstruments(ctx, nil)
		if err != nil {
			return fmt.Errorf("%s error fetching %s instruments: %w", e.Name, a, err)
		}
		l = make([]limits.MinMaxLevel, 0, len(instruments.Instruments))
		for i := range instruments.Instruments {
			ins := &instruments.Instruments[i]
			if !ins.Tradeable || ins.IsExpired {
				continue
			}
			pair, err := currency.NewPairFromString(ins.Symbol)
			if err != nil {
				return err
			}
			// A negative precision requires a multiple of a power of ten
			step := math.Pow10(-int(ins.ContractValueTradePrecision))
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				PriceStepIncrementSize:  ins.TickSize,
				MinimumBaseAmount:       step,
				AmountStepIncrementSize: step,
				Listed:                  ins.OpeningDate,
				Expiry:                  ins.LastTradingTime,
			})
		}
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return limits.Load(l)
}

// spotPairFromWebsocketName returns the pair a websocket name, such as XBT/USD, names, which Kraken gives in
// alternative names
func spotPairFromWebsocketName(name string) (currency.Pair, bool) {
	base, quote, ok := strings.Cut(name, "/")
	if !ok || base == "" || quote == "" {
		return currency.EMPTYPAIR, false
	}
	return currency.NewPair(currency.NewCode(base), currency.NewCode(quote)), true
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot:
		pairs, err := e.GetAssetPairs(ctx, nil)
		if err != nil {
			return nil, err
		}
		resp := make(currency.Pairs, 0, len(pairs))
		for name := range pairs {
			if pairs[name].Status != "online" {
				continue
			}
			if pair, ok := spotPairFromWebsocketName(pairs[name].WebsocketName); ok {
				resp = append(resp, pair)
			}
		}
		return resp, nil
	case asset.Futures:
		instruments, err := e.GetFuturesInstruments(ctx, nil)
		if err != nil {
			return nil, err
		}
		resp := make(currency.Pairs, 0, len(instruments.Instruments))
		for i := range instruments.Instruments {
			if !instruments.Instruments[i].Tradeable || instruments.Instruments[i].IsExpired {
				continue
			}
			pair, err := currency.NewPairFromString(instruments.Instruments[i].Symbol)
			if err != nil {
				return nil, err
			}
			resp = append(resp, pair)
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	// A failing asset, such as spot while maintenance takes every pair offline, does not stop the others from updating
	var errs error
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(ctx, a)
		if err == nil {
			err = e.UpdatePairs(pairs, a, false)
		}
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", a, err))
		}
	}
	// A pair is still enabled from whatever did update, but once an asset has failed its error is the one returned: with
	// nothing fetched there may be no pair to enable at all
	if err := e.EnsureOnePairEnabled(); err != nil && errs == nil {
		return err
	}
	return errs
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, a asset.Item) error {
	var prices []ticker.Price
	switch a {
	case asset.Spot:
		if err := e.ensureAssetNames(ctx); err != nil {
			return err
		}
		tickers, err := e.GetTickerInformation(ctx, &TickerInformationRequest{DisplayNames: true})
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(tickers))
		for symbol := range tickers {
			t := tickers[symbol]
			pair, err := e.pairFromDisplaySymbol(symbol)
			if err != nil {
				return err
			}
			if ok, err := e.CurrencyPairs.IsPairAvailable(pair, a); err != nil || !ok {
				continue
			}
			prices = append(prices, ticker.Price{
				Last:                       t.LastTradeClosed.Price.Float64(),
				LastSize:                   t.LastTradeClosed.LotVolume.Float64(),
				VolumeWeightedAveragePrice: t.VolumeWeightedAveragePrice.Last24Hours.Float64(),
				High:                       t.High.Last24Hours.Float64(),
				Low:                        t.Low.Last24Hours.Float64(),
				Bid:                        t.Bid.Price.Float64(),
				BidSize:                    t.Bid.LotVolume.Float64(),
				Ask:                        t.Ask.Price.Float64(),
				AskSize:                    t.Ask.LotVolume.Float64(),
				BaseVolume:                 t.Volume.Last24Hours.Float64(),
				Open:                       t.OpeningPrice.Float64(),
				Pair:                       pair,
				ExchangeName:               e.Name,
				AssetType:                  a,
			})
		}
	case asset.Futures:
		tickers, err := e.GetFuturesTickers(ctx, nil)
		if err != nil {
			return err
		}
		prices = make([]ticker.Price, 0, len(tickers.Tickers))
		for i := range tickers.Tickers {
			t := &tickers.Tickers[i]
			pair, err := e.MatchSymbolWithAvailablePairs(t.Symbol, a, true)
			if err != nil {
				if errors.Is(err, currency.ErrPairNotFound) {
					continue
				}
				return err
			}
			baseVolume, quoteVolume := futuresTickerVolumes(t.Symbol, t.Volume24Hour, t.QuoteVolume24Hour)
			prices = append(prices, ticker.Price{
				Last:                       t.Last,
				LastSize:                   t.LastSize,
				VolumeWeightedAveragePrice: t.VolumeWeightedAveragePrice24Hour,
				High:                       t.High24Hour,
				Low:                        t.Low24Hour,
				Bid:                        t.Bid,
				BidSize:                    t.BidSize,
				Ask:                        t.Ask,
				AskSize:                    t.AskSize,
				BaseVolume:                 baseVolume,
				QuoteVolume:                quoteVolume,
				Open:                       t.Open24Hour,
				PercentChange24Hour:        t.Change24Hour,
				OpenInterest:               t.OpenInterest,
				MarkPrice:                  t.MarkPrice,
				IndexPrice:                 t.IndexPrice,
				Pair:                       pair,
				ExchangeName:               e.Name,
				AssetType:                  a,
				LastUpdated:                t.LastTime,
			})
		}
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	// A ticker the store rejects, such as a crossed book's, does not stop the others from being stored
	_, err := ticker.ProcessBatch(prices)
	return err
}

// futuresTickerVolumes maps a futures ticker's two volume figures onto base and quote volume. The quote volume is in
// the quote currency on every contract, while the volume is in the base currency only on a linear contract: an inverse
// contract is worth one unit of its quote currency, so Kraken reports its volume as the quote volume and no base volume
func futuresTickerVolumes(symbol string, volume, quoteVolume float64) (base, quote float64) {
	if inverseFuturesSymbol(symbol) {
		return 0, quoteVolume
	}
	return volume, quoteVolume
}

// inverseFuturesSymbol reports whether a futures symbol names an inverse contract. Kraken prefixes inverse perpetuals
// with PI and inverse fixed maturity futures with FI, and the ticker carries no contract type of its own
func inverseFuturesSymbol(symbol string) bool {
	prefix, _, _ := strings.Cut(strings.ToUpper(symbol), "_")
	return prefix == "PI" || prefix == "FI"
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := e.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := e.CurrencyPairs.IsAssetEnabled(a); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	switch a {
	case asset.Spot:
		books, err := e.GetOrderBook(ctx, &OrderBookRequest{Pair: p, Count: 500, DisplayNames: true})
		if err != nil {
			return nil, err
		}
		// A request for one pair holds one book, keyed by the pair's display name. A level's timestamp is when it last
		// changed, so the latest is when the book last changed
		for _, b := range books {
			book.Bids = make(orderbook.Levels, len(b.Bids))
			for i := range b.Bids {
				book.Bids[i] = orderbook.Level{Price: b.Bids[i].Price.Float64(), Amount: b.Bids[i].Volume.Float64()}
				if t := b.Bids[i].Timestamp.Time(); t.After(book.LastUpdated) {
					book.LastUpdated = t
				}
			}
			book.Asks = make(orderbook.Levels, len(b.Asks))
			for i := range b.Asks {
				book.Asks[i] = orderbook.Level{Price: b.Asks[i].Price.Float64(), Amount: b.Asks[i].Volume.Float64()}
				if t := b.Asks[i].Timestamp.Time(); t.After(book.LastUpdated) {
					book.LastUpdated = t
				}
			}
		}
	case asset.Futures:
		resp, err := e.GetFuturesOrderbook(ctx, p)
		if err != nil {
			return nil, err
		}
		book.Bids = make(orderbook.Levels, len(resp.OrderBook.Bids))
		for i := range resp.OrderBook.Bids {
			book.Bids[i] = orderbook.Level{Price: resp.OrderBook.Bids[i].Price, Amount: resp.OrderBook.Bids[i].Size}
		}
		book.Asks = make(orderbook.Levels, len(resp.OrderBook.Asks))
		for i := range resp.OrderBook.Asks {
			book.Asks[i] = orderbook.Level{Price: resp.OrderBook.Asks[i].Price, Amount: resp.OrderBook.Asks[i].Size}
		}
		// Kraken sends bids in ascending price order
		book.Bids.SortBids()
		book.Asks.SortAsks()
		book.LastUpdated = resp.ServerTime
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, p, a)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, a asset.Item) (accounts.SubAccounts, error) {
	var subAccts accounts.SubAccounts
	switch a {
	case asset.Spot:
		if err := e.ensureAssetNames(ctx); err != nil {
			return nil, err
		}
		balances, err := e.GetExtendedBalance(ctx, nil)
		if err != nil {
			return nil, err
		}
		subAcct := accounts.NewSubAccount(a, "")
		for name, b := range balances {
			total, hold, creditUsed := b.Balance.Float64(), b.HoldTrade.Float64(), b.CreditUsed.Float64()
			subAcct.Balances.Set(e.assetCode(name), accounts.Balance{
				Total: total,
				Hold:  hold,
				// Kraken's balance available for trading includes the unused part of a credit line
				Free:                   max(total+b.Credit.Float64()-creditUsed-hold, 0),
				AvailableWithoutBorrow: max(total-hold, 0),
				Borrowed:               creditUsed,
			})
		}
		subAccts = accounts.SubAccounts{subAcct}
	case asset.Futures:
		resp, err := e.GetFuturesAccounts(ctx)
		if err != nil {
			return nil, err
		}
		// Cash and margin accounts key balances by lowercase codes, such as xbt
		cash := accounts.NewSubAccount(a, "cash")
		for code, amount := range resp.Accounts.Cash.Balances {
			cash.Balances.Set(currency.NewCode(code).Upper(), accounts.Balance{Total: amount.Float64(), Free: amount.Float64()})
		}
		// An emptied wallet is saved too, so that the balances it held are cleared
		flex := accounts.NewSubAccount(a, "flex")
		for code, c := range resp.Accounts.Flex.Currencies {
			free := min(max(c.Available, 0), c.Quantity)
			flex.Balances.Set(currency.NewCode(code).Upper(), accounts.Balance{Total: c.Quantity, Hold: c.Quantity - free, Free: free})
		}
		subAccts = append(subAccts, cash, flex)
		for name := range resp.Accounts.MarginAccounts {
			m := resp.Accounts.MarginAccounts[name]
			margin := accounts.NewSubAccount(a, name)
			for code, amount := range m.Balances {
				// Contract symbols, such as FI_XBTUSD_171215, key positions rather than balances
				if strings.Contains(code, "_") {
					continue
				}
				c, total := currency.NewCode(code).Upper(), amount.Float64()
				free := total
				// Available funds are in the account's currency, the only one it holds as margin
				if c.Equal(m.Currency) {
					free = min(max(m.Auxiliary.AvailableFunds, 0), total)
				}
				margin.Balances.Set(c, accounts.Balance{Total: total, Hold: total - free, Free: free})
			}
			subAccts = append(subAccts, margin)
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// assetCode returns the currency code of an asset Kraken names by its internal or display name, such as XBT for XXBT.
// A name the seeded names do not know is its own alternative name
func (e *Exchange) assetCode(name string) currency.Code {
	if alt, ok := e.assetNames.alternativeName(name); ok {
		return currency.NewCode(alt)
	}
	return currency.NewCode(name)
}

// GetAccountFundingHistory returns the recent deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	if err := e.ensureAssetNames(ctx); err != nil {
		return nil, err
	}
	deposits, err := e.GetRecentDepositsStatus(ctx, nil)
	if err != nil {
		return nil, err
	}
	withdrawals, err := e.GetRecentWithdrawalsStatus(ctx, nil)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, 0, len(deposits.Deposits)+len(withdrawals.Withdrawals))
	for i := range deposits.Deposits {
		d := &deposits.Deposits[i]
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          d.Status,
			TransferID:      d.ReferenceID,
			Description:     d.Method,
			Timestamp:       d.Time.Time(),
			Currency:        e.assetCode(d.Asset.String()).String(),
			Amount:          d.Amount.Float64(),
			Fee:             d.Fee.Float64(),
			TransferType:    "deposit",
			CryptoToAddress: d.Information,
			CryptoTxID:      d.TransactionID,
		})
	}
	for i := range withdrawals.Withdrawals {
		w := &withdrawals.Withdrawals[i]
		resp = append(resp, exchange.FundingHistory{
			ExchangeName:    e.Name,
			Status:          w.Status,
			TransferID:      w.ReferenceID,
			Description:     w.Method,
			Timestamp:       w.Time.Time(),
			Currency:        e.assetCode(w.Asset.String()).String(),
			Amount:          w.Amount.Float64(),
			Fee:             w.Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: w.Information,
			CryptoTxID:      w.TransactionID,
			CryptoChain:     w.Network,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns the recent withdrawals of a currency, or of every currency when it is empty
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	if err := e.ensureAssetNames(ctx); err != nil {
		return nil, err
	}
	withdrawals, err := e.GetRecentWithdrawalsStatus(ctx, &RecentTransfersStatusRequest{Asset: c})
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals.Withdrawals))
	for i := range withdrawals.Withdrawals {
		w := &withdrawals.Withdrawals[i]
		resp[i] = exchange.WithdrawalHistory{
			Status:          w.Status,
			TransferID:      w.ReferenceID,
			Description:     w.Method,
			Timestamp:       w.Time.Time(),
			Currency:        e.assetCode(w.Asset.String()).String(),
			Amount:          w.Amount.Float64(),
			Fee:             w.Fee.Float64(),
			TransferType:    "withdrawal",
			CryptoToAddress: w.Information,
			CryptoTxID:      w.TransactionID,
			CryptoChain:     w.Network,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, a asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	switch a {
	case asset.Spot:
		trades, err := e.GetTrades(ctx, &RecentTradesRequest{Pair: p, DisplayNames: true})
		if err != nil {
			return nil, err
		}
		resp = e.spotTrades(trades, p)
	case asset.Futures:
		trades, err := e.GetFuturesTradeHistory(ctx, &FuturesTradeHistoryRequest{Pair: p})
		if err != nil {
			return nil, err
		}
		resp = make([]trade.Data, 0, len(trades.History))
		for i := range trades.History {
			t := &trades.History[i]
			side, err := order.StringToOrderSide(t.Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          t.UID,
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    a,
				Side:         side,
				Price:        t.Price,
				Amount:       t.Size.Float64(),
				Timestamp:    t.Time,
			})
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// spotTrades converts the trades of a request for one pair, which holds one pair's trades
func (e *Exchange) spotTrades(trades *RecentTradesResponse, p currency.Pair) []trade.Data {
	var resp []trade.Data
	for _, pairTrades := range trades.Trades {
		resp = slices.Grow(resp, len(pairTrades))
		for i := range pairTrades {
			side := order.Buy
			if pairTrades[i].Side == "s" {
				side = order.Sell
			}
			resp = append(resp, trade.Data{
				TID:          strconv.FormatUint(pairTrades[i].TradeID, 10),
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    asset.Spot,
				Side:         side,
				Price:        pairTrades[i].Price.Float64(),
				Amount:       pairTrades[i].Volume.Float64(),
				Timestamp:    pairTrades[i].Time.Time(),
			})
		}
	}
	return resp
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(ctx context.Context, p currency.Pair, a asset.Item, from, to time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(from, to); err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch a {
	case asset.Spot:
		since := from
		var lastTradeID uint64
		for {
			trades, err := e.GetTrades(ctx, &RecentTradesRequest{Pair: p, Since: since, DisplayNames: true})
			if err != nil {
				return nil, err
			}
			// Since is inclusive, so each page repeats the trade the previous page ended on
			for name, pairTrades := range trades.Trades {
				pairTrades = slices.DeleteFunc(pairTrades, func(t RecentTrade) bool { return t.TradeID <= lastTradeID })
				if len(pairTrades) != 0 {
					lastTradeID = pairTrades[len(pairTrades)-1].TradeID
				}
				trades.Trades[name] = pairTrades
			}
			page := e.spotTrades(trades, p)
			for i := range page {
				if !page[i].Timestamp.Before(from) && !page[i].Timestamp.After(to) {
					resp = append(resp, page[i])
				}
			}
			next := trades.Last.Time()
			// The cursor stops advancing once every trade since it has been returned
			if len(page) == 0 || !next.After(since) || next.After(to) {
				break
			}
			since = next
		}
	case asset.Futures:
		req := &FuturesHistoryMarketEventsRequest{Pair: p, Since: from, Before: to, Ascending: true}
		for {
			events, err := e.GetFuturesPublicExecutionEvents(ctx, req)
			if err != nil {
				return nil, err
			}
			for i := range events.Elements {
				x := &events.Elements[i].Event.Execution.Execution
				side, err := order.StringToOrderSide(x.TakerOrder.Direction)
				// Kraken sends Unknown for a direction it could not decode, which leaves the side unknown
				if err != nil && x.TakerOrder.Direction != "Unknown" {
					return nil, err
				}
				resp = append(resp, trade.Data{
					TID:          x.UID,
					Exchange:     e.Name,
					CurrencyPair: p,
					AssetType:    a,
					Side:         side,
					Price:        x.Price.Float64(),
					Amount:       x.Quantity.Float64(),
					Timestamp:    x.Timestamp.Time(),
				})
			}
			if events.ContinuationToken == "" || events.ContinuationToken == req.ContinuationToken {
				break
			}
			req.ContinuationToken = events.ContinuationToken
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	if err := e.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	trade.SortByDate(resp)
	return resp, nil
}

// privateWebsocketAvailable reports whether spot orders can be placed over the private websocket connection, which
// connects only with private subscriptions
func (e *Exchange) privateWebsocketAvailable() bool {
	if !e.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		return false
	}
	_, err := e.Websocket.GetConnection(wsPrivateConnection)
	return err == nil
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		return e.submitSpotOrder(ctx, s)
	case asset.Futures:
		return e.submitFuturesOrder(ctx, s)
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, s.AssetType)
	}
}

// submitSpotOrder places a spot order over the private websocket connection when it is available, and REST otherwise
func (e *Exchange) submitSpotOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	orderType, err := spotOrderTypeString(s.Type)
	if err != nil {
		return nil, err
	}
	tif, err := spotTimeInForceString(s.TimeInForce)
	if err != nil {
		return nil, err
	}
	trigger, err := spotTriggerReference(s.TriggerPriceType)
	if err != nil {
		return nil, err
	}
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	if s.Type == order.TrailingStop && s.TrackingMode != order.Percentage && s.TrackingMode != order.Distance {
		return nil, fmt.Errorf("%w: %v", order.ErrUnknownTrackingMode, s.TrackingMode)
	}
	// Add Order takes a quote quantity for market buys alone, while websocket v2 requires a base quantity, so a market
	// buy sized in the quote currency goes over REST
	quoteSized := s.Type == order.Market && s.Side.IsLong() && s.QuoteAmount > 0
	if !quoteSized && e.privateWebsocketAvailable() {
		o := WsOrder{
			OrderType:     orderType,
			Side:          side,
			Quantity:      s.Amount,
			TimeInForce:   strings.ToLower(tif),
			PostOnly:      s.TimeInForce.Is(order.PostOnly),
			ReduceOnly:    s.ReduceOnly,
			ClientOrderID: s.ClientOrderID,
		}
		if tif == timeInForceGTD {
			o.ExpireTime = s.EndTime
		}
		switch {
		case s.Type == order.TrailingStop:
			o.Triggers = &WsOrderTriggers{Reference: trigger, Price: s.TrackingValue, PriceType: "quote"}
			if s.TrackingMode == order.Percentage {
				o.Triggers.PriceType = "pct"
			}
		case s.Type.Is(order.Stop) || s.Type.Is(order.TakeProfit):
			o.Triggers = &WsOrderTriggers{Reference: trigger, Price: s.TriggerPrice}
		}
		if s.Type.Is(order.Limit) {
			o.LimitPrice = s.Price
		}
		resp, err := e.WsAddOrder(ctx, &WsAddOrderRequest{Pair: s.Pair, Order: o})
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(resp.OrderID)
	}
	o := OrderParameters{
		OrderType:     orderType,
		Side:          side,
		Volume:        s.Amount,
		TimeInForce:   tif,
		PostOnly:      s.TimeInForce.Is(order.PostOnly),
		ReduceOnly:    s.ReduceOnly,
		ClientOrderID: s.ClientOrderID,
	}
	if tif == timeInForceGTD {
		o.ExpireTime = s.EndTime
	}
	if quoteSized {
		o.Volume, o.VolumeInQuote = s.QuoteAmount, true
	}
	switch {
	case s.Type == order.TrailingStop:
		o.Price = OrderPrice{Value: s.TrackingValue, Offset: "+", Percent: s.TrackingMode == order.Percentage}
		o.TriggerSignal = trigger
	case s.Type.Is(order.Stop) || s.Type.Is(order.TakeProfit):
		// Triggered orders take the trigger price as their price and a limit as their secondary price
		o.Price = OrderPrice{Value: s.TriggerPrice}
		o.TriggerSignal = trigger
		if s.Type.Is(order.Limit) {
			o.SecondaryPrice = OrderPrice{Value: s.Price}
		}
	case s.Type == order.Limit:
		o.Price = OrderPrice{Value: s.Price}
	}
	resp, err := e.AddOrder(ctx, &AddOrderRequest{Pair: s.Pair, Order: o})
	if err != nil {
		return nil, err
	}
	if len(resp.TransactionIDs) == 0 {
		return nil, fmt.Errorf("%w: no transaction ID returned", order.ErrPlaceFailed)
	}
	return s.DeriveSubmitResponse(resp.TransactionIDs[0])
}

// orderSideString returns the side Kraken takes, buy or sell, for an order's direction
func orderSideString(s order.Side) (string, error) {
	switch {
	case s.IsLong():
		return "buy", nil
	case s.IsShort():
		return "sell", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, s)
	}
}

// spotOrderTypeString returns the Kraken order type of a spot order type
func spotOrderTypeString(t order.Type) (string, error) {
	switch t {
	case order.Market:
		return "market", nil
	case order.Limit:
		return "limit", nil
	case order.Stop, order.StopMarket:
		return "stop-loss", nil
	case order.StopLimit:
		return "stop-loss-limit", nil
	case order.TakeProfit, order.TakeProfitMarket:
		return "take-profit", nil
	case order.TakeProfit | order.Limit:
		return "take-profit-limit", nil
	case order.TrailingStop:
		return "trailing-stop", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
}

// spotTimeInForceString returns the Kraken time in force of a time in force, which post-only does not affect
func spotTimeInForceString(tif order.TimeInForce) (string, error) {
	switch {
	case tif.Is(order.ImmediateOrCancel):
		return timeInForceIOC, nil
	case tif.Is(order.FillOrKill):
		return timeInForceFOK, nil
	case tif.Is(order.GoodTillTime):
		return timeInForceGTD, nil
	case tif.Is(order.GoodTillCancel), tif == order.UnknownTIF, tif == order.PostOnly:
		return "", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
	}
}

// spotTriggerReference returns the price a triggered spot order follows: last, Kraken's default, or index
func spotTriggerReference(p order.PriceType) (string, error) {
	switch p {
	case order.UnknownPriceType:
		return "", nil
	case order.LastPrice:
		return "last", nil
	case order.IndexPrice:
		return "index", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnknownPriceType, p)
	}
}

// submitFuturesOrder places a futures order
func (e *Exchange) submitFuturesOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	side, err := orderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	signal, err := futuresTriggerSignal(s.TriggerPriceType)
	if err != nil {
		return nil, err
	}
	req := &FuturesSendOrderRequest{
		Symbol:        s.Pair,
		Side:          side,
		Size:          s.Amount,
		ClientOrderID: s.ClientOrderID,
		ReduceOnly:    s.ReduceOnly,
	}
	if req.OrderType, err = futuresOrderTypeString(s.Type, s.TimeInForce); err != nil {
		return nil, err
	}
	switch req.OrderType {
	case "stp", "take_profit":
		req.StopPrice, req.TriggerSignal = s.TriggerPrice, signal
		if s.Type.Is(order.Limit) {
			req.LimitPrice = s.Price
		}
	case "trailing_stop":
		req.TriggerSignal = signal
		req.TrailingStopMaximumDeviation = s.TrackingValue
		switch s.TrackingMode {
		case order.Percentage:
			req.TrailingStopDeviationUnit = "PERCENT"
		case order.Distance:
			req.TrailingStopDeviationUnit = "QUOTE_CURRENCY"
		default:
			return nil, fmt.Errorf("%w: %v", order.ErrUnknownTrackingMode, s.TrackingMode)
		}
	case "mkt":
	default:
		req.LimitPrice = s.Price
	}
	resp, err := e.SendFuturesOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.SendStatus.Status != "placed" {
		return nil, fmt.Errorf("%w: %s", order.ErrPlaceFailed, resp.SendStatus.Status)
	}
	return s.DeriveSubmitResponse(resp.SendStatus.OrderID)
}

// futuresOrderTypeString returns the Kraken order type of a futures order, whose limit orders carry their time in
// force in their type
func futuresOrderTypeString(t order.Type, tif order.TimeInForce) (string, error) {
	if t == order.Limit {
		switch tif {
		case order.UnknownTIF, order.GoodTillCancel:
			return "lmt", nil
		case order.PostOnly, order.GoodTillCancel | order.PostOnly:
			return "post", nil
		case order.ImmediateOrCancel:
			return "ioc", nil
		case order.FillOrKill:
			return "fok", nil
		default:
			return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
		}
	}
	var orderType string
	switch t {
	case order.Market:
		orderType = "mkt"
	case order.Stop, order.StopMarket, order.StopLimit:
		orderType = "stp"
	case order.TakeProfit, order.TakeProfitMarket, order.TakeProfit | order.Limit:
		orderType = "take_profit"
	case order.TrailingStop:
		orderType = "trailing_stop"
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedOrderType, t)
	}
	// The other orders rest until they are cancelled, except a market order, which is immediate or cancel
	if tif != order.UnknownTIF && tif != order.GoodTillCancel && (t != order.Market || tif != order.ImmediateOrCancel) {
		return "", fmt.Errorf("%w: %s", order.ErrUnsupportedTimeInForce, tif)
	}
	return orderType, nil
}

// futuresTriggerSignal returns the price a triggered futures order follows
func futuresTriggerSignal(p order.PriceType) (string, error) {
	switch p {
	case order.UnknownPriceType:
		return "", nil
	case order.LastPrice:
		return "last", nil
	case order.MarkPrice:
		return "mark", nil
	case order.IndexPrice:
		return "index", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrUnknownPriceType, p)
	}
}

// ModifyOrder changes an order's quantity and prices. A spot order is amended in place, keeping its ID and, where
// possible, its queue priority
func (e *Exchange) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	switch action.AssetType {
	case asset.Spot:
		if e.privateWebsocketAvailable() {
			if _, err := e.WsAmendOrder(ctx, &WsAmendOrderRequest{
				OrderID:       action.OrderID,
				ClientOrderID: clientOrderIDWithoutOrderID(action.OrderID, action.ClientOrderID),
				Quantity:      action.Amount,
				LimitPrice:    action.Price,
				TriggerPrice:  action.TriggerPrice,
			}); err != nil {
				return nil, err
			}
			break
		}
		req := &AmendOrderRequest{
			TransactionID: action.OrderID,
			ClientOrderID: clientOrderIDWithoutOrderID(action.OrderID, action.ClientOrderID),
			OrderQuantity: action.Amount,
		}
		if action.Price != 0 {
			req.LimitPrice = OrderPrice{Value: action.Price}
		}
		if action.TriggerPrice != 0 {
			req.TriggerPrice = OrderPrice{Value: action.TriggerPrice}
		}
		if _, err := e.AmendOrder(ctx, req); err != nil {
			return nil, err
		}
	case asset.Futures:
		req := &FuturesEditOrderRequest{
			OrderID:       action.OrderID,
			ClientOrderID: clientOrderIDWithoutOrderID(action.OrderID, action.ClientOrderID),
			Size:          action.Amount,
			LimitPrice:    action.Price,
			StopPrice:     action.TriggerPrice,
		}
		// The amount is the order's total size, past fills included, as a spot amend's is, where Kraken's default
		// relative mode would set its open size
		if action.Amount != 0 {
			req.QuantityMode = "ABSOLUTE"
		}
		resp, err := e.EditFuturesOrder(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp.EditStatus.Status != "edited" {
			return nil, fmt.Errorf("%w: %s", errOrderEditFailed, resp.EditStatus.Status)
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, action.AssetType)
	}
	return action.DeriveModifyResponse()
}

// clientOrderIDWithoutOrderID returns the client order ID when there is no order ID, as Kraken takes one identifier
func clientOrderIDWithoutOrderID(orderID, clientOrderID string) string {
	if orderID != "" {
		return ""
	}
	return clientOrderID
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(); err != nil {
		return err
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	switch o.AssetType {
	case asset.Spot:
		if e.privateWebsocketAvailable() {
			req := &WsCancelOrdersRequest{OrderIDs: []string{o.OrderID}}
			if o.OrderID == "" {
				req = &WsCancelOrdersRequest{ClientOrderIDs: []string{o.ClientOrderID}}
			}
			resp, err := e.WsCancelOrders(ctx, req)
			if err != nil {
				return err
			}
			return resp[0].Error
		}
		_, err := e.CancelExistingOrder(ctx, &CancelExistingOrderRequest{
			TransactionID: o.OrderID,
			ClientOrderID: clientOrderIDWithoutOrderID(o.OrderID, o.ClientOrderID),
		})
		return err
	case asset.Futures:
		resp, err := e.CancelFuturesOrder(ctx, &FuturesCancelOrderRequest{
			OrderID:       o.OrderID,
			ClientOrderID: clientOrderIDWithoutOrderID(o.OrderID, o.ClientOrderID),
		})
		if err != nil {
			return err
		}
		return futuresCancelStatusError(resp.CancelStatus.Status)
	default:
		return fmt.Errorf("%w %q", asset.ErrNotSupported, o.AssetType)
	}
}

// futuresCancelStatusError returns the error a futures cancellation status reports, or nil once it is cancelled
func futuresCancelStatusError(status string) error {
	switch status {
	case "cancelled":
		return nil
	case "notFound":
		return order.ErrOrderNotFound
	default:
		return fmt.Errorf("%w: %s", order.ErrCancelFailed, status)
	}
}

// CancelBatchOrders cancels orders by their IDs, or client order IDs when they have no ID
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	resp := &order.CancelBatchResponse{Status: make(map[string]string, len(o))}
	var spotIDs, futuresIDs []string
	for i := range o {
		if err := o[i].Validate(); err != nil {
			return nil, err
		}
		id := o[i].OrderID
		if id == "" {
			if o[i].ClientOrderID != "" {
				return nil, fmt.Errorf("%w: batch cancellation takes order IDs", order.ErrOrderIDNotSet)
			}
			return nil, order.ErrOrderIDNotSet
		}
		switch o[i].AssetType {
		case asset.Spot:
			spotIDs = append(spotIDs, id)
		case asset.Futures:
			futuresIDs = append(futuresIDs, id)
		default:
			return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, o[i].AssetType)
		}
	}
	if len(spotIDs) != 0 {
		if err := e.cancelSpotOrders(ctx, spotIDs, resp.Status); err != nil {
			return nil, err
		}
	}
	for chunk := range slices.Chunk(futuresIDs, 500) {
		instructions := make([]FuturesBatchInstruction, len(chunk))
		for i := range chunk {
			instructions[i] = FuturesBatchInstruction{Cancel: &FuturesBatchCancelInstruction{OrderID: chunk[i]}}
			// An order Kraken reports no outcome for is not known to be cancelled
			resp.Status[chunk[i]] = fmt.Errorf("%w: no status returned", order.ErrCancelFailed).Error()
		}
		batch, err := e.FuturesBatchOrder(ctx, &FuturesBatchOrderRequest{Instructions: instructions})
		if err != nil {
			return nil, err
		}
		for i := range batch.BatchStatus {
			s := &batch.BatchStatus[i]
			if s.OrderID == "" {
				continue
			}
			resp.Status[s.OrderID] = order.Cancelled.String()
			if err := futuresCancelStatusError(s.Status); err != nil {
				resp.Status[s.OrderID] = err.Error()
			}
		}
	}
	return resp, nil
}

// cancelSpotOrders cancels spot orders by ID, recording each one's outcome in status: cancelled, or its error. The
// private websocket connection reports each order's outcome; Cancel Order Batch reports only how many it cancelled, so
// the orders of a batch it did not cancel in full are each looked up, and cancelled on their own if still open
func (e *Exchange) cancelSpotOrders(ctx context.Context, ids []string, status map[string]string) error {
	if e.privateWebsocketAvailable() {
		for chunk := range slices.Chunk(ids, 50) {
			results, err := e.WsCancelOrders(ctx, &WsCancelOrdersRequest{OrderIDs: chunk})
			if err != nil {
				return err
			}
			for i := range chunk {
				status[chunk[i]] = order.Cancelled.String()
				if i < len(results) && results[i].Error != nil {
					status[chunk[i]] = results[i].Error.Error()
				}
			}
		}
		return nil
	}
	for chunk := range slices.Chunk(ids, 50) {
		if len(chunk) == 1 {
			status[chunk[0]] = e.cancelSpotOrder(ctx, chunk[0])
			continue
		}
		resp, err := e.CancelOrderBatch(ctx, &CancelOrderBatchRequest{TransactionIDs: chunk})
		if err != nil {
			return err
		}
		for _, id := range chunk {
			if resp.Count == uint64(len(chunk)) {
				status[id] = order.Cancelled.String()
				continue
			}
			// Kraken does not know an order the batch cancelled to cancel it again
			orders, err := e.QueryOrdersInfo(ctx, &QueryOrdersRequest{OrderIDs: []string{id}})
			switch {
			case err != nil:
				status[id] = err.Error()
			case orders[id].Status == "canceled":
				status[id] = order.Cancelled.String()
			default:
				status[id] = e.cancelSpotOrder(ctx, id)
			}
		}
	}
	return nil
}

// cancelSpotOrder cancels a spot order over REST, returning its outcome: cancelled, or its error
func (e *Exchange) cancelSpotOrder(ctx context.Context, id string) string {
	if _, err := e.CancelExistingOrder(ctx, &CancelExistingOrderRequest{TransactionID: id}); err != nil {
		return err.Error()
	}
	return order.Cancelled.String()
}

// CancelAllOrders cancels every open order of an asset, or of a pair when one is given
func (e *Exchange) CancelAllOrders(ctx context.Context, req *order.Cancel) (order.CancelAllResponse, error) {
	resp := order.CancelAllResponse{Status: make(map[string]string)}
	if err := req.Validate(); err != nil {
		return resp, err
	}
	switch req.AssetType {
	case asset.Spot:
		var ids []string
		if req.Pair.IsEmpty() {
			// Every open order is cancelled, those on pairs missing from the available pairs, such as cancel only
			// pairs, included
			open, err := e.GetOpenOrders(ctx, nil)
			if err != nil {
				return resp, err
			}
			ids = slices.Collect(maps.Keys(open.Open))
		} else {
			orders, err := e.GetActiveOrders(ctx, &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{req.Pair}})
			if err != nil {
				return resp, err
			}
			ids = make([]string, len(orders))
			for i := range orders {
				ids[i] = orders[i].OrderID
			}
		}
		return resp, e.cancelSpotOrders(ctx, ids, resp.Status)
	case asset.Futures:
		cancelled, err := e.CancelAllFuturesOrders(ctx, &FuturesCancelAllOrdersRequest{Symbol: req.Pair})
		if err != nil {
			return resp, err
		}
		for i := range cancelled.CancelStatus.CancelledOrders {
			resp.Status[cancelled.CancelStatus.CancelledOrders[i].OrderID] = order.Cancelled.String()
		}
		return resp, nil
	default:
		return resp, fmt.Errorf("%w %q", asset.ErrNotSupported, req.AssetType)
	}
}

// GetOrderInfo returns information on a current open order
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, a asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	switch a {
	case asset.Spot:
		orders, err := e.QueryOrdersInfo(ctx, &QueryOrdersRequest{OrderIDs: []string{orderID}, IncludeTrades: true})
		if err != nil {
			return nil, err
		}
		info, ok := orders[orderID]
		if !ok {
			return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
		}
		d, err := e.spotOrderDetail(orderID, &info)
		if err != nil {
			return nil, err
		}
		if d.Trades, err = e.spotOrderTrades(ctx, &info, d); err != nil {
			return nil, err
		}
		return d, nil
	case asset.Futures:
		return e.futuresOrderInfo(ctx, orderID)
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// spotOrderDetail converts a spot order. An order on a pair no longer listed returns currency.ErrPairNotFound
func (e *Exchange) spotOrderDetail(orderID string, info *OrderInfo) (*order.Detail, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(info.Description.Pair, asset.Spot, false)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(info.Description.Side)
	if err != nil {
		return nil, err
	}
	orderType, err := orderTypeFromString(info.Description.OrderType)
	if err != nil {
		return nil, err
	}
	status, err := orderStatusFromString(info.Status)
	if err != nil {
		return nil, err
	}
	executed := info.VolumeExecuted.Float64()
	switch {
	case executed > 0 && status == order.Open:
		status = order.PartiallyFilled
	case executed > 0 && status == order.Cancelled:
		status = order.PartiallyFilledCancelled
	}
	var tif order.TimeInForce
	if info.TimeInForce != "" {
		if tif, err = timeInForceFromString(info.TimeInForce); err != nil {
			return nil, err
		}
	}
	if slices.Contains(strings.Split(info.OrderFlags, ","), "post") {
		tif |= order.PostOnly
	}
	d := &order.Detail{
		Exchange:             e.Name,
		AssetType:            asset.Spot,
		OrderID:              orderID,
		ClientOrderID:        info.ClientOrderID,
		Pair:                 pair,
		Side:                 side,
		Type:                 orderType,
		Status:               status,
		TimeInForce:          tif,
		ReduceOnly:           info.ReduceOnly,
		AverageExecutedPrice: info.AveragePrice.Float64(),
		Amount:               info.Volume.Float64(),
		ExecutedAmount:       executed,
		RemainingAmount:      info.Volume.Float64() - executed,
		Cost:                 info.Cost.Float64(),
		CostAsset:            pair.Quote,
		Fee:                  info.Fee.Float64(),
		// Kraken reports fees in the quote currency, whichever currency the fcib and fciq flags charge them in
		FeeAsset:    pair.Quote,
		Date:        info.OpenTime.Time(),
		CloseTime:   info.CloseTime.Time(),
		LastUpdated: info.OpenTime.Time(),
	}
	price, secondary := info.Description.Price.absolute(), info.Description.SecondaryPrice.absolute()
	switch {
	case orderType.Is(order.Stop) || orderType.Is(order.TakeProfit) || orderType.Is(order.TrailingStop):
		// A triggered order's description prices are its trigger price and limit, which are given as offsets for an
		// order placed relative to the market, as every trailing stop is, leaving the prices the order holds now
		d.TriggerPrice, d.Price = price, secondary
		if d.TriggerPrice == 0 {
			d.TriggerPrice = info.StopPrice.Float64()
		}
		if d.Price == 0 && orderType.Is(order.Limit) {
			d.Price = info.LimitPrice.Float64()
		}
	default:
		d.Price = price
	}
	if !d.CloseTime.IsZero() {
		d.LastUpdated = d.CloseTime
	}
	return d, nil
}

// spotOrderTrades returns a spot order's fills
func (e *Exchange) spotOrderTrades(ctx context.Context, info *OrderInfo, d *order.Detail) ([]order.TradeHistory, error) {
	resp := make([]order.TradeHistory, 0, len(info.TradeIDs))
	for chunk := range slices.Chunk(info.TradeIDs, 20) {
		trades, err := e.QueryTradesInfo(ctx, &QueryTradesRequest{TradeIDs: chunk})
		if err != nil {
			return nil, err
		}
		for _, id := range chunk {
			t, ok := trades[id]
			if !ok {
				continue
			}
			resp = append(resp, order.TradeHistory{
				TID:       id,
				Price:     t.Price.Float64(),
				Amount:    t.Volume.Float64(),
				Fee:       t.Fee.Float64(),
				FeeAsset:  d.FeeAsset.String(),
				Exchange:  e.Name,
				Type:      d.Type,
				Side:      d.Side,
				Timestamp: t.Time.Time(),
				IsMaker:   t.Maker,
				Total:     t.Cost.Float64(),
			})
		}
	}
	return resp, nil
}

// futuresOrderInfo returns a futures order, from the open orders, the orders closed within the last 5 seconds, or the
// fills of an order closed before then
func (e *Exchange) futuresOrderInfo(ctx context.Context, orderID string) (*order.Detail, error) {
	open, err := e.GetFuturesOpenOrders(ctx)
	if err != nil {
		return nil, err
	}
	for i := range open.OpenOrders {
		if open.OpenOrders[i].OrderID == orderID {
			return e.futuresOpenOrderDetail(&open.OpenOrders[i])
		}
	}
	status, err := e.GetFuturesOrdersStatus(ctx, &FuturesOrdersStatusRequest{OrderIDs: []string{orderID}})
	if err != nil {
		return nil, err
	}
	for i := range status.Orders {
		if status.Orders[i].Order.OrderID == orderID {
			return e.futuresOrderStatusDetail(&status.Orders[i])
		}
	}
	fills, err := e.GetFuturesFills(ctx, nil)
	if err != nil {
		return nil, err
	}
	var d *order.Detail
	for i := range fills.Fills {
		f := &fills.Fills[i]
		if f.OrderID != orderID {
			continue
		}
		side, err := order.StringToOrderSide(f.Side)
		if err != nil {
			return nil, err
		}
		if d == nil {
			pair, err := e.MatchSymbolWithAvailablePairs(f.Symbol, asset.Futures, true)
			if err != nil {
				return nil, err
			}
			d = &order.Detail{
				Exchange:      e.Name,
				AssetType:     asset.Futures,
				OrderID:       orderID,
				ClientOrderID: f.ClientOrderID,
				Pair:          pair,
				Side:          side,
				Status:        order.Filled,
				LastUpdated:   f.FillTime,
			}
		}
		d.ExecutedAmount += f.Size
		d.Cost += f.Size * f.Price
		d.Date = f.FillTime
		d.Trades = append(d.Trades, order.TradeHistory{
			TID:       f.FillID,
			Price:     f.Price,
			Amount:    f.Size,
			Exchange:  e.Name,
			Side:      side,
			Timestamp: f.FillTime,
			IsMaker:   f.FillType == "maker",
		})
	}
	if d == nil {
		return nil, fmt.Errorf("%w: %s", order.ErrOrderNotFound, orderID)
	}
	// Fills hold no order size, so the order's size is taken as what filled
	d.Amount = d.ExecutedAmount
	d.AverageExecutedPrice = d.Cost / d.ExecutedAmount
	return d, nil
}

// futuresOpenOrderDetail converts an open futures order
func (e *Exchange) futuresOpenOrderDetail(o *FuturesOpenOrder) (*order.Detail, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(o.Symbol, asset.Futures, true)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(o.Side)
	if err != nil {
		return nil, err
	}
	orderType, tif, err := futuresOrderTypeFromString(o.OrderType)
	if err != nil {
		return nil, err
	}
	status := order.Open
	if o.FilledSize > 0 {
		status = order.PartiallyFilled
	}
	// A stop or take profit order with a limit price is a limit order once it triggers
	if (orderType == order.Stop || orderType == order.TakeProfit) && o.LimitPrice > 0 {
		orderType |= order.Limit
	}
	return &order.Detail{
		Exchange:        e.Name,
		AssetType:       asset.Futures,
		OrderID:         o.OrderID,
		ClientOrderID:   o.ClientOrderID,
		Pair:            pair,
		Side:            side,
		Type:            orderType,
		TimeInForce:     tif,
		Status:          status,
		ReduceOnly:      o.ReduceOnly,
		Price:           o.LimitPrice,
		TriggerPrice:    o.StopPrice,
		Amount:          o.FilledSize + o.UnfilledSize,
		ExecutedAmount:  o.FilledSize,
		RemainingAmount: o.UnfilledSize,
		Date:            o.ReceivedTime,
		LastUpdated:     o.LastUpdateTime,
	}, nil
}

// futuresOrderStatusDetail converts a futures order's status
func (e *Exchange) futuresOrderStatusDetail(s *FuturesOrderStatusDetails) (*order.Detail, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(s.Order.Symbol, asset.Futures, true)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(s.Order.Side)
	if err != nil {
		return nil, err
	}
	var status order.Status
	switch s.Status {
	case "ENTERED_BOOK", "TRIGGER_PLACED":
		status = order.Open
		if s.Order.FilledQuantity > 0 {
			status = order.PartiallyFilled
		}
	case "FULLY_EXECUTED":
		status = order.Filled
	case "REJECTED", "TRIGGER_ACTIVATION_FAILURE":
		status = order.Rejected
	case "CANCELLED":
		status = order.Cancelled
		if s.Order.FilledQuantity > 0 {
			status = order.PartiallyFilledCancelled
		}
	default:
		return nil, fmt.Errorf("%w: %q", errUnknownOrderStatus, s.Status)
	}
	d := &order.Detail{
		Exchange:        e.Name,
		AssetType:       asset.Futures,
		OrderID:         s.Order.OrderID,
		ClientOrderID:   s.Order.ClientOrderID,
		Pair:            pair,
		Side:            side,
		Status:          status,
		ReduceOnly:      s.Order.ReduceOnly,
		Price:           s.Order.LimitPrice,
		Amount:          s.Order.Quantity,
		ExecutedAmount:  s.Order.FilledQuantity,
		RemainingAmount: s.Order.Quantity - s.Order.FilledQuantity,
		Date:            s.Order.PlacedTime,
		LastUpdated:     s.Order.LastUpdateTime,
	}
	if s.Order.PriceTriggerOptions != nil {
		d.TriggerPrice = s.Order.PriceTriggerOptions.TriggerPrice
	}
	return d, nil
}

// futuresOrderTypeFromString converts a futures order type, whose limit orders carry their time in force in their type
func futuresOrderTypeFromString(t string) (order.Type, order.TimeInForce, error) {
	switch strings.ToLower(t) {
	case "lmt", "limit":
		return order.Limit, order.UnknownTIF, nil
	case "post":
		return order.Limit, order.PostOnly, nil
	case "ioc", "hedgeimmediateorcancel":
		return order.Limit, order.ImmediateOrCancel, nil
	case "fok", "fillorkill":
		return order.Limit, order.FillOrKill, nil
	case "mkt", "market":
		return order.Market, order.UnknownTIF, nil
	case "stp", "stop":
		return order.Stop, order.UnknownTIF, nil
	case "take_profit":
		return order.TakeProfit, order.UnknownTIF, nil
	case "trailing_stop":
		return order.TrailingStop, order.UnknownTIF, nil
	case "liquidation", "partialliquidation", "coveredliquidation":
		return order.Liquidation, order.UnknownTIF, nil
	case "assignment", "hedgeassignment", "unwind", "block", "rfq", "unknown":
		// The history API documents these, which have no GoCryptoTrader order type, and Unknown for a type it could not
		// decode
		return order.UnknownType, order.UnknownTIF, nil
	default:
		return order.UnknownType, order.UnknownTIF, fmt.Errorf("%w: %q", errUnknownOrderType, t)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	methods, err := e.GetDepositMethods(ctx, &DepositMethodsRequest{Asset: c})
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("%w for %s", errNoDepositMethods, c)
	}
	method := methods[0].Method
	if chain != "" {
		i := slices.IndexFunc(methods, func(m DepositMethod) bool { return strings.EqualFold(m.Method, chain) })
		if i == -1 {
			return nil, fmt.Errorf("%w: %s for %s", errDepositMethodUnknown, chain, c)
		}
		method = methods[i].Method
	}
	addresses, err := e.GetDepositAddresses(ctx, &DepositAddressesRequest{Asset: c, Method: method})
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		if addresses, err = e.GetDepositAddresses(ctx, &DepositAddressesRequest{Asset: c, Method: method, New: true}); err != nil {
			return nil, err
		}
		if len(addresses) == 0 {
			return nil, fmt.Errorf("%w: no %s deposit address for %s", common.ErrNoResponse, method, c)
		}
	}
	tag := addresses[0].Tag
	if tag == "" {
		tag = addresses[0].Memo
	}
	return &deposit.Address{Address: addresses[0].Address, Tag: tag, Chain: method}, nil
}

// WithdrawCryptocurrencyFunds withdraws to the withdrawal key the request's TradePassword names, as set up on the
// account, returning the withdrawal's reference ID
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, r *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.WithdrawFunds(ctx, &WithdrawFundsRequest{
		Asset:   r.Currency,
		Key:     r.TradePassword,
		Address: r.Crypto.Address,
		Amount:  r.Amount,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: resp.ReferenceID}, nil
}

// WithdrawFiatFunds withdraws to the withdrawal key the request's TradePassword names, as set up on the account,
// returning the withdrawal's reference ID
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, r *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.WithdrawFunds(ctx, &WithdrawFundsRequest{Asset: r.Currency, Key: r.TradePassword, Amount: r.Amount})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: resp.ReferenceID}, nil
}

// WithdrawFiatFundsToInternationalBank withdraws to the withdrawal key the request's TradePassword names, as set up on
// the account, returning the withdrawal's reference ID
func (e *Exchange) WithdrawFiatFundsToInternationalBank(ctx context.Context, r *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return e.WithdrawFiatFunds(ctx, r)
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, f *exchange.FeeBuilder) (float64, error) {
	if f == nil {
		return 0, fmt.Errorf("%T %w", f, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && f.FeeType == exchange.CryptocurrencyTradeFee {
		f.FeeType = exchange.OfflineTradeFee
	}
	switch f.FeeType {
	case exchange.CryptocurrencyTradeFee:
		volume, err := e.GetTradeVolume(ctx, &TradeVolumeRequest{Pairs: currency.Pairs{f.Pair}})
		if err != nil {
			return 0, err
		}
		fees := volume.Fees
		if f.IsMaker && len(volume.FeesMaker) != 0 {
			fees = volume.FeesMaker
		}
		// A request for one pair holds one pair's fee
		for _, fee := range fees {
			return fee.Fee.Float64() / 100 * f.PurchasePrice * f.Amount, nil
		}
		return 0, fmt.Errorf("%w for %s", errTradeFeeMissing, f.Pair)
	case exchange.CryptocurrencyWithdrawalFee:
		return e.withdrawalFee(ctx, f.Pair.Base, f.Amount)
	case exchange.InternationalBankWithdrawalFee:
		return e.withdrawalFee(ctx, f.FiatCurrency, f.Amount)
	case exchange.CryptocurrencyDepositFee:
		return e.depositFee(ctx, f.Pair.Base, f.Amount)
	case exchange.InternationalBankDepositFee:
		return e.depositFee(ctx, f.FiatCurrency, f.Amount)
	case exchange.OfflineTradeFee:
		return offlineTakerFee * f.PurchasePrice * f.Amount, nil
	default:
		return 0, fmt.Errorf("%w: %v", common.ErrFunctionNotSupported, f.FeeType)
	}
}

// withdrawalFee returns the fee of an asset's first withdrawal method
func (e *Exchange) withdrawalFee(ctx context.Context, c currency.Code, amount float64) (float64, error) {
	// Without an asset, Get Withdrawal Methods lists every asset's methods
	if c.IsEmpty() {
		return 0, currency.ErrCurrencyCodeEmpty
	}
	methods, err := e.GetWithdrawalMethods(ctx, &WithdrawalMethodsRequest{Asset: c})
	if err != nil {
		return 0, err
	}
	if len(methods) == 0 {
		return 0, fmt.Errorf("%w for %s", errNoWithdrawalMethods, c)
	}
	return methods[0].Fee.Fee.Float64() + methods[0].Fee.FeePercentage.Float64()/100*amount, nil
}

// depositFee returns the fee of an asset's first deposit method
func (e *Exchange) depositFee(ctx context.Context, c currency.Code, amount float64) (float64, error) {
	methods, err := e.GetDepositMethods(ctx, &DepositMethodsRequest{Asset: c})
	if err != nil {
		return 0, err
	}
	if len(methods) == 0 {
		return 0, fmt.Errorf("%w for %s", errNoDepositMethods, c)
	}
	return methods[0].Fee.Float64() + methods[0].FeePercentage.Float64()/100*amount, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		open, err := e.GetOpenOrders(ctx, nil)
		if err != nil {
			return nil, err
		}
		orders = make([]order.Detail, 0, len(open.Open))
		for id := range open.Open {
			info := open.Open[id]
			d, err := e.spotOrderDetail(id, &info)
			if err != nil {
				// The pair of an order on a pair missing from the available pairs, such as a cancel only pair, cannot be
				// named
				if errors.Is(err, currency.ErrPairNotFound) {
					continue
				}
				return nil, err
			}
			orders = append(orders, *d)
		}
	case asset.Futures:
		open, err := e.GetFuturesOpenOrders(ctx)
		if err != nil {
			return nil, err
		}
		orders = make([]order.Detail, 0, len(open.OpenOrders))
		for i := range open.OpenOrders {
			d, err := e.futuresOpenOrderDetail(&open.OpenOrders[i])
			if err != nil {
				if errors.Is(err, currency.ErrPairNotFound) {
					continue
				}
				return nil, err
			}
			orders = append(orders, *d)
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		closedReq := &ClosedOrdersRequest{Start: req.StartTime, End: req.EndTime, WithCursor: true}
		for {
			closed, err := e.GetClosedOrders(ctx, closedReq)
			if err != nil {
				return nil, err
			}
			for id := range closed.Closed {
				info := closed.Closed[id]
				d, err := e.spotOrderDetail(id, &info)
				if err != nil {
					// The pair of an order closed before it was delisted cannot be named
					if errors.Is(err, currency.ErrPairNotFound) {
						continue
					}
					return nil, err
				}
				d.InferCostsAndTimes()
				orders = append(orders, *d)
			}
			if closed.Cursor.Next == "" || closed.Cursor.Next == closedReq.Cursor {
				break
			}
			closedReq.Cursor = closed.Cursor.Next
		}
	case asset.Futures:
		var err error
		if orders, err = e.futuresOrderHistory(ctx, req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(e.Name, orders), nil
}

// futuresOrderHistory returns the futures orders whose events fall within a window, each as its last event left it
func (e *Exchange) futuresOrderHistory(ctx context.Context, start, end time.Time) ([]order.Detail, error) {
	req := &FuturesHistoryOrderEventsRequest{Since: start, Before: end, Ascending: true}
	byID := make(map[string]*order.Detail)
	var ids []string
	for {
		events, err := e.GetFuturesOrderEvents(ctx, req)
		if err != nil {
			return nil, err
		}
		for i := range events.Elements {
			ev := &events.Elements[i].Event
			var o *FuturesHistoryOrder
			var status order.Status
			switch {
			case ev.OrderPlaced != nil:
				o, status = &ev.OrderPlaced.Order, order.Open
			case ev.OrderUpdated != nil:
				o, status = &ev.OrderUpdated.NewOrder, order.Open
			case ev.OrderCancelled != nil:
				o, status = &ev.OrderCancelled.Order, order.Cancelled
			case ev.OrderRejected != nil:
				o, status = &ev.OrderRejected.Order, order.Rejected
			default:
				continue
			}
			d, err := e.futuresHistoryOrderDetail(o, status)
			if err != nil {
				if errors.Is(err, currency.ErrPairNotFound) {
					continue
				}
				return nil, err
			}
			if ev.OrderCancelled != nil {
				// A cancelled order keeps its last update from before it was cancelled
				d.LastUpdated = events.Elements[i].Timestamp.Time()
			}
			if _, ok := byID[d.OrderID]; !ok {
				ids = append(ids, d.OrderID)
			} else if d.Date.IsZero() {
				d.Date = byID[d.OrderID].Date
			}
			byID[d.OrderID] = d
		}
		if events.ContinuationToken == "" || events.ContinuationToken == req.ContinuationToken {
			break
		}
		req.ContinuationToken = events.ContinuationToken
	}
	orders := make([]order.Detail, len(ids))
	for i, id := range ids {
		orders[i] = *byID[id]
	}
	return orders, nil
}

// futuresHistoryOrderDetail converts an order as a history event reports it, with the status its event gives it
func (e *Exchange) futuresHistoryOrderDetail(o *FuturesHistoryOrder, status order.Status) (*order.Detail, error) {
	pair, err := e.MatchSymbolWithAvailablePairs(o.Tradeable, asset.Futures, true)
	if err != nil {
		return nil, err
	}
	side, err := order.StringToOrderSide(o.Direction)
	// Kraken sends Unknown for a direction it could not decode, which leaves the side unknown
	if err != nil && o.Direction != "Unknown" {
		return nil, err
	}
	orderType, tif, err := futuresOrderTypeFromString(o.OrderType)
	if err != nil {
		return nil, err
	}
	quantity, filled := o.Quantity.Float64(), o.FilledQuantity.Float64()
	switch {
	case status == order.Open && filled > 0 && filled >= quantity:
		status = order.Filled
	case status == order.Open && filled > 0:
		status = order.PartiallyFilled
	case status == order.Cancelled && filled > 0:
		status = order.PartiallyFilledCancelled
	}
	return &order.Detail{
		Exchange:        e.Name,
		AssetType:       asset.Futures,
		OrderID:         o.UID,
		ClientOrderID:   o.ClientOrderID,
		Pair:            pair,
		Side:            side,
		Type:            orderType,
		TimeInForce:     tif,
		Status:          status,
		ReduceOnly:      o.ReduceOnly,
		Price:           o.LimitPrice.Float64(),
		Amount:          quantity,
		ExecutedAmount:  filled,
		RemainingAmount: quantity - filled,
		Date:            o.Timestamp.Time(),
		LastUpdated:     o.LastUpdateTime.Time(),
	}, nil
}

// AuthenticateWebsocket fetches a new websocket token, which authenticates private websocket subscriptions and
// requests
func (e *Exchange) AuthenticateWebsocket(ctx context.Context) error {
	_, err := e.websocketToken(ctx, true)
	return err
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, a asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, a)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	// Get OHLC Data serves only the 720 most recent candles, while the futures charts serve every candle since listing
	req, err := e.GetKlineRequest(pair, a, interval, start, end, a == asset.Spot)
	if err != nil {
		return nil, err
	}
	var candles []kline.Candle
	switch a {
	case asset.Spot:
		resp, err := e.GetOHLCData(ctx, &OHLCDataRequest{
			Pair:         req.RequestFormatted,
			Interval:     req.ExchangeInterval,
			Since:        req.Start.Add(-req.ExchangeInterval.Duration()),
			DisplayNames: true,
		})
		if err != nil {
			return nil, err
		}
		for _, pairCandles := range resp.Candles {
			candles = slices.Grow(candles, len(pairCandles))
			for i := range pairCandles {
				c := &pairCandles[i]
				if t := c.Time.Time(); t.Before(req.Start) || !t.Before(req.End) {
					continue
				}
				candles = append(candles, kline.Candle{
					Time:   c.Time.Time(),
					Open:   c.Open.Float64(),
					High:   c.High.Float64(),
					Low:    c.Low.Float64(),
					Close:  c.Close.Float64(),
					Volume: c.Volume.Float64(),
				})
			}
		}
	case asset.Futures:
		if candles, err = e.futuresCandles(ctx, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(candles)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval. Spot candles are limited
// to the 720 most recent, so only futures candles extend further
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if a != asset.Futures {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		c, err := e.futuresCandles(ctx, req.RequestFormatted, req.ExchangeInterval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
		if err != nil {
			return nil, err
		}
		candles = append(candles, c...)
	}
	return req.ProcessResponse(candles)
}

// futuresCandles returns a futures market's trade candles from start until before end
func (e *Exchange) futuresCandles(ctx context.Context, pair currency.Pair, interval kline.Interval, start, end time.Time) ([]kline.Candle, error) {
	resp, err := e.GetFuturesCandles(ctx, &FuturesCandlesRequest{
		TickType: "trade",
		Pair:     pair,
		Interval: interval,
		From:     start,
		// Kraken includes a candle starting at To
		To: end.Add(-time.Second),
	})
	if err != nil {
		return nil, err
	}
	candles := make([]kline.Candle, len(resp.Candles))
	for i := range resp.Candles {
		c := &resp.Candles[i]
		candles[i] = kline.Candle{
			Time:   c.Time.Time(),
			Open:   c.Open.Float64(),
			High:   c.High.Float64(),
			Low:    c.Low.Float64(),
			Close:  c.Close.Float64(),
			Volume: c.Volume.Float64(),
		}
	}
	return candles, nil
}

// GetAvailableTransferChains returns the deposit methods of a currency, which name the chains it can be transferred on
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, c currency.Code) ([]string, error) {
	methods, err := e.GetDepositMethods(ctx, &DepositMethodsRequest{Asset: c})
	if err != nil {
		return nil, err
	}
	chains := make([]string, len(methods))
	for i := range methods {
		chains[i] = methods[i].Method
	}
	return chains, nil
}

// GetServerTime returns the current exchange server time
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	resp, err := e.GetCurrentServerTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return resp.UnixTime.Time(), nil
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, a asset.Item) ([]futures.Contract, error) {
	if !a.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if a != asset.Futures {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	instruments, err := e.GetFuturesInstruments(ctx, nil)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, 0, len(instruments.Instruments))
	for i := range instruments.Instruments {
		ins := &instruments.Instruments[i]
		name, err := currency.NewPairFromString(ins.Symbol)
		if err != nil {
			return nil, err
		}
		c := futures.Contract{
			Exchange:           e.Name,
			Name:               name,
			Underlying:         currency.NewPair(ins.Base, ins.Quote),
			Asset:              a,
			StartDate:          ins.OpeningDate,
			EndDate:            ins.LastTradingTime,
			IsActive:           ins.Tradeable && !ins.IsExpired,
			Type:               futuresContractType(ins.LastTradingTime),
			SettlementType:     futures.Linear,
			SettlementCurrency: ins.Quote,
			Multiplier:         ins.ContractSize,
		}
		if ins.Type == "futures_inverse" {
			c.SettlementType, c.SettlementCurrency = futures.Inverse, ins.Base
		}
		if levels := ins.RetailMarginLevels; len(levels) != 0 && levels[0].InitialMargin > 0 {
			c.MaxLeverage = 1 / levels[0].InitialMargin
		}
		if c.Type == futures.Perpetual && ins.MaxRelativeFundingRate != 0 {
			c.FundingRateCeiling = decimal.MustFromFloat(ins.MaxRelativeFundingRate)
			c.FundingRateFloor = decimal.MustFromFloat(-ins.MaxRelativeFundingRate)
			if ins.MinRelativeFundingRate != 0 {
				c.FundingRateFloor = decimal.MustFromFloat(ins.MinRelativeFundingRate)
			}
		}
		resp = append(resp, c)
	}
	return resp, nil
}

// futuresContractType returns a contract's type: perpetual without a last trading time, or the cycle its last trading
// day falls on. Kraken lists fixed maturity futures well before their cycles start, so a contract's life does not tell
// its cycle: quarterly futures expire on the last Friday of March, June, September and December, monthly ones on the
// last Friday of the other months and weekly ones on the other Fridays
func futuresContractType(lastTrading time.Time) futures.ContractType {
	lastTrading = lastTrading.UTC()
	switch {
	case lastTrading.IsZero():
		return futures.Perpetual
	case lastTrading.AddDate(0, 0, 7).Month() == lastTrading.Month():
		return futures.Weekly
	case lastTrading.Month()%3 == 0:
		return futures.Quarterly
	default:
		return futures.Monthly
	}
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	if !r.Pair.IsEmpty() {
		if ok, err := e.CurrencyPairs.IsPairAvailable(r.Pair, r.Asset); err != nil {
			return nil, err
		} else if !ok {
			return nil, currency.ErrPairNotContainedInAvailablePairs
		}
		if !e.isPerpetual(r.Pair) {
			return nil, fmt.Errorf("%w: %s %s", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
	}
	tickers, err := e.GetFuturesTickers(ctx, nil)
	if err != nil {
		return nil, err
	}
	// Funding is hourly: the latest rate applies from the start of the server's hour, and the predicted rate from the next
	next := tickers.ServerTime.Truncate(time.Hour).Add(time.Hour)
	resp := make([]fundingrate.LatestRateResponse, 0, len(tickers.Tickers))
	for i := range tickers.Tickers {
		t := &tickers.Tickers[i]
		pair, enabled, err := e.MatchSymbolCheckEnabled(t.Symbol, r.Asset, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if r.Pair.IsEmpty() {
			// A request for every pair reports the enabled perpetuals
			if !enabled || !e.isPerpetual(pair) {
				continue
			}
		} else if !r.Pair.Equal(pair) {
			continue
		}
		rate := fundingrate.LatestRateResponse{
			Exchange: e.Name,
			Asset:    r.Asset,
			Pair:     pair,
			// Kraken's relative rate is a fraction of the price, as funding rates are given elsewhere
			LatestRate:     fundingrate.Rate{Time: next.Add(-time.Hour), Rate: decimal.MustFromFloat(t.RelativeFundingRate)},
			TimeOfNextRate: next,
			TimeChecked:    tickers.ServerTime,
		}
		if r.IncludePredictedRate {
			rate.PredictedUpcomingRate = fundingrate.Rate{Time: next, Rate: decimal.MustFromFloat(t.RelativeFundingRatePrediction)}
		}
		resp = append(resp, rate)
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, p currency.Pair) (bool, error) {
	return a == asset.Futures && e.isPerpetual(p), nil
}

// isPerpetual reports whether a futures pair is a perpetual: PF for a linear perpetual or PI for an inverse one
func (e *Exchange) isPerpetual(p currency.Pair) bool {
	return p.Base.Equal(currency.PF) || p.Base.Equal(currency.PI)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range keys {
		if keys[i].Asset != asset.Futures {
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, keys[i].Asset, keys[i].Pair())
		}
	}
	tickers, err := e.GetFuturesTickers(ctx, nil)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(tickers.Tickers))
	for i := range tickers.Tickers {
		pair, enabled, err := e.MatchSymbolCheckEnabled(tickers.Tickers[i].Symbol, asset.Futures, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !enabled {
			continue
		}
		if len(keys) != 0 && !slices.ContainsFunc(keys, func(k key.PairAsset) bool { return k.Pair().Equal(pair) }) {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.Futures, pair),
			OpenInterest: tickers.Tickers[i].OpenInterest,
		})
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if _, err := e.CurrencyPairs.IsPairEnabled(cp, a); err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		cp.Delimiter = currency.DashDelimiter
		return tradeBaseURL + cp.Lower().String(), nil
	case asset.Futures:
		cp.Delimiter = currency.UnderscoreDelimiter
		return tradeFuturesURL + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// orderTypeFromString converts a Kraken spot order type. Kraken's iceberg order is a limit order showing part of its
// quantity, and settle-position closes a margin position at market, so they map to limit and market orders
func orderTypeFromString(t string) (order.Type, error) {
	switch t {
	case "limit", "iceberg":
		return order.Limit, nil
	case "market", "settle-position":
		return order.Market, nil
	case "stop-loss":
		return order.Stop, nil
	case "stop-loss-limit":
		return order.StopLimit, nil
	case "take-profit":
		return order.TakeProfit, nil
	case "take-profit-limit":
		return order.TakeProfit | order.Limit, nil
	case "trailing-stop":
		return order.TrailingStop, nil
	case "trailing-stop-limit":
		return order.TrailingStopLimit, nil
	default:
		return order.UnknownType, fmt.Errorf("%w: %q", errUnknownOrderType, t)
	}
}

// orderStatusFromString converts a Kraken spot order status, as the REST API and websocket v2 report it
func orderStatusFromString(s string) (order.Status, error) {
	switch s {
	case "pending", "pending_new":
		return order.Pending, nil
	case "new":
		return order.New, nil
	case "open":
		return order.Open, nil
	case "partially_filled":
		return order.PartiallyFilled, nil
	case "filled", "closed":
		return order.Filled, nil
	case "canceled":
		return order.Cancelled, nil
	case "expired":
		return order.Expired, nil
	default:
		return order.UnknownStatus, fmt.Errorf("%w: %q", errUnknownOrderStatus, s)
	}
}

// timeInForceFromString converts a Kraken spot time in force. Kraken's GTD is good till a date, which GoCryptoTrader
// calls good till time
func timeInForceFromString(tif string) (order.TimeInForce, error) {
	switch strings.ToUpper(tif) {
	case timeInForceGTC:
		return order.GoodTillCancel, nil
	case timeInForceGTD:
		return order.GoodTillTime, nil
	case timeInForceIOC:
		return order.ImmediateOrCancel, nil
	case timeInForceFOK:
		return order.FillOrKill, nil
	default:
		return order.UnknownTIF, fmt.Errorf("%w: %q", order.ErrInvalidTimeInForce, tif)
	}
}
