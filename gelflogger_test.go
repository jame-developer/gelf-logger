package gelflogger

import (
	"crypto/tls"
	"net"
	"testing"
)

// startMockServer creates a mock TCP server and returns a network listener.
// The server listens on the specified TCP port on the loopback address, "127.0.0.1".
// The returned listener should be closed after use to free the associated resources.
func startMockServer(t *testing.T) net.Listener {
	mockServerPort := "5555"
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", mockServerPort))
	if err != nil {
		t.Fatalf("Failed to start mock TCP server: %v", err)
	}
	return l
}

// startMockTLSServer creates a mock TLS TCP server and returns a network listener.
// It loads the X.509 key pair from the given PEM-encoded files, 'testcert.pem' and 'testkey.pem', for TLS configuration.
// If the key pair cannot be loaded, the function aborts with a fatal error.
// The server listens on the specified TCP port on the loopback address, "127.0.0.1".
// The returned listener should be closed after use to free the associated resources.
// To create these test certificate files, you can use OpenSSL with the following commands in your `test_data` folder under project root:
//
//	`openssl req -newkey rsa:2048 -nodes -keyout testkey.pem -x509 -days 365 -out testcert.pem`
func startMockTLSServer(t *testing.T) net.Listener {
	mockTLSServerPort := "5556"
	cert, err := tls.LoadX509KeyPair("./test_data/testcert.pem", "./test_data/testkey.pem")
	if err != nil {
		t.Fatalf("Failed to load keypair for TLS: %v", err)
	}
	l, err := tls.Listen("tcp", net.JoinHostPort("127.0.0.1", mockTLSServerPort), &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("Failed to start mock TLS TCP server: %v", err)
	}
	return l
}

func TestNewLogger(t *testing.T) {
	// Set up the mock server here
	mockServer := startMockServer(t)
	mockTLSServer := startMockTLSServer(t)

	defer t.Cleanup(func() {
		_ = mockServer.Close()
		_ = mockTLSServer.Close()
	})
	var tests = []struct {
		name      string
		address   string
		useTLS    bool
		tlsConfig *tls.Config
		wantErr   bool
	}{
		{
			name:    "Valid TCP Address Without TLS",
			address: mockServer.Addr().String(),
			useTLS:  false,
			wantErr: false,
		},
		{
			name:    "Invalid TCP Address Without TLS",
			address: "invalid:address",
			useTLS:  false,
			wantErr: true,
		},
		{
			name:      "Valid TCP Address With TLS",
			address:   mockTLSServer.Addr().String(),
			useTLS:    true,
			tlsConfig: &tls.Config{InsecureSkipVerify: true},
			wantErr:   false,
		},
		{
			name:      "Invalid TCP Address With TLS",
			address:   "invalid:address",
			useTLS:    true,
			tlsConfig: &tls.Config{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLogger(tt.address, tt.useTLS, tt.tlsConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteWithMockServer(t *testing.T) {
	// Set up the mock server here
	mockServer := startMockServer(t)

	defer t.Cleanup(func() {
		_ = mockServer.Close()
	})
	tests := []struct {
		name                 string
		message              []byte
		stopServerBeforeTest bool
		wantN                int
		wantErr              bool
	}{
		{
			name:    "valid message",
			message: []byte(`{"message":"info", "test_field": "TEST FIELD VALUE"}`),
			wantN:   52,
			wantErr: false,
		},
		{
			name:    "invalid json",
			message: []byte(`{"message":}`),
			wantN:   0,
			wantErr: true,
		},
		{
			name:    "empty message",
			message: []byte(`{"message":""}`),
			wantN:   14,
			wantErr: false,
		},
		{
			name:    "non-string message",
			message: []byte(`{"message":1234}`),
			wantN:   0,
			wantErr: true,
		},
		{
			name:    "invalid JSON message",
			message: []byte(`{"message":1234`),
			wantN:   0,
			wantErr: true,
		},
		{
			name:                 "server gone",
			message:              []byte(`{"message":"info"}`),
			stopServerBeforeTest: true,
			wantN:                0,
			wantErr:              true,
		},
		{
			name:    "invalid time value",
			message: []byte(`{"message":"info", "time":"2024-01-01T01:01:01"}`),
			wantN:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Initialize our logger with the mock server's address
			logger, _ := NewLogger(mockServer.Addr().String(), false, nil)

			gw := &GelfWriter{
				Logger: logger,
			}
			if tt.stopServerBeforeTest {
				_ = mockServer.Close()
			}
			defer func() {
				if tt.stopServerBeforeTest {
					mockServer = startMockServer(t)
				}
			}()

			gotN, err := gw.Write(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("Write() gotN = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
