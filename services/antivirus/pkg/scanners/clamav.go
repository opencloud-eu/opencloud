package scanners

import (
	"fmt"
	"time"

	"github.com/dutchcoders/go-clamd"
)

// NewClamAV returns a Scanner talking to clamAV via socket
func NewClamAV(socket string, timeout time.Duration) (*ClamAV, error) {
	c := clamd.NewClamd(socket)

	if err := c.Ping(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrScannerNotReachable, err)
	}

	return &ClamAV{
		clamd:   clamd.NewClamd(socket),
		timeout: timeout,
	}, nil
}

// ClamAV is a Scanner based on clamav
type ClamAV struct {
	clamd   *clamd.Clamd
	timeout time.Duration
}

// Scan to fulfill Scanner interface
func (s ClamAV) Scan(in Input) (Result, error) {
	abort := make(chan bool, 1)
	defer close(abort)

	ch, err := s.clamd.ScanStream(in.Body, abort)
	if err != nil {
		return Result{}, err
	}

	select {
	case <-time.After(s.timeout):
		abort <- true
		return Result{}, fmt.Errorf("%w: %s", ErrScanTimeout, in.Url)
	case s := <-ch:
		return Result{
			Infected:    s.Status == clamd.RES_FOUND,
			Description: s.Description,
			ScanTime:    time.Now(),
		}, nil
	}
}
