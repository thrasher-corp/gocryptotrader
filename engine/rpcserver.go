package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/common/crypto"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/engine/events"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/gctrpc"
	"github.com/thrasher-/gocryptotrader/gctrpc/auth"
	log "github.com/thrasher-/gocryptotrader/logger"
	"github.com/thrasher-/gocryptotrader/portfolio"
	"github.com/thrasher-/gocryptotrader/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// RPCServer struct
type RPCServer struct{}

func authenticateClient(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("unable to extract metadata")
	}

	authStr, ok := md["authorization"]
	if !ok {
		return ctx, fmt.Errorf("authorization header missing")
	}

	if !common.StringContains(authStr[0], "Basic") {
		return ctx, fmt.Errorf("basic not found in authorization header")
	}

	decoded, err := crypto.Base64Decode(common.SplitStrings(authStr[0], " ")[1])
	if err != nil {
		return ctx, fmt.Errorf("unable to base64 decode authorization header")
	}

	username := common.SplitStrings(string(decoded), ":")[0]
	password := common.SplitStrings(string(decoded), ":")[1]

	if username != Bot.Config.RemoteControl.Username || password != Bot.Config.RemoteControl.Password {
		return ctx, fmt.Errorf("username/password mismatch")
	}

	return ctx, nil
}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer() {
	err := checkCerts()
	if err != nil {
		log.Errorf("gRPC checkCerts failed. err: %s", err)
		return
	}

	log.Debugf("gRPC server support enabled. Starting gRPC server on https://%v.", Bot.Config.RemoteControl.GRPC.ListenAddress)
	lis, err := net.Listen("tcp", Bot.Config.RemoteControl.GRPC.ListenAddress)
	if err != nil {
		log.Errorf("gRPC server failed to bind to port: %s", err)
		return
	}

	targetDir := utils.GetTLSDir(Bot.Settings.DataDir)
	creds, err := credentials.NewServerTLSFromFile(filepath.Join(targetDir, "cert.pem"), filepath.Join(targetDir, "key.pem"))
	if err != nil {
		log.Errorf("gRPC server could not load TLS keys: %s", err)
		return
	}

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(grpcauth.UnaryServerInterceptor(authenticateClient)),
	}
	server := grpc.NewServer(opts...)
	s := RPCServer{}
	gctrpc.RegisterGoCryptoTraderServer(server, &s)

	go func() {
		if err := server.Serve(lis); err != nil {
			log.Errorf("gRPC server failed to serve: %s", err)
			return
		}
	}()

	log.Debugf("gRPC server started!")

	if Bot.Settings.EnableGRPCProxy {
		StartRPCRESTProxy()
	}
}

// StartRPCRESTProxy starts a gRPC proxy
func StartRPCRESTProxy() {
	log.Debugf("gRPC proxy server support enabled. Starting gRPC proxy server on http://%v.", Bot.Config.RemoteControl.GRPC.GRPCProxyListenAddress)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	targetDir := utils.GetTLSDir(Bot.Settings.DataDir)
	creds, err := credentials.NewClientTLSFromFile(filepath.Join(targetDir, "cert.pem"), "")
	if err != nil {
		log.Errorf("Unabled to start gRPC proxy. Err: %s", err)
		return
	}

	mux := grpcruntime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: Bot.Config.RemoteControl.Username,
			Password: Bot.Config.RemoteControl.Password,
		}),
	}
	err = gctrpc.RegisterGoCryptoTraderHandlerFromEndpoint(ctx, mux, Bot.Config.RemoteControl.GRPC.ListenAddress, opts)
	if err != nil {
		log.Errorf("Failed to register gRPC proxy. Err: %s", err)
	}

	go func() {
		if err := http.ListenAndServe(Bot.Config.RemoteControl.GRPC.GRPCProxyListenAddress, mux); err != nil {
			log.Errorf("gRPC proxy failed to server: %s", err)
			return
		}
	}()

	log.Debugf("gRPC proxy server started!")
	select {}

}

// GetInfo returns info about the current GoCryptoTrader session
func (s *RPCServer) GetInfo(ctx context.Context, r *gctrpc.GetInfoRequest) (*gctrpc.GetInfoResponse, error) {
	d := time.Since(Bot.Uptime)
	resp := gctrpc.GetInfoResponse{
		Uptime:               d.String(),
		EnabledExchanges:     int64(Bot.Config.CountEnabledExchanges()),
		AvailableExchanges:   int64(len(Bot.Config.Exchanges)),
		DefaultFiatCurrency:  Bot.Config.Currency.FiatDisplayCurrency.String(),
		DefaultForexProvider: Bot.Config.GetPrimaryForexProvider(),
	}

	return &resp, nil
}

// GetExchanges returns a list of exchanges
// Param is whether or not you wish to list enabled exchanges
func (s *RPCServer) GetExchanges(ctx context.Context, r *gctrpc.GetExchangesRequest) (*gctrpc.GetExchangesResponse, error) {
	exchanges := common.JoinStrings(GetExchanges(r.Enabled), ",")
	return &gctrpc.GetExchangesResponse{Exchanges: exchanges}, nil
}

// DisableExchange disables an exchange
func (s *RPCServer) DisableExchange(ctx context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GenericExchangeNameResponse, error) {
	err := UnloadExchange(r.Exchange)
	return &gctrpc.GenericExchangeNameResponse{}, err
}

// EnableExchange enables an exchange
func (s *RPCServer) EnableExchange(ctx context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GenericExchangeNameResponse, error) {
	err := LoadExchange(r.Exchange, false, nil)
	return &gctrpc.GenericExchangeNameResponse{}, err
}

// GetExchangeInfo gets info for a specific exchange
func (s *RPCServer) GetExchangeInfo(ctx context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GetExchangeInfoResponse, error) {
	exchCfg, err := Bot.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetExchangeInfoResponse{
		Name:            exchCfg.Name,
		Enabled:         exchCfg.Enabled,
		Verbose:         exchCfg.Verbose,
		UsingSandbox:    exchCfg.UseSandbox,
		HttpTimeout:     exchCfg.HTTPTimeout.String(),
		HttpUseragent:   exchCfg.HTTPUserAgent,
		HttpProxy:       exchCfg.ProxyAddress,
		BaseCurrencies:  common.JoinStrings(exchCfg.BaseCurrencies.Strings(), ","),
		SupportedAssets: exchCfg.CurrencyPairs.AssetTypes.JoinToString(","),

		// TO-DO fix pairs
		//EnabledPairs: common.JoinStrings(
		//	exchCfg.CurrencyPairs.Pairs.GetPairs().Enabled.Strings(), ","),
		//AvailablePairs: common.JoinStrings(
		//	exchCfg.CurrencyPairs.Spot.Available.Strings(), ","),
	}, nil
}

// GetTicker returns the ticker for a specified exchange, currency pair and
// asset type
func (s *RPCServer) GetTicker(ctx context.Context, r *gctrpc.GetTickerRequest) (*gctrpc.TickerResponse, error) {
	t, err := GetSpecificTicker(
		currency.Pair{
			Delimiter: r.Pair.Delimiter,
			Base:      currency.NewCode(r.Pair.Base),
			Quote:     currency.NewCode(r.Pair.Quote),
		},
		r.Exchange,
		assets.AssetType(r.AssetType),
	)
	if err != nil {
		return nil, err
	}

	resp := &gctrpc.TickerResponse{
		Pair:        r.Pair,
		LastUpdated: t.LastUpdated.Unix(),
		Last:        t.Last,
		High:        t.High,
		Low:         t.Low,
		Bid:         t.Bid,
		Ask:         t.Ask,
		Volume:      t.Volume,
		PriceAth:    t.PriceATH,
	}

	return resp, nil
}

// GetTickers returns a list of tickers for all enabled exchanges and all
// enabled currency pairs
func (s *RPCServer) GetTickers(ctx context.Context, r *gctrpc.GetTickersRequest) (*gctrpc.GetTickersResponse, error) {
	activeTickers := GetAllActiveTickers()
	var tickers []*gctrpc.Tickers

	for x := range activeTickers {
		var ticker gctrpc.Tickers
		ticker.Exchange = activeTickers[x].ExchangeName
		for y := range activeTickers[x].ExchangeValues {
			t := activeTickers[x].ExchangeValues[y]
			ticker.Tickers = append(ticker.Tickers, &gctrpc.TickerResponse{
				Pair: &gctrpc.CurrencyPair{
					Delimiter: t.Pair.Delimiter,
					Base:      t.Pair.Base.String(),
					Quote:     t.Pair.Quote.String(),
				},
				LastUpdated: t.LastUpdated.Unix(),
				Last:        t.Last,
				High:        t.High,
				Low:         t.Low,
				Bid:         t.Bid,
				Ask:         t.Ask,
				Volume:      t.Volume,
				PriceAth:    t.PriceATH,
			})
		}
		tickers = append(tickers, &ticker)
	}

	return &gctrpc.GetTickersResponse{Tickers: tickers}, nil
}

// GetOrderbook returns an orderbook for a specific exchange, currency pair
// and asset type
func (s *RPCServer) GetOrderbook(ctx context.Context, r *gctrpc.GetOrderbookRequest) (*gctrpc.OrderbookResponse, error) {
	ob, err := GetSpecificOrderbook(
		currency.Pair{
			Delimiter: r.Pair.Delimiter,
			Base:      currency.NewCode(r.Pair.Base),
			Quote:     currency.NewCode(r.Pair.Quote),
		},
		r.Exchange,
		assets.AssetType(r.AssetType),
	)
	if err != nil {
		return nil, err
	}

	var bids []*gctrpc.OrderbookItem
	for x := range ob.Bids {
		bids = append(bids, &gctrpc.OrderbookItem{
			Amount: ob.Bids[x].Amount,
			Price:  ob.Bids[x].Price,
		})
	}

	var asks []*gctrpc.OrderbookItem
	for x := range ob.Asks {
		asks = append(asks, &gctrpc.OrderbookItem{
			Amount: ob.Asks[x].Amount,
			Price:  ob.Asks[x].Price,
		})
	}

	resp := &gctrpc.OrderbookResponse{
		Pair:        r.Pair,
		Bids:        bids,
		Asks:        asks,
		LastUpdated: ob.LastUpdated.Unix(),
		AssetType:   r.AssetType,
	}

	return resp, nil
}

// GetOrderbooks returns a list of orderbooks for all enabled exchanges and all
// enabled currency pairs
func (s *RPCServer) GetOrderbooks(ctx context.Context, r *gctrpc.GetOrderbooksRequest) (*gctrpc.GetOrderbooksResponse, error) {
	activeOrderbooks := GetAllActiveOrderbooks()
	var orderbooks []*gctrpc.Orderbooks

	for x := range activeOrderbooks {
		var ob gctrpc.Orderbooks
		ob.Exchange = activeOrderbooks[x].ExchangeName
		for y := range activeOrderbooks[x].ExchangeValues {
			o := activeOrderbooks[x].ExchangeValues[y]
			var bids []*gctrpc.OrderbookItem
			for z := range o.Bids {
				bids = append(bids, &gctrpc.OrderbookItem{
					Amount: o.Bids[z].Amount,
					Price:  o.Bids[z].Price,
				})
			}

			var asks []*gctrpc.OrderbookItem
			for z := range o.Asks {
				asks = append(asks, &gctrpc.OrderbookItem{
					Amount: o.Asks[z].Amount,
					Price:  o.Asks[z].Price,
				})
			}

			ob.Orderbooks = append(ob.Orderbooks, &gctrpc.OrderbookResponse{
				Pair: &gctrpc.CurrencyPair{
					Delimiter: o.Pair.Delimiter,
					Base:      o.Pair.Base.String(),
					Quote:     o.Pair.Quote.String(),
				},
				LastUpdated: o.LastUpdated.Unix(),
				Bids:        bids,
				Asks:        asks,
			})
		}
		orderbooks = append(orderbooks, &ob)
	}

	return &gctrpc.GetOrderbooksResponse{Orderbooks: orderbooks}, nil
}

// GetAccountInfo returns an account balance for a specific exchange
func (s *RPCServer) GetAccountInfo(ctx context.Context, r *gctrpc.GetAccountInfoRequest) (*gctrpc.GetAccountInfoResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	resp, err := exch.GetAccountInfo()
	if err != nil {
		return nil, err
	}

	var accounts []*gctrpc.Account
	for x := range resp.Accounts {
		var a gctrpc.Account
		a.Id = resp.Accounts[x].ID
		for _, y := range resp.Accounts[x].Currencies {
			a.Currencies = append(a.Currencies, &gctrpc.AccountCurrencyInfo{
				Currency:   y.CurrencyName.String(),
				Hold:       y.Hold,
				TotalValue: y.TotalValue,
			})
		}
		accounts = append(accounts, &a)
	}

	return &gctrpc.GetAccountInfoResponse{Exchange: r.Exchange, Accounts: accounts}, nil
}

// GetConfig returns the bots config
func (s *RPCServer) GetConfig(ctx context.Context, r *gctrpc.GetConfigRequest) (*gctrpc.GetConfigResponse, error) {
	return &gctrpc.GetConfigResponse{}, common.ErrNotYetImplemented
}

// GetPortfolio returns the portfolio details
func (s *RPCServer) GetPortfolio(ctx context.Context, r *gctrpc.GetPortfolioRequest) (*gctrpc.GetPortfolioResponse, error) {
	var addrs []*gctrpc.PortfolioAddress
	botAddrs := Bot.Portfolio.Addresses

	for x := range botAddrs {
		addrs = append(addrs, &gctrpc.PortfolioAddress{
			Address:     botAddrs[x].Address,
			CoinType:    botAddrs[x].CoinType.String(),
			Description: botAddrs[x].Description,
			Balance:     botAddrs[x].Balance,
		})
	}

	resp := &gctrpc.GetPortfolioResponse{
		Portfolio: addrs,
	}

	return resp, nil
}

// GetPortfolioSummary returns the portfolio summary
func (s *RPCServer) GetPortfolioSummary(ctx context.Context, r *gctrpc.GetPortfolioSummaryRequest) (*gctrpc.GetPortfolioSummaryResponse, error) {
	result := Bot.Portfolio.GetPortfolioSummary()
	var resp gctrpc.GetPortfolioSummaryResponse

	p := func(coins []portfolio.Coin) []*gctrpc.Coin {
		var c []*gctrpc.Coin
		for x := range coins {
			c = append(c,
				&gctrpc.Coin{
					Coin:       coins[x].Coin.String(),
					Balance:    coins[x].Balance,
					Address:    coins[x].Address,
					Percentage: coins[x].Percentage,
				},
			)
		}
		return c
	}

	resp.CoinTotals = p(result.Totals)
	resp.CoinsOffline = p(result.Offline)
	resp.CoinsOfflineSummary = make(map[string]*gctrpc.OfflineCoins)
	for k, v := range result.OfflineSummary {
		var o []*gctrpc.OfflineCoinSummary
		for x := range v {
			o = append(o,
				&gctrpc.OfflineCoinSummary{
					Address:    v[x].Address,
					Balance:    v[x].Balance,
					Percentage: v[x].Percentage,
				},
			)
		}
		resp.CoinsOfflineSummary[k.String()] = &gctrpc.OfflineCoins{
			Addresses: o,
		}
	}
	resp.CoinsOnline = p(result.Online)
	resp.CoinsOnlineSummary = make(map[string]*gctrpc.OnlineCoins)
	for k, v := range result.OnlineSummary {
		o := make(map[string]*gctrpc.OnlineCoinSummary)
		for x, y := range v {
			o[x.String()] = &gctrpc.OnlineCoinSummary{
				Balance:    y.Balance,
				Percentage: y.Percentage,
			}
		}
		resp.CoinsOnlineSummary[k] = &gctrpc.OnlineCoins{
			Coins: o,
		}
	}

	return &resp, nil
}

// AddPortfolioAddress adds an address to the portfolio manager
func (s *RPCServer) AddPortfolioAddress(ctx context.Context, r *gctrpc.AddPortfolioAddressRequest) (*gctrpc.AddPortfolioAddressResponse, error) {
	Bot.Portfolio.AddAddress(r.Address, r.Description, currency.NewCode(r.CoinType), r.Balance)
	return &gctrpc.AddPortfolioAddressResponse{}, nil
}

// RemovePortfolioAddress removes an address from the portfolio manager
func (s *RPCServer) RemovePortfolioAddress(ctx context.Context, r *gctrpc.RemovePortfolioAddressRequest) (*gctrpc.RemovePortfolioAddressResponse, error) {
	Bot.Portfolio.RemoveAddress(r.Address, r.Description, currency.NewCode(r.CoinType))
	return &gctrpc.RemovePortfolioAddressResponse{}, nil
}

// GetForexProviders returns a list of available forex providers
func (s *RPCServer) GetForexProviders(ctx context.Context, r *gctrpc.GetForexProvidersRequest) (*gctrpc.GetForexProvidersResponse, error) {
	providers := Bot.Config.GetForexProvidersConfig()
	if len(providers) == 0 {
		return nil, fmt.Errorf("forex providers is empty")
	}

	var forexProviders []*gctrpc.ForexProvider
	for x := range providers {
		forexProviders = append(forexProviders, &gctrpc.ForexProvider{
			Name:             providers[x].Name,
			Enabled:          providers[x].Enabled,
			Verbose:          providers[x].Verbose,
			RestRollingDelay: providers[x].RESTPollingDelay.String(),
			ApiKey:           providers[x].APIKey,
			ApiKeyLevel:      int64(providers[x].APIKeyLvl),
			PrimaryProvider:  providers[x].PrimaryProvider,
		})
	}
	return &gctrpc.GetForexProvidersResponse{ForexProviders: forexProviders}, nil
}

// GetForexRates returns a list of forex rates
func (s *RPCServer) GetForexRates(ctx context.Context, r *gctrpc.GetForexRatesRequest) (*gctrpc.GetForexRatesResponse, error) {
	rates, err := currency.GetExchangeRates()
	if err != nil {
		return nil, err
	}

	if len(rates) == 0 {
		return nil, fmt.Errorf("forex rates is empty")
	}

	var forexRates []*gctrpc.ForexRatesConversion
	for x := range rates {
		rate, err := rates[x].GetRate()
		if err != nil {
			continue
		}

		// TODO
		// inverseRate, err := rates[x].GetInversionRate()
		// if err != nil {
		//	 continue
		// }

		forexRates = append(forexRates, &gctrpc.ForexRatesConversion{
			From:        rates[x].From.String(),
			To:          rates[x].To.String(),
			Rate:        rate,
			InverseRate: 0,
		})
	}
	return &gctrpc.GetForexRatesResponse{ForexRates: forexRates}, nil
}

// GetOrders returns all open orders, filtered by exchange, currency pair or
// asset type
func (s *RPCServer) GetOrders(ctx context.Context, r *gctrpc.GetOrdersRequest) (*gctrpc.GetOrdersResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		log.Debugln(exch)
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	resp, err := exch.GetActiveOrders(&exchange.GetOrdersRequest{})
	if err != nil {
		return nil, err
	}

	var orders []*gctrpc.OrderDetails
	for x := range resp {
		orders = append(orders, &gctrpc.OrderDetails{
			Exchange:      r.Exchange,
			Id:            resp[x].ID,
			BaseCurrency:  resp[x].CurrencyPair.Base.String(),
			QuoteCurrency: resp[x].CurrencyPair.Quote.String(),
			AssetType:     assets.AssetTypeSpot.String(),
			OrderType:     resp[x].OrderType.ToString(),
			OrderSide:     resp[x].OrderSide.ToString(),
			CreationTime:  resp[x].OrderDate.Unix(),
			Status:        resp[x].Status,
			Price:         resp[x].Price,
			Amount:        resp[x].Amount,
		})
	}

	return &gctrpc.GetOrdersResponse{Orders: orders}, nil
}

// GetOrder returns order information based on exchange and order ID
func (s *RPCServer) GetOrder(ctx context.Context, r *gctrpc.GetOrderRequest) (*gctrpc.OrderDetails, error) {
	return &gctrpc.OrderDetails{}, common.ErrNotYetImplemented
}

// SubmitOrder submits an order specified by exchange, currency pair and asset
// type
func (s *RPCServer) SubmitOrder(ctx context.Context, r *gctrpc.SubmitOrderRequest) (*gctrpc.SubmitOrderResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	p := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	result, err := exch.SubmitOrder(p, exchange.OrderSide(r.Side),
		exchange.OrderType(r.OrderType), r.Amount, r.Price, r.ClientId)

	return &gctrpc.SubmitOrderResponse{
		OrderId:     result.OrderID,
		OrderPlaced: result.IsOrderPlaced,
	}, err
}

// CancelOrder cancels an order specified by exchange, currency pair and asset
// type
func (s *RPCServer) CancelOrder(ctx context.Context, r *gctrpc.CancelOrderRequest) (*gctrpc.CancelOrderResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	err := exch.CancelOrder(&exchange.OrderCancellation{
		AccountID:     r.AccountId,
		OrderID:       r.OrderId,
		Side:          exchange.OrderSide(r.Side),
		WalletAddress: r.WalletAddress,
	})

	return &gctrpc.CancelOrderResponse{}, err
}

// CancelAllOrders cancels all orders, filterable by exchange
func (s *RPCServer) CancelAllOrders(ctx context.Context, r *gctrpc.CancelAllOrdersRequest) (*gctrpc.CancelAllOrdersResponse, error) {
	return &gctrpc.CancelAllOrdersResponse{}, common.ErrNotYetImplemented
}

// GetEvents returns the stored events list
func (s *RPCServer) GetEvents(ctx context.Context, r *gctrpc.GetEventsRequest) (*gctrpc.GetEventsResponse, error) {
	return &gctrpc.GetEventsResponse{}, common.ErrNotYetImplemented
}

// AddEvent adds an event
func (s *RPCServer) AddEvent(ctx context.Context, r *gctrpc.AddEventRequest) (*gctrpc.AddEventResponse, error) {
	evtCondition := events.ConditionParams{
		CheckBids:        r.ConditionParams.CheckBids,
		CheckBidsAndAsks: r.ConditionParams.CheckBidsAndAsks,
		Condition:        r.ConditionParams.Condition,
		OrderbookAmount:  r.ConditionParams.OrderbookAmount,
		Price:            r.ConditionParams.Price,
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base,
		r.Pair.Quote, r.Pair.Delimiter)

	id, err := events.Add(r.Exchange, r.Item, evtCondition, p, assets.AssetType(r.AssetType), r.Action)
	if err != nil {
		return nil, err
	}

	return &gctrpc.AddEventResponse{Id: id}, nil
}

// RemoveEvent removes an event, specified by an event ID
func (s *RPCServer) RemoveEvent(ctx context.Context, r *gctrpc.RemoveEventRequest) (*gctrpc.RemoveEventResponse, error) {
	events.Remove(r.Id)
	return &gctrpc.RemoveEventResponse{}, nil
}

// GetCryptocurrencyDepositAddresses returns a list of cryptocurrency deposit
// addresses specified by an exchange
func (s *RPCServer) GetCryptocurrencyDepositAddresses(ctx context.Context, r *gctrpc.GetCryptocurrencyDepositAddressesRequest) (*gctrpc.GetCryptocurrencyDepositAddressesResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	return &gctrpc.GetCryptocurrencyDepositAddressesResponse{}, common.ErrNotYetImplemented
}

// GetCryptocurrencyDepositAddress returns a cryptocurrency deposit address
// specified by exchange and cryptocurrency
func (s *RPCServer) GetCryptocurrencyDepositAddress(ctx context.Context, r *gctrpc.GetCryptocurrencyDepositAddressRequest) (*gctrpc.GetCryptocurrencyDepositAddressResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	addr, err := exch.GetDepositAddress(currency.NewCode(r.Cryptocurrency), "")
	return &gctrpc.GetCryptocurrencyDepositAddressResponse{Address: addr}, err
}

// WithdrawCryptocurrencyFunds withdraws cryptocurrency funds specified by
// exchange
func (s *RPCServer) WithdrawCryptocurrencyFunds(ctx context.Context, r *gctrpc.WithdrawCurrencyRequest) (*gctrpc.WithdrawResponse, error) {
	return &gctrpc.WithdrawResponse{}, common.ErrNotYetImplemented
}

// WithdrawFiatFunds withdraws fiat funds specified by exchange
func (s *RPCServer) WithdrawFiatFunds(ctx context.Context, r *gctrpc.WithdrawCurrencyRequest) (*gctrpc.WithdrawResponse, error) {
	return &gctrpc.WithdrawResponse{}, common.ErrNotYetImplemented
}
