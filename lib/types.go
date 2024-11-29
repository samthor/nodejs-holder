package lib

import (
	"context"
)

// Request is made to a Node.JS runner.
type Request[X any] struct {
	Import string `json:"import,omitempty"`
	Method string `json:"method,omitempty"`
	Args   []X    `json:"args,omitempty"`
}

// Host represents an active Node.js process.
type Host interface {
	// Do performs an operation on the running Node.js code.
	Do(context.Context, Request[any]) (any, error)

	// Wait waits for the Node.js process to exit.
	Wait() error

	// Stop sends a stop message to the Node.js process.
	// It will die with a zero status code, but there are no guarantees about in-flight tasks.
	Stop() error
}

type OptionsFlags struct {
	DisableExperimentalWarning bool
	TransformTypes             bool
}

type Options struct {
	Flags      OptionsFlags
	ExtraFlags []string

	// If non-nil, Log receives all stdout/stderr messages from the Node.js process.
	Log func(msg string, stderr bool)
}
