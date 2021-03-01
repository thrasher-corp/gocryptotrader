package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

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
	fmt.Println("Firstly, please select which strategy you wish to use")
	strats := strategies.GetStrategies()
	var strategiesToUse []string
	for i := range strats {
		fmt.Printf("%v. %s\n", i+1, strats[i].Name())
		strategiesToUse = append(strategiesToUse, strats[i].Name())
	}

	var err error
	cfg.StrategySettings.Name, err = parseStratName(quickParse(reader), strategiesToUse)

	fmt.Println("Enter a nickname, it can help distinguish between different configs using the same strategy")
	cfg.Nickname = quickParse(reader)
	fmt.Println("Does this strategy have custom settings? y/n")
	customSettings := quickParse(reader)
	if strings.Contains(customSettings, "y") {
		cfg.StrategySettings.CustomSettings = customSettingsLoop(reader)
	}

	fmt.Println("-----Exchange Settings-----")
	addCurrency := "y"
	for strings.Contains(addCurrency, "y") {
		var currencySetting *config.CurrencySettings
		currencySetting, err = addCurrencySetting(reader)
		if err != nil {
			log.Fatal(err)
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
				if yn == "y" || yn == "yes" {
					cfg.StrategySettings.SimultaneousSignalProcessing = true
				}
			}
			break
		}
	}

	fmt.Println("-----Portfolio Settings-----")
	fmt.Println("Will there be global portfolio buy-side limits? y/n")
	yn := quickParse(reader)
	if yn == "y" || yn == "yes" {
		cfg.PortfolioSettings.BuySide, err = minMaxParse("buy", reader)
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Will there be global portfolio sell-side limits? y/n")
	yn = quickParse(reader)
	if yn == "y" || yn == "yes" {
		cfg.PortfolioSettings.SellSide, err = minMaxParse("sell", reader)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("-----Data Settings-----")
	fmt.Println("Will you be using \"candle\" or \"trade\" data?")
	cfg.DataSettings.DataType = quickParse(reader)
	fmt.Println("What time interval will you use?")
	allCandles := gctkline.SupportedCandles
	for i := range allCandles {
		fmt.Printf("%v. %s", i+1, allCandles[i])
	}
	cfg.DataSettings.DataType = quickParse(reader)

	fmt.Println("-----Statistics Settings-----")
	fmt.Println("Enter the risk free rate. eg 0.03")
	cfg.StatisticSettings.RiskFreeRate, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		log.Fatal(err)
	}

	var resp []byte
	resp, err = json.MarshalIndent(cfg, "", " ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Write strat to file? If no, the output will be on screen y/n")
	yn = quickParse(reader)
	if yn == "y" || yn == "yes" {
		var wd string
		wd, err = os.Getwd()
		wd = filepath.Join(wd, cfg.StrategySettings.Name+"-"+cfg.Nickname, ".strat")
		fmt.Printf("Enter output file. If blank, will output to \"%v\"\n", wd)
		path := quickParse(reader)
		if path == "" {
			path = wd
		}
		err = os.WriteFile(path, resp, 0770)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Print(string(resp))
	}
	log.Println("Config creation complete!")
}

func parseStratName(name string, strategiesToUse []string) (string, error) {
	num, err := strconv.ParseFloat(name, 64)
	if err == nil {
		intNum := int(num)
		if intNum > len(strategiesToUse) {
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

	fmt.Println("Enter the asset. Enter \"help\" for a list of supported asset types")
	setting.Asset = quickParse(reader)
	if setting.Asset == "help" {
		supported := asset.Supported()
		for i := range supported {
			fmt.Println(supported[i].String())
		}
		fmt.Println("Enter the asset. eg spot")
		setting.Asset = quickParse(reader)
	}

	fmt.Println("Enter the currency base. eg BTC")
	setting.Base = quickParse(reader)

	fmt.Println("Enter the currency quote. eg USDT")
	setting.Quote = quickParse(reader)

	fmt.Println("Enter the initial funds. eg 10000")
	var err error
	setting.InitialFunds, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return nil, err
	}

	fmt.Println("Enter the maker-fee. eg 0.001")
	setting.MakerFee, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return nil, err
	}
	fmt.Println("Enter the taker-fee. eg 0.01")
	setting.TakerFee, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return nil, err
	}

	fmt.Println("Will there be buy-side limits? y/n")
	yn := quickParse(reader)
	if yn == "y" || yn == "yes" {
		setting.BuySide, err = minMaxParse("buy", reader)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println("Will there be sell-side limits? y/n")
	yn = quickParse(reader)
	if yn == "y" || yn == "yes" {
		setting.SellSide, err = minMaxParse("sell", reader)
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
	resp.MaximumSize, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return resp, err
	}
	fmt.Printf("What is the minimum %s size? eg 0.1\n", buySell)
	resp.MinimumSize, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return resp, err
	}
	fmt.Printf("What is the maximum spend %s buy? eg 12000\n", buySell)
	resp.MaximumTotal, err = strconv.ParseFloat(quickParse(reader), 64)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func quickParse(reader *bufio.Reader) string {
	customSettingField, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(customSettingField, "\n", "", 1)
}
