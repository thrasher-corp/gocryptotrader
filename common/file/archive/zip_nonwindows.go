//go:build !windows

package archive

import "os"

var lstatSource = os.Lstat
