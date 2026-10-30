package main

import (
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var websocketManagerCommand = &cli.Command{
	Name:      "websocket",
	Usage:     "execute websocket management command",
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:  "getinfo",
			Usage: "returns all exchange websocket information",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
			},
			Action: getwebsocketInfo,
		},
		{
			Name:  "disable",
			Usage: "disables websocket connection for an exchange",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
			},
			Action: enableDisableWebsocket,
		},
		{
			Name:  enableFlag,
			Usage: "enables websocket connection for an exchange",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.BoolFlag{
					Name:   enableFlag,
					Hidden: true,
					Value:  true,
				},
			},
			Action: enableDisableWebsocket,
		},
		{
			Name:  "getsubs",
			Usage: "returns current subscriptions for an exchange",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
			},
			Action: getSubscriptions,
		},
		{
			Name:  "setproxy",
			Usage: "sets exchange websocket proxy, flushes and reroutes connection",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.StringFlag{
					Name:  "proxy",
					Usage: "proxy address to change to, if proxy string is not set, this will stop the utilisation of the prior set proxy.",
				},
			},
			Action: setProxy,
		},
		{
			Name:  "seturl",
			Usage: "sets exchange websocket endpoint URL and resets the websocket connection",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     exchangeFlag,
					Required: true,
					Usage:    exchangeUsage,
				},
				&cli.StringFlag{
					Name:  "url",
					Usage: "url string to change to, an empty string will set it back to the packaged defined default",
				},
			},
			Action: setURL,
		},
	},
}

func getwebsocketInfo(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WebsocketGetInfo(c.Context,
		&gctrpc.WebsocketGetInfoRequest{Exchange: c.String(exchangeFlag)})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func enableDisableWebsocket(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WebsocketSetEnabled(c.Context,
		&gctrpc.WebsocketSetEnabledRequest{Exchange: c.String(exchangeFlag), Enable: c.Bool(enableFlag)})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func getSubscriptions(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WebsocketGetSubscriptions(c.Context,
		&gctrpc.WebsocketGetSubscriptionsRequest{Exchange: c.String(exchangeFlag)})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func setProxy(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WebsocketSetProxy(c.Context,
		&gctrpc.WebsocketSetProxyRequest{Exchange: c.String(exchangeFlag), Proxy: c.String("proxy")})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func setURL(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WebsocketSetURL(c.Context,
		&gctrpc.WebsocketSetURLRequest{Exchange: c.String(exchangeFlag), Url: c.String("url")})
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}
