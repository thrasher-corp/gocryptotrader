# GoCryptoTrader package Datahistory Manager

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/datahistory_manager)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This datahistory_manager package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## What is the data history manager?
+ The data history manager is an engine subsystem responsible for ensuring that the candle/trade history in the range you define is synchronised to your database
+ It is a long running synchronisation task designed to not overwhelm resources and ensure that all data requested is accounted for and saved to the database
+ The data history manager is disabled by default and requires a database connection to function
  + It can be enabled either via a runtime param, config modification or via RPC command `enablesubsystem`
+ The data history manager accepts jobs from RPC commands
+ A job is defined in the `Database tables` section below
+ Jobs will be addressed by the data history manager at an interval defined in your config, this is detailed below in the `Application run time parameters` table below
+ Jobs will fetch data at sizes you request (which can cater to hardware limitations such as low RAM)
+ Jobs are completed once all data has been fetched/attempted to be fetched in the time range

## Current Features for the data history manager
+ Retrieval and storage of exchange API candle data
+ Retrieval and storage of exchange API trade data
+ Conversion of stored trade data into custom candle data
+ Conversion of stored candle data into custom candle data
+ Validation of stored candle data against exchange API data
  + Optionally can replace data when an issue is found on a customisable threshold
+ Validation of stored candle data against a secondary exchange's API data
+ Pausing and unpause jobs
+ Queue jobs via prerequisite jobs
+ GRPC command support for creating/modifying/checking jobs

## What are the requirements for the data history manager?
+ Ensure you have a database setup, you can read about that [here](/database)
+ Ensure you have run dbmigrate under `/cmd/dbmigrate` via `dbmigrate -command=up`, you can read about that [here](/database#create-and-run-migrations)
+ Ensure you have seeded exchanges to the database via the application dbseed under `/cmd/dbseed`, you can read about it [here](/cmd/dbseed)
+ Ensure you have the database setup and enabled in your config, this can also be seen [here](/database)
+ Data retrieval can only be made on exchanges that support it, see the readmes for [candles](/docs/OHLCV.md) and [trades](/exchanges/trade#exchange-support-table)
+ Read below on how to enable the data history manager and add data history jobs

## What is a data history job?
A job is a set of parameters which will allow GoCryptoTrader to periodically retrieve, convert or validate historical data. Its purpose is to break up the process of retrieving large sets of data for multiple currencies and exchanges into more manageable chunks in a "set and forget" style.
For a breakdown of what a job consists of and what each parameter does, please review the database tables and the cycle details below.

### What kind of data jobs are there?
A breakdown of each type is under the Add Jobs command list below

### What are the different job status types?

| Job Status | Description | Representative value |
| ---------- | ----------- | -------------------- |
| active | A job that is ready to processed | 0 |
| failed | The job has failed to retrieve/convert/validate the data you have specified. See the associated data history job results to understand why | 1 |
| complete | The job has successfully retrieved/converted/validated all data you have specified | 2 |
| removed | The job has been deleted. No data is removed, but the job can no longer be processed | 3 |
| missing data | The job is complete, however there is some missing data. See the associated data history job results to understand why | 4 |
| paused | The job has been paused and will not be processed. Either it has a prerequisite job that needs to be completed, or a user must unpause the job | 5 |

### How do I add a job?
+ First ensure that the data history monitor is enabled, you can do this via the config (see table `dataHistoryManager` under Config parameters below), via run time parameter (see table Application run time parameters below) or via the RPC command `enablesubsystem --subsystemname="data_history_manager"`
+ The simplest way of adding a new data history job is via the GCTCLI under `/cmd/gctcli`.
  + Modify the following example command to your needs: `.\gctcli.exe datahistory addjob savecandles --nickname=binance-spot-bnb-btc-1h-candles --exchange=binance --asset=spot --pair=BNB-BTC --interval=3600 --start_date="2020-06-02 12:00:00" --end_date="2020-12-02 12:00:00" --request_size_limit=10 --max_retry_attempts=3 --batch_size=3`

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

### Candle intervals and trade fetching
+ A candle interval is required for a job, even when fetching trade data. This is to appropriately break down requests into time interval chunks. However, it is restricted to only a small range of times. This is to prevent fetching issues as fetching trades over a period of days or weeks will take a significant amount of time. When setting a job to fetch trades, the allowable range is less than 4 hours and greater than 10 minutes.

## Job queuing and prerequisite jobs
You can add jobs which will be paused by default by using the `prerequisite` subcommand containing the associated job nickname. The prerequisite job will be checked to ensure it exists and has not yet completed and add the relationship.
+ Once you have set a prerequisite job, when the prerequisite job status is set to `complete`, the data history manager will search for any jobs which are pending its completion and update their status to `active`.
+ If the prerequisite job is deleted or fails, the upcoming job will _not_ be run.
+ Multiple jobs can use the same prerequisite job, but a job cannot have multiple prerequisites
  + Attempting to add a new prerequisite will overwrite the existing prerequisite
+ You can chain many queued jobs together allowing for automated and large scale data retrieval projects. For example:

| Job Type | Job Name | Description | Prerequisite Job |
| -------- | -------- | ----------- | ---------------- |
| savetrades | save-trades | Save jobs between 01-01-2021 and 01-02-2021 | |
| converttrades | convert-trades | Convert trades to 5m candles | save-trades |
| validatecandles | validate-candles | Ensure the converted trades match the exchange's API data | convert-trades |
| convertcandles | convert-candles-10m | Now that we have confidence in conversion, convert candle to 10m | validate-candles |
| convertcandles | convert-candles-25m | Now that we have confidence in conversion, convert candle to 25m | validate-candles |
| convertcandles | convert-candles-1d | Now that we have confidence in conversion, convert candle to 1d | validate-candles |

## Application run time parameters

| Parameter | Description | Example |
| ------ | ----------- | ------- |
| datahistorymanager | A boolean value which determines if the data history manager is enabled. Defaults to `false` | `-datahistorymanager=true` |


## Config parameters
### dataHistoryManager

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | If enabled will run the data history manager on startup | `true` |
| checkInterval | A golang `time.Duration` interval of when to attempt to fetch all active jobs' data | `15000000000` |
| maxJobsPerCycle | Allows you to control how many jobs are processed after the `checkInterval` timer finishes. Useful if you have many jobs, but don't wish to constantly be retrieving data | `5` |
| maxResultInsertions | When saving candle/trade results, loop it in batches of this number | `10000` |
| verbose | Displays some extra logs to your logging output to help debug | `false` |

## RPC commands
The below table is a summary of commands. For more details, view the commands in `/cmd/gctcli` or `/gctrpc/rpc.swagger.json`

| Command | Description |
| ------ | ----------- |
| AddJob | Shows a list of subcommands to add a new job type, detailed in the next table |
| GetDataHistoryJobDetails | Returns a job's details via its nickname or ID. Can optionally return an array of all run results |
| GetActiveDataHistoryJobs | Will return all jobs that have an `active` status |
| DeleteJob | Will remove a job for processing. Data is preserved in the database for later reference |
| GetDataHistoryJobsBetween | Returns all jobs, of all status types between the dates provided |
| GetDataHistoryJobSummary | Will return an executive summary of the progress of your job by nickname |
| PauseDataHistoryJob | Will set a job's status to paused |
| UnpauseDataHistoryJob | Will se a job's status to `active` |

### AddJob commands

| Command | Description | DataHistoryJobDataType |
| ------- | ----------- | ---------------------- |
| savecandles | Will fetch candle data from an exchange and save it to the database | 0 |
| savetrades | Will fetch trade data from an exchange and save it to the database | 1 |
| converttrades | Convert trades saved to the database to any candle resolution eg 30min | 2 |
| convertcandles | Convert candles saved to the database to a new resolution eg 1min -> 5min | 3 |
| validatecandles | Will compare database candle data with API candle data - useful for validating converted trades and candles | 4 |
| secondaryvalidatecandles | Will compare database candle data with a different exchange's API candle data | 5 |


## Database tables
The following is a screenshot of the relationship between relevant data history job tables
![image](https://user-images.githubusercontent.com/9261323/125889821-954b730a-01fd-4eb0-839d-3cc623178f20.png)

### datahistoryjob

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
| interval | A golang `time.Duration` representation of the candle interval to use | `30000000000` |
| data_type | The data type to fetch. See job types in table `AddJob commands` above | `0` |
| request_size | The number of candles to fetch. eg if `500`, the data history manager will break up the request into the appropriate timeframe to ensure the data history run interval will fetch 500 candles to save to the database | `500` |
| max_retries | For an interval period, the amount of attempts the data history manager is allowed to attempt to fetch data before moving onto the next period. This can be useful for determining whether the exchange is missing the data in that time period or, if just one failure of three, just means that the data history manager couldn't finish one request | `3` |
| batch_count | The number of requests to make when processing a job | `3` |
| status | A numerical representation for the status. See data history job status subsection | `0` |
| created | The date the job was created | `2020-01-01T13:33:37Z` |
| conversion_interval | When converting data as a job, this determines the resulting interval | `86400000000000` |
| overwrite_data | If data already exists, the setting allows you to overwrite it | `true` |
| secondary_exchange_id | For a `secondaryvalidatecandles` job, the exchange id of the exchange to compare data to | `bybit` |
| decimal_place_comparison | When validating API candles, this will round the data to the supplied decimal point to check for equality | `3` |
| replace_on_issue | When there is an issue validating candles for a `validatecandles` job, the API data will overwrite the existing candle data | `false` |

### datahistoryjobresult

| Field | Description | Example |
| ------ | ----------- | ------- |
| id | Unique ID of the job status | `deadbeef-dead-beef-dead-beef13371337` |
| job_id | The job ID being referenced | `deadbeef-dead-beef-dead-beef13371337` |
| result | If there is an error, it will be detailed here | `exchange missing candle data for 2020-01-01 13:37Z` |
| status | A numerical representation of the job result status. `1` is failed, `2` is complete and `4` is missing data | `2` |
| interval_start_time | The start date of the period fetched | `2020-01-01T13:33:37Z` |
| interval_end_time  | The end date of the period fetched | `2020-01-02T13:33:37Z` |
| run_time | The time the job was ran | `2020-01-03T13:33:37Z` |

### datahistoryjobrelations

| Field | Description | Example |
| ------ | ----------- | ------- |
| prerequisite_job_id | The job that must be completed before `job_id` can be run | `deadbeef-dead-beef-dead-beef13371337` |
| job_id | The job that will be run after `prerequisite_job_id` completes | `deadbeef-dead-beef-dead-beef13371337` |

### candle
The candle table also has relationships to data history jobs. Only the relevant columns are listed below:

| Field | Description | Example |
| ------ | ----------- | ------- |
| source_job_id | The source job id for where the candle data came from | `deadbeef-dead-beef-dead-beef13371337` |
| validation_job_id | When job id for what job validated the candle data | `deadbeef-dead-beef-dead-beef13371337` |
| validation_issues | If any discrepancies are found, the data will be written to the column | `issues found at 2020-07-08 00:00:00, Open api: 9262.62 db: 9262.69 diff: 3%, replacing database candle data with API data` |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
