//go:build windows

package downloader

func PauseSupported() bool {
	return false
}

func PauseProcess(dm *DownloadManager) bool {
	return false
}

func ResumeProcess(dm *DownloadManager) bool {
	return false
}
