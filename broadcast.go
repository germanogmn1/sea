package test

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

var listeners *entry
var wg *sync.WaitGroup
var lid = 1

// data sent by channel
type element string

// doubly linked list
type entry struct {
	next, prev *entry

	send chan element
	quit chan struct{}
}

func Register() (data chan element, quit chan struct{}) {
	e := new(entry)
	if listeners != nil {
		e.next = listeners
		listeners.prev = e
	}
	listeners = e

	e.send = make(chan element)
	e.quit = make(chan struct{})
	return e.send, e.quit
}

func Broadcast(message element) {
	for e := listeners; e != nil; e = e.next {
		select {
		case e.send <- message:
		case <-e.quit:
			close(e.send)
			if e.next != nil {
				e.next.prev = e.prev
			}
			if e.prev != nil {
				e.prev.next = e.next
			}
		}
	}
}

func Close() {
	for e := listeners; e != nil; e = e.next {
		close(e.send)
		if e.next != nil {
			e.next.prev = e.prev
		}
		if e.prev != nil {
			e.prev.next = e.next
		}
	}
}

func Listen(id int, recv chan element, quit chan struct{}) {
	defer wg.Done()
	sum := 0
	if rand.Intn(10) == 1 {
		close(quit)
	}
	for msg := range recv {
		sum += len(msg)
	}
}

func AddListeners(n int) {
	wg.Add(n)
	for i := 0; i < n; i++ {
		recv, quit := Register()
		go Listen(lid, recv, quit)
		lid += 1
	}
}

func main() {
	wg = new(sync.WaitGroup)
	rand.Seed(time.Now().UnixNano())

	start := time.Now()

	N := 1000
	AddListeners(N)
	Broadcast("Hello!!!")
	AddListeners(N)
	Broadcast("World!")
	AddListeners(N)
	Broadcast("Bye loosers!")
	AddListeners(N)
	Close()

	wg.Wait()

	fmt.Fprintf(os.Stderr, "%d %v\n", N, time.Since(start).Seconds())
}
