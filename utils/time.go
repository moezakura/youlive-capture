package utils

import (
	"golang.org/x/xerrors"
	"strconv"
	"strings"
	"time"
)

func GetZeroTime() time.Time {
	return time.Time{}
}

func ToJST(t time.Time) time.Time {
	loc := time.FixedZone("Asia/Tokyo", 9*60*60)
	return t.In(loc)
}

func GetTimeUnit(timeText string) (time.Duration, int64, error) {
	timeTextRune := []rune(timeText)
	timeNumberText := string(timeTextRune[:len(timeTextRune)-1])
	if strings.HasSuffix(timeText, "ms") {
		timeNumberText = string(timeTextRune[:len(timeTextRune)-2])
	}
	timeNumber, err := strconv.ParseInt(timeNumberText, 10, 64)
	if err != nil {
		return time.Second, 0, xerrors.Errorf("failed to parse time: %w", err)
	}

	switch {
	case strings.HasSuffix(timeText, "ms"):
		return time.Millisecond, timeNumber, nil
	case strings.HasSuffix(timeText, "s"):
		return time.Second, timeNumber, nil
	case strings.HasSuffix(timeText, "m"):
		return time.Minute, timeNumber, nil
	case strings.HasSuffix(timeText, "h"):
		return time.Hour, timeNumber, nil
	}

	timeNumber, err = strconv.ParseInt(timeText, 10, 64)
	if err != nil {
		return time.Second, 0, xerrors.Errorf("failed to parse time: %w", err)
	}
	return time.Second, timeNumber, nil
}
