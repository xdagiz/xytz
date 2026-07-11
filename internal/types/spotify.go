package types

type SpotifyEntityType string

const (
	SpotifyEntityTrack    SpotifyEntityType = "track"
	SpotifyEntityAlbum    SpotifyEntityType = "album"
	SpotifyEntityPlaylist SpotifyEntityType = "playlist"
)

type SpotifyTrackItem struct {
	ID         string
	Title      string
	Artist     string
	Album      string
	Duration   float64
	TrackNum   int
	DiscNum    int
	CoverURL   string
	SpotifyURL string
}

type SpotifyTrack struct {
	SpotifyTrackItem
	ReleaseDate string
}

type SpotifyTrackResultMsg struct {
	Type  SpotifyEntityType
	Track *SpotifyTrack
	Err   string
}

type StartSpotifyTrackMsg struct {
	URL string
}
