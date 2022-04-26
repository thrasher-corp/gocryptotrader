package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	gctengine "github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// RPCServer struct
type RPCServer struct {
	btrpc.UnimplementedBacktesterServer
	*config.BacktesterConfig
}

func SetupRPCServer(cfg *config.BacktesterConfig) *RPCServer {
	return &RPCServer{
		BacktesterConfig: cfg,
	}

}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer(server *RPCServer) error {
	targetDir := utils.GetTLSDir(server.GRPC.TLSDir)
	if err := gctengine.CheckCerts(targetDir); err != nil {
		return err
	}
	log.Debugf(log.GRPCSys, "gRPC server support enabled. Starting gRPC server on https://%v.\n", server.GRPC.ListenAddress)
	lis, err := net.Listen("tcp", server.GRPC.ListenAddress)
	if err != nil {
		return err

	}

	creds, err := credentials.NewServerTLSFromFile(filepath.Join(targetDir, "cert.pem"), filepath.Join(targetDir, "key.pem"))
	if err != nil {
		return err
	}

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(grpcauth.UnaryServerInterceptor(server.authenticateClient)),
	}
	s := grpc.NewServer(opts...)
	btrpc.RegisterBacktesterServer(s, server)

	go func() {
		if err = s.Serve(lis); err != nil {
			log.Error(log.GRPCSys, err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC server started!")

	if server.GRPC.GRPCProxyEnabled {
		server.StartRPCRESTProxy()
	}
	return nil
}

// StartRPCRESTProxy starts a gRPC proxy
func (s *RPCServer) StartRPCRESTProxy() {
	log.Debugf(log.GRPCSys, "gRPC proxy server support enabled. Starting gRPC proxy server on http://%v.\n", s.GRPC.GRPCProxyListenAddress)
	targetDir := utils.GetTLSDir(s.GRPC.TLSDir)
	creds, err := credentials.NewClientTLSFromFile(filepath.Join(targetDir, "cert.pem"), "")
	if err != nil {
		log.Errorf(log.GRPCSys, "Unabled to start gRPC proxy. Err: %s\n", err)
		return
	}

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: s.GRPC.Username,
			Password: s.GRPC.Password,
		}),
	}
	err = gctrpc.RegisterGoCryptoTraderHandlerFromEndpoint(context.Background(),
		mux, s.GRPC.ListenAddress, opts)
	if err != nil {
		log.Errorf(log.GRPCSys, "Failed to register gRPC proxy. Err: %s\n", err)
		return
	}

	go func() {
		if err := http.ListenAndServe(s.GRPC.GRPCProxyListenAddress, mux); err != nil {
			log.Errorf(log.GRPCSys, "gRPC proxy failed to server: %s\n", err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC proxy server started!")
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

	creds := strings.Split(string(decoded), ":")
	username := creds[0]
	password := creds[1]

	if username != s.GRPC.Username ||
		password != s.GRPC.Password {
		return ctx, fmt.Errorf("username/password mismatch")
	}
	return exchange.ParseCredentialsMetadata(ctx, md)
}

// ExecuteStrategyFromFile will backtest a strategy from the filepath provided
func (s *RPCServer) ExecuteStrategyFromFile(_ context.Context, request *btrpc.ExecuteStrategyFromFileRequest) (*btrpc.ExecuteStrategyResponse, error) {
	dir := request.StrategyFilePath
	cfg, err := config.ReadStrategyConfigFromFile(dir)
	if err != nil {
		return nil, err
	}
	err = ExecuteStrategy(cfg, s.BacktesterConfig)
	if err != nil {
		return nil, err
	}
	return &btrpc.ExecuteStrategyResponse{
		Success: true,
	}, nil
}

// ExecuteStrategyFromConfig will backtest a strategy config built from a GRPC command
// this should be a preferred method of interacting with backtester, as it allows for very quick
// minor tweaks to strategy to determine the best result - SO LONG AS YOU DONT OVERFIT
func (s *RPCServer) ExecuteStrategyFromConfig(_ context.Context, request *btrpc.ExecuteStrategyFromConfigRequest) (*btrpc.ExecuteStrategyResponse, error) {
	// al the decimal conversions
	rfr, err := decimal.NewFromString(request.Config.StatisticSettings.RiskFreeRate)
	if err != nil {
		return nil, err
	}
	maximumOrdersWithLeverageRatio, err := decimal.NewFromString(request.Config.PortfolioSettings.Leverage.MaximumOrdersWithLeverageRatio)
	if err != nil {
		return nil, err
	}
	maximumOrderLeverageRate, err := decimal.NewFromString(request.Config.PortfolioSettings.Leverage.MaximumLeverageRate)
	if err != nil {
		return nil, err
	}
	maximumCollateralLeverageRate, err := decimal.NewFromString(request.Config.PortfolioSettings.Leverage.MaximumCollateralLeverageRate)
	if err != nil {
		return nil, err
	}

	buySideMinimumSize, err := decimal.NewFromString(request.Config.PortfolioSettings.BuySide.MinimumSize)
	if err != nil {
		return nil, err
	}
	buySideMaximumSize, err := decimal.NewFromString(request.Config.PortfolioSettings.BuySide.MaximumSize)
	if err != nil {
		return nil, err
	}
	buySideMaximumTotal, err := decimal.NewFromString(request.Config.PortfolioSettings.BuySide.MaximumTotal)
	if err != nil {
		return nil, err
	}

	sellSideMinimumSize, err := decimal.NewFromString(request.Config.PortfolioSettings.SellSide.MinimumSize)
	if err != nil {
		return nil, err
	}
	sellSideMaximumSize, err := decimal.NewFromString(request.Config.PortfolioSettings.SellSide.MaximumSize)
	if err != nil {
		return nil, err
	}
	sellSideMaximumTotal, err := decimal.NewFromString(request.Config.PortfolioSettings.SellSide.MaximumTotal)
	if err != nil {
		return nil, err
	}

	var fundingSettings []config.ExchangeLevelFunding
	for i := range request.Config.FundingSettings.ExchangeLevelFunding {
		initialFunds, err := decimal.NewFromString(request.Config.FundingSettings.ExchangeLevelFunding[i].InitialFunds)
		if err != nil {
			return nil, err
		}
		transferFee, err := decimal.NewFromString(request.Config.FundingSettings.ExchangeLevelFunding[i].TransferFee)
		if err != nil {
			return nil, err
		}
		fundingSettings = append(fundingSettings, config.ExchangeLevelFunding{
			ExchangeName: request.Config.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
			Asset:        request.Config.FundingSettings.ExchangeLevelFunding[i].Asset,
			Currency:     request.Config.FundingSettings.ExchangeLevelFunding[i].Currency,
			InitialFunds: initialFunds,
			TransferFee:  transferFee,
		})
	}

	customSettings := make(map[string]interface{})
	for i := range request.Config.StrategySettings.CustomSettings {
		customSettings[request.Config.StrategySettings.CustomSettings[i].KeyField] = request.Config.StrategySettings.CustomSettings[i].KeyValue
	}

	var configSettings []config.CurrencySettings
	for i := range request.Config.CurrencySettings {
		currencySettingBuySideMinimumSize, err := decimal.NewFromString(request.Config.CurrencySettings[i].BuySide.MinimumSize)
		if err != nil {
			return nil, err
		}
		currencySettingBuySideMaximumSize, err := decimal.NewFromString(request.Config.CurrencySettings[i].BuySide.MaximumSize)
		if err != nil {
			return nil, err
		}
		currencySettingBuySideMaximumTotal, err := decimal.NewFromString(request.Config.CurrencySettings[i].BuySide.MaximumTotal)
		if err != nil {
			return nil, err
		}

		currencySettingSellSideMinimumSize, err := decimal.NewFromString(request.Config.CurrencySettings[i].SellSide.MinimumSize)
		if err != nil {
			return nil, err
		}
		currencySettingSellSideMaximumSize, err := decimal.NewFromString(request.Config.CurrencySettings[i].SellSide.MaximumSize)
		if err != nil {
			return nil, err
		}
		currencySettingSellSideMaximumTotal, err := decimal.NewFromString(request.Config.CurrencySettings[i].SellSide.MaximumTotal)
		if err != nil {
			return nil, err
		}

		minimumSlippagePercent, err := decimal.NewFromString(request.Config.CurrencySettings[i].MinSlippagePercent)
		if err != nil {
			return nil, err
		}

		maximumSlippagePercent, err := decimal.NewFromString(request.Config.CurrencySettings[i].MaxSlippagePercent)
		if err != nil {
			return nil, err
		}

		maximumHoldingsRatio, err := decimal.NewFromString(request.Config.CurrencySettings[i].MaximumHoldingsRatio)
		if err != nil {
			return nil, err
		}
		configSettings = append(configSettings, config.CurrencySettings{
			ExchangeName: request.Config.CurrencySettings[i].ExchangeName,
			Asset:        request.Config.CurrencySettings[i].Asset,
			Base:         request.Config.CurrencySettings[i].Base,
			Quote:        request.Config.CurrencySettings[i].Quote,
			//USDTrackingPair:               request.Config.CurrencySettings[i].,
			SpotDetails:    nil,
			FuturesDetails: nil,
			BuySide: config.MinMax{
				MinimumSize:  currencySettingBuySideMinimumSize,
				MaximumSize:  currencySettingBuySideMaximumSize,
				MaximumTotal: currencySettingBuySideMaximumTotal,
			},
			SellSide: config.MinMax{
				MinimumSize:  currencySettingSellSideMinimumSize,
				MaximumSize:  currencySettingSellSideMaximumSize,
				MaximumTotal: currencySettingSellSideMaximumTotal,
			},
			MinimumSlippagePercent:        minimumSlippagePercent,
			MaximumSlippagePercent:        maximumSlippagePercent,
			MakerFee:                      nil,
			TakerFee:                      nil,
			MaximumHoldingsRatio:          maximumHoldingsRatio,
			SkipCandleVolumeFitting:       request.Config.CurrencySettings[i].SkipCandleVolumeFitting,
			CanUseExchangeLimits:          request.Config.CurrencySettings[i].UseExchangeOrderLimits,
			ShowExchangeOrderLimitWarning: request.Config.CurrencySettings[i].UseExchangeOrderLimits,
			UseExchangePNLCalculation:     request.Config.CurrencySettings[i].UseExchange_PNLCalculation,
		})
	}

	cfg := &config.Config{
		Nickname: request.Config.Nickname,
		Goal:     request.Config.Goal,
		StrategySettings: config.StrategySettings{
			Name:                         request.Config.StrategySettings.Name,
			SimultaneousSignalProcessing: request.Config.StrategySettings.UseSimultaneousSignalProcessing,
			DisableUSDTracking:           request.Config.StrategySettings.Disable_USDTracking,
			CustomSettings:               customSettings,
		},
		FundingSettings: config.FundingSettings{
			UseExchangeLevelFunding: request.Config.FundingSettings.UseExchangeLevelFunding,
			ExchangeLevelFunding:    fundingSettings,
		},
		CurrencySettings: nil,
		DataSettings: config.DataSettings{
			Interval: time.Duration(request.Config.DataSettings.Interval),
			DataType: request.Config.DataSettings.Datatype,
		},
		PortfolioSettings: config.PortfolioSettings{
			Leverage: config.Leverage{
				CanUseLeverage:                 request.Config.PortfolioSettings.Leverage.CanUseLeverage,
				MaximumOrdersWithLeverageRatio: maximumOrdersWithLeverageRatio,
				MaximumOrderLeverageRate:       maximumOrderLeverageRate,
				MaximumCollateralLeverageRate:  maximumCollateralLeverageRate,
			},
			BuySide: config.MinMax{
				MinimumSize:  buySideMinimumSize,
				MaximumSize:  buySideMaximumSize,
				MaximumTotal: buySideMaximumTotal,
			},
			SellSide: config.MinMax{
				MinimumSize:  sellSideMinimumSize,
				MaximumSize:  sellSideMaximumSize,
				MaximumTotal: sellSideMaximumTotal,
			},
		},
		StatisticSettings: config.StatisticSettings{
			RiskFreeRate: rfr,
		},
	}

	if request.Config.DataSettings.ApiData != nil {

	}
	if request.Config.DataSettings.ApiData != nil {

	}
	if request.Config.DataSettings.ApiData != nil {

	}
	if request.Config.DataSettings.ApiData != nil {

	}

	err = ExecuteStrategy(cfg, s.BacktesterConfig)
	if err != nil {
		return nil, err
	}
	return &btrpc.ExecuteStrategyResponse{
		Success: true,
	}, nil
}
