package dbsError

type VoteErrorCode int

const (
	CodeVotedItemNotFound VoteErrorCode = iota + 1
	CodeUpdateVoteError
	CodeVoteRelationCreationError
	CodeVoteRelationExpirationError
	CodeItemRefreshError
	CodeItemGetRedisError
	CodeMissingCacheVoteThreshold
	CodeTwitchMessageTooManyRequests
	CodeUnknown
)

// Define a custom error type
type VoteError struct {
	DBSError DBSError
	Code     VoteErrorCode
}

// Error implements error.
func (e *VoteError) Error() string {
	return e.DBSError.Error()
}

// NewConfigError creates a new DBSError.
func NewVoteError(funcName string, code VoteErrorCode, message string, err error) *VoteError {
	return &VoteError{
		DBSError: DBSError{
			FuncName: funcName,
			Message:  message,
			Code:     int(code),
			Err:      err,
		},
		Code: code,
	}
}
