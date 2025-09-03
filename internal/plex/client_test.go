package plex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hnipps/refresharr/internal/config"
)

// Logger interface for testing
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	logs []string
}

func (m *mockLogger) Debug(format string, args ...interface{}) {
	m.logs = append(m.logs, "DEBUG")
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	m.logs = append(m.logs, "INFO")
}

func (m *mockLogger) Warn(format string, args ...interface{}) {
	m.logs = append(m.logs, "WARN")
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	m.logs = append(m.logs, "ERROR")
}

// newTestPlexClient creates a new Plex client for testing
func newTestPlexClient(cfg *config.PlexConfig, timeout time.Duration, logger Logger) *PlexClient {
	return &PlexClient{
		baseURL: cfg.URL,
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: &loggerAdapter{logger},
	}
}

// loggerAdapter adapts test logger to arr.Logger interface
type loggerAdapter struct {
	Logger
}

func TestNewPlexClient(t *testing.T) {
	cfg := &config.PlexConfig{
		URL:   "http://localhost:32400",
		Token: "test-token",
	}
	logger := &mockLogger{}
	
	client := newTestPlexClient(cfg, 30*time.Second, logger)
	
	if client == nil {
		t.Fatal("NewPlexClient returned nil")
	}
	
	if client.baseURL != "http://localhost:32400" {
		t.Errorf("Expected baseURL to be 'http://localhost:32400', got '%s'", client.baseURL)
	}
	
	if client.token != "test-token" {
		t.Errorf("Expected token to be 'test-token', got '%s'", client.token)
	}
}

func TestPlexClient_TestConnection(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  bool
	}{
		{
			name:          "successful connection",
			statusCode:    http.StatusOK,
			responseBody:  `{"MediaContainer": {}}`,
			expectedError: false,
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"errors": [{"code": 1001, "message": "Unauthorized"}]}`,
			expectedError: true,
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error": "Internal Server Error"}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify token is in query params
				if r.URL.Query().Get("X-Plex-Token") != "test-token" {
					t.Errorf("Expected X-Plex-Token query param to be 'test-token', got '%s'", r.URL.Query().Get("X-Plex-Token"))
				}
				
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			cfg := &config.PlexConfig{
				URL:   server.URL,
				Token: "test-token",
			}
			logger := &mockLogger{}
			client := newTestPlexClient(cfg, 30*time.Second, logger)

			err := client.TestConnection(context.Background())

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPlexClient_GetMovieByTMDBID(t *testing.T) {
	tests := []struct {
		name           string
		tmdbID         int
		sectionsResp   string
		movieResp      string
		mediaResp      string
		expectedError  bool
		expectedTitle  string
		expectedAvail  bool
	}{
		{
			name:   "movie found and available",
			tmdbID: 12345,
			sectionsResp: `{
				"MediaContainer": {
					"Directory": [
						{"key": "1", "title": "Movies", "type": "movie"}
					]
				}
			}`,
			movieResp: `{
				"MediaContainer": {
					"Metadata": [
						{
							"key": "/library/metadata/123",
							"title": "Test Movie",
							"year": 2023,
							"guid": "plex://movie/5d776b59ad5437001f79c6f8?lang=en&tmdb://12345"
						}
					]
				}
			}`,
			mediaResp: `{
				"MediaContainer": {
					"Metadata": [
						{
							"Media": [
								{
									"Part": [
										{"key": "1", "file": "/movies/test.mkv"}
									]
								}
							]
						}
					]
				}
			}`,
			expectedError: false,
			expectedTitle: "Test Movie",
			expectedAvail: true,
		},
		{
			name:   "movie found but not available",
			tmdbID: 12345,
			sectionsResp: `{
				"MediaContainer": {
					"Directory": [
						{"key": "1", "title": "Movies", "type": "movie"}
					]
				}
			}`,
			movieResp: `{
				"MediaContainer": {
					"Metadata": [
						{
							"key": "/library/metadata/123",
							"title": "Test Movie",
							"year": 2023,
							"guid": "plex://movie/5d776b59ad5437001f79c6f8?lang=en&tmdb://12345"
						}
					]
				}
			}`,
			mediaResp: `{
				"MediaContainer": {
					"Metadata": [
						{
							"Media": []
						}
					]
				}
			}`,
			expectedError: false,
			expectedTitle: "Test Movie",
			expectedAvail: false,
		},
		{
			name:   "movie not found",
			tmdbID: 99999,
			sectionsResp: `{
				"MediaContainer": {
					"Directory": [
						{"key": "1", "title": "Movies", "type": "movie"}
					]
				}
			}`,
			movieResp: `{
				"MediaContainer": {
					"Metadata": []
				}
			}`,
			mediaResp:     `{}`,
			expectedError: true,
		},
		{
			name:   "no movie sections",
			tmdbID: 12345,
			sectionsResp: `{
				"MediaContainer": {
					"Directory": [
						{"key": "1", "title": "TV Shows", "type": "show"}
					]
				}
			}`,
			movieResp:     `{}`,
			mediaResp:     `{}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/library/sections":
					w.Write([]byte(tt.sectionsResp))
				case "/library/sections/1/all":
					w.Write([]byte(tt.movieResp))
				case "/library/metadata/123":
					w.Write([]byte(tt.mediaResp))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			cfg := &config.PlexConfig{
				URL:   server.URL,
				Token: "test-token",
			}
			logger := &mockLogger{}
			client := newTestPlexClient(cfg, 30*time.Second, logger)

			movie, err := client.GetMovieByTMDBID(context.Background(), tt.tmdbID)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if movie == nil {
				t.Fatal("Expected movie but got nil")
			}

			if movie.Title != tt.expectedTitle {
				t.Errorf("Expected title '%s', got '%s'", tt.expectedTitle, movie.Title)
			}

			if movie.Available != tt.expectedAvail {
				t.Errorf("Expected available %t, got %t", tt.expectedAvail, movie.Available)
			}
		})
	}
}

func TestPlexClient_checkMovieAvailability(t *testing.T) {
	tests := []struct {
		name          string
		mediaResp     string
		statusCode    int
		expectedAvail bool
		expectedError bool
	}{
		{
			name: "movie available with media parts",
			mediaResp: `{
				"MediaContainer": {
					"Metadata": [
						{
							"Media": [
								{
									"Part": [
										{"key": "1", "file": "/movies/test.mkv"}
									]
								}
							]
						}
					]
				}
			}`,
			statusCode:    http.StatusOK,
			expectedAvail: true,
			expectedError: false,
		},
		{
			name: "movie not available - no media parts",
			mediaResp: `{
				"MediaContainer": {
					"Metadata": [
						{
							"Media": []
						}
					]
				}
			}`,
			statusCode:    http.StatusOK,
			expectedAvail: false,
			expectedError: false,
		},
		{
			name: "movie not available - no metadata",
			mediaResp: `{
				"MediaContainer": {
					"Metadata": []
				}
			}`,
			statusCode:    http.StatusOK,
			expectedAvail: false,
			expectedError: false,
		},
		{
			name:          "server error",
			mediaResp:     `{"error": "Internal Server Error"}`,
			statusCode:    http.StatusInternalServerError,
			expectedAvail: false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.mediaResp))
			}))
			defer server.Close()

			cfg := &config.PlexConfig{
				URL:   server.URL,
				Token: "test-token",
			}
			logger := &mockLogger{}
			client := newTestPlexClient(cfg, 30*time.Second, logger)

			available, err := client.checkMovieAvailability(context.Background(), "/test/path")

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if available != tt.expectedAvail {
				t.Errorf("Expected available %t, got %t", tt.expectedAvail, available)
			}
		})
	}
}

func TestPlexClient_getLibrarySections(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		statusCode     int
		expectedCount  int
		expectedError  bool
	}{
		{
			name: "successful sections retrieval",
			responseBody: `{
				"MediaContainer": {
					"Directory": [
						{"key": "1", "title": "Movies", "type": "movie"},
						{"key": "2", "title": "TV Shows", "type": "show"}
					]
				}
			}`,
			statusCode:    http.StatusOK,
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "empty sections",
			responseBody: `{
				"MediaContainer": {
					"Directory": []
				}
			}`,
			statusCode:    http.StatusOK,
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "server error",
			responseBody:  `{"error": "Internal Server Error"}`,
			statusCode:    http.StatusInternalServerError,
			expectedCount: 0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/library/sections" {
					t.Errorf("Expected path '/library/sections', got '%s'", r.URL.Path)
				}
				
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			cfg := &config.PlexConfig{
				URL:   server.URL,
				Token: "test-token",
			}
			logger := &mockLogger{}
			client := newTestPlexClient(cfg, 30*time.Second, logger)

			sections, err := client.getLibrarySections(context.Background())

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if len(sections) != tt.expectedCount {
				t.Errorf("Expected %d sections, got %d", tt.expectedCount, len(sections))
			}
		})
	}
}

func TestPlexClient_makeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept header to be 'application/json', got '%s'", r.Header.Get("Accept"))
		}
		
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header to be 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}
		
		// Verify token in query params
		if r.URL.Query().Get("X-Plex-Token") != "test-token" {
			t.Errorf("Expected X-Plex-Token query param to be 'test-token', got '%s'", r.URL.Query().Get("X-Plex-Token"))
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	cfg := &config.PlexConfig{
		URL:   server.URL,
		Token: "test-token",
	}
	logger := &mockLogger{}
	client := newTestPlexClient(cfg, 30*time.Second, logger)

	resp, err := client.makeRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
}