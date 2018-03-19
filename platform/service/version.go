package service

import (
	"os"

	"github.com/thrasher-/gocryptotrader/platform"
)

// Version prints the curret build version for GoCryptoTader
func Version() {
	bot := platform.GetBot(false, false, "")
	_ = bot
	os.Exit(0)
}
