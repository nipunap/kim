#!/bin/bash

echo "🔧 Testing UI Corruption Fix"
echo "============================"
echo ""

echo "📋 Building Kim..."
go build -o kim ./cmd/kim

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"
echo ""

echo "🎯 Testing the UI fix:"
echo "1. The interface should now properly clear the screen"
echo "2. Commands like :topics should not corrupt the UI"
echo "3. The command line should stay at the bottom"
echo "4. Content should not overlap with status bar or command line"
echo ""

echo "🚀 Starting interactive mode..."
echo "   Try the :topics command that was causing UI corruption"
echo "   The interface should now render cleanly"
echo ""

echo "Press Enter to start interactive mode (or Ctrl+C to cancel)..."
read -r

./kim -i
