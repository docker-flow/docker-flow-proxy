package actions

import "../proxy"

type Reloader interface {
	Execute() error
}

type Reload struct {}

func (m *Reload) Execute() error {
	if err := proxy.Instance.Reload(); err != nil {
		logPrintf(err.Error())
		return err
	}
	return nil
}

var NewReload = func() Reloader {
	return &Reload{}
}