package arr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/pkg/models"
)

// SonarrClient implements the Client interface for Sonarr API
type SonarrClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     Logger
}

// NewSonarrClient creates a new Sonarr client
func NewSonarrClient(cfg *config.SonarrConfig, timeout time.Duration, logger Logger) Client {
	return &SonarrClient{
		baseURL: strings.TrimRight(cfg.URL, "/"),
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// GetName returns the service name
func (c *SonarrClient) GetName() string {
	return "sonarr"
}

// TestConnection verifies the connection to Sonarr
func (c *SonarrClient) TestConnection(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "GET", "/api/v3/system/status", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Sonarr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Sonarr returned status %d", resp.StatusCode)
	}

	c.logger.Info("✅ Successfully connected to Sonarr")
	return nil
}

// GetAllSeries returns all series from Sonarr
func (c *SonarrClient) GetAllSeries(ctx context.Context) ([]models.Series, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v3/series", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch series: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch series, status: %d", resp.StatusCode)
	}

	var series []models.Series
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		return nil, fmt.Errorf("failed to decode series response: %w", err)
	}

	c.logger.Debug("Fetched %d series from Sonarr", len(series))
	return series, nil
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
	path := fmt.Sprintf("/api/v3/episode?seriesId=%d", seriesID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch episodes for series %d: %w", seriesID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch episodes for series %d, status: %d", seriesID, resp.StatusCode)
	}

	var episodes []models.Episode
	if err := json.NewDecoder(resp.Body).Decode(&episodes); err != nil {
		return nil, fmt.Errorf("failed to decode episodes response for series %d: %w", seriesID, err)
	}

	c.logger.Debug("Fetched %d episodes for series %d", len(episodes), seriesID)
	return episodes, nil
}

// GetEpisodeFile returns episode file details
func (c *SonarrClient) GetEpisodeFile(ctx context.Context, fileID int) (*models.EpisodeFile, error) {
	path := fmt.Sprintf("/api/v3/episodefile/%d", fileID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch episode file %d: %w", fileID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("episode file %d not found", fileID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch episode file %d, status: %d", fileID, resp.StatusCode)
	}

	var episodeFile models.EpisodeFile
	if err := json.NewDecoder(resp.Body).Decode(&episodeFile); err != nil {
		return nil, fmt.Errorf("failed to decode episode file response for %d: %w", fileID, err)
	}

	return &episodeFile, nil
}

// DeleteEpisodeFile deletes an episode file record
func (c *SonarrClient) DeleteEpisodeFile(ctx context.Context, fileID int) error {
	path := fmt.Sprintf("/api/v3/episodefile/%d", fileID)
	resp, err := c.makeRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete episode file %d: %w", fileID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete episode file %d, status: %d", fileID, resp.StatusCode)
	}

	c.logger.Debug("Successfully deleted episode file %d", fileID)
	return nil
}

// UpdateEpisode updates an episode's metadata
func (c *SonarrClient) UpdateEpisode(ctx context.Context, episode models.Episode) error {
	// First, fetch the current episode data to ensure we have the complete object
	path := fmt.Sprintf("/api/v3/episode/%d", episode.ID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch current episode %d data: %w", episode.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch current episode %d data, status: %d", episode.ID, resp.StatusCode)
	}

	var currentEpisode models.Episode
	if err := json.NewDecoder(resp.Body).Decode(&currentEpisode); err != nil {
		return fmt.Errorf("failed to decode current episode %d data: %w", episode.ID, err)
	}

	// Update the file reference fields
	currentEpisode.HasFile = false
	currentEpisode.EpisodeFileID = nil

	// Marshal the complete episode object
	jsonData, err := json.Marshal(currentEpisode)
	if err != nil {
		return fmt.Errorf("failed to marshal episode update: %w", err)
	}

	// Send the PUT request with the complete episode object
	resp, err = c.makeRequest(ctx, "PUT", path, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to update episode %d: %w", episode.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Get response body for better error reporting
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update episode %d, status: %d, response: %s", episode.ID, resp.StatusCode, string(bodyBytes))
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
	resp, err := c.makeRequest(ctx, "GET", "/api/v3/rootfolder", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch root folders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch root folders, status: %d", resp.StatusCode)
	}

	var rootFolders []models.RootFolder
	if err := json.NewDecoder(resp.Body).Decode(&rootFolders); err != nil {
		return nil, fmt.Errorf("failed to decode root folders response: %w", err)
	}

	c.logger.Debug("Fetched %d root folders from Sonarr", len(rootFolders))
	return rootFolders, nil
}

// GetQualityProfiles returns all quality profiles from Sonarr
func (c *SonarrClient) GetQualityProfiles(ctx context.Context) ([]models.QualityProfile, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v3/qualityprofile", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quality profiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch quality profiles, status: %d", resp.StatusCode)
	}

	var qualityProfiles []models.QualityProfile
	if err := json.NewDecoder(resp.Body).Decode(&qualityProfiles); err != nil {
		return nil, fmt.Errorf("failed to decode quality profiles response: %w", err)
	}

	c.logger.Debug("Fetched %d quality profiles from Sonarr", len(qualityProfiles))
	return qualityProfiles, nil
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
	command := map[string]string{
		"name": "MissingEpisodeSearch",
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal refresh command: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v3/command", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to trigger refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to trigger refresh, status: %d", resp.StatusCode)
	}

	c.logger.Info("✅ Refresh triggered successfully")
	return nil
}

// makeRequest makes an HTTP request to the Sonarr API
func (c *SonarrClient) makeRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Making %s request to %s", method, url)

	return c.httpClient.Do(req)
}

// AddSeries adds a series to the Sonarr collection
func (c *SonarrClient) AddSeries(ctx context.Context, series models.Series) (*models.Series, error) {
	// Marshal the series object
	jsonData, err := json.Marshal(series)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal series for addition: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v3/series", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to add series: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Get response body for better error reporting
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add series, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var addedSeries models.Series
	if err := json.NewDecoder(resp.Body).Decode(&addedSeries); err != nil {
		return nil, fmt.Errorf("failed to decode added series response: %w", err)
	}

	c.logger.Info("✅ Successfully added series: %s with TVDB ID %d", addedSeries.Title, addedSeries.TVDBID)
	return &addedSeries, nil
}

// GetSeriesByTVDBID returns a series by TVDB ID if it exists in the collection
func (c *SonarrClient) GetSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.Series, error) {
	// Get all series and find the one with matching TVDB ID
	series, err := c.GetAllSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch series to search for TVDB ID %d: %w", tvdbID, err)
	}

	for _, s := range series {
		if s.TVDBID == tvdbID {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("series with TVDB ID %d not found in collection", tvdbID)
}

// LookupSeriesByTVDBID looks up series information by TVDB ID
func (c *SonarrClient) LookupSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.SeriesLookup, error) {
	path := fmt.Sprintf("/api/v3/series/lookup?term=tvdb:%d", tvdbID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup series with TVDB ID %d: %w", tvdbID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("series with TVDB ID %d not found", tvdbID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to lookup series with TVDB ID %d, status: %d", tvdbID, resp.StatusCode)
	}

	var seriesLookupResults []models.SeriesLookup
	if err := json.NewDecoder(resp.Body).Decode(&seriesLookupResults); err != nil {
		return nil, fmt.Errorf("failed to decode series lookup response for TVDB ID %d: %w", tvdbID, err)
	}

	// Find the series with matching TVDB ID (API might return multiple results)
	for _, series := range seriesLookupResults {
		if series.TVDBID == tvdbID {
			c.logger.Debug("Successfully looked up series with TVDB ID %d: %s", tvdbID, series.Title)
			return &series, nil
		}
	}

	return nil, fmt.Errorf("series with TVDB ID %d not found in lookup results", tvdbID)
}

// GetQueue returns all items in the download queue
func (c *SonarrClient) GetQueue(ctx context.Context) ([]models.QueueItem, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v3/queue", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch queue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch queue, status: %d", resp.StatusCode)
	}

	var queueResponse models.QueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&queueResponse); err != nil {
		return nil, fmt.Errorf("failed to decode queue response: %w", err)
	}

	c.logger.Debug("Fetched %d items from queue", len(queueResponse.Records))
	return queueResponse.Records, nil
}

// GetQueueDetails returns detailed information about a specific queue item
func (c *SonarrClient) GetQueueDetails(ctx context.Context, queueID int) (*models.QueueItem, error) {
	path := fmt.Sprintf("/api/v3/queue/%d", queueID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch queue details for ID %d: %w", queueID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("queue item %d not found", queueID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch queue details for ID %d, status: %d", queueID, resp.StatusCode)
	}

	var queueItem models.QueueItem
	if err := json.NewDecoder(resp.Body).Decode(&queueItem); err != nil {
		return nil, fmt.Errorf("failed to decode queue details response for ID %d: %w", queueID, err)
	}

	return &queueItem, nil
}

// RemoveFromQueue removes an item from the queue
func (c *SonarrClient) RemoveFromQueue(ctx context.Context, queueID int, removeFromClient bool) error {
	path := fmt.Sprintf("/api/v3/queue/%d", queueID)

	// Construct query parameters
	params := fmt.Sprintf("?removeFromClient=%t&blocklist=false", removeFromClient)

	resp, err := c.makeRequest(ctx, "DELETE", path+params, nil)
	if err != nil {
		return fmt.Errorf("failed to remove queue item %d: %w", queueID, err)
	}
	defer resp.Body.Close()

	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		// Successfully removed
		c.logger.Debug("Successfully removed queue item %d", queueID)
		return nil
	case http.StatusNotFound:
		// Item not found - likely already removed (common with season packs)
		// This is not an error, just means the item was already cleaned up
		c.logger.Debug("Queue item %d not found (already removed)", queueID)
		return nil
	default:
		return fmt.Errorf("failed to remove queue item %d, status: %d", queueID, resp.StatusCode)
	}
}

// TriggerDownloadClientScan triggers a scan of completed downloads
func (c *SonarrClient) TriggerDownloadClientScan(ctx context.Context) error {
	command := map[string]string{
		"name": "DownloadedEpisodesScan",
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal download client scan command: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v3/command", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to trigger download client scan: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// For Sonarr v4+, the DownloadedEpisodesScan command may not be available
		// This is expected and not an error - we'll fall back to other methods
		c.logger.Debug("Download client scan command not available (likely Sonarr v4+), status: %d", resp.StatusCode)
		return nil
	}

	c.logger.Debug("Successfully triggered download client scan")
	return nil
}

// GetManualImport gets files available for manual import from a folder
func (c *SonarrClient) GetManualImport(ctx context.Context, folder string) ([]models.ManualImportItem, error) {
	path := fmt.Sprintf("/api/v3/manualimport?folder=%s&filterExistingFiles=true", folder)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manual import items for folder %s: %w", folder, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manual import items for folder %s, status: %d", folder, resp.StatusCode)
	}

	var manualImportItems []models.ManualImportItem
	if err := json.NewDecoder(resp.Body).Decode(&manualImportItems); err != nil {
		return nil, fmt.Errorf("failed to decode manual import response for folder %s: %w", folder, err)
	}

	c.logger.Debug("Found %d manual import items in folder %s", len(manualImportItems), folder)
	return manualImportItems, nil
}

// ExecuteManualImport executes manual import for the specified files
func (c *SonarrClient) ExecuteManualImport(ctx context.Context, files []models.ManualImportItem, importMode string) error {
	command := map[string]interface{}{
		"name":       "ManualImport",
		"files":      files,
		"importMode": importMode,
	}

	jsonData, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal manual import command: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v3/command", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to execute manual import: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Get response body for better error reporting
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to execute manual import, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	c.logger.Debug("Successfully initiated manual import for %d files", len(files))
	return nil
}

// GetManualImportWithParams gets files available for manual import with additional parameters
func (c *SonarrClient) GetManualImportWithParams(ctx context.Context, folder, downloadID string, seriesID int, filterExisting bool) ([]models.ManualImportItem, error) {
	// Build query parameters similar to golift.io/starr ManualImportParams
	var params []string
	if folder != "" {
		params = append(params, fmt.Sprintf("folder=%s", folder))
	}
	if downloadID != "" {
		params = append(params, fmt.Sprintf("downloadId=%s", downloadID))
	}
	if seriesID > 0 {
		params = append(params, fmt.Sprintf("seriesId=%d", seriesID))
	}
	if filterExisting {
		params = append(params, "filterExistingFiles=true")
	}

	var path string
	if len(params) > 0 {
		path = fmt.Sprintf("/api/v3/manualimport?%s", strings.Join(params, "&"))
	} else {
		path = "/api/v3/manualimport"
	}

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manual import items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manual import items, status: %d", resp.StatusCode)
	}

	var manualImportItems []models.ManualImportItem
	if err := json.NewDecoder(resp.Body).Decode(&manualImportItems); err != nil {
		return nil, fmt.Errorf("failed to decode manual import response: %w", err)
	}

	c.logger.Debug("Found %d manual import items with custom parameters", len(manualImportItems))
	return manualImportItems, nil
}
