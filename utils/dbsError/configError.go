package dbsError

type ConfigError struct {
	DBSError DBSError
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

// Error implements error.
func (e *ConfigError) Error() string {
	return e.DBSError.Error()
}

// NewConfigError creates a new DBSError.
func NewConfigError(funcName string, code ConfigErrorCode, message string, err error) *ConfigError {
	return &ConfigError{
		DBSError: DBSError{
			FuncName: funcName,
			Message:  message,
			Code:     int(code),
			Err:      err,
		},
		Code: code,
	}
}
