package bittrex

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
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
func (b *Bittrex) GetDefaultConfig() (*config.Exchange, error) {
	b.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = b.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = b.BaseCurrencies

	err := b.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if b.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = b.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Bittrex
func (b *Bittrex) SetDefaults() {
	b.Name = "Bittrex"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
	}

	err := b.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:      true,
				TickerFetching:      true,
				KlineFetching:       true,
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				CancelOrders:        true,
				SubmitOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
				Subscribe:         true,
				Unsubscribe:       true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():  true,
					kline.FiveMin.Word(): true,
					kline.OneHour.Word(): true,
					kline.OneDay.Word():  true,
				},
			},
		},
	}

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(ratePeriod, rateLimit)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.API.Endpoints = b.NewEndpoints()

	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:                   bittrexAPIRestURL,
		exchange.WebsocketSpot:              bittrexAPIWSURL,
		exchange.WebsocketSpotSupplementary: bittrexAPIWSNegotiationsURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.Websocket = stream.New()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *Bittrex) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		b.SetEnabled(false)
		return nil
	}
	err = b.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	// Websocket details setup below
	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            bittrexAPIWSURL, // Default ws endpoint so we can roll back via CLI if needed.
		RunningURL:            wsRunningEndpoint,
		Connector:             b.WsConnect,                                // Connector function outlined above.
		Subscriber:            b.Subscribe,                                // Subscriber function outlined above.
		Unsubscriber:          b.Unsubscribe,                              // Unsubscriber function outlined above.
		GenerateSubscriptions: b.GenerateDefaultSubscriptions,             // GenerateDefaultSubscriptions function outlined above.
		Features:              &b.Features.Supports.WebsocketCapabilities, // Defines the capabilities of the websocket outlined in supported features struct. This allows the websocket connection to be flushed appropriately if we have a pair/asset enable/disable change. This is outlined below.
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}
	// Sets up a new connection for the websocket, there are two separate connections denoted by the ConnectionSetup struct auth bool.
	return b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            wsRateLimit,
		// Authenticated        bool  sets if the connection is dedicated for an authenticated websocket stream which can be accessed from the Websocket field variable AuthConn e.g. f.Websocket.AuthConn
	})
}

// Start starts the Bittrex go routine
func (b *Bittrex) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		b.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Bittrex wrapper
func (b *Bittrex) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()))
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
	restURL, err := b.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to check REST Spot URL. Err: %s",
			b.Name,
			err)
	}
	if restURL == bittrexAPIDeprecatedURL {
		err = b.API.Endpoints.SetRunning(exchange.RestSpot.String(), bittrexAPIRestURL)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update deprecated REST Spot URL. Err: %s",
				b.Name,
				err)
		}
		b.Config.API.Endpoints[exchange.RestSpot.String()] = bittrexAPIRestURL
		log.Warnf(log.ExchangeSys,
			"Deprecated %s REST URL updated from %s to %s", b.Name, bittrexAPIDeprecatedURL, bittrexAPIRestURL)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bittrex) FetchTradablePairs(ctx context.Context, asset asset.Item) ([]string, error) {
	// Bittrex only supports spot trading
	if !b.SupportsAsset(asset) {
		return nil, fmt.Errorf("asset type of %s is not supported by %s", asset, b.Name)
	}
	markets, err := b.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}

	var resp []string
	for x := range markets {
		if markets[x].Status != "ONLINE" {
			continue
		}
		resp = append(resp, markets[x].Symbol)
	}

	return resp, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bittrex) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}

	return b.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bittrex) UpdateTickers(ctx context.Context, a asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bittrex) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	formattedPair, err := b.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	t, err := b.GetTicker(ctx, formattedPair.String())
	if err != nil {
		return nil, err
	}

	s, err := b.GetMarketSummary(ctx, formattedPair.String())
	if err != nil {
		return nil, err
	}

	pair, err := currency.NewPairFromString(t.Symbol)
	if err != nil {
		return nil, err
	}

	tickerPrice := b.constructTicker(t, &s, pair, a)

	err = ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(b.Name, p, a)
}

// constructTicker constructs a ticker price from the underlying data
func (b *Bittrex) constructTicker(t TickerData, s *MarketSummaryData, pair currency.Pair, assetType asset.Item) *ticker.Price {
	return &ticker.Price{
		Pair:         pair,
		Last:         t.LastTradeRate,
		Bid:          t.BidRate,
		Ask:          t.AskRate,
		High:         s.High,
		Low:          s.Low,
		Volume:       s.Volume,
		QuoteVolume:  s.QuoteVolume,
		LastUpdated:  s.UpdatedAt,
		AssetType:    assetType,
		ExchangeName: b.Name,
	}
}

// FetchTicker returns the ticker for a currency pair
func (b *Bittrex) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	resp, err := ticker.GetTicker(b.Name, p, assetType)
	if err != nil {
		return b.UpdateTicker(ctx, p, assetType)
	}
	return resp, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Bittrex) FetchOrderbook(ctx context.Context, c currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	resp, err := orderbook.Get(b.Name, c, assetType)
	if err != nil {
		return b.UpdateOrderbook(ctx, c, assetType)
	}
	return resp, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bittrex) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	formattedPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	// Valid order book depths are 1, 25 and 500
	orderbookData, sequence, err := b.GetOrderbook(ctx,
		formattedPair.String(), orderbookDepth)
	if err != nil {
		return nil, err
	}

	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
		LastUpdateID:    sequence,
	}

	for x := range orderbookData.Bid {
		book.Bids = append(book.Bids,
			orderbook.Item{
				Amount: orderbookData.Bid[x].Quantity,
				Price:  orderbookData.Bid[x].Rate,
			},
		)
	}

	for x := range orderbookData.Ask {
		book.Asks = append(book.Asks,
			orderbook.Item{
				Amount: orderbookData.Ask[x].Quantity,
				Price:  orderbookData.Ask[x].Rate,
			},
		)
	}
	err = book.Process()
	if err != nil {
		return book, err
	}

	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (b *Bittrex) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var resp account.Holdings
	balanceData, err := b.GetBalances(ctx)
	if err != nil {
		return resp, err
	}

	var currencies []account.Balance
	for i := range balanceData {
		currencies = append(currencies, account.Balance{
			CurrencyName: currency.NewCode(balanceData[i].CurrencySymbol),
			Total:        balanceData[i].Total,
			Hold:         balanceData[i].Total - balanceData[i].Available,
			Free:         balanceData[i].Available,
		})
	}

	resp.Accounts = append(resp.Accounts, account.SubAccount{
		Currencies: currencies,
	})
	resp.Exchange = b.Name

	return resp, account.Process(&resp)
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Bittrex) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	resp, err := account.GetHoldings(b.Name, assetType)
	if err != nil {
		return b.UpdateAccountInfo(ctx, assetType)
	}
	return resp, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bittrex) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	var resp []exchange.FundHistory
	closedDepositData, err := b.GetClosedDeposits(ctx)
	if err != nil {
		return resp, err
	}
	openDepositData, err := b.GetOpenDeposits(ctx)
	if err != nil {
		return resp, err
	}
	// nolint: gocritic
	depositData := append(closedDepositData, openDepositData...)

	for x := range depositData {
		resp = append(resp, exchange.FundHistory{
			ExchangeName:    b.Name,
			Status:          depositData[x].Status,
			Description:     depositData[x].CryptoAddressTag,
			Timestamp:       depositData[x].UpdatedAt,
			Currency:        depositData[x].CurrencySymbol,
			Amount:          depositData[x].Quantity,
			TransferType:    "deposit",
			CryptoToAddress: depositData[x].CryptoAddress,
			CryptoTxID:      depositData[x].TxID,
		})
	}
	closedWithdrawalData, err := b.GetClosedWithdrawals(ctx)
	if err != nil {
		return resp, err
	}
	openWithdrawalData, err := b.GetOpenWithdrawals(ctx)
	if err != nil {
		return resp, err
	}
	// nolint: gocritic
	withdrawalData := append(closedWithdrawalData, openWithdrawalData...)

	for x := range withdrawalData {
		resp = append(resp, exchange.FundHistory{
			ExchangeName:    b.Name,
			Status:          withdrawalData[x].Status,
			Description:     withdrawalData[x].CryptoAddressTag,
			Timestamp:       depositData[x].UpdatedAt,
			Currency:        withdrawalData[x].CurrencySymbol,
			Amount:          withdrawalData[x].Quantity,
			Fee:             withdrawalData[x].TxCost,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalData[x].CryptoAddress,
			CryptoTxID:      withdrawalData[x].TxID,
			TransferID:      withdrawalData[x].ID,
		})
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bittrex) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Bittrex) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	formattedPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tradeData, err := b.GetMarketHistory(ctx, formattedPair.String())
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData {
		var side order.Side
		side, err = order.StringToOrderSide(tradeData[i].TakerSide)
		if err != nil {
			return nil, err
		}
		resp = append(resp, trade.Data{
			Exchange:     b.Name,
			TID:          tradeData[i].ID,
			CurrencyPair: formattedPair,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Rate,
			Amount:       tradeData[i].Quantity,
			Timestamp:    tradeData[i].ExecutedAt,
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
// Bittrex only reports recent trades
func (b *Bittrex) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *Bittrex) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return order.SubmitResponse{}, err
	}

	if s.Side == order.Ask {
		s.Side = order.Sell
	}

	if s.Side == order.Bid {
		s.Side = order.Buy
	}

	formattedPair, err := b.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return order.SubmitResponse{}, err
	}

	orderData, err := b.Order(ctx,
		formattedPair.String(),
		s.Side.String(),
		s.Type.String(),
		GoodTilCancelled,
		s.Price,
		s.Amount,
		0.0)
	if err != nil {
		return order.SubmitResponse{}, err
	}

	return order.SubmitResponse{
		IsOrderPlaced: true,
		OrderID:       orderData.ID,
	}, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bittrex) ModifyOrder(_ context.Context, _ *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bittrex) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if err := ord.Validate(ord.StandardCancel()); err != nil {
		return err
	}
	_, err := b.CancelExistingOrder(ctx, ord.ID)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Bittrex) CancelBatchOrders(_ context.Context, _ []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair, or cancels all orders for all
// pairs if no pair was specified
func (b *Bittrex) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var pair string
	if orderCancellation != nil {
		formattedPair, err := b.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
		pair = formattedPair.String()
	}
	orderData, err := b.CancelOpenOrders(ctx, pair)
	if err != nil {
		return order.CancelAllResponse{}, err
	}

	tempMap := make(map[string]string)
	for x := range orderData {
		if orderData[x].Result.Status == "CLOSED" {
			tempMap[orderData[x].ID] = "Success"
		}
	}
	resp := order.CancelAllResponse{
		Status: tempMap,
		Count:  int64(len(tempMap)),
	}
	return resp, nil
}

// GetOrderInfo returns information on a current open order
func (b *Bittrex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	orderData, err := b.GetOrder(ctx, orderID)
	if err != nil {
		return order.Detail{}, err
	}

	return b.ConstructOrderDetail(&orderData)
}

// ConstructOrderDetail constructs an order detail item from the underlying data
func (b *Bittrex) ConstructOrderDetail(orderData *OrderData) (order.Detail, error) {
	immediateOrCancel := false
	if orderData.TimeInForce == string(ImmediateOrCancel) {
		immediateOrCancel = true
	}

	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		return order.Detail{}, err
	}
	orderPair, err := currency.NewPairDelimiter(orderData.MarketSymbol,
		format.Delimiter)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"Exchange %v Func %v Order %v Could not parse currency pair %v",
			b.Name,
			"GetActiveOrders",
			orderData.ID,
			err)
	}
	orderType := order.Type(strings.ToUpper(orderData.Type))

	var orderStatus order.Status

	switch orderData.Status {
	case order.Open.String():
		switch orderData.FillQuantity {
		case 0:
			orderStatus = order.Open
		default:
			orderStatus = order.PartiallyFilled
		}
	case order.Closed.String():
		switch orderData.FillQuantity {
		case 0:
			orderStatus = order.Cancelled
		case orderData.Quantity:
			orderStatus = order.Filled
		default:
			orderStatus = order.PartiallyCancelled
		}
	}

	resp := order.Detail{
		ImmediateOrCancel: immediateOrCancel,
		Amount:            orderData.Quantity,
		ExecutedAmount:    orderData.FillQuantity,
		RemainingAmount:   orderData.Quantity - orderData.FillQuantity,
		Price:             orderData.Limit,
		Date:              orderData.CreatedAt,
		ID:                orderData.ID,
		Exchange:          b.Name,
		Type:              orderType,
		Pair:              orderPair,
		Status:            orderStatus,
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bittrex) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	depositAddr, err := b.GetCryptoDepositAddress(ctx, cryptocurrency.String())
	if err != nil {
		return nil, err
	}

	return &deposit.Address{
		Address: depositAddr.CryptoAddress,
		Tag:     depositAddr.CryptoAddressTag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bittrex) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	result, err := b.Withdraw(ctx,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name:   b.Name,
		ID:     result.ID,
		Status: result.Status,
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bittrex) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bittrex) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var currPair string
	if len(req.Pairs) == 1 {
		formattedPair, err := b.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = formattedPair.String()
	}

	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	orderData, sequence, err := b.GetOpenOrders(ctx, currPair)
	if err != nil {
		return nil, err
	}

	var resp []order.Detail
	for i := range orderData {
		pair, err := currency.NewPairDelimiter(orderData[i].MarketSymbol,
			format.Delimiter)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"Exchange %v Func %v Order %v Could not parse currency pair %v",
				b.Name,
				"GetActiveOrders",
				orderData[i].ID,
				err)
		}
		orderType := order.Type(strings.ToUpper(orderData[i].Type))

		orderSide, err := order.StringToOrderSide(orderData[i].Direction)
		if err != nil {
			log.Errorf(log.ExchangeSys, "GetActiveOrders - %s - cannot get order side - %s\n", b.Name, err.Error())
			continue
		}

		resp = append(resp, order.Detail{
			Amount:          orderData[i].Quantity,
			RemainingAmount: orderData[i].Quantity - orderData[i].FillQuantity,
			ExecutedAmount:  orderData[i].FillQuantity,
			Price:           orderData[i].Limit,
			Date:            orderData[i].CreatedAt,
			ID:              orderData[i].ID,
			Exchange:        b.Name,
			Type:            orderType,
			Side:            orderSide,
			Status:          order.Active,
			Pair:            pair,
		})
	}

	order.FilterOrdersByType(&resp, req.Type)
	order.FilterOrdersByTimeRange(&resp, req.StartTime, req.EndTime)
	order.FilterOrdersByCurrencies(&resp, req.Pairs)

	b.WsSequenceOrders = sequence
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bittrex) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if len(req.Pairs) == 0 {
		return nil, errors.New("at least one currency is required to fetch order history")
	}

	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	var resp []order.Detail
	for x := range req.Pairs {
		formattedPair, err := b.FormatExchangeCurrency(req.Pairs[x], req.AssetType)
		if err != nil {
			return nil, err
		}

		orderData, err := b.GetOrderHistoryForCurrency(ctx, formattedPair.String())
		if err != nil {
			return nil, err
		}

		for i := range orderData {
			pair, err := currency.NewPairDelimiter(orderData[i].MarketSymbol,
				format.Delimiter)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"Exchange %v Func %v Order %v Could not parse currency pair %v",
					b.Name,
					"GetOrderHistory",
					orderData[i].ID,
					err)
			}
			orderType := order.Type(strings.ToUpper(orderData[i].Type))

			orderSide, err := order.StringToOrderSide(orderData[i].Direction)
			if err != nil {
				log.Errorf(log.ExchangeSys, "GetActiveOrders - %s - cannot get order side - %s\n", b.Name, err.Error())
				continue
			}
			orderStatus, err := order.StringToOrderStatus(orderData[i].Status)
			if err != nil {
				log.Errorf(log.ExchangeSys, "GetActiveOrders - %s - cannot get order status - %s\n", b.Name, err.Error())
				continue
			}

			detail := order.Detail{
				Amount:          orderData[i].Quantity,
				ExecutedAmount:  orderData[i].FillQuantity,
				RemainingAmount: orderData[i].Quantity - orderData[i].FillQuantity,
				Price:           orderData[i].Limit,
				Date:            orderData[i].CreatedAt,
				CloseTime:       orderData[i].ClosedAt,
				ID:              orderData[i].ID,
				Exchange:        b.Name,
				Type:            orderType,
				Side:            orderSide,
				Status:          orderStatus,
				Fee:             orderData[i].Commission,
				Pair:            pair,
			}
			detail.InferCostsAndTimes()
			resp = append(resp, detail)
		}

		order.FilterOrdersByType(&resp, req.Type)
		order.FilterOrdersByTimeRange(&resp, req.StartTime, req.EndTime)
		order.FilterOrdersByCurrencies(&resp, req.Pairs)
	}

	return resp, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bittrex) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !b.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(ctx, feeBuilder)
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Bittrex) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to string
// Overrides Base function
func (b *Bittrex) FormatExchangeKlineInterval(in kline.Interval) string {
	switch in {
	case kline.OneMin:
		return "MINUTE_1"
	case kline.FiveMin:
		return "MINUTE_5"
	case kline.OneHour:
		return "HOUR_1"
	case kline.OneDay:
		return "DAY_1"
	default:
		return "notfound"
	}
}

// GetHistoricCandles returns candles between a time period for a set time interval
// Candles set size returned by Bittrex depends on interval length:
// - 1m interval: candles for 1 day (0:00 - 23:59)
// - 5m interval: candles for 1 day (0:00 - 23:55)
// - 1 hour interval: candles for 31 days
// - 1 day interval: candles for 366 days
// This implementation rounds returns candles up to the next interval or to the end
// time (whichever comes first)
func (b *Bittrex) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := b.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	candleInterval := b.FormatExchangeKlineInterval(interval)
	if candleInterval == "notfound" {
		return kline.Item{}, errors.New("invalid interval")
	}

	formattedPair, err := b.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: b.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	year, month, day := start.Date()
	curYear, curMonth, curDay := time.Now().Date()

	getHistoric := false // nolint:ifshort,nolintlint // false positive and triggers only on Windows
	getRecent := false   // nolint:ifshort,nolintlint // false positive and triggers only on Windows

	switch interval {
	case kline.OneMin, kline.FiveMin:
		if time.Since(start) > 24*time.Hour {
			getHistoric = true
		}
		if year >= curYear && month >= curMonth && day >= curDay {
			getRecent = true
		}
	case kline.OneHour:
		if time.Since(start) > 31*24*time.Hour {
			getHistoric = true
		}
		if year >= curYear && month >= curMonth {
			getRecent = true
		}
	case kline.OneDay:
		if time.Since(start) > 366*24*time.Hour {
			getHistoric = true
		}
		if year >= curYear {
			getRecent = true
		}
	}

	var ohlcData []CandleData
	if getHistoric {
		var historicData []CandleData
		historicData, err = b.GetHistoricalCandles(ctx,
			formattedPair.String(),
			b.FormatExchangeKlineInterval(interval),
			"TRADE",
			year,
			int(month),
			day)
		if err != nil {
			return kline.Item{}, err
		}
		ohlcData = append(ohlcData, historicData...)
	}
	if getRecent {
		var recentData []CandleData
		recentData, err = b.GetRecentCandles(ctx,
			formattedPair.String(),
			b.FormatExchangeKlineInterval(interval),
			"TRADE")
		if err != nil {
			return kline.Item{}, err
		}
		ohlcData = append(ohlcData, recentData...)
	}

	for x := range ohlcData {
		timestamp := ohlcData[x].StartsAt
		if timestamp.Before(start) || timestamp.After(end) {
			continue
		}
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   timestamp,
			Open:   ohlcData[x].Open,
			High:   ohlcData[x].High,
			Low:    ohlcData[x].Low,
			Close:  ohlcData[x].Close,
			Volume: ohlcData[x].Volume,
		})
	}
	ret.SortCandlesByTimestamp(false)
	ret.RemoveDuplicates()
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *Bittrex) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{}, common.ErrNotYetImplemented
}
