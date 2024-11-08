package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

func Exit(msg string,err error) {
	CleanupPeerflix()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if msg != "" {
		fmt.Println(msg)
	}
	os.Exit(0)
}

func PrintUsage() {
	fmt.Println("Usage: buttercup <command>")
}

// FormatSize converts bytes to human readable string with appropriate unit
func FormatSize(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    
    return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func Output(data interface{}) {
	fmt.Println(data)
}

func Log(data interface{}, logFile string) error {
	// Open or create the log file
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close() // Ensure the file is closed when done

	// Attempt to marshal the data into JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Get the caller information
	_, filename, lineNumber, ok := runtime.Caller(1) // Caller 1 gives the caller of LogData
	if !ok {
		return fmt.Errorf("unable to get caller information")
	}

	// Log the current time and the JSON representation along with caller info
	currentTime := time.Now().Format("2006/01/02 15:04:05")
	logMessage := fmt.Sprintf("[LOG] %s %s:%d: %s\n", currentTime, filename, lineNumber, jsonData)
	_, err = fmt.Fprint(file, logMessage) // Write to the file
	if err != nil {
		return err
	}

	return nil
}