package actions

import "../proxy"

type Reloader interface {
	Execute(recreate bool, listenerAddr string) error
}

type Reload struct{}

func (m *Reload) Execute(recreate bool, listenerAddr string) error {
	if len(listenerAddr) > 0 {
		recon := NewReconfigure(BaseReconfigure{}, proxy.Service{}, "")
		if err := recon.ReloadServicesFromListener([]string{}, "", "", listenerAddr); err != nil {
			logPrintf(err.Error())
			return err
		}
	} else {
		if recreate {
			if err := proxy.Instance.CreateConfigFromTemplates(); err != nil {
				logPrintf(err.Error())
				return err
			}
		}
		if err := proxy.Instance.Reload(); err != nil {
			logPrintf(err.Error())
			return err
		}
	}
	return nil
}

var NewReload = func() Reloader {
	return &Reload{}
}
