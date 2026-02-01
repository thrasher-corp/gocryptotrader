# GoCryptoTrader package Telegram

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/communications/telegram)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This telegram package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Telegram Communications package

### What is telegram?

+ Telegram is a cloud-based instant messaging and voice over IP service
developed by Telegram Messenger LLP
+ Please visit: [Telegram](https://telegram.org/) for more information

### Current Features

+ Creation of bot that can retrieve
	- Bot status

	### How to enable

	+ [Enable via configuration](https://github.com/thrasher-corp/gocryptotrader/tree/master/config#enable-communications-via-config-example)

	+ See the individual package example below. NOTE: For privacy considerations, it's not possible to directly request a user's ID through the 
	Telegram Bot API unless the user interacts first. The user must message the bot directly. This allows the bot to identify and save the user's ID. 
	If this wasn't set initially, the user's ID will be stored by this package following a successful authentication when any supported command is issued.
	
	```go
	import (
		"github.com/thrasher-corp/gocryptotrader/communications/base"
		"github.com/thrasher-corp/gocryptotrader/communications/telegram"
	)

	t := new(telegram.Telegram)

	// Define Telegram configuration
	commsConfig := &base.CommunicationsConfig{
		TelegramConfig: base.TelegramConfig{
			Name:              "Telegram",
			Enabled:           true,
			Verbose:           false,
			VerificationToken: "token",
			AuthorisedClients: map[string]int64{"pepe": 0}, // 0 represents a placeholder for the user's ID, see note above for more info.
		},
	}

	t.Setup(commsConfig)
	err := t.Connect
	// Handle error
	```

+ Once the bot has started you can interact with the bot using these commands
via Telegram:

```
/start			- Will authenticate your ID
/status			- Displays the status of the bot
/help			- Displays current command list
```

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
