package ui

import (
	"context"
	"fmt"
	"strings"

	"kim/internal/client"
	"kim/internal/config"
	"kim/internal/logger"
	"kim/internal/manager"
	"kim/pkg/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InteractiveMode represents the interactive UI state
type InteractiveMode struct {
	cfg           *config.Config
	log           *logger.Logger
	clientManager *client.Manager
	currentView   string
	content       string
	statusMsg     string
	commandMode   bool
	searchMode    bool
	currentCmd    string
	searchPattern string
	scrollOffset  int
	maxLines      int
	width         int
	height        int
}

// NewInteractiveMode creates a new interactive mode instance
func NewInteractiveMode(cfg *config.Config, log *logger.Logger) *InteractiveMode {
	return &InteractiveMode{
		cfg:           cfg,
		log:           log,
		clientManager: client.NewManager(log),
		currentView:   "help",
		content:       getHelpContent(),
		statusMsg:     "Ready - Type :help for commands",
		maxLines:      20,
		height:        30, // Default height
		width:         80, // Default width
	}
}

// Run starts the interactive mode
func (im *InteractiveMode) Run() error {
	p := tea.NewProgram(im, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Init implements tea.Model
func (im *InteractiveMode) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (im *InteractiveMode) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		im.width = msg.Width
		im.height = msg.Height
		if msg.Height > 6 {
			im.maxLines = msg.Height - 6 // Account for header, status, and command line
		} else {
			im.maxLines = 1 // Minimum of 1 line
		}

	case tea.KeyMsg:
		return im.handleKeyPress(msg)
	}

	return im, nil
}

// View implements tea.Model
func (im *InteractiveMode) View() string {
	// Create styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	// Build header
	profile := "None"
	if im.cfg.ActiveProfile != "" {
		profile = im.cfg.ActiveProfile
	}
	header := headerStyle.Render(fmt.Sprintf("Kim - Kafka Management Tool | Profile: %s | View: %s", profile, im.currentView))

	// Build content with scrolling
	contentLines := strings.Split(im.content, "\n")
	visibleLines := im.getVisibleContent(contentLines)
	content := strings.Join(visibleLines, "\n")

	// Build status bar
	scrollInfo := ""
	if len(contentLines) > im.maxLines {
		scrollInfo = fmt.Sprintf(" | Line %d-%d of %d",
			im.scrollOffset+1,
			min(im.scrollOffset+im.maxLines, len(contentLines)),
			len(contentLines))
	}
	status := statusStyle.Render(im.statusMsg + scrollInfo)

	// Build command line
	commandLine := ""
	if im.commandMode {
		commandLine = commandStyle.Render(":" + im.currentCmd)
	} else if im.searchMode {
		commandLine = commandStyle.Render("/" + im.searchPattern)
	} else {
		commandLine = commandStyle.Render("Press ':' for commands, '/' to search, 'q' to quit")
	}

	// Combine all parts
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		status,
		commandLine,
	)
}

// handleKeyPress handles keyboard input
func (im *InteractiveMode) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case im.commandMode:
		return im.handleCommandMode(msg)
	case im.searchMode:
		return im.handleSearchMode(msg)
	default:
		return im.handleNormalMode(msg)
	}
}

// handleNormalMode handles normal mode key presses
func (im *InteractiveMode) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return im, tea.Quit

	case ":":
		im.commandMode = true
		im.currentCmd = ""
		return im, nil

	case "/":
		im.searchMode = true
		im.searchPattern = ""
		return im, nil

	case "j", "down":
		im.scrollDown()
		return im, nil

	case "k", "up":
		im.scrollUp()
		return im, nil

	case "f", "pgdown":
		im.scrollPageDown()
		return im, nil

	case "b", "pgup":
		im.scrollPageUp()
		return im, nil

	case "g":
		im.scrollToTop()
		return im, nil

	case "G":
		im.scrollToBottom()
		return im, nil

	case "r":
		return im.refreshCurrentView()
	}

	return im, nil
}

// handleCommandMode handles command mode key presses
func (im *InteractiveMode) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := im.currentCmd
		im.commandMode = false
		im.currentCmd = ""
		return im.executeCommand(cmd)

	case "esc":
		im.commandMode = false
		im.currentCmd = ""
		return im, nil

	case "backspace":
		if len(im.currentCmd) > 0 {
			im.currentCmd = im.currentCmd[:len(im.currentCmd)-1]
		}
		return im, nil

	default:
		if len(msg.String()) == 1 {
			im.currentCmd += msg.String()
		}
		return im, nil
	}
}

// handleSearchMode handles search mode key presses
func (im *InteractiveMode) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		pattern := im.searchPattern
		im.searchMode = false
		im.searchPattern = ""
		im.performSearch(pattern)
		return im, nil

	case "esc":
		im.searchMode = false
		im.searchPattern = ""
		return im, nil

	case "backspace":
		if len(im.searchPattern) > 0 {
			im.searchPattern = im.searchPattern[:len(im.searchPattern)-1]
		}
		return im, nil

	default:
		if len(msg.String()) == 1 {
			im.searchPattern += msg.String()
		}
		return im, nil
	}
}

// executeCommand executes a command
func (im *InteractiveMode) executeCommand(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return im, nil
	}

	switch parts[0] {
	case "q", "quit":
		return im, tea.Quit

	case "help":
		im.currentView = "help"
		im.content = getHelpContent()
		im.statusMsg = "Showing help"
		im.scrollOffset = 0

	case "topics":
		return im.showTopics()

	case "groups":
		return im.showGroups()

	case "profile":
		if len(parts) > 1 {
			return im.handleProfileCommand(parts[1:])
		}
		return im.showProfiles()

	default:
		im.statusMsg = fmt.Sprintf("Unknown command: %s", parts[0])
	}

	return im, nil
}

// showTopics displays the topics view
func (im *InteractiveMode) showTopics() (tea.Model, tea.Cmd) {
	profile, err := im.cfg.GetActiveProfile()
	if err != nil {
		im.statusMsg = "No active profile set"
		return im, nil
	}

	kafkaClient, err := im.clientManager.GetClient(profile)
	if err != nil {
		im.statusMsg = fmt.Sprintf("Failed to connect: %s", err.Error())
		return im, nil
	}

	topicManager := manager.NewTopicManager(kafkaClient, im.log)
	opts := &types.ListOptions{
		Page:     1,
		PageSize: 100,
		SortBy:   "name",
		Order:    "asc",
	}

	topicList, err := topicManager.ListTopics(context.Background(), opts)
	if err != nil {
		im.statusMsg = fmt.Sprintf("Failed to list topics: %s", err.Error())
		return im, nil
	}

	// Format topics for display
	var content strings.Builder
	content.WriteString("TOPICS\n")
	content.WriteString(strings.Repeat("=", 50) + "\n\n")

	if len(topicList.Topics) == 0 {
		content.WriteString("No topics found\n")
	} else {
		content.WriteString(fmt.Sprintf("%-40s %-10s %-15s\n", "NAME", "PARTITIONS", "REPLICATION"))
		content.WriteString(strings.Repeat("-", 65) + "\n")

		for _, topic := range topicList.Topics {
			content.WriteString(fmt.Sprintf("%-40s %-10d %-15d\n",
				topic.Name, topic.Partitions, topic.ReplicationFactor))
		}
	}

	im.currentView = "topics"
	im.content = content.String()
	im.statusMsg = fmt.Sprintf("Showing %d topics", len(topicList.Topics))
	im.scrollOffset = 0

	return im, nil
}

// showGroups displays the consumer groups view
func (im *InteractiveMode) showGroups() (tea.Model, tea.Cmd) {
	profile, err := im.cfg.GetActiveProfile()
	if err != nil {
		im.statusMsg = "No active profile set"
		return im, nil
	}

	kafkaClient, err := im.clientManager.GetClient(profile)
	if err != nil {
		im.statusMsg = fmt.Sprintf("Failed to connect: %s", err.Error())
		return im, nil
	}

	groupManager := manager.NewGroupManager(kafkaClient, im.log)
	opts := &types.ListOptions{
		Page:     1,
		PageSize: 100,
		SortBy:   "group_id",
		Order:    "asc",
	}

	groupList, err := groupManager.ListGroups(context.Background(), opts)
	if err != nil {
		im.statusMsg = fmt.Sprintf("Failed to list groups: %s", err.Error())
		return im, nil
	}

	// Format groups for display
	var content strings.Builder
	content.WriteString("CONSUMER GROUPS\n")
	content.WriteString(strings.Repeat("=", 50) + "\n\n")

	if len(groupList.Groups) == 0 {
		content.WriteString("No consumer groups found\n")
	} else {
		content.WriteString(fmt.Sprintf("%-30s %-15s %-15s\n", "GROUP ID", "STATE", "PROTOCOL TYPE"))
		content.WriteString(strings.Repeat("-", 60) + "\n")

		for _, group := range groupList.Groups {
			content.WriteString(fmt.Sprintf("%-30s %-15s %-15s\n",
				group.GroupID, group.State, group.ProtocolType))
		}
	}

	im.currentView = "groups"
	im.content = content.String()
	im.statusMsg = fmt.Sprintf("Showing %d consumer groups", len(groupList.Groups))
	im.scrollOffset = 0

	return im, nil
}

// showProfiles displays the profiles view
func (im *InteractiveMode) showProfiles() (tea.Model, tea.Cmd) {
	var content strings.Builder
	content.WriteString("PROFILES\n")
	content.WriteString(strings.Repeat("=", 50) + "\n\n")

	if len(im.cfg.Profiles) == 0 {
		content.WriteString("No profiles configured\n")
	} else {
		content.WriteString(fmt.Sprintf("%-20s %-8s %-30s %-6s\n", "NAME", "TYPE", "DETAILS", "ACTIVE"))
		content.WriteString(strings.Repeat("-", 64) + "\n")

		for name, profile := range im.cfg.Profiles {
			active := ""
			if name == im.cfg.ActiveProfile {
				active = "*"
			}

			details := ""
			switch profile.Type {
			case "msk":
				details = fmt.Sprintf("Region: %s", profile.Region)
			case "kafka":
				details = profile.BootstrapServers
			}

			content.WriteString(fmt.Sprintf("%-20s %-8s %-30s %-6s\n",
				name, profile.Type, details, active))
		}
	}

	im.currentView = "profiles"
	im.content = content.String()
	im.statusMsg = fmt.Sprintf("Showing %d profiles", len(im.cfg.Profiles))
	im.scrollOffset = 0

	return im, nil
}

// handleProfileCommand handles profile subcommands
func (im *InteractiveMode) handleProfileCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		return im.showProfiles()
	}

	switch args[0] {
	case "use":
		if len(args) < 2 {
			im.statusMsg = "Usage: profile use <name>"
			return im, nil
		}

		if err := im.cfg.SetActiveProfile(args[1]); err != nil {
			im.statusMsg = fmt.Sprintf("Failed to set profile: %s", err.Error())
		} else {
			im.statusMsg = fmt.Sprintf("Switched to profile: %s", args[1])
		}

	case "list":
		return im.showProfiles()

	default:
		im.statusMsg = fmt.Sprintf("Unknown profile command: %s", args[0])
	}

	return im, nil
}

// refreshCurrentView refreshes the current view
func (im *InteractiveMode) refreshCurrentView() (tea.Model, tea.Cmd) {
	switch im.currentView {
	case "topics":
		return im.showTopics()
	case "groups":
		return im.showGroups()
	case "profiles":
		return im.showProfiles()
	default:
		im.statusMsg = "View refreshed"
	}
	return im, nil
}

// performSearch performs a search in the current content
func (im *InteractiveMode) performSearch(pattern string) {
	if pattern == "" {
		return
	}

	lines := strings.Split(im.content, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			im.scrollOffset = max(0, i-2) // Show 2 lines before the match
			im.statusMsg = fmt.Sprintf("Found '%s' at line %d", pattern, i+1)
			return
		}
	}

	im.statusMsg = fmt.Sprintf("Pattern '%s' not found", pattern)
}

// Scrolling methods
func (im *InteractiveMode) scrollDown() {
	lines := strings.Split(im.content, "\n")
	if im.scrollOffset+im.maxLines < len(lines) {
		im.scrollOffset++
	}
}

func (im *InteractiveMode) scrollUp() {
	if im.scrollOffset > 0 {
		im.scrollOffset--
	}
}

func (im *InteractiveMode) scrollPageDown() {
	lines := strings.Split(im.content, "\n")
	im.scrollOffset = min(im.scrollOffset+im.maxLines, max(0, len(lines)-im.maxLines))
}

func (im *InteractiveMode) scrollPageUp() {
	im.scrollOffset = max(0, im.scrollOffset-im.maxLines)
}

func (im *InteractiveMode) scrollToTop() {
	im.scrollOffset = 0
}

func (im *InteractiveMode) scrollToBottom() {
	lines := strings.Split(im.content, "\n")
	im.scrollOffset = max(0, len(lines)-im.maxLines)
}

// getVisibleContent returns the visible portion of content based on scroll offset
func (im *InteractiveMode) getVisibleContent(lines []string) []string {
	if len(lines) <= im.maxLines {
		return lines
	}

	start := im.scrollOffset
	end := min(start+im.maxLines, len(lines))

	return lines[start:end]
}

// getHelpContent returns the help content
func getHelpContent() string {
	return `KIM - KAFKA MANAGEMENT TOOL
============================

COMMANDS:
  :help                 Show this help
  :topics               List all topics
  :groups               List consumer groups
  :profile list         List profiles
  :profile use <name>   Switch to profile
  :q or :quit           Quit

NAVIGATION:
  j/↓                   Scroll down
  k/↑                   Scroll up
  f/PgDn               Page down
  b/PgUp               Page up
  g                     Go to top
  G                     Go to bottom
  r                     Refresh current view

SEARCH:
  /<pattern>           Search for pattern

MODES:
  :                    Enter command mode
  /                    Enter search mode
  ESC                  Exit current mode

Press 'q' to quit or ':' to enter a command.`
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
