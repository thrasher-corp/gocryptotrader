package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	packageTests   = "%s_test.go"
	packageTypes   = "%s_types.go"
	packageWrapper = "%s_wrapper.go"
	packageMain    = "%s.go"
	packageReadme  = "README.md"

	exchangePackageLocation = "..%s..%sexchanges%s"
	exchangeLocation        = "..%s..%sexchange.go"
	exchangeConfigPath      = "..%s..%stestdata%sconfigtest.json"
)

var (
	exchangeDirectory string
	exchangeTest      string
	exchangeTypes     string
	exchangeWrapper   string
	exchangeMain      string
	exchangeReadme    string
	exchangeJSON      string
)

type exchange struct {
	Name        string
	CapitalName string
	Variable    string
	REST        bool
	WS          bool
	FIX         bool
}

func main() {
	var newExchangeName string
	var websocketSupport, restSupport, fixSupport bool

	flag.StringVar(&newExchangeName, "name", "", "-name [string] adds a new exchange")
	flag.BoolVar(&websocketSupport, "ws", false, "-websocket adds websocket support")
	flag.BoolVar(&restSupport, "rest", false, "-rest adds REST support")
	flag.BoolVar(&fixSupport, "fix", false, "-fix adds FIX support?")

	flag.Parse()

	fmt.Println("GoCryptoTrader: Exchange templating tool.")

	if newExchangeName == "" || newExchangeName == " " {
		log.Fatal(`GoCryptoTrader: Exchange templating tool exchange name not set e.g. "exchange_template -name [newExchangeNameString]"`)
	}

	if !websocketSupport && !restSupport && !fixSupport {
		log.Fatal(`GoCryptoTrader: Exchange templating tool support not set e.g. "exchange_template -name [newExchangeNameString] [-fix -ws -rest]"`)
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

	newExchangeName = common.StringToLower(newExchangeName)
	v := newExchangeName[:1]
	capName := common.StringToUpper(v) + newExchangeName[1:]

	exch := exchange{
		Name:        newExchangeName,
		CapitalName: capName,
		Variable:    v,
		REST:        restSupport,
		WS:          websocketSupport,
		FIX:         fixSupport,
	}

	osPathSlash := common.GetOSPathSlash()
	exchangeJSON := fmt.Sprintf(exchangeConfigPath, osPathSlash, osPathSlash, osPathSlash)

	configTestFile := config.GetConfig()
	err = configTestFile.LoadConfig(exchangeJSON)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating configuration retrieval error ", err)
	}
	// NOTE need to nullify encrypt configuration

	var configTestExchanges []string

	for _, exch := range configTestFile.Exchanges {
		configTestExchanges = append(configTestExchanges, exch.Name)
	}

	if common.StringDataContainsUpper(configTestExchanges, capName) {
		log.Fatal("GoCryptoTrader: Exchange templating configuration error - exchange already exists")
	}

	newExchConfig := config.ExchangeConfig{}
	newExchConfig.Name = capName
	newExchConfig.Enabled = true
	newExchConfig.RESTPollingDelay = 10
	newExchConfig.APIKey = "Key"
	newExchConfig.APISecret = "Secret"
	newExchConfig.AssetTypes = "SPOT"

	configTestFile.Exchanges = append(configTestFile.Exchanges, newExchConfig)
	// TODO sorting function so exchanges are in alphabetical order - low priority

	err = configTestFile.SaveConfig(exchangeJSON)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating configuration error - cannot save")
	}

	exchangeDirectory = fmt.Sprintf(
		exchangePackageLocation+newExchangeName+"%s",
		osPathSlash,
		osPathSlash,
		osPathSlash,
		osPathSlash)

	exchangeTest = fmt.Sprintf(exchangeDirectory+packageTests, newExchangeName)
	exchangeTypes = fmt.Sprintf(exchangeDirectory+packageTypes, newExchangeName)
	exchangeWrapper = fmt.Sprintf(exchangeDirectory+packageWrapper, newExchangeName)
	exchangeMain = fmt.Sprintf(exchangeDirectory+packageMain, newExchangeName)
	exchangeReadme = exchangeDirectory + packageReadme

	err = os.Mkdir(exchangeDirectory, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot make directory ", err)
	}

	tReadme, err := template.New("readme").ParseFiles("readme_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeReadme)
	r1, err := os.OpenFile(exchangeReadme, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tReadme.Execute(r1, exch)

	tMain, err := template.New("main").ParseFiles("main_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeMain)
	m1, err := os.OpenFile(exchangeMain, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tMain.Execute(m1, exch)

	tTest, err := template.New("test").ParseFiles("test_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeTest)
	t1, err := os.OpenFile(exchangeTest, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tTest.Execute(t1, exch)

	tType, err := template.New("type").ParseFiles("type_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeTypes)
	ty1, err := os.OpenFile(exchangeTypes, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tType.Execute(ty1, exch)

	tWrapper, err := template.New("wrapper").ParseFiles("wrapper_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeWrapper)
	w1, err := os.OpenFile(exchangeWrapper, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tWrapper.Execute(w1, exch)

	err = exec.Command("go", "fmt", exchangeDirectory).Run()
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool go fmt error ", err)
	}

	err = exec.Command("go", "test", exchangeDirectory).Run()
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool testing failed ", err)
	}

	fmt.Println("GoCryptoTrader: Exchange templating tool service complete")
	fmt.Println("When wrapper is finished add exchange to exchange.go")
	fmt.Println("Test exchange.go")
	fmt.Println("Update the config_test.go file")
	fmt.Println("Test config.go")
	fmt.Println("Open a pull request")
	fmt.Println("If help is needed please post a message on the slack.")
}

func newFile(path string) {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		defer file.Close()
		if err != nil {
			log.Fatal("GoCryptoTrader: Exchange templating tool file creation error ", err)
		}
	}
}
