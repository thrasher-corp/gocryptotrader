package coinbase

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
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
)

// SetDefaults sets default values for the exchange
func (e *Exchange) SetDefaults() {
	e.Name = "Coinbase"
	e.Enabled = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	requestFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	configFmt := &currency.PairFormat{Delimiter: currency.DashDelimiter, Uppercase: true}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Features = exchange.Features{
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
	if e.Requester, err = request.New(e.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout), request.WithLimiter(rateLimits)); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	if err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:              apiURL,
		exchange.RestSandbox:           sandboxAPIURL,
		exchange.WebsocketSpot:         coinbaseWebsocketURL,
		exchange.RestSpotSupplementary: v1APIURL,
	}); err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup initialises the exchange parameters with the current configuration
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
	e.checkSubscriptions()
	wsRunningURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	if err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            coinbaseWebsocketURL,
		RunningURL:            wsRunningURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer: true,
		},
	}); err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	aString := FormatAssetOutbound(a)
	products, err := e.GetAllProducts(ctx, 0, 0, aString, "", "", "", nil, false, true, false)
	if err != nil {
		return nil, err
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
	e.pairAliases.Load(aliases)
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(false)
	for i := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[i])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assets[i], false); err != nil {
			return err
		}
	}
	return e.EnsureOnePairEnabled()
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (subAccts accounts.SubAccounts, err error) {
	for cursor := int64(0); ; {
		resp, err := e.ListAccounts(ctx, 250, cursor)
		if err != nil {
			return subAccts, err
		}
		for _, subAcct := range resp.Accounts {
			a := accounts.NewSubAccount(assetType, subAcct.UUID)
			a.Balances.Set(subAcct.Currency, accounts.Balance{
				Total:                  subAcct.AvailableBalance.Value.Float64(),
				Hold:                   subAcct.Hold.Value.Float64(),
				Free:                   subAcct.AvailableBalance.Value.Float64() - subAcct.Hold.Value.Float64(),
				AvailableWithoutBorrow: subAcct.AvailableBalance.Value.Float64(),
			})
			subAccts = subAccts.Merge(a)
		}
		if !resp.HasNext {
			break
		}
		cursor = int64(resp.Cursor)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(context.Context, asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	if err := e.tickerHelper(ctx, fPair, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, a)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	var orderbookNew *ProductBookResp
	if orderbookNew, err = e.GetProductBookV3(ctx, p, 1000, 0, false); err != nil {
		return book, err
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
	aliases := e.pairAliases.GetAlias(p)
	var errs error
	var validPairs currency.Pairs
	for i := range aliases {
		isEnabled, err := e.CurrencyPairs.IsPairEnabled(aliases[i], assetType)
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if isEnabled {
			book.Pair = aliases[i]
			if err := book.Process(); err != nil {
				errs = common.AppendError(errs, err)
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
	return orderbook.Get(e.Name, validPairs[0], assetType)
}

// GetAccountFundingHistory returns funding history, deposits and withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	wallIDs, err := e.GetAllWallets(ctx, PaginationInp{})
	if err != nil {
		return nil, err
	}
	if len(wallIDs.Data) == 0 {
		return nil, errNoWalletsReturned
	}
	var accHistory []DeposWithdrData
	for i := range wallIDs.Data {
		tempAccHist, err := e.GetAllFiatTransfers(ctx, wallIDs.Data[i].ID, PaginationInp{}, FiatDeposit)
		if err != nil {
			return nil, err
		}
		accHistory = append(accHistory, tempAccHist.Data...)
		if tempAccHist, err = e.GetAllFiatTransfers(ctx, wallIDs.Data[i].ID, PaginationInp{}, FiatWithdrawal); err != nil {
			return nil, err
		}
		accHistory = append(accHistory, tempAccHist.Data...)
	}
	var cryptoHistory []TransactionData
	for i := range wallIDs.Data {
		tempCryptoHist, err := e.GetAllTransactions(ctx, wallIDs.Data[i].ID, PaginationInp{})
		if err != nil {
			return nil, err
		}
		for j := range tempCryptoHist.Data {
			if tempCryptoHist.Data[j].Type == "receive" || tempCryptoHist.Data[j].Type == "send" {
				cryptoHistory = append(cryptoHistory, tempCryptoHist.Data[j])
			}
		}
	}
	return e.processFundingData(accHistory, cryptoHistory)
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, cur currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	tempWallIDs, err := e.GetAllWallets(ctx, PaginationInp{})
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
		tempAccHist, err := e.GetAllFiatTransfers(ctx, wallIDs[i], PaginationInp{}, FiatWithdrawal)
		if err != nil {
			return nil, err
		}
		accHistory = append(accHistory, tempAccHist.Data...)
	}
	var cryptoHistory []TransactionData
	for i := range wallIDs {
		tempCryptoHist, err := e.GetAllTransactions(ctx, wallIDs[i], PaginationInp{})
		if err != nil {
			return nil, err
		}
		for j := range tempCryptoHist.Data {
			if tempCryptoHist.Data[j].Type == "send" {
				cryptoHistory = append(cryptoHistory, tempCryptoHist.Data[j])
			}
		}
	}
	tempFundingData, err := e.processFundingData(accHistory, cryptoHistory)
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
func (e *Exchange) GetRecentTrades(context.Context, currency.Pair, asset.Item) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	fPair, err := e.FormatExchangeCurrency(s.Pair, s.AssetType)
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
	resp, err := e.PlaceOrder(ctx, &PlaceOrderInfo{
		ClientOID:  s.ClientOrderID,
		ProductID:  fPair.String(),
		Side:       s.Side.String(),
		MarginType: s.MarginType.Upper(),
		Leverage:   s.Leverage,
		OrderInfo: OrderInfo{
			StopDirection: stopDir,
			OrderType:     s.Type,
			TimeInForce:   s.TimeInForce,
			BaseAmount:    s.Amount,
			QuoteAmount:   s.QuoteAmount,
			LimitPrice:    s.Price,
			StopPrice:     s.TriggerPrice,
			PostOnly:      s.TimeInForce.Is(order.PostOnly),
			RFQDisabled:   s.RFQDisabled,
			EndTime:       s.EndTime,
		},
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
		feeResp, err := e.GetOrderByID(ctx, resp.SuccessResponse.OrderID, s.ClientOrderID, currency.Code{})
		if err != nil {
			return nil, err
		}
		subResp.Fee = feeResp.TotalFees.Float64()
	}
	return subResp, nil
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(ctx context.Context, m *order.Modify) (*order.ModifyResponse, error) {
	if m == nil {
		return nil, common.ErrNilPointer
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	success, err := e.EditOrder(ctx, m.OrderID, m.Amount, m.Price)
	if err != nil {
		return nil, err
	}
	if !success {
		return nil, errOrderModFailNoRet
	}
	return m.DeriveModifyResponse()
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if o == nil {
		return common.ErrNilPointer
	}
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	canSlice := []order.Cancel{*o}
	resp, err := e.CancelBatchOrders(ctx, canSlice)
	if err != nil {
		return err
	}
	if resp.Status[o.OrderID] != order.Cancelled.String() {
		return fmt.Errorf("%w %v", errOrderFailedToCancel, o.OrderID)
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	var status order.CancelBatchResponse
	ordToCancel := len(o)
	if ordToCancel == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	status.Status = make(map[string]string)
	ordIDSlice := make([]string, ordToCancel)
	for i := range o {
		if err := o[i].Validate(o[i].StandardCancel()); err != nil {
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
		tempResp, err := e.CancelOrders(ctx, tempOrdIDSlice)
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
func (e *Exchange) CancelAllOrders(context.Context, *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrFunctionNotSupported
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetItem asset.Item) (*order.Detail, error) {
	genOrderDetail, err := e.GetOrderByID(ctx, orderID, "", currency.Code{})
	if err != nil {
		return nil, err
	}
	response := e.getOrderRespToOrderDetail(genOrderDetail, pair, assetItem)
	fillData, err := e.ListFills(ctx, []string{orderID}, nil, nil, 0, "", time.Time{}, time.Now(), defaultOrderFillCount)
	if err != nil {
		return nil, err
	}
	cursor := fillData.Cursor
	for cursor != 0 {
		tempFillData, err := e.ListFills(ctx, []string{orderID}, nil, nil, int64(cursor), "", time.Time{}, time.Now(), defaultOrderFillCount)
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
			Exchange:  e.GetName(),
			TID:       fillData.Fills[i].TradeID,
			Side:      orderSide,
			Timestamp: fillData.Fills[i].TradeTime,
			Total:     fillData.Fills[i].Price.Float64() * fillData.Fills[i].Size.Float64(),
		}
	}
	return response, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	allWalResp, err := e.GetAllWallets(ctx, PaginationInp{})
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
	resp, err := e.GetAllAddresses(ctx, targetWalletID, PaginationInp{})
	if err != nil || len(resp.Data) == 0 {
		resp2, err2 := e.CreateAddress(ctx, targetWalletID, "")
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
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.WalletID == "" {
		return nil, errWalletIDEmpty
	}
	travel := &TravelRule{
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
	resp, err := e.SendMoney(ctx, "send", withdrawRequest.WalletID, withdrawRequest.Crypto.Address, withdrawRequest.Description, withdrawRequest.IdempotencyToken, withdrawRequest.Crypto.AddressTag, "", withdrawRequest.Currency, withdrawRequest.Amount, false, travel)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{ID: resp.ID, Status: resp.Status}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.WalletID == "" {
		return nil, errWalletIDEmpty
	}
	paymentMethods, err := e.ListPaymentMethods(ctx)
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
	resp, err := e.FiatTransfer(ctx, withdrawRequest.WalletID, withdrawRequest.Currency.String(), selectedWithdrawalMethod.ID, withdrawRequest.Amount, true, FiatWithdrawal)
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
func (e *Exchange) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return e.WithdrawFiatFunds(ctx, withdrawRequest)
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) && feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var respOrders []GetOrderResponse
	if respOrders, err = e.iterativeGetAllOrders(ctx, req.Pairs, req.Type.String(), req.Side.String(), req.AssetType.Upper(), openStatus, 1000, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	orders := make([]order.Detail, len(respOrders))
	for i := range respOrders {
		orderRec := e.getOrderRespToOrderDetail(&respOrders[i], respOrders[i].ProductID, req.AssetType)
		orders[i] = *orderRec
	}
	if len(req.Pairs) > 1 {
		order.FilterOrdersByPairs(&orders, req.Pairs)
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information. Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	for i := range req.Pairs {
		req.Pairs[i], err = e.FormatExchangeCurrency(req.Pairs[i], req.AssetType)
		if err != nil {
			return nil, err
		}
	}
	var ord []GetOrderResponse
	interOrd, err := e.iterativeGetAllOrders(ctx, req.Pairs, req.Type.String(), req.Side.String(), req.AssetType.Upper(), closedStatuses, defaultOrderCount, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	ord = append(ord, interOrd...)
	if interOrd, err = e.iterativeGetAllOrders(ctx, req.Pairs, req.Type.String(), req.Side.String(), req.AssetType.Upper(), openStatus, defaultOrderCount, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	ord = append(ord, interOrd...)
	orders := make([]order.Detail, len(ord))
	for i := range ord {
		singleOrder := e.getOrderRespToOrderDetail(&ord[i], ord[i].ProductID, req.AssetType)
		orders[i] = *singleOrder
	}
	if len(req.Pairs) > 1 {
		order.FilterOrdersByPairs(&orders, req.Pairs)
	}
	return req.Filter(e.Name, orders), nil
}

// GetHistoricCandles returns a set of candle between two time periods for a designated time period
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	timeSeries, err := e.GetHistoricKlines(ctx, req.RequestFormatted.String(), interval, start, end, false)
	if err != nil {
		return nil, err
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var timeSeries []kline.Candle
	for x := range req.RangeHolder.Ranges {
		hist, err := e.GetHistoricKlines(ctx, req.RequestFormatted.String(), interval, req.RangeHolder.Ranges[x].Start.Time.Add(-time.Nanosecond), req.RangeHolder.Ranges[x].End.Time.Add(-time.Nanosecond), false)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, hist...)
	}
	return req.ProcessResponse(timeSeries)
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	st, err := e.GetV3Time(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return st.Iso, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, common.ErrNilPointer
	}
	if !e.SupportsAsset(r.Asset) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	products, perpStart, err := e.fetchFutures(ctx)
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
			Exchange:    e.Name,
			Asset:       r.Asset,
			Pair:        products.Products[i].ID,
			LatestRate:  funRate,
			TimeChecked: time.Now(),
		}
	}
	return funding, nil
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	products, perpStart, err := e.fetchFutures(ctx)
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
			Exchange:           e.Name,
			Name:               products.Products[i].ID,
			Asset:              item,
			EndDate:            products.Products[i].FutureProductDetails.ContractExpiry,
			IsActive:           !products.Products[i].IsDisabled,
			Status:             products.Products[i].Status,
			SettlementCurrency: products.Products[i].QuoteCurrencyID,
			Multiplier:         products.Products[i].BaseIncrement.Float64(),
			LatestRate:         funRate,
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
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
	aString := FormatAssetOutbound(a)
	data, err := e.GetAllProducts(ctx, 0, 0, aString, "", "", "", nil, false, true, false)
	if err != nil {
		return err
	}
	lim := make([]limits.MinMaxLevel, len(data.Products))
	for i := range data.Products {
		lim[i] = limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, data.Products[i].ID),
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
	return limits.Load(lim)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if _, err := e.CurrencyPairs.IsPairEnabled(cp, a); err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	return tradeBaseURL + cp.Upper().String(), nil
}

// fetchFutures is a helper function for GetLatestFundingRates and GetFuturesContractDetails that calls the List Products endpoint twice, to get both expiring futures and perpetual futures
func (e *Exchange) fetchFutures(ctx context.Context) (*AllProducts, int, error) {
	products, err := e.GetAllProducts(ctx, 0, 0, "FUTURE", "", "", "", nil, false, false, false)
	if err != nil {
		return nil, 0, err
	}
	products2, err := e.GetAllProducts(ctx, 0, 0, "FUTURE", "PERPETUAL", "", "", nil, false, false, false)
	if err != nil {
		return nil, 0, err
	}
	perpStart := len(products.Products)
	products.Products = append(products.Products, products2.Products...)
	return products, perpStart, nil
}

// processFundingData is a helper function for GetAccountFundingHistory and GetWithdrawalsHistory, transforming the data returned by the Coinbase API into a format suitable for the exchange package
func (e *Exchange) processFundingData(accHistory []DeposWithdrData, cryptoHistory []TransactionData) ([]exchange.FundingHistory, error) {
	fundingData := make([]exchange.FundingHistory, len(accHistory)+len(cryptoHistory))
	for i := range accHistory {
		fundingData[i] = exchange.FundingHistory{
			ExchangeName: e.Name,
			Status:       accHistory[i].Status,
			TransferID:   accHistory[i].ID,
			Timestamp:    accHistory[i].PayoutAt,
			Currency:     accHistory[i].Amount.Currency.String(),
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
			ExchangeName: e.Name,
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
func (e *Exchange) iterativeGetAllOrders(ctx context.Context, productIDs currency.Pairs, orderType, orderSide, productType string, orderStatus []string, limit int32, startDate, endDate time.Time) ([]GetOrderResponse, error) {
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
	orderTypeSlice := []string{orderType}
	if orderType == "" {
		orderTypeSlice = nil
	}
	for hasNext {
		interResp, err := e.ListOrders(ctx, &ListOrdersReq{
			OrderStatus: orderStatus,
			OrderTypes:  orderTypeSlice,
			ProductIDs:  productIDs,
			ProductType: productType,
			OrderSide:   orderSide,
			Cursor:      cursor,
			Limit:       limit,
			StartDate:   startDate,
			EndDate:     endDate,
		})
		if err != nil {
			return nil, err
		}
		resp = append(resp, interResp.Orders...)
		hasNext = interResp.HasNext
		cursor = int64(interResp.Cursor)
	}
	return resp, nil
}

// getOrderRespToOrderDetail is a helper function used in GetOrderInfo, GetActiveOrders, and GetOrderHistory to convert data returned by the Coinbase API into a format suitable for the exchange package
func (e *Exchange) getOrderRespToOrderDetail(genOrderDetail *GetOrderResponse, pair currency.Pair, assetItem asset.Item) *order.Detail {
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
		Exchange:             e.GetName(),
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

// tickerHelper fetches the ticker for a given currency pair, used by UpdateTicker
func (e *Exchange) tickerHelper(ctx context.Context, name currency.Pair, assetType asset.Item) error {
	newTick := &ticker.Price{
		Pair:         name,
		ExchangeName: e.Name,
		AssetType:    assetType,
	}
	ticks, err := e.GetTicker(ctx, name, 1, time.Time{}, time.Time{}, false)
	if err != nil {
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
