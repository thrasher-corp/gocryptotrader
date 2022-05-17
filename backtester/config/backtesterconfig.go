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
			Username: "backtester",
			Password: "helloImTheDefaultPassword",
			GRPCConfig: gctconfig.GRPCConfig{
				Enabled:       true,
				ListenAddress: "localhost:42069",
			},
			TLSDir: DefaultBTDir,
		},
		Colours: CMDColours{
			UseCMDColours: false,
			Default:       common.ColourDefault,
			Green:         common.ColourGreen,
			White:         common.ColourWhite,
			Grey:          common.ColourGrey,
			DarkGrey:      common.ColourDarkGrey,
			H1:            common.ColourH1,
			H2:            common.ColourH2,
			H3:            common.ColourH3,
			H4:            common.ColourH4,
			Success:       common.ColourSuccess,
			Info:          common.ColourInfo,
			Debug:         common.ColourDebug,
			Warn:          common.ColourWarn,
			Error:         common.ColourError,
		},
	}, nil
}
