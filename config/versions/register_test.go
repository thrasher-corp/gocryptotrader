package versions_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config/versions"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

func TestVersionsRegistered(t *testing.T) {
	t.Parallel()

	r, err := testutils.RootPathFromCWD()
	require.NoError(t, err)

	versionsDir := filepath.Join(r, "config", "versions")
	_, err = os.Stat(versionsDir)
	require.NoError(t, err, "config/versions must exist")

	err = filepath.WalkDir(versionsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() || p == versionsDir {
			return nil
		}
		verStr := filepath.Base(p)
		verMatch := regexp.MustCompile(`v(\d+)`).FindStringSubmatch(verStr)
		if len(verMatch) != 2 {
			return filepath.SkipDir
		}
		t.Run(verStr, func(t *testing.T) {
			version, err := strconv.ParseUint(verMatch[1], 10, 16)
			require.NoError(t, err, "verMatch must ParseUint without error")
			v := versions.Manager.Version(uint16(version))
			require.NotNil(t, v, "version.Manager init must register this version")
			require.Contains(t, reflect.TypeOf(v).String(), "*"+verStr+".Version", "version registered must be the correct type")
		})
		return filepath.SkipDir
	})
	require.NoError(t, err, "WalkDir must not error")
}
