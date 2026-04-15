// Package scenario provides error definitions for the scenario engine.
package scenario

import "errors"

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrConnectionNotFound   = errors.New("TCP connection not found")
	ErrScenarioAlreadyRunning = errors.New("scenario already running")
	ErrUnknownTestCase      = errors.New("unknown test case")
	ErrNotRunning           = errors.New("scenario not running")
	ErrSendNotSet           = errors.New("send function not set")
	ErrTimeout              = errors.New("step timeout")
)
