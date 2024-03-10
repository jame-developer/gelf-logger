package helper

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

// StartMockServer creates a mock TCP server and returns a network listener.
// The server listens on the specified TCP port on the loopback address, "127.0.0.1".
// The returned listener should be closed after use to free the associated resources.
func StartMockServer(t *testing.T) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock TCP server: %v", err)
	}
	return l
}

// StartMockTLSServer creates a mock TLS TCP server and returns a network listener.
// It loads the X.509 key pair from the given PEM-encoded files, 'testcert.pem' and 'testkey.pem', for TLS configuration.
// If the key pair cannot be loaded, the function aborts with a fatal error.
// The server listens on the specified TCP port on the loopback address, "127.0.0.1".
// The returned listener should be closed after use to free the associated resources.
// To create these test certificate files, you can use OpenSSL with the following commands in your `test_data` folder under project root:
//
//	`openssl req -newkey rsa:2048 -nodes -keyout testkey.pem -x509 -days 365 -out testcert.pem`
func StartMockTLSServer(t *testing.T) net.Listener {

	cert := CreateTestCertificate()
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("Failed to start mock TLS TCP server: %v", err)
	}
	return l
}

func CreateTestCertificate() tls.Certificate {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Acme Co"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 180),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	derBytes, _ := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	return tls.Certificate{Certificate: [][]byte{derBytes}, PrivateKey: privateKey}
}
