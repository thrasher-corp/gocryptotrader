package apexpro

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
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

	emptyDelimiter := &currency.PairFormat{Uppercase: true, Delimiter: ""}
	dashDeliimiter := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}

	err := e.SetAssetPairStore(asset.Spot, currency.PairStore{
		RequestFormat: emptyDelimiter,
		ConfigFormat:  dashDeliimiter,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = e.SetAssetPairStore(asset.PerpetualContract, currency.PairStore{
		RequestFormat: emptyDelimiter,
		ConfigFormat:  dashDeliimiter,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// The V3 market-data endpoints (ticker, depth, trades) expect RWA symbols without a
	// delimiter (e.g. AAPLUSDT), matching Spot and PerpetualContract, while pairs are
	// displayed and stored with a dash.
	err = e.SetAssetPairStore(asset.RealWorldAsset, currency.PairStore{
		RequestFormat: emptyDelimiter,
		ConfigFormat:  dashDeliimiter,
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
			FuturesCapabilities: exchange.FuturesCapabilities{
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportedViaTicker: true,
				},
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
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
					kline.IntervalCapacity{Interval: kline.EightHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 200,
			},
		},
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(rateLimits))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpotSupplementary:      apexProOmniAPIURL,
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
			ExchangeConfig:               exch,
			DefaultURL:                   apexProWebsocket,
			RunningURL:                   wsRunningEndpoint,
			Features:                     &e.Features.Supports.WebsocketCapabilities,
			UseMultiConnectionManagement: true,
		},
	)
	if err != nil {
		return err
	}
	err = e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                   apexProWebsocket,
		ResponseCheckTimeout:  exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      exch.WebsocketResponseMaxLimit,
		GenerateSubscriptions: e.GenerateDefaultSubscriptions,
		Handler: func(ctx context.Context, _ websocket.Connection, incoming []byte) error {
			return e.wsHandleData(ctx, incoming)
		},
		Connector:    e.WsConnect,
		Subscriber:   e.Subscribe,
		Unsubscriber: e.Unsubscribe,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  apexProPrivateWebsocket,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		Handler: func(ctx context.Context, _ websocket.Connection, incoming []byte) error {
			return e.wsHandleData(ctx, incoming)
		},
		Connector:                e.WsAuth,
		Subscriber:               e.Subscribe,
		Unsubscriber:             e.Unsubscribe,
		Authenticated:            true,
		SubscriptionsNotRequired: true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !e.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	configs, err := e.GetAllConfigDataV3(ctx)
	if err != nil {
		return nil, err
	}
	// Storing the configuration values for later use.
	e.SymbolsConfig = configs

	switch a {
	case asset.PerpetualContract:
		tradablePairs := make(currency.Pairs, 0, len(configs.ContractConfig.PerpetualContract))
		for a := range configs.ContractConfig.PerpetualContract {
			if !configs.ContractConfig.PerpetualContract[a].EnableTrade {
				continue
			}
			tradablePairs = append(tradablePairs, configs.ContractConfig.PerpetualContract[a].Symbol)
		}
		format, err := e.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		return tradablePairs.Format(format), nil
	case asset.Spot:
		tradablePairs := make(currency.Pairs, 0, len(configs.SpotConfig.Spot))
		for a := range configs.SpotConfig.Spot {
			cp, err := currency.NewPairFromString(configs.SpotConfig.Spot[a])
			if err != nil {
				return nil, err
			}
			tradablePairs = append(tradablePairs, cp)
		}
		format, err := e.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		return tradablePairs.Format(format), nil
	case asset.RealWorldAsset:
		tradablePairs := make(currency.Pairs, 0, len(configs.ContractConfig.StockContract))
		for a := range configs.ContractConfig.StockContract {
			cp, err := currency.NewPairFromString(configs.ContractConfig.StockContract[a].Symbol)
			if err != nil {
				return nil, err
			}
			tradablePairs = append(tradablePairs, cp)
		}
		format, err := e.GetPairFormat(a, true)
		if err != nil {
			return nil, err
		}
		return tradablePairs.Format(format), nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	for _, a := range e.GetAssetTypes(true) {
		pairs, err := e.FetchTradablePairs(ctx, a)
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, a, true); err != nil {
			return err
		}
	}
	return nil
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
	if err := ticker.ProcessTicker(e.tickerPriceFromData(tick[0], p.Format(pairFormat), assetType)); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// tickerPriceFromData converts a single TickerData payload into a ticker.Price.
func (e *Exchange) tickerPriceFromData(tick *TickerData, p currency.Pair, assetType asset.Item) *ticker.Price {
	return &ticker.Price{
		Last:         tick.LastPrice.Float64(),
		High:         tick.HighPrice24H.Float64(),
		Low:          tick.LowPrice24H.Float64(),
		Volume:       tick.Volume24H.Float64(),
		QuoteVolume:  tick.Turnover24H.Float64(),
		MarkPrice:    tick.MarkPrice.Float64(),
		IndexPrice:   tick.IndexPrice.Float64(),
		OpenInterest: tick.OpenInterest.Float64(),
		Pair:         p,
		ExchangeName: e.Name,
		AssetType:    assetType,
	}
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	pairs, err := e.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		return currency.ErrCurrencyPairsEmpty
	}
	pairFormat, err := e.GetPairFormat(assetType, true)
	if err != nil {
		return err
	}
	var errs error
	for _, p := range pairs {
		tick, err := e.GetTickerDataV3(ctx, pairFormat.Format(p))
		if err != nil {
			errs = common.AppendError(errs, err)
			continue
		}
		if len(tick) == 0 {
			continue
		}
		if err := ticker.ProcessTicker(e.tickerPriceFromData(tick[0], p.Format(pairFormat), assetType)); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
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
		Asks:              orderbookNew.Asks.Levels(),
		Bids:              orderbookNew.Bids.Levels(),
	}
	if err := book.Process(); err != nil {
		return nil, err
	}
	return orderbook.Get(e.Name, pair, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	switch assetType {
	case asset.Spot:
		accountInfo, err := e.GetUserAccountDataV3(ctx)
		if err != nil {
			return nil, err
		}
		spotSubAccounts := accounts.SubAccounts{}
		for a := range accountInfo.SpotWallets {
			tokenCcy := currency.NewCode(e.GetTokenByID(accountInfo.SpotWallets[a].TokenID))
			subAcct := accounts.NewSubAccount(assetType, accountInfo.SpotWallets[a].UserID)
			subAcct.Balances.Set(tokenCcy, accounts.Balance{
				Currency: tokenCcy,
				Total:    accountInfo.SpotWallets[a].Balance.Float64(),
				Hold: accountInfo.SpotWallets[a].PendingDepositAmount.Float64() +
					accountInfo.SpotWallets[a].PendingWithdrawAmount.Float64() +
					accountInfo.SpotWallets[a].PendingTransferOutAmount.Float64() +
					accountInfo.SpotWallets[a].PendingTransferInAmount.Float64(),
			})
			spotSubAccounts.Merge(subAcct)
		}
		return spotSubAccounts, nil
	case asset.PerpetualContract:
		accountInfo, err := e.GetUserAccountDataV3(ctx)
		if err != nil {
			return nil, err
		}
		return contractWalletBalances(assetType, accountInfo.ContractWallets), nil
	case asset.RealWorldAsset:
		rwaAccount, err := e.GetRWAAccountData(ctx)
		if err != nil {
			return nil, err
		}
		return contractWalletBalances(assetType, rwaAccount.ContractWallets), nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// contractWalletBalances converts a set of contract wallets into sub-account balances for the given asset.
func contractWalletBalances(assetType asset.Item, wallets []*ContractWallet) accounts.SubAccounts {
	subAccounts := accounts.SubAccounts{}
	for a := range wallets {
		subAcct := accounts.NewSubAccount(assetType, wallets[a].UserID)
		subAcct.Balances.Set(wallets[a].Asset, accounts.Balance{
			Currency: wallets[a].Asset,
			Total:    wallets[a].Balance.Float64(),
			Hold: wallets[a].PendingDepositAmount.Float64() +
				wallets[a].PendingWithdrawAmount.Float64() +
				wallets[a].PendingTransferOutAmount.Float64() +
				wallets[a].PendingTransferInAmount.Float64(),
		})
		subAccounts.Merge(subAcct)
	}
	return subAccounts
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
			Status:       transfers.Transfers[x].Status,
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
	if assetType != asset.PerpetualContract && assetType != asset.RealWorldAsset {
		return nil, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	pairFormat, err := e.GetPairFormat(assetType, true)
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
		side, err = order.StringToOrderSide(tradeData[i].Side)
		if err != nil {
			return nil, err
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			CurrencyPair: p.Format(pairFormat),
			AssetType:    assetType,
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
	params := &CreateOrderRequest{
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
	}
	var orderResp *OrderDetail
	var err error
	switch s.AssetType {
	case asset.RealWorldAsset:
		orderResp, err = e.CreateRWAOrder(ctx, params)
	default:
		orderResp, err = e.CreateOrderV3(ctx, params)
	}
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
	if err := e.CancelAllOpenOrdersV3(ctx, symbols); err != nil {
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
		RemainingAmount: orderDetail.Size.Float64() - orderDetail.CumMatchFillSize.Float64(),
		Fee:             orderDetail.Fee.Float64(),
		Exchange:        e.Name,
		OrderID:         orderDetail.ID,
		ClientOrderID:   orderDetail.ClientOrderID,
		AccountID:       orderDetail.AccountID,
		Type:            oType,
		Side:            oSide,
		Status:          oStatus,
		AssetType:       e.assetTypeFromSymbol(orderDetail.Symbol),
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
	withdrawalResponse, err := e.WithdrawAsset(ctx, &AssetWithdrawalRequest{
		Amount:           withdrawRequest.Amount,
		ClientWithdrawID: withdrawRequest.ClientOrderID,
		Timestamp:        time.Now(),
		EthereumAddress:  withdrawRequest.Crypto.Address,
		ToChainID:        withdrawRequest.Crypto.Chain,
		L2SourceTokenID:  withdrawRequest.Currency,
		L1TargetTokenID:  withdrawRequest.Currency,
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
			RemainingAmount: orders[a].Size.Float64() - orders[a].CumMatchFillSize.Float64(),
			Fee:             orders[a].Fee.Float64(),
			Exchange:        e.Name,
			OrderID:         orders[a].ID,
			ClientOrderID:   orders[a].ClientOrderID,
			AccountID:       orders[a].AccountID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       e.assetTypeFromSymbol(orders[a].Symbol),
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
	pairFormat, err := e.GetPairFormat(asset.PerpetualContract, true)
	if err != nil {
		return nil, err
	}
	getOrdersRequest.Pairs = getOrdersRequest.Pairs.Format(pairFormat)
	var symbol string
	if len(getOrdersRequest.Pairs) > 0 {
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
			AssetType:       e.assetTypeFromSymbol(orderHistoryResponse.Orders[a].Symbol),
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
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	pairFormat, err := e.GetPairFormat(a, true)
	if err != nil {
		return nil, err
	}
	requestSymbol := pairFormat.Format(pair)
	candles, err := e.GetCandlestickChartDataV3(ctx, requestSymbol, interval, start, end, 200)
	if err != nil {
		return nil, err
	}
	for x := range candles {
		// The response is keyed by the delimiter-less request symbol (e.g. AAPLUSDT), which
		// cannot be reliably split back into a pair for RWA stock symbols, so match it directly.
		if x != requestSymbol {
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
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	requestSymbol := req.RequestFormatted.String()
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		candles, err := e.GetCandlestickChartDataV3(ctx, requestSymbol, interval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time, 200)
		if err != nil {
			return nil, err
		}
		for y := range candles {
			// Match the delimiter-less response key directly; see GetHistoricCandles.
			if y != requestSymbol {
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
		var underlying currency.Pair
		underlying, err = currency.NewPairFromString(result.ContractConfig.PerpetualContract[x].SymbolDisplayName)
		if err != nil {
			return nil, err
		}
		resp = append(resp, futures.Contract{
			Exchange:           e.Name,
			Underlying:         underlying,
			Asset:              asset.PerpetualContract,
			Name:               result.ContractConfig.PerpetualContract[x].Symbol,
			StartDate:          result.ContractConfig.PerpetualContract[x].KlineStartTime.Time(),
			SettlementType:     futures.Linear,
			IsActive:           result.ContractConfig.PerpetualContract[x].EnableTrade,
			Type:               futures.Perpetual,
			SettlementCurrency: currency.USD,
		})
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, pair currency.Pair) (bool, error) {
	if a != asset.PerpetualContract {
		return false, futures.ErrNotFuturesAsset
	}
	if pair.IsEmpty() {
		return false, currency.ErrCurrencyPairEmpty
	}
	var contracts []*PerpetualContractDetail
	if e.SymbolsConfig != nil {
		contracts = e.SymbolsConfig.ContractConfig.PerpetualContract
	} else {
		resp, err := e.GetAllSymbolsConfigDataV1(context.Background())
		if err != nil {
			return false, err
		}
		contracts = resp.Data.PerpetualContract
	}
	for a := range contracts {
		if contracts[a].Symbol.Equal(pair) {
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
	if r.Asset != asset.PerpetualContract {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	pairFormat, err := e.GetPairFormat(asset.PerpetualContract, true)
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
			Asset:       asset.PerpetualContract,
			Pair:        cp,
			PredictedUpcomingRate: fundingrate.Rate{
				Time: tickerData[i].NextFundingTime,
				Rate: decimal.NewFromFloat(tickerData[i].PredictedFundingRate.Float64()),
			},
			LatestRate: fundingrate.Rate{
				Rate: decimal.NewFromFloat(tickerData[i].FundingRate.Float64()),
			},
			TimeOfNextRate: tickerData[i].NextFundingTime,
		})
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	switch a {
	case asset.PerpetualContract:
		var contracts []*PerpetualContractDetail
		if e.SymbolsConfig != nil {
			contracts = e.SymbolsConfig.ContractConfig.PerpetualContract
		} else {
			resp, err := e.GetAllSymbolsConfigDataV1(ctx)
			if err != nil {
				return err
			}
			contracts = resp.Data.PerpetualContract
		}
		ls := make([]limits.MinMaxLevel, len(contracts))
		for x, pContract := range contracts {
			ls[x] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, asset.PerpetualContract, pContract.Symbol),
				MinPrice:                pContract.TickSize.Float64(),
				PriceStepIncrementSize:  pContract.TickSize.Float64(),
				MinimumBaseAmount:       pContract.MinOrderSize.Float64(),
				MaximumBaseAmount:       pContract.MaxOrderSize.Float64(),
				AmountStepIncrementSize: pContract.StepSize.Float64(),
				QuoteStepIncrementSize:  pContract.IncrementalPositionValue.Float64(),
				MaximumQuoteAmount:      pContract.MaxPositionValue.Float64(),
				MarketMaxQty:            pContract.MaxOrderSize.Float64(),
				MaxTotalOrders:          pContract.MaxPositionSize.Int64(),
			}
		}
		return limits.Load(ls)
	case asset.RealWorldAsset:
		cfg := e.SymbolsConfig
		if cfg == nil {
			var err error
			cfg, err = e.GetAllConfigDataV3(ctx)
			if err != nil {
				return err
			}
		}
		contracts := cfg.ContractConfig.StockContract
		ls := make([]limits.MinMaxLevel, len(contracts))
		for x, sContract := range contracts {
			cp, err := currency.NewPairFromString(sContract.Symbol)
			if err != nil {
				return err
			}
			ls[x] = limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, asset.RealWorldAsset, cp),
				MinPrice:                sContract.TickSize.Float64(),
				PriceStepIncrementSize:  sContract.TickSize.Float64(),
				MinimumBaseAmount:       sContract.MinOrderSize.Float64(),
				MaximumBaseAmount:       sContract.MaxOrderSize.Float64(),
				AmountStepIncrementSize: sContract.StepSize.Float64(),
				QuoteStepIncrementSize:  sContract.IncrementalPositionValue.Float64(),
				MaximumQuoteAmount:      sContract.MaxPositionValue.Float64(),
				MarketMaxQty:            sContract.MaxOrderSize.Float64(),
				MaxTotalOrders:          sContract.MaxPositionSize.Int64(),
			}
		}
		return limits.Load(ls)
	default:
		return nil
	}
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, keys ...key.PairAsset) ([]futures.OpenInterest, error) {
	pairFormat, err := e.GetPairFormat(asset.PerpetualContract, true)
	if err != nil {
		return nil, err
	}
	var pairs currency.Pairs
	if len(keys) == 0 {
		pairs, err = e.GetEnabledPairs(asset.PerpetualContract)
		if err != nil {
			return nil, err
		}
	} else {
		for _, k := range keys {
			if k.Asset != asset.PerpetualContract {
				return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, k.Asset)
			}
			pairs = append(pairs, k.Pair())
		}
	}
	resp := make([]futures.OpenInterest, 0, len(pairs))
	for _, p := range pairs {
		tick, err := e.GetTickerDataV3(ctx, pairFormat.Format(p))
		if err != nil {
			return nil, err
		}
		if len(tick) == 0 {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.PerpetualContract, p),
			OpenInterest: tick[0].OpenInterest.Float64(),
		})
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if _, err := e.CurrencyPairs.IsPairEnabled(cp, a); err != nil {
		return "", err
	}
	pairFormat, err := e.GetPairFormat(a, true)
	if err != nil {
		return "", err
	}
	return "https://omni.apex.exchange/trade/" + pairFormat.Format(cp), nil
}

// GetAvailableTransferChains returns a list of supported transfer chains based on the supplied cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	cfg := e.SymbolsConfig
	if cfg == nil {
		var err error
		cfg, err = e.GetAllConfigDataV3(ctx)
		if err != nil {
			return nil, err
		}
	}
	if cfg.SpotConfig.MultiChain == nil {
		return nil, nil
	}
	chains := make([]string, 0, len(cfg.SpotConfig.MultiChain.Chains))
	for _, c := range cfg.SpotConfig.MultiChain.Chains {
		for _, t := range c.Tokens {
			if strings.EqualFold(t.Token, cryptocurrency.String()) {
				chains = append(chains, c.Chain)
				break
			}
		}
	}
	return chains, nil
}

// assetTypeFromSymbol return the asset item given a contract symbol using the cached V3 configuration.
// check whether the instrument is of asset.RealWorldAsset or asset.PerpetualContract.
func (e *Exchange) assetTypeFromSymbol(symbol string) asset.Item {
	if e.SymbolsConfig != nil {
		for _, sc := range e.SymbolsConfig.ContractConfig.StockContract {
			if strings.EqualFold(sc.Symbol, symbol) {
				return asset.RealWorldAsset
			}
		}
	}
	return asset.PerpetualContract
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
