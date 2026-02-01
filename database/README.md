# GoCryptoTrader package Database

<img src="/docs/assets/page-logo.png" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/portfolio)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This database package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for database package

+ Establishes & Maintains database connection across program life cycle
+ Migration handed by [Goose](https://github.com/thrasher-corp/goose) 
+ Model generation handled by [SQLBoiler](https://github.com/thrasher-corp/sqlboiler) 

## How to use

##### Prerequisites

[SQLBoiler](https://github.com/thrasher-corp/sqlboiler)
```shell script
go install github.com/thrasher-corp/sqlboiler
```

[Postgres Driver](https://github.com/thrasher-corp/sqlboiler/drivers/sqlboiler-psql)
```shell script
go install github.com/thrasher-corp/sqlboiler/drivers/sqlboiler-psql
```

[SQLite Driver](https://github.com/thrasher-corp/sqlboiler-sqlite3)
```shell script
go install github.com/thrasher-corp/sqlboiler-sqlite3
```

##### Configuration

The database configuration struct is currently: 
```shell script
type Config struct {
	Enabled                   bool   `json:"enabled"`
	Verbose                   bool   `json:"verbose"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}
```
And Connection Details:
```sh
type ConnectionDetails struct {
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode"`
}
```

With an example configuration being:

```sh
 "database": {
  "enabled": true,
  "verbose": true,
  "driver": "postgres",
  "connectionDetails": {
   "host": "localhost",
   "port": 5432,
   "username": "gct-dev",
   "password": "gct-dev",
   "database": "gct-dev",
   "sslmode": "disable"
  }
 },
```

##### Create and Run migrations
 Migrations are created using a modified version of [Goose](https://github.com/thrasher-corp/goose) 
 
 A helper tool sits in the ./cmd/dbmigrate folder that includes the following features:
 
+ Check current database version with the "status" command
```shell script
dbmigrate -command status
```

+ Create a new migration
```sh
dbmigrate -command "create" -args "model"
```
_This will create a folder in the ./database/migration folder that contains postgres.sql and sqlite.sql files_
 + Run dbmigrate command with -command up 
```shell script
dbmigrate -command "up"
```

dbmigrate provides a -migrationdir flag override to tell it what path to look in for migrations

###### Note: its highly recommended to backup any data before running migrations against a production database especially if you are running SQLite due to alter table limitations


##### Adding a new model
Model's are generated using [SQLBoiler](https://github.com/thrasher-corp/sqlboiler) 
A helper tool has been made located in gen_sqlboiler_config that will parse your GoCryptoTrader config and output a SQLBoiler config

```sh
gen_sqlboiler_config
```

By default this will look in your gocryptotrader data folder and default config, these can be overwritten 
along with the location of the sqlboiler generated config

```shell script
-config "configname.json"
-datadir "~/.gocryptotrader/"
-outdir "~/.gocryptotrader/"
```

Generate a new model that gets placed in ./database/models/<databasetype> folder

Linux:
```shell script
sqlboiler -o database/models/postgres -p postgres --no-auto-timestamps --wipe psql 
```
Windows: 
```sh
sqlboiler -o database\\models\\postgres -p postgres --no-auto-timestamps --wipe psql
```

Helpers have been provided in the Makefile for linux users 
```
make gen_db_models
```
And in the contrib/sqlboiler.cmd for windows users

##### Adding a Repository
+ Create Repository directory in github.com/thrasher-corp/gocryptotrader/database/repository/


##### DBSeed helper
A helper tool [cmd/dbseed](../cmd/dbseed/README.md) has been created for assisting with data migration 

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***

