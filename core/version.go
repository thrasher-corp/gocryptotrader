package core

import (
	"fmt"
	"runtime"
	"time"
)

// const vars related to the app version
const (
	MajorVersion = "0"
	MinorVersion = "1"

	PrereleaseBlurb = "This version is pre-release and is not inteded to be used as a production ready trading framework or bot - use at your own risk."
	IsRelease       = false
	GitHub          = "GitHub: https://github.com/thrasher-corp/gocryptotrader"
	Trello          = "Trello: https://trello.com/b/ZAhMhpOy/gocryptotrader"
	Slack           = "Slack:  https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk"
	Issues          = "Issues: https://github.com/thrasher-corp/gocryptotrader/issues"
)

// vars related to the app version
var (
	Copyright = fmt.Sprintf("Copyright (c) 2014-%d The GoCryptoTrader Developers.",
		time.Now().Year())
)

// Version returns the version string
func Version(short bool) string {
	versionStr := fmt.Sprintf("GoCryptoTrader v%s.%s %s %s",
		MajorVersion, MinorVersion, runtime.GOARCH, runtime.Version())
	if !IsRelease {
		versionStr += " pre-release.\n"
		if !short {
			versionStr += PrereleaseBlurb + "\n"
		}
	} else {
		versionStr += " release.\n"
	}
	if short {
		return versionStr
	}
	versionStr += Copyright + "\n\n"
	versionStr += GitHub + "\n"
	versionStr += Trello + "\n"
	versionStr += Slack + "\n"
	versionStr += Issues + "\n"
	return versionStr
}
