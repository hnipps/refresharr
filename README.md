# RefreshArr

A modular Go service that replicates and enhances the functionality of cleaning up missing file references in *arr applications (Sonarr, Radarr, etc.).

## Overview

RefreshArr addresses a common issue where *arr applications maintain database records of media files that no longer exist on disk. This can happen due to:
- Manual file deletion
- Storage failures
- File moves/reorganization
- Network storage disconnections

The service automatically:
1. 🔍 Scans all media items that claim to have files
2. 📁 Verifies if files actually exist on the filesystem  
3. 🗑️ Removes database records for missing files
4. 🔄 Triggers a refresh to update the application status

## Features

### Current (Sonarr)
- ✅ **Sonarr Support**: Full API integration with Sonarr v3
- ✅ **Dry Run Mode**: Preview changes before applying them
- ✅ **Detailed Logging**: Comprehensive progress reporting and statistics
- ✅ **Configurable**: Environment variables and command-line options
- ✅ **Safe Operations**: Validates connections and handles errors gracefully
- ✅ **Rate Limiting**: Configurable delays to avoid API overload

### Planned (Future)
- 🔄 **Radarr Support**: Extend to work with Radarr for movies
- 🔄 **Web UI**: Browser-based interface for easier management
- 🔄 **Scheduling**: Automated cleanup runs via cron-like scheduling
- 🔄 **Notifications**: Discord/Slack/Email notifications for cleanup results

## Architecture

The service is designed with modularity in mind to support multiple *arr applications:

```
RefreshArr/
├── cmd/refresharr/          # CLI application entry point
├── internal/
│   ├── arr/                 # Core interfaces and implementations
│   │   ├── interfaces.go    # Service contracts
│   │   ├── sonarr.go       # Sonarr API client
│   │   ├── cleanup.go      # Cleanup orchestration
│   │   ├── logger.go       # Logging implementation
│   │   └── progress.go     # Progress reporting
│   ├── config/             # Configuration management
│   └── filesystem/         # File system operations
├── pkg/models/             # Shared data models
└── main.go                 # Simple entry point
```

### Key Interfaces

- **`Client`**: API client interface (Sonarr, future Radarr)
- **`CleanupService`**: Orchestrates the cleanup process
- **`FileChecker`**: Handles filesystem operations
- **`Logger`**: Structured logging interface
- **`ProgressReporter`**: User feedback and statistics

This design makes it easy to add support for Radarr by implementing the `Client` interface with Radarr-specific API calls.

## Installation

### From Source

1. **Clone the repository**:
   ```bash
   git clone https://github.com/hnipps/refresharr
   cd refresharr
   ```

2. **Build the application**:
   ```bash
   go build -o refresharr cmd/refresharr/main.go
   ```

3. **Or use the simple main.go**:
   ```bash
   go build -o refresharr-simple main.go
   ```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SONARR_URL` | `http://127.0.0.1:8989` | Sonarr base URL |
| `SONARR_API_KEY` | *(required)* | Sonarr API key |
| `REQUEST_TIMEOUT` | `30s` | HTTP request timeout |
| `REQUEST_DELAY` | `500ms` | Delay between API requests |
| `CONCURRENT_LIMIT` | `5` | Max concurrent operations |
| `LOG_LEVEL` | `INFO` | Log level (DEBUG, INFO, WARN, ERROR) |
| `DRY_RUN` | `false` | Enable dry run mode |

### Getting Your Sonarr API Key

1. Open Sonarr web interface
2. Go to **Settings** → **General**
3. Copy the **API Key** value
4. Set it as `SONARR_API_KEY` environment variable

## Usage

### Basic Usage

```bash
# Set your API key
export SONARR_API_KEY="your-api-key-here"

# Run cleanup
./refresharr
```

### Command Line Options

```bash
# Dry run (no changes made)
./refresharr --dry-run

# Custom Sonarr instance
./refresharr --sonarr-url "http://192.168.1.100:8989" --sonarr-api-key "your-key"

# Debug logging
./refresharr --log-level DEBUG

# Process specific series only
./refresharr --series-ids "123,456,789"

# Show help
./refresharr --help

# Show version
./refresharr --version
```

### Docker Usage (Future)

```bash
docker run -e SONARR_API_KEY="your-key" -e SONARR_URL="http://sonarr:8989" refresharr
```

## Sample Output

```
[INFO] Starting RefreshArr v1.0.0 - Missing File Cleanup Service
[INFO] ================================================
[INFO] ✅ Successfully connected to Sonarr
[INFO] Step 1: Fetching all series...
[INFO] Found 15 series

[INFO] Processing series 1/15 (ID: 123)
[INFO] Series: Breaking Bad
[INFO]   Checking S1E1 (Episode ID: 1001)
[INFO]     ✅ File exists: /media/tv/Breaking Bad/Season 01/S01E01.mkv
[INFO]   Checking S1E2 (Episode ID: 1002)
[WARN]     ❌ MISSING: /media/tv/Breaking Bad/Season 01/S01E02.mkv
[INFO]     🗑️  Deleting episode file record 2001...
[INFO]     ✅ Successfully deleted episode file record (ID: 2001)

[INFO] ================================================
[INFO] Cleanup Summary:
[INFO]   Total items checked: 342
[INFO]   Missing files found: 5
[INFO]   Records deleted: 5
[INFO] 
[INFO] 🔄 Triggering refresh to update status...
[INFO] ✅ Refresh triggered successfully
[INFO] 🎉 Cleanup completed successfully!
```

## Development

### Project Structure

The codebase follows Go best practices with clear separation of concerns:

- **`cmd/`**: Application entry points
- **`internal/`**: Private application code
- **`pkg/`**: Public library code (models)
- **`main.go`**: Simple entry point for basic usage

### Adding Radarr Support

To add Radarr support, implement the `Client` interface:

```go
type RadarrClient struct {
    // ... implementation
}

func (c *RadarrClient) GetName() string {
    return "radarr"
}

func (c *RadarrClient) GetAllMovies(ctx context.Context) ([]models.Movie, error) {
    // Implement Radarr movie fetching
}

// ... other interface methods
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Verbose output
go test -v ./...
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## Original Shell Script

This Go service replicates the functionality of `tmp/sonarr-delete-missing.sh`. The original script:
- Used bash with curl and jq
- Was Sonarr-specific
- Required manual execution
- Had limited error handling

The Go rewrite provides:
- ✅ Better error handling and logging
- ✅ Modular, extensible architecture
- ✅ Cross-platform compatibility
- ✅ Configuration management
- ✅ Dry run capabilities
- ✅ Progress reporting

## License

[Add your license here]

## Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/hnipps/refresharr/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/hnipps/refresharr/discussions)
- 📚 **Wiki**: [GitHub Wiki](https://github.com/hnipps/refresharr/wiki)

## Roadmap

- [ ] Radarr support
- [ ] Web UI interface  
- [ ] Docker containerization
- [ ] Automated scheduling
- [ ] Configuration file support
- [ ] Database backup before cleanup
- [ ] Webhook notifications
- [ ] Prometheus metrics
- [ ] Multi-instance support
