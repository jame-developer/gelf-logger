package zerologger

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	gelflogger "github.com/jame-developer/gelf-logger"
	"github.com/rs/zerolog"
	"io"
	"log"
	"time"
)

// LogLevelMap maps zerolog levels to Graylog (Syslog) levels.
var LogLevelMap = map[zerolog.Level]int{
	zerolog.DebugLevel: 7, // Debug
	zerolog.InfoLevel:  6, // Info
	zerolog.WarnLevel:  4, // Warning
	zerolog.ErrorLevel: 3, // Error
	zerolog.FatalLevel: 2, // Critical
	zerolog.PanicLevel: 1, // Alert
	// Note: Zerolog does not have a direct equivalent for Notice (5) and Emergency (0) Syslog levels
}

// NewZeroLogger initializes and returns a zerolog logger with a Graylog (Syslog) writer.
// It takes the following arguments:
// - address: the address of the Graylog server
// - useTLS: a boolean indicating whether to use TLS for the connection
// - tslConfig: a *tls.Config object to configure the TLS connection (optional)
// - otherZeroLogWriter: zero or more additional io.Writer objects to write logs to (optional)
// The logger is created in the following steps:
// 1. The gelflogger.NewLogger function is called with the given address, useTLS, tslConfig, and ProcessZerologFields to create a gelflogger.Logger object.
// 2. If the gelflogger.Logger initialization is successful, a gelflogger.GelfWriter is created with the graylogLogger.
// 3. If otherZeroLogWriter is not nil, a new slice is created with only the gelfWriter. Otherwise, otherZeroLogWriter remains unchanged.
// 4. The zerolog.TimeFieldFormat is set to a GELF compatible timestamp format.
// 5. A zerolog.MultiLevelWriter is created with otherZeroLogWriter as the variadic argument.
// 6. A zerolog.Logger is created with the multiLevelWriter, Timestamp, and Logger options.
//
// Example usage:
//
//	logger, err := NewZeroLogger("graylog.example.com:12201", true, nil, os.Stdout)
//	if err != nil {
//	  // handle error
//	}
//	logger.Info().Msg("Hello, World!")
func NewZeroLogger(address string, useTSL bool, tslConfig *tls.Config, otherZeroLogWriter ...io.Writer) (zerolog.Logger, error) {
	graylogLogger, gelfLoggerInitErr := gelflogger.NewLogger(address, useTSL, tslConfig, ProcessZerologFields)
	if gelfLoggerInitErr == nil {
		gelfWriter := gelflogger.GelfWriter{
			Logger: graylogLogger,
		}

		if otherZeroLogWriter != nil {
			otherZeroLogWriter = []io.Writer{&gelfWriter}
		} else {
			otherZeroLogWriter = append(otherZeroLogWriter)
		}

		// Set the time field format to a GELF compatible timestamp format see also https://go2docs.graylog.org/5-0/getting_in_log_data/gelf.html?tocpath=Getting%20in%20Logs%7CLog%20Sources%7CGELF%7C_____0#GELFPayloadSpecification
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

		// Create the Multilevel writer and create the zero logger.
		multiLevelWriter := zerolog.MultiLevelWriter(otherZeroLogWriter...)
		return zerolog.New(multiLevelWriter).With().Timestamp().Logger(), nil
	}

	return zerolog.New(nil), gelfLoggerInitErr
}
func ProcessZerologFields(fields map[string]interface{}) (int, float64, []byte, error) {
	if _, ok := fields["time"]; !ok {
		fields["time"] = float64(time.Now().UnixMilli())
	}
	if _, ok := fields["time"].(float64); !ok {
		return 0, 0, nil, fmt.Errorf("field `time` is not of type loat64; invalid log message format")
	}
	if _, ok := fields["level"]; !ok {
		fields["level"] = "info"
	}
	graylogLevel := ConvertZerologLevelToGraylog(fields["level"].(string))
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

// ConvertZerologLevelToGraylog converts a zerolog level to the equivalent Graylog (Syslog) level.
func ConvertZerologLevelToGraylog(level string) int {
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return 6
	}
	if syslogLevel, exists := LogLevelMap[parsedLevel]; exists {
		return syslogLevel
	}
	return 6
}
