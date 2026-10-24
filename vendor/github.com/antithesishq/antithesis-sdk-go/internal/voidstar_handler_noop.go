//go:build enable_antithesis_sdk && (!linux || !amd64)

package internal

const sdkRunningInDegradedMode = false

func init_in_antithesis() libHandler {
	return nil
}
