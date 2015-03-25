package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var listeners []chan string
var lock sync.RWMutex
var wg sync.WaitGroup

func Register() chan string {
	c := make(chan string)
	lock.Lock()
	listeners = append(listeners, c)
	lock.Unlock()
	return c
}

func Unregister(ch chan string) {
	lock.RLock()
	for i, c := range listeners {
		if c == ch {
			go func(ignore chan string) {
				for _ = range ignore {
				}
			}(c)
			lock.RUnlock()
			lock.Lock()
			listeners[i] = listeners[len(listeners)-1]
			listeners = listeners[:len(listeners)-1]
			lock.Unlock()
			return
		}
	}
	lock.RUnlock()
	panic("WTF")
}

func Broadcast(message string) {
	lock.RLock()
	for _, c := range listeners {
		c <- message
	}
	lock.RUnlock()
}

func Close() {
	lock.RLock()
	for _, c := range listeners {
		close(c)
	}
	lock.RUnlock()
	lock.Lock()
	listeners = nil
	lock.Unlock()
}

func Listen(id int, ch chan string) {
	defer wg.Done()
	if rand.Intn(3) == 0 {
		fmt.Printf("goroutine %d unregistered\n", id)
		Unregister(ch)
		return
	}
	for msg := range ch {
		fmt.Printf("goroutine %d received %q\n", id, msg)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go Listen(i, Register())
	}

	Broadcast("Hello!!!")
	Broadcast("World!")
	Broadcast("Bye loosers!")
	Close()

	wg.Wait()
	fmt.Println("done...")
}
