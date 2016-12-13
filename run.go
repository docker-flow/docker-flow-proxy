package main

import haproxy "./proxy"

type Runnable interface {
	Execute(args []string) error
}

type Run struct{}

var run Run

var NewRun = func() Executable {
	return &Run{}
}

func (m *Run) Execute(args []string) error {
	return haproxy.HaProxy{}.RunCmd([]string{})
}
