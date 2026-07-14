package network

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/testcontainers/testcontainers-go"
)

// New creates a new network with a random UUID name, calling the already existing GenericNetwork APIs.
// Those existing APIs are deprecated and will be removed in the future, so this function will
// implement the new network APIs when they will be available.
// By default, the network is created with the following options:
// - Driver: bridge
// - Labels: the Testcontainers for Go generic labels, to be managed by Ryuk. Please see the GenericLabels() function
// And those options can be modified by the user, using the CreateModifier function field.
func New(ctx context.Context, opts ...NetworkCustomizer) (*testcontainers.DockerNetwork, error) {
	nc := client.NetworkCreateOptions{
		Driver: "bridge",
		Labels: testcontainers.GenericLabels(),
	}

	for _, opt := range opts {
		if err := opt.Customize(&nc); err != nil {
			return nil, err
		}
	}

	//nolint:staticcheck
	netReq := testcontainers.NetworkRequest{
		Driver:     nc.Driver,
		Internal:   nc.Internal,
		EnableIPv6: nc.EnableIPv6,
		Name:       uuid.NewString(),
		Labels:     nc.Labels,
		Attachable: nc.Attachable,
		IPAM:       nc.IPAM,
	}

	//nolint:staticcheck
	n, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: netReq,
	})
	if err != nil {
		return nil, err
	}

	// Return a DockerNetwork struct instead of the Network interface,
	// following the "accept interface, return struct" pattern.
	return n.(*testcontainers.DockerNetwork), nil
}

// NetworkCustomizer is an interface that can be used to configure the network create request.
type NetworkCustomizer interface {
	Customize(req *client.NetworkCreateOptions) error
}

// CustomizeNetworkOption is a type that can be used to configure the network create request.
type CustomizeNetworkOption func(req *client.NetworkCreateOptions) error

// Customize implements the NetworkCustomizer interface,
// applying the option to the network create request.
func (opt CustomizeNetworkOption) Customize(req *client.NetworkCreateOptions) error {
	return opt(req)
}

// WithAttachable allows to set the network as attachable.
func WithAttachable() CustomizeNetworkOption {
	return func(original *client.NetworkCreateOptions) error {
		original.Attachable = true

		return nil
	}
}

// WithCheckDuplicate allows to check if a network with the same name already exists.
//
// Deprecated: CheckDuplicate is deprecated since API v1.44, but it defaults to true when sent by the client package to older daemons.
func WithCheckDuplicate() CustomizeNetworkOption {
	return func(_ *client.NetworkCreateOptions) error {
		return nil
	}
}

// WithDriver allows to override the default network driver, which is "bridge".
func WithDriver(driver string) CustomizeNetworkOption {
	return func(original *client.NetworkCreateOptions) error {
		original.Driver = driver

		return nil
	}
}

// WithEnableIPv6 allows to set the network as IPv6 enabled.
// Please use this option if and only if IPv6 is enabled on the Docker daemon.
func WithEnableIPv6() CustomizeNetworkOption {
	return func(original *client.NetworkCreateOptions) error {
		enableIPv6 := true
		original.EnableIPv6 = &enableIPv6
		return nil
	}
}

// WithInternal allows to set the network as internal.
func WithInternal() CustomizeNetworkOption {
	return func(original *client.NetworkCreateOptions) error {
		original.Internal = true

		return nil
	}
}

// WithLabels allows to set the network labels, adding the new ones
// to the default Testcontainers for Go labels.
func WithLabels(labels map[string]string) CustomizeNetworkOption {
	return func(original *client.NetworkCreateOptions) error {
		maps.Copy(original.Labels, labels)

		return nil
	}
}

// WithIPAM allows to change the default IPAM configuration.
func WithIPAM(ipam *network.IPAM) CustomizeNetworkOption {
	return func(original *client.NetworkCreateOptions) error {
		original.IPAM = ipam

		return nil
	}
}

// WithNetwork reuses an already existing network, attaching the container to it.
// Finally it sets the network alias on that network to the given alias.
func WithNetwork(aliases []string, nw *testcontainers.DockerNetwork) testcontainers.CustomizeRequestOption {
	return WithNetworkName(aliases, nw.Name)
}

// WithNetworkName attachs a container to an already existing network, by its name.
// If the network is not "bridge", it sets the network alias on that network
// to the given alias, else, it returns an error. This is because network-scoped alias
// is supported only for containers in user defined networks.
func WithNetworkName(aliases []string, networkName string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		if networkName == "bridge" {
			return errors.New("network-scoped aliases are supported only for containers in user defined networks")
		}

		// attaching to the network because it was created with success or it already existed.
		req.Networks = append(req.Networks, networkName)

		if req.NetworkAliases == nil {
			req.NetworkAliases = make(map[string][]string)
		}
		req.NetworkAliases[networkName] = aliases

		return nil
	}
}

// WithBridgeNetwork attachs a container to the "bridge" network.
// There is no need to set the network alias, as it is not supported for the "bridge" network.
func WithBridgeNetwork() testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		req.Networks = append(req.Networks, "bridge")
		return nil
	}
}

// WithNewNetwork creates a new network with random name and customizers, and attaches the container to it.
// Finally it sets the network alias on that network to the given alias.
func WithNewNetwork(ctx context.Context, aliases []string, opts ...NetworkCustomizer) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.GenericContainerRequest) error {
		newNetwork, err := New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("new network: %w", err)
		}

		networkName := newNetwork.Name

		// attaching to the network because it was created with success or it already existed.
		req.Networks = append(req.Networks, networkName)

		if req.NetworkAliases == nil {
			req.NetworkAliases = make(map[string][]string)
		}
		req.NetworkAliases[networkName] = aliases

		return nil
	}
}
