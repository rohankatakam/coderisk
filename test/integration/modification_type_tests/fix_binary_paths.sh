#!/bin/bash
# Quick fix script to update binary paths in all test scenarios

echo "Fixing binary paths in test scripts..."

for script in scenario_*.sh; do
    if [ -f "$script" ]; then
        echo "  Fixing $script..."

        # Create temp file with fixed content
        awk '
        /^CRISK_BIN=/ {
            print "CRISK_BIN=\"./bin/crisk\"  # Will auto-detect ./crisk if bin/crisk not found"
            next
        }
        /# 1\. Verify/ {
            in_verify_block = 1
        }
        in_verify_block && /^fi$/ {
            # Replace entire verify block
            print "# 1. Verify we'\''re in coderisk-go root and find crisk binary"
            print "if [ -f \"./bin/crisk\" ]; then"
            print "    CRISK_BIN=\"./bin/crisk\""
            print "elif [ -f \"./crisk\" ]; then"
            print "    CRISK_BIN=\"./crisk\""
            print "else"
            print "    echo \"ERROR: crisk binary not found\""
            print "    echo \"Expected at: ./bin/crisk or ./crisk\""
            print "    echo \"Current directory: $(pwd)\""
            print "    echo \"Please run: make build\""
            print "    exit 1"
            print "fi"
            print "echo \"Using binary: $CRISK_BIN\""
            in_verify_block = 0
            next
        }
        in_verify_block {
            next
        }
        { print }
        ' "$script" > "$script.tmp"

        mv "$script.tmp" "$script"
        chmod +x "$script"
    fi
done

echo "✅ All scripts fixed!"
echo ""
echo "Test by running:"
echo "  ./run_all_tests.sh"
