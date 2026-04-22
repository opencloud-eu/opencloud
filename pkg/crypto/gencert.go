package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/opencloud-eu/opencloud/pkg/log"
	mtls "go-micro.dev/v4/util/tls"
)

var (
	defaultHosts = []string{"127.0.0.1", "localhost"}
)

// GenCert generates TLS-Certificates. This function has side effects: it creates the respective certificate / key pair at
// the destination locations unless the tuple already exists, if that is the case, this is a noop.
func GenCert(certName string, keyName string, l log.Logger) error {
	var pk *rsa.PrivateKey
	var err error

	pk, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	_, certErr := os.Stat(certName)
	_, keyErr := os.Stat(keyName)

	if certErr == nil || keyErr == nil {
		l.Info().Msg(
			fmt.Sprintf("%v certificate / key pair already present. skipping acme certificate generation",
				filepath.Base(certName)))
		return nil
	}

	if err := persistCertificate(certName, l, pk); err != nil {
		l.Fatal().Err(err).Msg("failed to store certificate")
	}

	if err := persistKey(keyName, l, pk); err != nil {
		l.Fatal().Err(err).Msg("failed to store key")
	}

	return nil
}

// GenTempCertForAddr generates temporary TLS-Certificates in memory.
func GenTempCertForAddr(addr string) (tls.Certificate, error) {
	subjects := defaultHosts

	if host, _, err := net.SplitHostPort(addr); err == nil && host != "" {
		subjects = []string{host}
	}
	return mtls.Certificate(subjects...)
}

// persistCertificate generates a certificate using pk as private key and proceeds to store it into a file named certName.
func persistCertificate(certName string, l log.Logger, pk any) error {
	if err := ensureExistsDir(certName); err != nil {
		return fmt.Errorf("creating certificate destination: %s", certName)
	}

	certificate, err := generateCertificate(pk)
	if err != nil {
		return fmt.Errorf("creating certificate: %s", filepath.Dir(certName))
	}

	certOut, err := os.Create(certName)
	if err != nil {
		return fmt.Errorf("failed to open `%v` for writing", certName)
	}

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certificate})
	if err != nil {
		return fmt.Errorf("failed to encode certificate")
	}

	err = certOut.Close()
	if err != nil {
		return fmt.Errorf("failed to write cert")
	}
	l.Info().Msg(fmt.Sprintf("written certificate to %v", certName))

	return nil
}

// genCert generates a self signed certificate using a random rsa key.
func generateCertificate(pk any) ([]byte, error) {
	for _, h := range defaultHosts {
		if ip := net.ParseIP(h); ip != nil {
			acmeTemplate.IPAddresses = append(acmeTemplate.IPAddresses, ip)
		} else {
			acmeTemplate.DNSNames = append(acmeTemplate.DNSNames, h)
		}
	}

	return x509.CreateCertificate(rand.Reader, &acmeTemplate, &acmeTemplate, publicKey(pk), pk)
}

// persistKey persists the private key used to generate the certificate at the configured location.
func persistKey(destination string, l log.Logger, pk any) error {
	if err := ensureExistsDir(destination); err != nil {
		return fmt.Errorf("creating key destination: %s", destination)
	}

	keyOut, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %v for writing", destination)
	}
	err = pem.Encode(keyOut, pemBlockForKey(pk, l))
	if err != nil {
		return fmt.Errorf("failed to encode key")
	}

	err = keyOut.Close()
	if err != nil {
		return fmt.Errorf("failed to write key")
	}
	l.Info().Msg(fmt.Sprintf("written key to %v", destination))

	return nil
}

func publicKey(pk any) any {
	switch k := pk.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(pk any, l log.Logger) *pem.Block {
	switch k := pk.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			l.Fatal().Err(err).Msg("Unable to marshal ECDSA private key")
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func ensureExistsDir(uri string) error {
	certPath := filepath.Dir(uri)
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		err = os.MkdirAll(certPath, 0700)
		if err != nil {
			return err
		}
	}
	return nil
}
