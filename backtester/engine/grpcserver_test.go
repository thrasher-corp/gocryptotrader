package engine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/binancecashandcarry"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var dcaConfigPath = filepath.Join("..", "config", "strategyexamples", "dca-api-candles.strat")

func TestStartRPCServer(t *testing.T) {
	t.Parallel()

	reserveAddress := func(t *testing.T) string {
		t.Helper()
		listenConfig := net.ListenConfig{}
		portListener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err, "Listen must not error")
		address := portListener.Addr().String()
		require.NoError(t, portListener.Close(), "Close must not error")
		return address
	}
	canDial := func(t *testing.T, address string) bool {
		t.Helper()
		dialer := net.Dialer{Timeout: 100 * time.Millisecond}
		conn, err := dialer.DialContext(t.Context(), "tcp", address)
		if err != nil {
			return false
		}
		assert.NoError(t, conn.Close(), "Close should not error")
		return true
	}
	newServer := func(t *testing.T) *GRPCServer {
		t.Helper()
		cfg, err := config.GenerateDefaultConfig()
		require.NoError(t, err, "GenerateDefaultConfig must not error")
		cfg.GRPC.ListenAddress = reserveAddress(t)
		cfg.GRPC.TLSDir = t.TempDir()
		cfg.GRPC.GRPCProxyEnabled = false
		server, err := SetupRPCServer(cfg, NewTaskManager())
		require.NoError(t, err, "SetupRPCServer must not error")
		return server
	}

	t.Run("nil server", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, StartRPCServer(t.Context(), nil), gctcommon.ErrNilPointer, "StartRPCServer should reject a nil server")
		assert.ErrorIs(t, StartRPCServer(t.Context(), &GRPCServer{}), gctcommon.ErrNilPointer, "StartRPCServer should reject a server without config")
	})

	t.Run("stops listener on context cancel", func(t *testing.T) {
		t.Parallel()
		server := newServer(t)
		rpcCtx, cancel := context.WithCancel(t.Context())
		require.NoError(t, StartRPCServer(rpcCtx, server), "StartRPCServer must not error")
		assert.Eventually(t, func() bool { return canDial(t, server.config.GRPC.ListenAddress) }, 5*time.Second, 100*time.Millisecond, "gRPC listener should accept TCP connections after startup")
		cancel()
		assert.Eventually(t, func() bool { return !canDial(t, server.config.GRPC.ListenAddress) }, 5*time.Second, 100*time.Millisecond, "gRPC listener should close after context cancellation")
	})

	t.Run("proxy failure stops listener", func(t *testing.T) {
		t.Parallel()
		server := newServer(t)
		listenConfig := net.ListenConfig{}
		proxyPortListener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err, "Listen must not error")
		t.Cleanup(func() { assert.NoError(t, proxyPortListener.Close(), "Close should not error") })
		server.config.GRPC.GRPCProxyEnabled = true
		server.config.GRPC.GRPCProxyListenAddress = proxyPortListener.Addr().String()

		err = StartRPCServer(t.Context(), server)
		assert.ErrorContains(t, err, "gRPC proxy listener failed to bind", "StartRPCServer error should include proxy listener bind context")
		assert.False(t, canDial(t, server.config.GRPC.ListenAddress), "gRPC listener should be closed when proxy setup fails")
	})

	t.Run("REST proxy requires credentials over HTTPS", func(t *testing.T) {
		t.Parallel()
		server := newServer(t)
		server.config.GRPC.GRPCProxyEnabled = true
		server.config.GRPC.GRPCProxyListenAddress = reserveAddress(t)
		rpcCtx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		require.NoError(t, StartRPCServer(rpcCtx, server), "StartRPCServer must not error")
		proxyAddress := server.config.GRPC.GRPCProxyListenAddress
		require.Eventually(t, func() bool { return canDial(t, proxyAddress) }, 5*time.Second, 10*time.Millisecond, "proxy must accept TCP connections after startup")

		caCert, err := os.ReadFile(filepath.Join(utils.GetTLSDir(server.config.GRPC.TLSDir), "cert.pem"))
		require.NoError(t, err, "ReadFile must not error")
		roots := x509.NewCertPool()
		require.True(t, roots.AppendCertsFromPEM(caCert), "AppendCertsFromPEM must accept the generated certificate")
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}, Timeout: 10 * time.Second}
		// Serial and in this order, so the valid request follows the refused ones on the same proxy
		for _, tc := range []struct {
			name               string
			username, password string
			wantStatus         int
		}{
			{name: "no credentials", wantStatus: http.StatusUnauthorized},
			{name: "wrong password", username: server.config.GRPC.Username, password: "wrong", wantStatus: http.StatusUnauthorized},
			{name: "valid credentials", username: server.config.GRPC.Username, password: server.config.GRPC.Password, wantStatus: http.StatusOK},
		} {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+proxyAddress+"/v1/listalltasks", http.NoBody)
				require.NoError(t, err, "NewRequestWithContext must not error")
				if tc.username != "" {
					req.SetBasicAuth(tc.username, tc.password)
				}
				resp, err := client.Do(req)
				require.NoError(t, err, "Do must not error")
				assert.Equal(t, tc.wantStatus, resp.StatusCode, "HTTP status code should match")
				assert.NoError(t, resp.Body.Close(), "Close should not error")
			})
		}
	})
}

func TestExecuteStrategyFromFile(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ExecuteStrategyFromFile(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromFile should error correctly with a nil config")

	s.config, err = config.GenerateDefaultConfig()
	require.NoError(t, err, "GenerateDefaultConfig must not error")

	s.config.Report.GenerateReport = false
	_, err = s.ExecuteStrategyFromFile(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromFile should error correctly with a nil task manager")

	s.manager = NewTaskManager()
	_, err = s.ExecuteStrategyFromFile(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromFile should error correctly with a nil request")

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{})
	assert.ErrorIs(t, err, common.ErrFileNotFound)

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath: dcaConfigPath,
	})
	assert.NoError(t, err, "ExecuteStrategyFromFile should not error")

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:  dcaConfigPath,
		StartTimeOverride: timestamppb.New(time.Now()),
		EndTimeOverride:   timestamppb.New(time.Now().Add(-time.Minute)),
	})
	assert.ErrorIs(t, err, gctcommon.ErrStartAfterEnd)

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:  dcaConfigPath,
		StartTimeOverride: timestamppb.New(time.Now().Add(-time.Minute)),
		EndTimeOverride:   timestamppb.New(time.Now()),
		IntervalOverride:  durationpb.New(time.Duration(1)),
	})
	assert.ErrorIs(t, err, gctkline.ErrInvalidInterval)

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:  dcaConfigPath,
		StartTimeOverride: timestamppb.New(time.Now().Add(-time.Hour * 6).Truncate(time.Hour)),
		EndTimeOverride:   timestamppb.New(time.Now().Add(-time.Hour * 2).Truncate(time.Hour)),
		IntervalOverride:  durationpb.New(time.Hour),
	})
	assert.NoError(t, err, "ExecuteStrategyFromFile should not error")

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:    dcaConfigPath,
		DoNotRunImmediately: true,
		DoNotStore:          true,
	})
	assert.ErrorIs(t, err, errCannotHandleRequest)
}

func TestExecuteStrategyFromConfig(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ExecuteStrategyFromConfig(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromConfig should error correctly with a nil config")

	s.config, err = config.GenerateDefaultConfig()
	require.NoError(t, err, "GenerateDefaultConfig must not error")
	_, err = s.ExecuteStrategyFromConfig(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromConfig should error correctly with a nil task manager")

	s.config.Report.GenerateReport = false
	s.manager = NewTaskManager()
	_, err = s.ExecuteStrategyFromConfig(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromConfig should error correctly with a nil request")

	defaultConfig, err := config.ReadStrategyConfigFromFile(dcaConfigPath)
	require.NoError(t, err, "ReadStrategyConfigFromFile must not error")
	customSettings := make([]*btrpc.CustomSettings, len(defaultConfig.StrategySettings.CustomSettings))
	x := 0
	for k, v := range defaultConfig.StrategySettings.CustomSettings {
		customSettings[x] = &btrpc.CustomSettings{
			KeyField: k,
			KeyValue: fmt.Sprintf("%v", v),
		}
		x++
	}

	currencySettings := make([]*btrpc.CurrencySettings, len(defaultConfig.CurrencySettings))
	for i := range defaultConfig.CurrencySettings {
		var sd *btrpc.SpotDetails
		if defaultConfig.CurrencySettings[i].SpotDetails != nil {
			if defaultConfig.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil {
				sd = &btrpc.SpotDetails{
					InitialBaseFunds: defaultConfig.CurrencySettings[i].SpotDetails.InitialBaseFunds.String(),
				}
			}
			if defaultConfig.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil {
				if sd == nil {
					sd = &btrpc.SpotDetails{}
				}
				sd.InitialQuoteFunds = defaultConfig.CurrencySettings[i].SpotDetails.InitialQuoteFunds.String()
			}
		}
		var fd *btrpc.FuturesDetails
		if defaultConfig.CurrencySettings[i].FuturesDetails != nil {
			fd = &btrpc.FuturesDetails{
				Leverage: &btrpc.Leverage{
					CanUseLeverage:                 defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.CanUseLeverage,
					MaximumOrdersWithLeverageRatio: defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio.String(),
					MaximumLeverageRate:            defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrderLeverageRate.String(),
					MaximumCollateralLeverageRate:  defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumCollateralLeverageRate.String(),
				},
			}
		}
		var makerFee, takerFee string
		if defaultConfig.CurrencySettings[i].MakerFee != nil {
			makerFee = defaultConfig.CurrencySettings[i].MakerFee.String()
		}
		if defaultConfig.CurrencySettings[i].TakerFee != nil {
			takerFee = defaultConfig.CurrencySettings[i].TakerFee.String()
		}
		currencySettings[i] = &btrpc.CurrencySettings{
			ExchangeName: defaultConfig.CurrencySettings[i].ExchangeName,
			Asset:        defaultConfig.CurrencySettings[i].Asset.String(),
			Base:         defaultConfig.CurrencySettings[i].Base.String(),
			Quote:        defaultConfig.CurrencySettings[i].Quote.String(),
			BuySide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.CurrencySettings[i].BuySide.MinimumSize.String(),
				MaximumSize:  defaultConfig.CurrencySettings[i].BuySide.MaximumSize.String(),
				MaximumTotal: defaultConfig.CurrencySettings[i].BuySide.MaximumTotal.String(),
			},
			SellSide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.CurrencySettings[i].SellSide.MinimumSize.String(),
				MaximumSize:  defaultConfig.CurrencySettings[i].SellSide.MaximumSize.String(),
				MaximumTotal: defaultConfig.CurrencySettings[i].SellSide.MaximumTotal.String(),
			},
			MinSlippagePercent:        defaultConfig.CurrencySettings[i].MinimumSlippagePercent.String(),
			MaxSlippagePercent:        defaultConfig.CurrencySettings[i].MaximumSlippagePercent.String(),
			MakerFeeOverride:          makerFee,
			TakerFeeOverride:          takerFee,
			MaximumHoldingsRatio:      defaultConfig.CurrencySettings[i].MaximumHoldingsRatio.String(),
			SkipCandleVolumeFitting:   defaultConfig.CurrencySettings[i].SkipCandleVolumeFitting,
			UseExchangeOrderLimits:    defaultConfig.CurrencySettings[i].CanUseExchangeLimits,
			UseExchangePnlCalculation: defaultConfig.CurrencySettings[i].UseExchangePNLCalculation,
			SpotDetails:               sd,
			FuturesDetails:            fd,
		}
	}

	exchangeLevelFunding := make([]*btrpc.ExchangeLevelFunding, len(defaultConfig.FundingSettings.ExchangeLevelFunding))
	for i := range defaultConfig.FundingSettings.ExchangeLevelFunding {
		exchangeLevelFunding[i] = &btrpc.ExchangeLevelFunding{
			ExchangeName: defaultConfig.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
			Asset:        defaultConfig.FundingSettings.ExchangeLevelFunding[i].Asset.String(),
			Currency:     defaultConfig.FundingSettings.ExchangeLevelFunding[i].Currency.String(),
			InitialFunds: defaultConfig.FundingSettings.ExchangeLevelFunding[i].InitialFunds.String(),
			TransferFee:  defaultConfig.FundingSettings.ExchangeLevelFunding[i].TransferFee.String(),
		}
	}

	dataSettings := &btrpc.DataSettings{
		Interval: durationpb.New(defaultConfig.DataSettings.Interval.Duration()),
		Datatype: defaultConfig.DataSettings.DataType,
	}
	if defaultConfig.DataSettings.APIData != nil {
		dataSettings.ApiData = &btrpc.ApiData{
			StartDate:        timestamppb.New(defaultConfig.DataSettings.APIData.StartDate),
			EndDate:          timestamppb.New(defaultConfig.DataSettings.APIData.EndDate),
			InclusiveEndDate: defaultConfig.DataSettings.APIData.InclusiveEndDate,
		}
	}
	if defaultConfig.DataSettings.LiveData != nil {
		creds := make([]*btrpc.Credentials, len(defaultConfig.DataSettings.LiveData.ExchangeCredentials))
		for i := range defaultConfig.DataSettings.LiveData.ExchangeCredentials {
			creds[i] = &btrpc.Credentials{
				Exchange: defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Exchange,
				Keys: &btrpc.ExchangeCredentials{
					Key:             defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.Key,
					Secret:          defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.Secret,
					ClientId:        defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.ClientID,
					PemKey:          defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.PEMKey,
					SubAccount:      defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.SubAccount,
					OneTimePassword: defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.OneTimePassword,
				},
			}
		}
		dataSettings.LiveData = &btrpc.LiveData{
			RealOrders:  defaultConfig.DataSettings.LiveData.RealOrders,
			Credentials: creds,
		}
	}
	if defaultConfig.DataSettings.CSVData != nil {
		dataSettings.CsvData = &btrpc.CSVData{
			Path: defaultConfig.DataSettings.CSVData.FullPath,
		}
	}
	if defaultConfig.DataSettings.DatabaseData != nil {
		dbConnectionDetails := &btrpc.DatabaseConnectionDetails{
			Host:     defaultConfig.DataSettings.DatabaseData.Config.Host,
			Port:     defaultConfig.DataSettings.DatabaseData.Config.Port,
			Password: defaultConfig.DataSettings.DatabaseData.Config.Password,
			Database: defaultConfig.DataSettings.DatabaseData.Config.Database,
			SslMode:  defaultConfig.DataSettings.DatabaseData.Config.SSLMode,
			UserName: defaultConfig.DataSettings.DatabaseData.Config.Username,
		}
		dbConfig := &btrpc.DatabaseConfig{
			Enabled: false,
			Verbose: false,
			Driver:  "",
			Config:  dbConnectionDetails,
		}
		dataSettings.DatabaseData = &btrpc.DatabaseData{
			StartDate:        timestamppb.New(defaultConfig.DataSettings.DatabaseData.StartDate),
			EndDate:          timestamppb.New(defaultConfig.DataSettings.DatabaseData.EndDate),
			Config:           dbConfig,
			Path:             defaultConfig.DataSettings.DatabaseData.Path,
			InclusiveEndDate: defaultConfig.DataSettings.DatabaseData.InclusiveEndDate,
		}
	}

	cfg := &btrpc.Config{
		Nickname: defaultConfig.Nickname,
		Goal:     defaultConfig.Goal,
		StrategySettings: &btrpc.StrategySettings{
			Name:                            defaultConfig.StrategySettings.Name,
			UseSimultaneousSignalProcessing: defaultConfig.StrategySettings.SimultaneousSignalProcessing,
			DisableUsdTracking:              defaultConfig.StrategySettings.DisableUSDTracking,
			CustomSettings:                  customSettings,
		},
		FundingSettings: &btrpc.FundingSettings{
			UseExchangeLevelFunding: defaultConfig.FundingSettings.UseExchangeLevelFunding,
			ExchangeLevelFunding:    exchangeLevelFunding,
		},
		CurrencySettings: currencySettings,
		DataSettings:     dataSettings,
		PortfolioSettings: &btrpc.PortfolioSettings{
			Leverage: &btrpc.Leverage{
				CanUseLeverage:                 defaultConfig.PortfolioSettings.Leverage.CanUseLeverage,
				MaximumOrdersWithLeverageRatio: defaultConfig.PortfolioSettings.Leverage.MaximumOrdersWithLeverageRatio.String(),
				MaximumLeverageRate:            defaultConfig.PortfolioSettings.Leverage.MaximumOrderLeverageRate.String(),
				MaximumCollateralLeverageRate:  defaultConfig.PortfolioSettings.Leverage.MaximumCollateralLeverageRate.String(),
			},
			BuySide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.PortfolioSettings.BuySide.MinimumSize.String(),
				MaximumSize:  defaultConfig.PortfolioSettings.BuySide.MaximumSize.String(),
				MaximumTotal: defaultConfig.PortfolioSettings.BuySide.MaximumTotal.String(),
			},
			SellSide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.PortfolioSettings.SellSide.MinimumSize.String(),
				MaximumSize:  defaultConfig.PortfolioSettings.SellSide.MaximumSize.String(),
				MaximumTotal: defaultConfig.PortfolioSettings.SellSide.MaximumTotal.String(),
			},
		},
		StatisticSettings: &btrpc.StatisticSettings{
			RiskFreeRate: defaultConfig.StatisticSettings.RiskFreeRate.String(),
		},
	}

	base := proto.CloneOf(cfg)

	_, err = s.ExecuteStrategyFromConfig(t.Context(), &btrpc.ExecuteStrategyFromConfigRequest{
		Config: cfg,
	})
	require.NoError(t, err, "ExecuteStrategyFromConfig must not error")
	require.Len(t, s.manager.tasks, 1, "ExecuteStrategyFromConfig must store the task")
	settings, err := s.manager.tasks[0].Exchange.GetCurrencySettings(defaultConfig.CurrencySettings[0].ExchangeName, defaultConfig.CurrencySettings[0].Asset, currency.NewPair(defaultConfig.CurrencySettings[0].Base, defaultConfig.CurrencySettings[0].Quote))
	require.NoError(t, err, "GetCurrencySettings must not error")
	assert.Truef(t, defaultConfig.CurrencySettings[0].MakerFee.Equal(settings.MakerFee), "maker fee should come from the maker fee override, got %s", settings.MakerFee)
	assert.Truef(t, defaultConfig.CurrencySettings[0].TakerFee.Equal(settings.TakerFee), "taker fee should come from the taker fee override, got %s", settings.TakerFee)
	assert.NotEmpty(t, s.config.Report.OutputPath, "ExecuteStrategyFromConfig should not clear the shared report path")

	_, err = s.ExecuteStrategyFromConfig(t.Context(), &btrpc.ExecuteStrategyFromConfigRequest{
		DoNotRunImmediately: true,
		DoNotStore:          true,
		Config:              cfg,
	})
	assert.ErrorIs(t, err, errCannotHandleRequest)

	// coverage test to ensure the rest of the config can successfully be converted
	// this will not have a successful response
	cfg.FundingSettings.UseExchangeLevelFunding = true
	cfg.StrategySettings.UseSimultaneousSignalProcessing = true
	cfg.FundingSettings.ExchangeLevelFunding = append(cfg.FundingSettings.ExchangeLevelFunding, &btrpc.ExchangeLevelFunding{
		ExchangeName: defaultConfig.CurrencySettings[0].ExchangeName,
		Asset:        defaultConfig.CurrencySettings[0].Asset.String(),
		Currency:     defaultConfig.CurrencySettings[0].Base.String(),
		InitialFunds: "1337",
		TransferFee:  "1337",
	})
	cfg.CurrencySettings[0].FuturesDetails = &btrpc.FuturesDetails{Leverage: &btrpc.Leverage{
		CanUseLeverage:                 false,
		MaximumOrdersWithLeverageRatio: "1337",
		MaximumLeverageRate:            "1337",
		MaximumCollateralLeverageRate:  "1337",
	}}
	cfg.DataSettings.DatabaseData = &btrpc.DatabaseData{
		StartDate: timestamppb.New(time.Now()),
		EndDate:   timestamppb.New(time.Now()),
		Config: &btrpc.DatabaseConfig{
			Enabled: false,
			Verbose: false,
			Driver:  "",
			Config:  &btrpc.DatabaseConnectionDetails{},
		},
		Path:             "test",
		InclusiveEndDate: false,
	}
	cfg.DataSettings.LiveData = &btrpc.LiveData{
		Credentials: []*btrpc.Credentials{
			{
				Exchange: "test",
				Keys: &btrpc.ExchangeCredentials{
					Key:             "1",
					Secret:          "2",
					ClientId:        "3",
					PemKey:          "4",
					SubAccount:      "5",
					OneTimePassword: "6",
				},
			},
		},
	}
	cfg.DataSettings.CsvData = &btrpc.CSVData{
		Path: "test",
	}
	for i := range cfg.CurrencySettings {
		cfg.CurrencySettings[i].SpotDetails.InitialQuoteFunds = ""
		cfg.CurrencySettings[i].SpotDetails.InitialBaseFunds = ""
	}
	_, err = s.ExecuteStrategyFromConfig(t.Context(), &btrpc.ExecuteStrategyFromConfigRequest{
		Config: cfg,
	})
	assert.True(t, errors.Is(err, gctcommon.ErrStartEqualsEnd) || errors.Is(err, errAmbiguousDataSource),
		"ExecuteStrategyFromConfig should error")

	for _, tc := range []struct {
		name string
		omit func(*btrpc.Config)
	}{
		{name: "statistic settings", omit: func(c *btrpc.Config) { c.StatisticSettings = nil }},
		{name: "portfolio settings", omit: func(c *btrpc.Config) { c.PortfolioSettings = nil }},
		{name: "portfolio leverage", omit: func(c *btrpc.Config) { c.PortfolioSettings.Leverage = nil }},
		{name: "portfolio buy side", omit: func(c *btrpc.Config) { c.PortfolioSettings.BuySide = nil }},
		{name: "portfolio sell side", omit: func(c *btrpc.Config) { c.PortfolioSettings.SellSide = nil }},
		{name: "funding settings", omit: func(c *btrpc.Config) { c.FundingSettings = nil }},
		{name: "strategy settings", omit: func(c *btrpc.Config) { c.StrategySettings = nil }},
		{name: "currency buy side", omit: func(c *btrpc.Config) { c.CurrencySettings[0].BuySide = nil }},
		{name: "currency sell side", omit: func(c *btrpc.Config) { c.CurrencySettings[0].SellSide = nil }},
		{name: "data settings", omit: func(c *btrpc.Config) { c.DataSettings = nil }},
		{name: "database config", omit: func(c *btrpc.Config) { c.DataSettings.DatabaseData = &btrpc.DatabaseData{} }},
		{name: "database connection details", omit: func(c *btrpc.Config) {
			c.DataSettings.DatabaseData = &btrpc.DatabaseData{Config: &btrpc.DatabaseConfig{}}
		}},
		{name: "live data credential keys", omit: func(c *btrpc.Config) {
			c.DataSettings.LiveData = &btrpc.LiveData{Credentials: []*btrpc.Credentials{{Exchange: "test"}}}
		}},
	} {
		t.Run("omitted "+tc.name, func(t *testing.T) {
			t.Parallel()
			c := proto.CloneOf(base)
			c.DataSettings.Interval = nil // Fails validation before any candle data is fetched
			tc.omit(c)
			_, err := s.ExecuteStrategyFromConfig(t.Context(), &btrpc.ExecuteStrategyFromConfigRequest{Config: c, DoNotRunImmediately: true})
			assert.Error(t, err, "ExecuteStrategyFromConfig should return an error for an incomplete config")
		})
	}
}

func TestListAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ListAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.ListAllTasks(t.Context(), nil)
	assert.NoError(t, err, "ListAllTasks should not error")

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	resp, err := s.ListAllTasks(t.Context(), &btrpc.ListAllTasksRequest{})
	assert.NoError(t, err, "ListAllTasks should not error")
	assert.Len(t, resp.Tasks, 1, "ListAllTasks should return 1 task")
}

func TestGRPCStopTask(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StopTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StopTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt := &BackTest{
		Strategy:   &fakeStrat{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	_, err = s.StopTask(t.Context(), &btrpc.StopTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	assert.ErrorIs(t, err, errTaskHasNotRan)
	require.Len(t, s.manager.tasks, 1, "StopTask must not remove task")

	s.manager.tasks[0].MetaData.DateStarted = time.Now()
	resp, err := s.StopTask(t.Context(), &btrpc.StopTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	require.NoError(t, err, "StopTask must not error")
	assert.False(t, s.manager.tasks[0].MetaData.DateEnded.IsZero(), "DateEnded should not be zero")
	assert.True(t, resp.StoppedTask.Closed, "response should describe the task after it stopped")
	assert.NotEmpty(t, resp.StoppedTask.DateEnded, "response should carry the task's end date")
}

func TestGRPCStopAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StopAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StopAllTasks(t.Context(), nil)
	assert.NoError(t, err, "StopAllTasks should not error")

	bt := &BackTest{
		Strategy:   &fakeStrat{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	resp, err := s.StopAllTasks(t.Context(), &btrpc.StopAllTasksRequest{})
	assert.NoError(t, err, "StopAllTasks should not error")
	require.Len(t, s.manager.tasks, 1, "StopAllTasks must return 1 task")
	assert.Empty(t, resp.TasksStopped, "TasksStopped should be empty")

	s.manager.tasks[0].MetaData.DateStarted = time.Now()
	resp, err = s.StopAllTasks(t.Context(), &btrpc.StopAllTasksRequest{})
	require.NoError(t, err, "StopAllTasks must not error")
	assert.False(t, s.manager.tasks[0].MetaData.DateEnded.IsZero(), "DateEnded should not be zero")
	assert.Len(t, resp.TasksStopped, 1, "TasksStopped should be 1 task")
}

func TestGRPCStartTask(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StartTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StartTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt := &BackTest{
		Strategy:   &fakeStrat{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	_, err = s.StartTask(t.Context(), &btrpc.StartTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	require.NoError(t, err, "StartTask must not error")
	require.Len(t, s.manager.tasks, 1, "StartTask must return 1 task")
	assert.False(t, s.manager.tasks[0].MetaData.DateStarted.IsZero(), "DateStarted should not be zero")
}

func TestGRPCStartAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StartAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StartAllTasks(t.Context(), nil)
	assert.NoError(t, err, "StartAllTasks should not error")

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")

	_, err = s.StartAllTasks(t.Context(), &btrpc.StartAllTasksRequest{})
	require.NoError(t, err, "StartAllTasks must not error")
	require.Len(t, s.manager.tasks, 1, "StartAllTasks must return 1 task")
	assert.False(t, s.manager.tasks[0].MetaData.DateStarted.IsZero(), "DateStarted should not be zero")
}

func TestGRPCClearTask(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ClearTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.ClearTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")

	_, err = s.ClearTask(t.Context(), &btrpc.ClearTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	assert.NoError(t, err, "ClearTask should not error")
	assert.Empty(t, s.manager.tasks, "tasks should be empty")
}

func TestGRPCClearAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ClearAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.ClearAllTasks(t.Context(), nil)
	assert.NoError(t, err, "ClearAllTasks should not error")

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")

	_, err = s.ClearAllTasks(t.Context(), &btrpc.ClearAllTasksRequest{})
	assert.NoError(t, err, "ClearAllTasks should not error")
	assert.Empty(t, s.manager.tasks, "tasks should be empty")
}

func TestRPCStatusCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "unmapped error", err: errors.New("unmapped"), want: codes.Unknown},
		{name: "nil request", err: fmt.Errorf("%w request", gctcommon.ErrNilPointer), want: codes.InvalidArgument},
		{name: "conflicting request flags", err: errCannotHandleRequest, want: codes.InvalidArgument},
		{name: "unknown task", err: errTaskNotFound, want: codes.NotFound},
		{name: "missing strategy file", err: fmt.Errorf("%w %v", common.ErrFileNotFound, "path"), want: codes.NotFound},
		{name: "task has not ran", err: errTaskHasNotRan, want: codes.FailedPrecondition},
		{name: "task is running", err: errTaskIsRunning, want: codes.FailedPrecondition},
		{name: "task already ran", err: errAlreadyRan, want: codes.FailedPrecondition},
		{name: "task cannot be cleared", err: errCannotClear, want: codes.FailedPrecondition},
		{name: "live task only", err: errLiveOnly, want: codes.FailedPrecondition},
		{name: "date unset", err: gctcommon.ErrDateUnset, want: codes.InvalidArgument},
		{name: "start after end", err: gctcommon.ErrStartAfterEnd, want: codes.InvalidArgument},
		{name: "start equals end", err: gctcommon.ErrStartEqualsEnd, want: codes.InvalidArgument},
		{name: "database port", err: errBadPort, want: codes.InvalidArgument},
		{name: "ambiguous data source", err: errAmbiguousDataSource, want: codes.InvalidArgument},
		{name: "no data source", err: errNoDataSource, want: codes.InvalidArgument},
		{name: "interval unset", err: errIntervalUnset, want: codes.InvalidArgument},
		{name: "invalid interval", err: gctkline.ErrInvalidInterval, want: codes.InvalidArgument},
		{name: "unsupported asset", err: asset.ErrNotSupported, want: codes.InvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, rpcStatusCode(tc.err), "rpcStatusCode should return the expected code")
		})
	}
}
