# GoCryptoTrader package Apichecker

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/cmd/apichecker)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This apichecker package is part of the GoCryptoTrader codebase.

#### This tool tracks changes in exchange API documentation
#### Keeps track of all the updates using the GoCryptoTrader trello board

Be aware, this tool will:
- Automatically update the live trello board if API keys and trello information are provided.
- Automatically update the main json updates file, however a backup of the copy before the updates will be stored.

## This is still in active development

You can track ideas, planned features and what's in progresss on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for apichecker

+ Checks for API updates
+ Can automatically update Trello checklist for the updates required

## Usage

+ To run a real check for updates, parse Trello API info as flags or add them to the updates.json file and use the following command from apichecker folder in GCT:
```bash
go build && apichecker.exe --verbose
```

+ Upon addition of a new exchange, to update Trello checklist and to add the exchange to updates.json the following would need to be done:

#### HTML Scraping method:
```go
func TestAdd(t *testing.T) {
	t.Parallel()
	data := HTMLScrapingData{TokenData: "div",
		Key:    "class",
		Val:    "col-md-12",
		RegExp: "col-md-12([\\s\\S]*?)clearfix",
		Path:   "https://localbitcoins.com/api-docs/"}
	err := Add("LocalBitcoins", htmlScrape, data.Path, data, true, &testConfigData)
	if err != nil {
		t.Error(err)
    }
}
```

#### Github SHA Check Method:
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

+ If all the authentication variables for trello are set trello checklist will be automatically updated with the format of 'Exchange Name (integer of how many updates have been released since the exhange API was last worked on):
```go
func NameStateChanges(currentName, currentState string) (string, error) {
	r, err := regexp.Compile(`[\s\S]* \d{1}$`) // nolint: gocritic
	if err != nil {
		return "", err
	}
	var tempNumber int64
	var finalNumber string
	if r.MatchString(currentName) {
		stringNum := string(currentName[len(currentName)-1])
		tempNumber, err = strconv.ParseInt(stringNum, 10, 64)
		if err != nil {
			return "", err
		}
		if tempNumber != 1 || currentState != complete {
			tempNumber++
		} else {
			tempNumber = 1
		}
		finalNumber = strconv.FormatInt(tempNumber, 10)
	}
	byteNumber := []byte(finalNumber)
	byteName := []byte(currentName)
	byteName = byteName[:len(byteName)-1]
	byteName = append(byteName, byteNumber[0])
	return string(byteName), nil
}
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

******
