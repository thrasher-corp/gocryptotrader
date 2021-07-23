package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/database"
	dbPSQL "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	yes = "yes"
	y   = "y"
)

var dataOptions = []string{
	"API",
	"CSV",
	"Database",
	"Live",
}

func main() {
	fmt.Print(common.ASCIILogo)
	fmt.Println("Welcome to the config generator!")
	reader := bufio.NewReader(os.Stdin)
	cfg := config.Config{
		StrategySettings: config.StrategySettings{
			Name:                         "",
			SimultaneousSignalProcessing: false,
			CustomSettings:               nil,
		},
		CurrencySettings: []config.CurrencySettings{},
		DataSettings: config.DataSettings{
			Interval:     0,
			DataType:     "",
			APIData:      nil,
			DatabaseData: nil,
			LiveData:     nil,
			CSVData:      nil,
		},
		PortfolioSettings: config.PortfolioSettings{
			Leverage: config.Leverage{},
			BuySide:  config.MinMax{},
			SellSide: config.MinMax{},
		},
		StatisticSettings:        config.StatisticSettings{},
		GoCryptoTraderConfigPath: "",
	}
	fmt.Println("-----Strategy Settings-----")
	var err error
	var strats []strategies.Handler
	firstRun := true
	for err != nil || firstRun {
		firstRun = false
		strats, err = parseStrategySettings(&cfg, reader)
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("-----Exchange Settings-----")
	firstRun = true
	for err != nil || firstRun {
		firstRun = false
		err = parseExchangeSettings(reader, &cfg, strats)
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("-----Portfolio Settings-----")
	firstRun = true
	for err != nil || firstRun {
		firstRun = false
		err = parsePortfolioSettings(reader, &cfg)
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("-----Data Settings-----")
	firstRun = true
	for err != nil || firstRun {
		firstRun = false
		err = parseDataSettings(&cfg, reader)
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("-----Statistics Settings-----")
	firstRun = true
	for err != nil || firstRun {
		firstRun = false
		err = parseStatisticsSettings(&cfg, reader)
		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("-----GoCryptoTrader config Settings-----")
	firstRun = true
	for err != nil || firstRun {
		firstRun = false
		fmt.Printf("Enter the path to the GoCryptoTrader config you wish to use. Leave blank to use \"%v\"\n", gctconfig.DefaultFilePath())
		path := quickParse(reader)
		if path != "" {
			cfg.GoCryptoTraderConfigPath = path
		} else {
			cfg.GoCryptoTraderConfigPath = gctconfig.DefaultFilePath()
		}
		_, err = os.Stat(cfg.GoCryptoTraderConfigPath)
		if err != nil {
			log.Println(err)
		}
	}

	var resp []byte
	resp, err = json.MarshalIndent(cfg, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Write strategy config to file? If no, the output will be on screen y/n")
	yn := quickParse(reader)
	if yn == y || yn == yes {
		var wd string
		wd, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		fn := cfg.StrategySettings.Name
		if cfg.Nickname != "" {
			fn += "-" + cfg.Nickname
		}
		fn += ".strat" // nolint:misspell // its shorthand for strategy
		wd = filepath.Join(wd, fn)
		fmt.Printf("Enter output file. If blank, will output to \"%v\"\n", wd)
		path := quickParse(reader)
		if path == "" {
			path = wd
		}
		err = ioutil.WriteFile(path, resp, 0770)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Print(string(resp))
	}
	log.Println("Config creation complete!")
}

func parseStatisticsSettings(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("Enter the risk free rate. eg 0.03")
	var err error
	cfg.StatisticSettings.RiskFreeRate, err = strconv.ParseFloat(quickParse(reader), 64)
	return err
}

func parseDataSettings(cfg *config.Config, reader *bufio.Reader) error {
	var err error
	fmt.Println("Will you be using \"candle\" or \"trade\" data?")
	cfg.DataSettings.DataType = quickParse(reader)
	if cfg.DataSettings.DataType == common.TradeStr {
		fmt.Println("Trade data will be converted into candles")
	}
	fmt.Println("What candle time interval will you use?")
	cfg.DataSettings.Interval, err = parseKlineInterval(reader)
	if err != nil {
		return err
	}

	fmt.Println("Where will this data be sourced?")
	var choice string
	choice, err = parseDataChoice(reader, len(cfg.CurrencySettings) > 1)
	if err != nil {
		return err
	}
	switch choice {
	case "API":
		err = parseAPI(reader, cfg)
	case "Database":
		err = parseDatabase(reader, cfg)
	case "CSV":
		parseCSV(reader, cfg)
	case "Live":
		parseLive(reader, cfg)
	}
	return err
}

func parsePortfolioSettings(reader *bufio.Reader, cfg *config.Config) error {
	var err error
	fmt.Println("Will there be global portfolio buy-side limits? y/n")
	yn := quickParse(reader)
	if yn == y || yn == yes {
		cfg.PortfolioSettings.BuySide, err = minMaxParse("buy", reader)
		if err != nil {
			return err
		}
	}
	fmt.Println("Will there be global portfolio sell-side limits? y/n")
	yn = quickParse(reader)
	if yn == y || yn == yes {
		cfg.PortfolioSettings.SellSide, err = minMaxParse("sell", reader)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseExchangeSettings(reader *bufio.Reader, cfg *config.Config, strats []strategies.Handler) error {
	var err error
	addCurrency := y
	for strings.Contains(addCurrency, y) {
		var currencySetting *config.CurrencySettings
		currencySetting, err = addCurrencySetting(reader)
		if err != nil {
			return err
		}

		cfg.CurrencySettings = append(cfg.CurrencySettings, *currencySetting)
		fmt.Println("Add another exchange currency setting? y/n")
		addCurrency = quickParse(reader)
	}

	if len(cfg.CurrencySettings) > 1 {
		for i := range strats {
			if strats[i].Name() == cfg.StrategySettings.Name &&
				strats[i].SupportsSimultaneousProcessing() {
				fmt.Println("Will this strategy use simultaneous processing? y/n")
				yn := quickParse(reader)
				if yn == y || yn == yes {
					cfg.StrategySettings.SimultaneousSignalProcessing = true
				}
				break
			}
		}
	}
	return nil
}

func parseStrategySettings(cfg *config.Config, reader *bufio.Reader) ([]strategies.Handler, error) {
	fmt.Println("Firstly, please select which strategy you wish to use")
	strats := strategies.GetStrategies()
	var strategiesToUse []string
	for i := range strats {
		fmt.Printf("%v. %s\n", i+1, strats[i].Name())
		strategiesToUse = append(strategiesToUse, strats[i].Name())
	}
	var err error
	cfg.StrategySettings.Name, err = parseStratName(quickParse(reader), strategiesToUse)
	if err != nil {
		return nil, err
	}

	fmt.Println("What is the goal of your strategy?")
	cfg.Goal = quickParse(reader)
	fmt.Println("Enter a nickname, it can help distinguish between different configs using the same strategy")
	cfg.Nickname = quickParse(reader)
	fmt.Println("Does this strategy have custom settings? y/n")
	customSettings := quickParse(reader)
	if strings.Contains(customSettings, y) {
		cfg.StrategySettings.CustomSettings = customSettingsLoop(reader)
	}
	return strats, nil
}

func parseAPI(reader *bufio.Reader, cfg *config.Config) error {
	cfg.DataSettings.APIData = &config.APIData{}
	var startDate, endDate, inclusive string
	var err error
	defaultStart := time.Now().Add(-time.Hour * 24 * 365)
	defaultEnd := time.Now()
	fmt.Printf("What is the start date? Leave blank for \"%v\"\n", defaultStart.Format(gctcommon.SimpleTimeFormat))
	startDate = quickParse(reader)
	if startDate != "" {
		cfg.DataSettings.APIData.StartDate, err = time.Parse(startDate, gctcommon.SimpleTimeFormat)
		if err != nil {
			return err
		}
	} else {
		cfg.DataSettings.APIData.StartDate = defaultStart
	}

	fmt.Printf("What is the end date? Leave blank for \"%v\"\n", defaultStart.Format(gctcommon.SimpleTimeFormat))
	endDate = quickParse(reader)
	if endDate != "" {
		cfg.DataSettings.APIData.EndDate, err = time.Parse(endDate, gctcommon.SimpleTimeFormat)
		if err != nil {
			return err
		}
	} else {
		cfg.DataSettings.APIData.EndDate = defaultEnd
	}
	fmt.Println("Is the end date inclusive? y/n")
	inclusive = quickParse(reader)
	cfg.DataSettings.APIData.InclusiveEndDate = inclusive == y || inclusive == yes

	return nil
}

func parseCSV(reader *bufio.Reader, cfg *config.Config) {
	cfg.DataSettings.CSVData = &config.CSVData{}
	fmt.Println("What is path of the CSV file to read?")
	cfg.DataSettings.CSVData.FullPath = quickParse(reader)
}

func parseDatabase(reader *bufio.Reader, cfg *config.Config) error {
	cfg.DataSettings.DatabaseData = &config.DatabaseData{}
	var input string
	var err error
	defaultStart := time.Now().Add(-time.Hour * 24 * 365)
	defaultEnd := time.Now()
	fmt.Printf("What is the start date? Leave blank for \"%v\"\n", defaultStart.Format(gctcommon.SimpleTimeFormat))
	startDate := quickParse(reader)
	if startDate != "" {
		cfg.DataSettings.DatabaseData.StartDate, err = time.Parse(startDate, gctcommon.SimpleTimeFormat)
		if err != nil {
			return err
		}
	} else {
		cfg.DataSettings.DatabaseData.StartDate = defaultStart
	}

	fmt.Printf("What is the end date? Leave blank for \"%v\"\n", defaultStart.Format(gctcommon.SimpleTimeFormat))
	endDate := quickParse(reader)
	if endDate != "" {
		cfg.DataSettings.DatabaseData.EndDate, err = time.Parse(endDate, gctcommon.SimpleTimeFormat)
		if err != nil {
			return err
		}
	} else {
		cfg.DataSettings.DatabaseData.EndDate = defaultEnd
	}
	fmt.Println("Is the end date inclusive? y/n")
	input = quickParse(reader)
	cfg.DataSettings.DatabaseData.InclusiveEndDate = input == y || input == yes

	fmt.Println("Do you wish to override GoCryptoTrader's database config? y/n")
	input = quickParse(reader)
	if input == y || input == yes {
		cfg.DataSettings.DatabaseData.ConfigOverride = &database.Config{
			Enabled: true,
		}
		fmt.Println("Do you want database verbose output? y/n")
		input = quickParse(reader)
		cfg.DataSettings.DatabaseData.ConfigOverride.Verbose = input == y || input == yes

		fmt.Printf("What database driver to use? %v %v or %v\n", database.DBPostgreSQL, database.DBSQLite, database.DBSQLite3)
		cfg.DataSettings.DatabaseData.ConfigOverride.Driver = quickParse(reader)

		fmt.Println("What is the database host?")
		cfg.DataSettings.DatabaseData.ConfigOverride.Host = quickParse(reader)

		fmt.Println("What is the database username?")
		cfg.DataSettings.DatabaseData.ConfigOverride.Username = quickParse(reader)

		fmt.Println("What is the database password? eg 1234")
		cfg.DataSettings.DatabaseData.ConfigOverride.Password = quickParse(reader)

		fmt.Println("What is the database? eg database.db")
		cfg.DataSettings.DatabaseData.ConfigOverride.Database = quickParse(reader)

		if cfg.DataSettings.DatabaseData.ConfigOverride.Driver == database.DBPostgreSQL {
			fmt.Println("What is the database SSLMode? eg disable")
			cfg.DataSettings.DatabaseData.ConfigOverride.SSLMode = quickParse(reader)
		}
		fmt.Println("What is the database Port? eg 1337")
		input = quickParse(reader)
		var port float64
		if input != "" {
			port, err = strconv.ParseFloat(input, 64)
			if err != nil {
				return err
			}
		}
		cfg.DataSettings.DatabaseData.ConfigOverride.Port = uint16(port)
		err = database.DB.SetConfig(cfg.DataSettings.DatabaseData.ConfigOverride)
		if err != nil {
			return fmt.Errorf("database failed to set config: %w", err)
		}
		if cfg.DataSettings.DatabaseData.ConfigOverride.Driver == database.DBPostgreSQL {
			_, err = dbPSQL.Connect(cfg.DataSettings.DatabaseData.ConfigOverride)
			if err != nil {
				return fmt.Errorf("database failed to connect: %v", err)
			}
		} else if cfg.DataSettings.DatabaseData.ConfigOverride.Driver == database.DBSQLite ||
			cfg.DataSettings.DatabaseData.ConfigOverride.Driver == database.DBSQLite3 {
			_, err = dbsqlite3.Connect(cfg.DataSettings.DatabaseData.ConfigOverride.Database)
			if err != nil {
				return fmt.Errorf("database failed to connect: %v", err)
			}
		}
	}

	return nil
}

func parseLive(reader *bufio.Reader, cfg *config.Config) {
	cfg.DataSettings.LiveData = &config.LiveData{}
	fmt.Println("Do you wish to use live trading? It's highly recommended that you do not. y/n")
	input := quickParse(reader)
	cfg.DataSettings.LiveData.RealOrders = input == y || input == yes
	if cfg.DataSettings.LiveData.RealOrders {
		fmt.Printf("Do you want to override GoCryptoTrader's API credentials for %s? y/n\n", cfg.CurrencySettings[0].ExchangeName)
		input = quickParse(reader)
		if input == y || input == yes {
			fmt.Println("What is the API key?")
			cfg.DataSettings.LiveData.APIKeyOverride = quickParse(reader)
			fmt.Println("What is the API secret?")
			cfg.DataSettings.LiveData.APISecretOverride = quickParse(reader)
			fmt.Println("What is the Client ID?")
			cfg.DataSettings.LiveData.APIClientIDOverride = quickParse(reader)
			fmt.Println("What is the 2FA seed?")
			cfg.DataSettings.LiveData.API2FAOverride = quickParse(reader)
			fmt.Println("What is the subaccount to use?")
			cfg.DataSettings.LiveData.APISubaccountOverride = quickParse(reader)
		}
	}
}

func parseDataChoice(reader *bufio.Reader, multiCurrency bool) (string, error) {
	if multiCurrency {
		// live trading does not support multiple currencies
		dataOptions = dataOptions[:3]
	}
	for i := range dataOptions {
		fmt.Printf("%v. %s\n", i+1, dataOptions[i])
	}
	response := quickParse(reader)
	num, err := strconv.ParseFloat(response, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(dataOptions) || intNum <= 0 {
			return "", errors.New("unknown option")
		}
		return dataOptions[intNum-1], nil
	}
	for i := range dataOptions {
		if strings.EqualFold(response, dataOptions[i]) {
			return dataOptions[i], nil
		}
	}
	return "", errors.New("unrecognised data option")
}

func parseKlineInterval(reader *bufio.Reader) (time.Duration, error) {
	allCandles := gctkline.SupportedIntervals
	for i := range allCandles {
		fmt.Printf("%v. %s\n", i+1, allCandles[i].Word())
	}
	response := quickParse(reader)
	num, err := strconv.ParseFloat(response, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(allCandles) || intNum <= 0 {
			return 0, errors.New("unknown option")
		}
		return allCandles[intNum-1].Duration(), nil
	}
	for i := range allCandles {
		if strings.EqualFold(response, allCandles[i].Word()) {
			return allCandles[i].Duration(), nil
		}
	}
	return 0, errors.New("unrecognised interval")
}

func parseStratName(name string, strategiesToUse []string) (string, error) {
	num, err := strconv.ParseFloat(name, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(strategiesToUse) || intNum <= 0 {
			return "", errors.New("unknown option")
		}
		return strategiesToUse[intNum-1], nil
	}
	for i := range strategiesToUse {
		if strings.EqualFold(name, strategiesToUse[i]) {
			return strategiesToUse[i], nil
		}
	}
	return "", errors.New("unrecognised strategy")
}

func customSettingsLoop(reader *bufio.Reader) map[string]interface{} {
	resp := make(map[string]interface{})
	customSettingField := "loopTime!"
	for customSettingField != "" {
		fmt.Println("Enter a custom setting name. Enter nothing to stop")
		customSettingField = quickParse(reader)
		if customSettingField != "" {
			fmt.Println("Enter a custom setting value")
			resp[customSettingField] = quickParse(reader)
		}
	}
	return resp
}

func addCurrencySetting(reader *bufio.Reader) (*config.CurrencySettings, error) {
	setting := config.CurrencySettings{
		BuySide:  config.MinMax{},
		SellSide: config.MinMax{},
	}
	fmt.Println("Enter the exchange name. eg Binance")
	setting.ExchangeName = quickParse(reader)

	fmt.Println("Please select an asset")
	supported := asset.Supported()
	for i := range supported {
		fmt.Printf("%v. %s\n", i+1, supported[i])
	}
	response := quickParse(reader)
	num, err := strconv.ParseFloat(response, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(supported) || intNum <= 0 {
			return nil, errors.New("unknown option")
		}
		setting.Asset = supported[intNum-1].String()
	}
	for i := range supported {
		if strings.EqualFold(response, supported[i].String()) {
			setting.Asset = supported[i].String()
		}
	}

	fmt.Println("Enter the currency base. eg BTC")
	setting.Base = quickParse(reader)

	fmt.Println("Enter the currency quote. eg USDT")
	setting.Quote = quickParse(reader)

	fmt.Println("Enter the initial funds. eg 10000")
	parseNum := quickParse(reader)
	if parseNum != "" {
		setting.InitialFunds, err = strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return nil, err
		}
	}

	fmt.Println("Enter the maker-fee. eg 0.001")
	parseNum = quickParse(reader)
	if parseNum != "" {
		setting.MakerFee, err = strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("Enter the taker-fee. eg 0.01")
	parseNum = quickParse(reader)
	if parseNum != "" {
		setting.TakerFee, err = strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return nil, err
		}
	}

	fmt.Println("Will there be buy-side limits? y/n")
	yn := quickParse(reader)
	if yn == y || yn == yes {
		setting.BuySide, err = minMaxParse("buy", reader)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("Will there be sell-side limits? y/n")
	yn = quickParse(reader)
	if yn == y || yn == yes {
		setting.SellSide, err = minMaxParse("sell", reader)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("Will the in-sample data amounts conform to current exchange defined order execution limits? i.e. If amount is 1337.001345 and the step size is 0.01 order amount will be re-adjusted to 1337. y/n")
	yn = quickParse(reader)
	if yn == y || yn == yes {
		setting.CanUseExchangeLimits = true
	}
	fmt.Println("Do you wish to include slippage? y/n")
	yn = quickParse(reader)
	if yn == y || yn == yes {
		fmt.Println("Slippage is randomly determined between the lower and upper bounds.")
		fmt.Println("If the lower bound is 80, then the price can change up to 80% of itself. eg if the price is 100 and the lower bound is 80, then the lowest slipped price is $80")
		fmt.Println("If the upper bound is 100, then the price can be unaffected. A minimum of 80 and a maximum of 100 means that the price will randomly be set between those bounds as a way of emulating slippage")

		fmt.Println("What is the lower bounds of slippage? eg 80")
		setting.MinimumSlippagePercent, err = strconv.ParseFloat(quickParse(reader), 64)
		if err != nil {
			return nil, err
		}

		fmt.Println("What is the upper bounds of slippage? eg 100")
		setting.MaximumSlippagePercent, err = strconv.ParseFloat(quickParse(reader), 64)
		if err != nil {
			return nil, err
		}
	}

	return &setting, nil
}

func minMaxParse(buySell string, reader *bufio.Reader) (config.MinMax, error) {
	resp := config.MinMax{}
	var err error
	fmt.Printf("What is the maximum %s size? eg 1\n", buySell)
	parseNum := quickParse(reader)
	if parseNum != "" {
		resp.MaximumSize, err = strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return resp, err
		}
	}
	fmt.Printf("What is the minimum %s size? eg 0.1\n", buySell)
	parseNum = quickParse(reader)
	if parseNum != "" {
		resp.MinimumSize, err = strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return resp, err
		}
	}
	fmt.Printf("What is the maximum spend %s buy? eg 12000\n", buySell)
	parseNum = quickParse(reader)
	if parseNum != "" {
		resp.MaximumTotal, err = strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}

func quickParse(reader *bufio.Reader) string {
	customSettingField, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	customSettingField = strings.Replace(customSettingField, "\r", "", -1)
	return strings.Replace(customSettingField, "\n", "", -1)
}
