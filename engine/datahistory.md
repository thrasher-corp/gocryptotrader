# GoCryptoTrader package Datahistory

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/datahistory)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This datahistory package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for Datahistory
+ The data history manager is responsible for ensuring that the candle/trade history in the range you define is synchronised to your database
+ It is a long running synchronisation task designed to not overwhelm resources and ensure that all data requested is accounted for and saved to the database
+ The data history manager is disabled by default and requires a database connection to function
+ The data history manager accepts jobs from RPC commands
  + A job is defined in the `jobs` table below
  + Jobs will be address by the data history manager at an interval defined in your config, this is detailed below in the `datahistory` table below
  + Jobs will fetch data at sizes you request (which can cater to hardware limitations such as low RAM)
  + Jobs are completed once all data has been fetched/attempted to be fetched in the time range


### datahistory

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | If enabled will create a REST server which will listen to commands on the listen address | `true` |
| checkInterval | A golang time.Duration interval of when to attempt to fetch job data | `15000000000` |

### jobs

| Field | Description | Example |
| ------ | ----------- | ------- |
| ID | Unique ID of the job. Generated at creation | `deadbeef-dead-beef-dead-beef13371337` |
| Nickname | A custom name for the job that is unique for lookups | `binance-xrp-doge-2017` |
| Exchange | The exchange to fetch data from | `binance` |
| Asset | The asset type of the data to be fetching | `spot` |
| Pair | The currency pair of the data to be fetching | `xrp-doge` |
| StartDate | When to begin fetching data | `01-01-2017` |
| EndDate | When to finish fetching data. If `-` the data history manager will continuously fetch data | `31-12-2017` |
| DataType | The data type to fetch. Can be `candle` or `trade` | `candle` |
| RequestSizeLimit | The number of candles to fetch. eg if `500`, the data history manager will break up the request into the appropriate timeframe to ensure the data history run interval will fetch 500 candles to save to the database | `500` |
| Interval | The candle size | `1d` |
| MaxRetryAttempts | For an interval period, the amount of attempts the data history manager is allowed to attempt to fetch data before moving onto the next period. This can be useful for determining whether the exchange is missing the data in that time period or, if just one failure of three, just means that the data history manager couldn't finish one request | `3` |
| Status | Can be `active`, `removed`, `failed` or `complete` | - |

### job interval status
| Field | Description | Example |
| ------ | ----------- | ------- |
| ID | Unique ID of the job status | `deadbeef-dead-beef-dead-beef13371337` |
| JobID | The job ID being referenced | `deadbeef-dead-beef-dead-beef13371337` |
| Interval Start date | The starting period of the job fetch attempt | |
| Interval End date | The ending period o fthe job fetch attempt | |
| Status | Can be `failed` or `complete` | `complete` |
| Date | The date the fetch was attempted | `1-3-3337` |

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
