# GoCryptoTrader package Charts

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/charts)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This charts package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for charts

#### Current Features for chart package

+ Generate Timeseries Chat from OHLCV Data
+ Generate Standard chart from time/value Data
+ Output to HTML
+ Serve over HTTP
+ GCTScript links

+ Coding example

##### Go
```go
import "github.com/thrasher-corp/gocryptotrader/charts"

func genIntervalData(totalCandles int) []IntervalData {
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]IntervalData, totalCandles)
	out[0] = IntervalData{Timestamp: start.Format("2006-01-02"), Value: 0}
	for x := 1; x &lt; totalCandles; x++ {
		out[x] = IntervalData{
			Timestamp: start.Add(time.Hour * 24 * time.Duration(x)).Format("2006-01-02"),
			Value:     out[x-1].Value + rand.Float64(),
		}
	}

	return out
}


func TestCharts(t *testing.T) {
	c := &Chart{
		template:     "timeseries.tmpl",
	}

	c.Data.Data = genIntervalData(365)
	_,err := c.ToFile().Generate()

	if err != nil {
	    t.Fatal(err)
	}
}
```

##### GCTScript
```go
exch := import("exchange")
t := import("times")
charts := import("charts")

load := func() {
    // define your start and end within reason.
    start := t.date(2017, 8 , 17, 0 , 0 , 0, 0)
    end := t.add_date(start, 0, 6 , 0)

    // This fetches the ohlcv
    ohlcvData := exch.ohlcv("binance", "BTC-USDT", "-", "spot", start, end, "1d")
    charts.gen("chart", true, ohlcvData.candles)
}

load()
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
