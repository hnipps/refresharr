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

// GetMovie returns a single movie by ID from Radarr
func (c *RadarrClient) GetMovie(ctx context.Context, movieID int) (*models.Movie, error) {
	path := fmt.Sprintf("/api/v3/movie/%d", movieID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie %d: %w", movieID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("movie %d not found", movieID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch movie %d, status: %d", movieID, resp.StatusCode)
	}

	var movie models.Movie
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return nil, fmt.Errorf("failed to decode movie response for %d: %w", movieID, err)
	}

	c.logger.Debug("Fetched movie %d from Radarr", movieID)
	return &movie, nil
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
	// First, fetch the current movie data to ensure we have the complete object
	path := fmt.Sprintf("/api/v3/movie/%d", movie.ID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch current movie %d data: %w", movie.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch current movie %d data, status: %d", movie.ID, resp.StatusCode)
	}

	var currentMovie models.Movie
	if err := json.NewDecoder(resp.Body).Decode(&currentMovie); err != nil {
		return fmt.Errorf("failed to decode current movie %d data: %w", movie.ID, err)
	}

	// Update the file reference fields
	currentMovie.HasFile = false
	currentMovie.MovieFileID = nil

	// Marshal the complete movie object
	jsonData, err := json.Marshal(currentMovie)
	if err != nil {
		return fmt.Errorf("failed to marshal movie update: %w", err)
	}

	// Send the PUT request with the complete movie object
	resp, err = c.makeRequest(ctx, "PUT", path, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to update movie %d: %w", movie.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Get response body for better error reporting
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update movie %d, status: %d, response: %s", movie.ID, resp.StatusCode, string(bodyBytes))
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

// GetRootFolders returns all root folders from Radarr
func (c *RadarrClient) GetRootFolders(ctx context.Context) ([]models.RootFolder, error) {
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

	c.logger.Debug("Fetched %d root folders from Radarr", len(rootFolders))
	return rootFolders, nil
}

// GetQualityProfiles returns all quality profiles from Radarr
func (c *RadarrClient) GetQualityProfiles(ctx context.Context) ([]models.QualityProfile, error) {
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

	c.logger.Debug("Fetched %d quality profiles from Radarr", len(qualityProfiles))
	return qualityProfiles, nil
}

// LookupMovieByTMDBID looks up movie information by TMDB ID
func (c *RadarrClient) LookupMovieByTMDBID(ctx context.Context, tmdbID int) (*models.MovieLookup, error) {
	path := fmt.Sprintf("/api/v3/movie/lookup/tmdb?tmdbId=%d", tmdbID)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup movie with TMDB ID %d: %w", tmdbID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("movie with TMDB ID %d not found", tmdbID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to lookup movie with TMDB ID %d, status: %d", tmdbID, resp.StatusCode)
	}

	var movieLookup models.MovieLookup
	if err := json.NewDecoder(resp.Body).Decode(&movieLookup); err != nil {
		return nil, fmt.Errorf("failed to decode movie lookup response for TMDB ID %d: %w", tmdbID, err)
	}

	c.logger.Debug("Successfully looked up movie with TMDB ID %d: %s", tmdbID, movieLookup.Title)
	return &movieLookup, nil
}

// GetMovieByTMDBID returns a movie by TMDB ID if it exists in the collection
func (c *RadarrClient) GetMovieByTMDBID(ctx context.Context, tmdbID int) (*models.Movie, error) {
	// Get all movies and find the one with matching TMDB ID
	movies, err := c.GetAllMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movies to search for TMDB ID %d: %w", tmdbID, err)
	}

	for _, movie := range movies {
		if movie.TMDBID == tmdbID {
			return &movie, nil
		}
	}

	return nil, fmt.Errorf("movie with TMDB ID %d not found in collection", tmdbID)
}

// AddMovie adds a movie to the Radarr collection
func (c *RadarrClient) AddMovie(ctx context.Context, movie models.Movie) (*models.Movie, error) {
	// Marshal the movie object
	jsonData, err := json.Marshal(movie)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal movie for addition: %w", err)
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v3/movie", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to add movie: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Get response body for better error reporting
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add movie, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var addedMovie models.Movie
	if err := json.NewDecoder(resp.Body).Decode(&addedMovie); err != nil {
		return nil, fmt.Errorf("failed to decode added movie response: %w", err)
	}

	c.logger.Info("✅ Successfully added movie: %s (%d) with TMDB ID %d", addedMovie.Title, addedMovie.Year, addedMovie.TMDBID)
	return &addedMovie, nil
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

// AddSeries is not applicable for Radarr (returns error)
func (c *RadarrClient) AddSeries(ctx context.Context, series models.Series) (*models.Series, error) {
	return nil, fmt.Errorf("AddSeries is not supported by Radarr client")
}

// GetSeriesByTVDBID is not applicable for Radarr (returns error)
func (c *RadarrClient) GetSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.Series, error) {
	return nil, fmt.Errorf("GetSeriesByTVDBID is not supported by Radarr client")
}

// LookupSeriesByTVDBID is not applicable for Radarr (returns error)
func (c *RadarrClient) LookupSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.SeriesLookup, error) {
	return nil, fmt.Errorf("LookupSeriesByTVDBID is not supported by Radarr client")
}
