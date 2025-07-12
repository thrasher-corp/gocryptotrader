# GoCryptoTrader package Mock

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/mock)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This mock package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Mock Testing Suite

## Current Features for mock
+ REST recording service
+ REST mock response server

### How to enable

+ Any exchange with mock testing will be enabled by default. This is done using build tags which are highlighted in the examples below via `//+build mock_test_off`. To disable and run live endpoint testing parse `-tags=mock_test_off` as a go test param.

## Mock test setup

+ Create two additional test files for the exchange. Examples are below:

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
	your_current_exchange_nameConfig.API.AuthenticatedSupport = true
	your_current_exchange_nameConfig.API.Credentials.Key = apiKey
	your_current_exchange_nameConfig.API.Credentials.Secret = apiSecret
	s.SetDefaults()
	s.Setup(&your_current_exchange_nameConfig)
	log.Printf(sharedtestvalues.LiveTesting, s.Name, s.API.Endpoints.URL)
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
	your_current_exchange_nameConfig.API.AuthenticatedSupport = true
	your_current_exchange_nameConfig.API.Credentials.Key = apiKey
	your_current_exchange_nameConfig.API.Credentials.Secret = apiSecret
	s.SetDefaults()
	s.Setup(&your_current_exchange_nameConfig)

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}

	s.HTTPClient = newClient
	s.API.Endpoints.URL = serverDetails

	log.Printf(sharedtestvalues.MockTesting, s.Name, s.API.Endpoints.URL)
	os.Exit(m.Run())
}

```

## Mock test storage

+ Under `testdata/http_mock` create a folder matching the name of your exchange. Then create a JSON file matching the name of your exchange with the following formatting:
```
{
	"routes": {
	}
}
```


## Recording a test result

+ Once the files `your_current_exchange_name_mock_test.go` and `your_current_exchange_name_live_test.go` along with the JSON file `testdata/http_mock/our_current_exchange_name/our_current_exchange_name.json` are created, go through each individual test function and add

```go
var s SomeExchange

func TestDummyTest(t *testing.T) {
	s.Verbose = true // This will show you some fancy debug output
	s.HTTPRecording = true // This will record the request and response payloads
	s.API.Endpoints.URL = apiURL // This will overwrite the current mock url at localhost
	s.API.Endpoints.URLSecondary = secondAPIURL // This is only if your API has multiple endpoints
	s.HTTPClient = http.DefaultClient // This will ensure that a real HTTPClient is used to record
	err := s.SomeExchangeEndpointFunction()
	// check error
}
```

+ This will store the request and results under the freshly created `testdata/http_mock/your_current_exchange/your_current_exchange.json`

## Validating

+ To check if the recording was successful, comment out recording and apiurl changes, then re-run test.

```go
var s SomeExchange

func TestDummyTest(t *testing.T) {
	s.Verbose = true // This will show you some fancy debug output
	// s.HTTPRecording = true // This will record the request and response payloads
	// s.API.Endpoints.URL = apiURL // This will overwrite the current mock url at localhost
	// s.API.Endpoints.URLSecondary = secondAPIURL // This is only if your API has multiple endpoints
	// s.HTTPClient = http.DefaultClient // This will ensure that a real HTTPClient is used to record
	err := s.SomeExchangeEndpointFunction()
	// check error
}
```

+ The payload should be the same.

## Considerations

+ Some functions require timestamps. Mock tests _must_ match the same request structure, so `time.Now()` will cause problems for mock testing.
	+ To address this, use the boolean variable `mockTests` to create a consistent date. An example is below.
```
	startTime := time.Now().Add(-time.Hour * 1)
	endTime := time.Now()
	if mockTests {
		startTime = time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
		endTime = time.Date(2020, 9, 2, 0, 0, 0, 0, time.UTC)
	}
```
+ Authenticated endpoints will typically require valid API keys and a signature to run successfully. Authenticated endpoints should be skipped. See an example below
```
	if mockTests {
		t.Skip("skipping authenticated function for mock testing")
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

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
