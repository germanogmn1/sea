package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/boltdb/bolt"
)

var DB *bolt.DB

func Run() int {
	var pipePath, dbPath, webAddr string
	flag.StringVar(&webAddr, "addr", ":8080", "TCP address to listen on")
	flag.StringVar(&pipePath, "pipe", "./tmp/seapipe", "named pipe to listen for git hooks")
	flag.StringVar(&dbPath, "db", "./tmp/sea.db", "database file")
	flag.Parse()

	killed := make(chan os.Signal, 1)
	signal.Notify(killed, syscall.SIGINT, syscall.SIGTERM)

	var err error
	DB, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Print(err)
		return 1
	}
	defer DB.Close()

	webErrors := WebServer(webAddr)

	var wg sync.WaitGroup

	quit := make(chan struct{})
	wg.Add(1)
	hooks, hookErrors := ListenGitHooks(pipePath, &wg, quit)

	for {
		select {
		case hook := <-hooks:
			wg.Add(1)
			go StartLocalBuild(hook, &wg)
		case err := <-webErrors:
			log.Print(err)
			return 1
		case err := <-hookErrors:
			log.Print(err)
			return 1
		case sig := <-killed:
			log.Printf("Catched signal %q. Exiting...", sig)
			close(quit)
			for _, build := range buildList {
				if build.State == BuildRunning {
					err := build.Cancel()
					if err != nil {
						log.Print("Failed to stop build:", err)
					}
				}
			}
			wg.Wait()
			return 130
		}
	}

	return 0
}

// TODO: prevent hook script from blocking when writing on pipe
func main() {
	os.Exit(Run())
}
