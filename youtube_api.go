package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/moezakura/youlive-capture/utils"
	"golang.org/x/xerrors"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type YoutubeAPI struct {
	apiKey string
}

func NewYoutubeAPI(apiKey string) *YoutubeAPI {
	y := new(YoutubeAPI)
	y.apiKey = apiKey
	return y
}

func (y *YoutubeAPI) NewYoutubeService(ctx context.Context) (*youtube.Service, error) {
	service, err := youtube.NewService(ctx, option.WithAPIKey(y.apiKey))
	if err != nil {
		return nil, xerrors.Errorf("YoutubeAPI.NewYoutubeService youtube.NewService error: %w", err)
	}

	return service, nil
}

func (y *YoutubeAPI) GetLiveTime(ctx context.Context, channelID string) (time.Time, string, error) {
	service, err := y.NewYoutubeService(ctx)
	if err != nil {
		return utils.GetZeroTime(), "", xerrors.Errorf("YoutubeAPI.GetLiveTime y.NewYoutubeService error: %w", err)
	}

	searchService := youtube.NewSearchService(service)
	req := searchService.List([]string{"snippet"}).
		ChannelId(channelID).
		MaxResults(3).
		Order("date")

	res, err := req.Do()
	if err != nil {
		return utils.GetZeroTime(), "", xerrors.Errorf("YoutubeAPI.GetLiveTime req.Do error: %w", err)
	}

	if len(res.Items) == 0 {
		return utils.GetZeroTime(), "", nil
	}

	for _, item := range res.Items {
		videoID := item.Id.VideoId
		// upcoming 配信前, none 配信後/動画, live 配信中
		if item.Snippet.LiveBroadcastContent == "none" {
			continue
		}

		if item.Snippet.LiveBroadcastContent == "live" {
			return time.Now(), videoID, nil
		}

		if item.Snippet.LiveBroadcastContent == "upcoming" {
			startTime, err := y.GetLiveStartTime(service, videoID)
			if err != nil {
				return utils.GetZeroTime(), "",
					xerrors.Errorf("YoutubeAPI.GetLiveTime y.GetLiveStartTime error: %w", err)
			}

			// 30分以上未来であればスキップする
			if y.isOverTime(startTime, videoID) {
				return utils.GetZeroTime(), "", nil
			}

			return startTime, videoID, nil
		}
	}

	return utils.GetZeroTime(), "", nil
}

func (y *YoutubeAPI) GetLiveStartTime(service *youtube.Service, videoID string) (time.Time, error) {
	liveInfo, err := y.getLiveInfo(service, videoID)
	if err != nil {
		return utils.GetZeroTime(), xerrors.Errorf("YoutubeAPI.GetLiveStartTime y.getLiveInfo error: %w", err)
	}
	if len(liveInfo.Items) == 0 {
		return utils.GetZeroTime(), nil
	}
	liveInfoItem := liveInfo.Items[0]
	startTimeStr := liveInfoItem.LiveStreamingDetails.ScheduledStartTime
	startTime, err := time.Parse("2006-01-02T15:04:05Z(MST)", fmt.Sprintf("%s(UTC)", startTimeStr))
	if err != nil {
		return utils.GetZeroTime(), xerrors.Errorf("YoutubeAPI.GetLiveStartTime time.Parse(%s) error: %w",
			startTimeStr, err)
	}
	return startTime.In(time.UTC), nil
}

func (y *YoutubeAPI) getLiveInfo(service *youtube.Service, videoID string) (*youtube.VideoListResponse, error) {
	listService := youtube.NewVideosService(service)

	req := listService.List([]string{"liveStreamingDetails"}).
		Id(videoID).
		MaxResults(1)

	res, err := req.Do()
	if err != nil {
		return nil, xerrors.Errorf("YoutubeAPI.getLiveInfo req.Do error: %w", err)
	}
	return res, nil
}

// isOverTime - 現在時刻と30分以上かけ離れているか
func (y *YoutubeAPI) isOverTime(t time.Time, videoID string) bool {
	startTimeUnix := t.Unix()
	nowUnix := time.Now().Unix()
	timeDiff := nowUnix - startTimeUnix
	if timeDiff < -(30 * 60) {
		d := time.Duration(timeDiff) * time.Second
		log.Printf("skip live (over 30m) (diff: %s) ID: %s", d.String(), videoID)
		return true
	}
	return false
}
