package rgrpc

import (
	"math"
	"os"
	"time"
)

const (
	_serverMaxConnectionAgeEnv = "GRPC_MAX_CONNECTION_AGE"

	// same default as grpc
	infinity                 = time.Duration(math.MaxInt64)
	_defaultMaxConnectionAge = infinity
)

// GetMaxConnectionAge returns the maximum grpc connection age.
func GetMaxConnectionAge() time.Duration {
	d, err := time.ParseDuration(os.Getenv(_serverMaxConnectionAgeEnv))
	if err != nil {
		return _defaultMaxConnectionAge
	}
	return d
}
