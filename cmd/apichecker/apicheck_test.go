package main

import (
	"log"
	"testing"
)

func TestCheckExistingExchanges(t *testing.T) {
	_, _, err := CheckExistingExchanges(testJSONFile, "Kraken")
	if err != nil {
		t.Error(err)
	}
}

func TestCheckChangeLog(t *testing.T) {
	data := HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
		Path: "https://exmo.com/en/api/"}
	a, err := CheckChangeLog(&data)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestAdd(t *testing.T) {
	data := HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
		Path: "https://exmo.com/en/api/"}
	err := Add(testJSONFile, "Exmo", htmlScrape, data.Path, data, true)
	if err != nil {
		t.Error(err)
	}
}

func TestCheckUpdates(t *testing.T) {
	err := CheckUpdates(testJSONFile)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitfinex(t *testing.T) {
	data := HTMLScrapingData{DateFormat: "2006-01-02",
		RegExp: `section-v-(2\d{3}-\d{1,2}-\d{1,2})`,
		Path:   "https://docs.bitfinex.com/docs/changelog"}
	_, err := HTMLScrapeBitfinex(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitmex(t *testing.T) {
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
	data := HTMLScrapingData{RegExp: `newest version \d{1}.\d{1}`,
		Path: "https://api.hitbtc.com/"}
	_, err := HTMLScrapeHitBTC(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeDefault(t *testing.T) {
	data := HTMLScrapingData{TokenData: "h3",
		Key:           "id",
		Val:           "change-change",
		TokenDataEnd:  "section",
		TextTokenData: "p",
		DateFormat:    "2006-01-02",
		RegExp:        "(2\\d{3}-\\d{1,2}-\\d{1,2})",
		CheckString:   "2019-04-28",
		Path:          "https://www.okcoin.com/docs/en/#change-change"}
	a, err := HTMLScrapeDefault(&data)
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBTSE(t *testing.T) {
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "btse-spot-api",
		TokenDataEnd:  "blockquote",
		TextTokenData: "h1",
		DateFormat:    "",
		RegExp:        `^BTSE Spot API v(\d){1}.(\d){1}$`,
		Path:          "https://www.btse.com/apiexplorer/spot/#btse-spot-api"}
	_, err := HTMLScrapeBTSE(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBTCMarkets(t *testing.T) {
	data := HTMLScrapingData{TokenData: "",
		Key:           "",
		Val:           "",
		TokenDataEnd:  "",
		TextTokenData: "",
		DateFormat:    "",
		RegExp:        `^version: \d{1}.\d{1}.\d{1}`,
		Path:          "https://api.btcmarkets.net/openapi/info/index.yaml"}
	_, err := HTMLScrapeBTCMarkets(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitflyer(t *testing.T) {
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
	data := HTMLScrapingData{RegExp: `ANX Exchange API v\d{1}`,
		Path: "https://anxv3.docs.apiary.io/#reference/quickstart-catalog"}
	_, err := HTMLScrapeANX(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLPoloniex(t *testing.T) {
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
	data := HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
		Path: "https://exmo.com/en/api/"}
	a, err := HTMLScrapeExmo(&data)
	log.Println(a[0])
	t.Log(a)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLBitstamp(t *testing.T) {
	data := HTMLScrapingData{RegExp: `refer to the v\d{1} API for future references.`,
		Path: "https://www.bitstamp.net/api/"}
	_, err := HTMLScrapeBitstamp(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLKraken(t *testing.T) {
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
	data := HTMLScrapingData{TokenData: "h2",
		Key:  "id",
		Path: "https://www.yobit.net/en/api/"}
	_, err := HTMLScrapeYobit(&data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLLocalBitcoins(t *testing.T) {
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
	_, err := GetListsData("5bd11e6998c8507ebbbec4fa")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewCard(t *testing.T) {
	fillData := CardFill{ListID: "5d75f87cf0aa430d0bf4f029",
		Name: "Exchange Updates"}
	err := CreateNewCard(fillData)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewCheck(t *testing.T) {
	err := CreateNewCheck("Gemini")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdate(t *testing.T) {
	finalResp, _, err := CheckExistingExchanges(testJSONFile, "Exmo")
	if err != nil {
		t.Error(err)
	}
	info := ExchangeInfo{Name: "Exmo",
		CheckType: "HTML String Check",
		Data: &CheckData{HTMLData: &HTMLScrapingData{RegExp: `Last updated on [\s\S]*, 20\d{2}`,
			Path: "https://exmo.com/en/api/"},
		},
	}
	Update("Exmo", finalResp, info)
}

func TestCheckMissingExchanges(t *testing.T) {
	_, err := CheckMissingExchanges(testJSONFile)
	if err != nil {
		t.Error(err)
	}
}

func TestGetChecklistItems(t *testing.T) {
	_, err := GetChecklistItems()
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateCheckItem(t *testing.T) {
	err := UpdateCheckItem("5dfc604fe901ac6a592e9b75", "Gemini 1", "incomplete")
	if err != nil {
		t.Error(err)
	}
}

func TestNameUpdates(t *testing.T) {
	_, err := NameStateChanges("Gemini 2", "complete")
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTestFile(t *testing.T) {
	err := UpdateTestFile("testupdates.json")
	if err != nil {
		t.Error(err)
	}
}
