package command

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/shared"
	searchsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type fakeSearchProvider struct {
	searchsvc.UnimplementedSearchProviderServer
}

func (fakeSearchProvider) IndexSpace(_ *searchsvc.IndexSpaceRequest, stream grpc.ServerStreamingServer[searchsvc.IndexSpaceResponse]) error {
	return stream.Send(&searchsvc.IndexSpaceResponse{SpaceId: "space-1", IndexedSpaces: 1, TotalSpaces: 1})
}

// startSearchServer serves a fake search service on a random local port and
// returns its address.
func startSearchServer(t *testing.T, opts ...grpc.ServerOption) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer(opts...)
	searchsvc.RegisterSearchProviderServer(srv, fakeSearchProvider{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

// selfSignedCert returns a certificate for 127.0.0.1 and the path of a PEM
// file holding it, to be used as the CA certificate by the client.
func selfSignedCert(t *testing.T) (tls.Certificate, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	cert, err := tls.X509KeyPair(certPEM, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	require.NoError(t, err)
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caFile, certPEM, 0o600))
	return cert, caFile
}

func TestIndexHonorsGRPCClientTLSMode(t *testing.T) {
	plain := startSearchServer(t)
	cert, caFile := selfSignedCert(t)
	withTLS := startSearchServer(t, grpc.Creds(credentials.NewServerTLSFromCert(&cert)))

	tests := []struct {
		name        string
		endpoint    string
		mode        string
		caCert      string
		insecure    bool
		wantErr     bool
		errContains string
	}{
		{name: "no TLS, mode unset", endpoint: plain},
		{name: "no TLS, mode off", endpoint: plain, mode: "off"},
		{name: "no TLS, --insecure", endpoint: plain, insecure: true},
		{name: "TLS, mode insecure", endpoint: withTLS, mode: "insecure"},
		{name: "TLS, mode on with CA certificate", endpoint: withTLS, mode: "on", caCert: caFile},
		{name: "no TLS, mode insecure", endpoint: plain, mode: "insecure", wantErr: true},
		{name: "TLS, mode off", endpoint: withTLS, mode: "off", wantErr: true},
		{name: "unknown mode", endpoint: plain, mode: "bogus", wantErr: true, errContains: "unknown TLS mode"},
		{name: "unknown mode, --insecure", endpoint: plain, mode: "bogus", insecure: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				GRPCClientTLS:         &shared.GRPCClientTLS{Mode: tt.mode, CACert: tt.caCert},
				ReindexMaxConcurrency: 3,
			}
			cmd := Index(cfg)
			require.NoError(t, cmd.Flags().Set("all-spaces", "true"))
			require.NoError(t, cmd.Flags().Set("endpoint", tt.endpoint))
			if tt.insecure {
				require.NoError(t, cmd.Flags().Set("insecure", "true"))
			}

			err := cmd.RunE(cmd, nil)
			switch {
			case tt.errContains != "":
				require.ErrorContains(t, err, tt.errContains)
			case tt.wantErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
			}
		})
	}
}
