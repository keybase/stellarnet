package errors

// StateError are errors indicating invalid state. Type is used to differentiate
// between network, i/o, marshaling, bad usage etc. errors and actual state errors.
// You can use type assertion or type switch to check for type.
type StateError struct {
	error
}

// NewStateError creates a new StateError.
func NewStateError(err error) StateError {
	return StateError{err}
}
