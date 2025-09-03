package models

import (
	"fmt"
	"regexp"
	"strconv"
)

// MediaItem represents a base media item (can be extended for TV shows or movies)
type MediaItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path,omitempty"`
}

// Series represents a TV series in Sonarr
type Series struct {
	MediaItem
	SeasonCount int `json:"seasonCount,omitempty"`
	// Extended fields for TVDB and monitoring (similar to Movie fields)
	TVDBID           int    `json:"tvdbId,omitempty"`
	Monitored        bool   `json:"monitored"`
	QualityProfileID int    `json:"qualityProfileId,omitempty"`
	RootFolderPath   string `json:"rootFolderPath,omitempty"`
}

// Movie represents a movie in Radarr
type Movie struct {
	MediaItem
	Year        int  `json:"year,omitempty"`
	HasFile     bool `json:"hasFile"`
	MovieFileID *int `json:"movieFileId,omitempty"`
	// Extended fields for TMDB and monitoring
	TMDBID           int    `json:"tmdbId,omitempty"`
	Monitored        bool   `json:"monitored"`
	QualityProfileID int    `json:"qualityProfileId,omitempty"`
	RootFolderPath   string `json:"rootFolderPath,omitempty"`
}

// Episode represents a TV episode
type Episode struct {
	ID            int    `json:"id"`
	SeriesID      int    `json:"seriesId"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	HasFile       bool   `json:"hasFile"`
	EpisodeFileID *int   `json:"episodeFileId,omitempty"`
}

// EpisodeFile represents a file associated with an episode
type EpisodeFile struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// MovieFile represents a file associated with a movie (for future Radarr support)
type MovieFile struct {
	ID      int    `json:"id"`
	Path    string `json:"path"`
	MovieID int    `json:"movieId"`
}

// RootFolder represents a Radarr root folder configuration
type RootFolder struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// QualityProfile represents a Radarr quality profile
type QualityProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MovieLookup represents a movie lookup result from TMDB
type MovieLookup struct {
	TMDBID   int    `json:"tmdbId"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Overview string `json:"overview,omitempty"`
	Images   []struct {
		CoverType string `json:"coverType"`
		URL       string `json:"url"`
	} `json:"images,omitempty"`
}

// SeriesLookup represents a series lookup result from TVDB
type SeriesLookup struct {
	TVDBID   int    `json:"tvdbId"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Overview string `json:"overview,omitempty"`
	Images   []struct {
		CoverType string `json:"coverType"`
		URL       string `json:"url"`
	} `json:"images,omitempty"`
}

// CleanupStats tracks cleanup operation statistics
type CleanupStats struct {
	TotalItemsChecked int
	MissingFiles      int
	DeletedRecords    int
	Errors            int
}

// MissingFileEntry represents a single missing file entry in the report
type MissingFileEntry struct {
	MediaType         string `json:"mediaType"`                   // "movie" or "series"
	MediaName         string `json:"mediaName"`                   // Movie title or series title
	EpisodeName       string `json:"episodeName,omitempty"`       // Episode name (only for series)
	Season            *int   `json:"season,omitempty"`            // Season number (only for series)
	Episode           *int   `json:"episode,omitempty"`           // Episode number (only for series)
	FilePath          string `json:"filePath"`                    // Path to the missing file
	FileID            int    `json:"fileId"`                      // File ID in the database
	ProcessedAt       string `json:"processedAt"`                 // Timestamp when processed
	AddedToCollection bool   `json:"addedToCollection,omitempty"` // Whether the movie/series was added to the collection
	TMDBID            int    `json:"tmdbId,omitempty"`            // TMDB ID for movies
	TVDBID            int    `json:"tvdbId,omitempty"`            // TVDB ID for series
}

// MissingFilesReport represents a complete missing files report
type MissingFilesReport struct {
	GeneratedAt  string             `json:"generatedAt"`
	RunType      string             `json:"runType"`     // "dry-run" or "real-run"
	ServiceType  string             `json:"serviceType"` // "sonarr" or "radarr"
	TotalMissing int                `json:"totalMissing"`
	MissingFiles []MissingFileEntry `json:"missingFiles"`
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	Stats    CleanupStats
	Messages []string
	Success  bool
	Report   *MissingFilesReport `json:"report,omitempty"` // Optional report data
}

// ParseTMDBIDFromPath extracts TMDB ID from a file path
// Expected format: ...path.../Movie Title (Year) [tmdb-12345]/...
func ParseTMDBIDFromPath(filePath string) (int, error) {
	// Use regex to find tmdb-### pattern
	re := regexp.MustCompile(`\[tmdb-(\d+)\]`)
	matches := re.FindStringSubmatch(filePath)

	if len(matches) < 2 {
		return 0, fmt.Errorf("TMDB ID not found in path: %s", filePath)
	}

	tmdbID, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid TMDB ID format in path %s: %w", filePath, err)
	}

	return tmdbID, nil
}

// QueueItem represents an item in the download queue
type QueueItem struct {
	ID             int             `json:"id"`
	Title          string          `json:"title"`
	Series         *Series         `json:"series,omitempty"`
	Status         string          `json:"status"`
	StatusMessages []StatusMessage `json:"statusMessages,omitempty"`
	ErrorMessage   string          `json:"errorMessage,omitempty"`
	Size           int64           `json:"size,omitempty"`
}

// StatusMessage represents a status message in the queue
type StatusMessage struct {
	Title    string   `json:"title"`
	Messages []string `json:"messages,omitempty"`
}

// QueueResponse represents the API response from the queue endpoint
type QueueResponse struct {
	Records []QueueItem `json:"records"`
}

// ImportFixResult represents the result of an import fix operation
type ImportFixResult struct {
	TotalStuckItems int
	FixedItems      int
	Errors          []string
	Success         bool
	DryRun          bool
}

// Expected format: ...path.../Series Title (Year) [tvdb-12345]/...
func ParseTVDBIDFromPath(filePath string) (int, error) {
	// Use regex to find tvdb-### pattern
	re := regexp.MustCompile(`\[tvdb-(\d+)\]`)
	matches := re.FindStringSubmatch(filePath)

	if len(matches) < 2 {
		return 0, fmt.Errorf("TVDB ID not found in path: %s", filePath)
	}

	tvdbID, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid TVDB ID format in path %s: %w", filePath, err)
	}

	return tvdbID, nil
}

// ParseTVDBIDFromPath extracts TVDB ID from a file path
