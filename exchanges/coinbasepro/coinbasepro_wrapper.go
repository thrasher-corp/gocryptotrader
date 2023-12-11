package coinbasepro

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gofrs/uuid"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (c *CoinbasePro) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	c.SetDefaults()
	exchCfg, err := c.GetStandardConfig()
	if err != nil {
		return nil, err
	}

	err = c.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if c.Features.Supports.RESTCapabilities.AutoPairUpdates && c.Base.API.AuthenticatedSupport {
		err = c.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets default values for the exchange
func (c *CoinbasePro) SetDefaults() {
	c.Name = "CoinbasePro"
	c.Enabled = true
	c.Verbose = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresSecret = true
	// c.API.CredentialsValidator.RequiresClientID = true
	c.API.CredentialsValidator.RequiresBase64DecodeSecret = false

	requestFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := c.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	c.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				KlineFetching:     true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrders:      true,
				CancelOrder:       true,
				SubmitOrder:       true,
				DepositHistory:    true,
				WithdrawalHistory: true,
				UserTradeHistory:  true,
				CryptoDeposit:     true,
				CryptoWithdrawal:  true,
				FiatDeposit:       true,
				FiatWithdraw:      true,
				TradeFee:          true,
				FiatDepositFee:    true,
				FiatWithdrawalFee: true,
				CandleHistory:     true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				MessageSequenceNumbers: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.AutoWithdrawFiatWithAPIPermission,
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
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 300,
			},
		},
	}

	c.Requester, err = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	c.API.Endpoints = c.NewEndpoints()
	err = c.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      coinbaseAPIURL,
		exchange.RestSandbox:   coinbaseproSandboxAPIURL,
		exchange.WebsocketSpot: coinbaseproWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	c.Websocket = stream.New()
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup initialises the exchange parameters with the current configuration
func (c *CoinbasePro) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		c.SetEnabled(false)
		return nil
	}
	err = c.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := c.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = c.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            coinbaseproWebsocketURL,
		RunningURL:            wsRunningURL,
		Connector:             c.WsConnect,
		Subscriber:            c.Subscribe,
		Unsubscriber:          c.Unsubscribe,
		GenerateSubscriptions: c.GenerateDefaultSubscriptions,
		Features:              &c.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer: true,
		},
	})
	if err != nil {
		fmt.Println("COINBASE ISSUE")
		return err
	}

	return c.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the coinbasepro go routine
func (c *CoinbasePro) Start(ctx context.Context, wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		c.Run(ctx)
		wg.Done()
	}()
	return nil
}

// Run implements the coinbasepro wrapper
func (c *CoinbasePro) Run(ctx context.Context) {
	if c.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			c.Name,
			common.IsEnabled(c.Websocket.IsEnabled()),
			coinbaseproWebsocketURL)
		c.PrintEnabledPairs()
	}

	if !c.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := c.UpdateTradablePairs(ctx, false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", c.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *CoinbasePro) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	var products AllProducts
	var err error
	switch a {
	case asset.Spot:
		products, err = c.GetAllProducts(ctx, 2<<30-1, 0, "SPOT", "", nil)
	case asset.Futures:
		products, err = c.GetAllProducts(ctx, 2<<30-1, 0, "FUTURE", "", nil)
	default:
		err = asset.ErrNotSupported
	}

	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(products.Products))
	for x := range products.Products {
		if products.Products[x].TradingDisabled {
			continue
		}
		var pair currency.Pair
		pair, err = currency.NewPairDelimiter(products.Products[x].ID, currency.DashDelimiter)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (c *CoinbasePro) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := c.GetAssetTypes(true)
	for i := range assets {
		pairs, err := c.FetchTradablePairs(ctx, assets[i])
		if err != nil {
			return err
		}
		err = c.UpdatePairs(pairs, assets[i], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return c.EnsureOnePairEnabled()
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// coinbasepro exchange
func (c *CoinbasePro) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var (
		response       account.Holdings
		accountBalance []Account
		done           bool
		err            error
		cursor         string
	)
	response.Exchange = c.Name

	for !done {
		accountResp, err := c.GetAllAccounts(ctx, 250, cursor)
		if err != nil {
			return response, err
		}
		accountBalance = append(accountBalance, accountResp.Accounts...)
		done = !accountResp.HasNext
		cursor = accountResp.Cursor
	}

	accountCurrencies := make(map[string][]account.Balance)
	for i := range accountBalance {
		profileID := accountBalance[i].UUID
		currencies := accountCurrencies[profileID]
		accountCurrencies[profileID] = append(currencies, account.Balance{
			Currency: currency.NewCode(accountBalance[i].Currency),
			Total:    accountBalance[i].AvailableBalance.Value,
			Hold:     accountBalance[i].Hold.Value,
			Free: accountBalance[i].AvailableBalance.Value -
				accountBalance[i].Hold.Value,
			AvailableWithoutBorrow: accountBalance[i].AvailableBalance.Value,
			Borrowed:               0,
		})
	}

	if response.Accounts, err = account.CollectBalances(accountCurrencies, assetType); err != nil {
		return account.Holdings{}, err
	}

	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (c *CoinbasePro) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := c.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(c.Name, creds, assetType)
	fmt.Printf("Error: %v\n", err)
	if err != nil {
		return c.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// UpdateTickers updates all currency pairs of a given asset type
func (c *CoinbasePro) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	var aTString string
	switch assetType {
	case asset.Futures:
		aTString = "FUTURE"
	case asset.Spot:
		aTString = "SPOT"
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}

	products, err := c.GetAllProducts(ctx, 2<<30-1, 0, aTString, "", nil)
	if err != nil {
		return err
	}
	for x := range products.Products {
		tick, err := c.GetTicker(ctx, products.Products[x].ID, 1)
		if err != nil {
			return err
		}
		pair, err := currency.NewPairDelimiter(products.Products[x].ID, currency.DashDelimiter)
		if err != nil {
			return err
		}
		var last float64
		if len(tick.Trades) != 0 {
			last = tick.Trades[0].Price
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         last,
			Bid:          tick.BestBid.Float64(),
			Ask:          tick.BestAsk.Float64(),
			Pair:         pair,
			ExchangeName: c.Name,
			AssetType:    assetType,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := c.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tick, err := c.GetTicker(ctx, fPair.String(), 1)
	if err != nil {
		return nil, err
	}

	var last float64
	if len(tick.Trades) != 0 {
		last = tick.Trades[0].Price
	}

	tickerPrice := &ticker.Price{
		Last:         last,
		Bid:          tick.BestBid.Float64(),
		Ask:          tick.BestAsk.Float64(),
		Pair:         p,
		ExchangeName: c.Name,
		AssetType:    a}

	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}

	return ticker.GetTicker(c.Name, p, a)
}

// FetchTicker returns the ticker for a currency pair
func (c *CoinbasePro) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	p, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tickerNew, err := ticker.GetTicker(c.Name, p, assetType)
	if err != nil {
		return c.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (c *CoinbasePro) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	p, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	ob, err := orderbook.Get(c.Name, p, assetType)
	if err != nil {
		return c.UpdateOrderbook(ctx, p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *CoinbasePro) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	p, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := c.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        c.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: c.CanVerifyOrderbook,
	}
	fPair, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := c.GetProductBook(ctx, fPair.String(), 1000)
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Size,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
			Amount: orderbookNew.Asks[x].Size,
			Price:  orderbookNew.Asks[x].Price,
		}
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(c.Name, p, assetType)
}

// ProcessFundingData is a helper function for GetAccountFundingHistory and GetWithdrawalsHistory
func (c *CoinbasePro) ProcessFundingData(accHistory []DeposWithdrData, cryptoHistory []TransactionData) []exchange.FundingHistory {
	fundingData := make([]exchange.FundingHistory, len(accHistory)+len(cryptoHistory))
	for i := range accHistory {
		fundingData[i] = exchange.FundingHistory{
			ExchangeName: c.Name,
			Status:       accHistory[i].Status,
			TransferID:   accHistory[i].ID,
			Timestamp:    accHistory[i].PayoutAt,
			Currency:     accHistory[i].Amount.Currency,
			Amount:       accHistory[i].Amount.Amount,
			Fee:          accHistory[i].Fee.Amount,
			TransferType: accHistory[i].TransferType.String(),
		}
	}

	for i := range cryptoHistory {
		fundingData[i+len(accHistory)] = exchange.FundingHistory{
			ExchangeName: c.Name,
			Status:       cryptoHistory[i].Status,
			TransferID:   cryptoHistory[i].ID,
			Description:  cryptoHistory[i].Details.Title + cryptoHistory[i].Details.Subtitle,
			Timestamp:    cryptoHistory[i].CreatedAt,
			Currency:     cryptoHistory[i].Amount.Currency,
			Amount:       cryptoHistory[i].Amount.Amount,
			CryptoChain:  cryptoHistory[i].Network.Name,
		}
		if cryptoHistory[i].Type == "receive" {
			fundingData[i+len(accHistory)].TransferType = "deposit"
			fundingData[i+len(accHistory)].CryptoFromAddress = cryptoHistory[i].To.ID
		}
		if cryptoHistory[i].Type == "send" {
			fundingData[i+len(accHistory)].TransferType = "withdrawal"
			fundingData[i+len(accHistory)].CryptoToAddress = cryptoHistory[i].From.ID
		}
	}
	return fundingData
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (c *CoinbasePro) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	wallIDs, err := c.GetAllWallets(ctx, PaginationInp{})

	if err != nil {
		return nil, err
	}
	if len(wallIDs.Data) == 0 {
		return nil, errors.New("no wallets returned")
	}

	var accHistory []DeposWithdrData

	for i := range wallIDs.Data {
		tempAccHist, err := c.GetAllFiatTransfers(ctx, wallIDs.Data[i].ID, PaginationInp{}, FiatDeposit)
		if err != nil {
			return nil, err
		}
		accHistory = append(accHistory, tempAccHist.Data...)
		tempAccHist, err = c.GetAllFiatTransfers(ctx, wallIDs.Data[i].ID, PaginationInp{}, FiatWithdrawal)
		if err != nil {
			return nil, err
		}
		accHistory = append(accHistory, tempAccHist.Data...)
	}

	var cryptoHistory []TransactionData

	for i := range wallIDs.Data {
		tempCryptoHist, err := c.GetAllTransactions(ctx, wallIDs.Data[i].ID, PaginationInp{})
		if err != nil {
			return nil, err
		}
		for j := range tempCryptoHist.Data {
			if tempCryptoHist.Data[j].Type == "receive" || tempCryptoHist.Data[j].Type == "send" {
				cryptoHistory = append(cryptoHistory, tempCryptoHist.Data[j])
			}
		}
	}

	fundingData := c.ProcessFundingData(accHistory, cryptoHistory)

	return fundingData, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (c *CoinbasePro) GetWithdrawalsHistory(ctx context.Context, cur currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	tempWallIDs, err := c.GetAllWallets(ctx, PaginationInp{})

	if err != nil {
		return nil, err
	}
	if len(tempWallIDs.Data) == 0 {
		return nil, errors.New("no wallets returned")
	}

	var wallIDs []string

	for i := range tempWallIDs.Data {
		if tempWallIDs.Data[i].Currency.Code == cur.String() {
			wallIDs = append(wallIDs, tempWallIDs.Data[i].ID)
		}
	}

	if len(wallIDs) == 0 {
		return nil, errNoMatchingWallets
	}

	var accHistory []DeposWithdrData

	for i := range wallIDs {
		tempAccHist, err := c.GetAllFiatTransfers(ctx, wallIDs[i], PaginationInp{}, FiatWithdrawal)
		if err != nil {
			return nil, err
		}
		accHistory = append(accHistory, tempAccHist.Data...)
	}

	var cryptoHistory []TransactionData

	for i := range wallIDs {
		tempCryptoHist, err := c.GetAllTransactions(ctx, wallIDs[i], PaginationInp{})
		if err != nil {
			return nil, err
		}
		for j := range tempCryptoHist.Data {
			if tempCryptoHist.Data[j].Type == "send" {
				cryptoHistory = append(cryptoHistory, tempCryptoHist.Data[j])
			}
		}
	}

	tempFundingData := c.ProcessFundingData(accHistory, cryptoHistory)

	fundingData := make([]exchange.WithdrawalHistory, len(tempFundingData))

	for i := range tempFundingData {
		fundingData[i] = exchange.WithdrawalHistory{
			Status:          tempFundingData[i].Status,
			TransferID:      tempFundingData[i].TransferID,
			Description:     tempFundingData[i].Description,
			Timestamp:       tempFundingData[i].Timestamp,
			Currency:        tempFundingData[i].Currency,
			Amount:          tempFundingData[i].Amount,
			Fee:             tempFundingData[i].Fee,
			TransferType:    tempFundingData[i].TransferType,
			CryptoToAddress: tempFundingData[i].CryptoToAddress,
			CryptoTxID:      tempFundingData[i].CryptoTxID,
			CryptoChain:     tempFundingData[i].CryptoChain,
			BankTo:          tempFundingData[i].BankTo,
		}
	}

	return fundingData, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (c *CoinbasePro) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return c.GetHistoricTrades(ctx, p, assetType, time.Time{}, time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (c *CoinbasePro) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, startDate, endDate time.Time) ([]trade.Data, error) {
	p, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	statuses := []string{"FILLED", "CANCELLED", "EXPIRED", "FAILED"}

	ord, err := c.GetAllOrders(ctx, p.String(), "", "", "", "", "", "", "", statuses, 2<<30-1, startDate, endDate)

	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(ord.Orders))

	for i := range ord.Orders {
		var side order.Side
		side, err = order.StringToOrderSide(ord.Orders[i].Side)
		if err != nil {
			return nil, err
		}
		id, err := uuid.FromString(ord.Orders[i].OrderID)
		if err != nil {
			return nil, err
		}
		var price float64
		var amount float64
		if ord.Orders[i].OrderConfiguration.MarketMarketIOC != nil {
			if ord.Orders[i].OrderConfiguration.MarketMarketIOC.QuoteSize != "" {
				amount, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.MarketMarketIOC.QuoteSize, 64)
				if err != nil {
					return nil, err
				}
			}
			if ord.Orders[i].OrderConfiguration.MarketMarketIOC.BaseSize != "" {
				amount, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.MarketMarketIOC.BaseSize, 64)
				if err != nil {
					return nil, err
				}
			}
		}
		if ord.Orders[i].OrderConfiguration.LimitLimitGTC != nil {
			if ord.Orders[i].OrderConfiguration.LimitLimitGTC.LimitPrice != "" {
				price, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.LimitLimitGTC.LimitPrice, 64)
				if err != nil {
					return nil, err
				}
			}
			if ord.Orders[i].OrderConfiguration.LimitLimitGTC.BaseSize != "" {
				amount, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.LimitLimitGTC.BaseSize, 64)
				if err != nil {
					return nil, err
				}
			}
		}
		if ord.Orders[i].OrderConfiguration.LimitLimitGTD != nil {
			if ord.Orders[i].OrderConfiguration.LimitLimitGTD.LimitPrice != "" {
				price, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.LimitLimitGTD.LimitPrice, 64)
				if err != nil {
					return nil, err
				}
			}
			if ord.Orders[i].OrderConfiguration.LimitLimitGTD.BaseSize != "" {
				amount, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.LimitLimitGTD.BaseSize, 64)
				if err != nil {
					return nil, err
				}
			}
		}
		if ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTC != nil {
			if ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTC.LimitPrice != "" {
				price, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTC.LimitPrice, 64)
				if err != nil {
					return nil, err
				}
			}
			if ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTC.BaseSize != "" {
				amount, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTC.BaseSize, 64)
				if err != nil {
					return nil, err
				}
			}
		}
		if ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTD != nil {
			if ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTD.LimitPrice != "" {
				price, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTD.LimitPrice, 64)
				if err != nil {
					return nil, err
				}
			}
			if ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTD.BaseSize != "" {
				amount, err = strconv.ParseFloat(ord.Orders[i].OrderConfiguration.StopLimitStopLimitGTD.BaseSize, 64)
				if err != nil {
					return nil, err
				}
			}
		}

		resp[i] = trade.Data{
			ID:           id,
			Exchange:     c.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        price,
			Amount:       amount,
			Timestamp:    ord.Orders[i].CreatedTime,
		}
	}

	err = c.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s == nil {
		return nil, common.ErrNilPointer
	}
	err := s.Validate()
	if err != nil {
		return nil, err
	}

	fPair, err := c.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	var stopDir string

	if s.Type == order.StopLimit {
		switch s.StopDirection {
		case order.StopUp:
			stopDir = "STOP_DIRECTION_STOP_UP"
		case order.StopDown:
			stopDir = "STOP_DIRECTION_STOP_DOWN"
		}
	}

	amount := s.Amount

	if (s.Type == order.Market || s.Type == order.ImmediateOrCancel) && s.Side == order.Buy {
		amount = s.QuoteAmount
	}

	resp, err := c.PlaceOrder(ctx, s.ClientOrderID, fPair.String(), s.Side.String(), stopDir, s.Type.String(),
		amount, s.Price, s.TriggerPrice, s.PostOnly, s.EndTime)

	if err != nil {
		return nil, err
	}

	subResp, err := s.DeriveSubmitResponse(resp.OrderID)
	if err != nil {
		return nil, err
	}

	if s.RetrieveFees {
		time.Sleep(s.RetrieveFeeDelay)
		feeResp, err := c.GetOrderByID(ctx, resp.OrderID, "", s.ClientOrderID)
		if err != nil {
			return nil, err
		}
		subResp.Fee = feeResp.TotalFees
	}
	return subResp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (c *CoinbasePro) ModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	if m == nil {
		return nil, common.ErrNilPointer
	}
	err := m.Validate()
	if err != nil {
		return nil, err
	}
	success, err := c.EditOrder(ctx, m.OrderID, m.Amount, m.Price)
	if err != nil {
		return nil, err
	}
	if !success {
		return nil, errOrderModFailNoErr
	}

	return m.DeriveModifyResponse()
}

// CancelOrder cancels an order by its corresponding ID number
func (c *CoinbasePro) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if o == nil {
		return common.ErrNilPointer
	}
	err := o.Validate(o.StandardCancel())
	if err != nil {
		return err
	}
	canSlice := []order.Cancel{*o}
	resp, err := c.CancelBatchOrders(ctx, canSlice)
	if err != nil {
		return err
	}
	if resp.Status[o.OrderID] != order.Cancelled.String() {
		return fmt.Errorf("order %s failed to cancel", o.OrderID)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (c *CoinbasePro) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, errOrderIDEmpty
	}
	var status order.CancelBatchResponse
	status.Status = make(map[string]string)
	ordIDSlice := make([]string, len(o))
	for i := range o {
		err := o[i].Validate(o[i].StandardCancel())
		if err != nil {
			return nil, err
		}
		ordIDSlice[i] = o[i].OrderID
		status.Status[o[i].OrderID] = "Failed to cancel"
	}
	resp, err := c.CancelOrders(ctx, ordIDSlice)
	if err != nil {
		return nil, err
	}
	for i := range resp.Results {
		if resp.Results[i].Success {
			status.Status[resp.Results[i].OrderID] = order.Cancelled.String()
		}
	}
	return &status, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(ctx context.Context, can *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	if can == nil {
		return resp, common.ErrNilPointer
	}
	err := can.Validate(can.PairAssetRequired())
	if err != nil {
		return resp, err
	}
	var ordIDs []GetOrderResponse
	var cursor string
	ordStatus := []string{"OPEN"}
	hasNext := true
	for hasNext {
		interResp, err := c.GetAllOrders(ctx, can.Pair.String(), "", "", "", cursor, "", "", "", ordStatus, 1000,
			time.Time{}, time.Time{})
		if err != nil {
			return resp, err
		}
		ordIDs = append(ordIDs, interResp.Orders...)
		hasNext = interResp.HasNext
		cursor = interResp.Cursor
	}
	if len(ordStatus) == 0 {
		return resp, errNoMatchingOrders
	}
	var orders []order.Cancel
	for i := range ordIDs {
		orders = append(orders, order.Cancel{OrderID: ordIDs[i].OrderID})
	}

	batchResp, err := c.CancelBatchOrders(ctx, orders)
	if err != nil {
		return resp, err
	}

	resp.Status = batchResp.Status
	resp.Count = int64(len(orders))

	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (c *CoinbasePro) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, asset asset.Item) (*order.Detail, error) {
	// genOrderDetail, err := c.GetOrderByID(ctx, orderID, "", "")
	// if err != nil {
	// 	return nil, err
	// }

	// var amount float64
	// if genOrderDetail.OrderConfiguration.MarketMarketIOC != nil {
	// 	if genOrderDetail.OrderConfiguration.MarketMarketIOC.QuoteSize != "" {
	// 		amount, err = strconv.ParseFloat(genOrderDetail.OrderConfiguration.MarketMarketIOC.QuoteSize, 64)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// 	if genOrderDetail.OrderConfiguration.MarketMarketIOC.BaseSize != "" {
	// 		amount, err = strconv.ParseFloat(genOrderDetail.OrderConfiguration.MarketMarketIOC.BaseSize, 64)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }
	// var price float64
	// var postOnly bool
	// if genOrderDetail.OrderConfiguration.LimitLimitGTC != nil {

	// 	postOnly = genOrderDetail.OrderConfiguration.LimitLimitGTC.PostOnly
	// }
	// if genOrderDetail.OrderConfiguration.LimitLimitGTD != nil {
	// 	postOnly = genOrderDetail.OrderConfiguration.LimitLimitGTD.PostOnly
	// }

	// response := order.Detail{
	// 	ImmediateOrCancel: genOrderDetail.OrderConfiguration.MarketMarketIOC != nil,
	// 	PostOnly: postOnly,
	// 	Price:
	// }

	// orderStatus, err := order.StringToOrderStatus(genOrderDetail.Status)
	// if err != nil {
	// 	return nil, fmt.Errorf("error parsing order status: %w", err)
	// }
	// orderType, err := order.StringToOrderType(genOrderDetail.Type)
	// if err != nil {
	// 	return nil, fmt.Errorf("error parsing order type: %w", err)
	// }
	// orderSide, err := order.StringToOrderSide(genOrderDetail.Side)
	// if err != nil {
	// 	return nil, fmt.Errorf("error parsing order side: %w", err)
	// }
	// pair, err := currency.NewPairDelimiter(genOrderDetail.ProductID, "-")
	// if err != nil {
	// 	return nil, fmt.Errorf("error parsing order pair: %w", err)
	// }

	// response := order.Detail{
	// 	Exchange:        c.GetName(),
	// 	OrderID:         genOrderDetail.ID,
	// 	Pair:            pair,
	// 	Side:            orderSide,
	// 	Type:            orderType,
	// 	Date:            genOrderDetail.DoneAt,
	// 	Status:          orderStatus,
	// 	Price:           genOrderDetail.Price,
	// 	Amount:          genOrderDetail.Size,
	// 	ExecutedAmount:  genOrderDetail.FilledSize,
	// 	RemainingAmount: genOrderDetail.Size - genOrderDetail.FilledSize,
	// 	Fee:             genOrderDetail.FillFees,
	// }
	// fillResponse, err := c.GetFills(ctx, orderID, genOrderDetail.ProductID)
	// if err != nil {
	// 	return nil, fmt.Errorf("error retrieving the order fills: %w", err)
	// }
	// for i := range fillResponse {
	// 	var fillSide order.Side
	// 	fillSide, err = order.StringToOrderSide(fillResponse[i].Side)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("error parsing order Side: %w", err)
	// 	}
	// 	response.Trades = append(response.Trades, order.TradeHistory{
	// 		Timestamp: fillResponse[i].CreatedAt,
	// 		TID:       strconv.FormatInt(fillResponse[i].TradeID, 10),
	// 		Price:     fillResponse[i].Price,
	// 		Amount:    fillResponse[i].Size,
	// 		Exchange:  c.GetName(),
	// 		Type:      orderType,
	// 		Side:      fillSide,
	// 		Fee:       fillResponse[i].Fee,
	// 	})
	// }
	return nil, errors.New("function not properly implemented")
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *CoinbasePro) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	// resp, err := c.WithdrawCrypto(ctx,
	// 	withdrawRequest.Amount,
	// 	withdrawRequest.Currency.String(),
	// 	withdrawRequest.Crypto.Address)
	// if err != nil {
	// 	return nil, err
	// }
	// return &withdraw.ExchangeResponse{
	// 	ID: resp.ID,
	// }, err
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	// paymentMethods, err := c.GetPayMethods(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	selectedWithdrawalMethod := PaymentMethod{}
	// for i := range paymentMethods {
	// 	if withdrawRequest.Fiat.Bank.BankName == paymentMethods[i].Name {
	// 		selectedWithdrawalMethod = paymentMethods[i]
	// 		break
	// 	}
	// }
	if selectedWithdrawalMethod.ID == "" {
		return nil, fmt.Errorf("could not find payment method '%v'. Check the name via the website and try again", withdrawRequest.Fiat.Bank.BankName)
	}

	// resp, err := c.WithdrawViaPaymentMethod(ctx,
	// 	withdrawRequest.Amount,
	// 	withdrawRequest.Currency.String(),
	// 	selectedWithdrawalMethod.ID)
	// if err != nil {
	// return nil, err
	// }

	// return &withdraw.ExchangeResponse{
	// 	Status: resp.ID,
	// }, nil
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := c.WithdrawFiatFunds(ctx, withdrawRequest)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID:     v.ID,
		Status: v.Status,
	}, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	// if feeBuilder == nil {
	// 	return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	// }
	// if !c.AreCredentialsValid(ctx) && // Todo check connection status
	// 	feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
	// 	feeBuilder.FeeType = exchange.OfflineTradeFee
	// }
	// return c.GetFee(ctx, feeBuilder)
	return 99999, errors.New(common.ErrFunctionNotSupported.Error())
}

// GetActiveOrders retrieves any orders that are active/open
func (c *CoinbasePro) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var respOrders []GetOrderResponse
	// var fPair currency.Pair
	// for i := range req.Pairs {
	// 	// fPair, err = c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	var resp []GetOrderResponse
	// 	// resp, err = c.GetOrders(ctx,
	// 	// 	[]string{"open", "pending", "active"},
	// 	// 	fPair.String())
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	respOrders = append(respOrders, resp...)
	// }

	format, err := c.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(respOrders))
	for i := range respOrders {
		var curr currency.Pair
		curr, err = currency.NewPairDelimiter(respOrders[i].ProductID,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(respOrders[i].Side)
		if err != nil {
			return nil, err
		}
		var orderType order.Type
		// orderType, err = order.StringToOrderType(respOrders[i].Type)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", c.Name, err)
		}
		orders[i] = order.Detail{
			// OrderID:        respOrders[i].ID,
			// Amount:         respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			Type:           orderType,
			Date:           respOrders[i].CreatedTime,
			Side:           side,
			Pair:           curr,
			Exchange:       c.Name,
		}
	}
	return req.Filter(c.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (c *CoinbasePro) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var respOrders []GetOrderResponse
	// if len(req.Pairs) > 0 {
	// var fPair currency.Pair
	// var resp []GetOrderResponse
	// for i := range req.Pairs {
	// fPair, err = c.FormatExchangeCurrency(req.Pairs[i], asset.Spot)
	// if err != nil {
	// 	return nil, err
	// }
	// resp, err = c.GetOrders(ctx, []string{"done"}, fPair.String())
	// if err != nil {
	// 	return nil, err
	// }
	// respOrders = append(respOrders, resp...)
	// }
	// } else {
	// respOrders, err = c.GetOrders(ctx, []string{"done"}, "")
	// if err != nil {
	// 	return nil, err
	// }
	// }

	format, err := c.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(respOrders))
	for i := range respOrders {
		var curr currency.Pair
		curr, err = currency.NewPairDelimiter(respOrders[i].ProductID,
			format.Delimiter)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(respOrders[i].Side)
		if err != nil {
			return nil, err
		}
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(respOrders[i].Status)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", c.Name, err)
		}
		var orderType order.Type
		// orderType, err = order.StringToOrderType(respOrders[i].Type)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", c.Name, err)
		}
		detail := order.Detail{
			OrderID: respOrders[i].OrderID,
			// Amount:          respOrders[i].Size,
			ExecutedAmount: respOrders[i].FilledSize,
			// RemainingAmount: respOrders[i].Size - respOrders[i].FilledSize,
			// Cost:            respOrders[i].ExecutedValue,
			CostAsset: curr.Quote,
			Type:      orderType,
			Date:      respOrders[i].CreatedTime,
			// CloseTime:       respOrders[i].DoneAt,
			// Fee:             respOrders[i].FillFees,
			FeeAsset: curr.Quote,
			Side:     side,
			Status:   orderStatus,
			Pair:     curr,
			// Price:           respOrders[i].Price,
			Exchange: c.Name,
		}
		detail.InferCostsAndTimes()
		orders[i] = detail
	}
	return req.Filter(c.Name, orders), nil
}

// GetHistoricCandles returns a set of candle between two time periods for a
// designated time period
func (c *CoinbasePro) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	// req, err := c.GetKlineRequest(pair, a, interval, start, end, false)
	// if err != nil {
	// 	return nil, err
	// }

	// history, err := c.GetHistoricRates(ctx,
	// 	req.RequestFormatted.String(),
	// 	start.Format(time.RFC3339),
	// 	end.Format(time.RFC3339),
	// 	int64(req.ExchangeInterval.Duration().Seconds()))
	// if err != nil {
	// 	return nil, err
	// }

	// timeSeries := make([]kline.Candle, len(history))
	// for x := range history {
	// 	timeSeries[x] = kline.Candle{
	// 		Time:   history[x].Time,
	// 		Low:    history[x].Low,
	// 		High:   history[x].High,
	// 		Open:   history[x].Open,
	// 		Close:  history[x].Close,
	// 		Volume: history[x].Volume,
	// 	}
	// }
	// return req.ProcessResponse(timeSeries)
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (c *CoinbasePro) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	// req, err := c.GetKlineExtendedRequest(pair, a, interval, start, end)
	// if err != nil {
	// 	return nil, err
	// }

	// timeSeries := make([]kline.Candle, 0, req.Size())
	// for x := range req.RangeHolder.Ranges {
	// 	var history []History
	// 	history, err = c.GetHistoricRates(ctx,
	// 		req.RequestFormatted.String(),
	// 		req.RangeHolder.Ranges[x].Start.Time.Format(time.RFC3339),
	// 		req.RangeHolder.Ranges[x].End.Time.Format(time.RFC3339),
	// 		int64(req.ExchangeInterval.Duration().Seconds()))
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	for i := range history {
	// 		timeSeries = append(timeSeries, kline.Candle{
	// 			Time:   history[i].Time,
	// 			Low:    history[i].Low,
	// 			High:   history[i].High,
	// 			Open:   history[i].Open,
	// 			Close:  history[i].Close,
	// 			Volume: history[i].Volume,
	// 		})
	// 	}
	// }
	// return req.ProcessResponse(timeSeries)
	return nil, common.ErrFunctionNotSupported
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (c *CoinbasePro) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := c.UpdateAccountInfo(ctx, assetType)
	return c.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (c *CoinbasePro) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	// st, err := c.GetCurrentServerTime(ctx)
	// if err != nil {
	// 	return time.Time{}, err
	// }
	// return st.ISO, nil
	return time.Time{}, errors.New(common.ErrFunctionNotSupported.Error())
}

// GetLatestFundingRates returns the latest funding rates data
func (c *CoinbasePro) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (c *CoinbasePro) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// UpdateOrderExecutionLimits updates order execution limits
func (c *CoinbasePro) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}
