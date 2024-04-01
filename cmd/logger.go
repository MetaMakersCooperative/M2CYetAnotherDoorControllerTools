package cmd

import (
	"fmt"
	"io"
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

func NewLogger(w io.Writer, usePrefix bool, useTimeStamp bool) *Logger {
	var logger *Logger
	if useTimeStamp {
		logger = &Logger{log: log.Default()}
		logger.log.SetOutput(w)
	} else {
		logger = &Logger{log: log.New(w, "", log.Ltime)}
	}
	if usePrefix {
		logger.log.SetPrefix("PORTER: ")
	}
	return logger
}

var logger = NewLogger(os.Stdout, true, true)
