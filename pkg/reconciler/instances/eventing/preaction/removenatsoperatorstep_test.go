package preaction

import (
	"context"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"testing"
	"time"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8sversion "k8s.io/apimachinery/pkg/version"

	"github.com/kyma-incubator/reconciler/pkg/reconciler/chart"
	pmock "github.com/kyma-incubator/reconciler/pkg/reconciler/chart/mocks"
	"github.com/stretchr/testify/mock"

	"k8s.io/client-go/discovery"

	corev1 "k8s.io/api/core/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"

	//natsv1alpha2 "github.com/nats-io/nats-operator/pkg/apis/nats/v1alpha2"
	//netoworkingistio "istio.io/client-go/"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes"
	"go.uber.org/zap"
	apiextensionsapis "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/restmapper"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/stretchr/testify/require"

	"github.com/kyma-incubator/reconciler/pkg/logger"
	"github.com/kyma-incubator/reconciler/pkg/reconciler"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/service"
)

func TestDeletingNatsOperatorResources(t *testing.T) {
	setup := func() (removeNatsOperatorStep, *service.ActionContext, dynamic.Interface) {
		ctx := context.TODO()
		log := logger.NewLogger(false)

		content, err := ioutil.ReadFile("natsOperatorResources.yaml")
		require.NoError(t, err)

		natsResources := convertManifestToUnstructs(t, content)

		// mock the dynamic client and deploy the unstructs
		dynamicClient := generateDynamicClient(t, natsResources)

		fakeDiscovery := fakediscovery.FakeDiscovery{
			Fake:               &dynamicClient.Fake,
			FakedServerVersion: &k8sversion.Info{},
		}
		fakeDiscovery.Resources = generateAPIResourceList()
		fakeCached := FakeCached{
			&fakeDiscovery,
		}

		config := &kubernetes.Config{
			ProgressInterval: 5 * time.Second,
			ProgressTimeout:  10 * time.Second,
		}
		restConfig := &rest.Config{}

		mapper := restmapper.NewDeferredDiscoveryRESTMapper(&fakeCached)
		kubeClientMock, err := kubernetes.NewFakeKubernetesClient(dynamicClient, log, restConfig, mapper, config)
		require.NoError(t, err)

		action := removeNatsOperatorStep{
			kubeClientProvider: func(context *service.ActionContext, logger *zap.SugaredLogger) (kubernetes.Client, error) {
				return kubeClientMock, nil
			},
		}

		mockProvider := pmock.Provider{}
		mockManifest := chart.Manifest{
			Manifest: string(content),
		}
		mockProvider.On("RenderManifest", mock.Anything).Return(&mockManifest, nil)

		actionContext := &service.ActionContext{
			KubeClient:    kubeClientMock,
			Context:       ctx,
			Logger:        log,
			Task:          &reconciler.Task{},
			ChartProvider: &mockProvider,
		}
		return action, actionContext, dynamicClient
	}

	action, actionContext, clientset := setup()

	err := action.Execute(actionContext, actionContext.Logger)
	require.NoError(t, err)

	// check that the action's step deleted all the nats-operator resources including CRDs
	_, err = getNATSDeployment(actionContext.Context, clientset, namespace)
	require.NotNil(t, err)
	require.True(t, k8sErrors.IsNotFound(err))

	_, err = getCRD(actionContext.Context, clientset, natsOperatorCRDsToDelete[0])
	require.NotNil(t, err)
	require.True(t, k8sErrors.IsNotFound(err))

	_, err = getCRD(actionContext.Context, clientset, natsOperatorCRDsToDelete[1])
	require.NotNil(t, err)
	require.True(t, k8sErrors.IsNotFound(err))
}

// convert resources from yaml manifest to runtime.Object unstructs
func convertManifestToUnstructs(t *testing.T, content []byte) []runtime.Object {
	unstructs, err := kubernetes.ToUnstructured(content, true)
	require.NoError(t, err)

	result := make([]runtime.Object, len(unstructs))
	for i, obj := range unstructs {
		crdUnstructMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
		crdUnstruct := &unstructured.Unstructured{Object: crdUnstructMap}
		result[i] = crdUnstruct
	}
	return result
}

func generateDynamicClient(t *testing.T, resources []runtime.Object) *fakeDynamic.FakeDynamicClient {
	scheme, err := getScheme(t)
	require.NoError(t, err)
	dynamicClient := fakeDynamic.NewSimpleDynamicClient(scheme, resources...)
	return dynamicClient
}

func generateAPIResourceList() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Namespaced: true, Kind: "Deployment", Group: "apps", Version: "v1"},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: "v1",
			},
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "serviceaccounts", Namespaced: true, Kind: "ServiceAccount", Group: "", Version: "v1"},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			GroupVersion: "rbac.authorization.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "clusterroles", Namespaced: true, Kind: "ClusterRole", Group: "rbac.authorization.k8s.io", Version: "v1"},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			GroupVersion: "rbac.authorization.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "clusterrolebindings", Namespaced: true, Kind: "ClusterRoleBinding", Group: "rbac.authorization.k8s.io", Version: "v1"},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DestinationRule",
				APIVersion: "networking.istio.io/v1alpha3",
			},
			GroupVersion: "networking.istio.io/v1alpha3",
			APIResources: []metav1.APIResource{
				{Name: "destinationrules", Namespaced: true, Kind: "DestinationRule", Group: "networking.istio.io", Version: "v1alpha3"},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "NatsCluster",
				APIVersion: "nats.io/v1alpha2",
			},
			GroupVersion: "nats.io/v1alpha2",
			APIResources: []metav1.APIResource{
				{Name: "natsclusters", Namespaced: true, Kind: "NatsCluster", Group: "nats.io", Version: "v1alpha2"},
			},
		},
	}
}

type FakeCached struct {
	discovery.DiscoveryInterface
}

func (*FakeCached) Fresh() bool {
	return true
}

func (*FakeCached) Invalidate() {
	return
}

func getScheme(t *testing.T) (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	err = appsv1.AddToScheme(scheme)
	require.NoError(t, err)

	err = apiextensionsapis.AddToScheme(scheme)
	require.NoError(t, err)

	err = rbacv1.AddToScheme(scheme)
	require.NoError(t, err)

	//err = netoworkingistio.addToScheme(scheme)
	//require.NoError(t, err)
	//
	//err = natsv1alpha2.AddToScheme(scheme)
	//require.NoError(t, err)

	return scheme, nil
}

func getNATSDeployment(ctx context.Context, clientset dynamic.Interface, namespace string) (*appsv1.Deployment, error) {
	deploy := new(appsv1.Deployment)
	deploymentUnstruct, err := clientset.Resource(deploymentGVR()).Namespace(namespace).Get(ctx, natsOperatorDeploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(deploymentUnstruct.Object, &deploy)
	if err != nil {
		return nil, err
	}
	return deploy, nil
}

func getCRD(ctx context.Context, clientset dynamic.Interface, name string) (*apiextensionsapis.CustomResourceDefinition, error) {
	customDefsUnstruct, err := clientset.Resource(customResourceDefsGVR()).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	customDef := new(apiextensionsapis.CustomResourceDefinition)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(customDefsUnstruct.Object, &customDef)
	if err != nil {
		return nil, err
	}
	return customDef, nil
}

func deploymentGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Version:  appsv1.SchemeGroupVersion.Version,
		Group:    appsv1.SchemeGroupVersion.Group,
		Resource: "deployments",
	}
}

func customResourceDefsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Version:  apiextensionsapis.SchemeGroupVersion.Version,
		Group:    apiextensionsapis.SchemeGroupVersion.Group,
		Resource: "customresourcedefinitions",
	}
}
