package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	dbPSQL "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	var cfg config.Config
	var err error

	fmt.Println("-----Strategy Settings-----")
	// loop in sections, so that if there is an error,
	// a user only needs to redo that section
	for {
		err = parseStrategySettings(&cfg, reader)
		if err != nil {
			log.Println(err)
		} else {
			break
		}
	}

	fmt.Println("-----Exchange Settings-----")

	for {
		err = parseExchangeSettings(reader, &cfg)
		if err != nil {
			log.Println(err)
		} else {
			break
		}
	}

	fmt.Println("-----Portfolio Settings-----")
	for {
		err = parsePortfolioSettings(reader, &cfg)
		if err != nil {
			log.Println(err)
		} else {
			break
		}
	}

	fmt.Println("-----Data Settings-----")
	for {
		err = parseDataSettings(&cfg, reader)
		if err != nil {
			log.Println(err)
		} else {
			break
		}
	}

	fmt.Println("-----Statistics Settings-----")
	for {
		err = parseStatisticsSettings(&cfg, reader)
		if err != nil {
			log.Println(err)
		} else {
			break
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
		var fp, wd string
		extension := "strat" //nolint:misspell // its shorthand for strategy
		for {
			wd, err = os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Enter output directory. If blank, will default to \"%v\"\n", wd)
			parsedPath := quickParse(reader)
			if parsedPath != "" {
				wd = parsedPath
			}

			fn := cfg.StrategySettings.Name
			if cfg.Nickname != "" {
				fn += "-" + cfg.Nickname
			}
			fn, err = common.GenerateFileName(fn, extension)
			if err != nil {
				log.Printf("Could not write file, please try again. err: %v", err)
				continue
			}
			fmt.Printf("Enter output file. If blank, will default to \"%v\"\n", fn)
			parsedFileName := quickParse(reader)
			if parsedFileName != "" {
				fn, err = common.GenerateFileName(parsedFileName, extension)
				if err != nil {
					log.Printf("Could not write file, please try again. err: %v", err)
					continue
				}
			}
			fp = filepath.Join(wd, fn)
			err = os.WriteFile(fp, resp, file.DefaultPermissionOctal)
			if err != nil {
				log.Printf("Could not write file, please try again. err: %v", err)
				continue
			}
			break
		}
		fmt.Printf("Successfully output strategy to \"%v\"\n", fp)
	} else {
		log.Print(string(resp))
	}
	log.Println("Config creation complete!")
}

func parseStatisticsSettings(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("Enter the risk free rate. eg 0.03")
	rfr, err := strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return err
	}
	cfg.StatisticSettings.RiskFreeRate = decimal.NewFromFloat(rfr)
	return nil
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

func parseExchangeSettings(reader *bufio.Reader, cfg *config.Config) error {
	var err error
	addCurrency := y
	for strings.Contains(addCurrency, y) {
		var currencySetting *config.CurrencySettings
		currencySetting, err = addCurrencySetting(reader, cfg.FundingSettings.UseExchangeLevelFunding)
		if err != nil {
			return err
		}

		cfg.CurrencySettings = append(cfg.CurrencySettings, *currencySetting)
		fmt.Println("Add another exchange currency setting? y/n")
		addCurrency = quickParse(reader)
	}

	return nil
}

func parseStrategySettings(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println("Firstly, please select which strategy you wish to use")
	strats := strategies.GetSupportedStrategies()
	strategiesToUse := make([]string, len(strats))
	for i := range strats {
		fmt.Printf("%v. %s\n", i+1, strats[i].Name())
		strategiesToUse[i] = strats[i].Name()
	}
	var err error
	cfg.StrategySettings.Name, err = parseStratName(quickParse(reader), strategiesToUse)
	if err != nil {
		return err
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
	fmt.Println("Do you wish to have strategy performance tracked against USD? y/n")
	yn := quickParse(reader)
	cfg.StrategySettings.DisableUSDTracking = !strings.Contains(yn, y)
	fmt.Println("Will this strategy use simultaneous processing? y/n")
	yn = quickParse(reader)
	cfg.StrategySettings.SimultaneousSignalProcessing = strings.Contains(yn, y)
	if !cfg.StrategySettings.SimultaneousSignalProcessing {
		return nil
	}
	fmt.Println("Will this strategy be able to share funds at an exchange level? y/n")
	yn = quickParse(reader)
	cfg.FundingSettings.UseExchangeLevelFunding = strings.Contains(yn, y)
	if !cfg.FundingSettings.UseExchangeLevelFunding {
		return nil
	}

	addFunding := y
	for strings.Contains(addFunding, y) {
		fund := config.ExchangeLevelFunding{}
		fmt.Println("What is the exchange name to add funding to?")
		fund.ExchangeName = quickParse(reader)
		fmt.Println("What is the asset to add funding to?")
		supported := asset.Supported()
		for i := range supported {
			fmt.Printf("%v. %s\n", i+1, supported[i])
		}
		response := quickParse(reader)
		num, err := strconv.ParseFloat(response, 64)
		if err == nil {
			intNum := int(num)
			if intNum > len(supported) || intNum <= 0 {
				return errors.New("unknown option")
			}
			fund.Asset = supported[intNum-1]
		} else {
			for i := range supported {
				if strings.EqualFold(response, supported[i].String()) {
					fund.Asset = supported[i]
					break
				}
			}
			if fund.Asset == asset.Empty {
				return errors.New("unrecognised data option")
			}
		}

		fmt.Println("What is the individual currency to add funding to? eg BTC")
		fund.Currency = currency.NewCode(quickParse(reader))
		fmt.Printf("How much funding for %v?\n", fund.Currency)
		fund.InitialFunds, err = decimal.NewFromString(quickParse(reader))
		if err != nil {
			return err
		}

		fmt.Println("If your strategy utilises fund transfer, what is the transfer fee?")
		fee := quickParse(reader)
		if fee != "" {
			fund.TransferFee, err = decimal.NewFromString(fee)
			if err != nil {
				return err
			}
		}
		cfg.FundingSettings.ExchangeLevelFunding = append(cfg.FundingSettings.ExchangeLevelFunding, fund)
		fmt.Println("Add another source of funds? y/n")
		addFunding = quickParse(reader)
	}

	return nil
}

func parseAPI(reader *bufio.Reader, cfg *config.Config) error {
	cfg.DataSettings.APIData = &config.APIData{}
	var startDate, endDate, inclusive string
	var err error
	defaultStart := time.Now().Add(-time.Hour * 24 * 365)
	defaultEnd := time.Now()
	fmt.Printf("What is the start date? Leave blank for \"%v\"\n", defaultStart.Format(time.DateTime))
	startDate = quickParse(reader)
	if startDate != "" {
		cfg.DataSettings.APIData.StartDate, err = time.Parse(time.DateTime, startDate)
		if err != nil {
			return err
		}
	} else {
		cfg.DataSettings.APIData.StartDate = defaultStart
	}

	fmt.Printf("What is the end date? Leave blank for \"%v\"\n", defaultEnd.Format(time.DateTime))
	endDate = quickParse(reader)
	if endDate != "" {
		cfg.DataSettings.APIData.EndDate, err = time.Parse(time.DateTime, endDate)
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
	fmt.Printf("What is the start date? Leave blank for \"%v\"\n", defaultStart.Format(time.DateTime))
	startDate := quickParse(reader)
	if startDate != "" {
		cfg.DataSettings.DatabaseData.StartDate, err = time.Parse(time.DateTime, startDate)
		if err != nil {
			return err
		}
	} else {
		cfg.DataSettings.DatabaseData.StartDate = defaultStart
	}

	fmt.Printf("What is the end date? Leave blank for \"%v\"\n", defaultEnd.Format(time.DateTime))
	if endDate := quickParse(reader); endDate != "" {
		cfg.DataSettings.DatabaseData.EndDate, err = time.Parse(time.DateTime, endDate)
		if err != nil {
			return err
		}
	} else {
		cfg.DataSettings.DatabaseData.EndDate = defaultEnd
	}
	fmt.Println("Is the end date inclusive? y/n")
	input = quickParse(reader)
	cfg.DataSettings.DatabaseData.InclusiveEndDate = input == y || input == yes
	cfg.DataSettings.DatabaseData.Config = database.Config{
		Enabled: true,
	}
	fmt.Println("Do you want database verbose output? y/n")
	input = quickParse(reader)
	cfg.DataSettings.DatabaseData.Config.Verbose = input == y || input == yes

	fmt.Printf("What database driver to use? %v %v or %v\n", database.DBPostgreSQL, database.DBSQLite, database.DBSQLite3)
	cfg.DataSettings.DatabaseData.Config.Driver = quickParse(reader)
	if cfg.DataSettings.DatabaseData.Config.Driver == database.DBSQLite || cfg.DataSettings.DatabaseData.Config.Driver == database.DBSQLite3 {
		fmt.Printf("What is the path to the database directory? Leaving blank will use: '%v'", filepath.Join(gctcommon.GetDefaultDataDir(runtime.GOOS), "database"))
		cfg.DataSettings.DatabaseData.Path = quickParse(reader)
	}
	fmt.Println("What is the database host?")
	cfg.DataSettings.DatabaseData.Config.Host = quickParse(reader)

	fmt.Println("What is the database username?")
	cfg.DataSettings.DatabaseData.Config.Username = quickParse(reader)

	fmt.Println("What is the database password? eg 1234")
	cfg.DataSettings.DatabaseData.Config.Password = quickParse(reader)

	fmt.Println("What is the database? eg database.db")
	cfg.DataSettings.DatabaseData.Config.Database = quickParse(reader)

	if cfg.DataSettings.DatabaseData.Config.Driver == database.DBPostgreSQL {
		fmt.Println("What is the database SSLMode? eg disable")
		cfg.DataSettings.DatabaseData.Config.SSLMode = quickParse(reader)
	}
	fmt.Println("What is the database Port? eg 1337")
	input = quickParse(reader)
	var port uint64
	if input != "" {
		port, err = strconv.ParseUint(input, 10, 32)
		if err != nil {
			return err
		}
	}
	cfg.DataSettings.DatabaseData.Config.Port = uint32(port) //nolint:gosec // No overflow risk

	if err = database.DB.SetConfig(&cfg.DataSettings.DatabaseData.Config); err != nil {
		return fmt.Errorf("database failed to set config: %w", err)
	}

	switch cfg.DataSettings.DatabaseData.Config.Driver {
	case database.DBPostgreSQL:
		_, err = dbPSQL.Connect(&cfg.DataSettings.DatabaseData.Config)
	case database.DBSQLite, database.DBSQLite3:
		_, err = dbsqlite3.Connect(cfg.DataSettings.DatabaseData.Config.Database)
	default:
		return fmt.Errorf("unsupported database driver: %q", cfg.DataSettings.DatabaseData.Config.Driver)
	}

	if err != nil {
		return fmt.Errorf("database failed to connect: %w", err)
	}

	return nil
}

func parseLive(reader *bufio.Reader, cfg *config.Config) {
	cfg.DataSettings.LiveData = &config.LiveData{}
	fmt.Println("Do you wish to use live trading? It's highly recommended that you do not. y/n")
	input := quickParse(reader)
	cfg.DataSettings.LiveData.RealOrders = input == y || input == yes
	if cfg.DataSettings.LiveData.RealOrders {
		fmt.Printf("Do you want to set credentials for exchanges? y/n\n")
		input = quickParse(reader)
		if input != yes && input != y {
			return
		}
		for {
			var creds config.Credentials
			fmt.Printf("What is the exchange name? y/n\n")
			creds.Exchange = quickParse(reader)
			fmt.Println("What is the API key?")
			creds.Keys.Key = quickParse(reader)
			fmt.Println("What is the API secret?")
			creds.Keys.Secret = quickParse(reader)
			fmt.Println("What is the Client ID? (leave blank if not applicable)")
			creds.Keys.ClientID = quickParse(reader)
			fmt.Println("What is the 2FA seed? (leave blank if not applicable)")
			creds.Keys.OneTimePassword = quickParse(reader)
			fmt.Println("What is the subaccount to use? (leave blank if not applicable)")
			creds.Keys.SubAccount = quickParse(reader)
			fmt.Println("What is the PEM key? (leave blank if not applicable)")
			creds.Keys.PEMKey = quickParse(reader)
			cfg.DataSettings.LiveData.ExchangeCredentials = append(cfg.DataSettings.LiveData.ExchangeCredentials, creds)
			fmt.Printf("Do you want to add another? y/n\n")
			if input != yes && input != y {
				break
			}
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

func parseKlineInterval(reader *bufio.Reader) (gctkline.Interval, error) {
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
		return allCandles[intNum-1], nil
	}
	for i := range allCandles {
		if strings.EqualFold(response, allCandles[i].Word()) {
			return allCandles[i], nil
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

func customSettingsLoop(reader *bufio.Reader) map[string]any {
	resp := make(map[string]any)
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

func addCurrencySetting(reader *bufio.Reader, usingExchangeLevelFunding bool) (*config.CurrencySettings, error) {
	setting := config.CurrencySettings{}
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
		setting.Asset = supported[intNum-1]
	}
	for i := range supported {
		if strings.EqualFold(response, supported[i].String()) {
			setting.Asset = supported[i]
		}
	}

	fmt.Println("Enter the currency base. eg BTC")
	setting.Base = currency.NewCode(quickParse(reader))
	if setting.Asset == asset.Spot {
		if !usingExchangeLevelFunding {
			fmt.Println("Enter the initial base funds. eg 0")
			parseNum := quickParse(reader)
			if parseNum != "" {
				var d decimal.Decimal
				d, err = decimal.NewFromString(parseNum)
				if err != nil {
					return nil, err
				}
				setting.SpotDetails = &config.SpotDetails{
					InitialBaseFunds: &d,
				}
			}
		}
	}

	fmt.Println("Enter the currency quote. eg USDT")
	setting.Quote = currency.NewCode(quickParse(reader))
	if setting.Asset == asset.Spot && !usingExchangeLevelFunding {
		fmt.Println("Enter the initial quote funds. eg 10000")
		parseNum := quickParse(reader)
		if parseNum != "" {
			var d decimal.Decimal
			d, err = decimal.NewFromString(parseNum)
			if err != nil {
				return nil, err
			}
			if setting.SpotDetails == nil {
				setting.SpotDetails = &config.SpotDetails{
					InitialQuoteFunds: &d,
				}
			} else {
				setting.SpotDetails.InitialQuoteFunds = &d
			}
		}
	}

	fmt.Println("Do you want to set custom fees? If no, Backtester will use default fees for exchange y/n")
	yn := quickParse(reader)
	if yn == y || yn == yes {
		fmt.Println("Enter the maker-fee. eg 0.001")
		parseNum := quickParse(reader)
		if parseNum != "" {
			var d decimal.Decimal
			d, err = decimal.NewFromString(parseNum)
			if err != nil {
				return nil, err
			}
			setting.MakerFee = &d
		}
		fmt.Println("Enter the taker-fee. eg 0.01")
		parseNum = quickParse(reader)
		if parseNum != "" {
			var d decimal.Decimal
			d, err = decimal.NewFromString(parseNum)
			if err != nil {
				return nil, err
			}
			setting.TakerFee = &d
		}
	}

	fmt.Println("Will there be buy-side limits? y/n")
	yn = quickParse(reader)
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

	fmt.Println("Should order size shrink to fit within candle volume? y/n")
	yn = quickParse(reader)
	if yn == y || yn == yes {
		setting.SkipCandleVolumeFitting = true
	}

	fmt.Println("Do you wish to include slippage? y/n")
	yn = quickParse(reader)
	if yn == y || yn == yes {
		fmt.Println("Slippage is randomly determined between the lower and upper bounds.")
		fmt.Println("If the lower bound is 80, then the price can change up to 80% of itself. eg if the price is 100 and the lower bound is 80, then the lowest slipped price is $80")
		fmt.Println("If the upper bound is 100, then the price can be unaffected. A minimum of 80 and a maximum of 100 means that the price will randomly be set between those bounds as a way of emulating slippage")

		fmt.Println("What is the lower bounds of slippage? eg 80")
		setting.MinimumSlippagePercent, err = decimal.NewFromString(quickParse(reader))
		if err != nil {
			return nil, err
		}

		fmt.Println("What is the upper bounds of slippage? eg 100")
		setting.MaximumSlippagePercent, err = decimal.NewFromString(quickParse(reader))
		if err != nil {
			return nil, err
		}
	}

	return &setting, nil
}

func minMaxParse(buySell string, reader *bufio.Reader) (config.MinMax, error) {
	resp := config.MinMax{}
	fmt.Printf("What is the maximum %s size? eg 1\n", buySell)
	parseNum := quickParse(reader)
	if parseNum != "" {
		f, err := strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return resp, err
		}
		resp.MaximumSize = decimal.NewFromFloat(f)
	}
	fmt.Printf("What is the minimum %s size? eg 0.1\n", buySell)
	parseNum = quickParse(reader)
	if parseNum != "" {
		f, err := strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return resp, err
		}
		resp.MinimumSize = decimal.NewFromFloat(f)
	}
	fmt.Printf("What is the maximum spend %s buy? eg 12000\n", buySell)
	parseNum = quickParse(reader)
	if parseNum != "" {
		f, err := strconv.ParseFloat(parseNum, 64)
		if err != nil {
			return resp, err
		}
		resp.MaximumTotal = decimal.NewFromFloat(f)
	}

	return resp, nil
}

func quickParse(reader *bufio.Reader) string {
	customSettingField, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimRight(customSettingField, "\r\n")
}
