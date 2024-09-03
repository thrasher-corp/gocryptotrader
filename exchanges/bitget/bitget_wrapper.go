package bitget

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/currencystate"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// GetDefaultConfig returns a default exchange config
func (bi *Bitget) GetDefaultConfig(ctx context.Context) (*config.Exchange, error) {
	bi.SetDefaults()
	exchCfg := new(config.Exchange)
	exchCfg.Name = bi.Name
	exchCfg.HTTPTimeout = exchange.DefaultHTTPTimeout
	exchCfg.BaseCurrencies = bi.BaseCurrencies

	bi.SetupDefaults(exchCfg)

	if bi.Features.Supports.RESTCapabilities.AutoPairUpdates {
		err := bi.UpdateTradablePairs(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	return exchCfg, nil
}

// SetDefaults sets the basic defaults for Bitget
func (bi *Bitget) SetDefaults() {
	bi.Name = "Bitget"
	bi.Enabled = true
	bi.Verbose = true
	bi.API.CredentialsValidator.RequiresKey = true
	bi.API.CredentialsValidator.RequiresSecret = true
	bi.API.CredentialsValidator.RequiresClientID = true

	// If using only one pair format for request and configuration, across all
	// supported asset types either SPOT and FUTURES etc. You can use the
	// example below:

	// Request format denotes what the pair as a string will be, when you send
	// a request to an exchange.
	requestFmt := &currency.PairFormat{Uppercase: true}
	// Config format denotes what the pair as a string will be, when saved to
	// the config.json file.
	configFmt := &currency.PairFormat{Uppercase: true, Delimiter: "-"}
	err := bi.SetGlobalPairsManager(requestFmt, configFmt, asset.Spot, asset.Futures, asset.Margin, asset.CrossMargin)
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// Fill out the capabilities/features that the exchange supports
	bi.Features = exchange.Features{
		Supports: exchange.FeaturesSupported{
			REST:      true,
			Websocket: true,
			RESTCapabilities: protocol.Features{
				TickerFetching:                 true,
				OrderbookFetching:              true,
				HasAssetTypeAccountSegregation: true,
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
				Intervals: kline.DeployExchangeIntervals(
					kline.IntervalCapacity{Interval: kline.OneMin},
					kline.IntervalCapacity{Interval: kline.ThreeMin},
					kline.IntervalCapacity{Interval: kline.FiveMin},
					kline.IntervalCapacity{Interval: kline.FifteenMin},
					kline.IntervalCapacity{Interval: kline.ThirtyMin},
					kline.IntervalCapacity{Interval: kline.OneHour},
					kline.IntervalCapacity{Interval: kline.FourHour},
					kline.IntervalCapacity{Interval: kline.SixHour},
					kline.IntervalCapacity{Interval: kline.TwelveHour},
					kline.IntervalCapacity{Interval: kline.OneDay},
					kline.IntervalCapacity{Interval: kline.ThreeDay},
					kline.IntervalCapacity{Interval: kline.OneWeek},
					kline.IntervalCapacity{Interval: kline.OneMonth},
				),
				GlobalResultLimit: 200,
			},
		},
	}
	// NOTE: SET THE EXCHANGES RATE LIMIT HERE
	bi.Requester, err = request.New(bi.Name,
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
		request.WithLimiter(GetRateLimits()))
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
	}

	// NOTE: SET THE URLs HERE
	bi.API.Endpoints = bi.NewEndpoints()
	bi.API.Endpoints.SetDefaultEndpoints(map[exchange.URL]string{
		exchange.RestSpot: bitgetAPIURL,
		// exchange.WebsocketSpot: bitgetWSAPIURL,
	})
	bi.Websocket = stream.NewWebsocket()
	bi.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	bi.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	bi.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (bi *Bitget) Setup(exch *config.Exchange) error {
	err := exch.Validate()
	if err != nil {
		return err
	}
	if !exch.Enabled {
		bi.SetEnabled(false)
		return nil
	}
	err = bi.SetupDefaults(exch)
	if err != nil {
		return err
	}

	/*
		wsRunningEndpoint, err := bi.API.Endpoints.GetURL(exchange.WebsocketSpot)
		if err != nil {
			return err
		}

		// If websocket is supported, please fill out the following

		err = bi.Websocket.Setup(
			&stream.WebsocketSetup{
				ExchangeConfig:  exch,
				DefaultURL:      bitgetWSAPIURL,
				RunningURL:      wsRunningEndpoint,
				Connector:       bi.WsConnect,
				Subscriber:      bi.Subscribe,
				UnSubscriber:    bi.Unsubscribe,
				Features:        &bi.Features.Supports.WebsocketCapabilities,
			})
		if err != nil {
			return err
		}

		bi.WebsocketConn = &stream.WebsocketConnection{
			ExchangeName:         bi.Name,
			URL:                  bi.Websocket.GetWebsocketURL(),
			ProxyURL:             bi.Websocket.GetProxyAddress(),
			Verbose:              bi.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
	*/
	return nil
}

// FetchTradablePairs returns a list of the exchanges tradable pairs
func (bi *Bitget) FetchTradablePairs(ctx context.Context, a asset.Item) (currency.Pairs, error) {
	switch a {
	case asset.Spot:
		resp, err := bi.GetSymbolInfo(ctx, "")
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(resp.Data))
		for x := range resp.Data {
			pair, err := currency.NewPairFromString(resp.Data[x].BaseCoin + "-" + resp.Data[x].QuoteCoin)
			if err != nil {
				return nil, err
			}
			pairs[x] = pair
		}
		return pairs, nil
	case asset.Futures:
		resp := new(FutureTickerResp)
		req := []string{"USDT-FUTURES", "COIN-FUTURES", "USDC-FUTURES"}
		for x := range req {
			resp2, err := bi.GetAllFuturesTickers(ctx, req[x])
			if err != nil {
				return nil, err
			}
			resp.Data = append(resp.Data, resp2.Data...)
		}
		pairs := make(currency.Pairs, len(resp.Data))
		for x := range resp.Data {
			pair, err := pairFromStringHelper(resp.Data[x].Symbol)
			if err != nil {
				return nil, err
			}
			pairs[x] = pair
		}
		return pairs, nil
	case asset.Margin, asset.CrossMargin:
		resp, err := bi.GetSupportedCurrencies(ctx)
		if err != nil {
			return nil, err
		}
		pairs := make(currency.Pairs, len(resp.Data))
		for x := range resp.Data {
			pair, err := currency.NewPairFromString(resp.Data[x].BaseCoin + "-" + resp.Data[x].QuoteCoin)
			if err != nil {
				return nil, err
			}
			pairs[x] = pair
		}
		return pairs, nil
	}
	return nil, asset.ErrNotSupported
}

// UpdateTradablePairs updates the exchanges available pairs and stores
// them in the exchanges config
func (bi *Bitget) UpdateTradablePairs(ctx context.Context, forceUpdate bool) error {
	assetTypes := bi.GetAssetTypes(true)
	for x := range assetTypes {
		pairs, err := bi.FetchTradablePairs(ctx, assetTypes[x])
		if err != nil {
			return err
		}
		err = bi.UpdatePairs(pairs, assetTypes[x], false, forceUpdate)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateTicker updates and returns the ticker for a currency pair
func (bi *Bitget) UpdateTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerPrice := new(ticker.Price)
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		tick, err := bi.GetSpotTickerInformation(ctx, p.String())
		if err != nil {
			return nil, err
		}
		tickerPrice = &ticker.Price{
			High:        tick.Data[0].High24H,
			Low:         tick.Data[0].Low24H,
			Bid:         tick.Data[0].BidPrice,
			Ask:         tick.Data[0].AskPrice,
			Volume:      tick.Data[0].BaseVolume,
			QuoteVolume: tick.Data[0].QuoteVolume,
			Open:        tick.Data[0].Open,
			Close:       tick.Data[0].LastPrice,
			LastUpdated: tick.Data[0].Timestamp.Time(),
		}
	case asset.Futures:
		tick, err := bi.GetFuturesTicker(ctx, p.String(), getProductType(p))
		if err != nil {
			return nil, err
		}
		tickerPrice = &ticker.Price{
			High:        tick.Data[0].High24H,
			Low:         tick.Data[0].Low24H,
			Bid:         tick.Data[0].BidPrice,
			Ask:         tick.Data[0].AskPrice,
			Volume:      tick.Data[0].BaseVolume,
			QuoteVolume: tick.Data[0].QuoteVolume,
			Open:        tick.Data[0].Open24H,
			Close:       tick.Data[0].LastPrice,
			IndexPrice:  tick.Data[0].IndexPrice,
			LastUpdated: tick.Data[0].Timestamp.Time(),
		}
	default:
		return nil, asset.ErrNotSupported
	}
	tickerPrice.Pair = p
	tickerPrice.ExchangeName = bi.Name
	tickerPrice.AssetType = assetType
	err := ticker.ProcessTicker(tickerPrice)
	if err != nil {
		return tickerPrice, err
	}
	return ticker.GetTicker(bi.Name, p, assetType)
}

// UpdateTickers updates all currency pairs of a given asset type
func (bi *Bitget) UpdateTickers(ctx context.Context, assetType asset.Item) error {
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		tick, err := bi.GetSpotTickerInformation(ctx, "")
		if err != nil {
			return err
		}
		for x := range tick.Data {
			p, err := bi.MatchSymbolWithAvailablePairs(tick.Data[x].Symbol, assetType, false)
			if err != nil {
				return err
			}
			err = ticker.ProcessTicker(&ticker.Price{
				High:         tick.Data[x].High24H,
				Low:          tick.Data[x].Low24H,
				Bid:          tick.Data[x].BidPrice,
				Ask:          tick.Data[x].AskPrice,
				Volume:       tick.Data[x].BaseVolume,
				QuoteVolume:  tick.Data[x].QuoteVolume,
				Open:         tick.Data[x].Open,
				Close:        tick.Data[x].LastPrice,
				LastUpdated:  tick.Data[x].Timestamp.Time(),
				Pair:         p,
				ExchangeName: bi.Name,
				AssetType:    assetType,
			})
			if err != nil {
				return err
			}
		}
	case asset.Futures:
		for i := range prodTypes {
			tick, err := bi.GetAllFuturesTickers(ctx, prodTypes[i])
			if err != nil {
				return err
			}
			for x := range tick.Data {
				p, err := bi.MatchSymbolWithAvailablePairs(tick.Data[x].Symbol, assetType, false)
				if err != nil {
					return err
				}
				err = ticker.ProcessTicker(&ticker.Price{
					High:         tick.Data[x].High24H,
					Low:          tick.Data[x].Low24H,
					Bid:          tick.Data[x].BidPrice,
					Ask:          tick.Data[x].AskPrice,
					Volume:       tick.Data[x].BaseVolume,
					QuoteVolume:  tick.Data[x].QuoteVolume,
					Open:         tick.Data[x].Open24H,
					Close:        tick.Data[x].LastPrice,
					IndexPrice:   tick.Data[x].IndexPrice,
					LastUpdated:  tick.Data[x].Timestamp.Time(),
					Pair:         p,
					ExchangeName: bi.Name,
					AssetType:    assetType,
				})
				if err != nil {
					return err
				}
			}
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// FetchTicker returns the ticker for a currency pair
func (bi *Bitget) FetchTicker(ctx context.Context, p currency.Pair, assetType asset.Item) (*ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(bi.Name, p, assetType)
	if err != nil {
		return bi.UpdateTicker(ctx, p, assetType)
	}
	return tickerNew, nil
}

// FetchOrderbook returns orderbook base on the currency pair
func (bi *Bitget) FetchOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	ob, err := orderbook.Get(bi.Name, pair, assetType)
	if err != nil {
		return bi.UpdateOrderbook(ctx, pair, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (bi *Bitget) UpdateOrderbook(ctx context.Context, pair currency.Pair, assetType asset.Item) (*orderbook.Base, error) {
	book := &orderbook.Base{
		Exchange:        bi.Name,
		Pair:            pair,
		Asset:           assetType,
		VerifyOrderbook: bi.CanVerifyOrderbook,
		MaxDepth:        150,
	}
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		orderbookNew, err := bi.GetOrderbookDepth(ctx, pair.String(), "", 150)
		if err != nil {
			return book, err
		}
		book.Bids = make([]orderbook.Tranche, len(orderbookNew.Data.Bids))
		for x := range orderbookNew.Data.Bids {
			book.Bids[x].Amount = orderbookNew.Data.Bids[x][1].Float64()
			book.Bids[x].Price = orderbookNew.Data.Bids[x][0].Float64()
		}
		book.Asks = make([]orderbook.Tranche, len(orderbookNew.Data.Asks))
		for x := range orderbookNew.Data.Asks {
			book.Asks[x].Amount = orderbookNew.Data.Asks[x][1].Float64()
			book.Asks[x].Price = orderbookNew.Data.Asks[x][0].Float64()
		}
	case asset.Futures:
		orderbookNew, err := bi.GetFuturesMergeDepth(ctx, pair.String(), getProductType(pair), "", "max")
		if err != nil {
			return book, err
		}
		book.Bids = make([]orderbook.Tranche, len(orderbookNew.Data.Bids))
		for x := range orderbookNew.Data.Bids {
			book.Bids[x].Amount = orderbookNew.Data.Bids[x][1]
			book.Bids[x].Price = orderbookNew.Data.Bids[x][0]
		}
		book.Asks = make([]orderbook.Tranche, len(orderbookNew.Data.Asks))
		for x := range orderbookNew.Data.Asks {
			book.Asks[x].Amount = orderbookNew.Data.Asks[x][1]
			book.Asks[x].Price = orderbookNew.Data.Asks[x][0]
		}
	default:
		return book, asset.ErrNotSupported
	}
	err := book.Process()
	if err != nil {
		return book, err
	}
	return orderbook.Get(bi.Name, pair, assetType)
}

// UpdateAccountInfo retrieves balances for all enabled currencies
func (bi *Bitget) UpdateAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	acc := account.Holdings{
		Exchange: bi.Name,
	}
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return acc, err
	}
	switch assetType {
	case asset.Spot:
		resp, err := bi.GetAccountAssets(ctx, "", "")
		if err != nil {
			return acc, err
		}
		acc.Accounts = make([]account.SubAccount, 1)
		acc.Accounts[0].Currencies = make([]account.Balance, len(resp.Data))
		for x := range resp.Data {
			acc.Accounts[0].Currencies[x].Currency = currency.NewCode(resp.Data[x].Coin)
			acc.Accounts[0].Currencies[x].Hold = resp.Data[x].Frozen + resp.Data[x].Locked +
				resp.Data[x].LimitAvailable
			acc.Accounts[0].Currencies[x].Total = resp.Data[x].Available + acc.Accounts[0].Currencies[x].Hold
			acc.Accounts[0].Currencies[x].Free = resp.Data[x].Available
		}
	case asset.Futures:
		acc.Accounts = make([]account.SubAccount, len(prodTypes))
		for i := range prodTypes {
			resp, err := bi.GetAllFuturesAccounts(ctx, prodTypes[i])
			if err != nil {
				return acc, err
			}
			acc.Accounts[i].Currencies = make([]account.Balance, len(resp.Data))
			for x := range resp.Data {
				acc.Accounts[i].Currencies[x].Currency = currency.NewCode(resp.Data[x].MarginCoin)
				acc.Accounts[i].Currencies[x].Hold = resp.Data[x].Locked
				acc.Accounts[i].Currencies[x].Total = resp.Data[x].Locked + resp.Data[x].Available
				acc.Accounts[i].Currencies[x].Free = resp.Data[x].Available
			}
		}
	case asset.Margin:
		resp, err := bi.GetIsolatedAccountAssets(ctx, "")
		if err != nil {
			return acc, err
		}
		acc.Accounts = make([]account.SubAccount, 1)
		acc.Accounts[0].Currencies = make([]account.Balance, len(resp.Data))
		for x := range resp.Data {
			acc.Accounts[0].Currencies[x].Currency = currency.NewCode(resp.Data[x].Coin)
			acc.Accounts[0].Currencies[x].Hold = resp.Data[x].Frozen
			acc.Accounts[0].Currencies[x].Total = resp.Data[x].TotalAmount
			acc.Accounts[0].Currencies[x].Free = resp.Data[x].Available
			acc.Accounts[0].Currencies[x].Borrowed = resp.Data[x].Borrow
		}
	case asset.CrossMargin:
		resp, err := bi.GetCrossAccountAssets(ctx, "")
		if err != nil {
			return acc, err
		}
		acc.Accounts = make([]account.SubAccount, 1)
		acc.Accounts[0].Currencies = make([]account.Balance, len(resp.Data))
		for x := range resp.Data {
			acc.Accounts[0].Currencies[x].Currency = currency.NewCode(resp.Data[x].Coin)
			acc.Accounts[0].Currencies[x].Hold = resp.Data[x].Frozen
			acc.Accounts[0].Currencies[x].Total = resp.Data[x].TotalAmount
			acc.Accounts[0].Currencies[x].Free = resp.Data[x].Available
			acc.Accounts[0].Currencies[x].Borrowed = resp.Data[x].Borrow
		}
	default:
		return acc, asset.ErrNotSupported
	}
	ID, err := bi.GetAccountInfo(ctx)
	if err != nil {
		return acc, err
	}
	for x := range acc.Accounts {
		acc.Accounts[x].ID = strconv.FormatInt(ID.Data.UserID, 10)
		acc.Accounts[x].AssetType = assetType
	}
	err = account.Process(&acc, creds)
	if err != nil {
		return acc, err
	}
	return acc, nil
}

// FetchAccountInfo retrieves balances for all enabled currencies
func (bi *Bitget) FetchAccountInfo(ctx context.Context, assetType asset.Item) (account.Holdings, error) {
	creds, err := bi.GetCredentials(ctx)
	if err != nil {
		return account.Holdings{}, err
	}
	acc, err := account.GetHoldings(bi.Name, creds, assetType)
	if err != nil {
		return bi.UpdateAccountInfo(ctx, assetType)
	}
	return acc, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (bi *Bitget) GetAccountFundingHistory(ctx context.Context) ([]exchange.FundingHistory, error) {
	// This exchange only allows requests covering the last 90 days
	resp, err := bi.withdrawalHistGrabber(ctx, "")
	if err != nil {
		return nil, err
	}
	funHist := make([]exchange.FundingHistory, len(resp.Data))
	for x := range resp.Data {
		funHist[x] = exchange.FundingHistory{
			ExchangeName:      bi.Name,
			Status:            resp.Data[x].Status,
			TransferID:        strconv.FormatInt(resp.Data[x].OrderID, 10),
			Timestamp:         resp.Data[x].CreationTime.Time(),
			Currency:          resp.Data[x].Coin,
			Amount:            resp.Data[x].Size,
			TransferType:      "Withdrawal",
			CryptoToAddress:   resp.Data[x].ToAddress,
			CryptoFromAddress: resp.Data[x].FromAddress,
			CryptoChain:       resp.Data[x].Chain,
		}
		if resp.Data[x].Destination == "on_chain" {
			funHist[x].CryptoTxID = strconv.FormatInt(resp.Data[x].TradeID, 10)
		}
	}
	var pagination int64
	pagination = 0
	for {
		resp, err := bi.GetDepositRecords(ctx, "", 0, pagination, 100, time.Now().Add(-time.Hour*24*90), time.Now())
		if err != nil {
			return nil, err
		}
		// Not sure that this is the right end to use for pagination
		if resp == nil || len(resp.Data) == 0 || pagination == resp.Data[len(resp.Data)-1].OrderID {
			break
		}
		pagination = resp.Data[len(resp.Data)-1].OrderID
		tempHist := make([]exchange.FundingHistory, len(resp.Data))
		for x := range resp.Data {
			tempHist[x] = exchange.FundingHistory{
				ExchangeName:      bi.Name,
				Status:            resp.Data[x].Status,
				TransferID:        strconv.FormatInt(resp.Data[x].OrderID, 10),
				Timestamp:         resp.Data[x].CreationTime.Time(),
				Currency:          resp.Data[x].Coin,
				Amount:            resp.Data[x].Size,
				TransferType:      "Deposit",
				CryptoToAddress:   resp.Data[x].ToAddress,
				CryptoFromAddress: resp.Data[x].FromAddress,
				CryptoChain:       resp.Data[x].Chain,
			}
			if resp.Data[x].Destination == "on_chain" {
				tempHist[x].CryptoTxID = strconv.FormatInt(resp.Data[x].TradeID, 10)
			}
		}
		funHist = append(funHist, tempHist...)
	}
	return funHist, nil
}

// GetWithdrawalsHistory returns previous withdrawals data
func (bi *Bitget) GetWithdrawalsHistory(ctx context.Context, c currency.Code, _ asset.Item) ([]exchange.WithdrawalHistory, error) {
	// This exchange only allows requests covering the last 90 days
	resp, err := bi.withdrawalHistGrabber(ctx, c.String())
	if err != nil {
		return nil, err
	}
	funHist := make([]exchange.WithdrawalHistory, len(resp.Data))
	for x := range resp.Data {
		funHist[x] = exchange.WithdrawalHistory{
			Status:          resp.Data[x].Status,
			TransferID:      strconv.FormatInt(resp.Data[x].OrderID, 10),
			Timestamp:       resp.Data[x].CreationTime.Time(),
			Currency:        resp.Data[x].Coin,
			Amount:          resp.Data[x].Size,
			TransferType:    "Withdrawal",
			CryptoToAddress: resp.Data[x].ToAddress,
			CryptoChain:     resp.Data[x].Chain,
		}
		if resp.Data[x].Destination == "on_chain" {
			funHist[x].CryptoTxID = strconv.FormatInt(resp.Data[x].TradeID, 10)
		}
	}
	return funHist, nil
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (bi *Bitget) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		resp, err := bi.GetRecentSpotFills(ctx, p.String(), 500)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp.Data))
		for x := range resp.Data {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp.Data[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp.Data[x].Side),
				Price:        resp.Data[x].Price,
				Amount:       resp.Data[x].Size,
				Timestamp:    resp.Data[x].Timestamp.Time(),
			}
		}
		return trades, nil
	case asset.Futures:
		resp, err := bi.GetRecentFuturesFills(ctx, p.String(), getProductType(p), 100)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp.Data))
		for x := range resp.Data {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp.Data[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp.Data[x].Side),
				Price:        resp.Data[x].Price,
				Amount:       resp.Data[x].Size,
				Timestamp:    resp.Data[x].Timestamp.Time(),
			}
		}
		return trades, nil
	}
	return nil, asset.ErrNotSupported
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (bi *Bitget) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	// This exchange only allows requests covering the last 7 days
	switch assetType {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		resp, err := bi.GetSpotMarketTrades(ctx, p.String(), timestampStart, timestampEnd, 1000, 0)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp.Data))
		for x := range resp.Data {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp.Data[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp.Data[x].Side),
				Price:        resp.Data[x].Price,
				Amount:       resp.Data[x].Size,
				Timestamp:    resp.Data[x].Timestamp.Time(),
			}
		}
		return trades, nil
	case asset.Futures:
		resp, err := bi.GetFuturesMarketTrades(ctx, p.String(), getProductType(p), 1000, 0, timestampStart,
			timestampEnd)
		if err != nil {
			return nil, err
		}
		trades := make([]trade.Data, len(resp.Data))
		for x := range resp.Data {
			trades[x] = trade.Data{
				TID:          strconv.FormatInt(resp.Data[x].TradeID, 10),
				Exchange:     bi.Name,
				CurrencyPair: p,
				AssetType:    assetType,
				Side:         sideDecoder(resp.Data[x].Side),
				Price:        resp.Data[x].Price,
				Amount:       resp.Data[x].Size,
				Timestamp:    resp.Data[x].Timestamp.Time(),
			}
		}
		return trades, nil
	}
	return nil, asset.ErrNotSupported
}

// GetServerTime returns the current exchange server time.
func (bi *Bitget) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	resp, err := bi.GetTime(ctx)
	return resp.Data.ServerTime.Time(), err
}

// SubmitOrder submits a new order
func (bi *Bitget) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	var IDs *OrderIDResp
	strat, err := strategyTruthTable(s.ImmediateOrCancel, s.FillOrKill, s.PostOnly)
	if err != nil {
		return nil, err
	}
	cID, err := uuid.DefaultGenerator.NewV4()
	if err != nil {
		return nil, err
	}
	switch s.AssetType {
	case asset.Spot:
		IDs, err = bi.PlaceSpotOrder(ctx, s.Pair.String(), s.Side.String(), s.Type.Lower(), strat, cID.String(),
			s.Price, s.Amount, false)
	case asset.Futures:
		IDs, err = bi.PlaceFuturesOrder(ctx, s.Pair.String(), getProductType(s.Pair), marginStringer(s.MarginType),
			s.Pair.Quote.String(), sideEncoder(s.Side, false), "", s.Type.Lower(), strat, cID.String(), 0, 0,
			s.Amount, s.Price, s.ReduceOnly, false)
	case asset.Margin, asset.CrossMargin:
		loanType := "normal"
		if s.AutoBorrow {
			loanType = "autoLoan"
		}
		if s.AssetType == asset.Margin {
			IDs, err = bi.PlaceIsolatedOrder(ctx, s.Pair.String(), s.Type.Lower(), loanType, strat,
				cID.String(), s.Side.String(), s.Price, s.Amount, s.QuoteAmount)
		} else {
			IDs, err = bi.PlaceCrossOrder(ctx, s.Pair.String(), s.Type.Lower(), loanType, strat, cID.String(),
				s.Side.String(), s.Price, s.Amount, s.QuoteAmount)
		}
	default:
		return nil, asset.ErrNotSupported
	}
	if err != nil {
		return nil, err
	}
	resp, err := s.DeriveSubmitResponse(strconv.FormatInt(IDs.Data.OrderID, 10))
	if err != nil {
		return nil, err
	}
	resp.ClientOrderID = IDs.Data.ClientOrderID
	return resp, nil
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (bi *Bitget) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	err := action.Validate()
	if err != nil {
		return nil, err
	}
	var IDs *OrderIDResp
	originalID, err := strconv.ParseInt(action.OrderID, 10, 64)
	if err != nil {
		return nil, err
	}
	switch action.AssetType {
	case asset.Spot:
		IDs, err = bi.ModifyPlanSpotOrder(ctx, originalID, action.ClientOrderID, action.Type.String(),
			action.TriggerPrice, action.Price, action.Amount)
	case asset.Futures:
		var cID uuid.UUID
		cID, err = uuid.DefaultGenerator.NewV4()
		if err != nil {
			return nil, err
		}
		IDs, err = bi.ModifyFuturesOrder(ctx, originalID, action.ClientOrderID, action.Pair.String(),
			getProductType(action.Pair), cID.String(), action.Amount, action.Price, 0, 0)
		fmt.Printf("Error: %v\n", err)
	default:
		return nil, asset.ErrNotSupported
	}
	if err != nil {
		return nil, err
	}
	resp, err := action.DeriveModifyResponse()
	if err != nil {
		return nil, err
	}
	resp.OrderID = strconv.FormatInt(IDs.Data.OrderID, 10)
	resp.ClientOrderID = IDs.Data.ClientOrderID
	return resp, nil
}

// CancelOrder cancels an order by its corresponding ID number
func (bi *Bitget) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	err := ord.Validate(ord.StandardCancel())
	if err != nil {
		return err
	}
	originalID, err := strconv.ParseInt(ord.OrderID, 10, 64)
	if err != nil {
		return err
	}
	switch ord.AssetType {
	case asset.Spot:
		_, err = bi.CancelSpotOrderByID(ctx, ord.Pair.String(), ord.ClientOrderID, originalID)
	case asset.Futures:
		_, err = bi.CancelFuturesOrder(ctx, ord.Pair.String(), getProductType(ord.Pair), ord.Pair.Quote.String(),
			ord.ClientOrderID, originalID)
	case asset.Margin:
		_, err = bi.CancelIsolatedOrder(ctx, ord.Pair.String(), ord.ClientOrderID, originalID)
	case asset.CrossMargin:
		_, err = bi.CancelCrossOrder(ctx, ord.Pair.String(), ord.ClientOrderID, originalID)
	default:
		return asset.ErrNotSupported
	}
	if err != nil {
		return err
	}
	return nil
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (bi *Bitget) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	batchByAsset := make(map[asset.Item][]order.Cancel)
	for i := range orders {
		batchByAsset[orders[i].AssetType] = append(batchByAsset[orders[i].AssetType], orders[i])
	}
	resp := &order.CancelBatchResponse{}
	resp.Status = make(map[string]string)
	for assetType, batch := range batchByAsset {
		var status *BatchOrderResp
		batchByPair, err := pairBatcher(batch)
		if err != nil {
			return nil, err
		}
		for pair, batch := range batchByPair {
			switch assetType {
			case asset.Spot:
				status, err = bi.BatchCancelOrders(ctx, pair.String(), batch)
			case asset.Futures:
				status, err = bi.BatchCancelFuturesOrders(ctx, batch, pair.String(), getProductType(pair),
					pair.Quote.String())
			case asset.Margin:
				status, err = bi.BatchCancelIsolatedOrders(ctx, pair.String(), batch)
			case asset.CrossMargin:
				status, err = bi.BatchCancelCrossOrders(ctx, pair.String(), batch)
			default:
				return nil, asset.ErrNotSupported
			}
			if err != nil {
				return nil, err
			}
			addStatuses(status, resp)
		}
	}
	return resp, nil
}

// CancelAllOrders cancels all orders associated with a currency pair
func (bi *Bitget) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	var resp order.CancelAllResponse
	err := orderCancellation.Validate()
	if err != nil {
		return resp, err
	}
	switch orderCancellation.AssetType {
	case asset.Spot:
		_, err = bi.CancelOrdersBySymbol(ctx, orderCancellation.Pair.String())
		if err != nil {
			return resp, err
		}
	case asset.Futures:
		resp2, err := bi.CancelAllFuturesOrders(ctx, orderCancellation.Pair.String(),
			getProductType(orderCancellation.Pair), orderCancellation.Pair.Quote.String(), time.Second*60)
		if err != nil {
			return resp, err
		}
		resp.Status = make(map[string]string)
		for i := range resp2.Data.SuccessList {
			resp.Status[resp2.Data.SuccessList[i].ClientOrderID] = "success"
			resp.Status[strconv.FormatInt(int64(resp2.Data.SuccessList[i].OrderID), 10)] = "success"
		}
		for i := range resp2.Data.FailureList {
			resp.Status[resp2.Data.FailureList[i].ClientOrderID] = resp2.Data.FailureList[i].ErrorMessage
			resp.Status[strconv.FormatInt(int64(resp2.Data.FailureList[i].OrderID), 10)] =
				resp2.Data.FailureList[i].ErrorMessage
		}
	default:
		return resp, asset.ErrNotSupported
	}
	return resp, nil
}

// GetOrderInfo returns order information based on order ID
func (bi *Bitget) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	ordID, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, err
	}
	resp := &order.Detail{
		Exchange:  bi.Name,
		Pair:      pair,
		AssetType: assetType,
		OrderID:   orderID,
	}
	switch assetType {
	case asset.Spot:
		ordInfo, err := bi.GetSpotOrderDetails(ctx, ordID, "")
		if err != nil {
			return nil, err
		}
		if len(ordInfo.Data) == 0 {
			return nil, errOrderNotFound
		}
		resp.AccountID = ordInfo.Data[0].UserID
		resp.ClientOrderID = ordInfo.Data[0].ClientOrderID
		resp.Price = ordInfo.Data[0].Price
		resp.Amount = ordInfo.Data[0].Size
		resp.Type = typeDecoder(ordInfo.Data[0].OrderType)
		resp.Side = sideDecoder(ordInfo.Data[0].Side)
		resp.Status = statusDecoder(ordInfo.Data[0].Status)
		resp.AverageExecutedPrice = ordInfo.Data[0].PriceAverage
		resp.QuoteAmount = ordInfo.Data[0].QuoteVolume
		resp.Date = ordInfo.Data[0].CreationTime.Time()
		resp.LastUpdated = ordInfo.Data[0].UpdateTime.Time()
		for s, f := range ordInfo.Data[0].FeeDetail {
			if s != "newFees" {
				resp.FeeAsset = currency.NewCode(f.FeeCoinCode)
				resp.Fee = f.TotalFee
				break
			}
		}
		fillInfo, err := bi.GetSpotFills(ctx, pair.String(), time.Time{}, time.Time{}, 0, 0, ordID)
		if err != nil {
			return nil, err
		}
		resp.Trades = make([]order.TradeHistory, len(fillInfo.Data))
		for x := range fillInfo.Data {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatInt(fillInfo.Data[x].TradeID, 10),
				Type:      typeDecoder(fillInfo.Data[x].OrderType),
				Side:      sideDecoder(fillInfo.Data[x].Side),
				Price:     fillInfo.Data[x].PriceAverage,
				Amount:    fillInfo.Data[x].Size,
				Fee:       fillInfo.Data[x].FeeDetail.TotalFee,
				FeeAsset:  fillInfo.Data[x].FeeDetail.FeeCoin,
				Timestamp: fillInfo.Data[x].CreationTime.Time(),
			}
		}
	case asset.Futures:
		ordInfo, err := bi.GetFuturesOrderDetails(ctx, pair.String(), getProductType(pair), "", ordID)
		if err != nil {
			return nil, err
		}
		resp.Amount = ordInfo.Size
		resp.ClientOrderID = ordInfo.ClientOrderID
		resp.AverageExecutedPrice = ordInfo.PriceAverage
		resp.Fee = ordInfo.Fee.Float64()
		resp.Price = ordInfo.Price
		resp.Status = statusDecoder(ordInfo.State)
		resp.Side = sideDecoder(ordInfo.Side)
		resp.ImmediateOrCancel, resp.FillOrKill, resp.PostOnly = strategyDecoder(ordInfo.Force)
		resp.SettlementCurrency = currency.NewCode(ordInfo.MarginCoin)
		resp.LimitPriceUpper = ordInfo.PresetStopSurplusPrice
		resp.LimitPriceLower = ordInfo.PresetStopLossPrice
		resp.QuoteAmount = ordInfo.QuoteVolume
		resp.Type = typeDecoder(ordInfo.OrderType)
		resp.Leverage = ordInfo.Leverage
		resp.MarginType = marginDecoder(ordInfo.MarginMode)
		resp.ReduceOnly = bool(ordInfo.ReduceOnly)
		resp.Date = ordInfo.CreationTime.Time()
		resp.LastUpdated = ordInfo.UpdateTime.Time()
		fillInfo, err := bi.GetFuturesFills(ctx, ordID, 0, 100, pair.String(), getProductType(pair), time.Time{},
			time.Time{})
		if err != nil {
			return nil, err
		}
		resp.Trades = make([]order.TradeHistory, len(fillInfo.Data.FillList))
		for x := range fillInfo.Data.FillList {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatInt(fillInfo.Data.FillList[x].TradeID, 10),
				Price:     fillInfo.Data.FillList[x].Price,
				Amount:    fillInfo.Data.FillList[x].BaseVolume,
				Side:      sideDecoder(fillInfo.Data.FillList[x].Side),
				Timestamp: fillInfo.Data.FillList[x].CreationTime.Time(),
			}
			for i := range fillInfo.Data.FillList[x].FeeDetail {
				resp.Trades[x].Fee += fillInfo.Data.FillList[x].FeeDetail[i].TotalFee
				resp.Trades[x].FeeAsset = fillInfo.Data.FillList[x].FeeDetail[i].FeeCoin
			}
			if fillInfo.Data.FillList[x].TradeScope == "maker" {
				resp.Trades[x].IsMaker = true
			}
		}
	case asset.Margin, asset.CrossMargin:
		var ordInfo *MarginOpenOrds
		var fillInfo *MarginOrderFills
		if assetType == asset.Margin {
			ordInfo, err = bi.GetIsolatedOpenOrders(ctx, pair.String(), "", ordID, 2, 0, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			fillInfo, err = bi.GetIsolatedOrderFills(ctx, pair.String(), ordID, 0, 500,
				time.Now().Add(-time.Hour*24*90), time.Now())
		} else {
			ordInfo, err = bi.GetCrossOpenOrders(ctx, pair.String(), "", ordID, 2, 0, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			fillInfo, err = bi.GetCrossOrderFills(ctx, pair.String(), ordID, 0, 500,
				time.Now().Add(-time.Hour*24*90), time.Now())
		}
		if err != nil {
			return nil, err
		}
		if len(ordInfo.Data.OrderList) == 0 {
			return nil, errOrderNotFound
		}
		resp.Type = typeDecoder(ordInfo.Data.OrderList[0].OrderType)
		resp.ClientOrderID = ordInfo.Data.OrderList[0].ClientOrderID
		resp.Price = ordInfo.Data.OrderList[0].Price
		resp.Side = sideDecoder(ordInfo.Data.OrderList[0].Side)
		resp.Status = statusDecoder(ordInfo.Data.OrderList[0].Status)
		resp.Amount = ordInfo.Data.OrderList[0].Size
		resp.QuoteAmount = ordInfo.Data.OrderList[0].QuoteSize
		resp.ImmediateOrCancel, resp.FillOrKill, resp.PostOnly = strategyDecoder(ordInfo.Data.OrderList[0].Force)
		resp.Date = ordInfo.Data.OrderList[0].CreationTime.Time()
		resp.LastUpdated = ordInfo.Data.OrderList[0].UpdateTime.Time()
		resp.Trades = make([]order.TradeHistory, len(fillInfo.Data.Fills))
		for x := range fillInfo.Data.Fills {
			resp.Trades[x] = order.TradeHistory{
				TID:       strconv.FormatInt(fillInfo.Data.Fills[x].TradeID, 10),
				Type:      typeDecoder(fillInfo.Data.Fills[x].OrderType),
				Side:      sideDecoder(fillInfo.Data.Fills[x].Side),
				Price:     fillInfo.Data.Fills[x].PriceAverage,
				Amount:    fillInfo.Data.Fills[x].Size,
				Timestamp: fillInfo.Data.Fills[x].CreationTime.Time(),
				Fee:       fillInfo.Data.Fills[x].FeeDetail.TotalFee,
				FeeAsset:  fillInfo.Data.Fills[x].FeeDetail.FeeCoin,
			}
		}
	default:
		return nil, asset.ErrNotSupported
	}
	return resp, nil
}

// GetDepositAddress returns a deposit address for a specified currency
func (bi *Bitget) GetDepositAddress(ctx context.Context, c currency.Code, _ string, chain string) (*deposit.Address, error) {
	resp, err := bi.GetDepositAddressForCurrency(ctx, c.String(), chain)
	if err != nil {
		return nil, err
	}
	add := &deposit.Address{
		Address: resp.Data.Address,
		Chain:   resp.Data.Chain,
		Tag:     resp.Data.Tag,
	}
	return add, nil
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (bi *Bitget) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	err := withdrawRequest.Validate()
	if err != nil {
		return nil, err
	}
	resp, err := bi.WithdrawFunds(ctx, withdrawRequest.Currency.String(), "on_chain",
		withdrawRequest.Crypto.Address, withdrawRequest.Crypto.Chain, "", "", withdrawRequest.Crypto.AddressTag,
		withdrawRequest.Description, "", withdrawRequest.Amount)
	if err != nil {
		return nil, err
	}
	ret := &withdraw.ExchangeResponse{
		ID: strconv.FormatInt(resp.Data.OrderID, 10),
	}
	return ret, nil
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (bi *Bitget) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (bi *Bitget) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// GetActiveOrders retrieves any orders that are active/open
func (bi *Bitget) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	pairs := make([]string, len(getOrdersRequest.Pairs))
	for x := range getOrdersRequest.Pairs {
		pairs[x] = getOrdersRequest.Pairs[x].String()
	}
	if len(pairs) == 0 {
		pairs = append(pairs, "")
	}
	var resp order.FilteredOrders
	for x := range pairs {
		switch getOrdersRequest.AssetType {
		case asset.Spot:
			var pagination int64
			for {
				genOrds, err := bi.GetUnfilledOrders(ctx, pairs[x], time.Time{}, time.Time{}, 100, pagination, 0)
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.Data) == 0 ||
					pagination == int64(genOrds.Data[len(genOrds.Data)-1].OrderID) {
					break
				}
				pagination = int64(genOrds.Data[len(genOrds.Data)-1].OrderID)
				tempOrds := make([]order.Detail, len(genOrds.Data))
				for i := range genOrds.Data {
					tempOrds[i] = order.Detail{
						Exchange:             bi.Name,
						AssetType:            asset.Spot,
						AccountID:            genOrds.Data[i].UserID,
						OrderID:              strconv.FormatInt(int64(genOrds.Data[i].OrderID), 10),
						ClientOrderID:        genOrds.Data[i].ClientOrderID,
						AverageExecutedPrice: genOrds.Data[i].PriceAverage,
						Amount:               genOrds.Data[i].Size,
						Type:                 typeDecoder(genOrds.Data[i].OrderType),
						Side:                 sideDecoder(genOrds.Data[i].Side),
						Status:               statusDecoder(genOrds.Data[i].Status),
						Price:                genOrds.Data[i].BasePrice,
						QuoteAmount:          genOrds.Data[i].QuoteVolume,
						Date:                 genOrds.Data[i].CreationTime.Time(),
						LastUpdated:          genOrds.Data[i].UpdateTime.Time(),
					}
					if pairs[x] != "" {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
				}
				resp = append(resp, tempOrds...)
			}
			if pairs[x] != "" {
				resp, err = bi.spotCurrentPlanOrdersHelper(ctx, pairs[x], getOrdersRequest.Pairs[x], resp)
				if err != nil {
					return nil, err
				}
			} else {
				newPairs, err := bi.FetchTradablePairs(ctx, asset.Spot)
				if err != nil {
					return nil, err
				}
				for y := range newPairs {
					callStr, err := bi.FormatExchangeCurrency(newPairs[y], asset.Spot)
					if err != nil {
						return nil, err
					}
					resp, err = bi.spotCurrentPlanOrdersHelper(ctx, callStr.String(), newPairs[y], resp)
					if err != nil {
						return nil, err
					}
				}

			}
		case asset.Futures:
			if pairs[x] != "" {
				resp, err = bi.activeFuturesOrderHelper(ctx, pairs[x], getProductType(getOrdersRequest.Pairs[x]),
					getOrdersRequest.Pairs[x], resp)
				if err != nil {
					return nil, err
				}
			} else {
				for y := range prodTypes {
					resp, err = bi.activeFuturesOrderHelper(ctx, "", prodTypes[y], currency.Pair{}, resp)
					if err != nil {
						return nil, err
					}
				}
			}
		case asset.Margin, asset.CrossMargin:
			var pagination int64
			var genOrds *MarginOpenOrds
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genOrds, err = bi.GetIsolatedOpenOrders(ctx, pairs[x], "", 0, 500, pagination,
						time.Now().Add(-time.Hour*24*90), time.Time{})
				} else {
					genOrds, err = bi.GetCrossOpenOrders(ctx, pairs[x], "", 0, 500, pagination,
						time.Now().Add(-time.Hour*24*90), time.Time{})
				}
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.Data.OrderList) == 0 || pagination == int64(genOrds.Data.MaxID) {
					break
				}
				pagination = int64(genOrds.Data.MaxID)
				tempOrds := make([]order.Detail, len(genOrds.Data.OrderList))
				for i := range genOrds.Data.OrderList {
					tempOrds[i] = order.Detail{
						Exchange:      bi.Name,
						AssetType:     getOrdersRequest.AssetType,
						OrderID:       strconv.FormatInt(int64(genOrds.Data.OrderList[i].OrderID), 10),
						Type:          typeDecoder(genOrds.Data.OrderList[i].OrderType),
						ClientOrderID: genOrds.Data.OrderList[i].ClientOrderID,
						Price:         genOrds.Data.OrderList[i].Price,
						Side:          sideDecoder(genOrds.Data.OrderList[i].Side),
						Status:        statusDecoder(genOrds.Data.OrderList[i].Status),
						QuoteAmount:   genOrds.Data.OrderList[i].QuoteSize,
						Amount:        genOrds.Data.OrderList[i].Size,
						Date:          genOrds.Data.OrderList[i].CreationTime.Time(),
						LastUpdated:   genOrds.Data.OrderList[i].UpdateTime.Time(),
					}
					if pairs[x] != "" {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data.OrderList[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
					tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly =
						strategyDecoder(genOrds.Data.OrderList[i].Force)
				}
				resp = append(resp, tempOrds...)
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return resp, nil
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (bi *Bitget) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	err := getOrdersRequest.Validate()
	if err != nil {
		return nil, err
	}
	pairs := make([]string, len(getOrdersRequest.Pairs))
	for x := range getOrdersRequest.Pairs {
		pairs[x] = getOrdersRequest.Pairs[x].String()
	}
	if len(pairs) == 0 {
		pairs = append(pairs, "")
	}
	var resp order.FilteredOrders
	for x := range pairs {
		switch getOrdersRequest.AssetType {
		case asset.Spot:
			fillMap := make(map[int64][]order.TradeHistory)
			var pagination int64
			if pairs[x] != "" {
				err = bi.spotFillsHelper(ctx, pairs[x], fillMap)
				if err != nil {
					return nil, err
				}
				resp, err = bi.spotHistoricPlanOrdersHelper(ctx, pairs[x], getOrdersRequest.Pairs[x], resp, fillMap)
				if err != nil {
					return nil, err
				}
			} else {
				newPairs, err := bi.FetchTradablePairs(ctx, asset.Spot)
				if err != nil {
					return nil, err
				}
				for y := range newPairs {
					callStr, err := bi.FormatExchangeCurrency(newPairs[y], asset.Spot)
					if err != nil {
						return nil, err
					}
					err = bi.spotFillsHelper(ctx, callStr.String(), fillMap)
					if err != nil {
						return nil, err
					}
					resp, err = bi.spotHistoricPlanOrdersHelper(ctx, callStr.String(), newPairs[y], resp,
						fillMap)
					if err != nil {
						return nil, err
					}
				}
			}
			for {
				genOrds, err := bi.GetHistoricalSpotOrders(ctx, pairs[x], time.Time{}, time.Time{}, 100, pagination,
					0)
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.Data) == 0 ||
					pagination == int64(genOrds.Data[len(genOrds.Data)-1].OrderID) {
					break
				}
				pagination = int64(genOrds.Data[len(genOrds.Data)-1].OrderID)
				tempOrds := make([]order.Detail, len(genOrds.Data))
				for i := range genOrds.Data {
					tempOrds[i] = order.Detail{
						Exchange:             bi.Name,
						AssetType:            asset.Spot,
						AccountID:            genOrds.Data[i].UserID,
						OrderID:              strconv.FormatInt(int64(genOrds.Data[i].OrderID), 10),
						ClientOrderID:        genOrds.Data[i].ClientOrderID,
						Price:                genOrds.Data[i].Price,
						Amount:               genOrds.Data[i].Size,
						Type:                 typeDecoder(genOrds.Data[i].OrderType),
						Side:                 sideDecoder(genOrds.Data[i].Side),
						Status:               statusDecoder(genOrds.Data[i].Status),
						AverageExecutedPrice: genOrds.Data[i].PriceAverage,
						QuoteAmount:          genOrds.Data[i].QuoteVolume,
						Date:                 genOrds.Data[i].CreationTime.Time(),
						LastUpdated:          genOrds.Data[i].UpdateTime.Time(),
					}
					if pairs[x] != "" {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
					for y := range genOrds.Data[i].FeeDetail {
						tempOrds[i].Fee += genOrds.Data[i].FeeDetail[y].TotalFee
						tempOrds[i].FeeAsset = currency.NewCode(genOrds.Data[i].FeeDetail[y].FeeCoinCode)
					}
					if len(fillMap[int64(genOrds.Data[i].OrderID)]) > 0 {
						tempOrds[i].Trades = fillMap[int64(genOrds.Data[i].OrderID)]
					}
				}
				resp = append(resp, tempOrds...)
			}
		case asset.Futures:
			if pairs[x] != "" {
				resp, err = bi.historicalFuturesOrderHelper(ctx, pairs[x],
					getProductType(getOrdersRequest.Pairs[x]), getOrdersRequest.Pairs[x], resp)
				if err != nil {
					return nil, err
				}
			} else {
				for y := range prodTypes {
					resp, err = bi.historicalFuturesOrderHelper(ctx, "", prodTypes[y], currency.Pair{}, resp)
					if err != nil {
						return nil, err
					}
				}
			}
		case asset.Margin, asset.CrossMargin:
			var pagination int64
			var genFills *MarginOrderFills
			fillMap := make(map[int64][]order.TradeHistory)
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genFills, err = bi.GetIsolatedOrderFills(ctx, pairs[x], 0, pagination, 500,
						time.Now().Add(-time.Hour*24*90), time.Now())
				} else {
					genFills, err = bi.GetCrossOrderFills(ctx, pairs[x], 0, pagination, 500,
						time.Now().Add(-time.Hour*24*90), time.Now())
				}
				if err != nil {
					return nil, err
				}
				if genFills == nil || len(genFills.Data.Fills) == 0 || pagination == int64(genFills.Data.MaxID) {
					break
				}
				pagination = int64(genFills.Data.MaxID)
				for i := range genFills.Data.Fills {
					fillMap[genFills.Data.Fills[i].TradeID] = append(fillMap[genFills.Data.Fills[i].TradeID],
						order.TradeHistory{
							TID:       strconv.FormatInt(genFills.Data.Fills[i].TradeID, 10),
							Type:      typeDecoder(genFills.Data.Fills[i].OrderType),
							Side:      sideDecoder(genFills.Data.Fills[i].Side),
							Price:     genFills.Data.Fills[i].PriceAverage,
							Amount:    genFills.Data.Fills[i].Size,
							Timestamp: genFills.Data.Fills[i].CreationTime.Time(),
							Fee:       genFills.Data.Fills[i].FeeDetail.TotalFee,
							FeeAsset:  genFills.Data.Fills[i].FeeDetail.FeeCoin,
						})
				}
			}
			pagination = 0
			var genOrds *MarginHistOrds
			for {
				if getOrdersRequest.AssetType == asset.Margin {
					genOrds, err = bi.GetIsolatedHistoricalOrders(ctx, pairs[x], "", "", 0, 500, pagination,
						time.Now().Add(-time.Hour*24*90), time.Time{})
				} else {
					genOrds, err = bi.GetCrossHistoricalOrders(ctx, pairs[x], "", "", 0, 500, pagination,
						time.Now().Add(-time.Hour*24*90), time.Time{})
				}
				if err != nil {
					return nil, err
				}
				if genOrds == nil || len(genOrds.Data.OrderList) == 0 || pagination == int64(genOrds.Data.MaxID) {
					break
				}
				pagination = int64(genOrds.Data.MaxID)
				tempOrds := make([]order.Detail, len(genOrds.Data.OrderList))
				for i := range genOrds.Data.OrderList {
					tempOrds[i] = order.Detail{
						Exchange:             bi.Name,
						AssetType:            getOrdersRequest.AssetType,
						OrderID:              strconv.FormatInt(int64(genOrds.Data.OrderList[i].OrderID), 10),
						Type:                 typeDecoder(genOrds.Data.OrderList[i].OrderType),
						ClientOrderID:        genOrds.Data.OrderList[i].ClientOrderID,
						Price:                genOrds.Data.OrderList[i].Price,
						Side:                 sideDecoder(genOrds.Data.OrderList[i].Side),
						Status:               statusDecoder(genOrds.Data.OrderList[i].Status),
						Amount:               genOrds.Data.OrderList[i].Size,
						QuoteAmount:          genOrds.Data.OrderList[i].QuoteSize,
						AverageExecutedPrice: genOrds.Data.OrderList[i].PriceAverage,
						Date:                 genOrds.Data.OrderList[i].CreationTime.Time(),
						LastUpdated:          genOrds.Data.OrderList[i].UpdateTime.Time(),
					}
					if pairs[x] != "" {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data.OrderList[i].Symbol)
						if err != nil {
							return nil, err
						}
					}
					tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly =
						strategyDecoder(genOrds.Data.OrderList[i].Force)
					if len(fillMap[int64(genOrds.Data.OrderList[i].OrderID)]) > 0 {
						tempOrds[i].Trades = fillMap[int64(genOrds.Data.OrderList[i].OrderID)]
					}
				}
				resp = append(resp, tempOrds...)
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return resp, nil
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (bi *Bitget) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	fee, err := bi.GetTradeRate(ctx, feeBuilder.Pair.String(), "spot")
	if err != nil {
		return 0, err
	}
	if feeBuilder.IsMaker {
		return fee.Data.MakerFeeRate * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
	}
	return fee.Data.TakerFeeRate * feeBuilder.Amount * feeBuilder.PurchasePrice, nil
}

// ValidateAPICredentials validates current credentials used for wrapper
func (bi *Bitget) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := bi.UpdateAccountInfo(ctx, assetType)
	return bi.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (bi *Bitget) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := bi.GetKlineRequest(pair, a, interval, start, end, false)
	if err != nil {
		return nil, err
	}
	var resp []kline.Candle
	switch a {
	case asset.Spot, asset.Margin, asset.CrossMargin:
		cndl, err := bi.GetSpotCandlestickData(ctx, req.RequestFormatted.String(),
			formatExchangeKlineIntervalSpot(req.ExchangeInterval), req.Start, req.End, 200, true)
		if err != nil {
			return nil, err
		}
		resp = make([]kline.Candle, len(cndl.SpotCandles))
		for i := range cndl.SpotCandles {
			resp[i] = kline.Candle{
				Time:   cndl.SpotCandles[i].Timestamp,
				Low:    cndl.SpotCandles[i].Low,
				High:   cndl.SpotCandles[i].High,
				Open:   cndl.SpotCandles[i].Open,
				Close:  cndl.SpotCandles[i].Close,
				Volume: cndl.SpotCandles[i].BaseVolume,
			}
		}
	case asset.Futures:
		cndl, err := bi.GetFuturesCandlestickData(ctx, req.RequestFormatted.String(), getProductType(pair),
			formatExchangeKlineIntervalFutures(req.ExchangeInterval), req.Start, req.End, 200, CallModeHistory)
		if err != nil {
			return nil, err
		}
		resp = make([]kline.Candle, len(cndl.FuturesCandles))
		for i := range cndl.FuturesCandles {
			resp[i] = kline.Candle{
				Time:   cndl.FuturesCandles[i].Timestamp,
				Low:    cndl.FuturesCandles[i].Low,
				High:   cndl.FuturesCandles[i].High,
				Open:   cndl.FuturesCandles[i].Entry,
				Close:  cndl.FuturesCandles[i].Exit,
				Volume: cndl.FuturesCandles[i].BaseVolume,
			}
		}
	default:
		return nil, asset.ErrNotSupported
	}
	return req.ProcessResponse(resp)
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (bi *Bitget) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	req, err := bi.GetKlineExtendedRequest(pair, a, interval, start, end)
	if err != nil {
		return nil, err
	}
	var resp []kline.Candle
	for x := range req.RangeHolder.Ranges {
		switch a {
		case asset.Spot, asset.Margin, asset.CrossMargin:
			cndl, err := bi.GetSpotCandlestickData(ctx, req.RequestFormatted.String(),
				formatExchangeKlineIntervalSpot(req.ExchangeInterval), req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time, 200, true)
			if err != nil {
				return nil, err
			}
			temp := make([]kline.Candle, len(cndl.SpotCandles))
			for i := range cndl.SpotCandles {
				temp[i] = kline.Candle{
					Time:   cndl.SpotCandles[i].Timestamp,
					Low:    cndl.SpotCandles[i].Low,
					High:   cndl.SpotCandles[i].High,
					Open:   cndl.SpotCandles[i].Open,
					Close:  cndl.SpotCandles[i].Close,
					Volume: cndl.SpotCandles[i].BaseVolume,
				}
			}
			resp = append(resp, temp...)
		case asset.Futures:
			cndl, err := bi.GetFuturesCandlestickData(ctx, req.RequestFormatted.String(), getProductType(pair),
				formatExchangeKlineIntervalFutures(req.ExchangeInterval), req.RangeHolder.Ranges[x].Start.Time,
				req.RangeHolder.Ranges[x].End.Time, 200, CallModeHistory)
			if err != nil {
				return nil, err
			}
			temp := make([]kline.Candle, len(cndl.FuturesCandles))
			for i := range cndl.FuturesCandles {
				temp[i] = kline.Candle{
					Time:   cndl.FuturesCandles[i].Timestamp,
					Low:    cndl.FuturesCandles[i].Low,
					High:   cndl.FuturesCandles[i].High,
					Open:   cndl.FuturesCandles[i].Entry,
					Close:  cndl.FuturesCandles[i].Exit,
					Volume: cndl.FuturesCandles[i].BaseVolume,
				}
			}
			resp = append(resp, temp...)
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return req.ProcessResponse(resp)
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (bi *Bitget) GetFuturesContractDetails(ctx context.Context, _ asset.Item) ([]futures.Contract, error) {
	var contracts []futures.Contract
	for i := range prodTypes {
		resp, err := bi.GetContractConfig(ctx, "", prodTypes[i])
		if err != nil {
			return nil, err
		}
		temp := make([]futures.Contract, len(resp.Data))
		for x := range resp.Data {
			temp[x] = futures.Contract{
				Exchange: bi.Name,
				Name: currency.NewPair(currency.NewCode(resp.Data[x].BaseCoin),
					currency.NewCode(resp.Data[x].QuoteCoin)),
				Multiplier:  resp.Data[x].SizeMultiplier,
				Asset:       itemDecoder(resp.Data[x].SymbolType),
				Type:        contractTypeDecoder(resp.Data[x].SymbolType),
				Status:      resp.Data[x].SymbolStatus,
				StartDate:   resp.Data[x].DeliveryStartTime.Time(),
				EndDate:     resp.Data[x].DeliveryTime.Time(),
				MaxLeverage: resp.Data[x].MaxLever,
			}
			set := make(currency.Currencies, len(resp.Data[x].SupportMarginCoins))
			for y := range resp.Data[x].SupportMarginCoins {
				set[y] = currency.NewCode(resp.Data[x].SupportMarginCoins[y])
			}
			temp[x].SettlementCurrencies = set
			if resp.Data[x].SymbolStatus == "listed" || resp.Data[x].SymbolStatus == "normal" {
				temp[x].IsActive = true
			}
		}
		contracts = append(contracts, temp...)
	}
	return contracts, nil
}

// GetLatestFundingRates returns the latest funding rates data
func (bi *Bitget) GetLatestFundingRates(ctx context.Context, req *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	curRate, err := bi.GetFundingCurrent(ctx, req.Pair.String(), getProductType(req.Pair))
	if err != nil {
		return nil, err
	}
	nextTime, err := bi.GetNextFundingTime(ctx, req.Pair.String(), getProductType(req.Pair))
	if err != nil {
		return nil, err
	}
	resp := []fundingrate.LatestRateResponse{
		{
			Exchange:       bi.Name,
			Pair:           req.Pair,
			TimeOfNextRate: nextTime.Data[0].NextFundingTime.Time(),
			TimeChecked:    time.Now(),
		},
	}
	dec := decimal.NewFromFloat(curRate.Data[0].FundingRate)
	resp[0].LatestRate.Rate = dec
	return resp, nil
}

// UpdateOrderExecutionLimits updates order execution limits
func (bi *Bitget) UpdateOrderExecutionLimits(ctx context.Context, a asset.Item) error {
	var limits []order.MinMaxLevel
	switch a {
	case asset.Spot:
		resp, err := bi.GetSymbolInfo(ctx, "")
		if err != nil {
			return err
		}
		limits = make([]order.MinMaxLevel, len(resp.Data))
		for i := range resp.Data {
			limits[i] = order.MinMaxLevel{
				Asset: a,
				Pair: currency.NewPair(currency.NewCode(resp.Data[i].BaseCoin),
					currency.NewCode(resp.Data[i].QuoteCoin)),
				PriceStepIncrementSize:  float64(resp.Data[i].PricePrecision),
				AmountStepIncrementSize: float64(resp.Data[i].QuantityPrecision),
				QuoteStepIncrementSize:  float64(resp.Data[i].QuotePrecision),
				MinNotional:             resp.Data[i].MinTradeUSDT,
				MarketMinQty:            resp.Data[i].MinTradeAmount,
				MarketMaxQty:            resp.Data[i].MaxTradeAmount,
			}
		}
	case asset.Futures:
		for i := range prodTypes {
			resp, err := bi.GetContractConfig(ctx, "", prodTypes[i])
			if err != nil {
				return err
			}
			tempResp := make([]order.MinMaxLevel, len(resp.Data))
			for x := range resp.Data {
				tempResp[x] = order.MinMaxLevel{
					Asset: a,
					Pair: currency.NewPair(currency.NewCode(resp.Data[x].BaseCoin),
						currency.NewCode(resp.Data[x].QuoteCoin)),
					MinNotional:    resp.Data[x].MinTradeUSDT,
					MaxTotalOrders: resp.Data[x].MaxSymbolOpenOrderNum,
				}
			}
			limits = append(limits, tempResp...)
		}
	case asset.Margin, asset.CrossMargin:
		resp, err := bi.GetSupportedCurrencies(ctx)
		if err != nil {
			return err
		}
		limits = make([]order.MinMaxLevel, len(resp.Data))
		for i := range resp.Data {
			limits[i] = order.MinMaxLevel{
				Asset: a,
				Pair: currency.NewPair(currency.NewCode(resp.Data[i].BaseCoin),
					currency.NewCode(resp.Data[i].QuoteCoin)),
				MinNotional:             resp.Data[i].MinTradeUSDT,
				MarketMinQty:            resp.Data[i].MinTradeAmount,
				MarketMaxQty:            resp.Data[i].MaxTradeAmount,
				QuoteStepIncrementSize:  float64(resp.Data[i].PricePrecision),
				AmountStepIncrementSize: float64(resp.Data[i].QuantityPrecision),
			}
		}
	default:
		return asset.ErrNotSupported
	}
	return bi.LoadLimits(limits)
}

// UpdateCurrencyStates updates currency states
func (bi *Bitget) UpdateCurrencyStates(ctx context.Context, a asset.Item) error {
	payload := make(map[currency.Code]currencystate.Options)
	resp, err := bi.GetCoinInfo(ctx, "")
	if err != nil {
		return err
	}
	for i := range resp.Data {
		var withdraw bool
		var deposit bool
		var trade bool
		for j := range resp.Data[i].Chains {
			if resp.Data[i].Chains[j].Withdrawable {
				withdraw = true
			}
			if resp.Data[i].Chains[j].Rechargeable {
				deposit = true
			}
		}
		if withdraw && deposit {
			trade = true
		}
		payload[currency.NewCode(resp.Data[i].Coin)] = currencystate.Options{
			Withdraw: &withdraw,
			Deposit:  &deposit,
			Trade:    &trade,
		}
	}
	return bi.States.UpdateAll(a, payload)
}

// GetAvailableTransferChains returns a list of supported transfer chains based
// on the supplied cryptocurrency
func (bi *Bitget) GetAvailableTransferChains(ctx context.Context, cur currency.Code) ([]string, error) {
	if cur.IsEmpty() {
		return nil, errCurrencyEmpty
	}
	resp, err := bi.GetCoinInfo(ctx, cur.String())
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errReturnEmpty
	}
	chains := make([]string, len(resp.Data[0].Chains))
	for i := range resp.Data[0].Chains {
		chains[i] = resp.Data[0].Chains[i].Chain
	}
	return chains, nil
}

// GetMarginRatesHistory returns the margin rate history for the supplied currency
func (bi *Bitget) GetMarginRatesHistory(ctx context.Context, req *margin.RateHistoryRequest) (*margin.RateHistoryResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	var pagination int64
	rates := new(margin.RateHistoryResponse)
loop:
	for {
		switch req.Asset {
		case asset.Margin:
			resp, err := bi.GetIsolatedInterestHistory(ctx, req.Pair.String(), req.Currency.String(),
				req.StartDate, req.EndDate, 500, pagination)
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.Data.ResultList) == 0 || pagination == int64(resp.Data.MaxID) {
				break loop
			}
			pagination = int64(resp.Data.MaxID)
			for i := range resp.Data.ResultList {
				rates.Rates = append(rates.Rates, margin.Rate{
					DailyBorrowRate: decimal.NewFromFloat(resp.Data.ResultList[i].DailyInterestRate),
					Time:            resp.Data.ResultList[i].CreationTime.Time(),
				})
			}
		case asset.CrossMargin:
			resp, err := bi.GetCrossInterestHistory(ctx, req.Currency.String(), req.StartDate, req.EndDate, 500,
				pagination)
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.Data.ResultList) == 0 || pagination == int64(resp.Data.MaxID) {
				break loop
			}
			pagination = int64(resp.Data.MaxID)
			for i := range resp.Data.ResultList {
				rates.Rates = append(rates.Rates, margin.Rate{
					DailyBorrowRate: decimal.NewFromFloat(resp.Data.ResultList[i].DailyInterestRate),
					Time:            resp.Data.ResultList[i].CreationTime.Time(),
				})
			}
		default:
			return nil, asset.ErrNotSupported
		}
	}
	return rates, nil
}

// GetFuturesPositionSummary returns stats for a future position
func (bi *Bitget) GetFuturesPositionSummary(ctx context.Context, req *futures.PositionSummaryRequest) (*futures.PositionSummary, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	resp, err := bi.GetSinglePosition(ctx, getProductType(req.Pair), req.Pair.String(), req.Pair.Quote.String())
	if err != nil {
		return nil, err
	}
	if len(resp.Data) != 1 {
		// I'm not sure that it should actually return one data point in this case, replace this with a properly
		// formatted error message once certain (i.e. once you can test GetSinglePosition properly)
		return nil, fmt.Errorf("expected 1 position, received %v", len(resp.Data))
	}
	summary := &futures.PositionSummary{
		Pair:                         req.Pair,
		Asset:                        req.Asset,
		CurrentSize:                  decimal.NewFromFloat(resp.Data[0].OpenDelegateSize),
		InitialMarginRequirement:     decimal.NewFromFloat(resp.Data[0].MarginSize),
		AvailableEquity:              decimal.NewFromFloat(resp.Data[0].Available),
		FrozenBalance:                decimal.NewFromFloat(resp.Data[0].Locked),
		Leverage:                     decimal.NewFromFloat(resp.Data[0].Leverage),
		RealisedPNL:                  decimal.NewFromFloat(resp.Data[0].AchievedProfits),
		AverageOpenPrice:             decimal.NewFromFloat(resp.Data[0].OpenPriceAverage),
		UnrealisedPNL:                decimal.NewFromFloat(resp.Data[0].UnrealizedPL),
		MaintenanceMarginRequirement: decimal.NewFromFloat(resp.Data[0].KeepMarginRate),
		MarkPrice:                    decimal.NewFromFloat(resp.Data[0].MarkPrice),
		StartDate:                    resp.Data[0].CreationTime.Time(),
	}
	return summary, nil
}

// GetFuturesPositions returns futures positions for all currencies
func (bi *Bitget) GetFuturesPositions(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionDetails, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	var resp []futures.PositionDetails
	// This exchange needs pairs to be passed through, since a MarginCoin has to be provided
	for i := range req.Pairs {
		temp, err := bi.GetAllPositions(ctx, getProductType(req.Pairs[i]), req.Pairs[i].Quote.String())
		if err != nil {
			return nil, err
		}
		for x := range temp.Data {
			pair, err := pairFromStringHelper(temp.Data[x].Symbol)
			if err != nil {
				return nil, err
			}
			ord := []order.Detail{
				{
					Exchange:             bi.Name,
					AssetType:            req.Asset,
					Pair:                 pair,
					Side:                 sideDecoder(temp.Data[x].HoldSide),
					RemainingAmount:      temp.Data[x].OpenDelegateSize,
					Amount:               temp.Data[x].Total,
					Leverage:             temp.Data[x].Leverage,
					AverageExecutedPrice: temp.Data[x].OpenPriceAverage,
					MarginType:           marginDecoder(temp.Data[x].MarginMode),
					Price:                temp.Data[x].MarkPrice,
					Date:                 temp.Data[x].CreationTime.Time(),
				},
			}
			resp = append(resp, futures.PositionDetails{
				Exchange: bi.Name,
				Pair:     pair,
				Asset:    req.Asset,
				Orders:   ord,
			})
		}
	}
	return resp, nil
}

// GetFuturesPositionOrders returns futures positions orders
func (bi *Bitget) GetFuturesPositionOrders(ctx context.Context, req *futures.PositionsRequest) ([]futures.PositionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	pairs := make([]string, len(req.Pairs))
	for x := range req.Pairs {
		pairs[x] = req.Pairs[x].String()
	}
	if len(pairs) == 0 {
		pairs = append(pairs, "")
	}
	var resp []futures.PositionResponse
	var err error
	for x := range pairs {
		if pairs[x] != "" {
			resp, err = bi.allFuturesOrderHelper(ctx, pairs[x], getProductType(req.Pairs[x]), req.Pairs[x], resp)
			if err != nil {
				return nil, err
			}
		} else {
			for y := range prodTypes {
				resp, err = bi.allFuturesOrderHelper(ctx, "", prodTypes[y], currency.Pair{}, resp)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return resp, nil
}

// GetHistoricalFundingRates returns historical funding rates for a future
func (bi *Bitget) GetHistoricalFundingRates(ctx context.Context, req *fundingrate.HistoricalRatesRequest) (*fundingrate.HistoricalRates, error) {
	if req == nil {
		return nil, fmt.Errorf("%T %w", req, common.ErrNilPointer)
	}
	resp, err := bi.GetFundingHistorical(ctx, req.Pair.String(), getProductType(req.Pair), 100, 0)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errReturnEmpty
	}
	rates := make([]fundingrate.Rate, len(resp.Data))
	for i := range resp.Data {
		rates[i] = fundingrate.Rate{
			Time: resp.Data[i].FundingTime.Time(),
			Rate: decimal.NewFromFloat(resp.Data[i].FundingRate),
		}
	}
	rateStruct := &fundingrate.HistoricalRates{
		Exchange:     bi.Name,
		Asset:        req.Asset,
		Pair:         req.Pair,
		StartDate:    rates[0].Time,
		EndDate:      rates[len(rates)-1].Time,
		LatestRate:   rates[0],
		FundingRates: rates,
	}
	if len(rates) > 1 {
		rateStruct.TimeOfNextRate = rates[0].Time.Add(rates[0].Time.Sub(rates[1].Time))
	}
	return rateStruct, nil
}

// SetCollateralMode sets the account's collateral mode for the asset type
func (bi *Bitget) SetCollateralMode(_ context.Context, _ asset.Item, _ collateral.Mode) error {
	return common.ErrFunctionNotSupported
}

// GetCollateralMode returns the account's collateral mode for the asset type
func (bi *Bitget) GetCollateralMode(_ context.Context, _ asset.Item) (collateral.Mode, error) {
	return 0, common.ErrFunctionNotSupported
}

// SetMarginType sets the account's margin type for the asset type
func (bi *Bitget) SetMarginType(ctx context.Context, a asset.Item, p currency.Pair, t margin.Type) error {
	switch a {
	case asset.Futures:
		var str string
		switch t {
		case margin.Isolated:
			str = "isolated"
		case margin.Multi:
			str = "crossed"
		}
		_, err := bi.ChangeMarginMode(ctx, p.String(), getProductType(p), p.Quote.String(), str)
		if err != nil {
			return err
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// ChangePositionMargin changes the margin type for a position
func (bi *Bitget) ChangePositionMargin(_ context.Context, _ *margin.PositionChangeRequest) (*margin.PositionChangeResponse, error) {
	return nil, common.ErrFunctionNotSupported
}

// SetLeverage sets the account's initial leverage for the asset type and pair
func (bi *Bitget) SetLeverage(ctx context.Context, a asset.Item, p currency.Pair, _ margin.Type, f float64, s order.Side) error {
	switch a {
	case asset.Futures:
		_, err := bi.ChangeLeverage(ctx, p.String(), getProductType(p), p.Quote.String(), sideEncoder(s, true), f)
		if err != nil {
			return err
		}
	default:
		return asset.ErrNotSupported
	}
	return nil
}

// GetLeverage gets the account's initial leverage for the asset type and pair
func (bi *Bitget) GetLeverage(ctx context.Context, a asset.Item, p currency.Pair, t margin.Type, s order.Side) (float64, error) {
	lev := -1.1
	switch a {
	case asset.Futures:
		resp, err := bi.GetOneFuturesAccount(ctx, p.String(), getProductType(p), p.Quote.String())
		if err != nil {
			return lev, err
		}
		switch t {
		case margin.Isolated:
			switch s {
			case order.Buy, order.Long:
				lev = resp.Data.IsolatedLongLever
			case order.Sell, order.Short:
				lev = resp.Data.IsolatedShortLever
			default:
				return lev, order.ErrSideIsInvalid
			}
		case margin.Multi:
			lev = resp.Data.CrossedMarginleverage
		default:
			return lev, margin.ErrMarginTypeUnsupported
		}
	case asset.Margin:
		resp, err := bi.GetIsolatedInterestRateAndMaxBorrowable(ctx, p.String())
		if err != nil {
			return lev, err
		}
		if len(resp.Data) == 0 {
			return lev, errReturnEmpty
		}
		lev = resp.Data[0].Leverage
	case asset.CrossMargin:
		resp, err := bi.GetCrossInterestRateAndMaxBorrowable(ctx, p.Quote.String())
		if err != nil {
			return lev, err
		}
		if len(resp.Data) == 0 {
			return lev, errReturnEmpty
		}
		lev = resp.Data[0].Leverage
	default:
		return lev, asset.ErrNotSupported
	}
	return lev, nil
}

// GetOpenInterest returns the open interest rate for a given asset pair
func (bi *Bitget) GetOpenInterest(ctx context.Context, pairs ...key.PairAsset) ([]futures.OpenInterest, error) {
	openInterest := make([]futures.OpenInterest, len(pairs))
	for i := range pairs {
		resp, err := bi.GetOpenPositions(ctx, pairs[i].Pair().String(), getProductType(pairs[i].Pair()))
		if err != nil {
			return nil, err
		}
		if len(resp.Data.OpenInterestList) == 0 {
			return nil, errReturnEmpty
		}
		openInterest[i] = futures.OpenInterest{
			OpenInterest: resp.Data.OpenInterestList[0].Size,
			Key: key.ExchangePairAsset{
				Exchange: bi.Name,
				Base:     pairs[i].Base,
				Quote:    pairs[i].Quote,
				Asset:    pairs[i].Asset,
			},
		}
	}
	return openInterest, nil
}

// GetProductType is a helper function that returns the appropriate product type for a given currency pair
func getProductType(p currency.Pair) string {
	var prodType string
	switch p.Quote {
	case currency.USDT:
		prodType = "USDT-FUTURES"
	case currency.PERP, currency.USDC:
		prodType = "USDC-FUTURES"
	default:
		prodType = "COIN-FUTURES"
	}
	return prodType
}

// SideDecoder is a helper function that returns the appropriate order side for a given string
func sideDecoder(d string) order.Side {
	switch strings.ToLower(d) {
	case "buy", "long":
		return order.Buy
	case "sell", "short":
		return order.Sell
	}
	return order.UnknownSide
}

// StrategyTruthTable is a helper function that returns the appropriate strategy for a given set of booleans
func strategyTruthTable(ioc, fok, po bool) (string, error) {
	if (ioc && fok) || (fok && po) || (ioc && po) {
		return "", errStrategyMutex
	}
	if ioc {
		return "ioc", nil
	}
	if fok {
		return "fok", nil
	}
	if po {
		return "post_only", nil
	}
	return "gtc", nil
}

// ClientIDGenerator is a helper function that generates a unique client ID
func clientIDGenerator() string {
	i := time.Now().UnixNano()>>29 + time.Now().UnixNano()<<35
	cID := strconv.FormatInt(i, 31) + strconv.FormatInt(i, 29) + strconv.FormatInt(i, 23) + strconv.FormatInt(i, 19)
	if len(cID) > 50 {
		cID = cID[:50]
	}
	return cID
}

// MarginStringer is a helper function that returns the appropriate string for a given margin type
func marginStringer(m margin.Type) string {
	switch m {
	case margin.Isolated:
		return "isolated"
	case margin.Multi:
		return "crossed"
	}
	return ""
}

// SideEncoder is a helper function that returns the appropriate string for a given order side
func sideEncoder(s order.Side, longshort bool) string {
	switch s {
	case order.Buy, order.Long:
		if longshort {
			return "long"
		}
		return "buy"
	case order.Sell, order.Short:
		if longshort {
			return "short"
		}
		return "sell"
	}
	return "unknown side"
}

// PairBatcher is a helper function that batches orders by currency pair
func pairBatcher(orders []order.Cancel) (map[currency.Pair][]OrderIDStruct, error) {
	batchByPair := make(map[currency.Pair][]OrderIDStruct)
	for i := range orders {
		originalID, err := strconv.ParseInt(orders[i].OrderID, 10, 64)
		if err != nil {
			return nil, err
		}
		batchByPair[orders[i].Pair] = append(batchByPair[orders[i].Pair], OrderIDStruct{
			ClientOrderID: orders[i].ClientOrderID,
			OrderID:       originalID,
		})
	}
	return batchByPair, nil
}

// AddStatuses is a helper function that adds statuses to a response
func addStatuses(status *BatchOrderResp, resp *order.CancelBatchResponse) {
	for i := range status.Data.SuccessList {
		resp.Status[status.Data.SuccessList[i].ClientOrderID] = "success"
		resp.Status[strconv.FormatInt(int64(status.Data.SuccessList[i].OrderID), 10)] = "success"
	}
	for i := range status.Data.FailureList {
		resp.Status[status.Data.FailureList[i].ClientOrderID] = status.Data.FailureList[i].ErrorMessage
		resp.Status[strconv.FormatInt(int64(status.Data.FailureList[i].OrderID), 10)] =
			status.Data.FailureList[i].ErrorMessage
	}
}

// StatusDecoder is a helper function that returns the appropriate status for a given string
func statusDecoder(status string) order.Status {
	switch status {
	case "live":
		return order.Pending
	case "new":
		return order.New
	case "partially_filled", "partially_fill":
		return order.PartiallyFilled
	case "filled", "full_fill":
		return order.Filled
	case "cancelled", "canceled":
		return order.Cancelled
	case "not_trigger":
		return order.PendingTrigger
	}
	return order.UnknownStatus
}

// StrategyDecoder is a helper function that returns the appropriate strategy bools for a given string
func strategyDecoder(s string) (ioc, fok, po bool) {
	switch strings.ToLower(s) {
	case "ioc":
		ioc = true
	case "fok":
		fok = true
	case "post_only":
		po = true
	}
	return
}

// TypeDecoder is a helper function that returns the appropriate order type for a given string
func typeDecoder(s string) order.Type {
	switch s {
	case "limit":
		return order.Limit
	case "market":
		return order.Market
	}
	return order.UnknownType
}

// WithdrawalHistGrabber is a helper function that repeatedly calls GetWithdrawalRecords and returns all data
func (bi *Bitget) withdrawalHistGrabber(ctx context.Context, currency string) (*WithdrawRecordsResp, error) {
	var allData WithdrawRecordsResp
	var pagination int64
	for {
		resp, err := bi.GetWithdrawalRecords(ctx, currency, "", time.Now().Add(-time.Hour*24*90), time.Now(),
			pagination, 0, 100)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Data) == 0 || pagination == resp.Data[len(resp.Data)-1].OrderID {
			break
		}
		pagination = resp.Data[len(resp.Data)-1].OrderID
		allData.Data = append(allData.Data, resp.Data...)
	}
	return &allData, nil
}

var printOnce bool

// PairFromStringHelper is a helper function that does some checks to help with common ambiguous cases in this
// exchange
func pairFromStringHelper(s string) (currency.Pair, error) {
	pair := currency.Pair{}
	i := strings.Index(s, "USD")
	if i == -1 {
		i = strings.Index(s, "PERP")
		if i == -1 {
			return pair, errUnknownPairQuote
		}
	}
	pair, err := currency.NewPairFromString(s[:i] + "-" + s[i:])
	if err != nil {
		return pair, err
	}
	pair = pair.Format(currency.PairFormat{Uppercase: true, Delimiter: ""})
	if !printOnce && pair.Base == currency.TRX {
		fmt.Printf("Pair %v has been formatted to %v. Base: %v. Quote: %v. Delimiter: %v. String: %v", s, pair,
			pair.Base, pair.Quote, pair.Delimiter, pair.String())
		printOnce = true
	}
	return pair, nil
}

// SpotPlanOrdersHelper is a helper function that repeatedly calls GetCurrentSpotPlanOrders and returns all data
func (bi *Bitget) spotCurrentPlanOrdersHelper(ctx context.Context, pairStr string, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination int64
	for {
		genOrds, err := bi.GetCurrentSpotPlanOrders(ctx, pairStr, time.Time{}, time.Time{}, 100,
			pagination)
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.Data.OrderList) == 0 || pagination == int64(genOrds.Data.IDLessThan) {
			break
		}
		pagination = int64(genOrds.Data.IDLessThan)
		tempOrds := make([]order.Detail, len(genOrds.Data.OrderList))
		for i := range genOrds.Data.OrderList {
			tempOrds[i] = order.Detail{
				Exchange:      bi.Name,
				AssetType:     asset.Spot,
				OrderID:       strconv.FormatInt(int64(genOrds.Data.OrderList[i].OrderID), 10),
				ClientOrderID: genOrds.Data.OrderList[i].ClientOrderID,
				TriggerPrice:  genOrds.Data.OrderList[i].TriggerPrice,
				Type:          typeDecoder(genOrds.Data.OrderList[i].OrderType),
				Price:         float64(genOrds.Data.OrderList[i].ExecutePrice),
				Amount:        genOrds.Data.OrderList[i].Size,
				Status:        statusDecoder(genOrds.Data.OrderList[i].Status),
				Side:          sideDecoder(genOrds.Data.OrderList[i].Side),
				Date:          genOrds.Data.OrderList[i].CreationTime.Time(),
				LastUpdated:   genOrds.Data.OrderList[i].UpdateTime.Time(),
			}
			tempOrds[i].Pair = pairCan
		}
		resp = append(resp, tempOrds...)
		if !genOrds.Data.NextFlag {
			break
		}
	}
	return resp, nil
}

// MarginDecoder is a helper function that returns the appropriate margin type for a given string
func marginDecoder(s string) margin.Type {
	switch s {
	case "isolated":
		return margin.Isolated
	case "cross":
		return margin.Multi
	}
	return margin.Unknown
}

// ActiveFuturesOrderHelper is a helper function that repeatedly calls GetPendingFuturesOrders and
// GetPendingFuturesTriggerOrders, returning the data formatted appropriately
func (bi *Bitget) activeFuturesOrderHelper(ctx context.Context, pairStr, productType string, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination int64
	for {
		genOrds, err := bi.GetPendingFuturesOrders(ctx, 0, pagination, 100, "", pairStr, productType, "",
			time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.Data.EntrustedList) == 0 || pagination == int64(genOrds.Data.EndID) {
			break
		}
		pagination = int64(genOrds.Data.EndID)
		tempOrds := make([]order.Detail, len(genOrds.Data.EntrustedList))
		for i := range genOrds.Data.EntrustedList {
			tempOrds[i] = order.Detail{
				Exchange:             bi.Name,
				AssetType:            asset.Futures,
				Amount:               genOrds.Data.EntrustedList[i].Size,
				OrderID:              strconv.FormatInt(int64(genOrds.Data.EntrustedList[i].OrderID), 10),
				ClientOrderID:        genOrds.Data.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.Data.EntrustedList[i].Fee),
				Price:                float64(genOrds.Data.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.Data.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.Data.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.Data.EntrustedList[i].Side),
				SettlementCurrency:   currency.NewCode(genOrds.Data.EntrustedList[i].MarginCoin),
				QuoteAmount:          genOrds.Data.EntrustedList[i].QuoteVolume,
				Leverage:             genOrds.Data.EntrustedList[i].Leverage,
				MarginType:           marginDecoder(genOrds.Data.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.Data.EntrustedList[i].OrderType),
				Date:                 genOrds.Data.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.Data.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.Data.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.Data.EntrustedList[i].PresetStopLossPrice),
			}
			if pairStr != "" {
				tempOrds[i].Pair = pairCan
			} else {
				tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly =
				strategyDecoder(genOrds.Data.EntrustedList[i].Force)
		}
		resp = append(resp, tempOrds...)
	}
	for y := range planTypes {
		pagination = 0
		for {
			genOrds, err := bi.GetPendingTriggerFuturesOrders(ctx, 0, pagination, 100, "", pairStr, planTypes[y],
				productType, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.Data.EntrustedList) == 0 || pagination == int64(genOrds.Data.EndID) {
				break
			}
			pagination = int64(genOrds.Data.EndID)
			tempOrds := make([]order.Detail, len(genOrds.Data.EntrustedList))
			for i := range genOrds.Data.EntrustedList {
				tempOrds[i] = order.Detail{
					Exchange:  bi.Name,
					AssetType: asset.Futures,
					Amount:    genOrds.Data.EntrustedList[i].Size,
					OrderID: strconv.FormatInt(int64(genOrds.Data.EntrustedList[i].OrderID),
						10),
					ClientOrderID:      genOrds.Data.EntrustedList[i].ClientOrderID,
					Price:              float64(genOrds.Data.EntrustedList[i].Price),
					TriggerPrice:       float64(genOrds.Data.EntrustedList[i].TriggerPrice),
					Status:             statusDecoder(genOrds.Data.EntrustedList[i].PlanStatus),
					Side:               sideDecoder(genOrds.Data.EntrustedList[i].Side),
					SettlementCurrency: currency.NewCode(genOrds.Data.EntrustedList[i].MarginCoin),
					MarginType:         marginDecoder(genOrds.Data.EntrustedList[i].MarginMode),
					Type:               typeDecoder(genOrds.Data.EntrustedList[i].OrderType),
					Date:               genOrds.Data.EntrustedList[i].CreationTime.Time(),
					LastUpdated:        genOrds.Data.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:    float64(genOrds.Data.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:    float64(genOrds.Data.EntrustedList[i].PresetStopLossPrice),
				}
				if pairStr != "" {
					tempOrds[i].Pair = pairCan
				} else {
					tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
			}
			resp = append(resp, tempOrds...)
		}
	}
	return resp, nil
}

// SpotHistoricPlanOrdersHelper is a helper function that repeatedly calls GetHistoricalSpotOrders and returns
// all data formatted appropriately
func (bi *Bitget) spotHistoricPlanOrdersHelper(ctx context.Context, pairStr string, pairCan currency.Pair, resp []order.Detail, fillMap map[int64][]order.TradeHistory) ([]order.Detail, error) {
	var pagination int64
	for {
		genOrds, err := bi.GetSpotPlanOrderHistory(ctx, pairStr, time.Now().Add(-time.Hour*24*90), time.Now(), 100,
			pagination)
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.Data.OrderList) == 0 || pagination == int64(genOrds.Data.IDLessThan) {
			break
		}
		pagination = int64(genOrds.Data.IDLessThan)
		tempOrds := make([]order.Detail, len(genOrds.Data.OrderList))
		for i := range genOrds.Data.OrderList {
			tempOrds[i] = order.Detail{
				Exchange:      bi.Name,
				AssetType:     asset.Spot,
				OrderID:       strconv.FormatInt(int64(genOrds.Data.OrderList[i].OrderID), 10),
				ClientOrderID: genOrds.Data.OrderList[i].ClientOrderID,
				TriggerPrice:  genOrds.Data.OrderList[i].TriggerPrice,
				Type:          typeDecoder(genOrds.Data.OrderList[i].OrderType),
				Price:         float64(genOrds.Data.OrderList[i].ExecutePrice),
				Amount:        genOrds.Data.OrderList[i].Size,
				Status:        statusDecoder(genOrds.Data.OrderList[i].Status),
				Side:          sideDecoder(genOrds.Data.OrderList[i].Side),
				Date:          genOrds.Data.OrderList[i].CreationTime.Time(),
				LastUpdated:   genOrds.Data.OrderList[i].UpdateTime.Time(),
			}
			tempOrds[i].Pair = pairCan
			if len(fillMap[int64(genOrds.Data.OrderList[i].OrderID)]) > 0 {
				tempOrds[i].Trades = fillMap[int64(genOrds.Data.OrderList[i].OrderID)]
			}
		}
		resp = append(resp, tempOrds...)
		if !genOrds.Data.NextFlag {
			break
		}
	}
	return resp, nil
}

// HistoricalFuturesOrderHelper is a helper function that repeatedly calls GetFuturesFills,
// GetHistoricalFuturesOrders, and GetHistoricalTriggerFuturesOrders, returning the data formatted appropriately
func (bi *Bitget) historicalFuturesOrderHelper(ctx context.Context, pairStr, productType string, pairCan currency.Pair, resp []order.Detail) ([]order.Detail, error) {
	var pagination int64
	fillMap := make(map[int64][]order.TradeHistory)
	for {
		fillOrds, err := bi.GetFuturesFills(ctx, 0, pagination, 100, pairStr, productType, time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if fillOrds == nil || len(fillOrds.Data.FillList) == 0 || pagination == int64(fillOrds.Data.EndID) {
			break
		}
		pagination = int64(fillOrds.Data.EndID)
		for i := range fillOrds.Data.FillList {
			tempFill := order.TradeHistory{
				TID:       strconv.FormatInt(fillOrds.Data.FillList[i].TradeID, 10),
				Price:     fillOrds.Data.FillList[i].Price,
				Amount:    fillOrds.Data.FillList[i].BaseVolume,
				Side:      sideDecoder(fillOrds.Data.FillList[i].Side),
				Timestamp: fillOrds.Data.FillList[i].CreationTime.Time(),
			}
			for y := range fillOrds.Data.FillList[i].FeeDetail {
				tempFill.Fee += fillOrds.Data.FillList[i].FeeDetail[y].TotalFee
				tempFill.FeeAsset = fillOrds.Data.FillList[i].FeeDetail[y].FeeCoin
			}
			fillMap[fillOrds.Data.FillList[i].OrderID] = append(fillMap[fillOrds.Data.FillList[i].OrderID],
				tempFill)
		}
	}
	pagination = 0
	for {
		genOrds, err := bi.GetHistoricalFuturesOrders(ctx, 0, pagination, 100, "", pairStr, productType,
			time.Time{}, time.Time{})
		if err != nil {
			return nil, err
		}
		if genOrds == nil || len(genOrds.Data.EntrustedList) == 0 || pagination == int64(genOrds.Data.EndID) {
			break
		}
		pagination = int64(genOrds.Data.EndID)
		tempOrds := make([]order.Detail, len(genOrds.Data.EntrustedList))
		for i := range genOrds.Data.EntrustedList {
			tempOrds[i] = order.Detail{
				Exchange:             bi.Name,
				AssetType:            asset.Futures,
				Amount:               genOrds.Data.EntrustedList[i].Size,
				OrderID:              strconv.FormatInt(int64(genOrds.Data.EntrustedList[i].OrderID), 10),
				ClientOrderID:        genOrds.Data.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.Data.EntrustedList[i].Fee),
				Price:                float64(genOrds.Data.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.Data.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.Data.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.Data.EntrustedList[i].Side),
				SettlementCurrency:   currency.NewCode(genOrds.Data.EntrustedList[i].MarginCoin),
				QuoteAmount:          genOrds.Data.EntrustedList[i].QuoteVolume,
				Leverage:             genOrds.Data.EntrustedList[i].Leverage,
				MarginType:           marginDecoder(genOrds.Data.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.Data.EntrustedList[i].OrderType),
				Date:                 genOrds.Data.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.Data.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.Data.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.Data.EntrustedList[i].PresetStopLossPrice),
			}
			if pairStr != "" {
				tempOrds[i].Pair = pairCan
			} else {
				tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			tempOrds[i].ImmediateOrCancel, tempOrds[i].FillOrKill, tempOrds[i].PostOnly =
				strategyDecoder(genOrds.Data.EntrustedList[i].Force)
			if len(fillMap[int64(genOrds.Data.EntrustedList[i].OrderID)]) > 0 {
				tempOrds[i].Trades = fillMap[int64(genOrds.Data.EntrustedList[i].OrderID)]
			}
		}
		resp = append(resp, tempOrds...)
	}
	for y := range planTypes {
		pagination = 0
		for {
			genOrds, err := bi.GetHistoricalTriggerFuturesOrders(ctx, 0, pagination, 100, "", planTypes[y], "",
				pairStr, productType, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.Data.EntrustedList) == 0 || pagination == int64(genOrds.Data.EndID) {
				break
			}
			pagination = int64(genOrds.Data.EndID)
			tempOrds := make([]order.Detail, len(genOrds.Data.EntrustedList))
			for i := range genOrds.Data.EntrustedList {
				tempOrds[i] = order.Detail{
					Exchange:  bi.Name,
					AssetType: asset.Futures,
					Amount:    genOrds.Data.EntrustedList[i].Size,
					OrderID: strconv.FormatInt(int64(genOrds.Data.EntrustedList[i].OrderID),
						10),
					ClientOrderID:        genOrds.Data.EntrustedList[i].ClientOrderID,
					Status:               statusDecoder(genOrds.Data.EntrustedList[i].PlanStatus),
					Price:                float64(genOrds.Data.EntrustedList[i].Price),
					AverageExecutedPrice: float64(genOrds.Data.EntrustedList[i].PriceAverage),
					TriggerPrice:         float64(genOrds.Data.EntrustedList[i].TriggerPrice),
					Side:                 sideDecoder(genOrds.Data.EntrustedList[i].Side),
					SettlementCurrency:   currency.NewCode(genOrds.Data.EntrustedList[i].MarginCoin),
					MarginType:           marginDecoder(genOrds.Data.EntrustedList[i].MarginMode),
					Type:                 typeDecoder(genOrds.Data.EntrustedList[i].OrderType),
					Date:                 genOrds.Data.EntrustedList[i].CreationTime.Time(),
					LastUpdated:          genOrds.Data.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:      float64(genOrds.Data.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:      float64(genOrds.Data.EntrustedList[i].PresetStopLossPrice),
				}
				if pairStr != "" {
					tempOrds[i].Pair = pairCan
				} else {
					tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				if len(fillMap[int64(genOrds.Data.EntrustedList[i].OrderID)]) > 0 {
					tempOrds[i].Trades = fillMap[int64(genOrds.Data.EntrustedList[i].OrderID)]
				}
			}
			resp = append(resp, tempOrds...)
		}
	}
	return resp, nil
}

// SpotFillsHelper is a helper function that repeatedly calls GetSpotFills, directly altering the supplied map with that data
func (bi *Bitget) spotFillsHelper(ctx context.Context, pairStr string, fillMap map[int64][]order.TradeHistory) error {
	var pagination int64
	for {
		genFills, err := bi.GetSpotFills(ctx, pairStr, time.Time{}, time.Time{}, 100, pagination, 0)
		if err != nil {
			return err
		}
		if genFills == nil || len(genFills.Data) == 0 ||
			pagination == int64(genFills.Data[len(genFills.Data)-1].TradeID) {
			break
		}
		pagination = int64(genFills.Data[len(genFills.Data)-1].TradeID)
		for i := range genFills.Data {
			fillMap[genFills.Data[i].TradeID] = append(fillMap[genFills.Data[i].TradeID],
				order.TradeHistory{
					TID:       strconv.FormatInt(genFills.Data[i].TradeID, 10),
					Type:      typeDecoder(genFills.Data[i].OrderType),
					Side:      sideDecoder(genFills.Data[i].Side),
					Price:     genFills.Data[i].PriceAverage,
					Amount:    genFills.Data[i].Size,
					Fee:       genFills.Data[i].FeeDetail.TotalFee,
					FeeAsset:  genFills.Data[i].FeeDetail.FeeCoin,
					Timestamp: genFills.Data[i].CreationTime.Time(),
				})
		}
	}
	return nil
}

// FormatExchangeKlineIntervalSpot is a helper function used to convert kline.Interval to the string format
// required by the spot API
func formatExchangeKlineIntervalSpot(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1min"
	case kline.FiveMin:
		return "5min"
	case kline.FifteenMin:
		return "15min"
	case kline.ThirtyMin:
		return "30min"
	case kline.OneHour:
		return "1h"
	case kline.FourHour:
		return "4h"
	case kline.SixHour:
		return "6h"
	case kline.TwelveHour:
		return "12h"
	case kline.OneDay:
		return "1day"
	case kline.ThreeDay:
		return "3day"
	case kline.OneWeek:
		return "1week"
	case kline.OneMonth:
		return "1M"
	}
	return errIntervalNotSupported
}

// FormatExchangeKlineIntervalFutures is a helper function used to convert kline.Interval to the string format
// required by the futures API
func formatExchangeKlineIntervalFutures(interval kline.Interval) string {
	switch interval {
	case kline.OneMin:
		return "1m"
	case kline.ThreeMin:
		return "3m"
	case kline.FiveMin:
		return "5m"
	case kline.FifteenMin:
		return "15m"
	case kline.ThirtyMin:
		return "30m"
	case kline.OneHour:
		return "1H"
	case kline.FourHour:
		return "4H"
	case kline.SixHour:
		return "6H"
	case kline.TwelveHour:
		return "12H"
	case kline.OneDay:
		return "1D"
	case kline.ThreeDay:
		return "3D"
	case kline.OneWeek:
		return "1W"
	case kline.OneMonth:
		return "1M"
	}
	return errIntervalNotSupported
}

// ItemDecoder is a helper function that returns the appropriate asset.Item for a given string
func itemDecoder(s string) asset.Item {
	switch s {
	case "spot":
		return asset.Spot
	case "margin":
		return asset.Margin
	case "futures":
		return asset.Futures
	case "perpetual":
		return asset.PerpetualContract
	case "delivery":
		return asset.DeliveryFutures
	}
	return asset.Empty
}

// contractTypeDecoder is a helper function that returns the appropriate contract type for a given string
func contractTypeDecoder(s string) futures.ContractType {
	switch s {
	case "delivery":
		return futures.LongDated
	case "perpetual":
		return futures.Perpetual
	}
	return futures.Unknown
}

// AllFuturesOrderHelper is a helper function that repeatedly calls GetPendingFuturesOrders and
// GetPendingFuturesTriggerOrders, returning the data formatted appropriately
func (bi *Bitget) allFuturesOrderHelper(ctx context.Context, pairStr, productType string, pairCan currency.Pair, resp []futures.PositionResponse) ([]futures.PositionResponse, error) {
	var pagination1 int64
	var pagination2 int64
	var breakbool1 bool
	var breakbool2 bool
	tempOrds := make(map[currency.Pair][]order.Detail)
	for {
		var genOrds *FuturesOrdResp
		var err error
		if !breakbool1 {
			genOrds, err = bi.GetPendingFuturesOrders(ctx, 0, pagination1, 100, "", pairStr, productType, "",
				time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.Data.EntrustedList) == 0 || pagination1 == int64(genOrds.Data.EndID) {
				breakbool1 = true
			}
			pagination1 = int64(genOrds.Data.EndID)
		}
		if !breakbool2 {
			genOrds2, err := bi.GetHistoricalFuturesOrders(ctx, 0, pagination2, 100, "", pairStr, productType,
				time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds2 == nil || len(genOrds2.Data.EntrustedList) == 0 || pagination2 == int64(genOrds2.Data.EndID) {
				breakbool2 = true
			} else {
				if genOrds == nil {
					genOrds = new(FuturesOrdResp)
				}
				genOrds.Data.EntrustedList = append(genOrds.Data.EntrustedList, genOrds2.Data.EntrustedList...)
			}
			pagination2 = int64(genOrds.Data.EndID)
		}
		if breakbool1 && breakbool2 {
			break
		}
		for i := range genOrds.Data.EntrustedList {
			var thisPair currency.Pair
			if pairStr != "" {
				thisPair = pairCan
			} else {
				thisPair, err = pairFromStringHelper(genOrds.Data.EntrustedList[i].Symbol)
				if err != nil {
					return nil, err
				}
			}
			ioc, fok, po := strategyDecoder(genOrds.Data.EntrustedList[i].Force)
			tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
				Exchange:             bi.Name,
				Pair:                 thisPair,
				AssetType:            asset.Futures,
				Amount:               genOrds.Data.EntrustedList[i].Size,
				OrderID:              strconv.FormatInt(int64(genOrds.Data.EntrustedList[i].OrderID), 10),
				ClientOrderID:        genOrds.Data.EntrustedList[i].ClientOrderID,
				Fee:                  float64(genOrds.Data.EntrustedList[i].Fee),
				Price:                float64(genOrds.Data.EntrustedList[i].Price),
				AverageExecutedPrice: float64(genOrds.Data.EntrustedList[i].PriceAverage),
				Status:               statusDecoder(genOrds.Data.EntrustedList[i].Status),
				Side:                 sideDecoder(genOrds.Data.EntrustedList[i].Side),
				SettlementCurrency:   currency.NewCode(genOrds.Data.EntrustedList[i].MarginCoin),
				QuoteAmount:          genOrds.Data.EntrustedList[i].QuoteVolume,
				Leverage:             genOrds.Data.EntrustedList[i].Leverage,
				MarginType:           marginDecoder(genOrds.Data.EntrustedList[i].MarginMode),
				Type:                 typeDecoder(genOrds.Data.EntrustedList[i].OrderType),
				Date:                 genOrds.Data.EntrustedList[i].CreationTime.Time(),
				LastUpdated:          genOrds.Data.EntrustedList[i].UpdateTime.Time(),
				LimitPriceUpper:      float64(genOrds.Data.EntrustedList[i].PresetStopSurplusPrice),
				LimitPriceLower:      float64(genOrds.Data.EntrustedList[i].PresetStopLossPrice),
				ImmediateOrCancel:    ioc,
				FillOrKill:           fok,
				PostOnly:             po,
			})
		}
	}
	for y := range planTypes {
		pagination1 = 0
		for {
			genOrds, err := bi.GetPendingTriggerFuturesOrders(ctx, 0, pagination1, 100, "", pairStr, planTypes[y],
				productType, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.Data.EntrustedList) == 0 || pagination1 == int64(genOrds.Data.EndID) {
				break
			}
			pagination1 = int64(genOrds.Data.EndID)
			for i := range genOrds.Data.EntrustedList {
				var thisPair currency.Pair
				if pairStr != "" {
					thisPair = pairCan
				} else {
					thisPair, err = pairFromStringHelper(genOrds.Data.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
					Exchange:           bi.Name,
					Pair:               thisPair,
					AssetType:          asset.Futures,
					Amount:             genOrds.Data.EntrustedList[i].Size,
					OrderID:            strconv.FormatInt(int64(genOrds.Data.EntrustedList[i].OrderID), 10),
					ClientOrderID:      genOrds.Data.EntrustedList[i].ClientOrderID,
					Price:              float64(genOrds.Data.EntrustedList[i].Price),
					TriggerPrice:       float64(genOrds.Data.EntrustedList[i].TriggerPrice),
					Status:             statusDecoder(genOrds.Data.EntrustedList[i].PlanStatus),
					Side:               sideDecoder(genOrds.Data.EntrustedList[i].Side),
					SettlementCurrency: currency.NewCode(genOrds.Data.EntrustedList[i].MarginCoin),
					MarginType:         marginDecoder(genOrds.Data.EntrustedList[i].MarginMode),
					Type:               typeDecoder(genOrds.Data.EntrustedList[i].OrderType),
					Date:               genOrds.Data.EntrustedList[i].CreationTime.Time(),
					LastUpdated:        genOrds.Data.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:    float64(genOrds.Data.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:    float64(genOrds.Data.EntrustedList[i].PresetStopLossPrice),
				})
			}
		}
		pagination1 = 0
		for {
			genOrds, err := bi.GetHistoricalTriggerFuturesOrders(ctx, 0, pagination1, 100, "", planTypes[y], "",
				pairStr, productType, time.Time{}, time.Time{})
			if err != nil {
				return nil, err
			}
			if genOrds == nil || len(genOrds.Data.EntrustedList) == 0 || pagination1 == int64(genOrds.Data.EndID) {
				break
			}
			pagination1 = int64(genOrds.Data.EndID)
			for i := range genOrds.Data.EntrustedList {
				var thisPair currency.Pair
				if pairStr != "" {
					thisPair = pairCan
				} else {
					thisPair, err = pairFromStringHelper(genOrds.Data.EntrustedList[i].Symbol)
					if err != nil {
						return nil, err
					}
				}
				tempOrds[thisPair] = append(tempOrds[thisPair], order.Detail{
					Exchange:             bi.Name,
					Pair:                 thisPair,
					AssetType:            asset.Futures,
					Amount:               genOrds.Data.EntrustedList[i].Size,
					OrderID:              strconv.FormatInt(int64(genOrds.Data.EntrustedList[i].OrderID), 10),
					ClientOrderID:        genOrds.Data.EntrustedList[i].ClientOrderID,
					Status:               statusDecoder(genOrds.Data.EntrustedList[i].PlanStatus),
					Price:                float64(genOrds.Data.EntrustedList[i].Price),
					AverageExecutedPrice: float64(genOrds.Data.EntrustedList[i].PriceAverage),
					TriggerPrice:         float64(genOrds.Data.EntrustedList[i].TriggerPrice),
					Side:                 sideDecoder(genOrds.Data.EntrustedList[i].Side),
					SettlementCurrency:   currency.NewCode(genOrds.Data.EntrustedList[i].MarginCoin),
					MarginType:           marginDecoder(genOrds.Data.EntrustedList[i].MarginMode),
					Type:                 typeDecoder(genOrds.Data.EntrustedList[i].OrderType),
					Date:                 genOrds.Data.EntrustedList[i].CreationTime.Time(),
					LastUpdated:          genOrds.Data.EntrustedList[i].UpdateTime.Time(),
					LimitPriceUpper:      float64(genOrds.Data.EntrustedList[i].PresetTakeProfitPrice),
					LimitPriceLower:      float64(genOrds.Data.EntrustedList[i].PresetStopLossPrice),
				})
			}
		}
	}
	for x, y := range tempOrds {
		resp = append(resp, futures.PositionResponse{
			Pair:   x,
			Orders: y,
			Asset:  asset.Futures,
		})
	}
	return resp, nil
}
