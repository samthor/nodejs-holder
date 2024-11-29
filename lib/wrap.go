package lib

import (
	"context"
)

// WrapHost wraps a host with convenient type information.
func WrapHost[X any, Y any](nh Host, rm RequestMethod) func(context.Context, X, *Y) error {
	return func(ctx context.Context, x X, y *Y) error {
		var req Request
		req.Import = rm.Import
		req.Method = rm.Method
		req.Arg = x
		req.Response = y

		return nh.Do(ctx, req)
	}
}
