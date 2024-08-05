package utils

type Error struct {
	Message string
	Err     error
}

func (e *Error) Error() string {
	return e.Message + ": " + e.Err.Error()
}

func ErrorWrapper(msg string, err error) *Error {
	return &Error{msg, err}
}
