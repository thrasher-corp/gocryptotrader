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

### FOR ALL THE NEW EXCHANGES ADDED, TRELLO CHECKLIST WILL NEED TO BE MANUALLY UPDATED SO THAT APICHECKER CAN CHECK FOR THE UPDATES ON ITS NEXT RUN.

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

***1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB***
