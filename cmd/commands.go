package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thrasher-/gocryptotrader/bot/services"
)

var bot services.Services

// Flags to be added to commands.
var (
	verbose bool
	debug   bool
	version bool
)

// GocryptotraderCmd is the root command utility
var GocryptotraderCmd = &cobra.Command{
	Use:   "gocryptotrader",
	Short: "GoCryptoTrader trading platform",
	Long: `GoCryptoTrader is a cryptocurrency specialized trading platform written in Golang.

		Automatic and assistive trading algorithms,
                Portfolio monitoring tools,
                Automatic tax file creation for financial years,
                Websocket, REST and FIX API integrations with high volume exchanges,

		Complete documentation is not available yet, please refer to the trello board @ https://trello.com/b/ZAhMhpOy/gocryptotrader
		Or join us on slack @ https://gocryptotrader.herokuapp.com/`,
	Run: func(cmd *cobra.Command, args []string) {
		if version {
			fmt.Println("GoCryptoTader v0.0 -- WARNING UNSTABLE!!!")
			os.Exit(0)
		}
		bot.DefaultMain.Run()
	},
}

func init() {
	// Commands to add
	GocryptotraderCmd.AddCommand(serviceCmd)

	//flags
	GocryptotraderCmd.Flags().BoolVarP(&version, "version", "v", false, "Displays the current version of GoCryptoTrader")

	//Global flags set
	GocryptotraderCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "verbose output")
	GocryptotraderCmd.PersistentFlags().BoolVarP(&debug, "debug", "", false, "debug output")
	bot = services.Setup()
}
