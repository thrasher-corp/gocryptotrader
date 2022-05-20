package config

import (
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
)

func GenerateDefaultConfig() (*BacktesterConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &BacktesterConfig{
		PrintLogo:               true,
		SingleRun:               false,
		SingleRunStrategyConfig: filepath.Join(wd, "config", "examples", "ftx-cash-carry.strat"),
		Verbose:                 false,
		LogSubheaders:           true,
		Report: Report{
			GenerateReport: true,
			TemplatePath:   filepath.Join(wd, "report", "tpl.gohtml"),
			OutputPath:     filepath.Join(wd, "results"),
			DarkMode:       false,
		},
		GRPC: GRPC{
			Username: "rpcuser",
			Password: "helloImTheDefaultPassword",
			GRPCConfig: gctconfig.GRPCConfig{
				Enabled:       true,
				ListenAddress: "localhost:42069",
			},
			TLSDir: DefaultBTDir,
		},
		UseCMDColours: true,
		Colours: common.Colours{
			Default:  common.CMDColours.Default,
			Green:    common.CMDColours.Green,
			White:    common.CMDColours.White,
			Grey:     common.CMDColours.Grey,
			DarkGrey: common.CMDColours.DarkGrey,
			H1:       common.CMDColours.H1,
			H2:       common.CMDColours.H2,
			H3:       common.CMDColours.H3,
			H4:       common.CMDColours.H4,
			Success:  common.CMDColours.Success,
			Info:     common.CMDColours.Info,
			Debug:    common.CMDColours.Debug,
			Warn:     common.CMDColours.Warn,
			Error:    common.CMDColours.Error,
		},
	}, nil
}
