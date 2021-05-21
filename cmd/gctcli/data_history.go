package main

import (
	"context"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli"
)

var dataHistoryCommands = cli.Command{
	Name:      "datahistory",
	Usage:     "manage data history jobs to retrieve historic trade or candle data over time",
	ArgsUsage: "<command> <args>",
	Subcommands: []cli.Command{
		{
			Name:   "getactivejobs",
			Usage:  "returns all jobs that are currently active",
			Flags:  []cli.Flag{},
			Action: getActiveDataHistoryJobs,
		},
		{
			Name:        "getajob",
			Usage:       "returns a job by either its id or nickname",
			Description: "na-na, why don't you get a job?",
			ArgsUsage:   "<id> or <nickname>",
			Action:      getActiveDataHistoryJobs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "id",
					Usage: "deadbeef-dead-beef-dead-beef13371337",
				},
				cli.StringFlag{
					Name:  "nickname",
					Usage: "binance-spot-btc-usdt-2019-trades",
				},
			},
		},
		{
			Name:        "getjobwithdetailedresults",
			Usage:       "returns a job by either its id or nickname along with all its data retrieval results",
			Description: "results may be large",
			ArgsUsage:   "<id> or <nickname>",
			Action:      getActiveDataHistoryJobs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "id",
					Usage: "deadbeef-dead-beef-dead-beef13371337",
				},
				cli.StringFlag{
					Name:  "nickname",
					Usage: "binance-spot-btc-usdt-2019-trades",
				},
			},
		},
		{
			Name:      "addnewjob",
			Usage:     "creates a new data history job",
			ArgsUsage: "<asset>",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "nickname",
					Usage:    "binance-spot-btc-usdt-2019-trades",
					Required: true,
				},
				cli.StringFlag{
					Name:  "exchange",
					Usage: "binance",
				},
				cli.StringFlag{
					Name:  "asset",
					Usage: "spot",
				},
				cli.StringFlag{
					Name:  "pair",
					Usage: "btc-usdt",
				},
				cli.StringFlag{
					Name:  "start_date",
					Usage: "2006-01-02 15:04:05",
				},
				cli.StringFlag{
					Name:  "end_date",
					Usage: "2006-01-02 15:04:05",
				},
				cli.StringFlag{
					Name:  "interval",
					Usage: klineMessage,
				},
				cli.StringFlag{
					Name:  "request_size_limit",
					Usage: "500 - will only retreive 500 candles per request",
				},
				cli.StringFlag{
					Name:  "data_type",
					Usage: "candles or trades",
				},
				cli.StringFlag{
					Name:  "max_retry_attempts",
					Usage: "3 - the maximum retry attempts for an interval period before giving up for a given interval",
				},
				cli.StringFlag{
					Name:  "batch_size",
					Usage: "500 - will only retreive 500 candles in a run",
				},
			},
			Action: getActiveDataHistoryJobs,
		},
		{
			Name:      "upsertjob",
			Usage:     "adds a new job, or updates an existing one if it matches jobid OR nickname",
			ArgsUsage: "<asset>",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "nickname",
					Usage:    "binance-spot-btc-usdt-2019-trades",
					Required: true,
				},
				cli.StringFlag{
					Name:  "exchange",
					Usage: "binance",
				},
				cli.StringFlag{
					Name:  "asset",
					Usage: "spot",
				},
				cli.StringFlag{
					Name:  "pair",
					Usage: "btc-usdt",
				},
				cli.StringFlag{
					Name:  "start_date",
					Usage: "2006-01-02 15:04:05",
				},
				cli.StringFlag{
					Name:  "end_date",
					Usage: "2006-01-02 15:04:05",
				},
				cli.StringFlag{
					Name:  "interval",
					Usage: klineMessage,
				},
				cli.StringFlag{
					Name:  "request_size_limit",
					Usage: "500 - will only retreive 500 candles per request",
				},
				cli.StringFlag{
					Name:  "data_type",
					Usage: "candles or trades",
				},
				cli.StringFlag{
					Name:  "max_retry_attempts",
					Usage: "3 - the maximum retry attempts for an interval period before giving up for a given interval",
				},
				cli.StringFlag{
					Name:  "batch_size",
					Usage: "500 - will only retreive 500 candles in a run",
				},
			},
			Action: getActiveDataHistoryJobs,
		},
		{
			Name:      "deletejob",
			Usage:     "sets a jobs status to deleted so it no longer is processed",
			ArgsUsage: "<id> or <nickname>",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "id",
					Usage: "deadbeef-dead-beef-dead-beef13371337",
				},
				cli.StringFlag{
					Name:  "nickname",
					Usage: "binance-spot-btc-usdt-2019-trades",
				},
			},
			Action: getActiveDataHistoryJobs,
		},
	},
}

func getActiveDataHistoryJobs(c *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Print(err)
		}
	}()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetDataHistoryJobDetails(context.Background(),
		&gctrpc.SetExchangeTradeProcessingRequest{
			Exchange: exchangeName,
			Status:   status,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
