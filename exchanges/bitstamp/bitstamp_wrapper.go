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
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets default for Bitstamp
func (b *Bitstamp) SetDefaults() {
	b.Name = "Bitstamp"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.API.CredentialsValidator.RequiresClientID = true
	requestFmt := &currency.EMPTYFORMAT
	configFmt := &currency.PairFormat{
		Uppercase: true,
		Delimiter: currency.ForwardSlashDelimiter,
	}
	err := b.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	b.Features = exchange.Features{
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

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(bitstampRateInterval, bitstampRequestRate, 1)))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      bitstampAPIURL,
		exchange.WebsocketSpot: bitstampWSURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.Websocket = websocket.NewManager()
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets configuration values to bitstamp
func (b *Bitstamp) Setup(exch *config.Exchange) error {
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

	wsURL, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:        exch,
		DefaultURL:            bitstampWSURL,
		RunningURL:            wsURL,
		Connector:             b.WsConnect,
		Subscriber:            b.Subscribe,
		Unsubscriber:          b.Unsubscribe,
		GenerateSubscriptions: b.generateSubscriptions,
		Features:              &b.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL:                  b.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitstamp) FetchTradablePairs(ctx context.Context, _ asset.Item) (currency.Pairs, error) {
	symbols, err := b.GetTradingPairs(ctx)
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
func (b *Bitstamp) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := b.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	err = b.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
	if err != nil {
		return err
	}
	return b.EnsureOnePairEnabled()
}

// UpdateOrderExecutionLimits sets exchange execution order limits for an asset type
func (b *Bitstamp) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if a != asset.Spot {
		return common.ErrNotYetImplemented
	}
	symbols, err := b.GetTradingPairs(ctx)
	if err != nil {
		return err
	}
	limits := make([]order.MinMaxLevel, 0, len(symbols))
	for x, info := range symbols {
		if symbols[x].Trading != "Enabled" {
			continue
		}
		pair, err := currency.NewPairFromString(symbols[x].Name)
		if err != nil {
			return err
		}
		limits = append(limits, order.MinMaxLevel{
			Asset:                   a,
			Pair:                    pair,
			PriceStepIncrementSize:  math.Pow10(-info.CounterDecimals),
			AmountStepIncrementSize: math.Pow10(-info.BaseDecimals),
			MinimumQuoteAmount:      info.MinimumOrder,
		})
	}
	if err := b.LoadLimits(limits); err != nil {
		return fmt.Errorf("%s Error loading exchange limits: %v", b.Name, err)
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bitstamp) UpdateTickers(_ context.Context, _ asset.Item) error {
	return common.ErrFunctionNotSupported
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitstamp) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	tick, err := b.GetTicker(ctx, fPair.String(), false)
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
		ExchangeName: b.Name,
		AssetType:    a,
	})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(b.Name, fPair, a)
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitstamp) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!b.AreCredentialsValid(ctx) || b.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(ctx, feeBuilder)
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Book, error) {
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Book{
		Exchange:          b.Name,
		Pair:              p,
		Asset:             assetType,
		ValidateOrderbook: b.ValidateOrderbook,
	}
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := b.GetOrderbook(ctx, fPair.String())
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
	return orderbook.Get(b.Name, fPair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Bitstamp exchange
func (b *Bitstamp) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var response account.Holdings
	response.Exchange = b.Name
	accountBalance, err := b.GetBalance(ctx)
	if err != nil {
		return response, err
	}

	currencies := make([]account.Balance, 0, len(accountBalance))
	for k, v := range accountBalance {
		currencies = append(currencies, account.Balance{
			Currency: currency.NewCode(k),
			Total:    v.Balance,
			Hold:     v.Reserved,
			Free:     v.Available,
		})
	}
	response.Accounts = append(response.Accounts, account.SubAccount{
		AssetType:  assetType,
		Currencies: currencies,
	})

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	err = account.Process(&response, creds)
	if err != nil {
		return account.Holdings{}, err
	}

	return response, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitstamp) GetAccountFundingHistory(_ context.Context) ([]exchange.FundingHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitstamp) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	withdrawals, err := b.GetWithdrawalRequests(ctx, 0)
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
func (b *Bitstamp) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tradeData, err := b.GetTransactions(ctx, p.String(), "")
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
			Exchange:     b.Name,
			TID:          strconv.FormatInt(tradeData[i].TradeID, 10),
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         s,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Amount,
			Timestamp:    tradeData[i].Date.Time(),
		}
	}

	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bitstamp) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (b *Bitstamp) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(b.GetTradingRequirements()); err != nil {
		return nil, err
	}

	fPair, err := b.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	response, err := b.PlaceOrder(ctx,
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

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitstamp) ModifyOrder(_ context.Context, _ *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitstamp) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}

	orderIDInt, err := strconv.ParseInt(o.OrderID, 10, 64)
	if err != nil {
		return err
	}
	_, err = b.CancelExistingOrder(ctx, orderIDInt)
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Bitstamp) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitstamp) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	success, err := b.CancelAllExistingOrders(ctx)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	if !success {
		err = errors.New("cancel all orders failed. Bitstamp provides no further information. Check order status to verify")
	}

	return order.CancelAllResponse{}, err
}

// GetOrderInfo returns order information based on order ID
func (b *Bitstamp) GetOrderInfo(ctx context.Context, orderID string, _ currency.Pair, _ asset.Item) (*order.Detail, error) {
	iOID, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	o, err := b.GetOrderStatus(ctx, iOID)
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
func (b *Bitstamp) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	addr, err := b.GetCryptoDepositAddress(ctx, cryptocurrency)
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
func (b *Bitstamp) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := b.CryptoWithdrawal(ctx,
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
func (b *Bitstamp) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := b.OpenBankWithdrawal(ctx,
		withdrawRequest.Amount,
		withdrawRequest.Currency.String(),
		withdrawRequest.Fiat.Bank.AccountName,
		withdrawRequest.Fiat.Bank.IBAN,
		withdrawRequest.Fiat.Bank.SWIFTCode,
		withdrawRequest.Fiat.Bank.BankAddress,
		withdrawRequest.Fiat.Bank.BankPostalCode,
		withdrawRequest.Fiat.Bank.BankPostalCity,
		withdrawRequest.Fiat.Bank.BankCountry,
		withdrawRequest.Description,
		sepaWithdrawal)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp.ID, 10),
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (b *Bitstamp) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	resp, err := b.OpenInternationalBankWithdrawal(ctx,
		withdrawRequest.Amount,
		withdrawRequest.Currency.String(),
		withdrawRequest.Fiat.Bank.AccountName,
		withdrawRequest.Fiat.Bank.IBAN,
		withdrawRequest.Fiat.Bank.SWIFTCode,
		withdrawRequest.Fiat.Bank.BankAddress,
		withdrawRequest.Fiat.Bank.BankPostalCode,
		withdrawRequest.Fiat.Bank.BankPostalCity,
		withdrawRequest.Fiat.Bank.BankCountry,
		withdrawRequest.Fiat.IntermediaryBankName,
		withdrawRequest.Fiat.IntermediaryBankAddress,
		withdrawRequest.Fiat.IntermediaryBankPostalCode,
		withdrawRequest.Fiat.IntermediaryBankCity,
		withdrawRequest.Fiat.IntermediaryBankCountry,
		withdrawRequest.Fiat.WireCurrency,
		withdrawRequest.Description,
		internationalWithdrawal)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp.ID, 10),
	}, nil
}

// GetActiveOrders retrieves any orders that are active/open
func (b *Bitstamp) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var currPair string
	if len(req.Pairs) != 1 {
		currPair = "all"
	} else {
		var fPair currency.Pair
		fPair, err = b.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = fPair.String()
	}

	resp, err := b.GetOpenOrders(ctx, currPair)
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
			Exchange: b.Name,
		}
	}
	return req.Filter(b.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitstamp) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	var currPair string
	if len(req.Pairs) == 1 {
		var fPair currency.Pair
		fPair, err = b.FormatExchangeCurrency(req.Pairs[0], asset.Spot)
		if err != nil {
			return nil, err
		}
		currPair = fPair.String()
	}

	format, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		return nil, err
	}

	resp, err := b.GetUserTransactions(ctx, currPair)
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
				b.Name,
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
				b.Name,
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
			Exchange: b.Name,
			Pair:     currPair,
		})
	}
	return req.Filter(b.Name, orders), nil
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (b *Bitstamp) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitstamp) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}

	candles, err := b.OHLC(ctx,
		req.RequestFormatted.String(),
		req.Start,
		req.End,
		b.FormatExchangeKlineInterval(req.ExchangeInterval),
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
func (b *Bitstamp) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, req.Size())
	for x := range req.RangeHolder.Ranges {
		var candles OHLCResponse
		candles, err = b.OHLC(ctx,
			req.RequestFormatted.String(),
			req.RangeHolder.Ranges[x].Start.Time,
			req.RangeHolder.Ranges[x].End.Time,
			b.FormatExchangeKlineInterval(req.ExchangeInterval),
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
func (b *Bitstamp) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (b *Bitstamp) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetLatestFundingRates returns the latest funding rates data
func (b *Bitstamp) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *Bitstamp) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := b.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = ""
	return tradeBaseURL + cp.Lower().String() + "/", nil
}
