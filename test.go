package main

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

func Register() (data <-chan element, quit chan<- struct{}) {
	e := &entry{
		send: make(chan element),
		quit: make(chan struct{}),
		next: listeners,
	}
	if listeners != nil {
		listeners.prev = e
	}
	listeners = e
	return listeners.send, listeners.quit
}

func unregister(e *entry) {
	close(e.send)
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	e.prev = nil
	e.next = nil
	return
}

func Broadcast(message element) {
	for e := listeners; e != nil; e = e.next {
		select {
		case e.send <- message:
		case <-e.quit:
			unregister(e)
		}
	}
}

func Close() {
	for e := listeners; e != nil; e = e.next {
		close(e.send)
		if e.prev != nil {
			e.prev.next = e.next
		}
		if e.next != nil {
			e.next.prev = e.prev
		}
	}
}

func Listen(id int, recv <-chan element, quit chan<- struct{}) {
	defer wg.Done()
	if rand.Intn(10) == 0 {
		fmt.Printf("goroutine %d unregistered\n", id)
		close(quit)
	}
	for msg := range recv {
		fmt.Printf("goroutine %d received %q\n", id, msg)
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

	N := 10
	AddListeners(N)
	Broadcast("Hello!!!")
	AddListeners(N)
	Broadcast("World!")
	AddListeners(N)
	Broadcast("Bye loosers!")
	AddListeners(N)
	Close()

	wg.Wait()

	fmt.Fprintf(os.Stderr, "%d %v\n", N, time.Since(start))
}
