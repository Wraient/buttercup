// internal/logger.go
package internal

import (
    "io"
    "log"
    "os"
    "fmt"
    "runtime"
)

var (
    InfoLogger  *log.Logger
    DebugLogger *log.Logger
    IsDebug     bool
)

func InitLogger(debug bool) {
    IsDebug = debug
    InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
    if debug {
        DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
    } else {
        DebugLogger = log.New(io.Discard, "", 0)
    }
}

func Debug(format string, v ...interface{}) {
    _, file, line, _ := runtime.Caller(1)
    prefix := fmt.Sprintf("DEBUG: [%s:%d] ", file, line)
    DebugLogger.SetPrefix(prefix)
    DebugLogger.Printf(format, v...)
}

func Info(format string, v ...interface{}) {
    InfoLogger.Printf(format, v...)
}