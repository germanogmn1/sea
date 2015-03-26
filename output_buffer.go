package main

import (
	"log"
	"sync"
)

type OutputBuffer struct {
	data      []byte
	dataLock  sync.RWMutex
	listeners *listener
}

type listener struct {
	next, prev *listener

	data chan []byte
	quit chan struct{}
}

func NewEmptyOutputBuffer() OutputBuffer {
	return OutputBuffer{}
}

func NewFilledOutputBuffer(content []byte) OutputBuffer {
	result := OutputBuffer{
		data: content,
	}
	return result
}

// io.Writer
func (o *OutputBuffer) Write(p []byte) (int, error) {
	for e := o.listeners; e != nil; e = e.next {
		dup := make([]byte, len(p), len(p))
		copy(dup, p)
		select {
		case e.data <- dup:
		case <-e.quit:
			if e.next != nil {
				e.next.prev = e.prev
			}
			if e.prev != nil {
				e.prev.next = e.next
			}
		}
	}
	o.dataLock.Lock()
	o.data = append(o.data, p...)
	o.dataLock.Unlock()
	return len(p), nil
}

// io.Closer
func (o *OutputBuffer) Close() error {
	for e := o.listeners; e != nil; e = e.next {
		close(e.data)
	}
	o.listeners = nil
	log.Print("&&&&& CLOSED &&&&&")
	return nil
}

func (o *OutputBuffer) ReadChunks() (<-chan []byte, chan<- struct{}) {
	cdata := make(chan []byte)
	cquit := make(chan struct{})
	go func() {
		o.dataLock.RLock()
		if len(o.data) > 0 {
			cdata <- o.data
		}
		o.dataLock.RUnlock()

		e := new(listener)
		if o.listeners != nil {
			e.next = o.listeners
			o.listeners.prev = e
		}
		o.listeners = e
		e.data = cdata
		e.quit = cquit
	}()
	return cdata, cquit
}
