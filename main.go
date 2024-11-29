package main

import (
	"context"
	"log"
	"time"

	njs "github.com/samthor/nodejs-holder/lib"
)

func main() {
	runner, err := njs.New(context.Background(), &njs.Options{
		Flags: njs.OptionsFlags{
			DisableExperimentalWarning: true,
			TransformTypes:             true,
		},
		Log: func(msg string, stderr bool) {
			log.Printf("! (stderr=%v) %s", stderr, msg)
		},
	})
	if err != nil {
		log.Fatalf("can't start runner: %v", err)
	}
	log.Printf("runner started: %+v", runner)

	tc, _ := context.WithTimeout(context.Background(), time.Second*1)
	out, err := runner.Do(tc, njs.Request[any]{
		Import: "./other.ts",
	})
	if err != nil {
		log.Printf("can't get answer: %v", err)
	}

	time.Sleep(time.Second * 10)

	log.Printf("got out: %+v", out)
}
