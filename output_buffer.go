package main

import "sync"

type OutputBuffer struct {
	data   []byte
	stream chan []byte
	mutex  sync.RWMutex
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

// io.Writer implementation
func (o *OutputBuffer) Write(p []byte) (int, error) {
	dup := make([]byte, len(p), len(p))
	copy(dup, p)
	select {
	case o.stream <- dup:
	default:
	}
	o.mutex.Lock()
	o.data = append(o.data, dup...)
	o.mutex.Unlock()
	return len(p), nil
}

func (o *OutputBuffer) Close() { close(o.stream) }

func (o *OutputBuffer) ReadChunks() chan []byte {
	response := make(chan []byte)
	go func() {
		o.mutex.RLock()
		if len(o.data) > 0 {
			response <- o.data
		}
		o.mutex.RUnlock()
		for chunk := range o.stream {
			response <- chunk
		}
		close(response)
	}()
	return response
}
