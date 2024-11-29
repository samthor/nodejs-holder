// Package lib exports a helper that runs Node.js on your behalf to call functions.
// Basically used to wrap up Node-specific code that needs to run within your Go server.

package lib

import (
	"context"
)

type RequestMethod struct {
	// Import specifies the path that Node.js will call `import(...)` on.
	Import string `json:"import"`

	// Method is the method name used from the ESM import. If unspecified, uses `"default"`.
	Method string `json:"method,omitempty"`
}

// Request specifies the import/method to call.
type Request struct {
	// Import specifies the path that Node.js will call `import(...)` on.
	Import string `json:"import"`

	// Method is the method name used from the ESM import. If unspecified, uses `"default"`.
	Method string `json:"method,omitempty"`

	// Arg is the argument passed 1st to the function.
	Arg any `json:"arg,omitempty"`

	// Response has the resulting value placed inside it. It must be a pointer to something.
	Response any `json:"-"`
}

// Host represents an active Node.js process.
type Host interface {
	// Do performs an operation on the running Node.js code.
	Do(ctx context.Context, r Request) error

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
