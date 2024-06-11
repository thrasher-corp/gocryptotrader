package coinbasepro

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
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
		Subscriptions: subscription.List{
			{Enabled: true, Channel: "heartbeat"},
			{Enabled: true, Channel: "level2_batch"}, // Other orderbook feeds require authentication; This is batched in 50ms lots
			{Enabled: true, Channel: "ticker"},
			{Enabled: true, Channel: "user", Authenticated: true},
			{Enabled: true, Channel: "matches"},
		},
	}
	c.Requester, err = request.New(c.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
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
	c.Websocket = stream.NewWebsocket()
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
		GenerateSubscriptions: c.generateSubscriptions,
		Features:              &c.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer: true,
		},
	})
	if err != nil {
		return err
	}
	return c.Websocket.SetupNewConnection(stream.ConnectionSetup{
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
	aString := a.Upper()
	if len(aString) == 7 {
		aString = aString[:6]
	}
	if verified {
		products, err = c.GetAllProducts(ctx, 2<<30-1, 0, aString, "", "", nil, verified)
		if err != nil {
			log.Warnf(log.ExchangeSys, warnAuth, err)
			verified = false
		}
	}
	if !verified {
		products, err = c.GetAllProducts(ctx, 2<<30-1, 0, aString, "", "", nil, verified)
		if err != nil {
			return nil, err
		}
	}
	pairs := make([]currency.Pair, len(products.Products))
	for x := range products.Products {
		if products.Products[x].TradingDisabled {
			continue
		}
		pair, err := currency.NewPairDelimiter(products.Products[x].ID, currency.DashDelimiter)
		if err != nil {
			return nil, err
		}
		pairs[x] = pair
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
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

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// coinbasepro exchange
func (c *CoinbasePro) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var (
		response       account.Holdings
		accountBalance []Account
		done           bool
		err            error
		cursor         string
		accountResp    *AllAccountsResponse
	)
	response.Exchange = c.Name
	for !done {
		accountResp, err = c.GetAllAccounts(ctx, 250, cursor)
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
	if err != nil {
		return c.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// UpdateTickers updates all currency pairs of a given asset type
func (c *CoinbasePro) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	verified, err := c.verificationCheck(ctx)
	if err != nil {
		return err
	}
	products, err := c.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	for x := range products {
		err = c.tickerHelper(ctx, products[x].String(), assetType, verified)
		if err != nil {
			return err
		}
	}
	return nil
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
	err = c.tickerHelper(ctx, fPair.String(), a, verified)
	if err != nil {
		return nil, err
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
	fPair, err := c.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        c.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: c.CanVerifyOrderbook,
	}
	var orderbookNew *ProductBook
	if verified {
		orderbookNew, err = c.GetProductBookV3(ctx, fPair.String(), 1000, true)
		if err != nil {
			log.Warnf(log.ExchangeSys, warnAuth, err)
			verified = false
		}
	}
	if !verified {
		orderbookNew, err = c.GetProductBookV3(ctx, fPair.String(), 1000, false)
		if err != nil {
			return book, err
		}
	}
	book.Bids = make(orderbook.Tranches, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Tranche{
			Amount: orderbookNew.Bids[x].Size,
			Price:  orderbookNew.Bids[x].Price,
		}
	}
	book.Asks = make(orderbook.Tranches, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Tranche{
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

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
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
	fundingData := c.processFundingData(accHistory, cryptoHistory)
	return fundingData, nil
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
	tempFundingData := c.processFundingData(accHistory, cryptoHistory)
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
func (c *CoinbasePro) GetRecentTrades(_ context.Context, _ currency.Pair, _ asset.Item) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (c *CoinbasePro) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
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
	resp, err := c.PlaceOrder(ctx, s.ClientOrderID, fPair.String(), s.Side.String(), stopDir, s.Type.String(), "",
		s.MarginType.Upper(), "", amount, s.Price, s.TriggerPrice, s.Leverage, s.PostOnly, s.EndTime)
	if err != nil {
		return nil, err
	}
	subResp, err := s.DeriveSubmitResponse(resp.OrderID)
	if err != nil {
		return nil, err
	}
	if s.RetrieveFees {
		time.Sleep(s.RetrieveFeeDelay)
		feeResp, err := c.GetOrderByID(ctx, resp.OrderID, s.ClientOrderID, "")
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
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (c *CoinbasePro) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	var status order.CancelBatchResponse
	var err error
	status.Status, _, err = c.cancelOrdersReturnMapAndCount(ctx, o)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (c *CoinbasePro) CancelAllOrders(_ context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrFunctionNotSupported
}

// GetOrderInfo returns order information based on order ID
func (c *CoinbasePro) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetItem asset.Item) (*order.Detail, error) {
	genOrderDetail, err := c.GetOrderByID(ctx, orderID, "", "")
	if err != nil {
		return nil, err
	}
	response, err := c.getOrderRespToOrderDetail(genOrderDetail, pair, assetItem)
	if err != nil {
		return nil, err
	}
	fillData, err := c.GetFills(ctx, orderID, "", "", time.Time{}, time.Now(), 2<<15-1)
	if err != nil {
		return nil, err
	}
	cursor := fillData.Cursor
	for cursor != "" {
		tempFillData, err := c.GetFills(ctx, orderID, "", cursor, time.Time{}, time.Now(), 2<<15-1)
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
			Price:     fillData.Fills[i].Price,
			Amount:    fillData.Fills[i].Size,
			Fee:       fillData.Fills[i].Commission,
			Exchange:  c.GetName(),
			TID:       fillData.Fills[i].TradeID,
			Side:      orderSide,
			Timestamp: fillData.Fills[i].TradeTime,
			Total:     fillData.Fills[i].Price * fillData.Fills[i].Size,
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
	resp, err := c.CreateAddress(ctx, targetWalletID, "")
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: resp.Data.Address,
		Tag:     resp.Data.Name,
		Chain:   resp.Data.Network,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.WalletID == "" {
		return nil, errWalletIDEmpty
	}
	t := time.Now().UnixNano()
	u := math.Float64bits(withdrawRequest.Amount)
	t ^= int64(u)
	message := strconv.FormatInt(t, 10)
	resp, err := c.SendMoney(ctx, "send", withdrawRequest.WalletID, withdrawRequest.Crypto.Address,
		withdrawRequest.Currency.String(), withdrawRequest.Description, message, "",
		withdrawRequest.Crypto.AddressTag, withdrawRequest.Amount, false, false)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{Name: resp.Data.Network.Name, ID: resp.Data.ID, Status: resp.Data.Status}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (c *CoinbasePro) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	if withdrawRequest.WalletID == "" {
		return nil, errWalletIDEmpty
	}
	paymentMethods, err := c.GetAllPaymentMethods(ctx)
	if err != nil {
		return nil, err
	}
	selectedWithdrawalMethod := PaymentMethodData{}
	for i := range paymentMethods.PaymentMethods {
		if withdrawRequest.Fiat.Bank.BankName == paymentMethods.PaymentMethods[i].Name {
			selectedWithdrawalMethod = paymentMethods.PaymentMethods[i]
			break
		}
	}
	if selectedWithdrawalMethod.ID == "" {
		return nil, fmt.Errorf("%w %v", errPayMethodNotFound, withdrawRequest.Fiat.Bank.BankName)
	}
	resp, err := c.FiatTransfer(ctx, withdrawRequest.WalletID, withdrawRequest.Currency.String(),
		selectedWithdrawalMethod.ID, withdrawRequest.Amount, true, FiatWithdrawal)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   selectedWithdrawalMethod.Name,
		ID:     resp.Data.ID,
		Status: resp.Data.Status,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
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
	pairIDs := req.Pairs.Strings()
	if len(pairIDs) == 0 {
		respOrders, err = c.iterativeGetAllOrders(ctx, "", req.Type.String(), req.Side.String(),
			req.AssetType.Upper(), ordStatus, 1000, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
	} else {
		for i := range pairIDs {
			interResp, err := c.iterativeGetAllOrders(ctx, pairIDs[i], req.Type.String(), req.Side.String(),
				req.AssetType.Upper(), ordStatus, 1000, req.StartTime, req.EndTime)
			if err != nil {
				return nil, err
			}
			respOrders = append(respOrders, interResp...)
		}
	}
	orders := make([]order.Detail, len(respOrders))
	for i := range respOrders {
		orderRec, err := c.getOrderRespToOrderDetail(&respOrders[i], req.Pairs[i], asset.Spot)
		if err != nil {
			return nil, err
		}
		orders[i] = *orderRec
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
	var p []string
	if len(req.Pairs) == 0 {
		p = make([]string, 1)
	} else {
		p = make([]string, len(req.Pairs))
		for i := range req.Pairs {
			req.Pairs[i], err = c.FormatExchangeCurrency(req.Pairs[i], req.AssetType)
			if err != nil {
				return nil, err
			}
			p[i] = req.Pairs[i].String()
		}
	}
	closedStatuses := []string{"FILLED", "CANCELLED", "EXPIRED", "FAILED"}
	openStatus := []string{"OPEN"}
	var ord []GetOrderResponse
	for i := range p {
		interOrd, err := c.iterativeGetAllOrders(ctx, p[i], req.Type.String(), req.Side.String(),
			req.AssetType.Upper(), closedStatuses, 2<<30-1, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		ord = append(ord, interOrd...)
		interOrd, err = c.iterativeGetAllOrders(ctx, p[i], req.Type.String(), req.Side.String(),
			req.AssetType.Upper(), openStatus, 2<<30-1, req.StartTime, req.EndTime)
		if err != nil {
			return nil, err
		}
		ord = append(ord, interOrd...)
	}
	orders := make([]order.Detail, len(ord))
	for i := range ord {
		singleOrder, err := c.getOrderRespToOrderDetail(&ord[i], req.Pairs[0], req.AssetType)
		if err != nil {
			return nil, err
		}
		orders[i] = *singleOrder
	}
	return req.Filter(c.Name, orders), nil
}

// GetHistoricCandles returns a set of candle between two time periods for a
// designated time period
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
		hist, err := c.candleHelper(ctx, req.RequestFormatted.String(), interval,
			req.RangeHolder.Ranges[x].Start.Time.Add(-time.Nanosecond),
			req.RangeHolder.Ranges[x].End.Time.Add(-time.Nanosecond), verified)
		if err != nil {
			return nil, err
		}
		timeSeries = append(timeSeries, hist...)
	}
	return req.ProcessResponse(timeSeries)
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (c *CoinbasePro) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := c.UpdateAccountInfo(ctx, assetType)
	return c.CheckTransientError(err)
}

// GetServerTime returns the current exchange server time.
func (c *CoinbasePro) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	st, err := c.GetV2Time(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return st.Data.ISO, nil
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
	products, err := c.fetchFutures(ctx, verified)
	if err != nil {
		return nil, err
	}
	funding := make([]fundingrate.LatestRateResponse, len(products.Products))
	for i := range products.Products {
		pair, err := currency.NewPairFromString(products.Products[i].ID)
		if err != nil {
			return nil, err
		}
		funRate := fundingrate.Rate{Time: products.Products[i].FutureProductDetails.PerpetualDetails.FundingTime,
			Rate: decimal.NewFromFloat(products.Products[i].FutureProductDetails.PerpetualDetails.FundingRate.Float64()),
		}
		funding[i] = fundingrate.LatestRateResponse{
			Exchange:    c.Name,
			Asset:       r.Asset,
			Pair:        pair,
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
	products, err := c.fetchFutures(ctx, verified)
	if err != nil {
		return nil, err
	}
	contracts := make([]futures.Contract, len(products.Products))
	for i := range products.Products {
		pair, err := currency.NewPairFromString(products.Products[i].ID)
		if err != nil {
			return nil, err
		}
		funRate := fundingrate.Rate{Time: products.Products[i].FutureProductDetails.PerpetualDetails.FundingTime,
			Rate: decimal.NewFromFloat(products.Products[i].FutureProductDetails.PerpetualDetails.FundingRate.Float64()),
		}
		contracts[i] = futures.Contract{
			Exchange:             c.Name,
			Name:                 pair,
			Asset:                item,
			EndDate:              products.Products[i].FutureProductDetails.ContractExpiry,
			IsActive:             !products.Products[i].IsDisabled,
			Status:               products.Products[i].Status,
			Type:                 futures.LongDated,
			SettlementCurrencies: []currency.Code{currency.NewCode(products.Products[i].QuoteCurrencyID)},
			Multiplier:           products.Products[i].BaseIncrement.Float64(),
			LatestRate:           funRate,
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
	aString := a.Upper()
	if len(aString) == 7 {
		aString = aString[:6]
	}
	if verified {
		data, err = c.GetAllProducts(ctx, 2<<30-1, 0, aString, "", "", nil, true)
		if err != nil {
			log.Warnf(log.ExchangeSys, warnAuth, err)
			verified = false
		}
	}
	if !verified {
		data, err = c.GetAllProducts(ctx, 2<<30-1, 0, aString, "", "", nil, false)
		if err != nil {
			return err
		}
	}
	limits := make([]order.MinMaxLevel, len(data.Products))
	for i := range data.Products {
		pair, err := currency.NewPairFromString(data.Products[i].ID)
		if err != nil {
			return err
		}
		limits[i] = order.MinMaxLevel{
			Pair:                    pair,
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

// fetchFutures is a helper function for FetchTradablePairs, GetLatestFundingRates, GetFuturesContractDetails,
// and UpdateOrderExecutionLimits that calls the List Products endpoint twice, to get both
// expiring futures and perpetual futures
func (c *CoinbasePro) fetchFutures(ctx context.Context, verified bool) (*AllProducts, error) {
	products, err := c.GetAllProducts(ctx, 2<<30-1, 0, "FUTURE", "", "", nil, verified)
	if err != nil {
		if verified {
			return c.fetchFutures(ctx, false)
		}
		return nil, err
	}
	products2, err := c.GetAllProducts(ctx, 2<<30-1, 0, "FUTURE", "PERPETUAL", "", nil, verified)
	if err != nil {
		if verified {
			return c.fetchFutures(ctx, false)
		}
		return nil, err
	}
	products.Products = append(products.Products, products2.Products...)
	return products, nil
}

// cancelOrdersReturnMapAndCount is a helper function for CancelBatchOrders, calling the appropriate Coinbase
// endpoint, and returning useful information
func (c *CoinbasePro) cancelOrdersReturnMapAndCount(ctx context.Context, o []order.Cancel) (status map[string]string, count int64, err error) {
	ordToCancel := len(o)
	if ordToCancel == 0 {
		return nil, 0, errOrderIDEmpty
	}
	status = make(map[string]string)
	ordIDSlice := make([]string, ordToCancel)
	for i := range o {
		err := o[i].Validate(o[i].StandardCancel())
		if err != nil {
			return nil, 0, err
		}
		ordIDSlice[i] = o[i].OrderID
		status[o[i].OrderID] = "Failed to cancel"
	}
	var resp CancelOrderResp
	for i := 0; i < ordToCancel; i += 100 {
		var tempOrdIDSlice []string
		if ordToCancel-i < 100 {
			tempOrdIDSlice = ordIDSlice[i:]
		} else {
			tempOrdIDSlice = ordIDSlice[i : i+100]
		}
		tempResp, err := c.CancelOrders(ctx, tempOrdIDSlice)
		if err != nil {
			return nil, 0, err
		}
		resp.Results = append(resp.Results, tempResp...)
	}
	var counter int64
	for i := range resp.Results {
		if resp.Results[i].Success {
			status[resp.Results[i].OrderID] = order.Cancelled.String()
			counter++
		}
	}
	return status, counter, nil
}

// processFundingData is a helper function for GetAccountFundingHistory and GetWithdrawalsHistory,
// transforming the data returned by the Coinbase API into a format suitable for the exchange package
func (c *CoinbasePro) processFundingData(accHistory []DeposWithdrData, cryptoHistory []TransactionData) []exchange.FundingHistory {
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

// iterativeGetAllOrders is a helper function used in GetActiveOrders and GetOrderHistory
// to repeatedly call GetAllOrders until all orders have been retrieved
func (c *CoinbasePro) iterativeGetAllOrders(ctx context.Context, productID, orderType, orderSide, productType string, orderStatus []string, limit int32, startDate, endDate time.Time) ([]GetOrderResponse, error) {
	var hasNext bool
	var resp []GetOrderResponse
	var cursor string
	for hasNext {
		interResp, err := c.GetAllOrders(ctx, productID, "", orderType, orderSide, cursor, productType, "", "", "",
			orderStatus, nil, limit, startDate, endDate)
		if err != nil {
			return nil, err
		}
		resp = append(resp, interResp.Orders...)
		hasNext = interResp.HasNext
		cursor = interResp.Cursor
	}
	return resp, nil
}

// formatExchangeKlineIntervalV3 is a helper function used in GetHistoricCandles and GetHistoricCandlesExtended
// to convert kline.Interval to the string format used by V3 of Coinbase's API
func formatExchangeKlineIntervalV3(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return granOneMin
	case kline.FiveMin:
		return granFiveMin
	case kline.FifteenMin:
		return granFifteenMin
	case kline.ThirtyMin:
		return granThirtyMin
	case kline.OneHour:
		return granOneHour
	case kline.TwoHour:
		return granTwoHour
	case kline.SixHour:
		return granSixHour
	case kline.OneDay:
		return granOneDay
	}
	return errIntervalNotSupported
}

// getOrderRespToOrderDetail is a helper function used in GetOrderInfo, GetActiveOrders, and GetOrderHistory
// to convert data returned by the Coinbase API into a format suitable for the exchange package
func (c *CoinbasePro) getOrderRespToOrderDetail(genOrderDetail *GetOrderResponse, pair currency.Pair, assetItem asset.Item) (*order.Detail, error) {
	var amount float64
	var quoteAmount float64
	var orderType order.Type
	var err error
	if genOrderDetail.OrderConfiguration.MarketMarketIOC != nil {
		err = stringToFloatPtr(&quoteAmount, genOrderDetail.OrderConfiguration.MarketMarketIOC.QuoteSize)
		if err != nil {
			return nil, err
		}
		err = stringToFloatPtr(&amount, genOrderDetail.OrderConfiguration.MarketMarketIOC.BaseSize)
		if err != nil {
			return nil, err
		}
		orderType = order.Market
	}
	var price float64
	var postOnly bool
	if genOrderDetail.OrderConfiguration.LimitLimitGTC != nil {
		err = stringToFloatPtr(&amount, genOrderDetail.OrderConfiguration.LimitLimitGTC.BaseSize)
		if err != nil {
			return nil, err
		}
		err = stringToFloatPtr(&price, genOrderDetail.OrderConfiguration.LimitLimitGTC.LimitPrice)
		if err != nil {
			return nil, err
		}
		postOnly = genOrderDetail.OrderConfiguration.LimitLimitGTC.PostOnly
		orderType = order.Limit
	}
	if genOrderDetail.OrderConfiguration.LimitLimitGTD != nil {
		err = stringToFloatPtr(&amount, genOrderDetail.OrderConfiguration.LimitLimitGTD.BaseSize)
		if err != nil {
			return nil, err
		}
		err = stringToFloatPtr(&price, genOrderDetail.OrderConfiguration.LimitLimitGTD.LimitPrice)
		if err != nil {
			return nil, err
		}
		postOnly = genOrderDetail.OrderConfiguration.LimitLimitGTD.PostOnly
		orderType = order.Limit
	}
	var triggerPrice float64
	if genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC != nil {
		err = stringToFloatPtr(&amount, genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC.BaseSize)
		if err != nil {
			return nil, err
		}
		err = stringToFloatPtr(&price, genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC.LimitPrice)
		if err != nil {
			return nil, err
		}
		err = stringToFloatPtr(&triggerPrice, genOrderDetail.OrderConfiguration.StopLimitStopLimitGTC.StopPrice)
		if err != nil {
			return nil, err
		}
		orderType = order.StopLimit
	}
	if genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD != nil {
		err = stringToFloatPtr(&amount, genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD.BaseSize)
		if err != nil {
			return nil, err
		}
		err = stringToFloatPtr(&price, genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD.LimitPrice)
		if err != nil {
			return nil, err
		}
		err = stringToFloatPtr(&triggerPrice, genOrderDetail.OrderConfiguration.StopLimitStopLimitGTD.StopPrice)
		if err != nil {
			return nil, err
		}
		orderType = order.StopLimit
	}
	var remainingAmount float64
	if !genOrderDetail.SizeInQuote {
		remainingAmount = amount - genOrderDetail.FilledSize
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
	response := order.Detail{
		ImmediateOrCancel:    genOrderDetail.OrderConfiguration.MarketMarketIOC != nil,
		PostOnly:             postOnly,
		Price:                price,
		Amount:               amount,
		TriggerPrice:         triggerPrice,
		AverageExecutedPrice: genOrderDetail.AverageFilledPrice,
		QuoteAmount:          quoteAmount,
		ExecutedAmount:       genOrderDetail.FilledSize,
		RemainingAmount:      remainingAmount,
		Cost:                 genOrderDetail.TotalValueAfterFees,
		Fee:                  genOrderDetail.TotalFees,
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
	return &response, nil
}

// stringToFloatPtr essentially calls ParseFloat, but leaves the float alone instead of erroring out
// if the string is empty.
func stringToFloatPtr(outgoing *float64, incoming string) error {
	if outgoing == nil {
		return errPointerNil
	}
	var err error
	if incoming != "" {
		*outgoing, err = strconv.ParseFloat(incoming, 64)
		return err
	}
	return nil
}

// VerificationCheck returns whether authentication support is enabled or not
func (c *CoinbasePro) verificationCheck(ctx context.Context) (bool, error) {
	_, err := c.GetCredentials(ctx)
	if err != nil {
		if errors.Is(err, exchange.ErrAuthenticationSupportNotEnabled) ||
			errors.Is(err, exchange.ErrCredentialsAreEmpty) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// TickerHelper fetches the ticker for a given currency pair, used by UpdateTickers and UpdateTicker
func (c *CoinbasePro) tickerHelper(ctx context.Context, name string, assetType asset.Item, verified bool) error {
	pair, err := currency.NewPairDelimiter(name, currency.DashDelimiter)
	if err != nil {
		return err
	}
	newTick := &ticker.Price{
		Pair:         pair,
		ExchangeName: c.Name,
		AssetType:    assetType,
	}
	var ticks *Ticker
	ticks, err = c.GetTicker(ctx, name, 1, time.Time{}, time.Time{}, verified)
	if err != nil {
		if verified {
			return c.tickerHelper(ctx, name, assetType, false)
		}
		return err
	}
	var last float64
	if len(ticks.Trades) != 0 {
		last = ticks.Trades[0].Price
	}
	newTick.Last = last
	newTick.Bid = ticks.BestBid.Float64()
	newTick.Ask = ticks.BestAsk.Float64()
	err = ticker.ProcessTicker(newTick)
	if err != nil {
		return err
	}
	return nil
}

// CandleHelper handles calling the candle function, and doing preliminary work on the data
func (c *CoinbasePro) candleHelper(ctx context.Context, pair string, granularity kline.Interval, start, end time.Time, verified bool) ([]kline.Candle, error) {
	var timeSeries []kline.Candle
	history, err := c.GetHistoricRates(ctx, pair, formatExchangeKlineIntervalV3(granularity), start, end, verified)
	if err != nil {
		if verified {
			return c.candleHelper(ctx, pair, granularity, start, end, false)
		}
		return nil, err
	}
	timeSeries = make([]kline.Candle, len(history))
	for x := range history {
		timeSeries[x] = kline.Candle{
			Time:   history[x].Start.Time(),
			Low:    history[x].Low,
			High:   history[x].High,
			Open:   history[x].Open,
			Close:  history[x].Close,
			Volume: history[x].Volume,
		}
	}
	return timeSeries, nil
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
