package models

import (
	"testing"
)

func TestMediaItem(t *testing.T) {
	item := MediaItem{
		ID:    123,
		Title: "Test Media",
		Path:  "/path/to/media",
	}

	if item.ID != 123 {
		t.Errorf("Expected ID 123, got %d", item.ID)
	}
	if item.Title != "Test Media" {
		t.Errorf("Expected Title 'Test Media', got '%s'", item.Title)
	}
	if item.Path != "/path/to/media" {
		t.Errorf("Expected Path '/path/to/media', got '%s'", item.Path)
	}
}

func TestSeries(t *testing.T) {
	series := Series{
		MediaItem: MediaItem{
			ID:    456,
			Title: "Breaking Bad",
			Path:  "/tv/breaking-bad",
		},
		SeasonCount: 5,
	}

	if series.ID != 456 {
		t.Errorf("Expected ID 456, got %d", series.ID)
	}
	if series.Title != "Breaking Bad" {
		t.Errorf("Expected Title 'Breaking Bad', got '%s'", series.Title)
	}
	if series.SeasonCount != 5 {
		t.Errorf("Expected SeasonCount 5, got %d", series.SeasonCount)
	}
}

func TestMovie(t *testing.T) {
	movie := Movie{
		MediaItem: MediaItem{
			ID:    789,
			Title: "The Matrix",
			Path:  "/movies/the-matrix",
		},
		Year: 1999,
	}

	if movie.ID != 789 {
		t.Errorf("Expected ID 789, got %d", movie.ID)
	}
	if movie.Title != "The Matrix" {
		t.Errorf("Expected Title 'The Matrix', got '%s'", movie.Title)
	}
	if movie.Year != 1999 {
		t.Errorf("Expected Year 1999, got %d", movie.Year)
	}
}

func TestEpisode(t *testing.T) {
	fileID := 100
	episode := Episode{
		ID:            1,
		SeriesID:      10,
		SeasonNumber:  2,
		EpisodeNumber: 5,
		Title:         "Test Episode",
		HasFile:       true,
		EpisodeFileID: &fileID,
	}

	if episode.ID != 1 {
		t.Errorf("Expected ID 1, got %d", episode.ID)
	}
	if episode.SeriesID != 10 {
		t.Errorf("Expected SeriesID 10, got %d", episode.SeriesID)
	}
	if episode.SeasonNumber != 2 {
		t.Errorf("Expected SeasonNumber 2, got %d", episode.SeasonNumber)
	}
	if episode.EpisodeNumber != 5 {
		t.Errorf("Expected EpisodeNumber 5, got %d", episode.EpisodeNumber)
	}
	if episode.Title != "Test Episode" {
		t.Errorf("Expected Title 'Test Episode', got '%s'", episode.Title)
	}
	if !episode.HasFile {
		t.Error("Expected HasFile to be true")
	}
	if episode.EpisodeFileID == nil {
		t.Error("Expected EpisodeFileID to not be nil")
	} else if *episode.EpisodeFileID != 100 {
		t.Errorf("Expected EpisodeFileID 100, got %d", *episode.EpisodeFileID)
	}
}

func TestEpisodeWithoutFile(t *testing.T) {
	episode := Episode{
		ID:            2,
		SeriesID:      10,
		SeasonNumber:  2,
		EpisodeNumber: 6,
		Title:         "Episode Without File",
		HasFile:       false,
		EpisodeFileID: nil,
	}

	if episode.HasFile {
		t.Error("Expected HasFile to be false")
	}
	if episode.EpisodeFileID != nil {
		t.Error("Expected EpisodeFileID to be nil")
	}
}

func TestEpisodeFile(t *testing.T) {
	file := EpisodeFile{
		ID:   200,
		Path: "/path/to/episode.mkv",
	}

	if file.ID != 200 {
		t.Errorf("Expected ID 200, got %d", file.ID)
	}
	if file.Path != "/path/to/episode.mkv" {
		t.Errorf("Expected Path '/path/to/episode.mkv', got '%s'", file.Path)
	}
}

func TestMovieFile(t *testing.T) {
	file := MovieFile{
		ID:      300,
		Path:    "/path/to/movie.mkv",
		MovieID: 50,
	}

	if file.ID != 300 {
		t.Errorf("Expected ID 300, got %d", file.ID)
	}
	if file.Path != "/path/to/movie.mkv" {
		t.Errorf("Expected Path '/path/to/movie.mkv', got '%s'", file.Path)
	}
	if file.MovieID != 50 {
		t.Errorf("Expected MovieID 50, got %d", file.MovieID)
	}
}

func TestCleanupStats(t *testing.T) {
	stats := CleanupStats{
		TotalItemsChecked: 100,
		MissingFiles:      15,
		DeletedRecords:    10,
		Errors:            2,
	}

	if stats.TotalItemsChecked != 100 {
		t.Errorf("Expected TotalItemsChecked 100, got %d", stats.TotalItemsChecked)
	}
	if stats.MissingFiles != 15 {
		t.Errorf("Expected MissingFiles 15, got %d", stats.MissingFiles)
	}
	if stats.DeletedRecords != 10 {
		t.Errorf("Expected DeletedRecords 10, got %d", stats.DeletedRecords)
	}
	if stats.Errors != 2 {
		t.Errorf("Expected Errors 2, got %d", stats.Errors)
	}
}

func TestCleanupResult(t *testing.T) {
	stats := CleanupStats{
		TotalItemsChecked: 50,
		MissingFiles:      5,
		DeletedRecords:    3,
		Errors:            0,
	}

	messages := []string{"File not found", "Operation completed"}

	result := CleanupResult{
		Stats:    stats,
		Messages: messages,
		Success:  true,
	}

	if result.Stats.TotalItemsChecked != 50 {
		t.Errorf("Expected TotalItemsChecked 50, got %d", result.Stats.TotalItemsChecked)
	}
	if len(result.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result.Messages))
	}
	if result.Messages[0] != "File not found" {
		t.Errorf("Expected first message 'File not found', got '%s'", result.Messages[0])
	}
	if !result.Success {
		t.Error("Expected Success to be true")
	}
}

func TestCleanupResultFailure(t *testing.T) {
	stats := CleanupStats{
		TotalItemsChecked: 30,
		MissingFiles:      8,
		DeletedRecords:    5,
		Errors:            3,
	}

	result := CleanupResult{
		Stats:    stats,
		Messages: []string{"Error occurred", "Another error"},
		Success:  false,
	}

	if result.Success {
		t.Error("Expected Success to be false")
	}
	if result.Stats.Errors != 3 {
		t.Errorf("Expected 3 errors, got %d", result.Stats.Errors)
	}
}

// Test zero values
func TestZeroValues(t *testing.T) {
	var item MediaItem
	if item.ID != 0 || item.Title != "" || item.Path != "" {
		t.Error("Expected zero values for MediaItem")
	}

	var stats CleanupStats
	if stats.TotalItemsChecked != 0 || stats.MissingFiles != 0 ||
		stats.DeletedRecords != 0 || stats.Errors != 0 {
		t.Error("Expected zero values for CleanupStats")
	}

	var result CleanupResult
	if result.Success || len(result.Messages) != 0 {
		t.Error("Expected zero values for CleanupResult")
	}
}
