# GoCryptoTrader package Mock

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/mock)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This Mock package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Mock Testing Suite

### Current Features

+ REST recording service 
+ REST mock response server

### How to enable

+ Mock testing is enabled by default in some exchanges; to disable and run live endpoint testing parse -tags=mock_test_off as a go test param.

+ To record a live endpoint create two files for an exchange.

### file one - your_current_exchange_name_live_test.go

```go
//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package your_current_exchange_name

import (
	"os"
	"testing"
	"log"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	your_current_exchange_nameConfig, err := cfg.GetExchangeConfig("your_current_exchange_name")
	if err != nil {
		log.Fatal("your_current_exchange_name Setup() init error", err)
	}
	your_current_exchange_nameConfig.AuthenticatedAPISupport = true
	your_current_exchange_nameConfig.APIKey = apiKey
	your_current_exchange_nameConfig.APISecret = apiSecret
	l.SetDefaults()
	l.Setup(&your_current_exchange_nameConfig)
	log.Printf(sharedtestvalues.LiveTesting, l.Name, l.APIUrl)
	os.Exit(m.Run())
}
```

### file two - your_current_exchange_name_mock_test.go

```go
//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package your_current_exchange_name

import (
	"os"
	"testing"
	"log"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const mockfile = "../../testdata/http_mock/your_current_exchange_name/your_current_exchange_name.json"

var mockTests = true

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	your_current_exchange_nameConfig, err := cfg.GetExchangeConfig("your_current_exchange_name")
	if err != nil {
		log.Fatal("your_current_exchange_name Setup() init error", err)
	}
	your_current_exchange_nameConfig.AuthenticatedAPISupport = true
	your_current_exchange_nameConfig.APIKey = apiKey
	your_current_exchange_nameConfig.APISecret = apiSecret
	l.SetDefaults()
	l.Setup(&your_current_exchange_nameConfig)

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}

	g.HTTPClient = newClient
	g.APIUrl = serverDetails

	log.Printf(sharedtestvalues.MockTesting, l.Name, l.APIUrl)
	os.Exit(m.Run())
}

```

+ Once those files are completed go through each invidual test function and add

```go
var s SomeExchange

func TestDummyTest(t *testing.T) {
    s.APIURL = exchangeDefaultURL // This will overwrite the current mock url at localhost
    s.Verbose = true // This will show you some fancy debug output
    s.HTTPRecording = true // This will record the request and response payloads

    err := s.SomeExchangeEndpointFunction()
    // check error
}
```

+ After this is completed it should populate a new mocktest.json file for you with the relavent payloads in testdata
+ To check if the recording was successful, comment out recording and apiurl changes, then re-run test.

```go
var s SomeExchange

func TestDummyTest(t *testing.T) {
    // s.APIURL = exchangeDefaultURL // This will overwrite the current mock url at localhost
    s.Verbose = true // This will show you some fancy debug output
    // s.HTTPRecording = true // This will record the request and response payloads

    err := s.SomeExchangeEndpointFunction()
    // check error
}
```

+ The payload should be the same.

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

