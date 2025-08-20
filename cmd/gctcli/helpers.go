package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"google.golang.org/grpc"
)

var (
	// use these to change text colours in CMD output
	redText     = "\033[38;5;203m"
	greenText   = "\033[38;5;157m"
	whiteText   = "\033[38;5;255m"
	grayText    = "\033[38;5;243m"
	defaultText = "\u001b[0m"
)

func clearScreen() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.CommandContext(context.TODO(), "cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		return cmd.Run()
	default:
		cmd := exec.CommandContext(context.TODO(), "clear")
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
