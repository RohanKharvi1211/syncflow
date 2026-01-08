package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

var DebugEnabled = os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1"

// Debug prints debug message if DEBUG env var is set
func Debug(format string, args ...interface{}) {
	if DebugEnabled {
		_, file, line, _ := runtime.Caller(1)
		fileParts := strings.Split(file, "/")
		fileName := fileParts[len(fileParts)-1]
		fmt.Printf("[DEBUG] %s:%d - %s\n", fileName, line, fmt.Sprintf(format, args...))
	}
}

// Debugf is an alias for Debug
func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

// DebugVar prints variable name and value
func DebugVar(name string, value interface{}) {
	if DebugEnabled {
		_, file, line, _ := runtime.Caller(1)
		fileParts := strings.Split(file, "/")
		fileName := fileParts[len(fileParts)-1]
		fmt.Printf("[DEBUG] %s:%d - %s = %+v\n", fileName, line, name, value)
	}
}

// DebugRequest logs HTTP request details
func DebugRequest(method, path string, params map[string]interface{}) {
	if DebugEnabled {
		_, file, line, _ := runtime.Caller(1)
		fileParts := strings.Split(file, "/")
		fileName := fileParts[len(fileParts)-1]
		fmt.Printf("[DEBUG] %s:%d - %s %s", fileName, line, method, path)
		if len(params) > 0 {
			fmt.Printf(" | Params: %+v", params)
		}
		fmt.Println()
	}
}

// DebugDB logs database query details
func DebugDB(query string, args ...interface{}) {
	if DebugEnabled {
		_, file, line, _ := runtime.Caller(1)
		fileParts := strings.Split(file, "/")
		fileName := fileParts[len(fileParts)-1]
		fmt.Printf("[DEBUG DB] %s:%d - Query: %s", fileName, line, query)
		if len(args) > 0 {
			fmt.Printf(" | Args: %+v", args)
		}
		fmt.Println()
	}
}


