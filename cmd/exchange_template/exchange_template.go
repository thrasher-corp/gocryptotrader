package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

const (
	exchangeConfigPath = "../../testdata/configtest.json"
	targetPath         = "../../exchanges"
)

type exchange struct {
	Name        string
	CapitalName string
	Variable    string
	REST        bool
	WS          bool
	FIX         bool
}

var (
	errInvalidExchangeName = errors.New("invalid exchange name")
)

func main() {
	var newExchangeName string
	var websocketSupport, restSupport, fixSupport bool

	flag.StringVar(&newExchangeName, "name", "", "the exchange name")
	flag.BoolVar(&websocketSupport, "ws", false, "whether the exchange supports websocket")
	flag.BoolVar(&restSupport, "rest", false, "whether the exchange supports REST")
	flag.BoolVar(&fixSupport, "fix", false, "whether the exchange supports FIX")

	flag.Parse()

	fmt.Println("GoCryptoTrader: Exchange templating tool.")
	fmt.Println(core.Copyright)
	fmt.Println()

	if len(os.Args) == 1 {
		log.Println("Invalid arguments supplied, please see application usage below:")
		flag.Usage()
		return
	}

	if err := checkExchangeName(newExchangeName); err != nil {
		log.Fatal(err)
	}
	newExchangeName = strings.ToLower(newExchangeName)

	if !websocketSupport && !restSupport && !fixSupport {
		log.Println("At least one protocol must be specified (rest/ws or fix)")
		flag.Usage()
		return
	}

	fmt.Println("Exchange Name: ", newExchangeName)
	fmt.Println("Websocket Supported: ", websocketSupport)
	fmt.Println("REST Supported: ", restSupport)
	fmt.Println("FIX Supported: ", fixSupport)
	fmt.Println()
	fmt.Println("Please check if everything is correct and then type y to continue or n to cancel...")

	var choice []byte
	_, err := fmt.Scanln(&choice)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool fmt.Scanln ", err)
	}

	if !common.YesOrNo(string(choice)) {
		log.Fatal("GoCryptoTrader: Exchange templating tool stopped...")
	}

	exch := exchange{
		Name: newExchangeName,
		REST: restSupport,
		WS:   websocketSupport,
		FIX:  fixSupport,
	}
	exchangeDirectory := filepath.Join(targetPath, exch.Name)
	configTestFile := config.GetConfig()

	var newConfig *config.ExchangeConfig
	newConfig, err = makeExchange(exchangeDirectory, configTestFile, &exch)
	if err != nil {
		log.Fatal(err)
	}

	err = saveConfig(exchangeDirectory, configTestFile, newConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("GoCryptoTrader: Exchange templating tool service complete")
	fmt.Println("When the exchange code implementation has been completed (REST/Websocket/wrappers and tests), please add the exchange to engine/exchange.go")
	fmt.Println("Add the exchange config settings to config_example.json (it will automatically be added to testdata/configtest.json)")
	fmt.Println("Increment the available exchanges counter in config/config_test.go")
	fmt.Println("Add the exchange name to exchanges/support.go")
	fmt.Println("Ensure go test ./... -race passes")
	fmt.Println("Open a pull request")
	fmt.Println("If help is needed, please post a message in Slack.")
}

func checkExchangeName(exchName string) error {
	if strings.Contains(exchName, " ") ||
		len(exchName) <= 2 {
		return errInvalidExchangeName
	}
	return nil
}

func makeExchange(exchangeDirectory string, configTestFile *config.Config, exch *exchange) (*config.ExchangeConfig, error) {
	err := configTestFile.LoadConfig(exchangeConfigPath, true)
	if err != nil {
		return nil, err
	}
	// NOTE need to nullify encrypt configuration

	_, err = configTestFile.GetExchangeConfig(exch.Name)
	if err == nil {
		return nil, errors.New("exchange already exists")
	}

	_, err = os.Stat(exchangeDirectory)
	if !os.IsNotExist(err) {
		return nil, errors.New("directory already exists")
	}
	err = os.MkdirAll(exchangeDirectory, 0770)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Output directory: %s\n", exchangeDirectory)

	exch.CapitalName = strings.Title(exch.Name)
	exch.Variable = exch.Name[0:2]
	newExchConfig := &config.ExchangeConfig{}
	newExchConfig.Name = exch.CapitalName
	newExchConfig.Enabled = true
	newExchConfig.API.Credentials.Key = "Key"
	newExchConfig.API.Credentials.Secret = "Secret"
	newExchConfig.CurrencyPairs = &currency.PairsManager{
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
		},
	}

	outputFiles := []struct {
		Name         string
		Filename     string
		FilePostfix  string
		TemplateFile string
	}{
		{
			Name:         "readme",
			Filename:     "README.md",
			TemplateFile: "readme_file.tmpl",
		},
		{
			Name:         "main",
			Filename:     "main_file.tmpl",
			FilePostfix:  ".go",
			TemplateFile: "main_file.tmpl",
		},
		{
			Name:         "test",
			Filename:     "test_file.tmpl",
			FilePostfix:  "_test.go",
			TemplateFile: "test_file.tmpl",
		},
		{
			Name:         "type",
			Filename:     "type_file.tmpl",
			FilePostfix:  "_types.go",
			TemplateFile: "type_file.tmpl",
		},
		{
			Name:         "wrapper",
			Filename:     "wrapper_file.tmpl",
			FilePostfix:  "_wrapper.go",
			TemplateFile: "wrapper_file.tmpl",
		},
	}

	for x := range outputFiles {
		var tmpl *template.Template
		tmpl, err = template.New(outputFiles[x].Name).ParseFiles(outputFiles[x].TemplateFile)
		if err != nil {
			return nil, fmt.Errorf("%s template error: %s", outputFiles[x].Name, err)
		}

		filename := outputFiles[x].Filename
		if outputFiles[x].FilePostfix != "" {
			filename = exch.Name + outputFiles[x].FilePostfix
		}

		outputFile := filepath.Join(exchangeDirectory, filename)
		newFile(outputFile)
		var f *os.File
		f, err = os.OpenFile(outputFile, os.O_WRONLY, 0770)
		if err != nil {
			return nil, err
		}

		if err = tmpl.Execute(f, exch); err != nil {
			f.Close()
			return nil, err
		}
		f.Close()
	}

	return newExchConfig, nil
}

func saveConfig(exchangeDirectory string, configTestFile *config.Config, newExchConfig *config.ExchangeConfig) error {
	cmd := exec.Command("go", "fmt")
	cmd.Dir = exchangeDirectory
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("unable to go fmt. output: %s err: %s", out, err)
	}

	configTestFile.Exchanges = append(configTestFile.Exchanges, *newExchConfig)
	err = configTestFile.SaveConfigToFile(exchangeConfigPath)
	if err != nil {
		return err
	}

	cmd = exec.Command("go", "test")
	cmd.Dir = exchangeDirectory
	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("unable to go test. output: %s err: %s", out, err)
	}
	return nil
}

func newFile(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		file.Close()
	}
}
