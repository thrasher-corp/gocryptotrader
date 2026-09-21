package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestValidateCommandArguments(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		args     string
		mixed    bool
		wantArgs []string
	}{
		{name: "empty", args: "getticker"},
		{name: "positional", args: "getticker Bitmex ETH-USD perpetualcontract"},
		{name: "flags", args: "getticker --exchange Bitmex --asset perpetualcontract --pair ETH-USD"},
		{name: "equals flags", args: "getticker --exchange=Bitmex --asset=perpetualcontract --pair=ETH-USD"},
		{name: "reported missing pair", args: "getticker --asset perpetualcontract Bitmex ETH-USD", mixed: true},
		{name: "flagged exchange", args: "getticker --exchange Bitmex ETH-USD perpetualcontract", mixed: true},
		{name: "flagged pair", args: "getticker --pair ETH-USD Bitmex perpetualcontract", mixed: true},
		{name: "two flags", args: "getticker --asset perpetualcontract --pair ETH-USD Bitmex", mixed: true},
		{name: "trailing flag", args: "getticker Bitmex ETH-USD --asset perpetualcontract", mixed: true},
		{name: "trailing equals flag", args: "getticker Bitmex ETH-USD --asset=perpetualcontract", mixed: true},
		{name: "trailing ignored flag", args: "getticker Bitmex ETH-USD perpetualcontract --asset spot", mixed: true},
		{name: "unknown trailing flag", args: "getticker Bitmex --typo spot", mixed: true},
		{name: "single dash flag", args: "getticker Bitmex -asset spot", mixed: true},
		{name: "global and positional", args: "--rpchost localhost:9052 getticker Bitmex ETH-USD perpetualcontract"},
		{name: "global and flags", args: "--rpchost localhost:9052 getticker --exchange Bitmex"},
		{name: "global cannot hide mixed", args: "--rpchost localhost:9052 getticker --asset spot Bitmex", mixed: true},
		{name: "negative number", args: "getticker Bitmex -1 -0.25 -1e-3"},
		{name: "single dash", args: "getticker Bitmex -"},
		{name: "literal leading flag", args: "getticker -- --literal", wantArgs: []string{"--literal"}},
		{name: "literal trailing flag", args: "getticker Bitmex -- --literal", mixed: true},
		{name: "terminator cannot hide mixed", args: "getticker --asset spot -- Bitmex", mixed: true},
		{name: "nested positional", args: "group nested getticker Bitmex ETH-USD perpetualcontract"},
		{name: "nested flags", args: "group nested getticker --exchange Bitmex"},
		{name: "nested mixed", args: "group nested getticker --asset spot Bitmex", mixed: true},
		{name: "nested trailing", args: "group nested getticker Bitmex --asset spot", mixed: true},
		{name: "parent flag mixed", args: "group --mode value nested getticker Bitmex", mixed: true},
		{name: "parent flag and flags", args: "group --mode value nested getticker --exchange Bitmex"},
		{name: "nested global positional", args: "--rpchost localhost:9052 group nested getticker Bitmex"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var actionCalled bool
			var actionArgs []string
			command := &cli.Command{
				Name:     "getticker",
				HideHelp: true,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: exchangeFlag},
					&cli.StringFlag{Name: pairFlag},
					&cli.StringFlag{Name: assetFlag},
				},
				Before: validateCommandArguments,
				Action: func(c *cli.Context) error {
					actionCalled = true
					actionArgs = c.Args().Slice()
					return nil
				},
			}
			// Disable the library's shared help/version flags for parallel parser tests.
			app := &cli.App{
				HideHelp:    true,
				HideVersion: true,
				Writer:      io.Discard,
				ErrWriter:   io.Discard,
				Flags:       []cli.Flag{&cli.StringFlag{Name: "rpchost"}},
				Commands: []*cli.Command{
					command,
					{
						Name: "group", HideHelp: true, Before: validateCommandArguments,
						Flags:       []cli.Flag{&cli.StringFlag{Name: "mode"}},
						Subcommands: []*cli.Command{{Name: "nested", HideHelp: true, Before: validateCommandArguments, Subcommands: []*cli.Command{command}}},
					},
				},
			}
			err := app.RunContext(t.Context(), append([]string{"gctcli"}, strings.Fields(tc.args)...))
			if tc.mixed {
				require.ErrorIs(t, err, errMixedArguments, "mixed arguments must be rejected")
			} else {
				require.NoError(t, err, "a single argument style must be accepted")
			}
			assert.Equal(t, !tc.mixed, actionCalled, "only valid input should reach the command action")
			if tc.wantArgs != nil {
				assert.Equal(t, tc.wantArgs, actionArgs, "command action should receive normalised positional arguments")
			}
		})
	}
}

func TestRegisterArgumentValidation(t *testing.T) {
	t.Parallel()
	errBefore := errors.New("existing before hook failed")
	for _, tc := range []struct {
		name        string
		args        []string
		beforeError error
		wantError   error
		wantBefore  bool
	}{
		{name: "preserves hook", args: []string{"value"}, wantBefore: true},
		{name: "preserves hook error", args: []string{"value"}, beforeError: errBefore, wantError: errBefore, wantBefore: true},
		{name: "rejects before hook", args: []string{"--arg", "value", "positional"}, wantError: errMixedArguments},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var beforeCalled, actionCalled bool
			commands := []*cli.Command{{Name: "parent", HideHelp: true, Subcommands: []*cli.Command{{
				Name: "child", HideHelp: true,
				Flags: []cli.Flag{&cli.StringFlag{Name: "arg"}},
				Before: func(*cli.Context) error {
					beforeCalled = true
					return tc.beforeError
				},
				Action: func(*cli.Context) error {
					actionCalled = true
					return nil
				},
			}}}}
			registerArgumentValidation(commands)
			require.NotNil(t, commands[0].Before, "parent must receive validation")
			app := &cli.App{HideHelp: true, HideVersion: true, Commands: commands, Writer: io.Discard, ErrWriter: io.Discard}
			err := app.RunContext(t.Context(), append([]string{"gctcli", "parent", "child"}, tc.args...))
			require.ErrorIs(t, err, tc.wantError, "validation must preserve the expected error")
			assert.Equal(t, tc.wantBefore, beforeCalled, "existing hook should run only after validation")
			assert.Equal(t, tc.wantError == nil, actionCalled, "action should run only after successful hooks")
		})
	}
}
