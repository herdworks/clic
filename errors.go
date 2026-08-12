package clic

import "errors"

// ErrHelpNotImplemented reports a namespace without a supplied help callback.
var ErrHelpNotImplemented = errors.New("command help is not implemented")
