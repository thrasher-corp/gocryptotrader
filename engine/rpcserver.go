package engine

import (
	"context"
	"encoding/json"
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
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/common/file/archive"
	"github.com/thrasher-corp/gocryptotrader/common/timeperiods"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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
	errExchangeNameUnset       = errors.New("exchange name unset")
	errCurrencyPairUnset       = errors.New("currency pair unset")
	errInvalidTimes            = errors.New("invalid start and end times")
	errAssetTypeDisabled       = errors.New("asset type is disabled")
	errAssetTypeUnset          = errors.New("asset type unset")
	errDispatchSystem          = errors.New("dispatch system offline")
	errCurrencyNotEnabled      = errors.New("currency not enabled")
	errCurrencyNotSpecified    = errors.New("a currency must be specified")
	errCurrencyPairInvalid     = errors.New("currency provided is not found in the available pairs list")
	errNoTrades                = errors.New("no trades returned from supplied params")
	errUnexpectedResponseSize  = errors.New("unexpected slice size")
	errNilRequestData          = errors.New("nil request data received, cannot continue")
	errNoAccountInformation    = errors.New("account information does not exist")
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

	cred := strings.Split(string(decoded), ":")
	username := cred[0]
	password := cred[1]

	if username != s.Config.RemoteControl.Username ||
		password != s.Config.RemoteControl.Password {
		return ctx, fmt.Errorf("username/password mismatch")
	}
	ctx, err = account.ParseCredentialsMetadata(ctx, md)
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
	lis, err := net.Listen("tcp", engine.Config.RemoteControl.GRPC.ListenAddress)
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
	log.Debugf(log.GRPCSys, "gRPC proxy server support enabled. Starting gRPC proxy server on http://%v.\n", s.Config.RemoteControl.GRPC.GRPCProxyListenAddress)

	targetDir := utils.GetTLSDir(s.Settings.DataDir)
	creds, err := credentials.NewClientTLSFromFile(filepath.Join(targetDir, "cert.pem"), "")
	if err != nil {
		log.Errorf(log.GRPCSys, "Unabled to start gRPC proxy. Err: %s\n", err)
		return
	}

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
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
		}

		if err = server.ListenAndServe(); err != nil {
			log.Errorf(log.GRPCSys, "GRPC proxy failed to server: %s\n", err)
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC proxy server started!")
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
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("subsystem %s enabled", r.Subsystem)}, nil
}

// DisableSubsystem disables a engine subsystem
func (s *RPCServer) DisableSubsystem(_ context.Context, r *gctrpc.GenericSubsystemRequest) (*gctrpc.GenericResponse, error) {
	err := s.SetSubsystem(r.Subsystem, false)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("subsystem %s disabled", r.Subsystem)}, nil
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
	err := s.LoadExchange(r.Exchange, nil)
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
func (s *RPCServer) GetTicker(ctx context.Context, r *gctrpc.GetTickerRequest) (*gctrpc.TickerResponse, error) {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	e, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, e, a, currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	})
	if err != nil {
		return nil, err
	}

	t, err := s.GetSpecificTicker(ctx,
		currency.Pair{
			Delimiter: r.Pair.Delimiter,
			Base:      currency.NewCode(r.Pair.Base),
			Quote:     currency.NewCode(r.Pair.Quote),
		},
		r.Exchange,
		a,
	)
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
func (s *RPCServer) GetTickers(ctx context.Context, _ *gctrpc.GetTickersRequest) (*gctrpc.GetTickersResponse, error) {
	activeTickers := s.GetAllActiveTickers(ctx)
	tickers := make([]*gctrpc.Tickers, len(activeTickers))

	for x := range activeTickers {
		t := &gctrpc.Tickers{
			Exchange: activeTickers[x].ExchangeName,
			Tickers:  make([]*gctrpc.TickerResponse, len(activeTickers[x].ExchangeValues)),
		}
		for y := range activeTickers[x].ExchangeValues {
			val := activeTickers[x].ExchangeValues[y]
			t.Tickers[y] = &gctrpc.TickerResponse{
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
		tickers[x] = t
	}

	return &gctrpc.GetTickersResponse{Tickers: tickers}, nil
}

// GetOrderbook returns an orderbook for a specific exchange, currency pair
// and asset type
func (s *RPCServer) GetOrderbook(ctx context.Context, r *gctrpc.GetOrderbookRequest) (*gctrpc.OrderbookResponse, error) {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	ob, err := s.GetSpecificOrderbook(ctx,
		currency.Pair{
			Delimiter: r.Pair.Delimiter,
			Base:      currency.NewCode(r.Pair.Base),
			Quote:     currency.NewCode(r.Pair.Quote),
		},
		r.Exchange,
		a,
	)
	if err != nil {
		return nil, err
	}

	bids := make([]*gctrpc.OrderbookItem, 0, len(ob.Bids))
	asks := make([]*gctrpc.OrderbookItem, 0, len(ob.Asks))
	ch := make(chan bool)

	go func() {
		for _, b := range ob.Bids {
			bids = append(bids, &gctrpc.OrderbookItem{
				Amount: b.Amount,
				Price:  b.Price,
			})
		}
		ch <- true
	}()

	for _, a := range ob.Asks {
		asks = append(asks, &gctrpc.OrderbookItem{
			Amount: a.Amount,
			Price:  a.Price,
		})
	}
	<-ch

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
func (s *RPCServer) GetOrderbooks(ctx context.Context, _ *gctrpc.GetOrderbooksRequest) (*gctrpc.GetOrderbooksResponse, error) {
	exchanges, err := s.ExchangeManager.GetExchanges()
	if err != nil {
		return nil, err
	}
	obResponse := make([]*gctrpc.Orderbooks, 0, len(exchanges))
	var obs []*gctrpc.OrderbookResponse
	for x := range exchanges {
		if !exchanges[x].IsEnabled() {
			continue
		}
		assets := exchanges[x].GetAssetTypes(true)
		exchName := exchanges[x].GetName()
		for y := range assets {
			currencies, err := exchanges[x].GetEnabledPairs(assets[y])
			if err != nil {
				log.Errorf(log.RESTSys,
					"Exchange %s could not retrieve enabled currencies. Err: %s\n",
					exchName,
					err)
				continue
			}
			for z := range currencies {
				resp, err := exchanges[x].FetchOrderbook(ctx, currencies[z], assets[y])
				if err != nil {
					log.Errorf(log.RESTSys,
						"Exchange %s failed to retrieve %s orderbook. Err: %s\n", exchName,
						currencies[z].String(),
						err)
					continue
				}
				ob := &gctrpc.OrderbookResponse{
					Pair: &gctrpc.CurrencyPair{
						Delimiter: currencies[z].Delimiter,
						Base:      currencies[z].Base.String(),
						Quote:     currencies[z].Quote.String(),
					},
					AssetType:   assets[y].String(),
					LastUpdated: s.unixTimestamp(resp.LastUpdated),
					Bids:        make([]*gctrpc.OrderbookItem, len(resp.Bids)),
					Asks:        make([]*gctrpc.OrderbookItem, len(resp.Asks)),
				}
				for i := range resp.Bids {
					ob.Bids[i] = &gctrpc.OrderbookItem{
						Amount: resp.Bids[i].Amount,
						Price:  resp.Bids[i].Price,
					}
				}
				for i := range resp.Asks {
					ob.Asks[i] = &gctrpc.OrderbookItem{
						Amount: resp.Asks[i].Amount,
						Price:  resp.Asks[i].Price,
					}
				}
				obs = append(obs, ob)
			}
		}
		obResponse = append(obResponse, &gctrpc.Orderbooks{
			Exchange:   exchanges[x].GetName(),
			Orderbooks: obs,
		})
	}

	return &gctrpc.GetOrderbooksResponse{Orderbooks: obResponse}, nil
}

// GetAccountInfo returns an account balance for a specific exchange
func (s *RPCServer) GetAccountInfo(ctx context.Context, r *gctrpc.GetAccountInfoRequest) (*gctrpc.GetAccountInfoResponse, error) {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, assetType, currency.EMPTYPAIR)
	if err != nil {
		return nil, err
	}

	resp, err := exch.FetchAccountInfo(ctx, assetType)
	if err != nil {
		return nil, err
	}

	return createAccountInfoRequest(resp)
}

// UpdateAccountInfo forces an update of the account info
func (s *RPCServer) UpdateAccountInfo(ctx context.Context, r *gctrpc.GetAccountInfoRequest) (*gctrpc.GetAccountInfoResponse, error) {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, assetType, currency.EMPTYPAIR)
	if err != nil {
		return nil, err
	}

	resp, err := exch.UpdateAccountInfo(ctx, assetType)
	if err != nil {
		return nil, err
	}

	return createAccountInfoRequest(resp)
}

func createAccountInfoRequest(h account.Holdings) (*gctrpc.GetAccountInfoResponse, error) {
	accounts := make([]*gctrpc.Account, len(h.Accounts))
	for x := range h.Accounts {
		var a gctrpc.Account
		a.Id = h.Accounts[x].Credentials.String()
		for _, y := range h.Accounts[x].Currencies {
			if y.Total == 0 &&
				y.Hold == 0 &&
				y.Free == 0 &&
				y.AvailableWithoutBorrow == 0 &&
				y.Borrowed == 0 {
				continue
			}
			a.Currencies = append(a.Currencies, &gctrpc.AccountCurrencyInfo{
				Currency:          y.Currency.String(),
				TotalValue:        y.Total,
				Hold:              y.Hold,
				Free:              y.Free,
				FreeWithoutBorrow: y.AvailableWithoutBorrow,
				Borrowed:          y.Borrowed,
			})
		}
		accounts[x] = &a
	}

	return &gctrpc.GetAccountInfoResponse{Exchange: h.Exchange, Accounts: accounts}, nil
}

// GetAccountInfoStream streams an account balance for a specific exchange
func (s *RPCServer) GetAccountInfoStream(r *gctrpc.GetAccountInfoRequest, stream gctrpc.GoCryptoTraderService_GetAccountInfoStreamServer) error {
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

	initAcc, err := exch.FetchAccountInfo(stream.Context(), assetType)
	if err != nil {
		return err
	}

	accounts := make([]*gctrpc.Account, len(initAcc.Accounts))
	for x := range initAcc.Accounts {
		subAccounts := make([]*gctrpc.AccountCurrencyInfo, len(initAcc.Accounts[x].Currencies))
		for y := range initAcc.Accounts[x].Currencies {
			subAccounts[y] = &gctrpc.AccountCurrencyInfo{
				Currency:   initAcc.Accounts[x].Currencies[y].Currency.String(),
				TotalValue: initAcc.Accounts[x].Currencies[y].Total,
				Hold:       initAcc.Accounts[x].Currencies[y].Hold,
			}
		}
		accounts[x] = &gctrpc.Account{
			Id:         initAcc.Accounts[x].ID,
			Currencies: subAccounts,
		}
	}

	err = stream.Send(&gctrpc.GetAccountInfoResponse{
		Exchange: initAcc.Exchange,
		Accounts: accounts,
	})
	if err != nil {
		return err
	}

	pipe, err := account.SubscribeToExchangeAccount(r.Exchange)
	if err != nil {
		return err
	}

	defer func() {
		pipeErr := pipe.Release()
		if pipeErr != nil {
			log.Error(log.DispatchMgr, pipeErr)
		}
	}()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errDispatchSystem
		}

		holdings, ok := data.(*account.Holdings)
		if !ok {
			return common.GetAssertError("*account.Holdings", data)
		}

		accounts := make([]*gctrpc.Account, len(holdings.Accounts))
		for x := range holdings.Accounts {
			subAccounts := make([]*gctrpc.AccountCurrencyInfo, len(holdings.Accounts[x].Currencies))
			for y := range holdings.Accounts[x].Currencies {
				subAccounts[y] = &gctrpc.AccountCurrencyInfo{
					Currency:   holdings.Accounts[x].Currencies[y].Currency.String(),
					TotalValue: holdings.Accounts[x].Currencies[y].Total,
					Hold:       holdings.Accounts[x].Currencies[y].Hold,
				}
			}
			accounts[x] = &gctrpc.Account{
				Id:         holdings.Accounts[x].ID,
				Currencies: subAccounts,
			}
		}

		if err := stream.Send(&gctrpc.GetAccountInfoResponse{
			Exchange: holdings.Exchange,
			Accounts: accounts,
		}); err != nil {
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
		return nil, fmt.Errorf("forex providers is empty")
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
		return nil, fmt.Errorf("forex rates is empty")
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

	request := &order.GetOrdersRequest{
		Pairs:     []currency.Pair{cp},
		AssetType: a,
		Type:      order.AnyType,
		Side:      order.AnySide,
	}
	if !start.IsZero() {
		request.StartTime = start
	}
	if !end.IsZero() {
		request.EndTime = end
	}

	var resp []order.Detail
	resp, err = exch.GetActiveOrders(ctx, request)
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

	pair := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

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

// SubmitOrder submits an order specified by exchange, currency pair and asset
// type
func (s *RPCServer) SubmitOrder(ctx context.Context, r *gctrpc.SubmitOrderRequest) (*gctrpc.SubmitOrderResponse, error) {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

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
func (s *RPCServer) SimulateOrder(ctx context.Context, r *gctrpc.SimulateOrderRequest) (*gctrpc.SimulateOrderResponse, error) {
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, asset.Spot, p)
	if err != nil {
		return nil, err
	}

	o, err := exch.FetchOrderbook(ctx, p, asset.Spot)
	if err != nil {
		return nil, err
	}

	var buy = true
	if !strings.EqualFold(r.Side, order.Buy.String()) &&
		!strings.EqualFold(r.Side, order.Bid.String()) {
		buy = false
	}

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
func (s *RPCServer) WhaleBomb(ctx context.Context, r *gctrpc.WhaleBombRequest) (*gctrpc.SimulateOrderResponse, error) {
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	o, err := exch.FetchOrderbook(ctx, p, a)
	if err != nil {
		return nil, err
	}

	var buy = true
	if !strings.EqualFold(r.Side, order.Buy.String()) &&
		!strings.EqualFold(r.Side, order.Bid.String()) {
		buy = false
	}

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

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

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

	var side order.Side
	side, err = order.StringToOrderSide(r.Side)
	if err != nil {
		return nil, err
	}

	err = s.OrderManager.Cancel(ctx,
		&order.Cancel{
			Exchange:      r.Exchange,
			AccountID:     r.AccountId,
			OrderID:       r.OrderId,
			Side:          side,
			WalletAddress: r.WalletAddress,
			Pair:          p,
			AssetType:     a,
		})
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("order %s cancelled", r.OrderId)}, nil
}

// CancelBatchOrders cancels an orders specified by exchange, currency pair and asset type
func (s *RPCServer) CancelBatchOrders(ctx context.Context, r *gctrpc.CancelBatchOrdersRequest) (*gctrpc.CancelBatchOrdersResponse, error) {
	pair := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

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
	request := make([]order.Cancel, len(orders))
	for x := range orders {
		orderID := orders[x]
		status[orderID] = order.Cancelled.String()
		request[x] = order.Cancel{
			AccountID:     r.AccountId,
			OrderID:       orderID,
			Side:          side,
			WalletAddress: r.WalletAddress,
			Pair:          pair,
			AssetType:     assetType,
		}
	}

	// TODO: Change to order manager
	_, err = exch.CancelBatchOrders(ctx, request)
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
		return &gctrpc.CancelAllOrdersResponse{}, err
	}

	return &gctrpc.CancelAllOrdersResponse{
		Count: resp.Count, // count of deleted orders
	}, nil
}

// ModifyOrder modifies an existing order if it exists
func (s *RPCServer) ModifyOrder(ctx context.Context, r *gctrpc.ModifyOrderRequest) (*gctrpc.ModifyOrderResponse, error) {
	assetType, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	pair := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

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
func (s *RPCServer) RemoveEvent(ctx context.Context, r *gctrpc.RemoveEventRequest) (*gctrpc.GenericResponse, error) {
	if !s.eventManager.Remove(r.Id) {
		return nil, fmt.Errorf("event %d not removed", r.Id)
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("event %d removed", r.Id)}, nil
}

// GetCryptocurrencyDepositAddresses returns a list of cryptocurrency deposit
// addresses specified by an exchange
func (s *RPCServer) GetCryptocurrencyDepositAddresses(ctx context.Context, r *gctrpc.GetCryptocurrencyDepositAddressesRequest) (*gctrpc.GetCryptocurrencyDepositAddressesResponse, error) {
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

	request := &withdraw.Request{
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
		request.OneTimePassword = codeNum
	}

	if exchCfg.API.Credentials.PIN != "" {
		pinCode, errPin := strconv.ParseInt(exchCfg.API.Credentials.PIN, 10, 64)
		if err != nil {
			return nil, errPin
		}
		request.PIN = pinCode
	}

	request.TradePassword = exchCfg.API.Credentials.TradePassword

	resp, err := s.Engine.WithdrawManager.SubmitWithdrawal(ctx, request)
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

	request := &withdraw.Request{
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
		if err != nil {
			return nil, errOTP
		}

		codeNum, errOTP := strconv.ParseInt(code, 10, 64)
		if err != nil {
			return nil, errOTP
		}
		request.OneTimePassword = codeNum
	}

	if exchCfg.API.Credentials.PIN != "" {
		pinCode, errPIN := strconv.ParseInt(exchCfg.API.Credentials.PIN, 10, 64)
		if err != nil {
			return nil, errPIN
		}
		request.PIN = pinCode
	}

	request.TradePassword = exchCfg.API.Credentials.TradePassword

	resp, err := s.Engine.WithdrawManager.SubmitWithdrawal(ctx, request)
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
				Type:        int32(v.RequestDetails.Type),
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

	if v.RequestDetails.Type == withdraw.Crypto {
		resp.Event.Request.Crypto = new(gctrpc.CryptoWithdrawalEvent)
		resp.Event.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
			Address:    v.RequestDetails.Crypto.Address,
			AddressTag: v.RequestDetails.Crypto.AddressTag,
			Fee:        v.RequestDetails.Crypto.FeeAmount,
		}
	} else if v.RequestDetails.Type == withdraw.Fiat {
		if v.RequestDetails.Fiat != (withdraw.FiatRequest{}) {
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
			return nil, fmt.Errorf("specified asset %s is not supported by exchange", a)
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

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return err
	}

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
			resp.LastUpdated = time.Now().Unix()
		} else {
			resp.Bids = make([]*gctrpc.OrderbookItem, len(base.Bids))
			for i := range base.Bids {
				resp.Bids[i] = &gctrpc.OrderbookItem{
					Amount: base.Bids[i].Amount,
					Price:  base.Bids[i].Price,
					Id:     base.Bids[i].ID}
			}
			resp.Asks = make([]*gctrpc.OrderbookItem, len(base.Asks))
			for i := range base.Asks {
				resp.Asks[i] = &gctrpc.OrderbookItem{
					Amount: base.Asks[i].Amount,
					Price:  base.Asks[i].Price,
					Id:     base.Asks[i].ID}
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
		return errExchangeNameUnset
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
			log.Error(log.DispatchMgr, pipeErr)
		}
	}()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errDispatchSystem
		}

		d, ok := data.(orderbook.Outbound)
		if !ok {
			return common.GetAssertError("orderbook.Outbound", data)
		}

		resp := &gctrpc.OrderbookResponse{}
		ob, err := d.Retrieve()
		if err != nil {
			resp.Error = err.Error()
			resp.LastUpdated = time.Now().Unix()
		} else {
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
					Id:     ob.Bids[i].ID}
			}
			resp.Asks = make([]*gctrpc.OrderbookItem, len(ob.Asks))
			for i := range ob.Asks {
				resp.Asks[i] = &gctrpc.OrderbookItem{
					Amount: ob.Asks[i].Amount,
					Price:  ob.Asks[i].Price,
					Id:     ob.Asks[i].ID}
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
		return errExchangeNameUnset
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
			log.Error(log.DispatchMgr, pipeErr)
		}
	}()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errDispatchSystem
		}

		t, ok := data.(*ticker.Price)
		if !ok {
			return common.GetAssertError("*ticker.Price", data)
		}

		err := stream.Send(&gctrpc.TickerResponse{
			Pair: &gctrpc.CurrencyPair{
				Base:      t.Pair.Base.String(),
				Quote:     t.Pair.Quote.String(),
				Delimiter: t.Pair.Delimiter},
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
		return errExchangeNameUnset
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
			log.Error(log.DispatchMgr, pipeErr)
		}
	}()

	for {
		data, ok := <-pipe.C
		if !ok {
			return errDispatchSystem
		}

		t, ok := data.(*ticker.Price)
		if !ok {
			return common.GetAssertError("*ticker.Price", data)
		}

		err := stream.Send(&gctrpc.TickerResponse{
			Pair: &gctrpc.CurrencyPair{
				Base:      t.Pair.Base.String(),
				Quote:     t.Pair.Quote.String(),
				Delimiter: t.Pair.Delimiter},
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
	pair := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

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

	gctscript.AllVMSync.Range(func(k, v interface{}) bool {
		vm, ok := v.(*gctscript.VM)
		if !ok {
			log.Errorf(log.GRPCSys, "%v", common.GetAssertError("*gctscript.VM", v))
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
		return nil, errors.New("unable to type assert gctscript.VM")
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
		return nil, errors.New("unable to type assert gctscript.VM")
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
	var fPathExits = fPath
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
		func(path string, info os.FileInfo, err error) error {
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

	base := exch.GetBase() // nolint:ifshort,nolintlint // false positive and triggers only on Windows
	if base == nil {
		return nil, errExchangeBaseNotFound
	}

	if !base.GetEnabledFeatures().AutoPairUpdates {
		return nil,
			errors.New("cannot auto pair update for exchange, a manual update is needed")
	}

	err = exch.UpdateTradablePairs(ctx, false)
	if err != nil {
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
func (s *RPCServer) WebsocketSetEnabled(_ context.Context, r *gctrpc.WebsocketSetEnabledRequest) (*gctrpc.GenericResponse, error) {
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
		err = w.Enable()
		if err != nil {
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
				Channel:  subs[i].Channel,
				Currency: subs[i].Currency.String(),
				Asset:    subs[i].Asset.String(),
				Params:   string(params),
			})
	}
	return payload, nil
}

// WebsocketSetProxy sets client websocket connection proxy
func (s *RPCServer) WebsocketSetProxy(_ context.Context, r *gctrpc.WebsocketSetProxyRequest) (*gctrpc.GenericResponse, error) {
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

	w, err := exch.GetWebsocket()
	if err != nil {
		return nil, fmt.Errorf("websocket not supported for exchange %s", r.Exchange)
	}

	err = w.SetProxyAddress(r.Proxy)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("new proxy has been set [%s] for %s websocket connection",
			r.Exchange,
			r.Proxy)}, nil
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
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("new URL has been set [%s] for %s websocket connection",
			r.Exchange,
			r.Url)}, nil
}

// GetSavedTrades returns trades from the database
func (s *RPCServer) GetSavedTrades(_ context.Context, r *gctrpc.GetSavedTradesRequest) (*gctrpc.SavedTradesResponse, error) {
	if r.End == "" || r.Start == "" || r.Exchange == "" || r.Pair == nil || r.AssetType == "" || r.Pair.String() == "" {
		return nil, errInvalidArguments
	}

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

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
	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

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
		return nil, fmt.Errorf("no candles generated from trades")
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
	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.ExchangeName)
	if err != nil {
		return nil, err
	}

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
	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.ExchangeName)
	if err != nil {
		return nil, err
	}

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
	cp := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return err
	}

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
	cp := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	a, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

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
			return fmt.Errorf("%v %w", a, errAssetTypeDisabled)
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
				Type:        int32(ret[x].RequestDetails.Type),
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

		if ret[x].RequestDetails.Type == withdraw.Crypto {
			tempEvent.Request.Crypto = new(gctrpc.CryptoWithdrawalEvent)
			tempEvent.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
				Address:    ret[x].RequestDetails.Crypto.Address,
				AddressTag: ret[x].RequestDetails.Crypto.AddressTag,
				Fee:        ret[x].RequestDetails.Crypto.FeeAmount,
			}
		} else if ret[x].RequestDetails.Type == withdraw.Fiat {
			if ret[x].RequestDetails.Fiat != (withdraw.FiatRequest{}) {
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
			Type:        int32(ret.RequestDetails.Type),
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

	if ret.RequestDetails.Type == withdraw.Crypto {
		tempEvent.Request.Crypto = new(gctrpc.CryptoWithdrawalEvent)
		tempEvent.Request.Crypto = &gctrpc.CryptoWithdrawalEvent{
			Address:    ret.RequestDetails.Crypto.Address,
			AddressTag: ret.RequestDetails.Crypto.AddressTag,
			Fee:        ret.RequestDetails.Crypto.FeeAmount,
		}
	} else if ret.RequestDetails.Type == withdraw.Fiat {
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

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	e, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}

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
						StartDate: v[i].IntervalStartDate.Format(common.SimpleTimeFormat),
						EndDate:   v[i].IntervalEndDate.Format(common.SimpleTimeFormat),
						HasData:   v[i].Status == dataHistoryStatusComplete,
						Message:   v[i].Result,
						RunDate:   v[i].Date.Format(common.SimpleTimeFormat),
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
		StartDate:                result.StartDate.Format(common.SimpleTimeFormat),
		EndDate:                  result.EndDate.Format(common.SimpleTimeFormat),
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
			StartDate:                jobs[i].StartDate.Format(common.SimpleTimeFormat),
			EndDate:                  jobs[i].EndDate.Format(common.SimpleTimeFormat),
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
			StartDate:                jobs[i].StartDate.Format(common.SimpleTimeFormat),
			EndDate:                  jobs[i].EndDate.Format(common.SimpleTimeFormat),
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
		StartDate:               job.StartDate.Format(common.SimpleTimeFormat),
		EndDate:                 job.EndDate.Format(common.SimpleTimeFormat),
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
		log.Error(log.GRPCSys, err)
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

func (s *RPCServer) buildFuturePosition(position *order.Position, getFundingPayments, includeFundingRates, includeOrders, includePredictedRate bool) *gctrpc.FuturePosition {
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
	if err := order.CheckFundingRatePrerequisites(r.GetFundingPayments, r.IncludePredictedRate, r.GetFundingPayments); err != nil {
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
	ai, err = asset.New(r.Asset)
	if err != nil {
		return nil, err
	}
	if !ai.IsFutures() {
		return nil, fmt.Errorf("%w '%v'", order.ErrNotFuturesAsset, ai)
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
		return nil, fmt.Errorf("%w GetAllManagedPositions", common.ErrNilPointer)
	}
	if err := order.CheckFundingRatePrerequisites(r.GetFundingPayments, r.IncludePredictedRate, r.GetFundingPayments); err != nil {
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

// GetFuturesPositions returns pnl positions for an exchange asset pair
func (s *RPCServer) GetFuturesPositions(ctx context.Context, r *gctrpc.GetFuturesPositionsRequest) (*gctrpc.GetFuturesPositionsResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetFuturesPositions", common.ErrNilPointer)
	}
	if err := order.CheckFundingRatePrerequisites(r.GetFundingPayments, r.IncludePredictedRate, r.GetFundingPayments); err != nil {
		return nil, err
	}
	exch, err := s.GetExchangeByName(r.Exchange)
	if err != nil {
		return nil, err
	}
	cp, err := currency.NewPairFromStrings(r.Pair.Base, r.Pair.Quote)
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
	if !ai.IsFutures() {
		return nil, fmt.Errorf("%s %w", ai, order.ErrNotFuturesAsset)
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

	b := exch.GetBase()
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	var subAccount string
	if creds.SubAccount != "" {
		subAccount = "for subaccount: " + creds.SubAccount
	}
	positionDetails, err := exch.GetFuturesPositions(ctx, &order.PositionsRequest{
		Asset:     ai,
		Pairs:     currency.Pairs{cp},
		StartDate: start,
	})
	if err != nil {
		return nil, fmt.Errorf("%w %v", err, subAccount)
	}
	if len(positionDetails) != 1 {
		return nil, errUnexpectedResponseSize
	}
	if r.Overwrite {
		err = s.OrderManager.ClearFuturesTracking(r.Exchange, ai, cp)
		if err != nil {
			return nil, fmt.Errorf("cannot overwrite %w %v", err, subAccount)
		}
	}
	for i := range positionDetails[0].Orders {
		err = s.OrderManager.orderStore.futuresPositionController.TrackNewOrder(&positionDetails[0].Orders[i])
		if err != nil {
			if !errors.Is(err, order.ErrPositionClosed) {
				return nil, err
			}
		}
	}
	pos, err := s.OrderManager.GetFuturesPositionsForExchange(r.Exchange, ai, cp)
	if err != nil {
		return nil, fmt.Errorf("cannot GetFuturesPositionsForExchange %w %v", err, subAccount)
	}

	response := &gctrpc.GetFuturesPositionsResponse{
		SubAccount: creds.SubAccount,
	}
	var totalRealisedPNL, totalUnrealisedPNL decimal.Decimal
	for i := range pos {
		if r.Status != "" && pos[i].Status.String() != strings.ToUpper(r.Status) {
			continue
		}
		if r.PositionLimit > 0 && len(response.Positions) >= int(r.PositionLimit) {
			break
		}
		if pos[i].Status == order.Open {
			var tick *ticker.Price
			tick, err = exch.FetchTicker(ctx, pos[i].Pair, pos[i].Asset)
			if err != nil {
				return nil, fmt.Errorf("%w when fetching ticker data for %v %v %v %v", err, pos[i].Exchange, pos[i].Asset, pos[i].Pair, subAccount)
			}
			pos[i].UnrealisedPNL, err = s.OrderManager.UpdateOpenPositionUnrealisedPNL(pos[i].Exchange, pos[i].Asset, pos[i].Pair, tick.Last, tick.LastUpdated)
			if err != nil {
				return nil, fmt.Errorf("%w when updating unrealised PNL for %v %v %v %v", err, pos[i].Exchange, pos[i].Asset, pos[i].Pair, subAccount)
			}
			pos[i].LatestPrice = decimal.NewFromFloat(tick.Last)
		}
		response.TotalOrders += int64(len(pos[i].Orders))
		details := &gctrpc.FuturePosition{
			Exchange: pos[i].Exchange,
			Asset:    pos[i].Asset.String(),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: pos[i].Pair.Delimiter,
				Base:      pos[i].Pair.Base.String(),
				Quote:     pos[i].Pair.Quote.String(),
			},
			Status:           pos[i].Status.String(),
			OpeningDate:      pos[i].OpeningDate.Format(common.SimpleTimeFormatWithTimezone),
			OpeningDirection: pos[i].OpeningDirection.String(),
			OpeningPrice:     pos[i].OpeningPrice.String(),
			OpeningSize:      pos[i].OpeningSize.String(),
			CurrentDirection: pos[i].LatestDirection.String(),
			CurrentPrice:     pos[i].LatestPrice.String(),
			CurrentSize:      pos[i].LatestSize.String(),
			UnrealisedPnl:    pos[i].UnrealisedPNL.String(),
			RealisedPnl:      pos[i].RealisedPNL.String(),
			OrderCount:       int64(len(pos[i].Orders)),
		}
		if !pos[i].UnrealisedPNL.IsZero() {
			details.UnrealisedPnl = pos[i].UnrealisedPNL.String()
		}
		if !pos[i].RealisedPNL.IsZero() {
			details.RealisedPnl = pos[i].RealisedPNL.String()
		}
		if pos[i].LatestDirection != order.UnknownSide {
			details.CurrentDirection = pos[i].LatestDirection.String()
		}
		if len(pos[i].PNLHistory) > 0 {
			details.OpeningDate = pos[i].PNLHistory[0].Time.Format(common.SimpleTimeFormatWithTimezone)
			if pos[i].Status == order.Closed {
				details.ClosingDate = pos[i].PNLHistory[len(pos[i].PNLHistory)-1].Time.Format(common.SimpleTimeFormatWithTimezone)
			}
		}
		totalRealisedPNL = totalRealisedPNL.Add(pos[i].RealisedPNL)
		totalUnrealisedPNL = totalUnrealisedPNL.Add(pos[i].UnrealisedPNL)
		if r.GetPositionStats {
			var stats *order.PositionSummary
			stats, err = exch.GetPositionSummary(ctx, &order.PositionSummaryRequest{
				Asset: pos[i].Asset,
				Pair:  pos[i].Pair,
			})
			if err != nil {
				return nil, fmt.Errorf("cannot GetPositionSummary %w %v", err, subAccount)
			}
			details.PositionStats = &gctrpc.FuturesPositionStats{
				MaintenanceMarginRequirement: stats.MaintenanceMarginRequirement.String(),
				InitialMarginRequirement:     stats.InitialMarginRequirement.String(),
				CollateralUsed:               stats.CollateralUsed.String(),
				MarkPrice:                    stats.MarkPrice.String(),
				CurrentSize:                  stats.CurrentSize.String(),
				BreakEvenPrice:               stats.BreakEvenPrice.String(),
				AverageOpenPrice:             stats.AverageOpenPrice.String(),
				RecentPnl:                    stats.RecentPNL.String(),
				MarginFraction:               stats.MarginFraction.String(),
				FreeCollateral:               stats.FreeCollateral.String(),
				TotalCollateral:              stats.TotalCollateral.String(),
			}
			if !stats.EstimatedLiquidationPrice.IsZero() {
				details.PositionStats.EstimatedLiquidationPrice = stats.EstimatedLiquidationPrice.String()
			}
		}
		if r.GetFundingPayments {
			var endDate = time.Now()
			if pos[i].Status == order.Closed {
				endDate = pos[i].Orders[len(pos[i].Orders)-1].Date
			}
			var fundingDetails []order.FundingRates
			fundingDetails, err = exch.GetFundingRates(ctx, &order.FundingRatesRequest{
				Asset:                pos[i].Asset,
				Pairs:                currency.Pairs{pos[i].Pair},
				StartDate:            pos[i].Orders[0].Date,
				EndDate:              endDate,
				IncludePayments:      r.GetFundingPayments,
				IncludePredictedRate: r.IncludePredictedRate,
			})
			if err != nil {
				return nil, err
			}
			switch {
			case len(fundingDetails) == 0:
			case len(fundingDetails) == 1:
				var funding []*gctrpc.FundingRate
				if r.IncludeFullFundingRates {
					for j := range fundingDetails[0].FundingRates {
						funding = append(funding, &gctrpc.FundingRate{
							Date:    fundingDetails[0].FundingRates[j].Time.Format(common.SimpleTimeFormatWithTimezone),
							Rate:    fundingDetails[0].FundingRates[j].Rate.String(),
							Payment: fundingDetails[0].FundingRates[j].Payment.String(),
						})
					}
				}
				fundingRates := &gctrpc.FundingData{
					Rates:      funding,
					PaymentSum: fundingDetails[0].PaymentSum.String(),
				}
				if r.IncludeFullFundingRates {
					fundingRates.LatestRate = funding[len(fundingRates.Rates)-1]
				}
				if r.IncludePredictedRate && !fundingDetails[0].PredictedUpcomingRate.Time.IsZero() {
					fundingRates.UpcomingRate = &gctrpc.FundingRate{
						Date: fundingDetails[0].PredictedUpcomingRate.Time.Format(common.SimpleTimeFormatWithTimezone),
						Rate: fundingDetails[0].PredictedUpcomingRate.Rate.String(),
					}
				}
				details.FundingData = fundingRates
				err = s.OrderManager.orderStore.futuresPositionController.TrackFundingDetails(&fundingDetails[0])
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("%w expected 1 set of funding rates, got %d %v", errUnexpectedResponseSize, len(fundingDetails), subAccount)
			}
		}
		if !r.IncludeFullOrderData {
			response.Positions = append(response.Positions, details)
			continue
		}
		for j := range pos[i].Orders {
			var trades []*gctrpc.TradeHistory
			for k := range pos[i].Orders[j].Trades {
				trades = append(trades, &gctrpc.TradeHistory{
					CreationTime: pos[i].Orders[j].Trades[k].Timestamp.Unix(),
					Id:           pos[i].Orders[j].Trades[k].TID,
					Price:        pos[i].Orders[j].Trades[k].Price,
					Amount:       pos[i].Orders[j].Trades[k].Amount,
					Exchange:     pos[i].Orders[j].Trades[k].Exchange,
					AssetType:    pos[i].Asset.String(),
					OrderSide:    pos[i].Orders[j].Trades[k].Side.String(),
					Fee:          pos[i].Orders[j].Trades[k].Fee,
					Total:        pos[i].Orders[j].Trades[k].Total,
				})
			}
			od := &gctrpc.OrderDetails{
				Exchange:      pos[i].Orders[j].Exchange,
				Id:            pos[i].Orders[j].OrderID,
				ClientOrderId: pos[i].Orders[j].ClientOrderID,
				BaseCurrency:  pos[i].Orders[j].Pair.Base.String(),
				QuoteCurrency: pos[i].Orders[j].Pair.Quote.String(),
				AssetType:     pos[i].Orders[j].AssetType.String(),
				OrderSide:     pos[i].Orders[j].Side.String(),
				OrderType:     pos[i].Orders[j].Type.String(),
				CreationTime:  pos[i].Orders[j].Date.Format(common.SimpleTimeFormatWithTimezone),
				Status:        pos[i].Orders[j].Status.String(),
				Price:         pos[i].Orders[j].Price,
				Amount:        pos[i].Orders[j].Amount,
				Fee:           pos[i].Orders[j].Fee,
				Cost:          pos[i].Orders[j].Cost,
				Trades:        trades,
			}
			if pos[i].Orders[j].LastUpdated.After(pos[i].Orders[j].Date) {
				od.UpdateTime = pos[i].Orders[j].LastUpdated.Format(common.SimpleTimeFormatWithTimezone)
			}
			details.Orders = append(details.Orders, od)
		}
		response.Positions = append(response.Positions, details)
	}

	if !totalUnrealisedPNL.IsZero() {
		response.TotalUnrealisedPnl = totalUnrealisedPNL.String()
	}
	if !totalRealisedPNL.IsZero() {
		response.TotalRealisedPnl = totalRealisedPNL.String()
	}
	if !totalUnrealisedPNL.IsZero() && !totalRealisedPNL.IsZero() {
		response.TotalPnl = totalRealisedPNL.Add(totalUnrealisedPNL).String()
	}
	return response, nil
}

// GetFundingRates returns the funding rates for a slice of pairs of an exchange, asset
func (s *RPCServer) GetFundingRates(ctx context.Context, r *gctrpc.GetFundingRatesRequest) (*gctrpc.GetFundingRatesResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetFundingRateRequest", common.ErrNilPointer)
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
		return nil, fmt.Errorf("%s %w", a, order.ErrNotFuturesAsset)
	}
	start := time.Now().AddDate(-1, 0, 0)
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
	pairs, err := currency.NewPairsFromStrings(r.Pairs)
	if err != nil {
		return nil, err
	}
	for i := range pairs {
		err = checkParams(r.Exchange, exch, a, pairs[i])
		if err != nil {
			return nil, err
		}
	}
	funding, err := exch.GetFundingRates(ctx, &order.FundingRatesRequest{
		Asset:                a,
		Pairs:                pairs,
		StartDate:            start,
		EndDate:              end,
		IncludePayments:      r.IncludePayments,
		IncludePredictedRate: r.IncludePredicted,
	})
	if err != nil {
		return nil, err
	}
	var response gctrpc.GetFundingRatesResponse
	responses := make([]*gctrpc.FundingData, len(funding))
	for i := range funding {
		fundingData := &gctrpc.FundingData{
			Exchange: r.Exchange,
			Asset:    r.Asset,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: funding[i].Pair.Delimiter,
				Base:      funding[i].Pair.Base.String(),
				Quote:     funding[i].Pair.Quote.String(),
			},
			StartDate: start.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:   end.Format(common.SimpleTimeFormatWithTimezone),
			LatestRate: &gctrpc.FundingRate{
				Date: funding[i].LatestRate.Time.Format(common.SimpleTimeFormatWithTimezone),
				Rate: funding[i].LatestRate.Rate.String(),
			},
		}
		var rates []*gctrpc.FundingRate
		for j := range funding[i].FundingRates {
			rate := &gctrpc.FundingRate{
				Rate: funding[i].FundingRates[j].Rate.String(),
				Date: funding[i].FundingRates[j].Time.Format(common.SimpleTimeFormatWithTimezone),
			}
			if r.IncludePayments {
				rate.Payment = funding[i].FundingRates[j].Payment.String()
			}
			rates = append(rates, rate)
		}
		if r.IncludePayments {
			fundingData.PaymentSum = funding[i].PaymentSum.String()
		}
		fundingData.Rates = rates
		if r.IncludePredicted {
			fundingData.UpcomingRate = &gctrpc.FundingRate{
				Date: funding[i].PredictedUpcomingRate.Time.Format(common.SimpleTimeFormatWithTimezone),
				Rate: funding[i].PredictedUpcomingRate.Rate.String(),
			}
		}
		responses[i] = fundingData
	}
	response.FundingPayments = responses
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

	a, err := asset.New(r.Asset)
	if err != nil {
		return nil, err
	}

	err = checkParams(r.Exchange, exch, a, currency.EMPTYPAIR)
	if err != nil {
		return nil, err
	}
	if !a.IsFutures() {
		return nil, fmt.Errorf("%s %w", a, order.ErrNotFuturesAsset)
	}
	ai, err := exch.FetchAccountInfo(ctx, a)
	if err != nil {
		return nil, err
	}
	creds, err := exch.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	subAccounts := make([]string, len(ai.Accounts))
	var acc *account.SubAccount
	for i := range ai.Accounts {
		subAccounts[i] = ai.Accounts[i].ID
		if ai.Accounts[i].ID == "main" && creds.SubAccount == "" {
			acc = &ai.Accounts[i]
			break
		}
		if strings.EqualFold(creds.SubAccount, ai.Accounts[i].ID) {
			acc = &ai.Accounts[i]
			break
		}
	}
	if acc == nil {
		return nil, fmt.Errorf("%w for %s %s and stored credentials - available subaccounts: %s",
			errNoAccountInformation,
			exch.GetName(),
			creds.SubAccount,
			strings.Join(subAccounts, ","))
	}
	var spotPairs currency.Pairs
	if r.CalculateOffline {
		spotPairs, err = exch.GetAvailablePairs(asset.Spot)
		if err != nil {
			return nil, fmt.Errorf("GetCollateral offline calculation error via GetAvailablePairs %s %s", exch.GetName(), err)
		}
	}

	calculators := make([]order.CollateralCalculator, 0, len(acc.Currencies))
	for i := range acc.Currencies {
		total := decimal.NewFromFloat(acc.Currencies[i].Total)
		free := decimal.NewFromFloat(acc.Currencies[i].AvailableWithoutBorrow)
		cal := order.CollateralCalculator{
			CalculateOffline:   r.CalculateOffline,
			CollateralCurrency: acc.Currencies[i].Currency,
			Asset:              a,
			FreeCollateral:     free,
			LockedCollateral:   total.Sub(free),
		}
		if r.CalculateOffline &&
			!acc.Currencies[i].Currency.Equal(currency.USD) {
			var tick *ticker.Price
			tickerCurr := currency.NewPair(acc.Currencies[i].Currency, currency.USD)
			if !spotPairs.Contains(tickerCurr, true) {
				// cannot price currency to calculate collateral
				continue
			}
			tick, err = exch.FetchTicker(ctx, tickerCurr, asset.Spot)
			if err != nil {
				log.Errorf(log.GRPCSys, fmt.Sprintf("GetCollateral offline calculation error via FetchTicker %s %s", exch.GetName(), err))
				continue
			}
			if tick.Last == 0 {
				continue
			}
			cal.USDPrice = decimal.NewFromFloat(tick.Last)
		}
		calculators = append(calculators, cal)
	}

	calc := &order.TotalCollateralCalculator{
		CollateralAssets: calculators,
		CalculateOffline: r.CalculateOffline,
		FetchPositions:   true,
	}

	collateral, err := exch.CalculateTotalCollateral(ctx, calc)
	if err != nil {
		return nil, err
	}

	var collateralDisplayCurrency = " " + collateral.CollateralCurrency.String()
	result := &gctrpc.GetCollateralResponse{
		SubAccount:          creds.SubAccount,
		CollateralCurrency:  collateral.CollateralCurrency.String(),
		AvailableCollateral: collateral.AvailableCollateral.String() + collateralDisplayCurrency,
		UsedCollateral:      collateral.UsedCollateral.String() + collateralDisplayCurrency,
	}
	if !collateral.CollateralContributedByPositiveSpotBalances.IsZero() {
		result.CollateralContributedByPositiveSpotBalances = collateral.CollateralContributedByPositiveSpotBalances.String() + collateralDisplayCurrency
	}
	if !collateral.TotalValueOfPositiveSpotBalances.IsZero() {
		result.TotalValueOfPositiveSpotBalances = collateral.TotalValueOfPositiveSpotBalances.String() + collateralDisplayCurrency
	}
	if !collateral.AvailableMaintenanceCollateral.IsZero() {
		result.MaintenanceCollateral = collateral.AvailableMaintenanceCollateral.String() + collateralDisplayCurrency
	}
	if !collateral.UnrealisedPNL.IsZero() {
		result.UnrealisedPnl = collateral.UnrealisedPNL.String()
	}
	if collateral.UsedBreakdown != nil {
		result.UsedBreakdown = &gctrpc.CollateralUsedBreakdown{}
		if !collateral.UsedBreakdown.LockedInStakes.IsZero() {
			result.UsedBreakdown.LockedInStakes = collateral.UsedBreakdown.LockedInStakes.String() + collateralDisplayCurrency
		}
		if !collateral.UsedBreakdown.LockedInNFTBids.IsZero() {
			result.UsedBreakdown.LockedInNftBids = collateral.UsedBreakdown.LockedInNFTBids.String() + collateralDisplayCurrency
		}
		if !collateral.UsedBreakdown.LockedInFeeVoucher.IsZero() {
			result.UsedBreakdown.LockedInFeeVoucher = collateral.UsedBreakdown.LockedInFeeVoucher.String() + collateralDisplayCurrency
		}
		if !collateral.UsedBreakdown.LockedInSpotMarginFundingOffers.IsZero() {
			result.UsedBreakdown.LockedInSpotMarginFundingOffers = collateral.UsedBreakdown.LockedInSpotMarginFundingOffers.String() + collateralDisplayCurrency
		}
		if !collateral.UsedBreakdown.LockedInSpotOrders.IsZero() {
			result.UsedBreakdown.LockedInSpotOrders = collateral.UsedBreakdown.LockedInSpotOrders.String() + collateralDisplayCurrency
		}
		if !collateral.UsedBreakdown.LockedAsCollateral.IsZero() {
			result.UsedBreakdown.LockedAsCollateral = collateral.UsedBreakdown.LockedAsCollateral.String() + collateralDisplayCurrency
		}
		if !collateral.UsedBreakdown.UsedInPositions.IsZero() {
			result.UsedBreakdown.UsedInFutures = collateral.UsedBreakdown.UsedInPositions.String() + collateralDisplayCurrency
		}
		if !collateral.UsedBreakdown.UsedInSpotMarginBorrows.IsZero() {
			result.UsedBreakdown.UsedInSpotMargin = collateral.UsedBreakdown.UsedInSpotMarginBorrows.String() + collateralDisplayCurrency
		}
	}
	if r.IncludeBreakdown {
		for i := range collateral.BreakdownOfPositions {
			result.PositionBreakdown = append(result.PositionBreakdown, &gctrpc.CollateralByPosition{
				Currency:            collateral.BreakdownOfPositions[i].PositionCurrency.String(),
				Size:                collateral.BreakdownOfPositions[i].Size.String(),
				OpenOrderSize:       collateral.BreakdownOfPositions[i].OpenOrderSize.String(),
				PositionSize:        collateral.BreakdownOfPositions[i].PositionSize.String(),
				MarkPrice:           collateral.BreakdownOfPositions[i].MarkPrice.String() + collateralDisplayCurrency,
				RequiredMargin:      collateral.BreakdownOfPositions[i].RequiredMargin.String(),
				TotalCollateralUsed: collateral.BreakdownOfPositions[i].CollateralUsed.String() + collateralDisplayCurrency,
			})
		}
		for i := range collateral.BreakdownByCurrency {
			if collateral.BreakdownByCurrency[i].TotalFunds.IsZero() && !r.IncludeZeroValues {
				continue
			}
			var originalDisplayCurrency = " " + collateral.BreakdownByCurrency[i].Currency.String()
			cb := &gctrpc.CollateralForCurrency{
				Currency:                    collateral.BreakdownByCurrency[i].Currency.String(),
				ExcludedFromCollateral:      collateral.BreakdownByCurrency[i].SkipContribution,
				TotalFunds:                  collateral.BreakdownByCurrency[i].TotalFunds.String() + originalDisplayCurrency,
				AvailableForUseAsCollateral: collateral.BreakdownByCurrency[i].AvailableForUseAsCollateral.String() + originalDisplayCurrency,
				ApproxFairMarketValue:       collateral.BreakdownByCurrency[i].FairMarketValue.String() + collateralDisplayCurrency,
				Weighting:                   collateral.BreakdownByCurrency[i].Weighting.String(),
				CollateralContribution:      collateral.BreakdownByCurrency[i].CollateralContribution.String() + collateralDisplayCurrency,
				ScaledToCurrency:            collateral.BreakdownByCurrency[i].ScaledCurrency.String(),
			}
			if !collateral.BreakdownByCurrency[i].AdditionalCollateralUsed.IsZero() {
				cb.AdditionalCollateralUsed = collateral.BreakdownByCurrency[i].AdditionalCollateralUsed.String() + collateralDisplayCurrency
			}

			if !collateral.BreakdownByCurrency[i].ScaledUsed.IsZero() {
				cb.FundsInUse = collateral.BreakdownByCurrency[i].ScaledUsed.String() + collateralDisplayCurrency
			}
			if !collateral.BreakdownByCurrency[i].UnrealisedPNL.IsZero() {
				cb.UnrealisedPnl = collateral.BreakdownByCurrency[i].UnrealisedPNL.String() + collateralDisplayCurrency
			}
			if collateral.BreakdownByCurrency[i].ScaledUsedBreakdown != nil {
				breakDownDisplayCurrency := collateralDisplayCurrency
				if collateral.BreakdownByCurrency[i].Weighting.IsZero() && collateral.BreakdownByCurrency[i].FairMarketValue.IsZero() {
					// cannot determine value, show in like currency instead
					breakDownDisplayCurrency = originalDisplayCurrency
				}
				cb.UsedBreakdown = &gctrpc.CollateralUsedBreakdown{}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInStakes.IsZero() {
					cb.UsedBreakdown.LockedInStakes = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInStakes.String() + breakDownDisplayCurrency
				}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInNFTBids.IsZero() {
					cb.UsedBreakdown.LockedInNftBids = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInNFTBids.String() + breakDownDisplayCurrency
				}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInFeeVoucher.IsZero() {
					cb.UsedBreakdown.LockedInFeeVoucher = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInFeeVoucher.String() + breakDownDisplayCurrency
				}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotMarginFundingOffers.IsZero() {
					cb.UsedBreakdown.LockedInSpotMarginFundingOffers = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotMarginFundingOffers.String() + breakDownDisplayCurrency
				}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotOrders.IsZero() {
					cb.UsedBreakdown.LockedInSpotOrders = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedInSpotOrders.String() + breakDownDisplayCurrency
				}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedAsCollateral.IsZero() {
					cb.UsedBreakdown.LockedAsCollateral = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.LockedAsCollateral.String() + breakDownDisplayCurrency
				}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInPositions.IsZero() {
					cb.UsedBreakdown.UsedInFutures = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInPositions.String() + breakDownDisplayCurrency
				}
				if !collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInSpotMarginBorrows.IsZero() {
					cb.UsedBreakdown.UsedInSpotMargin = collateral.BreakdownByCurrency[i].ScaledUsedBreakdown.UsedInSpotMarginBorrows.String() + breakDownDisplayCurrency
				}
			}
			if collateral.BreakdownByCurrency[i].Error != nil {
				cb.Error = collateral.BreakdownByCurrency[i].Error.Error()
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
			indicators.MaType(r.MovingAverageType))
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
		return nil, fmt.Errorf("%w '%s'", errInvalidStrategy, r.AlgorithmType)
	}

	return &gctrpc.GetTechnicalAnalysisResponse{Signals: signals}, nil
}

// GetMarginRatesHistory returns the margin lending or borrow rates for an exchange, asset, currency along with many customisable options
func (s *RPCServer) GetMarginRatesHistory(ctx context.Context, r *gctrpc.GetMarginRatesHistoryRequest) (*gctrpc.GetMarginRatesHistoryResponse, error) {
	if r == nil {
		return nil, fmt.Errorf("%w GetLendingRatesRequest", common.ErrNilPointer)
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
func (s *RPCServer) GetOrderbookMovement(ctx context.Context, r *gctrpc.GetOrderbookMovementRequest) (*gctrpc.GetOrderbookMovementResponse, error) {
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
func (s *RPCServer) GetOrderbookAmountByNominal(ctx context.Context, r *gctrpc.GetOrderbookAmountByNominalRequest) (*gctrpc.GetOrderbookAmountByNominalResponse, error) {
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
func (s *RPCServer) GetOrderbookAmountByImpact(ctx context.Context, r *gctrpc.GetOrderbookAmountByImpactRequest) (*gctrpc.GetOrderbookAmountByImpactResponse, error) {
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
