# GoCryptoTrader package Fxmacrodata

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/currency/forexprovider/fxmacrodata)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This fxmacrodata package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for fxmacrodata

+ Fetches up to date currency data from [FXMacroData](https://fxmacrodata.com/)
+ Provides official-source macroeconomic, central-bank and release-calendar data

### How to enable

+ [Enable via configuration](https://github.com/thrasher-corp/gocryptotrader/tree/master/config#enable-currency-via-config-example)

+ Individual package example below:
```go
import (
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/fxmacrodata"
)

c := fxmacrodata.FXMacroData{}

// Define configuration
newSettings := base.Settings{
	Name:             "FXMacroData",
	Enabled:          true,
	Verbose:          false,
	RESTPollingDelay: time.Duration,
	APIKey:           "key",
	PrimaryProvider:  true,
}

c.Setup(newSettings)

mapstringfloat, err := c.GetRates("USD", "EUR,AUD")
// Handle error
```

### Testing

The unit and contract tests run entirely against local `httptest` servers and
need no credentials or network access. Two additional smoke tests are opt-in
because they reach the live FXMacroData API.

Enable them either by flipping the package-level bool variables at the top of
`fxmacrodata_test.go`, or by setting the matching environment variables:

| Bool variable | Environment variable | Purpose |
|---|---|---|
| `testLive` | `GCT_RUN_LIVE_TESTS=true` | Runs the public endpoint smoke test. No API key required. |
| `testAuth` | `GCT_RUN_FXMACRODATA_AUTH_TESTS=true` | Runs the authenticated endpoint smoke test. Requires an API key. |
| `testAPIKey` | `FXMACRODATA_API_KEY` (or `FXMD_API_KEY`) | Supplies the API key used by the authenticated smoke test. |

Each smoke test skips with an explanatory message when its toggle is unset, so
the default `go test ./...` run stays hermetic.

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
