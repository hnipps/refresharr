package arr

import (
	"github.com/hnipps/refresharr/pkg/models"
	"golift.io/starr"
	"golift.io/starr/sonarr"
)

// mapSonarrSeriesToModels converts a starr Series to our models.Series
func mapSonarrSeriesToModels(s *sonarr.Series) models.Series {
	if s == nil {
		return models.Series{}
	}

	return models.Series{
		MediaItem: models.MediaItem{
			ID:    int(s.ID),
			Title: s.Title,
			Path:  s.Path,
		},
		SeasonCount:      len(s.Seasons),
		TVDBID:           int(s.TvdbID),
		Monitored:        s.Monitored,
		QualityProfileID: int(s.QualityProfileID),
		RootFolderPath:   s.RootFolderPath,
	}
}

// mapSonarrSeriesToModelsList converts a slice of starr Series to models.Series
func mapSonarrSeriesToModelsList(series []*sonarr.Series) []models.Series {
	result := make([]models.Series, len(series))
	for i, s := range series {
		result[i] = mapSonarrSeriesToModels(s)
	}
	return result
}

// mapSonarrEpisodeToModels converts a starr Episode to our models.Episode
func mapSonarrEpisodeToModels(e *sonarr.Episode) models.Episode {
	if e == nil {
		return models.Episode{}
	}

	var episodeFileID *int
	if e.EpisodeFileID != 0 {
		id := int(e.EpisodeFileID)
		episodeFileID = &id
	}

	return models.Episode{
		ID:            int(e.ID),
		SeriesID:      int(e.SeriesID),
		SeasonNumber:  e.SeasonNumber,
		EpisodeNumber: e.EpisodeNumber,
		Title:         e.Title,
		HasFile:       e.HasFile,
		EpisodeFileID: episodeFileID,
	}
}

// mapSonarrEpisodesToModelsList converts a slice of starr Episodes to models.Episode
func mapSonarrEpisodesToModelsList(episodes []*sonarr.Episode) []models.Episode {
	result := make([]models.Episode, len(episodes))
	for i, e := range episodes {
		result[i] = mapSonarrEpisodeToModels(e)
	}
	return result
}

// mapSonarrEpisodeFileToModels converts a starr EpisodeFile to our models.EpisodeFile
func mapSonarrEpisodeFileToModels(ef *sonarr.EpisodeFile) models.EpisodeFile {
	if ef == nil {
		return models.EpisodeFile{}
	}

	return models.EpisodeFile{
		ID:   int(ef.ID),
		Path: ef.Path,
	}
}

// mapSonarrRootFolderToModels converts a starr RootFolder to our models.RootFolder
func mapSonarrRootFolderToModels(rf *sonarr.RootFolder) models.RootFolder {
	if rf == nil {
		return models.RootFolder{}
	}

	return models.RootFolder{
		ID:   int(rf.ID),
		Path: rf.Path,
		Name: rf.Path, // starr doesn't have a separate name field
	}
}

// mapSonarrRootFoldersToModelsList converts a slice of starr RootFolders to models.RootFolder
func mapSonarrRootFoldersToModelsList(folders []*sonarr.RootFolder) []models.RootFolder {
	result := make([]models.RootFolder, len(folders))
	for i, rf := range folders {
		result[i] = mapSonarrRootFolderToModels(rf)
	}
	return result
}

// mapSonarrQualityProfileToModels converts a starr QualityProfile to our models.QualityProfile
func mapSonarrQualityProfileToModels(qp *sonarr.QualityProfile) models.QualityProfile {
	if qp == nil {
		return models.QualityProfile{}
	}

	return models.QualityProfile{
		ID:   int(qp.ID),
		Name: qp.Name,
	}
}

// mapSonarrQualityProfilesToModelsList converts a slice of starr QualityProfiles to models.QualityProfile
func mapSonarrQualityProfilesToModelsList(profiles []*sonarr.QualityProfile) []models.QualityProfile {
	result := make([]models.QualityProfile, len(profiles))
	for i, qp := range profiles {
		result[i] = mapSonarrQualityProfileToModels(qp)
	}
	return result
}

// mapSonarrQueueRecordToModels converts a starr QueueRecord to our models.QueueItem
func mapSonarrQueueRecordToModels(qr *sonarr.QueueRecord) models.QueueItem {
	if qr == nil {
		return models.QueueItem{}
	}

	var series *models.Series
	// Try to extract series information from QueueRecord
	// Check if SeriesID field exists and is not zero
	if qr.SeriesID != 0 {
		// Create a minimal Series object with the available information
		series = &models.Series{
			MediaItem: models.MediaItem{
				ID: int(qr.SeriesID),
				// We don't have series title from QueueRecord alone
				// The manual import process can work with just the ID
				Title: "", // Will be populated by other means if needed
			},
		}
	}

	// Map status messages
	var statusMessages []models.StatusMessage
	if qr.StatusMessages != nil {
		statusMessages = make([]models.StatusMessage, len(qr.StatusMessages))
		for i, sm := range qr.StatusMessages {
			statusMessages[i] = models.StatusMessage{
				Title:    sm.Title,
				Messages: sm.Messages,
			}
		}
	}

	return models.QueueItem{
		ID:             int(qr.ID),
		Title:          qr.Title,
		Series:         series,
		Status:         qr.Status,
		StatusMessages: statusMessages,
		ErrorMessage:   qr.ErrorMessage,
		Size:           int64(qr.Size),
		DownloadID:     qr.DownloadID,
		OutputPath:     qr.OutputPath,
		Protocol:       string(qr.Protocol),
		DownloadClient: qr.DownloadClient,
	}
}

// mapSonarrQueueToModelsList converts a starr Queue to models.QueueItem slice
func mapSonarrQueueToModelsList(queue *sonarr.Queue) []models.QueueItem {
	if queue == nil || queue.Records == nil {
		return nil
	}

	result := make([]models.QueueItem, len(queue.Records))
	for i, qr := range queue.Records {
		result[i] = mapSonarrQueueRecordToModels(qr)
	}
	return result
}

// mapModelsEpisodeToSonarr converts our models.Episode to starr compatible format for updates
func mapModelsEpisodeToSonarr(e models.Episode) *sonarr.Episode {
	var episodeFileID int64
	if e.EpisodeFileID != nil {
		episodeFileID = int64(*e.EpisodeFileID)
	}

	return &sonarr.Episode{
		ID:            int64(e.ID),
		SeriesID:      int64(e.SeriesID),
		SeasonNumber:  e.SeasonNumber,
		EpisodeNumber: e.EpisodeNumber,
		Title:         e.Title,
		HasFile:       e.HasFile,
		EpisodeFileID: episodeFileID,
	}
}

// mapSonarrManualImportToModels converts a starr ManualImportOutput to our models.ManualImportItem
func mapSonarrManualImportToModels(mi *sonarr.ManualImportOutput) models.ManualImportItem {
	if mi == nil {
		return models.ManualImportItem{}
	}

	var series *models.Series
	if mi.Series != nil {
		s := mapSonarrSeriesToModels(mi.Series)
		series = &s
	}

	var seasonNumber *int
	if mi.SeasonNumber != 0 {
		sn := mi.SeasonNumber
		seasonNumber = &sn
	}

	var episodes []models.Episode
	if mi.Episodes != nil {
		episodes = make([]models.Episode, len(mi.Episodes))
		for i, e := range mi.Episodes {
			episodes[i] = mapSonarrEpisodeToModels(e)
		}
	}

	var quality *models.Quality
	if mi.Quality != nil && mi.Quality.Quality != nil {
		quality = &models.Quality{
			ID:   int(mi.Quality.Quality.ID),
			Name: mi.Quality.Quality.Name,
		}
	}

	var rejections []string
	if mi.Rejections != nil {
		rejections = make([]string, len(mi.Rejections))
		for i, r := range mi.Rejections {
			rejections[i] = r.Reason
		}
	}

	return models.ManualImportItem{
		ID:            int(mi.ID),
		Path:          mi.Path,
		RelativePath:  mi.RelativePath,
		FolderName:    mi.FolderName,
		Name:          mi.Name,
		Size:          mi.Size,
		Series:        series,
		SeasonNumber:  seasonNumber,
		Episodes:      episodes,
		Quality:       quality,
		QualityWeight: int(mi.QualityWeight),
		DownloadID:    mi.DownloadID,
		Rejections:    rejections,
	}
}

// mapSonarrManualImportToModelsList converts a slice of starr ManualImportOutput to models.ManualImportItem
func mapSonarrManualImportToModelsList(items []*sonarr.ManualImportOutput) []models.ManualImportItem {
	if items == nil {
		return nil
	}

	result := make([]models.ManualImportItem, len(items))
	for i, item := range items {
		result[i] = mapSonarrManualImportToModels(item)
	}
	return result
}

// mapModelsManualImportToSonarr converts our models.ManualImportItem to starr format for execution
func mapModelsManualImportToSonarr(item models.ManualImportItem) *sonarr.ManualImportInput {
	var episodeIDs []int64
	if item.Episodes != nil {
		episodeIDs = make([]int64, len(item.Episodes))
		for i, e := range item.Episodes {
			episodeIDs[i] = int64(e.ID)
		}
	}

	var quality *starr.Quality
	if item.Quality != nil {
		quality = &starr.Quality{
			Quality: &starr.BaseQuality{
				Name: item.Quality.Name,
				ID:   int64(item.Quality.ID),
			},
		}
	}

	var seriesID int64
	if item.Series != nil {
		seriesID = int64(item.Series.ID)
	}

	var seasonNumber int
	if item.SeasonNumber != nil {
		seasonNumber = *item.SeasonNumber
	}

	return &sonarr.ManualImportInput{
		ID:           int64(item.ID),
		Path:         item.Path,
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
		EpisodeIDs:   episodeIDs,
		Quality:      quality,
		ReleaseGroup: "", // Not available in our models
		DownloadID:   item.DownloadID,
	}
}

// mapModelsManualImportToSonarrList converts a slice of our models.ManualImportItem to starr format
func mapModelsManualImportToSonarrList(items []models.ManualImportItem) []*sonarr.ManualImportInput {
	if items == nil {
		return nil
	}

	result := make([]*sonarr.ManualImportInput, len(items))
	for i, item := range items {
		result[i] = mapModelsManualImportToSonarr(item)
	}
	return result
}
