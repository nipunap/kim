# 🧪 Kim Interactive Mode Testing Checklist

## Pre-Testing Setup
- [ ] Build successful: `go build -o kim ./cmd/kim`
- [ ] All unit tests pass: `go test ./...`
- [ ] All integration tests pass: `go test ./internal/cmd -v`

## Interactive Mode Core Functionality

### 🎮 Command System Testing
- [ ] `:help` or `:h` - Shows comprehensive help
- [ ] `:topics` or `:t` - Lists topics (with active profile)
- [ ] `:groups` or `:g` - Lists consumer groups (with active profile)
- [ ] `:profile` or `:p` - Shows profiles
- [ ] `:clear` or `:c` - Clears screen
- [ ] `:refresh` or `:r` - Refreshes current view
- [ ] `:q`, `:quit`, `:exit` - All quit properly
- [ ] Invalid commands show helpful error messages

### ⌨️ Input Handling
- [ ] `:` enters command mode correctly
- [ ] `/` enters search mode correctly
- [ ] `Esc` cancels command/search mode
- [ ] `Ctrl+C` cancels command mode
- [ ] `Ctrl+U` clears entire command line
- [ ] `Ctrl+W` deletes last word
- [ ] `Backspace` works in command mode
- [ ] Only printable characters accepted in command mode

### 🖥️ Window Management
- [ ] Screen uses full terminal (alt-screen mode)
- [ ] Header shows current profile and view
- [ ] Status bar shows helpful information
- [ ] Command line shows current mode
- [ ] Content scrolls properly when too long
- [ ] Responsive to terminal resize
- [ ] Text truncates properly on narrow terminals

### 🧭 Navigation
- [ ] `j` or `↓` scrolls down
- [ ] `k` or `↑` scrolls up
- [ ] `f` or `PgDn` pages down
- [ ] `b` or `PgUp` pages up
- [ ] `g` goes to top
- [ ] `G` goes to bottom
- [ ] `r` refreshes current view
- [ ] `q` quits immediately

### 📊 Content Display
- [ ] Help content shows all commands and shortcuts
- [ ] Profile list displays correctly (if profiles exist)
- [ ] Topics list displays correctly (with active profile)
- [ ] Groups list displays correctly (with active profile)
- [ ] Scroll indicators show when content is long
- [ ] Content fits within terminal width

## Error Handling
- [ ] No active profile shows appropriate message
- [ ] Connection errors show helpful messages
- [ ] Invalid commands show suggestions
- [ ] Graceful handling of terminal resize
- [ ] No crashes or panics during normal use

## Performance
- [ ] Responsive command execution
- [ ] Smooth scrolling
- [ ] Quick screen updates
- [ ] No lag in input handling

## Edge Cases
- [ ] Very narrow terminal (< 40 chars)
- [ ] Very short terminal (< 10 lines)
- [ ] Empty content handling
- [ ] Long command lines
- [ ] Rapid key presses
- [ ] Terminal resize during operation

## Integration
- [ ] Works with existing profiles
- [ ] Maintains compatibility with CLI mode
- [ ] Debug logging works (with --debug flag)
- [ ] Config file integration works
