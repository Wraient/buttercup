package internal

import (
	"strings"
	"path/filepath"
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