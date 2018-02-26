package main

import "fmt"

// const vars related to the app version
const (
	MajorVersion = "0"
	MinorVersion = "1"

	PrereleaseBlurb = "This version is pre-release and is not inteded to be used as a production ready trading framework or bot - use at your own risk."
	IsRelease       = false
	Copyright       = "Copyright (c) 2018 The GoCryptoTrader Developers."
	GitHub          = "GitHub: https://github.com/thrasher-/gocryptotrader"
	Trello          = "Trello: https://trello.com/b/ZAhMhpOy/gocryptotrader"
	Slack           = "Slack:  https://gocryptotrader.herokuapp.com"
	Issues          = "Issues: https://github.com/thrasher-/gocryptotrader/issues"
)

// BuildVersion returns the version string
func BuildVersion(short bool) string {
	versionStr := fmt.Sprintf("GoCryptoTrader v%s.%s",
		MajorVersion, MinorVersion)
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
