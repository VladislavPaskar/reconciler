// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	context "context"

	appsv1 "k8s.io/api/apps/v1"

	dynamic "k8s.io/client-go/dynamic"

	kubernetes "k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock "github.com/stretchr/testify/mock"

	reconcilerkubernetes "github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes"

	types "k8s.io/apimachinery/pkg/types"

	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	v1 "k8s.io/api/core/v1"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// Clientset provides a mock function with given fields:
func (_m *Client) Clientset() (kubernetes.Interface, error) {
	ret := _m.Called()

	var r0 kubernetes.Interface
	if rf, ok := ret.Get(0).(func() kubernetes.Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(kubernetes.Interface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Delete provides a mock function with given fields: ctx, manifest, namespace
func (_m *Client) Delete(ctx context.Context, manifest string, namespace string) ([]*reconcilerkubernetes.Resource, error) {
	ret := _m.Called(ctx, manifest, namespace)

	var r0 []*reconcilerkubernetes.Resource
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []*reconcilerkubernetes.Resource); ok {
		r0 = rf(ctx, manifest, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*reconcilerkubernetes.Resource)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, manifest, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Deploy provides a mock function with given fields: ctx, manifest, namespace, interceptors
func (_m *Client) Deploy(ctx context.Context, manifest string, namespace string, interceptors ...reconcilerkubernetes.ResourceInterceptor) ([]*reconcilerkubernetes.Resource, error) {
	_va := make([]interface{}, len(interceptors))
	for _i := range interceptors {
		_va[_i] = interceptors[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, manifest, namespace)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []*reconcilerkubernetes.Resource
	if rf, ok := ret.Get(0).(func(context.Context, string, string, ...reconcilerkubernetes.ResourceInterceptor) []*reconcilerkubernetes.Resource); ok {
		r0 = rf(ctx, manifest, namespace, interceptors...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*reconcilerkubernetes.Resource)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string, ...reconcilerkubernetes.ResourceInterceptor) error); ok {
		r1 = rf(ctx, manifest, namespace, interceptors...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDynClient provides a mock function with given fields:
func (_m *Client) GetDynClient() (dynamic.Interface, error) {
	ret := _m.Called()

	var r0 dynamic.Interface
	if rf, ok := ret.Get(0).(func() dynamic.Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(dynamic.Interface)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSecret provides a mock function with given fields: ctx, name, namespace
func (_m *Client) GetSecret(ctx context.Context, name string, namespace string) (*v1.Secret, error) {
	ret := _m.Called(ctx, name, namespace)

	var r0 *v1.Secret
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v1.Secret); ok {
		r0 = rf(ctx, name, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Secret)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStatefulSet provides a mock function with given fields: ctx, name, namespace
func (_m *Client) GetStatefulSet(ctx context.Context, name string, namespace string) (*appsv1.StatefulSet, error) {
	ret := _m.Called(ctx, name, namespace)

	var r0 *appsv1.StatefulSet
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, name, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Kubeconfig provides a mock function with given fields:
func (_m *Client) Kubeconfig() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// ListResource provides a mock function with given fields: resource, lo
func (_m *Client) ListResource(resource string, lo metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	ret := _m.Called(resource, lo)

	var r0 *unstructured.UnstructuredList
	if rf, ok := ret.Get(0).(func(string, metav1.ListOptions) *unstructured.UnstructuredList); ok {
		r0 = rf(resource, lo)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*unstructured.UnstructuredList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, metav1.ListOptions) error); ok {
		r1 = rf(resource, lo)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PatchUsingStrategy provides a mock function with given fields: kind, name, namespace, p, strategy
func (_m *Client) PatchUsingStrategy(kind string, name string, namespace string, p []byte, strategy types.PatchType) error {
	ret := _m.Called(kind, name, namespace, p, strategy)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, []byte, types.PatchType) error); ok {
		r0 = rf(kind, name, namespace, p, strategy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
