package arr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/pkg/models"
)

func TestNewRadarrClient(t *testing.T) {
	cfg := &config.RadarrConfig{
		URL:    "http://test:7878",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	if client == nil {
		t.Error("NewRadarrClient() returned nil")
	}

	radarrClient, ok := client.(*RadarrClient)
	if !ok {
		t.Error("NewRadarrClient() did not return a *RadarrClient")
	}

	if radarrClient.GetName() != "radarr" {
		t.Errorf("Expected name 'radarr', got '%s'", radarrClient.GetName())
	}
}

func TestRadarrClient_TestConnection_Success(t *testing.T) {
	// Create a test server that responds with OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/system/status" {
			t.Errorf("Expected path '/api/v3/system/status', got '%s'", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("Expected API key 'test-key', got '%s'", r.Header.Get("X-Api-Key"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"3.0.0"}`))
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err != nil {
		t.Errorf("TestConnection() failed: %v", err)
	}
}

func TestRadarrClient_TestConnection_Failure(t *testing.T) {
	// Create a test server that responds with error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "wrong-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err == nil {
		t.Error("Expected TestConnection() to fail with unauthorized")
	}
}

func TestRadarrClient_GetAllMovies_Success(t *testing.T) {
	expectedMovies := []models.Movie{
		{MediaItem: models.MediaItem{ID: 1, Title: "The Matrix"}},
		{MediaItem: models.MediaItem{ID: 2, Title: "Inception"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/movie" {
			t.Errorf("Expected path '/api/v3/movie', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedMovies)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	movies, err := client.GetAllMovies(ctx)
	if err != nil {
		t.Errorf("GetAllMovies() failed: %v", err)
	}

	if len(movies) != 2 {
		t.Errorf("Expected 2 movies, got %d", len(movies))
	}

	if movies[0].ID != 1 || movies[0].Title != "The Matrix" {
		t.Errorf("Expected movie 1 'The Matrix', got %d '%s'", movies[0].ID, movies[0].Title)
	}
}

func TestRadarrClient_GetMovieFile_Success(t *testing.T) {
	expectedFile := &models.MovieFile{
		ID:   100,
		Path: "/path/to/movie.mkv",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/moviefile/100"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedFile)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	file, err := client.GetMovieFile(ctx, 100)
	if err != nil {
		t.Errorf("GetMovieFile() failed: %v", err)
	}

	if file.ID != 100 || file.Path != "/path/to/movie.mkv" {
		t.Errorf("Expected file 100 '/path/to/movie.mkv', got %d '%s'", file.ID, file.Path)
	}
}

func TestRadarrClient_GetMovieFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/moviefile/404"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	_, err := client.GetMovieFile(ctx, 404)
	if err == nil {
		t.Error("Expected GetMovieFile() to fail with not found")
	}
}

func TestRadarrClient_DeleteMovieFile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/moviefile/100"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.DeleteMovieFile(ctx, 100)
	if err != nil {
		t.Errorf("DeleteMovieFile() failed: %v", err)
	}
}

func TestRadarrClient_DeleteMovieFile_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/moviefile/100"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.DeleteMovieFile(ctx, 100)
	if err != nil {
		t.Errorf("DeleteMovieFile() failed: %v", err)
	}
}

func TestRadarrClient_UpdateMovie_Success(t *testing.T) {
	movie := models.Movie{
		MediaItem: models.MediaItem{
			ID:    1,
			Title: "Updated Movie",
		},
		HasFile:     false,
		MovieFileID: nil,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/movie/1"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got '%s'", r.Method)
		}

		// Verify that we received a request body (but don't parse it for simplicity)
		if r.ContentLength == 0 {
			t.Error("Expected request body, got empty body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movie)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.UpdateMovie(ctx, movie)
	if err != nil {
		t.Errorf("UpdateMovie() failed: %v", err)
	}
}

func TestRadarrClient_TriggerRefresh_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/command"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"name":"MissingMoviesSearch"}`))
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TriggerRefresh(ctx)
	if err != nil {
		t.Errorf("TriggerRefresh() failed: %v", err)
	}
}

func TestRadarrClient_TriggerRefresh_StatusOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/command"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":1,"name":"MissingMoviesSearch"}`))
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TriggerRefresh(ctx)
	if err != nil {
		t.Errorf("TriggerRefresh() failed: %v", err)
	}
}

func TestRadarrClient_HTTPError(t *testing.T) {
	// Server that returns 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	// Test various operations should fail
	_, err := client.GetAllMovies(ctx)
	if err == nil {
		t.Error("Expected GetAllMovies() to fail with HTTP error")
	}

	_, err = client.GetMovieFile(ctx, 100)
	if err == nil {
		t.Error("Expected GetMovieFile() to fail with HTTP error")
	}

	err = client.DeleteMovieFile(ctx, 100)
	if err == nil {
		t.Error("Expected DeleteMovieFile() to fail with HTTP error")
	}

	movie := models.Movie{MediaItem: models.MediaItem{ID: 1, Title: "Test"}}
	err = client.UpdateMovie(ctx, movie)
	if err == nil {
		t.Error("Expected UpdateMovie() to fail with HTTP error")
	}

	err = client.TriggerRefresh(ctx)
	if err == nil {
		t.Error("Expected TriggerRefresh() to fail with HTTP error")
	}
}

func TestRadarrClient_Timeout(t *testing.T) {
	// Server that doesn't respond quickly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	// Very short timeout
	client := NewRadarrClient(cfg, 10*time.Millisecond, logger)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err == nil {
		t.Error("Expected TestConnection() to timeout")
	}
}

func TestRadarrClient_GetAllSeries(t *testing.T) {
	cfg := &config.RadarrConfig{
		URL:    "http://test:7878",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	// GetAllSeries should return error for Radarr (it's Sonarr specific)
	series, err := client.GetAllSeries(ctx)
	if err == nil {
		t.Error("Expected GetAllSeries() to return error for Radarr")
	}
	if series != nil {
		t.Error("Expected GetAllSeries() to return nil series for Radarr")
	}
}

func TestRadarrClient_GetEpisodesForSeries(t *testing.T) {
	cfg := &config.RadarrConfig{
		URL:    "http://test:7878",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	// GetEpisodesForSeries should return error for Radarr (it's Sonarr specific)
	episodes, err := client.GetEpisodesForSeries(ctx, 1)
	if err == nil {
		t.Error("Expected GetEpisodesForSeries() to return error for Radarr")
	}
	if episodes != nil {
		t.Error("Expected GetEpisodesForSeries() to return nil episodes for Radarr")
	}
}

func TestRadarrClient_GetEpisodeFile(t *testing.T) {
	cfg := &config.RadarrConfig{
		URL:    "http://test:7878",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	// GetEpisodeFile should return error for Radarr (it's Sonarr specific)
	episodeFile, err := client.GetEpisodeFile(ctx, 1)
	if err == nil {
		t.Error("Expected GetEpisodeFile() to return error for Radarr")
	}
	if episodeFile != nil {
		t.Error("Expected GetEpisodeFile() to return nil episode file for Radarr")
	}
}

func TestRadarrClient_DeleteEpisodeFile(t *testing.T) {
	cfg := &config.RadarrConfig{
		URL:    "http://test:7878",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	// DeleteEpisodeFile should return error for Radarr (it's Sonarr specific)
	err := client.DeleteEpisodeFile(ctx, 1)
	if err == nil {
		t.Error("Expected DeleteEpisodeFile() to return error for Radarr")
	}
}

func TestRadarrClient_UpdateEpisode(t *testing.T) {
	cfg := &config.RadarrConfig{
		URL:    "http://test:7878",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	episode := models.Episode{ID: 1, Title: "Test Episode"}

	// UpdateEpisode should return error for Radarr (it's Sonarr specific)
	err := client.UpdateEpisode(ctx, episode)
	if err == nil {
		t.Error("Expected UpdateEpisode() to return error for Radarr")
	}
}

func TestRadarrClient_makeRequest_URLTrimming(t *testing.T) {
	// Test that trailing slashes are properly trimmed from baseURL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"3.0.0"}`))
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL + "/", // Note the trailing slash
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err != nil {
		t.Errorf("TestConnection() failed with trailing slash URL: %v", err)
	}
}

func TestRadarrClient_JSON_InvalidResponse(t *testing.T) {
	// Server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	cfg := &config.RadarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewRadarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	_, err := client.GetAllMovies(ctx)
	if err == nil {
		t.Error("Expected GetAllMovies() to fail with invalid JSON")
	}

	_, err = client.GetMovieFile(ctx, 100)
	if err == nil {
		t.Error("Expected GetMovieFile() to fail with invalid JSON")
	}
}
