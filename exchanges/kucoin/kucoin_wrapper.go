package kucoin

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// SetDefaults sets the basic defaults for Kucoin
func (e *Exchange) SetDefaults() {
	e.Name = "Kucoin"
	e.Enabled = true
	e.Verbose = false

	e.API.CredentialsValidator.RequiresKey = true
	e.API.CredentialsValidator.RequiresSecret = true
	e.API.CredentialsValidator.RequiresClientID = true

	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		ps := currency.PairStore{
			AssetEnabled:  true,
			RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
			ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		}
		if a == asset.Futures {
			ps.RequestFormat.Delimiter = ""
			ps.ConfigFormat.Delimiter = currency.UnderscoreDelimiter
		}
		if err := e.SetAssetPairStore(a, ps); err != nil {
			log.Errorf(log.ExchangeSys, "%s error storing %q default asset formats: %s", e.Name, a, err)
		}
	}

	e.Features = exchange.Features{
		CurrencyTranslations: currency.NewTranslations(map[currency.Code]currency.Code{
			currency.XBT:   currency.BTC,
			currency.USDTM: currency.USDT,
			currency.USDM:  currency.USD,
			currency.USDCM: currency.USDC,
		}),
		TradingRequirements: protocol.TradingRequirements{
			ClientOrderID: true,
		},
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				TickerBatching:    true,
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
			FuturesCapabilities: exchange.FuturesCapabilities{
				Positions:                 true,
				Leverage:                  true,
				CollateralMode:            true,
				FundingRates:              true,
				MaximumFundingRateHistory: kline.ThreeMonth.Duration(),
				SupportedFundingRateFrequencies: map[kline.Interval]bool{
					kline.EightHour: true,
				},
				FundingRateBatching: map[asset.Item]bool{
					asset.Futures: true,
				},
				OpenInterest: exchange.OpenInterestSupport{
					Supported:          true,
					SupportedViaTicker: true,
					SupportsRestBatch:  true,
				},
			},
			MaximumOrderHistory: kline.OneDay.Duration() * 7,
			WithdrawPermissions: exchange.AutoWithdrawCrypto,
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
					kline.IntervalCapacity{Interval: kline.OneWeek},
				),
				GlobalResultLimit: 500,
			},
		},
		Subscriptions: defaultSubscriptions.Clone(),
	}

	var err error
	e.Requester, err = request.New(e.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	e.API.Endpoints = e.NewEndpoints()
	err = e.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      kucoinAPIURL,
		exchange.RestFutures:   kucoinFuturesAPIURL,
		exchange.WebsocketSpot: kucoinWebsocketURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
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

	e.checkSubscriptions()

	wsRunningEndpoint, err := e.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}
	err = e.Websocket.Setup(
		&websocket.ManagerSetup{
			ExchangeConfig:        exch,
			DefaultURL:            kucoinWebsocketURL,
			RunningURL:            wsRunningEndpoint,
			Connector:             e.WsConnect,
			Subscriber:            e.Subscribe,
			Unsubscriber:          e.Unsubscribe,
			GenerateSubscriptions: e.generateSubscriptions,
			Features:              &e.Features.Supports.WebsocketCapabilities,
			OrderbookBufferConfig: buffer.Config{
				SortBuffer:            true,
				SortBufferByUpdateIDs: true,
			},
			TradeFeed: e.Features.Enabled.TradeFeed,
		})
	if err != nil {
		return err
	}
	return e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		RateLimit:            request.NewRateLimitWithWeight(time.Second, 2, 1),
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (e *Exchange) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	switch assetType {
	case asset.Futures:
		myPairs, err := e.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(myPairs))
		for x := range myPairs {
			if strings.ToLower(myPairs[x].Status) != "open" { //nolint:gocritic // strings.ToLower is faster
				continue
			}
			quote := currency.NewCode(myPairs[x].Symbol[len(myPairs[x].BaseCurrency.String()):])
			pairs = pairs.Add(currency.NewPair(myPairs[x].BaseCurrency, quote))
		}
		configFormat, err := e.GetPairFormat(asset.Futures, false)
		if err != nil {
			return nil, err
		}
		return pairs.Format(configFormat), nil
	case asset.Spot, asset.Margin:
		myPairs, err := e.GetSymbols(ctx, "")
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(myPairs))
		for x := range myPairs {
			if !myPairs[x].EnableTrading || (assetType == asset.Margin && !myPairs[x].IsMarginEnabled) {
				continue
			}
			// Symbol field must be used to generate pair as this is the symbol
			// to fetch data from the API. e.g. BSV-USDT name is BCHSV-USDT as symbol.
			pairs = pairs.Add(myPairs[x].Symbol)
		}
		return pairs, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (e *Exchange) UpdateTradablePairs(ctx context.Context) error {
	assets := e.GetAssetTypes(true)
	for a := range assets {
		pairs, err := e.FetchTradablePairs(ctx, assets[a])
		if err != nil {
			return err
		}
		if err := e.UpdatePairs(pairs, assets[a], false); err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (e *Exchange) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if err := e.UpdateTickers(ctx, assetType); err != nil {
		return nil, err
	}
	return ticker.GetTicker(e.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (e *Exchange) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	var errs error
	switch assetType {
	case asset.Futures:
		ticks, err := e.GetFuturesOpenContracts(ctx)
		if err != nil {
			return err
		}
		pairs, err := e.GetEnabledPairs(asset.Futures)
		if err != nil {
			return err
		}
		for x := range ticks {
			pair := currency.NewPair(ticks[x].BaseCurrency,
				currency.NewCode(ticks[x].Symbol[len(ticks[x].BaseCurrency.String()):]))
			if !pairs.Contains(pair, true) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks[x].LastTradePrice,
				High:         ticks[x].HighPrice,
				Low:          ticks[x].LowPrice,
				Volume:       ticks[x].VolumeOf24h,
				OpenInterest: ticks[x].OpenInterest.Float64(),
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    assetType,
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	case asset.Spot, asset.Margin:
		ticks, err := e.GetTickers(ctx)
		if err != nil {
			return err
		}
		for t := range ticks.Tickers {
			pair, enabled, err := e.MatchSymbolCheckEnabled(ticks.Tickers[t].Symbol, assetType, true)
			if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
				return err
			}
			if !enabled {
				continue
			}

			err = ticker.ProcessTicker(&ticker.Price{
				Last:         ticks.Tickers[t].Last,
				High:         ticks.Tickers[t].High,
				Low:          ticks.Tickers[t].Low,
				Volume:       ticks.Tickers[t].Volume,
				Ask:          ticks.Tickers[t].Sell,
				Bid:          ticks.Tickers[t].Buy,
				Pair:         pair,
				ExchangeName: e.Name,
				AssetType:    assetType,
				LastUpdated:  ticks.Time.Time(),
			})
			if err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return errs
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (e *Exchange) UpdateOrderbook(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
	err := e.CurrencyPairs.IsAssetEnabled(a)
	if err != nil {
		return nil, err
	}
	p, err = e.FormatExchangeCurrency(p, a)
	if err != nil {
		return nil, err
	}
	var ordBook *Orderbook
	switch a {
	case asset.Futures:
		ordBook, err = e.GetFuturesOrderbook(ctx, p.String())
	case asset.Spot, asset.Margin:
		ordBook, err = e.GetPartOrderbook100(ctx, p.String())
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	if err != nil {
		return nil, err
	}

	ob := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              p,
		Asset:             a,
		ValidateOrderbook: e.ValidateOrderbook,
		Asks:              ordBook.Asks,
		Bids:              ordBook.Bids,
	}

	if err := ob.Process(); err != nil {
		return nil, err
	}

	return orderbook.Get(e.Name, p, a)
}

// UpdateAccountBalances retrieves currency balances
func (e *Exchange) UpdateAccountBalances(ctx context.Context, assetType asset.Item) (accounts.SubAccounts, error) {
	if err := e.CurrencyPairs.IsAssetEnabled(assetType); err != nil {
		return nil, fmt.Errorf("%w: %q", asset.ErrNotSupported, assetType)
	}
	subAccts := accounts.SubAccounts{accounts.NewSubAccount(assetType, "")}
	switch assetType {
	case asset.Futures:
		for _, settlement := range []string{"XBT", "USDT"} {
			resp, err := e.GetFuturesAccountOverview(ctx, settlement)
			if err != nil {
				return nil, err
			}
			subAccts[0].Balances.Set(resp.Currency, accounts.Balance{
				Total: resp.AvailableBalance + resp.FrozenFunds,
				Hold:  resp.FrozenFunds,
				Free:  resp.AvailableBalance,
			})
		}
	case asset.Spot, asset.Margin:
		resp, err := e.GetAllAccounts(ctx, currency.EMPTYCODE, "")
		if err != nil {
			return nil, err
		}
		for i := range resp {
			if resp[i].AccountType == "margin" && assetType == asset.Spot {
				continue
			} else if resp[i].AccountType == "trade" && assetType == asset.Margin {
				continue
			}
			subAccts[0].Balances.Set(resp[i].Currency, accounts.Balance{
				Total: resp[i].Balance.Float64(),
				Hold:  resp[i].Holds.Float64(),
				Free:  resp[i].Available.Float64(),
			})
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return subAccts, e.Accounts.Save(ctx, subAccts, true)
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (e *Exchange) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	withdrawalsData, err := e.GetWithdrawalList(ctx, currency.EMPTYCODE, "", time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	depositsData, err := e.GetHistoricalDepositList(ctx, currency.EMPTYCODE, "", time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	fundingData := make([]exchange.FundingHistory, len(withdrawalsData.Items)+len(depositsData.Items))
	for x := range depositsData.Items {
		fundingData[x] = exchange.FundingHistory{
			Timestamp:    depositsData.Items[x].CreatedAt.Time(),
			ExchangeName: e.Name,
			TransferType: "deposit",
			CryptoTxID:   depositsData.Items[x].WalletTxID,
			Status:       depositsData.Items[x].Status,
			Amount:       depositsData.Items[x].Amount,
			Currency:     depositsData.Items[x].Currency,
		}
	}
	length := len(depositsData.Items)
	for x := range withdrawalsData.Items {
		fundingData[length+x] = exchange.FundingHistory{
			Fee:             withdrawalsData.Items[x].Fee,
			Timestamp:       withdrawalsData.Items[x].UpdatedAt.Time(),
			ExchangeName:    e.Name,
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
func (e *Exchange) GetWithdrawalsHistory(ctx context.Context, c currency.Code, assetType asset.Item) ([]exchange.WithdrawalHistory, error) {
	if !e.SupportsAsset(assetType) {
		return nil, asset.ErrNotSupported
	}
	var withdrawals *HistoricalDepositWithdrawalResponse
	withdrawals, err := e.GetHistoricalWithdrawalList(ctx, c.Upper(), "", time.Time{}, time.Time{})
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
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (e *Exchange) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	p, err := e.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	var resp []trade.Data
	switch assetType {
	case asset.Futures:
		tradeData, err := e.GetFuturesTradeHistory(ctx, p.String())
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
				Exchange:     e.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].FilledTime.Time(),
				Side:         side,
			})
		}
	case asset.Spot, asset.Margin:
		tradeData, err := e.GetTradeHistory(ctx, p.String())
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
				Exchange:     e.Name,
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
	if e.IsSaveTradeDataEnabled() {
		err := trade.AddTradesToBuffer(resp...)
		if err != nil {
			return nil, err
		}
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (e *Exchange) GetHistoricTrades(_ context.Context, _ currency.Pair, _ asset.Item, _, _ time.Time) ([]trade.Data, error) {
	return nil, common.ErrFunctionNotSupported
}

// SubmitOrder submits a new order
// For OCO (One Cancels the Other) orders, the StopLoss parameters under the order submission argument field RiskManagementModes are treated as stop values,
// and the TakeProfit parameters are treated as limit order.
func (e *Exchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	sideString, err := e.OrderSideString(s.Side)
	if err != nil {
		return nil, err
	}
	s.Pair, err = e.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	var o string
	switch s.AssetType {
	case asset.Futures:
		if s.Leverage == 0 {
			s.Leverage = 1
		}
		var orderType, stopOrderType, stopOrderBoundary string
		switch s.Type {
		case order.Stop, order.StopLimit, order.TrailingStop:
			orderType = "limit"
			if s.TriggerPrice == 0 {
				break
			}
			switch s.TriggerPriceType {
			case order.IndexPrice:
				stopOrderType = "IP"
			case order.MarkPrice:
				stopOrderType = "MP"
			case order.LastPrice:
				stopOrderType = "TP"
			}
			switch s.Type {
			case order.StopLimit:
				switch s.Side {
				case order.Sell:
					stopOrderBoundary = "up"
				case order.Buy:
					stopOrderBoundary = "down"
				}
			case order.TrailingStop, order.Stop:
				switch s.Side {
				case order.Sell:
					// Stop-loss when order type is order.Stop
					stopOrderBoundary = "down"
				case order.Buy:
					// Take Profit when order type is order.Stop
					stopOrderBoundary = "up"
				}
			}
		case order.Market, order.Limit:
			orderType = s.Type.Lower()
		default:
			return nil, order.ErrUnsupportedOrderType
		}
		o, err = e.PostFuturesOrder(ctx, &FuturesOrderParam{
			ClientOrderID: s.ClientOrderID,
			Side:          sideString,
			Symbol:        s.Pair,
			OrderType:     orderType,
			Size:          s.Amount,
			Price:         s.Price,
			Leverage:      s.Leverage,
			VisibleSize:   0,
			ReduceOnly:    s.ReduceOnly,
			PostOnly:      s.TimeInForce.Is(order.PostOnly),
			Hidden:        s.Hidden,
			Stop:          stopOrderBoundary,
			StopPrice:     s.TriggerPrice,
			StopPriceType: stopOrderType,
			Iceberg:       s.Iceberg,
		})
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(o)
	case asset.Spot:
		switch s.Type {
		case order.Limit, order.Market, order.StopLimit, order.StopMarket:
			var oType order.Type
			switch s.Type {
			case order.Limit, order.StopLimit:
				oType = order.Limit
			case order.Market, order.StopMarket:
				oType = order.Market
			}
			var timeInForce string
			if oType == order.Limit {
				switch {
				case s.TimeInForce.Is(order.FillOrKill) || s.TimeInForce.Is(order.ImmediateOrCancel):
					timeInForce = s.TimeInForce.String()
				default:
					timeInForce = order.GoodTillCancel.String()
				}
			}
			var stopType string
			var stopPrice float64
			switch {
			case s.RiskManagementModes.StopLoss.Enabled && s.RiskManagementModes.StopEntry.Enabled:
				return nil, errors.New("can not enable more than one risk management")
			case s.RiskManagementModes.StopEntry.Enabled:
				stopType = "entry"
				stopPrice = s.RiskManagementModes.StopEntry.Price
			case s.RiskManagementModes.StopLoss.Enabled:
				stopType = "loss"
				stopPrice = s.RiskManagementModes.StopLoss.Price
			}
			var o string
			if stopType != "" {
				o, err = e.PostStopOrder(ctx,
					s.ClientOrderID,
					sideString,
					s.Pair.String(),
					oType.Lower(), "", stopType, "", SpotTradeType,
					timeInForce, s.Amount, s.Price, stopPrice, 0,
					0, 0, s.TimeInForce.Is(order.PostOnly), s.Hidden, s.Iceberg)
				if err != nil {
					return nil, err
				}
				return s.DeriveSubmitResponse(o)
			}
			o, err = e.PostOrder(ctx, &SpotOrderParam{
				ClientOrderID: s.ClientOrderID,
				Side:          sideString,
				Symbol:        s.Pair,
				OrderType:     s.Type.Lower(),
				Size:          s.Amount,
				Price:         s.Price,
				PostOnly:      s.TimeInForce.Is(order.PostOnly),
				Hidden:        s.Hidden,
				TimeInForce:   timeInForce,
				Iceberg:       s.Iceberg,
				TradeType:     SpotTradeType,
				ReduceOnly:    s.ReduceOnly,
			})
			if err != nil {
				return nil, err
			}
			return s.DeriveSubmitResponse(o)
		case order.OCO:
			switch {
			case !s.RiskManagementModes.TakeProfit.Enabled || s.RiskManagementModes.TakeProfit.Price <= 0:
				return nil, errors.New("take profit price is required")
			case !s.RiskManagementModes.StopLoss.Enabled || s.RiskManagementModes.StopLoss.Price <= 0:
				return nil, errors.New("stop loss price is required")
			}
			switch s.Side {
			case order.Sell:
				if s.RiskManagementModes.TakeProfit.Price <= s.RiskManagementModes.StopLoss.Price {
					return nil, errors.New("stop loss price must be below take profit trigger price for sell orders")
				}
			case order.Buy:
				if s.RiskManagementModes.TakeProfit.Price >= s.RiskManagementModes.StopLoss.Price {
					return nil, errors.New("stop loss price must be greater than take profit trigger price for buy orders")
				}
			}

			limitPrice := s.RiskManagementModes.TakeProfit.Price
			stopPrice := s.RiskManagementModes.StopLoss.Price

			var o string
			o, err = e.PlaceOCOOrder(ctx, &OCOOrderParams{
				Symbol:        s.Pair,
				Side:          sideString,
				Price:         s.Price,
				Size:          s.Amount,
				StopPrice:     stopPrice,
				LimitPrice:    limitPrice,
				ClientOrderID: s.ClientOrderID,
			})
			if err != nil {
				return nil, err
			}
			return s.DeriveSubmitResponse(o)
		default:
			return nil, order.ErrUnsupportedOrderType
		}
	case asset.Margin:
		o, err := e.PostMarginOrder(ctx,
			&MarginOrderParam{
				ClientOrderID: s.ClientOrderID,
				Side:          sideString,
				Symbol:        s.Pair,
				OrderType:     s.Type.Lower(),
				MarginModel:   MarginModeToString(s.MarginType),
				Price:         s.Price,
				Size:          s.Amount,
				VisibleSize:   s.Amount,
				PostOnly:      s.TimeInForce.Is(order.PostOnly),
				Hidden:        s.Hidden,
				AutoBorrow:    s.AutoBorrow,
				AutoRepay:     s.AutoBorrow,
				Iceberg:       s.Iceberg,
			})
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

// MarginModeToString returns a string representation of a MarginMode
func MarginModeToString(mType margin.Type) string {
	switch mType {
	case margin.Isolated:
		return mType.String()
	case margin.Multi:
		return "cross"
	default:
		return ""
	}
}

// ModifyOrder modifies an existing order
func (e *Exchange) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelOrder cancels an order by its corresponding ID number
func (e *Exchange) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	if ord == nil {
		return common.ErrNilPointer
	}
	err := e.CurrencyPairs.IsAssetEnabled(ord.AssetType)
	if err != nil {
		return err
	}
	pairFormat, err := e.GetPairFormat(ord.AssetType, true)
	if err != nil {
		return err
	}
	ord.Pair = ord.Pair.Format(pairFormat)
	switch ord.AssetType {
	case asset.Spot, asset.Margin:
		if ord.OrderID == "" && ord.ClientOrderID == "" {
			return fmt.Errorf("%w, either clientOrderID or OrderID is required", order.ErrOrderIDNotSet)
		}
		switch ord.Type {
		case order.OCO:
			if ord.OrderID != "" {
				_, err = e.CancelOCOOrderByOrderID(ctx, ord.OrderID)
			} else if ord.ClientOrderID != "" {
				_, err = e.CancelOCOOrderByClientOrderID(ctx, ord.ClientOrderID)
			}
		case order.Stop, order.StopLimit:
			if ord.OrderID != "" {
				_, err = e.CancelStopOrder(ctx, ord.OrderID)
			} else {
				_, err = e.CancelStopOrderByClientOrderID(ctx, ord.ClientOrderID, ord.Pair.String())
			}
		default:
			if ord.OrderID != "" {
				_, err = e.CancelSingleOrder(ctx, ord.OrderID)
			} else {
				_, err = e.CancelOrderByClientOID(ctx, ord.ClientOrderID)
			}
		}
		return err
	case asset.Futures:
		if ord.OrderID == "" && ord.ClientOrderID == "" {
			return fmt.Errorf("%w, either clientOrderID or OrderID is required", order.ErrOrderIDNotSet)
		}
		if ord.OrderID == "" {
			if ord.Pair.IsEmpty() {
				return fmt.Errorf("%w, symbol information is required", currency.ErrCurrencyPairEmpty)
			}
			_, err = e.CancelFuturesOrderByClientOrderID(ctx, ord.Pair.String(), ord.ClientOrderID)
		} else {
			_, err = e.CancelFuturesOrderByOrderID(ctx, ord.OrderID)
		}
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, ord.AssetType)
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (e *Exchange) CancelBatchOrders(_ context.Context, _ []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// CancelAllOrders cancels all orders associated with a currency pair
func (e *Exchange) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	if orderCancellation == nil {
		return order.CancelAllResponse{}, common.ErrNilPointer
	}
	err := e.CurrencyPairs.IsAssetEnabled(orderCancellation.AssetType)
	if err != nil {
		return order.CancelAllResponse{}, err
	}
	result := order.CancelAllResponse{}
	err = orderCancellation.Validate()
	if err != nil {
		return result, err
	}
	var pairString string
	if !orderCancellation.Pair.IsEmpty() {
		orderCancellation.Pair, err = e.FormatExchangeCurrency(orderCancellation.Pair, orderCancellation.AssetType)
		if err != nil {
			return result, err
		}
		pairString = orderCancellation.Pair.String()
	}
	var values []string
	switch orderCancellation.AssetType {
	case asset.Margin, asset.Spot:
		var orderIDs []string
		if orderCancellation.OrderID != "" {
			orderIDs = append(orderIDs, orderCancellation.OrderID)
		}
		if orderCancellation.ClientOrderID != "" {
			orderIDs = append(orderIDs, orderCancellation.ClientOrderID)
		}
		switch orderCancellation.Type {
		case order.OCO:
			var response *OCOOrderCancellationResponse
			response, err = e.CancelOCOMultipleOrders(ctx, orderIDs, orderCancellation.Pair.String())
			if err != nil {
				return order.CancelAllResponse{}, err
			}
			values = response.CancelledOrderIDs
		case order.Stop, order.StopLimit:
			values, err = e.CancelStopOrders(ctx,
				orderCancellation.Pair.String(),
				e.AccountToTradeTypeString(orderCancellation.AssetType, MarginModeToString(orderCancellation.MarginType)),
				orderIDs)
			if err != nil {
				return order.CancelAllResponse{}, err
			}
		default:
			tradeType := e.AccountToTradeTypeString(orderCancellation.AssetType, MarginModeToString(orderCancellation.MarginType))
			values, err = e.CancelAllOpenOrders(ctx, pairString, tradeType)
			if err != nil {
				return order.CancelAllResponse{}, err
			}
		}
	case asset.Futures:
		values, err = e.CancelMultipleFuturesLimitOrders(ctx, orderCancellation.Pair.String())
		if err != nil {
			return result, err
		}
		stopOrders, err := e.CancelAllFuturesStopOrders(ctx, orderCancellation.Pair.String())
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
func (e *Exchange) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	err := e.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	pair, err = e.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	switch assetType {
	case asset.Futures:
		var orderDetail *FuturesOrder
		orderDetail, err = e.GetFuturesOrderDetails(ctx, orderID, "")
		if err != nil {
			return nil, err
		}
		var nPair currency.Pair
		nPair, err = currency.NewPairFromString(orderDetail.Symbol)
		if err != nil {
			return nil, err
		}
		var oType order.Type
		oType, err = order.StringToOrderType(orderDetail.OrderType)
		if err != nil {
			return nil, err
		}
		var side order.Side
		side, err = order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return nil, err
		}
		switch side {
		case order.Sell:
			side = order.Short
		case order.Buy:
			side = order.Long
		}
		if !pair.IsEmpty() && !nPair.Equal(pair) {
			return nil, fmt.Errorf("order with id %s and symbol %v does not exist", orderID, pair)
		}
		var oStatus order.Status
		if orderDetail.IsActive {
			oStatus = order.Active
		} else {
			oStatus = order.Closed
		}
		return &order.Detail{
			Exchange:             e.Name,
			OrderID:              orderDetail.ID,
			Pair:                 pair,
			Type:                 oType,
			Side:                 side,
			AssetType:            assetType,
			ExecutedAmount:       orderDetail.DealSize,
			RemainingAmount:      orderDetail.Size - orderDetail.DealSize,
			Amount:               orderDetail.Size,
			Price:                orderDetail.Price,
			Date:                 orderDetail.CreatedAt.Time(),
			HiddenOrder:          orderDetail.Hidden,
			TimeInForce:          StringToTimeInForce(orderDetail.TimeInForce, orderDetail.PostOnly),
			ReduceOnly:           orderDetail.ReduceOnly,
			Leverage:             orderDetail.Leverage,
			AverageExecutedPrice: orderDetail.Price,
			QuoteAmount:          orderDetail.Size,
			ClientOrderID:        orderDetail.ClientOid,
			Status:               oStatus,
			CloseTime:            orderDetail.EndAt.Time(),
			LastUpdated:          orderDetail.UpdatedAt.Time(),
		}, nil
	case asset.Spot, asset.Margin:
		orderDetail, err := e.GetOrderByID(ctx, orderID)
		if err != nil {
			return nil, err
		}
		nPair, err := currency.NewPairFromString(orderDetail.Symbol)
		if err != nil {
			return nil, err
		}
		oType, err := order.StringToOrderType(orderDetail.Type)
		if err != nil {
			return nil, err
		}
		side, err := order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return nil, err
		}
		if !pair.IsEmpty() && !nPair.Equal(pair) {
			return nil, fmt.Errorf("order with id %s and currency Pair %v does not exist", orderID, pair)
		}
		var oStatus order.Status
		if orderDetail.IsActive {
			oStatus = order.Active
		} else {
			oStatus = order.Closed
		}
		var orderAssetType asset.Item
		var mType margin.Type
		switch orderDetail.TradeType {
		case SpotTradeType:
			orderAssetType = asset.Spot
		case CrossMarginTradeType:
			mType = margin.Multi
			orderAssetType = asset.Margin
		case IsolatedMarginTradeType:
			mType = margin.Isolated
			orderAssetType = asset.Margin
		}
		if orderAssetType != assetType {
			return nil, fmt.Errorf("%w, expected order asset type %v, got %v", asset.ErrInvalidAsset, assetType, orderAssetType)
		}
		var remainingSize float64
		if orderDetail.RemainSize.Float64() != 0 {
			remainingSize = orderDetail.RemainSize.Float64()
		} else {
			remainingSize = orderDetail.Size.Float64() - orderDetail.DealSize.Float64()
		}
		return &order.Detail{
			Exchange:             e.Name,
			OrderID:              orderDetail.ID,
			Pair:                 pair,
			Type:                 oType,
			Side:                 side,
			Fee:                  orderDetail.Fee.Float64(),
			AssetType:            assetType,
			ExecutedAmount:       orderDetail.DealSize.Float64(),
			RemainingAmount:      remainingSize,
			Amount:               orderDetail.Size.Float64(),
			Price:                orderDetail.Price.Float64(),
			Date:                 orderDetail.CreatedAt.Time(),
			HiddenOrder:          orderDetail.Hidden,
			TimeInForce:          StringToTimeInForce(orderDetail.TimeInForce, orderDetail.PostOnly),
			AverageExecutedPrice: orderDetail.Price.Float64(),
			FeeAsset:             currency.NewCode(orderDetail.FeeCurrency),
			ClientOrderID:        orderDetail.ClientOID,
			Status:               oStatus,
			CloseTime:            orderDetail.CreatedAt.Time(),
			MarginType:           mType,
			LastUpdated:          orderDetail.LastUpdatedAt.Time(),
		}, nil
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (e *Exchange) GetDepositAddress(ctx context.Context, c currency.Code, _, chain string) (*deposit.Address, error) {
	ad, err := e.GetDepositAddressesV2(ctx, c.Upper())
	if err != nil {
		return nil, err
	}
	if chain != "" {
		// check if there is a matching chain address.
		for a := range ad {
			if strings.EqualFold(ad[a].Chain, chain) {
				return &deposit.Address{
					Address: ad[a].Address,
					Chain:   ad[a].Chain,
					Tag:     ad[a].Memo,
				}, nil
			}
		}
		return nil, fmt.Errorf("%w matching the chain name %s", errNoDepositAddress, chain)
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
// The endpoint was deprecated for futures, please transfer assets from the FUTURES account to the MAIN account first,
// and then withdraw from the MAIN account
func (e *Exchange) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	withdrawalID, err := e.ApplyWithdrawal(ctx, withdrawRequest.Currency, withdrawRequest.Crypto.Address, withdrawRequest.Crypto.AddressTag, withdrawRequest.Description, withdrawRequest.Crypto.Chain, "INTERNAL", withdrawRequest.InternalTransfer, withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		ID: withdrawalID,
	}, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is submitted
func (e *Exchange) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (e *Exchange) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// OrderTypeToString returns an order type instance from string.
func OrderTypeToString(oType order.Type) (string, error) {
	switch oType {
	case order.Limit:
		return "limit", nil
	case order.Market:
		return "market", nil
	case order.StopLimit:
		return "limit_stop", nil
	case order.StopMarket:
		return "market_stop", nil
	case order.AnyType, order.UnknownType:
		return "", nil
	default:
		return "", order.ErrUnsupportedOrderType
	}
}

// GetActiveOrders retrieves any orders that are active/open
func (e *Exchange) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if getOrdersRequest == nil {
		return nil, common.ErrNilPointer
	}
	err := e.CurrencyPairs.IsAssetEnabled(getOrdersRequest.AssetType)
	if err != nil {
		return nil, err
	}
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	format, err := e.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	getOrdersRequest.Pairs = getOrdersRequest.Pairs.Format(format)
	var pair string
	var orders []order.Detail
	switch getOrdersRequest.AssetType {
	case asset.Futures:
		if len(getOrdersRequest.Pairs) == 1 {
			pair = format.Format(getOrdersRequest.Pairs[0])
		}
		sideString, err := e.OrderSideString(getOrdersRequest.Side)
		if err != nil {
			return nil, err
		}
		oType, err := OrderTypeToString(getOrdersRequest.Type)
		if err != nil {
			return nil, err
		}
		futuresOrders, err := e.GetFuturesOrders(ctx, "active", pair, sideString, oType, getOrdersRequest.StartTime, getOrdersRequest.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range futuresOrders.Items {
			if !futuresOrders.Items[x].IsActive {
				continue
			}
			pair, err := e.MatchSymbolWithAvailablePairs(futuresOrders.Items[x].Symbol, getOrdersRequest.AssetType, false)
			if err != nil {
				return nil, err
			}
			side, err := order.StringToOrderSide(futuresOrders.Items[x].Side)
			if err != nil {
				return nil, err
			}

			switch side {
			case order.Sell:
				side = order.Short
			case order.Buy:
				side = order.Long
			}

			oType, err := order.StringToOrderType(futuresOrders.Items[x].OrderType)
			if err != nil {
				return nil, fmt.Errorf("asset type: %v order type: %v err: %w", getOrdersRequest.AssetType, getOrdersRequest.Type, err)
			}

			status, err := order.StringToOrderStatus(futuresOrders.Items[x].Status)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				OrderID:            futuresOrders.Items[x].ID,
				ClientOrderID:      futuresOrders.Items[x].ClientOid,
				Amount:             futuresOrders.Items[x].Size,
				ContractAmount:     futuresOrders.Items[x].Size,
				RemainingAmount:    futuresOrders.Items[x].Size - futuresOrders.Items[x].FilledSize,
				ExecutedAmount:     futuresOrders.Items[x].FilledSize,
				Exchange:           e.Name,
				Date:               futuresOrders.Items[x].CreatedAt.Time(),
				LastUpdated:        futuresOrders.Items[x].UpdatedAt.Time(),
				Price:              futuresOrders.Items[x].Price,
				Side:               side,
				Type:               oType,
				Pair:               pair,
				TimeInForce:        StringToTimeInForce(futuresOrders.Items[x].TimeInForce, futuresOrders.Items[x].PostOnly),
				ReduceOnly:         futuresOrders.Items[x].ReduceOnly,
				Status:             status,
				SettlementCurrency: currency.NewCode(futuresOrders.Items[x].SettleCurrency),
				Leverage:           futuresOrders.Items[x].Leverage,
				AssetType:          getOrdersRequest.AssetType,
				HiddenOrder:        futuresOrders.Items[x].Hidden,
			})
		}
	case asset.Spot, asset.Margin:
		var singlePair currency.Pair
		if len(getOrdersRequest.Pairs) == 1 {
			singlePair = getOrdersRequest.Pairs[0]
		}
		switch getOrdersRequest.Type {
		case order.OCO:
			response, err := e.GetOCOOrderList(ctx, 500, 1, singlePair.String(), getOrdersRequest.StartTime, getOrdersRequest.EndTime, []string{})
			if err != nil {
				return nil, err
			}
			for a := range response.Items {
				if response.Items[a].Status != "NEW" {
					continue
				}
				cp, err := currency.NewPairFromString(response.Items[a].Symbol)
				if err != nil {
					return nil, err
				}
				if len(getOrdersRequest.Pairs) > 1 && !getOrdersRequest.Pairs.Contains(cp, true) {
					continue
				}
				status, err := order.StringToOrderStatus(response.Items[a].Status)
				if err != nil {
					return nil, err
				}
				orders = append(orders, order.Detail{
					OrderID:       response.Items[a].OrderID,
					ClientOrderID: response.Items[a].ClientOrderID,
					Exchange:      e.Name,
					LastUpdated:   response.Items[a].OrderTime.Time(),
					Type:          order.OCO,
					Pair:          cp,
					Status:        status,
				})
			}
		case order.Stop, order.StopLimit, order.StopMarket, order.ConditionalStop:
			// NOTE: The orderType values 'limit', 'market', 'limit_stop', and 'market_stop' trigger an "The order type is invalid" error.
			// As a result, these options are currently unavailable.
			tradeType := SpotTradeType
			if getOrdersRequest.AssetType == asset.Margin {
				if getOrdersRequest.MarginType == margin.Multi {
					tradeType = CrossMarginTradeType
				} else {
					tradeType = IsolatedMarginTradeType
				}
			}
			response, err := e.ListStopOrders(ctx, singlePair.String(), getOrdersRequest.Side.Lower(), "", tradeType, []string{getOrdersRequest.FromOrderID}, getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0, 0)
			if err != nil {
				return nil, err
			}
			for a := range response.Items {
				if response.Items[a].Status != "New" {
					continue
				}
				pair, err := e.MatchSymbolWithAvailablePairs(response.Items[a].Symbol, getOrdersRequest.AssetType, false)
				if err != nil {
					return nil, err
				}
				if len(getOrdersRequest.Pairs) > 1 && !getOrdersRequest.Pairs.Contains(pair, true) {
					continue
				}
				side, err := order.StringToOrderSide(response.Items[a].Side)
				if err != nil {
					return nil, err
				}
				status, err := order.StringToOrderStatus(response.Items[a].Status)
				if err != nil {
					return nil, err
				}
				orders = append(orders, order.Detail{
					OrderID:        response.Items[a].ID,
					ClientOrderID:  response.Items[a].ClientOID,
					Amount:         response.Items[a].Size,
					ContractAmount: response.Items[a].Size,
					Exchange:       e.Name,
					Date:           response.Items[a].CreatedAt.Time(),
					LastUpdated:    response.Items[a].OrderTime.Time(),
					Price:          response.Items[a].Price,
					Side:           side,
					Type:           order.Stop,
					Pair:           pair,
					TimeInForce:    StringToTimeInForce(response.Items[a].TimeInForce, response.Items[a].PostOnly),
					Status:         status,
					AssetType:      getOrdersRequest.AssetType,
					HiddenOrder:    response.Items[a].Hidden,
				})
			}
		default:
			if len(getOrdersRequest.Pairs) == 1 {
				pair = format.Format(getOrdersRequest.Pairs[0])
			}
			sideString, err := e.OrderSideString(getOrdersRequest.Side)
			if err != nil {
				return nil, err
			}
			oType, err := OrderTypeToString(getOrdersRequest.Type)
			if err != nil {
				return nil, fmt.Errorf("asset type: %v order type: %v err: %w", getOrdersRequest.AssetType, getOrdersRequest.Type, err)
			}
			spotOrders, err := e.ListOrders(ctx, "active", pair, sideString, oType, "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
			if err != nil {
				return nil, err
			}
			for x := range spotOrders.Items {
				if !spotOrders.Items[x].IsActive {
					continue
				}
				pair, err := e.MatchSymbolWithAvailablePairs(spotOrders.Items[x].Symbol, getOrdersRequest.AssetType, true)
				if err != nil {
					return nil, err
				}
				if len(getOrdersRequest.Pairs) > 0 && !getOrdersRequest.Pairs.Contains(pair, true) {
					continue
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
					Amount:          spotOrders.Items[x].Size.Float64(),
					RemainingAmount: spotOrders.Items[x].Size.Float64() - spotOrders.Items[x].DealSize.Float64(),
					ExecutedAmount:  spotOrders.Items[x].DealSize.Float64(),
					Exchange:        e.Name,
					Date:            spotOrders.Items[x].CreatedAt.Time(),
					Price:           spotOrders.Items[x].Price.Float64(),
					Side:            side,
					Type:            oType,
					Pair:            pair,
				})
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, getOrdersRequest.AssetType)
	}
	return orders, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (e *Exchange) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if getOrdersRequest == nil {
		return nil, common.ErrNilPointer
	}
	if err := e.CurrencyPairs.IsAssetEnabled(getOrdersRequest.AssetType); err != nil {
		return nil, err
	}
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}

	sideString, err := e.OrderSideString(getOrdersRequest.Side)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	var orderStatus order.Status
	switch getOrdersRequest.AssetType {
	case asset.Futures:
		var futuresOrders *FutureOrdersResponse
		var newOrders *FutureOrdersResponse
		if len(getOrdersRequest.Pairs) == 0 {
			futuresOrders, err = e.GetFuturesOrders(ctx, "", "", sideString, getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
			if err != nil {
				return nil, err
			}
		} else {
			for x := range getOrdersRequest.Pairs {
				getOrdersRequest.Pairs[x], err = e.FormatExchangeCurrency(getOrdersRequest.Pairs[x], getOrdersRequest.AssetType)
				if err != nil {
					return nil, err
				}
				newOrders, err = e.GetFuturesOrders(ctx, "", getOrdersRequest.Pairs[x].String(), sideString, getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
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
		orders = make(order.FilteredOrders, 0, len(futuresOrders.Items))
		for i := range orders {
			orderSide, err := order.StringToOrderSide(futuresOrders.Items[i].Side)
			if err != nil {
				return nil, err
			}
			pair, err := e.MatchSymbolWithAvailablePairs(futuresOrders.Items[i].Symbol, getOrdersRequest.AssetType, true)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(futuresOrders.Items[i].OrderType)
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
			}
			orders = append(orders, order.Detail{
				Price:           futuresOrders.Items[i].Price,
				Amount:          futuresOrders.Items[i].Size,
				ExecutedAmount:  futuresOrders.Items[i].DealSize,
				RemainingAmount: futuresOrders.Items[i].Size - futuresOrders.Items[i].DealSize,
				Date:            futuresOrders.Items[i].CreatedAt.Time(),
				Exchange:        e.Name,
				OrderID:         futuresOrders.Items[i].ID,
				Side:            orderSide,
				Status:          orderStatus,
				Type:            oType,
				Pair:            pair,
			})
			orders[i].InferCostsAndTimes()
		}
	case asset.Spot, asset.Margin:
		var singlePair currency.Pair
		if len(getOrdersRequest.Pairs) == 1 {
			singlePair = getOrdersRequest.Pairs[0]
		}
		switch getOrdersRequest.Type {
		case order.OCO:
			var response *OCOOrders
			response, err = e.GetOCOOrderList(ctx, 500, 1, singlePair.String(), getOrdersRequest.StartTime, getOrdersRequest.EndTime, []string{})
			if err != nil {
				return nil, err
			}
			var cp currency.Pair
			for a := range response.Items {
				cp, err = currency.NewPairFromString(response.Items[a].Symbol)
				if err != nil {
					return nil, err
				}
				if len(getOrdersRequest.Pairs) > 1 && !getOrdersRequest.Pairs.Contains(cp, true) {
					continue
				}
				var status order.Status
				status, err = order.StringToOrderStatus(response.Items[a].Status)
				if err != nil {
					return nil, err
				}
				orders = append(orders, order.Detail{
					OrderID:       response.Items[a].OrderID,
					ClientOrderID: response.Items[a].ClientOrderID,
					Exchange:      e.Name,
					LastUpdated:   response.Items[a].OrderTime.Time(),
					Type:          order.OCO,
					Pair:          cp,
					Status:        status,
				})
			}
		case order.Stop, order.StopLimit, order.StopMarket, order.ConditionalStop:
			// NOTE: The orderType values 'limit', 'market', 'limit_stop', and 'market_stop' trigger an "The order type is invalid" error.
			// As a result, these options are currently unavailable.
			tradeType := SpotTradeType
			if getOrdersRequest.AssetType == asset.Margin {
				if getOrdersRequest.MarginType == margin.Multi {
					tradeType = CrossMarginTradeType
				} else {
					tradeType = IsolatedMarginTradeType
				}
			}
			var response *StopOrderListResponse
			response, err = e.ListStopOrders(ctx, singlePair.String(), sideString, "", tradeType, []string{getOrdersRequest.FromOrderID}, getOrdersRequest.StartTime, getOrdersRequest.EndTime, 0, 0)
			if err != nil {
				return nil, err
			}
			for a := range response.Items {
				pair, err := e.MatchSymbolWithAvailablePairs(response.Items[a].Symbol, getOrdersRequest.AssetType, false)
				if err != nil {
					return nil, err
				}
				if len(getOrdersRequest.Pairs) > 1 && !getOrdersRequest.Pairs.Contains(pair, true) {
					continue
				}
				var (
					side   order.Side
					status order.Status
				)
				side, err = order.StringToOrderSide(response.Items[a].Side)
				if err != nil {
					return nil, err
				}
				status, err = order.StringToOrderStatus(response.Items[a].Status)
				if err != nil {
					return nil, err
				}
				orders = append(orders, order.Detail{
					OrderID:        response.Items[a].ID,
					ClientOrderID:  response.Items[a].ClientOID,
					Amount:         response.Items[a].Size,
					ContractAmount: response.Items[a].Size,
					TriggerPrice:   response.Items[a].StopPrice,
					Exchange:       e.Name,
					Date:           response.Items[a].CreatedAt.Time(),
					LastUpdated:    response.Items[a].OrderTime.Time(),
					Price:          response.Items[a].Price,
					Side:           side,
					Type:           order.Stop,
					Pair:           pair,
					TimeInForce:    StringToTimeInForce(response.Items[a].TimeInForce, response.Items[a].PostOnly),
					Status:         status,
					AssetType:      getOrdersRequest.AssetType,
					HiddenOrder:    response.Items[a].Hidden,
				})
			}
		default:
			var responseOrders *OrdersListResponse
			var newOrders *OrdersListResponse
			if len(getOrdersRequest.Pairs) == 0 {
				responseOrders, err = e.ListOrders(ctx, "", "", sideString, getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
				if err != nil {
					return nil, err
				}
			} else {
				for x := range getOrdersRequest.Pairs {
					newOrders, err = e.ListOrders(ctx, "", getOrdersRequest.Pairs[x].String(), sideString, getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
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
				orderSide, err := order.StringToOrderSide(responseOrders.Items[i].Side)
				if err != nil {
					return nil, err
				}
				pair, err := currency.NewPairFromString(responseOrders.Items[i].Symbol)
				if err != nil {
					return nil, err
				}
				oType, err := order.StringToOrderType(responseOrders.Items[i].Type)
				if err != nil {
					log.Errorf(log.ExchangeSys, "%s %v", e.Name, err)
				}
				orders[i] = order.Detail{
					Price:           responseOrders.Items[i].Price.Float64(),
					Amount:          responseOrders.Items[i].Size.Float64(),
					ExecutedAmount:  responseOrders.Items[i].DealSize.Float64(),
					RemainingAmount: responseOrders.Items[i].Size.Float64() - responseOrders.Items[i].DealSize.Float64(),
					Date:            responseOrders.Items[i].CreatedAt.Time(),
					Exchange:        e.Name,
					OrderID:         responseOrders.Items[i].ID,
					Side:            orderSide,
					Status:          orderStatus,
					Type:            oType,
					Pair:            pair,
				}
				orders[i].InferCostsAndTimes()
			}
		}
	}
	return getOrdersRequest.Filter(e.Name, orders), nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (e *Exchange) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if !e.AreCredentialsValid(ctx) &&
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	if feeBuilder.Pair.IsEmpty() {
		return 0, currency.ErrCurrencyPairEmpty
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyWithdrawalFee,
		exchange.CryptocurrencyTradeFee:
		fee, err := e.GetTradingFee(ctx, currency.Pairs{feeBuilder.Pair})
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			return feeBuilder.Amount * fee[0].MakerFeeRate, nil
		}
		return feeBuilder.Amount * fee[0].TakerFeeRate, nil
	case exchange.OfflineTradeFee:
		return feeBuilder.Amount * 0.001, nil
	case exchange.CryptocurrencyDepositFee:
		return 0, nil
	default:
		if !feeBuilder.FiatCurrency.IsEmpty() {
			fee, err := e.GetBasicFee(ctx, "1")
			if err != nil {
				return 0, err
			}
			if feeBuilder.IsMaker {
				return feeBuilder.Amount * fee.MakerFeeRate, nil
			}
			return feeBuilder.Amount * fee.TakerFeeRate, nil
		}
		return 0, errors.New("can't construct fee")
	}
}

// ValidateCredentials validates current credentials used for wrapper
func (e *Exchange) ValidateCredentials(ctx context.Context, assetType asset.Item) error {
	err := e.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return err
	}
	_, err = e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	var timeseries []kline.Candle
	switch a {
	case asset.Futures:
		var candles []FuturesKline
		candles, err := e.GetFuturesKline(ctx, int64(interval.Duration().Minutes()), req.RequestFormatted.String(), req.Start, req.End)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			timeseries = append(
				timeseries, kline.Candle{
					Time:   candles[x].StartTime.Time(),
					Open:   candles[x].Open,
					High:   candles[x].High,
					Low:    candles[x].Low,
					Close:  candles[x].Close,
					Volume: candles[x].Volume,
				})
		}
	case asset.Spot, asset.Margin:
		intervalString, err := IntervalToString(interval)
		if err != nil {
			return nil, err
		}
		var candles []Kline
		candles, err = e.GetKlines(ctx, req.RequestFormatted.String(), intervalString, req.Start, req.End)
		if err != nil {
			return nil, err
		}
		for x := range candles {
			timeseries = append(
				timeseries, kline.Candle{
					Time:   candles[x].StartTime.Time(),
					Open:   candles[x].Open.Float64(),
					High:   candles[x].High.Float64(),
					Low:    candles[x].Low.Float64(),
					Close:  candles[x].Close.Float64(),
					Volume: candles[x].Volume.Float64(),
				})
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(timeseries)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (e *Exchange) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := e.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}

	var timeSeries []kline.Candle
	switch a {
	case asset.Futures:
		for x := range req.RangeHolder.Ranges {
			var candles []FuturesKline
			candles, err = e.GetFuturesKline(ctx, int64(interval.Duration().Minutes()), req.RequestFormatted.String(), req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for y := range candles {
				timeSeries = append(
					timeSeries, kline.Candle{
						Time:   candles[y].StartTime.Time(),
						Open:   candles[y].Open,
						High:   candles[y].High,
						Low:    candles[y].Low,
						Close:  candles[y].Close,
						Volume: candles[y].Volume,
					})
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.Spot, asset.Margin:
		var intervalString string
		intervalString, err = IntervalToString(interval)
		if err != nil {
			return nil, err
		}
		var candles []Kline
		for x := range req.RangeHolder.Ranges {
			candles, err = e.GetKlines(ctx, req.RequestFormatted.String(), intervalString, req.RangeHolder.Ranges[x].Start.Time, req.RangeHolder.Ranges[x].End.Time)
			if err != nil {
				return nil, err
			}
			for x := range candles {
				timeSeries = append(
					timeSeries, kline.Candle{
						Time:   candles[x].StartTime.Time(),
						Open:   candles[x].Open.Float64(),
						High:   candles[x].High.Float64(),
						Low:    candles[x].Low.Float64(),
						Close:  candles[x].Close.Float64(),
						Volume: candles[x].Volume.Float64(),
					})
			}
		}
		return req.ProcessResponse(timeSeries)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, a)
	}
}

// GetServerTime returns the current exchange server time.
func (e *Exchange) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	switch a {
	case asset.Spot, asset.Margin:
		return e.GetCurrentServerTime(ctx)
	case asset.Futures:
		return e.GetFuturesServerTime(ctx)
	default:
		return time.Time{}, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific cryptocurrency
func (e *Exchange) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencyDetail, err := e.GetCurrencyDetailV3(ctx, cryptocurrency, "")
	if err != nil {
		return nil, err
	}
	chains := make([]string, len(currencyDetail.Chains))
	for x := range currencyDetail.Chains {
		chains[x] = currencyDetail.Chains[x].ChainName
	}
	return chains, nil
}

// ValidateAPICredentials validates current credentials used for wrapper functionality
func (e *Exchange) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := e.UpdateAccountBalances(ctx, assetType)
	return e.CheckTransientError(err)
}

// GetFuturesContractDetails returns details about futures contracts
func (e *Exchange) GetFuturesContractDetails(ctx context.Context, item asset.Item) ([]futures.Contract, error) {
	if !item.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !e.SupportsAsset(item) || item != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, item)
	}

	contracts, err := e.GetFuturesOpenContracts(ctx)
	if err != nil {
		return nil, err
	}

	resp := make([]futures.Contract, len(contracts))
	for i := range contracts {
		ct := futures.Quarterly
		if contracts[i].ContractType == "FFWCSX" {
			ct = futures.Perpetual
		}
		contractSettlementType := futures.Linear
		if contracts[i].IsInverse {
			contractSettlementType = futures.Inverse
		}
		var fri time.Duration
		if len(e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
			// can infer funding rate interval from the only funding rate frequency defined
			for k := range e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
				fri = k.Duration()
			}
		}
		timeOfCurrentFundingRate := time.Now().Add((time.Duration(contracts[i].NextFundingRateTime) * time.Millisecond) - fri).Truncate(time.Hour).UTC()
		resp[i] = futures.Contract{
			Exchange: e.Name,
			Name: currency.NewPair(contracts[i].BaseCurrency,
				currency.NewCode(contracts[i].Symbol[len(contracts[i].BaseCurrency.String()):])),
			Underlying:         currency.NewPair(contracts[i].BaseCurrency, contracts[i].QuoteCurrency),
			SettlementCurrency: contracts[i].SettleCurrency,
			MarginCurrency:     contracts[i].SettleCurrency,
			Asset:              item,
			StartDate:          contracts[i].FirstOpenDate.Time(),
			EndDate:            contracts[i].ExpireDate.Time(),
			IsActive:           !strings.EqualFold(contracts[i].Status, "closed"),
			Status:             contracts[i].Status,
			Multiplier:         contracts[i].Multiplier,
			MaxLeverage:        contracts[i].MaxLeverage,
			SettlementType:     contractSettlementType,
			LatestRate: fundingrate.Rate{
				Rate: decimal.NewFromFloat(contracts[i].FundingFeeRate),
				Time: timeOfCurrentFundingRate, // kucoin pays every 8 hours
			},
			Type: ct,
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (e *Exchange) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	var fri time.Duration
	if len(e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
		// can infer funding rate interval from the only funding rate frequency defined
		for k := range e.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
			fri = k.Duration()
		}
	}
	if r.Pair.IsEmpty() {
		contracts, err := e.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		if r.IncludePredictedRate {
			log.Warnf(log.ExchangeSys, "%s predicted rate for all currencies requires an additional %v requests", e.Name, len(contracts))
		}
		timeChecked := time.Now()
		resp := make([]fundingrate.LatestRateResponse, 0, len(contracts))
		for i := range contracts {
			timeOfNextFundingRate := time.Now().Add(time.Duration(contracts[i].NextFundingRateTime) * time.Millisecond).Truncate(time.Hour).UTC()
			cp := currency.NewPair(contracts[i].BaseCurrency,
				currency.NewCode(contracts[i].Symbol[len(contracts[i].BaseCurrency.String()):]))
			var isPerp bool
			isPerp, err = e.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}

			rate := fundingrate.LatestRateResponse{
				Exchange: e.Name,
				Asset:    r.Asset,
				Pair:     cp,
				LatestRate: fundingrate.Rate{
					Time: timeOfNextFundingRate.Add(-fri),
					Rate: decimal.NewFromFloat(contracts[i].FundingFeeRate),
				},
				TimeOfNextRate: timeOfNextFundingRate,
				TimeChecked:    timeChecked,
			}
			if r.IncludePredictedRate {
				var fr *FuturesFundingRate
				fr, err = e.GetFuturesCurrentFundingRate(ctx, contracts[i].Symbol)
				if err != nil {
					return nil, err
				}
				rate.PredictedUpcomingRate = fundingrate.Rate{
					Time: timeOfNextFundingRate,
					Rate: decimal.NewFromFloat(fr.PredictedValue),
				}
			}
			resp = append(resp, rate)
		}
		return resp, nil
	}
	resp := make([]fundingrate.LatestRateResponse, 1)
	is, err := e.IsPerpetualFutureCurrency(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	if !is {
		return nil, fmt.Errorf("%w %s %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	var fr *FuturesFundingRate
	fr, err = e.GetFuturesCurrentFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	rate := fundingrate.LatestRateResponse{
		Exchange: e.Name,
		Asset:    r.Asset,
		Pair:     r.Pair,
		LatestRate: fundingrate.Rate{
			Time: fr.TimePoint.Time(),
			Rate: decimal.NewFromFloat(fr.Value),
		},
		TimeOfNextRate: fr.TimePoint.Time().Add(fri).Truncate(time.Hour).UTC(),
		TimeChecked:    time.Now(),
	}
	if r.IncludePredictedRate {
		rate.PredictedUpcomingRate = fundingrate.Rate{
			Time: rate.TimeOfNextRate,
			Rate: decimal.NewFromFloat(fr.PredictedValue),
		}
	}
	resp[0] = rate
	return resp, nil
}

// IsPerpetualFutureCurrency ensures a given asset and currency is a perpetual future
func (e *Exchange) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	return a == asset.Futures && (cp.Quote.Equal(currency.USDTM) || cp.Quote.Equal(currency.USDM)), nil
}

// GetHistoricalFundingRates returns funding rates for a given asset and currency for a time period
func (e *Exchange) GetHistoricalFundingRates(ctx context.Context, r *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}

	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	if !r.StartDate.IsZero() && !r.EndDate.IsZero() {
		err := common.StartEndTimeCheck(r.StartDate, r.EndDate)
		if err != nil {
			return nil, err
		}
	}
	var err error
	r.Pair, err = e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}

	records, err := e.GetPublicFundingRate(ctx, r.Pair.String(), r.StartDate, r.EndDate)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}
	fundingRates := make([]fundingrate.Rate, 0, len(records))
	for i := range records {
		if (!r.EndDate.IsZero() && r.EndDate.Before(records[i].Timepoint.Time())) ||
			(!r.StartDate.IsZero() && r.StartDate.After(records[i].Timepoint.Time())) {
			continue
		}

		fundingRates = append(fundingRates, fundingrate.Rate{
			Rate: decimal.NewFromFloat(records[i].FundingRate),
			Time: records[i].Timepoint.Time(),
		})
	}

	if len(fundingRates) == 0 {
		return nil, fundingrate.ErrNoFundingRatesFound
	}

	return &fundingrate.HistoricalRates{
		Exchange:        e.Name,
		Asset:           r.Asset,
		Pair:            r.Pair,
		FundingRates:    fundingRates,
		StartDate:       fundingRates[len(fundingRates)-1].Time,
		EndDate:         fundingRates[0].Time,
		LatestRate:      fundingRates[0],
		PaymentCurrency: r.PaymentCurrency,
	}, nil
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (e *Exchange) GetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ order.Side) (float64, error) {
	return -1, fmt.Errorf("%w leverage is set during order placement, view orders to view leverage", common.ErrFunctionNotSupported)
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (e *Exchange) SetLeverage(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type, _ float64, _ order.Side) error {
	return fmt.Errorf("%w leverage is set during order placement", common.ErrFunctionNotSupported)
}

// SetMarginType sets the default margin type for when opening a new position
func (e *Exchange) SetMarginType(_ context.Context, _ asset.Item, _ currency.Pair, _ margin.Type) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// SetCollateralMode sets the collateral type for your account
func (e *Exchange) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return fmt.Errorf("%w must be set via website", common.ErrFunctionNotSupported)
}

// GetCollateralMode returns the collateral type for your account
func (e *Exchange) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return collateral.UnknownMode, fmt.Errorf("%w only via website", common.ErrFunctionNotSupported)
}

// ChangePositionMargin will modify a position/currencies margin parameters
func (e *Exchange) ChangePositionMargin(ctx context.Context, r *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotFuturesAsset, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if r.MarginType != margin.Isolated {
		return nil, fmt.Errorf("%w %v", margin.ErrMarginTypeUnsupported, r.MarginType)
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}

	resp, err := e.AddMargin(ctx, fPair.String(), fmt.Sprintf("%s%v%v", r.Pair, r.NewAllocatedMargin, time.Now().Unix()), r.NewAllocatedMargin)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%s - %s", e.Name, "no response received")
	}
	return &margin.PositionChangeResponse{
		Exchange:        e.Name,
		Pair:            r.Pair,
		Asset:           r.Asset,
		AllocatedMargin: resp.PosMargin,
		MarginType:      r.MarginType,
	}, nil
}

// GetFuturesPositionSummary returns position summary details for an active position
func (e *Exchange) GetFuturesPositionSummary(ctx context.Context, r *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if r.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	fPair, err := e.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	pos, err := e.GetFuturesPosition(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	marginType := margin.Isolated
	if pos.CrossMode {
		marginType = margin.Multi
	}
	contracts, err := e.GetFuturesContractDetails(ctx, r.Asset)
	if err != nil {
		return nil, err
	}
	var multiplier, contractSize float64
	var settlementType futures.ContractSettlementType
	for i := range contracts {
		if !contracts[i].Name.Equal(fPair) {
			continue
		}
		multiplier = contracts[i].Multiplier
		contractSize = multiplier * pos.CurrentQty
		settlementType = contracts[i].SettlementType
	}

	ao, err := e.GetFuturesAccountOverview(ctx, fPair.Base.String())
	if err != nil {
		return nil, err
	}

	return &futures.PositionSummary{
		Pair:                         r.Pair,
		Asset:                        r.Asset,
		MarginType:                   marginType,
		CollateralMode:               collateral.MultiMode,
		Currency:                     currency.NewCode(pos.SettleCurrency),
		StartDate:                    pos.OpeningTimestamp.Time(),
		AvailableEquity:              decimal.NewFromFloat(ao.AccountEquity),
		MarginBalance:                decimal.NewFromFloat(ao.MarginBalance),
		NotionalSize:                 decimal.NewFromFloat(pos.MarkValue),
		Leverage:                     decimal.NewFromFloat(pos.RealLeverage),
		MaintenanceMarginRequirement: decimal.NewFromFloat(pos.MaintMarginReq),
		InitialMarginRequirement:     decimal.NewFromFloat(pos.PosInit),
		EstimatedLiquidationPrice:    decimal.NewFromFloat(pos.LiquidationPrice),
		CollateralUsed:               decimal.NewFromFloat(pos.PosCost),
		MarkPrice:                    decimal.NewFromFloat(pos.MarkPrice),
		CurrentSize:                  decimal.NewFromFloat(pos.CurrentQty),
		ContractSize:                 decimal.NewFromFloat(contractSize),
		ContractMultiplier:           decimal.NewFromFloat(multiplier),
		ContractSettlementType:       settlementType,
		AverageOpenPrice:             decimal.NewFromFloat(pos.AvgEntryPrice),
		UnrealisedPNL:                decimal.NewFromFloat(pos.UnrealisedPnl),
		RealisedPNL:                  decimal.NewFromFloat(pos.RealisedPnl),
		MaintenanceMarginFraction:    decimal.NewFromFloat(pos.MaintMarginReq),
		FreeCollateral:               decimal.NewFromFloat(ao.AvailableBalance),
		TotalCollateral:              decimal.NewFromFloat(ao.AccountEquity),
		FrozenBalance:                decimal.NewFromFloat(ao.FrozenFunds),
	}, nil
}

// GetFuturesPositionOrders returns the orders for futures positions
func (e *Exchange) GetFuturesPositionOrders(ctx context.Context, r *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w HistoricalRatesRequest", common.ErrNilPointer)
	}
	if r.Asset != asset.Futures {
		return nil, fmt.Errorf("%w %v", futures.ErrNotPerpetualFuture, r.Asset)
	}
	if len(r.Pairs) == 0 {
		return nil, currency.ErrCurrencyPairEmpty
	}
	err := common.StartEndTimeCheck(r.StartDate, r.EndDate)
	if err != nil {
		return nil, err
	}
	if !r.EndDate.IsZero() && r.EndDate.Sub(r.StartDate) > e.Features.Supports.MaximumOrderHistory {
		if r.RespectOrderHistoryLimits {
			r.StartDate = time.Now().Add(-e.Features.Supports.MaximumOrderHistory)
		} else {
			return nil, fmt.Errorf("%w max lookup %v", futures.ErrOrderHistoryTooLarge, time.Now().Add(-e.Features.Supports.MaximumOrderHistory))
		}
	}
	contracts, err := e.GetFuturesContractDetails(ctx, r.Asset)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.PositionResponse, len(r.Pairs))
	for x := range r.Pairs {
		var multiplier float64
		fPair, err := e.FormatExchangeCurrency(r.Pairs[x], r.Asset)
		if err != nil {
			return nil, err
		}
		for i := range contracts {
			if !contracts[i].Name.Equal(fPair) {
				continue
			}
			multiplier = contracts[i].Multiplier
		}

		positionOrders, err := e.GetFuturesOrders(ctx, "", fPair.String(), "", "", r.StartDate, r.EndDate)
		if err != nil {
			return nil, err
		}
		resp[x].Orders = make([]order.Detail, len(positionOrders.Items))
		for y := range positionOrders.Items {
			side, err := order.StringToOrderSide(positionOrders.Items[y].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(positionOrders.Items[y].OrderType)
			if err != nil {
				return nil, fmt.Errorf("asset type: %v err: %w", r.Asset, err)
			}
			oStatus, err := order.StringToOrderStatus(positionOrders.Items[y].Status)
			if err != nil {
				return nil, fmt.Errorf("asset type: %v err: %w", r.Asset, err)
			}
			resp[x].Orders[y] = order.Detail{
				Leverage:        positionOrders.Items[y].Leverage,
				Price:           positionOrders.Items[y].Price,
				Amount:          positionOrders.Items[y].Size * multiplier,
				ContractAmount:  positionOrders.Items[y].Size,
				ExecutedAmount:  positionOrders.Items[y].FilledSize,
				RemainingAmount: positionOrders.Items[y].Size - positionOrders.Items[y].FilledSize,
				CostAsset:       currency.NewCode(positionOrders.Items[y].SettleCurrency),
				Exchange:        e.Name,
				OrderID:         positionOrders.Items[y].ID,
				ClientOrderID:   positionOrders.Items[y].ClientOid,
				Type:            oType,
				Side:            side,
				Status:          oStatus,
				AssetType:       asset.Futures,
				Date:            positionOrders.Items[y].CreatedAt.Time(),
				CloseTime:       positionOrders.Items[y].EndAt.Time(),
				LastUpdated:     positionOrders.Items[y].UpdatedAt.Time(),
				Pair:            fPair,
			}
		}
	}
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (e *Exchange) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !e.SupportsAsset(a) {
		return fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}

	var l []limits.MinMaxLevel
	switch a {
	case asset.Spot, asset.Margin:
		symbols, err := e.GetSymbols(ctx, "")
		if err != nil {
			return err
		}
		l = make([]limits.MinMaxLevel, 0, len(symbols))
		for x := range symbols {
			if a == asset.Margin && !symbols[x].IsMarginEnabled {
				continue
			}
			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, symbols[x].Symbol),
				AmountStepIncrementSize: symbols[x].BaseIncrement.Float64(),
				QuoteStepIncrementSize:  symbols[x].QuoteIncrement.Float64(),
				PriceStepIncrementSize:  symbols[x].PriceIncrement.Float64(),
				MinimumBaseAmount:       symbols[x].BaseMinSize.Float64(),
				MaximumBaseAmount:       symbols[x].BaseMaxSize.Float64(),
				MinimumQuoteAmount:      symbols[x].QuoteMinSize.Float64(),
				MaximumQuoteAmount:      symbols[x].QuoteMaxSize.Float64(),
				Listed:                  symbols[x].TradingStartTime.Time(),
			})
		}
	case asset.Futures:
		contract, err := e.GetFuturesOpenContracts(ctx)
		if err != nil {
			return err
		}

		l = make([]limits.MinMaxLevel, 0, len(contract))
		for x := range contract {
			pair, err := e.MatchSymbolWithAvailablePairs(contract[x].Symbol, a, false)
			if err != nil {
				return err
			}

			priceDivisor := 1.0
			if contract[x].Symbol[:2] == "10" { // handle 1000SHIBUSDT, 1000PEPEUSDT etc; exclude 1INCHUSDT
				for _, r := range contract[x].Symbol[1:] {
					if r != '0' {
						break
					}
					priceDivisor *= 10
				}
			}

			l = append(l, limits.MinMaxLevel{
				Key:                     key.NewExchangeAssetPair(e.Name, a, pair),
				AmountStepIncrementSize: contract[x].LotSize,
				QuoteStepIncrementSize:  contract[x].TickSize,
				MinimumBaseAmount:       contract[x].LotSize,
				MaximumBaseAmount:       contract[x].MaxOrderQty,
				MaximumQuoteAmount:      contract[x].MaxPrice,
				MultiplierDecimal:       contract[x].Multiplier,
				Listed:                  contract[x].FirstOpenDate.Time(),
				Delisted:                contract[x].ExpireDate.Time(),
				Expiry:                  contract[x].SettleDate.Time(),
				PriceDivisor:            priceDivisor,
			})
		}
	}

	return limits.Load(l)
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (e *Exchange) GetOpenInterest(ctx context.Context, k ...key.PairAsset) ([]futures.OpenInterest, error) {
	for i := range k {
		if k[i].Asset != asset.Futures {
			// avoid API calls or returning errors after a successful retrieval
			return nil, fmt.Errorf("%w %v %v", asset.ErrNotSupported, k[i].Asset, k[i].Pair())
		}
	}
	contracts, err := e.GetFuturesOpenContracts(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]futures.OpenInterest, 0, len(contracts))
	for i := range contracts {
		pair, err := e.MatchSymbolWithAvailablePairs(contracts[i].Symbol, asset.Futures, true)
		if err != nil && !errors.Is(err, currency.ErrPairNotFound) {
			return nil, err
		}
		var appendData bool
		for j := range k {
			if k[j].Pair().Equal(pair) {
				appendData = true
				break
			}
		}
		if len(k) > 0 && !appendData {
			continue
		}
		resp = append(resp, futures.OpenInterest{
			Key:          key.NewExchangeAssetPair(e.Name, asset.Futures, pair),
			OpenInterest: contracts[i].OpenInterest.Float64(),
		})
	}
	return resp, nil
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (e *Exchange) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := e.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	cp.Delimiter = currency.DashDelimiter
	switch a {
	case asset.Spot:
		return tradeBaseURL + tradeSpot + cp.Upper().String(), nil
	case asset.Margin:
		return tradeBaseURL + tradeSpot + tradeMargin + cp.Upper().String(), nil
	case asset.Futures:
		cp.Delimiter = ""
		return tradeBaseURL + tradeFutures + tradeSpot + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}
}

// StringToTimeInForce returns an order.TimeInForce instance from string
func StringToTimeInForce(tif string, postOnly bool) order.TimeInForce {
	tif = strings.ToUpper(tif)
	var out order.TimeInForce
	switch tif {
	case "GTT":
		out = order.GoodTillTime
	case "IOC":
		out = order.ImmediateOrCancel
	case "FOK":
		out = order.FillOrKill
	default:
		out = order.GoodTillCancel
	}
	if postOnly {
		out |= order.PostOnly
	}
	return out
}
