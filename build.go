package main

import (
	"os/exec"
	"syscall"
)

const (
	BUILD_WAITING = iota
	BUILD_RUNNING
	BUILD_SUCCESS
	BUILD_FAILURE
)

var stateNames = []string{
	"Wating",
	"Running",
	"Success",
	"Failure",
}

type Build struct {
	Rev        string
	State      int
	ScriptPath string
	Output     OutputBuffer
	ReturnCode int
	// Previous *Build ??? in case o retry
}

func (b *Build) StateName() string {
	return stateNames[b.State]
}

func (b *Build) Exec() {
	b.State = BUILD_RUNNING
	cmd := exec.Command(b.ScriptPath)
	cmd.Stdout = &b.Output
	// TODO: stderr
	err := cmd.Run()
	if err == nil {
		b.State = BUILD_SUCCESS
	} else {
		// TODO: handle error if it's not a exec.ExitError
		ps := err.(*exec.ExitError).ProcessState
		ws := ps.Sys().(syscall.WaitStatus) // will panic if not Unix
		b.State = BUILD_FAILURE
		b.ReturnCode = ws.ExitStatus()
	}
	b.Output.Close()
}
