package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
)

var Config struct {
	WebAddr   string
	PipePath  string
	DBPath    string
	ReposPath string
}

func Run() int {
	flag.StringVar(&Config.WebAddr, "addr", ":8080", "TCP address for web server to listen on")
	flag.StringVar(&Config.PipePath, "pipe", "./tmp/seapipe", "named pipe to listen for git hooks")
	flag.StringVar(&Config.DBPath, "db", "./tmp/sea.db", "database file")
	flag.StringVar(&Config.ReposPath, "repos", "./tmp/repos", "directory to store registered repositories")
	flag.Parse()

	var err error
	for _, dir := range [...]string{
		Config.ReposPath,
		filepath.Base(Config.PipePath),
		filepath.Base(Config.DBPath),
	} {
		if err = os.MkdirAll(dir, 0); err != nil {
			log.Print(err)
			return 1
		}
	}

	killed := make(chan os.Signal, 1)
	signal.Notify(killed, syscall.SIGINT, syscall.SIGTERM)

	if err = InitDB(); err != nil {
		log.Print(err)
		return 1
	}
	defer DB.Close()

	webErrors := WebServer()

	var wg sync.WaitGroup

	quit := make(chan struct{})
	wg.Add(1)
	hooks, hookErrors := ListenGitHooks(&wg, quit)

	for {
		select {
		case hook := <-hooks:
			wg.Add(1)
			go func() {
				StartLocalBuild(hook)
				wg.Done()
			}()
		case err := <-webErrors:
			log.Print(err)
			return 1
		case err := <-hookErrors:
			log.Print(err)
			return 1
		case sig := <-killed:
			log.Printf("Catched signal %q. Exiting...", sig)
			close(quit)
			RunningBuilds.CancelAll()
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
