// Code generated by mockery 2.7.5. DO NOT EDIT.

package mock

import (
	mock "github.com/stretchr/testify/mock"
	kubernetes "k8s.io/client-go/kubernetes"

	zap "go.uber.org/zap"
)

// Provider is an autogenerated mock type for the Provider type
type Provider struct {
	mock.Mock
}

// RetrieveFrom provides a mock function with given fields: kubeConfig, log
func (_m *Provider) RetrieveFrom(kubeConfig string, log *zap.SugaredLogger) (kubernetes.Interface, error) {
	ret := _m.Called(kubeConfig, log)

	var r0 kubernetes.Interface
	if rf, ok := ret.Get(0).(func(string, *zap.SugaredLogger) kubernetes.Interface); ok {
		r0 = rf(kubeConfig, log)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(kubernetes.Interface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *zap.SugaredLogger) error); ok {
		r1 = rf(kubeConfig, log)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
