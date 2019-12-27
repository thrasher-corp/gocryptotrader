package main

import (
	"testing"
)

func TestCheckExistingExchanges(t *testing.T) {
	_, _, err := CheckExistingExchanges("Updates", "Kraken")
	if err != nil {
		t.Error(err)
	}
}

func TestCheckChangeLog(t *testing.T) {
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "class",
		Val:           "header-scroll",
		TokenDataEnd:  "p",
		TextTokenData: "",
		DateFormat:    "2006-01-02",
		RegExp:        `section-v-(2\d{3}-\d{1,2}-\d{1,2})`,
		Path:          "https://docs.bitfinex.com/docs/changelog"}
	_, err := CheckChangeLog(data)
	if err != nil {
		t.Error(err)
	}
}

func TestAdd(t *testing.T) {
	data := HTMLScrapingData{TokenData: "h2",
		Key:           "class",
		Val:           "text-subtitle mt48",
		TokenDataEnd:  "h4",
		TextTokenData: "p",
		DateFormat:    "",
		RegExp:        `refer to the v\d{1} API for future references.`,
		Path:          "https://www.bitstamp.net/api/"}
	err := Add("Bitstamp", htmlScrape, data.Path, data)
	if err != nil {
		t.Error(err)
	}
}

func TestCheckUpdates(t *testing.T) {
	_, err := CheckUpdates(jsonFile)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeBitfinex(t *testing.T) {
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "class",
		Val:           "header-scroll",
		TokenDataEnd:  "p",
		TextTokenData: "",
		DateFormat:    "2006-01-02",
		RegExp:        `section-v-(2\d{3}-\d{1,2}-\d{1,2})`,
		Path:          "https://docs.bitfinex.com/docs/changelog"}
	_, err := HTMLScrapeBitfinex(data)
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
	_, err := HTMLScrapeBitmex(data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeHitBTC(t *testing.T) {
	data := HTMLScrapingData{TokenData: "h1",
		Key:           "id",
		Val:           "about-companyname-api",
		TokenDataEnd:  "h2",
		TextTokenData: "p",
		DateFormat:    "",
		RegExp:        `newest version \d{1}.\d{1}`,
		Path:          "https://api.hitbtc.com/"}
	_, err := HTMLScrapeHitBTC(data)
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
	_, err := HTMLScrapeBTSE(data)
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
	_, err := HTMLScrapeBTCMarkets(data)
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
	_, err := HTMLScrapeBitflyer(data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLScrapeANX(t *testing.T) {
	data := HTMLScrapingData{TokenData: "div",
		Key:           "id",
		Val:           "applicationHeaderContainer",
		TokenDataEnd:  "script",
		TextTokenData: "",
		DateFormat:    "",
		RegExp:        `ANX Exchange API v\d{1}`,
		Path:          "https://anxv3.docs.apiary.io/#reference/quickstart-catalog"}
	_, err := HTMLScrapeANX(data)
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
	_, err := HTMLScrapePoloniex(data)
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
	_, err := HTMLScrapeItBit(data)
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
	_, err := HTMLScrapeLakeBTC(data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLExmo(t *testing.T) {
	data := HTMLScrapingData{TokenData: "div",
		Key:           "class",
		Val:           "row section_inner content_single",
		TokenDataEnd:  "p",
		TextTokenData: "p",
		DateFormat:    "",
		RegExp:        `Last updated on {a`,
		Path:          "https://exmo.com/en/api/"}
	_, err := HTMLScrapeExmo(data)
	if err != nil {
		t.Error(err)
	}
}

func TestHTMLBitstamp(t *testing.T) {
	data := HTMLScrapingData{TokenData: "h2",
		Key:           "class",
		Val:           "text-subtitle mt48",
		TokenDataEnd:  "h4",
		TextTokenData: "p",
		DateFormat:    "",
		RegExp:        `refer to the v\d{1} API for future references.`,
		Path:          "https://www.bitstamp.net/api/"}
	_, err := HTMLScrapeBitstamp(data)
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
	_, err := HTMLScrapeKraken(data)
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
