package main

import (
	"errors"
	"log"
	"os/exec"
	"path/filepath"
	"syscall"
)

const (
	BUILD_WAITING = iota
	BUILD_RUNNING
	BUILD_FAILED
	BUILD_CANCELED
	BUILD_SUCCESS
)

var stateNames = []string{
	"Wating",
	"Running",
	"Failed",
	"Canceled",
	"Success",
}

type Build struct {
	Rev        string
	State      int
	Path       string
	Output     OutputBuffer
	ReturnCode int

	cancel chan struct{}
}

func (b *Build) StateName() string {
	return stateNames[b.State]
}

// TODO: what to do with build state on error?
func (b *Build) Exec() error {
	script := filepath.Join(b.Path, "Seafile")

	cmd := exec.Command(script)
	cmd.Stdout = &b.Output // TODO: stderr
	defer b.Output.Close()

	err := cmd.Start()
	if err != nil {
		return err
	}

	b.State = BUILD_RUNNING
	waitResult := make(chan error)
	go func() { waitResult <- cmd.Wait() }()

	b.cancel = make(chan struct{}, 1)

	select {
	case err = <-waitResult:
		if err == nil {
			b.State = BUILD_SUCCESS
		} else if exit, ok := err.(*exec.ExitError); ok {
			b.State = BUILD_FAILED
			ws := exit.ProcessState.Sys().(syscall.WaitStatus) // will panic if not Unix
			b.ReturnCode = ws.ExitStatus()
		} else {
			return err
		}
	case <-b.cancel:
		log.Print("------------- Cancel -------------")
		err = cmd.Process.Kill()
		if err != nil {
			return err
		}
		b.State = BUILD_CANCELED
	}

	return nil
}

func (b *Build) Cancel() (err error) {
	if b.State == BUILD_RUNNING {
		b.cancel <- struct{}{}
	} else {
		err = errors.New("build must be in running state to cancel")
	}
	return
}
