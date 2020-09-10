package model

const (
	CancelReasonStatusUser       = 100
	CancelReasonStatusReSchedule = 105
	CancelReasonStatusDeleted    = 404
)

const (
	CancelReasonUser       = "User canceled"
	CancelReasonReSchedule = "Live start time re schedule"
	CancelReasonDeleted    = "Live deleted"
)

type CancelReason struct {
	StatusCode int
	Reason     string
}

func NewCancelReason(statusCode int, reason string) *CancelReason {
	return &CancelReason{StatusCode: statusCode, Reason: reason}
}
