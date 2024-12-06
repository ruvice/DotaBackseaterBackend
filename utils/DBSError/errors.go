package DBSError

import "fmt"

type ConfigError struct {
	FuncName string
	Message  string
	Err      error
	Code     ConfigErrorCode
}

type ConfigErrorCode int

const (
	ErrMissingEnv ConfigErrorCode = iota + 1
	ErrInvalidValue
	ErrFileNotFound
	ErrInvalidMongoConfig
	ErrInvalidTwitchConfig
)

// Error implements the error interface.
func (e *ConfigError) Error() string {
	return fmt.Sprintf("[%s] Error %d - %s: %v", e.FuncName, e.Code, e.Message, e.Err)
}

// Unwrap allows unwrapping the underlying error.
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new DBSError.
func NewConfigError(funcName string, code ConfigErrorCode, message string, err error) *ConfigError {
	return &ConfigError{
		FuncName: funcName,
		Code:     code,
		Message:  message,
		Err:      err,
	}
}

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
