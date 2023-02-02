package bitmex

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
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
func (b *Bitmex) GetDefaultConfig() (*config.Exchange, error) {
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

// SetDefaults sets the basic defaults for Bitmex
func (b *Bitmex) SetDefaults() {
	b.Name = "Bitmex"
	b.Enabled = true
	b.Verbose = true
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true

	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	standardRequestFmt := &currency.PairFormat{Uppercase: true}
	spotRequestFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter}

	spot := currency.PairStore{RequestFormat: spotRequestFormat, ConfigFormat: configFmt}
	err := b.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	perp := currency.PairStore{RequestFormat: standardRequestFmt, ConfigFormat: configFmt}
	err = b.StoreAssetPairFormat(asset.PerpetualContract, perp)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	futures := currency.PairStore{RequestFormat: standardRequestFmt, ConfigFormat: configFmt}
	err = b.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	index := currency.PairStore{RequestFormat: standardRequestFmt, ConfigFormat: configFmt}
	err = b.StoreAssetPairFormat(asset.Index, index)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	err = b.DisableAssetWebsocketSupport(asset.Index)
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
				TradeFetching:       true,
				OrderbookFetching:   true,
				AutoPairUpdates:     true,
				AccountInfo:         true,
				GetOrder:            true,
				GetOrders:           true,
				CancelOrders:        true,
				CancelOrder:         true,
				SubmitOrder:         true,
				SubmitOrders:        true,
				ModifyOrder:         true,
				DepositHistory:      true,
				WithdrawalHistory:   true,
				UserTradeHistory:    true,
				CryptoDeposit:       true,
				CryptoWithdrawal:    true,
				TradeFee:            true,
				CryptoWithdrawalFee: true,
			},
			WebsocketCapabilities: protocol.Features{
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				DeadMansSwitch:         true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.WithdrawCryptoWithEmail |
				exchange.WithdrawCryptoWith2FA |
				exchange.NoFiatWithdrawals,
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
		},
	}

	b.Requester, err = request.New(b.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      bitmexAPIURL,
		exchange.WebsocketSpot: bitmexWSURL,
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
func (b *Bitmex) Setup(exch *config.Exchange) error {
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

	wsEndpoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:         exch,
		DefaultURL:             bitmexWSURL,
		RunningURL:             wsEndpoint,
		Connector:              b.WsConnect,
		Subscriber:             b.Subscribe,
		Unsubscriber:           b.Unsubscribe,
		GenerateSubscriptions:  b.GenerateDefaultSubscriptions,
		ConnectionMonitorDelay: exch.ConnectionMonitorDelay,
		Features:               &b.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			UpdateEntriesByID: true,
		},
	})
	if err != nil {
		return err
	}
	return b.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  bitmexWSURL,
	})
}

// Start starts the Bitmex go routine
func (b *Bitmex) Start(wg *sync.WaitGroup) error {
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

// Run implements the Bitmex wrapper
func (b *Bitmex) Run() {
	if b.Verbose {
		wsEndpoint, err := b.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			log.Error(log.ExchangeSys, err)
		}
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s. (url: %s).\n",
			b.Name,
			common.IsEnabled(b.Websocket.IsEnabled()),
			wsEndpoint)
		b.PrintEnabledPairs()
	}

	if !b.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := b.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s failed to update tradable pairs. Err: %s", b.Name, err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (b *Bitmex) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	marketInfo, err := b.GetActiveAndIndexInstruments(ctx)
	if err != nil {
		return nil, err
	}

	pairs := make([]currency.Pair, 0, len(marketInfo))
	for x := range marketInfo {
		if marketInfo[x].State != "Open" && a != asset.Index {
			continue
		}

		var pair currency.Pair
		switch a {
		case asset.Spot:
			if marketInfo[x].Typ == spotID {
				pair, err = currency.NewPairFromString(marketInfo[x].Symbol)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		case asset.PerpetualContract:
			if marketInfo[x].Typ == perpetualContractID {
				var settleTrail string
				if strings.Contains(marketInfo[x].Symbol, currency.UnderscoreDelimiter) {
					// Example: ETHUSD_ETH quoted in USD, paid out in ETH.
					settlement := strings.Split(marketInfo[x].Symbol, currency.UnderscoreDelimiter)
					if len(settlement) != 2 {
						log.Warnf(log.ExchangeSys, "%s currency %s %s cannot be added to tradable pairs",
							b.Name,
							marketInfo[x].Symbol,
							a)
						break
					}
					settleTrail = currency.UnderscoreDelimiter + settlement[1]
				}
				pair, err = currency.NewPairFromStrings(marketInfo[x].Underlying,
					marketInfo[x].QuoteCurrency+settleTrail)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		case asset.Futures:
			if marketInfo[x].Typ == futuresID {
				isolate := strings.Split(marketInfo[x].Symbol, currency.UnderscoreDelimiter)
				if len(isolate[0]) < 3 {
					log.Warnf(log.ExchangeSys, "%s currency %s %s be cannot added to tradable pairs",
						b.Name,
						marketInfo[x].Symbol,
						a)
					break
				}
				var settleTrail string
				if len(isolate) == 2 {
					// Example: ETHUSDU22_ETH quoted in USD, paid out in ETH.
					settleTrail = currency.UnderscoreDelimiter + isolate[1]
				}

				root := isolate[0][:len(isolate[0])-3]
				contract := isolate[0][len(isolate[0])-3:]

				pair, err = currency.NewPairFromStrings(root, contract+settleTrail)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		case asset.Index:
			// TODO: This can be expanded into individual assets later.
			if marketInfo[x].Typ == bitMEXBasketIndexID ||
				marketInfo[x].Typ == bitMEXPriceIndexID ||
				marketInfo[x].Typ == bitMEXLendingPremiumIndexID ||
				marketInfo[x].Typ == bitMEXVolatilityIndexID {
				pair, err = currency.NewPairFromString(marketInfo[x].Symbol)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, pair)
			}
		default:
			return nil, errors.New("unhandled asset type")
		}
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (b *Bitmex) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := b.GetAssetTypes(false)

	for x := range assets {
		pairs, err := b.FetchTradablePairs(ctx, assets[x])
		if err != nil {
			return err
		}

		err = b.UpdatePairs(pairs, assets[x], false, false)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bitmex) UpdateTickers(ctx context.Context, a asset.Item) error {
	tick, err := b.GetActiveAndIndexInstruments(ctx)
	if err != nil {
		return err
	}

	pairs, err := b.GetEnabledPairs(a)
	if err != nil {
		return err
	}

	for j := range tick {
		var pair currency.Pair
		pair, err = currency.NewPairFromString(tick[j].Symbol)
		if err != nil {
			return err
		}

		if !pairs.Contains(pair, true) {
			continue
		}

		err = ticker.ProcessTicker(&ticker.Price{
			Last:         tick[j].LastPrice,
			High:         tick[j].HighPrice,
			Low:          tick[j].LowPrice,
			Bid:          tick[j].BidPrice,
			Ask:          tick[j].AskPrice,
			Volume:       tick[j].Volume24h,
			Close:        tick[j].PrevClosePrice,
			Pair:         pair,
			LastUpdated:  tick[j].Timestamp,
			ExchangeName: b.Name,
			AssetType:    a})
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (b *Bitmex) UpdateTicker(ctx context.Context, p currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := b.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}

	fPair, err := b.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}

	return ticker.GetTicker(b.Name, fPair, a)
}

// FetchTicker returns the ticker for a currency pair
func (b *Bitmex) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}

	tickerNew, err := ticker.GetTicker(b.Name, fPair, assetType)
	if err != nil {
		return b.UpdateTicker(ctx, fPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (b *Bitmex) FetchOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
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
func (b *Bitmex) UpdateOrderbook(ctx context.Context, p currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}

	if assetType == asset.Index {
		return book, common.ErrFunctionNotSupported
	}

	fpair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := b.GetOrderbook(ctx,
		OrderBookGetL2Params{
			Symbol: fpair.String(),
			Depth:  500})
	if err != nil {
		return book, err
	}

	book.Asks = make(orderbook.Items, 0, len(orderbookNew))
	book.Bids = make(orderbook.Items, 0, len(orderbookNew))
	for i := range orderbookNew {
		switch {
		case strings.EqualFold(orderbookNew[i].Side, order.Sell.String()):
			book.Asks = append(book.Asks, orderbook.Item{
				Amount: float64(orderbookNew[i].Size),
				Price:  orderbookNew[i].Price,
			})
		case strings.EqualFold(orderbookNew[i].Side, order.Buy.String()):
			book.Bids = append(book.Bids, orderbook.Item{
				Amount: float64(orderbookNew[i].Size),
				Price:  orderbookNew[i].Price,
			})
		default:
			return book,
				fmt.Errorf("could not process orderbook, order side [%s] could not be matched",
					orderbookNew[i].Side)
		}
	}
	book.Asks.Reverse() // Reverse order of asks to ascending

	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(b.Name, p, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Bitmex exchange
func (b *Bitmex) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	var info account.Holdings

	userMargins, err := b.GetAllUserMargin(ctx)
	if err != nil {
		return info, err
	}

	accountBalances := make(map[string][]account.Balance)
	// Need to update to add Margin/Liquidity availability
	for i := range userMargins {
		accountID := strconv.FormatInt(userMargins[i].Account, 10)

		var wallet WalletInfo
		wallet, err = b.GetWalletInfo(ctx, userMargins[i].Currency)
		if err != nil {
			continue
		}

		accountBalances[accountID] = append(
			accountBalances[accountID], account.Balance{
				Currency: currency.NewCode(wallet.Currency),
				Total:    wallet.Amount,
			},
		)
	}

	if info.Accounts, err = account.CollectBalances(accountBalances, assetType); err != nil {
		return account.Holdings{}, err
	}
	info.Exchange = b.Name

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	if err := account.Process(&info, creds); err != nil {
		return account.Holdings{}, err
	}

	return info, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (b *Bitmex) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
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
func (b *Bitmex) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitmex) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) (resp []exchange.WithdrawalHistory, err error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Bitmex) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return b.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bitmex) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if assetType == asset.Index {
		return nil, fmt.Errorf("asset type '%v' not supported", assetType)
	}
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	p, err = b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	limit := 1000
	req := &GenericRequestParams{
		Symbol:  p.String(),
		Count:   int32(limit),
		EndTime: timestampEnd.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	ts := timestampStart
	var resp []trade.Data
allTrades:
	for {
		req.StartTime = ts.UTC().Format("2006-01-02T15:04:05.000Z")
		var tradeData []Trade
		tradeData, err = b.GetTrade(ctx, req)
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			if tradeData[i].Timestamp.Before(timestampStart) || tradeData[i].Timestamp.After(timestampEnd) {
				break allTrades
			}
			var side order.Side
			side, err = order.StringToOrderSide(tradeData[i].Side)
			if err != nil {
				return nil, err
			}
			if tradeData[i].Price == 0 {
				// Please note that indices (symbols starting with .) post trades at intervals to the trade feed.
				// These have a size of 0 and are used only to indicate a changing price.
				continue
			}
			resp = append(resp, trade.Data{
				Exchange:     b.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price,
				Amount:       float64(tradeData[i].Size),
				Timestamp:    tradeData[i].Timestamp,
				TID:          tradeData[i].TrdMatchID,
			})
			if i == len(tradeData)-1 {
				if ts.Equal(tradeData[i].Timestamp) {
					// reached end of trades to crawl
					break allTrades
				}
				ts = tradeData[i].Timestamp
			}
		}
		if len(tradeData) != limit {
			break allTrades
		}
	}
	err = b.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}

	sort.Sort(trade.ByDate(resp))
	return trade.FilterTradesByTime(resp, timestampStart, timestampEnd), nil
}

// SubmitOrder submits a new order
func (b *Bitmex) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	if math.Mod(s.Amount, 1) != 0 {
		return nil,
			errors.New("order contract amount can not have decimals")
	}

	fPair, err := b.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}

	var orderNewParams = OrderNewParams{
		OrderType:     s.Type.Title(),
		Symbol:        fPair.String(),
		OrderQuantity: s.Amount,
		Side:          s.Side.Title(),
	}

	if s.Type == order.Limit {
		orderNewParams.Price = s.Price
	}

	response, err := b.CreateOrder(ctx, &orderNewParams)
	if err != nil {
		return nil, err
	}
	return s.DeriveSubmitResponse(response.OrderID)
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (b *Bitmex) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}

	if math.Mod(action.Amount, 1) != 0 {
		return nil, errors.New("contract amount can not have decimals")
	}

	o, err := b.AmendOrder(ctx, &OrderAmendParams{
		OrderID:  action.OrderID,
		OrderQty: int32(action.Amount),
		Price:    action.Price})
	if err != nil {
		return nil, err
	}

	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}

	resp.OrderID = o.OrderID
	resp.RemainingAmount = o.OrderQty
	resp.LastUpdated = o.TransactTime
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (b *Bitmex) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	_, err := b.CancelOrders(ctx, &OrderCancelParams{
		OrderID: o.OrderID,
	})
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (b *Bitmex) CancelBatchOrders(ctx context.Context, o []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (b *Bitmex) CancelAllOrders(ctx context.Context, _ *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	var emptyParams OrderCancelAllParams
	orders, err := b.CancelAllExistingOrders(ctx, emptyParams)
	if err != nil {
		return cancelAllOrdersResponse, err
	}

	for i := range orders {
		if orders[i].OrdRejReason != "" {
			cancelAllOrdersResponse.Status[orders[i].OrderID] = orders[i].OrdRejReason
		}
	}

	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (b *Bitmex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	var orderDetail order.Detail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (b *Bitmex) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, _ string) (*deposit.Address, error) {
	resp, err := b.GetCryptoDepositAddress(ctx, cryptocurrency.String())
	if err != nil {
		return nil, err
	}
	return &deposit.Address{
		Address: resp,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	var r = UserRequestWithdrawalParams{
		Address:  withdrawRequest.Crypto.Address,
		Amount:   withdrawRequest.Amount,
		Currency: withdrawRequest.Currency.String(),
		OtpToken: withdrawRequest.OneTimePassword,
	}
	if withdrawRequest.Crypto.FeeAmount > 0 {
		r.Fee = withdrawRequest.Crypto.FeeAmount
	}

	resp, err := b.UserRequestWithdrawal(ctx, r)
	if err != nil {
		return nil, err
	}

	return &withdraw.ExchangeResponse{
		Status: resp.Text,
		ID:     resp.Tx,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (b *Bitmex) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (b *Bitmex) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !b.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return b.GetFee(feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
// This function is not concurrency safe due to orderSide/orderType maps
func (b *Bitmex) GetActiveOrders(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	params := OrdersRequest{
		Filter: "{\"open\":true}",
	}
	resp, err := b.GetOrders(ctx, &params)
	if err != nil {
		return nil, err
	}

	format, err := b.GetPairFormat(asset.PerpetualContract, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(resp))
	for i := range resp {
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(resp[i].OrdStatus)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}
		var oType order.Type
		oType, err = b.getOrderType(resp[i].OrdType)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}
		orderDetail := order.Detail{
			Date:            resp[i].Timestamp,
			Price:           resp[i].Price,
			Amount:          resp[i].OrderQty,
			ExecutedAmount:  resp[i].CumQty,
			RemainingAmount: resp[i].LeavesQty,
			Exchange:        b.Name,
			OrderID:         resp[i].OrderID,
			Side:            orderSideMap[resp[i].Side],
			Status:          orderStatus,
			Type:            oType,
			Pair: currency.NewPairWithDelimiter(resp[i].Symbol,
				resp[i].SettlCurrency,
				format.Delimiter),
		}

		orders[i] = orderDetail
	}
	return req.Filter(b.Name, orders), nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
// This function is not concurrency safe due to orderSide/orderType maps
func (b *Bitmex) GetOrderHistory(ctx context.Context, req *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := req.Validate()
	if err != nil {
		return nil, err
	}

	params := OrdersRequest{}
	resp, err := b.GetOrders(ctx, &params)
	if err != nil {
		return nil, err
	}

	format, err := b.GetPairFormat(asset.PerpetualContract, false)
	if err != nil {
		return nil, err
	}

	orders := make([]order.Detail, len(resp))
	for i := range resp {
		orderSide := orderSideMap[resp[i].Side]
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(resp[i].OrdStatus)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}

		pair := currency.NewPairWithDelimiter(resp[i].Symbol, resp[i].SettlCurrency, format.Delimiter)

		var oType order.Type
		oType, err = b.getOrderType(resp[i].OrdType)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", b.Name, err)
		}

		orderDetail := order.Detail{
			Price:                resp[i].Price,
			AverageExecutedPrice: resp[i].AvgPx,
			Amount:               resp[i].OrderQty,
			ExecutedAmount:       resp[i].CumQty,
			RemainingAmount:      resp[i].LeavesQty,
			Date:                 resp[i].TransactTime,
			CloseTime:            resp[i].Timestamp,
			Exchange:             b.Name,
			OrderID:              resp[i].OrderID,
			Side:                 orderSide,
			Status:               orderStatus,
			Type:                 oType,
			Pair:                 pair,
		}
		orderDetail.InferCostsAndTimes()

		orders[i] = orderDetail
	}
	return req.Filter(b.Name, orders), nil
}

// AuthenticateWebsocket sends an authentication message to the websocket
func (b *Bitmex) AuthenticateWebsocket(ctx context.Context) error {
	return b.websocketSendAuth(ctx)
}

// ValidateCredentials validates current credentials used for wrapper
// functionality
func (b *Bitmex) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	_, err := b.UpdateAccountInfo(ctx, assetType)
	return b.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (b *Bitmex) GetHistoricCandles(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (b *Bitmex) GetHistoricCandlesExtended(_ context.Context, _ currency.Pair, _ asset.Item, _ kline.Interval, _, _ time.Time) (*kline.Item, error) {
	return nil, common.ErrFunctionNotSupported
}

// getOrderType derives an order type from bitmex int representation
func (b *Bitmex) getOrderType(id int64) (order.Type, error) {
	o, ok := orderTypeMap[id]
	if !ok {
		return order.UnknownType, fmt.Errorf("unhandled order type for '%d': %w", id, order.ErrTypeIsInvalid)
	}
	return o, nil
}
