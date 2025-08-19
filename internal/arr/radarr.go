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

// RadarrClient implements the Client interface for Radarr API
type RadarrClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     Logger
}

// NewRadarrClient creates a new Radarr client
func NewRadarrClient(cfg *config.RadarrConfig, timeout time.Duration, logger Logger) Client {
	return &RadarrClient{
		baseURL: strings.TrimRight(cfg.URL, "/"),
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// GetName returns the service name
func (c *RadarrClient) GetName() string {
	return "radarr"
}

// TestConnection verifies the connection to Radarr
func (c *RadarrClient) TestConnection(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "GET", "/api/v3/system/status", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Radarr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Radarr returned status %d", resp.StatusCode)
	}

	c.logger.Info("✅ Successfully connected to Radarr")
	return nil
}

// GetAllSeries is not applicable for Radarr (returns error)
func (c *RadarrClient) GetAllSeries(ctx context.Context) ([]models.Series, error) {
	return nil, fmt.Errorf("GetAllSeries is not supported by Radarr client")
}

// GetAllMovies returns all movies from Radarr
func (c *RadarrClient) GetAllMovies(ctx context.Context) ([]models.Movie, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v3/movie", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch movies, status: %d", resp.StatusCode)
	}

	var movies []models.Movie
	if err := json.NewDecoder(resp.Body).Decode(&movies); err != nil {
		return nil, fmt.Errorf("failed to decode movies response: %w", err)
	}

	c.logger.Debug("Fetched %d movies from Radarr", len(movies))
	return movies, nil
}

// GetEpisodesForSeries is not applicable for Radarr (returns error)
func (c *RadarrClient) GetEpisodesForSeries(ctx context.Context, seriesID int) ([]models.Episode, error) {
	return nil, fmt.Errorf("GetEpisodesForSeries is not supported by Radarr client")
}

// GetEpisodeFile is not applicable for Radarr (returns error)
func (c *RadarrClient) GetEpisodeFile(ctx context.Context, fileID int) (*models.EpisodeFile, error) {
	return nil, fmt.Errorf("GetEpisodeFile is not supported by Radarr client")
}

// DeleteEpisodeFile is not applicable for Radarr (returns error)
func (c *RadarrClient) DeleteEpisodeFile(ctx context.Context, fileID int) error {
	return fmt.Errorf("DeleteEpisodeFile is not supported by Radarr client")
}

// UpdateEpisode is not applicable for Radarr (returns error)
func (c *RadarrClient) UpdateEpisode(ctx context.Context, episode models.Episode) error {
	return fmt.Errorf("UpdateEpisode is not supported by Radarr client")
}

// GetMovieFile returns movie file details
func (c *RadarrClient) GetMovieFile(ctx context.Context, fileID int) (*models.MovieFile, error) {
	path := fmt.Sprintf("/api/v3/moviefile/%d", fileID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie file %d: %w", fileID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("movie file %d not found", fileID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch movie file %d, status: %d", fileID, resp.StatusCode)
	}

	var movieFile models.MovieFile
	if err := json.NewDecoder(resp.Body).Decode(&movieFile); err != nil {
		return nil, fmt.Errorf("failed to decode movie file response for %d: %w", fileID, err)
	}

	return &movieFile, nil
}

// DeleteMovieFile deletes a movie file record
func (c *RadarrClient) DeleteMovieFile(ctx context.Context, fileID int) error {
	path := fmt.Sprintf("/api/v3/moviefile/%d", fileID)
	resp, err := c.makeRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete movie file %d: %w", fileID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete movie file %d, status: %d", fileID, resp.StatusCode)
	}

	c.logger.Debug("Successfully deleted movie file %d", fileID)
	return nil
}

// UpdateMovie updates a movie's metadata
func (c *RadarrClient) UpdateMovie(ctx context.Context, movie models.Movie) error {
	// Reset the file reference
	movie.HasFile = false
	movie.MovieFileID = nil

	path := fmt.Sprintf("/api/v3/movie/%d", movie.ID)

	// Create a minimal update payload
	updateData := map[string]interface{}{
		"hasFile":     false,
		"movieFileId": nil,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal movie update: %w", err)
	}

	resp, err := c.makeRequest(ctx, "PUT", path, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to update movie %d: %w", movie.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update movie %d, status: %d", movie.ID, resp.StatusCode)
	}

	c.logger.Debug("Successfully updated movie %d", movie.ID)
	return nil
}

// TriggerRefresh triggers a missing movie search
func (c *RadarrClient) TriggerRefresh(ctx context.Context) error {
	command := map[string]string{
		"name": "MissingMoviesSearch",
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

// makeRequest makes an HTTP request to the Radarr API
func (c *RadarrClient) makeRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
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
