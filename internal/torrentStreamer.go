package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"net/url"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/dustin/go-humanize"
)

var currentWebtorrentProcess *os.Process

type TorrentFile struct {
	Path     string
	Size     int64
	Index    int
	Priority int
}

type PrioritizedPiece struct {
	Index    int
	Priority int
}

type StreamManager struct {
	reader      *torrent.Reader
	turntor     *torrent.Torrent
	pieceLength int64
	currentPos  int64
}

func NewStreamManager(reader *torrent.Reader, t *torrent.Torrent) *StreamManager {
	return &StreamManager{
		reader:      reader,
		turntor:     t,
		pieceLength: t.Info().PieceLength,
		currentPos:  0,
	}
}

func (sm *StreamManager) prioritizeFromPosition(pos int64) {
	startPiece := pos / sm.pieceLength

	// Reset all piece priorities
	for i := 0; i < sm.turntor.NumPieces(); i++ {
		sm.turntor.Piece(i).SetPriority(torrent.PiecePriorityNone)
	}

	// Only prioritize next 5 seconds worth of pieces
	endPiece := startPiece + 5 // Approximate 5 pieces for 5 seconds
	fmt.Printf("\nPrioritizing pieces %d to %d\n", startPiece, endPiece)

	for i := startPiece; i < endPiece && i < int64(sm.turntor.NumPieces()); i++ {
		piece := sm.turntor.Piece(int(i))
		piece.SetPriority(torrent.PiecePriorityNow)

		// Print piece status
		fmt.Printf("Piece %d: Complete=%v, Priority=Now\n",
			i,
			piece.State().Complete)
	}
}

// Add this function to get MPV position
func getMPVPosition(socketPath string) (float64, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	// Send get_property command
	cmd := struct {
		Command   []string `json:"command"`
		RequestID int      `json:"request_id"`
	}{
		Command:   []string{"get_property", "time-pos"},
		RequestID: 1,
	}

	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		return 0, err
	}

	// Read response
	var response struct {
		Data      float64 `json:"data"`
		Error     string  `json:"error"`
		RequestID int     `json:"request_id"`
	}

	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return 0, err
	}

	if response.Error != "" {
		return 0, fmt.Errorf("mpv error: %s", response.Error)
	}

	return response.Data, nil
}


func StartWebtorrentServer(magnetURI string, selectedIndex int) error {
	// First cleanup any existing webtorrent processes
	CleanupWebtorrent()

	// Get storage path from config
	config := GetGlobalConfig()

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Create the webtorrent command
	webtorrentCmd := exec.Command("webtorrent",
		magnetURI,
		"--select", fmt.Sprintf("%d", selectedIndex),
		"--keep-seeding",
		"--no-quit",
		"--quiet",
		"--port", "8000",
		"--out", config.StoragePath,
	)

	// Setup pipes for stdout/stderr
	webtorrentCmd.Stdout = os.Stdout
	webtorrentCmd.Stderr = os.Stderr

	// Start the process
	if err := webtorrentCmd.Start(); err != nil {
		return fmt.Errorf("failed to start webtorrent: %w", err)
	}

	// Store the process for cleanup
	currentWebtorrentProcess = webtorrentCmd.Process

	// Don't wait for it to complete
	go func() {
		webtorrentCmd.Wait()
	}()

	Debug("Started webtorrent process (PID: %d) with storage path: %s", 
		currentWebtorrentProcess.Pid, 
		config.StoragePath)
	return nil
}


func GetTorrentFiles(magnetURI string) ([]TorrentFileInfo, error) {
	// Create temporary directory for downloads
	tmpDir, err := os.MkdirTemp("", "torrent-stream-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Configure torrent client
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = tmpDir
	cfg.DefaultStorage = storage.NewFile(tmpDir)

	// Create torrent client
	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}
	defer client.Close()

	// Add the magnet link
	t, err := client.AddMagnet(magnetURI)
	if err != nil {
		return nil, fmt.Errorf("failed to add magnet: %w", err)
	}

	// Wait for torrent info
	<-t.GotInfo()

	// Create list of video files with their actual indices
	files := make([]TorrentFileInfo, 0)
	for i, file := range t.Files() {
		if IsVideoFile(file.Path()) {
			files = append(files, TorrentFileInfo{
				DisplayName: fmt.Sprintf("%s (%s)",
					file.Path(),
					humanize.Bytes(uint64(file.Length()))),
				ActualIndex: i,
			})
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no video files found in torrent")
	}

	return files, nil
}

func GetWebtorrentStreamURL(magnetURI string, selectedIndex int) (string, error) {
	// Create temporary directory for downloads
	tmpDir, err := os.MkdirTemp("", "torrent-stream-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Configure torrent client
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = tmpDir
	cfg.DefaultStorage = storage.NewFile(tmpDir)

	// Create torrent client
	client, err := torrent.NewClient(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create torrent client: %w", err)
	}
	defer client.Close()

	// Add the magnet link
	t, err := client.AddMagnet(magnetURI)
	if err != nil {
		return "", fmt.Errorf("failed to add magnet: %w", err)
	}

	// Wait for torrent info
	<-t.GotInfo()

	// Get the selected file
	selectedFile := t.Files()[selectedIndex]

	// Split the path and encode each component separately
	pathComponents := strings.Split(selectedFile.Path(), "/")
	encodedComponents := make([]string, len(pathComponents))
	for i, component := range pathComponents {
		encodedComponents[i] = url.PathEscape(component)
	}

	// Construct the webtorrent URL with properly encoded path components
	streamURL := fmt.Sprintf("http://localhost:8000/webtorrent/%x/%s",
		t.InfoHash(),
		strings.Join(encodedComponents, "/"))

	return streamURL, nil
}

// Add this helper function to check if a port is in use
func isPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	listener.Close()
	return false
}

func StreamTorrentWebtorrent(magnetURI string, selectedIndex int) (string, error) {
	// Start webtorrent server
	err := StartWebtorrentServer(magnetURI, selectedIndex)
	if err != nil {
		return "", err
	}

	// Get the stream URL
	streamURL, err := GetWebtorrentStreamURL(magnetURI, selectedIndex)
	if err != nil {
		return "", err
	}

	Debug("Stream URL: %s", streamURL)

	// Create socket path with random component
	socketPath := filepath.Join("/tmp", fmt.Sprintf("buttercup-%x.sock", time.Now().UnixNano()))

	// Start MPV once server is ready
	mpvCmd := exec.Command("mpv",
		"--force-seekable=yes",
		"--input-ipc-server="+socketPath,
		"--cache=yes",
		"--cache-secs=10",
		"--demuxer-max-bytes=50M",
		"--demuxer-readahead-secs=5",
		"--really-quiet",
		streamURL,
	)

	// Redirect output to /dev/null
	mpvCmd.Stdout = nil
	mpvCmd.Stderr = nil

	err = mpvCmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start mpv: %w", err)
	}

	Debug("Started mpv successfully")
	return socketPath, nil
}

func StreamTorrentSequentially(magnetURI string) error {
	// Create temporary directory for downloads
	tmpDir, err := os.MkdirTemp("", "torrent-stream-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Configure torrent client
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = tmpDir
	cfg.DefaultStorage = storage.NewFile(tmpDir)

	// Create torrent client
	client, err := torrent.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create torrent client: %w", err)
	}
	defer client.Close()

	// Add the magnet link
	t, err := client.AddMagnet(magnetURI)
	if err != nil {
		return fmt.Errorf("failed to add magnet: %w", err)
	}

	// Wait for torrent info
	<-t.GotInfo()

	// Create options map for selection menu
	options := make(map[string]string)
	for i, file := range t.Files() {
		options[fmt.Sprintf("%d", i)] = fmt.Sprintf("%s (%s)",
			file.Path(),
			humanize.Bytes(uint64(file.Length())))
	}

	// Use our existing selection menu
	selected, err := DynamicSelect(options)
	if err != nil {
		return fmt.Errorf("selection error: %w", err)
	}

	if selected.Key == "-1" {
		return fmt.Errorf("selection cancelled")
	}

	// Convert selected key to index
	selectedIndex, _ := strconv.Atoi(selected.Key)
	selectedFile := t.Files()[selectedIndex]

	// Create a reader for the selected file
	reader := selectedFile.NewReader()
	reader.SetResponsive() // Enable seeking
	defer reader.Close()

	streamManager := NewStreamManager(&reader, t)
	streamManager.prioritizeFromPosition(0)

	// Create named pipe for MPV
	pipePath := filepath.Join(tmpDir, "stream.pipe")
	err = syscall.Mkfifo(pipePath, 0600)
	if err != nil {
		return fmt.Errorf("failed to create named pipe: %w", err)
	}

	mpvPath, err := exec.LookPath("mpv")
	if err != nil {
		return fmt.Errorf("mpv not found: %w", err)
	}

	// Create MPV socket path
	socketPath := filepath.Join(tmpDir, "mpvsocket")

	// Start MPV with socket
	cmd := exec.Command(mpvPath,
		"--force-seekable=yes",
		"--input-ipc-server="+socketPath,
		"--cache=yes",                // Enable cache
		"--cache-secs=10",            // Cache 10 seconds
		"--demuxer-max-bytes=50M",    // Allow larger forward cache
		"--demuxer-readahead-secs=5", // Read 5 seconds ahead
		pipePath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}

	// Open pipe for writing
	pipe, err := os.OpenFile(pipePath, os.O_WRONLY, 0600)
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to open pipe: %w", err)
	}
	defer pipe.Close()

	// Start position monitoring goroutine using MPV socket
	done := make(chan struct{})
	go func() {
		defer close(done)
		var lastPos float64 = -1

		// Wait for socket to be created
		time.Sleep(time.Second)

		for {
			select {
			case <-done:
				return
			default:
				pos, err := getMPVPosition(socketPath)
				if err != nil {
					time.Sleep(time.Second)
					continue
				}

				if pos != lastPos {
					fmt.Printf("\nPlayback position changed to: %.2f seconds\n", pos)
					fmt.Printf("File length: %d bytes, Video length: %d seconds\n",
						selectedFile.Length(),
						t.Info().Length)

					// Convert seconds to bytes (approximate)
					bytesPerSecond := float64(selectedFile.Length()) / float64(t.Info().Length)
					bytePos := int64(pos * bytesPerSecond)
					fmt.Printf("Estimated byte position: %d\n", bytePos)

					streamManager.prioritizeFromPosition(bytePos)
					lastPos = pos
				}

				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Simple copy loop with better error handling
	buf := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				// This might happen during seeking, wait a bit and continue
				time.Sleep(100 * time.Millisecond)
				continue
			}
			cmd.Process.Kill()
			done <- struct{}{}
			return fmt.Errorf("read error: %w", err)
		}

		_, err = pipe.Write(buf[:n])
		if err != nil {
			if err == syscall.EPIPE {
				// MPV was closed
				break
			}
			cmd.Process.Kill()
			done <- struct{}{}
			return fmt.Errorf("write error: %w", err)
		}
	}

	done <- struct{}{}
	<-done
	return cmd.Wait()
}

func FindAndSortEpisodes(files []string) []string {
	type Episode struct {
		Path    string
		Season  int
		Episode int
	}

	var episodes []Episode

	// Regular expression to match common episode patterns
	// Matches: s01e01, s1e1, 1x01, etc.
	seasonEpRegex := regexp.MustCompile(`(?i)s(\d{1,2})e(\d{1,2})|(\d{1,2})x(\d{1,2})`)

	for _, file := range files {
		matches := seasonEpRegex.FindStringSubmatch(strings.ToLower(file))
		if matches != nil {
			var season, episode int
			if matches[1] != "" {
				// s01e01 format
				season, _ = strconv.Atoi(matches[1])
				episode, _ = strconv.Atoi(matches[2])
			} else {
				// 1x01 format
				season, _ = strconv.Atoi(matches[3])
				episode, _ = strconv.Atoi(matches[4])
			}
			episodes = append(episodes, Episode{
				Path:    file,
				Season:  season,
				Episode: episode,
			})
		}
	}

	// Sort episodes by season and episode number
	sort.Slice(episodes, func(i, j int) bool {
		if episodes[i].Season != episodes[j].Season {
			return episodes[i].Season < episodes[j].Season
		}
		return episodes[i].Episode < episodes[j].Episode
	})

	// Convert back to sorted paths
	sortedFiles := make([]string, len(episodes))
	for i, ep := range episodes {
		sortedFiles[i] = ep.Path
	}

	return sortedFiles
}

// Update the cleanup function to be more thorough
func CleanupWebtorrent() {
	// Kill any existing webtorrent processes
	exec.Command("pkill", "-f", "webtorrent").Run()
	
	// Also kill any process using our ports
	exec.Command("fuser", "-k", "8000/tcp").Run()
	exec.Command("fuser", "-k", "42069/tcp").Run()
	
	// If we have a stored process, kill it directly
	if currentWebtorrentProcess != nil {
		currentWebtorrentProcess.Kill()
		currentWebtorrentProcess = nil
	}

	// Give it a moment to clean up
	time.Sleep(500 * time.Millisecond)
}

