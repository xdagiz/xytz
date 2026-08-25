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
	OGType     string
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

type SpotifyAlbumTrack struct {
	ID       string
	Title    string
	Artist   string
	Duration float64
	Order    int
	Disc     int
	TrackNum int
	Playable bool
}

type SpotifyAlbum struct {
	ID          string
	Title       string
	Artist      string
	ReleaseDate string
	CoverURL    string
	SpotifyURL  string
	Tracks      []SpotifyAlbumTrack
}

func (a SpotifyAlbum) TotalDuration() float64 {
	var total float64
	for _, tr := range a.Tracks {
		total += tr.Duration
	}
	return total
}

type SpotifyAlbumResultMsg struct {
	Album *SpotifyAlbum
	Err   string
}

type SpotifyTrackResultMsg struct {
	Type  SpotifyEntityType
	Track *SpotifyTrack
	Err   string
}

type StartSpotifyTrackDownloadMsg struct {
	Track              SpotifyTrack
	OutputDir          string
	BaseName           string
	CookiesFromBrowser string
	Cookies            string
	OperationID        string
}
