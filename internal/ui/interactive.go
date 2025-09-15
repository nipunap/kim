package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nipunap/kim/internal/client"
	"github.com/nipunap/kim/internal/config"
	"github.com/nipunap/kim/internal/logger"
	"github.com/nipunap/kim/internal/manager"
	"github.com/nipunap/kim/pkg/types"

	"golang.org/x/term"
)

// InteractiveMode represents the VIM-like interactive UI state
type InteractiveMode struct {
	cfg           *config.Config
	log           *logger.Logger
	clientManager *client.Manager

	// Display state
	currentView string
	content     []string
	statusMsg   string

	// Terminal state
	width         int
	height        int
	terminalState *term.State

	// VIM-like modes
	inCommandMode bool
	inSearchMode  bool
	currentCmd    string
	searchPattern string

	// Scrolling
	scrollOffset int
	visibleLines int

	// Command history
	commandHistory []string
	historyIndex   int
}

// NewInteractiveMode creates a new VIM-like interactive mode instance
func NewInteractiveMode(cfg *config.Config, log *logger.Logger) *InteractiveMode {
	// Get terminal size
	width, height, _ := term.GetSize(int(os.Stdin.Fd()))
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	return &InteractiveMode{
		cfg:            cfg,
		log:            log,
		clientManager:  client.NewManager(log),
		currentView:    "help",
		content:        strings.Split(getHelpContent(), "\n"),
		statusMsg:      "Ready - Type :help for commands",
		width:          width,
		height:         height,
		visibleLines:   height - 4, // Reserve space for status bar, empty line, command line, padding
		commandHistory: make([]string, 0),
		historyIndex:   0,
	}
}

// Run starts the VIM-like interactive mode
func (im *InteractiveMode) Run() error {
	// Save terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Clear screen and hide cursor
	fmt.Print("\033[2J\033[H\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor on exit

	// Main interactive loop
	reader := bufio.NewReader(os.Stdin)
	for {
		// Update terminal size
		im.updateTerminalSize()

		// Render the interface
		im.render()

		// Read single character
		char, err := im.readChar(reader)
		if err != nil {
			return err
		}

		// Handle input based on mode
		quit, err := im.handleInput(char)
		if err != nil {
			return err
		}
		if quit {
			break
		}
	}

	return nil
}

// updateTerminalSize updates the terminal dimensions
func (im *InteractiveMode) updateTerminalSize() {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil && width > 0 && height > 0 {
		im.width = width
		im.height = height
		im.visibleLines = height - 4 // Reserve space for status bar, empty line, command line, padding
	}
}

// render draws the VIM-like interface to the terminal
func (im *InteractiveMode) render() {
	// Clear screen and move cursor to top-left
	fmt.Print("\033[2J\033[H")

	// Render status bar (top)
	im.renderStatusBar()

	// Render main content
	im.renderContent()

	// Render command line (bottom)
	im.renderCommandLine()
}

// renderStatusBar renders the top status bar
func (im *InteractiveMode) renderStatusBar() {
	profile := "None"
	if im.cfg.ActiveProfile != "" {
		profile = im.cfg.ActiveProfile
	}

	// Create status line
	statusLine := fmt.Sprintf("Kim - Kafka Management Tool | Profile: %s | View: %s", profile, im.currentView)

	// Truncate if too long
	if len(statusLine) > im.width-2 {
		statusLine = statusLine[:im.width-5] + "..."
	}

	// Pad to full width and add colors
	padding := im.width - len(statusLine)
	if padding < 0 {
		padding = 0
	}

	fmt.Printf("\033[1;37;44m%s%s\033[0m\n", statusLine, strings.Repeat(" ", padding))
	fmt.Println() // Empty line after status
}

// renderContent renders the main content area
func (im *InteractiveMode) renderContent() {
	visibleContent := im.getVisibleContent()

	// Calculate available lines for content (total height - status bar - command line - padding)
	availableLines := im.height - 4 // 1 for status, 1 for empty line, 1 for command line, 1 for padding
	if availableLines < 1 {
		availableLines = 1
	}

	// Render each line of content
	linesRendered := 0
	for _, line := range visibleContent {
		if linesRendered >= availableLines {
			break
		}

		// Truncate line if too long
		if len(line) > im.width {
			if im.width > 3 {
				line = line[:im.width-3] + "..."
			} else {
				line = line[:im.width]
			}
		}

		fmt.Printf("%s\n", line)
		linesRendered++
	}

	// Fill remaining lines with empty space to prevent overlap
	for linesRendered < availableLines {
		fmt.Println()
		linesRendered++
	}
}

// renderCommandLine renders the bottom command line (VIM-like)
func (im *InteractiveMode) renderCommandLine() {
	var commandLine string

	if im.inCommandMode {
		commandLine = ":" + im.currentCmd
	} else if im.inSearchMode {
		commandLine = "/" + im.searchPattern
	} else {
		commandLine = "Press ':' for commands, '/' to search, 'q' to quit"
	}

	// Truncate if too long
	if len(commandLine) > im.width-2 {
		if im.width > 5 {
			commandLine = commandLine[:im.width-5] + "..."
		} else {
			commandLine = commandLine[:im.width-2]
		}
	}

	// Pad to full width and add colors
	padding := im.width - len(commandLine)
	if padding < 0 {
		padding = 0
	}

	// Move cursor to bottom of screen and render command line
	fmt.Printf("\033[%d;1H\033[1;37;40m%s%s\033[0m", im.height, commandLine, strings.Repeat(" ", padding))
}

// readChar reads a single character from the terminal
func (im *InteractiveMode) readChar(reader *bufio.Reader) (rune, error) {
	char, _, err := reader.ReadRune()
	return char, err
}

// handleInput handles input based on current mode
func (im *InteractiveMode) handleInput(char rune) (quit bool, err error) {
	if im.inCommandMode {
		return im.handleCommandMode(char)
	} else if im.inSearchMode {
		return im.handleSearchMode(char)
	} else {
		return im.handleNormalMode(char)
	}
}

// handleNormalMode handles VIM-like normal mode key presses
func (im *InteractiveMode) handleNormalMode(char rune) (quit bool, err error) {
	switch char {
	case 'q', '\x03': // q or Ctrl+C
		return true, nil

	case ':':
		im.inCommandMode = true
		im.currentCmd = ""
		return false, nil

	case '/':
		im.inSearchMode = true
		im.searchPattern = ""
		return false, nil

	case 'j': // Scroll down
		im.scrollDown()
		return false, nil

	case 'k': // Scroll up
		im.scrollUp()
		return false, nil

	case 'f': // Page down
		im.scrollPageDown()
		return false, nil

	case 'b': // Page up
		im.scrollPageUp()
		return false, nil

	case 'g': // Go to top
		im.scrollToTop()
		return false, nil

	case 'G': // Go to bottom
		im.scrollToBottom()
		return false, nil

	case 'r': // Refresh
		im.refreshCurrentView()
		return false, nil
	}

	return false, nil
}

// handleCommandMode handles VIM-like command mode key presses
func (im *InteractiveMode) handleCommandMode(char rune) (quit bool, err error) {
	switch char {
	case '\r', '\n': // Enter
		cmd := strings.TrimSpace(im.currentCmd)
		im.inCommandMode = false
		im.currentCmd = ""

		if cmd == "q" || cmd == "quit" || cmd == "exit" {
			return true, nil
		}

		if cmd != "" {
			im.executeCommand(cmd)
		}
		return false, nil

	case '\x1b': // Escape
		im.inCommandMode = false
		im.currentCmd = ""
		im.statusMsg = "Command cancelled"
		return false, nil

	case '\x7f', '\b': // Backspace
		if len(im.currentCmd) > 0 {
			im.currentCmd = im.currentCmd[:len(im.currentCmd)-1]
		}
		return false, nil

	case '\x15': // Ctrl+U - Clear line
		im.currentCmd = ""
		return false, nil

	default:
		// Add printable characters
		if char >= 32 && char <= 126 {
			im.currentCmd += string(char)
		}
		return false, nil
	}
}

// handleSearchMode handles VIM-like search mode key presses
func (im *InteractiveMode) handleSearchMode(char rune) (quit bool, err error) {
	switch char {
	case '\r', '\n': // Enter
		pattern := im.searchPattern
		im.inSearchMode = false
		im.searchPattern = ""
		if pattern != "" {
			im.performSearch(pattern)
		}
		return false, nil

	case '\x1b': // Escape
		im.inSearchMode = false
		im.searchPattern = ""
		im.statusMsg = "Search cancelled"
		return false, nil

	case '\x7f', '\b': // Backspace
		if len(im.searchPattern) > 0 {
			im.searchPattern = im.searchPattern[:len(im.searchPattern)-1]
		}
		return false, nil

	default:
		// Add printable characters
		if char >= 32 && char <= 126 {
			im.searchPattern += string(char)
		}
		return false, nil
	}
}

// executeCommand executes a VIM-like command
func (im *InteractiveMode) executeCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	im.log.Debug("Executing command", "cmd", cmd, "parts", parts)

	switch parts[0] {
	case "help", "h":
		im.currentView = "help"
		im.content = strings.Split(getHelpContent(), "\n")
		im.statusMsg = "Showing help"
		im.scrollOffset = 0

	case "topics", "t":
		im.showTopics()

	case "groups", "g":
		im.showGroups()

	case "profile", "p":
		if len(parts) > 1 {
			im.handleProfileCommand(parts[1:])
		} else {
			im.showProfiles()
		}

	case "refresh", "r":
		im.refreshCurrentView()

	case "clear", "c":
		im.content = []string{}
		im.statusMsg = "Screen cleared"
		im.scrollOffset = 0

	default:
		im.statusMsg = fmt.Sprintf("Unknown command: %s. Type 'help' for available commands.", parts[0])
	}
}

// showTopics displays the topics view
func (im *InteractiveMode) showTopics() {
	profile, err := im.cfg.GetActiveProfile()
	if err != nil {
		im.statusMsg = "No active profile set"
		return
	}

	kafkaClient, err := im.clientManager.GetClient(profile)
	if err != nil {
		im.statusMsg = fmt.Sprintf("Failed to connect: %s", err.Error())
		return
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
		return
	}

	// Format topics for display
	var contentLines []string
	contentLines = append(contentLines, "TOPICS")
	contentLines = append(contentLines, strings.Repeat("=", 50))
	contentLines = append(contentLines, "")

	if len(topicList.Topics) == 0 {
		contentLines = append(contentLines, "No topics found")
	} else {
		contentLines = append(contentLines, fmt.Sprintf("%-40s %-10s %-15s", "NAME", "PARTITIONS", "REPLICATION"))
		contentLines = append(contentLines, strings.Repeat("-", 65))

		for _, topic := range topicList.Topics {
			contentLines = append(contentLines, fmt.Sprintf("%-40s %-10d %-15d",
				topic.Name, topic.Partitions, topic.ReplicationFactor))
		}
	}

	im.currentView = "topics"
	im.content = contentLines
	im.statusMsg = fmt.Sprintf("Showing %d topics", len(topicList.Topics))
	im.scrollOffset = 0
}

// getVisibleContent returns the visible portion of content based on scroll offset
func (im *InteractiveMode) getVisibleContent() []string {
	if len(im.content) == 0 {
		return []string{"No content to display"}
	}

	start := im.scrollOffset
	end := start + im.visibleLines

	if start >= len(im.content) {
		start = len(im.content) - 1
		if start < 0 {
			start = 0
		}
	}

	if end > len(im.content) {
		end = len(im.content)
	}

	if start >= end {
		return im.content[start : start+1]
	}

	return im.content[start:end]
}

// Scrolling functions
func (im *InteractiveMode) scrollDown() {
	if im.scrollOffset < len(im.content)-im.visibleLines {
		im.scrollOffset++
	}
}

func (im *InteractiveMode) scrollUp() {
	if im.scrollOffset > 0 {
		im.scrollOffset--
	}
}

func (im *InteractiveMode) scrollPageDown() {
	im.scrollOffset += im.visibleLines
	if im.scrollOffset >= len(im.content) {
		im.scrollOffset = len(im.content) - 1
		if im.scrollOffset < 0 {
			im.scrollOffset = 0
		}
	}
}

func (im *InteractiveMode) scrollPageUp() {
	im.scrollOffset -= im.visibleLines
	if im.scrollOffset < 0 {
		im.scrollOffset = 0
	}
}

func (im *InteractiveMode) scrollToTop() {
	im.scrollOffset = 0
}

func (im *InteractiveMode) scrollToBottom() {
	im.scrollOffset = len(im.content) - im.visibleLines
	if im.scrollOffset < 0 {
		im.scrollOffset = 0
	}
}

// refreshCurrentView refreshes the current view
func (im *InteractiveMode) refreshCurrentView() {
	switch im.currentView {
	case "topics":
		im.showTopics()
	case "groups":
		im.showGroups()
	case "profiles":
		im.showProfiles()
	default:
		im.statusMsg = "Nothing to refresh"
	}
}

// performSearch performs a search in the current content
func (im *InteractiveMode) performSearch(pattern string) {
	// Simple search implementation
	im.statusMsg = fmt.Sprintf("Searching for: %s", pattern)
	// TODO: Implement actual search functionality
}

// showGroups displays the consumer groups view (simplified)
func (im *InteractiveMode) showGroups() {
	im.currentView = "groups"
	im.content = []string{"Consumer Groups", "=================", "", "Feature coming soon..."}
	im.statusMsg = "Consumer groups view"
	im.scrollOffset = 0
}

// showProfiles displays the profiles view (simplified)
func (im *InteractiveMode) showProfiles() {
	im.currentView = "profiles"
	var contentLines []string
	contentLines = append(contentLines, "PROFILES")
	contentLines = append(contentLines, strings.Repeat("=", 50))
	contentLines = append(contentLines, "")

	if len(im.cfg.Profiles) == 0 {
		contentLines = append(contentLines, "No profiles configured")
	} else {
		contentLines = append(contentLines, fmt.Sprintf("%-20s %-10s %-30s %-8s", "NAME", "TYPE", "DETAILS", "ACTIVE"))
		contentLines = append(contentLines, strings.Repeat("-", 70))

		for name, profile := range im.cfg.Profiles {
			active := ""
			if name == im.cfg.ActiveProfile {
				active = "*"
			}

			details := ""
			if profile.Type == "kafka" {
				details = fmt.Sprintf("Servers: %s", profile.BootstrapServers)
			} else if profile.Type == "msk" {
				details = fmt.Sprintf("Region: %s", profile.Region)
			}

			contentLines = append(contentLines, fmt.Sprintf("%-20s %-10s %-30s %-8s",
				name, profile.Type, details, active))
		}
	}

	im.content = contentLines
	im.statusMsg = fmt.Sprintf("Showing %d profiles", len(im.cfg.Profiles))
	im.scrollOffset = 0
}

// handleProfileCommand handles profile subcommands (simplified)
func (im *InteractiveMode) handleProfileCommand(args []string) {
	if len(args) == 0 {
		im.showProfiles()
		return
	}

	switch args[0] {
	case "list":
		im.showProfiles()
	case "use":
		if len(args) > 1 {
			if _, exists := im.cfg.Profiles[args[1]]; exists {
				im.cfg.ActiveProfile = args[1]
				im.statusMsg = fmt.Sprintf("Switched to profile: %s", args[1])
			} else {
				im.statusMsg = fmt.Sprintf("Profile not found: %s", args[1])
			}
		} else {
			im.statusMsg = "Usage: profile use <name>"
		}
	default:
		im.statusMsg = fmt.Sprintf("Unknown profile command: %s", args[0])
	}
}

// getHelpContent returns the help content
func getHelpContent() string {
	return `KIM - KAFKA MANAGEMENT TOOL
============================

COMMANDS:
  :help, :h             Show this help
  :topics, :t           List all topics
  :groups, :g           List consumer groups
  :profile, :p          List profiles
  :profile list         List all profiles
  :profile use <name>   Switch to profile
  :refresh, :r          Refresh current view
  :clear, :c            Clear screen
  :q, :quit, :exit      Quit

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

Press 'q' to quit or ':' to enter a command.
`
}
