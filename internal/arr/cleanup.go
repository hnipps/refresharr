package arr

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hnipps/refresharr/pkg/models"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CleanupServiceImpl implements the CleanupService interface
type CleanupServiceImpl struct {
	client           Client
	fileChecker      FileChecker
	logger           Logger
	progressReporter ProgressReporter
	requestDelay     time.Duration
	concurrentLimit  int
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
		concurrentLimit:  5, // Default value, will be updated by NewCleanupServiceWithConcurrency
		dryRun:           dryRun,
	}
}

// NewCleanupServiceWithConcurrency creates a new cleanup service with configurable concurrency
func NewCleanupServiceWithConcurrency(
	client Client,
	fileChecker FileChecker,
	logger Logger,
	progressReporter ProgressReporter,
	requestDelay time.Duration,
	concurrentLimit int,
	dryRun bool,
) CleanupService {
	return &CleanupServiceImpl{
		client:           client,
		fileChecker:      fileChecker,
		logger:           logger,
		progressReporter: progressReporter,
		requestDelay:     requestDelay,
		concurrentLimit:  concurrentLimit,
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

// CleanupMissingFilesForSeries performs cleanup for specific series using concurrent processing
func (s *CleanupServiceImpl) CleanupMissingFilesForSeries(ctx context.Context, seriesIDs []int) (*models.CleanupResult, error) {
	stats := models.CleanupStats{}
	var messages []string
	var mu sync.Mutex

	seriesCount := len(seriesIDs)
	s.logger.Info("Processing %d series with concurrency limit of %d", seriesCount, s.concurrentLimit)

	// Create worker pool for concurrent processing
	semaphore := make(chan struct{}, s.concurrentLimit)
	var wg sync.WaitGroup

	// Channel for collecting results
	type seriesResult struct {
		seriesID int
		stats    models.CleanupStats
		err      error
	}
	resultsChan := make(chan seriesResult, seriesCount)

	// Process each series concurrently
	for i, seriesID := range seriesIDs {
		wg.Add(1)
		go func(seriesID, index int) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			select {
			case <-ctx.Done():
				resultsChan <- seriesResult{seriesID: seriesID, err: ctx.Err()}
				return
			default:
			}

			// Get series details for better logging
			seriesName := fmt.Sprintf("Series %d", seriesID)
			s.progressReporter.StartSeries(seriesID, seriesName, index+1, seriesCount)

			seriesStats, err := s.cleanupSeries(ctx, seriesID)
			resultsChan <- seriesResult{
				seriesID: seriesID,
				stats:    seriesStats,
				err:      err,
			}

			// Add delay after processing to be nice to the API
			if s.requestDelay > 0 {
				time.Sleep(s.requestDelay)
			}
		}(seriesID, i)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	processedCount := 0
	for result := range resultsChan {
		processedCount++

		if result.err != nil {
			if result.err == ctx.Err() {
				s.logger.Warn("Cleanup cancelled")
				return &models.CleanupResult{
					Stats:    stats,
					Messages: messages,
					Success:  false,
				}, result.err
			}

			s.logger.Error("Error processing series %d: %s", result.seriesID, result.err.Error())
			s.progressReporter.ReportError(result.err)

			mu.Lock()
			stats.Errors++
			messages = append(messages, fmt.Sprintf("Error processing series %d: %s", result.seriesID, result.err.Error()))
			mu.Unlock()
			continue
		}

		// Aggregate stats
		mu.Lock()
		stats.TotalItemsChecked += result.stats.TotalItemsChecked
		stats.MissingFiles += result.stats.MissingFiles
		stats.DeletedRecords += result.stats.DeletedRecords
		stats.Errors += result.stats.Errors
		mu.Unlock()
	}

	s.logger.Info("Completed processing %d series", processedCount)

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

// CleanupMissingFilesForMovies performs cleanup for specific movies using concurrent processing
func (s *CleanupServiceImpl) CleanupMissingFilesForMovies(ctx context.Context, movieIDs []int) (*models.CleanupResult, error) {
	stats := models.CleanupStats{}
	var messages []string
	var mu sync.Mutex

	movieCount := len(movieIDs)
	s.logger.Info("Processing %d movies with concurrency limit of %d", movieCount, s.concurrentLimit)

	// Create worker pool for concurrent processing
	semaphore := make(chan struct{}, s.concurrentLimit)
	var wg sync.WaitGroup

	// Channel for collecting results
	type movieResult struct {
		movieID int
		stats   models.CleanupStats
		err     error
	}
	resultsChan := make(chan movieResult, movieCount)

	// Process each movie concurrently
	for i, movieID := range movieIDs {
		wg.Add(1)
		go func(movieID, index int) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			select {
			case <-ctx.Done():
				resultsChan <- movieResult{movieID: movieID, err: ctx.Err()}
				return
			default:
			}

			// Get movie details for better logging
			movieName := fmt.Sprintf("Movie %d", movieID)
			s.progressReporter.StartMovie(movieID, movieName, index+1, movieCount)

			movieStats, err := s.cleanupMovie(ctx, movieID)
			resultsChan <- movieResult{
				movieID: movieID,
				stats:   movieStats,
				err:     err,
			}

			// Add delay after processing to be nice to the API
			if s.requestDelay > 0 {
				time.Sleep(s.requestDelay)
			}
		}(movieID, i)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	processedCount := 0
	for result := range resultsChan {
		processedCount++

		if result.err != nil {
			if result.err == ctx.Err() {
				s.logger.Warn("Cleanup cancelled")
				return &models.CleanupResult{
					Stats:    stats,
					Messages: messages,
					Success:  false,
				}, result.err
			}

			s.logger.Error("Error processing movie %d: %s", result.movieID, result.err.Error())
			s.progressReporter.ReportError(result.err)

			mu.Lock()
			stats.Errors++
			messages = append(messages, fmt.Sprintf("Error processing movie %d: %s", result.movieID, result.err.Error()))
			mu.Unlock()
			continue
		}

		// Aggregate stats
		mu.Lock()
		stats.TotalItemsChecked += result.stats.TotalItemsChecked
		stats.MissingFiles += result.stats.MissingFiles
		stats.DeletedRecords += result.stats.DeletedRecords
		stats.Errors += result.stats.Errors
		mu.Unlock()
	}

	s.logger.Info("Completed processing %d movies", processedCount)

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

	// Process episodes that claim to have files concurrently
	episodesWithFiles := make([]models.Episode, 0)
	for _, episode := range episodes {
		if episode.HasFile && episode.EpisodeFileID != nil {
			episodesWithFiles = append(episodesWithFiles, episode)
		}
	}

	if len(episodesWithFiles) == 0 {
		return stats, nil
	}

	// Use a smaller concurrency limit for episodes within a series to avoid overwhelming the API
	episodeConcurrency := min(s.concurrentLimit, 3)
	episodeSemaphore := make(chan struct{}, episodeConcurrency)
	var episodeWg sync.WaitGroup
	var episodeMu sync.Mutex

	// Channel for collecting episode results
	type episodeResult struct {
		episode models.Episode
		stats   models.CleanupStats
		err     error
	}
	episodeResultsChan := make(chan episodeResult, len(episodesWithFiles))

	// Process episodes concurrently
	for _, episode := range episodesWithFiles {
		episodeWg.Add(1)
		go func(ep models.Episode) {
			defer episodeWg.Done()

			// Acquire semaphore slot
			episodeSemaphore <- struct{}{}
			defer func() { <-episodeSemaphore }()

			select {
			case <-ctx.Done():
				episodeResultsChan <- episodeResult{episode: ep, err: ctx.Err()}
				return
			default:
			}

			episodeStats := models.CleanupStats{TotalItemsChecked: 1}
			s.progressReporter.StartEpisode(ep.ID, ep.SeasonNumber, ep.EpisodeNumber)

			// Get episode file details
			episodeFile, err := s.client.GetEpisodeFile(ctx, *ep.EpisodeFileID)
			if err != nil {
				// If episode file is not found, it might have been already deleted
				// This is not an error condition - just skip this episode
				if strings.Contains(strings.ToLower(err.Error()), "not found") {
					s.logger.Info("    ℹ️  Episode file %d already deleted or not found", *ep.EpisodeFileID)
					episodeResultsChan <- episodeResult{episode: ep, stats: episodeStats, err: nil}
					return
				}
				s.logger.Warn("    ⚠️  Failed to get episode file %d: %s", *ep.EpisodeFileID, err.Error())
				episodeStats.Errors++
				episodeResultsChan <- episodeResult{episode: ep, stats: episodeStats, err: nil}
				return
			}

			// Check if file exists
			if episodeFile.Path == "" {
				s.logger.Warn("    ⚠️  No file path found for episode file %d", *ep.EpisodeFileID)
				episodeResultsChan <- episodeResult{episode: ep, stats: episodeStats, err: nil}
				return
			}

			if s.fileChecker.FileExists(episodeFile.Path) {
				s.logger.Debug("    ✅ File exists: %s", episodeFile.Path)
				episodeResultsChan <- episodeResult{episode: ep, stats: episodeStats, err: nil}
				return
			}

			// File is missing
			episodeStats.MissingFiles++
			s.progressReporter.ReportMissingFile(episodeFile.Path)

			if s.dryRun {
				s.logger.Info("    🏃 DRY RUN: Would delete episode file record %d", *ep.EpisodeFileID)
				episodeResultsChan <- episodeResult{episode: ep, stats: episodeStats, err: nil}
				return
			}

			// Delete the episode file record
			s.logger.Info("    🗑️  Deleting episode file record %d...", *ep.EpisodeFileID)
			if err := s.client.DeleteEpisodeFile(ctx, *ep.EpisodeFileID); err != nil {
				s.logger.Error("    ❌ Failed to delete episode file record %d: %s", *ep.EpisodeFileID, err.Error())
				s.progressReporter.ReportError(err)
				episodeStats.Errors++
				episodeResultsChan <- episodeResult{episode: ep, stats: episodeStats, err: nil}
				return
			}

			episodeStats.DeletedRecords++
			s.progressReporter.ReportDeletedEpisodeRecord(*ep.EpisodeFileID)

			// Note: In modern Sonarr versions, deleting the episode file record
			// automatically updates the episode status, so explicit updates are not needed
			// and can cause HTTP 400 errors. If you need explicit updates, uncomment below:

			// s.logger.Debug("    🔄 Updating episode status...")
			// if err := s.client.UpdateEpisode(ctx, ep); err != nil {
			//     s.logger.Warn("    ⚠️  Failed to update episode %d: %s", ep.ID, err.Error())
			//     // This is not critical, so we continue
			// }

			episodeResultsChan <- episodeResult{episode: ep, stats: episodeStats, err: nil}

			// Small delay between operations
			if s.requestDelay > 0 {
				time.Sleep(s.requestDelay)
			}
		}(episode)
	}

	// Close results channel when all episode workers are done
	go func() {
		episodeWg.Wait()
		close(episodeResultsChan)
	}()

	// Collect episode results
	for result := range episodeResultsChan {
		if result.err != nil {
			if result.err == ctx.Err() {
				return stats, result.err
			}
		}

		episodeMu.Lock()
		stats.TotalItemsChecked += result.stats.TotalItemsChecked
		stats.MissingFiles += result.stats.MissingFiles
		stats.DeletedRecords += result.stats.DeletedRecords
		stats.Errors += result.stats.Errors
		episodeMu.Unlock()
	}

	return stats, nil
}

// cleanupMovie processes a single movie
func (s *CleanupServiceImpl) cleanupMovie(ctx context.Context, movieID int) (models.CleanupStats, error) {
	stats := models.CleanupStats{}

	// Get the specific movie directly
	s.logger.Debug("Fetching movie %d...", movieID)
	targetMovie, err := s.client.GetMovie(ctx, movieID)
	if err != nil {
		return stats, fmt.Errorf("failed to get movie %d: %w", movieID, err)
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
		// If movie file is not found, it might have been already deleted
		// This is not an error condition - just skip this movie
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			s.logger.Info("    ℹ️  Movie file %d already deleted or not found", *targetMovie.MovieFileID)
			return stats, nil
		}
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
	s.progressReporter.ReportDeletedMovieRecord(*targetMovie.MovieFileID)

	// Note: In modern Radarr versions, deleting the movie file record
	// automatically updates the movie status, so explicit updates are not needed
	// and can cause HTTP 400 errors. If you need explicit updates, uncomment below:

	// s.logger.Debug("    🔄 Updating movie status...")
	// if err := s.client.UpdateMovie(ctx, *targetMovie); err != nil {
	//     s.logger.Warn("    ⚠️  Failed to update movie %d: %s", targetMovie.ID, err.Error())
	//     // This is not critical, so we continue
	// }

	// Small delay between operations
	if s.requestDelay > 0 {
		time.Sleep(s.requestDelay)
	}

	return stats, nil
}
