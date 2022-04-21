package config

import gctconfig "github.com/thrasher-corp/gocryptotrader/config"

type BacktesterConfig struct {
	Verbose       bool       `json:"verbose"`
	LogSubheaders bool       `json:"log-subheaders"`
	Report        Report     `json:"report"`
	GRPC          GRPC       `json:"grpc"`
	Colours       CMDColours `json:"cmd-colours"`
}

type Report struct {
	OutputReport bool   `json:"output-report"`
	TemplatePath string `json:"template-path"`
	OutputPath   string `json:"output-path"`
	DarkMode     bool   `json:"dark-mode"`
}

type GRPC struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	RPCConfig gctconfig.GRPCConfig
}

type CMDColours struct {
	UseCMDColours bool   `json:"use-cmd-colours"`
	Default       string `json:"default"`
	Green         string `json:"green"`
	White         string `json:"white"`
	Grey          string `json:"grey"`
	DarkGrey      string `json:"dark-grey"`
	H1            string `json:"h1"`
	H2            string `json:"h2"`
	H3            string `json:"h3"`
	H4            string `json:"h4"`
	Success       string `json:"success"`
	Info          string `json:"info"`
	Debug         string `json:"debug"`
	Warn          string `json:"warn"`
	Problem       string `json:"problem"`
	Error         string `json:"error"`
}
