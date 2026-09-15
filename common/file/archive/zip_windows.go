package archive

import (
	"os"
	"syscall"
)

// lstatSource captures identity from a handle: os.Lstat on Windows loads file IDs lazily,
// so os.SameFile could otherwise reopen a replacement rather than identify the original
func lstatSource(src string) (os.FileInfo, error) {
	flags := os.O_RDONLY | syscall.FILE_FLAG_OPEN_REPARSE_POINT
	if src != "" && os.IsPathSeparator(src[len(src)-1]) {
		// Like os.Lstat, follow the last link when a trailing separator names its directory
		flags = os.O_RDONLY
	}
	f, err := os.OpenFile(src, flags, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}
