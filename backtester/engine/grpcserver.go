package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	gctengine "github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

var (
	errBadPort             = errors.New("received bad port")
	errCannotHandleRequest = errors.New("cannot handle request")
)

// GRPCServer struct
type GRPCServer struct {
	btrpc.BacktesterServiceServer
	config  *config.BacktesterConfig
	manager *RunManager
}

// SetupRPCServer sets up the gRPC server
func SetupRPCServer(cfg *config.BacktesterConfig, manager *RunManager) (*GRPCServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w backtester config", common.ErrNilArguments)
	}
	if manager == nil {
		return nil, fmt.Errorf("%w run manager", common.ErrNilArguments)
	}
	return &GRPCServer{
		config:  cfg,
		manager: manager,
	}, nil
}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer(server *GRPCServer) error {
	targetDir := utils.GetTLSDir(server.config.GRPC.TLSDir)
	if err := gctengine.CheckCerts(targetDir); err != nil {
		return err
	}
	log.Debugf(log.GRPCSys, "Backtester GRPC server enabled. Starting GRPC server on https://%v.\n", server.config.GRPC.ListenAddress)
	lis, err := net.Listen("tcp", server.config.GRPC.ListenAddress)
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
	btrpc.RegisterBacktesterServiceServer(s, server)

	go func() {
		if err = s.Serve(lis); err != nil {
			log.Error(log.GRPCSys, err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "GRPC server started!")

	if server.config.GRPC.GRPCProxyEnabled {
		return server.StartRPCRESTProxy()
	}
	return nil
}

// StartRPCRESTProxy starts a gRPC proxy
func (s *GRPCServer) StartRPCRESTProxy() error {
	log.Debugf(log.GRPCSys, "GRPC proxy server support enabled. Starting gRPC proxy server on http://%v.\n", s.config.GRPC.GRPCProxyListenAddress)
	targetDir := utils.GetTLSDir(s.config.GRPC.TLSDir)
	creds, err := credentials.NewClientTLSFromFile(filepath.Join(targetDir, "cert.pem"), "")
	if err != nil {
		return fmt.Errorf("unabled to start gRPC proxy. Err: %w", err)
	}

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: s.config.GRPC.Username,
			Password: s.config.GRPC.Password,
		}),
	}
	err = btrpc.RegisterBacktesterServiceHandlerFromEndpoint(context.Background(),
		mux, s.config.GRPC.ListenAddress, opts)
	if err != nil {
		return fmt.Errorf("failed to register gRPC proxy. Err: %w", err)
	}

	go func() {
		server := &http.Server{
			Addr:        s.config.GRPC.GRPCProxyListenAddress,
			ReadTimeout: time.Minute,
		}

		if err = server.ListenAndServe(); err != nil {
			log.Errorf(log.GRPCSys, "GRPC proxy failed to server: %s\n", err)
		}
	}()

	log.Debug(log.GRPCSys, "GRPC proxy server started!")
	return nil
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

	if username != s.config.GRPC.Username ||
		password != s.config.GRPC.Password {
		return ctx, fmt.Errorf("username/password mismatch")
	}
	return ctx, nil
}

// convertSummary converts a run summary into a RPC format
func convertSummary(run *RunSummary) *btrpc.RunSummary {
	runSummary := &btrpc.RunSummary{
		Id:           run.MetaData.ID.String(),
		StrategyName: run.MetaData.Strategy,
		Closed:       run.MetaData.Closed,
		LiveTesting:  run.MetaData.LiveTesting,
		RealOrders:   run.MetaData.RealOrders,
	}
	if !run.MetaData.DateStarted.IsZero() {
		runSummary.DateStarted = run.MetaData.DateStarted.Format(gctcommon.SimpleTimeFormatWithTimezone)
	}
	if !run.MetaData.DateLoaded.IsZero() {
		runSummary.DateLoaded = run.MetaData.DateLoaded.Format(gctcommon.SimpleTimeFormatWithTimezone)
	}
	if !run.MetaData.DateEnded.IsZero() {
		runSummary.DateEnded = run.MetaData.DateEnded.Format(gctcommon.SimpleTimeFormatWithTimezone)
	}
	return runSummary
}

// ExecuteStrategyFromFile will backtest a strategy from the filepath provided
func (s *GRPCServer) ExecuteStrategyFromFile(_ context.Context, request *btrpc.ExecuteStrategyFromFileRequest) (*btrpc.ExecuteStrategyResponse, error) {
	if s.config == nil {
		return nil, fmt.Errorf("%w server config", gctcommon.ErrNilPointer)
	}
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	if request == nil {
		return nil, fmt.Errorf("%w nil request", gctcommon.ErrNilPointer)
	}
	if request.DoNotRunImmediately && request.DoNotStore {
		return nil, fmt.Errorf("%w cannot manage a run with both dnr and dns", errCannotHandleRequest)
	}

	dir := request.StrategyFilePath
	cfg, err := config.ReadStrategyConfigFromFile(dir)
	if err != nil {
		return nil, err
	}

	err = cfg.Validate()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		err = fmt.Errorf("%w backtester config", common.ErrNilArguments)
		return nil, err
	}

	if !s.config.Report.GenerateReport {
		s.config.Report.OutputPath = ""
		s.config.Report.TemplatePath = ""
	}

	bt, err := NewFromConfig(cfg, s.config.Report.TemplatePath, s.config.Report.OutputPath, s.config.Verbose)
	if err != nil {
		return nil, err
	}

	if !request.DoNotStore {
		err = s.manager.AddRun(bt)
		if err != nil {
			return nil, err
		}
	}

	if !request.DoNotRunImmediately {
		err = bt.ExecuteStrategy(false)
		if err != nil {
			return nil, err
		}
	}
	btSum, err := bt.GenerateSummary()
	if err != nil {
		return nil, err
	}
	return &btrpc.ExecuteStrategyResponse{
		Run: convertSummary(btSum),
	}, nil
}

// ExecuteStrategyFromConfig will backtest a strategy config built from a GRPC command
// this should be a preferred method of interacting with backtester, as it allows for very quick
// minor tweaks to strategy to determine the best result - SO LONG AS YOU DONT OVERFIT
func (s *GRPCServer) ExecuteStrategyFromConfig(_ context.Context, request *btrpc.ExecuteStrategyFromConfigRequest) (*btrpc.ExecuteStrategyResponse, error) {
	if s.config == nil {
		return nil, fmt.Errorf("%w server config", gctcommon.ErrNilPointer)
	}
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	if request == nil || request.Config == nil {
		return nil, fmt.Errorf("%w nil request", gctcommon.ErrNilPointer)
	}
	if request.DoNotRunImmediately && request.DoNotStore {
		return nil, fmt.Errorf("%w cannot manage a run with both dnr and dns", errCannotHandleRequest)
	}

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
			UseExchangePNLCalculation:     request.Config.CurrencySettings[i].UseExchangePnlCalculation,
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
		if request.Config.DataSettings.DatabaseData.Config.Config.Port > math.MaxUint16 {
			return nil, fmt.Errorf("%w '%v' cannot exceed '%v'", errBadPort, request.Config.DataSettings.DatabaseData.Config.Config.Port, math.MaxUint16)
		}
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
			// TODO FIXXXX
			NewEventTimeout:           0,
			DataCheckTimer:            0,
			RealOrders:                request.Config.DataSettings.LiveData.UseRealOrders,
			ClosePositionsOnExit:      false,
			DataRequestRetryTolerance: 0,
			DataRequestRetryWaitTime:  0,
			ExchangeCredentials:       nil,
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
			DisableUSDTracking:           request.Config.StrategySettings.DisableUsdTracking,
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

	if !s.config.Report.GenerateReport {
		s.config.Report.OutputPath = ""
		s.config.Report.TemplatePath = ""
	}

	bt, err := NewFromConfig(cfg, s.config.Report.TemplatePath, s.config.Report.OutputPath, s.config.Verbose)
	if err != nil {
		return nil, err
	}

	if !request.DoNotStore {
		err = s.manager.AddRun(bt)
		if err != nil {
			return nil, err
		}
	}

	if !request.DoNotRunImmediately {
		err = bt.ExecuteStrategy(false)
		if err != nil {
			return nil, err
		}
	}
	btSum, err := bt.GenerateSummary()
	if err != nil {
		return nil, err
	}
	return &btrpc.ExecuteStrategyResponse{
		Run: convertSummary(btSum),
	}, nil
}

// ListAllRuns returns all backtesting/livestrategy runs managed by the server
func (s *GRPCServer) ListAllRuns(_ context.Context, _ *btrpc.ListAllRunsRequest) (*btrpc.ListAllRunsResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	list, err := s.manager.List()
	if err != nil {
		return nil, err
	}
	response := make([]*btrpc.RunSummary, len(list))
	for i := range list {
		response[i] = convertSummary(list[i])
	}
	return &btrpc.ListAllRunsResponse{
		Runs: response,
	}, nil
}

// StopRun stops a backtest/livestrategy run in its tracks
func (s *GRPCServer) StopRun(_ context.Context, req *btrpc.StopRunRequest) (*btrpc.StopRunResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	if req == nil {
		return nil, fmt.Errorf("%w StopRunRequest", gctcommon.ErrNilPointer)
	}
	id, err := uuid.FromString(req.Id)
	if err != nil {
		return nil, err
	}
	run, err := s.manager.GetSummary(id)
	if err != nil {
		return nil, err
	}
	err = s.manager.StopRun(id)
	if err != nil {
		return nil, err
	}
	return &btrpc.StopRunResponse{
		StoppedRun: convertSummary(run),
	}, nil
}

// StopAllRuns stops all backtest/livestrategy runs in its tracks
func (s *GRPCServer) StopAllRuns(_ context.Context, _ *btrpc.StopAllRunsRequest) (*btrpc.StopAllRunsResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	stopped, err := s.manager.StopAllRuns()
	if err != nil {
		return nil, err
	}

	stoppedRuns := make([]*btrpc.RunSummary, len(stopped))
	for i := range stopped {
		stoppedRuns[i] = convertSummary(stopped[i])
	}
	return &btrpc.StopAllRunsResponse{
		RunsStopped: stoppedRuns,
	}, nil
}

// StartRun starts a backtest/livestrategy that was set to not start automatically
func (s *GRPCServer) StartRun(_ context.Context, req *btrpc.StartRunRequest) (*btrpc.StartRunResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	if req == nil {
		return nil, fmt.Errorf("%w StartRunRequest", gctcommon.ErrNilPointer)
	}
	id, err := uuid.FromString(req.Id)
	if err != nil {
		return nil, err
	}
	err = s.manager.StartRun(id)
	if err != nil {
		return nil, err
	}
	return &btrpc.StartRunResponse{
		Started: true,
	}, nil
}

// StartAllRuns starts all backtest/livestrategy runs
func (s *GRPCServer) StartAllRuns(_ context.Context, _ *btrpc.StartAllRunsRequest) (*btrpc.StartAllRunsResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	started, err := s.manager.StartAllRuns()
	if err != nil {
		return nil, err
	}

	startedRuns := make([]string, len(started))
	for i := range started {
		startedRuns[i] = started[i].String()
	}
	return &btrpc.StartAllRunsResponse{
		RunsStarted: startedRuns,
	}, nil
}

// ClearRun removes a run from memory, but only if it is not running
func (s *GRPCServer) ClearRun(_ context.Context, req *btrpc.ClearRunRequest) (*btrpc.ClearRunResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	if req == nil {
		return nil, fmt.Errorf("%w ClearRunRequest", gctcommon.ErrNilPointer)
	}
	id, err := uuid.FromString(req.Id)
	if err != nil {
		return nil, err
	}
	run, err := s.manager.GetSummary(id)
	if err != nil {
		return nil, err
	}
	err = s.manager.ClearRun(id)
	if err != nil {
		return nil, err
	}
	return &btrpc.ClearRunResponse{
		ClearedRun: convertSummary(run),
	}, nil
}

// ClearAllRuns removes all runs from memory, but only if they are not running
func (s *GRPCServer) ClearAllRuns(_ context.Context, _ *btrpc.ClearAllRunsRequest) (*btrpc.ClearAllRunsResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w run manager", gctcommon.ErrNilPointer)
	}
	clearedRuns, remainingRuns, err := s.manager.ClearAllRuns()
	if err != nil {
		return nil, err
	}

	clearedResponse := make([]*btrpc.RunSummary, len(clearedRuns))
	for i := range clearedRuns {
		clearedResponse[i] = convertSummary(clearedRuns[i])
	}
	remainingResponse := make([]*btrpc.RunSummary, len(remainingRuns))
	for i := range remainingRuns {
		remainingResponse[i] = convertSummary(remainingRuns[i])
	}
	return &btrpc.ClearAllRunsResponse{
		ClearedRuns:   clearedResponse,
		RemainingRuns: remainingResponse,
	}, nil
}
