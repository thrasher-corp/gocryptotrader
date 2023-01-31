package kucoin

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
func (ku *Kucoin) GetDefaultConfig() (*config.Exchange, error) {
	ku.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = ku.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = ku.BaseCurrencies

	if ku.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := ku.UpdateTradablePairs(context.TODO(), true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Kucoin
func (ku *Kucoin) SetDefaults() {
	ku.Name = "Kucoin"
	ku.Enabled = true
	ku.Verbose = false

	ku.API.CredentialsValidator.RequiresKey = true
	ku.API.CredentialsValidator.RequiresSecret = true
	ku.API.CredentialsValidator.RequiresClientID = true

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}
	futures := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	}
	err := ku.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = ku.StoreAssetPairFormat(asset.Margin, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = ku.StoreAssetPairFormat(asset.Futures, futures)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	ku.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
				AutoPairUpdates:   true,
				AccountInfo:       true,
				CryptoWithdrawal:  true,
				SubmitOrder:       true,
				GetOrder:          true,
				GetOrders:         true,
				CancelOrder:       true,
				CancelOrders:      true,
				TradeFetching:     true,
				UserTradeHistory:  true,
				KlineFetching:     true,
				DepositHistory:    true,
				WithdrawalHistory: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				AccountInfo:            true,
				GetOrders:              true,
				TradeFetching:          true,
				KlineFetching:          true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
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
					kline.FourHour,
					kline.SixHour,
					kline.EightHour,
					kline.TwelveHour,
					kline.OneDay,
					kline.OneWeek,
				),
				ResultLimit: 1000,
			},
		},
	}
	ku.Requester, err = request.New(ku.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(SetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	ku.API.Endpoints = ku.NewEndpoints()
	err = ku.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      kucoinAPIURL,
		exchange.RestFutures:   kucoinFuturesAPIURL,
		exchange.WebsocketSpot: kucoinWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	ku.Websocket = stream.New()
	ku.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	ku.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	ku.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (ku *Kucoin) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		ku.SetEnabled(false)
		return nil
	}
	err = ku.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningEndpoint, err := ku.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = ku.Websocket.Setup(
		&stream.WebsocketSetup{
			ExchangeConfig:        exch,
			DefaultURL:            kucoinWebsocketURL,
			RunningURL:            wsRunningEndpoint,
			Connector:             ku.WsConnect,
			Subscriber:            ku.Subscribe,
			Unsubscriber:          ku.Unsubscribe,
			GenerateSubscriptions: ku.GenerateDefaultSubscriptions,
			Features:              &ku.Features.Supports.WebsocketCapabilities,
			OrderbookBufferConfig: buffer.Config{
				SortBuffer:            true,
				SortBufferByUpdateIDs: true,
				UpdateIDProgression:   true,
			},
			TradeFeed: ku.Features.Enabled.TradeFeed,
		})
	if err != nil {
		return err
	}
	return ku.Websocket.SetupNewConnection(stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            500,
	})
}

// Start starts the Kucoin go routine
func (ku *Kucoin) Start(wg *sync.WaitGroup) error {
	if wg == nil {
		return fmt.Errorf("%T %w", wg, common.ErrNilPointer)
	}
	wg.Add(1)
	go func() {
		ku.Run()
		wg.Done()
	}()
	return nil
}

// Run implements the Kucoin wrapper
func (ku *Kucoin) Run() {
	if ku.Verbose {
		log.Debugf(log.ExchangeSys,
			"%s Websocket: %s.",
			ku.Name,
			common.IsEnabled(ku.Websocket.IsEnabled()))
		ku.PrintEnabledPairs()
	}

	if !ku.GetEnabledFeatures().AutoPairUpdates {
		return
	}

	err := ku.UpdateTradablePairs(context.TODO(), false)
	if err != nil {
		log.Errorf(log.ExchangeSys,
			"%s failed to update tradable pairs. Err: %s",
			ku.Name,
			err)
	}
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (ku *Kucoin) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	switch assetType {
	case asset.Futures:
		myPairs, err := ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(myPairs))
		for x := range myPairs {
			pairs[x], err = currency.NewPairFromString(strings.ToUpper(myPairs[x].Symbol))
			if err != nil {
				return nil, err
			}
		}
		return pairs, nil
	case asset.Spot, asset.Margin:
		myPairs, err := ku.GetSymbols(ctx, "")
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(myPairs))
		for x := range myPairs {
			pairs[x], err = currency.NewPairFromString(strings.ToUpper(myPairs[x].Name))
			if err != nil {
				return nil, err
			}
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (ku *Kucoin) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assets := ku.GetAssetTypes(false)
	for a := range assets {
		// Since the tradable pairs of Margin is same as that of Spot, calling the tradable pairs for Margin is not necessary.
		// Instead, calling the UpdatePairs() method with Spot and Margin for Spot instruments saves call to FetchTradablePair() method with Margin.
		if assets[a] == asset.Margin {
			continue
		}
		pairs, err := ku.FetchTradablePairs(ctx, assets[a])
		if err != nil {
			return err
		}
		err = ku.UpdatePairs(pairs, assets[a], false, forceUpdate)
		if err != nil {
			return err
		}
		if assets[a] == asset.Spot {
			err = ku.UpdatePairs(pairs, asset.Margin, false, forceUpdate)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (ku *Kucoin) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	p, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if err := ku.UpdateTickers(ctx, assetType); err != nil {
		return nil, err
	}
	return ticker.GetTicker(ku.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (ku *Kucoin) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Futures:
		ticks, err := ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return err
		}
		for x := range ticks {
			pair, err := currency.NewPairFromString(ticks[x].Symbol)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks[x].LastTradePrice,
				High:         ticks[x].HighPrice,
				Low:          ticks[x].LowPrice,
				Volume:       ticks[x].VolumeOf24h,
				Ask:          ticks[x].IndexPrice,
				Bid:          ticks[x].MarkPrice,
				Pair:         pair,
				ExchangeName: ku.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
		return nil
	case asset.Spot:
		ticks, err := ku.GetAllTickers(ctx)
		if err != nil {
			return err
		}
		for t := range ticks {
			pair, err := currency.NewPairFromString(ticks[t].Symbol)
			if err != nil {
				return err
			}
			tick := &ticker.Price{
				Last:         ticks[t].Last,
				High:         ticks[t].High,
				Low:          ticks[t].Low,
				Volume:       ticks[t].Volume,
				Ask:          ticks[t].Buy,
				Bid:          ticks[t].Sell,
				Pair:         pair,
				ExchangeName: ku.Name,
				AssetType:    assetType,
			}
			err = ticker.ProcessTicker(tick)
			if err != nil {
				return err
			}
			tick.AssetType = asset.Margin
			err = ticker.ProcessTicker(tick)
			if err != nil {
				return err
			}
		}
	case asset.Margin:
		// Since margin and spot use the same currencies, update call to Spot tradable pairs ticker information updates
		// the ticker information of Margin currencies too.
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (ku *Kucoin) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	p, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tickerNew, err := ticker.GetTicker(ku.Name, p, assetType)
	if err != nil {
		return ku.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (ku *Kucoin) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	pair, err := ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	ob, err := orderbook.Get(ku.Name, pair, assetType)
	if err != nil {
		return ku.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (ku *Kucoin) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	err := ku.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	pair, err = ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	var ordBook *Orderbook
	switch assetType {
	case asset.Futures:
		ordBook, err = ku.GetFuturesOrderbook(ctx, pair.String())
	case asset.Spot, asset.Margin:
		_, err = ku.GetCredentials(ctx)
		if err != nil {
			ordBook, err = ku.GetPartOrderbook100(ctx, pair.String())
		} else {
			ordBook, err = ku.GetOrderbook(ctx, pair.String())
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        ku.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: ku.CanVerifyOrderbook,
		Asks:            ordBook.Asks,
		Bids:            ordBook.Bids,
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(ku.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (ku *Kucoin) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	holding := account.Holdings{
		Exchange: ku.Name,
	}
	err := ku.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return holding, asset.ErrNotSupported
	}
	switch assetType {
	case asset.Futures:
		accoutH, err := ku.GetFuturesAccountOverview(ctx, "")
		if err != nil {
			return account.Holdings{}, err
		}
		holding.Accounts = append(holding.Accounts, account.SubAccount{
			AssetType: assetType,
			Currencies: []account.Balance{{
				Currency: currency.NewCode(accoutH.Currency),
				Total:    accoutH.AvailableBalance + accoutH.MarginBalance,
				Hold:     accoutH.FrozenFunds,
				Free:     accoutH.AvailableBalance,
			}},
		})
	case asset.Spot, asset.Margin:
		acc := ku.accountTypeToString(assetType)
		accountH, err := ku.GetAllAccounts(ctx, "", acc)
		if err != nil {
			return account.Holdings{}, err
		}
		for x := range accountH {
			holding.Accounts = append(holding.Accounts, account.SubAccount{
				AssetType: assetType,
				Currencies: []account.Balance{
					{
						Currency: currency.NewCode(accountH[x].Currency),
						Total:    accountH[x].Balance,
						Hold:     accountH[x].Holds,
						Free:     accountH[x].Available,
					}},
			})
		}
	default:
		return holding, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return holding, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (ku *Kucoin) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := ku.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(ku.Name, creds, assetType)
	if err != nil {
		return ku.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (ku *Kucoin) GetFundingHistory(ctx context.Context) ([]exchange.FundHistory, error) {
	withdrawalsData, err := ku.GetWithdrawalList(ctx, "", "", time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	depositsData, err := ku.GetHistoricalDepositList(ctx, "", "", time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	fundingData := make([]exchange.FundHistory, len(withdrawalsData.Items)+len(depositsData.Items))
	for x := range depositsData.Items {
		fundingData[x] = exchange.FundHistory{
			Timestamp:    depositsData.Items[x].CreatedAt.Time(),
			ExchangeName: ku.Name,
			TransferType: "deposit",
			CryptoTxID:   depositsData.Items[x].WalletTxID,
			Status:       depositsData.Items[x].Status,
			Amount:       depositsData.Items[x].Amount,
			Currency:     depositsData.Items[x].Currency,
		}
	}
	length := len(depositsData.Items)
	for x := range withdrawalsData.Items {
		fundingData[length+x] = exchange.FundHistory{
			Fee:             withdrawalsData.Items[x].Fee,
			Timestamp:       withdrawalsData.Items[x].UpdatedAt.Time(),
			ExchangeName:    ku.Name,
			TransferType:    "withdrawal",
			CryptoToAddress: withdrawalsData.Items[x].Address,
			CryptoTxID:      withdrawalsData.Items[x].WalletTxID,
			Status:          withdrawalsData.Items[x].Status,
			Amount:          withdrawalsData.Items[x].Amount,
			Currency:        withdrawalsData.Items[x].Currency,
			TransferID:      withdrawalsData.Items[x].ID,
		}
	}
	return fundingData, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (ku *Kucoin) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	err := ku.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		var withdrawals *HistoricalDepositWithdrawalResponse
		withdrawals, err = ku.GetHistoricalWithdrawalList(ctx, c.String(), "", time.Time{}, time.Time{}, 0, 0)
		if err != nil {
			return nil, err
		}
		resp := make([]exchange.WithdrawalHistory, len(withdrawals.Items))
		for x := range withdrawals.Items {
			resp[x] = exchange.WithdrawalHistory{
				Status:       withdrawals.Items[x].Status,
				CryptoTxID:   withdrawals.Items[x].WalletTxID,
				Timestamp:    withdrawals.Items[x].CreatedAt.Time(),
				Amount:       withdrawals.Items[x].Amount,
				TransferType: "withdrawal",
				Currency:     c.String(),
			}
		}
		return resp, nil
	case asset.Futures:
		var futuresWithdrawals *FuturesWithdrawalsListResponse
		futuresWithdrawals, err = ku.GetFuturesWithdrawalList(ctx, c.String(), "", time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		resp := make([]exchange.WithdrawalHistory, len(futuresWithdrawals.Items))
		for y := range futuresWithdrawals.Items {
			resp[y] = exchange.WithdrawalHistory{
				Status:       futuresWithdrawals.Items[y].Status,
				CryptoTxID:   futuresWithdrawals.Items[y].WalletTxID,
				Timestamp:    futuresWithdrawals.Items[y].CreatedAt.Time(),
				Amount:       futuresWithdrawals.Items[y].Amount,
				Currency:     c.String(),
				TransferType: "withdrawal",
			}
		}
		return resp, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (ku *Kucoin) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	const limit = 1000
	switch assetType {
	case asset.Futures:
		tradeData, err := ku.GetFuturesTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		var side order.Side
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[0].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          tradeData[i].TradeID,
				Exchange:     ku.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].FilledTime.Time(),
				Side:         side,
			})
		}
	case asset.Spot, asset.Margin:
		tradeData, err := ku.GetTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		var side order.Side
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[0].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				TID:          tradeData[i].Sequence,
				Exchange:     ku.Name,
				CurrencyPair: p,
				Side:         side,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].Time.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	if ku.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(ku.Name, resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (ku *Kucoin) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
func (ku *Kucoin) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	err = ku.CurrencyPairs.IsAssetEnabled(s.AssetType)
	if err != nil {
		return nil, err
	}
	if s.Type != order.UnknownType && s.Type != order.Limit && s.Type != order.Market {
		return nil, fmt.Errorf("%w only limit and market are supported", order.ErrTypeIsInvalid)
	}
	s.Pair, err = ku.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Futures:
		o, err := ku.PostFuturesOrder(ctx, s.ClientOrderID, s.Side.Lower(), s.Pair.Upper().String(), s.Type.Lower(), "", "", "", "", s.Amount, s.Price, s.TriggerPrice, s.Leverage, 0, s.ReduceOnly, false, false, s.PostOnly, s.Hidden, false)
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(o)
	case asset.Spot:
		if s.ClientID != "" && s.ClientOrderID == "" {
			s.ClientOrderID = s.ClientID
		}
		o, err := ku.PostOrder(ctx, s.ClientOrderID, s.Side.Lower(), s.Pair.String(), s.Type.Lower(), "", "", "", s.Amount, s.Price, 0, 0, 0, s.PostOnly, s.Hidden, false)
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(o)
	case asset.Margin:
		o, err := ku.PostMarginOrder(ctx, s.ClientOrderID, s.Side.Lower(), s.Pair.String(), s.Type.Lower(), "", "", s.MarginMode, "", s.Price, s.Amount, s.TriggerPrice, s.Amount, 0, s.PostOnly, s.Hidden, false, s.AutoBorrow)
		if err != nil {
			return nil, err
		}
		ret, err := s.DeriveSubmitResponse(o.OrderID)
		if err != nil {
			return nil, err
		}
		ret.BorrowSize = o.BorrowSize
		ret.LoanApplyID = o.LoanApplyID
		return ret, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, s.AssetType)
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (ku *Kucoin) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (ku *Kucoin) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	err := ku.CurrencyPairs.IsAssetEnabled(ord.AssetType)
	if err != nil {
		return err
	}
	err = ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot, asset.Margin:
		if ord.OrderID == "" && ord.ClientOrderID == "" {
			return errors.New("either OrderID or ClientSuppliedOrderID must be specified")
		}
		if ord.OrderID != "" {
			_, err = ku.CancelSingleOrder(ctx, ord.OrderID)
		} else if ord.ClientOrderID != "" || ord.ClientID != "" {
			if ord.ClientID != "" && ord.ClientOrderID == "" {
				ord.ClientOrderID = ord.ClientID
			}
			_, err = ku.CancelOrderByClientOID(ctx, ord.ClientOrderID)
		}
		return err
	case asset.Futures:
		_, err := ku.CancelFuturesOrder(ctx, ord.OrderID)
		if err != nil {
			return err
		}
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (ku *Kucoin) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (ku *Kucoin) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if err := ku.CurrencyPairs.IsAssetEnabled(orderCancellation.AssetType); err != nil {
		return order.CancelAllResponse{}, err
	}
	result := order.CancelAllResponse{}
	err := orderCancellation.Validate()
	if err != nil {
		return result, err
	}
	orderCancellation.Pair, err = ku.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
	if err != nil {
		return result, err
	}
	var values []string
	switch orderCancellation.AssetType {
	case asset.Margin, asset.Spot:
		var pairString string
		if !orderCancellation.Pair.IsEmpty() {
			pairString = orderCancellation.Pair.String()
		}
		tradeType := ku.accountToTradeTypeString(orderCancellation.AssetType, orderCancellation.MarginMode)
		values, err = ku.CancelAllOpenOrders(ctx, pairString, tradeType)
		if err != nil {
			return order.CancelAllResponse{}, err
		}
	case asset.Futures:
		values, err = ku.CancelAllFuturesOpenOrders(ctx, orderCancellation.Pair.String())
		if err != nil {
			return result, err
		}
		stopOrders, err := ku.CancelAllFuturesStopOrders(ctx, orderCancellation.Pair.String())
		if err != nil {
			return result, err
		}
		values = append(values, stopOrders...)
	default:
		return order.CancelAllResponse{}, fmt.Errorf("%w %v", asset.ErrNotSupported, orderCancellation.AssetType)
	}
	result.Status = map[string]string{}
	for x := range values {
		result.Status[values[x]] = order.Cancelled.String()
	}
	return result, nil
}

// GetOrderInfo returns order information based on order ID
func (ku *Kucoin) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	if err := ku.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return order.Detail{}, err
	}
	pair, err := ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return order.Detail{}, err
	}
	switch assetType {
	case asset.Futures:
		orderDetail, err := ku.GetFuturesOrderDetails(ctx, orderID)
		if err != nil {
			return order.Detail{}, err
		}
		nPair, err := currency.NewPairFromString(orderDetail.Symbol)
		if err != nil {
			return order.Detail{}, err
		}
		oType, err := order.StringToOrderType(orderDetail.OrderType)
		if err != nil {
			return order.Detail{}, err
		}
		side, err := order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return order.Detail{}, err
		}
		if !pair.IsEmpty() && !nPair.Equal(pair) {
			return order.Detail{}, fmt.Errorf("order with id %s and currency Pair %v does not exist", orderID, pair)
		}
		return order.Detail{
			Exchange:        ku.Name,
			OrderID:         orderDetail.ID,
			Pair:            pair,
			Type:            oType,
			Side:            side,
			AssetType:       assetType,
			ExecutedAmount:  orderDetail.DealSize,
			RemainingAmount: orderDetail.Size - orderDetail.DealSize,
			Amount:          orderDetail.Size,
			Price:           orderDetail.Price,
			Date:            orderDetail.CreatedAt.Time()}, nil
	case asset.Spot, asset.Margin:
		orderDetail, err := ku.GetOrderByID(ctx, orderID)
		if err != nil {
			return order.Detail{}, err
		}
		nPair, err := currency.NewPairFromString(orderDetail.Symbol)
		if err != nil {
			return order.Detail{}, err
		}
		oType, err := order.StringToOrderType(orderDetail.Type)
		if err != nil {
			return order.Detail{}, err
		}
		side, err := order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return order.Detail{}, err
		}
		if !pair.IsEmpty() && !nPair.Equal(pair) {
			return order.Detail{}, fmt.Errorf("order with id %s and currency Pair %v does not exist", orderID, pair)
		}
		return order.Detail{
			Exchange:        ku.Name,
			OrderID:         orderDetail.ID,
			Pair:            pair,
			Type:            oType,
			Side:            side,
			Fee:             orderDetail.Fee,
			AssetType:       assetType,
			ExecutedAmount:  orderDetail.DealSize,
			RemainingAmount: orderDetail.Size - orderDetail.DealSize,
			Amount:          orderDetail.Size,
			Price:           orderDetail.Price,
			Date:            orderDetail.CreatedAt.Time(),
		}, nil
	default:
		return order.Detail{}, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (ku *Kucoin) GetDepositAddress(ctx context.Context, c currency.Code, accountID, chain string) (*deposit.Address, error) {
	ad, err := ku.GetDepositAddressesV2(ctx, c.Upper().String())
	if err != nil {
		fad, err := ku.GetFuturesDepositAddress(ctx, c.String())
		if err != nil {
			return nil, err
		}
		return &deposit.Address{
			Address: fad.Address,
			Chain:   fad.Chain,
			Tag:     fad.Memo,
		}, nil
	}
	if len(ad) > 1 {
		return nil, errMultipleDepositAddress
	} else if len(ad) == 0 {
		return nil, errNoDepositAddress
	}
	return &deposit.Address{
		Address: ad[0].Address,
		Chain:   ad[0].Chain,
		Tag:     ad[0].Memo,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
// The endpoint was deprecated for futures, please transfer assets from the FUTURES account to the MAIN account first, and then withdraw from the MAIN account
func (ku *Kucoin) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawalID, err := ku.ApplyWithdrawal(ctx, withdrawRequest.Currency.Upper().String(), withdrawRequest.Crypto.Address, withdrawRequest.Crypto.AddressTag, withdrawRequest.Description, withdrawRequest.Crypto.Chain, "INTERNAL", false, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: withdrawalID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (ku *Kucoin) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (ku *Kucoin) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (ku *Kucoin) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := ku.CurrencyPairs.IsAssetEnabled(getOrdersRequest.AssetType)
	if err != nil {
		return nil, err
	}
	if getOrdersRequest.Validate() != nil {
		return nil, err
	}
	format, err := ku.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	pair := ""
	orders := []order.Detail{}
	switch getOrdersRequest.AssetType {
	case asset.Futures:
		if len(getOrdersRequest.Pairs) == 1 {
			pair = format.Format(getOrdersRequest.Pairs[0])
		}
		futuresOrders, err := ku.GetFuturesOrders(ctx, "active", pair, getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range futuresOrders.Items {
			if !futuresOrders.Items[x].IsActive {
				return nil, err
			}
			dPair, err := currency.NewPairFromString(futuresOrders.Items[x].Symbol)
			if err != nil {
				return nil, err
			}
			if len(getOrdersRequest.Pairs) == 1 && !dPair.Equal(getOrdersRequest.Pairs[x]) {
				continue
			} else if len(getOrdersRequest.Pairs) > 1 {
				found := false
				for i := range getOrdersRequest.Pairs {
					if !getOrdersRequest.Pairs[i].Equal(dPair) {
						continue
					}
					found = true
				}
				if !found {
					continue
				}
			}
			side, err := order.StringToOrderSide(futuresOrders.Items[x].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(futuresOrders.Items[x].OrderType)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				OrderID:         futuresOrders.Items[x].ID,
				Amount:          futuresOrders.Items[x].Size,
				RemainingAmount: futuresOrders.Items[x].Size - futuresOrders.Items[x].FilledSize,
				ExecutedAmount:  futuresOrders.Items[x].FilledSize,
				Exchange:        ku.Name,
				Date:            futuresOrders.Items[x].CreatedAt.Time(),
				LastUpdated:     futuresOrders.Items[x].UpdatedAt.Time(),
				Price:           futuresOrders.Items[x].Price,
				Side:            side,
				Type:            oType,
				Pair:            dPair,
			})
		}
	case asset.Spot, asset.Margin:
		if len(getOrdersRequest.Pairs) == 1 {
			pair = format.Format(getOrdersRequest.Pairs[0])
		}
		spotOrders, err := ku.GetOrders(ctx, "active", pair, getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range spotOrders.Items {
			if !spotOrders.Items[x].IsActive {
				continue
			}
			dPair, err := currency.NewPairFromString(spotOrders.Items[x].Symbol)
			if err != nil {
				return nil, err
			}
			if len(getOrdersRequest.Pairs) == 1 && !dPair.Equal(getOrdersRequest.Pairs[x]) {
				continue
			} else if len(getOrdersRequest.Pairs) > 1 {
				found := false
				for i := range getOrdersRequest.Pairs {
					if !getOrdersRequest.Pairs[i].Equal(dPair) {
						continue
					}
					found = true
				}
				if !found {
					continue
				}
			}
			side, err := order.StringToOrderSide(spotOrders.Items[x].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(spotOrders.Items[x].TradeType)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				OrderID:         spotOrders.Items[x].ID,
				Amount:          spotOrders.Items[x].Size,
				RemainingAmount: spotOrders.Items[x].Size - spotOrders.Items[x].DealSize,
				ExecutedAmount:  spotOrders.Items[x].DealSize,
				Exchange:        ku.Name,
				Date:            spotOrders.Items[x].CreatedAt.Time(),
				Price:           spotOrders.Items[x].Price,
				Side:            side,
				Type:            oType,
				Pair:            dPair,
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (ku *Kucoin) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) (order.FilteredOrders, error) {
	err := ku.CurrencyPairs.IsAssetEnabled(getOrdersRequest.AssetType)
	if err != nil {
		return nil, err
	}
	if getOrdersRequest.Validate() != nil {
		return nil, err
	}
	var orders []order.Detail
	var orderSide order.Side
	var orderStatus order.Status
	var oType order.Type
	var pair currency.Pair
	switch getOrdersRequest.AssetType {
	case asset.Futures:
		var futuresOrders *FutureOrdersResponse
		var newOrders *FutureOrdersResponse
		if len(getOrdersRequest.Pairs) == 0 {
			futuresOrders, err = ku.GetFuturesOrders(ctx, "", "", getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
			if err != nil {
				return nil, err
			}
		} else {
			for x := range getOrdersRequest.Pairs {
				getOrdersRequest.Pairs[x], err = ku.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
				if err != nil {
					return nil, err
				}
				newOrders, err = ku.GetFuturesOrders(ctx, "", getOrdersRequest.Pairs[x].String(), getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
				if err != nil {
					return nil, fmt.Errorf("%w while fetching for symbol %s", err, getOrdersRequest.Pairs[x].String())
				}
				if futuresOrders == nil {
					futuresOrders = newOrders
				} else {
					futuresOrders.Items = append(futuresOrders.Items, newOrders.Items...)
				}
			}
		}
		orders = make(order.FilteredOrders, len(futuresOrders.Items))
		for i := range orders {
			orderSide, err = order.StringToOrderSide(futuresOrders.Items[i].Side)
			if err != nil {
				return nil, err
			}
			pair, err = currency.NewPairFromString(futuresOrders.Items[i].Symbol)
			if err != nil {
				return nil, err
			}
			oType, err = order.StringToOrderType(futuresOrders.Items[i].OrderType)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ku.Name, err)
			}
			orders[i] = order.Detail{
				Price:           futuresOrders.Items[i].Price,
				Amount:          futuresOrders.Items[i].Size,
				ExecutedAmount:  futuresOrders.Items[i].DealSize,
				RemainingAmount: futuresOrders.Items[i].Size - futuresOrders.Items[i].DealSize,
				Date:            futuresOrders.Items[i].CreatedAt.Time(),
				Exchange:        ku.Name,
				OrderID:         futuresOrders.Items[i].ID,
				Side:            orderSide,
				Status:          orderStatus,
				Type:            oType,
				Pair:            pair,
			}
			orders[i].InferCostsAndTimes()
		}
	case asset.Spot, asset.Margin:
		var responseOrders *OrdersListResponse
		var newOrders *OrdersListResponse
		if len(getOrdersRequest.Pairs) == 0 {
			responseOrders, err = ku.GetOrders(ctx, "", "", getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
			if err != nil {
				return nil, err
			}
		} else {
			for x := range getOrdersRequest.Pairs {
				getOrdersRequest.Pairs[x].Delimiter = currency.DashDelimiter
				newOrders, err = ku.GetOrders(ctx, "", getOrdersRequest.Pairs[x].String(), getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
				if err != nil {
					return nil, fmt.Errorf("%w while fetching for symbol %s", err, getOrdersRequest.Pairs[x].String())
				}
				if responseOrders == nil {
					responseOrders = newOrders
				} else {
					responseOrders.Items = append(responseOrders.Items, newOrders.Items...)
				}
			}
		}
		orders = make([]order.Detail, len(responseOrders.Items))
		for i := range orders {
			orderSide, err = order.StringToOrderSide(responseOrders.Items[i].Side)
			if err != nil {
				return nil, err
			}
			var orderStatus order.Status
			pair, err = currency.NewPairFromString(responseOrders.Items[i].Symbol)
			if err != nil {
				return nil, err
			}
			pair.Delimiter = currency.DashDelimiter
			var oType order.Type
			oType, err = order.StringToOrderType(responseOrders.Items[i].Type)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", ku.Name, err)
			}
			orders[i] = order.Detail{
				Price:           responseOrders.Items[i].Price,
				Amount:          responseOrders.Items[i].Size,
				ExecutedAmount:  responseOrders.Items[i].DealSize,
				RemainingAmount: responseOrders.Items[i].Size - responseOrders.Items[i].DealSize,
				Date:            responseOrders.Items[i].CreatedAt.Time(),
				Exchange:        ku.Name,
				OrderID:         responseOrders.Items[i].ID,
				Side:            orderSide,
				Status:          orderStatus,
				Type:            oType,
				Pair:            pair,
			}
			orders[i].InferCostsAndTimes()
		}
	}
	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)
	order.FilterOrdersByType(&orders, getOrdersRequest.Type)
	err = order.FilterOrdersByTimeRange(&orders, getOrdersRequest.StartTime, getOrdersRequest.EndTime)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", ku.Name, err)
	}
	order.FilterOrdersByPairs(&orders, getOrdersRequest.Pairs)
	return getOrdersRequest.Filter(ku.Name, orders), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (ku *Kucoin) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !ku.AreCredentialsValid(ctx) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyWithdrawalFee,
		exchange.CryptocurrencyTradeFee:
		fee, err := ku.GetBasicFee(ctx, "0")
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			return feeBuilder.Amount * fee.MakerFeeRate, nil
		}
		return feeBuilder.Amount * fee.TakerFeeRate, nil
	case exchange.OfflineTradeFee:
		return feeBuilder.Amount * 0.001, nil
	case exchange.CryptocurrencyDepositFee:
		return 0, nil
	default:
		if !feeBuilder.FiatCurrency.IsEmpty() {
			fee, err := ku.GetBasicFee(ctx, "1")
			if err != nil {
				return 0, err
			}
			if feeBuilder.IsMaker {
				return feeBuilder.Amount * fee.MakerFeeRate, nil
			}
			return feeBuilder.Amount * fee.TakerFeeRate, nil
		}
		return 0, fmt.Errorf("can't construct fee")
	}
}

// ValidateCredentials validates current credentials used for wrapper
func (ku *Kucoin) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	err := ku.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return err
	}
	_, err = ku.UpdateAccountInfo(ctx, assetType)
	return ku.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (ku *Kucoin) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	err := ku.ValidateKline(pair, a, interval)
	if err != nil {
		return nil, err
	}
	pair, err = ku.FormatExchangeCurrency(pair, a)
	if err != nil {
		return nil, err
	}
	ret := &kline.Item{
		Exchange: ku.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	switch a {
	case asset.Futures:
		var candles []FuturesKline
		pair.Delimiter = ""
		candles, err := ku.GetFuturesKline(ctx, "30", pair.String(), start, end)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			ret.Candles = append(
				ret.Candles, kline.Candle{
					Time:   candles[x].StartTime,
					Open:   candles[x].Open,
					High:   candles[x].High,
					Low:    candles[x].Low,
					Close:  candles[x].Close,
					Volume: candles[x].Volume,
				})
		}
		ret.SortCandlesByTimestamp(false)
	case asset.Spot:
		intervalString, err := ku.intervalToString(interval)
		if err != nil {
			return nil, err
		}
		var candles []Kline
		pair.Delimiter = currency.DashDelimiter
		candles, err = ku.GetKlines(ctx, pair.String(), intervalString, start, end)
		if err != nil {
			return ret, err
		}
		for x := range candles {
			ret.Candles = append(
				ret.Candles, kline.Candle{
					Time:   candles[x].StartTime,
					Open:   candles[x].Open,
					High:   candles[x].High,
					Low:    candles[x].Low,
					Close:  candles[x].Close,
					Volume: candles[x].Volume,
				})
		}
		ret.SortCandlesByTimestamp(false)
	}
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (ku *Kucoin) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	if err := ku.ValidateKline(pair, a, interval); err != nil {
		return nil, err
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, ku.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return nil, err
	}
	ret := &kline.Item{
		Exchange: ku.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	for x := range dates.Ranges {
		switch a {
		case asset.Futures:
			var candles []FuturesKline
			pair.Delimiter = ""
			candles, err = ku.GetFuturesKline(ctx, "720", pair.String(), dates.Ranges[x].Start.Time, dates.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for x := range candles {
				ret.Candles = append(
					ret.Candles, kline.Candle{
						Time:   candles[x].StartTime,
						Open:   candles[x].Open,
						High:   candles[x].High,
						Low:    candles[x].Low,
						Close:  candles[x].Close,
						Volume: candles[x].Volume,
					})
			}
			ret.SortCandlesByTimestamp(false)
		case asset.Spot:
			var intervalString string
			intervalString, err = ku.intervalToString(interval)
			if err != nil {
				return ret, err
			}
			var candles []Kline
			pair.Delimiter = currency.DashDelimiter
			candles, err = ku.GetKlines(ctx, pair.String(), intervalString, dates.Ranges[x].Start.Time, dates.Ranges[x].End.Time)
			if err != nil {
				return ret, err
			}
			for x := range candles {
				ret.Candles = append(
					ret.Candles, kline.Candle{
						Time:   candles[x].StartTime,
						Open:   candles[x].Open,
						High:   candles[x].High,
						Low:    candles[x].Low,
						Close:  candles[x].Close,
						Volume: candles[x].Volume,
					})
			}
			ret.SortCandlesByTimestamp(false)
		}
	}
	err = dates.SetHasDataFromCandles(ret.Candles)
	if err != nil {
		return nil, err
	}
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", ku.Name, summary)
	}
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}

// GetServerTime returns the current exchange server time.
func (ku *Kucoin) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	switch a {
	case asset.Spot, asset.Margin:
		return ku.GetCurrentServerTime(ctx)
	case asset.Futures:
		return ku.GetFuturesServerTime(ctx)
	default:
		return time.Time{}, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (ku *Kucoin) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencyDetail, err := ku.GetCurrencyDetail(ctx, cryptocurrency.String(), "")
	if err != nil {
		return nil, err
	}
	chains := make([]string, 0, len(currencyDetail.Chains))
	for x := range currencyDetail.Chains {
		chains = append(chains, currencyDetail.Chains[x].Name)
	}
	return chains, nil
}
