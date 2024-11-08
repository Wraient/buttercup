package internal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func RofiSelect(options map[string]string, addShowOpt bool) (SelectionOption, error) {
	config := GetGlobalConfig()
	if config.StoragePath == "" {
		config.StoragePath = os.ExpandEnv("${HOME}/.local/share/buttercup")
	}

	// Create a slice to store options with their seeders
	type optionWithSeeders struct {
		value   string
		key     string
		seeders int
	}
	var optionsList []optionWithSeeders

	// Parse options and extract seeder information
	for key, value := range options {
		parts := strings.Split(value, "|")
		seeders := 0
		label := parts[0]
		
		if len(parts) > 1 {
			seeders, _ = strconv.Atoi(parts[1])
		}

		optionsList = append(optionsList, optionWithSeeders{
			value:   label,
			key:     key,
			seeders: seeders,
		})
	}

	// Sort by seeders in descending order
	sort.Slice(optionsList, func(i, j int) bool {
		return optionsList[i].seeders > optionsList[j].seeders
	})

	// Create the final sorted list
	var sortedOptions []string
	for _, opt := range optionsList {
		sortedOptions = append(sortedOptions, opt.value)
	}

	// Add quit option
	sortedOptions = append(sortedOptions, "Quit")

	// Join all options into a single string, separated by newlines
	optionsString := strings.Join(sortedOptions, "\n")

	// Prepare the Rofi command
	cmd := exec.Command("rofi", "-dmenu", "-theme", filepath.Join(os.ExpandEnv(config.StoragePath), "select.rasi"), "-i", "-p", "Select a show")

	// Set up pipes for input and output
	cmd.Stdin = strings.NewReader(optionsString)
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return SelectionOption{}, fmt.Errorf("failed to run Rofi: %v", err)
	}

	// Get the selected option
	selected := strings.TrimSpace(out.String())

	// Handle special cases
	switch selected {
	case "":
		return SelectionOption{}, fmt.Errorf("no selection made")
	case "Quit":
		Exit("Have a great day!", nil)
	}

	// Find the key for the selected value
	for _, opt := range optionsList {
		if opt.value == selected {
			return SelectionOption{Label: selected, Key: opt.key}, nil
		}
	}

	return SelectionOption{}, fmt.Errorf("selected option not found in original list")
}


// GetUserInputFromRofi prompts the user for input using Rofi with a custom message
func GetUserInputFromRofi(message string) (string, error) {
	config := GetGlobalConfig()
	if config.StoragePath == "" {
		config.StoragePath = os.ExpandEnv("${HOME}/.local/share/buttercup")
	}
	// Create the Rofi command
	cmd := exec.Command("rofi", "-dmenu", "-theme", filepath.Join(os.ExpandEnv(config.StoragePath), "userInput.rasi"), "-p", "Input", "-mesg", message)
	
	Debug("Rofi command: %v", cmd.String())

	// Set up pipes for output
	var out bytes.Buffer
	cmd.Stdout = &out
	
	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run Rofi: %w", err)
	}
	
	// Get the entered input
	userInput := strings.TrimSpace(out.String())
	
	return userInput, nil
}