package version

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/opencloud-eu/reva/v2/pkg/logger"
)

const (

	// Dev is used as a placeholder.
	Dev = "dev"
	// EditionDev indicates the development build channel was used to build the binary.
	EditionDev = Dev
	// EditionRolling indicates the rolling release build channel was used to build the binary.
	EditionRolling = "rolling"
	// EditionStable indicates the stable release build channel was used to build the binary.
	EditionStable = "stable"
	// EditionLTS indicates the lts release build channel was used to build the binary.
	EditionLTS = "lts"
)

var (
	// String gets defined by the build system
	String string

	// Tag gets defined by the build system
	Tag string

	// LatestTag is the latest released version plus the dev meta version.
	// Will be overwritten by the release pipeline
	// Needs a manual change for every tagged release
	LatestTag = "5.1.0+dev"

	// Date indicates the build date.
	// This has been removed, it looks like you can only replace static strings with recent go versions
	//Date = time.Now().Format("20060102")
	Date = Dev

	// Legacy defines the old long 4 number OpenCloud version needed for some clients
	Legacy = "0.1.0.0"
	// LegacyString defines the old OpenCloud version needed for some clients
	LegacyString = "0.1.0"

	// Edition describes the build channel (stable, rolling, nightly, daily, dev)
	Edition = Dev // default for self-compiled builds
)

func init() { //nolint:gochecknoinits
	if err := initEdition(); err != nil {
		logger.New().Error().Err(err).Msg("falling back to dev")
	}
}

func initEdition() error {
	regularEditions := []string{EditionDev, EditionRolling, EditionStable}
	versionedEditions := []string{EditionLTS}
	if !slices.ContainsFunc(slices.Concat(regularEditions, versionedEditions), func(s string) bool {
		isRegularEdition := slices.Contains(regularEditions, Edition)
		if isRegularEdition && s == Edition {
			return true
		}

		// handle editions with a version
		editionParts := strings.Split(Edition, "-")
		if len(editionParts) != 2 { // a versioned edition channel must consist of exactly 2 parts.
			return false
		}

		isVersionedEdition := slices.Contains(versionedEditions, editionParts[0])
		if !isVersionedEdition { // not all channels can contain version information
			return false
		}

		_, err := semver.NewVersion(editionParts[1])
		return err == nil
	}) {
		defer func() {
			Edition = Dev
		}()

		return fmt.Errorf(`unknown edition channel '%s'`, Edition)
	}

	return nil
}

// Compiled returns the compile time of this service.
func Compiled() time.Time {
	if Date == Dev {
		return time.Now()
	}
	t, _ := time.Parse("20060102", Date)
	return t
}

// GetString returns a version string with pre-releases and metadata
func GetString() string {
	return Parsed().String()
}

// Parsed returns a semver Version
func Parsed() (version *semver.Version) {
	versionToParse := LatestTag
	// use the placeholder version if the tag is empty or when we are creating a daily build
	if Tag != "" && Tag != "daily" {
		versionToParse = Tag
	}
	version, err := semver.NewVersion(versionToParse)
	if err != nil {
		// this should never happen
		return &semver.Version{}
	}
	if String != "" {
		// We have no tagged version but a commitid
		nVersion, err := version.SetMetadata(String)
		if err != nil {
			return &semver.Version{}
		}
		version = &nVersion
	}
	return version
}

// ParsedLegacy returns the legacy version
func ParsedLegacy() *semver.Version {
	parsedVersion, err := semver.NewVersion(LegacyString)
	if err != nil {
		return &semver.Version{}
	}
	return parsedVersion
}
