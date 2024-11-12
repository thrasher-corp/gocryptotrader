# GoCryptoTrader package Apichecker

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/cmd/apichecker)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This apichecker package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for apichecker

+ Checks for API updates
+ Can automatically update Trello checklist for the updates required
+ Supports trello integration

#### This tool tracks changes in exchange API documentation
#### Keeps track of all the updates using the GoCryptoTrader trello board

Be aware, this tool will:
- Automatically update the live trello board if API keys and trello information are provided.
- Automatically update the main json updates file, however a backup of the copy before the updates will be stored.

## Usage

+ To run a real check for updates, parse Trello API info as flags or add them to the updates.json file and use the following command from apichecker folder in GCT:

###### Linux/OSX
GoCryptoTrader is built using [Go Modules](https://github.com/golang/go/wiki/Modules) and requires Go 1.11 or above
Using Go Modules you now clone this repository **outside** your GOPATH

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader/cmd/apichecker
go build
./apichecker
```

###### Windows

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader\cmd\apichecker
go build && apichecker.exe
```

+ Upon addition of a new exchange, to update Trello checklist and to add the exchange to updates.json the following would need to be done:

###### HTML Scraping method:
HTMLScrapingData is a struct which contains the necessary information to scrape data from the given path website. Not all the elements of HTMLScrapingData are necessary, its all dependent on site where information is being extracted from. Regexp is used to capture necessary bits of data using r.FindString() where r is the declared regular expression. If update dates data is available, DateFormat is used to convert the dates to a more standard format which can then be used for further comparisons of which update is most recent.
```go
func TestAdd(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "h1",
		Key:             "id",
		Val:             "revision-history",
		TokenDataEnd:    "table",
		TextTokenData:   "td",
		DateFormat:      "2006/01/02",
		RegExp:          "^20(\\d){2}/(\\d){2}/(\\d){2}$",
		CheckString:     "2019/11/15",
		Path:            "https://docs.gemini.com/rest-api/#revision-history"}
	err := Add("Gemini", htmlScrape, data.Path, data, true, &testConfigData)
	if err != nil {
		t.Error(err)
	}
}
```

###### Github SHA Check Method:
```go
func TestAdd(t *testing.T) {
	t.Parallel()
	data := GithubData{Repo: "LBank-exchange/lbank-official-api-docs"}
	err := Add("Lbank", github, fmt.Sprintf(githubPath, data.Repo), data, false, &configData)
	if err != nil {
		t.Error(err)
	}
}
```

###### Add using flags:
```bash
apichecker.exe -add=true -key=id -val=revision-history -tokendata=h1 -tokendataend=table -texttokendata=td -dateformat=2006/01/02 -checktype="HTML String Check" -regexp="^20(\d){2}/(\d){2}/(\d){2}$" -path="https://docs.gemini.com/rest-api/#revision-history" -exchangename=Gemini
```

+ If all the authentication variables for trello are set trello checklist will be automatically updated with the format of 'Exchange Name (integer of how many updates have been released since the exchange API was last worked on):

- To acquire your trello key and access token please login into trello using the following link and follow the steps: https://trello.com/app-key

- To acquire BoardID, ListID, CardID and ChecklistID inbuilt functions can be used such as trelloGetAllLists()

- To create a new list, card, checklist, and to populate the the checklist --create flag can be used.

- To create a new check within a checklist, an inbuilt function within apichecker can be used: trelloCreateNewCheck

- For the first time running the application & to create a list, card and checklist use the following:
```bash
apichecker.exe --create -apikey="insertkeyhere" -apitoken="inserttokenhere" -boardname="insertboardnamehere"
```


### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-corp/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
