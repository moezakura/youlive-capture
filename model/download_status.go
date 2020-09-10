package model

type DownloadStatus string

const (
	DownloadStatusNotYet      DownloadStatus = "not_yet_started"
	DownloadStatusDownloading DownloadStatus = "downloading"
	DownloadStatusCompleted   DownloadStatus = "completed"
)
