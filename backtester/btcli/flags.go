package main

import "github.com/urfave/cli/v2"

var taskIDFlag = &cli.StringFlag{
	Name:     "id",
	Usage:    "the id of the strategy task",
	Required: true,
}
