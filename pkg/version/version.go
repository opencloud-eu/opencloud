package version

import (
	"time"

	"github.com/Masterminds/semver"
)

var (
	// String gets defined by the build system
	String string

	// Tag gets defined by the build system
	Tag string

	// LatestTag is the latest released version plus the dev meta version.
	// Will be overwritten by the release pipeline
	// Needs a manual change for every tagged release
	LatestTag = "4.0.0-rc.3+dev"

	// Date indicates the build date.
	// This has been removed, it looks like you can only replace static strings with recent go versions
	//Date = time.Now().Format("20060102")
	Date = "dev"

	// Legacy defines the old long 4 number OpenCloud version needed for some clients
	Legacy = "0.1.0.0"
	// LegacyString defines the old OpenCloud version needed for some clients
	LegacyString = "0.1.0"
)

// Compiled returns the compile time of this service.
func Compiled() time.Time {
	if Date == "dev" {
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
