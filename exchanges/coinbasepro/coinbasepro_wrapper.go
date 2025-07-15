package coinbasepro

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
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

// SetDefaults sets default values for the exchange
func (c *CoinbasePro) SetDefaults() {
	c.Name = "CoinbasePro"
	c.Enabled = true
	c.API.CredentialsValidator.RequiresKey = true
	c.API.CredentialsValidator.RequiresSecret = true
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
				AutoPairUpdates:                true,
				AccountBalance:                 true,
				CryptoDeposit:                  true,
				CryptoWithdrawal:               true,
				FiatWithdraw:                   true,
				GetOrder:                       true,
				GetOrders:                      true,
				CancelOrders:                   true,
				CancelOrder:                    true,
				SubmitOrder:                    true,
				ModifyOrder:                    true,
				DepositHistory:                 true,
				WithdrawalHistory:              true,
				FiatWithdrawalFee:              true,
				CryptoWithdrawalFee:            true,
				TickerFetching:                 true,
				KlineFetching:                  true,
				OrderbookFetching:              true,
				AccountInfo:                    true,
				FiatDeposit:                    true,
				FundingRateFetching:            true,
				HasAssetTypeAccountSegregation: true,
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
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
				),
				GlobalResultLimit: 300,
			},
		},
		Subscriptions:       defaultSubscriptions.Clone(),
		TradingRequirements: protocol.TradingRequirements{},
	}
	c.Requester, err = request.New(c.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(rateLimits))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	c.API.Endpoints = c.NewEndpoints()
	err = c.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              coinbaseAPIURL,
		exchange.RestSandbox:           coinbaseproSandboxAPIURL,
		exchange.WebsocketSpot:         coinbaseproWebsocketURL,
		exchange.RestSpotSupplementary: coinbaseV1APIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	c.Websocket = websocket.NewManager()
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
	c.checkSubscriptions()
	wsRunningURL, err := c.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = c.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            coinbaseproWebsocketURL,
		RunningURL:            wsRunningURL,
		Connector:             c.WsConnect,
		Subscriber:            c.Subscribe,
		Unsubscriber:          c.Unsubscribe,
		GenerateSubscriptions: c.generateSubscriptions,
		Features:              &c.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer: true,
		},
	})
	if err != nil {
		return err
	}

	return c.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (c *CoinbasePro) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	var products *AllProducts
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return nil, err
	}
	aString := FormatAssetOutbound(a)
	if verified {
		products, err = c.GetAllProducts(ctx, 0, 0, aString, "", "", "", nil, true, true, verified)
		if err != nil {
			log.Warnf(log.ExchangeSys, warnAuth, err)
			verified = false
		}
	}
	if !verified {
		products, err = c.GetAllProducts(ctx, 0, 0, aString, "", "", "", nil, false, true, verified)
		if err != nil {
			return nil, err
		}
	}
	pairs := make([]currency.Pair, 0, len(products.Products))
	aliases := make(map[currency.Pair]currency.Pairs)
	for x := range products.Products {
		if products.Products[x].TradingDisabled {
			continue
		}
		if products.Products[x].Price == 0 {
			continue
		}
		pairs = append(pairs, products.Products[x].ID)
		if !products.Products[x].Alias.IsEmpty() {
			aliases[products.Products[x].Alias] = aliases[products.Products[x].Alias].Add(products.Products[x].ID)
		}
		if len(products.Products[x].AliasTo) > 0 {
			aliases[products.Products[x].ID] = aliases[products.Products[x].ID].Add(products.Products[x].AliasTo...)
		}
		// Products need to be considered aliases of themselves for some code in websocket, and it seems better to add that here
		aliases[products.Products[x].ID] = aliases[products.Products[x].ID].Add(products.Products[x].ID)
	}
	c.pairAliases.Load(aliases)
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (c *CoinbasePro) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := c.GetAssetTypes(false)
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

// UpdateAccountInfo retrieves balances for all enabled currencies for the coinbasepro exchange
func (c *CoinbasePro) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var (
		response       account.Holdings
		accountBalance []Account
		done           bool
		err            error
		cursor         int64
		accountResp    *AllAccountsResponse
	)
	response.Exchange = c.Name
	for !done {
		accountResp, err = c.ListAccounts(ctx, 250, cursor)
		if err != nil {
			return response, err
		}
		accountBalance = append(accountBalance, accountResp.Accounts...)
		done = !accountResp.HasNext
		cursor = int64(accountResp.Cursor)
	}
	accountCurrencies := make(map[string][]account.Balance)
	for i := range accountBalance {
		profileID := accountBalance[i].UUID
		currencies := accountCurrencies[profileID]
		accountCurrencies[profileID] = append(currencies, account.Balance{
			Currency:               currency.NewCode(accountBalance[i].Currency),
			Total:                  accountBalance[i].AvailableBalance.Value.Float64(),
			Hold:                   accountBalance[i].Hold.Value.Float64(),
			Free:                   accountBalance[i].AvailableBalance.Value.Float64() - accountBalance[i].Hold.Value.Float64(),
			AvailableWithoutBorrow: accountBalance[i].AvailableBalance.Value.Float64(),
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

// UpdateTickers updates all currency pairs of a given asset type
func (c *CoinbasePro) UpdateTickers(context.Context, asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (c *CoinbasePro) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return nil, err
	}
	fPair, err := c.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	err = c.tickerHelper(ctx, fPair, a, verified)
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(c.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (c *CoinbasePro) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return nil, err
	}
	p, err = c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	err = c.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          c.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: c.ValidateOrderbook,
	}
	var orderbookNew *ProductBookResp
	if verified {
		orderbookNew, err = c.GetProductBookV3(ctx, p, 1000, 0, true)
		if err != nil {
			log.Warnf(log.ExchangeSys, warnAuth, err)
			verified = false
		}
	}
	if !verified {
		orderbookNew, err = c.GetProductBookV3(ctx, p, 1000, 0, false)
		if err != nil {
			return book, err
		}
	}
	book.Bids = make(orderbook.Levels, len(orderbookNew.Pricebook.Bids))
	for x := range orderbookNew.Pricebook.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Pricebook.Bids[x].Size.Float64(),
			Price:  orderbookNew.Pricebook.Bids[x].Price.Float64(),
		}
	}
	book.Asks = make(orderbook.Levels, len(orderbookNew.Pricebook.Asks))
	for x := range orderbookNew.Pricebook.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Pricebook.Asks[x].Size.Float64(),
			Price:  orderbookNew.Pricebook.Asks[x].Price.Float64(),
		}
	}
	aliases := c.pairAliases.GetAlias(p)
	var errs error
	var validPairs currency.Pairs
	for i := range aliases {
		isEnabled, err := c.CurrencyPairs.IsPairEnabled(aliases[i], assetType)
		if err != nil {
			errs = fmt.Errorf("%v %v", errs, err)
			continue
		}
		if isEnabled {
			book.Pair = aliases[i]
			err = book.Process()
			if err != nil {
				errs = fmt.Errorf("%v %v", errs, err)
				continue
			}
			validPairs = append(validPairs, book.Pair)
		}
	}
	if errs != nil {
		return book, errs
	}
	if len(validPairs) == 0 {
		return book, errPairsDisabledOrErrored
	}
	return orderbook.Get(c.Name, validPairs[0], assetType)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (c *CoinbasePro) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	wallIDs, err := c.GetAllWallets(ctx, PaginationInp{})
	if err != nil {
		return nil, err
	}
	if len(wallIDs.Data) == 0 {
		return nil, errNoWalletsReturned
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
	return c.processFundingData(accHistory, cryptoHistory)
}

// GetWithdrawalsHistory returns previous withdrawals data
func (c *CoinbasePro) GetWithdrawalsHistory(ctx context.Context, cur currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	tempWallIDs, err := c.GetAllWallets(ctx, PaginationInp{})
	if err != nil {
		return nil, err
	}
	if len(tempWallIDs.Data) == 0 {
		return nil, errNoWalletsReturned
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
	tempFundingData, err := c.processFundingData(accHistory, cryptoHistory)
	if err != nil {
		return nil, err
	}
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
func (c *CoinbasePro) GetRecentTrades(context.Context, currency.Pair, asset.Item) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (c *CoinbasePro) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (c *CoinbasePro) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate(c.GetTradingRequirements())
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
	resp, err := c.PlaceOrder(ctx, &PlaceOrderInfo{
		ClientOID:     s.ClientOrderID,
		ProductID:     fPair.String(),
		Side:          s.Side.String(),
		StopDirection: stopDir,
		OrderType:     s.Type,
		TimeInForce:   s.TimeInForce,
		MarginType:    s.MarginType.Upper(),
		BaseAmount:    s.Amount,
		QuoteAmount:   s.QuoteAmount,
		LimitPrice:    s.Price,
		StopPrice:     s.TriggerPrice,
		Leverage:      s.Leverage,
		PostOnly:      s.TimeInForce.Is(order.PostOnly),
		RFQDisabled:   s.RFQDisabled,
		EndTime:       s.EndTime,
	})
	if err != nil {
		return nil, err
	}
	subResp, err := s.DeriveSubmitResponse(resp.SuccessResponse.OrderID)
	if err != nil {
		return nil, err
	}
	if s.RetrieveFees {
		time.Sleep(s.RetrieveFeeDelay)
		feeResp, err := c.GetOrderByID(ctx, resp.SuccessResponse.OrderID, s.ClientOrderID, currency.Code{})
		if err != nil {
			return nil, err
		}
		subResp.Fee = feeResp.TotalFees.Float64()
	}
	return subResp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to market conversion
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
		return nil, errOrderModFailNoRet
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
		return fmt.Errorf("%w %v", errOrderFailedToCancel, o.OrderID)
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (c *CoinbasePro) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	var status order.CancelBatchResponse
	ordToCancel := len(o)
	if ordToCancel == 0 {
		return nil, errOrderIDEmpty
	}
	status.Status = make(map[string]string)
	ordIDSlice := make([]string, ordToCancel)
	for i := range o {
		err := o[i].Validate(o[i].StandardCancel())
		if err != nil {
			return nil, err
		}
		ordIDSlice[i] = o[i].OrderID
		status.Status[o[i].OrderID] = "Failed to cancel"
	}
	resp := struct {
		Results []OrderCancelDetail `json:"results"`
	}{}
	for i := 0; i < ordToCancel; i += 100 {
		var tempOrdIDSlice []string
		if ordToCancel-i < 100 {
			tempOrdIDSlice = ordIDSlice[i:]
		} else {
			tempOrdIDSlice = ordIDSlice[i : i+100]
		}
		tempResp, err := c.CancelOrders(ctx, tempOrdIDSlice)
		if err != nil {
			return nil, err
		}
		resp.Results = append(resp.Results, tempResp...)
	}
	for i := range resp.Results {
		if resp.Results[i].Success {
			status.Status[resp.Results[i].OrderID] = order.Cancelled.String()
		}
	}
	return &status, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(context.Context, *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrFunctionNotSupported
}

// GetOrderInfo returns order information based on order ID
func (c *CoinbasePro) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetItem asset.Item) (*order.Detail, error) {
	genOrderDetail, err := c.GetOrderByID(ctx, orderID, "", currency.Code{})
	if err != nil {
		return nil, err
	}
	response := c.getOrderRespToOrderDetail(genOrderDetail, pair, assetItem)
	fillData, err := c.ListFills(ctx, []string{orderID}, nil, nil, 0, "", time.Time{}, time.Now(), manyFills)
	if err != nil {
		return nil, err
	}
	cursor := fillData.Cursor
	for cursor != 0 {
		tempFillData, err := c.ListFills(ctx, []string{orderID}, nil, nil, int64(cursor), "", time.Time{}, time.Now(), manyFills)
		if err != nil {
			return nil, err
		}
		fillData.Fills = append(fillData.Fills, tempFillData.Fills...)
		cursor = tempFillData.Cursor
	}
	response.Trades = make([]order.TradeHistory, len(fillData.Fills))
	var orderSide order.Side
	switch response.Side {
	case order.Buy:
		orderSide = order.Sell
	case order.Sell:
		orderSide = order.Buy
	}
	for i := range fillData.Fills {
		response.Trades[i] = order.TradeHistory{
			Price:     fillData.Fills[i].Price.Float64(),
			Amount:    fillData.Fills[i].Size.Float64(),
			Fee:       fillData.Fills[i].Commission.Float64(),
			Exchange:  c.GetName(),
			TID:       fillData.Fills[i].TradeID,
			Side:      orderSide,
			Timestamp: fillData.Fills[i].TradeTime,
			Total:     fillData.Fills[i].Price.Float64() * fillData.Fills[i].Size.Float64(),
		}
	}
	return response, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (c *CoinbasePro) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	allWalResp, err := c.GetAllWallets(ctx, PaginationInp{})
	if err != nil {
		return nil, err
	}
	var targetWalletID string
	for i := range allWalResp.Data {
		if allWalResp.Data[i].Currency.Code == cryptocurrency.String() {
			targetWalletID = allWalResp.Data[i].ID
			break
		}
	}
	if targetWalletID == "" {
		return nil, errNoWalletForCurrency
	}
	resp, err := c.GetAllAddresses(ctx, targetWalletID, PaginationInp{})
	if err != nil || len(resp.Data) == 0 {
		resp2, err2 := c.CreateAddress(ctx, targetWalletID, "")
		if err2 != nil {
			return nil, common.AppendError(err, err2)
		}
		return &deposit.Address{
			Address: resp2.Address,
			Tag:     resp2.Name,
			Chain:   resp2.Network,
		}, nil
	}
	return &deposit.Address{
		Address: resp.Data[0].Address,
		Tag:     resp.Data[0].Name,
		Chain:   resp.Data[0].Network,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.WalletID == "" {
		return nil, errWalletIDEmpty
	}
	travel := TravelRule{
		BeneficiaryWalletType: withdrawRequest.Travel.BeneficiaryWalletType,
		BeneficiaryName:       withdrawRequest.Travel.BeneficiaryName,
		BeneficiaryAddress: FullAddress{
			Address1:   withdrawRequest.Travel.BeneficiaryAddress.Address1,
			Address2:   withdrawRequest.Travel.BeneficiaryAddress.Address2,
			Address3:   withdrawRequest.Travel.BeneficiaryAddress.Address3,
			City:       withdrawRequest.Travel.BeneficiaryAddress.City,
			State:      withdrawRequest.Travel.BeneficiaryAddress.State,
			Country:    withdrawRequest.Travel.BeneficiaryAddress.Country,
			PostalCode: withdrawRequest.Travel.BeneficiaryAddress.PostalCode,
		},
		BeneficiaryFinancialInstitution: withdrawRequest.Travel.BeneficiaryFinancialInstitution,
		TransferPurpose:                 withdrawRequest.Travel.TransferPurpose,
	}
	if withdrawRequest.Travel.IsSelf {
		travel.IsSelf = "IS_SELF_TRUE"
	} else {
		travel.IsSelf = "IS_SELF_FALSE"
	}
	resp, err := c.SendMoney(ctx, "send", withdrawRequest.WalletID, withdrawRequest.Crypto.Address, withdrawRequest.Currency.String(), withdrawRequest.Description, withdrawRequest.IdempotencyToken, withdrawRequest.Crypto.AddressTag, "", withdrawRequest.Amount, false, travel)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: resp.ID, Status: resp.Status}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.WalletID == "" {
		return nil, errWalletIDEmpty
	}
	paymentMethods, err := c.ListPaymentMethods(ctx)
	if err != nil {
		return nil, err
	}
	selectedWithdrawalMethod := PaymentMethodData{}
	for i := range paymentMethods {
		if withdrawRequest.Fiat.Bank.BankName == paymentMethods[i].Name {
			selectedWithdrawalMethod = paymentMethods[i]
			break
		}
	}
	if selectedWithdrawalMethod.ID == "" {
		return nil, fmt.Errorf("%w %v", errPayMethodNotFound, withdrawRequest.Fiat.Bank.BankName)
	}
	resp, err := c.FiatTransfer(ctx, withdrawRequest.WalletID, withdrawRequest.Currency.String(), selectedWithdrawalMethod.ID, withdrawRequest.Amount, true, FiatWithdrawal)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   selectedWithdrawalMethod.Name,
		ID:     resp.ID,
		Status: resp.Status,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is submitted
func (c *CoinbasePro) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return c.WithdrawFiatFunds(ctx, withdrawRequest)
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (c *CoinbasePro) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !c.AreCredentialsValid(ctx) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return c.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (c *CoinbasePro) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var respOrders []GetOrderResponse
	ordStatus := []string{"OPEN"}
	respOrders, err = c.iterativeGetAllOrders(ctx, req.Pairs, req.Type.String(), req.Side.String(), req.AssetType.Upper(), ordStatus, 1000, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, len(respOrders))
	for i := range respOrders {
		orderRec := c.getOrderRespToOrderDetail(&respOrders[i], respOrders[i].ProductID, asset.Spot)
		orders[i] = *orderRec
	}
	if len(req.Pairs) > 1 {
		order.FilterOrdersByPairs(&orders, req.Pairs)
	}
	return req.Filter(c.Name, orders), nil
}

// GetOrderHistory retrieves account order information. Can Limit response to specific order status
func (c *CoinbasePro) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	for i := range req.Pairs {
		req.Pairs[i], err = c.FormatExchangeCurrency(req.Pairs[i], req.AssetType)
		if err != nil {
			return nil, err
		}
	}
	var ord []GetOrderResponse
	interOrd, err := c.iterativeGetAllOrders(ctx, req.Pairs, req.Type.String(), req.Side.String(), req.AssetType.Upper(), closedStatuses, manyOrds, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	ord = append(ord, interOrd...)
	interOrd, err = c.iterativeGetAllOrders(ctx, req.Pairs, req.Type.String(), req.Side.String(), req.AssetType.Upper(), openStatus, manyOrds, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	ord = append(ord, interOrd...)
	orders := make([]order.Detail, len(ord))
	for i := range ord {
		singleOrder := c.getOrderRespToOrderDetail(&ord[i], ord[i].ProductID, req.AssetType)
		orders[i] = *singleOrder
	}
	if len(req.Pairs) > 1 {
		order.FilterOrdersByPairs(&orders, req.Pairs)
	}
	return req.Filter(c.Name, orders), nil
}

// GetHistoricCandles returns a set of candle between two time periods for a designated time period
func (c *CoinbasePro) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := c.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return nil, err
	}
	timeSeries, err := c.candleHelper(ctx, req.RequestFormatted.String(), interval, start, end, verified)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (c *CoinbasePro) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := c.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return nil, err
	}
	var timeSeries []kline.Candle
	for x := range req.RangeHolder.Ranges {
		hist, err := c.candleHelper(ctx, req.RequestFormatted.String(), interval, req.RangeHolder.Ranges[x].Start.Time.Add(-time.Nanosecond), req.RangeHolder.Ranges[x].End.Time.Add(-time.Nanosecond), verified)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, hist...)
	}
	return req.ProcessResponse(timeSeries)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (c *CoinbasePro) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := c.UpdateAccountInfo(ctx, assetType)
	return c.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (c *CoinbasePro) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	st, err := c.GetV3Time(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return st.Iso, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (c *CoinbasePro) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if !c.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return nil, err
	}
	products, perpStart, err := c.fetchFutures(ctx, verified)
	if err != nil {
		return nil, err
	}
	funding := make([]fundingrate.LatestRateResponse, len(products.Products))
	for i := perpStart; i < len(products.Products); i++ {
		funRate := fundingrate.Rate{
			Time: products.Products[i].FutureProductDetails.PerpetualDetails.FundingTime,
			Rate: decimal.NewFromFloat(products.Products[i].FutureProductDetails.PerpetualDetails.FundingRate.Float64()),
		}
		funding[i] = fundingrate.LatestRateResponse{
			Exchange:    c.Name,
			Asset:       r.Asset,
			Pair:        products.Products[i].ID,
			LatestRate:  funRate,
			TimeChecked: time.Now(),
		}
	}
	return funding, nil
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (c *CoinbasePro) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !c.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return nil, err
	}
	products, perpStart, err := c.fetchFutures(ctx, verified)
	if err != nil {
		return nil, err
	}
	contracts := make([]futures.Contract, len(products.Products))
	for i := range products.Products {
		funRate := fundingrate.Rate{
			Time: products.Products[i].FutureProductDetails.PerpetualDetails.FundingTime,
			Rate: decimal.NewFromFloat(products.Products[i].FutureProductDetails.PerpetualDetails.FundingRate.Float64()),
		}
		contracts[i] = futures.Contract{
			Exchange:             c.Name,
			Name:                 products.Products[i].ID,
			Asset:                item,
			EndDate:              products.Products[i].FutureProductDetails.ContractExpiry,
			IsActive:             !products.Products[i].IsDisabled,
			Status:               products.Products[i].Status,
			SettlementCurrencies: currency.Currencies{products.Products[i].QuoteCurrencyID},
			Multiplier:           products.Products[i].BaseIncrement.Float64(),
			LatestRate:           funRate,
		}
		if i < perpStart {
			contracts[i].Type = futures.LongDated
		} else {
			contracts[i].Type = futures.Perpetual
		}
	}
	return contracts, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (c *CoinbasePro) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var data *AllProducts
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return err
	}
	aString := FormatAssetOutbound(a)
	if verified {
		data, err = c.GetAllProducts(ctx, 0, 0, aString, "", "", "", nil, true, true, true)
		if err != nil {
			log.Warnf(log.ExchangeSys, warnAuth, err)
			verified = false
		}
	}
	if !verified {
		data, err = c.GetAllProducts(ctx, 0, 0, aString, "", "", "", nil, false, true, false)
		if err != nil {
			return err
		}
	}
	limits := make([]order.MinMaxLevel, len(data.Products))
	for i := range data.Products {
		limits[i] = order.MinMaxLevel{
			Pair:                    data.Products[i].ID,
			Asset:                   a,
			MinPrice:                data.Products[i].QuoteMinSize.Float64(),
			MaxPrice:                data.Products[i].QuoteMaxSize.Float64(),
			PriceStepIncrementSize:  data.Products[i].PriceIncrement.Float64(),
			MinimumBaseAmount:       data.Products[i].BaseMinSize.Float64(),
			MaximumBaseAmount:       data.Products[i].BaseMaxSize.Float64(),
			MinimumQuoteAmount:      data.Products[i].QuoteMinSize.Float64(),
			MaximumQuoteAmount:      data.Products[i].QuoteMaxSize.Float64(),
			AmountStepIncrementSize: data.Products[i].BaseIncrement.Float64(),
			QuoteStepIncrementSize:  data.Products[i].QuoteIncrement.Float64(),
			MaxTotalOrders:          1000,
		}
	}
	return c.LoadLimits(limits)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (c *CoinbasePro) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := c.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	return tradeBaseURL + cp.Upper().String(), nil
}

// fetchFutures is a helper function for GetLatestFundingRates and GetFuturesContractDetails that calls the List Products endpoint twice, to get both expiring futures and perpetual futures
func (c *CoinbasePro) fetchFutures(ctx context.Context, verified bool) (*AllProducts, int, error) {
	products, err := c.GetAllProducts(ctx, 0, 0, "FUTURE", "", "", "", nil, false, false, verified)
	if err != nil {
		if verified {
			return c.fetchFutures(ctx, false)
		}
		return nil, 0, err
	}
	products2, err := c.GetAllProducts(ctx, 0, 0, "FUTURE", "PERPETUAL", "", "", nil, false, false, verified)
	if err != nil {
		if verified {
			return c.fetchFutures(ctx, false)
		}
		return nil, 0, err
	}
	perpStart := len(products.Products)
	products.Products = append(products.Products, products2.Products...)
	return products, perpStart, nil
}

// processFundingData is a helper function for GetAccountFundingHistory and GetWithdrawalsHistory, transforming the data returned by the Coinbase API into a format suitable for the exchange package
func (c *CoinbasePro) processFundingData(accHistory []DeposWithdrData, cryptoHistory []TransactionData) ([]exchange.FundingHistory, error) {
	fundingData := make([]exchange.FundingHistory, len(accHistory)+len(cryptoHistory))
	for i := range accHistory {
		fundingData[i] = exchange.FundingHistory{
			ExchangeName: c.Name,
			Status:       accHistory[i].Status,
			TransferID:   accHistory[i].ID,
			Timestamp:    accHistory[i].PayoutAt,
			Currency:     accHistory[i].Amount.Currency,
			Amount:       accHistory[i].Amount.Value.Float64(),
			Fee:          accHistory[i].TotalFee.Amount.Value.Float64(),
		}
		switch accHistory[i].Type {
		case "TRANSFER_TYPE_DEPOSIT":
			fundingData[i].TransferType = "deposit"
		case "TRANSFER_TYPE_WITHDRAWAL":
			fundingData[i].TransferType = "withdrawal"
		default:
			return nil, fmt.Errorf("%w %v", errUnknownTransferType, accHistory[i].Type)
		}
	}
	for i := range cryptoHistory {
		fundingData[i+len(accHistory)] = exchange.FundingHistory{
			ExchangeName: c.Name,
			Status:       cryptoHistory[i].Status,
			TransferID:   cryptoHistory[i].ID,
			Timestamp:    cryptoHistory[i].CreatedAt,
			Currency:     cryptoHistory[i].Amount.Currency,
			Amount:       cryptoHistory[i].Amount.Amount.Float64(),
		}
		if cryptoHistory[i].Type == "receive" {
			fundingData[i+len(accHistory)].TransferType = "deposit"
		}
		if cryptoHistory[i].Type == "send" {
			fundingData[i+len(accHistory)].TransferType = "withdrawal"
		}
	}
	return fundingData, nil
}

// iterativeGetAllOrders is a helper function used in GetActiveOrders and GetOrderHistory to repeatedly call GetAllOrders until all orders have been retrieved
func (c *CoinbasePro) iterativeGetAllOrders(ctx context.Context, productIDs currency.Pairs, orderType, orderSide, productType string, orderStatus []string, limit int32, startDate, endDate time.Time) ([]GetOrderResponse, error) {
	hasNext := true
	var resp []GetOrderResponse
	var cursor int64
	if orderSide == "ANY" {
		orderSide = ""
	}
	if orderType == "ANY" {
		orderType = ""
	}
	if productType == "FUTURES" {
		productType = "FUTURE"
	}
	for hasNext {
		interResp, err := c.ListOrders(ctx, nil, orderStatus, nil, []string{orderType}, nil, productIDs, productType, orderSide, "", "", "", "", cursor, limit, startDate, endDate, currency.Code{})
		if err != nil {
			return nil, err
		}
		resp = append(resp, interResp.Orders...)
		hasNext = interResp.HasNext
		cursor = int64(interResp.Cursor)
	}
	return resp, nil
}

// FormatExchangeKlineIntervalV3 is a helper function used in GetHistoricCandles and GetHistoricCandlesExtended to convert kline.Interval to the string format used by V3 of Coinbase's API
func FormatExchangeKlineIntervalV3(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return granOneMin, nil
	case kline.FiveMin:
		return granFiveMin, nil
	case kline.FifteenMin:
		return granFifteenMin, nil
	case kline.ThirtyMin:
		return granThirtyMin, nil
	case kline.OneHour:
		return granOneHour, nil
	case kline.TwoHour:
		return granTwoHour, nil
	case kline.SixHour:
		return granSixHour, nil
	case kline.OneDay:
		return granOneDay, nil
	}
	return "", kline.ErrUnsupportedInterval
}

// getOrderRespToOrderDetail is a helper function used in GetOrderInfo, GetActiveOrders, and GetOrderHistory to convert data returned by the Coinbase API into a format suitable for the exchange package
func (c *CoinbasePro) getOrderRespToOrderDetail(genOrderDetail *GetOrderResponse, pair currency.Pair, assetItem asset.Item) *order.Detail {
	var amount float64
	var quoteAmount float64
	var orderType order.Type
	if genOrderDetail.OrderConfiguration.MarketMarketIOC != nil {
		quoteAmount = genOrderDetail.OrderConfiguration.MarketMarketIOC.QuoteSize.Float64()
		amount = genOrderDetail.OrderConfiguration.MarketMarketIOC.BaseSize.Float64()
		orderType = order.Market
	}
	var price float64
	var postOnly bool
	if genOrderDetail.OrderConfiguration.LimitLimitGTC != nil {
		amount = genOrderDetail.OrderConfiguration.LimitLimitGTC.BaseSize.Float64()
		price = genOrderDetail.OrderConfiguration.LimitLimitGTC.LimitPrice.Float64()
		postOnly = genOrderDetail.OrderConfiguration.LimitLimitGTC.PostOnly
		orderType = order.Limit
	}
	if genOrderDetail.OrderConfiguration.LimitLimitGTD != nil {
		amount = genOrderDetail.OrderConfiguration.LimitLimitGTD.BaseSize.Float64()
		price = genOrderDetail.OrderConfiguration.LimitLimitGTD.LimitPrice.Float64()
		postOnly = genOrderDetail.OrderConfiguration.LimitLimitGTD.PostOnly
		orderType = order.Limit
	}
	var triggerPrice float64
	if genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC != nil {
		amount = genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC.BaseSize.Float64()
		price = genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC.LimitPrice.Float64()
		triggerPrice = genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC.StopPrice.Float64()
		orderType = order.StopLimit
	}
	if genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD != nil {
		amount = genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD.BaseSize.Float64()
		price = genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD.LimitPrice.Float64()
		triggerPrice = genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD.StopPrice.Float64()
		orderType = order.StopLimit
	}
	var remainingAmount float64
	if !genOrderDetail.SizeInQuote {
		remainingAmount = amount - genOrderDetail.FilledSize.Float64()
	}
	var orderSide order.Side
	switch genOrderDetail.Side {
	case order.Buy.String():
		orderSide = order.Buy
	case order.Sell.String():
		orderSide = order.Sell
	}
	var orderStatus order.Status
	switch genOrderDetail.Status {
	case order.Open.String():
		orderStatus = order.Open
	case order.Filled.String():
		orderStatus = order.Filled
	case order.Cancelled.String():
		orderStatus = order.Cancelled
	case order.Expired.String():
		orderStatus = order.Expired
	case "FAILED":
		orderStatus = order.Rejected
	case "UNKNOWN_ORDER_STATUS":
		orderStatus = order.UnknownStatus
	}
	var closeTime time.Time
	if genOrderDetail.Settled {
		closeTime = genOrderDetail.LastFillTime
	}
	var lastUpdateTime time.Time
	if len(genOrderDetail.EditHistory) > 0 {
		lastUpdateTime = genOrderDetail.EditHistory[len(genOrderDetail.EditHistory)-1].ReplaceAcceptTimestamp
	}
	var tif order.TimeInForce
	if postOnly {
		tif = order.PostOnly
	}
	if genOrderDetail.OrderConfiguration.MarketMarketIOC != nil {
		tif |= order.ImmediateOrCancel
	}
	response := order.Detail{
		TimeInForce:          tif,
		Price:                price,
		Amount:               amount,
		TriggerPrice:         triggerPrice,
		AverageExecutedPrice: genOrderDetail.AverageFilledPrice.Float64(),
		QuoteAmount:          quoteAmount,
		ExecutedAmount:       genOrderDetail.FilledSize.Float64(),
		RemainingAmount:      remainingAmount,
		Cost:                 genOrderDetail.TotalValueAfterFees.Float64(),
		Fee:                  genOrderDetail.TotalFees.Float64(),
		Exchange:             c.GetName(),
		OrderID:              genOrderDetail.OrderID,
		ClientOrderID:        genOrderDetail.ClientOID,
		ClientID:             genOrderDetail.UserID,
		Type:                 orderType,
		Side:                 orderSide,
		Status:               orderStatus,
		AssetType:            assetItem,
		Date:                 genOrderDetail.CreatedTime,
		CloseTime:            closeTime,
		LastUpdated:          lastUpdateTime,
		Pair:                 pair,
	}
	return &response
}

// VerificationCheck returns whether authentication support is enabled or not
func (c *CoinbasePro) verificationCheck(ctx context.Context) (bool, error) {
	_, err := c.GetCredentials(ctx)
	if err != nil {
		if errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled) || errors.Is(err, exchange.ErrCredentialsAreEmpty) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// TickerHelper fetches the ticker for a given currency pair, used by UpdateTickers and UpdateTicker
func (c *CoinbasePro) tickerHelper(ctx context.Context, name currency.Pair, assetType asset.Item, verified bool) error {
	newTick := &ticker.Price{
		Pair:         name,
		ExchangeName: c.Name,
		AssetType:    assetType,
	}
	ticks, err := c.GetTicker(ctx, name, 1, time.Time{}, time.Time{}, verified)
	if err != nil {
		if verified {
			return c.tickerHelper(ctx, name, assetType, false)
		}
		return err
	}
	var last float64
	if len(ticks.Trades) != 0 {
		last = ticks.Trades[0].Price.Float64()
	}
	newTick.Last = last
	newTick.Bid = ticks.BestBid.Float64()
	newTick.Ask = ticks.BestAsk.Float64()
	return ticker.ProcessTicker(newTick)
}

// CandleHelper handles calling the candle function, and doing preliminary work on the data
func (c *CoinbasePro) candleHelper(ctx context.Context, pair string, granularity kline.Interval, start, end time.Time, verified bool) ([]kline.Candle, error) {
	granString, err := FormatExchangeKlineIntervalV3(granularity)
	if err != nil {
		return nil, err
	}
	history, err := c.GetHistoricKlines(ctx, pair, granString, start, end, verified)
	if err != nil {
		if verified {
			return c.candleHelper(ctx, pair, granularity, start, end, false)
		}
		return nil, err
	}
	timeSeries := make([]kline.Candle, len(history))
	for x := range history {
		timeSeries[x] = kline.Candle{
			Time:   history[x].Start.Time(),
			Low:    history[x].Low.Float64(),
			High:   history[x].High.Float64(),
			Open:   history[x].Open.Float64(),
			Close:  history[x].Close.Float64(),
			Volume: history[x].Volume.Float64(),
		}
	}
	return timeSeries, nil
}

// FormatAssetOutbound formats asset items for outbound requests
func FormatAssetOutbound(a asset.Item) string {
	if a == asset.Futures {
		return "FUTURE"
	}
	return a.Upper()
}

// GetAlias returns the aliases for a currency pair
func (a *pairAliases) GetAlias(p currency.Pair) currency.Pairs {
	a.m.RLock()
	defer a.m.RUnlock()
	return slices.Clone(a.associatedAliases[p])
}

// GetAliases returns a map of all aliases associated with all pairs
func (a *pairAliases) GetAliases() map[currency.Pair]currency.Pairs {
	a.m.RLock()
	defer a.m.RUnlock()
	return maps.Clone(a.associatedAliases)
}

// Load adds a batch of aliases to the alias map
func (a *pairAliases) Load(aliases map[currency.Pair]currency.Pairs) {
	a.m.Lock()
	defer a.m.Unlock()
	if a.associatedAliases == nil {
		a.associatedAliases = make(map[currency.Pair]currency.Pairs)
	}
	for k, v := range aliases {
		a.associatedAliases[k] = a.associatedAliases[k].Add(v...)
	}
}
