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

func TestNewSonarrClient(t *testing.T) {
	cfg := &config.SonarrConfig{
		URL:    "http://test:8989",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	if client == nil {
		t.Error("NewSonarrClient() returned nil")
	}

	sonarrClient, ok := client.(*SonarrClient)
	if !ok {
		t.Error("NewSonarrClient() did not return a *SonarrClient")
	}

	if sonarrClient.GetName() != "sonarr" {
		t.Errorf("Expected name 'sonarr', got '%s'", sonarrClient.GetName())
	}
}

func TestSonarrClient_TestConnection_Success(t *testing.T) {
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

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err != nil {
		t.Errorf("TestConnection() failed: %v", err)
	}
}

func TestSonarrClient_TestConnection_Failure(t *testing.T) {
	// Create a test server that responds with error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "wrong-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err == nil {
		t.Error("Expected TestConnection() to fail with unauthorized")
	}
}

func TestSonarrClient_GetAllSeries_Success(t *testing.T) {
	expectedSeries := []models.Series{
		{MediaItem: models.MediaItem{ID: 1, Title: "Breaking Bad"}},
		{MediaItem: models.MediaItem{ID: 2, Title: "The Wire"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/series" {
			t.Errorf("Expected path '/api/v3/series', got '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedSeries)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	series, err := client.GetAllSeries(ctx)
	if err != nil {
		t.Errorf("GetAllSeries() failed: %v", err)
	}

	if len(series) != 2 {
		t.Errorf("Expected 2 series, got %d", len(series))
	}

	if series[0].ID != 1 || series[0].Title != "Breaking Bad" {
		t.Errorf("Expected series 1 'Breaking Bad', got %d '%s'", series[0].ID, series[0].Title)
	}
}

func TestSonarrClient_GetEpisodesForSeries_Success(t *testing.T) {
	expectedEpisodes := []models.Episode{
		{
			ID:            1,
			SeriesID:      10,
			SeasonNumber:  1,
			EpisodeNumber: 1,
			Title:         "Pilot",
			HasFile:       true,
			EpisodeFileID: intPtr(100),
		},
		{
			ID:            2,
			SeriesID:      10,
			SeasonNumber:  1,
			EpisodeNumber: 2,
			Title:         "Second Episode",
			HasFile:       false,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/episode"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		// Check query parameter
		seriesID := r.URL.Query().Get("seriesId")
		if seriesID != "10" {
			t.Errorf("Expected seriesId '10', got '%s'", seriesID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedEpisodes)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	episodes, err := client.GetEpisodesForSeries(ctx, 10)
	if err != nil {
		t.Errorf("GetEpisodesForSeries() failed: %v", err)
	}

	if len(episodes) != 2 {
		t.Errorf("Expected 2 episodes, got %d", len(episodes))
	}

	if episodes[0].ID != 1 || episodes[0].Title != "Pilot" {
		t.Errorf("Expected episode 1 'Pilot', got %d '%s'", episodes[0].ID, episodes[0].Title)
	}
}

func TestSonarrClient_GetEpisodeFile_Success(t *testing.T) {
	expectedFile := &models.EpisodeFile{
		ID:   100,
		Path: "/path/to/episode.mkv",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/episodefile/100"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedFile)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	file, err := client.GetEpisodeFile(ctx, 100)
	if err != nil {
		t.Errorf("GetEpisodeFile() failed: %v", err)
	}

	if file.ID != 100 || file.Path != "/path/to/episode.mkv" {
		t.Errorf("Expected file 100 '/path/to/episode.mkv', got %d '%s'", file.ID, file.Path)
	}
}

func TestSonarrClient_DeleteEpisodeFile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/episodefile/100"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.DeleteEpisodeFile(ctx, 100)
	if err != nil {
		t.Errorf("DeleteEpisodeFile() failed: %v", err)
	}
}

func TestSonarrClient_UpdateEpisode_Success(t *testing.T) {
	episode := models.Episode{
		ID:            1,
		SeriesID:      10,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         "Updated Title",
		HasFile:       false,
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/episode/1"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}

		callCount++
		if callCount == 1 {
			// First call should be GET to fetch current episode data
			if r.Method != "GET" {
				t.Errorf("Expected GET method on first call, got '%s'", r.Method)
			}
			// Return the current episode data
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(episode)
		} else if callCount == 2 {
			// Second call should be PUT to update the episode
			if r.Method != "PUT" {
				t.Errorf("Expected PUT method on second call, got '%s'", r.Method)
			}
			// Verify that we received a request body
			if r.ContentLength == 0 {
				t.Error("Expected request body, got empty body")
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(episode)
		} else {
			t.Errorf("Unexpected third call to UpdateEpisode")
		}
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.UpdateEpisode(ctx, episode)
	if err != nil {
		t.Errorf("UpdateEpisode() failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected exactly 2 calls, got %d", callCount)
	}
}

func TestSonarrClient_TriggerRefresh_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/command"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"name":"RefreshSeries"}`))
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.TriggerRefresh(ctx)
	if err != nil {
		t.Errorf("TriggerRefresh() failed: %v", err)
	}
}

func TestSonarrClient_HTTPError(t *testing.T) {
	// Server that returns 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	// Test various operations should fail
	_, err := client.GetAllSeries(ctx)
	if err == nil {
		t.Error("Expected GetAllSeries() to fail with HTTP error")
	}

	_, err = client.GetEpisodesForSeries(ctx, 1)
	if err == nil {
		t.Error("Expected GetEpisodesForSeries() to fail with HTTP error")
	}

	err = client.DeleteEpisodeFile(ctx, 100)
	if err == nil {
		t.Error("Expected DeleteEpisodeFile() to fail with HTTP error")
	}
}

func TestSonarrClient_Timeout(t *testing.T) {
	// Server that doesn't respond quickly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	// Very short timeout
	client := NewSonarrClient(cfg, 10*time.Millisecond, logger)
	ctx := context.Background()

	err := client.TestConnection(ctx)
	if err == nil {
		t.Error("Expected TestConnection() to timeout")
	}
}

func TestSonarrClient_GetAllMovies(t *testing.T) {
	cfg := &config.SonarrConfig{
		URL:    "http://test:8989",
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	// GetAllMovies should return error for Sonarr (it's Radarr specific)
	movies, err := client.GetAllMovies(ctx)
	if err == nil {
		t.Error("Expected GetAllMovies() to return error for Sonarr")
	}
	if movies != nil {
		t.Error("Expected GetAllMovies() to return nil movies for Sonarr")
	}
}

func TestSonarrClient_RemoveFromQueue_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/queue/12345"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got '%s'", r.Method)
		}

		// Check query parameters
		removeFromClient := r.URL.Query().Get("removeFromClient")
		if removeFromClient != "true" {
			t.Errorf("Expected removeFromClient 'true', got '%s'", removeFromClient)
		}

		blocklist := r.URL.Query().Get("blocklist")
		if blocklist != "false" {
			t.Errorf("Expected blocklist 'false', got '%s'", blocklist)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.RemoveFromQueue(ctx, 12345, true)
	if err != nil {
		t.Errorf("RemoveFromQueue() failed: %v", err)
	}
}

func TestSonarrClient_RemoveFromQueue_NotFound_Success(t *testing.T) {
	// Test that 404 responses are treated as successful (item already removed)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/queue/12345"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got '%s'", r.Method)
		}

		// Return 404 - item not found (already removed)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.RemoveFromQueue(ctx, 12345, false)
	// 404 should NOT be treated as an error - it means the item is already gone
	if err != nil {
		t.Errorf("RemoveFromQueue() should not fail on 404, but got: %v", err)
	}
}

func TestSonarrClient_RemoveFromQueue_Error(t *testing.T) {
	// Test that other HTTP errors are still treated as failures
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v3/queue/12345"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path '%s', got '%s'", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got '%s'", r.Method)
		}

		// Return 500 - server error (should still be treated as error)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.SonarrConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}
	logger := &mockLogger{}

	client := NewSonarrClient(cfg, 30*time.Second, logger)
	ctx := context.Background()

	err := client.RemoveFromQueue(ctx, 12345, false)
	// 500 should still be treated as an error
	if err == nil {
		t.Error("RemoveFromQueue() should fail on 500, but didn't return error")
	}
}
