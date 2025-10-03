package engine

import (
	"context"
	"encoding/base64"
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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	gctengine "github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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
	manager *TaskManager
}

// SetupRPCServer sets up the gRPC server
func SetupRPCServer(cfg *config.BacktesterConfig, manager *TaskManager) (*GRPCServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w backtester config", gctcommon.ErrNilPointer)
	}
	if manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
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
	lis, err := net.Listen("tcp", server.config.GRPC.ListenAddress) //nolint:noctx // TODO: #2006 Replace net.Listen with (*net.ListenConfig).Listen
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
		grpc.StreamInterceptor(grpcauth.StreamServerInterceptor(server.authenticateClient)),
	}
	s := grpc.NewServer(opts...)
	btrpc.RegisterBacktesterServiceServer(s, server)

	go func() {
		if err = s.Serve(lis); err != nil {
			log.Errorln(log.GRPCSys, err)
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
	log.Debugf(log.GRPCSys, "GRPC proxy server support enabled. Starting gRPC proxy server on %v\n", s.config.GRPC.GRPCProxyListenAddress)
	targetDir := utils.GetTLSDir(s.config.GRPC.TLSDir)
	creds, err := credentials.NewClientTLSFromFile(filepath.Join(targetDir, "cert.pem"), "")
	if err != nil {
		return fmt.Errorf("unable to start gRPC proxy. Err: %w", err)
	}

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
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
			Addr:              s.config.GRPC.GRPCProxyListenAddress,
			ReadHeaderTimeout: time.Minute,
			ReadTimeout:       time.Minute,
		}

		if err = server.ListenAndServe(); err != nil {
			log.Errorf(log.GRPCSys, "GRPC proxy failed to server: %s\n", err)
		}
	}()

	log.Debugln(log.GRPCSys, "GRPC proxy server started!")
	return nil
}

func (s *GRPCServer) authenticateClient(ctx context.Context) (context.Context, error) {
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

	creds := strings.Split(string(decoded), ":")
	username := creds[0]
	password := creds[1]

	if username != s.config.GRPC.Username ||
		password != s.config.GRPC.Password {
		return ctx, errors.New("username/password mismatch")
	}
	return ctx, nil
}

// convertSummary converts a task summary into a RPC format
func convertSummary(task *TaskSummary) *btrpc.TaskSummary {
	taskSummary := &btrpc.TaskSummary{
		Id:           task.MetaData.ID.String(),
		StrategyName: task.MetaData.Strategy,
		Closed:       task.MetaData.Closed,
		LiveTesting:  task.MetaData.LiveTesting,
		RealOrders:   task.MetaData.RealOrders,
	}
	if !task.MetaData.DateStarted.IsZero() {
		taskSummary.DateStarted = task.MetaData.DateStarted.Format(gctcommon.SimpleTimeFormatWithTimezone)
	}
	if !task.MetaData.DateLoaded.IsZero() {
		taskSummary.DateLoaded = task.MetaData.DateLoaded.Format(gctcommon.SimpleTimeFormatWithTimezone)
	}
	if !task.MetaData.DateEnded.IsZero() {
		taskSummary.DateEnded = task.MetaData.DateEnded.Format(gctcommon.SimpleTimeFormatWithTimezone)
	}
	return taskSummary
}

// ExecuteStrategyFromFile will backtest a strategy from the filepath provided
func (s *GRPCServer) ExecuteStrategyFromFile(_ context.Context, request *btrpc.ExecuteStrategyFromFileRequest) (*btrpc.ExecuteStrategyResponse, error) {
	if s.config == nil {
		return nil, fmt.Errorf("%w server config", gctcommon.ErrNilPointer)
	}
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	if request == nil {
		return nil, fmt.Errorf("%w request", gctcommon.ErrNilPointer)
	}
	if request.DoNotRunImmediately && request.DoNotStore {
		return nil, fmt.Errorf("%w cannot manage a task with both dnr and dns", errCannotHandleRequest)
	}

	dir := request.StrategyFilePath
	cfg, err := config.ReadStrategyConfigFromFile(dir)
	if err != nil {
		return nil, err
	}

	if io := request.IntervalOverride.AsDuration(); io > 0 {
		if io < gctkline.FifteenSecond.Duration() {
			return nil, fmt.Errorf("%w, interval must be >= 15 seconds, received '%v'", gctkline.ErrInvalidInterval, io)
		}
		cfg.DataSettings.Interval = gctkline.Interval(io)
	}

	if startTime := request.StartTimeOverride.AsTime(); startTime.Unix() != 0 && !startTime.IsZero() {
		if cfg.DataSettings.DatabaseData != nil {
			cfg.DataSettings.DatabaseData.StartDate = startTime
		} else if cfg.DataSettings.APIData != nil {
			cfg.DataSettings.APIData.StartDate = startTime
		}
	}
	if endTime := request.EndTimeOverride.AsTime(); endTime.Unix() != 0 && !endTime.IsZero() {
		if cfg.DataSettings.DatabaseData != nil {
			cfg.DataSettings.DatabaseData.EndDate = endTime
		} else if cfg.DataSettings.APIData != nil {
			cfg.DataSettings.APIData.EndDate = endTime
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if !s.config.Report.GenerateReport {
		s.config.Report.OutputPath = ""
		s.config.Report.TemplatePath = ""
	}

	bt, err := NewBacktesterFromConfigs(cfg, s.config)
	if err != nil {
		return nil, err
	}

	if !request.DoNotStore {
		err = s.manager.AddTask(bt)
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
		Task: convertSummary(btSum),
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
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	if request == nil || request.Config == nil {
		return nil, fmt.Errorf("%w request", gctcommon.ErrNilPointer)
	}
	if request.DoNotRunImmediately && request.DoNotStore {
		return nil, fmt.Errorf("%w cannot manage a task with both dnr and dns", errCannotHandleRequest)
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

	customSettings := make(map[string]any, len(request.Config.StrategySettings.CustomSettings))
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
				Port:     request.Config.DataSettings.DatabaseData.Config.Config.Port,
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
		creds := make([]config.Credentials, len(request.Config.DataSettings.LiveData.Credentials))
		for i := range request.Config.DataSettings.LiveData.Credentials {
			creds[i] = config.Credentials{
				Exchange: request.Config.DataSettings.LiveData.Credentials[i].Exchange,
				Keys: accounts.Credentials{
					Key:             request.Config.DataSettings.LiveData.Credentials[i].Keys.Key,
					Secret:          request.Config.DataSettings.LiveData.Credentials[i].Keys.Secret,
					ClientID:        request.Config.DataSettings.LiveData.Credentials[i].Keys.ClientId,
					PEMKey:          request.Config.DataSettings.LiveData.Credentials[i].Keys.PemKey,
					SubAccount:      request.Config.DataSettings.LiveData.Credentials[i].Keys.SubAccount,
					OneTimePassword: request.Config.DataSettings.LiveData.Credentials[i].Keys.OneTimePassword,
				},
			}
		}
		liveData = &config.LiveData{
			NewEventTimeout:           time.Duration(request.Config.DataSettings.LiveData.NewEventTimeout),
			DataCheckTimer:            time.Duration(request.Config.DataSettings.LiveData.DataCheckTimer),
			RealOrders:                request.Config.DataSettings.LiveData.RealOrders,
			ClosePositionsOnStop:      request.Config.DataSettings.LiveData.ClosePositionsOnStop,
			DataRequestRetryTolerance: request.Config.DataSettings.LiveData.DataRequestRetryTolerance,
			DataRequestRetryWaitTime:  time.Duration(request.Config.DataSettings.LiveData.DataRequestRetryWaitTime),
			ExchangeCredentials:       creds,
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
			Interval:     gctkline.Interval(request.Config.DataSettings.Interval.AsDuration()),
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

	bt, err := NewBacktesterFromConfigs(cfg, s.config)
	if err != nil {
		return nil, err
	}

	if !request.DoNotStore {
		err = s.manager.AddTask(bt)
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
		Task: convertSummary(btSum),
	}, nil
}

// ListAllTasks returns all strategy tasks managed by the server
func (s *GRPCServer) ListAllTasks(_ context.Context, _ *btrpc.ListAllTasksRequest) (*btrpc.ListAllTasksResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	list, err := s.manager.List()
	if err != nil {
		return nil, err
	}
	response := make([]*btrpc.TaskSummary, len(list))
	for i := range list {
		response[i] = convertSummary(list[i])
	}
	return &btrpc.ListAllTasksResponse{
		Tasks: response,
	}, nil
}

// StopTask stops a strategy task in its tracks
func (s *GRPCServer) StopTask(_ context.Context, req *btrpc.StopTaskRequest) (*btrpc.StopTaskResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	if req == nil {
		return nil, fmt.Errorf("%w StopTaskRequest", gctcommon.ErrNilPointer)
	}
	id, err := uuid.FromString(req.Id)
	if err != nil {
		return nil, err
	}
	task, err := s.manager.GetSummary(id)
	if err != nil {
		return nil, err
	}
	err = s.manager.StopTask(id)
	if err != nil {
		return nil, err
	}
	return &btrpc.StopTaskResponse{
		StoppedTask: convertSummary(task),
	}, nil
}

// StopAllTasks stops all strategy tasks in its tracks
func (s *GRPCServer) StopAllTasks(_ context.Context, _ *btrpc.StopAllTasksRequest) (*btrpc.StopAllTasksResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	stopped, err := s.manager.StopAllTasks()
	if err != nil {
		return nil, err
	}

	stoppedTasks := make([]*btrpc.TaskSummary, len(stopped))
	for i := range stopped {
		stoppedTasks[i] = convertSummary(stopped[i])
	}
	return &btrpc.StopAllTasksResponse{
		TasksStopped: stoppedTasks,
	}, nil
}

// StartTask starts a strategy that was set to not start automatically
func (s *GRPCServer) StartTask(_ context.Context, req *btrpc.StartTaskRequest) (*btrpc.StartTaskResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	if req == nil {
		return nil, fmt.Errorf("%w StartTaskRequest", gctcommon.ErrNilPointer)
	}
	id, err := uuid.FromString(req.Id)
	if err != nil {
		return nil, err
	}
	err = s.manager.StartTask(id)
	if err != nil {
		return nil, err
	}
	return &btrpc.StartTaskResponse{
		Started: true,
	}, nil
}

// StartAllTasks starts all strategy tasks
func (s *GRPCServer) StartAllTasks(_ context.Context, _ *btrpc.StartAllTasksRequest) (*btrpc.StartAllTasksResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	started, err := s.manager.StartAllTasks()
	if err != nil {
		return nil, err
	}

	startedTasks := make([]string, len(started))
	for i := range started {
		startedTasks[i] = started[i].String()
	}
	return &btrpc.StartAllTasksResponse{
		TasksStarted: startedTasks,
	}, nil
}

// ClearTask removes a task from memory, but only if it is not running
func (s *GRPCServer) ClearTask(_ context.Context, req *btrpc.ClearTaskRequest) (*btrpc.ClearTaskResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	if req == nil {
		return nil, fmt.Errorf("%w ClearTaskRequest", gctcommon.ErrNilPointer)
	}
	id, err := uuid.FromString(req.Id)
	if err != nil {
		return nil, err
	}
	task, err := s.manager.GetSummary(id)
	if err != nil {
		return nil, err
	}
	err = s.manager.ClearTask(id)
	if err != nil {
		return nil, err
	}
	return &btrpc.ClearTaskResponse{
		ClearedTask: convertSummary(task),
	}, nil
}

// ClearAllTasks removes all tasks from memory, but only if they are not running
func (s *GRPCServer) ClearAllTasks(_ context.Context, _ *btrpc.ClearAllTasksRequest) (*btrpc.ClearAllTasksResponse, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("%w task manager", gctcommon.ErrNilPointer)
	}
	clearedTasks, remainingTasks, err := s.manager.ClearAllTasks()
	if err != nil {
		return nil, err
	}

	clearedResponse := make([]*btrpc.TaskSummary, len(clearedTasks))
	for i := range clearedTasks {
		clearedResponse[i] = convertSummary(clearedTasks[i])
	}
	remainingResponse := make([]*btrpc.TaskSummary, len(remainingTasks))
	for i := range remainingTasks {
		remainingResponse[i] = convertSummary(remainingTasks[i])
	}
	return &btrpc.ClearAllTasksResponse{
		ClearedTasks:   clearedResponse,
		RemainingTasks: remainingResponse,
	}, nil
}
