package arr

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hnipps/refresharr/pkg/models"
)

// Mock implementations for testing

type mockClient struct {
	name                   string
	testConnectionError    error
	allSeries              []models.Series
	allSeriesError         error
	episodes               map[int][]models.Episode // seriesID -> episodes
	episodesError          error
	episodeFiles           map[int]*models.EpisodeFile // fileID -> episodeFile
	episodeFileError       error
	deleteEpisodeFileError error
	updateEpisodeError     error
	triggerRefreshError    error
	deletedFileIDs         []int
	updatedEpisodes        []models.Episode
}

func (m *mockClient) GetName() string {
	return m.name
}

func (m *mockClient) TestConnection(ctx context.Context) error {
	return m.testConnectionError
}

func (m *mockClient) GetAllSeries(ctx context.Context) ([]models.Series, error) {
	return m.allSeries, m.allSeriesError
}

func (m *mockClient) GetAllMovies(ctx context.Context) ([]models.Movie, error) {
	return nil, nil // Not implemented for this test
}

func (m *mockClient) GetMovie(ctx context.Context, movieID int) (*models.Movie, error) {
	return nil, nil // Not implemented for this test
}

func (m *mockClient) GetEpisodesForSeries(ctx context.Context, seriesID int) ([]models.Episode, error) {
	if m.episodesError != nil {
		return nil, m.episodesError
	}
	episodes, exists := m.episodes[seriesID]
	if !exists {
		return []models.Episode{}, nil
	}
	return episodes, nil
}

func (m *mockClient) GetEpisodeFile(ctx context.Context, fileID int) (*models.EpisodeFile, error) {
	if m.episodeFileError != nil {
		return nil, m.episodeFileError
	}
	file, exists := m.episodeFiles[fileID]
	if !exists {
		return nil, errors.New("episode file not found")
	}
	return file, nil
}

func (m *mockClient) DeleteEpisodeFile(ctx context.Context, fileID int) error {
	if m.deleteEpisodeFileError != nil {
		return m.deleteEpisodeFileError
	}
	m.deletedFileIDs = append(m.deletedFileIDs, fileID)
	return nil
}

func (m *mockClient) UpdateEpisode(ctx context.Context, episode models.Episode) error {
	if m.updateEpisodeError != nil {
		return m.updateEpisodeError
	}
	m.updatedEpisodes = append(m.updatedEpisodes, episode)
	return nil
}

func (m *mockClient) GetMovieFile(ctx context.Context, fileID int) (*models.MovieFile, error) {
	return nil, errors.New("GetMovieFile not implemented in mock")
}

func (m *mockClient) DeleteMovieFile(ctx context.Context, fileID int) error {
	return errors.New("DeleteMovieFile not implemented in mock")
}

func (m *mockClient) UpdateMovie(ctx context.Context, movie models.Movie) error {
	return errors.New("UpdateMovie not implemented in mock")
}

func (m *mockClient) TriggerRefresh(ctx context.Context) error {
	return m.triggerRefreshError
}

// New methods for broken symlink functionality (stubs for testing)
func (m *mockClient) GetRootFolders(ctx context.Context) ([]models.RootFolder, error) {
	return nil, errors.New("GetRootFolders not implemented in mock")
}

func (m *mockClient) GetQualityProfiles(ctx context.Context) ([]models.QualityProfile, error) {
	return nil, errors.New("GetQualityProfiles not implemented in mock")
}

func (m *mockClient) LookupMovieByTMDBID(ctx context.Context, tmdbID int) (*models.MovieLookup, error) {
	return nil, errors.New("LookupMovieByTMDBID not implemented in mock")
}

func (m *mockClient) GetMovieByTMDBID(ctx context.Context, tmdbID int) (*models.Movie, error) {
	return nil, errors.New("GetMovieByTMDBID not implemented in mock")
}

func (m *mockClient) AddMovie(ctx context.Context, movie models.Movie) (*models.Movie, error) {
	return nil, errors.New("AddMovie not implemented in mock")
}

func (m *mockClient) AddSeries(ctx context.Context, series models.Series) (*models.Series, error) {
	return nil, errors.New("AddSeries not implemented in mock")
}

func (m *mockClient) GetSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.Series, error) {
	return nil, errors.New("GetSeriesByTVDBID not implemented in mock")
}

func (m *mockClient) LookupSeriesByTVDBID(ctx context.Context, tvdbID int) (*models.SeriesLookup, error) {
	return nil, errors.New("LookupSeriesByTVDBID not implemented in mock")
}

func (m *mockClient) GetQueue(ctx context.Context) ([]models.QueueItem, error) {
	return []models.QueueItem{}, nil
}

func (m *mockClient) GetQueueDetails(ctx context.Context, queueID int) (*models.QueueItem, error) {
	return &models.QueueItem{}, nil
}

func (m *mockClient) RemoveFromQueue(ctx context.Context, queueID int, removeFromClient bool) error {
	return nil
}

func (m *mockClient) TriggerDownloadClientScan(ctx context.Context) error {
	return nil
}

func (m *mockClient) GetManualImport(ctx context.Context, folder string) ([]models.ManualImportItem, error) {
	return []models.ManualImportItem{}, nil
}

func (m *mockClient) ExecuteManualImport(ctx context.Context, files []models.ManualImportItem, importMode string) error {
	return nil
}

type mockFileChecker struct {
	fileExists map[string]bool
	readable   map[string]bool
}

func (m *mockFileChecker) FileExists(path string) bool {
	exists, found := m.fileExists[path]
	if !found {
		return false
	}
	return exists
}

func (m *mockFileChecker) IsReadable(path string) bool {
	readable, found := m.readable[path]
	if !found {
		return false
	}
	return readable
}

func (m *mockFileChecker) IsSymlink(path string) bool {
	// For testing, assume any path with "symlink" in it is a symlink
	return strings.Contains(path, "symlink")
}

func (m *mockFileChecker) FindBrokenSymlinks(rootDir string, extensions []string) ([]string, error) {
	// For testing, return empty list (can be expanded later for specific tests)
	return []string{}, nil
}

func (m *mockFileChecker) DeleteSymlink(path string) error {
	// For testing, just return nil (can be expanded later for specific tests)
	return nil
}

type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	m.errorMessages = append(m.errorMessages, msg)
}

type mockProgressReporter struct {
	seriesStarted        []string
	episodesStarted      []string
	missingFilesReported []string
	deletedRecords       []int
	errors               []error
	finishCalled         bool
	finalStats           models.CleanupStats
}

func (m *mockProgressReporter) StartSeries(seriesID int, seriesName string, current, total int) {
	m.seriesStarted = append(m.seriesStarted, seriesName)
}

func (m *mockProgressReporter) StartEpisode(episodeID int, seasonNum, episodeNum int) {
	m.episodesStarted = append(m.episodesStarted, "episode")
}

func (m *mockProgressReporter) StartMovie(movieID int, movieName string, current, total int) {
	// For testing purposes, we can track movie starts similar to series
	m.seriesStarted = append(m.seriesStarted, movieName)
}

func (m *mockProgressReporter) ReportMissingFile(filePath string) {
	m.missingFilesReported = append(m.missingFilesReported, filePath)
}

func (m *mockProgressReporter) ReportDeletedRecord(fileID int) {
	m.deletedRecords = append(m.deletedRecords, fileID)
}

func (m *mockProgressReporter) ReportDeletedEpisodeRecord(fileID int) {
	m.deletedRecords = append(m.deletedRecords, fileID)
}

func (m *mockProgressReporter) ReportDeletedMovieRecord(fileID int) {
	m.deletedRecords = append(m.deletedRecords, fileID)
}

func (m *mockProgressReporter) ReportError(err error) {
	m.errors = append(m.errors, err)
}

func (m *mockProgressReporter) Finish(stats models.CleanupStats) {
	m.finishCalled = true
	m.finalStats = stats
}

func TestNewCleanupService(t *testing.T) {
	client := &mockClient{}
	fileChecker := &mockFileChecker{}
	logger := &mockLogger{}
	progressReporter := &mockProgressReporter{}

	service := NewCleanupService(client, fileChecker, logger, progressReporter, time.Millisecond, false)
	if service == nil {
		t.Error("NewCleanupService() returned nil")
	}
}

func TestCleanupService_CleanupMissingFiles_Success(t *testing.T) {
	// Setup mocks
	client := &mockClient{
		name: "sonarr",
		allSeries: []models.Series{
			{MediaItem: models.MediaItem{ID: 1, Title: "Test Series 1"}},
			{MediaItem: models.MediaItem{ID: 2, Title: "Test Series 2"}},
		},
		episodes: map[int][]models.Episode{
			1: {
				{
					ID:            1,
					SeriesID:      1,
					SeasonNumber:  1,
					EpisodeNumber: 1,
					HasFile:       true,
					EpisodeFileID: intPtr(100),
				},
				{
					ID:            2,
					SeriesID:      1,
					SeasonNumber:  1,
					EpisodeNumber: 2,
					HasFile:       false, // This episode has no file, should be skipped
				},
			},
			2: {
				{
					ID:            3,
					SeriesID:      2,
					SeasonNumber:  1,
					EpisodeNumber: 1,
					HasFile:       true,
					EpisodeFileID: intPtr(200),
				},
			},
		},
		episodeFiles: map[int]*models.EpisodeFile{
			100: {ID: 100, Path: "/path/to/missing/episode1.mkv"},
			200: {ID: 200, Path: "/path/to/existing/episode2.mkv"},
		},
	}

	fileChecker := &mockFileChecker{
		fileExists: map[string]bool{
			"/path/to/missing/episode1.mkv":  false, // Missing file
			"/path/to/existing/episode2.mkv": true,  // Existing file
		},
	}

	logger := &mockLogger{}
	progressReporter := &mockProgressReporter{}

	service := NewCleanupService(client, fileChecker, logger, progressReporter, 0, false)

	ctx := context.Background()
	result, err := service.CleanupMissingFiles(ctx)

	// Assertions
	if err != nil {
		t.Errorf("CleanupMissingFiles() failed: %v", err)
	}
	if result == nil {
		t.Fatal("CleanupMissingFiles() returned nil result")
	}
	if !result.Success {
		t.Error("Expected cleanup to succeed")
	}
	if result.Stats.TotalItemsChecked != 2 {
		t.Errorf("Expected 2 items checked, got %d", result.Stats.TotalItemsChecked)
	}
	if result.Stats.MissingFiles != 1 {
		t.Errorf("Expected 1 missing file, got %d", result.Stats.MissingFiles)
	}
	if result.Stats.DeletedRecords != 1 {
		t.Errorf("Expected 1 deleted record, got %d", result.Stats.DeletedRecords)
	}
	if result.Stats.Errors != 0 {
		t.Errorf("Expected 0 errors, got %d", result.Stats.Errors)
	}

	// Check that the correct file was deleted
	if len(client.deletedFileIDs) != 1 || client.deletedFileIDs[0] != 100 {
		t.Errorf("Expected file ID 100 to be deleted, got %v", client.deletedFileIDs)
	}

	// Check progress reporting
	if !progressReporter.finishCalled {
		t.Error("Expected Finish() to be called on progress reporter")
	}
}

func TestCleanupService_CleanupMissingFiles_DryRun(t *testing.T) {
	// Setup mocks
	client := &mockClient{
		name: "sonarr",
		allSeries: []models.Series{
			{MediaItem: models.MediaItem{ID: 1, Title: "Test Series"}},
		},
		episodes: map[int][]models.Episode{
			1: {
				{
					ID:            1,
					SeriesID:      1,
					SeasonNumber:  1,
					EpisodeNumber: 1,
					HasFile:       true,
					EpisodeFileID: intPtr(100),
				},
			},
		},
		episodeFiles: map[int]*models.EpisodeFile{
			100: {ID: 100, Path: "/path/to/missing/episode.mkv"},
		},
	}

	fileChecker := &mockFileChecker{
		fileExists: map[string]bool{
			"/path/to/missing/episode.mkv": false, // Missing file
		},
	}

	logger := &mockLogger{}
	progressReporter := &mockProgressReporter{}

	// Enable dry run mode
	service := NewCleanupService(client, fileChecker, logger, progressReporter, 0, true)

	ctx := context.Background()
	result, err := service.CleanupMissingFiles(ctx)

	// Assertions
	if err != nil {
		t.Errorf("CleanupMissingFiles() failed: %v", err)
	}
	if result.Stats.MissingFiles != 1 {
		t.Errorf("Expected 1 missing file, got %d", result.Stats.MissingFiles)
	}
	// In dry run mode, no records should be deleted
	if result.Stats.DeletedRecords != 0 {
		t.Errorf("Expected 0 deleted records in dry run, got %d", result.Stats.DeletedRecords)
	}
	// No files should actually be deleted
	if len(client.deletedFileIDs) != 0 {
		t.Errorf("Expected no files to be deleted in dry run, got %v", client.deletedFileIDs)
	}
}

func TestCleanupService_ConnectionError(t *testing.T) {
	// Setup mocks with connection error
	client := &mockClient{
		name:                "sonarr",
		testConnectionError: errors.New("connection failed"),
	}

	fileChecker := &mockFileChecker{}
	logger := &mockLogger{}
	progressReporter := &mockProgressReporter{}

	service := NewCleanupService(client, fileChecker, logger, progressReporter, 0, false)

	ctx := context.Background()
	result, err := service.CleanupMissingFiles(ctx)

	// Should fail with connection error
	if err == nil {
		t.Error("Expected error due to connection failure")
	}
	if result != nil {
		t.Error("Expected nil result on connection failure")
	}
}

func TestCleanupService_NoSeries(t *testing.T) {
	// Setup mocks with no series
	client := &mockClient{
		name:      "sonarr",
		allSeries: []models.Series{}, // No series
	}

	fileChecker := &mockFileChecker{}
	logger := &mockLogger{}
	progressReporter := &mockProgressReporter{}

	service := NewCleanupService(client, fileChecker, logger, progressReporter, 0, false)

	ctx := context.Background()
	result, err := service.CleanupMissingFiles(ctx)

	// Should succeed but with zero stats
	if err != nil {
		t.Errorf("CleanupMissingFiles() failed: %v", err)
	}
	if !result.Success {
		t.Error("Expected success with no series")
	}
	if result.Stats.TotalItemsChecked != 0 {
		t.Errorf("Expected 0 items checked, got %d", result.Stats.TotalItemsChecked)
	}
}

func TestCleanupService_APIError(t *testing.T) {
	// Setup mocks with API error when getting series
	client := &mockClient{
		name:           "sonarr",
		allSeriesError: errors.New("API error"),
	}

	fileChecker := &mockFileChecker{}
	logger := &mockLogger{}
	progressReporter := &mockProgressReporter{}

	service := NewCleanupService(client, fileChecker, logger, progressReporter, 0, false)

	ctx := context.Background()
	result, err := service.CleanupMissingFiles(ctx)

	// Should fail with API error
	if err == nil {
		t.Error("Expected error due to API failure")
	}
	if result != nil {
		t.Error("Expected nil result on API failure")
	}
}

func TestCleanupService_CancelledContext(t *testing.T) {
	// Setup mocks
	client := &mockClient{
		name: "sonarr",
		allSeries: []models.Series{
			{MediaItem: models.MediaItem{ID: 1, Title: "Test Series"}},
		},
	}

	fileChecker := &mockFileChecker{}
	logger := &mockLogger{}
	progressReporter := &mockProgressReporter{}

	service := NewCleanupService(client, fileChecker, logger, progressReporter, 0, false)

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := service.CleanupMissingFilesForSeries(ctx, []int{1})

	// Should handle cancellation gracefully
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
	if result == nil {
		t.Error("Expected result even on cancellation")
	}
	if result.Success {
		t.Error("Expected success=false on cancellation")
	}
}

// intPtr is a helper function to get a pointer to an int
func intPtr(i int) *int {
	return &i
}
