package coinbaseinternational

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (co *CoinbaseInternational) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	co.SetDefaults()
	exchCfg, err := co.GetStandardConfig()
	if err != nil {
		return nil, err
	}

	err = co.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if co.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := co.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for CoinbaseInternational
func (co *CoinbaseInternational) SetDefaults() {
	co.Name = "CoinbaseInternational"
	co.Enabled = true
	co.Verbose = true
	co.API.CredentialsValidator.RequiresKey = true
	co.API.CredentialsValidator.RequiresClientID = true
	co.API.CredentialsValidator.RequiresSecret = true
	co.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	err := co.SetGlobalPairsManager(
		&currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		&currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		asset.Spot, asset.Futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	co.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				AutoPairUpdates:        true,
				AccountBalance:         true,
				CryptoWithdrawal:       true,
				GetOrder:               true,
				GetOrders:              true,
				CancelOrders:           true,
				CancelOrder:            true,
				SubmitOrder:            true,
				ModifyOrder:            true,
				WithdrawalHistory:      true,
				TradeFee:               true,
				AccountInfo:            true,
				AuthenticatedEndpoints: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.OneMin},
				),
				GlobalResultLimit: 1000,
			},
		},
	}
	co.Requester, err = request.New(co.Name, common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	co.API.Endpoints = co.NewEndpoints()
	err = co.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      coinbaseInternationalAPIURL,
		exchange.WebsocketSpot: coinbaseinternationalWSAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	co.Websocket = stream.NewWebsocket()
	co.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	co.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	co.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (co *CoinbaseInternational) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		co.SetEnabled(false)
		return nil
	}
	err = co.SetupDefaults(exch)
	if err != nil {
		return err
	}
	wsRunningEndpoint, err := co.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = co.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            coinbaseinternationalWSAPIURL,
		RunningURL:            wsRunningEndpoint,
		Connector:             co.WsConnect,
		Subscriber:            co.Subscribe,
		Unsubscriber:          co.Unsubscribe,
		GenerateSubscriptions: co.GenerateDefaultSubscriptions,
		Features:              &co.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}
	return co.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  coinbaseinternationalWSAPIURL,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (co *CoinbaseInternational) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	if !co.SupportsAsset(a) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	instruments, err := co.GetInstruments(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make([]currency.Pair, 0, len(instruments))
	for x := range instruments {
		if a == asset.Spot && instruments[x].Type != "SPOT" {
			continue
		} else if a == asset.Futures && instruments[x].Type != "PERP" {
			continue
		}
		if instruments[x].TradingState != "TRADING" {
			continue
		}
		cp, err := currency.NewPairFromString(instruments[x].Symbol)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, cp)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (co *CoinbaseInternational) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := co.GetAssetTypes(false)
	for x := range assetTypes {
		pairs, err := co.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}

		err = co.UpdatePairs(pairs, assetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (co *CoinbaseInternational) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	if assetType != asset.Spot {
		return nil, fmt.Errorf("%w asset type %v", asset.ErrNotSupported, asset.Spot)
	}
	format, err := co.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	p = p.Format(format)
	tick, err := co.GetQuotePerInstrument(ctx, p.String(), "", "")
	if err != nil {
		return nil, err
	}
	err = ticker.ProcessTicker(&ticker.Price{
		High:         tick.LimitUp.Float64(),
		Low:          tick.LimitDown.Float64(),
		Bid:          tick.BestBidPrice.Float64(),
		BidSize:      tick.BestBidSize.Float64(),
		Ask:          tick.BestAskPrice.Float64(),
		AskSize:      tick.BestAskSize.Float64(),
		LastUpdated:  tick.Timestamp,
		Volume:       tick.TradeQty.Float64(),
		ExchangeName: co.Name,
		AssetType:    asset.Spot,
		Pair:         p.Format(format),
	})
	if err != nil {
		return nil, err
	}
	return ticker.GetTicker(co.Name, p, asset.Spot)
}

// UpdateTickers updates all currency pairs of a given asset type
func (co *CoinbaseInternational) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	if !co.SupportsAsset(assetType) {
		return fmt.Errorf("%w asset type %v", asset.ErrNotSupported, assetType)
	}
	var tick *QuoteInformation
	enabledPairs, err := co.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	for x := range enabledPairs {
		tick, err = co.GetQuotePerInstrument(ctx, enabledPairs[x].String(), "", "")
		if err != nil {
			return err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			High:         tick.LimitUp.Float64(),
			Low:          tick.LimitDown.Float64(),
			Bid:          tick.BestBidPrice.Float64(),
			BidSize:      tick.BestBidSize.Float64(),
			Ask:          tick.BestAskPrice.Float64(),
			AskSize:      tick.BestAskSize.Float64(),
			Open:         tick.MarkPrice.Float64(),
			Close:        tick.SettlementPrice.Float64(),
			LastUpdated:  tick.Timestamp,
			Volume:       tick.TradeQty.Float64() / tick.TradePrice.Float64(),
			QuoteVolume:  tick.TradeQty.Float64(),
			Pair:         enabledPairs[x],
			AssetType:    asset.Spot,
			ExchangeName: co.Name,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (co *CoinbaseInternational) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(co.Name, p, assetType)
	if err != nil {
		return co.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (co *CoinbaseInternational) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(co.Name, pair, assetType)
	if err != nil {
		return co.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (co *CoinbaseInternational) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if !co.SupportsAsset(assetType) {
		return nil, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	book := &orderbook.Base{
		Exchange:        co.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: co.CanVerifyOrderbook,
	}
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	format, err := co.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}
	orderbookNew, err := co.GetQuotePerInstrument(ctx, format.Format(pair), "", "")
	if err != nil {
		return book, err
	}
	book.Bids = orderbook.Tranches{{
		Amount: orderbookNew.BestBidSize.Float64(),
		Price:  orderbookNew.BestBidPrice.Float64(),
	}}
	book.Asks = orderbook.Tranches{{
		Amount: orderbookNew.BestAskSize.Float64(),
		Price:  orderbookNew.BestAskPrice.Float64(),
	}}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(co.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (co *CoinbaseInternational) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	if !co.SupportsAsset(assetType) {
		return account.Holdings{}, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	portfolios, err := co.GetAllUserPortfolios(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	holdings := account.Holdings{
		Exchange: co.Name,
		Accounts: make([]account.SubAccount, len(portfolios)),
	}
	var balances []PortfolioBalance
	for p := range portfolios {
		balances, err = co.ListPortfolioBalances(ctx, portfolios[p].PortfolioUUID, portfolios[p].PortfolioID)
		if err != nil {
			return account.Holdings{}, err
		}
		holdings.Accounts[p] = account.SubAccount{
			AssetType:  asset.Spot,
			ID:         portfolios[p].PortfolioID,
			Currencies: make([]account.Balance, len(balances)),
		}
		for b := range balances {
			holdings.Accounts[p].Currencies[b] = account.Balance{
				Currency:               currency.NewCode(balances[b].AssetName),
				Total:                  balances[b].Quantity.Float64(),
				Hold:                   balances[b].Hold.Float64(),
				Free:                   balances[b].Quantity.Float64() - balances[b].Hold.Float64(),
				AvailableWithoutBorrow: balances[b].MaxWithdrawAmount.Float64(),
			}
		}
	}
	return holdings, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (co *CoinbaseInternational) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	if !co.SupportsAsset(assetType) {
		return account.Holdings{}, fmt.Errorf("%w, asset type: %v", asset.ErrNotSupported, assetType)
	}
	creds, err := co.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(co.Name, creds, assetType)
	if err != nil {
		return co.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (co *CoinbaseInternational) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	history, err := co.ListMatchingTransfers(ctx, "", "", "", "", 0, 0, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(history.Results))
	for j := range history.Results {
		resp[j] = exchange.FundingHistory{
			ExchangeName: co.Name,
			CryptoTxID:   history.Results[j].TransferUUID,
			CryptoChain:  history.Results[j].NetworkName,
			Timestamp:    history.Results[j].CreatedAt,
			Status:       history.Results[j].TransferStatus,
			Currency:     history.Results[j].Asset,
			Amount:       history.Results[j].Amount,
			TransferType: history.Results[j].TransferType,
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (co *CoinbaseInternational) GetWithdrawalsHistory(ctx context.Context, _ currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	if a != asset.Spot {
		return nil, asset.ErrNotSupported
	}
	history, err := co.ListMatchingTransfers(ctx, "", "", "", "WITHDRAW", 0, 0, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(history.Results))
	for j := range history.Results {
		resp[j] = exchange.WithdrawalHistory{
			Status:          history.Results[j].TransferStatus,
			Timestamp:       history.Results[j].CreatedAt,
			Currency:        history.Results[j].Asset,
			Amount:          history.Results[j].Amount,
			TransferType:    history.Results[j].TransferType,
			CryptoTxID:      history.Results[j].TransferUUID,
			CryptoChain:     history.Results[j].NetworkName,
			CryptoToAddress: history.Results[j].ToPortfolio.ID,
		}
		if resp[j].CryptoToAddress == "" && history.Results[j].ToPortfolio.UUID != "" {
			resp[j].CryptoToAddress = history.Results[j].ToPortfolio.UUID
		} else if resp[j].CryptoToAddress == "" {
			resp[j].CryptoToAddress = history.Results[j].ToPortfolio.Name
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (co *CoinbaseInternational) GetRecentTrades(context.Context, currency.Pair, asset.Item) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (co *CoinbaseInternational) GetHistoricTrades(context.Context, currency.Pair, asset.Item, time.Time, time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetServerTime returns the current exchange server time.
func (co *CoinbaseInternational) GetServerTime(context.Context, asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (co *CoinbaseInternational) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(co.GetTradingRequirements()); err != nil {
		return nil, err
	}
	oType, err := OrderTypeString(s.Type)
	if err != nil {
		return nil, err
	}

	// TODO: possible value GTT is not represented by the order.Submit
	// therefore we could not represent 'GTT'
	var tif string
	if s.ImmediateOrCancel {
		tif = "IOC"
	} else {
		tif = "GTC"
	}
	response, err := co.CreateOrder(ctx, &OrderRequestParams{
		ClientOrderID: s.ClientOrderID,
		Side:          s.Side.String(),
		BaseSize:      s.Amount,
		Instrument:    s.Pair.String(),
		OrderType:     oType,
		Price:         s.Price,
		StopPrice:     s.TriggerPrice,
		PostOnly:      s.PostOnly,
		TimeInForce:   tif,
	})
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(strconv.FormatInt(response.OrderID, 10))
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (co *CoinbaseInternational) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	var orderID string
	switch {
	case action.OrderID != "":
		orderID = action.OrderID
	case action.ClientOrderID != "":
		orderID = action.ClientOrderID
	}
	response, err := co.ModifyOpenOrder(ctx, orderID, &ModifyOrderParam{
		ClientOrderID: action.ClientOrderID,
		Portfolio:     "",
		Price:         action.Price,
		StopPrice:     action.TriggerPrice,
		Size:          action.Amount,
	})
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = response.OrderID
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (co *CoinbaseInternational) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	err := ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	_, err = co.CancelTradeOrder(ctx, ord.OrderID, ord.ClientOrderID, ord.AccountID, "")
	return err
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (co *CoinbaseInternational) CancelBatchOrders(context.Context, []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (co *CoinbaseInternational) CancelAllOrders(ctx context.Context, action *order.Cancel) (order.CancelAllResponse, error) {
	if action.AssetType != asset.Spot {
		return order.CancelAllResponse{}, fmt.Errorf("%w asset type %v", asset.ErrNotSupported, action.AssetType)
	}
	if action.AccountID == "" {
		return order.CancelAllResponse{}, fmt.Errorf("%w %w (account ID)", request.ErrAuthRequestFailed, errMissingPortfolioID)
	}
	format, err := co.GetPairFormat(asset.Spot, true)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	canceled, err := co.CancelOrders(ctx, action.AccountID, "", format.Format(action.Pair))
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	response := order.CancelAllResponse{
		Status: make(map[string]string, len(canceled)),
		Count:  int64(len(canceled)),
	}
	for a := range canceled {
		response.Status[canceled[a].OrderID] = canceled[a].OrderStatus
	}
	return response, nil
}

// GetOrderInfo returns order information based on order ID
func (co *CoinbaseInternational) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, _ asset.Item) (*order.Detail, error) {
	resp, err := co.GetOrderDetail(ctx, orderID)
	if err != nil {
		return nil, err
	}
	oType, err := order.StringToOrderType(resp.Type)
	if err != nil {
		return nil, err
	}
	oSide, err := order.StringToOrderSide(resp.Side)
	if err != nil {
		return nil, err
	}
	oStatus, err := order.StringToOrderStatus(resp.OrderStatus)
	if err != nil {
		return nil, err
	}
	newPair, err := currency.NewPairFromString(resp.Symbol)
	if err != nil {
		return nil, err
	} else if !newPair.Equal(pair) {
		return nil, fmt.Errorf("expected pair %v, got %v", pair, newPair)
	}
	return &order.Detail{
		Price:                resp.Price,
		Amount:               resp.Size,
		Exchange:             co.Name,
		TriggerPrice:         resp.StopPrice,
		AverageExecutedPrice: resp.AveragePrice.Float64(),
		QuoteAmount:          resp.Size * resp.AveragePrice.Float64(),
		ExecutedAmount:       resp.ExecQty.Float64(),
		RemainingAmount:      resp.Size - resp.ExecQty.Float64(),
		Fee:                  resp.Fee.Float64(),
		OrderID:              resp.OrderID,
		ClientOrderID:        resp.ClientOrderID,
		Type:                 oType,
		Side:                 oSide,
		Status:               oStatus,
		AssetType:            asset.Spot,
		CloseTime:            resp.ExpireTime,
		Pair:                 pair,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (co *CoinbaseInternational) GetDepositAddress(context.Context, currency.Code, string, string) (*deposit.Address, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (co *CoinbaseInternational) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	info, err := co.GetSupportedNetworksPerAsset(ctx, cryptocurrency, "", "")
	if err != nil {
		return nil, err
	}
	availableChains := make([]string, len(info))
	for x := range info {
		availableChains[x] = info[x].NetworkName
	}
	return availableChains, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (co *CoinbaseInternational) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := co.WithdrawToCryptoAddress(ctx, &WithdrawCryptoParams{
		Portfolio:       withdrawRequest.PortfolioID,
		AssetIdentifier: withdrawRequest.Currency.String(),
		Amount:          withdrawRequest.Amount,
		Address:         withdrawRequest.Crypto.Address,
	})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name: co.Name,
		ID:   resp.Idem,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (co *CoinbaseInternational) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (co *CoinbaseInternational) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (co *CoinbaseInternational) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	var instrument string
	if len(getOrdersRequest.Pairs) == 1 {
		instrument = getOrdersRequest.Pairs[0].String()
	}
	response, err := co.GetOpenOrders(ctx, "", "", instrument, "", "", getOrdersRequest.StartTime, 0, 0)
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, 0, len(response.Results))
	for x := range response.Results {
		oType, err := order.StringToOrderType(response.Results[x].Type)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(response.Results[x].Side)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(response.Results[x].OrderStatus)
		if err != nil {
			return nil, err
		}
		var pair currency.Pair
		pair, err = currency.NewPairFromString(response.Results[x].Symbol)
		if err != nil {
			return nil, err
		}
		if len(getOrdersRequest.Pairs) != 0 && getOrdersRequest.Pairs.Contains(pair, true) {
			continue
		}
		orders = append(orders, order.Detail{
			Amount:               response.Results[x].Size,
			Price:                response.Results[x].Price,
			TriggerPrice:         response.Results[x].StopPrice,
			AverageExecutedPrice: response.Results[x].AveragePrice.Float64(),
			QuoteAmount:          response.Results[x].Size * response.Results[x].AveragePrice.Float64(),
			RemainingAmount:      response.Results[x].Size - response.Results[x].ExecQty.Float64(),
			OrderID:              response.Results[x].OrderID,
			ExecutedAmount:       response.Results[x].ExecQty.Float64(),
			Fee:                  response.Results[x].Fee.Float64(),
			ClientOrderID:        response.Results[x].ClientOrderID,
			CloseTime:            response.Results[x].ExpireTime,
			Exchange:             co.Name,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            asset.Spot,
			Pair:                 pair,
		})
	}
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (co *CoinbaseInternational) GetOrderHistory(_ context.Context, _ *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (co *CoinbaseInternational) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !co.AreCredentialsValid(ctx) && // TODO check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return co.GetFee(ctx, feeBuilder)
}

// ValidateAPICredentials validates current credentials used for wrapper
func (co *CoinbaseInternational) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := co.UpdateAccountInfo(ctx, assetType)
	return co.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (co *CoinbaseInternational) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	pair, err := co.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	req, err := co.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	result, err := co.GetAggregatedCandlesDataPerInstrument(ctx, req.Pair.String(), interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, len(result.Aggregations))
	for a := range result.Aggregations {
		timeSeries[a] = kline.Candle{
			Time:   result.Aggregations[a].Start,
			Open:   result.Aggregations[a].Open.Float64(),
			High:   result.Aggregations[a].High.Float64(),
			Low:    result.Aggregations[a].Low.Float64(),
			Close:  result.Aggregations[a].Close.Float64(),
			Volume: result.Aggregations[a].Volume.Float64(),
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (co *CoinbaseInternational) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	pair, err := co.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	req, err := co.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		result, err := co.GetAggregatedCandlesDataPerInstrument(ctx, req.Pair.String(), interval, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
		if err != nil {
			return nil, err
		}
		for a := range result.Aggregations {
			timeSeries = append(timeSeries, kline.Candle{
				Time:   result.Aggregations[a].Start,
				Open:   result.Aggregations[a].Open.Float64(),
				High:   result.Aggregations[a].High.Float64(),
				Low:    result.Aggregations[a].Low.Float64(),
				Close:  result.Aggregations[a].Close.Float64(),
				Volume: result.Aggregations[a].Volume.Float64(),
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (co *CoinbaseInternational) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !co.SupportsAsset(item) {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}
	contracts, err := co.GetInstruments(ctx)
	if err != nil {
		return nil, err
	}
	format, err := co.GetPairFormat(item, false)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.Contract, 0, len(contracts))
	for a := range contracts {
		if contracts[a].Type != "PERP" {
			continue
		}
		cp, err := currency.NewPairFromString(contracts[a].Symbol)
		if err != nil {
			return nil, err
		}
		underlying, err := currency.NewPairFromStrings(contracts[a].BaseAssetName, contracts[a].QuoteAssetName)
		if err != nil {
			return nil, err
		}
		resp = append(resp, futures.Contract{
			Exchange:             co.Name,
			Name:                 cp.Format(format),
			Underlying:           underlying,
			Asset:                item,
			IsActive:             contracts[a].TradingState == "TRADING",
			Status:               contracts[a].TradingState,
			Type:                 futures.Perpetual,
			SettlementCurrencies: currency.Currencies{currency.NewCode(contracts[a].QuoteAssetName)},
		})
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (co *CoinbaseInternational) GetLatestFundingRates(ctx context.Context, fr *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if fr == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if fr.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	result, err := co.GetHistoricalFundingRate(ctx, fr.Pair.String(), 0, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]fundingrate.LatestRateResponse, len(result.Results))
	for a := range result.Results {
		var cp currency.Pair
		var isEnabled bool
		cp, isEnabled, err = co.MatchSymbolCheckEnabled(result.Results[a].InstrumentID, fr.Asset, false)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		} else if !isEnabled {
			continue
		}
		resp[a] = fundingrate.LatestRateResponse{
			Exchange:    co.Name,
			TimeChecked: time.Now(),
			Asset:       fr.Asset,
			Pair:        cp,
			LatestRate: fundingrate.Rate{
				Time: result.Results[a].EventTime,
				Rate: decimal.NewFromFloat(result.Results[a].FundingRate.Float64()),
			},
		}
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, fr.Asset, fr.Pair)
	}
	return resp, nil
}

// UpdateOrderExecutionLimits sets exchange executions for a required asset type
func (co *CoinbaseInternational) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	instruments, err := co.GetInstruments(ctx)
	if err != nil {
		return fmt.Errorf("%s failed to load %s pair execution limits. Err: %s", co.Name, a, err)
	}
	format, err := co.GetPairFormat(a, false)
	if err != nil {
		return err
	}
	limits := make([]order.MinMaxLevel, len(instruments))
	for index := range instruments {
		pair, err := currency.NewPairFromString(instruments[index].Symbol)
		if err != nil {
			return err
		}
		pair = pair.Format(format)
		limits[index] = order.MinMaxLevel{
			Asset:                   a,
			Pair:                    pair,
			AmountStepIncrementSize: instruments[index].BaseIncrement.Float64(),
			QuoteStepIncrementSize:  instruments[index].QuoteIncrement.Float64(),
			MinimumQuoteAmount:      instruments[index].Quote.LimitDown.Float64(),
			MaximumQuoteAmount:      instruments[index].Quote.LimitUp.Float64(),
		}
	}
	return co.LoadLimits(limits)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (co *CoinbaseInternational) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	if cp.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	if a != asset.Spot {
		return "", fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	cp.Delimiter = currency.DashDelimiter
	return "https://international.coinbase.com/instrument/" + cp.Lower().String() + "?active=price", nil
}
