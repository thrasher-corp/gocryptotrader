# GoCryptoTrader package Validate

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/validate)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This validate package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for validate

+ This package allows for validation options to occur exchange side e.g.
	- Checking for ID in an order cancellation struct.
	- Determining the correct withdrawal bank details for a specific exchange.

+ Example Usage below:

```go 
// import package
"github.com/thrasher-corp/exchanges/validate"

// define your data structure across potential exchanges
type Critical struct {
	ID string
	Person string
	Banks string
	MoneysUSD float64
}

// define validation and add a variadic param
func (supercritcalinfo *Critical) Validate(opt ...validate.Checker) error {
	// define base level validation
	if supercritcalinfo != nil {
			// oh no this is nil, could panic program!
	}

	// range over potential checks coming from individual packages
	var errs common.Errors
	for _, o := range opt {
		err := o.Check()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return errs
	}
	return nil
}

// define an exchange or package level check that returns a validate.Checker 
// interface
func (supercritcalinfo *Critical) PleaseDontSendMoneyToParents() validate.Checker {
	return validate.Check(func() error {
		if supercritcalinfo.Person == "Mother Dearest" ||
			supercritcalinfo.Person == "Father Dearest" {
			return errors.New("nope")
		}
	return nil
	})
}


// Now in the package all you have to do is add in your options or not...
d := Critical{Person: "Mother Dearest", MoneysUSD: 1337.30}

// This should not error 
err := d.Validate()
if err != nil {
	return err
}

// This should error 
err := d.Validate(d.PleaseDontSendMoneyToParents())
if err != nil {
	return err
}

```

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
