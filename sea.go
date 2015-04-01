package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/libgit2/git2go"
)

func Run() int {
	var pipePath, dbPath, webAddr string
	flag.StringVar(&webAddr, "addr", ":8080", "TCP address to listen on")
	flag.StringVar(&pipePath, "pipe", "./tmp/seapipe", "named pipe to listen for git hooks")
	flag.StringVar(&dbPath, "db", "./tmp/sea.db", "database file")
	flag.Parse()

	killed := make(chan os.Signal, 1)
	signal.Notify(killed, syscall.SIGINT, syscall.SIGTERM)

	if err := InitDB(dbPath); err != nil {
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
			RunningBuilds.CancelAll()
			wg.Wait()
			return 130
		}
	}

	return 0
}

// TODO: prevent hook script from blocking when writing on pipe
func main() {
	// os.Exit(Run())
	dir, err := ioutil.TempDir("", "repo")
	if err != nil {
		log.Fatal(err)
	}
	repo, err := git.Clone("git@github.com:germanogmn1/sea.git", dir, &git.CloneOptions{
		RemoteCallbacks: &git.RemoteCallbacks{
			CredentialsCallback:      credentialsCallback,
			CertificateCheckCallback: certificateCheckCallback,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Print(repo.Path())
}

func credentialsCallback(url string, username_from_url string, allowed_types git.CredType) (git.ErrorCode, *git.Cred) {
	log.Fatalf("credentialsCallback(%s, %s, %s)", url, username_from_url, allowed_types)
	return git.ErrOk, nil
}

func certificateCheckCallback(cert *git.Certificate, valid bool, hostname string) git.ErrorCode {
	log.Printf("certificateCheckCallback(%v, %v, %v)", cert, valid, hostname)
	return git.ErrOk
}
