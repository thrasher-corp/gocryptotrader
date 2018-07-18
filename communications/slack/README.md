# GoCryptoTrader package Slack

<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-/gocryptotrader/communications/slack)
[![Coverage Status](http://codecov.io/github/thrasher-/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-/gocryptotrader)


This slack package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progresss on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://gocryptotrader.herokuapp.com/)

## Slack Communications package

### What is Slack?

+ Slack is a code-centric collaboration hub that allows users to connect via an
app and share different types of data
+ Please visit: [Slack](https://slack.com/) for more information and account setup

### Current Features

+ Basic communication to your slack channel information includes:
  - Working status of bot
  - Recent ANX ticker
  - Current ANX orderbook

### How to enable

+ [Enable via configuration](https://github.com/thrasher-/gocryptotrader/tree/master/config#enable-communications-via-config-example)

+ Individual package example below:
```go
import (
"github.com/thrasher-/gocryptotrader/communications/slack"
"github.com/thrasher-/gocryptotrader/config"
)

s := new(slack.Slack)

// Define slack configuration
commsConfig := config.CommunicationsConfig{SlackConfig: config.SlackConfig{
  Name: "Slack",
	Enabled: true,
	Verbose: false,
	TargetChannel: "targetChan",
	VerificationToken: "slackGeneratedToken",
}}

s.Setup(commsConfig)
err := s.Connect
// Handle error
```

Once the bot has started you can interact with the bot using these commands
via Slack:

```
!status 		- Displays current working status of bot
!help 			- Displays help text
!settings		- Displays current settings
!ticker			- Displays recent ANX ticker
!portfolio	- Displays portfolio data
!orderbook	- Displays current ANX orderbook
```

### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB***

