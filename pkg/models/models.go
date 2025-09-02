package models

// MediaItem represents a base media item (can be extended for TV shows or movies)
type MediaItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path,omitempty"`
}

// Series represents a TV series in Sonarr
type Series struct {
	MediaItem
	SeasonCount int `json:"seasonCount,omitempty"`
}

// Movie represents a movie in Radarr
type Movie struct {
	MediaItem
	Year        int  `json:"year,omitempty"`
	HasFile     bool `json:"hasFile"`
	MovieFileID *int `json:"movieFileId,omitempty"`
}

// Episode represents a TV episode
type Episode struct {
	ID            int    `json:"id"`
	SeriesID      int    `json:"seriesId"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	HasFile       bool   `json:"hasFile"`
	EpisodeFileID *int   `json:"episodeFileId,omitempty"`
}

// EpisodeFile represents a file associated with an episode
type EpisodeFile struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

// MovieFile represents a file associated with a movie (for future Radarr support)
type MovieFile struct {
	ID      int    `json:"id"`
	Path    string `json:"path"`
	MovieID int    `json:"movieId"`
}

// CleanupStats tracks cleanup operation statistics
type CleanupStats struct {
	TotalItemsChecked int
	MissingFiles      int
	DeletedRecords    int
	Errors            int
}

// MissingFileEntry represents a single missing file entry in the report
type MissingFileEntry struct {
	MediaType   string `json:"mediaType"`             // "movie" or "series"
	MediaName   string `json:"mediaName"`             // Movie title or series title
	EpisodeName string `json:"episodeName,omitempty"` // Episode name (only for series)
	Season      *int   `json:"season,omitempty"`      // Season number (only for series)
	Episode     *int   `json:"episode,omitempty"`     // Episode number (only for series)
	FilePath    string `json:"filePath"`              // Path to the missing file
	FileID      int    `json:"fileId"`                // File ID in the database
	ProcessedAt string `json:"processedAt"`           // Timestamp when processed
}

// MissingFilesReport represents a complete missing files report
type MissingFilesReport struct {
	GeneratedAt  string             `json:"generatedAt"`
	RunType      string             `json:"runType"`     // "dry-run" or "real-run"
	ServiceType  string             `json:"serviceType"` // "sonarr" or "radarr"
	TotalMissing int                `json:"totalMissing"`
	MissingFiles []MissingFileEntry `json:"missingFiles"`
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	Stats    CleanupStats
	Messages []string
	Success  bool
	Report   *MissingFilesReport `json:"report,omitempty"` // Optional report data
}
