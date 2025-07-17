package apexpro

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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
	"github.com/thrasher-corp/gocryptotrader/internal/utils/starkex"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for Apexpro
func (e *Exchange) SetDefaults() {
	e.Name = "Apexpro"
	e.Enabled = true
	e.Verbose = false
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true

	requestFmt := &currency.PairFormat{Uppercase: true, Delimiter: "-"}
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: "-"}
	err := e.SetAssetPairStore(asset.Futures, currency.PairStore{
		RequestFormat: requestFmt,
		ConfigFormat:  configFmt,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.StarkConfig, err = starkex.NewStarkExConfig()
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpotSupplementary:      apexproAPIURL,
		exchange.RestSpot:                   apexproAPIURL,
		exchange.WebsocketSpot:              apexProWebsocket,
		exchange.WebsocketSpotSupplementary: apexProPrivateWebsocket,

		exchange.RestFutures: apexProOmniAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.NetworkID = 1 // 1 for Main Net
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (e *Exchange) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		e.SetEnabled(false)
		return nil
	}
	err = e.SetupDefaults(exch)
	if err != nil {
		return err
	}
	wsRunningEndpoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(
		&websocket.ManagerSetup{
			ExchangeConfig:        exch,
			DefaultURL:            apexProWebsocket,
			RunningURL:            wsRunningEndpoint,
			Connector:             e.WsConnect,
			Subscriber:            e.Subscribe,
			Unsubscriber:          e.Unsubscribe,
			GenerateSubscriptions: e.GenerateDefaultSubscriptions,
			Features:              &e.Features.Supports.WebsocketCapabilities,
		})
	if err != nil {
		return err
	}
	err = e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  apexProWebsocket,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  apexProPrivateWebsocket,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Authenticated:        true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	configs, err := e.GetAllSymbolsConfigDataV1(ctx)
	if err != nil {
		return nil, err
	}
	// Storing the configuration values for later use.
	e.SymbolsConfig = configs

	tradablePairs := make(currency.Pairs, 0, len((configs.Data.PerpetualContract)))
	for a := range configs.Data.PerpetualContract {
		if !configs.Data.PerpetualContract[a].EnableTrade {
			continue
		}
		cp, err := currency.NewPairFromString(configs.Data.PerpetualContract[a].Symbol)
		if err != nil {
			return nil, err
		}
		tradablePairs = append(tradablePairs, cp)
	}
	return tradablePairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Futures)
	if err != nil {
		return err
	}
	return e.UpdatePairs(pairs, asset.Futures, true, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	pairFormat, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	tick, err := e.GetTickerDataV3(ctx, pairFormat.Format(p))
	if err != nil {
		return nil, err
	}
	if len(tick) == 0 {
		return nil, ticker.ErrTickerNotFound
	}
	tickerPrice := &ticker.Price{
		Last:         tick[0].LastPrice.Float64(),
		High:         tick[0].HighPrice24H.Float64(),
		Low:          tick[0].LowPrice24H.Float64(),
		Volume:       tick[0].Volume24H.Float64(),
		Pair:         p.Format(pairFormat),
		ExchangeName: e.Name,
		AssetType:    assetType,
	}
	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// FetchTicker returns the ticker for a currency pair
func (e *Exchange) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(e.Name, p, assetType)
	if err != nil {
		return e.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (e *Exchange) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	ob, err := orderbook.Get(e.Name, pair, assetType)
	if err != nil {
		return e.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	pairFormat, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return nil, err
	}
	orderbookNew, err := e.GetMarketDepthV3(ctx, pairFormat.Format(pair), 1000)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              pair,
		Asset:             assetType,
		ValidateOrderbook: e.ValidateOrderbook,
	}
	book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Bids[x][1].Float64(),
			Price:  orderbookNew.Bids[x][0].Float64(),
		}
	}

	book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Asks[x][1].Float64(),
			Price:  orderbookNew.Asks[x][0].Float64(),
		}
	}
	err = book.Process()
	if err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (e *Exchange) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	accountInfo, err := e.GetUserAccountDataV3(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	spotSubAccount := account.SubAccount{
		AssetType:  assetType,
		Currencies: []account.Balance{},
	}
	for a := range accountInfo.ContractWallets {
		spotSubAccount.Currencies = append(spotSubAccount.Currencies, account.Balance{
			Currency: currency.NewCode(accountInfo.ContractWallets[a].Asset),
			Total:    accountInfo.ContractWallets[a].Balance.Float64(),
			Hold:     accountInfo.ContractWallets[a].PendingWithdrawAmount.Float64(),
		})
	}
	return account.Holdings{
		Exchange: e.Name,
		Accounts: []account.SubAccount{spotSubAccount},
	}, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (e *Exchange) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(e.Name, creds, assetType)
	if err != nil {
		return e.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	transfers, err := e.GetUserTransferDataV2(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, "", []string{}, 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(transfers.Transfers))
	for x := range transfers.Transfers {
		resp[x] = exchange.FundingHistory{
			ExchangeName: e.Name,
			Status:       resp[x].Status,
			Timestamp:    transfers.Transfers[x].UpdatedTime.Time(),
			Currency:     transfers.Transfers[x].CurrencyID,
			Amount:       transfers.Transfers[x].Amount.Float64(),
			Fee:          transfers.Transfers[x].Fee.Float64(),
			TransferType: transfers.Transfers[x].Type,
			CryptoTxID:   transfers.Transfers[x].ID,
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, _ currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.GetUserTransferDataV2(ctx, currency.EMPTYCODE, time.Time{}, time.Time{}, "WITHDRAW", []string{}, 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(withdrawals.Transfers))
	for x := range withdrawals.Transfers {
		resp[x] = exchange.WithdrawalHistory{
			Status:       withdrawals.Transfers[x].Status,
			Timestamp:    withdrawals.Transfers[x].UpdatedTime.Time(),
			Currency:     withdrawals.Transfers[x].CurrencyID,
			Amount:       withdrawals.Transfers[x].Amount.Float64(),
			TransferType: withdrawals.Transfers[x].Type,
			CryptoTxID:   withdrawals.Transfers[x].ID,
			Fee:          withdrawals.Transfers[x].Fee.Float64(),
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	if assetType != asset.Futures {
		return nil, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	pairFormat, err := e.GetPairFormat(asset.Futures, true)
	if err != nil {
		return nil, err
	}
	tradeData, err := e.GetNewestTradingDataV3(ctx, pairFormat.Format(p), 1000)
	if err != nil {
		return nil, err
	}
	var side order.Side
	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		side, err = order.StringToOrderSide(tradeData[0].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			CurrencyPair: p.Format(pairFormat),
			AssetType:    asset.Futures,
			Price:        tradeData[i].Price.Float64(),
			Amount:       tradeData[i].Volume.Float64(),
			Timestamp:    tradeData[i].TradeTime.Time(),
			Side:         side,
		}
	}
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	return e.GetSystemTimeV3(ctx)
}

// SubmitOrder submits a new order
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(e.GetTradingRequirements()); err != nil {
		return nil, err
	}
	orderResp, err := e.CreateOrderV2(ctx, &CreateOrderParams{
		Symbol:           s.Pair,
		Side:             s.Side.String(),
		OrderType:        orderTypeString(s.Type),
		Size:             s.Amount,
		Price:            s.Price,
		TriggerPrice:     s.TriggerPrice,
		ClientOrderID:    s.ClientOrderID,
		ReduceOnly:       s.ReduceOnly,
		TriggerPriceType: s.TriggerPriceType.String(),
		ClientID:         s.ClientID,
		TrailingPercent:  s.TrailingPercent,
	})
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(orderResp.ID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (e *Exchange) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	if ord.OrderID == "" && ord.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if ord.OrderID != "" {
		_, err := e.CancelPerpOrder(ctx, ord.OrderID)
		return err
	}
	_, err := e.CancelPerpOrderByClientOrderID(ctx, ord.ClientOrderID)
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := orderCancellation.Validate(); err != nil {
		return order.CancelAllResponse{}, err
	}
	var symbols []string
	if !orderCancellation.Pair.IsEmpty() {
		symbols = append(symbols, orderCancellation.Pair.String())
	}
	err := e.CancelAllOpenOrdersV3(ctx, symbols)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	return order.CancelAllResponse{Status: map[string]string{orderCancellation.OrderID: "success"}}, nil
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	orderDetail, err := e.GetOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	} else if orderDetail == nil {
		return nil, fmt.Errorf("%w, orderId: %s", order.ErrOrderNotFound, orderID)
	}
	oType, err := order.StringToOrderType(orderDetail.OrderType)
	if err != nil {
		return nil, err
	}
	oStatus, err := order.StringToOrderStatus(orderDetail.Status)
	if err != nil {
		return nil, err
	}
	oSide, err := order.StringToOrderSide(orderDetail.Side)
	if err != nil {
		return nil, err
	}
	cp, err := currency.NewPairFromString(orderDetail.Symbol)
	if err != nil {
		return nil, err
	}
	tif, err := order.StringToTimeInForce(orderDetail.TimeInForce)
	if err != nil {
		return nil, err
	}
	if orderDetail.PostOnly {
		tif |= order.PostOnly
	}
	return &order.Detail{
		TimeInForce:     tif,
		ReduceOnly:      orderDetail.ReduceOnly,
		Price:           orderDetail.Price.Float64(),
		Amount:          orderDetail.Size.Float64(),
		ContractAmount:  orderDetail.Size.Float64(),
		TriggerPrice:    orderDetail.TriggerPrice.Float64(),
		ExecutedAmount:  orderDetail.CumMatchFillSize.Float64(),
		RemainingAmount: orderDetail.Size.Float64() - orderDetail.TriggerPrice.Float64(),
		Fee:             orderDetail.Fee.Float64(),
		Exchange:        e.Name,
		OrderID:         orderDetail.ID,
		ClientOrderID:   orderDetail.ClientOrderID,
		AccountID:       orderDetail.AccountID,
		Type:            oType,
		Side:            oSide,
		Status:          oStatus,
		AssetType:       asset.Futures,
		LastUpdated:     orderDetail.UpdatedTime.Time(),
		Pair:            cp,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(_ context.Context, _ currency.Code, _, _ string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawalResponse, err := e.WithdrawAsset(ctx, &AssetWithdrawalParams{
		Amount:           withdrawRequest.Amount,
		ClientWithdrawID: withdrawRequest.ClientOrderID,
		EthereumAddress:  withdrawRequest.Crypto.Address,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   e.Name,
		ID:     withdrawalResponse.ID,
		Status: "success",
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	orders, err := e.GetOpenOrders(ctx)
	if err != nil {
		return nil, err
	}
	orderFilters := make(order.FilteredOrders, len(orders))
	for a := range orders {
		oType, err := order.StringToOrderType(orders[a].OrderType)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(orders[a].Status)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(orders[a].Side)
		if err != nil {
			return nil, err
		}
		cp, err := currency.NewPairFromString(orders[a].Symbol)
		if err != nil {
			return nil, err
		}
		tif, err := order.StringToTimeInForce(orders[a].TimeInForce)
		if err != nil {
			return nil, err
		}
		if orders[a].PostOnly {
			tif |= order.PostOnly
		}
		orderFilters[a] = order.Detail{
			TimeInForce:     tif,
			ReduceOnly:      orders[a].ReduceOnly,
			Price:           orders[a].Price.Float64(),
			Amount:          orders[a].Size.Float64(),
			ContractAmount:  orders[a].Size.Float64(),
			TriggerPrice:    orders[a].TriggerPrice.Float64(),
			ExecutedAmount:  orders[a].CumMatchFillSize.Float64(),
			RemainingAmount: orders[a].Size.Float64() - orders[a].TriggerPrice.Float64(),
			Fee:             orders[a].Fee.Float64(),
			Exchange:        e.Name,
			OrderID:         orders[a].ID,
			ClientOrderID:   orders[a].ClientOrderID,
			AccountID:       orders[a].AccountID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       asset.Futures,
			LastUpdated:     orders[a].UpdatedTime.Time(),
			Pair:            cp,
		}
	}
	return orderFilters, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	// getOrdersRequest.AssetType
	pairFormat, err := e.GetPairFormat(asset.Futures, true)
	if err != nil {
		return nil, err
	}
	getOrdersRequest.Pairs = getOrdersRequest.Pairs.Format(pairFormat)
	var symbol string
	if len(getOrdersRequest.Pairs) == 0 {
		symbol = getOrdersRequest.Pairs[0].String()
	}
	orderHistoryResponse, err := e.GetAllOrderHistory(ctx, symbol, getOrdersRequest.Side.String(), orderTypeString(getOrdersRequest.Type), "", "", getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0, 0)
	if err != nil {
		return nil, err
	}
	orderFilters := make(order.FilteredOrders, 0, len(orderHistoryResponse.Orders))
	for a := range orderHistoryResponse.Orders {
		cp, err := currency.NewPairFromString(orderHistoryResponse.Orders[a].Symbol)
		if err != nil {
			return nil, err
		}
		if len(getOrdersRequest.Pairs) > 0 && !getOrdersRequest.Pairs.Contains(cp, true) {
			continue
		}
		oType, err := order.StringToOrderType(orderHistoryResponse.Orders[a].OrderType)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(orderHistoryResponse.Orders[a].Status)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(orderHistoryResponse.Orders[a].Side)
		if err != nil {
			return nil, err
		}
		tif, err := order.StringToTimeInForce(orderHistoryResponse.Orders[a].TimeInForce)
		if err != nil {
			return nil, err
		}
		if orderHistoryResponse.Orders[a].PostOnly {
			tif |= order.PostOnly
		}
		orderFilters = append(orderFilters, order.Detail{
			TimeInForce:     tif,
			ReduceOnly:      orderHistoryResponse.Orders[a].ReduceOnly,
			Price:           orderHistoryResponse.Orders[a].Price.Float64(),
			Amount:          orderHistoryResponse.Orders[a].Size.Float64(),
			ContractAmount:  orderHistoryResponse.Orders[a].Size.Float64(),
			TriggerPrice:    orderHistoryResponse.Orders[a].TriggerPrice.Float64(),
			ExecutedAmount:  orderHistoryResponse.Orders[a].CumMatchFillSize.Float64(),
			RemainingAmount: orderHistoryResponse.Orders[a].Size.Float64() - orderHistoryResponse.Orders[a].TriggerPrice.Float64(),
			Fee:             orderHistoryResponse.Orders[a].Fee.Float64(),
			Exchange:        e.Name,
			OrderID:         orderHistoryResponse.Orders[a].ID,
			ClientOrderID:   orderHistoryResponse.Orders[a].ClientOrderID,
			AccountID:       orderHistoryResponse.Orders[a].AccountID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       asset.Futures,
			LastUpdated:     orderHistoryResponse.Orders[a].UpdatedTime.Time(),
			Pair:            cp,
		})
	}
	return orderFilters, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	switch feeBuilder.FeeType {
	case exchange.OfflineTradeFee:
		return feeBuilder.Amount * feeBuilder.PurchasePrice * 0.002, nil
	case exchange.CryptocurrencyTradeFee:
		userResp, err := e.GetUserAccountDataV3(ctx)
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			return userResp.ContractAccount.MakerFeeRate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
		}
		return userResp.ContractAccount.TakerFeeRate.Float64() * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	case exchange.CryptocurrencyWithdrawalFee:
		resp, err := e.GetFastAndCrossChainWithdrawalFeesV2(ctx, feeBuilder.Amount, "", feeBuilder.FiatCurrency)
		if err != nil {
			return 0, err
		}
		return resp.Fee.Float64(), nil
	}
	return 0, common.ErrNotYetImplemented
}

// ValidateAPICredentials validates current credentials used for wrapper
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountInfo(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, _ asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, asset.Futures, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(asset.Futures, true)
	if err != nil {
		return nil, err
	}
	candles, err := e.GetCandlestickChartDataV3(ctx, pairFormat.Format(pair), interval, start, end, 1000)
	if err != nil {
		return nil, err
	}
	for x := range candles {
		cp, err := currency.NewPairFromString(x)
		if err != nil {
			return nil, err
		}
		if !cp.Equal(pair) {
			continue
		}
		timeSeries := make([]kline.Candle, len(candles[x]))
		for p := range candles[x] {
			timeSeries[p] = kline.Candle{
				Time:   candles[x][p].Start.Time(),
				Open:   candles[x][p].Open.Float64(),
				High:   candles[x][p].High.Float64(),
				Low:    candles[x][p].Low.Float64(),
				Close:  candles[x][p].Close.Float64(),
				Volume: candles[x][p].Volume.Float64(),
			}
		}
		return req.ProcessResponse(timeSeries)
	}
	return nil, fmt.Errorf("%w for pair: %v", kline.ErrNoTimeSeriesDataToConvert, pair)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, _ asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, asset.Futures, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		candles, err := e.GetCandlestickChartDataV3(ctx, req.RequestFormatted.String(), interval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, 1000)
		if err != nil {
			return nil, err
		}
		for y := range candles {
			cp, err := currency.NewPairFromString(y)
			if err != nil {
				return nil, err
			}
			if !cp.Equal(pair) {
				continue
			}
			for p := range candles[y] {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   candles[y][p].Start.Time(),
					Open:   candles[y][p].Open.Float64(),
					High:   candles[y][p].High.Float64(),
					Low:    candles[y][p].Low.Float64(),
					Close:  candles[y][p].Close.Float64(),
					Volume: candles[y][p].Volume.Float64(),
				})
			}
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, _ asset.Item) ([]futures.Contract, error) {
	result, err := e.GetAllConfigDataV3(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, 0, len(result.ContractConfig.PerpetualContract))
	for x := range result.ContractConfig.PerpetualContract {
		var cp, underlying currency.Pair
		cp, err = currency.NewPairFromString(result.ContractConfig.PerpetualContract[x].Symbol)
		if err != nil {
			return nil, err
		}
		underlying, err = currency.NewPairFromStrings(result.ContractConfig.PerpetualContract[x].Symbol, "USD")
		if err != nil {
			return nil, err
		}
		resp = append(resp, futures.Contract{
			Exchange:             e.Name,
			Name:                 cp,
			Underlying:           underlying,
			Asset:                asset.Futures,
			StartDate:            result.ContractConfig.PerpetualContract[x].KlineStartTime.Time(),
			SettlementType:       futures.Linear,
			IsActive:             result.ContractConfig.PerpetualContract[x].EnableTrade,
			Type:                 futures.Perpetual,
			SettlementCurrencies: currency.Currencies{currency.USD},
		})
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, pair currency.Pair) (bool, error) {
	if a != asset.Futures {
		return false, futures.ErrNotFuturesAsset
	}
	if pair.IsEmpty() {
		return false, currency.ErrCurrencyPairEmpty
	}
	var contracts []PerpetualContractDetail
	if e.SymbolsConfig != nil {
		contracts = e.SymbolsConfig.Data.PerpetualContract
	} else {
		resp, err := e.GetAllSymbolsConfigDataV1(context.Background())
		if err != nil {
			return false, err
		}
		contracts = resp.Data.PerpetualContract
	}
	symbol := pair.String()
	for a := range contracts {
		if contracts[a].Symbol == symbol {
			return true, nil
		}
	}
	return false, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	pairFormat, err := e.GetPairFormat(asset.Futures, true)
	if err != nil {
		return nil, err
	}
	r.Pair = r.Pair.Format(pairFormat)
	tickerData, err := e.GetTickerDataV3(ctx, r.Pair.String())
	if err != nil {
		return nil, err
	}
	resp := make([]fundingrate.LatestRateResponse, 0, len(tickerData))
	for i := range tickerData {
		var cp currency.Pair
		var isEnabled bool
		cp, isEnabled, err = e.MatchSymbolCheckEnabled(tickerData[i].Symbol, r.Asset, false)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		} else if !isEnabled {
			continue
		}
		resp = append(resp, fundingrate.LatestRateResponse{
			Exchange:    e.Name,
			TimeChecked: time.Now(),
			Asset:       asset.Futures,
			Pair:        cp,
			PredictedUpcomingRate: fundingrate.Rate{
				Time: tickerData[i].NextFundingTime.Time(),
				Rate: decimal.NewFromFloat(tickerData[i].PredictedFundingRate.Float64()),
			},
			LatestRate: fundingrate.Rate{
				Rate: decimal.NewFromFloat(tickerData[i].FundingRate.Float64()),
			},
			TimeOfNextRate: tickerData[i].NextFundingTime.Time(),
		})
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, _ asset.Item) error {
	instrumentsInfo, err := e.GetAllConfigDataV3(ctx)
	if err != nil {
		return err
	}
	limits := make([]order.MinMaxLevel, 0, len(instrumentsInfo.ContractConfig.PerpetualContract))
	for x := range instrumentsInfo.ContractConfig.PerpetualContract {
		var pair currency.Pair
		pair, err = e.MatchSymbolWithAvailablePairs(instrumentsInfo.ContractConfig.PerpetualContract[x].Symbol, asset.Futures, false)
		if err != nil {
			log.Warnf(log.ExchangeSys, "%s unable to load limits for %v, pair data missing", e.Name, instrumentsInfo.ContractConfig.PerpetualContract[x].Symbol)
			continue
		}
		limits = append(limits, order.MinMaxLevel{
			Asset:                   asset.Futures,
			Pair:                    pair,
			MinimumBaseAmount:       instrumentsInfo.ContractConfig.PerpetualContract[x].MinOrderSize.Float64(),
			MaximumBaseAmount:       instrumentsInfo.ContractConfig.PerpetualContract[x].MaxOrderSize.Float64(),
			MaxTotalOrders:          instrumentsInfo.ContractConfig.PerpetualContract[x].MaxPositionSize.Int64(),
			MaxPrice:                instrumentsInfo.ContractConfig.PerpetualContract[x].MaxMarketPriceRange.Float64(),
			PriceStepIncrementSize:  instrumentsInfo.ContractConfig.PerpetualContract[x].TickSize.Float64(),
			AmountStepIncrementSize: instrumentsInfo.ContractConfig.PerpetualContract[x].StepSize.Float64(),
			QuoteStepIncrementSize:  instrumentsInfo.ContractConfig.PerpetualContract[x].IncrementalPositionValue.Float64(),
			MaximumQuoteAmount:      instrumentsInfo.ContractConfig.PerpetualContract[x].MaxPositionValue.Float64(),
		})
	}
	return e.LoadLimits(limits)
}

func orderTypeString(oType order.Type) string {
	switch oType {
	case order.StopLimit:
		return "STOP_LIMIT"
	case order.StopMarket:
		return "STOP_MARKET"
	case order.TakeProfit:
		return "TAKE_PROFIT_LIMIT"
	case order.TakeProfitMarket:
		return "TAKE_PROFIT_MARKET"
	default:
		return oType.String()
	}
}
