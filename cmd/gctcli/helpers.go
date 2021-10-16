package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"google.golang.org/grpc"
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
