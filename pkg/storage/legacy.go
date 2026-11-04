package storage

import "runtime"

// LegacyMetadataPrefix is the metadata namespace used by the former ocis and
// s3ng storage drivers.
func LegacyMetadataPrefix() string {
	if runtime.GOOS == "freebsd" {
		return "ocis."
	}
	return "user.ocis."
}

// EffectiveDriver resolves OpenCloud's deprecated driver names.
func EffectiveDriver(driver string) string {
	switch driver {
	case "ocis":
		return "decomposed"
	case "s3ng":
		return "decomposeds3"
	default:
		return driver
	}
}

// EffectiveMetadataPrefix preserves the prefix used by the legacy drivers,
// which did not honor the configurable metadata prefix.
func EffectiveMetadataPrefix(driver, configured string) string {
	if EffectiveDriver(driver) != driver {
		return LegacyMetadataPrefix()
	}
	return configured
}
