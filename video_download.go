package main

import (
	"bytes"
	"fmt"
	"golang.org/x/xerrors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type VideoDownload struct {
	CancelTick    chan struct{}
	targetVideoID chan string
	startTicker   *time.Ticker
}

func NewVideoDownload() *VideoDownload {
	return &VideoDownload{
		CancelTick:    make(chan struct{}, 1),
		targetVideoID: make(chan string, 1),
		startTicker:   nil,
	}
}

func (v *VideoDownload) SetData(videoID string, startTime time.Time) {
	duration := startTime.Sub(time.Now().In(time.UTC))
	if duration.Seconds() < 120 {
		v.startTicker = time.NewTicker(2 * time.Second)
	} else {
		duration = time.Duration(duration.Seconds()-30) * time.Second
		v.startTicker = time.NewTicker(duration)
	}
	v.targetVideoID <- videoID
}

func (v *VideoDownload) Run() {
	videoID := ""
	select {
	case <-v.CancelTick:
		return
	case videoID = <-v.targetVideoID:

	}
	<-v.startTicker.C
	v.startTicker.Stop()
	log.Printf("Download start live stream now!")

	t := time.NewTicker(time.Second)
	defer func() {
		t.Stop()
	}()
	downloadURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	for {
		<-t.C
		err := v.download(downloadURL)
		if err != nil {
			if xerrors.Is(err, LiveNotStarted) {
				log.Printf("live is not started: %+v", err)
				continue
			}
			log.Printf("download failed: %+v", err)
			continue
		}
	}
}

func (v *VideoDownload) download(url string) error {
	var (
		stdOutBuffer bytes.Buffer
		errOutBuffer bytes.Buffer
	)

	cmd := exec.Command("youtube-dl", "--hls-use-mpegts", url)
	stdOutBufferMultiWriter := io.MultiWriter(&stdOutBuffer, os.Stdout)
	errOutBufferMultiWriter := io.MultiWriter(&errOutBuffer, os.Stderr)
	cmd.Stdout = stdOutBufferMultiWriter
	cmd.Stderr = errOutBufferMultiWriter
	cancelTickCancel := make(chan struct{}, 1)
	defer func() {
		close(cancelTickCancel)
	}()

	go func() {
		select {
		case <-v.CancelTick:
		case <-cancelTickCancel:
		}
		process := cmd.Process
		err := process.Signal(syscall.SIGINT)
		if err != nil {
			log.Printf("fatal download quit")
		}
		log.Printf("download quit")
	}()

	err := cmd.Run()
	if err != nil {
		return xerrors.Errorf("VideoDownload.downloadWithAnyRetry exec error: %w", err)
	}

	out := fmt.Sprintf("%s\n%s", stdOutBuffer.String(), errOutBuffer.String())
	if strings.Contains(out, "This live event will begin in") {
		return LiveNotStarted
	}
	cancelTickCancel <- struct{}{}

	return nil
}
