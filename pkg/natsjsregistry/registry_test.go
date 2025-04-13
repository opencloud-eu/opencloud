// Package natsjsregistry contains tests for natsjsregistry.
package natsjsregistry

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/test"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
)

// TestSuite is the struct we use for tests.
type TestSuite struct {
	suite.Suite

	Server any

	Logger logger.Logger

	Ctx        context.Context
	Registries []registry.Registry

	nodes    []*registry.Node
	services []*registry.Service

	UpdateTime time.Duration

	CreateRegistry func(suite *TestSuite) (registry.Registry, error)
}

// SetupSuite setups the test suite.
func (r *TestSuite) SetupSuite() {
	r.Logger.Log(logger.InfoLevel, "Setting up suite")

	if len(r.Registries) < 2 {
		panic("at least 2 registries are required")
	}

	// Generate random ports to avoid conflicts
	basePort1 := 10000 + rand.Intn(10000) //nolint:gosec
	basePort2 := 20000 + rand.Intn(10000) //nolint:gosec

	r.nodes = []*registry.Node{
		{Id: "node0-http", Address: fmt.Sprintf("10.0.0.10:%d", basePort1)},
		{Id: "node0-grpc", Address: fmt.Sprintf("10.0.0.10:%d", basePort1+1)},
		{Id: "node0-grpcs", Address: fmt.Sprintf("10.0.0.10:%d", basePort1+2)},
		{Id: "node1-https", Address: fmt.Sprintf("10.0.0.11:%d", basePort2)},
		{Id: "node1-http2", Address: fmt.Sprintf("10.0.0.11:%d", basePort2+1)},
		{Id: "node1-drpc", Address: fmt.Sprintf("10.0.0.11:%d", basePort2+2)},
	}

	// All the test services.
	r.services = []*registry.Service{
		{Name: "micro.test.svc.0", Version: "v1", Nodes: []*registry.Node{r.nodes[0]}},
		{Name: "micro.test.svc.1", Version: "v1", Nodes: []*registry.Node{r.nodes[1]}},
		{Name: "micro.test.svc.2", Version: "v1", Nodes: []*registry.Node{r.nodes[2]}},
		{Name: "micro.test.svc.3", Version: "v1", Nodes: []*registry.Node{r.nodes[3]}},
		{Name: "micro.test.svc.4", Version: "v1", Nodes: []*registry.Node{r.nodes[4]}},
		{Name: "micro.test.svc.5", Version: "v1", Nodes: []*registry.Node{r.nodes[5]}},
	}

	for _, service := range r.services {
		r.Require().NoError(r.Registries[0].Register(service))
	}
}

// TearDownSuite is called once after all tests in the suite have been run.
func (r *TestSuite) TearDownSuite() {
	r.Logger.Logf(logger.InfoLevel, "Tearing down suite")

	for _, service := range r.services {
		r.Require().NoError(r.Registries[0].Deregister(service))
	}
}

// TestRegisterServiceNode tests the registration of a single service node.
func (r *TestSuite) TestRegisterServiceNode() {
	testNode := &registry.Node{ // Create a node for the test
		Id:      "test-node-6",
		Address: "10.0.0.12:8086",
	}
	service := &registry.Service{ // Create a Service struct
		Name:    "micro.test.svc.6",
		Version: "v1.0.0",
		Nodes:   []*registry.Node{testNode},
	}
	r.Require().NoError(r.Registries[0].Register(service))
	time.Sleep(r.UpdateTime)

	defer func() {
		err := r.Registries[0].Deregister(service)
		if err != nil {
			r.Logger.Logf(logger.ErrorLevel, "Failed to cleanup from TestRegister", "error", err)
		}
	}()

	for idx, reg := range r.Registries {
		r.Run(fmt.Sprintf("reg-%d", idx), func() {
			services, err := reg.ListServices()
			r.Require().NoError(err)

			r.Len(services, len(r.services)+1)

			services, err = reg.GetService(service.Name)
			r.Require().NoError(err)
			r.Len(services, 1)
			r.Equal(service.Version, services[0].Version)
		})
	}
}

// TestDeregister tests the deregistration of services.
func (r *TestSuite) TestDeregister() {
	// Create service structs for deregistration
	service1 := &registry.Service{
		Name:    "micro.test.deregister",
		Version: "v1",
		Nodes: []*registry.Node{
			{Id: "deregister-node1", Address: r.services[4].Nodes[0].Address},
		},
	}
	service2 := &registry.Service{
		Name:    "micro.test.deregister",
		Version: "v1",
		Nodes: []*registry.Node{
			{Id: "deregister-node2", Address: r.services[5].Nodes[0].Address},
		},
	}

	getReg, err := r.CreateRegistry(r)
	r.Require().NoError(err)

	r.Require().NoError(r.Registries[0].Register(service1))
	time.Sleep(r.UpdateTime)

	services, err := getReg.ListServices()
	r.Require().NoError(err)
	r.Len(services, len(r.services)+1)

	services, err = getReg.GetService(service1.Name)
	r.Require().NoError(err)
	r.Len(services, 1)
	r.Equal(service1.Version, services[0].Version)

	r.Require().NoError(r.Registries[0].Register(service2))
	time.Sleep(r.UpdateTime)

	services, err = getReg.GetService(service2.Name)
	r.Require().NoError(err)
	r.Len(services[0].Nodes, 2)

	r.Require().NoError(r.Registries[0].Deregister(service1))
	time.Sleep(r.UpdateTime)

	services, err = getReg.GetService(service1.Name)
	r.Require().NoError(err)
	r.Len(services, 1)

	r.Require().NoError(r.Registries[0].Deregister(service2))
	time.Sleep(r.UpdateTime)

	services, err = getReg.GetService(service1.Name)
	r.Require().ErrorIs(err, registry.ErrNotFound)
	r.Empty(services)
}

// TestGetServiceByNameAndVersion tests retrieving a service by name and version.
func (r *TestSuite) TestGetServiceByNameAndVersion() {
	const serviceName = "micro.test.get.bynameversion"
	// Create a service to register
	testNode := &registry.Node{
		Id:      r.services[3].Nodes[0].Id, // Use existing node Id for test data
		Address: r.services[3].Nodes[0].Address,
	}
	service := &registry.Service{
		Name:    serviceName,
		Version: "v1.0.0",
		Nodes:   []*registry.Node{testNode},
	}

	// Cleanup
	defer func() {
		r.Require().NoError(r.Registries[0].Deregister(service))
	}()

	// Register the service first to ensure it exists
	r.Require().NoError(r.Registries[0].Register(service))
	time.Sleep(r.UpdateTime)

	services, err := r.Registries[1].GetService(serviceName)
	r.Require().NoError(err)
	r.Len(services, 1)
	r.Equal(service.Version, services[0].Version)
}

// TestUpdateService tests updating an existing service, particularly its metadata.
func (r *TestSuite) TestUpdateService() {
	// Initial service registration
	initialNode := &registry.Node{
		Id:      r.services[0].Nodes[0].Id,
		Address: r.services[0].Nodes[0].Address,
	}
	service := &registry.Service{
		Name:    "micro.test.update",
		Version: "v1.0.0",
		Nodes:   []*registry.Node{initialNode},
	}

	// Register the service
	r.Require().NoError(r.Registries[0].Register(service))
	time.Sleep(r.UpdateTime) // Allow time for registration propagation

	// Service with updated metadata
	updatedNode := &registry.Node{ // Node details typically remain the same during metadata update
		Id:      r.services[0].Nodes[0].Id,
		Address: r.services[0].Nodes[0].Address,
	}
	updatedService := &registry.Service{
		Name:     "micro.test.update",
		Version:  "v1.0.0",
		Nodes:    []*registry.Node{updatedNode},
		Metadata: map[string]string{"updated": "true"},
	}

	// Update the service (assuming Register acts as upsert or there's an Update method)
	r.Require().NoError(r.Registries[0].Register(updatedService))
	time.Sleep(r.UpdateTime)

	services, err := r.Registries[1].GetService(service.Name)
	r.Require().NoError(err)
	r.Len(services, 1)

	// Should have metadata updated
	r.Equal("true", services[0].Metadata["updated"])

	// Cleanup
	r.Require().NoError(r.Registries[0].Deregister(updatedService))
}

// TestMetadataPersistence tests if metadata associated with a service is persisted.
func (r *TestSuite) TestMetadataPersistence() {
	testNode := &registry.Node{
		Id:      r.services[0].Nodes[0].Id,
		Address: r.services[0].Nodes[0].Address,
	}
	service := &registry.Service{
		Name:    "micro.test.metadata",
		Version: "v1.0.0",
		Nodes:   []*registry.Node{testNode},
		Metadata: map[string]string{
			"region": "us-west",
			"env":    "test",
		},
	}

	// Register the service
	r.Require().NoError(r.Registries[0].Register(service))
	time.Sleep(r.UpdateTime)

	services, err := r.Registries[1].GetService(service.Name)
	r.Require().NoError(err)
	r.Len(services, 1)

	// Verify metadata was preserved
	r.Equal(service.Metadata["region"], services[0].Metadata["region"])
	r.Equal(service.Metadata["env"], services[0].Metadata["env"])

	// Cleanup
	r.Require().NoError(r.Registries[0].Deregister(service))
}

// TestMultipleVersions tests registering and retrieving services with different versions.
func (r *TestSuite) TestMultipleVersions() {
	serviceV1 := &registry.Service{
		Name:    "micro.test.versions",
		Version: "v1.0.0",
		Nodes: []*registry.Node{
			{Id: "v1-node", Address: r.services[0].Nodes[0].Address},
		},
	}
	serviceV2 := &registry.Service{
		Name:    "micro.test.versions",
		Version: "v2.0.0",
		Nodes: []*registry.Node{
			{Id: "v2-node", Address: r.services[1].Nodes[0].Address},
		},
	}

	// Register both versions
	r.Require().NoError(r.Registries[0].Register(serviceV1))
	r.Require().NoError(r.Registries[0].Register(serviceV2))
	time.Sleep(r.UpdateTime)

	services, err := r.Registries[1].GetService(serviceV1.Name)
	r.Require().NoError(err)
	r.Require().GreaterOrEqual(len(services), 2, "Should have found both versions of the service")

	// Verify both versions are returned
	versions := map[string]bool{}
	for _, s := range services {
		versions[s.Version] = true
	}

	r.Require().True(versions["v1.0.0"], "v1.0.0 should be in the results")
	r.Require().True(versions["v2.0.0"], "v2.0.0 should be in the results")

	// Cleanup
	r.Require().NoError(r.Registries[0].Deregister(serviceV1))
	r.Require().NoError(r.Registries[0].Deregister(serviceV2))
}

// TestSameName tests the same name for different services.
func (r *TestSuite) TestSameName() {
	services := []*registry.Service{
		{
			Name:    "filter",
			Version: "v1.0.0",
			Nodes: []*registry.Node{
				{Id: "filter-http-node", Address: "10.0.1.1:8080"},
			},
			Metadata: map[string]string{"region": "us-west", "env": "staging"},
		},
		{
			Name:    "filter",
			Version: "v2.0.0",
			Nodes: []*registry.Node{
				{Id: "filter-grpc-node", Address: "10.0.1.2:8080"},
			},
			Metadata: map[string]string{"region": "us-east", "env": "staging"},
		},
		{
			Name:    "filter",
			Version: "v1.0.0",
			Nodes: []*registry.Node{
				{Id: "filter-https-node", Address: "10.0.1.3:8080"},
			},
			Metadata: map[string]string{"region": "us-west", "env": "production"},
		},
		{
			Name:    "filter",
			Version: "v1.0.0",
			Nodes: []*registry.Node{
				{Id: "filter2-http-node", Address: "10.0.1.4:8080"},
			},
			Metadata: map[string]string{"region": "us-west", "env": "staging"},
		},
		{
			Name:    "filter",
			Version: "v2.0.0",
			Nodes: []*registry.Node{
				{Id: "filter2-grpc-node", Address: "10.0.1.5:8080"},
			},
			Metadata: map[string]string{"region": "us-east", "env": "staging"},
		},
		{
			Name:    "filter",
			Version: "v1.0.0",
			Nodes: []*registry.Node{
				{Id: "filter2-https-node", Address: "10.0.1.6:8080"},
			},
			Metadata: map[string]string{"region": "us-west", "env": "production"},
		},
	}

	// Register all services
	for _, svc := range services {
		r.Require().NoError(r.Registries[0].Register(svc))
	}

	// Cleanup
	defer func() {
		for _, svc := range services {
			r.Require().NoError(r.Registries[0].Deregister(svc))
		}
	}()

	time.Sleep(r.UpdateTime)

	// Test filtering by version and other parameters
	filtered, err := r.Registries[1].GetService("filter")
	r.Require().NoError(err)
	r.Require().Len(filtered, 2, "Should find exactly two services with version v1.0.0 in us-west")
	r.Require().Equal("v1.0.0", filtered[0].Version)
}

func createRegistry(suite *TestSuite) (registry.Registry, error) {
	t := suite.T()
	t.Helper()

	reg1 := NewRegistry(registry.Logger(suite.Logger))
	require.NoError(t, reg1.Init())

	return reg1, nil
}

func createSuite(tb testing.TB) (*TestSuite, func() error) {
	tb.Helper()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)

	// Start embedded NATS server for testing
	tmpDir := tb.TempDir()

	opts := test.DefaultTestOptions
	opts.Port = -1 // Random port
	opts.JetStream = true
	opts.StoreDir = tmpDir
	// Configure JetStream
	opts.JetStreamMaxMemory = -1 // Unlimited
	opts.JetStreamMaxStore = -1  // Unlimited

	server := test.RunServer(&opts)
	require.True(tb, server.JetStreamEnabled())

	os.Setenv(_registryAddressEnv, server.ClientURL())

	// Create logger2
	logger2 := logger.NewLogger(logger.WithLevel(logger.TraceLevel))

	// Create first registry without caching
	reg1 := NewRegistry(registry.Logger(logger2))
	require.NoError(tb, reg1.Init())

	// Create second registry with caching
	reg2 := NewRegistry(registry.Logger(logger2))
	require.NoError(tb, reg2.Init())

	cleanup := func() error {
		cancel()

		server.Shutdown()
		return nil
	}

	r := &TestSuite{
		Ctx:            ctx,
		Logger:         logger2,
		Registries:     []registry.Registry{reg1, reg2},
		UpdateTime:     time.Second,
		CreateRegistry: createRegistry,
	}

	return r, cleanup
}

func TestSuiteTests(t *testing.T) {
	s, cleanup := createSuite(t)

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}
