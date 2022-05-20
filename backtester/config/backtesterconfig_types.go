package config

import (
	"path/filepath"
	"runtime"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
)

var (
	DefaultBTDir       = filepath.Join(gctcommon.GetDefaultDataDir(runtime.GOOS), "backtester")
	DefaultBTConfigDir = filepath.Join(DefaultBTDir, "config.json")
)

type BacktesterConfig struct {
	PrintLogo               bool           `json:"print-logo"`
	Verbose                 bool           `json:"verbose"`
	LogSubheaders           bool           `json:"log-subheaders"`
	SingleRun               bool           `json:"single-run"`
	SingleRunStrategyConfig string         `json:"single-run-strategy-config"`
	Report                  Report         `json:"report"`
	GRPC                    GRPC           `json:"grpc"`
	UseCMDColours           bool           `json:"use-cmd-colours"`
	Colours                 common.Colours `json:"cmd-colours"`
}

type Report struct {
	GenerateReport bool   `json:"output-report"`
	TemplatePath   string `json:"template-path"`
	OutputPath     string `json:"output-path"`
	DarkMode       bool   `json:"dark-mode"`
}

type GRPC struct {
	Username string `json:"username"`
	Password string `json:"password"`
	gctconfig.GRPCConfig
	TLSDir string `json:"tls-dir"`
}
