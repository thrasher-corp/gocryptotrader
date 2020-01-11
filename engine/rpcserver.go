package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	log "github.com/thrasher-corp/gocryptotrader/logger"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	errExchangeNameUnset = "exchange name unset"
	errCurrencyPairUnset = "currency pair unset"
	errAssetTypeUnset    = "asset type unset"
	errDispatchSystem    = "dispatch system offline"
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

	if !strings.Contains(authStr[0], "Basic") {
		return ctx, fmt.Errorf("basic not found in authorization header")
	}

	decoded, err := crypto.Base64Decode(strings.Split(authStr[0], " ")[1])
	if err != nil {
		return ctx, fmt.Errorf("unable to base64 decode authorization header")
	}

	username := strings.Split(string(decoded), ":")[0]
	password := strings.Split(string(decoded), ":")[1]

	if username != Bot.Config.RemoteControl.Username || password != Bot.Config.RemoteControl.Password {
		return ctx, fmt.Errorf("username/password mismatch")
	}

	return ctx, nil
}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer() {
	err := checkCerts()
	if err != nil {
		log.Errorf(log.GRPCSys, "gRPC checkCerts failed. err: %s\n", err)
		return
	}

	log.Debugf(log.GRPCSys, "gRPC server support enabled. Starting gRPC server on https://%v.\n", Bot.Config.RemoteControl.GRPC.ListenAddress)
	lis, err := net.Listen("tcp", Bot.Config.RemoteControl.GRPC.ListenAddress)
	if err != nil {
		log.Errorf(log.GRPCSys, "gRPC server failed to bind to port: %s", err)
		return
	}

	targetDir := utils.GetTLSDir(Bot.Settings.DataDir)
	creds, err := credentials.NewServerTLSFromFile(filepath.Join(targetDir, "cert.pem"), filepath.Join(targetDir, "key.pem"))
	if err != nil {
		log.Errorf(log.GRPCSys, "gRPC server could not load TLS keys: %s\n", err)
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
			log.Errorf(log.GRPCSys, "gRPC server failed to serve: %s\n", err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC server started!")

	if Bot.Settings.EnableGRPCProxy {
		StartRPCRESTProxy()
	}
}

// StartRPCRESTProxy starts a gRPC proxy
func StartRPCRESTProxy() {
	log.Debugf(log.GRPCSys, "gRPC proxy server support enabled. Starting gRPC proxy server on http://%v.\n", Bot.Config.RemoteControl.GRPC.GRPCProxyListenAddress)

	targetDir := utils.GetTLSDir(Bot.Settings.DataDir)
	creds, err := credentials.NewClientTLSFromFile(filepath.Join(targetDir, "cert.pem"), "")
	if err != nil {
		log.Errorf(log.GRPCSys, "Unabled to start gRPC proxy. Err: %s\n", err)
		return
	}

	mux := grpcruntime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: Bot.Config.RemoteControl.Username,
			Password: Bot.Config.RemoteControl.Password,
		}),
	}
	err = gctrpc.RegisterGoCryptoTraderHandlerFromEndpoint(context.Background(),
		mux, Bot.Config.RemoteControl.GRPC.ListenAddress, opts)
	if err != nil {
		log.Errorf(log.GRPCSys, "Failed to register gRPC proxy. Err: %s\n", err)
		return
	}

	go func() {
		if err := http.ListenAndServe(Bot.Config.RemoteControl.GRPC.GRPCProxyListenAddress, mux); err != nil {
			log.Errorf(log.GRPCSys, "gRPC proxy failed to server: %s\n", err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC proxy server started!")
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
		SubsystemStatus:      GetSubsystemsStatus(),
	}
	endpoints := GetRPCEndpoints()
	resp.RpcEndpoints = make(map[string]*gctrpc.RPCEndpoint)
	for k, v := range endpoints {
		resp.RpcEndpoints[k] = &gctrpc.RPCEndpoint{
			Started:       v.Started,
			ListenAddress: v.ListenAddr,
		}
	}
	return &resp, nil
}

// GetSubsystems returns a list of subsystems and their status
func (s *RPCServer) GetSubsystems(ctx context.Context, r *gctrpc.GetSubsystemsRequest) (*gctrpc.GetSusbsytemsResponse, error) {
	return &gctrpc.GetSusbsytemsResponse{SubsystemsStatus: GetSubsystemsStatus()}, nil
}

// EnableSubsystem enables a engine subsytem
func (s *RPCServer) EnableSubsystem(ctx context.Context, r *gctrpc.GenericSubsystemRequest) (*gctrpc.GenericSubsystemResponse, error) {
	err := SetSubsystem(r.Subsystem, true)
	return &gctrpc.GenericSubsystemResponse{}, err
}

// DisableSubsystem disables a engine subsytem
func (s *RPCServer) DisableSubsystem(ctx context.Context, r *gctrpc.GenericSubsystemRequest) (*gctrpc.GenericSubsystemResponse, error) {
	err := SetSubsystem(r.Subsystem, false)
	return &gctrpc.GenericSubsystemResponse{}, err
}

// GetRPCEndpoints returns a list of API endpoints
func (s *RPCServer) GetRPCEndpoints(ctx context.Context, r *gctrpc.GetRPCEndpointsRequest) (*gctrpc.GetRPCEndpointsResponse, error) {
	endpoints := GetRPCEndpoints()
	var resp gctrpc.GetRPCEndpointsResponse
	resp.Endpoints = make(map[string]*gctrpc.RPCEndpoint)
	for k, v := range endpoints {
		resp.Endpoints[k] = &gctrpc.RPCEndpoint{
			Started:       v.Started,
			ListenAddress: v.ListenAddr,
		}
	}
	return &resp, nil
}

// GetCommunicationRelayers returns the status of the engines communication relayers
func (s *RPCServer) GetCommunicationRelayers(ctx context.Context, r *gctrpc.GetCommunicationRelayersRequest) (*gctrpc.GetCommunicationRelayersResponse, error) {
	relayers, err := Bot.CommsManager.GetStatus()
	if err != nil {
		return nil, err
	}

	var resp gctrpc.GetCommunicationRelayersResponse
	resp.CommunicationRelayers = make(map[string]*gctrpc.CommunicationRelayer)
	for k, v := range relayers {
		resp.CommunicationRelayers[k] = &gctrpc.CommunicationRelayer{
			Enabled:   v.Enabled,
			Connected: v.Connected,
		}
	}
	return &resp, nil
}

// GetExchanges returns a list of exchanges
// Param is whether or not you wish to list enabled exchanges
func (s *RPCServer) GetExchanges(ctx context.Context, r *gctrpc.GetExchangesRequest) (*gctrpc.GetExchangesResponse, error) {
	exchanges := strings.Join(GetExchanges(r.Enabled), ",")
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

// GetExchangeOTPCode retrieves an exchanges OTP code
func (s *RPCServer) GetExchangeOTPCode(ctx context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GetExchangeOTPReponse, error) {
	result, err := GetExchangeoOTPByName(r.Exchange)
	return &gctrpc.GetExchangeOTPReponse{OtpCode: result}, err
}

// GetExchangeOTPCodes retrieves OTP codes for all exchanges which have an
// OTP secret installed
func (s *RPCServer) GetExchangeOTPCodes(ctx context.Context, r *gctrpc.GetExchangeOTPsRequest) (*gctrpc.GetExchangeOTPsResponse, error) {
	result, err := GetExchangeOTPs()
	return &gctrpc.GetExchangeOTPsResponse{OtpCodes: result}, err
}

// GetExchangeInfo gets info for a specific exchange
func (s *RPCServer) GetExchangeInfo(ctx context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GetExchangeInfoResponse, error) {
	exchCfg, err := Bot.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	resp := &gctrpc.GetExchangeInfoResponse{
		Name:           exchCfg.Name,
		Enabled:        exchCfg.Enabled,
		Verbose:        exchCfg.Verbose,
		UsingSandbox:   exchCfg.UseSandbox,
		HttpTimeout:    exchCfg.HTTPTimeout.String(),
		HttpUseragent:  exchCfg.HTTPUserAgent,
		HttpProxy:      exchCfg.ProxyAddress,
		BaseCurrencies: strings.Join(exchCfg.BaseCurrencies.Strings(), ","),
	}

	resp.SupportedAssets = make(map[string]*gctrpc.PairsSupported)
	for x := range exchCfg.CurrencyPairs.AssetTypes {
		a := exchCfg.CurrencyPairs.AssetTypes[x]
		resp.SupportedAssets[a.String()] = &gctrpc.PairsSupported{
			EnabledPairs:   exchCfg.CurrencyPairs.Get(a).Enabled.Join(),
			AvailablePairs: exchCfg.CurrencyPairs.Get(a).Available.Join(),
		}
	}
	return resp, nil
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
		asset.Item(r.AssetType),
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
		asset.Item(r.AssetType),
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
	err := Bot.Portfolio.AddAddress(r.Address, r.Description, currency.NewCode(r.CoinType), r.Balance)
	return &gctrpc.AddPortfolioAddressResponse{}, err
}

// RemovePortfolioAddress removes an address from the portfolio manager
func (s *RPCServer) RemovePortfolioAddress(ctx context.Context, r *gctrpc.RemovePortfolioAddressRequest) (*gctrpc.RemovePortfolioAddressResponse, error) {
	err := Bot.Portfolio.RemoveAddress(r.Address, r.Description, currency.NewCode(r.CoinType))
	return &gctrpc.RemovePortfolioAddressResponse{}, err
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
			RestPollingDelay: providers[x].RESTPollingDelay.String(),
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
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	resp, err := exch.GetActiveOrders(&order.GetOrdersRequest{
		Currencies: []currency.Pair{
			currency.NewPairWithDelimiter(r.Pair.Base,
				r.Pair.Quote, r.Pair.Delimiter),
		},
	})
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
			AssetType:     asset.Spot.String(),
			OrderType:     resp[x].OrderType.String(),
			OrderSide:     resp[x].OrderSide.String(),
			CreationTime:  resp[x].OrderDate.Unix(),
			Status:        resp[x].Status.String(),
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
	submission := &order.Submit{
		Pair:      p,
		OrderSide: order.Side(r.Side),
		OrderType: order.Type(r.OrderType),
		Amount:    r.Amount,
		Price:     r.Price,
		ClientID:  r.ClientId,
	}
	result, err := exch.SubmitOrder(submission)
	return &gctrpc.SubmitOrderResponse{
		OrderId:     result.OrderID,
		OrderPlaced: result.IsOrderPlaced,
	}, err
}

// SimulateOrder simulates an order specified by exchange, currency pair and asset
// type
func (s *RPCServer) SimulateOrder(ctx context.Context, r *gctrpc.SimulateOrderRequest) (*gctrpc.SimulateOrderResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	p := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	o, err := exch.FetchOrderbook(p, asset.Spot)
	if err != nil {
		return nil, err
	}

	var buy = true
	if !strings.EqualFold(r.Side, order.Buy.String()) &&
		!strings.EqualFold(r.Side, order.Bid.String()) {
		buy = false
	}

	result := o.SimulateOrder(r.Amount, buy)
	var resp gctrpc.SimulateOrderResponse
	for x := range result.Orders {
		resp.Orders = append(resp.Orders, &gctrpc.OrderbookItem{
			Price:  result.Orders[x].Price,
			Amount: result.Orders[x].Amount,
		})
	}

	resp.Amount = result.Amount
	resp.MaximumPrice = result.MaximumPrice
	resp.MinimumPrice = result.MinimumPrice
	resp.PercentageGainLoss = result.PercentageGainOrLoss
	resp.Status = result.Status
	return &resp, nil
}

// WhaleBomb finds the amount required to reach a specific price target for a given exchange, pair
// and asset type
func (s *RPCServer) WhaleBomb(ctx context.Context, r *gctrpc.WhaleBombRequest) (*gctrpc.SimulateOrderResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	p := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	o, err := exch.FetchOrderbook(p, asset.Spot)
	if err != nil {
		return nil, err
	}

	var buy = true
	if !strings.EqualFold(r.Side, order.Buy.String()) &&
		!strings.EqualFold(r.Side, order.Bid.String()) {
		buy = false
	}

	result, err := o.WhaleBomb(r.PriceTarget, buy)
	var resp gctrpc.SimulateOrderResponse
	for x := range result.Orders {
		resp.Orders = append(resp.Orders, &gctrpc.OrderbookItem{
			Price:  result.Orders[x].Price,
			Amount: result.Orders[x].Amount,
		})
	}

	resp.Amount = result.Amount
	resp.MaximumPrice = result.MaximumPrice
	resp.MinimumPrice = result.MinimumPrice
	resp.PercentageGainLoss = result.PercentageGainOrLoss
	resp.Status = result.Status
	return &resp, err
}

// CancelOrder cancels an order specified by exchange, currency pair and asset
// type
func (s *RPCServer) CancelOrder(ctx context.Context, r *gctrpc.CancelOrderRequest) (*gctrpc.CancelOrderResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	err := exch.CancelOrder(&order.Cancel{
		AccountID:     r.AccountId,
		OrderID:       r.OrderId,
		Side:          order.Side(r.Side),
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
	evtCondition := EventConditionParams{
		CheckBids:        r.ConditionParams.CheckBids,
		CheckBidsAndAsks: r.ConditionParams.CheckBidsAndAsks,
		Condition:        r.ConditionParams.Condition,
		OrderbookAmount:  r.ConditionParams.OrderbookAmount,
		Price:            r.ConditionParams.Price,
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base,
		r.Pair.Quote, r.Pair.Delimiter)

	id, err := Add(r.Exchange, r.Item, evtCondition, p, asset.Item(r.AssetType), r.Action)
	if err != nil {
		return nil, err
	}

	return &gctrpc.AddEventResponse{Id: id}, nil
}

// RemoveEvent removes an event, specified by an event ID
func (s *RPCServer) RemoveEvent(ctx context.Context, r *gctrpc.RemoveEventRequest) (*gctrpc.RemoveEventResponse, error) {
	Remove(r.Id)
	return &gctrpc.RemoveEventResponse{}, nil
}

// GetCryptocurrencyDepositAddresses returns a list of cryptocurrency deposit
// addresses specified by an exchange
func (s *RPCServer) GetCryptocurrencyDepositAddresses(ctx context.Context, r *gctrpc.GetCryptocurrencyDepositAddressesRequest) (*gctrpc.GetCryptocurrencyDepositAddressesResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	result, err := GetCryptocurrencyDepositAddressesByExchange(r.Exchange)
	return &gctrpc.GetCryptocurrencyDepositAddressesResponse{Addresses: result}, err
}

// GetCryptocurrencyDepositAddress returns a cryptocurrency deposit address
// specified by exchange and cryptocurrency
func (s *RPCServer) GetCryptocurrencyDepositAddress(ctx context.Context, r *gctrpc.GetCryptocurrencyDepositAddressRequest) (*gctrpc.GetCryptocurrencyDepositAddressResponse, error) {
	exch := GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errors.New("exchange is not loaded/doesn't exist")
	}

	addr, err := GetExchangeCryptocurrencyDepositAddress(r.Exchange, "", currency.NewCode(r.Cryptocurrency))
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

// GetLoggerDetails returns a loggers details
func (s *RPCServer) GetLoggerDetails(ctx context.Context, r *gctrpc.GetLoggerDetailsRequest) (*gctrpc.GetLoggerDetailsResponse, error) {
	levels, err := log.Level(r.Logger)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetLoggerDetailsResponse{
		Info:  levels.Info,
		Debug: levels.Debug,
		Warn:  levels.Warn,
		Error: levels.Error,
	}, nil
}

// SetLoggerDetails sets a loggers details
func (s *RPCServer) SetLoggerDetails(ctx context.Context, r *gctrpc.SetLoggerDetailsRequest) (*gctrpc.GetLoggerDetailsResponse, error) {
	levels, err := log.SetLevel(r.Logger, r.Level)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetLoggerDetailsResponse{
		Info:  levels.Info,
		Debug: levels.Debug,
		Warn:  levels.Warn,
		Error: levels.Error,
	}, nil
}

// GetExchangePairs returns a list of exchange supported assets and related pairs
func (s *RPCServer) GetExchangePairs(ctx context.Context, r *gctrpc.GetExchangePairsRequest) (*gctrpc.GetExchangePairsResponse, error) {
	exchCfg, err := Bot.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	if r.Asset != "" &&
		!exchCfg.CurrencyPairs.GetAssetTypes().Contains(asset.Item(r.Asset)) {
		return nil, errors.New("specified asset type does not exist")
	}

	var resp gctrpc.GetExchangePairsResponse
	resp.SupportedAssets = make(map[string]*gctrpc.PairsSupported)
	assetTypes := exchCfg.CurrencyPairs.GetAssetTypes()
	for x := range assetTypes {
		a := assetTypes[x]
		if r.Asset != "" && !strings.EqualFold(a.String(), r.Asset) {
			continue
		}
		resp.SupportedAssets[a.String()] = &gctrpc.PairsSupported{
			AvailablePairs: exchCfg.CurrencyPairs.Get(a).Available.Join(),
			EnabledPairs:   exchCfg.CurrencyPairs.Get(a).Enabled.Join(),
		}
	}
	return &resp, nil
}

// EnableExchangePair enables the specified pair on an exchange
func (s *RPCServer) EnableExchangePair(ctx context.Context, r *gctrpc.ExchangePairRequest) (*gctrpc.GenericExchangeNameResponse, error) {
	exchCfg, err := Bot.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	if r.AssetType != "" &&
		!exchCfg.CurrencyPairs.GetAssetTypes().Contains(asset.Item(r.AssetType)) {
		return nil, errors.New("specified asset type does not exist")
	}

	// Default to spot asset type unless set
	a := asset.Spot
	if r.AssetType != "" {
		a = asset.Item(r.AssetType)
	}

	pairFmt, err := Bot.Config.GetPairFormat(r.Exchange, a)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote).Format(
		pairFmt.Delimiter, pairFmt.Uppercase)
	err = exchCfg.CurrencyPairs.EnablePair(a, p)
	if err != nil {
		return nil, err
	}
	err = GetExchangeByName(r.Exchange).GetBase().CurrencyPairs.EnablePair(
		asset.Item(r.AssetType), p)
	return &gctrpc.GenericExchangeNameResponse{}, err
}

// DisableExchangePair disables the specified pair on an exchange
func (s *RPCServer) DisableExchangePair(ctx context.Context, r *gctrpc.ExchangePairRequest) (*gctrpc.GenericExchangeNameResponse, error) {
	exchCfg, err := Bot.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	if r.AssetType != "" &&
		!exchCfg.CurrencyPairs.GetAssetTypes().Contains(asset.Item(r.AssetType)) {
		return nil, errors.New("specified asset type does not exist")
	}

	// Default to spot asset type unless set
	a := asset.Spot
	if r.AssetType != "" {
		a = asset.Item(r.AssetType)
	}

	pairFmt, err := Bot.Config.GetPairFormat(r.Exchange, a)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote).Format(
		pairFmt.Delimiter, pairFmt.Uppercase)
	err = exchCfg.CurrencyPairs.DisablePair(asset.Item(r.AssetType), p)
	if err != nil {
		return nil, err
	}
	err = GetExchangeByName(r.Exchange).GetBase().CurrencyPairs.DisablePair(
		asset.Item(r.AssetType), p)
	return &gctrpc.GenericExchangeNameResponse{}, err
}

// GetOrderbookStream streams the requested updated orderbook
func (s *RPCServer) GetOrderbookStream(r *gctrpc.GetOrderbookStreamRequest, stream gctrpc.GoCryptoTrader_GetOrderbookStreamServer) error {
	if r.Exchange == "" {
		return errors.New(errExchangeNameUnset)
	}

	if r.Pair.String() == "" {
		return errors.New(errCurrencyPairUnset)
	}

	if r.AssetType == "" {
		return errors.New(errAssetTypeUnset)
	}

	p := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)

	pipe, err := orderbook.SubscribeOrderbook(r.Exchange, p, asset.Item(r.AssetType))
	if err != nil {
		return err
	}

	defer pipe.Release()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errors.New(errDispatchSystem)
		}

		ob := (*data.(*interface{})).(orderbook.Base)
		var bids, asks []*gctrpc.OrderbookItem
		for i := range ob.Bids {
			bids = append(bids, &gctrpc.OrderbookItem{
				Amount: ob.Bids[i].Amount,
				Price:  ob.Bids[i].Price,
				Id:     ob.Bids[i].ID,
			})
		}
		for i := range ob.Asks {
			asks = append(asks, &gctrpc.OrderbookItem{
				Amount: ob.Asks[i].Amount,
				Price:  ob.Asks[i].Price,
				Id:     ob.Asks[i].ID,
			})
		}
		err := stream.Send(&gctrpc.OrderbookResponse{
			Pair: &gctrpc.CurrencyPair{Base: ob.Pair.Base.String(),
				Quote: ob.Pair.Quote.String()},
			Bids:      bids,
			Asks:      asks,
			AssetType: ob.AssetType.String(),
		})
		if err != nil {
			return err
		}
	}
}

// GetExchangeOrderbookStream streams all orderbooks associated with an exchange
func (s *RPCServer) GetExchangeOrderbookStream(r *gctrpc.GetExchangeOrderbookStreamRequest, stream gctrpc.GoCryptoTrader_GetExchangeOrderbookStreamServer) error {
	if r.Exchange == "" {
		return errors.New(errExchangeNameUnset)
	}

	pipe, err := orderbook.SubscribeToExchangeOrderbooks(r.Exchange)
	if err != nil {
		return err
	}

	defer pipe.Release()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errors.New(errDispatchSystem)
		}

		ob := (*data.(*interface{})).(orderbook.Base)
		var bids, asks []*gctrpc.OrderbookItem
		for i := range ob.Bids {
			bids = append(bids, &gctrpc.OrderbookItem{
				Amount: ob.Bids[i].Amount,
				Price:  ob.Bids[i].Price,
				Id:     ob.Bids[i].ID,
			})
		}
		for i := range ob.Asks {
			asks = append(asks, &gctrpc.OrderbookItem{
				Amount: ob.Asks[i].Amount,
				Price:  ob.Asks[i].Price,
				Id:     ob.Asks[i].ID,
			})
		}
		err := stream.Send(&gctrpc.OrderbookResponse{
			Pair: &gctrpc.CurrencyPair{Base: ob.Pair.Base.String(),
				Quote: ob.Pair.Quote.String()},
			Bids:      bids,
			Asks:      asks,
			AssetType: ob.AssetType.String(),
		})
		if err != nil {
			return err
		}
	}
}

// GetTickerStream streams the requested updated ticker
func (s *RPCServer) GetTickerStream(r *gctrpc.GetTickerStreamRequest, stream gctrpc.GoCryptoTrader_GetTickerStreamServer) error {
	if r.Exchange == "" {
		return errors.New(errExchangeNameUnset)
	}

	if r.Pair.String() == "" {
		return errors.New(errCurrencyPairUnset)
	}

	if r.AssetType == "" {
		return errors.New(errAssetTypeUnset)
	}

	p := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)

	pipe, err := ticker.SubscribeTicker(r.Exchange, p, asset.Item(r.AssetType))
	if err != nil {
		return err
	}

	defer pipe.Release()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errors.New(errDispatchSystem)
		}
		t := (*data.(*interface{})).(ticker.Price)

		err := stream.Send(&gctrpc.TickerResponse{
			Pair: &gctrpc.CurrencyPair{
				Base:      t.Pair.Base.String(),
				Quote:     t.Pair.Quote.String(),
				Delimiter: t.Pair.Delimiter},
			LastUpdated: t.LastUpdated.Unix(),
			Last:        t.Last,
			High:        t.High,
			Low:         t.Low,
			Bid:         t.Bid,
			Ask:         t.Ask,
			Volume:      t.Volume,
			PriceAth:    t.PriceATH,
		})
		if err != nil {
			return err
		}
	}
}

// GetExchangeTickerStream streams all tickers associated with an exchange
func (s *RPCServer) GetExchangeTickerStream(r *gctrpc.GetExchangeTickerStreamRequest, stream gctrpc.GoCryptoTrader_GetExchangeTickerStreamServer) error {
	if r.Exchange == "" {
		return errors.New(errExchangeNameUnset)
	}

	pipe, err := ticker.SubscribeToExchangeTickers(r.Exchange)
	if err != nil {
		return err
	}

	defer pipe.Release()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errors.New(errDispatchSystem)
		}
		t := (*data.(*interface{})).(ticker.Price)

		err := stream.Send(&gctrpc.TickerResponse{
			Pair: &gctrpc.CurrencyPair{
				Base:      t.Pair.Base.String(),
				Quote:     t.Pair.Quote.String(),
				Delimiter: t.Pair.Delimiter},
			LastUpdated: t.LastUpdated.Unix(),
			Last:        t.Last,
			High:        t.High,
			Low:         t.Low,
			Bid:         t.Bid,
			Ask:         t.Ask,
			Volume:      t.Volume,
			PriceAth:    t.PriceATH,
		})
		if err != nil {
			return err
		}
	}
}

// GetAuditEvent returns matching audit events from database
func (s *RPCServer) GetAuditEvent(ctx context.Context, r *gctrpc.GetAuditEventRequest) (*gctrpc.GetAuditEventResponse, error) {
	UTCStartTime, err := time.Parse(audit.TableTimeFormat, r.StartDate)
	if err != nil {
		return nil, err
	}

	UTCSEndTime, err := time.Parse(audit.TableTimeFormat, r.EndDate)
	if err != nil {
		return nil, err
	}

	loc := time.FixedZone("", int(r.Offset))

	events, err := audit.GetEvent(UTCStartTime, UTCSEndTime, r.OrderBy, int(r.Limit))
	if err != nil {
		return nil, err
	}

	resp := gctrpc.GetAuditEventResponse{}

	switch v := events.(type) {
	case postgres.AuditEventSlice:
		for x := range v {
			tempEvent := &gctrpc.AuditEvent{
				Type:       v[x].Type,
				Identifier: v[x].Identifier,
				Message:    v[x].Message,
				Timestamp:  v[x].CreatedAt.In(loc).Format(audit.TableTimeFormat),
			}

			resp.Events = append(resp.Events, tempEvent)
		}
	case sqlite3.AuditEventSlice:
		for x := range v {
			tempEvent := &gctrpc.AuditEvent{
				Type:       v[x].Type,
				Identifier: v[x].Identifier,
				Message:    v[x].Message,
				Timestamp:  v[x].CreatedAt,
			}
			resp.Events = append(resp.Events, tempEvent)
		}
	}

	return &resp, nil
}

// GetHistoricCandles returns historical candles for a given exchange
func (s *RPCServer) GetHistoricCandles(context.Context, *gctrpc.GetHistoricCandlesRequest) (*gctrpc.GetHistoricCandlesResponse, error) {
	resp := gctrpc.GetHistoricCandlesResponse{}

	return &resp, nil
}