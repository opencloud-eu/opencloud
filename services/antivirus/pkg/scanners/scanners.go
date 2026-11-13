package scanners

import (
	"errors"
	"io"
	"time"
)

var (
	// ErrScanTimeout is returned when a scan times out
	ErrScanTimeout = errors.New("time out waiting for clamav to respond while scanning")
	// ErrScanFailed is returned when the scanner answers with an error instead of a verdict
	ErrScanFailed = errors.New("scanner failed to scan the file")
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
