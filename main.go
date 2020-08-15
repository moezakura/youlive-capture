package main

import (
	"context"
	"flag"
	"github.com/moezakura/youlive-capture/utils"
	"log"
	"time"
)

var (
	apiKey        = flag.String("api", "", "youtube api key")
	targetChannel = flag.String("channel", "", "youtube channel ID")
)

func main() {
	flag.Parse()

	if *targetChannel == "" {
		log.Fatal("Channel must be specified.")
	}
	if *apiKey == "" {
		log.Fatal("Api must be specified.")
	}

	mainTicker := time.NewTicker(3 * time.Minute)
	active := false
	y := NewYoutubeAPI(*apiKey)
	v := NewVideoDownload()

	go func() {
		for {
			v.Run()
			active = false
		}
	}()

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
