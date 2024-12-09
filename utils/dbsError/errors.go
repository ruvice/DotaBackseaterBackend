package dbsError

import "fmt"

type DBSError struct {
	FuncName string
	Message  string
	Err      error
	Code     int
}

// Error implements the error interface.
func (e *DBSError) Error() string {
	return fmt.Sprintf("[%s] Error %d - %s: %v", e.FuncName, e.Code, e.Message, e.Err)
}

// Unwrap allows unwrapping the underlying error.
func (e *DBSError) Unwrap() error {
	return e.Err
}
