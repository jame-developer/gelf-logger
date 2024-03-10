package gelflogger_test

import (
	"crypto/tls"
	gelflogger "github.com/jame-developer/gelf-logger"
	"github.com/jame-developer/gelf-logger/pkg/helper"
	"testing"
)

func TestNewLogger(t *testing.T) {
	// Set up the mock server here
	mockServer := helper.StartMockServer(t)
	mockTLSServer := helper.StartMockTLSServer(t)

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
			_, err := gelflogger.NewLogger(tt.address, tt.useTLS, tt.tlsConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteWithMockServer(t *testing.T) {
	// Set up the mock server here
	mockServer := helper.StartMockServer(t)

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
			logger, _ := gelflogger.NewLogger(mockServer.Addr().String(), false, nil)

			gw := &gelflogger.GelfWriter{
				Logger: logger,
			}
			if tt.stopServerBeforeTest {
				_ = mockServer.Close()
			}
			defer func() {
				if tt.stopServerBeforeTest {
					mockServer = helper.StartMockServer(t)
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
