package scanners_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/antivirus/pkg/scanners"
)

func newUnixListener(t testing.TB, lc net.ListenConfig, v ...string) net.Listener {
	d, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(d))
	})

	nl, err := lc.Listen(context.Background(), "unix", filepath.Join(d, "sock"))
	require.NoError(t, err)

	go func() {
		i := 0
		for {
			if len(v) == i {
				break
			}

			conn, err := nl.Accept()
			require.NoError(t, err)

			time.Sleep(100 * time.Millisecond)

			_, err = conn.Write([]byte(v[i]))
			require.NoError(t, err)
			require.NoError(t, conn.Close())
			i++
		}
	}()

	return nl
}

func TestNewClamAV(t *testing.T) {
	t.Run("returns a scanner", func(t *testing.T) {
		ul := newUnixListener(t, net.ListenConfig{}, "PONG\n")
		defer func() {
			assert.NoError(t, ul.Close())
		}()

		done := make(chan bool, 1)

		go func() {
			_, err := scanners.NewClamAV(ul.Addr().String(), 10*time.Second)
			assert.NoError(t, err)
			done <- true
		}()

		assert.True(t, <-done)
	})

	t.Run("fails if scanner is not pingable", func(t *testing.T) {
		_, err := scanners.NewClamAV("", 0)
		assert.ErrorIs(t, err, scanners.ErrScannerNotReachable)
	})
}

func TestNewClamAV_Scan(t *testing.T) {
	t.Run("returns a result", func(t *testing.T) {
		ul := newUnixListener(t, net.ListenConfig{}, "PONG\n", "stream: Win.Test.EICAR_HDB-1 FOUND\n")
		defer func() {
			assert.NoError(t, ul.Close())
		}()

		done := make(chan bool, 1)

		go func() {
			scanner, err := scanners.NewClamAV(ul.Addr().String(), 10*time.Second)
			assert.NoError(t, err)

			result, err := scanner.Scan(scanners.Input{Body: strings.NewReader("DATA")})
			assert.NoError(t, err)

			assert.Equal(t, result.Description, "Win.Test.EICAR_HDB-1")
			assert.True(t, result.Infected)
			done <- true
		}()

		assert.True(t, <-done)
	})

	t.Run("aborts after a certain time", func(t *testing.T) {
		ul := newUnixListener(t, net.ListenConfig{}, "PONG\n", "stream: Win.Test.EICAR_HDB-1 FOUND\n")
		defer func() {
			assert.NoError(t, ul.Close())
		}()

		done := make(chan bool, 1)

		go func() {
			scanner, err := scanners.NewClamAV(ul.Addr().String(), 10*time.Second)
			assert.NoError(t, err)

			result, err := scanner.Scan(scanners.Input{Body: strings.NewReader("DATA")})
			assert.NoError(t, err)

			assert.Equal(t, result.Description, "Win.Test.EICAR_HDB-1")
			assert.True(t, result.Infected)
			done <- true
		}()

		assert.True(t, <-done)
	})
}
