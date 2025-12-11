package engine

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pquerna/otp/totp"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/common/file/archive"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/common/timeperiods"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	gctscript "github.com/thrasher-corp/gocryptotrader/gctscript/vm"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errExchangeNotLoaded       = errors.New("exchange is not loaded/doesn't exist")
	errExchangeNotEnabled      = errors.New("exchange is not enabled")
	errExchangeBaseNotFound    = errors.New("cannot get exchange base")
	errInvalidArguments        = errors.New("invalid arguments received")
	errCurrencyPairUnset       = errors.New("currency pair unset")
	errInvalidTimes            = errors.New("invalid start and end times")
	errAssetTypeUnset          = errors.New("asset type unset")
	errDispatchSystem          = errors.New("dispatch system offline")
	errCurrencyNotEnabled      = errors.New("currency not enabled")
	errCurrencyNotSpecified    = errors.New("a currency must be specified")
	errCurrencyPairInvalid     = errors.New("currency provided is not found in the available pairs list")
	errNoTrades                = errors.New("no trades returned from supplied params")
	errNilRequestData          = errors.New("nil request data received, cannot continue")
	errShutdownNotAllowed      = errors.New("shutting down this bot instance is not allowed via gRPC, please enable by command line flag --grpcshutdown or config.json field grpcAllowBotShutdown")
	errGRPCShutdownSignalIsNil = errors.New("cannot shutdown, gRPC shutdown channel is nil")
	errInvalidStrategy         = errors.New("invalid strategy")
	errSpecificPairNotEnabled  = errors.New("specified pair is not enabled")
)

// RPCServer struct
type RPCServer struct {
	gctrpc.UnimplementedGoCryptoTraderServiceServer
	*Engine
}

func (s *RPCServer) authenticateClient(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, errors.New("unable to extract metadata")
	}

	authStr, ok := md["authorization"]
	if !ok {
		return ctx, errors.New("authorization header missing")
	}

	if !strings.Contains(authStr[0], "Basic") {
		return ctx, errors.New("basic not found in authorization header")
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.Split(authStr[0], " ")[1])
	if err != nil {
		return ctx, errors.New("unable to base64 decode authorization header")
	}

	cred := strings.Split(string(decoded), ":")
	username := cred[0]
	password := cred[1]

	if username != s.Config.RemoteControl.Username ||
		password != s.Config.RemoteControl.Password {
		return ctx, errors.New("username/password mismatch")
	}
	ctx, err = accounts.ParseCredentialsMetadata(ctx, md)
	if err != nil {
		return ctx, err
	}

	if _, ok := md["verbose"]; ok {
		ctx = request.WithVerbose(ctx)
	}
	return ctx, nil
}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer(engine *Engine) {
	targetDir := utils.GetTLSDir(engine.Settings.DataDir)
	if err := CheckCerts(targetDir); err != nil {
		log.Errorf(log.GRPCSys, "gRPC CheckCerts failed. err: %s\n", err)
		return
	}
	log.Debugf(log.GRPCSys, "gRPC server support enabled. Starting gRPC server on https://%v.\n", engine.Config.RemoteControl.GRPC.ListenAddress)
	lis, err := net.Listen("tcp", engine.Config.RemoteControl.GRPC.ListenAddress) //nolint:noctx // TODO: #2006 Replace net.Listen with (*net.ListenConfig).Listen
	if err != nil {
		log.Errorf(log.GRPCSys, "gRPC server failed to bind to port: %s", err)
		return
	}

	creds, err := credentials.NewServerTLSFromFile(filepath.Join(targetDir, "cert.pem"), filepath.Join(targetDir, "key.pem"))
	if err != nil {
		log.Errorf(log.GRPCSys, "gRPC server could not load TLS keys: %s\n", err)
		return
	}

	s := RPCServer{Engine: engine}
	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(grpcauth.UnaryServerInterceptor(s.authenticateClient)),
		grpc.StreamInterceptor(grpcauth.StreamServerInterceptor(s.authenticateClient)),
	}
	server := grpc.NewServer(opts...)
	gctrpc.RegisterGoCryptoTraderServiceServer(server, &s)

	go func() {
		if err := server.Serve(lis); err != nil {
			log.Errorf(log.GRPCSys, "gRPC server failed to serve: %s\n", err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC server started!")

	if s.Settings.EnableGRPCProxy {
		s.StartRPCRESTProxy()
	}
}

// StartRPCRESTProxy starts a gRPC proxy
func (s *RPCServer) StartRPCRESTProxy() {
	log.Debugf(log.GRPCSys, "gRPC proxy server support enabled. Starting gRPC proxy server on https://%v.\n", s.Config.RemoteControl.GRPC.GRPCProxyListenAddress)

	targetDir := utils.GetTLSDir(s.Settings.DataDir)
	certFile := filepath.Join(targetDir, "cert.pem")
	keyFile := filepath.Join(targetDir, "key.pem")
	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		log.Errorf(log.GRPCSys, "Unable to start gRPC proxy. Err: %s\n", err)
		return
	}

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: s.Config.RemoteControl.Username,
			Password: s.Config.RemoteControl.Password,
		}),
	}
	err = gctrpc.RegisterGoCryptoTraderServiceHandlerFromEndpoint(context.Background(),
		mux, s.Config.RemoteControl.GRPC.ListenAddress, opts)
	if err != nil {
		log.Errorf(log.GRPCSys, "Failed to register gRPC proxy. Err: %s\n", err)
		return
	}

	go func() {
		server := &http.Server{
			Addr:              s.Config.RemoteControl.GRPC.GRPCProxyListenAddress,
			ReadHeaderTimeout: time.Minute,
			ReadTimeout:       time.Minute,
			Handler:           s.authClient(mux),
		}

		if err = server.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Errorf(log.GRPCSys, "gRPC proxy server failed to serve: %s\n", err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC proxy server started!")
}

func (s *RPCServer) authClient(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != s.Config.RemoteControl.Username || password != s.Config.RemoteControl.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "Access denied", http.StatusUnauthorized)
			log.Warnf(log.GRPCSys, "gRPC proxy server unauthorised access attempt. IP: %s Path: %s\n", r.RemoteAddr, r.URL.Path)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// GetInfo returns info about the current GoCryptoTrader session
func (s *RPCServer) GetInfo(_ context.Context, _ *gctrpc.GetInfoRequest) (*gctrpc.GetInfoResponse, error) {
	rpcEndpoints, err := s.getRPCEndpoints()
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetInfoResponse{
		Uptime:               time.Since(s.uptime).String(),
		EnabledExchanges:     int64(s.Config.CountEnabledExchanges()),
		AvailableExchanges:   int64(len(s.Config.Exchanges)),
		DefaultFiatCurrency:  s.Config.Currency.FiatDisplayCurrency.String(),
		DefaultForexProvider: s.Config.GetPrimaryForexProvider(),
		SubsystemStatus:      s.GetSubsystemsStatus(),
		RpcEndpoints:         rpcEndpoints,
	}, nil
}

func (s *RPCServer) getRPCEndpoints() (map[string]*gctrpc.RPCEndpoint, error) {
	endpoints, err := s.Engine.GetRPCEndpoints()
	if err != nil {
		return nil, err
	}
	rpcEndpoints := make(map[string]*gctrpc.RPCEndpoint)
	for key, val := range endpoints {
		rpcEndpoints[key] = &gctrpc.RPCEndpoint{
			Started:       val.Started,
			ListenAddress: val.ListenAddr,
		}
	}
	return rpcEndpoints, nil
}

// GetSubsystems returns a list of subsystems and their status
func (s *RPCServer) GetSubsystems(_ context.Context, _ *gctrpc.GetSubsystemsRequest) (*gctrpc.GetSusbsytemsResponse, error) {
	return &gctrpc.GetSusbsytemsResponse{SubsystemsStatus: s.GetSubsystemsStatus()}, nil
}

// EnableSubsystem enables a engine subsystem
func (s *RPCServer) EnableSubsystem(_ context.Context, r *gctrpc.GenericSubsystemRequest) (*gctrpc.GenericResponse, error) {
	err := s.SetSubsystem(r.Subsystem, true)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{
		Status: MsgStatusSuccess,
		Data:   fmt.Sprintf("subsystem %s enabled", r.Subsystem),
	}, nil
}

// DisableSubsystem disables a engine subsystem
func (s *RPCServer) DisableSubsystem(_ context.Context, r *gctrpc.GenericSubsystemRequest) (*gctrpc.GenericResponse, error) {
	err := s.SetSubsystem(r.Subsystem, false)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{
		Status: MsgStatusSuccess,
		Data:   fmt.Sprintf("subsystem %s disabled", r.Subsystem),
	}, nil
}

// GetRPCEndpoints returns a list of API endpoints
func (s *RPCServer) GetRPCEndpoints(_ context.Context, _ *gctrpc.GetRPCEndpointsRequest) (*gctrpc.GetRPCEndpointsResponse, error) {
	endpoint, err := s.getRPCEndpoints()
	return &gctrpc.GetRPCEndpointsResponse{Endpoints: endpoint}, err
}

// GetCommunicationRelayers returns the status of the engines communication relayers
func (s *RPCServer) GetCommunicationRelayers(_ context.Context, _ *gctrpc.GetCommunicationRelayersRequest) (*gctrpc.GetCommunicationRelayersResponse, error) {
	relayers, err := s.CommunicationsManager.GetStatus()
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
func (s *RPCServer) GetExchanges(_ context.Context, r *gctrpc.GetExchangesRequest) (*gctrpc.GetExchangesResponse, error) {
	exchanges := strings.Join(s.GetExchangeNames(r.Enabled), ",")
	return &gctrpc.GetExchangesResponse{Exchanges: exchanges}, nil
}

// DisableExchange disables an exchange
func (s *RPCServer) DisableExchange(_ context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GenericResponse, error) {
	err := s.UnloadExchange(r.Exchange)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// EnableExchange enables an exchange
func (s *RPCServer) EnableExchange(_ context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GenericResponse, error) {
	err := s.LoadExchange(r.Exchange)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// GetExchangeOTPCode retrieves an exchanges OTP code
func (s *RPCServer) GetExchangeOTPCode(_ context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GetExchangeOTPResponse, error) {
	if _, err := s.GetExchangeByName(r.Exchange); err != nil {
		return nil, err
	}
	result, err := s.GetExchangeOTPByName(r.Exchange)
	return &gctrpc.GetExchangeOTPResponse{OtpCode: result}, err
}

// GetExchangeOTPCodes retrieves OTP codes for all exchanges which have an
// OTP secret installed
func (s *RPCServer) GetExchangeOTPCodes(_ context.Context, _ *gctrpc.GetExchangeOTPsRequest) (*gctrpc.GetExchangeOTPsResponse, error) {
	result, err := s.GetExchangeOTPs()
	return &gctrpc.GetExchangeOTPsResponse{OtpCodes: result}, err
}

// GetExchangeInfo gets info for a specific exchange
func (s *RPCServer) GetExchangeInfo(_ context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GetExchangeInfoResponse, error) {
	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
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
	assets := exchCfg.CurrencyPairs.GetAssetTypes(false)
	for i := range assets {
		var enabled currency.Pairs
		enabled, err = exchCfg.CurrencyPairs.GetPairs(assets[i], true)
		if err != nil {
			return nil, err
		}

		var available currency.Pairs
		available, err = exchCfg.CurrencyPairs.GetPairs(assets[i], false)
		if err != nil {
			return nil, err
		}

		resp.SupportedAssets[assets[i].String()] = &gctrpc.PairsSupported{
			EnabledPairs:   enabled.Join(),
			AvailablePairs: available.Join(),
		}
	}
	return resp, nil
}

// GetTicker returns the ticker for a specified exchange, currency pair and
// asset type
func (s *RPCServer) GetTicker(_ context.Context, r *gctrpc.GetTickerRequest) (*gctrpc.TickerResponse, error) {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	e, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	pair := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, e, a, pair)
	if err != nil {
		return nil, err
	}

	t, err := e.GetCachedTicker(pair, a)
	if err != nil {
		return nil, err
	}

	resp := &gctrpc.TickerResponse{
		Pair:        r.Pair,
		LastUpdated: s.unixTimestamp(t.LastUpdated),
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
func (s *RPCServer) GetTickers(_ context.Context, _ *gctrpc.GetTickersRequest) (*gctrpc.GetTickersResponse, error) {
	activeTickers := s.GetAllActiveTickers()
	tickers := make([]*gctrpc.Tickers, len(activeTickers))
	for x := range activeTickers {
		ticks := make([]*gctrpc.TickerResponse, len(activeTickers[x].ExchangeValues))
		for y, val := range activeTickers[x].ExchangeValues {
			ticks[y] = &gctrpc.TickerResponse{
				Pair: &gctrpc.CurrencyPair{
					Delimiter: val.Pair.Delimiter,
					Base:      val.Pair.Base.String(),
					Quote:     val.Pair.Quote.String(),
				},
				LastUpdated: s.unixTimestamp(val.LastUpdated),
				Last:        val.Last,
				High:        val.High,
				Low:         val.Low,
				Bid:         val.Bid,
				Ask:         val.Ask,
				Volume:      val.Volume,
				PriceAth:    val.PriceATH,
			}
		}
		tickers[x] = &gctrpc.Tickers{Exchange: activeTickers[x].ExchangeName, Tickers: ticks}
	}

	return &gctrpc.GetTickersResponse{Tickers: tickers}, nil
}

// GetOrderbook returns an orderbook for a specific exchange, currency pair
// and asset type
func (s *RPCServer) GetOrderbook(_ context.Context, r *gctrpc.GetOrderbookRequest) (*gctrpc.OrderbookResponse, error) {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	e, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	pair := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	ob, err := e.GetCachedOrderbook(pair, a)
	if err != nil {
		return nil, err
	}

	bids := make([]*gctrpc.OrderbookItem, len(ob.Bids))
	for x := range ob.Bids {
		bids[x] = &gctrpc.OrderbookItem{Amount: ob.Bids[x].Amount, Price: ob.Bids[x].Price}
	}

	asks := make([]*gctrpc.OrderbookItem, len(ob.Asks))
	for x := range ob.Asks {
		asks[x] = &gctrpc.OrderbookItem{Amount: ob.Asks[x].Amount, Price: ob.Asks[x].Price}
	}

	resp := &gctrpc.OrderbookResponse{
		Pair:        r.Pair,
		Bids:        bids,
		Asks:        asks,
		LastUpdated: s.unixTimestamp(ob.LastUpdated),
		AssetType:   r.AssetType,
	}

	return resp, nil
}

// GetOrderbooks returns a list of orderbooks for all enabled exchanges and all
// enabled currency pairs
func (s *RPCServer) GetOrderbooks(_ context.Context, _ *gctrpc.GetOrderbooksRequest) (*gctrpc.GetOrderbooksResponse, error) {
	exchanges, err := s.ExchangeManager.GetExchanges()
	if err != nil {
		return nil, err
	}
	obResponse := make([]*gctrpc.Orderbooks, 0, len(exchanges))
	var obs []*gctrpc.OrderbookResponse
	for _, e := range exchanges {
		if !e.IsEnabled() {
			continue
		}
		for _, a := range e.GetAssetTypes(true) {
			pairs, err := e.GetEnabledPairs(a)
			if err != nil {
				log.Errorf(log.RESTSys, "Exchange %s could not retrieve enabled currencies. Err: %s\n", e.GetName(), err)
				continue
			}
			for _, pair := range pairs {
				resp, err := e.GetCachedOrderbook(pair, a)
				if err != nil {
					log.Errorf(log.RESTSys, "Exchange %s failed to retrieve %s orderbook. Err: %s\n", e.GetName(), pair, err)
					continue
				}
				ob := &gctrpc.OrderbookResponse{
					Pair: &gctrpc.CurrencyPair{
						Delimiter: pair.Delimiter,
						Base:      pair.Base.String(),
						Quote:     pair.Quote.String(),
					},
					AssetType:   a.String(),
					LastUpdated: s.unixTimestamp(resp.LastUpdated),
					Bids:        make([]*gctrpc.OrderbookItem, len(resp.Bids)),
					Asks:        make([]*gctrpc.OrderbookItem, len(resp.Asks)),
				}
				for i := range resp.Bids {
					ob.Bids[i] = &gctrpc.OrderbookItem{Amount: resp.Bids[i].Amount, Price: resp.Bids[i].Price}
				}
				for i := range resp.Asks {
					ob.Asks[i] = &gctrpc.OrderbookItem{Amount: resp.Asks[i].Amount, Price: resp.Asks[i].Price}
				}
				obs = append(obs, ob)
			}
		}
		obResponse = append(obResponse, &gctrpc.Orderbooks{Exchange: e.GetName(), Orderbooks: obs})
	}

	return &gctrpc.GetOrderbooksResponse{Orderbooks: obResponse}, nil
}

// GetAccountBalances returns an account balance for a specific exchange.
func (s *RPCServer) GetAccountBalances(ctx context.Context, r *gctrpc.GetAccountBalancesRequest) (*gctrpc.GetAccountBalancesResponse, error) {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	e, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	if err := checkParams(r.Exchange, e, assetType, currency.EMPTYPAIR); err != nil {
		return nil, err
	}

	resp, err := e.GetCachedSubAccounts(ctx, assetType)
	if err != nil {
		return nil, err
	}

	return accountBalanceResp(r.Exchange, resp), nil
}

// UpdateAccountBalances forces an update of the account balances.
func (s *RPCServer) UpdateAccountBalances(ctx context.Context, r *gctrpc.GetAccountBalancesRequest) (*gctrpc.GetAccountBalancesResponse, error) {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	e, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	if err := checkParams(r.Exchange, e, assetType, currency.EMPTYPAIR); err != nil {
		return nil, err
	}

	resp, err := e.UpdateAccountBalances(ctx, assetType)
	if err != nil {
		return nil, err
	}

	return accountBalanceResp(r.Exchange, resp), nil
}

func accountBalanceResp(eName string, s accounts.SubAccounts) *gctrpc.GetAccountBalancesResponse {
	subAccts := make([]*gctrpc.Account, len(s))
	for i, sa := range s {
		subAccts[i] = &gctrpc.Account{
			Id: sa.ID,
		}
		for curr, bal := range sa.Balances {
			subAccts[i].Currencies = append(subAccts[i].Currencies, &gctrpc.AccountCurrencyInfo{
				Currency:          curr.String(),
				TotalValue:        bal.Total,
				Hold:              bal.Hold,
				Free:              bal.Free,
				FreeWithoutBorrow: bal.AvailableWithoutBorrow,
				Borrowed:          bal.Borrowed,
				UpdatedAt:         timestamppb.New(bal.UpdatedAt),
			})
		}
	}
	return &gctrpc.GetAccountBalancesResponse{
		Exchange: eName,
		Accounts: subAccts,
	}
}

// GetAccountBalancesStream streams an account balance for a specific exchange
func (s *RPCServer) GetAccountBalancesStream(r *gctrpc.GetAccountBalancesRequest, stream gctrpc.GoCryptoTraderService_GetAccountBalancesStreamServer) error {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return err
	}

	err = checkParams(r.Exchange, exch, assetType, currency.EMPTYPAIR)
	if err != nil {
		return err
	}

	pipe, err := exch.SubscribeAccountBalances()
	if err != nil {
		return err
	}

	defer func() {
		pipeErr := pipe.Release()
		if pipeErr != nil {
			log.Errorln(log.DispatchMgr, pipeErr)
		}
	}()
	init := make(chan struct{}, 1)
	init <- struct{}{}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case _, ok := <-pipe.Channel():
			if !ok {
				return errDispatchSystem
			}
		case <-init:
		}

		subAccts, err := exch.GetCachedSubAccounts(stream.Context(), assetType)
		if err != nil {
			return err
		}

		if err := stream.Send(accountBalanceResp(r.Exchange, subAccts)); err != nil {
			return err
		}
	}
}

// GetConfig returns the bots config
func (s *RPCServer) GetConfig(_ context.Context, _ *gctrpc.GetConfigRequest) (*gctrpc.GetConfigResponse, error) {
	return &gctrpc.GetConfigResponse{}, common.ErrNotYetImplemented
}

// GetPortfolio returns the portfoliomanager details
func (s *RPCServer) GetPortfolio(_ context.Context, _ *gctrpc.GetPortfolioRequest) (*gctrpc.GetPortfolioResponse, error) {
	botAddrs := s.portfolioManager.GetAddresses()
	addrs := make([]*gctrpc.PortfolioAddress, len(botAddrs))
	for x := range botAddrs {
		addrs[x] = &gctrpc.PortfolioAddress{
			Address:     botAddrs[x].Address,
			CoinType:    botAddrs[x].CoinType.String(),
			Description: botAddrs[x].Description,
			Balance:     botAddrs[x].Balance,
		}
	}

	resp := &gctrpc.GetPortfolioResponse{
		Portfolio: addrs,
	}

	return resp, nil
}

// GetPortfolioSummary returns the portfoliomanager summary
func (s *RPCServer) GetPortfolioSummary(_ context.Context, _ *gctrpc.GetPortfolioSummaryRequest) (*gctrpc.GetPortfolioSummaryResponse, error) {
	result := s.portfolioManager.GetPortfolioSummary()
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

// AddPortfolioAddress adds an address to the portfoliomanager manager
func (s *RPCServer) AddPortfolioAddress(_ context.Context, r *gctrpc.AddPortfolioAddressRequest) (*gctrpc.GenericResponse, error) {
	err := s.portfolioManager.AddAddress(r.Address,
		r.Description,
		currency.NewCode(r.CoinType),
		r.Balance)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// RemovePortfolioAddress removes an address from the portfoliomanager manager
func (s *RPCServer) RemovePortfolioAddress(_ context.Context, r *gctrpc.RemovePortfolioAddressRequest) (*gctrpc.GenericResponse, error) {
	err := s.portfolioManager.RemoveAddress(r.Address,
		r.Description,
		currency.NewCode(r.CoinType))
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// GetForexProviders returns a list of available forex providers
func (s *RPCServer) GetForexProviders(_ context.Context, _ *gctrpc.GetForexProvidersRequest) (*gctrpc.GetForexProvidersResponse, error) {
	providers := s.Config.GetForexProviders()
	if len(providers) == 0 {
		return nil, errors.New("forex providers is empty")
	}

	forexProviders := make([]*gctrpc.ForexProvider, len(providers))
	for x := range providers {
		forexProviders[x] = &gctrpc.ForexProvider{
			Name:             providers[x].Name,
			Enabled:          providers[x].Enabled,
			Verbose:          providers[x].Verbose,
			RestPollingDelay: s.Config.Currency.ForeignExchangeUpdateDuration.String(),
			ApiKey:           providers[x].APIKey,
			ApiKeyLevel:      int64(providers[x].APIKeyLvl),
			PrimaryProvider:  providers[x].PrimaryProvider,
		}
	}
	return &gctrpc.GetForexProvidersResponse{ForexProviders: forexProviders}, nil
}

// GetForexRates returns a list of forex rates
func (s *RPCServer) GetForexRates(_ context.Context, _ *gctrpc.GetForexRatesRequest) (*gctrpc.GetForexRatesResponse, error) {
	rates, err := currency.GetExchangeRates()
	if err != nil {
		return nil, err
	}

	if len(rates) == 0 {
		return nil, errors.New("forex rates is empty")
	}

	forexRates := make([]*gctrpc.ForexRatesConversion, 0, len(rates))
	for x := range rates {
		rate, err := rates[x].GetRate()
		if err != nil {
			continue
		}

		// TODO add inverse rate
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
// asset type between optional dates
func (s *RPCServer) GetOrders(ctx context.Context, r *gctrpc.GetOrdersRequest) (*gctrpc.GetOrdersResponse, error) {
	if r == nil {
		return nil, errInvalidArguments
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}
	cp := currency.NewPairWithDelimiter(
		r.Pair.Base,
		r.Pair.Quote,
		r.Pair.Delimiter)

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, a, cp)
	if err != nil {
		return nil, err
	}

	var start, end time.Time
	if r.StartDate != "" {
		start, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.StartDate)
		if err != nil {
			return nil, err
		}
	}
	if r.EndDate != "" {
		end, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.EndDate)
		if err != nil {
			return nil, err
		}
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}

	req := &order.MultiOrderRequest{
		Pairs:     []currency.Pair{cp},
		AssetType: a,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	if !start.IsZero() {
		req.StartTime = start
	}
	if !end.IsZero() {
		req.EndTime = end
	}

	var resp []order.Detail
	resp, err = exch.GetActiveOrders(ctx, req)
	if err != nil {
		return nil, err
	}

	orders := make([]*gctrpc.OrderDetails, len(resp))
	for x := range resp {
		trades := make([]*gctrpc.TradeHistory, len(resp[x].Trades))
		for i := range resp[x].Trades {
			t := &gctrpc.TradeHistory{
				Id:        resp[x].Trades[i].TID,
				Price:     resp[x].Trades[i].Price,
				Amount:    resp[x].Trades[i].Amount,
				Exchange:  r.Exchange,
				AssetType: a.String(),
				OrderSide: resp[x].Trades[i].Side.String(),
				Fee:       resp[x].Trades[i].Fee,
				Total:     resp[x].Trades[i].Total,
			}
			if !resp[x].Trades[i].Timestamp.IsZero() {
				t.CreationTime = s.unixTimestamp(resp[x].Trades[i].Timestamp)
			}
			trades[i] = t
		}
		o := &gctrpc.OrderDetails{
			Exchange:      r.Exchange,
			Id:            resp[x].OrderID,
			ClientOrderId: resp[x].ClientOrderID,
			BaseCurrency:  resp[x].Pair.Base.String(),
			QuoteCurrency: resp[x].Pair.Quote.String(),
			AssetType:     resp[x].AssetType.String(),
			OrderSide:     resp[x].Side.String(),
			OrderType:     resp[x].Type.String(),
			Status:        resp[x].Status.String(),
			Price:         resp[x].Price,
			Amount:        resp[x].Amount,
			OpenVolume:    resp[x].Amount - resp[x].ExecutedAmount,
			Fee:           resp[x].Fee,
			Cost:          resp[x].Cost,
			Trades:        trades,
		}
		if !resp[x].Date.IsZero() {
			o.CreationTime = resp[x].Date.Format(common.SimpleTimeFormatWithTimezone)
		}
		if !resp[x].LastUpdated.IsZero() {
			o.UpdateTime = resp[x].LastUpdated.Format(common.SimpleTimeFormatWithTimezone)
		}
		orders[x] = o
	}

	return &gctrpc.GetOrdersResponse{Orders: orders}, nil
}

// GetManagedOrders returns all orders from the Order Manager for the provided exchange,
// asset type  and currency pair
func (s *RPCServer) GetManagedOrders(_ context.Context, r *gctrpc.GetOrdersRequest) (*gctrpc.GetOrdersResponse, error) {
	if r == nil {
		return nil, errInvalidArguments
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}
	cp := currency.NewPairWithDelimiter(
		r.Pair.Base,
		r.Pair.Quote,
		r.Pair.Delimiter)

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, a, cp)
	if err != nil {
		return nil, err
	}

	var resp []order.Detail
	filter := order.Filter{
		Exchange:  exch.GetName(),
		Pair:      cp,
		AssetType: a,
	}
	resp, err = s.OrderManager.GetOrdersFiltered(&filter)
	if err != nil {
		return nil, err
	}

	orders := make([]*gctrpc.OrderDetails, len(resp))
	for x := range resp {
		trades := make([]*gctrpc.TradeHistory, len(resp[x].Trades))
		for i := range resp[x].Trades {
			t := &gctrpc.TradeHistory{
				Id:        resp[x].Trades[i].TID,
				Price:     resp[x].Trades[i].Price,
				Amount:    resp[x].Trades[i].Amount,
				Exchange:  r.Exchange,
				AssetType: a.String(),
				OrderSide: resp[x].Trades[i].Side.String(),
				Fee:       resp[x].Trades[i].Fee,
				Total:     resp[x].Trades[i].Total,
			}
			if !resp[x].Trades[i].Timestamp.IsZero() {
				t.CreationTime = s.unixTimestamp(resp[x].Trades[i].Timestamp)
			}
			trades[i] = t
		}
		o := &gctrpc.OrderDetails{
			Exchange:      r.Exchange,
			Id:            resp[x].OrderID,
			ClientOrderId: resp[x].ClientOrderID,
			BaseCurrency:  resp[x].Pair.Base.String(),
			QuoteCurrency: resp[x].Pair.Quote.String(),
			AssetType:     resp[x].AssetType.String(),
			OrderSide:     resp[x].Side.String(),
			OrderType:     resp[x].Type.String(),
			Status:        resp[x].Status.String(),
			Price:         resp[x].Price,
			Amount:        resp[x].Amount,
			OpenVolume:    resp[x].Amount - resp[x].ExecutedAmount,
			Fee:           resp[x].Fee,
			Cost:          resp[x].Cost,
			Trades:        trades,
		}
		if !resp[x].Date.IsZero() {
			o.CreationTime = resp[x].Date.Format(common.SimpleTimeFormatWithTimezone)
		}
		if !resp[x].LastUpdated.IsZero() {
			o.UpdateTime = resp[x].LastUpdated.Format(common.SimpleTimeFormatWithTimezone)
		}
		orders[x] = o
	}

	return &gctrpc.GetOrdersResponse{Orders: orders}, nil
}

// GetOrder returns order information based on exchange and order ID
func (s *RPCServer) GetOrder(ctx context.Context, r *gctrpc.GetOrderRequest) (*gctrpc.OrderDetails, error) {
	if r == nil {
		return nil, errInvalidArguments
	}

	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	pair := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, pair)
	if err != nil {
		return nil, err
	}

	result, err := s.OrderManager.GetOrderInfo(ctx,
		r.Exchange,
		r.OrderId,
		pair,
		a)
	if err != nil {
		return nil, fmt.Errorf("error whilst trying to retrieve info for order %s: %w", r.OrderId, err)
	}
	trades := make([]*gctrpc.TradeHistory, len(result.Trades))
	for i := range result.Trades {
		trades[i] = &gctrpc.TradeHistory{
			CreationTime: s.unixTimestamp(result.Trades[i].Timestamp),
			Id:           result.Trades[i].TID,
			Price:        result.Trades[i].Price,
			Amount:       result.Trades[i].Amount,
			Exchange:     result.Trades[i].Exchange,
			AssetType:    result.Trades[i].Type.String(),
			OrderSide:    result.Trades[i].Side.String(),
			Fee:          result.Trades[i].Fee,
			Total:        result.Trades[i].Total,
		}
	}

	var creationTime, updateTime string
	if !result.Date.IsZero() {
		creationTime = result.Date.Format(common.SimpleTimeFormatWithTimezone)
	}
	if !result.LastUpdated.IsZero() {
		updateTime = result.LastUpdated.Format(common.SimpleTimeFormatWithTimezone)
	}

	return &gctrpc.OrderDetails{
		Exchange:      result.Exchange,
		Id:            result.OrderID,
		ClientOrderId: result.ClientOrderID,
		BaseCurrency:  result.Pair.Base.String(),
		QuoteCurrency: result.Pair.Quote.String(),
		AssetType:     result.AssetType.String(),
		OrderSide:     result.Side.String(),
		OrderType:     result.Type.String(),
		CreationTime:  creationTime,
		Status:        result.Status.String(),
		Price:         result.Price,
		Amount:        result.Amount,
		OpenVolume:    result.RemainingAmount,
		Fee:           result.Fee,
		Trades:        trades,
		Cost:          result.Cost,
		UpdateTime:    updateTime,
	}, err
}

// SubmitOrder submits an order specified by exchange, currency pair and asset type
func (s *RPCServer) SubmitOrder(ctx context.Context, r *gctrpc.SubmitOrderRequest) (*gctrpc.SubmitOrderResponse, error) {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	var marginType margin.Type
	if r.MarginType != "" {
		marginType, err = margin.StringToMarginType(r.MarginType)
		if err != nil {
			return nil, err
		}
	}
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	side, err := order.StringToOrderSide(r.Side)
	if err != nil {
		return nil, err
	}

	oType, err := order.StringToOrderType(r.OrderType)
	if err != nil {
		return nil, err
	}

	submission := &order.Submit{
		Pair:          p,
		Side:          side,
		Type:          oType,
		Amount:        r.Amount,
		Price:         r.Price,
		ClientID:      r.ClientId,
		ClientOrderID: r.ClientId,
		Exchange:      r.Exchange,
		AssetType:     a,
	}
	if r.MarginType != "" {
		submission.MarginType = marginType
	}

	resp, err := s.OrderManager.Submit(ctx, submission)
	if err != nil {
		return &gctrpc.SubmitOrderResponse{}, err
	}

	trades := make([]*gctrpc.Trades, len(resp.Trades))
	for i := range resp.Trades {
		trades[i] = &gctrpc.Trades{
			Amount:   resp.Trades[i].Amount,
			Price:    resp.Trades[i].Price,
			Fee:      resp.Trades[i].Fee,
			FeeAsset: resp.Trades[i].FeeAsset,
		}
	}

	return &gctrpc.SubmitOrderResponse{
		OrderId:     resp.OrderID,
		OrderPlaced: resp.WasOrderPlaced(),
		Trades:      trades,
	}, nil
}

// SimulateOrder simulates an order specified by exchange, currency pair and asset
// type
func (s *RPCServer) SimulateOrder(_ context.Context, r *gctrpc.SimulateOrderRequest) (*gctrpc.SimulateOrderResponse, error) {
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, asset.Spot, p)
	if err != nil {
		return nil, err
	}

	o, err := exch.GetCachedOrderbook(p, asset.Spot)
	if err != nil {
		return nil, err
	}

	buy := strings.EqualFold(r.Side, order.Buy.String()) || strings.EqualFold(r.Side, order.Bid.String())

	result, err := o.SimulateOrder(r.Amount, buy)
	if err != nil {
		return nil, err
	}

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
func (s *RPCServer) WhaleBomb(_ context.Context, r *gctrpc.WhaleBombRequest) (*gctrpc.SimulateOrderResponse, error) {
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	o, err := exch.GetCachedOrderbook(p, a)
	if err != nil {
		return nil, err
	}

	buy := strings.EqualFold(r.Side, order.Buy.String()) || strings.EqualFold(r.Side, order.Bid.String())

	result, err := o.WhaleBomb(r.PriceTarget, buy)
	if err != nil {
		return nil, err
	}
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
func (s *RPCServer) CancelOrder(ctx context.Context, r *gctrpc.CancelOrderRequest) (*gctrpc.GenericResponse, error) {
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	var side order.Side
	side, err = order.StringToOrderSide(r.Side)
	if err != nil {
		return nil, err
	}

	err = s.OrderManager.Cancel(ctx,
		&order.Cancel{
			Exchange:  r.Exchange,
			AccountID: r.AccountId,
			OrderID:   r.OrderId,
			Side:      side,
			Pair:      p,
			AssetType: a,
		})
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{
		Status: MsgStatusSuccess,
		Data:   fmt.Sprintf("order %s cancelled", r.OrderId),
	}, nil
}

// CancelBatchOrders cancels an orders specified by exchange, currency pair and asset type
func (s *RPCServer) CancelBatchOrders(ctx context.Context, r *gctrpc.CancelBatchOrdersRequest) (*gctrpc.CancelBatchOrdersResponse, error) {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	pair := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, assetType, pair)
	if err != nil {
		return nil, err
	}

	var side order.Side
	side, err = order.StringToOrderSide(r.Side)
	if err != nil {
		return nil, err
	}

	status := make(map[string]string)
	orders := strings.Split(r.OrdersId, ",")
	req := make([]order.Cancel, len(orders))
	for x := range orders {
		orderID := orders[x]
		status[orderID] = order.Cancelled.String()
		req[x] = order.Cancel{
			AccountID: r.AccountId,
			OrderID:   orderID,
			Side:      side,
			Pair:      pair,
			AssetType: assetType,
		}
	}

	// TODO: Change to order manager
	_, err = exch.CancelBatchOrders(ctx, req)
	if err != nil {
		return nil, err
	}

	return &gctrpc.CancelBatchOrdersResponse{
		Orders: []*gctrpc.Orders{{
			Exchange:    r.Exchange,
			OrderStatus: status,
		}},
	}, nil
}

// CancelAllOrders cancels all orders, filterable by exchange
func (s *RPCServer) CancelAllOrders(ctx context.Context, r *gctrpc.CancelAllOrdersRequest) (*gctrpc.CancelAllOrdersResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	// TODO: Change to order manager
	resp, err := exch.CancelAllOrders(ctx, nil)
	if err != nil {
		return nil, err
	}

	cancelledOrders := new(gctrpc.Orders)
	cancelledOrders.Exchange = r.Exchange
	cancelledOrders.OrderStatus = resp.Status

	return &gctrpc.CancelAllOrdersResponse{Orders: []*gctrpc.Orders{cancelledOrders}, Count: int64(len(resp.Status))}, nil
}

// ModifyOrder modifies an existing order if it exists
func (s *RPCServer) ModifyOrder(ctx context.Context, r *gctrpc.ModifyOrderRequest) (*gctrpc.ModifyOrderResponse, error) {
	assetType, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	pair := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, assetType, pair)
	if err != nil {
		return nil, err
	}
	resp, err := s.OrderManager.Modify(ctx, &order.Modify{
		Exchange:  r.Exchange,
		AssetType: assetType,
		Pair:      pair,
		OrderID:   r.OrderId,
		Amount:    r.Amount,
		Price:     r.Price,
	})
	if err != nil {
		return nil, err
	}
	return &gctrpc.ModifyOrderResponse{
		ModifiedOrderId: resp.OrderID,
	}, nil
}

// GetEvents returns the stored events list
func (s *RPCServer) GetEvents(_ context.Context, _ *gctrpc.GetEventsRequest) (*gctrpc.GetEventsResponse, error) {
	return &gctrpc.GetEventsResponse{}, common.ErrNotYetImplemented
}

// AddEvent adds an event
func (s *RPCServer) AddEvent(_ context.Context, r *gctrpc.AddEventRequest) (*gctrpc.AddEventResponse, error) {
	evtCondition := EventConditionParams{
		CheckBids:       r.ConditionParams.CheckBids,
		CheckAsks:       r.ConditionParams.CheckAsks,
		Condition:       r.ConditionParams.Condition,
		OrderbookAmount: r.ConditionParams.OrderbookAmount,
		Price:           r.ConditionParams.Price,
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base,
		r.Pair.Quote, r.Pair.Delimiter)

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	id, err := s.eventManager.Add(r.Exchange, r.Item, evtCondition, p, a, r.Action)
	if err != nil {
		return nil, err
	}

	return &gctrpc.AddEventResponse{Id: id}, nil
}

// RemoveEvent removes an event, specified by an event ID
func (s *RPCServer) RemoveEvent(_ context.Context, r *gctrpc.RemoveEventRequest) (*gctrpc.GenericResponse, error) {
	if !s.eventManager.Remove(r.Id) {
		return nil, fmt.Errorf("event %d not removed", r.Id)
	}
	return &gctrpc.GenericResponse{
		Status: MsgStatusSuccess,
		Data:   fmt.Sprintf("event %d removed", r.Id),
	}, nil
}

// GetCryptocurrencyDepositAddresses returns a list of cryptocurrency deposit
// addresses specified by an exchange
func (s *RPCServer) GetCryptocurrencyDepositAddresses(_ context.Context, r *gctrpc.GetCryptocurrencyDepositAddressesRequest) (*gctrpc.GetCryptocurrencyDepositAddressesResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	if !exch.IsRESTAuthenticationSupported() {
		return nil, fmt.Errorf("%s, %w", r.Exchange, exchange.ErrAuthenticationSupportNotEnabled)
	}

	result, err := s.GetCryptocurrencyDepositAddressesByExchange(r.Exchange)
	if err != nil {
		return nil, err
	}

	var resp gctrpc.GetCryptocurrencyDepositAddressesResponse
	resp.Addresses = make(map[string]*gctrpc.DepositAddresses)
	for k, v := range result {
		var depositAddrs []*gctrpc.DepositAddress
		for a := range v {
			depositAddrs = append(depositAddrs, &gctrpc.DepositAddress{
				Address: v[a].Address,
				Tag:     v[a].Tag,
				Chain:   v[a].Chain,
			})
		}
		resp.Addresses[k] = &gctrpc.DepositAddresses{Addresses: depositAddrs}
	}
	return &resp, nil
}

// GetCryptocurrencyDepositAddress returns a cryptocurrency deposit address
// specified by exchange and cryptocurrency
func (s *RPCServer) GetCryptocurrencyDepositAddress(ctx context.Context, r *gctrpc.GetCryptocurrencyDepositAddressRequest) (*gctrpc.GetCryptocurrencyDepositAddressResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	if !exch.IsRESTAuthenticationSupported() {
		return nil, fmt.Errorf("%s, %w", r.Exchange, exchange.ErrAuthenticationSupportNotEnabled)
	}

	addr, err := s.GetExchangeCryptocurrencyDepositAddress(ctx,
		r.Exchange,
		"",
		r.Chain,
		currency.NewCode(r.Cryptocurrency),
		r.Bypass,
	)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetCryptocurrencyDepositAddressResponse{
		Address: addr.Address,
		Tag:     addr.Tag,
	}, nil
}

// GetAvailableTransferChains returns the supported transfer chains specified by
// exchange and cryptocurrency
func (s *RPCServer) GetAvailableTransferChains(ctx context.Context, r *gctrpc.GetAvailableTransferChainsRequest) (*gctrpc.GetAvailableTransferChainsResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	curr := currency.NewCode(r.Cryptocurrency)
	if curr.IsEmpty() {
		return nil, errCurrencyNotSpecified
	}

	resp, err := exch.GetAvailableTransferChains(ctx, curr)
	if err != nil {
		return nil, err
	}

	if len(resp) == 0 {
		return nil, errors.New("no available transfer chains found")
	}

	return &gctrpc.GetAvailableTransferChainsResponse{
		Chains: resp,
	}, nil
}

// WithdrawCryptocurrencyFunds withdraws cryptocurrency funds specified by
// exchange
func (s *RPCServer) WithdrawCryptocurrencyFunds(ctx context.Context, r *gctrpc.WithdrawCryptoRequest) (*gctrpc.WithdrawResponse, error) {
	_, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	req := &withdraw.Request{
		Exchange:    r.Exchange,
		Amount:      r.Amount,
		Currency:    currency.NewCode(strings.ToUpper(r.Currency)),
		Type:        withdraw.Crypto,
		Description: r.Description,
		Crypto: withdraw.CryptoRequest{
			Address:    r.Address,
			AddressTag: r.AddressTag,
			FeeAmount:  r.Fee,
			Chain:      r.Chain,
		},
	}

	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	if exchCfg.API.Credentials.OTPSecret != "" {
		code, errOTP := totp.GenerateCode(exchCfg.API.Credentials.OTPSecret, time.Now())
		if errOTP != nil {
			return nil, errOTP
		}

		codeNum, errOTP := strconv.ParseInt(code, 10, 64)
		if errOTP != nil {
			return nil, errOTP
		}
		req.OneTimePassword = codeNum
	}

	if exchCfg.API.Credentials.PIN != "" {
		pinCode, errPin := strconv.ParseInt(exchCfg.API.Credentials.PIN, 10, 64)
		if errPin != nil {
			return nil, errPin
		}
		req.PIN = pinCode
	}

	req.TradePassword = exchCfg.API.Credentials.TradePassword

	resp, err := s.Engine.WithdrawManager.SubmitWithdrawal(ctx, req)
	if err != nil {
		return nil, err
	}

	return &gctrpc.WithdrawResponse{
		Id:     resp.ID.String(),
		Status: resp.Exchange.Status,
	}, nil
}

// WithdrawFiatFunds withdraws fiat funds specified by exchange
func (s *RPCServer) WithdrawFiatFunds(ctx context.Context, r *gctrpc.WithdrawFiatRequest) (*gctrpc.WithdrawResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	bankAccount, err := banking.GetBankAccountByID(r.BankAccountId)
	if err != nil {
		base := exch.GetBase()
		if base == nil {
			return nil, errExchangeBaseNotFound
		}
		bankAccount, err = base.GetExchangeBankAccounts(r.BankAccountId,
			r.Currency)
		if err != nil {
			return nil, err
		}
	}

	req := &withdraw.Request{
		Exchange:    r.Exchange,
		Amount:      r.Amount,
		Currency:    currency.NewCode(strings.ToUpper(r.Currency)),
		Type:        withdraw.Fiat,
		Description: r.Description,
		Fiat: withdraw.FiatRequest{
			Bank: *bankAccount,
		},
	}

	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	if exchCfg.API.Credentials.OTPSecret != "" {
		code, errOTP := totp.GenerateCode(exchCfg.API.Credentials.OTPSecret, time.Now())
		if errOTP != nil {
			return nil, errOTP
		}

		codeNum, errOTP := strconv.ParseInt(code, 10, 64)
		if errOTP != nil {
			return nil, errOTP
		}
		req.OneTimePassword = codeNum
	}

	if exchCfg.API.Credentials.PIN != "" {
		pinCode, errPIN := strconv.ParseInt(exchCfg.API.Credentials.PIN, 10, 64)
		if errPIN != nil {
			return nil, errPIN
		}
		req.PIN = pinCode
	}

	req.TradePassword = exchCfg.API.Credentials.TradePassword

	resp, err := s.Engine.WithdrawManager.SubmitWithdrawal(ctx, req)
	if err != nil {
		return nil, err
	}

	return &gctrpc.WithdrawResponse{
		Id:     resp.ID.String(),
		Status: resp.Exchange.Status,
	}, nil
}

// WithdrawalEventByID returns previous withdrawal request details
func (s *RPCServer) WithdrawalEventByID(_ context.Context, r *gctrpc.WithdrawalEventByIDRequest) (*gctrpc.WithdrawalEventByIDResponse, error) {
	if !s.Config.Database.Enabled {
		return nil, database.ErrDatabaseSupportDisabled
	}
	v, err := s.WithdrawManager.WithdrawalEventByID(r.Id)
	if err != nil {
		return nil, err
	}

	resp := &gctrpc.WithdrawalEventByIDResponse{
		Event: &gctrpc.WithdrawalEventResponse{
			Id: v.ID.String(),
			Exchange: &gctrpc.WithdrawlExchangeEvent{
				Name:   v.Exchange.Name,
				Id:     v.Exchange.Name,
				Status: v.Exchange.Status,
			},
			Request: &gctrpc.WithdrawalRequestEvent{
				Currency:    v.RequestDetails.Currency.String(),
				Description: v.RequestDetails.Description,
				Amount:      v.RequestDetails.Amount,
				Type:        int64(v.RequestDetails.Type),
			},
		},
	}

	resp.Event.CreatedAt = timestamppb.New(v.CreatedAt)
	if err := resp.Event.CreatedAt.CheckValid(); err != nil {
		log.Errorf(log.GRPCSys, "withdrawal event by id CreatedAt: %s", err)
	}
	resp.Event.UpdatedAt = timestamppb.New(v.UpdatedAt)
	if err := resp.Event.UpdatedAt.CheckValid(); err != nil {
		log.Errorf(log.GRPCSys, "withdrawal event by id UpdatedAt: %s", err)
	}

	switch v.RequestDetails.Type {
	case withdraw.Crypto:
		resp.Event.Request.Crypto = new(gctrpc.CryptoWithdrawalEvent)
		resp.Event.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
			Address:    v.RequestDetails.Crypto.Address,
			AddressTag: v.RequestDetails.Crypto.AddressTag,
			Fee:        v.RequestDetails.Crypto.FeeAmount,
		}
	case withdraw.Fiat:
		resp.Event.Request.Fiat = new(gctrpc.FiatWithdrawalEvent)
		resp.Event.Request.Fiat = &gctrpc.FiatWithdrawalEvent{
			BankName:      v.RequestDetails.Fiat.Bank.BankName,
			AccountName:   v.RequestDetails.Fiat.Bank.AccountName,
			AccountNumber: v.RequestDetails.Fiat.Bank.AccountNumber,
			Bsb:           v.RequestDetails.Fiat.Bank.BSBNumber,
			Swift:         v.RequestDetails.Fiat.Bank.SWIFTCode,
			Iban:          v.RequestDetails.Fiat.Bank.IBAN,
		}
	}

	return resp, nil
}

// WithdrawalEventsByExchange returns previous withdrawal request details by exchange
func (s *RPCServer) WithdrawalEventsByExchange(ctx context.Context, r *gctrpc.WithdrawalEventsByExchangeRequest) (*gctrpc.WithdrawalEventsByExchangeResponse, error) {
	if !s.Config.Database.Enabled {
		if r.Id == "" {
			exch, err := s.GetExchangeByName(r.Exchange)
			if err != nil {
				return nil, err
			}

			c := currency.NewCode(strings.ToUpper(r.Currency))
			a, err := asset.New(r.AssetType)
			if err != nil {
				return nil, err
			}
			ret, err := exch.GetWithdrawalsHistory(ctx, c, a)
			if err != nil {
				return nil, err
			}

			return parseWithdrawalsHistory(ret, exch.GetName(), int(r.Limit)), nil
		}
		return nil, database.ErrDatabaseSupportDisabled
	}
	if r.Id == "" {
		ret, err := s.WithdrawManager.WithdrawalEventByExchange(r.Exchange, int(r.Limit))
		if err != nil {
			return nil, err
		}
		return parseMultipleEvents(ret), nil
	}

	ret, err := s.WithdrawManager.WithdrawalEventByExchangeID(r.Exchange, r.Id)
	if err != nil {
		return nil, err
	}

	return parseSingleEvents(ret), nil
}

// WithdrawalEventsByDate returns previous withdrawal request details by exchange
func (s *RPCServer) WithdrawalEventsByDate(_ context.Context, r *gctrpc.WithdrawalEventsByDateRequest) (*gctrpc.WithdrawalEventsByExchangeResponse, error) {
	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.End)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	var ret []*withdraw.Response
	ret, err = s.WithdrawManager.WithdrawEventByDate(r.Exchange, start, end, int(r.Limit))
	if err != nil {
		return nil, err
	}
	return parseMultipleEvents(ret), nil
}

// GetLoggerDetails returns a loggers details
func (s *RPCServer) GetLoggerDetails(_ context.Context, r *gctrpc.GetLoggerDetailsRequest) (*gctrpc.GetLoggerDetailsResponse, error) {
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
func (s *RPCServer) SetLoggerDetails(_ context.Context, r *gctrpc.SetLoggerDetailsRequest) (*gctrpc.GetLoggerDetailsResponse, error) {
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
func (s *RPCServer) GetExchangePairs(_ context.Context, r *gctrpc.GetExchangePairsRequest) (*gctrpc.GetExchangePairsResponse, error) {
	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}
	assetTypes := exchCfg.CurrencyPairs.GetAssetTypes(false)

	var a asset.Item
	if r.Asset != "" {
		a, err = asset.New(r.Asset)
		if err != nil {
			return nil, err
		}
		if !assetTypes.Contains(a) {
			return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
		}
	}

	var resp gctrpc.GetExchangePairsResponse
	resp.SupportedAssets = make(map[string]*gctrpc.PairsSupported)
	for x := range assetTypes {
		if r.Asset != "" && !strings.EqualFold(assetTypes[x].String(), r.Asset) {
			continue
		}

		var enabled currency.Pairs
		enabled, err = exchCfg.CurrencyPairs.GetPairs(assetTypes[x], true)
		if err != nil {
			return nil, err
		}

		var available currency.Pairs
		available, err = exchCfg.CurrencyPairs.GetPairs(assetTypes[x], false)
		if err != nil {
			return nil, err
		}

		resp.SupportedAssets[assetTypes[x].String()] = &gctrpc.PairsSupported{
			AvailablePairs: available.Join(),
			EnabledPairs:   enabled.Join(),
		}
	}
	return &resp, nil
}

// SetExchangePair enables/disabled the specified pair(s) on an exchange
func (s *RPCServer) SetExchangePair(_ context.Context, r *gctrpc.SetExchangePairRequest) (*gctrpc.GenericResponse, error) {
	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, a, currency.EMPTYPAIR)
	if err != nil {
		return nil, err
	}

	base := exch.GetBase()
	if base == nil {
		return nil, errExchangeBaseNotFound
	}

	pairFmt, err := s.Config.GetPairFormat(r.Exchange, a)
	if err != nil {
		return nil, err
	}
	var pass bool
	var newErrors error
	for i := range r.Pairs {
		var p currency.Pair
		p, err = currency.NewPairFromStrings(r.Pairs[i].Base, r.Pairs[i].Quote)
		if err != nil {
			return nil, err
		}

		if r.Enable {
			err = exchCfg.CurrencyPairs.EnablePair(a, p.Format(pairFmt))
			if err != nil {
				newErrors = common.AppendError(newErrors, fmt.Errorf("%s %w", r.Pairs[i], err))
				continue
			}
			err = base.CurrencyPairs.EnablePair(a, p)
			if err != nil {
				newErrors = common.AppendError(newErrors, fmt.Errorf("%s %w", r.Pairs[i], err))
				continue
			}
			pass = true
			continue
		}

		err = exchCfg.CurrencyPairs.DisablePair(a, p.Format(pairFmt))
		if err != nil {
			if errors.Is(err, currency.ErrPairNotFound) {
				newErrors = common.AppendError(newErrors, fmt.Errorf("%s %w", r.Pairs[i], errSpecificPairNotEnabled))
				continue
			}
			return nil, err
		}

		err = base.CurrencyPairs.DisablePair(a, p)
		if err != nil {
			if errors.Is(err, currency.ErrPairNotFound) {
				newErrors = common.AppendError(newErrors, fmt.Errorf("%s %w", r.Pairs[i], errSpecificPairNotEnabled))
				continue
			}
			return nil, err
		}
		pass = true
	}

	if exch.IsWebsocketEnabled() && pass && base.Websocket.IsConnected() {
		err = exch.FlushWebsocketChannels()
		if err != nil {
			newErrors = common.AppendError(newErrors, err)
		}
	}

	if newErrors != nil {
		return nil, newErrors
	}

	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// GetOrderbookStream streams the requested updated orderbook
func (s *RPCServer) GetOrderbookStream(r *gctrpc.GetOrderbookStreamRequest, stream gctrpc.GoCryptoTraderService_GetOrderbookStreamServer) error {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return err
	}

	depth, err := orderbook.GetDepth(r.Exchange, p, a)
	if err != nil {
		return err
	}

	for {
		resp := &gctrpc.OrderbookResponse{
			Pair:      &gctrpc.CurrencyPair{Base: r.Pair.Base, Quote: r.Pair.Quote},
			AssetType: r.AssetType,
		}
		base, err := depth.Retrieve()
		if err != nil {
			resp.Error = err.Error()
			resp.LastUpdated = time.Now().UnixMicro()
		} else {
			resp.LastUpdated = base.LastUpdated.UnixMicro()
			resp.Bids = make([]*gctrpc.OrderbookItem, len(base.Bids))
			for i := range base.Bids {
				resp.Bids[i] = &gctrpc.OrderbookItem{
					Amount: base.Bids[i].Amount,
					Price:  base.Bids[i].Price,
					Id:     base.Bids[i].ID,
				}
			}
			resp.Asks = make([]*gctrpc.OrderbookItem, len(base.Asks))
			for i := range base.Asks {
				resp.Asks[i] = &gctrpc.OrderbookItem{
					Amount: base.Asks[i].Amount,
					Price:  base.Asks[i].Price,
					Id:     base.Asks[i].ID,
				}
			}
		}

		err = stream.Send(resp)
		if err != nil {
			return err
		}
		<-depth.Wait(nil)
	}
}

// GetExchangeOrderbookStream streams all orderbooks associated with an exchange
func (s *RPCServer) GetExchangeOrderbookStream(r *gctrpc.GetExchangeOrderbookStreamRequest, stream gctrpc.GoCryptoTraderService_GetExchangeOrderbookStreamServer) error {
	if r.Exchange == "" {
		return common.ErrExchangeNameNotSet
	}

	if _, err := s.GetExchangeByName(r.Exchange); err != nil {
		return err
	}

	pipe, err := orderbook.SubscribeToExchangeOrderbooks(r.Exchange)
	if err != nil {
		return err
	}

	defer func() {
		pipeErr := pipe.Release()
		if pipeErr != nil {
			log.Errorln(log.DispatchMgr, pipeErr)
		}
	}()

	for {
		data, ok := <-pipe.Channel()
		if !ok {
			return errDispatchSystem
		}

		d, ok := data.(orderbook.Outbound)
		if !ok {
			return common.GetTypeAssertError("orderbook.Outbound", data)
		}

		resp := &gctrpc.OrderbookResponse{}
		ob, err := d.Retrieve()
		if err != nil {
			resp.Error = err.Error()
			resp.LastUpdated = time.Now().UnixMicro()
		} else {
			resp.LastUpdated = ob.LastUpdated.UnixMicro()
			resp.Pair = &gctrpc.CurrencyPair{
				Base:  ob.Pair.Base.String(),
				Quote: ob.Pair.Quote.String(),
			}
			resp.AssetType = ob.Asset.String()
			resp.Bids = make([]*gctrpc.OrderbookItem, len(ob.Bids))
			for i := range ob.Bids {
				resp.Bids[i] = &gctrpc.OrderbookItem{
					Amount: ob.Bids[i].Amount,
					Price:  ob.Bids[i].Price,
					Id:     ob.Bids[i].ID,
				}
			}
			resp.Asks = make([]*gctrpc.OrderbookItem, len(ob.Asks))
			for i := range ob.Asks {
				resp.Asks[i] = &gctrpc.OrderbookItem{
					Amount: ob.Asks[i].Amount,
					Price:  ob.Asks[i].Price,
					Id:     ob.Asks[i].ID,
				}
			}
		}

		err = stream.Send(resp)
		if err != nil {
			return err
		}
	}
}

// GetTickerStream streams the requested updated ticker
func (s *RPCServer) GetTickerStream(r *gctrpc.GetTickerStreamRequest, stream gctrpc.GoCryptoTraderService_GetTickerStreamServer) error {
	if r.Exchange == "" {
		return common.ErrExchangeNameNotSet
	}

	if _, err := s.GetExchangeByName(r.Exchange); err != nil {
		return err
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return err
	}

	if r.Pair.String() == "" {
		return errCurrencyPairUnset
	}

	if r.AssetType == "" {
		return errAssetTypeUnset
	}

	p, err := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	if err != nil {
		return err
	}

	pipe, err := ticker.SubscribeTicker(r.Exchange, p, a)
	if err != nil {
		return err
	}

	defer func() {
		pipeErr := pipe.Release()
		if pipeErr != nil {
			log.Errorln(log.DispatchMgr, pipeErr)
		}
	}()

	for {
		data, ok := <-pipe.Channel()
		if !ok {
			return errDispatchSystem
		}

		t, ok := data.(*ticker.Price)
		if !ok {
			return common.GetTypeAssertError("*ticker.Price", data)
		}

		err := stream.Send(&gctrpc.TickerResponse{
			Pair: &gctrpc.CurrencyPair{
				Base:      t.Pair.Base.String(),
				Quote:     t.Pair.Quote.String(),
				Delimiter: t.Pair.Delimiter,
			},
			LastUpdated: s.unixTimestamp(t.LastUpdated),
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
func (s *RPCServer) GetExchangeTickerStream(r *gctrpc.GetExchangeTickerStreamRequest, stream gctrpc.GoCryptoTraderService_GetExchangeTickerStreamServer) error {
	if r.Exchange == "" {
		return common.ErrExchangeNameNotSet
	}

	if _, err := s.GetExchangeByName(r.Exchange); err != nil {
		return err
	}

	pipe, err := ticker.SubscribeToExchangeTickers(r.Exchange)
	if err != nil {
		return err
	}

	defer func() {
		pipeErr := pipe.Release()
		if pipeErr != nil {
			log.Errorln(log.DispatchMgr, pipeErr)
		}
	}()

	for {
		data, ok := <-pipe.Channel()
		if !ok {
			return errDispatchSystem
		}

		t, ok := data.(*ticker.Price)
		if !ok {
			return common.GetTypeAssertError("*ticker.Price", data)
		}

		err := stream.Send(&gctrpc.TickerResponse{
			Pair: &gctrpc.CurrencyPair{
				Base:      t.Pair.Base.String(),
				Quote:     t.Pair.Quote.String(),
				Delimiter: t.Pair.Delimiter,
			},
			LastUpdated: s.unixTimestamp(t.LastUpdated),
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
func (s *RPCServer) GetAuditEvent(_ context.Context, r *gctrpc.GetAuditEventRequest) (*gctrpc.GetAuditEventResponse, error) {
	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	events, err := audit.GetEvent(start, end, r.OrderBy, int(r.Limit))
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
				Timestamp:  v[x].CreatedAt.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
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
func (s *RPCServer) GetHistoricCandles(ctx context.Context, r *gctrpc.GetHistoricCandlesRequest) (*gctrpc.GetHistoricCandlesResponse, error) {
	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.End)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	pair := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, pair)
	if err != nil {
		return nil, err
	}

	interval := kline.Interval(r.TimeInterval)

	resp := gctrpc.GetHistoricCandlesResponse{
		Interval: interval.Short(),
		Pair:     r.Pair,
		Start:    start.UTC().Format(common.SimpleTimeFormatWithTimezone),
		End:      end.UTC().Format(common.SimpleTimeFormatWithTimezone),
	}

	var klineItem *kline.Item
	if r.UseDb {
		klineItem, err = kline.LoadFromDatabase(r.Exchange,
			pair,
			a,
			interval,
			start,
			end)
	} else {
		if r.ExRequest {
			klineItem, err = exch.GetHistoricCandlesExtended(ctx, pair, a, interval, start, end)
		} else {
			klineItem, err = exch.GetHistoricCandles(ctx, pair, a, interval, start, end)
		}
	}
	if err != nil {
		return nil, err
	}

	if r.FillMissingWithTrades {
		var tradeDataKline *kline.Item
		tradeDataKline, err = fillMissingCandlesWithStoredTrades(start, end, klineItem)
		if err != nil {
			return nil, err
		}
		klineItem.Candles = append(klineItem.Candles, tradeDataKline.Candles...)
	}

	resp.Exchange = klineItem.Exchange
	for i := range klineItem.Candles {
		resp.Candle = append(resp.Candle, &gctrpc.Candle{
			Time:      klineItem.Candles[i].Time.UTC().Format(common.SimpleTimeFormatWithTimezone),
			Low:       klineItem.Candles[i].Low,
			High:      klineItem.Candles[i].High,
			Open:      klineItem.Candles[i].Open,
			Close:     klineItem.Candles[i].Close,
			Volume:    klineItem.Candles[i].Volume,
			IsPartial: klineItem.Candles[i].ValidationIssues == kline.PartialCandle,
		})
	}

	if r.Sync && !r.UseDb {
		_, err = kline.StoreInDatabase(klineItem, r.Force)
		if err != nil {
			if errors.Is(err, exchangeDB.ErrNoExchangeFound) {
				return nil, errors.New("exchange was not found in database, you can seed existing data or insert a new exchange via the dbseed")
			}
			return nil, err
		}
	}

	return &resp, nil
}

func fillMissingCandlesWithStoredTrades(startTime, endTime time.Time, klineItem *kline.Item) (*kline.Item, error) {
	candleTimes := make([]time.Time, len(klineItem.Candles))
	for i := range klineItem.Candles {
		candleTimes[i] = klineItem.Candles[i].Time
	}
	ranges, err := timeperiods.FindTimeRangesContainingData(startTime, endTime, klineItem.Interval.Duration(), candleTimes)
	if err != nil {
		return nil, err
	}

	var response kline.Item
	for i := range ranges {
		if ranges[i].HasDataInRange {
			continue
		}
		var tradeCandles *kline.Item
		trades, err := trade.GetTradesInRange(
			klineItem.Exchange,
			klineItem.Asset.String(),
			klineItem.Pair.Base.String(),
			klineItem.Pair.Quote.String(),
			ranges[i].StartOfRange,
			ranges[i].EndOfRange,
		)
		if err != nil {
			return klineItem, err
		}
		if len(trades) == 0 {
			continue
		}
		tradeCandles, err = trade.ConvertTradesToCandles(klineItem.Interval, trades...)
		if err != nil {
			return klineItem, err
		}
		if len(tradeCandles.Candles) == 0 {
			continue
		}
		response.Candles = append(response.Candles, tradeCandles.Candles...)

		for i := range response.Candles {
			log.Infof(log.GRPCSys,
				"Filled requested OHLCV data for %v %v %v interval at %v with trade data",
				klineItem.Exchange,
				klineItem.Pair.String(),
				klineItem.Asset,
				response.Candles[i].Time.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			)
		}
	}

	return &response, nil
}

// GCTScriptStatus returns a slice of current running scripts that includes next run time and uuid
func (s *RPCServer) GCTScriptStatus(_ context.Context, _ *gctrpc.GCTScriptStatusRequest) (*gctrpc.GCTScriptStatusResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GCTScriptStatusResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	if gctscript.VMSCount.Len() < 1 {
		return &gctrpc.GCTScriptStatusResponse{Status: "no scripts running"}, nil
	}

	resp := &gctrpc.GCTScriptStatusResponse{
		Status: fmt.Sprintf("%v of %v virtual machines running", gctscript.VMSCount.Len(), s.gctScriptManager.GetMaxVirtualMachines()),
	}

	gctscript.AllVMSync.Range(func(_, v any) bool {
		vm, ok := v.(*gctscript.VM)
		if !ok {
			log.Errorf(log.GRPCSys, "%v", common.GetTypeAssertError("*gctscript.VM", v))
			return false
		}
		resp.Scripts = append(resp.Scripts, &gctrpc.GCTScript{
			Uuid:    vm.ID.String(),
			Name:    vm.ShortName(),
			NextRun: vm.NextRun.String(),
		})

		return true
	})

	return resp, nil
}

// GCTScriptQuery queries a running script and returns script running information
func (s *RPCServer) GCTScriptQuery(_ context.Context, r *gctrpc.GCTScriptQueryRequest) (*gctrpc.GCTScriptQueryResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GCTScriptQueryResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	UUID, err := uuid.FromString(r.Script.Uuid)
	if err != nil {
		//nolint:nilerr // error is returned in the GCTScriptQueryResponse
		return &gctrpc.GCTScriptQueryResponse{Status: MsgStatusError, Data: err.Error()}, nil
	}

	v, f := gctscript.AllVMSync.Load(UUID)
	if !f {
		return &gctrpc.GCTScriptQueryResponse{Status: MsgStatusError, Data: "UUID not found"}, nil
	}

	vm, ok := v.(*gctscript.VM)
	if !ok {
		return nil, common.GetTypeAssertError("*gctscript.VM", v)
	}
	resp := &gctrpc.GCTScriptQueryResponse{
		Status: MsgStatusOK,
		Script: &gctrpc.GCTScript{
			Name:    vm.ShortName(),
			Uuid:    vm.ID.String(),
			Path:    vm.Path,
			NextRun: vm.NextRun.String(),
		},
	}
	data, err := vm.Read()
	if err != nil {
		return nil, err
	}
	resp.Data = string(data)
	return resp, nil
}

// GCTScriptExecute execute a script
func (s *RPCServer) GCTScriptExecute(_ context.Context, r *gctrpc.GCTScriptExecuteRequest) (*gctrpc.GenericResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GenericResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	if r.Script.Path == "" {
		r.Script.Path = gctscript.ScriptPath
	}

	gctVM := s.gctScriptManager.New()
	if gctVM == nil {
		return &gctrpc.GenericResponse{Status: MsgStatusError, Data: "unable to create VM instance"}, nil
	}

	script := filepath.Join(r.Script.Path, r.Script.Name)
	if err := gctVM.Load(script); err != nil {
		return &gctrpc.GenericResponse{ //nolint:nilerr // error is returned in the generic response
			Status: MsgStatusError,
			Data:   err.Error(),
		}, nil
	}

	go gctVM.CompileAndRun()

	return &gctrpc.GenericResponse{
		Status: MsgStatusOK,
		Data:   gctVM.ShortName() + " (" + gctVM.ID.String() + ") executed",
	}, nil
}

// GCTScriptStop terminate a running script
func (s *RPCServer) GCTScriptStop(_ context.Context, r *gctrpc.GCTScriptStopRequest) (*gctrpc.GenericResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GenericResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	UUID, err := uuid.FromString(r.Script.Uuid)
	if err != nil {
		return &gctrpc.GenericResponse{Status: MsgStatusError, Data: err.Error()}, nil //nolint:nilerr // error is returned in the generic response
	}

	v, f := gctscript.AllVMSync.Load(UUID)
	if !f {
		return &gctrpc.GenericResponse{Status: MsgStatusError, Data: "no running script found"}, nil
	}

	vm, ok := v.(*gctscript.VM)
	if !ok {
		return nil, common.GetTypeAssertError("*gctscript.VM", v)
	}
	err = vm.Shutdown()
	status := " terminated"
	if err != nil {
		status = " " + err.Error()
	}
	return &gctrpc.GenericResponse{Status: MsgStatusOK, Data: vm.ID.String() + status}, nil
}

// GCTScriptUpload upload a new script to ScriptPath
func (s *RPCServer) GCTScriptUpload(_ context.Context, r *gctrpc.GCTScriptUploadRequest) (*gctrpc.GenericResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GenericResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	fPath := filepath.Join(gctscript.ScriptPath, r.ScriptName)
	fPathExits := fPath
	if filepath.Ext(fPath) == ".zip" {
		fPathExits = fPathExits[0 : len(fPathExits)-4]
	}

	if s, err := os.Stat(fPathExits); !os.IsNotExist(err) {
		if !r.Overwrite {
			return nil, fmt.Errorf("%s script found and overwrite set to false", r.ScriptName)
		}
		f := filepath.Join(gctscript.ScriptPath, "version_history")
		err = os.MkdirAll(f, file.DefaultPermissionOctal)
		if err != nil {
			return nil, err
		}
		timeString := strconv.FormatInt(time.Now().UnixNano(), 10)
		renamedFile := filepath.Join(f, timeString+"-"+filepath.Base(fPathExits))
		if s.IsDir() {
			err = archive.Zip(fPathExits, renamedFile+".zip")
			if err != nil {
				return nil, err
			}
		} else {
			err = file.Move(fPathExits, renamedFile)
			if err != nil {
				return nil, err
			}
		}
	}

	newFile, err := os.Create(fPath)
	if err != nil {
		return nil, err
	}

	_, err = newFile.Write(r.Data)
	if err != nil {
		return nil, err
	}
	err = newFile.Close()
	if err != nil {
		log.Errorln(log.Global, "Failed to close file handle, archive removal may fail")
	}

	if r.Archived {
		files, errExtract := archive.UnZip(fPath, filepath.Join(gctscript.ScriptPath, r.ScriptName[:len(r.ScriptName)-4]))
		if errExtract != nil {
			log.Errorf(log.Global, "Failed to archive zip file %v", errExtract)
			return &gctrpc.GenericResponse{Status: MsgStatusError, Data: errExtract.Error()}, nil
		}
		var failedFiles []string
		for x := range files {
			err = s.gctScriptManager.Validate(files[x])
			if err != nil {
				failedFiles = append(failedFiles, files[x])
			}
		}
		err = os.Remove(fPath)
		if err != nil {
			return nil, err
		}
		if len(failedFiles) > 0 {
			err = os.RemoveAll(filepath.Join(gctscript.ScriptPath, r.ScriptName[:len(r.ScriptName)-4]))
			if err != nil {
				log.Errorf(log.GCTScriptMgr, "Failed to remove file %v (%v), manual deletion required", filepath.Base(fPath), err)
			}
			return &gctrpc.GenericResponse{Status: gctscript.ErrScriptFailedValidation, Data: strings.Join(failedFiles, ", ")}, nil
		}
	} else {
		err = s.gctScriptManager.Validate(fPath)
		if err != nil {
			errRemove := os.Remove(fPath)
			if errRemove != nil {
				log.Errorf(log.GCTScriptMgr, "Failed to remove file %v, manual deletion required: %v", filepath.Base(fPath), errRemove)
			}
			return &gctrpc.GenericResponse{Status: gctscript.ErrScriptFailedValidation, Data: err.Error()}, nil
		}
	}

	return &gctrpc.GenericResponse{
		Status: MsgStatusOK,
		Data:   fmt.Sprintf("script %s written", newFile.Name()),
	}, nil
}

// GCTScriptReadScript read a script and return contents
func (s *RPCServer) GCTScriptReadScript(_ context.Context, r *gctrpc.GCTScriptReadScriptRequest) (*gctrpc.GCTScriptQueryResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GCTScriptQueryResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	filename := filepath.Join(gctscript.ScriptPath, r.Script.Name)
	if !strings.HasPrefix(filename, filepath.Clean(gctscript.ScriptPath)+string(os.PathSeparator)) {
		return nil, fmt.Errorf("%s: invalid file path", filename)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GCTScriptQueryResponse{
		Status: MsgStatusOK,
		Script: &gctrpc.GCTScript{
			Name: filepath.Base(filename),
			Path: filepath.Dir(filename),
		},
		Data: string(data),
	}, nil
}

// GCTScriptListAll lists all scripts inside the default script path
func (s *RPCServer) GCTScriptListAll(context.Context, *gctrpc.GCTScriptListAllRequest) (*gctrpc.GCTScriptStatusResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GCTScriptStatusResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	resp := &gctrpc.GCTScriptStatusResponse{}
	err := filepath.Walk(gctscript.ScriptPath,
		func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == common.GctExt {
				resp.Scripts = append(resp.Scripts, &gctrpc.GCTScript{
					Name: path,
				})
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GCTScriptStopAll stops all running scripts
func (s *RPCServer) GCTScriptStopAll(context.Context, *gctrpc.GCTScriptStopAllRequest) (*gctrpc.GenericResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GenericResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	err := s.gctScriptManager.ShutdownAll()
	if err != nil {
		return &gctrpc.GenericResponse{Status: "error", Data: err.Error()}, nil //nolint:nilerr // error is returned in the generic response
	}

	return &gctrpc.GenericResponse{
		Status: MsgStatusOK,
		Data:   "all running scripts have been stopped",
	}, nil
}

// GCTScriptAutoLoadToggle adds or removes an entry to the autoload list
func (s *RPCServer) GCTScriptAutoLoadToggle(_ context.Context, r *gctrpc.GCTScriptAutoLoadRequest) (*gctrpc.GenericResponse, error) {
	if !s.gctScriptManager.IsRunning() {
		return &gctrpc.GenericResponse{Status: gctscript.ErrScriptingDisabled.Error()}, nil
	}

	if r.Status {
		err := s.gctScriptManager.Autoload(r.Script, true)
		if err != nil {
			//nolint:nilerr // error is returned in the generic response
			return &gctrpc.GenericResponse{Status: "error", Data: err.Error()}, nil
		}
		return &gctrpc.GenericResponse{Status: "success", Data: "script " + r.Script + " removed from autoload list"}, nil
	}

	err := s.gctScriptManager.Autoload(r.Script, false)
	if err != nil {
		return &gctrpc.GenericResponse{Status: "error", Data: err.Error()}, nil //nolint:nilerr // error is returned in the generic response
	}
	return &gctrpc.GenericResponse{Status: "success", Data: "script " + r.Script + " added to autoload list"}, nil
}

// SetExchangeAsset enables or disables an exchanges asset type
func (s *RPCServer) SetExchangeAsset(_ context.Context, r *gctrpc.SetExchangeAssetRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	base := exch.GetBase()
	if base == nil {
		return nil, errExchangeBaseNotFound
	}

	if r.Asset == "" {
		return nil, errors.New("asset type must be specified")
	}

	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	err = base.CurrencyPairs.SetAssetEnabled(a, r.Enable)
	if err != nil {
		return nil, err
	}
	err = exchCfg.CurrencyPairs.SetAssetEnabled(a, r.Enable)
	if err != nil {
		return nil, err
	}

	if base.IsWebsocketEnabled() && base.Websocket.IsConnected() {
		err = exch.FlushWebsocketChannels()
		if err != nil {
			return nil, err
		}
	}

	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// SetAllExchangePairs enables or disables an exchanges pairs
func (s *RPCServer) SetAllExchangePairs(_ context.Context, r *gctrpc.SetExchangeAllPairsRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	base := exch.GetBase()
	if base == nil {
		return nil, errExchangeBaseNotFound
	}

	assets := base.CurrencyPairs.GetAssetTypes(false)

	if r.Enable {
		for i := range assets {
			var pairs currency.Pairs
			pairs, err = base.CurrencyPairs.GetPairs(assets[i], false)
			if err != nil {
				return nil, err
			}
			err = exchCfg.CurrencyPairs.StorePairs(assets[i], pairs, true)
			if err != nil {
				return nil, err
			}
			err = base.CurrencyPairs.StorePairs(assets[i], pairs, true)
			if err != nil {
				return nil, err
			}
		}
	} else {
		for i := range assets {
			err = exchCfg.CurrencyPairs.StorePairs(assets[i], nil, true)
			if err != nil {
				return nil, err
			}
			err = base.CurrencyPairs.StorePairs(assets[i], nil, true)
			if err != nil {
				return nil, err
			}
		}
	}

	if exch.IsWebsocketEnabled() && base.Websocket.IsConnected() {
		err = exch.FlushWebsocketChannels()
		if err != nil {
			return nil, err
		}
	}

	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// UpdateExchangeSupportedPairs forces an update of the supported pairs which
// will update the available pairs list and remove any assets that are disabled
// by the exchange
func (s *RPCServer) UpdateExchangeSupportedPairs(ctx context.Context, r *gctrpc.UpdateExchangeSupportedPairsRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	base := exch.GetBase()
	if base == nil {
		return nil, errExchangeBaseNotFound
	}

	if !base.GetEnabledFeatures().AutoPairUpdates {
		return nil,
			errors.New("cannot auto pair update for exchange, a manual update is needed")
	}

	if err := exch.UpdateTradablePairs(ctx); err != nil {
		return nil, err
	}

	if exch.IsWebsocketEnabled() {
		err = exch.FlushWebsocketChannels()
		if err != nil {
			return nil, err
		}
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// GetExchangeAssets returns the supported asset types
func (s *RPCServer) GetExchangeAssets(_ context.Context, r *gctrpc.GetExchangeAssetsRequest) (*gctrpc.GetExchangeAssetsResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetExchangeAssetsResponse{
		Assets: exch.GetAssetTypes(false).JoinToString(","),
	}, nil
}

// WebsocketGetInfo returns websocket connection information
func (s *RPCServer) WebsocketGetInfo(_ context.Context, r *gctrpc.WebsocketGetInfoRequest) (*gctrpc.WebsocketGetInfoResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	w, err := exch.GetWebsocket()
	if err != nil {
		return nil, err
	}

	return &gctrpc.WebsocketGetInfoResponse{
		Exchange:      exch.GetName(),
		Supported:     exch.SupportsWebsocket(),
		Enabled:       exch.IsWebsocketEnabled(),
		Authenticated: w.CanUseAuthenticatedEndpoints(),
		RunningUrl:    w.GetWebsocketURL(),
		ProxyAddress:  w.GetProxyAddress(),
	}, nil
}

// WebsocketSetEnabled enables or disables the websocket client
func (s *RPCServer) WebsocketSetEnabled(ctx context.Context, r *gctrpc.WebsocketSetEnabledRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	w, err := exch.GetWebsocket()
	if err != nil {
		return nil, fmt.Errorf("websocket not supported for exchange %s", r.Exchange)
	}

	exchCfg, err := s.Config.GetExchangeConfig(r.Exchange)
	if err != nil {
		return nil, err
	}

	if r.Enable {
		if err := w.Enable(context.WithoutCancel(ctx)); err != nil {
			return nil, err
		}
		exchCfg.Features.Enabled.Websocket = true
		return &gctrpc.GenericResponse{Status: MsgStatusSuccess, Data: "websocket enabled"}, nil
	}

	err = w.Disable()
	if err != nil {
		return nil, err
	}
	exchCfg.Features.Enabled.Websocket = false
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess, Data: "websocket disabled"}, nil
}

// WebsocketGetSubscriptions returns websocket subscription analysis
func (s *RPCServer) WebsocketGetSubscriptions(_ context.Context, r *gctrpc.WebsocketGetSubscriptionsRequest) (*gctrpc.WebsocketGetSubscriptionsResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	w, err := exch.GetWebsocket()
	if err != nil {
		return nil, fmt.Errorf("websocket not supported for exchange %s", r.Exchange)
	}

	payload := new(gctrpc.WebsocketGetSubscriptionsResponse)
	payload.Exchange = exch.GetName()
	subs := w.GetSubscriptions()
	for i := range subs {
		params, err := json.Marshal(subs[i].Params)
		if err != nil {
			return nil, err
		}
		payload.Subscriptions = append(payload.Subscriptions,
			&gctrpc.WebsocketSubscription{
				Channel: subs[i].Channel,
				Pairs:   subs[i].Pairs.Join(),
				Asset:   subs[i].Asset.String(),
				Params:  string(params),
			})
	}
	return payload, nil
}

// WebsocketSetProxy sets client websocket connection proxy
func (s *RPCServer) WebsocketSetProxy(ctx context.Context, r *gctrpc.WebsocketSetProxyRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	w, err := exch.GetWebsocket()
	if err != nil {
		return nil, fmt.Errorf("websocket not supported for exchange %s", r.Exchange)
	}

	if err := w.SetProxyAddress(context.WithoutCancel(ctx), r.Proxy); err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{
		Status: MsgStatusSuccess,
		Data:   fmt.Sprintf("new proxy has been set [%s] for %s websocket connection", r.Exchange, r.Proxy),
	}, nil
}

// WebsocketSetURL sets exchange websocket client connection URL
func (s *RPCServer) WebsocketSetURL(_ context.Context, r *gctrpc.WebsocketSetURLRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	w, err := exch.GetWebsocket()
	if err != nil {
		return nil, fmt.Errorf("websocket not supported for exchange %s", r.Exchange)
	}

	err = w.SetWebsocketURL(r.Url, false, true)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{
		Status: MsgStatusSuccess,
		Data: fmt.Sprintf("new URL has been set [%s] for %s websocket connection",
			r.Exchange,
			r.Url),
	}, nil
}

// GetSavedTrades returns trades from the database
func (s *RPCServer) GetSavedTrades(_ context.Context, r *gctrpc.GetSavedTradesRequest) (*gctrpc.SavedTradesResponse, error) {
	if r.End == "" || r.Start == "" || r.Exchange == "" || r.Pair == nil || r.AssetType == "" || r.Pair.String() == "" {
		return nil, errInvalidArguments
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.End)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	var trades []trade.Data
	trades, err = trade.GetTradesInRange(r.Exchange, r.AssetType, r.Pair.Base, r.Pair.Quote, start, end)
	if err != nil {
		return nil, err
	}
	resp := &gctrpc.SavedTradesResponse{
		ExchangeName: r.Exchange,
		Asset:        r.AssetType,
		Pair:         r.Pair,
	}
	for i := range trades {
		resp.Trades = append(resp.Trades, &gctrpc.SavedTrades{
			Price:     trades[i].Price,
			Amount:    trades[i].Amount,
			Side:      trades[i].Side.String(),
			Timestamp: trades[i].Timestamp.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			TradeId:   trades[i].TID,
		})
	}
	if len(resp.Trades) == 0 {
		return nil, fmt.Errorf("request for %v %v trade data between %v and %v and returned no results", r.Exchange, r.AssetType, r.Start, r.End)
	}
	return resp, nil
}

// ConvertTradesToCandles converts trades to candles using the interval requested
// returns the data too for extra fun scrutiny
func (s *RPCServer) ConvertTradesToCandles(_ context.Context, r *gctrpc.ConvertTradesToCandlesRequest) (*gctrpc.GetHistoricCandlesResponse, error) {
	if r.End == "" || r.Start == "" || r.Exchange == "" || r.Pair == nil || r.AssetType == "" || r.Pair.String() == "" || r.TimeInterval == 0 {
		return nil, errInvalidArguments
	}
	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.End)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	trades, err := trade.GetTradesInRange(r.Exchange, r.AssetType, r.Pair.Base, r.Pair.Quote, start, end)
	if err != nil {
		return nil, err
	}
	if len(trades) == 0 {
		return nil, errNoTrades
	}

	interval := kline.Interval(r.TimeInterval)
	klineItem, err := trade.ConvertTradesToCandles(interval, trades...)
	if err != nil {
		return nil, err
	}
	if len(klineItem.Candles) == 0 {
		return nil, errors.New("no candles generated from trades")
	}

	resp := &gctrpc.GetHistoricCandlesResponse{
		Exchange: r.Exchange,
		Pair:     r.Pair,
		Start:    r.Start,
		End:      r.End,
		Interval: interval.String(),
	}
	for i := range klineItem.Candles {
		resp.Candle = append(resp.Candle, &gctrpc.Candle{
			Time:      klineItem.Candles[i].Time.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			Low:       klineItem.Candles[i].Low,
			High:      klineItem.Candles[i].High,
			Open:      klineItem.Candles[i].Open,
			Close:     klineItem.Candles[i].Close,
			Volume:    klineItem.Candles[i].Volume,
			IsPartial: klineItem.Candles[i].ValidationIssues == kline.PartialCandle,
		})
	}

	if r.Sync {
		_, err = kline.StoreInDatabase(klineItem, r.Force)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// FindMissingSavedCandleIntervals is used to help determine what candle data is missing
func (s *RPCServer) FindMissingSavedCandleIntervals(_ context.Context, r *gctrpc.FindMissingCandlePeriodsRequest) (*gctrpc.FindMissingIntervalsResponse, error) {
	if r.End == "" || r.Start == "" || r.ExchangeName == "" || r.Pair == nil || r.AssetType == "" || r.Pair.String() == "" || r.Interval <= 0 {
		return nil, errInvalidArguments
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.ExchangeName)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.ExchangeName, exch, a, p)
	if err != nil {
		return nil, err
	}

	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.End)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	klineItem, err := kline.LoadFromDatabase(
		r.ExchangeName,
		p,
		a,
		kline.Interval(r.Interval),
		start,
		end,
	)
	if err != nil {
		return nil, err
	}
	resp := &gctrpc.FindMissingIntervalsResponse{
		ExchangeName:   r.ExchangeName,
		AssetType:      r.AssetType,
		Pair:           r.Pair,
		MissingPeriods: []string{},
	}
	candleTimes := make([]time.Time, len(klineItem.Candles))
	for i := range klineItem.Candles {
		candleTimes[i] = klineItem.Candles[i].Time
	}
	var ranges []timeperiods.TimeRange
	ranges, err = timeperiods.FindTimeRangesContainingData(start, end, klineItem.Interval.Duration(), candleTimes)
	if err != nil {
		return nil, err
	}
	foundCount := 0
	for i := range ranges {
		if !ranges[i].HasDataInRange {
			resp.MissingPeriods = append(resp.MissingPeriods,
				ranges[i].StartOfRange.UTC().Format(common.SimpleTimeFormatWithTimezone)+
					" - "+
					ranges[i].EndOfRange.UTC().Format(common.SimpleTimeFormatWithTimezone))
		} else {
			foundCount++
		}
	}

	if len(resp.MissingPeriods) == 0 {
		resp.Status = fmt.Sprintf("no missing candles found between %v and %v",
			r.Start,
			r.End,
		)
	} else {
		resp.Status = fmt.Sprintf("Found %v candles. Missing %v candles in requested timeframe starting %v ending %v",
			foundCount,
			len(resp.MissingPeriods),
			start.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			end.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone))
	}

	return resp, nil
}

// FindMissingSavedTradeIntervals is used to help determine what trade data is missing
func (s *RPCServer) FindMissingSavedTradeIntervals(_ context.Context, r *gctrpc.FindMissingTradePeriodsRequest) (*gctrpc.FindMissingIntervalsResponse, error) {
	if r.End == "" || r.Start == "" || r.ExchangeName == "" || r.Pair == nil || r.AssetType == "" || r.Pair.String() == "" {
		return nil, errInvalidArguments
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.ExchangeName)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.ExchangeName, exch, a, p)
	if err != nil {
		return nil, err
	}
	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.End)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	start = start.Truncate(time.Hour)
	end = end.Truncate(time.Hour)

	intervalMap := make(map[int64]bool)
	iterationTime := start
	for iterationTime.Before(end) {
		intervalMap[iterationTime.Unix()] = false
		iterationTime = iterationTime.Add(time.Hour)
	}

	var trades []trade.Data
	trades, err = trade.GetTradesInRange(
		r.ExchangeName,
		r.AssetType,
		r.Pair.Base,
		r.Pair.Quote,
		start,
		end,
	)
	if err != nil {
		return nil, err
	}
	resp := &gctrpc.FindMissingIntervalsResponse{
		ExchangeName:   r.ExchangeName,
		AssetType:      r.AssetType,
		Pair:           r.Pair,
		MissingPeriods: []string{},
	}
	tradeTimes := make([]time.Time, len(trades))
	for i := range trades {
		tradeTimes[i] = trades[i].Timestamp
	}
	var ranges []timeperiods.TimeRange
	ranges, err = timeperiods.FindTimeRangesContainingData(start, end, time.Hour, tradeTimes)
	if err != nil {
		return nil, err
	}
	foundCount := 0
	for i := range ranges {
		if !ranges[i].HasDataInRange {
			resp.MissingPeriods = append(resp.MissingPeriods,
				ranges[i].StartOfRange.UTC().Format(common.SimpleTimeFormatWithTimezone)+
					" - "+
					ranges[i].EndOfRange.UTC().Format(common.SimpleTimeFormatWithTimezone))
		} else {
			foundCount++
		}
	}

	if len(resp.MissingPeriods) == 0 {
		resp.Status = fmt.Sprintf("no missing periods found between %v and %v",
			r.Start,
			r.End,
		)
	} else {
		resp.Status = fmt.Sprintf("Found %v periods. Missing %v periods between %v and %v",
			foundCount,
			len(resp.MissingPeriods),
			start.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			end.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone))
	}

	return resp, nil
}

// SetExchangeTradeProcessing allows the setting of exchange trade processing
func (s *RPCServer) SetExchangeTradeProcessing(_ context.Context, r *gctrpc.SetExchangeTradeProcessingRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	b := exch.GetBase()
	b.SetSaveTradeDataStatus(r.Status)

	return &gctrpc.GenericResponse{
		Status: "success",
	}, nil
}

// GetHistoricTrades returns trades between a set of dates
func (s *RPCServer) GetHistoricTrades(r *gctrpc.GetSavedTradesRequest, stream gctrpc.GoCryptoTraderService_GetHistoricTradesServer) error {
	if r.Exchange == "" || r.Pair == nil || r.AssetType == "" || r.Pair.String() == "" {
		return errInvalidArguments
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return err
	}

	cp := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, cp)
	if err != nil {
		return err
	}
	var trades []trade.Data
	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.Start)
	if err != nil {
		return fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.End)
	if err != nil {
		return fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return err
	}
	resp := &gctrpc.SavedTradesResponse{
		ExchangeName: r.Exchange,
		Asset:        r.AssetType,
		Pair:         r.Pair,
	}

	for iterateStartTime := start; iterateStartTime.Before(end); iterateStartTime = iterateStartTime.Add(time.Hour) {
		iterateEndTime := iterateStartTime.Add(time.Hour)
		trades, err = exch.GetHistoricTrades(stream.Context(), cp, a, iterateStartTime, iterateEndTime)
		if err != nil {
			return err
		}
		if len(trades) == 0 {
			continue
		}
		grpcTrades := &gctrpc.SavedTradesResponse{
			ExchangeName: r.Exchange,
			Asset:        r.AssetType,
			Pair:         r.Pair,
		}
		for i := range trades {
			tradeTS := trades[i].Timestamp.In(time.UTC)
			if tradeTS.After(end) {
				break
			}
			grpcTrades.Trades = append(grpcTrades.Trades, &gctrpc.SavedTrades{
				Price:     trades[i].Price,
				Amount:    trades[i].Amount,
				Side:      trades[i].Side.String(),
				Timestamp: tradeTS.Format(common.SimpleTimeFormatWithTimezone),
				TradeId:   trades[i].TID,
			})
		}

		err = stream.Send(grpcTrades)
		if err != nil {
			return err
		}
	}
	return stream.Send(resp)
}

// GetRecentTrades returns trades
func (s *RPCServer) GetRecentTrades(ctx context.Context, r *gctrpc.GetSavedTradesRequest) (*gctrpc.SavedTradesResponse, error) {
	if r.Exchange == "" || r.Pair == nil || r.AssetType == "" || r.Pair.String() == "" {
		return nil, errInvalidArguments
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	cp := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, exch, a, cp)
	if err != nil {
		return nil, err
	}

	var trades []trade.Data
	trades, err = exch.GetRecentTrades(ctx, cp, a)
	if err != nil {
		return nil, err
	}
	resp := &gctrpc.SavedTradesResponse{
		ExchangeName: r.Exchange,
		Asset:        r.AssetType,
		Pair:         r.Pair,
	}
	for i := range trades {
		resp.Trades = append(resp.Trades, &gctrpc.SavedTrades{
			Price:     trades[i].Price,
			Amount:    trades[i].Amount,
			Side:      trades[i].Side.String(),
			Timestamp: trades[i].Timestamp.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			TradeId:   trades[i].TID,
		})
	}
	if len(resp.Trades) == 0 {
		return nil, fmt.Errorf("request for %v %v trade data and returned no results", r.Exchange, r.AssetType)
	}

	return resp, nil
}

func checkParams(exchName string, e exchange.IBotExchange, a asset.Item, p currency.Pair) error {
	if e == nil {
		return fmt.Errorf("%s %w", exchName, errExchangeNotLoaded)
	}
	if !e.IsEnabled() {
		return fmt.Errorf("%s %w", exchName, errExchangeNotEnabled)
	}
	if a.IsValid() {
		b := e.GetBase()
		if b == nil {
			return fmt.Errorf("%s %w", exchName, errExchangeBaseNotFound)
		}
		err := b.CurrencyPairs.IsAssetEnabled(a)
		if err != nil {
			return err
		}
	}
	if p.IsEmpty() {
		return nil
	}
	enabledPairs, err := e.GetEnabledPairs(a)
	if err != nil {
		return err
	}
	if enabledPairs.Contains(p, true) {
		return nil
	}
	availablePairs, err := e.GetAvailablePairs(a)
	if err != nil {
		return err
	}
	if availablePairs.Contains(p, true) {
		return fmt.Errorf("%v %w", p, errCurrencyNotEnabled)
	}
	return fmt.Errorf("%v %w", p, errCurrencyPairInvalid)
}

func parseMultipleEvents(ret []*withdraw.Response) *gctrpc.WithdrawalEventsByExchangeResponse {
	v := &gctrpc.WithdrawalEventsByExchangeResponse{}
	for x := range ret {
		tempEvent := &gctrpc.WithdrawalEventResponse{
			Id: ret[x].ID.String(),
			Exchange: &gctrpc.WithdrawlExchangeEvent{
				Name:   ret[x].Exchange.Name,
				Id:     ret[x].Exchange.ID,
				Status: ret[x].Exchange.Status,
			},
			Request: &gctrpc.WithdrawalRequestEvent{
				Currency:    ret[x].RequestDetails.Currency.String(),
				Description: ret[x].RequestDetails.Description,
				Amount:      ret[x].RequestDetails.Amount,
				Type:        int64(ret[x].RequestDetails.Type),
			},
		}

		tempEvent.CreatedAt = timestamppb.New(ret[x].CreatedAt)
		if err := tempEvent.CreatedAt.CheckValid(); err != nil {
			log.Errorf(log.Global, "withdrawal parseMultipleEvents CreatedAt: %s", err)
		}
		tempEvent.UpdatedAt = timestamppb.New(ret[x].UpdatedAt)
		if err := tempEvent.UpdatedAt.CheckValid(); err != nil {
			log.Errorf(log.Global, "withdrawal parseMultipleEvents UpdatedAt: %s", err)
		}

		switch ret[x].RequestDetails.Type {
		case withdraw.Crypto:
			tempEvent.Request.Crypto = new(gctrpc.CryptoWithdrawalEvent)
			tempEvent.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
				Address:    ret[x].RequestDetails.Crypto.Address,
				AddressTag: ret[x].RequestDetails.Crypto.AddressTag,
				Fee:        ret[x].RequestDetails.Crypto.FeeAmount,
			}
		case withdraw.Fiat:
			tempEvent.Request.Fiat = new(gctrpc.FiatWithdrawalEvent)
			tempEvent.Request.Fiat = &gctrpc.FiatWithdrawalEvent{
				BankName:      ret[x].RequestDetails.Fiat.Bank.BankName,
				AccountName:   ret[x].RequestDetails.Fiat.Bank.AccountName,
				AccountNumber: ret[x].RequestDetails.Fiat.Bank.AccountNumber,
				Bsb:           ret[x].RequestDetails.Fiat.Bank.BSBNumber,
				Swift:         ret[x].RequestDetails.Fiat.Bank.SWIFTCode,
				Iban:          ret[x].RequestDetails.Fiat.Bank.IBAN,
			}
		}

		v.Event = append(v.Event, tempEvent)
	}
	return v
}

func parseWithdrawalsHistory(ret []exchange.WithdrawalHistory, exchName string, limit int) *gctrpc.WithdrawalEventsByExchangeResponse {
	v := &gctrpc.WithdrawalEventsByExchangeResponse{}
	for x := range ret {
		if limit > 0 && x >= limit {
			return v
		}

		tempEvent := &gctrpc.WithdrawalEventResponse{
			Id: ret[x].TransferID,
			Exchange: &gctrpc.WithdrawlExchangeEvent{
				Name:   exchName,
				Status: ret[x].Status,
			},
			Request: &gctrpc.WithdrawalRequestEvent{
				Currency:    ret[x].Currency,
				Description: ret[x].Description,
				Amount:      ret[x].Amount,
			},
		}

		tempEvent.UpdatedAt = timestamppb.New(ret[x].Timestamp)
		if err := tempEvent.UpdatedAt.CheckValid(); err != nil {
			log.Errorf(log.Global, "withdrawal parseWithdrawalsHistory UpdatedAt: %s", err)
		}

		tempEvent.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
			Address: ret[x].CryptoToAddress,
			Fee:     ret[x].Fee,
			TxId:    ret[x].CryptoTxID,
		}

		v.Event = append(v.Event, tempEvent)
	}
	return v
}

func parseSingleEvents(ret *withdraw.Response) *gctrpc.WithdrawalEventsByExchangeResponse {
	tempEvent := &gctrpc.WithdrawalEventResponse{
		Id: ret.ID.String(),
		Exchange: &gctrpc.WithdrawlExchangeEvent{
			Name:   ret.Exchange.Name,
			Id:     ret.Exchange.Name,
			Status: ret.Exchange.Status,
		},
		Request: &gctrpc.WithdrawalRequestEvent{
			Currency:    ret.RequestDetails.Currency.String(),
			Description: ret.RequestDetails.Description,
			Amount:      ret.RequestDetails.Amount,
			Type:        int64(ret.RequestDetails.Type),
		},
	}
	tempEvent.CreatedAt = timestamppb.New(ret.CreatedAt)
	if err := tempEvent.CreatedAt.CheckValid(); err != nil {
		log.Errorf(log.Global, "withdrawal parseSingleEvents CreatedAt %s", err)
	}
	tempEvent.UpdatedAt = timestamppb.New(ret.UpdatedAt)
	if err := tempEvent.UpdatedAt.CheckValid(); err != nil {
		log.Errorf(log.Global, "withdrawal parseSingleEvents UpdatedAt: %s", err)
	}

	switch ret.RequestDetails.Type {
	case withdraw.Crypto:
		tempEvent.Request.Crypto = new(gctrpc.CryptoWithdrawalEvent)
		tempEvent.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
			Address:    ret.RequestDetails.Crypto.Address,
			AddressTag: ret.RequestDetails.Crypto.AddressTag,
			Fee:        ret.RequestDetails.Crypto.FeeAmount,
		}
	case withdraw.Fiat:
		if ret.RequestDetails.Fiat != (withdraw.FiatRequest{}) {
			tempEvent.Request.Fiat = new(gctrpc.FiatWithdrawalEvent)
			tempEvent.Request.Fiat = &gctrpc.FiatWithdrawalEvent{
				BankName:      ret.RequestDetails.Fiat.Bank.BankName,
				AccountName:   ret.RequestDetails.Fiat.Bank.AccountName,
				AccountNumber: ret.RequestDetails.Fiat.Bank.AccountNumber,
				Bsb:           ret.RequestDetails.Fiat.Bank.BSBNumber,
				Swift:         ret.RequestDetails.Fiat.Bank.SWIFTCode,
				Iban:          ret.RequestDetails.Fiat.Bank.IBAN,
			}
		}
	}

	return &gctrpc.WithdrawalEventsByExchangeResponse{
		Event: []*gctrpc.WithdrawalEventResponse{tempEvent},
	}
}

// UpsertDataHistoryJob adds or updates a data history job for the data history manager
// It will upsert the entry in the database and allow for the processing of the job
func (s *RPCServer) UpsertDataHistoryJob(_ context.Context, r *gctrpc.UpsertDataHistoryJobRequest) (*gctrpc.UpsertDataHistoryJobResponse, error) {
	if r == nil {
		return nil, errNilRequestData
	}
	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	e, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	p := currency.NewPairWithDelimiter(r.Pair.Base, r.Pair.Quote, r.Pair.Delimiter)

	err = checkParams(r.Exchange, e, a, p)
	if err != nil {
		return nil, err
	}

	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}

	job := DataHistoryJob{
		Nickname:                 r.Nickname,
		Exchange:                 r.Exchange,
		Asset:                    a,
		Pair:                     p,
		StartDate:                start,
		EndDate:                  end,
		Interval:                 kline.Interval(r.Interval),
		RunBatchLimit:            r.BatchSize,
		RequestSizeLimit:         r.RequestSizeLimit,
		DataType:                 dataHistoryDataType(r.DataType),
		MaxRetryAttempts:         r.MaxRetryAttempts,
		Status:                   dataHistoryStatusActive,
		OverwriteExistingData:    r.OverwriteExistingData,
		ConversionInterval:       kline.Interval(r.ConversionInterval),
		DecimalPlaceComparison:   r.DecimalPlaceComparison,
		SecondaryExchangeSource:  r.SecondaryExchangeName,
		IssueTolerancePercentage: r.IssueTolerancePercentage,
		ReplaceOnIssue:           r.ReplaceOnIssue,
		PrerequisiteJobNickname:  r.PrerequisiteJobNickname,
	}

	err = s.dataHistoryManager.UpsertJob(&job, r.InsertOnly)
	if err != nil {
		return nil, err
	}

	result, err := s.dataHistoryManager.GetByNickname(r.Nickname, false)
	if err != nil {
		return nil, fmt.Errorf("%s %w", r.Nickname, err)
	}

	return &gctrpc.UpsertDataHistoryJobResponse{
		JobId:   result.ID.String(),
		Message: "successfully upserted job: " + result.Nickname,
	}, nil
}

// GetDataHistoryJobDetails returns a data history job's details
// can request all data history results with r.FullDetails
func (s *RPCServer) GetDataHistoryJobDetails(_ context.Context, r *gctrpc.GetDataHistoryJobDetailsRequest) (*gctrpc.DataHistoryJob, error) {
	if r == nil {
		return nil, errNilRequestData
	}
	if r.Id == "" && r.Nickname == "" {
		return nil, errNicknameIDUnset
	}
	if r.Nickname != "" && r.Id != "" {
		return nil, errOnlyNicknameOrID
	}
	var (
		result     *DataHistoryJob
		err        error
		jobResults []*gctrpc.DataHistoryJobResult
	)

	if r.Id != "" {
		var id uuid.UUID
		id, err = uuid.FromString(r.Id)
		if err != nil {
			return nil, fmt.Errorf("%s %w", r.Id, err)
		}
		result, err = s.dataHistoryManager.GetByID(id)
		if err != nil {
			return nil, fmt.Errorf("%s %w", r.Id, err)
		}
	} else {
		result, err = s.dataHistoryManager.GetByNickname(r.Nickname, r.FullDetails)
		if err != nil {
			return nil, fmt.Errorf("%s %w", r.Nickname, err)
		}
		if r.FullDetails {
			for _, v := range result.Results {
				for i := range v {
					jobResults = append(jobResults, &gctrpc.DataHistoryJobResult{
						StartDate: v[i].IntervalStartDate.Format(time.DateTime),
						EndDate:   v[i].IntervalEndDate.Format(time.DateTime),
						HasData:   v[i].Status == dataHistoryStatusComplete,
						Message:   v[i].Result,
						RunDate:   v[i].Date.Format(time.DateTime),
					})
				}
			}
		}
	}
	return &gctrpc.DataHistoryJob{
		Id:       result.ID.String(),
		Nickname: result.Nickname,
		Exchange: result.Exchange,
		Asset:    result.Asset.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: result.Pair.Delimiter,
			Base:      result.Pair.Base.String(),
			Quote:     result.Pair.Quote.String(),
		},
		StartDate:                result.StartDate.Format(time.DateTime),
		EndDate:                  result.EndDate.Format(time.DateTime),
		Interval:                 int64(result.Interval.Duration()),
		RequestSizeLimit:         result.RequestSizeLimit,
		MaxRetryAttempts:         result.MaxRetryAttempts,
		BatchSize:                result.RunBatchLimit,
		Status:                   result.Status.String(),
		DataType:                 result.DataType.String(),
		ConversionInterval:       int64(result.ConversionInterval.Duration()),
		OverwriteExistingData:    result.OverwriteExistingData,
		PrerequisiteJobNickname:  result.PrerequisiteJobNickname,
		DecimalPlaceComparison:   result.DecimalPlaceComparison,
		SecondaryExchangeName:    result.SecondaryExchangeSource,
		IssueTolerancePercentage: result.IssueTolerancePercentage,
		ReplaceOnIssue:           result.ReplaceOnIssue,
		JobResults:               jobResults,
	}, nil
}

// GetActiveDataHistoryJobs returns any active data history job details
func (s *RPCServer) GetActiveDataHistoryJobs(_ context.Context, _ *gctrpc.GetInfoRequest) (*gctrpc.DataHistoryJobs, error) {
	jobs, err := s.dataHistoryManager.GetActiveJobs()
	if err != nil {
		return nil, err
	}

	response := make([]*gctrpc.DataHistoryJob, len(jobs))
	for i := range jobs {
		response[i] = &gctrpc.DataHistoryJob{
			Id:       jobs[i].ID.String(),
			Nickname: jobs[i].Nickname,
			Exchange: jobs[i].Exchange,
			Asset:    jobs[i].Asset.String(),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: jobs[i].Pair.Delimiter,
				Base:      jobs[i].Pair.Base.String(),
				Quote:     jobs[i].Pair.Quote.String(),
			},
			StartDate:                jobs[i].StartDate.Format(time.DateTime),
			EndDate:                  jobs[i].EndDate.Format(time.DateTime),
			Interval:                 int64(jobs[i].Interval.Duration()),
			RequestSizeLimit:         jobs[i].RequestSizeLimit,
			MaxRetryAttempts:         jobs[i].MaxRetryAttempts,
			BatchSize:                jobs[i].RunBatchLimit,
			Status:                   jobs[i].Status.String(),
			DataType:                 jobs[i].DataType.String(),
			ConversionInterval:       int64(jobs[i].ConversionInterval.Duration()),
			OverwriteExistingData:    jobs[i].OverwriteExistingData,
			PrerequisiteJobNickname:  jobs[i].PrerequisiteJobNickname,
			DecimalPlaceComparison:   jobs[i].DecimalPlaceComparison,
			SecondaryExchangeName:    jobs[i].SecondaryExchangeSource,
			IssueTolerancePercentage: jobs[i].IssueTolerancePercentage,
			ReplaceOnIssue:           jobs[i].ReplaceOnIssue,
		}
	}
	return &gctrpc.DataHistoryJobs{Results: response}, nil
}

// GetDataHistoryJobsBetween returns all jobs created between supplied dates
func (s *RPCServer) GetDataHistoryJobsBetween(_ context.Context, r *gctrpc.GetDataHistoryJobsBetweenRequest) (*gctrpc.DataHistoryJobs, error) {
	if r == nil {
		return nil, errNilRequestData
	}
	start, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormatWithTimezone, r.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start.Local(), end)
	if err != nil {
		return nil, err
	}

	jobs, err := s.dataHistoryManager.GetAllJobStatusBetween(start, end)
	if err != nil {
		return nil, err
	}
	respJobs := make([]*gctrpc.DataHistoryJob, len(jobs))
	for i := range jobs {
		respJobs[i] = &gctrpc.DataHistoryJob{
			Id:       jobs[i].ID.String(),
			Nickname: jobs[i].Nickname,
			Exchange: jobs[i].Exchange,
			Asset:    jobs[i].Asset.String(),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: jobs[i].Pair.Delimiter,
				Base:      jobs[i].Pair.Base.String(),
				Quote:     jobs[i].Pair.Quote.String(),
			},
			StartDate:                jobs[i].StartDate.Format(time.DateTime),
			EndDate:                  jobs[i].EndDate.Format(time.DateTime),
			Interval:                 int64(jobs[i].Interval.Duration()),
			RequestSizeLimit:         jobs[i].RequestSizeLimit,
			MaxRetryAttempts:         jobs[i].MaxRetryAttempts,
			BatchSize:                jobs[i].RunBatchLimit,
			Status:                   jobs[i].Status.String(),
			DataType:                 jobs[i].DataType.String(),
			ConversionInterval:       int64(jobs[i].ConversionInterval.Duration()),
			OverwriteExistingData:    jobs[i].OverwriteExistingData,
			PrerequisiteJobNickname:  jobs[i].PrerequisiteJobNickname,
			DecimalPlaceComparison:   jobs[i].DecimalPlaceComparison,
			SecondaryExchangeName:    jobs[i].SecondaryExchangeSource,
			IssueTolerancePercentage: jobs[i].IssueTolerancePercentage,
			ReplaceOnIssue:           jobs[i].ReplaceOnIssue,
		}
	}
	return &gctrpc.DataHistoryJobs{
		Results: respJobs,
	}, nil
}

// GetDataHistoryJobSummary provides a general look at how a data history job is going with the "resultSummaries" property
func (s *RPCServer) GetDataHistoryJobSummary(_ context.Context, r *gctrpc.GetDataHistoryJobDetailsRequest) (*gctrpc.DataHistoryJob, error) {
	if r == nil {
		return nil, errNilRequestData
	}
	if r.Nickname == "" {
		return nil, fmt.Errorf("get job summary %w", errNicknameUnset)
	}
	job, err := s.dataHistoryManager.GenerateJobSummary(r.Nickname)
	if err != nil {
		return nil, err
	}
	return &gctrpc.DataHistoryJob{
		Nickname: job.Nickname,
		Exchange: job.Exchange,
		Asset:    job.Asset.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: job.Pair.Delimiter,
			Base:      job.Pair.Base.String(),
			Quote:     job.Pair.Quote.String(),
		},
		StartDate:               job.StartDate.Format(time.DateTime),
		EndDate:                 job.EndDate.Format(time.DateTime),
		Interval:                int64(job.Interval.Duration()),
		Status:                  job.Status.String(),
		DataType:                job.DataType.String(),
		ConversionInterval:      int64(job.ConversionInterval.Duration()),
		OverwriteExistingData:   job.OverwriteExistingData,
		PrerequisiteJobNickname: job.PrerequisiteJobNickname,
		ResultSummaries:         job.ResultRanges,
	}, nil
}

// unixTimestamp returns given time in either unix seconds or unix nanoseconds, depending
// on the remoteControl/gRPC/timeInNanoSeconds boolean configuration.
func (s *RPCServer) unixTimestamp(x time.Time) int64 {
	if s.Config.RemoteControl.GRPC.TimeInNanoSeconds {
		return x.UnixNano()
	}
	return x.Unix()
}

// SetDataHistoryJobStatus sets a data history job's status
func (s *RPCServer) SetDataHistoryJobStatus(_ context.Context, r *gctrpc.SetDataHistoryJobStatusRequest) (*gctrpc.GenericResponse, error) {
	if r == nil {
		return nil, errNilRequestData
	}
	if r.Nickname == "" && r.Id == "" {
		return nil, errNicknameIDUnset
	}
	if r.Nickname != "" && r.Id != "" {
		return nil, errOnlyNicknameOrID
	}
	status := "success"
	err := s.dataHistoryManager.SetJobStatus(r.Nickname, r.Id, dataHistoryStatus(r.Status))
	if err != nil {
		log.Errorln(log.GRPCSys, err)
		status = "failed"
	}

	return &gctrpc.GenericResponse{Status: status}, err
}

// UpdateDataHistoryJobPrerequisite sets or removes a prerequisite job for an existing job
// if the prerequisite job is "", then the relationship is removed
func (s *RPCServer) UpdateDataHistoryJobPrerequisite(_ context.Context, r *gctrpc.UpdateDataHistoryJobPrerequisiteRequest) (*gctrpc.GenericResponse, error) {
	if r == nil {
		return nil, errNilRequestData
	}
	if r.Nickname == "" {
		return nil, errNicknameUnset
	}
	status := "success"
	err := s.dataHistoryManager.SetJobRelationship(r.PrerequisiteJobNickname, r.Nickname)
	if err != nil {
		return nil, err
	}
	if r.PrerequisiteJobNickname == "" {
		return &gctrpc.GenericResponse{Status: status, Data: fmt.Sprintf("Removed prerequisite from job '%v'", r.Nickname)}, nil
	}
	return &gctrpc.GenericResponse{Status: status, Data: fmt.Sprintf("Set job '%v' prerequisite job to '%v' and set status to paused", r.Nickname, r.PrerequisiteJobNickname)}, nil
}

// CurrencyStateGetAll returns a full snapshot of currency states, whether they
// are able to be withdrawn, deposited or traded on an exchange.
func (s *RPCServer) CurrencyStateGetAll(_ context.Context, r *gctrpc.CurrencyStateGetAllRequest) (*gctrpc.CurrencyStateResponse, error) {
	return s.currencyStateManager.GetAllRPC(r.Exchange)
}

// CurrencyStateWithdraw determines via RPC if the currency code is operational for
// withdrawal from an exchange
func (s *RPCServer) CurrencyStateWithdraw(_ context.Context, r *gctrpc.CurrencyStateWithdrawRequest) (*gctrpc.GenericResponse, error) {
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	return s.currencyStateManager.CanWithdrawRPC(r.Exchange,
		currency.NewCode(r.Code),
		ai)
}

// CurrencyStateDeposit determines via RPC if the currency code is operational for
// depositing to an exchange
func (s *RPCServer) CurrencyStateDeposit(_ context.Context, r *gctrpc.CurrencyStateDepositRequest) (*gctrpc.GenericResponse, error) {
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	return s.currencyStateManager.CanDepositRPC(r.Exchange,
		currency.NewCode(r.Code),
		ai)
}

// CurrencyStateTrading determines via RPC if the currency code is operational for trading
func (s *RPCServer) CurrencyStateTrading(_ context.Context, r *gctrpc.CurrencyStateTradingRequest) (*gctrpc.GenericResponse, error) {
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	return s.currencyStateManager.CanTradeRPC(r.Exchange,
		currency.NewCode(r.Code),
		ai)
}

// CurrencyStateTradingPair determines via RPC if the pair is operational for trading
func (s *RPCServer) CurrencyStateTradingPair(_ context.Context, r *gctrpc.CurrencyStateTradingPairRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	cp, err := currency.NewPairFromString(r.Pair)
	if err != nil {
		return nil, err
	}

	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, ai, cp)
	if err != nil {
		return nil, err
	}

	err = exch.CanTradePair(cp, ai)
	if err != nil {
		return nil, err
	}
	return s.currencyStateManager.CanTradePairRPC(r.Exchange,
		cp,
		ai)
}

func (s *RPCServer) buildFuturePosition(position *futures.Position, getFundingPayments, includeFundingRates, includeOrders, includePredictedRate bool) *gctrpc.FuturePosition {
	response := &gctrpc.FuturePosition{
		Exchange: position.Exchange,
		Asset:    position.Asset.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: position.Pair.Delimiter,
			Base:      position.Pair.Base.String(),
			Quote:     position.Pair.Quote.String(),
		},
		Status:           position.Status.String(),
		OpeningDate:      position.OpeningDate.Format(common.SimpleTimeFormatWithTimezone),
		OpeningDirection: position.OpeningDirection.String(),
		OpeningPrice:     position.OpeningPrice.String(),
		OpeningSize:      position.OpeningSize.String(),
		CurrentDirection: position.LatestDirection.String(),
		CurrentPrice:     position.LatestPrice.String(),
		CurrentSize:      position.LatestSize.String(),
		UnrealisedPnl:    position.UnrealisedPNL.String(),
		RealisedPnl:      position.RealisedPNL.String(),
		OrderCount:       int64(len(position.Orders)),
	}
	if getFundingPayments {
		var sum decimal.Decimal
		fundingData := &gctrpc.FundingData{}
		for i := range position.FundingRates.FundingRates {
			if includeFundingRates {
				fundingData.Rates = append(fundingData.Rates, &gctrpc.FundingRate{
					Date:    position.FundingRates.FundingRates[i].Time.Format(common.SimpleTimeFormatWithTimezone),
					Rate:    position.FundingRates.FundingRates[i].Rate.String(),
					Payment: position.FundingRates.FundingRates[i].Payment.String(),
				})
			}
			sum = sum.Add(position.FundingRates.FundingRates[i].Payment)
		}
		fundingData.PaymentSum = sum.String()
		response.FundingData = fundingData
		if includePredictedRate && !position.FundingRates.PredictedUpcomingRate.Time.IsZero() {
			fundingData.UpcomingRate = &gctrpc.FundingRate{
				Date: position.FundingRates.PredictedUpcomingRate.Time.Format(common.SimpleTimeFormatWithTimezone),
				Rate: position.FundingRates.PredictedUpcomingRate.Rate.String(),
			}
		}
	}

	if includeOrders {
		for i := range position.Orders {
			od := &gctrpc.OrderDetails{
				Exchange:      position.Orders[i].Exchange,
				Id:            position.Orders[i].OrderID,
				ClientOrderId: position.Orders[i].ClientOrderID,
				BaseCurrency:  position.Orders[i].Pair.Base.String(),
				QuoteCurrency: position.Orders[i].Pair.Quote.String(),
				AssetType:     position.Orders[i].AssetType.String(),
				OrderSide:     position.Orders[i].Side.String(),
				OrderType:     position.Orders[i].Type.String(),
				CreationTime:  position.Orders[i].Date.Format(common.SimpleTimeFormatWithTimezone),
				Status:        position.Orders[i].Status.String(),
				Price:         position.Orders[i].Price,
				Amount:        position.Orders[i].Cost,
				OpenVolume:    position.Orders[i].RemainingAmount,
				Fee:           position.Orders[i].Fee,
				Cost:          position.Orders[i].Cost,
			}
			if !position.Orders[i].LastUpdated.IsZero() {
				od.UpdateTime = position.Orders[i].LastUpdated.Format(common.SimpleTimeFormatWithTimezone)
			}
			for j := range position.Orders[i].Trades {
				od.Trades = append(od.Trades, &gctrpc.TradeHistory{
					CreationTime: position.Orders[i].Trades[j].Timestamp.Unix(),
					Id:           position.Orders[i].Trades[j].TID,
					Price:        position.Orders[i].Trades[j].Price,
					Amount:       position.Orders[i].Trades[j].Amount,
					Exchange:     position.Orders[i].Trades[j].Exchange,
					AssetType:    position.Orders[i].AssetType.String(),
					OrderSide:    position.Orders[i].Trades[j].Side.String(),
					Fee:          position.Orders[i].Trades[j].Fee,
					Total:        position.Orders[i].Trades[j].Total,
				})
			}
			response.Orders = append(response.Orders, od)
		}
	}
	return response
}

// GetManagedPosition returns an open positions from the order manager, no calling any API endpoints to return this information
func (s *RPCServer) GetManagedPosition(_ context.Context, r *gctrpc.GetManagedPositionRequest) (*gctrpc.GetManagedPositionsResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetManagedPositionRequest", common.ErrNilPointer)
	}
	if err := futures.CheckFundingRatePrerequisites(r.GetFundingPayments, r.IncludePredictedRate, r.GetFundingPayments); err != nil {
		return nil, err
	}
	if r.Pair == nil {
		return nil, fmt.Errorf("%w request pair", common.ErrNilPointer)
	}
	var (
		exch exchange.IBotExchange
		ai   asset.Item
		cp   currency.Pair
		err  error
	)
	exch, err = s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%w '%v'", errExchangeDisabled, exch.GetName())
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.OrderManagerPositionTracking {
		return nil, fmt.Errorf("%w OrderManagerPositionTracking for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}
	ai, err = asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if !ai.IsFutures() {
		return nil, fmt.Errorf("%w '%v'", futures.ErrNotFuturesAsset, ai)
	}
	cp, err = currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	if err != nil {
		return nil, err
	}
	err = checkParams(r.Exchange, exch, ai, cp)
	if err != nil {
		return nil, err
	}
	position, err := s.OrderManager.GetOpenFuturesPosition(r.Exchange, ai, cp)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetManagedPositionsResponse{Positions: []*gctrpc.FuturePosition{
		s.buildFuturePosition(position, r.GetFundingPayments, r.IncludeFullFundingRates, r.IncludeFullOrderData, r.IncludePredictedRate),
	}}, nil
}

// GetAllManagedPositions returns all open positions from the order manager, no calling any API endpoints to return this information
func (s *RPCServer) GetAllManagedPositions(_ context.Context, r *gctrpc.GetAllManagedPositionsRequest) (*gctrpc.GetManagedPositionsResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetAllManagedPositionsRequest", common.ErrNilPointer)
	}
	if err := futures.CheckFundingRatePrerequisites(r.GetFundingPayments, r.IncludePredictedRate, r.GetFundingPayments); err != nil {
		return nil, err
	}
	positions, err := s.OrderManager.GetAllOpenFuturesPositions()
	if err != nil {
		return nil, err
	}
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].OpeningDate.Before(positions[j].OpeningDate)
	})
	response := make([]*gctrpc.FuturePosition, len(positions))
	for i := range positions {
		response[i] = s.buildFuturePosition(&positions[i], r.GetFundingPayments, r.IncludeFullFundingRates, r.IncludeFullOrderData, r.IncludePredictedRate)
	}

	return &gctrpc.GetManagedPositionsResponse{Positions: response}, nil
}

// GetFuturesPositionsSummary returns a summary of futures positions for an exchange asset pair from the API
func (s *RPCServer) GetFuturesPositionsSummary(ctx context.Context, r *gctrpc.GetFuturesPositionsSummaryRequest) (*gctrpc.GetFuturesPositionsSummaryResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionsSummaryRequest", common.ErrNilPointer)
	}
	if r.Pair == nil {
		return nil, currency.ErrCurrencyPairEmpty
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.Positions {
		return nil, fmt.Errorf("%w futures position tracking for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if !ai.IsFutures() {
		return nil, fmt.Errorf("%s %w", ai, futures.ErrNotFuturesAsset)
	}
	enabledPairs, err := exch.GetEnabledPairs(ai)
	if err != nil {
		return nil, err
	}
	cp, err := enabledPairs.DeriveFrom(r.Pair.Base + r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	var underlying currency.Pair
	if r.UnderlyingPair != nil {
		underlying, err = currency.NewPairFromStrings(r.UnderlyingPair.Base, r.UnderlyingPair.Quote)
		if err != nil {
			return nil, err
		}
	}

	var stats *futures.PositionSummary
	stats, err = exch.GetFuturesPositionSummary(ctx, &futures.PositionSummaryRequest{
		Asset:          ai,
		Pair:           cp,
		UnderlyingPair: underlying,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot GetFuturesPositionSummary %w", err)
	}

	positionStats := &gctrpc.FuturesPositionStats{}
	if !stats.MaintenanceMarginRequirement.IsZero() {
		positionStats.MaintenanceMarginRequirement = stats.MaintenanceMarginRequirement.String()
	}
	if !stats.InitialMarginRequirement.IsZero() {
		positionStats.InitialMarginRequirement = stats.InitialMarginRequirement.String()
	}
	if !stats.CollateralUsed.IsZero() {
		positionStats.CollateralUsed = stats.CollateralUsed.String()
	}
	if !stats.MarkPrice.IsZero() {
		positionStats.MarkPrice = stats.MarkPrice.String()
	}
	if !stats.CurrentSize.IsZero() {
		positionStats.CurrentSize = stats.CurrentSize.String()
	}
	if !stats.ContractMultiplier.IsZero() {
		positionStats.ContractMultiplier = stats.ContractMultiplier.String()
	}
	if !stats.ContractSize.IsZero() {
		positionStats.ContractSize = stats.ContractSize.String()
	}
	if !stats.AverageOpenPrice.IsZero() {
		positionStats.AverageOpenPrice = stats.AverageOpenPrice.String()
	}
	if !stats.UnrealisedPNL.IsZero() {
		positionStats.RecentPnl = stats.UnrealisedPNL.String()
	}
	if !stats.MaintenanceMarginFraction.IsZero() {
		positionStats.MarginFraction = stats.MaintenanceMarginFraction.String()
	}
	if !stats.FreeCollateral.IsZero() {
		positionStats.FreeCollateral = stats.FreeCollateral.String()
	}
	if !stats.TotalCollateral.IsZero() {
		positionStats.TotalCollateral = stats.TotalCollateral.String()
	}
	if !stats.EstimatedLiquidationPrice.IsZero() {
		positionStats.EstimatedLiquidationPrice = stats.EstimatedLiquidationPrice.String()
	}
	if !stats.FrozenBalance.IsZero() {
		positionStats.FrozenBalance = stats.FrozenBalance.String()
	}
	if !stats.EquityOfCurrency.IsZero() {
		positionStats.EquityOfCurrency = stats.EquityOfCurrency.String()
	}
	if !stats.AvailableEquity.IsZero() {
		positionStats.AvailableEquity = stats.AvailableEquity.String()
	}
	if !stats.CashBalance.IsZero() {
		positionStats.CashBalance = stats.CashBalance.String()
	}
	if !stats.DiscountEquity.IsZero() {
		positionStats.DiscountEquity = stats.DiscountEquity.String()
	}
	if !stats.EquityUSD.IsZero() {
		positionStats.EquityUsd = stats.EquityUSD.String()
	}
	if !stats.IsolatedEquity.IsZero() {
		positionStats.IsolatedEquity = stats.IsolatedEquity.String()
	}
	if stats.ContractSettlementType != futures.UnsetSettlementType {
		positionStats.ContractSettlementType = stats.ContractSettlementType.String()
	}
	if !stats.IsolatedLiabilities.IsZero() {
		positionStats.IsolatedLiabilities = stats.IsolatedLiabilities.String()
	}
	if !stats.IsolatedUPL.IsZero() {
		positionStats.IsolatedUpl = stats.IsolatedUPL.String()
	}
	if !stats.NotionalLeverage.IsZero() {
		positionStats.NotionalLeverage = stats.NotionalLeverage.String()
	}
	if !stats.TotalEquity.IsZero() {
		positionStats.TotalEquity = stats.TotalEquity.String()
	}
	if !stats.StrategyEquity.IsZero() {
		positionStats.StrategyEquity = stats.StrategyEquity.String()
	}
	return &gctrpc.GetFuturesPositionsSummaryResponse{
		Exchange: exch.GetName(),
		Asset:    ai.String(),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: cp.Delimiter,
			Base:      cp.Base.String(),
			Quote:     cp.Quote.String(),
		},
		PositionStats: positionStats,
	}, nil
}

// GetFuturesPositionsOrders returns futures position orders from exchange API
func (s *RPCServer) GetFuturesPositionsOrders(ctx context.Context, r *gctrpc.GetFuturesPositionsOrdersRequest) (*gctrpc.GetFuturesPositionsOrdersResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetFuturesPositionsOrdersRequest", common.ErrNilPointer)
	}
	if r.Pair == nil {
		return nil, currency.ErrCurrencyPairEmpty
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.Positions {
		return nil, fmt.Errorf("%w futures position tracking for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}
	if r.SyncWithOrderManager && !feat.FuturesCapabilities.OrderManagerPositionTracking {
		return nil, fmt.Errorf("%w OrderManagerPositionTracking", common.ErrFunctionNotSupported)
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if !ai.IsFutures() {
		return nil, fmt.Errorf("%s %w", ai, futures.ErrNotFuturesAsset)
	}
	enabledPairs, err := exch.GetEnabledPairs(ai)
	if err != nil {
		return nil, err
	}
	cp, err := enabledPairs.DeriveFrom(r.Pair.Base + r.Pair.Quote)
	if err != nil {
		return nil, err
	}
	var start, end time.Time
	if r.StartDate != "" {
		start, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.StartDate)
		if err != nil {
			return nil, err
		}
	}
	if r.EndDate != "" {
		end, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.EndDate)
		if err != nil {
			return nil, err
		}
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil && !errors.Is(err, common.ErrDateUnset) {
		return nil, err
	}

	positionDetails, err := exch.GetFuturesPositionOrders(ctx, &futures.PositionsRequest{
		Asset:                     ai,
		Pairs:                     currency.Pairs{cp},
		StartDate:                 start,
		EndDate:                   end,
		RespectOrderHistoryLimits: r.RespectOrderHistoryLimits,
	})
	if err != nil {
		return nil, err
	}
	response := &gctrpc.GetFuturesPositionsOrdersResponse{}
	positions := make([]*gctrpc.FuturePosition, len(positionDetails))
	var anyOrders bool
	for i := range positionDetails {
		details := &gctrpc.FuturePosition{
			Exchange: exch.GetName(),
			Asset:    positionDetails[i].Asset.String(),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: positionDetails[i].Pair.Delimiter,
				Base:      positionDetails[i].Pair.Base.String(),
				Quote:     positionDetails[i].Pair.Quote.String(),
			},
			ContractSettlementType: positionDetails[i].ContractSettlementType.String(),
			Orders:                 make([]*gctrpc.OrderDetails, len(positionDetails[i].Orders)),
		}
		for j := range positionDetails[i].Orders {
			anyOrders = true
			details.Orders[j] = &gctrpc.OrderDetails{
				Exchange:       exch.GetName(),
				Id:             positionDetails[i].Orders[j].OrderID,
				ClientOrderId:  positionDetails[i].Orders[j].ClientOrderID,
				BaseCurrency:   positionDetails[i].Orders[j].Pair.Base.String(),
				QuoteCurrency:  positionDetails[i].Orders[j].Pair.Quote.String(),
				AssetType:      positionDetails[i].Orders[j].AssetType.String(),
				OrderSide:      positionDetails[i].Orders[j].Side.String(),
				OrderType:      positionDetails[i].Orders[j].Type.String(),
				CreationTime:   positionDetails[i].Orders[j].Date.Format(common.SimpleTimeFormatWithTimezone),
				UpdateTime:     positionDetails[i].Orders[j].LastUpdated.Format(common.SimpleTimeFormatWithTimezone),
				Status:         positionDetails[i].Orders[j].Status.String(),
				Price:          positionDetails[i].Orders[j].Price,
				Amount:         positionDetails[i].Orders[j].Amount,
				OpenVolume:     positionDetails[i].Orders[j].RemainingAmount,
				Fee:            positionDetails[i].Orders[j].Fee,
				Cost:           positionDetails[i].Orders[j].Cost,
				ContractAmount: positionDetails[i].Orders[j].ContractAmount,
			}
		}
		positions[i] = details
	}
	if !anyOrders {
		return &gctrpc.GetFuturesPositionsOrdersResponse{}, nil
	}
	response.Positions = positions
	if r.SyncWithOrderManager {
		for i := range positionDetails {
			err = s.OrderManager.processFuturesPositions(exch, &positionDetails[i])
			if err != nil {
				return nil, err
			}
		}
	}
	return response, nil
}

// GetFundingRates returns the funding rates for an exchange, asset, pair
func (s *RPCServer) GetFundingRates(ctx context.Context, r *gctrpc.GetFundingRatesRequest) (*gctrpc.GetFundingRatesResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetFundingRatesRequest", common.ErrNilPointer)
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.FundingRates {
		return nil, fmt.Errorf("%w FundingRates for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}
	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if !a.IsFutures() {
		return nil, fmt.Errorf("%s %w", a, futures.ErrNotFuturesAsset)
	}
	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()
	if r.StartDate != "" {
		start, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.StartDate)
		if err != nil {
			return nil, err
		}
	}
	if r.EndDate != "" {
		end, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.EndDate)
		if err != nil {
			return nil, err
		}
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil && !errors.Is(err, common.ErrDateUnset) {
		return nil, err
	}

	cp, err := exch.MatchSymbolWithAvailablePairs(r.Pair.Base+r.Pair.Quote, a, false)
	if err != nil {
		return nil, err
	}

	pairs, err := exch.GetEnabledPairs(a)
	if err != nil {
		return nil, err
	}

	if !pairs.Contains(cp, true) {
		return nil, fmt.Errorf("%w %v", currency.ErrPairNotEnabled, cp)
	}

	funding, err := exch.GetHistoricalFundingRates(ctx, &fundingrate.HistoricalRatesRequest{
		Asset:                a,
		Pair:                 cp,
		StartDate:            start,
		EndDate:              end,
		IncludePayments:      r.IncludePayments,
		IncludePredictedRate: r.IncludePredicted,
		RespectHistoryLimits: r.RespectHistoryLimits,
		PaymentCurrency:      currency.NewCode(r.PaymentCurrency),
	})
	if err != nil {
		return nil, err
	}
	var hasPayment bool
	var response gctrpc.GetFundingRatesResponse
	fundingData := &gctrpc.FundingData{
		Exchange: r.Exchange,
		Asset:    r.Asset,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: funding.Pair.Delimiter,
			Base:      funding.Pair.Base.String(),
			Quote:     funding.Pair.Quote.String(),
		},
		StartDate: start.Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   end.Format(common.SimpleTimeFormatWithTimezone),
		LatestRate: &gctrpc.FundingRate{
			Date: funding.LatestRate.Time.Format(common.SimpleTimeFormatWithTimezone),
			Rate: funding.LatestRate.Rate.String(),
		},
	}
	rates := make([]*gctrpc.FundingRate, len(funding.FundingRates))
	for j := range funding.FundingRates {
		rates[j] = &gctrpc.FundingRate{
			Rate: funding.FundingRates[j].Rate.String(),
			Date: funding.FundingRates[j].Time.Format(common.SimpleTimeFormatWithTimezone),
		}
		if r.IncludePayments {
			if !funding.FundingRates[j].Payment.IsZero() {
				hasPayment = true
			}
			rates[j].Payment = funding.FundingRates[j].Payment.String()
		}
	}
	if r.IncludePayments {
		fundingData.PaymentSum = funding.PaymentSum.String()
		fundingData.PaymentCurrency = funding.PaymentCurrency.String()
		if !hasPayment {
			fundingData.PaymentMessage = "no payments found for payment currency " + funding.PaymentCurrency.String() +
				" please ensure you have set the correct payment currency in the request"
		}
	}
	if !funding.TimeOfNextRate.IsZero() {
		fundingData.TimeOfNextRate = funding.TimeOfNextRate.Format(common.SimpleTimeFormatWithTimezone)
	}
	fundingData.Rates = rates
	if r.IncludePredicted {
		fundingData.UpcomingRate = &gctrpc.FundingRate{
			Date: funding.PredictedUpcomingRate.Time.Format(common.SimpleTimeFormatWithTimezone),
			Rate: funding.PredictedUpcomingRate.Rate.String(),
		}
	}
	response.Rates = fundingData

	return &response, nil
}

// GetLatestFundingRate returns the latest funding rate for an exchange, asset, pair
func (s *RPCServer) GetLatestFundingRate(ctx context.Context, r *gctrpc.GetLatestFundingRateRequest) (*gctrpc.GetLatestFundingRateResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetLatestFundingRateRequest", common.ErrNilPointer)
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if !a.IsFutures() {
		return nil, fmt.Errorf("%s %w", a, futures.ErrNotFuturesAsset)
	}

	cp, err := exch.MatchSymbolWithAvailablePairs(r.Pair.Base+r.Pair.Quote, a, false)
	if err != nil {
		return nil, err
	}

	pairs, err := exch.GetEnabledPairs(a)
	if err != nil {
		return nil, err
	}

	if !pairs.Contains(cp, true) {
		return nil, fmt.Errorf("%w %v", currency.ErrPairNotEnabled, cp)
	}

	fundingRates, err := exch.GetLatestFundingRates(ctx, &fundingrate.LatestRateRequest{
		Asset:                a,
		Pair:                 cp,
		IncludePredictedRate: r.IncludePredicted,
	})
	if err != nil {
		return nil, err
	}
	if len(fundingRates) != 1 {
		return nil, fmt.Errorf("expected 1 funding rate, received %v", len(fundingRates))
	}
	var response gctrpc.GetLatestFundingRateResponse
	fundingData := &gctrpc.FundingData{
		Exchange: r.Exchange,
		Asset:    r.Asset,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: fundingRates[0].Pair.Delimiter,
			Base:      fundingRates[0].Pair.Base.String(),
			Quote:     fundingRates[0].Pair.Quote.String(),
		},
		LatestRate: &gctrpc.FundingRate{
			Date: fundingRates[0].LatestRate.Time.Format(common.SimpleTimeFormatWithTimezone),
			Rate: fundingRates[0].LatestRate.Rate.String(),
		},
	}
	if !fundingRates[0].TimeOfNextRate.IsZero() {
		fundingData.TimeOfNextRate = fundingRates[0].TimeOfNextRate.Format(common.SimpleTimeFormatWithTimezone)
	}
	if r.IncludePredicted {
		fundingData.UpcomingRate = &gctrpc.FundingRate{
			Date: fundingRates[0].PredictedUpcomingRate.Time.Format(common.SimpleTimeFormatWithTimezone),
			Rate: fundingRates[0].PredictedUpcomingRate.Rate.String(),
		}
	}
	response.Rate = fundingData
	return &response, nil
}

// GetCollateral returns the total collateral for an exchange's asset
// as exchanges can scale collateral and represent it in a singular currency,
// a user can opt to include a breakdown by currency
func (s *RPCServer) GetCollateral(ctx context.Context, r *gctrpc.GetCollateralRequest) (*gctrpc.GetCollateralResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if f := exch.GetSupportedFeatures(); !f.FuturesCapabilities.Collateral {
		return nil, fmt.Errorf("%w Get Collateral for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}

	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	if err := checkParams(r.Exchange, exch, a, currency.EMPTYPAIR); err != nil {
		return nil, err
	}
	if !a.IsFutures() {
		return nil, fmt.Errorf("%s %w", a, futures.ErrNotFuturesAsset)
	}
	currBalances, err := exch.GetCachedCurrencyBalances(ctx, a)
	if err != nil {
		return nil, err
	}
	var spotPairs currency.Pairs
	if r.CalculateOffline {
		spotPairs, err = exch.GetAvailablePairs(asset.Spot)
		if err != nil {
			return nil, fmt.Errorf("GetCollateral offline calculation error via GetAvailablePairs %s %s", exch.GetName(), err)
		}
	}

	calculators := make([]futures.CollateralCalculator, 0, len(currBalances))
	for curr, balance := range currBalances {
		total := decimal.NewFromFloat(balance.Total)
		free := decimal.NewFromFloat(balance.AvailableWithoutBorrow)
		cal := futures.CollateralCalculator{
			CalculateOffline:   r.CalculateOffline,
			CollateralCurrency: curr,
			Asset:              a,
			FreeCollateral:     free,
			LockedCollateral:   total.Sub(free),
		}
		if r.CalculateOffline && !curr.Equal(currency.USD) {
			var tick *ticker.Price
			tickerCurr := currency.NewPair(curr, currency.USD)
			if !spotPairs.Contains(tickerCurr, true) {
				continue // cannot price currency to calculate collateral
			}
			tick, err = exch.GetCachedTicker(tickerCurr, asset.Spot)
			if err != nil {
				log.Errorf(log.GRPCSys, "GetCollateral offline calculation error via GetCachedTicker %s %s", exch.GetName(), err)
				continue
			}
			if tick.Last == 0 {
				continue
			}
			cal.USDPrice = decimal.NewFromFloat(tick.Last)
		}
		calculators = append(calculators, cal)
	}

	calc := &futures.TotalCollateralCalculator{
		CollateralAssets: calculators,
		CalculateOffline: r.CalculateOffline,
		FetchPositions:   true,
	}

	c, err := exch.CalculateTotalCollateral(ctx, calc)
	if err != nil {
		return nil, err
	}

	collateralDisplayCurrency := " " + c.CollateralCurrency.String()
	result := &gctrpc.GetCollateralResponse{
		CollateralCurrency:  c.CollateralCurrency.String(),
		AvailableCollateral: c.AvailableCollateral.String() + collateralDisplayCurrency,
		UsedCollateral:      c.UsedCollateral.String() + collateralDisplayCurrency,
	}
	if !c.CollateralContributedByPositiveSpotBalances.IsZero() {
		result.CollateralContributedByPositiveSpotBalances = c.CollateralContributedByPositiveSpotBalances.String() + collateralDisplayCurrency
	}
	if !c.TotalValueOfPositiveSpotBalances.IsZero() {
		result.TotalValueOfPositiveSpotBalances = c.TotalValueOfPositiveSpotBalances.String() + collateralDisplayCurrency
	}
	if !c.AvailableMaintenanceCollateral.IsZero() {
		result.MaintenanceCollateral = c.AvailableMaintenanceCollateral.String() + collateralDisplayCurrency
	}
	if !c.UnrealisedPNL.IsZero() {
		result.UnrealisedPnl = c.UnrealisedPNL.String()
	}
	if c.UsedBreakdown != nil {
		result.UsedBreakdown = &gctrpc.CollateralUsedBreakdown{}
		if !c.UsedBreakdown.LockedInStakes.IsZero() {
			result.UsedBreakdown.LockedInStakes = c.UsedBreakdown.LockedInStakes.String() + collateralDisplayCurrency
		}
		if !c.UsedBreakdown.LockedInNFTBids.IsZero() {
			result.UsedBreakdown.LockedInNftBids = c.UsedBreakdown.LockedInNFTBids.String() + collateralDisplayCurrency
		}
		if !c.UsedBreakdown.LockedInFeeVoucher.IsZero() {
			result.UsedBreakdown.LockedInFeeVoucher = c.UsedBreakdown.LockedInFeeVoucher.String() + collateralDisplayCurrency
		}
		if !c.UsedBreakdown.LockedInSpotMarginFundingOffers.IsZero() {
			result.UsedBreakdown.LockedInSpotMarginFundingOffers = c.UsedBreakdown.LockedInSpotMarginFundingOffers.String() + collateralDisplayCurrency
		}
		if !c.UsedBreakdown.LockedInSpotOrders.IsZero() {
			result.UsedBreakdown.LockedInSpotOrders = c.UsedBreakdown.LockedInSpotOrders.String() + collateralDisplayCurrency
		}
		if !c.UsedBreakdown.LockedAsCollateral.IsZero() {
			result.UsedBreakdown.LockedAsCollateral = c.UsedBreakdown.LockedAsCollateral.String() + collateralDisplayCurrency
		}
		if !c.UsedBreakdown.UsedInPositions.IsZero() {
			result.UsedBreakdown.UsedInFutures = c.UsedBreakdown.UsedInPositions.String() + collateralDisplayCurrency
		}
		if !c.UsedBreakdown.UsedInSpotMarginBorrows.IsZero() {
			result.UsedBreakdown.UsedInSpotMargin = c.UsedBreakdown.UsedInSpotMarginBorrows.String() + collateralDisplayCurrency
		}
	}
	if r.IncludeBreakdown {
		for i := range c.BreakdownOfPositions {
			result.PositionBreakdown = append(result.PositionBreakdown, &gctrpc.CollateralByPosition{
				Currency:            c.BreakdownOfPositions[i].PositionCurrency.String(),
				Size:                c.BreakdownOfPositions[i].Size.String(),
				OpenOrderSize:       c.BreakdownOfPositions[i].OpenOrderSize.String(),
				PositionSize:        c.BreakdownOfPositions[i].PositionSize.String(),
				MarkPrice:           c.BreakdownOfPositions[i].MarkPrice.String() + collateralDisplayCurrency,
				RequiredMargin:      c.BreakdownOfPositions[i].RequiredMargin.String(),
				TotalCollateralUsed: c.BreakdownOfPositions[i].CollateralUsed.String() + collateralDisplayCurrency,
			})
		}
		for i := range c.BreakdownByCurrency {
			if c.BreakdownByCurrency[i].TotalFunds.IsZero() && !r.IncludeZeroValues {
				continue
			}
			originalDisplayCurrency := " " + c.BreakdownByCurrency[i].Currency.String()
			cb := &gctrpc.CollateralForCurrency{
				Currency:                    c.BreakdownByCurrency[i].Currency.String(),
				ExcludedFromCollateral:      c.BreakdownByCurrency[i].SkipContribution,
				TotalFunds:                  c.BreakdownByCurrency[i].TotalFunds.String() + originalDisplayCurrency,
				AvailableForUseAsCollateral: c.BreakdownByCurrency[i].AvailableForUseAsCollateral.String() + originalDisplayCurrency,
				ApproxFairMarketValue:       c.BreakdownByCurrency[i].FairMarketValue.String() + collateralDisplayCurrency,
				Weighting:                   c.BreakdownByCurrency[i].Weighting.String(),
				CollateralContribution:      c.BreakdownByCurrency[i].CollateralContribution.String() + collateralDisplayCurrency,
				ScaledToCurrency:            c.BreakdownByCurrency[i].ScaledCurrency.String(),
			}
			if !c.BreakdownByCurrency[i].AdditionalCollateralUsed.IsZero() {
				cb.AdditionalCollateralUsed = c.BreakdownByCurrency[i].AdditionalCollateralUsed.String() + collateralDisplayCurrency
			}

			if !c.BreakdownByCurrency[i].ScaledUsed.IsZero() {
				cb.FundsInUse = c.BreakdownByCurrency[i].ScaledUsed.String() + collateralDisplayCurrency
			}
			if !c.BreakdownByCurrency[i].UnrealisedPNL.IsZero() {
				cb.UnrealisedPnl = c.BreakdownByCurrency[i].UnrealisedPNL.String() + collateralDisplayCurrency
			}
			if c.BreakdownByCurrency[i].ScaledUsedBreakdown != nil {
				breakDownDisplayCurrency := collateralDisplayCurrency
				if c.BreakdownByCurrency[i].Weighting.IsZero() && c.BreakdownByCurrency[i].FairMarketValue.IsZero() {
					// cannot determine value, show in like currency instead
					breakDownDisplayCurrency = originalDisplayCurrency
				}
				cb.UsedBreakdown = &gctrpc.CollateralUsedBreakdown{}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInStakes.IsZero() {
					cb.UsedBreakdown.LockedInStakes = c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInStakes.String() + breakDownDisplayCurrency
				}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInNFTBids.IsZero() {
					cb.UsedBreakdown.LockedInNftBids = c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInNFTBids.String() + breakDownDisplayCurrency
				}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInFeeVoucher.IsZero() {
					cb.UsedBreakdown.LockedInFeeVoucher = c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInFeeVoucher.String() + breakDownDisplayCurrency
				}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotMarginFundingOffers.IsZero() {
					cb.UsedBreakdown.LockedInSpotMarginFundingOffers = c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotMarginFundingOffers.String() + breakDownDisplayCurrency
				}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotOrders.IsZero() {
					cb.UsedBreakdown.LockedInSpotOrders = c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotOrders.String() + breakDownDisplayCurrency
				}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedAsCollateral.IsZero() {
					cb.UsedBreakdown.LockedAsCollateral = c.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedAsCollateral.String() + breakDownDisplayCurrency
				}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInPositions.IsZero() {
					cb.UsedBreakdown.UsedInFutures = c.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInPositions.String() + breakDownDisplayCurrency
				}
				if !c.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInSpotMarginBorrows.IsZero() {
					cb.UsedBreakdown.UsedInSpotMargin = c.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInSpotMarginBorrows.String() + breakDownDisplayCurrency
				}
			}
			if c.BreakdownByCurrency[i].Error != nil {
				cb.Error = c.BreakdownByCurrency[i].Error.Error()
			}
			result.CurrencyBreakdown = append(result.CurrencyBreakdown, cb)
		}
	}
	return result, nil
}

// Shutdown terminates bot session externally
func (s *RPCServer) Shutdown(_ context.Context, _ *gctrpc.ShutdownRequest) (*gctrpc.ShutdownResponse, error) {
	if !s.Engine.Settings.EnableGRPCShutdown {
		return nil, errShutdownNotAllowed
	}

	if s.Engine.GRPCShutdownSignal == nil {
		return nil, errGRPCShutdownSignalIsNil
	}

	s.Engine.GRPCShutdownSignal <- struct{}{}
	s.Engine.GRPCShutdownSignal = nil
	return &gctrpc.ShutdownResponse{}, nil
}

// GetTechnicalAnalysis using the requested technical analysis method will
// return a set(s) of signals for price action analysis.
func (s *RPCServer) GetTechnicalAnalysis(ctx context.Context, r *gctrpc.GetTechnicalAnalysisRequest) (*gctrpc.GetTechnicalAnalysisResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	as, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	pair, err := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	klines, err := exch.GetHistoricCandlesExtended(ctx, pair,
		as,
		kline.Interval(r.Interval),
		r.Start.AsTime(),
		r.End.AsTime())
	if err != nil {
		return nil, err
	}

	signals := make(map[string]*gctrpc.ListOfSignals)
	switch strings.ToUpper(r.AlgorithmType) {
	case "TWAP":
		var price float64
		price, err = klines.GetTWAP()
		if err != nil {
			return nil, err
		}
		signals["TWAP"] = &gctrpc.ListOfSignals{Signals: []float64{price}}
	case "VWAP":
		var prices []float64
		prices, err = klines.GetVWAPs()
		if err != nil {
			return nil, err
		}
		signals["VWAP"] = &gctrpc.ListOfSignals{Signals: prices}
	case "ATR":
		var prices []float64
		prices, err = klines.GetAverageTrueRange(r.Period)
		if err != nil {
			return nil, err
		}
		signals["ATR"] = &gctrpc.ListOfSignals{Signals: prices}
	case "BBANDS":
		var bollinger *kline.Bollinger
		bollinger, err = klines.GetBollingerBands(r.Period,
			r.StandardDeviationUp,
			r.StandardDeviationDown,
			indicators.MaType(r.MovingAverageType)) //nolint:gosec,nolintlint // TODO: Make var types consistent, however this doesn't get flagged on Windows
		if err != nil {
			return nil, err
		}
		signals["UPPER"] = &gctrpc.ListOfSignals{Signals: bollinger.Upper}
		signals["MIDDLE"] = &gctrpc.ListOfSignals{Signals: bollinger.Middle}
		signals["LOWER"] = &gctrpc.ListOfSignals{Signals: bollinger.Lower}
	case "COCO":
		otherExch := exch
		if r.OtherExchange != "" {
			otherExch, err = s.GetExchangeByName(r.OtherExchange)
			if err != nil {
				return nil, err
			}
		}

		otherAs := as
		if r.OtherAssetType != "" {
			otherAs, err = asset.New(r.OtherAssetType)
			if err != nil {
				return nil, err
			}
		}

		if r.OtherPair.String() == "" {
			return nil, errors.New("other pair is empty, to compare this must be specified")
		}

		var otherPair currency.Pair
		otherPair, err = currency.NewPairFromStrings(r.OtherPair.Base, r.OtherPair.Quote)
		if err != nil {
			return nil, err
		}

		var otherKlines *kline.Item
		otherKlines, err = otherExch.GetHistoricCandlesExtended(ctx,
			otherPair,
			otherAs,
			kline.Interval(r.Interval),
			r.Start.AsTime(),
			r.End.AsTime())
		if err != nil {
			return nil, err
		}

		var correlation []float64
		correlation, err = klines.GetCorrelationCoefficient(otherKlines, r.Period)
		if err != nil {
			return nil, err
		}
		signals["COCO"] = &gctrpc.ListOfSignals{Signals: correlation}
	case "SMA":
		var prices []float64
		prices, err = klines.GetSimpleMovingAverageOnClose(r.Period)
		if err != nil {
			return nil, err
		}
		signals["SMA"] = &gctrpc.ListOfSignals{Signals: prices}
	case "EMA":
		var prices []float64
		prices, err = klines.GetExponentialMovingAverageOnClose(r.Period)
		if err != nil {
			return nil, err
		}
		signals["EMA"] = &gctrpc.ListOfSignals{Signals: prices}
	case "MACD":
		var macd *kline.MACD
		macd, err = klines.GetMovingAverageConvergenceDivergenceOnClose(r.FastPeriod,
			r.SlowPeriod,
			r.Period)
		if err != nil {
			return nil, err
		}
		signals["MACD"] = &gctrpc.ListOfSignals{Signals: macd.Results}
		signals["SIGNAL"] = &gctrpc.ListOfSignals{Signals: macd.SignalVals}
		signals["HISTOGRAM"] = &gctrpc.ListOfSignals{Signals: macd.Histogram}
	case "MFI":
		var prices []float64
		prices, err = klines.GetMoneyFlowIndex(r.Period)
		if err != nil {
			return nil, err
		}
		signals["MFI"] = &gctrpc.ListOfSignals{Signals: prices}
	case "OBV":
		var prices []float64
		prices, err = klines.GetOnBalanceVolume()
		if err != nil {
			return nil, err
		}
		signals["OBV"] = &gctrpc.ListOfSignals{Signals: prices}
	case "RSI":
		var prices []float64
		prices, err = klines.GetRelativeStrengthIndexOnClose(r.Period)
		if err != nil {
			return nil, err
		}
		signals["RSI"] = &gctrpc.ListOfSignals{Signals: prices}
	default:
		return nil, fmt.Errorf("%w %q", errInvalidStrategy, r.AlgorithmType)
	}

	return &gctrpc.GetTechnicalAnalysisResponse{Signals: signals}, nil
}

// GetMarginRatesHistory returns the margin lending or borrow rates for an exchange, asset, currency along with many customisable options
func (s *RPCServer) GetMarginRatesHistory(ctx context.Context, r *gctrpc.GetMarginRatesHistoryRequest) (*gctrpc.GetMarginRatesHistoryResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetMarginRatesHistoryRequest", common.ErrNilPointer)
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, a, currency.EMPTYPAIR)
	if err != nil {
		return nil, err
	}

	c := currency.NewCode(r.Currency)
	pairs, err := exch.GetEnabledPairs(a)
	if err != nil {
		return nil, err
	}
	if !pairs.ContainsCurrency(c) {
		return nil, fmt.Errorf("%w '%v' in enabled pairs", currency.ErrCurrencyNotFound, r.Currency)
	}

	start := time.Now().AddDate(0, -1, 0)
	end := time.Now()
	if r.StartDate != "" {
		start, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.StartDate)
		if err != nil {
			return nil, err
		}
	}
	if r.EndDate != "" {
		end, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.EndDate)
		if err != nil {
			return nil, err
		}
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}

	req := &margin.RateHistoryRequest{
		Exchange:           exch.GetName(),
		Asset:              a,
		Currency:           c,
		StartDate:          start,
		EndDate:            end,
		GetPredictedRate:   r.GetPredictedRate,
		GetLendingPayments: r.GetLendingPayments,
		GetBorrowRates:     r.GetBorrowRates,
		GetBorrowCosts:     r.GetBorrowCosts,
		CalculateOffline:   r.CalculateOffline,
	}
	if req.CalculateOffline {
		if r.TakerFeeRate == "" {
			return nil, fmt.Errorf("%w for offline calculations", common.ErrCannotCalculateOffline)
		}
		req.TakeFeeRate, err = decimal.NewFromString(r.TakerFeeRate)
		if err != nil {
			return nil, err
		}

		if req.TakeFeeRate.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("%w for offline calculations", common.ErrCannotCalculateOffline)
		}
		if len(r.Rates) == 0 {
			return nil, fmt.Errorf("%w for offline calculations", common.ErrCannotCalculateOffline)
		}
		req.Rates = make([]margin.Rate, len(r.Rates))
		for i := range r.Rates {
			var offlineRate margin.Rate
			offlineRate.Time, err = time.Parse(common.SimpleTimeFormatWithTimezone, r.Rates[i].Time)
			if err != nil {
				return nil, err
			}

			offlineRate.HourlyRate, err = decimal.NewFromString(r.Rates[i].HourlyRate)
			if err != nil {
				return nil, err
			}

			if r.Rates[i].BorrowCost != nil {
				offlineRate.BorrowCost.Size, err = decimal.NewFromString(r.Rates[i].BorrowCost.Size)
				if err != nil {
					return nil, err
				}
			}
			if r.Rates[i].LendingPayment != nil {
				offlineRate.LendingPayment.Size, err = decimal.NewFromString(r.Rates[i].LendingPayment.Size)
				if err != nil {
					return nil, err
				}
			}

			req.Rates[i] = offlineRate
		}
	}

	lendingResp, err := exch.GetMarginRatesHistory(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(lendingResp.Rates) == 0 {
		return nil, order.ErrNoRates
	}
	resp := &gctrpc.GetMarginRatesHistoryResponse{
		LatestRate: &gctrpc.MarginRate{
			Time:             lendingResp.Rates[len(lendingResp.Rates)-1].Time.Format(common.SimpleTimeFormatWithTimezone),
			HourlyRate:       lendingResp.Rates[len(lendingResp.Rates)-1].HourlyRate.String(),
			YearlyRate:       lendingResp.Rates[len(lendingResp.Rates)-1].YearlyRate.String(),
			MarketBorrowSize: lendingResp.Rates[len(lendingResp.Rates)-1].MarketBorrowSize.String(),
		},
		TotalRates: int64(len(lendingResp.Rates)),
	}
	if r.GetBorrowRates {
		resp.LatestRate.HourlyBorrowRate = lendingResp.Rates[len(lendingResp.Rates)-1].HourlyBorrowRate.String()
		resp.LatestRate.YearlyBorrowRate = lendingResp.Rates[len(lendingResp.Rates)-1].YearlyBorrowRate.String()
	}
	if r.GetBorrowRates || r.GetLendingPayments {
		resp.TakerFeeRate = lendingResp.TakerFeeRate.String()
	}
	if r.GetLendingPayments {
		resp.SumLendingPayments = lendingResp.SumLendingPayments.String()
		resp.AvgLendingSize = lendingResp.AverageLendingSize.String()
	}
	if r.GetBorrowCosts {
		resp.SumBorrowCosts = lendingResp.SumBorrowCosts.String()
		resp.AvgBorrowSize = lendingResp.AverageBorrowSize.String()
	}
	if r.GetPredictedRate {
		resp.PredictedRate = &gctrpc.MarginRate{
			Time:       lendingResp.PredictedRate.Time.Format(common.SimpleTimeFormatWithTimezone),
			HourlyRate: lendingResp.PredictedRate.HourlyRate.String(),
			YearlyRate: lendingResp.PredictedRate.YearlyRate.String(),
		}
		if r.GetBorrowRates {
			resp.PredictedRate.HourlyBorrowRate = lendingResp.PredictedRate.HourlyBorrowRate.String()
			resp.PredictedRate.YearlyBorrowRate = lendingResp.PredictedRate.YearlyBorrowRate.String()
		}
	}
	if r.IncludeAllRates {
		resp.Rates = make([]*gctrpc.MarginRate, len(lendingResp.Rates))
		for i := range lendingResp.Rates {
			rate := &gctrpc.MarginRate{
				Time:             lendingResp.Rates[i].Time.Format(common.SimpleTimeFormatWithTimezone),
				HourlyRate:       lendingResp.Rates[i].HourlyRate.String(),
				YearlyRate:       lendingResp.Rates[i].YearlyRate.String(),
				MarketBorrowSize: lendingResp.Rates[i].MarketBorrowSize.String(),
			}
			if r.GetBorrowRates {
				rate.HourlyBorrowRate = lendingResp.Rates[i].HourlyBorrowRate.String()
				rate.YearlyBorrowRate = lendingResp.Rates[i].YearlyBorrowRate.String()
			}
			if r.GetBorrowCosts {
				rate.BorrowCost = &gctrpc.BorrowCost{
					Cost: lendingResp.Rates[i].BorrowCost.Cost.String(),
					Size: lendingResp.Rates[i].BorrowCost.Size.String(),
				}
			}
			if r.GetLendingPayments {
				rate.LendingPayment = &gctrpc.LendingPayment{
					Payment: lendingResp.Rates[i].LendingPayment.Payment.String(),
					Size:    lendingResp.Rates[i].LendingPayment.Size.String(),
				}
			}
			resp.Rates[i] = rate
		}
	}

	return resp, nil
}

// GetOrderbookMovement using the requested amount simulates a buy or sell and
// returns the nominal/impact percentages and costings.
func (s *RPCServer) GetOrderbookMovement(_ context.Context, r *gctrpc.GetOrderbookMovementRequest) (*gctrpc.GetOrderbookMovementResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	as, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	pair, err := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	err = checkParams(r.Exchange, exch, as, pair)
	if err != nil {
		return nil, err
	}

	depth, err := orderbook.GetDepth(exch.GetName(), pair, as)
	if err != nil {
		return nil, err
	}

	isRest, err := depth.IsRESTSnapshot()
	if err != nil {
		return nil, err
	}

	updateProtocol := "WEBSOCKET"
	if isRest {
		updateProtocol = "REST"
	}

	var move *orderbook.Movement
	var bought, sold, side string
	if r.Sell {
		move, err = depth.HitTheBidsFromBest(r.Amount, r.Purchase)
		bought = pair.Quote.Upper().String()
		sold = pair.Base.Upper().String()
		side = order.Bid.String()
	} else {
		move, err = depth.LiftTheAsksFromBest(r.Amount, r.Purchase)
		bought = pair.Base.Upper().String()
		sold = pair.Quote.Upper().String()
		side = order.Ask.String()
	}
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetOrderbookMovementResponse{
		NominalPercentage:         move.NominalPercentage,
		ImpactPercentage:          move.ImpactPercentage,
		SlippageCost:              move.SlippageCost,
		CurrencyBought:            bought,
		Bought:                    move.Purchased,
		CurrencySold:              sold,
		Sold:                      move.Sold,
		SideAffected:              side,
		UpdateProtocol:            updateProtocol,
		FullOrderbookSideConsumed: move.FullBookSideConsumed,
		NoSlippageOccurred:        move.ImpactPercentage == 0,
		StartPrice:                move.StartPrice,
		EndPrice:                  move.EndPrice,
		AverageOrderCost:          move.AverageOrderCost,
	}, nil
}

// GetOrderbookAmountByNominal using the requested nominal percentage requirement
// returns the amount on orderbook that can fit without exceeding that value.
func (s *RPCServer) GetOrderbookAmountByNominal(_ context.Context, r *gctrpc.GetOrderbookAmountByNominalRequest) (*gctrpc.GetOrderbookAmountByNominalResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	as, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	pair, err := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	err = checkParams(r.Exchange, exch, as, pair)
	if err != nil {
		return nil, err
	}

	depth, err := orderbook.GetDepth(exch.GetName(), pair, as)
	if err != nil {
		return nil, err
	}

	isRest, err := depth.IsRESTSnapshot()
	if err != nil {
		return nil, err
	}

	updateProtocol := "WEBSOCKET"
	if isRest {
		updateProtocol = "REST"
	}

	var nominal *orderbook.Movement
	var selling, buying, side string
	if r.Sell {
		nominal, err = depth.HitTheBidsByNominalSlippageFromBest(r.NominalPercentage)
		selling = pair.Upper().Base.String()
		buying = pair.Upper().Quote.String()
		side = order.Bid.String()
	} else {
		nominal, err = depth.LiftTheAsksByNominalSlippageFromBest(r.NominalPercentage)
		buying = pair.Upper().Base.String()
		selling = pair.Upper().Quote.String()
		side = order.Ask.String()
	}
	if err != nil {
		return nil, err
	}
	return &gctrpc.GetOrderbookAmountByNominalResponse{
		AmountRequired:                       nominal.Sold,
		CurrencySelling:                      selling,
		AmountReceived:                       nominal.Purchased,
		CurrencyBuying:                       buying,
		SideAffected:                         side,
		ApproximateNominalSlippagePercentage: nominal.NominalPercentage,
		UpdateProtocol:                       updateProtocol,
		FullOrderbookSideConsumed:            nominal.FullBookSideConsumed,
		StartPrice:                           nominal.StartPrice,
		EndPrice:                             nominal.EndPrice,
		AverageOrderCost:                     nominal.AverageOrderCost,
	}, nil
}

// GetOrderbookAmountByImpact using the requested impact percentage requirement
// determines the amount on orderbook that can fit that will slip the orderbook.
func (s *RPCServer) GetOrderbookAmountByImpact(_ context.Context, r *gctrpc.GetOrderbookAmountByImpactRequest) (*gctrpc.GetOrderbookAmountByImpactResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	as, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	pair, err := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	err = checkParams(r.Exchange, exch, as, pair)
	if err != nil {
		return nil, err
	}

	depth, err := orderbook.GetDepth(exch.GetName(), pair, as)
	if err != nil {
		return nil, err
	}

	isRest, err := depth.IsRESTSnapshot()
	if err != nil {
		return nil, err
	}

	updateProtocol := "WEBSOCKET"
	if isRest {
		updateProtocol = "REST"
	}

	var impact *orderbook.Movement
	var selling, buying, side string
	if r.Sell {
		impact, err = depth.HitTheBidsByImpactSlippageFromBest(r.ImpactPercentage)
		selling = pair.Upper().Base.String()
		buying = pair.Upper().Quote.String()
		side = order.Bid.String()
	} else {
		impact, err = depth.LiftTheAsksByImpactSlippageFromBest(r.ImpactPercentage)
		buying = pair.Upper().Base.String()
		selling = pair.Upper().Quote.String()
		side = order.Ask.String()
	}
	if err != nil {
		return nil, err
	}
	return &gctrpc.GetOrderbookAmountByImpactResponse{
		AmountRequired:                      impact.Sold,
		CurrencySelling:                     selling,
		AmountReceived:                      impact.Purchased,
		CurrencyBuying:                      buying,
		SideAffected:                        side,
		ApproximateImpactSlippagePercentage: impact.ImpactPercentage,
		UpdateProtocol:                      updateProtocol,
		FullOrderbookSideConsumed:           impact.FullBookSideConsumed,
		StartPrice:                          impact.StartPrice,
		EndPrice:                            impact.EndPrice,
		AverageOrderCost:                    impact.AverageOrderCost,
	}, nil
}

// GetCollateralMode returns the collateral type for the account asset
func (s *RPCServer) GetCollateralMode(ctx context.Context, r *gctrpc.GetCollateralModeRequest) (*gctrpc.GetCollateralModeResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetCollateralModeRequest", common.ErrNilPointer)
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.CollateralMode {
		return nil, fmt.Errorf("%w GetCollateralMode for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}

	item, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", exch.GetName(), errExchangeNotEnabled)
	}
	if !item.IsValid() {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, r.Asset)
	}
	b := exch.GetBase()
	if b == nil {
		return nil, fmt.Errorf("%s %w", exch.GetName(), errExchangeBaseNotFound)
	}
	err = b.CurrencyPairs.IsAssetEnabled(item)
	if err != nil {
		return nil, err
	}
	collateralMode, err := exch.GetCollateralMode(ctx, item)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GetCollateralModeResponse{
		Exchange:       r.Exchange,
		Asset:          r.Asset,
		CollateralMode: collateralMode.String(),
	}, nil
}

// SetCollateralMode sets the collateral type for the account asset
func (s *RPCServer) SetCollateralMode(ctx context.Context, r *gctrpc.SetCollateralModeRequest) (*gctrpc.SetCollateralModeResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w SetCollateralModeRequest", common.ErrNilPointer)
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", exch.GetName(), errExchangeNotEnabled)
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.CollateralMode {
		return nil, fmt.Errorf("%w SetCollateralMode for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}
	item, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	b := exch.GetBase()
	if b == nil {
		return nil, fmt.Errorf("%s %w", exch.GetName(), errExchangeBaseNotFound)
	}
	err = b.CurrencyPairs.IsAssetEnabled(item)
	if err != nil {
		return nil, fmt.Errorf("%v %w", item, err)
	}
	cm, err := collateral.StringToMode(r.CollateralMode)
	if err != nil {
		return nil, fmt.Errorf("%w %v", order.ErrCollateralInvalid, r.CollateralMode)
	}
	err = exch.SetCollateralMode(ctx, item, cm)
	if err != nil {
		return nil, err
	}
	return &gctrpc.SetCollateralModeResponse{
		Exchange: r.Exchange,
		Asset:    r.Asset,
		Success:  true,
	}, nil
}

// SetMarginType sets the margin type for the account asset pair
func (s *RPCServer) SetMarginType(ctx context.Context, r *gctrpc.SetMarginTypeRequest) (*gctrpc.SetMarginTypeResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w SetMarginTypeRequest", common.ErrNilPointer)
	}
	if r.Pair == nil {
		return nil, currency.ErrCurrencyPairEmpty
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	enabledPairs, err := exch.GetEnabledPairs(ai)
	if err != nil {
		return nil, err
	}
	cp, err := enabledPairs.DeriveFrom(r.Pair.Base + r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	mt, err := margin.StringToMarginType(r.MarginType)
	if err != nil {
		return nil, err
	}

	err = exch.SetMarginType(ctx, ai, cp, mt)
	if err != nil {
		return nil, err
	}

	return &gctrpc.SetMarginTypeResponse{
		Exchange: r.Exchange,
		Asset:    r.Asset,
		Pair:     r.Pair,
		Success:  true,
	}, nil
}

// GetLeverage returns the leverage for the account asset pair
func (s *RPCServer) GetLeverage(ctx context.Context, r *gctrpc.GetLeverageRequest) (*gctrpc.GetLeverageResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetLeverageRequest", common.ErrNilPointer)
	}
	if r.Pair == nil {
		return nil, currency.ErrCurrencyPairEmpty
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.Leverage {
		return nil, fmt.Errorf("%w futures position tracking for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	enabledPairs, err := exch.GetEnabledPairs(ai)
	if err != nil {
		return nil, err
	}
	cp, err := enabledPairs.DeriveFrom(r.Pair.Base + r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	mt, err := margin.StringToMarginType(r.MarginType)
	if err != nil {
		return nil, err
	}

	var orderSide order.Side
	if r.OrderSide != "" {
		orderSide, err = order.StringToOrderSide(r.OrderSide)
		if err != nil {
			return nil, err
		}
	}

	leverage, err := exch.GetLeverage(ctx, ai, cp, mt, orderSide)
	if err != nil {
		return nil, err
	}

	return &gctrpc.GetLeverageResponse{
		Exchange:   r.Exchange,
		Asset:      r.Asset,
		Pair:       r.Pair,
		MarginType: r.MarginType,
		Leverage:   leverage,
		OrderSide:  r.OrderSide,
	}, nil
}

// SetLeverage sets the leverage for the account asset pair
func (s *RPCServer) SetLeverage(ctx context.Context, r *gctrpc.SetLeverageRequest) (*gctrpc.SetLeverageResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w SetLeverageRequest", common.ErrNilPointer)
	}
	if r.Pair == nil {
		return nil, currency.ErrCurrencyPairEmpty
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.Leverage {
		return nil, fmt.Errorf("%w futures position tracking for exchange %v", common.ErrFunctionNotSupported, exch.GetName())
	}
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	enabledPairs, err := exch.GetEnabledPairs(ai)
	if err != nil {
		return nil, err
	}
	cp, err := enabledPairs.DeriveFrom(r.Pair.Base + r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	mt, err := margin.StringToMarginType(r.MarginType)
	if err != nil {
		return nil, err
	}

	var orderSide order.Side
	if r.OrderSide != "" {
		orderSide, err = order.StringToOrderSide(r.OrderSide)
		if err != nil {
			return nil, err
		}
	}

	err = exch.SetLeverage(ctx, ai, cp, mt, r.Leverage, orderSide)
	if err != nil {
		return nil, err
	}

	return &gctrpc.SetLeverageResponse{
		Exchange:   r.Exchange,
		Asset:      r.Asset,
		Pair:       r.Pair,
		MarginType: r.MarginType,
		OrderSide:  r.OrderSide,
		Success:    true,
	}, nil
}

// ChangePositionMargin sets a position's margin
func (s *RPCServer) ChangePositionMargin(ctx context.Context, r *gctrpc.ChangePositionMarginRequest) (*gctrpc.ChangePositionMarginResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w ChangePositionMarginRequest", common.ErrNilPointer)
	}
	if r.Pair == nil {
		return nil, currency.ErrCurrencyPairEmpty
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	enabledPairs, err := exch.GetEnabledPairs(ai)
	if err != nil {
		return nil, err
	}
	cp, err := enabledPairs.DeriveFrom(r.Pair.Base + r.Pair.Quote)
	if err != nil {
		return nil, err
	}

	mt, err := margin.StringToMarginType(r.MarginType)
	if err != nil {
		return nil, err
	}
	resp, err := exch.ChangePositionMargin(ctx, &margin.PositionChangeRequest{
		Exchange:                exch.GetName(),
		Pair:                    cp,
		Asset:                   ai,
		MarginType:              mt,
		OriginalAllocatedMargin: r.OriginalAllocatedMargin,
		NewAllocatedMargin:      r.NewAllocatedMargin,
		MarginSide:              r.MarginSide,
	})
	if err != nil {
		return nil, err
	}

	return &gctrpc.ChangePositionMarginResponse{
		Exchange:           r.Exchange,
		Asset:              r.Asset,
		Pair:               r.Pair,
		MarginType:         r.MarginType,
		NewAllocatedMargin: resp.AllocatedMargin,
		MarginSide:         r.MarginSide,
	}, nil
}

// GetOpenInterest fetches the open interest from the exchange
func (s *RPCServer) GetOpenInterest(ctx context.Context, r *gctrpc.GetOpenInterestRequest) (*gctrpc.GetOpenInterestResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetOpenInterestRequest", common.ErrNilPointer)
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	feat := exch.GetSupportedFeatures()
	if !feat.FuturesCapabilities.OpenInterest.Supported {
		return nil, common.ErrFunctionNotSupported
	}
	keys := make([]key.PairAsset, len(r.Data))
	for i := range r.Data {
		var a asset.Item
		a, err = asset.New(r.Data[i].Asset)
		if err != nil {
			return nil, err
		}
		keys[i].Base = currency.NewCode(r.Data[i].Pair.Base).Item
		keys[i].Quote = currency.NewCode(r.Data[i].Pair.Quote).Item
		keys[i].Asset = a
	}

	openInterest, err := exch.GetOpenInterest(ctx, keys...)
	if err != nil {
		return nil, err
	}

	resp := make([]*gctrpc.OpenInterestDataResponse, len(openInterest))
	for i := range openInterest {
		resp[i] = &gctrpc.OpenInterestDataResponse{
			Exchange: openInterest[i].Key.Exchange,
			Pair: &gctrpc.CurrencyPair{
				Base:  openInterest[i].Key.Base.String(),
				Quote: openInterest[i].Key.Quote.String(),
			},
			Asset:        openInterest[i].Key.Asset.String(),
			OpenInterest: openInterest[i].OpenInterest,
		}
	}
	return &gctrpc.GetOpenInterestResponse{
		Data: resp,
	}, nil
}

// GetCurrencyTradeURL returns the URL for the trading pair
func (s *RPCServer) GetCurrencyTradeURL(ctx context.Context, r *gctrpc.GetCurrencyTradeURLRequest) (*gctrpc.GetCurrencyTradeURLResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetCurrencyTradeURLRequest", common.ErrNilPointer)
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	if !exch.IsEnabled() {
		return nil, fmt.Errorf("%s %w", r.Exchange, errExchangeNotEnabled)
	}
	ai, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if r.Pair == nil ||
		(r.Pair.Base == "" && r.Pair.Quote == "") {
		return nil, currency.ErrCurrencyPairEmpty
	}
	cp, err := exch.MatchSymbolWithAvailablePairs(r.Pair.Base+r.Pair.Quote, ai, false)
	if err != nil {
		return nil, err
	}
	url, err := exch.GetCurrencyTradeURL(ctx, ai, cp)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GetCurrencyTradeURLResponse{
		Url: url,
	}, nil
}
