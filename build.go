package main

import (
	"os/exec"
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
		b.State = BUILD_FAILURE
		// TODO: do something with return code
	}
	b.Output.Close()
}
