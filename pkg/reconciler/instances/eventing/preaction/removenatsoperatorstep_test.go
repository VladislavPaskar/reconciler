package preaction

import (
	"context"
	pmock "github.com/kyma-incubator/reconciler/pkg/reconciler/chart/mocks"
	kubenetes "github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/kubeclient"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

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
		expectedNatsClusterCRD         *unstructured.Unstructured
		expectedNatsServiceRoleCRD     *unstructured.Unstructured
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

	setup := func() (kubernetes.Interface, removeNatsOperatorStep, *service.ActionContext) {
		action := removeNatsOperatorStep{}

		k8sClient := mocks.Client{}
		clientSet := fake.NewSimpleClientset()

		k8sClient.On("Clientset").Return(clientSet, nil)
		k8sClient.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return([]*kubenetes.Resource{}, nil)
		k8sClient.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return([]*kubenetes.Resource{}, nil)
		k8sClient.On("ListResource", mock.Anything, mock.Anything).Return(&unstructured.UnstructuredList{}, nil)
		k8sClient.On("GetDeployment", mock.Anything, mock.Anything, mock.Anything).Return(appsv1.Deployment{}, nil)

		mockProvider := pmock.Provider{}
		// todo remove duplicate
		content, _ := ioutil.ReadFile("natsOperatorResources.yaml")
		mockManifest := chart.Manifest{
			Manifest: string(content),
		}
		mockProvider.On("RenderManifest", mock.Anything).Return(&mockManifest, nil)

		actionContext := &service.ActionContext{
			KubeClient:    &k8sClient,
			Context:       context.TODO(),
			Logger:        logger.NewLogger(false),
			Task:          &reconciler.Task{},
			ChartProvider: &mockProvider,
		}
		return clientSet, action, actionContext
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			clientSet, action, actionContext := setup()

			if tc.natsOperatorDeploymentExists {
				err = createNatsResources(actionContext, t)
				require.NoError(t, err)

				gotNatsClusterCRD, err := getCRD(actionContext.KubeClient, natsOperatorCRDsToDelete[0])
				require.NoError(t, err)
				require.NotNil(t, gotNatsClusterCRD)

				gotNatsServiceRoleCRD, err := getCRD(actionContext.KubeClient, natsOperatorCRDsToDelete[1])
				require.NoError(t, err)
				require.NotNil(t, gotNatsServiceRoleCRD)
			}

			err = action.Execute(actionContext, actionContext.Logger)
			require.NoError(t, err)

			// check that the preaction deleted all the nats-operator resources including CRDs
			gotPublisherDeployment, err := getDeployment(actionContext, clientSet, natsOperatorDeploymentName)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNatsOperatorDeployment, gotPublisherDeployment)

			gotNatsClusterCRD, err := getCRD(actionContext.KubeClient, natsOperatorCRDsToDelete[0])
			require.NoError(t, err)
			require.Equal(t, tc.expectedNatsClusterCRD, gotNatsClusterCRD)

			gotNatsServiceRoleCRD, err := getCRD(actionContext.KubeClient, natsOperatorCRDsToDelete[1])
			require.NoError(t, err)
			require.Equal(t, tc.expectedNatsServiceRoleCRD, gotNatsServiceRoleCRD)
		})
	}
}

func createNatsResources(actionContext *service.ActionContext, t *testing.T) error {
	comp := chart.NewComponentBuilder(natsOperatorLastVersion, natsSubChartPath).
		WithNamespace(namespace).
		Build()

	manifest, err := actionContext.ChartProvider.RenderManifest(comp)
	if err != nil {
		return err
	}

	actionContext.KubeClient.Deploy(actionContext.Context, manifest.Manifest, namespace)
	list, err := actionContext.KubeClient.ListResource(CRDKind, metav1.ListOptions{})
	require.NotEqual(t, len(list.Items), 0)
	//logger.NewLogger(false).Errorf("%+v\n", deploy)
	//logger.NewLogger(false).Error("after:" + fmt.Sprint(len(list.Items)))
	return nil
}

func getCRD(client kubenetes.Client, name string) (*unstructured.Unstructured, error) {
	list, err := client.ListResource(CRDKind, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, crd := range list.Items {
		if crd.GetName() == name {
			return &crd, nil
		}
	}
	return nil, nil
}

// apply the rendered helm manifest from eventing/nats-operator
func applyNatsOperatorManifest(ctx *service.ActionContext, t *testing.T) error {
	kubeClient, err := kubeclient.NewKubeClient(ctx.Task.Kubeconfig, nil)

	content, err := ioutil.ReadFile("natsOperatorResources.yaml")
	if err != nil {
		return err
	}
	resources, err := kubeclient.ToUnstructured(content, true)

	if err != nil {
		return err
	}

	for _, resource := range resources {
		deployedResource, err := kubeClient.ApplyWithNamespaceOverride(resource, namespace)
		if err != nil {
			return err
		}
		t.Logf("Deployed test resource '%s", deployedResource)
	}
	return nil
}
