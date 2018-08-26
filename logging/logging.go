package logging

import (
	"log"

	"github.com/ziutek/syslog"
)

var logPrintf = log.Printf

type handler struct {
	*syslog.BaseHandler
}

func newHandler() *handler {
	h := handler{syslog.NewBaseHandler(100, nil, false)}
	go h.mainLoop()
	return &h
}

func (h *handler) mainLoop() {
	for {
		m := h.Get()
		logPrintf("HAPRoxy: %s%s\n", m.Tag, m.Content)
	}
}

// StartLogging listens to rsyslog and outputs entries to stdout
var StartLogging = func() {
	s := syslog.NewServer()
	s.AddHandler(newHandler())
	err := s.Listen("127.0.0.1:1514")
	if err != nil {
		logPrintf("%v", err)
	} else {
		logPrintf("NO ERROR")
	}
}
