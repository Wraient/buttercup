package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"net/url"
	"golang.org/x/net/html"
	"os/exec"
	"time"
)

func sanitizeQuery(query string) string {
    // Trim spaces and convert to lowercase
    query = strings.TrimSpace(strings.ToLower(query))
    
    // Replace multiple spaces with single space
    query = strings.Join(strings.Fields(query), " ")
    
    // URL encode the query
    return url.QueryEscape(query)
}

func SearchJackett(query string) (*JackettResponse, error) {
	config := GetGlobalConfig()

	// Sanitize the search query
    sanitizedQuery := sanitizeQuery(query)
    
	// Build the Jackett API URL using config values
	jackettURL := fmt.Sprintf("http://%s:%s/api/v2.0/indexers/all/results",
		config.JackettUrl,
		config.JackettPort)

	// Create the request
	req, err := http.NewRequest("GET", jackettURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("apikey", config.JackettApiKey)
	q.Add("Query", sanitizedQuery)
	req.URL.RawQuery = q.Encode()

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response from Jackett server: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse and handle response
	var response JackettResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	Debug("Search results: %s", string(body))
	return &response, nil
}

func GetJackettApiKey() (string, error) {
    // Jackett config is typically stored in ~/.config/Jackett/ServerConfig.json
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("failed to get home directory: %w", err)
    }

    jackettConfig := filepath.Join(homeDir, ".config", "Jackett", "ServerConfig.json")
    data, err := os.ReadFile(jackettConfig)
    if err != nil {
        return "", fmt.Errorf("failed to read Jackett config: %w", err)
    }

    // Parse JSON config
    var config struct {
        APIKey string `json:"APIKey"`
    }

    if err := json.Unmarshal(data, &config); err != nil {
        return "", fmt.Errorf("failed to parse Jackett config: %w", err)
    }

    return config.APIKey, nil
}

// FetchMagnetURI fetches the magnet URI from a 1337x torrent page
func FetchMagnetURI(torrentURL string) (string, error) {
    // Make an HTTP GET request to the torrent URL
    resp, err := http.Get(torrentURL)
    if err != nil {
        return "", fmt.Errorf("failed to fetch torrent page: %w", err)
    }
    defer resp.Body.Close()

    // Parse the HTML document
    tokenizer := html.NewTokenizer(resp.Body)
    for {
        tokenType := tokenizer.Next()
        switch tokenType {
        case html.ErrorToken:
            return "", fmt.Errorf("magnet link not found")
        case html.StartTagToken, html.SelfClosingTagToken:
            token := tokenizer.Token()
            if token.Data == "a" {
                for _, attr := range token.Attr {
                    if attr.Key == "href" && strings.HasPrefix(attr.Val, "magnet:?xt=") {
                        return attr.Val, nil
                    }
                }
            }
        }
    }
}

func CheckJackettAvailability(config *ProgramConfig) error {
    // Try to connect to Jackett
    url := fmt.Sprintf("http://%s:%s/api/v2.0/indexers/all/results/torznab/api?apikey=%s",
        config.JackettUrl, config.JackettPort, config.JackettApiKey)
    
    _, err := http.Get(url)
    if err != nil {
        return err
    }
    return nil
}

func InstallJackett() error {
    fmt.Println("Installing Jackett...")
    cmd := exec.Command("bash", "-c", `
        curl -L -o /tmp/jackett.tar.gz https://github.com/Jackett/Jackett/releases/latest/download/Jackett.Binaries.LinuxAMDx64.tar.gz
        cd /tmp && tar -xf jackett.tar.gz
        sudo mv /tmp/Jackett /opt/
        sudo ln -s /opt/Jackett/jackett /usr/local/bin/jackett
    `)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func StartJackett() error {
    // Install default indexers before starting Jackett
    if err := InstallDefaultIndexers(); err != nil {
        return fmt.Errorf("failed to install default indexers: %w", err)
    }

    // Open /dev/null for redirecting output
    devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
    if err != nil {
        return fmt.Errorf("failed to open /dev/null: %w", err)
    }
    defer devNull.Close()

    // Start Jackett in background with output redirected to /dev/null
    cmd := exec.Command("jackett")
    cmd.Stdout = devNull
    cmd.Stderr = devNull
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start Jackett: %w", err)
    }

    // Wait for Jackett to be ready
    fmt.Println("Waiting for Jackett to start...")
    time.Sleep(10 * time.Second)
    
    return nil
}

func InstallDefaultIndexers() error {
    // Create Jackett indexers directory if it doesn't exist
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return fmt.Errorf("failed to get home directory: %w", err)
    }

    indexersDir := filepath.Join(homeDir, ".config", "Jackett", "Indexers")
    if err := os.MkdirAll(indexersDir, 0755); err != nil {
        return fmt.Errorf("failed to create indexers directory: %w", err)
    }

    // Define indexers to download
    indexers := []string{
        "1337x.json",
        "nyaasi.json",
    }

    baseURL := "https://raw.githubusercontent.com/Wraient/buttercup/refs/heads/main/jackett"

    // Download each indexer
    for _, indexer := range indexers {
        // Download the file
        resp, err := http.Get(fmt.Sprintf("%s/%s", baseURL, indexer))
        if err != nil {
            return fmt.Errorf("failed to download %s: %w", indexer, err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return fmt.Errorf("failed to download %s: status code %d", indexer, resp.StatusCode)
        }

        // Create the file
        outPath := filepath.Join(indexersDir, indexer)
        out, err := os.Create(outPath)
        if err != nil {
            return fmt.Errorf("failed to create file %s: %w", indexer, err)
        }
        defer out.Close()

        // Copy the content
        if _, err := io.Copy(out, resp.Body); err != nil {
            return fmt.Errorf("failed to write file %s: %w", indexer, err)
        }
    }

    return nil
}
