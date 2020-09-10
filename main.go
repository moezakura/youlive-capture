package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/moezakura/youlive-capture/model"
	"github.com/moezakura/youlive-capture/utils"
	"log"
	"os"
	"strings"
	"time"
)

const (
	version = "0.1.0"
)

var (
	apiKey          = flag.String("api", "", "Youtube api key")
	targetChannel   = flag.String("channel", "", "Youtube channel ID")
	apiIntervalTime = flag.String("interval", "3m", "Youtube Data API call interval time")
	isInfinity      = flag.Bool("infinity", false, "Find the next delivery when the download is complete")
	versionFlag     = flag.Bool("v", false, "print version")
)

var (
	active     bool
	mainTicker *time.Ticker

	startTime time.Time
	videoID   string
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	if *targetChannel == "" {
		log.Fatal("Channel must be specified.")
	}
	if *apiKey == "" {
		log.Fatal("Api must be specified.")
	}

	intervalTime := getTimeFromText(*apiIntervalTime)
	mainTicker = time.NewTicker(intervalTime)

	log.Printf("Youtube Data API call interval time: %s", intervalTime.String())

	active = false
	y := NewYoutubeAPI(*apiKey)
	v := NewVideoDownload()

	go func() {
		for {
			v.Run()
			active = false
		}
	}()

	go func() {
		for {
			liveLoop(y, v)
		}
	}()

	inputLines := make(chan string, 255)
	go func() {
		for {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			t := scanner.Text()
			inputLines <- strings.Trim(t, " \n\r")
		}
	}()

	for {
		input := ""
		select {
		case input = <-inputLines:
		case <-v.CompleteTick:
			fmt.Println("Live capture completed!")
			if !*isInfinity {
				return
			}
		}

		if input == "quit" || input == "q" {
			v.CancelTick <- model.NewCancelReason(model.CancelReasonStatusUser, model.CancelReasonUser)
			fmt.Println("exit from user")
			return
		}
	}
}

func liveLoop(youtubeAPI *YoutubeAPI, videoDownload *VideoDownload) {
	isSkipWait := false
	defer func() {
		if !isSkipWait {
			<-mainTicker.C
		}
	}()
	if active {
		if videoDownload.Status != model.DownloadStatusNotYet {
			return
		}

		newStartTime := getTimeFromVideoID(youtubeAPI, videoID)
		if newStartTime.IsZero() {
			videoDownload.CancelTick <- model.NewCancelReason(model.CancelReasonStatusDeleted,
				model.CancelReasonDeleted)
			isSkipWait = true
			return
		}
		if startTime.Unix() == newStartTime.Unix() {
			return
		}

		videoDownload.CancelTick <- model.NewCancelReason(model.CancelReasonStatusReSchedule,
			model.CancelReasonReSchedule)

		startTime = newStartTime
		videoDownload.SetData(videoID, newStartTime)
		log.Printf("Got a live feed new start time")
		log.Printf("It's re scheduled to start at %s (id: %s)",
			utils.ToJST(startTime).Format("01/02 15:04:05"),
			videoID)
		return
	}

	log.Print("Get channel info")
	startTime, videoID := run(youtubeAPI)
	if !startTime.IsZero() {
		active = true
		videoDownload.SetData(videoID, startTime)
		log.Printf("Got a live feed start time")
		log.Printf("It's scheduled to start at %s (id: %s)",
			utils.ToJST(startTime).Format("01/02 15:04:05"),
			videoID)
	} else {
		log.Printf("Failed to get a live feed start time")
	}
}

func run(y *YoutubeAPI) (time.Time, string) {
	var err error
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	startTime, videoID, err = y.GetLiveTime(ctx, *targetChannel)
	if err != nil {
		log.Printf("youtube api GetLiveTime error: %+v", err)
		return startTime, ""
	}
	return startTime, videoID
}

func getTimeFromVideoID(y *YoutubeAPI, videoID string) time.Time {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	service, err := y.NewYoutubeService(ctx)
	if err != nil {
		log.Printf("youtube api NewYoutubeService error: %+v", err)
		return utils.GetZeroTime()
	}

	startTime, err := y.GetLiveStartTime(service, videoID)
	if err != nil {
		log.Printf("youtube api GetLiveTime error: %+v", err)
		return startTime
	}
	return startTime
}

func getTimeFromText(timeText string) time.Duration {
	t := 3 * time.Minute
	timeUnit, timeNumber, err := utils.GetTimeUnit(timeText)
	if err != nil {
		return t
	}

	if timeNumber <= 0 {
		return t
	}

	return timeUnit * time.Duration(timeNumber)
}
