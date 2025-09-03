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

	// GetMovie returns a single movie by ID (Radarr specific)
	GetMovie(ctx context.Context, movieID int) (*models.Movie, error)

	// GetEpisodesForSeries returns all episodes for a given series
	GetEpisodesForSeries(ctx context.Context, seriesID int) ([]models.Episode, error)

	// GetEpisodeFile returns episode file details
	GetEpisodeFile(ctx context.Context, fileID int) (*models.EpisodeFile, error)

	// DeleteEpisodeFile deletes an episode file record
	DeleteEpisodeFile(ctx context.Context, fileID int) error

	// UpdateEpisode updates an episode's metadata
	UpdateEpisode(ctx context.Context, episode models.Episode) error

	// GetMovieFile returns movie file details (Radarr specific)
	GetMovieFile(ctx context.Context, fileID int) (*models.MovieFile, error)

	// DeleteMovieFile deletes a movie file record (Radarr specific)
	DeleteMovieFile(ctx context.Context, fileID int) error

	// UpdateMovie updates a movie's metadata (Radarr specific)
	UpdateMovie(ctx context.Context, movie models.Movie) error

	// TriggerRefresh triggers a refresh/rescan operation
	// GetRootFolders returns all root folders (Radarr specific)
	GetRootFolders(ctx context.Context) ([]models.RootFolder, error)

	// GetQualityProfiles returns all quality profiles
	GetQualityProfiles(ctx context.Context) ([]models.QualityProfile, error)

	// LookupMovieByTMDBID looks up movie information by TMDB ID
	LookupMovieByTMDBID(ctx context.Context, tmdbID int) (*models.MovieLookup, error)

	// AddMovie adds a movie to the collection
	AddMovie(ctx context.Context, movie models.Movie) (*models.Movie, error)

	// GetMovieByTMDBID returns a movie by TMDB ID if it exists in the collection
	GetMovieByTMDBID(ctx context.Context, tmdbID int) (*models.Movie, error)

	// GetSeriesByTVDBID returns a series by TVDB ID if it exists in the collection (Sonarr specific)
	GetSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.Series, error)

	// LookupSeriesByTVDBID looks up series information by TVDB ID (Sonarr specific)
	LookupSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.SeriesLookup, error)

	// AddSeries adds a series to the collection (Sonarr specific)
	AddSeries(ctx context.Context, series models.Series) (*models.Series, error)

	TriggerRefresh(ctx context.Context) error

	// Queue management methods (primarily for Sonarr import fixing)
	GetQueue(ctx context.Context) ([]models.QueueItem, error)
	GetQueueDetails(ctx context.Context, queueID int) (*models.QueueItem, error)
	RemoveFromQueue(ctx context.Context, queueID int, removeFromClient bool) error
	
	// Manual import methods for importing downloaded files
	TriggerDownloadClientScan(ctx context.Context) error
	GetManualImport(ctx context.Context, folder string) ([]models.ManualImportItem, error)
	ExecuteManualImport(ctx context.Context, files []models.ManualImportItem, importMode string) error
}

// FileChecker defines the interface for file system operations
type FileChecker interface {
	FileExists(path string) bool
	IsReadable(path string) bool
	FindBrokenSymlinks(rootDir string, extensions []string) ([]string, error)
	IsSymlink(path string) bool
	DeleteSymlink(path string) error
}

// CleanupService defines the interface for cleanup operations
type CleanupService interface {
	// CleanupMissingFiles performs the cleanup operation
	CleanupMissingFiles(ctx context.Context) (*models.CleanupResult, error)

	// CleanupMissingFilesForSeries performs cleanup for specific series
	CleanupMissingFilesForSeries(ctx context.Context, seriesIDs []int) (*models.CleanupResult, error)

	// CleanupMissingFilesForMovies performs cleanup for specific movies
	CleanupMissingFilesForMovies(ctx context.Context, movieIDs []int) (*models.CleanupResult, error)
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
	StartMovie(movieID int, movieName string, current, total int)
	ReportMissingFile(filePath string)
	ReportDeletedRecord(fileID int)
	ReportDeletedEpisodeRecord(fileID int)
	ReportDeletedMovieRecord(fileID int)
	ReportError(err error)
	Finish(stats models.CleanupStats)
}
