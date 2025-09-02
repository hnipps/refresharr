#!/bin/bash

# RefreshArr Usage Examples
# This script demonstrates common usage patterns for RefreshArr

set -e

echo "RefreshArr - Missing File Cleanup Examples"
echo "=========================================="
echo

# Check if binary exists
if [ ! -f "./refresharr" ]; then
    echo "Building RefreshArr CLI..."
    go build -o refresharr main.go
    echo "✅ Build complete"
    echo
fi

# Example 1: Show version
echo "1. Version Information:"
echo "----------------------"
./refresharr --version
echo

# Example 2: Show help
echo "2. Help Information:"
echo "-------------------"
./refresharr --help | head -20
echo "... (truncated for brevity)"
echo

# Example 3: Check environment
echo "3. Environment Check:"
echo "--------------------"
if [ -z "$SONARR_API_KEY" ]; then
    echo "⚠️  SONARR_API_KEY not set in environment"
    echo "   Please set your Sonarr API key:"
    echo "   export SONARR_API_KEY='your-api-key-here'"
    echo
else
    echo "✅ SONARR_API_KEY is set"
fi

if [ -z "$SONARR_URL" ]; then
    echo "ℹ️  SONARR_URL not set, will use default: http://127.0.0.1:8989"
else
    echo "✅ SONARR_URL is set to: $SONARR_URL"
fi
echo

# Example 4: Dry run example (safe to run)
echo "4. Dry Run Example (safe to run):"
echo "---------------------------------"
if [ -n "$SONARR_API_KEY" ]; then
    echo "Running: ./refresharr --dry-run --log-level DEBUG"
    echo "This will show what would be cleaned up without making changes..."
    echo
    # Comment out the next line to actually run the dry run
    echo "(Commented out for safety - uncomment to test)"
    # ./refresharr --dry-run --log-level DEBUG
else
    echo "Skipped: SONARR_API_KEY not configured"
fi
echo

# Example 5: Production usage patterns
echo "5. Common Usage Patterns:"
echo "------------------------"
echo "# Basic cleanup (after setting SONARR_API_KEY):"
echo "export SONARR_API_KEY='your-api-key'"
echo "./refresharr"
echo
echo "# Dry run first (recommended):"
echo "./refresharr --dry-run"
echo
echo "# Process specific series:"
echo "./refresharr --series-ids '123,456,789'"
echo
echo "# Custom Sonarr instance:"
echo "./refresharr --sonarr-url 'http://192.168.1.100:8989' --sonarr-api-key 'your-key'"
echo
echo "# Debug logging for troubleshooting:"
echo "./refresharr --log-level DEBUG"
echo

echo "6. Configuration via Environment Variables:"
echo "------------------------------------------"
cat << 'EOF'
# Create a .env file or export these variables:
export SONARR_URL="http://127.0.0.1:8989"
export SONARR_API_KEY="your-api-key-here"
export REQUEST_DELAY="1s"           # Be extra nice to the API
export LOG_LEVEL="DEBUG"            # Verbose output
export DRY_RUN="true"              # Safe mode

# Then simply run:
./refresharr
EOF

echo
echo "✅ Examples complete!"
echo
echo "💡 Tips:"
echo "   - Always run with --dry-run first to preview changes"
echo "   - Use DEBUG log level if you encounter issues"  
echo "   - Set REQUEST_DELAY higher for slower/busy systems"
echo "   - Use --series-ids to test with specific series first"
