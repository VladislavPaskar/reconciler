package preaction

import (
	"context"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/chart"
	pmock "github.com/kyma-incubator/reconciler/pkg/reconciler/chart/mocks"
	kubenetes "github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	"github.com/kyma-incubator/reconciler/pkg/reconciler/kubernetes/mocks"
	"github.com/stretchr/testify/require"

	"github.com/kyma-incubator/reconciler/pkg/logger"
	"github.com/kyma-incubator/reconciler/pkg/reconciler"
	"github.com/kyma-incubator/reconciler/pkg/reconciler/service"
)

func TestDeletingNatsOperatorResources(t *testing.T) {

	testCases := []struct {
		name                         string
		natsOperatorDeploymentExists bool
	}{
		// no deployments found
		{
			name:                         "Should do nothing if there is no nats operator deployment",
			natsOperatorDeploymentExists: false,
		},
		// nats operator deployment exist
		{
			name:                         "Should delete the nats deployment and the leftover CRDs",
			natsOperatorDeploymentExists: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			action := removeNatsOperatorStep{}

			k8sClient := mocks.Client{}
			clientSet := fake.NewSimpleClientset()
			k8sClient.On("Clientset").Return(clientSet, nil)

			mockProvider := pmock.Provider{}
			if tc.natsOperatorDeploymentExists {
				mockProvider.On("RenderManifest", mock.Anything).Return(&chart.Manifest{}, nil)
				k8sClient.On("Delete", context.TODO(), mock.Anything, namespace).Return([]*kubenetes.Resource{}, nil)
			}

			actionContext := service.ActionContext{
				KubeClient:    &k8sClient,
				Context:       context.TODO(),
				Logger:        logger.NewLogger(false),
				Task:          &reconciler.Task{},
				ChartProvider: &mockProvider,
			}

			// create the deployment if required
			if tc.natsOperatorDeploymentExists {
				_, err = createDeployment(actionContext.Context, clientSet, newDeployment(natsOperatorDeploymentName))
				require.NoError(t, err)
			}

			action.Execute(&actionContext, actionContext.Logger)

			k8sClient.AssertCalled(t, "Clientset")

			// should call the deletion onl if the nats operator deployment exists
			if tc.natsOperatorDeploymentExists {
				mockProvider.AssertCalled(t, "RenderManifest", mock.Anything)
				k8sClient.AssertCalled(t, "Delete", context.TODO(), mock.Anything, namespace)
			} else {
				k8sClient.AssertNotCalled(t, "RenderManifest", mock.Anything)
				k8sClient.AssertNotCalled(t, "Delete", context.TODO(), mock.Anything, namespace)
			}
		})
	}
}
