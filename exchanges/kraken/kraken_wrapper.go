package kraken

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
func (k *Kraken) GetDefaultConfig() (*config.ExchangeConfig, error) {
	k.SetDefaults()
	exchCfg := new(config.ExchangeConfig)
	exchCfg.Name = k.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = k.BaseCurrencies

	err := k.SetupDefaults(exchCfg)
	if err != nil {
		return nil, err
	}

	if k.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err = k.UpdateTradablePairs(true)
		if err != nil {
			return nil, err
		}
	}

	return exchCfg, nil
}

// SetDefaults sets current default settings
func (k *Kraken) SetDefaults() {
	k.Name = "Kraken"
	k.Enabled = true
	k.Verbose = true
	k.API.CredentialsValidator.RequiresKey = true
	k.API.CredentialsValidator.RequiresSecret = true
	k.API.CredentialsValidator.RequiresBase64DecodeSecret = true

	requestFmt := &currency.PairFormat{
		Uppercase: true,
		Separator: ",",
	}
	configFmt := &currency.PairFormat{
		Uppercase: true,
		Delimiter: currency.DashDelimiter,
		Separator: ",",
	}
	err := k.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	k.Features = exchange.Features{
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
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrder:         true,
				SubmitOrder:         true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				FiatDeposit:         true,
				FiatWithdraw:        true,
				TradeFee:            true,
				FiatDepositFee:      true,
				FiatWithdrawalFee:   true,
				CryptoDepositFee:    true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:     true,
				TradeFetching:      true,
				KlineFetching:      true,
				OrderbookFetching:  true,
				Subscribe:          true,
				Unsubscribe:        true,
				MessageCorrelation: true,
				SubmitOrder:        true,
				CancelOrder:        true,
				CancelOrders:       true,
				GetOrders:          true,
				GetOrder:           true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithSetup |
				exchange.WithdrawCryptoWith2FA |
				exchange.AutoWithdrawFiatWithSetup |
				exchange.WithdrawFiatWith2FA,
			Kline: kline.ExchangeCapabilitiesSupported{
				DateRanges: true,
				Intervals:  true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: map[string]bool{
					kline.OneMin.Word():     true,
					kline.ThreeMin.Word():   true,
					kline.FiveMin.Word():    true,
					kline.FifteenMin.Word(): true,
					kline.ThirtyMin.Word():  true,
					kline.OneHour.Word():    true,
					kline.FourHour.Word():   true,
					kline.OneDay.Word():     true,
					kline.FifteenDay.Word(): true,
					kline.OneWeek.Word():    true,
				},
			},
		},
	}

	k.Requester = request.New(k.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(request.NewBasicRateLimit(krakenRateInterval, krakenRequestRate)))

	k.API.Endpoints.URLDefault = krakenAPIURL
	k.API.Endpoints.URL = k.API.Endpoints.URLDefault
	k.Websocket = stream.New()
	k.API.Endpoints.WebsocketURL = krakenWSURL
	k.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	k.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	k.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets current exchange configuration
func (k *Kraken) Setup(exch *config.ExchangeConfig) error {
	if !exch.Enabled {
		k.SetEnabled(false)
		return nil
	}

	err := k.SetupDefaults(exch)
	if err != nil {
		return err
	}

	err = k.SeedAssets()
	if err != nil {
		return err
	}

	err = k.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:                          exch.Features.Enabled.Websocket,
		Verbose:                          exch.Verbose,
		AuthenticatedWebsocketAPISupport: exch.API.AuthenticatedWebsocketSupport,
		WebsocketTimeout:                 exch.WebsocketTrafficTimeout,
		DefaultURL:                       krakenWSURL,
		ExchangeName:                     exch.Name,
		RunningURL:                       exch.API.Endpoints.WebsocketURL,
		Connector:                        k.WsConnect,
		Subscriber:                       k.Subscribe,
		UnSubscriber:                     k.Unsubscribe,
		GenerateSubscriptions:            k.GenerateDefaultSubscriptions,
		Features:                         &k.Features.Supports.WebsocketCapabilities,
		OrderbookBufferLimit:             exch.WebsocketOrderbookBufferLimit,
		BufferEnabled:                    true,
		SortBuffer:                       true,
	})
	if err != nil {
		return err
	}

	err = k.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            krakenWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  krakenWSURL,
	})
	if err != nil {
		return err
	}

	return k.Websocket.SetupNewConnection(stream.ConnectionSetup{
		RateLimit:            krakenWsRateLimit,
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  krakenAuthWSURL,
		Authenticated:        true,
	})
}

// Start starts the Kraken go routine
func (k *Kraken) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		k.Run()
		wg.Done()
	}()
}

// Run implements the Kraken wrapper
func (k *Kraken) Run() {
	if k.Verbose {
		k.PrintEnabledPairs()
	}

	forceUpdate := false
	format, err := k.GetPairFormat(asset.Spot, false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			k.Name,
			err)
		return
	}
	enabled, err := k.GetEnabledPairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			k.Name,
			err)
		return
	}

	avail, err := k.GetAvailablePairs(asset.Spot)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			k.Name,
			err)
		return
	}

	if !common.StringDataContains(enabled.Strings(), format.Delimiter) ||
		!common.StringDataContains(avail.Strings(), format.Delimiter) ||
		common.StringDataContains(avail.Strings(), "ZUSD") {
		var p currency.Pairs
		p, err = currency.NewPairsFromStrings([]string{currency.XBT.String() +
			format.Delimiter +
			currency.USD.String()})
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%s failed to update currencies. Err: %s\n",
				k.Name,
				err)
		} else {
			log.Warn(log.ExchangeSys, "Available pairs for Kraken reset due to config upgrade, please enable the ones you would like again")
			forceUpdate = true

			err = k.UpdatePairs(p, asset.Spot, true, true)
			if err != nil {
				log.Errorf(log.ExchangeSys,
					"%s failed to update currencies. Err: %s\n",
					k.Name,
					err)
			}
		}
	}

	if !k.GetEnabledFeatures().AutoPairUpdates && !forceUpdate {
		return
	}

	err = k.UpdateTradablePairs(forceUpdate)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			k.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (k *Kraken) FetchTradablePairs(asset asset.Item) ([]string, error) {
	if !assetTranslator.Seeded() {
		if err := k.SeedAssets(); err != nil {
			return nil, err
		}
	}

	pairs, err := k.GetAssetPairs()
	if err != nil {
		return nil, err
	}

	format, err := k.GetPairFormat(asset, false)
	if err != nil {
		return nil, err
	}

	var products []string
	for i := range pairs {
		if strings.Contains(pairs[i].Altname, ".d") {
			continue
		}

		base := assetTranslator.LookupAltname(pairs[i].Base)
		if base == "" {
			log.Warnf(log.ExchangeSys,
				"%s unable to lookup altname for base currency %s",
				k.Name,
				pairs[i].Base)
			continue
		}

		quote := assetTranslator.LookupAltname(pairs[i].Quote)
		if quote == "" {
			log.Warnf(log.ExchangeSys,
				"%s unable to lookup altname for quote currency %s",
				k.Name,
				pairs[i].Quote)
			continue
		}
		products = append(products, base+format.Delimiter+quote)
	}
	return products, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (k *Kraken) UpdateTradablePairs(forceUpdate bool) error {
	pairs, err := k.FetchTradablePairs(asset.Spot)
	if err != nil {
		return err
	}

	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return k.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (k *Kraken) UpdateTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	pairs, err := k.GetEnabledPairs(assetType)
	if err != nil {
		return nil, err
	}
	pairsCollated, err := k.FormatExchangeCurrencies(pairs, assetType)
	if err != nil {
		return nil, err
	}
	tickers, err := k.GetTickers(pairsCollated)
	if err != nil {
		return nil, err
	}

	for i := range pairs {
		for c, t := range tickers {
			pairFmt, err := k.FormatExchangeCurrency(pairs[i], assetType)
			if err != nil {
				return nil, err
			}
			if !strings.EqualFold(pairFmt.String(), c) {
				altCurrency := assetTranslator.LookupAltname(c)
				if altCurrency == "" {
					continue
				}
				if !strings.EqualFold(pairFmt.String(), altCurrency) {
					continue // This looks dodge
				}
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         t.Last,
				High:         t.High,
				Low:          t.Low,
				Bid:          t.Bid,
				Ask:          t.Ask,
				Volume:       t.Volume,
				Open:         t.Open,
				Pair:         pairs[i],
				ExchangeName: k.Name,
				AssetType:    assetType})
			if err != nil {
				return nil, err
			}
		}
	}
	return ticker.GetTicker(k.Name, p, assetType)
}

// FetchTicker returns the ticker for a currency pair
func (k *Kraken) FetchTicker(p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(k.Name, p, assetType)
	if err != nil {
		return k.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (k *Kraken) FetchOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(k.Name, p, assetType)
	if err != nil {
		return k.UpdateOrderbook(p, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (k *Kraken) UpdateOrderbook(p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fpair, err := k.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	orderbookNew, err := k.GetDepth(fpair.String())
	if err != nil {
		return nil, err
	}

	var orderBook = new(orderbook.Base)
	for x := range orderbookNew.Bids {
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{
			Amount: orderbookNew.Bids[x].Amount,
			Price:  orderbookNew.Bids[x].Price,
		})
	}

	for x := range orderbookNew.Asks {
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{
			Amount: orderbookNew.Asks[x].Amount,
			Price:  orderbookNew.Asks[x].Price,
		})
	}

	orderBook.Pair = p
	orderBook.ExchangeName = k.Name
	orderBook.AssetType = assetType

	err = orderBook.Process()
	if err != nil {
		return orderBook, err
	}

	return orderbook.Get(k.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Kraken exchange - to-do
func (k *Kraken) UpdateAccountInfo() (account.Holdings, error) {
	var info account.Holdings
	info.Exchange = k.Name

	bal, err := k.GetBalance()
	if err != nil {
		return info, err
	}

	var balances []account.Balance
	for key := range bal {
		translatedCurrency := assetTranslator.LookupAltname(key)
		if translatedCurrency == "" {
			log.Warnf(log.ExchangeSys, "%s unable to translate currency: %s\n",
				k.Name,
				key)
			continue
		}
		balances = append(balances, account.Balance{
			CurrencyName: currency.NewCode(translatedCurrency),
			TotalValue:   bal[key],
		})
	}

	info.Accounts = append(info.Accounts, account.SubAccount{
		Currencies: balances,
	})

	err = account.Process(&info)
	if err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (k *Kraken) FetchAccountInfo() (account.Holdings, error) {
	acc, err := account.GetHoldings(k.Name)
	if err != nil {
		return k.UpdateAccountInfo()
	}

	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (k *Kraken) GetFundingHistory() ([]exchange.FundHistory, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (k *Kraken) GetRecentTrades(p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	p, err = k.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var tradeData []RecentTrades
	tradeData, err = k.GetTrades(assetTranslator.LookupCurrency(p.String()))
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	for i := range tradeData {
		side := order.Buy
		if tradeData[i].BuyOrSell == "s" {
			side = order.Sell
		}
		resp = append(resp, trade.Data{
			Exchange:     k.Name,
			CurrencyPair: p,
			AssetType:    assetType,
			Side:         side,
			Price:        tradeData[i].Price,
			Amount:       tradeData[i].Volume,
			Timestamp:    convert.TimeFromUnixTimestampDecimal(tradeData[i].Time),
		})
	}

	err = k.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (k *Kraken) GetHistoricTrades(_ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (k *Kraken) SubmitOrder(s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	err := s.Validate()
	if err != nil {
		return submitOrderResponse, err
	}

	if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		var resp string
		resp, err = k.wsAddOrder(&WsAddOrderRequest{
			OrderType: s.Type.String(),
			OrderSide: s.Side.String(),
			Pair:      s.Pair.String(),
			Price:     s.Price,
			Volume:    s.Amount,
		})
		if err != nil {
			return submitOrderResponse, err
		}
		submitOrderResponse.OrderID = resp
		submitOrderResponse.IsOrderPlaced = true
	} else {
		fPair, err := k.FormatExchangeCurrency(s.Pair, s.AssetType)
		if err != nil {
			return submitOrderResponse, err
		}
		var response AddOrderResponse
		response, err = k.AddOrder(fPair.String(),
			s.Side.String(),
			s.Type.String(),
			s.Amount,
			s.Price,
			0,
			0,
			&AddOrderOptions{})
		if err != nil {
			return submitOrderResponse, err
		}
		if len(response.TransactionIds) > 0 {
			submitOrderResponse.OrderID = strings.Join(response.TransactionIds, ", ")
		}
	}
	if s.Type == order.Market {
		submitOrderResponse.FullyMatched = true
	}
	submitOrderResponse.IsOrderPlaced = true
	return submitOrderResponse, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (k *Kraken) ModifyOrder(action *order.Modify) (string, error) {
	return "", common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (k *Kraken) CancelOrder(o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
		return k.wsCancelOrders([]string{o.ID})
	}
	_, err := k.CancelExistingOrder(o.ID)

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (k *Kraken) CancelAllOrders(_ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}

	var emptyOrderOptions OrderInfoOptions
	openOrders, err := k.GetOpenOrders(emptyOrderOptions)
	if err != nil {
		return cancelAllOrdersResponse, err
	}
	for orderID := range openOrders.Open {
		var err error
		if k.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			err = k.wsCancelOrders([]string{orderID})
		} else {
			_, err = k.CancelExistingOrder(orderID)
		}
		if err != nil {
			cancelAllOrdersResponse.Status[orderID] = err.Error()
		}
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (k *Kraken) GetOrderInfo(orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	resp, err := k.QueryOrdersInfo(OrderInfoOptions{
		Trades: true,
	}, orderID)
	if err != nil {
		return orderDetail, err
	}

	orderInfo, ok := resp[orderID]
	if !ok {
		return orderDetail, fmt.Errorf("order %s not found in response", orderID)
	}

	if assetType == "" {
		assetType = asset.Spot
	}

	avail, err := k.GetAvailablePairs(assetType)
	if err != nil {
		return orderDetail, err
	}

	format, err := k.GetPairFormat(assetType, true)
	if err != nil {
		return orderDetail, err
	}

	var trades []order.TradeHistory
	for i := range orderInfo.Trades {
		trades = append(trades, order.TradeHistory{
			TID: orderInfo.Trades[i],
		})
	}
	side, err := order.StringToOrderSide(orderInfo.Description.Type)
	if err != nil {
		return orderDetail, err
	}
	status, err := order.StringToOrderStatus(orderInfo.Status)
	if err != nil {
		return orderDetail, err
	}
	oType, err := order.StringToOrderType(orderInfo.Description.OrderType)
	if err != nil {
		return orderDetail, err
	}

	p, err := currency.NewPairFromFormattedPairs(orderInfo.Description.Pair,
		avail,
		format)
	if err != nil {
		return orderDetail, err
	}
	orderDetail = order.Detail{
		Exchange:        k.Name,
		ID:              orderID,
		Pair:            p,
		Side:            side,
		Type:            oType,
		Date:            convert.TimeFromUnixTimestampDecimal(orderInfo.OpenTime),
		CloseTime:       convert.TimeFromUnixTimestampDecimal(orderInfo.CloseTime),
		Status:          status,
		Price:           orderInfo.Price,
		Amount:          orderInfo.Volume,
		ExecutedAmount:  orderInfo.VolumeExecuted,
		RemainingAmount: orderInfo.Volume - orderInfo.VolumeExecuted,
		Fee:             orderInfo.Fee,
		Trades:          trades,
	}

	return orderDetail, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (k *Kraken) GetDepositAddress(cryptocurrency currency.Code, _ string) (string, error) {
	methods, err := k.GetDepositMethods(cryptocurrency.String())
	if err != nil {
		return "", err
	}

	var method string
	for _, m := range methods {
		method = m.Method
	}

	if method == "" {
		return "", errors.New("method not found")
	}

	return k.GetCryptoDepositAddress(method, cryptocurrency.String())
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal
// Populate exchange.WithdrawRequest.TradePassword with withdrawal key name, as set up on your account
func (k *Kraken) WithdrawCryptocurrencyFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := k.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.TradePassword, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: v,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFunds(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := k.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.TradePassword, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (k *Kraken) WithdrawFiatFundsToInternationalBank(withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := k.Withdraw(withdrawRequest.Currency.String(), withdrawRequest.TradePassword, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Status: v,
	}, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (k *Kraken) GetFeeByType(feeBuilder *exchange.FeeBuilder) (float64, error) {
	if !k.AllowAuthenticatedRequest() && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return k.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (k *Kraken) GetActiveOrders(req *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	resp, err := k.GetOpenOrders(OrderInfoOptions{})
	if err != nil {
		return nil, err
	}

	avail, err := k.GetAvailablePairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	fmt, err := k.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp.Open {
		p, err := currency.NewPairFromFormattedPairs(resp.Open[i].Description.Pair,
			avail,
			fmt)
		if err != nil {
			return nil, err
		}

		side := order.Side(strings.ToUpper(resp.Open[i].Description.Type))
		orderType := order.Type(strings.ToUpper(resp.Open[i].Description.OrderType))
		orders = append(orders, order.Detail{
			ID:              i,
			Amount:          resp.Open[i].Volume,
			RemainingAmount: (resp.Open[i].Volume - resp.Open[i].VolumeExecuted),
			ExecutedAmount:  resp.Open[i].VolumeExecuted,
			Exchange:        k.Name,
			Date:            convert.TimeFromUnixTimestampDecimal(resp.Open[i].OpenTime),
			Price:           resp.Open[i].Description.Price,
			Side:            side,
			Type:            orderType,
			Pair:            p,
		})
	}

	order.FilterOrdersByTickRange(&orders, req.StartTicks, req.EndTicks)
	order.FilterOrdersBySide(&orders, req.Side)
	order.FilterOrdersByCurrencies(&orders, req.Pairs)
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (k *Kraken) GetOrderHistory(getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	req := GetClosedOrdersOptions{}
	if getOrdersRequest.StartTicks.Unix() > 0 {
		req.Start = strconv.FormatInt(getOrdersRequest.StartTicks.Unix(), 10)
	}
	if getOrdersRequest.EndTicks.Unix() > 0 {
		req.End = strconv.FormatInt(getOrdersRequest.EndTicks.Unix(), 10)
	}

	avail, err := k.GetAvailablePairs(asset.Spot)
	if err != nil {
		return nil, err
	}

	fmt, err := k.GetPairFormat(asset.Spot, true)
	if err != nil {
		return nil, err
	}

	resp, err := k.GetClosedOrders(req)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	for i := range resp.Closed {
		p, err := currency.NewPairFromFormattedPairs(resp.Closed[i].Description.Pair,
			avail,
			fmt)
		if err != nil {
			return nil, err
		}

		side := order.Side(strings.ToUpper(resp.Closed[i].Description.Type))
		orderType := order.Type(strings.ToUpper(resp.Closed[i].Description.OrderType))
		orders = append(orders, order.Detail{
			ID:              i,
			Amount:          resp.Closed[i].Volume,
			RemainingAmount: (resp.Closed[i].Volume - resp.Closed[i].VolumeExecuted),
			ExecutedAmount:  resp.Closed[i].VolumeExecuted,
			Exchange:        k.Name,
			Date:            convert.TimeFromUnixTimestampDecimal(resp.Closed[i].OpenTime),
			CloseTime:       convert.TimeFromUnixTimestampDecimal(resp.Closed[i].CloseTime),
			Price:           resp.Closed[i].Description.Price,
			Side:            side,
			Type:            orderType,
			Pair:            p,
		})
	}

	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)
	order.FilterOrdersByCurrencies(&orders, getOrdersRequest.Pairs)
	return orders, nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (k *Kraken) AuthenticateWebsocket() error {
	resp, err := k.GetWebsocketToken()
	if resp != "" {
		authToken = resp
	}
	return err
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (k *Kraken) ValidateCredentials() error {
	_, err := k.UpdateAccountInfo()
	return k.CheckTransientError(err)
}

// FormatExchangeKlineInterval returns Interval to exchange formatted string
func (k *Kraken) FormatExchangeKlineInterval(in kline.Interval) string {
	return strconv.FormatFloat(in.Duration().Minutes(), 'f', -1, 64)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (k *Kraken) GetHistoricCandles(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := k.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}

	ret := kline.Item{
		Exchange: k.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	formattedPair, err := k.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	candles, err := k.GetOHLC(assetTranslator.LookupCurrency(formattedPair.Upper().String()), k.FormatExchangeKlineInterval(interval))
	if err != nil {
		return kline.Item{}, err
	}
	for x := range candles {
		timeValue, err := convert.TimeFromUnixTimestampFloat(candles[x].Time * 1000)
		if err != nil {
			return kline.Item{}, err
		}
		if timeValue.Before(start) || timeValue.After(end) {
			continue
		}
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   timeValue,
			Open:   candles[x].Open,
			High:   candles[x].High,
			Low:    candles[x].Low,
			Close:  candles[x].Close,
			Volume: candles[x].Volume,
		})
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (k *Kraken) GetHistoricCandlesExtended(pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := k.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: k.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}

	formattedPair, err := k.FormatExchangeCurrency(pair, a)
	if err != nil {
		return kline.Item{}, err
	}

	candles, err := k.GetOHLC(assetTranslator.LookupCurrency(formattedPair.Upper().String()), k.FormatExchangeKlineInterval(interval))
	if err != nil {
		return kline.Item{}, err
	}
	for i := range candles {
		timeValue, err := convert.TimeFromUnixTimestampFloat(candles[i].Time * 1000)
		if err != nil {
			return kline.Item{}, err
		}
		if timeValue.Before(start) || timeValue.After(end) {
			continue
		}
		ret.Candles = append(ret.Candles, kline.Candle{
			Time:   timeValue,
			Open:   candles[i].Open,
			High:   candles[i].High,
			Low:    candles[i].Low,
			Close:  candles[i].Close,
			Volume: candles[i].Volume,
		})
	}

	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
