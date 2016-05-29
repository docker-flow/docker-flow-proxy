package registry

import "github.com/stretchr/testify/mock"

type RegistrarableMock struct {
	mock.Mock
}

func (m *RegistrarableMock) PutService(address, instanceName string, r Registry) error {
	params := m.Called(address, instanceName, r)
	return params.Error(0)
}

func (m *RegistrarableMock) SendPutRequest(address, serviceName, key, value, instanceName string, c chan error) {
	m.Called(address, serviceName, key, value, instanceName, c)
}

func (m *RegistrarableMock) DeleteService(address, serviceName, instanceName string) error {
	m.Called(address, serviceName, instanceName)
	return nil
}

func (m *RegistrarableMock) SendDeleteRequest(address, serviceName, key, value, instanceName string, c chan error) {
	m.Called(address, serviceName, key, value, instanceName, c)
}

func GetRegistrarableMock(skipMethod string) *RegistrarableMock {
	mockObj := new(RegistrarableMock)
	if skipMethod != "PutService" {
		mockObj.On("PutService", mock.Anything, mock.Anything, mock.Anything)
	}
	if skipMethod != "SendPutRequest" {
		mockObj.On("SendPutRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}
	if skipMethod != "DeleteService" {
		mockObj.On("DeleteService", mock.Anything, mock.Anything, mock.Anything)
	}
	if skipMethod != "SendDeleteRequest" {
		mockObj.On("SendDeleteRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}
	return mockObj
}
