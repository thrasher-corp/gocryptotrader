package bitget

import (
	"context"
	"strconv"
	"strings"
	"time"

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
			i := strings.Index(resp.Data[x].Symbol, "USD")
			if i == -1 {
				i = strings.Index(resp.Data[x].Symbol, "PERP")
				if i == -1 {
					return nil, errUnknownPairQuote
				}
			}
			pair, err := currency.NewPairFromString(resp.Data[x].Symbol[:i] + "-" + resp.Data[x].Symbol[i:])
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
	// funHist := []exchange.FundingHistory{}
	// var pagination int64
	// var done bool
	// for !done {
	// 	resp, err := bi.GetWithdrawalRecords(ctx, "", "", time.Now().Add(-time.Hour*24*90), time.Now(), pagination, 0,
	// 		100)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if len(resp.Data) == 0 {
	// 		break
	// 	}
	// 	tempHist := make([]exchange.FundingHistory, len(resp.Data))
	// 	for x := range resp.Data {
	// 	}
	// }
	return nil, common.ErrNotYetImplemented
}

// GetWithdrawalsHistory returns previous withdrawals data
func (bi *Bitget) GetWithdrawalsHistory(ctx context.Context, c currency.Code, a asset.Item) ([]exchange.WithdrawalHistory, error) {
	return nil, common.ErrNotYetImplemented
}

// GetRecentTrades returns the most recent trades for a currency and asset
func (bi *Bitget) GetRecentTrades(ctx context.Context, p currency.Pair, assetType asset.Item) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetHistoricTrades returns historic trade data within the timeframe provided
func (bi *Bitget) GetHistoricTrades(ctx context.Context, p currency.Pair, assetType asset.Item, timestampStart, timestampEnd time.Time) ([]trade.Data, error) {
	return nil, common.ErrNotYetImplemented
}

// GetServerTime returns the current exchange server time.
func (bi *Bitget) GetServerTime(ctx context.Context, a asset.Item) (time.Time, error) {
	return time.Time{}, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (bi *Bitget) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	// When an order has been submitted you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.SubmitResponse if you think they are applicable.
	// resp, err := s.DeriveSubmitResponse( /*newOrderID*/)
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.Date = exampleTime // e.g. If this is supplied by the exchanges API.
	// return resp, nil
	return nil, common.ErrNotYetImplemented
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (bi *Bitget) ModifyOrder(ctx context.Context, action *order.Modify) (*order.ModifyResponse, error) {
	if err := action.Validate(); err != nil {
		return nil, err
	}
	// When an order has been modified you can use this helpful constructor to
	// return. Please add any additional order details to the
	// order.ModifyResponse if you think they are applicable.
	// resp, err := action.DeriveModifyResponse()
	// if err != nil {
	// 	return nil, nil
	// }
	// resp.OrderID = maybeANewOrderID // e.g. If this is supplied by the exchanges API.
	return nil, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (bi *Bitget) CancelOrder(ctx context.Context, ord *order.Cancel) error {
	// if err := ord.Validate(ord.StandardCancel()); err != nil {
	//	 return err
	// }
	return common.ErrNotYetImplemented
}

// CancelBatchOrders cancels orders by their corresponding ID numbers
func (bi *Bitget) CancelBatchOrders(ctx context.Context, orders []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, common.ErrNotYetImplemented
}

// CancelAllOrders cancels all orders associated with a currency pair
func (bi *Bitget) CancelAllOrders(ctx context.Context, orderCancellation *order.Cancel) (order.CancelAllResponse, error) {
	// if err := orderCancellation.Validate(); err != nil {
	//	 return err
	// }
	return order.CancelAllResponse{}, common.ErrNotYetImplemented
}

// GetOrderInfo returns order information based on order ID
func (bi *Bitget) GetOrderInfo(ctx context.Context, orderID string, pair currency.Pair, assetType asset.Item) (*order.Detail, error) {
	return nil, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (bi *Bitget) GetDepositAddress(ctx context.Context, c currency.Code, accountID string, chain string) (*deposit.Address, error) {
	return nil, common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (bi *Bitget) WithdrawCryptocurrencyFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a withdrawal is
// submitted
func (bi *Bitget) WithdrawFiatFunds(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a withdrawal is
// submitted
func (bi *Bitget) WithdrawFiatFundsToInternationalBank(ctx context.Context, withdrawRequest *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	// if err := withdrawRequest.Validate(); err != nil {
	//	return nil, err
	// }
	return nil, common.ErrNotYetImplemented
}

// GetActiveOrders retrieves any orders that are active/open
func (bi *Bitget) GetActiveOrders(ctx context.Context, getOrdersRequest *order.MultiOrderRequest) (order.FilteredOrders, error) {
	// if err := getOrdersRequest.Validate(); err != nil {
	//	return nil, err
	// }
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

// GetProductType is a halper function that returns the appropriate product type for a given currency pair
func getProductType(p currency.Pair) string {
	var prodType string
	switch p.Quote {
	case currency.USDT:
		prodType = "USDT-FUTURES"
	case currency.PERP:
		prodType = "USDC-FUTURES"
	default:
		prodType = "COIN-FUTURES"
	}
	return prodType
}
