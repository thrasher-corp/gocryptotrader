# GoCryptoTrader package Documentation

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/cmd/documentation)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This documentation package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for documentation

#### This tool allows for the generation of new documentation through templating

From the `gocryptotrader/cmd/documentation/` folder, using the go command: **go run documentation.go** this will auto-generate and regenerate documentation across the **GoCryptoTrader** code base.
>Using the -v command will, ie **go run documentation.go -v** put the tool into verbose mode allowing you to see what is happening with a little more depth.

Be aware, this tool will:
- Works off a configuration JSON file located at ``gocryptotrader/cmd/documentation/`` for future use with multiple repositories.
- Automatically find the directory and file tree for the GoCryptoTrader source code and alert you of undocumented file systems which **need** to be updated.
- Automatically find the template folder tree.
- Fetch an updated contributor list and rank on pull request commit amount. Set the `GITHUB_TOKEN` environment variable or use the `ghtoken` command-line flag for optional authentication.
- Sets up core folder docs for the root directory tree including **LICENSE** and **CONTRIBUTORS**

### config.json example

```json
{
 "githubRepo": "https://api.github.com/repos/thrasher-corp/gocryptotrader", This is your current repo
 "exclusionList": { This allows for excluded directories and files
  "Files": null,
  "Directories": [
   "_templates",
   ".git",
   "web"
  ]
 },
 "rootReadmeActive": true, allows a root directory README.md
 "licenseFileActive": true, allows for a license file to be generated
 "contributorFileActive": true, fetches a new contributor list
 "referencePathToRepo": "../../"
}
```
### Template example
>place a new template **example_file.tmpl** located in the current gocryptotrader/cmd/documentation/ folder; when the documentation tool finishes it will give you the define template associated name e.g. ``Template not found for path ../../cmd/documentation create new template with \{\{define "cmd documentation" -\}\} TEMPLATE HERE \{\{end}}`` so you can replace the below example with ``\{\{define "cmd documentation" -}}``

```
\{\{\define "example_definition_created_by_documentation_tool" -}}
\{\{\template "header" .}}
## Current Features for documentation

#### A concise blurb about the package or tool system

+ Coding examples
import "github.com/thrasher-corp/gocryptotrader/something"
testString := "aAaAa"
upper := strings.ToUpper(testString)
// upper == "AAAAA"

{\{\template "contributions"}}
{\{\template "donations"}}
{\{\end}}
```

### ALL NEW UPDATES AND FILE SYSTEM ADDITIONS NEED A DOCUMENTATION UPDATE USING THIS TOOL OR PR MERGE REQUEST MAY BE POSTPONED.

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
