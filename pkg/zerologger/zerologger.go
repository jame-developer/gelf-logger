package zerologger

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog"
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
