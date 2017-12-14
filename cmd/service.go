package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	config    bool
	portfolio bool
	websocket bool
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Service allows access to different capabilties to the bot",
	Long: `Service currently allows three different tools to be initiated:

		Configuration utility
                Portfolio viewer
                Websocket utility

                Complete documentation is not available yet, please refer to the trello board @ https://trello.com/b/ZAhMhpOy/gocryptotrader
		Or join us on slack @ https://gocryptotrader.herokuapp.com/`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if config == false && portfolio == false && websocket == false {
			fmt.Println("Please set a flag or append --help to service for help")
			os.Exit(0)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if config {
			bot.Configuration.Run()
		}
		if portfolio {
			bot.Portfolio.Run()
		}
		if websocket {
			bot.Websocket.Run()
		}
	},
}

func init() {
	//local set flags
	serviceCmd.Flags().BoolVarP(&config, "config", "c", false, "Starts the configuration utility")
	serviceCmd.Flags().BoolVarP(&portfolio, "portfolio", "p", false, "Starts the portfolio utility")
	serviceCmd.Flags().BoolVarP(&websocket, "websocket", "w", false, "Starts the websocket utility")
}
