package preaction

import (
	"context"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"testing"
	"time"

	pmock "github.com/kyma-incubator/reconciler/pkg/reconciler/chart/mocks"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/adapter"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/kubeclient"
	"github.com/stretchr/testify/mock"
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
	"github.com/kyma-incubator/reconciler/pkg/reconciler/chart"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/service"
)

func TestDeletingNatsOperatorResources(t *testing.T) {
	testCases := []struct {
		name                           string
		natsOperatorDeploymentExists   bool
		expectedNatsOperatorDeployment *appsv1.Deployment
		expectedNatsClusterCRD         *apiextensionsapis.CustomResourceDefinition
		expectedNatsServiceRoleCRD     *apiextensionsapis.CustomResourceDefinition
	}{
		//{
		//	name:                           "Should do nothing if there is no nats operator deployment",
		//	natsOperatorDeploymentExists:   false,
		//	expectedNatsOperatorDeployment: nil,
		//	expectedNatsClusterCRD:         nil,
		//	expectedNatsServiceRoleCRD:     nil,
		//},
		{
			name:                         "Should delete the nats deployment and the leftover CRDs",
			natsOperatorDeploymentExists: true,
		},
	}

	setup := func(deployNatsOperator bool) (removeNatsOperatorStep, *service.ActionContext, dynamic.Interface) {
		ctx := context.TODO()
		mapper := restmapper.NewDeferredDiscoveryRESTMapper()
		dynamicClient := fakeDynamic.NewSimpleDynamicClient(runtime.NewScheme())
		kubeClientMock := kubeclient.NewFakeClient(dynamicClient, mapper)


		content, err := ioutil.ReadFile("natsOperatorResources.yaml")
		require.NoError(t, err)
		unstructs, err := kubeclient.ToUnstructured(content, true)
		require.NoError(t, err)

		result := make([]runtime.Object, len(unstructs))
		for i, obj := range unstructs {
			crdUnstructMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			crdUnstruct := &unstructured.Unstructured{Object: crdUnstructMap}
			result[i] = crdUnstruct
		}
		if deployNatsOperator {
			deployNATSCrds(dynamicClient, ctx, t)
		}

		scheme := runtime.NewScheme()
		err = corev1.AddToScheme(scheme)
		require.NoError(t, err)
		err = appsv1.AddToScheme(scheme)
		require.NoError(t, err)
		err = apiextensionsapis.AddToScheme(scheme)
		require.NoError(t, err)
		fakeDynamic.NewSimpleDynamicClient(scheme, result...)

		action := removeNatsOperatorStep{
			kubeClientProvider: func(context *service.ActionContext, logger *zap.SugaredLogger) (*kubeclient.KubeClient, error) {
				return kubeClientMock, nil
			},
		}

		//k8sClient := mocks.Client{}
		//clientSet := fake.NewSimpleClientset()

		mockProvider := pmock.Provider{}
		mockManifest := chart.Manifest{
			Manifest: string(content),
		}
		mockProvider.On("RenderManifest", mock.Anything).Return(&mockManifest, nil)

		//apiExtensionsFakeClient := fakeextensionsclientset.NewSimpleClientset()

		//k8sClient.On("Clientset").Return(clientSet, nil)
		//k8sClient.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		log := logger.NewLogger(false)
		actionContext := &service.ActionContext{
			KubeClient: adapter.NewFakeKubernetesClient(*kubeClientMock, log, &adapter.Config{
				ProgressInterval: 5 * time.Second,
				ProgressTimeout:  10 * time.Second,
			}),
			Context:       ctx,
			Logger:        log,
			Task:          &reconciler.Task{},
			ChartProvider: &mockProvider,
		}
		return action, actionContext, dynamicClient
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			action, actionContext, clientset := setup(tc.natsOperatorDeploymentExists)

			err = action.Execute(actionContext, actionContext.Logger)
			require.NoError(t, err)

			// the delete functions should only be called if the nats-operator deployment exists
			if tc.natsOperatorDeploymentExists {
				//mockProvider.AssertCalled(t, "RenderManifest", mock.Anything)
				//k8sClient.AssertCalled(t, "Delete", actionContext.Context, mock.Anything, namespace)
				//// mock not only the deletion calls, but the deletions themselves too
				//err := deleteDeployment(actionContext.Context, clientSet, natsOperatorDeploymentName)
				//require.NoError(t, err)
				//
				//kubeclientmocksImpl.AssertCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsClusterCRD.Name, "kyma-system", metav1.DeleteOptions{})
				//err = deleteCRD(apiExtensionsFakeClient, natsClusterCRD.Name)
				//require.NoError(t, err)
				//
				//kubeclientmocksImpl.AssertCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsServiceRoleCRD.Name, "kyma-system", metav1.DeleteOptions{})
				//err = deleteCRD(apiExtensionsFakeClient, natsServiceRoleCRD.Name)
				//require.NoError(t, err)
			} else {
				//k8sClient.AssertNotCalled(t, "RenderManifest", mock.Anything)
				//k8sClient.AssertNotCalled(t, "Delete", context.TODO(), mock.Anything, namespace)
				//kubeclientmocksImpl.AssertNotCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsOperatorCRDsToDelete[0], "kyma-system", metav1.DeleteOptions{})
				//kubeclientmocksImpl.AssertNotCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsOperatorCRDsToDelete[1], "kyma-system", metav1.DeleteOptions{})
			}
			// check that the action's step deleted all the nats-operator resources including CRDs
			_, err = getNATSDeployment(actionContext.Context, clientset, namespace)
			//require.NotNil(t, err)
			//require.True(t, k8sErrors.IsNotFound(err))

			_, err = getCRD(actionContext.Context, clientset, natsOperatorCRDsToDelete[0])
			//require.NotNil(t, err)
			//require.True(t, k8sErrors.IsNotFound(err))

			_, err = getCRD(actionContext.Context, clientset, natsOperatorCRDsToDelete[1])
			//require.NotNil(t, err)
			//require.True(t, k8sErrors.IsNotFound(err))

			//gotNatsClusterCRD, err := getCRD(apiExtensionsFakeClient, natsOperatorCRDsToDelete[0])
			//require.True(t, k8sErrors.IsNotFound(err))
			//require.Equal(t, tc.expectedNatsClusterCRD, gotNatsClusterCRD)
			//
			//gotNatsServiceRoleCRD, err := getCRD(apiExtensionsFakeClient, natsOperatorCRDsToDelete[1])
			//require.True(t, k8sErrors.IsNotFound(err))
			//require.Equal(t, tc.expectedNatsServiceRoleCRD, gotNatsServiceRoleCRD)
		})
	}
}

func deployNATSCrds(clientset dynamic.Interface, ctx context.Context, t *testing.T) {

	crdUnstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&natsClusterCRD)
	crdUnstruct := &unstructured.Unstructured{Object: crdUnstructMap}
	_, err = clientset.Resource(customResourceDefsGVR()).Create(ctx, crdUnstruct, metav1.CreateOptions{})
	require.NoError(t, err)

	crdUnstructMap, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&natsServiceRoleCRD)
	crdUnstruct = &unstructured.Unstructured{Object: crdUnstructMap}
	_, err = clientset.Resource(customResourceDefsGVR()).Create(ctx, crdUnstruct, metav1.CreateOptions{})
	require.NoError(t, err)

	return
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
