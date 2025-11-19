# GoCryptoTrader package Connection Manager

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/connection_manager)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This connection_manager package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for Connection Manager
+ The connection manager subsystem is used to periodically check whether the application is connected to the internet and will provide alerts of any changes
+ In order to modify the behaviour of the connection manager subsystem, you can edit the following inside your config file under `connectionMonitor`:

### connectionMonitor

| Config | Description | Example |
| ------ | ----------- | ------- |
| perferredDNSList | Is a string array of DNS servers to periodically verify whether GoCryptoTrader is connected to the internet |  `["8.8.8.8","8.8.4.4","1.1.1.1","1.0.0.1"]` |
| preferredDomainList |  Is a string array of domains to periodically verify whether GoCryptoTrader is connected to the internet |  `["www.google.com","www.cloudflare.com","www.facebook.com"]` |
| checkInterval | A time period in golang `time.Duration` format to check whether GoCryptoTrader is connected to the internet | `1000000000` |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
