package arr

import (
	"testing"

	"github.com/hnipps/refresharr/pkg/models"
)

func TestCleanupServiceImpl_deduplicateMissingFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    []models.MissingFileEntry
		expected []models.MissingFileEntry
	}{
		{
			name: "deduplicate movie with TMDB ID - prioritize real FileID",
			input: []models.MissingFileEntry{
				{
					MediaType:         "movie",
					MediaName:         "All About Eve",
					FilePath:          "/mnt/data/media/movies/All About Eve (1950) [tmdb-705]/All About Eve (1950) [tmdb-705] - [Bluray-1080p][FLAC 1.0][x264]-PTer.mkv",
					FileID:            0, // Broken symlink entry
					ProcessedAt:       "2025-09-02T17:43:55Z",
					TMDBID:            705,
					AddedToCollection: false,
				},
				{
					MediaType:   "movie",
					MediaName:   "All About Eve",
					FilePath:    "/mnt/data/media/movies/All About Eve (1950) [tmdb-705]/All About Eve (1950) [tmdb-705] - [Bluray-1080p][FLAC 1.0][x264]-PTer.mkv",
					FileID:      1607, // Real file ID
					ProcessedAt: "2025-09-02T17:44:18Z",
					TMDBID:      705,
				},
			},
			expected: []models.MissingFileEntry{
				{
					MediaType:   "movie",
					MediaName:   "All About Eve",
					FilePath:    "/mnt/data/media/movies/All About Eve (1950) [tmdb-705]/All About Eve (1950) [tmdb-705] - [Bluray-1080p][FLAC 1.0][x264]-PTer.mkv",
					FileID:      1607, // Should keep the one with real FileID
					ProcessedAt: "2025-09-02T17:44:18Z",
					TMDBID:      705,
				},
			},
		},
		{
			name: "deduplicate movie without TMDB ID - use file path",
			input: []models.MissingFileEntry{
				{
					MediaType:   "movie",
					MediaName:   "Test Movie",
					FilePath:    "/mnt/data/media/movies/Test Movie/test.mkv",
					FileID:      100,
					ProcessedAt: "2025-09-02T17:43:55Z",
					TMDBID:      0,
				},
				{
					MediaType:   "movie",
					MediaName:   "Test Movie Updated",
					FilePath:    "/mnt/data/media/movies/Test Movie/test.mkv",
					FileID:      101,
					ProcessedAt: "2025-09-02T17:44:18Z",
					TMDBID:      0,
				},
			},
			expected: []models.MissingFileEntry{
				{
					MediaType:   "movie",
					MediaName:   "Test Movie Updated", // Should keep the more recent one
					FilePath:    "/mnt/data/media/movies/Test Movie/test.mkv",
					FileID:      101,
					ProcessedAt: "2025-09-02T17:44:18Z",
					TMDBID:      0,
				},
			},
		},
		{
			name: "no duplicates - should return all entries",
			input: []models.MissingFileEntry{
				{
					MediaType:   "movie",
					MediaName:   "Movie 1",
					FilePath:    "/path1.mkv",
					FileID:      100,
					ProcessedAt: "2025-09-02T17:43:55Z",
					TMDBID:      1,
				},
				{
					MediaType:   "movie",
					MediaName:   "Movie 2",
					FilePath:    "/path2.mkv",
					FileID:      200,
					ProcessedAt: "2025-09-02T17:44:18Z",
					TMDBID:      2,
				},
			},
			expected: []models.MissingFileEntry{
				{
					MediaType:   "movie",
					MediaName:   "Movie 1",
					FilePath:    "/path1.mkv",
					FileID:      100,
					ProcessedAt: "2025-09-02T17:43:55Z",
					TMDBID:      1,
				},
				{
					MediaType:   "movie",
					MediaName:   "Movie 2",
					FilePath:    "/path2.mkv",
					FileID:      200,
					ProcessedAt: "2025-09-02T17:44:18Z",
					TMDBID:      2,
				},
			},
		},
		{
			name: "series entries - use file path for deduplication",
			input: []models.MissingFileEntry{
				{
					MediaType:   "series",
					MediaName:   "Test Series",
					FilePath:    "/series/s01e01.mkv",
					FileID:      300,
					ProcessedAt: "2025-09-02T17:43:55Z",
				},
				{
					MediaType:   "series",
					MediaName:   "Test Series Updated",
					FilePath:    "/series/s01e01.mkv",
					FileID:      301,
					ProcessedAt: "2025-09-02T17:44:18Z",
				},
			},
			expected: []models.MissingFileEntry{
				{
					MediaType:   "series",
					MediaName:   "Test Series Updated", // Should keep the more recent one
					FilePath:    "/series/s01e01.mkv",
					FileID:      301,
					ProcessedAt: "2025-09-02T17:44:18Z",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CleanupServiceImpl{}
			result := s.deduplicateMissingFiles(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d entries, got %d", len(tt.expected), len(result))
				return
			}

			// Create a map for easier comparison
			resultMap := make(map[string]models.MissingFileEntry)
			for _, entry := range result {
				key := entry.FilePath + "-" + entry.MediaName
				resultMap[key] = entry
			}

			for _, expected := range tt.expected {
				key := expected.FilePath + "-" + expected.MediaName
				if actual, exists := resultMap[key]; !exists {
					t.Errorf("Expected entry not found: %+v", expected)
				} else {
					if actual.FileID != expected.FileID {
						t.Errorf("Expected FileID %d, got %d", expected.FileID, actual.FileID)
					}
					if actual.TMDBID != expected.TMDBID {
						t.Errorf("Expected TMDBID %d, got %d", expected.TMDBID, actual.TMDBID)
					}
					if actual.MediaType != expected.MediaType {
						t.Errorf("Expected MediaType %s, got %s", expected.MediaType, actual.MediaType)
					}
				}
			}
		})
	}
}

func TestCleanupServiceImpl_deduplicateMissingFiles_EdgeCases(t *testing.T) {
	s := &CleanupServiceImpl{}

	// Test empty slice
	result := s.deduplicateMissingFiles([]models.MissingFileEntry{})
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d entries", len(result))
	}

	// Test single entry
	singleEntry := []models.MissingFileEntry{
		{
			MediaType:   "movie",
			MediaName:   "Single Movie",
			FilePath:    "/single.mkv",
			FileID:      123,
			ProcessedAt: "2025-09-02T17:43:55Z",
			TMDBID:      456,
		},
	}
	result = s.deduplicateMissingFiles(singleEntry)
	if len(result) != 1 {
		t.Errorf("Expected 1 entry for single input, got %d", len(result))
	}
	if result[0].FileID != 123 {
		t.Errorf("Expected FileID 123, got %d", result[0].FileID)
	}
}
