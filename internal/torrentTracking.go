package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// TorrentData represents the structure for storing torrent playback information
type TorrentData struct {
	MagnetURI    string
	FileName     string
	FileIndex    int
	PlaybackTime int
	Title        string
}

// Function to add a torrent entry
func LocalAddTorrent(databaseFile string, magnetURI string, fileIndex int, playbackTime int, title string) {
	// Read existing entries first
	torrentList := LocalGetAllTorrents(databaseFile)

	// Check if magnet URI already exists
	for _, torrent := range torrentList {
		if torrent.MagnetURI == magnetURI {
			// Update existing entry instead of creating new one
			err := LocalUpdateTorrent(databaseFile, magnetURI, fileIndex, playbackTime, title)
			if err != nil {
				Output(fmt.Sprintf("Error updating existing torrent: %v", err))
			}
			return
		}
	}

	// If we get here, it's a new magnet URI
	file, err := os.OpenFile(databaseFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		Output(fmt.Sprintf("Error opening file: %v", err))
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '|'
	defer writer.Flush()

	err = writer.Write([]string{
		magnetURI,
		strconv.Itoa(fileIndex),
		strconv.Itoa(playbackTime),
		title,
	})
	if err != nil {
		Output(fmt.Sprintf("Error writing to file: %v", err))
	} else {
		Output("Written to file")
	}
}

// Function to get all torrent entries
func LocalGetAllTorrents(databaseFile string) []TorrentData {
	torrentList := []TorrentData{}

	// Ensure the directory exists
	dir := filepath.Dir(databaseFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		Output(fmt.Sprintf("Error creating directory: %v", err))
		return torrentList
	}

	// Open the file, create if it doesn't exist
	file, err := os.OpenFile(databaseFile, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		Output(fmt.Sprintf("Error opening or creating file: %v", err))
		return torrentList
	}
	defer file.Close()

	// If the file was just created, return empty list
	fileInfo, err := file.Stat()
	if err != nil {
		Output(fmt.Sprintf("Error getting file info: %v", err))
		return torrentList
	}
	if fileInfo.Size() == 0 {
		return torrentList
	}

	reader := csv.NewReader(file)
	reader.Comma = '|'
	reader.FieldsPerRecord = 4
	
	records, err := reader.ReadAll()
	if err != nil {
		Output(fmt.Sprintf("Error reading file: %v", err))
		return torrentList
	}

	for _, row := range records {
		t := parseTorrentRow(row)
		if t != nil {
			torrentList = append(torrentList, *t)
		}
	}

	return torrentList
}

// Function to parse a single row of torrent data
func parseTorrentRow(row []string) *TorrentData {
	if len(row) < 4 {
		Output(fmt.Sprintf("Invalid row format: %v", row))
		return nil
	}

	fileIndex, _ := strconv.Atoi(row[1])
	playbackTime, _ := strconv.Atoi(row[2])

	return &TorrentData{
		MagnetURI:    row[0],
		FileIndex:    fileIndex,
		PlaybackTime: playbackTime,
		Title:        row[3],
	}
}

// Function to update or add a torrent entry
func LocalUpdateTorrent(databaseFile string, magnetURI string, fileIndex int, playbackTime int, title string) error {
	// Read existing entries
	torrentList := LocalGetAllTorrents(databaseFile)

	// Find and update existing entry or add new one
	updated := false
	for i, torrent := range torrentList {
		if torrent.MagnetURI == magnetURI {
			torrentList[i].FileIndex = fileIndex
			torrentList[i].PlaybackTime = playbackTime
			torrentList[i].Title = title
			updated = true
			break
		}
	}

	if !updated {
		newTorrent := TorrentData{
			MagnetURI:    magnetURI,
			FileIndex:    fileIndex,
			PlaybackTime: playbackTime,
			Title:        title,
		}
		torrentList = append(torrentList, newTorrent)
	}

	// Write updated list back to file
	file, err := os.Create(databaseFile)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '|'
	defer writer.Flush()

	for _, torrent := range torrentList {
		record := []string{
			torrent.MagnetURI,
			strconv.Itoa(torrent.FileIndex),
			strconv.Itoa(torrent.PlaybackTime),
			torrent.Title,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error writing record: %v", err)
		}
	}
	
	writer.Flush()
	return nil
}

// Function to find a torrent by magnet URI and file index
func LocalFindTorrent(torrentList []TorrentData, magnetURI string, fileIndex int) *TorrentData {
	for _, torrent := range torrentList {
		if torrent.MagnetURI == magnetURI && torrent.FileIndex == fileIndex {
			return &torrent
		}
	}
	return nil
} 