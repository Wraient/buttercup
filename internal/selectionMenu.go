package internal

import (
	"fmt"
	"sort"
	"strings"
	"strconv"
	"github.com/charmbracelet/bubbletea"
)

// SelectionOption holds the label and the internal key
type SelectionOption struct {
	Label   string
	Key     string
	Seeders int    // Add seeder count
	URI     string // Add tracker URI
}

// Model represents the application state for the selection prompt
type Model struct {
	options        map[string]string  // id -> name mapping
	filter         string
	filteredKeys   []SelectionOption
	selected       int
	terminalWidth  int
	terminalHeight int
	scrollOffset   int
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.terminalWidth = wsm.Width
		m.terminalHeight = wsm.Height
	}

	updateFilter := false

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.filteredKeys[m.selected] = SelectionOption{Label: "quit", Key: "-1"}
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				updateFilter = true
			}
		case "down":
			if m.selected < len(m.filteredKeys)-1 {
				m.selected++
			}
			if m.selected >= m.scrollOffset+m.visibleItemsCount() {
				m.scrollOffset++
			}
		case "up":
			if m.selected > 0 {
				m.selected--
			}
			if m.selected < m.scrollOffset {
				m.scrollOffset--
			}
		default:
			if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
				m.filter += msg.String()
				updateFilter = true
			}
		}
	}

	if updateFilter {
		m.filterOptions()
		m.selected = 0
		m.scrollOffset = 0
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	b.WriteString("Search (Press Ctrl+C to quit):\n")
	b.WriteString("Filter: " + m.filter + "\n\n")

	if len(m.filteredKeys) == 0 {
		b.WriteString("No matches found.\n")
	} else {
		visibleItems := m.visibleItemsCount()
		start := m.scrollOffset
		end := start + visibleItems
		if end > len(m.filteredKeys) {
			end = len(m.filteredKeys)
		}

		for i := start; i < end; i++ {
			entry := m.filteredKeys[i]
			prefix := "  "
			if i == m.selected {
				prefix = "▶ "
			}

			// Display format: Name [URI] Seeders
			if entry.Key != "-1" { // Not quit option
				b.WriteString(fmt.Sprintf("%s%s [%s] %d\n", 
					prefix, entry.Label, entry.URI, entry.Seeders))
			} else {
				b.WriteString(fmt.Sprintf("%s%s\n", prefix, entry.Label))
			}
		}
	}

	return b.String()
}

func (m Model) visibleItemsCount() int {
	return m.terminalHeight - 4
}

func (m *Model) filterOptions() {
    m.filteredKeys = []SelectionOption{}

    // First collect all matching items
    for id, name := range m.options {
        parts := strings.Split(name, "|")
        seeders := 0
        uri := ""
        label := parts[0]
        
        if len(parts) > 1 {
            seeders, _ = strconv.Atoi(parts[1])
        }
        if len(parts) > 2 {
            uri = parts[2]
        }

        if strings.Contains(strings.ToLower(label), strings.ToLower(m.filter)) {
            m.filteredKeys = append(m.filteredKeys, SelectionOption{
                Key:     id,
                Label:   label,
                Seeders: seeders,
                URI:     uri,
            })
        }
    }

    // Sort filteredKeys by seeders in descending order
    sort.Slice(m.filteredKeys, func(i, j int) bool {
        return m.filteredKeys[i].Seeders > m.filteredKeys[j].Seeders
    })
}

// DynamicSelect shows a selection menu and returns the selected option
func DynamicSelect(options map[string]string) (SelectionOption, error) {
	config := GetGlobalConfig()
	if config != nil && config.RofiSelection {
		return RofiSelect(options, false)
	}
	model := &Model{
		options:      options,
		filteredKeys: make([]SelectionOption, 0),
	}

	model.filterOptions()
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return SelectionOption{}, err
	}

	finalSelectionModel, ok := finalModel.(*Model)
	if !ok {
		return SelectionOption{}, fmt.Errorf("unexpected model type")
	}

	if finalSelectionModel.selected < len(finalSelectionModel.filteredKeys) {
		selected := finalSelectionModel.filteredKeys[finalSelectionModel.selected]
		return selected, nil
	}
	return SelectionOption{}, nil
}
