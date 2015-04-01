package main

import (
	"fmt"
	"io"
	"sync"
)

type OutputBuffer struct {
	sync.RWMutex
	cond *sync.Cond
	done bool
	data []byte
}

type reader struct {
	*OutputBuffer
	index int
}

// fmt.Stringer
func (o *OutputBuffer) String() string {
	o.RLock()
	s := "open"
	if o.done {
		s = "done"
	}
	str := fmt.Sprintf("OutputBuffer[%s]{%q}", s, o.data)
	o.RUnlock()
	return str
}

func NewOutputBuffer() *OutputBuffer {
	o := new(OutputBuffer)
	o.cond = sync.NewCond(o.RLocker())
	return o
}

// io.Writer
func (o *OutputBuffer) Write(p []byte) (int, error) {
	o.Lock()
	o.data = append(o.data, p...)
	o.Unlock()
	o.cond.Broadcast()
	return len(p), nil
}

func (o *OutputBuffer) Stream() io.Reader {
	return &reader{o, 0}
}

func (o *OutputBuffer) End() {
	o.Lock()
	o.done = true
	o.Unlock()
	o.cond.Broadcast()
}

func (o *OutputBuffer) Bytes() []byte {
	return o.data
}

// io.Reader
func (r *reader) Read(p []byte) (n int, err error) {
	r.cond.L.Lock()
	for !(r.done || r.index < len(r.data)) {
		r.cond.Wait()
	}
	if r.index < len(r.data) {
		n = copy(p, r.data[r.index:])
		r.index += n
	} else {
		err = io.EOF
	}
	r.cond.L.Unlock()
	return
}
