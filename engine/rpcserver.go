package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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
	errExchangeNotLoaded    = errors.New("exchange is not loaded/doesn't exist")
	errExchangeBaseNotFound = errors.New("cannot get exchange base")
	errInvalidArguments     = errors.New("invalid arguments received")
	errExchangeNameUnset    = errors.New("exchange name unset")
	errCurrencyPairUnset    = errors.New("currency pair unset")
	errInvalidTimes         = errors.New("invalid start and end times")
	errAssetTypeDisabled    = errors.New("asset type is disabled")
	errAssetTypeUnset       = errors.New("asset type unset")
	errDispatchSystem       = errors.New("dispatch system offline")
	errCurrencyNotEnabled   = errors.New("currency not enabled")
	errCurrencyPairInvalid  = errors.New("currency provided is not found in the available pairs list")
	errNoTrades             = errors.New("no trades returned from supplied params")
	errNilRequestData       = errors.New("nil request data received, cannot continue")
)

// RPCServer struct
type RPCServer struct {
	gctrpc.UnimplementedGoCryptoTraderServer
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

	username := strings.Split(string(decoded), ":")[0]
	password := strings.Split(string(decoded), ":")[1]

	if username != s.Config.RemoteControl.Username || password != s.Config.RemoteControl.Password {
		return ctx, fmt.Errorf("username/password mismatch")
	}

	return ctx, nil
}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer(engine *Engine) {
	targetDir := utils.GetTLSDir(engine.Settings.DataDir)
	err := checkCerts(targetDir)
	if err != nil {
		log.Errorf(log.GRPCSys, "gRPC checkCerts failed. err: %s\n", err)
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
	gctrpc.RegisterGoCryptoTraderServer(server, &s)

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
	err = gctrpc.RegisterGoCryptoTraderHandlerFromEndpoint(context.Background(),
		mux, s.Config.RemoteControl.GRPC.ListenAddress, opts)
	if err != nil {
		log.Errorf(log.GRPCSys, "Failed to register gRPC proxy. Err: %s\n", err)
		return
	}

	go func() {
		if err := http.ListenAndServe(s.Config.RemoteControl.GRPC.GRPCProxyListenAddress, mux); err != nil {
			log.Errorf(log.GRPCSys, "gRPC proxy failed to server: %s\n", err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC proxy server started!")
}

// GetInfo returns info about the current GoCryptoTrader session
func (s *RPCServer) GetInfo(_ context.Context, _ *gctrpc.GetInfoRequest) (*gctrpc.GetInfoResponse, error) {
	d := time.Since(s.uptime)
	resp := gctrpc.GetInfoResponse{
		Uptime:               d.String(),
		EnabledExchanges:     int64(s.Config.CountEnabledExchanges()),
		AvailableExchanges:   int64(len(s.Config.Exchanges)),
		DefaultFiatCurrency:  s.Config.Currency.FiatDisplayCurrency.String(),
		DefaultForexProvider: s.Config.GetPrimaryForexProvider(),
		SubsystemStatus:      s.GetSubsystemsStatus(),
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
func (s *RPCServer) GetSubsystems(_ context.Context, _ *gctrpc.GetSubsystemsRequest) (*gctrpc.GetSusbsytemsResponse, error) {
	return &gctrpc.GetSusbsytemsResponse{SubsystemsStatus: s.GetSubsystemsStatus()}, nil
}

// EnableSubsystem enables a engine subsytem
func (s *RPCServer) EnableSubsystem(_ context.Context, r *gctrpc.GenericSubsystemRequest) (*gctrpc.GenericResponse, error) {
	err := s.SetSubsystem(r.Subsystem, true)
	if err != nil {
		return nil, err
	}
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("subsystem %s enabled", r.Subsystem)}, nil
}

// DisableSubsystem disables a engine subsytem
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
func (s *RPCServer) GetExchangeOTPCode(_ context.Context, r *gctrpc.GenericExchangeNameRequest) (*gctrpc.GetExchangeOTPReponse, error) {
	result, err := s.GetExchangeOTPByName(r.Exchange)
	return &gctrpc.GetExchangeOTPReponse{OtpCode: result}, err
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
		ps, err := exchCfg.CurrencyPairs.Get(assets[i])
		if err != nil {
			return nil, err
		}

		resp.SupportedAssets[assets[i].String()] = &gctrpc.PairsSupported{
			EnabledPairs:   ps.Enabled.Join(),
			AvailablePairs: ps.Available.Join(),
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

	e := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, e, a, currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	})
	if err != nil {
		return nil, err
	}

	t, err := s.GetSpecificTicker(currency.Pair{
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
func (s *RPCServer) GetTickers(_ context.Context, _ *gctrpc.GetTickersRequest) (*gctrpc.GetTickersResponse, error) {
	activeTickers := s.GetAllActiveTickers()
	var tickers []*gctrpc.Tickers

	for x := range activeTickers {
		t := &gctrpc.Tickers{
			Exchange: activeTickers[x].ExchangeName,
		}
		for y := range activeTickers[x].ExchangeValues {
			val := activeTickers[x].ExchangeValues[y]
			t.Tickers = append(t.Tickers, &gctrpc.TickerResponse{
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
			})
		}
		tickers = append(tickers, t)
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

	ob, err := s.GetSpecificOrderbook(currency.Pair{
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
func (s *RPCServer) GetOrderbooks(_ context.Context, _ *gctrpc.GetOrderbooksRequest) (*gctrpc.GetOrderbooksResponse, error) {
	exchanges := s.ExchangeManager.GetExchanges()
	var obResponse []*gctrpc.Orderbooks
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
				resp, err := exchanges[x].FetchOrderbook(currencies[z], assets[y])
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
				}
				for i := range resp.Bids {
					ob.Bids = append(ob.Bids, &gctrpc.OrderbookItem{
						Amount: resp.Bids[i].Amount,
						Price:  resp.Bids[i].Price,
					})
				}

				for i := range resp.Asks {
					ob.Asks = append(ob.Asks, &gctrpc.OrderbookItem{
						Amount: resp.Asks[i].Amount,
						Price:  resp.Asks[i].Price,
					})
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
func (s *RPCServer) GetAccountInfo(_ context.Context, r *gctrpc.GetAccountInfoRequest) (*gctrpc.GetAccountInfoResponse, error) {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch := s.GetExchangeByName(r.Exchange)

	err = checkParams(r.Exchange, exch, assetType, currency.Pair{})
	if err != nil {
		return nil, err
	}

	resp, err := exch.FetchAccountInfo(assetType)
	if err != nil {
		return nil, err
	}

	return createAccountInfoRequest(resp)
}

// UpdateAccountInfo forces an update of the account info
func (s *RPCServer) UpdateAccountInfo(_ context.Context, r *gctrpc.GetAccountInfoRequest) (*gctrpc.GetAccountInfoResponse, error) {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, assetType, currency.Pair{})
	if err != nil {
		return nil, err
	}

	resp, err := exch.UpdateAccountInfo(assetType)
	if err != nil {
		return nil, err
	}

	return createAccountInfoRequest(resp)
}

func createAccountInfoRequest(h account.Holdings) (*gctrpc.GetAccountInfoResponse, error) {
	var accounts []*gctrpc.Account
	for x := range h.Accounts {
		var a gctrpc.Account
		a.Id = h.Accounts[x].ID
		for _, y := range h.Accounts[x].Currencies {
			a.Currencies = append(a.Currencies, &gctrpc.AccountCurrencyInfo{
				Currency:   y.CurrencyName.String(),
				Hold:       y.Hold,
				TotalValue: y.TotalValue,
			})
		}
		accounts = append(accounts, &a)
	}

	return &gctrpc.GetAccountInfoResponse{Exchange: h.Exchange, Accounts: accounts}, nil
}

// GetAccountInfoStream streams an account balance for a specific exchange
func (s *RPCServer) GetAccountInfoStream(r *gctrpc.GetAccountInfoRequest, stream gctrpc.GoCryptoTrader_GetAccountInfoStreamServer) error {
	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return err
	}

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, assetType, currency.Pair{})
	if err != nil {
		return err
	}

	initAcc, err := exch.FetchAccountInfo(assetType)
	if err != nil {
		return err
	}

	var accounts []*gctrpc.Account
	for x := range initAcc.Accounts {
		var subAccounts []*gctrpc.AccountCurrencyInfo
		for y := range initAcc.Accounts[x].Currencies {
			subAccounts = append(subAccounts, &gctrpc.AccountCurrencyInfo{
				Currency:   initAcc.Accounts[x].Currencies[y].CurrencyName.String(),
				TotalValue: initAcc.Accounts[x].Currencies[y].TotalValue,
				Hold:       initAcc.Accounts[x].Currencies[y].Hold,
			})
		}
		accounts = append(accounts, &gctrpc.Account{
			Id:         initAcc.Accounts[x].ID,
			Currencies: subAccounts,
		})
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

		acc := (*data.(*interface{})).(account.Holdings)

		var accounts []*gctrpc.Account
		for x := range acc.Accounts {
			var subAccounts []*gctrpc.AccountCurrencyInfo
			for y := range acc.Accounts[x].Currencies {
				subAccounts = append(subAccounts, &gctrpc.AccountCurrencyInfo{
					Currency:   acc.Accounts[x].Currencies[y].CurrencyName.String(),
					TotalValue: acc.Accounts[x].Currencies[y].TotalValue,
					Hold:       acc.Accounts[x].Currencies[y].Hold,
				})
			}
			accounts = append(accounts, &gctrpc.Account{
				Id:         acc.Accounts[x].ID,
				Currencies: subAccounts,
			})
		}

		err := stream.Send(&gctrpc.GetAccountInfoResponse{
			Exchange: acc.Exchange,
			Accounts: accounts,
		})
		if err != nil {
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
	var addrs []*gctrpc.PortfolioAddress
	botAddrs := s.portfolioManager.GetAddresses()
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
func (s *RPCServer) GetForexRates(_ context.Context, _ *gctrpc.GetForexRatesRequest) (*gctrpc.GetForexRatesResponse, error) {
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
// asset type between optional dates
func (s *RPCServer) GetOrders(_ context.Context, r *gctrpc.GetOrdersRequest) (*gctrpc.GetOrdersResponse, error) {
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
	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, cp)
	if err != nil {
		return nil, err
	}

	var start, end time.Time
	if r.StartDate != "" {
		start, err = time.Parse(common.SimpleTimeFormat, r.StartDate)
		if err != nil {
			return nil, err
		}
	}
	if r.EndDate != "" {
		end, err = time.Parse(common.SimpleTimeFormat, r.EndDate)
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
	}
	if !start.IsZero() {
		request.StartTime = start
	}
	if !end.IsZero() {
		request.EndTime = end
	}

	var resp []order.Detail
	resp, err = exch.GetActiveOrders(request)
	if err != nil {
		return nil, err
	}

	var orders []*gctrpc.OrderDetails
	for x := range resp {
		var trades []*gctrpc.TradeHistory
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
			trades = append(trades, t)
		}
		o := &gctrpc.OrderDetails{
			Exchange:      r.Exchange,
			Id:            resp[x].ID,
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
			o.CreationTime = s.unixTimestamp(resp[x].Date)
		}
		if !resp[x].LastUpdated.IsZero() {
			o.UpdateTime = s.unixTimestamp(resp[x].LastUpdated)
		}
		orders = append(orders, o)
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
	exch := s.GetExchangeByName(r.Exchange)
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

	var orders []*gctrpc.OrderDetails
	for x := range resp {
		var trades []*gctrpc.TradeHistory
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
			trades = append(trades, t)
		}
		o := &gctrpc.OrderDetails{
			Exchange:      r.Exchange,
			Id:            resp[x].ID,
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
			o.CreationTime = s.unixTimestamp(resp[x].Date)
		}
		if !resp[x].LastUpdated.IsZero() {
			o.UpdateTime = s.unixTimestamp(resp[x].LastUpdated)
		}
		orders = append(orders, o)
	}

	return &gctrpc.GetOrdersResponse{Orders: orders}, nil
}

// GetOrder returns order information based on exchange and order ID
func (s *RPCServer) GetOrder(_ context.Context, r *gctrpc.GetOrderRequest) (*gctrpc.OrderDetails, error) {
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, pair)
	if err != nil {
		return nil, err
	}

	result, err := s.OrderManager.GetOrderInfo(r.Exchange, r.OrderId, pair, a)
	if err != nil {
		return nil, fmt.Errorf("error whilst trying to retrieve info for order %s: %w", r.OrderId, err)
	}
	var trades []*gctrpc.TradeHistory
	for i := range result.Trades {
		trades = append(trades, &gctrpc.TradeHistory{
			CreationTime: s.unixTimestamp(result.Trades[i].Timestamp),
			Id:           result.Trades[i].TID,
			Price:        result.Trades[i].Price,
			Amount:       result.Trades[i].Amount,
			Exchange:     result.Trades[i].Exchange,
			AssetType:    result.Trades[i].Type.String(),
			OrderSide:    result.Trades[i].Side.String(),
			Fee:          result.Trades[i].Fee,
			Total:        result.Trades[i].Total,
		})
	}

	var creationTime, updateTime int64
	if !result.Date.IsZero() {
		creationTime = s.unixTimestamp(result.Date)
	}
	if !result.LastUpdated.IsZero() {
		updateTime = s.unixTimestamp(result.LastUpdated)
	}

	return &gctrpc.OrderDetails{
		Exchange:      result.Exchange,
		Id:            result.ID,
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
func (s *RPCServer) SubmitOrder(_ context.Context, r *gctrpc.SubmitOrderRequest) (*gctrpc.SubmitOrderResponse, error) {
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	submission := &order.Submit{
		Pair:          p,
		Side:          order.Side(r.Side),
		Type:          order.Type(r.OrderType),
		Amount:        r.Amount,
		Price:         r.Price,
		ClientID:      r.ClientId,
		ClientOrderID: r.ClientId,
		Exchange:      r.Exchange,
		AssetType:     a,
	}

	resp, err := s.OrderManager.Submit(submission)
	if err != nil {
		return &gctrpc.SubmitOrderResponse{}, err
	}

	var trades []*gctrpc.Trades
	for i := range resp.Trades {
		trades = append(trades, &gctrpc.Trades{
			Amount:   resp.Trades[i].Amount,
			Price:    resp.Trades[i].Price,
			Fee:      resp.Trades[i].Fee,
			FeeAsset: resp.Trades[i].FeeAsset,
		})
	}

	return &gctrpc.SubmitOrderResponse{
		OrderId:     resp.OrderID,
		OrderPlaced: resp.IsOrderPlaced,
		Trades:      trades,
	}, err
}

// SimulateOrder simulates an order specified by exchange, currency pair and asset
// type
func (s *RPCServer) SimulateOrder(_ context.Context, r *gctrpc.SimulateOrderRequest) (*gctrpc.SimulateOrderResponse, error) {
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch := s.GetExchangeByName(r.Exchange)
	err := checkParams(r.Exchange, exch, asset.Spot, p)
	if err != nil {
		return nil, err
	}

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
func (s *RPCServer) WhaleBomb(_ context.Context, r *gctrpc.WhaleBombRequest) (*gctrpc.SimulateOrderResponse, error) {
	if r.Pair == nil {
		return nil, errCurrencyPairUnset
	}

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch := s.GetExchangeByName(r.Exchange)
	err := checkParams(r.Exchange, exch, asset.Spot, p)
	if err != nil {
		return nil, err
	}

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
func (s *RPCServer) CancelOrder(_ context.Context, r *gctrpc.CancelOrderRequest) (*gctrpc.GenericResponse, error) {
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	err = s.OrderManager.Cancel(&order.Cancel{
		Exchange:      r.Exchange,
		AccountID:     r.AccountId,
		ID:            r.OrderId,
		Side:          order.Side(r.Side),
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
func (s *RPCServer) CancelBatchOrders(_ context.Context, r *gctrpc.CancelBatchOrdersRequest) (*gctrpc.CancelBatchOrdersResponse, error) {
	pair := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	assetType, err := asset.New(r.AssetType)
	if err != nil {
		return nil, err
	}

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, assetType, pair)
	if err != nil {
		return nil, err
	}

	status := make(map[string]string)
	var request []order.Cancel
	orders := strings.Split(r.OrdersId, ",")
	for _, orderID := range orders {
		status[orderID] = order.Cancelled.String()
		request = append(request, order.Cancel{
			AccountID:     r.AccountId,
			ID:            orderID,
			Side:          order.Side(r.Side),
			WalletAddress: r.WalletAddress,
			Pair:          pair,
			AssetType:     assetType,
		})
	}

	_, err = exch.CancelBatchOrders(request)
	if err != nil {
		return nil, err
	}

	return &gctrpc.CancelBatchOrdersResponse{
		Orders: []*gctrpc.CancelBatchOrdersResponse_Orders{{
			OrderStatus: status,
		}},
	}, nil
}

// CancelAllOrders cancels all orders, filterable by exchange
func (s *RPCServer) CancelAllOrders(_ context.Context, r *gctrpc.CancelAllOrdersRequest) (*gctrpc.CancelAllOrdersResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return &gctrpc.CancelAllOrdersResponse{}, errExchangeNotLoaded
	}

	resp, err := exch.CancelAllOrders(nil)
	if err != nil {
		return &gctrpc.CancelAllOrdersResponse{}, err
	}

	return &gctrpc.CancelAllOrdersResponse{
		Count: resp.Count, // count of deleted orders
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

	exch := s.GetExchangeByName(r.Exchange)
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
	return &gctrpc.GenericResponse{Status: MsgStatusSuccess,
		Data: fmt.Sprintf("event %d removed", r.Id)}, nil
}

// GetCryptocurrencyDepositAddresses returns a list of cryptocurrency deposit
// addresses specified by an exchange
func (s *RPCServer) GetCryptocurrencyDepositAddresses(_ context.Context, r *gctrpc.GetCryptocurrencyDepositAddressesRequest) (*gctrpc.GetCryptocurrencyDepositAddressesResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
	}

	result, err := s.GetCryptocurrencyDepositAddressesByExchange(r.Exchange)
	return &gctrpc.GetCryptocurrencyDepositAddressesResponse{Addresses: result}, err
}

// GetCryptocurrencyDepositAddress returns a cryptocurrency deposit address
// specified by exchange and cryptocurrency
func (s *RPCServer) GetCryptocurrencyDepositAddress(_ context.Context, r *gctrpc.GetCryptocurrencyDepositAddressRequest) (*gctrpc.GetCryptocurrencyDepositAddressResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
	}

	addr, err := s.GetExchangeCryptocurrencyDepositAddress(r.Exchange, "", currency.NewCode(r.Cryptocurrency))
	return &gctrpc.GetCryptocurrencyDepositAddressResponse{Address: addr}, err
}

// WithdrawCryptocurrencyFunds withdraws cryptocurrency funds specified by
// exchange
func (s *RPCServer) WithdrawCryptocurrencyFunds(_ context.Context, r *gctrpc.WithdrawCryptoRequest) (*gctrpc.WithdrawResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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
		},
	}

	resp, err := s.Engine.WithdrawManager.SubmitWithdrawal(request)
	if err != nil {
		return nil, err
	}

	return &gctrpc.WithdrawResponse{
		Id:     resp.ID.String(),
		Status: resp.Exchange.Status,
	}, nil
}

// WithdrawFiatFunds withdraws fiat funds specified by exchange
func (s *RPCServer) WithdrawFiatFunds(_ context.Context, r *gctrpc.WithdrawFiatRequest) (*gctrpc.WithdrawResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
	}

	var bankAccount *banking.Account
	bankAccount, err := banking.GetBankAccountByID(r.BankAccountId)
	if err != nil {
		base := exch.GetBase()
		if base == nil {
			return nil, errExchangeBaseNotFound
		}
		bankAccount, err = base.GetExchangeBankAccounts(r.BankAccountId, r.Currency)
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

	resp, err := s.Engine.WithdrawManager.SubmitWithdrawal(request)
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
func (s *RPCServer) WithdrawalEventsByExchange(_ context.Context, r *gctrpc.WithdrawalEventsByExchangeRequest) (*gctrpc.WithdrawalEventsByExchangeResponse, error) {
	if !s.Config.Database.Enabled {
		if r.Id == "" {
			exch := s.GetExchangeByName(r.Exchange)
			if exch == nil {
				return nil, errExchangeNotLoaded
			}

			c := currency.NewCode(strings.ToUpper(r.Currency))
			ret, err := exch.GetWithdrawalsHistory(c)
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
	start, err := time.Parse(common.SimpleTimeFormat, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.End)
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

		ps, err := exchCfg.CurrencyPairs.Get(assetTypes[x])
		if err != nil {
			return nil, err
		}

		resp.SupportedAssets[assetTypes[x].String()] = &gctrpc.PairsSupported{
			AvailablePairs: ps.Available.Join(),
			EnabledPairs:   ps.Enabled.Join(),
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, currency.Pair{})
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
	var newErrors common.Errors
	for i := range r.Pairs {
		var p currency.Pair
		p, err = currency.NewPairFromStrings(r.Pairs[i].Base, r.Pairs[i].Quote)
		if err != nil {
			return nil, err
		}

		if r.Enable {
			err = exchCfg.CurrencyPairs.EnablePair(a,
				p.Format(pairFmt.Delimiter, pairFmt.Uppercase))
			if err != nil {
				newErrors = append(newErrors, err)
				continue
			}
			err = base.CurrencyPairs.EnablePair(a, p)
			if err != nil {
				newErrors = append(newErrors, err)
				continue
			}
			pass = true
			continue
		}

		err = exchCfg.CurrencyPairs.DisablePair(a,
			p.Format(pairFmt.Delimiter, pairFmt.Uppercase))
		if err != nil {
			newErrors = append(newErrors, err)
			continue
		}
		err = base.CurrencyPairs.DisablePair(a, p)
		if err != nil {
			newErrors = append(newErrors, err)
			continue
		}
		pass = true
	}

	if exch.IsWebsocketEnabled() && pass && base.Websocket.IsConnected() {
		err = exch.FlushWebsocketChannels()
		if err != nil {
			newErrors = append(newErrors, err)
		}
	}

	if newErrors != nil {
		return nil, newErrors
	}

	return &gctrpc.GenericResponse{Status: MsgStatusSuccess}, nil
}

// GetOrderbookStream streams the requested updated orderbook
func (s *RPCServer) GetOrderbookStream(r *gctrpc.GetOrderbookStreamRequest, stream gctrpc.GoCryptoTrader_GetOrderbookStreamServer) error {
	a, err := asset.New(r.AssetType)
	if err != nil {
		return err
	}

	p := currency.Pair{
		Delimiter: r.Pair.Delimiter,
		Base:      currency.NewCode(r.Pair.Base),
		Quote:     currency.NewCode(r.Pair.Quote),
	}

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return err
	}

	depth, err := orderbook.GetDepth(r.Exchange, p, a)
	if err != nil {
		return err
	}

	for {
		base := depth.Retrieve()
		bids := make([]*gctrpc.OrderbookItem, len(base.Bids))
		for i := range base.Bids {
			bids[i] = &gctrpc.OrderbookItem{
				Amount: base.Bids[i].Amount,
				Price:  base.Bids[i].Price,
				Id:     base.Bids[i].ID}
		}
		asks := make([]*gctrpc.OrderbookItem, len(base.Asks))
		for i := range base.Asks {
			asks[i] = &gctrpc.OrderbookItem{
				Amount: base.Asks[i].Amount,
				Price:  base.Asks[i].Price,
				Id:     base.Asks[i].ID}
		}
		err := stream.Send(&gctrpc.OrderbookResponse{
			Pair:      &gctrpc.CurrencyPair{Base: r.Pair.Base, Quote: r.Pair.Quote},
			Bids:      bids,
			Asks:      asks,
			AssetType: r.AssetType,
		})
		if err != nil {
			return err
		}
		<-depth.Wait(nil)
	}
}

// GetExchangeOrderbookStream streams all orderbooks associated with an exchange
func (s *RPCServer) GetExchangeOrderbookStream(r *gctrpc.GetExchangeOrderbookStreamRequest, stream gctrpc.GoCryptoTrader_GetExchangeOrderbookStreamServer) error {
	if r.Exchange == "" {
		return errExchangeNameUnset
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

		ob := (*data.(*interface{})).(orderbook.Base)
		bids := make([]*gctrpc.OrderbookItem, len(ob.Bids))
		for i := range ob.Bids {
			bids[i] = &gctrpc.OrderbookItem{
				Amount: ob.Bids[i].Amount,
				Price:  ob.Bids[i].Price,
				Id:     ob.Bids[i].ID}
		}
		asks := make([]*gctrpc.OrderbookItem, len(ob.Asks))
		for i := range ob.Asks {
			asks[i] = &gctrpc.OrderbookItem{
				Amount: ob.Asks[i].Amount,
				Price:  ob.Asks[i].Price,
				Id:     ob.Asks[i].ID}
		}
		err := stream.Send(&gctrpc.OrderbookResponse{
			Pair: &gctrpc.CurrencyPair{Base: ob.Pair.Base.String(),
				Quote: ob.Pair.Quote.String()},
			Bids:      bids,
			Asks:      asks,
			AssetType: ob.Asset.String(),
		})
		if err != nil {
			return err
		}
	}
}

// GetTickerStream streams the requested updated ticker
func (s *RPCServer) GetTickerStream(r *gctrpc.GetTickerStreamRequest, stream gctrpc.GoCryptoTrader_GetTickerStreamServer) error {
	if r.Exchange == "" {
		return errExchangeNameUnset
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
		t := (*data.(*interface{})).(ticker.Price)

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
func (s *RPCServer) GetExchangeTickerStream(r *gctrpc.GetExchangeTickerStreamRequest, stream gctrpc.GoCryptoTrader_GetExchangeTickerStreamServer) error {
	if r.Exchange == "" {
		return errExchangeNameUnset
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
		t := (*data.(*interface{})).(ticker.Price)

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
	start, err := time.Parse(common.SimpleTimeFormat, r.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.EndDate)
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
func (s *RPCServer) GetHistoricCandles(_ context.Context, r *gctrpc.GetHistoricCandlesRequest) (*gctrpc.GetHistoricCandlesResponse, error) {
	start, err := time.Parse(common.SimpleTimeFormat, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.End)
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, pair)
	if err != nil {
		return nil, err
	}

	interval := kline.Interval(r.TimeInterval)

	resp := gctrpc.GetHistoricCandlesResponse{
		Interval: interval.Short(),
		Pair:     r.Pair,
		Start:    r.Start,
		End:      r.End,
	}

	var klineItem kline.Item
	if r.UseDb {
		klineItem, err = kline.LoadFromDatabase(r.Exchange,
			pair,
			a,
			interval,
			start,
			end)
		if err != nil {
			return nil, err
		}
	} else {
		if r.ExRequest {
			klineItem, err = exch.GetHistoricCandlesExtended(pair,
				a,
				start,
				end,
				interval)
		} else {
			klineItem, err = exch.GetHistoricCandles(pair,
				a,
				start,
				end,
				interval)
		}
	}

	if err != nil {
		return nil, err
	}

	if r.FillMissingWithTrades {
		var tradeDataKline *kline.Item
		tradeDataKline, err = fillMissingCandlesWithStoredTrades(start, end, &klineItem)
		if err != nil {
			return nil, err
		}
		klineItem.Candles = append(klineItem.Candles, tradeDataKline.Candles...)
	}

	resp.Exchange = klineItem.Exchange
	for i := range klineItem.Candles {
		resp.Candle = append(resp.Candle, &gctrpc.Candle{
			Time:   klineItem.Candles[i].Time.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			Low:    klineItem.Candles[i].Low,
			High:   klineItem.Candles[i].High,
			Open:   klineItem.Candles[i].Open,
			Close:  klineItem.Candles[i].Close,
			Volume: klineItem.Candles[i].Volume,
		})
	}

	if r.Sync && !r.UseDb {
		_, err = kline.StoreInDatabase(&klineItem, r.Force)
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
	var response kline.Item
	var candleTimes []time.Time
	for i := range klineItem.Candles {
		candleTimes = append(candleTimes, klineItem.Candles[i].Time)
	}
	ranges, err := timeperiods.FindTimeRangesContainingData(startTime, endTime, klineItem.Interval.Duration(), candleTimes)
	if err != nil {
		return nil, err
	}

	for i := range ranges {
		if ranges[i].HasDataInRange {
			continue
		}
		var tradeCandles kline.Item
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

		for i := range tradeCandles.Candles {
			response.Candles = append(response.Candles, tradeCandles.Candles[i])
		}

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
		vm := v.(*gctscript.VM)
		resp.Scripts = append(resp.Scripts, &gctrpc.GCTScript{
			UUID:    vm.ID.String(),
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

	UUID, err := uuid.FromString(r.Script.UUID)
	if err != nil {
		return &gctrpc.GCTScriptQueryResponse{Status: MsgStatusError, Data: err.Error()}, nil
	}

	if v, f := gctscript.AllVMSync.Load(UUID); f {
		resp := &gctrpc.GCTScriptQueryResponse{
			Status: MsgStatusOK,
			Script: &gctrpc.GCTScript{
				Name:    v.(*gctscript.VM).ShortName(),
				UUID:    v.(*gctscript.VM).ID.String(),
				Path:    v.(*gctscript.VM).Path,
				NextRun: v.(*gctscript.VM).NextRun.String(),
			},
		}
		data, err := v.(*gctscript.VM).Read()
		if err != nil {
			return nil, err
		}
		resp.Data = string(data)
		return resp, nil
	}
	return &gctrpc.GCTScriptQueryResponse{Status: MsgStatusError, Data: "UUID not found"}, nil
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
	err := gctVM.Load(script)
	if err != nil {
		return &gctrpc.GenericResponse{
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

	UUID, err := uuid.FromString(r.Script.UUID)
	if err != nil {
		return &gctrpc.GenericResponse{Status: MsgStatusError, Data: err.Error()}, nil
	}

	if v, f := gctscript.AllVMSync.Load(UUID); f {
		err = v.(*gctscript.VM).Shutdown()
		status := " terminated"
		if err != nil {
			status = " " + err.Error()
		}
		return &gctrpc.GenericResponse{Status: MsgStatusOK, Data: v.(*gctscript.VM).ID.String() + status}, nil
	}
	return &gctrpc.GenericResponse{Status: MsgStatusError, Data: "no running script found"}, nil
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
		err = os.MkdirAll(f, 0770)
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
	data, err := ioutil.ReadFile(filename)
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
		return &gctrpc.GenericResponse{Status: "error", Data: err.Error()}, nil
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
			return &gctrpc.GenericResponse{Status: "error", Data: err.Error()}, nil
		}
		return &gctrpc.GenericResponse{Status: "success", Data: "script " + r.Script + " removed from autoload list"}, nil
	}

	err := s.gctScriptManager.Autoload(r.Script, false)
	if err != nil {
		return &gctrpc.GenericResponse{Status: "error", Data: err.Error()}, nil
	}
	return &gctrpc.GenericResponse{Status: "success", Data: "script " + r.Script + " added to autoload list"}, nil
}

// SetExchangeAsset enables or disables an exchanges asset type
func (s *RPCServer) SetExchangeAsset(_ context.Context, r *gctrpc.SetExchangeAssetRequest) (*gctrpc.GenericResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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
			exchCfg.CurrencyPairs.StorePairs(assets[i], pairs, true)
			base.CurrencyPairs.StorePairs(assets[i], pairs, true)
		}
	} else {
		for i := range assets {
			exchCfg.CurrencyPairs.StorePairs(assets[i], nil, true)
			base.CurrencyPairs.StorePairs(assets[i], nil, true)
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
func (s *RPCServer) UpdateExchangeSupportedPairs(_ context.Context, r *gctrpc.UpdateExchangeSupportedPairsRequest) (*gctrpc.GenericResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
	}

	base := exch.GetBase()
	if base == nil {
		return nil, errExchangeBaseNotFound
	}

	if !base.GetEnabledFeatures().AutoPairUpdates {
		return nil,
			errors.New("cannot auto pair update for exchange, a manual update is needed")
	}

	err := exch.UpdateTradablePairs(false)
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
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
	}

	return &gctrpc.GetExchangeAssetsResponse{
		Assets: exch.GetAssetTypes(false).JoinToString(","),
	}, nil
}

// WebsocketGetInfo returns websocket connection information
func (s *RPCServer) WebsocketGetInfo(_ context.Context, r *gctrpc.WebsocketGetInfoRequest) (*gctrpc.WebsocketGetInfoResponse, error) {
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	start, err := time.Parse(common.SimpleTimeFormat, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.End)
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
	start, err := time.Parse(common.SimpleTimeFormat, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.End)
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, p)
	if err != nil {
		return nil, err
	}

	var trades []trade.Data
	trades, err = trade.GetTradesInRange(r.Exchange, r.AssetType, r.Pair.Base, r.Pair.Quote, start, end)
	if err != nil {
		return nil, err
	}
	if len(trades) == 0 {
		return nil, errNoTrades
	}
	interval := kline.Interval(r.TimeInterval)
	var klineItem kline.Item
	klineItem, err = trade.ConvertTradesToCandles(interval, trades...)
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
			Time:   klineItem.Candles[i].Time.In(time.UTC).Format(common.SimpleTimeFormatWithTimezone),
			Low:    klineItem.Candles[i].Low,
			High:   klineItem.Candles[i].High,
			Open:   klineItem.Candles[i].Open,
			Close:  klineItem.Candles[i].Close,
			Volume: klineItem.Candles[i].Volume,
		})
	}

	if r.Sync {
		_, err = kline.StoreInDatabase(&klineItem, r.Force)
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

	exch := s.GetExchangeByName(r.ExchangeName)
	err = checkParams(r.ExchangeName, exch, a, p)
	if err != nil {
		return nil, err
	}

	start, err := time.Parse(common.SimpleTimeFormat, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.End)
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
	var candleTimes []time.Time
	for i := range klineItem.Candles {
		candleTimes = append(candleTimes, klineItem.Candles[i].Time)
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

	exch := s.GetExchangeByName(r.ExchangeName)
	err = checkParams(r.ExchangeName, exch, a, p)
	if err != nil {
		return nil, err
	}
	start, err := time.Parse(common.SimpleTimeFormat, r.Start)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.End)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}
	start = start.Truncate(time.Hour)
	end = end.Truncate(time.Hour)

	intervalMap := make(map[time.Time]bool)
	iterationTime := start
	for iterationTime.Before(end) {
		intervalMap[iterationTime] = false
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
	var tradeTimes []time.Time
	for i := range trades {
		tradeTimes = append(tradeTimes, trades[i].Timestamp)
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
	exch := s.GetExchangeByName(r.Exchange)
	if exch == nil {
		return nil, errExchangeNotLoaded
	}
	b := exch.GetBase()
	b.SetSaveTradeDataStatus(r.Status)

	return &gctrpc.GenericResponse{
		Status: "success",
	}, nil
}

// GetHistoricTrades returns trades between a set of dates
func (s *RPCServer) GetHistoricTrades(r *gctrpc.GetSavedTradesRequest, stream gctrpc.GoCryptoTrader_GetHistoricTradesServer) error {
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, cp)
	if err != nil {
		return err
	}
	var trades []trade.Data
	start, err := time.Parse(common.SimpleTimeFormat, r.Start)
	if err != nil {
		return fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.End)
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
		trades, err = exch.GetHistoricTrades(cp, a, iterateStartTime, iterateEndTime)
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
func (s *RPCServer) GetRecentTrades(_ context.Context, r *gctrpc.GetSavedTradesRequest) (*gctrpc.SavedTradesResponse, error) {
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

	exch := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, exch, a, cp)
	if err != nil {
		return nil, err
	}
	var trades []trade.Data
	trades, err = exch.GetRecentTrades(cp, asset.Item(r.AssetType))
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
		return fmt.Errorf("%s %w", exchName, ErrExchangeNotFound)
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
	e := s.GetExchangeByName(r.Exchange)
	err = checkParams(r.Exchange, e, a, p)
	if err != nil {
		return nil, err
	}

	start, err := time.Parse(common.SimpleTimeFormat, r.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse end time %v", errInvalidTimes, err)
	}
	err = common.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}

	job := DataHistoryJob{
		Nickname:         r.Nickname,
		Exchange:         r.Exchange,
		Asset:            a,
		Pair:             p,
		StartDate:        start,
		EndDate:          end,
		Interval:         kline.Interval(r.Interval),
		RunBatchLimit:    r.BatchSize,
		RequestSizeLimit: r.RequestSizeLimit,
		DataType:         dataHistoryDataType(r.DataType),
		Status:           dataHistoryStatusActive,
		MaxRetryAttempts: r.MaxRetryAttempts,
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
		StartDate:        result.StartDate.Format(common.SimpleTimeFormat),
		EndDate:          result.EndDate.Format(common.SimpleTimeFormat),
		Interval:         int64(result.Interval.Duration()),
		RequestSizeLimit: result.RequestSizeLimit,
		DataType:         result.DataType.String(),
		MaxRetryAttempts: result.MaxRetryAttempts,
		BatchSize:        result.RunBatchLimit,
		JobResults:       jobResults,
		Status:           result.Status.String(),
	}, nil
}

// DeleteDataHistoryJob deletes a data history job from the database
func (s *RPCServer) DeleteDataHistoryJob(_ context.Context, r *gctrpc.GetDataHistoryJobDetailsRequest) (*gctrpc.GenericResponse, error) {
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
	err := s.dataHistoryManager.DeleteJob(r.Nickname, r.Id)
	if err != nil {
		log.Error(log.GRPCSys, err)
		status = "failed"
	}

	return &gctrpc.GenericResponse{Status: status}, err
}

// GetActiveDataHistoryJobs returns any active data history job details
func (s *RPCServer) GetActiveDataHistoryJobs(_ context.Context, _ *gctrpc.GetInfoRequest) (*gctrpc.DataHistoryJobs, error) {
	jobs, err := s.dataHistoryManager.GetActiveJobs()
	if err != nil {
		return nil, err
	}

	var response []*gctrpc.DataHistoryJob
	for i := range jobs {
		response = append(response, &gctrpc.DataHistoryJob{
			Id:       jobs[i].ID.String(),
			Nickname: jobs[i].Nickname,
			Exchange: jobs[i].Exchange,
			Asset:    jobs[i].Asset.String(),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: jobs[i].Pair.Delimiter,
				Base:      jobs[i].Pair.Base.String(),
				Quote:     jobs[i].Pair.Quote.String(),
			},
			StartDate:        jobs[i].StartDate.Format(common.SimpleTimeFormat),
			EndDate:          jobs[i].EndDate.Format(common.SimpleTimeFormat),
			Interval:         int64(jobs[i].Interval.Duration()),
			RequestSizeLimit: jobs[i].RequestSizeLimit,
			DataType:         jobs[i].DataType.String(),
			MaxRetryAttempts: jobs[i].MaxRetryAttempts,
			BatchSize:        jobs[i].RunBatchLimit,
			Status:           jobs[i].Status.String(),
		})
	}
	return &gctrpc.DataHistoryJobs{Results: response}, nil
}

// GetDataHistoryJobsBetween returns all jobs created between supplied dates
func (s *RPCServer) GetDataHistoryJobsBetween(_ context.Context, r *gctrpc.GetDataHistoryJobsBetweenRequest) (*gctrpc.DataHistoryJobs, error) {
	if r == nil {
		return nil, errNilRequestData
	}
	start, err := time.Parse(common.SimpleTimeFormat, r.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%w cannot parse start time %v", errInvalidTimes, err)
	}
	end, err := time.Parse(common.SimpleTimeFormat, r.EndDate)
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
	var respJobs []*gctrpc.DataHistoryJob
	for i := range jobs {
		respJobs = append(respJobs, &gctrpc.DataHistoryJob{
			Id:       jobs[i].ID.String(),
			Nickname: jobs[i].Nickname,
			Exchange: jobs[i].Exchange,
			Asset:    jobs[i].Asset.String(),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: jobs[i].Pair.Delimiter,
				Base:      jobs[i].Pair.Base.String(),
				Quote:     jobs[i].Pair.Quote.String(),
			},
			StartDate:        jobs[i].StartDate.Format(common.SimpleTimeFormat),
			EndDate:          jobs[i].EndDate.Format(common.SimpleTimeFormat),
			Interval:         int64(jobs[i].Interval.Duration()),
			RequestSizeLimit: jobs[i].RequestSizeLimit,
			DataType:         jobs[i].DataType.String(),
			MaxRetryAttempts: jobs[i].MaxRetryAttempts,
			BatchSize:        jobs[i].RunBatchLimit,
			Status:           jobs[i].Status.String(),
		})
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
		StartDate:       job.StartDate.Format(common.SimpleTimeFormat),
		EndDate:         job.EndDate.Format(common.SimpleTimeFormat),
		Interval:        int64(job.Interval.Duration()),
		DataType:        job.DataType.String(),
		Status:          job.Status.String(),
		ResultSummaries: job.ResultRanges,
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
