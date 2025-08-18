package arr

import (
	"context"
	"github.com/hnipps/refresharr/pkg/models"
)

// Client defines the interface for *arr API clients (Sonarr, Radarr, etc.)
type Client interface {
	// GetName returns the name of the service (e.g., "sonarr", "radarr")
	GetName() string

	// TestConnection verifies the connection to the *arr instance
	TestConnection(ctx context.Context) error

	// GetAllSeries returns all series (Sonarr specific)
	GetAllSeries(ctx context.Context) ([]models.Series, error)

	// GetAllMovies returns all movies (Radarr specific - can be nil for Sonarr)
	GetAllMovies(ctx context.Context) ([]models.Movie, error)

	// GetEpisodesForSeries returns all episodes for a given series
	GetEpisodesForSeries(ctx context.Context, seriesID int) ([]models.Episode, error)

	// GetEpisodeFile returns episode file details
	GetEpisodeFile(ctx context.Context, fileID int) (*models.EpisodeFile, error)

	// DeleteEpisodeFile deletes an episode file record
	DeleteEpisodeFile(ctx context.Context, fileID int) error

	// UpdateEpisode updates an episode's metadata
	UpdateEpisode(ctx context.Context, episode models.Episode) error

	// TriggerRefresh triggers a refresh/rescan operation
	TriggerRefresh(ctx context.Context) error
}

// FileChecker defines the interface for file system operations
type FileChecker interface {
	FileExists(path string) bool
	IsReadable(path string) bool
}

// CleanupService defines the interface for cleanup operations
type CleanupService interface {
	// CleanupMissingFiles performs the cleanup operation
	CleanupMissingFiles(ctx context.Context) (*models.CleanupResult, error)

	// CleanupMissingFilesForSeries performs cleanup for specific series
	CleanupMissingFilesForSeries(ctx context.Context, seriesIDs []int) (*models.CleanupResult, error)
}

// Logger defines the interface for logging operations
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// ProgressReporter defines the interface for progress reporting
type ProgressReporter interface {
	StartSeries(seriesID int, seriesName string, current, total int)
	StartEpisode(episodeID int, seasonNum, episodeNum int)
	ReportMissingFile(filePath string)
	ReportDeletedRecord(fileID int)
	ReportError(err error)
	Finish(stats models.CleanupStats)
}
