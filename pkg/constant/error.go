package constant

import "errors"

var (
	ErrUnsupportedAction = errors.New("unsupported action")
	ErrOnlyPostMethod    = errors.New("only POST method is supported")
)
