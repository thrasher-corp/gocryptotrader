package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestExecuteStrategyFromFile(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{executeStrategyFromFileCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "executestrategyfromfile", "--path", ""}), errStrategyPathRequired)
}

func TestStartTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{startTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "starttask", "--id", ""}), errTaskIDRequired)
}

func TestStopTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{stopTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "stoptask", "--id", ""}), errTaskIDRequired)
}

func TestClearTask(t *testing.T) {
	app := &cli.App{Commands: []*cli.Command{clearTaskCommand}}
	require.ErrorIs(t, app.Run([]string{"btcli", "cleartask", "--id", ""}), errTaskIDRequired)
}
