package main

import "fmt"

type Removable interface {
	Executable
}

type Remove struct {
	ServiceName   string `short:"s" long:"service-name" required:"true" description:"The name of the service that should be removed (e.g. my-service)."`
	ConfigsPath   string `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	TemplatesPath string `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
}

var remove Remove

var NewRemove = func(serviceName, configsPath, templatesPath string) Removable {
	return &Remove{
		ServiceName:   serviceName,
		TemplatesPath: templatesPath,
		ConfigsPath:   configsPath,
	}
}

func (m *Remove) Execute(args []string) error {
	path := fmt.Sprintf("%s/%s.cfg", m.TemplatesPath, m.ServiceName)
	mu.Lock()
	defer mu.Unlock()
	if err := osRemove(path); err != nil {
		return err
	}
	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		return err
	}
	return proxy.Reload()
}
