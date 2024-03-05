# GO GELF LOGGER

[![Go Reference](https://pkg.go.dev/badge/github.com/jame-developer/gelf-logger.svg)](https://pkg.go.dev/<github.com/yourusername/yourproject>) [![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md) [![CodeQL](https://github.com/jame-developer/gelf-logger/actions/workflows/codeql.yml/badge.svg)](https://github.com/jame-developer/gelf-logger/actions/workflows/codeql.yml) [![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=jame-developer_gelf-logger&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=jame-developer_gelf-logger) 

[![](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)

This package provides an `io.Writer` that can be used with logging libraries such as [zerolog](https://github.com/rs/zerolog) or [zap](https://github.com/uber-go/zap) for sending your logs to a [GELF-compatible](https://go2docs.graylog.org/5-0/getting_in_log_data/gelf.html?tocpath=Getting%20in%20Logs%7CLog%20Sources%7CGELF%7C_____0#GELFPayloadSpecification) server like Graylog.

This project was inspired by https://github.com/Graylog2/go-gelf.

I was missing the capability of also sending the additional fields, which can be specified in [zerolog](https://github.com/rs/zerolog) or [zap](https://github.com/uber-go/zap), as actual additional field with GELF.

This package takes all fields except of "level", "message"/"msg" and "time" and adds them as additional fields in the GELF request.

So you don't need an additional custom extractor or pipline for those fields. An example could be a "request-id" field.

## Requirements

- Go1.22.0
- The timestamp field must be a UNIX timestamp with milliseconds.
- The log message must be a JSON string.

## Installation

TODO: To install the package, use the following command:


## Usage

Below is a basic usage example:

```go
package main

import (
	"crypto/tls"
	"crypto/x509"
	gelflogger "github.com/jame-developer/gelf-logger"
	"github.com/rs/zerolog"
	"io"
	"log"
	"net"
	"os"
)

// Main function with zerolog integration using the MultilevelWriter func.
func main() {
	// Create an array of log writers for the MultilevelWriter and add `os.Stderr` as default.
	logWriters := []io.Writer{os.Stderr}

	// Load our Root CA certificate chain
	caCert, err := os.ReadFile("path to your rootCA certificate chain")
	if err != nil {
		log.Fatal(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
		// Filled other fields as necessary
	}

	graylogLogger, gelfLoggerInitErr := gelflogger.NewLogger("<YOUR_GRAYLOG_SERVER>:12201", true, config)
	
	// Only append the gelf-writer to the logWrites if initialization was successful.
	if gelfLoggerInitErr == nil {

		gelfWriter := gelflogger.GelfWriter{
			Logger: graylogLogger,
		}
		logWriters = append(logWriters, &gelfWriter)
	}

	// Set the time field format to a GELF compatible timestamp format see also https://go2docs.graylog.org/5-0/getting_in_log_data/gelf.html?tocpath=Getting%20in%20Logs%7CLog%20Sources%7CGELF%7C_____0#GELFPayloadSpecification
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	// Create the Multilevel writer and create the zero logger.
	multiLevelWriter := zerolog.MultiLevelWriter(logWriters...)
	logger := zerolog.New(multiLevelWriter).With().Timestamp().Logger()

	// Use the created zero logger to log an error message, in case the initialization of the GELF-logger failed
	if gelfLoggerInitErr != nil {
		logger.Error().Err(gelfLoggerInitErr).Msg("Failed to initialize GELF logger.")
	}

	// Log a test message, you should see this message in your console and on the server you specified above.
	logger.Info().Str("custom_field", "custom_value").IPAddr("remoteIP", net.ParseIP("192.168.0.1")).Msg("This is a test log message with zerolog2")
}


```

## Testing

-  create test certificate files. You can use OpenSSL with the following commands in your `test_data` folder under project root:

```shell
openssl req -newkey rsa:2048 -nodes -keyout testkey.pem -x509 -days 1 -out testcert.pem
```

To run tests in the terminal, go to the directory where the project is located and type: `go test ./...`

## License

This project is licensed under the terms of [MIT license](LICENSE).

## Contact

If you have specific questions about this project, you can reach out. I'll get back to you as soon as I can.
