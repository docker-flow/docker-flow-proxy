package main

type Run struct {}

var run Run

func (m Run) Execute(args []string) error {
	return HaProxy{}.RunCmd([]string{})
}

