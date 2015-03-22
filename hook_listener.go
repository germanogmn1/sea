package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

type GitHook struct {
	RepoPath string
	OldRev   string
	NewRev   string
	RefName  string
}

func ListenGitHooks(pipePath string) (<-chan GitHook, <-chan error) {
	results := make(chan GitHook)
	errors := make(chan error)
	go func() {
		defer close(results)
		defer close(errors)
		pipe, err := createPipe(pipePath)
		if err != nil {
			errors <- err
			return
		}
		defer removePipe(pipe)

		log.Printf("Listening for git hooks on %s", pipePath)

		readBuffer := make([]byte, 512)
		for {
			var n int
			n, err = pipe.Read(readBuffer)
			if err != nil {
				errors <- err
				return
			}
			hookString := string(readBuffer[:n])
			values := ShellSplit(hookString)
			if len(values) != 4 {
				err = fmt.Errorf("Invalid hook value: %q", hookString)
				errors <- err
				return
			}
			hook := GitHook{
				RepoPath: values[0],
				OldRev:   values[1],
				NewRev:   values[2],
				RefName:  values[3],
			}
			log.Printf("hook: %#v", hook)
			results <- hook
		}
	}()
	return results, errors
}

func createPipe(path string) (file *os.File, err error) {
	oldmask := syscall.Umask(0)
	err = syscall.Mkfifo(path, 0622)
	syscall.Umask(oldmask)
	if err != nil {
		return
	}
	file, err = os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		os.Remove(path) // TODO: ignore error?
	}
	return
}

func removePipe(file *os.File) (err error) {
	if file != nil {
		err = file.Close()
		if err == nil {
			err = os.Remove(file.Name())
		}
	}
	return err
}
