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

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	Stats    CleanupStats
	Messages []string
	Success  bool
}
