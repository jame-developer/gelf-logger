package zaplogger_test

import (
	"crypto/tls"
	"encoding/json"
	"github.com/jame-developer/gelf-logger/pkg/helper"
	"github.com/jame-developer/gelf-logger/pkg/zaplogger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"testing"
	"time"
)

func TestNewZapLogger(t *testing.T) {
	// Set up the mock server here
	mockServer := helper.StartMockServer(t)
	mockTLSServer := helper.StartMockTLSServer(t)
	defer t.Cleanup(func() {
		_ = mockServer.Close()
		_ = mockTLSServer.Close()
	})
	validOtherZapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		os.Stdout,
		zap.InfoLevel,
	)

	testCases := []struct {
		name       string
		address    string
		useTLS     bool
		otherCores []zapcore.Core
		tlsConfig  *tls.Config
		wantErr    bool
	}{
		{
			name:       "Valid TCP Address Without TLS",
			address:    mockServer.Addr().String(),
			otherCores: []zapcore.Core{validOtherZapCore},
			useTLS:     false,
			wantErr:    false,
		},
		{
			name:       "Invalid TCP Address Without TLS",
			address:    "invalid:address",
			otherCores: []zapcore.Core{validOtherZapCore},
			useTLS:     false,
			wantErr:    true,
		},
		{
			name:       "Valid TCP Address With TLS",
			address:    mockTLSServer.Addr().String(),
			otherCores: []zapcore.Core{validOtherZapCore},
			useTLS:     true,
			tlsConfig:  &tls.Config{InsecureSkipVerify: true},
			wantErr:    false,
		},
		{
			name:       "Invalid TCP Address With TLS",
			address:    "invalid:address",
			otherCores: []zapcore.Core{validOtherZapCore},
			useTLS:     true,
			tlsConfig:  &tls.Config{},
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test NewZapLogger
			_, err := zaplogger.NewZapLogger(tc.address, tc.useTLS, &tls.Config{}, tc.otherCores...)
			if !tc.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestProcessZapLoggerFields(t *testing.T) {
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
			_, _, gotOutput, err := zaplogger.ProcessZapLoggerFields(tc.input)
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

func TestConvertZapLogLevelToGraylog(t *testing.T) {
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
			actualLevel := zaplogger.ConvertZapLogLevelToGraylog(tt.level)
			assert.Equal(t, tt.expectedLevel, actualLevel)
		})
	}
}
