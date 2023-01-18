package config

import (
	"path/filepath"
	"runtime"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
)

var (
	// DefaultBTDir is the default backtester config directory
	DefaultBTDir = filepath.Join(gctcommon.GetDefaultDataDir(runtime.GOOS), "backtester")
	// DefaultBTConfigDir is the default backtester config file
	DefaultBTConfigDir = filepath.Join(DefaultBTDir, "config.json")
)

// BacktesterConfig contains the configuration for the backtester
type BacktesterConfig struct {
	PrintLogo           bool           `json:"print-logo"`
	LogSubheaders       bool           `json:"log-subheaders"`
	Verbose             bool           `json:"verbose"`
	StopAllTasksOnClose bool           `json:"stop-all-tasks-on-close"`
	PluginPath          string         `json:"plugin-path"`
	Report              Report         `json:"report"`
	GRPC                GRPC           `json:"grpc"`
	UseCMDColours       bool           `json:"use-cmd-colours"`
	Colours             common.Colours `json:"cmd-colours"`
}

// Report contains the report settings
type Report struct {
	GenerateReport bool   `json:"output-report"`
	TemplatePath   string `json:"template-path"`
	OutputPath     string `json:"output-path"`
	DarkMode       bool   `json:"dark-mode"`
}

// GRPC holds the GRPC configuration
type GRPC struct {
	Username string `json:"username"`
	Password string `json:"password"`
	gctconfig.GRPCConfig
	TLSDir string `json:"tls-dir"`
}
