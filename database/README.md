# GoCryptoTrader package Database

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/page-logo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://travis-ci.org/thrasher-corp/gocryptotrader.svg?branch=master)](https://travis-ci.org/thrasher-corp/gocryptotrader)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/portfolio)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This database package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progresss on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTQyYjIxNGVhMWU5MDZlOGYzMmE0NTJmM2MzYWY5NGMzMmM4MzUwNTBjZTEzNjIwODM5NDcxODQwZDljMGQyNGY)

## Current Features for database package

+ Establishes & Maintains database connection across program life cycle
+ Multiple database support via simple repository model
+ Run migration on connection to assure database is at correct version

## How to use

##### Create and Run migrations
 Migrations are created using a modified version of [Goose](https://github.com/thrasher-corp/goose) 
 
 A helper tool sits in the ./cmd/dbmigrate folder that includes the following features:
 
+ Check current database version with the "status" command
```shell script
go run ./cmd/dbmigrate -command status
```
+ Create a new migration
```sh
go run ./cmd/dbmigrate -command "create" -args "somemodel"
```
_This will create a folder in the ./database/migration folder that contains postgres.sql and sqlite.sql files_
 + Run dbmigrate command with -command up 
```shell script
go run ./cmd/dbmigrate -command "up"
```
##### Adding a new model
Model's are generated using [SQLBoilers](https://github.com/volatiletech/sqlboiler) 
A helper tool has been made located in ./cmd/gen_sqlboiler_config that will parse your GoCryptoTrader config and output a SQLBoiler config

```sh
go run ./cmd/gen_sqlboiler_config
```

Models generated and sit in the ./database/models/<databasetype> folder using sqlboiler 

```shell script
sqlboiler -o database/models/postgres -p "postgres" psql
```

##### Adding a Repository
+ Create Repository directory in github.com/thrasher-corp/gocryptotrader/database/repository/

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

***1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB***

