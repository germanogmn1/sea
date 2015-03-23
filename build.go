package main

import (
	"errors"
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
	cmd.Stdout = &b.Output
	cmd.Stderr = &b.Output
	defer b.Output.Close()

	err := cmd.Start()
	if err != nil {
		return err
	}

	b.State = BUILD_RUNNING
	waitResult := make(chan error)
	go func() { waitResult <- cmd.Wait() }()

	b.cancel = make(chan struct{}, 1) // TODO: why this channel have to be buffered?

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
		b.cancel <- struct{}{} // TODO: maybe this shouldn't block...
	} else {
		err = errors.New("build must be in running state to cancel")
	}
	return
}
