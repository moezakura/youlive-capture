package main

import (
	"bytes"
	"fmt"
	"github.com/moezakura/youlive-capture/model"
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
	CancelTick   chan *model.CancelReason
	CompleteTick chan struct{}
	Status       model.DownloadStatus

	targetVideoID chan string
	startTicker   *time.Ticker
}

func NewVideoDownload() *VideoDownload {
	return &VideoDownload{
		CancelTick:    make(chan *model.CancelReason, 1),
		CompleteTick:  make(chan struct{}, 1),
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
	v.Status = model.DownloadStatusNotYet
	defer func() {
		v.Status = model.DownloadStatusCompleted
	}()
	videoID := ""
	select {
	case reason := <-v.CancelTick:
		log.Printf("run cancel (wait video id): %s", reason.Reason)
		return
	case videoID = <-v.targetVideoID:
		log.Printf("Confirm video id: %s", videoID)
	}

	select {
	case reason := <-v.CancelTick:
		v.startTicker.Stop()
		log.Printf("run cancel (wait timer): %s", reason.Reason)
		return
	case <-v.startTicker.C:
		v.startTicker.Stop()
	}
	log.Printf("Download start live stream now!")

	t := time.NewTicker(time.Second)
	defer func() {
		t.Stop()
	}()
	downloadURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	v.Status = model.DownloadStatusDownloading
	for {
		<-t.C
		err := v.download(downloadURL)
		if err != nil {
			switch {
			case xerrors.Is(err, LiveNotStarted):
				log.Printf("live is not started: %+v", err)
				continue
			case xerrors.Is(err, AlreadyDownloaded):
				v.CompleteTick <- struct{}{}
				return
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
		case reason := <-v.CancelTick:
			log.Printf("download cancel: %s", reason.Reason)
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
	if strings.Contains(out, "already been downloaded and merged") {
		return AlreadyDownloaded
	}
	cancelTickCancel <- struct{}{}

	return nil
}
