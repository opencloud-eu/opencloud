package scanners

import (
	"errors"
	"io"
	"time"
)

var (
	// ErrScanTimeout is returned when a scan times out
	ErrScanTimeout = errors.New("time out waiting for clamav to respond while scanning")
	// ErrScannerNotReachable is returned when the scanner is not reachable
	ErrScannerNotReachable = errors.New("failed to reach the scanner")
)

type (
	// The Result is the common scan result to all scanners
	Result struct {
		Infected    bool
		ScanTime    time.Time
		Description string
	}

	// The Input is the common input to all scanners
	Input struct {
		Body io.Reader
		Size int64
		Url  string
		Name string
	}
)
