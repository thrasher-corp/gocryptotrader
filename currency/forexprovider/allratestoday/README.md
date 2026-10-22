# GoCryptoTrader package Allratestoday

<img src="../../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/currency/forexprovider/allratestoday)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This allratestoday package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for allratestoday

+ Fetches up to date currency data from [AllRatesToday](https://allratestoday.com/)
+ Returns every rate for a base currency in one request
+ API key required (free trial, paid plans)

### How to enable

+ [Enable via configuration](../../../config/README.md#enable-currency-via-config-example)

+ Individual package example below:

```go
import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/allratestoday"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

c := allratestoday.AllRatesToday{}

// Define configuration
newSettings := base.Settings{
	Name:            "AllRatesToday",
	Enabled:         true,
	Verbose:         false,
	APIKey:          "key",
	PrimaryProvider: true,
}

c.Setup(newSettings)

mapstringfloat, err := c.GetRates("USD", "EUR,AUD")
// Handle error
```

### Testing

The unit tests run against local `httptest` servers and need no credentials or
network access.

## Donations

<img src="../../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
