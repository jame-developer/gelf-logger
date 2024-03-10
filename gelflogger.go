package gelflogger

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// Logger represents a logging client that connects to a Graylog server using TCP.
//
// The Logger struct has the following fields:
// - conn: The network connection to the Graylog server.
// - connLock: A mutex used to ensure thread-safe access to the conn field.
// - address: The address of the Graylog server to connect to.
// - useTLS: A boolean value indicating whether to use TLS for the connection.
// - tslConfig: The TLS configuration to use if useTLS is true.
// - host: The hostname of the client machine.
//
// The Logger struct provides the following methods:
// - connect: Establishes a connection to the Graylog server.
// - ensureConnection: Ensures that a connection to the Graylog server is established, reconnecting if necessary.
// - Log: Sends a log message to the Graylog server.
type Logger struct {
	conn      net.Conn
	connLock  sync.Mutex
	address   string
	useTLS    bool
	tslConfig *tls.Config
	host      string
}

// NewLogger creates a new Logger.
//
// Example with TLS:
//
//	    // Load our Root CA certificate
//		caCert, err := os.ReadFile("/path/to/ca.crt")
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		caCertPool := x509.NewCertPool()
//		caCertPool.AppendCertsFromPEM(caCert)
//
//		// Create the credentials and return it
//		config := &tls.Config{
//			RootCAs:            caCertPool,
//			InsecureSkipVerify: true,
//			// Other fields can be filled in as necessary
//		}
//
//		writer, err := NewLogger("localhost:1234", true, config)
//
// This creates a new Logger that will use TLS when connecting
// to the specified address.
func NewLogger(address string, useTSL bool, tslConfig *tls.Config) (*Logger, error) {
	host, _ := os.Hostname()
	logger := &Logger{address: address, useTLS: useTSL, tslConfig: tslConfig, host: host}
	err := logger.connect()
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// connect establishes a connection to the specified address using either TCP or TLS, depending on the value of the useTLS flag. If the connection is successful, it is stored in the
func (l *Logger) connect() error {
	dialer := net.Dialer{
		Timeout:   5 * time.Second,  // 5 seconds timeout for the connection attempt
		KeepAlive: 30 * time.Second, // 30 seconds keep-alive interval
	}
	var conn net.Conn
	var err error

	if l.useTLS {
		conn, err = dialer.Dial("tcp", l.address)
		if conn != nil {
			conn = tls.Client(conn, l.tslConfig) // Wrap the connection with TLS
		}
	} else {
		conn, err = dialer.Dial("tcp", l.address)
	}

	if err != nil {
		//log.Printf("Failed to connect to Graylog: %v", err)
		return err
	}

	l.connLock.Lock()
	l.conn = conn
	l.connLock.Unlock()
	return nil
}

// ensureConnection checks if the Logger has an active connection. If not, it tries to establish a new connection.
// If the connection is already established, it sends a zero-byte message to the server to check if the connection is alive.
// If the connection is not alive, it tries to reconnect.
// It is called by the Log method to make sure that there is an active connection before sending log messages.
func (l *Logger) ensureConnection() error {
	l.connLock.Lock()
	defer l.connLock.Unlock()

	if l.conn == nil {
		err := l.connect()
		if err != nil {
			return err
		}
	} else {
		// Simple way to check if the connection is alive
		_, err := l.conn.Write(nil)
		if err != nil {
			err := l.connect()
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

// Log Ensure the connection is alive before logging
func (l *Logger) Log(message string, fields map[string]interface{}) error {
	gelfMessage, err := formatGELFMessage(message, fields, l.host)
	if err != nil {
		return err
	}
	l.connLock.Lock()
	defer l.connLock.Unlock()

	_, err = l.conn.Write(gelfMessage)
	if err != nil {
		err := l.connect()
		if err != nil {
			return err
		} // Attempt to reconnect
		_, err = l.conn.Write(gelfMessage) // Retry the log
		if err != nil {
			return err
		}
	}
	return nil
}

// formatGELFMessage formats a GELF (Graylog Extended Log Format) message with the given message, fields, and host information.
// It converts the level field to the equivalent Graylog level using the ConvertZerologLevelToGraylog function.
// The timestamp is divided by 1000 to convert it from milliseconds to seconds.
// The "level", "time", and "message" fields are deleted from the fields map.
// The GELF message is created by constructing a map with the required fields and adding the remaining fields prefixed with an underscore.
// The GELF message is then marshaled into a byte slice.
// If an error occurs during marshaling, it is logged and returned.
// Finally, the GELF message byte slice is returned along with any error that occurred.
func formatGELFMessage(message string, fields map[string]interface{}, host string) ([]byte, error) {
	graylogLevel, glTimeStamp, fullMessage, err2 := zerologger.ProcessZerologFields(fields)
	if err2 != nil {
		return nil, err2
	}

	gelfMsg := map[string]interface{}{
		"version":       "1.1",
		"host":          host,
		"short_message": message,
		"full_message":  string(fullMessage),
		"timestamp":     glTimeStamp,
		"level":         graylogLevel,
	}

	for k, v := range fields {
		if boolVal, ok := v.(bool); ok {
			gelfMsg["_"+k] = strconv.FormatBool(boolVal)
		} else {
			gelfMsg["_"+k] = v

		}
	}

	msgBytes, err := json.Marshal(gelfMsg)
	if err != nil {
		return nil, err
	}

	return msgBytes, nil
}

// GelfWriter Use the logger to write log messages
type GelfWriter struct {
	Logger *Logger
}

// Write writes the log message to Graylog. It first unmarshals the log message into a map, and then retrieves the "message" key from the map.
// It ensures that the connection to Graylog is alive before writing the log message. If the connection is not alive, it calls the ensureConnection method to establish a new connection
func (gw *GelfWriter) Write(p []byte) (n int, err error) {
	var logMsg map[string]interface{}
	if err := json.Unmarshal(p, &logMsg); err != nil {
		return 0, err
	}

	message, ok := logMsg["message"].(string)
	if !ok {
		return 0, fmt.Errorf("log message is not a string")
	}

	// Ensure the connection is alive before logging
	err = gw.Logger.ensureConnection()
	if err != nil {
		return 0, err
	}

	err = gw.Logger.Log(message, logMsg)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
