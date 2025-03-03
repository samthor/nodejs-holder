package lib

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"sync"
)

type internalRequest struct {
	Request
	Cancel bool   `json:"cancel,omitempty"`
	Id     string `json:"id"`
}

type internalResponse struct {
	Id        string          `json:"id"`
	Status    string          `json:"status"`
	Response  json.RawMessage `json:"res,omitempty"`
	ErrorText string          `json:"errtext,omitempty"`
}

// New starts a new Host that can execute Node.js code.
func New(ctx context.Context, options *Options) (Host, error) {
	if options == nil {
		options = &Options{}
	}

	c := exec.Command("node",
		// nb. can't pass code in NODE_OPTIONS
		"-e",
		jsHarness,
	)

	if options.Flags.DisableExperimentalWarning {
		c.Args = append(c.Args, "--disable-warning=ExperimentalWarning")
	}
	if options.Flags.TransformTypes {
		c.Args = append(c.Args, "--experimental-transform-types")
	}

	// setup stdin/stderr and pass to optional logger
	nodeOut, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	nodeErr, err := c.StderrPipe()
	if err != nil {
		return nil, err
	}
	doLog := func(r io.Reader, stderr bool) {
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			line := scan.Text()
			if options.Log != nil {
				options.Log(line, stderr)
			}
		}
	}

	var filesToClose []*os.File
	defer func() {
		for _, f := range filesToClose {
			f.Close()
		}
	}()

	// create pipes for message passing
	// os.Pipe seems to create R/W pipes in that order (undocumented!)
	remoteRead, localWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	filesToClose = append(filesToClose, remoteRead, localWrite)

	localRead, remoteWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	filesToClose = append(filesToClose, localRead, remoteWrite)

	c.ExtraFiles = []*os.File{remoteRead, remoteWrite}

	// start process :partyparrot:
	err = c.Start()
	if err != nil {
		return nil, err
	}

	// create helper obj
	nh := &nodeHost{
		proc:         c.Process,
		outerContext: ctx,
		localRead:    localRead,
		localWrite:   localWrite,
		waiters:      map[int]chan<- internalResponse{},
	}

	// start reading stdin/stderr
	go doLog(nodeOut, false)
	go doLog(nodeErr, true)

	// start reply handler
	go func() {
		scan := bufio.NewScanner(localRead)
		for scan.Scan() {
			line := scan.Bytes()

			var res internalResponse
			err := json.Unmarshal(line, &res)
			if err != nil {
				// TODO: something else?
				panic(fmt.Sprintf("could not decode from runner: %v", err))
			}

			go nh.handleResponse(&res)
		}
	}()

	waitFilesToClose := filesToClose
	filesToClose = nil

	// wait for stuff to die
	go func() {
		go func() {
			<-ctx.Done()
			c.Process.Kill()
		}()

		c.Process.Wait()
		for _, f := range waitFilesToClose {
			f.Close()
		}
	}()

	return nh, nil
}

type nodeHost struct {
	lock         sync.Mutex
	outerContext context.Context
	seq          int
	localRead    io.Reader
	localWrite   io.Writer
	waiters      map[int]chan<- internalResponse
	proc         *os.Process
}

func (nh *nodeHost) Stop() error {
	nh.lock.Lock()
	defer nh.lock.Unlock()

	return writeJsonLine(nh.localWrite, internalRequest{
		Id:     "", // with blank ID, shuts process
		Cancel: true,
	})
}

func (nh *nodeHost) handleResponse(res *internalResponse) {
	seq, _ := strconv.Atoi(res.Id)

	if seq <= 0 {
		log.Printf("got invalid seq from Node.JS: %v", seq)
		return // bad seq?
	}

	nh.lock.Lock()
	defer nh.lock.Unlock()

	w := nh.waiters[seq]
	if w == nil {
		log.Printf("got unknown seq from Node.JS: %v", seq)
		return
	}
	delete(nh.waiters, seq)

	select {
	case w <- *res:
		// should always work because we create buffered ch
	default:
		panic("cannot write response, ch not buffered?")
	}
}

func writeJsonLine(w io.Writer, arg any) error {
	err := json.NewEncoder(w).Encode(arg)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}

func (n *nodeHost) Do(ctx context.Context, req Request) error {
	n.lock.Lock()
	n.seq++

	seq := n.seq
	id := strconv.Itoa(seq)

	err := writeJsonLine(n.localWrite, internalRequest{
		Request: req,
		Id:      id,
	})
	if err != nil {
		// probably host stopped
		n.lock.Unlock()
		return err
	}

	ch := make(chan internalResponse, 1)
	n.waiters[seq] = ch
	defer func() {
		n.lock.Lock()
		defer n.lock.Unlock()
		delete(n.waiters, seq) // may already be deleted, but this safeguards us
	}()

	n.lock.Unlock()

	select {
	case out := <-ch:
		if out.Status == "ok" {
			// only decode if the output is not undefined and the _target_ is not nil
			if len(out.Response) != 0 && !isTypeNil(req.Response) {
				return json.Unmarshal(out.Response, req.Response)
			}
			return nil
		}
		return fmt.Errorf("from Node.js %s:\n%s", out.Status, out.ErrorText)

	case <-ctx.Done():
		err := writeJsonLine(n.localWrite, internalRequest{
			Id:     id,
			Cancel: true,
		})
		if err != nil {
			return err
		}
		// we don't wait for the real reply
		return ctx.Err()

	case <-n.outerContext.Done():
		return n.outerContext.Err()
	}
}

func (nh *nodeHost) Wait() error {
	_, err := nh.proc.Wait()
	return err
}

func isTypeNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}
