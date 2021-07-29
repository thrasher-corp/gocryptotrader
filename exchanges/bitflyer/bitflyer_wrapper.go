package bitflyer

import (
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

// GetDefaultConfig returns a default exchange config
func (b *Bitflyer) GetDefaultConfig() (*config.ExchangeConfig, error) {
	b.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

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

	b.Requester = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
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
func (b *Bitflyer) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	return b.SetupDefaults(exch)
}

// Start starts the Bitflyer go routine
func (b *Bitflyer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
}

// Run implements the Bitflyer wrapper
func (b *Bitflyer) Run() {
	if b.Verbose {
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitflyer) FetchTradablePairs(assetType asset.Item) ([]string, error) {
	pairs, err := b.GetMarkets()
	if err != nil {
		return nil, err
	}

	format, err := b.GetPairFormat(assetType, false)
	if err != nil {
		return nil, err
	}

	var products []string
	for i := range pairs {
		if pairs[i].Alias != "" && assetType == asset.Futures {
			products = append(products, pairs[i].Alias)
		} else if pairs[i].Alias == "" &&
			assetType == asset.Spot &&
			strings.Contains(pairs[i].ProductCode,
				format.Delimiter) {
			products = append(products, pairs[i].ProductCode)
		}
	}
	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitflyer) UpdateTradablePairs(forceUpdate bool) error {
	assets := b.CurrencyPairs.GetAssetTypes(false)
	for x := range assets {
		pairs, err := b.FetchTradablePairs(assets[x])
		if err != nil {
			return err
		}

		p, err := currency.NewPairsFromStrings(pairs)
		if err != nil {
			return err
		}

		err = b.UpdatePairs(p, assets[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitflyer) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := b.GetTicker(b.CheckFXString(fPair).String())
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
		AssetType:    assetType})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(b.Name, fPair, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitflyer) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tick, err := ticker.GetTicker(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateTicker(fPair, assetType)
	}
	return tick, nil
}

// CheckFXString upgrades currency pair if needed
func (b *Bitflyer) CheckFXString(p currency.Pair) currency.Pair {
	if strings.Contains(p.Base.String(), "FX") {
		p.Base = currency.FX_BTC
		return p
	}
	return p
}

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitflyer) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	ob, err := orderbook.Get(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateOrderbook(fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitflyer) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}

	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := b.GetOrderBook(b.CheckFXString(fPair).String())
	if err != nil {
		return book, err
	}

	for x := range orderbookNew.Asks {
		book.Asks = append(book.Asks, orderbook.Item{
			Price:  orderbookNew.Asks[x].Price,
			Amount: orderbookNew.Asks[x].Size})
	}

	for x := range orderbookNew.Bids {
		book.Bids = append(book.Bids, orderbook.Item{
			Price:  orderbookNew.Bids[x].Price,
			Amount: orderbookNew.Bids[x].Size})
	}

	err = book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(b.Name, fPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies on the
// Bitflyer exchange
func (b *Bitflyer) UpdateAccountInfo(assetType asset.Item) (account.Holdings, error) {
	return account.Holdings{}, common.ErrNotYetImplemented
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Bitflyer) FetchAccountInfo(assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(b.Name, assetType)
	if err != nil {
		return b.UpdateAccountInfo(assetType)
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitflyer) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitflyer) GetWithdrawalsHistory(c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns recent historic trades
func (b *Bitflyer) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tradeData, err := b.GetExecutionHistory(p.String())
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData {
		var timestamp time.Time
		timestamp, err = time.Parse("2006-01-02T15:04:05.999999999", tradeData[i].ExecDate)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].Side)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			TID:          strconv.FormatInt(tradeData[i].ID, 10),
			Exchange:     b.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Size,
			Timestamp:    timestamp,
		})
	}

	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bitflyer) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *Bitflyer) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	return order.SubmitResponse{}, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitflyer) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitflyer) CancelOrder(_ *order.Cancel) error {
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Bitflyer) CancelBatchOrders(_ []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitflyer) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	// TODO, implement BitFlyer API
	b.CancelAllExistingOrders()
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (b *Bitflyer) GetOrderInfo(_ string, _ currency.Pair, _ asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitflyer) GetDepositAddress(_ currency.Code, _ string) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitflyer) WithdrawCryptocurrencyFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFunds(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitflyer) WithdrawFiatFundsToInternationalBank(_ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitflyer) GetActiveOrders(_ *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitflyer) GetOrderHistory(_ *order.GetOrdersRequest) ([]order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (b *Bitflyer) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !b.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Bitflyer) ValidateCredentials(assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(assetType)
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitflyer) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *Bitflyer) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrFunctionNotSupported
}
