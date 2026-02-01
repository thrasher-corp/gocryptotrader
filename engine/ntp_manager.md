# GoCryptoTrader package Ntp Manager

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/ntp_manager)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This ntp_manager package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for Ntp Manager
+ The NTP manager subsystem is used highlight discrepancies between your system time and specified NTP server times
+ It is useful for debugging and understanding why a request to an exchange may be rejected
+ The NTP manager cannot update your system clock, so when it does alert you of issues, you must take it upon yourself to change your system time in the event your requests are being rejected for being too far out of sync
+ In order to modify the behaviour of the NTP manager subsystem, you can edit the following inside your config file under `ntpclient`:

### ntpclient

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | An integer value representing whether the NTP manager is enabled. It will warn you of time sync discrepancies on startup with a value of 0 and will alert you periodically with a value of 1. A value of -1 will disable the manager  |  `1` |
| pool | A string array of the NTP servers to check for time discrepancies |  `["0.pool.ntp.org:123","pool.ntp.org:123"]` |
| allowedDifference | A Golang time.Duration representation of the allowable time discrepancy between NTP server and your system time. Any discrepancy greater than this allowance will display an alert to your logging output |  `50000000` |
| allowedNegativeDifference | A Golang time.Duration representation of the allowable negative time discrepancy between NTP server and your system time. Any discrepancy greater than this allowance will display an alert to your logging output |  `50000000` |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
