package lib_test

import (
	"context"
	"math/rand"
	"reflect"
	"testing"

	"github.com/samthor/nodejs-holder/lib"
)

func TestRunner(t *testing.T) {

	ctx := context.Background()

	runner, err := lib.New(ctx, nil)
	if err != nil {
		t.Fatalf("couldn't start runner: %v", err)
	}

	sourceNumber := rand.Int31() & 0xfff

	out, err := runner.Do(ctx, lib.Request[any]{
		Import: "./fortest.js",
		Method: "whatever",
		Args:   []any{sourceNumber, "hello", true},
	})
	if err != nil {
		t.Fatalf("couldn't run whatever method: %v", err)
	}

	// TODO: javascript! (shakes fist)
	f64, ok := out.(float64)
	i := int32(f64)
	if !ok || i != sourceNumber+1 {
		t.Fatalf("unexpected test answer=%v (type=%v), expected=%v", out, reflect.TypeOf(out), sourceNumber+1)
	}
}
