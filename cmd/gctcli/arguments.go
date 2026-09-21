package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

var errMixedArguments = errors.New("cannot mix positional arguments and command flags; use either form exclusively")

func registerArgumentValidation(commands []*cli.Command) {
	for _, command := range commands {
		before := command.Before
		command.Before = func(c *cli.Context) error {
			if err := validateCommandArguments(c); err != nil {
				return err
			}
			if before != nil {
				return before(c)
			}
			return nil
		}
		registerArgumentValidation(command.Subcommands)
	}
}

func validateCommandArguments(c *cli.Context) error {
	args := c.Args().Slice()
	if len(args) == 0 || c.Command.Command(args[0]) != nil {
		return nil
	}

	lineage := c.Lineage()
	// Parent command flags also supply command parameters, but app flags only
	// configure the connection and may accompany either argument style.
	for i, context := range lineage {
		if i+1 == len(lineage) || lineage[i+1].Command == nil {
			break
		}
		if context.NumFlags() != 0 {
			return errMixedArguments
		}
	}

	// The parser stops at the first positional argument. Inspect the remainder
	// so that trailing flags cannot silently become ignored positional values.
	// An explicit -- permits literal values which look like flags.
	if len(lineage) > 1 {
		raw := lineage[1].Args().Tail()
		if offset := len(raw) - len(args); offset > 0 && raw[offset-1] == "--" {
			return nil
		}
	}
	for _, arg := range args {
		if arg == "--" {
			return errMixedArguments
		}
		if len(arg) > 1 && strings.HasPrefix(arg, "-") {
			// Negative numeric parameters are positional values, not flags.
			if _, err := strconv.ParseFloat(arg, 64); err != nil {
				return errMixedArguments
			}
		}
	}
	return nil
}
