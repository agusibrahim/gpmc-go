package errors

// CustomError is the base custom error type
type CustomError struct {
	Message string
}

// Error returns the error message
func (e *CustomError) Error() string {
	return e.Message
}

// NewCustomError creates a new CustomError
func NewCustomError(message string) *CustomError {
	return &CustomError{Message: message}
}

// UploadRejectedError is returned when a file upload is rejected by the API
type UploadRejectedError struct {
	CustomError
}

// NewUploadRejectedError creates a new UploadRejectedError
func NewUploadRejectedError(message string) *UploadRejectedError {
	return &UploadRejectedError{CustomError{Message: message}}
}

// SyncCycleError is returned when a sync cycle is detected (sync token unchanged)
type SyncCycleError struct {
	CustomError
}

// NewSyncCycleError creates a new SyncCycleError
func NewSyncCycleError(message string) *SyncCycleError {
	return &SyncCycleError{CustomError{Message: message}}
}
