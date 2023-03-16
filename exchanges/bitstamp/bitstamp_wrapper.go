package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (b *Bitstamp) GetDefaultConfig() (*config.Exchange, error) {
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
					kline.OneMin,
					kline.ThreeMin,
					kline.FiveMin,
					kline.FifteenMin,
					kline.ThirtyMin,
					kline.OneHour,
					kline.TwoHour,
					kline.FourHour,
					kline.SixHour,
					kline.TwelveHour,
					kline.OneDay,
					kline.ThreeDay,
				),
				ResultLimit: 1000,
			},
		},
	}

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(bitstampRateInterval, bitstampRequestRate)))
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
	b.Websocket = stream.New()
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

	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             bitstampWSURL,
		RunningURL:             wsURL,
		Connector:              b.WsConnect,
		Subscriber:             b.Subscribe,
		Unsubscriber:           b.Unsubscribe,
		GenerateSubscriptions:  b.generateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &b.Features.Supports.WebsocketCapabilities,
	})
	if err != nil {
		return err
	}

	return b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		URL:                  b.Websocket.GetWebsocketURL(),
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
	})
}

// Start starts the Bitstamp go routine
func (b *Bitstamp) Start(wg *sync.WaitGroup) error {
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

// Run implements the Bitstamp wrapper
func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()))
		b.PrintEnabledPairs()
	}

	forceUpdate := false
	if !b.BypassConfigFormatUpgrades {
		format, err := b.GetPairFormat(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to get pair format. Err %s\n",
				b.Name,
				err)
			return
		}

		enabled, err := b.CurrencyPairs.GetPairs(asset.Spot, true)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to get enabled currencies. Err %s\n",
				b.Name,
				err)
			return
		}

		avail, err := b.CurrencyPairs.GetPairs(asset.Spot, false)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s failed to get available currencies. Err %s\n",
				b.Name,
				err)
			return
		}

		if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
			!common.StringDataContains(avail.Strings(), format.Delimiter) {
			var enabledPairs currency.Pairs
			enabledPairs, err = currency.NewPairsFromStrings([]string{
				currency.BTC.String() + format.Delimiter + currency.USD.String(),
			})
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s failed to update currencies. Err %s\n",
					b.Name,
					err)
			} else {
				log.Warnf(log.ExchangeSys,
					exchange.ResetConfigPairsWarningMessage, b.Name, asset.Spot, enabledPairs)
				forceUpdate = true

				err = b.UpdatePairs(enabledPairs, asset.Spot, true, true)
				if err != nil {
					log.Errorf(log.ExchangeSys,
						"%s failed to update currencies. Err: %s\n",
						b.Name,
						err)
				}
			}
		}
	}

	if !b.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err := b.UpdateTradablePairs(context.TODO(), forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			b.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitstamp) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	symbols, err := b.GetTradingPairs(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(symbols))
	for x := range symbols {
		if symbols[x].Trading != "Enabled" {
			continue
		}
		var pair currency.Pair
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
	return b.UpdatePairs(pairs, asset.Spot, false, forceUpdate)
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bitstamp) UpdateTickers(ctx context.Context, a asset.Item) error {
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
		LastUpdated:  time.Unix(tick.Timestamp, 0),
		ExchangeName: b.Name,
		AssetType:    a})
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(b.Name, fPair, a)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitstamp) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tick, err := ticker.GetTicker(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateTicker(ctx, fPair, assetType)
	}
	return tick, nil
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

// FetchOrderbook returns the orderbook for a currency pair
func (b *Bitstamp) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	ob, err := orderbook.Get(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateOrderbook(ctx, fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (b *Bitstamp) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := b.GetOrderbook(ctx, fPair.String())
	if err != nil {
		return book, err
	}

	book.Bids = make(orderbook.Items, len(orderbookNew.Bids))
	for x := range orderbookNew.Bids {
		book.Bids[x] = orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		}
	}

	filterOrderbookZeroBidPrice(book)

	book.Asks = make(orderbook.Items, len(orderbookNew.Asks))
	for x := range orderbookNew.Asks {
		book.Asks[x] = orderbook.Item{
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

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Bitstamp) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(b.Name, creds, assetType)
	if err != nil {
		return b.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitstamp) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitstamp) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
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
			Timestamp:    time.Unix(tradeData[i].Date, 0),
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
	if err := s.Validate(); err != nil {
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
		s.Side == order.Buy,
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
func (b *Bitstamp) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
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
func (b *Bitstamp) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
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
func (b *Bitstamp) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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

		var tm time.Time
		tm, err = parseTime(resp[i].DateTime)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetActiveOrders unable to parse time: %s\n", b.Name, err)
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
			Date:     tm,
			Pair:     p,
			Exchange: b.Name,
		}
	}
	return req.Filter(b.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (b *Bitstamp) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
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

		var tm time.Time
		tm, err = parseTime(resp[i].Date)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s GetOrderHistory unable to parse time: %s\n", b.Name, err)
		}

		orders = append(orders, order.Detail{
			OrderID:  strconv.FormatInt(resp[i].OrderID, 10),
			Date:     tm,
			Exchange: b.Name,
			Pair:     currPair,
		})
	}
	return req.Filter(b.Name, orders), nil
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Bitstamp) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitstamp) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := b.GetKlineRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	candles, err := b.OHLC(ctx,
		req.RequestFormatted.String(),
		req.Start,
		req.End,
		b.FormatExchangeKlineInterval(req.ExchangeInterval),
		strconv.FormatInt(int64(b.Features.Enabled.Kline.ResultLimit), 10))
	if err != nil {
		return nil, err
	}

	timeSeries := make([]kline.Candle, 0, len(candles.Data.OHLCV))
	for x := range candles.Data.OHLCV {
		timestamp := time.Unix(candles.Data.OHLCV[x].Timestamp, 0)
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
			strconv.FormatInt(int64(b.Features.Enabled.Kline.ResultLimit), 10),
		)
		if err != nil {
			return nil, err
		}

		for i := range candles.Data.OHLCV {
			timstamp := time.Unix(candles.Data.OHLCV[i].Timestamp, 0)
			if timstamp.Before(req.RangeHolder.Ranges[x].Start.Time) ||
				timstamp.After(req.RangeHolder.Ranges[x].End.Time) {
				continue
			}
			timeSeries = append(timeSeries, kline.Candle{
				Time:   time.Unix(candles.Data.OHLCV[i].Timestamp, 0),
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
