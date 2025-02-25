package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// ReadBacktesterConfigFromPath will take a config from a path
func ReadBacktesterConfigFromPath(path string) (*BacktesterConfig, error) {
	if !file.Exists(path) {
		return nil, fmt.Errorf("%w %v", common.ErrFileNotFound, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var resp *BacktesterConfig
	err = json.Unmarshal(data, &resp)
	return resp, err
}

// GenerateDefaultConfig will return the default backtester config
func GenerateDefaultConfig() (*BacktesterConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &BacktesterConfig{
		PrintLogo:     true,
		LogSubheaders: true,
		Report: Report{
			GenerateReport: true,
			TemplatePath:   filepath.Join(wd, "report", "tpl.gohtml"),
			OutputPath:     filepath.Join(wd, "results"),
		},
		GRPC: GRPC{
			Username: "rpcuser",
			Password: "helloImTheDefaultPassword",
			GRPCConfig: gctconfig.GRPCConfig{
				Enabled:       true,
				ListenAddress: "localhost:9054",
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
		StopAllTasksOnClose: true,
	}, nil
}
