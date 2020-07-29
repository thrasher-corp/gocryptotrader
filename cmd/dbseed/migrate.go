package main

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	"github.com/urfave/cli/v2"
)

var migrateCommands = &cli.Command{
	Name:  "migrate",
	Usage: "migrate existing data",
	Subcommands: []*cli.Command{
		{
			Name:  "withdrawal",
			Usage: "migrate withdrawal data",
			Subcommands: []*cli.Command{
				{
					Name:      "history",
					Usage:     "migrate exchange to exchange_name_id history data",
					ArgsUsage: "update me",
					Action:    migrateWithdrawalHistoryExchangeName,
				},
			},
		},
	},
}

func migrateWithdrawalHistoryExchangeName(c *cli.Context) error {
	err := Load(c)
	if err != nil {
		return err
	}

	_, f, err := withdraw.MigrateData()
	if err != nil {
		return err
	}
	fmt.Printf("\nSuccessfully migrated data the following records failed to update and may require manual updating: \n\n")
	for x := range f {
		fmt.Println(f[x])
	}
	return nil
}