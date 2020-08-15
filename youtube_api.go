package main

import (
	"context"
	"fmt"
	"github.com/moezakura/youlive-capture/utils"
	"golang.org/x/xerrors"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"time"
)

type YoutubeAPI struct {
	apiKey string
}

func NewYoutubeAPI(apiKey string) *YoutubeAPI {
	y := new(YoutubeAPI)
	y.apiKey = apiKey
	return y
}

func (y *YoutubeAPI) GetLiveTime(ctx context.Context, channelID string) (time.Time, string, error) {
	service, err := youtube.NewService(ctx, option.WithAPIKey(y.apiKey))
	if err != nil {
		return utils.GetZeroTime(), "", xerrors.Errorf("YoutubeAPI.GetLiveTime youtube.NewService error: %w", err)
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
			liveInfo, err := y.getLiveInfo(service, videoID)
			if err != nil {
				return utils.GetZeroTime(), "", xerrors.Errorf("YoutubeAPI.GetLiveTime y.getLiveInfo error: %w", err)
			}
			if len(liveInfo.Items) == 0 {
				return utils.GetZeroTime(), "", nil
			}
			liveInfoItem := liveInfo.Items[0]
			startTimeStr := liveInfoItem.LiveStreamingDetails.ScheduledStartTime
			startTime, err := time.Parse("2006-01-02T15:04:05Z(MST)", fmt.Sprintf("%s(UTC)", startTimeStr))
			if err != nil {
				return utils.GetZeroTime(), "", xerrors.Errorf("YoutubeAPI.GetLiveTime time.Parse(%s) error: %w", startTimeStr, err)
			}
			startTime = startTime.In(time.UTC)
			return startTime, videoID, nil
		}
	}

	return utils.GetZeroTime(), "", nil
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
