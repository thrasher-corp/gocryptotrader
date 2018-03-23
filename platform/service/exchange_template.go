package service

import (
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	mainPath = "%s%ssrc%sgithub.com%sthrasher-%sgocryptotrader%s"

	packageTests   = "%s_test.go"
	packageTypes   = "%s_types.go"
	packageWrapper = "%s_wrapper.go"
	packageMain    = "%s.go"
	packageReadme  = "README.md"

	exchangeServiceFile = "%splatform%sservice%sexchange_template_files%s"

	exchangePackageLocation = "%sexchanges%s"
	exchangeConfigPath      = "%stestdata%sconfigtest.json"
)

var (
	exchangeDirectory string
	exchangeTest      string
	exchangeTypes     string
	exchangeWrapper   string
	exchangeMain      string
	exchangeReadme    string
	exchangeJSON      string

	templateFilePath string
)

type exchange struct {
	Name        string
	CapitalName string
	Variable    string
	REST        bool
	WS          bool
	FIX         bool
}

// StartExchangeTemplate creates a new exchange template
func StartExchangeTemplate(newExchangeName, goPath string, websocketSupport, restSupport, fixSupport bool) {

	if newExchangeName == "" || newExchangeName == " " {
		log.Fatal(`GoCryptoTrader: Exchange templating tool exchange name not set e.g. "-createexchange <exchange name>"`)
	}

	if !websocketSupport && !restSupport && !fixSupport {
		log.Fatal(`GoCryptoTrader: Exchange templating tool support not set e.g. "-createexchange <exchange name> [-rs | -ws | -fs]"`)
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

	pathSeparator := common.GetOSPathSlash()
	mainAbsolutePath := fmt.Sprintf(mainPath,
		goPath,
		pathSeparator,
		pathSeparator,
		pathSeparator,
		pathSeparator,
		pathSeparator)

	exchangeJSON := fmt.Sprintf(exchangeConfigPath, mainAbsolutePath, pathSeparator)

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
		mainAbsolutePath,
		pathSeparator,
		pathSeparator)

	exchangeTest = fmt.Sprintf(exchangeDirectory+packageTests, newExchangeName)
	exchangeTypes = fmt.Sprintf(exchangeDirectory+packageTypes, newExchangeName)
	exchangeWrapper = fmt.Sprintf(exchangeDirectory+packageWrapper, newExchangeName)
	exchangeMain = fmt.Sprintf(exchangeDirectory+packageMain, newExchangeName)
	exchangeReadme = exchangeDirectory + packageReadme

	templateFilePath := fmt.Sprintf(exchangeServiceFile,
		absolutePath,
		pathSeparator,
		pathSeparator,
		pathSeparator)

	err = os.Mkdir(exchangeDirectory, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot make directory ", err)
	}

	tReadme, err := template.New("readme").ParseFiles(templateFilePath + "readme_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeReadme)
	r1, err := os.OpenFile(exchangeReadme, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tReadme.Execute(r1, exch)

	tMain, err := template.New("main").ParseFiles(templateFilePath + "main_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeMain)
	m1, err := os.OpenFile(exchangeMain, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tMain.Execute(m1, exch)

	tTest, err := template.New("test").ParseFiles(templateFilePath + "test_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeTest)
	t1, err := os.OpenFile(exchangeTest, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tTest.Execute(t1, exch)

	tType, err := template.New("type").ParseFiles(templateFilePath + "type_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeTypes)
	ty1, err := os.OpenFile(exchangeTypes, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tType.Execute(ty1, exch)

	tWrapper, err := template.New("wrapper").ParseFiles(templateFilePath + "wrapper_file.tmpl")
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool error ", err)
	}
	newFile(exchangeWrapper)
	w1, err := os.OpenFile(exchangeWrapper, os.O_WRONLY, 0700)
	if err != nil {
		log.Fatal("GoCryptoTrader: Exchange templating tool cannot open file ", err)
	}
	tWrapper.Execute(w1, exch)

	fmt.Println("GoCryptoTrader: Exchange templating tool service complete")
	fmt.Println("When wrapper is finished add exchange to exchange.go")
	fmt.Println("Test exchange.go")
	fmt.Println("Update the config_test.go file")
	fmt.Println("Test config.go")
	fmt.Println("Open a pull request")
	fmt.Println("If help is needed please post a message on the slack.")
	os.Exit(0)
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
