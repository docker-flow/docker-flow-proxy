package main

import "fmt"

type Removable interface {
	Executable
}

type Remove struct {
	ConfigsPath   string `short:"c" long:"configs-path" default:"/cfg" description:"The path to the configurations directory"`
	ConsulAddress string `short:"a" long:"consul-address" env:"CONSUL_ADDRESS" required:"true" description:"The address of the Consul service (e.g. /api/v1/my-service)."`
	InstanceName  string `long:"proxy-instance-name" env:"PROXY_INSTANCE_NAME" default:"docker-flow" required:"true" description:"The name of the proxy instance."`
	ServiceName   string `short:"s" long:"service-name" required:"true" description:"The name of the service that should be removed (e.g. my-service)."`
	TemplatesPath string `short:"t" long:"templates-path" default:"/cfg/tmpl" description:"The path to the templates directory"`
}

var remove Remove

var NewRemove = func(serviceName, configsPath, templatesPath, consulAddress, instanceName string) Removable {
	return &Remove{
		ServiceName:   serviceName,
		TemplatesPath: templatesPath,
		ConfigsPath:   configsPath,
		ConsulAddress: consulAddress,
		InstanceName:  instanceName,
	}
}

func (m *Remove) Execute(args []string) error {
	if err := m.removeFiles(m.TemplatesPath, m.ServiceName, m.ConsulAddress, m.InstanceName); err != nil {
		logPrintf(err.Error())
		return err
	}
	if err := proxy.CreateConfigFromTemplates(m.TemplatesPath, m.ConfigsPath); err != nil {
		logPrintf(err.Error())
		return err
	}
	if err := proxy.Reload(); err != nil {
		logPrintf(err.Error())
		return err
	}
	return nil
}

func (m *Remove) removeFiles(templatesPath, serviceName, registryAddress, instanceName string) error {
	paths := []string{
		fmt.Sprintf("%s/%s-fe.cfg", templatesPath, serviceName),
		fmt.Sprintf("%s/%s-be.cfg", templatesPath, serviceName),
	}
	mu.Lock()
	defer mu.Unlock()
	for _, path := range paths {
		if err := osRemove(path); err != nil {
			return err
		}
	}
	if err := registryInstance.DeleteService(registryAddress, serviceName, instanceName); err != nil {
		return fmt.Errorf("Could not remove the service from Consul\n%s", err.Error())
	}
	return nil
}
