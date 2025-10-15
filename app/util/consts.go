package util

// error log messages
const (
	ErrLogInvalidAddReq      = "invalid add request"
	ErrLogLongServiceName    = "service name is too long, 255 is allowed"
	ErrLogAddSub             = "failed to add a subscription"
	ErrLogInvalidSubID       = "invalid subscription id"
	ErrLogGetSub             = "failed to get a subscription"
	ErrLogInvalidUpdateReq   = "invalid update request"
	ErrLogSubNotFound        = "subscription not found"
	ErrLogParseStartDate     = "failed to parse start date"
	ErrLogParseEndDate       = "failed to parse end date"
	ErrLodEndBeforeStartDate = "end date could not be before start date"
	ErrLogUpdateSub          = "failed to update a subscription"
	ErrLogDeleteSub          = "failed to delete a subscription"
	ErrLogInvalidListReq     = "invalid list request"
	ErrLogInvalidUserID      = "invalid user id"
	ErrLogWriteResp          = "failed to write a response"
	ErrLogLoggerSync         = "failed to sync logger"
)

// success log messages
const (
	SuccessLogAddSub    = "subscription added successfully"
	SussessLogGetSub    = "subscription retrieved successfully"
	SuccessLogUpdateSub = "subscription updated successfully"
	SuccessLogDeleteSub = "subscription deleted successfully"
	SuccessLogListSubs  = "subscriptions listed successfully"
	SuccessLogSumSubs   = "total sum listed successfully"
)

// date format
const (
	DateFormat = "01-2006"
)
