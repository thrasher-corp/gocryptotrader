# GoCryptoTrader package Datahistory manager

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/datahistory_manager)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This datahistory_manager package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Current Features for Datahistory manager
+ The data history manager is an engine subsystem responsible for ensuring that the candle/trade history in the range you define is synchronised to your database
+ It is a long running synchronisation task designed to not overwhelm resources and ensure that all data requested is accounted for and saved to the database
+ The data history manager is disabled by default and requires a database connection to function
  + It can be enabled either via a runtime param, config modification or via RPC command `enablesubsystem`
+ The data history manager accepts jobs from RPC commands
+ A job is defined in the `Database tables` section below
+ Jobs will be addressed by the data history manager at an interval defined in your config, this is detailed below in the `Application run time parameters` table below
+ Jobs will fetch data at sizes you request (which can cater to hardware limitations such as low RAM)
+ Jobs are completed once all data has been fetched/attempted to be fetched in the time range

## What are the prerequisites?
+ Ensure you have a database setup, you can read about that [here](/database)
+ Ensure you have run dbmigrate under `/cmd/dbmigrate` via `dbmigrate -command=up`, you can read about that [here](/database#create-and-run-migrations)
+ Ensure you have seeded exchanges to the database via the application dbseed under `/cmd/dbseed`, you can read about it [here](/cmd/dbseed)
+ Ensure you have the database setup and enabled in your config, this can also be seen [here](/database)
+ Data retrieval can only be made on exchanges that support it, see the readmes for [candles](/docs/OHLCV.md) and [trades](/exchanges/trade#exchange-support-table)
+ Read below on how to enable the data history manager and add data history jobs

## What is a data history job?
A job is a set of parameters which will allow GoCryptoTrader to periodically retrieve historical data. Its purpose is to break up the process of retrieving large sets of data for multiple currencies and exchanges into more manageable chunks in a "set and forget" style.
For a breakdown of what a job consists of and what each parameter does, please review the database tables and the cycle details below.

## What happens during a data history cycle?
+ Once the checkInterval ticker timer has finished, the data history manager will process all jobs considered `active`.
+ A job's start and end time is broken down into intervals defined by the `interval` variable of a job. For a job beginning `2020-01-01` to `2020-01-02` with an interval of one hour will create 24 chunks to retrieve
+ The number of intervals it will then request from an API is defined by the `RequestSizeLimit`. A `RequestSizeLimit` of 2 will mean when processing a job, the data history manager will fetch 2 hours worth of data
+ When processing a job the `RunBatchLimit` defines how many `RequestSizeLimits` it will fetch. A `RunBatchLimit` of 3 means when processing a job, the history manager will fetch 3 lots of 2 hour chunks from the API in a run of a job
+ If the data is successfully retrieved, that chunk will be considered `complete` and saved to the database
+ The `MaxRetryAttempts` defines how many times the data history manager will attempt to fetch a chunk of data before flagging it as `failed`.
  + A chunk is only attempted once per processing time.
  + If it fails, the next attempt will be after the `checkInterval` has finished again.
  + The errors for retrieval failures are stored in the database, allowing you to understand why a certain chunk of time is unavailable (eg exchange downtime and missing data)
+ All results are saved to the database, the data history manager will analyse all results and ready jobs for the next round of processing

## How do I add one?
+ First ensure that the data history monitor is enabled, you can do this via the config (see table `dataHistoryManager` under Config parameters below), via run time parameter (see table Application run time parameters below) or via the RPC command `enablesubsystem --subsystemname="data_history_manager"`
+ The simplest way of adding a new data history job is via the GCTCLI under `/cmd/gctcli`.
  + Modify the following example command to your needs: `.\gctcli.exe datahistory upsertjob --nickname=binance-spot-bnb-btc-1h-candles --exchange=binance --asset=spot --pair=BNB-BTC --interval=3600 --start_date="2020-06-02 12:00:00" --end_date="2020-12-02 12:00:00" --request_size_limit=10 --data_type=0 --max_retry_attempts=3 --batch_size=3`

### Candle intervals and trade fetching
+ A candle interval is required for a job, even when fetching trade data. This is to appropriately break down requests into time interval chunks. However, it is restricted to only a small range of times. This is to prevent fetching issues as fetching trades over a period of days or weeks will take a significant amount of time. When setting a job to fetch trades, the allowable range is less than 4 hours and greater than 10 minutes.

### Application run time parameters

| Parameter | Description | Example |
| ------ | ----------- | ------- |
| datahistorymanager | A boolean value which determines if the data history manager is enabled. Defaults to `false` | `-datahistorymanager=true` |


### Config parameters
#### dataHistoryManager

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | If enabled will run the data history manager on startup | `true` |
| checkInterval | A golang `time.Duration` interval of when to attempt to fetch all active jobs' data | `15000000000` |
| maxJobsPerCycle | Allows you to control how many jobs are processed after the `checkInterval` timer finishes. Useful if you have many jobs, but don't wish to constantly be retrieving data | `5` |
| verbose | Displays some extra logs to your logging output to help debug | `false` |

### RPC commands
The below table is a summary of commands. For more details, view the commands in `/cmd/gctcli` or `/gctrpc/rpc.swagger.json`

| Command | Description |
| ------ | ----------- |
| UpsertDataHistoryJob | Updates or Inserts a job to the manager and database |
| GetDataHistoryJobDetails | Returns a job's details via its nickname or ID. Can optionally return an array of all run results |
| GetActiveDataHistoryJobs | Will return all jobs that have an `active` status |
| DeleteJob | Will remove a job for processing. Data is preserved in the database for later reference |
| GetDataHistoryJobsBetween | Returns all jobs, of all status types between the dates provided |
| GetDataHistoryJobSummary | Will return an executive summary of the progress of your job by nickname |

### Database tables
#### datahistoryjob

| Field | Description | Example |
| ------ | ----------- | ------- |
| id | Unique ID of the job. Generated at creation | `deadbeef-dead-beef-dead-beef13371337` |
| nickname | A custom name for the job that is unique for lookups | `binance-xrp-doge-2017` |
| exchange_name_id | The exchange id to fetch data from. The ID should be generated via `/cmd/dbmigrate`. When creating a job, you only need to provide the exchange name  | `binance` |
| asset | The asset type of the data to be fetching | `spot` |
| base | The currency pair base of the data to be fetching | `xrp` |
| quote | The currency pair quote of the data to be fetching | `doge` |
| start_time | When to begin fetching data | `01-01-2017T13:33:37Z` |
| end_time | When to finish fetching data | `01-01-2018T13:33:37Z`  |
| interval | A golang `time.Duration` representation of the candle interval to use. | `30000000000` |
| data_type | The data type to fetch. `0` is candles and `1` is trades | `0` |
| request_size | The number of candles to fetch. eg if `500`, the data history manager will break up the request into the appropriate timeframe to ensure the data history run interval will fetch 500 candles to save to the database | `500` |
| max_retries | For an interval period, the amount of attempts the data history manager is allowed to attempt to fetch data before moving onto the next period. This can be useful for determining whether the exchange is missing the data in that time period or, if just one failure of three, just means that the data history manager couldn't finish one request | `3` |
| batch_count | The number of requests to make when processing a job | `3` |
| status | A numerical representation for the status. `0` is active, `1` is failed `2` is complete, `3` is removed and `4` is missing data | `0` |
| created | The date the job was created. | `2020-01-01T13:33:37Z` |

#### datahistoryjobresult
| Field | Description | Example |
| ------ | ----------- | ------- |
| id | Unique ID of the job status | `deadbeef-dead-beef-dead-beef13371337` |
| job_id | The job ID being referenced | `deadbeef-dead-beef-dead-beef13371337` |
| result | If there is an error, it will be detailed here | `exchange missing candle data for 2020-01-01 13:37Z` |
| status | A numerical representation of the job result status. `1` is failed, `2` is complete and `4` is missing data | `2` |
| interval_start_time | The start date of the period fetched | `2020-01-01T13:33:37Z` |
| interval_end_time  | The end date of the period fetched | `2020-01-02T13:33:37Z` |
| run_time | The time the job was ran | `2020-01-03T13:33:37Z` |

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
