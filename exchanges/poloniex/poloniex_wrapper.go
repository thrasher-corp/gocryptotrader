package poloniex

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
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

// SetDefaults sets default settings for poloniex
func (p *Poloniex) SetDefaults() {
	p.Name = "Poloniex"
	p.Enabled = true
	p.Verbose = true
	p.API.CredentialsValidator.RequiresKey = true
	p.API.CredentialsValidator.RequiresSecret = true

	err := p.StoreAssetPairFormat(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = p.StoreAssetPairFormat(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter},
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	p.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerBatching:        true,
				TickerFetching:        true,
				KlineFetching:         true,
				TradeFetching:         true,
				OrderbookFetching:     true,
				AutoPairUpdates:       true,
				AccountInfo:           true,
				GetOrder:              true,
				GetOrders:             true,
				CancelOrder:           true,
				CancelOrders:          true,
				SubmitOrder:           true,
				DepositHistory:        true,
				WithdrawalHistory:     true,
				UserTradeHistory:      true,
				CryptoDeposit:         true,
				CryptoWithdrawal:      true,
				TradeFee:              true,
				CryptoWithdrawalFee:   true,
				MultiChainDeposits:    true,
				MultiChainWithdrawals: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:         true,
				TradeFetching:          true,
				OrderbookFetching:      true,
				Subscribe:              true,
				Unsubscribe:            true,
				AuthenticatedEndpoints: true,
				GetOrders:              true,
				GetOrder:               true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCryptoWithAPIPermission |
				exchange.NoFiatWithdrawals,
			Kline: kline.ExchangeCapabilitiesSupported{
				Intervals: true,
			},
		},
		Enabled: exchange.FeaturesEnabled{
			AutoPairUpdates: true,
			Kline: kline.ExchangeCapabilitiesEnabled{
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.TenMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.TwoHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 500,
			},
		},
	}

	p.Requester, err = request.New(p.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimit()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	p.API.Endpoints = p.NewEndpoints()
	err = p.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      poloniexAPIURL,
		exchange.WebsocketSpot: poloniexWebsocketAddress,
		exchange.RestFutures:   poloniexFuturesAPIURL,
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	p.Websocket = stream.NewWebsocket()
	p.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	p.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	p.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup sets user exchange configuration settings
func (p *Poloniex) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		p.SetEnabled(false)
		return nil
	}
	err = p.SetupDefaults(exch)
	if err != nil {
		return err
	}

	wsRunningURL, err := p.API.Endpoints.GetURL(exchange.WebsocketSpot)
	if err != nil {
		return err
	}

	err = p.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig:        exch,
		DefaultURL:            poloniexWebsocketAddress,
		RunningURL:            wsRunningURL,
		Connector:             p.WsConnect,
		Subscriber:            p.Subscribe,
		Unsubscriber:          p.Unsubscribe,
		GenerateSubscriptions: p.GenerateDefaultSubscriptions,
		Features:              &p.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	if err != nil {
		return err
	}

	err = p.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  poloniexWebsocketAddress,
		RateLimit:            request.NewWeightedRateLimitByDuration(500 * time.Millisecond),
	})
	if err != nil {
		return err
	}
	return p.Websocket.SetupNewConnection(&stream.ConnectionSetup{
		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		URL:                  poloniexPrivateWebsocketAddress,
		RateLimit:            request.NewWeightedRateLimitByDuration(500 * time.Millisecond),
		Authenticated:        true,
	})
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (p *Poloniex) FetchTradablePairs(ctx context.Context, assetType asset.Item) (currency.Pairs, error) {
	switch assetType {
	case asset.Spot, asset.Margin:
		resp, err := p.GetSymbolInformation(ctx, currency.EMPTYPAIR)
		if err != nil {
			return nil, err
		}

		pairs := make([]currency.Pair, 0, len(resp))
		for x := range resp {
			if strings.EqualFold(resp[x].State, "PAUSE") {
				continue
			}
			var pair currency.Pair
			pair, err = currency.NewPairFromString(resp[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
		return pairs, nil
	case asset.Futures:
		instruments, err := p.GetOpenContractList(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, 0, len(instruments.Data))
		var cp currency.Pair
		for i := range instruments.Data {
			if !strings.EqualFold(instruments.Data[i].Status, "Open") {
				continue
			}
			cp, err = currency.NewPairFromString(instruments.Data[i].Symbol)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, cp)
		}
		return pairs, nil
	}
	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (p *Poloniex) UpdateTradablePairs(ctx context.Context, forceUpgrade bool) error {
	enabledAssets := p.GetAssetTypes(true)
	for _, assetType := range enabledAssets {
		pairs, err := p.FetchTradablePairs(ctx, assetType)
		if err != nil {
			return err
		}
		err = p.UpdatePairs(pairs, assetType, false, forceUpgrade)
		if err != nil {
			return err
		}
	}
	return p.EnsureOnePairEnabled()
}

// UpdateTickers updates the ticker for all currency pairs of a given asset type
func (p *Poloniex) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	enabledPairs, err := p.GetEnabledPairs(assetType)
	if err != nil {
		return err
	}
	switch assetType {
	case asset.Spot:
		ticks, err := p.GetTickers(ctx)
		if err != nil {
			return err
		}
		for i := range ticks {
			pair, err := currency.NewPairFromString(ticks[i].Symbol)
			if err != nil {
				return err
			}
			if !enabledPairs.Contains(pair, true) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				AssetType:    assetType,
				Pair:         pair,
				ExchangeName: p.Name,
				Last:         ticks[i].MarkPrice.Float64(),
				Low:          ticks[i].Low.Float64(),
				Ask:          ticks[i].Ask.Float64(),
				Bid:          ticks[i].Bid.Float64(),
				High:         ticks[i].High.Float64(),
				QuoteVolume:  ticks[i].Amount.Float64(),
				Volume:       ticks[i].Quantity.Float64(),
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		ticks, err := p.GetFuturesRealTimeTickersOfSymbols(context.Background())
		if err != nil {
			return err
		}
		for i := range ticks.Data {
			pair, err := currency.NewPairFromString(ticks.Data[i].Symbol)
			if err != nil {
				return err
			}
			if !enabledPairs.Contains(pair, true) {
				continue
			}
			err = ticker.ProcessTicker(&ticker.Price{
				AssetType:    assetType,
				Pair:         pair,
				ExchangeName: p.Name,
				LastUpdated:  ticks.Data[i].Timestamp.Time(),
				Volume:       ticks.Data[i].Size,
				BidSize:      ticks.Data[i].BestBidSize,
				Bid:          ticks.Data[i].BestBidPrice.Float64(),
				AskSize:      ticks.Data[i].BestAskSize,
				Ask:          ticks.Data[i].BestAskPrice.Float64(),
			})
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (p *Poloniex) UpdateTicker(ctx context.Context, currencyPair currency.Pair, a asset.Item) (*ticker.Price, error) {
	if err := p.UpdateTickers(ctx, a); err != nil {
		return nil, err
	}
	return ticker.GetTicker(p.Name, currencyPair, a)
}

// FetchTicker returns the ticker for a currency pair
func (p *Poloniex) FetchTicker(ctx context.Context, currencyPair currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateTicker(ctx, currencyPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (p *Poloniex) FetchOrderbook(ctx context.Context, currencyPair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(p.Name, currencyPair, assetType)
	if err != nil {
		return p.UpdateOrderbook(ctx, currencyPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (p *Poloniex) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	err := p.CurrencyPairs.IsAssetEnabled(assetType)
	if err != nil {
		return nil, err
	}
	pair, err = p.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}
	book := &orderbook.Base{
		Exchange:        p.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: p.CanVerifyOrderbook,
	}
	switch assetType {
	case asset.Spot:
		var orderbookNew *OrderbookData
		orderbookNew, err = p.GetOrderbook(ctx, pair, 0, 0)
		if err != nil {
			return nil, err
		}
		book.Bids = make(orderbook.Tranches, len(orderbookNew.Bids)/2)
		for y := range book.Bids {
			book.Bids[y].Price = orderbookNew.Bids[y*2].Float64()
			book.Bids[y].Amount = orderbookNew.Bids[y*2+1].Float64()
		}
		book.Asks = make(orderbook.Tranches, len(orderbookNew.Asks)/2)
		for y := range book.Asks {
			book.Asks[y].Price = orderbookNew.Asks[y*2].Float64()
			book.Asks[y].Amount = orderbookNew.Asks[y*2+1].Float64()
		}
	case asset.Futures:
		var orderbookNew *Orderbook
		orderbookNew, err = p.GetFullOrderbookLevel2(ctx, pair.String())
		if err != nil {
			return nil, err
		}
		book.Bids = make(orderbook.Tranches, len(orderbookNew.Data.Bids))
		for y := range book.Bids {
			book.Bids[y].Price = orderbookNew.Data.Bids[y][0].Float64()
			book.Bids[y].Amount = orderbookNew.Data.Bids[y][1].Float64()
		}
		book.Asks = make(orderbook.Tranches, len(orderbookNew.Data.Asks))
		for y := range book.Asks {
			book.Asks[y].Price = orderbookNew.Data.Asks[y][0].Float64()
			book.Asks[y].Amount = orderbookNew.Data.Asks[y][1].Float64()
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	err = book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(p.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies for the
// Poloniex exchange
func (p *Poloniex) UpdateAccountInfo(ctx context.Context, _ asset.Item) (account.Holdings, error) {
	var response account.Holdings
	accountBalance, err := p.GetSubAccountBalances(ctx)
	if err != nil {
		return response, err
	}

	subAccounts := make([]account.SubAccount, len(accountBalance))
	for i := range accountBalance {
		subAccount := account.SubAccount{
			ID:        accountBalance[i].AccountID,
			AssetType: stringToAccountType(accountBalance[i].AccountType),
		}
		currencyBalances := make([]account.Balance, len(accountBalance[i].Balances))
		for x := range accountBalance[i].Balances {
			currencyBalances[x] = account.Balance{
				Currency:               currency.NewCode(accountBalance[i].Balances[x].Currency),
				Total:                  accountBalance[i].Balances[x].AvailableBalance.Float64(),
				Hold:                   accountBalance[i].Balances[x].Hold.Float64(),
				Free:                   accountBalance[i].Balances[x].Available.Float64(),
				AvailableWithoutBorrow: accountBalance[i].Balances[x].AvailableBalance.Float64(),
			}
		}
		subAccounts[i] = subAccount
	}
	response = account.Holdings{
		Exchange: p.Name,
		Accounts: subAccounts,
	}
	creds, err := p.GetCredentials(ctx)
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
func (p *Poloniex) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := p.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(p.Name, creds, assetType)
	if err != nil {
		return p.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetAccountFundingHistory returns funding history, deposits and
// withdrawals
func (p *Poloniex) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	end := time.Now()
	walletActivity, err := p.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.FundingHistory, len(walletActivity.Deposits))
	for i := range walletActivity.Deposits {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    p.Name,
			Status:          walletActivity.Deposits[i].Status,
			Timestamp:       walletActivity.Deposits[i].Timestamp.Time(),
			Currency:        walletActivity.Deposits[i].Currency,
			Amount:          walletActivity.Deposits[i].Amount.Float64(),
			CryptoToAddress: walletActivity.Deposits[i].Address,
			CryptoTxID:      walletActivity.Deposits[i].TransactionID,
			TransferType:    "deposit",
		}
	}
	for i := range walletActivity.Withdrawals {
		resp[i] = exchange.FundingHistory{
			ExchangeName:    p.Name,
			Status:          walletActivity.Withdrawals[i].Status,
			Timestamp:       walletActivity.Withdrawals[i].Timestamp.Time(),
			Currency:        walletActivity.Withdrawals[i].Currency,
			Amount:          walletActivity.Withdrawals[i].Amount.Float64(),
			Fee:             walletActivity.Withdrawals[i].Fee.Float64(),
			CryptoToAddress: walletActivity.Withdrawals[i].Address,
			CryptoTxID:      walletActivity.Withdrawals[i].TransactionID,
			TransferType:    "withdrawals",
		}
	}
	return resp, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (p *Poloniex) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	end := time.Now()
	withdrawals, err := p.WalletActivity(ctx, end.Add(-time.Hour*24*365), end, "withdrawals")
	if err != nil {
		return nil, err
	}
	resp := make([]exchange.WithdrawalHistory, 0, len(withdrawals.Withdrawals))
	for i := range withdrawals.Withdrawals {
		if !c.Equal(currency.NewCode(withdrawals.Withdrawals[i].Currency)) {
			continue
		}
		resp[i] = exchange.WithdrawalHistory{
			Status:          withdrawals.Withdrawals[i].Status,
			Timestamp:       withdrawals.Withdrawals[i].Timestamp.Time(),
			Currency:        withdrawals.Withdrawals[i].Currency,
			Amount:          withdrawals.Withdrawals[i].Amount.Float64(),
			Fee:             withdrawals.Withdrawals[i].Fee.Float64(),
			CryptoToAddress: withdrawals.Withdrawals[i].Address,
			CryptoTxID:      withdrawals.Withdrawals[i].TransactionID,
		}
	}
	return resp, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (p *Poloniex) GetRecentTrades(ctx context.Context, pair currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var err error
	pair, err = p.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		var tradeData []Trade
		tradeData, err = p.GetTrades(ctx, pair, 0)
		if err != nil {
			return nil, err
		}
		var side order.Side
		for i := range tradeData {
			side, err = order.StringToOrderSide(tradeData[i].TakerSide)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData[i].Price.Float64(),
				Amount:       tradeData[i].Amount.Float64(),
				Timestamp:    tradeData[i].Timestamp.Time(),
			})
		}
	case asset.Futures:
		var tradeData *TransactionHistory
		tradeData, err = p.GetTransactionHistory(ctx, pair.String())
		if err != nil {
			return nil, err
		}
		for i := range tradeData.Data {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData.Data[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData.Data[i].Price.Float64(),
				Amount:       tradeData.Data[i].Size.Float64(),
				Timestamp:    tradeData.Data[i].Timestamp.Time(),
			})
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	err = p.AddTradesToBuffer(resp...)
	if err != nil {
		return nil, err
	}
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (p *Poloniex) GetHistoricTrades(ctx context.Context, pair currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	if err := common.StartEndTimeCheck(timestampStart, timestampEnd); err != nil {
		return nil, fmt.Errorf("invalid time range supplied. Start: %v End %v %w", timestampStart, timestampEnd, err)
	}
	var err error
	pair, err = p.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	var resp []trade.Data
	switch assetType {
	case asset.Spot:
		ts := timestampStart
	allTrades:
		for {
			var tradeData []TradeHistoryItem
			tradeData, err = p.GetTradeHistory(ctx, currency.Pairs{pair}, "", 0, 0, ts, timestampEnd)
			if err != nil {
				return nil, err
			}
			for i := range tradeData {
				var tt time.Time
				if (tradeData[i].CreateTime.Time().Before(timestampStart) && !timestampStart.IsZero()) || (tradeData[i].CreateTime.Time().After(timestampEnd) && !timestampEnd.IsZero()) {
					break allTrades
				}
				var side order.Side
				side, err = order.StringToOrderSide(tradeData[i].Type)
				if err != nil {
					return nil, err
				}
				resp = append(resp, trade.Data{
					Exchange:     p.Name,
					CurrencyPair: pair,
					AssetType:    assetType,
					Side:         side,
					Price:        tradeData[i].Price.Float64(),
					Amount:       tradeData[i].Amount.Float64(),
					Timestamp:    tt,
				})
				if i == len(tradeData)-1 {
					if ts.Equal(tt) {
						// reached end of trades to crawl
						break allTrades
					}
					if timestampStart.IsZero() {
						break allTrades
					}
					ts = tt
				}
			}
		}
	case asset.Futures:
		var tradeData *TransactionHistory
		tradeData, err = p.GetTransactionHistory(ctx, pair.String())
		if err != nil {
			return nil, err
		}
		for i := range tradeData.Data {
			var side order.Side
			side, err = order.StringToOrderSide(tradeData.Data[i].Side)
			if err != nil {
				return nil, err
			}
			resp = append(resp, trade.Data{
				Exchange:     p.Name,
				CurrencyPair: pair,
				AssetType:    assetType,
				Side:         side,
				Price:        tradeData.Data[i].Price.Float64(),
				Amount:       tradeData.Data[i].Size.Float64(),
				Timestamp:    tradeData.Data[i].Timestamp.Time(),
			})
		}
	}
	if err := p.AddTradesToBuffer(resp...); err != nil {
		return nil, err
	}
	resp = trade.FilterTradesByTime(resp, timestampStart, timestampEnd)
	sort.Sort(trade.ByDate(resp))
	return resp, nil
}

// SubmitOrder submits a new order
func (p *Poloniex) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if s == nil {
		return nil, common.ErrNilPointer
	}
	if err := s.Validate(p.GetTradingRequirements()); err != nil {
		return nil, err
	}
	var err error
	s.Pair, err = p.FormatExchangeCurrency(s.Pair, s.AssetType)
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		var smartOrder bool
		var response *PlaceOrderResponse
		switch s.Type {
		case order.Stop, order.StopLimit, order.TrailingStop:
			smartOrder = true
		case order.Limit, order.Market, order.LimitMaker:
		default:
			return nil, fmt.Errorf("%v order type %v is not supported", order.ErrTypeIsInvalid, s.Type)
		}
		if smartOrder {
			var sOrder *PlaceOrderResponse
			sOrder, err = p.CreateSmartOrder(ctx, &SmartOrderRequestParam{
				Symbol:        s.Pair,
				Side:          orderSideString(s.Side),
				Type:          orderTypeString(s.Type),
				AccountType:   accountTypeString(s.AssetType),
				Price:         s.Price,
				StopPrice:     s.TriggerPrice,
				Quantity:      s.Amount,
				ClientOrderID: s.ClientOrderID,
			})
			if err != nil {
				return nil, err
			}
			return s.DeriveSubmitResponse(sOrder.ID)
		}
		arg := &PlaceOrderParams{
			Symbol:      s.Pair,
			Price:       s.Price,
			Amount:      s.Amount,
			AllowBorrow: false,
			Type:        s.Type.String(),
			Side:        s.Side.String(),
		}
		if p.Websocket.IsConnected() && p.Websocket.CanUseAuthenticatedEndpoints() && p.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
			response, err = p.WsCreateOrder(arg)
		} else {
			response, err = p.PlaceOrder(ctx, arg)
		}
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(response.ID)
	case asset.Futures:
		var stopOrderType, stopOrderBoundary string
		switch s.Type {
		case order.Stop, order.StopLimit, order.TrailingStop:
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
		}
		response, err := p.PlaceFuturesOrder(ctx, &FuturesOrderParams{
			ClientOrderID: s.ClientOrderID,
			Side:          orderSideString(s.Side),
			Symbol:        s.Pair.String(),
			OrderType:     orderTypeString(s.Type),
			Leverage:      int64(s.Leverage),
			Stop:          stopOrderBoundary,
			StopPrice:     s.TriggerPrice,
			StopPriceType: stopOrderType,
			ReduceOnly:    s.ReduceOnly,
			Hidden:        s.Hidden,
			PostOnly:      s.PostOnly,
			Price:         s.Price,
			Size:          s.Amount,
		})
		if err != nil {
			return nil, err
		}
		return s.DeriveSubmitResponse(response.OrderID)
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, s.AssetType)
	}
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (p *Poloniex) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if action == nil {
		return nil, common.ErrNilPointer
	}
	if err := action.Validate(); err != nil {
		return nil, err
	}
	if action.AssetType != asset.Spot {
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, action.AssetType)
	}
	switch action.Type {
	case order.Market, order.Limit, order.LimitMaker:
		resp, err := p.CancelReplaceOrder(ctx, &CancelReplaceOrderParam{
			orderID:       action.OrderID,
			ClientOrderID: action.ClientOrderID,
			Price:         action.Price,
			Quantity:      action.Amount,
			AmendedType:   action.Type.String(),
		})
		if err != nil {
			return nil, err
		}
		modResp, err := action.DeriveModifyResponse()
		if err != nil {
			return nil, err
		}
		modResp.OrderID = resp.ID
		return modResp, nil
	case order.Stop, order.StopLimit:
		oResp, err := p.CancelReplaceSmartOrder(ctx, &CancelReplaceSmartOrderParam{
			orderID:          action.OrderID,
			ClientOrderID:    action.ClientOrderID,
			Price:            action.Price,
			StopPrice:        action.TriggerPrice,
			Amount:           action.Amount,
			AmendedType:      orderTypeString(action.Type),
			ProceedOnFailure: !action.ImmediateOrCancel,
		})
		if err != nil {
			return nil, err
		}
		modResp, err := action.DeriveModifyResponse()
		if err != nil {
			return nil, err
		}
		modResp.OrderID = oResp.ID
		return modResp, nil
	default:
		return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, action.Type)
	}
}

// CancelOrder cancels an order by its corresponding ID number
func (p *Poloniex) CancelOrder(ctx context.Context, o *order.Cancel) error {
	if err := o.Validate(o.StandardCancel()); err != nil {
		return err
	}
	if o.OrderID == "" && o.ClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	var err error
	switch o.AssetType {
	case asset.Spot:
		switch o.Type {
		case order.Limit, order.Market:
			_, err = p.CancelOrderByID(ctx, o.OrderID)
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			_, err = p.CancelSmartOrderByID(ctx, o.OrderID, o.ClientOrderID)
		default:
			return fmt.Errorf("%w order type: %v", order.ErrUnsupportedOrderType, o.Type)
		}
	case asset.Futures:
		_, err = p.CancelFuturesOrderByID(ctx, o.OrderID)
	default:
		return fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, o.AssetType)
	}
	return err
}

// CancelBatchOrders cancels an orders by their corresponding ID numbers
func (p *Poloniex) CancelBatchOrders(ctx context.Context, o []order.Cancel) (*order.CancelBatchResponse, error) {
	if len(o) == 0 {
		return nil, order.ErrCancelOrderIsNil
	}
	orderIDs := make([]string, 0, len(o))
	clientOrderIDs := make([]string, 0, len(o))
	assetType := o[0].AssetType
	commonOrderType := o[0].Type
	for i := range o {
		if assetType != o[i].AssetType {
			return nil, errors.New("order asset type mismatch detected")
		}
		if commonOrderType != o[i].Type {
			commonOrderType = order.AnyType
		}
		switch {
		case o[i].ClientOrderID != "":
			clientOrderIDs = append(clientOrderIDs, o[i].ClientOrderID)
		case o[i].OrderID != "":
			orderIDs = append(orderIDs, o[i].OrderID)
		default:
			return nil, order.ErrOrderIDNotSet
		}
	}
	resp := &order.CancelBatchResponse{
		Status: make(map[string]string),
	}
	switch assetType {
	case asset.Spot:
		switch commonOrderType {
		case order.Market, order.Limit:
			if p.Websocket.IsConnected() && p.Websocket.CanUseAuthenticatedEndpoints() && p.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				wsCancelledOrders, err := p.WsCancelMultipleOrdersByIDs(&OrderCancellationParams{OrderIDs: orderIDs, ClientOrderIDs: clientOrderIDs})
				if err != nil {
					return nil, err
				}
				for i := range wsCancelledOrders {
					if wsCancelledOrders[i].ClientOrderID != "" {
						resp.Status[wsCancelledOrders[i].ClientOrderID] = wsCancelledOrders[i].State + " " + wsCancelledOrders[i].Message
						continue
					}
					orderID := strconv.FormatInt(wsCancelledOrders[i].OrderID, 10)
					resp.Status[orderID] = wsCancelledOrders[i].State + " " + wsCancelledOrders[i].Message
				}
			} else {
				cancelledOrders, err := p.CancelMultipleOrdersByIDs(ctx, &OrderCancellationParams{OrderIDs: orderIDs, ClientOrderIDs: clientOrderIDs})
				if err != nil {
					return nil, err
				}
				for i := range cancelledOrders {
					if cancelledOrders[i].ClientOrderID != "" {
						resp.Status[cancelledOrders[i].ClientOrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
						continue
					}
					resp.Status[cancelledOrders[i].OrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
				}
			}
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			cancelledOrders, err := p.CancelMultipleSmartOrders(ctx, &OrderCancellationParams{
				OrderIDs:       orderIDs,
				ClientOrderIDs: clientOrderIDs,
			})
			if err != nil {
				return nil, err
			}
			for i := range cancelledOrders {
				if cancelledOrders[i].ClientOrderID != "" {
					resp.Status[cancelledOrders[i].ClientOrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
					continue
				}
				resp.Status[cancelledOrders[i].OrderID] = cancelledOrders[i].State + " " + cancelledOrders[i].Message
			}
		default:
			return nil, fmt.Errorf("%w %s", order.ErrUnsupportedOrderType, commonOrderType.String())
		}
	case asset.Futures:
		switch commonOrderType {
		case order.Limit, order.Market:
			cancelledOrders, err := p.CancelMultipleFuturesLimitOrders(ctx, orderIDs, clientOrderIDs)
			if err != nil {
				return nil, err
			}
			resp.Status = make(map[string]string, len(cancelledOrders.CancelFailedOrderIDs)+len(cancelledOrders.CancelledOrderIDs))
			for x := range cancelledOrders.CancelledOrderIDs {
				resp.Status[cancelledOrders.CancelledOrderIDs[x]] = "Cancelled"
			}
			for x := range cancelledOrders.CancelFailedOrderIDs {
				resp.Status[cancelledOrders.CancelFailedOrderIDs[x]] = "Failed"
			}
		default:
			return nil, fmt.Errorf("futures order cancellation for %s orders is not supported", commonOrderType.String())
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (p *Poloniex) CancelAllOrders(ctx context.Context, cancelOrd *order.Cancel) (order.CancelAllResponse, error) {
	cancelAllOrdersResponse := order.CancelAllResponse{
		Status: make(map[string]string),
	}
	if cancelOrd == nil {
		return cancelAllOrdersResponse, common.ErrNilPointer
	}
	var err error
	var pairs currency.Pairs
	if !cancelOrd.Pair.IsEmpty() {
		pairs = append(pairs, cancelOrd.Pair)
	}
	var resp []CancelOrderResponse
	switch cancelOrd.AssetType {
	case asset.Spot:
		switch cancelOrd.Type {
		case order.Market, order.Limit:
			if p.Websocket.IsConnected() && p.Websocket.CanUseAuthenticatedEndpoints() && p.Websocket.CanUseAuthenticatedWebsocketForWrapper() {
				var wsResponse []WsCancelOrderResponse
				wsResponse, err = p.WsCancelAllTradeOrders(pairs.Strings(), []string{accountTypeString(cancelOrd.AssetType)})
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				for x := range wsResponse {
					cancelAllOrdersResponse.Status[strconv.FormatInt(wsResponse[x].OrderID, 10)] = wsResponse[x].State
				}
			} else {
				resp, err = p.CancelAllTradeOrders(ctx, pairs.Strings(), []string{accountTypeString(cancelOrd.AssetType)})
				if err != nil {
					return cancelAllOrdersResponse, err
				}
				for x := range resp {
					cancelAllOrdersResponse.Status[resp[x].OrderID] = resp[x].State
				}
			}
		case order.TrailingStop, order.TrailingStopLimit, order.StopLimit, order.Stop:
			pairsString := []string{}
			if !cancelOrd.Pair.IsEmpty() {
				pairsString = append(pairsString, cancelOrd.Pair.String())
			}
			orderTypes := []string{}
			if cancelOrd.Type != order.UnknownType {
				orderTypes = append(orderTypes, orderTypeString(cancelOrd.Type))
			}
			var resp []CancelOrderResponse
			resp, err = p.CancelAllSmartOrders(ctx, pairsString, nil, orderTypes)
			if err != nil {
				return cancelAllOrdersResponse, err
			}
			for x := range resp {
				cancelAllOrdersResponse.Status[resp[x].OrderID] = resp[x].State
			}
		default:
			return cancelAllOrdersResponse, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, cancelOrd.Type)
		}
	case asset.Futures:
		var result *FuturesCancelOrderResponse
		switch cancelOrd.Type {
		case order.Limit:
			result, err = p.CancelAllFuturesLimitOrders(ctx, cancelOrd.Pair.String(), orderSideString(cancelOrd.Side))
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			result, err = p.CancelAllFuturesStopOrders(ctx, cancelOrd.Pair.String())
			if err != nil {
				return cancelAllOrdersResponse, err
			}
		default:
			return cancelAllOrdersResponse, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, cancelOrd.Type)
		}
		for x := range result.CancelledOrderIDs {
			cancelAllOrdersResponse.Status[result.CancelledOrderIDs[x]] = "Cancelled"
		}
		for x := range result.CancelFailedOrderIDs {
			cancelAllOrdersResponse.Status[result.CancelFailedOrderIDs[x]] = "Failed"
		}
	default:
		return cancelAllOrdersResponse, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, cancelOrd.AssetType)
	}
	return cancelAllOrdersResponse, nil
}

// GetOrderInfo returns order information based on order ID
func (p *Poloniex) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch assetType {
	case asset.Spot:
		trades, err := p.GetTradesByOrderID(ctx, orderID)
		if err != nil && !strings.Contains(err.Error(), "Order not found") {
			return nil, err
		}
		orderTrades := make([]order.TradeHistory, len(trades))
		var oType order.Type
		var oSide order.Side
		for i := range trades {
			oType, err = order.StringToOrderType(trades[i].Type)
			if err != nil {
				return nil, err
			}
			oSide, err = order.StringToOrderSide(trades[i].Side)
			if err != nil {
				return nil, err
			}
			orderTrades[i] = order.TradeHistory{
				Price:     trades[i].Price.Float64(),
				Amount:    trades[i].Quantity.Float64(),
				Fee:       trades[i].FeeAmount.Float64(),
				Exchange:  p.Name,
				TID:       trades[i].ID,
				Type:      oType,
				Side:      oSide,
				Timestamp: trades[i].CreateTime.Time(),
				FeeAsset:  trades[i].FeeCurrency,
				Total:     trades[i].Amount.Float64(),
			}
		}
		var smartOrders []SmartOrderDetail
		resp, err := p.GetOrderDetail(ctx, orderID, "")
		if err != nil {
			smartOrders, err = p.GetSmartOrderDetail(ctx, orderID, "")
			if err != nil {
				return nil, err
			} else if len(smartOrders) == 0 {
				return nil, order.ErrOrderNotFound
			}
		}

		var dPair currency.Pair
		var oStatus order.Status
		if len(smartOrders) > 0 {
			dPair, err = currency.NewPairFromString(smartOrders[0].Symbol)
			if err != nil {
				return nil, err
			} else if !pair.IsEmpty() && !dPair.Equal(pair) {
				return nil, fmt.Errorf("order with ID %s expected a symbol %v, but got %v", orderID, pair, dPair)
			}
			oType, err = order.StringToOrderType(smartOrders[0].Type)
			if err != nil {
				return nil, err
			}
			oStatus, err = order.StringToOrderStatus(smartOrders[0].State)
			if err != nil {
				return nil, err
			}
			oSide, err = order.StringToOrderSide(smartOrders[0].Side)
			if err != nil {
				return nil, err
			}
			return &order.Detail{
				Price:         smartOrders[0].Price.Float64(),
				Amount:        smartOrders[0].Quantity.Float64(),
				QuoteAmount:   smartOrders[0].Amount.Float64(),
				Exchange:      p.Name,
				OrderID:       smartOrders[0].ID,
				ClientOrderID: smartOrders[0].ClientOrderID,
				Type:          oType,
				Side:          oSide,
				Status:        oStatus,
				AssetType:     stringToAccountType(smartOrders[0].AccountType),
				Date:          smartOrders[0].CreateTime.Time(),
				LastUpdated:   smartOrders[0].UpdateTime.Time(),
				Pair:          dPair,
				Trades:        orderTrades,
			}, nil
		}
		dPair, err = currency.NewPairFromString(resp.Symbol)
		if err != nil {
			return nil, err
		} else if !pair.IsEmpty() && !dPair.Equal(pair) {
			return nil, fmt.Errorf("order with ID %s expected a symbol %v, but got %v", orderID, pair, dPair)
		}
		oType, err = order.StringToOrderType(resp.Type)
		if err != nil {
			return nil, err
		}
		oStatus, err = order.StringToOrderStatus(resp.State)
		if err != nil {
			return nil, err
		}
		oSide, err = order.StringToOrderSide(resp.Side)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Price:                resp.Price.Float64(),
			Amount:               resp.Quantity.Float64(),
			AverageExecutedPrice: resp.AvgPrice.Float64(),
			QuoteAmount:          resp.Amount.Float64(),
			ExecutedAmount:       resp.FilledQuantity.Float64(),
			RemainingAmount:      resp.Quantity.Float64() - resp.FilledAmount.Float64(),
			Cost:                 resp.FilledQuantity.Float64() * resp.AvgPrice.Float64(),
			Exchange:             p.Name,
			OrderID:              resp.ID,
			ClientOrderID:        resp.ClientOrderID,
			Type:                 oType,
			Side:                 oSide,
			Status:               oStatus,
			AssetType:            stringToAccountType(resp.AccountType),
			Date:                 resp.CreateTime.Time(),
			LastUpdated:          resp.UpdateTime.Time(),
			Pair:                 dPair,
			Trades:               orderTrades,
		}, nil
	case asset.Futures:
		fResults, err := p.GetFuturesSingleOrderDetailByOrderID(ctx, orderID)
		if err != nil {
			return nil, err
		}
		dPair, err := currency.NewPairFromString(fResults.Symbol)
		if err != nil {
			return nil, err
		} else if !pair.IsEmpty() && !dPair.Equal(pair) {
			return nil, fmt.Errorf("order with ID %s expected a symbol %v, but got %v", orderID, pair, dPair)
		}
		oType, err := order.StringToOrderType(fResults.OrderType)
		if err != nil {
			return nil, err
		}
		oStatus, err := order.StringToOrderStatus(fResults.Status)
		if err != nil {
			return nil, err
		}
		oSide, err := order.StringToOrderSide(fResults.Side)
		if err != nil {
			return nil, err
		}
		return &order.Detail{
			Price:  fResults.Price.Float64(),
			Amount: fResults.Size,
			// AverageExecutedPrice: fResults..Float64(),
			QuoteAmount:     fResults.Value.Float64(),
			ExecutedAmount:  fResults.FilledSize,
			RemainingAmount: fResults.Size - fResults.FilledSize,
			OrderID:         fResults.OrderID,
			Exchange:        p.Name,
			ClientOrderID:   fResults.ClientOrderID,
			Type:            oType,
			Side:            oSide,
			Status:          oStatus,
			AssetType:       asset.Futures,
			Date:            fResults.CreatedAt.Time(),
			LastUpdated:     fResults.UpdatedAt.Time(),
			Pair:            dPair,
		}, nil
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, assetType)
	}
}

// GetDepositAddress returns a deposit address for a specified currency
func (p *Poloniex) GetDepositAddress(ctx context.Context, cryptocurrency currency.Code, _, chain string) (*deposit.Address, error) {
	depositAddrs, err := p.GetDepositAddresses(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	// Some coins use a main address, so we must use this in conjunction with the returned
	// deposit address to produce the full deposit address and payment-id
	currencies, err := p.GetCurrencyInformation(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}

	coinParams, ok := currencies[cryptocurrency.Upper().String()]
	if !ok {
		return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
	}

	var address, paymentID string
	if coinParams.Type == "address-payment-id" && coinParams.DepositAddress != "" {
		paymentID, ok = (*depositAddrs)[cryptocurrency.Upper().String()]
		if !ok {
			newAddr, err := p.NewCurrencyDepositAddress(ctx, cryptocurrency)
			if err != nil {
				return nil, err
			}
			paymentID = newAddr
		}
		return &deposit.Address{
			Address: coinParams.DepositAddress,
			Tag:     paymentID,
			Chain:   coinParams.ParentChain,
		}, nil
	}

	address, ok = (*depositAddrs)[cryptocurrency.Upper().String()]
	if !ok {
		if len(coinParams.ChildChains) > 1 && chain != "" && !slices.Contains(coinParams.ChildChains, chain) {
			return nil, fmt.Errorf("currency %s has %v chains available, one of these must be specified",
				cryptocurrency,
				coinParams.ChildChains)
		}

		coinParams, ok = currencies[cryptocurrency.Upper().String()]
		if !ok {
			return nil, fmt.Errorf("unable to find currency %s in map", cryptocurrency)
		}
		if coinParams.WalletDepositState != "ENABLED" {
			return nil, fmt.Errorf("deposits and withdrawals for %v are currently disabled", cryptocurrency.Upper().String())
		}

		newAddr, err := p.NewCurrencyDepositAddress(ctx, cryptocurrency)
		if err != nil {
			return nil, err
		}
		address = newAddr
	}
	return &deposit.Address{
		Address: address,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (p *Poloniex) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	if withdrawRequest == nil {
		return nil, common.ErrNilPointer
	}
	if err := withdrawRequest.Validate(); err != nil {
		return nil, err
	}
	v, err := p.WithdrawCurrency(ctx, &WithdrawCurrencyParam{
		Currency: withdrawRequest.Currency,
		Address:  withdrawRequest.Crypto.Address,
		Amount:   withdrawRequest.Amount})
	if err != nil {
		return nil, err
	}
	return &withdraw.ExchangeResponse{
		Name: p.Name,
		ID:   strconv.FormatInt(v.WithdrawRequestID, 10),
	}, err
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFunds(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (p *Poloniex) WithdrawFiatFundsToInternationalBank(_ context.Context, _ *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (p *Poloniex) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	if (!p.AreCredentialsValid(ctx) || p.SkipAuthCheck) && // Todo check connection status
		feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		feeBuilder.FeeType = exchange.OfflineTradeFee
	}
	return p.GetFee(ctx, feeBuilder)
}

// GetActiveOrders retrieves any orders that are active/open
func (p *Poloniex) GetActiveOrders(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	var samplePair currency.Pair
	if len(req.Pairs) == 1 {
		samplePair = req.Pairs[0]
	}
	var orders []order.Detail
	switch req.AssetType {
	case asset.Spot:
		resp, err := p.GetOpenOrders(ctx, samplePair, orderSideString(req.Side), "", req.FromOrderID, 0)
		if err != nil {
			return nil, err
		}
		for a := range resp {
			var symbol currency.Pair
			symbol, err = currency.NewPairFromString(resp[a].Symbol)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) != 0 && req.Pairs.Contains(symbol, true) {
				continue
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(resp[a].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(resp[a].Type)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order.Detail{
				Type:     oType,
				OrderID:  resp[a].ID,
				Side:     orderSide,
				Amount:   resp[a].Amount.Float64(),
				Date:     resp[a].CreateTime.Time(),
				Price:    resp[a].Price.Float64(),
				Pair:     symbol,
				Exchange: p.Name,
			})
		}
	case asset.Futures:
		fOrders, err := p.GetFuturesOrderList(context.Background(), "active", samplePair.String(), orderSideString(req.Side), orderTypeString(req.Type), req.StartTime, req.EndTime, margin.Unset)
		if err != nil {
			return nil, err
		}
		for a := range fOrders.Items {
			var symbol currency.Pair
			symbol, err = currency.NewPairFromString(fOrders.Items[a].Symbol)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) != 0 && req.Pairs.Contains(symbol, true) {
				continue
			}
			var orderSide order.Side
			orderSide, err = order.StringToOrderSide(fOrders.Items[a].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(fOrders.Items[a].OrderType)
			if err != nil {
				return nil, err
			}
			oStatus, err := order.StringToOrderStatus(fOrders.Items[a].Status)
			if err != nil {
				return nil, err
			}
			var mType margin.Type
			switch fOrders.Items[a].MarginType {
			case 0:
				mType = margin.Isolated
			case 1:
				mType = margin.Multi
			}
			trades := make([]order.TradeHistory, len(fOrders.Items[a].Trades))
			for t := range fOrders.Items[a].Trades {
				trades[t] = order.TradeHistory{
					TID:      fOrders.Items[a].Trades[t].TradeID,
					Fee:      fOrders.Items[a].Trades[t].FeePay,
					Exchange: p.Name,
				}
			}
			orders = append(orders, order.Detail{
				Type:               oType,
				OrderID:            fOrders.Items[a].OrderID,
				Side:               orderSide,
				Amount:             fOrders.Items[a].Size,
				Date:               fOrders.Items[a].CreatedAt.Time(),
				Price:              fOrders.Items[a].Price.Float64(),
				Pair:               symbol,
				Exchange:           p.Name,
				HiddenOrder:        fOrders.Items[a].Hidden,
				PostOnly:           fOrders.Items[a].PostOnly,
				ReduceOnly:         fOrders.Items[a].ReduceOnly,
				Leverage:           fOrders.Items[a].Leverage.Float64(),
				ExecutedAmount:     fOrders.Items[a].FilledSize,
				RemainingAmount:    fOrders.Items[a].Size - fOrders.Items[a].FilledSize,
				ClientOrderID:      fOrders.Items[a].ClientOrderID,
				Status:             oStatus,
				AssetType:          req.AssetType,
				LastUpdated:        fOrders.Items[a].UpdatedAt.Time(),
				MarginType:         mType,
				Trades:             trades,
				SettlementCurrency: currency.NewCode(fOrders.Items[a].SettleCurrency),
			})
		}
	default:
		return nil, fmt.Errorf("%w asset type: %v", asset.ErrNotSupported, req.AssetType)
	}
	return req.Filter(p.Name, orders), nil
}

func accountTypeString(assetType asset.Item) string {
	switch assetType {
	case asset.Spot:
		return "SPOT"
	case asset.Futures:
		return "FUTURE"
	default:
		return ""
	}
}

func stringToAccountType(assetType string) asset.Item {
	switch assetType {
	case "SPOT":
		return asset.Spot
	case "FUTURE":
		return asset.Futures
	default:
		return asset.Empty
	}
}

func orderSideString(oSide order.Side) string {
	switch oSide {
	case order.Buy, order.Sell:
		return oSide.String()
	default:
		return ""
	}
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (p *Poloniex) GetOrderHistory(ctx context.Context, req *order.MultiOrderRequest) (order.FilteredOrders, error) {
	if req == nil {
		return nil, common.ErrNilPointer
	}
	err := req.Validate()
	if err != nil {
		return nil, err
	}
	switch req.AssetType {
	case asset.Spot:
		switch req.Type {
		case order.Market, order.Limit:
			resp, err := p.GetOrdersHistory(ctx, currency.EMPTYPAIR, accountTypeString(req.AssetType), orderTypeString(req.Type), orderSideString(req.Side), "", "", 0, 100, req.StartTime, req.EndTime, false)
			if err != nil {
				return nil, err
			}

			var oSide order.Side
			var oType order.Type
			orders := make([]order.Detail, 0, len(resp))
			for i := range resp {
				var pair currency.Pair
				pair, err = currency.NewPairFromString(resp[i].Symbol)
				if err != nil {
					return nil, err
				}
				if len(req.Pairs) != 0 && !req.Pairs.Contains(pair, true) {
					continue
				}
				oSide, err = order.StringToOrderSide(resp[i].Side)
				if err != nil {
					return nil, err
				}
				oType, err = order.StringToOrderType(resp[i].Type)
				if err != nil {
					return nil, err
				}
				var assetType asset.Item
				assetType, err = asset.New(resp[i].AccountType)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					Side:                 oSide,
					Amount:               resp[i].Amount.Float64(),
					ExecutedAmount:       resp[i].FilledAmount.Float64(),
					Price:                resp[i].Price.Float64(),
					AverageExecutedPrice: resp[i].AvgPrice.Float64(),
					Pair:                 pair,
					Type:                 oType,
					Exchange:             p.Name,
					QuoteAmount:          resp[i].Amount.Float64() * resp[i].AvgPrice.Float64(),
					RemainingAmount:      resp[i].Quantity.Float64() - resp[i].FilledQuantity.Float64(),
					OrderID:              resp[i].ID,
					ClientOrderID:        resp[i].ClientOrderID,
					Status:               order.Filled,
					AssetType:            assetType,
					Date:                 resp[i].CreateTime.Time(),
					LastUpdated:          resp[i].UpdateTime.Time(),
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
			return req.Filter(p.Name, orders), nil
		case order.Stop, order.StopLimit, order.TrailingStop, order.TrailingStopLimit:
			smartOrders, err := p.GetSmartOrderHistory(ctx, currency.EMPTYPAIR, accountTypeString(req.AssetType),
				orderTypeString(req.Type), orderSideString(req.Side), "", "", 0, 100, req.StartTime, req.EndTime, false)
			if err != nil {
				return nil, err
			}
			var oSide order.Side
			var oType order.Type
			orders := make([]order.Detail, 0, len(smartOrders))
			for i := range smartOrders {
				var pair currency.Pair
				pair, err = currency.NewPairFromString(smartOrders[i].Symbol)
				if err != nil {
					return nil, err
				}
				if len(req.Pairs) != 0 && !req.Pairs.Contains(pair, true) {
					continue
				}
				oSide, err = order.StringToOrderSide(smartOrders[i].Side)
				if err != nil {
					return nil, err
				}
				oType, err = order.StringToOrderType(smartOrders[i].Type)
				if err != nil {
					return nil, err
				}
				assetType, err := asset.New(smartOrders[i].AccountType)
				if err != nil {
					return nil, err
				}
				detail := order.Detail{
					Side:          oSide,
					Amount:        smartOrders[i].Amount.Float64(),
					Price:         smartOrders[i].Price.Float64(),
					TriggerPrice:  smartOrders[i].StopPrice.Float64(),
					Pair:          pair,
					Type:          oType,
					Exchange:      p.Name,
					OrderID:       smartOrders[i].ID,
					ClientOrderID: smartOrders[i].ClientOrderID,
					Status:        order.Filled,
					AssetType:     assetType,
					Date:          smartOrders[i].CreateTime.Time(),
					LastUpdated:   smartOrders[i].UpdateTime.Time(),
				}
				detail.InferCostsAndTimes()
				orders = append(orders, detail)
			}
			return orders, nil
		default:
			return nil, fmt.Errorf("%w %v", order.ErrUnsupportedOrderType, req.Type)
		}
	case asset.Futures:
		orderHistory, err := p.GetFuturesOrderListV2(ctx, "", "", orderSideString(req.Side), orderTypeString(req.Type), "", req.StartTime, req.EndTime, 0)
		if err != nil {
			return nil, err
		}
		var oSide order.Side
		var oType order.Type
		orders := make([]order.Detail, 0, len(orderHistory.Items))
		for i := range orderHistory.Items {
			var pair currency.Pair
			pair, err = currency.NewPairFromString(orderHistory.Items[i].Symbol)
			if err != nil {
				return nil, err
			}
			if len(req.Pairs) != 0 && !req.Pairs.Contains(pair, true) {
				continue
			}
			oSide, err = order.StringToOrderSide(orderHistory.Items[i].Side)
			if err != nil {
				return nil, err
			}
			oType, err = order.StringToOrderType(orderHistory.Items[i].OrderType)
			if err != nil {
				return nil, err
			}
			detail := order.Detail{
				Side:            oSide,
				Amount:          orderHistory.Items[i].Size,
				ExecutedAmount:  orderHistory.Items[i].FilledSize,
				Price:           orderHistory.Items[i].Price.Float64(),
				Pair:            pair,
				Type:            oType,
				Exchange:        p.Name,
				RemainingAmount: orderHistory.Items[i].Size - orderHistory.Items[i].FilledSize,
				OrderID:         orderHistory.Items[i].OrderID,
				ClientOrderID:   orderHistory.Items[i].ClientOrderID,
				Status:          order.Filled,
				AssetType:       asset.Futures,
				Date:            orderHistory.Items[i].CreatedAt.Time(),
				LastUpdated:     orderHistory.Items[i].UpdatedAt.Time(),
			}
			detail.InferCostsAndTimes()
			orders = append(orders, detail)
		}
		return req.Filter(p.Name, orders), nil
	default:
		return nil, fmt.Errorf("%w asset type %v", asset.ErrNotSupported, req.AssetType)
	}
}

// ValidateAPICredentials validates current credentials used for wrapper
// functionality
func (p *Poloniex) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := p.UpdateAccountInfo(ctx, assetType)
	return p.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := p.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	switch a {
	case asset.Spot:
		resp, err := p.GetCandlesticks(ctx, req.RequestFormatted, req.ExchangeInterval, req.Start, req.End, req.RequestLimit)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(resp))
		for x := range resp {
			timeSeries[x] = kline.Candle{
				Time:   resp[x].StartTime,
				Open:   resp[x].Open,
				High:   resp[x].High,
				Low:    resp[x].Low,
				Close:  resp[x].Close,
				Volume: resp[x].Quantity,
			}
		}
		return req.ProcessResponse(timeSeries)
	case asset.Futures:
		resp, err := p.GetFuturesKlineDataOfContract(ctx, req.RequestFormatted.String(), int64(req.ExchangeInterval.Duration().Minutes()), req.Start, req.End)
		if err != nil {
			return nil, err
		}
		timeSeries := make([]kline.Candle, len(resp))
		for x := range resp {
			timeSeries[x] = kline.Candle{
				Time:   resp[x].Timestamp,
				Open:   resp[x].EntryPrice,
				High:   resp[x].HighestPrice,
				Low:    resp[x].LowestPrice,
				Volume: resp[x].TradingVolume,
			}
		}
		return req.ProcessResponse(timeSeries)
	}

	return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
}

func (p *Poloniex) getGranularityFromInterval(interval kline.Interval) (int64, error) {
	switch interval {
	case kline.OneMin, kline.FiveMin, kline.FifteenMin, kline.ThirtyMin, kline.OneHour, kline.TwoHour, kline.FourHour,
		kline.EightHour, kline.TwelveHour, kline.OneDay, kline.SevenDay:
		return int64(interval.Duration().Minutes()), nil
	default:
		return 0, kline.ErrUnsupportedInterval
	}
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (p *Poloniex) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := p.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var timeSeries []kline.Candle
	switch a {
	case asset.Spot:
		for i := range req.RangeHolder.Ranges {
			resp, err := p.GetCandlesticks(ctx,
				req.RequestFormatted,
				req.ExchangeInterval,
				req.RangeHolder.Ranges[i].Start.Time,
				req.RangeHolder.Ranges[i].End.Time,
				req.RequestLimit,
			)
			if err != nil {
				return nil, err
			}
			for x := range resp {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   resp[x].StartTime,
					Open:   resp[x].Open,
					High:   resp[x].High,
					Low:    resp[x].Low,
					Close:  resp[x].Close,
					Volume: resp[x].Quantity,
				})
			}
		}
	case asset.Futures:
		granularity, err := p.getGranularityFromInterval(interval)
		if err != nil {
			return nil, err
		}
		for i := range req.RangeHolder.Ranges {
			resp, err := p.GetFuturesKlineDataOfContract(ctx,
				req.RequestFormatted.String(),
				granularity,
				req.RangeHolder.Ranges[i].Start.Time,
				req.RangeHolder.Ranges[i].End.Time,
			)
			if err != nil {
				return nil, err
			}
			for x := range resp {
				timeSeries = append(timeSeries, kline.Candle{
					Time:   resp[x].Timestamp,
					Open:   resp[x].EntryPrice,
					High:   resp[x].HighestPrice,
					Low:    resp[x].LowestPrice,
					Volume: resp[x].TradingVolume,
				})
			}
		}
	default:
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
	return req.ProcessResponse(timeSeries)
}

// GetAvailableTransferChains returns the available transfer blockchains for the specific
// cryptocurrency
func (p *Poloniex) GetAvailableTransferChains(ctx context.Context, cryptocurrency currency.Code) ([]string, error) {
	if cryptocurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	currencies, err := p.GetV2CurrencyInformation(ctx, cryptocurrency)
	if err != nil {
		return nil, err
	}
	if len(currencies.NetworkList) == 0 {
		return nil, fmt.Errorf("%w for currency %v", errChainsNotFound, cryptocurrency)
	}
	chains := make([]string, len(currencies.NetworkList))
	for a := range currencies.NetworkList {
		chains[a] = currencies.NetworkList[a].Blockchain
	}
	return chains, nil
}

// GetServerTime returns the current exchange server time.
func (p *Poloniex) GetServerTime(ctx context.Context, assetType asset.Item) (time.Time, error) {
	switch assetType {
	case asset.Spot:
		sysServerTime, err := p.GetSystemTimestamp(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return sysServerTime.ServerTime.Time(), nil
	case asset.Futures:
		sysServerTime, err := p.GetFuturesServerTime(ctx)
		if err != nil {
			return time.Time{}, err
		}
		return sysServerTime.Data.Time(), nil
	default:
		return time.Time{}, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (p *Poloniex) GetFuturesContractDetails(ctx context.Context, assetType asset.Item) ([]futures.Contract, error) {
	if !assetType.IsFutures() {
		return nil, futures.ErrNotFuturesAsset
	}
	if !p.SupportsAsset(assetType) || assetType != asset.Futures {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, assetType)
	}

	contracts, err := p.GetOpenContractList(ctx)
	if err != nil {
		return nil, err
	}

	resp := make([]futures.Contract, len(contracts.Data))
	for i := range contracts.Data {
		var cp, underlying currency.Pair
		underlying, err = currency.NewPairFromStrings(contracts.Data[i].BaseCurrency, contracts.Data[i].QuoteCurrency)
		if err != nil {
			return nil, err
		}
		cp, err = currency.NewPairFromStrings(contracts.Data[i].BaseCurrency, contracts.Data[i].Symbol[len(contracts.Data[i].BaseCurrency):])
		if err != nil {
			return nil, err
		}
		settleCurr := currency.NewCode(contracts.Data[i].SettleCurrency)
		var ct futures.ContractType
		if contracts.Data[i].ContractType == "FFWCSX" {
			ct = futures.Perpetual
		} else {
			ct = futures.Quarterly
		}
		contractSettlementType := futures.Linear
		if contracts.Data[i].IsInverse {
			contractSettlementType = futures.Inverse
		}
		resp[i] = futures.Contract{
			Exchange:             p.Name,
			Name:                 cp,
			Underlying:           underlying,
			SettlementCurrencies: currency.Currencies{settleCurr},
			MarginCurrency:       settleCurr,
			Asset:                assetType,
			StartDate:            time.UnixMilli(contracts.Data[i].CreatedAt),
			IsActive:             !strings.EqualFold(contracts.Data[i].Status, "closed"),
			Status:               contracts.Data[i].Status,
			Multiplier:           contracts.Data[i].Multiplier,
			MaxLeverage:          contracts.Data[i].MaxLeverage,
			SettlementType:       contractSettlementType,
			LatestRate: fundingrate.Rate{
				Rate: decimal.NewFromFloat(contracts.Data[i].FundingFeeRate),
				Time: contracts.Data[i].NextFundingRateTime.Time(),
			},
			Type: ct,
		}
	}
	return resp, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (p *Poloniex) GetLatestFundingRates(ctx context.Context, r *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w LatestRateRequest", common.ErrNilPointer)
	}
	var fri time.Duration
	if len(p.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies) == 1 {
		for k := range p.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies {
			fri = k.Duration()
		}
	}
	if r.Pair.IsEmpty() {
		contracts, err := p.GetOpenContractList(ctx)
		if err != nil {
			return nil, err
		}
		if r.IncludePredictedRate {
			log.Warnf(log.ExchangeSys, "%s predicted rate for all currencies requires an additional %v requests", p.Name, len(contracts.Data))
		}
		timeChecked := time.Now()
		resp := make([]fundingrate.LatestRateResponse, 0, len(contracts.Data))
		for i := range contracts.Data {
			var cp currency.Pair
			cp, err = currency.NewPairFromStrings(contracts.Data[i].BaseCurrency, contracts.Data[i].Symbol[len(contracts.Data[i].BaseCurrency):])
			if err != nil {
				return nil, err
			}
			var isPerp bool
			isPerp, err = p.IsPerpetualFutureCurrency(r.Asset, cp)
			if err != nil {
				return nil, err
			}
			if !isPerp {
				continue
			}

			rate := fundingrate.LatestRateResponse{
				Exchange: p.Name,
				Asset:    r.Asset,
				Pair:     cp,
				LatestRate: fundingrate.Rate{
					Time: contracts.Data[i].NextFundingRateTime.Time().Add(-fri),
					Rate: decimal.NewFromFloat(contracts.Data[i].FundingFeeRate),
				},
				TimeOfNextRate: contracts.Data[i].NextFundingRateTime.Time(),
				TimeChecked:    timeChecked,
			}
			if r.IncludePredictedRate {
				fr, err := p.GetCurrentFundingRate(ctx, contracts.Data[i].Symbol)
				if err != nil {
					return nil, err
				}
				rate.PredictedUpcomingRate = fundingrate.Rate{
					Time: contracts.Data[i].NextFundingRateTime.Time(),
					Rate: decimal.NewFromFloat(fr.PredictedValue),
				}
			}
			resp = append(resp, rate)
		}
		return resp, nil
	}
	resp := make([]fundingrate.LatestRateResponse, 1)
	is, err := p.IsPerpetualFutureCurrency(r.Asset, r.Pair)
	if err != nil {
		return nil, err
	}
	if !is {
		return nil, fmt.Errorf("%w %s %v", futures.ErrNotPerpetualFuture, r.Asset, r.Pair)
	}
	fPair, err := p.FormatExchangeCurrency(r.Pair, r.Asset)
	if err != nil {
		return nil, err
	}
	fr, err := p.GetCurrentFundingRate(ctx, fPair.String())
	if err != nil {
		return nil, err
	}
	rate := fundingrate.LatestRateResponse{
		Exchange: p.Name,
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
func (p *Poloniex) IsPerpetualFutureCurrency(a asset.Item, cp currency.Pair) (bool, error) {
	switch {
	case cp.IsEmpty() || a != asset.Futures:
		return false, nil
	case strings.HasSuffix(cp.Quote.String(), "PERP"):
		return true, nil
	default:
		pairString, err := p.FormatSymbol(cp, asset.Futures)
		if err != nil {
			return false, err
		}
		info, err := p.GetOrderInfoOfTheContract(context.Background(), pairString)
		if err != nil {
			return false, err
		}
		if info.ContractType == "FFWCSX" {
			return true, nil
		}
		return false, nil
	}
}

// UpdateOrderExecutionLimits updates order execution limits
func (p *Poloniex) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	if !p.SupportsAsset(a) {
		return fmt.Errorf("%w asset: %v", asset.ErrNotSupported, a)
	}
	instruments, err := p.GetSymbolInformation(ctx, currency.EMPTYPAIR)
	if err != nil {
		return err
	}
	limits := make([]order.MinMaxLevel, len(instruments))
	for x := range instruments {
		pair, err := currency.NewPairFromString(instruments[x].Symbol)
		if err != nil {
			return err
		}

		limits[x] = order.MinMaxLevel{
			Pair:                    pair,
			Asset:                   a,
			PriceStepIncrementSize:  instruments[x].SymbolTradeLimit.PriceScale,
			MinimumBaseAmount:       instruments[x].SymbolTradeLimit.MinQuantity.Float64(),
			MinimumQuoteAmount:      instruments[x].SymbolTradeLimit.MinAmount.Float64(),
			AmountStepIncrementSize: instruments[x].SymbolTradeLimit.AmountScale,
			QuoteStepIncrementSize:  instruments[x].SymbolTradeLimit.QuantityScale,
		}
	}
	return p.LoadLimits(limits)
}

// GetCurrencyTradeURL returns the URL to the exchange's trade page for the given asset and currency pair
func (p *Poloniex) GetCurrencyTradeURL(_ context.Context, a asset.Item, cp currency.Pair) (string, error) {
	_, err := p.CurrencyPairs.IsPairEnabled(cp, a)
	if err != nil {
		return "", err
	}
	switch a {
	case asset.Spot:
		cp.Delimiter = currency.UnderscoreDelimiter
		return poloniexAPIURL + tradeSpot + cp.Upper().String(), nil
	case asset.Futures:
		cp.Delimiter = ""
		return poloniexAPIURL + tradeFutures + cp.Upper().String(), nil
	default:
		return "", fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}
}
