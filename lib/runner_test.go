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
	var out int32

	err = runner.Do(ctx, lib.Request{
		Import:   "./fortest.js",
		Method:   "whatever",
		Arg:      sourceNumber,
		Response: &out,
	})
	if err != nil {
		t.Fatalf("couldn't run whatever method: %v", err)
	}

	if out != sourceNumber+1 {
		t.Fatalf("unexpected test answer=%v (type=%v), expected=%v", out, reflect.TypeOf(out), sourceNumber+1)
	}
}

func TestWrap(t *testing.T) {
	ctx := context.Background()

	runner, err := lib.New(ctx, nil)
	if err != nil {
		t.Fatalf("couldn't start runner: %v", err)
	}

	wrap := lib.WrapHost[int32, int32](runner, lib.RequestMethod{
		Import: "./fortest.js",
		Method: "whatever",
	})

	var out int32
	err = wrap(ctx, 1, &out)
	if err != nil {
		t.Fatalf("could not do op: %v", err)
	}
	if out != 2 {
		t.Fatalf("unexpected answer for wrap, was: %+v", out)
	}

	err = wrap(ctx, 1234, nil)
	if err != nil {
		t.Fatalf("could not do nil response op: %v", err)
	}
}
