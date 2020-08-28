package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/moezakura/youlive-capture/utils"
	"log"
	"os"
	"strings"
	"time"
)

var (
	apiKey          = flag.String("api", "", "Youtube api key")
	targetChannel   = flag.String("channel", "", "Youtube channel ID")
	apiIntervalTime = flag.String("interval", "3m", "Youtube Data API call interval time")
	isInfinity      = flag.Bool("infinity", false, "Find the next delivery when the download is complete")
)

func main() {
	flag.Parse()

	if *targetChannel == "" {
		log.Fatal("Channel must be specified.")
	}
	if *apiKey == "" {
		log.Fatal("Api must be specified.")
	}

	intervalTime := getTimeFromText(*apiIntervalTime)
	mainTicker := time.NewTicker(intervalTime)

	log.Printf("Youtube Data API call interval time: %s", intervalTime.String())

	active := false
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
			func() {
				defer func() {
					<-mainTicker.C
				}()
				if active {
					return
				}

				log.Print("Get channel info")
				startTime, videoID := run(y)
				if !startTime.IsZero() {
					active = true
					v.SetData(videoID, startTime)
					log.Printf("Got a live feed start time")
					log.Printf("It's scheduled to start at %s", utils.ToJST(startTime).Format("15:04:05"))
				} else {
					log.Printf("Failed to get a live feed start time")
				}
			}()
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
			v.CancelTick <- struct{}{}
			fmt.Println("exit from user")
			return
		}
	}
}

func run(y *YoutubeAPI) (time.Time, string) {
	ctx, _ := context.WithTimeout(context.TODO(), 30*time.Second)
	startTime, videoID, err := y.GetLiveTime(ctx, *targetChannel)
	if err != nil {
		log.Printf("youtube api GetLiveTime error: %+v", err)
		return startTime, ""
	}
	return startTime, videoID
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
