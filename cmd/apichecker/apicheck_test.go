package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	gctfile "github.com/thrasher-corp/gocryptotrader/common/file"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	testAPIKey      = ""
	testAPIToken    = ""
	testChecklistID = ""
	testCardID      = ""
	testListID      = ""
	testBoardID     = ""
	testBoardName   = ""
	canTestMainFile = false
)

func TestMain(m *testing.M) {
	setTestVars()
	err := log.SetGlobalLogConfig(log.GenDefaultSettings())
	if err != nil {
		log.Errorln(log.Global, err)
		os.Exit(1)
	}
	log.Infoln(log.Global, "set verbose to true for more detailed output")
	configData, err = readFileData(jsonFile)
	if err != nil {
		log.Errorln(log.Global, err)
		os.Exit(1)
	}
	testConfigData, err = readFileData(testJSONFile)
	if err != nil {
		log.Errorln(log.Global, err)
		os.Exit(1)
	}
	usageData = testConfigData
	setTestVars()
	testExitCode := m.Run()
	err = removeTestFileVars()
	if err != nil {
		log.Errorln(log.Global, err)
		os.Exit(1)
	}
	os.Exit(testExitCode)
}

func areTestAPIKeysSet() bool {
	return (testAPIKey != "" && testAPIToken != "")
}

func setTestVars() {
	if !canUpdateTrello() {
		apiKey = testAPIKey
		apiToken = testAPIToken
		trelloChecklistID = testChecklistID
		trelloCardID = testCardID
		trelloListID = testListID
		trelloBoardID = testBoardID
		trelloBoardName = testBoardName
		return
	}
}

func removeTestFileVars() error {
	a, err := readFileData(testJSONFile)
	if err != nil {
		return err
	}
	a.BoardID = ""
	a.CardID = ""
	a.ChecklistID = ""
	a.Key = ""
	a.ListID = ""
	a.Token = ""
	file, err := json.MarshalIndent(&a, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(testJSONFile, file, gctfile.DefaultPermissionOctal)
}

func canTestTrello() bool {
	if testAPIKey != "" && testAPIToken != "" && testChecklistID != "" && testCardID != "" && testListID != "" && (testBoardID != "" || testBoardName != "") {
		return true
	}
	return false
}

func TestCheckUpdates(t *testing.T) {
	if !canUpdateTrello() || !canTestTrello() {
		t.Skip("cannot update or test trello, skipping")
	}
	err := checkUpdates(testJSONFile)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateFile(t *testing.T) {
	realConf, err := readFileData(jsonFile)
	if err != nil {
		t.Fatal(err)
	}
	configData = realConf
	err = updateFile(testJSONFile)
	if err != nil {
		t.Fatal(err)
	}
	testConf, err := readFileData(testJSONFile)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(realConf, testConf) {
		t.Error("test file update failed")
	}
}

func TestCheckExistingExchanges(t *testing.T) {
	t.Parallel()
	if !checkExistingExchanges("Kraken") {
		t.Fatal("Kraken data not found")
	}
}

func TestCheckChangeLog(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "revision-history",
		TokenDataEnd:  "table",
		TextTokenData: "td",
		DateFormat:    "2006/01/02",
		RegExp:        `^20(\d){2}/(\d){2}/(\d){2}$`,
		Path:          "https://docs.gemini.com/rest-api/#revision-history"}
	if _, err := checkChangeLog(&data); err != nil {
		t.Error(err)
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()
	data2 := HTMLScrapingData{
		TokenData:     "h1",
		Key:           "id",
		Val:           "change-log",
		TextTokenData: "strong",
		TokenDataEnd:  "p",
		Path:          "incorrectpath",
	}
	err := addExch("FalseName", htmlScrape, data2, false)
	if err == nil {
		t.Error("expected an error due to invalid path being parsed in")
	}
}

func TestHTMLScrapeGemini(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "revision-history",
		TokenDataEnd:  "table",
		TextTokenData: "td",
		DateFormat:    "2006/01/02",
		RegExp:        "^20(\\d){2}/(\\d){2}/(\\d){2}$",
		CheckString:   "2019/11/15",
		Path:          "https://docs.gemini.com/rest-api/#revision-history"}
	_, err := htmlScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeHuobi(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "change-log",
		TokenDataEnd:  "h2",
		TextTokenData: "td",
		DateFormat:    "2006.01.02 15:04",
		RegExp:        "^20(\\d){2}.(\\d){2}.(\\d){2} (\\d){2}:(\\d){2}$",
		CheckString:   "2019.12.27 19:00",
		Path:          "https://huobiapi.github.io/docs/spot/v1/en/#change-log"}
	_, err := htmlScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeCoinbasepro(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "changelog",
		TokenDataEnd:  "ul",
		TextTokenData: "strong",
		DateFormat:    "01/02/06",
		RegExp:        "^(\\d){1,2}/(\\d){1,2}/(\\d){2}$",
		CheckString:   "12/16/19",
		Path:          "https://docs.pro.coinbase.com/#changelog"}
	_, err := htmlScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitfinex(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{DateFormat: "2006-01-02",
		RegExp: `section-v-(2\d{3}-\d{1,2}-\d{1,2})`,
		Path:   "https://docs.bitfinex.com/docs/changelog"}
	_, err := htmlScrapeBitfinex(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitmex(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h4",
		Key:           "id",
		Val:           "",
		TokenDataEnd:  "",
		TextTokenData: "",
		DateFormat:    "Jan-2-2006",
		RegExp:        `([A-Z]{1}[a-z]{2}-\d{1,2}-2\d{3})`,
		Path:          "https://www.bitmex.com/static/md/en-US/apiChangelog"}
	if _, err := htmlScrapeBitmex(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeHitBTC(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `newest version \d{1}.\d{1}`,
		Path: "https://api.hitbtc.com/"}
	if _, err := htmlScrapeHitBTC(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeDefault(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h3",
		Key:           "id",
		Val:           "change-change",
		TokenDataEnd:  "section",
		TextTokenData: "p",
		DateFormat:    "2006-01-02",
		RegExp:        "(2\\d{3}-\\d{1,2}-\\d{1,2})",
		CheckString:   "2019-04-28",
		Path:          "https://www.okcoin.com/docs/en/#change-change"}
	_, err := htmlScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBTSE(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `^version: \d{1}.\d{1}.\d{1}`,
		Path: "https://api.btcmarkets.net/openapi/info/index.yaml"}
	if _, err := htmlScrapeBTSE(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBTCMarkets(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `^version: \d{1}.\d{1}.\d{1}`,
		Path: "https://api.btcmarkets.net/openapi/info/index.yaml"}
	if _, err := htmlScrapeBTCMarkets(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitflyer(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "p",
		TokenDataEnd:  "h3",
		TextTokenData: "code",
		RegExp:        `^https://api.bitflyer.com/v\d{1}/$`,
		Path:          "https://lightning.bitflyer.com/docs?lang=en"}
	if _, err := htmlScrapeBitflyer(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeANX(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `ANX Exchange API v\d{1}`,
		Path: "https://anxv3.docs.apiary.io/#reference/quickstart-catalog"}
	if _, err := htmlScrapeANX(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLPoloniex(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "changelog",
		TokenDataEnd:  "div",
		TextTokenData: "h2",
		DateFormat:    "2006-01-02",
		RegExp:        `(2\d{3}-\d{1,2}-\d{1,2})`,
		Path:          "https://docs.poloniex.com/#changelog"}
	if _, err := htmlScrapePoloniex(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLItBit(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "a",
		Key:           "href",
		Val:           "changelog",
		TokenDataEnd:  "div",
		TextTokenData: "h2",
		DateFormat:    "2006-01-02",
		RegExp:        `^https://api.itbit.com/v\d{1}/$`,
		Path:          "https://api.itbit.com/docs"}
	if _, err := htmlScrapeItBit(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeExmo(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
		Path: "https://exmo.com/en/api/"}
	if _, err := htmlScrapeExmo(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLBitstamp(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `refer to the v\d{1} API for future references.`,
		Path: "https://www.bitstamp.net/api/"}
	if _, err := htmlScrapeBitstamp(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLKraken(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h3",
		TokenDataEnd:  "p",
		TextTokenData: "p",
		RegExp:        `URL: https://api.kraken.com/\d{1}/private/Balance`,
		Path:          "https://www.kraken.com/features/api"}
	if _, err := htmlScrapeKraken(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLAlphaPoint(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "introduction",
		TokenDataEnd:  "blockquote",
		TextTokenData: "h3",
		RegExp:        `revised-calls-\d{1}-\d{1}-\d{1}-gt-\d{1}-\d{1}-\d{1}`,
		Path:          "https://alphapoint.github.io/slate/#introduction"}
	if _, err := htmlScrapeAlphaPoint(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLYobit(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h2",
		Key:  "id",
		Path: "https://www.yobit.net/en/api/"}
	if _, err := htmlScrapeYobit(&data); err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeOk(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "a",
		Key:          "href",
		Val:          "./#change-change",
		TokenDataEnd: "./#change-",
		RegExp:       `./#change-\d{8}`,
		Path:         "https://www.okx.com/docs/en/"}
	if _, err := htmlScrapeOk(&data); err != nil {
		t.Error(err)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	var exchCheck, updatedExch HTMLScrapingData
	for x := range configData.Exchanges {
		if configData.Exchanges[x].Name == "Exmo" {
			exchCheck = *configData.Exchanges[x].Data.HTMLData
		}
	}
	info := ExchangeInfo{Name: "Exmo",
		CheckType: "HTML String Check",
		Data: &CheckData{HTMLData: &HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
			Path: "https://exmo.com/en/api/"},
		},
	}
	updatedExchs := update("Exmo", configData.Exchanges, info)
	for y := range updatedExchs {
		if updatedExchs[y].Name == "Exmo" {
			updatedExch = *updatedExchs[y].Data.HTMLData
		}
	}
	if updatedExch == exchCheck {
		t.Fatal("update failed")
	}
}

func TestCheckMissingExchanges(t *testing.T) {
	t.Parallel()
	if a := checkMissingExchanges(); len(a) > len(exchange.Exchanges) {
		t.Fatal("invalid response")
	}
}

func TestNameUpdates(t *testing.T) {
	t.Parallel()
	tester := []struct {
		Name          string
		Status        string
		ExpectedName  string
		ErrorExpected bool
	}{
		{
			Name:          "incorrectname",
			Status:        "incomplete",
			ErrorExpected: true,
		},
		{
			Name:          "Gemini 2 2",
			Status:        "incomplete",
			ErrorExpected: false,
		},
		{
			Name:          " Gemini 23",
			Status:        "incomplete",
			ErrorExpected: true,
		},
		{
			Name:          "Gemini 123",
			Status:        "complete",
			ExpectedName:  "Gemini 1",
			ErrorExpected: false,
		},
		{
			Name:          "Gemini",
			Status:        "complete",
			ExpectedName:  "Gemini 1",
			ErrorExpected: false,
		},
		{
			Name:          "Gemini 24 ",
			Status:        "incomplete",
			ErrorExpected: false,
		},
	}
	for x := range tester {
		r, err := nameStateChanges(tester[x].Name, tester[x].Status)
		if r != tester[x].ExpectedName && err != nil && !tester[x].ErrorExpected {
			t.Errorf("%d failed, expected %v, %v, got: %v, %v\n", x,
				tester[x].ExpectedName,
				tester[x].ErrorExpected,
				r,
				err)
		}
	}
}

func TestReadFileData(t *testing.T) {
	t.Parallel()
	_, err := readFileData(testJSONFile)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSha(t *testing.T) {
	t.Parallel()
	_, err := getSha("binance-exchange/binance-official-api-docs")
	if err != nil {
		t.Error(err)
	}
}

func TestCheckBoardID(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	a, err := trelloCheckBoardID()
	if err != nil {
		t.Error(err)
	}
	if a != true {
		t.Error("no match found for the given boardID")
	}
}

func TestTrelloGetLists(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	if _, err := trelloGetLists(); err != nil {
		t.Error(err)
	}
}

func TestGetAllCards(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	if _, err := trelloGetAllCards(); err != nil {
		t.Error(err)
	}
}

func TestGetAllChecklists(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	if _, err := trelloGetAllChecklists(); err != nil {
		t.Error(err)
	}
}

func TestTrelloGetAllBoards(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	if trelloBoardID != "" || testBoardName != "" {
		t.Skip("trello details empty, skipping")
	}
	if _, err := trelloGetBoardID(); err != nil {
		t.Error(err)
	}
}

func TestCreateNewList(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	if err := trelloCreateNewList(); err != nil {
		t.Error(err)
	}
}

func TestTrelloCreateNewCard(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	if err := trelloCreateNewCard(); err != nil {
		t.Error(err)
	}
}

func TestCreateNewChecklist(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	if err := trelloCreateNewChecklist(); err != nil {
		t.Error(err)
	}
}

func TestWriteAuthVars(t *testing.T) {
	if canTestMainFile {
		trelloCardID = "jdsfl"
		if err := writeAuthVars(testMode); err != nil {
			t.Error(err)
		}
	}
}

func TestCreateNewCheck(t *testing.T) {
	if !canTestTrello() {
		t.Skip("cannot test trello, skipping")
	}
	err := trelloCreateNewCheck("Gemini")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateCheckItem(t *testing.T) {
	if !canTestTrello() {
		t.Skip("cannot test trello, skipping")
	}
	a, err := trelloGetChecklistItems()
	if err != nil {
		t.Error(err)
	}
	var checkID string
	for x := range a.CheckItems {
		if a.CheckItems[x].Name == "Gemini 1" {
			checkID = a.CheckItems[x].ID
		}
	}
	err = trelloUpdateCheckItem(checkID, "Gemini 1", "incomplete")
	if err != nil {
		t.Error(err)
	}
}

func TestGetChecklistItems(t *testing.T) {
	if !canTestTrello() {
		t.Skip("cannot test trello, skipping")
	}
	_, err := trelloGetChecklistItems()
	if err != nil {
		t.Error(err)
	}
}

func TestSetAuthVars(t *testing.T) {
	t.Parallel()
	apiKey = ""
	configData.Key = ""
	apiToken = ""
	configData.Token = ""
	setAuthVars()
	if usageData.Key != "" && usageData.Token != "" {
		t.Errorf("incorrect key and token values")
	}
}

func TestTrelloDeleteCheckItems(t *testing.T) {
	if !areTestAPIKeysSet() {
		t.Skip("API Keys unset, skipping")
	}
	err := trelloDeleteCheckItem("")
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBinance(t *testing.T) {
	data := HTMLScrapingData{
		TokenData:     "h1",
		Key:           "id",
		Val:           "change-log",
		TextTokenData: "strong",
		TokenDataEnd:  "p",
		Path:          "https://binance-docs.github.io/apidocs/spot/en/#change-log",
	}
	_, err := htmlScrapeBinance(&data)
	if err != nil {
		t.Error(err)
	}
}
