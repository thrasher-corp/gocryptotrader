package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	exchangeConfigPath = "../../testdata/configtest.json"
	targetPath         = "../../exchanges"
)

type exchange struct {
	Name        string
	CapitalName string
	REST        bool
	WS          bool
}

var errInvalidExchangeName = errors.New("invalid exchange name")

func main() {
	var newExchangeName string
	var websocketSupport, restSupport bool

	flag.StringVar(&newExchangeName, "name", "", "the exchange name")
	flag.BoolVar(&websocketSupport, "ws", false, "whether the exchange supports websocket")
	flag.BoolVar(&restSupport, "rest", false, "whether the exchange supports REST")

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

	if !websocketSupport && !restSupport {
		log.Println("At least one protocol must be specified (rest/ws)")
		flag.Usage()
		return
	}

	fmt.Println("Exchange Name: ", newExchangeName)
	fmt.Println("Websocket Supported: ", websocketSupport)
	fmt.Println("REST Supported: ", restSupport)
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
	}
	exchangeDirectory := filepath.Join(targetPath, exch.Name)
	configTestFile := config.GetConfig()

	var newConfig *config.Exchange
	newConfig, err = makeExchange(exchangeDirectory, configTestFile, &exch)
	if err != nil {
		log.Fatal(err)
	}

	err = saveConfig(exchangeDirectory, configTestFile, newConfig)
	if err != nil {
		log.Fatal(err)
	}
}

func checkExchangeName(exchName string) error {
	if strings.Contains(exchName, " ") ||
		len(exchName) <= 2 {
		return errInvalidExchangeName
	}
	return nil
}

func makeExchange(exchangeDirectory string, configTestFile *config.Config, exch *exchange) (*config.Exchange, error) {
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
	err = os.MkdirAll(exchangeDirectory, file.DefaultPermissionOctal)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Output directory: %s\n", exchangeDirectory)

	exch.CapitalName = cases.Title(language.English).String(exch.Name)
	newExchConfig := &config.Exchange{}
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
			Delimiter: currency.DashDelimiter,
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
			TemplateFile: "readme.tmpl",
		},
		{
			Name:         "rest",
			Filename:     "rest.go",
			TemplateFile: "rest.tmpl",
		},
		{
			Name:         "test",
			Filename:     "test_file.tmpl",
			FilePostfix:  "_test.go",
			TemplateFile: "test.tmpl",
		},
		{
			Name:         "types",
			Filename:     "types.go",
			TemplateFile: "types.tmpl",
		},
		{
			Name:         "wrapper",
			Filename:     "wrapper.go",
			TemplateFile: "wrapper.tmpl",
		},
		{
			Name:         "subscriptions",
			Filename:     "subscriptions.go",
			TemplateFile: "subscriptions.tmpl",
		},
		{
			Name:         "websocket",
			Filename:     "websocket.go",
			TemplateFile: "websocket.tmpl",
		},
	}

	for x := range outputFiles {
		var tmpl *template.Template
		tmpl, err = template.New(outputFiles[x].Name).ParseFiles(outputFiles[x].TemplateFile)
		if err != nil {
			return nil, fmt.Errorf("%s template error: %s", outputFiles[x].Name, err)
		}

		filename := outputFiles[x].Filename
		if !exch.WS && slices.Contains([]string{"websocket", "subscriptions"}, outputFiles[x].Name) {
			continue
		}
		if outputFiles[x].FilePostfix != "" {
			filename = exch.Name + outputFiles[x].FilePostfix
		}

		outputFile := filepath.Join(exchangeDirectory, filename)
		newFile(outputFile)
		var f *os.File
		f, err = os.OpenFile(outputFile, os.O_WRONLY, file.DefaultPermissionOctal)
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

func saveConfig(exchangeDirectory string, configTestFile *config.Config, newExchConfig *config.Exchange) error {
	if err := runCommand(exchangeDirectory, "fmt"); err != nil {
		return err
	}

	configTestFile.Exchanges = append(configTestFile.Exchanges, *newExchConfig)
	if err := configTestFile.SaveConfigToFile(exchangeConfigPath); err != nil {
		return err
	}

	return runCommand(exchangeDirectory, "test")
}

func runCommand(dir, param string) error {
	cmd := exec.CommandContext(context.TODO(), "go", param)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to go %s stdout: %s stderr: %s",
			param, out, err)
	}
	return nil
}

func newFile(path string) {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
}
