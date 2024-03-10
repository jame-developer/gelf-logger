package zerologger_test

import (
	"crypto/tls"
	"encoding/json"
	"github.com/jame-developer/gelf-logger/pkg/helper"
	"github.com/jame-developer/gelf-logger/pkg/zerologger"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
	"time"
)

func TestNewZeroLogger(t *testing.T) {
	// Set up the mock server here
	mockServer := helper.StartMockServer(t)
	mockTLSServer := helper.StartMockTLSServer(t)
	defer t.Cleanup(func() {
		_ = mockServer.Close()
		_ = mockTLSServer.Close()
	})

	testCases := []struct {
		name               string
		address            string
		useTLS             bool
		OtherZeroLogWriter []io.Writer
		tlsConfig          *tls.Config
		wantErr            bool
	}{
		{
			name:               "Valid TCP Address Without TLS",
			address:            mockServer.Addr().String(),
			OtherZeroLogWriter: []io.Writer{os.Stderr},
			useTLS:             false,
			wantErr:            false,
		},
		{
			name:               "Invalid TCP Address Without TLS",
			address:            "invalid:address",
			OtherZeroLogWriter: []io.Writer{os.Stderr},
			useTLS:             false,
			wantErr:            true,
		},
		{
			name:               "Valid TCP Address With TLS",
			address:            mockTLSServer.Addr().String(),
			OtherZeroLogWriter: []io.Writer{os.Stderr},
			useTLS:             true,
			tlsConfig:          &tls.Config{InsecureSkipVerify: true},
			wantErr:            false,
		},
		{
			name:               "Invalid TCP Address With TLS",
			address:            "invalid:address",
			OtherZeroLogWriter: []io.Writer{os.Stderr},
			useTLS:             true,
			tlsConfig:          &tls.Config{},
			wantErr:            true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test NewZapLogger
			_, err := zerologger.NewZeroLogger(tc.address, tc.useTLS, &tls.Config{}, tc.OtherZeroLogWriter...)
			if !tc.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestProcessZerologFields(t *testing.T) {
	tt := []struct {
		name       string
		input      map[string]interface{}
		wantOutput []byte
		wantErr    bool
	}{
		{
			name: "Correct_Inputs",
			input: map[string]interface{}{
				"level":   "error",
				"time":    float64(time.Now().UnixMilli()),
				"message": "This is a test log message",
			},
			wantErr: false,
		},
		{
			name: "Incorrect_Time",
			input: map[string]interface{}{
				"level":   "error",
				"time":    "incorrect value",
				"message": "This is a test log message",
			},
			wantErr: true,
		},
		{
			name:    "Empty_Fields",
			input:   map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, _, gotOutput, err := zerologger.ProcessZerologFields(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				message := make(map[string]interface{})
				err := json.Unmarshal(gotOutput, &message)
				assert.NoError(t, err)
			}
		})
	}
}

func TestConvertZerologLevelToGraylog(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		expectedLevel int
	}{
		{
			name:          "TestDebug",
			level:         "debug",
			expectedLevel: 7,
		},
		{
			name:          "TestInfo",
			level:         "info",
			expectedLevel: 6,
		},
		{
			name:          "TestWarn",
			level:         "warn",
			expectedLevel: 4,
		},
		{
			name:          "TestNonExistentLevel",
			level:         "nonExistentLevel",
			expectedLevel: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualLevel := zerologger.ConvertZerologLevelToGraylog(tt.level)
			if actualLevel != tt.expectedLevel {
				t.Errorf("expected level %v but got %v", tt.expectedLevel, actualLevel)
			}
		})
	}
}
