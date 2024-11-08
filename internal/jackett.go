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
	q.Add("apikey", config.JackettApiKey) // You'll need to add ApiKey to your config struct
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
