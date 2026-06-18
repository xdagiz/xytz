package types

type QueueStatus string

const (
	QueueStatusPending     QueueStatus = "pending"
	QueueStatusDownloading QueueStatus = "downloading"
	QueueStatusComplete    QueueStatus = "complete"
	QueueStatusError       QueueStatus = "error"
	QueueStatusSkipped     QueueStatus = "skipped"
)

type QueueItem struct {
	Index       int
	Video       VideoItem
	URL         string
	Status      QueueStatus
	Progress    float64
	Speed       string
	ETA         string
	Error       string
	Destination string
}

type QueueState struct {
	Items      []QueueItem
	CurrentIdx int
	Total      int
	Completed  int
	Failed     int
	IsPaused   bool
}

type ToggleVideoSelectionMsg struct {
	Video VideoItem
}

type ClearQueueSelectionMsg struct{}

type SelectAllVideosMsg struct{}

type StartQueueConfirmMsg struct {
	Videos []VideoItem
}

type StartQueueDownloadMsg struct {
	FormatID        string
	IsAudioTab      bool
	ABR             float64
	DownloadOptions []DownloadOption
	Videos          []VideoItem
}

type StartQueueConfirmWithFormatMsg struct {
	Videos     []VideoItem
	FormatID   string
	IsAudioTab bool
	ABR        float64
}

type SkipCurrentQueueItemMsg struct{}

type RetryCurrentQueueItemMsg struct{}
