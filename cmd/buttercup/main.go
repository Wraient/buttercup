package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"Github.com/wraient/buttercup/internal"
)

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging")
	editConfig := flag.Bool("e", false, "Edit configuration file")	
	flag.Parse()

	internal.InitLogger(*debug)
	
	configPath := os.ExpandEnv("$HOME/.config/buttercup/config")

	// Handle config editing flag
	if *editConfig {
		cmd := exec.Command("vim", configPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			internal.Exit("Failed to open config in vim", err)
		}
		return
	}

	// Load config from default location
	internal.Debug("Loading config from default location")
	config, err := internal.LoadConfig(configPath)
	if err != nil {
		internal.Exit("Failed to load config", err)
	}

	// Check if Jackett is available
	if err := internal.CheckJackettAvailability(&config); err != nil {
		internal.Debug("Jackett not available")
		if config.RunJackettAtStartup {
			internal.Info("Starting Jackett service...")
			err := internal.StartJackett()
			if err != nil {
				internal.Info("Failed to start Jackett", err)
			}
		}

		if err := internal.CheckJackettAvailability(&config); err != nil {

			// Create options map for Jackett setup menu
			jackettOptions := map[string]string{
				"1": "Install Jackett",
				"2": "Configure Jackett URL and API key manually",
			}

			selected, err := internal.DynamicSelect(jackettOptions)
			if err != nil {
				internal.Exit("Error showing Jackett setup menu", err)
			}

			switch selected.Key {
			case "1":
				if err := internal.InstallJackett(); err != nil {
					internal.Exit("Failed to install Jackett", err)
				}
				internal.Info("Starting Jackett service...")
				err := internal.StartJackett()
				if err != nil {
					internal.Exit("Failed to start Jackett", err)
				}
				if err := internal.CheckJackettAvailability(&config); err != nil {
					internal.Exit("Failed to check Jackett availability", err)
				}

				internal.Info("Getting Jackett API key...")
				apiKey, err := internal.GetJackettApiKey()
				if err != nil {
					internal.Exit("Failed to get Jackett API key", err)
				}
				internal.Info("Jackett API key: %s", apiKey)
				
				config.JackettApiKey = apiKey
				internal.SetGlobalConfig(&config)
				
				// Save updated config
				if err := internal.SaveConfig(configPath, config); err != nil {
					internal.Exit("Failed to save config", err)
				}


			case "2":
				fmt.Print("Enter Jackett URL (e.g., 127.0.0.1): ")
				fmt.Scanln(&config.JackettUrl)
				
				fmt.Print("Enter Jackett Port (e.g., 9117): ")
				fmt.Scanln(&config.JackettPort)
				
				fmt.Print("Enter Jackett API Key: ")
				fmt.Scanln(&config.JackettApiKey)

				// Save the updated config
				if err := internal.SaveConfig(configPath, config); err != nil {
					internal.Exit("Failed to save config", err)
				}

			default:
				internal.Exit("No selection made", nil)
			}
		}
	}

	if config.RunJackettAtStartup {
		// Get Jackett API key and store it in config
		if config.JackettApiKey == "" {
			internal.Info("Getting Jackett API key...")
			apiKey, err := internal.GetJackettApiKey()
			if err != nil {
				internal.Exit("Failed to get Jackett API key", err)
			}
			internal.Info("Jackett API key: %s", apiKey)
			
			config.JackettApiKey = apiKey
			internal.SetGlobalConfig(&config)
			
			// Save updated config
			if err := internal.SaveConfig(configPath, config); err != nil {
				internal.Exit("Failed to save config", err)
			}
		}
		internal.SetGlobalConfig(&config)
		internal.Info("Jackett API key: %s", config.JackettApiKey)
	}

	internal.Debug("Config loaded successfully: %+v", config)

	// Load animes in database
	databaseFile := filepath.Join(os.ExpandEnv(config.StoragePath), "torrent_history.txt")
	databaseTorrents := internal.LocalGetAllTorrents(databaseFile)

    defer internal.CleanupPeerflix() // Keep this as a backup

	// Add initial menu options
	initialOptions := map[string]string{
		"1": "Start New Show",
		"2": "Continue Watching",
	}

	initialSelection, err := internal.DynamicSelect(initialOptions)
	if err != nil {
		internal.Exit("Error showing initial menu", err)
	}

	var selected internal.SelectionOption
	var user internal.User

	switch initialSelection.Key {
	case "1":
		// Handle new search
		var searchQuery string
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter search query: ")
		input, _ := reader.ReadString('\n')
		searchQuery = strings.TrimSpace(input)
	
		// Search Jackett with the provided query
		jackettResponse, err := internal.SearchJackett(searchQuery)
		if err != nil {
			internal.Exit("Error searching jackett", err)
		}
	
		// Check if we got any results
		if len(jackettResponse.Results) == 0 {
			internal.Exit("No results found", nil)
		}
	
		// Create options map for selection menu
		options := make(map[string]string)
		for i, result := range jackettResponse.Results {
			// Format the size to be human readable
			// size := internal.FormatSize(result.Size)
			
			// Format display string with pipe separation
			key := fmt.Sprintf("%d", i)
			// Format: "title|seeders|uri"
			options[key] = fmt.Sprintf("%s|%d|%s", 
				result.Title,
				result.Seeders,
				result.Tracker)
		}
	
		// Show selection menu
		selected, err = internal.DynamicSelect(options) 
		if err != nil {
			internal.Exit("Error showing selection menu", err)
		}
	
		if selected.Key == "-1" {
			internal.Info("No selection made, exiting")
			internal.Exit("No selection made, exiting", nil)
		}
	
		// Get the selected result using the index 
		selectedIndex, _ := strconv.Atoi(selected.Key)
		selectedResult := jackettResponse.Results[selectedIndex]
	
		internal.Debug("Selected: %s", selectedResult)
	
		// Ensure the MagnetUri is correctly retrieved
		user.Watching.URI = selectedResult.MagnetUri
		if user.Watching.URI == "" {
			user.Watching.URI, err = internal.FetchMagnetURI(selectedResult.Guid)
			if err != nil {
				internal.Exit("Failed to retrieve magnet URI", err)
			}
		}

		// Get list of files in the torrent
		user.Watching.Files, err = internal.GetTorrentFiles(user.Watching.URI)
		if err != nil {
			internal.Exit("Failed to get torrent files", err)
		}

		// Show file selection menu for new shows only
		options = make(map[string]string)
		for i, file := range user.Watching.Files {
			key := fmt.Sprintf("%d", i)
			options[key] = file
		}

		selected, err = internal.DynamicSelect(options)
		if err != nil {
			internal.Exit("Error showing selection menu", err)
		}

		if selected.Key == "-1" {
			internal.Exit("No selection made, exiting", nil)
		}

		user.Watching.FileIndex, _ = strconv.Atoi(selected.Key)

	case "2":
		if len(databaseTorrents) == 0 {
			internal.Exit("No shows in watch history", nil)
		}
		// Create options map for database selection
		dbOptions := make(map[string]string)
		for i, torrent := range databaseTorrents {
			dbOptions[fmt.Sprintf("%d", i)] = fmt.Sprintf("%s|%s", 
				torrent.Title,
				torrent.FileName)
		}

		// Show selection menu
		selected, err = internal.DynamicSelect(dbOptions)
		if err != nil {
			internal.Exit("Error showing selection menu", err)
		}

		if selected.Key == "-1" {
			internal.Exit("No selection made, exiting", nil)
		}

		// Get the selected torrent
		selectedIndex, _ := strconv.Atoi(selected.Key)
		selectedTorrent := databaseTorrents[selectedIndex]

		// Set up user watching details
		user.Watching.URI = selectedTorrent.MagnetURI
		user.Watching.FileIndex = selectedTorrent.FileIndex
		user.Player.PlaybackTime = selectedTorrent.PlaybackTime
		user.Resume = true

		internal.Info("Resuming %s at %d seconds", selectedTorrent.FileName, user.Player.PlaybackTime)
	}

	internal.Debug("MagnetUri: %s", user.Watching.URI)

	// Get list of files in the torrent
	user.Watching.Files, err = internal.GetTorrentFiles(user.Watching.URI)
	if err != nil {
		internal.Exit("Failed to get torrent files", err)
	}
	
	// Start streaming directly with the selected/resumed file index
	user.Player.SocketPath, err = internal.StreamTorrentPeerflix(user.Watching.URI, user.Watching.FileIndex)
	if err != nil {
		internal.Exit("Failed to stream torrent", err)
	}

	internal.Debug("MPV socket path: %s", user.Player.SocketPath)


    for {
        // Get video duration
        go func() {
            for {
                if user.Player.Started {
                    if user.Player.Duration == 0 {
                        // Get video duration
                        durationPos, err := internal.MPVSendCommand(user.Player.SocketPath, []interface{}{"get_property", "duration"})
                        if err != nil {
                            internal.Debug("Error getting video duration: "+err.Error())
                        } else if durationPos != nil {
                            if duration, ok := durationPos.(float64); ok {
                                user.Player.Duration = int(duration + 0.5) // Round to nearest integer
                                internal.Debug(fmt.Sprintf("Video duration: %d seconds", user.Player.Duration))
                            } else {
                                internal.Debug("Error: duration is not a float64")
                            }
                        }
                        break
                    }
                }
                time.Sleep(1 * time.Second)
            }
        }()

		// Set the playback speed and seek to the playback time and check if player has started
		go func() {
			for {
				timePos, err := internal.MPVSendCommand(user.Player.SocketPath, []interface{}{"get_property", "time-pos"})
				if err != nil {
					internal.Debug("Error getting time position: "+err.Error())
				} else if timePos != nil {
					if !user.Player.Started {
						internal.Debug("Player started")
						if user.Resume {
							internal.Debug("Seeking to playback time: %d", user.Player.PlaybackTime)
							mpvOutput, err := internal.SeekMPV(user.Player.SocketPath, user.Player.PlaybackTime)
							if err != nil {
								internal.Debug("Error seeking to playback time: "+err.Error())
							} else {
								internal.Debug("MPV output: %v", mpvOutput)
							}
							user.Resume = false
						}
						user.Player.Started = true
						// Set the playback speed
						if config.SaveMpvSpeed {
							speedCmd := []interface{}{"set_property", "speed", user.Player.Speed}
							_, err := internal.MPVSendCommand(user.Player.SocketPath, speedCmd)
							if err != nil {
								internal.Debug("Error setting playback speed: "+err.Error())
							}
						}
						break
					}
				}
				time.Sleep(1 * time.Second)
			}
		}()

        // Playback monitoring and database updates
        skipLoop:
        for {
            time.Sleep(1 * time.Second)
            timePos, err := internal.MPVSendCommand(user.Player.SocketPath, []interface{}{"get_property", "time-pos"})
            if err != nil && user.Player.Started {
                internal.Debug("Error getting time position: "+err.Error())
                // MPV closed or error occurred
                // Check if we reached completion percentage before starting next episode
                if user.Player.Started { 
                    percentage := float64(user.Player.PlaybackTime) / float64(user.Player.Duration) * 100
                    if err != nil {
                        internal.Debug("Error getting percentage watched: "+err.Error())
                    }
                    internal.Debug(fmt.Sprintf("Percentage watched: %f", percentage))
                    internal.Debug(fmt.Sprintf("Percentage to mark complete: %d", config.PercentageToMarkCompleted))
                    if percentage >= float64(config.PercentageToMarkCompleted) {
                        // Sort episodes if not already sorted
                        if user.Watching.SortedFiles == nil {
                            user.Watching.SortedFiles = internal.FindAndSortEpisodes(user.Watching.Files)
                        }

                        // Find current episode in sorted list
                        currentFile := user.Watching.Files[user.Watching.FileIndex]
                        nextIndex := -1
                        for i, file := range user.Watching.SortedFiles {
                            if file == currentFile && i < len(user.Watching.SortedFiles)-1 {
                                nextIndex = i + 1
                                break
                            }
                        }

                        if nextIndex != -1 {
                            // Find the index in original files slice
                            for i, file := range user.Watching.Files {
                                if file == user.Watching.SortedFiles[nextIndex] {
                                    internal.Output(fmt.Sprintf("Starting next episode: %s", file))
                                    user.Watching.FileIndex = i
                                    user.Player.PlaybackTime = 0
                                    // Update database with new episode and reset playback time
                                    err = internal.LocalUpdateTorrent(databaseFile, user.Watching.URI, i, 0, file)
                                    if err != nil {
                                        internal.Debug(fmt.Sprintf("Error updating database for next episode: %v", err))
                                    }
                                    break skipLoop
                                }
                            }
                        } else {
                            internal.Output("No more episodes in series")
                            internal.Exit("", nil)
                        }
                    } else {
                        internal.Exit("", nil)
                    }
                }
                break skipLoop  // Add this to ensure we break the loop on any MPV error
            }

            // Episode started
            if timePos != nil && user.Player.Started {

                showPosition, ok := timePos.(float64)
                if !ok {
                    continue
                }

                // Update playback time
                user.Player.PlaybackTime = int(showPosition + 0.5)
                user.Player.Speed, err = internal.GetMPVPlaybackSpeed(user.Player.SocketPath)
                if err != nil {
                    internal.Debug(fmt.Sprintf("Error getting playback speed: %v", err))
                }
                // Save to database using the current file name as the title
                currentFileName := user.Watching.Files[user.Watching.FileIndex]
                err = internal.LocalUpdateTorrent(databaseFile, user.Watching.URI, user.Watching.FileIndex, user.Player.PlaybackTime, currentFileName)
                if err != nil {
                    internal.Debug(fmt.Sprintf("Error updating database: %v", err))
                }
                internal.Debug("Database updated successfully")
            }
        }

        // Start the next episode after the skipLoop if we have one
        if user.Player.PlaybackTime == 0 {  // This indicates we're ready for next episode
            var err error
            user.Player.Duration = 0  // Reset duration for new episode
            user.Player.Started = false  // Reset started flag
            user.Player.SocketPath, err = internal.StreamTorrentPeerflix(user.Watching.URI, user.Watching.FileIndex)
            if err != nil {
                internal.Debug(fmt.Sprintf("Error starting next episode: %v", err))
                internal.Exit("", err)
            }
        }
    }

    // Set up signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        internal.CleanupPeerflix()
        os.Exit(0)
    }()

}
