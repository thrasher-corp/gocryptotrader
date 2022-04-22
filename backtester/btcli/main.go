package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/signaler"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

var (
	host          string
	username      string
	password      string
	pairDelimiter string
	certPath      string
	timeout       time.Duration
	exchangeCreds exchange.Credentials
)

const defaultTimeout = time.Second * 30

func jsonOutput(in interface{}) {
	j, err := json.MarshalIndent(in, "", " ")
	if err != nil {
		return
	}
	fmt.Print(string(j))
}

func setupClient(c *cli.Context) (*grpc.ClientConn, context.CancelFunc, error) {
	creds, err := credentials.NewClientTLSFromFile(certPath, "")
	if err != nil {
		return nil, nil, err
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: username,
			Password: password,
		}),
	}

	var cancel context.CancelFunc
	c.Context, cancel = context.WithTimeout(c.Context, timeout)
	if !exchangeCreds.IsEmpty() {
		flag, values := exchangeCreds.GetMetaData()
		c.Context = metadata.AppendToOutgoingContext(c.Context, flag, values)
	}
	conn, err := grpc.DialContext(c.Context, host, opts...)
	return conn, cancel, err
}

func main() {
	app := cli.NewApp()
	app.Name = "gctcli"
	app.Version = core.Version(true)
	app.EnableBashCompletion = true
	app.Usage = "command line interface for managing the gocryptotrader backtester daemon"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "rpchost",
			Value:       "localhost:42069",
			Usage:       "the gRPC host to connect to",
			Destination: &host,
		},
		&cli.StringFlag{
			Name:        "rpcuser",
			Value:       "backtester",
			Usage:       "the gRPC username",
			Destination: &username,
		},
		&cli.StringFlag{
			Name:        "rpcpassword",
			Value:       "helloImTheDefaultPassword",
			Usage:       "the gRPC password",
			Destination: &password,
		},
		&cli.StringFlag{
			Name:        "delimiter",
			Value:       "-",
			Usage:       "the default currency pair delimiter used to standardise currency pair input",
			Destination: &pairDelimiter,
		},
		&cli.StringFlag{
			Name:        "cert",
			Value:       filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "backtester", "tls", "cert.pem"),
			Usage:       "the path to TLS cert of the gRPC server",
			Destination: &certPath,
		},
		&cli.DurationFlag{
			Name:        "timeout",
			Value:       defaultTimeout,
			Usage:       "the default context timeout value for requests",
			Destination: &timeout,
		},
	}
	app.Commands = []*cli.Command{
		executeStrategyFromFileCommand,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Capture cancel for interrupt
		signaler.WaitForInterrupt()
		cancel()
		fmt.Println("rpc process interrupted")
		os.Exit(1)
	}()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
