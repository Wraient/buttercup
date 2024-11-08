package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func IsVideoFile(filename string) bool {
    videoExtensions := []string{
        ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", 
        ".webm", ".m4v", ".mpg", ".mpeg", ".3gp",
    }
    
    ext := strings.ToLower(filepath.Ext(filename))
    for _, videoExt := range videoExtensions {
        if ext == videoExt {
            return true
        }
    }
    return false
}


func UpdateButtercup(repo, fileName string) error {
    // Get the path of the currently running executable
    executablePath, err := os.Executable()
    if err != nil {
        return fmt.Errorf("unable to find current executable: %v", err)
    }

    // Adjust file name for Windows
    if runtime.GOOS == "windows" {
        fileName += ".exe"
    }

    // GitHub release URL for buttercup
    url := fmt.Sprintf("https://github.com/%s/releases/latest/download/%s", repo, fileName)

    // Temporary path for the downloaded buttercup executable
    tmpPath := executablePath + ".tmp"

    // Download the buttercup executable
    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("failed to download file: %v", err)
    }
    defer resp.Body.Close()

    // Check if the download was successful
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed to download file: received status code %d", resp.StatusCode)
    }

    // Create a new temporary file
    out, err := os.Create(tmpPath)
    if err != nil {
        return fmt.Errorf("failed to create temporary file: %v", err)
    }
    defer out.Close()

    // Write the downloaded content to the temporary file
    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return fmt.Errorf("failed to save downloaded file: %v", err)
    }

    // Close and rename the temporary file to replace the current executable
    out.Close()

    // Replace the current executable with the downloaded buttercup
    if err := os.Rename(tmpPath, executablePath); err != nil {
        return fmt.Errorf("failed to replace the current executable: %v", err)
    }
    Exit(fmt.Sprintf("Downloaded buttercup executable to %v", executablePath), nil)

	if runtime.GOOS != "windows" {
		// Ensure the new file has executable permissions
		if err := os.Chmod(executablePath, 0755); err != nil {
			return fmt.Errorf("failed to set permissions on the new file: %v", err)
		}
	}
	
    return nil
}

func CheckAndDownloadFiles(storagePath string, filesToCheck []string) error {
	// Create storage directory if it doesn't exist
	storagePath = os.ExpandEnv(storagePath)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %v", err)
	}

	// Base URL for downloading config files
	baseURL := "https://raw.githubusercontent.com/Wraient/buttercup/main/rofi/"

	// Check each file
	for _, fileName := range filesToCheck {
		filePath := filepath.Join(os.ExpandEnv(storagePath), fileName)

		// Skip if file already exists
		if _, err := os.Stat(filePath); err == nil {
			continue
		}

		// Download file if it doesn't exist
		resp, err := http.Get(baseURL + fileName)
		if err != nil {
			return fmt.Errorf("failed to download %s: %v", fileName, err)
		}
		defer resp.Body.Close()

		// Create the file
		out, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", fileName, err)
		}
		defer out.Close()

		// Write the content
		if _, err := io.Copy(out, resp.Body); err != nil {
			return fmt.Errorf("failed to write file %s: %v", fileName, err)
		}
	}

	return nil
}