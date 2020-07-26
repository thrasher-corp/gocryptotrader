package main

import (
	"os"
	"os/exec"
	"runtime"
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
