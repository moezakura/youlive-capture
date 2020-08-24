package main

import (
	"errors"
)

var (
	LiveNotStarted    = errors.New("Live not started")
	AlreadyDownloaded = errors.New("Live already downloaded")
)
