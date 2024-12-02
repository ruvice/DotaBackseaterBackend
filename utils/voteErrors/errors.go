package voteErrors

import "fmt"

type ErrorCode int

// Define a custom error type
type VoteError struct {
	Message string
	Code    ErrorCode
}

const (
	CodeVotedItemNotFound ErrorCode = iota + 1
	CodeUpdateVoteError
	CodeVoteRelationCreationError
	CodeVoteRelationExpirationError
	CodeItemRefreshError
	CodeItemGetRedisError
	CodeMissingCacheVoteThreshold
	CodeTwitchMessageTooManyRequests
	CodeUnknown
)

// Implement the `Error()` method for the `error` interface
func (e *VoteError) Error() string {
	return fmt.Sprintf("Error: %s (Code: %d)", e.Message, e.Code)
}

func NewError(code ErrorCode, message string) *VoteError {
	return &VoteError{
		Code:    code,
		Message: message,
	}
}
