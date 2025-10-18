#!/bin/bash
# Test script for first-run experience

echo "Testing first-run detection..."

# Read current config
if [ -f ~/.todobi.conf ]; then
    echo "Current config exists"
    if grep -q "github_setup_complete.*true" ~/.todobi.conf; then
        echo "GitHub setup is already complete"
    else
        echo "GitHub setup is NOT complete - first-run will trigger"
    fi
else
    echo "No config file - first-run will trigger"
fi

echo ""
echo "Build successful! The first-run experience will show when:"
echo "  1. No config file exists (first install)"
echo "  2. Config file exists but github_setup_complete is false/missing"
echo ""
echo "To test, you can:"
echo "  1. Delete ~/.todobi.conf and run ./todobi-test"
echo "  2. Or edit ~/.todobi.conf and set 'github_setup_complete' to false"
