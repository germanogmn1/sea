package main

import (
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

func NewEmptyOutputBuffer() *OutputBuffer {
	o := new(OutputBuffer)
	o.cond = sync.NewCond(o.RLocker())
	return o
}

func NewFilledOutputBuffer(content []byte) *OutputBuffer {
	o := NewEmptyOutputBuffer()
	o.data = content
	o.done = true
	return o
}

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
