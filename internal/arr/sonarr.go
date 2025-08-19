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
	// Reset the file reference
	episode.HasFile = false
	episode.EpisodeFileID = nil

	path := fmt.Sprintf("/api/v3/episode/%d", episode.ID)

	// Create a minimal update payload
	updateData := map[string]interface{}{
		"hasFile":       false,
		"episodeFileId": nil,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal episode update: %w", err)
	}

	resp, err := c.makeRequest(ctx, "PUT", path, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to update episode %d: %w", episode.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update episode %d, status: %d", episode.ID, resp.StatusCode)
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
