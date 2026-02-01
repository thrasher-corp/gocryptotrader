# GoCryptoTrader package Database Connection

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/database_connection)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This database_connection package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for Database Connection
+ The database connection manager subsystem is used to periodically check whether the application is connected to the database and will provide alerts of any changes
+ In order to modify the behaviour of the database connection manager subsystem, you can edit the following inside your config file under `database`:

### database

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | Enabled or disables the database connection subsystem |  `true` |
| verbose | Displays more information to the logger which can be helpful for debugging | `false` |
| driver | The SQL driver to use. Can be `postgres` or `sqlite` | `sqlite` |
| connectionDetails | See below |  |

### connectionDetails

| Config | Description | Example |
| ------ | ----------- | ------- |
| host | The host address of the database |  `localhost` |
| port |  The port used to connect to the database |  `5432` |
| username | An optional username to connect to the database | `username` |
| password | An optional password to connect to the database | `password` |
| database | The name of the database | `database.db` |
| sslmode | The connection type of the database for Postgres databases only | `disable` |

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
