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

	PrereleaseBlurb = "This version is pre-release and is not intended to be used as a production ready trading framework or bot - use at your own risk."
	IsRelease       = false
	GitHub          = "GitHub: https://github.com/thrasher-corp/gocryptotrader"
	ProjectKanban   = "Kanban: https://github.com/orgs/thrasher-corp/projects/3"
	Slack           = "Slack:  https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g"
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
	versionStr += ProjectKanban + "\n"
	versionStr += Slack + "\n"
	versionStr += Issues + "\n"
	return versionStr
}
