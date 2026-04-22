package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

func TestEnsureExistsDir(t *testing.T) {
	var tmpDir = t.TempDir()

	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "creates a dir if it does not exist",
			args: args{
				uri: filepath.Join(tmpDir, "example"),
			},
			wantErr: false,
		},
		{
			name: "noop if the target directory exists",
			args: args{
				uri: filepath.Join(tmpDir, "example"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ensureExistsDir(tt.args.uri); (err != nil) != tt.wantErr {
				t.Errorf("ensureExistsDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPersistKey(t *testing.T) {
	p256 := elliptic.P256()
	var (
		tmpDir     = t.TempDir()
		keyPath    = filepath.Join(tmpDir, "opencloud", "testKey")
		rsaPk, _   = rsa.GenerateKey(rand.Reader, 2048)
		ecdsaPk, _ = ecdsa.GenerateKey(p256, rand.Reader)
	)

	type args struct {
		keyName string
		pk      any
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "writes a private key (rsa) to the specified location",
			args: args{
				keyName: keyPath,
				pk:      rsaPk,
			},
		},
		{
			name: "writes a private key (ecdsa) to the specified location",
			args: args{
				keyName: keyPath,
				pk:      ecdsaPk,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := persistKey(tt.args.keyName, log.NopLogger(), tt.args.pk); err != nil {
				t.Error(err)
			}
		})

		// side effect: tt.args.keyName is created
		if _, err := os.Stat(tt.args.keyName); err != nil {
			t.Errorf("persistKey() error = %v", err)
		}
	}
}

func TestPersistCertificate(t *testing.T) {
	p256 := elliptic.P256()
	var (
		tmpDir     = t.TempDir()
		certPath   = filepath.Join(tmpDir, "opencloud", "testCert")
		rsaPk, _   = rsa.GenerateKey(rand.Reader, 2048)
		ecdsaPk, _ = ecdsa.GenerateKey(p256, rand.Reader)
	)

	type args struct {
		certName string
		pk       any
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "store a certificate with an rsa private key",
			args: args{
				certName: certPath,
				pk:       rsaPk,
			},
			wantErr: false,
		},
		{
			name: "store a certificate with an ecdsa private key",
			args: args{
				certName: certPath,
				pk:       ecdsaPk,
			},
			wantErr: false,
		},
		{
			name: "should fail",
			args: args{
				certName: certPath,
				pk:       42,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run(tt.name, func(t *testing.T) {
				if err := persistCertificate(tt.args.certName, log.NopLogger(), tt.args.pk); err != nil {
					if !tt.wantErr {
						t.Error(err)
					}
				}
			})

			// side effect: tt.args.keyName is created
			if _, err := os.Stat(tt.args.certName); err != nil {
				t.Errorf("persistCertificate() error = %v", err)
			}
		})
	}
}
