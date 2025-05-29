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
	"math"
	"os"
	"slices"
	"strconv"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
)

// UseLatestVersion used as version param to Deploy to automatically use the latest version
const UseLatestVersion = math.MaxUint16

var (
	errVersionIncompatible   = errors.New("version does not implement ConfigVersion or ExchangeVersion")
	errAlreadyRegistered     = errors.New("version is already registered")
	errModifyingExchange     = errors.New("error modifying exchange config")
	errNoVersions            = errors.New("error retrieving latest config version: No config versions are registered")
	errApplyingVersion       = errors.New("error applying version")
	errTargetVersion         = errors.New("target downgrade version is higher than the latest available version")
	errConfigVersion         = errors.New("invalid version in config")
	errConfigVersionUnavail  = errors.New("version is higher than the latest available version")
	errConfigVersionNegative = errors.New("version is negative")
	errConfigVersionMax      = errors.New("version is above max versions")
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
// Pass UseLatestVersion for version to use the latest version automatically
// Prints an error an exits if the config file version or version param is not registered
func (m *manager) Deploy(ctx context.Context, j []byte, version uint16) ([]byte, error) {
	if err := m.checkVersions(); err != nil {
		return j, err
	}

	latest, err := m.latest()
	if err != nil {
		return j, err
	}

	target := latest
	if version != UseLatestVersion {
		target = version
	}

	m.m.RLock()
	defer m.m.RUnlock()

	current64, err := jsonparser.GetInt(j, "version")
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		// With no version first upgrade is to Version1; current64 is already 0
	case err != nil:
		return j, fmt.Errorf("%w: %w `version`: %w", errConfigVersion, common.ErrGettingField, err)
	case current64 < 0:
		return j, fmt.Errorf("%w: %w `version`: `%d`", errConfigVersion, errConfigVersionNegative, current64)
	case current64 >= UseLatestVersion:
		return j, fmt.Errorf("%w: %w `version`: `%d`", errConfigVersion, errConfigVersionMax, current64)
	}
	current := uint16(current64)

	switch {
	case target == current:
		return j, nil
	case latest < current:
		err := fmt.Errorf("%w: %w", errConfigVersion, errConfigVersionUnavail)
		warnVersionNotRegistered(current, latest, err)
		return j, err
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

		patch := m.versions[patchVersion]

		current = patchVersion
		if target < current {
			current = patchVersion - 1
		}

		if patch == nil {
			log.Printf("Skipping missing config version %v\n", patchVersion)
			continue
		}

		log.Printf("Running %s config version %v\n", action, patchVersion)

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

		if j, err = jsonparser.Set(j, []byte(strconv.FormatUint(uint64(current), 10)), "version"); err != nil {
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
			errs = common.AppendError(errs, fmt.Errorf("%w [%d]: %w `name`: %w", errModifyingExchange, i, common.ErrGettingField, err))
			return
		}
		for _, want := range wanted {
			if want != "*" && want != name {
				continue
			}
			exchNew, err := method(patch, ctx, exchOrig)
			if err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w for %q: %w", errModifyingExchange, name, err))
				continue
			}
			if !bytes.Equal(exchNew, exchOrig) {
				if j, err = jsonparser.Set(j, exchNew, "exchanges", "["+strconv.Itoa(i)+"]"); err != nil {
					errs = common.AppendError(errs, fmt.Errorf("%w %q/exchanges[%d]: %w: %w", errModifyingExchange, name, i, common.ErrSettingField, err))
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
func (m *manager) registerVersion(ver uint16, v any) {
	m.m.Lock()
	defer m.m.Unlock()
	if int(ver) >= len(m.versions) {
		m.versions = slices.Grow(m.versions, int(ver+1))[:ver+1]
	}
	if m.versions[ver] != nil {
		panic(fmt.Errorf("%w: %d", errAlreadyRegistered, ver))
	}
	m.versions[ver] = v
}

// Version returns a version registered by init or nil if nothing has been registered with that version number
func (m *manager) Version(version uint16) any {
	m.m.RLock()
	defer m.m.RUnlock()
	if int(version) < len(m.versions) {
		return m.versions[version]
	}
	return nil
}

// latest returns the highest version number
func (m *manager) latest() (uint16, error) {
	m.m.RLock()
	defer m.m.RUnlock()
	if len(m.versions) == 0 {
		return 0, errNoVersions
	}
	return uint16(len(m.versions)) - 1, nil //nolint:gosec // Ignore this as we hardcode version numbers
}

// checkVersions ensures that registered versions are consistent
func (m *manager) checkVersions() error {
	m.m.RLock()
	defer m.m.RUnlock()
	for ver, v := range m.versions {
		switch v.(type) {
		case ExchangeVersion, ConfigVersion, nil:
		default:
			return fmt.Errorf("%w: %v", errVersionIncompatible, ver)
		}
	}
	return nil
}

func warnVersionNotRegistered(current, latest uint16, msg error) {
	fmt.Fprintf(os.Stderr, `
%s ('%d' > '%d')
Switch back to the version of GoCryptoTrader containing config version '%d' and run:
$ ./cmd/config downgrade %d 
`, msg, current, latest, current, latest)
}
