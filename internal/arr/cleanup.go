package arr

import (
	"context"
	"fmt"
	"sort"
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
	qualityProfileID int  // Quality profile ID for adding movies/series
	addMissingMovies bool // Whether to add missing movies/series from broken symlinks to collection
	missingFiles     []models.MissingFileEntry
	missingFilesMu   sync.Mutex
	seriesInfo       map[int]string // seriesID -> seriesName
	movieInfo        map[int]string // movieID -> movieName
	mediaInfoMu      sync.RWMutex
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
		qualityProfileID: 12,    // Default quality profile ID
		addMissingMovies: false, // Default to disabled
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
	qualityProfileID int,
	addMissingMovies bool,
) CleanupService {
	return &CleanupServiceImpl{
		client:           client,
		fileChecker:      fileChecker,
		logger:           logger,
		progressReporter: progressReporter,
		requestDelay:     requestDelay,
		concurrentLimit:  concurrentLimit,
		dryRun:           dryRun,
		qualityProfileID: qualityProfileID,
		addMissingMovies: addMissingMovies,
	}
}

// CleanupMissingFiles performs cleanup for all series or movies based on client type
// addMissingFileEntry safely adds a missing file entry to the collection
func (s *CleanupServiceImpl) addMissingFileEntry(entry models.MissingFileEntry) {
	s.missingFilesMu.Lock()
	defer s.missingFilesMu.Unlock()
	s.missingFiles = append(s.missingFiles, entry)
}

// deduplicateMissingFiles removes duplicate entries, prioritizing those with real FileIDs
func (s *CleanupServiceImpl) deduplicateMissingFiles(entries []models.MissingFileEntry) []models.MissingFileEntry {
	// Use a map to track the best entry for each unique identifier
	entryMap := make(map[string]models.MissingFileEntry)

	for _, entry := range entries {
		// Create a unique key for deduplication
		var key string
		if entry.MediaType == "movie" && entry.TMDBID > 0 {
			// For movies with TMDB ID, use TMDB ID as primary key
			key = fmt.Sprintf("movie-tmdb-%d", entry.TMDBID)
		} else if entry.MediaType == "series" && entry.TVDBID > 0 {
			// For series with TVDB ID, use TVDB ID as primary key
			key = fmt.Sprintf("series-tvdb-%d", entry.TVDBID)
		} else {
			// For series or movies without TMDB/TVDB ID, use file path
			key = fmt.Sprintf("%s-path-%s", entry.MediaType, entry.FilePath)
		}

		// Check if we already have an entry for this key
		if existing, exists := entryMap[key]; exists {
			// Prioritize entry with real FileID (> 0) over broken symlink entries (FileID = 0)
			if entry.FileID > 0 && existing.FileID == 0 {
				entryMap[key] = entry
			} else if entry.FileID == 0 && existing.FileID > 0 {
				// Keep existing entry (which has real FileID)
				continue
			} else {
				// Both have same FileID type, keep the more recent one
				if entry.ProcessedAt > existing.ProcessedAt {
					entryMap[key] = entry
				}
			}
		} else {
			// First entry for this key
			entryMap[key] = entry
		}
	}

	// Convert map back to slice
	deduplicated := make([]models.MissingFileEntry, 0, len(entryMap))
	for _, entry := range entryMap {
		deduplicated = append(deduplicated, entry)
	}

	// Sort by ProcessedAt to maintain consistent order
	sort.Slice(deduplicated, func(i, j int) bool {
		return deduplicated[i].ProcessedAt < deduplicated[j].ProcessedAt
	})

	return deduplicated
}

// buildReport creates a missing files report from collected data
func (s *CleanupServiceImpl) buildReport() *models.MissingFilesReport {
	s.missingFilesMu.Lock()
	defer s.missingFilesMu.Unlock()

	runType := "real-run"
	if s.dryRun {
		runType = "dry-run"
	}

	// Deduplicate missing files before building the report
	deduplicatedFiles := s.deduplicateMissingFiles(s.missingFiles)

	return &models.MissingFilesReport{
		GeneratedAt:  time.Now().Format(time.RFC3339),
		RunType:      runType,
		ServiceType:  s.client.GetName(),
		TotalMissing: len(deduplicatedFiles),
		MissingFiles: deduplicatedFiles,
	}
}

// setSeriesInfo safely sets series information
func (s *CleanupServiceImpl) setSeriesInfo(seriesID int, seriesName string) {
	s.mediaInfoMu.Lock()
	defer s.mediaInfoMu.Unlock()
	if s.seriesInfo == nil {
		s.seriesInfo = make(map[int]string)
	}
	s.seriesInfo[seriesID] = seriesName
}

// getSeriesInfo safely gets series information
func (s *CleanupServiceImpl) getSeriesInfo(seriesID int) string {
	s.mediaInfoMu.RLock()
	defer s.mediaInfoMu.RUnlock()
	if name, exists := s.seriesInfo[seriesID]; exists {
		return name
	}
	return fmt.Sprintf("Series %d", seriesID)
}

// setMovieInfo safely sets movie information
func (s *CleanupServiceImpl) setMovieInfo(movieID int, movieName string) {
	s.mediaInfoMu.Lock()
	defer s.mediaInfoMu.Unlock()
	if s.movieInfo == nil {
		s.movieInfo = make(map[int]string)
	}
	s.movieInfo[movieID] = movieName
}

// getMovieInfo safely gets movie information
func (s *CleanupServiceImpl) getMovieInfo(movieID int) string {
	s.mediaInfoMu.RLock()
	defer s.mediaInfoMu.RUnlock()
	if name, exists := s.movieInfo[movieID]; exists {
		return name
	}
	return fmt.Sprintf("Movie %d", movieID)
}

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
				Report:  s.buildReport(),
			}, nil
		}

		s.logger.Info("Found %d series", len(series))

		// Store series information and extract series IDs
		var seriesIDs []int
		for _, series := range series {
			s.setSeriesInfo(series.ID, series.Title)
			seriesIDs = append(seriesIDs, series.ID)
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
				Report:  s.buildReport(),
			}, nil
		}

		s.logger.Info("Found %d movies", len(movies))

		// Store movie information and extract movie IDs
		var movieIDs []int
		for _, movie := range movies {
			s.setMovieInfo(movie.ID, movie.Title)
			movieIDs = append(movieIDs, movie.ID)
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

	// Handle broken symlinks if this is a Sonarr client
	if s.client.GetName() == "sonarr" {
		s.logger.Info("Step 1.5: Checking for broken symlinks and missing series...")
		symlinkStats, err := s.handleBrokenSymlinksForSeries(ctx)
		if err != nil {
			s.logger.Warn("Broken symlink handling failed: %s", err.Error())
			// Don't fail the entire operation, just add to messages
			messages = append(messages, fmt.Sprintf("Broken symlink handling failed: %s", err.Error()))
		} else {
			// Merge symlink stats into main stats
			mu.Lock()
			stats.TotalItemsChecked += symlinkStats.TotalItemsChecked
			stats.MissingFiles += symlinkStats.MissingFiles
			stats.Errors += symlinkStats.Errors
			mu.Unlock()
		}
	}

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
					Report:   s.buildReport(),
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
		Report:   s.buildReport(),
	}, nil
}

// CleanupMissingFilesForMovies performs cleanup for specific movies using concurrent processing
func (s *CleanupServiceImpl) CleanupMissingFilesForMovies(ctx context.Context, movieIDs []int) (*models.CleanupResult, error) {
	stats := models.CleanupStats{}
	var messages []string
	var mu sync.Mutex

	movieCount := len(movieIDs)
	s.logger.Info("Processing %d movies with concurrency limit of %d", movieCount, s.concurrentLimit)

	// Handle broken symlinks if this is a Radarr client
	if s.client.GetName() == "radarr" {
		s.logger.Info("Step 1.5: Checking for broken symlinks and missing movies...")
		symlinkStats, err := s.handleBrokenSymlinks(ctx)
		if err != nil {
			s.logger.Warn("Broken symlink handling failed: %s", err.Error())
			// Don't fail the entire operation, just add to messages
			messages = append(messages, fmt.Sprintf("Broken symlink handling failed: %s", err.Error()))
		} else {
			// Merge symlink stats into main stats
			mu.Lock()
			stats.TotalItemsChecked += symlinkStats.TotalItemsChecked
			stats.MissingFiles += symlinkStats.MissingFiles
			stats.Errors += symlinkStats.Errors
			mu.Unlock()
		}
	}

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
					Report:   s.buildReport(),
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
		Report:   s.buildReport(),
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

			// Add to missing files report
			seriesName := s.getSeriesInfo(ep.SeriesID)
			season := ep.SeasonNumber
			episode := ep.EpisodeNumber
			missingEntry := models.MissingFileEntry{
				MediaType:   "series",
				MediaName:   seriesName,
				EpisodeName: ep.Title,
				Season:      &season,
				Episode:     &episode,
				FilePath:    episodeFile.Path,
				FileID:      *ep.EpisodeFileID,
				ProcessedAt: time.Now().Format(time.RFC3339),
			}
			s.addMissingFileEntry(missingEntry)

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

	// Add to missing files report
	movieName := s.getMovieInfo(targetMovie.ID)
	missingEntry := models.MissingFileEntry{
		MediaType:   "movie",
		MediaName:   movieName,
		FilePath:    movieFile.Path,
		FileID:      *targetMovie.MovieFileID,
		ProcessedAt: time.Now().Format(time.RFC3339),
		TMDBID:      targetMovie.TMDBID,
	}
	s.addMissingFileEntry(missingEntry)

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

// handleBrokenSymlinks scans for broken symlinks and adds missing movies to Radarr collection
func (s *CleanupServiceImpl) handleBrokenSymlinks(ctx context.Context) (models.CleanupStats, error) {
	stats := models.CleanupStats{}

	s.logger.Info("Scanning for broken symlinks in Radarr root directories...")

	// Get Radarr root folders
	rootFolders, err := s.client.GetRootFolders(ctx)
	if err != nil {
		return stats, fmt.Errorf("failed to get root folders: %w", err)
	}

	if len(rootFolders) == 0 {
		s.logger.Info("No root folders configured in Radarr")
		return stats, nil
	}

	// Define movie file extensions to look for
	movieExtensions := []string{".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"}

	// Scan each root folder for broken symlinks
	var allBrokenSymlinks []string
	for _, folder := range rootFolders {
		s.logger.Info("Scanning root folder: %s", folder.Path)

		brokenSymlinks, err := s.fileChecker.FindBrokenSymlinks(folder.Path, movieExtensions)
		if err != nil {
			s.logger.Warn("Failed to scan folder %s: %s", folder.Path, err.Error())
			stats.Errors++
			continue
		}

		s.logger.Info("Found %d broken symlinks in %s", len(brokenSymlinks), folder.Path)
		allBrokenSymlinks = append(allBrokenSymlinks, brokenSymlinks...)
	}

	if len(allBrokenSymlinks) == 0 {
		s.logger.Info("No broken symlinks found")
		return stats, nil
	}

	s.logger.Info("Processing %d broken symlinks...", len(allBrokenSymlinks))

	// Process each broken symlink
	for _, symlinkPath := range allBrokenSymlinks {
		symlinkStats, err := s.handleBrokenSymlink(ctx, symlinkPath, rootFolders)
		if err != nil {
			s.logger.Error("Failed to handle broken symlink %s: %s", symlinkPath, err.Error())
			stats.Errors++
			continue
		}

		stats.TotalItemsChecked += symlinkStats.TotalItemsChecked
		stats.MissingFiles += symlinkStats.MissingFiles
	}

	return stats, nil
}

// handleBrokenSymlink processes a single broken symlink
func (s *CleanupServiceImpl) handleBrokenSymlink(ctx context.Context, symlinkPath string, rootFolders []models.RootFolder) (models.CleanupStats, error) {
	stats := models.CleanupStats{TotalItemsChecked: 1}

	s.logger.Debug("Processing broken symlink: %s", symlinkPath)

	// Extract TMDB ID from path
	tmdbID, err := models.ParseTMDBIDFromPath(symlinkPath)
	if err != nil {
		s.logger.Warn("Could not parse TMDB ID from path %s: %s", symlinkPath, err.Error())
		return stats, nil // Not an error, just skip this file
	}

	s.logger.Debug("Extracted TMDB ID %d from %s", tmdbID, symlinkPath)

	// Delete the broken symlink before processing (if not in dry-run mode)
	if !s.dryRun {
		s.logger.Info("🗑️  Deleting broken symlink: %s", symlinkPath)
		if err := s.fileChecker.DeleteSymlink(symlinkPath); err != nil {
			s.logger.Error("Failed to delete broken symlink %s: %s", symlinkPath, err.Error())
			stats.Errors++
			return stats, fmt.Errorf("failed to delete broken symlink %s: %w", symlinkPath, err)
		}
		s.logger.Info("✅ Successfully deleted broken symlink: %s", symlinkPath)
	} else {
		s.logger.Info("🏃 DRY RUN: Would delete broken symlink: %s", symlinkPath)
	}

	// Check if movie already exists in Radarr collection
	existingMovie, err := s.client.GetMovieByTMDBID(ctx, tmdbID)
	if err == nil {
		// Movie already exists in collection
		s.logger.Debug("Movie with TMDB ID %d already exists in collection: %s", tmdbID, existingMovie.Title)

		// Add to missing files report but don't add to collection
		missingEntry := models.MissingFileEntry{
			MediaType:         "movie",
			MediaName:         existingMovie.Title,
			FilePath:          symlinkPath,
			FileID:            0, // No file ID since it's a broken symlink
			ProcessedAt:       time.Now().Format(time.RFC3339),
			AddedToCollection: false,
			TMDBID:            tmdbID,
		}
		s.addMissingFileEntry(missingEntry)
		stats.MissingFiles++
		return stats, nil
	}

	// Movie not found in collection, need to add it
	s.logger.Info("Movie with TMDB ID %d not found in collection, looking up details...", tmdbID)

	// Lookup movie details from TMDB
	movieLookup, err := s.client.LookupMovieByTMDBID(ctx, tmdbID)
	if err != nil {
		return stats, fmt.Errorf("failed to lookup movie with TMDB ID %d: %w", tmdbID, err)
	}

	// Determine which root folder to use (prefer the one that contains the broken symlink)
	var selectedRootFolder *models.RootFolder
	for _, folder := range rootFolders {
		if strings.HasPrefix(symlinkPath, folder.Path) {
			selectedRootFolder = &folder
			break
		}
	}

	// If no matching root folder found, use the first one
	if selectedRootFolder == nil && len(rootFolders) > 0 {
		selectedRootFolder = &rootFolders[0]
		s.logger.Debug("Using first available root folder: %s", selectedRootFolder.Path)
	}

	if selectedRootFolder == nil {
		return stats, fmt.Errorf("no suitable root folder found for movie")
	}

	// Create movie object for adding to collection
	movieToAdd := models.Movie{
		MediaItem: models.MediaItem{
			Title: movieLookup.Title,
		},
		Year:             movieLookup.Year,
		TMDBID:           movieLookup.TMDBID,
		Monitored:        true,
		QualityProfileID: s.qualityProfileID,
		RootFolderPath:   selectedRootFolder.Path,
		HasFile:          false,
	}

	if s.addMissingMovies && !s.dryRun {
		// Add movie to Radarr collection
		s.logger.Info("Adding movie to collection: %s (%d)", movieLookup.Title, movieLookup.Year)
		addedMovie, err := s.client.AddMovie(ctx, movieToAdd)
		if err != nil {
			return stats, fmt.Errorf("failed to add movie %s: %w", movieLookup.Title, err)
		}

		// Update our movie info cache
		s.setMovieInfo(addedMovie.ID, addedMovie.Title)
	} else if s.dryRun {
		s.logger.Info("🏃 DRY RUN: Would add movie to collection: %s (%d)", movieLookup.Title, movieLookup.Year)
	} else if !s.addMissingMovies {
		s.logger.Info("📋 ADD_MISSING_MOVIES=false: Would add movie to collection: %s (%d)", movieLookup.Title, movieLookup.Year)
	}

	// Add to missing files report
	missingEntry := models.MissingFileEntry{
		MediaType:         "movie",
		MediaName:         movieLookup.Title,
		FilePath:          symlinkPath,
		FileID:            0, // No file ID since it's a broken symlink
		ProcessedAt:       time.Now().Format(time.RFC3339),
		AddedToCollection: s.addMissingMovies && !s.dryRun,
		TMDBID:            tmdbID,
	}
	s.addMissingFileEntry(missingEntry)
	stats.MissingFiles++

	return stats, nil
}

// handleBrokenSymlinksForSeries scans for broken symlinks and adds missing series to Sonarr collection
func (s *CleanupServiceImpl) handleBrokenSymlinksForSeries(ctx context.Context) (models.CleanupStats, error) {
	stats := models.CleanupStats{}

	s.logger.Info("Scanning for broken symlinks in Sonarr root directories...")

	// Get Sonarr root folders
	rootFolders, err := s.client.GetRootFolders(ctx)
	if err != nil {
		return stats, fmt.Errorf("failed to get root folders: %w", err)
	}

	if len(rootFolders) == 0 {
		s.logger.Info("No root folders configured in Sonarr")
		return stats, nil
	}

	// Define series file extensions to look for
	seriesExtensions := []string{".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"}

	// Scan each root folder for broken symlinks
	var allBrokenSymlinks []string
	for _, folder := range rootFolders {
		s.logger.Info("Scanning root folder: %s", folder.Path)

		brokenSymlinks, err := s.fileChecker.FindBrokenSymlinks(folder.Path, seriesExtensions)
		if err != nil {
			s.logger.Warn("Failed to scan folder %s: %s", folder.Path, err.Error())
			stats.Errors++
			continue
		}

		s.logger.Info("Found %d broken symlinks in %s", len(brokenSymlinks), folder.Path)
		allBrokenSymlinks = append(allBrokenSymlinks, brokenSymlinks...)
	}

	if len(allBrokenSymlinks) == 0 {
		s.logger.Info("No broken symlinks found")
		return stats, nil
	}

	s.logger.Info("Processing %d broken symlinks...", len(allBrokenSymlinks))

	// Process each broken symlink
	for _, symlinkPath := range allBrokenSymlinks {
		symlinkStats, err := s.handleBrokenSymlinkForSeries(ctx, symlinkPath, rootFolders)
		if err != nil {
			s.logger.Error("Failed to handle broken symlink %s: %s", symlinkPath, err.Error())
			stats.Errors++
			continue
		}

		stats.TotalItemsChecked += symlinkStats.TotalItemsChecked
		stats.MissingFiles += symlinkStats.MissingFiles
	}

	return stats, nil
}

// handleBrokenSymlinkForSeries processes a single broken symlink for series
func (s *CleanupServiceImpl) handleBrokenSymlinkForSeries(ctx context.Context, symlinkPath string, rootFolders []models.RootFolder) (models.CleanupStats, error) {
	stats := models.CleanupStats{TotalItemsChecked: 1}

	s.logger.Debug("Processing broken symlink: %s", symlinkPath)

	// Extract TVDB ID from path
	tvdbID, err := models.ParseTVDBIDFromPath(symlinkPath)
	if err != nil {
		s.logger.Warn("Could not parse TVDB ID from path %s: %s", symlinkPath, err.Error())
		return stats, nil // Not an error, just skip this file
	}

	s.logger.Debug("Extracted TVDB ID %d from %s", tvdbID, symlinkPath)

	// Delete the broken symlink before processing (if not in dry-run mode)
	if !s.dryRun {
		s.logger.Info("🗑️  Deleting broken symlink: %s", symlinkPath)
		if err := s.fileChecker.DeleteSymlink(symlinkPath); err != nil {
			s.logger.Error("Failed to delete broken symlink %s: %s", symlinkPath, err.Error())
			stats.Errors++
			return stats, fmt.Errorf("failed to delete broken symlink %s: %w", symlinkPath, err)
		}
		s.logger.Info("✅ Successfully deleted broken symlink: %s", symlinkPath)
	} else {
		s.logger.Info("🏃 DRY RUN: Would delete broken symlink: %s", symlinkPath)
	}

	// Check if series already exists in Sonarr collection
	existingSeries, err := s.client.GetSeriesByTVDBID(ctx, tvdbID)
	if err == nil {
		// Series already exists in collection
		s.logger.Debug("Series with TVDB ID %d already exists in collection: %s", tvdbID, existingSeries.Title)

		// Add to missing files report but don't add to collection
		missingEntry := models.MissingFileEntry{
			MediaType:         "series",
			MediaName:         existingSeries.Title,
			FilePath:          symlinkPath,
			FileID:            0, // No file ID since it's a broken symlink
			ProcessedAt:       time.Now().Format(time.RFC3339),
			AddedToCollection: false,
			TVDBID:            tvdbID,
		}
		s.addMissingFileEntry(missingEntry)
		stats.MissingFiles++
		return stats, nil
	}

	// Series not found in collection, need to add it
	s.logger.Info("Series with TVDB ID %d not found in collection, looking up details...", tvdbID)

	// Lookup series details from TVDB
	seriesLookup, err := s.client.LookupSeriesByTVDBID(ctx, tvdbID)
	if err != nil {
		return stats, fmt.Errorf("failed to lookup series with TVDB ID %d: %w", tvdbID, err)
	}

	// Determine which root folder to use (prefer the one that contains the broken symlink)
	var selectedRootFolder *models.RootFolder
	for _, folder := range rootFolders {
		if strings.HasPrefix(symlinkPath, folder.Path) {
			selectedRootFolder = &folder
			break
		}
	}

	// If no matching root folder found, use the first one
	if selectedRootFolder == nil && len(rootFolders) > 0 {
		selectedRootFolder = &rootFolders[0]
		s.logger.Debug("Using first available root folder: %s", selectedRootFolder.Path)
	}

	if selectedRootFolder == nil {
		return stats, fmt.Errorf("no suitable root folder found for series")
	}

	// Create series object for adding to collection
	seriesToAdd := models.Series{
		MediaItem: models.MediaItem{
			Title: seriesLookup.Title,
		},
		TVDBID:           seriesLookup.TVDBID,
		Monitored:        true,
		QualityProfileID: s.qualityProfileID,
		RootFolderPath:   selectedRootFolder.Path,
	}

	if s.addMissingMovies && !s.dryRun {
		// Add series to Sonarr collection
		s.logger.Info("Adding series to collection: %s", seriesLookup.Title)
		addedSeries, err := s.client.AddSeries(ctx, seriesToAdd)
		if err != nil {
			return stats, fmt.Errorf("failed to add series %s: %w", seriesLookup.Title, err)
		}

		// Update our series info cache
		s.setSeriesInfo(addedSeries.ID, addedSeries.Title)
	} else if s.dryRun {
		s.logger.Info("🏃 DRY RUN: Would add series to collection: %s", seriesLookup.Title)
	} else if !s.addMissingMovies {
		s.logger.Info("📋 ADD_MISSING_MOVIES=false: Would add series to collection: %s", seriesLookup.Title)
	}

	// Add to missing files report
	missingEntry := models.MissingFileEntry{
		MediaType:         "series",
		MediaName:         seriesLookup.Title,
		FilePath:          symlinkPath,
		FileID:            0, // No file ID since it's a broken symlink
		ProcessedAt:       time.Now().Format(time.RFC3339),
		AddedToCollection: s.addMissingMovies && !s.dryRun,
		TVDBID:            tvdbID,
	}
	s.addMissingFileEntry(missingEntry)
	stats.MissingFiles++

	return stats, nil
}
