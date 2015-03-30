package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
)

type GitHook struct {
	RepoPath string
	OldRev   string
	NewRev   string
	RefName  string
}

func ListenGitHooks(pipePath string, wg *sync.WaitGroup, stop chan struct{}) (<-chan GitHook, <-chan error) {
	results := make(chan GitHook)
	errors := make(chan error, 1)
	go func() {
		defer wg.Done()
		defer close(results)
		defer close(errors)
		pipe, err := createPipe(pipePath)
		if err != nil {
			errors <- err
			return
		}
		defer removePipe(pipe)

		log.Printf("Listening for git hooks on %s", pipePath)

		lines, readErrs := readPipe(pipe)
		for {
			select {
			case line := <-lines:
				values := ShellSplit(line)
				if len(values) != 4 {
					errors <- fmt.Errorf("Invalid hook value: %q", line)
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
			case err = <-readErrs:
				errors <- err
				return
			case <-stop:
				return
			}
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
		err = os.Remove(file.Name())
	}
	return err
}

func readPipe(f *os.File) (<-chan string, <-chan error) {
	results := make(chan string)
	errors := make(chan error, 1)
	go func() {
		defer close(results)
		defer close(errors)
		buffer := make([]byte, 512)
		for {
			n, err := f.Read(buffer)
			if err != nil {
				errors <- err
				return
			}
			results <- string(buffer[:n])
		}
	}()
	return results, errors
}
