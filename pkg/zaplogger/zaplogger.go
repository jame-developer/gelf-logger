package zaplogger

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	gelflogger "github.com/jame-developer/gelf-logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"log"
	"time"
)

// LogLevelMap maps zap logger levels to Graylog (Syslog) levels.
var LogLevelMap = map[zapcore.Level]int{
	zapcore.DebugLevel:  7, // Debug
	zapcore.InfoLevel:   6, // Info
	zapcore.WarnLevel:   4, // Warning
	zapcore.ErrorLevel:  3, // Error
	zapcore.DPanicLevel: 2, // Critical
	zapcore.PanicLevel:  1, // Alert
	zapcore.FatalLevel:  0, // Emergency
}

// NewZapLogger creates a new Zap logger with the specified Graylog address and TLS configuration.
// It takes the following parameters:
//   - address: the address of the Graylog server
//   - useTLS: a boolean indicating whether to use TLS for the connection
//   - tslConfig: the TLS configuration to use (can be nil if useTLS is false)
//   - otherZapCores: optional additional Zap cores to include in the logger's core
//
// It first initializes a new GelfLogger using the provided address, useTLS, tslConfig, and ProcessZapLoggerFields function.
// If the GelfLogger initialization is successful, it creates a GelfWriter using the GelfLogger.
// It then creates a Zap core by adding the GelfWriter to a MultiWriter and creating a Zap core with JSON encoder and InfoLevel.
// If otherZapCores are provided, it appends the Gelf core to the otherZapCores and creates a Tee core.
// Otherwise, it creates the Tee core with only the Gelf core.
// Finally, it creates and returns a new Zap logger with the Tee core.
// If the GelfLogger initialization fails, it returns nil and the error from the GelfLogger initialization.
func NewZapLogger(address string, useTSL bool, tslConfig *tls.Config, otherZapCores ...zapcore.Core) (*zap.Logger, error) {
	graylogLogger, gelfLoggerInitErr := gelflogger.NewLogger(address, useTSL, tslConfig, ProcessZapLoggerFields)
	if gelfLoggerInitErr == nil {
		gelfWriter := gelflogger.GelfWriter{
			Logger: graylogLogger,
		}
		logWriter := zapcore.AddSync(io.MultiWriter(&gelfWriter))
		gelfCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			logWriter,
			zap.InfoLevel,
		)
		if otherZapCores != nil {
			otherZapCores = []zapcore.Core{gelfCore}
		} else {
			otherZapCores = append(otherZapCores)
		}

		core := zapcore.NewTee(otherZapCores...)

		return zap.New(core), nil
	}

	return nil, gelfLoggerInitErr
}

func ProcessZapLoggerFields(fields map[string]interface{}) (int, float64, []byte, error) {
	if _, ok := fields["time"]; !ok {
		fields["time"] = float64(time.Now().UnixMilli())
	}
	if _, ok := fields["time"].(float64); !ok {
		return 0, 0, nil, fmt.Errorf("field `time` is not of type loat64; invalid log message format")
	}
	if _, ok := fields["level"]; !ok {
		fields["level"] = "info"
	}
	graylogLevel := ConvertZapLogLevelToGraylog(fields["level"].(string))
	glTimeStamp := fields["time"].(float64) / 1000
	fields["level"] = graylogLevel
	fullMessage, err := json.Marshal(&fields)
	if err != nil {
		log.Println(err)
	}
	delete(fields, "level")
	delete(fields, "time")
	delete(fields, "message")

	return graylogLevel, glTimeStamp, fullMessage, nil
}

// ConvertZapLogLevelToGraylog converts a Zap log level to a Graylog log level.
// It takes a string `level` as input and returns an integer corresponding to the Graylog log level.
// It first parses the input `level` using `zapcore.ParseLevel`.
// If the parsing is successful, it checks if the parsed level exists in the `LogLevelMap` map.
// If it exists, it returns the corresponding Graylog log level.
// If it does not exist, it returns the default Graylog log level 6.
// If the parsing fails, it also returns the default Graylog log level 6.
func ConvertZapLogLevelToGraylog(level string) int {
	parsedLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return 6
	}
	if syslogLevel, exists := LogLevelMap[parsedLevel]; exists {
		return syslogLevel
	}
	return 6
}
