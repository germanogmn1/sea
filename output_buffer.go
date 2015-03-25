package main

import (
	"log"
	"sync"
)

type OutputBuffer struct {
	data      []byte
	dataLock  sync.RWMutex
	listeners []chan []byte
	// listenersLock sync.Mutex
}

func NewEmptyOutputBuffer() OutputBuffer {
	return OutputBuffer{stream: make(chan []byte)}
}

func NewFilledOutputBuffer(content []byte) OutputBuffer {
	result := OutputBuffer{
		data:   content,
		stream: make(chan []byte),
	}
	close(result.stream)
	return result
}

// io.Writer
func (o *OutputBuffer) Write(p []byte) (int, error) {
	for _, c := range o.listeners {
		dup := make([]byte, len(p), len(p))
		copy(dup, p)
		c <- dup
	}
	o.dataLock.Lock()
	o.data = append(o.data, dup...)
	o.dataLock.Unlock()
	return len(p), nil
}

// io.Closer
func (o *OutputBuffer) Close() error {
	for _, c := range o.listeners {
		close(c)
	}
	o.listeners = nil
	log.Print("&&&&& CLOSED &&&&&")
	return nil
}

func (o *OutputBuffer) ReadChunks() chan []byte {
	response := make(chan []byte)
	go func() {
		o.dataLock.RLock()
		if len(o.data) > 0 {
			response <- o.data
		}
		o.dataLock.RUnlock()

		newListener := make(chan []byte)
		o.listeners = append(o.listeners, newListener)
		for chunk := range newListener {
			response <- chunk
		}
		close(response)
	}()
	return response
}
