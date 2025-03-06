// Package versions handles config upgrades and downgrades
/*
  - Versions must be stateful, and not rely upon type definitions in the config pkg

  - Instead versions should localise types into vN/types.go to avoid issues with subsequent changes

  - Versions must upgrade to the next version. Do not retrospectively change versions to match new type changes. Create a new version

  - Versions must implement ExchangeVersion or ConfigVersion, and may implement both
*/
package versions

import (
	"bytes"
	"context"
	"encoding/json" //nolint:depguard // Used instead of gct encoding/json so that we can ensure consistent library functionality between versions
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
)

// LatestVersion used as version param to Deploy to automatically use the latest version
const LatestVersion = -1

var (
	errMissingVersion      = errors.New("missing version")
	errVersionIncompatible = errors.New("version does not implement ConfigVersion or ExchangeVersion")
	errModifyingExchange   = errors.New("error modifying exchange config")
	errNoVersions          = errors.New("error retrieving latest config version: No config versions are registered")
	errApplyingVersion     = errors.New("error applying version")
	errConfigVersion       = errors.New("version in config file is higher than the latest available version")
	errTargetVersion       = errors.New("target downgrade version is higher than the latest available version")
)

// ConfigVersion is a version that affects the general configuration
type ConfigVersion interface {
	UpgradeConfig(context.Context, []byte) ([]byte, error)
	DowngradeConfig(context.Context, []byte) ([]byte, error)
}

// ExchangeVersion is a version that affects specific exchange configurations
type ExchangeVersion interface {
	Exchanges() []string // Use `*` for all exchanges
	UpgradeExchange(context.Context, []byte) ([]byte, error)
	DowngradeExchange(context.Context, []byte) ([]byte, error)
}

// manager contains versions registerVersioned during import init
type manager struct {
	m        sync.RWMutex
	versions []any
}

// Manager is a public instance of the config version manager
var Manager = &manager{}

// Deploy upgrades or downgrades the config between versions
// Pass LatestVersion for version to use the latest version automatically
// Prints an error an exits if the config file version or version param is not registered
func (m *manager) Deploy(ctx context.Context, j []byte, version int) ([]byte, error) {
	if err := m.checkVersions(); err != nil {
		return j, err
	}

	latest, err := m.latest()
	if err != nil {
		return j, err
	}

	target := latest
	if version != LatestVersion {
		target = version
	}

	m.m.RLock()
	defer m.m.RUnlock()

	current64, err := jsonparser.GetInt(j, "version")
	current := int(current64)
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		current = -1
	case err != nil:
		return j, fmt.Errorf("%w `version`: %w", common.ErrGettingField, err)
	}

	switch {
	case target == current:
		return j, nil
	case latest < current:
		warnVersionNotRegistered(current, latest, errConfigVersion)
		return j, errConfigVersion
	case target > latest:
		warnVersionNotRegistered(target, latest, errTargetVersion)
		return j, errTargetVersion
	}

	for current != target {
		patchVersion := current + 1
		action := "upgrade to"
		configMethod := ConfigVersion.UpgradeConfig
		exchMethod := ExchangeVersion.UpgradeExchange

		if target < current {
			patchVersion = current
			action = "downgrade from"
			configMethod = ConfigVersion.DowngradeConfig
			exchMethod = ExchangeVersion.DowngradeExchange
		}

		log.Printf("Running %s config version %v\n", action, patchVersion)

		patch := m.versions[patchVersion]

		if cPatch, ok := patch.(ConfigVersion); ok {
			if j, err = configMethod(cPatch, ctx, j); err != nil {
				return j, fmt.Errorf("%w %s %v: %w", errApplyingVersion, action, patchVersion, err)
			}
		}

		if ePatch, ok := patch.(ExchangeVersion); ok {
			if j, err = exchangeDeploy(ctx, ePatch, exchMethod, j); err != nil {
				return j, fmt.Errorf("%w %s %v: %w", errApplyingVersion, action, patchVersion, err)
			}
		}

		current = patchVersion
		if target < current {
			current = patchVersion - 1
		}

		if j, err = jsonparser.Set(j, []byte(strconv.Itoa(current)), "version"); err != nil {
			return j, fmt.Errorf("%w `version` during %s %v: %w", common.ErrSettingField, action, patchVersion, err)
		}
	}

	var out bytes.Buffer
	if err = json.Indent(&out, j, "", " "); err != nil {
		return j, fmt.Errorf("error formatting json: %w", err)
	}

	log.Println("Version management finished")

	return out.Bytes(), nil
}

func exchangeDeploy(ctx context.Context, patch ExchangeVersion, method func(ExchangeVersion, context.Context, []byte) ([]byte, error), j []byte) ([]byte, error) {
	var errs error
	wanted := patch.Exchanges()
	var i int
	eFunc := func(exchOrig []byte, _ jsonparser.ValueType, _ int, _ error) {
		defer func() { i++ }()
		name, err := jsonparser.GetString(exchOrig, "name")
		if err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %w `name`: %w", errModifyingExchange, common.ErrGettingField, err))
			return
		}
		for _, want := range wanted {
			if want != "*" && want != name {
				continue
			}
			exchNew, err := method(patch, ctx, exchOrig)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w: %w", errModifyingExchange, err))
				continue
			}
			if !bytes.Equal(exchNew, exchOrig) {
				if j, err = jsonparser.Set(j, exchNew, "exchanges", "["+strconv.Itoa(i)+"]"); err != nil {
					errs = common.AppendError(errs, fmt.Errorf("%w: %w `exchanges.[%d]`: %w", errModifyingExchange, common.ErrSettingField, i, err))
				}
			}
		}
	}
	v, dataType, _, err := jsonparser.Get(j, "exchanges")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError), dataType != jsonparser.Array:
		return j, nil
	case err != nil:
		return j, fmt.Errorf("%w: %w `exchanges`: %w", errModifyingExchange, common.ErrGettingField, err)
	}
	if _, err := jsonparser.ArrayEach(bytes.Clone(v), eFunc); err != nil {
		return j, err
	}
	return j, errs
}

// registerVersion takes instances of config versions and adds them to the registry
func (m *manager) registerVersion(ver int, v any) {
	m.m.Lock()
	defer m.m.Unlock()
	if ver >= len(m.versions) {
		m.versions = slices.Grow(m.versions, ver+1)[:ver+1]
	}
	m.versions[ver] = v
}

// latest returns the highest version number
func (m *manager) latest() (int, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	if len(m.versions) == 0 {
		return 0, errNoVersions
	}
	return len(m.versions) - 1, nil
}

// checkVersions ensures that registered versions are consistent
func (m *manager) checkVersions() error {
	m.m.RLock()
	defer m.m.RUnlock()
	for ver, v := range m.versions {
		switch v.(type) {
		case ExchangeVersion, ConfigVersion:
		default:
			return fmt.Errorf("%w: %v", errVersionIncompatible, ver)
		}
		if v == nil {
			return fmt.Errorf("%w: v%v", errMissingVersion, ver)
		}
	}
	return nil
}

func warnVersionNotRegistered(current, latest int, msg error) {
	fmt.Fprintf(os.Stderr, `
%s ('%d' > '%d')
Switch back to the version of GoCryptoTrader containing config version '%d' and run:
$ ./cmd/config downgrade %d 
`, msg, current, latest, current, latest)
}
