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
	ku.SetupDefaults(exchCfg)
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
	ku.Verbose = true
	ku.API.CredentialsValidator.RequiresKey = true
	ku.API.CredentialsValidator.RequiresSecret = true

	spot := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}

	margin := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	}

	err := ku.StoreAssetPairFormat(asset.Spot, spot)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}
	err = ku.StoreAssetPairFormat(asset.Margin, margin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	ku.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WebsocketCapabilities: protocol.Features{
				TickerFetching:    true,
				OrderbookFetching: true,
			},
			WithdrawPermissions: exchange.AutoWithdrawCrypto |
				exchange.AutoWithdrawFiat,
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
					kline.SixHour.Word():    true,
					kline.EightHour.Word():  true,
					kline.TwelveHour.Word(): true,
					kline.OneDay.Word():     true,
					kline.OneWeek.Word():    true,
				},
			},
		},
	}
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	ku.Requester, err = request.New(ku.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	ku.API.Endpoints = ku.NewEndpoints()
	ku.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot:      kucoinAPIURL,
		exchange.RestFutures:   kucoinFuturesAPIURL,
		exchange.WebsocketSpot: kucoinWebsocketURL,
	})
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
		})
	if err != nil {
		return err
	}
	ku.Websocket.Conn = &stream.WebsocketConnection{
		ExchangeName: ku.Name,
		URL:          ku.Websocket.GetWebsocketURL(),
		ProxyURL:     ku.Websocket.GetProxyAddress(),
		Verbose:      ku.Verbose,
		// ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		ResponseMaxLimit: exch.WebsocketResponseMaxLimit,
	}
	return nil
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
func (ku *Kucoin) FetchTradablePairs(ctx context.Context, assetType asset.Item) ([]string, error) {
	if assetType.IsFutures() {
		myPairs, err := ku.GetFuturesOpenContracts(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make([]string, len(myPairs))
		for x := range myPairs {
			pairs[x] = strings.ToUpper(myPairs[x].Symbol)
		}
		return pairs, nil
	}
	myPairs, err := ku.GetSymbols(ctx, "")
	if err != nil {
		return nil, err
	}
	pairs := make([]string, len(myPairs))
	for x := range myPairs {
		pairs[x] = strings.ToUpper(myPairs[x].Name)
	}
	return pairs, nil
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (ku *Kucoin) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	pairs, err := ku.FetchTradablePairs(ctx, asset.Spot)
	if err != nil {
		return err
	}
	p, err := currency.NewPairsFromStrings(pairs)
	if err != nil {
		return err
	}
	return ku.UpdatePairs(p, asset.Spot, false, forceUpdate)
}

// UpdateTicker updates and returns the ticker for a currency pair
func (ku *Kucoin) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	if err := ku.UpdateTickers(ctx, assetType); err != nil {
		return nil, err
	}
	return ticker.GetTicker(ku.Name, fPair, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (ku *Kucoin) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	if assetType.IsFutures() {
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
				Pair:         pair,
				ExchangeName: ku.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
		return nil
	}
	ticks, err := ku.GetAllTickers(ctx)
	if err != nil {
		return err
	}
	for t := range ticks {
		pair, err := currency.NewPairFromString(ticks[t].Symbol)
		if err != nil {
			return err
		}
		err = ticker.ProcessTicker(&ticker.Price{
			Last:         ticks[t].Last,
			High:         ticks[t].High,
			Low:          ticks[t].Low,
			Volume:       ticks[t].Volume,
			Pair:         pair,
			ExchangeName: ku.Name,
			AssetType:    assetType,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (ku *Kucoin) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	fPair, err := ku.FormatExchangeCurrency(p, assetType)
	if err != nil {
		return nil, err
	}
	tickerNew, err := ticker.GetTicker(ku.Name, fPair, assetType)
	if err != nil {
		return ku.UpdateTicker(ctx, fPair, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (ku *Kucoin) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	fPair, err := ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return nil, err
	}

	ob, err := orderbook.Get(ku.Name, fPair, assetType)
	if err != nil {
		return ku.UpdateOrderbook(ctx, fPair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (ku *Kucoin) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        ku.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: ku.CanVerifyOrderbook,
	}
	if !ku.SupportsAsset(assetType) {
		return book, asset.ErrNotSupported
	}
	fPair, err := ku.FormatExchangeCurrency(pair, assetType)
	if err != nil {
		return book, err
	}
	var ordBook *Orderbook
	if assetType.IsFutures() {
		ordBook, err = ku.GetPartOrderbook100(ctx, fPair.String())
	} else {
		fPair.Delimiter = currency.DashDelimiter
		ordBook, err = ku.GetPartOrderbook100(ctx, fPair.String())
	}
	if err != nil {
		return book, err
	}
	book.Asks = ordBook.Asks
	book.Bids = ordBook.Bids
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
	if !ku.SupportsAsset(assetType) {
		return holding, asset.ErrNotSupported
	}
	if assetType.IsFutures() {
		accoutH, err := ku.GetFuturesAccountOverview(ctx, "")
		if err != nil {
			return account.Holdings{}, err
		}
		holding.Accounts = append(holding.Accounts, account.SubAccount{
			AssetType: assetType,
			Currencies: []account.Balance{{
				CurrencyName: currency.NewCode(accoutH.Currency),
				Total:        accoutH.AvailableBalance + accoutH.MarginBalance,
				Hold:         accoutH.FrozenFunds,
				Free:         accoutH.AvailableBalance,
			}},
		})
	}
	accountH, err := ku.GetMarginAccount(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	for x := range accountH.Accounts {
		holding.Accounts = append(holding.Accounts, account.SubAccount{
			AssetType: assetType,
			Currencies: []account.Balance{
				{
					CurrencyName: currency.NewCode(accountH.Accounts[x].Currency),
					Total:        accountH.Accounts[x].TotalBalance,
					Hold:         accountH.Accounts[x].HoldBalance,
					Free:         accountH.Accounts[x].TotalBalance - accountH.Accounts[x].HoldBalance,
				}},
		})
	}
	return holding, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (ku *Kucoin) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc, err := account.GetHoldings(ku.Name, assetType)
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
	fundingData := make([]exchange.FundHistory, 0, len(withdrawalsData)+len(depositsData))
	for x := range depositsData {
		fundingData = append(fundingData, exchange.FundHistory{
			// Fee: depositsData[x].
			Timestamp:    depositsData[x].CreatedAt.Time(),
			ExchangeName: ku.Name,
			CryptoTxID:   depositsData[x].WalletTxID,
			Status:       depositsData[x].Status,
			Amount:       depositsData[x].Amount,
			Currency:     depositsData[x].Currency,
		})
	}

	for x := range withdrawalsData {
		fundingData = append(fundingData, exchange.FundHistory{
			Fee:             withdrawalsData[x].Fee,
			Timestamp:       withdrawalsData[x].UpdatedAt.Time(),
			ExchangeName:    ku.Name,
			CryptoToAddress: withdrawalsData[x].Address,
			CryptoTxID:      withdrawalsData[x].WalletTxID,
			Status:          withdrawalsData[x].Status,
			Amount:          withdrawalsData[x].Amount,
			Currency:        withdrawalsData[x].Currency,
			TransferID:      withdrawalsData[x].ID,
		})
	}
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (ku *Kucoin) GetWithdrawalsHistory(ctx context.Context, c currency.Code) (resp []exchange.WithdrawalHistory, err error) {
	withdrawals, err := ku.GetHistoricalWithdrawalList(ctx, c.String(), "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		return
	}
	for x := range withdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:     withdrawals[x].Status,
			CryptoTxID: withdrawals[x].WalletTxID,
			Timestamp:  withdrawals[x].CreatedAt.Time(),
			Amount:     withdrawals[x].Amount,
			// Fee: withdrawals[x].Fee,
			Currency: c.String(),
		})
	}
	futuresWithdrawals, err := ku.GetFuturesWithdrawalList(ctx, c.String(), "", time.Time{}, time.Time{})
	if err != nil {
		return
	}
	for y := range futuresWithdrawals {
		resp = append(resp, exchange.WithdrawalHistory{
			Status:     futuresWithdrawals[y].Status,
			CryptoTxID: futuresWithdrawals[y].WalletTxID,
			Timestamp:  futuresWithdrawals[y].CreatedAt.Time(),
			Amount:     futuresWithdrawals[y].Amount,
			Currency:   c.String(),
		})
	}
	return
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (ku *Kucoin) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	var resp []trade.Data
	const limit = 1000
	if assetType.IsFutures() {
		tradeData, err := ku.GetFuturesTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			resp = append(resp, trade.Data{
				TID:          tradeData[i].TradeID,
				Exchange:     ku.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].FilledTime.Time(),
			})
		}
	} else {
		p.Delimiter = currency.DashDelimiter
		tradeData, err := ku.GetTradeHistory(ctx, p.String())
		if err != nil {
			return nil, err
		}
		for i := range tradeData {
			resp = append(resp, trade.Data{
				TID:          tradeData[i].Sequence,
				Exchange:     ku.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Price:        tradeData[i].Price,
				Amount:       tradeData[i].Size,
				Timestamp:    tradeData[i].Time.Time(),
			})
		}
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
func (ku *Kucoin) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return ku.GetRecentTrades(ctx, p, assetType)
}

// SubmitOrder submits a new order
func (ku *Kucoin) SubmitOrder(ctx context.Context, s *order.Submit) (order.SubmitResponse, error) {
	var submitOrderResponse order.SubmitResponse
	if err := s.Validate(); err != nil {
		return submitOrderResponse, err
	}
	if s.AssetType.IsFutures() {
		o, err := ku.PostFuturesOrder(ctx, s.ClientOrderID, s.Side.String(), s.Pair.String(), s.Type.Lower(), "", "", "", "", "", s.Amount, s.Price, s.Leverage, 0, s.ReduceOnly, false, false, s.PostOnly, s.HiddenOrder, false)
		if err != nil {
			return submitOrderResponse, err
		}
		return order.SubmitResponse{OrderID: o}, nil
	}
	o, err := ku.PostOrder(ctx, s.ClientOrderID, s.Side.Lower(), s.Pair.Upper().String(), s.Type.Lower(), "", "", "", s.Amount, s.Price, 0, 0, 0, s.PostOnly, s.HiddenOrder, false)
	if err != nil {
		return submitOrderResponse, err
	}
	return order.SubmitResponse{
		OrderID: o,
	}, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (ku *Kucoin) ModifyOrder(ctx context.Context, action *order.Modify) (order.Modify, error) {
	return order.Modify{}, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (ku *Kucoin) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (ku *Kucoin) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (order.CancelBatchResponse, error) {
	return order.CancelBatchResponse{}, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (ku *Kucoin) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (ku *Kucoin) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (order.Detail, error) {
	if assetType.IsFutures() {
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
		// status , err := order.StringToOrderStatus(orderDetail.)
		side, err := order.StringToOrderSide(orderDetail.Side)
		if err != nil {
			return order.Detail{}, err
		}
		if !pair.IsEmpty() && !nPair.Equal(pair) {
			return order.Detail{}, fmt.Errorf("order with id %s and currency Pair %v does not exist", orderID, pair)
		}
		return order.Detail{
			Exchange:        ku.Name,
			ID:              orderDetail.ID,
			Pair:            pair,
			Type:            oType,
			Side:            side,
			Fee:             orderDetail.Fee,
			AssetType:       assetType,
			ExecutedAmount:  orderDetail.DealSize,
			RemainingAmount: orderDetail.Size - orderDetail.DealSize,
			Amount:          orderDetail.Size,
			Price:           orderDetail.Price,
			Date:            time.Time(orderDetail.CreatedAt),
		}, nil
	}
	// TODO: for Spot order goes here.
	return order.Detail{}, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (ku *Kucoin) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	ad, err := ku.GetDepositAddressV2(ctx, c.Upper().String())
	if err != nil {
		return nil, err
	}
	if len(ad) > 1 {
		return nil, errors.New("multiple deposit addresses")
	} else if len(ad) == 0 {
		return nil, errors.New("no deposit address found")
	}
	return &deposit.Address{
		Address: ad[0].Address,
		Chain:   ad[0].Chain,
		Tag:     ad[0].Memo,
	}, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
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
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (ku *Kucoin) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (ku *Kucoin) GetActiveOrders(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	format, err := ku.GetPairFormat(getOrdersRequest.AssetType, true)
	if err != nil {
		return nil, err
	}
	pair := ""
	var orders []order.Detail
	if getOrdersRequest.AssetType.IsFutures() {
		if len(getOrdersRequest.Pairs) == 1 {
			pair = format.Format(getOrdersRequest.Pairs[0])
		}
		futuresOrders, err := ku.GetFuturesOrders(ctx, "", pair, getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), getOrdersRequest.StartTime, getOrdersRequest.EndTime)
		if err != nil {
			return nil, err
		}
		for x := range futuresOrders {
			side, err := order.StringToOrderSide(futuresOrders[x].Side)
			if err != nil {
				return nil, err
			}
			oType, err := order.StringToOrderType(futuresOrders[x].OrderType)
			if err != nil {
				return nil, err
			}
			dPair, err := currency.NewPairFromString(futuresOrders[x].Symbol)
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
			orders = append(orders, order.Detail{
				ID:              futuresOrders[x].ID,
				Amount:          futuresOrders[x].Size,
				RemainingAmount: futuresOrders[x].Size - futuresOrders[x].FilledSize,
				ExecutedAmount:  futuresOrders[x].FilledSize,
				Exchange:        ku.Name,
				Date:            futuresOrders[x].UpdatedAt.Time(),
				Price:           futuresOrders[x].Price,
				Side:            side,
				Type:            oType,
				Pair:            dPair,
			})
		}
	}
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (ku *Kucoin) GetOrderHistory(ctx context.Context, getOrdersRequest *order.GetOrdersRequest) ([]order.Detail, error) {
	if err := getOrdersRequest.Validate(); err != nil {
		return nil, err
	}
	responseOrders, err := ku.GetOrders(ctx, "", "", getOrdersRequest.Side.Lower(), getOrdersRequest.Type.Lower(), "", getOrdersRequest.StartTime, getOrdersRequest.EndTime)
	if err != nil {
		return nil, err
	}
	orders := make([]order.Detail, len(responseOrders))
	for i := range orders {
		orderSide, err := order.StringToOrderSide(responseOrders[i].Side)
		if err != nil {
			return nil, err
		}
		var orderStatus order.Status
		pair, err := currency.NewPairFromString(responseOrders[i].Symbol)
		if err != nil {
			return nil, err
		}
		pair.Delimiter = currency.DashDelimiter
		var oType order.Type
		oType, err = order.StringToOrderType(responseOrders[i].Type)
		if err != nil {
			log.Errorf(log.ExchangeSys, "%s %v", ku.Name, err)
		}
		orderDetail := order.Detail{
			Price:           responseOrders[i].Price,
			Amount:          responseOrders[i].Size,
			ExecutedAmount:  responseOrders[i].DealSize,
			RemainingAmount: responseOrders[i].Size - responseOrders[i].DealSize,
			Date:            responseOrders[i].CreatedAt.Time(),
			Exchange:        ku.Name,
			ID:              responseOrders[i].ID,
			Side:            orderSide,
			Status:          orderStatus,
			Type:            oType,
			Pair:            pair,
		}
		orderDetail.InferCostsAndTimes()
		orders[i] = orderDetail
	}
	order.FilterOrdersBySide(&orders, getOrdersRequest.Side)
	order.FilterOrdersByType(&orders, getOrdersRequest.Type)
	err = order.FilterOrdersByTimeRange(&orders, getOrdersRequest.StartTime, getOrdersRequest.EndTime)
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %v", ku.Name, err)
	}
	order.FilterOrdersByPairs(&orders, getOrdersRequest.Pairs)
	return orders, nil
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
	_, err := ku.UpdateAccountInfo(ctx, assetType)
	return ku.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (ku *Kucoin) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := ku.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	pair.Delimiter = currency.DashDelimiter
	intervalString, err := ku.intervalToString(interval)
	candles, err := ku.GetKlines(ctx, pair.String(), intervalString, start, end)
	if err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: ku.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
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
	return ret, nil
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (ku *Kucoin) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, interval kline.Interval) (kline.Item, error) {
	if err := ku.ValidateKline(pair, a, interval); err != nil {
		return kline.Item{}, err
	}
	ret := kline.Item{
		Exchange: ku.Name,
		Pair:     pair,
		Asset:    a,
		Interval: interval,
	}
	dates, err := kline.CalculateCandleDateRanges(start, end, interval, ku.Features.Enabled.Kline.ResultLimit)
	if err != nil {
		return kline.Item{}, err
	}
	pair.Delimiter = currency.DashDelimiter
	for x := range dates.Ranges {
		intervalString, err := ku.intervalToString(interval)
		candles, err := ku.GetKlines(ctx, pair.String(), intervalString, dates.Ranges[x].Start.Time, dates.Ranges[x].End.Time)
		if err != nil {
			return kline.Item{}, err
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
	}
	dates.SetHasDataFromCandles(ret.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.ExchangeSys, "%v - %v", ku.Name, summary)
	}
	ret.RemoveDuplicates()
	ret.RemoveOutsideRange(start, end)
	ret.SortCandlesByTimestamp(false)
	return ret, nil
}
