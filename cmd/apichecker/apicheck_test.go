package main

import (
	"log"
	"os"
	"testing"
	"reflect"
	"fmt"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func TestMain(m *testing.M) {
	var err error
	configData, err = ReadFileData(jsonFile)
	if err != nil {
		log.Fatal(err)
	}
	testConfigData, err = ReadFileData(testJSONFile)
	if err != nil {
		log.Fatal(err)
	}
	a := m.Run()
	os.Exit(a)
}

func TestCheckExistingExchanges(t *testing.T) {
	t.Parallel()
	a := CheckExistingExchanges("Kraken", &testConfigData)
	if a != true {
		t.Log("Kraken data not found")
		t.Fail()
	}
}

func TestCheckChangeLog(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h3",
		Key:           "id",
		Val:           "change-change",
		TokenDataEnd:  "table",
		TextTokenData: "p",
		DateFormat:    "2006-01-02",
		RegExp:        "(2\\d{3}-\\d{1,2}-\\d{1,2})",
		CheckString:   "2019-04-28",
		Path:          "https://www.okex.com/docs/en/#change-change"}
	_, err := CheckChangeLog(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()
	data := GithubData{Repo: "LBank-exchange/lbank-official-api-docs"}
	err := Add("Lbank", github, fmt.Sprintf(githubPath, data.Repo), data, false, &configData)
	if err != nil {
		t.Error(err)
	}
	data2 := HTMLScrapingData{Path: "brokenpath"}
	err = Add("TestExch", htmlScrape, data2.Path, data2, false, &testConfigData)
	if err == nil {
		t.Log("expected an error due to invalid path being parsed in")
	}
}

func TestCheckUpdates(t *testing.T) {
	t.Parallel()
	err := CheckUpdates(testJSONFile, &configData)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeGemini(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key: "id",
		Val: "revision-history",
		TokenDataEnd: "table",
		TextTokenData: "td",
		DateFormat: "2006/01/02",
		RegExp: "^20(\\d){2}/(\\d){2}/(\\d){2}$",
		CheckString: "2019/11/15",
		Path: "https://docs.gemini.com/rest-api/#revision-history"}
	_, err := HTMLScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeHuobi(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key: "id",
		Val: "change-log",
		TokenDataEnd: "h2",
		TextTokenData: "td",
		DateFormat: "2006.01.02 15:04",
		RegExp: "^20(\\d){2}.(\\d){2}.(\\d){2} (\\d){2}:(\\d){2}$",
		CheckString: "2019.12.27 19:00",
		Path: "https://huobiapi.github.io/docs/spot/v1/en/#change-log"}
	_, err := HTMLScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeCoinbasepro(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key: "id",
		Val: "changelog",
		TokenDataEnd: "ul",
		TextTokenData: "strong",
		DateFormat: "01/02/06",
		RegExp: "^(\\d){2}/(\\d){2}/(\\d){2}$",
		CheckString: "12/16/19",
		Path: "https://docs.pro.coinbase.com/#changelog"}
	_, err := HTMLScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitfinex(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{DateFormat: "2006-01-02",
		RegExp: `section-v-(2\d{3}-\d{1,2}-\d{1,2})`,
		Path:   "https://docs.bitfinex.com/docs/changelog"}
	_, err := HTMLScrapeBitfinex(&data)
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
	_, err := HTMLScrapeBitmex(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeHitBTC(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `newest version \d{1}.\d{1}`,
		Path: "https://api.hitbtc.com/"}
	_, err := HTMLScrapeHitBTC(&data)
	if err != nil {
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
	_, err := HTMLScrapeDefault(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBTSE(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp:        `^version: \d{1}.\d{1}.\d{1}`,
	Path:          "https://api.btcmarkets.net/openapi/info/index.yaml"}
	_, err := HTMLScrapeBTSE(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBTCMarkets(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp:        `^version: \d{1}.\d{1}.\d{1}`,
		Path:          "https://api.btcmarkets.net/openapi/info/index.yaml"}
	_, err := HTMLScrapeBTCMarkets(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitflyer(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "p",
		Key:           "",
		Val:           "",
		TokenDataEnd:  "h3",
		TextTokenData: "code",
		DateFormat:    "",
		RegExp:        `^https://api.bitflyer.com/v\d{1}/$`,
		Path:          "https://lightning.bitflyer.com/docs?lang=en"}
	_, err := HTMLScrapeBitflyer(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeANX(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `ANX Exchange API v\d{1}`,
		Path: "https://anxv3.docs.apiary.io/#reference/quickstart-catalog"}
	_, err := HTMLScrapeANX(&data)
	if err != nil {
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
	_, err := HTMLScrapePoloniex(&data)
	if err != nil {
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
	_, err := HTMLScrapeItBit(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLLakeBTC(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "div",
		Key:           "class",
		Val:           "flash-message",
		TokenDataEnd:  "h2",
		TextTokenData: "h1",
		DateFormat:    "",
		RegExp:        `APIv\d{1}`,
		Path:          "https://www.lakebtc.com/s/api_v2"}
	_, err := HTMLScrapeLakeBTC(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeExmo(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
		Path: "https://exmo.com/en/api/"}
	_, err := HTMLScrapeExmo(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLBitstamp(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{RegExp: `refer to the v\d{1} API for future references.`,
		Path: "https://www.bitstamp.net/api/"}
	_, err := HTMLScrapeBitstamp(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLKraken(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h3",
		Key:           "",
		Val:           "",
		TokenDataEnd:  "p",
		TextTokenData: "p",
		DateFormat:    "",
		RegExp:        `URL: https://api.kraken.com/\d{1}/private/Balance`,
		Path:          "https://www.kraken.com/features/api"}
	_, err := HTMLScrapeKraken(&data)
	if err != nil {
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
		DateFormat:    "",
		RegExp:        `revised-calls-\d{1}-\d{1}-\d{1}-gt-\d{1}-\d{1}-\d{1}`,
		Path:          "https://alphapoint.github.io/slate/#introduction"}
	_, err := HTMLScrapeAlphaPoint(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLYobit(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h2",
		Key:  "id",
		Path: "https://www.yobit.net/en/api/"}
	_, err := HTMLScrapeYobit(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLLocalBitcoins(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "div",
		Key:           "class",
		Val:           "col-md-12",
		TokenDataEnd:  "",
		TextTokenData: "",
		DateFormat:    "",
		RegExp:        `col-md-12([\s\S]*?)clearfix`,
		Path:          "https://localbitcoins.com/api-docs/"}
	_, err := HTMLScrapeLocalBitcoins(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestGetListsData(t *testing.T) {
	t.Parallel()
	if !CanUpdateTrello() {
		t.Skip()
	}	
	_, err := TrelloGetListsData(trelloBoardID)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewCard(t *testing.T) {
	t.Parallel()
	if !CanUpdateTrello() {
		t.Skip()
	}	
	fillData := CardFill{ListID: trelloListID,
		Name: "Exchange Updates"}
	err := TrelloCreateNewCard(&fillData)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewCheck(t *testing.T) {
	t.Parallel()
	if !CanUpdateTrello() {
		t.Skip()
	}	
	err := TrelloCreateNewCheck("Gemini")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	var exchCheck, updatedExch HTMLScrapingData
	for x := range testConfigData.Exchanges {
		if testConfigData.Exchanges[x].Name == "Exmo" {
			exchCheck = *testConfigData.Exchanges[x].Data.HTMLData
		}
	}
	info := ExchangeInfo{Name: "Exmo",
		CheckType: "HTML String Check",
		Data: &CheckData{HTMLData: &HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
			Path: "https://exmo.com/en/api/"},
		},
	}
	updatedExchs := Update("Exmo", testConfigData.Exchanges, info)
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
	a := CheckMissingExchanges(&testConfigData)
	if len(a) > len(exchange.Exchanges) {
		log.Println("invalid response")
	}
}

func TestGetChecklistItems(t *testing.T) {
	t.Parallel()
	if !CanUpdateTrello() {
		t.Skip()
	}	
	_, err := TrelloGetChecklistItems()
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateCheckItem(t *testing.T) {
	t.Parallel()
	if !CanUpdateTrello() {
		t.Skip()
	}	
	err := TrelloUpdateCheckItem("5dfc604fe901ac6a592e9b75", "Gemini 1", "incomplete")
	if err != nil {
		t.Error(err)
	}
}

func TestNameUpdates(t *testing.T) {
	t.Parallel()
	_, err := NameStateChanges("Gemini 2", "complete")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateFile(t *testing.T) {
	t.Parallel()
	err := UpdateFile(&configData, testJSONFile)
	if err != nil {
		t.Error(err)
	}
	realConf, err := ReadFileData(jsonFile)
	if err != nil {
		log.Fatal(err)
	}
	testConf, err := ReadFileData(testJSONFile)
	if err != nil {
		log.Fatal(err)
	}
	if !reflect.DeepEqual(realConf, testConf) {
		t.Log("test file update failed")
		t.Fail()
	}
}

func TestReadFileData(t *testing.T) {
	t.Parallel()
	_, err := ReadFileData(testJSONFile)
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