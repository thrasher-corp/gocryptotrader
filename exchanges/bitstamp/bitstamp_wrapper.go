package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

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
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets default for Bitstamp
func (e *Exchange) SetDefaults() {
	e.Name = "Bitstamp"
	e.Enabled = true
	e.Verbose = true
	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresClientID = true
	requestFmt := &currency.EMPTYFORMAT
	configFmt := &currency.PairFormat{
		Uppercase: true,
		Delimiter: currency.ForwardSlashDelimiter,
	}
	err := e.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TradeFetching:     true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
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
				CryptoDepositFee:  true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:     true,
				OrderbookFetching: true,
				Subscribe:         true,
				Unsubscribe:       true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals:  true,
				DateRanges: true,
			},
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
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
				),
				GlobalResultLimit: 1000,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(bitstampRateInterval, bitstampRequestRate, 1)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      bitstampAPIURL,
		exchange.WebsocketSpot: bitstampWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	e.Websocket = websocket.NewManager()
	e.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	e.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	e.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets configuration values to bitstamp
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

	wsURL, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            bitstampWSURL,
		RunningURL:            wsURL,
		Connector:             e.WsConnect,
		Subscriber:            e.Subscribe,
		Unsubscriber:          e.Unsubscribe,
		GenerateSubscriptions: e.generateSubscriptions,
		Features:              &e.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  e.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	symbols, err := e.GetTradingPairs(ctx)
	if err != nil {
		return nil, err
	}
	var pair currency.Pair
	pairs := make([]currency.Pair, 0, len(symbols))
	for x := range symbols {
		if symbols[x].Trading != "Enabled" {
			continue
		}
		pair, err = currency.NewPairFromString(symbols[x].Name)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	pairs, err := e.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	if err := e.UpdatePairs(pairs, asset.Spot, false); err != nil {
		return err
	}
	return e.EnsureOnePairEnabled()
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return common.ErrNotYetImplemented
	}
	symbols, err := e.GetTradingPairs(ctx)
	if err != nil {
		return err
	}
	l := make([]limits.MinMaxLevel, 0, len(symbols))
	for x, info := range symbols {
		if symbols[x].Trading != "Enabled" {
			continue
		}
		pair, err := currency.NewPairFromString(symbols[x].Name)
		if err != nil {
			return err
		}
		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
			PriceStepIncrementSize:  math.Pow10(-info.CounterDecimals),
			AmountStepIncrementSize: math.Pow10(-info.BaseDecimals),
			MinimumQuoteAmount:      info.MinimumOrder,
		})
	}
	if err := limits.Load(l); err != nil {
		return fmt.Errorf("%s Error loading exchange limits: %v", e.Name, err)
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tick, err := e.GetTicker(ctx, fPair.String(), false)
	if err != nil {
		return nil, err
	}

	err = ticker.ProcessTicker(&ticker.Price{
		Last:         tick.Last,
		High:         tick.High,
		Low:          tick.Low,
		Bid:          tick.Bid,
		Ask:          tick.Ask,
		Volume:       tick.Volume,
		Open:         tick.Open,
		Pair:         fPair,
		LastUpdated:  tick.Timestamp.Time(),
		ExchangeName: e.Name,
		AssetType:    a,
	})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(e.Name, fPair, a)
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!e.AreCredentialsValid(ctx) || e.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return e.GetFee(ctx, feeBuilder)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
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
	fPair, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := e.GetOrderbook(ctx, fPair.String())
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Levels, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Level{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	filterOrderbookZeroBidPrice(book)

	book.Asks = make(orderbook.Levels, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Level{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		}
	}

	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(e.Name, fPair, assetType)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	accountBalance, err := e.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	for k, v := range accountBalance {
		subAccts[0].Balances.Set(currency.NewCode(k), accounts.Balance{
			Total: v.Balance,
			Hold:  v.Reserved,
			Free:  v.Available,
		})
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := e.GetWithdrawalRequests(ctx, 0)
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals))
	for i := range withdrawals {
		if c.IsEmpty() || c.Equal(withdrawals[i].Currency) {
			resp = append(resp, exchange.WithdrawalHistory{
				Status:          strconv.FormatInt(withdrawals[i].Status, 10),
				Timestamp:       withdrawals[i].Date.Time(),
				Currency:        withdrawals[i].Currency.String(),
				Amount:          withdrawals[i].Amount,
				TransferType:    strconv.FormatInt(withdrawals[i].Type, 10),
				CryptoToAddress: withdrawals[i].Address,
				CryptoTxID:      withdrawals[i].TransactionID,
			})
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tradeData, err := e.GetTransactions(ctx, p.String(), "")
	if err != nil {
		return nil, err
	}

	resp := make([]trade.Data, len(tradeData))
	for i := range tradeData {
		s := order.Buy
		if tradeData[i].Type == 1 {
			s = order.Sell
		}
		resp[i] = trade.Data{
			Exchange:     e.Name,
			TID:          strconv.FormatInt(tradeData[i].TradeID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         s,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeData[i].Date.Time(),
		}
	}

	err = e.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
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

	response, err := e.PlaceOrder(ctx,
		fPair.String(),
		s.Price,
		s.Amount,
		s.Side.IsLong(),
		s.Type == order.Market)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(strconv.FormatInt(response.ID, 10))
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}
	_, err = e.CancelExistingOrder(ctx, orderIDInt)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	success, err := e.CancelAllExistingOrders(ctx)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	if !success {
		err = errors.New("cancel all orders failed. Bitstamp provides no further information. Check order status to verify")
	}

	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns order information based on order ID
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	iOID, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	o, err := e.GetOrderStatus(ctx, iOID)
	if err != nil {
		return nil, err
	}

	th := make([]order.TradeHistory, len(o.Transactions))
	for i := range o.Transactions {
		th[i] = order.TradeHistory{
			TID:    strconv.FormatInt(o.Transactions[i].TradeID, 10),
			Price:  o.Transactions[i].Price,
			Fee:    o.Transactions[i].Fee,
			Amount: o.Transactions[i].ToCurrency,
		}
	}
	status, err := order.StringToOrderStatus(o.Status)
	if err != nil {
		return nil, err
	}
	return &order.Detail{
		RemainingAmount: o.AmountRemaining,
		OrderID:         o.ID,
		Date:            o.DateTime.Time(),
		Trades:          th,
		Status:          status,
	}, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	addr, err := e.GetCryptoDepositAddress(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	var tag string
	if addr.DestinationTag != 0 {
		tag = strconv.FormatInt(addr.DestinationTag, 10)
	}

	return &deposit.Address{
		Address: addr.Address,
		Tag:     tag,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.CryptoWithdrawal(ctx,
		withdrawRequest.Amount,
		withdrawRequest.Crypto.Address,
		withdrawRequest.Currency.String(),
		withdrawRequest.Crypto.AddressTag)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp.ID, 10),
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}

	resp, err := e.OpenBankWithdrawal(ctx, &OpenBankWithdrawalRequest{
		Amount:         withdrawRequest.Amount,
		Currency:       withdrawRequest.Currency,
		Name:           withdrawRequest.Fiat.Bank.AccountName,
		IBAN:           withdrawRequest.Fiat.Bank.IBAN,
		BIC:            withdrawRequest.Fiat.Bank.SWIFTCode,
		Address:        withdrawRequest.Fiat.Bank.BankAddress,
		PostalCode:     withdrawRequest.Fiat.Bank.BankPostalCode,
		City:           withdrawRequest.Fiat.Bank.BankPostalCity,
		Country:        withdrawRequest.Fiat.Bank.BankCountry,
		Comment:        withdrawRequest.Description,
		WithdrawalType: sepaWithdrawal,
	})
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp.ID, 10),
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := e.OpenInternationalBankWithdrawal(ctx, &OpenBankWithdrawalRequest{
		Amount:                withdrawRequest.Amount,
		Currency:              withdrawRequest.Currency,
		Name:                  withdrawRequest.Fiat.Bank.AccountName,
		IBAN:                  withdrawRequest.Fiat.Bank.IBAN,
		BIC:                   withdrawRequest.Fiat.Bank.SWIFTCode,
		Address:               withdrawRequest.Fiat.Bank.BankAddress,
		PostalCode:            withdrawRequest.Fiat.Bank.BankPostalCode,
		City:                  withdrawRequest.Fiat.Bank.BankPostalCity,
		Country:               withdrawRequest.Fiat.Bank.BankCountry,
		BankName:              withdrawRequest.Fiat.IntermediaryBankName,
		BankAddress:           withdrawRequest.Fiat.IntermediaryBankAddress,
		BankPostalCode:        withdrawRequest.Fiat.IntermediaryBankPostalCode,
		BankCity:              withdrawRequest.Fiat.IntermediaryBankCity,
		BankCountry:           withdrawRequest.Fiat.IntermediaryBankCountry,
		InternationalCurrency: withdrawRequest.Fiat.WireCurrency,
		Comment:               withdrawRequest.Description,
		WithdrawalType:        internationalWithdrawal,
	})
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp.ID, 10),
	}, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var currPair string
	if len(req.Pairs) != 1 {
		currPair = "all"
	} else {
		var fPair currency.Pair
		fPair, err = e.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = fPair.String()
	}

	resp, err := e.GetOpenOrders(ctx, currPair)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(resp))
	for i := range resp {
		orderSide := order.Buy
		if resp[i].Type == SellOrder {
			orderSide = order.Sell
		}

		var p currency.Pair
		if currPair == "all" {
			// Currency pairs are returned as format "currency_pair": "BTC/USD"
			// only when all is specified
			p, err = currency.NewPairFromString(resp[i].Currency)
			if err != nil {
				return nil, err
			}
		} else {
			p = req.Pairs[0]
		}

		orders[i] = order.Detail{
			Amount:   resp[i].Amount,
			OrderID:  strconv.FormatInt(resp[i].ID, 10),
			Price:    resp[i].Price,
			Type:     order.Limit,
			Side:     orderSide,
			Date:     resp[i].DateTime.Time(),
			Pair:     p,
			Exchange: e.Name,
		}
	}
	return req.Filter(e.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var currPair string
	if len(req.Pairs) == 1 {
		var fPair currency.Pair
		fPair, err = e.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = fPair.String()
	}

	format, err := e.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	resp, err := e.GetUserTransactions(ctx, currPair)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, 0, len(resp))
	for i := range resp {
		if resp[i].Type != MarketTrade {
			continue
		}
		var quoteCurrency, baseCurrency currency.Code

		switch {
		case resp[i].BTC > 0:
			baseCurrency = currency.BTC
		case resp[i].XRP > 0:
			baseCurrency = currency.XRP
		default:
			log.Warnf(log.ExchangeSys,
				"%s No base currency found for ID '%d'\n",
				e.Name,
				resp[i].OrderID)
		}

		switch {
		case resp[i].USD > 0:
			quoteCurrency = currency.USD
		case resp[i].EUR > 0:
			quoteCurrency = currency.EUR
		default:
			log.Warnf(log.ExchangeSys,
				"%s No quote currency found for orderID '%d'\n",
				e.Name,
				resp[i].OrderID)
		}

		var currPair currency.Pair
		if quoteCurrency.String() != "" && baseCurrency.String() != "" {
			currPair = currency.NewPairWithDelimiter(baseCurrency.String(),
				quoteCurrency.String(),
				format.Delimiter)
		}

		orders = append(orders, order.Detail{
			OrderID:  strconv.FormatInt(resp[i].OrderID, 10),
			Date:     resp[i].Date.Time(),
			Exchange: e.Name,
			Pair:     currPair,
		})
	}
	return req.Filter(e.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
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

	candles, err := e.OHLC(ctx,
		req.RequestFormatted.String(),
		req.Start,
		req.End,
		e.FormatExchangeKlineInterval(req.ExchangeInterval),
		strconv.FormatUint(req.RequestLimit, 10))
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, len(candles.Data.OHLCV))
	for x := range candles.Data.OHLCV {
		timestamp := candles.Data.OHLCV[x].Timestamp.Time()
		if timestamp.Before(req.Start) || timestamp.After(req.End) {
			continue
		}
		timeSeries = append(timeSeries, kline.Candle{
			Time:   timestamp,
			Open:   candles.Data.OHLCV[x].Open,
			High:   candles.Data.OHLCV[x].High,
			Low:    candles.Data.OHLCV[x].Low,
			Close:  candles.Data.OHLCV[x].Close,
			Volume: candles.Data.OHLCV[x].Volume,
		})
	}
	return req.ProcessResponse(timeSeries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles OHLCResponse
		candles, err = e.OHLC(ctx,
			req.RequestFormatted.String(),
			req.RangeHolder.Ranges[x].Start.Time,
			req.RangeHolder.Ranges[x].End.Time,
			e.FormatExchangeKlineInterval(req.ExchangeInterval),
			strconv.FormatUint(req.RequestLimit, 10),
		)
		if err != nil {
			return nil, err
		}

		for i := range candles.Data.OHLCV {
			timestamp := candles.Data.OHLCV[i].Timestamp.Time()
			if timestamp.Before(req.RangeHolder.Ranges[x].Start.Time) ||
				timestamp.After(req.RangeHolder.Ranges[x].End.Time) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   timestamp,
				Open:   candles.Data.OHLCV[i].Open,
				High:   candles.Data.OHLCV[i].High,
				Low:    candles.Data.OHLCV[i].Low,
				Close:  candles.Data.OHLCV[i].Close,
				Volume: candles.Data.OHLCV[i].Volume,
			})
		}
	}
	return req.ProcessResponse(timeSeries)
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (e *Exchange) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = ""
	return tradeBaseURL + cp.Lower().String() + "/", nil
}
