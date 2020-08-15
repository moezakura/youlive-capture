package utils

import "time"

func GetZeroTime() time.Time {
	return time.Time{}
}

func ToJST(t time.Time) time.Time {
	loc := time.FixedZone("Asia/Tokyo", 9*60*60)
	return t.In(loc)
}
