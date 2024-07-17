package bitget

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
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
		if len(resp.Data) == 0 {
			break
		}
		// Not sure that this is the right end to use for pagination
		if pagination == resp.Data[len(resp.Data)-1].OrderID {
			break
		} else {
			pagination = resp.Data[len(resp.Data)-1].OrderID
		}
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
			s.Pair.Quote.String(), sideEncoder(s.Side), "", s.Type.Lower(), strat, cID.String(), 0, 0,
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
		_, err = bi.CancelOrderBySymbol(ctx, orderCancellation.Pair.String())
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
		switch ordInfo.MarginMode {
		case "isolated":
			resp.MarginType = margin.Isolated
		case "cross":
			resp.MarginType = margin.Multi
		}
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
	// var resp order.FilteredOrders
	switch getOrdersRequest.AssetType {
	case asset.Spot:
		for x := range pairs {
			var pagination int64
			for {
				genOrds, err := bi.GetUnfilledOrders(ctx, pairs[x], time.Time{}, time.Time{}, 100, pagination, 0)
				if err != nil {
					return nil, err
				}
				if len(genOrds.Data) == 0 {
					break
				}
				if pagination == int64(genOrds.Data[len(genOrds.Data)-1].OrderID) {
					break
				}
				pagination = int64(genOrds.Data[len(genOrds.Data)-1].OrderID)
				tempOrds := make([]order.Detail, len(genOrds.Data))
				for i := range genOrds.Data {
					tempOrds[i] = order.Detail{
						Exchange:  bi.Name,
						AccountID: genOrds.Data[i].UserID,

						AssetType: asset.Spot,
						// OrderID:   strconv.FormatInt(genOrds.Data[i].OrderID, 10),
						Side:   sideDecoder(genOrds.Data[i].Side),
						Type:   order.Limit,
						Amount: genOrds.Data[i].Size,
						// Price:     genOrds.Data[i].Price,
						Status: statusDecoder(genOrds.Data[i].Status),
						Date:   genOrds.Data[i].CreationTime.Time(),
					}
					if pairs[x] != "" {
						tempOrds[i].Pair = getOrdersRequest.Pairs[x]
					} else {
						tempOrds[i].Pair, err = pairFromStringHelper(genOrds.Data[i].Symbol)
					}
				}
			}
		}
	case asset.Futures:
	case asset.Margin, asset.CrossMargin:
	default:
		return nil, asset.ErrNotSupported
	}
	// Include trigger orders that haven't hit their price yet
	return nil, common.ErrNotYetImplemented
}

// GetOrderHistory retrieves account order information
// Can Limit response to specific order status
func (bi *Bitget) GetOrderHistory(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on the type of transaction
func (bi *Bitget) GetFeeByType(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	return 0, common.ErrNotYetImplemented
}

// ValidateAPICredentials validates current credentials used for wrapper
func (bi *Bitget) ValidateAPICredentials(ctx context.Context, assetType asset.Item) error {
	_, err := bi.UpdateAccountInfo(ctx, assetType)
	return bi.CheckTransientError(err)
}

// GetHistoricCandles returns candles between a time period for a set time interval
func (bi *Bitget) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricCandlesExtended returns candles between a time period for a set time interval
func (bi *Bitget) GetHistoricCandlesExtended(ctx context.Context, pair currency.Pair, a asset.Item, interval kline.Interval, start, end time.Time) (*kline.Item, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFuturesContractDetails returns all contracts from the exchange by asset type
func (bi *Bitget) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrNotYetImplemented
}

// GetLatestFundingRates returns the latest funding rates data
func (bi *Bitget) GetLatestFundingRates(_ context.Context, _ *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// UpdateOrderExecutionLimits updates order execution limits
func (bi *Bitget) UpdateOrderExecutionLimits(_ context.Context, _ asset.Item) error {
	return common.ErrNotYetImplemented
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
	case "buy":
		return order.Buy
	case "sell":
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
func sideEncoder(s order.Side) string {
	switch s {
	case order.Buy, order.Long:
		return "buy"
	case order.Sell, order.Short:
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

// WithdrawalHistGrabber repeatedly calls GetWithdrawalRecords and returns all data
func (bi *Bitget) withdrawalHistGrabber(ctx context.Context, currency string) (*WithdrawRecordsResp, error) {
	var allData WithdrawRecordsResp
	var pagination int64
	for {
		resp, err := bi.GetWithdrawalRecords(ctx, currency, "", time.Now().Add(-time.Hour*24*90), time.Now(),
			pagination, 0, 100)
		if err != nil {
			return nil, err
		}
		if len(resp.Data) == 0 {
			break
		}
		if pagination == resp.Data[len(resp.Data)-1].OrderID {
			break
		}
		pagination = resp.Data[len(resp.Data)-1].OrderID
		allData.Data = append(allData.Data, resp.Data...)
	}
	return &allData, nil
}

// PairFromStringHelper does some checks to help with common ambiguous cases in this exchange
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
	return pair, nil
}
