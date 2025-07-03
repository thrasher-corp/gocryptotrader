package bitflyer

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
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
)

// SetDefaults sets the basic defaults for Bitflyer
func (b *Bitflyer) SetDefaults() {
	b.Name = "Bitflyer"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}
	configFmt := &currency.PairFormat{
		Delimiter: currency.UnderscoreDelimiter,
		Uppercase: true,
	}
	err := b.SetGlobalPairsManager(requestFmt,
		configFmt,
		asset.Spot,
		asset.Futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: false,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
			},
			WithdrawPermissions: exchange.WithdrawCryptoViaWebsiteOnly |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      japanURL,
		exchange.ChainAnalysis: chainAnalysis,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bitflyer) Setup(exch *config.Exchange) error {
	if err := exch.Validate(); err != nil {
		return err
	}
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	return b.SetupDefaults(exch)
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitflyer) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	symbols, err := b.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(symbols))
	for i := range symbols {
		var pair currency.Pair
		if a == asset.Spot && symbols[i].MarketType == "Spot" {
			pair, err = currency.NewPairFromString(symbols[i].ProductCode)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		} else if a == asset.Futures && symbols[i].MarketType == "FX" {
			splitter := strings.Split(symbols[i].ProductCode, currency.UnderscoreDelimiter)
			if len(splitter) != 3 {
				return nil, fmt.Errorf("%w %s", errUnhandledCurrency, symbols[i].ProductCode)
			}
			pair = currency.NewPair(currency.NewCode(splitter[0]+splitter[1]), currency.NewCode(splitter[2]))
			pair.Delimiter = currency.UnderscoreDelimiter
			pairs = append(pairs, pair)
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitflyer) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := b.CurrencyPairs.GetAssetTypes(false)
	for _, a := range assets {
		pairs, err := b.FetchTradablePairs(ctx, a)
		if err != nil {
			return err
		}
		err = b.UpdatePairs(pairs, a, false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return b.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bitflyer) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (b *Bitflyer) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitflyer) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tickerNew, err := b.GetTicker(ctx, b.CheckFXString(fPair).String())
	if err != nil {
		return nil, err
	}

	err = ticker.ProcessTicker(&ticker.Price{
		Pair:         fPair,
		Ask:          tickerNew.BestAsk,
		Bid:          tickerNew.BestBid,
		Last:         tickerNew.Last,
		Volume:       tickerNew.Volume,
		ExchangeName: b.Name,
		AssetType:    a,
	})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(b.Name, fPair, a)
}

// CheckFXString upgrades currency pair if needed
func (b *Bitflyer) CheckFXString(p currency.Pair) currency.Pair {
	if strings.Contains(p.Base.String(), "FX") {
		p.Base = currency.FX_BTC
		return p
	}
	return p
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitflyer) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          b.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: b.ValidateOrderbook,
	}

	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := b.GetOrderBook(ctx, b.CheckFXString(fPair).String())
	if err != nil {
		return book, err
	}

	book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Level{
			Price:  orderbookNew.Asks[x].Price,
			Amount: orderbookNew.Asks[x].Size,
		}
	}

	book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Level{
			Price:  orderbookNew.Bids[x].Price,
			Amount: orderbookNew.Bids[x].Size,
		}
	}

	err = book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(b.Name, fPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies on the
// Bitflyer exchange
func (b *Bitflyer) UpdateAccountInfo(_ context.Context, _ asset.Item) (account.Holdings, error) {
	return account.Holdings{}, common.ErrNotYetImplemented
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitflyer) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitflyer) GetWithdrawalsHistory(_ context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns recent historic trades
func (b *Bitflyer) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tradeData, err := b.GetExecutionHistory(ctx, p.String())
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
			TID:          strconv.FormatInt(tradeData[i].ID, 10),
			Exchange:     b.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Size,
			Timestamp:    tradeData[i].ExecDate.Time(),
		}
	}

	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bitflyer) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *Bitflyer) SubmitOrder(_ context.Context, _ *order.Submit) (*order.SubmitResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitflyer) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitflyer) CancelOrder(_ context.Context, _ *order.Cancel) error {
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Bitflyer) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitflyer) CancelAllOrders(_ context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	// TODO, implement BitFlyer API
	b.CancelAllExistingOrders()
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (b *Bitflyer) GetOrderInfo(_ context.Context, _ string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitflyer) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitflyer) WithdrawCryptocurrencyFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitflyer) GetActiveOrders(_ context.Context, _ *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitflyer) GetOrderHistory(_ context.Context, _ *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (b *Bitflyer) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !b.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (b *Bitflyer) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitflyer) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *Bitflyer) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (b *Bitflyer) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (b *Bitflyer) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (b *Bitflyer) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *Bitflyer) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := b.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = ""
	return tradeBaseURL + cp.Lower().String(), nil
}
