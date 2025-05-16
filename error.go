package sqlite

type Error struct {
	msg  string
	code int
}

// Error implements error.
func (e *Error) Error() string { return e.msg }

// Code returns the sqlite result code for this error.
func (e *Error) Code() int { return e.code }
