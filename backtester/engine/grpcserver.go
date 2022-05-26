package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	gctengine "github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// GRPCServer struct
type GRPCServer struct {
	btrpc.BacktesterServer
	*config.BacktesterConfig
}

// SetupRPCServer sets up the gRPC server
func SetupRPCServer(cfg *config.BacktesterConfig) *GRPCServer {
	return &GRPCServer{
		BacktesterConfig: cfg,
	}
}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer(server *GRPCServer) error {
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
func (s *GRPCServer) StartRPCRESTProxy() {
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
	err = btrpc.RegisterBacktesterHandlerFromEndpoint(context.Background(),
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

func (s *GRPCServer) authenticateClient(ctx context.Context) (context.Context, error) {
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
func (s *GRPCServer) ExecuteStrategyFromFile(_ context.Context, request *btrpc.ExecuteStrategyFromFileRequest) (*btrpc.ExecuteStrategyResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("%w nil request", common.ErrNilArguments)
	}
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
func (s *GRPCServer) ExecuteStrategyFromConfig(_ context.Context, request *btrpc.ExecuteStrategyFromConfigRequest) (*btrpc.ExecuteStrategyResponse, error) {
	if request == nil || request.Config == nil {
		return nil, fmt.Errorf("%w nil request", common.ErrNilArguments)
	}

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

	fundingSettings := make([]config.ExchangeLevelFunding, len(request.Config.FundingSettings.ExchangeLevelFunding))
	for i := range request.Config.FundingSettings.ExchangeLevelFunding {
		var initialFunds, transferFee decimal.Decimal
		var a asset.Item
		initialFunds, err = decimal.NewFromString(request.Config.FundingSettings.ExchangeLevelFunding[i].InitialFunds)
		if err != nil {
			return nil, err
		}
		transferFee, err = decimal.NewFromString(request.Config.FundingSettings.ExchangeLevelFunding[i].TransferFee)
		if err != nil {
			return nil, err
		}
		a, err = asset.New(request.Config.FundingSettings.ExchangeLevelFunding[i].Asset)
		if err != nil {
			return nil, err
		}

		fundingSettings[i] = config.ExchangeLevelFunding{
			ExchangeName: request.Config.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
			Asset:        a,
			Currency:     currency.NewCode(request.Config.FundingSettings.ExchangeLevelFunding[i].Currency),
			InitialFunds: initialFunds,
			TransferFee:  transferFee,
		}
	}

	customSettings := make(map[string]interface{}, len(request.Config.StrategySettings.CustomSettings))
	for i := range request.Config.StrategySettings.CustomSettings {
		customSettings[request.Config.StrategySettings.CustomSettings[i].KeyField] = request.Config.StrategySettings.CustomSettings[i].KeyValue
	}

	configSettings := make([]config.CurrencySettings, len(request.Config.CurrencySettings))
	for i := range request.Config.CurrencySettings {
		var currencySettingBuySideMinimumSize, currencySettingBuySideMaximumSize,
			currencySettingBuySideMaximumTotal, currencySettingSellSideMinimumSize,
			currencySettingSellSideMaximumSize, currencySettingSellSideMaximumTotal,
			minimumSlippagePercent, maximumSlippagePercent, maximumHoldingsRatio decimal.Decimal
		var a asset.Item
		currencySettingBuySideMinimumSize, err = decimal.NewFromString(request.Config.CurrencySettings[i].BuySide.MinimumSize)
		if err != nil {
			return nil, err
		}
		currencySettingBuySideMaximumSize, err = decimal.NewFromString(request.Config.CurrencySettings[i].BuySide.MaximumSize)
		if err != nil {
			return nil, err
		}
		currencySettingBuySideMaximumTotal, err = decimal.NewFromString(request.Config.CurrencySettings[i].BuySide.MaximumTotal)
		if err != nil {
			return nil, err
		}

		currencySettingSellSideMinimumSize, err = decimal.NewFromString(request.Config.CurrencySettings[i].SellSide.MinimumSize)
		if err != nil {
			return nil, err
		}
		currencySettingSellSideMaximumSize, err = decimal.NewFromString(request.Config.CurrencySettings[i].SellSide.MaximumSize)
		if err != nil {
			return nil, err
		}
		currencySettingSellSideMaximumTotal, err = decimal.NewFromString(request.Config.CurrencySettings[i].SellSide.MaximumTotal)
		if err != nil {
			return nil, err
		}

		minimumSlippagePercent, err = decimal.NewFromString(request.Config.CurrencySettings[i].MinSlippagePercent)
		if err != nil {
			return nil, err
		}

		maximumSlippagePercent, err = decimal.NewFromString(request.Config.CurrencySettings[i].MaxSlippagePercent)
		if err != nil {
			return nil, err
		}

		maximumHoldingsRatio, err = decimal.NewFromString(request.Config.CurrencySettings[i].MaximumHoldingsRatio)
		if err != nil {
			return nil, err
		}
		a, err = asset.New(request.Config.CurrencySettings[i].Asset)
		if err != nil {
			return nil, err
		}
		var maker, taker *decimal.Decimal
		if request.Config.CurrencySettings[i].MakerFeeOverride != "" {
			// nil is a valid option
			var m decimal.Decimal
			m, err = decimal.NewFromString(request.Config.CurrencySettings[i].MakerFeeOverride)
			if err != nil {
				return nil, fmt.Errorf("%v %v %v-%v maker fee %w", request.Config.CurrencySettings[i].ExchangeName, request.Config.CurrencySettings[i].Asset, request.Config.CurrencySettings[i].Base, request.Config.CurrencySettings[i].Quote, err)
			}
			maker = &m
		}
		if request.Config.CurrencySettings[i].TakerFeeOverride != "" {
			// nil is a valid option
			var t decimal.Decimal
			t, err = decimal.NewFromString(request.Config.CurrencySettings[i].MakerFeeOverride)
			if err != nil {
				return nil, fmt.Errorf("%v %v %v-%v taker fee %w", request.Config.CurrencySettings[i].ExchangeName, request.Config.CurrencySettings[i].Asset, request.Config.CurrencySettings[i].Base, request.Config.CurrencySettings[i].Quote, err)
			}
			taker = &t
		}

		var spotDetails *config.SpotDetails
		if request.Config.CurrencySettings[i].SpotDetails != nil {
			spotDetails = &config.SpotDetails{}
			if request.Config.CurrencySettings[i].SpotDetails.InitialBaseFunds != "" {
				var ibf decimal.Decimal
				ibf, err = decimal.NewFromString(request.Config.CurrencySettings[i].SpotDetails.InitialBaseFunds)
				if err != nil {
					return nil, err
				}
				spotDetails.InitialBaseFunds = &ibf
			}
			if request.Config.CurrencySettings[i].SpotDetails.InitialQuoteFunds != "" {
				var iqf decimal.Decimal
				iqf, err = decimal.NewFromString(request.Config.CurrencySettings[i].SpotDetails.InitialQuoteFunds)
				if err != nil {
					return nil, err
				}
				spotDetails.InitialQuoteFunds = &iqf
			}
		}

		var futuresDetails *config.FuturesDetails
		if request.Config.CurrencySettings[i].FuturesDetails != nil &&
			request.Config.CurrencySettings[i].FuturesDetails.Leverage != nil {
			futuresDetails = &config.FuturesDetails{}
			var mowlr, mlr, mclr decimal.Decimal
			mowlr, err = decimal.NewFromString(request.Config.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio)
			if err != nil {
				return nil, err
			}
			mlr, err = decimal.NewFromString(request.Config.CurrencySettings[i].FuturesDetails.Leverage.MaximumLeverageRate)
			if err != nil {
				return nil, err
			}
			mclr, err = decimal.NewFromString(request.Config.CurrencySettings[i].FuturesDetails.Leverage.MaximumCollateralLeverageRate)
			if err != nil {
				return nil, err
			}

			futuresDetails.Leverage = config.Leverage{
				CanUseLeverage:                 request.Config.CurrencySettings[i].FuturesDetails.Leverage.CanUseLeverage,
				MaximumOrdersWithLeverageRatio: mowlr,
				MaximumOrderLeverageRate:       mlr,
				MaximumCollateralLeverageRate:  mclr,
			}
		}

		configSettings[i] = config.CurrencySettings{
			ExchangeName:   request.Config.CurrencySettings[i].ExchangeName,
			Asset:          a,
			Base:           currency.NewCode(request.Config.CurrencySettings[i].Base),
			Quote:          currency.NewCode(request.Config.CurrencySettings[i].Quote),
			SpotDetails:    spotDetails,
			FuturesDetails: futuresDetails,
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
			MakerFee:                      maker,
			TakerFee:                      taker,
			MaximumHoldingsRatio:          maximumHoldingsRatio,
			SkipCandleVolumeFitting:       request.Config.CurrencySettings[i].SkipCandleVolumeFitting,
			CanUseExchangeLimits:          request.Config.CurrencySettings[i].UseExchangeOrderLimits,
			ShowExchangeOrderLimitWarning: request.Config.CurrencySettings[i].UseExchangeOrderLimits,
			UseExchangePNLCalculation:     request.Config.CurrencySettings[i].UseExchange_PNLCalculation,
		}
	}

	var apiData *config.APIData
	if request.Config.DataSettings.ApiData != nil {
		apiData = &config.APIData{
			StartDate:        request.Config.DataSettings.ApiData.StartDate.AsTime(),
			EndDate:          request.Config.DataSettings.ApiData.EndDate.AsTime(),
			InclusiveEndDate: request.Config.DataSettings.ApiData.InclusiveEndDate,
		}
	}
	var dbData *config.DatabaseData
	if request.Config.DataSettings.DatabaseData != nil {
		cfg := database.Config{
			Enabled: request.Config.DataSettings.DatabaseData.Config.Enabled,
			Verbose: request.Config.DataSettings.DatabaseData.Config.Verbose,
			Driver:  request.Config.DataSettings.DatabaseData.Config.Driver,
			ConnectionDetails: drivers.ConnectionDetails{
				Host:     request.Config.DataSettings.DatabaseData.Config.Config.Host,
				Port:     uint16(request.Config.DataSettings.DatabaseData.Config.Config.Port),
				Username: request.Config.DataSettings.DatabaseData.Config.Config.UserName,
				Password: request.Config.DataSettings.DatabaseData.Config.Config.Password,
				Database: request.Config.DataSettings.DatabaseData.Config.Config.Database,
				SSLMode:  request.Config.DataSettings.DatabaseData.Config.Config.SslMode,
			},
		}
		dbData = &config.DatabaseData{
			StartDate:        request.Config.DataSettings.DatabaseData.StartDate.AsTime(),
			EndDate:          request.Config.DataSettings.DatabaseData.EndDate.AsTime(),
			Path:             request.Config.DataSettings.DatabaseData.Path,
			Config:           cfg,
			InclusiveEndDate: request.Config.DataSettings.DatabaseData.InclusiveEndDate,
		}
	}
	var liveData *config.LiveData
	if request.Config.DataSettings.LiveData != nil {
		liveData = &config.LiveData{
			APIKeyOverride:        request.Config.DataSettings.LiveData.ApiKeyOverride,
			APISecretOverride:     request.Config.DataSettings.LiveData.ApiSecretOverride,
			APIClientIDOverride:   request.Config.DataSettings.LiveData.ApiClientIdOverride,
			API2FAOverride:        request.Config.DataSettings.LiveData.Api_2FaOverride,
			APISubAccountOverride: request.Config.DataSettings.LiveData.ApiSubAccountOverride,
			RealOrders:            request.Config.DataSettings.LiveData.UseRealOrders,
		}
	}
	var csvData *config.CSVData
	if request.Config.DataSettings.CsvData != nil {
		csvData = &config.CSVData{
			FullPath: request.Config.DataSettings.CsvData.Path,
		}
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
		CurrencySettings: configSettings,
		DataSettings: config.DataSettings{
			Interval:     gctkline.Interval(request.Config.DataSettings.Interval),
			DataType:     request.Config.DataSettings.Datatype,
			APIData:      apiData,
			DatabaseData: dbData,
			LiveData:     liveData,
			CSVData:      csvData,
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

	err = ExecuteStrategy(cfg, s.BacktesterConfig)
	if err != nil {
		return nil, err
	}
	return &btrpc.ExecuteStrategyResponse{
		Success: true,
	}, nil
}
