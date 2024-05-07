package bitmex

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
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
				FundingRateFetching: true,
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
				FundingRateFetching:    false, // supported but not implemented // TODO when multi-websocket support added
			},
			FuturesCapabilities: exchange.FuturesCapabilities{
				FundingRates: true,
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.PerpetualContract: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportedViaTicker: true,
					SupportsRestBatch:  true,
				},
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
	b.Websocket = stream.NewWebsocket()
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
		ExchangeConfig:        exch,
		DefaultURL:            bitmexWSURL,
		RunningURL:            wsEndpoint,
		Connector:             b.WsConnect,
		Subscriber:            b.Subscribe,
		Unsubscriber:          b.Unsubscribe,
		GenerateSubscriptions: b.GenerateDefaultSubscriptions,
		Features:              &b.Features.Supports.WebsocketCapabilities,
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
func (b *Bitmex) UpdateTradablePairs(ctx context.Context, _ bool) error {
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
	return b.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (b *Bitmex) UpdateTickers(ctx context.Context, a asset.Item) error {
	if !b.SupportsAsset(a) {
		return fmt.Errorf("%w for [%v]", asset.ErrNotSupported, a)
	}

	tick, err := b.GetActiveAndIndexInstruments(ctx)
	if err != nil {
		return err
	}

	var enabled bool
instruments:
	for j := range tick {
		var pair currency.Pair
		switch a {
		case asset.Futures:
			if tick[j].Typ != futuresID {
				continue instruments
			}
			pair, enabled, err = b.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		case asset.Index:
			switch tick[j].Typ {
			case bitMEXBasketIndexID,
				bitMEXPriceIndexID,
				bitMEXLendingPremiumIndexID,
				bitMEXVolatilityIndexID:
			default:
				continue instruments
			}
			// NOTE: Filtering is done below to remove the underscore in a
			// limited amount of index asset strings while the rest do not
			// contain an underscore. Calling DeriveFrom will then error and
			// the instruments will be missed.
			tick[j].Symbol = strings.Replace(tick[j].Symbol, currency.UnderscoreDelimiter, "", 1)
			pair, enabled, err = b.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		case asset.PerpetualContract:
			if tick[j].Typ != perpetualContractID {
				continue instruments
			}
			pair, enabled, err = b.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		case asset.Spot:
			if tick[j].Typ != spotID {
				continue instruments
			}
			tick[j].Symbol = strings.Replace(tick[j].Symbol, currency.UnderscoreDelimiter, "", 1)
			pair, enabled, err = b.MatchSymbolCheckEnabled(tick[j].Symbol, a, false)
		}

		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return err
		}
		if !enabled {
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
			OpenInterest: tick[j].OpenInterest,
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
	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        b.Name,
		Pair:            p,
		Asset:           assetType,
		VerifyOrderbook: b.CanVerifyOrderbook,
	}

	if assetType == asset.Index {
		return book, common.ErrFunctionNotSupported
	}

	fPair, err := b.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return book, err
	}

	orderbookNew, err := b.GetOrderbook(ctx,
		OrderBookGetL2Params{
			Symbol: fPair.String(),
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

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (b *Bitmex) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	history, err := b.GetWalletHistory(ctx, "all")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(history))
	for i := range history {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    b.Name,
			Status:          history[i].TransactStatus,
			Timestamp:       history[i].Timestamp,
			Currency:        history[i].Currency,
			Amount:          history[i].Amount,
			Fee:             history[i].Fee,
			TransferType:    history[i].TransactType,
			CryptoToAddress: history[i].Address,
			CryptoTxID:      history[i].TransactID,
			CryptoChain:     history[i].Network,
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (b *Bitmex) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	history, err := b.GetWalletHistory(ctx, c.String())
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, len(history))
	for i := range history {
		resp[i] = exchange.WithdrawalHistory{
			Status:          history[i].TransactStatus,
			Timestamp:       history[i].Timestamp,
			Currency:        history[i].Currency,
			Amount:          history[i].Amount,
			Fee:             history[i].Fee,
			TransferType:    history[i].TransactType,
			CryptoToAddress: history[i].Address,
			CryptoTxID:      history[i].TransactID,
			CryptoChain:     history[i].Network,
		}
	}
	return resp, nil
}

// GetServerTime returns the current exchange server time.
func (b *Bitmex) GetServerTime(_ context.Context, _ asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrFunctionNotSupported
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (b *Bitmex) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return b.GetHistoricTrades(ctx, p, assetType, time.Now().Add(-time.Minute*15), time.Now())
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (b *Bitmex) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if assetType == asset.Index {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
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

	if math.Trunc(s.Amount) != s.Amount {
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

	if math.Trunc(action.Amount) != action.Amount {
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
func (b *Bitmex) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	var orderIDs, clientIDs []string
	for i := range o {
		switch {
		case o[i].ClientOrderID != "":
			clientIDs = append(clientIDs, o[i].ClientID)
		case o[i].OrderID != "":
			orderIDs = append(orderIDs, o[i].OrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	joinedOrderIDs := strings.Join(orderIDs, ",")
	joinedClientIDs := strings.Join(clientIDs, ",")
	params := &OrderCancelParams{
		OrderID:       joinedOrderIDs,
		ClientOrderID: joinedClientIDs,
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	cancelResponse, err := b.CancelOrders(ctx, params)
	if err != nil {
		return nil, err
	}
	for i := range cancelResponse {
		resp.Status[cancelResponse[i].OrderID] = cancelResponse[i].OrdStatus
	}
	return resp, nil
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
func (b *Bitmex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if err := b.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, err
	}

	resp, err := b.GetOrders(ctx, &OrdersRequest{
		Filter: `{"orderID":"` + orderID + `"}`,
	})
	if err != nil {
		return nil, err
	}
	for i := range resp {
		if resp[i].OrderID != orderID {
			continue
		}
		var orderStatus order.Status
		orderStatus, err = order.StringToOrderStatus(resp[i].OrdStatus)
		if err != nil {
			return nil, err
		}
		var oType order.Type
		oType, err = b.getOrderType(resp[i].OrdType)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
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
			Pair:            pair,
			AssetType:       assetType,
		}, nil
	}
	return nil, fmt.Errorf("%w %v", order.ErrOrderNotFound, orderID)
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
func (b *Bitmex) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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
func (b *Bitmex) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
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

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (b *Bitmex) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
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

// GetFuturesContractDetails returns details about futures contracts
func (b *Bitmex) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !b.SupportsAsset(item) || item == asset.Index {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}

	marketInfo, err := b.GetInstruments(ctx, &GenericRequestParams{Reverse: true, Count: 500})
	if err != nil {
		return nil, err
	}

	resp := make([]futures.Contract, 0, len(marketInfo))
	switch item {
	case asset.PerpetualContract:
		for x := range marketInfo {
			if marketInfo[x].Typ != perpetualContractID {
				continue
			}
			var cp, underlying currency.Pair
			cp, err = currency.NewPairFromStrings(marketInfo[x].RootSymbol, marketInfo[x].QuoteCurrency)
			if err != nil {
				return nil, err
			}
			underlying, err = currency.NewPairFromStrings(marketInfo[x].RootSymbol, marketInfo[x].SettlCurrency)
			if err != nil {
				return nil, err
			}
			var s time.Time
			if marketInfo[x].Front != "" {
				s, err = time.Parse(time.RFC3339, marketInfo[x].Front)
				if err != nil {
					return nil, err
				}
			}
			var contractSettlementType futures.ContractSettlementType
			switch {
			case cp.Quote.Equal(currency.USDT):
				contractSettlementType = futures.Linear
			case cp.Quote.Equal(currency.USD):
				contractSettlementType = futures.Quanto
			default:
				contractSettlementType = futures.Inverse
			}
			resp = append(resp, futures.Contract{
				Exchange:             b.Name,
				Name:                 cp,
				Underlying:           underlying,
				Asset:                item,
				StartDate:            s,
				IsActive:             marketInfo[x].State == "Open",
				Status:               marketInfo[x].State,
				Type:                 futures.Perpetual,
				SettlementType:       contractSettlementType,
				SettlementCurrencies: currency.Currencies{currency.NewCode(marketInfo[x].SettlCurrency)},
				Multiplier:           marketInfo[x].Multiplier,
				LatestRate: fundingrate.Rate{
					Time: marketInfo[x].FundingTimestamp,
					Rate: decimal.NewFromFloat(marketInfo[x].FundingRate),
				},
			})
		}
	case asset.Futures:
		for x := range marketInfo {
			if marketInfo[x].Typ != futuresID {
				continue
			}
			var cp, underlying currency.Pair
			cp, err = currency.NewPairFromStrings(marketInfo[x].RootSymbol, marketInfo[x].Symbol[len(marketInfo[x].RootSymbol):])
			if err != nil {
				return nil, err
			}
			underlying, err = currency.NewPairFromStrings(marketInfo[x].RootSymbol, marketInfo[x].SettlCurrency)
			if err != nil {
				return nil, err
			}
			var s, e time.Time
			if marketInfo[x].Front != "" {
				s, err = time.Parse(time.RFC3339, marketInfo[x].Front)
				if err != nil {
					return nil, err
				}
			}
			if marketInfo[x].Expiry != "" {
				e, err = time.Parse(time.RFC3339, marketInfo[x].Expiry)
				if err != nil {
					return nil, err
				}
			}
			var ct futures.ContractType
			contractDuration := e.Sub(s)
			switch {
			case contractDuration <= kline.OneWeek.Duration()+kline.ThreeDay.Duration():
				ct = futures.Weekly
			case contractDuration <= kline.TwoWeek.Duration()+kline.ThreeDay.Duration():
				ct = futures.Fortnightly
			case contractDuration <= kline.OneMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.Monthly
			case contractDuration <= kline.ThreeMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.Quarterly
			case contractDuration <= kline.SixMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.HalfYearly
			case contractDuration <= kline.NineMonth.Duration()+kline.ThreeWeek.Duration():
				ct = futures.NineMonthly
			case contractDuration <= kline.OneYear.Duration()+kline.ThreeWeek.Duration():
				ct = futures.Yearly
			}
			contractSettlementType := futures.Inverse
			switch {
			case strings.Contains(cp.Quote.String(), "USDT"):
				contractSettlementType = futures.Linear
			case strings.Contains(cp.Quote.String(), "USD"):
				contractSettlementType = futures.Quanto
			}
			resp = append(resp, futures.Contract{
				Exchange:             b.Name,
				Name:                 cp,
				Underlying:           underlying,
				Asset:                item,
				StartDate:            s,
				EndDate:              e,
				IsActive:             marketInfo[x].State == "Open",
				Status:               marketInfo[x].State,
				Type:                 ct,
				SettlementCurrencies: currency.Currencies{currency.NewCode(marketInfo[x].SettlCurrency)},
				Multiplier:           marketInfo[x].Multiplier,
				SettlementType:       contractSettlementType,
			})
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (b *Bitmex) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}

	if r.IncludePredictedRate {
		return nil, fmt.Errorf("%w IncludePredictedRate", common.ErrFunctionNotSupported)
	}

	count := "1"
	if r.Pair.IsEmpty() {
		count = "500"
	} else {
		isPerp, err := b.IsPerpetualFutureCurrency(r.Asset, r.Pair)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			return nil, fmt.Errorf("%w %v %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
		}
	}

	format, err := b.GetPairFormat(r.Asset, true)
	if err != nil {
		return nil, err
	}
	fPair := format.Format(r.Pair)
	rates, err := b.GetFullFundingHistory(ctx, fPair, count, "", "", "", true, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}

	resp := make([]fundingrate.LatestRateResponse, 0, len(rates))
	// Bitmex returns historical rates from this endpoint, we only want the latest
	latestRateSymbol := make(map[string]bool)
	for i := range rates {
		if _, ok := latestRateSymbol[rates[i].Symbol]; ok {
			continue
		}
		latestRateSymbol[rates[i].Symbol] = true
		var nr time.Time
		nr, err = time.Parse(time.RFC3339, rates[i].FundingInterval)
		if err != nil {
			return nil, err
		}
		var cp currency.Pair
		var isEnabled bool
		cp, isEnabled, err = b.MatchSymbolCheckEnabled(rates[i].Symbol, r.Asset, false)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		if !isEnabled {
			continue
		}
		var isPerp bool
		isPerp, err = b.IsPerpetualFutureCurrency(r.Asset, cp)
		if err != nil {
			return nil, err
		}
		if !isPerp {
			continue
		}
		resp = append(resp, fundingrate.LatestRateResponse{
			Exchange: b.Name,
			Asset:    r.Asset,
			Pair:     cp,
			LatestRate: fundingrate.Rate{
				Time: rates[i].Timestamp,
				Rate: decimal.NewFromFloat(rates[i].FundingRate),
			},
			TimeOfNextRate: rates[i].Timestamp.Add(time.Duration(nr.Hour()) * time.Hour),
			TimeChecked:    time.Now(),
		})
	}
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (b *Bitmex) IsPerpetualFutureCurrency(a asset.Item, _ currency.Pair) (bool, error) {
	return a == asset.PerpetualContract, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (b *Bitmex) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (b *Bitmex) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset == asset.Spot || k[i].Asset == asset.Index {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	if len(k) != 1 {
		activeInstruments, err := b.GetActiveAndIndexInstruments(ctx)
		if err != nil {
			return nil, err
		}
		resp := make([]futures.OpenInterest, 0, len(activeInstruments))
		for i := range activeInstruments {
			for _, a := range b.CurrencyPairs.GetAssetTypes(true) {
				var symbol currency.Pair
				var enabled bool
				symbol, enabled, err = b.MatchSymbolCheckEnabled(activeInstruments[i].Symbol, a, false)
				if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
					return nil, err
				}
				if !enabled {
					continue
				}
				var appendData bool
				for j := range k {
					if k[j].Pair().Equal(symbol) && k[j].Asset == a {
						appendData = true
						break
					}
				}
				if len(k) > 0 && !appendData {
					continue
				}
				resp = append(resp, futures.OpenInterest{
					Key: key.ExchangePairAsset{
						Exchange: b.Name,
						Base:     symbol.Base.Item,
						Quote:    symbol.Quote.Item,
						Asset:    a,
					},
					OpenInterest: activeInstruments[i].OpenInterest,
				})
			}
		}
		return resp, nil
	}
	_, isEnabled, err := b.MatchSymbolCheckEnabled(k[0].Pair().String(), k[0].Asset, false)
	if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
		return nil, err
	}
	if !isEnabled {
		return nil, fmt.Errorf("%w %v %v", currency.ErrPairNotEnabled, k[0].Asset, k[0].Pair())
	}
	symbolStr, err := b.FormatSymbol(k[0].Pair(), k[0].Asset)
	if err != nil {
		return nil, err
	}
	instrument, err := b.GetInstrument(ctx, &GenericRequestParams{Symbol: symbolStr})
	if err != nil {
		return nil, err
	}
	if len(instrument) != 1 {
		return nil, fmt.Errorf("%w %v", currency.ErrPairNotFound, k[0].Pair())
	}
	resp := make([]futures.OpenInterest, 1)
	resp[0] = futures.OpenInterest{
		Key: key.ExchangePairAsset{
			Exchange: b.Name,
			Base:     k[0].Base,
			Quote:    k[0].Quote,
			Asset:    k[0].Asset,
		},
		OpenInterest: instrument[0].OpenInterest,
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (b *Bitmex) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := b.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	return tradeBaseURL + cp.Upper().String(), nil
}
