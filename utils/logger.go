package utils

import (
	"log"
	"os"
	"strings"
)

type LoggerObject struct {
	bucketName     string
	isBucketLogger bool
	toTerminal     bool
}

const (
	debug = 0
	info  = 1
	warn  = 2
	err   = 3
)

var verbosity = info
var toTerminal = true
var MainLogger LoggerObject

func ConfigureLogging(config *LoggingConfig) {
	if config.File != "" {
		file, err := os.OpenFile(config.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		log.SetOutput(file)
		toTerminal = false
	}
	if config.Level != "" {
		switch strings.ToUpper(config.Level) {
		case "DEBUG":
			verbosity = debug
		case "INFO":
			verbosity = info
		case "WARN":
			verbosity = warn
		case "ERR":
			verbosity = err
		default:
			panic("Invalid log level")
		}
	}
	MainLogger = NewLogger()
}

func NewLogger() LoggerObject {
	return LoggerObject{
		bucketName:     "",
		isBucketLogger: false,
		toTerminal:     toTerminal,
	}
}

func NewBucketLogger(bucketName string) LoggerObject {
	return LoggerObject{
		bucketName:     bucketName,
		isBucketLogger: true,
		toTerminal:     toTerminal,
	}
}

func (logger *LoggerObject) println(v ...interface{}) {
	log.Println(v...)
}

func (logger *LoggerObject) getColorfulPrefix(prefix string, verbosity int) string {
	if logger.toTerminal {
		switch verbosity {
		case info:
			return "\033[1;32m" + prefix + "\033[0m"
		case warn:
			return "\033[1;33m" + prefix + "\033[0m"
		case err:
			return "\033[1;31m" + prefix + "\033[0m"
		default:
			return prefix
		}
	}
	return prefix
}

func (logger *LoggerObject) Debug(v ...interface{}) {
	if verbosity <= debug {
		if logger.isBucketLogger {
			v = append([]interface{}{"[" + logger.bucketName + "]"}, v...)
		}
		logger.println(append([]interface{}{logger.getColorfulPrefix("[DEBUG]", debug)}, v...)...)
	}
}

func (logger *LoggerObject) Info(v ...interface{}) {
	if verbosity <= info {
		if logger.isBucketLogger {
			v = append([]interface{}{"[" + logger.bucketName + "]"}, v...)
		}
		logger.println(append([]interface{}{logger.getColorfulPrefix("[INFO]", info)}, v...)...)
	}
}

func (logger *LoggerObject) Warn(v ...interface{}) {
	if verbosity <= warn {
		if logger.isBucketLogger {
			v = append([]interface{}{"[" + logger.bucketName + "]"}, v...)
		}
		logger.println(append([]interface{}{logger.getColorfulPrefix("[WARN]", warn)}, v...)...)
	}
}

func (logger *LoggerObject) Error(v ...interface{}) {
	if verbosity <= err {
		if logger.isBucketLogger {
			v = append([]interface{}{"[" + logger.bucketName + "]"}, v...)
		}
		logger.println(append([]interface{}{logger.getColorfulPrefix("[ERROR]", err)}, v...)...)
	}
}
