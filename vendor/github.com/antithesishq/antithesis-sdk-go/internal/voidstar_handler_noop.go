//go:build !no_antithesis_sdk && (!linux || !amd64 || !cgo)

package internal

func init_in_antithesis() libHandler {
	return nil
}
