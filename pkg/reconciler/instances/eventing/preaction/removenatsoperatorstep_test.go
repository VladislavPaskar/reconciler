package preaction

import (
	"context"
	pmock "github.com/kyma-incubator/reconciler/pkg/reconciler/chart/mocks"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/kubeclient"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	apiextensionsapis "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	kubeclientmocks "github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/kubeclient/mocks"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/mocks"
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
		// no deployments found
		{
			name:                           "Should do nothing if there is no nats operator deployment",
			natsOperatorDeploymentExists:   false,
			expectedNatsOperatorDeployment: nil,
			expectedNatsClusterCRD:         nil,
			expectedNatsServiceRoleCRD:     nil,
		},
		// nats operator deployment exist
		{
			name:                           "Should delete the nats deployment and the leftover CRDs",
			natsOperatorDeploymentExists:   true,
			expectedNatsOperatorDeployment: nil,
			expectedNatsClusterCRD:         nil,
			expectedNatsServiceRoleCRD:     nil,
		},
	}

	setup := func(deployNatsOperator bool) (kubernetes.Interface, removeNatsOperatorStep, *service.ActionContext, *apiextensionsfake.Clientset, *pmock.Provider, *mocks.Client, *kubeclientmocks.Client) {
		mainContext := context.TODO()
		kubeClientMock := kubeclientmocks.Client{}
		action := removeNatsOperatorStep{
			kubeClientProvider: func(context *service.ActionContext, logger *zap.SugaredLogger) (kubeclient.Client, error) {
				return &kubeClientMock, nil
			},
		}

		k8sClient := mocks.Client{}
		clientSet := fake.NewSimpleClientset()

		mockProvider := pmock.Provider{}
		mockManifest := chart.Manifest{
			Manifest: "testManifest",
		}

		var apiExtensionsFakeClient *apiextensionsfake.Clientset
		if deployNatsOperator {
			apiExtensionsFakeClient = apiextensionsfake.NewSimpleClientset(natsClusterCRD, natsServiceRoleCRD)
			natsOperatorDeployment, err := clientSet.AppsV1().Deployments(namespace).Create(mainContext, natsDeployment, metav1.CreateOptions{})
			// check the required resources were deployed
			require.NoError(t, err)
			require.NotNil(t, natsOperatorDeployment)

			c1, err := getCRD(apiExtensionsFakeClient, natsClusterCRD.Name)
			require.NoError(t, err)
			require.NotNil(t, c1)

			c2, err := getCRD(apiExtensionsFakeClient, natsServiceRoleCRD.Name)
			require.NoError(t, err)
			require.NotNil(t, c2)
		} else {
			apiExtensionsFakeClient = apiextensionsfake.NewSimpleClientset()
		}

		k8sClient.On("Clientset").Return(clientSet, nil)
		mockProvider.On("RenderManifest", mock.Anything).Return(&mockManifest, nil)
		k8sClient.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		kubeClientMock.On(
			"DeleteResourceByKindAndNameAndNamespace",
			"customresourcedefinitions",
			natsClusterCRD.Name,
			"kyma-system",
			metav1.DeleteOptions{},
		).Return(nil, nil)
		kubeClientMock.On(
			"DeleteResourceByKindAndNameAndNamespace",
			"customresourcedefinitions",
			natsServiceRoleCRD.Name,
			"kyma-system",
			metav1.DeleteOptions{},
		).Return(nil, nil)

		actionContext := &service.ActionContext{
			KubeClient:    &k8sClient,
			Context:       mainContext,
			Logger:        logger.NewLogger(false),
			Task:          &reconciler.Task{},
			ChartProvider: &mockProvider,
		}
		return clientSet, action, actionContext, apiExtensionsFakeClient, &mockProvider, &k8sClient, &kubeClientMock
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			clientSet, action, actionContext, apiExtensionsFakeClient, mockProvider, k8sClient, kubeclientmocksImpl := setup(tc.natsOperatorDeploymentExists)

			err = action.Execute(actionContext, actionContext.Logger)
			require.NoError(t, err)

			// the delete functions should only be called if the nats-operator deployment exists
			if tc.natsOperatorDeploymentExists {
				mockProvider.AssertCalled(t, "RenderManifest", mock.Anything)
				k8sClient.AssertCalled(t, "Delete", actionContext.Context, mock.Anything, namespace)
				// mock not only the deletion calls, but the deletions themselves too
				err := deleteDeployment(actionContext.Context, clientSet, natsOperatorDeploymentName)
				require.NoError(t, err)

				kubeclientmocksImpl.AssertCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsClusterCRD.Name, "kyma-system", metav1.DeleteOptions{})
				err = deleteCRD(apiExtensionsFakeClient, natsClusterCRD.Name)
				require.NoError(t, err)

				kubeclientmocksImpl.AssertCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsServiceRoleCRD.Name, "kyma-system", metav1.DeleteOptions{})
				err = deleteCRD(apiExtensionsFakeClient, natsServiceRoleCRD.Name)
				require.NoError(t, err)
			} else {
				k8sClient.AssertNotCalled(t, "RenderManifest", mock.Anything)
				k8sClient.AssertNotCalled(t, "Delete", context.TODO(), mock.Anything, namespace)
				kubeclientmocksImpl.AssertNotCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsOperatorCRDsToDelete[0], "kyma-system", metav1.DeleteOptions{})
				kubeclientmocksImpl.AssertNotCalled(t, "DeleteResourceByKindAndNameAndNamespace", "customresourcedefinitions", natsOperatorCRDsToDelete[1], "kyma-system", metav1.DeleteOptions{})
			}
			// check that the action's step deleted all the nats-operator resources including CRDs
			gotPublisherDeployment, err := getDeployment(actionContext, clientSet, natsOperatorDeploymentName)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNatsOperatorDeployment, gotPublisherDeployment)

			gotNatsClusterCRD, err := getCRD(apiExtensionsFakeClient, natsOperatorCRDsToDelete[0])
			require.True(t, k8sErrors.IsNotFound(err))
			require.Equal(t, tc.expectedNatsClusterCRD, gotNatsClusterCRD)

			gotNatsServiceRoleCRD, err := getCRD(apiExtensionsFakeClient, natsOperatorCRDsToDelete[1])
			require.True(t, k8sErrors.IsNotFound(err))
			require.Equal(t, tc.expectedNatsServiceRoleCRD, gotNatsServiceRoleCRD)
		})
	}
}

func getCRD(client *apiextensionsfake.Clientset, name string) (*apiextensionsapis.CustomResourceDefinition, error) {
	return client.ApiextensionsV1beta1().CustomResourceDefinitions().Get(context.TODO(), name, metav1.GetOptions{})
}

func deleteCRD(client *apiextensionsfake.Clientset, name string) error {
	return client.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(context.TODO(), name, metav1.DeleteOptions{})
}
func deleteDeployment(context context.Context, client kubernetes.Interface, name string) error {
	if err := client.AppsV1().Deployments(namespace).Delete(context, name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}
