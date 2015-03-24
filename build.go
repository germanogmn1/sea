package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"syscall"
)

type BuildState uint

const (
	BuildWaiting BuildState = iota
	BuildRunning
	BuildFailed
	BuildCanceled
	BuildSuccess
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
	State      BuildState
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

	b.State = BuildRunning
	waitResult := make(chan error)
	go func() { waitResult <- cmd.Wait() }()

	b.cancel = make(chan struct{}, 1) // TODO: why this channel have to be buffered?

	select {
	case err = <-waitResult:
		if err == nil {
			b.State = BuildSuccess
		} else if exit, ok := err.(*exec.ExitError); ok {
			b.State = BuildFailed
			ws := exit.ProcessState.Sys().(syscall.WaitStatus) // will panic if not Unix
			b.ReturnCode = ws.ExitStatus()
		} else {
			return err
		}
	case <-b.cancel:
		err = syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
		if err != nil && err != syscall.ESRCH { // Already finished
			return err
		}

		b.State = BuildCanceled
	}

	return nil
}

func (b *Build) Cancel() (err error) {
	if b.State == BuildRunning {
		b.cancel <- struct{}{} // TODO: maybe this shouldn't block...
	} else {
		err = errors.New("build must be in running state to cancel")
	}
	return
}
