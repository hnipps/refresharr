package arr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/pkg/models"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

// SonarrClient implements the Client interface for Sonarr API
type SonarrClient struct {
	client *sonarr.Sonarr
	logger Logger
}

// NewSonarrClient creates a new Sonarr client
func NewSonarrClient(cfg *config.SonarrConfig, timeout time.Duration, logger Logger) Client {
	// Create starr config
	starrConfig := starr.New(cfg.APIKey, cfg.URL, timeout)

	// Create sonarr client
	sonarrClient := sonarr.New(starrConfig)

	return &SonarrClient{
		client: sonarrClient,
		logger: logger,
	}
}

// GetName returns the service name
func (c *SonarrClient) GetName() string {
	return "sonarr"
}

// TestConnection verifies the connection to Sonarr
func (c *SonarrClient) TestConnection(ctx context.Context) error {
	_, err := c.client.GetSystemStatusContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Sonarr: %w", err)
	}

	c.logger.Info("✅ Successfully connected to Sonarr")
	return nil
}

// GetAllSeries returns all series from Sonarr
func (c *SonarrClient) GetAllSeries(ctx context.Context) ([]models.Series, error) {
	series, err := c.client.GetAllSeriesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch series: %w", err)
	}

	result := mapSonarrSeriesToModelsList(series)
	c.logger.Debug("Fetched %d series from Sonarr", len(result))
	return result, nil
}

// GetAllMovies is not applicable for Sonarr (returns nil)
func (c *SonarrClient) GetAllMovies(ctx context.Context) ([]models.Movie, error) {
	return nil, fmt.Errorf("GetAllMovies is not supported by Sonarr client")
}

// GetMovie is not applicable for Sonarr (returns error)
func (c *SonarrClient) GetMovie(ctx context.Context, movieID int) (*models.Movie, error) {
	return nil, fmt.Errorf("GetMovie is not supported by Sonarr client")
}

// GetEpisodesForSeries returns all episodes for a given series
func (c *SonarrClient) GetEpisodesForSeries(ctx context.Context, seriesID int) ([]models.Episode, error) {
	getEpisode := &sonarr.GetEpisode{
		SeriesID: int64(seriesID),
	}

	episodes, err := c.client.GetSeriesEpisodesContext(ctx, getEpisode)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch episodes for series %d: %w", seriesID, err)
	}

	result := mapSonarrEpisodesToModelsList(episodes)
	c.logger.Debug("Fetched %d episodes for series %d", len(result), seriesID)
	return result, nil
}

// GetEpisodeFile returns episode file details
func (c *SonarrClient) GetEpisodeFile(ctx context.Context, fileID int) (*models.EpisodeFile, error) {
	episodeFiles, err := c.client.GetEpisodeFilesContext(ctx, int64(fileID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch episode file %d: %w", fileID, err)
	}

	if len(episodeFiles) == 0 {
		return nil, fmt.Errorf("episode file %d not found", fileID)
	}

	result := mapSonarrEpisodeFileToModels(episodeFiles[0])
	return &result, nil
}

// DeleteEpisodeFile deletes an episode file record
func (c *SonarrClient) DeleteEpisodeFile(ctx context.Context, fileID int) error {
	err := c.client.DeleteEpisodeFileContext(ctx, int64(fileID))
	if err != nil {
		return fmt.Errorf("failed to delete episode file %d: %w", fileID, err)
	}

	c.logger.Debug("Successfully deleted episode file %d", fileID)
	return nil
}

// UpdateEpisode updates an episode's metadata
func (c *SonarrClient) UpdateEpisode(ctx context.Context, episode models.Episode) error {
	// First get the current episode data
	currentEpisode, err := c.client.GetEpisodeByIDContext(ctx, int64(episode.ID))
	if err != nil {
		return fmt.Errorf("failed to fetch current episode %d data: %w", episode.ID, err)
	}

	// Update the file reference fields
	currentEpisode.HasFile = false
	currentEpisode.EpisodeFileID = 0

	// Update the episode using starr's MonitorEpisode method
	// Note: starr doesn't have a direct update episode method, so we use MonitorEpisode
	// with monitoring set to current state to trigger an update
	_, err = c.client.MonitorEpisodeContext(ctx, []int64{int64(episode.ID)}, currentEpisode.Monitored)
	if err != nil {
		return fmt.Errorf("failed to update episode %d: %w", episode.ID, err)
	}

	c.logger.Debug("Successfully updated episode %d", episode.ID)
	return nil
}

// GetMovieFile is not applicable for Sonarr (returns error)
func (c *SonarrClient) GetMovieFile(ctx context.Context, fileID int) (*models.MovieFile, error) {
	return nil, fmt.Errorf("GetMovieFile is not supported by Sonarr client")
}

// DeleteMovieFile is not applicable for Sonarr (returns error)
func (c *SonarrClient) DeleteMovieFile(ctx context.Context, fileID int) error {
	return fmt.Errorf("DeleteMovieFile is not supported by Sonarr client")
}

// UpdateMovie is not applicable for Sonarr (returns error)
func (c *SonarrClient) UpdateMovie(ctx context.Context, movie models.Movie) error {
	return fmt.Errorf("UpdateMovie is not supported by Sonarr client")
}

// GetRootFolders returns all root folders from Sonarr
func (c *SonarrClient) GetRootFolders(ctx context.Context) ([]models.RootFolder, error) {
	rootFolders, err := c.client.GetRootFoldersContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch root folders: %w", err)
	}

	result := mapSonarrRootFoldersToModelsList(rootFolders)
	c.logger.Debug("Fetched %d root folders from Sonarr", len(result))
	return result, nil
}

// GetQualityProfiles returns all quality profiles from Sonarr
func (c *SonarrClient) GetQualityProfiles(ctx context.Context) ([]models.QualityProfile, error) {
	qualityProfiles, err := c.client.GetQualityProfilesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quality profiles: %w", err)
	}

	result := mapSonarrQualityProfilesToModelsList(qualityProfiles)
	c.logger.Debug("Fetched %d quality profiles from Sonarr", len(result))
	return result, nil
}

// LookupMovieByTMDBID is not applicable for Sonarr (returns error)
func (c *SonarrClient) LookupMovieByTMDBID(ctx context.Context, tmdbID int) (*models.MovieLookup, error) {
	return nil, fmt.Errorf("LookupMovieByTMDBID is not supported by Sonarr client")
}

// GetMovieByTMDBID is not applicable for Sonarr (returns error)
func (c *SonarrClient) GetMovieByTMDBID(ctx context.Context, tmdbID int) (*models.Movie, error) {
	return nil, fmt.Errorf("GetMovieByTMDBID is not supported by Sonarr client")
}

// AddMovie is not applicable for Sonarr (returns error)
func (c *SonarrClient) AddMovie(ctx context.Context, movie models.Movie) (*models.Movie, error) {
	return nil, fmt.Errorf("AddMovie is not supported by Sonarr client")
}

// TriggerRefresh triggers a missing episode search
func (c *SonarrClient) TriggerRefresh(ctx context.Context) error {
	command := &sonarr.CommandRequest{
		Name: "MissingEpisodeSearch",
	}

	_, err := c.client.SendCommandContext(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to trigger refresh: %w", err)
	}

	c.logger.Info("✅ Refresh triggered successfully")
	return nil
}

// AddSeries adds a series to the Sonarr collection
func (c *SonarrClient) AddSeries(ctx context.Context, series models.Series) (*models.Series, error) {
	// Convert models.Series to sonarr.AddSeriesInput
	addSeriesInput := &sonarr.AddSeriesInput{
		Title:            series.Title,
		TvdbID:           int64(series.TVDBID),
		Path:             series.Path,
		QualityProfileID: int64(series.QualityProfileID),
		RootFolderPath:   series.RootFolderPath,
		Monitored:        series.Monitored,
		SeasonFolder:     true, // Default to true
	}

	addedSeries, err := c.client.AddSeriesContext(ctx, addSeriesInput)
	if err != nil {
		return nil, fmt.Errorf("failed to add series: %w", err)
	}

	result := mapSonarrSeriesToModels(addedSeries)
	c.logger.Info("✅ Successfully added series: %s with TVDB ID %d", result.Title, result.TVDBID)
	return &result, nil
}

// GetSeriesByTVDBID returns a series by TVDB ID if it exists in the collection
func (c *SonarrClient) GetSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.Series, error) {
	// Get series by TVDB ID using starr's GetSeries method
	series, err := c.client.GetSeriesContext(ctx, int64(tvdbID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch series with TVDB ID %d: %w", tvdbID, err)
	}

	if len(series) == 0 {
		return nil, fmt.Errorf("series with TVDB ID %d not found in collection", tvdbID)
	}

	// Find the series with matching TVDB ID
	for _, s := range series {
		if int(s.TvdbID) == tvdbID {
			result := mapSonarrSeriesToModels(s)
			return &result, nil
		}
	}

	return nil, fmt.Errorf("series with TVDB ID %d not found in collection", tvdbID)
}

// LookupSeriesByTVDBID looks up series information by TVDB ID
func (c *SonarrClient) LookupSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.SeriesLookup, error) {
	term := fmt.Sprintf("tvdb:%d", tvdbID)
	series, err := c.client.GetSeriesLookupContext(ctx, term, int64(tvdbID))
	if err != nil {
		return nil, fmt.Errorf("failed to lookup series with TVDB ID %d: %w", tvdbID, err)
	}

	if len(series) == 0 {
		return nil, fmt.Errorf("series with TVDB ID %d not found", tvdbID)
	}

	// Find the series with matching TVDB ID (API might return multiple results)
	for _, s := range series {
		if int(s.TvdbID) == tvdbID {
			result := &models.SeriesLookup{
				TVDBID:   int(s.TvdbID),
				Title:    s.Title,
				Year:     s.Year,
				Overview: s.Overview,
				Images: make([]struct {
					CoverType string `json:"coverType"`
					URL       string `json:"url"`
				}, len(s.Images)),
			}

			// Map images if present
			for i, img := range s.Images {
				result.Images[i].CoverType = img.CoverType
				result.Images[i].URL = img.URL
			}

			c.logger.Debug("Successfully looked up series with TVDB ID %d: %s", tvdbID, result.Title)
			return result, nil
		}
	}

	return nil, fmt.Errorf("series with TVDB ID %d not found in lookup results", tvdbID)
}

// GetQueue returns all items in the download queue
func (c *SonarrClient) GetQueue(ctx context.Context) ([]models.QueueItem, error) {
	queue, err := c.client.GetQueueContext(ctx, 0, 0) // Get all records
	if err != nil {
		return nil, fmt.Errorf("failed to fetch queue: %w", err)
	}

	result := mapSonarrQueueToModelsList(queue)
	c.logger.Debug("Fetched %d items from queue", len(result))
	return result, nil
}

// GetQueueDetails returns detailed information about a specific queue item
func (c *SonarrClient) GetQueueDetails(ctx context.Context, queueID int) (*models.QueueItem, error) {
	// starr doesn't have a method to get a specific queue item by ID
	// so we'll get all queue items and find the one with matching ID
	queue, err := c.client.GetQueueContext(ctx, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch queue details for ID %d: %w", queueID, err)
	}

	// Find the queue item with matching ID
	for _, qr := range queue.Records {
		if int(qr.ID) == queueID {
			result := mapSonarrQueueRecordToModels(qr)
			return &result, nil
		}
	}

	return nil, fmt.Errorf("queue item %d not found", queueID)
}

// RemoveFromQueue removes an item from the queue
func (c *SonarrClient) RemoveFromQueue(ctx context.Context, queueID int, removeFromClient bool) error {
	// Create queue delete options
	opts := &starr.QueueDeleteOpts{
		RemoveFromClient: &removeFromClient,
		BlockList:        false,
		SkipRedownload:   false,
		ChangeCategory:   false,
	}

	err := c.client.DeleteQueueContext(ctx, int64(queueID), opts)
	if err != nil {
		// Check if it's a "not found" error - this is common and not a real error
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			c.logger.Debug("Queue item %d not found (already removed)", queueID)
			return nil
		}
		return fmt.Errorf("failed to remove queue item %d: %w", queueID, err)
	}

	c.logger.Debug("Successfully removed queue item %d", queueID)
	return nil
}

// TriggerDownloadClientScan triggers a scan of completed downloads
func (c *SonarrClient) TriggerDownloadClientScan(ctx context.Context) error {
	command := &sonarr.CommandRequest{
		Name: "DownloadedEpisodesScan",
	}

	_, err := c.client.SendCommandContext(ctx, command)
	if err != nil {
		// For Sonarr v4+, the DownloadedEpisodesScan command may not be available
		// This is expected and not an error - we'll fall back to other methods
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			c.logger.Debug("Download client scan command not available (likely Sonarr v4+)")
			return nil
		}
		return fmt.Errorf("failed to trigger download client scan: %w", err)
	}

	c.logger.Debug("Successfully triggered download client scan")
	return nil
}

// GetManualImport gets files available for manual import from a folder
func (c *SonarrClient) GetManualImport(ctx context.Context, folder string) ([]models.ManualImportItem, error) {
	params := &sonarr.ManualImportParams{
		Folder:              folder,
		FilterExistingFiles: true,
	}

	manualImportOutput, err := c.client.ManualImportContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manual import items for folder %s: %w", folder, err)
	}

	// Convert from starr ManualImportOutput to our models
	result := make([]models.ManualImportItem, 0)
	if manualImportOutput != nil {
		result = append(result, mapSonarrManualImportToModels(manualImportOutput))
	}

	c.logger.Debug("Found %d manual import items in folder %s", len(result), folder)
	return result, nil
}

// ExecuteManualImport executes manual import for the specified files
func (c *SonarrClient) ExecuteManualImport(ctx context.Context, files []models.ManualImportItem, importMode string) error {
	// Convert each manual import item to starr format and process individually
	for _, file := range files {
		manualImportInput := mapModelsManualImportToSonarr(file)

		err := c.client.ManualImportReprocessContext(ctx, manualImportInput)
		if err != nil {
			return fmt.Errorf("failed to execute manual import for file %s: %w", file.Path, err)
		}
	}

	c.logger.Debug("Successfully initiated manual import for %d files", len(files))
	return nil
}

// GetManualImportWithParams gets files available for manual import with additional parameters
func (c *SonarrClient) GetManualImportWithParams(ctx context.Context, folder, downloadID string, seriesID int, filterExisting bool) ([]models.ManualImportItem, error) {
	params := &sonarr.ManualImportParams{
		Folder:              folder,
		DownloadID:          downloadID,
		FilterExistingFiles: filterExisting,
	}

	if seriesID > 0 {
		params.SeriesID = int64(seriesID)
	}

	manualImportOutput, err := c.client.ManualImportContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manual import items: %w", err)
	}

	// Convert from starr ManualImportOutput to our models
	result := make([]models.ManualImportItem, 0)
	if manualImportOutput != nil {
		result = append(result, mapSonarrManualImportToModels(manualImportOutput))
	}

	c.logger.Debug("Found %d manual import items with custom parameters", len(result))
	return result, nil
}
