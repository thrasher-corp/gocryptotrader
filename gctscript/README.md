# GoCryptoTrader package gctscript

<img src="/docs/assets/page-logo.png" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/portfolio)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This gctscript package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for gctscript package

+ Execute scripts
+ Terminate scripts
+ Autoload scripts on bot startup
+ Current Exchange features supported:
  + Enabled Exchanges
  + Enabled currency pairs
  + Account information
  + Query Order
  + Submit Order
  + Cancel Order
  + Ticker
  + Orderbook

## How to use

##### Prerequisites

To Enable database logging support you must have an active migrated database by following the [database setup guide](../database/README.md)

##### Syntax Highlighting

To enable syntax highlighting for vscode download extension [graphman65/vscode-tengo](https://github.com/graphman65/vscode-tengo/) then add `".gct"` to vscode-tengo package.json [settings](https://github.com/graphman65/vscode-tengo/blob/master/package.json#L27) to enable highlighting of our files. 

##### Configuration

The gctscript configuration struct is currently: 
```shell script
type Config struct {
	Enabled       bool          `json:"enabled"`
	ScriptTimeout time.Duration `json:"timeout"`
	AllowImports  bool          `json:"allow_imports"`
	AutoLoad      []string      `json:"auto_load"`
	Verbose       bool          `json:"Verbose"`
}
```

With an example configuration being:

```sh
 "gctscript": {
  "enabled": true,
  "timeout": 600000000,
  "allow_imports": true,
  "auto_load": [],
  "debug": false
 },
```
##### Script Control
+ You can autoload scripts on bot start up by placing their name in the "auto_load" config entry
  ```shell script
  "auto_load": ["one","two"]
  ```
  This will look in your GoCryptoTrader data directory in a folder called "scripts" for files one.gct and two.gct and autoload them
+ Manual control of scripts can be done via the gctcli command with support for the following:

  - Enable/Disable GCTScript:
   ```shell script
    gctcli enablesubsystem "gctscript"
    gctcli disablesubsystem "gctscript"
  ```
  - Start/Execute:
  ```shell script
    gctcli script execute <scriptname> <pathoverride>
    gctcli script execute "timer.gct" "~/gctscript"
  
    {
      "status": "ok",
      "data": "timer.gct executed"
    }
  ```
  - Stop:
  ```shell script
    gctcli script stop <uuid>
    gctcli script stop 821bd73e-02b1-4974-9463-874cb49f130d
  
    {
      "status": "ok",
      "data": "821bd73e-02b1-4974-9463-874cb49f130d terminated"
    }
  ```
  - Status:
  ```shell script
    gctcli script status 
  
    {
      "status": "ok",
      "scripts": [
        {
          "uuid": "821bd73e-02b1-4974-9463-874cb49f130d",
          "name": "timer.gct",
          "next_run": "2019-11-14 13:11:40.224919456 +1100 AEDT m=+91.062103259"
        }
      ]
    }
  ```
  - Read file:
  ```shell script
    gctcli script read <filename>
    gctcli script read "timer.gct"
  
    {
      "status": "ok",
      "script": {
      "name": "timer.gct",
      "path": "/home/x/.gocryptotrader/scripts"
      },
      "data": "fmt := import(\"fmt\")\nt := import(\"times\")\n\nname := \"run\"\ntimer := \"5s\"\n\nload := func() {\n\tfmt.printf(\"5s %s\\n\",t.now())\n}\n\nload()\n"
    }
   ```
    - Query running script:
    ```shell script
      gctcli script query <uuid>
      gctcli script query 821bd73e-02b1-4974-9463-874cb49f130d
      {
        "status": "ok",
        "script": {
        "UUID": "bf692e2d-fa1e-4d95-92fd-33d7634d3d77",
        "name": "timer.gct",
        "path": "/home/x/.gocryptotrader/scripts",
        "next_run": "2019-12-12 07:44:19.747572406 +1100 AEDT m=+16.782773385"
      },
      "data": "fmt := import(\"fmt\")\nt := import(\"times\")\n\nname := \"run\"\ntimer := \"5s\"\n\nload := func() {\n\tfmt.printf(\"5s %s\\n\",t.now())\n}\n\nload()\n"
      }
      load()  
     ```
     - Add script to autoload:
    ```shell script
    gctcli script autoload add timer
    {
      "status": "success",
      "data": "script timer added to autoload list"
    }
    ```
    - Remove script from autoload:
    ```shell script
      gctcli script autoload remove timer
      {
        "status": "success",
        "data": "script timer removed from autoload list"
      }
    ```
##### Scripting & Extending modules

The scripting engine utilises [tengo](https://github.com/d5/tengo) an intro tutorial for it can be found [here](https://github.com/d5/tengo/blob/master/docs/tutorial.md)

Modules have been written so far linking up common exchange features including 

- Orderbook
- Ticker
- Order Management
- Account information
- Withdraw funds 
- Get Deposit Addresses

Extending or creating new modules:

Extending an existing module the exchange module for example is simple
- Open required [module](modules/gct/exchange.go)
- Add to exchangeModule map
- Define function with signature ```(args ...objects.Object) (ret objects.Object, err error)```

Similar steps can be taken to add a new module with a few adjustments
- Open required [GCT](modules/gct/gct_types.go)
- Add module name to GCTModules map

##### GCT module methods

Current supported methods added and exposed to scripts are as follows:

```
accountinfo
-> exchange:string

depositaddress
-> exchange:string
-> currency:string

orderbook
-> exchange:string
-> currency pair:string
-> delimiter:string
-> asset:string

ticker
-> exchange:string
-> currency pair:string
-> delimiter:string
-> asset:string

pairs
-> exchange:string
-> enabled only:bool
-> asset:string

queryorder
-> exchange:string
-> order id:string

submitorder
-> exchange:string
-> currency pair:string
-> delimiter:string
-> order type:string
-> order side:string
-> price:float64
-> amount:float64
-> client_id:string

withdrawfiat
-> exchange:string
-> currency:string
-> description:string
-> amount:float64
-> bank id:string

withdrawcrypto
-> exchange:string
-> currency:string
-> address:string
-> address tag:string
-> amount:float64
-> fee:float64
-> description:string
```

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***

