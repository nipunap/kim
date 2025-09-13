#!/bin/bash

echo "🧪 Testing Kim Interactive Mode Improvements"
echo "============================================="
echo ""

echo "📋 Building Kim..."
go build -o kim ./cmd/kim

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"
echo ""

echo "🎮 Testing Interactive Mode Features:"
echo "1. Window management and screen sizing"
echo "2. Command handling (beyond just :q)"
echo "3. Improved help system"
echo "4. Better input handling"
echo ""

echo "🚀 Starting interactive mode..."
echo "   Try these commands:"
echo "   - :help (or :h) - Show help"
echo "   - :topics (or :t) - List topics (requires active profile)"
echo "   - :groups (or :g) - List groups (requires active profile)"
echo "   - :profile (or :p) - Show profiles"
echo "   - :clear (or :c) - Clear screen"
echo "   - :refresh (or :r) - Refresh current view"
echo "   - :q, :quit, or :exit - Quit"
echo ""
echo "   Navigation:"
echo "   - j/k or ↓/↑ - Scroll up/down"
echo "   - f/b or PgDn/PgUp - Page up/down"
echo "   - g/G - Go to top/bottom"
echo ""
echo "   Other:"
echo "   - / - Search mode"
echo "   - Esc - Cancel command/search"
echo "   - Ctrl+C - Force quit"
echo ""

echo "Press Enter to start interactive mode (or Ctrl+C to cancel)..."
read -r

./kim -i
