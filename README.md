# Refresharr

A modular Go service that replicates and enhances the functionality of cleaning up missing file references in *arr applications (Sonarr, Radarr, etc.).

## Overview

Refresharr addresses a common issue where *arr applications maintain database records of media files that no longer exist on disk. This can happen due to:
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

### Current
- ✅ **Sonarr Support**: Full API integration with Sonarr v3 for TV shows
- ✅ **Radarr Support**: Full API integration with Radarr v3 for movies
- ✅ **Multi-Service**: Run cleanup for both services simultaneously or individually
- ✅ **Dry Run Mode**: Preview changes before applying them
- ✅ **Detailed Logging**: Comprehensive progress reporting and statistics
- ✅ **Configurable**: Environment variables and command-line options
- ✅ **Safe Operations**: Validates connections and handles errors gracefully
- ✅ **Rate Limiting**: Configurable delays to avoid API overload
- ✅ **Selective Processing**: Process specific series or movies by ID
- ✅ **Missing Files Report**: Generate detailed JSON and terminal reports of missing files
- ✅ **Broken Symlink Detection**: Scan Radarr root directories for broken symlinks and automatically add missing movies to collection
- ✅ **Import Fixer**: Automatically resolve stuck Sonarr import issues (already imported episodes)

### Planned (Future)
- 🔄 **Web UI**: Browser-based interface for easier management
- 🔄 **Scheduling**: Automated cleanup runs via cron-like scheduling
- 🔄 **Notifications**: Discord/Slack/Email notifications for cleanup results

## Architecture

The service is designed with modularity in mind to support multiple *arr applications:

```
Refresharr/
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
   make build
   ```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SONARR_URL` | `http://127.0.0.1:8989` | Sonarr base URL (auto-set if API key provided) |
| `SONARR_API_KEY` | *(optional)* | Sonarr API key |
| `RADARR_URL` | `http://127.0.0.1:7878` | Radarr base URL (auto-set if API key provided) |
| `RADARR_API_KEY` | *(optional)* | Radarr API key |
| `REQUEST_TIMEOUT` | `30s` | HTTP request timeout |
| `REQUEST_DELAY` | `500ms` | Delay between API requests |
| `CONCURRENT_LIMIT` | `5` | Max concurrent operations |
| `LOG_LEVEL` | `INFO` | Log level (DEBUG, INFO, WARN, ERROR) |
| `DRY_RUN` | `false` | Enable dry run mode |
| `ADD_MISSING_MOVIES` | `false` | Add movies/series to collection when found from broken symlinks |
| `QUALITY_PROFILE_ID` | `12` | Quality profile ID to use when adding new movies |

**Note**: At least one service (Sonarr or Radarr) must be configured with both URL and API key.

### Getting Your API Keys

**Sonarr API Key:**
1. Open Sonarr web interface
2. Go to **Settings** → **General**
3. Copy the **API Key** value
4. Set it as `SONARR_API_KEY` environment variable

**Radarr API Key:**
1. Open Radarr web interface
2. Go to **Settings** → **General**
3. Copy the **API Key** value
4. Set it as `RADARR_API_KEY` environment variable

## Broken Symlink Detection

RefreshArr can automatically detect broken symlinks in your Radarr and Sonarr root directories and optionally add missing movies/series to your collection. Broken symlink detection always runs and reports findings, while adding media to your collection is controlled by the `ADD_MISSING_MOVIES` setting.

### How It Works

1. **Scan Root Directories**: RefreshArr fetches all configured root directories from Radarr
2. **Find Broken Symlinks**: Recursively scans for broken symlinks with movie file extensions (.mkv, .mp4, .avi, etc.)
3. **Extract TMDB ID**: Parses the TMDB ID from directory/filename (e.g., `Movie Title (2023) [tmdb-12345]`)
4. **Check Collection**: Verifies if the movie already exists in your Radarr collection
5. **Add Missing Movies**: If not in collection, adds the movie with monitoring enabled and your specified quality profile
6. **Report Results**: Includes these movies in the missing files report with an indication they were added

### Requirements

- Movie directories must include TMDB ID in the format: `Movie Title (Year) [tmdb-12345]`
- Quality profile must exist in Radarr (default ID: 12, configurable via `QUALITY_PROFILE_ID`)
- Set `ADD_MISSING_MOVIES=true` to add missing movies to collection (detection always runs)

## Usage

### Basic Usage

```bash
# Set your API keys (at least one required)
export SONARR_API_KEY="your-sonarr-api-key-here"
export RADARR_API_KEY="your-radarr-api-key-here"

# Run cleanup for all configured services (default)
./refresharr

# Fix stuck Sonarr imports (already imported issues)
./refresharr fix-imports

# Run cleanup for specific service only
./refresharr --service sonarr
./refresharr --service radarr
```

### Command Line Options

```bash
# Dry run (no changes made)
./refresharr --dry-run

# Disable terminal report output (report still saved to file)
./refresharr --no-report

# Run for both services
./refresharr --service both

# Custom instances
./refresharr --sonarr-url "http://192.168.1.100:8989" --sonarr-api-key "your-sonarr-key"
./refresharr --radarr-url "http://192.168.1.100:7878" --radarr-api-key "your-radarr-key"

# Debug logging
./refresharr --log-level DEBUG

# Process specific series or movies only
./refresharr --service sonarr --series-ids "123,456,789"
./refresharr --service radarr --movie-ids "123,456,789"

# Show help
./refresharr --help

# Show version
./refresharr --version

# Fix stuck Sonarr imports (dry run)
./refresharr fix-imports --dry-run

# Fix stuck Sonarr imports (actually remove them)
./refresharr fix-imports
```

### Fix-Imports Command

The `fix-imports` command addresses a common Sonarr issue where downloads get stuck in the queue with "already imported" or similar import errors. This typically happens when:

- Episodes are manually imported outside of Sonarr
- Files are moved or reorganized after download
- Import processes are interrupted
- Database inconsistencies occur

The command identifies stuck import items and attempts to import them before removing from the queue, ensuring no content is lost.

**Usage:**
```bash
# Preview stuck imports (recommended first step)
./refresharr fix-imports --dry-run

# Actually fix the stuck imports
./refresharr fix-imports

# Fix imports with custom Sonarr configuration
./refresharr fix-imports --sonarr-url "http://custom:8989" --sonarr-api-key "your-key"
```

**What it does:**
1. 🔍 Scans the Sonarr download queue for stuck items
2. 📋 Identifies items with import issues (status = "completed" but not imported)  
3. 🎯 Attempts to import stuck items using manual import process
4. 📥 Triggers download client scan to refresh import status
5. 📝 Logs failures without removing items from queue (for manual resolution)
6. 📊 Reports the number of items successfully imported vs requiring manual attention

**Import Issues Detected:**
- "already imported"
- "episode file already imported"  
- "one or more episodes expected"
- "missing from the release"

**Note:** This command only works with Sonarr (not Radarr) as download queue management is specific to Sonarr's import process.

### Docker Usage (Future)

```bash
# Run with Sonarr only
docker run -e SONARR_API_KEY="your-key" -e SONARR_URL="http://sonarr:8989" refresharr

# Run with Radarr only
docker run -e RADARR_API_KEY="your-key" -e RADARR_URL="http://radarr:7878" refresharr

# Run with both services
docker run \
  -e SONARR_API_KEY="your-sonarr-key" -e SONARR_URL="http://sonarr:8989" \
  -e RADARR_API_KEY="your-radarr-key" -e RADARR_URL="http://radarr:7878" \
  refresharr
```

## Missing Files Report

The service now generates comprehensive reports of missing files found during cleanup operations. Reports are automatically saved to the `reports/` directory in JSON format and displayed in the terminal in human-readable format.

### Report Features

- **JSON Export**: Detailed reports saved as timestamped JSON files in `reports/` directory
- **Terminal Display**: Human-readable summary printed to console (unless `--no-report` flag is used)
- **Dry Run Support**: Reports generated for both dry runs and actual cleanup operations
- **Detailed Information**: Includes media names, episode details, file paths, and timestamps

### Report Content

Each report includes:
- **Service Type**: Whether from Sonarr or Radarr
- **Run Type**: "dry-run" or "real-run"
- **Generation Timestamp**: When the report was created
- **Total Missing Files**: Count of missing files found
- **File Details**: For each missing file:
  - Media name (series/movie title)
  - Episode name and season/episode numbers (for TV shows)
  - Complete file path
  - Database file ID
  - Processing timestamp

### Sample Report Output

**Terminal Display:**
```
📊 MISSING FILES REPORT
==========================================
Generated: 2023-12-01T15:30:00Z
Service: sonarr
Run Type: dry-run
Total Missing Files: 3

Missing Files:
==========================================
1. Breaking Bad
   Episode: S01E02 - Cat's in the Bag...
   Missing File: /media/tv/Breaking Bad/Season 01/S01E02.mkv
   File ID: 2001
   Processed: 2023-12-01T15:30:15Z

2. The Office
   Episode: S02E05 - Halloween
   Missing File: /media/tv/The Office/Season 02/S02E05.mkv
   File ID: 2456
   Processed: 2023-12-01T15:30:32Z

3. Inception
   Missing File: /media/movies/Inception (2010)/Inception.mkv
   File ID: 3001
   Processed: 2023-12-01T15:30:45Z
==========================================
📄 Report saved to: reports/sonarr-missing-files-report-dryrun-20231201-153000.json
```

**JSON File Structure:**
```json
{
  "generatedAt": "2023-12-01T15:30:00Z",
  "runType": "dry-run",
  "serviceType": "sonarr",
  "totalMissing": 3,
  "missingFiles": [
    {
      "mediaType": "series",
      "mediaName": "Breaking Bad",
      "episodeName": "Cat's in the Bag...",
      "season": 1,
      "episode": 2,
      "filePath": "/media/tv/Breaking Bad/Season 01/S01E02.mkv",
      "fileId": 2001,
      "processedAt": "2023-12-01T15:30:15Z"
    },
    {
      "mediaType": "movie",
      "mediaName": "Inception",
      "filePath": "/media/movies/Inception (2010)/Inception.mkv",
      "fileId": 3001,
      "processedAt": "2023-12-01T15:30:45Z"
    }
  ]
}
```

## Sample Output

**Sonarr Cleanup:**
```
[INFO] Starting Refresharr v1.0.0 - Missing File Cleanup Service
[INFO] ================================================
[INFO] Starting sonarr missing file cleanup...
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

**Radarr Cleanup:**
```
[INFO] Starting radarr missing file cleanup...
[INFO] ================================================
[INFO] ✅ Successfully connected to Radarr
[INFO] Step 1: Fetching all movies...
[INFO] Found 250 movies

[INFO] Processing movie 1/250 (ID: 456)
[INFO] Movie: The Matrix
[WARN]     ❌ MISSING: /media/movies/The Matrix (1999)/The Matrix (1999).mkv
[INFO]     🗑️  Deleting movie file record 3001...
[INFO]     ✅ Successfully deleted movie file record (ID: 3001)

[INFO] ================================================
[INFO] Cleanup Summary:
[INFO]   Total items checked: 250
[INFO]   Missing files found: 12
[INFO]   Records deleted: 12
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

### Extending Support

The modular architecture makes it easy to add support for other *arr applications. The `Client` interface defines the contract that any new service must implement:

```go
type Client interface {
    GetName() string
    TestConnection(ctx context.Context) error
    
    // TV Shows (Sonarr)
    GetAllSeries(ctx context.Context) ([]models.Series, error)
    GetEpisodesForSeries(ctx context.Context, seriesID int) ([]models.Episode, error)
    GetEpisodeFile(ctx context.Context, fileID int) (*models.EpisodeFile, error)
    DeleteEpisodeFile(ctx context.Context, fileID int) error
    UpdateEpisode(ctx context.Context, episode models.Episode) error
    
    // Movies (Radarr)
    GetAllMovies(ctx context.Context) ([]models.Movie, error)
    GetMovieFile(ctx context.Context, fileID int) (*models.MovieFile, error)
    DeleteMovieFile(ctx context.Context, fileID int) error
    UpdateMovie(ctx context.Context, movie models.Movie) error
    
    TriggerRefresh(ctx context.Context) error
}
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

- [x] Radarr support
- [ ] Web UI interface  
- [ ] Docker containerization
- [ ] Automated scheduling
- [ ] Configuration file support
- [ ] Database backup before cleanup
- [ ] Webhook notifications
- [ ] Prometheus metrics
- [ ] Multi-instance support
- [ ] Lidarr support (music)
- [ ] Readarr support (books)
