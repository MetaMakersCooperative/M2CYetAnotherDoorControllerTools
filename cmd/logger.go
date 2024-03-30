package cmd

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	log *log.Logger
}

func (logger *Logger) Info(format string, args ...any) {
	logger.log.Output(2, fmt.Sprintf("INFO: "+format, args...))
}

func (logger *Logger) Warn(format string, args ...any) {
	logger.log.Output(2, fmt.Sprintf("WARN: "+format, args...))
}

func (logger *Logger) Error(format string, args ...any) {
	logger.log.Output(2, fmt.Sprintf("error: "+format, args...))
}

func New() *Logger {
	logger := &Logger{log: log.Default()}
	logger.log.SetPrefix("PORTER: ")
	logger.log.SetOutput(os.Stdout)
	return logger
}

var logger = New()
