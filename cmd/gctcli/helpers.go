package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func clearScreen() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		return cmd.Run()
	default:
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}
}

func closeConn(conn *grpc.ClientConn, cancel context.CancelFunc) {
	if err := conn.Close(); err != nil {
		fmt.Println(err)
	}
	if cancel != nil {
		cancel()
	}
}

// negateLocalOffset helps negate the offset of time generation
// when the unix time gets to rpcserver, it no longer is the same time
// that was sent as it handles it as a UTC value, even though when
// using starttime it is generated as your local time
// eg 2020-01-01 12:00:00 +10 will convert into
// 2020-01-01 12:00:00 +00 when at RPCServer
// so this function will minus the offset from the local sent time
// to allow for proper use at RPCServer
func negateLocalOffset(t time.Time) string {
	_, offset := time.Now().Zone()
	loc := time.FixedZone("", -offset)

	return t.In(loc).Format(common.SimpleTimeFormat)
}

func negateLocalOffsetTS(t time.Time) *timestamppb.Timestamp {
	_, offset := time.Now().Zone()
	return timestamppb.New(t.Add(time.Duration(-offset) * time.Second))
}
