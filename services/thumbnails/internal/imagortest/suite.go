// Package imagortest starts a real imagor for the generator parity suite.
package imagortest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/defaults"
)

const (
	port           = "8000/tcp"
	startupTimeout = 2 * time.Minute
)

// Variant is one imagor image to test against. Reading an ICO needs a libvips
// with ImageMagick, which only the magick build has.
type Variant struct {
	Name  string
	Image string

	// Key is this variant's own override key in the parity suite; a divergence
	// both variants have is keyed "imagor" instead.
	Key string

	// AddressEnv names an imagor to use instead of starting one, which is how
	// CI hands in its service containers.
	AddressEnv string
}

// Variants pins the images on purpose, imagor is the reference. Bump IMAGOR and
// IMAGOR_MAGICK in .woodpecker.star along with them.
func Variants() []Variant {
	return []Variant{
		{
			Name: "imagor", Key: "imagor-default",
			Image:      "shumc/imagor:1.9.11",
			AddressEnv: "THUMBNAILS_PARITY_IMAGOR_URL",
		},
		{
			Name: "imagor-magick", Key: "imagor-magick",
			Image:      "ghcr.io/cshum/imagor-magick:1.9.11",
			AddressEnv: "THUMBNAILS_PARITY_IMAGOR_MAGICK_URL",
		},
	}
}

// push based processing only: unsigned URLs, uploads in, nothing loaded by URL.
// The vips limits are ours, imagor's own default edge is 9999.
func Args() []string {
	limits := defaults.DefaultConfig().Thumbnail

	return []string{
		"-imagor-unsafe", "-upload-loader-enable", "-http-loader-disable",
		"-vips-max-width", strconv.Itoa(limits.MaxInputWidth),
		"-vips-max-height", strconv.Itoa(limits.MaxInputHeight),
	}
}

// prefers an address from the variant's AddressEnv over starting a container.
func Setup(ctx context.Context, v Variant) (string, func(), error) {
	if addr := os.Getenv(v.AddressEnv); addr != "" {
		return strings.TrimRight(addr, "/"), func() {}, nil
	}

	external := slices.Contains([]bool{
		os.Getenv("CI") == "woodpecker",
		os.Getenv("CI_SYSTEM_NAME") == "woodpecker",
		os.Getenv("USE_TESTCONTAINERS") == "false",
	}, true)
	if external {
		return "", nil, fmt.Errorf("no imagor address in %s and container startup is disabled", v.AddressEnv)
	}

	keep := os.Getenv("KEEP_TEST_CONTAINER") == "true"
	if keep {
		// the reaper would take the kept container down with the session
		if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
			return "", nil, fmt.Errorf("disable the testcontainers reaper: %w", err)
		}
	}

	name := strings.NewReplacer("/", "__", ":", "_").Replace("opencloud/test/" + v.Image)
	container, err := testcontainers.Run(ctx, v.Image,
		testcontainers.WithExposedPorts(port),
		testcontainers.WithCmd(Args()...),
		testcontainers.WithName(name),
		testcontainers.WithReuseByName(name),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/").
				WithPort(port).
				WithStatusCodeMatcher(func(status int) bool { return status == http.StatusOK }).
				WithStartupTimeout(startupTimeout),
		),
	)
	if err != nil {
		return "", nil, fmt.Errorf("start %s container: %w", v.Name, err)
	}

	endpoint, err := container.PortEndpoint(ctx, port, "http")
	if err != nil {
		_ = container.Terminate(ctx)
		return "", nil, fmt.Errorf("%s container endpoint: %w", v.Name, err)
	}

	return strings.TrimRight(endpoint, "/"), func() {
		if keep {
			_, _ = fmt.Fprintf(os.Stderr, "keeping container %s\n", name)
			return
		}
		if err := container.Terminate(ctx); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to terminate the %s container: %v\n", v.Name, err)
		}
	}, nil
}
