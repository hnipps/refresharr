package arr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hnipps/refresharr/pkg/models"
)

// CleanupServiceImpl implements the CleanupService interface
type CleanupServiceImpl struct {
	client           Client
	fileChecker      FileChecker
	logger           Logger
	progressReporter ProgressReporter
	requestDelay     time.Duration
	dryRun           bool
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(
	client Client,
	fileChecker FileChecker,
	logger Logger,
	progressReporter ProgressReporter,
	requestDelay time.Duration,
	dryRun bool,
) CleanupService {
	return &CleanupServiceImpl{
		client:           client,
		fileChecker:      fileChecker,
		logger:           logger,
		progressReporter: progressReporter,
		requestDelay:     requestDelay,
		dryRun:           dryRun,
	}
}

// CleanupMissingFiles performs cleanup for all series or movies based on client type
func (s *CleanupServiceImpl) CleanupMissingFiles(ctx context.Context) (*models.CleanupResult, error) {
	s.logger.Info("Starting %s missing file cleanup...", s.client.GetName())
	s.logger.Info("================================================")

	if s.dryRun {
		s.logger.Info("🏃 DRY RUN MODE: No changes will be made")
		s.logger.Info("")
	}

	// Test connection first
	if err := s.client.TestConnection(ctx); err != nil {
		return nil, fmt.Errorf("connection test failed: %w", err)
	}

	// Handle based on client type
	if s.client.GetName() == "sonarr" {
		// Get all series
		s.logger.Info("Step 1: Fetching all series...")
		series, err := s.client.GetAllSeries(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch series: %w", err)
		}

		if len(series) == 0 {
			s.logger.Info("No series found")
			return &models.CleanupResult{
				Stats:   models.CleanupStats{},
				Success: true,
			}, nil
		}

		s.logger.Info("Found %d series", len(series))

		// Extract series IDs
		var seriesIDs []int
		for _, s := range series {
			seriesIDs = append(seriesIDs, s.ID)
		}

		// Cleanup specific series
		return s.CleanupMissingFilesForSeries(ctx, seriesIDs)
	} else if s.client.GetName() == "radarr" {
		// Get all movies
		s.logger.Info("Step 1: Fetching all movies...")
		movies, err := s.client.GetAllMovies(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch movies: %w", err)
		}

		if len(movies) == 0 {
			s.logger.Info("No movies found")
			return &models.CleanupResult{
				Stats:   models.CleanupStats{},
				Success: true,
			}, nil
		}

		s.logger.Info("Found %d movies", len(movies))

		// Extract movie IDs
		var movieIDs []int
		for _, m := range movies {
			movieIDs = append(movieIDs, m.ID)
		}

		// Cleanup specific movies
		return s.CleanupMissingFilesForMovies(ctx, movieIDs)
	}

	return nil, fmt.Errorf("unsupported client type: %s", s.client.GetName())
}

// CleanupMissingFilesForSeries performs cleanup for specific series
func (s *CleanupServiceImpl) CleanupMissingFilesForSeries(ctx context.Context, seriesIDs []int) (*models.CleanupResult, error) {
	stats := models.CleanupStats{}
	var messages []string
	var mu sync.Mutex

	seriesCount := len(seriesIDs)

	// Process each series
	for i, seriesID := range seriesIDs {
		select {
		case <-ctx.Done():
			s.logger.Warn("Cleanup cancelled")
			return &models.CleanupResult{
				Stats:    stats,
				Messages: messages,
				Success:  false,
			}, ctx.Err()
		default:
		}

		// Get series details for better logging
		seriesName := fmt.Sprintf("Series %d", seriesID)
		// Note: We could fetch the series name here, but it's an extra API call
		// For simplicity, we'll use the ID in reporting

		s.progressReporter.StartSeries(seriesID, seriesName, i+1, seriesCount)

		seriesStats, err := s.cleanupSeries(ctx, seriesID)
		if err != nil {
			s.logger.Error("Error processing series %d: %s", seriesID, err.Error())
			s.progressReporter.ReportError(err)

			mu.Lock()
			stats.Errors++
			messages = append(messages, fmt.Sprintf("Error processing series %d: %s", seriesID, err.Error()))
			mu.Unlock()

			continue
		}

		// Aggregate stats
		mu.Lock()
		stats.TotalItemsChecked += seriesStats.TotalItemsChecked
		stats.MissingFiles += seriesStats.MissingFiles
		stats.DeletedRecords += seriesStats.DeletedRecords
		stats.Errors += seriesStats.Errors
		mu.Unlock()

		// Add delay between series to be nice to the API
		if i < seriesCount-1 && s.requestDelay > 0 {
			time.Sleep(s.requestDelay)
		}
	}

	// Report final statistics
	s.progressReporter.Finish(stats)

	// Trigger refresh if we deleted any records
	if stats.DeletedRecords > 0 && !s.dryRun {
		if err := s.client.TriggerRefresh(ctx); err != nil {
			s.logger.Warn("Failed to trigger refresh: %s", err.Error())
			messages = append(messages, fmt.Sprintf("Failed to trigger refresh: %s", err.Error()))
		}
	}

	return &models.CleanupResult{
		Stats:    stats,
		Messages: messages,
		Success:  stats.Errors == 0,
	}, nil
}

// CleanupMissingFilesForMovies performs cleanup for specific movies
func (s *CleanupServiceImpl) CleanupMissingFilesForMovies(ctx context.Context, movieIDs []int) (*models.CleanupResult, error) {
	stats := models.CleanupStats{}
	var messages []string
	var mu sync.Mutex

	movieCount := len(movieIDs)

	// Process each movie
	for i, movieID := range movieIDs {
		select {
		case <-ctx.Done():
			s.logger.Warn("Cleanup cancelled")
			return &models.CleanupResult{
				Stats:    stats,
				Messages: messages,
				Success:  false,
			}, ctx.Err()
		default:
		}

		// Get movie details for better logging
		movieName := fmt.Sprintf("Movie %d", movieID)
		// Note: We could fetch the movie name here, but it's an extra API call
		// For simplicity, we'll use the ID in reporting

		s.progressReporter.StartMovie(movieID, movieName, i+1, movieCount)

		movieStats, err := s.cleanupMovie(ctx, movieID)
		if err != nil {
			s.logger.Error("Error processing movie %d: %s", movieID, err.Error())
			s.progressReporter.ReportError(err)

			mu.Lock()
			stats.Errors++
			messages = append(messages, fmt.Sprintf("Error processing movie %d: %s", movieID, err.Error()))
			mu.Unlock()

			continue
		}

		// Aggregate stats
		mu.Lock()
		stats.TotalItemsChecked += movieStats.TotalItemsChecked
		stats.MissingFiles += movieStats.MissingFiles
		stats.DeletedRecords += movieStats.DeletedRecords
		stats.Errors += movieStats.Errors
		mu.Unlock()

		// Add delay between movies to be nice to the API
		if i < movieCount-1 && s.requestDelay > 0 {
			time.Sleep(s.requestDelay)
		}
	}

	// Report final statistics
	s.progressReporter.Finish(stats)

	// Trigger refresh if we deleted any records
	if stats.DeletedRecords > 0 && !s.dryRun {
		if err := s.client.TriggerRefresh(ctx); err != nil {
			s.logger.Warn("Failed to trigger refresh: %s", err.Error())
			messages = append(messages, fmt.Sprintf("Failed to trigger refresh: %s", err.Error()))
		}
	}

	return &models.CleanupResult{
		Stats:    stats,
		Messages: messages,
		Success:  stats.Errors == 0,
	}, nil
}

// cleanupSeries processes a single series
func (s *CleanupServiceImpl) cleanupSeries(ctx context.Context, seriesID int) (models.CleanupStats, error) {
	stats := models.CleanupStats{}

	// Get episodes for this series
	s.logger.Debug("Fetching episodes for series %d...", seriesID)
	episodes, err := s.client.GetEpisodesForSeries(ctx, seriesID)
	if err != nil {
		return stats, fmt.Errorf("failed to get episodes for series %d: %w", seriesID, err)
	}

	if len(episodes) == 0 {
		s.logger.Debug("  No episodes found for series %d", seriesID)
		return stats, nil
	}

	// Process episodes that claim to have files
	for _, episode := range episodes {
		if !episode.HasFile || episode.EpisodeFileID == nil {
			continue
		}

		stats.TotalItemsChecked++
		s.progressReporter.StartEpisode(episode.ID, episode.SeasonNumber, episode.EpisodeNumber)

		// Get episode file details
		episodeFile, err := s.client.GetEpisodeFile(ctx, *episode.EpisodeFileID)
		if err != nil {
			s.logger.Warn("    ⚠️  Failed to get episode file %d: %s", *episode.EpisodeFileID, err.Error())
			stats.Errors++
			continue
		}

		// Check if file exists
		if episodeFile.Path == "" {
			s.logger.Warn("    ⚠️  No file path found for episode file %d", *episode.EpisodeFileID)
			continue
		}

		if s.fileChecker.FileExists(episodeFile.Path) {
			s.logger.Debug("    ✅ File exists: %s", episodeFile.Path)
			continue
		}

		// File is missing
		stats.MissingFiles++
		s.progressReporter.ReportMissingFile(episodeFile.Path)

		if s.dryRun {
			s.logger.Info("    🏃 DRY RUN: Would delete episode file record %d", *episode.EpisodeFileID)
			continue
		}

		// Delete the episode file record
		s.logger.Info("    🗑️  Deleting episode file record %d...", *episode.EpisodeFileID)
		if err := s.client.DeleteEpisodeFile(ctx, *episode.EpisodeFileID); err != nil {
			s.logger.Error("    ❌ Failed to delete episode file record %d: %s", *episode.EpisodeFileID, err.Error())
			s.progressReporter.ReportError(err)
			stats.Errors++
			continue
		}

		stats.DeletedRecords++
		s.progressReporter.ReportDeletedRecord(*episode.EpisodeFileID)

		// Update episode status
		s.logger.Debug("    🔄 Updating episode status...")
		if err := s.client.UpdateEpisode(ctx, episode); err != nil {
			s.logger.Warn("    ⚠️  Failed to update episode %d: %s", episode.ID, err.Error())
			// This is not critical, so we continue
		}

		// Small delay between operations
		if s.requestDelay > 0 {
			time.Sleep(s.requestDelay)
		}
	}

	return stats, nil
}

// cleanupMovie processes a single movie
func (s *CleanupServiceImpl) cleanupMovie(ctx context.Context, movieID int) (models.CleanupStats, error) {
	stats := models.CleanupStats{}

	// Get all movies to find the specific one
	s.logger.Debug("Fetching movies to find movie %d...", movieID)
	movies, err := s.client.GetAllMovies(ctx)
	if err != nil {
		return stats, fmt.Errorf("failed to get movies: %w", err)
	}

	// Find the specific movie
	var targetMovie *models.Movie
	for _, movie := range movies {
		if movie.ID == movieID {
			targetMovie = &movie
			break
		}
	}

	if targetMovie == nil {
		return stats, fmt.Errorf("movie %d not found", movieID)
	}

	// Check if movie has a file
	if !targetMovie.HasFile || targetMovie.MovieFileID == nil {
		s.logger.Debug("  Movie %d has no file reference", movieID)
		return stats, nil
	}

	stats.TotalItemsChecked++

	// Get movie file details
	movieFile, err := s.client.GetMovieFile(ctx, *targetMovie.MovieFileID)
	if err != nil {
		s.logger.Warn("    ⚠️  Failed to get movie file %d: %s", *targetMovie.MovieFileID, err.Error())
		stats.Errors++
		return stats, nil
	}

	// Check if file exists
	if movieFile.Path == "" {
		s.logger.Warn("    ⚠️  No file path found for movie file %d", *targetMovie.MovieFileID)
		return stats, nil
	}

	if s.fileChecker.FileExists(movieFile.Path) {
		s.logger.Debug("    ✅ File exists: %s", movieFile.Path)
		return stats, nil
	}

	// File is missing
	stats.MissingFiles++
	s.progressReporter.ReportMissingFile(movieFile.Path)

	if s.dryRun {
		s.logger.Info("    🏃 DRY RUN: Would delete movie file record %d", *targetMovie.MovieFileID)
		return stats, nil
	}

	// Delete the movie file record
	s.logger.Info("    🗑️  Deleting movie file record %d...", *targetMovie.MovieFileID)
	if err := s.client.DeleteMovieFile(ctx, *targetMovie.MovieFileID); err != nil {
		s.logger.Error("    ❌ Failed to delete movie file record %d: %s", *targetMovie.MovieFileID, err.Error())
		s.progressReporter.ReportError(err)
		stats.Errors++
		return stats, nil
	}

	stats.DeletedRecords++
	s.progressReporter.ReportDeletedRecord(*targetMovie.MovieFileID)

	// Update movie status
	s.logger.Debug("    🔄 Updating movie status...")
	if err := s.client.UpdateMovie(ctx, *targetMovie); err != nil {
		s.logger.Warn("    ⚠️  Failed to update movie %d: %s", targetMovie.ID, err.Error())
		// This is not critical, so we continue
	}

	// Small delay between operations
	if s.requestDelay > 0 {
		time.Sleep(s.requestDelay)
	}

	return stats, nil
}
