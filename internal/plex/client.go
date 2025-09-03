package plex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hnipps/refresharr/internal/arr"
	"github.com/hnipps/refresharr/internal/config"
)

// PlexClient implements a client for Plex Media Server API
type PlexClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     arr.Logger
}

// PlexMovie represents a movie in Plex
type PlexMovie struct {
	Key        string      `json:"key"`
	Title      string      `json:"title"`
	Year       int         `json:"year"`
	GUID       string      `json:"guid"`
	Available  bool        `json:"-"` // Computed field
	MediaParts []MediaPart `json:"-"` // Media parts for availability check
}

// MediaPart represents a media part in Plex
type MediaPart struct {
	Key  string `json:"key"`
	File string `json:"file"`
}

// PlexResponse represents the standard Plex API response structure
type PlexResponse struct {
	MediaContainer struct {
		Size     int         `json:"size"`
		Metadata []PlexMovie `json:"Metadata"`
	} `json:"MediaContainer"`
}

// PlexMediaResponse represents media details response
type PlexMediaResponse struct {
	MediaContainer struct {
		Metadata []struct {
			Media []struct {
				Part []MediaPart `json:"Part"`
			} `json:"Media"`
		} `json:"Metadata"`
	} `json:"MediaContainer"`
}

// NewPlexClient creates a new Plex client
func NewPlexClient(cfg *config.PlexConfig, timeout time.Duration, logger arr.Logger) *PlexClient {
	return &PlexClient{
		baseURL: strings.TrimRight(cfg.URL, "/"),
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// TestConnection verifies the connection to Plex
func (c *PlexClient) TestConnection(ctx context.Context) error {
	resp, err := c.makeRequest(ctx, "GET", "/", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to Plex: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Plex returned status %d", resp.StatusCode)
	}

	c.logger.Info("✅ Successfully connected to Plex")
	return nil
}

// GetMovieByTMDBID searches for a movie by TMDB ID in Plex
func (c *PlexClient) GetMovieByTMDBID(ctx context.Context, tmdbID int) (*PlexMovie, error) {
	// Search for the movie using TMDB GUID
	tmdbGUID := fmt.Sprintf("tmdb://%d", tmdbID)

	// Get all movies from library sections
	sections, err := c.getLibrarySections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get library sections: %w", err)
	}

	// Search in movie sections
	for _, section := range sections {
		if section.Type == "movie" {
			movie, err := c.searchMovieInSection(ctx, section.Key, tmdbGUID, tmdbID)
			if err != nil {
				c.logger.Debug("Error searching in section %s: %v", section.Title, err)
				continue
			}
			if movie != nil {
				return movie, nil
			}
		}
	}

	return nil, fmt.Errorf("movie with TMDB ID %d not found in Plex", tmdbID)
}

// LibrarySection represents a Plex library section
type LibrarySection struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// LibrarySectionsResponse represents the library sections response
type LibrarySectionsResponse struct {
	MediaContainer struct {
		Directory []LibrarySection `json:"Directory"`
	} `json:"MediaContainer"`
}

// getLibrarySections returns all library sections
func (c *PlexClient) getLibrarySections(ctx context.Context) ([]LibrarySection, error) {
	resp, err := c.makeRequest(ctx, "GET", "/library/sections", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get library sections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get library sections, status: %d", resp.StatusCode)
	}

	var sectionsResp LibrarySectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&sectionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode library sections response: %w", err)
	}

	return sectionsResp.MediaContainer.Directory, nil
}

// searchMovieInSection searches for a movie in a specific library section
func (c *PlexClient) searchMovieInSection(ctx context.Context, sectionKey, tmdbGUID string, tmdbID int) (*PlexMovie, error) {
	// First try searching by GUID
	path := fmt.Sprintf("/library/sections/%s/all", sectionKey)
	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search section %s: %w", sectionKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to search section %s, status: %d", sectionKey, resp.StatusCode)
	}

	var plexResp PlexResponse
	if err := json.NewDecoder(resp.Body).Decode(&plexResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Look for movie with matching TMDB GUID
	for _, movie := range plexResp.MediaContainer.Metadata {
		if strings.Contains(movie.GUID, fmt.Sprintf("tmdb://%d", tmdbID)) {
			// Get media details to check availability
			available, err := c.checkMovieAvailability(ctx, movie.Key)
			if err != nil {
				c.logger.Warn("Failed to check availability for movie %s: %v", movie.Title, err)
				available = false // Assume not available if we can't check
			}
			movie.Available = available
			return &movie, nil
		}
	}

	return nil, nil // Not found in this section
}

// checkMovieAvailability checks if a movie's media files are available
func (c *PlexClient) checkMovieAvailability(ctx context.Context, movieKey string) (bool, error) {
	resp, err := c.makeRequest(ctx, "GET", movieKey, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get movie details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to get movie details, status: %d", resp.StatusCode)
	}

	var mediaResp PlexMediaResponse
	if err := json.NewDecoder(resp.Body).Decode(&mediaResp); err != nil {
		return false, fmt.Errorf("failed to decode media response: %w", err)
	}

	// Check if movie has any media parts
	if len(mediaResp.MediaContainer.Metadata) == 0 {
		return false, nil
	}

	for _, metadata := range mediaResp.MediaContainer.Metadata {
		for _, media := range metadata.Media {
			if len(media.Part) > 0 {
				// Movie has media parts, consider it available
				return true, nil
			}
		}
	}

	return false, nil
}

// makeRequest makes an HTTP request to the Plex API
func (c *PlexClient) makeRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	fullURL := c.baseURL + path

	// Parse URL to add token parameter
	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add token to query parameters
	q := u.Query()
	q.Set("X-Plex-Token", c.token)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Making %s request to %s", method, u.String())

	return c.httpClient.Do(req)
}
