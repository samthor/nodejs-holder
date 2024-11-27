package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
)

func main() {

	err := hostNode()
	if err != nil {
		log.Fatalf("couldn't host node: %v", err)
	}

	log.Printf("done OK")

}

type Request struct {
	Import string `json:"import"`
	Method string `json:"method,omitempty"`
	Id     string `json:"id"`
	Args   []any  `json:"args,omitempty"`
}

func hostNode() error {
	c := exec.Command("node",
		"-e",
		jsHarness,
	)
	defer func() {
		if c.Process != nil {
			c.Process.Kill()
		}
	}()

	nodeOut, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	nodeErr, err := c.StderrPipe()
	if err != nil {
		return err
	}

	c.Env = os.Environ()
	c.Env = append(c.Env, "NODE_OPTIONS=--disable-warning=ExperimentalWarning --experimental-transform-types")

	// os.Pipe seems to create R/W pipes in that order (undocumented!)
	remoteRead, localWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer remoteRead.Close()
	defer localWrite.Close()

	localRead, remoteWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer localRead.Close()
	defer remoteWrite.Close()

	c.ExtraFiles = []*os.File{remoteRead, remoteWrite}

	err = c.Start()
	if err != nil {
		return err
	}

	doLog := func(r io.Reader) {
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			line := scan.Text()
			log.Printf("got line from proc: %s", line)
		}
	}
	go doLog(nodeErr)
	go doLog(nodeOut)

	go doLog(localRead)

	r := &Request{
		Import: "./other.ts",
		Id:     "123",
	}
	enc, _ := json.Marshal(r)
	localWrite.Write(enc)
	localWrite.Write([]byte("\n"))

	return c.Wait()
}
